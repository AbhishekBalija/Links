package integration

import (
	"net/http"
	"slices"
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
		Opportunities *struct {
			Items   []opportunityItem `json:"items"`
			HasMore bool              `json:"has_more"`
		} `json:"opportunities"`
		Placement *struct {
			OpenCount           int `json:"open_count"`
			AwaitingReviewCount int `json:"awaiting_review_count"`
			Drives              []struct {
				Title           string          `json:"title"`
				ApplicantCounts applicantCounts `json:"applicant_counts"`
			} `json:"drives"`
		} `json:"placement"`
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

func TestDashboardShowsOpenJobsToStudentsAndTheDrivesToPlacementStaff(t *testing.T) {
	h := apitest.New(t)
	officer := placementOfficer(t, h)
	forCS := []map[string]any{{"department_id": h.DepartmentID(t, "CS")}}
	soon := publishedOpportunity(t, h, officer, map[string]any{"title": "Soon", "apply_by": inDays(3), "eligibility": forCS})
	publishedOpportunity(t, h, officer, map[string]any{"title": "Next week", "apply_by": inDays(10), "eligibility": forCS})
	publishedOpportunity(t, h, officer, map[string]any{"title": "Later", "apply_by": inDays(20), "eligibility": forCS})
	publishedOpportunity(t, h, officer, map[string]any{"title": "Much later", "apply_by": inDays(30), "eligibility": forCS})
	closed := publishedOpportunity(t, h, officer, map[string]any{"title": "Closed early", "eligibility": forCS})
	decodeOpportunity(t, closeOpportunity(t, h, officer.Token, closed), http.StatusOK)
	createOpportunity(t, h, officer.Token, opportunityBody(map[string]any{"title": "Still a draft", "eligibility": forCS}))

	reader := studentOf(t, h, "CS", 2023)
	shortlisted := studentOf(t, h, "CS", 2023)
	decodeApplication(t, applyTo(t, h, reader.Token, soon), http.StatusCreated)
	application := decodeApplication(t, applyTo(t, h, shortlisted.Token, soon), http.StatusCreated).ID
	expectStatus(t, "shortlist", setApplicationStatus(t, h, officer.Token, application, "applied", "shortlisted"), http.StatusOK)
	// An applied Application to a closed drive still waits for review.
	late := studentOf(t, h, "CS", 2023)
	lateDrive := publishedOpportunity(t, h, officer, map[string]any{"title": "Closing now", "apply_by": inDays(40), "eligibility": forCS})
	decodeApplication(t, applyTo(t, h, late.Token, lateDrive), http.StatusCreated)
	decodeOpportunity(t, closeOpportunity(t, h, officer.Token, lateDrive), http.StatusOK)

	student := getDashboard(t, h, reader.Token).Data
	if student.Opportunities == nil {
		t.Fatal("student dashboard has no opportunities section")
	}
	if got := opportunityTitles(student.Opportunities.Items); !slices.Equal(got, []string{"Soon", "Next week", "Later"}) || !student.Opportunities.HasMore {
		t.Errorf("student opportunities = %v (has_more %v), want the three soonest with more to come", got, student.Opportunities.HasMore)
	}
	if mine := student.Opportunities.Items[0].MyApplication; mine == nil || mine.Status != "applied" {
		t.Errorf("Soon on Home = %+v, want the student's own application", mine)
	}
	if student.Placement != nil {
		t.Errorf("student dashboard has a placement section: %+v", student.Placement)
	}
	if other := getDashboard(t, h, studentOf(t, h, "EC", 2023).Token).Data; other.Opportunities != nil {
		t.Errorf("EC student has an opportunities section with nothing open for them: %+v", other.Opportunities)
	}

	staff := getDashboard(t, h, officer.Token).Data
	if staff.Placement == nil {
		t.Fatal("placement officer dashboard has no placement section")
	}
	if staff.Placement.OpenCount != 4 || staff.Placement.AwaitingReviewCount != 2 {
		t.Errorf("placement = open %d, awaiting review %d, want 4 and 2", staff.Placement.OpenCount, staff.Placement.AwaitingReviewCount)
	}
	drives := []string{}
	for _, drive := range staff.Placement.Drives {
		drives = append(drives, drive.Title)
	}
	if !slices.Equal(drives, []string{"Soon", "Next week", "Later", "Much later"}) {
		t.Errorf("drives = %v, want the open ones, soonest deadline first", drives)
	}
	if got := staff.Placement.Drives[0].ApplicantCounts; got != (applicantCounts{Total: 2, Applied: 1, Shortlisted: 1}) {
		t.Errorf("Soon counts = %+v, want 2 in total, 1 applied and 1 shortlisted", got)
	}
	if staff.Opportunities != nil {
		t.Errorf("placement officer sees CS students' jobs on Home: %+v", staff.Opportunities)
	}

	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	if got := getDashboard(t, h, faculty.Token).Data; got.Placement != nil {
		t.Errorf("faculty dashboard has a placement section: %+v", got.Placement)
	}
}
