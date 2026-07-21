package auth_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/AbhishekBalija/Links/server/internal/auth"
)

func TestValidateUSN_Valid(t *testing.T) {
	tests := []struct {
		usn      string
		expected string
	}{
		{"4MN20EC002", "EC"},
		{"4MN21CS042", "CS"},
		{"4MN20AD001", "AD"},
		{"4MN19CV100", "CV"},
		{"4MN22ME007", "ME"},
		{"4MN21CI033", "CI"},
		{"4mn20ec002", "EC"},
		{"4Mn20Ec002", "EC"},
	}
	for _, tc := range tests {
		code, err := auth.ValidateUSN(tc.usn)
		if err != nil {
			t.Errorf("ValidateUSN(%q) unexpected error: %v", tc.usn, err)
		}
		if code != tc.expected {
			t.Errorf("ValidateUSN(%q) = %q, want %q", tc.usn, code, tc.expected)
		}
	}
}

func TestValidateUSN_YearBoundary(t *testing.T) {
	nowYear := time.Now().Year()
	maxYY := (nowYear + 2) % 100
	minYY := 5

	for _, usn := range []string{
		fmt.Sprintf("4MN%02dEC001", minYY),
		fmt.Sprintf("4MN%02dEC001", maxYY),
	} {
		_, err := auth.ValidateUSN(usn)
		if err != nil {
			t.Errorf("ValidateUSN(%q) expected valid, got: %v", usn, err)
		}
	}

	for _, usn := range []string{
		fmt.Sprintf("4MN%02dEC001", minYY-1),
		fmt.Sprintf("4MN%02dEC001", maxYY+1),
	} {
		_, err := auth.ValidateUSN(usn)
		if err == nil {
			t.Errorf("ValidateUSN(%q) expected year-range error, got nil", usn)
		}
	}
}

func TestValidateUSN_InvalidFormat(t *testing.T) {
	invalid := []string{
		"",
		"4MN20XX002",
		"4MN20002",
		"ABC123",
		"4MN20EC00",
		"XMN20EC002",
		"4XX20EC002",
	}
	for _, usn := range invalid {
		_, err := auth.ValidateUSN(usn)
		if err == nil {
			t.Errorf("ValidateUSN(%q) expected error, got nil", usn)
		}
	}
}

func TestValidateUSN_UnknownDepartment(t *testing.T) {
	_, err := auth.ValidateUSN("4MN20MB002")
	if err == nil {
		t.Error("ValidateUSN(4MN20MB002) expected error for unknown dept code MB, got nil")
	}
}
