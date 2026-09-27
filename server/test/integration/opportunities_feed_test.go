package integration

import (
	"net/http"
	"net/url"
	"slices"
	"testing"
	"time"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

func publishOpportunity(t *testing.T, h *apitest.Harness, token, id string) apitest.Response {
	t.Helper()
	return h.Do(t, http.MethodPost, "/api/v1/opportunities/"+id+"/publish", token, nil)
}

func closeOpportunity(t *testing.T, h *apitest.Harness, token, id string) apitest.Response {
	t.Helper()
	return h.Do(t, http.MethodPost, "/api/v1/opportunities/"+id+"/close", token, nil)
}

// publishedOpportunity creates and publishes an Opportunity and returns its ID.
func publishedOpportunity(t *testing.T, h *apitest.Harness, officer apitest.User, overrides map[string]any) string {
	t.Helper()
	id := decodeOpportunity(t, createOpportunity(t, h, officer.Token, opportunityBody(overrides)), http.StatusCreated).ID
	decodeOpportunity(t, publishOpportunity(t, h, officer.Token, id), http.StatusOK)
	return id
}

func opportunityFeed(t *testing.T, h *apitest.Harness, token string, query url.Values) ([]opportunityItem, string) {
	t.Helper()
	response := h.Do(t, http.MethodGet, "/api/v1/opportunities?"+query.Encode(), token, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("feed status = %d, want %d: %s", response.Status, http.StatusOK, response.Body)
	}
	var page struct {
		Data []opportunityItem `json:"data"`
		Meta struct {
			NextCursor string `json:"next_cursor"`
		} `json:"meta"`
	}
	response.Decode(t, &page)
	return page.Data, page.Meta.NextCursor
}

func opportunityTitles(items []opportunityItem) []string {
	titles := []string{}
	for _, item := range items {
		titles = append(titles, item.Title)
	}
	return titles
}

func studentOf(t *testing.T, h *apitest.Harness, department string, batch int) apitest.User {
	t.Helper()
	return h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "student"}}, Student: &apitest.StudentSeed{DepartmentCode: department, BatchYear: batch}})
}

func inDays(days int) string {
	return time.Now().Add(time.Duration(days) * 24 * time.Hour).UTC().Truncate(time.Minute).Format(time.RFC3339)
}

func TestPublishingOpensAnOpportunityToItsEligibleStudents(t *testing.T) {
	h := apitest.New(t)
	officer := placementOfficer(t, h)
	eligible := studentOf(t, h, "CS", 2023)
	otherBatch := studentOf(t, h, "CS", 2024)
	otherDepartment := studentOf(t, h, "EC", 2023)
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	csStudents2023 := []map[string]any{{"department_id": h.DepartmentID(t, "CS"), "batch_year": 2023, "role": "student"}}

	id := decodeOpportunity(t, createOpportunity(t, h, officer.Token, opportunityBody(map[string]any{"title": "Firmware trainee", "eligibility": csStudents2023})), http.StatusCreated).ID
	decodeOpportunity(t, createOpportunity(t, h, officer.Token, opportunityBody(map[string]any{"title": "Still a draft"})), http.StatusCreated)
	expectStatus(t, "student publishes", publishOpportunity(t, h, eligible.Token, id), http.StatusForbidden)

	published := decodeOpportunity(t, publishOpportunity(t, h, officer.Token, id), http.StatusOK)
	if published.Status != "published" || !published.Open || published.PublishedAt == nil {
		t.Fatalf("published = %+v, want published, open, with published_at", published)
	}
	expectStatus(t, "publish twice", publishOpportunity(t, h, officer.Token, id), http.StatusConflict)

	if items, _ := opportunityFeed(t, h, eligible.Token, nil); !slices.Equal(opportunityTitles(items), []string{"Firmware trainee"}) {
		t.Errorf("eligible feed = %v, want only the published one", opportunityTitles(items))
	}
	expectStatus(t, "eligible student opens it", h.Do(t, http.MethodGet, "/api/v1/opportunities/"+id, eligible.Token, nil), http.StatusOK)
	for name, user := range map[string]apitest.User{"other batch": otherBatch, "other department": otherDepartment, "faculty": faculty} {
		if items, _ := opportunityFeed(t, h, user.Token, nil); len(items) != 0 {
			t.Errorf("%s feed = %v, want nothing", name, opportunityTitles(items))
		}
		expectStatus(t, name+" opens it", h.Do(t, http.MethodGet, "/api/v1/opportunities/"+id, user.Token, nil), http.StatusNotFound)
	}

	var audits int
	h.DB().Raw(`SELECT count(*) FROM audit_logs WHERE action = 'opportunity_published' AND resource_id = ? AND actor_id = ?`, id, officer.ID).Scan(&audits)
	if audits != 1 {
		t.Errorf("opportunity_published audits = %d, want 1", audits)
	}
}

