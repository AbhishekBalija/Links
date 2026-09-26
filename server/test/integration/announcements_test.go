package integration

import (
	"net/http"
	"testing"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

type feedItem struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Category string `json:"category"`
	Status   string `json:"status"`
}

type feedResponse struct {
	Data []feedItem `json:"data"`
	Meta struct {
		NextCursor string `json:"next_cursor"`
	} `json:"meta"`
}

func feedTitles(t *testing.T, h *apitest.Harness, token string) []string {
	t.Helper()
	response := h.Do(t, http.MethodGet, "/api/v1/announcements", token, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("feed status = %d, want %d: %s", response.Status, http.StatusOK, response.Body)
	}
	var feed feedResponse
	response.Decode(t, &feed)
	titles := make([]string, 0, len(feed.Data))
	for _, item := range feed.Data {
		titles = append(titles, item.Title)
	}
	return titles
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestHODPublishesToOwnDepartmentAndOnlyItsStudentsSeeIt(t *testing.T) {
	h := apitest.New(t)
	hod := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	csStudent := h.SeedUser(t, apitest.UserSeed{
		Roles:   []apitest.RoleSeed{{Role: "student"}},
		Student: &apitest.StudentSeed{DepartmentCode: "CS", BatchYear: 2023},
	})
	ecStudent := h.SeedUser(t, apitest.UserSeed{
		Roles:   []apitest.RoleSeed{{Role: "student"}},
		Student: &apitest.StudentSeed{DepartmentCode: "EC", BatchYear: 2023},
	})

	create := h.Do(t, http.MethodPost, "/api/v1/announcements", hod.Token, map[string]any{
		"title":    "CS lab closed on Friday",
		"body":     "The CS labs are closed for maintenance this Friday.",
		"category": "department",
		"audience": []map[string]any{{"department_id": h.DepartmentID(t, "CS")}},
	})
	if create.Status != http.StatusCreated {
		t.Fatalf("create status = %d, want %d: %s", create.Status, http.StatusCreated, create.Body)
	}
	var created struct {
		Data feedItem `json:"data"`
	}
	create.Decode(t, &created)
	if created.Data.Status != "published" {
		t.Errorf("status = %q, want published", created.Data.Status)
	}

	if titles := feedTitles(t, h, csStudent.Token); !contains(titles, "CS lab closed on Friday") {
		t.Errorf("CS student feed %v is missing the CS announcement", titles)
	}
	if titles := feedTitles(t, h, ecStudent.Token); contains(titles, "CS lab closed on Friday") {
		t.Errorf("EC student feed %v shows the CS-only announcement", titles)
	}
}

func student(t *testing.T, h *apitest.Harness, department string, batch int) apitest.User {
	t.Helper()
	return h.SeedUser(t, apitest.UserSeed{
		Roles:   []apitest.RoleSeed{{Role: "student"}},
		Student: &apitest.StudentSeed{DepartmentCode: department, BatchYear: batch},
	})
}

func publish(t *testing.T, h *apitest.Harness, token string, body map[string]any) apitest.Response {
	t.Helper()
	if body["body"] == nil {
		body["body"] = "Details inside."
	}
	if body["category"] == nil {
		body["category"] = "department"
	}
	return h.Do(t, http.MethodPost, "/api/v1/announcements", token, body)
}

func mustPublish(t *testing.T, h *apitest.Harness, token string, body map[string]any) string {
	t.Helper()
	response := publish(t, h, token, body)
	if response.Status != http.StatusCreated {
		t.Fatalf("publish status = %d, want %d: %s", response.Status, http.StatusCreated, response.Body)
	}
	var created struct {
		Data feedItem `json:"data"`
	}
	response.Decode(t, &created)
	return created.Data.ID
}

func TestWholeCollegeAnnouncementReachesEveryone(t *testing.T) {
	h := apitest.New(t)
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "ME"}}})
	mustPublish(t, h, principal.Token, map[string]any{"title": "Holiday on Monday", "category": "official"})

	for name, reader := range map[string]apitest.User{
		"CS student": student(t, h, "CS", 2023),
		"CV student": student(t, h, "CV", 2021),
		"ME faculty": faculty,
	} {
		if titles := feedTitles(t, h, reader.Token); !contains(titles, "Holiday on Monday") {
			t.Errorf("%s feed %v is missing the college-wide announcement", name, titles)
		}
	}
}

