# Nexogent — Prerequisites (Free Build, v2)

What the client provides so the MSP can deploy Nexogent into the client's AWS
account. Nexogent is **read-only**: it reads the account and reports; it changes
nothing. The client keeps full admin; the supplier retains no access after handover.

---

## 1. AWS account & region
- One AWS account per deployment (AWS Organizations / multi-account is out of scope).
- A chosen "home" region (e.g. `us-east-1`) for the deployment and, if used, Bedrock.
  Discovery covers **all enabled regions** automatically.

## 2. Amazon EKS cluster
- An existing EKS cluster (or we help create one).
- Permission to create one namespace and the Nexogent workloads in it.
- An **OIDC provider** on the cluster (for IRSA). We can enable it if absent.
- Two cluster add-ons (installed once; we can do this at deploy):
  - **Amazon EBS CSI driver** — for the shared volume scanners write results to.
  - **AWS Load Balancer Controller** — to expose the portal via an ALB.

## 3. Deploy-time access (temporary)
Provide **one** of: a temporary IAM role we can assume, or short-lived deploy
credentials. The deploy identity needs to:
- Create **two ECR repositories** (agent + dashboard) and push images.
- Create a **read-only IAM policy** and an **IRSA role/service account**.
- Create a **DynamoDB table** (history, register, reports, audit log, settings).
- Create a namespace-scoped **Kubernetes Role + RoleBinding** so the backend can run
  on-demand deep scans (read the Prowler CronJob; create/get/list Jobs — nothing on
  AWS, nothing outside the `nexogent` namespace).
- Install the two cluster add-ons (if absent).
- Run **Helm/kubectl** against the cluster; optionally adjust node-group scaling.

> Do not send long-lived access keys over chat or email. Prefer an assumable role or
> short-lived credentials. Rotate anything shared insecurely.

## 4. Read-only runtime permissions (the agent's own role)
At runtime Nexogent uses an **IRSA role** limited to read-only actions, attached to
the agent and scanner pods. It allows:
- Inventory & checks (read-only): `ec2:Describe*` (instances, security groups,
  volumes), `s3:GetBucket*` / `GetBucketPolicyStatus` / `GetBucketAcl` /
  `GetBucketPublicAccessBlock` / `ListAllMyBuckets` / encryption,
  `iam:Get*` / `List*` (incl. `GetRolePolicy`, `ListAttachedRolePolicies`,
  `GetPolicyVersion`, `ListAccessKeys`, `GetLoginProfile`, `GetAccountSummary`,
  credential report), `lambda:List*` / `Get*` (incl. `GetFunctionUrlConfig`),
  `rds:Describe*`, `dynamodb:Describe*` / `List*`, `ecs` / `eks:Describe*` / `List*`,
  `cloudtrail:DescribeTrails` / `GetTrailStatus` / `Get*`, `access-analyzer`, `kms`,
  `cloudwatch`, `logs`, `sns` read, `sts:GetCallerIdentity`.
- `bedrock:InvokeModel` on Anthropic Claude models (optional; for wording only).
- Read/write **only** on the single `nexogent-reviews` DynamoDB table (Nexogent's own
  data — not the client's).

**No write, delete, modify, enable, suppress, or remediate** permission on the
client's AWS resources is ever requested.

## 5. Portal access & hardening
- Single role **superadmin** (= the MSP lead). Provide a **superadmin password** at
  deploy time (stored as a Kubernetes Secret; never committed).
- A **session secret** (≥ 32 bytes) for signed, stateless sessions that survive
  restarts and work across replicas. The Helm chart generates one if not supplied and
  reuses it on upgrade.
- **HTTPS:** a domain name and an **ACM certificate** are needed to serve the portal
  over HTTPS (the ALB then redirects HTTP → HTTPS). Without them the portal runs over
  HTTP and shows a warning; an **IP allow-list** (`inboundCidrs`) is then strongly
  advised.

## 6. Amazon Bedrock (optional)
- Model access for an Anthropic Claude model in the chosen region, and a valid
  payment method so the model subscription can complete.
- Bedrock is used only to improve wording of explanations/report narrative. If it is
  unavailable, every feature still works from built-in templates.

## 7. Slack alerts (optional)
- An incoming **Slack webhook** (`https://hooks.slack.com/...`) if alerts are wanted.
  Stored as a Secret; validated to be an HTTPS Slack URL; never returned by the API.

## 8. What we do NOT need
- No admin credentials to keep; deploy access is temporary.
- No paid AWS security services. GuardDuty / Security Hub / Inspector are shown in the
  portal as **disabled toggles** (free build) with a cost note.
- No change to running workloads.

## 9. Cost awareness
- EKS control plane and worker nodes bill hourly even when idle.
- The ALB and the on-demand DynamoDB table incur small ongoing costs.
- Bedrock is pay-per-use (only when wording is generated).
- Use **local mode** and **demo mode** for development and sales without a cluster.
- A billing budget alarm is recommended before deployment.

## 10. After handover
- The client holds full admin. Nexogent runs under the read-only IRSA role only.
- To remove it entirely: run `stop.sh --purge`, then confirm the ECR repos, the IAM
  policy/role, the DynamoDB table, the Kubernetes Role/RoleBinding, and the
  `nexogent` namespace are gone.
