package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

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

// ---------------------------------------------------------------------------
// ESC points table
// ---------------------------------------------------------------------------

// escPointTable maps 0-indexed rank to ESC televote points.
// Rank 0 = 1st place (12 pts) … rank 9 = 10th place (1 pt), rank 10+ = 0 pts.
var escPointTable = []int{12, 10, 8, 7, 6, 5, 4, 3, 2, 1}

func escPointsForRank(rank int) int {
	if rank < len(escPointTable) {
		return escPointTable[rank]
	}
	return 0
}

// ---------------------------------------------------------------------------
// Domain types
// ---------------------------------------------------------------------------

type Song struct {
	ID        int    `json:"songId"`
	Name      string `json:"songName"`
	Country   string `json:"country"`
	LandID    string `json:"countryId"`
	RawVotes  int    `json:"rawPublicVotes"`
	ESCPoints int    `json:"escPoints"`
	Rank      int    `json:"rank"`
}

// ---------------------------------------------------------------------------
// Observability globals
// ---------------------------------------------------------------------------

var (
	logger *slog.Logger
	tracer trace.Tracer

	requestCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "esc_converter_http_requests_total",
			Help: "Total HTTP requests handled by the ESC points converter",
		},
		[]string{"method", "path", "status"},
	)

	requestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "esc_converter_http_request_duration_seconds",
			Help:    "HTTP request duration for the ESC points converter",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

)

// ---------------------------------------------------------------------------
// OTel setup — mirrors the CRUD API pattern
// ---------------------------------------------------------------------------

func otlpEndpointHostPort() string {
	raw := getEnv("OTEL_EXPORTER_OTLP_HTTP_ENDPOINT", "http://otel-collector:4318")
	if parsed, err := url.Parse(raw); err == nil && parsed.Host != "" {
		return parsed.Host
	}
	return raw
}

func initTracer() (*sdktrace.TracerProvider, error) {
	exporter, err := otlptracehttp.New(
		context.Background(),
		otlptracehttp.WithEndpoint(otlpEndpointHostPort()),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("create OTLP trace exporter: %w", err)
	}
	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceName("esc-points-converter"),
			semconv.ServiceVersion("1.0.0"),
			attribute.String("environment", getEnv("ENVIRONMENT", "production")),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create resource: %w", err)
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
		return nil, fmt.Errorf("create OTLP log exporter: %w", err)
	}
	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceName("esc-points-converter"),
			semconv.ServiceVersion("1.0.0"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create resource: %w", err)
	}
	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
		sdklog.WithResource(res),
	)
	global.SetLoggerProvider(lp)
	return lp, nil
}

// otelSlogHandler bridges slog records to the OTel log API so every log line
// is forwarded to the collector (→ Loki) in addition to being written to stdout.
type otelSlogHandler struct {
	inner  slog.Handler
	otelLg otellog.Logger
}

func newOtelSlogHandler(inner slog.Handler) *otelSlogHandler {
	return &otelSlogHandler{
		inner:  inner,
		otelLg: global.GetLoggerProvider().Logger("esc-points-converter"),
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
	if sc := trace.SpanFromContext(ctx).SpanContext(); sc.IsValid() {
		attrs = append(attrs,
			otellog.String("traceID", sc.TraceID().String()),
			otellog.String("spanID", sc.SpanID().String()),
		)
	}
	rec.AddAttributes(attrs...)
	h.otelLg.Emit(ctx, rec)
	return nil
}

func (h *otelSlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &otelSlogHandler{inner: h.inner.WithAttrs(attrs), otelLg: h.otelLg}
}
func (h *otelSlogHandler) WithGroup(name string) slog.Handler {
	return &otelSlogHandler{inner: h.inner.WithGroup(name), otelLg: h.otelLg}
}

// ---------------------------------------------------------------------------
// HTTP observability middleware
// ---------------------------------------------------------------------------

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (sw *statusWriter) WriteHeader(code int) {
	sw.status = code
	sw.ResponseWriter.WriteHeader(code)
}

func observabilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip tracing + metrics for health and metrics endpoints to avoid noise.
		if r.URL.Path == "/health" || r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		ctx, span := tracer.Start(r.Context(), fmt.Sprintf("%s %s", r.Method, r.URL.Path))
		defer span.End()

		span.SetAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.path", r.URL.Path),
			attribute.String("http.remote_addr", r.RemoteAddr),
		)

		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

		logger.InfoContext(ctx, "incoming request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("remote_addr", r.RemoteAddr),
			slog.String("trace_id", span.SpanContext().TraceID().String()),
		)

		next.ServeHTTP(sw, r.WithContext(ctx))

		dur := time.Since(start)
		statusStr := fmt.Sprintf("%d", sw.status)

		span.SetAttributes(
			attribute.Int("http.status_code", sw.status),
			attribute.Float64("http.duration_ms", float64(dur.Milliseconds())),
		)
		requestCounter.WithLabelValues(r.Method, r.URL.Path, statusStr).Inc()
		requestDuration.WithLabelValues(r.Method, r.URL.Path, statusStr).Observe(dur.Seconds())

		logger.InfoContext(ctx, "request completed",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", sw.status),
			slog.Duration("duration", dur),
		)
	})
}

// ---------------------------------------------------------------------------
// Business logic
// ---------------------------------------------------------------------------

