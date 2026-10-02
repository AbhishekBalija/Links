package announcements

// Read queries: the feed, the approval queue, the author's list and
// audience reach. Writes, locks and single lookups stay in repository.go.

import (
	"context"
	"strings"
	"time"
)

// visibleQuery selects published, unexpired Announcements whose Audience
// includes the reader. A rule matches when one of the reader's Memberships
// satisfies the rule's role and Department together and the batch year (if
// set) matches; any matching rule is enough; no rules means the whole college.
func visibleQuery(reader Reader) (string, []any) {
	args := []any{targetTypeAnnouncement}
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
		      WHERE r.target_type = ? AND r.target_id = a.id
		        AND (r.batch_year IS NULL OR r.batch_year = ?)
		    )`
		args = append(args, targetTypeAnnouncement, reader.BatchYear)
	}

	query := `
		SELECT a.*, p.full_name AS publisher_name
		FROM announcements a
		JOIN profiles p ON p.user_id = a.publisher_id
		WHERE a.status = 'published'
		  AND (a.expires_at IS NULL OR a.expires_at > now())
		  AND (
		    NOT EXISTS (
		      SELECT 1 FROM audience_rules r
		      WHERE r.target_type = ? AND r.target_id = a.id
		    )
		    OR ` + matchesRule + `
		  )`
	return query, args
}

// Feed returns one page of the reader's visible Announcements, newest first,
// optionally limited to one category.
func (r *GormRepository) Feed(ctx context.Context, reader Reader, category Category, cursor *FeedCursor, limit int) ([]FeedEntry, error) {
	query, args := visibleQuery(reader)
	if category != "" {
		query += ` AND a.category = ?`
		args = append(args, category)
	}
	if cursor != nil {
		query += ` AND (a.published_at, a.id) < (?, CAST(? AS uuid))`
		args = append(args, cursor.PublishedAt, cursor.ID)
	}
	query += ` ORDER BY a.published_at DESC, a.id DESC LIMIT ?`
	args = append(args, limit)

	var entries []FeedEntry
	err := r.db.WithContext(ctx).Raw(query, args...).Scan(&entries).Error
	return entries, err
}

// VisibleTo returns the Announcement if it is in the reader's feed.
func (r *GormRepository) VisibleTo(ctx context.Context, reader Reader, id string) (*FeedEntry, error) {
	query, args := visibleQuery(reader)
	query += ` AND a.id = ?`
	args = append(args, id)
	var entries []FeedEntry
	if err := r.db.WithContext(ctx).Raw(query, args...).Scan(&entries).Error; err != nil || len(entries) == 0 {
		return nil, err
	}
	return &entries[0], nil
}

// QueueSummary counts the pending revisions an approver may act on and when
// the oldest was submitted.
func (r *GormRepository) QueueSummary(ctx context.Context, scope ApproverScope) (int, *time.Time, error) {
	query := `SELECT count(*) AS count, min(submitted_at) AS oldest FROM announcement_revisions
		WHERE status = 'pending' AND submitted_by <> ?`
	args := []any{scope.UserID}
	if !scope.All {
		if len(scope.DepartmentIDs) == 0 {
			return 0, nil, nil
		}
		query += ` AND approver_department_id IN ?`
		args = append(args, scope.DepartmentIDs)
	}
	var row struct {
		Count  int        `gorm:"column:count"`
		Oldest *time.Time `gorm:"column:oldest"`
	}
	err := r.db.WithContext(ctx).Raw(query, args...).Scan(&row).Error
	return row.Count, row.Oldest, err
}

// AuthorCounts counts the author's Announcements by status, and their
// published Announcements with an edit waiting for approval.
func (r *GormRepository) AuthorCounts(ctx context.Context, authorID string) (map[Status]int, int, error) {
	var rows []struct {
		Status Status `gorm:"column:status"`
		Count  int    `gorm:"column:count"`
	}
	if err := r.db.WithContext(ctx).Raw(
		`SELECT status, count(*) AS count FROM announcements WHERE publisher_id = ? GROUP BY status`, authorID,
	).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	counts := map[Status]int{}
	for _, row := range rows {
		counts[row.Status] = row.Count
	}
	var editsWaiting int64
	err := r.db.WithContext(ctx).Raw(`
		SELECT count(*) FROM announcements a
		JOIN announcement_revisions v ON v.announcement_id = a.id AND v.status = 'pending'
		WHERE a.publisher_id = ? AND a.status = 'published'`, authorID,
	).Scan(&editsWaiting).Error
	return counts, int(editsWaiting), err
}

// AuthorWorkRow is an Announcement of the author's whose newest revision was
// sent back or waits: a new Announcement, or an edit to a published one.
type AuthorWorkRow struct {
	ID                   string     `gorm:"column:id"`
	Title                string     `gorm:"column:title"`
	AnnouncementStatus   Status     `gorm:"column:announcement_status"`
	ReviewNote           *string    `gorm:"column:review_note"`
	ReviewedAt           *time.Time `gorm:"column:reviewed_at"`
	SubmittedAt          *time.Time `gorm:"column:submitted_at"`
	ApproverDepartmentID *string    `gorm:"column:approver_department_id"`
	ReviewerName         string     `gorm:"column:reviewer_name"`
}

// AuthorWork returns the author's Announcements whose newest revision has
// the given status, rejected ones newest first and pending ones longest
// waiting first. A published Announcement that expired is left out.
func (r *GormRepository) AuthorWork(ctx context.Context, authorID string, revision RevisionStatus, limit int) ([]AuthorWorkRow, error) {
	otherStatus, order := "rejected", "v.reviewed_at DESC, a.id"
	if revision == RevisionPending {
		otherStatus, order = "pending", "v.submitted_at, a.id"
	}
	var rows []AuthorWorkRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT a.id, a.title, a.status AS announcement_status, v.review_note, v.reviewed_at,
		       v.submitted_at, v.approver_department_id, COALESCE(p.full_name, '') AS reviewer_name
		FROM announcements a
		JOIN LATERAL (
			SELECT * FROM announcement_revisions rv WHERE rv.announcement_id = a.id
			ORDER BY rv.created_at DESC, rv.id DESC LIMIT 1
		) v ON true
		LEFT JOIN profiles p ON p.user_id = v.reviewed_by
		WHERE a.publisher_id = ? AND v.status = ?
		  AND (a.status = ? OR (a.status = 'published' AND `+unexpiredSQL+`))
		ORDER BY `+order+`
		LIMIT ?`, authorID, revision, otherStatus, limit,
	).Scan(&rows).Error
	return rows, err
}

