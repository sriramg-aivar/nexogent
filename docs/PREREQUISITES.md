# Nexogent — Prerequisites (for the client)

This is what you (the client) provide so we can deploy Nexogent into your AWS
account. Nexogent is **read-only**: it reads your account and reports; it changes
nothing. You keep full admin; we retain no access after handover.

---

## 1. AWS account & region
- One AWS account where Nexogent will run.
- A chosen "home" region (e.g. `us-east-1`) for the deployment and Amazon Bedrock
  access. (Discovery itself covers **all enabled regions** automatically.)

## 2. Amazon EKS cluster
- An existing EKS cluster (or we help you create one).
- Permission to create one small namespace and workload on it.
- An **OIDC provider** on the cluster (needed for IRSA). We can enable this if absent.
- Two cluster add-ons (installed once, we can do this during deploy):
  - **Amazon EBS CSI driver** — for the shared volume scanners write results to.
  - **AWS Load Balancer Controller** — to expose the portal via an ALB.

## 3. Deploy-time access (temporary)
Provide **one** of the following so we can perform the install:
- A temporary IAM role we can assume, **or**
- Short-lived deploy credentials.

The deploy identity needs permission to:
- Create an **ECR repository** (two: agent + dashboard) and push images.
- Create a **read-only IAM policy** and an **IRSA role/service account**.
- Create a **DynamoDB table** (for review history).
- Install the two cluster add-ons above (if not already present).
- Run **Helm/kubectl** against the cluster (EKS access entry or kubeconfig).
- Optionally update the node group scaling config.

> Do not send long-lived access keys over chat or email. Prefer an assumable role
> or short-lived credentials. Any credential shared insecurely should be rotated.

## 4. Read-only runtime permissions (the agent's own role)
At runtime Nexogent uses an **IRSA role** limited to read-only actions. It is created
during deploy and attached to the agent and scanner pods. It allows only:
- `Describe*/Get*/List*` on EC2, S3, IAM, Lambda, RDS, DynamoDB, ECS, EKS,
  CloudTrail, IAM Access Analyzer, KMS, CloudWatch, Logs, SNS.
- `sts:GetCallerIdentity`.
- `bedrock:InvokeModel` on Anthropic Claude models (for explanations).
- Read/write **only** on the single `nexogent-reviews` DynamoDB table (Nexogent's
  own review history — not your data).

No write, delete, modify, or remediate permission on your AWS resources is requested
— ever.

## 5. Amazon Bedrock
- **Model access** granted for an Anthropic Claude model in the chosen region
  (e.g. Claude Haiku). This powers the plain-English explanations.
- The account must have a valid payment method so the model subscription can
  complete; otherwise `InvokeModel` is blocked at the account level.
- If Bedrock is not available, Nexogent still works using built-in offline
  explanations — the portal never breaks.

## 6. Portal access (superadmin)
- v1 has a single role, **superadmin** (= you, the client).
- Provide a **superadmin password** at deploy time (we set it as a Kubernetes
  Secret; it is never committed). The portal login gate uses it so findings are not
  publicly readable.

## 7. What we do NOT need
- We do **not** need admin credentials to keep. Deploy access is temporary.
- We do **not** enable paid services (GuardDuty, Security Hub, Inspector). They are
  shown in the portal as disabled toggles for a future version.
- We do **not** require any change to your running workloads.

## 8. Cost awareness
- EKS control plane and worker nodes bill per hour even when idle.
- The ALB and the DynamoDB table (on-demand) incur small ongoing costs.
- Bedrock is pay-per-use (only when explanations are generated).
- A billing budget alarm is recommended before deployment.

## 9. After handover
- You hold full admin of your account.
- Nexogent runs under the read-only IRSA role only.
- To remove it entirely: run `stop.sh --purge`, then confirm the ECR repos, the IAM
  policy/role, the DynamoDB table, and the `nexogent` namespace are gone.
