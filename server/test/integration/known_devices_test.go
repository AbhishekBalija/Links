package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/AbhishekBalija/Links/server/internal/auth"
	"github.com/AbhishekBalija/Links/server/test/apitest"
)

// A browser that has signed in to an account before is a known device for
// it: its own code requests get their own allowance, so nobody else, even on
// the same campus Wi-Fi, can stop it getting codes (#178 follow-up).

const campusIP = "100.64.0.10"

const deviceCookie = "links_device"

// askFromBrowser asks for a code from an address, with the browser's device
// cookie when it has one.
func askFromBrowser(t *testing.T, h *apitest.Harness, ip, email, device string) apitest.Response {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/code", strings.NewReader(fmt.Sprintf(`{"email":%q}`, email)))
	request.Header.Set("Content-Type", "application/json")
	request.RemoteAddr = ip + ":40000"
	if device != "" {
		request.AddCookie(&http.Cookie{Name: deviceCookie, Value: device})
	}
	return h.Send(request)
}

// signInOnce signs in with a code from a new browser and returns the device
// cookie it was given.
func signInOnce(t *testing.T, h *apitest.Harness, ip, email string) string {
	t.Helper()
	challenge := challengeFrom(t, askFromBrowser(t, h, ip, email, ""))
	response := enterCode(t, h, challenge, h.Outbox.LastCodeTo(t, email))
	expectStatus(t, "sign in", response, http.StatusOK)
	for _, cookie := range response.Cookies() {
		if cookie.Name == deviceCookie && cookie.Value != "" {
			if !cookie.HttpOnly || cookie.Path != "/api/v1/auth" {
				t.Errorf("device cookie = %+v, want httpOnly on /api/v1/auth", cookie)
			}
			return cookie.Value
		}
	}
	t.Fatalf("signing in set no %s cookie", deviceCookie)
	return ""
}

// blockFromCampus has a stranger on the campus address use up every limit
// that address has for the email: requests, wrong guesses, and the address's
// own allowance.
func blockFromCampus(t *testing.T, h *apitest.Harness, email string) {
	t.Helper()
	for round := 1; round <= 2; round++ {
		challenge := challengeFrom(t, askFromBrowser(t, h, campusIP, email, ""))
		code := h.Outbox.LastCodeTo(t, email)
		for try := 1; try <= 5; try++ {
			enterCode(t, h, challenge, wrongCode(code))
		}
	}
	for i := 0; i < auth.DefaultCodeSettings().PerIPLimit; i++ {
		askFromBrowser(t, h, campusIP, fmt.Sprintf("someone%d@apitest.local", i), "")
	}
	expectStatus(t, "the stranger is stopped", askFromBrowser(t, h, campusIP, email, ""), http.StatusTooManyRequests)
}

func TestAKnownDeviceGetsCodesWhateverOthersOnItsNetworkDo(t *testing.T) {
	h := apitest.New(t)
	member := studentOf(t, h, "CS", 2023)
	device := signInOnce(t, h, campusIP, member.Email)
	sent := len(h.Outbox.CodesTo(member.Email))

	blockFromCampus(t, h, member.Email)
	sent = len(h.Outbox.CodesTo(member.Email))

	challenge := challengeFrom(t, askFromBrowser(t, h, campusIP, member.Email, device))
	if codes := h.Outbox.CodesTo(member.Email); len(codes) != sent+1 {
		t.Fatalf("the member's own browser got no code: %d sent, want %d", len(codes), sent+1)
	}
	expectStatus(t, "the member signs in", enterCode(t, h, challenge, h.Outbox.LastCodeTo(t, member.Email)), http.StatusOK)
}

func TestADeviceIsKnownOnlyForItsOwnAccount(t *testing.T) {
	h := apitest.New(t)
	member, stranger := studentOf(t, h, "CS", 2023), studentOf(t, h, "CS", 2023)
	strangersDevice := signInOnce(t, h, "203.0.113.50", stranger.Email)

	blockFromCampus(t, h, member.Email)
	// The stranger's own device cookie doesn't open the member's allowance.
	expectStatus(t, "another account's device", askFromBrowser(t, h, campusIP, member.Email, strangersDevice), http.StatusTooManyRequests)
	expectStatus(t, "a made-up device", askFromBrowser(t, h, campusIP, member.Email, "not-a-device"), http.StatusTooManyRequests)
}

func TestAKnownDeviceStillHasLimitsOfItsOwn(t *testing.T) {
	h := apitest.New(t)
	member := studentOf(t, h, "CS", 2023)
	device := signInOnce(t, h, campusIP, member.Email)
	ageCodes(h, "16 minutes")

	for range 3 {
		challengeFrom(t, askFromBrowser(t, h, campusIP, member.Email, device))
	}
	expectStatus(t, "a fourth in fifteen minutes", askFromBrowser(t, h, campusIP, member.Email, device), http.StatusTooManyRequests)
}

func TestTheDeviceTokenIsStoredOnlyAsAHash(t *testing.T) {
	h := apitest.New(t)
	member := studentOf(t, h, "CS", 2023)
	device := signInOnce(t, h, campusIP, member.Email)

	var stored []string
	h.DB().Raw(`SELECT token_hash FROM known_devices`).Scan(&stored)
	if len(stored) != 1 || stored[0] == device || strings.Contains(stored[0], device) {
		t.Errorf("stored = %v, want one hash that isn't the token", stored)
	}
}
