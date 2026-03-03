import logging
import os

from fastapi import FastAPI
from opentelemetry import metrics, trace
from opentelemetry.exporter.otlp.proto.grpc.metric_exporter import OTLPMetricExporter
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.instrumentation.fastapi import FastAPIInstrumentor
from opentelemetry.instrumentation.grpc import GrpcInstrumentorClient
from opentelemetry.sdk.metrics import MeterProvider
from opentelemetry.sdk.metrics.export import PeriodicExportingMetricReader
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor

logger = logging.getLogger(__name__)


def setup_telemetry(app: FastAPI):
    """
    Configure OpenTelemetry tracing and metrics for the EuroStats service.
    Exports telemetry data to the OpenTelemetry Collector via gRPC.
    """
    try:
        # Define the service resource
        resource = Resource.create(
            {"service.name": "eurostats", "service.version": "1.0.0"}
        )

        # OTLP Collector endpoint (defaulting to the container network name)
        otlp_endpoint = os.getenv(
            "OTEL_EXPORTER_OTLP_ENDPOINT", "http://otel-collector:4317"
        )

        # --- Setup Tracing ---
        tracer_provider = TracerProvider(resource=resource)
        span_exporter = OTLPSpanExporter(endpoint=otlp_endpoint, insecure=True)
        tracer_provider.add_span_processor(BatchSpanProcessor(span_exporter))
        trace.set_tracer_provider(tracer_provider)

        # --- Setup Metrics ---
        metric_exporter = OTLPMetricExporter(endpoint=otlp_endpoint, insecure=True)
        metric_reader = PeriodicExportingMetricReader(metric_exporter)
        meter_provider = MeterProvider(
            resource=resource, metric_readers=[metric_reader]
        )
        metrics.set_meter_provider(meter_provider)

        # --- Instrumentations ---
        # Instrument FastAPI to trace HTTP and WebSocket requests
        FastAPIInstrumentor.instrument_app(app)

        # Instrument gRPC Client to trace outgoing requests to the CRUD API
        GrpcInstrumentorClient().instrument()

        logger.info(f"Telemetry setup complete. Exporting to {otlp_endpoint}")

    except Exception as e:
        logger.error(f"Failed to setup telemetry: {e}")
