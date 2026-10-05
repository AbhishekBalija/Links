-- Migration: Remember when someone was welcomed to a new role
-- Source: #142 follow-up (the student coordinator welcome)
-- Dependencies: role_assignments
--
-- The first time a student opens LINKS after becoming a student coordinator,
-- Home explains the role once. welcomed_at records that they closed it, on
-- the account, so it doesn't come back on another device. Empty until then.

ALTER TABLE role_assignments ADD COLUMN IF NOT EXISTS welcomed_at TIMESTAMPTZ;
