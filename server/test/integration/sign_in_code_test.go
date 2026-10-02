package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/AbhishekBalija/Links/server/internal/auth"
	"github.com/AbhishekBalija/Links/server/test/apitest"
)

const codeSentMessage = "If this email can use LINKS, a code is on its way."

type codeReply struct {
	Data struct {
		ChallengeID string `json:"challenge_id"`
		Message     string `json:"message"`
	} `json:"data"`
}

// askForCode asks for a sign-in code the way the screen does and returns the
// challenge the browser keeps.
func askForCode(t *testing.T, h *apitest.Harness, email string) string {
	t.Helper()
	response := h.Do(t, http.MethodPost, "/api/v1/auth/code", "", map[string]string{"email": email})
	expectStatus(t, "ask for a code", response, http.StatusOK)
	var reply codeReply
	response.Decode(t, &reply)
	if reply.Data.ChallengeID == "" {
		t.Fatalf("no challenge_id in %s", response.Body)
	}
	if reply.Data.Message != codeSentMessage {
		t.Errorf("message = %q, want %q", reply.Data.Message, codeSentMessage)
	}
	return reply.Data.ChallengeID
}

// askForCodeFrom asks for a code from a given IP address.
func askForCodeFrom(t *testing.T, h *apitest.Harness, ip, email string) apitest.Response {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/code", strings.NewReader(fmt.Sprintf(`{"email":%q}`, email)))
	request.Header.Set("Content-Type", "application/json")
	request.RemoteAddr = ip + ":40000"
	return h.Send(request)
}

func enterCode(t *testing.T, h *apitest.Harness, challengeID, code string) apitest.Response {
	t.Helper()
	return h.Do(t, http.MethodPost, "/api/v1/auth/code/verify", "", map[string]string{"challenge_id": challengeID, "code": code})
}

func wrongCode(code string) string {
	if code == "000000" {
		return "000001"
	}
	return "000000"
}

func TestAnEmailCodeSignsAnActiveMemberIn(t *testing.T) {
	h := apitest.New(t)
	member := studentOf(t, h, "CS", 2023)

	challenge := askForCode(t, h, member.Email)
	code := h.Outbox.LastCodeTo(t, member.Email)
	if !regexp.MustCompile(`^[0-9]{6}$`).MatchString(code) {
		t.Fatalf("code = %q, want 6 digits", code)
	}

	response := enterCode(t, h, challenge, code)
	expectStatus(t, "enter the code", response, http.StatusOK)
	var session struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	response.Decode(t, &session)
	expectStatus(t, "the token works", h.Do(t, http.MethodGet, "/api/v1/me", session.Data.AccessToken, nil), http.StatusOK)
	var hasRefresh bool
	for _, cookie := range response.Cookies() {
		hasRefresh = hasRefresh || (cookie.Name == "refresh_token" && cookie.HttpOnly && cookie.Value != "")
	}
	if !hasRefresh {
		t.Error("no httpOnly refresh cookie, so the app couldn't stay signed in")
	}

	var methods []string
	h.DB().Raw(`SELECT metadata->>'method' FROM audit_logs WHERE action = 'auth.signed_in' AND actor_id = ?`, member.ID).Scan(&methods)
	if len(methods) != 1 || methods[0] != "email_code" {
		t.Errorf("sign-in audit methods = %v, want one email_code", methods)
	}
}

func TestTheCodeReplyIsTheSameForAnUnknownEmail(t *testing.T) {
	h := apitest.New(t)
	member := studentOf(t, h, "CS", 2023)

	known := h.Do(t, http.MethodPost, "/api/v1/auth/code", "", map[string]string{"email": member.Email})
	unknown := h.Do(t, http.MethodPost, "/api/v1/auth/code", "", map[string]string{"email": "nobody@apitest.local"})
	if known.Status != unknown.Status {
		t.Fatalf("status known = %d, unknown = %d", known.Status, unknown.Status)
	}
	var knownReply, unknownReply codeReply
	known.Decode(t, &knownReply)
	unknown.Decode(t, &unknownReply)
	if knownReply.Data.Message != unknownReply.Data.Message || unknownReply.Data.ChallengeID == "" {
		t.Errorf("replies differ: %s vs %s", known.Body, unknown.Body)
	}
	if codes := h.Outbox.CodesTo("nobody@apitest.local"); len(codes) != 0 {
		t.Errorf("sent %d codes to an unknown email", len(codes))
	}
	for _, guess := range []string{"000000", "123456"} {
		expectStatus(t, "a guess for an unknown email", enterCode(t, h, unknownReply.Data.ChallengeID, guess), http.StatusUnauthorized)
	}
}

