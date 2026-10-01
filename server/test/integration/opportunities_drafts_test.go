package integration

import (
	"net/http"
	"net/url"
	"slices"
	"testing"
	"time"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

type opportunityItem struct {
	ID              string           `json:"id"`
	OpportunityType string           `json:"opportunity_type"`
	Title           string           `json:"title"`
	Company         string           `json:"company"`
	Description     string           `json:"description"`
	Location        *string          `json:"location"`
	Compensation    *string          `json:"compensation"`
	ApplyBy         string           `json:"apply_by"`
	ApplicationMode string           `json:"application_mode"`
	ExternalURL     *string          `json:"external_url"`
	Status          string           `json:"status"`
	Open            bool             `json:"open"`
	MyApplication   *myApplication   `json:"my_application"`
	ApplicantCounts *applicantCounts `json:"applicant_counts"`
	PublishedAt     *string          `json:"published_at"`
	ClosedAt        *string          `json:"closed_at"`
	Eligibility     []struct {
		DepartmentID   *string `json:"department_id"`
		DepartmentCode *string `json:"department_code"`
		BatchYear      *int    `json:"batch_year"`
		Role           *string `json:"role"`
	} `json:"eligibility"`
	PostedBy struct {
		UserID   string `json:"user_id"`
		FullName string `json:"full_name"`
	} `json:"posted_by"`
}

// opportunityBody is a valid internal job open for two weeks; fields can be
// overridden, and a nil override removes the field.
func opportunityBody(overrides map[string]any) map[string]any {
	body := map[string]any{
		"opportunity_type": "job",
		"title":            "Graduate Engineer Trainee",
		"company":          "Acme Systems",
		"description":      "Embedded firmware for industrial controllers.",
		"location":         "Mysuru",
		"compensation":     "4.5 LPA",
		"apply_by":         time.Now().Add(14 * 24 * time.Hour).UTC().Truncate(time.Minute).Format(time.RFC3339),
		"application_mode": "internal",
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

func createOpportunity(t *testing.T, h *apitest.Harness, token string, body map[string]any) apitest.Response {
	t.Helper()
	return h.Do(t, http.MethodPost, "/api/v1/opportunities", token, body)
}

func decodeOpportunity(t *testing.T, response apitest.Response, want int) opportunityItem {
	t.Helper()
	if response.Status != want {
		t.Fatalf("status = %d, want %d: %s", response.Status, want, response.Body)
	}
	var body struct {
		Data opportunityItem `json:"data"`
	}
	response.Decode(t, &body)
	return body.Data
}

func placementOfficer(t *testing.T, h *apitest.Harness) apitest.User {
	t.Helper()
	return h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "placement_officer"}}})
}

func managedOpportunities(t *testing.T, h *apitest.Harness, token string, query url.Values) ([]opportunityItem, string) {
	t.Helper()
	response := h.Do(t, http.MethodGet, "/api/v1/opportunities/manage?"+query.Encode(), token, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("manage status = %d, want %d: %s", response.Status, http.StatusOK, response.Body)
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

func TestPlacementStaffSaveOpportunityDrafts(t *testing.T) {
	h := apitest.New(t)
	cs := h.DepartmentID(t, "CS")
	officer := placementOfficer(t, h)

	created := decodeOpportunity(t, createOpportunity(t, h, officer.Token, opportunityBody(map[string]any{
		"eligibility": []map[string]any{{"department_id": cs, "batch_year": 2023, "role": "student"}},
	})), http.StatusCreated)
	if created.Status != "draft" || created.Title != "Graduate Engineer Trainee" || created.Company != "Acme Systems" || created.ApplicationMode != "internal" {
		t.Errorf("created = %+v, want the job as a draft", created)
	}
	if created.PostedBy.UserID != officer.ID {
		t.Errorf("posted_by = %+v, want the officer", created.PostedBy)
	}
	if len(created.Eligibility) != 1 || created.Eligibility[0].DepartmentCode == nil || *created.Eligibility[0].DepartmentCode != "CS" {
		t.Errorf("eligibility = %+v, want CS 2023 students", created.Eligibility)
	}

	for _, role := range []string{"principal", "admin"} {
		staff := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: role}}})
		decodeOpportunity(t, createOpportunity(t, h, staff.Token, opportunityBody(nil)), http.StatusCreated)
	}
	outsiders := map[string]apitest.UserSeed{
		"student": {Roles: []apitest.RoleSeed{{Role: "student"}}, Student: &apitest.StudentSeed{DepartmentCode: "CS", BatchYear: 2023}},
		"faculty": {Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}},
		"HOD":     {Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}},
	}
	for name, seed := range outsiders {
		user := h.SeedUser(t, seed)
		expectStatus(t, name+" posts", createOpportunity(t, h, user.Token, opportunityBody(nil)), http.StatusForbidden)
		expectStatus(t, name+" opens a draft", h.Do(t, http.MethodGet, "/api/v1/opportunities/"+created.ID, user.Token, nil), http.StatusNotFound)
	}

	var audits int
	h.DB().Raw(`SELECT count(*) FROM audit_logs WHERE action = 'opportunity_created' AND resource_id = ? AND actor_id = ?`, created.ID, officer.ID).Scan(&audits)
	if audits != 1 {
		t.Errorf("opportunity_created audits = %d, want 1", audits)
	}
}

