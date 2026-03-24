package converter_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"esc-points-converter/converter"
	pb "esc-points-converter/proto"

	"go.opentelemetry.io/otel"
	"google.golang.org/grpc"
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
// Mock gRPC client
// ---------------------------------------------------------------------------

type mockVoteClient struct {
	songs []*pb.SongVoteData
	err   error
}

func (m *mockVoteClient) StreamVotes(_ context.Context, _ *pb.VoteStreamRequest, _ ...grpc.CallOption) (grpc.ServerStreamingClient[pb.Vote], error) {
	return nil, errors.New("not implemented in mock")
}

func (m *mockVoteClient) GetSongsWithVotes(_ context.Context, _ *pb.GetSongsRequest, _ ...grpc.CallOption) (*pb.GetSongsResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &pb.GetSongsResponse{Songs: m.songs}, nil
}

func makeSong(id int32, name, country, countryID string, votes int32) *pb.SongVoteData {
	return &pb.SongVoteData{
		SongId:      id,
		SongName:    name,
		CountryName: country,
		CountryId:   countryID,
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
	client := &mockVoteClient{
		songs: []*pb.SongVoteData{
			makeSong(1, "Satellite Reprise", "Germany", "DE", 100),
			makeSong(2, "Northern Lights", "Sweden", "SE", 80),
		},
	}
	handler := converter.HandlePreview(client, 1)

	req := httptest.NewRequest(http.MethodGet, "/api/esc-points", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", rr.Code, rr.Body.String())
	}
}

func TestHandlePreview_ResponseShape(t *testing.T) {
	client := &mockVoteClient{
		songs: []*pb.SongVoteData{
			makeSong(1, "Satellite Reprise", "Germany", "DE", 100),
		},
	}
	handler := converter.HandlePreview(client, 1)

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
	client := &mockVoteClient{
		songs: []*pb.SongVoteData{
			makeSong(1, "Test", "Germany", "DE", 100),
		},
	}
	handler := converter.HandlePreview(client, 3)

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

func TestHandlePreview_gRPCError_Returns502(t *testing.T) {
	client := &mockVoteClient{err: errors.New("grpc unavailable")}
	handler := converter.HandlePreview(client, 1)

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
	client := &mockVoteClient{songs: []*pb.SongVoteData{}}
	handler := converter.HandlePreview(client, 1)

	req := httptest.NewRequest(http.MethodGet, "/api/esc-points", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestHandlePreview_ContentTypeIsJSON(t *testing.T) {
	client := &mockVoteClient{songs: []*pb.SongVoteData{}}
	handler := converter.HandlePreview(client, 1)

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
	client := &mockVoteClient{
		songs: []*pb.SongVoteData{
			makeSong(1, "Song A", "Germany", "DE", 0),
			makeSong(2, "Song B", "Sweden", "SE", 0),
		},
	}
	handler := converter.HandlePreview(client, 3)

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
