-- Migration: 019_create_opportunity_applications
-- A Student's Application to an Opportunity (ADR 0024): internal ones are made
-- in LINKS, external ones are the Student's record that they applied on the
-- company's site. One per Student per Opportunity; withdrawing is final.

CREATE TABLE IF NOT EXISTS opportunity_applications (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  opportunity_id    UUID NOT NULL REFERENCES opportunities(id),
  student_id        UUID NOT NULL REFERENCES users(id),
  mode              TEXT NOT NULL,
  status            TEXT NOT NULL DEFAULT 'applied',
  applied_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  withdrawn_at      TIMESTAMPTZ,
  status_changed_at TIMESTAMPTZ,
  status_changed_by UUID REFERENCES users(id),
  created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),

  CONSTRAINT chk_opportunity_applications_mode CHECK (mode IN ('internal', 'external')),
  CONSTRAINT chk_opportunity_applications_status CHECK (status IN ('applied', 'shortlisted', 'rejected', 'selected', 'withdrawn')),
  CONSTRAINT chk_opportunity_applications_withdrawn CHECK ((status = 'withdrawn') = (withdrawn_at IS NOT NULL))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_opportunity_applications_opportunity_student ON opportunity_applications (opportunity_id, student_id);
CREATE INDEX IF NOT EXISTS idx_opportunity_applications_opportunity ON opportunity_applications (opportunity_id, status);
CREATE INDEX IF NOT EXISTS idx_opportunity_applications_student ON opportunity_applications (student_id, status);
