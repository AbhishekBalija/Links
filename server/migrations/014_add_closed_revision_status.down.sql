UPDATE announcement_revisions SET status = 'rejected' WHERE status = 'closed';
ALTER TABLE announcement_revisions DROP CONSTRAINT IF EXISTS chk_announcement_revisions_status;
ALTER TABLE announcement_revisions ADD CONSTRAINT chk_announcement_revisions_status
  CHECK (status IN ('draft', 'pending', 'approved', 'rejected'));
