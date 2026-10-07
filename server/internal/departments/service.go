package departments

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/AbhishekBalija/Links/server/internal/auth"
	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
	"github.com/google/uuid"
)

var departmentCodePattern = regexp.MustCompile(`^[A-Z]{2,10}$`)

// newDepartmentCodePattern is stricter than departmentCodePattern: a new
// Department's code is the two letters a USN carries (auth.ValidateUSNFormat),
// so a longer one could never hold a student. Existing longer codes can still
// be read, renamed and deleted.
var newDepartmentCodePattern = regexp.MustCompile(`^[A-Z]{2}$`)

type Service struct {
	repository Repository
	unitOfWork UnitOfWork
}

func NewService(repository Repository, unitOfWork UnitOfWork) *Service {
	return &Service{repository: repository, unitOfWork: unitOfWork}
}

func (s *Service) List(ctx context.Context) (*DepartmentListResponse, error) {
	departments, err := s.repository.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list departments: %w", err)
	}

	response := make([]DepartmentResponse, 0, len(departments))
	for i := range departments {
		response = append(response, toResponse(&departments[i]))
	}
	return &DepartmentListResponse{Departments: response}, nil
}

// ListForAdmin returns every Department with its HOD and counts, for the
// admin's Departments screen.
func (s *Service) ListForAdmin(ctx context.Context) (*AdminDepartmentList, error) {
	rows, err := s.repository.ListForAdmin(ctx)
	if err != nil {
		return nil, fmt.Errorf("list departments: %w", err)
	}
	list := &AdminDepartmentList{Departments: make([]AdminDepartment, 0, len(rows))}
	for _, row := range rows {
		department := AdminDepartment{
			ID:          row.ID,
			Code:        row.Code,
			Name:        row.Name,
			Description: row.Description,
			Students:    row.Students,
			Staff:       row.Staff,
		}
		if row.HODUserID != nil {
			department.HOD = &AdminHOD{UserID: *row.HODUserID, FullName: deref(row.HODFullName), Username: deref(row.HODUsername)}
		}
		list.Departments = append(list.Departments, department)
	}
	return list, nil
}

// ListPublic returns every Department's code and name for the Access request
// form, which is used before anyone has an account.
func (s *Service) ListPublic(ctx context.Context) ([]PublicDepartment, error) {
	departments, err := s.repository.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list departments: %w", err)
	}
	response := make([]PublicDepartment, 0, len(departments))
	for _, department := range departments {
		response = append(response, PublicDepartment{Code: department.Code, Name: department.Name, HasHOD: department.HODUserID != nil})
	}
	return response, nil
}

func (s *Service) GetByCode(ctx context.Context, code string) (*DepartmentResponse, error) {
	department, err := s.repository.FindByCode(ctx, normalizeCode(code))
	if err != nil {
		return nil, fmt.Errorf("find department: %w", err)
	}
	if department == nil {
		return nil, apperrors.NewNotFound("department not found")
	}
	response := toResponse(department)
	return &response, nil
}

func (s *Service) Create(ctx context.Context, actorID string, input CreateDepartmentInput) (*DepartmentResponse, error) {
	code := normalizeCode(input.Code)
	if !newDepartmentCodePattern.MatchString(code) {
		return nil, apperrors.NewValidation("invalid department code", map[string]string{"code": "use the two letters the Department's USNs carry, such as CS"})
	}
	name, description, hodUserID, err := validateDepartmentInput(code, input.Name, input.Description, input.HODUserID)
	if err != nil {
		return nil, err
	}
	if hodUserID != nil {
		return nil, apperrors.NewValidation(
			"invalid HOD assignment",
			map[string]string{"hodUserId": "assign the HOD after creating the department and its scoped role"},
		)
	}

	var created Department
	err = s.unitOfWork.WithinTransaction(ctx, func(repositories Repositories) error {
		existing, findErr := repositories.Departments.FindByCode(ctx, code)
		if findErr != nil {
			return findErr
		}
		if existing != nil {
			return apperrors.NewConflict("department code already exists")
		}
		created = Department{Code: code, Name: name, Description: description, HODUserID: hodUserID}
		if createErr := repositories.Departments.Create(ctx, &created); createErr != nil {
			return createErr
		}
		return repositories.AuditLogs.Create(ctx, &auth.AuditLog{
			ActorID:      &actorID,
			Action:       "department.created",
			ResourceType: "department",
			ResourceID:   &created.ID,
			Metadata:     map[string]interface{}{"code": created.Code, "name": created.Name},
		})
	})
	if err != nil {
		return nil, fmt.Errorf("create department: %w", err)
	}

	response := toResponse(&created)
	return &response, nil
}

