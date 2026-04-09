"""
telemetry.py — Observability setup for the ESC Voting System frontend.

Mirrors the pattern used by the EuroStats Python service, adapted for Flask.

Pillars
-------
1. Traces  — OpenTelemetry SDK → OTLP/HTTP → otel-collector → (logging exporter)
2. Metrics — OpenTelemetry SDK → OTLP/gRPC → otel-collector → Prometheus scrape
             + prometheus_client scrape endpoint at GET /metrics
3. Logs    — Python logging JSON handler → OTLP/gRPC logs exporter
             → otel-collector → Loki

Custom business metrics exposed
--------------------------------
- esc_frontend_http_requests_total           (counter)   method, path, status_code
- esc_frontend_http_request_duration_seconds (histogram) method, path, status_code
- esc_frontend_http_request_size_bytes       (histogram) method, path
- esc_frontend_http_response_size_bytes      (histogram) method, path, status_code
- esc_frontend_api_backend_duration_seconds  (histogram) endpoint, status_code
- esc_frontend_votes_cast_total              (counter)   type=public|jury
- esc_frontend_active_sessions               (gauge)     —

Multiprocess note
-----------------
prometheus_client requires special handling when running under gunicorn with
multiple workers. When PROMETHEUS_MULTIPROC_DIR is set, every worker writes
its metrics to that shared directory and the /metrics endpoint aggregates
them all via a multiprocess CollectorRegistry before responding.
"""

import logging
import os
import time

from flask import Flask, Response, g, request

from opentelemetry import metrics, trace
from opentelemetry.exporter.otlp.proto.grpc._log_exporter import OTLPLogExporter
from opentelemetry.exporter.otlp.proto.grpc.metric_exporter import OTLPMetricExporter
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.instrumentation.flask import FlaskInstrumentor
from opentelemetry.instrumentation.requests import RequestsInstrumentor
from opentelemetry.sdk._logs import LoggerProvider, LoggingHandler
from opentelemetry.sdk._logs.export import BatchLogRecordProcessor
from opentelemetry.sdk.metrics import MeterProvider
from opentelemetry.sdk.metrics.export import PeriodicExportingMetricReader
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor

from prometheus_client import (
    CONTENT_TYPE_LATEST,
    REGISTRY,
    CollectorRegistry,
    Counter,
    Gauge,
    Histogram,
    generate_latest,
    multiprocess,
)

from pythonjsonlogger import jsonlogger

logger = logging.getLogger(__name__)


PROM_REQUEST_COUNT = Counter(
    "esc_frontend_http_requests_total",
    "Total HTTP requests handled by the frontend",
    ["method", "path", "status_code"],
)

PROM_REQUEST_DURATION = Histogram(
    "esc_frontend_http_request_duration_seconds",
    "HTTP request duration in seconds",
    ["method", "path", "status_code"],
    buckets=[0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0],
)

PROM_REQUEST_SIZE = Histogram(
    "esc_frontend_http_request_size_bytes",
    "HTTP request body size in bytes",
    ["method", "path"],
    buckets=[64, 256, 1024, 4096, 16384, 65536, 262144, 1048576],
)

PROM_RESPONSE_SIZE = Histogram(
    "esc_frontend_http_response_size_bytes",
    "HTTP response body size in bytes",
    ["method", "path", "status_code"],
    buckets=[64, 256, 1024, 4096, 16384, 65536, 262144, 1048576],
)

PROM_BACKEND_DURATION = Histogram(
    "esc_frontend_api_backend_duration_seconds",
    "Latency of outbound calls from frontend to the CRUD API backend",
    ["endpoint", "status_code"],
    buckets=[0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0],
)

PROM_VOTES_CAST = Counter(
    "esc_frontend_votes_cast_total",
    "Total votes successfully submitted through the frontend",
    ["type"],  # public | jury
)

PROM_ACTIVE_SESSIONS = Gauge(
    "esc_frontend_active_sessions",
    "Approximate number of active authenticated sessions",
    multiprocess_mode="livesum",
)

_otel_request_counter = None
_otel_request_duration = None
_otel_backend_duration = None
_otel_votes_counter = None

_telemetry_initialised = False


def _build_resource() -> Resource:
    return Resource.create(
        {
            "service.name": os.getenv("OTEL_SERVICE_NAME", "esc-voting-frontend"),
            "service.version": os.getenv("OTEL_SERVICE_VERSION", "1.0.0"),
            "deployment.environment": os.getenv("OTEL_ENVIRONMENT", "production"),
        }
    )


def _setup_tracing(resource: Resource, otlp_http_endpoint: str) -> None:
    """Configure the OTel TracerProvider with OTLP/HTTP exporter."""
    tracer_provider = TracerProvider(resource=resource)
    span_exporter = OTLPSpanExporter(
        endpoint=f"{otlp_http_endpoint}/v1/traces",
    )
    tracer_provider.add_span_processor(BatchSpanProcessor(span_exporter))
    trace.set_tracer_provider(tracer_provider)
    logger.info("OTel tracing initialised → %s", otlp_http_endpoint)


