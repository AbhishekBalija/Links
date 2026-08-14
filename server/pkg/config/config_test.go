package config

import (
	"testing"
	"time"
)

func TestConfigValidate(t *testing.T) {
	t.Parallel()

	valid := Config{
		AppEnv:           "local",
		DatabaseURL:      "postgres://example",
		GINMode:          "debug",
		Auth:             AuthConfig{JWTAccessSecret: "local-access-secret", JWTRefreshSecret: "local-refresh-secret"},
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
}
