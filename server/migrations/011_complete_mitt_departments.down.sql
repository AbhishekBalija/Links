DELETE FROM departments
WHERE code = 'AI'
  AND NOT EXISTS (
    SELECT 1 FROM student_identities WHERE department_id = departments.id
  )
  AND NOT EXISTS (
    SELECT 1
    FROM role_assignments
    WHERE scope_type = 'department'
      AND scope_id = departments.id
  );

-- Revert only rows that still carry the 011 names.
UPDATE departments
SET name = CASE code
  WHEN 'CS' THEN 'Computer Science & Engineering'
  WHEN 'AD' THEN 'Artificial Intelligence & Data Science'
  WHEN 'EC' THEN 'Electronics & Communication Engineering'
END,
updated_at = now()
WHERE (code = 'CS' AND name = 'Computer Science and Engineering')
   OR (code = 'AD' AND name = 'Artificial Intelligence and Data Science')
   OR (code = 'EC' AND name = 'Electronics and Communication Engineering');
