package mailer

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	SendActivationEmail(to string, name string, activationLink string) error
}

type ResendMailer struct {
	apiKey    string
	fromEmail string
	client    *http.Client
}

func NewResendMailer(apiKey, fromEmail string) *ResendMailer {
	return &ResendMailer{
		apiKey:    apiKey,
		fromEmail: fromEmail,
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

	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal email: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, resendBaseURL+"/emails", bytes.NewReader(payload))
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
</html>`, name, link)
}
