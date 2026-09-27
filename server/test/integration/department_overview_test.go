package integration

import (
	"net/http"
	"testing"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

type overview struct {
	Department struct {
		Code        string  `json:"code"`
		Name        string  `json:"name"`
		Description *string `json:"description"`
	} `json:"department"`
	HOD    *directoryEntry `json:"hod"`
	Counts struct {
		Students        int `json:"students"`
		Faculty         int `json:"faculty"`
		StudentsByBatch []struct {
			BatchYear int `json:"batch_year"`
			Count     int `json:"count"`
		} `json:"students_by_batch"`
	} `json:"counts"`
	Staff []directoryEntry `json:"staff"`
}

func departmentOverview(t *testing.T, h *apitest.Harness, token, code string) overview {
	t.Helper()
	response := h.Do(t, http.MethodGet, "/api/v1/departments/"+code+"/overview", token, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("overview status = %d, want %d: %s", response.Status, http.StatusOK, response.Body)
	}
	var body struct {
		Data overview `json:"data"`
	}
	response.Decode(t, &body)
	return body.Data
}

func TestDepartmentOverviewCountsEveryoneButListsOnlyVisibleStaff(t *testing.T) {
	h := apitest.New(t)
	viewer := member(t, h, "Zed Viewer", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "student"}}, Student: &apitest.StudentSeed{DepartmentCode: "EC", BatchYear: 2023}})
	h.DB().Exec(`UPDATE departments SET description = 'Computing since 2001' WHERE code = 'CS'`)

	member(t, h, "Hema HOD", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}, {Role: "faculty", DepartmentCode: "CS"}}})
	member(t, h, "Farah Faculty", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	member(t, h, "Anil Faculty", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	member(t, h, "Pooja Placement", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "placement_officer", DepartmentCode: "CS"}}})
	member(t, h, "Global Placement", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "placement_officer"}}})
	member(t, h, "Eshan EC Faculty", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "EC"}}})
	hiddenFaculty := member(t, h, "Hidden Faculty", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	h.DB().Exec(`UPDATE profiles SET public_profile_enabled = false WHERE user_id = ?`, hiddenFaculty.ID)

	cs2023 := apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "student"}}, Student: &apitest.StudentSeed{DepartmentCode: "CS", BatchYear: 2023}}
	member(t, h, "Stu One", cs2023)
	hiddenStudent := member(t, h, "Stu Hidden", cs2023)
	h.DB().Exec(`UPDATE profiles SET public_profile_enabled = false WHERE user_id = ?`, hiddenStudent.ID)
	suspended := member(t, h, "Stu Suspended", cs2023)
	h.DB().Exec(`UPDATE users SET status = 'suspended' WHERE id = ?`, suspended.ID)
	member(t, h, "Stu Coordinator", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "student"}, {Role: "student_coordinator", DepartmentCode: "CS"}}, Student: &apitest.StudentSeed{DepartmentCode: "CS", BatchYear: 2024}})
	graduate := member(t, h, "Stu Graduate", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "alumni"}}, Student: &apitest.StudentSeed{DepartmentCode: "CS", BatchYear: 2019}})
	_ = graduate

	got := departmentOverview(t, h, viewer.Token, "cs")
	if got.Department.Code != "CS" || got.Department.Name == "" || got.Department.Description == nil || *got.Department.Description != "Computing since 2001" {
		t.Errorf("department = %+v, want CS with its description", got.Department)
	}
	if got.HOD == nil || got.HOD.FullName != "Hema HOD" {
		t.Errorf("hod = %+v, want Hema HOD", got.HOD)
	}
	// Counts include hidden profiles but not suspended accounts, alumni or
	// other Departments; faculty counts the HOD, who also teaches.
	if got.Counts.Students != 3 || got.Counts.Faculty != 4 {
		t.Errorf("counts = %+v, want 3 students and 4 faculty", got.Counts)
	}
	batches := map[int]int{}
	for _, batch := range got.Counts.StudentsByBatch {
		batches[batch.BatchYear] = batch.Count
	}
	if len(batches) != 2 || batches[2023] != 2 || batches[2024] != 1 {
		t.Errorf("students by batch = %v, want 2023: 2, 2024: 1", batches)
	}
	if len(got.Counts.StudentsByBatch) == 2 && got.Counts.StudentsByBatch[0].BatchYear != 2023 {
		t.Errorf("students by batch = %+v, want oldest Batch first", got.Counts.StudentsByBatch)
	}

	staff := []string{}
	for _, entry := range got.Staff {
		staff = append(staff, entry.FullName)
	}
	if want := []string{"Hema HOD", "Pooja Placement", "Anil Faculty", "Farah Faculty"}; !sameNames(staff, want) {
		t.Errorf("staff = %v, want %v (seniority, then name; visible only)", staff, want)
	}
}

func TestDepartmentOverviewWithoutAVisibleHOD(t *testing.T) {
	h := apitest.New(t)
	viewer := member(t, h, "Zed Viewer", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "student"}}})
	hod := member(t, h, "Hidden HOD", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "EC"}}})
	h.DB().Exec(`UPDATE profiles SET public_profile_enabled = false WHERE user_id = ?`, hod.ID)

	if got := departmentOverview(t, h, viewer.Token, "EC"); got.HOD != nil || len(got.Staff) != 0 {
		t.Errorf("EC overview = %+v, want no HOD and no staff shown", got)
	}
	got := departmentOverview(t, h, viewer.Token, "ME")
	if got.HOD != nil || got.Counts.Students != 0 || got.Counts.StudentsByBatch == nil || got.Staff == nil {
		t.Errorf("ME overview = %+v, want no HOD, zero counts and empty lists", got)
	}
}

func TestDepartmentOverviewErrors(t *testing.T) {
	h := apitest.New(t)
	viewer := member(t, h, "Zed Viewer", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "student"}}})
	if response := h.Do(t, http.MethodGet, "/api/v1/departments/ZZ/overview", viewer.Token, nil); response.Status != http.StatusNotFound {
		t.Errorf("unknown code status = %d, want %d", response.Status, http.StatusNotFound)
	}
	if response := h.Do(t, http.MethodGet, "/api/v1/departments/CS/overview", "", nil); response.Status != http.StatusUnauthorized {
		t.Errorf("no token status = %d, want %d", response.Status, http.StatusUnauthorized)
	}
}
