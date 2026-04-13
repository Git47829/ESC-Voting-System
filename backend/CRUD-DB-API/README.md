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
├── go.mod
├── go.sum
├── main.go              # Entry point — calls server.Run()
├── proto/               # Generated protobuf stubs
│   ├── votes.proto
│   ├── votes.pb.go
│   └── votes_grpc.pb.go
├── server/              # HTTP handlers, gRPC server, DB, and telemetry
│   ├── database.go
│   ├── grpc_server.go
│   ├── handlers.go
│   ├── handlers_admin.go
│   ├── handlers_admin_data.go
│   ├── handlers_auth.go
│   ├── handlers_contest.go
│   ├── handlers_jury.go
│   ├── handlers_songs.go
│   ├── handlers_votes.go
│   └── telemetry.go
└── tests/               # Integration tests
    ├── main_test.go
    ├── auth_test.go
    ├── admin_test.go
    ├── contest_test.go
    ├── cookie_test.go
    ├── countries_test.go
    ├── health_test.go
    ├── jury_test.go
    ├── songs_test.go
    ├── vote_handler_test.go
    └── votes_test.go
```

## REST API Endpoints

### Public (no auth required)

| Method | Path | Description |
|---|---|---|
| `GET` | `/health` | Health check — returns `200 OK` |
| `GET` | `/votes/` | All songs ranked by total points |
| `GET` | `/countries/` | List all registered countries |
| `GET` | `/countryByName/{NAME}` | Fetch a single country by name |
| `GET` | `/songs/` | Full song list with artist, country, composer, and voting status |
| `GET` | `/songByID/{ID}` | Single song detail by ID |
| `POST` | `/vote/` | Cast a public vote (phone number + cookie deduplication) |
| `GET` | `/contest/current/` | Current song in the active contest run, with full details and progress |
| `GET` | `/metrics/` | Prometheus metrics scrape endpoint |

### Admin (token required via `?Token=` query param)

| Method | Path | Description |
|---|---|---|
| `GET` | `/admin/authenticate` | Validate an admin token — returns `202` on success, `403` on failure |
| `POST` | `/admin/open/` | Open the voting period |
| `POST` | `/admin/close` | Close the voting period |
| `DELETE` | `/admin/deleteVotes/` | Reset all vote counts to zero |
| `POST` | `/admin/addCountry/` | Register a new country |
| `POST` | `/admin/addSong/` | Add a new song entry (accepts optional `YoutubeURL` parameter) |
| `POST` | `/admin/addArtist/` | Add a new artist (`Kuenstler`) |
| `POST` | `/admin/addInterpret/` | Add a new composer (`Komponist`) |
| `POST` | `/admin/startContest/` | Fetch all songs, shuffle into a random order, and start a new contest run |
| `POST` | `/admin/advanceContest/` | Advance the active contest to the next song |

### Jury (token required via `?Token=` query param)

| Method | Path | Description |
|---|---|---|
| `GET` | `/jury/authenticate` | Validate a jury token — returns `202` on success, `403` on failure |
| `POST` | `/jury/vote/` | Cast a jury vote with a specific point value |

## Contest Run

The contest run feature allows an admin to run through all registered songs one by one in a randomised order, with each song displayed on the live **Running Now** page (`/now`).

### Flow

1. Admin presses **Start Contest** in the admin panel → `POST /admin/startContest/`
   - All song IDs are fetched from the database.
   - They are shuffled using a Fisher-Yates shuffle seeded with `crypto/rand`.
   - Any previously active `Contest_Run` row is deactivated.
   - A new `Contest_Run` row is inserted with the shuffled order (stored as a JSON array), `CurrentIndex = 0`, and `IsActive = TRUE`.
2. Viewers visit `/now` → frontend calls `GET /contest/current/`
   - Returns full song details for the song at the current index, plus `currentIndex` and `totalSongs` for the progress bar.
3. Admin presses **Next Song** → `POST /admin/advanceContest/`
   - `CurrentIndex` is incremented by 1.
   - If the index would exceed the total number of songs, the run is marked `IsActive = FALSE` and a `finished: true` response is returned.
4. The `/now` page polls `/api/contest/current` every 5 seconds and reloads automatically when the song changes.

### `GET /contest/current/` Response

```json
{
  "message": "Success",
  "payload": {
    "runId": 1,
    "currentIndex": 0,
    "totalSongs": 3,
    "songId": 2,
    "songName": "Northern Lights",
    "youtubeUrl": "https://www.youtube.com/embed/VIDEO_ID",
    "countryId": "SE",
    "countryName": "Sweden",
    "artistId": 2,
    "artistFirstName": "Alice",
    "artistLastName": "Lindgren",
    "artistType": "duo",
    "publicVotes": 110,
    "juryVotes": 118,
    "totalVotes": 228,
    "votingIsOpen": true
  }
}
```

Returns `404` when no contest is active, `410 Gone` when the contest has finished.

### `POST /admin/startContest/` Parameters

| Parameter | Required | Description |
|---|---|---|
| `Token` | Yes | Admin token |

### `POST /admin/advanceContest/` Parameters

| Parameter | Required | Description |
|---|---|---|
| `Token` | Yes | Admin token |

### `POST /admin/addSong/` Parameters

| Parameter | Required | Description |
|---|---|---|
| `Token` | Yes | Admin token |
| `Name` | Yes | Song title |
| `Land` | Yes | Country ID (alpha-2, e.g. `DE`) |
| `ID` | Yes | Artist (`Kuenstler`) ID |
| `YoutubeURL` | No | YouTube embed URL — any YouTube URL format is accepted; the frontend normalizes it to `youtube.com/embed/VIDEO_ID` before submission |

## Rate Limits

Per-IP token-bucket rate limiting is applied globally via `RateLimitingMiddleware`.

| Endpoint | Requests/s | Burst |
|---|---|---|
| `GET /health` | 100 | 100 |
| `GET /votes/`, `/countries/`, `/songs/` | 10 | 20 |
| `GET /contest/current/` | 10 | 20 |
| `POST /vote/` | 1 | 1 |
| `POST /jury/vote/` | 5 | 5 |
| `GET /admin/authenticate` | 5 | 5 |
| `GET /jury/authenticate` | 5 | 5 |
| `POST /admin/open/`, `/admin/close` | 2 | 2 |
| `POST /admin/startContest/`, `/admin/advanceContest/` | 2 | 2 |
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

### `StreamVotes(VoteStreamRequest) returns (stream Vote)`

- Clients subscribe and receive a real-time stream of `Vote` messages.
- If `include_historical = true`, the current vote totals for all songs are sent first (queried from MySQL), followed by live updates.
- Every time a public or jury vote is accepted by the HTTP handler, `NotifyVote()` is called to broadcast a fresh `Vote` message to all active gRPC subscribers.
- Subscriber channels are buffered (capacity 100). Slow consumers are skipped with a warning log rather than blocking the broadcaster.

The [EuroStats](../EuroStats/README.md) service connects as a gRPC client and consumes this stream.

### `GetSongsWithVotes(GetSongsRequest) returns (GetSongsResponse)`

- Unary RPC that returns the current list of all songs with their public vote counts.
- Queries `Song` and `Land` tables and returns a `GetSongsResponse` containing a repeated `SongVoteData` with `song_id`, `song_name`, `country_id`, `country_name`, and `public_votes`.
- Used by the [PublicVoteConverter](../PublicVoteConverter/README.md) to retrieve song data without accessing the database directly.

### Proto definition

```
backend/CRUD-DB-API/src/proto/votes.proto
```

Generated Go stubs (`votes.pb.go`, `votes_grpc.pb.go`) are committed alongside the proto source. To regenerate after editing the proto:

```bash
cd backend/CRUD-DB-API
protoc --go_out=. --go-grpc_out=. \
  --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative \
  proto/votes.proto