def _setup_metrics(resource: Resource, otlp_grpc_endpoint: str) -> None:
    """Configure the OTel MeterProvider with OTLP/gRPC exporter."""
    global _otel_request_counter, _otel_request_duration
    global _otel_backend_duration, _otel_votes_counter

    metric_exporter = OTLPMetricExporter(
        endpoint=otlp_grpc_endpoint,
        insecure=True,
    )
    reader = PeriodicExportingMetricReader(
        metric_exporter, export_interval_millis=15_000
    )
    meter_provider = MeterProvider(resource=resource, metric_readers=[reader])
    metrics.set_meter_provider(meter_provider)

    meter = metrics.get_meter("esc-voting-frontend", version="1.0.0")

    _otel_request_counter = meter.create_counter(
        name="esc.frontend.http.requests",
        description="Total HTTP requests",
        unit="1",
    )
    _otel_request_duration = meter.create_histogram(
        name="esc.frontend.http.request.duration",
        description="HTTP request duration",
        unit="s",
    )
    _otel_backend_duration = meter.create_histogram(
        name="esc.frontend.api.backend.duration",
        description="Outbound backend API call duration",
        unit="s",
    )
    _otel_votes_counter = meter.create_counter(
        name="esc.frontend.votes.cast",
        description="Votes successfully submitted through the frontend",
        unit="1",
    )
    logger.info("OTel metrics initialised → %s", otlp_grpc_endpoint)


def _setup_logging(resource: Resource, otlp_grpc_endpoint: str) -> None:
    """
    Configure a two-pronged logging setup:
      1. OTel LoggerProvider → OTLP/gRPC → otel-collector → Loki
      2. JSON formatter on the root handler so every log line emitted
         by gunicorn/Flask is structured and carries trace context.
    """
    # OTel log pipeline
    log_provider = LoggerProvider(resource=resource)
    log_exporter = OTLPLogExporter(endpoint=otlp_grpc_endpoint, insecure=True)
    log_provider.add_log_record_processor(BatchLogRecordProcessor(log_exporter))

    otel_handler = LoggingHandler(
        level=logging.INFO,
        logger_provider=log_provider,
    )

    # JSON formatter for stdout — gunicorn captures stdout, and any log
    # aggregator that tails container stdout will receive structured records.
    json_formatter = jsonlogger.JsonFormatter(
        fmt="%(asctime)s %(name)s %(levelname)s %(message)s",
        rename_fields={"asctime": "timestamp", "levelname": "level"},
    )

    stream_handler = logging.StreamHandler()
    stream_handler.setFormatter(json_formatter)

    root_logger = logging.getLogger()
    root_logger.setLevel(logging.INFO)
    # Remove any existing handlers to avoid duplicate output on --preload forks.
    root_logger.handlers.clear()
    root_logger.addHandler(stream_handler)
    root_logger.addHandler(otel_handler)

    logger.info("OTel logging initialised → %s", otlp_grpc_endpoint)

def setup_telemetry(app: Flask) -> None:
    """
    Initialise all three observability pillars and wire them into the Flask app.

    Idempotent — safe to call multiple times (subsequent calls are no-ops).
    This matters under gunicorn --preload: the master process calls this once
    and workers inherit the instrumented state via fork() without re-running it.
    """
    global _telemetry_initialised
    if _telemetry_initialised:
        return
    _telemetry_initialised = True

    otlp_http = os.getenv(
        "OTEL_EXPORTER_OTLP_HTTP_ENDPOINT", "http://otel-collector:4318"
    )
    otlp_grpc = os.getenv(
        "OTEL_EXPORTER_OTLP_GRPC_ENDPOINT", "http://otel-collector:4317"
    )

    try:
        resource = _build_resource()
        _setup_tracing(resource, otlp_http)
        _setup_metrics(resource, otlp_grpc)
        _setup_logging(resource, otlp_grpc)

        # Auto-instrument Flask — injects trace context into every request
        # and creates spans automatically for each route handler.
        FlaskInstrumentor().instrument_app(
            app,
            excluded_urls="/metrics,/health",
        )

        # Auto-instrument the `requests` library — every outbound call to
        # the CRUD API backend gets a child span and the W3C traceparent
        # header injected so traces are correlated end-to-end.
        RequestsInstrumentor().instrument()

        # Per-request hooks for Prometheus recording and structured logging.
        _register_request_hooks(app)

        # Prometheus pull endpoint — multiprocess-aware.
        _register_metrics_endpoint(app)

        # Lightweight liveness probe that doesn't touch the backend API.
        _register_health_endpoint(app)

        app.logger.info("Telemetry fully initialised for esc-voting-frontend")

    except Exception as exc:
        # Observability must never crash the application.
        app.logger.error("Failed to initialise telemetry: %s", exc, exc_info=True)



