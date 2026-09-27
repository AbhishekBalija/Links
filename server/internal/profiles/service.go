package profiles

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/AbhishekBalija/Links/server/internal/auth"
	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
)

type UserReader interface {
	FindEmailByUserID(ctx context.Context, userID string) (*string, error)
	FindPhoneByUserID(ctx context.Context, userID string) (*string, error)
}

// Membership is who a member is at the college: their roles in effect, most
// senior first, their Department and, for students, their Batch.
type Membership struct {
	Roles      []string
	Department *MembershipDepartment
	BatchYear  *int
}

type MembershipDepartment struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// MembershipReader reads a member's Membership. The directory implements it,
// so a profile and a directory entry describe a member the same way.
type MembershipReader interface {
	Membership(ctx context.Context, userID string) (*Membership, error)
}

type Service struct {
	repo        ProfileRepository
	userReader  UserReader
	memberships MembershipReader
	unitOfWork  UnitOfWork
}

func NewService(repo ProfileRepository, userReader UserReader, memberships MembershipReader, unitOfWork UnitOfWork) *Service {
	return &Service{repo: repo, userReader: userReader, memberships: memberships, unitOfWork: unitOfWork}
}

func (s *Service) GetPublicProfile(ctx context.Context, username string, viewerID *string) (*ProfileResponse, error) {
	profile, err := s.repo.FindByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("find profile: %w", err)
	}
	if profile == nil {
		return nil, apperrors.NewNotFound("profile not found")
	}

	if !profile.Privacy().VisibleTo(viewerID) {
		return nil, apperrors.NewNotFound("profile not found")
	}

	resp := s.profileToResponse(ctx, profile, viewerID)
	// Roles, Department and Batch are for members only, like the directory.
	if viewerID != nil {
		membership, err := s.memberships.Membership(ctx, profile.UserID)
		if err != nil {
			return nil, fmt.Errorf("read membership: %w", err)
		}
		resp.Roles = membership.Roles
		resp.Department = membership.Department
		resp.BatchYear = membership.BatchYear
	}
	return resp, nil
}

func (s *Service) UpdateMyProfile(ctx context.Context, userID string, input UpdateProfileInput) (*ProfileResponse, error) {
	if err := validateProfileURLs(input); err != nil {
		return nil, err
	}

	var profile *Profile
	if err := s.unitOfWork.WithinTransaction(ctx, func(repos Repositories) error {
		var err error
		profile, err = repos.Profiles.FindByUserID(ctx, userID)
		if err != nil {
			return fmt.Errorf("find profile: %w", err)
		}
		if profile == nil {
			return apperrors.NewNotFound("profile not found")
		}

		oldShowEmail := profile.ShowEmail
		oldShowPhone := profile.ShowPhone
		oldPublic := profile.PublicProfileEnabled
		applyUpdates(profile, input)
		profile.UpdatedAt = time.Now()

		if err := repos.Profiles.Update(ctx, profile); err != nil {
			return fmt.Errorf("update profile: %w", err)
		}
		if oldShowEmail != profile.ShowEmail || oldShowPhone != profile.ShowPhone {
			auditLog := &auth.AuditLog{
				ActorID:      &userID,
				Action:       "profile_privacy_updated",
				ResourceType: "profile",
				ResourceID:   &userID,
				Metadata: map[string]bool{
					"show_email": profile.ShowEmail,
					"show_phone": profile.ShowPhone,
				},
				CreatedAt: time.Now(),
			}
			if err := repos.AuditLogs.Create(ctx, auditLog); err != nil {
				return fmt.Errorf("audit privacy update: %w", err)
			}
		}
		// Visibility decides who can see the member at all, so it gets its
		// own entry.
		if oldPublic != profile.PublicProfileEnabled {
			auditLog := &auth.AuditLog{
				ActorID:      &userID,
				Action:       "profile_visibility_changed",
				ResourceType: "profile",
				ResourceID:   &userID,
				Metadata:     map[string]bool{"public_profile_enabled": profile.PublicProfileEnabled},
				CreatedAt:    time.Now(),
			}
			if err := repos.AuditLogs.Create(ctx, auditLog); err != nil {
				return fmt.Errorf("audit visibility change: %w", err)
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return s.profileToResponse(ctx, profile, &userID), nil
}

func validateProfileURLs(input UpdateProfileInput) error {
	fields := []struct {
		name  string
		value *string
	}{
		{name: "avatar_url", value: input.AvatarURL},
		{name: "linkedin_url", value: input.LinkedInURL},
		{name: "github_url", value: input.GitHubURL},
		{name: "portfolio_url", value: input.PortfolioURL},
	}

	for _, field := range fields {
		if field.value == nil || *field.value == "" {
			continue
		}
		parsed, err := url.ParseRequestURI(*field.value)
		if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return apperrors.NewValidation(field.name+" must be a valid HTTP or HTTPS URL", nil)
		}
	}

	return nil
}

func (s *Service) profileToResponse(ctx context.Context, p *Profile, viewerID *string) *ProfileResponse {
	resp := &ProfileResponse{
		UserID:               p.UserID,
		Username:             p.Username,
		FullName:             p.FullName,
		Headline:             p.Headline,
		Bio:                  p.Bio,
		AvatarURL:            p.AvatarURL,
		PublicProfileEnabled: p.PublicProfileEnabled,
		ShowEmail:            p.ShowEmail,
		ShowPhone:            p.ShowPhone,
		LinkedInURL:          p.LinkedInURL,
		GitHubURL:            p.GitHubURL,
		PortfolioURL:         p.PortfolioURL,
		CreatedAt:            p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:            p.UpdatedAt.Format(time.RFC3339),
	}

	privacy := p.Privacy()
	if privacy.ShowsEmailTo(viewerID) {
		email, err := s.userReader.FindEmailByUserID(ctx, p.UserID)
		if err == nil && email != nil {
			resp.Email = email
		}
	}
	if privacy.ShowsPhoneTo(viewerID) {
		phone, err := s.userReader.FindPhoneByUserID(ctx, p.UserID)
		if err == nil && phone != nil {
			resp.Phone = phone
		}
	}

	return resp
}

func applyUpdates(p *Profile, input UpdateProfileInput) {
	if input.Headline != nil {
		if *input.Headline == "" {
			p.Headline = nil
		} else {
			p.Headline = input.Headline
		}
	}
	if input.Bio != nil {
		if *input.Bio == "" {
			p.Bio = nil
		} else {
			p.Bio = input.Bio
		}
	}
	if input.AvatarURL != nil {
		if *input.AvatarURL == "" {
			p.AvatarURL = nil
		} else {
			p.AvatarURL = input.AvatarURL
		}
	}
	if input.PublicProfileEnabled != nil {
		p.PublicProfileEnabled = *input.PublicProfileEnabled
	}
	if input.ShowEmail != nil {
		p.ShowEmail = *input.ShowEmail
	}
	if input.ShowPhone != nil {
		p.ShowPhone = *input.ShowPhone
	}
	if input.LinkedInURL != nil {
		if *input.LinkedInURL == "" {
			p.LinkedInURL = nil
		} else {
			p.LinkedInURL = input.LinkedInURL
		}
	}
	if input.GitHubURL != nil {
		if *input.GitHubURL == "" {
			p.GitHubURL = nil
		} else {
			p.GitHubURL = input.GitHubURL
		}
	}
	if input.PortfolioURL != nil {
		if *input.PortfolioURL == "" {
			p.PortfolioURL = nil
		} else {
			p.PortfolioURL = input.PortfolioURL
		}
	}
}
