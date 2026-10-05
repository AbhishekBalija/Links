package auth

// Welcoming someone to a new role, once. Only student coordinators are
// welcomed for now: their Home and what they can do change the most.

import (
	"context"
	"time"

	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
)

// NewRole is a current role its holder hasn't been welcomed to yet.
type NewRole struct {
	ID         string            `json:"id"`
	Role       Role              `json:"role"`
	Department NewRoleDepartment `json:"department"`
	// AssignedBy is who gave the role, when it was given in LINKS.
	AssignedBy *string   `json:"assigned_by"`
	StartedAt  time.Time `json:"started_at"`
}

type NewRoleDepartment struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// welcomedRoles are the roles that get a welcome.
var welcomedRoles = []Role{RoleStudentCoordinator}

// NewRole returns the user's oldest current role still waiting for its
// welcome, or nil.
func (s *authService) NewRole(ctx context.Context, userID string) (*NewRole, error) {
	return s.userRepo.UnwelcomedRole(ctx, userID, welcomedRoles, time.Now())
}

// Welcomed records that the holder closed the welcome. Doing it again is
// fine; someone else's role, or no role, is not found.
func (s *authService) Welcomed(ctx context.Context, userID, roleID string) error {
	found, err := s.userRepo.MarkWelcomed(ctx, userID, roleID, time.Now())
	if err != nil {
		return err
	}
	if !found {
		return apperrors.NewNotFound("role not found")
	}
	return nil
}
