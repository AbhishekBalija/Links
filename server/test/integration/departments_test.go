package integration

import (
	"net/http"
	"testing"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

func TestStudentCanListDepartmentsButNotCreateThem(t *testing.T) {
	h := apitest.New(t)
	student := h.SeedUser(t, apitest.UserSeed{
		Roles:   []apitest.RoleSeed{{Role: "student"}},
		Student: &apitest.StudentSeed{DepartmentCode: "CS", BatchYear: 2023},
	})

	list := h.Do(t, http.MethodGet, "/api/v1/departments", student.Token, nil)
	if list.Status != http.StatusOK {
		t.Fatalf("list status = %d, want %d: %s", list.Status, http.StatusOK, list.Body)
	}
	var listed struct {
		Data struct {
			Departments []struct {
				Code string `json:"code"`
			} `json:"departments"`
		} `json:"data"`
	}
	list.Decode(t, &listed)
	codes := map[string]bool{}
	for _, department := range listed.Data.Departments {
		codes[department.Code] = true
	}
	for _, want := range []string{"CS", "AD", "AI", "CV", "EC", "ME"} {
		if !codes[want] {
			t.Errorf("department %s missing from list", want)
		}
	}

	create := h.Do(t, http.MethodPost, "/api/v1/admin/departments", student.Token,
		map[string]string{"code": "IS", "name": "Information Science"})
	if create.Status != http.StatusForbidden {
		t.Fatalf("create status = %d, want %d: %s", create.Status, http.StatusForbidden, create.Body)
	}
}

func TestAdminCreatesDepartmentThatStudentsCanThenRead(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	student := h.SeedUser(t, apitest.UserSeed{
		Roles:   []apitest.RoleSeed{{Role: "student"}},
		Student: &apitest.StudentSeed{DepartmentCode: "EC", BatchYear: 2022},
	})

	create := h.Do(t, http.MethodPost, "/api/v1/admin/departments", admin.Token,
		map[string]string{"code": "IS", "name": "Information Science"})
	if create.Status != http.StatusCreated {
		t.Fatalf("create status = %d, want %d: %s", create.Status, http.StatusCreated, create.Body)
	}

	get := h.Do(t, http.MethodGet, "/api/v1/departments/IS", student.Token, nil)
	if get.Status != http.StatusOK {
		t.Fatalf("get status = %d, want %d: %s", get.Status, http.StatusOK, get.Body)
	}
}

type departmentBody struct {
	Code      string  `json:"code"`
	HODUserID *string `json:"hodUserId"`
}

func getDepartment(t *testing.T, h *apitest.Harness, code, token string) departmentBody {
	t.Helper()
	response := h.Do(t, http.MethodGet, "/api/v1/departments/"+code, token, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("get %s status = %d, want %d: %s", code, response.Status, http.StatusOK, response.Body)
	}
	var body struct {
		Data departmentBody `json:"data"`
	}
	response.Decode(t, &body)
	return body.Data
}

// A Department's HOD is whoever holds the HOD role scoped to it (#179), the
// same answer the admin's Departments screen gives.
func TestDepartmentNamesTheHODFromTheirRole(t *testing.T) {
	h := apitest.New(t)
	hod := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	viewer := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "EC"}}})

	if got := getDepartment(t, h, "CS", viewer.Token); got.HODUserID == nil || *got.HODUserID != hod.ID {
		t.Errorf("CS hodUserId = %v, want %s", got.HODUserID, hod.ID)
	}
	if got := getDepartment(t, h, "EC", viewer.Token); got.HODUserID != nil {
		t.Errorf("EC hodUserId = %s, want none", *got.HODUserID)
	}

	list := h.Do(t, http.MethodGet, "/api/v1/departments", viewer.Token, nil)
	var listed struct {
		Data struct {
			Departments []departmentBody `json:"departments"`
		} `json:"data"`
	}
	list.Decode(t, &listed)
	for _, department := range listed.Data.Departments {
		if department.Code == "CS" && (department.HODUserID == nil || *department.HODUserID != hod.ID) {
			t.Errorf("listed CS hodUserId = %v, want %s", department.HODUserID, hod.ID)
		}
	}
}

// Saving a Department changes its name and description, never its HOD: the
// HOD comes from the role, so leaving hodUserId out doesn't clear it (#179).
func TestSavingADepartmentKeepsItsHOD(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	hod := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})

	response := h.Do(t, http.MethodPut, "/api/v1/admin/departments/CS", admin.Token,
		map[string]any{"name": "Computer Science", "description": "Since 2004"})
	if response.Status != http.StatusOK {
		t.Fatalf("save status = %d, want %d: %s", response.Status, http.StatusOK, response.Body)
	}
	var saved struct {
		Data departmentBody `json:"data"`
	}
	response.Decode(t, &saved)
	if saved.Data.HODUserID == nil || *saved.Data.HODUserID != hod.ID {
		t.Errorf("saved hodUserId = %v, want %s", saved.Data.HODUserID, hod.ID)
	}
	if got := getDepartment(t, h, "CS", admin.Token); got.HODUserID == nil || *got.HODUserID != hod.ID {
		t.Errorf("CS hodUserId after saving = %v, want %s", got.HODUserID, hod.ID)
	}

	// Naming someone without the HOD role here is still refused.
	other := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	refused := h.Do(t, http.MethodPut, "/api/v1/admin/departments/CS", admin.Token,
		map[string]any{"name": "Computer Science", "hodUserId": other.ID})
	if refused.Status != http.StatusBadRequest {
		t.Errorf("naming a non-HOD: status = %d, want %d: %s", refused.Status, http.StatusBadRequest, refused.Body)
	}
}
