# Observability Stack

Full observability stack for the ESC Voting System, providing distributed tracing, metrics collection, log aggregation, and dashboards.

All observability services are **internal-only** — no ports are exposed directly to the host. Access is provided exclusively through the Caddy reverse proxy over HTTPS.

## Components

| Service | Image | Internal Port | Caddy Route | Purpose |
|---|---|---|---|---|
| OTel Collector | `otel/opentelemetry-collector-contrib:0.94.0` | 4317 (gRPC), 4318 (HTTP), 9464 (Prometheus scrape), 8888 (internal metrics) | Internal only | Receives telemetry from all services and fans it out to backends |
| Prometheus | `prom/prometheus:v2.51.1` | 9090 | `/prometheus` | Scrapes metrics from the OTel Collector and service endpoints; stores and queries metrics |
| Grafana | `grafana/grafana:10.4.2` | 3000 | `/grafana` | Dashboards, alerting, and unified querying of all data sources |
| Loki | `grafana/loki:2.9.6` | 3100 | `/loki` | Log aggregation and querying for structured logs |
| Tempo | `grafana/tempo:2.4.2` | 3200 | `/tempo` | Distributed trace storage and querying with metrics generation |
| Caddy | `caddy:2.8-alpine` | 2019 (admin/metrics) | 443, 80 | HTTPS reverse proxy with automatic certificate management and access logging to OTel Collector |

## Architecture

```
┌──────────────┐    ┌──────────────┐   ┌──────────────┐   ┌────────────────────┐
│  CRUD API    │    │  Frontend    │   │  EuroStats   │   │ PublicVoteConverter│
│  (Go)        │    │  (Flask)     │   │  (Python)    │   │       (Go)         │
│ (db-crud-api)│    │ (esc-frontend)│  │ (eurostats)  │   │(public-vote-conv.) │
└──────┬───────┘    └──────┬───────┘   └──────┬───────┘   └────────┬───────────┘
       │ OTLP/HTTP         │ OTLP/HTTP         │ OTLP/gRPC          │ OTLP/HTTP
       │ (8000)            │ (5000/metrics)    │ (:4317)            │ (8090)
       └───────────────────┼───────────────────┴────────────────────┘
                           │ observability network
                           ▼
              ┌────────────────────────┐
              │    OTel Collector      │
              │    4317 (gRPC)         │
              │    4318 (HTTP)         │
              │    9464 (Prometheus)   │
              │    8888 (self-metrics) │
              │    (internal only)     │
              └───────┬────────────────┘
                      │
         ┌────────────┼────────────┐
         ▼            ▼            ▼
    ┌─────────┐ ┌─────────┐ ┌──────────┐
    │Prometheus│ │ Tempo   │ │   Loki   │
    │ (9090)  │ │(3200)   │ │ (3100)   │
    └────┬────┘ └────┬────┘ └────┬─────┘
         │           │           │
         │      ┌────┴───────────┤
         │      │                │
         └──────┼────┬───────────┘
                │    │
                ▼    ▼
           ┌──────────────┐
           │   Grafana    │
           │   (3000)     │
           │  /grafana    │
           └──────┬───────┘
                  │ HTTPS
                  ▼
           ┌──────────────┐        ┌─────────────────┐
           │    Caddy     │◄──────►│ Caddy Logs      │
           │  443 / 80    │        │ (JSON access)   │
           └──────────────┘        └────────┬────────┘
                                           │ OTel Collector
                                           │ (via filelog receiver)
                                           ▼
                                        Loki
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

- **Receivers:**
  - `otlp` — accepts OTLP over gRPC (`:4317`) and HTTP (`:4318`)
  - `filelog` — tails Caddy's access logs (`/var/log/caddy/access.log`) and parses JSON-formatted entries into structured logs

- **Exporters:**
  - `prometheus` — exposes metrics on `:9464` (Prometheus scrapes this)
  - `otlp` — forwards traces to Tempo (`:4417`)
  - `loki` — sends logs to Loki (`:3100`)
  - `logging` — debug exporter for trace visibility

- **Processors:**
  - `batch` — batches telemetry for efficient downstream delivery
  - `resource/loki_labels` — promotes `service.name` from resource attributes to Loki stream labels
  - `attributes/loki_labels` — promotes `traceID`, `spanID`, and `level` from log attributes to Loki stream labels for efficient querying and trace-to-log correlation

- **Telemetry:**
  - Self-metrics exposed on `:8888` (Prometheus scrapes these too)
  - Health check, pprof, and zpages extensions enabled for diagnostics

### Prometheus

Configured via `Prometheus/src/prometheus.yml`. Scrapes the following targets every 15 seconds:

| Job | Target | Metrics Path |
|---|---|---|
| `otel-collector` | `otel-collector:9464` | `/metrics` | Application metrics forwarded by the OTel Collector |
| `otel-collector-internal` | `otel-collector:8888` | `/metrics` | OTel Collector self-metrics |
| `esc-frontend` | `esc-frontend:5000` | `/metrics` | Flask frontend metrics (multiprocess-safe via `PROMETHEUS_MULTIPROC_DIR`); custom metrics: request duration, vote counters, session count |
| `esc-crud-api` | `db-crud-api:8000` | `/metrics/` | Go CRUD API metrics: request duration, size, count histograms + database operation counters |
| `esc-points-converter` | `public-vote-converter:8090` | `/metrics` | Go vote converter metrics: request counts and latencies |
| `esc-euromail` | `euromail:3000` | `/metrics` | Email service metrics |
| `loki` | `loki:3100` | `/metrics` | Loki self-metrics |
| `tempo` | `tempo:3200` | `/metrics` | Tempo self-metrics |
| `caddy` | `caddy:2019` | `/metrics` | Caddy reverse proxy metrics (admin API) |

Prometheus stores metrics in local memory with remote write receiver enabled for Tempo's metrics generator. Accessible via Caddy at `https://<host>/prometheus`.

