# PublicVoteConverter (`backend/PublicVoteConverter`)

Go service that converts raw public votes into ESC ranking points. It is read-only and fetches song data from CRUD API via gRPC.

## Startup

```bash
docker compose up -d api public-vote-converter
```

## Port

- `8090` HTTP

## Endpoints

- `GET /health`
- `GET /metrics`
- `GET /api/esc-points`

`/api/esc-points` response includes:

- `songId`, `songName`, `country`, `countryId`
- `rawPublicVotes`
- `escPoints`
- `rank`

## Ranking logic

- Sort by `rawPublicVotes` descending
- Tie-break by `songId` ascending
- ESC table: `12, 10, 8, 7, 6, 5, 4, 3, 2, 1`, then `0`
- Songs with `rawPublicVotes == 0` always get `0`
- Returned `escPoints` are multiplied by `NUM_JURY_MEMBERS` (default `3`)

## gRPC dependency

On startup, the service connects to CRUD API gRPC (`GetSongsWithVotes`) with retries (20 attempts, 3s interval). It exits if connection cannot be established.

## Environment variables

| Variable | Default |
|---|---|
| `GRPC_HOST` | `db-crud-api` |
| `GRPC_PORT` | `50051` |
| `PORT` | `8090` |
| `NUM_JURY_MEMBERS` | `3` |
| `OTEL_EXPORTER_OTLP_HTTP_ENDPOINT` | `http://otel-collector:4318` |
| `ENVIRONMENT` | `production` (telemetry resource attribute) |

## Observability

- Prometheus: `esc_converter_http_requests_total`, `esc_converter_http_request_duration_seconds`
- OTLP traces/logs via HTTP exporter