// QueueEntry is a pending revision with its submitter's name.
type QueueEntry struct {
	Revision
	SubmitterName string `gorm:"column:submitter_name"`
	// The Announcement as readers see it now: published means this is an edit.
	AnnouncementStatus Status `gorm:"column:announcement_status"`
	LiveTitle          string `gorm:"column:live_title"`
	LiveBody           string `gorm:"column:live_body"`
}

// Queue returns pending revisions the approver may act on, oldest first.
func (r *GormRepository) Queue(ctx context.Context, scope ApproverScope, after *FeedCursor, limit int) ([]QueueEntry, error) {
	query := `
		SELECT v.*, p.full_name AS submitter_name,
		       a.status AS announcement_status, a.title AS live_title, a.body AS live_body
		FROM announcement_revisions v
		JOIN profiles p ON p.user_id = v.submitted_by
		JOIN announcements a ON a.id = v.announcement_id
		WHERE v.status = 'pending' AND v.submitted_by <> ?`
	args := []any{scope.UserID}
	if !scope.All {
		if len(scope.DepartmentIDs) == 0 {
			return []QueueEntry{}, nil
		}
		query += ` AND v.approver_department_id IN ?`
		args = append(args, scope.DepartmentIDs)
	}
	if after != nil {
		query += ` AND (v.submitted_at, v.id) > (?, CAST(? AS uuid))`
		args = append(args, after.PublishedAt, after.ID)
	}
	query += ` ORDER BY v.submitted_at, v.id LIMIT ?`
	args = append(args, limit)

	var entries []QueueEntry
	err := r.db.WithContext(ctx).Raw(query, args...).Scan(&entries).Error
	return entries, err
}

