package integration

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

type roleAssignment struct {
	ID         string  `json:"id"`
	Role       string  `json:"role"`
	ScopeType  string  `json:"scope_type"`
	ScopeID    *string `json:"scope_id"`
	Department *struct {
		Code string `json:"code"`
	} `json:"department"`
	StartsAt time.Time  `json:"starts_at"`
	EndsAt   *time.Time `json:"ends_at"`
	State    string     `json:"state"`
}

func grantRole(t *testing.T, h *apitest.Harness, token, userID string, body map[string]any) apitest.Response {
	t.Helper()
	return h.Do(t, http.MethodPost, "/api/v1/admin/users/"+userID+"/roles", token, body)
}

func grantedID(t *testing.T, response apitest.Response) string {
	t.Helper()
	if response.Status != http.StatusCreated {
		t.Fatalf("grant status = %d, want %d: %s", response.Status, http.StatusCreated, response.Body)
	}
	var granted struct {
		Data roleAssignment `json:"data"`
	}
	response.Decode(t, &granted)
	return granted.Data.ID
}

func listRoles(t *testing.T, h *apitest.Harness, token, userID string) []roleAssignment {
	t.Helper()
	response := h.Do(t, http.MethodGet, "/api/v1/admin/users/"+userID+"/roles", token, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("list roles status = %d, want %d: %s", response.Status, http.StatusOK, response.Body)
	}
	var list struct {
		Data []roleAssignment `json:"data"`
	}
	response.Decode(t, &list)
	return list.Data
}

func meRoles(t *testing.T, h *apitest.Harness, token string) []string {
	t.Helper()
	response := h.Do(t, http.MethodGet, "/api/v1/me", token, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("me status = %d, want %d: %s", response.Status, http.StatusOK, response.Body)
	}
	var me struct {
		Data struct {
			Roles []string `json:"roles"`
		} `json:"data"`
	}
	response.Decode(t, &me)
	return me.Data.Roles
}

func refresh(h *apitest.Harness, cookie *http.Cookie) apitest.Response {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	request.AddCookie(cookie)
	return h.Send(request)
}

func auditCount(t *testing.T, h *apitest.Harness, action, resourceID string) int {
	t.Helper()
	var count int
	if err := h.DB().Raw(`SELECT count(*) FROM audit_logs WHERE action = ? AND resource_id = ?`, action, resourceID).Scan(&count).Error; err != nil {
		t.Fatalf("count audit logs: %v", err)
	}
	return count
}

func TestAdminGrantsADepartmentRoleThatShowsAfterSignIn(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	teacher := h.SeedUser(t, apitest.UserSeed{})

	id := grantedID(t, grantRole(t, h, admin.Token, teacher.ID, map[string]any{
		"role": "faculty", "scope_type": "department", "scope_id": h.DepartmentID(t, "CS"),
	}))

	roles := listRoles(t, h, admin.Token, teacher.ID)
	if len(roles) != 1 || roles[0].ID != id || roles[0].Role != "faculty" || roles[0].State != "active" {
		t.Fatalf("roles = %+v, want one active faculty assignment", roles)
	}
	if roles[0].Department == nil || roles[0].Department.Code != "CS" {
		t.Fatalf("department = %+v, want CS", roles[0].Department)
	}

	h.SignIn(t, teacher)
	if got := meRoles(t, h, h.TokenFor(t, teacher.ID)); !contains(got, "faculty") {
		t.Fatalf("me roles = %v, want faculty", got)
	}
	if auditCount(t, h, "role_granted", teacher.ID) != 1 {
		t.Fatal("granting a role wrote no audit log")
	}
}

func TestPrincipalGrantsGlobalRoleButNotAdmin(t *testing.T) {
	h := apitest.New(t)
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	officer := h.SeedUser(t, apitest.UserSeed{})

	grantedID(t, grantRole(t, h, principal.Token, officer.ID, map[string]any{"role": "placement_officer", "scope_type": "global"}))
	if response := grantRole(t, h, principal.Token, officer.ID, map[string]any{"role": "admin", "scope_type": "global"}); response.Status != http.StatusForbidden {
		t.Fatalf("principal granting admin status = %d, want %d: %s", response.Status, http.StatusForbidden, response.Body)
	}
}

