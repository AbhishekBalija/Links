package db

import (
	"reflect"
	"testing"
	"testing/fstest"
)

func TestMigrationFilesReturnsSortedUpMigrations(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"002_second.up.sql":   &fstest.MapFile{Data: []byte("SELECT 1;")},
		"001_first.up.sql":    &fstest.MapFile{Data: []byte("SELECT 1;")},
		"003_ignore.down.sql": &fstest.MapFile{Data: []byte("SELECT 1;")},
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

func TestPendingMigrationsSkipsAppliedOnesInOrder(t *testing.T) {
	t.Parallel()

	all := []string{"001_a.up.sql", "002_b.up.sql", "003_c.up.sql", "004_d.up.sql"}
	applied := map[string]bool{"001_a.up.sql": true, "003_c.up.sql": true}

	got := pendingMigrations(all, applied)
	want := []string{"002_b.up.sql", "004_d.up.sql"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("pending = %v, want %v", got, want)
	}
	if none := pendingMigrations(all, map[string]bool{"001_a.up.sql": true, "002_b.up.sql": true, "003_c.up.sql": true, "004_d.up.sql": true}); len(none) != 0 {
		t.Fatalf("pending = %v, want none when everything is applied", none)
	}
}
