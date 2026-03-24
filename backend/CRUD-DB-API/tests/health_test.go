package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"crud-db-api/server"
)

func TestGetHealth_Returns200(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	server.GetHealth(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestGetHealth_BodyHasStatusKey(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	server.GetHealth(rr, req)

	var body map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if body["status"] != "healthy" {
		t.Errorf("expected status=healthy, got %q", body["status"])
	}
	if body["message"] != "Success" {
		t.Errorf("expected message=Success, got %q", body["message"])
	}
}

func TestGetHealth_ContentTypeJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	server.GetHealth(rr, req)

	ct := rr.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}
}
