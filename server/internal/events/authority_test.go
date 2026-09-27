package events

import (
	"testing"

	"github.com/AbhishekBalija/Links/server/internal/auth"
)

func TestProposalProblem(t *testing.T) {
	cs, ec := "cs-id", "ec-id"
	tests := []struct {
		name       string
		grants     []Grant
		eventType  Type
		department *string
		wantField  string
	}{
		{"admin, college-wide", []Grant{{Role: auth.RoleAdmin}}, TypeTalk, nil, ""},
		{"principal, any department", []Grant{{Role: auth.RolePrincipal}}, TypeTalk, &ec, ""},
		{"HOD, own", []Grant{{Role: auth.RoleHOD, DepartmentID: cs}}, TypeTalk, &cs, ""},
		{"HOD, other", []Grant{{Role: auth.RoleHOD, DepartmentID: cs}}, TypeTalk, &ec, "department_id"},
		{"coordinator, own", []Grant{{Role: auth.RoleStudent}, {Role: auth.RoleStudentCoordinator, DepartmentID: cs}}, TypeCultural, &cs, ""},
		{"faculty, college-wide", []Grant{{Role: auth.RoleFaculty, DepartmentID: cs}}, TypeTalk, nil, "department_id"},
		{"officer, training anywhere", []Grant{{Role: auth.RolePlacementOfficer}}, TypeTraining, &ec, ""},
		{"officer, a talk", []Grant{{Role: auth.RolePlacementOfficer}}, TypeTalk, nil, "event_type"},
		{"officer who also teaches CS, a CS talk", []Grant{{Role: auth.RolePlacementOfficer}, {Role: auth.RoleFaculty, DepartmentID: cs}}, TypeTalk, &cs, ""},
		{"student", []Grant{{Role: auth.RoleStudent}}, TypeTalk, &cs, "department_id"},
	}
	for _, test := range tests {
		field, problem := proposalProblem(test.grants, test.eventType, test.department)
		if field != test.wantField || (field == "") != (problem == "") {
			t.Errorf("%s: got %q %q, want field %q", test.name, field, problem, test.wantField)
		}
	}
}
