package integration

import (
	"net/http"
	"testing"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

// A profile is readable only for listed members (active, verified, a role in
// effect), like the directory; admins and the principal still see anyone's
// to manage them, and the owner always sees their own (#176).

func usernameByEmail(t *testing.T, h *apitest.Harness, email string) string {
	t.Helper()
	var username string
	if err := h.DB().Raw(`SELECT p.username FROM profiles p JOIN users u ON u.id = p.user_id WHERE u.email = ?`, email).Scan(&username).Error; err != nil || username == "" {
		t.Fatalf("username of %s: %v", email, err)
	}
	return username
}

func profileStatus(t *testing.T, h *apitest.Harness, token, username string) int {
	t.Helper()
	return h.Do(t, http.MethodGet, "/api/v1/profiles/"+username, token, nil).Status
}

func TestAnImportedStudentsProfileIsHiddenUntilTheySignIn(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	classmate := student(t, h, "CS", 2025)
	imported(t, importCSV(h, admin.Token, "email,full_name,usn\nkavya@gmail.com,Kavya Rao,4MN25CS301\n"))
	kavya := usernameByEmail(t, h, "kavya@gmail.com")

	if got := profileStatus(t, h, "", kavya); got != http.StatusNotFound {
		t.Errorf("anonymous: status = %d, want %d", got, http.StatusNotFound)
	}
	if got := profileStatus(t, h, classmate.Token, kavya); got != http.StatusNotFound {
		t.Errorf("a classmate: status = %d, want %d", got, http.StatusNotFound)
	}
	for name, token := range map[string]string{"admin": admin.Token, "principal": principal.Token} {
		if got := profileStatus(t, h, token, kavya); got != http.StatusOK {
			t.Errorf("%s: status = %d, want %d", name, got, http.StatusOK)
		}
	}
}

func TestASuspendedMembersProfileIsHidden(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	reader := student(t, h, "CS", 2023)
	other := student(t, h, "CS", 2023)
	name := usernameOf(t, h, other.ID)

	if got := profileStatus(t, h, "", name); got != http.StatusOK {
		t.Fatalf("an active member's profile: status = %d, want %d", got, http.StatusOK)
	}
	expectStatus(t, "suspend", h.Do(t, http.MethodPatch, "/api/v1/admin/users/"+other.ID+"/status", admin.Token, map[string]string{"status": "suspended", "note": "misuse"}), http.StatusOK)
	if got := profileStatus(t, h, reader.Token, name); got != http.StatusNotFound {
		t.Errorf("a suspended member's profile: status = %d, want %d", got, http.StatusNotFound)
	}
	if got := profileStatus(t, h, admin.Token, name); got != http.StatusOK {
		t.Errorf("admin: status = %d, want %d", got, http.StatusOK)
	}
}
