package auth

// Email code: someone types their email, gets a 6-digit code, and types it in
// the same browser to sign in (spec #129, ADR 0026). The reply never says
// whether the email belongs to anyone.

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"

	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
)

// CodeSentMessage is the one reply to every accepted code request.
const CodeSentMessage = "If this email can use LINKS, a code is on its way."

const codeDigits = 6

// CodeSettings are the email code's limits.
type CodeSettings struct {
	// TTL is how long a code works.
	TTL time.Duration
	// MaxAttempts wrong tries kill a code.
	MaxAttempts int
	// PerEmailLimit and PerIPLimit cap code requests within Window.
	PerEmailLimit int
	PerIPLimit    int
	Window        time.Duration
	// PerEmailDailyLimit caps code requests per email within a day.
	PerEmailDailyLimit int
	// WrongTriesPerDay wrong guesses at an email's codes within a day stop
	// new codes going to it until the day is over, so nobody can keep
	// guessing by asking for code after code.
	WrongTriesPerDay int
	// MinReplyTime pads every code request to at least this long, so the
	// time taken to send an email doesn't tell a known email from an
	// unknown one.
	MinReplyTime time.Duration
}

// DefaultCodeSettings are the limits from spec #129. The per-IP limit is
// high enough for a class signing in together behind one campus address.
func DefaultCodeSettings() CodeSettings {
	return CodeSettings{
		TTL:                10 * time.Minute,
		MaxAttempts:        5,
		PerEmailLimit:      3,
		PerIPLimit:         60,
		Window:             15 * time.Minute,
		PerEmailDailyLimit: 10,
		WrongTriesPerDay:   10,
		MinReplyTime:       time.Second,
	}
}

// codeRecordsKept is how long code requests stay stored. The daily limits
// count over the same day.
const codeRecordsKept = 24 * time.Hour

func errCodeRefused() error {
	return apperrors.NewUnauthenticated("the code is wrong or has expired")
}

// RequestCode records a code request and, when the email belongs to an account
// that may sign in this way, emails it a code. It returns the challenge ID the
// browser keeps, the same way for every email.
func (s *authService) RequestCode(ctx context.Context, email, ip string) (string, error) {
	start := time.Now()
	defer waitUntil(ctx, start.Add(s.codeSettings.MinReplyTime))

	email = normalizeEmail(email)
	emailHash := s.keyedHash("email", email)
	ipHash := s.keyedHash("ip", ip)

	var code string
	var sendTo string
	challenge := &SignInCode{
		ID:        uuid.NewString(),
		EmailHash: emailHash,
		IPHash:    ipHash,
		CreatedAt: start,
		ExpiresAt: start.Add(s.codeSettings.TTL),
	}
	if err := s.unitOfWork.WithinTransaction(ctx, func(repos AuthRepositories) error {
		// One request per email at a time, so concurrent requests can't
		// slip past the per-email limit.
		if err := repos.SignInCodes.LockEmail(ctx, emailHash); err != nil {
			return fmt.Errorf("lock email: %w", err)
		}
		if err := repos.SignInCodes.DeleteCreatedBefore(ctx, start.Add(-codeRecordsKept)); err != nil {
			return fmt.Errorf("delete old codes: %w", err)
		}
		since := start.Add(-s.codeSettings.Window)
		byEmail, err := repos.SignInCodes.CountByEmailSince(ctx, emailHash, since)
		if err != nil {
			return fmt.Errorf("count codes for email: %w", err)
		}
		byIP, err := repos.SignInCodes.CountByIPSince(ctx, ipHash, since)
		if err != nil {
			return fmt.Errorf("count codes for address: %w", err)
		}
		if byEmail >= int64(s.codeSettings.PerEmailLimit) || byIP >= int64(s.codeSettings.PerIPLimit) {
			return apperrors.NewRateLimited("too many codes asked for; try again in 15 minutes")
		}
		dayAgo := start.Add(-codeRecordsKept)
		byEmailToday, err := repos.SignInCodes.CountByEmailSince(ctx, emailHash, dayAgo)
		if err != nil {
			return fmt.Errorf("count codes for email today: %w", err)
		}
		if byEmailToday >= int64(s.codeSettings.PerEmailDailyLimit) {
			return apperrors.NewRateLimited("too many codes asked for today; try again tomorrow")
		}
		// Only accounts get codes, so only they collect wrong guesses. They
		// get the usual reply with no code, so the reply can't tell anyone
		// the email has an account.
		wrongToday, err := repos.SignInCodes.SumWrongTriesByEmailSince(ctx, emailHash, dayAgo)
		if err != nil {
			return fmt.Errorf("count wrong tries for email: %w", err)
		}
		if wrongToday >= int64(s.codeSettings.WrongTriesPerDay) {
			return repos.SignInCodes.Create(ctx, challenge)
		}

		user, err := repos.Users.FindByEmail(ctx, email)
		if err != nil {
			return fmt.Errorf("find user: %w", err)
		}
		if user != nil {
			allowed, err := mayUseEmailCode(ctx, repos.Users, user)
			if err != nil {
				return err
			}
			if allowed {
				code, err = generateCode()
				if err != nil {
					return err
				}
				codeHash := s.codeHash(challenge.ID, code)
				challenge.UserID = &user.ID
				challenge.CodeHash = &codeHash
				sendTo = *user.Email
			}
		}
		return repos.SignInCodes.Create(ctx, challenge)
	}); err != nil {
		return "", err
	}

	if code != "" {
		if err := s.mailer.SendSignInCode(sendTo, code); err != nil {
			return "", fmt.Errorf("send sign-in code: %w", err)
		}
	}
	return challenge.ID, nil
}