func (s *Service) Update(ctx context.Context, actorID, code string, input UpdateDepartmentInput) (*DepartmentResponse, error) {
	normalizedCode := normalizeCode(code)
	if err := refuseCodeChange(normalizedCode, input.Code); err != nil {
		return nil, err
	}
	name, description, hodUserID, err := validateDepartmentInput(normalizedCode, input.Name, input.Description, input.HODUserID)
	if err != nil {
		return nil, err
	}

	var updated *Department
	err = s.unitOfWork.WithinTransaction(ctx, func(repositories Repositories) error {
		department, findErr := repositories.Departments.FindByCode(ctx, normalizedCode)
		if findErr != nil {
			return findErr
		}
		if department == nil {
			return apperrors.NewNotFound("department not found")
		}
		// hodUserId, when given, must name the Department's HOD. The HOD
		// itself comes from the role, so saving never changes it (#179).
		if assignErr := validateHODAssignment(ctx, repositories.Departments, hodUserID, department.ID); assignErr != nil {
			return assignErr
		}

		department.Name = name
		department.Description = description
		if updateErr := repositories.Departments.Update(ctx, department); updateErr != nil {
			return updateErr
		}
		updated = department
		return repositories.AuditLogs.Create(ctx, &auth.AuditLog{
			ActorID:      &actorID,
			Action:       "department.updated",
			ResourceType: "department",
			ResourceID:   &department.ID,
			Metadata:     map[string]interface{}{"code": department.Code, "name": department.Name},
		})
	})
	if err != nil {
		return nil, fmt.Errorf("update department: %w", err)
	}

	response := toResponse(updated)
	return &response, nil
}

// Rename changes a Department's name and nothing else, so its description
// and HOD stay as they are.
func (s *Service) Rename(ctx context.Context, actorID, code string, input RenameDepartmentInput) (*DepartmentResponse, error) {
	normalizedCode := normalizeCode(code)
	if err := refuseCodeChange(normalizedCode, input.Code); err != nil {
		return nil, err
	}
	name, _, _, err := validateDepartmentInput(normalizedCode, input.Name, nil, nil)
	if err != nil {
		return nil, err
	}

	var renamed *Department
	err = s.unitOfWork.WithinTransaction(ctx, func(repositories Repositories) error {
		department, findErr := repositories.Departments.FindByCodeForUpdate(ctx, normalizedCode)
		if findErr != nil {
			return findErr
		}
		if department == nil {
			return apperrors.NewNotFound("department not found")
		}
		previous := department.Name
		department.Name = name
		if updateErr := repositories.Departments.Update(ctx, department); updateErr != nil {
			return updateErr
		}
		renamed = department
		return repositories.AuditLogs.Create(ctx, &auth.AuditLog{
			ActorID:      &actorID,
			Action:       "department.updated",
			ResourceType: "department",
			ResourceID:   &department.ID,
			Metadata:     map[string]interface{}{"code": department.Code, "name": department.Name, "previous_name": previous},
		})
	})
	if err != nil {
		return nil, fmt.Errorf("rename department: %w", err)
	}

	response := toResponse(renamed)
	return &response, nil
}

