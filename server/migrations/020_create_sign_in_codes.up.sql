-- Migration: Create sign_in_codes table
-- Source: spec #129, ticket #131 (sign in with a one-time email code)
-- Dependencies: users
--
-- One row per code request, whether or not a code was sent, so the per-email
-- and per-IP limits count every request alike and hold across serverless
-- instances. The row's id is the challenge the browser keeps. Emails and IP
-- addresses are stored only as hashes; the code only as an HMAC.

CREATE TABLE IF NOT EXISTS sign_in_codes (
  id         UUID PRIMARY KEY,
  email_hash TEXT NOT NULL,
  ip_hash    TEXT NOT NULL,
  -- NULL when no code was sent: the email is unknown or can't use codes.
  user_id    UUID REFERENCES users(id) ON DELETE CASCADE,
  code_hash  TEXT,
  attempts   INT NOT NULL DEFAULT 0,
  expires_at TIMESTAMPTZ NOT NULL,
  used_at    TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_sign_in_codes_email_created
  ON sign_in_codes (email_hash, created_at);

CREATE INDEX IF NOT EXISTS idx_sign_in_codes_ip_created
  ON sign_in_codes (ip_hash, created_at);
