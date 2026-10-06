package integration

import (
	"net/http"
	"testing"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

// A malformed user ID is a missing user, not a server error, and the scope an
// approval gives the student role is checked (#180).

func departmentID(t *testing.T, h *apitest.Harness, code string) string {
	t.Helper()
	var id string
	h.DB().Raw(`SELECT id FROM departments WHERE code = ?`, code).Scan(&id)
	if id == "" {
		t.Fatalf("no department %s", code)
	}
	return id
}

func TestAMalformedUserIDOnVerifyOrStatusIsNotFound(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})

	expectStatus(t, "verify", h.Do(t, http.MethodPatch, "/api/v1/admin/users/not-a-uuid/verify", admin.Token, nil), http.StatusNotFound)
	body := map[string]string{"status": "suspended", "note": "Misuse"}
	expectStatus(t, "status", h.Do(t, http.MethodPatch, "/api/v1/admin/users/not-a-uuid/status", admin.Token, body), http.StatusNotFound)
}

func TestApprovingWithAnUnknownScopeIsRefusedWithTheField(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	request := signUp(t, h, "4MN23CS821")
	verify := func(body map[string]string) apitest.Response {
		return h.Do(t, http.MethodPatch, "/api/v1/admin/users/"+request+"/verify", admin.Token, body)
	}

	if details := refusedWith(t, verify(map[string]string{"scope_type": "galaxy"}), http.StatusBadRequest); details["scope_type"] == "" {
		t.Errorf("unknown scope type: details = %v, want scope_type named", details)
	}
	if details := refusedWith(t, verify(map[string]string{"scope_type": "global", "scope_id": departmentID(t, h, "CS")}), http.StatusBadRequest); details["scope_id"] == "" {
		t.Errorf("global scope with an ID: details = %v, want scope_id named", details)
	}
	if details := refusedWith(t, verify(map[string]string{"scope_type": "department"}), http.StatusBadRequest); details["scope_id"] == "" {
		t.Errorf("department scope without an ID: details = %v, want scope_id named", details)
	}

	// Nothing was approved by the refusals, so a valid approval still works.
	expectStatus(t, "valid approval", verify(map[string]string{"scope_type": "department", "scope_id": departmentID(t, h, "CS")}), http.StatusOK)
}

func TestAnHODCannotScopeAnApprovalToAnotherDepartment(t *testing.T) {
	h := apitest.New(t)
	hod := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	request := signUp(t, h, "4MN23CS822")
	verify := func(departmentCode string) apitest.Response {
		body := map[string]string{"scope_type": "department", "scope_id": departmentID(t, h, departmentCode)}
		return h.Do(t, http.MethodPatch, "/api/v1/admin/users/"+request+"/verify", hod.Token, body)
	}

	if details := refusedWith(t, verify("EC"), http.StatusBadRequest); details["scope_id"] == "" {
		t.Errorf("other Department: details = %v, want scope_id named", details)
	}
	expectStatus(t, "own Department", verify("CS"), http.StatusOK)
}
