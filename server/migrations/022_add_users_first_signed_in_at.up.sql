-- Migration: Add users.first_signed_in_at
-- Source: spec #129, ticket #133 (class list and staff invites wait for first sign-in)
-- Dependencies: users
--
-- When an account waiting for its first sign-in (an imported row, a staff
-- invite or an approved Access request) was first signed into. "Not you?"
-- is offered only shortly after it, and clears it again.

ALTER TABLE users ADD COLUMN IF NOT EXISTS first_signed_in_at TIMESTAMPTZ;
