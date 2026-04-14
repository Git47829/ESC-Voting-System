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
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
)

type Client struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type RateLimitConfig struct {
	RequestsPerSecond float64
	BurstSize         int
}

var clients = make(map[string]*Client)

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

var (
	rateLimitConfigs = map[string]RateLimitConfig{
		"GET /health":          {RequestsPerSecond: 100, BurstSize: 100},
		"GET /votes/":          {RequestsPerSecond: 10, BurstSize: 20},
		"GET /countries/":      {RequestsPerSecond: 10, BurstSize: 20},
		"GET /songs/":          {RequestsPerSecond: 10, BurstSize: 20},
		"GET /songByID/{ID}":   {RequestsPerSecond: 10, BurstSize: 20},
		"GET /contest/current": {RequestsPerSecond: 10, BurstSize: 20},
		"GET /auth/requestToken":        {RequestsPerSecond: 1, BurstSize: 1},
		"GET /auth/verifyToken/{token}": {RequestsPerSecond: 1, BurstSize: 1},
		"POST /auth/login":              {RequestsPerSecond: 1, BurstSize: 3},
		"POST /auth/verify":             {RequestsPerSecond: 1, BurstSize: 5},

		"POST /vote/":             {RequestsPerSecond: 1, BurstSize: 1},
		"POST /jury/vote":         {RequestsPerSecond: 5, BurstSize: 5},
		"GET /jury/authenticate":  {RequestsPerSecond: 1, BurstSize: 1},
		"GET /admin/authenticate": {RequestsPerSecond: 1, BurstSize: 1},

		"POST /admin/open":           {RequestsPerSecond: 2, BurstSize: 2},
		"POST /admin/close":          {RequestsPerSecond: 2, BurstSize: 2},
		"POST /admin/addCountry":     {RequestsPerSecond: 5, BurstSize: 5},
		"POST /admin/addSong":        {RequestsPerSecond: 5, BurstSize: 5},
		"POST /admin/addArtist":      {RequestsPerSecond: 5, BurstSize: 5},
		"POST /admin/addInterpret":   {RequestsPerSecond: 5, BurstSize: 5},
		"POST /admin/startContest":   {RequestsPerSecond: 5, BurstSize: 5},
		"POST /admin/advanceContest": {RequestsPerSecond: 5, BurstSize: 5},
		"DELETE /admin/deleteVotes":  {RequestsPerSecond: 1, BurstSize: 1},

		"GET /metrics/": {RequestsPerSecond: 10000, BurstSize: 10000},
	}
)

var (
	usedTokens = make(map[string]bool)
	mu         sync.Mutex
)

const totalVotePoints = 20

type CookieVoteState struct {
	VotesRemaining int            `json:"votes_remaining"`
	VotesCast      map[string]int `json:"votes_cast"`
}

var SignedCookieSecret []byte

func cleanupClients() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		mu.Lock()
		for key, c := range clients {
			if time.Since(c.lastSeen) > 10*time.Minute {
				delete(clients, key)
			}
		}
		mu.Unlock()
	}
}

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

		limiter := getCLientLimiter(clientIP, endpoint)

		if !limiter.Allow() {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Rate limit exceeded",
			})

			Logger.WarnContext(r.Context(), "rate limit exeeded",
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

	go cleanupClients()

	Logger = slog.New(newOtelSlogHandler(baseHandler))
	slog.SetDefault(Logger)

	Tracer = otel.Tracer("esc-voting-crud-api")

	router := http.NewServeMux()
	router.HandleFunc("GET /health", GetHealth)
	router.HandleFunc("GET /votes/", GetVotes)
	router.HandleFunc("POST /vote/", Vote)
	router.HandleFunc("GET /countries/", GetCountries)
	router.HandleFunc("GET /countryByName/{NAME}", GetCountryByName)
	router.HandleFunc("GET /songs/", HTTPGetSongs)
	router.HandleFunc("GET /songByID/{ID}", GetSongByID)
	router.HandleFunc("GET /auth/requestToken", RequestToken)
	router.HandleFunc("GET /auth/verifyToken/{token}", VerifiyWithToken)
	router.HandleFunc("POST /auth/login", AuthLogin)
	router.HandleFunc("POST /auth/verify", AuthVerify)
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

		voteService, err := StartGRPCServer(DB, "50051")
		if err != nil {
			Logger.Error("Failed to start gRPC server", slog.String("error", err.Error()))
			os.Exit(1)
		}
		SetGlobalVoteServer(voteService)
		Logger.Info("gRPC vote streaming service initialized")
	}()

	log.Println("Listening and Serving on Port 8000")
	Logger.Info("server starting",
		slog.Int("port", 8000),
		slog.String("service", "esc-voting-crud-api"),
	)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	Logger.Info("shutdown singal recieved, draining connections...")

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
