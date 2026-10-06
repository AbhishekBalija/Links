package config

import (
	"os"
	"regexp"
	"sort"
	"testing"
)

// Variables the hosting platform sets itself, so nobody writes them in an
// example file.
var platformVariables = map[string]bool{
	"VERCEL":                        true,
	"VERCEL_ENV":                    true,
	"VERCEL_URL":                    true,
	"VERCEL_PROJECT_PRODUCTION_URL": true,
}

// Settings that only make sense on a developer's machine, so the production
// example leaves them out.
var localOnlyVariables = map[string]bool{
	"APP_PORT":                  true,
	"ENABLE_TEST_SIGN_IN":       true,
	"EMAIL_CODE_FOR_EVERY_ROLE": true,
	"DB_HOST":                   true,
	"DB_PORT":                   true,
	"DB_USER":                   true,
	"DB_NAME":                   true,
	"DB_PASSWORD":               true,
	"DB_SSLMODE":                true,
}

// A name like "JWT_ACCESS_SECRET" in quotes: every variable the server reads
// is written this way in the code.
var variableNamePattern = regexp.MustCompile(`"([A-Z][A-Z0-9]*(?:_[A-Z0-9]+)*)"`)

// Every variable the server reads must be listed in the example files and in
// docs/environment.md, so a new copy can be set up from them alone (#186).
func TestEveryVariableTheServerReadsIsDocumented(t *testing.T) {
	t.Parallel()

	// config.go reads nearly everything; main.go reads SENTRY_DSN.
	found := map[string]bool{}
	for _, path := range []string{"config.go", "../../cmd/api/main.go"} {
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		for _, match := range variableNamePattern.FindAllSubmatch(source, -1) {
			name := string(match[1])
			// Skip short words such as "GET" that are not variable names.
			if len(name) < 4 || platformVariables[name] {
				continue
			}
			found[name] = true
		}
	}
	if len(found) < 20 {
		t.Fatalf("found only %d variable names, the pattern may be broken", len(found))
	}
	names := make([]string, 0, len(found))
	for name := range found {
		names = append(names, name)
	}
	sort.Strings(names)

	// In an example file a key starts a line, with or without a leading "# "
	// (a commented-out setting). In the docs it only has to be mentioned.
	files := []struct {
		path      string
		before    string
		after     string
		skipLocal bool
	}{
		{"../../.env.local.example", `(?m)^#?\s*`, "=", false},
		{"../../.env.prod.example", `(?m)^#?\s*`, "=", true},
		{"../../../docs/environment.md", "", "", false},
	}
	for _, file := range files {
		content, err := os.ReadFile(file.path)
		if err != nil {
			t.Fatalf("read %s: %v", file.path, err)
		}
		for _, name := range names {
			if file.skipLocal && localOnlyVariables[name] {
				continue
			}
			pattern := file.before + regexp.QuoteMeta(name) + file.after
			if !regexp.MustCompile(pattern).Match(content) {
				t.Errorf("%s does not list %s", file.path, name)
			}
		}
	}
}
