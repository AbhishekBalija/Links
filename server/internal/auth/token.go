package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	accessTokenIssuer   = "links-api"
	accessTokenAudience = "links-client"
)

// TokenConfig holds configuration for token generation.
type TokenConfig struct {
	AccessSecret  string
	RefreshSecret string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
}

// DefaultTokenConfig returns a TokenConfig with sensible defaults.
func DefaultTokenConfig() TokenConfig {
	return TokenConfig{
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 7 * 24 * time.Hour,
	}
}

// GenerateAccessToken creates a new signed JWT access token.
func GenerateAccessToken(userID string, roles []string, cfg TokenConfig) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub": userID,
		"roles": roles,
		"iss": accessTokenIssuer,
		"aud": accessTokenAudience,
		"iat": now.Unix(),
		"exp": now.Add(cfg.AccessTTL).Unix(),
		"jti": uuid.New().String(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(cfg.AccessSecret))
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}
	return signed, nil
}

// ValidateAccessToken validates and parses a JWT access token.
// Returns the claims if valid, or an error if invalid/expired.
func ValidateAccessToken(tokenString string, cfg TokenConfig) (*TokenClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(cfg.AccessSecret), nil
	}, jwt.WithIssuer(accessTokenIssuer), jwt.WithAudience(accessTokenAudience))

	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid claims")
	}

	userID := claims["sub"].(string)

	var roles []string
	if rolesRaw, ok := claims["roles"].([]interface{}); ok {
		for _, r := range rolesRaw {
			if s, ok := r.(string); ok {
				roles = append(roles, s)
			}
		}
	}

	iss, _ := claims["iss"].(string)
	aud, _ := claims["aud"].(string)
	iat := int64(claims["iat"].(float64))
	exp := int64(claims["exp"].(float64))
	jti, _ := claims["jti"].(string)

	return &TokenClaims{
		UserID: userID,
		Roles:  roles,
		Issuer: iss,
		Aud:    aud,
		IAT:    iat,
		Exp:    exp,
		JTI:    jti,
	}, nil
}

// GenerateRefreshTokenRaw creates a new cryptographically random refresh token
// and returns both the raw token and its SHA-256 hash.
// Per docs/auth.md: refresh tokens are high-entropy, so SHA-256 is sufficient
// and avoids DoS risk of slow hashes on verification endpoints.
func GenerateRefreshTokenRaw(cfg TokenConfig) (*RefreshTokenRaw, error) {
	rawBytes := make([]byte, 32)
	if _, err := rand.Read(rawBytes); err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}
	raw := base64.RawURLEncoding.EncodeToString(rawBytes)

	hash := sha256.Sum256([]byte(raw))
	hashHex := hex.EncodeToString(hash[:])

	expiresAt := time.Now().Add(cfg.RefreshTTL)

	return &RefreshTokenRaw{
		Token:     raw,
		TokenHash: hashHex,
		ExpiresAt: expiresAt,
	}, nil
}

// HashRefreshToken returns the SHA-256 hash of a raw refresh token.
func HashRefreshToken(raw string) string {
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:])
}