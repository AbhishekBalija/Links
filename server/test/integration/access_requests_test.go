package integration

import (
	"net/http"
	"slices"
	"testing"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

type accessRequest struct {
	ID              string `json:"id"`
	Email           string `json:"email"`
	StudentIdentity *struct {
		USN            string `json:"usn"`
		DepartmentCode string `json:"department_code"`
		BatchYear      int    `json:"batch_year"`
	} `json:"student_identity"`
}

func accessQueue(t *testing.T, h *apitest.Harness, token string) []accessRequest {
	t.Helper()
	response := h.Do(t, http.MethodGet, "/api/v1/admin/users/review-queue", token, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("review queue status = %d: %s", response.Status, response.Body)
	}
	var page struct {
		Data struct {
			Users []accessRequest `json:"users"`
			Total int             `json:"total"`
		} `json:"data"`
	}
	response.Decode(t, &page)
	return page.Data.Users
}

func requestIDs(items []accessRequest) []string {
	ids := []string{}
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func TestTheAccessRequestQueueHoldsOnlyRequestsWaitingForApproval(t *testing.T) {
	h := apitest.New(t)
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	first := signUp(t, h, "4MN23CS801")
	second := signUp(t, h, "4MN24EC802")
	// An approved student is pending too until they activate, but they no
	// longer wait for anyone.
	approved := signUp(t, h, "4MN23CS803")
	expectStatus(t, "approve", h.Do(t, http.MethodPatch, "/api/v1/admin/users/"+approved+"/verify", principal.Token, nil), http.StatusOK)

	queue := accessQueue(t, h, principal.Token)
	if got := requestIDs(queue); !slices.Equal(got, []string{first, second}) {
		t.Fatalf("queue = %v, want the two unapproved requests, oldest first", got)
	}
	if queue[0].StudentIdentity == nil || queue[0].StudentIdentity.DepartmentCode != "CS" || queue[0].StudentIdentity.BatchYear != 2023 {
		t.Errorf("first request = %+v, want CS, batch 2023", queue[0].StudentIdentity)
	}
}

func TestHODsHandleAccessRequestsForTheirOwnDepartment(t *testing.T) {
	h := apitest.New(t)
	hod := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	csFirst := signUp(t, h, "4MN23CS811")
	csSecond := signUp(t, h, "4MN23CS812")
	ec := signUp(t, h, "4MN24EC813")

	if got := requestIDs(accessQueue(t, h, hod.Token)); !slices.Equal(got, []string{csFirst, csSecond}) {
		t.Fatalf("CS HOD queue = %v, want only the CS requests", got)
	}
	if got := requestIDs(accessQueue(t, h, principal.Token)); !slices.Equal(got, []string{csFirst, csSecond, ec}) {
		t.Fatalf("principal queue = %v, want every request", got)
	}
	expectStatus(t, "faculty opens the queue", h.Do(t, http.MethodGet, "/api/v1/admin/users/review-queue", faculty.Token, nil), http.StatusForbidden)

	expectStatus(t, "HOD approves a CS request", h.Do(t, http.MethodPatch, "/api/v1/admin/users/"+csFirst+"/verify", hod.Token, nil), http.StatusOK)
	expectStatus(t, "HOD approves an EC request", h.Do(t, http.MethodPatch, "/api/v1/admin/users/"+ec+"/verify", hod.Token, nil), http.StatusNotFound)
	reject := map[string]string{"status": "rejected", "note": "Not a student here"}
	expectStatus(t, "HOD rejects a CS request", h.Do(t, http.MethodPatch, "/api/v1/admin/users/"+csSecond+"/status", hod.Token, reject), http.StatusOK)
	expectStatus(t, "HOD rejects an EC request", h.Do(t, http.MethodPatch, "/api/v1/admin/users/"+ec+"/status", hod.Token, reject), http.StatusNotFound)

	// Suspending an account stays with the principal and admins.
	member := studentOf(t, h, "CS", 2023)
	suspend := map[string]string{"status": "suspended", "note": "Misuse"}
	expectStatus(t, "HOD suspends a CS student", h.Do(t, http.MethodPatch, "/api/v1/admin/users/"+member.ID+"/status", hod.Token, suspend), http.StatusForbidden)

	var audits int
	h.DB().Raw(`SELECT count(*) FROM audit_logs WHERE actor_id = ? AND resource_id IN (?, ?)`, hod.ID, csFirst, csSecond).Scan(&audits)
	if audits < 2 {
		t.Errorf("HOD decisions audited = %d, want both", audits)
	}
}
