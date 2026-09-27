package directory

import (
	"context"
	"strings"

	"gorm.io/gorm"
)

type GormRepository struct {
	db *gorm.DB
}

func NewGormRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

// inEffect is the condition for a Role assignment that applies now.
const inEffect = `r.starts_at <= now() AND (r.ends_at IS NULL OR r.ends_at > now())`

// listed is who the directory shows: active, verified members with a public
// profile and at least one role in effect. Suspended, pending and rejected
// accounts, hidden profiles and people whose roles have all ended never appear.
const listed = `u.status = 'active' AND u.is_verified AND p.public_profile_enabled
	AND EXISTS (SELECT 1 FROM role_assignments r WHERE r.user_id = u.id AND ` + inEffect + `)`

const memberColumns = `u.id AS user_id, p.username, p.full_name, lower(p.full_name) AS sort_name,
	p.headline, p.avatar_url, p.public_profile_enabled, p.show_email, p.show_phone, u.email, u.phone,
	si.batch_year AS student_batch_year, sd.code AS student_department_code, sd.name AS student_department_name`

const memberFrom = `FROM users u
	JOIN profiles p ON p.user_id = u.id
	LEFT JOIN student_identities si ON si.user_id = u.id
	LEFT JOIN departments sd ON sd.id = si.department_id`

func (r *GormRepository) List(ctx context.Context, filter Filter, after *Cursor, limit int) ([]Member, error) {
	conditions, args := filterConditions(filter)
	if after != nil {
		conditions = append(conditions, `(lower(p.full_name), u.id) > (?, ?)`)
		args = append(args, after.SortName, after.UserID)
	}
	query := `SELECT ` + memberColumns + ` ` + memberFrom + `
		WHERE ` + strings.Join(conditions, " AND ") + `
		ORDER BY lower(p.full_name), u.id
		LIMIT ?`
	args = append(args, limit)
	var members []Member
	err := r.db.WithContext(ctx).Raw(query, args...).Scan(&members).Error
	return members, err
}

// filterConditions turns a Filter into SQL. A member belongs to a Department
// through their Student identity or any Department-scoped role in effect.
func filterConditions(filter Filter) ([]string, []any) {
	conditions := []string{listed}
	var args []any
	if filter.DepartmentID != nil {
		conditions = append(conditions, `(si.department_id = ? OR EXISTS (
			SELECT 1 FROM role_assignments r WHERE r.user_id = u.id AND `+inEffect+`
			AND r.scope_type = 'department' AND r.scope_id = ?))`)
		args = append(args, *filter.DepartmentID, *filter.DepartmentID)
	}
	if filter.Role != nil {
		conditions = append(conditions, `EXISTS (SELECT 1 FROM role_assignments r WHERE r.user_id = u.id AND `+inEffect+` AND r.role = ?)`)
		args = append(args, string(*filter.Role))
	}
	if filter.BatchYear != nil {
		conditions = append(conditions, `si.batch_year = ?`)
		args = append(args, *filter.BatchYear)
	}
	return conditions, args
}

func (r *GormRepository) Grants(ctx context.Context, userIDs []string) ([]Grant, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	var grants []Grant
	err := r.db.WithContext(ctx).Raw(`
		SELECT r.user_id, r.role, d.code AS department_code, d.name AS department_name
		FROM role_assignments r
		LEFT JOIN departments d ON r.scope_type = 'department' AND d.id = r.scope_id
		WHERE r.user_id IN ? AND `+inEffect, userIDs).
		Scan(&grants).Error
	return grants, err
}

func (r *GormRepository) DepartmentByCode(ctx context.Context, code string) (*Department, error) {
	var departments []Department
	err := r.db.WithContext(ctx).Raw(`SELECT id, code, name, description FROM departments WHERE code = ?`, code).Scan(&departments).Error
	if err != nil || len(departments) == 0 {
		return nil, err
	}
	return &departments[0], nil
}
