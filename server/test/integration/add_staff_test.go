package integration

import (
	"net/http"
	"testing"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

// Adding staff (#189): the person is emailed how to sign in, and a refusal
// says which field is wrong and why, so the screen can say what to do.

func addStaff(t *testing.T, h *apitest.Harness, token string, body map[string]string) apitest.Response {
	t.Helper()
	return h.Do(t, http.MethodPost, "/api/v1/admin/users", token, body)
}

type refusal struct {
	Error struct {
		Code    string            `json:"code"`
		Details map[string]string `json:"details"`
	} `json:"error"`
}

func refusedWith(t *testing.T, response apitest.Response, status int) map[string]string {
	t.Helper()
	if response.Status != status {
		t.Fatalf("status = %d, want %d: %s", response.Status, status, response.Body)
	}
	var body refusal
	response.Decode(t, &body)
	return body.Error.Details
}

func TestAddingStaffEmailsThemHowToSignIn(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})

	added := addStaff(t, h, admin.Token, map[string]string{
		"email": "kiran.hegde@college.edu", "full_name": "Prof. Kiran Hegde",
		"role": "faculty", "scope_type": "department", "scope_id": h.DepartmentID(t, "CS"),
	})
	if added.Status != http.StatusCreated {
		t.Fatalf("add faculty status = %d: %s", added.Status, added.Body)
	}
	var body struct {
		Data struct {
			Emailed bool `json:"emailed"`
		} `json:"data"`
	}
	added.Decode(t, &body)
	if !body.Data.Emailed {
		t.Error("emailed = false, want true")
	}

	letter := h.Outbox.LastStaffAddedTo(t, "kiran.hegde@college.edu")
	if letter.FullName != "Prof. Kiran Hegde" || letter.Role != "Faculty, Computer Science and Engineering" || letter.GoogleOnly {
		t.Errorf("faculty letter = %+v", letter)
	}
	if letter.AddedBy == "" {
		t.Error("the letter doesn't say who added them")
	}

	addStaff(t, h, admin.Token, map[string]string{"email": "principal@college.edu", "full_name": "Dr. Shalini Rao", "role": "principal", "scope_type": "global"})
	if principal := h.Outbox.LastStaffAddedTo(t, "principal@college.edu"); principal.Role != "Principal" || !principal.GoogleOnly {
		t.Errorf("principal letter = %+v, want Principal, Google only", principal)
	}
}

func TestAddingStaffStillWorksWhenTheEmailCannotBeSent(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	h.Outbox.FailStaffAdded()

	added := addStaff(t, h, admin.Token, map[string]string{
		"email": "kiran.hegde@college.edu", "full_name": "Prof. Kiran Hegde",
		"role": "faculty", "scope_type": "department", "scope_id": h.DepartmentID(t, "CS"),
	})
	if added.Status != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", added.Status, http.StatusCreated, added.Body)
	}
	var body struct {
		Data struct {
			Emailed bool `json:"emailed"`
		} `json:"data"`
	}
	added.Decode(t, &body)
	if body.Data.Emailed {
		t.Error("emailed = true although sending failed")
	}
}

func TestAddingStaffSaysWhyAnEmailOrDepartmentIsRefused(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	cs := h.DepartmentID(t, "CS")
	faculty := func(email string) map[string]string {
		return map[string]string{"email": email, "full_name": "Someone", "role": "faculty", "scope_type": "department", "scope_id": cs}
	}

	current := student(t, h, "CS", 2023)
	if details := refusedWith(t, addStaff(t, h, admin.Token, faculty(current.Email)), http.StatusConflict); details["email"] != "student" {
		t.Errorf("a current student's email: details = %v, want email=student", details)
	}

	asking := student(t, h, "CS", 2024)
	if err := h.DB().Exec(`UPDATE users SET status = 'pending', is_verified = false WHERE id = ?`, asking.ID).Error; err != nil {
		t.Fatal(err)
	}
	if details := refusedWith(t, addStaff(t, h, admin.Token, faculty(asking.Email)), http.StatusConflict); details["email"] != "request" {
		t.Errorf("an email waiting as a student request: details = %v, want email=request", details)
	}

	member := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	details := refusedWith(t, addStaff(t, h, admin.Token, faculty(member.Email)), http.StatusConflict)
	if details["email"] != "member" || details["username"] == "" {
		t.Errorf("a member's email: details = %v, want email=member and their username", details)
	}

	hod := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	var hodName string
	if err := h.DB().Raw(`SELECT full_name FROM profiles WHERE user_id = ?`, hod.ID).Scan(&hodName).Error; err != nil {
		t.Fatal(err)
	}
	second := map[string]string{"email": "new.hod@college.edu", "full_name": "Someone", "role": "hod", "scope_type": "department", "scope_id": cs}
	details = refusedWith(t, addStaff(t, h, admin.Token, second), http.StatusConflict)
	if details["scope_id"] != "has_hod" || details["hod_name"] != hodName {
		t.Errorf("a second HOD: details = %v, want scope_id=has_hod and hod_name=%q", details, hodName)
	}
}
