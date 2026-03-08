# CRUD DB API

The primary backend service for the ESC Voting System. A REST + gRPC server written in Go that owns all database interaction, authentication, rate limiting, and observability.

## Tech Stack

| Component | Detail |
|---|---|
| Language | Go 1.24 |
| HTTP Router | `net/http` (stdlib) |
| Database Driver | `github.com/go-sql-driver/mysql` v1.9 |
| Token Comparison | `crypto/subtle` (constant-time) |
| Rate Limiting | `golang.org/x/time/rate` (token bucket) |
| Tracing | OpenTelemetry SDK v1.40 — OTLP/HTTP exporter |
| Metrics | `prometheus/client_golang` v1.23 (promauto) |
| Logging | `log/slog` — structured JSON to stdout |
| gRPC | `google.golang.org/grpc` v1.79 |

## Ports

| Port | Protocol | Purpose |
|---|---|---|
| `8000` | HTTP | REST API + Prometheus metrics (internal only — not exposed to host) |
| `50051` | gRPC | Vote streaming (`VoteService`) (internal only) |

All external access is routed through Caddy. See the [root README](../../README.md) for URLs.

## Container

The service is built as a **multi-stage Docker image**:

1. **Builder stage** — uses `golang:latest` to compile a fully static binary (`CGO_ENABLED=0`) and a small static healthcheck binary.
2. **Runtime stage** — uses `gcr.io/distroless/static-debian12:nonroot`, a minimal image with no shell, no package manager, and no root user. The final image is ~5 MB.

The container runs as UID `65532` (`nonroot`) and is never started as root.

### Healthcheck

A dedicated `/healthcheck` binary is compiled in the builder stage and copied into the distroless image. It performs an `http.Get` to `localhost:8000/health` and exits `0` on success or `1` on failure. This is necessary because `distroless` provides no shell utilities such as `curl` or `wget`.

## Project Structure

```
CRUD-DB-API/
├── Dockerfile
├── README.md
└── src/
    ├── go.mod
    ├── go.sum
    ├── main.go          # All HTTP handlers, middleware, DB logic
    ├── grpc_server.go   # gRPC VoteService — streaming & broadcasting
    └── proto/           # Generated protobuf stubs
```

## REST API Endpoints

### Public (no auth required)

| Method | Path | Description |
|---|---|---|
| `GET` | `/health` | Health check — returns `200 OK` |
| `GET` | `/votes/` | All songs ranked by total points |
| `GET` | `/countries/` | List all registered countries |
| `GET` | `/countryByName/{NAME}` | Fetch a single country by ID |
| `GET` | `/songs/` | Full song list with artist, country, composer, and voting status |
| `GET` | `/songByID/{ID}` | Single song detail by ID |
| `POST` | `/vote/` | Cast a public vote (phone number + cookie deduplication) |
| `GET` | `/metrics/` | Prometheus metrics scrape endpoint |

### Admin (token required via `?Token=` query param)

| Method | Path | Description |
|---|---|---|
| `POST` | `/admin/open/` | Open the voting period |
| `POST` | `/admin/close` | Close the voting period |
| `DELETE` | `/admin/deleteVotes/` | Reset all vote counts to zero |
| `POST` | `/admin/addCountry/` | Register a new country |
| `POST` | `/admin/addSong/` | Add a new song entry |
| `POST` | `/admin/addArtist/` | Add a new artist (`Kuenstler`) |
| `POST` | `/admin/addInterpret/` | Add a new composer (`Komponist`) |
| `GET` | `/admin/authenticate` | Validate an admin token — returns `202` on success, `403` on failure |

### Jury (token required via `?Token=` query param)

| Method | Path | Description |
|---|---|---|
| `POST` | `/jury/vote/` | Cast a jury vote with a specific point value |
| `GET` | `/jury/authenticate` | Validate a jury token — returns `202` on success, `403` on failure |

## Rate Limits

Per-IP token-bucket rate limiting is applied globally via `RateLimitingMiddleware`.

| Endpoint | Requests/s | Burst |
|---|---|---|
| `GET /health` | 100 | 100 |
| `GET /votes/`, `/countries/`, `/songs/` | 10 | 20 |
| `POST /vote/` | 1 | 1 |
| `POST /jury/vote/` | 5 | 5 |
| `GET /admin/authenticate` | 5 | 5 |
| `GET /jury/authenticate` | 5 | 5 |
| `POST /admin/open/`, `/admin/close` | 2 | 2 |
| `POST /admin/add*` | 5 | 5 |
| `DELETE /admin/deleteVotes/` | 1 | 1 |
| `GET /metrics/` | unlimited | — |

