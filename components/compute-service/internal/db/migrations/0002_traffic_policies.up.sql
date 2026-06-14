-- AxisML Compute MLService — traffic_policies (v2).
-- See docs/system_design/components/compute-service.md §4.5 and
-- docs/system_design/database.md §3.4.
--
-- A traffic policy distributes a stable external entrypoint's inbound traffic
-- by weight across multiple online mlservices (mlservices rows with kind='service')
-- in the same namespace. compute-service is the sole spec writer; the
-- MLTrafficPolicy CR is the derived product. Only spec.backends[*].{weight,role}
-- is mutable after create; endpoint / mode / backend tuple are frozen.

CREATE TABLE traffic_policies (
    id                   uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    namespace            text        NOT NULL,
    name                 text        NOT NULL,
    mode                 text        NOT NULL,                 -- 'weighted' | 'canary' | 'bluegreen'
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

CREATE UNIQUE INDEX traffic_policies_namespace_name_active_uniq
    ON traffic_policies (namespace, name) WHERE deleted_at IS NULL;
CREATE INDEX traffic_policies_phase
    ON traffic_policies (phase) WHERE deleted_at IS NULL;
CREATE INDEX traffic_policies_created_at
    ON traffic_policies (created_at DESC);
CREATE INDEX traffic_policies_sync_pending
    ON traffic_policies (id) WHERE generation <> observed_generation AND deleted_at IS NULL;
CREATE INDEX traffic_policies_labels_gin
    ON traffic_policies USING GIN (labels jsonb_path_ops);
