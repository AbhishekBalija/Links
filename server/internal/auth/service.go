package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/AbhishekBalija/Links/server/internal/mailer"
	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
)

type authService struct {
	userRepo        UserRepository
	refreshRepo     RefreshTokenRepository
	activationRepo  ActivationTokenRepository
	auditLogRepo    AuditLogRepository
	tokenCfg        TokenConfig
	passwordHasher  PasswordHasher
	mailer          mailer.Mailer
	frontendURL     string
}

func NewAuthService(
	userRepo UserRepository,
	refreshRepo RefreshTokenRepository,
	activationRepo ActivationTokenRepository,
	auditLogRepo AuditLogRepository,
	tokenCfg TokenConfig,
	passwordHasher PasswordHasher,
	mailer mailer.Mailer,
	frontendURL string,
) AuthService {
	return &authService{
		userRepo:        userRepo,
		refreshRepo:     refreshRepo,
		activationRepo:  activationRepo,
		auditLogRepo:    auditLogRepo,
		tokenCfg:        tokenCfg,
		passwordHasher:  passwordHasher,
		mailer:          mailer,
		frontendURL:     frontendURL,
	}
}

func (s *authService) RequestAccess(ctx context.Context, input RequestAccessInput) (*RequestAccessResponse, error) {
	existing, err := s.userRepo.FindByEmail(ctx, input.Email)
	if err != nil {
		return nil, fmt.Errorf("check email: %w", err)
	}
	if existing != nil {
		return nil, apperrors.NewConflict("email already registered")
	}

	passwordHash, err := s.passwordHasher.Hash(input.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	var deptID string
	if input.DepartmentCode != "" {
		dept, err := s.userRepo.FindDepartmentByCode(ctx, input.DepartmentCode)
		if err != nil {
			return nil, fmt.Errorf("find department: %w", err)
		}
		if dept == nil {
			return nil, apperrors.NewValidation("invalid department code", nil)
		}
		deptID = dept.ID
	}

	user := &User{
		Email:        &input.Email,
		PasswordHash: passwordHash,
		Status:       UserStatusPending,
		IsVerified:   false,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	profile := &Profile{
		UserID:    user.ID,
		FullName:  input.FullName,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.createProfileWithRetry(ctx, profile); err != nil {
		return nil, fmt.Errorf("create profile: %w", err)
	}

	if input.USN != "" {
		usnCode, err := ValidateUSN(input.USN)
		if err != nil {
			return nil, apperrors.NewValidation("invalid USN: "+err.Error(), nil)
		}
		if deptID == "" {
			dept, err := s.userRepo.FindDepartmentByCode(ctx, usnCode)
			if err != nil {
				return nil, fmt.Errorf("find department from USN: %w", err)
			}
			if dept == nil {
				return nil, apperrors.NewValidation("department code "+usnCode+" from USN not found in system; contact admin", nil)
			}
			deptID = dept.ID
		}
		identity := &StudentIdentity{
			UserID:       user.ID,
			USN:          input.USN,
			DepartmentID: deptID,
			BatchYear:    0,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		if input.BatchYear != nil {
			identity.BatchYear = *input.BatchYear
		}
		if err := s.userRepo.CreateStudentIdentity(ctx, identity); err != nil {
			return nil, fmt.Errorf("create student identity: %w", err)
		}
	}

	if err := s.sendActivationEmail(ctx, user.ID, input.Email, profile.FullName); err != nil {
		return nil, fmt.Errorf("send activation email: %w", err)
	}

	return &RequestAccessResponse{
		UserID: user.ID,
		Status: string(UserStatusPending),
	}, nil
}

func (s *authService) Login(ctx context.Context, input LoginInput) (*LoginResponse, string, error) {
	user, err := s.userRepo.FindByEmail(ctx, input.Email)
	if err != nil {
		return nil, "", fmt.Errorf("find user: %w", err)
	}
	if user == nil {
		return nil, "", apperrors.NewUnauthenticated("invalid credentials")
	}

	if !user.Status.CanLogin() {
		return nil, "", apperrors.NewUnauthenticated("account not active")
	}

	ok, err := s.passwordHasher.Verify(input.Password, user.PasswordHash)
	if err != nil {
		return nil, "", fmt.Errorf("verify password: %w", err)
	}
	if !ok {
		return nil, "", apperrors.NewUnauthenticated("invalid credentials")
	}

	roles, err := s.userRepo.GetRoleAssignments(ctx, user.ID)
	if err != nil {
		return nil, "", fmt.Errorf("get roles: %w", err)
	}
	roleNames := make([]string, len(roles))
	for i, r := range roles {
		roleNames[i] = string(r.Role)
	}

	accessToken, err := GenerateAccessToken(user.ID, roleNames, s.tokenCfg)
	if err != nil {
		return nil, "", fmt.Errorf("generate access token: %w", err)
	}

	refreshRaw, err := GenerateRefreshTokenRaw(s.tokenCfg)
	if err != nil {
		return nil, "", fmt.Errorf("generate refresh token: %w", err)
	}

	refreshToken := &RefreshToken{
		UserID:    user.ID,
		TokenHash: refreshRaw.TokenHash,
		ExpiresAt: refreshRaw.ExpiresAt,
		CreatedAt: time.Now(),
	}
	if err := s.refreshRepo.Create(ctx, refreshToken); err != nil {
		return nil, "", fmt.Errorf("store refresh token: %w", err)
	}

	return &LoginResponse{
		AccessToken: accessToken,
		ExpiresIn:   int(s.tokenCfg.AccessTTL.Seconds()),
	}, refreshRaw.Token, nil
}

func (s *authService) Refresh(ctx context.Context, refreshTokenRaw string) (*RefreshResponse, string, error) {
	hash := HashRefreshToken(refreshTokenRaw)

	stored, err := s.refreshRepo.FindByHash(ctx, hash)
	if err != nil {
		return nil, "", fmt.Errorf("find refresh token: %w", err)
	}
	if stored == nil {
		return nil, "", apperrors.NewUnauthenticated("invalid refresh token")
	}

	if stored.IsExpired() {
		return nil, "", apperrors.NewUnauthenticated("refresh token expired")
	}
	if stored.IsRevoked() {
		return nil, "", apperrors.NewUnauthenticated("refresh token revoked")
	}

	user, err := s.userRepo.FindByID(ctx, stored.UserID)
	if err != nil {
		return nil, "", fmt.Errorf("find user: %w", err)
	}
	if user == nil || !user.Status.CanLogin() {
		return nil, "", apperrors.NewUnauthenticated("user not active")
	}

	roles, err := s.userRepo.GetRoleAssignments(ctx, user.ID)
	if err != nil {
		return nil, "", fmt.Errorf("get roles: %w", err)
	}
	roleNames := make([]string, len(roles))
	for i, r := range roles {
		roleNames[i] = string(r.Role)
	}

	if err := s.refreshRepo.RevokeByHash(ctx, hash); err != nil {
		return nil, "", fmt.Errorf("revoke old token: %w", err)
	}

	accessToken, err := GenerateAccessToken(user.ID, roleNames, s.tokenCfg)
	if err != nil {
		return nil, "", fmt.Errorf("generate access token: %w", err)
	}

	refreshRaw, err := GenerateRefreshTokenRaw(s.tokenCfg)
	if err != nil {
		return nil, "", fmt.Errorf("generate refresh token: %w", err)
	}

	newRefreshToken := &RefreshToken{
		UserID:    user.ID,
		TokenHash: refreshRaw.TokenHash,
		ExpiresAt: refreshRaw.ExpiresAt,
		CreatedAt: time.Now(),
	}
	if err := s.refreshRepo.Create(ctx, newRefreshToken); err != nil {
		return nil, "", fmt.Errorf("store refresh token: %w", err)
	}

	return &RefreshResponse{
		AccessToken: accessToken,
		ExpiresIn:   int(s.tokenCfg.AccessTTL.Seconds()),
	}, refreshRaw.Token, nil
}

func (s *authService) Logout(ctx context.Context, refreshTokenRaw string) error {
	hash := HashRefreshToken(refreshTokenRaw)
	return s.refreshRepo.RevokeByHash(ctx, hash)
}

func (s *authService) ActivateAccount(ctx context.Context, token, password string) error {
	hash := HashRefreshToken(token)

	activation, err := s.activationRepo.FindByHash(ctx, hash)
	if err != nil {
		return fmt.Errorf("find activation token: %w", err)
	}
	if activation == nil || activation.UsedAt != nil || time.Now().After(activation.ExpiresAt) {
		return apperrors.NewUnauthenticated("invalid or expired activation token")
	}

	passwordHash, err := s.passwordHasher.Hash(password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	user, err := s.userRepo.FindByID(ctx, activation.UserID)
	if err != nil {
		return fmt.Errorf("find user: %w", err)
	}
	if user == nil {
		return apperrors.NewNotFound("user not found")
	}

	user.PasswordHash = passwordHash
	user.Status = UserStatusActive
	user.IsVerified = true
	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	if err := s.activationRepo.MarkUsed(ctx, activation.ID); err != nil {
		return fmt.Errorf("mark token used: %w", err)
	}

	return nil
}

func (s *authService) ResendActivation(ctx context.Context, email string) error {
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("find user: %w", err)
	}
	if user == nil || user.Status != UserStatusPending {
		return nil
	}

	last, err := s.activationRepo.FindLatestByUserID(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("find latest token: %w", err)
	}
	if last != nil && last.UsedAt == nil && time.Since(last.CreatedAt) < 5*time.Minute {
		return apperrors.NewRateLimited("activation email was sent recently; try again later")
	}

	if err := s.activationRepo.RevokeAllUnusedByUserID(ctx, user.ID); err != nil {
		return fmt.Errorf("revoke old tokens: %w", err)
	}

	fullName := ""
	if user.Profile != nil {
		fullName = user.Profile.FullName
	}

	if err := s.sendActivationEmail(ctx, user.ID, email, fullName); err != nil {
		return fmt.Errorf("send activation email: %w", err)
	}

	return nil
}

func (s *authService) sendActivationEmail(ctx context.Context, userID, email, name string) error {
	tokenRaw, tokenHash, err := generateActivationToken()
	if err != nil {
		return fmt.Errorf("generate token: %w", err)
	}

	token := &AccountActivationToken{
		ID:        uuid.New().String(),
		UserID:    userID,
		TokenHash: tokenHash,
		Purpose:   "activate",
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		CreatedAt: time.Now(),
	}
	if err := s.activationRepo.Create(ctx, token); err != nil {
		return fmt.Errorf("create activation token: %w", err)
	}

	activationLink := s.frontendURL + "/activate?token=" + tokenRaw
	if err := s.mailer.SendActivationEmail(email, name, activationLink); err != nil {
		return fmt.Errorf("send email: %w", err)
	}

	return nil
}

func generateActivationToken() (raw string, hash string, err error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", fmt.Errorf("generate random bytes: %w", err)
	}
	raw = base64.RawURLEncoding.EncodeToString(bytes)
	hash = HashRefreshToken(raw)
	return raw, hash, nil
}

func (s *authService) GetMe(ctx context.Context, userID string) (*MeResponse, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}
	if user == nil {
		return nil, apperrors.NewNotFound("user not found")
	}

	roles, err := s.userRepo.GetRoleAssignments(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get roles: %w", err)
	}
	roleNames := make([]string, len(roles))
	for i, r := range roles {
		roleNames[i] = string(r.Role)
	}

	resp := &MeResponse{
		UserID: user.ID,
		Email:  user.Email,
		Phone:  user.Phone,
		Roles:  roleNames,
	}

	if user.Profile != nil {
		resp.Profile = &MeProfileResponse{
			UserID:               user.Profile.UserID,
			FullName:             user.Profile.FullName,
			Username:             user.Profile.Username,
			Headline:             user.Profile.Headline,
			Bio:                  user.Profile.Bio,
			AvatarURL:            user.Profile.AvatarURL,
			PublicProfileEnabled: user.Profile.PublicProfileEnabled,
			ShowEmail:            user.Profile.ShowEmail,
			ShowPhone:            user.Profile.ShowPhone,
			LinkedInURL:          user.Profile.LinkedInURL,
			GitHubURL:            user.Profile.GitHubURL,
			PortfolioURL:         user.Profile.PortfolioURL,
		}
	}

	if user.StudentIdentity != nil {
		var rn *string
		if user.StudentIdentity.RollNumber != "" {
			rn = &user.StudentIdentity.RollNumber
		}
		resp.StudentIdentity = &MeStudentIdentityResponse{
			USN:        user.StudentIdentity.USN,
			BatchYear:  user.StudentIdentity.BatchYear,
			RollNumber: rn,
		}
	}

	return resp, nil
}

func (s *authService) ReviewQueue(ctx context.Context) (*ReviewQueueResponse, error) {
	users, err := s.userRepo.FindPendingUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("find pending users: %w", err)
	}

	responses := make([]PendingUserResponse, 0, len(users))
	for _, u := range users {
		pur := PendingUserResponse{
			ID:        u.ID,
			Email:     u.Email,
			CreatedAt: u.CreatedAt,
		}
		if u.Profile != nil {
			pur.Profile = &PendingUserProfile{
				FullName: u.Profile.FullName,
				Username: u.Profile.Username,
			}
		}
		if u.StudentIdentity != nil {
			pur.StudentIdentity = &PendingUserStudentID{
				USN:       u.StudentIdentity.USN,
				BatchYear: u.StudentIdentity.BatchYear,
			}
		}
		responses = append(responses, pur)
	}

	return &ReviewQueueResponse{Users: responses, Total: len(responses)}, nil
}

func (s *authService) VerifyUser(ctx context.Context, actorID, userID, scopeType, scopeID, note string) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("find user: %w", err)
	}
	if user == nil {
		return apperrors.NewNotFound("user not found")
	}
	if user.Status != UserStatusPending {
		return apperrors.NewConflict("user is not in pending status")
	}

	role := RoleStudent
	now := time.Now()
	st := ScopeType(scopeType)
	if st == "" {
		st = ScopeGlobal
	}
	ra := &RoleAssignment{
		UserID:     userID,
		Role:       role,
		ScopeType:  st,
		ScopeID:    stringPtrOrNil(scopeID),
		AssignedBy: &actorID,
		StartsAt:   now,
		CreatedAt:  now,
	}
	if err := s.userRepo.CreateRoleAssignment(ctx, ra); err != nil {
		return fmt.Errorf("create role assignment: %w", err)
	}

	user.Status = UserStatusActive
	user.IsVerified = true
	user.UpdatedAt = now
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	auditLog := &AuditLog{
		ActorID:      &actorID,
		Action:       "user_verified",
		ResourceType: "user",
		ResourceID:   &userID,
		CreatedAt:    now,
	}
	if note != "" {
		auditLog.Metadata = map[string]string{"note": note}
	}
	if err := s.auditLogRepo.Create(ctx, auditLog); err != nil {
		return fmt.Errorf("create audit log: %w", err)
	}

	return nil
}

