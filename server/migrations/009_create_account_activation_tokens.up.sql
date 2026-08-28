-- Migration: Create account_activation_tokens table
-- Source: docs/auth.md § Activation Token Flow, ADR-012
-- Dependencies: users
--
-- This table stores activation tokens for the email-based account activation flow.
-- Tokens are generated as 32-byte crypto/rand, base64url-encoded, and stored
-- as SHA-256 hashes. The raw token is sent via email and never stored.

CREATE TABLE IF NOT EXISTS account_activation_tokens (
  id         TEXT PRIMARY KEY,
  user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL,
  purpose    TEXT NOT NULL DEFAULT 'activate',
  expires_at TIMESTAMPTZ NOT NULL,
  used_at    TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_account_activation_tokens_user_id
  ON account_activation_tokens (user_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_account_activation_tokens_hash
  ON account_activation_tokens (token_hash);
