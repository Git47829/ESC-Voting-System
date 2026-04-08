package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"strings"
	"github.com/nyaruka/phonenumbers"
	"golang.org/x/crypto/bcrypt"
)

func extractToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return ""

}

func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
		token := extractToken(r)
		if ok, msg := CheckAccessAdmin(token); !ok {
			Logger.Warn("Invalid Login Attempt", slog.String("message", "Invalid Login Attempt"))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{"error": msg})
			return
		}
		next.ServeHTTP(w,r)
	}) 
}

func CheckPhoneNum(num string) (string, error) {
	parsed, err := phonenumbers.Parse(num, "")
	if err != nil {
		return "", nil
	}

	if !phonenumbers.IsValidNumber(parsed) {
		return "", nil
	}

	numRegion := phonenumbers.GetRegionCodeForNumber(parsed)

	return numRegion, nil
}

func HashPassword(password string) (string, error) {
	sum, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	if err != nil {
		return "", err
	}

	return string(sum), nil
}

func CheckPassword(password, storedToken string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(storedToken), []byte(password))
	return err == nil
}

func CheckAccessAdmin(input string) (bool, string) {
	adminPassword := os.Getenv("adminPassword")

	if input == "" {
		return false, "Token has to be provided"
	}
	if CheckPassword(input, adminPassword) {
		return true, "Authorized"
	}

	return false, "Wrong Token provided"
}

func CheckAccessJury(input string) (bool, string) {
	juryPassword1 := os.Getenv("juryPassword1")
	juryPassword2 := os.Getenv("juryPassword2")
	juryPassword3 := os.Getenv("juryPassword3")

	TestToken := []string{juryPassword1, juryPassword2, juryPassword3}

	results := make(chan bool, len(TestToken))

	var wg sync.WaitGroup

	for _, token := range TestToken {
		wg.Add(1)
		t := token
		go func() {
			defer wg.Done()
			results <- CheckPassword(input, t)
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	authorized := false

	for matched := range results {
		if matched {
			authorized = true
		}
	}

	if authorized {
		return true, "Authorized"
	}
	return false, "Wrong Token Provided"
}

func generateToken() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	return hex.EncodeToString(b), err
}

func AdminLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()
	Logger.InfoContext(ctx, "New Admin Login", slog.String("message", "New Admin Login"))
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "authenticated",
	})

}

func JuryLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()

	token := extractToken(r)

	authenticated, message := CheckAccessJury(token)

	if authenticated {
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{
			"message": message,
		})
		return
	}
	if !authenticated {
		Logger.WarnContext(ctx, "Invalid Login Atempt", slog.Any("token:", token))
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{
			"error": message,
		})
		return
	}
}