func TestRoleManagementIsForbiddenToOtherRoles(t *testing.T) {
	h := apitest.New(t)
	teacher := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "faculty", DepartmentCode: "CS"}}})
	target := student(t, h, "CS", 2023)

	checks := []apitest.Response{
		grantRole(t, h, teacher.Token, target.ID, map[string]any{"role": "student_coordinator", "scope_type": "department", "scope_id": h.DepartmentID(t, "CS")}),
		h.Do(t, http.MethodGet, "/api/v1/admin/users/"+target.ID+"/roles", teacher.Token, nil),
		h.Do(t, http.MethodDelete, "/api/v1/admin/users/"+target.ID+"/roles/00000000-0000-0000-0000-000000000000", teacher.Token, nil),
	}
	for i, response := range checks {
		if response.Status != http.StatusForbidden {
			t.Errorf("call %d status = %d, want %d: %s", i, response.Status, http.StatusForbidden, response.Body)
		}
	}
}

func TestHODAppointsAndRemovesACoordinatorInTheirDepartment(t *testing.T) {
	h := apitest.New(t)
	hod := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	csStudent := student(t, h, "CS", 2023)

	id := grantedID(t, grantRole(t, h, hod.Token, csStudent.ID, map[string]any{
		"role": "student_coordinator", "scope_type": "department", "scope_id": h.DepartmentID(t, "CS"),
	}))
	roles := listRoles(t, h, hod.Token, csStudent.ID)
	if !hasActiveRole(roles, id) {
		t.Fatalf("roles = %+v, want the coordinator assignment active", roles)
	}

	response := h.Do(t, http.MethodDelete, "/api/v1/admin/users/"+csStudent.ID+"/roles/"+id, hod.Token, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("HOD ending a coordinator status = %d, want %d: %s", response.Status, http.StatusOK, response.Body)
	}
	if auditCount(t, h, "role_ended", csStudent.ID) != 1 {
		t.Fatal("ending a coordinator wrote no audit log")
	}
}

func TestHODManagesOnlyCoordinatorsOfTheirOwnDepartment(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	hod := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	csStudent := student(t, h, "CS", 2023)
	ecStudent := student(t, h, "EC", 2023)
	cs, ec := h.DepartmentID(t, "CS"), h.DepartmentID(t, "EC")
	ecCoordinator := grantedID(t, grantRole(t, h, admin.Token, ecStudent.ID, map[string]any{"role": "student_coordinator", "scope_type": "department", "scope_id": ec}))

	forbidden := map[string]apitest.Response{
		"grant faculty to own student": grantRole(t, h, hod.Token, csStudent.ID, map[string]any{"role": "faculty", "scope_type": "department", "scope_id": cs}),
		"grant HOD":                    grantRole(t, h, hod.Token, csStudent.ID, map[string]any{"role": "hod", "scope_type": "department", "scope_id": cs}),
	}
	for name, response := range forbidden {
		if response.Status != http.StatusForbidden {
			t.Errorf("%s status = %d, want %d: %s", name, response.Status, http.StatusForbidden, response.Body)
		}
	}

	hidden := map[string]apitest.Response{
		"grant to another department's student": grantRole(t, h, hod.Token, ecStudent.ID, map[string]any{"role": "student_coordinator", "scope_type": "department", "scope_id": ec}),
		"list another department's student":     h.Do(t, http.MethodGet, "/api/v1/admin/users/"+ecStudent.ID+"/roles", hod.Token, nil),
		"end another department's coordinator":  h.Do(t, http.MethodDelete, "/api/v1/admin/users/"+ecStudent.ID+"/roles/"+ecCoordinator, hod.Token, nil),
	}
	for name, response := range hidden {
		if response.Status != http.StatusNotFound {
			t.Errorf("%s status = %d, want %d: %s", name, response.Status, http.StatusNotFound, response.Body)
		}
	}
	if roles := listRoles(t, h, admin.Token, ecStudent.ID); !hasActiveRole(roles, ecCoordinator) {
		t.Fatalf("roles = %+v, the EC coordinator should still be active", roles)
	}
}

