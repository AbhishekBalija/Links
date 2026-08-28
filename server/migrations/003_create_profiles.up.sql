-- Migration: Create profiles table
-- Source: docs/database-design.md § profiles
-- Dependencies: users

CREATE TABLE IF NOT EXISTS profiles (
  user_id                UUID PRIMARY KEY REFERENCES users(id),
  username               TEXT NOT NULL,
  full_name              TEXT NOT NULL,
  headline               TEXT,
  bio                    TEXT,
  avatar_url             TEXT,
  public_profile_enabled BOOLEAN NOT NULL DEFAULT true,
  show_email             BOOLEAN NOT NULL DEFAULT false,
  show_phone             BOOLEAN NOT NULL DEFAULT false,
  linkedin_url           TEXT,
  github_url             TEXT,
  portfolio_url          TEXT,
  created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at             TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Case-insensitive unique username
-- Source: docs/database-design.md § Important Indexes
CREATE UNIQUE INDEX IF NOT EXISTS idx_profiles_username ON profiles (lower(username));
