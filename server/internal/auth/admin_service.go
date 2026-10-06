package auth

// Admin review of access requests and account status (the admin handlers).

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/AbhishekBalija/Links/server/internal/mailer"
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
	reviewDepartments, err := s.userRepo.ReviewDepartments(ctx)
	if err != nil {
		return nil, fmt.Errorf("find departments: %w", err)
	}
	departmentByID := map[string]ReviewDepartment{}
	for _, d := range reviewDepartments {
		departmentByID[d.ID] = d
	}
	ids := make([]string, 0, len(users))
	for _, u := range users {
		ids = append(ids, u.ID)
	}
	reported, err := s.userRepo.ReportedAt(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("find reported rows: %w", err)
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
		if at, ok := reported[u.ID]; ok {
			pur.ReportedAt = &at
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
			department := departmentByID[u.StudentIdentity.DepartmentID]
			pur.StudentIdentity = &PendingUserStudentID{
				USN:              u.StudentIdentity.USN,
				DepartmentCode:   departmentCode,
				DepartmentName:   department.Name,
				DepartmentHasHOD: department.HasHOD,
				BatchYear:        u.StudentIdentity.BatchYear,
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
	if err := checkApprovalScope(st, scopeID, anywhere, departments); err != nil {
		return err
	}
	// Approval leaves the account waiting for its first sign-in (spec #129).
	// The person is emailed once it's saved (#206).
	var approved *User
	err = s.unitOfWork.WithinTransaction(ctx, func(repos AuthRepositories) error {
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

		if st == ScopeDepartment {
			exists, lockErr := repos.Users.LockDepartmentForShare(ctx, scopeID)
			if lockErr != nil {
				return fmt.Errorf("lock department: %w", lockErr)
			}
			if !exists {
				return apperrors.NewValidation("invalid scope", map[string]string{"scope_id": "department not found"})
			}
		}

		// A row reported with "Not you?" kept its role: approving it again
		// must not add a second one.
		existing, err := repos.Users.GetRoleAssignments(ctx, userID)
		if err != nil {
			return fmt.Errorf("get roles: %w", err)
		}
		// A reported staff row has no Student identity: approving it gives
		// back the staff role it kept, never a student role.
		hasRole := user.StudentIdentity == nil
		for _, assignment := range existing {
			hasRole = hasRole || assignment.Role == role
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
		if !hasRole {
			if err := repos.Users.CreateRoleAssignment(ctx, ra); err != nil {
				return fmt.Errorf("create role assignment: %w", err)
			}
		}

		// Approval grants the role; the first sign-in makes the account active.
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
		if mayEmail, err := isOwnRequest(ctx, repos, user); err != nil {
			return err
		} else if mayEmail {
			approved = user
		}
		return nil
	})
	if err == nil && approved != nil {
		s.emailDecision(ctx, actorID, *approved, true, "")
	}
	return err
}

// UpdateUserStatus suspends, reactivates or rejects a user. The principal and
// admins may do any of these; an HOD only rejects an Access request in their
// Department (the handler lets nobody else through).
// checkApprovalScope checks the scope an approval gives the student role:
// global (no ID) or one Department. An HOD can only pick their own.
func checkApprovalScope(st ScopeType, scopeID string, anywhere bool, departments map[string]bool) error {
	switch st {
	case ScopeGlobal:
		if scopeID != "" {
			return apperrors.NewValidation("invalid scope", map[string]string{"scope_id": "must be empty for a global scope"})
		}
	case ScopeDepartment:
		if _, err := uuid.Parse(scopeID); err != nil {
			return apperrors.NewValidation("invalid scope", map[string]string{"scope_id": "must be a department ID"})
		}
		if !anywhere && !departments[scopeID] {
			return apperrors.NewValidation("invalid scope", map[string]string{"scope_id": "must be your own department"})
		}
	default:
		return apperrors.NewValidation("invalid scope", map[string]string{"scope_type": "must be global or department"})
	}
	return nil
}

func (s *authService) UpdateUserStatus(ctx context.Context, actorID, userID, status, note string) error {
	newStatus := UserStatus(status)
	anywhere, departments, err := s.departmentScope(ctx, actorID, accessRefusal)
	if err != nil {
		return err
	}
	var rejected *User
	err = s.unitOfWork.WithinTransaction(ctx, func(repos AuthRepositories) error {
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
		// Rejecting is for Access requests only, and removes the request so
		// the email and USN can be used again (#203). Members are suspended.
		if newStatus == UserStatusRejected {
			if user.Status != UserStatusPending || user.IsVerified {
				return apperrors.NewConflict("only an Access request can be rejected; suspend a member instead")
			}
			if mayEmail, err := isOwnRequest(ctx, repos, user); err != nil {
				return err
			} else if mayEmail {
				rejected = user
			}
			return rejectRequest(ctx, repos, actorID, user, note)
		}
		if newStatus == UserStatusSuspended {
			if err := requireSuspendable(ctx, repos, actorID, user.ID); err != nil {
				return err
			}
		}

		if newStatus == UserStatusActive && user.Status == UserStatusRejected {
			return apperrors.NewValidation("cannot activate a rejected user", nil)
		}
		if newStatus == UserStatusActive && user.Status == UserStatusPending {
			return apperrors.NewValidation("use the verify endpoint to activate a pending user", nil)
		}
		// Reactivating never skips approval or a first sign-in.
		if newStatus == UserStatusActive && !user.IsVerified {
			return apperrors.NewConflict("this account was never approved; approve its Access request instead")
		}
		if newStatus == UserStatusActive && user.CreatedBy != nil && user.FirstSignedInAt == nil {
			newStatus = UserStatusPending
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
	if err == nil && rejected != nil {
		s.emailDecision(ctx, actorID, *rejected, false, note)
	}
	return err
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

// requireSuspendable guards suspending (#177): nobody suspends themselves,
// only an admin suspends an admin or the principal, and the college keeps
// at least one active admin. Roles are read from the database.
func requireSuspendable(ctx context.Context, repos AuthRepositories, actorID, userID string) error {
	if actorID == userID {
		return apperrors.NewForbidden("you can't suspend yourself")
	}
	actorRoles, err := repos.Users.GetRoleAssignments(ctx, actorID)
	if err != nil {
		return fmt.Errorf("get actor roles: %w", err)
	}
	targetRoles, err := repos.Users.GetRoleAssignments(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user roles: %w", err)
	}
	has := func(grants []RoleAssignment, role Role) bool {
		for _, grant := range grants {
			if grant.Role == role {
				return true
			}
		}
		return false
	}
	targetAdmin := has(targetRoles, RoleAdmin)
	if (targetAdmin || has(targetRoles, RolePrincipal)) && !has(actorRoles, RoleAdmin) {
		return apperrors.NewForbidden("only an admin can suspend an admin or the principal")
	}
	if !targetAdmin {
		return nil
	}
	admins, err := repos.Users.LockAdminAssignmentsInEffect(ctx)
	if err != nil {
		return fmt.Errorf("lock admin roles: %w", err)
	}
	for _, other := range admins {
		if other.UserID == userID {
			continue
		}
		person, err := repos.Users.FindByID(ctx, other.UserID)
		if err != nil {
			return fmt.Errorf("find admin: %w", err)
		}
		if person != nil && person.Status == UserStatusActive {
			return nil
		}
	}
	return apperrors.NewConflict("this is the last active admin; make someone else an admin first")
}

// rejectRequest removes a rejected Access request and keeps a record of it.
func rejectRequest(ctx context.Context, repos AuthRepositories, actorID string, user *User, note string) error {
	metadata := map[string]string{"note": note}
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
			return apperrors.NewConflict("this account already has activity in LINKS, so it can't be removed; suspend it instead")
		}
		return fmt.Errorf("remove request: %w", err)
	}
	return repos.AuditLogs.Create(ctx, &AuditLog{
		ActorID:      &actorID,
		Action:       "access_request_rejected",
		ResourceType: "user",
		ResourceID:   &user.ID,
		Metadata:     metadata,
		CreatedAt:    time.Now(),
	})
}

func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

// isOwnRequest is true for an Access request the person sent themselves. A
// class-list row reported with "Not you?" isn't: the person at that email
// said it isn't theirs, so they aren't told about it.
func isOwnRequest(ctx context.Context, repos AuthRepositories, user *User) (bool, error) {
	if user.StudentIdentity == nil || user.Email == nil {
		return false, nil
	}
	reported, err := repos.Users.ReportedAt(ctx, []string{user.ID})
	if err != nil {
		return false, fmt.Errorf("find report: %w", err)
	}
	_, isReported := reported[user.ID]
	return !isReported, nil
}

// emailDecision tells someone whether their Access request was approved
// (#206). The decision stands even if the email can't be sent.
func (s *authService) emailDecision(ctx context.Context, actorID string, user User, approved bool, note string) {
	reviewer := "Your department"
	if actor, err := s.userRepo.FindByID(ctx, actorID); err == nil && actor != nil && actor.Profile != nil {
		reviewer = actor.Profile.FullName
	}
	joined := "a student"
	if user.StudentIdentity != nil {
		department := user.StudentIdentity.DepartmentID
		if departments, err := s.userRepo.ReviewDepartments(ctx); err == nil {
			for _, d := range departments {
				if d.ID == user.StudentIdentity.DepartmentID {
					department = d.Name
				}
			}
		}
		joined = fmt.Sprintf("a student of %s, batch %d", department, user.StudentIdentity.BatchYear)
	}
	fullName := ""
	if user.Profile != nil {
		fullName = user.Profile.FullName
	}
	_ = s.mailer.SendAccessDecision(*user.Email, mailer.AccessDecision{
		FullName: fullName, Approved: approved, ReviewerName: reviewer, Joined: joined, Note: strings.TrimSpace(note),
	})
}