func TestPrincipalAppointsSeniorStaffButNotCoordinators(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	csStudent := student(t, h, "CS", 2023)
	teacher := h.SeedUser(t, apitest.UserSeed{})
	cs := h.DepartmentID(t, "CS")

	grantedID(t, grantRole(t, h, principal.Token, teacher.ID, map[string]any{"role": "faculty", "scope_type": "department", "scope_id": cs}))
	grantedID(t, grantRole(t, h, principal.Token, teacher.ID, map[string]any{"role": "hod", "scope_type": "department", "scope_id": cs}))
	coordinator := grantedID(t, grantRole(t, h, admin.Token, csStudent.ID, map[string]any{"role": "student_coordinator", "scope_type": "department", "scope_id": cs}))

	forbidden := map[string]apitest.Response{
		"grant coordinator": grantRole(t, h, principal.Token, csStudent.ID, map[string]any{"role": "student_coordinator", "scope_type": "department", "scope_id": cs}),
		"grant principal":   grantRole(t, h, principal.Token, teacher.ID, map[string]any{"role": "principal", "scope_type": "global"}),
		"end coordinator":   h.Do(t, http.MethodDelete, "/api/v1/admin/users/"+csStudent.ID+"/roles/"+coordinator, principal.Token, nil),
	}
	for name, response := range forbidden {
		if response.Status != http.StatusForbidden {
			t.Errorf("%s status = %d, want %d: %s", name, response.Status, http.StatusForbidden, response.Body)
		}
	}
	// The principal still sees every role on a profile.
	if roles := listRoles(t, h, principal.Token, csStudent.ID); !hasActiveRole(roles, coordinator) {
		t.Fatalf("roles = %+v, want the coordinator listed for the principal", roles)
	}
}

func hasActiveRole(roles []roleAssignment, id string) bool {
	for _, role := range roles {
		if role.ID == id && role.State == "active" {
			return true
		}
	}
	return false
}

func TestGrantRejectsBadRolesScopesAndDates(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	target := h.SeedUser(t, apitest.UserSeed{})
	cs := h.DepartmentID(t, "CS")
	past := time.Now().Add(-48 * time.Hour).UTC().Format(time.RFC3339)
	future := time.Now().Add(48 * time.Hour).UTC().Format(time.RFC3339)

	bad := map[string]map[string]any{
		"unknown role":            {"role": "dean", "scope_type": "global"},
		"student is not granted":  {"role": "student", "scope_type": "global"},
		"faculty without dept":    {"role": "faculty", "scope_type": "global"},
		"hod missing scope id":    {"role": "hod", "scope_type": "department"},
		"principal in dept":       {"role": "principal", "scope_type": "department", "scope_id": cs},
		"unknown department":      {"role": "faculty", "scope_type": "department", "scope_id": "00000000-0000-0000-0000-000000000000"},
		"scope id not a uuid":     {"role": "faculty", "scope_type": "department", "scope_id": "CS"},
		"club scope":              {"role": "faculty", "scope_type": "club", "scope_id": cs},
		"ends before it starts":   {"role": "principal", "scope_type": "global", "starts_at": future, "ends_at": time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)},
		"ends in the past":        {"role": "principal", "scope_type": "global", "ends_at": past},
		"starts in the past":      {"role": "principal", "scope_type": "global", "starts_at": past},
		"coordinator not student": {"role": "student_coordinator", "scope_type": "department", "scope_id": cs},
	}
	for name, body := range bad {
		if response := grantRole(t, h, admin.Token, target.ID, body); response.Status != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want %d: %s", name, response.Status, http.StatusBadRequest, response.Body)
		}
	}
	if response := grantRole(t, h, admin.Token, "00000000-0000-0000-0000-000000000000", map[string]any{"role": "principal", "scope_type": "global"}); response.Status != http.StatusNotFound {
		t.Errorf("unknown user status = %d, want %d: %s", response.Status, http.StatusNotFound, response.Body)
	}
}

