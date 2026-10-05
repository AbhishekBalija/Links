// Package events runs Event proposals from draft through HOD review and final
// approval to publication, then RSVPs and participant exports (ADR 0006,
// ADR 0023).
package events

import (
	"context"
	"github.com/AbhishekBalija/Links/server/internal/mailer"
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
	ID          string `gorm:"column:id;primaryKey"`
	Title       string `gorm:"column:title"`
	Description string `gorm:"column:description"`
	EventType   Type   `gorm:"column:event_type"`
	ProposerID  string `gorm:"column:proposer_id"`
	// OrganiserID runs the Event; it starts as the proposer (ADR 0028).
	OrganiserID     string     `gorm:"column:organiser_id"`
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
	OrganiserName  string  `gorm:"column:organiser_name"`
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
	// UnfinishedOf and DepartmentHOD serve ending a role (ADR 0028).
	UnfinishedOf(ctx context.Context, personID string, at time.Time) ([]Event, error)
	DepartmentHOD(ctx context.Context, departmentID, exceptUserID string) (*auth.PersonRef, error)
	OrganiserCandidates(ctx context.Context, departmentIDs []string, exceptUserID string) ([]auth.PersonRef, error)
	UpcomingInDepartment(ctx context.Context, departmentID string, limit int) ([]View, error)
	Create(ctx context.Context, event *Event, audience []AudienceRule) error
	Update(ctx context.Context, event *Event) error
	DeleteDraft(ctx context.Context, eventID string) error
	ReplaceAudience(ctx context.Context, eventID string, audience []AudienceRule) error
	FindForUpdate(ctx context.Context, id string) (*Event, error)
	Find(ctx context.Context, id string) (*View, error)
	Audience(ctx context.Context, eventID string) ([]AudienceRule, error)
	AudienceRules(ctx context.Context, eventIDs []string) ([]AudienceRuleView, error)
	Authored(ctx context.Context, proposerID string, filter MineFilter, after *Cursor, limit int) ([]View, error)
	LockDepartments(ctx context.Context, departmentIDs []string) (int, error)
	IsFaculty(ctx context.Context, userID string) (bool, error)
	CreateReview(ctx context.Context, review *Review) error
	Reviews(ctx context.Context, eventIDs []string) ([]ReviewView, error)
	SentBack(ctx context.Context, proposerID string, limit int) ([]SentBackRow, error)
	Waiting(ctx context.Context, proposerID string, limit int) ([]WaitingRow, error)
	DepartmentHasHOD(ctx context.Context, departmentID string) (bool, error)
	Queue(ctx context.Context, scope ReviewerScope, after *Cursor, limit int) ([]View, error)
	QueueSummary(ctx context.Context, scope ReviewerScope) (int, *time.Time, error)
	StudentPlacement(ctx context.Context, userID string) (*string, *int, error)
	DepartmentIDByCode(ctx context.Context, code string) (*string, error)
	Feed(ctx context.Context, readerID string, reader Reader, filter FeedFilter, after *Cursor, limit int) ([]View, error)
	RSVPSummaries(ctx context.Context, eventIDs []string, userID string) (map[string]map[RSVPStatus]int, map[string]RSVPStatus, error)
	VisibleTo(ctx context.Context, reader Reader, id string) (bool, error)
	FindRSVP(ctx context.Context, eventID, userID string) (*RSVP, error)
	SaveRSVP(ctx context.Context, rsvp *RSVP) error
	RSVPCounts(ctx context.Context, eventID string) (map[RSVPStatus]int, error)
	RSVPPeople(ctx context.Context, eventID string, after *Cursor, limit int) ([]RSVPPerson, error)
	Answerers(ctx context.Context, eventID string) ([]mailer.Recipient, error)
	ExportRows(ctx context.Context, eventID string) ([]ExportRow, error)
}

