-- Seed the six current MITT B.E. departments with VTU course codes.
-- HOD account links remain null until those users exist in LINKS.

INSERT INTO departments (code, name)
VALUES
  ('CS', 'Computer Science and Engineering'),
  ('AD', 'Artificial Intelligence and Data Science'),
  ('AI', 'Computer Science and Engineering (AI and ML)'),
  ('CV', 'Civil Engineering'),
  ('EC', 'Electronics and Communication Engineering'),
  ('ME', 'Mechanical Engineering')
ON CONFLICT (code) DO UPDATE
SET name = EXCLUDED.name,
    updated_at = now();
