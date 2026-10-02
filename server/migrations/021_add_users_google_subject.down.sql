DROP INDEX IF EXISTS idx_users_google_subject;
ALTER TABLE users DROP COLUMN IF EXISTS google_subject;
