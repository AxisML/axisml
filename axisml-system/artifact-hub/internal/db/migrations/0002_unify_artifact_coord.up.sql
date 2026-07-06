-- Unify the artifact addressing coordinate to (namespace, name, version).
--
-- kind is demoted from an addressing coordinate to a plain immutable attribute
-- so the HTTP API can expose a single /artifacts resource instead of per-kind
-- (models / datasets / images) families. A name is now unique within a
-- namespace across all kinds.

-- Guard: refuse to unify if any (namespace, name, version) already exists under
-- more than one kind — otherwise the new unique index would silently drop rows.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM artifacts
        GROUP BY namespace, name, version
        HAVING count(DISTINCT kind) > 1
    ) THEN
        RAISE EXCEPTION
            'cannot unify artifact coordinate: (namespace, name, version) collides across kinds';
    END IF;
END $$;

-- Re-key the natural coordinate. As before (database.md §1.2) the artifact
-- coordinate never recycles even after soft delete, so the unique index does
-- NOT carry the deleted_at filter.
DROP INDEX IF EXISTS uq_artifacts_coord;
CREATE UNIQUE INDEX uq_artifacts_coord ON artifacts(namespace, name, version);
