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

func HashPhoneNumber(phone string) string {
	mac := hmac.New(sha256.New, SignedPhoneSecret)
	mac.Write([]byte(phone))
	return hex.EncodeToString(mac.Sum(nil))
}

func HashPassword(password string) (string, error) {
	sum, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(sum), nil
}

func CheckPassword(password, storedToken string) bool {
	return bcrypt.CompareHashAndPassword([]byte(storedToken), []byte(password)) == nil
}

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
