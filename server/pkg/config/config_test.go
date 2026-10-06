package config

import (
	"strings"
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

// The test-only sign-in issues a session for any email, so it must never be
// switched on outside a developer's machine or the e2e suite.
func TestTestSignInIsRefusedOutsideLocal(t *testing.T) {
	t.Parallel()
	base := Config{
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
		DatabasePool:     DatabasePoolConfig{MaxOpenConns: 10, MaxIdleConns: 5, ConnMaxLifetime: time.Minute, ConnMaxIdleTime: time.Minute},
		EnableTestSignIn: true,
	}
	if err := base.Validate(); err != nil {
		t.Fatalf("test sign-in on a local server: %v", err)
	}
	for _, env := range []string{"production", "preview", "staging"} {
		other := base
		other.AppEnv = env
		other.GINMode = "release"
		other.Cookie = CookieConfig{Secure: true}
		if err := other.Validate(); err == nil {
			t.Errorf("test sign-in accepted with APP_ENV=%s", env)
		}
	}
}

// Test copies may let every role sign in with an email code, so testers can
// use throwaway inboxes; production keeps the principal and admins on Google.
func TestEmailCodeForEveryRoleIsRefusedInProduction(t *testing.T) {
	t.Parallel()
	base := Config{
		AppEnv:      "preview",
		DatabaseURL: "postgres://example",
		GINMode:     "release",
		Cookie:      CookieConfig{Secure: true},
		Auth: AuthConfig{
			JWTAccessSecret:  "preview-access-secret",
			JWTRefreshSecret: "preview-refresh-secret",
			AccessTokenTTL:   15 * time.Minute,
			RefreshTokenTTL:  7 * 24 * time.Hour,
		},
		RequestBodyLimit:      1024,
		DatabasePool:          DatabasePoolConfig{MaxOpenConns: 10, MaxIdleConns: 5, ConnMaxLifetime: time.Minute, ConnMaxIdleTime: time.Minute},
		EmailCodeForEveryRole: true,
	}
	if err := base.Validate(); err != nil {
		t.Fatalf("email code for every role on a preview: %v", err)
	}
	production := base
	production.AppEnv = "production"
	if err := production.Validate(); err == nil {
		t.Error("email code for every role accepted with APP_ENV=production")
	}
}

// MAIL_PROVIDER=smtp sends through an SMTP server (a testing inbox such as
// Mailtrap, or Gmail for a pilot), which needs its host and port.
func TestSMTPMailNeedsAServer(t *testing.T) {
	t.Parallel()
	base := Config{
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
		DatabasePool:     DatabasePoolConfig{MaxOpenConns: 10, MaxIdleConns: 5, ConnMaxLifetime: time.Minute, ConnMaxIdleTime: time.Minute},
		Mailer:           MailerConfig{Provider: "smtp", SMTPHost: "sandbox.smtp.mailtrap.io", SMTPPort: "2525"},
	}
	if err := base.Validate(); err != nil {
		t.Fatalf("SMTP with a server: %v", err)
	}
	missing := base
	missing.Mailer.SMTPHost = ""
	if err := missing.Validate(); err == nil {
		t.Error("SMTP without a host was accepted")
	}
	unknown := base
	unknown.Mailer.Provider = "carrier-pigeon"
	if err := unknown.Validate(); err == nil {
		t.Error("an unknown MAIL_PROVIDER was accepted")
	}
}

func TestNotOnListCodesPerDayIsASettingOfEachCopy(t *testing.T) {
	t.Setenv("NOT_ON_LIST_CODES_PER_DAY", "")
	if got := countValue("NOT_ON_LIST_CODES_PER_DAY", 50); got != 50 {
		t.Errorf("default = %d, want 50", got)
	}
	t.Setenv("NOT_ON_LIST_CODES_PER_DAY", "300")
	if got := countValue("NOT_ON_LIST_CODES_PER_DAY", 50); got != 300 {
		t.Errorf("set to 300, got %d", got)
	}

	valid := Config{
		AppEnv:               "local",
		DatabaseURL:          "postgres://example",
		GINMode:              "debug",
		Auth:                 AuthConfig{JWTAccessSecret: "a", JWTRefreshSecret: "b", AccessTokenTTL: time.Minute, RefreshTokenTTL: time.Hour},
		RequestBodyLimit:     1024,
		DatabasePool:         DatabasePoolConfig{MaxOpenConns: 10, MaxIdleConns: 5, ConnMaxLifetime: time.Minute, ConnMaxIdleTime: time.Minute},
		NotOnListCodesPerDay: 50,
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid config: %v", err)
	}
	for _, bad := range []string{"0", "-5", "lots"} {
		t.Setenv("NOT_ON_LIST_CODES_PER_DAY", bad)
		config := valid
		config.NotOnListCodesPerDay = countValue("NOT_ON_LIST_CODES_PER_DAY", 50)
		if err := config.Validate(); err == nil || !strings.Contains(err.Error(), "NOT_ON_LIST_CODES_PER_DAY") {
			t.Errorf("%q: Validate() = %v, want it refused", bad, err)
		}
	}
}
