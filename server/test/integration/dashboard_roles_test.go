package integration

import (
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

type roleHome struct {
	Data struct {
		Department *struct {
			Code            string `json:"code"`
			Students        int    `json:"students"`
			Staff           int    `json:"staff"`
			StudentsByBatch []struct {
				BatchYear int `json:"batch_year"`
				Count     int `json:"count"`
			} `json:"students_by_batch"`
			UpcomingEvents []struct {
				Title string `json:"title"`
			} `json:"upcoming_events"`
		} `json:"department"`
		College *struct {
			Departments []struct {
				Code     string `json:"code"`
				Students int    `json:"students"`
				Staff    int    `json:"staff"`
				HOD      *struct {
					FullName string `json:"full_name"`
				} `json:"hod"`
			} `json:"departments"`
		} `json:"college"`
		AccessRequests *struct {
			PendingCount int        `json:"pending_count"`
			Oldest       *time.Time `json:"oldest_requested_at"`
		} `json:"access_requests"`
	} `json:"data"`
}

func roleDashboard(t *testing.T, h *apitest.Harness, token string) roleHome {
	t.Helper()
	response := h.Do(t, http.MethodGet, "/api/v1/dashboard", token, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("dashboard status = %d: %s", response.Status, response.Body)
	}
	var home roleHome
	response.Decode(t, &home)
	return home
}

func TestHomeShowsHODsTheirDepartmentAndThePrincipalTheCollege(t *testing.T) {
	h := apitest.New(t)
	hod := member(t, h, "Asha Rao", apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	faculty := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	studentOf(t, h, "CS", 2023)
	studentOf(t, h, "CS", 2023)
	studentOf(t, h, "CS", 2024)
	studentOf(t, h, "EC", 2023)
	cs, ec := h.DepartmentID(t, "CS"), h.DepartmentID(t, "EC")
	soon := time.Now().Add(3 * 24 * time.Hour).UTC().Truncate(time.Minute)
	later := time.Now().Add(20 * 24 * time.Hour).UTC().Truncate(time.Minute)
	publish := func(title, department string, starts time.Time, draft bool) {
		createdEvent(t, createEvent(t, h, principal.Token, eventBody(map[string]any{
			"title": title, "department_id": department, "draft": draft,
			"starts_at": starts.Format(time.RFC3339), "ends_at": starts.Add(time.Hour).Format(time.RFC3339),
		})))
	}
	publish("CS workshop", cs, soon, false)
	publish("CS hackathon", cs, later, false)
	publish("CS draft idea", cs, soon, true)
	publish("EC talk", ec, soon, false)
	signUp(t, h, "4MN25CS821")
	signUp(t, h, "4MN25EC822")

	head := roleDashboard(t, h, hod.Token).Data
	if head.Department == nil {
		t.Fatal("HOD Home has no department section")
	}
	if head.Department.Code != "CS" || head.Department.Students != 3 || head.Department.Staff < 1 {
		t.Errorf("department = %+v, want CS with 3 students and its staff", head.Department)
	}
	batches := map[int]int{}
	for _, b := range head.Department.StudentsByBatch {
		batches[b.BatchYear] = b.Count
	}
	if batches[2023] != 2 || batches[2024] != 1 {
		t.Errorf("students by batch = %v, want 2023: 2 and 2024: 1", batches)
	}
	titles := []string{}
	for _, event := range head.Department.UpcomingEvents {
		titles = append(titles, event.Title)
	}
	if !slices.Equal(titles, []string{"CS workshop", "CS hackathon"}) {
		t.Errorf("upcoming department events = %v, want the published CS ones, soonest first", titles)
	}
	if head.AccessRequests == nil || head.AccessRequests.PendingCount != 1 || head.AccessRequests.Oldest == nil {
		t.Errorf("HOD access requests = %+v, want the one CS request", head.AccessRequests)
	}
	if head.College != nil {
		t.Errorf("HOD Home has a college section: %+v", head.College)
	}

	top := roleDashboard(t, h, principal.Token).Data
	if top.College == nil {
		t.Fatal("principal Home has no college section")
	}
	byCode := map[string]int{}
	for i, d := range top.College.Departments {
		byCode[d.Code] = i
	}
	csRow, ecRow := top.College.Departments[byCode["CS"]], top.College.Departments[byCode["EC"]]
	if csRow.Students != 3 || csRow.HOD == nil || csRow.HOD.FullName != "Asha Rao" {
		t.Errorf("CS row = %+v, want 3 students and Asha Rao as HOD", csRow)
	}
	if ecRow.Students != 1 || ecRow.HOD != nil {
		t.Errorf("EC row = %+v, want 1 student and no HOD", ecRow)
	}
	if top.AccessRequests == nil || top.AccessRequests.PendingCount != 2 {
		t.Errorf("principal access requests = %+v, want both", top.AccessRequests)
	}
	if top.Department != nil {
		t.Errorf("principal Home has a department section: %+v", top.Department)
	}

	staff := roleDashboard(t, h, faculty.Token).Data
	if staff.Department != nil || staff.College != nil || staff.AccessRequests != nil {
		t.Errorf("faculty Home has role sections: %+v %+v %+v", staff.Department, staff.College, staff.AccessRequests)
	}
}
