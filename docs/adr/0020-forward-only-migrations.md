# Migrations only go forward

The API runs its embedded `*.up.sql` migrations on startup, in filename order, each once, under an advisory lock (`pkg/db/migrations.go`). It never runs a `.down.sql`. Some migrations (011 to 015) ship one anyway and most don't, which made it look as if rollback were supported when it isn't.

## Decision

- Migrations are forward-only. A mistake is fixed by a new migration, never by editing or reverting an applied one.
- A `.down.sql` is optional. When present it is a hand-run note for undoing that migration on a local or dev database, and nothing runs it. It is not required for new migrations.
- Before a risky production migration, rely on Neon's point-in-time restore or a branch taken just before deploying, not on down files.

## Considered options

- **Down migrations with a `migrate down` command:** rejected for now. On a single-college deployment with Neon branching, a tested restore is safer than rarely-run down scripts, and pre-1.0 the schema changes faster than downs would be maintained.

## Consequences

- Reviews check that each migration is safe to run twice on a partly migrated database, since there's no automatic undo.
- If LINKS is deployed for more colleges, revisit this with a real `down` command and tests for it.
