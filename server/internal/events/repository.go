package events

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/AbhishekBalija/Links/server/internal/auth"
)

const targetTypeEvent = "event"

type GormUnitOfWork struct {
	db *gorm.DB
}

func NewGormUnitOfWork(db *gorm.DB) *GormUnitOfWork {
	return &GormUnitOfWork{db: db}
}

func (u *GormUnitOfWork) WithinTransaction(ctx context.Context, fn func(Repositories) error) error {
	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(Repositories{
			Events:    NewGormRepository(tx),
			AuditLogs: auth.NewGormAuditLogRepository(tx),
		})
	})
}

type GormRepository struct {
	db *gorm.DB
}

func NewGormRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

// Create stores an Event and its Audience rules.
func (r *GormRepository) Create(ctx context.Context, event *Event, audience []AudienceRule) error {
	if event.ID == "" {
		event.ID = uuid.NewString()
	}
	if err := r.db.WithContext(ctx).Create(event).Error; err != nil {
		return err
	}
	return r.insertAudience(ctx, event.ID, audience)
}

func (r *GormRepository) Update(ctx context.Context, event *Event) error {
	return r.db.WithContext(ctx).Save(event).Error
}

// ReplaceAudience swaps an Event's Audience rules for new ones.
func (r *GormRepository) ReplaceAudience(ctx context.Context, eventID string, audience []AudienceRule) error {
	err := r.db.WithContext(ctx).Exec(
		`DELETE FROM audience_rules WHERE target_type = ? AND target_id = ?`, targetTypeEvent, eventID,
	).Error
	if err != nil {
		return err
	}
	return r.insertAudience(ctx, eventID, audience)
}

func (r *GormRepository) insertAudience(ctx context.Context, eventID string, audience []AudienceRule) error {
	for _, rule := range audience {
		var role *string
		if rule.Role != nil {
			value := string(*rule.Role)
			role = &value
		}
		err := r.db.WithContext(ctx).Exec(
			`INSERT INTO audience_rules (id, target_type, target_id, department_id, batch_year, role)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			uuid.NewString(), targetTypeEvent, eventID, rule.DepartmentID, rule.BatchYear, role,
		).Error
		if err != nil {
			return err
		}
	}
	return nil
}

// FindForUpdate locks the Event row, so every workflow step on one Event runs
// one after another and a second decision sees the first.
func (r *GormRepository) FindForUpdate(ctx context.Context, id string) (*Event, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, nil
	}
	var event Event
	err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&event, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &event, err
}

const viewColumns = `e.*, p.full_name AS proposer_name, d.code AS department_code, mp.full_name AS mentor_name`

const viewFrom = `FROM events e
	JOIN profiles p ON p.user_id = e.proposer_id
	LEFT JOIN departments d ON d.id = e.department_id
	LEFT JOIN profiles mp ON mp.user_id = e.faculty_mentor_id`

// Find loads an Event with its display names, without locking.
func (r *GormRepository) Find(ctx context.Context, id string) (*View, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, nil
	}
	var views []View
	err := r.db.WithContext(ctx).Raw(`SELECT `+viewColumns+` `+viewFrom+` WHERE e.id = ?`, id).Scan(&views).Error
	if err != nil || len(views) == 0 {
		return nil, err
	}
	return &views[0], nil
}

func (r *GormRepository) Audience(ctx context.Context, eventID string) ([]AudienceRule, error) {
	views, err := r.AudienceRules(ctx, []string{eventID})
	if err != nil {
		return nil, err
	}
	rules := make([]AudienceRule, 0, len(views))
	for _, view := range views {
		rule := AudienceRule{DepartmentID: view.DepartmentID, BatchYear: view.BatchYear}
		if view.Role != nil {
			role := auth.Role(*view.Role)
			rule.Role = &role
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

func (r *GormRepository) AudienceRules(ctx context.Context, eventIDs []string) ([]AudienceRuleView, error) {
	if len(eventIDs) == 0 {
		return nil, nil
	}
	var rules []AudienceRuleView
	err := r.db.WithContext(ctx).Raw(`
		SELECT r.target_id, r.department_id, d.code AS department_code, r.batch_year, r.role
		FROM audience_rules r
		LEFT JOIN departments d ON d.id = r.department_id
		WHERE r.target_type = ? AND r.target_id IN ?
		ORDER BY r.created_at, r.id`, targetTypeEvent, eventIDs).
		Scan(&rules).Error
	return rules, err
}

// Authored lists the proposer's own Events, newest first.
func (r *GormRepository) Authored(ctx context.Context, proposerID string, filter MineFilter, after *Cursor, limit int) ([]View, error) {
	query := `SELECT ` + viewColumns + ` ` + viewFrom + ` WHERE e.proposer_id = ?`
	args := []any{proposerID}
	switch filter {
	case MineDraft:
		query += ` AND e.status = 'draft'`
	case MineWaiting:
		query += ` AND e.status IN ('submitted', 'hod_approved')`
	case MineAttention:
		query += ` AND e.status IN ('hod_changes_requested', 'final_changes_requested', 'hod_rejected', 'final_rejected')`
	case MineLive:
		query += ` AND e.status = 'published' AND e.ends_at > now()`
	case MineEnded:
		query += ` AND (e.status = 'cancelled' OR (e.status = 'published' AND e.ends_at <= now()))`
	}
	if after != nil {
		query += ` AND (e.created_at, e.id) < (?, CAST(? AS uuid))`
		args = append(args, after.At, after.ID)
	}
	query += ` ORDER BY e.created_at DESC, e.id DESC LIMIT ?`
	args = append(args, limit)
	var views []View
	err := r.db.WithContext(ctx).Raw(query, args...).Scan(&views).Error
	return views, err
}

// LockDepartments counts how many of the Departments exist and share-locks
// them, so none can be deleted while an Event points at it.
func (r *GormRepository) LockDepartments(ctx context.Context, departmentIDs []string) (int, error) {
	var ids []string
	err := r.db.WithContext(ctx).Raw(`SELECT id FROM departments WHERE id IN ? FOR SHARE`, departmentIDs).Scan(&ids).Error
	return len(ids), err
}

// IsFaculty reports whether the user is active and teaches somewhere now.
func (r *GormRepository) IsFaculty(ctx context.Context, userID string) (bool, error) {
	if _, err := uuid.Parse(userID); err != nil {
		return false, nil
	}
	var count int64
	err := r.db.WithContext(ctx).Raw(`
		SELECT count(*) FROM role_assignments r JOIN users u ON u.id = r.user_id
		WHERE r.user_id = ? AND r.role = 'faculty' AND u.status = 'active'
		  AND r.starts_at <= now() AND (r.ends_at IS NULL OR r.ends_at > now())`, userID).
		Scan(&count).Error
	return count > 0, err
}
