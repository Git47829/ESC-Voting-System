package main

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

	pb "esc-points-converter/proto"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// escPointTable maps 0-indexed rank to ESC televote points.
// Rank 0 = 1st place (12 pts) … rank 9 = 10th place (1 pt), rank 10+ = 0 pts.
var escPointTable = []int{12, 10, 8, 7, 6, 5, 4, 3, 2, 1}

func escPointsForRank(rank int) int {
	if rank < len(escPointTable) {
		return escPointTable[rank]
	}
	return 0
}

type Song struct {
	ID        int    `json:"songId"`
	Name      string `json:"songName"`
	Country   string `json:"country"`
	LandID    string `json:"countryId"`
	RawVotes  int    `json:"rawPublicVotes"`
	ESCPoints int    `json:"escPoints"`
	Rank      int    `json:"rank"`
}

func fetchSongs(ctx context.Context, client pb.VoteServiceClient) ([]Song, error) {
	resp, err := client.GetSongsWithVotes(ctx, &pb.GetSongsRequest{})
	if err != nil {
		return nil, fmt.Errorf("GetSongsWithVotes: %w", err)
	}

	songs := make([]Song, 0, len(resp.Songs))
	for _, s := range resp.Songs {
		songs = append(songs, Song{
			ID:       int(s.SongId),
			Name:     s.SongName,
			Country:  s.CountryName,
			LandID:   s.CountryId,
			RawVotes: int(s.PublicVotes),
		})
	}
	return songs, nil
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
		if songs[i].RawVotes > 0 {
			songs[i].ESCPoints = escPointsForRank(i)
		}
	}
	return songs
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func handlePreview(client pb.VoteServiceClient, juryScale int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		_, span := tracer.Start(ctx, "preview.fetchAndRank")
		defer span.End()

		songs, err := fetchSongs(ctx, client)
		if err != nil {
			logger.ErrorContext(ctx, "preview: failed to fetch songs", "error", err)
			span.RecordError(err)
			w.WriteHeader(http.StatusBadGateway)
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

func connectToGRPC() (*grpc.ClientConn, error) {
	host := getEnv("GRPC_HOST", "db-crud-api")
	port := getEnv("GRPC_PORT", "50051")
	target := fmt.Sprintf("%s:%s", host, port)

	const (
		maxAttempts = 20
		retryDelay  = 3 * time.Second
	)

	logger.Info("connecting to CrudAPI gRPC", slog.String("target", target))

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			logger.Warn("gRPC dial failed, retrying",
				slog.Int("attempt", attempt),
				slog.Int("max", maxAttempts),
				"error", err,
			)
			if attempt < maxAttempts {
				time.Sleep(retryDelay)
			}
			continue
		}

		// Probe with a short-timeout RPC to verify the server is ready.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		client := pb.NewVoteServiceClient(conn)
		_, probeErr := client.GetSongsWithVotes(ctx, &pb.GetSongsRequest{})
		cancel()

		if probeErr == nil {
			logger.Info("gRPC connection established", slog.Int("attempt", attempt))
			return conn, nil
		}

		conn.Close()
		logger.Warn("gRPC server not ready, retrying",
			slog.Int("attempt", attempt),
			slog.Int("max", maxAttempts),
			"error", probeErr,
		)
		if attempt < maxAttempts {
			time.Sleep(retryDelay)
		}
	}
	return nil, fmt.Errorf("could not connect to gRPC after %d attempts", maxAttempts)
}

func main() {
	baseHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger = slog.New(baseHandler)
	slog.SetDefault(logger)

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

	logger = slog.New(newOtelSlogHandler(baseHandler))
	slog.SetDefault(logger)

	tracer = otel.Tracer("esc-points-converter")

	conn, err := connectToGRPC()
	if err != nil {
		logger.Error("failed to connect to CrudAPI gRPC", "error", err)
		os.Exit(1)
	}
	defer conn.Close()
	logger.Info("CrudAPI gRPC connection established")

	grpcClient := pb.NewVoteServiceClient(conn)

	port := getEnv("PORT", "8090")

	// juryScale equalises the 50/50 jury vs televote weighting.
	// The public vote produces one ESC set (max 12 pts); multiplying by the
	// number of jury members brings it to the same maximum as the combined jury.
	juryScale := getEnvInt("NUM_JURY_MEMBERS", 3)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("GET /api/esc-points", handlePreview(grpcClient, juryScale))
	mux.Handle("GET /metrics", promhttp.Handler())

	logger.Info("ESC points converter starting",
		slog.String("port", port),
		slog.Int("jury_scale", juryScale),
		slog.String("grpc_host", getEnv("GRPC_HOST", "db-crud-api")),
		slog.String("grpc_port", getEnv("GRPC_PORT", "50051")),
		slog.String("otel_endpoint", getEnv("OTEL_EXPORTER_OTLP_HTTP_ENDPOINT", "http://otel-collector:4318")),
	)

	if err := http.ListenAndServe(":"+port, observabilityMiddleware(mux)); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}
