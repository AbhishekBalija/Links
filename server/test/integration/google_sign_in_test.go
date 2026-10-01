package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/AbhishekBalija/Links/server/pkg/config"
	"github.com/AbhishekBalija/Links/server/test/apitest"
)

// googleSignIn posts a Google credential the way the screen does, with the
// nonce cookie the browser holds (nil for none).
func googleSignIn(t *testing.T, h *apitest.Harness, credential string, nonceCookie *http.Cookie) apitest.Response {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/google", strings.NewReader(fmt.Sprintf(`{"credential":%q}`, credential)))
	request.Header.Set("Content-Type", "application/json")
	if nonceCookie != nil {
		request.AddCookie(nonceCookie)
	}
	return h.Send(request)
}

// signInWithGoogle signs in as the Google account and returns the user ID
// the session belongs to.
func signInWithGoogle(t *testing.T, h *apitest.Harness, subject, email string) string {
	t.Helper()
	nonce, cookie := h.GoogleNonce(t)
	response := googleSignIn(t, h, h.GoogleToken(t, apitest.GoogleClaims{Subject: subject, Email: email, Nonce: nonce}), cookie)
	expectStatus(t, "Google sign-in", response, http.StatusOK)
	var session struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	response.Decode(t, &session)
	me := h.Do(t, http.MethodGet, "/api/v1/me", session.Data.AccessToken, nil)
	expectStatus(t, "me", me, http.StatusOK)
	var who struct {
		Data struct {
			UserID string `json:"user_id"`
		} `json:"data"`
	}
	me.Decode(t, &who)
	return who.Data.UserID
}

func TestAValidGoogleTokenSignsAMemberIn(t *testing.T) {
	h := apitest.New(t)
	member := studentOf(t, h, "CS", 2023)

	nonce, cookie := h.GoogleNonce(t)
	if !cookie.HttpOnly || len(nonce) < 20 {
		t.Fatalf("nonce cookie httpOnly = %v, nonce %q; want an httpOnly cookie and a long random nonce", cookie.HttpOnly, nonce)
	}
	response := googleSignIn(t, h, h.GoogleToken(t, apitest.GoogleClaims{Subject: "google-1", Email: member.Email, Nonce: nonce}), cookie)
	expectStatus(t, "Google sign-in", response, http.StatusOK)

	var gotRefresh, clearedNonce bool
	for _, set := range response.Cookies() {
		gotRefresh = gotRefresh || (set.Name == "refresh_token" && set.Value != "")
		clearedNonce = clearedNonce || (set.Name == "google_nonce" && set.MaxAge < 0)
	}
	if !gotRefresh {
		t.Error("no refresh cookie, so the app couldn't stay signed in")
	}
	if !clearedNonce {
		t.Error("the nonce cookie wasn't cleared after use")
	}

	var methods []string
	h.DB().Raw(`SELECT metadata->>'method' FROM audit_logs WHERE action = 'auth.signed_in' AND actor_id = ?`, member.ID).Scan(&methods)
	if len(methods) != 1 || methods[0] != "google" {
		t.Errorf("sign-in audit methods = %v, want one google", methods)
	}
}

func TestGoogleTokensFailingACheckAreRefused(t *testing.T) {
	h := apitest.New(t)
	member := studentOf(t, h, "CS", 2023)
	unverified := false

	cases := map[string]func(nonce string) (apitest.GoogleClaims, bool){
		"wrong audience": func(nonce string) (apitest.GoogleClaims, bool) {
			return apitest.GoogleClaims{Email: member.Email, Nonce: nonce, Audience: "someone-else.apps.googleusercontent.com"}, true
		},
		"wrong issuer": func(nonce string) (apitest.GoogleClaims, bool) {
			return apitest.GoogleClaims{Email: member.Email, Nonce: nonce, Issuer: "https://evil.example"}, true
		},
		"expired": func(nonce string) (apitest.GoogleClaims, bool) {
			return apitest.GoogleClaims{Email: member.Email, Nonce: nonce, ExpiresAt: time.Now().Add(-time.Minute)}, true
		},
		"unverified email": func(nonce string) (apitest.GoogleClaims, bool) {
			return apitest.GoogleClaims{Email: member.Email, Nonce: nonce, EmailVerified: &unverified}, true
		},
		"another browser's nonce": func(string) (apitest.GoogleClaims, bool) {
			return apitest.GoogleClaims{Email: member.Email, Nonce: "nonce-from-the-attackers-browser"}, true
		},
		"no nonce cookie": func(nonce string) (apitest.GoogleClaims, bool) {
			return apitest.GoogleClaims{Email: member.Email, Nonce: nonce}, false
		},
		"no nonce in the token": func(string) (apitest.GoogleClaims, bool) {
			return apitest.GoogleClaims{Email: member.Email}, true
		},
	}
	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			nonce, cookie := h.GoogleNonce(t)
			claims, sendCookie := build(nonce)
			claims.Subject = "google-1"
			if !sendCookie {
				cookie = nil
			}
			expectStatus(t, name, googleSignIn(t, h, h.GoogleToken(t, claims), cookie), http.StatusUnauthorized)
		})
	}

	t.Run("not a token", func(t *testing.T) {
		_, cookie := h.GoogleNonce(t)
		expectStatus(t, "garbage", googleSignIn(t, h, "not.a.token", cookie), http.StatusUnauthorized)
	})

	// None of the refusals linked the Google account.
	signInWithGoogle(t, h, "google-1", member.Email)
}

