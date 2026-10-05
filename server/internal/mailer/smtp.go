package mailer

// SMTPMailer sends LINKS's emails through any SMTP server: a testing inbox
// such as Mailtrap, which keeps every email instead of delivering it, or a
// real account (Gmail with an app password) for a pilot before the college
// has a verified domain. Production uses Resend.

import (
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// SMTPSettings is the mail server to send through.
type SMTPSettings struct {
	Host     string
	Port     string
	Username string
	Password string
}

type SMTPMailer struct {
	settings  SMTPSettings
	fromEmail string
	signInURL string
}

func NewSMTPMailer(settings SMTPSettings, fromEmail, signInURL string) *SMTPMailer {
	return &SMTPMailer{settings: settings, fromEmail: fromEmail, signInURL: signInURL}
}

func (m *SMTPMailer) SendSignInCode(to, code string) error {
	return m.send(signInCodeEmail(to, code))
}

func (m *SMTPMailer) SendStaffAdded(to string, letter StaffAdded) error {
	return m.send(staffAddedEmail(to, letter, m.signInURL))
}

func (m *SMTPMailer) SendAccessDecision(to string, letter AccessDecision) error {
	return m.send(accessDecisionEmail(to, letter, m.signInURL))
}

// SendEventNotice sends each person their own copy, one after another.
func (m *SMTPMailer) SendEventNotice(to []Recipient, letter EventNotice) error {
	for _, e := range eventNoticeEmails(to, letter, m.signInURL) {
		if err := m.send(e); err != nil {
			return err
		}
	}
	return nil
}

func (m *SMTPMailer) SendApplicationUpdate(to string, letter ApplicationUpdate) error {
	return m.send(applicationUpdateEmail(to, letter, m.signInURL))
}

// send delivers one email. The server upgrades to TLS when it offers it.
func (m *SMTPMailer) send(e email) error {
	auth := smtp.PlainAuth("", m.settings.Username, m.settings.Password, m.settings.Host)
	address := net.JoinHostPort(m.settings.Host, m.settings.Port)
	if err := smtp.SendMail(address, auth, m.fromEmail, []string{e.To}, message(m.fromEmail, e)); err != nil {
		return fmt.Errorf("send email over SMTP: %w", err)
	}
	return nil
}

// message is the email as SMTP carries it: headers, a blank line, the HTML.
func message(from string, e email) []byte {
	headers := []string{
		"From: LINKS <" + from + ">",
		"To: " + e.To,
		"Subject: " + strings.ReplaceAll(e.Subject, "\n", " "),
		"Date: " + time.Now().Format(time.RFC1123Z),
		"MIME-Version: 1.0",
		"Content-Type: text/html; charset=UTF-8",
	}
	return []byte(strings.Join(headers, "\r\n") + "\r\n\r\n" + e.HTML)
}
