package db

import (
	"context"
	"fmt"
	"io/fs"

	"github.com/AbhishekBalija/Links/server/pkg/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Database owns the application's database connection pool.
type Database struct {
	gormDB *gorm.DB
}

// New opens and configures a PostgreSQL connection.
func New(cfg config.Config) (*Database, error) {
	gormDB, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, fmt.Errorf("get database handle: %w", err)
	}
	sqlDB.SetMaxOpenConns(cfg.DatabasePool.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.DatabasePool.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.DatabasePool.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.DatabasePool.ConnMaxIdleTime)

	return &Database{gormDB: gormDB}, nil
}

// Migrate applies each unapplied SQL migration exactly once.
// Accepts any fs.FS (embed.FS, os.DirFS, or test fstest.MapFS) so the
// binary has zero filesystem dependency at runtime — all migrations are
// compiled in via //go:embed.
func (d *Database) Migrate(ctx context.Context, fsys fs.FS) error {
	if d == nil || d.gormDB == nil {
		return fmt.Errorf("database is not initialized")
	}
	return RunMigrations(ctx, d.gormDB, fsys)
}

// Ping verifies that PostgreSQL is available.
func (d *Database) Ping(ctx context.Context) error {
	if d == nil || d.gormDB == nil {
		return fmt.Errorf("database is not initialized")
	}
	sqlDB, err := d.gormDB.DB()
	if err != nil {
		return fmt.Errorf("get database handle: %w", err)
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	return nil
}

// GORM returns the underlying gorm.DB for use by repository layers.
func (d *Database) GORM() *gorm.DB {
	return d.gormDB
}

// Close closes the database pool during graceful shutdown.
func (d *Database) Close() error {
	if d == nil || d.gormDB == nil {
		return nil
	}
	sqlDB, err := d.gormDB.DB()
	if err != nil {
		return fmt.Errorf("get database handle: %w", err)
	}
	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("close database: %w", err)
	}
	return nil
}
