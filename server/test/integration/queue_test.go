package integration

import (
	"net/http"
	"testing"
	"time"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

type queueItem struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Kind        string     `json:"kind"`
	SubmittedAt *time.Time `json:"submitted_at"`
	Live        *struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	} `json:"live"`
}

func TestQueueSaysWhetherAnItemIsNewOrAnEditAndShowsTheLiveText(t *testing.T) {
	h := apitest.New(t)
	csAudience := []map[string]any{{"department_id": h.DepartmentID(t, "CS")}}
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	hod := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})

	createdStatus(t, publish(t, h, faculty.Token, map[string]any{"title": "Brand new", "audience": csAudience}))
	live := publishedByFaculty(t, h, faculty, hod, "Reviews on 14 October")
	edit(t, h, faculty.Token, live, "Reviews on 16 October", csAudience)

	response := h.Do(t, http.MethodGet, "/api/v1/announcements/approvals", hod.Token, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("queue status = %d: %s", response.Status, response.Body)
	}
	var queue struct {
		Data []queueItem `json:"data"`
	}
	response.Decode(t, &queue)
	byTitle := map[string]queueItem{}
	for _, item := range queue.Data {
		byTitle[item.Title] = item
	}

	fresh := byTitle["Brand new"]
	if fresh.Kind != "new" || fresh.SubmittedAt == nil || fresh.Live != nil {
		t.Errorf("new item = %+v, want kind new, a submitted time and no live text", fresh)
	}
	edited := byTitle["Reviews on 16 October"]
	if edited.Kind != "edit" || edited.SubmittedAt == nil {
		t.Errorf("edit item = %+v, want kind edit with a submitted time", edited)
	}
	if edited.Live == nil || edited.Live.Title != "Reviews on 14 October" || edited.Live.Body == "" {
		t.Errorf("edit live = %+v, want the text readers see now", edited.Live)
	}
}
