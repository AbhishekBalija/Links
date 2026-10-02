-- Migration: Drop passwords and activation tokens
-- Source: spec #129, ticket #136 (ADR 0026: no passwords)
-- Dependencies: users, account_activation_tokens
--
-- Everyone signs in with Google or an email code, and an approved or listed
-- account waits for its first sign-in instead of an Activation link, so
-- nothing reads or writes these any more.

DROP TABLE IF EXISTS account_activation_tokens;

ALTER TABLE users DROP COLUMN IF EXISTS password_hash;
