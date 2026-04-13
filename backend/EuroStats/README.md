# EuroStats

Real-time vote streaming microservice for the Eurovision Song Contest Voting System.

## Overview

EuroStats is a **FastAPI** service that connects to the CRUD API's gRPC server and consumes a live stream of vote events. It exposes both a REST endpoint and a WebSocket endpoint so downstream consumers can subscribe to vote updates in real-time or on-demand.

## Tech Stack

| Component | Version |
|-----------|---------|
| Python | 3.11 |
| FastAPI | 0.104.1 |
| Uvicorn | 0.24.0 |
| grpcio | 1.68.0 |
| OpenTelemetry SDK | latest |

## Architecture

```
┌─────────────────┐        gRPC StreamVotes        ┌──────────────────────┐
│  CRUD DB API    │ ──────────────────────────────► │     EuroStats        │
│  Port 8000      │        Port 50051               │     Port 8880        │
│  (gRPC Server)  │                                 │  (FastAPI + Consumer)│
└─────────────────┘                                 └──────────┬───────────┘
                                                               │
                                           ┌───────────────────┼──────────────────┐
                                           │                   │                  │
                                    GET /votes/         WS /ws/votes        OTel Collector
                                     subscribe          (real-time)          Port 4317
```

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Health check |
| `GET` | `/votes/subscribe` | Poll current + historical vote snapshot (up to 100 entries) |
| `WS` | `/ws/votes` | WebSocket stream — pushes each new vote to connected clients in real-time |

### `GET /votes/subscribe`

| Query Parameter | Type | Default | Description |
|-----------------|------|---------|-------------|
| `include_historical` | `bool` | `true` | Whether to replay historical votes before streaming new ones |

**Response**
```json
{
  "votes": [
    {
      "song_id": 1,
      "song_name": "Irgendwie, Irgendwo, Irgendwann",
      "country_voted_for": "DE",
      "country_voted_for_name": "Germany",
      "voter_country": "SE",
      "voter_country_name": "Sweden",
      "vote_count": 225,
      "timestamp": 1714000000
    }
  ],
  "count": 1
}
```

### `WS /ws/votes`

Streams individual vote events as JSON objects:

```json
{
  "type": "vote",
  "data": {
    "song_id": 1,
    "song_name": "Irgendwie, Irgendwo, Irgendwann",
    "country_voted_for": "DE",
    "country_voted_for_name": "Germany",
    "voter_country": "FR",
    "voter_country_name": "France",
    "vote_count": 226,
    "timestamp": 1714000042
  }
}
```

## Accessing the Service

EuroStats has **no port exposed directly to the host**. All external access goes through Caddy:

| Access method | URL |
|---|---|
| Via Caddy (external) | `https://<host>/eurostats/votes/subscribe` |
| Direct (internal Docker network only) | `http://eurostats:8880` |

## Project Structure

```
EuroStats/
├── Dockerfile
├── README.md
├── requirements.txt          # Production dependencies
├── requirements-test.txt     # Test dependencies
├── .python-version
├── conftest.py
├── pytest.ini
├── proto/                    # Protobuf source definition
│   └── votes.proto
├── scripts/
│   └── generate_proto.sh     # Generates Python gRPC stubs from .proto files
├── tests/                    # Integration tests
│   ├── conftest.py
│   ├── test_handle_vote.py
│   ├── test_health.py
│   ├── test_votes_subscribe.py
│   └── test_websocket.py
└── src/
    ├── main.py               # FastAPI application, lifespan, endpoints
    ├── grpc_consumer.py      # VoteStreamConsumer — gRPC client wrapper
    ├── telemetry.py          # OpenTelemetry tracing + metrics setup
    └── proto/                # Generated Python gRPC stubs
        ├── votes_pb2.py
        ├── votes_pb2_grpc.py
        └── __init__.py
```

## Docker

The image is built in a single stage from `python:3.11-slim`. A dedicated non-root user (`appuser`, UID 1001) is created at build time and all application files are owned by that user. The container runs entirely as `appuser` — it never has root access at runtime.

Port `8880` is not published to the host. The container is reachable only from other services on the shared Docker networks (`backend`, `observability`).

## gRPC Consumer

`VoteStreamConsumer` manages the connection to the CRUD API's `VoteService` gRPC server:

- **`connect()`** — opens an async gRPC channel to `crud-db-api:50051`
- **`subscribe_to_votes(include_historical)`** — async generator that yields `Vote` messages from the `StreamVotes` RPC
- **`process_votes(fn)`** — convenience wrapper to apply an async callback to each incoming vote
- **`disconnect()`** — gracefully closes the channel on shutdown

The consumer is initialised during the FastAPI application lifespan (`startup`) and torn down on `shutdown`.

## Observability

Telemetry is configured in `telemetry.py` and initialised automatically on app startup:

- **Distributed Tracing** — `TracerProvider` with `OTLPSpanExporter` (gRPC) → OTel Collector
- **Metrics** — `MeterProvider` with `OTLPMetricExporter` (gRPC) → OTel Collector
- **Auto-instrumentation** — `FastAPIInstrumentor` and `GrpcInstrumentorClient` wrap all inbound and outbound calls

The service reports itself as `service.name = eurostats`.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `GRPC_HOST` | `crud-db-api` | Hostname of the CRUD API gRPC server |
| `GRPC_PORT` | `50051` | Port of the CRUD API gRPC server |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `http://otel-collector:4317` | OTel Collector endpoint |
| `LOG_LEVEL` | `INFO` | Python logging level |