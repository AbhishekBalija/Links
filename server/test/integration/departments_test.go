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
