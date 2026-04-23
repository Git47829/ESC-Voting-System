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
          ┌─────────────────────────────┼─────────────────────────────┐
          │ /                           │ /crud-api/   /eurostats/    │
          ▼                             ▼              ▼              │
┌─────────────────────┐   ┌────────────────────┐  ┌──────────────┐    │
│     Frontend        │   │   CRUD DB API (Go) │  │  EuroStats   │    │
│  Node.js/Express    │   │  REST + gRPC       │  │  FastAPI     │    │
│  Port 3001          │   │  Port 8000 / 50051 │  │  Port 8880   │    │
└──────────┬──────────┘   └──────────┬─────────┘  └──────┬───────┘    │
           │                         │ SQL    ▲  gRPC     │           │
           │                         ▼        │            │          │
           │               ┌──────────────────┐            │          │
           │               │   MySQL 8.0      │ ┌──────────────────-┐ │
           │               │   Port 3306      │ │PublicVoteConverter│ │
           │               └──────────────────┘ │  Go, Port 8090    │ │
           │                                    └──────────────────-┘ │
           │                                             │            │
           │  /grafana  /prometheus  /tempo  /loki       │            │
           └─────────────────────────────────────────────┘            │
                                        │                             │
                                        ▼                             │
┌──────────────────────────────────────────────────────────────────┐  │
│                        Observability Stack                       │  │
├──────────────────────────────────────────────────────────────────┤  │
│  ┌──────────────────┐  ┌────────────┐  ┌─────────┐  ┌─────────┐  │  │
│  │ OTel Collector   │  │ Prometheus │  │  Tempo  │  │  Loki   │  │  │
│  │ 4317 / 4318      │─►│   9090     │  │  3200   │  │  3100   │  │  │
│  └──────────────────┘  └─────┬──────┘  └────┬────┘  └────┬────┘  │  │
│                              └──────────────┼─────────────┘      │  │
│                                             ▼                    │  │
│                                    ┌─────────────┐               │  │
│                                    │   Grafana   │               │  │
│                                    │    3000     │               │  │
│                                    └─────────────┘               │  │
└──────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
```

## ✨ Key Features

### 🎤 Running Now — Live Contest Stage

- **Start Contest button** in the admin panel fetches all songs from the database, shuffles them into a random order using a Fisher-Yates shuffle, and persists the result as an active `Contest_Run` row.
- **Running Now page (`/now`)** shows the currently performing song with a full-width **YouTube embed**, song and artist details, a live score counter, and a contest progress bar.
- **Inline voting panel** — viewers cast points directly below the video without leaving the page. Includes a point stepper (+/−), phone number input, and country selector.
- **Auto-advance** — the admin clicks "Next Song" to move to the next entry; all viewers on `/now` detect the change within 5 seconds and reload automatically.
- **Contest-end banner** — when the last song finishes, a banner appears linking to the results page.
- **YouTube URL normalization** — any YouTube URL format (watch, share, short, embed) is automatically converted to the correct embed format before storage and rendering.

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
| Caddy | `caddy:2.8-alpine` | 80, 443 | [backend/Caddy/README.md](backend/Caddy/README.md) |
| Frontend | Node.js 22, Express.js, Tailwind CSS | 3001 | [esc-frontend/README.md](esc-frontend/README.md) |
| CRUD DB API | Go 1.24, net/http, gRPC, Prometheus, OTel | 8000, 50051 | [backend/CRUD-DB-API/README.md](backend/CRUD-DB-API/README.md) |
| PublicVoteConverter | Go 1.24, net/http, gRPC client, Prometheus, OTel | 8090 | [backend/PublicVoteConverter/README.md](backend/PublicVoteConverter/README.md) |
| EuroStats | Python 3.11, FastAPI, gRPC, OTel | 8880 | [backend/EuroStats/README.md](backend/EuroStats/README.md) |
| MySQL | MySQL 8.0 | 3306 | [backend/DB/README.md](backend/DB/README.md) |
| EuroMail | Node.js, Express, Resend API | 3000 (internal) | - |
| Observability | OTel Collector, Prometheus, Grafana, Loki, Tempo | 4317–4318, 9090, 3000, 3100, 3200 | [backend/Observability/README.md](backend/Observability/README.md) |

## 🌐 Service URLs

All services are accessed through Caddy on a single HTTPS host. Replace `<host>` with the IP address or hostname of the machine running Docker.

| Service | URL |
|---|---|
| Frontend — Vote | `https://<host>/` |
| Frontend — Running Now | `https://<host>/now` |
| Frontend — Live Results | `https://<host>/results` |
| Frontend — Admin Dashboard | `https://<host>/admin` |
| Frontend — Jury Voting | `https://<host>/jury` |
| CRUD DB API (direct access) | `https://<host>/crud-api/` |
| PublicVoteConverter (direct access) | `https://<host>/esc-converter/` |
| EuroStats | `https://<host>/eurostats/` |
| Grafana | `https://<host>/grafana/` |
| Prometheus | `https://<host>/prometheus/` |
| Tempo | `https://<host>/tempo/` |
| Loki | `https://<host>/loki/` |

> **Note:** Plain HTTP (`http://<host>`) redirects automatically to HTTPS.

### Public Internet Access via Cloudflare Tunnel

To expose the ESC Voting System to the internet via `escvoting.dev`, use Cloudflare Tunnel:

