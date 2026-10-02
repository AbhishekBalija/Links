package departments

import (
	"context"
	"errors"
	"testing"

	"github.com/AbhishekBalija/Links/server/internal/auth"
	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
)

type fakeRepository struct {
	departments map[string]*Department
	referenced  map[string]bool
	hodUsers    map[string]bool
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		departments: make(map[string]*Department),
		referenced:  make(map[string]bool),
		hodUsers:    make(map[string]bool),
	}
}

func (r *fakeRepository) List(context.Context) ([]Department, error) {
	departments := make([]Department, 0, len(r.departments))
	for _, department := range r.departments {
		departments = append(departments, *department)
	}
	return departments, nil
}

func (r *fakeRepository) FindByCode(_ context.Context, code string) (*Department, error) {
	department := r.departments[code]
	if department == nil {
		return nil, nil
	}
	copy := *department
	return &copy, nil
}

func (r *fakeRepository) FindByCodeForUpdate(ctx context.Context, code string) (*Department, error) {
	return r.FindByCode(ctx, code)
}

func (r *fakeRepository) Create(_ context.Context, department *Department) error {
	department.ID = "department-" + department.Code
	copy := *department
	r.departments[department.Code] = &copy
	return nil
}

func (r *fakeRepository) Update(_ context.Context, department *Department) error {
	copy := *department
	r.departments[department.Code] = &copy
	return nil
}

func (r *fakeRepository) Delete(_ context.Context, department *Department) error {
	delete(r.departments, department.Code)
	return nil
}

func (r *fakeRepository) IsReferenced(_ context.Context, departmentID string) (bool, error) {
	return r.referenced[departmentID], nil
}

func (r *fakeRepository) CanAssignHOD(_ context.Context, userID, departmentID string) (bool, error) {
	return r.hodUsers[userID+":"+departmentID], nil
}

func (r *fakeRepository) ListForAdmin(context.Context) ([]AdminRow, error) {
	return nil, nil
}

type fakeAuditRepository struct {
	logs []*auth.AuditLog
}

func (r *fakeAuditRepository) Create(_ context.Context, log *auth.AuditLog) error {
	r.logs = append(r.logs, log)
	return nil
}

type fakeUnitOfWork struct {
	repository *fakeRepository
	audit      *fakeAuditRepository
}

func (u *fakeUnitOfWork) WithinTransaction(ctx context.Context, fn func(Repositories) error) error {
	return fn(Repositories{Departments: u.repository, AuditLogs: u.audit})
}

func newTestService() (*Service, *fakeRepository, *fakeAuditRepository) {
	repository := newFakeRepository()
	audit := &fakeAuditRepository{}
	unitOfWork := &fakeUnitOfWork{repository: repository, audit: audit}
	return NewService(repository, unitOfWork), repository, audit
}

func TestServiceCreateNormalizesAndAuditsDepartment(t *testing.T) {
	service, repository, audit := newTestService()
	description := "  Core computing department  "

	created, err := service.Create(context.Background(), "admin-1", CreateDepartmentInput{
		Code:        " cs ",
		Name:        "  Computer Science and Engineering ",
		Description: &description,
	})

	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Code != "CS" || created.Name != "Computer Science and Engineering" {
		t.Fatalf("Create() response = %#v", created)
	}
	if created.Description == nil || *created.Description != "Core computing department" {
		t.Fatalf("Create() description = %#v", created.Description)
	}
	if repository.departments["CS"] == nil {
		t.Fatal("Create() did not persist the department")
	}
	if len(audit.logs) != 1 || audit.logs[0].Action != "department.created" {
		t.Fatalf("Create() audit logs = %#v", audit.logs)
	}
}

func TestServiceCreateRejectsInvalidOrDuplicateCode(t *testing.T) {
	tests := []struct {
		name  string
		input CreateDepartmentInput
		seed  bool
		code  string
	}{
		{
			name:  "invalid code",
			input: CreateDepartmentInput{Code: "C1", Name: "Computer Science"},
			code:  "VALIDATION_ERROR",
		},
		{
			name:  "duplicate code",
			input: CreateDepartmentInput{Code: "CS", Name: "Computer Science"},
			seed:  true,
			code:  "CONFLICT",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service, repository, _ := newTestService()
			if test.seed {
				repository.departments["CS"] = &Department{ID: "department-CS", Code: "CS", Name: "Existing"}
			}
			_, err := service.Create(context.Background(), "admin-1", test.input)
			assertAppErrorCode(t, err, test.code)
		})
	}
}

