package auth

// Activation: a verified user sets their password through a single-use
// emailed link, which is the only way a self-service account becomes active.

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
)

func (s *authService) ActivateAccount(ctx context.Context, token, password string) error {
	hash := HashRefreshToken(token)

	passwordHash, err := s.passwordHasher.Hash(password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	return s.unitOfWork.WithinTransaction(ctx, func(repos AuthRepositories) error {
		activation, err := repos.Activations.FindByHash(ctx, hash)
		if err != nil {
			return fmt.Errorf("find activation token: %w", err)
		}
		if activation == nil || activation.UsedAt != nil || time.Now().After(activation.ExpiresAt) {
			return apperrors.NewUnauthenticated("invalid or expired activation token")
		}
		if err := repos.Activations.MarkUsed(ctx, activation.ID); err != nil {
			if errors.Is(err, errActivationTokenUnavailable) {
				return apperrors.NewUnauthenticated("invalid or expired activation token")
			}
			return fmt.Errorf("consume activation token: %w", err)
		}

		user, err := repos.Users.FindByID(ctx, activation.UserID)
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
		if err := repos.Users.Update(ctx, user); err != nil {
			return fmt.Errorf("update user: %w", err)
		}

		return nil
	})
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

	fullName := ""
	if user.Profile != nil {
		fullName = user.Profile.FullName
	}

	token, tokenRaw, err := newActivationToken(user.ID)
	if err != nil {
		return err
	}
	if err := s.unitOfWork.WithinTransaction(ctx, func(repos AuthRepositories) error {
		if err := repos.Activations.RevokeAllUnusedByUserID(ctx, user.ID); err != nil {
			return fmt.Errorf("revoke old tokens: %w", err)
		}
		if err := repos.Activations.Create(ctx, token); err != nil {
			return fmt.Errorf("create activation token: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}

	if err := s.sendStoredActivationEmail(email, fullName, tokenRaw); err != nil {
		if invalidateErr := s.invalidateActivationToken(ctx, token.ID); invalidateErr != nil {
			return fmt.Errorf("send activation email: %v; invalidate failed token: %w", err, invalidateErr)
		}
		return fmt.Errorf("send activation email: %w", err)
	}

	return nil
}

func newActivationToken(userID string) (*AccountActivationToken, string, error) {
	tokenRaw, tokenHash, err := generateActivationToken()
	if err != nil {
		return nil, "", fmt.Errorf("generate token: %w", err)
	}

	token := &AccountActivationToken{
		ID:        uuid.New().String(),
		UserID:    userID,
		TokenHash: tokenHash,
		Purpose:   "activate",
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		CreatedAt: time.Now(),
	}
	return token, tokenRaw, nil
}

func (s *authService) sendStoredActivationEmail(email, name, tokenRaw string) error {
	activationLink := s.frontendURL + "/activate?token=" + tokenRaw
	if err := s.mailer.SendActivationEmail(email, name, activationLink); err != nil {
		return fmt.Errorf("send email: %w", err)
	}

	return nil
}

func (s *authService) invalidateActivationToken(ctx context.Context, tokenID string) error {
	return s.activationRepo.MarkUsed(ctx, tokenID)
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
