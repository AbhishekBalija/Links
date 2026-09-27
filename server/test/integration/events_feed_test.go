package integration

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

func eventFeed(t *testing.T, h *apitest.Harness, token string, query url.Values) ([]eventItem, string) {
	t.Helper()
	response := h.Do(t, http.MethodGet, "/api/v1/events?"+query.Encode(), token, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("feed status = %d, want %d: %s", response.Status, http.StatusOK, response.Body)
	}
	var page struct {
		Data []eventItem `json:"data"`
		Meta struct {
			NextCursor string `json:"next_cursor"`
		} `json:"meta"`
	}
	response.Decode(t, &page)
	return page.Data, page.Meta.NextCursor
}

func eventTitles(events []eventItem) []string {
	titles := []string{}
	for _, event := range events {
		titles = append(titles, event.Title)
	}
	return titles
}

// publishedEvent has the principal publish an Event starting in the given number of days.
func publishedEvent(t *testing.T, h *apitest.Harness, principal apitest.User, title string, days int, overrides map[string]any) string {
	t.Helper()
	starts := time.Now().Add(time.Duration(days) * 24 * time.Hour).UTC().Truncate(time.Minute)
	body := map[string]any{"title": title, "draft": false, "starts_at": starts.Format(time.RFC3339), "ends_at": starts.Add(2 * time.Hour).Format(time.RFC3339)}
	for key, value := range overrides {
		body[key] = value
	}
	event := createdEvent(t, createEvent(t, h, principal.Token, eventBody(body)))
	if event.Status != "published" {
		t.Fatalf("%s status = %q, want published", title, event.Status)
	}
	return event.ID
}

func TestEventFeedShowsUpcomingPublishedEventsForTheReadersAudience(t *testing.T) {
	h := apitest.New(t)
	cs := h.DepartmentID(t, "CS")
	ec := h.DepartmentID(t, "EC")
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	csStudent := student(t, h, "CS", 2023)
	ecFaculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "EC"}}})

	publishedEvent(t, h, principal, "College fest", 3, nil)
	publishedEvent(t, h, principal, "CS 2023 orientation", 1, map[string]any{"audience": []map[string]any{{"department_id": cs, "batch_year": 2023}}})
	publishedEvent(t, h, principal, "EC faculty meet", 2, map[string]any{"audience": []map[string]any{{"department_id": ec, "role": "faculty"}}, "department_id": ec, "event_type": "talk"})
	ended := publishedEvent(t, h, principal, "Last week's talk", 1, nil)
	h.DB().Exec(`UPDATE events SET starts_at = now() - interval '8 days', ends_at = now() - interval '7 days' WHERE id = ?`, ended)
	cancelled := publishedEvent(t, h, principal, "Cancelled show", 4, nil)
	h.DB().Exec(`UPDATE events SET status = 'cancelled', cancelled_at = now(), cancel_reason = 'Rain' WHERE id = ?`, cancelled)
	createdEvent(t, createEvent(t, h, faculty.Token, eventBody(map[string]any{"title": "Waiting for HOD", "department_id": cs, "draft": false})))

	got, _ := eventFeed(t, h, csStudent.Token, nil)
	if want := []string{"CS 2023 orientation", "College fest"}; !sameTitles(eventTitles(got), want) {
		t.Errorf("CS student's feed = %v, want %v (soonest first)", eventTitles(got), want)
	}
	for _, event := range got {
		if len(event.Audience) > 0 && event.Audience[0].DepartmentCode == nil {
			t.Errorf("audience rule without department code: %+v", event.Audience)
		}
	}
	got, _ = eventFeed(t, h, ecFaculty.Token, nil)
	if want := []string{"EC faculty meet", "College fest"}; !sameTitles(eventTitles(got), want) {
		t.Errorf("EC faculty's feed = %v, want %v", eventTitles(got), want)
	}

	filters := []struct {
		query url.Values
		want  []string
	}{
		{url.Values{"department": {"EC"}}, []string{"EC faculty meet"}},
		{url.Values{"event_type": {"talk"}}, []string{"EC faculty meet"}},
		{url.Values{"from": {time.Now().Add(50 * time.Hour).UTC().Format(time.RFC3339)}}, []string{"College fest"}},
		{url.Values{"to": {time.Now().Add(60 * time.Hour).UTC().Format(time.RFC3339)}}, []string{"EC faculty meet"}},
		{url.Values{"from": {time.Now().Add(-10 * 24 * time.Hour).UTC().Format(time.RFC3339)}}, []string{"Last week's talk", "EC faculty meet", "College fest"}},
	}
	for _, filter := range filters {
		got, _ := eventFeed(t, h, ecFaculty.Token, filter.query)
		if !sameTitles(eventTitles(got), filter.want) {
			t.Errorf("%s: got %v, want %v", filter.query.Encode(), eventTitles(got), filter.want)
		}
	}
	for _, bad := range []string{"department=ZZ", "event_type=party", "from=yesterday", "limit=51"} {
		if response := h.Do(t, http.MethodGet, "/api/v1/events?"+bad, csStudent.Token, nil); response.Status != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want %d", bad, response.Status, http.StatusBadRequest)
		}
	}

	first, cursor := eventFeed(t, h, ecFaculty.Token, url.Values{"limit": {"1"}})
	second, last := eventFeed(t, h, ecFaculty.Token, url.Values{"limit": {"1"}, "cursor": {cursor}})
	if len(first) != 1 || len(second) != 1 || first[0].Title != "EC faculty meet" || second[0].Title != "College fest" || cursor == "" || last != "" {
		t.Errorf("pages = %v then %v (cursor %q, last %q)", eventTitles(first), eventTitles(second), cursor, last)
	}
}

