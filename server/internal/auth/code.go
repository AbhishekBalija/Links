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
	// PerEmailLimit caps code requests for one email from one address within
	// Window, and PerIPLimit all requests from one address. Counting an
	// email's requests per address means someone else asking for a member's
	// codes uses up only their own allowance, not the member's (#178).
	PerEmailLimit int
	PerIPLimit    int
	Window        time.Duration
	// PerEmailDailyLimit caps code requests for one email from one address
	// within a day.
	PerEmailDailyLimit int
	// EmailDailyCeiling caps the codes one email gets in a day from all
	// addresses together, so many addresses can't flood an inbox or guess
	// much: at most this many codes with MaxAttempts tries each.
	EmailDailyCeiling int
	// EveryRoleUsesCodes lets the principal and admins sign in with a code
	// too. Only test copies turn it on (EMAIL_CODE_FOR_EVERY_ROLE).
	EveryRoleUsesCodes bool
	// WrongTriesPerDay wrong guesses at an email's codes asked for from one
	// address within a day stop new codes for that email from that address
	// until the day is over, so nobody can keep guessing by asking for code
	// after code. Other addresses, the member's own phone included, still
	// get codes.
	WrongTriesPerDay int
	// NotOnListDailyLimit caps codes sent to emails on no list across the
	// whole site within a day, so nobody can use LINKS to flood inboxes or
	// use up the email quota. Members are never caught by it.
	NotOnListDailyLimit int
	// MinReplyTime pads every code request to at least this long, so the
	// time taken to send an email doesn't tell a known email from an
	// unknown one.
	MinReplyTime time.Duration
}

// DefaultCodeSettings are the limits from spec #129. The per-IP limit is
// high enough for a campus signing in together behind one address on day
// one, when no browser is known yet: about 2,400 an hour. Each email still
// has its own limits; this one only stops a single source mailing codes to
// many people. CODES_PER_NETWORK_PER_15_MIN changes it per copy.
func DefaultCodeSettings() CodeSettings {
	return CodeSettings{
		TTL:                 10 * time.Minute,
		MaxAttempts:         5,
		PerEmailLimit:       3,
		PerIPLimit:          600,
		Window:              15 * time.Minute,
		PerEmailDailyLimit:  10,
		EmailDailyCeiling:   30,
		WrongTriesPerDay:    10,
		NotOnListDailyLimit: 50,
		MinReplyTime:        time.Second,
	}
}

// codeRecordsKept is how long code requests stay stored. The daily limits
// count over the same day.
const codeRecordsKept = 24 * time.Hour

func errCodeRefused() error {
	return apperrors.NewUnauthenticated("the code is wrong or has expired")
}

