package announcements

import (
	"context"
	"time"

	"github.com/AbhishekBalija/Links/server/internal/auth"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const targetTypeAnnouncement = "announcement"

// Reader is who is looking at the feed: the Departments, batch year and roles
// that Audience rules are matched against.
type Reader struct {
	DepartmentIDs []string
	BatchYear     *int
	Roles         []string
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
	for _, rule := range audience {
		var role *string
		if rule.Role != nil {
			value := string(*rule.Role)
			role = &value
		}
		err := r.db.WithContext(ctx).Exec(
			`INSERT INTO audience_rules (id, target_type, target_id, department_id, batch_year, role)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			uuid.NewString(), targetTypeAnnouncement, announcement.ID, rule.DepartmentID, rule.BatchYear, role,
		).Error
		if err != nil {
			return err
		}
	}
	return nil
}

// Feed returns published, unexpired Announcements whose Audience includes the
// reader, newest first. Fields a rule sets must all match; any matching rule
// is enough; no rules means the whole college.
func (r *GormRepository) Feed(ctx context.Context, reader Reader, cursor *FeedCursor, limit int) ([]FeedEntry, error) {
	query := `
		SELECT a.*, p.full_name AS publisher_name
		FROM announcements a
		JOIN profiles p ON p.user_id = a.publisher_id
		WHERE a.status = 'published'
		  AND (a.expires_at IS NULL OR a.expires_at > now())
		  AND (
		    NOT EXISTS (
		      SELECT 1 FROM audience_rules r
		      WHERE r.target_type = @target_type AND r.target_id = a.id
		    )
		    OR EXISTS (
		      SELECT 1 FROM audience_rules r
		      WHERE r.target_type = @target_type AND r.target_id = a.id
		        AND (r.department_id IS NULL OR r.department_id IN @departments)
		        AND (r.batch_year IS NULL OR r.batch_year = @batch_year)
		        AND (r.role IS NULL OR r.role IN @roles)
		    )
		  )`
	args := map[string]any{
		"target_type": targetTypeAnnouncement,
		"departments": reader.DepartmentIDs,
		"batch_year":  reader.BatchYear,
		"roles":       reader.Roles,
		"limit":       limit,
	}
	if cursor != nil {
		query += ` AND (a.published_at, a.id) < (@cursor_time, CAST(@cursor_id AS uuid))`
		args["cursor_time"] = cursor.PublishedAt
		args["cursor_id"] = cursor.ID
	}
	query += ` ORDER BY a.published_at DESC, a.id DESC LIMIT @limit`

	var entries []FeedEntry
	err := r.db.WithContext(ctx).Raw(query, args).Scan(&entries).Error
	return entries, err
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
