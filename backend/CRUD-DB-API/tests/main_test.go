package server_test

import (
	"io"
	"log/slog"
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
	os.Setenv("juryPassword1", jury1Hash)
	os.Setenv("juryPassword2", jury2Hash)
	os.Setenv("juryPassword3", jury3Hash)
	os.Setenv("TESTADMINPW", "test-admin-pw")
	server.InitCookieSecret()

	// globalVoteServer stays nil — NotifyVote() is a safe no-op when nil.

	os.Exit(m.Run())
}

// adminURL appends the test admin token as a query parameter.
func adminURL(path string) string {
	return path + "?Token=" + os.Getenv("TESTADMINPW")
}

// badTokenURL appends an invalid token as a query parameter.
func badTokenURL(path string) string {
	return path + "?Token=invalid-token"
}
