package integration

import (
	"fmt"
	"github.com/AbhishekBalija/Links/server/internal/auth"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

type firstSignIn struct {
	FullName       string   `json:"full_name"`
	Email          string   `json:"email"`
	USN            string   `json:"usn"`
	DepartmentCode string   `json:"department_code"`
	DepartmentName string   `json:"department_name"`
	BatchYear      int      `json:"batch_year"`
	Roles          []string `json:"roles"`
}

type signedIn struct {
	AccessToken string       `json:"access_token"`
	FirstSignIn *firstSignIn `json:"first_sign_in"`
}

// signInWithCode asks for a code and enters it, the way the screen does.
func signInWithCode(t *testing.T, h *apitest.Harness, email string) (signedIn, apitest.Response) {
	t.Helper()
	challenge := askForCode(t, h, email)
	response := h.Do(t, http.MethodPost, "/api/v1/auth/code/verify", "", map[string]string{
		"challenge_id": challenge, "email": email, "code": h.Outbox.LastCodeTo(t, email),
	})
	var reply struct {
		Data signedIn `json:"data"`
	}
	if response.Status == http.StatusOK {
		response.Decode(t, &reply)
	}
	return reply.Data, response
}

func userStatus(t *testing.T, h *apitest.Harness, email string) string {
	t.Helper()
	var status string
	h.DB().Raw(`SELECT status FROM users WHERE lower(email) = lower(?)`, email).Scan(&status)
	return status
}

func userIDByEmail(t *testing.T, h *apitest.Harness, email string) string {
	t.Helper()
	var id string
	h.DB().Raw(`SELECT id FROM users WHERE lower(email) = lower(?)`, email).Scan(&id)
	if id == "" {
		t.Fatalf("no user with email %s", email)
	}
	return id
}

func TestAnImportedStudentWaitsForFirstSignIn(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	imported(t, importCSV(h, admin.Token, "email,full_name,usn\nasha@gmail.com,Asha Rao,4MN23CS042\n"))

	if got := userStatus(t, h, "asha@gmail.com"); got != "pending" {
		t.Fatalf("status before first sign-in = %q, want pending", got)
	}

	session, response := signInWithCode(t, h, "asha@gmail.com")
	expectStatus(t, "first sign-in by code", response, http.StatusOK)
	first := session.FirstSignIn
	if first == nil {
		t.Fatalf("no first_sign_in in %s", response.Body)
	}
	if first.FullName != "Asha Rao" || first.USN != "4MN23CS042" || first.DepartmentCode != "CS" || first.DepartmentName == "" || first.BatchYear != 2023 || first.Email != "asha@gmail.com" {
		t.Errorf("first_sign_in = %+v, want Asha Rao, 4MN23CS042, CS, 2023", *first)
	}
	if got := userStatus(t, h, "asha@gmail.com"); got != "active" {
		t.Errorf("status after first sign-in = %q, want active", got)
	}
	expectStatus(t, "the new member's home", h.Do(t, http.MethodGet, "/api/v1/dashboard", session.AccessToken, nil), http.StatusOK)

	id := userIDByEmail(t, h, "asha@gmail.com")
	if got := auditCount(t, h, "auth.first_sign_in", id); got != 1 {
		t.Errorf("first sign-in audits = %d, want 1", got)
	}

	again, response := signInWithCode(t, h, "asha@gmail.com")
	expectStatus(t, "second sign-in", response, http.StatusOK)
	if again.FirstSignIn != nil {
		t.Errorf("a second sign-in still says first_sign_in: %+v", *again.FirstSignIn)
	}
}

func TestAnImportedStudentCanSignInWithGoogleFirst(t *testing.T) {
	h := apitest.New(t)
	hod := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	imported(t, importCSV(h, hod.Token, "email,full_name,usn\nravi@gmail.com,Ravi Kumar,4MN24CS007\n"))

	nonce, cookie := h.GoogleNonce(t)
	response := googleSignIn(t, h, h.GoogleToken(t, apitest.GoogleClaims{Subject: "google-ravi", Email: "Ravi@gmail.com", Nonce: nonce}), cookie)
	expectStatus(t, "first sign-in by Google", response, http.StatusOK)
	var reply struct {
		Data signedIn `json:"data"`
	}
	response.Decode(t, &reply)
	if reply.Data.FirstSignIn == nil || reply.Data.FirstSignIn.USN != "4MN24CS007" {
		t.Errorf("reply = %s, want first_sign_in for 4MN24CS007", response.Body)
	}
	if got := userStatus(t, h, "ravi@gmail.com"); got != "active" {
		t.Errorf("status = %q, want active", got)
	}
}

func TestAStaffInviteWaitsForFirstSignIn(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	cs := h.DepartmentID(t, "CS")

	response := h.Do(t, http.MethodPost, "/api/v1/admin/users", admin.Token, map[string]string{
		"email": "meera@college.example", "full_name": "Meera Iyer", "role": "faculty", "scope_type": "department", "scope_id": cs,
	})
	expectStatus(t, "invite faculty", response, http.StatusCreated)
	var created struct {
		Data struct {
			UserID string `json:"user_id"`
			Status string `json:"status"`
		} `json:"data"`
	}
	response.Decode(t, &created)
	if created.Data.UserID == "" || created.Data.Status != "pending" {
		t.Fatalf("invite reply = %s, want a pending user", response.Body)
	}
	if got := auditCount(t, h, "user_invited", created.Data.UserID); got != 1 {
		t.Errorf("invite audits = %d, want 1", got)
	}

	session, signIn := signInWithCode(t, h, "meera@college.example")
	expectStatus(t, "staff first sign-in", signIn, http.StatusOK)
	first := session.FirstSignIn
	if first == nil || first.FullName != "Meera Iyer" || first.DepartmentCode != "CS" || first.USN != "" || len(first.Roles) != 1 || first.Roles[0] != "faculty" {
		t.Errorf("first_sign_in = %s, want Meera Iyer, CS faculty, no USN", signIn.Body)
	}

	expectStatus(t, "the same email again", h.Do(t, http.MethodPost, "/api/v1/admin/users", admin.Token, map[string]string{
		"email": "Meera@college.example", "full_name": "Meera Iyer", "role": "faculty", "scope_type": "department", "scope_id": cs,
	}), http.StatusConflict)
}

// Admins add staff anywhere; an HOD adds faculty to their own Department;
// the principal no longer adds staff (ADR 0029).
func TestWhoCanInviteStaff(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	hod := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	cs, ec := h.DepartmentID(t, "CS"), h.DepartmentID(t, "EC")
	invite := func(token, email, role, scopeType, scopeID string) apitest.Response {
		return h.Do(t, http.MethodPost, "/api/v1/admin/users", token, map[string]string{
			"email": email, "full_name": "New Staff", "role": role, "scope_type": scopeType, "scope_id": scopeID,
		})
	}

	expectStatus(t, "admin invites a placement officer", invite(admin.Token, "po@college.example", "placement_officer", "global", ""), http.StatusCreated)
	expectStatus(t, "admin invites an admin", invite(admin.Token, "admin2@college.example", "admin", "global", ""), http.StatusCreated)
	expectStatus(t, "HOD invites CS faculty", invite(hod.Token, "f@college.example", "faculty", "department", cs), http.StatusCreated)

	expectStatus(t, "principal invites a placement officer", invite(principal.Token, "po2@college.example", "placement_officer", "global", ""), http.StatusForbidden)
	expectStatus(t, "principal invites CS faculty", invite(principal.Token, "f2@college.example", "faculty", "department", cs), http.StatusForbidden)
	expectStatus(t, "HOD invites EC faculty", invite(hod.Token, "f3@college.example", "faculty", "department", ec), http.StatusForbidden)
	expectStatus(t, "HOD invites a placement officer", invite(hod.Token, "po3@college.example", "placement_officer", "global", ""), http.StatusForbidden)
	expectStatus(t, "HOD invites another CS HOD", invite(hod.Token, "hod2@college.example", "hod", "department", cs), http.StatusForbidden)
	expectStatus(t, "faculty invites faculty", invite(faculty.Token, "f4@college.example", "faculty", "department", cs), http.StatusForbidden)
	expectStatus(t, "a second CS HOD", invite(admin.Token, "hod3@college.example", "hod", "department", cs), http.StatusConflict)
	expectStatus(t, "not an email", invite(admin.Token, "nope", "faculty", "department", cs), http.StatusBadRequest)

	var leftovers int
	h.DB().Raw(`SELECT count(*) FROM users WHERE email IN
		('po2@college.example', 'f2@college.example', 'f3@college.example', 'po3@college.example',
		 'hod2@college.example', 'f4@college.example', 'hod3@college.example')`).Scan(&leftovers)
	if leftovers != 0 {
		t.Errorf("%d refused invites left an account behind", leftovers)
	}
}

func TestNotYouOnFirstSignInSignsOutAndReturnsTheRowForReview(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	imported(t, importCSV(h, admin.Token, "email,full_name,usn\nasha@gmail.com,Wrong Name,4MN23CS042\n"))

	nonce, cookie := h.GoogleNonce(t)
	response := googleSignIn(t, h, h.GoogleToken(t, apitest.GoogleClaims{Subject: "google-asha", Email: "asha@gmail.com", Nonce: nonce}), cookie)
	expectStatus(t, "first sign-in", response, http.StatusOK)
	var reply struct {
		Data signedIn `json:"data"`
	}
	response.Decode(t, &reply)
	var refresh *http.Cookie
	for _, set := range response.Cookies() {
		if set.Name == "refresh_token" {
			refresh = set
		}
	}

	notMe := h.Do(t, http.MethodPost, "/api/v1/auth/not-me", reply.Data.AccessToken, nil)
	expectStatus(t, "not you", notMe, http.StatusOK)

	if got := userStatus(t, h, "asha@gmail.com"); got != "pending" {
		t.Errorf("status = %q, want pending (back for review)", got)
	}
	refreshRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	refreshRequest.AddCookie(refresh)
	expectStatus(t, "refresh after not you", h.Send(refreshRequest), http.StatusUnauthorized)
	id := userIDByEmail(t, h, "asha@gmail.com")
	if got := auditCount(t, h, "auth.not_me", id); got != 1 {
		t.Errorf("not-me audits = %d, want 1", got)
	}
	var subject *string
	h.DB().Raw(`SELECT google_subject FROM users WHERE id = ?`, id).Scan(&subject)
	if subject != nil {
		t.Errorf("Google account still linked (%q); the row must be fixed before anyone is linked to it", *subject)
	}
}

func TestARowReportedWithNotYouCantBeSignedIntoUntilSomeoneReviewsIt(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	imported(t, importCSV(h, admin.Token, "email,full_name,usn\nasha@gmail.com,Wrong Name,4MN23CS042\n"))
	session, response := signInWithCode(t, h, "asha@gmail.com")
	expectStatus(t, "first sign-in", response, http.StatusOK)
	expectStatus(t, "not you", h.Do(t, http.MethodPost, "/api/v1/auth/not-me", session.AccessToken, nil), http.StatusOK)

	// The same person signing in again must not land in the wrong row.
	_, again := signInWithCode(t, h, "asha@gmail.com")
	expectStatus(t, "signing in again", again, http.StatusForbidden)
	nonce, cookie := h.GoogleNonce(t)
	google := googleSignIn(t, h, h.GoogleToken(t, apitest.GoogleClaims{Subject: "google-asha", Email: "asha@gmail.com", Nonce: nonce}), cookie)
	expectStatus(t, "signing in again with Google", google, http.StatusForbidden)

	// The row waits in the review queue for someone to look at it.
	if emails := reviewQueueEmails(t, h, admin.Token); !slices.Contains(emails, "asha@gmail.com") {
		t.Errorf("review queue = %v, want the reported row in it", emails)
	}
}

func TestNotYouIsOnlyForAFirstSignIn(t *testing.T) {
	h := apitest.New(t)
	member := studentOf(t, h, "CS", 2023)
	expectStatus(t, "a long-standing member", h.Do(t, http.MethodPost, "/api/v1/auth/not-me", member.Token, nil), http.StatusConflict)

	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	imported(t, importCSV(h, admin.Token, "email,full_name,usn\nasha@gmail.com,Asha Rao,4MN23CS042\n"))
	session, response := signInWithCode(t, h, "asha@gmail.com")
	expectStatus(t, "first sign-in", response, http.StatusOK)
	// A day later the moment has passed.
	h.DB().Exec(`UPDATE users SET first_signed_in_at = now() - interval '1 day' WHERE lower(email) = 'asha@gmail.com'`)
	expectStatus(t, "not you a day later", h.Do(t, http.MethodPost, "/api/v1/auth/not-me", session.AccessToken, nil), http.StatusConflict)
}

// notOnList reads a NOT_ON_LIST reply.
func notOnList(t *testing.T, response apitest.Response) (email, name, requestToken string) {
	t.Helper()
	expectStatus(t, "not on the list", response, http.StatusForbidden)
	var reply struct {
		Error struct {
			Code    string `json:"code"`
			Details struct {
				Email        string `json:"email"`
				FullName     string `json:"full_name"`
				RequestToken string `json:"request_token"`
			} `json:"details"`
		} `json:"error"`
	}
	response.Decode(t, &reply)
	if reply.Error.Code != "NOT_ON_LIST" || reply.Error.Details.RequestToken == "" {
		t.Fatalf("reply = %s, want NOT_ON_LIST with a request_token", response.Body)
	}
	return reply.Error.Details.Email, reply.Error.Details.FullName, reply.Error.Details.RequestToken
}

func sendAccessRequest(t *testing.T, h *apitest.Harness, requestToken, usn, fullName string) apitest.Response {
	t.Helper()
	return h.Do(t, http.MethodPost, "/api/v1/auth/access-request", "", map[string]string{
		"request_token": requestToken, "usn": usn, "full_name": fullName,
	})
}

func reviewQueueEmails(t *testing.T, h *apitest.Harness, token string) []string {
	t.Helper()
	response := h.Do(t, http.MethodGet, "/api/v1/admin/users/review-queue", token, nil)
	expectStatus(t, "review queue", response, http.StatusOK)
	var queue struct {
		Data struct {
			Users []struct {
				Email string `json:"email"`
			} `json:"users"`
		} `json:"data"`
	}
	response.Decode(t, &queue)
	var emails []string
	for _, user := range queue.Data.Users {
		emails = append(emails, user.Email)
	}
	return emails
}

func TestSomeoneNotOnTheListRequestsAccessWithAnEmailCode(t *testing.T) {
	h := apitest.New(t)
	csHOD := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	ecHOD := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "EC"}}})

	_, response := signInWithCode(t, h, "kiran@gmail.com")
	email, _, requestToken := notOnList(t, response)
	if email != "kiran@gmail.com" {
		t.Errorf("email = %q, want the proven kiran@gmail.com", email)
	}

	request := sendAccessRequest(t, h, requestToken, "4mn23cs077", "Kiran S")
	expectStatus(t, "send the request", request, http.StatusCreated)
	if !contains(reviewQueueEmails(t, h, csHOD.Token), "kiran@gmail.com") {
		t.Error("the request isn't in the CS HOD's queue")
	}
	if contains(reviewQueueEmails(t, h, ecHOD.Token), "kiran@gmail.com") {
		t.Error("the request reached the EC HOD's queue")
	}

	// Waiting for the HOD, a sign-in says so.
	_, waiting := signInWithCode(t, h, "kiran@gmail.com")
	expectStatus(t, "sign-in while waiting", waiting, http.StatusForbidden)

	id := userIDByEmail(t, h, "kiran@gmail.com")
	expectStatus(t, "HOD approves", h.Do(t, http.MethodPatch, "/api/v1/admin/users/"+id+"/verify", csHOD.Token, map[string]string{}), http.StatusOK)

	session, approved := signInWithCode(t, h, "kiran@gmail.com")
	expectStatus(t, "sign-in once approved", approved, http.StatusOK)
	if session.FirstSignIn == nil || session.FirstSignIn.USN != "4MN23CS077" || session.FirstSignIn.FullName != "Kiran S" {
		t.Errorf("first_sign_in = %s, want Kiran S, 4MN23CS077", approved.Body)
	}
}

