-- Add the version provenance column (webUpload / oras / dockerPush / external).
-- database.md §3 lists `source` on the artifacts row; external versions are born
-- Ready and reference a remote URI (no upload).
ALTER TABLE artifacts ADD COLUMN IF NOT EXISTS source text NOT NULL DEFAULT 'webUpload';
