package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

const defaultPort = "8080"

// Config contains validated runtime configuration for the API.
type Config struct {
	AppEnv           string
	Port             string
	DatabaseURL      string
	DatabasePool     DatabasePoolConfig
	RequestBodyLimit int64
	GINMode          string
}

// DatabasePoolConfig controls the database/sql pool used by GORM.
type DatabasePoolConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// Load loads local environment variables when present and validates runtime settings.
func Load() (Config, error) {
	if err := loadLocalEnv(); err != nil {
		return Config{}, err
	}

	cfg := Config{
		AppEnv:           valueOrDefault("APP_ENV", "local"),
		Port:             firstSet("PORT", "APP_PORT"),
		DatabaseURL:      databaseURL(),
		GINMode:          valueOrDefault("GIN_MODE", "debug"),
		RequestBodyLimit: int64Value("REQUEST_BODY_LIMIT", 1<<20),
		DatabasePool: DatabasePoolConfig{
			MaxOpenConns:    intValue("DB_MAX_OPEN_CONNS", 15),
			MaxIdleConns:    intValue("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: durationValue("DB_CONN_MAX_LIFETIME", 45*time.Minute),
			ConnMaxIdleTime: durationValue("DB_CONN_MAX_IDLE_TIME", 5*time.Minute),
		},
	}

	if cfg.Port == "" {
		cfg.Port = defaultPort
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// Validate checks values that would otherwise cause unsafe or confusing runtime behavior.
func (c Config) Validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL or database connection settings are required")
	}
	if c.RequestBodyLimit <= 0 {
		return fmt.Errorf("REQUEST_BODY_LIMIT must be greater than zero")
	}
	if c.DatabasePool.MaxOpenConns <= 0 || c.DatabasePool.MaxIdleConns < 0 {
		return fmt.Errorf("database connection limits must be valid")
	}
	if c.DatabasePool.MaxIdleConns > c.DatabasePool.MaxOpenConns {
		return fmt.Errorf("DB_MAX_IDLE_CONNS cannot exceed DB_MAX_OPEN_CONNS")
	}
	if c.DatabasePool.ConnMaxLifetime <= 0 || c.DatabasePool.ConnMaxIdleTime <= 0 {
		return fmt.Errorf("database connection durations must be greater than zero")
	}
	if c.AppEnv == "production" && c.GINMode != "release" {
		return fmt.Errorf("GIN_MODE must be release in production")
	}

	return nil
}

func loadLocalEnv() error {
	if err := godotenv.Load(".env.local"); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("load .env.local: %w", err)
	}
	return nil
}

func databaseURL() string {
	if value := os.Getenv("DATABASE_URL"); value != "" {
		return value
	}
	if !hasAny("DB_HOST", "DB_PORT", "DB_USER", "DB_NAME", "DB_PASSWORD", "DB_SSLMODE") {
		return ""
	}

	host := valueOrDefault("DB_HOST", "localhost")
	port := valueOrDefault("DB_PORT", "5432")
	user := valueOrDefault("DB_USER", "postgres")
	database := valueOrDefault("DB_NAME", "linksdb")
	sslMode := valueOrDefault("DB_SSLMODE", "disable")
	password := os.Getenv("DB_PASSWORD")

	if password == "" {
		return fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=%s", host, port, user, database, sslMode)
	}
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s", host, port, user, password, database, sslMode)
}

func hasAny(keys ...string) bool {
	for _, key := range keys {
		if os.Getenv(key) != "" {
			return true
		}
	}
	return false
}

func firstSet(keys ...string) string {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			return value
		}
	}
	return ""
}

func valueOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func intValue(key string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil || value == 0 {
		return fallback
	}
	return value
}

func int64Value(key string, fallback int64) int64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return -1
	}
	return parsed
}

func durationValue(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return -1
	}
	return parsed
}
