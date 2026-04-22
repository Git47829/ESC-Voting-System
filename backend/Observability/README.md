# Observability (`backend/Observability`)

Observability stack used by ESC Voting:

- OTel Collector (`otel/opentelemetry-collector-contrib:0.94.0`)
- Prometheus (`prom/prometheus:v2.51.1`)
- Grafana (`grafana/grafana:10.4.2`)
- Loki (`grafana/loki:2.9.6`)
- Tempo (`grafana/tempo:2.4.2` in root compose; `latest` in local observability compose)

## Config files

- Collector: `OTel/src/otel-collector.yml`
- Prometheus: `Prometheus/src/prometheus.yml`
- Tempo: `Tempo/src/tempo.yml`
- Grafana provisioning: `grafana/src/provisioning/**`

## Collector pipelines (current config)

- **Receivers:** `otlp` (gRPC + HTTP), `filelog` (tails `/var/log/traefik/access.log`)
- **Metrics pipeline:** `otlp -> batch -> prometheus` (`:9464`)
- **Traces pipeline:** `otlp -> batch -> otlp(tempo:4417) + logging`
- **Logs pipeline:** `otlp + filelog -> batch + loki label processors -> loki`

## Prometheus scrape jobs

`prometheus.yml` scrapes:

- `otel-collector:9464` and `otel-collector:8888`
- `esc-frontend:5000/metrics`
- `db-crud-api:8000/metrics/`
- `public-vote-converter:8090/metrics`
- `euromail:3000/metrics`
- `traefik:8082/metrics`
- `loki:3100/metrics`
- `tempo:3200/metrics`

## Access paths behind Traefik (root `docker-compose.yaml`)

- `/grafana`
- `/prometheus`
- `/tempo`
- `/loki`

OTLP ingest ports `4317`/`4318` are internal-only in the root stack.

## Local observability-only compose

`backend/Observability/docker-compose.yml` is a standalone stack for observability components and exposes ports directly (e.g. `4317`, `4318`, `9464`, `9090`, `3000`, `3100`, `3200`).
