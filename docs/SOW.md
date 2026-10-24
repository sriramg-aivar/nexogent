# Nexogent — Statement of Work

| | |
|---|---|
| **Product** | Nexogent — AWS SecOps agent |
| **Delivery** | Deployed into one client AWS account (per-client, not SaaS) |
| **Nature** | Read-only security posture agent, runs 24/7 in the client's EKS |
| **Access** | Temporary scoped deploy access from the client; handover after install |
| **Document status** | v1.0 |

---

## 1. Summary
Nexogent is deployed into a client's AWS account and continuously reports the
account's security posture. It discovers all resources (grouped by service), runs
read-only checks and the open-source Prowler scanner, ranks findings by risk, and
explains each one via Amazon Bedrock in a dashboard. It changes nothing in the
account.

## 2. Scope of work

### 2.1 In scope
- **Resource discovery** across supported services (EC2, S3, IAM, Lambda, RDS,
  DynamoDB, ECS, EKS), grouped by service in the dashboard.
- **Read-only security checks**: Nexogent's own checks plus Prowler
  (CIS / AWS Foundational Security Best Practices controls).
- **Risk ranking** by rubric: severity, exposure, reachability, resource criticality.
- **Posture score** (0–100) and rating for the whole account.
- **AI explanations** of findings via Amazon Bedrock, with an offline fallback.
- **Dashboard**: select a service or resource, view its findings and posture,
  open a finding for an explanation and remediation steps.
- **Deployment** into the client's Amazon EKS via the Helm chart.
- **start/stop scripts** for full spin-up and teardown.

### 2.2 Out of scope
- Any change, deletion, remediation, restart, or reconfiguration of AWS resources.
- Multi-tenant / shared SaaS operation. One deployment serves one client account.
- Managed 24×7 SOC staffing, incident response, or compliance certification.
- Penetration testing of resources not owned by the client.
- Paid AWS security services (GuardDuty, Security Hub, Inspector) — Nexogent uses
  free read-only APIs and open-source Prowler instead.
- Any Python / local ML model. The AI layer is Amazon Bedrock only.

## 3. Architecture
- **Agent / API** — pure Go, single container, runs in the client's EKS.
- **Discovery** — read-only AWS SDK calls, grouped by service.
- **Collectors** — Go own-checks + Prowler (open source) run read-only.
- **Ranking** — deterministic rubric computed in Go.
- **Explanations** — Amazon Bedrock (Claude) with offline fallback.
- **Dashboard** — React.
- **Access in-cluster** — IAM Roles for Service Accounts (IRSA), read-only policy.

## 4. Risk rubric
Each finding gets a 0–100 risk score from four weighted factors:

| Factor | Weight | Meaning |
|---|---|---|
| Severity | 0.40 | The tool's own severity |
| Exposure | 0.25 | Internet-facing / public? |
| Reachability | 0.20 | Can it actually be reached/used? |
| Resource criticality | 0.15 | Holds data / grants power? |

Posture score starts at 100 and subtracts per finding
(Critical −20, High −12, Medium −5, Low −2). Ratings: 90+ Excellent, 75+ Good,
60+ Needs Improvement, else Critical.

## 5. Deliverables
1. The Nexogent agent (Go) and dashboard (React).
2. Helm chart for per-client deployment.
3. `start.sh` / `stop.sh` for spin-up and teardown.
4. Docs: PRODUCT.md, SOW.md, DEPLOYMENT.md, PREREQUISITES.md.

## 6. Safety principles
- Read-only IAM role; no write/remediate permissions are ever requested.
- Runs only in the one client account that is agreed.
- No static AWS keys inside the cluster — access is via IRSA.
- Client retains full admin; we do not retain access after handover.

## 7. Honest framing
Nexogent delegates detection to native read-only AWS APIs and the open-source
Prowler. Its contribution is the inventory, ranking, explanations, dashboard, and
safe per-client deployment — not a new detection engine.
