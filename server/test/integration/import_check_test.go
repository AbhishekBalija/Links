package integration

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

// checkedImport is the import response with the fields the check step uses.
type checkedImport struct {
	DryRun     bool `json:"dry_run"`
	Department *struct {
		Code string `json:"code"`
		Name string `json:"name"`
	} `json:"department"`
	Ready   int `json:"ready"`
	Created int `json:"created"`
	Failed  int `json:"failed"`
	Groups  []struct {
		DepartmentCode string `json:"department_code"`
		DepartmentName string `json:"department_name"`
		BatchYear      int    `json:"batch_year"`
		Rows           int    `json:"rows"`
		Ready          int    `json:"ready"`
		Created        int    `json:"created"`
		Failed         int    `json:"failed"`
		Outside        bool   `json:"outside"`
	} `json:"groups"`
	Rows []struct {
		Row            int    `json:"row"`
		Email          string `json:"email"`
		USN            string `json:"usn"`
		DepartmentCode string `json:"department_code"`
		BatchYear      int    `json:"batch_year"`
		Status         string `json:"status"`
		Outside        bool   `json:"outside"`
		Error          string `json:"error"`
	} `json:"rows"`
}

// importWith uploads the CSV with extra form fields, such as department and
// dry_run.
func importWith(h *apitest.Harness, token, csv string, fields map[string]string) apitest.Response {
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	for name, value := range fields {
		_ = form.WriteField(name, value)
	}
	part, _ := form.CreateFormFile("file", "students.csv")
	_, _ = part.Write([]byte(csv))
	_ = form.Close()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/import", &body)
	request.Header.Set("Content-Type", form.FormDataContentType())
	request.Header.Set("Authorization", "Bearer "+token)
	return h.Send(request)
}

func checked(t *testing.T, response apitest.Response) checkedImport {
	t.Helper()
	if response.Status != http.StatusOK {
		t.Fatalf("import status = %d, want %d: %s", response.Status, http.StatusOK, response.Body)
	}
	var result struct {
		Data checkedImport `json:"data"`
	}
	response.Decode(t, &result)
	return result.Data
}

func countUsers(t *testing.T, h *apitest.Harness) int {
	t.Helper()
	var count int
	h.DB().Raw(`SELECT count(*) FROM users`).Scan(&count)
	return count
}

func TestImportDryRunChecksEveryRowAndSavesNothing(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	taken := student(t, h, "CS", 2023) // USN 4MN23CS001
	before := countUsers(t, h)

	csv := "email,full_name,usn\n" +
		"asha@gmail.com,Asha Rao,4mn23cs101\n" + // 2: ready
		"ravi@gmail.com,Ravi Kumar,4MN24CS102\n" + // 3: ready
		"dup@gmail.com,Taken USN,4MN23CS001\n" + // 4: already registered
		taken.Email + ",Taken Email,4MN23CS103\n" // 5: already registered
	result := checked(t, importWith(h, admin.Token, csv, map[string]string{"dry_run": "true"}))

	if !result.DryRun || result.Ready != 2 || result.Created != 0 || result.Failed != 2 {
		t.Fatalf("result = %+v, want a dry run with 2 ready, 0 created and 2 failed", result)
	}
	first := result.Rows[0]
	if first.Status != "ready" || first.USN != "4MN23CS101" || first.DepartmentCode != "CS" || first.BatchYear != 2023 {
		t.Errorf("row 2 = %+v, want ready, 4MN23CS101, CS, Batch 2023", first)
	}
	if result.Rows[2].Error != "the USN is already registered" || result.Rows[3].Error != "the email is already registered" {
		t.Errorf("rows 4 and 5 = %+v, want already registered", result.Rows[2:])
	}
	if after := countUsers(t, h); after != before {
		t.Errorf("users = %d after a dry run, want %d", after, before)
	}
	var audits int
	h.DB().Raw(`SELECT count(*) FROM audit_logs WHERE actor_id = ?`, admin.ID).Scan(&audits)
	if audits != 0 {
		t.Errorf("a dry run wrote %d audit logs, want none", audits)
	}

	// The same file saved for real creates the rows the check said were ready.
	saved := checked(t, importWith(h, admin.Token, csv, nil))
	if saved.DryRun || saved.Created != 2 || saved.Failed != 2 || saved.Rows[0].Status != "created" {
		t.Errorf("saved = %+v, want 2 created and 2 failed", saved)
	}
}

