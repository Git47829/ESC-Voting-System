package server

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	accessTokenCookieName  = "esc_access_token"
	refreshTokenCookieName = "esc_refresh_token"
	tokenTypeAccess        = "access"
	tokenTypeRefresh       = "refresh"
)

var escPointTable = []int{12, 10, 8, 7, 6, 5, 4, 3, 2, 1}

type AuthLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type authClaims struct {
	Role      string `json:"role"`
	TokenType string `json:"token_type"`
	SessionID string `json:"sid,omitempty"`
	jwt.RegisteredClaims
}

type refreshSession struct {
	Email     string
	Role      string
	JTI       string
	ExpiresAt time.Time
}

type authConfig struct {
	secret        []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
	secureCookies bool
	juryScale     int
}

var (
	authCfgOnce        sync.Once
	cachedAuthConfig   authConfig
	refreshSessionsMu  sync.RWMutex
	refreshSessionData = map[string]refreshSession{}
)

func isTest2FABypassEnabled() bool {
	if !strings.HasSuffix(os.Args[0], ".test") {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(os.Getenv("AUTH_TEST_BYPASS_2FA")), "true")
}

func getTest2FACode() string {
	if !strings.HasSuffix(os.Args[0], ".test") {
		return ""
	}
	return strings.TrimSpace(os.Getenv("AUTH_FIXED_2FA_CODE"))
}

func getAuthConfig() authConfig {
	authCfgOnce.Do(func() {
		cachedAuthConfig = authConfig{
			secret:        loadJWTSecret(),
			accessTTL:     readPositiveDuration("ACCESS_TOKEN_TTL_MINUTES", 15, time.Minute),
			refreshTTL:    readPositiveDuration("REFRESH_TOKEN_TTL_HOURS", 168, time.Hour),
			secureCookies: !strings.EqualFold(strings.TrimSpace(os.Getenv("AUTH_COOKIE_SECURE")), "false"),
			juryScale:     readPositiveInt("NUM_JURY_MEMBERS", 3),
		}
	})
	return cachedAuthConfig
}

func readPositiveDuration(envKey string, fallback int, unit time.Duration) time.Duration {
	value := readPositiveInt(envKey, fallback)
	return time.Duration(value) * unit
}

func readPositiveInt(envKey string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(envKey))
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func loadJWTSecret() []byte {
	if raw := strings.TrimSpace(os.Getenv("JWT_SECRET")); raw != "" {
		sum := sha256Digest("jwt-secret:" + raw)
		return sum
	}
	if raw := strings.TrimSpace(os.Getenv("COOKIESIGNINGKEY")); raw != "" {
		sum := sha256Digest("jwt-cookie-secret:" + raw)
		return sum
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		panic("cannot generate JWT secret: " + err.Error())
	}
	Logger.Warn("JWT_SECRET and COOKIESIGNINGKEY missing: generated ephemeral JWT key")
	return secret
}

func sha256Digest(input string) []byte {
	sum := sha256.Sum256([]byte(input))
	return sum[:]
}

func generateTokenID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func isSupportedRole(role string) bool {
	switch role {
	case "admin", "jury", "user":
		return true
	default:
		return false
	}
}

func issueJWT(email, role, tokenType, sessionID string, ttl time.Duration) (string, string, error) {
	if !isSupportedRole(role) {
		return "", "", errors.New("unsupported role")
	}
	now := time.Now()
	jti, err := generateTokenID()
	if err != nil {
		return "", "", err
	}
	claims := authClaims{
		Role:      role,
		TokenType: tokenType,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   email,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			ID:        jti,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(getAuthConfig().secret)
	if err != nil {
		return "", "", err
	}
	return signed, jti, nil
}

func parseJWTToken(raw, expectedType string) (*authClaims, error) {
	claims := &authClaims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("invalid signing method")
		}
		return getAuthConfig().secret, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	if claims.TokenType != expectedType {
		return nil, errors.New("invalid token type")
	}
	if claims.Subject == "" || !isSupportedRole(claims.Role) {
		return nil, errors.New("invalid claims")
	}
	return claims, nil
}

