-- Migration: 017_create_events
-- Events, their review history and RSVPs (ADR 0023, docs/database-design.md).
-- One status column carries the whole workflow; each review decision is kept
-- in event_reviews so reviewers see earlier notes.

CREATE TABLE IF NOT EXISTS events (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  title             TEXT NOT NULL,
  description       TEXT NOT NULL DEFAULT '',
  event_type        TEXT NOT NULL,
  proposer_id       UUID NOT NULL REFERENCES users(id),
  -- NULL means a college-wide event.
  department_id     UUID REFERENCES departments(id),
  faculty_mentor_id UUID REFERENCES users(id),
  location          TEXT NOT NULL,
  starts_at         TIMESTAMPTZ NOT NULL,
  ends_at           TIMESTAMPTZ NOT NULL,
  capacity          INT,
  status            TEXT NOT NULL DEFAULT 'draft',
  submitted_at      TIMESTAMPTZ,
  published_at      TIMESTAMPTZ,
  cancelled_at      TIMESTAMPTZ,
  cancelled_by      UUID REFERENCES users(id),
  cancel_reason     TEXT,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),

  CONSTRAINT chk_events_type CHECK (event_type IN ('talk', 'workshop', 'competition', 'cultural', 'sports', 'training', 'other')),
  CONSTRAINT chk_events_status CHECK (status IN (
    'draft', 'submitted', 'hod_changes_requested', 'hod_rejected', 'hod_approved',
    'final_changes_requested', 'final_rejected', 'published', 'cancelled'
  )),
  CONSTRAINT chk_events_times CHECK (ends_at > starts_at),
  CONSTRAINT chk_events_capacity CHECK (capacity IS NULL OR capacity > 0),
  CONSTRAINT chk_events_published_at CHECK (status <> 'published' OR published_at IS NOT NULL),
  CONSTRAINT chk_events_cancelled CHECK (status <> 'cancelled' OR (cancelled_at IS NOT NULL AND cancel_reason IS NOT NULL))
);

-- The feed reads published events by start time; review queues read by status.
CREATE INDEX IF NOT EXISTS idx_events_published_starts ON events (starts_at, id) WHERE status = 'published';
CREATE INDEX IF NOT EXISTS idx_events_status_submitted ON events (status, submitted_at);
CREATE INDEX IF NOT EXISTS idx_events_proposer ON events (proposer_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_events_department_starts ON events (department_id, starts_at);

CREATE TABLE IF NOT EXISTS event_reviews (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  event_id    UUID NOT NULL REFERENCES events(id),
  stage       TEXT NOT NULL,
  reviewer_id UUID NOT NULL REFERENCES users(id),
  decision    TEXT NOT NULL,
  note        TEXT,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

  CONSTRAINT chk_event_reviews_stage CHECK (stage IN ('hod', 'final')),
  CONSTRAINT chk_event_reviews_decision CHECK (decision IN ('approve', 'request_changes', 'reject')),
  CONSTRAINT chk_event_reviews_note CHECK (decision = 'approve' OR (note IS NOT NULL AND note <> ''))
);

CREATE INDEX IF NOT EXISTS idx_event_reviews_event ON event_reviews (event_id, created_at);

CREATE TABLE IF NOT EXISTS event_rsvps (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  event_id   UUID NOT NULL REFERENCES events(id),
  user_id    UUID NOT NULL REFERENCES users(id),
  status     TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

  CONSTRAINT chk_event_rsvps_status CHECK (status IN ('going', 'interested', 'not_going'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_event_rsvps_event_user ON event_rsvps (event_id, user_id);
CREATE INDEX IF NOT EXISTS idx_event_rsvps_event_status ON event_rsvps (event_id, status);

-- Events use the same Audience rules as Announcements.
ALTER TABLE audience_rules DROP CONSTRAINT IF EXISTS chk_audience_rules_target_type;
ALTER TABLE audience_rules ADD CONSTRAINT chk_audience_rules_target_type CHECK (target_type IN ('announcement', 'event'));
