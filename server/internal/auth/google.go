package auth

// Google sign-in: the browser gets an ID token from Google Identity Services
// and the API verifies it, then finds the member by Google's permanent account
// ID, or by verified email on their first Google sign-in (spec #129, ADR 0026).

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"google.golang.org/api/idtoken"

	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
)

// GoogleIdentity is what a verified Google ID token says about who signed in.
type GoogleIdentity struct {
	Subject string
	Email   string
	Name    string
	Nonce   string
}

// GoogleVerifier checks a Google ID token and returns who it names.
type GoogleVerifier interface {
	Verify(ctx context.Context, credential string) (*GoogleIdentity, error)
}

var errGoogleTokenRejected = errors.New("google token rejected")

// googleIssuers are the two spellings Google uses for its ID tokens' iss.
var googleIssuers = map[string]bool{
	"accounts.google.com":         true,
	"https://accounts.google.com": true,
}

// IDTokenVerifier verifies Google ID tokens with Google's official library:
// the signature against Google's published keys, the audience and expiry, and
// then the issuer and a verified email.
type IDTokenVerifier struct {
	validator *idtoken.Validator
	clientID  string
}

// NewIDTokenVerifier builds a verifier for tokens issued to clientID. Pass
// idtoken options only to change where Google's keys are fetched from.
func NewIDTokenVerifier(ctx context.Context, clientID string, opts ...idtoken.ClientOption) (*IDTokenVerifier, error) {
	validator, err := idtoken.NewValidator(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create Google token validator: %w", err)
	}
	return &IDTokenVerifier{validator: validator, clientID: clientID}, nil
}

func (v *IDTokenVerifier) Verify(ctx context.Context, credential string) (*GoogleIdentity, error) {
	// An empty audience would skip the audience check.
	if v.clientID == "" {
		return nil, fmt.Errorf("%w: no client ID configured", errGoogleTokenRejected)
	}
	payload, err := v.validator.Validate(ctx, credential, v.clientID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errGoogleTokenRejected, err)
	}
	if !googleIssuers[payload.Issuer] {
		return nil, fmt.Errorf("%w: issuer %q", errGoogleTokenRejected, payload.Issuer)
	}
	if verified, _ := payload.Claims["email_verified"].(bool); !verified {
		return nil, fmt.Errorf("%w: email not verified", errGoogleTokenRejected)
	}
	email, _ := payload.Claims["email"].(string)
	if payload.Subject == "" || email == "" {
		return nil, fmt.Errorf("%w: no subject or email", errGoogleTokenRejected)
	}
	name, _ := payload.Claims["name"].(string)
	nonce, _ := payload.Claims["nonce"].(string)
	return &GoogleIdentity{Subject: payload.Subject, Email: normalizeEmail(email), Name: name, Nonce: nonce}, nil
}

// NewGoogleNonce returns a random nonce for one Google sign-in.
func NewGoogleNonce() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func errGoogleRefused() error {
	return apperrors.NewUnauthenticated("Google sign-in failed; try again")
}

// NotOnListDetails let the screen prefill an Access request with what was
// proven, and carry the request token that sends it.
type NotOnListDetails struct {
	Email        string `json:"email"`
	FullName     string `json:"full_name"`
	RequestToken string `json:"request_token"`
}

func errNotOnList(email, name, requestToken string) error {
	return &apperrors.AppError{
		Code:       "NOT_ON_LIST",
		Message:    "this email isn't on any list for LINKS yet",
		HTTPStatus: 403,
		Details:    NotOnListDetails{Email: email, FullName: name, RequestToken: requestToken},
	}
}

func errAccountNotActive(status UserStatus) error {
	return &apperrors.AppError{
		Code:       "ACCOUNT_NOT_ACTIVE",
		Message:    "this account can't sign in now",
		HTTPStatus: 403,
		Details:    map[string]string{"status": string(status)},
	}
}

// SignInWithGoogle signs in the member a verified Google ID token names.
// expectedNonce is the nonce this browser was given; the token must carry it,
// so a token obtained in another browser can't sign this one in.
func (s *authService) SignInWithGoogle(ctx context.Context, credential, expectedNonce string) (*LoginResponse, string, error) {
	if s.google == nil {
		return nil, "", errGoogleRefused()
	}
	identity, err := s.google.Verify(ctx, credential)
	if err != nil {
		if errors.Is(err, errGoogleTokenRejected) {
			return nil, "", errGoogleRefused()
		}
		return nil, "", fmt.Errorf("verify Google token: %w", err)
	}
	if expectedNonce == "" || subtle.ConstantTimeCompare([]byte(identity.Nonce), []byte(expectedNonce)) != 1 {
		return nil, "", errGoogleRefused()
	}

	var resp *LoginResponse
	var refreshRaw string
	err = s.unitOfWork.WithinTransaction(ctx, func(repos AuthRepositories) error {
		user, linked, err := findGoogleMember(ctx, repos.Users, identity)
		if err != nil {
			return err
		}
		if user == nil {
			return s.notOnList(identity.Email, identity.Name)
		}
		if !user.CanSignIn() {
			return errAccountNotActive(user.Status)
		}

		now := time.Now()
		if !linked {
			if err := repos.Users.SetGoogleSubject(ctx, user.ID, identity.Subject); err != nil {
				var pgErr *pgconn.PgError
				if errors.As(err, &pgErr) && pgErr.Code == "23505" {
					return errGoogleRefused()
				}
				return fmt.Errorf("link Google account: %w", err)
			}
			if err := repos.AuditLogs.Create(ctx, &AuditLog{
				ActorID:      &user.ID,
				Action:       "auth.google_linked",
				ResourceType: "user",
				ResourceID:   &user.ID,
				CreatedAt:    now,
			}); err != nil {
				return fmt.Errorf("audit Google link: %w", err)
			}
		}

		resp, refreshRaw, err = s.signIn(ctx, repos, user, "google", now)
		return err
	})
	if err != nil {
		return nil, "", err
	}
	return resp, refreshRaw, nil
}

// findGoogleMember finds the member by Google account ID, or else by verified
// email if that member has no Google account linked yet. linked reports
// whether the member was found by Google account ID. A member already linked
// to a different Google account is refused, so a second Google account with
// the same email can't take theirs.
func findGoogleMember(ctx context.Context, users UserRepository, identity *GoogleIdentity) (*User, bool, error) {
	user, err := users.FindByGoogleSubjectForUpdate(ctx, identity.Subject)
	if err != nil {
		return nil, false, fmt.Errorf("find user by Google account: %w", err)
	}
	if user != nil {
		return user, true, nil
	}

	user, err = users.FindByEmailForUpdate(ctx, identity.Email)
	if err != nil {
		return nil, false, fmt.Errorf("find user by email: %w", err)
	}
	if user == nil {
		return nil, false, nil
	}
	if user.GoogleSubject != nil {
		return nil, false, errGoogleRefused()
	}
	return user, false, nil
}
