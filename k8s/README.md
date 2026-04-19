# K3s deployment with private GHCR images

This K8s setup expects these 5 private images in GHCR:

- `ghcr.io/git47829/esc-voting-crud-db-api:latest`
- `ghcr.io/git47829/esc-voting-public-vote-converter:latest`
- `ghcr.io/git47829/esc-voting-eurostats:latest`
- `ghcr.io/git47829/esc-voting-euromail:latest`
- `ghcr.io/git47829/esc-voting-esc-frontend:latest`

## Storage assumption (PVC readiness)

Persistent workloads in this repo (`mysql`, `redis`, `rabbitmq`, `prometheus`, `loki`, `tempo`, `grafana`) are configured with:

```yaml
storageClassName: local-path
```

This assumes a K3s cluster with the default **local-path provisioner** enabled.

Verify before deploy:

```bash
kubectl get storageclass
```

You should see a `local-path` storage class. If your cluster uses another class, update the PVCs/StatefulSet `volumeClaimTemplates` accordingly before go-live.

## 1) CI-driven GHCR publishing (authoritative path)

Application images are published automatically by GitHub Actions (workflow file: `.github/workflows/ghcr-images.yml`).
Treat CI as the source of truth for image publishing; manual local pushes should only be used for emergency recovery.

### Workflow run conditions

- GHCR publish runs on trusted events only:
  - `workflow_run` of `CI` **after CI completes successfully** for `push` events on the default branch
  - manual `workflow_dispatch`
- It does **not** publish on `pull_request` CI runs, which avoids untrusted PR contexts pushing private images.

### Tagging strategy

The Kubernetes manifests in `k8s/apps/*.yaml` currently deploy `:latest`:

- `ghcr.io/git47829/esc-voting-crud-db-api:latest`
- `ghcr.io/git47829/esc-voting-public-vote-converter:latest`
- `ghcr.io/git47829/esc-voting-eurostats:latest`
- `ghcr.io/git47829/esc-voting-euromail:latest`
- `ghcr.io/git47829/esc-voting-esc-frontend:latest`

Current CI tags:

- immutable commit SHA (`type=sha,format=long`) on every publish run
- `latest` only when the run is on the repository default branch (`{{is_default_branch}}`)

So non-default-branch manual runs publish SHA tags only, while default-branch runs also refresh `:latest` used by current manifests.

### Minimal GitHub Actions permissions/secrets for private GHCR push

- Workflow permissions:
  - `contents: read`
  - `packages: write`
- Authentication:
  - Prefer built-in `${{ secrets.GITHUB_TOKEN }}` (no extra repo secret required) for publishing packages in this repository namespace.
  - If org policy blocks `GITHUB_TOKEN` package publish, use a PAT secret with `write:packages`.

## 2) Create image pull secret in namespace `esc-voting`

```bash
kubectl create namespace esc-voting --dry-run=client -o yaml | kubectl apply -f -

kubectl -n esc-voting create secret docker-registry ghcr-pull-secret \
  --docker-server=ghcr.io \
  --docker-username="$GITHUB_USERNAME" \
  --docker-password="$GITHUB_TOKEN" \
  --dry-run=client -o yaml | kubectl apply -f -
```

For private pulls, use a token with at least `read:packages` (PAT is typical for cluster pull secrets).

## 3) Install cert-manager (required for public TLS certificates)

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.15.3/cert-manager.yaml
kubectl wait --for=condition=Available deployment -n cert-manager --all --timeout=180s
```

## 4) Configure ingress domain + ACME email

Edit `k8s/ingress/ingress.yaml` and replace:

- `esc.example.com` with your real public domain (used in `Certificate` + `IngressRoute` host match)
- `ops@example.com` with your operational email for Let's Encrypt account registration

The Traefik middlewares in that file strip external prefixes (for example `/crud-api/votes/` → `/votes/`) so upstream services keep their existing root-relative routes.

## 5) Create app secrets locally (do not commit)

`k8s/secrets.yaml` is intentionally gitignored. Create it from the tracked template:

```bash
cp k8s/secrets.example.yaml k8s/secrets.yaml
```

Then set real values in `k8s/secrets.yaml` and ensure:

```yaml
metadata:
  name: esc-secrets
```

## 6) Deploy manifests (recursive apply)

```bash
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/secrets.yaml
kubectl apply -R -f k8s/
```

`k8s/secrets.example.yaml` uses `esc-secrets-example`, so it is safe to keep in the repo while your real `k8s/secrets.yaml` provides the required `esc-secrets`.

## 7) Post-deploy validation checklist

> This checklist is intended for a real K3s cluster. If you are running this from CI or a workstation without cluster access, run the same commands later in the target cluster and compare against the expected outcomes below.

### 7.1 Verify startup readiness and dependency order

```bash
kubectl -n esc-voting wait --for=condition=ready pod -l app=mysql --timeout=300s
kubectl -n esc-voting wait --for=condition=ready pod -l app=redis --timeout=300s
kubectl -n esc-voting wait --for=condition=ready pod -l app=rabbitmq --timeout=300s