func (s *authService) UpdateUserStatus(ctx context.Context, actorID, userID, status, note string) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("find user: %w", err)
	}
	if user == nil {
		return apperrors.NewNotFound("user not found")
	}

	newStatus := UserStatus(status)
	if newStatus == user.Status {
		return apperrors.NewConflict("user already has status " + status)
	}

	if newStatus == UserStatusActive && user.Status == UserStatusRejected {
		return apperrors.NewValidation("cannot activate a rejected user", nil)
	}

	user.Status = newStatus
	user.UpdatedAt = time.Now()
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	auditLog := &AuditLog{
		ActorID:      &actorID,
		Action:       "user_status_changed",
		ResourceType: "user",
		ResourceID:   &userID,
		CreatedAt:    time.Now(),
		Metadata:     map[string]string{"new_status": status, "note": note},
	}
	if err := s.auditLogRepo.Create(ctx, auditLog); err != nil {
		return fmt.Errorf("create audit log: %w", err)
	}

	return nil
}

func stringPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (s *authService) createProfileWithRetry(ctx context.Context, profile *Profile) error {
	const maxAttempts = 5
	for range maxAttempts {
		profile.Username = generateUsername(profile.FullName)
		err := s.userRepo.CreateProfile(ctx, profile)
		if err == nil {
			return nil
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			continue
		}
		return err
	}
	return fmt.Errorf("username generation failed after %d attempts", maxAttempts)
}

// generateUsername creates a unique-ish username from the user's full name.
// TODO: This auto-generate approach will be replaced by a user-chosen username field
// with a real-time availability check in a later phase. When that ships, delete this
// function and the corresponding retry wrapper in createProfileWithRetry.
func generateUsername(fullName string) string {
	base := ""
	for _, r := range fullName {
		if r == ' ' {
			base += "."
		} else if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			if r >= 'A' && r <= 'Z' {
				r += 32
			}
			base += string(r)
		}
	}
	if len(base) > 30 {
		base = base[:30]
	}
	return base + fmt.Sprintf("%d", time.Now().UnixMilli()%10000)
}