// The check step groups rows by the Department and Batch read from each USN
// and flags rows outside the Department the admin picked; saving creates
// only the rows inside it.
func TestImportForAChosenDepartmentFlagsRowsOutsideIt(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	csv := "email,full_name,usn\n" +
		"a@gmail.com,A,4MN24CS101\n" + // 2
		"b@gmail.com,B,4MN23CS102\n" + // 3
		"c@gmail.com,C,4MN24EC103\n" + // 4: outside
		"d@gmail.com,D,4MN24CS104\n" + // 5
		"bad-email,E,4MN24CS105\n" // 6: fails, still grouped

	check := checked(t, importWith(h, admin.Token, csv, map[string]string{"department": "cs", "dry_run": "true"}))
	if check.Department == nil || check.Department.Code != "CS" || check.Department.Name != "Computer Science and Engineering" {
		t.Fatalf("department = %+v, want CS", check.Department)
	}
	if check.Ready != 3 || check.Failed != 2 {
		t.Errorf("ready %d, failed %d; want 3 and 2", check.Ready, check.Failed)
	}
	outside := check.Rows[2]
	if outside.Status != "failed" || !outside.Outside || outside.Error != "the USN is in EC, not CS" {
		t.Errorf("row 4 = %+v, want failed and flagged outside CS", outside)
	}
	if check.Rows[4].Outside {
		t.Errorf("row 6 = %+v, a bad email isn't outside the Department", check.Rows[4])
	}

	type group struct {
		code           string
		batch          int
		rows, ready    int
		failed         int
		outside        bool
		departmentName string
	}
	want := []group{
		{"EC", 2024, 1, 0, 1, true, "Electronics and Communication Engineering"},
		{"CS", 2023, 1, 1, 0, false, "Computer Science and Engineering"},
		{"CS", 2024, 3, 2, 1, false, "Computer Science and Engineering"},
	}
	if len(check.Groups) != len(want) {
		t.Fatalf("groups = %+v, want %+v", check.Groups, want)
	}
	for i, w := range want {
		g := check.Groups[i]
		got := group{g.DepartmentCode, g.BatchYear, g.Rows, g.Ready, g.Failed, g.Outside, g.DepartmentName}
		if got != w {
			t.Errorf("group %d = %+v, want %+v", i, got, w)
		}
	}

	saved := checked(t, importWith(h, admin.Token, csv, map[string]string{"department": "CS"}))
	if saved.Created != 3 || saved.Failed != 2 || saved.Groups[0].Created != 0 || saved.Groups[2].Created != 2 {
		t.Errorf("saved = %+v, want the three CS rows created", saved)
	}
	var ec int
	h.DB().Raw(`SELECT count(*) FROM users WHERE email = 'c@gmail.com'`).Scan(&ec)
	if ec != 0 {
		t.Error("the EC row was created in a CS import")
	}
}

// An HOD's import is locked to their Department without naming it, and they
// can't name another.
func TestHODImportIsLockedToTheirDepartment(t *testing.T) {
	h := apitest.New(t)
	hod := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	csv := "email,full_name,usn\ncs@gmail.com,CS Student,4MN24CS301\nec@gmail.com,EC Student,4MN24EC302\n"

	check := checked(t, importWith(h, hod.Token, csv, map[string]string{"dry_run": "true"}))
	if check.Department == nil || check.Department.Code != "CS" {
		t.Fatalf("department = %+v, want CS without asking", check.Department)
	}
	if check.Rows[0].Status != "ready" || !check.Rows[1].Outside || check.Rows[1].Error != "the USN is in EC, not CS" {
		t.Errorf("rows = %+v, want CS ready and EC flagged", check.Rows)
	}
	if len(check.Groups) != 2 || check.Groups[0].DepartmentCode != "EC" || !check.Groups[0].Outside {
		t.Errorf("groups = %+v, want the EC group flagged first", check.Groups)
	}

	if response := importWith(h, hod.Token, csv, map[string]string{"department": "EC", "dry_run": "true"}); response.Status != http.StatusForbidden {
		t.Errorf("HOD naming EC: status = %d, want %d: %s", response.Status, http.StatusForbidden, response.Body)
	}
}

func TestImportRejectsAnUnknownDepartmentOrDryRunValue(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	csv := "email,full_name,usn\na@gmail.com,A,4MN24CS101\n"
	for name, fields := range map[string]map[string]string{
		"unknown department":  {"department": "ZZ"},
		"overlong department": {"department": "ABCDEFGHIJKL"},
		"dry_run not a bool":  {"dry_run": "maybe"},
	} {
		if response := importWith(h, admin.Token, csv, fields); response.Status != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want %d: %s", name, response.Status, http.StatusBadRequest, response.Body)
		}
	}
	if count := countUsers(t, h); count != 1 {
		t.Errorf("users = %d, a refused import created accounts", count)
	}
}
