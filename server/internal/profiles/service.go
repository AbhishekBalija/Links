package profiles

import (
	"context"
	"fmt"
	"time"

	"github.com/AbhishekBalija/Links/server/internal/auth"
	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
)

type UserReader interface {
	FindEmailByUserID(ctx context.Context, userID string) (*string, error)
	FindPhoneByUserID(ctx context.Context, userID string) (*string, error)
}

type Service struct {
	repo       ProfileRepository
	userReader UserReader
	unitOfWork UnitOfWork
}

func NewService(repo ProfileRepository, userReader UserReader, unitOfWork UnitOfWork) *Service {
	return &Service{repo: repo, userReader: userReader, unitOfWork: unitOfWork}
}

func (s *Service) GetPublicProfile(ctx context.Context, username string, viewerID *string) (*ProfileResponse, error) {
	profile, err := s.repo.FindByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("find profile: %w", err)
	}
	if profile == nil {
		return nil, apperrors.NewNotFound("profile not found")
	}

	if !profile.PublicProfileEnabled && (viewerID == nil || *viewerID != profile.UserID) {
		return nil, apperrors.NewNotFound("profile not found")
	}

	isOwner := viewerID != nil && *viewerID == profile.UserID
	resp := s.profileToResponse(ctx, profile, isOwner)
	return resp, nil
}

func (s *Service) UpdateMyProfile(ctx context.Context, userID string, input UpdateProfileInput) (*ProfileResponse, error) {
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

		return nil
	}); err != nil {
		return nil, err
	}

	return s.profileToResponse(ctx, profile, true), nil
}

func (s *Service) profileToResponse(ctx context.Context, p *Profile, includePrivate bool) *ProfileResponse {
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

	if includePrivate || p.ShowEmail {
		email, err := s.userReader.FindEmailByUserID(ctx, p.UserID)
		if err == nil && email != nil {
			resp.Email = email
		}
	}
	if includePrivate || p.ShowPhone {
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
