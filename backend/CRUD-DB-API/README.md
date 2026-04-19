# CRUD DB API (`backend/CRUD-DB-API`)

Go backend service for ESC Voting. It serves REST endpoints on `:8000` and gRPC on `:50051`.

## Startup

Typical local stack startup:

```bash
docker compose up -d db euromail api
```

## Ports

- `8000` HTTP (REST + `/metrics/`)
- `50051` gRPC (`VoteService`)

## Runtime dependencies

- **Required:** MySQL (`DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASS`)
- **Optional but used when available:** Redis (rate limits, auth/session/cache), RabbitMQ (vote fanout + email jobs)

## Auth model (current code)

Protected routes use headers, not `?Token=`:

- `Authorization: Bearer <password-or-code>`
- `X-Email: <user email>`

Credential checks:

- admin: `adminMail` + bcrypt-hashed `adminPassword`
- jury: `juryMail1..3` + bcrypt-hashed `juryPassword1..3`

2FA flow endpoints:

- `POST /auth/login` with JSON `{email,password,role}` (`role`: `admin` or `jury`)
- `POST /auth/verify` with JSON `{email,code}`

Legacy token endpoints still exist:

- `GET /auth/requestToken`
- `GET /auth/verifyToken/{token}`

## REST endpoints

### Public

- `GET /health`
- `GET /votes/`
- `GET /countries/`
- `GET /countryByName/{NAME}`
- `GET /songs/`
- `GET /songByID/{ID}`
- `GET /contest/current`
- `GET /metrics/`
- `POST /vote/` (query params: `ownCountry`, `phoneNum`, `songID`, `points`)

### Protected (admin/jury middleware)

- `GET /admin/authenticate`
- `GET /jury/authenticate`
- `POST /admin/open`
- `POST /admin/close`
- `DELETE /admin/deleteVotes/`
- `POST /admin/addCountry/`
- `POST /admin/addSong/`
- `POST /admin/addArtist/`
- `POST /admin/addInterpret/`
- `POST /admin/startContest`
- `POST /admin/advanceContest`
- `POST /jury/vote/`

## gRPC service (`VoteService`)

- `StreamVotes(VoteStreamRequest) returns (stream Vote)`
  - optional historical snapshot
  - live events come from RabbitMQ fanout exchange `votes.fanout`
- `GetSongsWithVotes(GetSongsRequest) returns (GetSongsResponse)`
  - used by PublicVoteConverter

## Environment variables

| Variable | Default / note |
|---|---|
| `DB_HOST` | `localhost` |
| `DB_PORT` | `3306` |
| `DB_NAME` | `esc_voting` |
| `DB_USER` | `root` |
| `DB_PASS` | empty |
| `adminMail` | required for admin auth |
| `adminPassword` | bcrypt hash |
| `juryMail1..3` | jury emails |
| `juryPassword1..3` | bcrypt hashes |
| `COOKIESIGNINGKEY` | optional; random secret generated if missing |
| `REDIS_URL` | `redis:6379` |
| `RABBITMQ_URL` | `amqp://guest:guest@rabbitmq:5672/` |
| `OTEL_EXPORTER_OTLP_HTTP_ENDPOINT` | `localhost:4318` |

## Observability

- Prometheus endpoint: `GET /metrics/`
- OpenTelemetry traces/logs exported via OTLP HTTP
- Structured logs via `log/slog`