func TestGrantRejectsDuplicatesAndASecondHOD(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	teacher := h.SeedUser(t, apitest.UserSeed{})
	cs := h.DepartmentID(t, "CS")
	ec := h.DepartmentID(t, "EC")

	grantedID(t, grantRole(t, h, admin.Token, teacher.ID, map[string]any{"role": "faculty", "scope_type": "department", "scope_id": cs}))
	if response := grantRole(t, h, admin.Token, teacher.ID, map[string]any{"role": "faculty", "scope_type": "department", "scope_id": cs}); response.Status != http.StatusConflict {
		t.Errorf("duplicate status = %d, want %d: %s", response.Status, http.StatusConflict, response.Body)
	}
	// The same role in another Department is a different assignment.
	grantedID(t, grantRole(t, h, admin.Token, teacher.ID, map[string]any{"role": "faculty", "scope_type": "department", "scope_id": ec}))

	if response := grantRole(t, h, admin.Token, teacher.ID, map[string]any{"role": "hod", "scope_type": "department", "scope_id": cs}); response.Status != http.StatusConflict {
		t.Errorf("second CS HOD status = %d, want %d: %s", response.Status, http.StatusConflict, response.Body)
	}
	grantedID(t, grantRole(t, h, admin.Token, teacher.ID, map[string]any{"role": "hod", "scope_type": "department", "scope_id": ec}))
}

func TestCoordinatorRoleNeedsAStudentOfThatDepartment(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	csStudent := student(t, h, "CS", 2023)

	if response := grantRole(t, h, admin.Token, csStudent.ID, map[string]any{"role": "student_coordinator", "scope_type": "department", "scope_id": h.DepartmentID(t, "EC")}); response.Status != http.StatusBadRequest {
		t.Errorf("EC coordinator for a CS student status = %d, want %d: %s", response.Status, http.StatusBadRequest, response.Body)
	}
	grantedID(t, grantRole(t, h, admin.Token, csStudent.ID, map[string]any{"role": "student_coordinator", "scope_type": "department", "scope_id": h.DepartmentID(t, "CS")}))
}

func TestFutureRoleIsListedButNotInEffect(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	target := h.SeedUser(t, apitest.UserSeed{})

	grantedID(t, grantRole(t, h, admin.Token, target.ID, map[string]any{
		"role": "principal", "scope_type": "global", "starts_at": time.Now().Add(72 * time.Hour).UTC().Format(time.RFC3339),
	}))
	roles := listRoles(t, h, admin.Token, target.ID)
	if len(roles) != 1 || roles[0].State != "scheduled" {
		t.Fatalf("roles = %+v, want one scheduled assignment", roles)
	}
	if got := meRoles(t, h, target.Token); contains(got, "principal") {
		t.Fatalf("me roles = %v, a future role is already in effect", got)
	}
}

func TestEndingARoleEndsTheUsersSessions(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	teacher := h.SeedUser(t, apitest.UserSeed{})
	id := grantedID(t, grantRole(t, h, admin.Token, teacher.ID, map[string]any{"role": "faculty", "scope_type": "department", "scope_id": h.DepartmentID(t, "CS")}))
	cookie := h.SignIn(t, teacher)

	response := h.Do(t, http.MethodDelete, "/api/v1/admin/users/"+teacher.ID+"/roles/"+id, admin.Token, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("end role status = %d, want %d: %s", response.Status, http.StatusOK, response.Body)
	}
	if response := refresh(h, cookie); response.Status != http.StatusUnauthorized {
		t.Fatalf("refresh after role ended status = %d, want %d: %s", response.Status, http.StatusUnauthorized, response.Body)
	}

	roles := listRoles(t, h, admin.Token, teacher.ID)
	if len(roles) != 1 || roles[0].State != "ended" || roles[0].EndsAt == nil {
		t.Fatalf("roles = %+v, want the assignment kept as ended", roles)
	}
	if got := meRoles(t, h, h.TokenFor(t, teacher.ID)); contains(got, "faculty") {
		t.Fatalf("me roles = %v, ended role is still in effect", got)
	}
	if response := h.Do(t, http.MethodDelete, "/api/v1/admin/users/"+teacher.ID+"/roles/"+id, admin.Token, nil); response.Status != http.StatusConflict {
		t.Errorf("ending twice status = %d, want %d: %s", response.Status, http.StatusConflict, response.Body)
	}
	if auditCount(t, h, "role_ended", teacher.ID) != 1 {
		t.Fatal("ending a role wrote no audit log")
	}
}

