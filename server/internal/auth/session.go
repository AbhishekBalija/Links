package auth

// Signing in and out: login, refresh-token rotation and logout.

import (
	"context"
	"errors"
	"fmt"
	"time"

	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
)

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
	return s.issueSession(ctx, user)
}

// TestSignIn signs an active member in by email alone, for the e2e suite.
// Its route exists only when the config allows it (APP_ENV=local).
func (s *authService) TestSignIn(ctx context.Context, email string) (*LoginResponse, string, error) {
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, "", fmt.Errorf("find user: %w", err)
	}
	if user == nil || !user.Status.CanLogin() {
		return nil, "", apperrors.NewUnauthenticated("no active member with this email")
	}
	return s.issueSession(ctx, user)
}

// issueSession gives a signed-in user an access token and a stored refresh
// token, whichever way they proved who they are.
func (s *authService) issueSession(ctx context.Context, user *User) (*LoginResponse, string, error) {
	return s.issueSessionWith(ctx, s.userRepo, s.refreshRepo, user)
}

// issueSessionWith is issueSession inside a caller's transaction.
func (s *authService) issueSessionWith(ctx context.Context, users UserRepository, refreshTokens RefreshTokenRepository, user *User) (*LoginResponse, string, error) {
	roles, err := users.GetRoleAssignments(ctx, user.ID)
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
	if err := refreshTokens.Create(ctx, refreshToken); err != nil {
		return nil, "", fmt.Errorf("store refresh token: %w", err)
	}

	return &LoginResponse{
		AccessToken: accessToken,
		ExpiresIn:   int(s.tokenCfg.AccessTTL.Seconds()),
	}, refreshRaw.Token, nil
}

func (s *authService) Refresh(ctx context.Context, refreshTokenRaw string) (*RefreshResponse, string, error) {
	hash := HashRefreshToken(refreshTokenRaw)

	var accessToken string
	var refreshRaw *RefreshTokenRaw
	if err := s.unitOfWork.WithinTransaction(ctx, func(repos AuthRepositories) error {
		stored, err := repos.RefreshTokens.FindByHash(ctx, hash)
		if err != nil {
			return fmt.Errorf("find refresh token: %w", err)
		}
		if stored == nil || stored.IsExpired() || stored.IsRevoked() {
			return apperrors.NewUnauthenticated("invalid refresh token")
		}
		if err := repos.RefreshTokens.RevokeIfActive(ctx, hash); err != nil {
			if errors.Is(err, errRefreshTokenUnavailable) {
				return apperrors.NewUnauthenticated("invalid refresh token")
			}
			return fmt.Errorf("revoke old token: %w", err)
		}

		user, err := repos.Users.FindByID(ctx, stored.UserID)
		if err != nil {
			return fmt.Errorf("find user: %w", err)
		}
		if user == nil || !user.Status.CanLogin() {
			return apperrors.NewUnauthenticated("user not active")
		}

		roles, err := repos.Users.GetRoleAssignments(ctx, user.ID)
		if err != nil {
			return fmt.Errorf("get roles: %w", err)
		}
		roleNames := make([]string, len(roles))
		for i, role := range roles {
			roleNames[i] = string(role.Role)
		}

		accessToken, err = GenerateAccessToken(user.ID, roleNames, s.tokenCfg)
		if err != nil {
			return fmt.Errorf("generate access token: %w", err)
		}
		refreshRaw, err = GenerateRefreshTokenRaw(s.tokenCfg)
		if err != nil {
			return fmt.Errorf("generate refresh token: %w", err)
		}
		newRefreshToken := &RefreshToken{
			UserID:    user.ID,
			TokenHash: refreshRaw.TokenHash,
			ExpiresAt: refreshRaw.ExpiresAt,
			CreatedAt: time.Now(),
		}
		if err := repos.RefreshTokens.Create(ctx, newRefreshToken); err != nil {
			return fmt.Errorf("store refresh token: %w", err)
		}
		return nil
	}); err != nil {
		return nil, "", err
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
