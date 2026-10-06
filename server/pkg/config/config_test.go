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
	secureProduction.Google.ClientID = "client-id"
	secureProduction.Mailer = MailerConfig{Provider: "resend", ResendAPIKey: "re_key", FromEmail: "noreply@college.example"}
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

func TestCodesPerNetworkIsASettingOfEachCopy(t *testing.T) {
	t.Setenv("CODES_PER_NETWORK_PER_15_MIN", "")
	if got := countValue("CODES_PER_NETWORK_PER_15_MIN", 600); got != 600 {
		t.Errorf("default = %d, want 600", got)
	}
	config := Config{AppEnv: "local", DatabaseURL: "postgres://example", GINMode: "debug",
		Auth:             AuthConfig{JWTAccessSecret: "a", JWTRefreshSecret: "b", AccessTokenTTL: time.Minute, RefreshTokenTTL: time.Hour},
		RequestBodyLimit: 1024,
		DatabasePool:     DatabasePoolConfig{MaxOpenConns: 10, MaxIdleConns: 5, ConnMaxLifetime: time.Minute, ConnMaxIdleTime: time.Minute},
		CodesPerNetwork:  -1,
	}
	if err := config.Validate(); err == nil || !strings.Contains(err.Error(), "CODES_PER_NETWORK_PER_15_MIN") {
		t.Errorf("Validate() = %v, want a bad value refused", err)
	}
}

// productionConfig is a complete production copy: Google sign-in and Resend
// mail are both set up, so people can actually sign in.
func productionConfig() Config {
	return Config{
		AppEnv:      "production",
		DatabaseURL: "postgres://example",
		GINMode:     "release",
		Auth: AuthConfig{
			JWTAccessSecret:  "access-secret",
			JWTRefreshSecret: "refresh-secret",
			AccessTokenTTL:   15 * time.Minute,
			RefreshTokenTTL:  7 * 24 * time.Hour,
		},
		Cookie:           CookieConfig{Secure: true, SameSite: "lax"},
		RequestBodyLimit: 1024,
		DatabasePool:     DatabasePoolConfig{MaxOpenConns: 10, MaxIdleConns: 5, ConnMaxLifetime: time.Minute, ConnMaxIdleTime: time.Minute},
		Google:           GoogleConfig{ClientID: "client-id.apps.googleusercontent.com"},
		Mailer:           MailerConfig{Provider: "resend", ResendAPIKey: "re_key", FromEmail: "noreply@college.example"},
	}
}

// A production copy that can't send codes or sign in with Google would start
// green and lock people out, so it must refuse to start (#182).
func TestProductionRefusesToStartWithoutSignInSettings(t *testing.T) {
	t.Parallel()
	if err := productionConfig().Validate(); err != nil {
		t.Fatalf("complete production config refused: %v", err)
	}

	// The Resend sandbox sender only reaches the account owner, but a copy
	// without a verified domain yet still has to start; the server warns.
	sandbox := productionConfig()
	sandbox.Mailer.FromEmail = "onboarding@resend.dev"
	if err := sandbox.Validate(); err != nil {
		t.Fatalf("the @resend.dev sender was refused: %v", err)
	}

	smtp := productionConfig()
	smtp.Mailer = MailerConfig{Provider: "smtp", SMTPHost: "smtp.example", SMTPPort: "587", SMTPUsername: "user", SMTPPassword: "pass", FromEmail: "noreply@college.example"}
	if err := smtp.Validate(); err != nil {
		t.Fatalf("complete SMTP production config refused: %v", err)
	}

	cases := []struct {
		name   string
		change func(*Config)
		want   string
	}{
		{"no Google client ID", func(c *Config) { c.Google.ClientID = "" }, "GOOGLE_CLIENT_ID"},
		{"blank Google client ID", func(c *Config) { c.Google.ClientID = "  " }, "GOOGLE_CLIENT_ID"},
		{"no Resend key", func(c *Config) { c.Mailer.ResendAPIKey = "" }, "RESEND_API_KEY"},
		{"no from email", func(c *Config) { c.Mailer.FromEmail = "" }, "FROM_EMAIL"},
		{"default mail provider with no key", func(c *Config) { c.Mailer.Provider = ""; c.Mailer.ResendAPIKey = "" }, "RESEND_API_KEY"},
		{"smtp without username", func(c *Config) { *c = smtp; c.Mailer.SMTPUsername = "" }, "SMTP_USERNAME"},
		{"smtp without password", func(c *Config) { *c = smtp; c.Mailer.SMTPPassword = "" }, "SMTP_PASSWORD"},
		{"smtp without from email", func(c *Config) { *c = smtp; c.Mailer.FromEmail = "" }, "FROM_EMAIL"},
	}
	for _, tc := range cases {
		cfg := productionConfig()
		tc.change(&cfg)
		if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: Validate() = %v, want an error naming %s", tc.name, err, tc.want)
		}
	}
}

// Local and preview copies stay easy to start: no Google, no mail key.
func TestNonProductionStartsWithoutSignInSettings(t *testing.T) {
	t.Parallel()
	for _, env := range []string{"local", "preview"} {
		cfg := productionConfig()
		cfg.AppEnv = env
		cfg.Google.ClientID = ""
		cfg.Mailer = MailerConfig{}
		if err := cfg.Validate(); err != nil {
			t.Errorf("APP_ENV=%s without sign-in settings: %v", env, err)
		}
	}
}
