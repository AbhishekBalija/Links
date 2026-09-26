-- Migration: 014_add_closed_revision_status
-- A revision is "closed" when it can never go live: its Announcement was
-- withdrawn, or a direct edit superseded it. Closed revisions are not open, so
-- they can't be resubmitted or approved.

ALTER TABLE announcement_revisions DROP CONSTRAINT IF EXISTS chk_announcement_revisions_status;
ALTER TABLE announcement_revisions ADD CONSTRAINT chk_announcement_revisions_status
  CHECK (status IN ('draft', 'pending', 'approved', 'rejected', 'closed'));
