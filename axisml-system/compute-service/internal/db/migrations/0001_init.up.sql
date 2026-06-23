-- AxisML Compute MLService initial schema (v1). See
-- axisml-system/docs/compute-service.md.
--
-- compute-service is the home of the MLRun / MLService partition tables. It
-- does NOT own the Tenant domain: the Tenant CR is owned by cluster-manager and
-- the durable tenant record lives in Platform. MLRuns and MLServices partition
-- on a bare `namespace` string that equals the tenant identifier (= K8s
-- namespace) supplied by Platform; compute does no tenant lookup or name
-- resolution.
--
-- ResourcePool / ResourceUnit are NOT persisted here — they live in the
-- cluster-scoped ResourcePool CRD owned by cluster-manager and consumed by
-- compute via a SharedInformer cache (design §5.4). MLRun / MLService rows
-- store the (poolName, unitName) pair as provenance labels; the expanded
-- nodeSelector / tolerations / resources are baked into spec jsonb at
-- create time.

-- pgcrypto provides gen_random_uuid().
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- MLRuns ----------------------------------------------------------------------
-- Spec is a frozen snapshot at create time (already pool/unit-expanded).
-- Phase is the top-level high-frequency filter column; the rest of CR
-- status ({message, startedAt, finishedAt, conditions[]}) lives in
-- status jsonb (database.md §3.2). Pool/Unit provenance lives in labels
-- (axisml.io/resource-pool / -unit), not in a dedicated column.
CREATE TABLE mlruns (
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
CREATE UNIQUE INDEX mlruns_namespace_name_active_uniq
    ON mlruns(namespace, name) WHERE deleted_at IS NULL;
CREATE INDEX mlruns_phase                  ON mlruns(phase) WHERE deleted_at IS NULL;
CREATE INDEX mlruns_created_at             ON mlruns(created_at DESC);
CREATE INDEX mlruns_labels_gin             ON mlruns USING GIN (labels jsonb_path_ops);
CREATE INDEX mlruns_namespace_project_created
    ON mlruns (namespace, (labels->>'axisml.io/project'), created_at DESC)
    WHERE deleted_at IS NULL;

-- MLServices ------------------------------------------------------------------
-- Mirrors the mlruns table shape: phase + status jsonb. Replicas and
-- ready_replicas live inside status jsonb (status.readyReplicas) per
-- database.md §3.3. Pool/Unit provenance lives in labels.
CREATE TABLE mlservices (
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
CREATE UNIQUE INDEX mlservices_namespace_name_active_uniq
    ON mlservices(namespace, name) WHERE deleted_at IS NULL;
CREATE INDEX mlservices_namespace_kind
    ON mlservices(namespace, kind) WHERE deleted_at IS NULL;
CREATE INDEX mlservices_phase              ON mlservices(phase) WHERE deleted_at IS NULL;
CREATE INDEX mlservices_created_at         ON mlservices(created_at DESC);
CREATE INDEX mlservices_sync_pending
    ON mlservices(id) WHERE generation <> observed_generation AND deleted_at IS NULL;
CREATE INDEX mlservices_labels_gin
    ON mlservices USING GIN (labels jsonb_path_ops);
CREATE INDEX mlservices_namespace_project_created
    ON mlservices (namespace, (labels->>'axisml.io/project'), created_at DESC)
    WHERE deleted_at IS NULL;
