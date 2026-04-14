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

// 2FA pending verifications keyed by email
type PendingVerification struct {
	Code      string
	Role      string
	CreatedAt time.Time
}

var (
	pendingVerifications = make(map[string]*PendingVerification)
	verifyMu             sync.Mutex
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

func generate2FACode() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	num := (int(b[0])<<16 | int(b[1])<<8 | int(b[2])) % 1000000
	return fmt.Sprintf("%06d", num), nil
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

type AuthLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type AuthVerifyRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

func AuthLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	var req AuthLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	// Validate credentials based on role
	var ok bool
	var msg string
	switch req.Role {
	case "admin":
		ok, msg = CheckAccessAdmin(req.Password, req.Email)
	case "jury":
		ok, msg = CheckAccessJury(req.Password, req.Email)
	default:
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid role"})
		return
	}

	if !ok {
		Logger.WarnContext(ctx, "2FA login: credential check failed", slog.String("email", req.Email), slog.String("reason", msg))
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": msg})
		return
	}

	// Generate 6-digit code
	code, err := generate2FACode()
	if err != nil {
		Logger.ErrorContext(ctx, "Failed to generate 2FA code", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unable to generate verification code"})
		return
	}

	// Store pending verification
	verifyMu.Lock()
	pendingVerifications[req.Email] = &PendingVerification{
		Code:      code,
		Role:      req.Role,
		CreatedAt: time.Now(),
	}
	verifyMu.Unlock()

	// Send code via EuroMail
	if err := requestVerificationMail(req.Email, code); err != nil {
		Logger.ErrorContext(ctx, "Failed to send verification mail", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unable to send verification email"})
		return
	}

	Logger.InfoContext(ctx, "2FA code sent", slog.String("email", req.Email), slog.String("role", req.Role))
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"message": "Verification code sent"})
}

func AuthVerify(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	var req AuthVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	verifyMu.Lock()
	pending, exists := pendingVerifications[req.Email]
	if exists && pending.Code == req.Code && time.Since(pending.CreatedAt) <= 5*time.Minute {
		delete(pendingVerifications, req.Email)
		// Clean up expired entries
		for k, v := range pendingVerifications {
			if time.Since(v.CreatedAt) > 5*time.Minute {
				delete(pendingVerifications, k)
			}
		}
		verifyMu.Unlock()

		Logger.InfoContext(ctx, "2FA verified", slog.String("email", req.Email))
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{"message": "Verified"})
		return
	}
	verifyMu.Unlock()

	Logger.WarnContext(ctx, "2FA verification failed", slog.String("email", req.Email))
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{"error": "Invalid or expired verification code"})
}
