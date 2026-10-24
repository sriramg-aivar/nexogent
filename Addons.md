# Nexogent — Addons & Next Build (v1)

> This file captures the **new requirements** on top of what already exists in the
> repo, plus a ready-to-paste **prompt** for the build session, plus an honest list
> of **improvements**. Nothing else in the repo was modified when this file was added.

---

## A. Context recap (what already exists)
The repo already has (pure Go backend + React dashboard + Helm + Prowler + Bedrock):
- Resource discovery (EC2, S3, IAM, Lambda, RDS, DynamoDB, EKS)
- Own read-only checks + Prowler collector + risk ranking + posture score
- Bedrock explanations with offline fallback
- Dashboard grouped by service; Helm chart; start.sh/stop.sh; deployed to EKS via IRSA

This Addons file defines **what to build next (v1)**.

---

## B. New requirements (v1) — what the client experience must be

### B1. Single role: `superadmin`
- Only **one role** in v1: **`superadmin`**, and **superadmin = the client**.
- After we deploy the product into the client's AWS account, the client opens the
  portal as superadmin. (No other roles/users in v1.)

### B2. Portal landing = list of AWS services
- When superadmin opens the portal, it shows **all AWS services** as a list/grid
  (EC2, S3, IAM, Lambda, RDS, DynamoDB, EKS, … exactly as supported).

### B3. Service page = resources of that service, with region filter
- Clicking a service (e.g. **EC2**) opens that service's page.
- It lists **all resources of that service** (e.g. all running EC2 instances),
  **showing each resource's region**.
- There must be a **region-wise filter**. Selecting a region shows only that
  region's resources (e.g. filter → only that region's running EC2 instances).

### B4. Resource page = our security/audit page
- Clicking a specific resource (e.g. one EC2 instance) opens **our page** — the
  **security page** for that resource.
- This page shows the security view already described earlier:
  findings on that resource, risk ranking, plain-English explanation/remediation
  (Bedrock + offline fallback), and whether it's fine or has issues.

### B5. "Audit" runs the scanner on demand
- On the resource (or service) page there is an **Audit** action.
- Clicking **Audit only** runs the scanner **on demand** for that target.
- v1 scanner engine = **Prowler** (+ our own Go checks). (See section D for the
  additional open-source tools to add.)

### B6. GuardDuty / Security Hub / Inspector — DISABLED in v1
- **Do NOT connect** GuardDuty, Security Hub, or Inspector in this version.
- BUT show them in the UI as **toggle options** ("Enable these auditings") that are
  **disabled/greyed-out for now** — so they can be turned on in a later version.

---

## C. Build prompt (paste this into the Nexogent build session)

> Read `.kiro/product-context.json` and `Addons.md` in the repo root — Addons.md is
> the authoritative v1 spec. We are extending the existing pure-Go + React + Helm
> product (do not reintroduce Python; keep it read-only; no Aivar/Ingram naming).
>
> Build this v1 client experience:
> 1. **Single role `superadmin`** (superadmin = the client). Add a simple auth gate
>    on the dashboard (even basic for v1) so the portal isn't fully public.
> 2. **Portal landing lists all AWS services** (EC2, S3, IAM, Lambda, RDS, DynamoDB,
>    EKS, …) as cards/list.
> 3. **Service page**: clicking a service lists all its resources **with region**,
>    and a **region filter** that narrows the list (multi-region discovery required —
>    today discovery is single-region; extend it to enumerate all enabled regions).
> 4. **Resource page = the security/audit page**: clicking a resource shows its
>    findings, risk score, and Bedrock/offline explanation + remediation.
> 5. **Audit on demand**: an "Audit" button that runs the scanners for that
>    service/resource and returns findings (Prowler + own Go checks).
> 6. **GuardDuty / Security Hub / Inspector**: show as **disabled toggles** in the
>    UI ("Enable these auditings") — not wired to AWS in v1.
> 7. Keep everything read-only and deployable via the existing Helm chart.
>
> Also address the gaps noted in section E where feasible (persistence of reviews,
> multi-region discovery, scheduled runs), and wire in the extra open-source scanners
> in section D behind the same normalized Finding model.

---

## D. Security tools to add (I said "many security tools" — here they are)
v1 uses Prowler. These are additional **free / open-source**, mostly **read-only**
tools worth wrapping behind the same normalized `Finding` model (run as Jobs/CronJobs
in-cluster under the read-only IRSA role):

| Tool | What it adds | Notes |
|---|---|---|
| **Prowler** (already) | Broad CIS / AWS FSBP config checks | OCSF JSON ingested |
| **ScoutSuite** | Multi-service posture audit, nice per-service grouping | read-only; JSON output |
| **Steampipe** (+ AWS compliance mods) | SQL-queryable AWS posture, CIS/FSBP/PCI/HIPAA packs | great for region-wide queries |
| **cloudsplaining** | IAM policy risk (privilege-escalation, data-exfil paths) | deep IAM analysis |
| **PMapper** | IAM privilege-escalation graph between principals | complements cloudsplaining |
| **Trivy (AWS/k8s mode)** | Scans EKS workloads + misconfig + CVEs in-cluster | read-only |
| **kube-bench** | CIS Kubernetes benchmark for the EKS cluster itself | runs in-cluster |
| **kube-hunter** | Finds exploitable paths in the k8s cluster | read-only mode |

Recommendation for v1: **Prowler + ScoutSuite + cloudsplaining (IAM)** — the best
"bang for buck" trio that directly serves the per-service / per-resource audit view.

---

## E. Honest improvements to make (gaps in the current build)
These are real weaknesses in what exists today; fix as part of v1 where possible:

1. **Persistence** — reviews are stored **in memory only** (lost on pod restart, and
   broken with >1 replica). Add a store (DynamoDB fits: serverless, cheap, read/write
   to one table) so reports + history survive and the dashboard shows trends.
2. **Multi-region discovery** — discovery is **single-region** today. B3's region
   filter needs discovery across **all enabled regions**. Build a region enumerator.
3. **Scheduled 24/7 runs** — backend review is **on-demand only**. Add a scheduler
   (CronJob or in-process ticker) so posture is refreshed automatically, not just on
   button click. Keep the manual "Audit" button too.
4. **Auth** — the dashboard is currently a **public ALB with no login**. Even with a
   single `superadmin`, add a login gate (v1: a simple admin password/secret; later:
   Cognito/OIDC). A security product should not expose findings to anyone with the URL.
5. **Per-resource audit depth** — today findings are matched to resources loosely.
   Build a clear **resource → findings** mapping so the resource page is precise.
6. **More own-checks** — add RDS (public/unencrypted), Lambda (public URL/role),
   DynamoDB (encryption), CloudTrail (enabled/multi-region), EBS encryption.
7. **Report export** — add "Download report" (JSON/PDF) so the client can save/share.
8. **Threat view (later)** — v1 is misconfiguration auditing only. "Are there active
   threats?" needs GuardDuty (kept as a disabled toggle for now, per B6).

---

## F. Explicitly deferred to a later version
- GuardDuty / Security Hub / Inspector integration (shown as disabled toggles in v1).
- Any ML model / Python layer (Bedrock handles the AI; revisit only if truly needed).
- Multi-role / multi-user (v1 is single `superadmin` = client).
- Remediation / write actions (product stays **read-only, recommend-only**).
