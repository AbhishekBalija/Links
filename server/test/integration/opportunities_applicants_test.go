package integration

import (
	"encoding/json"
	"net/http"
	"net/url"
	"slices"
	"sync"
	"testing"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

type applicant struct {
	ID      string `json:"id"`
	Student struct {
		UserID         string  `json:"user_id"`
		FullName       string  `json:"full_name"`
		Username       string  `json:"username"`
		Email          *string `json:"email"`
		USN            *string `json:"usn"`
		DepartmentCode *string `json:"department_code"`
		BatchYear      *int    `json:"batch_year"`
	} `json:"student"`
	Mode            string  `json:"mode"`
	Status          string  `json:"status"`
	AppliedAt       string  `json:"applied_at"`
	StatusChangedAt *string `json:"status_changed_at"`
}

func applicants(t *testing.T, h *apitest.Harness, token, id string, query url.Values) ([]applicant, string) {
	t.Helper()
	response := h.Do(t, http.MethodGet, "/api/v1/opportunities/"+id+"/applications?"+query.Encode(), token, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("applicants status = %d, want %d: %s", response.Status, http.StatusOK, response.Body)
	}
	var page struct {
		Data []applicant `json:"data"`
		Meta struct {
			NextCursor string `json:"next_cursor"`
		} `json:"meta"`
	}
	response.Decode(t, &page)
	return page.Data, page.Meta.NextCursor
}

func applicantNames(items []applicant) []string {
	names := []string{}
	for _, item := range items {
		names = append(names, item.Student.FullName)
	}
	return names
}

func setApplicationStatus(t *testing.T, h *apitest.Harness, token, applicationID, from, to string) apitest.Response {
	t.Helper()
	return h.Do(t, http.MethodPatch, "/api/v1/opportunity-applications/"+applicationID+"/status", token, map[string]string{"from": from, "status": to})
}

func namedStudent(t *testing.T, h *apitest.Harness, name, department string, batch int) apitest.User {
	t.Helper()
	return member(t, h, name, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "student"}}, Student: &apitest.StudentSeed{DepartmentCode: department, BatchYear: batch}})
}

func TestPlacementStaffListApplicantsWithFilters(t *testing.T) {
	h := apitest.New(t)
	officer := placementOfficer(t, h)
	id := publishedOpportunity(t, h, officer, nil)
	asha := namedStudent(t, h, "Asha CS 2023", "CS", 2023)
	bala := namedStudent(t, h, "Bala CS 2024", "CS", 2024)
	chitra := namedStudent(t, h, "Chitra EC 2023", "EC", 2023)
	for _, student := range []apitest.User{asha, bala, chitra} {
		decodeApplication(t, applyTo(t, h, student.Token, id), http.StatusCreated)
	}
	decodeApplication(t, withdrawFrom(t, h, bala.Token, id), http.StatusOK)

	all, _ := applicants(t, h, officer.Token, id, nil)
	if got := applicantNames(all); !slices.Equal(got, []string{"Asha CS 2023", "Bala CS 2024", "Chitra EC 2023"}) {
		t.Fatalf("applicants = %v, want all three in the order they applied", got)
	}
	first := all[0]
	if first.Student.USN == nil || first.Student.Email == nil || *first.Student.Email != asha.Email ||
		first.Student.DepartmentCode == nil || *first.Student.DepartmentCode != "CS" || first.Student.BatchYear == nil || *first.Student.BatchYear != 2023 {
		t.Errorf("first applicant = %+v, want Asha's USN, email, CS and 2023", first.Student)
	}
	var raw struct {
		Data []map[string]map[string]any `json:"data"`
	}
	json.Unmarshal([]byte(h.Do(t, http.MethodGet, "/api/v1/opportunities/"+id+"/applications", officer.Token, nil).Body), &raw)
	if _, hasPhone := raw.Data[0]["student"]["phone"]; hasPhone {
		t.Error("the applicant list includes a phone number")
	}

	filters := map[string]struct {
		query url.Values
		want  []string
	}{
		"withdrawn":  {url.Values{"status": {"withdrawn"}}, []string{"Bala CS 2024"}},
		"CS":         {url.Values{"department": {"cs"}}, []string{"Asha CS 2023", "Bala CS 2024"}},
		"batch 2023": {url.Values{"batch": {"2023"}}, []string{"Asha CS 2023", "Chitra EC 2023"}},
	}
	for name, filter := range filters {
		if items, _ := applicants(t, h, officer.Token, id, filter.query); !slices.Equal(applicantNames(items), filter.want) {
			t.Errorf("%s = %v, want %v", name, applicantNames(items), filter.want)
		}
	}
	page, next := applicants(t, h, officer.Token, id, url.Values{"limit": {"2"}})
	rest, last := applicants(t, h, officer.Token, id, url.Values{"limit": {"2"}, "cursor": {next}})
	if got := append(applicantNames(page), applicantNames(rest)...); !slices.Equal(got, []string{"Asha CS 2023", "Bala CS 2024", "Chitra EC 2023"}) || last != "" {
		t.Errorf("pages = %v (next %q), want everyone once", got, last)
	}
	for name, query := range map[string]string{"status": "status=hired", "department": "department=ZZ", "batch": "batch=twenty"} {
		expectStatus(t, "bad "+name, h.Do(t, http.MethodGet, "/api/v1/opportunities/"+id+"/applications?"+query, officer.Token, nil), http.StatusBadRequest)
	}

	hod := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	for name, token := range map[string]string{"student": asha.Token, "HOD": hod.Token} {
		expectStatus(t, name+" lists applicants", h.Do(t, http.MethodGet, "/api/v1/opportunities/"+id+"/applications", token, nil), http.StatusForbidden)
	}
	expectStatus(t, "unknown opportunity", h.Do(t, http.MethodGet, "/api/v1/opportunities/7a1a4d9e-0c1e-4c7b-9d0e-1f2a3b4c5d6e/applications", officer.Token, nil), http.StatusNotFound)

	// Opening the list is audited once per first page, not per later page.
	var views int
	h.DB().Raw(`SELECT count(*) FROM audit_logs WHERE action = 'applicants_viewed' AND resource_id = ? AND actor_id = ?`, id, officer.ID).Scan(&views)
	if views != 6 {
		t.Errorf("applicants_viewed audits = %d, want 6 (one per first page opened)", views)
	}
}

