package auth_test

import (
	"strings"
	"testing"

	"github.com/AbhishekBalija/Links/server/internal/auth"
)

func TestHashPassword_ValidPassword(t *testing.T) {
	hash, err := auth.HashPassword("Secure1pass")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Errorf("expected PHC-format hash, got: %s", hash)
	}
}

func TestHashPassword_UniqueSalts(t *testing.T) {
	h1, err := auth.HashPassword("Secure1pass")
	if err != nil {
		t.Fatalf("hash 1: %v", err)
	}
	h2, err := auth.HashPassword("Secure1pass")
	if err != nil {
		t.Fatalf("hash 2: %v", err)
	}
	if h1 == h2 {
		t.Error("two hashes of the same password must differ (unique salt)")
	}
}

func TestVerifyPassword_CorrectPassword(t *testing.T) {
	hash, err := auth.HashPassword("Correct1pw")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	ok, err := auth.VerifyPassword("Correct1pw", hash)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !ok {
		t.Error("expected password to match")
	}
}

func TestVerifyPassword_WrongPassword(t *testing.T) {
	hash, err := auth.HashPassword("Correct1pw")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	ok, err := auth.VerifyPassword("Wrong1pass", hash)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if ok {
		t.Error("expected wrong password to not match")
	}
}

func TestVerifyPassword_InvalidHash(t *testing.T) {
	_, err := auth.VerifyPassword("anything", "not-a-hash")
	if err == nil {
		t.Error("expected error for invalid hash format")
	}
}

func TestValidatePasswordStrength(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"valid", "Abcdef1x", false},
		{"too short", "Ab1x", true},
		{"no uppercase", "abcdef1x", true},
		{"no lowercase", "ABCDEF1X", true},
		{"no digit", "Abcdefxx", true},
		{"empty", "", true},
		{"exactly 8 chars valid", "Abcdef1x", false},
		{"long valid", "ThisIsAVeryLong1Password", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := auth.ValidatePasswordStrength(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePasswordStrength(%q) error = %v, wantErr = %v", tt.password, err, tt.wantErr)
			}
		})
	}
}

func TestHashPassword_WeakPasswordRejected(t *testing.T) {
	_, err := auth.HashPassword("weak")
	if err == nil {
		t.Error("expected error for weak password")
	}
}