### Grafana

Configured via `grafana/src/provisioning/`. Data sources and dashboards are provisioned automatically on startup:

**Data Sources:**
- **Prometheus** (UID: `PBFA97CFB590B2093`) — metrics queries from `http://prometheus:9090`; default data source with 15-second time interval
- **Loki** (UID: `P8E80F9AEF21F6940`) — log queries from `http://loki:3100`
- **Tempo** (UID: `P214B5B846CF3925F`) — trace queries from `http://tempo:3200`
  - Configured with `tracesToLogsV2` linking to Loki (filters by `service_name` and `traceID`)
  - Configured with `tracesToMetrics` linking to Prometheus
  - Service map and node graph enabled for distributed trace visualization
  - Loki search enabled for span-based log discovery

**Dashboards:**
Auto-provisioned from JSON files in `dashboards/`:
- `esc-observability-stack.json` — Overview of all observability components
- `esc-crud-api.json` — CRUD API performance and health
- `esc-frontend.json` — Frontend performance, sessions, and custom metrics
- `esc-eurostats.json` — EuroStats service metrics
- `esc-public-vote-converter.json` — Vote converter performance and request patterns
- `esc-euromail.json` — Email service metrics
- `esc-caddy.json` — Reverse proxy performance and request routing

Grafana is configured with `GF_SERVER_ROOT_URL=https://localhost/grafana` and `GF_SERVER_SERVE_FROM_SUB_PATH=true` so it functions correctly behind the `/grafana` subpath when accessed through Caddy.

Accessible via Caddy at `https://<host>/grafana`.

### Loki

Configured with default settings. Receives and stores structured logs pushed by the OTel Collector via the `loki` exporter endpoint (`http://loki:3100/loki/api/v1/push`).

**Log Sources:**
- **Application logs** — from `otlp` receiver (sent by backend services)
- **Caddy access logs** — from `filelog` receiver (JSON-formatted, tailed from `/var/log/caddy/access.log`)

