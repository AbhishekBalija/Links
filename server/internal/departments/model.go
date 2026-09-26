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
	Create(ctx context.Context, department *Department) error
	Update(ctx context.Context, department *Department) error
	Delete(ctx context.Context, department *Department) error
	IsReferenced(ctx context.Context, departmentID string) (bool, error)
	CanAssignHOD(ctx context.Context, userID, departmentID string) (bool, error)
}

type Repositories struct {
	Departments Repository
	AuditLogs   auth.AuditLogRepository
}

type UnitOfWork interface {
	WithinTransaction(ctx context.Context, fn func(Repositories) error) error
}
