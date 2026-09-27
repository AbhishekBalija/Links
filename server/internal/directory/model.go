// Package directory lists LINKS members for other signed-in members: who is
// who, with their roles and Department, and contact details only where the
// member opted in (#12).
package directory

import (
	"context"

	"github.com/AbhishekBalija/Links/server/internal/auth"
)

// Member is one listed user, as read from the database. Email and phone are
// read here and dropped in the service unless the member opted in.
type Member struct {
	UserID                string  `gorm:"column:user_id"`
	Username              string  `gorm:"column:username"`
	FullName              string  `gorm:"column:full_name"`
	SortName              string  `gorm:"column:sort_name"`
	Headline              *string `gorm:"column:headline"`
	AvatarURL             *string `gorm:"column:avatar_url"`
	PublicProfileEnabled  bool    `gorm:"column:public_profile_enabled"`
	ShowEmail             bool    `gorm:"column:show_email"`
	ShowPhone             bool    `gorm:"column:show_phone"`
	Email                 *string `gorm:"column:email"`
	Phone                 *string `gorm:"column:phone"`
	StudentBatchYear      *int    `gorm:"column:student_batch_year"`
	StudentDepartmentCode *string `gorm:"column:student_department_code"`
	StudentDepartmentName *string `gorm:"column:student_department_name"`
}

// Grant is one Role assignment in effect now, with its Department when it is
// Department-scoped.
type Grant struct {
	UserID         string    `gorm:"column:user_id"`
	Role           auth.Role `gorm:"column:role"`
	DepartmentCode *string   `gorm:"column:department_code"`
	DepartmentName *string   `gorm:"column:department_name"`
}

// Department is the part of a Department the directory needs.
type Department struct {
	ID          string  `gorm:"column:id"`
	Code        string  `gorm:"column:code"`
	Name        string  `gorm:"column:name"`
	Description *string `gorm:"column:description"`
}

// Filter narrows the list; nil fields don't filter. All set fields must match.
type Filter struct {
	DepartmentID *string
	Role         *auth.Role
	BatchYear    *int
}

// Cursor marks the last member of the previous page in list order.
type Cursor struct {
	SortName string `json:"n"`
	UserID   string `json:"i"`
}

type Repository interface {
	List(ctx context.Context, filter Filter, after *Cursor, limit int) ([]Member, error)
	Count(ctx context.Context, filter Filter) (int, error)
	MemberByID(ctx context.Context, userID string) (*Member, error)
	Search(ctx context.Context, filter Filter, q string, limit int) ([]Member, error)
	Grants(ctx context.Context, userIDs []string) ([]Grant, error)
	DepartmentByCode(ctx context.Context, code string) (*Department, error)
	Staff(ctx context.Context, departmentID string) ([]Member, error)
	StudentsByBatch(ctx context.Context, departmentID string) ([]BatchCount, error)
	FacultyCount(ctx context.Context, departmentID string) (int, error)
}

// BatchCount is how many active students one Batch of a Department has.
type BatchCount struct {
	BatchYear int `gorm:"column:batch_year" json:"batch_year"`
	Count     int `gorm:"column:count" json:"count"`
}

// staffRoles are the Department-scoped roles that put someone on a
// Department's staff list.
var staffRoles = []auth.Role{auth.RoleHOD, auth.RolePlacementOfficer, auth.RoleFaculty}
