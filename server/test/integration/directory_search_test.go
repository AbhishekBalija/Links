package integration

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

func searchNames(t *testing.T, h *apitest.Harness, token, q string, extra url.Values) []string {
	t.Helper()
	query := url.Values{"q": {q}}
	for key, values := range extra {
		query[key] = values
	}
	return directoryNames(t, h, token, query)
}

func TestDirectorySearchIsCaseInsensitiveAndTypoTolerant(t *testing.T) {
	h := apitest.New(t)
	viewer := member(t, h, "Zed Viewer", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "student"}}})
	member(t, h, "Asha Rao", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	member(t, h, "Ashok Kumar", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "EC"}}})
	member(t, h, "Bala Krishna", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "student"}}, Student: &apitest.StudentSeed{DepartmentCode: "CS", BatchYear: 2023}})

	for _, q := range []string{"ash", "ASHA", "ahsa", "asha rau", "Rao"} {
		got := searchNames(t, h, viewer.Token, q, nil)
		if len(got) == 0 || got[0] != "Asha Rao" {
			t.Errorf("q=%q: got %v, want Asha Rao first", q, got)
		}
	}
	if got := searchNames(t, h, viewer.Token, "krisna", nil); !sameNames(got, []string{"Bala Krishna"}) {
		t.Errorf("q=krisna: got %v, want [Bala Krishna]", got)
	}
	if got := searchNames(t, h, viewer.Token, "abcd", nil); len(got) != 0 {
		t.Errorf("q=abcd: got %v, want nobody", got)
	}
	// Two equally good matches come back in name order.
	if got := searchNames(t, h, viewer.Token, "ash", nil); !sameNames(got, []string{"Asha Rao", "Ashok Kumar"}) {
		t.Errorf("q=ash: got %v, want [Asha Rao Ashok Kumar]", got)
	}
}

func TestDirectorySearchMatchesUsernameAndHeadlineButNotUSN(t *testing.T) {
	h := apitest.New(t)
	viewer := member(t, h, "Zed Viewer", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "student"}}})
	teacher := member(t, h, "Asha Rao", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	learner := member(t, h, "Bala Krishna", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "student"}}, Student: &apitest.StudentSeed{DepartmentCode: "CS", BatchYear: 2023}})
	if err := h.DB().Exec(`UPDATE profiles SET headline = 'Computer networks researcher' WHERE user_id = ?`, teacher.ID).Error; err != nil {
		t.Fatalf("set headline: %v", err)
	}
	if err := h.DB().Exec(`UPDATE profiles SET username = 'bk_codes' WHERE user_id = ?`, learner.ID).Error; err != nil {
		t.Fatalf("set username: %v", err)
	}
	var usn string
	h.DB().Raw(`SELECT usn FROM student_identities WHERE user_id = ?`, learner.ID).Scan(&usn)

	if got := searchNames(t, h, viewer.Token, "networks", nil); !sameNames(got, []string{"Asha Rao"}) {
		t.Errorf("headline search: got %v, want [Asha Rao]", got)
	}
	if got := searchNames(t, h, viewer.Token, "bk_codes", nil); !sameNames(got, []string{"Bala Krishna"}) {
		t.Errorf("username search: got %v, want [Bala Krishna]", got)
	}
	for _, q := range []string{usn, strings.ToLower(usn), usn[:7]} {
		if got := searchNames(t, h, viewer.Token, q, nil); len(got) != 0 {
			t.Errorf("q=%q matched %v by USN", q, got)
		}
	}
}

func TestDirectorySearchCombinesWithFiltersAndIgnoresShortQueries(t *testing.T) {
	h := apitest.New(t)
	viewer := member(t, h, "Zed Viewer", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "student"}}})
	member(t, h, "Asha Rao", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	member(t, h, "Ashok Kumar", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "EC"}}})
	hidden := member(t, h, "Ashwin Hidden", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "student"}}})
	h.DB().Exec(`UPDATE profiles SET public_profile_enabled = false WHERE user_id = ?`, hidden.ID)

	if got := searchNames(t, h, viewer.Token, "ash", url.Values{"department": {"EC"}}); !sameNames(got, []string{"Ashok Kumar"}) {
		t.Errorf("ash in EC: got %v, want [Ashok Kumar]", got)
	}
	// A one-character q is ignored: the whole directory, in name order.
	if got := searchNames(t, h, viewer.Token, "a", nil); !sameNames(got, []string{"Asha Rao", "Ashok Kumar", "Zed Viewer"}) {
		t.Errorf("q=a: got %v, want everyone visible", got)
	}

	response := h.Do(t, http.MethodGet, "/api/v1/directory?q="+strings.Repeat("a", 101), viewer.Token, nil)
	if response.Status != http.StatusBadRequest || !strings.Contains(response.Body, `"q"`) {
		t.Errorf("long q status = %d, want 400 naming q: %s", response.Status, response.Body)
	}
	response = h.Do(t, http.MethodGet, "/api/v1/directory?q=ash&cursor=abc", viewer.Token, nil)
	if response.Status != http.StatusBadRequest || !strings.Contains(response.Body, `"cursor"`) {
		t.Errorf("q with cursor status = %d, want 400 naming cursor: %s", response.Status, response.Body)
	}
	page := directory(t, h, viewer.Token, url.Values{"q": {"ash"}, "limit": {"1"}})
	if len(page.Data) != 2 || page.Meta.NextCursor != "" {
		t.Errorf("search page = %d entries, cursor %q; want all matches (limit ignored) and no cursor", len(page.Data), page.Meta.NextCursor)
	}
}