// RequestCode records a code request and emails a code, unless the email
// belongs to an account that can't use one (the principal, an admin, or a
// suspended or rejected account). An email on no list gets a code too, so its
// owner can prove it and send an Access request. It returns the challenge ID
// the browser keeps, the same way for every email. deviceToken is the
// browser's known-device cookie, if it has one.
func (s *authService) RequestCode(ctx context.Context, email, ip, deviceToken string) (string, error) {
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
		dayAgo := start.Add(-codeRecordsKept)
		user, err := repos.Users.FindByEmail(ctx, email)
		if err != nil {
			return fmt.Errorf("find user: %w", err)
		}
		userID := ""
		if user != nil {
			userID = user.ID
		}
		device, err := s.knownDevice(ctx, repos, deviceToken, userID, start)
		if err != nil {
			return err
		}
		var stop bool
		if device != nil {
			challenge.DeviceID = &device.ID
			stop, err = s.deviceLimits(ctx, repos, device.ID, start)
		} else {
			stop, err = s.addressLimits(ctx, repos, emailHash, ipHash, start)
		}
		if err != nil {
			return err
		}
		if stop {
			return repos.SignInCodes.Create(ctx, challenge)
		}

		sendTo = email
		if user != nil {
			allowed, err := s.mayUseEmailCode(ctx, repos.Users, user)
			if err != nil {
				return err
			}
			if !allowed {
				return repos.SignInCodes.Create(ctx, challenge)
			}
			challenge.UserID = &user.ID
			sendTo = *user.Email
		} else {
			sentToday, err := repos.SignInCodes.CountSentToNoListSince(ctx, dayAgo)
			if err != nil {
				return fmt.Errorf("count codes for emails on no list: %w", err)
			}
			if sentToday >= int64(s.codeSettings.NotOnListDailyLimit) {
				return repos.SignInCodes.Create(ctx, challenge)
			}
		}
		code, err = generateCode()
		if err != nil {
			return err
		}
		codeHash := s.codeHash(challenge.ID, code)
		challenge.CodeHash = &codeHash
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

// addressLimits are the limits for a browser not known for the account:
// requests for the email from this address, all requests from this address,
// the email's ceiling from everywhere, and wrong guesses from this address.
// stop means record the request but send no code: the reply still can't
// tell anyone whether the email has an account.
func (s *authService) addressLimits(ctx context.Context, repos AuthRepositories, emailHash, ipHash string, now time.Time) (bool, error) {
	since := now.Add(-s.codeSettings.Window)
	byEmail, err := repos.SignInCodes.CountByEmailAndIPSince(ctx, emailHash, ipHash, since)
	if err != nil {
		return false, fmt.Errorf("count codes for email: %w", err)
	}
	byIP, err := repos.SignInCodes.CountByIPSince(ctx, ipHash, since)
	if err != nil {
		return false, fmt.Errorf("count codes for address: %w", err)
	}
	// Which limit, so the screen can tell one person asking too often from a
	// whole campus on one network; neither says whether the email has an
	// account.
	if byIP >= int64(s.codeSettings.PerIPLimit) {
		return false, apperrors.NewRateLimitedBy("too many codes asked for from this network; try again in 15 minutes", "network")
	}
	if byEmail >= int64(s.codeSettings.PerEmailLimit) {
		return false, apperrors.NewRateLimitedBy("too many codes asked for; try again in 15 minutes", "email")
	}
	dayAgo := now.Add(-codeRecordsKept)
	byEmailToday, err := repos.SignInCodes.CountByEmailAndIPSince(ctx, emailHash, ipHash, dayAgo)
	if err != nil {
		return false, fmt.Errorf("count codes for email today: %w", err)
	}
	everywhereToday, err := repos.SignInCodes.CountByEmailSince(ctx, emailHash, dayAgo)
	if err != nil {
		return false, fmt.Errorf("count codes for email from anywhere today: %w", err)
	}
	if byEmailToday >= int64(s.codeSettings.PerEmailDailyLimit) || everywhereToday >= int64(s.codeSettings.EmailDailyCeiling) {
		return false, apperrors.NewRateLimitedBy("too many codes asked for today; try again tomorrow", "email")
	}
	wrongToday, err := repos.SignInCodes.SumWrongTriesByEmailAndIPSince(ctx, emailHash, ipHash, dayAgo)
	if err != nil {
		return false, fmt.Errorf("count wrong tries for email: %w", err)
	}
	return wrongToday >= int64(s.codeSettings.WrongTriesPerDay), nil
}

// deviceLimits are the limits for a browser known for the account: the same
// numbers, counted for this browser alone, so nothing anyone else does
// counts against it. Only the person who signed in on it has its token.
func (s *authService) deviceLimits(ctx context.Context, repos AuthRepositories, deviceID string, now time.Time) (bool, error) {
	recent, err := repos.SignInCodes.CountByDeviceSince(ctx, deviceID, now.Add(-s.codeSettings.Window))
	if err != nil {
		return false, fmt.Errorf("count codes for device: %w", err)
	}
	if recent >= int64(s.codeSettings.PerEmailLimit) {
		return false, apperrors.NewRateLimitedBy("too many codes asked for; try again in 15 minutes", "email")
	}
	dayAgo := now.Add(-codeRecordsKept)
	today, err := repos.SignInCodes.CountByDeviceSince(ctx, deviceID, dayAgo)
	if err != nil {
		return false, fmt.Errorf("count codes for device today: %w", err)
	}
	if today >= int64(s.codeSettings.PerEmailDailyLimit) {
		return false, apperrors.NewRateLimitedBy("too many codes asked for today; try again tomorrow", "email")
	}
	wrongToday, err := repos.SignInCodes.SumWrongTriesByDeviceSince(ctx, deviceID, dayAgo)
	if err != nil {
		return false, fmt.Errorf("count wrong tries for device: %w", err)
	}
	return wrongToday >= int64(s.codeSettings.WrongTriesPerDay), nil
}

// VerifyCode signs in whoever typed the right code for the challenge, and
// remembers the browser for the account (deviceToken is its cookie, if any). A wrong,
// used, expired or killed code, or an unknown challenge, all get the same
// refusal. The right code for an email on no list gets NOT_ON_LIST with a
// request token; email must then be the address the code was asked for.
func (s *authService) VerifyCode(ctx context.Context, challengeID, email, code, deviceToken string) (*CodeSignIn, error) {
	if _, err := uuid.Parse(challengeID); err != nil {
		return nil, errCodeRefused()
	}

	var resp *LoginResponse
	var refreshRaw, device string
	// outcome is the refusal to give once the transaction has committed, so
	// a used code or a wrong try is kept.
	outcome := errCodeRefused()
	if err := s.unitOfWork.WithinTransaction(ctx, func(repos AuthRepositories) error {
		challenge, err := repos.SignInCodes.FindForUpdate(ctx, challengeID)
		if err != nil {
			return fmt.Errorf("find challenge: %w", err)
		}
		now := time.Now()
		if challenge == nil || challenge.CodeHash == nil ||
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

		user, err := s.codeOwner(ctx, repos.Users, challenge, email)
		if err != nil {
			return err
		}
		if user == nil {
			// The code proved an email on no list.
			email = normalizeEmail(email)
			if s.keyedHash("email", email) != challenge.EmailHash {
				return nil
			}
			outcome = s.notOnList(email, "")
			return nil
		}
		// The account may have changed since the code went out.
		allowed, err := s.mayUseEmailCode(ctx, repos.Users, user)
		if err != nil || !allowed {
			return err
		}
		if !user.CanSignIn() {
			outcome = errAccountNotActive(user.Status)
			return nil
		}
		resp, refreshRaw, err = s.signIn(ctx, repos, user, "email_code", now)
		if err != nil {
			return err
		}
		device, err = s.rememberDevice(ctx, repos, deviceToken, user.ID, now)
		return err
	}); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, outcome
	}
	return &CodeSignIn{Login: resp, Refresh: refreshRaw, Device: device}, nil
}

