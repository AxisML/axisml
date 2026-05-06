-- AxisML Artifacts initial schema. See docs/system_design/artifacts.md §4.

-- pgcrypto provides gen_random_uuid().
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ArtifactRepo --------------------------------------------------------------
CREATE TABLE artifact_repos (
    id                 uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_name        text,
    kind               text        NOT NULL,
    name               text        NOT NULL,
    display_name       text        NOT NULL DEFAULT '',
    description        text        NOT NULL DEFAULT '',
    labels             jsonb       NOT NULL DEFAULT '{}'::jsonb,
    annotations        jsonb       NOT NULL DEFAULT '{}'::jsonb,
    owner_user         text        NOT NULL DEFAULT '',
    status             text        NOT NULL,
    latest_artifact_id uuid,
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now(),
    deleted_at         timestamptz
);

-- Two partial unique indexes (one for tenant_name IS NULL, one for IS NOT NULL)
-- per design §4.2 — avoids COALESCE sentinels.
CREATE UNIQUE INDEX uq_artifact_repos_tenant_kind_name
    ON artifact_repos(tenant_name, kind, name)
    WHERE tenant_name IS NOT NULL;
CREATE UNIQUE INDEX uq_artifact_repos_public_kind_name
    ON artifact_repos(kind, name)
    WHERE tenant_name IS NULL;
CREATE INDEX idx_artifact_repos_workset
    ON artifact_repos(status, deleted_at);

-- Artifact ------------------------------------------------------------------
CREATE TABLE artifacts (
    id           uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    repo_id      uuid        NOT NULL REFERENCES artifact_repos(id),
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
CREATE UNIQUE INDEX uq_artifacts_repo_version ON artifacts(repo_id, version);
CREATE INDEX idx_artifacts_workset ON artifacts(status, deleted_at);
-- Speeds up the GC scan for stale Uploading rows (design §3.4).
CREATE INDEX idx_artifacts_uploading_ttl
    ON artifacts(created_at)
    WHERE status = 'Uploading';
