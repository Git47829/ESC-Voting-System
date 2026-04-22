package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

func (h *Handlers) ServeOpenVote(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	rowsAffected, err := h.votes.SetVotingOpen(ctx, true)
	if err != nil {
		Logger.ErrorContext(ctx, "Could not open vote", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Error querying rows"})
		return
	}
	if rowsAffected == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Error opening the Votes"})
		return
	}

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]any{"payload": "The Vote has been opened"})
}

func (h *Handlers) ServeCloseVote(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	rowsAffected, err := h.votes.SetVotingOpen(ctx, false)
	if err != nil {
		Logger.ErrorContext(ctx, "Could not close vote", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Error querying rows"})
		return
	}
	if rowsAffected == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Error Closing Votes"})
		return
	}

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]any{"payload": "The Vote has been closed"})
}

func (h *Handlers) ServeDeleteVotes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	rowsAffected, err := h.votes.ResetAllVotes(ctx)
	if err != nil {
		Logger.ErrorContext(ctx, "Error resetting song votes", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to delete Votes"})
		return
	}
	if rowsAffected == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "No votes found to delete"})
		return
	}

	if err := h.votes.ResetPhoneBudgets(ctx, totalVotePoints); err != nil {
		Logger.ErrorContext(ctx, "failed to reset phone vote budgets", slog.Any("error", err))
	}

	Logger.InfoContext(ctx, "all votes deleted and vote budgets reset")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]any{
		"payload": "All votes have been deleted and vote budgets reset",
	})
}
