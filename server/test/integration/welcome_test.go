package integration

import (
	"net/http"
	"testing"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

// The coordinator welcome: the first time a student opens LINKS after
// becoming a student coordinator, Home says so, once per account.

type newRoleHome struct {
	Data struct {
		NewRole *struct {
			ID         string `json:"id"`
			Role       string `json:"role"`
			Department struct {
				Code string `json:"code"`
				Name string `json:"name"`
			} `json:"department"`
			AssignedBy *string `json:"assigned_by"`
		} `json:"new_role"`
	} `json:"data"`
}

func homeNewRole(t *testing.T, h *apitest.Harness, token string) newRoleHome {
	t.Helper()
	response := h.Do(t, http.MethodGet, "/api/v1/dashboard", token, nil)
	expectStatus(t, "dashboard", response, http.StatusOK)
	var home newRoleHome
	response.Decode(t, &home)
	return home
}

func welcomed(t *testing.T, h *apitest.Harness, token, roleID string) apitest.Response {
	t.Helper()
	return h.Do(t, http.MethodPost, "/api/v1/me/roles/"+roleID+"/welcomed", token, nil)
}

func TestANewCoordinatorIsWelcomedOnce(t *testing.T) {
	h := apitest.New(t)
	hod := member(t, h, "Asha Rao", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	rohan := student(t, h, "CS", 2024)
	if home := homeNewRole(t, h, rohan.Token); home.Data.NewRole != nil {
		t.Fatalf("before the role, new_role = %+v, want none", home.Data.NewRole)
	}

	grantedID(t, grantRole(t, h, hod.Token, rohan.ID, map[string]any{"role": "student_coordinator", "scope_type": "department", "scope_id": h.DepartmentID(t, "CS")}))
	role := homeNewRole(t, h, rohan.Token).Data.NewRole
	if role == nil || role.Role != "student_coordinator" || role.Department.Code != "CS" || role.Department.Name == "" || role.AssignedBy == nil || *role.AssignedBy != "Asha Rao" {
		t.Fatalf("new_role = %+v, want the CS coordinator role given by Asha Rao", role)
	}

	expectStatus(t, "got it", welcomed(t, h, rohan.Token, role.ID), http.StatusOK)
	if home := homeNewRole(t, h, rohan.Token); home.Data.NewRole != nil {
		t.Errorf("after Got it, new_role = %+v, want none", home.Data.NewRole)
	}
	// A second tap, from another device, is fine.
	expectStatus(t, "got it again", welcomed(t, h, rohan.Token, role.ID), http.StatusOK)
}

func TestOnlyTheRoleHolderMarksTheirWelcome(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	rohan, priya := student(t, h, "CS", 2024), student(t, h, "CS", 2024)
	roleID := grantedID(t, grantRole(t, h, admin.Token, rohan.ID, map[string]any{"role": "student_coordinator", "scope_type": "department", "scope_id": h.DepartmentID(t, "CS")}))

	expectStatus(t, "someone else", welcomed(t, h, priya.Token, roleID), http.StatusNotFound)
	expectStatus(t, "not an id", welcomed(t, h, rohan.Token, "nope"), http.StatusNotFound)
	if home := homeNewRole(t, h, rohan.Token); home.Data.NewRole == nil {
		t.Error("the holder's welcome was marked by someone else")
	}
}

func TestOtherRolesGetNoWelcome(t *testing.T) {
	h := apitest.New(t)
	teacher := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	if home := homeNewRole(t, h, teacher.Token); home.Data.NewRole != nil {
		t.Errorf("new_role = %+v, want none for faculty", home.Data.NewRole)
	}
}
