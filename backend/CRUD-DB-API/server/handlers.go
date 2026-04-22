package server

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"golang.org/x/sync/errgroup"
)

type Countrys struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Pot  *int   `json:"pot"`
}

type Composer struct {
	ID        int `json:"id"`
	firstName string
	name      string
}

type CompleteESCEntryWithComposers struct {
	SongID       int    `json:"songId"`
	SongName     string `json:"songName"`
	PublicPoints int    `json:"publicVotes"`
	JuryPoints   int    `json:"juryVotes"`
	TotalPoints  int    `json:"totalVotes"`

	CountryID   string `json:"countryId"`
	CountryName string `json:"countryName"`
	CountryPOT  *int   `json:"countryPot,omitempty"`

	ArtistID        int    `json:"artistId"`
	ArtistFirstName string `json:"artistFirstName"`
	ArtistName      string `json:"artistLastName"`
	ArtistType      string `json:"artistType"`

	Composer []Composer `json:"composers"`

	VotingID         int    `json:"votingId"`
	VotingIsOpen     bool   `json:"votingIsOpen"`
	VotingLastChange string `json:"votingLastChange"`
}

type RateLimitConfig struct {
	Limit  int
	Window time.Duration
}

var rateLimitConfigs = map[string]RateLimitConfig{
	"GET /health":                {Limit: 100, Window: time.Second},
	"GET /votes/":                {Limit: 20, Window: time.Second},
	"GET /countries/":            {Limit: 20, Window: time.Second},
	"GET /songs/":                {Limit: 20, Window: time.Second},
	"GET /songByID/{ID}":         {Limit: 20, Window: time.Second},
	"GET /contest/current":       {Limit: 20, Window: time.Second},
	"POST /auth/login":           {Limit: 3, Window: time.Second},
	"POST /auth/verify-code":     {Limit: 5, Window: time.Second},
	"GET /auth/verify":           {Limit: 10, Window: time.Second},
	"POST /auth/verify":          {Limit: 10, Window: time.Second},
	"GET /auth/me":               {Limit: 10, Window: time.Second},
	"POST /auth/refresh":         {Limit: 10, Window: time.Second},
	"POST /auth/logout":          {Limit: 10, Window: time.Second},
	"POST /vote/":                {Limit: 1, Window: time.Second},
	"POST /jury/vote":            {Limit: 5, Window: time.Second},
	"GET /jury/authenticate":     {Limit: 1, Window: time.Second},
	"GET /admin/authenticate":    {Limit: 1, Window: time.Second},
	"POST /admin/open":           {Limit: 2, Window: time.Second},
	"POST /admin/close":          {Limit: 2, Window: time.Second},
	"POST /admin/addCountry":     {Limit: 5, Window: time.Second},
	"POST /admin/addSong":        {Limit: 5, Window: time.Second},
	"POST /admin/addArtist":      {Limit: 5, Window: time.Second},
	"POST /admin/addInterpret":   {Limit: 5, Window: time.Second},
	"POST /admin/startContest":   {Limit: 5, Window: time.Second},
	"POST /admin/advanceContest": {Limit: 5, Window: time.Second},
	"DELETE /admin/deleteVotes":  {Limit: 1, Window: time.Second},
	"GET /results":               {Limit: 20, Window: time.Second},
	"GET /metrics/":              {Limit: 10000, Window: time.Second},
}

const totalVotePoints = 20

type CookieVoteState struct {
	VotesRemaining int            `json:"votes_remaining"`
	VotesCast      map[string]int `json:"votes_cast"`
}

var SignedCookieSecret []byte
var SignedPhoneSecret []byte

func InitCookieSecret() {
	if key := os.Getenv("COOKIESIGNINGKEY"); key != "" {
		sum := sha256.Sum256([]byte("cookie-secret:" + key))
		SignedCookieSecret = sum[:]
		return
	}
	SignedCookieSecret = make([]byte, 32)
	if _, err := rand.Read(SignedCookieSecret); err != nil {
		panic("cannot generate cookie secret: " + err.Error())
	}
}

func InitPhoneSecret() {
	if key := os.Getenv("PHONESIGNINGSECRET"); key != "" {
		sum := sha256.Sum256([]byte("phone-secret:" + key))
		SignedPhoneSecret = sum[:]
	}
}

func EncodeCookieValue(state CookieVoteState) (string, error) {
	data, err := json.Marshal(state)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, SignedCookieSecret)
	mac.Write(data)
	sig := mac.Sum(nil)
	payload := append(data, '.')
	payload = append(payload, []byte(hex.EncodeToString(sig))...)
	return hex.EncodeToString(payload), nil
}

func getClientIP(r *http.Request) string {
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.Header.Get("X-Real_IP")
	}
	if ip == "" {
		ip, _, _ = net.SplitHostPort(r.RemoteAddr)
	}
	return ip
}

func RateLimitingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		endpoint := fmt.Sprintf("%s %s", r.Method, r.URL.Path)
		clientIP := getClientIP(r)

		config, exists := rateLimitConfigs[endpoint]
		if !exists {
			config = RateLimitConfig{Limit: 20, Window: time.Second}
		}

		key := fmt.Sprintf("ratelimit:%s::%s", clientIP, endpoint)
		allowed, err := CheckRateLimit(r.Context(), key, config.Limit, config.Window)
		if err != nil {
			Logger.WarnContext(r.Context(), "rate limit check failed, allowing request",
				slog.Any("error", err),
			)
			next.ServeHTTP(w, r)
			return
		}

		if !allowed {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Rate limit exceeded",
			})

			Logger.WarnContext(r.Context(), "rate limit exceeded",
				slog.String("ip", clientIP),
				slog.String("endpoint", endpoint),
			)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func Run() {
	InitCookieSecret()

	baseHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	Logger = slog.New(baseHandler)
	slog.SetDefault(Logger)

	var (
		tp *sdktrace.TracerProvider
		lp *sdklog.LoggerProvider
	)

	{
		g, _ := errgroup.WithContext(context.Background())

		g.Go(func() error {
			var err error
			tp, err = initTracer()
			if err != nil {
				log.Printf("Warning: Failed to initialize tracer: %v. Continuing without tracing", err)
			}
			return nil
		})

		g.Go(func() error {
			var err error
			lp, err = initLogProvider()
			if err != nil {
				log.Printf("Warning: Failed to initalize log provider: %v. Logs will not be forwarded via OTLP", err)
			}
			return nil
		})

		g.Wait()
	}

	if tp != nil {
		defer func() {
			if err := tp.Shutdown(context.Background()); err != nil {
				log.Printf("Error shutting down tracer provider: %v", err)
			}
		}()
	}

	if lp != nil {
		defer func() {
			if err := lp.Shutdown(context.Background()); err != nil {
				log.Printf("Error shutting down log provider: %v", err)
			}
		}()
	}

	// Initialize Redis
	if err := InitRedis(); err != nil {
		log.Printf("Warning: Failed to initialize Redis: %v. Some features may be degraded", err)
	}

	// Initialize RabbitMQ
	if err := InitRabbitMQ(); err != nil {
		log.Printf("Warning: Failed to initialize RabbitMQ: %v. Vote broadcasting and email will be degraded", err)
	} else {
		defer CloseRabbitMQ()
	}

	Logger = slog.New(newOtelSlogHandler(baseHandler))
	slog.SetDefault(Logger)

	Tracer = otel.Tracer("esc-voting-crud-api")

	router := http.NewServeMux()
	router.HandleFunc("GET /health", GetHealth)
	router.HandleFunc("GET /votes/", GetVotes)
	router.HandleFunc("GET /api/songs-with-votes", GetSongsWithPublicVotes)
	router.HandleFunc("POST /vote/", Vote)
	router.HandleFunc("GET /countries/", GetCountries)
	router.HandleFunc("GET /countryByName/{NAME}", GetCountryByName)
	router.HandleFunc("GET /songs/", HTTPGetSongs)
	router.HandleFunc("GET /songByID/{ID}", GetSongByID)
	router.HandleFunc("POST /auth/login", AuthLogin)
	router.HandleFunc("POST /auth/verify-code", AuthVerifyCode)
	router.Handle("GET /auth/verify", RequireAuth(http.HandlerFunc(AuthVerify)))
	router.Handle("POST /auth/verify", RequireAuth(http.HandlerFunc(AuthVerify)))
	router.Handle("GET /auth/me", RequireAuth(http.HandlerFunc(AuthMe)))
	router.HandleFunc("POST /auth/refresh", AuthRefresh)
	router.HandleFunc("POST /auth/logout", AuthLogout)
	router.Handle("POST /admin/open", RequireAdmin(http.HandlerFunc(OpenVote)))
	router.Handle("POST /admin/close", RequireAdmin(http.HandlerFunc(CloseVote)))
	router.Handle("DELETE /admin/deleteVotes/", RequireAdmin(http.HandlerFunc(DeleteVotes)))
	router.Handle("POST /admin/addCountry/", RequireAdmin(http.HandlerFunc(AddCountry)))
	router.Handle("POST /admin/addSong/", RequireAdmin(http.HandlerFunc(AddSong)))
	router.Handle("POST /admin/addArtist/", RequireAdmin(http.HandlerFunc(AddArtist)))
	router.Handle("POST /admin/addInterpret/", RequireAdmin(http.HandlerFunc(AddInterpret)))
	router.Handle("POST /jury/vote/", RequireJury(http.HandlerFunc(JuryVote)))
	router.Handle("GET /admin/authenticate", RequireAdmin(http.HandlerFunc(AdminLogin)))
	router.Handle("GET /jury/authenticate", RequireJury(http.HandlerFunc(JuryLogin)))
	router.Handle("POST /admin/startContest", RequireAdmin(http.HandlerFunc(StartContest)))
	router.Handle("POST /admin/advanceContest", RequireAdmin(http.HandlerFunc(AdvanceContest)))
	router.HandleFunc("GET /contest/current", GetCurrentSong)
	router.HandleFunc("GET /results", GetResults)

	router.Handle("GET /metrics/", promhttp.Handler())

	dbReadinessMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" {
				next.ServeHTTP(w, r)
				return
			}

			select {
			case <-dbReady:
				next.ServeHTTP(w, r)
			case <-r.Context().Done():
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "service is starting up, please retry",
				})
			}
		})
	}

	handler := dbReadinessMiddleware(RateLimitingMiddleware(ObservabilityMiddleware(router)))

	srv := &http.Server{
		Addr:    ":8000",
		Handler: handler,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			Logger.Error("Server failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	go func() {
		conn, dbErr := connectToDatabase(loadLocalConfig())

		if dbErr != nil {
			Logger.Error("could not connect to Database after all retries",
				slog.Any("error", dbErr),
			)
			os.Exit(1)
		}

		DB = conn
		close(dbReady)

		Logger.Info("database connection established - service fully ready")

	}()

	log.Println("Listening and Serving on Port 8000")
	Logger.Info("server starting",
		slog.Int("port", 8000),
		slog.String("service", "esc-voting-crud-api"),
	)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	Logger.Info("shutdown signal received, draining connections...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("forced shutdown %v", err)
	}

	Logger.Info("closing database")
	if DB != nil {
		DB.Close()
	}

	log.Println("server exited cleanly")
}
