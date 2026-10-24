# Nexogent — Verification checklist

Run after deploying or upgrading. Automated checks run without AWS; the manual
section needs a real account (do once, at handover).

## Automated (no AWS needed)
```bash
go build ./...
go vet ./...
gofmt -l .                      # must print nothing
go test ./... -race -count=1

cd dashboard && npm ci && npm run build && cd ..   # or: docker build ./dashboard

helm lint deploy/helm/nexogent
helm template nexogent deploy/helm/nexogent --set ingress.certificateArn=arn:aws:acm:us-east-1:111:certificate/x >/dev/null
```

## Demo smoke (no AWS)
```bash
DEMO_MODE=true SUPERADMIN_PASSWORD=demo SESSION_SECRET=$(openssl rand -hex 32) go run ./cmd/nexogent &
sleep 2
TOKEN=$(curl -s -XPOST localhost:8080/api/v1/auth/login -d '{"password":"demo"}' | jq -r .token)
for p in overview findings paths risks reports history activity settings; do
  printf "%-10s " $p; curl -s -o /dev/null -w "%{http_code}\n" -H "Authorization: Bearer $TOKEN" localhost:8080/api/v1/$p
done
```
(Demo mode has auth disabled, so the no-token checks are exercised separately below.)

## Auth gate (non-demo)
```bash
SUPERADMIN_PASSWORD=secret SESSION_SECRET=$(openssl rand -hex 32) go run ./cmd/nexogent &
curl -s -o /dev/null -w "%{http_code}\n" localhost:8080/api/v1/overview            # 401
curl -s -o /dev/null -w "%{http_code}\n" -XPOST localhost:8080/api/v1/auth/login -d '{"password":"nope"}'  # 401
TOKEN=$(curl -s -XPOST localhost:8080/api/v1/auth/login -d '{"password":"secret"}' | jq -r .token)
curl -s -o /dev/null -w "%{http_code}\n" -H "Authorization: Bearer $TOKEN" localhost:8080/api/v1/services   # 200
curl -s -o /dev/null -w "%{http_code}\n" "localhost:8080/api/v1/services?token=$TOKEN"                        # 401 (no ?token=)
```

## Manual, on a real account (phase 7)
1. Run a full audit; confirm the CloudTrail card shows real trails.
2. Quick-audit one resource; confirm it's fast and leaves Prowler findings alone.
3. Run a deep scan; watch it go running → completed and ingest findings.
4. Fix a finding in the account, rescan → it becomes **fixed**; revert, rescan →
   **regressed** (and keeps its first-seen).
5. Accept a risk; confirm the posture score changes and it is excluded.
6. Create → submit → approve → download a Word report; open it in Word/LibreOffice.
7. Configure a Slack webhook; confirm a test alert arrives and is not repeated.
8. Scale the backend to 2 replicas; confirm a session from one is accepted by the other.
```

Record anything you could not run and why. Never claim a check passed that you did
not run.
