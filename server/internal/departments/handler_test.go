package departments

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AbhishekBalija/Links/server/internal/auth"
	"github.com/gin-gonic/gin"
)

func TestHandlerCreateRequiresAdminRole(t *testing.T) {
	service, _, _ := newTestService()
	router := testRouter(service, auth.RolePrincipal)
	body := []byte(`{"code":"AI","name":"Computer Science and Engineering (AI and ML)"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/departments", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("POST status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
}

func TestHandlerCreateReturnsDepartmentForAdmin(t *testing.T) {
	service, _, audit := newTestService()
	router := testRouter(service, auth.RoleAdmin)
	body := []byte(`{"code":"ai","name":"Computer Science and Engineering (AI and ML)"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/departments", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("POST status = %d, want %d; body = %s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}
	var payload struct {
		Data DepartmentResponse `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Data.Code != "AI" {
		t.Fatalf("POST code = %q, want AI", payload.Data.Code)
	}
	if len(audit.logs) != 1 {
		t.Fatalf("POST audit log count = %d, want 1", len(audit.logs))
	}
}

func TestHandlerListReturnsDepartments(t *testing.T) {
	service, repository, _ := newTestService()
	repository.departments["CS"] = &Department{ID: "department-CS", Code: "CS", Name: "Computer Science and Engineering"}
	router := testRouter(service, auth.RoleStudent)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/departments", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d", recorder.Code, http.StatusOK)
	}
}

func testRouter(service *Service, role auth.Role) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	v1 := router.Group("/api/v1")
	v1.Use(func(c *gin.Context) {
		c.Set("actor", &auth.Actor{UserID: "user-1", Roles: []string{string(role)}})
		c.Next()
	})
	NewHandler(service, auth.NewPolicy()).RegisterRoutes(v1)
	return router
}
