package integration

import (
	"net/http"
	"testing"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

// What a reviewer needs to see about each Access request (#124): its
// Department by name, whether that Department has an HOD (without one the
// request lands with admins), and whether the row was reported with "Not
// you?" on a first sign-in.

type requestContext struct {
	ID              string  `json:"id"`
	Email           string  `json:"email"`
	ReportedAt      *string `json:"reported_at"`
	StudentIdentity struct {
		USN              string `json:"usn"`
		DepartmentCode   string `json:"department_code"`
		DepartmentName   string `json:"department_name"`
		DepartmentHasHOD bool   `json:"department_has_hod"`
	} `json:"student_identity"`
}

func requestContexts(t *testing.T, h *apitest.Harness, token string) map[string]requestContext {
	t.Helper()
	response := h.Do(t, http.MethodGet, "/api/v1/admin/users/review-queue", token, nil)
	expectStatus(t, "review queue", response, http.StatusOK)
	var page struct {
		Data struct {
			Users []requestContext `json:"users"`
		} `json:"data"`
	}
	response.Decode(t, &page)
	byEmail := map[string]requestContext{}
	for _, item := range page.Data.Users {
		byEmail[item.Email] = item
	}
	return byEmail
}

func TestEachAccessRequestNamesItsDepartmentAndWhetherItHasAnHOD(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	signUp(t, h, "4MN23CS901")
	signUp(t, h, "4MN23EC902")

	queue := requestContexts(t, h, admin.Token)
	cs := queue["request-4mn23cs901@apitest.local"].StudentIdentity
	if cs.DepartmentName != "Computer Science and Engineering" || !cs.DepartmentHasHOD {
		t.Errorf("CS request = %+v, want its name and an HOD", cs)
	}
	ec := queue["request-4mn23ec902@apitest.local"].StudentIdentity
	if ec.DepartmentName == "" || ec.DepartmentHasHOD {
		t.Errorf("EC request = %+v, want its name and no HOD", ec)
	}
}

func TestARowReportedWithNotYouSaysSoInTheQueue(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	imported(t, importCSV(h, admin.Token, "email,full_name,usn\nasha@gmail.com,Wrong Name,4MN23CS042\n"))
	session, response := signInWithCode(t, h, "asha@gmail.com")
	expectStatus(t, "first sign-in", response, http.StatusOK)
	expectStatus(t, "not you", h.Do(t, http.MethodPost, "/api/v1/auth/not-me", session.AccessToken, nil), http.StatusOK)
	signUp(t, h, "4MN23CS903")

	queue := requestContexts(t, h, admin.Token)
	if queue["asha@gmail.com"].ReportedAt == nil {
		t.Error("the reported row has no reported_at")
	}
	if queue["request-4mn23cs903@apitest.local"].ReportedAt != nil {
		t.Error("an ordinary request has a reported_at")
	}
}

func TestApprovingAReportedRowLetsThemSignInWithOneStudentRole(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	imported(t, importCSV(h, admin.Token, "email,full_name,usn\nasha@gmail.com,Asha Rao,4MN23CS042\n"))
	session, response := signInWithCode(t, h, "asha@gmail.com")
	expectStatus(t, "first sign-in", response, http.StatusOK)
	expectStatus(t, "not you", h.Do(t, http.MethodPost, "/api/v1/auth/not-me", session.AccessToken, nil), http.StatusOK)

	// The admin fixes the row's details elsewhere, then approves it again.
	id := userIDByEmail(t, h, "asha@gmail.com")
	expectStatus(t, "approve", h.Do(t, http.MethodPatch, "/api/v1/admin/users/"+id+"/verify", admin.Token, map[string]string{}), http.StatusOK)

	var students int
	h.DB().Raw(`SELECT count(*) FROM role_assignments WHERE user_id = ? AND role = 'student' AND (ends_at IS NULL OR ends_at > now())`, id).Scan(&students)
	if students != 1 {
		t.Errorf("student roles in effect = %d, want 1", students)
	}
	_, again := signInWithCode(t, h, "asha@gmail.com")
	expectStatus(t, "sign in once approved again", again, http.StatusOK)
}
