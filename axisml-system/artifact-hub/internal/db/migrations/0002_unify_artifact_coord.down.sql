-- Revert the artifact coordinate to (namespace, kind, name, version).
DROP INDEX IF EXISTS uq_artifacts_coord;
CREATE UNIQUE INDEX uq_artifacts_coord ON artifacts(namespace, kind, name, version);
