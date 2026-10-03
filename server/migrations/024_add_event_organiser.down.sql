-- Drops the Organiser. Events handed over to someone else go back to being
-- run by their proposer only.

DROP INDEX IF EXISTS idx_events_organiser;

ALTER TABLE events DROP COLUMN IF EXISTS organiser_id;
