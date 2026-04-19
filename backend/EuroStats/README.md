# EuroStats (`backend/EuroStats`)

FastAPI service (`:8880`) that consumes vote events from RabbitMQ and serves real-time vote/stat streams.

## Startup

This service expects RabbitMQ (and optionally Redis) to be reachable:

```bash
docker compose up -d eurostats
```

## Data flow (current implementation)

1. Consume vote messages from RabbitMQ fanout exchange `votes.fanout`
2. Append messages to in-memory dataframe
3. If Redis is configured, persist votes to `votes:all` and publish chart updates on `stats.broadcast`
4. Broadcast updates to WebSocket clients

## Endpoints

- `GET /health`
- `GET /votes/subscribe` → current accumulated votes (`{"votes":[...], "count": N}`)
- `WS /ws/votes` → sends snapshot on connect (if available), then keepalive pings
- `WS /ws/stats` → sends pie-chart payloads (`voters_by_country`, `votes_received_by_country`) as base64 PNG data URLs

## Environment variables

| Variable | Default |
|---|---|
| `RABBITMQ_URL` | `amqp://guest:guest@rabbitmq:5672/` |
| `REDIS_URL` | `redis://redis:6379` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `http://otel-collector:4317` |

## Docker/runtime notes

- Image: `python:3.11-slim`
- Runs as non-root user `appuser` (uid `1001`)
- Healthcheck: `GET http://localhost:8880/health`
- Uvicorn command: `python -m uvicorn main:app --host 0.0.0.0 --port 8880`

## Observability

- FastAPI is instrumented with OpenTelemetry
- Traces/metrics/logs exported via OTLP gRPC
