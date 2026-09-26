package integration

import (
	"net/http"
	"testing"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

func edit(t *testing.T, h *apitest.Harness, token, id, title string, audience []map[string]any) apitest.Response {
	t.Helper()
	return h.Do(t, http.MethodPatch, "/api/v1/announcements/"+id, token, map[string]any{
		"title": title, "body": "Updated details.", "category": "department", "audience": audience,
	})
}

func withdraw(t *testing.T, h *apitest.Harness, token, id string) apitest.Response {
	t.Helper()
	return h.Do(t, http.MethodPost, "/api/v1/announcements/"+id+"/withdraw", token, nil)
}

// publishedByFaculty posts as CS faculty and has the CS HOD approve it.
func publishedByFaculty(t *testing.T, h *apitest.Harness, faculty, hod apitest.User, title string) string {
	t.Helper()
	id, _ := createdStatus(t, publish(t, h, faculty.Token, map[string]any{"title": title, "audience": []map[string]any{{"department_id": h.DepartmentID(t, "CS")}}}))
	if response := review(t, h, hod.Token, id, "approve", ""); response.Status != http.StatusOK {
		t.Fatalf("approve status = %d: %s", response.Status, response.Body)
	}
	return id
}

func TestAuthorWithPublishingAuthorityEditsDirectly(t *testing.T) {
	h := apitest.New(t)
	csAudience := []map[string]any{{"department_id": h.DepartmentID(t, "CS")}}
	hod := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	reader := student(t, h, "CS", 2023)
	id := mustPublish(t, h, hod.Token, map[string]any{"title": "Exam on Monday", "audience": csAudience})

	response := edit(t, h, hod.Token, id, "Exam moved to Tuesday", csAudience)
	if response.Status != http.StatusOK {
		t.Fatalf("edit status = %d: %s", response.Status, response.Body)
	}
	titles := feedTitles(t, h, reader.Token)
	if !contains(titles, "Exam moved to Tuesday") || contains(titles, "Exam on Monday") {
		t.Fatalf("feed %v should show only the edited title", titles)
	}
}

func TestEditWithoutAuthorityWaitsWhileTheApprovedTextStaysVisible(t *testing.T) {
	h := apitest.New(t)
	csAudience := []map[string]any{{"department_id": h.DepartmentID(t, "CS")}}
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	hod := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	reader := student(t, h, "CS", 2023)
	id := publishedByFaculty(t, h, faculty, hod, "Lab in room 101")

	if response := edit(t, h, faculty.Token, id, "Lab in room 202", csAudience); response.Status != http.StatusOK {
		t.Fatalf("edit status = %d: %s", response.Status, response.Body)
	}
	titles := feedTitles(t, h, reader.Token)
	if !contains(titles, "Lab in room 101") || contains(titles, "Lab in room 202") {
		t.Fatalf("feed %v should still show the approved text", titles)
	}
	if !contains(queueTitles(t, h, hod.Token), "Lab in room 202") {
		t.Fatal("the edit is missing from the HOD's queue")
	}
	if response := edit(t, h, faculty.Token, id, "Lab in room 303", csAudience); response.Status != http.StatusConflict {
		t.Fatalf("second edit status = %d, want %d: %s", response.Status, http.StatusConflict, response.Body)
	}

	if response := review(t, h, hod.Token, id, "approve", ""); response.Status != http.StatusOK {
		t.Fatalf("approve status = %d: %s", response.Status, response.Body)
	}
	titles = feedTitles(t, h, reader.Token)
	if !contains(titles, "Lab in room 202") || contains(titles, "Lab in room 101") {
		t.Fatalf("feed %v should show only the approved edit", titles)
	}
}

func TestRejectedEditKeepsTheOldTextAndCanBeFixed(t *testing.T) {
	h := apitest.New(t)
	csAudience := []map[string]any{{"department_id": h.DepartmentID(t, "CS")}}
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	hod := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	reader := student(t, h, "CS", 2023)
	id := publishedByFaculty(t, h, faculty, hod, "Workshop at 2pm")

	edit(t, h, faculty.Token, id, "Workshop at 3pm", csAudience)
	if response := review(t, h, hod.Token, id, "reject", "Confirm the hall booking first"); response.Status != http.StatusOK {
		t.Fatalf("reject status = %d: %s", response.Status, response.Body)
	}
	if !contains(feedTitles(t, h, reader.Token), "Workshop at 2pm") {
		t.Fatal("rejecting the edit removed the published announcement")
	}
	item := mine(t, h, faculty.Token)["Workshop at 2pm"]
	if item.Status != "published" || item.ReviewNote == nil || *item.ReviewNote != "Confirm the hall booking first" {
		t.Fatalf("author sees %+v, want published with the edit's rejection note", item)
	}

	if response := edit(t, h, faculty.Token, id, "Workshop at 3pm in Hall B", csAudience); response.Status != http.StatusOK {
		t.Fatalf("fixed edit status = %d: %s", response.Status, response.Body)
	}
	if response := review(t, h, hod.Token, id, "approve", ""); response.Status != http.StatusOK {
		t.Fatalf("approve status = %d: %s", response.Status, response.Body)
	}
	if !contains(feedTitles(t, h, reader.Token), "Workshop at 3pm in Hall B") {
		t.Fatal("feed is missing the fixed edit")
	}
}

func TestWithdrawRemovesAnnouncementForAuthorsAndApproversOnly(t *testing.T) {
	h := apitest.New(t)
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	csHOD := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	ecHOD := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "EC"}}})
	reader := student(t, h, "CS", 2023)
	byAuthor := publishedByFaculty(t, h, faculty, csHOD, "Author withdraws this")
	byHOD := publishedByFaculty(t, h, faculty, csHOD, "HOD withdraws this")

	for name, token := range map[string]string{"EC HOD": ecHOD.Token, "student": reader.Token} {
		if response := withdraw(t, h, token, byHOD); response.Status != http.StatusForbidden {
			t.Errorf("%s withdraw status = %d, want %d: %s", name, response.Status, http.StatusForbidden, response.Body)
		}
	}
	if response := withdraw(t, h, faculty.Token, byAuthor); response.Status != http.StatusOK {
		t.Fatalf("author withdraw status = %d: %s", response.Status, response.Body)
	}
	if response := withdraw(t, h, csHOD.Token, byHOD); response.Status != http.StatusOK {
		t.Fatalf("HOD withdraw status = %d: %s", response.Status, response.Body)
	}
	titles := feedTitles(t, h, reader.Token)
	for _, gone := range []string{"Author withdraws this", "HOD withdraws this"} {
		if contains(titles, gone) {
			t.Errorf("feed %v still shows withdrawn %q", titles, gone)
		}
	}
	if response := withdraw(t, h, faculty.Token, byAuthor); response.Status != http.StatusConflict {
		t.Errorf("second withdraw status = %d, want %d: %s", response.Status, http.StatusConflict, response.Body)
	}
	if got := mine(t, h, faculty.Token)["Author withdraws this"].Status; got != "withdrawn" {
		t.Errorf("author sees status %q, want withdrawn", got)
	}
}

