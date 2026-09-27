package integration

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

// cookieCall sends a refresh or logout the way a browser would, with the
// refresh cookie and the given Origin and Referer (empty means not sent).
func cookieCall(h *apitest.Harness, path string, cookie *http.Cookie, origin, referer string) apitest.Response {
	request := httptest.NewRequest(http.MethodPost, path, nil)
	request.AddCookie(cookie)
	if origin != "" {
		request.Header.Set("Origin", origin)
	}
	if referer != "" {
		request.Header.Set("Referer", referer)
	}
	return h.Send(request)
}

func TestRefreshAndLogoutRefuseOtherOrigins(t *testing.T) {
	h := apitest.New(t)
	user := student(t, h, "CS", 2023)
	cookie := h.SignIn(t, user)

	refused := map[string][2]string{
		"other origin":          {"https://evil.example", ""},
		"lookalike origin":      {"http://localhost:5173.evil.example", ""},
		"other port":            {"http://localhost:5174", ""},
		"null origin":           {"null", ""},
		"other referer":         {"", "https://evil.example/page"},
		"allowed referer loses": {"https://evil.example", "http://localhost:5173/home"},
	}
	for name, headers := range refused {
		for _, path := range []string{"/api/v1/auth/refresh", "/api/v1/auth/logout"} {
			if response := cookieCall(h, path, cookie, headers[0], headers[1]); response.Status != http.StatusForbidden {
				t.Errorf("%s %s: status = %d, want %d: %s", name, path, response.Status, http.StatusForbidden, response.Body)
			}
		}
	}

	// The refused logout didn't revoke anything, so the session still works.
	if response := cookieCall(h, "/api/v1/auth/refresh", cookie, "http://localhost:5173", ""); response.Status != http.StatusOK {
		t.Fatalf("allowed origin refresh status = %d, want %d: %s", response.Status, http.StatusOK, response.Body)
	}
}

func TestRefreshWorksFromAllowedAndSameOrigins(t *testing.T) {
	h := apitest.New(t)
	user := student(t, h, "CS", 2023)

	allowed := map[string][2]string{
		"configured origin":  {"http://localhost:5173", ""},
		"configured referer": {"", "http://localhost:5173/notices?x=1"},
		"same origin":        {"http://example.com", ""}, // httptest requests go to example.com
		"no browser headers": {"", ""},
	}
	for name, headers := range allowed {
		cookie := h.SignIn(t, user)
		if response := cookieCall(h, "/api/v1/auth/refresh", cookie, headers[0], headers[1]); response.Status != http.StatusOK {
			t.Errorf("%s: status = %d, want %d: %s", name, response.Status, http.StatusOK, response.Body)
		}
	}

	cookie := h.SignIn(t, user)
	if response := cookieCall(h, "/api/v1/auth/logout", cookie, "http://localhost:5173", ""); response.Status != http.StatusOK {
		t.Fatalf("allowed logout status = %d, want %d: %s", response.Status, http.StatusOK, response.Body)
	}
}

func TestEveryAPIResponseCarriesSecurityHeaders(t *testing.T) {
	h := apitest.New(t)
	user := student(t, h, "CS", 2023)

	responses := map[string]apitest.Response{
		"health":       h.Do(t, http.MethodGet, "/api/health", "", nil),
		"success":      h.Do(t, http.MethodGet, "/api/v1/me", user.Token, nil),
		"unauthorized": h.Do(t, http.MethodGet, "/api/v1/me", "", nil),
		"not found":    h.Do(t, http.MethodGet, "/api/v1/no-such-route", "", nil),
	}
	want := map[string]string{
		"X-Content-Type-Options":  "nosniff",
		"Referrer-Policy":         "strict-origin-when-cross-origin",
		"X-Frame-Options":         "DENY",
		"Content-Security-Policy": "default-src 'none'; frame-ancestors 'none'",
	}
	for name, response := range responses {
		for header, value := range want {
			if got := response.Header.Get(header); got != value {
				t.Errorf("%s: %s = %q, want %q", name, header, got, value)
			}
		}
	}
}

func TestCSPReportsAreAcceptedWithoutAToken(t *testing.T) {
	h := apitest.New(t)
	report := map[string]any{"csp-report": map[string]any{
		"document-uri":        "https://links.example/notices",
		"violated-directive":  "img-src 'self' data:",
		"effective-directive": "img-src",
		"blocked-uri":         "https://images.example/avatar.png",
	}}
	if response := h.Do(t, http.MethodPost, "/api/csp-report", "", report); response.Status != http.StatusNoContent {
		t.Fatalf("report status = %d, want %d: %s", response.Status, http.StatusNoContent, response.Body)
	}
	if response := h.Do(t, http.MethodPost, "/api/csp-report", "", "not a report"); response.Status != http.StatusBadRequest {
		t.Fatalf("bad report status = %d, want %d: %s", response.Status, http.StatusBadRequest, response.Body)
	}
}
