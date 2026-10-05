package integration

import (
	"net/http"
	"testing"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

// A student coordinator posts department notices to their own department's
// students; nothing wider reaches another HOD's or the principal's queue
// (#209).
func TestACoordinatorPostsOnlyToTheirOwnStudents(t *testing.T) {
	h := apitest.New(t)
	cs, ec := h.DepartmentID(t, "CS"), h.DepartmentID(t, "EC")
	coordinator := h.SeedUser(t, apitest.UserSeed{
		Roles:   []apitest.RoleSeed{{Role: "student"}, {Role: "student_coordinator", DepartmentCode: "CS"}},
		Student: &apitest.StudentSeed{DepartmentCode: "CS", BatchYear: 2024},
	})
	csStudents := []map[string]any{{"department_id": cs, "role": "student"}}

	refused := []struct {
		name string
		body map[string]any
	}{
		{"an official notice", map[string]any{"title": "Holiday", "category": "official", "audience": csStudents}},
		{"another department", map[string]any{"title": "Fest", "category": "department", "audience": []map[string]any{{"department_id": ec, "role": "student"}}}},
		{"the whole college", map[string]any{"title": "Fest", "category": "department", "audience": []map[string]any{}}},
		{"the department's staff", map[string]any{"title": "Fest", "category": "department", "audience": []map[string]any{{"department_id": cs}}}},
	}
	for _, tt := range refused {
		tt.body["body"] = "Details inside."
		expectStatus(t, tt.name, publish(t, h, coordinator.Token, tt.body), http.StatusBadRequest)
	}

	createdStatus(t, publish(t, h, coordinator.Token, map[string]any{"title": "Tech fest volunteers", "category": "department", "audience": csStudents}))
	createdStatus(t, publish(t, h, coordinator.Token, map[string]any{"title": "Batch 2024 meetup", "category": "department", "audience": []map[string]any{{"department_id": cs, "role": "student", "batch_year": 2024}}}))
}
