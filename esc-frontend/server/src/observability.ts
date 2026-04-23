import { NodeSDK } from "@opentelemetry/sdk-node";
import { getNodeAutoInstrumentations } from "@opentelemetry/auto-instrumentations-node";
import { OTLPTraceExporter } from "@opentelemetry/exporter-trace-otlp-grpc";
import { OTLPMetricExporter } from "@opentelemetry/exporter-metrics-otlp-http";
import { OTLPLogExporter } from "@opentelemetry/exporter-logs-otlp-http";
import { PeriodicExportingMetricReader } from "@opentelemetry/sdk-metrics";
import { BatchLogRecordProcessor } from "@opentelemetry/sdk-logs";
import { resourceFromAttributes } from "@opentelemetry/resources";
import { SemanticResourceAttributes } from "@opentelemetry/semantic-conventions";

const getExporterEndpoint = (): string => {
  return process.env.OTEL_EXPORTER_OTLP_ENDPOINT ?? "http://localhost:4318";
};

const resource = resourceFromAttributes({
  [SemanticResourceAttributes.SERVICE_NAME]: "esc-frontend",
  [SemanticResourceAttributes.SERVICE_VERSION]: "1.0.0"
});

const sdk = new NodeSDK({
  resource,
  traceExporter: new OTLPTraceExporter({
    url: `${getExporterEndpoint()}/v1/traces`
  }),
  metricReader: new PeriodicExportingMetricReader({
    exporter: new OTLPMetricExporter({
      url: `${getExporterEndpoint()}/v1/metrics`
    })
  }),
  logRecordProcessor: new BatchLogRecordProcessor(
    new OTLPLogExporter({
      url: `${getExporterEndpoint()}/v1/logs`
    })
  ),
  instrumentations: [getNodeAutoInstrumentations()]
});

const startSdk = async (): Promise<void> => {
  try {
    await sdk.start();
    // eslint-disable-next-line no-console
    console.log("OpenTelemetry SDK initialized successfully");
  } catch (error) {
    // eslint-disable-next-line no-console
    console.error("Failed to initialize OpenTelemetry SDK:", error);
  }
};

void startSdk();

const shutdownSdk = async (): Promise<void> => {
  try {
    await sdk.shutdown();
    // eslint-disable-next-line no-console
    console.log("OpenTelemetry SDK shut down gracefully");
  } catch (error) {
    // eslint-disable-next-line no-console
    console.error("Failed to shutdown OpenTelemetry SDK:", error);
  }
};

process.on("SIGTERM", () => {
  void shutdownSdk();
});

export { sdk };
