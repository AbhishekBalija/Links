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

// onTheLists is who a count includes (#213): everyone signed in, and everyone
// on a class list or added as staff who hasn't signed in yet (pending and
// verified). Access requests waiting for a decision, and suspended or
// rejected accounts, aren't counted.
const onTheLists = `(u.status = 'active' OR (u.status = 'pending' AND u.is_verified))`

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

// MemberByID reads one member whether or not the directory lists them.
func (r *GormRepository) MemberByID(ctx context.Context, userID string) (*Member, error) {
	var members []Member
	err := r.db.WithContext(ctx).Raw(`SELECT `+memberColumns+` `+memberFrom+` WHERE u.id = ?`, userID).Scan(&members).Error
	if err != nil || len(members) == 0 {
		return nil, err
	}
	return &members[0], nil
}

// Count is how many listed members match the filter.
func (r *GormRepository) Count(ctx context.Context, filter Filter) (int, error) {
	conditions, args := filterConditions(filter)
	var count int
	err := r.db.WithContext(ctx).Raw(`SELECT count(*) `+memberFrom+` WHERE `+strings.Join(conditions, " AND "), args...).
		Scan(&count).Error
	return count, err
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

// Staff returns the listed members holding a staff role scoped to the
// Department, in name order; the service ranks them by role.
func (r *GormRepository) Staff(ctx context.Context, departmentID string) ([]Member, error) {
	var members []Member
	err := r.db.WithContext(ctx).Raw(`SELECT `+memberColumns+` `+memberFrom+`
		WHERE `+listed+` AND EXISTS (
			SELECT 1 FROM role_assignments r WHERE r.user_id = u.id AND `+inEffect+`
			AND r.scope_type = 'department' AND r.scope_id = ? AND r.role IN ?)
		ORDER BY lower(p.full_name), u.id`, departmentID, staffRoles).
		Scan(&members).Error
	return members, err
}

// StudentsByBatch counts the Department's students per Batch, signed in or
// not yet, hidden profiles included: a count reveals no one.
func (r *GormRepository) StudentsByBatch(ctx context.Context, departmentID string) ([]BatchCount, error) {
	var counts []BatchCount
	err := r.db.WithContext(ctx).Raw(`
		SELECT si.batch_year, count(*) AS count
		FROM student_identities si
		JOIN users u ON u.id = si.user_id
		WHERE si.department_id = ? AND `+onTheLists+`
		  AND EXISTS (SELECT 1 FROM role_assignments r WHERE r.user_id = u.id AND `+inEffect+` AND r.role = 'student')
		GROUP BY si.batch_year
		ORDER BY si.batch_year`, departmentID).
		Scan(&counts).Error
	return counts, err
}

// FacultyCount counts the Department's faculty, signed in or not yet, hidden
// profiles included.
func (r *GormRepository) FacultyCount(ctx context.Context, departmentID string) (int, error) {
	var count int
	err := r.db.WithContext(ctx).Raw(`
		SELECT count(DISTINCT u.id)
		FROM users u
		JOIN role_assignments r ON r.user_id = u.id
		WHERE `+onTheLists+` AND `+inEffect+`
		  AND r.role = 'faculty' AND r.scope_type = 'department' AND r.scope_id = ?`, departmentID).
		Scan(&count).Error
	return count, err
}

// StaffCount counts everyone with a staff role in the Department (HOD,
// placement officer, faculty), each once, signed in or not yet.
func (r *GormRepository) StaffCount(ctx context.Context, departmentID string) (int, error) {
	var count int
	err := r.db.WithContext(ctx).Raw(`
		SELECT count(DISTINCT u.id)
		FROM users u
		JOIN role_assignments r ON r.user_id = u.id
		WHERE `+onTheLists+` AND `+inEffect+`
		  AND r.role IN ? AND r.scope_type = 'department' AND r.scope_id = ?`, staffRoles, departmentID).
		Scan(&count).Error
	return count, err
}

// HODHolder finds the Department's HOD role in effect, whatever the holder's
// account state or profile visibility, or nil.
func (r *GormRepository) HODHolder(ctx context.Context, departmentID string) (*HODHolder, error) {
	var holders []HODHolder
	err := r.db.WithContext(ctx).Raw(`
		SELECT u.id AS user_id, p.full_name, p.username, u.status
		FROM role_assignments r
		JOIN users u ON u.id = r.user_id
		JOIN profiles p ON p.user_id = u.id
		WHERE r.role = 'hod' AND r.scope_type = 'department' AND r.scope_id = ? AND `+inEffect+`
		ORDER BY r.starts_at, r.id
		LIMIT 1`, departmentID).Scan(&holders).Error
	if err != nil || len(holders) == 0 {
		return nil, err
	}
	return &holders[0], nil
}
