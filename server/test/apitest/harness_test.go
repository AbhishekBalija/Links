package apitest

import "testing"

// Every test schema must get the full set of constraints, even while other
// test schemas (or a migrated public schema) exist in the same database.
func TestEverySchemaGetsForeignKeysFromMigrations(t *testing.T) {
	first := New(t)
	second := New(t)

	for _, h := range []*Harness{first, second} {
		var names []string
		err := h.DB().Raw(`
			SELECT c.conname
			FROM pg_constraint c
			JOIN pg_namespace n ON n.oid = c.connamespace
			WHERE n.nspname = current_schema()
			  AND c.conname IN ('fk_users_created_by', 'fk_departments_hod_user_id')
			ORDER BY c.conname`).Scan(&names).Error
		if err != nil {
			t.Fatalf("read constraints: %v", err)
		}
		if len(names) != 2 {
			t.Errorf("schema has foreign keys %v, want both fk_departments_hod_user_id and fk_users_created_by", names)
		}
	}
}
