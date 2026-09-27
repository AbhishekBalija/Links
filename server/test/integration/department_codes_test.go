package integration

import (
	"net/http"
	"testing"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

func requestAccess(t *testing.T, h *apitest.Harness, usn, department string) apitest.Response {
	t.Helper()
	return h.Do(t, http.MethodPost, "/api/v1/auth/request-access", "", map[string]any{
		"email":           "request-" + usn + "@apitest.local",
		"password":        "SignUp123",
		"full_name":       "Requested " + usn,
		"usn":             usn,
		"department_code": department,
	})
}

func TestNewDepartmentCanBeUsedToRequestAccess(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	if response := h.Do(t, http.MethodPost, "/api/v1/admin/departments", admin.Token,
		map[string]string{"code": "IS", "name": "Information Science and Engineering"}); response.Status != http.StatusCreated {
		t.Fatalf("create department status = %d: %s", response.Status, response.Body)
	}

	userID := signUp(t, h, "4MN24IS001", "IS")
	var code string
	if err := h.DB().Raw(`SELECT d.code FROM student_identities s JOIN departments d ON d.id = s.department_id WHERE s.user_id = ?`, userID).Scan(&code).Error; err != nil {
		t.Fatalf("read department: %v", err)
	}
	if code != "IS" {
		t.Fatalf("student identity department = %q, want IS", code)
	}
}

func TestRequestAccessRejectsUnknownOrMismatchedDepartments(t *testing.T) {
	h := apitest.New(t)

	cases := map[string]apitest.Response{
		"unknown code in USN":     requestAccess(t, h, "4MN24ZZ001", "CS"),
		"unknown code, form too":  requestAccess(t, h, "4MN24ZZ002", "ZZ"),
		"USN and form disagree":   requestAccess(t, h, "4MN24CS003", "EC"),
		"bad USN format":          requestAccess(t, h, "4MN24C5004", "CS"),
		"USN year out of range":   requestAccess(t, h, "4MN99CS006", "CS"),
		"department code missing": requestAccess(t, h, "4MN24ME007", "XX"),
	}
	for name, response := range cases {
		if response.Status != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want %d: %s", name, response.Status, http.StatusBadRequest, response.Body)
		}
	}
}

func TestPublicDepartmentListNeedsNoToken(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	h.Do(t, http.MethodPost, "/api/v1/admin/departments", admin.Token, map[string]string{"code": "IS", "name": "Information Science and Engineering"})

	response := h.Do(t, http.MethodGet, "/api/v1/public/departments", "", nil)
	if response.Status != http.StatusOK {
		t.Fatalf("public list status = %d, want %d: %s", response.Status, http.StatusOK, response.Body)
	}
	if got := response.Header.Get("Cache-Control"); got != "public, max-age=300" {
		t.Errorf("Cache-Control = %q, want public, max-age=300", got)
	}
	var list struct {
		Data []map[string]any `json:"data"`
	}
	response.Decode(t, &list)
	codes := map[string]bool{}
	for _, department := range list.Data {
		if len(department) != 2 || department["code"] == nil || department["name"] == nil {
			t.Fatalf("public department = %v, want only code and name", department)
		}
		codes[department["code"].(string)] = true
	}
	for _, want := range []string{"CS", "EC", "IS"} {
		if !codes[want] {
			t.Errorf("department %s missing from the public list", want)
		}
	}
}