**Setup:**
1. Create a Cloudflare account and add your domain to Cloudflare DNS
2. In the Cloudflare dashboard, go to **Zero Trust → Tunnels** and create a new tunnel named `escvoting`
3. Copy the tunnel token and add it to your `.env`:
   ```env
   CLOUDFLARE_TUNNEL_NAME=escvoting
   CLOUDFLARE_SECRET=<your-token-from-cloudflare-dashboard>
   ```
4. Start Docker Compose — the `cloudflared` service will automatically connect to your tunnel

**Public Routes (via `escvoting.dev`):**
- Frontend: `https://escvoting.dev/`
- Grafana: `https://escvoting.dev/grafana/`

**Security:** Only the frontend and Grafana are exposed to the internet. All other services (database, APIs, Prometheus, Tempo, Loki) remain internal and are unreachable from outside the Docker network.

**TLS:** Cloudflare manages the public HTTPS certificate at the edge. Internally, services communicate via Caddy's self-signed certificates.

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

# Frontend
SESSION_SECRET=change-me-session-secret
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
| `Land` | Countries — ISO 2-letter ID (alpha-2), name, pot assignment |
| `Kuenstler` | Artists — solo, duo, or group; linked to a country |
| `Komponist` | Composers — first and last name |
| `Song` | Songs — linked to country and artist; stores public, jury, and computed total points; optional YouTube embed URL |
| `Song_Komponist` | Many-to-many relationship between songs and composers |
| `Voting_Status` | Single-row global flag controlling whether voting is open |
| `Phone_Nums` | Registry of bcrypt-hashed phone numbers that have already voted |
| `Contest_Run` | Active contest state — shuffled song order (JSON), current index, start timestamp, and active flag |

## 📡 API Overview

All API paths below are relative to the CRUD API's internal address (`db-crud-api:8000`). When accessed externally via Caddy, prefix them with `/crud-api/` — e.g. `https://<host>/crud-api/votes/`.

### Public Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/health` | Health check |
| `GET` | `/votes/` | All songs ranked by total points |
| `GET` | `/countries/` | List all countries |
| `GET` | `/countryByName/{NAME}` | Get a country by name |
| `GET` | `/songs/` | Full song list with artist, country, composers, and voting status |
| `GET` | `/songByID/{ID}` | Single song detail by ID |
| `POST` | `/vote/` | Cast a public vote |
| `GET` | `/contest/current` | Current song in the active contest run with full details and progress |
| `GET` | `/metrics/` | Prometheus metrics |

### Authentication Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/auth/requestToken` | Request an authentication token |
| `GET` | `/auth/verifyToken/{token}` | Verify an authentication token |
| `POST` | `/auth/login` | Login with credentials |
| `POST` | `/auth/verify` | Verify authentication |

### Admin Endpoints

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/admin/authenticate` | Admin token | Validate an admin token — `202` on success, `403` on failure |
| `POST` | `/admin/open` | Admin token | Open voting |
| `POST` | `/admin/close` | Admin token | Close voting |
| `DELETE` | `/admin/deleteVotes/` | Admin token | Reset all votes |
| `POST` | `/admin/addCountry/` | Admin token | Add a country |
| `POST` | `/admin/addSong/` | Admin token | Add a song (accepts optional `YoutubeURL` parameter) |
| `POST` | `/admin/addArtist/` | Admin token | Add an artist |
| `POST` | `/admin/addInterpret/` | Admin token | Add a composer |
| `POST` | `/admin/startContest` | Admin token | Shuffle all songs and start a new contest run |
| `POST` | `/admin/advanceContest` | Admin token | Advance to the next song in the active contest |

### Jury Endpoints

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/jury/authenticate` | Jury token | Validate a jury token — `202` on success, `403` on failure |
| `POST` | `/jury/vote/` | Jury token | Cast a jury vote |

### EuroStats Endpoints

Accessible externally at `https://<host>/eurostats/`.

| Method | Path | Description |
|---|---|---|
| `GET` | `/health` | Health check |
| `GET` | `/votes/subscribe` | Poll current vote snapshot (up to 100 entries) |
| `WS` | `/ws/votes` | WebSocket — real-time vote event stream |
| `WS` | `/ws/stats` | WebSocket — real-time statistics update stream |

## 🐳 Docker Networks

Services communicate over isolated Docker networks. No service other than Caddy has ports bound on the host.

| Network | Connected Services |
|---|---|
| `backend` | `db`, `api`, `public-vote-converter`, `eurostats` |
| `frontend` | `api`, `public-vote-converter`, `frontend`, `caddy` |
| `mail` | `api`, `euromail` |
| `observability` | `api`, `public-vote-converter`, `frontend`, `eurostats`, `euromail`, `otel-collector`, `prometheus`, `grafana`, `loki`, `tempo`, `caddy` |

## 🔭 Observability Pipeline

```
CRUD API             ──OTLP/HTTP──►
Frontend             ──OTLP/HTTP──►  OTel Collector ──► Tempo      (traces)
EuroStats            ──OTLP/gRPC──►                 ──► Prometheus  (metrics)
PublicVoteConverter  ──OTLP/HTTP──►                 ──► Loki        (logs)
                                                           │
                                                           └──► Grafana (unified view)
                                                                https://<host>/grafana/
```
