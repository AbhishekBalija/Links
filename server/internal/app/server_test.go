package app

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AbhishekBalija/Links/server/pkg/config"
	"github.com/AbhishekBalija/Links/server/pkg/db"
)

func TestHealthEndpoint(t *testing.T) {
	router, err := NewServer(
		config.Config{
			GINMode:          "test",
			RequestBodyLimit: 1024,
			CORS:             config.CORSConfig{AllowedOrigins: []string{"http://localhost:5173"}},
		},
		&db.Database{},
		slog.Default(),
	)
	if err != nil {
		t.Fatalf("create router: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if response.Header().Get(requestIDHeader) == "" {
		t.Fatal("expected request ID response header")
	}
}
