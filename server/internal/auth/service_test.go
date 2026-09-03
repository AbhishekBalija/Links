package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
)

// Fakes for the auth service dependencies. All function fields default to
// inert behavior (nil/empty results, no-op writes) so each test only stubs
// the calls the path under test actually makes.

type fakeUserRepo struct {
	findByEmail           func(ctx context.Context, email string) (*User, error)
	findByID              func(ctx context.Context, id string) (*User, error)
	findByIDForUpdate     func(ctx context.Context, id string) (*User, error)
	findPendingUsers      func(ctx context.Context) ([]User, error)
	findEmailByUserID     func(ctx context.Context, userID string) (*string, error)
	findPhoneByUserID     func(ctx context.Context, userID string) (*string, error)
	findDepartmentByCode  func(ctx context.Context, code string) (*Department, error)
	create                func(ctx context.Context, user *User) error
	update                func(ctx context.Context, user *User) error
	updateStatus          func(ctx context.Context, id string, status UserStatus) error
	createProfile         func(ctx context.Context, profile *Profile) error
	createStudentIdentity func(ctx context.Context, identity *StudentIdentity) error
	getRoleAssignments    func(ctx context.Context, userID string) ([]RoleAssignment, error)
	createRoleAssignment  func(ctx context.Context, ra *RoleAssignment) error

	createdUsers           []*User
	updatedUsers           []*User
	createdProfiles        []*Profile
	createdIdentities      []*StudentIdentity
	createdRoleAssignments []*RoleAssignment
}

func (f *fakeUserRepo) Create(ctx context.Context, user *User) error {
	if f.create != nil {
		return f.create(ctx, user)
	}
	if user.ID == "" {
		user.ID = uuid.New().String()
	}
	f.createdUsers = append(f.createdUsers, user)
	return nil
}

func (f *fakeUserRepo) FindByEmail(ctx context.Context, email string) (*User, error) {
	if f.findByEmail != nil {
		return f.findByEmail(ctx, email)
	}
	return nil, nil
}

func (f *fakeUserRepo) FindByID(ctx context.Context, id string) (*User, error) {
	if f.findByID != nil {
		return f.findByID(ctx, id)
	}
	return nil, nil
}

func (f *fakeUserRepo) FindByIDForUpdate(ctx context.Context, id string) (*User, error) {
	if f.findByIDForUpdate != nil {
		return f.findByIDForUpdate(ctx, id)
	}
	return nil, nil
}

func (f *fakeUserRepo) FindPendingUsers(ctx context.Context) ([]User, error) {
	if f.findPendingUsers != nil {
		return f.findPendingUsers(ctx)
	}
	return nil, nil
}

func (f *fakeUserRepo) FindEmailByUserID(ctx context.Context, userID string) (*string, error) {
	if f.findEmailByUserID != nil {
		return f.findEmailByUserID(ctx, userID)
	}
	return nil, nil
}

func (f *fakeUserRepo) FindPhoneByUserID(ctx context.Context, userID string) (*string, error) {
	if f.findPhoneByUserID != nil {
		return f.findPhoneByUserID(ctx, userID)
	}
	return nil, nil
}

func (f *fakeUserRepo) FindDepartmentByCode(ctx context.Context, code string) (*Department, error) {
	if f.findDepartmentByCode != nil {
		return f.findDepartmentByCode(ctx, code)
	}
	return nil, nil
}

func (f *fakeUserRepo) Update(ctx context.Context, user *User) error {
	if f.update != nil {
		return f.update(ctx, user)
	}
	f.updatedUsers = append(f.updatedUsers, user)
	return nil
}

func (f *fakeUserRepo) UpdateStatus(ctx context.Context, id string, status UserStatus) error {
	if f.updateStatus != nil {
		return f.updateStatus(ctx, id, status)
	}
	return nil
}

func (f *fakeUserRepo) CreateProfile(ctx context.Context, profile *Profile) error {
	if f.createProfile != nil {
		return f.createProfile(ctx, profile)
	}
	f.createdProfiles = append(f.createdProfiles, profile)
	return nil
}

func (f *fakeUserRepo) CreateStudentIdentity(ctx context.Context, identity *StudentIdentity) error {
	if f.createStudentIdentity != nil {
		return f.createStudentIdentity(ctx, identity)
	}
	f.createdIdentities = append(f.createdIdentities, identity)
	return nil
}

func (f *fakeUserRepo) GetRoleAssignments(ctx context.Context, userID string) ([]RoleAssignment, error) {
	if f.getRoleAssignments != nil {
		return f.getRoleAssignments(ctx, userID)
	}
	return nil, nil
}

func (f *fakeUserRepo) CreateRoleAssignment(ctx context.Context, ra *RoleAssignment) error {
	if f.createRoleAssignment != nil {
		return f.createRoleAssignment(ctx, ra)
	}
	f.createdRoleAssignments = append(f.createdRoleAssignments, ra)
	return nil
}

type fakeRefreshTokenRepo struct {
	create            func(ctx context.Context, token *RefreshToken) error
	findByHash        func(ctx context.Context, hash string) (*RefreshToken, error)
	revokeIfActive    func(ctx context.Context, hash string) error
	revokeByHash      func(ctx context.Context, hash string) error
	revokeAllByUserID func(ctx context.Context, userID string) error

	created             []*RefreshToken
	revokeIfActiveCalls []string
	revokedHashes       []string
}