func TestPublishingNeedsAnApplyByDateAhead(t *testing.T) {
	h := apitest.New(t)
	officer := placementOfficer(t, h)
	id := decodeOpportunity(t, createOpportunity(t, h, officer.Token, opportunityBody(map[string]any{"apply_by": inDays(-1)})), http.StatusCreated).ID
	expectStatus(t, "publish with apply_by passed", publishOpportunity(t, h, officer.Token, id), http.StatusBadRequest)
	expectStatus(t, "publish an unknown one", publishOpportunity(t, h, officer.Token, "7a1a4d9e-0c1e-4c7b-9d0e-1f2a3b4c5d6e"), http.StatusNotFound)
}

func TestClosedOpportunitiesMoveToTheClosedView(t *testing.T) {
	h := apitest.New(t)
	officer := placementOfficer(t, h)
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	reader := studentOf(t, h, "CS", 2023)
	closedEarly := publishedOpportunity(t, h, officer, map[string]any{"title": "Closed early"})
	expired := publishedOpportunity(t, h, officer, map[string]any{"title": "Deadline passed"})
	publishedOpportunity(t, h, officer, map[string]any{"title": "Still open"})
	draft := decodeOpportunity(t, createOpportunity(t, h, officer.Token, opportunityBody(nil)), http.StatusCreated).ID
	h.DB().Exec(`UPDATE opportunities SET apply_by = now() - interval '1 hour' WHERE id = ?`, expired)

	closed := decodeOpportunity(t, closeOpportunity(t, h, principal.Token, closedEarly), http.StatusOK)
	if closed.Status != "closed" || closed.Open || closed.ClosedAt == nil {
		t.Errorf("closed = %+v, want closed and not open", closed)
	}
	expectStatus(t, "close twice", closeOpportunity(t, h, officer.Token, closedEarly), http.StatusConflict)
	expectStatus(t, "close a draft", closeOpportunity(t, h, officer.Token, draft), http.StatusConflict)
	expectStatus(t, "student closes", closeOpportunity(t, h, reader.Token, expired), http.StatusForbidden)

	if items, _ := opportunityFeed(t, h, reader.Token, nil); !slices.Equal(opportunityTitles(items), []string{"Still open"}) {
		t.Errorf("open view = %v, want only the one still open", opportunityTitles(items))
	}
	items, _ := opportunityFeed(t, h, reader.Token, url.Values{"state": {"closed"}})
	if got := opportunityTitles(items); !slices.Equal(got, []string{"Closed early", "Deadline passed"}) {
		t.Errorf("closed view = %v, want the closed and the expired one, latest deadline first", got)
	}
	expectStatus(t, "a closed one still opens", h.Do(t, http.MethodGet, "/api/v1/opportunities/"+closedEarly, reader.Token, nil), http.StatusOK)

	var audits int
	h.DB().Raw(`SELECT count(*) FROM audit_logs WHERE action = 'opportunity_closed' AND resource_id = ? AND actor_id = ?`, closedEarly, principal.ID).Scan(&audits)
	if audits != 1 {
		t.Errorf("opportunity_closed audits = %d, want 1", audits)
	}
}

