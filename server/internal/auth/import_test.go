package auth

import (
	"context"
	"errors"
	"fmt"
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

// importHarness is an auth service whose actor is an admin and whose CS
// Department exists, so every well-formed row is created.
func importHarness(t *testing.T) *authHarness {
	t.Helper()
	h := newAuthHarness(t)
	h.users.getRoleAssignments = func(context.Context, string) ([]RoleAssignment, error) {
		return []RoleAssignment{{Role: RoleAdmin, ScopeType: ScopeGlobal}}, nil
	}
	h.users.findDepartmentByCode = func(_ context.Context, code string) (*Department, error) {
		return &Department{ID: "dept-" + code, Code: code}, nil
	}
	return h
}

func importFile(rows int) string {
	var file strings.Builder
	file.WriteString("email,full_name,usn\n")
	for i := range rows {
		fmt.Fprintf(&file, "s%d@gmail.com,Student %d,4MN24CS%03d\n", i, i, i)
	}
	return file.String()
}

func TestImportSendsActivationEmailsInBatchesAfterCreatingAccounts(t *testing.T) {
	h := importHarness(t)
	// Every account must exist before the first email goes out.
	h.users.create = func(_ context.Context, user *User) error {
		if len(h.mailer.batches) > 0 {
			t.Fatalf("user %s created after an email batch was sent", *user.Email)
		}
		user.ID = "user-" + *user.Email
		h.users.createdUsers = append(h.users.createdUsers, user)
		return nil
	}

	result, err := h.service.ImportStudents(context.Background(), "admin-1", strings.NewReader(importFile(maxImportRows)))
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if result.Created != maxImportRows || result.Failed != 0 {
		t.Fatalf("created %d, failed %d; want %d and 0", result.Created, result.Failed, maxImportRows)
	}
	if len(h.mailer.sent) != 0 {
		t.Errorf("%d emails sent one at a time, want all in batches", len(h.mailer.sent))
	}
	var sizes []int
	for _, batch := range h.mailer.batches {
		sizes = append(sizes, len(batch))
	}
	if fmt.Sprint(sizes) != "[100 100]" {
		t.Errorf("batch sizes = %v, want [100 100]", sizes)
	}
	first := h.mailer.batches[0][0]
	if first.To != "s0@gmail.com" || first.Name != "Student 0" || !strings.HasPrefix(first.Link, "https://links.example.com/activate?token=") {
		t.Errorf("first email = %+v", first)
	}
	for _, row := range result.Rows {
		if row.Error != "" {
			t.Errorf("row %d error = %q, want none", row.Row, row.Error)
		}
	}
}

func TestImportKeepsRowsCreatedWhenTheirEmailBatchFails(t *testing.T) {
	h := importHarness(t)
	h.mailer.batchErr = map[int]error{1: errors.New("resend is down")}

	result, err := h.service.ImportStudents(context.Background(), "admin-1", strings.NewReader(importFile(150)))
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if result.Created != 150 {
		t.Fatalf("created = %d, want 150", result.Created)
	}
	for i, row := range result.Rows {
		inFailedBatch := i >= 100
		if row.Status != ImportCreated {
			t.Errorf("row %d status = %q, want created", row.Row, row.Status)
		}
		if inFailedBatch != (row.Error != "") {
			t.Errorf("row %d error = %q, want a note only for the failed batch", row.Row, row.Error)
		}
	}
	if len(h.activations.markedUsed) != 50 {
		t.Fatalf("invalidated %d tokens, want the 50 of the failed batch", len(h.activations.markedUsed))
	}
	for i, id := range h.activations.markedUsed {
		if want := h.activations.created[100+i].ID; id != want {
			t.Errorf("invalidated token %d = %q, want %q", i, id, want)
		}
	}
}
