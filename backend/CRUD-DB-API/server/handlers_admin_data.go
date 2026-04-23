package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
)

func (h *Handlers) ServeAddCountry(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	query := r.URL.Query()
	ID := query.Get("ID")
	Name := query.Get("Name")
	POTstr := query.Get("Pot")
	POT, err := strconv.Atoi(POTstr)

	if len(ID) != 2 {
		Logger.ErrorContext(ctx, "CountryID must be 2 characters", slog.String("Input", ID))
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{"error": "CountryID must be 2 Characters of length"})
		return
	}

	if err != nil {
		Logger.ErrorContext(ctx, "Invalid POT value", slog.Any("error", err), slog.String("POT", POTstr))
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{"error": "Pot Value must be Integer"})
		return
	}

	rowsAffected, err := h.songs.InsertCountry(ctx, ID, Name, POT)
	if err != nil {
		Logger.ErrorContext(ctx, "Failed to Insert Data", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "error inserting into DB"})
		return
	}
	if rowsAffected == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Error verifying Query"})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"payload": "Country added"})
}

func (h *Handlers) ServeAddSong(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	query := r.URL.Query()
	Name := query.Get("SongName")
	if Name == "" {
		Name = query.Get("Name")
	}
	country := query.Get("CountryID")
	if country == "" {
		country = query.Get("Land")
	}
	artistIDStr := query.Get("KuenstlerID")
	if artistIDStr == "" {
		artistIDStr = query.Get("ID")
	}
	artistID, err := strconv.Atoi(artistIDStr)
	if err != nil {
		Logger.ErrorContext(ctx, "Invalid KuenstlerID", slog.Any("error", err), slog.String("KuenstlerID", artistIDStr))
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{"error": "KuenstlerID must be an Integer"})
		return
	}

	youtubeURL := query.Get("YoutubeURL")

	if len(country) != 2 {
		Logger.ErrorContext(ctx, "CountryID must be 2 characters", slog.String("CountryID: ", country))
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{"error": "CountryID can only be two characters in length"})
		return
	}

	rowsAffected, err := h.songs.InsertSong(ctx, Name, country, artistID, youtubeURL)
	if err != nil {
		Logger.ErrorContext(ctx, "Failed to Insert Data", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "error inserting into DB"})
		return
	}
	if rowsAffected == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Error verifying Query"})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"payload": "Song added"})
}

func (h *Handlers) ServeAddArtist(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	query := r.URL.Query()
	Name := query.Get("Name")
	if Name == "" {
		Name = query.Get("LastName")
	}
	vorName := query.Get("vorName")
	if vorName == "" {
		vorName = query.Get("FirstName")
	}
	typ := query.Get("typ")
	if typ == "" {
		typ = query.Get("Type")
	}
	if typ == "" {
		typ = "solo"
	}
	country := query.Get("CountryID")
	if country == "" {
		country = query.Get("Land")
	}

	if len(country) != 2 {
		Logger.ErrorContext(ctx, "CountryID must be 2 characters", slog.String("CountryID: ", country))
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{"error": "CountryID can only be two characters in length"})
		return
	}

	rowsAffected, err := h.songs.InsertArtist(ctx, vorName, Name, typ, country)
	if err != nil {
		Logger.ErrorContext(ctx, "Failed to Insert Data", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "error inserting into DB"})
		return
	}
	if rowsAffected == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Error verifying Query"})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"payload": "Artist added"})
}

func (h *Handlers) ServeAddInterpret(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	query := r.URL.Query()
	IDstr := query.Get("ID")
	ID, err := strconv.Atoi(IDstr)
	if err != nil {
		Logger.ErrorContext(ctx, "Error Parsing ID", slog.Any("error", err), slog.String("ID", IDstr))
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "ID must be an Integer"})
		return
	}

	Name := query.Get("Name")
	vorName := query.Get("Vorname")

	rowsAffected, err := h.songs.InsertInterpret(ctx, ID, Name, vorName)
	if err != nil {
		Logger.ErrorContext(ctx, "Failed to Insert Data", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "error inserting into DB"})
		return
	}
	if rowsAffected == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Error verifying Query"})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"payload": "Interpret added"})
}
