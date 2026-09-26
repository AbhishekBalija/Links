package announcements

import "github.com/AbhishekBalija/Links/server/internal/auth"

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
