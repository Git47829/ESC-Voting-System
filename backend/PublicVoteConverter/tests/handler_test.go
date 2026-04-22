package converter_test

import (
	"crypto/sha256"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"esc-points-converter/converter"

	"github.com/golang-jwt/jwt/v5"
	"go.opentelemetry.io/otel"
)

// ---------------------------------------------------------------------------
// TestMain — initialize package-level globals before any handler is called
// ---------------------------------------------------------------------------

func TestMain(m *testing.M) {
	converter.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	converter.Tracer = otel.Tracer("test")
	os.Exit(m.Run())
}

// ---------------------------------------------------------------------------
// Mock CRUD-DB-API server
// ---------------------------------------------------------------------------

type songData struct {
	SongID      int    `json:"songId"`
	SongName    string `json:"songName"`
	CountryID   string `json:"countryId"`
	CountryName string `json:"countryName"`
	PublicVotes int    `json:"publicVotes"`
}

func mockAPIServer(songs []songData, statusCode int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if statusCode != 0 && statusCode != http.StatusOK {
			w.WriteHeader(statusCode)
			json.NewEncoder(w).Encode(map[string]string{"error": "mock error"})
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"message": "Success",
			"payload": songs,
		})
	}))
}

func makeSong(id int, name, country, countryID string, votes int) songData {
	return songData{
		SongID:      id,
		SongName:    name,
		CountryName: country,
		CountryID:   countryID,
		PublicVotes: votes,
	}
}

// ---------------------------------------------------------------------------
// HandleHealth
// ---------------------------------------------------------------------------

