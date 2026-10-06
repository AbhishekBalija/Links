package integration

import (
	"net/http"
	"testing"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

type adminDepartment struct {
	ID          string  `json:"id"`
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	HOD         *struct {
		UserID   string `json:"user_id"`
		FullName string `json:"full_name"`
		Username string `json:"username"`
	} `json:"hod"`
	Students int `json:"students"`
	Staff    int `json:"staff"`
}

func adminDepartments(t *testing.T, h *apitest.Harness, token string) map[string]adminDepartment {
	t.Helper()
	response := h.Do(t, http.MethodGet, "/api/v1/admin/departments", token, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("admin departments status = %d, want %d: %s", response.Status, http.StatusOK, response.Body)
	}
	var body struct {
		Data struct {
			Departments []adminDepartment `json:"departments"`
		} `json:"data"`
	}
	response.Decode(t, &body)
	byCode := map[string]adminDepartment{}
	for _, department := range body.Data.Departments {
		byCode[department.Code] = department
	}
	return byCode
}

// The admin's Departments screen shows each Department's HOD (or none) and
// how many students and staff it has, counted like Home's college panel.
func TestAdminDepartmentsListShowsHODAndCounts(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	hod := member(t, h, "Meera Iyer", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	// A private profile still names the HOD on the admin's screen.
	if err := h.DB().Exec(`UPDATE profiles SET public_profile_enabled = false WHERE user_id = ?`, hod.ID).Error; err != nil {
		t.Fatalf("hide HOD: %v", err)
	}
	student(t, h, "CS", 2023)
	student(t, h, "CS", 2024)
	// An Access request still waiting for a decision isn't counted (#213).
	waiting := student(t, h, "CS", 2024)
	if err := h.DB().Exec(`UPDATE users SET status = 'pending', is_verified = false WHERE id = ?`, waiting.ID).Error; err != nil {
		t.Fatalf("make pending: %v", err)
	}
	h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "EC"}}})

	departments := adminDepartments(t, h, admin.Token)
	cs, ec := departments["CS"], departments["EC"]
	if cs.ID != h.DepartmentID(t, "CS") || cs.Name != "Computer Science and Engineering" {
		t.Errorf("CS = %+v, want its ID and name", cs)
	}
	if cs.HOD == nil || cs.HOD.UserID != hod.ID || cs.HOD.FullName != "Meera Iyer" || cs.HOD.Username == "" {
		t.Errorf("CS HOD = %+v, want Meera Iyer", cs.HOD)
	}
	// Staff counts the HOD as well as faculty (#213).
	if cs.Students != 2 || cs.Staff != 2 {
		t.Errorf("CS counts = %d students, %d staff; want 2 and 2 (the HOD is staff; a waiting request isn't counted)", cs.Students, cs.Staff)
	}
	if ec.HOD != nil || ec.Students != 0 || ec.Staff != 1 {
		t.Errorf("EC = %+v, want no HOD, 0 students, 1 staff", ec)
	}
	if len(departments) != 6 {
		t.Errorf("departments = %d, want the six seeded ones", len(departments))
	}
}

func TestOnlyAdminsListDepartmentsForManagement(t *testing.T) {
	h := apitest.New(t)
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	hod := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	for name, token := range map[string]string{"principal": principal.Token, "HOD": hod.Token} {
		if response := h.Do(t, http.MethodGet, "/api/v1/admin/departments", token, nil); response.Status != http.StatusForbidden {
			t.Errorf("%s: status = %d, want %d", name, response.Status, http.StatusForbidden)
		}
	}
}

