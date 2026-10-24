# Nexogent — Prerequisites (for the client)

This is what you (the client) provide so we can deploy Nexogent into your AWS
account. Nexogent is **read-only**: it reads your account and reports; it changes
nothing. You keep full admin; we retain no access after handover.

---

## 1. AWS account & region
- One AWS account where Nexogent will run.
- A chosen region (e.g. `us-east-1`) where your resources and Amazon Bedrock access live.

## 2. Amazon EKS cluster
- An existing EKS cluster (or we help you create one).
- Permission to create one small namespace and workload on it.
- An **OIDC provider** on the cluster (needed for IRSA). We can enable this if absent.

## 3. Deploy-time access (temporary)
Provide **one** of the following so we can perform the install:
- A temporary IAM role we can assume, **or**
- Short-lived deploy credentials.

The deploy identity needs permission to:
- Create an **ECR repository** and push an image.
- Create a **read-only IAM policy** and an **IRSA role/service account**.
- Run **Helm/kubectl** against the cluster (EKS access entry or kubeconfig).
- Optionally update the node group scaling config.

> Do not send long-lived access keys over chat or email. Prefer an assumable role
> or short-lived credentials. Any credential shared insecurely should be rotated.

## 4. Read-only runtime permissions (the agent's own role)
At runtime Nexogent uses an **IRSA role** limited to read-only actions. It is created
during deploy and attached to the agent pod. It allows only:
- `Describe*/Get*/List*` on EC2, S3, IAM, Lambda, RDS, DynamoDB, ECS, EKS,
  IAM Access Analyzer.
- `sts:GetCallerIdentity`.
- `bedrock:InvokeModel` on Anthropic Claude models (for explanations).

No write, delete, modify, or remediate permission is requested — ever.

## 5. Amazon Bedrock
- **Model access** granted for an Anthropic Claude model in the chosen region
  (e.g. Claude Haiku). This powers the plain-English explanations.
- If Bedrock is not available, Nexogent still works using built-in offline
  explanations — the dashboard never breaks.

## 6. What we do NOT need
- We do **not** need admin credentials to keep. Deploy access is temporary.
- We do **not** enable paid services (GuardDuty, Security Hub, Inspector).
- We do **not** require any change to your running workloads.

## 7. Cost awareness
- EKS control plane and worker nodes bill per hour even when idle.
- Bedrock is pay-per-use (only when explanations are generated).
- A billing budget alarm is recommended before deployment.

## 8. After handover
- You hold full admin of your account.
- Nexogent runs under the read-only IRSA role only.
- To remove it entirely: run `stop.sh`, then delete the ECR repo, the IAM
  policy/role, and the `nexogent` namespace.