func TestSomeoneNotOnTheListRequestsAccessWithGoogle(t *testing.T) {
	h := apitest.New(t)
	hod := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "EC"}}})

	nonce, cookie := h.GoogleNonce(t)
	email, name, requestToken := notOnList(t, googleSignIn(t, h, h.GoogleToken(t, apitest.GoogleClaims{Subject: "google-n", Email: "Nisha@gmail.com", Name: "Nisha P", Nonce: nonce}), cookie))
	if email != "nisha@gmail.com" || name != "Nisha P" {
		t.Errorf("prefill = %q, %q", email, name)
	}
	expectStatus(t, "send the request", sendAccessRequest(t, h, requestToken, "4MN22EC015", name), http.StatusCreated)
	if !contains(reviewQueueEmails(t, h, hod.Token), "nisha@gmail.com") {
		t.Error("the request isn't in the EC HOD's queue")
	}
	expectStatus(t, "the same request twice", sendAccessRequest(t, h, requestToken, "4MN22EC015", name), http.StatusConflict)
}

func TestAnAccessRequestNeedsAProvenEmailAndAValidUSN(t *testing.T) {
	h := apitest.New(t)
	_, response := signInWithCode(t, h, "kiran@gmail.com")
	_, _, requestToken := notOnList(t, response)

	expectStatus(t, "a made-up token", sendAccessRequest(t, h, "made.up.token", "4MN23CS077", "Kiran S"), http.StatusUnauthorized)
	expectStatus(t, "a session token as proof", sendAccessRequest(t, h, studentOf(t, h, "CS", 2023).Token, "4MN23CS078", "Kiran S"), http.StatusUnauthorized)
	expectStatus(t, "a malformed USN", sendAccessRequest(t, h, requestToken, "CS077", "Kiran S"), http.StatusBadRequest)
	expectStatus(t, "a Department that doesn't exist", sendAccessRequest(t, h, requestToken, "4MN23ZZ077", "Kiran S"), http.StatusBadRequest)
	expectStatus(t, "no name", sendAccessRequest(t, h, requestToken, "4MN23CS077", ""), http.StatusBadRequest)

	taken := studentOf(t, h, "CS", 2023)
	var usn string
	h.DB().Raw(`SELECT usn FROM student_identities WHERE user_id = ?`, taken.ID).Scan(&usn)
	expectStatus(t, "a USN already registered", sendAccessRequest(t, h, requestToken, usn, "Kiran S"), http.StatusConflict)

	// Half an hour later the proof has expired.
	expired := h.AccessRequestToken(t, "late@gmail.com", time.Now().Add(-time.Minute))
	expectStatus(t, "an expired proof", sendAccessRequest(t, h, expired, "4MN23CS079", "Late"), http.StatusUnauthorized)
}

