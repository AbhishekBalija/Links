package integration

import (
	"net/http"
	"net/url"
	"slices"
	"testing"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

type myApplication struct {
	ID          string  `json:"id"`
	Mode        string  `json:"mode"`
	Status      string  `json:"status"`
	AppliedAt   string  `json:"applied_at"`
	WithdrawnAt *string `json:"withdrawn_at"`
}

func applyTo(t *testing.T, h *apitest.Harness, token, id string) apitest.Response {
	t.Helper()
	return h.Do(t, http.MethodPost, "/api/v1/opportunities/"+id+"/apply", token, nil)
}

func withdrawFrom(t *testing.T, h *apitest.Harness, token, id string) apitest.Response {
	t.Helper()
	return h.Do(t, http.MethodPost, "/api/v1/opportunities/"+id+"/withdraw", token, nil)
}

func decodeApplication(t *testing.T, response apitest.Response, want int) myApplication {
	t.Helper()
	if response.Status != want {
		t.Fatalf("status = %d, want %d: %s", response.Status, want, response.Body)
	}
	var body struct {
		Data myApplication `json:"data"`
	}
	response.Decode(t, &body)
	return body.Data
}

// openedBy returns the Opportunity as this member sees it.
func openedBy(t *testing.T, h *apitest.Harness, token, id string) opportunityItem {
	t.Helper()
	return decodeOpportunity(t, h.Do(t, http.MethodGet, "/api/v1/opportunities/"+id, token, nil), http.StatusOK)
}

func auditsBy(t *testing.T, h *apitest.Harness, action, resourceID, actorID string) int {
	t.Helper()
	var count int
	h.DB().Raw(`SELECT count(*) FROM audit_logs WHERE action = ? AND resource_id = ? AND actor_id = ?`, action, resourceID, actorID).Scan(&count)
	return count
}

func TestAnEligibleStudentAppliesOnceAndWithdrawsBeforeShortlisting(t *testing.T) {
	h := apitest.New(t)
	officer := placementOfficer(t, h)
	asha := studentOf(t, h, "CS", 2023)
	ravi := studentOf(t, h, "CS", 2023)
	id := publishedOpportunity(t, h, officer, map[string]any{
		"eligibility": []map[string]any{{"department_id": h.DepartmentID(t, "CS"), "batch_year": 2023}},
	})

	applied := decodeApplication(t, applyTo(t, h, asha.Token, id), http.StatusCreated)
	if applied.Mode != "internal" || applied.Status != "applied" || applied.AppliedAt == "" {
		t.Fatalf("application = %+v, want an internal application", applied)
	}
	expectStatus(t, "apply twice", applyTo(t, h, asha.Token, id), http.StatusConflict)
	if mine := openedBy(t, h, asha.Token, id).MyApplication; mine == nil || mine.Status != "applied" {
		t.Errorf("Asha's my_application = %+v, want applied", mine)
	}
	if theirs := openedBy(t, h, ravi.Token, id).MyApplication; theirs != nil {
		t.Errorf("Ravi sees an application: %+v, want none (his own only)", theirs)
	}
	if items, _ := opportunityFeed(t, h, asha.Token, nil); len(items) != 1 || items[0].MyApplication == nil {
		t.Errorf("feed = %+v, want the Opportunity with her application", items)
	}

	withdrawn := decodeApplication(t, withdrawFrom(t, h, asha.Token, id), http.StatusOK)
	if withdrawn.Status != "withdrawn" || withdrawn.WithdrawnAt == nil {
		t.Errorf("withdrawn = %+v, want withdrawn with a time", withdrawn)
	}
	expectStatus(t, "withdraw twice", withdrawFrom(t, h, asha.Token, id), http.StatusConflict)
	expectStatus(t, "apply after withdrawing", applyTo(t, h, asha.Token, id), http.StatusConflict)
	expectStatus(t, "withdraw without applying", withdrawFrom(t, h, ravi.Token, id), http.StatusNotFound)

	shortlisted := decodeApplication(t, applyTo(t, h, ravi.Token, id), http.StatusCreated)
	h.DB().Exec(`UPDATE opportunity_applications SET status = 'shortlisted' WHERE id = ?`, shortlisted.ID)
	expectStatus(t, "withdraw once shortlisted", withdrawFrom(t, h, ravi.Token, id), http.StatusConflict)

	if n := auditsBy(t, h, "application_submitted", applied.ID, asha.ID); n != 1 {
		t.Errorf("application_submitted audits = %d, want 1", n)
	}
	if n := auditsBy(t, h, "application_withdrawn", applied.ID, asha.ID); n != 1 {
		t.Errorf("application_withdrawn audits = %d, want 1", n)
	}
}

func TestOnlyEligibleStudentsApplyWhileItIsOpen(t *testing.T) {
	h := apitest.New(t)
	officer := placementOfficer(t, h)
	student := studentOf(t, h, "CS", 2023)
	csOnly := []map[string]any{{"department_id": h.DepartmentID(t, "CS")}}
	forEveryone := publishedOpportunity(t, h, officer, nil)
	forCS := publishedOpportunity(t, h, officer, map[string]any{"eligibility": csOnly})
	closed := publishedOpportunity(t, h, officer, nil)
	decodeOpportunity(t, closeOpportunity(t, h, officer.Token, closed), http.StatusOK)
	expired := publishedOpportunity(t, h, officer, nil)
	h.DB().Exec(`UPDATE opportunities SET apply_by = now() - interval '1 minute' WHERE id = ?`, expired)
	draft := decodeOpportunity(t, createOpportunity(t, h, officer.Token, opportunityBody(nil)), http.StatusCreated).ID

	ecStudent := studentOf(t, h, "EC", 2023)
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	expectStatus(t, "another department's student", applyTo(t, h, ecStudent.Token, forCS), http.StatusNotFound)
	expectStatus(t, "a draft", applyTo(t, h, student.Token, draft), http.StatusNotFound)
	expectStatus(t, "closed early", applyTo(t, h, student.Token, closed), http.StatusConflict)
	expectStatus(t, "past apply_by", applyTo(t, h, student.Token, expired), http.StatusConflict)
	expectStatus(t, "faculty", applyTo(t, h, faculty.Token, forEveryone), http.StatusForbidden)
	expectStatus(t, "placement officer", applyTo(t, h, officer.Token, forEveryone), http.StatusForbidden)
	expectStatus(t, "unknown", applyTo(t, h, student.Token, "7a1a4d9e-0c1e-4c7b-9d0e-1f2a3b4c5d6e"), http.StatusNotFound)
	decodeApplication(t, applyTo(t, h, student.Token, forCS), http.StatusCreated)
}

func TestStudentsTrackExternalApplicationsAndSeeWhatTheyAppliedTo(t *testing.T) {
	h := apitest.New(t)
	officer := placementOfficer(t, h)
	student := studentOf(t, h, "CS", 2023)
	internal := publishedOpportunity(t, h, officer, map[string]any{"title": "Applied inside LINKS"})
	external := publishedOpportunity(t, h, officer, map[string]any{
		"title": "Applied on the company site", "application_mode": "external", "external_url": "https://acme.example/careers",
	})
	publishedOpportunity(t, h, officer, map[string]any{"title": "Not applied"})

	decodeApplication(t, applyTo(t, h, student.Token, internal), http.StatusCreated)
	marked := decodeApplication(t, applyTo(t, h, student.Token, external), http.StatusCreated)
	if marked.Mode != "external" || marked.Status != "applied" {
		t.Errorf("external = %+v, want a tracked external application", marked)
	}
	// Past the deadline, what they applied to is still theirs to see.
	h.DB().Exec(`UPDATE opportunities SET apply_by = now() - interval '1 minute' WHERE id = ?`, internal)

	items, _ := opportunityFeed(t, h, student.Token, url.Values{"state": {"applied"}})
	got := opportunityTitles(items)
	slices.Sort(got)
	if want := []string{"Applied inside LINKS", "Applied on the company site"}; !slices.Equal(got, want) {
		t.Errorf("applied view = %v, want %v", got, want)
	}
	for _, item := range items {
		if item.MyApplication == nil {
			t.Errorf("%q has no my_application in the applied view", item.Title)
		}
	}
}