func TestEmailMatchingIgnoresCaseAndSpaces(t *testing.T) {
	h := apitest.New(t)
	member := studentOf(t, h, "CS", 2023)

	challenge := askForCode(t, h, "  "+strings.ToUpper(member.Email)+" ")
	codes := h.Outbox.CodesTo(member.Email)
	if len(codes) != 1 {
		t.Fatalf("sent %d codes to the stored address, want 1", len(codes))
	}
	expectStatus(t, "enter the code", enterCode(t, h, challenge, codes[0]), http.StatusOK)
}

func TestACodeWorksOnlyOnce(t *testing.T) {
	h := apitest.New(t)
	member := studentOf(t, h, "CS", 2023)

	challenge := askForCode(t, h, member.Email)
	code := h.Outbox.LastCodeTo(t, member.Email)
	expectStatus(t, "first use", enterCode(t, h, challenge, code), http.StatusOK)
	expectStatus(t, "second use", enterCode(t, h, challenge, code), http.StatusUnauthorized)
}

func TestACodeExpiresAfterTenMinutes(t *testing.T) {
	h := apitest.New(t)
	member := studentOf(t, h, "CS", 2023)

	challenge := askForCode(t, h, member.Email)
	code := h.Outbox.LastCodeTo(t, member.Email)
	var minutes float64
	h.DB().Raw(`SELECT extract(epoch FROM expires_at - created_at) / 60 FROM sign_in_codes WHERE id = ?`, challenge).Scan(&minutes)
	if minutes != 10 {
		t.Errorf("code lives %v minutes, want 10", minutes)
	}
	// Eleven minutes pass.
	h.DB().Exec(`UPDATE sign_in_codes SET created_at = created_at - interval '11 minutes', expires_at = expires_at - interval '11 minutes' WHERE id = ?`, challenge)

	response := enterCode(t, h, challenge, code)
	expectStatus(t, "expired code", response, http.StatusUnauthorized)
	wrong := enterCode(t, h, challenge, wrongCode(code))
	if response.Body != wrong.Body {
		t.Errorf("an expired code and a wrong code give different replies: %s vs %s", response.Body, wrong.Body)
	}
}

func TestFiveWrongTriesKillTheCode(t *testing.T) {
	h := apitest.New(t)
	member := studentOf(t, h, "CS", 2023)

	challenge := askForCode(t, h, member.Email)
	code := h.Outbox.LastCodeTo(t, member.Email)
	for try := 1; try <= 5; try++ {
		expectStatus(t, fmt.Sprintf("wrong try %d", try), enterCode(t, h, challenge, wrongCode(code)), http.StatusUnauthorized)
	}
	expectStatus(t, "the right code after five wrong tries", enterCode(t, h, challenge, code), http.StatusUnauthorized)
}

func TestFourWrongTriesStillLeaveTheCodeUsable(t *testing.T) {
	h := apitest.New(t)
	member := studentOf(t, h, "CS", 2023)

	challenge := askForCode(t, h, member.Email)
	code := h.Outbox.LastCodeTo(t, member.Email)
	for try := 1; try <= 4; try++ {
		expectStatus(t, fmt.Sprintf("wrong try %d", try), enterCode(t, h, challenge, wrongCode(code)), http.StatusUnauthorized)
	}
	expectStatus(t, "the right code on the fifth try", enterCode(t, h, challenge, code), http.StatusOK)
}

