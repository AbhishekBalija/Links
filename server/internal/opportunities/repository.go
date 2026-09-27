package opportunities

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/AbhishekBalija/Links/server/internal/auth"
)

const targetTypeOpportunity = "opportunity"

type GormUnitOfWork struct {
	db *gorm.DB
}

func NewGormUnitOfWork(db *gorm.DB) *GormUnitOfWork {
	return &GormUnitOfWork{db: db}
}

func (u *GormUnitOfWork) WithinTransaction(ctx context.Context, fn func(Repositories) error) error {
	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(Repositories{
			Opportunities: NewGormRepository(tx),
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

// Create stores an Opportunity and its Eligibility rules.
func (r *GormRepository) Create(ctx context.Context, opportunity *Opportunity, eligibility []EligibilityRule) error {
	if opportunity.ID == "" {
		opportunity.ID = uuid.NewString()
	}
	if err := r.db.WithContext(ctx).Create(opportunity).Error; err != nil {
		return err
	}
	return r.insertEligibility(ctx, opportunity.ID, eligibility)
}

func (r *GormRepository) Update(ctx context.Context, opportunity *Opportunity) error {
	return r.db.WithContext(ctx).Save(opportunity).Error
}

// ReplaceEligibility swaps an Opportunity's Eligibility rules for new ones.
func (r *GormRepository) ReplaceEligibility(ctx context.Context, opportunityID string, eligibility []EligibilityRule) error {
	err := r.db.WithContext(ctx).Exec(
		`DELETE FROM audience_rules WHERE target_type = ? AND target_id = ?`, targetTypeOpportunity, opportunityID,
	).Error
	if err != nil {
		return err
	}
	return r.insertEligibility(ctx, opportunityID, eligibility)
}

func (r *GormRepository) insertEligibility(ctx context.Context, opportunityID string, eligibility []EligibilityRule) error {
	for _, rule := range eligibility {
		var role *string
		if rule.Role != nil {
			value := string(*rule.Role)
			role = &value
		}
		err := r.db.WithContext(ctx).Exec(
			`INSERT INTO audience_rules (id, target_type, target_id, department_id, batch_year, role)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			uuid.NewString(), targetTypeOpportunity, opportunityID, rule.DepartmentID, rule.BatchYear, role,
		).Error
		if err != nil {
			return err
		}
	}
	return nil
}

// FindForUpdate locks the Opportunity row, so changes to one Opportunity run
// one after another.
func (r *GormRepository) FindForUpdate(ctx context.Context, id string) (*Opportunity, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, nil
	}
	var opportunity Opportunity
	err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&opportunity, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &opportunity, err
}

const viewColumns = `o.*, p.full_name AS poster_name`

const viewFrom = `FROM opportunities o JOIN profiles p ON p.user_id = o.posted_by`

// Find loads an Opportunity with its poster's name, without locking.
func (r *GormRepository) Find(ctx context.Context, id string) (*View, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, nil
	}
	var views []View
	err := r.db.WithContext(ctx).Raw(`SELECT `+viewColumns+` `+viewFrom+` WHERE o.id = ?`, id).Scan(&views).Error
	if err != nil || len(views) == 0 {
		return nil, err
	}
	return &views[0], nil
}

func (r *GormRepository) Eligibility(ctx context.Context, opportunityID string) ([]EligibilityRule, error) {
	views, err := r.EligibilityRules(ctx, []string{opportunityID})
	if err != nil {
		return nil, err
	}
	rules := make([]EligibilityRule, 0, len(views))
	for _, view := range views {
		rule := EligibilityRule{DepartmentID: view.DepartmentID, BatchYear: view.BatchYear}
		if view.Role != nil {
			role := auth.Role(*view.Role)
			rule.Role = &role
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

func (r *GormRepository) EligibilityRules(ctx context.Context, opportunityIDs []string) ([]EligibilityRuleView, error) {
	if len(opportunityIDs) == 0 {
		return nil, nil
	}
	var rules []EligibilityRuleView
	err := r.db.WithContext(ctx).Raw(`
		SELECT r.target_id, r.department_id, d.code AS department_code, r.batch_year, r.role
		FROM audience_rules r
		LEFT JOIN departments d ON d.id = r.department_id
		WHERE r.target_type = ? AND r.target_id IN ?
		ORDER BY r.created_at, r.id`, targetTypeOpportunity, opportunityIDs).
		Scan(&rules).Error
	return rules, err
}

// Managed lists every Opportunity for placement staff, newest first.
func (r *GormRepository) Managed(ctx context.Context, status *Status, after *Cursor, limit int) ([]View, error) {
	query := `SELECT ` + viewColumns + ` ` + viewFrom + ` WHERE TRUE`
	args := []any{}
	if status != nil {
		query += ` AND o.status = ?`
		args = append(args, string(*status))
	}
	if after != nil {
		query += ` AND (o.created_at, o.id) < (?, CAST(? AS uuid))`
		args = append(args, after.At, after.ID)
	}
	query += ` ORDER BY o.created_at DESC, o.id DESC LIMIT ?`
	args = append(args, limit)
	var views []View
	err := r.db.WithContext(ctx).Raw(query, args...).Scan(&views).Error
	return views, err
}

// LockDepartments counts how many of the Departments exist and share-locks
// them, so none can be deleted while an Eligibility points at it.
func (r *GormRepository) LockDepartments(ctx context.Context, departmentIDs []string) (int, error) {
	var ids []string
	err := r.db.WithContext(ctx).Raw(`SELECT id FROM departments WHERE id IN ? FOR SHARE`, departmentIDs).Scan(&ids).Error
	return len(ids), err
}

// LockForShare reads the Opportunity and share-locks it, so it can't be
// closed or edited while an Application to it is being made.
func (r *GormRepository) LockForShare(ctx context.Context, id string) (*Opportunity, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, nil
	}
	var opportunity Opportunity
	err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "SHARE"}).First(&opportunity, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &opportunity, err
}

func (r *GormRepository) CreateApplication(ctx context.Context, application *Application) error {
	if application.ID == "" {
		application.ID = uuid.NewString()
	}
	return r.db.WithContext(ctx).Create(application).Error
}

func (r *GormRepository) UpdateApplication(ctx context.Context, application *Application) error {
	return r.db.WithContext(ctx).Save(application).Error
}

// FindApplicationForUpdate locks the Student's Application to the
// Opportunity, if they made one.
func (r *GormRepository) FindApplicationForUpdate(ctx context.Context, opportunityID, studentID string) (*Application, error) {
	if _, err := uuid.Parse(opportunityID); err != nil {
		return nil, nil
	}
	var applications []Application
	err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("opportunity_id = ? AND student_id = ?", opportunityID, studentID).Limit(1).Find(&applications).Error
	if err != nil || len(applications) == 0 {
		return nil, err
	}
	return &applications[0], nil
}

// ApplicationsOf returns the Student's own Applications to these
// Opportunities.
func (r *GormRepository) ApplicationsOf(ctx context.Context, studentID string, opportunityIDs []string) ([]Application, error) {
	if len(opportunityIDs) == 0 {
		return nil, nil
	}
	var applications []Application
	err := r.db.WithContext(ctx).Where("student_id = ? AND opportunity_id IN ?", studentID, opportunityIDs).Find(&applications).Error
	return applications, err
}
