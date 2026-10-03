package auth

// Bulk import: an admin or an HOD uploads a CSV of students
// (the class list), and each valid row becomes a student waiting for their
// first sign-in, with Google or an email code (#17, ADR 0026). No email is
// sent.

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/mail"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
)

const (
	// MaxImportBytes keeps a file with its multipart wrapping under the
	// API's 1 MiB request body limit.
	MaxImportBytes = 1_000_000
	// maxImportRows keeps a whole file inside the API's 30 second
	// WriteTimeout; see ADR 0014's notes for the time per row.
	maxImportRows = 200
	maxNameLength = 200
)

var importColumns = []string{"email", "full_name", "usn"}

// importRow is one data row of the file. Line is its line in the file, which
// is its row number in a spreadsheet (the header is row 1).
type importRow struct {
	Line     int
	Email    string
	FullName string
	USN      string
}

// parseImportCSV reads the whole file. A malformed file, wrong header or too
// many rows fails the whole import, before anything is written.
func parseImportCSV(file io.Reader) ([]importRow, error) {
	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true

	header, err := reader.Read()
	if errors.Is(err, io.EOF) {
		return nil, apperrors.NewValidation("the file is empty", map[string]string{"file": "needs a header row: email,full_name,usn"})
	}
	if err != nil {
		return nil, invalidCSV(err)
	}
	positions, err := importHeader(header)
	if err != nil {
		return nil, err
	}

	var rows []importRow
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, invalidCSV(err)
		}
		if len(rows) == maxImportRows {
			return nil, apperrors.NewValidation("too many rows", map[string]string{"file": fmt.Sprintf("at most %d students per file", maxImportRows)})
		}
		line, _ := reader.FieldPos(0)
		rows = append(rows, importRow{
			Line:     line,
			Email:    strings.TrimSpace(record[positions["email"]]),
			FullName: strings.TrimSpace(record[positions["full_name"]]),
			USN:      strings.ToUpper(strings.TrimSpace(record[positions["usn"]])),
		})
	}
	if len(rows) == 0 {
		return nil, apperrors.NewValidation("the file has no students", map[string]string{"file": "add one row per student under the header"})
	}
	return rows, nil
}

// importHeader maps each expected column to its position. The columns may be
// in any order, but must be exactly these three, so a mistyped column name
// isn't silently ignored.
func importHeader(header []string) (map[string]int, error) {
	positions := map[string]int{}
	for i, name := range header {
		name = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(name, "\ufeff")))
		if _, seen := positions[name]; seen {
			return nil, apperrors.NewValidation("invalid header", map[string]string{"file": "column " + name + " appears twice"})
		}
		positions[name] = i
	}
	for _, column := range importColumns {
		if _, ok := positions[column]; !ok {
			return nil, apperrors.NewValidation("invalid header", map[string]string{"file": "the header must be email,full_name,usn"})
		}
	}
	if len(positions) != len(importColumns) {
		return nil, apperrors.NewValidation("invalid header", map[string]string{"file": "only the columns email, full_name and usn are allowed"})
	}
	return positions, nil
}

func invalidCSV(err error) error {
	return apperrors.NewValidation("the file is not a valid CSV", map[string]string{"file": err.Error()})
}

// importer carries what one import needs across its rows.
type importer struct {
	service     *authService
	actorID     string
	anywhere    bool
	departments map[string]bool
	byCode      map[string]*Department
	emails      map[string]bool
	usns        map[string]bool
	created     map[importKey]int
}

// importKey is a Department and Batch an import created students in.
type importKey struct {
	department string
	batchYear  int
}

func (s *authService) ImportStudents(ctx context.Context, actorID string, file io.Reader) (*ImportResponse, error) {
	rows, err := parseImportCSV(file)
	if err != nil {
		return nil, err
	}
	anywhere, departments, err := s.importScope(ctx, actorID)
	if err != nil {
		return nil, err
	}

	run := &importer{
		service:     s,
		actorID:     actorID,
		anywhere:    anywhere,
		departments: departments,
		byCode:      map[string]*Department{},
		emails:      map[string]bool{},
		usns:        map[string]bool{},
		created:     map[importKey]int{},
	}
	result := &ImportResponse{Rows: make([]ImportRowResult, 0, len(rows))}
	for _, row := range rows {
		outcome := run.importRow(ctx, row)
		if outcome.Status == ImportCreated {
			result.Created++
		} else {
			result.Failed++
		}
		result.Rows = append(result.Rows, outcome)
	}

	now := time.Now()
	summary := &AuditLog{
		ActorID:      &actorID,
		Action:       "students_imported",
		ResourceType: "user_import",
		CreatedAt:    now,
		Metadata: map[string]any{
			"rows": len(rows), "created": result.Created, "failed": result.Failed,
			"batches": run.batches(),
		},
	}
	if err := s.unitOfWork.WithinTransaction(ctx, func(repos AuthRepositories) error {
		return repos.AuditLogs.Create(ctx, summary)
	}); err != nil {
		return nil, fmt.Errorf("audit import: %w", err)
	}
	return result, nil
}

