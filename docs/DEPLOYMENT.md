# Nexogent — Deployment Guide

How Nexogent is deployed into a client's AWS account, onto their Amazon EKS cluster.

> Read-only product. It reads the account and reports; it changes nothing.
> No static AWS keys run inside the cluster — the agent uses IRSA.

---

## 0. Prerequisites
See **PREREQUISITES.md** for the exact AWS access and resources the client provides.
In short: an EKS cluster, permission to create an ECR repo + a read-only IAM role +
an IRSA service account + a DynamoDB table, two cluster add-ons (EBS CSI driver and
the AWS Load Balancer Controller), and Amazon Bedrock model access in the chosen region.

Deploy-time credentials are configured **locally** (`aws configure`) or assumed via a
role. Never paste AWS keys into a chat or commit them.

---

## 1. Architecture (in-cluster, microservices)

```
                         Internet
                            │
                   ┌────────▼─────────┐
                   │   ALB Ingress    │   (AWS Load Balancer Controller)
                   │  / → frontend    │
                   │  /api → backend  │
                   └────────┬─────────┘
                            │
   ┌────────────────────────┼──────────── EKS (client's cluster) ───────────────┐
   │  Namespace: nexogent   │                                                    │
   │   ┌──────────────┐     │     ┌───────────────────────────┐                 │
   │   │  frontend    │◄────┴────►│   backend (Go API/agent)   │                 │
   │   │ (React/Nginx)│   HTTP    │   Deployment + Service      │                 │
   │   │  superadmin  │           │   auth gate · ranking · API │                 │
   │   └──────────────┘           └──────┬───────────┬─────────┘                 │
   │                                     │           │ reads results (shared PVC)│
   │   ┌───────────────── scanners ──────┼───┐       │                           │
   │   │ prowler CronJob  (+ scoutsuite, │   │  ┌────▼─────┐                      │
   │   │ cloudsplaining — optional)      │   │  │ results  │ (EBS PVC)            │
   │   └─────────────────────────────────┼───┘  └──────────┘                     │
   │                                     │ IRSA (read-only role)                 │
   └─────────────────────────────────────┼────────────────────┬──────────────────┘
                                          │                    │
            ┌─────────────────────────────┼──────────┐   ┌─────▼──────────┐
            ▼        ▼        ▼        ▼   ▼          ▼   │  DynamoDB       │
          EC2       S3       IAM      RDS Lambda   Bedrock│  (review history)│
        (Describe/Get/List — READ ONLY, ALL REGIONS)     └─────────────────┘
```

Workloads a `helm install` creates: **nexogent-backend** (Deployment+Service),
**nexogent-frontend** (Deployment+Service), **nexogent-prowler** (CronJob, plus an
optional initial Job), a shared **PVC** for scanner results, an **Ingress** (ALB),
and an auth **Secret**. ScoutSuite and cloudsplaining CronJobs are included but
disabled by default.

---

## 2. Cluster prerequisites (one-time per cluster)
Two add-ons are required and installed once per cluster (full commands in
`deploy/CLUSTER_PREREQS.md`):
- **Amazon EBS CSI driver** — provisions the shared PVC that scanners write to.
- **AWS Load Balancer Controller** — turns the `alb` Ingress into a public ALB.

Verify:
```bash
kubectl get csidriver ebs.csi.aws.com
kubectl get ingressclass alb
```
If you skip either, deploy with `--set prowler.enabled=false` and/or
`--set ingress.enabled=false`.

---

## 3. Build and push the images (ECR)
Two private ECR repos, one per image.

```bash
ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
REGION=us-east-1
ECR=$ACCOUNT_ID.dkr.ecr.$REGION.amazonaws.com
aws ecr get-login-password --region $REGION | docker login --username AWS --password-stdin $ECR

for repo in nexogent nexogent-frontend; do
  aws ecr describe-repositories --repository-names $repo --region $REGION >/dev/null 2>&1 \
    || aws ecr create-repository --repository-name $repo --region $REGION
done

# Backend (pure Go) and frontend (React/Nginx)
docker build --platform linux/amd64 -t $ECR/nexogent:latest .
docker push $ECR/nexogent:latest
docker build --platform linux/amd64 -t $ECR/nexogent-frontend:latest ./dashboard
docker push $ECR/nexogent-frontend:latest
```

---

## 4. Read-only IAM role + IRSA
The agent and scanners assume one role whose policy allows only `Describe*/Get*/List*`
on the read services plus `bedrock:InvokeModel`, and read/write on the single
DynamoDB review table. No write/delete actions on account resources.

