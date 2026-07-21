package auth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

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
	err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error
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
	return r.db.WithContext(ctx).Model(&RefreshToken{}).Where("token_hash = ?", hash).Update("revoked_at", "NOW()").Error
}

func (r *GormRefreshTokenRepository) RevokeAllByUserID(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).Model(&RefreshToken{}).Where("user_id = ?", userID).Update("revoked_at", "NOW()").Error
}

// GormActivationTokenRepository implements ActivationTokenRepository.
type GormActivationTokenRepository struct {
	db *gorm.DB
}

func NewGormActivationTokenRepository(db *gorm.DB) *GormActivationTokenRepository {
	return &GormActivationTokenRepository{db: db}
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
	return r.db.WithContext(ctx).Model(&AccountActivationToken{}).Where("id = ?", id).Update("used_at", now).Error
}