func (f *fakeRefreshTokenRepo) Create(ctx context.Context, token *RefreshToken) error {
	if f.create != nil {
		return f.create(ctx, token)
	}
	f.created = append(f.created, token)
	return nil
}

func (f *fakeRefreshTokenRepo) FindByHash(ctx context.Context, hash string) (*RefreshToken, error) {
	if f.findByHash != nil {
		return f.findByHash(ctx, hash)
	}
	return nil, nil
}

func (f *fakeRefreshTokenRepo) RevokeIfActive(ctx context.Context, hash string) error {
	f.revokeIfActiveCalls = append(f.revokeIfActiveCalls, hash)
	if f.revokeIfActive != nil {
		return f.revokeIfActive(ctx, hash)
	}
	return nil
}

func (f *fakeRefreshTokenRepo) RevokeByHash(ctx context.Context, hash string) error {
	f.revokedHashes = append(f.revokedHashes, hash)
	if f.revokeByHash != nil {
		return f.revokeByHash(ctx, hash)
	}
	return nil
}

func (f *fakeRefreshTokenRepo) RevokeAllByUserID(ctx context.Context, userID string) error {
	if f.revokeAllByUserID != nil {
		return f.revokeAllByUserID(ctx, userID)
	}
	return nil
}

type fakeActivationTokenRepo struct {
	create                  func(ctx context.Context, token *AccountActivationToken) error
	findByHash              func(ctx context.Context, hash string) (*AccountActivationToken, error)
	findLatestByUserID      func(ctx context.Context, userID string) (*AccountActivationToken, error)
	markUsed                func(ctx context.Context, id string) error
	revokeAllUnusedByUserID func(ctx context.Context, userID string) error

	created       []*AccountActivationToken
	markedUsed    []string
	revokedUnused []string
}

func (f *fakeActivationTokenRepo) Create(ctx context.Context, token *AccountActivationToken) error {
	if f.create != nil {
		return f.create(ctx, token)
	}
	f.created = append(f.created, token)
	return nil
}

func (f *fakeActivationTokenRepo) FindByHash(ctx context.Context, hash string) (*AccountActivationToken, error) {
	if f.findByHash != nil {
		return f.findByHash(ctx, hash)
	}
	return nil, nil
}

func (f *fakeActivationTokenRepo) FindLatestByUserID(ctx context.Context, userID string) (*AccountActivationToken, error) {
	if f.findLatestByUserID != nil {
		return f.findLatestByUserID(ctx, userID)
	}
	return nil, nil
}

func (f *fakeActivationTokenRepo) MarkUsed(ctx context.Context, id string) error {
	f.markedUsed = append(f.markedUsed, id)
	if f.markUsed != nil {
		return f.markUsed(ctx, id)
	}
	return nil
}

func (f *fakeActivationTokenRepo) RevokeAllUnusedByUserID(ctx context.Context, userID string) error {
	f.revokedUnused = append(f.revokedUnused, userID)
	if f.revokeAllUnusedByUserID != nil {
		return f.revokeAllUnusedByUserID(ctx, userID)
	}
	return nil
}

type fakeAuditLogRepo struct {
	created []*AuditLog
}

func (f *fakeAuditLogRepo) Create(_ context.Context, log *AuditLog) error {
	f.created = append(f.created, log)
	return nil
}

type fakeUnitOfWork struct {
	users         *fakeUserRepo
	refreshTokens *fakeRefreshTokenRepo
	activations   *fakeActivationTokenRepo
	auditLogs     *fakeAuditLogRepo
}

func (u *fakeUnitOfWork) WithinTransaction(ctx context.Context, fn func(AuthRepositories) error) error {
	return fn(AuthRepositories{
		Users:         u.users,
		RefreshTokens: u.refreshTokens,
		Activations:   u.activations,
		AuditLogs:     u.auditLogs,
	})
}

type fakeHasher struct {
	hashFn   func(password string) (string, error)
	verifyFn func(password, hash string) (bool, error)
}

func (f *fakeHasher) Hash(password string) (string, error) {
	if f.hashFn != nil {
		return f.hashFn(password)
	}
	return "hash:" + password, nil
}

func (f *fakeHasher) Verify(password, hash string) (bool, error) {
	if f.verifyFn != nil {
		return f.verifyFn(password, hash)
	}
	return hash == "hash:"+password, nil
}

type sentMail struct {
	to, name, link string
}

type fakeMailer struct {
	err  error
	sent []sentMail
}

func (f *fakeMailer) SendActivationEmail(to, name, activationLink string) error {
	f.sent = append(f.sent, sentMail{to: to, name: name, link: activationLink})
	return f.err
}

type authHarness struct {
	service       AuthService
	users         *fakeUserRepo
	refreshTokens *fakeRefreshTokenRepo
	activations   *fakeActivationTokenRepo
	auditLogs     *fakeAuditLogRepo
	hasher        *fakeHasher
	mailer        *fakeMailer
	cfg           TokenConfig
}