// ExportRow is one participant in an Event's export. USN and Batch are set
// only for students.
type ExportRow struct {
	FullName       string     `gorm:"column:full_name"`
	Email          *string    `gorm:"column:email"`
	USN            *string    `gorm:"column:usn"`
	BatchYear      *int       `gorm:"column:batch_year"`
	DepartmentCode *string    `gorm:"column:department_code"`
	Status         RSVPStatus `gorm:"column:status"`
	RespondedAt    time.Time  `gorm:"column:updated_at"`
}

// RSVPStatus is a reader's answer to an Event.
type RSVPStatus string

const (
	RSVPGoing      RSVPStatus = "going"
	RSVPInterested RSVPStatus = "interested"
	RSVPNotGoing   RSVPStatus = "not_going"
)

// RSVP is one reader's answer; one per reader per Event.
type RSVP struct {
	ID        string     `gorm:"column:id;primaryKey"`
	EventID   string     `gorm:"column:event_id"`
	UserID    string     `gorm:"column:user_id"`
	Status    RSVPStatus `gorm:"column:status"`
	CreatedAt time.Time  `gorm:"column:created_at"`
	UpdatedAt time.Time  `gorm:"column:updated_at"`
}

func (RSVP) TableName() string { return "event_rsvps" }

// RSVPPerson is an RSVP with who gave it, for the Event's organisers.
type RSVPPerson struct {
	UserID    string     `gorm:"column:user_id"`
	FullName  string     `gorm:"column:full_name"`
	Username  string     `gorm:"column:username"`
	Status    RSVPStatus `gorm:"column:status"`
	UpdatedAt time.Time  `gorm:"column:updated_at"`
}

// Membership is one role a reader holds with the Department it belongs to
// (nil when none), as for Announcements.
type Membership struct {
	Role         string
	DepartmentID *string
}

// Reader is who is looking at Events. A rule matches only when one
// Membership satisfies its role and Department together.
type Reader struct {
	Memberships []Membership
	BatchYear   *int
}

// FeedFilter narrows the Event feed. Without From, only Events not yet over
// are listed.
// Show picks which Events the feed lists.
type Show string

const (
	// ShowUpcoming is the default: Events not over yet, soonest first.
	ShowUpcoming Show = "upcoming"
	// ShowGoing is upcoming Events the reader answered going to.
	ShowGoing Show = "going"
	// ShowPast is Events that have ended, most recent first.
	ShowPast Show = "past"
)

type FeedFilter struct {
	Show         Show
	From         *time.Time
	To           *time.Time
	DepartmentID *string
	EventType    *Type
}

// Stage is which review an Event is at.
type Stage string

const (
	StageHOD   Stage = "hod"
	StageFinal Stage = "final"
)

// Decision is a reviewer's answer.
type Decision string

const (
	DecisionApprove        Decision = "approve"
	DecisionRequestChanges Decision = "request_changes"
	DecisionReject         Decision = "reject"
)

// Review is one decision at one stage, kept as the Event's history.
type Review struct {
	ID         string    `gorm:"column:id;primaryKey"`
	EventID    string    `gorm:"column:event_id"`
	Stage      Stage     `gorm:"column:stage"`
	ReviewerID string    `gorm:"column:reviewer_id"`
	Decision   Decision  `gorm:"column:decision"`
	Note       *string   `gorm:"column:note"`
	CreatedAt  time.Time `gorm:"column:created_at"`
}

func (Review) TableName() string { return "event_reviews" }

// ReviewView is a Review with its reviewer's name.
type ReviewView struct {
	Review
	ReviewerName string `gorm:"column:reviewer_name"`
}

// ReviewerScope is what a reviewer may act on: the HOD stage of their
// Departments' Events, and with All (principal or admin) the HOD stage of
// Departments without an HOD and every final approval.
type ReviewerScope struct {
	UserID         string
	All            bool
	HODDepartments []string
}

// Repositories groups what must share one transaction.
type Repositories struct {
	Events    Repository
	AuditLogs auth.AuditLogRepository
}

type UnitOfWork interface {
	WithinTransaction(ctx context.Context, fn func(Repositories) error) error
}
