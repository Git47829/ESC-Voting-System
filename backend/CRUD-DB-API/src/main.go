package main

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
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
	"golang.org/x/time/rate"

	"github.com/nyaruka/phonenumbers"
)

type Client struct {
	limiter *rate.Limiter
}

type RateLimitConfig struct {
	RequestsPerSecond float64
	BurstSize         int
}

type LocalConfig struct {
	DbHost string `env:"dbHost"`
	DbName string `env:"dbName"`
	DbUser string `env:"dbUser"`
	DbPass string `env:"dbPass"`
	DbPort int    `env:"dbPort"`
}

func loadLocalConfig() LocalConfig {
	return LocalConfig{
		DbHost: getEnvOrDefault("DB_HOST", "localhost"),
		DbName: getEnvOrDefault("DB_NAME", "esc_voting"),
		DbUser: getEnvOrDefault("DB_USER", "root"),
		DbPass: getEnvOrDefault("DB_PASS", ""),
		DbPort: getEnvOrDefaultInt("DB_PORT", 3306),
	}
}

func getEnvOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvOrDefaultInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return fallback
}

// Singleton Pattern
var db *sql.DB

var clients = make(map[string]*Client)

var (
	rateLimitConfigs = map[string]RateLimitConfig{
		"GET /health":     {RequestsPerSecond: 100, BurstSize: 100},
		"GET /votes/":     {RequestsPerSecond: 10, BurstSize: 20},
		"GET /countries/": {RequestsPerSecond: 10, BurstSize: 20},
		"GET /songs/":     {RequestsPerSecond: 10, BurstSize: 20},

		// User voting - strict limits
		"POST /vote/":     {RequestsPerSecond: 1, BurstSize: 1},
		"POST /jury/vote": {RequestsPerSecond: 5, BurstSize: 5},

		// Admin endpoints - moderate limits
		"POST /admin/open/":        {RequestsPerSecond: 2, BurstSize: 2},
		"POST /admin/close":        {RequestsPerSecond: 2, BurstSize: 2},
		"POST /admin/addCountry":   {RequestsPerSecond: 5, BurstSize: 5},
		"POST /admin/addSong":      {RequestsPerSecond: 5, BurstSize: 5},
		"POST /admin/addArtist":    {RequestsPerSecond: 5, BurstSize: 5},
		"POST /admin/addInterpret": {RequestsPerSecond: 5, BurstSize: 5},
		"DELETE /admin/delteVotes": {RequestsPerSecond: 1, BurstSize: 1},

		// Metrics - unlimited
		"GET /metrics/": {RequestsPerSecond: 10000, BurstSize: 10000},
	}
)

// Token Store (kept for legacy cookie invalidation checks)
var (
	usedTokens = make(map[string]bool)
	mu         sync.Mutex
)

const totalVotePoints = 20

// cookieVoteState is the structure stored in the vote cookie.
type cookieVoteState struct {
	VotesRemaining int            `json:"votes_remaining"`
	VotesCast      map[string]int `json:"votes_cast"` // songID (string) -> points awarded
}

// signedCookieSecret is derived from the adminPassword env var at startup.
// Falls back to a random per-process secret so the service still works without one.
var signedCookieSecret []byte

func initCookieSecret() {
	if pw := os.Getenv("adminPassword"); pw != "" {
		sum := sha256.Sum256([]byte("cookie-secret:" + pw))
		signedCookieSecret = sum[:]
		return
	}
	signedCookieSecret = make([]byte, 32)
	if _, err := rand.Read(signedCookieSecret); err != nil {
		panic("cannot generate cookie secret: " + err.Error())
	}
}

// encodeCookieValue serialises a cookieVoteState and appends an HMAC-SHA256
// signature so the client cannot tamper with it.
func encodeCookieValue(state cookieVoteState) (string, error) {
	data, err := json.Marshal(state)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, signedCookieSecret)
	mac.Write(data)
	sig := mac.Sum(nil)
	payload := append(data, '.')
	payload = append(payload, []byte(hex.EncodeToString(sig))...)
	return hex.EncodeToString(payload), nil
}

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

func otlpEndpointHostPort() string {
	otlpEndpoint := getEnvOrDefault("OTEL_EXPORTER_OTLP_HTTP_ENDPOINT", "localhost:4318")
	// WithEndpoint expects only host:port, so strip any http:// or https:// scheme
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

// otelSlogHandler wraps another slog.Handler and additionally emits each log
// record as an OTLP log, embedding the active trace/span IDs so that Loki
// can correlate logs with Tempo traces via the traceID label.
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
	// Emit to the original handler (stdout) first.
	if err := h.inner.Handle(ctx, r); err != nil {
		return err
	}

	// Build an OTel log record.
	var rec otellog.Record
	rec.SetTimestamp(r.Time)
	rec.SetBody(otellog.StringValue(r.Message))

	// Map slog levels to OTel severity.
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

	// Carry all slog attributes across as OTel log attributes.
	attrs := make([]otellog.KeyValue, 0, r.NumAttrs()+2)
	r.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs, otellog.String(a.Key, a.Value.String()))
		return true
	})

	// Explicitly attach traceID and spanID as log *attributes* (in addition
	// to the OTel SDK automatically setting them as standard OTLP trace
	// context fields from ctx). The Loki exporter in the OTel collector
	// promotes attributes listed under loki.attribute.labels as Loki stream
	// labels, which is what enables Grafana's tracesToLogsV2 filterByTraceID
	// link to perform an efficient label-based query instead of a full scan.
	if spanCtx := trace.SpanFromContext(ctx).SpanContext(); spanCtx.IsValid() {
		attrs = append(attrs,
			otellog.String("traceID", spanCtx.TraceID().String()),
			otellog.String("spanID", spanCtx.SpanID().String()),
		)
	}

	rec.AddAttributes(attrs...)

	h.logger.Emit(ctx, rec)
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

// Chain of Responsibility Design Pattern
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

