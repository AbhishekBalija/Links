package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func testRouter(cfg TokenConfig, optional bool) *gin.Engine {
	engine := gin.New()
	middleware := RequireAuth(cfg)
	if optional {
		middleware = OptionalAuth(cfg)
	}
	engine.Use(middleware)
	engine.GET("/", func(c *gin.Context) {
		actor := GetActor(c)
		if actor == nil {
			c.String(http.StatusOK, "anonymous")
			return
		}
		c.String(http.StatusOK, actor.UserID+":"+strings.Join(actor.Roles, ","))
	})
	return engine
}

func doRequest(engine http.Handler, authHeader string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec
}

func TestRequireAuth_RejectsMissingOrMalformedTokens(t *testing.T) {
	cfg := TokenConfig{AccessSecret: "secret", AccessTTL: time.Minute}
	tests := []struct {
		name   string
		header string
	}{
		{"missing header", ""},
		{"non bearer scheme", "Basic dXNlcjpwYXNz"},
		{"empty token", "Bearer "},
		{"garbage token", "Bearer not-a-jwt"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			engine := testRouter(cfg, false)
			rec := doRequest(engine, tc.header)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
			}
			if body := rec.Body.String(); !strings.Contains(body, "UNAUTHENTICATED") {
				t.Fatalf("expected UNAUTHENTICATED error body, got %q", body)
			}
		})
	}
}

func TestRequireAuth_RejectsExpiredAndForeignSignedTokens(t *testing.T) {
	cfg := TokenConfig{AccessSecret: "correct-secret", AccessTTL: time.Minute}

	expired, err := GenerateAccessToken("user-1", nil, TokenConfig{AccessSecret: "correct-secret", AccessTTL: -time.Minute})
	if err != nil {
		t.Fatalf("generate expired token: %v", err)
	}
	foreign, err := GenerateAccessToken("user-1", nil, TokenConfig{AccessSecret: "other-secret", AccessTTL: time.Minute})
	if err != nil {
		t.Fatalf("generate foreign token: %v", err)
	}

	for _, tc := range []struct {
		name string
		tok  string
	}{
		{"expired token", expired},
		{"token signed with wrong secret", foreign},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := doRequest(testRouter(cfg, false), "Bearer "+tc.tok)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
			}
		})
	}
}

func TestRequireAuth_SetsActorFromValidToken(t *testing.T) {
	cfg := TokenConfig{AccessSecret: "correct-secret", AccessTTL: time.Minute}
	valid, err := GenerateAccessToken("user-42", []string{"student", "hod"}, cfg)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	rec := doRequest(testRouter(cfg, false), "Bearer "+valid)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if rec.Body.String() != "user-42:student,hod" {
		t.Fatalf("handler saw unexpected actor: %q", rec.Body.String())
	}
}

func TestOptionalAuth_PassesAnonymousAndInvalidRequestsThrough(t *testing.T) {
	cfg := TokenConfig{AccessSecret: "correct-secret", AccessTTL: time.Minute}

	for _, tc := range []struct {
		name   string
		header string
	}{
		{"no token", ""},
		{"invalid token", "Bearer garbage"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := doRequest(testRouter(cfg, true), tc.header)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
			}
			if rec.Body.String() != "anonymous" {
				t.Fatalf("expected anonymous actor, got %q", rec.Body.String())
			}
		})
	}
}

func TestOptionalAuth_SetsActorWhenTokenPresent(t *testing.T) {
	cfg := TokenConfig{AccessSecret: "correct-secret", AccessTTL: time.Minute}
	valid, err := GenerateAccessToken("user-7", []string{"placement_officer"}, cfg)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	rec := doRequest(testRouter(cfg, true), "Bearer "+valid)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "user-7:placement_officer" {
		t.Fatalf("handler saw unexpected actor: %q", rec.Body.String())
	}
}
