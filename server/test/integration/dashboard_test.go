package integration

import (
	"net/http"
	"testing"
	"time"

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
			PendingCount           int        `json:"pending_count"`
			EventsPendingCount     int        `json:"events_pending_count"`
			OldestEventSubmittedAt *time.Time `json:"oldest_event_submitted_at"`
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

func TestDashboardCountsTheEventsWaitingForEachReviewer(t *testing.T) {
	h := apitest.New(t)
	cs, me := h.DepartmentID(t, "CS"), h.DepartmentID(t, "ME")
	csFaculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	meFaculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "ME"}}})
	csHOD := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	ecHOD := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "EC"}}})
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})

	submit := func(token, department, title string) string {
		t.Helper()
		return createdEvent(t, createEvent(t, h, token, eventBody(map[string]any{"title": title, "department_id": department, "draft": false}))).ID
	}
	submit(csFaculty.Token, cs, "Waits for the CS HOD")
	submit(meFaculty.Token, me, "Waits for the principal: ME has no HOD")
	passed := submit(csFaculty.Token, cs, "Passed the CS HOD")
	expectStatus(t, "hod approve", hodReview(t, h, csHOD.Token, passed, "approve", ""), http.StatusOK)
	submit(csHOD.Token, cs, "The CS HOD's own event, straight to final")
	createEvent(t, h, csFaculty.Token, eventBody(map[string]any{"title": "A draft waits for nobody", "department_id": cs}))

	cases := []struct {
		name  string
		token string
		want  int
	}{
		{"CS HOD", csHOD.Token, 1},
		{"EC HOD", ecHOD.Token, 0},
		{"principal", principal.Token, 3},
		{"admin", admin.Token, 3},
	}
	for _, c := range cases {
		approvals := getDashboard(t, h, c.token).Data.Approvals
		if approvals == nil {
			t.Errorf("%s: no approvals section", c.name)
			continue
		}
		if approvals.EventsPendingCount != c.want {
			t.Errorf("%s: events_pending_count = %d, want %d", c.name, approvals.EventsPendingCount, c.want)
		}
		if (approvals.OldestEventSubmittedAt != nil) != (c.want > 0) {
			t.Errorf("%s: oldest_event_submitted_at = %v with %d waiting", c.name, approvals.OldestEventSubmittedAt, c.want)
		}
		if approvals.PendingCount != 0 {
			t.Errorf("%s: announcements pending_count = %d, want 0: events are counted apart", c.name, approvals.PendingCount)
		}
	}
	if approvals := getDashboard(t, h, csFaculty.Token).Data.Approvals; approvals != nil {
		t.Errorf("faculty has an approvals section: %+v", approvals)
	}
}
