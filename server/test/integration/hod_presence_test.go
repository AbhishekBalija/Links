package integration

import (
	"net/http"
	"testing"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

// "Does this department have an HOD?" has one answer everywhere (#207): an
// HOD role in effect, the rule that routes event reviews and refuses a
// second HOD. Someone added as HOD who hasn't signed in yet, or whose
// account is paused, still counts, and Home says which.

type collegeHOD struct {
	Data struct {
		College struct {
			Departments []struct {
				Code string `json:"code"`
				HOD  *struct {
					FullName string `json:"full_name"`
					State    string `json:"state"`
				} `json:"hod"`
			} `json:"departments"`
			DepartmentsWithoutHOD int `json:"departments_without_hod"`
		} `json:"college"`
	} `json:"data"`
}

func collegeRow(t *testing.T, h *apitest.Harness, token, code string) (*struct {
	FullName string `json:"full_name"`
	State    string `json:"state"`
}, int) {
	t.Helper()
	response := h.Do(t, http.MethodGet, "/api/v1/dashboard", token, nil)
	expectStatus(t, "dashboard", response, http.StatusOK)
	var home collegeHOD
	response.Decode(t, &home)
	for _, department := range home.Data.College.Departments {
		if department.Code == code {
			return department.HOD, home.Data.College.DepartmentsWithoutHOD
		}
	}
	t.Fatalf("no %s row in %+v", code, home.Data.College)
	return nil, 0
}

func hasHOD(t *testing.T, h *apitest.Harness, token, code string) bool {
	t.Helper()
	response := h.Do(t, http.MethodGet, "/api/v1/departments/"+code+"/overview", token, nil)
	expectStatus(t, "overview", response, http.StatusOK)
	var body struct {
		Data struct {
			HasHOD bool `json:"has_hod"`
		} `json:"data"`
	}
	response.Decode(t, &body)
	return body.Data.HasHOD
}

func publicHasHOD(t *testing.T, h *apitest.Harness) map[string]bool {
	t.Helper()
	response := h.Do(t, http.MethodGet, "/api/v1/public/departments", "", nil)
	expectStatus(t, "public departments", response, http.StatusOK)
	var body struct {
		Data []struct {
			Code   string `json:"code"`
			HasHOD bool   `json:"has_hod"`
		} `json:"data"`
	}
	response.Decode(t, &body)
	found := map[string]bool{}
	for _, department := range body.Data {
		found[department.Code] = department.HasHOD
	}
	return found
}

func TestAnHODWhoHasntSignedInYetStillCounts(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	_, before := collegeRow(t, h, admin.Token, "CS")

	invite(t, h, admin.Token, "asha.rao@college.edu", "Asha Rao", "hod", "CS")

	hod, without := collegeRow(t, h, admin.Token, "CS")
	if hod == nil || hod.FullName != "Asha Rao" || hod.State != "not_signed_in" {
		t.Errorf("CS HOD on Home = %+v, want Asha Rao, not signed in yet", hod)
	}
	if without != before-1 {
		t.Errorf("departments without an HOD = %d, want %d", without, before-1)
	}
	if !hasHOD(t, h, faculty.Token, "CS") || hasHOD(t, h, faculty.Token, "EC") {
		t.Error("the department overview should say CS has an HOD and EC doesn't")
	}
	public := publicHasHOD(t, h)
	if !public["CS"] || public["EC"] {
		t.Errorf("public departments has_hod = %v, want CS only", public)
	}
}

func TestAPausedHODStillCountsAndHomeSaysSo(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	hod := member(t, h, "Dev Nair", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "EC"}}})
	if got, _ := collegeRow(t, h, admin.Token, "EC"); got == nil || got.State != "active" {
		t.Fatalf("EC HOD = %+v, want an active one", got)
	}
	if err := h.DB().Exec(`UPDATE users SET status = 'suspended' WHERE id = ?`, hod.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got, _ := collegeRow(t, h, admin.Token, "EC"); got == nil || got.FullName != "Dev Nair" || got.State != "paused" {
		t.Errorf("EC HOD = %+v, want Dev Nair, paused", got)
	}
}
