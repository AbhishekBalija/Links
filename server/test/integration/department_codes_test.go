package integration

import (
	"net/http"
	"strings"
	"testing"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

// requestAccess proves an email on no list with a code and sends an Access
// request for the USN.
func requestAccess(t *testing.T, h *apitest.Harness, usn string) apitest.Response {
	t.Helper()
	email := "request-" + strings.ToLower(usn) + "@apitest.local"
	_, signIn := signInWithCode(t, h, email)
	_, _, token := notOnList(t, signIn)
	return sendAccessRequest(t, h, token, usn, "Requested "+usn)
}

func TestNewDepartmentCanBeUsedToRequestAccess(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	if response := h.Do(t, http.MethodPost, "/api/v1/admin/departments", admin.Token,
		map[string]string{"code": "IS", "name": "Information Science and Engineering"}); response.Status != http.StatusCreated {
		t.Fatalf("create department status = %d: %s", response.Status, response.Body)
	}

	userID := signUp(t, h, "4MN24IS001")
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
		"unknown code in USN":     requestAccess(t, h, "4MN24ZZ001"),
		"bad USN format":          requestAccess(t, h, "4MN24C5004"),
		"USN year out of range":   requestAccess(t, h, "4MN99CS006"),
		"department code missing": requestAccess(t, h, "4MN24XX007"),
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
		// Code, name and whether requests go to an HOD (#207); no IDs or
		// people's names before anyone has signed in.
		if len(department) != 3 || department["code"] == nil || department["name"] == nil || department["has_hod"] == nil {
			t.Fatalf("public department = %v, want only code, name and has_hod", department)
		}
		codes[department["code"].(string)] = true
	}
	for _, want := range []string{"CS", "EC", "IS"} {
		if !codes[want] {
			t.Errorf("department %s missing from the public list", want)
		}
	}
}
