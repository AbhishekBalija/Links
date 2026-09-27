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

func TestNextStatus(t *testing.T) {
	cs := "cs-id"
	faculty := []Grant{{Role: auth.RoleFaculty, DepartmentID: cs}}
	tests := []struct {
		name   string
		grants []Grant
		event  Event
		want   Status
	}{
		{"faculty draft", faculty, Event{Status: StatusDraft, DepartmentID: &cs}, StatusSubmitted},
		{"faculty after HOD changes", faculty, Event{Status: StatusHODChangesRequested, DepartmentID: &cs}, StatusSubmitted},
		{"faculty after final changes", faculty, Event{Status: StatusFinalChangesRequested, DepartmentID: &cs}, StatusHODApproved},
		{"HOD's own department", []Grant{{Role: auth.RoleHOD, DepartmentID: cs}}, Event{Status: StatusDraft, DepartmentID: &cs}, StatusHODApproved},
		{"officer training", []Grant{{Role: auth.RolePlacementOfficer}}, Event{Status: StatusDraft, EventType: TypeTraining}, StatusHODApproved},
		{"principal", []Grant{{Role: auth.RolePrincipal}}, Event{Status: StatusDraft}, StatusPublished},
		{"admin after final changes", []Grant{{Role: auth.RoleAdmin}}, Event{Status: StatusFinalChangesRequested}, StatusPublished},
	}
	for _, test := range tests {
		if got := nextStatus(test.grants, test.event); got != test.want {
			t.Errorf("%s: got %s, want %s", test.name, got, test.want)
		}
	}
}

func TestAfterDecision(t *testing.T) {
	want := map[Stage]map[Decision]Status{
		StageHOD:   {DecisionApprove: StatusHODApproved, DecisionRequestChanges: StatusHODChangesRequested, DecisionReject: StatusHODRejected},
		StageFinal: {DecisionApprove: StatusPublished, DecisionRequestChanges: StatusFinalChangesRequested, DecisionReject: StatusFinalRejected},
	}
	for stage, decisions := range want {
		for decision, status := range decisions {
			if got := afterDecision(stage, decision); got != status {
				t.Errorf("%s %s: got %s, want %s", stage, decision, got, status)
			}
		}
	}
}
