package converter_test

import (
	"testing"

	"esc-points-converter/converter"
)

// ---------------------------------------------------------------------------
// EscPointsForRank
// ---------------------------------------------------------------------------

func TestEscPointsForRank_TableDriven(t *testing.T) {
	cases := []struct {
		rank     int
		expected int
	}{
		{0, 12},
		{1, 10},
		{2, 8},
		{3, 7},
		{4, 6},
		{5, 5},
		{6, 4},
		{7, 3},
		{8, 2},
		{9, 1},
		{10, 0},
		{11, 0},
		{100, 0},
	}

	for _, tc := range cases {
		got := converter.EscPointsForRank(tc.rank)
		if got != tc.expected {
			t.Errorf("EscPointsForRank(%d) = %d, want %d", tc.rank, got, tc.expected)
		}
	}
}

// ---------------------------------------------------------------------------
// RankAndConvert
// ---------------------------------------------------------------------------

func TestRankAndConvert_EmptyInput(t *testing.T) {
	result := converter.RankAndConvert([]converter.Song{})
	if len(result) != 0 {
		t.Errorf("expected empty result, got %v", result)
	}
}

func TestRankAndConvert_SortsByRawVotesDesc(t *testing.T) {
	songs := []converter.Song{
		{ID: 1, RawVotes: 5},
		{ID: 2, RawVotes: 10},
		{ID: 3, RawVotes: 3},
	}
	result := converter.RankAndConvert(songs)
	if result[0].ID != 2 {
		t.Errorf("expected song 2 (10 votes) first, got song %d", result[0].ID)
	}
	if result[1].ID != 1 {
		t.Errorf("expected song 1 (5 votes) second, got song %d", result[1].ID)
	}
	if result[2].ID != 3 {
		t.Errorf("expected song 3 (3 votes) third, got song %d", result[2].ID)
	}
}

func TestRankAndConvert_TieBreaksByIDAsc(t *testing.T) {
	songs := []converter.Song{
		{ID: 5, RawVotes: 100},
		{ID: 2, RawVotes: 100},
		{ID: 8, RawVotes: 100},
	}
	result := converter.RankAndConvert(songs)
	if result[0].ID != 2 {
		t.Errorf("expected lowest ID (2) to win tie, got %d", result[0].ID)
	}
	if result[1].ID != 5 {
		t.Errorf("expected second ID (5) in tie, got %d", result[1].ID)
	}
}

func TestRankAndConvert_AssignsCorrectESCPoints(t *testing.T) {
	songs := make([]converter.Song, 12)
	for i := range songs {
		songs[i] = converter.Song{ID: i + 1, RawVotes: 100 - i}
	}
	result := converter.RankAndConvert(songs)

	expected := []int{12, 10, 8, 7, 6, 5, 4, 3, 2, 1, 0, 0}
	for i, s := range result {
		if s.ESCPoints != expected[i] {
			t.Errorf("rank %d: expected ESCPoints=%d, got %d", i+1, expected[i], s.ESCPoints)
		}
	}
}

func TestRankAndConvert_ZeroVotesSongsGet0Points(t *testing.T) {
	songs := []converter.Song{
		{ID: 1, RawVotes: 0},
		{ID: 2, RawVotes: 0},
	}
	result := converter.RankAndConvert(songs)
	for _, s := range result {
		if s.ESCPoints != 0 {
			t.Errorf("song with 0 votes should get 0 ESC points, got %d", s.ESCPoints)
		}
	}
}

func TestRankAndConvert_SingleSong_Gets12Points(t *testing.T) {
	songs := []converter.Song{{ID: 1, RawVotes: 50}}
	result := converter.RankAndConvert(songs)
	if result[0].ESCPoints != 12 {
		t.Errorf("expected 12 pts for single song with votes, got %d", result[0].ESCPoints)
	}
}

func TestRankAndConvert_Rank1Based(t *testing.T) {
	songs := []converter.Song{
		{ID: 1, RawVotes: 10},
		{ID: 2, RawVotes: 5},
	}
	result := converter.RankAndConvert(songs)
	if result[0].Rank != 1 {
		t.Errorf("first song should have Rank=1, got %d", result[0].Rank)
	}
	if result[1].Rank != 2 {
		t.Errorf("second song should have Rank=2, got %d", result[1].Rank)
	}
}

func TestRankAndConvert_PreservesAllFields(t *testing.T) {
	songs := []converter.Song{
		{ID: 42, Name: "Test Song", Country: "Germany", LandID: "DE", RawVotes: 100},
	}
	result := converter.RankAndConvert(songs)
	s := result[0]
	if s.ID != 42 {
		t.Errorf("ID should be preserved: got %d", s.ID)
	}
	if s.Name != "Test Song" {
		t.Errorf("Name should be preserved: got %q", s.Name)
	}
	if s.Country != "Germany" {
		t.Errorf("Country should be preserved: got %q", s.Country)
	}
	if s.LandID != "DE" {
		t.Errorf("LandID should be preserved: got %q", s.LandID)
	}
}

func TestRankAndConvert_EleventhSong_Gets0Points(t *testing.T) {
	songs := make([]converter.Song, 11)
	for i := range songs {
		songs[i] = converter.Song{ID: i + 1, RawVotes: 100 - i}
	}
	result := converter.RankAndConvert(songs)
	if result[10].ESCPoints != 0 {
		t.Errorf("11th song should get 0 ESC points, got %d", result[10].ESCPoints)
	}
}
