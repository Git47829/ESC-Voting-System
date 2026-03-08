# ESC Voting System

A distributed voting system for the Eurovision Song Contest, featuring a modern web frontend, a Go REST + gRPC backend, real-time vote streaming, a full observability stack with tracing, metrics, and log aggregation — all served through a Caddy HTTPS reverse proxy.

## 🏗️ System Architecture

```
                         ┌─────────────────────────────┐
                         │          Caddy              │
                         │   HTTPS reverse proxy       │
                         │   Ports 80 (→443) + 443     │
                         │   tls internal (local CA)   │
                         └──────────────┬──────────────┘
                                        │ routes by subpath
          ┌─────────────────────────────┼──────────────────────────────┐
          │ /                           │ /crud-api/   /eurostats/     │
          ▼                             ▼              ▼               │
┌─────────────────────┐   ┌────────────────────┐  ┌──────────────┐   │
│     Frontend        │   │   CRUD DB API (Go) │  │  EuroStats   │   │
│  Flask + Gunicorn   │   │  REST + gRPC       │  │  FastAPI     │   │
│  Port 5000          │   │  Port 8000 / 50051 │  │  Port 8880   │   │
└──────────┬──────────┘   └──────────┬─────────┘  └──────┬───────┘   │
           │                         │ SQL               │           │
           │                         ▼                   │           │
           │               ┌──────────────────┐          │           │
           │               │   MySQL 8.0      │          │           │
           │               │   Port 3306      │          │           │
           │               └──────────────────┘          │           │
           │                                             │           │
           │  /grafana  /prometheus  /tempo  /loki       │           │
           └─────────────────────────────────────────────┘           │
                                        │                            │
                                        ▼                            │
┌───────────────────────────────────────────────────────────────────┐│
│                        Observability Stack                        ││
├───────────────────────────────────────────────────────────────────┤│
│  ┌──────────────────┐  ┌────────────┐  ┌─────────┐  ┌──────────┐  ││
│  │ OTel Collector   │  │ Prometheus │  │  Tempo  │  │  Loki    │  ││
│  │ 4317 / 4318      │─►│   9090     │  │  3200   │  │  3100    │  ││
│  └──────────────────┘  └─────┬──────┘  └────┬────┘  └────┬─────┘  ││
│                              └──────────────┼─────────────┘        ││
│                                             ▼                      ││
│                                    ┌─────────────┐                 ││
│                                    │   Grafana   │                 ││
│                                    │    3000     │                 ││
│                                    └─────────────┘                 ││
└───────────────────────────────────────────────────────────────────┘│
└────────────────────────────────────────────────────────────────────┘
```

## ✨ Key Features

### 🔐 Security & Access Control
- **HTTPS everywhere** — Caddy is the sole external entry point; all services are unreachable directly from outside Docker. TLS is provided by Caddy's built-in local CA (`tls internal`).
- **Multi-tier Authentication** — Admin, Jury, and Public user roles
- **bcrypt Password Hashing** — Secure credential storage and token validation
- **Phone Number Verification** — One vote per phone number (bcrypt-hashed in DB)
- **Cookie-based Vote Tracking** — Prevents duplicate voting from the same browser
- **Token Replay Prevention** — Used tokens are tracked in-memory to block reuse
- **Rate Limiting** — Per-IP token-bucket rate limiting on every endpoint
- **Non-root containers** — CRUD API runs in a minimal distroless image; EuroStats runs as a dedicated `appuser` (UID 1001)

### 📡 Real-time Vote Streaming
- **gRPC VoteService** — The CRUD API exposes a `StreamVotes` server-side streaming RPC on port `50051`
- **Historical + Live** — New subscribers optionally receive all current vote totals before live updates
- **EuroStats Consumer** — FastAPI microservice that connects as a gRPC client and re-exposes votes via REST and WebSocket

### 📊 Observability & Monitoring
- **Distributed Tracing** — Full request tracing with OpenTelemetry, stored in Tempo
- **Metrics Collection** — Prometheus metrics for request duration, size, counts, backend call latency, vote counts, and active sessions
- **Structured Logging** — JSON-formatted logs from all services, aggregated in Loki
- **Grafana Dashboards** — Auto-provisioned data sources (Prometheus, Loki, Tempo) with trace-to-log correlation
- **Gunicorn Multiprocess Metrics** — Prometheus multiprocess mode ensures correct metric aggregation across all frontend workers

### 🗳️ Voting System
- **Public Voting** — Phone number verification, duplicate prevention via cookie + hashed phone registry
- **Jury Voting** — Authenticated jury votes with configurable point values (Eurovision-style 1–8, 10, 12)
- **Vote Status Control** — Admin open/close toggle; all endpoints respect the global voting state
- **Multi-country Support** — Voters cannot vote for their own country
- **Real-time Results** — Live results page polling every 10 s; WebSocket stream available via EuroStats

## 🛠️ Components

| Service | Technology | Internal Port | README |
|---|---|---|---|
| Caddy | `caddy:2.8-alpine` | 80, 443 | — |
| Frontend | Python 3.12, Flask 3.x, Gunicorn, Tailwind CSS | 5000 | [frontend/README.md](frontend/README.md) |
| CRUD DB API | Go 1.24, net/http, gRPC, Prometheus, OTel | 8000, 50051 | [backend/CRUD-DB-API/README.md](backend/CRUD-DB-API/README.md) |
| EuroStats | Python 3.11, FastAPI, gRPC, OTel | 8880 | [backend/EuroStats/README.md](backend/EuroStats/README.md) |
| MySQL | MySQL 8.0 | 3306 | — |
| Observability | OTel Collector, Prometheus, Grafana, Loki, Tempo | 4317–4318, 9090, 3000, 3100, 3200 | [backend/Observability/README.md](backend/Observability/README.md) |

