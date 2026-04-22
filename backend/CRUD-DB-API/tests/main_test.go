package server_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"

	"crud-db-api/server"
)

var (
	adminAccessToken string
	juryAccessTokens = map[string]string{}
)

const fixed2FACodeEnv = "AUTH_FIXED_2FA_CODE"

func TestMain(m *testing.M) {
	// Initialize globals that handlers depend on.
	server.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	server.Tracer = otel.Tracer("test")

	adminHash, _ := server.HashPassword("test-admin-pw")
	jury1Hash, _ := server.HashPassword("jury1")
	jury2Hash, _ := server.HashPassword("jury2")
	jury3Hash, _ := server.HashPassword("jury3")

	os.Setenv("adminPassword", adminHash)
	os.Setenv("adminMail", "test-admin@test.com")
	os.Setenv("juryPassword1", jury1Hash)
	os.Setenv("juryPassword2", jury2Hash)
	os.Setenv("juryPassword3", jury3Hash)
	os.Setenv("juryMail1", "jury1@test.com")
	os.Setenv("juryMail2", "jury2@test.com")
	os.Setenv("juryMail3", "jury3@test.com")
	os.Setenv("TESTADMINPW", "test-admin-pw")
	os.Setenv("JWT_SECRET", "test-jwt-secret")
	os.Setenv("AUTH_COOKIE_SECURE", "false")
	os.Setenv(fixed2FACodeEnv, "123456")
	server.InitCookieSecret()

	// Mock EuroMail server so AdminLogin/JuryLogin can send verification emails in tests.
	mockEuroMail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	os.Setenv("EuroMailURL", mockEuroMail.URL)

	var err error
	adminAccessToken, _, err = loginAndExtractCookies("test-admin@test.com", "test-admin-pw", "admin")
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "test setup failed: %v\n", err)
		os.Exit(1)
	}
	if juryAccessTokens["jury1@test.com"], _, err = loginAndExtractCookies("jury1@test.com", "jury1", "jury"); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "test setup failed: %v\n", err)
		os.Exit(1)
	}
	if juryAccessTokens["jury2@test.com"], _, err = loginAndExtractCookies("jury2@test.com", "jury2", "jury"); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "test setup failed: %v\n", err)
		os.Exit(1)
	}
	if juryAccessTokens["jury3@test.com"], _, err = loginAndExtractCookies("jury3@test.com", "jury3", "jury"); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "test setup failed: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()
	mockEuroMail.Close()
	os.Exit(code)
}

func loginAndExtractCookies(email, password, role string) (string, string, error) {
	rr, err := setupLoginSession(email, password, role)
	if err != nil {
		return "", "", err
	}

	accessToken, refreshToken := extractAuthCookies(rr)
	if accessToken == "" || refreshToken == "" {
		return "", "", fmt.Errorf("login completed but auth cookies missing for %s (%s)", email, role)
	}
	return accessToken, refreshToken, nil
}

func setupLoginSession(email, password, role string) (*httptest.ResponseRecorder, error) {
	body, err := json.Marshal(map[string]string{"email": email, "password": password, "role": role})
	if err != nil {
		return nil, err
	}
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	server.AuthLogin(rr, req)
	if rr.Code != http.StatusOK {
		return nil, fmt.Errorf("login failed for %s (%s): status=%d body=%s", email, role, rr.Code, rr.Body.String())
	}

	if accessToken, refreshToken := extractAuthCookies(rr); accessToken != "" && refreshToken != "" {
		return rr, nil
	}

	code := strings.TrimSpace(os.Getenv(fixed2FACodeEnv))
	if code == "" {
		return nil, fmt.Errorf("2FA verification required for %s (%s), but %s is not set", email, role, fixed2FACodeEnv)
	}

	verifyBody, err := json.Marshal(map[string]string{"email": email, "code": code, "role": role})
	if err != nil {
		return nil, err
	}
	verifyReq := httptest.NewRequest(http.MethodPost, "/auth/verify-code", bytes.NewReader(verifyBody))
	verifyRR := httptest.NewRecorder()
	server.AuthVerifyCode(verifyRR, verifyReq)
	if verifyRR.Code != http.StatusOK {
		return nil, fmt.Errorf("2FA verify failed for %s (%s): status=%d body=%s", email, role, verifyRR.Code, verifyRR.Body.String())
	}
	return verifyRR, nil
}

func extractAuthCookies(rr *httptest.ResponseRecorder) (string, string) {
	var accessToken, refreshToken string
	for _, cookie := range rr.Result().Cookies() {
		switch cookie.Name {
		case "esc_access_token":
			accessToken = cookie.Value
		case "esc_refresh_token":
			refreshToken = cookie.Value
		}
	}
	if accessToken == "" || refreshToken == "" {
		return "", ""
	}
	return accessToken, refreshToken
}

func adminAuthRequest(method, path string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	req.AddCookie(&http.Cookie{Name: "esc_access_token", Value: adminAccessToken})
	return req
}

func badAdminAuthRequest(method, path string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	req.Header.Set("X-Email", os.Getenv("adminMail"))
	return req
}

func juryAuthRequest(method, path, token, email string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	if strings.Contains(strings.ToLower(token), "wrong") || strings.Contains(strings.ToLower(token), "invalid") {
		req.AddCookie(&http.Cookie{Name: "esc_access_token", Value: token})
		return req
	}
	if accessToken, ok := juryAccessTokens[email]; ok {
		req.AddCookie(&http.Cookie{Name: "esc_access_token", Value: accessToken})
	}
	return req
}