func TestEventDetailIsForItsAudienceProposerAndReviewers(t *testing.T) {
	h := apitest.New(t)
	cs := h.DepartmentID(t, "CS")
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	csHOD := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	ecHOD := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "EC"}}})
	csStudent := student(t, h, "CS", 2023)
	ecStudent := student(t, h, "EC", 2023)

	detail := func(token, id string) (int, reviewedEvent) {
		response := h.Do(t, http.MethodGet, "/api/v1/events/"+id, token, nil)
		var body struct {
			Data reviewedEvent `json:"data"`
		}
		if response.Status == http.StatusOK {
			response.Decode(t, &body)
		}
		return response.Status, body.Data
	}

	csOnly := publishedEvent(t, h, principal, "CS only", 2, map[string]any{"audience": []map[string]any{{"department_id": cs}}})
	if status, event := detail(csStudent.Token, csOnly); status != http.StatusOK || event.Title != "CS only" {
		t.Errorf("CS student: %d %+v", status, event)
	}
	if status, _ := detail(ecStudent.Token, csOnly); status != http.StatusNotFound {
		t.Errorf("EC student: status = %d, want 404", status)
	}

	draft := createdEvent(t, createEvent(t, h, faculty.Token, eventBody(map[string]any{"title": "Faculty draft", "department_id": cs}))).ID
	if status, _ := detail(faculty.Token, draft); status != http.StatusOK {
		t.Errorf("proposer's draft: status = %d, want 200", status)
	}
	for name, token := range map[string]string{"CS HOD": csHOD.Token, "principal": principal.Token, "CS student": csStudent.Token} {
		if status, _ := detail(token, draft); status != http.StatusNotFound {
			t.Errorf("%s on a draft: status = %d, want 404", name, status)
		}
	}

	eventStatus(t, submitEvent(t, h, faculty.Token, draft))
	eventStatus(t, hodReview(t, h, csHOD.Token, draft, "request_changes", "Book the hall first."))
	for name, token := range map[string]string{"CS HOD": csHOD.Token, "principal": principal.Token, "proposer": faculty.Token} {
		status, event := detail(token, draft)
		if status != http.StatusOK || len(event.Reviews) != 1 || event.Reviews[0].Note != "Book the hall first." {
			t.Errorf("%s on a reviewed event: %d, reviews %+v", name, status, event.Reviews)
		}
	}
	for name, token := range map[string]string{"EC HOD": ecHOD.Token, "CS student": csStudent.Token} {
		if status, _ := detail(token, draft); status != http.StatusNotFound {
			t.Errorf("%s on an unpublished event: status = %d, want 404", name, status)
		}
	}

	// Readers of a published event don't see the internal review notes.
	eventStatus(t, submitEvent(t, h, faculty.Token, draft))
	eventStatus(t, hodReview(t, h, csHOD.Token, draft, "approve", ""))
	eventStatus(t, finalReview(t, h, principal.Token, draft, "approve", ""))
	if status, event := detail(csStudent.Token, draft); status != http.StatusOK || len(event.Reviews) != 0 {
		t.Errorf("reader: %d, reviews %+v; want the event without review notes", status, event.Reviews)
	}
	if status, _ := detail(csStudent.Token, "not-a-uuid"); status != http.StatusNotFound {
		t.Errorf("bad id: status = %d, want 404", status)
	}
}

func sameTitles(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
