package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"golang.org/x/crypto/argon2"
)

// Argon2id parameters — lighter defaults suitable for lower-resource hosting.
// docs/auth.md: "Prefer Argon2id for new password hashes."
// docs/security.md: "Prefer Argon2id for password hashing."
const (
	argon2Memory     = 32 * 1024 // 32 MB
	argon2Iterations = 1
	argon2Threads    = 2
	argon2SaltLength = 16
	argon2KeyLength  = 32
)

// Password validation rules.
// docs/auth.md: "Enforce minimum password strength."
const minPasswordLength = 8

// ErrPasswordTooWeak is returned when a password does not meet the minimum
// strength requirements: at least 8 characters, one uppercase letter, one
// lowercase letter, and one digit.
var ErrPasswordTooWeak = errors.New("password must be at least 8 characters and contain an uppercase letter, a lowercase letter, and a number")

// HashPassword hashes a plaintext password using Argon2id.
// It returns a PHC-format string that embeds the algorithm, version,
// parameters, salt, and hash so the result is self-describing.
//
// docs/auth.md: "Never store plaintext passwords."
func HashPassword(password string) (string, error) {
	if err := ValidatePasswordStrength(password); err != nil {
		return "", err
	}

	salt := make([]byte, argon2SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		argon2Iterations,
		argon2Memory,
		argon2Threads,
		argon2KeyLength,
	)

	// PHC format: $argon2id$v=19$m=32768,t=1,p=2$<salt>$<hash>
	encoded := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		argon2Memory,
		argon2Iterations,
		argon2Threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)
	return encoded, nil
}

// VerifyPassword checks a plaintext password against a PHC-format Argon2id
// hash. Returns true when the password matches.
func VerifyPassword(password, encodedHash string) (bool, error) {
	salt, hash, memory, iterations, threads, keyLen, err := decodeHash(encodedHash)
	if err != nil {
		return false, err
	}

	candidate := argon2.IDKey(
		[]byte(password),
		salt,
		iterations,
		memory,
		threads,
		keyLen,
	)

	// Constant-time comparison prevents timing attacks.
	return subtle.ConstantTimeCompare(hash, candidate) == 1, nil
}

// ValidatePasswordStrength enforces the minimum password rules:
// at least 8 characters, one uppercase, one lowercase, one digit.
//
// docs/auth.md: "Enforce minimum password strength."
// docs/auth.md: "Never log passwords."
func ValidatePasswordStrength(password string) error {
	if len(password) < minPasswordLength {
		return ErrPasswordTooWeak
	}

	var hasUpper, hasLower, hasDigit bool
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}
	if !hasUpper || !hasLower || !hasDigit {
		return ErrPasswordTooWeak
	}
	return nil
}

// Argon2PasswordHasher implements PasswordHasher using Argon2id.
type Argon2PasswordHasher struct{}

func NewArgon2PasswordHasher() *Argon2PasswordHasher {
	return &Argon2PasswordHasher{}
}

func (h *Argon2PasswordHasher) Hash(password string) (string, error) {
	return HashPassword(password)
}

func (h *Argon2PasswordHasher) Verify(password, hash string) (bool, error) {
	return VerifyPassword(password, hash)
}

// decodeHash parses a PHC-format Argon2id hash string.
func decodeHash(encoded string) (salt, hash []byte, memory, iterations uint32, threads uint8, keyLen uint32, err error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 {
		return nil, nil, 0, 0, 0, 0, errors.New("invalid hash format")
	}
	if parts[1] != "argon2id" {
		return nil, nil, 0, 0, 0, 0, fmt.Errorf("unsupported algorithm: %s", parts[1])
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return nil, nil, 0, 0, 0, 0, fmt.Errorf("parse version: %w", err)
	}
	if version != argon2.Version {
		return nil, nil, 0, 0, 0, 0, fmt.Errorf("unsupported argon2 version: %d", version)
	}

	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &threads); err != nil {
		return nil, nil, 0, 0, 0, 0, fmt.Errorf("parse parameters: %w", err)
	}

	salt, err = base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, nil, 0, 0, 0, 0, fmt.Errorf("decode salt: %w", err)
	}

	hash, err = base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return nil, nil, 0, 0, 0, 0, fmt.Errorf("decode hash: %w", err)
	}
	keyLen = uint32(len(hash))

	return salt, hash, memory, iterations, threads, keyLen, nil
}
