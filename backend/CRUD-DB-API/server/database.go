package server

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

// DB is the package-level database connection.
// Kept as a package-level var for test compatibility (tests assign mockDB here).
var DB *sql.DB

var (
	maxDBAttempts  = 40
	dbAttemptDelay = time.Second * 10
	dbReady        = make(chan struct{})
)

// connectToDatabase establishes a MySQL connection with retry logic.
// Accepts AppConfig so all configuration comes from a single source.
func connectToDatabase(cfg AppConfig) (*sql.DB, error) {
	escapedPass := url.QueryEscape(cfg.DbPass)
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&timeout=10s&readTimeout=10s&writeTimeout=10s",
		cfg.DbUser,
		escapedPass,
		cfg.DbHost,
		cfg.DbPort,
		cfg.DbName,
	)

	log.Printf("Attempting to connect to database at %s:%d/%s", cfg.DbHost, cfg.DbPort, cfg.DbName)

	var (
		conn *sql.DB
		err  error
	)
	for attempt := 1; attempt <= maxDBAttempts; attempt++ {
		conn, err = sql.Open("mysql", dsn)
		if err == nil {
			conn.SetMaxOpenConns(50)
			conn.SetMaxIdleConns(25)
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
		errMsg := "unknown error"
		if err != nil {
			errMsg = err.Error()
		}
		log.Printf("Database connection attempt %d/%d failed: %s", attempt, maxDBAttempts, errMsg)
		if attempt < maxDBAttempts {
			time.Sleep(dbAttemptDelay)
		}
	}
	return nil, fmt.Errorf("could not connect after %d attempts: %w", maxDBAttempts, err)
}
