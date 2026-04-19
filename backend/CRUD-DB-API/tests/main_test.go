package server_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"go.opentelemetry.io/otel"

	"crud-db-api/server"
)

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
	server.InitCookieSecret()

	// Mock EuroMail server so AdminLogin/JuryLogin can send verification emails in tests.
	mockEuroMail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	os.Setenv("EuroMailURL", mockEuroMail.URL)

	// globalVoteServer stays nil — NotifyVote() is a safe no-op when nil.

	code := m.Run()
	mockEuroMail.Close()
	os.Exit(code)
}

func adminAuthRequest(method, path string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Authorization", "Bearer "+os.Getenv("TESTADMINPW"))
	req.Header.Set("X-Email", os.Getenv("adminMail"))
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
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Email", email)
	return req
}
