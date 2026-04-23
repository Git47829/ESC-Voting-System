package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/sync/errgroup"
)

func (h *Handlers) ServeGetHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Success",
		"status":  "healthy",
	})
}

func (h *Handlers) ServeGetVotes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	votes, err := h.votes.GetAllVotes(ctx)
	if err != nil {
		Logger.ErrorContext(ctx, "failed to query votes", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to retrieve Votes"})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message": "Success",
		"payload": votes,
	})
}

func (h *Handlers) ServeVote(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	const cookieName = "vote_state"

	ownCountry := r.URL.Query().Get("ownCountry")
	phone := r.URL.Query().Get("phoneNum")

	if len(ownCountry) != 2 {
		Logger.ErrorContext(ctx, "Country ID must be two characters", slog.String("CountryID:", ownCountry))
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Country code can only be two Charakters in length",
		})
		return
	}

	phoneCountry, phoneErr := CheckPhoneNum(phone)
	if phoneErr != nil {
		Logger.WarnContext(ctx, "phone validation failed", slog.String("phone", phone), slog.Any("error", phoneErr))
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{"error": phoneErr.Error()})
		return
	}
	if phoneCountry == "" {
		Logger.WarnContext(ctx, "invalid phone number", slog.String("phone", phone))
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid phone number"})
		return
	}

	phoneHash := HashPhoneNumber(phone)

	rawID := r.URL.Query().Get("songID")
	songID, err := strconv.Atoi(rawID)
	if err != nil {
		Logger.ErrorContext(ctx, "songID must be integer", slog.Any("error", err), slog.String("songID", rawID))
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid songID value - must be an integer"})
		return
	}

	rawPoints := r.URL.Query().Get("points")
	points, err := strconv.Atoi(rawPoints)
	if err != nil || points < 1 || points > totalVotePoints {
		Logger.WarnContext(ctx, "invalid points value", slog.String("points", rawPoints))
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("points must be an integer between 1 and %d", totalVotePoints),
		})
		return
	}

	var (
		votingIsOpen bool
		songLandID   string
		songFound    bool
	)

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		votingIsOpen, err = h.votes.GetVotingStatus(gctx)
		if err != nil {
			Logger.ErrorContext(gctx, "failed to query voting status", slog.Any("error", err))
		}
		return err
	})

	g.Go(func() error {
		var err error
		songLandID, err = h.votes.GetSongCountry(gctx, songID)
		if err != nil {
			// sql.ErrNoRows means song not found — not a hard error
			songFound = false
			return nil
		}
		songFound = true
		return nil
	})

	if err := g.Wait(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "error querying DB"})
		return
	}

	if !votingIsOpen {
		w.WriteHeader(http.StatusTooEarly)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "voting has not started yet, please try again in a few minutes",
		})
		return
	}

	if !songFound {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Song not found"})
		return
	}

	if songLandID == phoneCountry {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "cannot vote for your own country"})
		return
	}

	tx, txErr := h.votes.BeginTx(ctx)
	if txErr != nil {
		Logger.ErrorContext(ctx, "failed to begin transaction", slog.Any("error", txErr))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to record vote"})
		return
	}
	defer tx.Rollback()

	if err := h.votes.UpsertPhoneVoter(ctx, tx, phoneHash, totalVotePoints); err != nil {
		Logger.ErrorContext(ctx, "failed to upsert phone", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to record vote"})
		return
	}

	rowsAffected, deductErr := h.votes.DeductVotePoints(ctx, tx, phoneHash, rawID, points)
	if deductErr != nil {
		Logger.ErrorContext(ctx, "failed to deduct vote points", slog.Any("error", deductErr))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to record vote"})
		return
	}

	if rowsAffected == 0 {
		remaining, _ := h.votes.GetPhoneVotesRemainingTx(ctx, tx, phoneHash)
		Logger.WarnContext(ctx, "insufficient vote points",
			slog.String("phone_hash", phoneHash),
			slog.Int("requested", points),
			slog.Int("remaining", remaining),
		)
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]any{
			"error":           fmt.Sprintf("not enough vote points remaining (have %d, requested %d)", remaining, points),
			"votes_remaining": remaining,
		})
		return
	}

	songRowsAffected, songErr := h.votes.UpdateSongPublicVotes(ctx, tx, songID, points)
	if songErr != nil {
		Logger.ErrorContext(ctx, "failed to update song votes", slog.Any("error", songErr))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to record vote"})
		return
	}
	if songRowsAffected == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Song not found"})
		return
	}

	if err := tx.Commit(); err != nil {
		Logger.ErrorContext(ctx, "failed to commit vote transaction", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to record vote"})
		return
	}

	var (
		remaining int
		votesCast = make(map[string]int)
		cookieVal string
	)

	remaining, castJSON, readErr := h.votes.GetPhoneVoteState(ctx, phoneHash)
	if readErr != nil {
		Logger.ErrorContext(ctx, "failed to read back vote state", slog.Any("error", readErr))
	} else {
		if castJSON != "" {
			if err := json.Unmarshal([]byte(castJSON), &votesCast); err != nil {
				Logger.ErrorContext(ctx, "failed to unmarshal votes_cast", slog.Any("error", err))
			}
		}
		state := CookieVoteState{
			VotesRemaining: remaining,
			VotesCast:      votesCast,
		}
		cookieVal, _ = EncodeCookieValue(state)
	}

	if cookieVal != "" {
		http.SetCookie(w, &http.Cookie{
			Name:     cookieName,
			Value:    cookieVal,
			Expires:  time.Now().Add(365 * 24 * time.Hour),
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
			Path:     "/",
		})
	}

	h.notifier.NotifyVote(ctx, songID, ownCountry) //nolint:errcheck

	Logger.InfoContext(ctx, "vote cast",
		slog.String("phone_hash", phoneHash),
		slog.Int("song_id", songID),
		slog.Int("points_spent", points),
		slog.Int("votes_remaining", remaining),
	)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"message":         "Vote cast successfully",
		"voted_for":       songID,
		"points_spent":    points,
		"votes_remaining": remaining,
		"votes_cast":      votesCast,
	})
}
