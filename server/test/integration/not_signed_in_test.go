package integration

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

// The Not signed in list (#127): who class lists and staff invites let in
// and hasn't signed in yet, with filters, paging and who added them.

type notSignedIn struct {
	Data []struct {
		UserID         string `json:"user_id"`
		FullName       string `json:"full_name"`
		Email          string `json:"email"`
		Kind           string `json:"kind"`
		Role           string `json:"role"`
		USN            string `json:"usn"`
		DepartmentCode string `json:"department_code"`
		AddedBy        struct {
			FullName string `json:"full_name"`
		} `json:"added_by"`
	} `json:"data"`
	Meta struct {
		Total      int    `json:"total"`
		NextCursor string `json:"next_cursor"`
	} `json:"meta"`
}

func listNotSignedIn(t *testing.T, h *apitest.Harness, token string, query url.Values) notSignedIn {
	t.Helper()
	response := h.Do(t, http.MethodGet, "/api/v1/admin/users/not-signed-in?"+query.Encode(), token, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("not signed in status = %d, want %d: %s", response.Status, http.StatusOK, response.Body)
	}
	var list notSignedIn
	response.Decode(t, &list)
	return list
}

func emailsNotSignedIn(t *testing.T, h *apitest.Harness, token string, query url.Values) []string {
	t.Helper()
	response := h.Do(t, http.MethodGet, "/api/v1/admin/users/not-signed-in/emails?"+query.Encode(), token, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("emails status = %d, want %d: %s", response.Status, http.StatusOK, response.Body)
	}
	var body struct {
		Data struct {
			Emails []string `json:"emails"`
		} `json:"data"`
	}
	response.Decode(t, &body)
	return body.Data.Emails
}

func seedWaiting(t *testing.T, h *apitest.Harness) (admin apitest.User) {
	t.Helper()
	admin = member(t, h, "Nikhil Bhat", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	imported(t, importCSV(h, admin.Token, "email,full_name,usn\n"+
		"kavya@gmail.com,Kavya Rao,4MN25CS001\n"+
		"arjun@gmail.com,Arjun Nayak,4MN25CS002\n"+
		"sneha@gmail.com,Sneha D,4MN25EC019\n"))
	invite(t, h, admin.Token, "kiran@college.example", "Kiran Hegde", "faculty", "CS")
	return admin
}

func TestNotSignedInListsEveryoneWaitingOldestFirstWithWhoAddedThem(t *testing.T) {
	h := apitest.New(t)
	admin := seedWaiting(t, h)

	first := listNotSignedIn(t, h, admin.Token, url.Values{"limit": {"2"}})
	if first.Meta.Total != 4 || len(first.Data) != 2 || first.Meta.NextCursor == "" {
		t.Fatalf("first page = %+v, want 2 of 4 with a next cursor", first)
	}
	if first.Data[0].FullName != "Kavya Rao" || first.Data[0].Kind != "student" || first.Data[0].USN != "4MN25CS001" || first.Data[0].AddedBy.FullName != "Nikhil Bhat" {
		t.Errorf("oldest = %+v, want Kavya, a student added by Nikhil Bhat", first.Data[0])
	}
	second := listNotSignedIn(t, h, admin.Token, url.Values{"limit": {"2"}, "cursor": {first.Meta.NextCursor}})
	if len(second.Data) != 2 || second.Meta.NextCursor != "" || second.Data[1].Kind != "staff" || second.Data[1].Role != "faculty" {
		t.Fatalf("second page = %+v, want the last two, ending with Kiran on staff", second)
	}
}

func TestNotSignedInFiltersByDepartmentAndKind(t *testing.T) {
	h := apitest.New(t)
	admin := seedWaiting(t, h)

	ec := listNotSignedIn(t, h, admin.Token, url.Values{"department": {"EC"}})
	if ec.Meta.Total != 1 || ec.Data[0].FullName != "Sneha D" {
		t.Errorf("EC = %+v, want only Sneha", ec)
	}
	staff := listNotSignedIn(t, h, admin.Token, url.Values{"kind": {"staff"}})
	if staff.Meta.Total != 1 || staff.Data[0].FullName != "Kiran Hegde" {
		t.Errorf("staff = %+v, want only Kiran", staff)
	}
	expectStatus(t, "unknown kind", h.Do(t, http.MethodGet, "/api/v1/admin/users/not-signed-in?kind=alumni", admin.Token, nil), http.StatusBadRequest)
	expectStatus(t, "unknown department", h.Do(t, http.MethodGet, "/api/v1/admin/users/not-signed-in?department=ZZ", admin.Token, nil), http.StatusBadRequest)

	emails := emailsNotSignedIn(t, h, admin.Token, url.Values{"department": {"CS"}, "kind": {"student"}})
	if len(emails) != 2 || emails[0] != "kavya@gmail.com" || emails[1] != "arjun@gmail.com" {
		t.Errorf("CS student emails = %v, want Kavya's and Arjun's", emails)
	}
}

func TestAnHODSeesOnlyTheirOwnDepartmentsList(t *testing.T) {
	h := apitest.New(t)
	seedWaiting(t, h)
	hod := member(t, h, "Asha Rao", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	principal := member(t, h, "Suresh Kumar", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	faculty := member(t, h, "Ravi Menon", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})

	cs := listNotSignedIn(t, h, hod.Token, url.Values{})
	if cs.Meta.Total != 3 {
		t.Errorf("CS HOD sees %d, want the 3 CS people", cs.Meta.Total)
	}
	if got := emailsNotSignedIn(t, h, hod.Token, url.Values{}); len(got) != 3 {
		t.Errorf("CS HOD emails = %v, want 3", got)
	}
	expectStatus(t, "HOD naming another department", h.Do(t, http.MethodGet, "/api/v1/admin/users/not-signed-in?department=EC", hod.Token, nil), http.StatusForbidden)
	expectStatus(t, "HOD copying another department", h.Do(t, http.MethodGet, "/api/v1/admin/users/not-signed-in/emails?department=EC", hod.Token, nil), http.StatusForbidden)
	if all := listNotSignedIn(t, h, principal.Token, url.Values{}); all.Meta.Total != 4 {
		t.Errorf("principal sees %d, want all 4", all.Meta.Total)
	}
	expectStatus(t, "faculty", h.Do(t, http.MethodGet, "/api/v1/admin/users/not-signed-in", faculty.Token, nil), http.StatusForbidden)
}
