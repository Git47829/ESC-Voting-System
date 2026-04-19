package converter

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

// escPointTable maps 0-indexed rank to ESC televote points.
// Rank 0 = 1st place (12 pts) … rank 9 = 10th place (1 pt), rank 10+ = 0 pts.
var escPointTable = []int{12, 10, 8, 7, 6, 5, 4, 3, 2, 1}

func EscPointsForRank(rank int) int {
	if rank < len(escPointTable) {
		return escPointTable[rank]
	}
	return 0
}

type Song struct {
	ID        int    `json:"songId"`
	Name      string `json:"songName"`
	Country   string `json:"country"`
	CountryID string `json:"countryId"`
	RawVotes  int    `json:"rawPublicVotes"`
	ESCPoints int    `json:"escPoints"`
	Rank      int    `json:"rank"`
}

// songAPIData matches the JSON shape returned by CRUD-DB-API GET /api/songs-with-votes.
type songAPIData struct {
	SongID      int    `json:"songId"`
	SongName    string `json:"songName"`
	CountryID   string `json:"countryId"`
	CountryName string `json:"countryName"`
	PublicVotes int    `json:"publicVotes"`
}

func fetchSongs(ctx context.Context, apiURL string) ([]Song, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL+"/api/songs-with-votes", nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch songs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch songs: status %d", resp.StatusCode)
	}

	var body struct {
		Payload []songAPIData `json:"payload"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	songs := make([]Song, 0, len(body.Payload))
	for _, s := range body.Payload {
		songs = append(songs, Song{
			ID:       s.SongID,
			Name:     s.SongName,
			Country:  s.CountryName,
			CountryID: s.CountryID,
			RawVotes: s.PublicVotes,
		})
	}
	return songs, nil
}

// RankAndConvert sorts songs by raw public votes (desc), breaks ties by song
// ID (asc) for determinism, then assigns ESC points based on final position.
func RankAndConvert(songs []Song) []Song {
	sort.SliceStable(songs, func(i, j int) bool {
		if songs[i].RawVotes != songs[j].RawVotes {
			return songs[i].RawVotes > songs[j].RawVotes
		}
		return songs[i].ID < songs[j].ID
	})
	for i := range songs {
		songs[i].Rank = i + 1
		if songs[i].RawVotes > 0 {
			songs[i].ESCPoints = EscPointsForRank(i)
		}
	}
	return songs
}

func HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func HandlePreview(apiURL string, juryScale int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		_, span := Tracer.Start(ctx, "preview.fetchAndRank")
		defer span.End()

		songs, err := fetchSongs(ctx, apiURL)
		if err != nil {
			Logger.ErrorContext(ctx, "preview: failed to fetch songs", "error", err)
			span.RecordError(err)
			w.WriteHeader(http.StatusBadGateway)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to fetch songs"})
			return
		}

		result := RankAndConvert(songs)
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

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func GetEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return fallback
}

func waitForAPI(apiURL string) error {
	const (
		maxAttempts = 20
		retryDelay  = 3 * time.Second
	)

	Logger.Info("waiting for CRUD-DB-API", slog.String("url", apiURL))

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, apiURL+"/health", nil)
		resp, err := http.DefaultClient.Do(req)
		cancel()

		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			Logger.Info("CRUD-DB-API is ready", slog.Int("attempt", attempt))
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}

		Logger.Warn("CRUD-DB-API not ready, retrying",
			slog.Int("attempt", attempt),
			slog.Int("max", maxAttempts),
		)
		if attempt < maxAttempts {
			time.Sleep(retryDelay)
		}
	}
	return fmt.Errorf("CRUD-DB-API not reachable after %d attempts", maxAttempts)
}

func Run() {
	baseHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	Logger = slog.New(baseHandler)
	slog.SetDefault(Logger)

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

	Logger = slog.New(newOtelSlogHandler(baseHandler))
	slog.SetDefault(Logger)

	Tracer = otel.Tracer("esc-points-converter")

	apiURL := getEnv("CRUD_API_URL", "http://db-crud-api:8000")
	if err := waitForAPI(apiURL); err != nil {
		Logger.Error("failed to connect to CRUD-DB-API", "error", err)
		os.Exit(1)
	}

	port := getEnv("PORT", "8090")
	jwtVerifier, err := NewJWTVerifierFromEnv()
	if err != nil {
		Logger.Error("failed to configure JWT auth", "error", err)
		os.Exit(1)
	}

	// juryScale equalises the 50/50 jury vs televote weighting.
	juryScale := GetEnvInt("NUM_JURY_MEMBERS", 3)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", HandleHealth)
	mux.Handle("GET /api/esc-points", RequireJWTAuth(jwtVerifier, "admin", "jury", "user")(HandlePreview(apiURL, juryScale)))
	mux.Handle("GET /metrics", promhttp.Handler())

	Logger.Info("ESC points converter starting",
		slog.String("port", port),
		slog.Int("jury_scale", juryScale),
		slog.String("crud_api_url", apiURL),
		slog.String("otel_endpoint", getEnv("OTEL_EXPORTER_OTLP_HTTP_ENDPOINT", "http://otel-collector:4318")),
	)

	if err := http.ListenAndServe(":"+port, observabilityMiddleware(mux)); err != nil {
		Logger.Error("server error", "error", err)
		os.Exit(1)
	}
}
