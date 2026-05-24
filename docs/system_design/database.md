# AxisML 数据库设计

本文档汇总 AxisML 控制平面所有持久化在 PostgreSQL 中的表 schema。所有控制面服务共用同一个 database `axisml`，按表名前缀逻辑隔离；schema 迁移由各服务二进制内嵌 `golang-migrate` 在启动时执行。Postgres 部署形态见 [infra.md §4.4](infra.md#44-数据库postgresql)；系统级位置见 [overview.md](overview.md)。

| 服务 | 表 | 用途 |
| --- | --- | --- |
| [Cluster Manager](components/cluster-manager.md) | `tenants` | 租户 / 配额 / namespace spec（写路径权威） |
| [Compute](components/compute.md) | `resource_pools` | 资源池（纯 PG 配置） |
| Compute | `resource_units` | 资源单元（纯 PG 配置） |
| Compute | `jobs` | 一次性计算任务 |
| Compute | `services` | 常驻在线服务 / 工作区 |
| [Artifacts](components/artifacts.md) | `artifacts` | 制品（model / dataset / image / eval_report） |
| [Platform](components/platform.md) | `users` / `roles` / `permissions` / `role_permissions` / `user_tenant_roles` / `sessions` / `audit_logs` | 身份、授权、会话、审计 |

---

## 1. 通用约定

下列约定对**业务表**生效（`tenants` / `jobs` / `services` / `artifacts` / `resource_pools` / `resource_units`）；Platform 的身份 / 会话 / 审计表（§5）按需自定义，不强制遵循。新增业务表或扩展现有表时必须满足。

### 1.1 通用字段

每张业务表都带 `id uuid` 主键、`created_at` / `updated_at` / `deleted_at timestamptz`。时间戳一律 `timestamptz`。字段归属表只列业务特有字段，通用字段不重复列。

### 1.2 软删与 UNIQUE 约束

- 唯一键统一实现为 PG **partial unique index** `WHERE deleted_at IS NULL`——软删行不占用唯一键，同名资源在原行被软删后可被再次创建（与 K8s namespace 复用规则一致）；
- `id`（uuid）永远全局唯一、不可复用，作为跨服务持久引用键；
- **例外**：`artifacts` 表的 `(namespace, kind, name, version)` 一旦创建即不复用，删除后也不释放唯一键，以保证跨组件持久引用语义。

### 1.3 命名校验

承载业务标识并会映射到 K8s 对象名的 `name` 字段（Tenant / ResourcePool / Job / Service / Artifact）：

- 字符集 `[a-z0-9-]`；首尾为字母或数字；长度 3–40；不允许连续 `--`；
- **DNS-1123 兼容**；
- ResourceUnit 名称叠加 [compute.md §4.4 ResourceUnit](components/compute.md#44-resourceunit) 的语义命名约定；
- Artifact `version` 改用 OCI tag-safe 子集（`A-Za-z0-9_.-`，长度 1–128，禁止 `/`）。

### 1.4 generation / observed_generation

允许变更 spec 的 CR-backed 表（当前为 `tenants` / `services`）采用 K8s 风格的 generation 双字段做 outbox 信号：

| 字段 | 类型 | 写入方 | 含义 |
| --- | --- | --- | --- |
| `generation` | `bigint` | API 层 | spec mutation 时 +1；对齐 K8s `metadata.generation` |
| `observed_generation` | `bigint` | reconciler | 成功 patch CR 后写入；`generation == observed_generation` 表示已同步；对齐 K8s `status.observedGeneration` |

reconciler 通过 partial index `WHERE generation <> observed_generation AND deleted_at IS NULL` 高效定位待同步行；spec 内容未变但 mutation 重复触发时仍会 +generation，reconciler 走幂等 server-side apply 不会产生副作用。

`jobs` 表 spec 完全不可变，不使用 generation（同步信号借用 `status` 谓词扫描，见 [compute.md §5.1](components/compute.md#51-写路径内嵌-outbox--4-谓词扫描)）；`resource_pools` / `resource_units` / `artifacts` / Platform 表无对应 CR，更不使用。

### 1.5 CR 稳定锚点

所有 CR-backed 对象在 PG `id` 之外，向对应 CR 打 label `axisml.io/<resource>-id=<uuid>`：

| PG 表 | 对应 CR | label key |
| --- | --- | --- |
| `tenants` | `Tenant` | `axisml.io/tenant-id` |
| `jobs` | `MLJob` | `axisml.io/job-id` |
| `services` | `MLService` | `axisml.io/service-id` |

`services` 在 CR `metadata.labels` 上额外冗余写 `axisml.io/service-kind=<kind>`（`service` / `workspace`），便于 `kubectl` selector 区分工作区与普通服务；compute / operator 不按该 label 改变行为。

### 1.6 扩展元数据 `labels` / `annotations`

所有业务表统一以 `labels jsonb` + `annotations jsonb` 承载扩展元数据，K8s 风格：`labels` 短键短值用于过滤索引（key/value ≤ 63 字符、总条目 ≤ 64）；`annotations` 自由文本用于展示与跟踪（单 value ≤ 4 KiB、总 ≤ 32 KiB）。Key 前缀约定：`platform.axisml.io/*` 由 Platform 内部使用，`user.axisml.io/*` 或无前缀由业务服务调用方透传。

**只落 PG**：扩展位不下发到 CR、不触发 `+generation`、不参与 reconcile。写入路径由业务服务 REST 唯一承载，Platform 走业务服务，不直写 K8s。

---

## 2. Cluster Manager

### 2.1 `tenants` 表

```sql
CREATE TABLE tenants (
  id                   uuid PRIMARY KEY,
  name                 text NOT NULL,
  namespace            text NOT NULL,                            -- 组织分组维度（如 "ai-team"）；与 jobs/services/artifacts 的 namespace（K8s namespace）同名异义，详见下方
  display_name         text,
  description          text,
  owner                text,                       -- 创建者；来自 X-Axisml-User；不可变
  labels               jsonb NOT NULL DEFAULT '{}',
  annotations          jsonb NOT NULL DEFAULT '{}',

  spec                 jsonb NOT NULL,             -- 与 Tenant CR spec 一一对应；字段以 CRD yaml 为准
  generation           bigint NOT NULL DEFAULT 1,
  observed_generation  bigint NOT NULL DEFAULT 0,

  status               jsonb NOT NULL DEFAULT '{"phase":"Creating"}',  -- informer 整块回流 Tenant CR status

  last_modified_by     text NOT NULL DEFAULT '',
  created_at           timestamptz NOT NULL DEFAULT now(),
  updated_at           timestamptz NOT NULL DEFAULT now(),
  deleted_at           timestamptz
);

CREATE UNIQUE INDEX tenants_name_active_uniq ON tenants (name) WHERE deleted_at IS NULL;
CREATE INDEX tenants_deleted_at  ON tenants (deleted_at);
CREATE INDEX tenants_namespace   ON tenants (namespace) WHERE deleted_at IS NULL;
CREATE INDEX tenants_created_at  ON tenants (created_at DESC);
CREATE INDEX tenants_phase       ON tenants ((status->>'phase')) WHERE deleted_at IS NULL;
CREATE INDEX tenants_sync_pending
  ON tenants (id) WHERE generation <> observed_generation AND deleted_at IS NULL;
```

`tenants.namespace`（顶层）是组织分组维度（`ai-team` / `search-team` 等），可变；`tenants.spec.namespace.name` 是 tenant 关联的 K8s namespace（不可变，多 tenant 可共享）。两者同名异义，靠路径区分。`jobs` / `services` / `artifacts` 的 `namespace` 与后者同义。

`status` 结构：`{phase, message, conditions[], quotas[]}`（`quotas[].used` 含每条配额的实际用量），由 informer 从 Tenant CR `status` 整块回流。

**字段归属**

| 字段 | 写入方 | 备注 |
| --- | --- | --- |
| `id` | API | 同时写入 CR `metadata.labels[axisml.io/tenant-id]` |
| `name` | API | 创建后不可变 |
| `namespace` | API | 顶层组织分组维度（非 K8s namespace） |
| `spec` | API | `spec.namespace.name` / `spec.quotas[].{pool,name}` 创建后不可变 |
| `generation` | API | spec mutation 时 +1 |
| `observed_generation` | reconciler | 成功 patch CR 后写入 |
| `status` | informer | 从 Tenant CR `status` 整块回流 |
| `last_modified_by` | API | 每次 mutation 用 `X-Axisml-User` 刷新 |
| `deleted_at` | DELETE / restore | retention 365 天后 GC 物理清理 |

---

## 3. Compute

所有表的 `namespace text` 字段是裸字符串分区键，写入对应 CR `metadata.namespace`；Compute 不校验该 namespace 是否真实存在。

### 3.1 `resource_pools` 表

```sql
CREATE TABLE resource_pools (
  id             uuid PRIMARY KEY,
  name           text NOT NULL,
  description    text,
  node_selector  jsonb NOT NULL DEFAULT '{}',     -- {"axisml.io/pool": "gpu-a100"}
  tolerations    jsonb NOT NULL DEFAULT '[]',     -- K8s Toleration 数组
  labels         jsonb NOT NULL DEFAULT '{}',
  annotations    jsonb NOT NULL DEFAULT '{}',
  created_at     timestamptz NOT NULL DEFAULT now(),
  updated_at     timestamptz NOT NULL DEFAULT now(),
  deleted_at     timestamptz
);

CREATE UNIQUE INDEX resource_pools_name_active_uniq
  ON resource_pools (name) WHERE deleted_at IS NULL;
```

无对应 CR；纯 PG 配置对象。

### 3.2 `resource_units` 表

```sql
CREATE TABLE resource_units (
  id             uuid PRIMARY KEY,
  pool_id        uuid NOT NULL REFERENCES resource_pools(id),
  name           text NOT NULL,
  description    text,
  requests       jsonb NOT NULL DEFAULT '{}',     -- {"cpu":"8","memory":"64Gi","nvidia.com/gpu":"1"}
  limits         jsonb NOT NULL DEFAULT '{}',
  node_selector  jsonb NOT NULL DEFAULT '{}',     -- 通用节点标签匹配
  labels         jsonb NOT NULL DEFAULT '{}',
  annotations    jsonb NOT NULL DEFAULT '{}',
  created_at     timestamptz NOT NULL DEFAULT now(),
  updated_at     timestamptz NOT NULL DEFAULT now(),
  deleted_at     timestamptz
);

CREATE UNIQUE INDEX resource_units_pool_name_active_uniq
  ON resource_units (pool_id, name) WHERE deleted_at IS NULL;
```

无对应 CR；纯 PG 配置对象。

### 3.3 `jobs` 表

```sql
CREATE TABLE jobs (
  id                   uuid PRIMARY KEY,
  namespace            text NOT NULL,
  pool_id              uuid NOT NULL REFERENCES resource_pools(id),
  resource_unit_id     uuid NOT NULL REFERENCES resource_units(id),
  name                 text NOT NULL,
  display_name         text,
  description          text,
  owner                text,                     -- 创建者；不可变
  labels               jsonb NOT NULL DEFAULT '{}',
  annotations          jsonb NOT NULL DEFAULT '{}',
  spec                 jsonb NOT NULL,           -- MLJob spec 快照；不可变
  resources            jsonb,                    -- 按 resource_unit_id 注入的资源申请快照
  status               jsonb NOT NULL DEFAULT '{"phase":"Creating"}',
  created_at           timestamptz NOT NULL DEFAULT now(),
  updated_at           timestamptz NOT NULL DEFAULT now(),
  deleted_at           timestamptz
);

CREATE UNIQUE INDEX jobs_namespace_name_active_uniq ON jobs (namespace, name) WHERE deleted_at IS NULL;
CREATE INDEX jobs_phase      ON jobs ((status->>'phase')) WHERE deleted_at IS NULL;
CREATE INDEX jobs_created_at ON jobs (created_at DESC);
```

`status` 结构：`{phase, message, startedAt, finishedAt, conditions[]}`，由 informer 从 MLJob CR `status` 整块回流。`resources` 是 spec 中 `roles[*].template.resources` 的便利冗余，供 SQL 用量核算；ResourceUnit 后续修改不影响已创建对象。`spec.backend` 缺省时 Compute 写 CR 时补 `{name: "native", engine: "job"}`，创建后不可变。

### 3.4 `services` 表

`services` 表同时承载普通在线服务（`kind='service'`）和 [Platform 工作区](components/platform.md#44-工作区编排)（`kind='workspace'`）。

```sql
CREATE TABLE services (
  id                   uuid PRIMARY KEY,
  namespace            text NOT NULL,
  pool_id              uuid NOT NULL REFERENCES resource_pools(id),
  resource_unit_id     uuid NOT NULL REFERENCES resource_units(id),
  name                 text NOT NULL,
  kind                 text NOT NULL DEFAULT 'service',   -- 'service' | 'workspace'；不可变；冗余到 CR label axisml.io/service-kind
  display_name         text,
  description          text,
  owner                text,                              -- 创建者；不可变
  labels               jsonb NOT NULL DEFAULT '{}',
  annotations          jsonb NOT NULL DEFAULT '{}',
  spec                 jsonb NOT NULL,                    -- MLService spec 快照；仅 spec.roles[0].replicas 可变
  generation           bigint NOT NULL DEFAULT 1,
  observed_generation  bigint NOT NULL DEFAULT 0,
  resources            jsonb,                             -- 同 jobs.resources
  status               jsonb NOT NULL DEFAULT '{"phase":"Creating"}',
  created_at           timestamptz NOT NULL DEFAULT now(),
  updated_at           timestamptz NOT NULL DEFAULT now(),
  deleted_at           timestamptz
);

CREATE UNIQUE INDEX services_namespace_name_active_uniq ON services (namespace, name) WHERE deleted_at IS NULL;
CREATE INDEX services_namespace_kind ON services (namespace, kind) WHERE deleted_at IS NULL;
CREATE INDEX services_phase          ON services ((status->>'phase')) WHERE deleted_at IS NULL;
CREATE INDEX services_created_at     ON services (created_at DESC);
CREATE INDEX services_sync_pending   ON services (id) WHERE generation <> observed_generation AND deleted_at IS NULL;
```

`status` 结构：`{phase, message, readyReplicas, endpoint, conditions[]}`，由 informer 从 MLService CR `status` 整块回流。`spec` 中仅 `spec.roles[0].replicas` 可变（`/scale` 写入并 `+generation`）。`spec.backend` 缺省时 Compute 补 `{name: "native", engine: "deployment"}`。

---

## 4. Artifacts

### 4.1 `artifacts` 表

```sql
CREATE TABLE artifacts (
  id            uuid PRIMARY KEY,
  namespace     text NOT NULL,
  kind          text NOT NULL,                  -- model / dataset / image / eval_report
  name          text NOT NULL,
  version       text NOT NULL,                  -- OCI tag-safe
  display_name  text,
  description   text,
  labels        jsonb NOT NULL DEFAULT '{}',
  annotations   jsonb NOT NULL DEFAULT '{}',
  owner         text,                           -- 创建者；不可变

  spec          jsonb NOT NULL,                 -- Kind 特化业务字段；Ready 后冻结

  status        text NOT NULL,                  -- Uploading / Ready / Failed / Deleting / Deleted
  message       text,
  digest        text,                           -- 后端校验通过后写入；OCI Kind 作为不可变引用键 <name>@<digest>
  ready_at      timestamptz,

  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now(),
  deleted_at    timestamptz
);

CREATE UNIQUE INDEX artifacts_nknv_uniq    ON artifacts (namespace, kind, name, version);
CREATE INDEX artifacts_namespace_kind      ON artifacts (namespace, kind);
CREATE INDEX artifacts_created_at          ON artifacts (created_at DESC);
```

`(namespace, kind, name, version)` 是 §1.2 软删唯一性的例外——一旦创建即不复用、软删后也不释放，因此 unique index **不带** `WHERE deleted_at IS NULL`。`spec` / `digest` Ready 后冻结，"改"= 同 `(namespace, kind, name)` 下新建 `version`。`display_name` / `description` / `labels` / `annotations` 任何阶段可改。

存储地址不入表：`storage_kind` 是 `kind` 的纯函数，`uri` 由 `Handler.BuildStorageURI(namespace, name, version)` 即时构造。新增 Kind 无需 schema 迁移（`spec jsonb` + `kind text` 兼容），只需新增 handler 与 OpenAPI 枚举。

---

## 5. Platform

Platform PG 仅覆盖 **身份、授权、会话、审计** 四类，**不缓存任何下游业务元数据**——Tenant / Workspace / Job / Service / Artifact 等业务对象一律向下游服务实时查询。

### 5.1 schema

```sql
CREATE TABLE users (
  id            uuid PRIMARY KEY,
  username      text NOT NULL,
  password_hash text NOT NULL,
  email         text,
  display_name  text,
  disabled      bool NOT NULL DEFAULT false,
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX users_username_uniq ON users (username);

CREATE TABLE roles (
  id          uuid PRIMARY KEY,
  name        text NOT NULL,
  description text,
  built_in    bool NOT NULL DEFAULT false           -- built_in=true 的角色不可删除
);
CREATE UNIQUE INDEX roles_name_uniq ON roles (name);

CREATE TABLE permissions (
  id          uuid PRIMARY KEY,
  name        text NOT NULL,
  description text
);
CREATE UNIQUE INDEX permissions_name_uniq ON permissions (name);

CREATE TABLE role_permissions (
  role_id       uuid NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
  permission_id uuid NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
  PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE user_tenant_roles (
  user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  tenant_name text NOT NULL,                       -- 引用 Tenant CR metadata.name（稳定 FK，跨服务不约束）
  role_id     uuid NOT NULL REFERENCES roles(id),
  created_at  timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, tenant_name, role_id)
);
CREATE INDEX user_tenant_roles_user_tenant ON user_tenant_roles (user_id, tenant_name);

CREATE TABLE sessions (
  jti        text PRIMARY KEY,                     -- JWT ID
  user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  expires_at timestamptz NOT NULL,
  revoked    bool NOT NULL DEFAULT false
);
CREATE INDEX sessions_expires_at ON sessions (expires_at);

CREATE TABLE audit_logs (
  id         bigserial PRIMARY KEY,
  user_id    uuid REFERENCES users(id) ON DELETE SET NULL,
  action     text NOT NULL,
  target     text NOT NULL,
  metadata   jsonb NOT NULL DEFAULT '{}',
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX audit_logs_created_at      ON audit_logs (created_at DESC);
CREATE INDEX audit_logs_user_created_at ON audit_logs (user_id, created_at DESC);
```

`user_tenant_roles.tenant_name` 不做跨服务 FK——cluster-manager `tenants.name` 在 `WHERE deleted_at IS NULL` 上 partial unique 且创建后不可变，等价于稳定 FK；级联清理由 [platform.md §4.1](components/platform.md#41-租户编排) 在应用层实现。`audit_logs` 保留期由 `--audit-log-retention-days` 配置（默认 90 天）。
