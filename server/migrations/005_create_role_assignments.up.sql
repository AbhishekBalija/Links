-- Migration: Create role_assignments table
-- Source: docs/database-design.md § role_assignments, docs/auth.md § Roles (ADR-005)
-- Dependencies: users

CREATE TABLE IF NOT EXISTS role_assignments (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     UUID NOT NULL REFERENCES users(id),
  role        TEXT NOT NULL,
  scope_type  TEXT NOT NULL,
  scope_id    UUID,
  assigned_by UUID REFERENCES users(id),
  starts_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  ends_at     TIMESTAMPTZ,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

  -- Roles from docs/auth.md § Roles
  CONSTRAINT chk_role_assignments_role CHECK (
    role IN (
      'student', 'student_coordinator', 'faculty', 'hod',
      'placement_officer', 'principal', 'alumni', 'club_organizer', 'admin'
    )
  ),

  -- Scope types per ADR-005: scoped RBAC
  CONSTRAINT chk_role_assignments_scope_type CHECK (
    scope_type IN ('global', 'department', 'club')
  ),
  CONSTRAINT chk_role_assignments_scope_id CHECK (
    (scope_type = 'global' AND scope_id IS NULL)
    OR (scope_type IN ('department', 'club') AND scope_id IS NOT NULL)
  )
);

-- Source: docs/database-design.md § Important Indexes
CREATE INDEX IF NOT EXISTS idx_role_assignments_user ON role_assignments (user_id);
CREATE INDEX IF NOT EXISTS idx_role_assignments_role_scope ON role_assignments (role, scope_type, scope_id);