// Authored returns the author's Announcements in any status, newest first.
// latestRevisionStatus is the status of an Announcement's newest revision,
// which says whether an edit to a published one is waiting or was sent back.
const latestRevisionStatus = `(SELECT rv.status FROM announcement_revisions rv
	WHERE rv.announcement_id = a.id ORDER BY rv.created_at DESC, rv.id DESC LIMIT 1)`

const unexpiredSQL = `(a.expires_at IS NULL OR a.expires_at > now())`

var mineConditions = map[MineFilter]string{
	MineAttention: `(a.status = 'rejected' OR (a.status = 'published' AND ` + unexpiredSQL + ` AND ` + latestRevisionStatus + ` = 'rejected'))`,
	MineDraft:     `a.status = 'draft'`,
	MineWaiting:   `a.status = 'pending'`,
	MineLive:      `(a.status = 'published' AND ` + unexpiredSQL + ` AND ` + latestRevisionStatus + ` IS DISTINCT FROM 'rejected')`,
	MineEnded:     `(a.status = 'withdrawn' OR (a.status = 'published' AND NOT ` + unexpiredSQL + `))`,
}

func (r *GormRepository) Authored(ctx context.Context, authorID string, filter MineFilter, after *FeedCursor, limit int) ([]FeedEntry, error) {
	query := `
		SELECT a.*, p.full_name AS publisher_name
		FROM announcements a
		JOIN profiles p ON p.user_id = a.publisher_id
		WHERE a.publisher_id = ?`
	args := []any{authorID}
	if condition, ok := mineConditions[filter]; ok {
		query += ` AND ` + condition
	}
	if after != nil {
		query += ` AND (a.created_at, a.id) < (?, CAST(? AS uuid))`
		args = append(args, after.PublishedAt, after.ID)
	}
	query += ` ORDER BY a.created_at DESC, a.id DESC LIMIT ?`
	args = append(args, limit)

	var entries []FeedEntry
	err := r.db.WithContext(ctx).Raw(query, args...).Scan(&entries).Error
	return entries, err
}

// Reach counts the active users an Audience would reach, using the same
// matching as the feed: each role pairs with its own Department (its scope,
// or the Student identity's Department) and batch comes from the Student
// identity. An empty Audience is everyone with a current role.
func (r *GormRepository) Reach(ctx context.Context, audience []AudienceRule) (int, error) {
	query := `
		WITH members AS (
		  SELECT ra.user_id, ra.role,
		         CASE WHEN ra.scope_type = 'department' THEN ra.scope_id ELSE si.department_id END AS department_id,
		         si.batch_year
		  FROM role_assignments ra
		  JOIN users u ON u.id = ra.user_id AND u.status = 'active'
		  LEFT JOIN student_identities si ON si.user_id = ra.user_id
		  WHERE ra.starts_at <= now() AND (ra.ends_at IS NULL OR ra.ends_at > now())
		)
		SELECT COUNT(DISTINCT m.user_id) FROM members m`
	args := []any{}
	if len(audience) > 0 {
		rows := make([]string, 0, len(audience))
		for _, rule := range audience {
			rows = append(rows, "(CAST(? AS text), CAST(? AS uuid), CAST(? AS int))")
			var role *string
			if rule.Role != nil {
				name := string(*rule.Role)
				role = &name
			}
			args = append(args, role, rule.DepartmentID, rule.BatchYear)
		}
		query += `
		WHERE EXISTS (
		  SELECT 1 FROM (VALUES ` + strings.Join(rows, ", ") + `) AS r(role, department_id, batch_year)
		  WHERE (r.role IS NULL OR r.role = m.role)
		    AND (r.department_id IS NULL OR r.department_id = m.department_id)
		    AND (r.batch_year IS NULL OR r.batch_year = m.batch_year)
		)`
	}
	var count int
	err := r.db.WithContext(ctx).Raw(query, args...).Scan(&count).Error
	return count, err
}