func TestTheCodeAndGoogleTellAWaitingRequestApart(t *testing.T) {
	h := apitest.New(t)
	for i, status := range []string{"pending", "rejected"} {
		member := studentOf(t, h, "CS", 2023)
		h.DB().Exec(`UPDATE users SET status = ?, is_verified = false WHERE id = ?`, status, member.ID)
		nonce, cookie := h.GoogleNonce(t)
		expectStatus(t, status, googleSignIn(t, h, h.GoogleToken(t, apitest.GoogleClaims{Subject: fmt.Sprint("g", i), Email: member.Email, Nonce: nonce}), cookie), http.StatusForbidden)
	}
}

func TestCodesToEmailsOnNoListAreCappedForTheWholeSiteEachDay(t *testing.T) {
	h := apitest.New(t)
	member := studentOf(t, h, "CS", 2023)
	limit := auth.DefaultCodeSettings().NotOnListDailyLimit

	for i := range limit {
		askForCode(t, h, fmt.Sprintf("stranger%d@gmail.com", i))
	}
	// One more stranger gets the usual reply but no code.
	askForCode(t, h, "one-too-many@gmail.com")
	if codes := h.Outbox.CodesTo("one-too-many@gmail.com"); len(codes) != 0 {
		t.Errorf("sent %d codes past the daily cap for emails on no list", len(codes))
	}
	// Members are never caught by it.
	askForCode(t, h, member.Email)
	if codes := h.Outbox.CodesTo(member.Email); len(codes) != 1 {
		t.Errorf("sent the member %d codes, want 1", len(codes))
	}
}
