-- Migration: 015_backfill_batch_year_from_usn
-- Access requests used to save batch_year = 0 when the form sent none, so
-- batch-targeted announcements reached nobody who signed up that way (#45).
-- Take the Batch from the joining year in the USN (4MN23CS001 -> 2023).

UPDATE student_identities
SET batch_year = 2000 + CAST(substring(upper(usn) FROM 4 FOR 2) AS INT),
    updated_at = now()
WHERE batch_year = 0
  AND upper(usn) ~ '^4MN[0-9]{2}[A-Z]{2}[0-9]{3}$';
