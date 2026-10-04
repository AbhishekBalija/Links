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

func TestEventNoticesGoOutAHundredAtATime(t *testing.T) {
	var batches []int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/emails/batch" {
			t.Errorf("path = %s, want /emails/batch", r.URL.Path)
		}
		var sent []sendRequest
		if err := json.NewDecoder(r.Body).Decode(&sent); err != nil {
			t.Errorf("decode batch: %v", err)
		}
		batches = append(batches, len(sent))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	m := NewResendMailer("key", "links@example.com", "https://links.example.com")
	m.baseURL = server.URL

	to := make([]Recipient, 250)
	for i := range to {
		to[i] = Recipient{Email: "student@example.com", FullName: "Asha Rao"}
	}
	if err := m.SendEventNotice(to, EventNotice{Title: "Robotics meetup", Cancelled: true, When: "Fri 9 Oct, 11 am", Where: "CS Lab 2"}); err != nil {
		t.Fatalf("send: %v", err)
	}
	if len(batches) != 3 || batches[0] != 100 || batches[1] != 100 || batches[2] != 50 {
		t.Errorf("batches = %v, want 100, 100 and 50", batches)
	}
}

func TestEveryLetterSaysRepliesArentReadAndGreetsWithoutTitles(t *testing.T) {
	html := accessDecisionHTML(AccessDecision{FullName: "Dr. Meera Iyer", Approved: true, ReviewerName: "Asha Rao", Joined: "a student"}, "https://links.example.com")
	if !strings.Contains(html, "Hello Meera,") || !strings.Contains(html, "Replies to this email aren't read") {
		t.Errorf("letter = %s", html)
	}
}
