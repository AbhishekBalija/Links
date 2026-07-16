package db

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestMigrationFilesReturnsSortedUpMigrations(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	for _, name := range []string{"002_second.up.sql", "001_first.up.sql", "003_ignore.down.sql"} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte("SELECT 1;"), 0o600); err != nil {
			t.Fatalf("write test migration: %v", err)
		}
	}

	files, err := migrationFiles(directory)
	if err != nil {
		t.Fatalf("read migration files: %v", err)
	}
	want := []string{"001_first.up.sql", "002_second.up.sql"}
	if !reflect.DeepEqual(files, want) {
		t.Fatalf("migration files = %v, want %v", files, want)
	}
}