func TestACodeWorksOnlyWithItsOwnChallenge(t *testing.T) {
	h := apitest.New(t)
	member := studentOf(t, h, "CS", 2023)

	first := askForCode(t, h, member.Email)
	firstCode := h.Outbox.LastCodeTo(t, member.Email)
	second := askForCode(t, h, member.Email)
	secondCode := h.Outbox.LastCodeTo(t, member.Email)
	if firstCode == secondCode {
		t.Skip("the two random codes happen to match")
	}

	expectStatus(t, "first code with the second challenge", enterCode(t, h, second, firstCode), http.StatusUnauthorized)
	expectStatus(t, "a made-up challenge", enterCode(t, h, "not-a-challenge", firstCode), http.StatusUnauthorized)
	expectStatus(t, "first code with its own challenge", enterCode(t, h, first, firstCode), http.StatusOK)
}

func TestAtMostThreeCodesPerEmailInFifteenMinutes(t *testing.T) {
	h := apitest.New(t)
	member := studentOf(t, h, "CS", 2023)

	for range 3 {
		askForCode(t, h, member.Email)
	}
	expectStatus(t, "a fourth code", h.Do(t, http.MethodPost, "/api/v1/auth/code", "", map[string]string{"email": member.Email}), http.StatusTooManyRequests)
	if codes := h.Outbox.CodesTo(member.Email); len(codes) != 3 {
		t.Errorf("sent %d codes, want 3", len(codes))
	}

	// An unknown email hits the same limit, so the limit can't tell them apart.
	for range 3 {
		askForCode(t, h, "nobody@apitest.local")
	}
	expectStatus(t, "a fourth code for an unknown email", h.Do(t, http.MethodPost, "/api/v1/auth/code", "", map[string]string{"email": "nobody@apitest.local"}), http.StatusTooManyRequests)

	// Fifteen minutes later the member can ask again.
	h.DB().Exec(`UPDATE sign_in_codes SET created_at = created_at - interval '16 minutes', expires_at = expires_at - interval '16 minutes'`)
	askForCode(t, h, member.Email)
}

// ageCodes moves every stored code request back in time, as if that much
// time had passed.
func ageCodes(h *apitest.Harness, interval string) {
	h.DB().Exec(`UPDATE sign_in_codes SET created_at = created_at - ?::interval, expires_at = expires_at - ?::interval`, interval, interval)
}

func TestAtMostTenCodesPerEmailInADay(t *testing.T) {
	h := apitest.New(t)
	member := studentOf(t, h, "CS", 2023)

	for i := 1; i <= 10; i++ {
		askForCode(t, h, member.Email)
		if i%3 == 0 {
			ageCodes(h, "16 minutes")
		}
	}
	ageCodes(h, "16 minutes")
	expectStatus(t, "an eleventh code in a day", h.Do(t, http.MethodPost, "/api/v1/auth/code", "", map[string]string{"email": member.Email}), http.StatusTooManyRequests)
	if codes := h.Outbox.CodesTo(member.Email); len(codes) != 10 {
		t.Errorf("sent %d codes, want 10", len(codes))
	}

	// A day later the member can ask again.
	ageCodes(h, "24 hours")
	askForCode(t, h, member.Email)
}

func TestTenWrongGuessesInADayStopNewCodes(t *testing.T) {
	h := apitest.New(t)
	member := studentOf(t, h, "CS", 2023)

	for round := 1; round <= 2; round++ {
		challenge := askForCode(t, h, member.Email)
		code := h.Outbox.LastCodeTo(t, member.Email)
		for try := 1; try <= 5; try++ {
			enterCode(t, h, challenge, wrongCode(code))
		}
		ageCodes(h, "16 minutes")
	}

	// The reply looks the same, so it doesn't say the email has an account,
	// but no code goes out.
	askForCode(t, h, member.Email)
	if codes := h.Outbox.CodesTo(member.Email); len(codes) != 2 {
		t.Errorf("sent %d codes after ten wrong guesses, want 2", len(codes))
	}

	// A day later codes go out again.
	ageCodes(h, "24 hours")
	askForCode(t, h, member.Email)
	if codes := h.Outbox.CodesTo(member.Email); len(codes) != 3 {
		t.Errorf("sent %d codes a day later, want 3", len(codes))
	}
}

