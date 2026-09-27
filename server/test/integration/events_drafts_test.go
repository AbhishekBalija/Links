package integration

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

type eventItem struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Status     string `json:"status"`
	EventType  string `json:"event_type"`
	Location   string `json:"location"`
	Capacity   *int   `json:"capacity"`
	ProposerID string `json:"proposer_id"`
	Department *struct {
		ID   string `json:"id"`
		Code string `json:"code"`
	} `json:"department"`
	FacultyMentor *struct {
		UserID   string `json:"user_id"`
		FullName string `json:"full_name"`
	} `json:"faculty_mentor"`
	Audience []struct {
		DepartmentCode *string `json:"department_code"`
		BatchYear      *int    `json:"batch_year"`
	} `json:"audience"`
	StartsAt time.Time `json:"starts_at"`
	EndsAt   time.Time `json:"ends_at"`
}

// eventBody is a valid event a week from now; fields can be overridden.
func eventBody(overrides map[string]any) map[string]any {
	starts := time.Now().Add(7 * 24 * time.Hour).UTC().Truncate(time.Minute)
	body := map[string]any{
		"title":       "Intro to embedded systems",
		"description": "A hands-on session with microcontrollers.",
		"event_type":  "workshop",
		"location":    "CS Lab 3",
		"starts_at":   starts.Format(time.RFC3339),
		"ends_at":     starts.Add(2 * time.Hour).Format(time.RFC3339),
		"draft":       true,
	}
	for key, value := range overrides {
		if value == nil {
			delete(body, key)
		} else {
			body[key] = value
		}
	}
	return body
}

func createEvent(t *testing.T, h *apitest.Harness, token string, body map[string]any) apitest.Response {
	t.Helper()
	return h.Do(t, http.MethodPost, "/api/v1/events", token, body)
}

func createdEvent(t *testing.T, response apitest.Response) eventItem {
	t.Helper()
	if response.Status != http.StatusCreated {
		t.Fatalf("create event status = %d, want %d: %s", response.Status, http.StatusCreated, response.Body)
	}
	var created struct {
		Data eventItem `json:"data"`
	}
	response.Decode(t, &created)
	return created.Data
}

func myEvents(t *testing.T, h *apitest.Harness, token, status string) []eventItem {
	t.Helper()
	path := "/api/v1/events/mine"
	if status != "" {
		path += "?status=" + status
	}
	response := h.Do(t, http.MethodGet, path, token, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("mine status = %d, want %d: %s", response.Status, http.StatusOK, response.Body)
	}
	var list struct {
		Data []eventItem `json:"data"`
	}
	response.Decode(t, &list)
	return list.Data
}

func TestCoordinatorSavesADraftForTheirDepartment(t *testing.T) {
	h := apitest.New(t)
	cs := h.DepartmentID(t, "CS")
	coordinator := h.SeedUser(t, apitest.UserSeed{
		Roles:   []apitest.RoleSeed{{Role: "student"}, {Role: "student_coordinator", DepartmentCode: "CS"}},
		Student: &apitest.StudentSeed{DepartmentCode: "CS", BatchYear: 2023},
	})
	mentor := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})

	event := createdEvent(t, createEvent(t, h, coordinator.Token, eventBody(map[string]any{
		"department_id":     cs,
		"faculty_mentor_id": mentor.ID,
		"capacity":          40,
		"audience":          []map[string]any{{"department_id": cs, "batch_year": 2023}},
	})))
	if event.Status != "draft" || event.ProposerID != coordinator.ID || event.Department == nil || event.Department.Code != "CS" {
		t.Fatalf("event = %+v, want a CS draft by the coordinator", event)
	}
	if event.FacultyMentor == nil || event.FacultyMentor.UserID != mentor.ID || event.Capacity == nil || *event.Capacity != 40 {
		t.Fatalf("event = %+v, want the mentor and capacity kept", event)
	}
	if len(event.Audience) != 1 || event.Audience[0].DepartmentCode == nil || *event.Audience[0].DepartmentCode != "CS" {
		t.Fatalf("audience = %+v, want CS 2023", event.Audience)
	}

	drafts := myEvents(t, h, coordinator.Token, "draft")
	if len(drafts) != 1 || drafts[0].ID != event.ID {
		t.Fatalf("mine?status=draft = %+v, want the draft", drafts)
	}
	var audit int
	h.DB().Raw(`SELECT count(*) FROM audit_logs WHERE action = 'event_created' AND resource_id = ?`, event.ID).Scan(&audit)
	if audit != 1 {
		t.Fatalf("event_created audit rows = %d, want 1", audit)
	}
}