func TestPlacementStaffMoveApplicationsWithoutOverwritingEachOther(t *testing.T) {
	h := apitest.New(t)
	officer := placementOfficer(t, h)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	id := publishedOpportunity(t, h, officer, nil)
	student := studentOf(t, h, "CS", 2023)
	quitter := studentOf(t, h, "CS", 2023)
	application := decodeApplication(t, applyTo(t, h, student.Token, id), http.StatusCreated).ID
	withdrawn := decodeApplication(t, applyTo(t, h, quitter.Token, id), http.StatusCreated).ID
	decodeApplication(t, withdrawFrom(t, h, quitter.Token, id), http.StatusOK)

	shortlisted := setApplicationStatus(t, h, officer.Token, application, "applied", "shortlisted")
	expectStatus(t, "shortlist", shortlisted, http.StatusOK)
	// The admin saw it as applied and meant to reject it; it has moved on.
	expectStatus(t, "stale change", setApplicationStatus(t, h, admin.Token, application, "applied", "rejected"), http.StatusConflict)
	expectStatus(t, "select", setApplicationStatus(t, h, admin.Token, application, "shortlisted", "selected"), http.StatusOK)
	expectStatus(t, "undo to shortlisted", setApplicationStatus(t, h, officer.Token, application, "selected", "shortlisted"), http.StatusOK)
	expectStatus(t, "select again", setApplicationStatus(t, h, officer.Token, application, "shortlisted", "selected"), http.StatusOK)

	expectStatus(t, "staff set withdrawn", setApplicationStatus(t, h, officer.Token, application, "selected", "withdrawn"), http.StatusBadRequest)
	expectStatus(t, "same status", setApplicationStatus(t, h, officer.Token, application, "selected", "selected"), http.StatusBadRequest)
	expectStatus(t, "missing from", h.Do(t, http.MethodPatch, "/api/v1/opportunity-applications/"+application+"/status", officer.Token, map[string]string{"status": "rejected"}), http.StatusBadRequest)
	expectStatus(t, "move a withdrawn one", setApplicationStatus(t, h, officer.Token, withdrawn, "withdrawn", "shortlisted"), http.StatusConflict)
	expectStatus(t, "student moves their own", setApplicationStatus(t, h, student.Token, application, "selected", "rejected"), http.StatusForbidden)
	expectStatus(t, "unknown application", setApplicationStatus(t, h, officer.Token, "7a1a4d9e-0c1e-4c7b-9d0e-1f2a3b4c5d6e", "applied", "shortlisted"), http.StatusNotFound)

	if mine := openedBy(t, h, student.Token, id).MyApplication; mine == nil || mine.Status != "selected" {
		t.Errorf("student's my_application = %+v, want selected", mine)
	}
	var moves []string
	h.DB().Raw(`SELECT concat(metadata->>'from', '>', metadata->>'to') FROM audit_logs
		WHERE action = 'application_status_changed' AND resource_id = ? ORDER BY created_at`, application).Scan(&moves)
	if want := []string{"applied>shortlisted", "shortlisted>selected", "selected>shortlisted", "shortlisted>selected"}; !slices.Equal(moves, want) {
		t.Errorf("status audits = %v, want %v", moves, want)
	}
}

func TestTwoStaffMovingOneApplicationAtOnceCantBothWin(t *testing.T) {
	h := apitest.New(t)
	officer := placementOfficer(t, h)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	id := publishedOpportunity(t, h, officer, nil)
	student := studentOf(t, h, "CS", 2023)
	application := decodeApplication(t, applyTo(t, h, student.Token, id), http.StatusCreated).ID

	statuses := make([]int, 2)
	var wg sync.WaitGroup
	for i, move := range []struct{ token, to string }{{officer.Token, "shortlisted"}, {admin.Token, "rejected"}} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			statuses[i] = setApplicationStatus(t, h, move.token, application, "applied", move.to).Status
		}()
	}
	wg.Wait()
	slices.Sort(statuses)
	if !slices.Equal(statuses, []int{http.StatusOK, http.StatusConflict}) {
		t.Errorf("statuses = %v, want one 200 and one 409", statuses)
	}
}
