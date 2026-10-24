# Nexogent — Product Overview (Free Build, v2)

> **Edition:** Free build. Uses no paid AWS security service — no GuardDuty, no
> Security Hub, no Inspector. Works on an account where none of those are switched
> on, including a free-tier account. The companion **paid build** adds those as
> optional read-only sources.

## What it is
Nexogent is an **AWS SecOps agent for managed service providers (MSPs)**. The MSP
deploys it into one managed client's own AWS account, where it runs inside the
client's Amazon EKS cluster. Each deployment serves one account — it is a per-client
product, not a shared SaaS. The client's data never leaves their account.

It is **read-only and recommend-only**. It never changes, deletes, restarts, or
remediates anything in the client account.

## What it does
Nexogent turns raw findings into the things an MSP actually hands a client:

1. **Discovers every resource** across **all enabled regions** and groups them by
   service (EC2, S3, IAM, Lambda, RDS, DynamoDB, EKS, CloudTrail).
2. **Audits** with its own read-only Go checks plus the open-source **Prowler**
   scanner (CIS / AWS FSBP). ScoutSuite and cloudsplaining can be enabled per client,
   all normalized behind one finding model.
3. **Gives findings a stable identity and a lifecycle** — open, fixed, regressed,
   accepted — so it can show what is new, what was fixed, and how long something has
   been open. One finding per problem, with every source that reported it merged.
4. **Ranks each finding by real risk** using a weighted rubric (severity + exposure
   + reachability + resource criticality), producing a 0–100 score and an account
   posture score. The rubric is editable in Settings.
5. **Builds attack paths** — hand-built chains such as "internet-open port → instance
   → admin role" — and shows the cheapest step that breaks the chain.
6. **Gives fix-as-code** — a Terraform snippet and an AWS CLI command for common
   findings, with what could break, always labelled "review before applying".
   Nexogent never runs them.
7. **Explains each finding in plain English** from built-in templates, optionally
   improved by **Amazon Bedrock**. Bedrock never detects anything and never changes
   a number; everything works with Bedrock unavailable.
8. **Produces branded reports** (Word, Markdown, print-ready HTML) with a review and
   approval step, and keeps a **history** so the score trend proves improvement.
9. **Alerts and records** — Slack alerts for new critical/high findings, regressions,
   new attack paths and expiring accepted risks; an audit log of every action.

## The MSP workflow
| Moment | What happens | What the MSP gets |
|---|---|---|
| **Onboarding** | Deploy, run a full scan + a Prowler deep scan | A baseline: score, ranked fixes, attack paths |
| **Weekly review** | Open the Overview | What changed since last scan; the next five fixes |
| **Monthly report** | Generate → review → approve → download | A branded Word report with the score trend |
| **Change window** | Audit a service/resource before and after | Proof a change worked, or an alert if it regressed |
| **Between reviews** | Scheduled scans + Slack alerts | Notice of a new critical finding, no one watching |

## The portal (risk-first)
A single sign-in role, **superadmin** (the MSP's security or cloud lead). The portal
is organised so the first screen answers "what is wrong and what do I fix first":

- **Overview** — posture score, trend, what changed since last scan, the top five
  fixes, attack-path count, and per-source scan status.
- **Findings** — filter by status/severity/service/source/region; open a finding to
  see why it matters, its fix-as-code, its sources, age, and a plan form.
- **Attack paths** — each chain with its evidence and where to break it.
- **Inventory** — the service cards, then per-service resource lists with a region
  **and** resource-type filter; per-resource security view with a quick-audit button.
- **Risks** — the accepted-risk register (reason, owner, expiry).
- **Reports** — create, review, approve, and download in Word / Markdown / HTML.
- **Activity** — the audit log, exportable as CSV / JSON.
- **Settings** — branding, sources, schedule, rubric, reviewer, Slack alerts, and the
  disabled native-service toggles.

## Scans
- **Quick audit** re-runs the own checks for one service or resource in seconds,
  without re-discovering the whole account.
- **Deep scan** starts a Prowler job on demand, shows its progress, and ingests the
  result.
- **Scheduled scans** keep posture current without anyone clicking.

## What it does NOT do (free build)
- It is read-only and recommend-only; fix-as-code is text a human reviews.
- No GuardDuty / Security Hub / Inspector. They appear as **disabled toggles** with a
  cost note and a pointer to the paid build.
- No AWS Organizations / multi-account view (one account per deployment).
- No named users / roles / SSO (single `superadmin` in this release).
- No ticketing (Jira / ServiceNow) integration; reports and the plan export instead.
- No Python or local ML model; the agent is pure Go. Bedrock is a hosted API used
  only to improve wording.

## Modes
- **Cluster** — Helm into the client's EKS (production).
- **Local** — `docker compose up` on a laptop against the MSP's own read-only AWS
  profile; no cluster needed.
- **Demo** — `DEMO_MODE=true` loads clearly fake sample data (no AWS calls) for sales
  demonstrations and portal testing; every page shows a "Demo data" banner.

## Technology
- **Pure Go** backend — small static binary, k8s-native, no interpreter sprawl.
- **React** portal (Vite + unprivileged Nginx), served behind an Application Load
  Balancer, light theme by default, accessible by keyboard.
- **Open-source scanners** — Prowler (default); ScoutSuite / cloudsplaining optional.
- **Amazon Bedrock** (Claude) for wording only, with an offline template fallback.
- **DynamoDB** for persistence (reviews, history, risk register, plans, reports,
  audit log, settings).
- **Helm** for per-client deployment; access via IRSA — no static keys in the cluster.

## Positioning (honest)
Nexogent does not reinvent detection. Detection is delegated to native read-only AWS
APIs and open-source scanners. Nexogent's value is the MSP layer on top: a stable
finding lifecycle and history, cross-source deduplication, risk ranking, attack
paths, fix-as-code, branded reports with approval, alerts, an audit log, and a
risk-first portal — delivered safely as read-only, inside the client's own account.
