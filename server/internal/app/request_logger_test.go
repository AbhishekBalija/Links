package app

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/AbhishekBalija/Links/server/internal/shared/response"
)

func loggedRouter(buffer *bytes.Buffer) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(requestLogger(slog.New(slog.NewJSONHandler(buffer, nil))))
	router.GET("/broken", func(c *gin.Context) {
		response.InternalError(c, errors.New("dial tcp: connection refused"))
	})
	router.GET("/refused", func(c *gin.Context) {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "not yours", nil)
	})
	return router
}

func TestAServerErrorIsLoggedWithItsCause(t *testing.T) {
	var logs bytes.Buffer
	recorder := httptest.NewRecorder()
	loggedRouter(&logs).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/broken", nil))

	if recorder.Code != http.StatusInternalServerError || strings.Contains(recorder.Body.String(), "connection refused") {
		t.Fatalf("response = %d %s, want a plain 500 that hides the cause", recorder.Code, recorder.Body.String())
	}
	out := logs.String()
	if !strings.Contains(out, `"level":"ERROR"`) || !strings.Contains(out, "connection refused") || !strings.Contains(out, `"request_id"`) {
		t.Fatalf("logs = %s, want an error line with the cause and the request ID", out)
	}
}

func TestAClientErrorIsNotLoggedAsAServerError(t *testing.T) {
	var logs bytes.Buffer
	loggedRouter(&logs).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/refused", nil))
	if strings.Contains(logs.String(), `"level":"ERROR"`) {
		t.Fatalf("logs = %s, a 403 is not a server error", logs.String())
	}
}
