-- Seed: Insert confirmed MITT department codes (VTU stream-wise course code list)
-- MBA/MCA excluded — they use a separate ID scheme, not the 4MN format.
-- Source: docs/decision-log.md § ADR-013

INSERT INTO departments (code, name)
VALUES
  ('CS', 'Computer Science & Engineering'),
  ('AD', 'Artificial Intelligence & Data Science'),
  ('CV', 'Civil Engineering'),
  ('ME', 'Mechanical Engineering'),
  ('EC', 'Electronics & Communication Engineering')
ON CONFLICT (code) DO NOTHING;