func persistRefreshSession(sessionID string, session refreshSession, ttl time.Duration) error {
	ctx := context.Background()
	if redisAvailable() {
		key := "auth:refresh_session:" + sessionID
		pipe := RedisClient.Pipeline()
		pipe.HSet(ctx, key, map[string]any{
			"email":      session.Email,
			"role":       session.Role,
			"jti":        session.JTI,
			"expires_at": session.ExpiresAt.Unix(),
		})
		pipe.Expire(ctx, key, ttl)
		_, err := pipe.Exec(ctx)
		if err != nil {
			return err
		}
		return nil
	}

	refreshSessionsMu.Lock()
	refreshSessionData[sessionID] = session
	refreshSessionsMu.Unlock()
	return nil
}

func readRefreshSession(sessionID string) (refreshSession, bool, error) {
	ctx := context.Background()
	if redisAvailable() {
		values, err := RedisClient.HGetAll(ctx, "auth:refresh_session:"+sessionID).Result()
		if err != nil {
			return refreshSession{}, false, err
		}
		if len(values) == 0 {
			return refreshSession{}, false, nil
		}

		expiresUnix, err := strconv.ParseInt(values["expires_at"], 10, 64)
		if err != nil {
			return refreshSession{}, false, err
		}

		session := refreshSession{
			Email:     values["email"],
			Role:      values["role"],
			JTI:       values["jti"],
			ExpiresAt: time.Unix(expiresUnix, 0),
		}
		if session.ExpiresAt.Before(time.Now()) {
			deleteRefreshSession(sessionID)
			return refreshSession{}, false, nil
		}
		return session, true, nil
	}

	refreshSessionsMu.RLock()
	session, ok := refreshSessionData[sessionID]
	refreshSessionsMu.RUnlock()
	if !ok {
		return refreshSession{}, false, nil
	}
	if session.ExpiresAt.Before(time.Now()) {
		deleteRefreshSession(sessionID)
		return refreshSession{}, false, nil
	}
	return session, true, nil
}

func deleteRefreshSession(sessionID string) error {
	ctx := context.Background()
	if redisAvailable() {
		return RedisClient.Del(ctx, "auth:refresh_session:"+sessionID).Err()
	}
	refreshSessionsMu.Lock()
	delete(refreshSessionData, sessionID)
	refreshSessionsMu.Unlock()
	return nil
}

func setAuthCookies(w http.ResponseWriter, accessToken, refreshToken string) {
	cfg := getAuthConfig()
	setAccessCookie(w, accessToken, cfg)
	http.SetCookie(w, &http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    refreshToken,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   cfg.secureCookies,
		MaxAge:   int(cfg.refreshTTL.Seconds()),
		Expires:  time.Now().Add(cfg.refreshTTL),
	})
}

func setAccessCookie(w http.ResponseWriter, accessToken string, cfg authConfig) {
	http.SetCookie(w, &http.Cookie{
		Name:     accessTokenCookieName,
		Value:    accessToken,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   cfg.secureCookies,
		MaxAge:   int(cfg.accessTTL.Seconds()),
		Expires:  time.Now().Add(cfg.accessTTL),
	})
}

func clearAuthCookies(w http.ResponseWriter) {
	cfg := getAuthConfig()
	for _, name := range []string{accessTokenCookieName, refreshTokenCookieName} {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   cfg.secureCookies,
			MaxAge:   -1,
			Expires:  time.Unix(0, 0),
		})
	}
}

func writeAuthSuccess(w http.ResponseWriter, claims *authClaims) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message":       "authenticated",
		"authenticated": true,
		"user": map[string]string{
			"email": claims.Subject,
			"role":  claims.Role,
		},
	})
}

