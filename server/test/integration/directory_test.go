package integration

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

type directoryEntry struct {
	Username   string   `json:"username"`
	FullName   string   `json:"full_name"`
	Headline   *string  `json:"headline"`
	AvatarURL  *string  `json:"avatar_url"`
	Roles      []string `json:"roles"`
	Department *struct {
		Code string `json:"code"`
		Name string `json:"name"`
	} `json:"department"`
	BatchYear *int    `json:"batch_year"`
	Email     *string `json:"email"`
	Phone     *string `json:"phone"`
}

type directoryPage struct {
	Data []directoryEntry `json:"data"`
	Meta struct {
		NextCursor string `json:"next_cursor"`
	} `json:"meta"`
}

// member seeds a user and gives them a known name, so tests can find them.
func member(t *testing.T, h *apitest.Harness, name string, seed apitest.UserSeed) apitest.User {
	t.Helper()
	user := h.SeedUser(t, seed)
	// The ID suffix keeps usernames unique when two members share a name.
	username := strings.ToLower(strings.ReplaceAll(name, " ", ".")) + "." + user.ID[:4]
	if err := h.DB().Exec(`UPDATE profiles SET full_name = ?, username = ? WHERE user_id = ?`, name, username, user.ID).Error; err != nil {
		t.Fatalf("name member: %v", err)
	}
	return user
}

func directory(t *testing.T, h *apitest.Harness, token string, query url.Values) directoryPage {
	t.Helper()
	response := h.Do(t, http.MethodGet, "/api/v1/directory?"+query.Encode(), token, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("directory status = %d, want %d: %s", response.Status, http.StatusOK, response.Body)
	}
	var page directoryPage
	response.Decode(t, &page)
	return page
}

func directoryNames(t *testing.T, h *apitest.Harness, token string, query url.Values) []string {
	t.Helper()
	names := []string{}
	for _, entry := range directory(t, h, token, query).Data {
		names = append(names, entry.FullName)
	}
	return names
}

func sameNames(got, want []string) bool {
	return strings.Join(got, "|") == strings.Join(want, "|")
}

func TestDirectoryListsOnlyActiveVisibleMembersInEffect(t *testing.T) {
	h := apitest.New(t)
	viewer := member(t, h, "Zed Viewer", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "student"}}, Student: &apitest.StudentSeed{DepartmentCode: "CS", BatchYear: 2023}})
	member(t, h, "Asha Rao", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	member(t, h, "bala Krishna", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "student"}}, Student: &apitest.StudentSeed{DepartmentCode: "EC", BatchYear: 2024}})

	hidden := member(t, h, "Hidden Person", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "student"}}})
	suspended := member(t, h, "Suspended Person", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	pending := member(t, h, "Pending Person", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "student"}}})
	rejected := member(t, h, "Rejected Person", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "student"}}})
	unverified := member(t, h, "Unverified Person", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "student"}}})
	ended := member(t, h, "Ended Person", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	future := member(t, h, "Future Person", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	member(t, h, "Roleless Person", apitest.UserSeed{})
	for _, statement := range []struct {
		sql string
		id  string
	}{
		{`UPDATE profiles SET public_profile_enabled = false WHERE user_id = ?`, hidden.ID},
		{`UPDATE users SET status = 'suspended' WHERE id = ?`, suspended.ID},
		{`UPDATE users SET status = 'pending' WHERE id = ?`, pending.ID},
		{`UPDATE users SET status = 'rejected' WHERE id = ?`, rejected.ID},
		{`UPDATE users SET is_verified = false WHERE id = ?`, unverified.ID},
		{`UPDATE role_assignments SET ends_at = now() - interval '1 day', starts_at = now() - interval '2 days' WHERE user_id = ?`, ended.ID},
		{`UPDATE role_assignments SET starts_at = now() + interval '1 day' WHERE user_id = ?`, future.ID},
	} {
		if err := h.DB().Exec(statement.sql, statement.id).Error; err != nil {
			t.Fatalf("seed state: %v", err)
		}
	}

	got := directoryNames(t, h, viewer.Token, nil)
	if want := []string{"Asha Rao", "bala Krishna", "Zed Viewer"}; !sameNames(got, want) {
		t.Fatalf("directory = %v, want %v (alphabetical, only active visible members with a role in effect)", got, want)
	}
}

func TestDirectoryEntryShowsRolesDepartmentAndOnlyOptedInContacts(t *testing.T) {
	h := apitest.New(t)
	viewer := member(t, h, "Zed Viewer", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "ME"}}})
	head := member(t, h, "Asha Rao", apitest.UserSeed{Roles: []apitest.RoleSeed{
		{Role: "faculty", DepartmentCode: "EC"}, {Role: "faculty", DepartmentCode: "CS"}, {Role: "hod", DepartmentCode: "CS"},
	}})
	learner := member(t, h, "Bala Krishna", apitest.UserSeed{
		Roles:   []apitest.RoleSeed{{Role: "student_coordinator", DepartmentCode: "CS"}, {Role: "student"}},
		Student: &apitest.StudentSeed{DepartmentCode: "CS", BatchYear: 2023},
	})
	if err := h.DB().Exec(`UPDATE profiles SET show_email = true, headline = 'Networks and systems' WHERE user_id = ?`, head.ID).Error; err != nil {
		t.Fatalf("opt in: %v", err)
	}
	if err := h.DB().Exec(`UPDATE users SET phone = '+91 90000 00001' WHERE id IN (?, ?)`, head.ID, learner.ID).Error; err != nil {
		t.Fatalf("set phones: %v", err)
	}
	if err := h.DB().Exec(`UPDATE profiles SET show_phone = true WHERE user_id = ?`, learner.ID).Error; err != nil {
		t.Fatalf("opt in: %v", err)
	}

	response := h.Do(t, http.MethodGet, "/api/v1/directory", viewer.Token, nil)
	if strings.Contains(strings.ToLower(response.Body), "usn") || strings.Contains(response.Body, "4MN23CS") {
		t.Fatalf("the directory exposes a USN: %s", response.Body)
	}
	var page directoryPage
	response.Decode(t, &page)
	entries := map[string]directoryEntry{}
	for _, entry := range page.Data {
		entries[entry.FullName] = entry
	}

	asha := entries["Asha Rao"]
	if strings.Join(asha.Roles, ",") != "hod,faculty" {
		t.Errorf("Asha's roles = %v, want [hod faculty]", asha.Roles)
	}
	if asha.Department == nil || asha.Department.Code != "CS" || asha.Department.Name == "" {
		t.Errorf("Asha's department = %+v, want CS (her HOD role's)", asha.Department)
	}
	if asha.Email == nil || asha.Phone != nil || asha.Headline == nil || *asha.Headline != "Networks and systems" {
		t.Errorf("Asha = %+v, want her email and headline but not her phone", asha)
	}
	if asha.BatchYear != nil {
		t.Errorf("Asha has batch_year %d, but only students carry one", *asha.BatchYear)
	}

	bala := entries["Bala Krishna"]
	if strings.Join(bala.Roles, ",") != "student_coordinator,student" {
		t.Errorf("Bala's roles = %v, want [student_coordinator student]", bala.Roles)
	}
	if bala.Department == nil || bala.Department.Code != "CS" || bala.BatchYear == nil || *bala.BatchYear != 2023 {
		t.Errorf("Bala = %+v, want CS and Batch 2023", bala)
	}
	if bala.Email != nil || bala.Phone == nil {
		t.Errorf("Bala = %+v, want his phone but not his email", bala)
	}
}