func TestGoogleMatchesByEmailOnceThenByGoogleAccount(t *testing.T) {
	h := apitest.New(t)
	member := studentOf(t, h, "CS", 2023)

	if got := signInWithGoogle(t, h, "google-1", strings.ToUpper(member.Email)); got != member.ID {
		t.Fatalf("first sign-in by email gave user %s, want %s", got, member.ID)
	}
	// The Google account's email changes; the account still finds its member.
	if got := signInWithGoogle(t, h, "google-1", "new-address@gmail.example"); got != member.ID {
		t.Fatalf("sign-in by Google account gave user %s, want %s", got, member.ID)
	}
}

func TestAnotherGoogleAccountCantTakeALinkedEmail(t *testing.T) {
	h := apitest.New(t)
	member := studentOf(t, h, "CS", 2023)
	signInWithGoogle(t, h, "google-1", member.Email)

	nonce, cookie := h.GoogleNonce(t)
	response := googleSignIn(t, h, h.GoogleToken(t, apitest.GoogleClaims{Subject: "google-2", Email: member.Email, Nonce: nonce}), cookie)
	expectStatus(t, "a second Google account with the same email", response, http.StatusUnauthorized)
}

func TestAnEmailOnNoListGetsNotOnList(t *testing.T) {
	h := apitest.New(t)
	var before int64
	h.DB().Raw(`SELECT count(*) FROM users`).Scan(&before)

	nonce, cookie := h.GoogleNonce(t)
	response := googleSignIn(t, h, h.GoogleToken(t, apitest.GoogleClaims{Subject: "google-9", Email: "Asha.Rao@gmail.example", Name: "Asha Rao", Nonce: nonce}), cookie)
	expectStatus(t, "unknown email", response, http.StatusForbidden)
	var reply struct {
		Error struct {
			Code    string `json:"code"`
			Details struct {
				Email    string `json:"email"`
				FullName string `json:"full_name"`
			} `json:"details"`
		} `json:"error"`
	}
	response.Decode(t, &reply)
	if reply.Error.Code != "NOT_ON_LIST" || reply.Error.Details.Email != "asha.rao@gmail.example" || reply.Error.Details.FullName != "Asha Rao" {
		t.Errorf("reply = %s, want NOT_ON_LIST with the verified email and name", response.Body)
	}

	var after int64
	h.DB().Raw(`SELECT count(*) FROM users`).Scan(&after)
	if after != before {
		t.Errorf("users went from %d to %d; an unknown email must not create an account", before, after)
	}
}

func TestPrincipalAndAdminSignInWithGoogle(t *testing.T) {
	h := apitest.New(t)
	for i, role := range []string{"principal", "admin"} {
		leader := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: role}}})
		if got := signInWithGoogle(t, h, fmt.Sprintf("google-%d", i), leader.Email); got != leader.ID {
			t.Errorf("%s signed in as %s, want %s", role, got, leader.ID)
		}
	}
}

func TestAnAccountThatIsNotActiveCantSignInWithGoogle(t *testing.T) {
	h := apitest.New(t)
	for _, status := range []string{"suspended", "pending", "rejected"} {
		t.Run(status, func(t *testing.T) {
			member := studentOf(t, h, "CS", 2023)
			h.DB().Exec(`UPDATE users SET status = ? WHERE id = ?`, status, member.ID)
			nonce, cookie := h.GoogleNonce(t)
			response := googleSignIn(t, h, h.GoogleToken(t, apitest.GoogleClaims{Subject: "google-" + status, Email: member.Email, Nonce: nonce}), cookie)
			expectStatus(t, status, response, http.StatusForbidden)
			if !strings.Contains(response.Body, `"ACCOUNT_NOT_ACTIVE"`) || !strings.Contains(response.Body, `"`+status+`"`) {
				t.Errorf("reply = %s, want ACCOUNT_NOT_ACTIVE with the status", response.Body)
			}
		})
	}
}

func TestAnotherSiteCantPostAGoogleCredential(t *testing.T) {
	h := apitest.New(t)
	member := studentOf(t, h, "CS", 2023)
	nonce, cookie := h.GoogleNonce(t)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/google",
		strings.NewReader(fmt.Sprintf(`{"credential":%q}`, h.GoogleToken(t, apitest.GoogleClaims{Subject: "google-1", Email: member.Email, Nonce: nonce}))))
	request.Header.Set("Content-Type", "text/plain")
	request.Header.Set("Origin", "https://evil.example")
	request.AddCookie(cookie)
	expectStatus(t, "cross-site Google sign-in", h.Send(request), http.StatusForbidden)
}

func TestGoogleSignInIsOffWithoutAClientID(t *testing.T) {
	h := apitest.NewWith(t, func(cfg *config.Config) { cfg.Google.ClientID = "" })
	expectStatus(t, "nonce", h.Do(t, http.MethodGet, "/api/v1/auth/google/nonce", "", nil), http.StatusNotFound)
	expectStatus(t, "sign-in", h.Do(t, http.MethodPost, "/api/v1/auth/google", "", map[string]string{"credential": "x"}), http.StatusNotFound)
}
