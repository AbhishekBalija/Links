package events

import "testing"

func TestSafeCellNeutralisesFormulas(t *testing.T) {
	for input, want := range map[string]string{
		"Asha Rao":        "Asha Rao",
		"=SUM(A1:A2)":     "'=SUM(A1:A2)",
		"+91 90000 00001": "'+91 90000 00001",
		"-1":              "'-1",
		"@cmd":            "'@cmd",
		"":                "",
	} {
		if got := safeCell(input); got != want {
			t.Errorf("safeCell(%q) = %q, want %q", input, got, want)
		}
	}
}