// VerifyCode signs in whoever typed the right code for the challenge. A wrong,
// used, expired or killed code, or an unknown challenge, all get the same
// refusal.
func (s *authService) VerifyCode(ctx context.Context, challengeID, code string) (*LoginResponse, string, error) {
	if _, err := uuid.Parse(challengeID); err != nil {
		return nil, "", errCodeRefused()
	}

	var resp *LoginResponse
	var refreshRaw string
	if err := s.unitOfWork.WithinTransaction(ctx, func(repos AuthRepositories) error {
		challenge, err := repos.SignInCodes.FindForUpdate(ctx, challengeID)
		if err != nil {
			return fmt.Errorf("find challenge: %w", err)
		}
		now := time.Now()
		if challenge == nil || challenge.CodeHash == nil || challenge.UserID == nil ||
			challenge.UsedAt != nil || !now.Before(challenge.ExpiresAt) ||
			challenge.Attempts >= s.codeSettings.MaxAttempts {
			return nil
		}
		if !hmac.Equal([]byte(s.codeHash(challenge.ID, code)), []byte(*challenge.CodeHash)) {
			// Committed with the refusal, so the try counts.
			return repos.SignInCodes.RecordWrongTry(ctx, challenge.ID)
		}
		if err := repos.SignInCodes.MarkUsed(ctx, challenge.ID, now); err != nil {
			return fmt.Errorf("use code: %w", err)
		}

		user, err := repos.Users.FindByID(ctx, *challenge.UserID)
		if err != nil {
			return fmt.Errorf("find user: %w", err)
		}
		if user == nil {
			return nil
		}
		// The account may have changed since the code went out.
		allowed, err := mayUseEmailCode(ctx, repos.Users, user)
		if err != nil || !allowed {
			return err
		}

		resp, refreshRaw, err = s.issueSessionWith(ctx, repos.Users, repos.RefreshTokens, user)
		if err != nil {
			return err
		}
		return repos.AuditLogs.Create(ctx, signInAuditLog(user.ID, "email_code", now))
	}); err != nil {
		return nil, "", err
	}
	if resp == nil {
		return nil, "", errCodeRefused()
	}
	return resp, refreshRaw, nil
}

// mayUseEmailCode reports whether the account can sign in with an email code:
// an active member who isn't the principal or an admin, who sign in with
// Google only (ADR 0026).
func mayUseEmailCode(ctx context.Context, users UserRepository, user *User) (bool, error) {
	if !user.Status.CanLogin() || user.Email == nil {
		return false, nil
	}
	roles, err := users.GetRoleAssignments(ctx, user.ID)
	if err != nil {
		return false, fmt.Errorf("get roles: %w", err)
	}
	return !holdsGoogleOnlyRole(roles), nil
}

// holdsGoogleOnlyRole reports whether any role in effect is principal or
// admin, the accounts that must sign in with Google.
func holdsGoogleOnlyRole(roles []RoleAssignment) bool {
	for _, role := range roles {
		if role.Role == RolePrincipal || role.Role == RoleAdmin {
			return true
		}
	}
	return false
}

func signInAuditLog(userID, method string, at time.Time) *AuditLog {
	return &AuditLog{
		ActorID:      &userID,
		Action:       "auth.signed_in",
		ResourceType: "user",
		ResourceID:   &userID,
		Metadata:     map[string]string{"method": method},
		CreatedAt:    at,
	}
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// keyedHash hides an email or IP address behind an HMAC, so the stored value
// only serves to count requests.
func (s *authService) keyedHash(kind, value string) string {
	mac := hmac.New(sha256.New, []byte(s.tokenCfg.RefreshSecret))
	mac.Write([]byte("sign-in-code:" + kind + ":" + value))
	return hex.EncodeToString(mac.Sum(nil))
}

// codeHash ties a code to its challenge, so a code is useless with any other
// challenge and a leaked table can't be reversed without the server secret.
func (s *authService) codeHash(challengeID, code string) string {
	return s.keyedHash("code", challengeID+":"+code)
}

func generateCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", fmt.Errorf("generate code: %w", err)
	}
	return fmt.Sprintf("%0*d", codeDigits, n.Int64()), nil
}

// waitUntil sleeps until the deadline, or until the request is gone.
func waitUntil(ctx context.Context, deadline time.Time) {
	wait := time.Until(deadline)
	if wait <= 0 {
		return
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-timer.C:
	case <-ctx.Done():
	}
}
