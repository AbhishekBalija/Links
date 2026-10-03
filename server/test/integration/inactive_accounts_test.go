package integration

import (
	"net/http"
	"testing"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

// An account that is suspended or rejected can't use the API, even with an
// access token issued before (#175).

func TestASuspendedAccountIsRefusedAtOnce(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	reader := student(t, h, "CS", 2023)

	expectStatus(t, "before", h.Do(t, http.MethodGet, "/api/v1/me", reader.Token, nil), http.StatusOK)
	expectStatus(t, "suspend", h.Do(t, http.MethodPatch, "/api/v1/admin/users/"+reader.ID+"/status", admin.Token, map[string]string{"status": "suspended", "note": "misuse"}), http.StatusOK)

	for _, path := range []string{"/api/v1/me", "/api/v1/announcements", "/api/v1/events?view=upcoming"} {
		if response := h.Do(t, http.MethodGet, path, reader.Token, nil); response.Status != http.StatusUnauthorized {
			t.Errorf("%s with the old token: status = %d, want %d", path, response.Status, http.StatusUnauthorized)
		}
	}

	expectStatus(t, "reactivate", h.Do(t, http.MethodPatch, "/api/v1/admin/users/"+reader.ID+"/status", admin.Token, map[string]string{"status": "active", "note": "ok"}), http.StatusOK)
	expectStatus(t, "after reactivating", h.Do(t, http.MethodGet, "/api/v1/me", reader.Token, nil), http.StatusOK)
}
