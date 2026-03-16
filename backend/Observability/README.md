# Observability Stack

Full observability stack for the ESC Voting System, providing distributed tracing, metrics collection, log aggregation, and dashboards.

All observability services are **internal-only** — no ports are exposed directly to the host. Access is provided exclusively through the Caddy reverse proxy over HTTPS.

## Components

| Service | Image | Internal Port | Caddy Subpath | Purpose |
|---|---|---|---|---|
| OTel Collector | `otel/opentelemetry-collector-contrib:0.94.0` | 4317 (gRPC), 4318 (HTTP) | Internal only | Receives telemetry from all services and fans it out to backends |
| Prometheus | `prom/prometheus:v2.51.1` | 9090 | `/prometheus` | Scrapes metrics from the OTel Collector; stores and queries them |
| Grafana | `grafana/grafana:10.4.2` | 3000 | `/grafana` | Dashboards, alerting, and unified querying of all data sources |
| Loki | `grafana/loki:2.9.6` | 3100 | `/loki` | Log aggregation and querying |
| Tempo | `grafana/tempo:2.4.2` | 3200 | `/tempo` | Distributed trace storage and querying |

## Architecture

```
┌──────────────┐    ┌──────────────┐   ┌──────────────┐   ┌────────────────────┐
│  CRUD API    │    │  Frontend    │   │  EuroStats   │   │ PublicVoteConverter│
│  (Go)        │    │  (Flask)     │   │  (Python)    │   │       (Go)         │
└──────┬───────┘    └──────┬───────┘   └──────┬───────┘   └────────┬───────────┘
       │ OTLP/HTTP         │ OTLP/HTTP         │ OTLP/gRPC          │ OTLP/HTTP
       └───────────────────┼───────────────────┴────────────────────┘
                           ▼
              ┌────────────────────────┐
              │    OTel Collector      │
              │    4317 (gRPC)         │
              │    4318 (HTTP)         │
              │    (internal only)     │
              └───────────┬────────────┘
                          │
          ┌───────────────┼───────────────┐
          ▼               ▼               ▼
   ┌────────────┐  ┌────────────┐  ┌────────────┐
   │ Prometheus │  │   Tempo    │  │    Loki    │
   │  (9090)    │  │  (3200)    │  │  (3100)    │
   │ /prometheus│  │  /tempo    │  │  /loki     │
   └─────┬──────┘  └─────┬──────┘  └─────┬──────┘
         └───────────────┼────────────────┘
                         ▼
                 ┌───────────────┐
                 │    Grafana    │
                 │    (3000)     │
                 │   /grafana    │
                 └───────┬───────┘
                         │ HTTPS (via Caddy)
                         ▼
                 ┌───────────────┐
                 │     Caddy     │
                 │  443 / 80     │
                 └───────────────┘
```

## Accessing the Stack

All URLs are relative to the server's hostname or IP address. Replace `<host>` with `localhost` or your LAN IP.

| Service | URL |
|---|---|
| Grafana | `https://<host>/grafana` |
| Prometheus | `https://<host>/prometheus` |
| Tempo | `https://<host>/tempo` |
| Loki | `https://<host>/loki` |

> **Note:** The OTel Collector's ingestion ports (4317/4318) are not exposed externally. Application services push telemetry directly to `otel-collector:4317` / `otel-collector:4318` over the internal `observability` Docker network.

### Default Grafana Login

```
Username: admin
Password: admin
```

## Service Details

### OTel Collector

Configured via `OTel/src/otel-collector.yml`. Acts as the central telemetry hub:

- **Receivers** — accepts OTLP over gRPC (`:4317`) and HTTP (`:4318`)
- **Exporters** — forwards traces to Tempo, metrics to Prometheus (scraped at `:9464`), and logs to Loki
- **Processors** — `batch` processor for efficient downstream delivery; `resource/loki_labels` and `attributes/loki_labels` processors promote `service.name`, `traceID`, `spanID`, and `level` to Loki stream labels for efficient querying and trace-to-log correlation

### Prometheus

Configured via `Prometheus/src/prometheus.yml`. Scrapes the following targets every 15 s:

