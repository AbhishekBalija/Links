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