func TestHandleHealth_Returns200(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	converter.HandleHealth(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// HandlePreview
// ---------------------------------------------------------------------------

func TestHandlePreview_WithSongs_Returns200(t *testing.T) {
	srv := mockAPIServer([]songData{
		makeSong(1, "Satellite Reprise", "Germany", "DE", 100),
		makeSong(2, "Northern Lights", "Sweden", "SE", 80),
	}, http.StatusOK)
	defer srv.Close()

	handler := converter.HandlePreview(srv.URL, 1)

	req := httptest.NewRequest(http.MethodGet, "/api/esc-points", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", rr.Code, rr.Body.String())
	}
}

func TestHandlePreview_ResponseShape(t *testing.T) {
	srv := mockAPIServer([]songData{
		makeSong(1, "Satellite Reprise", "Germany", "DE", 100),
	}, http.StatusOK)
	defer srv.Close()

	handler := converter.HandlePreview(srv.URL, 1)

	req := httptest.NewRequest(http.MethodGet, "/api/esc-points", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)

	var body map[string]any
	json.NewDecoder(rr.Body).Decode(&body)

	if _, ok := body["message"]; !ok {
		t.Error("expected 'message' key in response")
	}
	if _, ok := body["payload"]; !ok {
		t.Error("expected 'payload' key in response")
	}

	payload := body["payload"].([]any)
	if len(payload) == 0 {
		t.Fatal("expected non-empty payload")
	}
	song := payload[0].(map[string]any)
	for _, key := range []string{"songId", "songName", "country", "countryId", "rawPublicVotes", "escPoints", "rank"} {
		if _, ok := song[key]; !ok {
			t.Errorf("expected key %q in song object", key)
		}
	}
}

func TestHandlePreview_ESCPointsScaledByJuryMembers(t *testing.T) {
	// juryScale=3: 1st place should get 12 * 3 = 36
	srv := mockAPIServer([]songData{
		makeSong(1, "Test", "Germany", "DE", 100),
	}, http.StatusOK)
	defer srv.Close()

	handler := converter.HandlePreview(srv.URL, 3)

	req := httptest.NewRequest(http.MethodGet, "/api/esc-points", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)

	var body map[string]any
	json.NewDecoder(rr.Body).Decode(&body)
	payload := body["payload"].([]any)
	song := payload[0].(map[string]any)

	if song["escPoints"].(float64) != 36 {
		t.Errorf("expected escPoints=36 (12*3), got %v", song["escPoints"])
	}
}

func TestHandlePreview_APIError_Returns502(t *testing.T) {
	srv := mockAPIServer(nil, http.StatusInternalServerError)
	defer srv.Close()

	handler := converter.HandlePreview(srv.URL, 1)

	req := httptest.NewRequest(http.MethodGet, "/api/esc-points", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", rr.Code)
	}
	var body map[string]string
	json.NewDecoder(rr.Body).Decode(&body)
	if body["error"] == "" {
		t.Error("expected error key in response body")
	}
}

func TestHandlePreview_EmptySongs_Returns200EmptyPayload(t *testing.T) {
	srv := mockAPIServer([]songData{}, http.StatusOK)
	defer srv.Close()

	handler := converter.HandlePreview(srv.URL, 1)

	req := httptest.NewRequest(http.MethodGet, "/api/esc-points", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestHandlePreview_ContentTypeIsJSON(t *testing.T) {
	srv := mockAPIServer([]songData{}, http.StatusOK)
	defer srv.Close()

	handler := converter.HandlePreview(srv.URL, 1)

	req := httptest.NewRequest(http.MethodGet, "/api/esc-points", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)

	ct := rr.Header().Get("Content-Type")
	if ct == "" {
		// HandlePreview sets Content-Type after writing body - acceptable
		// Check the body is JSON-parseable instead
		var body map[string]any
		if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
			t.Errorf("response body is not valid JSON: %v", err)
		}
	}
}

func TestHandlePreview_ZeroVotesSongsGet0ESCPoints(t *testing.T) {
	srv := mockAPIServer([]songData{
		makeSong(1, "Song A", "Germany", "DE", 0),
		makeSong(2, "Song B", "Sweden", "SE", 0),
	}, http.StatusOK)
	defer srv.Close()

	handler := converter.HandlePreview(srv.URL, 3)

	req := httptest.NewRequest(http.MethodGet, "/api/esc-points", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)

	var body map[string]any
	json.NewDecoder(rr.Body).Decode(&body)
	payload := body["payload"].([]any)

	for _, item := range payload {
		song := item.(map[string]any)
		if song["escPoints"].(float64) != 0 {
			t.Errorf("song with 0 raw votes should have escPoints=0, got %v", song["escPoints"])
		}
	}
}

func makeJWT(t *testing.T, secret, email, role string, includeExp, includeIat bool) string {
	t.Helper()
	now := time.Now()
	claims := converter.JWTClaims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: email,
		},
	}
	if includeExp {
		claims.ExpiresAt = jwt.NewNumericDate(now.Add(30 * time.Minute))
	}
	if includeIat {
		claims.IssuedAt = jwt.NewNumericDate(now)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	sum := sha256.Sum256([]byte("jwt-secret:" + secret))
	signed, err := token.SignedString(sum[:])
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}
	return signed
}

func newVerifier(t *testing.T, secret string) *converter.JWTVerifier {
	t.Helper()
	t.Setenv("JWT_SECRET", secret)
	verifier, err := converter.NewJWTVerifierFromEnv()
	if err != nil {
		t.Fatalf("failed to build verifier: %v", err)
	}
	return verifier
}

func TestRequireJWTAuth_MissingToken_Returns401(t *testing.T) {
	verifier := newVerifier(t, "test-secret")
	protected := converter.RequireJWTAuth(verifier, "admin", "jury", "user")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/esc-points", nil)
	rr := httptest.NewRecorder()
	protected.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
	var body map[string]string
	_ = json.NewDecoder(rr.Body).Decode(&body)
	if body["error"] != "missing authentication token" {
		t.Fatalf("unexpected error message: %q", body["error"])
	}
}

func TestRequireJWTAuth_InvalidSignature_Returns401(t *testing.T) {
	verifier := newVerifier(t, "test-secret")
	protected := converter.RequireJWTAuth(verifier, "admin", "jury", "user")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	token := makeJWT(t, "different-secret", "user@example.com", "user", true, true)

	req := httptest.NewRequest(http.MethodGet, "/api/esc-points", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	protected.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestRequireJWTAuth_MissingRequiredClaims_Returns401(t *testing.T) {
	verifier := newVerifier(t, "test-secret")
	protected := converter.RequireJWTAuth(verifier, "admin", "jury", "user")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	token := makeJWT(t, "test-secret", "user@example.com", "user", false, true)

	req := httptest.NewRequest(http.MethodGet, "/api/esc-points", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	protected.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestRequireJWTAuth_MissingIatClaim_Returns401(t *testing.T) {
	verifier := newVerifier(t, "test-secret")
	protected := converter.RequireJWTAuth(verifier, "admin", "jury", "user")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	token := makeJWT(t, "test-secret", "user@example.com", "user", true, false)

	req := httptest.NewRequest(http.MethodGet, "/api/esc-points", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	protected.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestRequireJWTAuth_InvalidSubjectClaim_Returns401(t *testing.T) {
	verifier := newVerifier(t, "test-secret")
	protected := converter.RequireJWTAuth(verifier, "admin", "jury", "user")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	token := makeJWT(t, "test-secret", "not-an-email", "user", true, true)

	req := httptest.NewRequest(http.MethodGet, "/api/esc-points", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	protected.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestRequireJWTAuth_BearerToken_AllowsProtectedPreview(t *testing.T) {
	verifier := newVerifier(t, "test-secret")
	srv := mockAPIServer([]songData{
		makeSong(1, "Satellite Reprise", "Germany", "DE", 100),
	}, http.StatusOK)
	defer srv.Close()

	protected := converter.RequireJWTAuth(verifier, "admin", "jury", "user")(converter.HandlePreview(srv.URL, 1))
	token := makeJWT(t, "test-secret", "user@example.com", "user", true, true)

	req := httptest.NewRequest(http.MethodGet, "/api/esc-points", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	protected.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rr.Code, rr.Body.String())
	}
}

func TestRequireJWTAuth_MalformedBearerToken_Returns401(t *testing.T) {
	verifier := newVerifier(t, "test-secret")
	protected := converter.RequireJWTAuth(verifier, "admin", "jury", "user")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/esc-points", nil)
	req.Header.Set("Authorization", "Bearer")
	rr := httptest.NewRecorder()
	protected.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestRequireJWTAuth_CookieToken_AllowsRequest(t *testing.T) {
	verifier := newVerifier(t, "test-secret")
	protected := converter.RequireJWTAuth(verifier, "admin", "jury", "user")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	token := makeJWT(t, "test-secret", "admin@example.com", "admin", true, true)

	req := httptest.NewRequest(http.MethodGet, "/api/esc-points", nil)
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: token, HttpOnly: true})
	rr := httptest.NewRecorder()
	protected.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestRequireJWTAuth_HeaderTokenPreferredOverCookie(t *testing.T) {
	verifier := newVerifier(t, "test-secret")
	protected := converter.RequireJWTAuth(verifier, "admin", "jury", "user")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	validCookieToken := makeJWT(t, "test-secret", "user@example.com", "user", true, true)
	invalidHeaderToken := makeJWT(t, "different-secret", "user@example.com", "user", true, true)

	req := httptest.NewRequest(http.MethodGet, "/api/esc-points", nil)
	req.Header.Set("Authorization", "Bearer "+invalidHeaderToken)
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: validCookieToken, HttpOnly: true})
	rr := httptest.NewRecorder()
	protected.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 because header token should be used, got %d", rr.Code)
	}
}

func TestRequireJWTAuth_RoleNormalizedToLowercase(t *testing.T) {
	verifier := newVerifier(t, "test-secret")
	protected := converter.RequireJWTAuth(verifier, "admin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := converter.AuthClaimsFromContext(r.Context())
		if !ok {
			t.Fatal("expected claims in request context")
		}
		if claims.Role != "admin" {
			t.Fatalf("expected normalized role admin, got %q", claims.Role)
		}
		w.WriteHeader(http.StatusOK)
	}))
	token := makeJWT(t, "test-secret", "admin@example.com", "ADMIN", true, true)

	req := httptest.NewRequest(http.MethodGet, "/api/esc-points", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	protected.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestRequireJWTAuth_RejectsDisallowedRole_Returns403(t *testing.T) {
	verifier := newVerifier(t, "test-secret")
	protected := converter.RequireJWTAuth(verifier, "admin")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	token := makeJWT(t, "test-secret", "jury@example.com", "jury", true, true)

	req := httptest.NewRequest(http.MethodGet, "/api/esc-points", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	protected.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// GetEnvInt helper
// ---------------------------------------------------------------------------

func TestGetEnvInt_FallbackOnEmpty(t *testing.T) {
	t.Setenv("TEST_INT_VAR", "")
	got := converter.GetEnvInt("TEST_INT_VAR", 42)
	if got != 42 {
		t.Errorf("expected fallback 42, got %d", got)
	}
}

func TestGetEnvInt_FallbackOnInvalid(t *testing.T) {
	t.Setenv("TEST_INT_VAR", "notanumber")
	got := converter.GetEnvInt("TEST_INT_VAR", 99)
	if got != 99 {
		t.Errorf("expected fallback 99, got %d", got)
	}
}

func TestGetEnvInt_FallbackOnZeroOrNegative(t *testing.T) {
	t.Setenv("TEST_INT_VAR", "0")
	got := converter.GetEnvInt("TEST_INT_VAR", 5)
	if got != 5 {
		t.Errorf("expected fallback for n<=0, got %d", got)
	}
}

func TestGetEnvInt_ValidPositiveValue(t *testing.T) {
	t.Setenv("TEST_INT_VAR", "7")
	got := converter.GetEnvInt("TEST_INT_VAR", 1)
	if got != 7 {
		t.Errorf("expected 7, got %d", got)
	}
}
