-- Migration: 018_create_opportunities
-- Placement Opportunities (ADR 0024). One status column; "open" is computed
-- from status and apply_by. An Opportunity's Eligibility is its
-- audience_rules with target_type = 'opportunity', matched like an
-- Announcement's Audience.

CREATE TABLE IF NOT EXISTS opportunities (
  id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  opportunity_type TEXT NOT NULL,
  title            TEXT NOT NULL,
  company          TEXT NOT NULL,
  description      TEXT NOT NULL DEFAULT '',
  location         TEXT,
  -- Free text: a stipend or CTC is written many ways ("4.5 LPA", "15k per month").
  compensation     TEXT,
  apply_by         TIMESTAMPTZ NOT NULL,
  application_mode TEXT NOT NULL,
  external_url     TEXT,
  status           TEXT NOT NULL DEFAULT 'draft',
  posted_by        UUID NOT NULL REFERENCES users(id),
  published_at     TIMESTAMPTZ,
  closed_at        TIMESTAMPTZ,
  closed_by        UUID REFERENCES users(id),
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),

  CONSTRAINT chk_opportunities_type CHECK (opportunity_type IN ('job', 'internship', 'training')),
  CONSTRAINT chk_opportunities_mode CHECK (application_mode IN ('internal', 'external')),
  CONSTRAINT chk_opportunities_external_url CHECK ((application_mode = 'external') = (external_url IS NOT NULL)),
  CONSTRAINT chk_opportunities_status CHECK (status IN ('draft', 'published', 'closed')),
  CONSTRAINT chk_opportunities_published_at CHECK (status = 'draft' OR published_at IS NOT NULL),
  CONSTRAINT chk_opportunities_closed_at CHECK (status <> 'closed' OR closed_at IS NOT NULL)
);

-- Placement staff list by status, newest first; the feed reads published
-- Opportunities by apply_by.
CREATE INDEX IF NOT EXISTS idx_opportunities_status_created ON opportunities (status, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_opportunities_published_apply_by ON opportunities (apply_by, id) WHERE status IN ('published', 'closed');

-- Opportunities use the same Audience rules as Announcements and Events.
ALTER TABLE audience_rules DROP CONSTRAINT IF EXISTS chk_audience_rules_target_type;
ALTER TABLE audience_rules ADD CONSTRAINT chk_audience_rules_target_type CHECK (target_type IN ('announcement', 'event', 'opportunity'));
