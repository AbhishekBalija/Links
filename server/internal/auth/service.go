package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/AbhishekBalija/Links/server/internal/shared/errors"
)

type authService struct {
	userRepo        UserRepository
	refreshRepo     RefreshTokenRepository
	activationRepo  ActivationTokenRepository
	tokenCfg        TokenConfig
	passwordHasher  PasswordHasher
}

func NewAuthService(
	userRepo UserRepository,
	refreshRepo RefreshTokenRepository,
	activationRepo ActivationTokenRepository,
	tokenCfg TokenConfig,
	passwordHasher PasswordHasher,
) AuthService {
	return &authService{
		userRepo:        userRepo,
		refreshRepo:     refreshRepo,
		activationRepo:  activationRepo,
		tokenCfg:        tokenCfg,
		passwordHasher:  passwordHasher,
	}
}

func (s *authService) RequestAccess(ctx context.Context, input RequestAccessInput) (*RequestAccessResponse, error) {
	existing, err := s.userRepo.FindByEmail(ctx, input.Email)
	if err != nil {
		return nil, fmt.Errorf("check email: %w", err)
	}
	if existing != nil {
		return nil, errors.NewConflict("email already registered")
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
			return nil, errors.NewValidation("invalid department code", nil)
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
		UserID:   user.ID,
		Username: generateUsername(input.FullName),
		FullName: input.FullName,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.userRepo.CreateProfile(ctx, profile); err != nil {
		return nil, fmt.Errorf("create profile: %w", err)
	}

	if input.USN != "" {
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
		return nil, "", errors.NewUnauthenticated("invalid credentials")
	}

	if !user.Status.CanLogin() {
		return nil, "", errors.NewUnauthenticated("account not active")
	}

	ok, err := s.passwordHasher.Verify(input.Password, user.PasswordHash)
	if err != nil {
		return nil, "", fmt.Errorf("verify password: %w", err)
	}
	if !ok {
		return nil, "", errors.NewUnauthenticated("invalid credentials")
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
		return nil, "", errors.NewUnauthenticated("invalid refresh token")
	}

	if stored.IsExpired() {
		return nil, "", errors.NewUnauthenticated("refresh token expired")
	}
	if stored.IsRevoked() {
		return nil, "", errors.NewUnauthenticated("refresh token revoked")
	}

	user, err := s.userRepo.FindByID(ctx, stored.UserID)
	if err != nil {
		return nil, "", fmt.Errorf("find user: %w", err)
	}
	if user == nil || !user.Status.CanLogin() {
		return nil, "", errors.NewUnauthenticated("user not active")
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
	if activation == nil {
		return errors.NewUnauthenticated("invalid or expired activation token")
	}

	if activation.UsedAt != nil {
		return errors.NewConflict("activation token already used")
	}

	if time.Now().After(activation.ExpiresAt) {
		return errors.NewUnauthenticated("activation token expired")
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
		return errors.NewNotFound("user not found")
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