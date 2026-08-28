package profiles

import (
	"context"
	"errors"

	"github.com/AbhishekBalija/Links/server/internal/auth"
	"gorm.io/gorm"
)

type GormUnitOfWork struct {
	db *gorm.DB
}

func NewGormUnitOfWork(db *gorm.DB) *GormUnitOfWork {
	return &GormUnitOfWork{db: db}
}

func (u *GormUnitOfWork) WithinTransaction(ctx context.Context, fn func(Repositories) error) error {
	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(Repositories{
			Profiles:  NewGormProfileRepository(tx),
			AuditLogs: auth.NewGormAuditLogRepository(tx),
		})
	})
}

type GormProfileRepository struct {
	db *gorm.DB
}

func NewGormProfileRepository(db *gorm.DB) *GormProfileRepository {
	return &GormProfileRepository{db: db}
}

func (r *GormProfileRepository) FindByUserID(ctx context.Context, userID string) (*Profile, error) {
	var p Profile
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &p, err
}

func (r *GormProfileRepository) FindByUsername(ctx context.Context, username string) (*Profile, error) {
	var p Profile
	err := r.db.WithContext(ctx).Where("lower(username) = lower(?)", username).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &p, err
}

func (r *GormProfileRepository) Update(ctx context.Context, profile *Profile) error {
	return r.db.WithContext(ctx).Save(profile).Error
}