// importScope reads the actor's roles from the database: an admin may import
// anyone, an HOD only students of their Departments (ADR 0029).
func (s *authService) importScope(ctx context.Context, actorID string) (bool, map[string]bool, error) {
	return s.adminOrHODScope(ctx, actorID, "only an admin or an HOD can import students")
}

// adminOrHODScope says where the actor may add people: anywhere for an admin,
// otherwise the Departments they are HOD of. Being the principal doesn't
// count (ADR 0029). Anyone else is refused with the given message.
func (s *authService) adminOrHODScope(ctx context.Context, actorID, refusal string) (bool, map[string]bool, error) {
	grants, err := s.userRepo.GetRoleAssignments(ctx, actorID)
	if err != nil {
		return false, nil, fmt.Errorf("get actor roles: %w", err)
	}
	departments := map[string]bool{}
	for _, grant := range grants {
		switch grant.Role {
		case RoleAdmin:
			return true, nil, nil
		case RoleHOD:
			if grant.ScopeType == ScopeDepartment && grant.ScopeID != nil {
				departments[*grant.ScopeID] = true
			}
		}
	}
	if len(departments) == 0 {
		return false, nil, apperrors.NewForbidden(refusal)
	}
	return false, departments, nil
}

// departmentScope says where the actor may act on students: anywhere for
// the principal and admins, otherwise the Departments they are HOD of.
// Anyone else is refused with the given message.
func (s *authService) departmentScope(ctx context.Context, actorID, refusal string) (bool, map[string]bool, error) {
	grants, err := s.userRepo.GetRoleAssignments(ctx, actorID)
	if err != nil {
		return false, nil, fmt.Errorf("get actor roles: %w", err)
	}
	departments := map[string]bool{}
	for _, grant := range grants {
		switch grant.Role {
		case RoleAdmin, RolePrincipal:
			return true, nil, nil
		case RoleHOD:
			if grant.ScopeType == ScopeDepartment && grant.ScopeID != nil {
				departments[*grant.ScopeID] = true
			}
		}
	}
	if len(departments) == 0 {
		return false, nil, apperrors.NewForbidden(refusal)
	}
	return false, departments, nil
}

func (run *importer) importRow(ctx context.Context, row importRow) ImportRowResult {
	outcome := ImportRowResult{Row: row.Line, Email: row.Email, Status: ImportFailed}
	department, problem := run.check(ctx, row)
	if problem != "" {
		outcome.Error = problem
		return outcome
	}

	userID, problem, err := run.create(ctx, row, department)
	if err != nil {
		outcome.Error = "could not be saved; try this row again"
		return outcome
	}
	if problem != "" {
		outcome.Error = problem
		return outcome
	}
	outcome.Status = ImportCreated
	outcome.UserID = userID
	// The USN was valid or the row would have failed above.
	batchYear, _ := BatchYearFromUSN(row.USN)
	run.created[importKey{department: department.Code, batchYear: batchYear}]++
	return outcome
}

// batches lists how many students the import created in each Department and
// Batch, for the audit row Home reads.
func (run *importer) batches() []ImportBatch {
	batches := make([]ImportBatch, 0, len(run.created))
	for key, count := range run.created {
		batches = append(batches, ImportBatch{DepartmentCode: key.department, BatchYear: key.batchYear, Created: count})
	}
	sort.Slice(batches, func(i, j int) bool {
		if batches[i].DepartmentCode != batches[j].DepartmentCode {
			return batches[i].DepartmentCode < batches[j].DepartmentCode
		}
		return batches[i].BatchYear < batches[j].BatchYear
	})
	return batches
}

