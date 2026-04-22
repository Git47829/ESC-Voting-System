package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/nyaruka/phonenumbers"
	"golang.org/x/crypto/bcrypt"
)

func extractToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return ""
}

func extractEmail(r *http.Request) string {
	return strings.TrimSpace(r.Header.Get("X-Email"))
}

// RequireJury is middleware that enforces jury authentication.
// Signature unchanged for test compatibility.
func RequireJury(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		email := extractEmail(r)
		if ok, msg := CheckAccessJury(token, email); !ok {
			Logger.Warn("Invalid Jury Login Attempt", slog.String("message", "Invalid Login Attempt"))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{"error": msg})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireAdmin is middleware that enforces admin authentication.
// Signature unchanged for test compatibility.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		email := extractEmail(r)
		if ok, msg := CheckAccessAdmin(token, email); !ok {
			Logger.Warn("Invalid Login Attempt", slog.String("message", "Invalid Login Attempt"))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{"error": msg})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// CheckPhoneNum parses and validates a phone number, returning the region code.
func CheckPhoneNum(num string) (string, error) {
	parsed, err := phonenumbers.Parse(num, "")
	if err != nil {
		return "", fmt.Errorf("could not parse phone number: %w", err)
	}
	if !phonenumbers.IsValidNumber(parsed) {
		return "", fmt.Errorf("invalid phone number")
	}
	return phonenumbers.GetRegionCodeForNumber(parsed), nil
}

// HashPhoneNumber returns an HMAC-SHA256 hex digest of the phone number.
func HashPhoneNumber(phone string) string {
	mac := hmac.New(sha256.New, SignedPhoneSecret)
	mac.Write([]byte(phone))
	return hex.EncodeToString(mac.Sum(nil))
}

// HashPassword bcrypt-hashes a plaintext password.
func HashPassword(password string) (string, error) {
	sum, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(sum), nil
}

// CheckPassword reports whether plaintext matches a bcrypt hash.
func CheckPassword(password, storedToken string) bool {
	return bcrypt.CompareHashAndPassword([]byte(storedToken), []byte(password)) == nil
}

// CheckAccessAdmin validates admin credentials against environment variables.
// Kept as a package-level function for test compatibility.
func CheckAccessAdmin(input, email string) (bool, string) {
	if input == "" {
		return false, "Token has to be provided"
	}
	if email == "" {
		return false, "Email has to be provided"
	}
	adminMail := os.Getenv("adminMail")
	if !strings.EqualFold(email, strings.TrimSpace(adminMail)) {
		return false, "Invalid email"
	}
	if CheckPassword(input, os.Getenv("adminPassword")) {
		return true, "Authorized"
	}
	return false, "Wrong Token provided"
}

// CheckAccessJury validates jury credentials against environment variables.
// Reads juryMail1..N / juryPassword1..N dynamically, enabling adding jury
// members via env vars without code changes (OCP).
// Kept as a package-level function for test compatibility.
func CheckAccessJury(input, email string) (bool, string) {
	if input == "" {
		return false, "Token has to be provided"
	}
	if email == "" {
		return false, "Email has to be provided"
	}

	for i := 1; ; i++ {
		mail := strings.TrimSpace(os.Getenv(fmt.Sprintf("juryMail%d", i)))
		if mail == "" {
			break
		}
		pass := os.Getenv(fmt.Sprintf("juryPassword%d", i))
		if strings.EqualFold(email, mail) {
			if CheckPassword(input, pass) {
				return true, "Authorized"
			}
			return false, "Wrong Token Provided"
		}
	}

	return false, "Invalid email"
}

// ---------------------------------------------------------------------------
// Package-level shims for AdminLogin / JuryLogin
// ---------------------------------------------------------------------------

func (h *Handlers) ServeAdminLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	email := extractEmail(r)
	token, err := h.auth.GenerateAndStoreToken()
	if err != nil {
		Logger.ErrorContext(ctx, "Failed to generate token", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unable to generate token"})
		return
	}

	if err := h.auth.SendVerificationMail(email, token); err != nil {
		Logger.ErrorContext(ctx, "Failed to send verification mail", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unable to send verification email"})
		return
	}

	Logger.InfoContext(ctx, "Admin login: verification email sent", slog.String("email", email))
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"message": "verification email sent"})
}

func (h *Handlers) ServeJuryLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	email := extractEmail(r)
	token, err := h.auth.GenerateAndStoreToken()
	if err != nil {
		Logger.ErrorContext(ctx, "Failed to generate token", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unable to generate token"})
		return
	}

	if err := h.auth.SendVerificationMail(email, token); err != nil {
		Logger.ErrorContext(ctx, "Failed to send verification mail", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unable to send verification email"})
		return
	}

	Logger.InfoContext(ctx, "Jury login: verification email sent", slog.String("email", email))
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"message": "verification email sent"})
}
