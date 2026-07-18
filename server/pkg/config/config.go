package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// LoadEnv loads local development environment variables when the file exists.
func LoadEnv() error {
	if err := godotenv.Load(".env.local"); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("load .env.local: %w", err)
	}
	return nil
}

// GetEnv returns the value of an environment variable or a fallback default.
func GetEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// GetPort returns the HTTP port used by the API.
func GetPort() string {
	if port := os.Getenv("PORT"); port != "" {
		return port
	}
	if port := os.Getenv("APP_PORT"); port != "" {
		return port
	}
	return "8080"
}

// GetDatabaseDSN returns the PostgreSQL connection string
func GetDatabaseDSN() string {
	if databaseURL := os.Getenv("DATABASE_URL"); databaseURL != "" {
		return databaseURL
	}

	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
	}

	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}

	user := os.Getenv("DB_USER")
	if user == "" {
		user = "postgres"
	}

	password := os.Getenv("DB_PASSWORD")

	dbname := os.Getenv("DB_NAME")
	if dbname == "" {
		dbname = "linksdb"
	}

	sslmode := os.Getenv("DB_SSLMODE")
	if sslmode == "" {
		sslmode = "disable"
	}

	// Build DSN without password if not provided
	if password != "" {
		return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			host, port, user, password, dbname, sslmode)
	}
	return fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=%s",
		host, port, user, dbname, sslmode)
}