func getCLientLimiter(ip string, endpoint string) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	key := ip + "::" + endpoint
	if client, exists := clients[key]; exists {
		return client.limiter
	}

	config, exists := rateLimitConfigs[endpoint]
	if !exists {
		config = RateLimitConfig{RequestsPerSecond: 10, BurstSize: 20}
	}

	limiter := rate.NewLimiter(rate.Limit(config.RequestsPerSecond), config.BurstSize)
	clients[key] = &Client{limiter: limiter}
	return limiter
}

func RateLimitingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		endpoint := fmt.Sprintf("%s %s", r.Method, r.URL.Path)
		clientIP := r.RemoteAddr

		limiter := getCLientLimiter(clientIP, endpoint)

		if !limiter.Allow() {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Rate limit exceeded",
			})

			logger.WarnContext(r.Context(), "rate limit exeeded",
				slog.String("ip", clientIP),
				slog.String("endpoint", endpoint),
			)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// alpha3ToAlpha2 maps ISO 3166-1 alpha-3 codes (used as primary keys in the
// Land table) to the alpha-2 codes expected by the phonenumbers library.
var alpha3ToAlpha2 = map[string]string{
	"ALB": "AL", // Albania
	"ARM": "AM", // Armenia
	"AUT": "AT", // Austria
	"AZE": "AZ", // Azerbaijan
	"BIH": "BA", // Bosnia and Herzegovina
	"BEL": "BE", // Belgium
	"BGR": "BG", // Bulgaria
	"BLR": "BY", // Belarus
	"CHE": "CH", // Switzerland
	"CYP": "CY", // Cyprus
	"CZE": "CZ", // Czech Republic
	"DEU": "DE", // Germany
	"DNK": "DK", // Denmark
	"EST": "EE", // Estonia
	"ESP": "ES", // Spain
	"FIN": "FI", // Finland
	"FRA": "FR", // France
	"GBR": "GB", // United Kingdom
	"GEO": "GE", // Georgia
	"GRC": "GR", // Greece
	"HRV": "HR", // Croatia
	"HUN": "HU", // Hungary
	"IRL": "IE", // Ireland
	"ISR": "IL", // Israel
	"ISL": "IS", // Iceland
	"ITA": "IT", // Italy
	"LTU": "LT", // Lithuania
	"LUX": "LU", // Luxembourg
	"LVA": "LV", // Latvia
	"MCO": "MC", // Monaco
	"MDA": "MD", // Moldova
	"MNE": "ME", // Montenegro
	"MKD": "MK", // North Macedonia
	"MLT": "MT", // Malta
	"NLD": "NL", // Netherlands
	"NOR": "NO", // Norway
	"POL": "PL", // Poland
	"PRT": "PT", // Portugal
	"ROU": "RO", // Romania
	"SRB": "RS", // Serbia
	"RUS": "RU", // Russia
	"SWE": "SE", // Sweden
	"SVN": "SI", // Slovenia
	"SVK": "SK", // Slovakia
	"SMR": "SM", // San Marino
	"TUR": "TR", // Turkey
	"UKR": "UA", // Ukraine
}

// checkPhoneNum validates a phone number and returns the alpha-3 country code
// it belongs to (as used in the Land table), so the caller can enforce voting rules.
//
// Returns:
//   - (alpha3, nil)   — number is valid and from a European / ESC-participating country
//   - ("",    nil)    — number is structurally invalid
//   - ("",    error)  — number is valid but not from a European / ESC-participating country
func checkPhoneNum(num string) (string, error) {
	// Parse without a region hint so the region is derived purely from the
	// number itself (requires E.164 or an explicit country prefix).
	parsed, err := phonenumbers.Parse(num, "")
	if err != nil {
		// Could not be parsed at all — not a valid phone number.
		return "", nil
	}

	// Check 1 — is the number itself structurally valid?
	if !phonenumbers.IsValidNumber(parsed) {
		return "", nil
	}

	// Check 2 — is the number from a European / ESC-participating country?
	numRegion := phonenumbers.GetRegionCodeForNumber(parsed)
	alpha3 := regionAlpha2ToAlpha3(numRegion)
	if alpha3 == "" {
		return "", fmt.Errorf("phone number is not from a valid European / ESC-participating country (region: %q)", numRegion)
	}

	return alpha3, nil
}

// regionAlpha2ToAlpha3 reverses alpha3ToAlpha2 for a single lookup.
// Returns an empty string if the alpha-2 code is not in the ESC map.
func regionAlpha2ToAlpha3(alpha2 string) string {
	for a3, a2 := range alpha3ToAlpha2 {
		if a2 == alpha2 {
			return a3
		}
	}
	return ""
}

func hashPassword(password string) (string, error) {
	sum := sha256.Sum256([]byte(password))
	return fmt.Sprintf("%x", sum), nil
}

func checkPassword(password, storedToken string) bool {
	return subtle.ConstantTimeCompare([]byte(password), []byte(storedToken)) == 1
}

func checkAccessAdmin(input string) (bool, string) {

	adminPassword := os.Getenv("adminPassword")

	if input == "" {
		return false, "Token has to be provided"
	}
	if checkPassword(input, adminPassword) {
		return true, "Authorized"
	}

	return false, "Wrong Token provided"
}

func checkAccessJury(input string) (bool, string) {

	juryPassword1 := os.Getenv("juryPassword1")
	juryPassword2 := os.Getenv("juryPassword2")
	juryPassword3 := os.Getenv("juryPassword3")

	TestToken := []string{juryPassword1, juryPassword2, juryPassword3}

	if input == "" {
		return false, "Token has to be provided"
	}

	for _, token := range TestToken {
		if checkPassword(input, token) {
			return true, "Authorized"
		}
	}

	return false, "Wrong Token Provided"
}

func generateToken() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	return hex.EncodeToString(b), err
}

func getHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Success",
		"status":  "healthy",
	})

}
func getVotes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	query := `
		SELECT
			s.ID,
			s.Name,
			l.Name as Country,
			s.PublikumsPunkte,
			s.JuryPunkte,
			s.GesamtPunkte
		FROM Song s
		JOIN Land l on s.Land_ID = l.ID
		ORDER BY s.GesamtPunkte DESC
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		logger.ErrorContext(ctx, "failed to query votes", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to retrieve Votes",
		})
		return
	}
	defer rows.Close()

	type SongVote struct {
		ID              int    `json:"id"`
		Name            string `json:"name"`
		Country         string `json:"country"`
		PublikumsPunkte int    `json:"publicVotes"`
		JuryPunkte      int    `json:"juryVotes"`
		GesamtPunkte    int    `json:"totalVotes"`
	}

	var votes []SongVote
	for rows.Next() {
		var v SongVote
		if err := rows.Scan(&v.ID, &v.Name, &v.Country, &v.PublikumsPunkte, &v.JuryPunkte, &v.GesamtPunkte); err != nil {
			logger.ErrorContext(ctx, "failed to scan row", slog.Any("error", err))
			continue
		}
		votes = append(votes, v)
	}

	if err := rows.Err(); err != nil {
		logger.ErrorContext(ctx, "rows iteration error", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to process votes",
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message": "Success",
		"payload": votes,
	})
}

func vote(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	const cookieName = "vote_state"

	ownCountry := r.URL.Query().Get("ownCountry")
	phone := r.URL.Query().Get("phoneNum")

	// --- Validate phone number ---
	phoneCountry, phoneErr := checkPhoneNum(phone)
	if phoneErr != nil {
		logger.WarnContext(ctx, "phone number failed validation", slog.String("phone", phone), slog.Any("error", phoneErr))
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": phoneErr.Error()})
		return
	}
	if phoneCountry == "" {
		logger.WarnContext(ctx, "invalid phone number provided", slog.String("phone", phone))
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid phone number"})
		return
	}

	phoneHash, _ := hashPassword(phone)

	// --- Parse songID ---
	rawID := r.URL.Query().Get("songID")
	songID, err := strconv.Atoi(rawID)
	if err != nil {
		logger.ErrorContext(ctx, "songID must be an integer", slog.Any("error", err), slog.String("songID", rawID))
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid songID value - must be an integer"})
		return
	}

	// --- Parse points ---
	rawPoints := r.URL.Query().Get("points")
	points, err := strconv.Atoi(rawPoints)
	if err != nil || points < 1 || points > totalVotePoints {
		logger.WarnContext(ctx, "invalid points value", slog.String("points", rawPoints))
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("points must be an integer between 1 and %d", totalVotePoints),
		})
		return
	}

	// --- Check voting is open ---
	type voteStatus struct {
		VotingID   int    `json:"votingId"`
		IsOpen     bool   `json:"isOpen"`
		LastChange string `json:"lastChange,omitempty"`
	}
	statusRows, err := db.QueryContext(ctx, `SELECT VotingID, isOpen, lastChange FROM Voting_Status`)
	if err != nil {
		logger.ErrorContext(ctx, "failed to query voting status", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to query voting status"})
		return
	}
	defer statusRows.Close()

	for statusRows.Next() {
		var v voteStatus
		var lastChange sql.NullString
		if scanErr := statusRows.Scan(&v.VotingID, &v.IsOpen, &lastChange); scanErr != nil {
			logger.ErrorContext(ctx, "failed to scan voting status row", slog.Any("error", scanErr))
			continue
		}
		if lastChange.Valid {
			v.LastChange = lastChange.String
		}
		if !v.IsOpen {
			w.WriteHeader(http.StatusTooEarly)
			json.NewEncoder(w).Encode(map[string]any{
				"error":   "voting has not started yet, please try again in a few minutes",
				"payload": v,
			})
			return
		}
	}
	if statusErr := statusRows.Err(); statusErr != nil {
		logger.ErrorContext(ctx, "voting status rows iteration error", slog.Any("error", statusErr))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to process voting status"})
		return
	}

	// --- Verify song exists and is not from the voter's own country ---
	type voteVerify struct {
		SongID int    `json:"songId"`
		LandID string `json:"landId"`
	}
	songRows, dbErr := db.QueryContext(ctx, `SELECT ID, Land_ID FROM Song WHERE ID = ?`, songID)
	if dbErr != nil {
		logger.ErrorContext(ctx, "error querying song", slog.Any("error", dbErr), slog.Int("songID", songID))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "error querying DB"})
		return
	}
	defer songRows.Close()

	var songFound bool
	for songRows.Next() {
		var v voteVerify
		if scanErr := songRows.Scan(&v.SongID, &v.LandID); scanErr != nil {
			logger.ErrorContext(ctx, "failed to scan song row", slog.Any("error", scanErr))
			continue
		}
		songFound = true
		if v.LandID == phoneCountry {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{
				"error":   "cannot vote for your own country",
				"payload": v,
			})
			return
		}
	}
	if songErr := songRows.Err(); songErr != nil {
		logger.ErrorContext(ctx, "song rows iteration error", slog.Any("error", songErr))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to process song details"})
		return
	}
	if !songFound {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Song not found"})
		return
	}

	// --- Upsert phone number row; initialise with full point budget on first insert ---
	// ON DUPLICATE KEY UPDATE with votes_remaining unchanged keeps the existing value.
	upsertQuery := `
		INSERT INTO Phone_Nums (Phone_Number, votes_remaining, votes_cast)
		VALUES (?, ?, JSON_OBJECT())
		ON DUPLICATE KEY UPDATE Phone_Number = Phone_Number`
	if _, upsertErr := db.ExecContext(ctx, upsertQuery, phoneHash, totalVotePoints); upsertErr != nil {
		logger.ErrorContext(ctx, "failed to upsert phone number", slog.Any("error", upsertErr))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to record vote"})
		return
	}

	// --- Atomically deduct points if enough remain ---
	// We use a conditional UPDATE that only applies when votes_remaining >= points.
	deductQuery := `
		UPDATE Phone_Nums
		SET
			votes_remaining = votes_remaining - ?,
			votes_cast = JSON_SET(
				COALESCE(votes_cast, JSON_OBJECT()),
				CONCAT('$.', ?),
				COALESCE(JSON_EXTRACT(votes_cast, CONCAT('$.', ?)), 0) + ?
			)
		WHERE Phone_Number = ? AND votes_remaining >= ?`
	deductResult, deductErr := db.ExecContext(ctx, deductQuery,
		points,    // subtract
		rawID,     // JSON key (song ID as string)
		rawID,     // JSON key for existing value lookup
		points,    // add to existing tally for this song
		phoneHash, // WHERE clause
		points,    // votes_remaining >= points guard
	)
	if deductErr != nil {
		logger.ErrorContext(ctx, "failed to deduct vote points", slog.Any("error", deductErr))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to record vote"})
		return
	}

	rowsAffected, _ := deductResult.RowsAffected()
	if rowsAffected == 0 {
		// Either already at 0 or points requested exceed the remaining budget.
		var remaining int
		db.QueryRowContext(ctx,
			`SELECT votes_remaining FROM Phone_Nums WHERE Phone_Number = ?`, phoneHash,
		).Scan(&remaining)

		logger.WarnContext(ctx, "insufficient vote points",
			slog.String("phone_hash", phoneHash),
			slog.Int("requested", points),
			slog.Int("remaining", remaining),
		)
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]any{
			"error":           fmt.Sprintf("not enough vote points remaining (have %d, requested %d)", remaining, points),
			"votes_remaining": remaining,
		})
		return
	}

	// --- Read back the updated state from DB ---
	var remaining int
	var votesCastJSON sql.NullString
	db.QueryRowContext(ctx,
		`SELECT votes_remaining, votes_cast FROM Phone_Nums WHERE Phone_Number = ?`, phoneHash,
	).Scan(&remaining, &votesCastJSON)

	votesCast := make(map[string]int)
	if votesCastJSON.Valid && votesCastJSON.String != "" {
		json.Unmarshal([]byte(votesCastJSON.String), &votesCast)
	}

	// --- Update the vote state cookie ---
	state := cookieVoteState{
		VotesRemaining: remaining,
		VotesCast:      votesCast,
	}
	cookieVal, encErr := encodeCookieValue(state)
	if encErr != nil {
		logger.ErrorContext(ctx, "failed to encode vote cookie", slog.Any("error", encErr))
	} else {
		http.SetCookie(w, &http.Cookie{
			Name:     cookieName,
			Value:    cookieVal,
			Expires:  time.Now().Add(365 * 24 * time.Hour),
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
			Path:     "/",
		})
	}

	// --- Apply the public vote weight to the Song table ---
	const weightPublicVote = 1 // 1 point in Song table per vote point spent
	updateSong := `UPDATE Song SET PublikumsPunkte = PublikumsPunkte + ? WHERE ID = ?`
	voteResult, voteErr := db.ExecContext(ctx, updateSong, points*weightPublicVote, songID)
	if voteErr != nil {
		logger.ErrorContext(ctx, "failed to update song votes", slog.Any("error", voteErr))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to record vote"})
		return
	}
	if updatedRows, rowsErr := voteResult.RowsAffected(); rowsErr != nil {
		logger.ErrorContext(ctx, "failed to get affected rows", slog.Any("error", rowsErr))
	} else if updatedRows == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Song not found"})
		return
	}

	NotifyVote(songID, ownCountry, db)

	logger.InfoContext(ctx, "vote cast",
		slog.String("phone_hash", phoneHash),
		slog.Int("song_id", songID),
		slog.Int("points_spent", points),
		slog.Int("votes_remaining", remaining),
	)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message":         "Vote cast successfully",
		"voted_for":       songID,
		"points_spent":    points,
		"votes_remaining": remaining,
		"votes_cast":      votesCast,
	})
}

func getCountries(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	query := `SELECT ID, Name, POT FROM Land ORDER BY Name ASC`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		logger.ErrorContext(ctx, "failed to query countries", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to retrieve countries",
		})
		return
	}
	defer rows.Close()

	type Country struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Pot  *int   `json:"pot"`
	}

	var countries []Country
	for rows.Next() {
		var c Country
		var potValue sql.NullInt64
		if err := rows.Scan(&c.ID, &c.Name, &potValue); err != nil {
			logger.ErrorContext(ctx, "failed to scan row", slog.Any("error", err))
			continue
		}
		if potValue.Valid {
			pot := int(potValue.Int64)
			c.Pot = &pot
		}
		countries = append(countries, c)
	}

	if err := rows.Err(); err != nil {
		logger.ErrorContext(ctx, "rows iteration error", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to process countries",
		})
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message": "Success",
		"payload": countries,
	})
}

func getCountryByName(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")
	idStr := r.PathValue("NAME")

	query := `SELECT ID, Name, POT FROM Land WHERE ID = ?`

	rows, err := db.QueryContext(ctx, query, idStr)

	if err != nil {
		logger.ErrorContext(ctx, "Error Querying Database", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{
			"error":       "Could not retrieve Country with ID",
			"requestedID": idStr,
		})
		return
	}
	defer rows.Close()

	type Country struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Pot  *int   `json:"pot"`
	}

	var countries []Country
	for rows.Next() {
		var c Country
		var potValue sql.NullInt64
		if err := rows.Scan(&c.ID, &c.Name, &potValue); err != nil {
			logger.ErrorContext(ctx, "failed to scan row", slog.Any("error", err))
			continue
		}
		if potValue.Valid {
			pot := int(potValue.Int64)
			c.Pot = &pot
		}
		countries = append(countries, c)
	}

	if err := rows.Err(); err != nil {
		logger.ErrorContext(ctx, "rows iteration error", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to process countries",
		})
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message": "Success",
		"payload": countries,
	})
}

func httpGetSongs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	query := `SELECT
    s.ID AS Song_ID,
    s.Name AS Song_Name,
    s.PublikumsPunkte,
    s.JuryPunkte,
    s.GesamtPunkte,
    l.ID AS Land_ID,
    l.Name AS Land_Name,
    l.POT AS Land_POT,
    k.ID AS Kuenstler_ID,
    k.Vorname AS Kuenstler_Vorname,
    k.Name AS Kuenstler_Name,
    k.Typ AS Kuenstler_Typ,
    komp.ID AS Komponist_ID,
    komp.Vorname AS Komponist_Vorname,
    komp.Name AS Komponist_Name,
    vs.VotingID,
    vs.isOpen AS Voting_IsOpen,
    vs.lastChange AS Voting_LastChange
    FROM Song s
    INNER JOIN Land l ON s.Land_ID = l.ID
    INNER JOIN Kuenstler k ON s.Kuenstler_ID = k.ID
    LEFT JOIN Song_Komponist sk ON s.ID = sk.Song_ID
    LEFT JOIN Komponist komp ON sk.Komponist_ID = komp.ID
    LEFT JOIN Voting_Status vs ON vs.VotingID = 1  -- Assuming VotingID 1 is the current contest
    ORDER BY s.ID, komp.ID;
