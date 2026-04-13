package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"
)

var (
	storedTokens = make(map[string]time.Time)
	tokenMu      sync.Mutex
)

func requestVerificationMail() {
	reqURL := os.Getenv("EuroMailURL")
	resp, err := http.Post(reqURL, "", nil)
	if err != nil {
		Logger.Error("Unable to Conntect to EuroMail", slog.Any("error:", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusAccepted {
		Logger.Info("Successfully requested Verification Mail")
		return
	} else {
		Logger.Error("Error requesting Verification Mail")
		return
	}
}

func RequestToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	Logger.InfoContext(ctx, "New Verification Token requested")

	w.Header().Set("Content-Type", "application/json")
	token, err := generateToken()
	if err != nil {
		Logger.ErrorContext(ctx, "Invaild Token generation", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Unable to generate Token",
		})
	}

	tokenMu.Lock()
	storedTokens[token] = time.Now()
	tokenMu.Unlock()

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "authorized",
		"token":   token,
	})
}

func VerifiyWithToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	providedToken := r.PathValue("token")

	w.Header().Set("Content-Type", "application/json")

	tokenMu.Lock()
	_, exists := storedTokens[providedToken]
	if exists {
		evictToken(providedToken)
	}
	tokenMu.Unlock()

	if exists {
		Logger.InfoContext(ctx, "New Token verified, evicting from TokenStore", slog.String("message:", "New Verification via Token"))
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "authenticated",
		})
		return
	}

	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{
		"messsage": "Invalid Token provided",
	})
}

// called with tokenMu held
func evictToken(input string) {
	delete(storedTokens, input)
	for k, v := range storedTokens {
		if time.Since(v) > 5*time.Minute {
			delete(storedTokens, k)
		}
	}
}
