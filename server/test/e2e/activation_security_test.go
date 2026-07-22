//go:build e2e

package e2e

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestSecurity_ActivationToken_Expired(t *testing.T) {
	ts := startServer(t)
	defer ts.cleanup()

	email := fmt.Sprintf("expired-%s@test.com", uuid.New().String())
	status, resp := doPost(t, ts.baseURL+"/api/v1/auth/request-access", map[string]interface{}{
		"email":     email,
		"password":  "TestPass123",
		"full_name": "Expired Token User",
	})
	if status != http.StatusCreated {
		t.Fatalf("setup: expected 201, got %d: %s", status, string(resp))
	}

	rawToken := base64.RawURLEncoding.EncodeToString([]byte("test-expired-token"))
	tokenHash := sha256Hex(rawToken)
	ts.gormDB.Exec(
		`INSERT INTO account_activation_tokens (id, user_id, token_hash, expires_at, created_at)
		 VALUES (?, (SELECT id FROM users WHERE email = ?), ?, now() - interval '1 hour', now() - interval '8 days')`,
		uuid.New().String(), email, tokenHash,
	)

	status, resp = doPost(t, ts.baseURL+"/api/v1/auth/activate", map[string]interface{}{
		"token":    rawToken,
		"password": "NewPass123",
	})

	assertGenericActivationError(t, status, resp)
}

func TestSecurity_ActivationToken_AlreadyUsed(t *testing.T) {
	ts := startServer(t)
	defer ts.cleanup()

	email := fmt.Sprintf("reuse-%s@test.com", uuid.New().String())
	status, resp := doPost(t, ts.baseURL+"/api/v1/auth/request-access", map[string]interface{}{
		"email":     email,
		"password":  "TestPass123",
		"full_name": "Reuse Token User",
	})
	if status != http.StatusCreated {
		t.Fatalf("setup: expected 201, got %d: %s", status, string(resp))
	}

	var tokenHash string
	ts.gormDB.Raw(
		`SELECT token_hash FROM account_activation_tokens WHERE user_id = (SELECT id FROM users WHERE email = ?) ORDER BY created_at DESC LIMIT 1`,
		email,
	).Scan(&tokenHash)
	if tokenHash == "" {
		t.Fatal("no activation token found in DB")
	}

	rawToken := "token-from-email"
	ts.gormDB.Exec(
		`UPDATE account_activation_tokens SET token_hash = ? WHERE user_id = (SELECT id FROM users WHERE email = ?)`,
		sha256Hex(rawToken), email,
	)

	status, resp = doPost(t, ts.baseURL+"/api/v1/auth/activate", map[string]interface{}{
		"token":    rawToken,
		"password": "NewPass123",
	})
	if status != http.StatusOK {
		t.Fatalf("first activate: expected 200, got %d: %s", status, string(resp))
	}

	status, resp = doPost(t, ts.baseURL+"/api/v1/auth/activate", map[string]interface{}{
		"token":    rawToken,
		"password": "NewPass456",
	})

	assertGenericActivationError(t, status, resp)
}

func TestSecurity_ActivationToken_Malformed(t *testing.T) {
	ts := startServer(t)
	defer ts.cleanup()

	payloads := []string{
		"",
		"short",
		base64.RawURLEncoding.EncodeToString([]byte("not-a-valid-token")),
		"../../../etc/passwd",
		"' OR '1'='1",
		"<script>alert(1)</script>",
	}
	for _, payload := range payloads {
		t.Run(truncate(payload, 20), func(t *testing.T) {
			status, resp := doPost(t, ts.baseURL+"/api/v1/auth/activate", map[string]interface{}{
				"token":    payload,
				"password": "TestPass123",
			})
			assertGenericActivationError(t, status, resp)
		})
	}
}

func TestSecurity_ResendActivation_Enumeration(t *testing.T) {
	ts := startServer(t)
	defer ts.cleanup()

	registeredEmail := fmt.Sprintf("enum-resend-%s@test.com", uuid.New().String())
	_, _ = doPost(t, ts.baseURL+"/api/v1/auth/request-access", map[string]interface{}{
		"email":     registeredEmail,
		"password":  "TestPass123",
		"full_name": "Enum Resend User",
	})

	unknownEmail := fmt.Sprintf("unknown-%s@test.com", uuid.New().String())

	statusKnown, bodyKnown := doPost(t, ts.baseURL+"/api/v1/auth/resend-activation", map[string]interface{}{
		"email": registeredEmail,
	})
	statusUnknown, bodyUnknown := doPost(t, ts.baseURL+"/api/v1/auth/resend-activation", map[string]interface{}{
		"email": unknownEmail,
	})

	if statusKnown != statusUnknown {
		t.Errorf("resend status differs: known=%d unknown=%d", statusKnown, statusUnknown)
	}
	if string(bodyKnown) != string(bodyUnknown) {
		t.Errorf("resend response body differs: known=%s unknown=%s", string(bodyKnown), string(bodyUnknown))
	}
}

func assertGenericActivationError(t *testing.T, status int, body []byte) {
	t.Helper()
	if status != http.StatusUnauthorized {
		t.Errorf("expected 401 for activation error, got %d: %s", status, string(body))
	}
}

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h)
}
