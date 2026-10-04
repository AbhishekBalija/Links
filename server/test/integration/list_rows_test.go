package integration

import (
	"net/http"
	"slices"
	"testing"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

// Fixing a wrong class-list row (#174): an admin, or the HOD of its
// department, fixes the email of a row nobody has signed into (or one
// reported with "Not you?"), or removes the row so its USN can be added again.

func fixEmail(t *testing.T, h *apitest.Harness, token, userID, email string) apitest.Response {
	t.Helper()
	return h.Do(t, http.MethodPatch, "/api/v1/admin/users/"+userID+"/email", token, map[string]string{"email": email})
}

func removeRow(t *testing.T, h *apitest.Harness, token, userID string) apitest.Response {
	t.Helper()
	return h.Do(t, http.MethodDelete, "/api/v1/admin/users/"+userID, token, nil)
}

func TestAnAdminFixesTheEmailOfARowNobodySignedInto(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	imported(t, importCSV(h, admin.Token, "email,full_name,usn\nkavya.rao@gmial.com,Kavya Rao,4MN25CS001\n"))
	row := userIDByEmail(t, h, "kavya.rao@gmial.com")

	expectStatus(t, "fix the email", fixEmail(t, h, admin.Token, row, "kavya.rao@gmail.com"), http.StatusOK)
	if got := auditCount(t, h, "list_row_email_fixed", row); got != 1 {
		t.Errorf("email fix audits = %d, want 1", got)
	}

	session, response := signInWithCode(t, h, "kavya.rao@gmail.com")
	expectStatus(t, "sign in with the fixed email", response, http.StatusOK)
	if session.FirstSignIn == nil || session.FirstSignIn.USN != "4MN25CS001" {
		t.Errorf("first sign-in = %s, want the 4MN25CS001 row", response.Body)
	}
}

func TestFixingAnEmailIsRefusedWhenItBelongsToSomeoneOrTheyAreIn(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	imported(t, importCSV(h, admin.Token, "email,full_name,usn\nkavya@gmail.com,Kavya Rao,4MN25CS001\narjun@gmail.com,Arjun Nayak,4MN25CS002\n"))
	kavya := userIDByEmail(t, h, "kavya@gmail.com")

	if details := refusedWith(t, fixEmail(t, h, admin.Token, kavya, "arjun@gmail.com"), http.StatusConflict); details["email"] == "" {
		t.Errorf("someone else's email: details = %v, want the reason it's taken", details)
	}
	expectStatus(t, "not an email", fixEmail(t, h, admin.Token, kavya, "kavya@"), http.StatusBadRequest)

	_, response := signInWithCode(t, h, "kavya@gmail.com")
	expectStatus(t, "Kavya signs in", response, http.StatusOK)
	expectStatus(t, "fix after they signed in", fixEmail(t, h, admin.Token, kavya, "kavya.rao@gmail.com"), http.StatusConflict)
}

func TestFixingAReportedRowsEmailSendsItBackToWaitForFirstSignIn(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	imported(t, importCSV(h, admin.Token, "email,full_name,usn\nrohan.s@outlook.com,Rohan Shetty,4MN25CS017\n"))
	session, response := signInWithCode(t, h, "rohan.s@outlook.com")
	expectStatus(t, "the wrong person signs in", response, http.StatusOK)
	expectStatus(t, "not you", h.Do(t, http.MethodPost, "/api/v1/auth/not-me", session.AccessToken, nil), http.StatusOK)
	row := userIDByEmail(t, h, "rohan.s@outlook.com")

	expectStatus(t, "fix the email", fixEmail(t, h, admin.Token, row, "rohan.shetty25@gmail.com"), http.StatusOK)
	if emails := reviewQueueEmails(t, h, admin.Token); slices.Contains(emails, "rohan.shetty25@gmail.com") {
		t.Errorf("review queue = %v; a fixed row waits for its first sign-in instead", emails)
	}
	rohan, again := signInWithCode(t, h, "rohan.shetty25@gmail.com")
	expectStatus(t, "Rohan signs in", again, http.StatusOK)
	if rohan.FirstSignIn == nil {
		t.Error("Rohan skipped the first sign-in check")
	}
}

func TestRemovingARowFreesItsUSN(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	imported(t, importCSV(h, admin.Token, "email,full_name,usn\nwrong@gmail.com,Kavya Rao,4MN25CS001\nkept@gmail.com,Arjun Nayak,4MN25CS002\n"))
	row := userIDByEmail(t, h, "wrong@gmail.com")

	expectStatus(t, "remove", removeRow(t, h, admin.Token, row), http.StatusOK)
	if accounts(t, h, row) != 0 {
		t.Fatal("the row is still there")
	}
	if got := auditCount(t, h, "list_row_removed", row); got != 1 {
		t.Errorf("removal audits = %d, want 1", got)
	}
	imported(t, importCSV(h, admin.Token, "email,full_name,usn\nkavya.rao@gmail.com,Kavya Rao,4MN25CS001\n"))

	kept := userIDByEmail(t, h, "kept@gmail.com")
	_, response := signInWithCode(t, h, "kept@gmail.com")
	expectStatus(t, "Arjun signs in", response, http.StatusOK)
	expectStatus(t, "remove someone who signed in", removeRow(t, h, admin.Token, kept), http.StatusConflict)
}

func TestAnHODFixesOnlyTheirOwnDepartmentsRows(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	hod := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	imported(t, importCSV(h, admin.Token, "email,full_name,usn\ncs@gmail.com,CS Student,4MN25CS001\nec@gmail.com,EC Student,4MN25EC001\n"))
	cs, ec := userIDByEmail(t, h, "cs@gmail.com"), userIDByEmail(t, h, "ec@gmail.com")

	expectStatus(t, "HOD fixes a CS row", fixEmail(t, h, hod.Token, cs, "cs.student@gmail.com"), http.StatusOK)
	expectStatus(t, "HOD fixes an EC row", fixEmail(t, h, hod.Token, ec, "ec.student@gmail.com"), http.StatusNotFound)
	expectStatus(t, "HOD removes an EC row", removeRow(t, h, hod.Token, ec), http.StatusNotFound)
	expectStatus(t, "the principal removes a row", removeRow(t, h, principal.Token, ec), http.StatusForbidden)
	expectStatus(t, "HOD removes a CS row", removeRow(t, h, hod.Token, cs), http.StatusOK)
}
