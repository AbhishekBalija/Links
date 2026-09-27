-- Migration: Directory search (#12)
-- Trigram indexes let the directory match names, usernames and headlines
-- case-insensitively and with typos. The extension lives in public so every
-- schema (including the per-test schemas) uses one copy; queries name it as
-- public.* so they don't depend on the search_path.

CREATE EXTENSION IF NOT EXISTS pg_trgm WITH SCHEMA public;

CREATE INDEX IF NOT EXISTS idx_profiles_full_name_trgm ON profiles USING gin (lower(full_name) public.gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_profiles_username_trgm ON profiles USING gin (lower(username) public.gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_profiles_headline_trgm ON profiles USING gin (lower(headline) public.gin_trgm_ops);

-- directory_letters sorts a word's letters, so a search word with two letters
-- swapped ("ahsa") still finds the name word it was meant to be ("asha").
-- Trigrams alone score such short transpositions too low to tell from noise.
CREATE OR REPLACE FUNCTION directory_letters(word text) RETURNS text
  LANGUAGE sql IMMUTABLE PARALLEL SAFE
  AS $$ SELECT string_agg(letter, '' ORDER BY letter) FROM regexp_split_to_table(lower(word), '') AS letter $$;
