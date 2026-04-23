package server

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
)

func (h *Handlers) ServeStartContest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	ids, err := h.songs.GetSongIDs(ctx)
	if err != nil {
		Logger.ErrorContext(ctx, "startContest: failed to query songs", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to query songs"})
		return
	}

	if len(ids) == 0 {
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{"error": "No songs in database"})
		return
	}

	// Fisher-Yates shuffle
	//Shuffels the Order of the Songs Randomly
	for i := len(ids) - 1; i > 0; i-- {
		b := make([]byte, 4)
		rand.Read(b)
		j := int(b[0]) % (i + 1)
		ids[i], ids[j] = ids[j], ids[i]
	}

	orderJSON, err := json.Marshal(ids)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to encode song order"})
		return
	}

	if err := h.songs.DeactivateContestRuns(ctx); err != nil {
		Logger.ErrorContext(ctx, "startContest: failed to deactivate old runs", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to deactivate previous contest"})
		return
	}

	if err := h.songs.InsertContestRun(ctx, string(orderJSON)); err != nil {
		Logger.ErrorContext(ctx, "startContest: failed to insert contest run", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to start contest"})
		return
	}

	Logger.InfoContext(ctx, "contest started", slog.Int("songCount", len(ids)))
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"message":   "Contest started",
		"songCount": len(ids),
		"order":     ids,
	})
}

func (h *Handlers) ServeAdvanceContest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	runID, orderJSON, currentIndex, err := h.songs.GetCurrentContestRun(ctx)
	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "No active contest"})
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to query contest"})
		return
	}

	var ids []int
	if err := json.Unmarshal([]byte(orderJSON), &ids); err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to parse song order"})
		return
	}

	nextIndex := currentIndex + 1
	if nextIndex >= len(ids) {
		if err := h.songs.FinishContestRun(ctx, runID); err != nil {
			Logger.ErrorContext(ctx, "failed to finish contest run", slog.Any("error", err))
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"message":  "Contest finished",
			"finished": true,
		})
		return
	}

	if err := h.songs.AdvanceContestRun(ctx, runID, nextIndex); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to advance contest"})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message":      "Advanced to next song",
		"currentIndex": nextIndex,
		"songId":       ids[nextIndex],
		"finished":     false,
	})
}

func (h *Handlers) ServeGetCurrentSong(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	runID, orderJSON, currentIndex, err := h.songs.GetCurrentContestRun(ctx)
	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "No active contest"})
		return
	}
	if err != nil {
		Logger.ErrorContext(ctx, "getCurrentSong: db error", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to query contest"})
		return
	}

	var ids []int
	if err := json.Unmarshal([]byte(orderJSON), &ids); err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to parse song order"})
		return
	}

	if currentIndex >= len(ids) {
		w.WriteHeader(http.StatusGone)
		json.NewEncoder(w).Encode(map[string]string{"error": "Contest has ended"})
		return
	}

	cs, err := h.songs.GetContestSong(ctx, ids[currentIndex])
	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Song not found"})
		return
	}
	if err != nil {
		Logger.ErrorContext(ctx, "getCurrentSong: song query error", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch song"})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message": "Success",
		"payload": map[string]any{
			"runId":         runID,
			"currentIndex":  currentIndex,
			"totalSongs":    len(ids),
			"contestActive": true,
			"currentSong": map[string]any{
				"songId":          cs.SongID,
				"songName":        cs.SongName,
				"youtubeUrl":      cs.YoutubeURL,
				"countryId":       cs.CountryID,
				"countryName":     cs.CountryName,
				"artistId":        cs.ArtistID,
				"artistFirstName": cs.ArtistFirstName,
				"artistLastName":  cs.ArtistLastName,
				"artistType":      cs.ArtistType,
				"publicVotes":     cs.PublicVotes,
				"juryVotes":       cs.JuryVotes,
				"totalVotes":      cs.TotalVotes,
				"votingIsOpen":    cs.VotingIsOpen,
			},
		},
	})
}
