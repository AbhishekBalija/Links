package config

import (
	"testing"
	"time"
)

func TestConfigValidate(t *testing.T) {
	t.Parallel()

	valid := Config{
		AppEnv:      "local",
		DatabaseURL: "postgres://example",
		GINMode:     "debug",
		Auth: AuthConfig{
			JWTAccessSecret:  "local-access-secret",
			JWTRefreshSecret: "local-refresh-secret",
			AccessTokenTTL:   15 * time.Minute,
			RefreshTokenTTL:  7 * 24 * time.Hour,
		},
		RequestBodyLimit: 1024,
		DatabasePool: DatabasePoolConfig{
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: time.Minute,
			ConnMaxIdleTime: time.Minute,
		},
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected valid config, got %v", err)
	}

	invalid := valid
	invalid.DatabaseURL = ""
	if err := invalid.Validate(); err == nil {
		t.Fatal("expected missing database URL to fail validation")
	}

	insecureNone := valid
	insecureNone.Cookie = CookieConfig{SameSite: "none", Secure: false}
	if err := insecureNone.Validate(); err == nil {
		t.Fatal("expected SameSite=None without Secure to fail validation")
	}

	invalidAccessTTL := valid
	invalidAccessTTL.Auth.AccessTokenTTL = 0
	if err := invalidAccessTTL.Validate(); err == nil {
		t.Fatal("expected non-positive access token TTL to fail validation")
	}

	invalidRefreshTTL := valid
	invalidRefreshTTL.Auth.RefreshTokenTTL = -time.Minute
	if err := invalidRefreshTTL.Validate(); err == nil {
		t.Fatal("expected non-positive refresh token TTL to fail validation")
	}
}
