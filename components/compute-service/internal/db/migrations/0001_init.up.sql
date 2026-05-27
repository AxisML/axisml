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
-- Phase is the top-level high-frequency filter column; the rest of CR
-- status ({message, startedAt, finishedAt, conditions[]}) lives in
-- status jsonb (database.md §3.2). Pool/Unit provenance lives in labels
-- (axisml.io/resource-pool / -unit), not in a dedicated column.
CREATE TABLE jobs (
    id                   uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    namespace            text        NOT NULL,
    name                 text        NOT NULL,
    display_name         text        NOT NULL DEFAULT '',
    description          text        NOT NULL DEFAULT '',
    owner                text        NOT NULL DEFAULT '',
    labels               jsonb       NOT NULL DEFAULT '{}'::jsonb,
    annotations          jsonb       NOT NULL DEFAULT '{}'::jsonb,
    spec                 jsonb       NOT NULL,
    phase                text        NOT NULL DEFAULT 'Creating',
    status               jsonb       NOT NULL DEFAULT '{}'::jsonb,
    created_at           timestamptz NOT NULL DEFAULT now(),
    updated_at           timestamptz NOT NULL DEFAULT now(),
    deleted_at           timestamptz
);
CREATE UNIQUE INDEX jobs_namespace_name_active_uniq
    ON jobs(namespace, name) WHERE deleted_at IS NULL;
CREATE INDEX jobs_phase                  ON jobs(phase) WHERE deleted_at IS NULL;
CREATE INDEX jobs_created_at             ON jobs(created_at DESC);
CREATE INDEX jobs_labels_gin             ON jobs USING GIN (labels jsonb_path_ops);
CREATE INDEX jobs_namespace_project_created
    ON jobs (namespace, (labels->>'axisml.io/project'), created_at DESC)
    WHERE deleted_at IS NULL;

-- Services ------------------------------------------------------------------
-- Mirrors the jobs table shape: phase + status jsonb. Replicas and
-- ready_replicas live inside status jsonb (status.readyReplicas) per
-- database.md §3.3. Pool/Unit provenance lives in labels.
CREATE TABLE services (
    id                   uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    namespace            text        NOT NULL,
    name                 text        NOT NULL,
    kind                 text        NOT NULL DEFAULT 'service',
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
    created_at           timestamptz NOT NULL DEFAULT now(),
    updated_at           timestamptz NOT NULL DEFAULT now(),
    deleted_at           timestamptz
);
CREATE UNIQUE INDEX services_namespace_name_active_uniq
    ON services(namespace, name) WHERE deleted_at IS NULL;
CREATE INDEX services_namespace_kind
    ON services(namespace, kind) WHERE deleted_at IS NULL;
CREATE INDEX services_phase              ON services(phase) WHERE deleted_at IS NULL;
CREATE INDEX services_created_at         ON services(created_at DESC);
CREATE INDEX services_sync_pending
    ON services(id) WHERE generation <> observed_generation AND deleted_at IS NULL;
CREATE INDEX services_labels_gin
    ON services USING GIN (labels jsonb_path_ops);
CREATE INDEX services_namespace_project_created
    ON services (namespace, (labels->>'axisml.io/project'), created_at DESC)
    WHERE deleted_at IS NULL;
