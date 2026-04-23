package server_test

import (
	"database/sql"
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

// newTestHandlers creates a Handlers for testing. db may be nil for tests that
// do not hit the database (e.g. pure auth or health tests).
func newTestHandlers(t *testing.T, db *sql.DB) *server.Handlers {
	t.Helper()
	return server.NewHandlers(db, server.NoopVoteNotifier{}, server.AppConfig{
		CookieSecret: server.SignedCookieSecret,
		PhoneSecret:  server.SignedPhoneSecret,
	})
}

// adminURL appends the test admin token as a query parameter.
func adminURL(path string) string {
	return path + "?Token=" + os.Getenv("TESTADMINPW")
}

// badTokenURL appends an invalid token as a query parameter.
func badTokenURL(path string) string {
	return path + "?Token=invalid-token"
}
