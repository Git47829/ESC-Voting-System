# ESC Voting System

A distributed Eurovision-style voting platform with a React frontend, an Express BFF, Go/Python/Rust backend services, and full observability behind a Traefik HTTPS reverse proxy.

## 🚦 Deployment Modes

- **Local development/integration:** `docker-compose.yaml` (this README)
- **Go-live on K3s:** manifests + ops runbook in [`k8s/README.md`](k8s/README.md)

### Container publishing for K3s

K3s uses private GHCR images (`ghcr.io/git47829/esc-voting-*`) published by CI. Keep cluster pull credentials (`read:packages`) configured as described in `k8s/README.md`.

## 🏗️ System Architecture (current compose stack)

```text
Internet
  |
  v
Traefik (80/443, TLS with local default cert)
  |-- /, /api/* ------------------> ESC Frontend server (Express, :3001)
  |                                  \-> serves React SPA assets
  |
  |-- /crud-api/* -----------------> CRUD DB API (Go, :8000 + gRPC :50051)
  |                                    |-> MySQL (:3306)
  |                                    |-> EuroMail (/send) for auth code emails
  |                                    |-> optional Redis/RabbitMQ integration
  |
  |-- /esc-converter/* ------------> PublicVoteConverter (Go, :8090)
  |                                    \-> gRPC GetSongsWithVotes from CRUD API
  |
  |-- /eurostats/* ----------------> EuroStats (FastAPI, :8880)
  |                                    \-> consumes vote events from RabbitMQ
  |                                    \-> optional Redis persistence/pubsub
  |
  |-- /grafana* /prometheus* /tempo/* /loki/* --> Observability services
```

## ✨ Key Behavior

- **Frontend runtime:** `esc-frontend/` is the active UI (`docker-compose` builds this), including React client + Express `/api` BFF.
- **Auth flow:** two-step login (`/auth/login` + `/auth/verify`) for admin/jury, with legacy token auth still supported.
- **Contest mode:** admin can start/advance a contest run; `/now` auto-refreshes when song index changes.
- **Voting controls:** admin open/close toggle, per-session vote budget for public voting, jury single-use point values (1–8,10,12).
- **Stats streaming:** EuroStats exposes vote/stat streams over WebSocket (`/ws/votes`, `/ws/stats`).
- **Observability:** OTLP traces/logs/metrics via collector into Tempo/Loki/Prometheus with Grafana dashboards.

## 🛠️ Components

| Service | Technology | Internal Port(s) | README |
|---|---|---|---|
| Traefik | `traefik:v3.1` | 80, 443 | [backend/Traefik/README.md](backend/Traefik/README.md) |
| ESC Frontend (active) | React 18 + Vite + Express (TypeScript) | 3001 | [esc-frontend/README.md](esc-frontend/README.md) |
| CRUD DB API | Go, net/http, gRPC, Prometheus, OTel | 8000, 50051 | [backend/CRUD-DB-API/README.md](backend/CRUD-DB-API/README.md) |
| PublicVoteConverter | Go HTTP + gRPC client | 8090 | [backend/PublicVoteConverter/README.md](backend/PublicVoteConverter/README.md) |
| EuroStats | FastAPI + WebSockets + RabbitMQ consumer | 8880 | [backend/EuroStats/README.md](backend/EuroStats/README.md) |
| EuroMail | Rust (Axum), email sender + RabbitMQ consumer | 3000 | _(no component README yet)_ |
| MySQL | MySQL 8.0 | 3306 | [backend/DB/README.md](backend/DB/README.md) |
| Observability | OTel Collector, Prometheus, Grafana, Loki, Tempo | 4317–4318, 9090, 3000, 3100, 3200 | [backend/Observability/README.md](backend/Observability/README.md) |

### Legacy frontend note

The old Flask frontend still exists in [`frontend/`](frontend/) and is documented in [`frontend/README.md`](frontend/README.md), but it is **not** the frontend service used by `docker-compose.yaml`.

## 🌐 Service URLs (via Traefik)

Replace `<host>` with your Docker host/IP.