func TestWhoMayProposeWhichEvents(t *testing.T) {
	h := apitest.New(t)
	cs := h.DepartmentID(t, "CS")
	ec := h.DepartmentID(t, "EC")
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	hod := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	officer := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "placement_officer"}}})
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	reader := student(t, h, "CS", 2023)

	allowed := map[string]struct {
		token string
		body  map[string]any
	}{
		"faculty, own department":        {faculty.Token, eventBody(map[string]any{"department_id": cs})},
		"HOD, own department":            {hod.Token, eventBody(map[string]any{"department_id": cs})},
		"officer, college-wide training": {officer.Token, eventBody(map[string]any{"event_type": "training"})},
		"principal, college-wide":        {principal.Token, eventBody(nil)},
		"principal, any department":      {principal.Token, eventBody(map[string]any{"department_id": ec})},
	}
	for name, test := range allowed {
		if response := createEvent(t, h, test.token, test.body); response.Status != http.StatusCreated {
			t.Errorf("%s: status = %d, want %d: %s", name, response.Status, http.StatusCreated, response.Body)
		}
	}

	refused := map[string]struct {
		token string
		body  map[string]any
		want  int
	}{
		"student":                   {reader.Token, eventBody(map[string]any{"department_id": cs}), http.StatusForbidden},
		"faculty, other department": {faculty.Token, eventBody(map[string]any{"department_id": ec}), http.StatusBadRequest},
		"faculty, college-wide":     {faculty.Token, eventBody(nil), http.StatusBadRequest},
		"HOD, other department":     {hod.Token, eventBody(map[string]any{"department_id": ec}), http.StatusBadRequest},
		"officer, not training":     {officer.Token, eventBody(map[string]any{"event_type": "talk"}), http.StatusBadRequest},
	}
	for name, test := range refused {
		if response := createEvent(t, h, test.token, test.body); response.Status != test.want {
			t.Errorf("%s: status = %d, want %d: %s", name, response.Status, test.want, response.Body)
		}
	}
}

func TestEventFieldsAreValidated(t *testing.T) {
	h := apitest.New(t)
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	notFaculty := student(t, h, "CS", 2023)
	starts := time.Now().Add(48 * time.Hour).UTC()

	bad := map[string]map[string]any{
		"title":             {"title": "Hi"},
		"event_type":        {"event_type": "party"},
		"location":          {"location": "  "},
		"ends_at":           {"ends_at": starts.Add(-time.Hour).Format(time.RFC3339), "starts_at": starts.Format(time.RFC3339)},
		"capacity":          {"capacity": 0},
		"department_id":     {"department_id": "00000000-0000-0000-0000-000000000000"},
		"faculty_mentor_id": {"faculty_mentor_id": notFaculty.ID},
		"audience":          {"audience": []map[string]any{{"batch_year": 1990}}},
		"description":       {"description": strings.Repeat("x", 5001)},
	}
	for field, override := range bad {
		response := createEvent(t, h, principal.Token, eventBody(override))
		if response.Status != http.StatusBadRequest || !strings.Contains(response.Body, `"`+field+`"`) {
			t.Errorf("%s: status = %d, want 400 naming the field: %s", field, response.Status, response.Body)
		}
	}
	// A draft may be in the past; only submitting needs a future date.
	past := time.Now().Add(-48 * time.Hour).UTC()
	createdEvent(t, createEvent(t, h, principal.Token, eventBody(map[string]any{
		"starts_at": past.Format(time.RFC3339), "ends_at": past.Add(time.Hour).Format(time.RFC3339),
	})))
}

func TestProposerEditsTheirDraftOnly(t *testing.T) {
	h := apitest.New(t)
	cs := h.DepartmentID(t, "CS")
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	colleague := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	event := createdEvent(t, createEvent(t, h, faculty.Token, eventBody(map[string]any{"department_id": cs})))

	response := h.Do(t, http.MethodPatch, "/api/v1/events/"+event.ID, faculty.Token, map[string]any{
		"title": "Embedded systems bootcamp", "capacity": 25, "audience": []map[string]any{{"department_id": cs}},
	})
	if response.Status != http.StatusOK {
		t.Fatalf("edit status = %d, want %d: %s", response.Status, http.StatusOK, response.Body)
	}
	var edited struct {
		Data eventItem `json:"data"`
	}
	response.Decode(t, &edited)
	if edited.Data.Title != "Embedded systems bootcamp" || edited.Data.Location != "CS Lab 3" || edited.Data.Capacity == nil || *edited.Data.Capacity != 25 || len(edited.Data.Audience) != 1 {
		t.Fatalf("edited = %+v, want the new title, capacity and audience with the rest kept", edited.Data)
	}

	if response := h.Do(t, http.MethodPatch, "/api/v1/events/"+event.ID, colleague.Token, map[string]any{"title": "Taken over"}); response.Status != http.StatusNotFound {
		t.Errorf("colleague edit status = %d, want %d: %s", response.Status, http.StatusNotFound, response.Body)
	}
	if response := h.Do(t, http.MethodPatch, "/api/v1/events/"+event.ID, faculty.Token, map[string]any{"department_id": h.DepartmentID(t, "EC")}); response.Status != http.StatusBadRequest {
		t.Errorf("moving to another department status = %d, want %d: %s", response.Status, http.StatusBadRequest, response.Body)
	}
	if response := h.Do(t, http.MethodPatch, "/api/v1/events/"+event.ID, faculty.Token, map[string]any{"ends_at": time.Now().Add(time.Hour).UTC().Format(time.RFC3339)}); response.Status != http.StatusBadRequest {
		t.Errorf("ending before it starts status = %d, want %d: %s", response.Status, http.StatusBadRequest, response.Body)
	}
}

