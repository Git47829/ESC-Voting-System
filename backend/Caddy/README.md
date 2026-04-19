# Caddy Reverse Proxy

Caddy is the HTTPS entry point for this stack. Configuration lives in `src/Caddyfile`.

## Ports

- `80` → redirects to HTTPS
- `443` → public HTTPS ingress
- `2019` → Caddy admin API/metrics (internal; scraped by Prometheus as `caddy:2019/metrics`)

## Routing (from `src/Caddyfile`)

| Incoming path | Upstream |
|---|---|
| `/crud-api/*` | `db-crud-api:8000` |
| `/eurostats/*` | `eurostats:8880` |
| `/esc-converter/*` | `public-vote-converter:8090` |
| `/grafana*` | `grafana:3000` |
| `/prometheus*` | `prometheus:9090` |
| `/tempo/*` | `tempo:3200` |
| `/loki/*` | `loki:3100` |
| fallback | `esc-frontend:3001` |

`handle_path` routes strip their prefix before proxying.

## TLS

The site block uses `tls internal` with `on_demand`, so certificates are issued by Caddy’s local CA and stored in `caddy_data`.

## Logging

Caddy writes JSON access logs to `/var/log/caddy/access.log`. The OTel Collector tails this file and forwards logs to Loki.

## Notes

- OTLP collector ingest ports (`4317`, `4318`) are **not** proxied through Caddy.
- Caddy state persists in `caddy_data` and `caddy_config`.
