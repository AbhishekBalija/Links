package integration

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

type importResult struct {
	Created int `json:"created"`
	Failed  int `json:"failed"`
	Rows    []struct {
		Row    int    `json:"row"`
		Email  string `json:"email"`
		Status string `json:"status"`
		Error  string `json:"error"`
	} `json:"rows"`
}

func importCSV(h *apitest.Harness, token, csv string) apitest.Response {
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	part, _ := form.CreateFormFile("file", "students.csv")
	_, _ = part.Write([]byte(csv))
	_ = form.Close()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/import", &body)
	request.Header.Set("Content-Type", form.FormDataContentType())
	request.Header.Set("Authorization", "Bearer "+token)
	return h.Send(request)
}

func imported(t *testing.T, response apitest.Response) importResult {
	t.Helper()
	if response.Status != http.StatusOK {
		t.Fatalf("import status = %d, want %d: %s", response.Status, http.StatusOK, response.Body)
	}
	var result struct {
		Data importResult `json:"data"`
	}
	response.Decode(t, &result)
	return result.Data
}

func TestAdminImportCreatesPendingVerifiedStudents(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})

	result := imported(t, importCSV(h, admin.Token, "email,full_name,usn\nasha@gmail.com,Asha Rao,4mn23cs101\nravi@gmail.com,Ravi Kumar,4MN24EC102\n"))
	if result.Created != 2 || result.Failed != 0 || len(result.Rows) != 2 {
		t.Fatalf("result = %+v, want 2 created", result)
	}
	if result.Rows[0].Row != 2 || result.Rows[0].Status != "created" || result.Rows[0].Email != "asha@gmail.com" {
		t.Fatalf("first row = %+v, want spreadsheet row 2 created", result.Rows[0])
	}

	var student struct {
		Status     string
		IsVerified bool
		CreatedBy  string
		USN        string
		Code       string
		BatchYear  int
		Role       string
	}
	err := h.DB().Raw(`
		SELECT u.status, u.is_verified, u.created_by, s.usn, d.code, s.batch_year, r.role
		FROM users u
		JOIN student_identities s ON s.user_id = u.id
		JOIN departments d ON d.id = s.department_id
		JOIN role_assignments r ON r.user_id = u.id
		WHERE u.email = 'asha@gmail.com'`).Scan(&student).Error
	if err != nil {
		t.Fatalf("read student: %v", err)
	}
	if student.Status != "pending" || !student.IsVerified || student.CreatedBy != admin.ID {
		t.Errorf("user = %+v, want pending, verified, created by the admin", student)
	}
	if student.USN != "4MN23CS101" || student.Code != "CS" || student.BatchYear != 2023 || student.Role != "student" {
		t.Errorf("identity = %+v, want 4MN23CS101 in CS, Batch 2023, student role", student)
	}

	var audits struct{ Import, Users int }
	h.DB().Raw(`SELECT
		(SELECT count(*) FROM audit_logs WHERE action = 'students_imported' AND actor_id = ?) AS import,
		(SELECT count(*) FROM audit_logs WHERE action = 'user_imported' AND actor_id = ?) AS users`, admin.ID, admin.ID).Scan(&audits)
	if audits.Import != 1 || audits.Users != 2 {
		t.Errorf("audit logs = %+v, want one import and two user rows", audits)
	}
}

func TestImportReportsEachBadRowAndKeepsTheGoodOnes(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	existing := h.SeedUser(t, apitest.UserSeed{})
	existingStudent := student(t, h, "CS", 2023) // gets USN 4MN23CS001 from the harness

	rows := []string{
		"email,full_name,usn",
		"good@gmail.com,Good Row,4MN23CS201",        // 2
		"not-an-email,Bad Email,4MN23CS202",         // 3
		"nousn@gmail.com,No USN,",                   // 4
		"badusn@gmail.com,Bad USN,4MN23C5203",       // 5
		"unknown@gmail.com,Unknown Dept,4MN23ZZ204", // 6
		existing.Email + ",Taken Email,4MN23CS205",  // 7
		"takenusn@gmail.com,Taken USN,4mn23cs001",   // 8
		"GOOD@gmail.com,Repeat Email,4MN23CS206",    // 9
		"repeatusn@gmail.com,Repeat USN,4MN23CS201", // 10
		"noname@gmail.com,  ,4MN23CS207",            // 11
		"old@gmail.com,Old Batch,4MN99CS208",        // 12
		"second@gmail.com,Second Good,4MN22EC209",   // 13
	}
	_ = existingStudent
	result := imported(t, importCSV(h, admin.Token, strings.Join(rows, "\n")))
	if result.Created != 2 || result.Failed != 10 {
		t.Fatalf("result = %+v, want 2 created and 10 failed", result)
	}
	for _, row := range result.Rows {
		wantCreated := row.Row == 2 || row.Row == 13
		if wantCreated != (row.Status == "created") {
			t.Errorf("row %d status = %q (%s)", row.Row, row.Status, row.Error)
		}
		if row.Status == "failed" && row.Error == "" {
			t.Errorf("row %d failed without an error message", row.Row)
		}
	}
	var count int
	h.DB().Raw(`SELECT count(*) FROM users WHERE email IN ('good@gmail.com', 'second@gmail.com')`).Scan(&count)
	if count != 2 {
		t.Fatalf("created users = %d, want 2", count)
	}
}

