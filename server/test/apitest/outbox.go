package apitest

import (
	"errors"
	"sync"
	"testing"

	"github.com/AbhishekBalija/Links/server/internal/mailer"
)

// SentCode is one sign-in code email the API sent.
type SentCode struct {
	To   string
	Code string
}

// Outbox stands in for the mailer and keeps what the API sent, so tests can
// read a sign-in code the way a person reads their inbox.
type Outbox struct {
	mu         sync.Mutex
	codes      []SentCode
	staffAdded map[string][]mailer.StaffAdded
	failStaff  bool
	decisions  map[string][]mailer.AccessDecision
	notices    []SentNotice
	updates    map[string][]mailer.ApplicationUpdate
}

// SentNotice is one event notice and everyone it went to.
type SentNotice struct {
	To     []string
	Letter mailer.EventNotice
}

func (o *Outbox) SendAccessDecision(to string, letter mailer.AccessDecision) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.decisions == nil {
		o.decisions = map[string][]mailer.AccessDecision{}
	}
	o.decisions[to] = append(o.decisions[to], letter)
	return nil
}

func (o *Outbox) SendEventNotice(to []mailer.Recipient, letter mailer.EventNotice) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	emails := make([]string, len(to))
	for i, recipient := range to {
		emails[i] = recipient.Email
	}
	o.notices = append(o.notices, SentNotice{To: emails, Letter: letter})
	return nil
}

func (o *Outbox) SendApplicationUpdate(to string, letter mailer.ApplicationUpdate) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.updates == nil {
		o.updates = map[string][]mailer.ApplicationUpdate{}
	}
	o.updates[to] = append(o.updates[to], letter)
	return nil
}

// DecisionsTo returns the "your request" emails sent to the address.
func (o *Outbox) DecisionsTo(to string) []mailer.AccessDecision {
	o.mu.Lock()
	defer o.mu.Unlock()
	return append([]mailer.AccessDecision(nil), o.decisions[to]...)
}

// Notices returns every event notice sent, oldest first.
func (o *Outbox) Notices() []SentNotice {
	o.mu.Lock()
	defer o.mu.Unlock()
	return append([]SentNotice(nil), o.notices...)
}

// UpdatesTo returns the application emails sent to the address.
func (o *Outbox) UpdatesTo(to string) []mailer.ApplicationUpdate {
	o.mu.Lock()
	defer o.mu.Unlock()
	return append([]mailer.ApplicationUpdate(nil), o.updates[to]...)
}

// SendStaffAdded keeps the "you were added" email, or fails like a mail
// service that is down once FailStaffAdded was called.
func (o *Outbox) SendStaffAdded(to string, letter mailer.StaffAdded) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.failStaff {
		return errors.New("mail service unavailable")
	}
	if o.staffAdded == nil {
		o.staffAdded = map[string][]mailer.StaffAdded{}
	}
	o.staffAdded[to] = append(o.staffAdded[to], letter)
	return nil
}

// FailStaffAdded makes every later "you were added" email fail.
func (o *Outbox) FailStaffAdded() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.failStaff = true
}

// LastStaffAddedTo returns the newest "you were added" email to the address
// and fails the test if there is none.
func (o *Outbox) LastStaffAddedTo(t *testing.T, to string) mailer.StaffAdded {
	t.Helper()
	o.mu.Lock()
	defer o.mu.Unlock()
	letters := o.staffAdded[to]
	if len(letters) == 0 {
		t.Fatalf("no \"you were added\" email was sent to %s", to)
	}
	return letters[len(letters)-1]
}

var _ mailer.Mailer = (*Outbox)(nil)

func (o *Outbox) SendSignInCode(to, code string) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.codes = append(o.codes, SentCode{To: to, Code: code})
	return nil
}

// CodesTo returns every sign-in code sent to the address, oldest first.
func (o *Outbox) CodesTo(to string) []string {
	o.mu.Lock()
	defer o.mu.Unlock()
	var codes []string
	for _, sent := range o.codes {
		if sent.To == to {
			codes = append(codes, sent.Code)
		}
	}
	return codes
}

// LastCodeTo returns the newest sign-in code sent to the address and fails
// the test if there is none.
func (o *Outbox) LastCodeTo(t *testing.T, to string) string {
	t.Helper()
	codes := o.CodesTo(to)
	if len(codes) == 0 {
		t.Fatalf("no sign-in code was sent to %s", to)
	}
	return codes[len(codes)-1]
}
