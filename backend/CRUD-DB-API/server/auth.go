package server

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

func requestVerificationMail(email, token string) error {
	if err := PublishEmailJob(email, token); err != nil {
		Logger.Error("Failed to publish email job to RabbitMQ", slog.Any("error", err))
		return err
	}
	Logger.Info("Verification mail job published", slog.String("email", email))
	return nil
}

func RequestToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	Logger.InfoContext(ctx, "New Verification Token requested")

	w.Header().Set("Content-Type", "application/json")
	token, err := generateToken()
	if err != nil {
		Logger.ErrorContext(ctx, "Invalid Token generation", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Unable to generate Token",
		})
		return
	}

	if err := StoreToken(ctx, token, 5*time.Minute); err != nil {
		Logger.ErrorContext(ctx, "Failed to store token in Redis", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Unable to store Token",
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

	exists, err := CheckAndEvictToken(ctx, providedToken)
	if err != nil {
		Logger.ErrorContext(ctx, "Failed to check token in Redis", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Token verification failed",
		})
		return
	}

	if exists {
		Logger.InfoContext(ctx, "New Token verified, evicted from Redis", slog.String("message:", "New Verification via Token"))
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

	code, err := generate2FACode()
	if err != nil {
		Logger.ErrorContext(ctx, "Failed to generate 2FA code", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unable to generate verification code"})
		return
	}

	if err := StorePendingVerification(ctx, req.Email, code, req.Role); err != nil {
		Logger.ErrorContext(ctx, "Failed to store pending verification in Redis", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unable to store verification"})
		return
	}

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

	code, _, createdAt, exists, err := GetAndDeletePendingVerification(ctx, req.Email)
	if err != nil {
		Logger.ErrorContext(ctx, "Failed to check pending verification in Redis", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Verification check failed"})
		return
	}

	if exists && subtle.ConstantTimeCompare([]byte(code), []byte(req.Code)) == 1 && time.Since(time.Unix(createdAt, 0)) <= 5*time.Minute {
		Logger.InfoContext(ctx, "2FA verified", slog.String("email", req.Email))
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{"message": "Verified"})
		return
	}

	Logger.WarnContext(ctx, "2FA verification failed", slog.String("email", req.Email))
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{"error": "Invalid or expired verification code"})
}
