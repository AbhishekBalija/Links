package integration

import (
	"net/http"
	"testing"
	"time"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

type waitingPerson struct {
	UserID         string    `json:"user_id"`
	FullName       string    `json:"full_name"`
	Email          string    `json:"email"`
	Kind           string    `json:"kind"`
	Role           string    `json:"role"`
	USN            string    `json:"usn"`
	DepartmentCode string    `json:"department_code"`
	BatchYear      int       `json:"batch_year"`
	AddedAt        time.Time `json:"added_at"`
}

type importRun struct {
	ImportedAt time.Time `json:"imported_at"`
	ImportedBy struct {
		FullName string `json:"full_name"`
	} `json:"imported_by"`
	Rows    int `json:"rows"`
	Created int `json:"created"`
	Failed  int `json:"failed"`
	Batches []struct {
		DepartmentCode string `json:"department_code"`
		BatchYear      int    `json:"batch_year"`
		Created        int    `json:"created"`
	} `json:"batches"`
}

type listsHome struct {
	Data struct {
		Lists   *listsSection `json:"lists"`
		College *struct {
			DepartmentsWithoutHOD int `json:"departments_without_hod"`
		} `json:"college"`
	} `json:"data"`
}

func listsOf(t *testing.T, h *apitest.Harness, token string) listsHome {
	t.Helper()
	response := h.Do(t, http.MethodGet, "/api/v1/dashboard", token, nil)
	expectStatus(t, "dashboard", response, http.StatusOK)
	var home listsHome
	response.Decode(t, &home)
	return home
}

type listsSection = struct {
	WaitingCount  int             `json:"waiting_count"`
	Waiting       []waitingPerson `json:"waiting"`
	HasMore       bool            `json:"has_more"`
	RecentImports []importRun     `json:"recent_imports"`
}

// listsFor fails the test when the section is missing.
func listsFor(t *testing.T, h *apitest.Harness, token string) *listsSection {
	t.Helper()
	section := listsOf(t, h, token).Data.Lists
	if section == nil {
		t.Fatal("the dashboard has no lists section")
	}
	return section
}

func invite(t *testing.T, h *apitest.Harness, token, email, name, role, department string) {
	t.Helper()
	expectStatus(t, "invite "+email, h.Do(t, http.MethodPost, "/api/v1/admin/users", token, map[string]string{
		"email": email, "full_name": name, "role": role, "scope_type": "department", "scope_id": h.DepartmentID(t, department),
	}), http.StatusCreated)
}

func waitingEmails(people []waitingPerson) map[string]waitingPerson {
	byEmail := map[string]waitingPerson{}
	for _, person := range people {
		byEmail[person.Email] = person
	}
	return byEmail
}

func TestHomeListsWhoHasNotSignedInYetFromClassListsAndInvites(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	csHOD := member(t, h, "Asha Rao", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	ecHOD := member(t, h, "Dev Nair", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "EC"}}})
	student := studentOf(t, h, "CS", 2023)
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})

	imported(t, importCSV(h, csHOD.Token, "email,full_name,usn\nmeera@gmail.com,Meera Iyer,4MN23CS101\nravi@gmail.com,Ravi Kumar,4MN24CS102\n"))
	imported(t, importCSV(h, admin.Token, "email,full_name,usn\nkiran@gmail.com,Kiran Das,4MN23EC103\n"))
	invite(t, h, admin.Token, "teacher@college.example", "Latha Rao", "faculty", "CS")
	invite(t, h, admin.Token, "lecturer@college.example", "Om Shah", "faculty", "EC")
	// An Access request nobody has approved is not on a list.
	signUp(t, h, "4MN25CS821")

	head := listsFor(t, h, csHOD.Token)
	if head == nil || head.WaitingCount != 3 || len(head.Waiting) != 3 || head.HasMore {
		t.Fatalf("CS HOD lists = %+v, want 3 waiting in CS and nothing more", head)
	}
	people := waitingEmails(head.Waiting)
	if p := people["meera@gmail.com"]; p.Kind != "student" || p.FullName != "Meera Iyer" || p.USN != "4MN23CS101" || p.DepartmentCode != "CS" || p.BatchYear != 2023 || p.AddedAt.IsZero() {
		t.Errorf("Meera = %+v, want a CS student of Batch 2023 with the time she was added", p)
	}
	if p := people["teacher@college.example"]; p.Kind != "staff" || p.Role != "faculty" || p.DepartmentCode != "CS" || p.USN != "" {
		t.Errorf("Latha = %+v, want CS faculty", p)
	}
	if _, leaked := people["kiran@gmail.com"]; leaked {
		t.Error("the CS HOD sees an EC student")
	}
	if _, leaked := people["lecturer@college.example"]; leaked {
		t.Error("the CS HOD sees EC staff")
	}

	if got := listsOf(t, h, ecHOD.Token).Data.Lists; got == nil || got.WaitingCount != 2 {
		t.Errorf("EC HOD lists = %+v, want 2 waiting (Kiran and Om)", got)
	}
	for name, token := range map[string]string{"principal": principal.Token, "admin": admin.Token} {
		if got := listsOf(t, h, token).Data.Lists; got == nil || got.WaitingCount != 5 || len(got.Waiting) != 5 {
			t.Errorf("%s lists = %+v, want all 5 waiting", name, got)
		}
	}
	for name, token := range map[string]string{"student": student.Token, "faculty": faculty.Token} {
		if got := listsOf(t, h, token).Data.Lists; got != nil {
			t.Errorf("%s gets an lists section: %+v", name, got)
		}
	}

	signedIn, response := signInWithCode(t, h, "meera@gmail.com")
	expectStatus(t, "first sign-in", response, http.StatusOK)
	if signedIn.FirstSignIn == nil {
		t.Fatalf("first sign-in reply = %s", response.Body)
	}
	got := listsFor(t, h, csHOD.Token)
	if got.WaitingCount != 2 {
		t.Errorf("after Meera signs in the CS HOD sees %d waiting, want 2", got.WaitingCount)
	}
	if _, still := waitingEmails(got.Waiting)["meera@gmail.com"]; still {
		t.Error("Meera still waits after signing in")
	}
}

