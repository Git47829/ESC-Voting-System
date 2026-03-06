import logging
import os

from fastapi import FastAPI
from opentelemetry import metrics, trace
from opentelemetry._logs import set_logger_provider
from opentelemetry.exporter.otlp.proto.grpc._log_exporter import OTLPLogExporter
from opentelemetry.exporter.otlp.proto.grpc.metric_exporter import OTLPMetricExporter
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.instrumentation.fastapi import FastAPIInstrumentor
from opentelemetry.instrumentation.grpc import GrpcAioInstrumentorClient
from opentelemetry.sdk._logs import LoggerProvider, LoggingHandler
from opentelemetry.sdk._logs.export import BatchLogRecordProcessor
from opentelemetry.sdk.metrics import MeterProvider
from opentelemetry.sdk.metrics.export import PeriodicExportingMetricReader
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor

logger = logging.getLogger(__name__)


def setup_telemetry(app: FastAPI):
    """
    Configure OpenTelemetry tracing, metrics, and logging for the EuroStats service.
    Exports all telemetry to the OpenTelemetry Collector via gRPC (port 4317).

    Pillars
    -------
    1. Traces  — OTel SDK TracerProvider  → OTLP/gRPC → otel-collector → Tempo
    2. Metrics — OTel SDK MeterProvider   → OTLP/gRPC → otel-collector → Prometheus
                 FastAPIInstrumentor emits http.server.duration (ms) and
                 http.server.request.count (old semconv by default), which the
                 collector namespaces to esc_http_server_duration_milliseconds and
                 esc_http_server_request_count_total respectively.
                 GrpcAioInstrumentorClient emits rpc.client.duration (ms), namespaced
                 to esc_rpc_client_duration_milliseconds.
    3. Logs    — Python logging → OTel LoggingHandler → OTLP/gRPC → otel-collector → Loki
    """
    try:
        resource = Resource.create(
            {"service.name": "eurostats", "service.version": "1.0.0"}
        )

        otlp_endpoint = os.getenv(
            "OTEL_EXPORTER_OTLP_ENDPOINT", "http://otel-collector:4317"
        )

        # ------------------------------------------------------------------ #
        # Traces
        # ------------------------------------------------------------------ #
        tracer_provider = TracerProvider(resource=resource)
        span_exporter = OTLPSpanExporter(endpoint=otlp_endpoint, insecure=True)
        tracer_provider.add_span_processor(BatchSpanProcessor(span_exporter))
        trace.set_tracer_provider(tracer_provider)

        # ------------------------------------------------------------------ #
        # Metrics
        # ------------------------------------------------------------------ #
        metric_exporter = OTLPMetricExporter(endpoint=otlp_endpoint, insecure=True)
        metric_reader = PeriodicExportingMetricReader(
            metric_exporter, export_interval_millis=15_000
        )
        meter_provider = MeterProvider(
            resource=resource, metric_readers=[metric_reader]
        )
        metrics.set_meter_provider(meter_provider)

        # ------------------------------------------------------------------ #
        # Logs  — attach OTel handler to the root logger so that every
        # logging.getLogger(...) call in the service flows to Loki via the
        # collector.  The handler is added at WARNING level by default;
        # lower it to INFO so structured request logs are captured.
        # ------------------------------------------------------------------ #
        logger_provider = LoggerProvider(resource=resource)
        log_exporter = OTLPLogExporter(endpoint=otlp_endpoint, insecure=True)
        logger_provider.add_log_record_processor(BatchLogRecordProcessor(log_exporter))
        set_logger_provider(logger_provider)

        otel_log_handler = LoggingHandler(
            level=logging.INFO, logger_provider=logger_provider
        )
        # Attach to the root logger so all loggers in the process inherit it.
        logging.getLogger().addHandler(otel_log_handler)

        # ------------------------------------------------------------------ #
        # Instrumentation
        # ------------------------------------------------------------------ #

        # FastAPI / ASGI — emits http.server.* metrics (old semconv default)
        FastAPIInstrumentor.instrument_app(app)

        # gRPC async client — must use GrpcAioInstrumentorClient, NOT
        # GrpcInstrumentorClient, because the channel is grpc.aio (asyncio).
        # GrpcInstrumentorClient only patches the synchronous grpc stubs and
        # would silently produce no metrics or traces for grpc.aio calls.
        GrpcAioInstrumentorClient().instrument()

        logger.info(
            "EuroStats telemetry setup complete. Exporting to %s", otlp_endpoint
        )

    except Exception as e:
        logger.error("Failed to setup telemetry: %s", e)
