package auth

import (
	"fmt"
	"regexp"
	"time"
)

// Note this regex can change when I decide to make this platform support all colleges under VTU/Karnataka. For now, it is hardcoded to MITT's 4MNxx format.
var usnRegex = regexp.MustCompile(`^4MN(\d{2})([A-Z]{2})(\d{3})$`)

// Department codes confirmed against VTU's official stream-wise course code list (vtu.ac.in).
// Key: 2-letter code, Value: full department name.
// Add new codes here — the validation regex and lookup stay in sync automatically.
var departmentCodes = map[string]string{
	"CS": "Computer Science & Engineering",
	"AD": "Artificial Intelligence & Data Science",
	"CV": "Civil Engineering",
	"ME": "Mechanical Engineering",
	"EC": "Electronics & Communication Engineering",
	// CI is accepted by the parser but is not seeded until MITT confirms an
	// actual 4MNxxCIxxx student USN.
	"CI": "CSE (Artificial Intelligence & Machine Learning)",
}

// ValidateUSN checks that a USN matches the VTU format and uses a known
// department code. Normalizes to uppercase before validation.
// Returns the extracted department code and nil on success.
// MBA/MCA are intentionally excluded — they use a different ID scheme.
func ValidateUSN(usn string) (string, error) {
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

	code := matches[2]
	if _, ok := departmentCodes[code]; !ok {
		return "", fmt.Errorf("unknown department code: %s", code)
	}

	return code, nil
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
