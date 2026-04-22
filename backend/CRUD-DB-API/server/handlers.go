package server

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"log"
	"log/slog"
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

const totalVotePoints = 20

// SignedCookieSecret and SignedPhoneSecret are package-level vars retained for
// test compatibility (tests read/set them directly).
var SignedCookieSecret []byte
var SignedPhoneSecret []byte

// InitCookieSecret derives or generates the cookie signing secret and stores it
// in SignedCookieSecret. Called by tests and by Run().
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

// InitPhoneSecret derives the phone signing secret from env and stores it in
// SignedPhoneSecret. Called by Run().
func InitPhoneSecret() {
	if key := os.Getenv("PHONESIGNINGSECRET"); key != "" {
		sum := sha256.Sum256([]byte("phone-secret:" + key))
		SignedPhoneSecret = sum[:]
	}
}

// EncodeCookieValue marshals state and appends an HMAC-SHA256 signature.
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

// ---------------------------------------------------------------------------
// Handlers struct — dependency-injected handler container (DIP / ISP)
// ---------------------------------------------------------------------------

// Handlers holds all dependencies for HTTP handler methods.
type Handlers struct {
	votes    VoteRepository
	songs    SongRepository
	auth     AuthService
	notifier VoteNotifier
	cfg      AppConfig
}

// NewHandlers constructs a Handlers with the provided database, notifier, and config.
func NewHandlers(db *sql.DB, notifier VoteNotifier, cfg AppConfig) *Handlers {
	return &Handlers{
		votes:    newMySQLVoteRepository(db),
		songs:    newMySQLSongRepository(db),
		auth:     &authService{},
		notifier: notifier,
		cfg:      cfg,
	}
}

// ---------------------------------------------------------------------------
// Run — server entry point
// ---------------------------------------------------------------------------

func Run() {
	InitCookieSecret()
	InitPhoneSecret()

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

	cfg := LoadConfig()
	// Sync package-level secrets with what LoadConfig derived, so that
	// EncodeCookieValue (which reads the globals) stays consistent.
	SignedCookieSecret = cfg.CookieSecret
	SignedPhoneSecret = cfg.PhoneSecret

	// h is set in the DB goroutine before dbReady is closed.
	// dbReadinessMiddleware blocks all non-health requests until then.
	var h *Handlers

	router := http.NewServeMux()
	router.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) { h.ServeGetHealth(w, r) })
	router.HandleFunc("GET /votes/", func(w http.ResponseWriter, r *http.Request) { h.ServeGetVotes(w, r) })
	router.HandleFunc("POST /vote/", func(w http.ResponseWriter, r *http.Request) { h.ServeVote(w, r) })
	router.HandleFunc("GET /countries/", func(w http.ResponseWriter, r *http.Request) { h.ServeGetCountries(w, r) })
	router.HandleFunc("GET /countryByName/{NAME}", func(w http.ResponseWriter, r *http.Request) { h.ServeGetCountryByName(w, r) })
	router.HandleFunc("GET /songs/", func(w http.ResponseWriter, r *http.Request) { h.ServeGetSongs(w, r) })
	router.HandleFunc("GET /songByID/{ID}", func(w http.ResponseWriter, r *http.Request) { h.ServeGetSongByID(w, r) })
	router.HandleFunc("GET /auth/requestToken", func(w http.ResponseWriter, r *http.Request) { h.ServeRequestToken(w, r) })
	router.HandleFunc("GET /auth/verifyToken/{token}", func(w http.ResponseWriter, r *http.Request) { h.ServeVerifyToken(w, r) })
	router.HandleFunc("POST /auth/login", func(w http.ResponseWriter, r *http.Request) { h.ServeAuthLogin(w, r) })
	router.HandleFunc("POST /auth/verify", func(w http.ResponseWriter, r *http.Request) { h.ServeAuthVerify(w, r) })
	router.Handle("POST /admin/open", RequireAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { h.ServeOpenVote(w, r) })))
	router.Handle("POST /admin/close", RequireAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { h.ServeCloseVote(w, r) })))
	router.Handle("DELETE /admin/deleteVotes/", RequireAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { h.ServeDeleteVotes(w, r) })))
	router.Handle("POST /admin/addCountry/", RequireAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { h.ServeAddCountry(w, r) })))
	router.Handle("POST /admin/addSong/", RequireAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { h.ServeAddSong(w, r) })))
	router.Handle("POST /admin/addArtist/", RequireAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { h.ServeAddArtist(w, r) })))
	router.Handle("POST /admin/addInterpret/", RequireAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { h.ServeAddInterpret(w, r) })))
	router.Handle("POST /jury/vote/", RequireJury(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { h.ServeJuryVote(w, r) })))
	router.Handle("GET /admin/authenticate", RequireAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { h.ServeAdminLogin(w, r) })))
	router.Handle("GET /jury/authenticate", RequireJury(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { h.ServeJuryLogin(w, r) })))
	router.Handle("POST /admin/startContest", RequireAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { h.ServeStartContest(w, r) })))
	router.Handle("POST /admin/advanceContest", RequireAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { h.ServeAdvanceContest(w, r) })))
	router.HandleFunc("GET /contest/current", func(w http.ResponseWriter, r *http.Request) { h.ServeGetCurrentSong(w, r) })
	router.Handle("GET /metrics/", promhttp.Handler())

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
		conn, dbErr := connectToDatabase(cfg)
		if dbErr != nil {
			Logger.Error("could not connect to Database after all retries",
				slog.Any("error", dbErr),
			)
			os.Exit(1)
		}

		DB = conn

		notifier, err := StartGRPCServer(DB, cfg.GRPCPort)
		if err != nil {
			Logger.Error("Failed to start gRPC server", slog.String("error", err.Error()))
			os.Exit(1)
		}

		h = NewHandlers(DB, notifier, cfg)
		close(dbReady)

		Logger.Info("database connection established - service fully ready")
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
