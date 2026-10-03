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

	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
)

// Fakes for the auth service dependencies. All function fields default to
// inert behavior (nil/empty results, no-op writes) so each test only stubs
// the calls the path under test actually makes.

type fakeUserRepo struct {
	findByEmail            func(ctx context.Context, email string) (*User, error)
	findByID               func(ctx context.Context, id string) (*User, error)
	findByIDForUpdate      func(ctx context.Context, id string) (*User, error)
	findPendingUsers       func(ctx context.Context) ([]User, error)
	findEmailByUserID      func(ctx context.Context, userID string) (*string, error)
	findPhoneByUserID      func(ctx context.Context, userID string) (*string, error)
	findDepartmentByCode   func(ctx context.Context, code string) (*Department, error)
	lockDepartmentForShare func(ctx context.Context, id string) (bool, error)
	create                 func(ctx context.Context, user *User) error
	update                 func(ctx context.Context, user *User) error
	updateStatus           func(ctx context.Context, id string, status UserStatus) error
	createProfile          func(ctx context.Context, profile *Profile) error
	createStudentIdentity  func(ctx context.Context, identity *StudentIdentity) error
	getRoleAssignments     func(ctx context.Context, userID string) ([]RoleAssignment, error)
	createRoleAssignment   func(ctx context.Context, ra *RoleAssignment) error

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

func (f *fakeUserRepo) FindByEmailForUpdate(ctx context.Context, email string) (*User, error) {
	return f.FindByEmail(ctx, email)
}

func (f *fakeUserRepo) FindByGoogleSubjectForUpdate(context.Context, string) (*User, error) {
	return nil, nil
}

func (f *fakeUserRepo) SetGoogleSubject(context.Context, string, string) error { return nil }

func (f *fakeUserRepo) CompleteFirstSignIn(context.Context, string, time.Time) error { return nil }

func (f *fakeUserRepo) ReturnForReview(context.Context, string) error { return nil }

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

func (f *fakeUserRepo) LockDepartmentForShare(ctx context.Context, id string) (bool, error) {
	if f.lockDepartmentForShare != nil {
		return f.lockDepartmentForShare(ctx, id)
	}
	return true, nil
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
	// The admin the tests act as: deciding Access requests checks the actor's
	// roles (ADR 0025).
	if userID == "admin-1" {
		return []RoleAssignment{{UserID: userID, Role: RoleAdmin, ScopeType: ScopeGlobal}}, nil
	}
	return nil, nil
}

func (f *fakeUserRepo) HasAdmin(context.Context) (bool, error) { return false, nil }

func (f *fakeUserRepo) ReviewDepartments(context.Context) ([]ReviewDepartment, error) {
	return nil, nil
}

func (f *fakeUserRepo) ReportedAt(context.Context, []string) (map[string]time.Time, error) {
	return map[string]time.Time{}, nil
}

func (f *fakeUserRepo) WaitingForFirstSignIn(context.Context, bool, []string, int) (WaitingList, error) {
	return WaitingList{}, nil
}
func (f *fakeUserRepo) NotSignedIn(context.Context, NotSignedInFilter) (WaitingList, error) {
	return WaitingList{}, nil
}
func (f *fakeUserRepo) NotSignedInEmails(context.Context, NotSignedInFilter) ([]string, error) {
	return nil, nil
}

func (f *fakeUserRepo) RecentImportAudits(context.Context, bool, []string, int) ([]ImportAuditRow, map[string]bool, error) {
	return nil, nil, nil
}

func (f *fakeUserRepo) CreateRoleAssignment(ctx context.Context, ra *RoleAssignment) error {
	if f.createRoleAssignment != nil {
		return f.createRoleAssignment(ctx, ra)
	}
	f.createdRoleAssignments = append(f.createdRoleAssignments, ra)
	return nil
}

// Role management is covered by the API tests; these stubs only satisfy the interface.
func (f *fakeUserRepo) FindDepartmentByID(context.Context, string) (*Department, error) {
	return nil, nil
}
func (f *fakeUserRepo) ListRoleAssignments(context.Context, string) ([]RoleAssignmentView, error) {
	return nil, nil
}
func (f *fakeUserRepo) FindRoleAssignmentForUpdate(context.Context, string, string) (*RoleAssignment, error) {
	return nil, nil
}
func (f *fakeUserRepo) HasOverlappingAssignment(context.Context, OverlapFilter) (bool, error) {
	return false, nil
}
func (f *fakeUserRepo) LockDepartmentForUpdate(context.Context, string) (bool, error) {
	return false, nil
}
func (f *fakeUserRepo) LockAdminAssignmentsInEffect(context.Context) ([]RoleAssignment, error) {
	return nil, nil
}
func (f *fakeUserRepo) USNExists(context.Context, string) (bool, error)            { return false, nil }
func (f *fakeUserRepo) EndRoleAssignment(context.Context, string, time.Time) error { return nil }
func (f *fakeUserRepo) ClearDepartmentHOD(context.Context, string, string) error   { return nil }

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
	auditLogs     *fakeAuditLogRepo
}

func (u *fakeUnitOfWork) WithinTransaction(ctx context.Context, fn func(AuthRepositories) error) error {
	return fn(AuthRepositories{
		Users:         u.users,
		RefreshTokens: u.refreshTokens,
		AuditLogs:     u.auditLogs,
	})
}

type fakeMailer struct{}

func (fakeMailer) SendSignInCode(_, _ string) error { return nil }

type authHarness struct {
	service       AuthService
	users         *fakeUserRepo
	refreshTokens *fakeRefreshTokenRepo
	auditLogs     *fakeAuditLogRepo
	cfg           TokenConfig
}

func newAuthHarness(t *testing.T) *authHarness {
	t.Helper()
	users := &fakeUserRepo{}
	refreshTokens := &fakeRefreshTokenRepo{}
	auditLogs := &fakeAuditLogRepo{}
	uow := &fakeUnitOfWork{
		users:         users,
		refreshTokens: refreshTokens,
		auditLogs:     auditLogs,
	}
	cfg := TokenConfig{
		AccessSecret:  "newAuthHarness-access-secret",
		RefreshSecret: "newAuthHarness-refresh-secret",
		AccessTTL:     15 * time.Minute,
		RefreshTTL:    7 * 24 * time.Hour,
	}
	service := NewAuthService(users, refreshTokens, uow, cfg, fakeMailer{}, DefaultCodeSettings(), nil)
	return &authHarness{
		service:       service,
		users:         users,
		refreshTokens: refreshTokens,
		auditLogs:     auditLogs,
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

func TestVerifyUser_ApprovesPendingStudentToWaitForFirstSignIn(t *testing.T) {
	h := newAuthHarness(t)
	ctx := context.Background()
	userID, actorID := "user-1", "admin-1"
	email := "student@example.com"

	h.users.findByIDForUpdate = func(_ context.Context, id string) (*User, error) {
		return &User{
			ID:      id,
			Email:   &email,
			Status:  UserStatusPending,
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
	if h.users.updatedUsers[0].Status != UserStatusPending {
		t.Fatalf("status = %q, want pending until the first sign-in", h.users.updatedUsers[0].Status)
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
		})
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
