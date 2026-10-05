-- Forgets every known browser: code requests are counted per address again.

DROP INDEX IF EXISTS idx_sign_in_codes_device;

ALTER TABLE sign_in_codes DROP COLUMN IF EXISTS device_id;

DROP TABLE IF EXISTS known_devices;
