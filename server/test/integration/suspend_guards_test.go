package integration

import (
	"net/http"
	"testing"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

// Guards on suspending (#177): nobody suspends themselves, only an admin
// suspends an admin or the principal, and the last active admin stays.

func setStatus(h *apitest.Harness, t *testing.T, token, userID, status string) apitest.Response {
	t.Helper()
	return h.Do(t, http.MethodPatch, "/api/v1/admin/users/"+userID+"/status", token, map[string]string{"status": status, "note": "test"})
}

func TestNobodySuspendsThemselves(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})

	expectStatus(t, "admin suspends self", setStatus(h, t, admin.Token, admin.ID, "suspended"), http.StatusForbidden)
	expectStatus(t, "principal suspends self", setStatus(h, t, principal.Token, principal.ID, "suspended"), http.StatusForbidden)
}

func TestOnlyAnAdminSuspendsAnAdminOrThePrincipal(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	otherAdmin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})

	expectStatus(t, "principal suspends an admin", setStatus(h, t, principal.Token, otherAdmin.ID, "suspended"), http.StatusForbidden)
	expectStatus(t, "principal suspends faculty", setStatus(h, t, principal.Token, faculty.ID, "suspended"), http.StatusOK)
	expectStatus(t, "admin suspends the principal", setStatus(h, t, admin.Token, principal.ID, "suspended"), http.StatusOK)
	expectStatus(t, "admin suspends another admin", setStatus(h, t, admin.Token, otherAdmin.ID, "suspended"), http.StatusOK)
}

func TestTheLastActiveAdminCantBeSuspended(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	second := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})

	expectStatus(t, "suspend the second admin", setStatus(h, t, admin.Token, second.ID, "suspended"), http.StatusOK)
	// The suspended admin's old token tries to suspend the only active one.
	// Refused either as the last active admin (409) or as a suspended
	// account (401, #175), never allowed.
	if response := setStatus(h, t, second.Token, admin.ID, "suspended"); response.Status == http.StatusOK {
		t.Fatalf("the last active admin was suspended: %s", response.Body)
	}
}