func TestHODImportsOnlyTheirOwnDepartment(t *testing.T) {
	h := apitest.New(t)
	hod := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})

	result := imported(t, importCSV(h, hod.Token, "email,full_name,usn\ncs@gmail.com,CS Student,4MN24CS301\nec@gmail.com,EC Student,4MN24EC302\n"))
	if result.Created != 1 || result.Rows[0].Status != "created" || result.Rows[1].Status != "failed" {
		t.Fatalf("result = %+v, want the CS row created and the EC row refused", result)
	}
}

// An HOD who is also the principal still imports only their own
// Department: being principal no longer widens an import (ADR 0029).
func TestPrincipalWhoIsAnHODImportsOnlyTheirOwnDepartment(t *testing.T) {
	h := apitest.New(t)
	both := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}, {Role: "hod", DepartmentCode: "CS"}}})

	result := imported(t, importCSV(h, both.Token, "email,full_name,usn\ncs@gmail.com,CS Student,4MN24CS301\nec@gmail.com,EC Student,4MN24EC302\n"))
	if result.Created != 1 || result.Rows[0].Status != "created" || result.Rows[1].Status != "failed" {
		t.Fatalf("result = %+v, want the CS row created and the EC row refused", result)
	}
}

// Admins and HODs import students; the principal no longer does (ADR 0029).
func TestImportIsForbiddenToOtherRoles(t *testing.T) {
	h := apitest.New(t)
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	reader := student(t, h, "CS", 2023)
	for name, token := range map[string]string{"principal": principal.Token, "faculty": faculty.Token, "student": reader.Token} {
		if response := importCSV(h, token, "email,full_name,usn\na@gmail.com,A,4MN24CS401\n"); response.Status != http.StatusForbidden {
			t.Errorf("%s: status = %d, want %d: %s", name, response.Status, http.StatusForbidden, response.Body)
		}
	}
	var count int
	h.DB().Raw(`SELECT count(*) FROM users WHERE email = 'a@gmail.com'`).Scan(&count)
	if count != 0 {
		t.Errorf("a refused import created %d accounts", count)
	}
}

func TestImportRejectsBadFilesWhole(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})

	var tooMany strings.Builder
	tooMany.WriteString("email,full_name,usn\n")
	for i := range 201 {
		fmt.Fprintf(&tooMany, "s%d@gmail.com,Student %d,4MN24CS%03d\n", i, i, i)
	}
	bad := map[string]string{
		"empty file":     "",
		"header only":    "email,full_name,usn\n",
		"missing column": "email,full_name\na@gmail.com,A\n",
		"unknown column": "email,full_name,usn,phone\na@gmail.com,A,4MN24CS401,123\n",
		"ragged row":     "email,full_name,usn\na@gmail.com,A\n",
		"over 200 rows":  tooMany.String(),
		"over one MB":    "email,full_name,usn\n" + strings.Repeat("a", 1_000_001),
		"broken quotes":  "email,full_name,usn\n\"a@gmail.com,A,4MN24CS401\n",
	}
	for name, csv := range bad {
		if response := importCSV(h, admin.Token, csv); response.Status != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want %d: %.200s", name, response.Status, http.StatusBadRequest, response.Body)
		}
	}
	var count int
	h.DB().Raw(`SELECT count(*) FROM users`).Scan(&count)
	if count != 1 {
		t.Fatalf("users = %d, a rejected file created accounts", count)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/import", strings.NewReader("{}"))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+admin.Token)
	if response := h.Send(request); response.Status != http.StatusBadRequest {
		t.Errorf("no file status = %d, want %d: %s", response.Status, http.StatusBadRequest, response.Body)
	}
}

// The API's WriteTimeout is 30 seconds (cmd/api/main.go). A full file must
// finish well inside it, or the admin gets a broken response instead of the
// report while the accounts are still created.
func TestAFullImportFinishesWellInsideTheWriteTimeout(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})

	var file strings.Builder
	file.WriteString("email,full_name,usn\n")
	for i := range 200 {
		fmt.Fprintf(&file, "full%d@gmail.com,Student %d,4MN24CS%03d\n", i, i, i)
	}
	started := time.Now()
	result := imported(t, importCSV(h, admin.Token, file.String()))
	took := time.Since(started)

	if result.Created != 200 || result.Failed != 0 {
		t.Fatalf("created %d, failed %d; want 200 and 0", result.Created, result.Failed)
	}
	t.Logf("200 rows took %s (%s per row)", took, took/200)
	if took > 10*time.Second {
		t.Errorf("200 rows took %s, want well under the 30 s WriteTimeout", took)
	}
}

// Two students with the same name in one class list both get in: their
// usernames differ (#166).
func TestImportAddsStudentsWhoShareAName(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	csv := "email,full_name,usn\n"
	for i := 0; i < 30; i++ {
		csv += fmt.Sprintf("rahul%02d@gmail.com,Rahul K,4MN23CS%03d\n", i, 200+i)
	}

	result := imported(t, importCSV(h, admin.Token, csv))
	if result.Created != 30 || result.Failed != 0 {
		for _, row := range result.Rows {
			if row.Error != "" {
				t.Logf("row %d: %s", row.Row, row.Error)
			}
		}
		t.Fatalf("created %d, failed %d; want all 30 Rahul Ks created", result.Created, result.Failed)
	}
	var usernames int
	h.DB().Raw(`SELECT count(DISTINCT p.username) FROM profiles p JOIN users u ON u.id = p.user_id WHERE u.email LIKE 'rahul%@gmail.com'`).Scan(&usernames)
	if usernames != 30 {
		t.Errorf("distinct usernames = %d, want 30", usernames)
	}
}
