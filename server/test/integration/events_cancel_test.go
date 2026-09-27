package integration

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

func cancelEvent(t *testing.T, h *apitest.Harness, token, id, reason string) apitest.Response {
	t.Helper()
	return h.Do(t, http.MethodPost, "/api/v1/events/"+id+"/cancel", token, map[string]string{"reason": reason})
}

// facultyPublished runs a CS faculty member's event through both reviews.
func facultyPublished(t *testing.T, h *apitest.Harness, faculty, hod, principal apitest.User, overrides map[string]any) string {
	t.Helper()
	body := map[string]any{"department_id": h.DepartmentID(t, "CS"), "draft": false}
	for key, value := range overrides {
		body[key] = value
	}
	id := createdEvent(t, createEvent(t, h, faculty.Token, eventBody(body))).ID
	eventStatus(t, hodReview(t, h, hod.Token, id, "approve", ""))
	eventStatus(t, finalReview(t, h, principal.Token, id, "approve", ""))
	return id
}

func TestPublishedEventsTakeLogisticsEditsOnly(t *testing.T) {
	h := apitest.New(t)
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	csHOD := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	reader, second := student(t, h, "CS", 2023), student(t, h, "CS", 2023)
	id := facultyPublished(t, h, faculty, csHOD, principal, map[string]any{"capacity": 10})

	later := time.Now().Add(9 * 24 * time.Hour).UTC().Truncate(time.Minute)
	response := h.Do(t, http.MethodPatch, "/api/v1/events/"+id, faculty.Token, map[string]any{
		"location": "Main auditorium", "starts_at": later.Format(time.RFC3339), "ends_at": later.Add(3 * time.Hour).Format(time.RFC3339),
	})
	if response.Status != http.StatusOK {
		t.Fatalf("proposer logistics edit = %d: %s", response.Status, response.Body)
	}
	var edited struct {
		Data eventItem `json:"data"`
	}
	response.Decode(t, &edited)
	if edited.Data.Status != "published" || edited.Data.Location != "Main auditorium" || !edited.Data.StartsAt.Equal(later) {
		t.Fatalf("edited = %+v, want it still published at the new place and time", edited.Data)
	}
	expectStatus(t, "HOD edits the description", h.Do(t, http.MethodPatch, "/api/v1/events/"+id, csHOD.Token, map[string]any{"description": "Bring a laptop."}), http.StatusOK)
	expectStatus(t, "admin raises capacity", h.Do(t, http.MethodPatch, "/api/v1/events/"+id, admin.Token, map[string]any{"capacity": 20}), http.StatusOK)

	for field, value := range map[string]any{
		"title":             "New title",
		"event_type":        "talk",
		"department_id":     h.DepartmentID(t, "EC"),
		"audience":          []map[string]any{{"batch_year": 2023}},
		"faculty_mentor_id": faculty.ID,
	} {
		response := h.Do(t, http.MethodPatch, "/api/v1/events/"+id, faculty.Token, map[string]any{field: value})
		if response.Status != http.StatusBadRequest {
			t.Errorf("changing %s after publishing: status = %d, want 400: %s", field, response.Status, response.Body)
		}
	}

	expectStatus(t, "first going", rsvp(t, h, reader.Token, id, "going"), http.StatusOK)
	expectStatus(t, "second going", rsvp(t, h, second.Token, id, "going"), http.StatusOK)
	expectStatus(t, "capacity below going", h.Do(t, http.MethodPatch, "/api/v1/events/"+id, faculty.Token, map[string]any{"capacity": 1}), http.StatusBadRequest)
	past := time.Now().Add(-time.Hour).UTC()
	expectStatus(t, "moving into the past", h.Do(t, http.MethodPatch, "/api/v1/events/"+id, faculty.Token, map[string]any{
		"starts_at": past.Format(time.RFC3339), "ends_at": past.Add(2 * time.Hour).Format(time.RFC3339),
	}), http.StatusBadRequest)
	expectStatus(t, "reader edits", h.Do(t, http.MethodPatch, "/api/v1/events/"+id, reader.Token, map[string]any{"location": "My room"}), http.StatusNotFound)

	var audits int
	h.DB().Raw(`SELECT count(*) FROM audit_logs WHERE action = 'event_logistics_updated' AND resource_id = ?`, id).Scan(&audits)
	if audits != 3 {
		t.Fatalf("logistics audit rows = %d, want 3", audits)
	}
}

