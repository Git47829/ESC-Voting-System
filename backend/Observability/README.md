# Observability Stack

Full observability stack for the ESC Voting System, providing distributed tracing, metrics collection, log aggregation, and dashboards.

## Components

| Service | Image | Port(s) | Purpose |
|---|---|---|---|
| OTel Collector | `otel/opentelemetry-collector-contrib:0.94.0` | 4317 (gRPC), 4318 (HTTP), 9464 (Prometheus) | Receives telemetry from all services and fans it out to backends |
| Prometheus | `prom/prometheus:v2.51.1` | 9090 | Scrapes metrics from the OTel Collector; stores and queries them |
| Grafana | `grafana/grafana:10.4.2` | 3000 | Dashboards, alerting, and unified querying of all data sources |
| Loki | `grafana/loki:2.9.6` | 3100 | Log aggregation and querying |
| Tempo | `grafana/tempo:2.4.2` | 3200 | Distributed trace storage and querying |

## Architecture

```
┌──────────────┐   ┌──────────────┐   ┌──────────────┐
│  CRUD API    │   │  Frontend    │   │  EuroStats   │
│  (Go)        │   │  (Flask)     │   │  (Python)    │
└──────┬───────┘   └──────┬───────┘   └──────┬───────┘
       │ OTLP/HTTP         │ OTLP/HTTP         │ OTLP/gRPC
       └───────────────────┼───────────────────┘
                           ▼
              ┌────────────────────────┐
              │    OTel Collector      │
              │    4317 (gRPC)         │
              │    4318 (HTTP)         │
              │    9464 (scrape)       │
              └───────────┬────────────┘
                          │
          ┌───────────────┼───────────────┐
          ▼               ▼               ▼
   ┌────────────┐  ┌────────────┐  ┌────────────┐
   │ Prometheus │  │   Tempo    │  │    Loki    │
   │   9090     │  │   3200     │  │   3100     │
   └─────┬──────┘  └─────┬──────┘  └─────┬──────┘
         └───────────────┼────────────────┘
                         ▼
                 ┌───────────────┐
                 │    Grafana    │
                 │    3000       │
                 └───────────────┘
```

## Service Details

### OTel Collector

Configured via `OTel/src/otel-collector.yml`. Acts as the central telemetry hub:

- **Receivers** — accepts OTLP over gRPC (`:4317`) and HTTP (`:4318`)
- **Exporters** — forwards traces to Tempo, metrics to Prometheus (scraped at `:9464`), and logs to Loki
- **Processors** — batch processor for efficient downstream delivery

### Prometheus

Configured via `Prometheus/src/prometheus.yml`. Scrapes the OTel Collector's Prometheus endpoint (`:9464`) to ingest metrics from all instrumented services. Accessible at [http://localhost:9090](http://localhost:9090).

### Grafana

Configured via `grafana/src/provisioning/`. Data sources and dashboards are provisioned automatically on startup:

- **Prometheus** — metrics queries
- **Loki** — log queries
- **Tempo** — trace queries and trace-to-log correlation

Accessible at [http://localhost:3000](http://localhost:3000). Default credentials: `admin` / `admin`.

### Loki

Stores and indexes structured log output from all services. Uses a local filesystem volume (`loki_data`) for persistence. Queried directly from Grafana using LogQL.

### Tempo

Configured via `Tempo/src/tempo.yml`. Receives trace spans forwarded by the OTel Collector and stores them in a local volume (`tempo_data`). Supports trace lookup by trace ID and integration with Loki for log correlation.

A `tempo-init` helper container runs first to set the correct permissions on the data volume before Tempo starts.

## Volumes

| Volume | Used By | Purpose |
|---|---|---|
| `grafana_data` | Grafana | Dashboard state, users, preferences |
| `loki_data` | Loki | Persisted log chunks and index |
| `tempo_data` | Tempo | Persisted trace data |

## Network

All observability services are attached to the `observability` Docker network. Application services (`api`, `frontend`, `eurostats`) are also connected to this network so they can reach the OTel Collector at `otel-collector:4317` / `otel-collector:4318`.

## Instrumented Services

| Service | Exporter | Signals |
|---|---|---|
| CRUD DB API (Go) | OTLP/HTTP → `:4318` | Traces, Metrics (Prometheus) |
| Frontend (Flask) | OTLP/HTTP → `:4318` | Traces, Metrics |
| EuroStats (Python) | OTLP/gRPC → `:4317` | Traces, Metrics |
