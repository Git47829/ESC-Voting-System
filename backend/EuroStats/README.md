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
      "song_name": "Satellite Reprise",
      "country_voted_for": "DEU",
      "country_voted_for_name": "Germany",
      "voter_country": "SWE",
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
    "song_name": "Satellite Reprise",
    "country_voted_for": "DEU",
    "country_voted_for_name": "Germany",
    "voter_country": "FRA",
    "voter_country_name": "France",
    "vote_count": 226,
    "timestamp": 1714000042
  }
}
```

## Project Structure

```
EuroStats/
├── Dockerfile
├── README.md
├── requirements.txt
├── .python-version
├── proto/                   # Protobuf definition files
├── scripts/
│   └── generate_proto.sh    # Generates Python gRPC stubs from .proto files
└── src/
    ├── main.py              # FastAPI application, lifespan, endpoints
    ├── grpc_consumer.py     # VoteStreamConsumer — gRPC client wrapper
    └── telemetry.py         # OpenTelemetry tracing + metrics setup
```

## gRPC Consumer

`VoteStreamConsumer` manages the connection to the CRUD API's `VoteService` gRPC server:

- **`connect()`** — opens a secure async gRPC channel to `crud-db-api:50051`
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
