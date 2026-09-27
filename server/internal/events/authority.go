package events

import "github.com/AbhishekBalija/Links/server/internal/auth"

// proposalProblem says why these grants can't propose this Event, as a field
// and a message, or returns empty strings when they can (ADR 0023):
//   - the principal and admins: any Event, for any Department or none;
//   - an HOD: their own Department;
//   - the placement officer: training only, for any Department or none;
//   - faculty and student coordinators: their own Department only.
func proposalProblem(grants []Grant, eventType Type, departmentID *string) (string, string) {
	officer := false
	for _, grant := range grants {
		switch grant.Role {
		case auth.RolePrincipal, auth.RoleAdmin:
			return "", ""
		case auth.RolePlacementOfficer:
			officer = true
			if eventType == TypeTraining {
				return "", ""
			}
		case auth.RoleHOD, auth.RoleFaculty, auth.RoleStudentCoordinator:
			if departmentID != nil && grant.DepartmentID == *departmentID {
				return "", ""
			}
		}
	}
	if officer && eventType != TypeTraining && !holdsDepartmentRole(grants) {
		return "event_type", "the placement officer proposes training events only"
	}
	if departmentID == nil {
		return "department_id", "choose your department; only the principal, admins and the placement officer propose college-wide events"
	}
	return "department_id", "you can propose events only for a department you belong to"
}

func holdsDepartmentRole(grants []Grant) bool {
	for _, grant := range grants {
		if grant.DepartmentID != "" && (grant.Role == auth.RoleHOD || grant.Role == auth.RoleFaculty || grant.Role == auth.RoleStudentCoordinator) {
			return true
		}
	}
	return false
}
