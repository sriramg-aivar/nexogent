# MSP onboarding checklist

A step-by-step for bringing a new managed client onto Nexogent (free build).

## 1. Access & prerequisites
- [ ] Client provides temporary deploy access (assumable role or short-lived creds).
- [ ] Confirm the EKS cluster has an OIDC provider, the EBS CSI driver, and the AWS
      Load Balancer Controller (see `PREREQUISITES.md` / `deploy/CLUSTER_PREREQS.md`).
- [ ] Agree a "home" region; discovery covers all enabled regions automatically.
- [ ] (Optional) ACM certificate + domain for HTTPS; otherwise set an IP allow-list.
- [ ] (Optional) Bedrock model access; (optional) a Slack webhook for alerts.
- [ ] Choose a superadmin password (never committed).

## 2. Deploy
- [ ] `export SUPERADMIN_PASSWORD=...`
- [ ] `./start.sh` (builds+pushes both images, ensures IAM/IRSA, DynamoDB table,
      RBAC, then `helm install`). Or follow `DEPLOYMENT.md` step by step.
- [ ] Confirm the ALB URL responds and the portal login works.

## 3. Baseline scan
- [ ] Sign in as superadmin.
- [ ] Run a full audit (Overview → Quick audit all).
- [ ] Run a deep scan (Prowler) and wait for it to complete.
- [ ] Review the Overview: score, attack paths, top fixes.

## 4. First review
- [ ] Mark genuine exceptions as accepted risks (reason, owner, expiry).
- [ ] Assign owner + effort + due date to the top findings (plan).

## 5. First report
- [ ] Reports → New report (account scope) → review the draft.
- [ ] Submit → approve → download the Word report; send it to the client.

## 6. Ongoing
- [ ] Confirm the scheduled scan interval (Settings → Schedule).
- [ ] Confirm alert rules and the Slack webhook (Settings → Alerts; send a test).
- [ ] Set branding (MSP + client name, logo).

## 7. Handover note
The client keeps full admin. Nexogent runs under the read-only IRSA role only. To
remove it entirely, run `./stop.sh --purge`.
