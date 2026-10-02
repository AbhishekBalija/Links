-- Migration: Add users.google_subject
-- Source: spec #129, ticket #132 (Google sign-in)
-- Dependencies: users
--
-- Google's permanent account ID (the ID token's sub). Stored on the first
-- Google sign-in, which matches by verified email, and matched on from then
-- on, so a later change of email at Google can't hand the account to
-- someone else.

ALTER TABLE users ADD COLUMN IF NOT EXISTS google_subject TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_google_subject
  ON users (google_subject)
  WHERE google_subject IS NOT NULL;
