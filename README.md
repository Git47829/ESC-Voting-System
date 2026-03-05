# ESC Voting System

A distributed voting system for the Eurovision Song Contest, featuring a modern web frontend, a Go REST + gRPC backend, real-time vote streaming, and a full observability stack with tracing, metrics, and log aggregation.

## 🏗️ System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Frontend Layer                          │
│                Flask + Jinja2 + Tailwind CSS                    │
│               Port 5000 — served by Gunicorn                    │
└────────────────────────────────┬────────────────────────────────┘
                                 │ REST (HTTP)
                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                       Backend Services                          │
├─────────────────────────────────────────────────────────────────┤
│  ┌──────────────────────────────────┐                           │
│  │   CRUD DB API (Go)               │                           │
│  │   • Port 8000  — REST API        │                           │
│  │   • Port 50051 — gRPC streaming  │                           │
│  │   • Rate limiting (token bucket) │                           │
│  │   • bcrypt token auth            │                           │
│  │   • OpenTelemetry tracing        │                           │
│  │   • Prometheus metrics           │                           │
│  └──────────┬───────────────────────┘                           │
│             │ SQL                    │ gRPC StreamVotes         │
│  ┌──────────▼────────────┐  ┌───────▼────────────────────────┐  │
│  │   MySQL 8.0           │  │   EuroStats (Python/FastAPI)   │  │
│  │   Port 3306           │  │   Port 8880 (8881 on host)     │  │
│  │   ESC data model      │  │   REST + WebSocket endpoints   │  │
│  │   Vote persistence    │  │   Real-time vote consumer      │  │
│  └───────────────────────┘  └────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                                 │ OTLP
                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Observability Stack                          │
├─────────────────────────────────────────────────────────────────┤
│  ┌──────────────────┐  ┌────────────┐  ┌─────────┐  ┌───────┐   │
│  │ OTel Collector   │  │ Prometheus │  │  Tempo  │  │ Loki  │   │
│  │ 4317/4318/9464   │─►│   9090     │  │  3200   │  │ 3100  │   │
│  └──────────────────┘  └─────┬──────┘  └────┬────┘  └───┬───┘   │
│                              └──────────────┼────────────┘      │
│                                             ▼                   │
│                                    ┌─────────────┐              │
│                                    │   Grafana   │              │
│                                    │    3000     │              │
│                                    └─────────────┘              │
└─────────────────────────────────────────────────────────────────┘
```

## ✨ Key Features

### 🔐 Security & Access Control
- **Multi-tier Authentication** — Admin, Jury, and Public user roles
- **bcrypt Password Hashing** — Secure credential storage and token validation
- **Phone Number Verification** — One vote per phone number (bcrypt-hashed in DB)
- **Cookie-based Vote Tracking** — Prevents duplicate voting from the same browser
- **Token Replay Prevention** — Used tokens are tracked in-memory to block reuse
- **Rate Limiting** — Per-IP token-bucket rate limiting on every endpoint

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

| Service | Technology | Port | README |
|---|---|---|---|
| Frontend | Python 3.12, Flask 3.x, Gunicorn, Tailwind CSS | 5000 | [frontend/README.md](frontend/README.md) |
| CRUD DB API | Go 1.24, net/http, gRPC, Prometheus, OTel | 8000, 50051 | [backend/CRUD-DB-API/README.md](backend/CRUD-DB-API/README.md) |
| EuroStats | Python 3.11, FastAPI, gRPC, OTel | 8881 | [backend/EuroStats/README.md](backend/EuroStats/README.md) |
| MySQL | MySQL 8.0 | 3306 | [backend/DB/README.md](backend/DB/README.md) |
| Observability | OTel Collector, Prometheus, Grafana, Loki, Tempo | 4317–4318, 9090, 3000, 3100, 3200 | [backend/Observability/README.md](backend/Observability/README.md) |

### Service URLs

| Service | URL |
|---|---|
| Frontend (Voting UI) | http://localhost:5000 |
| CRUD DB API | http://localhost:8000 |
| EuroStats | http://localhost:8881 |
| Grafana | http://localhost:3000 |
| Prometheus | http://localhost:9090 |
| Loki | http://localhost:3100 |
| Tempo | http://localhost:3200 |

### Default Grafana Login

```
Username: admin
Password: admin
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

### Public Endpoints (CRUD API — port 8000)

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
| `POST` | `/admin/open/` | Admin token | Open voting |
| `POST` | `/admin/close` | Admin token | Close voting |
| `DELETE` | `/admin/deleteVotes/` | Admin token | Reset all votes |
| `POST` | `/admin/addCountry/` | Admin token | Add a country |
| `POST` | `/admin/addSong/` | Admin token | Add a song |
| `POST` | `/admin/addArtist/` | Admin token | Add an artist |
| `POST` | `/admin/addInterpret/` | Admin token | Add a composer |
| `POST` | `/jury/vote/` | Jury token | Cast a jury vote |

### EuroStats Endpoints (port 8881)

| Method | Path | Description |
|---|---|---|
| `GET` | `/health` | Health check |
| `GET` | `/votes/subscribe` | Poll current vote snapshot (up to 100 entries) |
| `WS` | `/ws/votes` | WebSocket — real-time vote event stream |

## 🐳 Docker Networks

| Network | Connected Services |
|---|---|
| `backend` | `db`, `api`, `eurostats` |
| `frontend` | `api`, `frontend` |
| `observability` | `api`, `frontend`, `eurostats`, `otel-collector`, `prometheus`, `grafana`, `loki`, `tempo` |

## 🔭 Observability Pipeline

```
CRUD API  ──OTLP/HTTP──►
Frontend  ──OTLP/HTTP──►  OTel Collector ──► Tempo      (traces)
EuroStats ──OTLP/gRPC──►                 ──► Prometheus  (metrics)
                                         ──► Loki        (logs)
                                                │
                                                └──► Grafana (unified view)
```
