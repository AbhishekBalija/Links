package integration

import (
	"encoding/csv"
	"net/http"
	"strings"
	"testing"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

func exportApplicants(t *testing.T, h *apitest.Harness, token, id, query string) (apitest.Response, [][]string) {
	t.Helper()
	response := h.Do(t, http.MethodGet, "/api/v1/opportunities/"+id+"/export"+query, token, nil)
	if response.Status != http.StatusOK {
		return response, nil
	}
	rows, err := csv.NewReader(strings.NewReader(response.Body)).ReadAll()
	if err != nil {
		t.Fatalf("read csv: %v\n%s", err, response.Body)
	}
	return response, rows
}

func TestPlacementStaffExportApplicantsSafelyAndEveryExportIsAudited(t *testing.T) {
	h := apitest.New(t)
	officer := placementOfficer(t, h)
	id := publishedOpportunity(t, h, officer, nil)
	asha := namedStudent(t, h, "Asha Rao", "CS", 2023)
	mallory := namedStudent(t, h, "=HYPERLINK(\"http://evil.example\")", "EC", 2024)
	quitter := namedStudent(t, h, "Quinn Quit", "CS", 2023)
	for _, student := range []apitest.User{asha, mallory, quitter} {
		decodeApplication(t, applyTo(t, h, student.Token, id), http.StatusCreated)
	}
	decodeApplication(t, withdrawFrom(t, h, quitter.Token, id), http.StatusOK)
	all, _ := applicants(t, h, officer.Token, id, nil)
	expectStatus(t, "shortlist Asha", setApplicationStatus(t, h, officer.Token, all[0].ID, "applied", "shortlisted"), http.StatusOK)

	response, rows := exportApplicants(t, h, officer.Token, id, "")
	if response.Status != http.StatusOK {
		t.Fatalf("export status = %d: %s", response.Status, response.Body)
	}
	if got := response.Header.Get("Content-Type"); !strings.HasPrefix(got, "text/csv") {
		t.Errorf("Content-Type = %q, want text/csv", got)
	}
	if !strings.Contains(response.Header.Get("Content-Disposition"), "attachment") || response.Header.Get("Cache-Control") != "no-store" {
		t.Errorf("headers = %v, want an attachment that isn't cached", response.Header)
	}
	wantHeader := "full_name,email,usn,department,batch_year,mode,status,applied_at,status_changed_at"
	if len(rows) != 4 || strings.Join(rows[0], ",") != wantHeader {
		t.Fatalf("rows = %v, want the header and three applicants", rows)
	}
	if rows[1][0] != "Asha Rao" || rows[1][1] != asha.Email || rows[1][3] != "CS" || rows[1][4] != "2023" || rows[1][6] != "shortlisted" || rows[1][8] == "" {
		t.Errorf("Asha's row = %v", rows[1])
	}
	if !strings.HasPrefix(rows[2][0], "'=") {
		t.Errorf("formula name = %q, want it neutralised with a leading quote", rows[2][0])
	}
	if rows[3][6] != "withdrawn" {
		t.Errorf("Quinn's row = %v, want withdrawn", rows[3])
	}
	if strings.Contains(strings.ToLower(response.Body), "phone") {
		t.Error("the export mentions phone numbers")
	}

	_, shortlisted := exportApplicants(t, h, officer.Token, id, "?status=shortlisted")
	if len(shortlisted) != 2 || shortlisted[1][0] != "Asha Rao" {
		t.Errorf("shortlisted export = %v, want only Asha", shortlisted)
	}
	expectStatus(t, "bad status", h.Do(t, http.MethodGet, "/api/v1/opportunities/"+id+"/export?status=hired", officer.Token, nil), http.StatusBadRequest)
	hod := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	for name, token := range map[string]string{"student": asha.Token, "HOD": hod.Token} {
		expectStatus(t, name+" exports", h.Do(t, http.MethodGet, "/api/v1/opportunities/"+id+"/export", token, nil), http.StatusForbidden)
	}
	expectStatus(t, "unknown opportunity", h.Do(t, http.MethodGet, "/api/v1/opportunities/7a1a4d9e-0c1e-4c7b-9d0e-1f2a3b4c5d6e/export", officer.Token, nil), http.StatusNotFound)

	var exports []string
	h.DB().Raw(`SELECT metadata->>'rows' FROM audit_logs WHERE action = 'applicants_exported' AND resource_id = ? AND actor_id = ? ORDER BY created_at`, id, officer.ID).Scan(&exports)
	if strings.Join(exports, ",") != "3,1" {
		t.Errorf("export audits = %v, want 3 rows then 1", exports)
	}
}