def _register_request_hooks(app: Flask) -> None:
    """Register before/after request hooks to record metrics and logs."""

    @app.before_request
    def _before() -> None:
        g._request_start_time = time.perf_counter()

    @app.after_request
    def _after(response: Response) -> Response:
        # Skip the scrape and health endpoints to avoid feedback loops.
        if request.path in ("/metrics", "/health", "/favicon.ico"):
            return response

        duration = time.perf_counter() - getattr(
            g, "_request_start_time", time.perf_counter()
        )
        status_code = str(response.status_code)
        method = request.method
        # Normalise dynamic/high-cardinality path segments.
        path = _normalise_path(request.path)

        req_size = request.content_length or 0
        resp_size = response.calculate_content_length() or 0

        # --- Prometheus ---
        PROM_REQUEST_COUNT.labels(method, path, status_code).inc()
        PROM_REQUEST_DURATION.labels(method, path, status_code).observe(duration)
        if req_size > 0:
            PROM_REQUEST_SIZE.labels(method, path).observe(req_size)
        if resp_size > 0:
            PROM_RESPONSE_SIZE.labels(method, path, status_code).observe(resp_size)

        # --- OTel ---
        attrs = {
            "http.method": method,
            "http.route": path,
            "http.status_code": int(status_code),
        }
        if _otel_request_counter:
            _otel_request_counter.add(1, attrs)
        if _otel_request_duration:
            _otel_request_duration.record(duration, attrs)

        current_span = trace.get_current_span()
        ctx = current_span.get_span_context()
        app.logger.info(
            "request completed",
            extra={
                "http.method": method,
                "http.path": request.path,
                "http.route": path,
                "http.status_code": response.status_code,
                "http.duration_s": round(duration, 4),
                "http.request_size_bytes": req_size,
                "http.response_size_bytes": resp_size,
                "http.remote_addr": request.remote_addr,
                "http.user_agent": request.user_agent.string,
                "trace_id": format(ctx.trace_id, "032x") if ctx.is_valid else "",
                "span_id": format(ctx.span_id, "016x") if ctx.is_valid else "",
            },
        )

        return response


def _normalise_path(path: str) -> str:
    """
    Collapse high-cardinality path segments to keep Prometheus label cardinality
    bounded. All frontend routes are fixed strings, so only two rules are needed:

      /static/...    → /static   (many filenames, one logical route)
      /admin/<x>     → /admin/<x> capped at two segments
      everything else → kept verbatim
    """
    parts = path.strip("/").split("/")
    if not parts or parts == [""]:
        return "/"
    if parts[0] == "static":
        return "/static"
    if len(parts) >= 2 and parts[0] == "admin":
        return "/" + "/".join(parts[:2])
    return path


# ---------------------------------------------------------------------------
# Business metric helpers — called from main.py
# ---------------------------------------------------------------------------


def record_backend_call(endpoint: str, status_code: int, duration_s: float) -> None:
    """
    Record the latency and outcome of an outbound call to the CRUD API.
    Called from the api_get / api_post / api_delete helpers in main.py.
    """
    status_str = str(status_code)
    PROM_BACKEND_DURATION.labels(endpoint, status_str).observe(duration_s)
    if _otel_backend_duration:
        _otel_backend_duration.record(
            duration_s,
            {"api.endpoint": endpoint, "http.status_code": status_code},
        )


def record_vote(vote_type: str) -> None:
    """
    Increment the votes-cast counter.
    vote_type must be "public" or "jury".
    """
    PROM_VOTES_CAST.labels(vote_type).inc()
    if _otel_votes_counter:
        _otel_votes_counter.add(1, {"vote.type": vote_type})


def set_active_sessions(count: int) -> None:
    """Update the active-sessions gauge. Called on login and logout."""
    PROM_ACTIVE_SESSIONS.set(count)



def _register_metrics_endpoint(app: Flask) -> None:
    @app.route("/metrics")
    def metrics_endpoint() -> Response:
        # When PROMETHEUS_MULTIPROC_DIR is set, we must collect from all
        # worker files rather than the in-process registry alone.
        if "PROMETHEUS_MULTIPROC_DIR" in os.environ:
            registry = CollectorRegistry()
            multiprocess.MultiProcessCollector(registry)
            data = generate_latest(registry)
        else:
            # Single-process mode (local dev without gunicorn).
            data = generate_latest(REGISTRY)
        return Response(data, status=200, mimetype=CONTENT_TYPE_LATEST)



def _register_health_endpoint(app: Flask) -> None:
    @app.route("/health")
    def health_endpoint() -> Response:
        return Response(
            '{"status":"healthy","service":"esc-voting-frontend"}',
            status=200,
            mimetype="application/json",
        )
