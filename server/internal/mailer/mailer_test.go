package mailer

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func batch(n int) []ActivationEmail {
	emails := make([]ActivationEmail, n)
	for i := range emails {
		emails[i] = ActivationEmail{To: "s@gmail.com", Name: "S", Link: "https://links.example.com/activate?token=t"}
	}
	return emails
}

func TestResendMailerSendsABatchInOneRequest(t *testing.T) {
	var requests int
	var sent []sendRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != http.MethodPost || r.URL.Path != "/emails/batch" {
			t.Errorf("request = %s %s, want POST /emails/batch", r.Method, r.URL.Path)
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
	if err := m.SendActivationEmails(batch(MaxBatchSize)); err != nil {
		t.Fatalf("send: %v", err)
	}
	if requests != 1 || len(sent) != MaxBatchSize {
		t.Fatalf("requests = %d with %d emails, want 1 with %d", requests, len(sent), MaxBatchSize)
	}
	if sent[0].From != "links@example.com" || sent[0].To != "s@gmail.com" || !strings.Contains(sent[0].HTML, "activate?token=t") {
		t.Errorf("first email = %+v", sent[0])
	}
}

func TestResendMailerReportsAFailedBatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
	}))
	defer server.Close()

	m := NewResendMailer("key", "links@example.com")
	m.baseURL = server.URL
	if err := m.SendActivationEmails(batch(2)); err == nil {
		t.Fatal("send succeeded, want the 422 reported")
	}
}

func TestResendMailerRefusesAnOversizedBatch(t *testing.T) {
	m := NewResendMailer("key", "links@example.com")
	m.baseURL = "http://127.0.0.1:0"
	if err := m.SendActivationEmails(batch(MaxBatchSize + 1)); err == nil {
		t.Fatal("send succeeded, want an oversized batch refused")
	}
}
