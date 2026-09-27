package events

// Read queries for readers: the Event feed and whether one Event is visible.

import (
	"context"
	"strings"
)

// audienceMatch is the condition that the reader is in Event e's Audience:
// no rules means the whole college; otherwise one of the reader's Memberships
// must satisfy a rule's role and Department together, and its batch year if
// set, exactly as the Announcement feed matches.
func audienceMatch(reader Reader) (string, []any) {
	args := []any{targetTypeEvent}
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
			WHERE r.target_type = ? AND r.target_id = e.id
			  AND (r.batch_year IS NULL OR r.batch_year = ?)
		)`
		args = append(args, targetTypeEvent, reader.BatchYear)
	}
	condition := `(NOT EXISTS (SELECT 1 FROM audience_rules r WHERE r.target_type = ? AND r.target_id = e.id) OR ` + matchesRule + `)`
	return condition, args
}

// Feed returns one page of published Events in the reader's Audience,
// soonest first.
//
// A cancelled Event stays listed, until it ends, for readers who answered
// it, so the people planning to come find out. The going view keeps only
// Events the reader answered going to; the past view lists ended Events,
// most recent first.
func (r *GormRepository) Feed(ctx context.Context, readerID string, reader Reader, filter FeedFilter, after *Cursor, limit int) ([]View, error) {
	condition, args := audienceMatch(reader)
	query := `SELECT ` + viewColumns + ` ` + viewFrom + ` WHERE ` + condition
	past := filter.Show == ShowPast
	if past {
		query += ` AND e.status = 'published' AND e.ends_at <= now()`
	} else {
		query += ` AND (e.status = 'published' OR (e.status = 'cancelled'
			AND EXISTS (SELECT 1 FROM event_rsvps v WHERE v.event_id = e.id AND v.user_id = ?)))`
		args = append(args, readerID)
		if filter.From == nil {
			query += ` AND e.ends_at > now()`
		}
	}
	if filter.Show == ShowGoing {
		query += ` AND EXISTS (SELECT 1 FROM event_rsvps v WHERE v.event_id = e.id AND v.user_id = ? AND v.status = 'going')`
		args = append(args, readerID)
	}
	if filter.From != nil {
		query += ` AND e.starts_at >= ?`
		args = append(args, *filter.From)
	}
	if filter.To != nil {
		query += ` AND e.starts_at <= ?`
		args = append(args, *filter.To)
	}
	if filter.DepartmentID != nil {
		query += ` AND e.department_id = ?`
		args = append(args, *filter.DepartmentID)
	}
	if filter.EventType != nil {
		query += ` AND e.event_type = ?`
		args = append(args, string(*filter.EventType))
	}
	order, compare := `e.starts_at, e.id`, `>`
	if past {
		order, compare = `e.starts_at DESC, e.id DESC`, `<`
	}
	if after != nil {
		query += ` AND (e.starts_at, e.id) ` + compare + ` (?, CAST(? AS uuid))`
		args = append(args, after.At, after.ID)
	}
	query += ` ORDER BY ` + order + ` LIMIT ?`
	args = append(args, limit)
	var views []View
	err := r.db.WithContext(ctx).Raw(query, args...).Scan(&views).Error
	return views, err
}

// VisibleTo reports whether the reader may open the Event as a reader: it was
// published (it may since have been cancelled) and they are in its Audience.
func (r *GormRepository) VisibleTo(ctx context.Context, reader Reader, id string) (bool, error) {
	condition, args := audienceMatch(reader)
	query := `SELECT count(*) FROM events e WHERE e.id = ? AND e.published_at IS NOT NULL
		AND e.status IN ('published', 'cancelled') AND ` + condition
	var count int64
	err := r.db.WithContext(ctx).Raw(query, append([]any{id}, args...)...).Scan(&count).Error
	return count > 0, err
}
