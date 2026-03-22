package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type LocalConfig struct {
	DbHost string `env:"dbHost"`
	DbName string `env:"dbName"`
	DbUser string `env:"dbUser"`
	DbPass string `env:"dbPass"`
	DbPort int    `env:"dbPort"`
}

func loadLocalConfig() LocalConfig {
	return LocalConfig{
		DbHost: getEnvOrDefault("DB_HOST", "localhost"),
		DbName: getEnvOrDefault("DB_NAME", "esc_voting"),
		DbUser: getEnvOrDefault("DB_USER", "root"),
		DbPass: getEnvOrDefault("DB_PASS", ""),
		DbPort: getEnvOrDefaultInt("DB_PORT", 3306),
	}
}

func getEnvOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvOrDefaultInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return fallback
}

// Singleton Pattern
var db *sql.DB

var (
	maxDBAttempts  = 60
	dbAttemptDelay = time.Second * 3
	dbReady        = make(chan struct{})
)

func connectToDatabase(cfg LocalConfig) (*sql.DB, error) {
	escapedPass := url.QueryEscape(cfg.DbPass)
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&timeout=10s&readTimeout=10s&writeTimeout=10s",
		cfg.DbUser,
		escapedPass,
		cfg.DbHost,
		cfg.DbPort,
		cfg.DbName,
	)

	log.Printf("Attempting to connect to database at %s:%d/%s", cfg.DbHost, cfg.DbPort, cfg.DbName)
	log.Printf("Database config: Host=%s, Port=%d, User=%s, Database=%s", cfg.DbHost, cfg.DbPort, cfg.DbUser, cfg.DbName)

	var (
		conn *sql.DB
		err  error
	)
	for attempt := 1; attempt <= maxDBAttempts; attempt++ {
		conn, err = sql.Open("mysql", dsn)
		if err == nil {
			conn.SetMaxOpenConns(25)
			conn.SetMaxIdleConns(5)
			conn.SetConnMaxLifetime(5 * time.Minute)

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			pingErr := conn.PingContext(ctx)
			cancel()

			if pingErr == nil {
				log.Printf("Successfully connected to database on attempt %d", attempt)
				return conn, nil
			}
			err = pingErr
		}
		var errMsg string
		if err != nil {
			errMsg = err.Error()
		} else {
			errMsg = "unknown error"
		}
		log.Printf("Database connection attempt %d/%d failed: %s", attempt, maxDBAttempts, errMsg)
		if attempt < maxDBAttempts {
			time.Sleep(dbAttemptDelay * time.Duration(attempt))
		}
	}
	return nil, fmt.Errorf("could not connect after %d attempts: %w", maxDBAttempts, err)
}