func TestTheWaitingListIsOldestFirstAndCapped(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	rows := "email,full_name,usn\n"
	for i := 0; i < 52; i++ {
		rows += "w" + pad3(i) + "@gmail.com,Student,4MN23CS" + pad3(100+i) + "\n"
	}
	imported(t, importCSV(h, admin.Token, rows))

	got := listsFor(t, h, admin.Token)
	if got.WaitingCount != 52 || len(got.Waiting) != 50 || !got.HasMore {
		t.Fatalf("waiting = %d counted, %d listed, has_more %v; want 52, 50, true", got.WaitingCount, len(got.Waiting), got.HasMore)
	}
	for i := 1; i < len(got.Waiting); i++ {
		if got.Waiting[i].AddedAt.Before(got.Waiting[i-1].AddedAt) {
			t.Fatalf("waiting list is not oldest first at %d", i)
		}
	}
}

func pad3(n int) string {
	return string([]byte{byte('0' + n/100%10), byte('0' + n/10%10), byte('0' + n%10)})
}

func TestHomeShowsRecentImportsWithWhoWhereAndHowMany(t *testing.T) {
	h := apitest.New(t)
	principal := member(t, h, "Prof Mehta", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	csHOD := member(t, h, "Asha Rao", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	ecHOD := member(t, h, "Dev Nair", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "EC"}}})

	imported(t, importCSV(h, csHOD.Token, "email,full_name,usn\na@gmail.com,A,4MN23CS101\nb@gmail.com,B,4MN23CS102\nc@gmail.com,C,4MN24CS103\nbad,D,4MN23CS104\n"))
	imported(t, importCSV(h, principal.Token, "email,full_name,usn\ne@gmail.com,E,4MN23CS105\nf@gmail.com,F,4MN23EC106\n"))

	all := listsFor(t, h, principal.Token).RecentImports
	if len(all) != 2 {
		t.Fatalf("principal sees %d imports, want 2", len(all))
	}
	if all[0].ImportedBy.FullName != "Prof Mehta" || all[0].Rows != 2 || all[0].Created != 2 || all[0].Failed != 0 || len(all[0].Batches) != 2 {
		t.Errorf("newest import = %+v, want Prof Mehta's 2 rows, 2 created, in two Departments", all[0])
	}
	older := all[1]
	if older.ImportedBy.FullName != "Asha Rao" || older.Rows != 4 || older.Created != 3 || older.Failed != 1 || older.ImportedAt.IsZero() {
		t.Errorf("older import = %+v, want Asha Rao's 4 rows, 3 created, 1 failed", older)
	}
	if len(older.Batches) != 2 || older.Batches[0].DepartmentCode != "CS" || older.Batches[0].BatchYear != 2023 || older.Batches[0].Created != 2 ||
		older.Batches[1].BatchYear != 2024 || older.Batches[1].Created != 1 {
		t.Errorf("older batches = %+v, want CS 2023 x2 and CS 2024 x1", older.Batches)
	}

	// An HOD sees the imports that created students in their Department,
	// limited to their own Department's batches.
	cs := listsFor(t, h, csHOD.Token).RecentImports
	if len(cs) != 2 || len(cs[0].Batches) != 1 || cs[0].Batches[0].DepartmentCode != "CS" || cs[0].Created != 1 {
		t.Errorf("CS HOD imports = %+v, want both, the principal's narrowed to its 1 CS student", cs)
	}
	ec := listsFor(t, h, ecHOD.Token).RecentImports
	if len(ec) != 1 || ec[0].ImportedBy.FullName != "Prof Mehta" || ec[0].Created != 1 || ec[0].Batches[0].DepartmentCode != "EC" {
		t.Errorf("EC HOD imports = %+v, want only the principal's, narrowed to EC", ec)
	}
}

func TestHomeShowsOnlyTheFiveLatestImports(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	for i := 0; i < 6; i++ {
		imported(t, importCSV(h, admin.Token, "email,full_name,usn\nx"+pad3(i)+"@gmail.com,X,4MN23CS"+pad3(200+i)+"\n"))
	}
	if got := listsFor(t, h, admin.Token).RecentImports; len(got) != 5 {
		t.Errorf("recent imports = %d, want the 5 latest", len(got))
	}
}

func TestHomeCountsDepartmentsWithoutAnHOD(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	var departments int
	h.DB().Raw(`SELECT count(*) FROM departments`).Scan(&departments)

	if got := listsOf(t, h, admin.Token).Data.College; got == nil || got.DepartmentsWithoutHOD != departments {
		t.Fatalf("college = %+v, want all %d Departments without an HOD", got, departments)
	}
	h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	if got := listsOf(t, h, admin.Token).Data.College; got.DepartmentsWithoutHOD != departments-1 {
		t.Errorf("departments without an HOD = %d, want %d once CS has one", got.DepartmentsWithoutHOD, departments-1)
	}
}
