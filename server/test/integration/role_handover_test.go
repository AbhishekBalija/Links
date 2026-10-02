package integration

import (
	"net/http"
	"strings"
	"testing"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

// Ending a role withdraws or hands over the person's unfinished work
// (ADR 0028).

func csCoordinator(t *testing.T, h *apitest.Harness) apitest.User {
	t.Helper()
	return h.SeedUser(t, apitest.UserSeed{
		Roles:   []apitest.RoleSeed{{Role: "student"}, {Role: "student_coordinator", DepartmentCode: "CS"}},
		Student: &apitest.StudentSeed{DepartmentCode: "CS", BatchYear: 2023},
	})
}

// roleID finds the user's active assignment of the role.
func roleID(t *testing.T, h *apitest.Harness, token, userID, role string) string {
	t.Helper()
	for _, assignment := range listRoles(t, h, token, userID) {
		if assignment.Role == role && assignment.State == "active" {
			return assignment.ID
		}
	}
	t.Fatalf("user %s has no active %s role", userID, role)
	return ""
}

func endRole(t *testing.T, h *apitest.Harness, token, userID, assignmentID, organiserID string) apitest.Response {
	t.Helper()
	path := "/api/v1/admin/users/" + userID + "/roles/" + assignmentID
	if organiserID != "" {
		path += "?organiser_id=" + organiserID
	}
	return h.Do(t, http.MethodDelete, path, token, nil)
}

func TestEndingACoordinatorWithdrawsTheirWaitingAnnouncements(t *testing.T) {
	h := apitest.New(t)
	hod := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	coordinator := csCoordinator(t, h)
	cs := h.DepartmentID(t, "CS")
	csAudience := []map[string]any{{"department_id": cs}}

	waitingID, status := createdStatus(t, publish(t, h, coordinator.Token, map[string]any{"title": "Waiting notice", "audience": csAudience}))
	if status != "pending" {
		t.Fatalf("status = %s, want pending", status)
	}
	sentBackID, _ := createdStatus(t, publish(t, h, coordinator.Token, map[string]any{"title": "Sent back notice", "audience": csAudience}))
	expectStatus(t, "send back", review(t, h, hod.Token, sentBackID, "reject", "Add the room number"), http.StatusOK)
	liveID, _ := createdStatus(t, publish(t, h, coordinator.Token, map[string]any{"title": "Live notice", "audience": csAudience}))
	expectStatus(t, "approve", review(t, h, hod.Token, liveID, "approve", ""), http.StatusOK)
	draftID, _ := createdStatus(t, publish(t, h, coordinator.Token, map[string]any{"title": "Draft notice", "audience": csAudience, "draft": true}))

	response := endRole(t, h, hod.Token, coordinator.ID, roleID(t, h, hod.Token, coordinator.ID, "student_coordinator"), "")
	expectStatus(t, "end coordinator", response, http.StatusOK)
	var ended struct {
		Data struct {
			Handover struct {
				WithdrawnAnnouncements []struct {
					Title string `json:"title"`
				} `json:"withdrawn_announcements"`
			} `json:"handover"`
		} `json:"data"`
	}
	response.Decode(t, &ended)
	if withdrawn := ended.Data.Handover.WithdrawnAnnouncements; len(withdrawn) != 2 || withdrawn[0].Title != "Waiting notice" {
		t.Errorf("withdrawn announcements = %+v, want the waiting and sent back notices", withdrawn)
	}

	if titles := queueTitles(t, h, hod.Token); contains(titles, "Waiting notice") {
		t.Errorf("approval queue = %v, the withdrawn notice is still waiting", titles)
	}
	statuses := map[string]string{}
	for _, item := range mine(t, h, h.TokenFor(t, coordinator.ID, "student")) {
		statuses[item.ID] = item.Status
	}
	want := map[string]string{waitingID: "withdrawn", sentBackID: "withdrawn", liveID: "published", draftID: "draft"}
	for id, status := range want {
		if statuses[id] != status {
			t.Errorf("announcement %s status = %q, want %q", id, statuses[id], status)
		}
	}
}

// getEvent reads an Event as the caller, failing unless it's visible.
func getEvent(t *testing.T, h *apitest.Harness, token, id string) eventItem {
	t.Helper()
	response := h.Do(t, http.MethodGet, "/api/v1/events/"+id, token, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("get event status = %d, want %d: %s", response.Status, http.StatusOK, response.Body)
	}
	var body struct {
		Data eventItem `json:"data"`
	}
	response.Decode(t, &body)
	return body.Data
}

func TestEndingACoordinatorReturnsTheirProposalsToDraft(t *testing.T) {
	h := apitest.New(t)
	hod := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	coordinator := csCoordinator(t, h)
	cs := h.DepartmentID(t, "CS")

	waiting := createdEvent(t, createEvent(t, h, coordinator.Token, eventBody(map[string]any{"title": "Waiting talk", "department_id": cs, "draft": false}))).ID
	sentBack := createdEvent(t, createEvent(t, h, coordinator.Token, eventBody(map[string]any{"title": "Sent back talk", "department_id": cs, "draft": false}))).ID
	eventStatus(t, hodReview(t, h, hod.Token, sentBack, "request_changes", "Pick a bigger room"))
	draft := createdEvent(t, createEvent(t, h, coordinator.Token, eventBody(map[string]any{"title": "Draft talk", "department_id": cs}))).ID

	response := endRole(t, h, hod.Token, coordinator.ID, roleID(t, h, hod.Token, coordinator.ID, "student_coordinator"), "")
	expectStatus(t, "end coordinator", response, http.StatusOK)
	var ended struct {
		Data struct {
			Handover struct {
				ReturnedEvents []struct {
					ID string `json:"id"`
				} `json:"returned_events"`
			} `json:"handover"`
		} `json:"data"`
	}
	response.Decode(t, &ended)
	if returned := ended.Data.Handover.ReturnedEvents; len(returned) != 2 {
		t.Errorf("returned events = %+v, want 2", returned)
	}

	if queue := reviewQueue(t, h, hod.Token); len(queue) != 0 {
		t.Errorf("HOD review queue = %v, want the returned proposals gone", queue)
	}
	student := h.TokenFor(t, coordinator.ID, "student")
	for _, id := range []string{waiting, sentBack, draft} {
		if status := getEvent(t, h, student, id).Status; status != "draft" {
			t.Errorf("event %s status = %q, want draft", id, status)
		}
	}
	if response := submitEvent(t, h, student, draft); response.Status == http.StatusOK {
		t.Errorf("a former coordinator submitted a draft: %s", response.Body)
	}
}

func TestEndingACoordinatorHandsTheirUpcomingEventsToTheHOD(t *testing.T) {
	h := apitest.New(t)
	hod := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	coordinator := csCoordinator(t, h)
	reader := student(t, h, "CS", 2023)
	id := facultyPublished(t, h, coordinator, hod, principal, map[string]any{"title": "Robotics meetup"})
	expectStatus(t, "RSVP", rsvp(t, h, reader.Token, id, "going"), http.StatusOK)

	preview := h.Do(t, http.MethodGet, "/api/v1/admin/users/"+coordinator.ID+"/roles/"+roleID(t, h, hod.Token, coordinator.ID, "student_coordinator")+"/ending", hod.Token, nil)
	expectStatus(t, "preview", preview, http.StatusOK)
	var previewed struct {
		Data struct {
			MovedEvents []struct {
				ID        string `json:"id"`
				Organiser *struct {
					UserID string `json:"user_id"`
				} `json:"organiser"`
			} `json:"moved_events"`
			OrganiserNeeded bool `json:"organiser_needed"`
		} `json:"data"`
	}
	preview.Decode(t, &previewed)
	moved := previewed.Data.MovedEvents
	if len(moved) != 1 || moved[0].ID != id || moved[0].Organiser == nil || moved[0].Organiser.UserID != hod.ID || previewed.Data.OrganiserNeeded {
		t.Fatalf("preview = %+v, want the meetup moving to the HOD", previewed.Data)
	}
	if organiser := getEvent(t, h, hod.Token, id).Organiser; organiser == nil || organiser.UserID != coordinator.ID {
		t.Fatalf("organiser after preview = %+v, a preview must change nothing", organiser)
	}

	expectStatus(t, "end coordinator", endRole(t, h, hod.Token, coordinator.ID, roleID(t, h, hod.Token, coordinator.ID, "student_coordinator"), ""), http.StatusOK)
	event := getEvent(t, h, hod.Token, id)
	if event.Organiser == nil || event.Organiser.UserID != hod.ID || event.ProposerID != coordinator.ID || event.Status != "published" {
		t.Fatalf("event = %+v, want it still published, proposed by the coordinator and run by the HOD", event)
	}
	if _, summary := rsvps(t, h, hod.Token, id); summary.Counts.Going != 1 {
		t.Errorf("going = %d, want the RSVP kept", summary.Counts.Going)
	}
	if response := cancelEvent(t, h, h.TokenFor(t, coordinator.ID, "student"), id, "Not running it"); response.Status == http.StatusOK {
		t.Error("the former coordinator still cancelled the event")
	}
}

func TestEndingARoleCanHandEventsToAnotherOrganiser(t *testing.T) {
	h := apitest.New(t)
	hod := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	coordinator := csCoordinator(t, h)
	other := student(t, h, "CS", 2023)
	id := facultyPublished(t, h, coordinator, hod, principal, nil)
	assignment := roleID(t, h, hod.Token, coordinator.ID, "student_coordinator")

	response := endRole(t, h, hod.Token, coordinator.ID, assignment, other.ID)
	expectStatus(t, "a student can't run the event", response, http.StatusBadRequest)
	if !strings.Contains(response.Body, "organiser_id") {
		t.Errorf("body = %s, want the organiser_id field named", response.Body)
	}

	expectStatus(t, "end with a faculty organiser", endRole(t, h, hod.Token, coordinator.ID, assignment, faculty.ID), http.StatusOK)
	if organiser := getEvent(t, h, faculty.Token, id).Organiser; organiser == nil || organiser.UserID != faculty.ID {
		t.Fatalf("organiser = %+v, want the faculty member", organiser)
	}
	// The faculty member isn't a reviewer, so cancelling proves they run it now.
	expectStatus(t, "new organiser cancels", cancelEvent(t, h, faculty.Token, id, "Speaker unwell"), http.StatusOK)
}

func TestEndingARoleNeedsAnOrganiserWhenNoOneTakesOverByDefault(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	hod := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	id := createdEvent(t, createEvent(t, h, hod.Token, eventBody(map[string]any{"department_id": h.DepartmentID(t, "CS"), "draft": false}))).ID
	eventStatus(t, finalReview(t, h, principal.Token, id, "approve", ""))
	assignment := roleID(t, h, admin.Token, hod.ID, "hod")

	preview := h.Do(t, http.MethodGet, "/api/v1/admin/users/"+hod.ID+"/roles/"+assignment+"/ending", admin.Token, nil)
	expectStatus(t, "preview", preview, http.StatusOK)
	if !strings.Contains(preview.Body, `"organiser_needed":true`) {
		t.Fatalf("preview = %s, want organiser_needed: the HOD whose role ends can't take their own events", preview.Body)
	}
	expectStatus(t, "end without an organiser", endRole(t, h, admin.Token, hod.ID, assignment, ""), http.StatusBadRequest)
	if roles := listRoles(t, h, admin.Token, hod.ID); !hasActiveRole(roles, assignment) {
		t.Fatalf("roles = %+v, a refused end must leave the role in effect", roles)
	}
	expectStatus(t, "end with the principal as organiser", endRole(t, h, admin.Token, hod.ID, assignment, principal.ID), http.StatusOK)
}

func TestWorkStaysWithSomeoneWhoCanStillAuthorIt(t *testing.T) {
	h := apitest.New(t)
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	teacher := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}, {Role: "faculty", DepartmentCode: "CS"}}})
	id := createdEvent(t, createEvent(t, h, teacher.Token, eventBody(map[string]any{"department_id": h.DepartmentID(t, "CS"), "draft": false}))).ID

	expectStatus(t, "end HOD role", endRole(t, h, principal.Token, teacher.ID, roleID(t, h, principal.Token, teacher.ID, "hod"), ""), http.StatusOK)
	if status := getEvent(t, h, principal.Token, id).Status; status != "hod_approved" {
		t.Errorf("status = %q, want the proposal still waiting: they are still CS faculty", status)
	}
}

