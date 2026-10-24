# Local mode

Run Nexogent on a laptop against your own read-only AWS profile — no cluster.

```bash
cp .env.example .env     # set SUPERADMIN_PASSWORD, AWS_PROFILE, AWS_REGION, SESSION_SECRET
docker compose up        # backend :8080, portal :8081
```

- The portal is at `http://localhost:8081`, the API at `http://localhost:8080`.
- Your `~/.aws` is mounted **read-only**; no credentials are copied into any image.
- Deep scans are unavailable locally (`kube_available=false`); the button is hidden.
- To run Prowler once into the shared volume:
  ```bash
  docker compose --profile scan up prowler
  ```

## Demo mode (no AWS at all)
```bash
DEMO_MODE=true docker compose up
```
Loads synthetic data and makes no AWS calls. Every page shows a "Demo data" banner.
Or run the backend directly:
```bash
DEMO_MODE=true SUPERADMIN_PASSWORD=demo SESSION_SECRET=$(openssl rand -hex 32) go run ./cmd/nexogent
```
