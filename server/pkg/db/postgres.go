package db

import (
	"context"
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

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
	log.Println("Database connected and migrations applied")
	return nil
}

// Ping verifies that the database connection is alive.
func Ping(ctx context.Context) error {
	if DB == nil {
		return fmt.Errorf("database is not initialized")
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("get database handle: %w", err)
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	return nil
}