func TestOpportunityFieldsAreValidated(t *testing.T) {
	h := apitest.New(t)
	officer := placementOfficer(t, h)
	cases := map[string]map[string]any{
		"short title":              {"title": "GE"},
		"missing company":          {"company": nil},
		"unknown type":             {"opportunity_type": "hackathon"},
		"missing apply_by":         {"apply_by": nil},
		"unknown mode":             {"application_mode": "email"},
		"external without a link":  {"application_mode": "external"},
		"external with a bad link": {"application_mode": "external", "external_url": "javascript:alert(1)"},
		"internal with a link":     {"external_url": "https://acme.example/careers"},
		"long compensation":        {"compensation": string(make([]byte, 201))},
		"empty eligibility rule":   {"eligibility": []map[string]any{{}}},
		"unknown department":       {"eligibility": []map[string]any{{"department_id": "7a1a4d9e-0c1e-4c7b-9d0e-1f2a3b4c5d6e"}}},
		"unknown role":             {"eligibility": []map[string]any{{"role": "dean"}}},
	}
	for name, overrides := range cases {
		expectStatus(t, name, createOpportunity(t, h, officer.Token, opportunityBody(overrides)), http.StatusBadRequest)
	}
	external := decodeOpportunity(t, createOpportunity(t, h, officer.Token, opportunityBody(map[string]any{
		"opportunity_type": "internship", "application_mode": "external", "external_url": "https://acme.example/careers", "compensation": nil, "location": nil,
	})), http.StatusCreated)
	if external.ExternalURL == nil || *external.ExternalURL != "https://acme.example/careers" || external.Compensation != nil || external.Location != nil {
		t.Errorf("external = %+v, want the link and no compensation or location", external)
	}
}

func TestAnyPlacementStaffEditsAnyDraft(t *testing.T) {
	h := apitest.New(t)
	ec := h.DepartmentID(t, "EC")
	officer := placementOfficer(t, h)
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	id := decodeOpportunity(t, createOpportunity(t, h, officer.Token, opportunityBody(nil)), http.StatusCreated).ID

	edited := decodeOpportunity(t, h.Do(t, http.MethodPatch, "/api/v1/opportunities/"+id, principal.Token, map[string]any{
		"title":        "Graduate Engineer Trainee (Firmware)",
		"compensation": nil,
		"eligibility":  []map[string]any{{"department_id": ec}},
	}), http.StatusOK)
	if edited.Title != "Graduate Engineer Trainee (Firmware)" || edited.Compensation != nil || edited.Company != "Acme Systems" {
		t.Errorf("edited = %+v, want the new title, no compensation, the rest kept", edited)
	}
	if len(edited.Eligibility) != 1 || *edited.Eligibility[0].DepartmentCode != "EC" {
		t.Errorf("eligibility = %+v, want EC only", edited.Eligibility)
	}
	expectStatus(t, "external without a link", h.Do(t, http.MethodPatch, "/api/v1/opportunities/"+id, principal.Token, map[string]any{"application_mode": "external"}), http.StatusBadRequest)
	expectStatus(t, "unknown opportunity", h.Do(t, http.MethodPatch, "/api/v1/opportunities/7a1a4d9e-0c1e-4c7b-9d0e-1f2a3b4c5d6e", principal.Token, map[string]any{"title": "Anything"}), http.StatusNotFound)

	student := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "student"}}, Student: &apitest.StudentSeed{DepartmentCode: "EC", BatchYear: 2023}})
	expectStatus(t, "student edits", h.Do(t, http.MethodPatch, "/api/v1/opportunities/"+id, student.Token, map[string]any{"title": "Mine now"}), http.StatusForbidden)

	var actors []string
	h.DB().Raw(`SELECT actor_id::text FROM audit_logs WHERE action = 'opportunity_updated' AND resource_id = ?`, id).Scan(&actors)
	if !slices.Equal(actors, []string{principal.ID}) {
		t.Errorf("opportunity_updated actors = %v, want the principal once", actors)
	}
}

func TestPlacementStaffListEveryOpportunityNewestFirst(t *testing.T) {
	h := apitest.New(t)
	officer := placementOfficer(t, h)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	titles := []string{"First role", "Second role", "Third role"}
	for i, title := range titles {
		poster := officer
		if i == 1 {
			poster = admin
		}
		decodeOpportunity(t, createOpportunity(t, h, poster.Token, opportunityBody(map[string]any{"title": title})), http.StatusCreated)
	}

	got := []string{}
	cursor := ""
	for range 3 {
		page, next := managedOpportunities(t, h, officer.Token, url.Values{"status": {"draft"}, "limit": {"2"}, "cursor": {cursor}})
		for _, item := range page {
			got = append(got, item.Title)
		}
		if next == "" {
			break
		}
		cursor = next
	}
	if want := []string{"Third role", "Second role", "First role"}; !slices.Equal(got, want) {
		t.Errorf("drafts = %v, want %v across pages", got, want)
	}
	if page, _ := managedOpportunities(t, h, officer.Token, url.Values{"status": {"published"}}); len(page) != 0 {
		t.Errorf("published = %d items, want none yet", len(page))
	}
	expectStatus(t, "unknown status", h.Do(t, http.MethodGet, "/api/v1/opportunities/manage?status=open", officer.Token, nil), http.StatusBadRequest)

	student := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "student"}}, Student: &apitest.StudentSeed{DepartmentCode: "CS", BatchYear: 2023}})
	expectStatus(t, "student lists drafts", h.Do(t, http.MethodGet, "/api/v1/opportunities/manage", student.Token, nil), http.StatusForbidden)
}