func TestDirectoryFiltersCombine(t *testing.T) {
	h := apitest.New(t)
	viewer := member(t, h, "Zed Viewer", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	member(t, h, "Asha Rao", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	member(t, h, "Bala Krishna", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "student"}}, Student: &apitest.StudentSeed{DepartmentCode: "CS", BatchYear: 2023}})
	member(t, h, "Chitra Nair", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "student"}}, Student: &apitest.StudentSeed{DepartmentCode: "CS", BatchYear: 2024}})
	member(t, h, "Dev Patil", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "student"}}, Student: &apitest.StudentSeed{DepartmentCode: "EC", BatchYear: 2023}})

	cases := []struct {
		query url.Values
		want  []string
	}{
		{url.Values{"department": {"CS"}}, []string{"Asha Rao", "Bala Krishna", "Chitra Nair"}},
		{url.Values{"department": {"cs"}}, []string{"Asha Rao", "Bala Krishna", "Chitra Nair"}},
		{url.Values{"role": {"student"}}, []string{"Bala Krishna", "Chitra Nair", "Dev Patil"}},
		{url.Values{"batch": {"2023"}}, []string{"Bala Krishna", "Dev Patil"}},
		{url.Values{"department": {"CS"}, "role": {"student"}, "batch": {"2023"}}, []string{"Bala Krishna"}},
		{url.Values{"role": {"principal"}}, []string{"Zed Viewer"}},
		{url.Values{"department": {"ME"}}, []string{}},
	}
	for _, test := range cases {
		if got := directoryNames(t, h, viewer.Token, test.query); !sameNames(got, test.want) {
			t.Errorf("%s: got %v, want %v", test.query.Encode(), got, test.want)
		}
	}

	bad := map[string]string{
		"department=ZZ":  "department",
		"role=dean":      "role",
		"batch=1999":     "batch",
		"batch=2101":     "batch",
		"batch=abc":      "batch",
		"limit=0":        "limit",
		"limit=51":       "limit",
		"cursor=garbage": "cursor",
	}
	for query, field := range bad {
		response := h.Do(t, http.MethodGet, "/api/v1/directory?"+query, viewer.Token, nil)
		if response.Status != http.StatusBadRequest || !strings.Contains(response.Body, `"`+field+`"`) {
			t.Errorf("%s: status = %d, want 400 naming %s: %s", query, response.Status, field, response.Body)
		}
	}
}

func TestDirectoryPagesThroughEveryoneOnceInOrder(t *testing.T) {
	h := apitest.New(t)
	viewer := member(t, h, "Aaron Viewer", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "student"}}})
	want := []string{"Aaron Viewer"}
	// Two people share a name, so the order falls back to the ID.
	for _, name := range []string{"Bala", "Chitra", "Dev", "Dev", "Esha", "Farah", "Gopal"} {
		member(t, h, name, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "student"}}})
		want = append(want, name)
	}

	var got []string
	cursor := ""
	for pages := 0; ; pages++ {
		if pages > 10 {
			t.Fatal("pagination doesn't end")
		}
		query := url.Values{"limit": {"3"}}
		if cursor != "" {
			query.Set("cursor", cursor)
		}
		page := directory(t, h, viewer.Token, query)
		for _, entry := range page.Data {
			got = append(got, entry.FullName)
		}
		if page.Meta.NextCursor == "" {
			break
		}
		cursor = page.Meta.NextCursor
	}
	if !sameNames(got, want) {
		t.Fatalf("paged directory = %v, want %v", got, want)
	}
}

func TestDirectoryNeedsSignIn(t *testing.T) {
	h := apitest.New(t)
	if response := h.Do(t, http.MethodGet, "/api/v1/directory", "", nil); response.Status != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Status, http.StatusUnauthorized)
	}
}
