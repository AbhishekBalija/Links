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

	// SameSite=None would send the refresh cookie on cross-site requests,
	// which the CSRF protection doesn't rely on but shouldn't have to face.
	secureNone := valid
	secureNone.Cookie = CookieConfig{SameSite: "none", Secure: true}
	if err := secureNone.Validate(); err == nil {
		t.Fatal("expected SameSite=None to fail validation")
	}

	strict := valid
	strict.Cookie = CookieConfig{SameSite: "strict"}
	if err := strict.Validate(); err != nil {
		t.Fatalf("expected SameSite=Strict to be valid, got %v", err)
	}

	insecureProduction := valid
	insecureProduction.AppEnv = "production"
	insecureProduction.GINMode = "release"
	insecureProduction.Cookie = CookieConfig{SameSite: "lax", Secure: false}
	if err := insecureProduction.Validate(); err == nil {
		t.Fatal("expected a cookie without Secure outside local to fail validation")
	}
	secureProduction := insecureProduction
	secureProduction.Cookie.Secure = true
	if err := secureProduction.Validate(); err != nil {
		t.Fatalf("expected a Secure Lax cookie in production to be valid, got %v", err)
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
