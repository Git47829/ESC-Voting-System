package server_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func TestGetVotes_EmptyDB_Returns200EmptyPayload(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	h := newTestHandlers(t, mockDB)

	cols := []string{"ID", "Name", "Country", "PublikumsPunkte", "JuryPunkte", "GesamtPunkte"}
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows(cols))

	req := httptest.NewRequest(http.MethodGet, "/votes/", nil)
	rr := httptest.NewRecorder()
	h.ServeGetVotes(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var body map[string]any
	json.NewDecoder(rr.Body).Decode(&body)
	payload, ok := body["payload"].([]any)
	if !ok || payload == nil {
		// nil JSON array decodes as null; empty slice is also acceptable
		if body["payload"] != nil {
			t.Errorf("expected empty payload, got %v", body["payload"])
		}
	}
}

func TestGetVotes_WithSongs_Returns200(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	h := newTestHandlers(t, mockDB)

	cols := []string{"ID", "Name", "Country", "PublikumsPunkte", "JuryPunkte", "GesamtPunkte"}
	rows := sqlmock.NewRows(cols).
		AddRow(1, "Satellite Reprise", "Germany", 100, 50, 150).
		AddRow(2, "Northern Lights", "Sweden", 80, 60, 140)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/votes/", nil)
	rr := httptest.NewRecorder()
	h.ServeGetVotes(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var body map[string]any
	json.NewDecoder(rr.Body).Decode(&body)

	if body["message"] != "Success" {
		t.Errorf("expected message=Success, got %v", body["message"])
	}

	payload, ok := body["payload"].([]any)
	if !ok || len(payload) != 2 {
		t.Errorf("expected 2 songs in payload, got %v", body["payload"])
	}
}

func TestGetVotes_ResponseShape(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	h := newTestHandlers(t, mockDB)

	cols := []string{"ID", "Name", "Country", "PublikumsPunkte", "JuryPunkte", "GesamtPunkte"}
	rows := sqlmock.NewRows(cols).
		AddRow(1, "Test Song", "Germany", 10, 5, 15)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/votes/", nil)
	rr := httptest.NewRecorder()
	h.ServeGetVotes(rr, req)

	var body map[string]any
	json.NewDecoder(rr.Body).Decode(&body)

	payload := body["payload"].([]any)
	song := payload[0].(map[string]any)

	for _, key := range []string{"id", "name", "country", "publicVotes", "juryVotes", "totalVotes"} {
		if _, ok := song[key]; !ok {
			t.Errorf("expected key %q in song object", key)
		}
	}
}

func TestGetVotes_DBError_Returns500(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	h := newTestHandlers(t, mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("db error"))

	req := httptest.NewRequest(http.MethodGet, "/votes/", nil)
	rr := httptest.NewRecorder()
	h.ServeGetVotes(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}

	var body map[string]string
	json.NewDecoder(rr.Body).Decode(&body)
	if body["error"] == "" {
		t.Error("expected error key in response body")
	}
}
