package integration

import (
	"net/http"
	"os"
	"testing"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

// signUp requests access through the real endpoint, the way the form does
// (no batch year sent), and returns the new user's ID.
func signUp(t *testing.T, h *apitest.Harness, usn, department string) string {
	t.Helper()
	response := h.Do(t, http.MethodPost, "/api/v1/auth/request-access", "", map[string]any{
		"email":           "signup-" + usn + "@apitest.local",
		"password":        "SignUp123",
		"full_name":       "Signed Up " + usn,
		"usn":             usn,
		"department_code": department,
	})
	if response.Status != http.StatusCreated {
		t.Fatalf("request access status = %d: %s", response.Status, response.Body)
	}
	var created struct {
		Data struct {
			UserID string `json:"user_id"`
		} `json:"data"`
	}
	response.Decode(t, &created)
	return created.Data.UserID
}

func batchYear(t *testing.T, h *apitest.Harness, userID string) int {
	t.Helper()
	var year int
	if err := h.DB().Raw(`SELECT batch_year FROM student_identities WHERE user_id = ?`, userID).Scan(&year).Error; err != nil {
		t.Fatalf("read batch year: %v", err)
	}
	return year
}

func TestSignUpTakesTheBatchFromTheUSN(t *testing.T) {
	h := apitest.New(t)
	userID := signUp(t, h, "4MN23CS101", "CS")
	if got := batchYear(t, h, userID); got != 2023 {
		t.Errorf("batch year = %d, want 2023 from the USN", got)
	}
}

func TestBatchTargetedNoticeReachesAStudentWhoSignedUp(t *testing.T) {
	h := apitest.New(t)
	hod := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "hod", DepartmentCode: "CS"}}})
	userID := signUp(t, h, "4MN23CS102", "CS")
	// Approval and activation are covered elsewhere; here the student is simply let in.
	if err := h.DB().Exec(`UPDATE users SET status = 'active', is_verified = true WHERE id = ?`, userID).Error; err != nil {
		t.Fatalf("activate: %v", err)
	}
	if err := h.DB().Exec(`INSERT INTO role_assignments (user_id, role, scope_type) VALUES (?, 'student', 'global')`, userID).Error; err != nil {
		t.Fatalf("grant student role: %v", err)
	}
	token := h.TokenFor(t, userID, "student")

	mustPublish(t, h, hod.Token, map[string]any{
		"title":    "Batch 2023 lab timings",
		"audience": []map[string]any{{"department_id": h.DepartmentID(t, "CS"), "role": "student", "batch_year": 2023}},
	})
	if !contains(feedTitles(t, h, token), "Batch 2023 lab timings") {
		t.Error("a CS student of batch 2023 who signed up through the form didn't get the batch 2023 notice")
	}
}

func TestBackfillFixesStudentsSavedWithBatchZero(t *testing.T) {
	h := apitest.New(t)
	userID := signUp(t, h, "4MN22EC103", "EC")
	if err := h.DB().Exec(`UPDATE student_identities SET batch_year = 0 WHERE user_id = ?`, userID).Error; err != nil {
		t.Fatalf("simulate old row: %v", err)
	}
	backfill, err := os.ReadFile("../../migrations/015_backfill_batch_year_from_usn.up.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	if err := h.DB().Exec(string(backfill)).Error; err != nil {
		t.Fatalf("run backfill: %v", err)
	}
	if got := batchYear(t, h, userID); got != 2022 {
		t.Errorf("batch year after backfill = %d, want 2022", got)
	}
}
