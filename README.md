# ESC Voting System

A distributed voting system for the Eurovision Song Contest, featuring a modern frontend, microservices architecture, and comprehensive observability stack.

## System Architecture

```
┌─────────────────┐
│    Frontend     │
│  Flask + HTML   │
└────────┬────────┘
         │ REST API + WebSockets
         │
┌────────▼────────────────────────┐
│      Backend Services           │
├─────────────────────────────────┤
│ • CRUD API (Go + MySQL)         │
│ • EuroStats (Python + gRPC)     │
└────────┬────────────────────────┘
         │
┌────────▼────────────────────────┐
│    Observability Stack          │
├─────────────────────────────────┤
│ • OpenTelemetry Collector       │
│ • Prometheus Monitoring         │
│ • Grafana Dashboards            │
└─────────────────────────────────┘
```

## Components

### Frontend
- **Technology**: Flask, HTML, CSS
- **Location**: `/frontend`
- **Purpose**: User-facing interface for voting
- **Communication**: REST API for data retrieval, WebSockets for real-time stats

### Backend

#### CRUD API
- **Technology**: Go, SQLite
- **Location**: `/backend/CRUD-DB-API`
- **Purpose**: Database operations and data persistence
- **API**: RESTful endpoints for vote management

#### EuroStats Microservice
- **Technology**: Python, gRPC, NumPy
- **Location**: `/backend/EuroStats`
- **Purpose**: Statistical analysis and real-time vote aggregation
- **Communication**: gRPC protocol for inter-service communication

### Observability Stack
- **Technology**: OpenTelemetry, Prometheus, Grafana
- **Location**: `/backend/Observability`
- **Purpose**: System monitoring, metrics collection, and visualization
- **Features**: 
  - Centralized telemetry collection
  - Prometheus metrics scraping
  - Grafana dashboards for visualization

## Getting Started

Each component has its own `README.md` with specific setup instructions. See the respective directories for detailed documentation:
- [Frontend Setup](./frontend/README.md)
- [CRUD API Setup](./backend/CRUD-DB-API/README.md)
- [EuroStats Setup](./backend/EuroStats/README.md)
- [Observability Setup](./backend/Observability/README.md)

## Development

The project uses Docker containerization for all services. See individual component READMEs for container-specific instructions.