func TestAudienceRulesMatchBatchYearAndRole(t *testing.T) {
	h := apitest.New(t)
	hod := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	cs := h.DepartmentID(t, "CS")
	csFinalYear := student(t, h, "CS", 2022)
	csSecondYear := student(t, h, "CS", 2024)
	csFaculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	ecFaculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "EC"}}})

	mustPublish(t, h, hod.Token, map[string]any{
		"title":    "Final year project review",
		"audience": []map[string]any{{"department_id": cs, "batch_year": 2022}},
	})
	mustPublish(t, h, hod.Token, map[string]any{
		"title":    "Faculty meeting",
		"audience": []map[string]any{{"department_id": cs, "role": "faculty"}},
	})
	mustPublish(t, h, hod.Token, map[string]any{
		"title": "Seminar for final years and faculty",
		"audience": []map[string]any{
			{"department_id": cs, "batch_year": 2022},
			{"department_id": cs, "role": "faculty"},
		},
	})

	tests := []struct {
		reader apitest.User
		name   string
		sees   []string
		misses []string
	}{
		{csFinalYear, "CS 2022 student", []string{"Final year project review", "Seminar for final years and faculty"}, []string{"Faculty meeting"}},
		{csSecondYear, "CS 2024 student", nil, []string{"Final year project review", "Faculty meeting", "Seminar for final years and faculty"}},
		{csFaculty, "CS faculty", []string{"Faculty meeting", "Seminar for final years and faculty"}, []string{"Final year project review"}},
		{ecFaculty, "EC faculty", nil, []string{"Faculty meeting", "Seminar for final years and faculty"}},
	}
	for _, test := range tests {
		titles := feedTitles(t, h, test.reader.Token)
		for _, want := range test.sees {
			if !contains(titles, want) {
				t.Errorf("%s feed %v is missing %q", test.name, titles, want)
			}
		}
		for _, unwanted := range test.misses {
			if contains(titles, unwanted) {
				t.Errorf("%s feed %v shows %q", test.name, titles, unwanted)
			}
		}
	}
}

func TestExpiredAnnouncementLeavesTheFeed(t *testing.T) {
	h := apitest.New(t)
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	reader := student(t, h, "CS", 2023)
	id := mustPublish(t, h, principal.Token, map[string]any{
		"title": "Fee deadline today", "category": "official", "expires_at": "2099-01-01T00:00:00Z",
	})
	if titles := feedTitles(t, h, reader.Token); !contains(titles, "Fee deadline today") {
		t.Fatalf("feed %v is missing the announcement before expiry", titles)
	}

	// Time can't pass inside a test, so move the expiry into the past instead.
	if err := h.DB().Exec(`UPDATE announcements SET expires_at = now() - interval '1 minute' WHERE id = ?`, id).Error; err != nil {
		t.Fatalf("backdate expiry: %v", err)
	}
	if titles := feedTitles(t, h, reader.Token); contains(titles, "Fee deadline today") {
		t.Errorf("feed %v still shows the expired announcement", titles)
	}
}

func TestExpiryMustBeInTheFuture(t *testing.T) {
	h := apitest.New(t)
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	response := publish(t, h, principal.Token, map[string]any{"title": "Old news", "category": "official", "expires_at": "2020-01-01T00:00:00Z"})
	if response.Status != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d: %s", response.Status, http.StatusBadRequest, response.Body)
	}
}

