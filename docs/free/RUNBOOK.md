# Nexogent — Operations Runbook

Common issues and how to resolve them. All operations are read-only on the client
account; none change client resources.

## A scan seems stuck
- Check the backend logs: `kubectl -n nexogent logs deploy/nexogent-backend`.
- A full scan across many regions can take a minute. The in-process scheduler and
  the HTTP `POST /api/v1/review` are single-flight (a second request returns 409).

## Deep scan failed or never completes
- `kubectl -n nexogent get jobs -l nexogent/deep-scan=true` and describe the pod.
- Prowler exits non-zero when it finds failing controls; that is expected and the
  results are still written. The backend ingests on the first `succeeded` poll.
- If the Job can't mount the results volume, confirm the Prowler pod landed on the
  same node as the backend (podAffinity). For multi-node flexibility, move the PVC
  to EFS (ReadWriteMany) and drop the affinity.
- If deep scan returns 501, the cluster API wasn't reachable (local/demo) — use the
  scheduled Prowler CronJob instead.

## Prowler results file missing
- The CronJob writes to the shared PVC under `/data/prowler`. Check
  `kubectl -n nexogent get cronjob nexogent-prowler` and its last run.
- Until the first Prowler run completes, the backend shows a warning and uses only
  the own checks — this is expected, not an error.

## DynamoDB throttling
- The table is on-demand billing and should scale automatically. If you see
  throttling under heavy history writes, confirm the table is PAY_PER_REQUEST.
- Large reviews are chunked automatically to stay under the item-size limit.

## Rotate the superadmin password
```bash
helm upgrade nexogent deploy/helm/nexogent --namespace nexogent --reuse-values \
  --set-string auth.password="NEW-PASSWORD"
kubectl -n nexogent rollout restart deploy/nexogent-backend
```

## Rotate the session secret
Delete the `SESSION_SECRET` key from the `nexogent-auth` Secret (or the whole Secret)
and `helm upgrade` — the chart generates a fresh one. All existing sessions are
invalidated and users must sign in again.

## Scale down to stop billing
```bash
./stop.sh                 # helm uninstall + scale nodegroup to 0
./stop.sh --purge         # also remove IAM/ECR/DynamoDB/namespace/RBAC
```

## Bedrock unavailable
Explanations fall back to built-in templates automatically. Nothing breaks. To enable
live wording, grant model access and ensure the account has a valid payment method.
