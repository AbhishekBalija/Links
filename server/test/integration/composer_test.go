package integration

import (
	"net/http"
	"sort"
	"testing"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

type previewReach struct {
	Data struct {
		Reach int `json:"reach"`
	} `json:"data"`
}

func reach(t *testing.T, h *apitest.Harness, token string, audience []map[string]any) int {
	t.Helper()
	response := h.Do(t, http.MethodPost, "/api/v1/announcements/preview", token, map[string]any{"category": "department", "audience": audience})
	if response.Status != http.StatusOK {
		t.Fatalf("preview status = %d: %s", response.Status, response.Body)
	}
	var p previewReach
	response.Decode(t, &p)
	return p.Data.Reach
}

func TestPreviewCountsThePeopleTheAudienceReaches(t *testing.T) {
	h := apitest.New(t)
	cs := h.DepartmentID(t, "CS")
	author := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	// Before anyone else joins, so users seeded by migrations don't skew the count.
	baseline := reach(t, h, author.Token, nil)

	student(t, h, "CS", 2023)
	student(t, h, "CS", 2023)
	student(t, h, "CS", 2024)
	student(t, h, "EC", 2023)
	h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	suspended := student(t, h, "CS", 2023)
	if err := h.DB().Exec(`UPDATE users SET status = 'suspended' WHERE id = ?`, suspended.ID).Error; err != nil {
		t.Fatalf("suspend user: %v", err)
	}

	cases := []struct {
		name     string
		audience []map[string]any
		want     int
	}{
		{"whole college", nil, baseline + 5},
		{"everyone in CS", []map[string]any{{"department_id": cs}}, 5},
		{"CS students", []map[string]any{{"department_id": cs, "role": "student"}}, 3},
		{"CS students, batch 2023", []map[string]any{{"department_id": cs, "role": "student", "batch_year": 2023}}, 2},
		{"CS faculty", []map[string]any{{"department_id": cs, "role": "faculty"}}, 1},
		{"faculty with a batch matches no one", []map[string]any{{"role": "faculty", "batch_year": 2023}}, 0},
		{"two groups count each person once", []map[string]any{
			{"department_id": cs, "role": "student"},
			{"department_id": cs, "batch_year": 2023},
		}, 3},
	}
	for _, c := range cases {
		if got := reach(t, h, author.Token, c.audience); got != c.want {
			t.Errorf("%s: reach = %d, want %d", c.name, got, c.want)
		}
	}
}

type mineItem struct {
	Title string `json:"title"`
	Edit  *struct {
		Status string  `json:"status"`
		Title  *string `json:"title"`
		Body   *string `json:"body"`
	} `json:"edit"`
}

func mineTitles(t *testing.T, h *apitest.Harness, token, status string) []string {
	t.Helper()
	response := h.Do(t, http.MethodGet, "/api/v1/announcements/mine?status="+status, token, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("mine?status=%s status = %d: %s", status, response.Status, response.Body)
	}
	var list struct {
		Data []mineItem `json:"data"`
	}
	response.Decode(t, &list)
	titles := make([]string, 0, len(list.Data))
	for _, item := range list.Data {
		titles = append(titles, item.Title)
	}
	sort.Strings(titles)
	return titles
}

func TestMyAnnouncementsFilterByWhatNeedsTheAuthor(t *testing.T) {
	h := apitest.New(t)
	csAudience := []map[string]any{{"department_id": h.DepartmentID(t, "CS")}}
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	hod := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})

	createdStatus(t, publish(t, h, faculty.Token, map[string]any{"title": "Draft", "audience": csAudience, "draft": true}))
	createdStatus(t, publish(t, h, faculty.Token, map[string]any{"title": "Waiting", "audience": csAudience}))
	rejected, _ := createdStatus(t, publish(t, h, faculty.Token, map[string]any{"title": "Sent back", "audience": csAudience}))
	review(t, h, hod.Token, rejected, "reject", "Add the room")
	publishedByFaculty(t, h, faculty, hod, "Live")
	editWaiting := publishedByFaculty(t, h, faculty, hod, "Live, edit waiting")
	edit(t, h, faculty.Token, editWaiting, "Live, edit waiting v2", csAudience)
	editRejected := publishedByFaculty(t, h, faculty, hod, "Live, edit sent back")
	edit(t, h, faculty.Token, editRejected, "Live, edit sent back v2", csAudience)
	review(t, h, hod.Token, editRejected, "reject", "Keep the old room")
	withdrawn := publishedByFaculty(t, h, faculty, hod, "Withdrawn")
	withdraw(t, h, faculty.Token, withdrawn)
	expired := publishedByFaculty(t, h, faculty, hod, "Expired")
	if err := h.DB().Exec(`UPDATE announcements SET expires_at = now() - interval '1 day' WHERE id = ?`, expired).Error; err != nil {
		t.Fatalf("expire announcement: %v", err)
	}

	want := map[string][]string{
		"attention": {"Live, edit sent back", "Sent back"},
		"draft":     {"Draft"},
		"waiting":   {"Waiting"},
		"live":      {"Live", "Live, edit waiting"},
		"ended":     {"Expired", "Withdrawn"},
	}
	for status, titles := range want {
		got := mineTitles(t, h, faculty.Token, status)
		if len(got) != len(titles) {
			t.Errorf("status %s: got %v, want %v", status, got, titles)
			continue
		}
		for i := range got {
			if got[i] != titles[i] {
				t.Errorf("status %s: got %v, want %v", status, got, titles)
				break
			}
		}
	}
	if all := mineTitles(t, h, faculty.Token, ""); len(all) != 8 {
		t.Errorf("no filter: got %d announcements, want all 8: %v", len(all), all)
	}
	if bad := h.Do(t, http.MethodGet, "/api/v1/announcements/mine?status=archived", faculty.Token, nil); bad.Status != http.StatusBadRequest {
		t.Errorf("unknown status filter = %d, want %d", bad.Status, http.StatusBadRequest)
	}
}

func TestSentBackEditCarriesItsOwnTextForTheAuthor(t *testing.T) {
	h := apitest.New(t)
	csAudience := []map[string]any{{"department_id": h.DepartmentID(t, "CS")}}
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	hod := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	id := publishedByFaculty(t, h, faculty, hod, "Meeting at 3 pm")
	edit(t, h, faculty.Token, id, "Meeting at 4 pm", csAudience)
	review(t, h, hod.Token, id, "reject", "Keep 3 pm")

	var got struct {
		Data mineItem `json:"data"`
	}
	h.Do(t, http.MethodGet, "/api/v1/announcements/"+id, faculty.Token, nil).Decode(t, &got)
	if got.Data.Title != "Meeting at 3 pm" {
		t.Errorf("title = %q, want the live text", got.Data.Title)
	}
	if got.Data.Edit == nil || got.Data.Edit.Title == nil || *got.Data.Edit.Title != "Meeting at 4 pm" || got.Data.Edit.Body == nil {
		t.Fatalf("edit = %+v, want the sent-back edit's own text so the author can fix it", got.Data.Edit)
	}
}
