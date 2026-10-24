# Nexogent — Deployment Guide (Free Build, v2)

How the MSP deploys Nexogent into a managed client's AWS account, onto their EKS
cluster. Also covers **local mode** (laptop) and **demo mode** (no AWS).

> Read-only product. It reads the account and reports; it changes nothing.
> No static AWS keys run inside the cluster — access is via IRSA.

---

## 0. Prerequisites
See **PREREQUISITES.md**. In short: an EKS cluster with an OIDC provider, the EBS CSI
driver and the AWS Load Balancer Controller; permission to create two ECR repos, a
read-only IAM role + IRSA service account, a DynamoDB table, and a namespace-scoped
Kubernetes Role/RoleBinding; a superadmin password and a session secret; optionally
an ACM certificate (HTTPS), a Bedrock model, and a Slack webhook.

Deploy-time credentials are configured locally (`aws configure`) or assumed via a
role. Never paste AWS keys into a chat or commit them.

---

## 1. Architecture (in-cluster, microservices)

```
                         Internet
                            │                 (HTTPS when an ACM cert is set;
                   ┌────────▼─────────┐        optional source-IP allow-list)
                   │   ALB Ingress    │
                   │  / → frontend    │
                   │  /api → backend  │
                   └────────┬─────────┘
                            │
   ┌────────────────────────┼──────────── EKS (client's cluster) ───────────────┐
   │  Namespace: nexogent   │                                                    │
   │   ┌──────────────┐     │     ┌───────────────────────────┐                 │
   │   │  frontend    │◄────┴────►│   backend (Go API/agent)   │                 │
   │   │ (React/Nginx)│   HTTP    │  discovery · own checks ·   │                 │
   │   │  risk-first  │           │  lifecycle · dedupe · rank · │                │
   │   │  portal      │           │  attack paths · reports ·    │                │
   │   └──────────────┘           │  alerts · signed sessions    │                │
   │                              └────┬───────────┬──────┬──────┘                │
   │     reads results (shared PVC) ───┘           │      │ creates Jobs (RBAC)   │
   │   ┌───────────── scanners ───────────────┐    │      ▼                       │
   │   │ prowler CronJob + on-demand Jobs      │    │   batch/Jobs (deep scan)     │
   │   │ (scoutsuite, cloudsplaining optional) │    │                              │
   │   │ co-located with backend via podAffinity│   │ IRSA (read-only role)        │
   │   └──────────────────┬────────────────────┘   │                              │
   └──────────────────────┼─────────────────────────┼──────────────┬──────────────┘
                          │ results volume (EBS PVC) │              │
       ┌──────────────────┴───────────┐         ┌────┴───────┐  ┌───▼──────────────┐
       ▼        ▼      ▼     ▼     ▼   ▼         ▼            ▼  │  DynamoDB        │
     EC2       S3     IAM   RDS  Lambda EKS  CloudTrail  Bedrock │  reviews/history │
   (Describe/Get/List — READ ONLY, ALL REGIONS)        (wording) │  risks/plans/... │
                                                                  └──────────────────┘
```

`helm install` creates: **nexogent-backend** (Deployment+Service),
**nexogent-frontend** (Deployment+Service), **nexogent-prowler** (CronJob, plus
backend-created on-demand Jobs), a shared **PVC**, an **Ingress** (ALB), an auth
**Secret** (superadmin password + session secret), and a **Role + RoleBinding** that
lets the backend run deep scans (read the CronJob; create/get/list Jobs — nothing on
AWS, nothing outside the namespace). ScoutSuite and cloudsplaining CronJobs are
included but disabled by default.

---

## 2. Cluster prerequisites (one-time per cluster)
Full commands in `deploy/CLUSTER_PREREQS.md`:
- **Amazon EBS CSI driver** — provisions the shared PVC.
- **AWS Load Balancer Controller** — turns the `alb` Ingress into an ALB.

```bash
kubectl get csidriver ebs.csi.aws.com
kubectl get ingressclass alb
```
If either is absent, deploy with `--set prowler.enabled=false` and/or
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

docker build --platform linux/amd64 -t $ECR/nexogent:latest .
docker push $ECR/nexogent:latest
docker build --platform linux/amd64 -t $ECR/nexogent-frontend:latest ./dashboard
docker push $ECR/nexogent-frontend:latest
```

---

## 4. Read-only IAM role + IRSA
The agent and scanners assume one role allowing only `Describe*/Get*/List*` on the
read services plus (optional) `bedrock:InvokeModel`, plus read/write on the single
DynamoDB table. No write/delete on account resources.

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

The policy covers the reads listed in PREREQUISITES §4 (including the extra IAM/EC2/
S3/CloudTrail calls the lifecycle, facts, and new own checks need). The only write
grant is to the `nexogent-reviews` table.

---

## 5. DynamoDB table (persistence)
One on-demand table holds reviews, history, the accepted-risk register, plans,
reports, the audit log, settings, and the scheduler lease. Large reviews are chunked
to stay under the item-size limit.

```bash
aws dynamodb create-table --table-name nexogent-reviews --region $REGION \
  --billing-mode PAY_PER_REQUEST \
  --attribute-definitions AttributeName=pk,AttributeType=S AttributeName=sk,AttributeType=S \
  --key-schema AttributeName=pk,KeyType=HASH AttributeName=sk,KeyType=RANGE