func generate2FACode() (string, error) {
	fixedCode := strings.TrimSpace(os.Getenv("AUTH_FIXED_2FA_CODE"))
	if fixedCode != "" {
		if len(fixedCode) != 6 {
			return "", fmt.Errorf("AUTH_FIXED_2FA_CODE must be a 6-digit number")
		}
		if _, err := strconv.Atoi(fixedCode); err != nil {
			return "", fmt.Errorf("AUTH_FIXED_2FA_CODE must be a 6-digit number")
		}
		return fixedCode, nil
	}

	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	code := int(b[0])<<16 | int(b[1])<<8 | int(b[2])
	return strconv.Itoa(100000 + code%900000), nil
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

	req.Email = strings.TrimSpace(req.Email)
	req.Role = strings.TrimSpace(req.Role)
	if req.Email == "" || req.Password == "" || !isSupportedRole(req.Role) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{"error": "Email, password, and valid role are required"})
		return
	}

	var ok bool
	var msg string
	switch req.Role {
	case "admin":
		ok, msg = CheckAccessAdmin(req.Password, req.Email)
	case "jury":
		ok, msg = CheckAccessJury(req.Password, req.Email)
	case "user":
		ok, msg = CheckAccessUser(req.Password, req.Email)
	}
	if !ok {
		Logger.WarnContext(ctx, "login failed", slog.String("email", req.Email), slog.String("role", req.Role), slog.String("reason", msg))
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": msg})
		return
	}

	if isTest2FABypassEnabled() {
		sessionID, err := generateTokenID()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create session"})
			return
		}
		cfg := getAuthConfig()
		accessToken, _, err := issueJWT(req.Email, req.Role, tokenTypeAccess, sessionID, cfg.accessTTL)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to issue access token"})
			return
		}
		refreshToken, refreshJTI, err := issueJWT(req.Email, req.Role, tokenTypeRefresh, sessionID, cfg.refreshTTL)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to issue refresh token"})
			return
		}

		session := refreshSession{
			Email:     req.Email,
			Role:      req.Role,
			JTI:       refreshJTI,
			ExpiresAt: time.Now().Add(cfg.refreshTTL),
		}
		if err := persistRefreshSession(sessionID, session, cfg.refreshTTL); err != nil {
			Logger.ErrorContext(ctx, "failed to persist refresh session", slog.Any("error", err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to persist refresh session"})
			return
		}

		setAuthCookies(w, accessToken, refreshToken)
		writeAuthSuccess(w, &authClaims{Role: req.Role, RegisteredClaims: jwt.RegisteredClaims{Subject: req.Email}})
		return
	}

	code := getTest2FACode()
	if code == "" {
		var err error
		code, err = generate2FACode()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to generate verification code"})
			return
		}
	}

	if err := StorePendingVerification(ctx, req.Email, code, req.Role); err != nil {
		Logger.ErrorContext(ctx, "failed to store pending verification", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to store verification"})
		return
	}

	if err := PublishEmailJob(req.Email, code); err != nil {
		Logger.ErrorContext(ctx, "failed to publish email job", slog.Any("error", err))
		// Don't fail the login — code is in Redis, user can retry
	}

	Logger.InfoContext(ctx, "2FA code sent", slog.String("email", req.Email), slog.String("role", req.Role))
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Verification code sent to your email"})
}

type AuthVerifyCodeRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
	Role  string `json:"role"`
}

