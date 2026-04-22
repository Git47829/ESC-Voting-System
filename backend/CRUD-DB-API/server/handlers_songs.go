package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
)

func (h *Handlers) ServeGetCountries(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	countries, err := h.songs.GetAllCountries(ctx)
	if err != nil {
		Logger.ErrorContext(ctx, "failed to query countries", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to retrieve countries"})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message": "Success",
		"payload": countries,
	})
}

func (h *Handlers) ServeGetCountryByName(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	idStr := r.PathValue("NAME")
	if len(idStr) != 2 {
		Logger.ErrorContext(ctx, "Country code must be two characters", slog.String("CountryID", idStr))
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{"error": "Country must be two Characters in length"})
		return
	}

	countries, err := h.songs.GetCountriesByID(ctx, idStr)
	if err != nil {
		Logger.ErrorContext(ctx, "Error querying database", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{
			"error":       "Could not retrieve Country with ID",
			"requestedID": idStr,
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message": "Success",
		"payload": countries,
	})
}

func (h *Handlers) ServeGetSongs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	songs, err := h.songs.GetAllSongs(ctx)
	if err != nil {
		Logger.ErrorContext(ctx, "Error querying database", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{"error": "Error querying Database"})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message": "Success",
		"payload": songs,
	})
}

func (h *Handlers) ServeGetSongByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	idStr := r.PathValue("ID")
	ID, err := strconv.Atoi(idStr)
	if err != nil {
		Logger.ErrorContext(ctx, "Invalid ID Value", slog.Any("error", err), slog.String("ID", idStr))
		return
	}

	songs, err := h.songs.GetSongsByID(ctx, ID)
	if err != nil {
		Logger.ErrorContext(ctx, "Error querying database", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{"error": "Error querying Database"})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message": "Success",
		"payload": songs,
	})
}
