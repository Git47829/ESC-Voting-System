package server

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"os"
	"strings"
)

type AppConfig struct {
	DbHost        string
	DbName        string
	DbUser        string
	DbPass        string
	DbPort        int
	CookieSecret  []byte
	PhoneSecret   []byte
	AdminMail     string
	AdminPassword string
	JuryMembers   []JuryMember
	EuroMailURL   string
	GRPCPort      string
}

type JuryMember struct {
	Email    string
	Password string
}

func LoadConfig() AppConfig {
	cfg := AppConfig{
		DbHost:        getEnvOrDefault("DB_HOST", "localhost"),
		DbName:        getEnvOrDefault("DB_NAME", "esc_voting"),
		DbUser:        getEnvOrDefault("DB_USER", "root"),
		DbPass:        getEnvOrDefault("DB_PASS", ""),
		DbPort:        getEnvOrDefaultInt("DB_PORT", 3306),
		AdminMail:     os.Getenv("adminMail"),
		AdminPassword: os.Getenv("adminPassword"),
		EuroMailURL:   os.Getenv("EuroMailURL"),
		GRPCPort:      getEnvOrDefault("GRPC_PORT", "50051"),
	}
	cfg.JuryMembers = loadJuryMembersFromEnv()
	cfg.CookieSecret = deriveCookieSecret()
	cfg.PhoneSecret = derivePhoneSecret()
	return cfg
}

func loadJuryMembersFromEnv() []JuryMember {
	var members []JuryMember
	for i := 1; ; i++ {
		mail := strings.TrimSpace(os.Getenv(fmt.Sprintf("juryMail%d", i)))
		if mail == "" {
			break
		}
		pass := os.Getenv(fmt.Sprintf("juryPassword%d", i))
		members = append(members, JuryMember{Email: mail, Password: pass})
	}
	return members
}

func deriveCookieSecret() []byte {
	if key := os.Getenv("COOKIESIGNINGKEY"); key != "" {
		sum := sha256.Sum256([]byte("cookie-secret:" + key))
		return sum[:]
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		panic("cannot generate cookie secret: " + err.Error())
	}
	return secret
}

func derivePhoneSecret() []byte {
	if key := os.Getenv("PHONESIGNINGSECRET"); key != "" {
		sum := sha256.Sum256([]byte("phone-secret:" + key))
		return sum[:]
	}
	return nil
}
