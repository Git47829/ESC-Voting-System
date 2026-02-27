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
│  ┌────────────────────────────┐  ┌──────────────────────────┐  │
│  │   CRUD API (Go)            │  │  EuroStats (Python)      │  │
│  │   • Port 8000              │  │  • Real-time Stats       │  │
│  │   • Rate Limiting          │◄─┤  • gRPC Consumer         │  │
│  │   • JWT/Token Auth         │  │  • Vote Aggregation      │  │
│  │   • OpenTelemetry Traces   │  │  • NumPy Analytics       │  │
│  └───────────┬────────────────┘  └──────────────────────────┘  │
│              │                                                   │
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
│  ┌──────────────────┐  ┌──────────────┐  ┌──────────────────┐  │
│  │ OTel Collector   │  │ Prometheus   │  │ Grafana          │  │
│  │ Port 4317/4318   │─►│ Port 9090    │─►│ Port 3000        │  │
│  └──────────────────┘  └──────────────┘  └──────────────────┘  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ Loki (Log Aggregation)                                   │  │
│  │ Port 3100                                                │  │
│  └──────────────────────────────────────────────────────────┘  │
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

## 🚀 Getting Started

### Prerequisites
- Docker & Docker Compose
- Git

### Quick Start

1. **Clone the repository**:
   ```bash
   git clone <repository-url>
   cd ESC-Voting-System
   ```

2. **Create environment file**:
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

   Required environment variables:
   ```env
   # Database
   MYSQL_DATABASE=esc_voting
   MYSQL_USER=esc_user
   MYSQL_PASSWORD=esc_password
   MYSQL_ROOT_PASSWORD=secretroot
   
   # Authentication (bcrypt hashed passwords)
   adminPassword=<bcrypt_hash>
   juryPassword1=<bcrypt_hash>
   juryPassword2=<bcrypt_hash>
   juryPassword3=<bcrypt_hash>
   ```

3. **Start all services**:
   ```bash
   docker-compose up -d
   ```

4. **Access the services**:
   - Frontend: http://localhost:5000
   - CRUD API: http://localhost:8000
   - Prometheus: http://localhost:9090
   - Grafana: http://localhost:3000
   - Loki: http://localhost:3100

5. **Check service health**:
   ```bash
   curl http://localhost:8000/health
   ```

### Generate Admin Password Hash

To create a bcrypt hash for admin/jury passwords:
```bash
# Using Go
go run -exec "import golang.org/x/crypto/bcrypt; hash, _ := bcrypt.GenerateFromPassword([]byte(\"your_password\"), bcrypt.DefaultCost); fmt.Println(string(hash))"

# Or using Python
python3 -c "import bcrypt; print(bcrypt.hashpw(b'your_password', bcrypt.gensalt()).decode())"
```

## 📝 Development

### Project Structure
```
ESC-Voting-System/
├── backend/
│   ├── CRUD-DB-API/       # Go REST API
│   │   ├── src/
│   │   │   ├── main.go    # 1664 lines, 13 design patterns
│   │   │   ├── go.mod
│   │   │   └── go.sum
│   │   └── Dockerfile
│   ├── DB/                 # MySQL database
│   │   ├── db_scheme.sql   # Database schema
│   │   ├── seed_data.sql   # Initial data
│   │   └── Dockerfile
│   ├── EuroStats/          # Python analytics service
│   │   ├── src/
│   │   │   ├── main.py
│   │   │   ├── grpc_consumer.py
│   │   │   └── stats_store.py
│   │   └── proto/
│   └── Observability/      # Monitoring stack
│       ├── OTel/
│       ├── Prometheus/
│       └── grafana/
├── frontend/
│   └── src/
│       ├── main.py         # Flask application
│       ├── index.html
│       └── style.css
├── docker-compose.yaml
└── README.md
```

### Docker Services

All services are containerized and orchestrated via Docker Compose:

- **db**: MySQL database with health checks
- **api**: Go CRUD API with retry logic for DB connection
- **otel-collector**: OpenTelemetry collector for traces/metrics
- **prometheus**: Metrics storage and scraping
- **grafana**: Visualization with auto-provisioned dashboards
- **loki**: Log aggregation

### Networks

- `backend`: Database and API communication
- `frontend`: Frontend and API communication  
- `observability`: Monitoring stack communication

### Volumes

- `mysql_data`: Persistent database storage
- `grafana_data`: Grafana configuration and dashboards
- `loki_data`: Log storage

## 📖 Additional Documentation

Each component has its own README with detailed setup instructions:
- [Frontend Setup](./frontend/README.md)
- [CRUD API Setup](./backend/CRUD-DB-API/README.md)
- [EuroStats Setup](./backend/EuroStats/README.md)
- [Observability Setup](./backend/Observability/README.md)
- [Database Schema](./backend/DB/README.md)

## 🔧 Configuration

### Rate Limiting

Rate limits are configured per-endpoint in the CRUD API (`main.go`):
- Adjust `rateLimitConfigs` map for custom limits
- Format: `{RequestsPerSecond: float64, BurstSize: int}`

### Observability

- **Traces**: Exported to OTel Collector on port 4318 (HTTP)
- **Metrics**: Scraped by Prometheus from `/metrics` endpoint
- **Logs**: Structured JSON logs to stdout

## 🏆 Eurovision Voting Rules

1. **Public voting**: Vote once per phone number, weighted at 3 points
2. **Jury voting**: Professional juries can award points (1-12), weighted at 5x
3. **Country restriction**: Cannot vote for your own country
4. **Voting periods**: Controlled by admin (open/close)
5. **Final scores**: Combined public + jury points

## 📊 Monitoring & Metrics

Available Prometheus metrics:
- `http_request_duration_seconds` - Request latency histogram
- `http_request_size_bytes` - Request body size
- `http_response_size_bytes` - Response body size
- `http_requests_total` - Total request counter

All metrics include labels: `method`, `path`, `status`

## 🤝 Contributing

This is a university project demonstrating microservices architecture, observability patterns, and production-ready API design.

## 📄 License

[Add your license here]

## 🎓 Educational Purpose

This project demonstrates:
- Microservices architecture with Go and Python
- RESTful API design with comprehensive middleware
- Database design for complex relationships
- Rate limiting and authentication strategies
- Distributed tracing with OpenTelemetry
- Metrics collection with Prometheus
- Container orchestration with Docker Compose
- Multiple design patterns in production code

---

**Last Updated**: February 2026
