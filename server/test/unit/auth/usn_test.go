package auth_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/AbhishekBalija/Links/server/internal/auth"
)

func TestValidateUSNFormat_Valid(t *testing.T) {
	tests := []struct {
		usn      string
		expected string
	}{
		{"4MN20EC002", "EC"},
		{"4MN21CS042", "CS"},
		{"4MN20AD001", "AD"},
		{"4MN23AI015", "AI"},
		{"4MN19CV100", "CV"},
		{"4MN22ME007", "ME"},
		{"4mn20ec002", "EC"},
		{"4Mn20Ec002", "EC"},
	}
	for _, tc := range tests {
		code, err := auth.ValidateUSNFormat(tc.usn)
		if err != nil {
			t.Errorf("ValidateUSNFormat(%q) unexpected error: %v", tc.usn, err)
		}
		if code != tc.expected {
			t.Errorf("ValidateUSNFormat(%q) = %q, want %q", tc.usn, code, tc.expected)
		}
	}
}

func TestValidateUSNFormat_YearBoundary(t *testing.T) {
	nowYear := time.Now().Year()
	maxYY := (nowYear + 2) % 100
	minYY := 5

	for _, usn := range []string{
		fmt.Sprintf("4MN%02dEC001", minYY),
		fmt.Sprintf("4MN%02dEC001", maxYY),
	} {
		_, err := auth.ValidateUSNFormat(usn)
		if err != nil {
			t.Errorf("ValidateUSNFormat(%q) expected valid, got: %v", usn, err)
		}
	}

	for _, usn := range []string{
		fmt.Sprintf("4MN%02dEC001", minYY-1),
		fmt.Sprintf("4MN%02dEC001", maxYY+1),
	} {
		_, err := auth.ValidateUSNFormat(usn)
		if err == nil {
			t.Errorf("ValidateUSNFormat(%q) expected year-range error, got nil", usn)
		}
	}
}

func TestValidateUSNFormat_InvalidFormat(t *testing.T) {
	invalid := []string{
		"",
		"4MN20002",
		"4MN20C5002",
		"4MN20ec002x",
		"ABC123",
		"4MN20EC00",
		"XMN20EC002",
		"4XX20EC002",
	}
	for _, usn := range invalid {
		_, err := auth.ValidateUSNFormat(usn)
		if err == nil {
			t.Errorf("ValidateUSNFormat(%q) expected error, got nil", usn)
		}
	}
}

// Whether a department code exists is the departments table's call, not the
// format check's, so a department an admin adds works without a code change.
func TestValidateUSNFormat_AcceptsAnyTwoLetterDepartmentCode(t *testing.T) {
	for usn, want := range map[string]string{"4MN20MB002": "MB", "4MN24IS001": "IS", "4mn21xx033": "XX"} {
		code, err := auth.ValidateUSNFormat(usn)
		if err != nil || code != want {
			t.Errorf("ValidateUSNFormat(%q) = %q, %v; want %q", usn, code, err, want)
		}
	}
}

func TestBatchYearFromUSN(t *testing.T) {
	tests := []struct {
		usn  string
		want int
	}{
		{"4MN23CS001", 2023},
		{"4MN20EC002", 2020},
		{"4mn21ec042", 2021},
		{"4MN05ME100", 2005},
		// Lateral entry (roll 400+) is assumed to carry the batch's year,
		// not the year they joined second year (ADR 0019).
		{"4MN21EC401", 2021},
	}
	for _, test := range tests {
		got, err := auth.BatchYearFromUSN(test.usn)
		if err != nil || got != test.want {
			t.Errorf("BatchYearFromUSN(%q) = %d, %v; want %d", test.usn, got, err, test.want)
		}
	}
	if _, err := auth.BatchYearFromUSN("not-a-usn"); err == nil {
		t.Error("BatchYearFromUSN accepted an invalid USN")
	}
}
