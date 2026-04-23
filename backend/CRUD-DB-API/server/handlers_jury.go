package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
)

func (h *Handlers) ServeJuryVote(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()

	songIDStr := r.URL.Query().Get("songID")
	songID, err := strconv.Atoi(songIDStr)
	if err != nil {
		Logger.ErrorContext(ctx, "invalid songID value", slog.Any("error", err), slog.String("songID", songIDStr))
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid songID value - must be an Integer",
		})
		return
	}

	pointsStr := r.URL.Query().Get("points")
	parsedPoints, err := strconv.Atoi(pointsStr)
	if err != nil {
		Logger.ErrorContext(ctx, "invalid points value", slog.Any("error", err), slog.String("points", pointsStr))
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid points value - must be an Integer",
		})
		return
	}

	validJuryPoints := map[int]bool{1: true, 2: true, 3: true, 4: true, 5: true, 6: true, 7: true, 8: true, 10: true, 12: true}
	if !validJuryPoints[parsedPoints] {
		Logger.WarnContext(ctx, "invalid jury points value", slog.Int("points", parsedPoints))
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid points value - must be one of: 1, 2, 3, 4, 5, 6, 7, 8, 10, 12",
		})
		return
	}

	isOpen, err := h.votes.GetVotingStatus(ctx)
	if err != nil {
		Logger.ErrorContext(ctx, "failed to query voting status", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Error querying DB"})
		return
	}

	if !isOpen {
		w.WriteHeader(http.StatusTooEarly)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "voting has not started yet, please try again in a few minutes",
		})
		return
	}

	const juryWeight = 1
	rowsAffected, err := h.votes.UpdateSongJuryVotes(ctx, songID, juryWeight*parsedPoints)
	if err != nil {
		Logger.ErrorContext(ctx, "failed to update vote", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to record vote"})
		return
	}
	if rowsAffected == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Song not found"})
		return
	}

	h.notifier.NotifyVote(ctx, songID, "JURY") //nolint:errcheck

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]any{"payload": "Vote Successfully Cast"})
}