kubectl -n esc-voting rollout status deployment/crud-db-api --timeout=300s
kubectl -n esc-voting rollout status deployment/public-vote-converter --timeout=300s
kubectl -n esc-voting rollout status deployment/eurostats --timeout=300s
kubectl -n esc-voting rollout status deployment/euromail --timeout=300s
kubectl -n esc-voting rollout status deployment/esc-frontend --timeout=300s

kubectl -n esc-voting get pods -o wide
```

Expected outcome:
- Data pods (`mysql`, `redis`, `rabbitmq`) become `Ready` first.
- App Deployments then complete rollout without `CrashLoopBackOff` caused by missing dependencies.

### 7.2 Test frontend session persistence across pod restart

```bash
BASE_URL="https://esc.example.com"

curl -k -c session.cookies -b session.cookies \
  -H "Content-Type: application/json" \
  -X POST "$BASE_URL/api/login" \
  -d '{"role":"admin","token":"<admin-token>"}'

curl -k -c session.cookies -b session.cookies "$BASE_URL/api/session"

FRONTEND_POD=$(kubectl -n esc-voting get pod -l app=esc-frontend -o jsonpath='{.items[0].metadata.name}')
kubectl -n esc-voting delete pod "$FRONTEND_POD"
kubectl -n esc-voting rollout status deployment/esc-frontend --timeout=180s

curl -k -c session.cookies -b session.cookies "$BASE_URL/api/session"
```

Expected outcome:
- Last `/api/session` response remains authenticated (`"authenticated": true`) after pod restart (Redis-backed session survives).

### 7.3 Test vote broadcast with scaled `crud-db-api` and all EuroStats pods

```bash
kubectl -n esc-voting scale deployment/crud-db-api --replicas=4
kubectl -n esc-voting scale deployment/eurostats --replicas=3
kubectl -n esc-voting rollout status deployment/crud-db-api --timeout=300s
kubectl -n esc-voting rollout status deployment/eurostats --timeout=300s

curl -k -X POST \
  "https://esc.example.com/crud-api/vote/?ownCountry=DE&phoneNum=%2B4915112345678&songID=2&points=4"

for p in $(kubectl -n esc-voting get pods -l app=eurostats -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}'); do
  echo "== $p =="
  kubectl -n esc-voting logs "$p" --since=60s | grep "Processing vote" | tail -n 1
done
```

Expected outcome:
- Every EuroStats pod shows a recent `Processing vote:` line for the submitted vote (fanout + per-pod consumer behavior).

### 7.4 Test WebSocket cross-pod behavior while scaling EuroStats

```bash
kubectl -n esc-voting scale deployment/eurostats --replicas=4
kubectl -n esc-voting rollout status deployment/eurostats --timeout=300s

kubectl -n esc-voting scale deployment/eurostats --replicas=2
kubectl -n esc-voting rollout status deployment/eurostats --timeout=300s
```

Then, while one or more clients are connected to `wss://esc.example.com/eurostats/ws/stats`, submit votes and confirm updates continue arriving during scale-up/down.

Expected outcome:
- Connected clients continue receiving stats updates/pings.
- No prolonged disconnect storm in `kubectl -n esc-voting logs deployment/eurostats --since=5m`.

### 7.5 Run stress test tool against ingress

```bash
cd tools/stress-test
go run . -url "https://esc.example.com/crud-api" -c 50
```

Use the TUI to run CRUD endpoints (`/health`, `/votes/`, `/countries/`, `/songs/`, `/contest/current`, `/vote/`) through ingress.

Optional second pass for converter path:

```bash
go run . -url "https://esc.example.com" -c 50
```

Then select `ESC Points` in the TUI.

Expected outcome:
- Stable response codes (mostly `2xx`) and no early timeout collapse at normal load levels.

### 7.6 Validate HPA scale up/down workflow

```bash
kubectl -n esc-voting get hpa
kubectl -n esc-voting get hpa crud-db-api eurostats esc-frontend public-vote-converter euromail -w
```

Generate load (for example with the stress test above), then observe:
- **Scale up:** replicas increase as CPU utilization rises above target.
- **Scale down:** after load stops and stabilization window passes, replicas return toward minimum.

If HPAs show `unknown` metrics, verify `metrics-server` is healthy in the cluster before go-live.

## 8) MySQL backup recommendation before go-live

Add a scheduled backup job for the MySQL StatefulSet and push dumps to durable object storage (S3/MinIO/NAS), not only to cluster-local volumes.

Practical baseline: run a Kubernetes `CronJob` every 6 hours with `mysqldump`, gzip the dump, and keep 14–30 days of retention externally.

Example command in the CronJob container:

```bash
mysqldump -h mysql -u root -p"$MYSQL_ROOT_PASSWORD" --single-transaction --quick esc_voting | gzip > /backup/esc_voting-$(date +%F-%H%M).sql.gz
```

Also test restore regularly on a non-production namespace before release.

All app Deployments in `k8s/apps/*.yaml` are configured with:

```yaml
imagePullSecrets:
  - name: ghcr-pull-secret
```
