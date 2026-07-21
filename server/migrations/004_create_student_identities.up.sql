-- Migration: Create student_identities table
-- Source: docs/database-design.md § student_identities
-- Dependencies: users, departments

CREATE TABLE IF NOT EXISTS student_identities (
  user_id        UUID PRIMARY KEY REFERENCES users(id),
  usn            TEXT UNIQUE NOT NULL,
  department_id  UUID REFERENCES departments(id),
  batch_year     INT NOT NULL,
  admission_year INT,
  roll_number    TEXT,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Case-insensitive unique USN (prevents 4mn20ec002 vs 4MN20EC002 collisions)
-- USN is normalized to uppercase at application layer before storage
-- Source: docs/database-design.md § Important Indexes
CREATE UNIQUE INDEX IF NOT EXISTS idx_student_identities_usn ON student_identities (lower(usn));