## Observability

Every inbound HTTP request passes through `ObservabilityMiddleware`, which records:

- **Traces** — sent to the OTel Collector via OTLP/HTTP. Span includes method, path, status code, and request/response sizes.
- **Prometheus metrics** — four Histogram/Counter vectors exported at `/metrics/`:
  - `http_request_size_bytes` — request body size by method + path
  - `http_response_size_bytes` — response body size by method + path + status
  - `http_request_duration_seconds` — latency by method + path + status
  - `http_requests_total` — total request count by method + path + status

The tracer is configured via the `OTEL_EXPORTER_OTLP_HTTP_ENDPOINT` environment variable and identified as `esc-voting-crud-api`.

## gRPC Vote Streaming

`grpc_server.go` implements the `VoteService` protobuf service on port `50051`.

**`StreamVotes(VoteStreamRequest) returns (stream Vote)`**

- Clients subscribe and receive a real-time stream of `Vote` messages.
- If `include_historical = true`, the current vote totals for all songs are sent first (queried from MySQL), followed by live updates.
- Every time a public or jury vote is accepted by the HTTP handler, `NotifyVote()` is called to broadcast a fresh `Vote` message to all active gRPC subscribers.
- Subscriber channels are buffered (capacity 100). Slow consumers are skipped with a warning log rather than blocking the broadcaster.

The [EuroStats](../EuroStats/README.md) service connects as a gRPC client and consumes this stream.

## Design Patterns

| Pattern | Where used |
|---|---|
| **Middleware / Chain of Responsibility** | `RateLimitingMiddleware` → `ObservabilityMiddleware` → router |
| **Singleton** | `db`, `logger`, `tracer` package-level vars |
| **Decorator** | `responseWriter` wraps `http.ResponseWriter` to capture status code & response size |
| **Factory** | `getCLientLimiter` lazily creates per-IP `rate.Limiter` instances |
| **Repository** | Handler functions encapsulate all SQL queries |
| **Retry** | `connectToDatabase` retries with configurable delay and max attempts |
| **Observer / Pub-Sub** | gRPC subscriber channel list in `voteServer` |

## Authentication

Admin and jury endpoints verify a shared token passed as `?Token=<value>`. The incoming token is compared against the corresponding environment variable using a **constant-time comparison** (`crypto/subtle.ConstantTimeCompare`) to prevent timing attacks.

| Endpoint | Checked against | Environment variable(s) |
|---|---|---|
| `GET /admin/authenticate` + all `/admin/*` routes | `checkAccessAdmin` | `adminPassword` (plaintext token) |
| `GET /jury/authenticate` + `/jury/vote/` | `checkAccessJury` | `juryPassword1`, `juryPassword2`, `juryPassword3` (plaintext tokens) |

The dedicated authenticate endpoints are used by the frontend login flow to validate a token before establishing a session. They return `HTTP 202` with `{"message": "..."}` on success and `HTTP 403` with `{"error": "..."}` on failure.

> **Example `.env` entries:**
> ```
> adminPassword=my-secret-admin-token
> juryPassword1=jury-token-one
> juryPassword2=jury-token-two
> juryPassword3=jury-token-three
> ```

## Voting Logic (`POST /vote/`)

1. Verifies that voting is currently open (`Voting_Status.isOpen = true`).
2. Checks the song exists and belongs to a different country than the voter's own.
3. Hashes the provided phone number and checks it against `Phone_Nums` — one vote per number.
4. Sets a cookie (`voted`) to prevent double-voting from the same browser session.
5. Increments `Song.PublikumsPunkte` by the public vote weight.
6. Calls `NotifyVote()` to push the update to all gRPC subscribers.

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `DB_HOST` | `localhost` | MySQL hostname |
| `DB_PORT` | `3306` | MySQL port |
| `DB_NAME` | `esc_voting` | Database name |
| `DB_USER` | `root` | Database user |
| `DB_PASS` | *(empty)* | Database password |
| `adminPassword` | *(required)* | Plaintext admin token |
| `juryPassword1` | *(required)* | Plaintext jury token #1 |
| `juryPassword2` | *(optional)* | Plaintext jury token #2 |
| `juryPassword3` | *(optional)* | Plaintext jury token #3 |
| `OTEL_EXPORTER_OTLP_HTTP_ENDPOINT` | `localhost:4318` | OTel Collector HTTP endpoint |