func TestCancellingKeepsRSVPsAndStopsNewOnes(t *testing.T) {
	h := apitest.New(t)
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	csHOD := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	ecHOD := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "EC"}}})
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	reader, late := student(t, h, "CS", 2023), student(t, h, "CS", 2023)
	id := facultyPublished(t, h, faculty, csHOD, principal, nil)
	expectStatus(t, "going", rsvp(t, h, reader.Token, id, "going"), http.StatusOK)

	expectStatus(t, "no reason", cancelEvent(t, h, faculty.Token, id, " "), http.StatusBadRequest)
	expectStatus(t, "reader cancels", cancelEvent(t, h, reader.Token, id, "Bored"), http.StatusForbidden)
	expectStatus(t, "other HOD cancels", cancelEvent(t, h, ecHOD.Token, id, "Not mine"), http.StatusForbidden)
	if status := eventStatus(t, cancelEvent(t, h, faculty.Token, id, "Speaker is unwell.")); status != "cancelled" {
		t.Fatalf("status = %q, want cancelled", status)
	}
	expectStatus(t, "cancel twice", cancelEvent(t, h, principal.Token, id, "Again"), http.StatusConflict)
	expectStatus(t, "RSVP after cancel", rsvp(t, h, late.Token, id, "going"), http.StatusConflict)
	expectStatus(t, "edit after cancel", h.Do(t, http.MethodPatch, "/api/v1/events/"+id, faculty.Token, map[string]any{"location": "Elsewhere"}), http.StatusConflict)

	// Someone who answered keeps seeing it, marked cancelled, so they find out;
	// everyone else no longer sees it.
	if got, _ := eventFeed(t, h, reader.Token, url.Values{}); len(got) != 1 || got[0].Status != "cancelled" {
		t.Errorf("feed for the reader who answered = %+v, want the cancelled event", got)
	}
	if got, _ := eventFeed(t, h, late.Token, url.Values{}); len(got) != 0 {
		t.Errorf("feed for someone who never answered = %v, want the cancelled event gone", eventTitles(got))
	}
	response := h.Do(t, http.MethodGet, "/api/v1/events/"+id, reader.Token, nil)
	var detail struct {
		Data struct {
			Status       string `json:"status"`
			CancelReason string `json:"cancel_reason"`
		} `json:"data"`
	}
	response.Decode(t, &detail)
	if response.Status != http.StatusOK || detail.Data.Status != "cancelled" || detail.Data.CancelReason != "Speaker is unwell." {
		t.Errorf("reader's detail = %d %+v, want the cancellation and its reason", response.Status, detail.Data)
	}
	if _, summary := rsvps(t, h, faculty.Token, id); summary.Counts.Going != 1 || len(summary.People) != 1 {
		t.Errorf("rsvps after cancel = %+v, want them kept", summary)
	}
	if ended := myEvents(t, h, faculty.Token, "ended"); len(ended) != 1 || ended[0].ID != id {
		t.Errorf("mine?status=ended = %+v, want the cancelled event", ended)
	}
	var audits int
	h.DB().Raw(`SELECT count(*) FROM audit_logs WHERE action = 'event_cancelled' AND resource_id = ?`, id).Scan(&audits)
	if audits != 1 {
		t.Errorf("cancel audit rows = %d, want 1", audits)
	}
}

func TestWhichEventsCanBeCancelled(t *testing.T) {
	h := apitest.New(t)
	cs := h.DepartmentID(t, "CS")
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	csHOD := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})

	waiting := createdEvent(t, createEvent(t, h, faculty.Token, eventBody(map[string]any{"department_id": cs, "draft": false}))).ID
	if status := eventStatus(t, cancelEvent(t, h, csHOD.Token, waiting, "Clashes with exams.")); status != "cancelled" {
		t.Errorf("HOD cancelling a waiting event = %q, want cancelled", status)
	}
	rejected := createdEvent(t, createEvent(t, h, faculty.Token, eventBody(map[string]any{"department_id": cs, "draft": false}))).ID
	eventStatus(t, hodReview(t, h, csHOD.Token, rejected, "reject", "No budget."))
	expectStatus(t, "cancelling a rejected event", cancelEvent(t, h, faculty.Token, rejected, "Never mind"), http.StatusConflict)

	over := facultyPublished(t, h, faculty, csHOD, principal, nil)
	h.DB().Exec(`UPDATE events SET starts_at = now() - interval '3 hours', ends_at = now() - interval '1 hour' WHERE id = ?`, over)
	expectStatus(t, "cancelling an event that is over", cancelEvent(t, h, principal.Token, over, "Late"), http.StatusConflict)
	expectStatus(t, "editing an event that is over", h.Do(t, http.MethodPatch, "/api/v1/events/"+over, faculty.Token, map[string]any{"location": "Hall"}), http.StatusConflict)
	expectStatus(t, "unknown event", cancelEvent(t, h, principal.Token, "00000000-0000-0000-0000-000000000000", "Gone"), http.StatusNotFound)
}
