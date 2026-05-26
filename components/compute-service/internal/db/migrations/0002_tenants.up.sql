-- Tenant table. Per the new compute-service design, Compute is the
-- authoritative writer for the Tenant CR; the row is the PG source of
-- truth and the CR is derived. Quotas are inlined in spec jsonb
-- (spec.quotas[]) rather than carried as a separate table.

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
CREATE INDEX tenants_deleted_at ON tenants (deleted_at);
CREATE INDEX tenants_created_at ON tenants (created_at DESC);
CREATE INDEX tenants_phase      ON tenants (phase) WHERE deleted_at IS NULL;
CREATE INDEX tenants_sync_pending
    ON tenants (id) WHERE generation <> observed_generation AND deleted_at IS NULL;
