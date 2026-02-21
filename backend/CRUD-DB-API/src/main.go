package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

// Structured Logger
var (
	logger *slog.Logger

	//OTel Tracer
	tracer trace.Tracer

	//Prometheus metrics for request size
	requestSizeBytes = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_size_bytes",
			Help:    "Size of HTTP request bodies in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8), //100B to 100 MB
		},
		[]string{"method", "path"},
	)

	// Response size
	responseSizeBytes = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_response_size_bytes",
			Help:    "Size of HTTP response bodies in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "path", "status"},
	)

	// Request Duration
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

func initTracer() (*sdktrace.TracerProvider, error) {
	// Create OTLP exporter

	exporter, err := otlptracehttp.New(
		context.Background(),
		otlptracehttp.WithEndpoint("localhost:4318"), // Default OTLP Endpoint
		otlptracehttp.WithInsecure(),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
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
		sdktrace.WithSampler(sdktrace.AlwaysSample()), // Sample all Traces
	)

	otel.SetTracerProvider(tp)
	return tp, nil
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

		ctx, span := tracer.Start(r.Context(), fmt.Sprintf("%s %s", r.Method, r.URL.Path))
		defer span.End()

		span.SetAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.url", r.URL.String()),
			attribute.String("http.path", r.URL.Path),
			attribute.String("http.remote_addr", r.RemoteAddr),
			attribute.Int64("http.request_content_length", r.ContentLength),
		)

		// Wrap response writer to capture status and size
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Record request size metric

		if r.ContentLength > 0 {
			requestSizeBytes.WithLabelValues(r.Method, r.URL.Path).Observe(float64(r.ContentLength))
		}

		// Start rewuest
		logger.InfoContext(ctx, "incoming request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int64("content_length", r.ContentLength),
			slog.String("remoteaddr", r.RemoteAddr),
			slog.String("user_agent", r.UserAgent()),
			slog.String("trace_id", span.SpanContext().TraceID().String()),
			slog.String("span_id", span.SpanContext().SpanID().String()),
		)

		// Call the next handler with new Context
		next.ServeHTTP(rw, r.WithContext(ctx))

		duration := time.Since(startTime)

		// Update span with response info
		span.SetAttributes(
			attribute.Int("http.status_code", rw.statusCode),
			attribute.Int("http.response_size", rw.size),
			attribute.Float64("http.duration_ms", float64(duration.Milliseconds())),
		)

		// Record Metrics
		statusStr := fmt.Sprintf("%d", rw.statusCode)
		requestCounter.WithLabelValues(r.Method, r.URL.Path, statusStr).Inc()
		requestDuration.WithLabelValues(r.Method, r.URL.Path, statusStr).Observe(duration.Seconds())
		responseSizeBytes.WithLabelValues(r.Method, r.URL.Path, statusStr).Observe(float64(rw.size))

		// Request Complete
		logger.InfoContext(ctx, "request completed",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", rw.statusCode),
			slog.Int("response_size", rw.size),
			slog.Duration("duration", duration),
			slog.String("trace_id", span.SpanContext().TraceID().String()),
		)
	})
}

func getHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Success",
		"status":  "healthy",
	})

}

func main() {

	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	tp, err := initTracer()
	if err != nil {
		log.Printf("Warning: Failed to initialize tracer: %v. Continue without tracig", err)
	} else {
		defer func() {
			if err := tp.Shutdown(context.Background()); err != nil {
				log.Printf("Error shutting down tracer provider: %v", err)
			}
		}()
	}

	tracer = otel.Tracer("esc-voting-crud-api")
	log.Println("Listening and Serving on Port 8000")
	logger.Info("server starting",
		slog.Int("port", 8000),
		slog.String("service", "esc-voting-crud-api"),
	)

	router := http.NewServeMux()
	router.HandleFunc("GET /health", getHealth)
	router.Handle("GET /metrics", promhttp.Handler())

	handler := ObservabilityMiddleware(router)

	// Start Sercer
	err = http.ListenAndServe(":8000", handler)
	if err != nil {
		logger.Error("Server failed", slog.String("error", err.Error()))
		fmt.Println("Error Starting the Server")
	}
}
