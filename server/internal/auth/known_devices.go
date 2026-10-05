package auth

// Known devices (#178 follow-up): a browser that signed in to an account
// with an email code is remembered for it. Its own code requests then get an
// allowance of their own, so nobody else, even on the same campus Wi-Fi, can
// use it up and stop the person getting codes.

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	// DeviceCookieName holds the browser's device token. It is only sent to
	// the sign-in routes.
	DeviceCookieName = "links_device"
	DeviceCookiePath = "/api/v1/auth"
	// deviceTTL is how long a browser stays known without signing in again.
	deviceTTL = 180 * 24 * time.Hour
	// devicesKept is how many browsers an account keeps known; signing in on
	// another forgets the one used longest ago.
	devicesKept = 10
)

// DeviceMaxAge is the cookie's lifetime in seconds.
var DeviceMaxAge = int(deviceTTL.Seconds())

type KnownDevice struct {
	ID         string    `gorm:"column:id;primaryKey"`
	UserID     string    `gorm:"column:user_id"`
	TokenHash  string    `gorm:"column:token_hash"`
	CreatedAt  time.Time `gorm:"column:created_at"`
	LastUsedAt time.Time `gorm:"column:last_used_at"`
	ExpiresAt  time.Time `gorm:"column:expires_at"`
}

func (KnownDevice) TableName() string { return "known_devices" }

type KnownDeviceRepository interface {
	// FindValid returns the unexpired device with this token hash, or nil.
	FindValid(ctx context.Context, tokenHash string, now time.Time) (*KnownDevice, error)
	Create(ctx context.Context, device *KnownDevice) error
	// Touch marks the device used and keeps it known for another deviceTTL.
	Touch(ctx context.Context, id string, now time.Time) error
	// KeepNewest forgets all but the account's most recently used devices,
	// and any that have expired.
	KeepNewest(ctx context.Context, userID string, keep int, now time.Time) error
}

type GormKnownDeviceRepository struct {
	db *gorm.DB
}

func NewGormKnownDeviceRepository(db *gorm.DB) *GormKnownDeviceRepository {
	return &GormKnownDeviceRepository{db: db}
}

func (r *GormKnownDeviceRepository) FindValid(ctx context.Context, tokenHash string, now time.Time) (*KnownDevice, error) {
	var devices []KnownDevice
	err := r.db.WithContext(ctx).Where("token_hash = ? AND expires_at > ?", tokenHash, now).Limit(1).Find(&devices).Error
	if err != nil || len(devices) == 0 {
		return nil, err
	}
	return &devices[0], nil
}

func (r *GormKnownDeviceRepository) Create(ctx context.Context, device *KnownDevice) error {
	return r.db.WithContext(ctx).Create(device).Error
}

func (r *GormKnownDeviceRepository) Touch(ctx context.Context, id string, now time.Time) error {
	return r.db.WithContext(ctx).Model(&KnownDevice{}).Where("id = ?", id).
		Updates(map[string]any{"last_used_at": now, "expires_at": now.Add(deviceTTL)}).Error
}

func (r *GormKnownDeviceRepository) KeepNewest(ctx context.Context, userID string, keep int, now time.Time) error {
	return r.db.WithContext(ctx).Exec(`
		DELETE FROM known_devices
		WHERE user_id = ? AND (expires_at <= ? OR id NOT IN (
			SELECT id FROM known_devices WHERE user_id = ? ORDER BY last_used_at DESC, id LIMIT ?
		))`, userID, now, userID, keep).Error
}

// knownDevice returns the device a token names when it is known for this
// user, or nil. A missing, made-up or expired token, or another account's
// device, is simply not known.
func (s *authService) knownDevice(ctx context.Context, repos AuthRepositories, token, userID string, now time.Time) (*KnownDevice, error) {
	if token == "" || userID == "" {
		return nil, nil
	}
	device, err := repos.KnownDevices.FindValid(ctx, s.keyedHash("device", token), now)
	if err != nil {
		return nil, fmt.Errorf("find device: %w", err)
	}
	if device == nil || device.UserID != userID {
		return nil, nil
	}
	return device, nil
}

// rememberDevice keeps this browser known for the user after a code sign-in
// and returns the token for its cookie: the same one when it was already
// known, a new one otherwise.
func (s *authService) rememberDevice(ctx context.Context, repos AuthRepositories, token, userID string, now time.Time) (string, error) {
	device, err := s.knownDevice(ctx, repos, token, userID, now)
	if err != nil {
		return "", err
	}
	if device != nil {
		return token, repos.KnownDevices.Touch(ctx, device.ID, now)
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate device token: %w", err)
	}
	token = base64.RawURLEncoding.EncodeToString(raw)
	device = &KnownDevice{
		ID:         uuid.NewString(),
		UserID:     userID,
		TokenHash:  s.keyedHash("device", token),
		CreatedAt:  now,
		LastUsedAt: now,
		ExpiresAt:  now.Add(deviceTTL),
	}
	if err := repos.KnownDevices.Create(ctx, device); err != nil {
		return "", fmt.Errorf("remember device: %w", err)
	}
	if err := repos.KnownDevices.KeepNewest(ctx, userID, devicesKept, now); err != nil {
		return "", fmt.Errorf("forget old devices: %w", err)
	}
	return token, nil
}