```bash
CLUSTER=eks-lab
aws iam create-policy --policy-name NexogentReadOnly \
  --policy-document file://deploy/iam/nexogent-readonly-policy.json || true

eksctl utils associate-iam-oidc-provider --cluster $CLUSTER --approve
eksctl create iamserviceaccount \
  --cluster $CLUSTER --namespace nexogent --name nexogent-sa \
  --role-name NexogentReadOnly \
  --attach-policy-arn arn:aws:iam::$ACCOUNT_ID:policy/NexogentReadOnly \
  --approve --override-existing-serviceaccounts
```

The policy grants (read-only): `ec2:Describe*`, `s3:GetBucket*`/`ListAllMyBuckets`/
encryption, `iam:Get*/List*` + credential report, `lambda:List*/Get*`,
`rds:Describe*`, `dynamodb:Describe*/List*`, `ecs/eks:Describe*/List*`,
`cloudtrail:Describe*/Get*/ListTrails`, `access-analyzer`, `kms`, `cloudwatch`,
`logs`, `sns` read, `sts:GetCallerIdentity`; plus `bedrock:InvokeModel` on Anthropic
models; plus read/write limited to the `nexogent-reviews` DynamoDB table only.

---

## 5. DynamoDB review table (persistence)
Review history is stored in one on-demand DynamoDB table so reports survive pod
restarts and multiple replicas.

```bash
aws dynamodb create-table --table-name nexogent-reviews --region $REGION \
  --billing-mode PAY_PER_REQUEST \
  --attribute-definitions AttributeName=pk,AttributeType=S AttributeName=sk,AttributeType=S \
  --key-schema AttributeName=pk,KeyType=HASH AttributeName=sk,KeyType=RANGE
```
(If you skip this, the backend falls back to in-memory storage — fine for a demo,
but history is lost on restart.)

---

## 6. Deploy with Helm
Set the superadmin password so the portal isn't public. Pass it at deploy time;
never commit it.

```bash
ROLE_ARN=$(aws iam get-role --role-name NexogentReadOnly --query 'Role.Arn' --output text)

helm upgrade --install nexogent deploy/helm/nexogent \
  --namespace nexogent --create-namespace \
  --set backend.image.repository=$ECR/nexogent  --set backend.image.tag=latest \
  --set frontend.image.repository=$ECR/nexogent-frontend --set frontend.image.tag=latest \
  --set serviceAccount.roleArn=$ROLE_ARN \
  --set region=$REGION \
  --set persistence.reviewsTable=nexogent-reviews \
  --set-string auth.password="$SUPERADMIN_PASSWORD"
```

Useful values:
- `scanInterval` — in-process scheduled review interval (default `6h`; empty disables).
- `prowler.schedule` — Prowler CronJob schedule (default every 6h).
- `scoutsuite.enabled` / `cloudsplaining.enabled` — enable the extra scanners.
- `ingress.enabled` / `prowler.enabled` — toggle the ALB / scanner if a prereq is absent.

Verify:
```bash
kubectl -n nexogent get deploy,svc,cronjob,ingress,pods
URL=$(kubectl -n nexogent get ingress nexogent -o jsonpath='{.status.loadBalancer.ingress[0].hostname}')
curl -s http://$URL/health
# Sign in as superadmin, then the portal drives the API:
#   POST /api/v1/auth/login  → bearer token
#   GET  /api/v1/services    → landing
#   GET  /api/v1/resources?service=ec2&region=us-east-1
#   POST /api/v1/audit?service=ec2        (on-demand)
```

---

## 7. Scale the node group (if needed)
```bash
aws eks update-nodegroup-config --cluster-name $CLUSTER --nodegroup-name core \
  --scaling-config minSize=1,maxSize=3,desiredSize=2
```
> Cost note: EKS control plane and worker nodes bill per hour even when idle.
> Keep the desired count minimal; scale down when not in use (see `stop.sh`).

---

## 8. Spin-up / teardown scripts
- `start.sh` — ensures both ECR repos, builds+pushes both images, ensures the IAM
  role/IRSA, the DynamoDB table, checks the cluster add-ons, scales the node group
  up, and `helm install`s Nexogent. Reads `SUPERADMIN_PASSWORD` from the environment.
- `stop.sh` — `helm uninstall`s Nexogent and scales the node group down to stop
  billing. `--purge` also removes the IAM policy/role, both ECR repos, the DynamoDB
  table, and the `nexogent` namespace. (Never deletes the cluster or other workloads.)

---

## 9. Handover
After deployment, the client holds full admin of their account. Nexogent runs under
the read-only IRSA role. We retain no standing access. To remove Nexogent entirely,
run `stop.sh --purge` and delete the ECR repos, IAM policy/role, DynamoDB table, and
the `nexogent` namespace.
