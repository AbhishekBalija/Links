package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

type sentBackItem struct {
	Kind       string    `json:"kind"`
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	Note       *string   `json:"note"`
	SentBackBy string    `json:"sent_back_by"`
	SentBackAt time.Time `json:"sent_back_at"`
	IsEdit     bool      `json:"is_edit"`
}

type waitingItem struct {
	Kind      string    `json:"kind"`
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	WaitingOn string    `json:"waiting_on"`
	Since     time.Time `json:"since"`
	IsEdit    bool      `json:"is_edit"`
}

type myWork struct {
	SentBack        []sentBackItem `json:"sent_back"`
	SentBackHasMore bool           `json:"sent_back_has_more"`
	Waiting         []waitingItem  `json:"waiting"`
	WaitingHasMore  bool           `json:"waiting_has_more"`
}

func myWorkOf(t *testing.T, h *apitest.Harness, token string) *myWork {
	t.Helper()
	response := h.Do(t, http.MethodGet, "/api/v1/dashboard", token, nil)
	expectStatus(t, "dashboard", response, http.StatusOK)
	var home struct {
		Data struct {
			MyWork *myWork `json:"my_work"`
		} `json:"data"`
	}
	response.Decode(t, &home)
	return home.Data.MyWork
}

func sentBackByTitle(work *myWork) map[string]sentBackItem {
	byTitle := map[string]sentBackItem{}
	for _, item := range work.SentBack {
		byTitle[item.Title] = item
	}
	return byTitle
}

func waitingByTitle(work *myWork) map[string]waitingItem {
	byTitle := map[string]waitingItem{}
	for _, item := range work.Waiting {
		byTitle[item.Title] = item
	}
	return byTitle
}

func eventAt(t *testing.T, h *apitest.Harness, token, department, title string) string {
	t.Helper()
	starts := time.Now().Add(10 * 24 * time.Hour).UTC().Truncate(time.Minute)
	return createdEvent(t, createEvent(t, h, token, eventBody(map[string]any{
		"title": title, "department_id": department, "draft": false,
		"starts_at": starts.Format(time.RFC3339), "ends_at": starts.Add(time.Hour).Format(time.RFC3339),
	}))).ID
}

