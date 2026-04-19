# Traefik Reverse Proxy

Traefik is the HTTPS entry point for the local `docker-compose.yaml` stack.

## Ports

- `80` → redirects to HTTPS (`web` → `websecure`)
- `443` → public HTTPS ingress
- `8082` → Prometheus metrics (`traefik:8082/metrics`, internal)

## Routing (Docker labels in root compose)

| Incoming path | Upstream | Prefix stripped |
|---|---|---|
| `/crud-api/*` | `db-crud-api:8000` | yes (`/crud-api`) |
| `/eurostats/*` | `eurostats:8880` | yes (`/eurostats`) |
| `/esc-converter/*` | `public-vote-converter:8090` | yes (`/esc-converter`) |
| `/grafana*` | `grafana:3000` | no |
| `/prometheus*` | `prometheus:9090` | yes (`/prometheus`) |
| `/tempo/*` | `tempo:3200` | yes (`/tempo`) |
| `/loki/*` | `loki:3100` | yes (`/loki`) |
| fallback | `esc-frontend:3001` | no |

## TLS

Routers use TLS on `websecure`. For local/dev use, Traefik serves its default self-signed certificate unless custom certs are configured.

## Logging

Traefik writes JSON access logs to `/var/log/traefik/access.log`. The OTel Collector tails this file and forwards logs to Loki.
