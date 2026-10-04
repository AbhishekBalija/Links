package announcements

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/AbhishekBalija/Links/server/internal/auth"
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
	// UserID is the reader: their own published Announcements are always in
	// their feed, even when sent only to others.
	UserID      string
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

func (r *GormRepository) UnfinishedOf(ctx context.Context, authorID string) ([]Announcement, error) {
	var unfinished []Announcement
	err := r.db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where(`publisher_id = ? AND (status IN ? OR (status = ? AND EXISTS (
			SELECT 1 FROM announcement_revisions ar
			WHERE ar.announcement_id = announcements.id AND ar.status IN ?)))`,
			authorID, []Status{StatusPending, StatusRejected}, StatusPublished,
			[]RevisionStatus{RevisionPending, RevisionRejected}).
		Order("created_at").
		Find(&unfinished).Error
	return unfinished, err
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
