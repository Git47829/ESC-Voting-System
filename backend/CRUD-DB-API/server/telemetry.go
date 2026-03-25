package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otlploghttp "go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	Logger *slog.Logger

	Tracer trace.Tracer

	requestSizeBytes = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_size_bytes",
			Help:    "Size of HTTP request bodies in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8), //100B to 100 MB
		},
		[]string{"method", "path"},
	)

	responseSizeBytes = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_response_size_bytes",
			Help:    "Size of HTTP response bodies in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "path", "status"},
	)

	requestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

	requestCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)
)

func otlpEndpointHostPort() string {
	otlpEndpoint := getEnvOrDefault("OTEL_EXPORTER_OTLP_HTTP_ENDPOINT", "localhost:4318")
	if parsed, err := url.Parse(otlpEndpoint); err == nil && parsed.Host != "" {
		return parsed.Host
	}
	return otlpEndpoint
}

func initTracer() (*sdktrace.TracerProvider, error) {
	exporter, err := otlptracehttp.New(
		context.Background(),
		otlptracehttp.WithEndpoint(otlpEndpointHostPort()),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP trace exporter: %w", err)
	}

	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceName("esc-voting-crud-api"),
			semconv.ServiceVersion("0.1.0"),
			attribute.String("environment", "development"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(tp)
	return tp, nil
}

func initLogProvider() (*sdklog.LoggerProvider, error) {
	exporter, err := otlploghttp.New(
		context.Background(),
		otlploghttp.WithEndpoint(otlpEndpointHostPort()),
		otlploghttp.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP log exporter: %w", err)
	}

	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceName("esc-voting-crud-api"),
			semconv.ServiceVersion("0.1.0"),
			attribute.String("environment", "development"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
		sdklog.WithResource(res),
	)

	global.SetLoggerProvider(lp)
	return lp, nil
}

type otelSlogHandler struct {
	inner  slog.Handler
	logger otellog.Logger
}

func newOtelSlogHandler(inner slog.Handler) *otelSlogHandler {
	return &otelSlogHandler{
		inner:  inner,
		logger: global.GetLoggerProvider().Logger("esc-voting-crud-api"),
	}
}

func (h *otelSlogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *otelSlogHandler) Handle(ctx context.Context, r slog.Record) error {
	if err := h.inner.Handle(ctx, r); err != nil {
		return err
	}

	var rec otellog.Record
	rec.SetTimestamp(r.Time)
	rec.SetBody(otellog.StringValue(r.Message))

	switch {
	case r.Level >= slog.LevelError:
		rec.SetSeverity(otellog.SeverityError)
		rec.SetSeverityText("ERROR")
	case r.Level >= slog.LevelWarn:
		rec.SetSeverity(otellog.SeverityWarn)
		rec.SetSeverityText("WARN")
	case r.Level >= slog.LevelInfo:
		rec.SetSeverity(otellog.SeverityInfo)
		rec.SetSeverityText("INFO")
	default:
		rec.SetSeverity(otellog.SeverityDebug)
		rec.SetSeverityText("DEBUG")
	}

	attrs := make([]otellog.KeyValue, 0, r.NumAttrs()+2)
	r.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs, otellog.String(a.Key, a.Value.String()))
		return true
	})

	rec.AddAttributes(attrs...)

	h.logger.Emit(ctx, rec)

	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		spanAttrs := make([]attribute.KeyValue, 0, r.NumAttrs()+1)
		spanAttrs = append(spanAttrs, attribute.String("log.severity", r.Level.String()))
		r.Attrs(func(a slog.Attr) bool {
			spanAttrs = append(spanAttrs, attribute.String(a.Key, a.Value.String()))
			return true
		})
		span.AddEvent(r.Message, trace.WithAttributes(spanAttrs...))
	}

	return nil
}

func (h *otelSlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &otelSlogHandler{inner: h.inner.WithAttrs(attrs), logger: h.logger}
}

func (h *otelSlogHandler) WithGroup(name string) slog.Handler {
	return &otelSlogHandler{inner: h.inner.WithGroup(name), logger: h.logger}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

func ObservabilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		ctx, span := Tracer.Start(r.Context(), fmt.Sprintf("%s %s", r.Method, r.URL.Path))
		defer span.End()

		span.SetAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.url", r.URL.String()),
			attribute.String("http.path", r.URL.Path),
			attribute.String("http.remote_addr", r.RemoteAddr),
			attribute.Int64("http.request_content_length", r.ContentLength),
		)

		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		if r.ContentLength > 0 {
			requestSizeBytes.WithLabelValues(r.Method, r.URL.Path).Observe(float64(r.ContentLength))
		}

		// Start request
		Logger.InfoContext(ctx, "incoming request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int64("content_length", r.ContentLength),
			slog.String("remoteaddr", r.RemoteAddr),
			slog.String("user_agent", r.UserAgent()),
			slog.String("trace_id", span.SpanContext().TraceID().String()),
			slog.String("span_id", span.SpanContext().SpanID().String()),
		)

		next.ServeHTTP(rw, r.WithContext(ctx))

		duration := time.Since(startTime)

		span.SetAttributes(
			attribute.Int("http.status_code", rw.statusCode),
			attribute.Int("http.response_size", rw.size),
			attribute.Float64("http.duration_ms", float64(duration.Milliseconds())),
		)

		statusStr := fmt.Sprintf("%d", rw.statusCode)
		requestCounter.WithLabelValues(r.Method, r.URL.Path, statusStr).Inc()
		requestDuration.WithLabelValues(r.Method, r.URL.Path, statusStr).Observe(duration.Seconds())
		responseSizeBytes.WithLabelValues(r.Method, r.URL.Path, statusStr).Observe(float64(rw.size))

		Logger.InfoContext(ctx, "request completed",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", rw.statusCode),
			slog.Int("response_size", rw.size),
			slog.Duration("duration", duration),
			slog.String("trace_id", span.SpanContext().TraceID().String()),
		)
	})
}
