package db

import (
	"context"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

const migrationTable = "schema_migrations"

// RunMigrations records every successful migration so startup never reapplies it.
func RunMigrations(ctx context.Context, database *gorm.DB, migrationsPath string) error {
	if err := database.WithContext(ctx).Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version text PRIMARY KEY,
			applied_at timestamptz NOT NULL
		)
	`).Error; err != nil {
		return fmt.Errorf("create migration table: %w", err)
	}

	migrations, err := migrationFiles(migrationsPath)
	if err != nil {
		return err
	}

	for _, migration := range migrations {
		if err := applyMigration(ctx, database, migrationsPath, migration); err != nil {
			return err
		}
	}
	return nil
}

func migrationFiles(migrationsPath string) ([]string, error) {
	files, err := os.ReadDir(migrationsPath)
	if err != nil {
		return nil, fmt.Errorf("read migrations directory: %w", err)
	}

	migrations := []string{}
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".up.sql") {
			migrations = append(migrations, file.Name())
		}
	}
	sort.Strings(migrations)
	return migrations, nil
}

func applyMigration(ctx context.Context, database *gorm.DB, migrationsPath, migration string) error {
	content, err := os.ReadFile(filepath.Join(migrationsPath, migration))
	if err != nil {
		return fmt.Errorf("read migration %q: %w", migration, err)
	}

	return database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", migrationLockKey(migration)).Error; err != nil {
			return fmt.Errorf("acquire migration lock for %q: %w", migration, err)
		}

		var count int64
		if err := tx.Table(migrationTable).Where("version = ?", migration).Count(&count).Error; err != nil {
			return fmt.Errorf("check migration %q: %w", migration, err)
		}
		if count > 0 {
			return nil
		}
		if err := tx.Exec(string(content)).Error; err != nil {
			return fmt.Errorf("execute migration %q: %w", migration, err)
		}
		if err := tx.Table(migrationTable).Create(map[string]any{
			"version":    migration,
			"applied_at": time.Now().UTC(),
		}).Error; err != nil {
			return fmt.Errorf("record migration %q: %w", migration, err)
		}
		return nil
	})
}

func migrationLockKey(migration string) int64 {
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(migration))
	return int64(hash.Sum64())
}