// check applies the rules that need no transaction and returns why the row
// can't be imported, or its Department.
func (run *importer) check(ctx context.Context, row importRow) (*Department, string) {
	address, err := mail.ParseAddress(row.Email)
	if row.Email == "" || err != nil || address.Address != row.Email || address.Name != "" {
		return nil, "email is not a valid address"
	}
	if row.FullName == "" {
		return nil, "full_name is empty"
	}
	if len([]rune(row.FullName)) > maxNameLength {
		return nil, fmt.Sprintf("full_name is longer than %d characters", maxNameLength)
	}
	if row.USN == "" {
		return nil, "usn is empty"
	}
	code, err := ValidateUSNFormat(row.USN)
	if err != nil {
		return nil, "invalid USN: " + err.Error()
	}

	department, err := run.department(ctx, code)
	if err != nil {
		return nil, "could not check the department; try this row again"
	}
	if department == nil {
		return nil, "no department has the code " + code
	}
	if !run.anywhere && !run.departments[department.ID] {
		return nil, "you can only import students of your own department"
	}

	email := strings.ToLower(row.Email)
	if run.emails[email] {
		return nil, "the email appears earlier in this file"
	}
	if run.usns[row.USN] {
		return nil, "the USN appears earlier in this file"
	}
	run.emails[email] = true
	run.usns[row.USN] = true
	return department, ""
}

func (run *importer) department(ctx context.Context, code string) (*Department, error) {
	if department, ok := run.byCode[code]; ok {
		return department, nil
	}
	department, err := run.service.userRepo.FindDepartmentByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	run.byCode[code] = department
	return department, nil
}

// create writes one student in its own transaction, so one bad row never
// undoes the others. problem is a row error to report; err is unexpected.
func (run *importer) create(ctx context.Context, row importRow, department *Department) (string, string, error) {
	now := time.Now()
	batchYear, err := BatchYearFromUSN(row.USN)
	if err != nil {
		return "", "invalid USN: " + err.Error(), nil
	}
	user := &User{
		Email: &row.Email,
		// No password: pending and verified means waiting for the first
		// sign-in, which makes the account active.
		Status:     UserStatusPending,
		IsVerified: true,
		CreatedBy:  &run.actorID,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	var problem string
	err = run.service.unitOfWork.WithinTransaction(ctx, func(repos AuthRepositories) error {
		existing, err := repos.Users.FindByEmail(ctx, row.Email)
		if err != nil {
			return fmt.Errorf("find email: %w", err)
		}
		if existing != nil {
			problem = "the email is already registered"
			return errRowRejected
		}
		taken, err := repos.Users.USNExists(ctx, row.USN)
		if err != nil {
			return fmt.Errorf("find USN: %w", err)
		}
		if taken {
			problem = "the USN is already registered"
			return errRowRejected
		}
		exists, err := repos.Users.LockDepartmentForShare(ctx, department.ID)
		if err != nil {
			return fmt.Errorf("lock department: %w", err)
		}
		if !exists {
			problem = "no department has the code " + department.Code
			return errRowRejected
		}

		if err := repos.Users.Create(ctx, user); err != nil {
			return err
		}
		profile := &Profile{UserID: user.ID, FullName: row.FullName, PublicProfileEnabled: true, CreatedAt: now, UpdatedAt: now}
		if err := run.service.createProfileWithRetry(ctx, repos.Users, profile); err != nil {
			return fmt.Errorf("create profile: %w", err)
		}
		identity := &StudentIdentity{UserID: user.ID, USN: row.USN, DepartmentID: department.ID, BatchYear: batchYear, CreatedAt: now, UpdatedAt: now}
		if err := repos.Users.CreateStudentIdentity(ctx, identity); err != nil {
			return err
		}
		role := &RoleAssignment{UserID: user.ID, Role: RoleStudent, ScopeType: ScopeGlobal, AssignedBy: &run.actorID, StartsAt: now, CreatedAt: now}
		if err := repos.Users.CreateRoleAssignment(ctx, role); err != nil {
			return fmt.Errorf("create role assignment: %w", err)
		}
		userID := user.ID
		return repos.AuditLogs.Create(ctx, &AuditLog{
			ActorID:      &run.actorID,
			Action:       "user_imported",
			ResourceType: "user",
			ResourceID:   &userID,
			CreatedAt:    now,
			Metadata:     map[string]string{"usn": row.USN, "department_code": department.Code, "row": fmt.Sprint(row.Line)},
		})
	})
	if errors.Is(err, errRowRejected) {
		return "", problem, nil
	}
	// Another request can take the email or USN between the check and the
	// insert; the unique index then decides.
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		switch pgErr.ConstraintName {
		case "idx_users_email":
			return "", "the email is already registered", nil
		case "idx_student_identities_usn":
			return "", "the USN is already registered", nil
		}
	}
	if err != nil {
		return "", "", err
	}
	return user.ID, "", nil
}

// errRowRejected rolls back a row's transaction for a reason already put in
// the row's result.
var errRowRejected = errors.New("row rejected")
