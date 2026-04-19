package server

import (
	"context"
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

type authContextKey string

const authClaimsContextKey authContextKey = "auth_claims"

func extractAccessToken(r *http.Request) string {
	if cookie, err := r.Cookie(accessTokenCookieName); err == nil && strings.TrimSpace(cookie.Value) != "" {
		return strings.TrimSpace(cookie.Value)
	}
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	}
	return ""
}

func claimsFromContext(ctx context.Context) (*authClaims, bool) {
	claims, ok := ctx.Value(authClaimsContextKey).(*authClaims)
	return claims, ok
}

func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		token := extractAccessToken(r)
		if token == "" {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{"error": "Authentication required"})
			return
		}

		claims, err := parseJWTToken(token, tokenTypeAccess)
		if err != nil {
			Logger.WarnContext(r.Context(), "access token rejected", slog.Any("error", err))
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid access token"})
			return
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), authClaimsContextKey, claims)))
	})
}

func RequireRole(role string, next http.Handler) http.Handler {
	return RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		claims, ok := claimsFromContext(r.Context())
		if !ok {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{"error": "Authentication required"})
			return
		}
		if claims.Role != role {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{"error": "Insufficient permissions"})
			return
		}
		next.ServeHTTP(w, r)
	}))
}

func RequireJury(next http.Handler) http.Handler {
	return RequireRole("jury", next)
}

func RequireAdmin(next http.Handler) http.Handler {
	return RequireRole("admin", next)
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

func CheckAccessUser(input, email string) (bool, string) {
	if input == "" {
		return false, "Token has to be provided"
	}
	if email == "" {
		return false, "Email has to be provided"
	}
	configuredEmail := strings.TrimSpace(os.Getenv("userMail"))
	configuredPassword := os.Getenv("userPassword")
	if configuredEmail == "" || configuredPassword == "" {
		return false, "User login is not configured"
	}
	if !strings.EqualFold(email, configuredEmail) {
		return false, "Invalid email"
	}
	if CheckPassword(input, configuredPassword) {
		return true, "Authorized"
	}
	return false, "Wrong Token Provided"
}

func AdminLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	claims, ok := claimsFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "Authentication required"})
		return
	}
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]any{"message": "authorized", "email": claims.Subject, "role": claims.Role})
}

func JuryLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	claims, ok := claimsFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "Authentication required"})
		return
	}
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]any{"message": "authorized", "email": claims.Subject, "role": claims.Role})
}
