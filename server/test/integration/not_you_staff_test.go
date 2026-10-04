package integration

import (
	"net/http"
	"testing"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

// "Not you?" on staff accounts (#201): the principal and admins can't report
// themselves, which could lock a college out, and a reported staff row is
// approved back to its staff role, never made a student.

func TestThePrincipalAndAdminCantReportNotYou(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	for _, invite := range []struct{ email, role string }{{"principal@college.edu", "principal"}, {"office@college.edu", "admin"}} {
		t.Run(invite.role, func(t *testing.T) {
			addStaff(t, h, admin.Token, map[string]string{"email": invite.email, "full_name": "Someone", "role": invite.role, "scope_type": "global"})
			nonce, cookie := h.GoogleNonce(t)
			response := googleSignIn(t, h, h.GoogleToken(t, apitest.GoogleClaims{Subject: "google-" + invite.role, Email: invite.email, Nonce: nonce}), cookie)
			expectStatus(t, "first sign-in", response, http.StatusOK)
			var reply struct {
				Data signedIn `json:"data"`
			}
			response.Decode(t, &reply)

			expectStatus(t, "not you", h.Do(t, http.MethodPost, "/api/v1/auth/not-me", reply.Data.AccessToken, nil), http.StatusConflict)
			if got := userStatus(t, h, invite.email); got != "active" {
				t.Errorf("status = %q, want active: nobody may lock the college out by mistake", got)
			}
		})
	}
}

func TestApprovingAReportedStaffRowKeepsTheirStaffRole(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	addStaff(t, h, admin.Token, map[string]string{
		"email": "meera@college.edu", "full_name": "Meera Iyer", "role": "faculty", "scope_type": "department", "scope_id": h.DepartmentID(t, "CS"),
	})
	session, response := signInWithCode(t, h, "meera@college.edu")
	expectStatus(t, "first sign-in", response, http.StatusOK)
	expectStatus(t, "not you", h.Do(t, http.MethodPost, "/api/v1/auth/not-me", session.AccessToken, nil), http.StatusOK)

	id := userIDByEmail(t, h, "meera@college.edu")
	expectStatus(t, "approve", h.Do(t, http.MethodPatch, "/api/v1/admin/users/"+id+"/verify", admin.Token, nil), http.StatusOK)

	var roles []string
	h.DB().Raw(`SELECT role FROM role_assignments WHERE user_id = ? ORDER BY role`, id).Scan(&roles)
	if len(roles) != 1 || roles[0] != "faculty" {
		t.Errorf("roles = %v, want only faculty", roles)
	}
}