func newAuthHarness(t *testing.T) *authHarness {
	t.Helper()
	users := &fakeUserRepo{}
	refreshTokens := &fakeRefreshTokenRepo{}
	activations := &fakeActivationTokenRepo{}
	auditLogs := &fakeAuditLogRepo{}
	hasher := &fakeHasher{}
	mailer := &fakeMailer{}
	uow := &fakeUnitOfWork{
		users:         users,
		refreshTokens: refreshTokens,
		activations:   activations,
		auditLogs:     auditLogs,
	}
	cfg := TokenConfig{
		AccessSecret:  "newAuthHarness-access-secret",
		RefreshSecret: "newAuthHarness-refresh-secret",
		AccessTTL:     15 * time.Minute,
		RefreshTTL:    7 * 24 * time.Hour,
	}
	service := NewAuthService(users, refreshTokens, activations, uow, cfg, hasher, mailer, "https://links.example.com")
	return &authHarness{
		service:       service,
		users:         users,
		refreshTokens: refreshTokens,
		activations:   activations,
		auditLogs:     auditLogs,
		hasher:        hasher,
		mailer:        mailer,
		cfg:           cfg,
	}
}

func requireAppError(t *testing.T, err error, code string, status int) {
	t.Helper()
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *AppError with code %s, got: %v", code, err)
	}
	if appErr.Code != code {
		t.Fatalf("error code = %q, want %q", appErr.Code, code)
	}
	if appErr.HTTPStatus != status {
		t.Fatalf("error status = %d, want %d", appErr.HTTPStatus, status)
	}
}

func ptrTo[T any](v T) *T {
	return &v
}

func TestRefresh_RotatesAndRevokesOldToken(t *testing.T) {
	h := newAuthHarness(t)
	ctx := context.Background()
	userID := "user-1"
	raw := "old-refresh-token"

	h.users.findByID = func(_ context.Context, id string) (*User, error) {
		return &User{ID: id, Status: UserStatusActive}, nil
	}
	h.users.getRoleAssignments = func(_ context.Context, userID string) ([]RoleAssignment, error) {
		return []RoleAssignment{{UserID: userID, Role: RoleStudent}}, nil
	}
	stored := &RefreshToken{
		ID:        "rt-1",
		UserID:    userID,
		TokenHash: HashRefreshToken(raw),
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}
	h.refreshTokens.findByHash = func(_ context.Context, hash string) (*RefreshToken, error) {
		if hash != stored.TokenHash {
			return nil, nil
		}
		return stored, nil
	}
	revokedHash := ""
	h.refreshTokens.revokeIfActive = func(_ context.Context, hash string) error {
		revokedHash = hash
		return nil
	}

	resp, newRaw, err := h.service.Refresh(ctx, raw)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if revokedHash != stored.TokenHash {
		t.Fatalf("old token hash %q was not revoked (revoked %q)", stored.TokenHash, revokedHash)
	}
	if newRaw == "" || newRaw == raw {
		t.Fatal("expected a rotated refresh token")
	}
	if len(h.refreshTokens.created) != 1 {
		t.Fatalf("expected 1 new refresh token stored, got %d", len(h.refreshTokens.created))
	}
	created := h.refreshTokens.created[0]
	if created.TokenHash != HashRefreshToken(newRaw) {
		t.Fatal("stored token hash does not match the rotated token")
	}
	if created.UserID != userID {
		t.Fatalf("new token user = %q, want %q", created.UserID, userID)
	}
	if resp.ExpiresIn != int(h.cfg.AccessTTL.Seconds()) {
		t.Fatalf("expires_in = %d, want %d", resp.ExpiresIn, int(h.cfg.AccessTTL.Seconds()))
	}
	claims, err := ValidateAccessToken(resp.AccessToken, h.cfg)
	if err != nil {
		t.Fatalf("new access token does not validate: %v", err)
	}
	if claims.UserID != userID {
		t.Fatalf("access token subject = %q, want %q", claims.UserID, userID)
	}
	if len(claims.Roles) != 1 || claims.Roles[0] != string(RoleStudent) {
		t.Fatalf("access token roles = %v, want [student]", claims.Roles)
	}
}

func TestRefresh_RejectsExpiredRevokedAndUnknownTokens(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name   string
		stored *RefreshToken
	}{
		{"expired", &RefreshToken{ID: "rt-2", UserID: "user-1", TokenHash: "h", ExpiresAt: now.Add(-time.Minute)}},
		{"revoked", &RefreshToken{ID: "rt-3", UserID: "user-1", TokenHash: "h", ExpiresAt: now.Add(time.Hour), RevokedAt: &now}},
		{"unknown", nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newAuthHarness(t)
			h.refreshTokens.findByHash = func(_ context.Context, _ string) (*RefreshToken, error) {
				return tc.stored, nil
			}
			_, _, err := h.service.Refresh(context.Background(), "raw-token")
			requireAppError(t, err, "UNAUTHENTICATED", http.StatusUnauthorized)
			if len(h.refreshTokens.created) != 0 {
				t.Fatal("no new token may be issued for a rejected refresh")
			}
		})
	}
}

