package mailer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strings"
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
	// SendAccessDecision tells someone who asked to join whether they're in.
	SendAccessDecision(to string, letter AccessDecision) error
	// SendEventNotice tells everyone who answered an Event that it was
	// cancelled or that its date, time or place changed.
	SendEventNotice(to []Recipient, letter EventNotice) error
	// SendApplicationUpdate tells an applicant their application moved on.
	SendApplicationUpdate(to string, letter ApplicationUpdate) error
}

// Recipient is one person an email goes to.
type Recipient struct {
	Email    string
	FullName string
}

// AccessDecision is what the "your request" email says.
type AccessDecision struct {
	FullName     string
	Approved     bool
	ReviewerName string
	// Joined reads like "a student of Computer Science, batch 2024".
	Joined string
	// Note is the reviewer's reason, sent with a request not approved.
	Note string
}

// EventNotice is what the "cancelled" or "changed" email says.
type EventNotice struct {
	Title     string
	EventPath string
	Cancelled bool
	// Reason is the organiser's, for a cancelled event.
	Reason        string
	OrganiserName string
	// When and Where are as they are now; WasWhen and WasWhere are set only
	// for what changed.
	When, Where, WasWhen, WasWhere string
}

// ApplicationUpdate is what an applicant is told when their application
// moves on: shortlisted, selected or not taken further.
type ApplicationUpdate struct {
	FullName string
	Status   string
	Title    string
	Company  string
	JobPath  string
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

func (NoopMailer) SendAccessDecision(string, AccessDecision) error { return nil }

func (NoopMailer) SendEventNotice([]Recipient, EventNotice) error { return nil }

func (NoopMailer) SendApplicationUpdate(string, ApplicationUpdate) error { return nil }

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

// batchSize is the most emails the mail service takes in one batch call.
const batchSize = 100

func (m *ResendMailer) SendAccessDecision(to string, letter AccessDecision) error {
	subject := "You're in: your LINKS request was approved"
	if !letter.Approved {
		subject = "Your LINKS request wasn't approved"
	}
	return m.post("/emails", sendRequest{From: m.fromEmail, To: to, Subject: subject, HTML: accessDecisionHTML(letter, m.signInURL)})
}

// SendEventNotice sends the same notice to everyone, a hundred at a time, so
// a large event is told in a few calls instead of one per person.
func (m *ResendMailer) SendEventNotice(to []Recipient, letter EventNotice) error {
	subject := "Changed: " + letter.Title
	if letter.Cancelled {
		subject = "Cancelled: " + letter.Title + ", " + letter.When
	}
	for start := 0; start < len(to); start += batchSize {
		end := min(start+batchSize, len(to))
		batch := make([]sendRequest, 0, end-start)
		for _, recipient := range to[start:end] {
			batch = append(batch, sendRequest{From: m.fromEmail, To: recipient.Email, Subject: subject, HTML: eventNoticeHTML(recipient, letter, m.signInURL)})
		}
		if err := m.post("/emails/batch", batch); err != nil {
			return err
		}
	}
	return nil
}

func (m *ResendMailer) SendApplicationUpdate(to string, letter ApplicationUpdate) error {
	subject := "Update on " + letter.Title + " at " + letter.Company
	switch letter.Status {
	case "shortlisted":
		subject = "Shortlisted: " + letter.Title + " at " + letter.Company
	case "selected":
		subject = "Selected: " + letter.Title + " at " + letter.Company
	}
	return m.post("/emails", sendRequest{From: m.fromEmail, To: to, Subject: subject, HTML: applicationUpdateHTML(letter, m.signInURL)})
}

// firstName is the greeting's name: the first word that isn't a title.
func firstName(fullName string) string {
	for _, word := range strings.Fields(fullName) {
		switch strings.ToLower(strings.TrimSuffix(word, ".")) {
		case "dr", "prof", "mr", "mrs", "ms", "shri", "smt":
			continue
		}
		return word
	}
	return "there"
}

// letterHTML wraps a letter in the shared layout. Every email ends saying
// replies aren't read, since they come from a no-reply address.
func letterHTML(greetingName, body, buttonLabel, buttonURL string) string {
	button := ""
	if buttonLabel != "" {
		button = fmt.Sprintf(`<p><a href="%s" style="display:inline-block;padding:12px 20px;background:#1B1814;color:#F4F0E8;border-radius:8px;text-decoration:none;font-weight:bold">%s</a></p>`,
			html.EscapeString(buttonURL), html.EscapeString(buttonLabel))
	}
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="font-family:sans-serif;padding:24px;max-width:520px">
<p>Hello %s,</p>
%s
%s
<p style="margin-top:24px;font-size:12px;color:#666">Replies to this email aren't read. For help, contact your department office.</p>
</body>
</html>`, html.EscapeString(greetingName), body, button)
}

func accessDecisionHTML(letter AccessDecision, signInURL string) string {
	if letter.Approved {
		body := fmt.Sprintf(`<p>%s approved your request. You are in LINKS as %s.</p><p>Sign in the same way as before.</p>`,
			html.EscapeString(letter.ReviewerName), html.EscapeString(letter.Joined))
		return letterHTML(firstName(letter.FullName), body, "Open LINKS", signInURL)
	}
	body := fmt.Sprintf(`<p>Your request to join LINKS as %s wasn't approved.</p>`, html.EscapeString(letter.Joined))
	if letter.Note != "" {
		body += fmt.Sprintf(`<p style="padding:12px 14px;background:#F4F0E8;border-radius:8px"><span style="font-size:13px;color:#666">%s wrote</span><br>%s</p>`,
			html.EscapeString(letter.ReviewerName), html.EscapeString(letter.Note))
	}
	body += `<p>If you study here, ask your department office to check your details. Once they add you to the class list, sign in again.</p>`
	return letterHTML(firstName(letter.FullName), body, "", "")
}

func eventNoticeHTML(to Recipient, letter EventNotice, signInURL string) string {
	link := strings.TrimRight(signInURL, "/") + letter.EventPath
	if letter.Cancelled {
		body := fmt.Sprintf(`<p><strong>%s</strong> on %s, %s, is cancelled.</p>`,
			html.EscapeString(letter.Title), html.EscapeString(letter.When), html.EscapeString(letter.Where))
		if letter.Reason != "" {
			body += fmt.Sprintf(`<p style="padding:12px 14px;background:#F4F0E8;border-radius:8px"><span style="font-size:13px;color:#666">%s, who runs it, wrote</span><br>%s</p>`,
				html.EscapeString(letter.OrganiserName), html.EscapeString(letter.Reason))
		}
		return letterHTML(firstName(to.FullName), body, "See the event", link)
	}
	rows := ""
	if letter.WasWhen != "" {
		rows += fmt.Sprintf(`<tr><td style="color:#666;padding-right:14px">When</td><td><s style="color:#666">%s</s> <strong>%s</strong></td></tr>`,
			html.EscapeString(letter.WasWhen), html.EscapeString(letter.When))
	}
	if letter.WasWhere != "" {
		rows += fmt.Sprintf(`<tr><td style="color:#666;padding-right:14px">Where</td><td><s style="color:#666">%s</s> <strong>%s</strong></td></tr>`,
			html.EscapeString(letter.WasWhere), html.EscapeString(letter.Where))
	}
	body := fmt.Sprintf(`<p><strong>%s</strong>, which you answered, has changed:</p><table>%s</table><p>Your answer stays as it is. Change it on the event page if you can't make it now.</p>`,
		html.EscapeString(letter.Title), rows)
	return letterHTML(firstName(to.FullName), body, "See the event", link)
}

func applicationUpdateHTML(letter ApplicationUpdate, signInURL string) string {
	job := fmt.Sprintf(`<strong>%s at %s</strong>`, html.EscapeString(letter.Title), html.EscapeString(letter.Company))
	link := strings.TrimRight(signInURL, "/") + letter.JobPath
	switch letter.Status {
	case "shortlisted":
		return letterHTML(firstName(letter.FullName), `<p>The placement office shortlisted you for `+job+`.</p><p>What happens next is on the job page, and the placement office will share the details.</p>`, "See the job", link)
	case "selected":
		return letterHTML(firstName(letter.FullName), `<p>You were selected for `+job+`. Congratulations.</p><p>The placement office will share what happens next.</p>`, "See the job", link)
	}
	return letterHTML(firstName(letter.FullName), `<p>`+html.EscapeString(letter.Company)+` won't take your application for `+job+` further this time.</p><p>Your other applications aren't affected. Open drives you can apply to are in Jobs.</p>`, "See open jobs", strings.TrimRight(signInURL, "/")+"/jobs")
}