func TestHomeShowsAnAuthorWhatWasSentBackAndWhatWaitsOnWhom(t *testing.T) {
	h := apitest.New(t)
	cs := h.DepartmentID(t, "CS")
	csAudience := []map[string]any{{"department_id": cs}}
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	colleague := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	hod := member(t, h, "Asha Rao", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	reader := student(t, h, "CS", 2023)

	quiet := myWorkOf(t, h, faculty.Token)
	if quiet == nil || len(quiet.SentBack) != 0 || len(quiet.Waiting) != 0 {
		t.Fatalf("a new author's my_work = %+v, want present and empty", quiet)
	}
	if got := myWorkOf(t, h, reader.Token); got != nil {
		t.Fatalf("a student gets my_work: %+v", got)
	}

	// Two notices and two events, all waiting on the CS HOD.
	rejectedID, _ := createdStatus(t, publish(t, h, faculty.Token, map[string]any{"title": "Lab moved", "audience": csAudience}))
	waitingNoticeID, _ := createdStatus(t, publish(t, h, faculty.Token, map[string]any{"title": "Guest lecture", "audience": csAudience}))
	changesID := eventAt(t, h, faculty.Token, cs, "Robotics workshop")
	approvedID := eventAt(t, h, faculty.Token, cs, "Coding contest")
	createdStatus(t, publish(t, h, colleague.Token, map[string]any{"title": "Not mine", "audience": csAudience}))
	eventAt(t, h, colleague.Token, cs, "Not mine either")

	work := myWorkOf(t, h, faculty.Token)
	if len(work.SentBack) != 0 || len(work.Waiting) != 4 {
		t.Fatalf("my_work = %+v, want 4 waiting and nothing sent back", work)
	}
	waiting := waitingByTitle(work)
	for _, title := range []string{"Lab moved", "Guest lecture", "Robotics workshop", "Coding contest"} {
		if item := waiting[title]; item.WaitingOn != "CS HOD" || item.Since.IsZero() || item.IsEdit {
			t.Errorf("%q = %+v, want waiting on the CS HOD since it was submitted", title, item)
		}
	}
	if waiting["Lab moved"].Kind != "announcement" || waiting["Robotics workshop"].Kind != "event" || waiting["Robotics workshop"].ID != changesID {
		t.Errorf("waiting = %+v, want kinds announcement and event with their IDs", waiting)
	}
	for i := 1; i < len(work.Waiting); i++ {
		if work.Waiting[i].Since.Before(work.Waiting[i-1].Since) {
			t.Fatalf("waiting is not longest first: %+v", work.Waiting)
		}
	}

	// The HOD sends one notice and one event back, and approves the other event.
	expectStatus(t, "reject notice", review(t, h, hod.Token, rejectedID, "reject", "Add the room number."), http.StatusOK)
	expectStatus(t, "event changes", hodReview(t, h, hod.Token, changesID, "request_changes", "Add the room booking."), http.StatusOK)
	expectStatus(t, "event approved", hodReview(t, h, hod.Token, approvedID, "approve", ""), http.StatusOK)

	work = myWorkOf(t, h, faculty.Token)
	sentBack := sentBackByTitle(work)
	if len(sentBack) != 2 {
		t.Fatalf("sent back = %+v, want the notice and the event", work.SentBack)
	}
	if item := sentBack["Lab moved"]; item.Kind != "announcement" || item.ID != rejectedID || item.Note == nil || *item.Note != "Add the room number." ||
		item.SentBackBy != "Asha Rao" || item.SentBackAt.IsZero() || item.IsEdit {
		t.Errorf("sent-back notice = %+v, want Asha Rao's note", item)
	}
	if item := sentBack["Robotics workshop"]; item.Kind != "event" || item.ID != changesID || item.Note == nil ||
		*item.Note != "Add the room booking." || item.SentBackBy != "Asha Rao" || item.SentBackAt.IsZero() {
		t.Errorf("sent-back event = %+v, want Asha Rao's note", item)
	}
	waiting = waitingByTitle(work)
	if len(waiting) != 2 {
		t.Fatalf("waiting = %+v, want the notice and the approved event", work.Waiting)
	}
	if item := waiting["Coding contest"]; item.WaitingOn != "Principal or admin" || item.Kind != "event" {
		t.Errorf("approved event = %+v, want it waiting on the principal or an admin", item)
	}
	if waiting["Coding contest"].Since.Before(waiting["Guest lecture"].Since) {
		t.Errorf("an event the HOD approved waits since the approval, so after the notice was submitted")
	}
	if len(myWorkOf(t, h, colleague.Token).Waiting) != 2 {
		t.Errorf("the colleague sees someone else's work")
	}

	// Fixing and resubmitting moves an item from sent back to waiting.
	expectStatus(t, "edit event", h.Do(t, http.MethodPatch, "/api/v1/events/"+changesID, faculty.Token, map[string]any{"location": "Seminar Hall (booked)"}), http.StatusOK)
	expectStatus(t, "resubmit event", submitEvent(t, h, faculty.Token, changesID), http.StatusOK)
	work = myWorkOf(t, h, faculty.Token)
	if _, still := sentBackByTitle(work)["Robotics workshop"]; still || len(work.SentBack) != 1 {
		t.Errorf("sent back = %+v, want only the notice after the event was resubmitted", work.SentBack)
	}
	if item := waitingByTitle(work)["Robotics workshop"]; item.WaitingOn != "CS HOD" {
		t.Errorf("resubmitted event = %+v, want it back with the CS HOD", item)
	}

	// The principal's final decision ends the wait; a rejected event is final, not a to-do.
	expectStatus(t, "final approval", finalReview(t, h, principal.Token, approvedID, "approve", ""), http.StatusOK)
	rejectedEvent := eventAt(t, h, faculty.Token, cs, "Late night party")
	expectStatus(t, "event rejected", hodReview(t, h, hod.Token, rejectedEvent, "reject", "Not this term."), http.StatusOK)
	work = myWorkOf(t, h, faculty.Token)
	if _, listed := waitingByTitle(work)["Coding contest"]; listed {
		t.Errorf("a published event still waits: %+v", work.Waiting)
	}
	if _, listed := sentBackByTitle(work)["Late night party"]; listed {
		t.Errorf("a rejected event is listed as sent back: %+v", work.SentBack)
	}

	// Fixing the notice resubmits it to the HOD, who approves it.
	expectStatus(t, "fix notice", h.Do(t, http.MethodPatch, "/api/v1/announcements/"+rejectedID, faculty.Token, map[string]any{
		"title": "Lab moved to room 12", "body": "Block B.", "category": "department", "audience": csAudience,
	}), http.StatusOK)
	expectStatus(t, "resubmit notice", h.Do(t, http.MethodPost, "/api/v1/announcements/"+rejectedID+"/submit-for-approval", faculty.Token, nil), http.StatusOK)
	expectStatus(t, "approve notice", review(t, h, hod.Token, rejectedID, "approve", ""), http.StatusOK)
	expectStatus(t, "approve other notice", review(t, h, hod.Token, waitingNoticeID, "approve", ""), http.StatusOK)

	// An edit to a published notice waits like any other, and is marked as an edit.
	expectStatus(t, "edit published", edit(t, h, faculty.Token, rejectedID, "Lab moved to room 14", csAudience), http.StatusOK)
	item := waitingByTitle(myWorkOf(t, h, faculty.Token))["Lab moved to room 12"]
	if item.Kind != "announcement" || !item.IsEdit || item.WaitingOn != "CS HOD" || item.Since.IsZero() {
		t.Errorf("waiting edit = %+v, want an edit waiting on the CS HOD", item)
	}
	expectStatus(t, "reject the edit", review(t, h, hod.Token, rejectedID, "reject", "Keep room 12."), http.StatusOK)
	back := sentBackByTitle(myWorkOf(t, h, faculty.Token))["Lab moved to room 12"]
	if !back.IsEdit || back.Note == nil || *back.Note != "Keep room 12." || back.SentBackBy != "Asha Rao" {
		t.Errorf("sent-back edit = %+v, want an edit with the HOD's note", back)
	}
}

func TestStudentCoordinatorsGetTheSameSectionAndOtherUsersDoNot(t *testing.T) {
	h := apitest.New(t)
	cs := h.DepartmentID(t, "CS")
	coordinator := h.SeedUser(t, apitest.UserSeed{
		Roles:   []apitest.RoleSeed{{Role: "student"}, {Role: "student_coordinator", DepartmentCode: "CS"}},
		Student: &apitest.StudentSeed{DepartmentCode: "CS", BatchYear: 2023},
	})
	hod := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	eventAt(t, h, coordinator.Token, cs, "Hack night")

	work := myWorkOf(t, h, coordinator.Token)
	if work == nil || len(work.Waiting) != 1 || work.Waiting[0].Kind != "event" || work.Waiting[0].WaitingOn != "CS HOD" {
		t.Fatalf("coordinator my_work = %+v, want the event waiting on the CS HOD", work)
	}
	plain := student(t, h, "CS", 2023)
	if got := myWorkOf(t, h, plain.Token); got != nil {
		t.Errorf("a plain student gets my_work: %+v", got)
	}
	// An HOD's own proposals are theirs too: the HOD's Home still reviews others' work.
	if got := myWorkOf(t, h, hod.Token); got == nil {
		t.Errorf("an HOD who can post has no my_work")
	}
}

func TestAnEventInADepartmentWithoutAnHODWaitsOnThePrincipalOrAdmin(t *testing.T) {
	h := apitest.New(t)
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "EC"}}})
	eventAt(t, h, faculty.Token, h.DepartmentID(t, "EC"), "EC seminar")

	work := myWorkOf(t, h, faculty.Token)
	if len(work.Waiting) != 1 || work.Waiting[0].WaitingOn != "Principal or admin" {
		t.Fatalf("my_work = %+v, want waiting on the principal or an admin", work)
	}
}

func TestMyWorkListsAreCappedAtTenWithAHasMoreFlag(t *testing.T) {
	h := apitest.New(t)
	cs := h.DepartmentID(t, "CS")
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	for i := 0; i < 11; i++ {
		eventAt(t, h, faculty.Token, cs, fmt.Sprintf("Event %02d", i))
	}

	work := myWorkOf(t, h, faculty.Token)
	if len(work.Waiting) != 10 || !work.WaitingHasMore || work.SentBackHasMore {
		t.Fatalf("my_work = %d waiting, has_more %v, sent_back_has_more %v; want 10, true, false", len(work.Waiting), work.WaitingHasMore, work.SentBackHasMore)
	}
}