func TestRefresh_ReplayDoesNotIssueNewToken(t *testing.T) {
	h := newAuthHarness(t)
	raw := "raw-token"
	now := time.Now()

	h.refreshTokens.findByHash = func(_ context.Context, hash string) (*RefreshToken, error) {
		return &RefreshToken{
			ID:        "rt-1",
			UserID:    "user-1",
			TokenHash: HashRefreshToken(raw),
			ExpiresAt: now.Add(time.Hour),
		}, nil
	}
	h.refreshTokens.revokeIfActive = func(_ context.Context, _ string) error {
		return fmt.Errorf("%w: token already revoked, expired, or missing", errRefreshTokenUnavailable)
	}

	_, _, err := h.service.Refresh(context.Background(), raw)
	requireAppError(t, err, "UNAUTHENTICATED", http.StatusUnauthorized)
	if len(h.refreshTokens.created) != 0 {
		t.Fatal("a replayed refresh token must not produce a new token")
	}
}

func TestRefresh_RejectsInactiveUser(t *testing.T) {
	h := newAuthHarness(t)
	now := time.Now()

	h.refreshTokens.findByHash = func(_ context.Context, _ string) (*RefreshToken, error) {
		return &RefreshToken{ID: "rt-1", UserID: "user-1", TokenHash: "h", ExpiresAt: now.Add(time.Hour)}, nil
	}
	h.users.findByID = func(_ context.Context, id string) (*User, error) {
		return &User{ID: id, Status: UserStatusSuspended}, nil
	}

	_, _, err := h.service.Refresh(context.Background(), "raw-token")
	requireAppError(t, err, "UNAUTHENTICATED", http.StatusUnauthorized)
}

func TestActivateAccount_ActivatesPendingUser(t *testing.T) {
	h := newAuthHarness(t)
	ctx := context.Background()
	userID := "user-1"
	raw := "raw-activation-token"
	activation := &AccountActivationToken{
		ID:        "tok-1",
		UserID:    userID,
		TokenHash: HashRefreshToken(raw),
		Purpose:   "activate",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}

	h.activations.findByHash = func(_ context.Context, hash string) (*AccountActivationToken, error) {
		if hash != activation.TokenHash {
			return nil, nil
		}
		return activation, nil
	}
	h.users.findByID = func(_ context.Context, id string) (*User, error) {
		return &User{ID: id, Status: UserStatusPending, PasswordHash: "hash:old-password"}, nil
	}

	if err := h.service.ActivateAccount(ctx, raw, "NewPass1"); err != nil {
		t.Fatalf("activate: %v", err)
	}
	if len(h.activations.markedUsed) != 1 || h.activations.markedUsed[0] != activation.ID {
		t.Fatalf("activation token %q not marked used, got %v", activation.ID, h.activations.markedUsed)
	}
	if len(h.users.updatedUsers) != 1 {
		t.Fatalf("expected one user update, got %d", len(h.users.updatedUsers))
	}
	updated := h.users.updatedUsers[0]
	if updated.Status != UserStatusActive || !updated.IsVerified {
		t.Fatalf("user not activated: status=%q verified=%v", updated.Status, updated.IsVerified)
	}
	if updated.PasswordHash != "hash:NewPass1" {
		t.Fatalf("password hash not updated: %q", updated.PasswordHash)
	}
}

func TestActivateAccount_RejectsInvalidTokens(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name       string
		activation *AccountActivationToken
		markUsed   func(ctx context.Context, id string) error
	}{
		{"used", &AccountActivationToken{ID: "t", UserID: "u", ExpiresAt: now.Add(time.Hour), UsedAt: &now}, nil},
		{"expired", &AccountActivationToken{ID: "t", UserID: "u", ExpiresAt: now.Add(-time.Hour)}, nil},
		{"unknown", nil, nil},
		{"consumed concurrently", &AccountActivationToken{ID: "t", UserID: "u", ExpiresAt: now.Add(time.Hour)}, func(_ context.Context, _ string) error {
			return fmt.Errorf("%w: token already used, expired, or missing", errActivationTokenUnavailable)
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newAuthHarness(t)
			h.activations.findByHash = func(_ context.Context, _ string) (*AccountActivationToken, error) {
				return tc.activation, nil
			}
			if tc.markUsed != nil {
				h.activations.markUsed = tc.markUsed
			}
			err := h.service.ActivateAccount(context.Background(), "raw-token", "NewPass1")
			requireAppError(t, err, "UNAUTHENTICATED", http.StatusUnauthorized)
			if len(h.users.updatedUsers) != 0 {
				t.Fatal("user must not be updated when activation fails")
			}
		})
	}
}

func TestActivateAccount_MissingUserIsNotFound(t *testing.T) {
	h := newAuthHarness(t)
	now := time.Now()

	h.activations.findByHash = func(_ context.Context, _ string) (*AccountActivationToken, error) {
		return &AccountActivationToken{ID: "t", UserID: "u", ExpiresAt: now.Add(time.Hour)}, nil
	}
	h.users.findByID = func(_ context.Context, _ string) (*User, error) {
		return nil, nil
	}

	requireAppError(t, h.service.ActivateAccount(context.Background(), "raw-token", "NewPass1"), "NOT_FOUND", http.StatusNotFound)
}