func TestPublishedOpportunitiesKeepTheirApplicationMode(t *testing.T) {
	h := apitest.New(t)
	officer := placementOfficer(t, h)
	id := publishedOpportunity(t, h, officer, nil)
	path := "/api/v1/opportunities/" + id

	edited := decodeOpportunity(t, h.Do(t, http.MethodPatch, path, officer.Token, map[string]any{"title": "Graduate Engineer Trainee 2026", "apply_by": inDays(21)}), http.StatusOK)
	if edited.Title != "Graduate Engineer Trainee 2026" || edited.Status != "published" {
		t.Errorf("edited = %+v, want the new title, still published", edited)
	}
	expectStatus(t, "switch to external", h.Do(t, http.MethodPatch, path, officer.Token, map[string]any{"application_mode": "external", "external_url": "https://acme.example/careers"}), http.StatusConflict)
	expectStatus(t, "apply_by in the past", h.Do(t, http.MethodPatch, path, officer.Token, map[string]any{"apply_by": inDays(-1)}), http.StatusBadRequest)

	decodeOpportunity(t, closeOpportunity(t, h, officer.Token, id), http.StatusOK)
	expectStatus(t, "edit a closed one", h.Do(t, http.MethodPatch, path, officer.Token, map[string]any{"title": "Reopened?"}), http.StatusConflict)
}

func TestOpportunityFeedFiltersAndPages(t *testing.T) {
	h := apitest.New(t)
	officer := placementOfficer(t, h)
	cs, ec := h.DepartmentID(t, "CS"), h.DepartmentID(t, "EC")
	reader := studentOf(t, h, "CS", 2023)
	publishedOpportunity(t, h, officer, map[string]any{"title": "Due in 10", "apply_by": inDays(10), "eligibility": []map[string]any{{"department_id": cs}}})
	publishedOpportunity(t, h, officer, map[string]any{"title": "Due in 3", "apply_by": inDays(3), "opportunity_type": "internship"})
	publishedOpportunity(t, h, officer, map[string]any{"title": "Due in 7", "apply_by": inDays(7), "opportunity_type": "internship", "eligibility": []map[string]any{{"department_id": ec}, {"role": "student"}}})

	got := []string{}
	cursor := ""
	for range 4 {
		page, next := opportunityFeed(t, h, reader.Token, url.Values{"limit": {"2"}, "cursor": {cursor}})
		got = append(got, opportunityTitles(page)...)
		if next == "" {
			break
		}
		cursor = next
	}
	if want := []string{"Due in 3", "Due in 7", "Due in 10"}; !slices.Equal(got, want) {
		t.Errorf("feed = %v, want %v: soonest deadline first, across pages", got, want)
	}
	if items, _ := opportunityFeed(t, h, reader.Token, url.Values{"type": {"internship"}}); !slices.Equal(opportunityTitles(items), []string{"Due in 3", "Due in 7"}) {
		t.Errorf("internships = %v", opportunityTitles(items))
	}
	if items, _ := opportunityFeed(t, h, reader.Token, url.Values{"department": {"ec"}}); !slices.Equal(opportunityTitles(items), []string{"Due in 7"}) {
		t.Errorf("EC opportunities = %v, want the one naming EC", opportunityTitles(items))
	}
	for name, query := range map[string]string{"type": "type=hackathon", "state": "state=soon", "department": "department=ZZ", "cursor": "cursor=nope"} {
		expectStatus(t, "bad "+name, h.Do(t, http.MethodGet, "/api/v1/opportunities?"+query, reader.Token, nil), http.StatusBadRequest)
	}
	expectStatus(t, "signed out", h.Do(t, http.MethodGet, "/api/v1/opportunities", "", nil), http.StatusUnauthorized)
}
