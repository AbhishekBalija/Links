package mailer

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestResendMailerSendsTheSignInCodeInOneRequest(t *testing.T) {
	var requests int
	var sent sendRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != http.MethodPost || r.URL.Path != "/emails" {
			t.Errorf("request = %s %s, want POST /emails", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer key" {
			t.Errorf("Authorization = %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&sent); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	m := NewResendMailer("key", "links@example.com")
	m.baseURL = server.URL
	if err := m.SendSignInCode("asha@gmail.com", "482913"); err != nil {
		t.Fatalf("send: %v", err)
	}
	if requests != 1 || sent.From != "links@example.com" || sent.To != "asha@gmail.com" {
		t.Fatalf("requests = %d, email = %+v", requests, sent)
	}
	if !strings.Contains(sent.Subject, "482913") || !strings.Contains(sent.HTML, "482913") || !strings.Contains(sent.HTML, "Never share this code") {
		t.Errorf("email doesn't carry the code and the warning: %+v", sent)
	}
}

func TestResendMailerReportsAFailedSend(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	m := NewResendMailer("key", "links@example.com")
	m.baseURL = server.URL
	if err := m.SendSignInCode("asha@gmail.com", "482913"); err == nil {
		t.Error("a failed send reported success")
	}
}