```

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
4. Sets a signed cookie (`vote_state`) tracking remaining points and votes cast per song.
5. Increments `Song.PublikumsPunkte` by the allocated point value weighted by the public vote weight.
6. Calls `NotifyVote()` to push the update to all gRPC subscribers.

## Database Tables

| Table | Description |
|---|---|
| `Land` | Countries — ISO alpha-2 ID, name, pot assignment |
| `Kuenstler` | Artists — solo, duo, or group; linked to a country |
| `Komponist` | Composers — first and last name |
| `Song` | Songs — linked to country and artist; stores public, jury, and computed total points; optional `YoutubeURL` |
| `Song_Komponist` | Many-to-many join between songs and composers |
| `Voting_Status` | Single-row global flag controlling whether voting is open |
| `Phone_Nums` | Registry of bcrypt-hashed phone numbers that have already voted |
| `Contest_Run` | Active contest state — JSON array of shuffled song IDs, current index, start timestamp, and active flag |

### `Contest_Run` Schema

| Column | Type | Notes |
|---|---|---|
| `ID` | `INT` PK AI | Auto-increment |
| `SongOrder` | `JSON` | Shuffled array of song IDs, e.g. `[3, 1, 2]` |
| `CurrentIndex` | `INT` | Index into `SongOrder` for the song currently on stage |
| `StartedAt` | `DATETIME` | Set to `CURRENT_TIMESTAMP` on insert |
| `IsActive` | `BOOL` | `TRUE` for the current run; set to `FALSE` on finish or when a new contest starts |

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