func TestOneIPAddressCanAskForOnlySoManyCodes(t *testing.T) {
	h := apitest.New(t)
	limit := auth.DefaultCodeSettings().PerIPLimit
	for i := range limit {
		expectStatus(t, fmt.Sprintf("code %d", i+1), askForCodeFrom(t, h, "198.51.100.7", fmt.Sprintf("person%d@apitest.local", i)), http.StatusOK)
	}
	expectStatus(t, "one more from the same address", askForCodeFrom(t, h, "198.51.100.7", "another@apitest.local"), http.StatusTooManyRequests)
	expectStatus(t, "another address", askForCodeFrom(t, h, "198.51.100.8", "another@apitest.local"), http.StatusOK)

	var stored []string
	h.DB().Raw(`SELECT ip_hash FROM sign_in_codes`).Scan(&stored)
	for _, value := range stored {
		if strings.Contains(value, "198.51.100") {
			t.Fatalf("stored the IP address in clear: %q", value)
		}
	}
}

func TestPrincipalAndAdminGetNoEmailCode(t *testing.T) {
	h := apitest.New(t)
	for _, role := range []string{"principal", "admin"} {
		t.Run(role, func(t *testing.T) {
			leader := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: role}}})
			challenge := askForCode(t, h, leader.Email)
			if codes := h.Outbox.CodesTo(leader.Email); len(codes) != 0 {
				t.Errorf("sent a %s %d codes; they sign in with Google only", role, len(codes))
			}
			expectStatus(t, "a guess", enterCode(t, h, challenge, "123456"), http.StatusUnauthorized)
		})
	}
}

func TestASuspendedMemberCantSignInWithACode(t *testing.T) {
	h := apitest.New(t)
	suspended := studentOf(t, h, "CS", 2023)
	h.DB().Exec(`UPDATE users SET status = 'suspended' WHERE id = ?`, suspended.ID)
	askForCode(t, h, suspended.Email)
	if codes := h.Outbox.CodesTo(suspended.Email); len(codes) != 0 {
		t.Errorf("sent a suspended member %d codes", len(codes))
	}

	// Suspended after the code went out.
	member := studentOf(t, h, "CS", 2023)
	challenge := askForCode(t, h, member.Email)
	code := h.Outbox.LastCodeTo(t, member.Email)
	h.DB().Exec(`UPDATE users SET status = 'suspended' WHERE id = ?`, member.ID)
	expectStatus(t, "code entered after suspension", enterCode(t, h, challenge, code), http.StatusUnauthorized)
}

func TestAskingForACodeNeedsAnEmail(t *testing.T) {
	h := apitest.New(t)
	expectStatus(t, "not an email", h.Do(t, http.MethodPost, "/api/v1/auth/code", "", map[string]string{"email": "not-an-email"}), http.StatusBadRequest)
	expectStatus(t, "no code", h.Do(t, http.MethodPost, "/api/v1/auth/code/verify", "", map[string]string{"challenge_id": "x"}), http.StatusBadRequest)
}

func TestAnotherSiteCantUseTheEmailCode(t *testing.T) {
	h := apitest.New(t)
	member := studentOf(t, h, "CS", 2023)
	challenge := askForCode(t, h, member.Email)
	code := h.Outbox.LastCodeTo(t, member.Email)

	// A plain-text body skips the browser's CORS preflight, so only the
	// Origin check stops another site signing a visitor into this account.
	for path, body := range map[string]string{
		"/api/v1/auth/code":        fmt.Sprintf(`{"email":%q}`, member.Email),
		"/api/v1/auth/code/verify": fmt.Sprintf(`{"challenge_id":%q,"code":%q}`, challenge, code),
	} {
		request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		request.Header.Set("Content-Type", "text/plain")
		request.Header.Set("Origin", "https://evil.example")
		expectStatus(t, "cross-site "+path, h.Send(request), http.StatusForbidden)
	}
	expectStatus(t, "the code still works from LINKS", enterCode(t, h, challenge, code), http.StatusOK)
}
