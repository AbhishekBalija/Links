package integration

import (
	"net/http"
	"sync"
	"testing"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

type authoredItem struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Status     string  `json:"status"`
	ReviewNote *string `json:"review_note"`
}

func createdStatus(t *testing.T, response apitest.Response) (string, string) {
	t.Helper()
	if response.Status != http.StatusCreated {
		t.Fatalf("create status = %d, want %d: %s", response.Status, http.StatusCreated, response.Body)
	}
	var created struct {
		Data authoredItem `json:"data"`
	}
	response.Decode(t, &created)
	return created.Data.ID, created.Data.Status
}

func queueTitles(t *testing.T, h *apitest.Harness, token string) []string {
	t.Helper()
	response := h.Do(t, http.MethodGet, "/api/v1/announcements/approvals", token, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("queue status = %d, want %d: %s", response.Status, http.StatusOK, response.Body)
	}
	var queue struct {
		Data []authoredItem `json:"data"`
	}
	response.Decode(t, &queue)
	titles := []string{}
	for _, item := range queue.Data {
		titles = append(titles, item.Title)
	}
	return titles
}

func mine(t *testing.T, h *apitest.Harness, token string) map[string]authoredItem {
	t.Helper()
	response := h.Do(t, http.MethodGet, "/api/v1/announcements/mine", token, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("mine status = %d, want %d: %s", response.Status, http.StatusOK, response.Body)
	}
	var list struct {
		Data []authoredItem `json:"data"`
	}
	response.Decode(t, &list)
	byTitle := map[string]authoredItem{}
	for _, item := range list.Data {
		byTitle[item.Title] = item
	}
	return byTitle
}

func review(t *testing.T, h *apitest.Harness, token, id, decision, note string) apitest.Response {
	t.Helper()
	return h.Do(t, http.MethodPatch, "/api/v1/announcements/"+id+"/approval", token, map[string]string{"decision": decision, "note": note})
}

func TestFacultyAnnouncementPublishesOnlyAfterTheirHODApproves(t *testing.T) {
	h := apitest.New(t)
	cs := h.DepartmentID(t, "CS")
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	csHOD := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	ecHOD := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "EC"}}})
	reader := student(t, h, "CS", 2023)

	id, status := createdStatus(t, publish(t, h, faculty.Token, map[string]any{"title": "Lab moved to block B", "audience": []map[string]any{{"department_id": cs}}}))
	if status != "pending" {
		t.Fatalf("status = %q, want pending", status)
	}
	if contains(feedTitles(t, h, reader.Token), "Lab moved to block B") {
		t.Fatal("pending announcement is visible in the feed")
	}
	if !contains(queueTitles(t, h, csHOD.Token), "Lab moved to block B") {
		t.Fatal("CS HOD's queue is missing the submission")
	}
	if contains(queueTitles(t, h, ecHOD.Token), "Lab moved to block B") {
		t.Fatal("EC HOD's queue shows a CS submission")
	}
	if response := review(t, h, ecHOD.Token, id, "approve", ""); response.Status != http.StatusForbidden {
		t.Fatalf("EC HOD approve status = %d, want %d: %s", response.Status, http.StatusForbidden, response.Body)
	}

	if response := review(t, h, csHOD.Token, id, "approve", ""); response.Status != http.StatusOK {
		t.Fatalf("approve status = %d, want %d: %s", response.Status, http.StatusOK, response.Body)
	}
	if !contains(feedTitles(t, h, reader.Token), "Lab moved to block B") {
		t.Fatal("approved announcement is missing from the feed")
	}
	if contains(queueTitles(t, h, csHOD.Token), "Lab moved to block B") {
		t.Fatal("approved announcement is still in the queue")
	}
	if got := mine(t, h, faculty.Token)["Lab moved to block B"].Status; got != "published" {
		t.Fatalf("author sees status %q, want published", got)
	}
}

func TestPrincipalApprovesWiderAudiencesAndDepartmentsWithoutAnHOD(t *testing.T) {
	h := apitest.New(t)
	cs := h.DepartmentID(t, "CS")
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	csHOD := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	cvFaculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CV"}}})
	reader := student(t, h, "CV", 2022)

	collegeWide, _ := createdStatus(t, publish(t, h, csHOD.Token, map[string]any{"title": "Blood donation camp", "category": "official"}))
	cvOnly, _ := createdStatus(t, publish(t, h, cvFaculty.Token, map[string]any{"title": "Survey camp", "audience": []map[string]any{{"department_id": h.DepartmentID(t, "CV")}}}))

	if contains(queueTitles(t, h, csHOD.Token), "Blood donation camp") {
		t.Error("an HOD's own college-wide submission is in their own queue")
	}
	queue := queueTitles(t, h, principal.Token)
	for _, want := range []string{"Blood donation camp", "Survey camp"} {
		if !contains(queue, want) {
			t.Errorf("principal queue %v is missing %q", queue, want)
		}
	}
	for _, id := range []string{collegeWide, cvOnly} {
		if response := review(t, h, principal.Token, id, "approve", ""); response.Status != http.StatusOK {
			t.Fatalf("principal approve status = %d: %s", response.Status, response.Body)
		}
	}
	titles := feedTitles(t, h, reader.Token)
	for _, want := range []string{"Blood donation camp", "Survey camp"} {
		if !contains(titles, want) {
			t.Errorf("feed %v is missing %q", titles, want)
		}
	}
	_ = cs
}

