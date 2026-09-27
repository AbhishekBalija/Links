package auth

import (
	"strings"
	"testing"
)

func TestParseImportCSVAcceptsExcelExportsAndAnyColumnOrder(t *testing.T) {
	// Excel's "CSV UTF-8" starts with a byte order mark and may use CRLF.
	file := "\ufeffUSN, Email ,full_name\r\n4mn23cs001, asha@gmail.com ,Asha Rao\r\n\r\n4MN23CS002,ravi@gmail.com,\"Kumar, Ravi\"\r\n"
	rows, err := parseImportCSV(strings.NewReader(file))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := []importRow{
		{Line: 2, Email: "asha@gmail.com", FullName: "Asha Rao", USN: "4MN23CS001"},
		{Line: 4, Email: "ravi@gmail.com", FullName: "Kumar, Ravi", USN: "4MN23CS002"},
	}
	if len(rows) != len(want) {
		t.Fatalf("rows = %+v, want %+v", rows, want)
	}
	for i := range want {
		if rows[i] != want[i] {
			t.Errorf("row %d = %+v, want %+v", i, rows[i], want[i])
		}
	}
}

func TestParseImportCSVRejectsDuplicateColumns(t *testing.T) {
	if _, err := parseImportCSV(strings.NewReader("email,email,full_name,usn\n")); err == nil {
		t.Fatal("a header with email twice was accepted")
	}
}
