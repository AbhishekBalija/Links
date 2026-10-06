package integration

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/AbhishekBalija/Links/server/internal/auth"
	"github.com/AbhishekBalija/Links/server/test/apitest"
)

// Someone else asking for codes, or guessing wrong, for a member's email
// must not stop the member getting codes from their own phone (#178).

const (
	strangerIP = "203.0.113.9"
	ownerIP    = "198.51.100.20"
)

func challengeFrom(t *testing.T, response apitest.Response) string {
	t.Helper()
	expectStatus(t, "ask for a code", response, http.StatusOK)
	var reply codeReply
	response.Decode(t, &reply)
	return reply.Data.ChallengeID
}

func TestWrongGuessesFromElsewhereDontStopTheOwnersCodes(t *testing.T) {
	h := apitest.New(t)
	member := studentOf(t, h, "CS", 2023)

	// A stranger asks for codes for the member and guesses wrong ten times.
	for round := 1; round <= 2; round++ {
		challenge := challengeFrom(t, askForCodeFrom(t, h, strangerIP, member.Email))
		code := h.Outbox.LastCodeTo(t, member.Email)
		for try := 1; try <= 5; try++ {
			enterCode(t, h, challenge, wrongCode(code))
		}
		ageCodes(h, "16 minutes")
	}

	// The stranger gets no more codes, but the member still does.
	challengeFrom(t, askForCodeFrom(t, h, strangerIP, member.Email))
	if codes := h.Outbox.CodesTo(member.Email); len(codes) != 2 {
		t.Fatalf("sent %d codes after the stranger's ten wrong guesses, want 2", len(codes))
	}
	challenge := challengeFrom(t, askForCodeFrom(t, h, ownerIP, member.Email))
	if codes := h.Outbox.CodesTo(member.Email); len(codes) != 3 {
		t.Fatalf("sent %d codes to the member's own request, want 3", len(codes))
	}
	expectStatus(t, "the member signs in", enterCode(t, h, challenge, h.Outbox.LastCodeTo(t, member.Email)), http.StatusOK)
}

func TestAStrangerAskingForCodesDoesntUseUpTheOwners(t *testing.T) {
	h := apitest.New(t)
	member := studentOf(t, h, "CS", 2023)

	for range 3 {
		challengeFrom(t, askForCodeFrom(t, h, strangerIP, member.Email))
	}
	expectStatus(t, "the stranger's fourth", askForCodeFrom(t, h, strangerIP, member.Email), http.StatusTooManyRequests)
	challengeFrom(t, askForCodeFrom(t, h, ownerIP, member.Email))

	// Even after the stranger's whole day's worth.
	for i := 4; i <= 10; i++ {
		if i%3 == 1 {
			ageCodes(h, "16 minutes")
		}
		challengeFrom(t, askForCodeFrom(t, h, strangerIP, member.Email))
	}
	ageCodes(h, "16 minutes")
	expectStatus(t, "the stranger's eleventh today", askForCodeFrom(t, h, strangerIP, member.Email), http.StatusTooManyRequests)
	challengeFrom(t, askForCodeFrom(t, h, ownerIP, member.Email))
}

func TestAnEmailGetsAtMostThirtyCodesADayFromAnywhere(t *testing.T) {
	h := apitest.New(t)
	member := studentOf(t, h, "CS", 2023)

	// Ten addresses, three codes each: the inbox's ceiling for the day.
	for address := 1; address <= 10; address++ {
		for range 3 {
			challengeFrom(t, askForCodeFrom(t, h, fmt.Sprintf("192.0.2.%d", address), member.Email))
		}
	}
	expectStatus(t, "a thirty-first from yet another address", askForCodeFrom(t, h, "192.0.2.99", member.Email), http.StatusTooManyRequests)
	if codes := h.Outbox.CodesTo(member.Email); len(codes) != 30 {
		t.Errorf("sent %d codes, want 30", len(codes))
	}
}

// The "too many codes" reply says which limit was hit, so the screen can
// tell one person asking too often from a whole campus on one network
// (#202). It never says whether the email has an account.
func TestTooManyCodesSaysWhichLimit(t *testing.T) {
	h := apitest.New(t)
	limitOf := func(response apitest.Response) string {
		t.Helper()
		expectStatus(t, "refused", response, http.StatusTooManyRequests)
		var body struct {
			Error struct {
				Details struct {
					Limit string `json:"limit"`
				} `json:"details"`
			} `json:"error"`
		}
		response.Decode(t, &body)
		return body.Error.Details.Limit
	}

	for range 3 {
		challengeFrom(t, askForCodeFrom(t, h, ownerIP, "someone@apitest.local"))
	}
	if limit := limitOf(askForCodeFrom(t, h, ownerIP, "someone@apitest.local")); limit != "email" {
		t.Errorf("a fourth for one email: limit = %q, want email", limit)
	}

	for i := 0; i < auth.DefaultCodeSettings().PerIPLimit; i++ {
		askForCodeFrom(t, h, campusIP, fmt.Sprintf("student%d@apitest.local", i))
	}
	if limit := limitOf(askForCodeFrom(t, h, campusIP, "another@apitest.local")); limit != "network" {
		t.Errorf("a whole network's worth: limit = %q, want network", limit)
	}
}

// A class signing in together on day one shares the campus address and has
// no known browsers yet; they must all get codes.
func TestAClassOnOneCampusNetworkAllGetCodes(t *testing.T) {
	h := apitest.New(t)
	for i := 0; i < 200; i++ {
		challengeFrom(t, askForCodeFrom(t, h, campusIP, fmt.Sprintf("student%d@apitest.local", i)))
	}
}
