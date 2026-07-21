package auth

import (
	"context"
	"time"
)

// User represents the users table per docs/database-design.md § users.
// Per docs/backend-standards.md § Domain State: use typed statuses.
type User struct {
	ID           string     `gorm:"column:id;primaryKey"`
	Email        *string    `gorm:"column:email"`
	Phone        *string    `gorm:"column:phone"`
	PasswordHash string     `gorm:"column:password_hash"`
	Status       UserStatus `gorm:"column:status"`
	IsVerified   bool       `gorm:"column:is_verified"`
	CreatedBy    *string    `gorm:"column:created_by"`
	CreatedAt    time.Time  `gorm:"column:created_at"`
	UpdatedAt    time.Time  `gorm:"column:updated_at"`
}

// TableName returns the database table name.
func (User) TableName() string { return "users" }

// UserStatus represents the user lifecycle states per docs/auth.md § User Statuses.
//
//	[*] → pending → active → suspended → active
//	                pending → rejected
type UserStatus string

const (
	UserStatusUnknown   UserStatus = ""
	UserStatusPending   UserStatus = "pending"
	UserStatusActive    UserStatus = "active"
	UserStatusSuspended UserStatus = "suspended"
	UserStatusRejected  UserStatus = "rejected"
)

// CanLogin returns true if the user status allows login.
// Per docs/auth.md: only active users can log in.
func (s UserStatus) CanLogin() bool {
	return s == UserStatusActive
}

// RoleAssignment represents the role_assignments table
// per docs/database-design.md § role_assignments.
type RoleAssignment struct {
	ID         string    `gorm:"column:id;primaryKey"`
	UserID     string    `gorm:"column:user_id"`
	Role       Role      `gorm:"column:role"`
	ScopeType  ScopeType `gorm:"column:scope_type"`
	ScopeID    *string   `gorm:"column:scope_id"`
	AssignedBy *string   `gorm:"column:assigned_by"`
	StartsAt   time.Time `gorm:"column:starts_at"`
	EndsAt     *time.Time `gorm:"column:ends_at"`
	CreatedAt  time.Time `gorm:"column:created_at"`
}

// TableName returns the database table name.
func (RoleAssignment) TableName() string { return "role_assignments" }

// Role represents the base roles per docs/auth.md § Roles.
type Role string

const (
	RoleStudent            Role = "student"
	RoleStudentCoordinator Role = "student_coordinator"
	RoleFaculty            Role = "faculty"
	RoleHOD                Role = "hod"
	RolePlacementOfficer   Role = "placement_officer"
	RolePrincipal          Role = "principal"
	RoleAlumni             Role = "alumni"
	RoleClubOrganizer      Role = "club_organizer"
	RoleAdmin              Role = "admin"
)

// ScopeType represents role assignment scope types per ADR-005.
type ScopeType string

const (
	ScopeGlobal     ScopeType = "global"
	ScopeDepartment ScopeType = "department"
	ScopeClub       ScopeType = "club"
)

// RefreshToken represents a hashed refresh token stored in the database.
// Per docs/auth.md § Token Strategy: "Store refresh tokens hashed",
// "Rotate refresh tokens on every refresh".
type RefreshToken struct {
	ID        string     `gorm:"column:id;primaryKey"`
	UserID    string     `gorm:"column:user_id"`
	TokenHash string     `gorm:"column:token_hash"`
	ExpiresAt time.Time  `gorm:"column:expires_at"`
	RevokedAt *time.Time `gorm:"column:revoked_at"`
	CreatedAt time.Time  `gorm:"column:created_at"`
}

// TableName returns the database table name.
func (RefreshToken) TableName() string { return "refresh_tokens" }

// IsExpired returns true if the token has passed its expiry time.
func (t RefreshToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// IsRevoked returns true if the token has been revoked.
func (t RefreshToken) IsRevoked() bool {
	return t.RevokedAt != nil
}

// Profile represents the profiles table per docs/database-design.md § profiles.
// Only the fields needed for auth/request-access are included here;
// the full profile CRUD belongs in the profiles module.
type Profile struct {
	UserID    string    `gorm:"column:user_id;primaryKey"`
	Username  string    `gorm:"column:username"`
	FullName  string    `gorm:"column:full_name"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (Profile) TableName() string { return "profiles" }

// Department represents a department.
type Department struct {
	ID   string `gorm:"column:id;primaryKey"`
	Code string `gorm:"column:code;uniqueIndex"`
	Name string `gorm:"column:name"`
}

func (Department) TableName() string { return "departments" }

// StudentIdentity represents the student_identities table.
type StudentIdentity struct {
	UserID        string `gorm:"column:user_id;primaryKey"`
	USN           string `gorm:"column:usn;uniqueIndex;not null"`
	DepartmentID  string `gorm:"column:department_id"`
	BatchYear     int    `gorm:"column:batch_year;not null"`
	AdmissionYear *int   `gorm:"column:admission_year"`
	RollNumber    string `gorm:"column:roll_number"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (StudentIdentity) TableName() string { return "student_identities" }

// AccountActivationToken represents the account_activation_tokens table.
type AccountActivationToken struct {
	ID        string     `gorm:"column:id;primaryKey"`
	UserID    string     `gorm:"column:user_id;not null;index"`
	TokenHash string     `gorm:"column:token_hash;not null"`
	Purpose   string     `gorm:"column:purpose;not null;default:'activate'"`
	ExpiresAt time.Time  `gorm:"column:expires_at;not null"`
	UsedAt    *time.Time `gorm:"column:used_at"`
	CreatedAt time.Time  `gorm:"column:created_at;not null;default:now()"`
}

func (AccountActivationToken) TableName() string { return "account_activation_tokens" }

// PasswordHasher defines the interface for password hashing.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(password, hash string) (bool, error)
}

// AuthService defines the business logic interface for authentication.
type AuthService interface {
	RequestAccess(ctx context.Context, input RequestAccessInput) (*RequestAccessResponse, error)
	Login(ctx context.Context, input LoginInput) (*LoginResponse, string, error)
	Refresh(ctx context.Context, refreshTokenRaw string) (*RefreshResponse, string, error)
	Logout(ctx context.Context, refreshTokenRaw string) error
	ActivateAccount(ctx context.Context, token, password string) error
}

// UserRepository defines the interface for user persistence operations.
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByID(ctx context.Context, id string) (*User, error)
	Update(ctx context.Context, user *User) error
	UpdateStatus(ctx context.Context, id string, status UserStatus) error
	FindDepartmentByCode(ctx context.Context, code string) (*Department, error)
	CreateProfile(ctx context.Context, profile *Profile) error
	CreateStudentIdentity(ctx context.Context, identity *StudentIdentity) error
	GetRoleAssignments(ctx context.Context, userID string) ([]RoleAssignment, error)
}

// RefreshTokenRepository defines the interface for refresh token persistence operations.
type RefreshTokenRepository interface {
	Create(ctx context.Context, token *RefreshToken) error
	FindByHash(ctx context.Context, hash string) (*RefreshToken, error)
	RevokeByHash(ctx context.Context, hash string) error
	RevokeAllByUserID(ctx context.Context, userID string) error
}

// ActivationTokenRepository defines the interface for account activation tokens.
type ActivationTokenRepository interface {
	FindByHash(ctx context.Context, hash string) (*AccountActivationToken, error)
	MarkUsed(ctx context.Context, id string) error
}
