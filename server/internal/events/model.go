// Package events runs Event proposals from draft through HOD review and final
// approval to publication, then RSVPs and participant exports (ADR 0006,
// ADR 0023).
package events

import (
	"context"
	"time"

	"github.com/AbhishekBalija/Links/server/internal/auth"
)

// Status is where an Event is in its workflow. One column carries it all.
type Status string

const (
	StatusDraft                 Status = "draft"
	StatusSubmitted             Status = "submitted"
	StatusHODChangesRequested   Status = "hod_changes_requested"
	StatusHODRejected           Status = "hod_rejected"
	StatusHODApproved           Status = "hod_approved"
	StatusFinalChangesRequested Status = "final_changes_requested"
	StatusFinalRejected         Status = "final_rejected"
	StatusPublished             Status = "published"
	StatusCancelled             Status = "cancelled"
)

// Type is the kind of Event. The placement officer only proposes training.
type Type string

const (
	TypeTalk        Type = "talk"
	TypeWorkshop    Type = "workshop"
	TypeCompetition Type = "competition"
	TypeCultural    Type = "cultural"
	TypeSports      Type = "sports"
	TypeTraining    Type = "training"
	TypeOther       Type = "other"
)

func validType(eventType Type) bool {
	switch eventType {
	case TypeTalk, TypeWorkshop, TypeCompetition, TypeCultural, TypeSports, TypeTraining, TypeOther:
		return true
	}
	return false
}

// Event is an Event proposal or a published Event.
type Event struct {
	ID              string     `gorm:"column:id;primaryKey"`
	Title           string     `gorm:"column:title"`
	Description     string     `gorm:"column:description"`
	EventType       Type       `gorm:"column:event_type"`
	ProposerID      string     `gorm:"column:proposer_id"`
	DepartmentID    *string    `gorm:"column:department_id"`
	FacultyMentorID *string    `gorm:"column:faculty_mentor_id"`
	Location        string     `gorm:"column:location"`
	StartsAt        time.Time  `gorm:"column:starts_at"`
	EndsAt          time.Time  `gorm:"column:ends_at"`
	Capacity        *int       `gorm:"column:capacity"`
	Status          Status     `gorm:"column:status"`
	SubmittedAt     *time.Time `gorm:"column:submitted_at"`
	PublishedAt     *time.Time `gorm:"column:published_at"`
	CancelledAt     *time.Time `gorm:"column:cancelled_at"`
	CancelledBy     *string    `gorm:"column:cancelled_by"`
	CancelReason    *string    `gorm:"column:cancel_reason"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at"`
}

func (Event) TableName() string { return "events" }

// View is an Event with the names shown next to it.
type View struct {
	Event
	ProposerName   string  `gorm:"column:proposer_name"`
	DepartmentCode *string `gorm:"column:department_code"`
	MentorName     *string `gorm:"column:mentor_name"`
}

// AudienceRule is one rule of an Event's Audience, as for Announcements: the
// fields a rule sets must all match; matching any rule is enough; no rules
// means the whole college.
type AudienceRule struct {
	DepartmentID *string
	BatchYear    *int
	Role         *auth.Role
}

// AudienceRuleView is a stored rule with its Department code for display.
type AudienceRuleView struct {
	TargetID       string  `gorm:"column:target_id"`
	DepartmentID   *string `gorm:"column:department_id"`
	DepartmentCode *string `gorm:"column:department_code"`
	BatchYear      *int    `gorm:"column:batch_year"`
	Role           *string `gorm:"column:role"`
}

// Grant is one role the user holds now; DepartmentID is empty for a global role.
type Grant struct {
	Role         auth.Role
	DepartmentID string
}

// RoleReader reads a user's roles in effect from the database, never the token.
type RoleReader interface {
	GetRoleAssignments(ctx context.Context, userID string) ([]auth.RoleAssignment, error)
}

// Cursor marks the last Event of the previous page.
type Cursor struct {
	At time.Time
	ID string
}

// MineFilter narrows a proposer's own Events to what needs them.
type MineFilter string

const (
	MineAll   MineFilter = ""
	MineDraft MineFilter = "draft"
	// MineWaiting is waiting for a reviewer at either stage.
	MineWaiting MineFilter = "waiting"
	// MineAttention is sent back or rejected at either stage.
	MineAttention MineFilter = "attention"
	// MineLive is published and not over yet.
	MineLive MineFilter = "live"
	// MineEnded is cancelled, or published and over.
	MineEnded MineFilter = "ended"
)

func validMineFilter(filter MineFilter) bool {
	switch filter {
	case MineAll, MineDraft, MineWaiting, MineAttention, MineLive, MineEnded:
		return true
	}
	return false
}

// Repository is the Events data access used by the service.
type Repository interface {
	Create(ctx context.Context, event *Event, audience []AudienceRule) error
	Update(ctx context.Context, event *Event) error
	ReplaceAudience(ctx context.Context, eventID string, audience []AudienceRule) error
	FindForUpdate(ctx context.Context, id string) (*Event, error)
	Find(ctx context.Context, id string) (*View, error)
	Audience(ctx context.Context, eventID string) ([]AudienceRule, error)
	AudienceRules(ctx context.Context, eventIDs []string) ([]AudienceRuleView, error)
	Authored(ctx context.Context, proposerID string, filter MineFilter, after *Cursor, limit int) ([]View, error)
	LockDepartments(ctx context.Context, departmentIDs []string) (int, error)
	IsFaculty(ctx context.Context, userID string) (bool, error)
}

// Repositories groups what must share one transaction.
type Repositories struct {
	Events    Repository
	AuditLogs auth.AuditLogRepository
}

type UnitOfWork interface {
	WithinTransaction(ctx context.Context, fn func(Repositories) error) error
}
