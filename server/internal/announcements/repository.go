package announcements

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/AbhishekBalija/Links/server/internal/auth"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const targetTypeAnnouncement = "announcement"

// Membership is one role a reader holds, with the Department it belongs to
// (nil when it belongs to none).
type Membership struct {
	Role         string
	DepartmentID *string
}

// Reader is who is looking at the feed. A rule matches only when one
// Membership satisfies its role and Department together, so a CS student who
// is also EC faculty is not "CS faculty".
type Reader struct {
	Memberships []Membership
	BatchYear   *int
}

// FeedCursor marks the last Announcement of the previous page.
type FeedCursor struct {
	PublishedAt time.Time
	ID          string
}

// FeedEntry is a published Announcement with its publisher's name.
type FeedEntry struct {
	Announcement
	PublisherName string `gorm:"column:publisher_name"`
}

// AudienceRuleView is a stored Audience rule with its Department code for display.
type AudienceRuleView struct {
	TargetID       string  `gorm:"column:target_id"`
	DepartmentID   *string `gorm:"column:department_id"`
	DepartmentCode *string `gorm:"column:department_code"`
	BatchYear      *int    `gorm:"column:batch_year"`
	Role           *string `gorm:"column:role"`
}

type GormUnitOfWork struct {
	db *gorm.DB
}

func NewGormUnitOfWork(db *gorm.DB) *GormUnitOfWork {
	return &GormUnitOfWork{db: db}
}

func (u *GormUnitOfWork) WithinTransaction(ctx context.Context, fn func(Repositories) error) error {
	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(Repositories{
			Announcements: NewGormRepository(tx),
			AuditLogs:     auth.NewGormAuditLogRepository(tx),
		})
	})
}

type GormRepository struct {
	db *gorm.DB
}

func NewGormRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

// Create stores an Announcement and its Audience rules.
func (r *GormRepository) Create(ctx context.Context, announcement *Announcement, audience []AudienceRule) error {
	if announcement.ID == "" {
		announcement.ID = uuid.NewString()
	}
	if err := r.db.WithContext(ctx).Create(announcement).Error; err != nil {
		return err
	}
	return r.insertAudience(ctx, announcement.ID, audience)
}

// ReplaceAudience swaps an Announcement's Audience rules for new ones.
func (r *GormRepository) ReplaceAudience(ctx context.Context, announcementID string, audience []AudienceRule) error {
	err := r.db.WithContext(ctx).Exec(
		`DELETE FROM audience_rules WHERE target_type = ? AND target_id = ?`, targetTypeAnnouncement, announcementID,
	).Error
	if err != nil {
		return err
	}
	return r.insertAudience(ctx, announcementID, audience)
}

