package auth

// Role management: granting and ending staff Role assignments (the admin
// role handlers).

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"

	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
)

// grantableScopes lists the roles granted through role management and the
// Scope each needs. Student and alumni come from Access approval and
// Graduation, and club roles wait for clubs.
var grantableScopes = map[Role]ScopeType{
	RoleFaculty:            ScopeDepartment,
	RoleHOD:                ScopeDepartment,
	RoleStudentCoordinator: ScopeDepartment,
	RolePlacementOfficer:   ScopeGlobal,
	RolePrincipal:          ScopeGlobal,
	RoleAdmin:              ScopeGlobal,
}

// startsAtTolerance lets a client send "now" without a clock skew making it
// look like a date in the past.
const startsAtTolerance = time.Minute

func (s *authService) ListUserRoles(ctx context.Context, actorID, userID string) ([]RoleAssignmentResponse, error) {
	if _, err := uuid.Parse(userID); err != nil {
		return nil, apperrors.NewNotFound("user not found")
	}
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}
	if user == nil {
		return nil, apperrors.NewNotFound("user not found")
	}
	manager, err := readRoleManager(ctx, s.userRepo, actorID)
	if err != nil {
		return nil, err
	}
	if !manager.sees(user) {
		return nil, apperrors.NewNotFound("user not found")
	}
	views, err := s.userRepo.ListRoleAssignments(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list role assignments: %w", err)
	}
	now := time.Now()
	responses := make([]RoleAssignmentResponse, 0, len(views))
	for _, view := range views {
		responses = append(responses, roleAssignmentResponse(view, now))
	}
	return responses, nil
}

func (s *authService) GrantRole(ctx context.Context, actorID, userID string, input GrantRoleInput) (*RoleAssignmentResponse, error) {
	now := time.Now()
	role := Role(input.Role)
	scopeType := ScopeType(input.ScopeType)
	if err := validateGrant(role, scopeType, input.ScopeID); err != nil {
		return nil, err
	}
	startsAt := now
	if input.StartsAt != nil {
		if input.StartsAt.Before(now.Add(-startsAtTolerance)) {
			return nil, apperrors.NewValidation("invalid role dates", map[string]string{"starts_at": "must not be in the past"})
		}
		startsAt = *input.StartsAt
	}
	if input.EndsAt != nil && (!input.EndsAt.After(startsAt) || !input.EndsAt.After(now)) {
		return nil, apperrors.NewValidation("invalid role dates", map[string]string{"ends_at": "must be after starts_at and in the future"})
	}
	if _, err := uuid.Parse(userID); err != nil {
		return nil, apperrors.NewNotFound("user not found")
	}

	var created RoleAssignmentResponse
	err := s.unitOfWork.WithinTransaction(ctx, func(repos AuthRepositories) error {
		var err error
		created, err = grantRoleIn(ctx, repos, actorID, userID, input, startsAt, now)
		return err
	})
	if err != nil {
		return nil, err
	}
	return &created, nil
}

