package opportunities

// Read queries for members: the Opportunity feed and whether one Opportunity
// is visible.

import (
	"context"
	"strings"
)

// eligibilityMatch is the condition that the reader is in Opportunity o's
// Eligibility: no rules means everyone; otherwise one of the reader's
// Memberships must satisfy a rule's role and Department together, and its
// batch year if set, exactly as the Announcement and Event feeds match.
func eligibilityMatch(reader Reader) (string, []any) {
	args := []any{targetTypeOpportunity}
	matchesRule := "FALSE"
	if len(reader.Memberships) > 0 {
		rows := make([]string, 0, len(reader.Memberships))
		for _, membership := range reader.Memberships {
			rows = append(rows, "(?, CAST(? AS uuid))")
			args = append(args, membership.Role, membership.DepartmentID)
		}
		matchesRule = `EXISTS (
			SELECT 1 FROM audience_rules r
			JOIN (VALUES ` + strings.Join(rows, ", ") + `) AS m(role, department_id)
			  ON (r.role IS NULL OR r.role = m.role)
			 AND (r.department_id IS NULL OR r.department_id = m.department_id)
			WHERE r.target_type = ? AND r.target_id = o.id
			  AND (r.batch_year IS NULL OR r.batch_year = ?)
		)`
		args = append(args, targetTypeOpportunity, reader.BatchYear)
	}
	condition := `(NOT EXISTS (SELECT 1 FROM audience_rules r WHERE r.target_type = ? AND r.target_id = o.id) OR ` + matchesRule + `)`
	return condition, args
}

// appliedBy is the condition that the reader has an Application to
// Opportunity o, in any status.
const appliedBy = `EXISTS (SELECT 1 FROM opportunity_applications a WHERE a.opportunity_id = o.id AND a.student_id = ?)`

// Feed returns one page of published or closed Opportunities. Open ones the
// reader is eligible for come soonest deadline first; closed ones (closed
// early or past apply_by) latest deadline first. The applied view is what
// the reader applied to, eligible now or not, latest deadline first.
func (r *GormRepository) Feed(ctx context.Context, readerID string, reader Reader, filter FeedFilter, after *Cursor, limit int) ([]View, error) {
	condition, args := eligibilityMatch(reader)
	if filter.State == StateApplied {
		condition, args = appliedBy, []any{readerID}
	}
	query := `SELECT ` + viewColumns + ` ` + viewFrom + ` WHERE o.status IN ('published', 'closed') AND ` + condition
	order, compare := `o.apply_by, o.id`, `>`
	switch filter.State {
	case StateClosed:
		query += ` AND (o.status = 'closed' OR o.apply_by <= now())`
		order, compare = `o.apply_by DESC, o.id DESC`, `<`
	case StateApplied:
		order, compare = `o.apply_by DESC, o.id DESC`, `<`
	default:
		query += ` AND o.status = 'published' AND o.apply_by > now()`
	}
	if filter.Type != nil {
		query += ` AND o.opportunity_type = ?`
		args = append(args, string(*filter.Type))
	}
	if filter.DepartmentID != nil {
		query += ` AND EXISTS (SELECT 1 FROM audience_rules d WHERE d.target_type = ? AND d.target_id = o.id AND d.department_id = ?)`
		args = append(args, targetTypeOpportunity, *filter.DepartmentID)
	}
	if after != nil {
		query += ` AND (o.apply_by, o.id) ` + compare + ` (?, CAST(? AS uuid))`
		args = append(args, after.At, after.ID)
	}
	query += ` ORDER BY ` + order + ` LIMIT ?`
	args = append(args, limit)
	var views []View
	err := r.db.WithContext(ctx).Raw(query, args...).Scan(&views).Error
	return views, err
}

// VisibleTo reports whether the reader may open the Opportunity: it was
// published (it may since have closed) and they are in its Eligibility or
// applied to it.
func (r *GormRepository) VisibleTo(ctx context.Context, readerID string, reader Reader, id string) (bool, error) {
	condition, args := eligibilityMatch(reader)
	query := `SELECT count(*) FROM opportunities o WHERE o.id = ? AND o.status IN ('published', 'closed') AND (` + condition + ` OR ` + appliedBy + `)`
	args = append(append([]any{id}, args...), readerID)
	var count int64
	err := r.db.WithContext(ctx).Raw(query, args...).Scan(&count).Error
	return count > 0, err
}

// Eligible reports whether the reader is in the published or closed
// Opportunity's Eligibility now.
func (r *GormRepository) Eligible(ctx context.Context, reader Reader, id string) (bool, error) {
	condition, args := eligibilityMatch(reader)
	query := `SELECT count(*) FROM opportunities o WHERE o.id = ? AND o.status IN ('published', 'closed') AND ` + condition
	var count int64
	err := r.db.WithContext(ctx).Raw(query, append([]any{id}, args...)...).Scan(&count).Error
	return count > 0, err
}

// StudentPlacement returns the Department and Batch of the user's Student
// identity, if they have one.
func (r *GormRepository) StudentPlacement(ctx context.Context, userID string) (*string, *int, error) {
	var rows []struct {
		DepartmentID *string `gorm:"column:department_id"`
		BatchYear    *int    `gorm:"column:batch_year"`
	}
	err := r.db.WithContext(ctx).Raw(`SELECT department_id, batch_year FROM student_identities WHERE user_id = ?`, userID).Scan(&rows).Error
	if err != nil || len(rows) == 0 {
		return nil, nil, err
	}
	return rows[0].DepartmentID, rows[0].BatchYear, nil
}

func (r *GormRepository) DepartmentIDByCode(ctx context.Context, code string) (*string, error) {
	var ids []string
	err := r.db.WithContext(ctx).Raw(`SELECT id FROM departments WHERE code = ?`, code).Scan(&ids).Error
	if err != nil || len(ids) == 0 {
		return nil, err
	}
	return &ids[0], nil
}
