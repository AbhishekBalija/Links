package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/AbhishekBalija/Links/server/internal/mailer"
	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
)

type authService struct {
	userRepo       UserRepository
	refreshRepo    RefreshTokenRepository
	activationRepo ActivationTokenRepository
	unitOfWork     AuthUnitOfWork
	tokenCfg       TokenConfig
	passwordHasher PasswordHasher
	mailer         mailer.Mailer
	frontendURL    string
}

func NewAuthService(
	userRepo UserRepository,
	refreshRepo RefreshTokenRepository,
	activationRepo ActivationTokenRepository,
	unitOfWork AuthUnitOfWork,
	tokenCfg TokenConfig,
	passwordHasher PasswordHasher,
	mailer mailer.Mailer,
	frontendURL string,
) AuthService {
	return &authService{
		userRepo:       userRepo,
		refreshRepo:    refreshRepo,
		activationRepo: activationRepo,
		unitOfWork:     unitOfWork,
		tokenCfg:       tokenCfg,
		passwordHasher: passwordHasher,
		mailer:         mailer,
		frontendURL:    frontendURL,
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

	if input.USN != "" {
		usnCode, err := ValidateUSNFormat(input.USN)
		if err != nil {
			return nil, apperrors.NewValidation("invalid USN: "+err.Error(), nil)
		}
		// The USN names the Department, so a form choice can't contradict it.
		if input.DepartmentCode != "" && input.DepartmentCode != usnCode {
			return nil, apperrors.NewValidation("the USN's department code "+usnCode+" doesn't match the chosen department", nil)
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
	}

	var phone *string
	if normalizedPhone := strings.TrimSpace(input.Phone); normalizedPhone != "" {
		phone = &normalizedPhone
	}
	user := &User{
		Email:        &input.Email,
		Phone:        phone,
		PasswordHash: passwordHash,
		Status:       UserStatusPending,
		IsVerified:   false,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := s.unitOfWork.WithinTransaction(ctx, func(repos AuthRepositories) error {
		existing, err := repos.Users.FindByEmail(ctx, input.Email)
		if err != nil {
			return fmt.Errorf("recheck email: %w", err)
		}
		if existing != nil {
			return apperrors.NewConflict("email already registered")
		}

		if err := repos.Users.Create(ctx, user); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "idx_users_email" {
				return apperrors.NewConflict("email already registered")
			}
			return fmt.Errorf("create user: %w", err)
		}

		profile := &Profile{
			UserID:               user.ID,
			FullName:             input.FullName,
			PublicProfileEnabled: true,
			CreatedAt:            time.Now(),
			UpdatedAt:            time.Now(),
		}
		if err := s.createProfileWithRetry(ctx, repos.Users, profile); err != nil {
			return fmt.Errorf("create profile: %w", err)
		}

		if input.USN != "" {
			// The Batch comes from the USN, never from the form, so it
			// can't be missing or disagree with the USN (#45).
			batchYear, err := BatchYearFromUSN(input.USN)
			if err != nil {
				return apperrors.NewValidation("invalid USN: "+err.Error(), nil)
			}
			identity := &StudentIdentity{
				UserID:       user.ID,
				USN:          input.USN,
				DepartmentID: deptID,
				BatchYear:    batchYear,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}
			if err := repos.Users.CreateStudentIdentity(ctx, identity); err != nil {
				return fmt.Errorf("create student identity: %w", err)
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return &RequestAccessResponse{
		UserID: user.ID,
		Status: string(UserStatusPending),
	}, nil
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
