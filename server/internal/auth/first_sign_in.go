package auth

// First sign-in: an account waiting for its first sign-in (an imported row, a
// staff invite or an approved Access request) becomes active the first time
// someone signs in with its email, and the screen shows who they signed in as
// so a wrong row is reported ("Not you?") rather than used (spec #129).

import (
	"context"
	"fmt"
	"time"

	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
)

// NotMeWindow is how long after a first sign-in "Not you?" is offered.
const NotMeWindow = time.Hour

// FirstSignInResponse says who a first sign-in signed in as: "Asha Rao ·
// 4MN23CS042 · CS 2023. Not you?".
type FirstSignInResponse struct {
	FullName       string   `json:"full_name"`
	Email          string   `json:"email"`
	USN            *string  `json:"usn,omitempty"`
	DepartmentCode *string  `json:"department_code,omitempty"`
	DepartmentName *string  `json:"department_name,omitempty"`
	BatchYear      *int     `json:"batch_year,omitempty"`
	Roles          []string `json:"roles"`
}

// signIn finishes any way of signing in, inside the caller's transaction:
// it completes a first sign-in, issues the session and audits both.
func (s *authService) signIn(ctx context.Context, repos AuthRepositories, user *User, method string, now time.Time) (*LoginResponse, string, error) {
	var first *FirstSignInResponse
	if user.WaitsForFirstSignIn() {
		if err := repos.Users.CompleteFirstSignIn(ctx, user.ID, now); err != nil {
			return nil, "", fmt.Errorf("complete first sign-in: %w", err)
		}
		var err error
		first, err = firstSignInOf(ctx, repos.Users, user)
		if err != nil {
			return nil, "", err
		}
		if err := repos.AuditLogs.Create(ctx, &AuditLog{
			ActorID:      &user.ID,
			Action:       "auth.first_sign_in",
			ResourceType: "user",
			ResourceID:   &user.ID,
			Metadata:     map[string]string{"method": method},
			CreatedAt:    now,
		}); err != nil {
			return nil, "", fmt.Errorf("audit first sign-in: %w", err)
		}
	}

	resp, refreshRaw, err := s.issueSessionWith(ctx, repos.Users, repos.RefreshTokens, user)
	if err != nil {
		return nil, "", err
	}
	if err := repos.AuditLogs.Create(ctx, signInAuditLog(user.ID, method, now)); err != nil {
		return nil, "", fmt.Errorf("audit sign-in: %w", err)
	}
	resp.FirstSignIn = first
	return resp, refreshRaw, nil
}

// firstSignInOf describes the account: its Student identity's Department and
// Batch, or for staff the first Department one of their roles is scoped to.
func firstSignInOf(ctx context.Context, users UserRepository, user *User) (*FirstSignInResponse, error) {
	first := &FirstSignInResponse{Roles: []string{}}
	if user.Email != nil {
		first.Email = *user.Email
	}
	if user.Profile != nil {
		first.FullName = user.Profile.FullName
	}

	roles, err := users.GetRoleAssignments(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("get roles: %w", err)
	}
	departmentID := ""
	for _, role := range roles {
		first.Roles = append(first.Roles, string(role.Role))
		if departmentID == "" && role.ScopeType == ScopeDepartment && role.ScopeID != nil {
			departmentID = *role.ScopeID
		}
	}
	if identity := user.StudentIdentity; identity != nil {
		first.USN = &identity.USN
		first.BatchYear = &identity.BatchYear
		departmentID = identity.DepartmentID
	}
	if departmentID != "" {
		department, err := users.FindDepartmentByID(ctx, departmentID)
		if err != nil {
			return nil, fmt.Errorf("find department: %w", err)
		}
		if department != nil {
			first.DepartmentCode, first.DepartmentName = &department.Code, &department.Name
		}
	}
	return first, nil
}

// NotMe is "Not you?" on a first sign-in: the account isn't the person who
// signed in, so a list row is wrong. It signs them out everywhere, unlinks
// any Google account and returns the account to waiting, so an admin can fix
// the row. Offered only within NotMeWindow of the first sign-in.
func (s *authService) NotMe(ctx context.Context, userID string) error {
	now := time.Now()
	return s.unitOfWork.WithinTransaction(ctx, func(repos AuthRepositories) error {
		user, err := repos.Users.FindByIDForUpdate(ctx, userID)
		if err != nil {
			return fmt.Errorf("find user: %w", err)
		}
		if user == nil {
			return apperrors.NewNotFound("user not found")
		}
		if user.Status != UserStatusActive || user.FirstSignedInAt == nil || now.Sub(*user.FirstSignedInAt) > NotMeWindow {
			return apperrors.NewConflict("\"Not you?\" is only offered right after a first sign-in")
		}
		if err := repos.Users.ReturnToWaiting(ctx, user.ID); err != nil {
			return fmt.Errorf("return to waiting: %w", err)
		}
		if err := repos.RefreshTokens.RevokeAllByUserID(ctx, user.ID); err != nil {
			return fmt.Errorf("sign out everywhere: %w", err)
		}
		metadata := map[string]string{}
		if user.StudentIdentity != nil {
			metadata["usn"] = user.StudentIdentity.USN
		}
		if user.GoogleSubject != nil {
			metadata["unlinked_google_account"] = "true"
		}
		return repos.AuditLogs.Create(ctx, &AuditLog{
			ActorID:      &user.ID,
			Action:       "auth.not_me",
			ResourceType: "user",
			ResourceID:   &user.ID,
			Metadata:     metadata,
			CreatedAt:    now,
		})
	})
}
