package departments

import (
	"context"
	"errors"

	"github.com/AbhishekBalija/Links/server/internal/auth"
	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GormUnitOfWork struct {
	db *gorm.DB
}

func NewGormUnitOfWork(db *gorm.DB) *GormUnitOfWork {
	return &GormUnitOfWork{db: db}
}

func (u *GormUnitOfWork) WithinTransaction(ctx context.Context, fn func(Repositories) error) error {
	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(Repositories{
			Departments: NewGormRepository(tx),
			AuditLogs:   auth.NewGormAuditLogRepository(tx),
		})
	})
}

type GormRepository struct {
	db *gorm.DB
}

func NewGormRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) List(ctx context.Context) ([]Department, error) {
	var departments []Department
	err := r.db.WithContext(ctx).Order("name ASC").Find(&departments).Error
	return departments, err
}

func (r *GormRepository) FindByCode(ctx context.Context, code string) (*Department, error) {
	var department Department
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&department).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &department, err
}

// FindByCodeForUpdate locks the department row until the transaction ends.
// Approvals that assign a department-scoped role take a share lock on the same
// row, so a delete and a new scoped role can never both succeed.
func (r *GormRepository) FindByCodeForUpdate(ctx context.Context, code string) (*Department, error) {
	var department Department
	err := r.db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("code = ?", code).
		First(&department).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &department, err
}

func (r *GormRepository) Create(ctx context.Context, department *Department) error {
	if department.ID == "" {
		department.ID = uuid.New().String()
	}
	err := r.db.WithContext(ctx).Create(department).Error
	// Two admins creating the same code at once both pass the service's
	// existence check; the unique constraint catches the second one.
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "departments_code_key" {
		return apperrors.NewConflict("department code already exists")
	}
	return err
}

func (r *GormRepository) Update(ctx context.Context, department *Department) error {
	return r.db.WithContext(ctx).Save(department).Error
}

func (r *GormRepository) Delete(ctx context.Context, department *Department) error {
	return r.db.WithContext(ctx).Delete(department).Error
}

func (r *GormRepository) IsReferenced(ctx context.Context, departmentID string) (bool, error) {
	var studentCount int64
	if err := r.db.WithContext(ctx).
		Table("student_identities").
		Where("department_id = ?", departmentID).
		Count(&studentCount).Error; err != nil {
		return false, err
	}
	if studentCount > 0 {
		return true, nil
	}

	var scopedRoleCount int64
	if err := r.db.WithContext(ctx).
		Table("role_assignments").
		Where("scope_type = ? AND scope_id = ?", "department", departmentID).
		Count(&scopedRoleCount).Error; err != nil {
		return false, err
	}
	return scopedRoleCount > 0, nil
}

func (r *GormRepository) CanAssignHOD(ctx context.Context, userID, departmentID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("role_assignments").
		Where(
			"user_id = ? AND role = ? AND scope_type = ? AND scope_id = ?",
			userID,
			auth.RoleHOD,
			"department",
			departmentID,
		).
		Count(&count).Error
	return count > 0, err
}
