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
account's security posture. It discovers all resources across all enabled regions
(grouped by service), runs read-only checks and the open-source Prowler scanner,
ranks findings by risk, and explains each one via Amazon Bedrock in a superadmin
portal. It changes nothing in the account.

## 2. Scope of work

### 2.1 In scope
- **Multi-region resource discovery** across supported services (EC2, S3, IAM,
  Lambda, RDS, DynamoDB, EKS, CloudTrail), grouped by service, each resource
  tagged with its region.
- **Read-only security checks**: Nexogent's own checks (IAM MFA/admin, S3 public/
  encryption, EC2 security groups, EBS encryption, EKS public endpoint, RDS public/
  encryption, Lambda public URL, DynamoDB encryption, CloudTrail multi-region) plus
  **Prowler** (CIS / AWS FSBP). ScoutSuite and cloudsplaining can be enabled per
  client behind the same normalized finding model.
- **Risk ranking** by rubric: severity, exposure, reachability, resource criticality.
- **Posture score** (0–100) and rating for the whole account.
- **AI explanations** of findings via Amazon Bedrock, with an offline fallback.
- **Superadmin portal**: single-role login; services landing; per-service resource
  list with a region filter; per-resource security/audit page with findings,
  explanations, and remediation.
- **On-demand audit** of a service or resource, plus **scheduled background scans**.
- **Persistence** of review history (DynamoDB) so reports survive restarts.
- **Deployment** into the client's Amazon EKS via the Helm chart (public URL via an
  ALB Ingress), with access via IRSA.
- **start/stop scripts** for full spin-up and teardown.

### 2.2 Out of scope
- Any change, deletion, remediation, restart, or reconfiguration of AWS resources.
- Multi-tenant / shared SaaS operation. One deployment serves one client account.
- Multi-role / multi-user access (v1 has a single `superadmin` role = the client).
- GuardDuty, Security Hub, and Inspector integration — shown as **disabled toggles**
  in v1 and deferred to a later version.
- Managed 24×7 SOC staffing, incident response, or compliance certification.
- Penetration testing of resources not owned by the client.
- Any Python / local ML model. The AI layer is Amazon Bedrock only.

## 3. Architecture
Deployed as **microservices** in the client's EKS namespace:
- **Backend / API** — pure Go, read-only AWS SDK calls across all enabled regions,
  risk ranking, review persistence, and the HTTP API.
- **Frontend** — React portal (Vite) served by Nginx; proxies the API.
- **Scanners** — Prowler runs as a Kubernetes CronJob (ScoutSuite / cloudsplaining
  optional), writing results the backend ingests behind one finding model.
- **Explanations** — Amazon Bedrock (Claude) with offline fallback.
- **Persistence** — DynamoDB table for review history.
- **Access in-cluster** — IAM Roles for Service Accounts (IRSA), read-only policy,
  no static keys.
- **Exposure** — an Application Load Balancer Ingress (`/` → frontend, `/api` →
  backend), gated by the superadmin login.

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
1. The Nexogent agent (Go) and superadmin portal (React).
2. Helm chart for per-client deployment (backend, frontend, scanner CronJobs,
   shared results volume, ALB Ingress, auth secret, DynamoDB persistence).
3. `start.sh` / `stop.sh` for spin-up and teardown.
4. Docs: PRODUCT.md, SOW.md, DEPLOYMENT.md, PREREQUISITES.md.

## 6. Safety principles
- Read-only IAM role; no write/remediate permissions on AWS resources are ever
  requested. (The one exception is read/write to the single DynamoDB table that
  stores Nexogent's own review history.)
- Runs only in the one client account that is agreed.
- No static AWS keys inside the cluster — access is via IRSA.
- The portal is protected by a superadmin login; it is not publicly readable.
- Client retains full admin; we do not retain access after handover.

## 7. Honest framing
Nexogent delegates detection to native read-only AWS APIs and open-source scanners
(Prowler, optionally ScoutSuite / cloudsplaining). Its contribution is the
multi-region inventory, the per-resource security view, ranking, explanations, the
superadmin portal, on-demand + scheduled audits, and safe per-client deployment —
not a new detection engine.
