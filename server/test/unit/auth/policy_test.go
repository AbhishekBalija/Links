package auth_test

import (
	"testing"

	"github.com/AbhishekBalija/Links/server/internal/auth"
)

func TestPolicy_AllRolesCanViewPublicProfiles(t *testing.T) {
	p := auth.NewPolicy()
	allRoles := []auth.Role{
		auth.RoleStudent, auth.RoleStudentCoordinator, auth.RoleFaculty, auth.RoleHOD,
		auth.RolePlacementOfficer, auth.RolePrincipal, auth.RoleAlumni,
		auth.RoleClubOrganizer, auth.RoleAdmin,
	}
	for _, role := range allRoles {
		actor := &auth.Actor{UserID: "test", Roles: []string{string(role)}}
		if err := p.Authorize(actor, auth.PermissionViewPublicProfiles); err != nil {
			t.Errorf("role %s should be able to view public profiles, got: %v", role, err)
		}
	}
}

func TestPolicy_OnlyAdminAndPrincipalCanFinalApprove(t *testing.T) {
	p := auth.NewPolicy()
	tests := []struct {
		role    auth.Role
		allowed bool
	}{
		{auth.RoleStudent, false},
		{auth.RoleStudentCoordinator, false},
		{auth.RoleFaculty, false},
		{auth.RoleHOD, false},
		{auth.RolePlacementOfficer, false},
		{auth.RolePrincipal, true},
		{auth.RoleAlumni, false},
		{auth.RoleClubOrganizer, false},
		{auth.RoleAdmin, true},
	}
	for _, tt := range tests {
		actor := &auth.Actor{UserID: "test", Roles: []string{string(tt.role)}}
		err := p.Authorize(actor, auth.PermissionFinalEventApproval)
		if tt.allowed && err != nil {
			t.Errorf("role %s should be allowed, got: %v", tt.role, err)
		}
		if !tt.allowed && err == nil {
			t.Errorf("role %s should be denied", tt.role)
		}
	}
}

func TestPolicy_NilActor(t *testing.T) {
	p := auth.NewPolicy()
	if err := p.Authorize(nil, auth.PermissionViewPublicProfiles); err == nil {
		t.Error("nil actor should be denied")
	}
}

func TestPolicy_EmptyRoles(t *testing.T) {
	p := auth.NewPolicy()
	actor := &auth.Actor{UserID: "test", Roles: []string{}}
	if err := p.Authorize(actor, auth.PermissionFinalEventApproval); err == nil {
		t.Error("empty roles should be denied for admin-only permission")
	}
}

func TestPolicy_PlacementPermissions(t *testing.T) {
	p := auth.NewPolicy()
	placementOnly := []auth.Permission{
		auth.PermissionPostOpportunity,
		auth.PermissionViewApplicantData,
		auth.PermissionShortlistApplicants,
	}
	for _, perm := range placementOnly {
		if err := p.Authorize(&auth.Actor{Roles: []string{string(auth.RolePlacementOfficer)}}, perm); err != nil {
			t.Errorf("placement officer should be allowed for %s", perm)
		}
		if err := p.Authorize(&auth.Actor{Roles: []string{string(auth.RoleStudent)}}, perm); err == nil {
			t.Errorf("student should be denied for %s", perm)
		}
	}
}
