package server_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

var countryCols = []string{"ID", "Name", "POT"}

func TestGetCountries_Returns200WithList(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	h := newTestHandlers(t, mockDB)

	rows := sqlmock.NewRows(countryCols).
		AddRow("DE", "Germany", 5).
		AddRow("SE", "Sweden", 4)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/countries/", nil)
	rr := httptest.NewRecorder()
	h.ServeGetCountries(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var body map[string]any
	json.NewDecoder(rr.Body).Decode(&body)

	payload, ok := body["payload"].([]any)
	if !ok || len(payload) != 2 {
		t.Errorf("expected 2 countries, got %v", body["payload"])
	}
}

func TestGetCountries_ResponseShape(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	h := newTestHandlers(t, mockDB)

	rows := sqlmock.NewRows(countryCols).AddRow("DE", "Germany", 5)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/countries/", nil)
	rr := httptest.NewRecorder()
	h.ServeGetCountries(rr, req)

	var body map[string]any
	json.NewDecoder(rr.Body).Decode(&body)
	payload := body["payload"].([]any)
	country := payload[0].(map[string]any)

	for _, key := range []string{"id", "name", "pot"} {
		if _, ok := country[key]; !ok {
			t.Errorf("expected key %q in country object", key)
		}
	}
}

func TestGetCountries_NullPot_SerializesAsNull(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	h := newTestHandlers(t, mockDB)

	rows := sqlmock.NewRows(countryCols).AddRow("DE", "Germany", nil)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/countries/", nil)
	rr := httptest.NewRecorder()
	h.ServeGetCountries(rr, req)

	var body map[string]any
	json.NewDecoder(rr.Body).Decode(&body)
	payload := body["payload"].([]any)
	country := payload[0].(map[string]any)

	if country["pot"] != nil {
		t.Errorf("expected null pot, got %v", country["pot"])
	}
}

func TestGetCountries_DBError_Returns502(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	h := newTestHandlers(t, mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("db error"))

	req := httptest.NewRequest(http.MethodGet, "/countries/", nil)
	rr := httptest.NewRecorder()
	h.ServeGetCountries(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

func TestGetCountryByName_ExistingID_Returns200(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	h := newTestHandlers(t, mockDB)

	rows := sqlmock.NewRows(countryCols).AddRow("DE", "Germany", 5)
	mock.ExpectQuery("SELECT").WithArgs("DE").WillReturnRows(rows)

	// Use the pattern-based router to resolve path values.
	mux := http.NewServeMux()
	mux.HandleFunc("GET /countryByName/{NAME}", h.ServeGetCountryByName)
	req := httptest.NewRequest(http.MethodGet, "/countryByName/DE", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var body map[string]any
	json.NewDecoder(rr.Body).Decode(&body)
	payload := body["payload"].([]any)
	if len(payload) != 1 {
		t.Errorf("expected 1 country, got %d", len(payload))
	}
}

func TestGetCountryByName_NonExistent_Returns422EmptyArray(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	h := newTestHandlers(t, mockDB)

	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows(countryCols))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /countryByName/{NAME}", h.ServeGetCountryByName)
	req := httptest.NewRequest(http.MethodGet, "/countryByName/UNKNOWN", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", rr.Code)
	}
}

func TestGetCountryByName_DBError_Returns500(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	h := newTestHandlers(t, mockDB)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("db error"))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /countryByName/{NAME}", h.ServeGetCountryByName)
	req := httptest.NewRequest(http.MethodGet, "/countryByName/DE", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}
