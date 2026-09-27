package events

import (
	"context"
	"errors"
	"fmt"

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

func (r *GormRepository) CreateReview(ctx context.Context, review *Review) error {
	if review.ID == "" {
		review.ID = uuid.NewString()
	}
	return r.db.WithContext(ctx).Create(review).Error
}

// Reviews returns the decisions on these Events, oldest first.
func (r *GormRepository) Reviews(ctx context.Context, eventIDs []string) ([]ReviewView, error) {
	if len(eventIDs) == 0 {
		return nil, nil
	}
	var reviews []ReviewView
	err := r.db.WithContext(ctx).Raw(`
		SELECT v.*, p.full_name AS reviewer_name
		FROM event_reviews v JOIN profiles p ON p.user_id = v.reviewer_id
		WHERE v.event_id IN ?
		ORDER BY v.created_at, v.id`, eventIDs).
		Scan(&reviews).Error
	return reviews, err
}

// hasHOD is true when someone holds the HOD role for the Department now.
const hasHOD = `EXISTS (SELECT 1 FROM role_assignments h
	WHERE h.role = 'hod' AND h.scope_type = 'department' AND h.scope_id = %s
	  AND h.starts_at <= now() AND (h.ends_at IS NULL OR h.ends_at > now()))`

func (r *GormRepository) DepartmentHasHOD(ctx context.Context, departmentID string) (bool, error) {
	var found bool
	err := r.db.WithContext(ctx).Raw(`SELECT `+fmt.Sprintf(hasHOD, "?"), departmentID).Scan(&found).Error
	return found, err
}

// Queue returns the Events waiting for this reviewer, oldest submission
// first. Their own proposals are never included.
func (r *GormRepository) Queue(ctx context.Context, scope ReviewerScope, after *Cursor, limit int) ([]View, error) {
	query := `SELECT ` + viewColumns + ` ` + viewFrom + `
		WHERE e.proposer_id <> ? AND (
			(e.status = 'submitted' AND (e.department_id IN ? OR (? AND NOT ` + fmt.Sprintf(hasHOD, "e.department_id") + `)))
			OR (e.status = 'hod_approved' AND ?)
		)`
	args := []any{scope.UserID, scope.HODDepartments, scope.All, scope.All}
	if after != nil {
		query += ` AND (e.submitted_at, e.id) > (?, CAST(? AS uuid))`
		args = append(args, after.At, after.ID)
	}
	query += ` ORDER BY e.submitted_at, e.id LIMIT ?`
	args = append(args, limit)
	var views []View
	err := r.db.WithContext(ctx).Raw(query, args...).Scan(&views).Error
	return views, err
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

func (r *GormRepository) FindRSVP(ctx context.Context, eventID, userID string) (*RSVP, error) {
	var rsvps []RSVP
	err := r.db.WithContext(ctx).Where("event_id = ? AND user_id = ?", eventID, userID).Limit(1).Find(&rsvps).Error
	if err != nil || len(rsvps) == 0 {
		return nil, err
	}
	return &rsvps[0], nil
}

func (r *GormRepository) SaveRSVP(ctx context.Context, rsvp *RSVP) error {
	if rsvp.ID == "" {
		rsvp.ID = uuid.NewString()
		return r.db.WithContext(ctx).Create(rsvp).Error
	}
	return r.db.WithContext(ctx).Save(rsvp).Error
}

func (r *GormRepository) RSVPCounts(ctx context.Context, eventID string) (map[RSVPStatus]int, error) {
	var rows []struct {
		Status RSVPStatus `gorm:"column:status"`
		Count  int        `gorm:"column:count"`
	}
	err := r.db.WithContext(ctx).Raw(`SELECT status, count(*) AS count FROM event_rsvps WHERE event_id = ? GROUP BY status`, eventID).Scan(&rows).Error
	counts := map[RSVPStatus]int{}
	for _, row := range rows {
		counts[row.Status] = row.Count
	}
	return counts, err
}

// RSVPPeople lists who answered, earliest answer first.
func (r *GormRepository) RSVPPeople(ctx context.Context, eventID string, after *Cursor, limit int) ([]RSVPPerson, error) {
	query := `SELECT v.user_id, p.full_name, p.username, v.status, v.updated_at
		FROM event_rsvps v JOIN profiles p ON p.user_id = v.user_id
		WHERE v.event_id = ?`
	args := []any{eventID}
	if after != nil {
		query += ` AND (v.updated_at, v.user_id) > (?, CAST(? AS uuid))`
		args = append(args, after.At, after.ID)
	}
	query += ` ORDER BY v.updated_at, v.user_id LIMIT ?`
	args = append(args, limit)
	var people []RSVPPerson
	err := r.db.WithContext(ctx).Raw(query, args...).Scan(&people).Error
	return people, err
}

// ExportRows lists everyone who answered, by answer and then name. The
// Department is the Student identity's, otherwise one of their
// Department-scoped roles in effect.
func (r *GormRepository) ExportRows(ctx context.Context, eventID string) ([]ExportRow, error) {
	var rows []ExportRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT p.full_name, u.email,
			CASE WHEN s.is_student THEN si.usn END AS usn,
			CASE WHEN s.is_student THEN si.batch_year END AS batch_year,
			COALESCE(sd.code, (
				SELECT d.code FROM role_assignments r JOIN departments d ON d.id = r.scope_id
				WHERE r.user_id = u.id AND r.scope_type = 'department'
				  AND r.starts_at <= now() AND (r.ends_at IS NULL OR r.ends_at > now())
				ORDER BY d.code LIMIT 1
			)) AS department_code,
			v.status, v.updated_at
		FROM event_rsvps v
		JOIN users u ON u.id = v.user_id
		JOIN profiles p ON p.user_id = v.user_id
		LEFT JOIN student_identities si ON si.user_id = v.user_id
		LEFT JOIN departments sd ON sd.id = si.department_id
		CROSS JOIN LATERAL (SELECT EXISTS (
			SELECT 1 FROM role_assignments r WHERE r.user_id = u.id AND r.role = 'student'
			  AND r.starts_at <= now() AND (r.ends_at IS NULL OR r.ends_at > now())
		) AS is_student) s
		WHERE v.event_id = ?
		ORDER BY CASE v.status WHEN 'going' THEN 0 WHEN 'interested' THEN 1 ELSE 2 END, lower(p.full_name), v.user_id`, eventID).
		Scan(&rows).Error
	return rows, err
}
