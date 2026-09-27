package auth

import (
	"fmt"
	"regexp"
	"time"
)

// Note this regex can change when I decide to make this platform support all colleges under VTU/Karnataka. For now, it is hardcoded to MITT's 4MNxx format.
var usnRegex = regexp.MustCompile(`^4MN(\d{2})([A-Z]{2})(\d{3})$`)

// ValidateUSNFormat checks that a USN has the VTU shape and a joining year in
// range, and returns its department code. It doesn't know which departments
// exist: those live in the departments table, so the caller checks the code
// there (a department an admin adds works without a code change, #18).
// Normalizes to uppercase before validation. MBA/MCA are intentionally
// excluded; they use a different ID scheme.
func ValidateUSNFormat(usn string) (string, error) {
	normalized := toUSNCase(usn)

	matches := usnRegex.FindStringSubmatch(normalized)
	if matches == nil {
		return "", fmt.Errorf("invalid USN format")
	}

	year := matches[1]
	yy := 2000 + int(year[0]-'0')*10 + int(year[1]-'0')
	nowYear := time.Now().Year()
	if yy < 2005 || yy > nowYear+2 {
		return "", fmt.Errorf("USN year %s is out of valid range (2005-%d)", year, nowYear+2)
	}

	return matches[2], nil
}

// BatchYearFromUSN reads a student's Batch from the joining year in their
// USN: 4MN23CS001 joined in 2023. An admin can later move a year-back
// student to another Batch (ADR 0019); this is only where it starts.
func BatchYearFromUSN(usn string) (int, error) {
	matches := usnRegex.FindStringSubmatch(toUSNCase(usn))
	if matches == nil {
		return 0, fmt.Errorf("invalid USN format")
	}
	year := matches[1]
	return 2000 + int(year[0]-'0')*10 + int(year[1]-'0'), nil
}

// toUSNCase normalizes a USN to uppercase (matching the existing convention
// documented in database-design.md and migration 004).
func toUSNCase(usn string) string {
	b := make([]byte, len(usn))
	for i := range len(usn) {
		c := usn[i]
		if c >= 'a' && c <= 'z' {
			c -= 32
		}
		b[i] = c
	}
	return string(b)
}
