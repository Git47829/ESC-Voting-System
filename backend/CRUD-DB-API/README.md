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

## Auth model

CRUD-DB-API now issues JWT access/refresh tokens and stores them in HttpOnly cookies:

- access cookie: `esc_access_token`
- refresh cookie: `esc_refresh_token`
- claims: `sub=email`, `role`, `iat`, `exp`

Roles: `admin`, `jury`, `user`.

Auth endpoints:

- `POST /auth/login` with JSON `{email,password,role}`
- `GET|POST /auth/verify` (requires valid access token)
- `GET /auth/me` (requires valid access token)
- `POST /auth/refresh` (requires valid refresh cookie)
- `POST /auth/logout` (revokes refresh session + clears cookies)

Refresh-token sessions are validated server-side (Redis when available, in-memory fallback).

## REST endpoints

### Public

- `GET /health`
- `GET /votes/`
- `GET /countries/`
- `GET /countryByName/{NAME}`
- `GET /songs/`
- `GET /songByID/{ID}`
- `GET /contest/current`
- `GET /results`
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
| `userMail` | optional user email (for `role=user`) |
| `userPassword` | optional bcrypt hash (for `role=user`) |
| `JWT_SECRET` | preferred signing secret for JWTs |
| `ACCESS_TOKEN_TTL_MINUTES` | default `15` |
| `REFRESH_TOKEN_TTL_HOURS` | default `168` |
| `AUTH_COOKIE_SECURE` | `true` unless set to `false` |
| `NUM_JURY_MEMBERS` | default `3` (for `/results` public-point scaling) |
| `COOKIESIGNINGKEY` | optional; random secret generated if missing |
| `REDIS_URL` | `redis:6379` |
| `RABBITMQ_URL` | `amqp://guest:guest@rabbitmq:5672/` |
| `OTEL_EXPORTER_OTLP_HTTP_ENDPOINT` | `localhost:4318` |

## Observability

- Prometheus endpoint: `GET /metrics/`
- OpenTelemetry traces/logs exported via OTLP HTTP
- Structured logs via `log/slog`
