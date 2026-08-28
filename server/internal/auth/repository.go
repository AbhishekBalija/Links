package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var errActivationTokenUnavailable = errors.New("activation token is unavailable")
var errRefreshTokenUnavailable = errors.New("refresh token is unavailable")

// GormAuthUnitOfWork creates transaction-scoped auth repositories.
type GormAuthUnitOfWork struct {
	db *gorm.DB
}

func NewGormAuthUnitOfWork(db *gorm.DB) *GormAuthUnitOfWork {
	return &GormAuthUnitOfWork{db: db}
}

func (u *GormAuthUnitOfWork) WithinTransaction(ctx context.Context, fn func(AuthRepositories) error) error {
	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(AuthRepositories{
			Users:         NewGormUserRepository(tx),
			RefreshTokens: NewGormRefreshTokenRepository(tx),
			Activations:   NewGormActivationTokenRepository(tx),
			AuditLogs:     NewGormAuditLogRepository(tx),
		})
	})
}

// GormUserRepository implements UserRepository using GORM.
type GormUserRepository struct {
	db *gorm.DB
}

// NewGormUserRepository creates a new GormUserRepository.
func NewGormUserRepository(db *gorm.DB) *GormUserRepository {
	return &GormUserRepository{db: db}
}

func (r *GormUserRepository) Create(ctx context.Context, user *User) error {
	if user.ID == "" {
		user.ID = uuid.New().String()
	}
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *GormUserRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	err := r.db.WithContext(ctx).Where("lower(email) = lower(?)", email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, err
}

func (r *GormUserRepository) FindByID(ctx context.Context, id string) (*User, error) {
	var user User
	err := r.db.WithContext(ctx).
		Preload("Profile").
		Preload("StudentIdentity").
		First(&user, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, err
}

func (r *GormUserRepository) FindByIDForUpdate(ctx context.Context, id string) (*User, error) {
	var user User
	err := r.db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Preload("Profile").
		Preload("StudentIdentity").
		First(&user, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, err
}

func (r *GormUserRepository) Update(ctx context.Context, user *User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *GormUserRepository) UpdateStatus(ctx context.Context, id string, status UserStatus) error {
	return r.db.WithContext(ctx).Model(&User{}).Where("id = ?", id).Update("status", string(status)).Error
}

func (r *GormUserRepository) FindEmailByUserID(ctx context.Context, userID string) (*string, error) {
	var email *string
	err := r.db.WithContext(ctx).Model(&User{}).Where("id = ?", userID).Select("email").Scan(&email).Error
	return email, err
}

func (r *GormUserRepository) FindPhoneByUserID(ctx context.Context, userID string) (*string, error) {
	var phone *string
	err := r.db.WithContext(ctx).Model(&User{}).Where("id = ?", userID).Select("phone").Scan(&phone).Error
	return phone, err
}

func (r *GormUserRepository) FindDepartmentByCode(ctx context.Context, code string) (*Department, error) {
	var dept Department
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&dept).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &dept, err
}

func (r *GormUserRepository) CreateProfile(ctx context.Context, profile *Profile) error {
	return r.db.WithContext(ctx).Create(profile).Error
}

func (r *GormUserRepository) CreateStudentIdentity(ctx context.Context, identity *StudentIdentity) error {
	return r.db.WithContext(ctx).Create(identity).Error
}

func (r *GormUserRepository) GetRoleAssignments(ctx context.Context, userID string) ([]RoleAssignment, error) {
	var roles []RoleAssignment
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&roles).Error
	return roles, err
}

func (r *GormUserRepository) FindPendingUsers(ctx context.Context) ([]User, error) {
	var users []User
	err := r.db.WithContext(ctx).
		Preload("Profile").
		Preload("StudentIdentity").
		Where("status = ?", UserStatusPending).
		Order("created_at asc").
		Find(&users).Error
	return users, err
}

func (r *GormUserRepository) CreateRoleAssignment(ctx context.Context, ra *RoleAssignment) error {
	if ra.ID == "" {
		ra.ID = uuid.New().String()
	}
	return r.db.WithContext(ctx).Create(ra).Error
}

// GormRefreshTokenRepository implements RefreshTokenRepository using GORM.
type GormRefreshTokenRepository struct {
	db *gorm.DB
}

// NewGormRefreshTokenRepository creates a new GormRefreshTokenRepository.
func NewGormRefreshTokenRepository(db *gorm.DB) *GormRefreshTokenRepository {
	return &GormRefreshTokenRepository{db: db}
}

func (r *GormRefreshTokenRepository) Create(ctx context.Context, token *RefreshToken) error {
	if token.ID == "" {
		token.ID = uuid.New().String()
	}
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *GormRefreshTokenRepository) FindByHash(ctx context.Context, hash string) (*RefreshToken, error) {
	var token RefreshToken
	err := r.db.WithContext(ctx).Where("token_hash = ?", hash).First(&token).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &token, err
}

func (r *GormRefreshTokenRepository) RevokeByHash(ctx context.Context, hash string) error {
	return r.db.WithContext(ctx).Model(&RefreshToken{}).Where("token_hash = ?", hash).Update("revoked_at", time.Now()).Error
}

func (r *GormRefreshTokenRepository) RevokeIfActive(ctx context.Context, hash string) error {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&RefreshToken{}).
		Where("token_hash = ? AND revoked_at IS NULL AND expires_at > ?", hash, now).
		Update("revoked_at", now)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("%w: token already revoked, expired, or missing", errRefreshTokenUnavailable)
	}
	return nil
}

