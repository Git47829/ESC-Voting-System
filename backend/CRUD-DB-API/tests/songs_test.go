package server_test

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

// songCols matches the SELECT in ServeGetSongs / ServeGetSongByID (18 columns).
var songCols = []string{
	"Song_ID", "Song_Name",
	"PublikumsPunkte", "JuryPunkte", "GesamtPunkte",
	"Land_ID", "Land_Name", "Land_POT",
	"Kuenstler_ID", "Kuenstler_Vorname", "Kuenstler_Name", "Kuenstler_Typ",
	"Komponist_ID", "Komponist_Vorname", "Komponist_Name",
	"VotingID", "Voting_IsOpen", "Voting_LastChange",
}

func sampleSongRow() []driver.Value {
	return []driver.Value{
		int64(1), "Satellite Reprise",
		int64(100), int64(50), int64(150),
		"DE", "Germany", int64(5),
		int64(1), "Max", "Mueller", "solo",
		nil, nil, nil,
		int64(1), true, "2024-01-01T00:00:00Z",
	}
}

func TestGetSongs_Returns200WithPayload(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	h := newTestHandlers(t, mockDB)

	rows := sqlmock.NewRows(songCols).AddRow(sampleSongRow()...)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/songs/", nil)
	rr := httptest.NewRecorder()
	h.ServeGetSongs(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var body map[string]any
	json.NewDecoder(rr.Body).Decode(&body)
	if body["message"] != "Success" {
		t.Errorf("expected message=Success, got %v", body["message"])
	}
	payload, ok := body["payload"].([]any)
	if !ok || len(payload) == 0 {
		t.Error("expected non-empty payload")
	}
}

func TestGetSongs_ResponseShape(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	h := newTestHandlers(t, mockDB)

	rows := sqlmock.NewRows(songCols).AddRow(sampleSongRow()...)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/songs/", nil)
	rr := httptest.NewRecorder()
	h.ServeGetSongs(rr, req)

	var body map[string]any
	json.NewDecoder(rr.Body).Decode(&body)
	payload := body["payload"].([]any)
	song := payload[0].(map[string]any)

	for _, key := range []string{
		"songId", "songName", "publicVotes", "juryVotes", "totalVotes",
		"countryId", "countryName",
		"artistId", "artistFirstName", "artistLastName", "artistType",
		"composers", "votingId", "votingIsOpen", "votingLastChange",
	} {
		if _, ok := song[key]; !ok {
			t.Errorf("expected key %q in song object", key)
		}
	}
}

func TestGetSongs_DBError_Returns502(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	h := newTestHandlers(t, mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("db error"))

	req := httptest.NewRequest(http.MethodGet, "/songs/", nil)
	rr := httptest.NewRecorder()
	h.ServeGetSongs(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

func TestGetSongByID_ValidID_Returns200(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	h := newTestHandlers(t, mockDB)

	rows := sqlmock.NewRows(songCols).AddRow(sampleSongRow()...)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /songByID/{ID}", h.ServeGetSongByID)
	req := httptest.NewRequest(http.MethodGet, "/songByID/1", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var body map[string]any
	json.NewDecoder(rr.Body).Decode(&body)
	payload, ok := body["payload"].([]any)
	if !ok || len(payload) != 1 {
		t.Errorf("expected 1 song, got %v", body["payload"])
	}
}

func TestGetSongByID_NotFound_Returns200EmptyPayload(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	h := newTestHandlers(t, mockDB)

	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows(songCols))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /songByID/{ID}", h.ServeGetSongByID)
	req := httptest.NewRequest(http.MethodGet, "/songByID/999", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 with empty payload, got %d", rr.Code)
	}
}

// TestGetSongByID_InvalidIDString_FallsThroughToQuery documents the known bug
// where strconv.Atoi failure does NOT return early — the handler falls through
// and queries with ID=0.
func TestGetSongByID_InvalidIDString_FallsThroughToQuery(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	h := newTestHandlers(t, mockDB)

	// Handler falls through with ID=0 and runs the query.
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows(songCols))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /songByID/{ID}", h.ServeGetSongByID)
	req := httptest.NewRequest(http.MethodGet, "/songByID/notanumber", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	// Bug: no early return on invalid ID — returns 200 with empty payload.
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 (bug: no early return on invalid ID), got %d", rr.Code)
	}
}

func TestGetSongByID_DBError_Returns500(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	h := newTestHandlers(t, mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("db error"))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /songByID/{ID}", h.ServeGetSongByID)
	req := httptest.NewRequest(http.MethodGet, "/songByID/1", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}