| Job | Target |
|---|---|
| `otel-collector` | `otel-collector:9464` — application metrics forwarded by the collector |
| `otel-collector-internal` | `otel-collector:8888` — collector self-metrics |
| `esc-frontend` | `esc-frontend:5000/metrics` — multiprocess-safe Prometheus metrics via `PROMETHEUS_MULTIPROC_DIR` |
| `esc-crud-api` | `db-crud-api:8000/metrics/` — request duration, size, count histograms + contest and vote counters |
| `loki` | `loki:3100/metrics` |
| `tempo` | `tempo:3200/metrics` |

Accessible via Caddy at `https://<host>/prometheus`.

### Grafana

Configured via `grafana/src/provisioning/`. Data sources and dashboards are provisioned automatically on startup:

- **Prometheus** — metrics queries (`http://prometheus:9090`)
- **Loki** — log queries (`http://loki:3100`)
- **Tempo** — trace queries and trace-to-log correlation (`http://tempo:3200`)

Grafana is configured with `GF_SERVER_ROOT_URL=https://<host>/grafana` and `GF_SERVER_SERVE_FROM_SUB_PATH=true` so it functions correctly behind the `/grafana` subpath.

Accessible via Caddy at `https://<host>/grafana`.

### Loki

Stores and indexes structured log output pushed by the OTel Collector. Uses a local filesystem volume (`loki_data`) for persistence. Queried directly from Grafana using LogQL.

### Tempo

Configured via `Tempo/src/tempo.yml`. Receives trace spans forwarded by the OTel Collector and stores them in a local volume (`tempo_data`). The metrics generator is enabled with the `span-metrics` and `service-graphs` processors, which write RED metrics and service graph metrics to Prometheus via remote write.

A `tempo-init` helper container runs first to set the correct ownership (`10001:10001`) on the data volume before Tempo starts.

## Volumes

| Volume | Used By | Purpose |
|---|---|---|
| `grafana_data` | Grafana | Dashboard state, users, preferences |
| `loki_data` | Loki | Persisted log chunks and index |
| `tempo_data` | Tempo | Persisted trace data |

## Networks

All observability services are attached to the `observability` Docker network. Application services (`api`, `frontend`, `eurostats`) are also connected to this network so they can reach the OTel Collector at `otel-collector:4317` / `otel-collector:4318`.

Caddy is attached to both the `frontend` and `observability` networks, making it the sole external entry point for the entire stack.

## Instrumented Services

| Service | Exporter | Signals |
|---|---|---|
| CRUD DB API (Go) | OTLP/HTTP → `otel-collector:4318` | Traces, Metrics (Prometheus scrape at `/metrics/`) |
| Frontend (Flask) | OTLP/HTTP → `otel-collector:4318` | Traces, Metrics (Prometheus scrape at `/metrics`); custom metrics: `backend_call_duration_seconds`, `votes_total`, `active_sessions` |
| EuroStats (Python) | OTLP/gRPC → `otel-collector:4317` | Traces, Metrics |
| PublicVoteConverter (Go) | OTLP/HTTP → `otel-collector:4318` | Traces, Metrics (Prometheus scrape at `/metrics`); custom metrics: `esc_converter_http_requests_total`, `esc_converter_http_request_duration_seconds` |

## Application Metrics (Frontend)

The Flask frontend exposes the following custom Prometheus metrics at `/metrics`:

| Metric | Type | Labels | Description |
|---|---|---|---|
| `backend_call_duration_seconds` | Histogram | `endpoint`, `status_code` | Duration of each call from the frontend to the CRUD API |
| `votes_total` | Counter | `type` (`public` or `jury`) | Total votes cast, broken down by voter type |
| `active_sessions` | Gauge | — | Number of currently logged-in admin/jury sessions |

## Application Metrics (CRUD DB API)

The Go CRUD API exposes the following Prometheus metrics at `/metrics/`:

| Metric | Type | Labels | Description |
|---|---|---|---|
| `http_request_size_bytes` | Histogram | `method`, `path` | Request body size per endpoint |
| `http_response_size_bytes` | Histogram | `method`, `path`, `status` | Response body size per endpoint and status code |
| `http_request_duration_seconds` | Histogram | `method`, `path`, `status` | Request latency per endpoint and status code |
| `http_requests_total` | Counter | `method`, `path`, `status` | Total request count per endpoint and status code |