func fetchSongs(ctx context.Context, db *sql.DB) ([]Song, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT s.ID, s.Name, l.Name, l.ID, s.PublikumsPunkte
		FROM Song s
		JOIN Land l ON s.Land_ID = l.ID
	`)
	if err != nil {
		return nil, fmt.Errorf("query songs: %w", err)
	}
	defer rows.Close()

	var songs []Song
	for rows.Next() {
		var s Song
		if err := rows.Scan(&s.ID, &s.Name, &s.Country, &s.LandID, &s.RawVotes); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		songs = append(songs, s)
	}
	return songs, rows.Err()
}

// rankAndConvert sorts songs by raw public votes (desc), breaks ties by song
// ID (asc) for determinism, then assigns ESC points based on final position.
func rankAndConvert(songs []Song) []Song {
	sort.SliceStable(songs, func(i, j int) bool {
		if songs[i].RawVotes != songs[j].RawVotes {
			return songs[i].RawVotes > songs[j].RawVotes
		}
		return songs[i].ID < songs[j].ID
	})
	for i := range songs {
		songs[i].Rank = i + 1
		songs[i].ESCPoints = escPointsForRank(i)
	}
	return songs
}

// ---------------------------------------------------------------------------
// HTTP handlers
// ---------------------------------------------------------------------------

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func handlePreview(db *sql.DB, juryScale int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		_, span := tracer.Start(ctx, "preview.fetchAndRank")
		defer span.End()

		songs, err := fetchSongs(ctx, db)
		if err != nil {
			logger.ErrorContext(ctx, "preview: failed to fetch songs", "error", err)
			span.RecordError(err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to fetch songs"})
			return
		}

		result := rankAndConvert(songs)
		for i := range result {
			result[i].ESCPoints *= juryScale
		}
		span.SetAttributes(attribute.Int("songs.count", len(result)))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"message": "ESC points preview (not yet applied)",
			"payload": result,
		})
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return fallback
}

func connectToDatabase() (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&timeout=10s&readTimeout=10s&writeTimeout=10s",
		getEnv("DB_USER", "esc_user"),
		url.QueryEscape(getEnv("DB_PASS", "esc_password")),
		getEnv("DB_HOST", "localhost"),
		getEnv("DB_PORT", "3306"),
		getEnv("DB_NAME", "esc_voting"),
	)

	const (
		maxAttempts = 60
		retryDelay  = 3 * time.Second
	)

	logger.Info("connecting to database",
		slog.String("host", getEnv("DB_HOST", "localhost")),
		slog.String("port", getEnv("DB_PORT", "3306")),
		slog.String("db", getEnv("DB_NAME", "esc_voting")),
	)

	var (
		conn *sql.DB
		err  error
	)
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		conn, err = sql.Open("mysql", dsn)
		if err == nil {
			conn.SetMaxOpenConns(25)
			conn.SetMaxIdleConns(5)
			conn.SetConnMaxLifetime(5 * time.Minute)

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			pingErr := conn.PingContext(ctx)
			cancel()

			if pingErr == nil {
				logger.Info("database connected", slog.Int("attempt", attempt))
				return conn, nil
			}
			err = pingErr
		}
		logger.Warn("database not ready, retrying",
			slog.Int("attempt", attempt),
			slog.Int("max", maxAttempts),
			"error", err,
		)
		if attempt < maxAttempts {
			time.Sleep(retryDelay * time.Duration(attempt))
		}
	}
	return nil, fmt.Errorf("could not connect after %d attempts: %w", maxAttempts, err)
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

func main() {
	// ── Logging bootstrap (stdout JSON before OTel is ready) ─────────────────
	baseHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger = slog.New(baseHandler)
	slog.SetDefault(logger)

	// ── OTel tracing ──────────────────────────────────────────────────────────
	tp, err := initTracer()
	if err != nil {
		log.Printf("warning: tracing unavailable: %v", err)
	} else {
		defer func() {
			if shutErr := tp.Shutdown(context.Background()); shutErr != nil {
				log.Printf("tracer shutdown error: %v", shutErr)
			}
		}()
	}

	// ── OTel logging (→ Loki) ─────────────────────────────────────────────────
	lp, err := initLogProvider()
	if err != nil {
		log.Printf("warning: OTLP log forwarding unavailable: %v", err)
	} else {
		defer func() {
			if shutErr := lp.Shutdown(context.Background()); shutErr != nil {
				log.Printf("log provider shutdown error: %v", shutErr)
			}
		}()
	}

	// Replace stdout-only logger with the OTel-bridged one.
	logger = slog.New(newOtelSlogHandler(baseHandler))
	slog.SetDefault(logger)

	tracer = otel.Tracer("esc-points-converter")

	// ── Database ──────────────────────────────────────────────────────────────
	db, err := connectToDatabase()
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("database connection established")

	// ── Routes ────────────────────────────────────────────────────────────────
	port := getEnv("PORT", "8090")

	// juryScale equalises the 50/50 jury vs televote weighting.
	// The public vote produces one ESC set (max 12 pts); multiplying by the
	// number of jury members brings it to the same maximum as the combined jury.
	juryScale := getEnvInt("NUM_JURY_MEMBERS", 3)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("GET /api/esc-points", handlePreview(db, juryScale))
	mux.Handle("GET /metrics", promhttp.Handler())

	logger.Info("ESC points converter starting",
		slog.String("port", port),
		slog.Int("jury_scale", juryScale),
		slog.String("otel_endpoint", getEnv("OTEL_EXPORTER_OTLP_HTTP_ENDPOINT", "http://otel-collector:4318")),
	)

	if err := http.ListenAndServe(":"+port, observabilityMiddleware(mux)); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}
