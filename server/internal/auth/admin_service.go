package auth

// Admin review of access requests and account status (the admin handlers).

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
)

// ReviewQueue lists the Access requests waiting for the actor, oldest
// first: every one for the principal and admins, their own Department's for
// an HOD.
func (s *authService) ReviewQueue(ctx context.Context, actorID string) (*ReviewQueueResponse, error) {
	anywhere, departments, err := s.departmentScope(ctx, actorID, accessRefusal)
	if err != nil {
		return nil, err
	}
	users, err := s.userRepo.FindPendingUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("find pending users: %w", err)
	}

	responses := make([]PendingUserResponse, 0, len(users))
	for _, u := range users {
		if !anywhere && !inScope(u, departments) {
			continue
		}
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
			departmentCode := ""
			if code, err := ValidateUSNFormat(u.StudentIdentity.USN); err == nil {
				departmentCode = code
			}
			pur.StudentIdentity = &PendingUserStudentID{
				USN:            u.StudentIdentity.USN,
				DepartmentCode: departmentCode,
				BatchYear:      u.StudentIdentity.BatchYear,
			}
		}
		responses = append(responses, pur)
	}

	return &ReviewQueueResponse{Users: responses, Total: len(responses)}, nil
}

const accessRefusal = "only an admin, the principal or an HOD can decide access requests"

// inScope reports whether the user's Student identity is in one of the
// Departments.
func inScope(user User, departments map[string]bool) bool {
	return user.StudentIdentity != nil && departments[user.StudentIdentity.DepartmentID]
}

