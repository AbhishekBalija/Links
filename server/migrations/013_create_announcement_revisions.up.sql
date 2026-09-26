-- Migration: 013_create_announcement_revisions
-- Content waiting for Announcement approval (ADR 0017). A new Announcement's
-- first publish and later edits both go through a revision; approving one
-- copies it onto the Announcement.

CREATE TABLE IF NOT EXISTS announcement_revisions (
  id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  announcement_id        UUID NOT NULL REFERENCES announcements(id),
  title                  TEXT NOT NULL,
  body                   TEXT NOT NULL,
  category               TEXT NOT NULL,
  -- Audience rules as submitted: [{"department_id", "batch_year", "role"}].
  audience               JSONB NOT NULL DEFAULT '[]',
  expires_at             TIMESTAMPTZ,
  status                 TEXT NOT NULL,
  -- Who approves: this Department's HOD, or the principal/admins when null.
  -- No foreign key, so deleting a Department never blocks on old revisions.
  approver_department_id UUID,
  submitted_by           UUID NOT NULL REFERENCES users(id),
  submitted_at           TIMESTAMPTZ,
  reviewed_by            UUID REFERENCES users(id),
  reviewed_at            TIMESTAMPTZ,
  review_note            TEXT,
  created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at             TIMESTAMPTZ NOT NULL DEFAULT now(),

  CONSTRAINT chk_announcement_revisions_category CHECK (category IN ('official', 'department', 'placement')),
  CONSTRAINT chk_announcement_revisions_status CHECK (status IN ('draft', 'pending', 'approved', 'rejected')),
  CONSTRAINT chk_announcement_revisions_submitted CHECK (status = 'draft' OR submitted_at IS NOT NULL)
);

-- At most one open revision per Announcement.
CREATE UNIQUE INDEX IF NOT EXISTS idx_announcement_revisions_open
  ON announcement_revisions (announcement_id)
  WHERE status IN ('draft', 'pending', 'rejected');

-- The approval queue reads pending revisions oldest first.
CREATE INDEX IF NOT EXISTS idx_announcement_revisions_queue
  ON announcement_revisions (submitted_at, id)
  WHERE status = 'pending';
