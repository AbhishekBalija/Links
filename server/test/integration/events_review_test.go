package integration

import (
	"net/http"
	"testing"
	"time"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

type reviewedEvent struct {
	eventItem
	Stage   string `json:"stage"`
	Reviews []struct {
		Stage        string `json:"stage"`
		Decision     string `json:"decision"`
		Note         string `json:"note"`
		ReviewerName string `json:"reviewer_name"`
	} `json:"reviews"`
}

func submitEvent(t *testing.T, h *apitest.Harness, token, id string) apitest.Response {
	t.Helper()
	return h.Do(t, http.MethodPost, "/api/v1/events/"+id+"/submit-for-approval", token, nil)
}

func eventStatus(t *testing.T, response apitest.Response) string {
	t.Helper()
	if response.Status != http.StatusOK && response.Status != http.StatusCreated {
		t.Fatalf("status = %d, want 200 or 201: %s", response.Status, response.Body)
	}
	var body struct {
		Data eventItem `json:"data"`
	}
	response.Decode(t, &body)
	return body.Data.Status
}

func hodReview(t *testing.T, h *apitest.Harness, token, id, decision, note string) apitest.Response {
	t.Helper()
	return h.Do(t, http.MethodPatch, "/api/v1/events/"+id+"/hod-review", token, map[string]string{"decision": decision, "note": note})
}

func finalReview(t *testing.T, h *apitest.Harness, token, id, decision, note string) apitest.Response {
	t.Helper()
	return h.Do(t, http.MethodPatch, "/api/v1/events/"+id+"/final-approval", token, map[string]string{"decision": decision, "note": note})
}

func reviewQueue(t *testing.T, h *apitest.Harness, token string) map[string]reviewedEvent {
	t.Helper()
	response := h.Do(t, http.MethodGet, "/api/v1/events/reviews", token, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("reviews status = %d, want %d: %s", response.Status, http.StatusOK, response.Body)
	}
	var list struct {
		Data []reviewedEvent `json:"data"`
	}
	response.Decode(t, &list)
	byID := map[string]reviewedEvent{}
	for _, item := range list.Data {
		byID[item.ID] = item
	}
	return byID
}

func expectStatus(t *testing.T, name string, response apitest.Response, want int) {
	t.Helper()
	if response.Status != want {
		t.Fatalf("%s: status = %d, want %d: %s", name, response.Status, want, response.Body)
	}
}

func TestCoordinatorEventGoesThroughHODThenFinalApproval(t *testing.T) {
	h := apitest.New(t)
	cs := h.DepartmentID(t, "CS")
	coordinator := h.SeedUser(t, apitest.UserSeed{
		Roles:   []apitest.RoleSeed{{Role: "student"}, {Role: "student_coordinator", DepartmentCode: "CS"}},
		Student: &apitest.StudentSeed{DepartmentCode: "CS", BatchYear: 2023},
	})
	csHOD := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	ecHOD := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "EC"}}})
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})

	created := createEvent(t, h, coordinator.Token, eventBody(map[string]any{"department_id": cs, "draft": false}))
	if status := eventStatus(t, created); status != "submitted" {
		t.Fatalf("status after submit = %q, want submitted", status)
	}
	id := createdEvent(t, created).ID

	if _, ok := reviewQueue(t, h, ecHOD.Token)[id]; ok {
		t.Fatal("EC HOD's queue shows a CS event")
	}
	if item, ok := reviewQueue(t, h, csHOD.Token)[id]; !ok || item.Stage != "hod" {
		t.Fatalf("CS HOD's queue item = %+v, want it at the hod stage", item)
	}
	if _, ok := reviewQueue(t, h, principal.Token)[id]; ok {
		t.Fatal("the principal sees a CS event before its HOD reviewed it")
	}
	expectStatus(t, "EC HOD review", hodReview(t, h, ecHOD.Token, id, "approve", ""), http.StatusForbidden)
	expectStatus(t, "principal at the HOD stage", hodReview(t, h, principal.Token, id, "approve", ""), http.StatusForbidden)
	expectStatus(t, "final approval too early", finalReview(t, h, principal.Token, id, "approve", ""), http.StatusConflict)
	expectStatus(t, "changes without a note", hodReview(t, h, csHOD.Token, id, "request_changes", ""), http.StatusBadRequest)
	expectStatus(t, "unknown decision", hodReview(t, h, csHOD.Token, id, "maybe", "hmm"), http.StatusBadRequest)

	if status := eventStatus(t, hodReview(t, h, csHOD.Token, id, "request_changes", "Add the room booking.")); status != "hod_changes_requested" {
		t.Fatalf("status = %q, want hod_changes_requested", status)
	}
	attention := myEvents(t, h, coordinator.Token, "attention")
	if len(attention) != 1 || attention[0].ID != id {
		t.Fatalf("attention = %+v, want the sent-back event", attention)
	}

	expectStatus(t, "edit after changes requested", h.Do(t, http.MethodPatch, "/api/v1/events/"+id, coordinator.Token, map[string]any{"location": "Seminar Hall (booked)"}), http.StatusOK)
	if status := eventStatus(t, submitEvent(t, h, coordinator.Token, id)); status != "submitted" {
		t.Fatalf("resubmitted status = %q, want submitted (back to the HOD)", status)
	}
	item := reviewQueue(t, h, csHOD.Token)[id]
	if len(item.Reviews) != 1 || item.Reviews[0].Note != "Add the room booking." || item.Reviews[0].Stage != "hod" {
		t.Fatalf("reviews = %+v, want the earlier HOD note", item.Reviews)
	}

	if status := eventStatus(t, hodReview(t, h, csHOD.Token, id, "approve", "")); status != "hod_approved" {
		t.Fatalf("status = %q, want hod_approved", status)
	}
	if _, ok := reviewQueue(t, h, csHOD.Token)[id]; ok {
		t.Fatal("approved event is still in the HOD's queue")
	}
	if item, ok := reviewQueue(t, h, principal.Token)[id]; !ok || item.Stage != "final" {
		t.Fatalf("principal's queue item = %+v, want it at the final stage", item)
	}
	expectStatus(t, "HOD reviewing twice", hodReview(t, h, csHOD.Token, id, "approve", ""), http.StatusConflict)
	expectStatus(t, "HOD at the final stage", finalReview(t, h, csHOD.Token, id, "approve", ""), http.StatusForbidden)

	if status := eventStatus(t, finalReview(t, h, principal.Token, id, "approve", "Good to go.")); status != "published" {
		t.Fatalf("status = %q, want published", status)
	}
	var reviews, audits int
	h.DB().Raw(`SELECT count(*) FROM event_reviews WHERE event_id = ?`, id).Scan(&reviews)
	h.DB().Raw(`SELECT count(*) FROM audit_logs WHERE resource_id = ? AND action IN ('event_submitted', 'event_reviewed')`, id).Scan(&audits)
	if reviews != 3 || audits != 5 {
		t.Fatalf("reviews = %d, audits = %d; want 3 decisions and 5 audited steps", reviews, audits)
	}
}

