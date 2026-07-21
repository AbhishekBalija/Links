package auth_test

import (
	"strings"
	"testing"

	"github.com/AbhishekBalija/Links/server/internal/auth"
)

func TestValidateUSN_Hardening_SQLInjection(t *testing.T) {
	payloads := []string{
		"4MN20EC001'",
		"4MN20EC001'; DROP TABLE students;--",
		"4MN20EC001\" OR \"1\"=\"1",
		"4MN20EC001\\'; SELECT * FROM users;--",
		"4MN20EC001' UNION SELECT * FROM users--",
	}
	for _, usn := range payloads {
		_, err := auth.ValidateUSN(usn)
		if err == nil {
			t.Errorf("ValidateUSN(%q) should reject SQL-ish payload", usn)
		}
	}
}

func TestValidateUSN_Hardening_NullBytes(t *testing.T) {
	payloads := []string{
		"4MN20EC\x00EC001",
		"4MN20\x00EC001",
		"\x004MN20EC001",
		"4MN20EC001\x00",
	}
	for _, usn := range payloads {
		_, err := auth.ValidateUSN(usn)
		if err == nil {
			t.Errorf("ValidateUSN(%q) should reject null bytes", usn)
		}
	}
}

func TestValidateUSN_Hardening_Unicode(t *testing.T) {
	payloads := []string{
		"4MN20𝐄𝐂001",
		"4MN20EC𝟬𝟬𝟭",
		"𝟒𝐌𝐍𝟐𝟎𝐄𝐂𝟎𝟎𝟏",
		"4MN20EC001™",
		"4MN20ÉC001",
		"4MN20EC001ð",
	}
	for _, usn := range payloads {
		_, err := auth.ValidateUSN(usn)
		if err == nil {
			t.Errorf("ValidateUSN(%q) should reject Unicode lookalikes", usn)
		}
	}
}

func TestValidateUSN_Hardening_Overrun(t *testing.T) {
	payloads := []string{
		"4MN20EC001" + strings.Repeat("X", 1000),
		"4MN20EC001" + strings.Repeat(" ", 1000),
		strings.Repeat("X", 10000),
	}
	for _, usn := range payloads {
		_, err := auth.ValidateUSN(usn)
		if err == nil {
			t.Errorf("ValidateUSN(overrun payload) should reject, got nil")
		}
	}
}

func TestValidateUSN_Hardening_Casing(t *testing.T) {
	payloads := []string{
		strings.ToLower("4MN20EC001"),
		"4mn20ec001",
		"4Mn20eC001",
	}
	for _, usn := range payloads {
		code, err := auth.ValidateUSN(usn)
		if err != nil {
			t.Errorf("ValidateUSN(%q) expected valid (case normalization), got: %v", usn, err)
		}
		if code != "EC" {
			t.Errorf("ValidateUSN(%q) = %q, want EC", usn, code)
		}
	}
}

func TestValidateUSN_Hardening_BoundaryLength(t *testing.T) {
	short := []string{
		"4MN",
		"4MN2",
		"4MN20",
		"4MN20E",
		"4MN20EC",
		"4MN20EC0",
		"4MN20EC00",
	}
	for _, usn := range short {
		_, err := auth.ValidateUSN(usn)
		if err == nil {
			t.Errorf("ValidateUSN(%q) should reject under-length USN", usn)
		}
	}

	long := []string{
		"4MN20EC0001",
		"4MN20EC0012",
	}
	for _, usn := range long {
		_, err := auth.ValidateUSN(usn)
		if err == nil {
			t.Errorf("ValidateUSN(%q) should reject over-length USN", usn)
		}
	}
}
