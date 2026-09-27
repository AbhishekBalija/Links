package db

import (
	"context"
	"fmt"
	"hash/fnv"
	"io/fs"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

const migrationTable = "schema_migrations"

// RunMigrations records every successful migration so startup never reapplies it.
func RunMigrations(ctx context.Context, database *gorm.DB, fsys fs.FS) error {
	if err := database.WithContext(ctx).Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version text PRIMARY KEY,
			applied_at timestamptz NOT NULL
		)
	`).Error; err != nil {
		return fmt.Errorf("create migration table: %w", err)
	}

	migrations, err := migrationFiles(fsys)
	if err != nil {
		return err
	}

	// Read what's already applied in one query, so a normal startup costs one
	// round trip instead of a transaction per migration. Serverless cold
	// starts run this on every boot, and far from the database those
	// per-migration round trips added up past the startup deadline.
	applied, err := appliedMigrations(ctx, database)
	if err != nil {
		return err
	}

	for _, migration := range pendingMigrations(migrations, applied) {
		// applyMigration still locks and re-checks, so two instances starting
		// together apply each pending migration once.
		if err := applyMigration(ctx, database, fsys, migration); err != nil {
			return err
		}
	}
	return nil
}

func appliedMigrations(ctx context.Context, database *gorm.DB) (map[string]bool, error) {
	var versions []string
	if err := database.WithContext(ctx).Table(migrationTable).Pluck("version", &versions).Error; err != nil {
		return nil, fmt.Errorf("read applied migrations: %w", err)
	}
	applied := make(map[string]bool, len(versions))
	for _, version := range versions {
		applied[version] = true
	}
	return applied, nil
}

// pendingMigrations keeps the migrations not yet applied, in their order.
func pendingMigrations(migrations []string, applied map[string]bool) []string {
	pending := []string{}
	for _, migration := range migrations {
		if !applied[migration] {
			pending = append(pending, migration)
		}
	}
	return pending
}

func migrationFiles(fsys fs.FS) ([]string, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("read migrations: %w", err)
	}

	migrations := []string{}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".up.sql") {
			migrations = append(migrations, entry.Name())
		}
	}
	sort.Strings(migrations)
	return migrations, nil
}

func applyMigration(ctx context.Context, database *gorm.DB, fsys fs.FS, migration string) error {
	content, err := fs.ReadFile(fsys, migration)
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
