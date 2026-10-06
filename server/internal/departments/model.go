package departments

import (
	"context"
	"time"

	"github.com/AbhishekBalija/Links/server/internal/auth"
)

// Department represents one academic department in LINKS.
type Department struct {
	ID          string    `gorm:"column:id;primaryKey"`
	Code        string    `gorm:"column:code;uniqueIndex;not null"`
	Name        string    `gorm:"column:name;not null"`
	Description *string   `gorm:"column:description"`
	HODUserID   *string   `gorm:"column:hod_user_id"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

func (Department) TableName() string { return "departments" }

type Repository interface {
	List(ctx context.Context) ([]Department, error)
	FindByCode(ctx context.Context, code string) (*Department, error)
	FindByCodeForUpdate(ctx context.Context, code string) (*Department, error)
	Create(ctx context.Context, department *Department) error
	Update(ctx context.Context, department *Department) error
	Delete(ctx context.Context, department *Department) error
	IsReferenced(ctx context.Context, departmentID string) (bool, error)
	CanAssignHOD(ctx context.Context, userID, departmentID string) (bool, error)
	ListForAdmin(ctx context.Context) ([]AdminRow, error)
	// WithHOD is the IDs of Departments with an HOD role in effect.
	WithHOD(ctx context.Context) (map[string]bool, error)
}

// AdminRow is one Department as the admin's Departments screen shows it:
// its HOD from the Role assignment in effect, if any, and its counts.
type AdminRow struct {
	ID          string
	Code        string
	Name        string
	Description *string
	HODUserID   *string
	HODFullName *string
	HODUsername *string
	Students    int
	Staff       int
}

type Repositories struct {
	Departments Repository
	AuditLogs   auth.AuditLogRepository
}

type UnitOfWork interface {
	WithinTransaction(ctx context.Context, fn func(Repositories) error) error
}
