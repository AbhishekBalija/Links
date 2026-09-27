// Package opportunities runs placement Opportunities from draft to
// publication, and the Applications Students make to them (ADR 0008,
// ADR 0024).
package opportunities

import (
	"context"
	"time"

	"github.com/AbhishekBalija/Links/server/internal/auth"
)

// Type is the kind of Opportunity.
type Type string

const (
	TypeJob        Type = "job"
	TypeInternship Type = "internship"
	TypeTraining   Type = "training"
)

func validType(opportunityType Type) bool {
	switch opportunityType {
	case TypeJob, TypeInternship, TypeTraining:
		return true
	}
	return false
}

// Mode is where a Student applies: in LINKS, or on the company's site.
type Mode string

const (
	ModeInternal Mode = "internal"
	ModeExternal Mode = "external"
)

func validMode(mode Mode) bool {
	return mode == ModeInternal || mode == ModeExternal
}

// Status is where an Opportunity is. Whether it is open is computed from
// Status and ApplyBy.
type Status string

const (
	StatusDraft     Status = "draft"
	StatusPublished Status = "published"
	StatusClosed    Status = "closed"
)

func validStatus(status Status) bool {
	switch status {
	case StatusDraft, StatusPublished, StatusClosed:
		return true
	}
	return false
}

// Opportunity is a job, internship or training posted by placement staff.
type Opportunity struct {
	ID              string     `gorm:"column:id;primaryKey"`
	OpportunityType Type       `gorm:"column:opportunity_type"`
	Title           string     `gorm:"column:title"`
	Company         string     `gorm:"column:company"`
	Description     string     `gorm:"column:description"`
	Location        *string    `gorm:"column:location"`
	Compensation    *string    `gorm:"column:compensation"`
	ApplyBy         time.Time  `gorm:"column:apply_by"`
	ApplicationMode Mode       `gorm:"column:application_mode"`
	ExternalURL     *string    `gorm:"column:external_url"`
	Status          Status     `gorm:"column:status"`
	PostedBy        string     `gorm:"column:posted_by"`
	PublishedAt     *time.Time `gorm:"column:published_at"`
	ClosedAt        *time.Time `gorm:"column:closed_at"`
	ClosedBy        *string    `gorm:"column:closed_by"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at"`
}

func (Opportunity) TableName() string { return "opportunities" }

// View is an Opportunity with its poster's name.
type View struct {
	Opportunity
	PosterName string `gorm:"column:poster_name"`
}

// EligibilityRule is one rule of an Opportunity's Eligibility, matched like
// an Announcement's Audience: the fields a rule sets must all match; matching
// any rule is enough; no rules means everyone.
type EligibilityRule struct {
	DepartmentID *string
	BatchYear    *int
	Role         *auth.Role
}

// EligibilityRuleView is a stored rule with its Department code for display.
type EligibilityRuleView struct {
	TargetID       string  `gorm:"column:target_id"`
	DepartmentID   *string `gorm:"column:department_id"`
	DepartmentCode *string `gorm:"column:department_code"`
	BatchYear      *int    `gorm:"column:batch_year"`
	Role           *string `gorm:"column:role"`
}

// RoleReader reads a user's roles in effect from the database, never the token.
type RoleReader interface {
	GetRoleAssignments(ctx context.Context, userID string) ([]auth.RoleAssignment, error)
}

// Cursor marks the last item of the previous page.
type Cursor struct {
	At time.Time
	ID string
}

// Repository is the Opportunities data access used by the service.
type Repository interface {
	Create(ctx context.Context, opportunity *Opportunity, eligibility []EligibilityRule) error
	Update(ctx context.Context, opportunity *Opportunity) error
	ReplaceEligibility(ctx context.Context, opportunityID string, eligibility []EligibilityRule) error
	FindForUpdate(ctx context.Context, id string) (*Opportunity, error)
	Find(ctx context.Context, id string) (*View, error)
	Eligibility(ctx context.Context, opportunityID string) ([]EligibilityRule, error)
	EligibilityRules(ctx context.Context, opportunityIDs []string) ([]EligibilityRuleView, error)
	Managed(ctx context.Context, status *Status, after *Cursor, limit int) ([]View, error)
	LockDepartments(ctx context.Context, departmentIDs []string) (int, error)
	Feed(ctx context.Context, reader Reader, filter FeedFilter, after *Cursor, limit int) ([]View, error)
	VisibleTo(ctx context.Context, reader Reader, id string) (bool, error)
	StudentPlacement(ctx context.Context, userID string) (*string, *int, error)
	DepartmentIDByCode(ctx context.Context, code string) (*string, error)
}

// Membership is one role a reader holds with the Department it belongs to
// (nil when none), as for Announcements and Events.
type Membership struct {
	Role         string
	DepartmentID *string
}

// Reader is who is looking at Opportunities. A rule matches only when one
// Membership satisfies its role and Department together.
type Reader struct {
	Memberships []Membership
	BatchYear   *int
}

// State picks open or closed Opportunities in the feed.
type State string

const (
	// StateOpen is the default: published and apply_by still ahead, soonest
	// deadline first.
	StateOpen State = "open"
	// StateClosed is closed early or past apply_by, latest deadline first.
	StateClosed State = "closed"
)

// FeedFilter narrows the Opportunity feed.
type FeedFilter struct {
	State        State
	Type         *Type
	DepartmentID *string
}

// Repositories groups what must share one transaction.
type Repositories struct {
	Opportunities Repository
	AuditLogs     auth.AuditLogRepository
}

type UnitOfWork interface {
	WithinTransaction(ctx context.Context, fn func(Repositories) error) error
}