func TestVerifyUser_ApprovesPendingStudentAndIssuesActivation(t *testing.T) {
	h := newAuthHarness(t)
	ctx := context.Background()
	userID, actorID := "user-1", "admin-1"
	email := "student@example.com"

	h.users.findByIDForUpdate = func(_ context.Context, id string) (*User, error) {
		return &User{
			ID:     id,
			Email:  &email,
			Status: UserStatusPending,
			Profile: &Profile{UserID: id, FullName: "Test Student"},
		}, nil
	}

	if err := h.service.VerifyUser(ctx, actorID, userID, "", "", "looks good"); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if len(h.users.createdRoleAssignments) != 1 {
		t.Fatalf("expected one role assignment, got %d", len(h.users.createdRoleAssignments))
	}
	ra := h.users.createdRoleAssignments[0]
	if ra.UserID != userID || ra.Role != RoleStudent {
		t.Fatalf("unexpected role assignment: %+v", ra)
	}
	if ra.ScopeType != ScopeGlobal || ra.ScopeID != nil {
		t.Fatalf("expected default global scope, got scope_type=%q scope_id=%v", ra.ScopeType, ra.ScopeID)
	}
	if ra.AssignedBy == nil || *ra.AssignedBy != actorID {
		t.Fatalf("role not assigned by actor: %+v", ra.AssignedBy)
	}
	if len(h.users.updatedUsers) != 1 || !h.users.updatedUsers[0].IsVerified {
		t.Fatal("user was not marked verified")
	}
	if len(h.auditLogs.created) != 1 || h.auditLogs.created[0].Action != "user_verified" {
		t.Fatalf("expected user_verified audit log, got %+v", h.auditLogs.created)
	}
	if len(h.activations.created) != 1 {
		t.Fatalf("expected one activation token, got %d", len(h.activations.created))
	}
	token := h.activations.created[0]
	if token.UserID != userID || token.Purpose != "activate" {
		t.Fatalf("unexpected activation token: %+v", token)
	}
	if len(h.mailer.sent) != 1 {
		t.Fatalf("expected one activation email, got %d", len(h.mailer.sent))
	}
	sent := h.mailer.sent[0]
	if sent.to != email || sent.name != "Test Student" {
		t.Fatalf("email sent to %q (%q), want %q (Test Student)", sent.to, sent.name, email)
	}
	const prefix = "https://links.example.com/activate?token="
	if !strings.HasPrefix(sent.link, prefix) {
		t.Fatalf("activation link %q does not start with %q", sent.link, prefix)
	}
	if HashRefreshToken(strings.TrimPrefix(sent.link, prefix)) != token.TokenHash {
		t.Fatal("activation link token does not hash to the stored token")
	}
}

func TestVerifyUser_RequiresPendingUnverifiedUser(t *testing.T) {
	tests := []struct {
		name   string
		user   *User
		code   string
		status int
	}{
		{"already active", &User{ID: "u", Status: UserStatusActive}, "CONFLICT", http.StatusConflict},
		{"already verified", &User{ID: "u", Status: UserStatusPending, IsVerified: true}, "CONFLICT", http.StatusConflict},
		{"missing", nil, "NOT_FOUND", http.StatusNotFound},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newAuthHarness(t)
			h.users.findByIDForUpdate = func(_ context.Context, _ string) (*User, error) {
				return tc.user, nil
			}
			err := h.service.VerifyUser(context.Background(), "admin-1", "u", "", "", "")
			requireAppError(t, err, tc.code, tc.status)
			if len(h.mailer.sent) != 0 {
				t.Fatal("no email should be sent for a rejected approval")
			}
		})
	}
}

func TestVerifyUser_InvalidatesTokenWhenEmailFails(t *testing.T) {
	h := newAuthHarness(t)
	ctx := context.Background()
	email := "student@example.com"

	h.users.findByIDForUpdate = func(_ context.Context, id string) (*User, error) {
		return &User{ID: id, Email: &email, Status: UserStatusPending}, nil
	}
	h.mailer.err = errors.New("resend is down")

	if err := h.service.VerifyUser(ctx, "admin-1", "u", "", "", ""); err == nil {
		t.Fatal("expected error when activation email fails")
	}
	if len(h.activations.created) != 1 {
		t.Fatalf("expected an activation token to be issued, got %d", len(h.activations.created))
	}
	if len(h.activations.markedUsed) != 1 || h.activations.markedUsed[0] != h.activations.created[0].ID {
		t.Fatalf("failed email must invalidate the issued token, marked used: %v", h.activations.markedUsed)
	}
}

