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

// MaxBatchSize is the most emails Resend's batch endpoint takes in one call.
const MaxBatchSize = 100

// ActivationEmail is one Activation email in a batch.
type ActivationEmail struct {
	To   string
	Name string
	Link string
}

type Mailer interface {
	SendActivationEmail(to string, name string, activationLink string) error
	// SendActivationEmails sends up to MaxBatchSize emails in one request.
	// The batch succeeds or fails as a whole.
	SendActivationEmails(emails []ActivationEmail) error
	// SendSignInCode emails a one-time sign-in code.
	SendSignInCode(to string, code string) error
}

type ResendMailer struct {
	apiKey    string
	fromEmail string
	baseURL   string
	client    *http.Client
}

func NewResendMailer(apiKey, fromEmail string) *ResendMailer {
	return &ResendMailer{
		apiKey:    apiKey,
		fromEmail: fromEmail,
		baseURL:   resendBaseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (m *ResendMailer) SendActivationEmail(to, name, activationLink string) error {
	body := sendRequest{
		From:    m.fromEmail,
		To:      to,
		Subject: "Activate your LINKS account",
		HTML:    activationEmailHTML(name, activationLink),
	}

	return m.post("/emails", body)
}

// SendActivationEmails uses Resend's batch endpoint, which validates the
// whole batch and sends all of it or none of it.
func (m *ResendMailer) SendActivationEmails(emails []ActivationEmail) error {
	if len(emails) > MaxBatchSize {
		return fmt.Errorf("batch of %d emails is over the limit of %d", len(emails), MaxBatchSize)
	}
	body := make([]sendRequest, len(emails))
	for i, email := range emails {
		body[i] = sendRequest{
			From:    m.fromEmail,
			To:      email.To,
			Subject: "Activate your LINKS account",
			HTML:    activationEmailHTML(email.Name, email.Link),
		}
	}
	return m.post("/emails/batch", body)
}

func (m *ResendMailer) SendSignInCode(to, code string) error {
	return m.post("/emails", sendRequest{
		From:    m.fromEmail,
		To:      to,
		Subject: "Your LINKS sign-in code: " + code,
		HTML:    signInCodeHTML(code),
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

func (NoopMailer) SendActivationEmail(_ string, _ string, _ string) error {
	return nil
}

func (NoopMailer) SendActivationEmails(_ []ActivationEmail) error {
	return nil
}

func (NoopMailer) SendSignInCode(_ string, _ string) error {
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

func activationEmailHTML(name, link string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="font-family:sans-serif;padding:24px;max-width:480px">
<h2>Welcome to LINKS, %s</h2>
<p>Click the button below to activate your account and set your password.</p>
<a href="%s" style="display:inline-block;padding:12px 24px;background:#2563eb;color:#fff;text-decoration:none;border-radius:6px">Activate Account</a>
<p style="margin-top:24px;font-size:12px;color:#666">This link expires in 7 days. If you did not request this, ignore this email.</p>
	</body>
	</html>`, html.EscapeString(name), html.EscapeString(link))
}
