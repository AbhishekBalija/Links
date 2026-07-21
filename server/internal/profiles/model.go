package profiles

import (
	"context"
	"time"
)

type Profile struct {
	UserID               string     `gorm:"column:user_id;primaryKey"`
	Username             string     `gorm:"column:username;uniqueIndex"`
	FullName             string     `gorm:"column:full_name"`
	Headline             *string    `gorm:"column:headline"`
	Bio                  *string    `gorm:"column:bio"`
	AvatarURL            *string    `gorm:"column:avatar_url"`
	PublicProfileEnabled bool       `gorm:"column:public_profile_enabled"`
	ShowEmail            bool       `gorm:"column:show_email"`
	ShowPhone            bool       `gorm:"column:show_phone"`
	LinkedInURL          *string    `gorm:"column:linkedin_url"`
	GitHubURL            *string    `gorm:"column:github_url"`
	PortfolioURL         *string    `gorm:"column:portfolio_url"`
	CreatedAt            time.Time  `gorm:"column:created_at"`
	UpdatedAt            time.Time  `gorm:"column:updated_at"`
}

func (Profile) TableName() string { return "profiles" }

type ProfileRepository interface {
	FindByUserID(ctx context.Context, userID string) (*Profile, error)
	FindByUsername(ctx context.Context, username string) (*Profile, error)
	Update(ctx context.Context, profile *Profile) error
}
