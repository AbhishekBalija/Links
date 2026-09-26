-- Seed the six current MITT B.E. departments with VTU course codes.
-- HOD account links remain null until those users exist in LINKS.

INSERT INTO departments (code, name)
VALUES ('AI', 'Computer Science and Engineering (AI and ML)')
ON CONFLICT (code) DO NOTHING;

-- Rename only rows that still carry the 008 seed name, so a name an admin has
-- already changed is left alone.
UPDATE departments
SET name = CASE code
  WHEN 'CS' THEN 'Computer Science and Engineering'
  WHEN 'AD' THEN 'Artificial Intelligence and Data Science'
  WHEN 'EC' THEN 'Electronics and Communication Engineering'
END,
updated_at = now()
WHERE (code = 'CS' AND name = 'Computer Science & Engineering')
   OR (code = 'AD' AND name = 'Artificial Intelligence & Data Science')
   OR (code = 'EC' AND name = 'Electronics & Communication Engineering');
