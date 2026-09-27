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

// typoThreshold is the lowest trigram word similarity that counts as a match.
// Typos of a name score about 0.4 and up; unrelated names that only share a
// first letter score 0.2.
const typoThreshold = 0.3

// Search finds members whose name, username or headline matches q, best match
// first. A match is the text containing q, a trigram word similarity of at
// least typoThreshold, or (for a one-word q) a name word with the same
// letters, which catches swapped letters. q is matched in lower case.
func (r *GormRepository) Search(ctx context.Context, filter Filter, q string, limit int) ([]Member, error) {
	conditions, filterArgs := filterConditions(filter)
	args := []any{q, "%" + escapeLike(q) + "%", !strings.ContainsAny(q, " \t")}
	args = append(args, filterArgs...)
	args = append(args, typoThreshold, typoThreshold, typoThreshold, limit)
	query := `WITH search AS (SELECT ?::text AS q, ?::text AS pattern, ?::boolean AS single_word)
		SELECT * FROM (
			SELECT ` + memberColumns + `,
				public.word_similarity(s.q, lower(p.full_name)) AS name_score,
				public.word_similarity(s.q, lower(p.username)) AS username_score,
				public.word_similarity(s.q, lower(coalesce(p.headline, ''))) AS headline_score,
				(lower(p.full_name) LIKE s.pattern OR lower(p.username) LIKE s.pattern) AS name_contains,
				lower(coalesce(p.headline, '')) LIKE s.pattern AS headline_contains,
				(s.single_word AND EXISTS (
					SELECT 1 FROM regexp_split_to_table(lower(p.full_name), '[^[:alnum:]]+') AS word
					WHERE length(word) = length(s.q) AND directory_letters(word) = directory_letters(s.q)
				)) AS same_letters
			` + memberFrom + `
			CROSS JOIN search s
			WHERE ` + strings.Join(conditions, " AND ") + `
		) m
		WHERE name_contains OR headline_contains OR same_letters
			OR name_score >= ? OR username_score >= ? OR headline_score >= ?
		ORDER BY GREATEST(name_score, username_score, headline_score * 0.9)
			+ CASE WHEN name_contains THEN 1 WHEN headline_contains THEN 0.5 ELSE 0 END
			+ CASE WHEN same_letters THEN 0.5 ELSE 0 END DESC,
			sort_name, user_id
		LIMIT ?`
	var members []Member
	err := r.db.WithContext(ctx).Raw(query, args...).Scan(&members).Error
	return members, err
}

// escapeLike makes q match literally inside a LIKE pattern.
func escapeLike(q string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(q)
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
