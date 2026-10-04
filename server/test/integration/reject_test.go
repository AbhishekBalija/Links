package integration

import (
	"net/http"
	"testing"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

// Rejecting (#203): rejecting an Access request removes it, so the email and
// USN can be used again; only a request can be rejected, and reactivating a
// suspended account never skips approval or a first sign-in.

func accounts(t *testing.T, h *apitest.Harness, userID string) int {
	t.Helper()
	var count int
	h.DB().Raw(`SELECT count(*) FROM users WHERE id = ?`, userID).Scan(&count)
	return count
}

func TestRejectingARequestFreesTheEmailAndUSN(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	request := signUp(t, h, "4MN23CS901")
	var email string
	h.DB().Raw(`SELECT email FROM users WHERE id = ?`, request).Scan(&email)

	expectStatus(t, "reject", setStatus(h, t, admin.Token, request, "rejected"), http.StatusOK)
	if accounts(t, h, request) != 0 {
		t.Fatal("the rejected request is still there")
	}
	if got := auditCount(t, h, "access_request_rejected", request); got != 1 {
		t.Errorf("rejection audits = %d, want 1", got)
	}

	// A corrected class list can now add the same person.
	imported(t, importCSV(h, admin.Token, "email,full_name,usn\n"+email+",Asha Rao,4MN23CS901\n"))
}

func TestRejectingAReportedRowFreesItsUSN(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	imported(t, importCSV(h, admin.Token, "email,full_name,usn\nwrong@gmail.com,Asha Rao,4MN23CS902\n"))
	session, response := signInWithCode(t, h, "wrong@gmail.com")
	expectStatus(t, "first sign-in", response, http.StatusOK)
	expectStatus(t, "not you", h.Do(t, http.MethodPost, "/api/v1/auth/not-me", session.AccessToken, nil), http.StatusOK)

	row := userIDByEmail(t, h, "wrong@gmail.com")
	expectStatus(t, "reject the reported row", setStatus(h, t, admin.Token, row, "rejected"), http.StatusOK)
	if accounts(t, h, row) != 0 {
		t.Fatal("the reported row is still there")
	}
	imported(t, importCSV(h, admin.Token, "email,full_name,usn\nasha.rao@gmail.com,Asha Rao,4MN23CS902\n"))
}

func TestOnlyARequestCanBeRejected(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	member := student(t, h, "CS", 2023)

	expectStatus(t, "reject a member", setStatus(h, t, admin.Token, member.ID, "rejected"), http.StatusConflict)
	var status string
	h.DB().Raw(`SELECT status FROM users WHERE id = ?`, member.ID).Scan(&status)
	if status != "active" {
		t.Errorf("status = %q, want active", status)
	}
}

func TestReactivatingNeverSkipsApprovalOrAFirstSignIn(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})

	request := signUp(t, h, "4MN23CS903")
	expectStatus(t, "suspend a request", setStatus(h, t, admin.Token, request, "suspended"), http.StatusOK)
	expectStatus(t, "reactivate a request never approved", setStatus(h, t, admin.Token, request, "active"), http.StatusConflict)

	addStaff(t, h, admin.Token, map[string]string{
		"email": "meera@college.edu", "full_name": "Meera Iyer", "role": "faculty", "scope_type": "department", "scope_id": h.DepartmentID(t, "CS"),
	})
	invited := userIDByEmail(t, h, "meera@college.edu")
	expectStatus(t, "suspend an invite", setStatus(h, t, admin.Token, invited, "suspended"), http.StatusOK)
	expectStatus(t, "reactivate the invite", setStatus(h, t, admin.Token, invited, "active"), http.StatusOK)
	if got := userStatus(t, h, "meera@college.edu"); got != "pending" {
		t.Errorf("status = %q, want pending: still waiting for its first sign-in", got)
	}
	session, response := signInWithCode(t, h, "meera@college.edu")
	expectStatus(t, "first sign-in after reactivating", response, http.StatusOK)
	if session.FirstSignIn == nil {
		t.Error("signing in skipped the first sign-in check")
	}
}
