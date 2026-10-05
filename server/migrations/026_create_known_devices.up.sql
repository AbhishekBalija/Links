-- Migration: Remember the browsers people sign in on
-- Source: #178 follow-up (sign-in codes on shared campus Wi-Fi)
-- Dependencies: users, sign_in_codes
--
-- A browser that signed in to an account with an email code keeps a random
-- token in an httpOnly cookie; only its keyed hash is stored here. Code
-- requests from that browser for that account get their own allowance, so
-- nobody else on the same network can use it up. sign_in_codes.device_id
-- records which requests came from a known browser.

CREATE TABLE IF NOT EXISTS known_devices (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash   TEXT NOT NULL UNIQUE,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_used_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at   TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_known_devices_user ON known_devices (user_id);

ALTER TABLE sign_in_codes ADD COLUMN IF NOT EXISTS device_id UUID REFERENCES known_devices(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_sign_in_codes_device ON sign_in_codes (device_id) WHERE device_id IS NOT NULL;
