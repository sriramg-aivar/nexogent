# Nexogent

**An AWS SecOps agent you deploy into a client's own AWS account.**

Nexogent runs inside the client's Amazon EKS cluster, discovers every AWS resource
in the account (grouped by service), runs read-only security checks plus the
open-source **Prowler** scanner, ranks findings by real risk, and presents a
security posture dashboard with AI-generated explanations (via Amazon Bedrock).

It is **read-only and recommend-only** — it never changes, deletes, or remediates
anything in the account. Each client gets their own private deployment; this is a
per-client product, not a shared SaaS.

## Why Go
A single small static binary, k8s-native, no interpreter or dependency sprawl inside
the client's account. The AI layer is Amazon Bedrock (a hosted API), so no local ML
runtime is needed.

## Architecture (short)
- `cmd/nexogent` — the agent / API server (Go)
- `internal/discovery` — read-only AWS resource inventory, grouped by service
- `internal/collectors` — read-only security checks (+ Prowler, external tool)
- `internal/ranking` — deterministic risk rubric (severity/exposure/reachability/criticality)
- `internal/bedrock` — Bedrock explanations with offline fallback
- `internal/api` — HTTP API for the dashboard
- `dashboard/` — React dashboard
- `deploy/helm/nexogent` — Helm chart for per-client deployment

## Docs
- `docs/PRODUCT.md` — the product vision and how it works
- `docs/SOW.md` — scope of work
- `docs/DEPLOYMENT.md` — how it is deployed into a client account
- `docs/PREREQUISITES.md` — what access/resources the client must provide

## Status
Early build. Core engine (discovery, checks, ranking, Bedrock, API) works live
against a real AWS account, read-only.
