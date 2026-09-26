package auth

// Admin review of access requests and account status (the admin handlers).

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
)

func (s *authService) ReviewQueue(ctx context.Context) (*ReviewQueueResponse, error) {
	users, err := s.userRepo.FindPendingUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("find pending users: %w", err)
	}

	responses := make([]PendingUserResponse, 0, len(users))
	for _, u := range users {
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
			if code, err := ValidateUSN(u.StudentIdentity.USN); err == nil {
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

func (s *authService) VerifyUser(ctx context.Context, actorID, userID, scopeType, scopeID, note string) error {
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
	var email, fullName string
	if err := s.unitOfWork.WithinTransaction(ctx, func(repos AuthRepositories) error {
		user, err := repos.Users.FindByIDForUpdate(ctx, userID)
		if err != nil {
			return fmt.Errorf("find user: %w", err)
		}
		if user == nil {
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

func (s *authService) UpdateUserStatus(ctx context.Context, actorID, userID, status, note string) error {
	newStatus := UserStatus(status)
	return s.unitOfWork.WithinTransaction(ctx, func(repos AuthRepositories) error {
		user, err := repos.Users.FindByIDForUpdate(ctx, userID)
		if err != nil {
			return fmt.Errorf("find user: %w", err)
		}
		if user == nil {
			return apperrors.NewNotFound("user not found")
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
