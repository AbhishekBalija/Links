-- Migration: 012_create_announcements
-- Announcements and their Audience rules (docs/database-design.md, ADR 0017).
-- An Announcement with no audience rules targets the whole college.

CREATE TABLE IF NOT EXISTS announcements (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  title        TEXT NOT NULL,
  body         TEXT NOT NULL,
  category     TEXT NOT NULL,
  publisher_id UUID NOT NULL REFERENCES users(id),
  status       TEXT NOT NULL,
  published_at TIMESTAMPTZ,
  expires_at   TIMESTAMPTZ,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),

  CONSTRAINT chk_announcements_category CHECK (category IN ('official', 'department', 'placement')),
  CONSTRAINT chk_announcements_status CHECK (status IN ('draft', 'pending', 'published', 'rejected', 'withdrawn')),
  CONSTRAINT chk_announcements_published_at CHECK (status <> 'published' OR published_at IS NOT NULL)
);

-- The feed reads published Announcements newest first.
CREATE INDEX IF NOT EXISTS idx_announcements_feed
  ON announcements (published_at DESC, id DESC)
  WHERE status = 'published';

CREATE INDEX IF NOT EXISTS idx_announcements_publisher ON announcements (publisher_id, created_at DESC);

-- Audience rules are shared with later targeted content (events, opportunities),
-- so they point at their target by type and ID.
CREATE TABLE IF NOT EXISTS audience_rules (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  target_type   TEXT NOT NULL,
  target_id     UUID NOT NULL,
  department_id UUID REFERENCES departments(id),
  batch_year    INT,
  role          TEXT,
  club_id       UUID,
  eligibility   JSONB,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

  CONSTRAINT chk_audience_rules_target_type CHECK (target_type IN ('announcement'))
);

CREATE INDEX IF NOT EXISTS idx_audience_rules_target ON audience_rules (target_type, target_id);
