# PublicVoteConverter

Go microservice that converts raw public vote counts into ESC-style ranking points and exposes them as a JSON preview endpoint for the frontend.

## Overview

PublicVoteConverter is a **gRPC client** of the CRUD DB API. On each request it calls `GetSongsWithVotes` to retrieve the current public vote totals, ranks the songs by vote count, and maps each rank to the official ESC televote points table (12, 10, 8, 7, 6, 5, 4, 3, 2, 1). The result is returned as a JSON array, optionally scaled by a jury multiplier.

## Tech Stack

| Component | Detail |
|---|---|
| Language | Go 1.24 |
| HTTP Router | `net/http` (stdlib) |
| gRPC | `google.golang.org/grpc` v1.78 (client only) |
| Tracing | OpenTelemetry SDK v1.40 — OTLP/HTTP exporter |
| Metrics | `prometheus/client_golang` v1.23 (promauto) |
| Logging | `log/slog` — structured JSON to stdout |

## Port

| Port | Protocol | Purpose |
|---|---|---|
| `8090` | HTTP | REST API + Prometheus metrics (internal only) |

No port is published to the host. External access goes through Caddy → Frontend; the frontend fetches `http://public-vote-converter:8090/api/esc-points` over the internal Docker `frontend` network.

## Endpoint

### `GET /api/esc-points`

Returns a preview of public vote totals converted to ESC ranking points. Points are **not** written to the database — this is a read-only preview.

**Response**

```json
{
  "message": "ESC points preview (not yet applied)",
  "payload": [
    { "songId": 2, "songName": "Northern Lights", "country": "Sweden",  "countryId": "SWE", "rawPublicVotes": 110, "escPoints": 36, "rank": 1 },
    { "songId": 1, "songName": "Satellite Reprise","country": "Germany", "countryId": "DEU", "rawPublicVotes":  85, "escPoints": 30, "rank": 2 },
    { "songId": 3, "songName": "Parisian Nights",  "country": "France",  "countryId": "FRA", "rawPublicVotes":   0, "escPoints":  0, "rank": 3 }
  ]
}
```

**Ranking & Points**

Songs are ranked by `rawPublicVotes` descending; ties are broken by `songId` ascending. The ESC points table is:

| Rank | Points |
|---|---|
| 1st | 12 |
| 2nd | 10 |
| 3rd | 8 |
| 4th | 7 |
| 5th | 6 |
| 6th | 5 |
| 7th | 4 |
| 8th | 3 |
| 9th | 2 |
| 10th | 1 |
| 11th+ | 0 |

Songs with 0 raw votes always receive 0 ESC points regardless of rank.

**Jury scale multiplier**

The returned `escPoints` values are multiplied by `NUM_JURY_MEMBERS` (default `3`). This equalises the 50/50 jury vs. televote weighting: one set of public ESC points (max 12) is scaled to the same maximum as the combined jury scores.

### `GET /health`

Returns `200 OK`. Used by Docker Compose for the container healthcheck.

### `GET /metrics`

Prometheus scrape endpoint exposing:

| Metric | Type | Labels | Description |
|---|---|---|---|
| `esc_converter_http_requests_total` | Counter | `method`, `path`, `status` | Total HTTP requests |
| `esc_converter_http_request_duration_seconds` | Histogram | `method`, `path`, `status` | Request latency |

## gRPC Communication

PublicVoteConverter connects to the CRUD DB API's gRPC server on startup and calls `GetSongsWithVotes` on every incoming `/api/esc-points` request.

```
PublicVoteConverter  ──gRPC GetSongsWithVotes──►  CRUD DB API
     Port 8090                                    Port 50051
```

The connection is insecure (no TLS) — both services share the `backend` Docker network and TLS is terminated at the Caddy boundary for external traffic.

On startup, `connectToGRPC()` retries up to 20 times (3-second intervals), probing with an actual RPC call to ensure the CRUD API is ready before the service begins serving HTTP traffic.

## Project Structure

```
PublicVoteConverter/
├── Dockerfile
├── README.md
└── src/
    ├── go.mod
    ├── go.sum
    ├── main.go          # HTTP handlers, gRPC client, ranking logic, OTel setup
    └── proto/           # Generated protobuf stubs (client-side)
        ├── votes.proto
        ├── votes.pb.go
        └── votes_grpc.pb.go
```

### Regenerating proto stubs

The proto definition mirrors `backend/CRUD-DB-API/src/proto/votes.proto` but uses a different `go_package` (`esc-points-converter/proto`). After any proto changes:

```bash
cd backend/PublicVoteConverter/src
protoc --go_out=. --go-grpc_out=. \
  --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative \
  proto/votes.proto
```

## Container

Built as a **multi-stage Docker image**:

1. **Builder stage** — `golang:latest`; runs `go mod tidy && go mod download` then compiles a fully static binary and a small healthcheck binary.
2. **Runtime stage** — `gcr.io/distroless/static-debian12:nonroot`; final image has no shell and runs as UID `65532`.

### Healthcheck

A dedicated `/healthcheck` binary (compiled in the builder stage) performs `GET http://localhost:8090/health` and exits `0` on success or `1` on failure, as required by the distroless runtime.

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `GRPC_HOST` | `db-crud-api` | Hostname of the CRUD DB API gRPC server |
| `GRPC_PORT` | `50051` | Port of the CRUD DB API gRPC server |
| `PORT` | `8090` | HTTP port this service listens on |
| `NUM_JURY_MEMBERS` | `3` | Jury scale multiplier applied to ESC points |
| `OTEL_EXPORTER_OTLP_HTTP_ENDPOINT` | `http://otel-collector:4318` | OTel Collector HTTP endpoint |

## Observability

Every inbound HTTP request (except `/health` and `/metrics`) is wrapped by `observabilityMiddleware`, which:

- Opens a trace span via OpenTelemetry (exported to Loki via the OTel Collector)
- Records request duration and count in Prometheus
- Emits structured JSON log lines to stdout and forwards them to Loki via OTLP/HTTP

The service identifies itself as `service.name = esc-points-converter`.
