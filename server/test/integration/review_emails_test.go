package integration

import (
	"net/http"
	"testing"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

// Telling an author what a reviewer decided (#206): their post is out, or it
// came back with a note, so they needn't keep checking My posts.

func TestSendingBackAnAnnouncementEmailsTheAuthorTheNote(t *testing.T) {
	h := apitest.New(t)
	csAudience := []map[string]any{{"department_id": h.DepartmentID(t, "CS")}}
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	hod := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})

	id, _ := createdStatus(t, publish(t, h, faculty.Token, map[string]any{"title": "Lab 2 timings", "audience": csAudience}))
	expectStatus(t, "send back", review(t, h, hod.Token, id, "reject", "Add the room number"), http.StatusOK)

	letters := h.Outbox.ReviewsTo(faculty.Email)
	if len(letters) != 1 {
		t.Fatalf("letters = %+v, want one", letters)
	}
	got := letters[0]
	if got.Outcome != "sent_back" || got.Kind != "announcement" || got.Title != "Lab 2 timings" || got.Note != "Add the room number" || got.ReviewerName == "" || got.Path == "" {
		t.Errorf("letter = %+v, want the announcement sent back with the note and the reviewer", got)
	}
}

func TestApprovingAnAnnouncementEmailsTheAuthorItIsOut(t *testing.T) {
	h := apitest.New(t)
	csAudience := []map[string]any{{"department_id": h.DepartmentID(t, "CS")}}
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	hod := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})

	id, _ := createdStatus(t, publish(t, h, faculty.Token, map[string]any{"title": "Lab 2 timings", "audience": csAudience}))
	expectStatus(t, "approve", review(t, h, hod.Token, id, "approve", ""), http.StatusOK)

	letters := h.Outbox.ReviewsTo(faculty.Email)
	if len(letters) != 1 || letters[0].Outcome != "published" || letters[0].Title != "Lab 2 timings" {
		t.Errorf("letters = %+v, want one saying it is published", letters)
	}
}

func TestAnEventEmailsItsProposerAtEachDecisionThatReachesThem(t *testing.T) {
	h := apitest.New(t)
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	hod := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	id := createdEvent(t, createEvent(t, h, faculty.Token, eventBody(map[string]any{"department_id": h.DepartmentID(t, "CS"), "draft": false}))).ID

	// Sent back by the HOD: the proposer has something to fix.
	eventStatus(t, hodReview(t, h, hod.Token, id, "request_changes", "Book Seminar hall 1 instead"))
	letters := h.Outbox.ReviewsTo(faculty.Email)
	if len(letters) != 1 || letters[0].Outcome != "sent_back" || letters[0].Kind != "event" || letters[0].Note != "Book Seminar hall 1 instead" {
		t.Fatalf("after send back, letters = %+v, want one sent back with the note", letters)
	}

	// The HOD's approval only moves it on to the principal: nothing to tell.
	expectStatus(t, "resubmit", submitEvent(t, h, faculty.Token, id), http.StatusOK)
	eventStatus(t, hodReview(t, h, hod.Token, id, "approve", ""))
	if letters := h.Outbox.ReviewsTo(faculty.Email); len(letters) != 1 {
		t.Fatalf("after the HOD approved, letters = %+v, want still one", letters)
	}

	eventStatus(t, finalReview(t, h, principal.Token, id, "approve", ""))
	letters = h.Outbox.ReviewsTo(faculty.Email)
	if len(letters) != 2 || letters[1].Outcome != "published" {
		t.Errorf("after final approval, letters = %+v, want a second saying it is published", letters)
	}
}

func TestRejectingAnEventEmailsTheReason(t *testing.T) {
	h := apitest.New(t)
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	hod := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	id := createdEvent(t, createEvent(t, h, faculty.Token, eventBody(map[string]any{"department_id": h.DepartmentID(t, "CS"), "draft": false}))).ID

	eventStatus(t, hodReview(t, h, hod.Token, id, "reject", "Exams that week"))
	letters := h.Outbox.ReviewsTo(faculty.Email)
	if len(letters) != 1 || letters[0].Outcome != "rejected" || letters[0].Note != "Exams that week" {
		t.Errorf("letters = %+v, want one rejection with the reason", letters)
	}
}