func TestRequestAccess_CreatesPendingUserAndProfile(t *testing.T) {
	h := newAuthHarness(t)
	ctx := context.Background()
	year := 2024

	h.users.findDepartmentByCode = func(_ context.Context, code string) (*Department, error) {
		if code != "CS" {
			return nil, nil
		}
		return &Department{ID: "dept-cs", Code: "CS", Name: "Computer Science"}, nil
	}

	resp, err := h.service.RequestAccess(ctx, RequestAccessInput{
		Email:          "new@example.com",
		Password:       "StrongPass1",
		FullName:       "New Student",
		DepartmentCode: "CS",
		BatchYear:      &year,
		Phone:          "  +1-555-0100  ",
	})
	if err != nil {
		t.Fatalf("request access: %v", err)
	}
	if resp.UserID == "" || resp.Status != string(UserStatusPending) {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if len(h.users.createdUsers) != 1 {
		t.Fatalf("expected one created user, got %d", len(h.users.createdUsers))
	}
	u := h.users.createdUsers[0]
	if u.Status != UserStatusPending || u.IsVerified {
		t.Fatalf("new user must start pending and unverified: %+v", u)
	}
	if u.Email == nil || *u.Email != "new@example.com" {
		t.Fatalf("unexpected email: %v", u.Email)
	}
	if u.Phone == nil || *u.Phone != "+1-555-0100" {
		t.Fatalf("phone was not trimmed: %v", u.Phone)
	}
	if len(h.users.createdProfiles) != 1 {
		t.Fatalf("expected one profile, got %d", len(h.users.createdProfiles))
	}
	p := h.users.createdProfiles[0]
	if p.UserID != u.ID || p.FullName != "New Student" {
		t.Fatalf("unexpected profile: %+v", p)
	}
	if !strings.HasPrefix(p.Username, "new.student") {
		t.Fatalf("expected username derived from full name, got %q", p.Username)
	}
	if len(h.users.createdIdentities) != 0 {
		t.Fatal("no student identity expected when USN is empty")
	}
}

func TestRequestAccess_RejectsRegisteredEmail(t *testing.T) {
	h := newAuthHarness(t)
	email := "taken@example.com"

	h.users.findByEmail = func(_ context.Context, e string) (*User, error) {
		if e == email {
			return &User{ID: "u-1", Email: &email}, nil
		}
		return nil, nil
	}

	_, err := h.service.RequestAccess(context.Background(), RequestAccessInput{
		Email: email, Password: "StrongPass1", FullName: "Someone",
	})
	requireAppError(t, err, "CONFLICT", http.StatusConflict)
	if len(h.users.createdUsers) != 0 {
		t.Fatal("no user may be created for a duplicate email")
	}
}

func TestRequestAccess_MapsUniqueEmailViolationToConflict(t *testing.T) {
	h := newAuthHarness(t)

	h.users.create = func(_ context.Context, _ *User) error {
		return &pgconn.PgError{Code: "23505", ConstraintName: "idx_users_email"}
	}

	_, err := h.service.RequestAccess(context.Background(), RequestAccessInput{
		Email: "dup@example.com", Password: "StrongPass1", FullName: "Not So Fast",
	})
	requireAppError(t, err, "CONFLICT", http.StatusConflict)
}

func TestRequestAccess_DerivesDepartmentFromUSN(t *testing.T) {
	h := newAuthHarness(t)
	ctx := context.Background()
	year := 2023

	h.users.findDepartmentByCode = func(_ context.Context, code string) (*Department, error) {
		if code != "CS" {
			return nil, nil
		}
		return &Department{ID: "dept-cs", Code: "CS", Name: "Computer Science"}, nil
	}

	_, err := h.service.RequestAccess(ctx, RequestAccessInput{
		Email: "student@example.com", Password: "StrongPass1", FullName: "A B",
		USN: "4MN23CS005", BatchYear: &year,
	})
	if err != nil {
		t.Fatalf("request access: %v", err)
	}
	if len(h.users.createdIdentities) != 1 {
		t.Fatalf("expected one student identity, got %d", len(h.users.createdIdentities))
	}
	id := h.users.createdIdentities[0]
	if id.USN != "4MN23CS005" || id.DepartmentID != "dept-cs" || id.BatchYear != year {
		t.Fatalf("unexpected student identity: %+v", id)
	}
}

func TestRequestAccess_RejectsBadDepartmentInputs(t *testing.T) {
	tests := []struct {
		name  string
		input RequestAccessInput
	}{
		{"invalid usn", RequestAccessInput{
			Email: "a@example.com", Password: "StrongPass1", FullName: "A B", USN: "4MN99CS001",
		}},
		{"unknown department code", RequestAccessInput{
			Email: "a@example.com", Password: "StrongPass1", FullName: "A B", DepartmentCode: "XX",
		}},
		{"valid usn whose department is not seeded", RequestAccessInput{
			Email: "a@example.com", Password: "StrongPass1", FullName: "A B", USN: "4MN23CS001",
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newAuthHarness(t)
			_, err := h.service.RequestAccess(context.Background(), tc.input)
			requireAppError(t, err, "VALIDATION_ERROR", http.StatusBadRequest)
		})
	}
}

func TestRequestAccess_RetriesUsernameCollision(t *testing.T) {
	h := newAuthHarness(t)
	calls := 0

	h.users.createProfile = func(_ context.Context, _ *Profile) error {
		calls++
		if calls == 1 {
			return &pgconn.PgError{Code: "23505", ConstraintName: "idx_profiles_username"}
		}
		return nil
	}

	_, err := h.service.RequestAccess(context.Background(), RequestAccessInput{
		Email: "x@example.com", Password: "StrongPass1", FullName: "Collision User",
	})
	if err != nil {
		t.Fatalf("request access: %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected profile creation to retry once, got %d calls", calls)
	}
}

func TestRequestAccess_GivesUpAfterFiveUsernameCollisions(t *testing.T) {
	h := newAuthHarness(t)
	attempts := 0

	h.users.createProfile = func(_ context.Context, _ *Profile) error {
		attempts++
		return &pgconn.PgError{Code: "23505", ConstraintName: "idx_profiles_username"}
	}

	_, err := h.service.RequestAccess(context.Background(), RequestAccessInput{
		Email: "x@example.com", Password: "StrongPass1", FullName: "Collision User",
	})
	if err == nil {
		t.Fatal("expected request access to fail after repeated username collisions")
	}
	if attempts != 5 {
		t.Fatalf("expected 5 attempts, got %d", attempts)
	}
}

func TestUpdateUserStatus_TransitionGuards(t *testing.T) {
	tests := []struct {
		name    string
		current UserStatus
		target  string
		code    string
		status  int
	}{
		{"pending to active must use verify endpoint", UserStatusPending, "active", "VALIDATION_ERROR", http.StatusBadRequest},
		{"rejected cannot be activated", UserStatusRejected, "active", "VALIDATION_ERROR", http.StatusBadRequest},
		{"same status is a conflict", UserStatusActive, "active", "CONFLICT", http.StatusConflict},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newAuthHarness(t)
			h.users.findByIDForUpdate = func(_ context.Context, _ string) (*User, error) {
				return &User{ID: "u-1", Status: tc.current}, nil
			}
			err := h.service.UpdateUserStatus(context.Background(), "admin-1", "u-1", tc.target, "")
			requireAppError(t, err, tc.code, tc.status)
			if len(h.auditLogs.created) != 0 {
				t.Fatal("no audit log expected for a rejected transition")
			}
		})
	}
}