func TestFeedPagesReturnEveryAnnouncementOnceNewestFirst(t *testing.T) {
	h := apitest.New(t)
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	reader := student(t, h, "CS", 2023)
	want := []string{"Notice 5", "Notice 4", "Notice 3", "Notice 2", "Notice 1"}
	for i := len(want) - 1; i >= 0; i-- {
		mustPublish(t, h, principal.Token, map[string]any{"title": want[i], "category": "official"})
	}

	var got []string
	path := "/api/v1/announcements?limit=2"
	for page := 0; page < 5; page++ {
		response := h.Do(t, http.MethodGet, path, reader.Token, nil)
		if response.Status != http.StatusOK {
			t.Fatalf("page %d status = %d: %s", page, response.Status, response.Body)
		}
		var feed feedResponse
		response.Decode(t, &feed)
		for _, item := range feed.Data {
			got = append(got, item.Title)
		}
		if feed.Meta.NextCursor == "" {
			break
		}
		path = "/api/v1/announcements?limit=2&cursor=" + feed.Meta.NextCursor
	}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestAuthorsWithoutPublishingAuthorityAreRefusedForNow(t *testing.T) {
	h := apitest.New(t)
	cs := h.DepartmentID(t, "CS")
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	ecHOD := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "EC"}}})
	placement := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "placement_officer"}}})
	reader := student(t, h, "CS", 2023)

	tests := []struct {
		name   string
		token  string
		body   map[string]any
		status int
	}{
		{"faculty to own department", faculty.Token, map[string]any{"title": "Lab update", "audience": []map[string]any{{"department_id": cs}}}, http.StatusForbidden},
		{"HOD to another department", ecHOD.Token, map[string]any{"title": "Guest lecture", "audience": []map[string]any{{"department_id": cs}}}, http.StatusForbidden},
		{"student", reader.Token, map[string]any{"title": "Party tonight"}, http.StatusForbidden},
		{"placement officer, non-placement", placement.Token, map[string]any{"title": "Sports day", "category": "official"}, http.StatusForbidden},
		{"placement officer, placement", placement.Token, map[string]any{"title": "Infosys drive on Monday", "category": "placement"}, http.StatusCreated},
	}
	for _, test := range tests {
		if response := publish(t, h, test.token, test.body); response.Status != test.status {
			t.Errorf("%s: status = %d, want %d: %s", test.name, response.Status, test.status, response.Body)
		}
	}

	titles := feedTitles(t, h, reader.Token)
	for _, refused := range []string{"Lab update", "Guest lecture", "Party tonight", "Sports day"} {
		if contains(titles, refused) {
			t.Errorf("feed %v shows refused announcement %q", titles, refused)
		}
	}
	if !contains(titles, "Infosys drive on Monday") {
		t.Errorf("feed %v is missing the placement announcement", titles)
	}
}

func TestInvalidAudienceIsRejected(t *testing.T) {
	h := apitest.New(t)
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	tests := map[string][]map[string]any{
		"unknown department": {{"department_id": "00000000-0000-0000-0000-000000000000"}},
		"not a UUID":         {{"department_id": "CS"}},
		"not a LINKS role":   {{"role": "janitor"}},
		"empty rule":         {{}},
		"batch out of range": {{"batch_year": 1990}},
	}
	for name, audience := range tests {
		response := publish(t, h, principal.Token, map[string]any{"title": "Bad audience", "category": "official", "audience": audience})
		if response.Status != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want %d: %s", name, response.Status, http.StatusBadRequest, response.Body)
		}
	}
}

func TestPublishingIsAudited(t *testing.T) {
	h := apitest.New(t)
	hod := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	id := mustPublish(t, h, hod.Token, map[string]any{
		"title": "Audited notice", "audience": []map[string]any{{"department_id": h.DepartmentID(t, "CS")}},
	})

	// The audit log has no API yet, so read it directly.
	var actors []string
	err := h.DB().Raw(`SELECT actor_id::text FROM audit_logs WHERE action = 'announcement.published' AND resource_id = ?`, id).Scan(&actors).Error
	if err != nil {
		t.Fatalf("read audit log: %v", err)
	}
	if len(actors) != 1 || actors[0] != hod.ID {
		t.Errorf("audit actors = %v, want [%s]", actors, hod.ID)
	}
}

func TestEndedHODRoleNoLongerPublishesEvenWithAnOldToken(t *testing.T) {
	h := apitest.New(t)
	hod := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	// The access token still says "hod", but the role has ended in the database.
	if err := h.DB().Exec(`UPDATE role_assignments SET ends_at = now() - interval '1 minute' WHERE user_id = ?`, hod.ID).Error; err != nil {
		t.Fatalf("end role: %v", err)
	}
	response := publish(t, h, hod.Token, map[string]any{
		"title": "Should not publish", "audience": []map[string]any{{"department_id": h.DepartmentID(t, "CS")}},
	})
	if response.Status != http.StatusForbidden {
		t.Fatalf("status = %d, want %d: %s", response.Status, http.StatusForbidden, response.Body)
	}
}
