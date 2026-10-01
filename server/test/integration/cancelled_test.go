package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

// A browser that navigates away cancels its request. That is not a server
// error, so it must not be answered or logged as one.
func TestACancelledRequestIsNotAServerError(t *testing.T) {
	h := apitest.New(t)
	student := studentOf(t, h, "CS", 2023)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/opportunities", nil).WithContext(ctx)
	request.Header.Set("Authorization", "Bearer "+student.Token)
	response := h.Send(request)
	if response.Status != 499 {
		t.Fatalf("cancelled request status = %d, want 499 (client closed request): %s", response.Status, response.Body)
	}

	// The same request still open is answered normally.
	expectStatus(t, "open request", h.Do(t, http.MethodGet, "/api/v1/opportunities", student.Token, nil), http.StatusOK)
}
