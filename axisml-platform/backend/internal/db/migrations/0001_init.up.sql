-- Platform schema: durable tenant records, identity / authz / sessions, and the
-- four name-level definition tables. Mirrors docs/system_design/database.md §4.
-- Platform shares the axisml database with compute-service / artifact-hub and is
-- isolated by table name; it caches no downstream instance state.

-- ---- Tenants (durable record; guarded HARD delete, not soft) ----------------
CREATE TABLE IF NOT EXISTS tenants (
  id                   uuid PRIMARY KEY,
  identifier           text NOT NULL,                 -- = Tenant CR name = tenant scope; DNS-1123; immutable
  kubernetes_namespace text NOT NULL,                 -- = Tenant.spec.namespace.name; may be shared
  display_name         text,
  description          text,
  owner                text,                          -- from X-Axisml-User; immutable
  labels               jsonb NOT NULL DEFAULT '{}',
  annotations          jsonb NOT NULL DEFAULT '{}',
  suspended_at         timestamptz,                   -- non-null = suspended; new-workload gate
  last_modified_by     text NOT NULL DEFAULT '',
  created_at           timestamptz NOT NULL DEFAULT now(),
  updated_at           timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS tenants_identifier_uniq    ON tenants (identifier);
CREATE INDEX        IF NOT EXISTS tenants_kubernetes_namespace ON tenants (kubernetes_namespace);
CREATE INDEX        IF NOT EXISTS tenants_suspended          ON tenants (suspended_at);
CREATE INDEX        IF NOT EXISTS tenants_created_at         ON tenants (created_at DESC);

-- ---- Identity ---------------------------------------------------------------
CREATE TABLE IF NOT EXISTS users (
  id                   uuid PRIMARY KEY,
  username             text NOT NULL,
  password_hash        text NOT NULL,
  must_change_password boolean NOT NULL DEFAULT false,
  email                text,
  display_name         text,
  disabled             boolean NOT NULL DEFAULT false,
  -- Global system-admin flag. auth.md §3/§4 hard-codes system-admin as a GLOBAL
  -- role (user_roles only binds tenant-admin/user), but database.md §4.1 omits
  -- where it lives; this column is the minimal faithful representation.
  is_system_admin      boolean NOT NULL DEFAULT false,
  created_at           timestamptz NOT NULL DEFAULT now(),
  updated_at           timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS users_username_uniq ON users (username);

CREATE TABLE IF NOT EXISTS user_roles (
  user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  tenant_name text NOT NULL,                          -- = tenants.identifier (app-level FK)
  role        text NOT NULL,                          -- 'tenant-admin' | 'user'
  created_at  timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, tenant_name, role)
);
CREATE INDEX IF NOT EXISTS user_roles_user_tenant ON user_roles (user_id, tenant_name);
CREATE INDEX IF NOT EXISTS user_roles_tenant      ON user_roles (tenant_name);

CREATE TABLE IF NOT EXISTS sessions (
  jti        text PRIMARY KEY,
  user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  expires_at timestamptz NOT NULL,
  revoked    boolean NOT NULL DEFAULT false
);
CREATE INDEX IF NOT EXISTS sessions_expires_at ON sessions (expires_at);

-- ---- Definitions (jobs / experiments / models / images) ---------------------
-- Four isomorphic name-level template tables. Run/version INSTANCES live
-- downstream and are associated live; Platform builds no run/version index.

CREATE TABLE IF NOT EXISTS jobs (
  id           uuid PRIMARY KEY,
  tenant_name  text NOT NULL,                         -- partition key (= tenants.identifier)
  name         text NOT NULL,
  display_name text,
  description  text,
  owner_user   text,
  labels       jsonb NOT NULL DEFAULT '{}',
  annotations  jsonb NOT NULL DEFAULT '{}',
  spec         jsonb NOT NULL,                        -- reusable template (no run columns)
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now(),
  deleted_at   timestamptz
);
CREATE UNIQUE INDEX IF NOT EXISTS jobs_tenant_name_active_uniq ON jobs (tenant_name, name) WHERE deleted_at IS NULL;
CREATE INDEX        IF NOT EXISTS jobs_created_at              ON jobs (created_at DESC);
CREATE INDEX        IF NOT EXISTS jobs_labels_gin              ON jobs USING GIN (labels jsonb_path_ops);

CREATE TABLE IF NOT EXISTS experiments (
  id           uuid PRIMARY KEY,
  tenant_name  text NOT NULL,
  name         text NOT NULL,
  display_name text,
  description  text,
  owner_user   text,
  labels       jsonb NOT NULL DEFAULT '{}',
  annotations  jsonb NOT NULL DEFAULT '{}',
  spec         jsonb NOT NULL,
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now(),
  deleted_at   timestamptz
);
CREATE UNIQUE INDEX IF NOT EXISTS experiments_tenant_name_active_uniq ON experiments (tenant_name, name) WHERE deleted_at IS NULL;
CREATE INDEX        IF NOT EXISTS experiments_created_at              ON experiments (created_at DESC);
CREATE INDEX        IF NOT EXISTS experiments_labels_gin              ON experiments USING GIN (labels jsonb_path_ops);

CREATE TABLE IF NOT EXISTS models (
  id           uuid PRIMARY KEY,
  tenant_name  text NOT NULL,
  name         text NOT NULL,
  display_name text,
  description  text,
  owner_user   text,
  labels       jsonb NOT NULL DEFAULT '{}',
  annotations  jsonb NOT NULL DEFAULT '{}',
  spec         jsonb NOT NULL DEFAULT '{}',
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now(),
  deleted_at   timestamptz
);
CREATE UNIQUE INDEX IF NOT EXISTS models_tenant_name_active_uniq ON models (tenant_name, name) WHERE deleted_at IS NULL;
CREATE INDEX        IF NOT EXISTS models_created_at              ON models (created_at DESC);
CREATE INDEX        IF NOT EXISTS models_labels_gin              ON models USING GIN (labels jsonb_path_ops);

CREATE TABLE IF NOT EXISTS images (
  id           uuid PRIMARY KEY,
  tenant_name  text NOT NULL,
  name         text NOT NULL,
  display_name text,
  description  text,
  owner_user   text,
  labels       jsonb NOT NULL DEFAULT '{}',
  annotations  jsonb NOT NULL DEFAULT '{}',
  spec         jsonb NOT NULL DEFAULT '{}',
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now(),
  deleted_at   timestamptz
);
CREATE UNIQUE INDEX IF NOT EXISTS images_tenant_name_active_uniq ON images (tenant_name, name) WHERE deleted_at IS NULL;
CREATE INDEX        IF NOT EXISTS images_created_at              ON images (created_at DESC);
CREATE INDEX        IF NOT EXISTS images_labels_gin              ON images USING GIN (labels jsonb_path_ops);
