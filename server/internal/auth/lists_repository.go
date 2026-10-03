package auth

import (
	"context"
)

// WaitingForFirstSignIn lists the accounts a class list or a staff invite
// created that nobody has signed into yet, oldest first, with the full count.
// Access requests are left out: a person asked for those, nobody added them.
// Unless anywhere, only Students and staff of the given Departments appear.
func (r *GormUserRepository) WaitingForFirstSignIn(ctx context.Context, anywhere bool, departmentIDs []string, limit int) (WaitingList, error) {
	return r.NotSignedIn(ctx, NotSignedInFilter{Anywhere: anywhere, DepartmentIDs: departmentIDs, Limit: limit})
}

// waitingFrom is who a class list or a staff invite let in and who hasn't
// signed in yet, with their Department and who added them.
const waitingFrom = `
		FROM users u
		JOIN profiles p ON p.user_id = u.id
		LEFT JOIN profiles adder ON adder.user_id = u.created_by
		LEFT JOIN student_identities si ON si.user_id = u.id
		LEFT JOIN LATERAL (
			SELECT r.role, r.scope_id FROM role_assignments r
			WHERE r.user_id = u.id AND si.user_id IS NULL
			  AND r.starts_at <= now() AND (r.ends_at IS NULL OR r.ends_at > now())
			ORDER BY r.created_at, r.id LIMIT 1
		) ra ON true
		LEFT JOIN departments d ON d.id = COALESCE(si.department_id, ra.scope_id)
		WHERE u.status = 'pending' AND u.is_verified AND u.created_by IS NOT NULL`

// waitingWhere adds the filter's scope, kind and cursor to waitingFrom.
func waitingWhere(filter NotSignedInFilter, withCursor bool) (string, []any) {
	where := ""
	args := []any{}
	if !filter.Anywhere {
		where += " AND COALESCE(si.department_id, ra.scope_id) IN ?"
		args = append(args, filter.DepartmentIDs)
	}
	switch filter.Kind {
	case "student":
		where += " AND si.user_id IS NOT NULL"
	case "staff":
		where += " AND si.user_id IS NULL"
	}
	if withCursor && filter.After != nil {
		where += " AND (u.created_at, u.id) > (?, CAST(? AS uuid))"
		args = append(args, filter.After.At, filter.After.ID)
	}
	return where, args
}

// NotSignedIn lists the people waiting for a first sign-in, oldest first,
// and how many match the filter in all (not just this page).
func (r *GormUserRepository) NotSignedIn(ctx context.Context, filter NotSignedInFilter) (WaitingList, error) {
	var total int64
	where, args := waitingWhere(filter, false)
	if err := r.db.WithContext(ctx).Raw(`SELECT count(*)`+waitingFrom+where, args...).Scan(&total).Error; err != nil {
		return WaitingList{}, err
	}
	var people []WaitingPerson
	where, args = waitingWhere(filter, true)
	args = append(args, filter.Limit)
	err := r.db.WithContext(ctx).Raw(`
		SELECT u.id AS user_id, p.full_name, u.email, u.created_at AS added_at,
		       CASE WHEN si.user_id IS NOT NULL THEN 'student' ELSE 'staff' END AS kind,
		       COALESCE(ra.role, '') AS role,
		       COALESCE(si.usn, '') AS usn,
		       COALESCE(si.batch_year, 0) AS batch_year,
		       COALESCE(d.code, '') AS department_code,
		       COALESCE(adder.full_name, '') AS added_by_name`+waitingFrom+where+`
		ORDER BY u.created_at, u.id
		LIMIT ?`, args...,
	).Scan(&people).Error
	if err != nil {
		return WaitingList{}, err
	}
	for i := range people {
		people[i].AddedBy = AddedBy{FullName: people[i].AddedByName}
	}
	return WaitingList{Total: int(total), People: people}, nil
}

// NotSignedInEmails is every matching email, oldest first, for copying
// into a reminder. The list is bounded by how many a college adds.
func (r *GormUserRepository) NotSignedInEmails(ctx context.Context, filter NotSignedInFilter) ([]string, error) {
	var emails []string
	where, args := waitingWhere(filter, false)
	err := r.db.WithContext(ctx).Raw(`SELECT u.email`+waitingFrom+where+` ORDER BY u.created_at, u.id`, args...).Scan(&emails).Error
	return emails, err
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