```
Without a table the backend falls back to in-memory (local and demo modes only).

---

## 6. Deploy with Helm
Set the superadmin password; the chart generates/reuses a session secret. For HTTPS,
pass the ACM certificate ARN (and optionally an IP allow-list).

```bash
ROLE_ARN=$(aws iam get-role --role-name NexogentReadOnly --query 'Role.Arn' --output text)

helm upgrade --install nexogent deploy/helm/nexogent \
  --namespace nexogent --create-namespace \
  --set backend.image.repository=$ECR/nexogent        --set backend.image.tag=latest \
  --set frontend.image.repository=$ECR/nexogent-frontend --set frontend.image.tag=latest \
  --set serviceAccount.roleArn=$ROLE_ARN \
  --set region=$REGION \
  --set persistence.reviewsTable=nexogent-reviews \
  --set-string auth.password="$SUPERADMIN_PASSWORD" \
  --set ingress.host="nexogent.example.com" \
  --set ingress.certificateArn="arn:aws:acm:$REGION:$ACCOUNT_ID:certificate/XXXX" \
  --set ingress.inboundCidrs="203.0.113.0/24"
```

Useful values:
- `ingress.certificateArn` — enables HTTPS + HTTP→HTTPS redirect. Omit to run HTTP
  (a warning is shown; use `ingress.inboundCidrs` to restrict access).
- `ingress.inboundCidrs` — source IP allow-list on the ALB.
- `scanInterval` / `settings.schedule` — scheduled review interval (default `6h`).
- `prowler.schedule` — Prowler CronJob schedule; Prowler image is pinned (no `latest`).
- `scoutsuite.enabled` / `cloudsplaining.enabled` — enable the optional scanners.
- `backend.replicaCount` — safe to raise (a DynamoDB lease makes only one replica run
  scheduled scans; signed sessions work across replicas).

Backend environment set by the chart: `REVIEWS_TABLE`, `SESSION_SECRET`,
`SESSION_TTL`, `SUPERADMIN_PASSWORD`, `TRUST_PROXY=true`, `PUBLIC_URL`,
`KUBE_NAMESPACE`, `PROWLER_CRONJOB_NAME`, `ALLOWED_ORIGIN` (empty = same-origin),
`DEMO_MODE=false`, optional `BEDROCK_MODEL_ID`.

Verify:
```bash
kubectl -n nexogent get deploy,svc,cronjob,ingress,role,rolebinding,pods
URL=$(kubectl -n nexogent get ingress nexogent -o jsonpath='{.status.loadBalancer.ingress[0].hostname}')
curl -s https://$URL/health            # or http:// if no certificate configured
# Sign in (superadmin) → the portal drives the API:
#   POST /api/v1/auth/login            → signed session token
#   GET  /api/v1/overview              → score, trend, what changed, top fixes
#   GET  /api/v1/findings?status=open  → the queue
#   POST /api/v1/scans/deep            → on-demand Prowler job
#   POST /api/v1/reports {scope}       → create a report, then submit/approve/download
```

---

## 7. Scale the node group (if needed)
```bash
aws eks update-nodegroup-config --cluster-name $CLUSTER --nodegroup-name core \
  --scaling-config minSize=1,maxSize=3,desiredSize=2
```
> EKS control plane and nodes bill hourly even when idle. Keep the count minimal;
> scale down with `stop.sh` when not in use.

---

## 8. Local mode (no cluster)
Run the backend + portal on a laptop against the MSP's own read-only AWS profile.
See `docs/LOCAL.md`.

```bash
cp .env.example .env     # set SUPERADMIN_PASSWORD, AWS_PROFILE, AWS_REGION
docker compose up        # backend + frontend; deep scan is unavailable (no cluster)
docker compose --profile scan up   # also run Prowler once into a shared volume
```
The host `~/.aws` is mounted read-only; no keys are copied into any image.

---

## 9. Demo mode (no AWS)
For sales demonstrations and portal testing with clearly fake data.

```bash
DEMO_MODE=true SUPERADMIN_PASSWORD=demo SESSION_SECRET=$(openssl rand -hex 32) \
  go run ./cmd/nexogent
```
No AWS call is made and no credential is needed. Every page and report shows a
"Demo data, not a real account" banner.

---

## 10. Spin-up / teardown scripts
- `start.sh` — ensures both ECR repos, builds+pushes both images, ensures the IAM
  role/IRSA, the DynamoDB table and the Kubernetes Role/RoleBinding, checks the
  cluster add-ons, scales the node group up, and `helm install`s Nexogent. Reads
  `SUPERADMIN_PASSWORD` from the environment.
- `stop.sh` — `helm uninstall`s Nexogent and scales the node group down to stop
  billing. `--purge` also removes the IAM policy/role, both ECR repos, the DynamoDB
  table, the Role/RoleBinding, and the `nexogent` namespace. (Never deletes the
  cluster or other workloads.)

---

## 11. Handover
After deployment the client holds full admin. Nexogent runs under the read-only IRSA
role; the supplier retains no standing access. To remove Nexogent entirely, run
`stop.sh --purge` and confirm the ECR repos, IAM policy/role, DynamoDB table,
Kubernetes Role/RoleBinding, and the `nexogent` namespace are gone.

---

## 12. Troubleshooting
See `docs/RUNBOOK.md` (scan stuck, deep scan failed, Prowler file missing, DynamoDB
throttling, rotating the password/session secret, scaling down to stop billing) and
`docs/VERIFY.md` (the post-deploy verification checklist).
