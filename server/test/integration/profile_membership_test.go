package integration

import (
	"net/http"
	"testing"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

type profileMembership struct {
	Data struct {
		Username   string   `json:"username"`
		Roles      []string `json:"roles"`
		Department *struct {
			Code string `json:"code"`
			Name string `json:"name"`
		} `json:"department"`
		BatchYear *int `json:"batch_year"`
	} `json:"data"`
}

func profileOf(t *testing.T, h *apitest.Harness, token, username string) profileMembership {
	t.Helper()
	response := h.Do(t, http.MethodGet, "/api/v1/profiles/"+username, token, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("profile status = %d, want %d: %s", response.Status, http.StatusOK, response.Body)
	}
	var profile profileMembership
	response.Decode(t, &profile)
	return profile
}

func usernameOf(t *testing.T, h *apitest.Harness, userID string) string {
	t.Helper()
	var username string
	if err := h.DB().Raw(`SELECT username FROM profiles WHERE user_id = ?`, userID).Scan(&username).Error; err != nil {
		t.Fatalf("read username: %v", err)
	}
	return username
}

func TestProfileShowsRolesDepartmentAndBatchToMembers(t *testing.T) {
	h := apitest.New(t)
	viewer := member(t, h, "Viewer", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "student"}}, Student: &apitest.StudentSeed{DepartmentCode: "EC", BatchYear: 2024}})
	hod := member(t, h, "Asha Rao", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}, {Role: "hod", DepartmentCode: "CS"}}})
	coordinator := member(t, h, "Kavya M", apitest.UserSeed{
		Roles:   []apitest.RoleSeed{{Role: "student"}, {Role: "student_coordinator", DepartmentCode: "CS"}},
		Student: &apitest.StudentSeed{DepartmentCode: "CS", BatchYear: 2023},
	})

	staff := profileOf(t, h, viewer.Token, usernameOf(t, h, hod.ID)).Data
	if !sameNames(staff.Roles, []string{"hod", "faculty"}) {
		t.Errorf("HOD roles = %v, want [hod faculty], most senior first", staff.Roles)
	}
	if staff.Department == nil || staff.Department.Code != "CS" || staff.Department.Name == "" {
		t.Errorf("HOD department = %+v, want CS with its name", staff.Department)
	}
	if staff.BatchYear != nil {
		t.Errorf("HOD batch = %d, want none", *staff.BatchYear)
	}

	student := profileOf(t, h, viewer.Token, usernameOf(t, h, coordinator.ID)).Data
	if !sameNames(student.Roles, []string{"student_coordinator", "student"}) {
		t.Errorf("coordinator roles = %v, want [student_coordinator student]", student.Roles)
	}
	if student.Department == nil || student.Department.Code != "CS" {
		t.Errorf("coordinator department = %+v, want CS", student.Department)
	}
	if student.BatchYear == nil || *student.BatchYear != 2023 {
		t.Errorf("coordinator batch = %v, want 2023", student.BatchYear)
	}

	// The owner sees their own, even with a private profile.
	if err := h.DB().Exec(`UPDATE profiles SET public_profile_enabled = false WHERE user_id = ?`, coordinator.ID).Error; err != nil {
		t.Fatalf("hide profile: %v", err)
	}
	own := profileOf(t, h, coordinator.Token, usernameOf(t, h, coordinator.ID)).Data
	if own.BatchYear == nil || *own.BatchYear != 2023 || len(own.Roles) != 2 {
		t.Errorf("own profile = %+v, want roles and batch", own)
	}
}

func TestProfileHidesMembershipFromAnonymousVisitors(t *testing.T) {
	h := apitest.New(t)
	student := member(t, h, "Priya Kumar", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "student"}}, Student: &apitest.StudentSeed{DepartmentCode: "CS", BatchYear: 2023}})

	anonymous := profileOf(t, h, "", usernameOf(t, h, student.ID)).Data
	if len(anonymous.Roles) != 0 || anonymous.Department != nil || anonymous.BatchYear != nil {
		t.Errorf("anonymous profile = %+v, want no roles, department or batch", anonymous)
	}
}