func TestRejectedAnnouncementCarriesTheNoteAndCanBeFixedAndResubmitted(t *testing.T) {
	h := apitest.New(t)
	cs := h.DepartmentID(t, "CS")
	coordinator := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "student_coordinator", DepartmentCode: "CS"}}})
	csHOD := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	reader := student(t, h, "CS", 2023)

	id, _ := createdStatus(t, publish(t, h, coordinator.Token, map[string]any{"title": "Hackathon on Saturday", "audience": []map[string]any{{"department_id": cs}}}))
	if response := review(t, h, csHOD.Token, id, "reject", "Add the venue and timings"); response.Status != http.StatusOK {
		t.Fatalf("reject status = %d: %s", response.Status, response.Body)
	}
	rejected := mine(t, h, coordinator.Token)["Hackathon on Saturday"]
	if rejected.Status != "rejected" || rejected.ReviewNote == nil || *rejected.ReviewNote != "Add the venue and timings" {
		t.Fatalf("author sees %+v, want rejected with the note", rejected)
	}
	if contains(queueTitles(t, h, csHOD.Token), "Hackathon on Saturday") {
		t.Fatal("rejected announcement is still in the queue")
	}

	edit := h.Do(t, http.MethodPatch, "/api/v1/announcements/"+id, coordinator.Token, map[string]any{
		"title": "Hackathon on Saturday, 10am, Seminar Hall", "body": "Bring your laptops.", "category": "department",
		"audience": []map[string]any{{"department_id": cs}},
	})
	if edit.Status != http.StatusOK {
		t.Fatalf("edit status = %d: %s", edit.Status, edit.Body)
	}
	submit := h.Do(t, http.MethodPost, "/api/v1/announcements/"+id+"/submit-for-approval", coordinator.Token, nil)
	if submit.Status != http.StatusOK {
		t.Fatalf("resubmit status = %d: %s", submit.Status, submit.Body)
	}
	if response := review(t, h, csHOD.Token, id, "approve", ""); response.Status != http.StatusOK {
		t.Fatalf("approve status = %d: %s", response.Status, response.Body)
	}
	if !contains(feedTitles(t, h, reader.Token), "Hackathon on Saturday, 10am, Seminar Hall") {
		t.Fatal("feed is missing the fixed announcement")
	}
}

func TestDraftsStayPrivateUntilSubmitted(t *testing.T) {
	h := apitest.New(t)
	cs := h.DepartmentID(t, "CS")
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	csHOD := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})

	draftID, status := createdStatus(t, publish(t, h, faculty.Token, map[string]any{"title": "Draft notice", "draft": true, "audience": []map[string]any{{"department_id": cs}}}))
	if status != "draft" {
		t.Fatalf("status = %q, want draft", status)
	}
	if contains(queueTitles(t, h, csHOD.Token), "Draft notice") {
		t.Fatal("a draft is in the approval queue")
	}
	if response := h.Do(t, http.MethodPost, "/api/v1/announcements/"+draftID+"/submit-for-approval", faculty.Token, nil); response.Status != http.StatusOK {
		t.Fatalf("submit status = %d: %s", response.Status, response.Body)
	}
	if !contains(queueTitles(t, h, csHOD.Token), "Draft notice") {
		t.Fatal("submitted draft is missing from the queue")
	}

	// An author with Publishing authority publishes a draft directly when submitting it.
	hodDraft, _ := createdStatus(t, publish(t, h, csHOD.Token, map[string]any{"title": "HOD draft", "draft": true, "audience": []map[string]any{{"department_id": cs}}}))
	if response := h.Do(t, http.MethodPost, "/api/v1/announcements/"+hodDraft+"/submit-for-approval", csHOD.Token, nil); response.Status != http.StatusOK {
		t.Fatalf("HOD submit status = %d: %s", response.Status, response.Body)
	}
	if got := mine(t, h, csHOD.Token)["HOD draft"].Status; got != "published" {
		t.Fatalf("HOD draft status after submit = %q, want published", got)
	}
}

func TestOnlyTheAuthorCanEditOrSubmitTheirAnnouncement(t *testing.T) {
	h := apitest.New(t)
	cs := h.DepartmentID(t, "CS")
	author := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	other := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	id, _ := createdStatus(t, publish(t, h, author.Token, map[string]any{"title": "Mine", "draft": true, "audience": []map[string]any{{"department_id": cs}}}))

	if response := h.Do(t, http.MethodPost, "/api/v1/announcements/"+id+"/submit-for-approval", other.Token, nil); response.Status != http.StatusNotFound {
		t.Errorf("other submit status = %d, want %d: %s", response.Status, http.StatusNotFound, response.Body)
	}
	edit := h.Do(t, http.MethodPatch, "/api/v1/announcements/"+id, other.Token, map[string]any{"title": "Hijacked", "body": "x", "category": "department"})
	if edit.Status != http.StatusNotFound {
		t.Errorf("other edit status = %d, want %d: %s", edit.Status, http.StatusNotFound, edit.Body)
	}
}

