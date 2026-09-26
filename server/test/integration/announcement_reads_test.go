package integration

import (
	"net/http"
	"testing"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

type detail struct {
	Data struct {
		ID       string  `json:"id"`
		Title    string  `json:"title"`
		Status   string  `json:"status"`
		Approver *string `json:"approver"`
		Edit     *struct {
			Status     string  `json:"status"`
			ReviewNote *string `json:"review_note"`
			Approver   *string `json:"approver"`
		} `json:"edit"`
	} `json:"data"`
}

func TestAnnouncementDetailVisibility(t *testing.T) {
	h := apitest.New(t)
	csAudience := []map[string]any{{"department_id": h.DepartmentID(t, "CS")}}
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	csHOD := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	ecHOD := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "EC"}}})
	csStudent := student(t, h, "CS", 2023)
	ecStudent := student(t, h, "EC", 2023)

	pending, _ := createdStatus(t, publish(t, h, faculty.Token, map[string]any{"title": "Pending notice", "audience": csAudience}))
	published := publishedByFaculty(t, h, faculty, csHOD, "Published notice")

	tests := []struct {
		name   string
		token  string
		id     string
		status int
	}{
		{"author reads pending", faculty.Token, pending, http.StatusOK},
		{"in-scope HOD reads pending", csHOD.Token, pending, http.StatusOK},
		{"other HOD reads pending", ecHOD.Token, pending, http.StatusNotFound},
		{"CS student reads pending", csStudent.Token, pending, http.StatusNotFound},
		{"CS student reads published", csStudent.Token, published, http.StatusOK},
		{"EC student reads CS-only published", ecStudent.Token, published, http.StatusNotFound},
		{"malformed ID", csStudent.Token, "abc", http.StatusNotFound},
	}
	for _, test := range tests {
		if response := h.Do(t, http.MethodGet, "/api/v1/announcements/"+test.id, test.token, nil); response.Status != test.status {
			t.Errorf("%s: status = %d, want %d: %s", test.name, response.Status, test.status, response.Body)
		}
	}
}

func TestPendingItemsNameTheirApprover(t *testing.T) {
	h := apitest.New(t)
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	csHOD := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})

	deptResponse := publish(t, h, faculty.Token, map[string]any{"title": "Dept notice", "audience": []map[string]any{{"department_id": h.DepartmentID(t, "CS")}}})
	var dept detail
	deptResponse.Decode(t, &dept)
	if dept.Data.Approver == nil || *dept.Data.Approver != "CS HOD" {
		t.Errorf("department approver = %v, want CS HOD", dept.Data.Approver)
	}
	wideResponse := publish(t, h, csHOD.Token, map[string]any{"title": "College notice", "category": "official"})
	var wide detail
	wideResponse.Decode(t, &wide)
	if wide.Data.Approver == nil || *wide.Data.Approver != "Principal or admin" {
		t.Errorf("college-wide approver = %v, want Principal or admin", wide.Data.Approver)
	}

	// The author's edit to a published announcement shows up on its detail.
	published := publishedByFaculty(t, h, faculty, csHOD, "Published")
	edit(t, h, faculty.Token, published, "Published, edited", []map[string]any{{"department_id": h.DepartmentID(t, "CS")}})
	var withEdit detail
	h.Do(t, http.MethodGet, "/api/v1/announcements/"+published, faculty.Token, nil).Decode(t, &withEdit)
	if withEdit.Data.Edit == nil || withEdit.Data.Edit.Status != "pending" || withEdit.Data.Edit.Approver == nil || *withEdit.Data.Edit.Approver != "CS HOD" {
		t.Errorf("edit = %+v, want a pending edit for the CS HOD", withEdit.Data.Edit)
	}
}

func TestFeedFiltersByCategory(t *testing.T) {
	h := apitest.New(t)
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	reader := student(t, h, "CS", 2023)
	mustPublish(t, h, principal.Token, map[string]any{"title": "Official one", "category": "official"})
	mustPublish(t, h, principal.Token, map[string]any{"title": "Placement one", "category": "placement"})

	response := h.Do(t, http.MethodGet, "/api/v1/announcements?category=placement", reader.Token, nil)
	var feed feedResponse
	response.Decode(t, &feed)
	if len(feed.Data) != 1 || feed.Data[0].Title != "Placement one" {
		t.Errorf("placement feed = %+v, want only the placement notice", feed.Data)
	}
	if bad := h.Do(t, http.MethodGet, "/api/v1/announcements?category=gossip", reader.Token, nil); bad.Status != http.StatusBadRequest {
		t.Errorf("unknown category status = %d, want %d", bad.Status, http.StatusBadRequest)
	}
}

func TestPreviewMatchesWhatPostingDoes(t *testing.T) {
	h := apitest.New(t)
	cs := h.DepartmentID(t, "CS")
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	hod := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	placement := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "placement_officer"}}})
	reader := student(t, h, "CS", 2023)

	type previewResult struct {
		Data struct {
			PublishesDirectly bool    `json:"publishes_directly"`
			Approver          *string `json:"approver"`
		} `json:"data"`
	}
	cases := []struct {
		name     string
		token    string
		category string
		audience []map[string]any
	}{
		{"faculty to own department", faculty.Token, "department", []map[string]any{{"department_id": cs}}},
		{"HOD to own department", hod.Token, "department", []map[string]any{{"department_id": cs}}},
		{"HOD to whole college", hod.Token, "official", nil},
		{"placement officer placement", placement.Token, "placement", nil},
	}
	for _, c := range cases {
		preview := h.Do(t, http.MethodPost, "/api/v1/announcements/preview", c.token, map[string]any{"category": c.category, "audience": c.audience})
		if preview.Status != http.StatusOK {
			t.Fatalf("%s: preview status = %d: %s", c.name, preview.Status, preview.Body)
		}
		var p previewResult
		preview.Decode(t, &p)

		posted := publish(t, h, c.token, map[string]any{"title": "Post " + c.name, "category": c.category, "audience": c.audience})
		var d detail
		posted.Decode(t, &d)
		if p.Data.PublishesDirectly != (d.Data.Status == "published") {
			t.Errorf("%s: preview publishes_directly = %v, but posting gave %q", c.name, p.Data.PublishesDirectly, d.Data.Status)
		}
		if !p.Data.PublishesDirectly && (p.Data.Approver == nil || d.Data.Approver == nil || *p.Data.Approver != *d.Data.Approver) {
			t.Errorf("%s: preview approver %v != posted approver %v", c.name, p.Data.Approver, d.Data.Approver)
		}
	}
	if forbidden := h.Do(t, http.MethodPost, "/api/v1/announcements/preview", reader.Token, map[string]any{"category": "official"}); forbidden.Status != http.StatusForbidden {
		t.Errorf("student preview status = %d, want %d", forbidden.Status, http.StatusForbidden)
	}
}