func (r *GormRepository) insertAudience(ctx context.Context, announcementID string, audience []AudienceRule) error {
	for _, rule := range audience {
		var role *string
		if rule.Role != nil {
			value := string(*rule.Role)
			role = &value
		}
		err := r.db.WithContext(ctx).Exec(
			`INSERT INTO audience_rules (id, target_type, target_id, department_id, batch_year, role)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			uuid.NewString(), targetTypeAnnouncement, announcementID, rule.DepartmentID, rule.BatchYear, role,
		).Error
		if err != nil {
			return err
		}
	}
	return nil
}

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

// Find loads an Announcement with its publisher's name, without locking.
func (r *GormRepository) Find(ctx context.Context, id string) (*FeedEntry, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, nil
	}
	var entries []FeedEntry
	err := r.db.WithContext(ctx).Raw(`
		SELECT a.*, p.full_name AS publisher_name
		FROM announcements a
		JOIN profiles p ON p.user_id = a.publisher_id
		WHERE a.id = ?`, id,
	).Scan(&entries).Error
	if err != nil || len(entries) == 0 {
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

// AudienceRules returns the Audience rules of the given Announcements.
func (r *GormRepository) AudienceRules(ctx context.Context, announcementIDs []string) ([]AudienceRuleView, error) {
	var rules []AudienceRuleView
	if len(announcementIDs) == 0 {
		return rules, nil
	}
	err := r.db.WithContext(ctx).Raw(`
		SELECT r.target_id, r.department_id, d.code AS department_code, r.batch_year, r.role
		FROM audience_rules r
		LEFT JOIN departments d ON d.id = r.department_id
		WHERE r.target_type = ? AND r.target_id IN ?
		ORDER BY r.created_at, r.id`,
		targetTypeAnnouncement, announcementIDs,
	).Scan(&rules).Error
	return rules, err
}

// Audience returns an Announcement's current Audience rules.
func (r *GormRepository) Audience(ctx context.Context, announcementID string) ([]AudienceRule, error) {
	views, err := r.AudienceRules(ctx, []string{announcementID})
	if err != nil {
		return nil, err
	}
	return audienceFromViews(views), nil
}

// LockDepartments counts how many of the given Department IDs exist and
// share-locks them until the transaction ends, so a concurrent Department
// delete either waits for this Announcement or runs first and fails the check.
func (r *GormRepository) LockDepartments(ctx context.Context, departmentIDs []string) (int, error) {
	var ids []string
	err := r.db.WithContext(ctx).Raw(
		`SELECT id FROM departments WHERE id IN ? FOR SHARE`, departmentIDs,
	).Scan(&ids).Error
	return len(ids), err
}

// StudentPlacement returns the reader's Department and batch year from their
// Student identity, if they have one.
func (r *GormRepository) StudentPlacement(ctx context.Context, userID string) (*string, *int, error) {
	var rows []struct {
		DepartmentID *string `gorm:"column:department_id"`
		BatchYear    int     `gorm:"column:batch_year"`
	}
	err := r.db.WithContext(ctx).Raw(
		`SELECT department_id, batch_year FROM student_identities WHERE user_id = ?`, userID,
	).Scan(&rows).Error
	if err != nil || len(rows) == 0 {
		return nil, nil, err
	}
	return rows[0].DepartmentID, &rows[0].BatchYear, nil
}

// FullName returns a user's display name from their profile.
func (r *GormRepository) FullName(ctx context.Context, userID string) (string, error) {
	var names []string
	err := r.db.WithContext(ctx).Raw(`SELECT full_name FROM profiles WHERE user_id = ?`, userID).Scan(&names).Error
	if err != nil || len(names) == 0 {
		return "", err
	}
	return names[0], nil
}

// FindForUpdate loads an Announcement and locks its row until the transaction
// ends, so approvals, rejections and resubmissions of it happen one at a time.
func (r *GormRepository) FindForUpdate(ctx context.Context, id string) (*Announcement, error) {
	// An ID that isn't a UUID can't match any row; asking Postgres would error.
	if _, err := uuid.Parse(id); err != nil {
		return nil, nil
	}
	var announcement Announcement
	err := r.db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", id).
		First(&announcement).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &announcement, err
}

func (r *GormRepository) UpdateAnnouncement(ctx context.Context, announcement *Announcement) error {
	return r.db.WithContext(ctx).Save(announcement).Error
}

func (r *GormRepository) CreateRevision(ctx context.Context, revision *Revision) error {
	if revision.ID == "" {
		revision.ID = uuid.NewString()
	}
	return r.db.WithContext(ctx).Create(revision).Error
}

func (r *GormRepository) UpdateRevision(ctx context.Context, revision *Revision) error {
	return r.db.WithContext(ctx).Save(revision).Error
}

// OpenRevision returns the Announcement's draft, pending or rejected revision, if any.
func (r *GormRepository) OpenRevision(ctx context.Context, announcementID string) (*Revision, error) {
	var revision Revision
	err := r.db.WithContext(ctx).
		Where("announcement_id = ? AND status IN ?", announcementID, []RevisionStatus{RevisionDraft, RevisionPending, RevisionRejected}).
		First(&revision).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &revision, err
}

// LatestRevisions returns the newest revision of each given Announcement.
func (r *GormRepository) LatestRevisions(ctx context.Context, announcementIDs []string) ([]Revision, error) {
	var revisions []Revision
	if len(announcementIDs) == 0 {
		return revisions, nil
	}
	err := r.db.WithContext(ctx).Raw(`
		SELECT DISTINCT ON (announcement_id) *
		FROM announcement_revisions
		WHERE announcement_id IN ?
		ORDER BY announcement_id, created_at DESC, id DESC`, announcementIDs,
	).Scan(&revisions).Error
	return revisions, err
}

// ApproverScope is what an approver may approve: everything (principal or
// admin), or single-Department submissions for the Departments they are HOD of.
// Their own submissions are never included.
type ApproverScope struct {
	UserID        string
	All           bool
	DepartmentIDs []string
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

// DepartmentCodes maps Department IDs to their codes.
func (r *GormRepository) DepartmentCodes(ctx context.Context, departmentIDs []string) (map[string]string, error) {
	codes := map[string]string{}
	if len(departmentIDs) == 0 {
		return codes, nil
	}
	var rows []struct {
		ID   string `gorm:"column:id"`
		Code string `gorm:"column:code"`
	}
	if err := r.db.WithContext(ctx).Raw(`SELECT id, code FROM departments WHERE id IN ?`, departmentIDs).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		codes[row.ID] = row.Code
	}
	return codes, nil
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
