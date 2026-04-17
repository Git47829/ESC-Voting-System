package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/nyaruka/phonenumbers"
	"golang.org/x/crypto/bcrypt"
	"log/slog"
	"net/http"
	"os"
	"strings"
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

	numRegion := phonenumbers.GetRegionCodeForNumber(parsed)

	return numRegion, nil
}

func HashPhoneNumber(phone string) string {
	mac := hmac.New(sha256.New, SignedCookieSecret)
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
	err := bcrypt.CompareHashAndPassword([]byte(storedToken), []byte(password))
	return err == nil
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
	adminPassword := os.Getenv("adminPassword")
	if CheckPassword(input, adminPassword) {
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

	type juryMember struct {
		mail     string
		password string
	}

	members := []juryMember{
		{strings.TrimSpace(os.Getenv("juryMail1")), os.Getenv("juryPassword1")},
		{strings.TrimSpace(os.Getenv("juryMail2")), os.Getenv("juryPassword2")},
		{strings.TrimSpace(os.Getenv("juryMail3")), os.Getenv("juryPassword3")},
	}

	for _, m := range members {
		if strings.EqualFold(email, m.mail) {
			if CheckPassword(input, m.password) {
				return true, "Authorized"
			}
			return false, "Wrong Token Provided"
		}
	}

	return false, "Invalid email"
}


func AdminLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()

	email := extractEmail(r)
	token, err := generateAndStoreToken()
	if err != nil {
		Logger.ErrorContext(ctx, "Failed to generate token", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unable to generate token"})
		return
	}

	if err := requestVerificationMail(email, token); err != nil {
		Logger.ErrorContext(ctx, "Failed to send verification mail", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unable to send verification email"})
		return
	}

	Logger.InfoContext(ctx, "Admin login: verification email sent", slog.String("email", email))
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"message": "verification email sent"})
}

func JuryLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()

	email := extractEmail(r)
	token, err := generateAndStoreToken()
	if err != nil {
		Logger.ErrorContext(ctx, "Failed to generate token", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unable to generate token"})
		return
	}

	if err := requestVerificationMail(email, token); err != nil {
		Logger.ErrorContext(ctx, "Failed to send verification mail", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unable to send verification email"})
		return
	}

	Logger.InfoContext(ctx, "Jury login: verification email sent", slog.String("email", email))
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"message": "verification email sent"})
}
