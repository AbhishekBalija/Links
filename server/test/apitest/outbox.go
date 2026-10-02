package apitest

import (
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
	mu    sync.Mutex
	codes []SentCode
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
