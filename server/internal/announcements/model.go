package announcements

import (
	"context"
	"time"

	"github.com/AbhishekBalija/Links/server/internal/auth"
)

// Category separates official, Department and placement notices.
type Category string

const (
	CategoryOfficial   Category = "official"
	CategoryDepartment Category = "department"
	CategoryPlacement  Category = "placement"
)

// Status is where an Announcement is in its lifecycle. Expiry is not a status:
// an expired Announcement simply drops out of the feed.
type Status string

const (
	StatusDraft     Status = "draft"
	StatusPending   Status = "pending"
	StatusPublished Status = "published"
	StatusRejected  Status = "rejected"
	StatusWithdrawn Status = "withdrawn"
)

// Announcement is an official notice sent to an Audience.
type Announcement struct {
	ID          string     `gorm:"column:id;primaryKey"`
	Title       string     `gorm:"column:title"`
	Body        string     `gorm:"column:body"`
	Category    Category   `gorm:"column:category"`
	PublisherID string     `gorm:"column:publisher_id"`
	Status      Status     `gorm:"column:status"`
	PublishedAt *time.Time `gorm:"column:published_at"`
	ExpiresAt   *time.Time `gorm:"column:expires_at"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
	UpdatedAt   time.Time  `gorm:"column:updated_at"`
}

func (Announcement) TableName() string { return "announcements" }

// RevisionStatus is where content waiting for approval stands.
type RevisionStatus string

const (
	RevisionDraft    RevisionStatus = "draft"
	RevisionPending  RevisionStatus = "pending"
	RevisionApproved RevisionStatus = "approved"
	RevisionRejected RevisionStatus = "rejected"
)

// StoredRule is an Audience rule as saved on a revision.
type StoredRule struct {
	DepartmentID *string `json:"department_id,omitempty"`
	BatchYear    *int    `json:"batch_year,omitempty"`
	Role         *string `json:"role,omitempty"`
}

// Revision is content waiting for Announcement approval: a new Announcement's
// first version, or an edit to a published one.
type Revision struct {
	ID                   string         `gorm:"column:id;primaryKey"`
	AnnouncementID       string         `gorm:"column:announcement_id"`
	Title                string         `gorm:"column:title"`
	Body                 string         `gorm:"column:body"`
	Category             Category       `gorm:"column:category"`
	Audience             []StoredRule   `gorm:"column:audience;serializer:json"`
	ExpiresAt            *time.Time     `gorm:"column:expires_at"`
	Status               RevisionStatus `gorm:"column:status"`
	ApproverDepartmentID *string        `gorm:"column:approver_department_id"`
	SubmittedBy          string         `gorm:"column:submitted_by"`
	SubmittedAt          *time.Time     `gorm:"column:submitted_at"`
	ReviewedBy           *string        `gorm:"column:reviewed_by"`
	ReviewedAt           *time.Time     `gorm:"column:reviewed_at"`
	ReviewNote           *string        `gorm:"column:review_note"`
	CreatedAt            time.Time      `gorm:"column:created_at"`
	UpdatedAt            time.Time      `gorm:"column:updated_at"`
}

func (Revision) TableName() string { return "announcement_revisions" }

// AudienceRule is one rule of an Audience. The fields a rule sets must all
// match a reader; a reader matching any rule is in the Audience.
type AudienceRule struct {
	DepartmentID *string
	BatchYear    *int
	Role         *auth.Role
}

// Repository is the Announcements data access used by the service.
type Repository interface {
	Create(ctx context.Context, announcement *Announcement, audience []AudienceRule) error
	Feed(ctx context.Context, reader Reader, cursor *FeedCursor, limit int) ([]FeedEntry, error)
	AudienceRules(ctx context.Context, announcementIDs []string) ([]AudienceRuleView, error)
	LockDepartments(ctx context.Context, departmentIDs []string) (int, error)
	StudentPlacement(ctx context.Context, userID string) (*string, *int, error)
	FullName(ctx context.Context, userID string) (string, error)

	FindForUpdate(ctx context.Context, id string) (*Announcement, error)
	UpdateAnnouncement(ctx context.Context, announcement *Announcement) error
	ReplaceAudience(ctx context.Context, announcementID string, audience []AudienceRule) error
	CreateRevision(ctx context.Context, revision *Revision) error
	UpdateRevision(ctx context.Context, revision *Revision) error
	OpenRevision(ctx context.Context, announcementID string) (*Revision, error)
	LatestRevisions(ctx context.Context, announcementIDs []string) ([]Revision, error)
	Queue(ctx context.Context, scope ApproverScope, after *FeedCursor, limit int) ([]QueueEntry, error)
	Authored(ctx context.Context, authorID string, after *FeedCursor, limit int) ([]FeedEntry, error)
	DepartmentCodes(ctx context.Context, departmentIDs []string) (map[string]string, error)
}

// RoleReader loads the role assignments a user currently holds.
type RoleReader interface {
	GetRoleAssignments(ctx context.Context, userID string) ([]auth.RoleAssignment, error)
}

type Repositories struct {
	Announcements Repository
	AuditLogs     auth.AuditLogRepository
}

type UnitOfWork interface {
	WithinTransaction(ctx context.Context, fn func(Repositories) error) error
}
