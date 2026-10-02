package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/AbhishekBalija/Links/server/internal/mailer"
	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
)

type authService struct {
	userRepo     UserRepository
	refreshRepo  RefreshTokenRepository
	unitOfWork   AuthUnitOfWork
	tokenCfg     TokenConfig
	mailer       mailer.Mailer
	codeSettings CodeSettings
	google       GoogleVerifier
}

func NewAuthService(
	userRepo UserRepository,
	refreshRepo RefreshTokenRepository,
	unitOfWork AuthUnitOfWork,
	tokenCfg TokenConfig,
	mailer mailer.Mailer,
	codeSettings CodeSettings,
	google GoogleVerifier,
) AuthService {
	return &authService{
		userRepo:     userRepo,
		refreshRepo:  refreshRepo,
		unitOfWork:   unitOfWork,
		tokenCfg:     tokenCfg,
		mailer:       mailer,
		codeSettings: codeSettings,
		google:       google,
	}
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

func stringPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (s *authService) createProfileWithRetry(ctx context.Context, repo UserRepository, profile *Profile) error {
	const maxAttempts = 5
	for range maxAttempts {
		profile.Username = generateUsername(profile.FullName)
		err := repo.CreateProfile(ctx, profile)
		if err == nil {
			return nil
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "idx_profiles_username" {
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