| Service | URL |
|---|---|
| App (React UI) | `https://<host>/` |
| App API (BFF) | `https://<host>/api/...` |
| CRUD API | `https://<host>/crud-api/` |
| PublicVoteConverter | `https://<host>/esc-converter/` |
| EuroStats | `https://<host>/eurostats/` |
| Grafana | `https://<host>/grafana/` |
| Prometheus | `https://<host>/prometheus/` |
| Tempo | `https://<host>/tempo/` |
| Loki | `https://<host>/loki/` |

> Plain HTTP redirects to HTTPS automatically.

## 🚀 Getting Started (docker compose)

### Prerequisites

- Docker + Docker Compose v2
- `.env` file in project root (see `.env.example`)

### Typical `.env` values

```env
# Database
MYSQL_ROOT_PASSWORD=secretroot

# Auth + app secrets
SESSION_SECRET=change-me
adminMail=admin@example.com
adminPassword=<bcrypt hash>
juryMail1=jury1@example.com
juryPassword1=<bcrypt hash>
juryMail2=jury2@example.com
juryPassword2=<bcrypt hash>
juryMail3=jury3@example.com
juryPassword3=<bcrypt hash>

# Optional integrations (degraded behavior if unavailable)
REDIS_URL=redis://redis:6379
RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672/
EuroMailURL=http://euromail:3000/send
RESEND_API_KEY=<provider-key>
```

### Run

```bash
docker compose up -d --build
```

Then open `https://<host>/`.

### Stop

```bash
docker compose down
```

Remove data volumes:

```bash
docker compose down -v
```

## 📡 API / Service Overview

### Frontend BFF (`/api` on esc-frontend)

Key routes exposed by Express server:

- Auth/session: `POST /api/auth/login`, `POST /api/auth/verify`, `POST /api/logout`, `GET /api/session`
- Public data/actions: `GET /api/songs`, `GET /api/votes`, `GET /api/countries`, `GET /api/contest/current`, `POST /api/vote`
- Jury/admin actions: `POST /api/jury/vote`, `POST /api/admin/open`, `POST /api/admin/close`, `POST /api/admin/startContest`, `POST /api/admin/advanceContest`, etc.
- Aggregated views: `GET /api/results` (joins converter + CRUD votes), `GET /api/stats` (EuroStats)

### CRUD API (behind `/crud-api` externally)

Public endpoints:

- `GET /health`
- `GET /votes/`, `GET /countries/`, `GET /countryByName/{NAME}`
- `GET /songs/`, `GET /songByID/{ID}`
- `POST /vote/`
- `GET /contest/current`
- `GET /metrics/`

Auth endpoints:

- `POST /auth/login`
- `POST /auth/verify`
- Legacy: `GET /auth/requestToken`, `GET /auth/verifyToken/{token}`

Protected endpoints use `Authorization: Bearer <token-or-password>` and `X-Email` headers:

- Admin: `/admin/open`, `/admin/close`, `/admin/deleteVotes/`, `/admin/addCountry/`, `/admin/addSong/`, `/admin/addArtist/`, `/admin/addInterpret/`, `/admin/startContest`, `/admin/advanceContest`, `/admin/authenticate`
- Jury: `/jury/vote/`, `/jury/authenticate`

### EuroStats (behind `/eurostats` externally)

- `GET /health`
- `GET /votes/subscribe`
- `WS /ws/votes`
- `WS /ws/stats`

### PublicVoteConverter (behind `/esc-converter` externally)

- `GET /health`
- `GET /metrics`
- `GET /api/esc-points`

## 🔭 Integration Notes

- `esc-frontend` BFF orchestrates CRUD API, PublicVoteConverter, and EuroStats.
- CRUD API is source-of-truth for songs/countries/votes and also exposes gRPC for converter consumption.
- EuroMail handles email token delivery for auth verification; API degrades gracefully if optional infra (Redis/RabbitMQ) is unavailable.
- EuroStats currently ingests votes from RabbitMQ and can persist/broadcast state with Redis when configured.