// grantRoleIn grants a role inside the caller's transaction. The caller has
// validated the role, Scope and dates.
func grantRoleIn(ctx context.Context, repos AuthRepositories, actorID, userID string, input GrantRoleInput, startsAt, now time.Time) (RoleAssignmentResponse, error) {
	role := Role(input.Role)
	scopeType := ScopeType(input.ScopeType)
	// Locking the user makes two identical grants run one after another,
	// so the duplicate check below sees the first one.
	user, err := repos.Users.FindByIDForUpdate(ctx, userID)
	if err != nil {
		return RoleAssignmentResponse{}, fmt.Errorf("find user: %w", err)
	}
	if user == nil {
		return RoleAssignmentResponse{}, apperrors.NewNotFound("user not found")
	}
	if err := requireRoleManager(ctx, repos.Users, actorID, user, role); err != nil {
		return RoleAssignmentResponse{}, err
	}
	if user.Status == UserStatusRejected {
		return RoleAssignmentResponse{}, apperrors.NewConflict("a rejected user can't be given a role")
	}

	var scopeID *string
	var department *RoleDepartment
	if scopeType == ScopeDepartment {
		scopeID = &input.ScopeID
		department, err = lockScopeDepartment(ctx, repos, role, input.ScopeID)
		if err != nil {
			return RoleAssignmentResponse{}, err
		}
	}
	if role == RoleStudentCoordinator {
		if err := requireStudentOf(ctx, repos, user, input.ScopeID); err != nil {
			return RoleAssignmentResponse{}, err
		}
	}

	overlap := OverlapFilter{UserID: userID, Role: role, ScopeType: scopeType, ScopeID: scopeID, StartsAt: startsAt, EndsAt: input.EndsAt}
	duplicate, err := repos.Users.HasOverlappingAssignment(ctx, overlap)
	if err != nil {
		return RoleAssignmentResponse{}, fmt.Errorf("check duplicate role: %w", err)
	}
	if duplicate {
		return RoleAssignmentResponse{}, apperrors.NewConflict("the user already has this role and scope for that time")
	}
	if role == RoleHOD {
		// A Department has at most one HOD at a time (CONTEXT.md).
		overlap.UserID = ""
		taken, err := repos.Users.HasOverlappingAssignment(ctx, overlap)
		if err != nil {
			return RoleAssignmentResponse{}, fmt.Errorf("check existing HOD: %w", err)
		}
		if taken {
			return RoleAssignmentResponse{}, apperrors.NewConflict("the department already has an HOD for that time")
		}
	}

	assignment := &RoleAssignment{
		UserID:     userID,
		Role:       role,
		ScopeType:  scopeType,
		ScopeID:    scopeID,
		AssignedBy: &actorID,
		StartsAt:   startsAt,
		EndsAt:     input.EndsAt,
		CreatedAt:  now,
	}
	if err := repos.Users.CreateRoleAssignment(ctx, assignment); err != nil {
		return RoleAssignmentResponse{}, fmt.Errorf("create role assignment: %w", err)
	}
	if err := repos.AuditLogs.Create(ctx, roleAuditLog("role_granted", actorID, *assignment, input.Note, now)); err != nil {
		return RoleAssignmentResponse{}, fmt.Errorf("create audit log: %w", err)
	}
	view := RoleAssignmentView{RoleAssignment: *assignment}
	if department != nil {
		view.DepartmentCode, view.DepartmentName = &department.Code, &department.Name
	}
	return roleAssignmentResponse(view, now), nil
}

