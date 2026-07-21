-- Migration: Create refresh_tokens table
-- Source: docs/auth.md § Token Strategy
-- "Store refresh tokens hashed"
-- "Rotate refresh tokens on every refresh"
-- "Revoke tokens on logout, password reset, suspension, or role risk event"
--
-- Note: database-design.md does not define this table explicitly, but auth.md
-- requires hashed refresh token storage and rotation. This is the minimal
-- implementation of those requirements.
-- Dependencies: users

CREATE TABLE IF NOT EXISTS refresh_tokens (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  revoked_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user ON refresh_tokens (user_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_refresh_tokens_hash ON refresh_tokens (token_hash);
