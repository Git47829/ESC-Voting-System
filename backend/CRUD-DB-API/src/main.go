package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"

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
	"golang.org/x/time/rate"
)

type Client struct {
	limiter *rate.Limiter
}

type RateLimitConfig struct {
	RequestsPerSecond float64
	BurstSize         int
}

type LocalConfig struct {
	dbName     string `env:"dbName"`
	dbUser     string `env:"dbUser"`
	dbPass     string `env:"dbPass"`
	dbEndpoint string `env:"dbEndpoint"`
	dbPort     int    `env:"dbPort"`
}

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

// Token Store
var (
	usedTokens = make(map[string]bool)
	mu         sync.Mutex
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

func checkAccessAdmin(input string) (bool, string) {

	var TestToken string = "test123"

	if input == "" {
		return false, "Token has to be provided"
	}
	if input != TestToken {
		return false, "Wrong Token provided"
	}

	if input == TestToken {
		return true, "Autorized"
	}

	return false, "Error Processing Token"
}

func checkAccessJury(input string) (bool, string) {

	TestToken := []string{"test123", "test456", "test789"}

	if input == "" {
		return false, "Token has to be provided"
	}

	for _, token := range TestToken {
		if input == token {
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

	songID := r.URL.Query().Get("songID")

	query := `SELECT isOpen FROM Voting_Status`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to query db Rows, is Database connected?", slog.Any("error", err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to query Rows",
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

	const cookieName string = "vote_cookie"
	const weightPublicVote int = 3

	cookie, err := r.Cookie(cookieName)

	if err == nil {
		mu.Lock()
		alreadyVoted := usedTokens[cookie.Value]
		mu.Unlock()

		if alreadyVoted == true {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{
				"message": "Vote has already been cast",
			})
		}
	}

	token, _ := generateToken()
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Expires:  time.Now().Add(365 * 24 * time.Hour),
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	mu.Lock()
	usedTokens[token] = true
	mu.Unlock()

	query = `UPDATE SONG
			 SET PublikumsPunkte = PublikumsPunkte + ?
			 WHERE ID = ?`

	result, err := db.ExecContext(ctx, query, weightPublicVote, songID)
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

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message":   "Success",
		"voted_for": songID,
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
		if err := rows.Scan(&c.ID, &c.Name, &c.Pot); err != nil {
			logger.ErrorContext(ctx, "failed to scan row", slog.Any("error", err))
			continue
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
		if err := rows.Scan(&c.ID, &c.Name, &c.Pot); err != nil {
			logger.ErrorContext(ctx, "failed to scan row", slog.Any("error", err))
			continue
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
		ID      int    `json:"id"`
		vorname string `json:"first_name"`
		name    string `json:"surname"`
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
		if err := rows.Scan(&c.SongID, &c.SongName, &c.PublikumsPunkte, &c.JuryPunkte, &c.GesamtPunkte, &c.LandID, &c.LandName, &c.LandPOT, &c.KuenstlerID, &c.KuenstlerVorname, &c.KuenstlerName, &c.KuenstlerTyp, &c.Komponisten); err != nil {
			logger.ErrorContext(ctx, "failed to scan row", slog.Any("error", err))
			continue
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
    WHERE ID = ?
    INNER JOIN Land l ON s.Land_ID = l.ID
    INNER JOIN Kuenstler k ON s.Kuenstler_ID = k.ID
    LEFT JOIN Song_Komponist sk ON s.ID = sk.Song_ID
    LEFT JOIN Komponist komp ON sk.Komponist_ID = komp.ID
    LEFT JOIN Voting_Status vs ON vs.VotingID = 1  -- Assuming VotingID 1 is the current contest
    ORDER BY s.ID, komp.ID;
`

	rows, err := db.QueryContext(ctx, query, idStr)
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
		ID      int    `json:"id"`
		vorname string `json:"first_name"`
		name    string `json:"surname"`
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
		if err := rows.Scan(&c.SongID, &c.SongName, &c.PublikumsPunkte, &c.JuryPunkte, &c.GesamtPunkte, &c.LandID, &c.LandName, &c.LandPOT, &c.KuenstlerID, &c.KuenstlerVorname, &c.KuenstlerName, &c.KuenstlerTyp, &c.Komponisten); err != nil {
			logger.ErrorContext(ctx, "failed to scan row", slog.Any("error", err))
			continue
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
	dbQuery := `UPDATE Voting_Status SET isOpen = true, SET lastChange = ?`

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
		defer db.Close()

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
	dbQuery := `UPDATE Voting_Status SET isOpen = true, SET lastChange = ?`

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
		defer db.Close()

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
	dbQuery := `DELETE PublikumsPunkte, JuryPunkte, GesamtPunkte FROM Song`

	autorized, message := checkAccessAdmin(token)

	if autorized == true {

		result, err := db.ExecContext(ctx, dbQuery)
		if err != nil {
			logger.ErrorContext(ctx, "Error Executing DB Query", slog.Any("error", err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Failed to delete Votes",
			})
		}
		defer db.Close()

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			logger.ErrorContext(ctx, "failed delete Votes", slog.Any("error", err))
		}
		if rowsAffected == 0 {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Error deleting Votes",
			})
			return
		}

		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]any{
			"message": message,
			"payload": "All Votes have been deleted",
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

func addCountry(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	query := r.URL.Query()
	token := query.Get("Token")
	ID := query.Get("ID")
	Name := query.Get("Name")
	POT := query.Get("Pot")
	dbQuery := `INSERT INTO Land (ID, Name, POT)
				VALUES (?, ?, ?);`

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
		defer db.Close()

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
	w.Header().Set("Content-Type", "application/json")

	query := r.URL.Query()
	token := query.Get("Token")
	ID := query.Get("ID")
	Name := query.Get("Name")

	autorized, message := checkAccessAdmin(token)

	response := []any{ID, Name}

	if autorized == true {

		// Business Logic

		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]any{
			"message": message,
			"payload": response,
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
	w.Header().Set("Content-Type", "application/json")

	query := r.URL.Query()
	token := query.Get("Token")
	ID := query.Get("ID")
	Name := query.Get("Name")
	vorName := query.Get("vorName")

	autorized, message := checkAccessAdmin(token)
	response := []any{ID, Name, vorName}

	if autorized == true {

		// Business Logic

		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]any{
			"message": message,
			"payload": response,
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
	w.Header().Set("Content-Type", "application/json")

	query := r.URL.Query()
	token := query.Get("Token")
	ID := query.Get("ID")
	Name := query.Get("Name")

	autorized, message := checkAccessAdmin(token)
	response := []any{ID, Name}

	if autorized == true {

		// Business Logic

		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]any{
			"message": message,
			"payload": response,
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

	query := r.URL.Query()
	token := query.Get("Token")
	vote := query.Get("vote")
	points := query.Get("points")

	authorized, message := checkAccessJury(token)
	response := []any{vote, points}

	if authorized == true {

		//Business Logic

		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]any{
			"message": message,
			"payload": response,
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

	router.Handle("GET /metrics/", promhttp.Handler())

	handler := RateLimitingMiddleware(ObservabilityMiddleware(router))

	err = godotenv.Load("../.env")
	if err != nil {
		slog.Any("Failed to Load .env", err)
	}

	var localconfig LocalConfig
	if err := env.Parse(&localconfig); err != nil {
		log.Fatalf("Error reading the environment variables: %v", err)
		return
	}

	log.Printf("%+v\n", localconfig)

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		localconfig.dbUser,
		localconfig.dbPass,
		localconfig.dbEndpoint,
		localconfig.dbPort,
		localconfig.dbName,
	)

	db, err = sql.Open("mysql", dsn)
	if err != nil {
		logger.Error("Error Connecting the Database")
		panic(err.Error())
	}
	defer db.Close()

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(3 * time.Minute)

	err = db.Ping()
	if err != nil {
		logger.Error("Database is unreachable")
		panic(err.Error())
	}

	logger.Info("Successfully Connected to the Database")

	// Start Sercer
	err = http.ListenAndServe(":8000", handler)
	if err != nil {
		logger.Error("Server failed", slog.String("error", err.Error()))
		fmt.Println("Error Starting the Server")
	}
}
