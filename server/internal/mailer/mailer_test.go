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

	m := NewResendMailer("key", "links@example.com", "https://links.example.com")
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

	m := NewResendMailer("key", "links@example.com", "https://links.example.com")
	m.baseURL = server.URL
	if err := m.SendSignInCode("asha@gmail.com", "482913"); err == nil {
		t.Error("a failed send reported success")
	}
}

func TestStaffAddedEmailSaysWhoAddedThemAndHowToSignIn(t *testing.T) {
	var sent sendRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&sent); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	m := NewResendMailer("key", "links@example.com", "https://links.example.com")
	m.baseURL = server.URL

	if err := m.SendStaffAdded("kiran@college.edu", StaffAdded{FullName: "Prof. Kiran <Hegde>", AddedBy: "Nikhil Bhat", Role: "Faculty, Computer Science"}); err != nil {
		t.Fatalf("send: %v", err)
	}
	for _, want := range []string{"Nikhil Bhat", "Faculty, Computer Science", "kiran@college.edu", "https://links.example.com", "or ask for a code by email", "Prof. Kiran &lt;Hegde&gt;"} {
		if !strings.Contains(sent.HTML, want) {
			t.Errorf("faculty email lacks %q", want)
		}
	}

	if err := m.SendStaffAdded("principal@college.edu", StaffAdded{FullName: "Dr. Shalini Rao", AddedBy: "Nikhil Bhat", Role: "Principal", GoogleOnly: true}); err != nil {
		t.Fatalf("send: %v", err)
	}
	if !strings.Contains(sent.HTML, "never with an email code") || strings.Contains(sent.HTML, "or ask for a code") {
		t.Error("the principal's email should say Google only")
	}
}
