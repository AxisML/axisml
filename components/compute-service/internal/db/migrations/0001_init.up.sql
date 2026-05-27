-- AxisML Compute Service initial schema (v1). See
-- docs/system_design/components/compute-service.md.
--
-- compute-service is the authority for the Tenant domain (compute owns the
-- Tenant CR write path) and the home of the Job / Service partition tables.
-- Jobs and Services partition on a bare `namespace` string that equals
-- `tenants.name`; compute joins the tenants table to resolve the K8s
-- namespace at CR-write time.
--
-- ResourcePool / ResourceUnit are NOT persisted here — they live in the
-- cluster-scoped ResourcePool CRD owned by cluster-manager and consumed by
-- compute via a SharedInformer cache (design §5.4). Job / Service rows
-- store the (poolName, unitName) pair as provenance labels; the expanded
-- nodeSelector / tolerations / resources are baked into spec jsonb at
-- create time.

-- pgcrypto provides gen_random_uuid().
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Tenants -------------------------------------------------------------------
CREATE TABLE tenants (
    id                   uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    name                 text        NOT NULL,
    display_name         text        NOT NULL DEFAULT '',
    description          text        NOT NULL DEFAULT '',
    owner                text        NOT NULL DEFAULT '',
    labels               jsonb       NOT NULL DEFAULT '{}'::jsonb,
    annotations          jsonb       NOT NULL DEFAULT '{}'::jsonb,
    spec                 jsonb       NOT NULL,
    generation           bigint      NOT NULL DEFAULT 1,
    observed_generation  bigint      NOT NULL DEFAULT 0,
    phase                text        NOT NULL DEFAULT 'Creating',
    status               jsonb       NOT NULL DEFAULT '{}'::jsonb,
    last_modified_by     text        NOT NULL DEFAULT '',
    created_at           timestamptz NOT NULL DEFAULT now(),
    updated_at           timestamptz NOT NULL DEFAULT now(),
    deleted_at           timestamptz
);
CREATE UNIQUE INDEX tenants_name_active_uniq
    ON tenants (name) WHERE deleted_at IS NULL;
CREATE INDEX tenants_deleted_at  ON tenants (deleted_at);
CREATE INDEX tenants_created_at  ON tenants (created_at DESC);
CREATE INDEX tenants_phase       ON tenants (phase) WHERE deleted_at IS NULL;
CREATE INDEX tenants_sync_pending
    ON tenants (id) WHERE generation <> observed_generation AND deleted_at IS NULL;

-- Jobs ----------------------------------------------------------------------
-- Spec is a frozen snapshot at create time (already pool/unit-expanded).
-- pool_name / unit_name are provenance only (no FK — ResourcePool lives in
-- the K8s CRD).
CREATE TABLE jobs (
    id                   uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    namespace            text        NOT NULL,
    name                 text        NOT NULL,
    pool_name            text        NOT NULL DEFAULT '',
    unit_name            text        NOT NULL DEFAULT '',
    display_name         text        NOT NULL DEFAULT '',
    description          text        NOT NULL DEFAULT '',
    owner_user           text        NOT NULL DEFAULT '',
    labels               jsonb       NOT NULL DEFAULT '{}'::jsonb,
    annotations          jsonb       NOT NULL DEFAULT '{}'::jsonb,
    spec                 jsonb       NOT NULL,
    requested_resources  jsonb       NOT NULL DEFAULT '{}'::jsonb,
    status               text        NOT NULL,
    message              text        NOT NULL DEFAULT '',
    started_at           timestamptz,
    finished_at          timestamptz,
    created_at           timestamptz NOT NULL DEFAULT now(),
    updated_at           timestamptz NOT NULL DEFAULT now(),
    deleted_at           timestamptz
);
CREATE UNIQUE INDEX uq_jobs_namespace_name
    ON jobs(namespace, name) WHERE deleted_at IS NULL;
CREATE INDEX idx_jobs_workset            ON jobs(status, deleted_at);
CREATE INDEX idx_jobs_namespace_status   ON jobs(namespace, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_jobs_labels_gin         ON jobs USING GIN (labels jsonb_path_ops);

-- Services ------------------------------------------------------------------
CREATE TABLE services (
    id                   uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    namespace            text        NOT NULL,
    name                 text        NOT NULL,
    kind                 text        NOT NULL DEFAULT 'service',
    pool_name            text        NOT NULL DEFAULT '',
    unit_name            text        NOT NULL DEFAULT '',
    display_name         text        NOT NULL DEFAULT '',
    description          text        NOT NULL DEFAULT '',
    owner_user           text        NOT NULL DEFAULT '',
    labels               jsonb       NOT NULL DEFAULT '{}'::jsonb,
    annotations          jsonb       NOT NULL DEFAULT '{}'::jsonb,
    spec                 jsonb       NOT NULL,
    generation           bigint      NOT NULL DEFAULT 1,
    observed_generation  bigint      NOT NULL DEFAULT 0,
    requested_resources  jsonb       NOT NULL DEFAULT '{}'::jsonb,
    replicas             integer     NOT NULL DEFAULT 1,
    ready_replicas       integer     NOT NULL DEFAULT 0,
    endpoint             text        NOT NULL DEFAULT '',
    status               text        NOT NULL,
    message              text        NOT NULL DEFAULT '',
    created_at           timestamptz NOT NULL DEFAULT now(),
    updated_at           timestamptz NOT NULL DEFAULT now(),
    deleted_at           timestamptz
);
CREATE UNIQUE INDEX uq_services_namespace_name
    ON services(namespace, name) WHERE deleted_at IS NULL;
CREATE INDEX idx_services_namespace_kind
    ON services(namespace, kind) WHERE deleted_at IS NULL;
CREATE INDEX idx_services_workset
    ON services(status, deleted_at);
CREATE INDEX services_sync_pending
    ON services(id) WHERE generation <> observed_generation AND deleted_at IS NULL;
CREATE INDEX idx_services_namespace_status
    ON services(namespace, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_services_labels_gin
    ON services USING GIN (labels jsonb_path_ops);
