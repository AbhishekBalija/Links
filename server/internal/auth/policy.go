package auth

import (
	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
	"github.com/gin-gonic/gin"
)

type Permission string

const (
	PermissionViewPublicProfiles  Permission = "view_public_profiles"
	PermissionEditOwnProfile      Permission = "edit_own_profile"
	PermissionViewTargetedNotices Permission = "view_targeted_notices"
	PermissionPostAnnouncement    Permission = "post_announcement"
	PermissionApproveAnnouncement Permission = "approve_announcement"
	PermissionProposeEvent        Permission = "propose_event"
	PermissionReviewBranchEvent   Permission = "review_branch_event"
	PermissionFinalEventApproval  Permission = "final_event_approval"
	PermissionPostOpportunity     Permission = "post_opportunity"
	PermissionViewApplicantData   Permission = "view_applicant_data"
	PermissionShortlistApplicants Permission = "shortlist_applicants"
	PermissionManageUsersAndRoles Permission = "manage_users_and_roles"
	PermissionManageDepartments   Permission = "manage_departments"
	PermissionImportStudents      Permission = "import_students"
	PermissionApproveAccess       Permission = "approve_access"
	PermissionManageRoles         Permission = "manage_roles"
)

type Policy struct {
	grants map[Permission][]Role
}

func NewPolicy() *Policy {
	return &Policy{
		grants: map[Permission][]Role{
			PermissionViewPublicProfiles:  allRoles(),
			PermissionEditOwnProfile:      allRoles(),
			PermissionViewTargetedNotices: allRoles(),
			// Who may post at all (docs/auth.md). Whether a post publishes directly
			// or needs approval is decided in the announcements service (ADR 0017).
			PermissionPostAnnouncement: {RoleStudentCoordinator, RoleFaculty, RoleHOD, RolePlacementOfficer, RolePrincipal, RoleAdmin},
			// Which Announcements each approver may review is decided in the
			// announcements service (ADR 0017).
			PermissionApproveAnnouncement: {RoleHOD, RolePrincipal, RoleAdmin},
			PermissionProposeEvent:        {RoleStudentCoordinator, RoleFaculty, RoleHOD, RolePlacementOfficer, RolePrincipal, RoleAdmin},
			PermissionReviewBranchEvent:   {RoleHOD, RolePrincipal, RoleAdmin},
			PermissionFinalEventApproval:  {RolePrincipal, RoleAdmin},
			PermissionPostOpportunity:     {RolePlacementOfficer, RolePrincipal, RoleAdmin},
			PermissionViewApplicantData:   {RolePlacementOfficer, RolePrincipal, RoleAdmin},
			PermissionShortlistApplicants: {RolePlacementOfficer, RolePrincipal, RoleAdmin},
			PermissionManageUsersAndRoles: {RolePrincipal, RoleAdmin},
			PermissionManageDepartments:   {RoleAdmin},
			// HODs import only their own Department's students; the service
			// checks the Department (#17).
			PermissionImportStudents: {RoleHOD, RolePrincipal, RoleAdmin},
			// HODs decide Access requests for their own Department only; the
			// service checks the Department (ADR 0025).
			PermissionApproveAccess: {RoleHOD, RolePrincipal, RoleAdmin},
			// Which roles each manager grants or ends, and for whom, is decided
			// in the auth service (ADR 0027).
			PermissionManageRoles: {RoleHOD, RolePrincipal, RoleAdmin},
		},
	}
}

func (p *Policy) Authorize(actor *Actor, perm Permission) error {
	if actor == nil {
		return apperrors.NewForbidden("not authenticated")
	}
	allowed, ok := p.grants[perm]
	if !ok {
		return apperrors.NewForbidden("unknown permission")
	}
	for _, actorRole := range actor.Roles {
		for _, allowedRole := range allowed {
			if Role(actorRole) == allowedRole {
				return nil
			}
		}
	}
	return apperrors.NewForbidden("insufficient permissions")
}

func AuthorizeActor(c *gin.Context, p *Policy, perm Permission) error {
	return p.Authorize(GetActor(c), perm)
}

func allRoles() []Role {
	return []Role{
		RoleStudent, RoleStudentCoordinator, RoleFaculty, RoleHOD,
		RolePlacementOfficer, RolePrincipal, RoleAlumni,
		RoleClubOrganizer, RoleAdmin,
	}
}
