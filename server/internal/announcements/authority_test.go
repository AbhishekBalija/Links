package announcements

import (
	"testing"

	"github.com/AbhishekBalija/Links/server/internal/auth"
)

// Cases follow ADR 0017: who publishes directly, and who approves otherwise.
func TestDecidePublishing(t *testing.T) {
	const cs, ec = "dept-cs", "dept-ec"
	csOnly := []AudienceRule{{DepartmentID: ptr(cs)}}
	csFinalYear := []AudienceRule{{DepartmentID: ptr(cs), BatchYear: ptr(2022)}, {DepartmentID: ptr(cs), Role: ptr(auth.RoleFaculty)}}
	csAndEC := []AudienceRule{{DepartmentID: ptr(cs)}, {DepartmentID: ptr(ec)}}
	wholeCollege := []AudienceRule{}
	allStudents := []AudienceRule{{Role: ptr(auth.RoleStudent)}}

	direct := Decision{PublishDirectly: true}
	hodOf := func(department string) Decision { return Decision{ApproverDepartmentID: department} }
	principalOrAdmin := Decision{}

	tests := []struct {
		name     string
		grants   []Grant
		category Category
		audience []AudienceRule
		want     Decision
	}{
		{"principal, whole college", []Grant{{Role: auth.RolePrincipal}}, CategoryOfficial, wholeCollege, direct},
		{"admin, another department", []Grant{{Role: auth.RoleAdmin}}, CategoryDepartment, csOnly, direct},
		{"HOD, own department", []Grant{{Role: auth.RoleHOD, DepartmentID: cs}}, CategoryDepartment, csOnly, direct},
		{"HOD, own department with batch and role rules", []Grant{{Role: auth.RoleHOD, DepartmentID: cs}}, CategoryDepartment, csFinalYear, direct},
		{"HOD, other department", []Grant{{Role: auth.RoleHOD, DepartmentID: ec}}, CategoryDepartment, csOnly, hodOf(cs)},
		{"HOD, several departments", []Grant{{Role: auth.RoleHOD, DepartmentID: cs}}, CategoryDepartment, csAndEC, principalOrAdmin},
		{"HOD, whole college", []Grant{{Role: auth.RoleHOD, DepartmentID: cs}}, CategoryOfficial, wholeCollege, principalOrAdmin},
		{"HOD, rule without a department", []Grant{{Role: auth.RoleHOD, DepartmentID: cs}}, CategoryOfficial, allStudents, principalOrAdmin},
		{"HOD granted globally has no department", []Grant{{Role: auth.RoleHOD}}, CategoryDepartment, csOnly, hodOf(cs)},
		{"faculty, own department", []Grant{{Role: auth.RoleFaculty, DepartmentID: cs}}, CategoryDepartment, csOnly, hodOf(cs)},
		{"faculty, whole college", []Grant{{Role: auth.RoleFaculty}}, CategoryOfficial, wholeCollege, principalOrAdmin},
		{"student coordinator, own department", []Grant{{Role: auth.RoleStudentCoordinator, DepartmentID: cs}}, CategoryDepartment, csOnly, hodOf(cs)},
		{"placement officer, placement, whole college", []Grant{{Role: auth.RolePlacementOfficer}}, CategoryPlacement, wholeCollege, direct},
		{"placement officer, placement, one department", []Grant{{Role: auth.RolePlacementOfficer}}, CategoryPlacement, csOnly, direct},
		{"HOD, placement to own department", []Grant{{Role: auth.RoleHOD, DepartmentID: cs}}, CategoryPlacement, csOnly, direct},
		{"faculty and HOD of another department", []Grant{{Role: auth.RoleFaculty, DepartmentID: cs}, {Role: auth.RoleHOD, DepartmentID: ec}}, CategoryDepartment, csOnly, hodOf(cs)},
		{"student, no posting role", []Grant{{Role: auth.RoleStudent}}, CategoryDepartment, csOnly, hodOf(cs)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := DecidePublishing(test.grants, test.category, test.audience)
			if got != test.want {
				t.Errorf("DecidePublishing() = %+v, want %+v", got, test.want)
			}
		})
	}
}

func TestCanPost(t *testing.T) {
	tests := []struct {
		name     string
		grants   []Grant
		category Category
		want     bool
	}{
		{"student cannot post", []Grant{{Role: auth.RoleStudent}}, CategoryDepartment, false},
		{"alumni cannot post", []Grant{{Role: auth.RoleAlumni}}, CategoryOfficial, false},
		{"faculty posts department notices", []Grant{{Role: auth.RoleFaculty, DepartmentID: "dept-cs"}}, CategoryDepartment, true},
		{"student coordinator posts", []Grant{{Role: auth.RoleStudentCoordinator}}, CategoryOfficial, true},
		{"placement officer posts placement", []Grant{{Role: auth.RolePlacementOfficer}}, CategoryPlacement, true},
		{"placement officer cannot post other categories", []Grant{{Role: auth.RolePlacementOfficer}}, CategoryOfficial, false},
		{"principal posts anything", []Grant{{Role: auth.RolePrincipal}}, CategoryPlacement, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := CanPost(test.grants, test.category); got != test.want {
				t.Errorf("CanPost() = %v, want %v", got, test.want)
			}
		})
	}
}

func ptr[T any](value T) *T { return &value }