func TestFinalChangesReturnToTheFinalStage(t *testing.T) {
	h := apitest.New(t)
	cs := h.DepartmentID(t, "CS")
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	csHOD := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})

	id := createdEvent(t, createEvent(t, h, faculty.Token, eventBody(map[string]any{"department_id": cs, "draft": false}))).ID
	eventStatus(t, hodReview(t, h, csHOD.Token, id, "approve", ""))
	if status := eventStatus(t, finalReview(t, h, admin.Token, id, "request_changes", "Move it off the exam week.")); status != "final_changes_requested" {
		t.Fatalf("status = %q, want final_changes_requested", status)
	}
	later := time.Now().Add(20 * 24 * time.Hour).UTC()
	expectStatus(t, "edit", h.Do(t, http.MethodPatch, "/api/v1/events/"+id, faculty.Token, map[string]any{
		"starts_at": later.Format(time.RFC3339), "ends_at": later.Add(time.Hour).Format(time.RFC3339),
	}), http.StatusOK)
	if status := eventStatus(t, submitEvent(t, h, faculty.Token, id)); status != "hod_approved" {
		t.Fatalf("resubmitted status = %q, want hod_approved (back to final approval)", status)
	}
	if _, ok := reviewQueue(t, h, csHOD.Token)[id]; ok {
		t.Fatal("the HOD is asked to review again after final changes")
	}
	if status := eventStatus(t, finalReview(t, h, admin.Token, id, "reject", "Clashes with the fest.")); status != "final_rejected" {
		t.Fatalf("status = %q, want final_rejected", status)
	}
	expectStatus(t, "resubmitting a rejected event", submitEvent(t, h, faculty.Token, id), http.StatusConflict)
	expectStatus(t, "editing a rejected event", h.Do(t, http.MethodPatch, "/api/v1/events/"+id, faculty.Token, map[string]any{"title": "Try again"}), http.StatusConflict)
}

