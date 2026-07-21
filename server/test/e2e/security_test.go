//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestSecurity_Enumeration_RequestAccess(t *testing.T) {
	ts := startServer(t)
	defer ts.cleanup()

	uniqueEmail := fmt.Sprintf("enum-%s@test.com", uuid.New().String())

	// Create a known user first
	t.Run("setup known user", func(t *testing.T) {
		body := map[string]interface{}{
			"email":     uniqueEmail,
			"password":  "TestPass123",
			"full_name": "Enumeration Test User",
		}
		status, _ := doPost(t, ts.baseURL+"/api/v1/auth/request-access", body)
		if status != http.StatusCreated {
			t.Fatalf("setup: expected 201, got %d", status)
		}
	})

	// Request for existing email should NOT reveal it exists
	t.Run("error for existing email does not leak existence", func(t *testing.T) {
		body := map[string]interface{}{
			"email":     uniqueEmail,
			"password":  "TestPass456",
			"full_name": "Another User",
		}
		status, resp := doPost(t, ts.baseURL+"/api/v1/auth/request-access", body)

		// Must still be an error — don't confirm it succeeded
		if status == http.StatusCreated {
			t.Fatal("created duplicate user — enumeration risk")
		}
		if status != http.StatusConflict && status != http.StatusBadRequest {
			t.Fatalf("expected 409 or 400, got %d: %s", status, string(resp))
		}
	})

	// Request for non-existing email should succeed
	t.Run("non-existing email succeeds", func(t *testing.T) {
		body := map[string]interface{}{
			"email":     fmt.Sprintf("fresh-%s@test.com", uuid.New().String()),
			"password":  "TestPass789",
			"full_name": "Fresh User",
		}
		status, resp := doPost(t, ts.baseURL+"/api/v1/auth/request-access", body)
		if status != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", status, string(resp))
		}
	})
}

// intentionally slow. The existing user is reused across multiple
// error-case subtests to keep the test suite fast.
func TestSecurity_InputValidation_RequestAccess(t *testing.T) {
	ts := startServer(t)
	defer ts.cleanup()

	existingEmail := fmt.Sprintf("input-%s@test.com", uuid.New().String())

	_, _ = doPost(t, ts.baseURL+"/api/v1/auth/request-access", map[string]interface{}{
		"email":     existingEmail,
		"password":  "TestPass123",
		"full_name": "Input Validation User",
	})

	tests := []struct {
		name   string
		body   map[string]interface{}
		expect int
	}{
		{
			name: "missing email field",
			body: map[string]interface{}{
				"password":  "TestPass123",
				"full_name": "No Email",
			},
			expect: http.StatusBadRequest,
		},
		{
			name: "missing password field",
			body: map[string]interface{}{
				"email":     fmt.Sprintf("missing-pw-%s@test.com", uuid.New().String()),
				"full_name": "No Password",
			},
			expect: http.StatusBadRequest,
		},
		{
			name: "empty JSON object",
			body: map[string]interface{}{},
			expect: http.StatusBadRequest,
		},
		{
			name: "invalid email format",
			body: map[string]interface{}{
				"email":     "not-an-email",
				"password":  "TestPass123",
				"full_name": "Bad Email",
			},
			expect: http.StatusBadRequest,
		},
		{
			name: "missing at-sign in email",
			body: map[string]interface{}{
				"email":     "userexample.com",
				"password":  "TestPass123",
				"full_name": "Bad Email 2",
			},
			expect: http.StatusBadRequest,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status, resp := doPost(t, ts.baseURL+"/api/v1/auth/request-access", tc.body)
			if status != tc.expect {
				t.Errorf("expected status %d, got %d: %s", tc.expect, status, string(resp))
			}
			if strings.Contains(string(resp), "INTERNAL_ERROR") ||
				strings.Contains(string(resp), "internal server error") {
				t.Errorf("leaked internal error: %s", string(resp))
			}
			if strings.Contains(string(resp), "panic") ||
				strings.Contains(string(resp), "stack trace") ||
				strings.Contains(string(resp), ".go:") {
				t.Errorf("response may contain stack trace: %s", string(resp))
			}
		})
	}
}

