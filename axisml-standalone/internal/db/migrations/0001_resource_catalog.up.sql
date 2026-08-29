CREATE TABLE IF NOT EXISTS standalone_resource_pools (
    name             varchar(253) PRIMARY KEY,
    object           jsonb        NOT NULL,
    resource_version bigint       NOT NULL CHECK (resource_version > 0),
    deleted          boolean      NOT NULL DEFAULT false,
    created_at       timestamptz  NOT NULL,
    updated_at       timestamptz  NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_standalone_resource_pools_live_name
    ON standalone_resource_pools (name)
    WHERE deleted = false;

CREATE TABLE IF NOT EXISTS standalone_tenants (
    name             varchar(253) PRIMARY KEY,
    object           jsonb        NOT NULL,
    resource_version bigint       NOT NULL CHECK (resource_version > 0),
    deleted          boolean      NOT NULL DEFAULT false,
    created_at       timestamptz  NOT NULL,
    updated_at       timestamptz  NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_standalone_tenants_live_name
    ON standalone_tenants (name)
    WHERE deleted = false;