`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		logger.ErrorContext(ctx, "Error Querying Database", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{
			"error": "Faules querying Database",
		})
		return
	}
	defer rows.Close()

	type Komponist struct {
		ID      int `json:"id"`
		vorname string
		name    string
	}

	type CompleteESCEntryWithComposers struct {
		// Song information
		SongID          int    `json:"songId"`
		SongName        string `json:"songName"`
		PublikumsPunkte int    `json:"publicVotes"`
		JuryPunkte      int    `json:"juryVotes"`
		GesamtPunkte    int    `json:"totalVotes"`

		// Land (Country) information for the song
		LandID   string `json:"countryId"`
		LandName string `json:"countryName"`
		LandPOT  *int   `json:"countryPot,omitempty"`

		// Künstler (Artist) information
		KuenstlerID      int    `json:"artistId"`
		KuenstlerVorname string `json:"artistFirstName"`
		KuenstlerName    string `json:"artistLastName"`
		KuenstlerTyp     string `json:"artistType"`

		// Komponist (Composers) information - aggregated as array
		Komponisten []Komponist `json:"composers"`

		// Voting Status information
		VotingID         int    `json:"votingId"`
		VotingIsOpen     bool   `json:"votingIsOpen"`
		VotingLastChange string `json:"votingLastChange"`
	}

	var song []CompleteESCEntryWithComposers
	for rows.Next() {
		var c CompleteESCEntryWithComposers
		var komponentID sql.NullInt64
		var komponentVorname sql.NullString
		var komponentName sql.NullString
		if err := rows.Scan(&c.SongID, &c.SongName, &c.PublikumsPunkte, &c.JuryPunkte, &c.GesamtPunkte, &c.LandID, &c.LandName, &c.LandPOT, &c.KuenstlerID, &c.KuenstlerVorname, &c.KuenstlerName, &c.KuenstlerTyp, &komponentID, &komponentVorname, &komponentName, &c.VotingID, &c.VotingIsOpen, &c.VotingLastChange); err != nil {
			logger.ErrorContext(ctx, "failed to scan row", slog.Any("error", err))
			continue
		}
		if komponentID.Valid {
			komponent := Komponist{
				ID:      int(komponentID.Int64),
				vorname: komponentVorname.String,
				name:    komponentName.String,
			}
			c.Komponisten = append(c.Komponisten, komponent)
		}
		song = append(song, c)
	}

	if err := rows.Err(); err != nil {
		logger.ErrorContext(ctx, "rows iteration error", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to process Song",
		})
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message": "Success",
		"payload": song,
	})
}

func getSongbyID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")
	idStr := r.PathValue("ID")
	ID, err := strconv.Atoi(idStr)
	if err != nil {
		logger.ErrorContext(ctx, "Invalid ID Value", slog.Any("error", err), slog.String("ID", idStr))
	}

	query := `SELECT
    s.ID AS Song_ID,
    s.Name AS Song_Name,
    s.PublikumsPunkte,
    s.JuryPunkte,
    s.GesamtPunkte,
    l.ID AS Land_ID,
    l.Name AS Land_Name,
    l.POT AS Land_POT,
    k.ID AS Kuenstler_ID,
    k.Vorname AS Kuenstler_Vorname,
    k.Name AS Kuenstler_Name,
    k.Typ AS Kuenstler_Typ,
    komp.ID AS Komponist_ID,
    komp.Vorname AS Komponist_Vorname,
    komp.Name AS Komponist_Name,
    vs.VotingID,
    vs.isOpen AS Voting_IsOpen,
    vs.lastChange AS Voting_LastChange
    FROM Song s
    INNER JOIN Land l ON s.Land_ID = l.ID
    INNER JOIN Kuenstler k ON s.Kuenstler_ID = k.ID
    LEFT JOIN Song_Komponist sk ON s.ID = sk.Song_ID
    LEFT JOIN Komponist komp ON sk.Komponist_ID = komp.ID
    LEFT JOIN Voting_Status vs ON vs.VotingID = 1
    WHERE s.ID = ?
    ORDER BY s.ID, komp.ID;
