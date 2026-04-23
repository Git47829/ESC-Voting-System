package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

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

var (
	clients = make(map[string]*Client)
	rateMu  sync.Mutex
)

var rateLimitConfigs = map[string]RateLimitConfig{
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

func cleanupClients() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		rateMu.Lock()
		for key, c := range clients {
			if time.Since(c.lastSeen) > 10*time.Minute {
				delete(clients, key)
			}
		}
		rateMu.Unlock()
	}
}

func getCLientLimiter(ip, endpoint string) *rate.Limiter {
	rateMu.Lock()
	defer rateMu.Unlock()

	key := ip + "::" + endpoint
	if client, exists := clients[key]; exists {
		client.lastSeen = time.Now()
		return client.limiter
	}

	config, exists := rateLimitConfigs[endpoint]
	if !exists {
		config = RateLimitConfig{RequestsPerSecond: 10, BurstSize: 20}
	}

	limiter := rate.NewLimiter(rate.Limit(config.RequestsPerSecond), config.BurstSize)
	clients[key] = &Client{limiter: limiter, lastSeen: time.Now()}
	return limiter
}

func getClientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return ip
}

func RateLimitingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		endpoint := fmt.Sprintf("%s %s", r.Method, r.URL.Path)
		clientIP := getClientIP(r)

		if !getCLientLimiter(clientIP, endpoint).Allow() {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]string{"error": "Rate limit exceeded"})
			Logger.WarnContext(r.Context(), "rate limit exceeded",
				slog.String("ip", clientIP),
				slog.String("endpoint", endpoint),
			)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func dbReadinessMiddleware(next http.Handler) http.Handler {
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
