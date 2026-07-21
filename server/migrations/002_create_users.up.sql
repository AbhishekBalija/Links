-- Migration: Create users table
-- Source: docs/database-design.md § users, docs/auth.md § User Statuses
-- Dependencies: departments (for deferred hod_user_id FK)

CREATE TABLE IF NOT EXISTS users (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email         TEXT,
  phone         TEXT,
  password_hash TEXT NOT NULL,
  status        TEXT NOT NULL DEFAULT 'pending',
  is_verified   BOOLEAN NOT NULL DEFAULT false,
  created_by    UUID,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

  CONSTRAINT chk_users_status CHECK (status IN ('pending', 'active', 'suspended', 'rejected'))
);

-- Case-insensitive unique email; NULLs allowed for admin-created accounts
-- Source: docs/database-design.md § Important Indexes
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users (lower(email)) WHERE email IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_users_status ON users (status);

-- Self-referencing FK: tracks which admin/HOD created this user
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_users_created_by') THEN
    ALTER TABLE users ADD CONSTRAINT fk_users_created_by FOREIGN KEY (created_by) REFERENCES users(id);
  END IF;
END $$;

-- Deferred FK from migration 001: departments.hod_user_id → users.id
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_departments_hod_user_id') THEN
    ALTER TABLE departments ADD CONSTRAINT fk_departments_hod_user_id FOREIGN KEY (hod_user_id) REFERENCES users(id);
  END IF;
END $$;
