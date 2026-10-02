package auth

// The first admin of a college is added once from the command line
// (cmd/add-admin, ADR 0026). Later admins are granted inside LINKS.

import (
	"context"
	"fmt"
	"net/mail"
	"strings"
	"time"

	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
)

// AddFirstAdmin creates the college's first admin: an account waiting for
// its first sign-in, which must be with Google. It refuses once any admin
// exists, so the command can't be used to add admins behind LINKS's back.
func (s *authService) AddFirstAdmin(ctx context.Context, email, fullName string) (string, error) {
	email = strings.TrimSpace(email)
	if address, err := mail.ParseAddress(email); err != nil || address.Address != email {
		return "", apperrors.NewValidation("email is not a valid address", map[string]string{"email": "invalid"})
	}
	fullName = strings.TrimSpace(fullName)
	if fullName == "" {
		return "", apperrors.NewValidation("full_name is empty", map[string]string{"full_name": "required"})
	}

	now := time.Now()
	user := &User{Email: &email, Status: UserStatusPending, IsVerified: true, CreatedAt: now, UpdatedAt: now}
	err := s.unitOfWork.WithinTransaction(ctx, func(repos AuthRepositories) error {
		hasAdmin, err := repos.Users.HasAdmin(ctx)
		if err != nil {
			return fmt.Errorf("find admins: %w", err)
		}
		if hasAdmin {
			return apperrors.NewConflict("this college already has an admin; grant more admins inside LINKS")
		}
		if err := requireNewEmail(ctx, repos.Users, email); err != nil {
			return err
		}
		if err := s.createAccount(ctx, repos.Users, user, fullName, now); err != nil {
			return err
		}
		if err := repos.Users.CreateRoleAssignment(ctx, &RoleAssignment{
			UserID: user.ID, Role: RoleAdmin, ScopeType: ScopeGlobal, StartsAt: now, CreatedAt: now,
		}); err != nil {
			return fmt.Errorf("grant admin role: %w", err)
		}
		return repos.AuditLogs.Create(ctx, &AuditLog{
			Action:       "first_admin_added",
			ResourceType: "user",
			ResourceID:   &user.ID,
			CreatedAt:    now,
		})
	})
	if err := uniqueViolation(err); err != nil {
		return "", err
	}
	return user.ID, nil
}
