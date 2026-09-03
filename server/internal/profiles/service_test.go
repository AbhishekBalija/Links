package profiles

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/AbhishekBalija/Links/server/internal/auth"
	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
)

type fakeProfileRepo struct {
	findByUserID       func(ctx context.Context, userID string) (*Profile, error)
	findByUsername     func(ctx context.Context, username string) (*Profile, error)
	update             func(ctx context.Context, profile *Profile) error
	findByUserIDCalls  int
	findByUsernameCalls int
	updated             []*Profile
}

func (f *fakeProfileRepo) FindByUserID(ctx context.Context, userID string) (*Profile, error) {
	f.findByUserIDCalls++
	if f.findByUserID != nil {
		return f.findByUserID(ctx, userID)
	}
	return nil, nil
}

func (f *fakeProfileRepo) FindByUsername(ctx context.Context, username string) (*Profile, error) {
	f.findByUsernameCalls++
	if f.findByUsername != nil {
		return f.findByUsername(ctx, username)
	}
	return nil, nil
}

func (f *fakeProfileRepo) Update(ctx context.Context, profile *Profile) error {
	if f.update != nil {
		return f.update(ctx, profile)
	}
	f.updated = append(f.updated, profile)
	return nil
}

type fakeUserReader struct {
	email      *string
	phone      *string
	err        error
	emailCalls int
	phoneCalls int
}

func (f *fakeUserReader) FindEmailByUserID(_ context.Context, _ string) (*string, error) {
	f.emailCalls++
	if f.err != nil {
		return nil, f.err
	}
	return f.email, nil
}

func (f *fakeUserReader) FindPhoneByUserID(_ context.Context, _ string) (*string, error) {
	f.phoneCalls++
	if f.err != nil {
		return nil, f.err
	}
	return f.phone, nil
}

type fakeAuditLogRepo struct {
	created []*auth.AuditLog
}

func (f *fakeAuditLogRepo) Create(_ context.Context, log *auth.AuditLog) error {
	f.created = append(f.created, log)
	return nil
}

type fakeProfileUnitOfWork struct {
	profiles  *fakeProfileRepo
	auditLogs *fakeAuditLogRepo
}

func (u *fakeProfileUnitOfWork) WithinTransaction(ctx context.Context, fn func(Repositories) error) error {
	return fn(Repositories{Profiles: u.profiles, AuditLogs: u.auditLogs})
}

type profileHarness struct {
	service  *Service
	repo     *fakeProfileRepo
	reader   *fakeUserReader
	auditLogs *fakeAuditLogRepo
}

