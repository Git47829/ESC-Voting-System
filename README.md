# ESC Voting System

A distributed voting system for the Eurovision Song Contest, featuring a modern frontend, microservices architecture, and comprehensive observability stack with production-ready features including rate limiting, authentication, and distributed tracing.

## 🏗️ System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Frontend Layer                          │
│                      Flask + HTML + CSS                         │
│                  Port 5000 (User Interface)                     │
└────────────────────────────────┬────────────────────────────────┘
                                 │ REST API
                                 │
┌────────────────────────────────▼────────────────────────────────┐
│                       Backend Services                          │
├─────────────────────────────────────────────────────────────────┤
│  ┌────────────────────────────┐  ┌──────────────────────────┐   │
│  │   CRUD API (Go)            │  │  EuroStats (Python)      │   │
│  │   • Port 8000              │  │  • Real-time Stats       │   │
│  │   • Rate Limiting          │◄─┤  • gRPC Consumer         │   │
<<<<<<< HEAD
│  │   • JWT/Token Auth         │  │  • Vote Aggregation      │   │
│  │   • OpenTelemetry Traces   │  │  • NumPy Analytics       │   │
│  └───────────┬────────────────┘  └──────────────────────────┘   │
│              │                                                  │ 
=======
│  │   • Token Auth             │  │  • Vote Aggregation      │   │
│  │   • OpenTelemetry Traces   │  │  • NumPy Analytics       │   │
│  └───────────┬────────────────┘  └──────────────────────────┘   │
│              │                                                  │
>>>>>>> aeb3bd6ddc5e6123af4c9a8ec41cd8a31a4350dc
│  ┌───────────▼────────────────┐                                 │
│  │   MySQL Database           │                                 │
│  │   • Port 3306              │                                 │
│  │   • ESC Data Model         │                                 │
│  │   • Vote Persistence       │                                 │
│  └────────────────────────────┘                                 │
└─────────────────────────────────────────────────────────────────┘
                                 │
┌────────────────────────────────▼────────────────────────────────┐
│                    Observability Stack                          │
├─────────────────────────────────────────────────────────────────┤
│  ┌──────────────────┐  ┌──────────────┐  ┌──────────────────┐   │
│  │ OTel Collector   │  │ Prometheus   │  │ Grafana          │   │
│  │ Port 4317/4318   │─►│ Port 9090    │─►│ Port 3000        │   │
│  └──────────────────┘  └──────────────┘  └──────────────────┘   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ Loki (Log Aggregation)                                   │   │
│  │ Port 3100                                                │   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

## ✨ Key Features

### 🔐 Security & Access Control
- **Multi-tier Authentication**: Admin, Jury, and Public user roles
- **bcrypt Password Hashing**: Secure credential storage
- **Phone Number Verification**: One vote per phone number (hashed)
- **Cookie-based Vote Tracking**: Prevents duplicate voting
- **Rate Limiting**: Configurable per-endpoint request throttling

### 📊 Observability & Monitoring
- **Distributed Tracing**: Full request tracing with OpenTelemetry
- **Metrics Collection**: Prometheus metrics for request duration, size, and counts
- **Structured Logging**: JSON-formatted logs with context propagation
- **Grafana Dashboards**: Visual monitoring and alerting
- **Log Aggregation**: Centralized logging with Loki

### 🎯 Voting System Features
- **Public Voting**: Weighted voting with phone number verification
- **Jury Voting**: Professional jury votes with higher weights
- **Real-time Statistics**: Live vote aggregation via gRPC
- **Vote Status Control**: Admin controls to open/close voting periods
- **Multi-country Support**: Vote for any country except your own

## 🛠️ Components

### Frontend
- **Technology**: Flask, HTML, CSS
- **Location**: `/frontend`
- **Purpose**: User-facing interface for voting and results
- **Features**: 
  - Responsive web design
  - Real-time vote updates
  - Country and song browsing

### Backend Services

#### CRUD API (Go)
- **Technology**: Go 1.21+, MySQL Driver, bcrypt
- **Location**: `/backend/CRUD-DB-API`
- **Port**: 8000
- **Design Patterns**: 
  - Middleware/Chain of Responsibility
  - Singleton (DB, Logger, Tracer)
  - Decorator (Response Writer)
  - Factory (Rate Limiter)
  - Repository (Data Access)
  - Retry Pattern (DB Connection)

**API Endpoints**:
```
Public Endpoints:
  GET  /health              - Health check
  GET  /votes/              - Get all votes and rankings
  GET  /countries/          - List all countries
  GET  /countryByName/{ID}  - Get country by ID
  GET  /songs/              - List all songs with details
  GET  /songByID/{ID}       - Get song by ID
  POST /vote/               - Cast a public vote
  GET  /metrics/            - Prometheus metrics

Admin Endpoints (Token Required):
  POST   /admin/open/         - Open voting
  POST   /admin/close         - Close voting
  DELETE /admin/deleteVotes/  - Reset all votes
  POST   /admin/addCountry/   - Add new country
  POST   /admin/addSong/      - Add new song
  POST   /admin/addArtist/    - Add new artist
  POST   /admin/addInterpret/ - Add new composer

Jury Endpoints (Token Required):
  POST /jury/vote/  - Cast jury vote with points
```

**Rate Limits**:
- Health: 100 req/s
- Public GET: 10 req/s
- Public Vote: 1 req/s (burst: 1)
- Jury Vote: 5 req/s
- Admin: 2-5 req/s
- Metrics: Unlimited

#### EuroStats Microservice
- **Technology**: Python 3.11+, gRPC, NumPy
- **Location**: `/backend/EuroStats`
- **Purpose**: Real-time statistical analysis and vote aggregation
- **Features**:
  - gRPC consumer for vote events
  - Statistical computations
  - Vote trend analysis

#### MySQL Database
- **Technology**: MySQL 8.0+
- **Location**: `/backend/DB`
- **Port**: 3306
- **Schema**:
  - `Land` (Countries) - ID, Name, Pot
  - `Song` - Songs with public/jury/total points
  - `Kuenstler` (Artists) - Solo, duo, or group performers
  - `Komponist` (Composers) - Song composers
  - `Song_Komponist` - Many-to-many song-composer relationship
  - `Voting_Status` - Global voting state control
  - `Phone_Nums` - Hashed phone number registry

### Observability Stack

#### OpenTelemetry Collector
- **Version**: 0.94.0
- **Ports**: 4317 (gRPC), 4318 (HTTP), 9464 (Prometheus)
- **Purpose**: Centralized telemetry collection and export

#### Prometheus
- **Version**: 2.51.1
- **Port**: 9090
- **Purpose**: Metrics storage and querying

#### Grafana
- **Version**: 10.4.2
- **Port**: 3000
- **Purpose**: Visualization dashboards and alerting

#### Loki
- **Version**: 2.9.6
- **Port**: 3100
- **Purpose**: Log aggregation and querying