func TestPendingAnnouncementCannotBeReviewedTwice(t *testing.T) {
	h := apitest.New(t)
	cs := h.DepartmentID(t, "CS")
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	csHOD := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	id, _ := createdStatus(t, publish(t, h, faculty.Token, map[string]any{"title": "Race", "audience": []map[string]any{{"department_id": cs}}}))

	var wg sync.WaitGroup
	statuses := make([]int, 2)
	for i, reviewer := range []struct{ token, decision string }{{csHOD.Token, "approve"}, {admin.Token, "reject"}} {
		wg.Add(1)
		go func(i int, token, decision string) {
			defer wg.Done()
			statuses[i] = review(t, h, token, id, decision, "no").Status
		}(i, reviewer.token, reviewer.decision)
	}
	wg.Wait()

	ok, conflict := 0, 0
	for _, status := range statuses {
		switch status {
		case http.StatusOK:
			ok++
		case http.StatusConflict:
			conflict++
		}
	}
	if ok != 1 || conflict != 1 {
		t.Fatalf("statuses = %v, want one 200 and one 409", statuses)
	}
}

func TestApprovalStepsAreAudited(t *testing.T) {
	h := apitest.New(t)
	cs := h.DepartmentID(t, "CS")
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	csHOD := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	id, _ := createdStatus(t, publish(t, h, faculty.Token, map[string]any{"title": "Audited", "audience": []map[string]any{{"department_id": cs}}}))
	review(t, h, csHOD.Token, id, "reject", "Too short")
	h.Do(t, http.MethodPost, "/api/v1/announcements/"+id+"/submit-for-approval", faculty.Token, nil)
	review(t, h, csHOD.Token, id, "approve", "")

	// The audit log has no API yet, so read it directly.
	var actions []string
	if err := h.DB().Raw(`SELECT action FROM audit_logs WHERE resource_id = ? ORDER BY created_at, action`, id).Scan(&actions).Error; err != nil {
		t.Fatalf("read audit log: %v", err)
	}
	want := map[string]int{"announcement.submitted": 2, "announcement.rejected": 1, "announcement.approved": 1}
	got := map[string]int{}
	for _, action := range actions {
		got[action]++
	}
	for action, count := range want {
		if got[action] != count {
			t.Errorf("audit %s count = %d, want %d (all: %v)", action, got[action], count, actions)
		}
	}
}

func TestMalformedAnnouncementIDIsNotFound(t *testing.T) {
	h := apitest.New(t)
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	calls := []struct {
		method, path, token string
		body                any
	}{
		{http.MethodPatch, "/api/v1/announcements/abc", faculty.Token, map[string]any{"title": "Title", "body": "Body", "category": "department"}},
		{http.MethodPost, "/api/v1/announcements/abc/submit-for-approval", faculty.Token, nil},
		{http.MethodPatch, "/api/v1/announcements/abc/approval", admin.Token, map[string]string{"decision": "approve"}},
	}
	for _, call := range calls {
		if response := h.Do(t, call.method, call.path, call.token, call.body); response.Status != http.StatusNotFound {
			t.Errorf("%s %s: status = %d, want %d: %s", call.method, call.path, response.Status, http.StatusNotFound, response.Body)
		}
	}
}

func TestExpiredAnnouncementCannotBeSubmittedOrApproved(t *testing.T) {
	h := apitest.New(t)
	cs := h.DepartmentID(t, "CS")
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	csHOD := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	body := func(title string, draft bool) map[string]any {
		return map[string]any{"title": title, "draft": draft, "expires_at": "2099-01-01T00:00:00Z", "audience": []map[string]any{{"department_id": cs}}}
	}
	draft, _ := createdStatus(t, publish(t, h, faculty.Token, body("Expiring draft", true)))
	pending, _ := createdStatus(t, publish(t, h, faculty.Token, body("Expiring submission", false)))

	// Time can't pass inside a test, so move both expiries into the past.
	if err := h.DB().Exec(`UPDATE announcement_revisions SET expires_at = now() - interval '1 minute'`).Error; err != nil {
		t.Fatalf("backdate expiry: %v", err)
	}

	if response := h.Do(t, http.MethodPost, "/api/v1/announcements/"+draft+"/submit-for-approval", faculty.Token, nil); response.Status != http.StatusBadRequest {
		t.Errorf("submit status = %d, want %d: %s", response.Status, http.StatusBadRequest, response.Body)
	}
	if response := review(t, h, csHOD.Token, pending, "approve", ""); response.Status != http.StatusBadRequest {
		t.Errorf("approve status = %d, want %d: %s", response.Status, http.StatusBadRequest, response.Body)
	}
	if response := review(t, h, csHOD.Token, pending, "reject", "Expired, please update the date"); response.Status != http.StatusOK {
		t.Errorf("reject status = %d, want %d: %s", response.Status, http.StatusOK, response.Body)
	}
}