func TestWithdrawClosesAPendingEdit(t *testing.T) {
	h := apitest.New(t)
	csAudience := []map[string]any{{"department_id": h.DepartmentID(t, "CS")}}
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	hod := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	id := publishedByFaculty(t, h, faculty, hod, "Trip on Friday")
	edit(t, h, faculty.Token, id, "Trip on Saturday", csAudience)

	if response := withdraw(t, h, faculty.Token, id); response.Status != http.StatusOK {
		t.Fatalf("withdraw status = %d: %s", response.Status, response.Body)
	}
	if contains(queueTitles(t, h, hod.Token), "Trip on Saturday") {
		t.Fatal("the withdrawn announcement's edit is still in the queue")
	}
	if response := review(t, h, hod.Token, id, "approve", ""); response.Status != http.StatusConflict {
		t.Fatalf("approving after withdraw status = %d, want %d: %s", response.Status, http.StatusConflict, response.Body)
	}
}

func TestEditsAndWithdrawalsAreAudited(t *testing.T) {
	h := apitest.New(t)
	csAudience := []map[string]any{{"department_id": h.DepartmentID(t, "CS")}}
	hod := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	id := mustPublish(t, h, hod.Token, map[string]any{"title": "Original", "audience": csAudience})
	edit(t, h, hod.Token, id, "Edited", csAudience)
	withdraw(t, h, hod.Token, id)

	// The audit log has no API yet, so read it directly.
	var actions []string
	if err := h.DB().Raw(`SELECT action FROM audit_logs WHERE resource_id = ? ORDER BY created_at`, id).Scan(&actions).Error; err != nil {
		t.Fatalf("read audit log: %v", err)
	}
	for _, want := range []string{"announcement.published", "announcement.edited", "announcement.withdrawn"} {
		if !contains(actions, want) {
			t.Errorf("audit log %v is missing %s", actions, want)
		}
	}
}
