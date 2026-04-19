package converter

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/mail"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrMissingToken  = errors.New("missing authentication token")
	ErrInvalidToken  = errors.New("invalid authentication token")
	ErrForbiddenRole = errors.New("insufficient role")
)

type authContextKey struct{}

type JWTClaims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

type JWTVerifier struct {
	secret      []byte
	cookieNames []string
}

func NewJWTVerifierFromEnv() (*JWTVerifier, error) {
	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	secretPrefix := "jwt-secret:"
	if secret == "" {
		secret = strings.TrimSpace(os.Getenv("COOKIESIGNINGKEY"))
		secretPrefix = "jwt-cookie-secret:"
	}
	if secret == "" {
		return nil, fmt.Errorf("JWT secret missing: set JWT_SECRET")
	}

	cookieName := strings.TrimSpace(getEnv("AUTH_COOKIE_NAME", "esc_access_token"))
	return &JWTVerifier{
		secret:      deriveSigningKey(secretPrefix, secret),
		cookieNames: uniqueNonEmpty(cookieName, "esc_access_token", "auth_token", "access_token", "token"),
	}, nil
}

func deriveSigningKey(prefix, secret string) []byte {
	sum := sha256.Sum256([]byte(prefix + secret))
	return sum[:]
}

func RequireJWTAuth(verifier *JWTVerifier, allowedRoles ...string) func(http.Handler) http.Handler {
	roleSet := make(map[string]struct{}, len(allowedRoles))
	for _, role := range allowedRoles {
		r := strings.TrimSpace(strings.ToLower(role))
		if r != "" {
			roleSet[r] = struct{}{}
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, err := verifier.ValidateRequest(r, roleSet)
			if err != nil {
				status := http.StatusUnauthorized
				msg := "invalid authentication token"
				switch {
				case errors.Is(err, ErrMissingToken):
					msg = "missing authentication token"
				case errors.Is(err, ErrForbiddenRole):
					status = http.StatusForbidden
					msg = "insufficient role"
				case errors.Is(err, ErrInvalidToken):
					msg = "invalid authentication token"
				default:
					status = http.StatusInternalServerError
					msg = "authentication configuration error"
				}

				Logger.WarnContext(r.Context(), "auth check failed", slog.String("error", err.Error()))
				writeAuthError(w, status, msg)
				return
			}

			ctx := context.WithValue(r.Context(), authContextKey{}, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func AuthClaimsFromContext(ctx context.Context) (JWTClaims, bool) {
	claims, ok := ctx.Value(authContextKey{}).(*JWTClaims)
	if !ok || claims == nil {
		return JWTClaims{}, false
	}
	return *claims, true
}

func (v *JWTVerifier) ValidateRequest(r *http.Request, allowedRoles map[string]struct{}) (*JWTClaims, error) {
	if v == nil || len(v.secret) == 0 {
		return nil, fmt.Errorf("verifier unavailable")
	}

	token, err := v.extractToken(r)
	if err != nil {
		return nil, err
	}
	return v.ValidateToken(token, allowedRoles)
}

func (v *JWTVerifier) extractToken(r *http.Request) (string, error) {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			return "", fmt.Errorf("%w: malformed bearer token", ErrInvalidToken)
		}
		return strings.TrimSpace(parts[1]), nil
	}

	for _, name := range v.cookieNames {
		cookie, err := r.Cookie(name)
		if err == nil && strings.TrimSpace(cookie.Value) != "" {
			return strings.TrimSpace(cookie.Value), nil
		}
	}

	return "", ErrMissingToken
}

func (v *JWTVerifier) ValidateToken(token string, allowedRoles map[string]struct{}) (*JWTClaims, error) {
	parsed, err := jwt.ParseWithClaims(token, &JWTClaims{}, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %s", t.Method.Alg())
		}
		return v.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	claims, ok := parsed.Claims.(*JWTClaims)
	if !ok || !parsed.Valid {
		return nil, ErrInvalidToken
	}
	if claims.Subject == "" {
		return nil, fmt.Errorf("%w: missing sub claim", ErrInvalidToken)
	}
	if _, err := mail.ParseAddress(claims.Subject); err != nil {
		return nil, fmt.Errorf("%w: invalid sub claim", ErrInvalidToken)
	}
	if claims.ExpiresAt == nil {
		return nil, fmt.Errorf("%w: missing exp claim", ErrInvalidToken)
	}
	if claims.IssuedAt == nil {
		return nil, fmt.Errorf("%w: missing iat claim", ErrInvalidToken)
	}

	role := strings.ToLower(strings.TrimSpace(claims.Role))
	if !isSupportedRole(role) {
		return nil, fmt.Errorf("%w: invalid role claim", ErrInvalidToken)
	}
	claims.Role = role

	if len(allowedRoles) > 0 {
		if _, ok := allowedRoles[role]; !ok {
			return nil, ErrForbiddenRole
		}
	}
	return claims, nil
}

func isSupportedRole(role string) bool {
	switch role {
	case "admin", "jury", "user":
		return true
	default:
		return false
	}
}

func writeAuthError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func uniqueNonEmpty(items ...string) []string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		k := strings.TrimSpace(item)
		if k == "" {
			continue
		}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, k)
	}
	return out
}