// Renaming changes only the name: the code, description and HOD stay.
func TestAdminRenamesADepartment(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	hod := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	if err := h.DB().Exec(`UPDATE departments SET description = 'Since 2004', hod_user_id = ? WHERE code = 'CS'`, hod.ID).Error; err != nil {
		t.Fatalf("describe CS: %v", err)
	}

	response := h.Do(t, http.MethodPatch, "/api/v1/admin/departments/cs", admin.Token, map[string]string{"name": "  Computer Science  "})
	if response.Status != http.StatusOK {
		t.Fatalf("rename status = %d, want %d: %s", response.Status, http.StatusOK, response.Body)
	}
	var stored struct {
		Code, Name, Description, HODUserID string
	}
	h.DB().Raw(`SELECT code, name, description, hod_user_id::text AS hod_user_id FROM departments WHERE id = ?`, h.DepartmentID(t, "CS")).Scan(&stored)
	if stored.Code != "CS" || stored.Name != "Computer Science" || stored.Description != "Since 2004" || stored.HODUserID != hod.ID {
		t.Errorf("CS = %+v, want only the name changed", stored)
	}
	if got := adminDepartments(t, h, admin.Token)["CS"]; got.Name != "Computer Science" || got.HOD == nil {
		t.Errorf("listed CS = %+v, want the new name and its HOD", got)
	}
	var audits int
	h.DB().Raw(`SELECT count(*) FROM audit_logs WHERE action = 'department.updated' AND actor_id = ?`, admin.ID).Scan(&audits)
	if audits != 1 {
		t.Errorf("audit logs = %d, want one department.updated", audits)
	}

	// The same code in the body is not a change.
	same := h.Do(t, http.MethodPatch, "/api/v1/admin/departments/CS", admin.Token, map[string]string{"code": "cs", "name": "Computer Science and Engineering"})
	if same.Status != http.StatusOK {
		t.Errorf("rename with its own code: status = %d, want %d: %s", same.Status, http.StatusOK, same.Body)
	}

	for name, body := range map[string]map[string]string{
		"empty name":    {"name": " "},
		"one letter":    {"name": "C"},
		"no name field": {},
	} {
		if response := h.Do(t, http.MethodPatch, "/api/v1/admin/departments/CS", admin.Token, body); response.Status != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want %d: %s", name, response.Status, http.StatusBadRequest, response.Body)
		}
	}
	if response := h.Do(t, http.MethodPatch, "/api/v1/admin/departments/ZZ", admin.Token, map[string]string{"name": "Nowhere"}); response.Status != http.StatusNotFound {
		t.Errorf("unknown department: status = %d, want %d", response.Status, http.StatusNotFound)
	}
	if response := h.Do(t, http.MethodPatch, "/api/v1/admin/departments/CS", hod.Token, map[string]string{"name": "Mine Now"}); response.Status != http.StatusForbidden {
		t.Errorf("HOD renaming: status = %d, want %d", response.Status, http.StatusForbidden)
	}
}

// A Department code never changes once created (ADR 0021): it is in every
// USN of the Department. Neither PATCH nor PUT can change it.
func TestDepartmentCodeNeverChanges(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})

	for _, method := range []string{http.MethodPatch, http.MethodPut} {
		response := h.Do(t, method, "/api/v1/admin/departments/CS", admin.Token, map[string]string{"code": "CE", "name": "Computer Engineering"})
		if response.Status != http.StatusBadRequest {
			t.Errorf("%s with a new code: status = %d, want %d: %s", method, response.Status, http.StatusBadRequest, response.Body)
		}
	}
	var stored struct{ Code, Name string }
	h.DB().Raw(`SELECT code, name FROM departments WHERE id = ?`, h.DepartmentID(t, "CS")).Scan(&stored)
	if stored.Code != "CS" || stored.Name != "Computer Science and Engineering" {
		t.Errorf("CS = %+v, want it unchanged", stored)
	}
}

// A Department code is the two letters a USN carries (4MN24IS001), so a
// longer code would make a Department no student could ever join.
func TestNewDepartmentCodesAreTwoLetters(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	for _, code := range []string{"ISE", "I", "I5"} {
		response := h.Do(t, http.MethodPost, "/api/v1/admin/departments", admin.Token, map[string]string{"code": code, "name": "Information Science"})
		if response.Status != http.StatusBadRequest {
			t.Errorf("code %q: status = %d, want %d: %s", code, response.Status, http.StatusBadRequest, response.Body)
		}
	}
	created := h.Do(t, http.MethodPost, "/api/v1/admin/departments", admin.Token, map[string]string{"code": "is", "name": "Information Science"})
	if created.Status != http.StatusCreated {
		t.Fatalf("code is: status = %d, want %d: %s", created.Status, http.StatusCreated, created.Body)
	}
	if got := adminDepartments(t, h, admin.Token)["IS"]; got.HOD != nil || got.Students != 0 || got.Staff != 0 {
		t.Errorf("new IS = %+v, want no HOD and no members", got)
	}
}
