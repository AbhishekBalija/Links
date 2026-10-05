package announcements

import (
	"github.com/AbhishekBalija/Links/server/internal/auth"
	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
)

// Grant is one role the author currently holds. DepartmentID is empty for a
// global role and set for a Department-scoped one.
type Grant struct {
	Role         auth.Role
	DepartmentID string
}

// Decision says whether an Announcement publishes straight away and, if not,
// who approves it. An empty ApproverDepartmentID means the principal or an admin.
type Decision struct {
	PublishDirectly      bool
	ApproverDepartmentID string
}

// CanPost reports whether the author may post an Announcement of this category
// at all (docs/auth.md). The placement officer only posts placement notices.
func CanPost(grants []Grant, category Category) bool {
	for _, grant := range grants {
		switch grant.Role {
		case auth.RolePrincipal, auth.RoleAdmin, auth.RoleHOD, auth.RoleFaculty, auth.RoleStudentCoordinator:
			return true
		case auth.RolePlacementOfficer:
			if category == CategoryPlacement {
				return true
			}
		}
	}
	return false
}

// DecidePublishing applies ADR 0017: an author with Publishing authority over
// the whole Audience publishes directly; anyone else needs Announcement
// approval from the Department's HOD (single-Department Audience) or from the
// principal or an admin (anything wider).
func DecidePublishing(grants []Grant, category Category, audience []AudienceRule) Decision {
	department, single := singleDepartment(audience)

	for _, grant := range grants {
		switch grant.Role {
		case auth.RolePrincipal, auth.RoleAdmin:
			return Decision{PublishDirectly: true}
		case auth.RolePlacementOfficer:
			if category == CategoryPlacement {
				return Decision{PublishDirectly: true}
			}
		case auth.RoleHOD:
			if single && grant.DepartmentID != "" && grant.DepartmentID == department {
				return Decision{PublishDirectly: true}
			}
		}
	}

	if single {
		return Decision{ApproverDepartmentID: department}
	}
	return Decision{}
}

// singleDepartment returns the Department an Audience is limited to, when every
// rule names the same one. An empty Audience means the whole college.
func singleDepartment(audience []AudienceRule) (string, bool) {
	if len(audience) == 0 {
		return "", false
	}
	department := ""
	for _, rule := range audience {
		if rule.DepartmentID == nil {
			return "", false
		}
		if department == "" {
			department = *rule.DepartmentID
		} else if *rule.DepartmentID != department {
			return "", false
		}
	}
	return department, true
}

// ReachProblem limits who a student coordinator's Announcement may reach
// (#209): Department notices, to their own Department's students (any batch,
// coordinators included). Anyone who also holds a wider posting role posts as
// that role. It returns the field at fault and why, or "" when it's fine.
func ReachProblem(grants []Grant, category Category, audience []AudienceRule) (string, string) {
	coordinatorOf := map[string]bool{}
	for _, grant := range grants {
		switch grant.Role {
		case auth.RolePrincipal, auth.RoleAdmin, auth.RoleHOD, auth.RoleFaculty, auth.RolePlacementOfficer:
			return "", ""
		case auth.RoleStudentCoordinator:
			coordinatorOf[grant.DepartmentID] = true
		}
	}
	if len(coordinatorOf) == 0 {
		return "", ""
	}
	if category != CategoryDepartment {
		return "category", "student coordinators post department notices"
	}
	if len(audience) == 0 {
		return "audience", "student coordinators post to their own department's students"
	}
	for _, rule := range audience {
		students := rule.Role != nil && (*rule.Role == auth.RoleStudent || *rule.Role == auth.RoleStudentCoordinator)
		if rule.DepartmentID == nil || !coordinatorOf[*rule.DepartmentID] || !students {
			return "audience", "student coordinators post to their own department's students"
		}
	}
	return "", ""
}

// reachRefused is ReachProblem as a validation error, or nil.
func reachRefused(grants []Grant, content announcementContent) error {
	if field, problem := ReachProblem(grants, content.Category, content.Audience); field != "" {
		return apperrors.NewValidation("invalid announcement", map[string]string{field: problem})
	}
	return nil
}