func TestEndingARoleChecksTheUserAndTheLastAdmin(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	other := h.SeedUser(t, apitest.UserSeed{})
	adminRoles := listRoles(t, h, admin.Token, admin.ID)

	if response := h.Do(t, http.MethodDelete, "/api/v1/admin/users/"+other.ID+"/roles/"+adminRoles[0].ID, admin.Token, nil); response.Status != http.StatusNotFound {
		t.Errorf("assignment of another user status = %d, want %d: %s", response.Status, http.StatusNotFound, response.Body)
	}
	if response := h.Do(t, http.MethodDelete, "/api/v1/admin/users/"+admin.ID+"/roles/"+adminRoles[0].ID, admin.Token, nil); response.Status != http.StatusConflict {
		t.Errorf("ending the last admin status = %d, want %d: %s", response.Status, http.StatusConflict, response.Body)
	}
}

func TestEndingAnHODRoleClearsTheDepartmentsHOD(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	hod := h.SeedUser(t, apitest.UserSeed{})
	id := grantedID(t, grantRole(t, h, admin.Token, hod.ID, map[string]any{"role": "hod", "scope_type": "department", "scope_id": h.DepartmentID(t, "CS")}))
	if err := h.DB().Exec(`UPDATE departments SET hod_user_id = ? WHERE code = 'CS'`, hod.ID).Error; err != nil {
		t.Fatalf("set HOD: %v", err)
	}

	if response := h.Do(t, http.MethodDelete, "/api/v1/admin/users/"+hod.ID+"/roles/"+id, admin.Token, nil); response.Status != http.StatusOK {
		t.Fatalf("end role status = %d, want %d: %s", response.Status, http.StatusOK, response.Body)
	}
	var hodUserID *string
	if err := h.DB().Raw(`SELECT hod_user_id FROM departments WHERE code = 'CS'`).Scan(&hodUserID).Error; err != nil {
		t.Fatalf("read HOD: %v", err)
	}
	if hodUserID != nil {
		t.Fatalf("department still names %s as HOD", *hodUserID)
	}
}

func TestSuspendingAUserEndsTheirSessions(t *testing.T) {
	h := apitest.New(t)
	admin := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "admin"}}})
	reader := student(t, h, "CS", 2023)
	cookie := h.SignIn(t, reader)

	response := h.Do(t, http.MethodPatch, "/api/v1/admin/users/"+reader.ID+"/status", admin.Token, map[string]string{"status": "suspended", "note": "misuse"})
	if response.Status != http.StatusOK {
		t.Fatalf("suspend status = %d, want %d: %s", response.Status, http.StatusOK, response.Body)
	}
	var active int
	if err := h.DB().Raw(`SELECT count(*) FROM refresh_tokens WHERE user_id = ? AND revoked_at IS NULL`, reader.ID).Scan(&active).Error; err != nil {
		t.Fatalf("count refresh tokens: %v", err)
	}
	if active != 0 {
		t.Fatalf("%d refresh tokens still active after suspension", active)
	}
	if response := refresh(h, cookie); response.Status != http.StatusUnauthorized {
		t.Fatalf("refresh after suspension status = %d, want %d: %s", response.Status, http.StatusUnauthorized, response.Body)
	}
	if auditCount(t, h, "user_status_changed", reader.ID) != 1 {
		t.Fatal("suspension wrote no audit log")
	}
}
