-- Forgets who was welcomed: anyone holding a coordinator role sees the
-- welcome again if the column comes back.

ALTER TABLE role_assignments DROP COLUMN IF EXISTS welcomed_at;
