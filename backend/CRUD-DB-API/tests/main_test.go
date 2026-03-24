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

	os.Setenv("adminPassword", "test-admin-pw")
	os.Setenv("juryPassword1", "jury1")
	os.Setenv("juryPassword2", "jury2")
	os.Setenv("juryPassword3", "jury3")
	server.InitCookieSecret()

	// globalVoteServer stays nil — NotifyVote() is a safe no-op when nil.

	os.Exit(m.Run())
}
