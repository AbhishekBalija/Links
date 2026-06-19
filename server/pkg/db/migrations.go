package db

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gorm.io/gorm"
)

// RunMigrations executes all migration files in order
func RunMigrations(db *gorm.DB, migrationsPath string) error {
	// Get all .up.sql files
	files, err := os.ReadDir(migrationsPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("Migration directory not found, skipping migrations: %s", migrationsPath)
			return nil
		}
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var migrations []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".up.sql") {
			migrations = append(migrations, file.Name())
		}
	}

	// Sort to ensure they run in order
	sort.Strings(migrations)

	for _, migration := range migrations {
		fullPath := filepath.Join(migrationsPath, migration)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", migration, err)
		}

		if err := db.Exec(string(content)).Error; err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", migration, err)
		}

		log.Printf("Migration applied: %s", migration)
	}

	return nil
}