**Stream Labels:**
- `service_name` — promoted from OTel resource attributes (values: `caddy`, `esc-frontend`, `db-crud-api`, `eurostats`, `public-vote-converter`, `euromail`)
- `level` — promoted from log attributes (values: `INFO`, `WARN`, `ERROR`)
- `traceID` and `spanID` — promoted to labels for efficient trace-to-log correlation in Grafana

Uses a local filesystem volume (`loki_data`) for persistence. Queried directly from Grafana using LogQL with efficient label-based filtering and trace correlation.

Accessible via Caddy at `https://<host>/loki`.

### Tempo

Configured via `Tempo/src/tempo.yml`. Receives trace spans forwarded by the OTel Collector via the `otlp` exporter and stores them in a local volume (`tempo_data`).

**Trace Ingestion:**
- Receives OTLP over gRPC (`:4417`) and HTTP (`:4418`) from the OTel Collector
- Stores traces in local backend with write-ahead logging (WAL) for durability
- Max block duration: 5 minutes

**Metrics Generation:**
- **Enabled** — automatically generates RED metrics (Rate, Errors, Duration) and service graphs from trace spans
- Processors:
  - `span-metrics` — extracts per-span latencies and error rates
  - `service-graphs` — builds inter-service dependency graphs and call patterns
- Remote write to Prometheus (`:9090`) for long-term metrics storage
- Exemplars enabled for bridging traces to metrics in Grafana

A `tempo-init` helper container ensures proper volume ownership (`10001:10001`) before Tempo starts.

Accessible via Caddy at `https://<host>/tempo`.

### Caddy

Acts as the HTTPS reverse proxy and entry point for the entire system. Configured via `Caddyfile` in `backend/Caddy/src/`.

