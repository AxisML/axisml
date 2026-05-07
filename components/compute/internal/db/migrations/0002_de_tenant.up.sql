-- De-tenant rewrite. Drop tenant + quota tables and the tenant_id /
-- quota_id FKs from jobs / services; partition by bare namespace string
-- supplied by the caller. No data migration: per the design redesign the
-- old rows are deemed obsolete and existing deployments wipe + reseed.

-- jobs --------------------------------------------------------------------
DROP INDEX IF EXISTS uq_jobs_tenant_name;
DROP INDEX IF EXISTS idx_jobs_tenant_status;
DROP TABLE IF EXISTS jobs;

CREATE TABLE jobs (
    id                   uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    namespace            text        NOT NULL,
    pool_id              uuid        NOT NULL REFERENCES resource_pools(id),
    resource_unit_id     uuid        NOT NULL REFERENCES resource_units(id),
    name                 text        NOT NULL,
    display_name         text        NOT NULL DEFAULT '',
    description          text        NOT NULL DEFAULT '',
    owner_user           text        NOT NULL DEFAULT '',
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
CREATE INDEX idx_jobs_workset ON jobs(status, deleted_at);
CREATE INDEX idx_jobs_namespace_status ON jobs(namespace, status) WHERE deleted_at IS NULL;

-- services ----------------------------------------------------------------
DROP INDEX IF EXISTS uq_services_tenant_name;
DROP INDEX IF EXISTS idx_services_tenant_status;
DROP INDEX IF EXISTS idx_services_spec_sync;
DROP TABLE IF EXISTS services;

CREATE TABLE services (
    id                   uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    namespace            text        NOT NULL,
    pool_id              uuid        NOT NULL REFERENCES resource_pools(id),
    resource_unit_id     uuid        NOT NULL REFERENCES resource_units(id),
    name                 text        NOT NULL,
    display_name         text        NOT NULL DEFAULT '',
    description          text        NOT NULL DEFAULT '',
    owner_user           text        NOT NULL DEFAULT '',
    spec                 jsonb       NOT NULL,
    desired_spec_hash    text        NOT NULL DEFAULT '',
    applied_spec_hash    text        NOT NULL DEFAULT '',
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
CREATE INDEX idx_services_workset ON services(status, deleted_at);
CREATE INDEX idx_services_spec_sync
    ON services(deleted_at) WHERE desired_spec_hash <> applied_spec_hash;
CREATE INDEX idx_services_namespace_status ON services(namespace, status) WHERE deleted_at IS NULL;

-- Now drop the parent tables that nothing references.
DROP TABLE IF EXISTS quotas;
DROP TABLE IF EXISTS tenants;
