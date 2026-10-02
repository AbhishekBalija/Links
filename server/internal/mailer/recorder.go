package mailer

import (
	"strings"
	"sync"
)

// CodeRecorder passes every email on to the real mailer and remembers the
// last sign-in code sent to each address, so the e2e suite can sign in with
// a real code. It is only used while the test sign-in is on (local only).
type CodeRecorder struct {
	Mailer
	mu    sync.Mutex
	codes map[string]string
}

// NewCodeRecorder wraps next.
func NewCodeRecorder(next Mailer) *CodeRecorder {
	return &CodeRecorder{Mailer: next, codes: map[string]string{}}
}

// SendSignInCode records the code, then sends it as usual.
func (r *CodeRecorder) SendSignInCode(to, code string) error {
	r.mu.Lock()
	r.codes[strings.ToLower(strings.TrimSpace(to))] = code
	r.mu.Unlock()
	return r.Mailer.SendSignInCode(to, code)
}

// LastCode returns the last code sent to the address.
func (r *CodeRecorder) LastCode(to string) (string, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	code, ok := r.codes[strings.ToLower(strings.TrimSpace(to))]
	return code, ok
}