`

	rows, err := db.QueryContext(ctx, query, ID)
	if err != nil {
		logger.ErrorContext(ctx, "Error Querying Database", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{
			"error": "Faules querying Database",
		})
		return
	}
	defer rows.Close()

	type Komponist struct {
		ID      int `json:"id"`
		vorname string
		name    string
	}

	type CompleteESCEntryWithComposers struct {
		// Song information
		SongID          int    `json:"songId"`
		SongName        string `json:"songName"`
		PublikumsPunkte int    `json:"publicVotes"`
		JuryPunkte      int    `json:"juryVotes"`
		GesamtPunkte    int    `json:"totalVotes"`

		// Land (Country) information for the song
		LandID   string `json:"countryId"`
		LandName string `json:"countryName"`
		LandPOT  *int   `json:"countryPot,omitempty"`

		// Künstler (Artist) information
		KuenstlerID      int    `json:"artistId"`
		KuenstlerVorname string `json:"artistFirstName"`
		KuenstlerName    string `json:"artistLastName"`
		KuenstlerTyp     string `json:"artistType"`

		// Komponist (Composers) information - aggregated as array
		Komponisten []Komponist `json:"composers"`

		// Voting Status information
		VotingID         int    `json:"votingId"`
		VotingIsOpen     bool   `json:"votingIsOpen"`
		VotingLastChange string `json:"votingLastChange"`
	}

	var song []CompleteESCEntryWithComposers
	for rows.Next() {
		var c CompleteESCEntryWithComposers
		var komponentID sql.NullInt64
		var komponentVorname sql.NullString
		var komponentName sql.NullString
		if err := rows.Scan(&c.SongID, &c.SongName, &c.PublikumsPunkte, &c.JuryPunkte, &c.GesamtPunkte, &c.LandID, &c.LandName, &c.LandPOT, &c.KuenstlerID, &c.KuenstlerVorname, &c.KuenstlerName, &c.KuenstlerTyp, &komponentID, &komponentVorname, &komponentName, &c.VotingID, &c.VotingIsOpen, &c.VotingLastChange); err != nil {
			logger.ErrorContext(ctx, "failed to scan row", slog.Any("error", err))
			continue
		}
		if komponentID.Valid {
			komponent := Komponist{
				ID:      int(komponentID.Int64),
				vorname: komponentVorname.String,
				name:    komponentName.String,
			}
			c.Komponisten = append(c.Komponisten, komponent)
		}
		song = append(song, c)
	}

	if err := rows.Err(); err != nil {
		logger.ErrorContext(ctx, "rows iteration error", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to process Song",
		})
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message": "Success",
		"payload": song,
	})
}

func openVote(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ctx := r.Context()
	query := r.URL.Query()
	token := query.Get("Token")
	dbQuery := `UPDATE Voting_Status SET isOpen = true, lastChange = ?`

	autorized, message := checkAccessAdmin(token)

	if autorized == true {

		changeTime := time.Now()

		result, err := db.ExecContext(ctx, dbQuery, changeTime)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			logger.ErrorContext(ctx, "Could not Query Rows", slog.Any("error", err))
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Error querying rows",
			})
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			logger.ErrorContext(ctx, "failed to open Vote", slog.Any("error", err))
		}
		if rowsAffected == 0 {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Error opening the Votes",
			})
			return
		}

		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]any{
			"message": message,
			"payload": "The Vote has been opened",
		})
	}

	if autorized != true {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{
			"message": message,
		})
		return
	}
}

func closeVote(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ctx := r.Context()
	query := r.URL.Query()
	token := query.Get("Token")
	dbQuery := `UPDATE Voting_Status SET isOpen = false, lastChange = ?`

	autorized, message := checkAccessAdmin(token)

	if autorized == true {

		changeTime := time.Now()

		result, err := db.ExecContext(ctx, dbQuery, changeTime)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			logger.ErrorContext(ctx, "Could not Query Rows", slog.Any("error", err))
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Error querying rows",
			})
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			logger.ErrorContext(ctx, "failed to close Votes", slog.Any("error", err))
		}
		if rowsAffected == 0 {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Error Closing Votes",
			})
			return
		}

		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]any{
			"message": message,
			"payload": "The Vote has been closed",
		})
		return
	}

	if autorized != true {

		logger.Warn("Invalid Login Attempt")
		slog.String("token", token)

		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{
			"message": message,
		})
	}
}

func deleteVotes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ctx := r.Context()
	query := r.URL.Query()
	token := query.Get("Token")

	autorized, message := checkAccessAdmin(token)

	if autorized != true {
		logger.Warn("Invalid Login Attempt", slog.String("token", token))
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"message": message})
		return
	}

	// Reset all song vote tallies.
	result, err := db.ExecContext(ctx, `UPDATE Song SET PublikumsPunkte = 0, JuryPunkte = 0`)
	if err != nil {
		logger.ErrorContext(ctx, "Error resetting song votes", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to delete Votes"})
		return
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		logger.ErrorContext(ctx, "failed to get rows affected after vote reset", slog.Any("error", err))
	}
	if rowsAffected == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "No votes found to delete"})
		return
	}

	// Reset all phone number vote budgets so everyone gets their 20 points back.
	if _, phoneErr := db.ExecContext(ctx,
		`UPDATE Phone_Nums SET votes_remaining = ?, votes_cast = JSON_OBJECT()`, totalVotePoints,
	); phoneErr != nil {
		logger.ErrorContext(ctx, "failed to reset phone vote budgets", slog.Any("error", phoneErr))
		// Non-fatal: song votes are already cleared; warn but continue.
	}

	// Also clear the in-memory token store so cookies are invalidated.
	mu.Lock()
	usedTokens = make(map[string]bool)
	mu.Unlock()

	logger.InfoContext(ctx, "all votes deleted and vote budgets reset")

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]any{
		"message": message,
		"payload": "All votes have been deleted and vote budgets reset",
	})
}

func addCountry(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	query := r.URL.Query()
	token := query.Get("Token")
	ID := query.Get("ID")
	Name := query.Get("Name")
	POTstr := query.Get("Pot")
	POT, err := strconv.Atoi(POTstr)
	if err != nil {
		logger.ErrorContext(ctx, "Invalid POT value", slog.Any("error", err), slog.String("POT", POTstr))
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Pot Value must be Integer",
		})
		return
	}
	dbQuery := `INSERT INTO Land (ID, Name, POT)
				VALUES (?, ?, ?)`

	autorized, message := checkAccessAdmin(token)

	if autorized == true {

		result, err := db.ExecContext(ctx, dbQuery, ID, Name, POT)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to Insert Data", slog.Any("error", err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "error inserting into DB",
			})
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			logger.ErrorContext(ctx, "failed to verify insertions", slog.Any("error", err))
		}
		if rowsAffected == 0 {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Error verifying Query",
			})
			return
		}

		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]any{
			"message": message,
			"payload": result,
		})
	}

	if autorized != true {

		logger.Warn("Invalid Login Attempt")
		slog.String("token", token)

		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{
			"message": message,
		})
	}
}

func addSong(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	query := r.URL.Query()
	token := query.Get("Token")
	IDstr := query.Get("ID")
	ID, err := strconv.Atoi(IDstr)
	if err != nil {
		logger.ErrorContext(ctx, "Invalid ID Value", slog.Any("error", err), slog.String("ID", IDstr))
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "ID must be an Integer",
		})
		return
	}
	Name := query.Get("Name")
	country := query.Get("Land")
	dbQuery := `
				INSERT INTO Song (Name, Land_ID, Kuenstler_ID, PublikumsPunkte, JuryPunkte)
				VALUES (?, ?, ?, 0, 0)`

	autorized, message := checkAccessAdmin(token)

	if autorized == true {

		result, err := db.ExecContext(ctx, dbQuery, Name, country, ID)

		if err != nil {
			logger.ErrorContext(ctx, "Failed to Insert Data", slog.Any("error", err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "error inserting into DB",
			})
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			logger.ErrorContext(ctx, "failed to verify insertions", slog.Any("error", err))
		}
		if rowsAffected == 0 {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Error verifying Query",
			})
			return
		}

		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]any{
			"message": message,
			"payload": result,
		})
	}

	if autorized != true {

		logger.Warn("Invalid Login Attempt")
		slog.String("token", token)

		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{
			"message": message,
		})
	}

}

func addArtist(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	query := r.URL.Query()
	token := query.Get("Token")
	IDstr := query.Get("ID")
	ID, err := strconv.Atoi(IDstr)
	if err != nil {
		logger.ErrorContext(ctx, "Error Parsing ID", slog.Any("error", err), slog.String("ID", IDstr))
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "ID must be an Integer",
		})
		return
	}
	Name := query.Get("Name")
	vorName := query.Get("vorName")
	typ := query.Get("typ")
	country := query.Get("Land")
	dbQuery := `INSERT INTO Kuenstler (ID, Vorname, Name, Typ, Land_ID) VALUES (?, ?, ?, ?, ?)`

	autorized, message := checkAccessAdmin(token)

	if autorized == true {

		result, err := db.ExecContext(ctx, dbQuery, ID, Name, vorName, typ, country)

		if err != nil {
			logger.ErrorContext(ctx, "Failed to Insert Data", slog.Any("error", err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "error inserting into DB",
			})
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			logger.ErrorContext(ctx, "failed to verify insertions", slog.Any("error", err))
		}
		if rowsAffected == 0 {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Error verifying Query",
			})
			return
		}

		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]any{
			"message": message,
			"payload": result,
		})
	}

	if autorized != true {

		logger.Warn("Invalid Login Attempt")
		slog.String("token", token)

		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{
			"message": message,
		})
	}
}

func addInterpret(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	query := r.URL.Query()
	token := query.Get("Token")
	IDstr := query.Get("ID")
	ID, err := strconv.Atoi(IDstr)
	if err != nil {
		logger.ErrorContext(ctx, "Error Parsing ID", slog.Any("error", err), slog.String("ID", IDstr))
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "ID must be an Integer",
		})
		return
	}
	Name := query.Get("Name")
	vorName := query.Get("Vorname")
	dbQuery := `INSERT INTO Komponist (ID, Vorname, Name) VALUES (?,?,?)`

	autorized, message := checkAccessAdmin(token)

	if autorized == true {

		result, err := db.ExecContext(ctx, dbQuery, ID, Name, vorName)

		if err != nil {
			logger.ErrorContext(ctx, "Failed to Insert Data", slog.Any("error", err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "error inserting into DB",
			})
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			logger.ErrorContext(ctx, "failed to verify insertions", slog.Any("error", err))
		}
		if rowsAffected == 0 {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Error verifying Query",
			})
			return
		}

		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]any{
			"message": message,
			"payload": result,
		})
	}

	if autorized != true {

		logger.Warn("Invalid Login Attempt")
		slog.String("token", token)

		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{
			"message": message,
		})
	}
}

func juryVote(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()

	query := r.URL.Query()
	token := query.Get("Token")
	points := query.Get("points")
	songIDStr := query.Get("songID")
	songID, err := strconv.Atoi(songIDStr)
	if err != nil {
		logger.ErrorContext(ctx, "invalid songID value", slog.Any("error", err), slog.String("songID", songIDStr))
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid songID value - must be an Integer",
		})
		return
	}
	dbQuery := `SELECT isOpen FROM Voting_Status`

	authorized, message := checkAccessJury(token)

	if authorized == true {

		rows, err := db.QueryContext(ctx, dbQuery)
		if err != nil {
			logger.ErrorContext(ctx, "Error Querying DB", slog.Any("error", err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Error Querying DB",
			})
			return
		}

		defer rows.Close()

		type vote_status struct {
			id         string
			isOpen     bool
			lastChange time.Time
		}

		for rows.Next() {
			var v vote_status
			if err := rows.Scan(&v.id, &v.isOpen, &v.lastChange); err != nil {
				logger.ErrorContext(ctx, "failed to scan row", slog.Any("error", err))
				continue
			}
			if v.isOpen == false {
				w.WriteHeader(http.StatusTooEarly)
				json.NewEncoder(w).Encode(map[string]any{
					"error":   "voting has not started yet, please try again in a few minutes",
					"payload": v,
				})
				return
			}
		}

		const juryWeight int = 5
		parsedPoints, err := strconv.Atoi(points)
		if err != nil {
			logger.ErrorContext(ctx, "invalid points value", slog.Any("error", err), slog.String("points", points))
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Invalid points value - must be an Integer",
			})
			return
		}

		totalPoints := juryWeight * parsedPoints

		dbQuery = `UPDATE Song
				 SET JuryPunkte = JuryPunkte + ?
				 WHERE ID = ?`

		result, err := db.ExecContext(ctx, dbQuery, totalPoints, songID)
		if err != nil {
			logger.ErrorContext(ctx, "failed to update vote", slog.Any("error", err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Failed to record vote",
			})
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			logger.ErrorContext(ctx, "failed to get affected rows", slog.Any("error", err))
		}
		if rowsAffected == 0 {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Song not found",
			})
			return
		}

		NotifyVote(songID, "JURY", db)

		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]any{
			"message": message,
			"payload": "Vote Successfully Cast",
		})
	}

	if authorized != true {

		logger.Warn("Invalid Login Attempt")
		slog.String("token", token)

		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{
			"message": message,
		})
	}

}

func adminLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()

	query := r.URL.Query()
	token := query.Get("Token")

	authenticated, message := checkAccessAdmin(token)

	if authenticated == true {
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{
			"message": message,
		})
		return
	}
	if authenticated == false {
		logger.WarnContext(ctx, "Invalid Login Atempt", slog.Any("token:", token))
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{
			"error": message,
		})
		return
	}
}

func juryLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()

	query := r.URL.Query()
	token := query.Get("Token")

	authenticated, message := checkAccessJury(token)

	if authenticated == true {
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{
			"message": message,
		})
		return
	}
	if authenticated == false {
		logger.WarnContext(ctx, "Invalid Login Atempt", slog.Any("token:", token))
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{
			"error": message,
		})
		return
	}
}

var (
	maxDBAttempts  = 60
	dbAttemptDelay = time.Second * 3
)

func connectToDatabase(cfg LocalConfig) (*sql.DB, error) {
	// URL encode the password to handle special characters
	escapedPass := url.QueryEscape(cfg.DbPass)
	// Add connection timeout parameters to DSN
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&timeout=10s&readTimeout=10s&writeTimeout=10s",
		cfg.DbUser,
		escapedPass,
		cfg.DbHost,
		cfg.DbPort,
		cfg.DbName,
	)

	log.Printf("Attempting to connect to database at %s:%d/%s", cfg.DbHost, cfg.DbPort, cfg.DbName)
	log.Printf("Database config: Host=%s, Port=%d, User=%s, Database=%s", cfg.DbHost, cfg.DbPort, cfg.DbUser, cfg.DbName)

	var (
		conn *sql.DB
		err  error
	)
	for attempt := 1; attempt <= maxDBAttempts; attempt++ {
		conn, err = sql.Open("mysql", dsn)
		if err == nil {
			// Set connection pool settings
			conn.SetMaxOpenConns(25)
			conn.SetMaxIdleConns(5)
			conn.SetConnMaxLifetime(5 * time.Minute)

			// Use a context with timeout for ping
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			pingErr := conn.PingContext(ctx)
			cancel()

			if pingErr == nil {
				log.Printf("Successfully connected to database on attempt %d", attempt)
				return conn, nil
			}
			err = pingErr
		}
		var errMsg string
		if err != nil {
			errMsg = err.Error()
		} else {
			errMsg = "unknown error"
		}
		log.Printf("Database connection attempt %d/%d failed: %s", attempt, maxDBAttempts, errMsg)
		if attempt < maxDBAttempts {
			time.Sleep(dbAttemptDelay * time.Duration(attempt))
		}
	}
	return nil, fmt.Errorf("could not connect after %d attempts: %w", maxDBAttempts, err)
}

func main() {
	initCookieSecret()

	// Base JSON handler writing to stdout (keeps existing log lines intact).
	baseHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	// Temporarily set a plain logger so that initLogProvider / initTracer
	// failures can still be reported before the OTel handler is ready.
	logger = slog.New(baseHandler)
	slog.SetDefault(logger)

	tp, erro := initTracer()
	if erro != nil {
		log.Printf("Warning: Failed to initialize tracer: %v. Continuing without tracing", erro)
	} else {
		defer func() {
			if err := tp.Shutdown(context.Background()); err != nil {
				log.Printf("Error shutting down tracer provider: %v", err)
			}
		}()
	}

	lp, err := initLogProvider()
	if err != nil {
		log.Printf("Warning: Failed to initialize log provider: %v. Logs will not be forwarded via OTLP", err)
	} else {
		defer func() {
			if err := lp.Shutdown(context.Background()); err != nil {
				log.Printf("Error shutting down log provider: %v", err)
			}
		}()
	}

	// Replace the logger with one that emits to both stdout and the OTel
	// collector (which forwards to Loki with traceID as an index label).
	logger = slog.New(newOtelSlogHandler(baseHandler))
	slog.SetDefault(logger)

	tracer = otel.Tracer("esc-voting-crud-api")
	log.Println("Listening and Serving on Port 8000")
	logger.Info("server starting",
		slog.Int("port", 8000),
		slog.String("service", "esc-voting-crud-api"),
	)

	router := http.NewServeMux()
	router.HandleFunc("GET /health", getHealth)
	router.HandleFunc("GET /votes/", getVotes)
	router.HandleFunc("POST /vote/", vote)
	router.HandleFunc("GET /countries/", getCountries)
	router.HandleFunc("GET /countryByName/{NAME}", getCountryByName)
	router.HandleFunc("GET /songs/", httpGetSongs)
	router.HandleFunc("GET /songByID/{ID}", getSongbyID)
	router.HandleFunc("POST /admin/open/", openVote)
	router.HandleFunc("POST /admin/close", closeVote)
	router.HandleFunc("DELETE /admin/deleteVotes/", deleteVotes)
	router.HandleFunc("POST /admin/addCountry/", addCountry)
	router.HandleFunc("POST /admin/addSong/", addSong)
	router.HandleFunc("POST /admin/addArtist/", addArtist)
	router.HandleFunc("POST /admin/addInterpret/", addInterpret)
	router.HandleFunc("POST /jury/vote/", juryVote)
	router.HandleFunc("GET /admin/authenticate", adminLogin)
	router.HandleFunc("GET /jury/authenticate", juryLogin)

	router.Handle("GET /metrics/", promhttp.Handler())

	handler := RateLimitingMiddleware(ObservabilityMiddleware(router))

	var dbErr error
	db, dbErr = connectToDatabase(loadLocalConfig())
	if dbErr != nil {
		logger.Error("Error Connecting the Database")
		panic(dbErr.Error())
	}
	defer db.Close()

	voteService, err := StartGRPCServer(db, "50051")
	if err != nil {
		logger.Error("Failed to start gRPC server", slog.String("error", err.Error()))
		panic(err)
	}
	SetGlobalVoteServer(voteService)
	logger.Info("gRPC vote streaming service initialized")

	// Start Sercer
	err = http.ListenAndServe(":8000", handler)
	if err != nil {
		logger.Error("Server failed", slog.String("error", err.Error()))
		fmt.Println("Error Starting the Server")
	}
}
