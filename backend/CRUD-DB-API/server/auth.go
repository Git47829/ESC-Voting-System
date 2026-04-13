package server

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
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

func requestVerificationMail(email, token string) error {
	reqURL := os.Getenv("EuroMailURL")
	body, _ := json.Marshal(map[string]string{"email": email, "token": token})
	resp, err := http.Post(reqURL, "application/json", bytes.NewReader(body))
	if err != nil {
		Logger.Error("Unable to connect to EuroMail", slog.Any("error", err))
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		Logger.Error("EuroMail returned non-202", slog.Int("status", resp.StatusCode))
		return fmt.Errorf("euromail status %d", resp.StatusCode)
	}

	Logger.Info("Verification mail sent", slog.String("email", email))
	return nil
}

func RequestToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	Logger.InfoContext(ctx, "New Verification Token requested")

	w.Header().Set("Content-Type", "application/json")
	token, err := generateAndStoreToken()
	if err != nil {
		Logger.ErrorContext(ctx, "Invalid Token generation", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Unable to generate Token",
		})
		return
	}

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

func generateToken() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	return hex.EncodeToString(b), err
}

func generateAndStoreToken() (string, error) {
	token, err := generateToken()
	if err != nil {
		return "", err
	}
	tokenMu.Lock()
	storedTokens[token] = time.Now()
	tokenMu.Unlock()
	return token, nil
}

func evictToken(input string) {
	delete(storedTokens, input)
	for k, v := range storedTokens {
		if time.Since(v) > 5*time.Minute {
			delete(storedTokens, k)
		}
	}
}
