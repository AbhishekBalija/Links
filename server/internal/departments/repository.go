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

	var audienceRuleCount int64
	if err := r.db.WithContext(ctx).
		Table("audience_rules").
		Where("department_id = ?", departmentID).
		Count(&audienceRuleCount).Error; err != nil {
		return false, err
	}
	if audienceRuleCount > 0 {
		return true, nil
	}

	var eventCount int64
	if err := r.db.WithContext(ctx).
		Table("events").
		Where("department_id = ?", departmentID).
		Count(&eventCount).Error; err != nil {
		return false, err
	}
	if eventCount > 0 {
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
			"user_id = ? AND role = ? AND scope_type = ? AND scope_id = ? AND starts_at <= now() AND (ends_at IS NULL OR ends_at > now())",
			userID,
			auth.RoleHOD,
			"department",
			departmentID,
		).
		Count(&count).Error
	return count > 0, err
}

// inEffect is a Role assignment that has started and not ended.
const inEffect = `r.starts_at <= now() AND (r.ends_at IS NULL OR r.ends_at > now())`

// ListForAdmin reads every Department with its HOD and counts in one query.
// The counts match the Department page and Home (internal/directory, #213):
// everyone on the lists, signed in or not yet, with the student role and a
// Student identity here, and everyone with a staff role scoped here, once.
func (r *GormRepository) ListForAdmin(ctx context.Context) ([]AdminRow, error) {
	var rows []AdminRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT d.id, d.code, d.name, d.description,
			hod.user_id AS hod_user_id, hod.full_name AS hod_full_name, hod.username AS hod_username,
			(SELECT count(*) FROM student_identities si
				JOIN users u ON u.id = si.user_id
				WHERE si.department_id = d.id AND (u.status = 'active' OR (u.status = 'pending' AND u.is_verified))
				  AND EXISTS (SELECT 1 FROM role_assignments r WHERE r.user_id = u.id AND ` + inEffect + ` AND r.role = 'student')
			) AS students,
			(SELECT count(DISTINCT u.id) FROM users u
				JOIN role_assignments r ON r.user_id = u.id
				WHERE (u.status = 'active' OR (u.status = 'pending' AND u.is_verified)) AND ` + inEffect + `
				  AND r.role IN ('hod', 'placement_officer', 'faculty') AND r.scope_type = 'department' AND r.scope_id = d.id
			) AS staff
		FROM departments d
		LEFT JOIN LATERAL (
			SELECT r.user_id, p.full_name, p.username
			FROM role_assignments r
			JOIN profiles p ON p.user_id = r.user_id
			WHERE r.role = 'hod' AND r.scope_type = 'department' AND r.scope_id = d.id AND ` + inEffect + `
			ORDER BY r.starts_at
			LIMIT 1
		) hod ON true
		ORDER BY d.name`).Scan(&rows).Error
	return rows, err
}

func (r *GormRepository) WithHOD(ctx context.Context) (map[string]bool, error) {
	var ids []string
	err := r.db.WithContext(ctx).Raw(`
		SELECT DISTINCT r.scope_id FROM role_assignments r
		WHERE r.role = 'hod' AND r.scope_type = 'department' AND ` + inEffect).Scan(&ids).Error
	found := make(map[string]bool, len(ids))
	for _, id := range ids {
		found[id] = true
	}
	return found, err
}
