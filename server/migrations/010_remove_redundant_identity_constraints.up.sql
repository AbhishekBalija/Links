-- Earlier installs created case-sensitive UNIQUE constraints in addition to
-- the intended case-insensitive unique indexes. Keep only the lower(...) indexes.
ALTER TABLE profiles DROP CONSTRAINT IF EXISTS profiles_username_key;
ALTER TABLE student_identities DROP CONSTRAINT IF EXISTS student_identities_usn_key;