func AuthVerifyCode(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	var req AuthVerifyCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	req.Code = strings.TrimSpace(req.Code)
	req.Role = strings.TrimSpace(req.Role)
	if req.Email == "" || req.Code == "" || !isSupportedRole(req.Role) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{"error": "Email, code, and valid role are required"})
		return
	}

	storedCode, storedRole, createdAt, exists, err := GetAndDeletePendingVerification(ctx, req.Email)
	if err != nil {
		Logger.ErrorContext(ctx, "failed to retrieve pending verification", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Verification lookup failed"})
		return
	}
	if !exists {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "No pending verification found. Please login again."})
		return
	}

	if time.Since(time.Unix(createdAt, 0)) > 5*time.Minute {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "Verification code expired"})
		return
	}

	if subtle.ConstantTimeCompare([]byte(storedCode), []byte(req.Code)) != 1 {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid verification code"})
		return
	}

	if storedRole != req.Role {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "Role mismatch"})
		return
	}

	sessionID, err := generateTokenID()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create session"})
		return
	}

	cfg := getAuthConfig()
	accessToken, _, err := issueJWT(req.Email, req.Role, tokenTypeAccess, sessionID, cfg.accessTTL)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to issue access token"})
		return
	}
	refreshToken, refreshJTI, err := issueJWT(req.Email, req.Role, tokenTypeRefresh, sessionID, cfg.refreshTTL)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to issue refresh token"})
		return
	}

	session := refreshSession{
		Email:     req.Email,
		Role:      req.Role,
		JTI:       refreshJTI,
		ExpiresAt: time.Now().Add(cfg.refreshTTL),
	}
	if err := persistRefreshSession(sessionID, session, cfg.refreshTTL); err != nil {
		Logger.ErrorContext(ctx, "failed to persist refresh session", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to persist refresh session"})
		return
	}

	setAuthCookies(w, accessToken, refreshToken)
	writeAuthSuccess(w, &authClaims{Role: req.Role, RegisteredClaims: jwt.RegisteredClaims{Subject: req.Email}})
}

func AuthVerify(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	claims, ok := claimsFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "Authentication required"})
		return
	}
	writeAuthSuccess(w, claims)
}

func AuthMe(w http.ResponseWriter, r *http.Request) {
	AuthVerify(w, r)
}

func AuthRefresh(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	refreshCookie, err := r.Cookie(refreshTokenCookieName)
	if err != nil || strings.TrimSpace(refreshCookie.Value) == "" {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "Refresh token missing"})
		return
	}

	claims, err := parseJWTToken(refreshCookie.Value, tokenTypeRefresh)
	if err != nil {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid refresh token"})
		return
	}
	if claims.SessionID == "" || claims.ID == "" {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid refresh token claims"})
		return
	}

	session, found, err := readRefreshSession(claims.SessionID)
	if err != nil {
		Logger.ErrorContext(ctx, "failed to read refresh session", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to validate refresh session"})
		return
	}
	if !found {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "Refresh session not found"})
		return
	}

	if !strings.EqualFold(session.Email, claims.Subject) || session.Role != claims.Role || subtle.ConstantTimeCompare([]byte(session.JTI), []byte(claims.ID)) != 1 {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "Refresh session mismatch"})
		return
	}

	cfg := getAuthConfig()
	accessToken, _, err := issueJWT(claims.Subject, claims.Role, tokenTypeAccess, claims.SessionID, cfg.accessTTL)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to issue access token"})
		return
	}
	setAccessCookie(w, accessToken, cfg)
	writeAuthSuccess(w, &authClaims{Role: claims.Role, RegisteredClaims: jwt.RegisteredClaims{Subject: claims.Subject}})
}

func AuthLogout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	refreshCookie, err := r.Cookie(refreshTokenCookieName)
	if err == nil && strings.TrimSpace(refreshCookie.Value) != "" {
		if claims, parseErr := parseJWTToken(refreshCookie.Value, tokenTypeRefresh); parseErr == nil && claims.SessionID != "" {
			if delErr := deleteRefreshSession(claims.SessionID); delErr != nil {
				Logger.ErrorContext(ctx, "failed to delete refresh session", slog.Any("error", delErr))
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": "Failed to revoke session"})
				return
			}
		}
	}

	clearAuthCookies(w)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{"message": "logged out", "ok": true})
}
