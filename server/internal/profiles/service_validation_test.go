package profiles

import "testing"

func TestValidateProfileURLsAllowsEmptyValuesForClearing(t *testing.T) {
	empty := ""
	input := UpdateProfileInput{
		AvatarURL:    &empty,
		LinkedInURL:  &empty,
		GitHubURL:    &empty,
		PortfolioURL: &empty,
	}

	if err := validateProfileURLs(input); err != nil {
		t.Fatalf("expected empty URLs to be valid, got %v", err)
	}
}

func TestValidateProfileURLsAcceptsHTTPAndHTTPS(t *testing.T) {
	httpURL := "http://example.com/avatar.png"
	httpsURL := "https://example.com/profile"
	input := UpdateProfileInput{
		AvatarURL:   &httpURL,
		LinkedInURL: &httpsURL,
	}

	if err := validateProfileURLs(input); err != nil {
		t.Fatalf("expected HTTP and HTTPS URLs to be valid, got %v", err)
	}
}

func TestValidateProfileURLsRejectsUnsupportedOrRelativeURLs(t *testing.T) {
	tests := []string{
		"javascript:alert(1)",
		"/relative/path",
		"not a url",
	}

	for _, value := range tests {
		t.Run(value, func(t *testing.T) {
			input := UpdateProfileInput{PortfolioURL: &value}
			if err := validateProfileURLs(input); err == nil {
				t.Fatal("expected invalid URL to be rejected")
			}
		})
	}
}
