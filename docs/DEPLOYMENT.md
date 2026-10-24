# Nexogent — Deployment Guide

How Nexogent is deployed into a client's AWS account, onto their Amazon EKS cluster.

> Read-only product. It reads the account and reports; it changes nothing.
> No static AWS keys run inside the cluster — the agent uses IRSA.

---

## 0. Prerequisites
See **PREREQUISITES.md** for the exact AWS access and resources the client provides.
In short: an EKS cluster, permission to create an ECR repo + a read-only IAM role +
an IRSA service account, and Amazon Bedrock model access in the chosen region.

Deploy-time credentials are configured **locally** (`aws configure`) or assumed via a
role. Never paste AWS keys into a chat or commit them.

---

## 1. Architecture (in-cluster)

```
                 Internet
                    │
             ┌──────▼────────┐
             │   Ingress /   │   (ALB or LoadBalancer)
             │  CloudFront   │
             └──────┬────────┘
                    │
        ┌───────────▼───────────────── EKS (client's cluster) ─────────────┐
        │  Namespace: nexogent                                             │
        │   ┌──────────────┐        ┌───────────────────────────┐          │
        │   │  dashboard   │◄──────►│   nexogent agent (Go)     │          │
        │   │ (React/Nginx)│  HTTP  │   Deployment + Service     │          │
        │   └──────────────┘        └──────────┬────────────────┘          │
        │                                      │ IRSA (read-only role)      │
        └──────────────────────────────────────┼───────────────────────────┘
                                               │
              ┌────────────────────────────────┼─────────────────────────┐
              ▼            ▼          ▼         ▼           ▼              ▼
            EC2          S3         IAM       RDS        Lambda       Bedrock
          (Describe/Get/List — READ ONLY)                            (InvokeModel)
```

---

## 2. Build and push the agent image (ECR)

```bash
ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
REGION=us-east-1
aws ecr create-repository --repository-name nexogent || true
aws ecr get-login-password --region $REGION | docker login --username AWS \
  --password-stdin $ACCOUNT_ID.dkr.ecr.$REGION.amazonaws.com

# Build the pure-Go agent image (tiny, static)
docker build -t $ACCOUNT_ID.dkr.ecr.$REGION.amazonaws.com/nexogent:latest .
docker push $ACCOUNT_ID.dkr.ecr.$REGION.amazonaws.com/nexogent:latest
```

---

## 3. Read-only IAM role + IRSA

The agent pod assumes a role whose policy allows only `Describe*/Get*/List*` on the
read services plus `bedrock:InvokeModel`. No write/delete actions.

```bash
CLUSTER=eks-lab
eksctl utils associate-iam-oidc-provider --cluster $CLUSTER --approve
eksctl create iamserviceaccount \
  --cluster $CLUSTER --namespace nexogent --name nexogent-sa \
  --attach-policy-arn arn:aws:iam::$ACCOUNT_ID:policy/NexogentReadOnly \
  --approve
```

Minimum policy (read-only):
```json
{
  "Version": "2012-10-17",
  "Statement": [
    { "Effect": "Allow", "Action": [
        "ec2:Describe*", "s3:GetBucket*", "s3:ListAllMyBuckets",
        "s3:GetEncryptionConfiguration",
        "iam:Get*", "iam:List*", "iam:GenerateCredentialReport", "iam:GetCredentialReport",
        "lambda:List*", "lambda:Get*",
        "rds:Describe*", "dynamodb:Describe*", "dynamodb:List*",
        "ecs:List*", "ecs:Describe*", "eks:Describe*", "eks:List*",
        "access-analyzer:List*", "access-analyzer:Get*",
        "sts:GetCallerIdentity"
      ], "Resource": "*" },
    { "Effect": "Allow", "Action": ["bedrock:InvokeModel"],
      "Resource": "arn:aws:bedrock:*::foundation-model/anthropic.*" }
  ]
}
```

---

## 4. Deploy with Helm

```bash
helm upgrade --install nexogent deploy/helm/nexogent \
  --namespace nexogent --create-namespace \
  --set image.repository=$ACCOUNT_ID.dkr.ecr.$REGION.amazonaws.com/nexogent \
  --set image.tag=latest \
  --set serviceAccount.roleArn=arn:aws:iam::$ACCOUNT_ID:role/NexogentReadOnly \
  --set region=$REGION
```

Verify:
```bash
kubectl -n nexogent get pods,svc,ingress
kubectl -n nexogent port-forward svc/nexogent 8080:8080   # or open the ingress URL
curl -s localhost:8080/health
curl -s -X POST localhost:8080/api/v1/review | head
```

---

## 5. Scale the node group (if needed)

```bash
aws eks update-nodegroup-config --cluster-name $CLUSTER --nodegroup-name core \
  --scaling-config minSize=1,maxSize=3,desiredSize=2
```
> Cost note: EKS control plane and worker nodes bill per hour even when idle.
> Keep the desired count minimal; scale down when not in use (see `stop.sh`).

---

## 6. Spin-up / teardown scripts
- `start.sh` — builds+pushes the image, ensures the IAM role/IRSA, scales the node
  group up, and `helm install`s Nexogent. Brings the dashboard live.
- `stop.sh` — `helm uninstall`s Nexogent and scales the node group down to stop
  billing. (Does not delete the cluster or other workloads.)

---

## 7. Handover
After deployment, the client holds full admin of their account. Nexogent runs under
the read-only IRSA role. We retain no standing access. To remove Nexogent entirely,
run `stop.sh` and delete the ECR repo, IAM policy/role, and the `nexogent` namespace.
