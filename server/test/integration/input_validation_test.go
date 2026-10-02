package integration

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

// Bad input to a public route gets a plain refusal: never a 500, and never
// a stack trace or other internals in the reply.

func expectCleanRefusal(t *testing.T, name string, response apitest.Response) {
	t.Helper()
	if response.Status < 400 || response.Status >= 500 {
		t.Errorf("%s: status = %d, want a 4xx refusal: %s", name, response.Status, response.Body)
	}
	for _, leak := range []string{"INTERNAL_ERROR", "panic", "stack trace", ".go:"} {
		if strings.Contains(response.Body, leak) {
			t.Errorf("%s: reply leaks %q: %s", name, leak, response.Body)
		}
	}
}

func postRaw(h *apitest.Harness, path, contentType, body string) apitest.Response {
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	request.Header.Set("Content-Type", contentType)
	return h.Send(request)
}

func TestGarbageJSONIsRefusedCleanly(t *testing.T) {
	h := apitest.New(t)
	for _, payload := range []string{`{garbage}`, `"just a string"`, `<html>bonk</html>`, `null`, `[1,2,3]`} {
		expectCleanRefusal(t, payload, postRaw(h, "/api/v1/auth/code", "application/json", payload))
	}
}

func TestAnOversizedBodyIsRefusedCleanly(t *testing.T) {
	h := apitest.New(t)
	body := `{"email":"` + strings.Repeat("a", 2<<20) + `@college.edu"}`
	expectCleanRefusal(t, "oversized body", postRaw(h, "/api/v1/auth/code", "application/json", body))
}

// A JSON body labelled text/plain is still read as JSON; it just mustn't
// break anything.
func TestTheWrongContentTypeDoesNotBreakAnything(t *testing.T) {
	h := apitest.New(t)
	response := postRaw(h, "/api/v1/auth/code", "text/plain", `{"email":"someone@college.edu"}`)
	if response.Status >= 500 || strings.Contains(response.Body, ".go:") || strings.Contains(response.Body, "panic") {
		t.Errorf("text/plain body: status = %d: %s", response.Status, response.Body)
	}
}
