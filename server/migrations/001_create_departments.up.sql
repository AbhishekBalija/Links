-- Migration: Create departments table
-- Source: docs/database-design.md § departments
-- Dependencies: none (first table — referenced by student_identities, role_assignments)

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS departments (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  code        TEXT UNIQUE NOT NULL,
  name        TEXT NOT NULL,
  description TEXT,
  hod_user_id UUID,  -- FK to users added in 002 after users table exists
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
