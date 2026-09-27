package integration

import (
	"net/http"
	"net/url"
	"slices"
	"testing"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

// setVisibility turns the caller's Public profile on or off and returns
// their username.
func setVisibility(t *testing.T, h *apitest.Harness, token string, public bool) string {
	t.Helper()
	response := h.Do(t, http.MethodPatch, "/api/v1/me/profile", token, map[string]any{"public_profile_enabled": public})
	if response.Status != http.StatusOK {
		t.Fatalf("update profile status = %d, want %d: %s", response.Status, http.StatusOK, response.Body)
	}
	var body struct {
		Data struct {
			Username             string `json:"username"`
			PublicProfileEnabled bool   `json:"public_profile_enabled"`
		} `json:"data"`
	}
	response.Decode(t, &body)
	if body.Data.PublicProfileEnabled != public {
		t.Fatalf("public_profile_enabled = %v, want %v", body.Data.PublicProfileEnabled, public)
	}
	return body.Data.Username
}

func visibilityAudits(t *testing.T, h *apitest.Harness, userID string) []string {
	t.Helper()
	var changes []string
	if err := h.DB().Raw(`SELECT metadata->>'public_profile_enabled' FROM audit_logs
		WHERE action = 'profile_visibility_changed' AND resource_id = ? AND actor_id = ? ORDER BY created_at`, userID, userID).Scan(&changes).Error; err != nil {
		t.Fatalf("read audit logs: %v", err)
	}
	return changes
}

func TestAMemberMakesTheirProfilePrivateAndPublicAgain(t *testing.T) {
	h := apitest.New(t)
	studentSeed := func(batch int) apitest.UserSeed {
		return apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "student"}}, Student: &apitest.StudentSeed{DepartmentCode: "CS", BatchYear: batch}}
	}
	pia := member(t, h, "Pia Private", studentSeed(2023))
	viewer := member(t, h, "Vic Viewer", studentSeed(2024))
	cs := url.Values{"department": {"CS"}}
	countsBefore := departmentOverview(t, h, viewer.Token, "CS").Counts

	username := setVisibility(t, h, pia.Token, false)

	if names := directoryNames(t, h, viewer.Token, cs); slices.Contains(names, "Pia Private") {
		t.Errorf("directory = %v, a private member is still listed", names)
	}
	profilePath := "/api/v1/profiles/" + username
	expectStatus(t, "another member opens a private profile", h.Do(t, http.MethodGet, profilePath, viewer.Token, nil), http.StatusNotFound)
	expectStatus(t, "a visitor opens a private profile", h.Do(t, http.MethodGet, profilePath, "", nil), http.StatusNotFound)
	expectStatus(t, "the owner opens their own private profile", h.Do(t, http.MethodGet, profilePath, pia.Token, nil), http.StatusOK)
	if countsAfter := departmentOverview(t, h, viewer.Token, "CS").Counts; countsAfter.Students != countsBefore.Students {
		t.Errorf("CS students = %d, want %d: counts still include private members", countsAfter.Students, countsBefore.Students)
	}

	// Saving other fields, or the same value again, changes nothing that's audited.
	expectStatus(t, "edit headline", h.Do(t, http.MethodPatch, "/api/v1/me/profile", pia.Token, map[string]any{"headline": "Robotics"}), http.StatusOK)
	setVisibility(t, h, pia.Token, false)
	if changes := visibilityAudits(t, h, pia.ID); !slices.Equal(changes, []string{"false"}) {
		t.Fatalf("visibility audits = %v, want one change to false", changes)
	}

	setVisibility(t, h, pia.Token, true)
	if names := directoryNames(t, h, viewer.Token, cs); !slices.Contains(names, "Pia Private") {
		t.Errorf("directory = %v, a public member is missing", names)
	}
	expectStatus(t, "another member opens a public profile", h.Do(t, http.MethodGet, profilePath, viewer.Token, nil), http.StatusOK)
	if changes := visibilityAudits(t, h, pia.ID); !slices.Equal(changes, []string{"false", "true"}) {
		t.Errorf("visibility audits = %v, want false then true", changes)
	}
}
