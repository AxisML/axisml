-- Reverses 0002. Drops the namespace-keyed jobs / services and recreates
-- the tenant + quota tables along with the tenant-keyed jobs / services
-- copy from 0001. No data is preserved.
DROP TABLE IF EXISTS services;
DROP TABLE IF EXISTS jobs;

CREATE TABLE tenants (
    id                  uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    name                text        NOT NULL,
    namespace           text        NOT NULL,
    display_name        text        NOT NULL DEFAULT '',
    spec                jsonb       NOT NULL,
    desired_spec_hash   text        NOT NULL DEFAULT '',
    applied_spec_hash   text        NOT NULL DEFAULT '',
    status              text        NOT NULL,
    message             text        NOT NULL DEFAULT '',
    annotations         jsonb       NOT NULL DEFAULT '{}'::jsonb,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    deleted_at          timestamptz
);
CREATE UNIQUE INDEX uq_tenants_name ON tenants(name) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenants_namespace ON tenants(namespace) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenants_workset ON tenants(status, deleted_at);

CREATE TABLE quotas (
    id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       uuid        NOT NULL REFERENCES tenants(id),
    pool_id         uuid        NOT NULL REFERENCES resource_pools(id),
    name            text        NOT NULL,
    spec            jsonb       NOT NULL,
    status          text        NOT NULL,
    used            jsonb       NOT NULL DEFAULT '{}'::jsonb,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    deleted_at      timestamptz
);
CREATE UNIQUE INDEX uq_quotas_tenant_pool_name
    ON quotas(tenant_id, pool_id, name) WHERE deleted_at IS NULL;
CREATE INDEX idx_quotas_tenant ON quotas(tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_quotas_workset ON quotas(status, deleted_at);

CREATE TABLE jobs (
    id                   uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            uuid        NOT NULL REFERENCES tenants(id),
    pool_id              uuid        NOT NULL REFERENCES resource_pools(id),
    quota_id             uuid        NOT NULL REFERENCES quotas(id),
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
CREATE UNIQUE INDEX uq_jobs_tenant_name
    ON jobs(tenant_id, name) WHERE deleted_at IS NULL;
CREATE INDEX idx_jobs_workset ON jobs(status, deleted_at);
CREATE INDEX idx_jobs_tenant_status ON jobs(tenant_id, status) WHERE deleted_at IS NULL;

CREATE TABLE services (
    id                   uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            uuid        NOT NULL REFERENCES tenants(id),
    pool_id              uuid        NOT NULL REFERENCES resource_pools(id),
    quota_id             uuid        NOT NULL REFERENCES quotas(id),
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
CREATE UNIQUE INDEX uq_services_tenant_name
    ON services(tenant_id, name) WHERE deleted_at IS NULL;
CREATE INDEX idx_services_workset ON services(status, deleted_at);
CREATE INDEX idx_services_spec_sync
    ON services(deleted_at) WHERE desired_spec_hash <> applied_spec_hash;
CREATE INDEX idx_services_tenant_status ON services(tenant_id, status) WHERE deleted_at IS NULL;
