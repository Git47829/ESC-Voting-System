package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

type TokenStore struct {
	storedTokens map[string]time.Time
}

func (t *TokenStore) requestVerificationToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	Logger.InfoContext(ctx, "New Verification Token requested")


	w.Header().Set("Content-Type", "application/json")
	token, err := generateToken()
	if err != nil {
		Logger.ErrorContext(ctx, "Invaild Token generation", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Unable to generate Token",
		})
	}

	t.storedTokens[token] = time.Now()

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "authorized",
		"token": token,
	})

}

func (t *TokenStore) verifiyWithToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	providedToken := r.PathValue("token") 

	w.Header().Set("Content-Type", "application/json")

	_, exists := t.storedTokens[providedToken]

	if exists {
		Logger.InfoContext(ctx, "New Token verified, evicting from TokenStore", slog.String("message:", "New Verification via Token"))
		t.evictToken(providedToken)
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "authenticated",
		})	
	return
	}

	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{
		"messsage": "Invalid Token provided",
	})
}

func (t *TokenStore) evictToken(input string) {
	for _, v := range t.storedTokens {
		if time.Since(v) < 5*time.Minute {
			delete(t.storedTokens, input)
		}
	}
}
