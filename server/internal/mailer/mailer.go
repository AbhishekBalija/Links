package mailer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"time"
)

const resendBaseURL = "https://api.resend.com"

type sendRequest struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	HTML    string `json:"html"`
}

type Mailer interface {
	// SendSignInCode emails a one-time sign-in code.
	SendSignInCode(to string, code string) error
	// SendStaffAdded tells someone they were added to LINKS as staff and how
	// to sign in.
	SendStaffAdded(to string, letter StaffAdded) error
}

// StaffAdded is what the "you were added" email says.
type StaffAdded struct {
	FullName string
	AddedBy  string
	// Role reads like "Faculty, Computer Science and Engineering" or "Principal".
	Role string
	// GoogleOnly is set for the principal and admins, who sign in with
	// Google and never with an email code.
	GoogleOnly bool
}

type ResendMailer struct {
	apiKey    string
	fromEmail string
	// signInURL is where emails send people to sign in: the app's address.
	signInURL string
	baseURL   string
	client    *http.Client
}

func NewResendMailer(apiKey, fromEmail, signInURL string) *ResendMailer {
	return &ResendMailer{
		apiKey:    apiKey,
		fromEmail: fromEmail,
		signInURL: signInURL,
		baseURL:   resendBaseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (m *ResendMailer) SendSignInCode(to, code string) error {
	return m.post("/emails", sendRequest{
		From:    m.fromEmail,
		To:      to,
		Subject: "Your LINKS sign-in code: " + code,
		HTML:    signInCodeHTML(code),
	})
}

func (m *ResendMailer) SendStaffAdded(to string, letter StaffAdded) error {
	return m.post("/emails", sendRequest{
		From:    m.fromEmail,
		To:      to,
		Subject: "You were added to LINKS",
		HTML:    staffAddedHTML(to, letter, m.signInURL),
	})
}

func (m *ResendMailer) post(path string, body any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal email: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, m.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+m.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("send email: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("resend API returned status %d", resp.StatusCode)
	}

	return nil
}

// NoopMailer is a stub for tests — logs instead of sending.
type NoopMailer struct{}

func (NoopMailer) SendSignInCode(_ string, _ string) error {
	return nil
}

func (NoopMailer) SendStaffAdded(_ string, _ StaffAdded) error {
	return nil
}

func signInCodeHTML(code string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="font-family:sans-serif;padding:24px;max-width:480px">
<h2>Your LINKS sign-in code</h2>
<p style="font-size:32px;letter-spacing:6px;font-weight:bold">%s</p>
<p>Type it in the browser where you asked for it. It works once, for 10 minutes.</p>
<p><strong>Never share this code.</strong> LINKS staff will never ask for it.</p>
<p style="margin-top:24px;font-size:12px;color:#666">If you didn't ask for a code, ignore this email. Nobody can sign in without it.</p>
</body>
</html>`, html.EscapeString(code))
}

// staffAddedHTML is the "you were added" email. The principal and admins are
// told Google only, since an email code never works for them.
func staffAddedHTML(to string, letter StaffAdded, signInURL string) string {
	how := "choose <strong>Continue with Google</strong>, or ask for a code by email"
	if letter.GoogleOnly {
		how = "choose <strong>Continue with Google</strong>. You always sign in with Google, never with an email code"
	}
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="font-family:sans-serif;padding:24px;max-width:520px">
<h2>You were added to LINKS</h2>
<p>Hello %s,</p>
<p>%s added you to LINKS, your college's campus hub, as <strong>%s</strong>.</p>
<p>Sign in with this email address, %s: %s.</p>
<p><a href="%s" style="display:inline-block;padding:12px 20px;background:#1B1814;color:#F4F0E8;border-radius:8px;text-decoration:none;font-weight:bold">Open LINKS</a></p>
<p style="margin-top:24px;font-size:12px;color:#666">If you weren't expecting this, you can ignore it. Nobody can sign in as you without your email.</p>
</body>
</html>`,
		html.EscapeString(letter.FullName), html.EscapeString(letter.AddedBy), html.EscapeString(letter.Role),
		html.EscapeString(to), how, html.EscapeString(signInURL))
}
