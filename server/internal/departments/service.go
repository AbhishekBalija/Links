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
		if assignErr := validateHODAssignment(ctx, repositories.Departments, hodUserID, department.ID); assignErr != nil {
			return assignErr
		}

		department.Name = name
		department.Description = description
		department.HODUserID = hodUserID
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

func (s *Service) Delete(ctx context.Context, actorID, code string) error {
	normalizedCode := normalizeCode(code)
	if !departmentCodePattern.MatchString(normalizedCode) {
		return apperrors.NewValidation("invalid department code", map[string]string{"code": "use 2 to 10 uppercase letters"})
	}

	err := s.unitOfWork.WithinTransaction(ctx, func(repositories Repositories) error {
		department, findErr := repositories.Departments.FindByCode(ctx, normalizedCode)
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
