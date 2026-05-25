-- AxisML Artifact Hub initial schema. See docs/system_design/components/artifact-hub.md.
--
-- Artifacts are addressed by (namespace, kind, name, version) directly.
-- Namespace is a bare string partition key supplied by the caller —
-- Artifacts performs no existence check on it.

-- pgcrypto provides gen_random_uuid().
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE artifacts (
    id           uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    namespace    text        NOT NULL,
    kind         text        NOT NULL,
    name         text        NOT NULL,
    version      text        NOT NULL,
    display_name text        NOT NULL DEFAULT '',
    description  text        NOT NULL DEFAULT '',
    labels       jsonb       NOT NULL DEFAULT '{}'::jsonb,
    annotations  jsonb       NOT NULL DEFAULT '{}'::jsonb,
    owner_user   text        NOT NULL DEFAULT '',
    spec         jsonb       NOT NULL,
    status       text        NOT NULL,
    message      text        NOT NULL DEFAULT '',
    digest       text        NOT NULL DEFAULT '',
    ready_at     timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    deleted_at   timestamptz
);
CREATE UNIQUE INDEX uq_artifacts_coord
    ON artifacts(namespace, kind, name, version) WHERE deleted_at IS NULL;
CREATE INDEX idx_artifacts_namespace_kind ON artifacts(namespace, kind) WHERE deleted_at IS NULL;
CREATE INDEX idx_artifacts_workset ON artifacts(status, deleted_at);
CREATE INDEX idx_artifacts_uploading_ttl
    ON artifacts(created_at)
    WHERE status = 'Uploading';
