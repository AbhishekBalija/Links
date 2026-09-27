package integration

import (
	"encoding/csv"
	"net/http"
	"strings"
	"testing"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

func exportCSV(t *testing.T, h *apitest.Harness, token, id string) (int, [][]string) {
	t.Helper()
	response := h.Do(t, http.MethodGet, "/api/v1/events/"+id+"/export", token, nil)
	if response.Status != http.StatusOK {
		return response.Status, nil
	}
	rows, err := csv.NewReader(strings.NewReader(response.Body)).ReadAll()
	if err != nil {
		t.Fatalf("parse export: %v\n%s", err, response.Body)
	}
	return response.Status, rows
}

func TestOrganisersExportParticipantsAndEveryExportIsAudited(t *testing.T) {
	h := apitest.New(t)
	cs := h.DepartmentID(t, "CS")
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	csHOD := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	ecHOD := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "EC"}}})
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	learner := student(t, h, "CS", 2023)
	teacher := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "EC"}}})
	// A name that a spreadsheet would run as a formula must be neutralised.
	h.DB().Exec(`UPDATE profiles SET full_name = '=HYPERLINK("http://evil")' WHERE user_id = ?`, teacher.ID)

	id := createdEvent(t, createEvent(t, h, faculty.Token, eventBody(map[string]any{"department_id": cs, "draft": false}))).ID
	eventStatus(t, hodReview(t, h, csHOD.Token, id, "approve", ""))
	eventStatus(t, finalReview(t, h, principal.Token, id, "approve", ""))
	expectStatus(t, "student going", rsvp(t, h, learner.Token, id, "going"), http.StatusOK)
	expectStatus(t, "teacher interested", rsvp(t, h, teacher.Token, id, "interested"), http.StatusOK)

	status, rows := exportCSV(t, h, faculty.Token, id)
	if status != http.StatusOK || len(rows) != 3 {
		t.Fatalf("export = %d with %d rows, want a header and 2 people", status, len(rows))
	}
	if strings.Join(rows[0], ",") != "full_name,email,usn,batch_year,department,rsvp_status,responded_at" {
		t.Fatalf("header = %v", rows[0])
	}
	byStatus := map[string][]string{}
	for _, row := range rows[1:] {
		byStatus[row[5]] = row
	}
	var learnerEmail string
	h.DB().Raw(`SELECT email FROM users WHERE id = ?`, learner.ID).Scan(&learnerEmail)
	going := byStatus["going"]
	if going[1] != learnerEmail || !strings.HasPrefix(going[2], "4MN23CS") || going[3] != "2023" || going[4] != "CS" || going[6] == "" {
		t.Errorf("student row = %v, want email, USN, Batch 2023, CS and a time", going)
	}
	interested := byStatus["interested"]
	if !strings.HasPrefix(interested[0], "'=") || interested[2] != "" || interested[3] != "" || interested[4] != "EC" {
		t.Errorf("staff row = %v, want the formula escaped, no USN or Batch, and EC", interested)
	}

	for name, token := range map[string]string{"CS HOD": csHOD.Token, "principal": principal.Token} {
		if status, _ := exportCSV(t, h, token, id); status != http.StatusOK {
			t.Errorf("%s export status = %d, want 200", name, status)
		}
	}
	if status, _ := exportCSV(t, h, learner.Token, id); status != http.StatusForbidden {
		t.Errorf("participant export status = %d, want 403", status)
	}
	if status, _ := exportCSV(t, h, ecHOD.Token, id); status != http.StatusForbidden {
		t.Errorf("other HOD export status = %d, want 403", status)
	}
	outsider := student(t, h, "EC", 2023)
	h.DB().Exec(`INSERT INTO audience_rules (target_type, target_id, department_id) VALUES ('event', ?, ?)`, id, cs)
	if status, _ := exportCSV(t, h, outsider.Token, id); status != http.StatusNotFound {
		t.Errorf("outsider export status = %d, want 404", status)
	}

	var audits int
	h.DB().Raw(`SELECT count(*) FROM audit_logs WHERE action = 'event_participants_exported' AND resource_id = ?`, id).Scan(&audits)
	if audits != 3 {
		t.Fatalf("export audit rows = %d, want 3 (one per successful export)", audits)
	}
}
