package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

func signInWithGoogle(t *testing.T, h *apitest.Harness, email string) apitest.Response {
	t.Helper()
	nonce, cookie := h.GoogleNonce(t)
	credential := h.GoogleToken(t, apitest.GoogleClaims{Subject: "google-" + email, Email: email, Nonce: nonce})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/google", strings.NewReader(fmt.Sprintf(`{"credential":%q}`, credential)))
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(cookie)
	return h.Send(request)
}

func TestTheFirstAdminSignsInWithGoogleAndIsAnAdmin(t *testing.T) {
	h := apitest.New(t)
	if err := addAdmin(context.Background(), h.DB(), "principal.office@college.edu", "Nikhil Bhat"); err != nil {
		t.Fatalf("add admin: %v", err)
	}

	response := signInWithGoogle(t, h, "principal.office@college.edu")
	if response.Status != http.StatusOK {
		t.Fatalf("Google sign-in status = %d: %s", response.Status, response.Body)
	}
	var reply struct {
		Data struct {
			FirstSignIn struct {
				Roles []string `json:"roles"`
			} `json:"first_sign_in"`
		} `json:"data"`
	}
	response.Decode(t, &reply)
	if roles := reply.Data.FirstSignIn.Roles; len(roles) != 1 || roles[0] != "admin" {
		t.Errorf("roles = %v, want [admin]", roles)
	}
}

func TestTheFirstAdminCanBeAddedOnlyOnce(t *testing.T) {
	h := apitest.New(t)
	ctx := context.Background()
	if err := addAdmin(ctx, h.DB(), "first@college.edu", "First Admin"); err != nil {
		t.Fatalf("add admin: %v", err)
	}
	if err := addAdmin(ctx, h.DB(), "second@college.edu", "Second Admin"); err == nil {
		t.Error("a second admin was added from the command line; later admins come from inside LINKS")
	}
}

func TestAddingTheFirstAdminChecksTheDetails(t *testing.T) {
	h := apitest.New(t)
	ctx := context.Background()
	if err := addAdmin(ctx, h.DB(), "not-an-email", "Someone"); err == nil {
		t.Error("accepted an address that isn't an email")
	}
	if err := addAdmin(ctx, h.DB(), "someone@college.edu", "  "); err == nil {
		t.Error("accepted an empty name")
	}
}