func TestProposerPathsSkipStagesTheyDontNeed(t *testing.T) {
	h := apitest.New(t)
	cs := h.DepartmentID(t, "CS")
	hod := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	officer := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "placement_officer"}}})

	hodEvent := createEvent(t, h, hod.Token, eventBody(map[string]any{"department_id": cs, "draft": false}))
	if status := eventStatus(t, hodEvent); status != "hod_approved" {
		t.Errorf("HOD's own event = %q, want hod_approved", status)
	}
	if _, ok := reviewQueue(t, h, principal.Token)[createdEvent(t, hodEvent).ID]; !ok {
		t.Error("the HOD's event is not waiting for final approval")
	}
	if status := eventStatus(t, createEvent(t, h, principal.Token, eventBody(map[string]any{"draft": false}))); status != "published" {
		t.Errorf("principal's event = %q, want published", status)
	}
	if status := eventStatus(t, createEvent(t, h, admin.Token, eventBody(map[string]any{"department_id": cs, "draft": false}))); status != "published" {
		t.Errorf("admin's event = %q, want published", status)
	}
	training := createEvent(t, h, officer.Token, eventBody(map[string]any{"event_type": "training", "department_id": cs, "draft": false}))
	if status := eventStatus(t, training); status != "hod_approved" {
		t.Errorf("training event = %q, want hod_approved (no HOD stage)", status)
	}
	if _, ok := reviewQueue(t, h, hod.Token)[createdEvent(t, training).ID]; ok {
		t.Error("a training event waits for the CS HOD")
	}
}

func TestPrincipalDoesTheHODStageWhenTheDepartmentHasNoHOD(t *testing.T) {
	h := apitest.New(t)
	me := h.DepartmentID(t, "ME")
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "ME"}}})
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})

	id := createdEvent(t, createEvent(t, h, faculty.Token, eventBody(map[string]any{"department_id": me, "draft": false}))).ID
	if item, ok := reviewQueue(t, h, principal.Token)[id]; !ok || item.Stage != "hod" {
		t.Fatalf("principal's queue item = %+v, want the HOD stage of an HOD-less department", item)
	}
	if status := eventStatus(t, hodReview(t, h, principal.Token, id, "approve", "")); status != "hod_approved" {
		t.Fatalf("status = %q, want hod_approved", status)
	}
	if status := eventStatus(t, finalReview(t, h, admin.Token, id, "approve", "")); status != "published" {
		t.Fatalf("status = %q, want published", status)
	}
}

func TestNobodyReviewsTheirOwnProposalAndSecondDecisionsConflict(t *testing.T) {
	h := apitest.New(t)
	cs := h.DepartmentID(t, "CS")
	proposer := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})

	id := createdEvent(t, createEvent(t, h, proposer.Token, eventBody(map[string]any{"department_id": cs, "draft": false}))).ID
	// The proposer becomes CS HOD while their event waits.
	h.DB().Exec(`INSERT INTO role_assignments (user_id, role, scope_type, scope_id, starts_at) VALUES (?, 'hod', 'department', ?, now())`, proposer.ID, cs)
	hodToken := h.TokenFor(t, proposer.ID, "faculty", "hod")
	if _, ok := reviewQueue(t, h, hodToken)[id]; ok {
		t.Error("the proposer's own event is in their review queue")
	}
	expectStatus(t, "own HOD review", hodReview(t, h, hodToken, id, "approve", ""), http.StatusForbidden)

	// Remove the HOD role again so the principal can do the HOD stage.
	h.DB().Exec(`DELETE FROM role_assignments WHERE user_id = ? AND role = 'hod'`, proposer.ID)
	eventStatus(t, hodReview(t, h, principal.Token, id, "approve", ""))
	eventStatus(t, finalReview(t, h, principal.Token, id, "approve", ""))
	expectStatus(t, "second final decision", finalReview(t, h, admin.Token, id, "reject", "Too late"), http.StatusConflict)
}

func TestSubmittingChecksDatesAndOwnership(t *testing.T) {
	h := apitest.New(t)
	cs := h.DepartmentID(t, "CS")
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	other := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	past := time.Now().Add(-2 * time.Hour).UTC()

	stale := createdEvent(t, createEvent(t, h, faculty.Token, eventBody(map[string]any{
		"department_id": cs, "starts_at": past.Format(time.RFC3339), "ends_at": past.Add(time.Hour).Format(time.RFC3339),
	}))).ID
	expectStatus(t, "past event", submitEvent(t, h, faculty.Token, stale), http.StatusBadRequest)
	expectStatus(t, "past event on create", createEvent(t, h, faculty.Token, eventBody(map[string]any{
		"department_id": cs, "draft": false, "starts_at": past.Format(time.RFC3339), "ends_at": past.Add(time.Hour).Format(time.RFC3339),
	})), http.StatusBadRequest)

	draft := createdEvent(t, createEvent(t, h, faculty.Token, eventBody(map[string]any{"department_id": cs}))).ID
	expectStatus(t, "someone else's draft", submitEvent(t, h, other.Token, draft), http.StatusNotFound)
	eventStatus(t, submitEvent(t, h, faculty.Token, draft))
	expectStatus(t, "submitting twice", submitEvent(t, h, faculty.Token, draft), http.StatusConflict)
	expectStatus(t, "editing while waiting", h.Do(t, http.MethodPatch, "/api/v1/events/"+draft, faculty.Token, map[string]any{"title": "Changed"}), http.StatusConflict)
	expectStatus(t, "queue for faculty", h.Do(t, http.MethodGet, "/api/v1/events/reviews", faculty.Token, nil), http.StatusForbidden)
}
