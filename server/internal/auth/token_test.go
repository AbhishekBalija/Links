package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestValidateAccessTokenRejectsMalformedClaims(t *testing.T) {
	t.Parallel()

	cfg := TokenConfig{AccessSecret: "test-secret"}
	validClaims := jwt.MapClaims{
		"sub":   "user-1",
		"roles": []string{"student"},
		"iss":   accessTokenIssuer,
		"aud":   accessTokenAudience,
		"iat":   time.Now().Add(-time.Minute).Unix(),
		"exp":   time.Now().Add(time.Minute).Unix(),
		"jti":   "token-1",
	}

	tests := []struct {
		name   string
		mutate func(jwt.MapClaims)
	}{
		{name: "missing subject", mutate: func(c jwt.MapClaims) { delete(c, "sub") }},
		{name: "numeric subject", mutate: func(c jwt.MapClaims) { c["sub"] = 123 }},
		{name: "missing issued at", mutate: func(c jwt.MapClaims) { delete(c, "iat") }},
		{name: "missing expiration", mutate: func(c jwt.MapClaims) { delete(c, "exp") }},
		{name: "missing token id", mutate: func(c jwt.MapClaims) { delete(c, "jti") }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			claims := jwt.MapClaims{}
			for key, value := range validClaims {
				claims[key] = value
			}
			tc.mutate(claims)

			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			signed, err := token.SignedString([]byte(cfg.AccessSecret))
			if err != nil {
				t.Fatalf("sign token: %v", err)
			}
			if _, err := ValidateAccessToken(signed, cfg); err == nil {
				t.Fatal("expected malformed token to be rejected")
			}
		})
	}
}

func TestValidateAccessTokenRoundTrip(t *testing.T) {
	t.Parallel()

	cfg := TokenConfig{AccessSecret: "test-secret", AccessTTL: time.Minute}
	signed, err := GenerateAccessToken("user-1", []string{"student"}, cfg)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	claims, err := ValidateAccessToken(signed, cfg)
	if err != nil {
		t.Fatalf("validate token: %v", err)
	}
	if claims.UserID != "user-1" || len(claims.Roles) != 1 || claims.Roles[0] != "student" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}
