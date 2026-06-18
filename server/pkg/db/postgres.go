package db

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

// User model for the database
type User struct {
	ID           uint   `gorm:"primaryKey"`
	Username     string `gorm:"unique;not null"`
	Email        string `gorm:"unique;not null"`
	PasswordHash string `gorm:"column:password_hash;not null"`
	FullName     string
	Role         string    `gorm:"default:'student'"` // student, faculty, admin, alumni
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`
}

// InitDB connects to PostgreSQL and runs migrations
func InitDB(dsn string) error {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Run migrations from migration files
	if err := RunMigrations(db, "migrations"); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	DB = db
	log.Println("✓ Database connected and migrations applied")
	return nil
}
