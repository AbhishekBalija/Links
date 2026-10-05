package integration

import (
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

// Notification emails, first slice (#206): people are told when a decision
// or change reaches them, so they needn't keep checking LINKS.

func TestApprovingARequestEmailsThePerson(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	id := signUp(t, h, "4MN24CS701")
	expectStatus(t, "approve", h.Do(t, http.MethodPatch, "/api/v1/admin/users/"+id+"/verify", admin.Token, nil), http.StatusOK)

	decisions := h.Outbox.DecisionsTo("request-4mn24cs701@apitest.local")
	if len(decisions) != 1 || !decisions[0].Approved || !strings.Contains(decisions[0].Joined, "Computer Science") || decisions[0].ReviewerName == "" {
		t.Errorf("decisions = %+v, want one approval naming Computer Science and the reviewer", decisions)
	}
}

func TestRejectingARequestEmailsTheReviewersNote(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	id := signUp(t, h, "4MN24CS702")
	reject := map[string]string{"status": "rejected", "note": "This USN isn't in our records."}
	expectStatus(t, "reject", h.Do(t, http.MethodPatch, "/api/v1/admin/users/"+id+"/status", admin.Token, reject), http.StatusOK)

	decisions := h.Outbox.DecisionsTo("request-4mn24cs702@apitest.local")
	if len(decisions) != 1 || decisions[0].Approved || decisions[0].Note != "This USN isn't in our records." {
		t.Errorf("decisions = %+v, want one not-approved with the note", decisions)
	}
}

func TestAReportedRowGetsNoRequestEmail(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	imported(t, importCSV(h, admin.Token, "email,full_name,usn\nwrong@gmail.com,Asha Rao,4MN23CS703\n"))
	session, response := signInWithCode(t, h, "wrong@gmail.com")
	expectStatus(t, "first sign-in", response, http.StatusOK)
	expectStatus(t, "not you", h.Do(t, http.MethodPost, "/api/v1/auth/not-me", session.AccessToken, nil), http.StatusOK)

	// The person at this email said the row isn't them.
	reject := map[string]string{"status": "rejected", "note": "Wrong email on the list"}
	expectStatus(t, "reject", h.Do(t, http.MethodPatch, "/api/v1/admin/users/"+userIDByEmail(t, h, "wrong@gmail.com")+"/status", admin.Token, reject), http.StatusOK)
	if decisions := h.Outbox.DecisionsTo("wrong@gmail.com"); len(decisions) != 0 {
		t.Errorf("decisions = %+v, want none for a reported row", decisions)
	}
}

func TestCancellingAnEventEmailsEveryoneGoingOrInterested(t *testing.T) {
	h := apitest.New(t)
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	hod := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	going, interested, declined := student(t, h, "CS", 2023), student(t, h, "CS", 2023), student(t, h, "CS", 2023)
	id := facultyPublished(t, h, faculty, hod, principal, nil)
	expectStatus(t, "going", rsvp(t, h, going.Token, id, "going"), http.StatusOK)
	expectStatus(t, "interested", rsvp(t, h, interested.Token, id, "interested"), http.StatusOK)
	expectStatus(t, "can't go", rsvp(t, h, declined.Token, id, "not_going"), http.StatusOK)

	expectStatus(t, "cancel", cancelEvent(t, h, faculty.Token, id, "The speaker is unwell."), http.StatusOK)
	notices := h.Outbox.Notices()
	if len(notices) != 1 || !notices[0].Letter.Cancelled || notices[0].Letter.Reason != "The speaker is unwell." {
		t.Fatalf("notices = %+v, want one cancellation with the reason", notices)
	}
	to := notices[0].To
	if len(to) != 2 || !slices.Contains(to, going.Email) || !slices.Contains(to, interested.Email) {
		t.Errorf("sent to %v, want only those going or interested", to)
	}
}

func TestMovingAnEventEmailsAnswerersButADescriptionEditDoesNot(t *testing.T) {
	h := apitest.New(t)
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	hod := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	going := student(t, h, "CS", 2023)
	id := facultyPublished(t, h, faculty, hod, principal, nil)
	expectStatus(t, "going", rsvp(t, h, going.Token, id, "going"), http.StatusOK)

	expectStatus(t, "fix a typo", h.Do(t, http.MethodPatch, "/api/v1/events/"+id, faculty.Token, map[string]any{"description": "A hands-on session with microcontrollers, fixed."}), http.StatusOK)
	if notices := h.Outbox.Notices(); len(notices) != 0 {
		t.Fatalf("notices = %+v, want none for a description edit", notices)
	}

	later := time.Now().Add(9 * 24 * time.Hour).UTC().Truncate(time.Minute)
	expectStatus(t, "move it", h.Do(t, http.MethodPatch, "/api/v1/events/"+id, faculty.Token, map[string]any{
		"location": "Seminar hall 1", "starts_at": later.Format(time.RFC3339), "ends_at": later.Add(2 * time.Hour).Format(time.RFC3339),
	}), http.StatusOK)
	notices := h.Outbox.Notices()
	if len(notices) != 1 || notices[0].Letter.Cancelled || notices[0].Letter.WasWhen == "" || notices[0].Letter.WasWhere != "CS Lab 3" || notices[0].Letter.Where != "Seminar hall 1" {
		t.Fatalf("notices = %+v, want one change with the old and new time and place", notices)
	}
	if len(notices[0].To) != 1 || notices[0].To[0] != going.Email {
		t.Errorf("sent to %v, want the person going", notices[0].To)
	}
}

func TestShortlistingEmailsTheApplicant(t *testing.T) {
	h := apitest.New(t)
	officer := placementOfficer(t, h)
	id := publishedOpportunity(t, h, officer, nil)
	asha := namedStudent(t, h, "Asha Rao", "CS", 2023)
	application := decodeApplication(t, applyTo(t, h, asha.Token, id), http.StatusCreated)

	expectStatus(t, "shortlist", setApplicationStatus(t, h, officer.Token, application.ID, "applied", "shortlisted"), http.StatusOK)
	updates := h.Outbox.UpdatesTo(asha.Email)
	if len(updates) != 1 || updates[0].Status != "shortlisted" || updates[0].Company == "" || updates[0].Title == "" {
		t.Errorf("updates = %+v, want one shortlisting naming the job", updates)
	}
}
