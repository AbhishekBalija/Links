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
	Auth             AuthConfig
	Cookie           CookieConfig
	CORS             CORSConfig
	Mailer           MailerConfig
}

// DatabasePoolConfig controls the database/sql pool used by GORM.
type DatabasePoolConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// AuthConfig holds JWT signing secrets and token lifetimes.
// Per docs/environment.md: JWT_ACCESS_SECRET, JWT_REFRESH_SECRET,
// ACCESS_TOKEN_TTL, REFRESH_TOKEN_TTL.
// Per docs/auth.md § Token Strategy: access 10-15m, refresh 7-30d.
type AuthConfig struct {
	JWTAccessSecret  string
	JWTRefreshSecret string
	AccessTokenTTL   time.Duration
	RefreshTokenTTL  time.Duration
}

// CookieConfig controls refresh-token cookie attributes.
// Per docs/auth.md § Cookie Strategy and docs/environment.md:
// COOKIE_SECURE, COOKIE_SAME_SITE.
type CookieConfig struct {
	Secure   bool
	SameSite string
}

// CORSConfig controls allowed origins.
// Per docs/environment.md: CORS_ALLOWED_ORIGINS.
type CORSConfig struct {
	AllowedOrigins []string
}

// MailerConfig holds Resend API credentials for transactional emails.
type MailerConfig struct {
	ResendAPIKey string
	FromEmail    string
	FrontendURL  string
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
		Auth: AuthConfig{
			JWTAccessSecret:  os.Getenv("JWT_ACCESS_SECRET"),
			JWTRefreshSecret: os.Getenv("JWT_REFRESH_SECRET"),
			AccessTokenTTL:   durationValue("ACCESS_TOKEN_TTL", 15*time.Minute),
			RefreshTokenTTL:  durationValue("REFRESH_TOKEN_TTL", 7*24*time.Hour),
		},
		Cookie: CookieConfig{
			Secure:   boolValue("COOKIE_SECURE", true),
			SameSite: valueOrDefault("COOKIE_SAME_SITE", "lax"),
		},
		CORS: CORSConfig{
			AllowedOrigins: csvValue("CORS_ALLOWED_ORIGINS"),
		},
		Mailer: MailerConfig{
			ResendAPIKey: os.Getenv("RESEND_API_KEY"),
			FromEmail:    valueOrDefault("FROM_EMAIL", "onboarding@resend.dev"),
			FrontendURL:  frontendURL(),
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

// LoadEnv loads local development environment variables when the file exists.
func LoadEnv() error {
	return loadLocalEnv()
}

// GetEnv returns the value of an environment variable or a fallback default.
func GetEnv(key, fallback string) string {
	return valueOrDefault(key, fallback)
}

// GetPort returns the HTTP port used by the API.
func GetPort() string {
	if port := firstSet("PORT", "APP_PORT"); port != "" {
		return port
	}
	return defaultPort
}

// GetDatabaseDSN returns the PostgreSQL connection string.
func GetDatabaseDSN() string {
	return databaseURL()
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

	// JWT secrets are required in every environment because auth routes are
	// always registered and an empty HMAC key would make tokens forgeable.
	if c.Auth.JWTAccessSecret == "" {
		return fmt.Errorf("JWT_ACCESS_SECRET is required")
	}
	if c.Auth.JWTRefreshSecret == "" {
		return fmt.Errorf("JWT_REFRESH_SECRET is required")
	}
	if c.Auth.AccessTokenTTL <= 0 {
		return fmt.Errorf("ACCESS_TOKEN_TTL must be greater than zero")
	}
	if c.Auth.RefreshTokenTTL <= 0 {
		return fmt.Errorf("REFRESH_TOKEN_TTL must be greater than zero")
	}
	if c.Cookie.SameSite == "none" && !c.Cookie.Secure {
		return fmt.Errorf("COOKIE_SECURE must be true when COOKIE_SAME_SITE is none")
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

func frontendURL() string {
	if v := os.Getenv("FRONTEND_URL"); v != "" {
		return v
	}
	if os.Getenv("VERCEL_ENV") == "production" {
		if v := os.Getenv("VERCEL_PROJECT_PRODUCTION_URL"); v != "" {
			return "https://" + v
		}
	}
	if v := os.Getenv("VERCEL_URL"); v != "" {
		return "https://" + v
	}
	return "http://localhost:5173"
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
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
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

func boolValue(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func csvValue(key string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