// EndRole ends a Role assignment now, keeping the row as history, and signs
// the user out everywhere so their next access token carries the new roles.
func (s *authService) EndRole(ctx context.Context, actorID, userID, assignmentID string) (*RoleAssignmentResponse, error) {
	if _, err := uuid.Parse(userID); err != nil {
		return nil, apperrors.NewNotFound("role assignment not found")
	}
	if _, err := uuid.Parse(assignmentID); err != nil {
		return nil, apperrors.NewNotFound("role assignment not found")
	}
	now := time.Now()
	var ended RoleAssignmentResponse
	err := s.unitOfWork.WithinTransaction(ctx, func(repos AuthRepositories) error {
		assignment, err := repos.Users.FindRoleAssignmentForUpdate(ctx, userID, assignmentID)
		if err != nil {
			return fmt.Errorf("find role assignment: %w", err)
		}
		if assignment == nil {
			return apperrors.NewNotFound("role assignment not found")
		}
		user, err := repos.Users.FindByID(ctx, userID)
		if err != nil {
			return fmt.Errorf("find user: %w", err)
		}
		if err := requireRoleManager(ctx, repos.Users, actorID, user, assignment.Role); err != nil {
			return err
		}
		if roleState(*assignment, now) == "ended" {
			return apperrors.NewConflict("the role assignment has already ended")
		}
		if assignment.Role == RoleAdmin {
			if err := requireAnotherAdmin(ctx, repos, assignment.ID); err != nil {
				return err
			}
		}

		// A scheduled assignment ends before it starts, so it never takes effect.
		endsAt := now
		if assignment.StartsAt.After(now) {
			endsAt = assignment.StartsAt
		}
		if err := repos.Users.EndRoleAssignment(ctx, assignment.ID, endsAt); err != nil {
			return fmt.Errorf("end role assignment: %w", err)
		}
		assignment.EndsAt = &endsAt
		if assignment.Role == RoleHOD && assignment.ScopeID != nil {
			if err := repos.Users.ClearDepartmentHOD(ctx, *assignment.ScopeID, userID); err != nil {
				return fmt.Errorf("clear department HOD: %w", err)
			}
		}
		if err := repos.RefreshTokens.RevokeAllByUserID(ctx, userID); err != nil {
			return fmt.Errorf("revoke refresh tokens: %w", err)
		}
		if err := repos.AuditLogs.Create(ctx, roleAuditLog("role_ended", actorID, *assignment, "", now)); err != nil {
			return fmt.Errorf("create audit log: %w", err)
		}
		ended = roleAssignmentResponse(RoleAssignmentView{RoleAssignment: *assignment}, now)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &ended, nil
}

func validateGrant(role Role, scopeType ScopeType, scopeID string) error {
	wantScope, ok := grantableScopes[role]
	if !ok {
		return apperrors.NewValidation("invalid role", map[string]string{"role": "must be faculty, hod, student_coordinator, placement_officer, principal or admin"})
	}
	if scopeType != wantScope {
		return apperrors.NewValidation("invalid scope", map[string]string{"scope_type": fmt.Sprintf("%s needs a %s scope", role, wantScope)})
	}
	if scopeType == ScopeGlobal && scopeID != "" {
		return apperrors.NewValidation("invalid scope", map[string]string{"scope_id": "must be empty for a global role"})
	}
	if scopeType == ScopeDepartment {
		if _, err := uuid.Parse(scopeID); err != nil {
			return apperrors.NewValidation("invalid scope", map[string]string{"scope_id": "must be a department ID"})
		}
	}
	return nil
}

// roleManager is what the actor may do in role management, read from the
// database rather than the token (ADR 0027).
type roleManager struct {
	admin     bool
	principal bool
	// hodOf lists the Departments the actor is HOD of.
	hodOf []string
}

func readRoleManager(ctx context.Context, users UserRepository, actorID string) (roleManager, error) {
	grants, err := users.GetRoleAssignments(ctx, actorID)
	if err != nil {
		return roleManager{}, fmt.Errorf("get actor roles: %w", err)
	}
	var manager roleManager
	for _, grant := range grants {
		switch grant.Role {
		case RoleAdmin:
			manager.admin = true
		case RolePrincipal:
			manager.principal = true
		case RoleHOD:
			if grant.ScopeID != nil {
				manager.hodOf = append(manager.hodOf, *grant.ScopeID)
			}
		}
	}
	return manager, nil
}

// sees reports whether the actor may look at the user's roles: the principal
// and admins see everyone, an HOD only the students of their Department.
func (m roleManager) sees(user *User) bool {
	if m.admin || m.principal {
		return true
	}
	if user.StudentIdentity == nil {
		return false
	}
	return slices.Contains(m.hodOf, user.StudentIdentity.DepartmentID)
}

// mayManage reports whether the actor grants and ends the role: an HOD the
// Student coordinator role, the principal senior staff roles, an admin all.
func (m roleManager) mayManage(role Role) bool {
	switch {
	case m.admin:
		return true
	case m.principal && (role == RoleFaculty || role == RoleHOD || role == RolePlacementOfficer):
		return true
	case len(m.hodOf) > 0 && role == RoleStudentCoordinator:
		return true
	}
	return false
}

// requireRoleManager checks the actor may grant or end the role for the
// user. A user the actor can't see answers as if they didn't exist.
func requireRoleManager(ctx context.Context, users UserRepository, actorID string, user *User, role Role) error {
	manager, err := readRoleManager(ctx, users, actorID)
	if err != nil {
		return err
	}
	if user == nil || !manager.sees(user) {
		return apperrors.NewNotFound("user not found")
	}
	if !manager.mayManage(role) {
		if role == RoleAdmin {
			return apperrors.NewForbidden("only an admin can grant or end the admin role")
		}
		return apperrors.NewForbidden(fmt.Sprintf("you can't grant or end the %s role", role))
	}
	return nil
}

// lockScopeDepartment checks the department exists and locks it: a share lock
// so it can't be deleted meanwhile, or a full lock for an HOD grant so two
// grants for one department can't both pass the one-HOD check.
func lockScopeDepartment(ctx context.Context, repos AuthRepositories, role Role, departmentID string) (*RoleDepartment, error) {
	var exists bool
	var err error
	if role == RoleHOD {
		exists, err = repos.Users.LockDepartmentForUpdate(ctx, departmentID)
	} else {
		exists, err = repos.Users.LockDepartmentForShare(ctx, departmentID)
	}
	if err != nil {
		return nil, fmt.Errorf("lock department: %w", err)
	}
	if !exists {
		return nil, apperrors.NewValidation("invalid scope", map[string]string{"scope_id": "department not found"})
	}
	department, err := repos.Users.FindDepartmentByID(ctx, departmentID)
	if err != nil {
		return nil, fmt.Errorf("find department: %w", err)
	}
	return &RoleDepartment{ID: department.ID, Code: department.Code, Name: department.Name}, nil
}

// requireStudentOf checks that a Student coordinator is a current Student of
// the Department they coordinate for.
func requireStudentOf(ctx context.Context, repos AuthRepositories, user *User, departmentID string) error {
	notStudent := apperrors.NewValidation("invalid role", map[string]string{"role": "a student coordinator must be a current student of that department"})
	if user.StudentIdentity == nil || user.StudentIdentity.DepartmentID != departmentID {
		return notStudent
	}
	grants, err := repos.Users.GetRoleAssignments(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("get user roles: %w", err)
	}
	for _, grant := range grants {
		if grant.Role == RoleStudent {
			return nil
		}
	}
	return notStudent
}

// requireAnotherAdmin refuses to end the last admin assignment in effect, so
// the college can't lose every admin by accident.
func requireAnotherAdmin(ctx context.Context, repos AuthRepositories, assignmentID string) error {
	admins, err := repos.Users.LockAdminAssignmentsInEffect(ctx)
	if err != nil {
		return fmt.Errorf("lock admin roles: %w", err)
	}
	for _, admin := range admins {
		if admin.ID != assignmentID {
			return nil
		}
	}
	return apperrors.NewConflict("this is the last admin; grant another admin first")
}

func roleState(assignment RoleAssignment, now time.Time) string {
	switch {
	case assignment.EndsAt != nil && !assignment.EndsAt.After(now):
		return "ended"
	case assignment.EndsAt != nil && !assignment.EndsAt.After(assignment.StartsAt):
		return "ended"
	case assignment.StartsAt.After(now):
		return "scheduled"
	default:
		return "active"
	}
}

func roleAssignmentResponse(view RoleAssignmentView, now time.Time) RoleAssignmentResponse {
	response := RoleAssignmentResponse{
		ID:         view.ID,
		Role:       string(view.Role),
		ScopeType:  string(view.ScopeType),
		ScopeID:    view.ScopeID,
		AssignedBy: view.AssignedBy,
		StartsAt:   view.StartsAt,
		EndsAt:     view.EndsAt,
		State:      roleState(view.RoleAssignment, now),
		CreatedAt:  view.CreatedAt,
	}
	if view.ScopeType == ScopeDepartment && view.ScopeID != nil && view.DepartmentCode != nil {
		response.Department = &RoleDepartment{ID: *view.ScopeID, Code: *view.DepartmentCode}
		if view.DepartmentName != nil {
			response.Department.Name = *view.DepartmentName
		}
	}
	return response
}

func roleAuditLog(action, actorID string, assignment RoleAssignment, note string, now time.Time) *AuditLog {
	metadata := map[string]string{
		"role_assignment_id": assignment.ID,
		"role":               string(assignment.Role),
		"scope_type":         string(assignment.ScopeType),
		"starts_at":          assignment.StartsAt.UTC().Format(time.RFC3339),
	}
	if assignment.ScopeID != nil {
		metadata["scope_id"] = *assignment.ScopeID
	}
	if assignment.EndsAt != nil {
		metadata["ends_at"] = assignment.EndsAt.UTC().Format(time.RFC3339)
	}
	if note != "" {
		metadata["note"] = note
	}
	userID := assignment.UserID
	return &AuditLog{
		ActorID:      &actorID,
		Action:       action,
		ResourceType: "user",
		ResourceID:   &userID,
		CreatedAt:    now,
		Metadata:     metadata,
	}
}