// codeOwner finds the account the code was for: the one it was sent to, or,
// for a code sent to an email on no list, an account created for that email
// since.
func (s *authService) codeOwner(ctx context.Context, users UserRepository, challenge *SignInCode, email string) (*User, error) {
	// Locked, so two first sign-ins at once complete the account only once.
	if challenge.UserID != nil {
		user, err := users.FindByIDForUpdate(ctx, *challenge.UserID)
		if err != nil {
			return nil, fmt.Errorf("find user: %w", err)
		}
		return user, nil
	}
	email = normalizeEmail(email)
	if email == "" || s.keyedHash("email", email) != challenge.EmailHash {
		return nil, nil
	}
	user, err := users.FindByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}
	if user == nil {
		return nil, nil
	}
	// FindByEmail doesn't load the profile and identity a first sign-in shows.
	return users.FindByIDForUpdate(ctx, user.ID)
}

// notOnList is the NOT_ON_LIST refusal for a proven email, carrying a request
// token for an Access request.
func (s *authService) notOnList(email, name string) error {
	token, err := SignAccessRequestToken(s.tokenCfg, email, time.Now().Add(RequestTokenTTL))
	if err != nil {
		return err
	}
	return errNotOnList(email, name, token)
}

// mayUseEmailCode reports whether the account may be sent an email code:
// not the principal or an admin, who sign in with Google only (ADR 0026)
// unless this is a test copy, and not a suspended or rejected account.
func (s *authService) mayUseEmailCode(ctx context.Context, users UserRepository, user *User) (bool, error) {
	if user.Email == nil || user.Status == UserStatusSuspended || user.Status == UserStatusRejected {
		return false, nil
	}
	if s.codeSettings.EveryRoleUsesCodes {
		return true, nil
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