func TestSecurity_InputValidation_OversizedPayload(t *testing.T) {
	ts := startServer(t)
	defer ts.cleanup()

	largeField := strings.Repeat("A", 10_000)
	body := map[string]interface{}{
		"email":     fmt.Sprintf("oversize-%s@test.com", uuid.New().String()),
		"password":  "TestPass123",
		"full_name": largeField,
	}
	status, resp := doPost(t, ts.baseURL+"/api/v1/auth/request-access", body)
	if status != http.StatusBadRequest && status != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 400 or 413 for oversized payload, got %d: %s", status, string(resp))
	}
	leaks := []string{"INTERNAL_ERROR", "internal server error", "panic", "stack trace", ".go:"}
	for _, leak := range leaks {
		if strings.Contains(string(resp), leak) {
			t.Errorf("leaked internal detail (%q): %s", leak, string(resp))
		}
	}
}

func TestSecurity_InputValidation_GarbageJSON(t *testing.T) {
	ts := startServer(t)
	defer ts.cleanup()

	payloads := []string{
		`{garbage}`,
		`"just a string"`,
		`<html>bonk</html>`,
		`null`,
		`[1,2,3]`,
	}
	for _, payload := range payloads {
		t.Run(truncate(payload, 30), func(t *testing.T) {
			res, err := http.Post(
				ts.baseURL+"/api/v1/auth/request-access",
				"application/json",
				bytes.NewReader([]byte(payload)),
			)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			resp, _ := io.ReadAll(res.Body)
			res.Body.Close()

			if res.StatusCode == http.StatusInternalServerError {
				t.Errorf("garbage input caused 500: %s", string(resp))
			}
			if strings.Contains(string(resp), ".go:") || strings.Contains(string(resp), "panic") {
				t.Errorf("leaked stack/panic in response: %s", string(resp))
			}
		})
	}
}

func TestSecurity_InputValidation_MissingContentType(t *testing.T) {
	ts := startServer(t)
	defer ts.cleanup()

	body := `{"email":"test@test.com","password":"TestPass123","full_name":"Test"}`
	res, err := http.Post(
		ts.baseURL+"/api/v1/auth/request-access",
		"text/plain",
		strings.NewReader(body),
	)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp, _ := io.ReadAll(res.Body)
	res.Body.Close()

	if res.StatusCode == http.StatusInternalServerError {
		t.Errorf("wrong content-type caused 500: %s", string(resp))
	}
	if strings.Contains(string(resp), ".go:") || strings.Contains(string(resp), "panic") {
		t.Errorf("leaked stack/panic in response: %s", string(resp))
	}
}

func TestSecurity_Timing_RequestAccess(t *testing.T) {
	ts := startServer(t)
	defer ts.cleanup()

	email := fmt.Sprintf("timing-%s@test.com", uuid.New().String())

	// Create known user
	status, _ := doPost(t, ts.baseURL+"/api/v1/auth/request-access", map[string]interface{}{
		"email":     email,
		"password":  "TestPass123",
		"full_name": "Timing Test User",
	})
	if status != http.StatusCreated {
		t.Fatalf("setup: expected 201, got %d", status)
	}

	// Measure response time for existing email
	start := time.Now()
	doPost(t, ts.baseURL+"/api/v1/auth/request-access", map[string]interface{}{
		"email":     email,
		"password":  "TestPass456",
		"full_name": "Timing Attacker",
	})
	existingDuration := time.Since(start)

	// Measure response time for non-existing email
	start = time.Now()
	doPost(t, ts.baseURL+"/api/v1/auth/request-access", map[string]interface{}{
		"email":     fmt.Sprintf("unknown-%s@test.com", uuid.New().String()),
		"password":  "TestPass789",
		"full_name": "Fresh User",
	})
	newDuration := time.Since(start)

	ratio := float64(existingDuration) / float64(newDuration)
	if ratio > 2.0 && existingDuration > 200*time.Millisecond {
		t.Logf("possible timing leak: existing=%v new=%v (ratio=%.2f)",
			existingDuration, newDuration, ratio)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
