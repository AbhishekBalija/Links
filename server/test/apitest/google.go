package apitest

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"testing"
	"time"
)

// GoogleClientID is the OAuth client ID the test API expects in a token's aud.
const GoogleClientID = "apitest-client.apps.googleusercontent.com"

// googleSigner stands in for Google: it signs ID tokens with its own key and
// serves that key where the real verifier fetches Google's certificates, so
// tests exercise the real verification without the network.
type googleSigner struct {
	key *rsa.PrivateKey
	kid string
}

func newGoogleSigner(t *testing.T) *googleSigner {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate Google test key: %v", err)
	}
	return &googleSigner{key: key, kid: "apitest-" + randomHex(t, 4)}
}

// certsClient answers every request with the signer's public key set.
func (s *googleSigner) certsClient() *http.Client {
	return &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		body, _ := json.Marshal(map[string]any{"keys": []map[string]string{{
			"kty": "RSA",
			"alg": "RS256",
			"use": "sig",
			"kid": s.kid,
			"n":   base64.RawURLEncoding.EncodeToString(s.key.N.Bytes()),
			"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(s.key.E)).Bytes()),
		}}})
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(body)),
			Request:    request,
		}, nil
	})}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

// GoogleClaims describes an ID token. Empty fields get a valid default:
// this client's audience, Google's issuer, an hour to live and a verified
// email.
type GoogleClaims struct {
	Subject       string
	Email         string
	Name          string
	Nonce         string
	Audience      string
	Issuer        string
	ExpiresAt     time.Time
	EmailVerified *bool
}

// GoogleToken signs an ID token the way Google would.
func (h *Harness) GoogleToken(t *testing.T, claims GoogleClaims) string {
	t.Helper()
	if claims.Audience == "" {
		claims.Audience = GoogleClientID
	}
	if claims.Issuer == "" {
		claims.Issuer = "https://accounts.google.com"
	}
	if claims.ExpiresAt.IsZero() {
		claims.ExpiresAt = time.Now().Add(time.Hour)
	}
	verified := true
	if claims.EmailVerified != nil {
		verified = *claims.EmailVerified
	}
	header, _ := json.Marshal(map[string]string{"alg": "RS256", "typ": "JWT", "kid": h.google.kid})
	payload, _ := json.Marshal(map[string]any{
		"iss":            claims.Issuer,
		"aud":            claims.Audience,
		"sub":            claims.Subject,
		"email":          claims.Email,
		"email_verified": verified,
		"name":           claims.Name,
		"nonce":          claims.Nonce,
		"iat":            time.Now().Add(-time.Minute).Unix(),
		"exp":            claims.ExpiresAt.Unix(),
	})
	signingInput := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	digest := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, h.google.key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatalf("sign Google test token: %v", err)
	}
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature)
}

// GoogleNonce asks the API for a sign-in nonce, the way the screen does
// before showing the Google button, and returns it with the cookie it set.
func (h *Harness) GoogleNonce(t *testing.T) (string, *http.Cookie) {
	t.Helper()
	response := h.Do(t, http.MethodGet, "/api/v1/auth/google/nonce", "", nil)
	if response.Status != http.StatusOK {
		t.Fatalf("google nonce status = %d: %s", response.Status, response.Body)
	}
	var reply struct {
		Data struct {
			Nonce string `json:"nonce"`
		} `json:"data"`
	}
	response.Decode(t, &reply)
	for _, cookie := range response.Cookies() {
		if cookie.Name == "google_nonce" {
			return reply.Data.Nonce, cookie
		}
	}
	t.Fatal("the nonce endpoint set no google_nonce cookie")
	return "", nil
}
