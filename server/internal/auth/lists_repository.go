package auth

import (
	"context"
)

// WaitingForFirstSignIn lists the accounts a class list or a staff invite
// created that nobody has signed into yet, oldest first, with the full count.
// Access requests are left out: a person asked for those, nobody added them.
// Unless anywhere, only Students and staff of the given Departments appear.
func (r *GormUserRepository) WaitingForFirstSignIn(ctx context.Context, anywhere bool, departmentIDs []string, limit int) (WaitingList, error) {
	var rows []struct {
		WaitingPerson
		Total int `gorm:"column:total"`
	}
	scope := ""
	args := []any{}
	if !anywhere {
		scope = " AND COALESCE(si.department_id, ra.scope_id) IN ?"
		args = append(args, departmentIDs)
	}
	args = append(args, limit)
	err := r.db.WithContext(ctx).Raw(`
		SELECT u.id AS user_id, p.full_name, u.email, u.created_at AS added_at,
		       CASE WHEN si.user_id IS NOT NULL THEN 'student' ELSE 'staff' END AS kind,
		       COALESCE(ra.role, '') AS role,
		       COALESCE(si.usn, '') AS usn,
		       COALESCE(si.batch_year, 0) AS batch_year,
		       COALESCE(d.code, '') AS department_code,
		       count(*) OVER () AS total
		FROM users u
		JOIN profiles p ON p.user_id = u.id
		LEFT JOIN student_identities si ON si.user_id = u.id
		LEFT JOIN LATERAL (
			SELECT r.role, r.scope_id FROM role_assignments r
			WHERE r.user_id = u.id AND si.user_id IS NULL
			  AND r.starts_at <= now() AND (r.ends_at IS NULL OR r.ends_at > now())
			ORDER BY r.created_at, r.id LIMIT 1
		) ra ON true
		LEFT JOIN departments d ON d.id = COALESCE(si.department_id, ra.scope_id)
		WHERE u.status = 'pending' AND u.is_verified AND u.created_by IS NOT NULL`+scope+`
		ORDER BY u.created_at, u.id
		LIMIT ?`, args...,
	).Scan(&rows).Error
	if err != nil {
		return WaitingList{}, err
	}
	list := WaitingList{People: make([]WaitingPerson, 0, len(rows))}
	for _, row := range rows {
		list.Total = row.Total
		list.People = append(list.People, row.WaitingPerson)
	}
	return list, nil
}

// RecentImportAudits returns the latest students_imported audit rows, newest
// first, and the codes of the given Departments. Unless anywhere, only
// imports that created students in those Departments are returned.
func (r *GormUserRepository) RecentImportAudits(ctx context.Context, anywhere bool, departmentIDs []string, limit int) ([]ImportAuditRow, map[string]bool, error) {
	codes := map[string]bool{}
	if !anywhere {
		var found []string
		if err := r.db.WithContext(ctx).Raw(`SELECT code FROM departments WHERE id IN ?`, departmentIDs).Scan(&found).Error; err != nil {
			return nil, nil, err
		}
		for _, code := range found {
			codes[code] = true
		}
	}
	scope := ""
	args := []any{}
	if !anywhere {
		scope = ` AND EXISTS (
			SELECT 1 FROM jsonb_array_elements(COALESCE(a.metadata->'batches', '[]'::jsonb)) b
			WHERE b->>'department_code' IN (SELECT code FROM departments WHERE id IN ?))`
		args = append(args, departmentIDs)
	}
	args = append(args, limit)
	var audits []ImportAuditRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT a.created_at AS at, COALESCE(p.full_name, '') AS full_name, COALESCE(a.metadata::text, '{}') AS metadata
		FROM audit_logs a
		LEFT JOIN profiles p ON p.user_id = a.actor_id
		WHERE a.action = 'students_imported'`+scope+`
		ORDER BY a.created_at DESC, a.id DESC
		LIMIT ?`, args...,
	).Scan(&audits).Error
	return audits, codes, err
}
