package integration

import (
	"net/http"
	"testing"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

// Counts on Home, the Department page and the admin's Departments list count
// who is on the lists, signed in or not, and every kind of staff (#213): a
// class of 120 read "12 students" until everyone had signed in, and the HOD
// wasn't counted as staff.

type departmentCounts struct{ students, staff, faculty int }

func countsOf(t *testing.T, h *apitest.Harness, principalToken, adminToken, readerToken, code string) (college, admin departmentCounts, overview departmentCounts) {
	t.Helper()
	var home struct {
		Data struct {
			College struct {
				Departments []struct {
					Code     string `json:"code"`
					Students int    `json:"students"`
					Staff    int    `json:"staff"`
				} `json:"departments"`
			} `json:"college"`
		} `json:"data"`
	}
	response := h.Do(t, http.MethodGet, "/api/v1/dashboard", principalToken, nil)
	expectStatus(t, "dashboard", response, http.StatusOK)
	response.Decode(t, &home)
	for _, d := range home.Data.College.Departments {
		if d.Code == code {
			college = departmentCounts{students: d.Students, staff: d.Staff}
		}
	}

	var list struct {
		Data struct {
			Departments []struct {
				Code     string `json:"code"`
				Students int    `json:"students"`
				Staff    int    `json:"staff"`
			} `json:"departments"`
		} `json:"data"`
	}
	response = h.Do(t, http.MethodGet, "/api/v1/admin/departments", adminToken, nil)
	expectStatus(t, "admin departments", response, http.StatusOK)
	response.Decode(t, &list)
	for _, d := range list.Data.Departments {
		if d.Code == code {
			admin = departmentCounts{students: d.Students, staff: d.Staff}
		}
	}

	var page struct {
		Data struct {
			Counts struct {
				Students int `json:"students"`
				Faculty  int `json:"faculty"`
				Staff    int `json:"staff"`
			} `json:"counts"`
		} `json:"data"`
	}
	response = h.Do(t, http.MethodGet, "/api/v1/departments/"+code+"/overview", readerToken, nil)
	expectStatus(t, "overview", response, http.StatusOK)
	response.Decode(t, &page)
	overview = departmentCounts{students: page.Data.Counts.Students, staff: page.Data.Counts.Staff, faculty: page.Data.Counts.Faculty}
	return college, admin, overview
}

func TestCountsIncludeEveryoneOnTheListsAndEveryKindOfStaff(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	// Added but not signed in yet.
	invite(t, h, admin.Token, "new.faculty@college.edu", "New Faculty", "faculty", "CS")
	imported(t, importCSV(h, admin.Token, "email,full_name,usn\na@gmail.com,A One,4MN24CS801\nb@gmail.com,B Two,4MN24CS802\nc@gmail.com,C Three,4MN24CS803\n"))

	college, list, overview := countsOf(t, h, principal.Token, admin.Token, faculty.Token, "CS")
	want := departmentCounts{students: 3, staff: 3}
	if college != want {
		t.Errorf("Home's college panel = %+v, want %+v", college, want)
	}
	if list != want {
		t.Errorf("admin Departments list = %+v, want %+v", list, want)
	}
	if overview != (departmentCounts{students: 3, staff: 3, faculty: 2}) {
		t.Errorf("Department page = %+v, want 3 students, 3 staff, 2 faculty", overview)
	}
}