## 🌐 Service URLs

All services are accessed through Caddy on a single HTTPS host. Replace `<host>` with the IP address or hostname of the machine running Docker.

| Service | URL |
|---|---|
| Frontend (Voting UI) | `https://<host>/` |
| CRUD DB API (direct access) | `https://<host>/crud-api/` |
| EuroStats | `https://<host>/eurostats/` |
| Grafana | `https://<host>/grafana/` |
| Prometheus | `https://<host>/prometheus/` |
| Tempo | `https://<host>/tempo/` |
| Loki | `https://<host>/loki/` |

> **Note:** Plain HTTP (`http://<host>`) redirects automatically to HTTPS.

### HTTPS / Certificate Trust

Caddy uses its built-in local CA to issue a self-signed certificate. Your browser will show a security warning until you trust the CA cert. To install it:

```bash
# Copy the CA cert out of the running Caddy container
docker compose cp caddy:/data/caddy/pki/authorities/local/root.crt ./caddy-local-ca.crt

# macOS
sudo security add-trusted-cert -d -r trustRoot \
  -k /Library/Keychains/System.keychain ./caddy-local-ca.crt

# Linux (Debian/Ubuntu)
sudo cp caddy-local-ca.crt /usr/local/share/ca-certificates/
sudo update-ca-certificates

# Windows: double-click caddy-local-ca.crt →
#   Install Certificate → Local Machine →
#   Trusted Root Certification Authorities
```

Firefox maintains its own trust store — go to **Settings → Privacy & Security → Certificates → View Certificates → Authorities → Import**.

### Default Grafana Login

```
Username: admin
Password: admin
```

## 🚀 Getting Started

### Prerequisites

- Docker + Docker Compose v2
- A `.env` file in the project root (see below)

### Environment Variables (`.env`)

```env
# MySQL
MYSQL_ROOT_PASSWORD=secretroot

# Admin / Jury tokens — change these before deploying
adminPassword=change-me-admin
juryPassword1=change-me-jury-1
juryPassword2=change-me-jury-2
juryPassword3=change-me-jury-3

# Flask
FLASK_SECRET_KEY=change-me-flask-secret
```

### Running

```bash
docker compose up -d --build
```

Then open `https://<host>/` in your browser (accept or trust the certificate on first visit).

### Stopping

```bash
docker compose down
```

To also remove all persistent data volumes:

```bash
docker compose down -v
```

## 🗄️ Database Schema

| Table | Description |
|---|---|
| `Land` | Countries — ISO 3-letter ID, name, pot assignment |
| `Kuenstler` | Artists — solo, duo, or group; linked to a country |
| `Komponist` | Composers — first and last name |
| `Song` | Songs — linked to country and artist; stores public, jury, and computed total points |
| `Song_Komponist` | Many-to-many relationship between songs and composers |
| `Voting_Status` | Single-row global flag controlling whether voting is open |
| `Phone_Nums` | Registry of bcrypt-hashed phone numbers that have already voted |

## 📡 API Overview

All API paths below are relative to the CRUD API's internal address (`db-crud-api:8000`). When accessed externally via Caddy, prefix them with `/crud-api/` — e.g. `https://<host>/crud-api/votes/`.

### Public Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/health` | Health check |
| `GET` | `/votes/` | All songs ranked by total points |
| `GET` | `/countries/` | List all countries |
| `GET` | `/songs/` | Full song list with artist, country, composers, and voting status |
| `POST` | `/vote/` | Cast a public vote |
| `GET` | `/metrics/` | Prometheus metrics |

### Protected Endpoints

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/admin/authenticate` | Admin token | Validate an admin token — `202` on success, `403` on failure |
| `POST` | `/admin/open/` | Admin token | Open voting |
| `POST` | `/admin/close` | Admin token | Close voting |
| `DELETE` | `/admin/deleteVotes/` | Admin token | Reset all votes |
| `POST` | `/admin/addCountry/` | Admin token | Add a country |
| `POST` | `/admin/addSong/` | Admin token | Add a song |
| `POST` | `/admin/addArtist/` | Admin token | Add an artist |
| `POST` | `/admin/addInterpret/` | Admin token | Add a composer |
| `GET` | `/jury/authenticate` | Jury token | Validate a jury token — `202` on success, `403` on failure |
| `POST` | `/jury/vote/` | Jury token | Cast a jury vote |

### EuroStats Endpoints

Accessible externally at `https://<host>/eurostats/`.

| Method | Path | Description |
|---|---|---|
| `GET` | `/health` | Health check |
| `GET` | `/votes/subscribe` | Poll current vote snapshot (up to 100 entries) |
| `WS` | `/ws/votes` | WebSocket — real-time vote event stream |

## 🐳 Docker Networks

Services communicate over isolated Docker networks. No service other than Caddy has ports bound on the host.

| Network | Connected Services |
|---|---|
| `backend` | `db`, `api`, `eurostats` |
| `frontend` | `api`, `frontend`, `caddy` |
| `observability` | `api`, `frontend`, `eurostats`, `otel-collector`, `prometheus`, `grafana`, `loki`, `tempo`, `caddy` |

## 🔭 Observability Pipeline

```
CRUD API  ──OTLP/HTTP──►
Frontend  ──OTLP/HTTP──►  OTel Collector ──► Tempo      (traces)
EuroStats ──OTLP/gRPC──►                 ──► Prometheus  (metrics)
                                         ──► Loki        (logs)
                                                │
                                                └──► Grafana (unified view)
                                                     https://<host>/grafana/