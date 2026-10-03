-- Migration: Add an Organiser to events
-- Source: #143 (ADR 0028: ending a role hands over the person's work)
-- Dependencies: events, users
--
-- The Organiser runs an Event (edits it once published, cancels it, exports
-- its participants). It starts as the proposer and can move to someone else
-- when the proposer's role ends, while proposer_id keeps who proposed it.

ALTER TABLE events ADD COLUMN IF NOT EXISTS organiser_id UUID REFERENCES users(id);

UPDATE events SET organiser_id = proposer_id WHERE organiser_id IS NULL;

ALTER TABLE events ALTER COLUMN organiser_id SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_events_organiser ON events (organiser_id);