func (r *GormRefreshTokenRepository) RevokeAllByUserID(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).Model(&RefreshToken{}).Where("user_id = ?", userID).Update("revoked_at", time.Now()).Error
}

// GormActivationTokenRepository implements ActivationTokenRepository.
type GormActivationTokenRepository struct {
	db *gorm.DB
}

func NewGormActivationTokenRepository(db *gorm.DB) *GormActivationTokenRepository {
	return &GormActivationTokenRepository{db: db}
}

func (r *GormActivationTokenRepository) Create(ctx context.Context, token *AccountActivationToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *GormActivationTokenRepository) FindLatestByUserID(ctx context.Context, userID string) (*AccountActivationToken, error) {
	var token AccountActivationToken
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at desc").First(&token).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &token, err
}

func (r *GormActivationTokenRepository) RevokeAllUnusedByUserID(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).Model(&AccountActivationToken{}).
		Where("user_id = ? AND used_at IS NULL", userID).
		UpdateColumn("used_at", time.Now()).
		Error
}

func (r *GormActivationTokenRepository) FindByHash(ctx context.Context, hash string) (*AccountActivationToken, error) {
	var token AccountActivationToken
	err := r.db.WithContext(ctx).Where("token_hash = ?", hash).First(&token).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &token, err
}

func (r *GormActivationTokenRepository) MarkUsed(ctx context.Context, id string) error {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&AccountActivationToken{}).
		Where("id = ? AND used_at IS NULL AND expires_at > ?", id, now).
		Update("used_at", now)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("%w: token already used, expired, or missing", errActivationTokenUnavailable)
	}
	return nil
}

// GormAuditLogRepository implements AuditLogRepository using GORM.
type GormAuditLogRepository struct {
	db *gorm.DB
}

func NewGormAuditLogRepository(db *gorm.DB) *GormAuditLogRepository {
	return &GormAuditLogRepository{db: db}
}

func (r *GormAuditLogRepository) Create(ctx context.Context, log *AuditLog) error {
	if log.ID == "" {
		log.ID = uuid.New().String()
	}
	if log.Metadata != nil {
		b, err := json.Marshal(log.Metadata)
		if err != nil {
			return err
		}
		s := string(b)
		log.MetadataJSON = &s
	}
	return r.db.WithContext(ctx).Create(log).Error
}