func TestUpdateUserStatus_ActiveToSuspendedWritesAuditLog(t *testing.T) {
	h := newAuthHarness(t)
	ctx := context.Background()

	h.users.findByIDForUpdate = func(_ context.Context, _ string) (*User, error) {
		return &User{ID: "u-1", Status: UserStatusActive}, nil
	}

	if err := h.service.UpdateUserStatus(ctx, "admin-1", "u-1", "suspended", "policy violation"); err != nil {
		t.Fatalf("update status: %v", err)
	}
	if len(h.users.updatedUsers) != 1 || h.users.updatedUsers[0].Status != UserStatusSuspended {
		t.Fatalf("user status not updated: %+v", h.users.updatedUsers)
	}
	if len(h.auditLogs.created) != 1 {
		t.Fatalf("expected one audit log, got %d", len(h.auditLogs.created))
	}
	al := h.auditLogs.created[0]
	if al.Action != "user_status_changed" {
		t.Fatalf("unexpected audit action: %q", al.Action)
	}
	meta, ok := al.Metadata.(map[string]string)
	if !ok {
		t.Fatalf("audit metadata has unexpected type %T", al.Metadata)
	}
	if meta["new_status"] != "suspended" || meta["note"] != "policy violation" {
		t.Fatalf("unexpected audit metadata: %v", meta)
	}
}

func TestUpdateUserStatus_MissingUserIsNotFound(t *testing.T) {
	h := newAuthHarness(t)
	h.users.findByIDForUpdate = func(_ context.Context, _ string) (*User, error) {
		return nil, nil
	}
	requireAppError(t, h.service.UpdateUserStatus(context.Background(), "admin-1", "u-1", "suspended", ""), "NOT_FOUND", http.StatusNotFound)
}

func TestLogin_IssuesTokensForActiveUser(t *testing.T) {
	h := newAuthHarness(t)
	ctx := context.Background()
	email := "active@example.com"
	hash := "hash:CorrectPass1"

	h.users.findByEmail = func(_ context.Context, e string) (*User, error) {
		if e == email {
			return &User{ID: "u-1", Email: &email, PasswordHash: hash, Status: UserStatusActive}, nil
		}
		return nil, nil
	}
	h.users.getRoleAssignments = func(_ context.Context, userID string) ([]RoleAssignment, error) {
		return []RoleAssignment{
			{UserID: userID, Role: RoleStudent},
			{UserID: userID, Role: RoleHOD},
		}, nil
	}

	resp, raw, err := h.service.Login(ctx, LoginInput{Email: email, Password: "CorrectPass1"})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if raw == "" {
		t.Fatal("expected a refresh token")
	}
	if len(h.refreshTokens.created) != 1 || h.refreshTokens.created[0].TokenHash != HashRefreshToken(raw) {
		t.Fatalf("refresh token not stored hashed: %+v", h.refreshTokens.created)
	}
	claims, err := ValidateAccessToken(resp.AccessToken, h.cfg)
	if err != nil {
		t.Fatalf("access token does not validate: %v", err)
	}
	if claims.UserID != "u-1" {
		t.Fatalf("access token subject = %q, want u-1", claims.UserID)
	}
	if len(claims.Roles) != 2 || claims.Roles[0] != string(RoleStudent) || claims.Roles[1] != string(RoleHOD) {
		t.Fatalf("access token roles = %v, want [student hod]", claims.Roles)
	}
}

func TestLogin_RejectsInactiveUsersAndBadCredentials(t *testing.T) {
	email := "x@example.com"
	tests := []struct {
		name     string
		user     *User
		password string
	}{
		{"pending user cannot log in", &User{ID: "u", Email: &email, PasswordHash: "hash:Right1pass", Status: UserStatusPending}, "Right1pass"},
		{"wrong password", &User{ID: "u", Email: &email, PasswordHash: "hash:Right1pass", Status: UserStatusActive}, "Wrong1pass"},
		{"unknown email", nil, "Whatever1"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newAuthHarness(t)
			h.users.findByEmail = func(_ context.Context, _ string) (*User, error) {
				return tc.user, nil
			}
			_, _, err := h.service.Login(context.Background(), LoginInput{Email: email, Password: tc.password})
			requireAppError(t, err, "UNAUTHENTICATED", http.StatusUnauthorized)
			if len(h.refreshTokens.created) != 0 {
				t.Fatal("no refresh token may be issued for a failed login")
			}
		})
	}
}

