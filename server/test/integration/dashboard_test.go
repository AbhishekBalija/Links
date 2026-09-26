package integration

import (
	"net/http"
	"testing"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

type dashboard struct {
	Data struct {
		User struct {
			FullName   string   `json:"full_name"`
			Roles      []string `json:"roles"`
			Department *struct {
				Code string `json:"code"`
			} `json:"department"`
		} `json:"user"`
		Notices struct {
			Items   []feedItem `json:"items"`
			HasMore bool       `json:"has_more"`
		} `json:"notices"`
		Approvals *struct {
			PendingCount int `json:"pending_count"`
		} `json:"approvals"`
		MyAnnouncements *struct {
			Draft        int `json:"draft"`
			Pending      int `json:"pending"`
			Rejected     int `json:"rejected"`
			EditsWaiting int `json:"edits_waiting"`
		} `json:"my_announcements"`
	} `json:"data"`
}

func getDashboard(t *testing.T, h *apitest.Harness, token string) dashboard {
	t.Helper()
	response := h.Do(t, http.MethodGet, "/api/v1/dashboard", token, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("dashboard status = %d: %s", response.Status, response.Body)
	}
	var result dashboard
	response.Decode(t, &result)
	return result
}

func TestStudentDashboardShowsLatestNoticesOnly(t *testing.T) {
	h := apitest.New(t)
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	reader := student(t, h, "CS", 2023)
	for _, title := range []string{"N1", "N2", "N3", "N4", "N5", "N6"} {
		mustPublish(t, h, principal.Token, map[string]any{"title": "Notice " + title, "category": "official"})
	}

	got := getDashboard(t, h, reader.Token).Data
	if got.User.Department == nil || got.User.Department.Code != "CS" {
		t.Errorf("department = %+v, want CS", got.User.Department)
	}
	if len(got.Notices.Items) != 5 || !got.Notices.HasMore || got.Notices.Items[0].Title != "Notice N6" {
		t.Errorf("notices = %+v (has_more %v), want the newest five with more to come", got.Notices.Items, got.Notices.HasMore)
	}
	if got.Approvals != nil || got.MyAnnouncements != nil {
		t.Errorf("student dashboard has staff sections: approvals %+v, mine %+v", got.Approvals, got.MyAnnouncements)
	}
}

func TestStaffDashboardShowsTheirWorkSections(t *testing.T) {
	h := apitest.New(t)
	csAudience := []map[string]any{{"department_id": h.DepartmentID(t, "CS")}}
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	hod := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})

	publish(t, h, faculty.Token, map[string]any{"title": "Draft one", "draft": true, "audience": csAudience})
	pendingID, _ := createdStatus(t, publish(t, h, faculty.Token, map[string]any{"title": "Pending one", "audience": csAudience}))
	rejectedID, _ := createdStatus(t, publish(t, h, faculty.Token, map[string]any{"title": "Rejected one", "audience": csAudience}))
	review(t, h, hod.Token, rejectedID, "reject", "No")
	published := publishedByFaculty(t, h, faculty, hod, "Published one")
	edit(t, h, faculty.Token, published, "Published one, edited", csAudience)
	_ = pendingID

	mine := getDashboard(t, h, faculty.Token).Data.MyAnnouncements
	if mine == nil || mine.Draft != 1 || mine.Pending != 1 || mine.Rejected != 1 || mine.EditsWaiting != 1 {
		t.Errorf("faculty my_announcements = %+v, want 1 draft, 1 pending, 1 rejected, 1 edit waiting", mine)
	}
	if approvals := getDashboard(t, h, faculty.Token).Data.Approvals; approvals != nil {
		t.Errorf("faculty has an approvals section: %+v", approvals)
	}
	// Waiting for the CS HOD: the pending submission and the edit.
	if approvals := getDashboard(t, h, hod.Token).Data.Approvals; approvals == nil || approvals.PendingCount != 2 {
		t.Errorf("HOD approvals = %+v, want 2 pending", approvals)
	}
	if approvals := getDashboard(t, h, principal.Token).Data.Approvals; approvals == nil || approvals.PendingCount != 2 {
		t.Errorf("principal approvals = %+v, want 2 pending", approvals)
	}
}
