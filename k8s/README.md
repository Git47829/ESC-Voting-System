# Kubernetes deployment (k8s/)

This directory deploys the ESC Voting stack into namespace `esc-voting`.

## Images used by app Deployments

All app Deployments pull private GHCR images (tagged `latest`) and require `imagePullSecrets: ghcr-pull-secret`:

- `ghcr.io/git47829/esc-voting-crud-db-api:latest`
- `ghcr.io/git47829/esc-voting-public-vote-converter:latest`
- `ghcr.io/git47829/esc-voting-eurostats:latest`
- `ghcr.io/git47829/esc-voting-euromail:latest`
- `ghcr.io/git47829/esc-voting-esc-frontend:latest`

Data/observability components use public images from MySQL, Redis, RabbitMQ, OpenTelemetry, Prometheus, Grafana, Loki, and Tempo manifests under `k8s/data` and `k8s/observability`.

## Prerequisites

- Kubernetes cluster with Traefik CRDs available (`IngressRoute`, `Middleware`)
- `cert-manager` installed (required by `k8s/ingress/ingress.yaml` for `ClusterIssuer`/`Certificate`)
- A storage class named `local-path` (used by PVCs/StatefulSets)

Check storage class:

```bash
kubectl get storageclass
```

## Required secrets

### 1) GHCR pull secret

Create in namespace `esc-voting`:

```bash
kubectl create namespace esc-voting --dry-run=client -o yaml | kubectl apply -f -

kubectl -n esc-voting create secret docker-registry ghcr-pull-secret \
  --docker-server=ghcr.io \
  --docker-username="$GITHUB_USERNAME" \
  --docker-password="$GITHUB_TOKEN" \
  --dry-run=client -o yaml | kubectl apply -f -
```

### 2) Application secret `esc-secrets`

Create from template and set real values:

```bash
cp k8s/secrets.example.yaml k8s/secrets.yaml
```

`k8s/secrets.yaml` must contain:

- `metadata.name: esc-secrets`
- `metadata.namespace: esc-voting`
- keys used by manifests:
  - `MYSQL_ROOT_PASSWORD`
  - `MYSQL_PASSWORD`
  - `RESEND_API_KEY`
  - `COOKIESIGNINGKEY`
  - `PHONESIGNINGKEY`
  - `SESSION_SECRET`
  - `adminMail`, `adminPassword`
  - `juryMail1`, `juryPassword1`
  - `juryMail2`, `juryPassword2`
  - `juryMail3`, `juryPassword3`

## Deployment sequence

1. **Namespace + secrets**

```bash
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/secrets.yaml
```

2. **Config + stateful dependencies**

```bash
kubectl apply -f k8s/config/
kubectl apply -f k8s/data/
```

3. **Observability + apps + ingress**

```bash
kubectl apply -f k8s/observability/
kubectl apply -f k8s/apps/
kubectl apply -f k8s/ingress/ingress.yaml
```

Before applying ingress, replace placeholders in `k8s/ingress/ingress.yaml`:

- `esc.example.com` (certificate + host rules)
- `ops@example.com` (Let's Encrypt email)

## Services and exposed routes

### Internal services

- `crud-db-api` (`8000`, `50051`)
- `public-vote-converter` (`8090`)
- `eurostats` (`8880`)
- `euromail` (`3000`)
- `esc-frontend` (`3001`)
- `mysql` (`3306`, headless)
- `redis` (`6379`, headless)
- `rabbitmq` (`5672`, `15672`, headless)
- `otel-collector` (`4317`, `4318`, `9464`, `8888`)
- `prometheus` (`9090`)
- `grafana` (`3000`)
- `loki` (`3100`)
- `tempo` (`3200`, `4417`)

### Ingress routes (`Host: esc.example.com`)

- `/crud-api` -> `crud-db-api:8000` (prefix stripped)
- `/eurostats` -> `eurostats:8880` (prefix stripped)
- `/esc-converter` -> `public-vote-converter:8090` (prefix stripped)
- `/grafana` -> `grafana:3000`
- `/` -> `esc-frontend:3001`

TLS secret: `esc-voting-tls` (managed by cert-manager `Certificate`).

## Operations quick commands

```bash
# rollout/status
kubectl -n esc-voting get pods
kubectl -n esc-voting rollout status deployment/crud-db-api
kubectl -n esc-voting rollout status deployment/public-vote-converter
kubectl -n esc-voting rollout status deployment/eurostats
kubectl -n esc-voting rollout status deployment/euromail
kubectl -n esc-voting rollout status deployment/esc-frontend

# autoscaling
kubectl -n esc-voting get hpa

# logs
kubectl -n esc-voting logs deployment/crud-db-api --tail=200
kubectl -n esc-voting logs deployment/eurostats --tail=200

# local access to ops UIs
kubectl -n esc-voting port-forward svc/grafana 3000:3000
kubectl -n esc-voting port-forward svc/prometheus 9090:9090
kubectl -n esc-voting port-forward svc/rabbitmq 15672:15672
```