func newProfileHarness() *profileHarness {
	repo := &fakeProfileRepo{}
	reader := &fakeUserReader{}
	auditLogs := &fakeAuditLogRepo{}
	uow := &fakeProfileUnitOfWork{profiles: repo, auditLogs: auditLogs}
	return &profileHarness{
		service:   NewService(repo, reader, uow),
		repo:      repo,
		reader:    reader,
		auditLogs: auditLogs,
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

func testProfile(userID, username string) *Profile {
	return &Profile{
		UserID:               userID,
		Username:             username,
		FullName:             "Test User",
		PublicProfileEnabled: true,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}
}

func TestGetPublicProfile_HidesPrivateProfilesFromOthers(t *testing.T) {
	tests := []struct {
		name     string
		viewerID *string
	}{
		{"anonymous viewer", nil},
		{"non-owner viewer", ptrTo("other-user")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newProfileHarness()
			profile := testProfile("u-1", "owner")
			profile.PublicProfileEnabled = false
			h.repo.findByUsername = func(_ context.Context, _ string) (*Profile, error) {
				return profile, nil
			}

			_, err := h.service.GetPublicProfile(context.Background(), "owner", tc.viewerID)
			requireAppError(t, err, "NOT_FOUND", http.StatusNotFound)
			if h.reader.emailCalls != 0 || h.reader.phoneCalls != 0 {
				t.Fatal("hidden profile must not leak owner contact details")
			}
		})
	}
}

func TestGetPublicProfile_MissingProfileIsNotFound(t *testing.T) {
	h := newProfileHarness()
	_, err := h.service.GetPublicProfile(context.Background(), "nobody", nil)
	requireAppError(t, err, "NOT_FOUND", http.StatusNotFound)
}

func TestGetPublicProfile_OwnerCanViewPrivateProfile(t *testing.T) {
	h := newProfileHarness()
	email, phone := "owner@example.com", "+1-555-0100"
	h.reader.email, h.reader.phone = ptrTo(email), ptrTo(phone)
	profile := testProfile("u-1", "owner")
	profile.PublicProfileEnabled = false
	h.repo.findByUsername = func(_ context.Context, _ string) (*Profile, error) {
		return profile, nil
	}

	resp, err := h.service.GetPublicProfile(context.Background(), "owner", ptrTo("u-1"))
	if err != nil {
		t.Fatalf("get public profile: %v", err)
	}
	if resp.Email == nil || *resp.Email != email {
		t.Fatalf("owner should see own email, got %v", resp.Email)
	}
	if resp.Phone == nil || *resp.Phone != phone {
		t.Fatalf("owner should see own phone, got %v", resp.Phone)
	}
}

func TestGetPublicProfile_OnlyExposesOptedInContactFields(t *testing.T) {
	h := newProfileHarness()
	email, phone := "owner@example.com", "+1-555-0100"
	h.reader.email, h.reader.phone = ptrTo(email), ptrTo(phone)
	profile := testProfile("u-1", "public-user")
	profile.ShowEmail = true // phone intentionally not opted in
	h.repo.findByUsername = func(_ context.Context, _ string) (*Profile, error) {
		return profile, nil
	}

	resp, err := h.service.GetPublicProfile(context.Background(), "public-user", nil)
	if err != nil {
		t.Fatalf("get public profile: %v", err)
	}
	if resp.Email == nil || *resp.Email != email {
		t.Fatalf("email should be exposed when show_email is true, got %v", resp.Email)
	}
	if resp.Phone != nil {
		t.Fatalf("phone leaked without show_phone, got %v", *resp.Phone)
	}
	if h.reader.phoneCalls != 0 {
		t.Fatal("phone must not be fetched when not opted in")
	}
}

func TestUpdateMyProfile_NotFound(t *testing.T) {
	h := newProfileHarness()
	_, err := h.service.UpdateMyProfile(context.Background(), "u-1", UpdateProfileInput{})
	requireAppError(t, err, "NOT_FOUND", http.StatusNotFound)
}

func TestUpdateMyProfile_RejectsInvalidURLBeforeWrite(t *testing.T) {
	h := newProfileHarness()
	bad := "javascript:alert(1)"
	h.repo.findByUserID = func(_ context.Context, _ string) (*Profile, error) {
		t.Fatal("profile lookup must not happen for invalid input")
		return nil, nil
	}

	_, err := h.service.UpdateMyProfile(context.Background(), "u-1", UpdateProfileInput{PortfolioURL: &bad})
	requireAppError(t, err, "VALIDATION_ERROR", http.StatusBadRequest)
	if len(h.auditLogs.created) != 0 {
		t.Fatal("no audit log expected for a rejected update")
	}
}

func TestUpdateMyProfile_AuditsPrivacyChanges(t *testing.T) {
	h := newProfileHarness()
	email := "owner@example.com"
	h.reader.email = &email
	profile := testProfile("u-1", "owner")
	profile.ShowEmail = true
	h.repo.findByUserID = func(_ context.Context, _ string) (*Profile, error) {
		return profile, nil
	}

	showEmail, showPhone := false, true
	resp, err := h.service.UpdateMyProfile(context.Background(), "u-1", UpdateProfileInput{
		ShowEmail: &showEmail,
		ShowPhone: &showPhone,
	})
	if err != nil {
		t.Fatalf("update profile: %v", err)
	}
	if resp.ShowEmail || !resp.ShowPhone {
		t.Fatalf("profile fields not applied: show_email=%v show_phone=%v", resp.ShowEmail, resp.ShowPhone)
	}
	if len(h.auditLogs.created) != 1 {
		t.Fatalf("expected one privacy audit log, got %d", len(h.auditLogs.created))
	}
	log := h.auditLogs.created[0]
	if log.Action != "profile_privacy_updated" || log.ResourceType != "profile" {
		t.Fatalf("unexpected audit log: %+v", log)
	}
	meta, ok := log.Metadata.(map[string]bool)
	if !ok {
		t.Fatalf("privacy audit metadata has unexpected type %T", log.Metadata)
	}
	if meta["show_email"] || !meta["show_phone"] {
		t.Fatalf("unexpected privacy audit metadata: %v", meta)
	}
}

func TestUpdateMyProfile_SkipsAuditWhenPrivacyUnchanged(t *testing.T) {
	h := newProfileHarness()
	email := "owner@example.com"
	h.reader.email = &email
	profile := testProfile("u-1", "owner")
	profile.ShowEmail = false
	profile.ShowPhone = false
	h.repo.findByUserID = func(_ context.Context, _ string) (*Profile, error) {
		return profile, nil
	}

	showEmail := false
	headline := "New headline"
	resp, err := h.service.UpdateMyProfile(context.Background(), "u-1", UpdateProfileInput{
		ShowEmail: &showEmail,
		Headline:  &headline,
	})
	if err != nil {
		t.Fatalf("update profile: %v", err)
	}
	if resp.Headline == nil || *resp.Headline != headline {
		t.Fatalf("headline not applied: %+v", resp.Headline)
	}
	if len(h.auditLogs.created) != 0 {
		t.Fatalf("no audit log expected when privacy settings are unchanged, got %d", len(h.auditLogs.created))
	}
}