func TestMineListsOnlyTheCallersEventsNewestFirst(t *testing.T) {
	h := apitest.New(t)
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	for _, title := range []string{"First event", "Second event", "Third event"} {
		createdEvent(t, createEvent(t, h, principal.Token, eventBody(map[string]any{"title": title})))
	}
	createdEvent(t, createEvent(t, h, admin.Token, eventBody(map[string]any{"title": "Admin event"})))

	response := h.Do(t, http.MethodGet, "/api/v1/events/mine?limit=2", principal.Token, nil)
	var page struct {
		Data []eventItem `json:"data"`
		Meta struct {
			NextCursor string `json:"next_cursor"`
		} `json:"meta"`
	}
	response.Decode(t, &page)
	if len(page.Data) != 2 || page.Data[0].Title != "Third event" || page.Meta.NextCursor == "" {
		t.Fatalf("first page = %+v, want the two newest and a cursor", page)
	}
	response = h.Do(t, http.MethodGet, "/api/v1/events/mine?limit=2&cursor="+page.Meta.NextCursor, principal.Token, nil)
	page.Data, page.Meta.NextCursor = nil, ""
	response.Decode(t, &page)
	if len(page.Data) != 1 || page.Data[0].Title != "First event" || page.Meta.NextCursor != "" {
		t.Fatalf("second page = %+v, want the oldest and no cursor", page)
	}
	if response := h.Do(t, http.MethodGet, "/api/v1/events/mine?status=bogus", principal.Token, nil); response.Status != http.StatusBadRequest {
		t.Errorf("bad status filter = %d, want %d", response.Status, http.StatusBadRequest)
	}
}

func TestProposerDeletesOnlyTheirOwnDrafts(t *testing.T) {
	h := apitest.New(t)
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	colleague := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	cs := h.DepartmentID(t, "CS")
	draft := createdEvent(t, createEvent(t, h, faculty.Token, eventBody(map[string]any{"department_id": cs, "audience": []map[string]any{{"department_id": cs}}})))
	submitted := createdEvent(t, createEvent(t, h, faculty.Token, eventBody(map[string]any{"department_id": cs, "title": "Sent for review"})))
	if status := eventStatus(t, submitEvent(t, h, faculty.Token, submitted.ID)); status != "submitted" {
		t.Fatalf("submitted status = %q", status)
	}

	del := func(token, id string) int {
		return h.Do(t, http.MethodDelete, "/api/v1/events/"+id, token, nil).Status
	}
	if got := del(colleague.Token, draft.ID); got != http.StatusNotFound {
		t.Errorf("colleague deletes = %d, want %d", got, http.StatusNotFound)
	}
	if got := del(faculty.Token, submitted.ID); got != http.StatusConflict {
		t.Errorf("deleting a submitted event = %d, want %d (cancel it instead)", got, http.StatusConflict)
	}
	if got := del(faculty.Token, draft.ID); got != http.StatusNoContent {
		t.Fatalf("deleting own draft = %d, want %d", got, http.StatusNoContent)
	}
	if got := h.Do(t, http.MethodGet, "/api/v1/events/"+draft.ID, faculty.Token, nil).Status; got != http.StatusNotFound {
		t.Errorf("deleted draft = %d, want %d", got, http.StatusNotFound)
	}
	if drafts := myEvents(t, h, faculty.Token, "draft"); len(drafts) != 0 {
		t.Errorf("drafts after delete = %+v, want none", drafts)
	}
	var rules, audits int
	h.DB().Raw(`SELECT count(*) FROM audience_rules WHERE target_type = 'event' AND target_id = ?`, draft.ID).Scan(&rules)
	h.DB().Raw(`SELECT count(*) FROM audit_logs WHERE action = 'event_draft_deleted' AND resource_id = ?`, draft.ID).Scan(&audits)
	if rules != 0 || audits != 1 {
		t.Errorf("audience rules = %d, audit rows = %d, want 0 and 1", rules, audits)
	}
}
