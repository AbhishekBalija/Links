package auth

import (
	"context"
	"time"
)

// User represents the users table per docs/database-design.md § users.
// Per docs/backend-standards.md § Domain State: use typed statuses.
type User struct {
	ID              string           `gorm:"column:id;primaryKey"`
	Email           *string          `gorm:"column:email"`
	Phone           *string          `gorm:"column:phone"`
	PasswordHash    string           `gorm:"column:password_hash"`
	Status          UserStatus       `gorm:"column:status"`
	IsVerified      bool             `gorm:"column:is_verified"`
	CreatedBy       *string          `gorm:"column:created_by"`
	CreatedAt       time.Time        `gorm:"column:created_at"`
	UpdatedAt       time.Time        `gorm:"column:updated_at"`
	Profile         *Profile         `gorm:"foreignKey:UserID;references:ID"`
	StudentIdentity *StudentIdentity `gorm:"foreignKey:UserID;references:ID"`
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
	ID         string     `gorm:"column:id;primaryKey"`
	UserID     string     `gorm:"column:user_id"`
	Role       Role       `gorm:"column:role"`
	ScopeType  ScopeType  `gorm:"column:scope_type"`
	ScopeID    *string    `gorm:"column:scope_id"`
	AssignedBy *string    `gorm:"column:assigned_by"`
	StartsAt   time.Time  `gorm:"column:starts_at"`
	EndsAt     *time.Time `gorm:"column:ends_at"`
	CreatedAt  time.Time  `gorm:"column:created_at"`
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
// Auth creates the initial row; the profiles module owns full CRUD.
type Profile struct {
	UserID               string    `gorm:"column:user_id;primaryKey"`
	Username             string    `gorm:"column:username;uniqueIndex"`
	FullName             string    `gorm:"column:full_name"`
	Headline             *string   `gorm:"column:headline"`
	Bio                  *string   `gorm:"column:bio"`
	AvatarURL            *string   `gorm:"column:avatar_url"`
	PublicProfileEnabled bool      `gorm:"column:public_profile_enabled"`
	ShowEmail            bool      `gorm:"column:show_email"`
	ShowPhone            bool      `gorm:"column:show_phone"`
	LinkedInURL          *string   `gorm:"column:linkedin_url"`
	GitHubURL            *string   `gorm:"column:github_url"`
	PortfolioURL         *string   `gorm:"column:portfolio_url"`
	CreatedAt            time.Time `gorm:"column:created_at"`
	UpdatedAt            time.Time `gorm:"column:updated_at"`
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
	ResendActivation(ctx context.Context, email string) error
	GetMe(ctx context.Context, userID string) (*MeResponse, error)
	ReviewQueue(ctx context.Context) (*ReviewQueueResponse, error)
	VerifyUser(ctx context.Context, actorID, userID, scopeType, scopeID, note string) error
	UpdateUserStatus(ctx context.Context, actorID, userID, status, note string) error
}

// AuditLog represents the audit_logs table per docs/database-design.md § audit_logs.
type AuditLog struct {
	ID           string      `gorm:"column:id;primaryKey"`
	ActorID      *string     `gorm:"column:actor_id"`
	Action       string      `gorm:"column:action"`
	ResourceType string      `gorm:"column:resource_type"`
	ResourceID   *string     `gorm:"column:resource_id"`
	Metadata     interface{} `gorm:"-"`
	MetadataJSON *string     `gorm:"column:metadata"`
	IPAddress    *string     `gorm:"column:ip_address"`
	UserAgent    *string     `gorm:"column:user_agent"`
	CreatedAt    time.Time   `gorm:"column:created_at"`
}

func (AuditLog) TableName() string { return "audit_logs" }

// UserRepository defines the interface for user persistence operations.
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByID(ctx context.Context, id string) (*User, error)
	FindByIDForUpdate(ctx context.Context, id string) (*User, error)
	FindEmailByUserID(ctx context.Context, userID string) (*string, error)
	FindPhoneByUserID(ctx context.Context, userID string) (*string, error)
	FindPendingUsers(ctx context.Context) ([]User, error)
	Update(ctx context.Context, user *User) error
	UpdateStatus(ctx context.Context, id string, status UserStatus) error
	FindDepartmentByCode(ctx context.Context, code string) (*Department, error)
	CreateProfile(ctx context.Context, profile *Profile) error
	CreateStudentIdentity(ctx context.Context, identity *StudentIdentity) error
	GetRoleAssignments(ctx context.Context, userID string) ([]RoleAssignment, error)
	CreateRoleAssignment(ctx context.Context, ra *RoleAssignment) error
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
	Create(ctx context.Context, token *AccountActivationToken) error
	FindByHash(ctx context.Context, hash string) (*AccountActivationToken, error)
	FindLatestByUserID(ctx context.Context, userID string) (*AccountActivationToken, error)
	MarkUsed(ctx context.Context, id string) error
	RevokeAllUnusedByUserID(ctx context.Context, userID string) error
}

// AuditLogRepository defines the interface for audit log persistence.
type AuditLogRepository interface {
	Create(ctx context.Context, log *AuditLog) error
}

// AuthRepositories groups repositories that must share one database transaction.
type AuthRepositories struct {
	Users       UserRepository
	Activations ActivationTokenRepository
	AuditLogs   AuditLogRepository
}

// AuthUnitOfWork executes related auth writes atomically.
type AuthUnitOfWork interface {
	WithinTransaction(ctx context.Context, fn func(AuthRepositories) error) error
}
