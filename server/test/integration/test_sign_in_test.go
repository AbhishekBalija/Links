package integration

import (
	"net/http"
	"strings"
	"testing"

	"github.com/AbhishekBalija/Links/server/pkg/config"
	"github.com/AbhishekBalija/Links/server/test/apitest"
)

func TestTheTestSignInDoesNotExistUnlessSwitchedOn(t *testing.T) {
	h := apitest.New(t)
	member := studentOf(t, h, "CS", 2023)
	expectStatus(t, "test sign-in", h.Do(t, http.MethodPost, "/api/v1/test/sign-in", "", map[string]string{"email": member.Email}), http.StatusNotFound)
}

func TestTheTestSignInGivesAnActiveMemberANormalSession(t *testing.T) {
	h := apitest.NewWith(t, func(cfg *config.Config) { cfg.EnableTestSignIn = true })
	member := studentOf(t, h, "CS", 2023)

	response := h.Do(t, http.MethodPost, "/api/v1/test/sign-in", "", map[string]string{"email": member.Email})
	if response.Status != http.StatusOK {
		t.Fatalf("test sign-in status = %d: %s", response.Status, response.Body)
	}
	var session struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	response.Decode(t, &session)
	if session.Data.AccessToken == "" {
		t.Fatal("no access token")
	}
	if cookie := response.Header.Get("Set-Cookie"); cookie == "" {
		t.Error("no refresh cookie, so the app couldn't stay signed in")
	}
	expectStatus(t, "the token works", h.Do(t, http.MethodGet, "/api/v1/dashboard", session.Data.AccessToken, nil), http.StatusOK)

	expectStatus(t, "unknown email", h.Do(t, http.MethodPost, "/api/v1/test/sign-in", "", map[string]string{"email": "nobody@apitest.local"}), http.StatusUnauthorized)
	h.DB().Exec(`UPDATE users SET status = 'suspended' WHERE id = ?`, member.ID)
	expectStatus(t, "suspended member", h.Do(t, http.MethodPost, "/api/v1/test/sign-in", "", map[string]string{"email": member.Email}), http.StatusUnauthorized)
}

func TestTheE2ESuiteCanReadTheLastCodeSentToAnEmail(t *testing.T) {
	h := apitest.NewWith(t, func(cfg *config.Config) { cfg.EnableTestSignIn = true })
	member := studentOf(t, h, "CS", 2023)

	expectStatus(t, "no code yet", h.Do(t, http.MethodGet, "/api/v1/test/sign-in-code?email="+member.Email, "", nil), http.StatusNotFound)
	askForCode(t, h, member.Email)

	response := h.Do(t, http.MethodGet, "/api/v1/test/sign-in-code?email="+strings.ToUpper(member.Email), "", nil)
	expectStatus(t, "read the code", response, http.StatusOK)
	var reply struct {
		Data struct {
			Code string `json:"code"`
		} `json:"data"`
	}
	response.Decode(t, &reply)
	if want := h.Outbox.LastCodeTo(t, member.Email); reply.Data.Code != want {
		t.Errorf("code = %q, want the emailed %q", reply.Data.Code, want)
	}
}

func TestTheCodeReaderDoesNotExistUnlessSwitchedOn(t *testing.T) {
	h := apitest.New(t)
	member := studentOf(t, h, "CS", 2023)
	askForCode(t, h, member.Email)
	expectStatus(t, "read the code", h.Do(t, http.MethodGet, "/api/v1/test/sign-in-code?email="+member.Email, "", nil), http.StatusNotFound)
}