func (s *Service) Delete(ctx context.Context, actorID, code string) error {
	normalizedCode := normalizeCode(code)
	if !departmentCodePattern.MatchString(normalizedCode) {
		return apperrors.NewValidation("invalid department code", map[string]string{"code": "use 2 to 10 uppercase letters"})
	}

	err := s.unitOfWork.WithinTransaction(ctx, func(repositories Repositories) error {
		department, findErr := repositories.Departments.FindByCodeForUpdate(ctx, normalizedCode)
		if findErr != nil {
			return findErr
		}
		if department == nil {
			return apperrors.NewNotFound("department not found")
		}
		referenced, referenceErr := repositories.Departments.IsReferenced(ctx, department.ID)
		if referenceErr != nil {
			return referenceErr
		}
		if referenced {
			return apperrors.NewConflict("department is in use and cannot be deleted")
		}
		if deleteErr := repositories.Departments.Delete(ctx, department); deleteErr != nil {
			return deleteErr
		}
		return repositories.AuditLogs.Create(ctx, &auth.AuditLog{
			ActorID:      &actorID,
			Action:       "department.deleted",
			ResourceType: "department",
			ResourceID:   &department.ID,
			Metadata:     map[string]interface{}{"code": department.Code, "name": department.Name},
		})
	})
	if err != nil {
		return fmt.Errorf("delete department: %w", err)
	}
	return nil
}

// refuseCodeChange refuses a request body whose code differs from the
// Department's: the code is in every USN of the Department, so it never
// changes once created (ADR 0021).
func refuseCodeChange(code string, requested *string) error {
	if requested == nil || normalizeCode(*requested) == code {
		return nil
	}
	return apperrors.NewValidation(
		"department codes never change",
		map[string]string{"code": "a department's code can't change once it is created; add a new department instead"},
	)
}

func validateDepartmentInput(code, rawName string, rawDescription, hodUserID *string) (string, *string, *string, error) {
	if !departmentCodePattern.MatchString(code) {
		return "", nil, nil, apperrors.NewValidation("invalid department code", map[string]string{"code": "use 2 to 10 uppercase letters"})
	}

	name := strings.TrimSpace(rawName)
	if len(name) < 2 || len(name) > 120 {
		return "", nil, nil, apperrors.NewValidation("invalid department name", map[string]string{"name": "use 2 to 120 characters"})
	}

	description := trimOptional(rawDescription)
	if description != nil && len(*description) > 1000 {
		return "", nil, nil, apperrors.NewValidation("invalid department description", map[string]string{"description": "use at most 1000 characters"})
	}

	normalizedHOD := trimOptional(hodUserID)
	if normalizedHOD != nil {
		if _, err := uuid.Parse(*normalizedHOD); err != nil {
			return "", nil, nil, apperrors.NewValidation("invalid HOD user id", map[string]string{"hodUserId": "use a valid UUID"})
		}
	}
	return name, description, normalizedHOD, nil
}

func validateHODAssignment(ctx context.Context, repository Repository, hodUserID *string, departmentID string) error {
	if hodUserID == nil {
		return nil
	}
	canAssign, err := repository.CanAssignHOD(ctx, *hodUserID, departmentID)
	if err != nil {
		return err
	}
	if !canAssign {
		return apperrors.NewValidation("invalid HOD assignment", map[string]string{"hodUserId": "user must have an HOD role scoped to this department"})
	}
	return nil
}

func normalizeCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

func deref(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func trimOptional(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func toResponse(department *Department) DepartmentResponse {
	return DepartmentResponse{
		ID:          department.ID,
		Code:        department.Code,
		Name:        department.Name,
		Description: department.Description,
		HODUserID:   department.HODUserID,
		CreatedAt:   department.CreatedAt,
		UpdatedAt:   department.UpdatedAt,
	}
}
