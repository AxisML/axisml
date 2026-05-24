# AxisML 数据库设计

本文档汇总 AxisML 控制平面所有持久化在 PostgreSQL 中的表 schema。所有控制面服务共用同一个 database `axisml`，按表名前缀逻辑隔离；schema 迁移由各服务二进制内嵌 `golang-migrate` 在启动时执行。Postgres 部署形态见 [infra.md §4.4](infra.md#44-数据库postgresql)。

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

下列约定对所有表生效——新增表或扩展现有表时必须满足。

### 1.1 通用字段

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | `uuid` | 主键；服务端 `uuid_generate_v4()` 生成 |
| `created_at` | `timestamptz` | 默认 `now()` |
| `updated_at` | `timestamptz` | 默认 `now()`；写路径同步刷新 |
| `deleted_at` | `timestamptz` NULL | 软删除标记 |

所有时间戳一律使用 `timestamptz`（不使用 `timestamp` / `time`）。

### 1.2 软删与 UNIQUE 约束

- 唯一键统一实现为 PG **partial unique index** `WHERE deleted_at IS NULL`——软删行不占用唯一键，同名资源在原行被软删后可被再次创建（与 K8s namespace 复用规则一致）；
- `id`（uuid）永远全局唯一、不可复用，作为跨服务持久引用键；
- **例外**：`artifacts` 表的 `(namespace, kind, name, version)` 一旦创建即不复用，删除后也不释放唯一键，以保证跨组件持久引用语义。

### 1.3 命名校验

承载业务标识并会映射到 K8s 对象名的 `name` 字段（Tenant / ResourcePool / Job / Service / Artifact）：

- 字符集 `[a-z0-9-]`；首尾为字母或数字；长度 3–40；不允许连续 `--`；
- **DNS-1123 兼容**；
- ResourceUnit 名称叠加 [compute.md §5.3](components/compute.md#53-命名约定) 的语义命名约定；
- Artifact `version` 改用 OCI tag-safe 子集（`A-Za-z0-9_.-`，长度 1–128，禁止 `/`）。

### 1.4 desired / applied spec hash

允许变更 spec 的 CR-backed 表（当前为 `tenants` / `services`）采用双 hash 机制：

| 字段 | 写入方 | 含义 |
| --- | --- | --- |
| `desired_spec_hash` | API 层 | 每次 mutation 后用规范化 JSON 计算 `sha256` |
| `applied_spec_hash` | reconciler | 成功 patch CR 后写入；与 `desired_spec_hash` 相等表示已同步 |

`jobs` 表 spec 完全不可变，不使用双 hash；`resource_pools` / `resource_units` / `artifacts` / Platform 表无对应 CR，更不使用。

### 1.5 CR 稳定锚点

所有 CR-backed 对象在 PG `id` 之外，向对应 CR 打 label `axisml.io/<resource>-id=<uuid>`：

| PG 表 | 对应 CR | label key |
| --- | --- | --- |
| `tenants` | `Tenant` | `axisml.io/tenant-id` |
| `jobs` | `MLJob` | `axisml.io/job-id` |
| `services` | `MLService` | `axisml.io/service-id` |

---

## 2. Cluster Manager

### 2.1 `tenants` 表

```sql
CREATE TABLE tenants (
  id                       uuid PRIMARY KEY,
  name                     text NOT NULL,
  display_name             text NOT NULL,
  description              text NOT NULL DEFAULT '',
  business_unit            text NOT NULL DEFAULT '',
  annotations              jsonb NOT NULL DEFAULT '{}',     -- 透传到 Tenant CR spec.annotations 的扩展位

  namespace_name           text NOT NULL,                    -- 写入 Tenant CR spec.namespace.name；创建后不可变
  namespace_labels         jsonb NOT NULL DEFAULT '{}',
  namespace_annotations    jsonb NOT NULL DEFAULT '{}',

  quotas                   jsonb NOT NULL DEFAULT '[]',      -- [{pool, name, min, max}]；(pool, name) 创建后不可变
  init_resources           jsonb NOT NULL DEFAULT '{}',
  suspended                bool NOT NULL DEFAULT false,

  desired_spec_hash        text NOT NULL,
  applied_spec_hash        text NOT NULL DEFAULT '',         -- reconciler 写入

  phase                    text NOT NULL DEFAULT 'Creating', -- Informer 回流
  namespace_ready          bool NOT NULL DEFAULT false,      -- Informer 回流
  conditions               jsonb NOT NULL DEFAULT '[]',      -- Informer 回流
  quota_status             jsonb NOT NULL DEFAULT '[]',      -- Informer 回流：[{pool, name, ready, used, message}]
  message                  text NOT NULL DEFAULT '',
  last_modified_by         text NOT NULL DEFAULT '',         -- 来自 X-Axisml-User

  created_at               timestamptz NOT NULL DEFAULT now(),
  updated_at               timestamptz NOT NULL DEFAULT now(),
  deleted_at               timestamptz
);

CREATE UNIQUE INDEX tenants_name_active_uniq ON tenants (name) WHERE deleted_at IS NULL;
CREATE INDEX tenants_deleted_at        ON tenants (deleted_at);
CREATE INDEX tenants_business_unit     ON tenants (business_unit);
CREATE INDEX tenants_created_at        ON tenants (created_at DESC);
CREATE INDEX tenants_sync_pending      ON tenants (desired_spec_hash, applied_spec_hash) WHERE desired_spec_hash <> applied_spec_hash;
```

**字段归属**

| 字段 | 写入方 | 备注 |
| --- | --- | --- |
| `id` | API 层 | 同时写入 Tenant CR `metadata.labels[axisml.io/tenant-id]` |
| `name` / `display_name` / `description` / `business_unit` / `annotations` | API 层 | `description` / `business_unit` 升级为 PG 一级字段；reconciler 渲染到 CR `spec.annotations[axisml.io/description, axisml.io/business-unit]` |
| `namespace_name` | API 层 | 创建后不可变 |
| `namespace_labels` / `namespace_annotations` | API 层 | 只在 Namespace 首次创建时落地 |
| `quotas` / `init_resources` / `suspended` | API 层 | `quotas` 内每条 `(pool, name)` 创建后不可变 |
| `desired_spec_hash` | API 层 | 每次 mutation 重算；输入字段：`display_name` / `description` / `business_unit` / `annotations` / `namespace_*` / `quotas` / `init_resources` / `suspended` / `deleted_at` |
| `applied_spec_hash` | reconciler | 成功 patch CR 后写入 |
| `phase` / `namespace_ready` / `conditions` / `quota_status` / `message` | informer | 由 Tenant CR `status.*` 回流 |
| `last_modified_by` | API 层 | 来自 `X-Axisml-User` |
| `deleted_at` | DELETE / restore 端点 | retention（默认 365 天）期满后由后台 GC 物理清理 |

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
  metadata       jsonb NOT NULL DEFAULT '{}',
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
  namespace            text NOT NULL,            -- 写入 MLJob CR metadata.namespace
  pool_id              uuid REFERENCES resource_pools(id),
  resource_unit_id     uuid REFERENCES resource_units(id),
  name                 text NOT NULL,            -- MLJob CR metadata.name
  display_name         text,
  description          text,
  owner_user           text,                     -- 来自 X-Axisml-User
  spec                 jsonb NOT NULL,           -- 提交时的 MLJob.spec 完整快照（不可变）
  requested_resources  jsonb,                    -- 资源申请快照
  status               text NOT NULL,            -- Creating/Pending/Running/Succeeded/Failed/Canceling/Cancelled/Deleting/Deleted
  message              text,
  started_at           timestamptz,
  finished_at          timestamptz,
  created_at           timestamptz NOT NULL DEFAULT now(),
  updated_at           timestamptz NOT NULL DEFAULT now(),
  deleted_at           timestamptz
);

CREATE UNIQUE INDEX jobs_namespace_name_active_uniq
  ON jobs (namespace, name) WHERE deleted_at IS NULL;
```

**字段归属**

| 字段 | 写入方 | 备注 |
| --- | --- | --- |
| `spec` | API 层 | 提交时 MLJob.spec 完整快照；**不可变**；结构详见 [compute-operator.md §4.2](components/compute-operator.md) |
| `requested_resources` | API 层 | 冗余存提交时的资源申请，解耦后续 ResourceUnit 修改对已提交任务的影响 |
| `status` / `message` / `started_at` / `finished_at` | informer | 由 MLJob CR `status.*` 回流 |

`spec.backend` 默认值：用户未指定时 Compute 写 CR 时显式补 `{name: "native", engine: "job"}`；`backend.{name, engine}` 创建后不可变。

### 3.4 `services` 表

`services` 表同时承载普通在线服务（`kind='service'`）和 [Platform 工作区](components/platform.md#8-工作区)（`kind='workspace'`）。

```sql
CREATE TABLE services (
  id                   uuid PRIMARY KEY,
  namespace            text NOT NULL,
  pool_id              uuid REFERENCES resource_pools(id),
  resource_unit_id     uuid REFERENCES resource_units(id),
  name                 text NOT NULL,
  kind                 text NOT NULL DEFAULT 'service', -- 'service' | 'workspace'；创建后不可变
  display_name         text,
  description          text,
  owner_user           text,
  spec                 jsonb NOT NULL,           -- 当前 MLService.spec 快照
  desired_spec_hash    text NOT NULL,
  applied_spec_hash    text NOT NULL DEFAULT '',
  requested_resources  jsonb,
  replicas             int NOT NULL DEFAULT 1,   -- 单 role 约定下 = spec.roles[0].replicas
  ready_replicas       int NOT NULL DEFAULT 0,   -- Informer 回流
  endpoint             text,                     -- Informer 回流
  status               text NOT NULL,            -- Creating/Pending/Ready/Degraded/Failed/Deleting/Deleted
  message              text,
  created_at           timestamptz NOT NULL DEFAULT now(),
  updated_at           timestamptz NOT NULL DEFAULT now(),
  deleted_at           timestamptz
);

CREATE UNIQUE INDEX services_namespace_name_active_uniq
  ON services (namespace, name) WHERE deleted_at IS NULL;
CREATE INDEX services_namespace_kind ON services (namespace, kind) WHERE deleted_at IS NULL;
CREATE INDEX services_sync_pending
  ON services (desired_spec_hash, applied_spec_hash)
  WHERE desired_spec_hash <> applied_spec_hash AND deleted_at IS NULL;
```

**字段归属**

| 字段 | 写入方 | 备注 |
| --- | --- | --- |
| `spec` | API 层 | 扩缩容 API 可更新 `spec.roles[0].replicas` 并重算 `desired_spec_hash`，其他字段不可变 |
| `kind` | API 层 | `'service'` / `'workspace'`；Compute 不按 `kind` 改变行为，仅作分类过滤；创建后不可变 |
| `desired_spec_hash` | API 层 | 输入字段：`spec` 子集（不含 `replicas` 之外的不可变字段散列） |
| `applied_spec_hash` | reconciler | 成功 patch CR 后写入 |
| `replicas` | API 层（`/scale`） | 与 `spec.roles[0].replicas` 同步 |
| `ready_replicas` / `endpoint` / `status` / `message` | informer | 由 MLService CR `status.*` 回流 |

`spec.backend` 默认值：用户未指定时 Compute 写 CR 时显式补 `{name: "native", engine: "deployment"}`。

---

## 4. Artifacts

### 4.1 `artifacts` 表

```sql
CREATE TABLE artifacts (
  -- metadata
  id            uuid PRIMARY KEY,
  namespace     text NOT NULL,                  -- 裸字符串分区键
  kind          text NOT NULL,                  -- 'model' / 'dataset' / 'image' / 'eval_report'
  name          text NOT NULL,                  -- DNS-1123；OCI Kind 兼容 OCI repo 名
  version       text NOT NULL,                  -- OCI tag-safe
  display_name  text,
  description   text,                           -- 此版本说明 / changelog
  labels        jsonb NOT NULL DEFAULT '{}',
  annotations   jsonb NOT NULL DEFAULT '{}',
  owner_user    text,

  -- spec：Kind 特化业务字段，进入 Ready 后冻结
  spec          jsonb NOT NULL,

  -- observed
  status        text NOT NULL,                  -- Uploading / Ready / Failed / Deleting / Deleted
  message       text,
  digest        text,                           -- complete 校验通过后写入
  ready_at      timestamptz,

  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now(),
  deleted_at    timestamptz
);

CREATE UNIQUE INDEX artifacts_nknv_active_uniq
  ON artifacts (namespace, kind, name, version) WHERE deleted_at IS NULL;
CREATE INDEX artifacts_namespace_kind ON artifacts (namespace, kind);
```

**三段式归属**

| 段 | 字段 | 可变性 |
| --- | --- | --- |
| metadata | `id` / `namespace` / `kind` / `name` / `version` / `owner_user` | 创建后不可变 |
| metadata | `display_name` / `description` / `labels` / `annotations` | 任何状态阶段可改 |
| spec | `spec` | 进入 Ready 后冻结；想"改"→ 同 `(namespace, kind, name)` 下新建版本 |
| observed | `status` / `message` / `digest` / `ready_at` | 仅由服务端 / GC / 后端校验回写 |

**存储地址不入表**：`storage_kind` 是 `kind` 的纯函数，`uri` 由 `Handler.BuildStorageURI(namespace, name, version)` 即时构造；`digest` 是唯一入表的"内容哈希"——OCI Kind 用作不可变引用键 `<name>@<digest>`；S3 Kind 仅作 manifest 完整性校验。

**唯一键不复用**：`(namespace, kind, name, version)` 一旦创建即不复用，软删后也不释放——与 [§1.2](#12-软删与-unique-约束) 通用规则的例外，以保证跨组件持久引用语义。

新增 Kind 不需要 schema 迁移（`spec jsonb` + `kind text` 兼容），只需新增 handler 包并扩展 OpenAPI 枚举与允许列表。

---

## 5. Platform

Platform PG 仅覆盖 **身份、授权、会话、审计** 四类，**不缓存任何下游业务元数据**——Tenant / Workspace / Job / Service / Artifact 等业务对象一律向下游服务实时查询。

### 5.1 schema

| 表 | 主键 | 关键字段 | 备注 |
| --- | --- | --- | --- |
| `users` | `id` (uuid) | `username` (uniq), `password_hash`, `email`, `display_name`, `disabled`, `created_at`, `updated_at` | 内置用户体系；外部 IdP 模式下退化为身份缓存 |
| `roles` | `id` (uuid) | `name` (uniq), `description`, `built_in` | `built_in=true` 的内置角色不可删除 |
| `permissions` | `id` (uuid) | `name` (uniq), `description` | 字典表 |
| `role_permissions` | `(role_id, permission_id)` | — | 多对多 |
| `user_tenant_roles` | `(user_id, tenant_name, role_id)` | `created_at` | 用户在某租户内的角色绑定；`tenant_name` 直接引用 Tenant CR `metadata.name`（不可变，等价于稳定 FK） |
| `sessions` | `jti` | `user_id`, `expires_at`, `revoked` | JWT 黑名单（登出 / 强制注销）；按 `expires_at` 定期清理 |
| `audit_logs` | `id` (bigserial) | `user_id`, `action`, `target`, `metadata` (jsonb), `created_at` | 关键管理员操作的审计；保留期由 `--audit-log-retention-days` 配置（默认 90 天） |

### 5.2 索引

- `users.username` 唯一索引；
- `user_tenant_roles (user_id, tenant_name)` 复合索引——前端可见租户列表查询的主入口；
- `audit_logs (created_at DESC)` 与 `(user_id, created_at DESC)` 时间序查询；
- `sessions (expires_at)` 用于定期清理。

`user_tenant_roles.tenant_name` **不做跨服务 FK 约束**；级联清理由 [platform.md §6.5.2](components/platform.md#652-一致性策略与级联) 在应用层实现。