func (s *authService) VerifyUser(ctx context.Context, actorID, userID, scopeType, scopeID, note string) error {
	anywhere, departments, err := s.departmentScope(ctx, actorID, accessRefusal)
	if err != nil {
		return err
	}
	role := RoleStudent
	now := time.Now()
	st := ScopeType(scopeType)
	if st == "" {
		st = ScopeGlobal
	}
	token, tokenRaw, err := newActivationToken(userID)
	if err != nil {
		return err
	}
	// A request sent without a password (spec #129) has nothing to
	// activate: approval leaves it waiting for its first sign-in.
	var passwordless bool
	var email, fullName string
	if err := s.unitOfWork.WithinTransaction(ctx, func(repos AuthRepositories) error {
		user, err := repos.Users.FindByIDForUpdate(ctx, userID)
		if err != nil {
			return fmt.Errorf("find user: %w", err)
		}
		// A request outside an HOD's Department is not theirs to see.
		if user == nil || (!anywhere && !inScope(*user, departments)) {
			return apperrors.NewNotFound("user not found")
		}
		if user.Status != UserStatusPending {
			return apperrors.NewConflict("user is not in pending status")
		}
		if user.IsVerified {
			return apperrors.NewConflict("user is already verified")
		}
		if user.Email == nil {
			return fmt.Errorf("verified user has no email")
		}
		email = *user.Email
		passwordless = user.PasswordHash == ""
		if user.Profile != nil {
			fullName = user.Profile.FullName
		}

		if st == ScopeDepartment {
			if _, parseErr := uuid.Parse(scopeID); parseErr != nil {
				return apperrors.NewValidation("invalid scope", map[string]string{"scope_id": "must be a department ID"})
			}
			exists, lockErr := repos.Users.LockDepartmentForShare(ctx, scopeID)
			if lockErr != nil {
				return fmt.Errorf("lock department: %w", lockErr)
			}
			if !exists {
				return apperrors.NewValidation("invalid scope", map[string]string{"scope_id": "department not found"})
			}
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
		if err := repos.Users.CreateRoleAssignment(ctx, ra); err != nil {
			return fmt.Errorf("create role assignment: %w", err)
		}

		// Approval grants the role and permits activation; the activation link is
		// the only transition that makes a self-service account active.
		user.IsVerified = true
		user.UpdatedAt = now
		if err := repos.Users.Update(ctx, user); err != nil {
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
		if err := repos.AuditLogs.Create(ctx, auditLog); err != nil {
			return fmt.Errorf("create audit log: %w", err)
		}
		if passwordless {
			return nil
		}
		if err := repos.Activations.Create(ctx, token); err != nil {
			return fmt.Errorf("create activation token: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}
	if passwordless {
		return nil
	}

	if err := s.sendStoredActivationEmail(email, fullName, tokenRaw); err != nil {
		if invalidateErr := s.invalidateActivationToken(ctx, token.ID); invalidateErr != nil {
			return fmt.Errorf("send activation email: %v; invalidate failed token: %w", err, invalidateErr)
		}
		return fmt.Errorf("send activation email: %w", err)
	}

	return nil
}

// UpdateUserStatus suspends, reactivates or rejects a user. The principal and
// admins may do any of these; an HOD only rejects an Access request in their
// Department (the handler lets nobody else through).
func (s *authService) UpdateUserStatus(ctx context.Context, actorID, userID, status, note string) error {
	newStatus := UserStatus(status)
	anywhere, departments, err := s.departmentScope(ctx, actorID, accessRefusal)
	if err != nil {
		return err
	}
	return s.unitOfWork.WithinTransaction(ctx, func(repos AuthRepositories) error {
		user, err := repos.Users.FindByIDForUpdate(ctx, userID)
		if err != nil {
			return fmt.Errorf("find user: %w", err)
		}
		if user == nil || (!anywhere && !inScope(*user, departments)) {
			return apperrors.NewNotFound("user not found")
		}
		if !anywhere && (newStatus != UserStatusRejected || user.Status != UserStatusPending || user.IsVerified) {
			return apperrors.NewForbidden("an HOD can only reject an access request")
		}
		if newStatus == user.Status {
			return apperrors.NewConflict("user already has status " + status)
		}

		if newStatus == UserStatusActive && user.Status == UserStatusRejected {
			return apperrors.NewValidation("cannot activate a rejected user", nil)
		}
		if newStatus == UserStatusActive && user.Status == UserStatusPending {
			return apperrors.NewValidation("use the verify endpoint to activate a pending user", nil)
		}

		user.Status = newStatus
		user.UpdatedAt = time.Now()
		if err := repos.Users.Update(ctx, user); err != nil {
			return fmt.Errorf("update user: %w", err)
		}
		// A suspended or rejected user is signed out everywhere, not just
		// refused at their next refresh.
		if newStatus == UserStatusSuspended || newStatus == UserStatusRejected {
			if err := repos.RefreshTokens.RevokeAllByUserID(ctx, userID); err != nil {
				return fmt.Errorf("revoke refresh tokens: %w", err)
			}
		}

		auditLog := &AuditLog{
			ActorID:      &actorID,
			Action:       "user_status_changed",
			ResourceType: "user",
			ResourceID:   &userID,
			CreatedAt:    time.Now(),
			Metadata:     map[string]string{"new_status": status, "note": note},
		}
		if err := repos.AuditLogs.Create(ctx, auditLog); err != nil {
			return fmt.Errorf("create audit log: %w", err)
		}

		return nil
	})
}

// AccessSummary is how many Access requests wait for the actor and since
// when, for Home. It is nil for anyone who doesn't decide them.
type AccessSummary struct {
	PendingCount int        `json:"pending_count"`
	Oldest       *time.Time `json:"oldest_requested_at"`
}

func (s *authService) AccessSummary(ctx context.Context, actorID string) (*AccessSummary, error) {
	queue, err := s.ReviewQueue(ctx, actorID)
	var refused *apperrors.AppError
	if errors.As(err, &refused) && refused.HTTPStatus == http.StatusForbidden {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	summary := &AccessSummary{PendingCount: len(queue.Users)}
	if len(queue.Users) > 0 {
		oldest := queue.Users[0].CreatedAt
		summary.Oldest = &oldest
	}
	return summary, nil
}