**Features:**
- **HTTPS with automatic certificates** — uses internal local CA (`ESC Voting Local CA`) to issue certificates on-demand for any hostname/IP
- **HTTP → HTTPS redirect** — all HTTP requests on port 80 are redirected to HTTPS on 443
- **Access logging** — JSON-formatted logs written to `/var/log/caddy/access.log` (bind-mounted and tailed by OTel Collector's `filelog` receiver for centralized log aggregation)
- **Admin metrics** — exposes `/metrics` on port 2019 (internal, scraped by Prometheus)

**Reverse Proxy Routes:**
- `/` → `esc-frontend:3001` (catch-all, lowest priority)
- `/crud-api/*` → `db-crud-api:8000` (strip prefix)
- `/eurostats/*` → `eurostats:8880` (strip prefix)
- `/esc-converter/*` → `public-vote-converter:8090` (strip prefix)
- `/grafana/*` → `grafana:3000` (no prefix strip; Grafana serves under `/grafana` internally)
- `/prometheus/*` → `prometheus:9090` (strip prefix; Prometheus serves at `/` internally)
- `/tempo/*` → `tempo:3200` (strip prefix)
- `/loki/*` → `loki:3100` (strip prefix)

## Volumes

| Volume | Used By | Purpose |
|---|---|---|
| `grafana_data` | Grafana | Dashboard state, users, preferences, and data source configurations |
| `loki_data` | Loki | Persisted log chunks, index, and metadata |
| `tempo_data` | Tempo | Persisted trace blocks, WAL, and metrics generator state |
| `caddy_data` | Caddy | Certificate data and CA keys for HTTPS |
| `caddy_config` | Caddy | Caddy runtime configuration state |
| `caddy_logs` | Caddy, OTel Collector | JSON access logs written by Caddy and tailed by OTel Collector's `filelog` receiver |

## Networks

All observability services are attached to the `observability` Docker network. Application services (`api`, `euromail`, `frontend`, `eurostats`, `public-vote-converter`) are also connected to this network so they can reach the OTel Collector at `otel-collector:4317` / `otel-collector:4318`.

Caddy is attached to both the `frontend` and `observability` networks, making it the sole external entry point for the entire system. All HTTP/HTTPS traffic flows through Caddy, which logs each request in JSON format to `/var/log/caddy/access.log`, where the OTel Collector's `filelog` receiver tails it and forwards the structured logs to Loki.

## Instrumented Services

| Service | Exporter | Signals |
|---|---|---|
| CRUD DB API (Go, `db-crud-api`) | OTLP/HTTP → `otel-collector:4318` | Traces, Metrics (Prometheus scrape at `/metrics/`); histograms: request size, response size, latency; counters: request count per endpoint and status |
| Frontend (Flask, `esc-frontend`) | OTLP/HTTP → `otel-collector:4318` | Traces, Metrics (Prometheus scrape at `/metrics`); custom metrics: `backend_call_duration_seconds`, `votes_total`, `active_sessions` |
| EuroStats (Python, `eurostats`) | OTLP/gRPC → `otel-collector:4317` | Traces, Metrics (Prometheus scrape at `:8880/metrics`); gRPC service latencies and call counts |
| PublicVoteConverter (Go, `public-vote-converter`) | OTLP/HTTP → `otel-collector:4318` | Traces, Metrics (Prometheus scrape at `/metrics`); custom metrics: `esc_converter_http_requests_total`, `esc_converter_http_request_duration_seconds` |
| EuroMail (Node.js, `euromail`) | Not instrumented yet | — |

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

## Application Metrics (PublicVoteConverter)

The Go vote converter exposes the following custom Prometheus metrics at `/metrics`:

| Metric | Type | Labels | Description |
|---|---|---|---|
| `esc_converter_http_requests_total` | Counter | `method`, `endpoint`, `status` | Total HTTP request count per endpoint and status code |
| `esc_converter_http_request_duration_seconds` | Histogram | `method`, `endpoint` | Request latency per endpoint |

## Application Metrics (EuroStats)

The Python EuroStats service exposes Prometheus metrics at `:8880/metrics` with OpenTelemetry instrumentation that automatically generates:

- `http_server_request_duration` — gRPC request latency
- `http_server_requests_total` — gRPC request count
- Database query latencies and call counts (from OTel instrumentation)

## Data Flow

1. **Traces**: Services → OTel Collector (OTLP/gRPC or HTTP) → Tempo (via `otlp` exporter at `:4417`) → Grafana (via Tempo data source)
2. **Metrics**: 
   - From services → OTel Collector (OTLP) → Prometheus (scrapes `:9464`)
   - From direct scrapes → Prometheus (scrapes service `/metrics` endpoints directly)
   - From Tempo metrics generator → Prometheus (remote write)
3. **Logs**:
   - From services → OTel Collector (OTLP) → Loki (via `loki` exporter)
   - From Caddy → Loki (OTel Collector tails Caddy access logs via `filelog` receiver)
4. **Visualization**: Grafana queries Prometheus (metrics), Loki (logs), and Tempo (traces); inter-datasource links enable seamless navigation between traces, logs, and metrics

## Getting Started

### Starting the Stack

```bash
docker-compose up -d
```

Wait for all services to be healthy, then access Grafana at `https://localhost/grafana`.

### Viewing Logs

In Grafana, go to **Explore** and select the **Loki** data source. Try these queries:

```logql
{service_name="esc-frontend"} | json
{level="ERROR"}
{traceID="<your-trace-id>"}
```

### Viewing Traces

In Grafana, go to **Explore** and select the **Tempo** data source. Search for traces by service name or tags, or use the Tempo search UI.

### Viewing Metrics

In Grafana, go to **Explore** and select the **Prometheus** data source, or view pre-built dashboards in the **ESC Voting System** folder.

### Troubleshooting

- **Services not sending telemetry**: Verify `OTEL_EXPORTER_OTLP_HTTP_ENDPOINT` or `OTEL_EXPORTER_OTLP_ENDPOINT` environment variables point to `otel-collector:4318` (HTTP) or `otel-collector:4317` (gRPC)
- **No logs appearing**: Check that the OTel Collector is running and Loki is accessible; verify Caddy logs exist at `/var/log/caddy/access.log`
- **Prometheus targets down**: Check network connectivity; ensure all services are on the `observability` network
- **Grafana data source errors**: Verify data source URLs match the internal Docker network hostnames (e.g., `http://prometheus:9090`, not `localhost`)