func TestEndingPreviewOffersWhoCouldRunTheEvents(t *testing.T) {
	h := apitest.New(t)
	hod := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	csFaculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	ecFaculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "EC"}}})
	coordinator := csCoordinator(t, h)
	other := csCoordinator(t, h)
	facultyPublished(t, h, coordinator, hod, principal, nil)

	preview := h.Do(t, http.MethodGet, "/api/v1/admin/users/"+coordinator.ID+"/roles/"+roleID(t, h, hod.Token, coordinator.ID, "student_coordinator")+"/ending", hod.Token, nil)
	expectStatus(t, "preview", preview, http.StatusOK)
	var previewed struct {
		Data struct {
			Options []struct {
				UserID   string `json:"user_id"`
				FullName string `json:"full_name"`
			} `json:"organiser_options"`
		} `json:"data"`
	}
	preview.Decode(t, &previewed)
	got := map[string]bool{}
	for _, option := range previewed.Data.Options {
		if option.FullName == "" {
			t.Errorf("option %s has no name", option.UserID)
		}
		got[option.UserID] = true
	}
	for name, id := range map[string]string{"CS HOD": hod.ID, "CS faculty": csFaculty.ID, "principal": principal.ID} {
		if !got[id] {
			t.Errorf("options miss the %s", name)
		}
	}
	// Admins can run events but stay a quiet fallback; the others can't run a CS event.
	for name, id := range map[string]string{"admin": admin.ID, "EC faculty": ecFaculty.ID, "another coordinator": other.ID, "the person": coordinator.ID} {
		if got[id] {
			t.Errorf("options include the %s", name)
		}
	}
}
