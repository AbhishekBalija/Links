package auth

// A request token proves someone owns an email that is on no list, so they
// can send an Access request without a password (spec #129). It is handed
// out with NOT_ON_LIST after a Google sign-in or an email code, and is good
// for one Access request within its lifetime.

import (
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	requestTokenAudience = "links-access-request"
	// RequestTokenTTL is how long someone has to send their Access request.
	RequestTokenTTL = 30 * time.Minute
)

var errBadRequestToken = errors.New("invalid request token")

// requestTokenKey derives the token's own key from the server secret, so it
// can never pass as an access token or the other way round.
func requestTokenKey(cfg TokenConfig) []byte {
	mac := hmac.New(sha256.New, []byte(cfg.RefreshSecret))
	mac.Write([]byte("access-request-token"))
	return mac.Sum(nil)
}

// SignAccessRequestToken proves the email until expiresAt.
func SignAccessRequestToken(cfg TokenConfig, email string, expiresAt time.Time) (string, error) {
	claims := jwt.MapClaims{
		"sub": email,
		"iss": accessTokenIssuer,
		"aud": requestTokenAudience,
		"iat": time.Now().Unix(),
		"exp": expiresAt.Unix(),
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(requestTokenKey(cfg))
	if err != nil {
		return "", fmt.Errorf("sign request token: %w", err)
	}
	return signed, nil
}

// parseAccessRequestToken returns the proven email.
func parseAccessRequestToken(cfg TokenConfig, raw string) (string, error) {
	token, err := jwt.Parse(raw, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return requestTokenKey(cfg), nil
	},
		jwt.WithIssuer(accessTokenIssuer),
		jwt.WithAudience(requestTokenAudience),
		jwt.WithExpirationRequired(),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil || !token.Valid {
		return "", errBadRequestToken
	}
	email, err := token.Claims.GetSubject()
	if err != nil || email == "" {
		return "", errBadRequestToken
	}
	return email, nil
}