func TestResendActivation_IgnoresNonPendingUsers(t *testing.T) {
	h := newAuthHarness(t)
	email := "x@example.com"

	h.users.findByEmail = func(_ context.Context, e string) (*User, error) {
		return &User{ID: "u", Email: &email, Status: UserStatusActive}, nil
	}

	if err := h.service.ResendActivation(context.Background(), email); err != nil {
		t.Fatalf("resend activation: %v", err)
	}
	if len(h.activations.created) != 0 || len(h.mailer.sent) != 0 {
		t.Fatal("no token or email expected for a non-pending user")
	}
}

func TestResendActivation_RateLimitsRecentTokens(t *testing.T) {
	h := newAuthHarness(t)
	email := "x@example.com"

	h.users.findByEmail = func(_ context.Context, e string) (*User, error) {
		return &User{ID: "u", Email: &email, Status: UserStatusPending}, nil
	}
	h.activations.findLatestByUserID = func(_ context.Context, _ string) (*AccountActivationToken, error) {
		return &AccountActivationToken{
			ID: "t", UserID: "u",
			CreatedAt: time.Now().Add(-time.Minute),
			ExpiresAt: time.Now().Add(6 * 24 * time.Hour),
		}, nil
	}

	err := h.service.ResendActivation(context.Background(), email)
	requireAppError(t, err, "RATE_LIMITED", http.StatusTooManyRequests)
	if len(h.mailer.sent) != 0 {
		t.Fatal("no email expected for a rate-limited resend")
	}
}

func TestResendActivation_IssuesNewTokenAfterCooldown(t *testing.T) {
	h := newAuthHarness(t)
	email := "x@example.com"
	now := time.Now()

	h.users.findByEmail = func(_ context.Context, e string) (*User, error) {
		return &User{ID: "u", Email: &email, Status: UserStatusPending, Profile: &Profile{FullName: "Resend Me"}}, nil
	}
	h.activations.findLatestByUserID = func(_ context.Context, _ string) (*AccountActivationToken, error) {
		return &AccountActivationToken{ID: "t-old", UserID: "u", UsedAt: &now, CreatedAt: now.Add(-10 * time.Minute)}, nil
	}

	if err := h.service.ResendActivation(context.Background(), email); err != nil {
		t.Fatalf("resend activation: %v", err)
	}
	if len(h.activations.revokedUnused) != 1 {
		t.Fatalf("old unused tokens not revoked: %v", h.activations.revokedUnused)
	}
	if len(h.activations.created) != 1 || h.activations.created[0].UserID != "u" {
		t.Fatalf("expected one new activation token: %+v", h.activations.created)
	}
	if len(h.mailer.sent) != 1 || h.mailer.sent[0].to != email || h.mailer.sent[0].name != "Resend Me" {
		t.Fatalf("unexpected email: %+v", h.mailer.sent)
	}
}

func TestResendActivation_InvalidatesTokenWhenEmailFails(t *testing.T) {
	h := newAuthHarness(t)
	email := "x@example.com"

	h.users.findByEmail = func(_ context.Context, e string) (*User, error) {
		return &User{ID: "u", Email: &email, Status: UserStatusPending}, nil
	}
	h.mailer.err = errors.New("resend is down")

	if err := h.service.ResendActivation(context.Background(), email); err == nil {
		t.Fatal("expected error when email send fails")
	}
	if len(h.activations.created) != 1 {
		t.Fatalf("expected an activation token, got %d", len(h.activations.created))
	}
	if len(h.activations.markedUsed) != 1 || h.activations.markedUsed[0] != h.activations.created[0].ID {
		t.Fatalf("failed email must invalidate the issued token, marked used: %v", h.activations.markedUsed)
	}
}

func TestGenerateUsername_SanitizesFullName(t *testing.T) {
	tests := []struct {
		name     string
		wantBase string
	}{
		{"Mohan Kumar", "mohan.kumar"},
		{"Priya-Kumari", "priyakumari"},
		{"ABC", "abc"},
		{"x", "x"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := generateUsername(tc.name)
			if !strings.HasPrefix(got, tc.wantBase) {
				t.Fatalf("generateUsername(%q) = %q, want prefix %q", tc.name, got, tc.wantBase)
			}
			suffix := strings.TrimPrefix(got, tc.wantBase)
			if suffix == "" {
				t.Fatalf("expected a numeric suffix, got none in %q", got)
			}
			for _, r := range suffix {
				if r < '0' || r > '9' {
					t.Fatalf("suffix %q in %q is not numeric", suffix, got)
				}
			}
		})
	}
}

func TestGenerateUsername_CapsBaseAt30Chars(t *testing.T) {
	got := generateUsername(strings.Repeat("a", 60))
	if !strings.HasPrefix(got, strings.Repeat("a", 30)) {
		t.Fatalf("expected 30-char base, got %q", got)
	}
	if len(got) <= 30 || len(got) > 34 {
		t.Fatalf("expected 30-char base plus 1-4 digit suffix, got %d chars: %q", len(got), got)
	}
}
