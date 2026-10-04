package auth

// Fixing a wrong class-list row (#174): the email of a row nobody has
// signed into can be corrected, or the row removed so its USN and email can
// be added again. Admins act anywhere, an HOD in their own Departments: the
// same people who import (ADR 0029).

import (
	"context"
	"fmt"
	"net/mail"
	"strings"
	"time"

	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
)

const listRowRefusal = "only an admin or an HOD can fix class lists"

// FixRowEmail corrects the email of a row nobody has signed into, or of a
// row reported with "Not you?", which then waits for its first sign-in
// again with the new email.
func (s *authService) FixRowEmail(ctx context.Context, actorID, userID, email string) error {
	email = strings.TrimSpace(email)
	if address, err := mail.ParseAddress(email); err != nil || address.Address != email {
		return apperrors.NewValidation("email is not a valid address", map[string]string{"email": "invalid"})
	}
	anywhere, departments, err := s.adminOrHODScope(ctx, actorID, listRowRefusal)
	if err != nil {
		return err
	}
	now := time.Now()
	return s.unitOfWork.WithinTransaction(ctx, func(repos AuthRepositories) error {
		user, err := rowInScope(ctx, repos, userID, anywhere, departments)
		if err != nil {
			return err
		}
		waiting := user.WaitsForFirstSignIn() && user.FirstSignedInAt == nil && user.CreatedBy != nil
		reported := false
		if !waiting && user.Status == UserStatusPending && !user.IsVerified {
			found, err := repos.Users.ReportedAt(ctx, []string{user.ID})
			if err != nil {
				return fmt.Errorf("find report: %w", err)
			}
			_, reported = found[user.ID]
		}
		if !waiting && !reported {
			return apperrors.NewConflict("only a row nobody has signed into can have its email fixed")
		}
		if user.Email != nil && strings.EqualFold(*user.Email, email) {
			return apperrors.NewValidation("that is already their email", map[string]string{"email": "unchanged"})
		}
		if err := requireStaffEmail(ctx, repos.Users, email, now); err != nil {
			return err
		}
		old := ""
		if user.Email != nil {
			old = *user.Email
		}
		if err := repos.Users.FixEmail(ctx, user.ID, email, now); err != nil {
			return fmt.Errorf("fix email: %w", err)
		}
		// Whoever signed into a reported row is signed out of it for good.
		if err := repos.RefreshTokens.RevokeAllByUserID(ctx, user.ID); err != nil {
			return fmt.Errorf("sign out: %w", err)
		}
		return repos.AuditLogs.Create(ctx, &AuditLog{
			ActorID:      &actorID,
			Action:       "list_row_email_fixed",
			ResourceType: "user",
			ResourceID:   &user.ID,
			Metadata:     map[string]string{"from": old, "to": email},
			CreatedAt:    now,
		})
	})
}

// RemoveRow removes a row nobody has signed into, so its USN and email can
// be added again, for example from a corrected class list.
func (s *authService) RemoveRow(ctx context.Context, actorID, userID string) error {
	anywhere, departments, err := s.adminOrHODScope(ctx, actorID, listRowRefusal)
	if err != nil {
		return err
	}
	return s.unitOfWork.WithinTransaction(ctx, func(repos AuthRepositories) error {
		user, err := rowInScope(ctx, repos, userID, anywhere, departments)
		if err != nil {
			return err
		}
		if !user.WaitsForFirstSignIn() || user.FirstSignedInAt != nil || user.CreatedBy == nil {
			return apperrors.NewConflict("only a row nobody has signed into can be removed")
		}
		metadata := map[string]string{}
		if user.Email != nil {
			metadata["email"] = *user.Email
		}
		if user.Profile != nil {
			metadata["full_name"] = user.Profile.FullName
		}
		if user.StudentIdentity != nil {
			metadata["usn"] = user.StudentIdentity.USN
		}
		if err := repos.Users.RemoveNeverActive(ctx, user.ID); err != nil {
			if isForeignKeyViolation(err) {
				return apperrors.NewConflict("this account is already part of something in LINKS, so it can't be removed")
			}
			return fmt.Errorf("remove row: %w", err)
		}
		return repos.AuditLogs.Create(ctx, &AuditLog{
			ActorID:      &actorID,
			Action:       "list_row_removed",
			ResourceType: "user",
			ResourceID:   &user.ID,
			Metadata:     metadata,
			CreatedAt:    time.Now(),
		})
	})
}

// rowInScope loads and locks the row, or answers not found when it is
// outside an HOD's Departments: a student by their Student identity, staff
// by the Department of their roles.
func rowInScope(ctx context.Context, repos AuthRepositories, userID string, anywhere bool, departments map[string]bool) (*User, error) {
	user, err := repos.Users.FindByIDForUpdate(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}
	if user == nil {
		return nil, apperrors.NewNotFound("user not found")
	}
	if anywhere || inScope(*user, departments) {
		return user, nil
	}
	if user.StudentIdentity == nil {
		roles, err := repos.Users.GetRoleAssignments(ctx, user.ID)
		if err != nil {
			return nil, fmt.Errorf("get roles: %w", err)
		}
		for _, role := range roles {
			if role.ScopeID != nil && departments[*role.ScopeID] {
				return user, nil
			}
		}
	}
	return nil, apperrors.NewNotFound("user not found")
}
