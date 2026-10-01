package config

import "testing"

func TestClientIPHeaderIsTrustedOnlyOnVercel(t *testing.T) {
	t.Setenv("VERCEL", "")
	if got := clientIPHeader(); got != "" {
		t.Errorf("off Vercel, header = %q, want none (it could be forged)", got)
	}
	t.Setenv("VERCEL", "1")
	if got := clientIPHeader(); got != "X-Real-IP" {
		t.Errorf("on Vercel, header = %q, want X-Real-IP", got)
	}
}
