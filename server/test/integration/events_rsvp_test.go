package integration

import (
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

type rsvpSummary struct {
	Counts struct {
		Going      int `json:"going"`
		Interested int `json:"interested"`
		NotGoing   int `json:"not_going"`
	} `json:"counts"`
	MyStatus *string `json:"my_status"`
	People   []struct {
		UserID   string `json:"user_id"`
		FullName string `json:"full_name"`
		Status   string `json:"status"`
	} `json:"people"`
}

func rsvp(t *testing.T, h *apitest.Harness, token, id, status string) apitest.Response {
	t.Helper()
	return h.Do(t, http.MethodPost, "/api/v1/events/"+id+"/rsvp", token, map[string]string{"status": status})
}

func rsvps(t *testing.T, h *apitest.Harness, token, id string) (int, rsvpSummary) {
	t.Helper()
	response := h.Do(t, http.MethodGet, "/api/v1/events/"+id+"/rsvps", token, nil)
	var body struct {
		Data rsvpSummary `json:"data"`
	}
	if response.Status == http.StatusOK {
		response.Decode(t, &body)
	}
	return response.Status, body.Data
}

func TestRSVPRespectsCapacity(t *testing.T) {
	h := apitest.New(t)
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	id := publishedEvent(t, h, principal, "Robotics workshop", 3, map[string]any{"capacity": 2})
	asha, bala, chitra := student(t, h, "CS", 2023), student(t, h, "CS", 2023), student(t, h, "EC", 2024)

	expectStatus(t, "Asha going", rsvp(t, h, asha.Token, id, "going"), http.StatusOK)
	expectStatus(t, "Bala going", rsvp(t, h, bala.Token, id, "going"), http.StatusOK)
	full := rsvp(t, h, chitra.Token, id, "going")
	if full.Status != http.StatusConflict || !strings.Contains(full.Body, "event is full") {
		t.Fatalf("third going: status = %d, want 409: %s", full.Status, full.Body)
	}
	expectStatus(t, "Chitra interested", rsvp(t, h, chitra.Token, id, "interested"), http.StatusOK)
	expectStatus(t, "Asha going again", rsvp(t, h, asha.Token, id, "going"), http.StatusOK)
	expectStatus(t, "Asha not going", rsvp(t, h, asha.Token, id, "not_going"), http.StatusOK)
	expectStatus(t, "Chitra takes the seat", rsvp(t, h, chitra.Token, id, "going"), http.StatusOK)

	_, summary := rsvps(t, h, chitra.Token, id)
	if summary.Counts.Going != 2 || summary.Counts.Interested != 0 || summary.Counts.NotGoing != 1 {
		t.Fatalf("counts = %+v, want 2 going and 1 not going", summary.Counts)
	}
	if summary.MyStatus == nil || *summary.MyStatus != "going" {
		t.Fatalf("my_status = %v, want going", summary.MyStatus)
	}
	expectStatus(t, "bad status", rsvp(t, h, chitra.Token, id, "maybe"), http.StatusBadRequest)
}

func TestTheLastSeatGoesToOnlyOnePerson(t *testing.T) {
	h := apitest.New(t)
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	id := publishedEvent(t, h, principal, "One seat left", 3, map[string]any{"capacity": 1})
	var readers []apitest.User
	for range 6 {
		readers = append(readers, student(t, h, "CS", 2023))
	}

	statuses := make([]int, len(readers))
	var wait sync.WaitGroup
	for i, reader := range readers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			statuses[i] = rsvp(t, h, reader.Token, id, "going").Status
		}()
	}
	wait.Wait()
	ok := 0
	for _, status := range statuses {
		if status == http.StatusOK {
			ok++
		} else if status != http.StatusConflict {
			t.Errorf("status = %d, want 200 or 409", status)
		}
	}
	if ok != 1 {
		t.Fatalf("%d people got the one seat", ok)
	}
}

func TestRSVPOnlyForPublishedUpcomingEventsInTheAudience(t *testing.T) {
	h := apitest.New(t)
	cs := h.DepartmentID(t, "CS")
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	csStudent, ecStudent := student(t, h, "CS", 2023), student(t, h, "EC", 2023)

	csOnly := publishedEvent(t, h, principal, "CS only", 3, map[string]any{"audience": []map[string]any{{"department_id": cs}}})
	expectStatus(t, "outside the audience", rsvp(t, h, ecStudent.Token, csOnly, "going"), http.StatusNotFound)
	if status, _ := rsvps(t, h, ecStudent.Token, csOnly); status != http.StatusNotFound {
		t.Errorf("counts outside the audience: status = %d, want 404", status)
	}

	waiting := createdEvent(t, createEvent(t, h, faculty.Token, eventBody(map[string]any{"department_id": cs, "draft": false}))).ID
	expectStatus(t, "unpublished", rsvp(t, h, csStudent.Token, waiting, "going"), http.StatusNotFound)

	started := publishedEvent(t, h, principal, "Already on", 3, nil)
	h.DB().Exec(`UPDATE events SET starts_at = now() - interval '1 hour', ends_at = now() + interval '1 hour' WHERE id = ?`, started)
	expectStatus(t, "started", rsvp(t, h, csStudent.Token, started, "going"), http.StatusConflict)

	cancelled := publishedEvent(t, h, principal, "Called off", 3, nil)
	h.DB().Exec(`UPDATE events SET status = 'cancelled', cancelled_at = now(), cancel_reason = 'Rain' WHERE id = ?`, cancelled)
	expectStatus(t, "cancelled", rsvp(t, h, csStudent.Token, cancelled, "going"), http.StatusConflict)
}

func TestOnlyOrganisersSeeWhoRSVPd(t *testing.T) {
	h := apitest.New(t)
	cs := h.DepartmentID(t, "CS")
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	csHOD := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	ecHOD := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "EC"}}})
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	reader := student(t, h, "CS", 2023)

	id := createdEvent(t, createEvent(t, h, faculty.Token, eventBody(map[string]any{"department_id": cs, "draft": false}))).ID
	eventStatus(t, hodReview(t, h, csHOD.Token, id, "approve", ""))
	eventStatus(t, finalReview(t, h, principal.Token, id, "approve", ""))
	expectStatus(t, "reader going", rsvp(t, h, reader.Token, id, "going"), http.StatusOK)

	for name, token := range map[string]string{"proposer": faculty.Token, "CS HOD": csHOD.Token, "principal": principal.Token, "admin": admin.Token} {
		status, summary := rsvps(t, h, token, id)
		if status != http.StatusOK || len(summary.People) != 1 || summary.People[0].UserID != reader.ID || summary.People[0].Status != "going" {
			t.Errorf("%s: %d %+v, want the one person going", name, status, summary)
		}
	}
	for name, token := range map[string]string{"reader": reader.Token, "EC HOD": ecHOD.Token} {
		status, summary := rsvps(t, h, token, id)
		if status != http.StatusOK || summary.Counts.Going != 1 || summary.People != nil {
			t.Errorf("%s: %d %+v, want counts only", name, status, summary)
		}
	}
}