func TestServiceCreateRequiresHODAssignmentAfterCreation(t *testing.T) {
	service, _, _ := newTestService()
	hodUserID := "a45ae319-0f64-42a3-b2c5-19b25891f861"

	_, err := service.Create(context.Background(), "admin-1", CreateDepartmentInput{
		Code: "AI", Name: "Computer Science and Engineering (AI and ML)", HODUserID: &hodUserID,
	})
	assertAppErrorCode(t, err, "VALIDATION_ERROR")
}

func TestServiceUpdateValidatesDepartmentScopedHODRole(t *testing.T) {
	service, repository, _ := newTestService()
	hodUserID := "a45ae319-0f64-42a3-b2c5-19b25891f861"
	repository.departments["AI"] = &Department{
		ID: "department-AI", Code: "AI", Name: "Computer Science and Engineering (AI and ML)",
	}

	_, err := service.Update(context.Background(), "admin-1", "AI", UpdateDepartmentInput{
		Name: "Computer Science and Engineering (AI and ML)", HODUserID: &hodUserID,
	})
	assertAppErrorCode(t, err, "VALIDATION_ERROR")

	repository.hodUsers[hodUserID+":department-AI"] = true
	updated, err := service.Update(context.Background(), "admin-1", "AI", UpdateDepartmentInput{
		Name: "Computer Science and Engineering (AI and ML)", HODUserID: &hodUserID,
	})
	if err != nil {
		t.Fatalf("Update() with scoped HOD role error = %v", err)
	}
	if updated.HODUserID == nil || *updated.HODUserID != hodUserID {
		t.Fatalf("Update() HODUserID = %#v", updated.HODUserID)
	}
}

func TestServiceUpdateReplacesOptionalFields(t *testing.T) {
	service, repository, audit := newTestService()
	description := "Old description"
	hodUserID := "a45ae319-0f64-42a3-b2c5-19b25891f861"
	repository.departments["EC"] = &Department{
		ID: "department-EC", Code: "EC", Name: "Old name", Description: &description, HODUserID: &hodUserID,
	}

	updated, err := service.Update(context.Background(), "admin-1", " ec ", UpdateDepartmentInput{
		Name: "Electronics and Communication Engineering",
	})

	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Description != nil || updated.HODUserID != nil {
		t.Fatalf("Update() optional fields = %#v", updated)
	}
	if len(audit.logs) != 1 || audit.logs[0].Action != "department.updated" {
		t.Fatalf("Update() audit logs = %#v", audit.logs)
	}
}

func TestServiceDeleteProtectsReferencedDepartment(t *testing.T) {
	service, repository, audit := newTestService()
	repository.departments["CS"] = &Department{ID: "department-CS", Code: "CS", Name: "Computer Science"}
	repository.referenced["department-CS"] = true

	err := service.Delete(context.Background(), "admin-1", "CS")
	assertAppErrorCode(t, err, "CONFLICT")
	if repository.departments["CS"] == nil {
		t.Fatal("Delete() removed a referenced department")
	}
	if len(audit.logs) != 0 {
		t.Fatalf("Delete() audit logs = %#v", audit.logs)
	}
}

func TestServiceDeleteRemovesUnreferencedDepartmentAndAudits(t *testing.T) {
	service, repository, audit := newTestService()
	repository.departments["ME"] = &Department{ID: "department-ME", Code: "ME", Name: "Mechanical Engineering"}

	err := service.Delete(context.Background(), "admin-1", "ME")

	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if repository.departments["ME"] != nil {
		t.Fatal("Delete() kept an unreferenced department")
	}
	if len(audit.logs) != 1 || audit.logs[0].Action != "department.deleted" {
		t.Fatalf("Delete() audit logs = %#v", audit.logs)
	}
}

func assertAppErrorCode(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected %s error, got nil", code)
	}
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("error %v is not an AppError", err)
	}
	if appErr.Code != code {
		t.Fatalf("AppError.Code = %q, want %q", appErr.Code, code)
	}
}
