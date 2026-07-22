package db

import (
	"reflect"
	"testing"
	"testing/fstest"
)

func TestMigrationFilesReturnsSortedUpMigrations(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"002_second.up.sql":     &fstest.MapFile{Data: []byte("SELECT 1;")},
		"001_first.up.sql":      &fstest.MapFile{Data: []byte("SELECT 1;")},
		"003_ignore.down.sql":   &fstest.MapFile{Data: []byte("SELECT 1;")},
	}

	files, err := migrationFiles(fsys)
	if err != nil {
		t.Fatalf("read migration files: %v", err)
	}
	want := []string{"001_first.up.sql", "002_second.up.sql"}
	if !reflect.DeepEqual(files, want) {
		t.Fatalf("migration files = %v, want %v", files, want)
	}
}
