# AxisML Platform 层数据库设计

汇总 Platform backend 持久化在 PostgreSQL 中的表 schema：**租户持久记录**（`tenants`）、**身份 / 授权 / 会话**（`users` / `user_roles` / `sessions`）、**四张定义**（`jobs` / `experiments` / `models` / `images`）。与 System 层服务共用同一个 database `axisml`，按表名前缀逻辑隔离；schema 迁移由 backend 二进制内嵌 `golang-migrate` 在启动时执行。Postgres 部署形态见 [infra/overview.md §4.3](../../../axisml-infra/docs/system_design/overview.md#43-数据库)。System 层表 schema 见 [system/database.md](../../../axisml-system/docs/system_design/database.md)。

**连接契约**：PostgreSQL 引擎归 Infra 层（`axisml-infra` chart，Service `axisml-database`）。Platform backend 经跨 namespace FQDN `axisml-database.axisml-infra:5432` 连接；凭据从与 Infra 层同值的 `database.auth.password` 在本 namespace 自渲染为 Secret（Secret namespace-scoped 不跨 namespace 引用，故密码作为各 chart 的共享输入各出现一次）。

Platform **不缓存任何下游可变实例状态**（run / version / phase / digest / quota 用量一律实时回源），Workspace / Service / TrafficPolicy 等无 Platform 视图表；`tenants` 持 `identifier`、K8s Namespace 映射、展示元数据与停用状态，经 cluster-manager REST 物化为 Tenant CR，租户采用受保护的硬删除。

## 1. 通用约定

对**业务表**（`tenants` / 四张定义）生效；身份 / 会话表（`users` / `user_roles` / `sessions`）按需自定义。

### 1.1 通用字段

除 `tenants`（硬删除）外，每张业务表带 `id uuid` 主键、`created_at` / `updated_at` / `deleted_at timestamptz`（时间戳一律 `timestamptz`）；字段归属表只列业务特有字段。

### 1.2 软删与 UNIQUE 约束

唯一键统一为 PG **partial unique index** `WHERE deleted_at IS NULL`（软删行不占唯一键，同名可再创建）；`id` 永久唯一不复用，作跨服务持久引用键。**例外**：`tenants.identifier` 使用普通 UNIQUE 并硬删除。

### 1.3 命名校验

会映射到 K8s 对象名的标识字段（Tenant `identifier` / Job 等定义 `name`）字符集 `[a-z0-9-]`、首尾字母或数字、长度 3–40、不允许连续 `--`、DNS-1123 兼容。

### 1.4 CR 稳定锚点

`tenants` 经 cluster-manager 物化为 Tenant CR，并在 `id` 之外向 CR 打 label `axisml.io/tenant-id=<uuid>` 作稳定锚点。Platform 四张定义无对应 CR、不下发，故不用 generation / observed_generation。

### 1.5 扩展元数据 labels / annotations

所有业务表以 `labels jsonb` + `annotations jsonb` 承载扩展元数据（K8s 风格：`labels` 短键短值用于过滤、`annotations` 自由文本用于展示）。Key 前缀：`axisml.io/*` 系统保留（如 `axisml.io/{job,experiment}`），`platform.axisml.io/*` Platform 内部，`user.axisml.io/*` 或无前缀由调用方透传。list 端点接受 `?labelSelector=`（K8s grammar），各表建 GIN 索引兜底、高频 key 额外建复合表达式索引。

## 2. 身份 / 租户

```sql
CREATE TABLE tenants (
  id           uuid PRIMARY KEY,
  identifier   text NOT NULL,                 -- = Tenant CR 名 = tenant scope；DNS-1123；创建后不可变
  kubernetes_namespace text NOT NULL,         -- = Tenant.spec.namespace.name；可被多个 Tenant 共享
  display_name text,  description text,
  owner        text,                          -- 来自 X-Axisml-User；不可变
  labels       jsonb NOT NULL DEFAULT '{}',
  annotations  jsonb NOT NULL DEFAULT '{}',
  suspended_at timestamptz,                   -- 非空 = 停用态；新建工作负载闸门由 Platform 在创建入口强制
  last_modified_by text NOT NULL DEFAULT '',
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX tenants_identifier_uniq ON tenants (identifier);
CREATE INDEX tenants_kubernetes_namespace ON tenants (kubernetes_namespace);
CREATE INDEX tenants_suspended  ON tenants (suspended_at);
CREATE INDEX tenants_created_at ON tenants (created_at DESC);

CREATE TABLE users (
  id uuid PRIMARY KEY,
  username     text NOT NULL,
  password_hash text NOT NULL,
  must_change_password bool NOT NULL DEFAULT false,  -- bootstrap admin/admin 默认 true
  email text, display_name text,
  disabled bool NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX users_username_uniq ON users (username);
-- 角色硬编码三档（system-admin / tenant-admin / user），不入表；矩阵见 auth.md §3。

CREATE TABLE user_roles (
  user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  tenant_name text NOT NULL,                  -- = tenants.identifier（同库，可建真实 FK；当前走应用层约束）
  role        text NOT NULL,                  -- 'tenant-admin' | 'user'
  created_at  timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, tenant_name, role)
);
CREATE INDEX user_roles_user_tenant ON user_roles (user_id, tenant_name);

CREATE TABLE sessions (
  jti text PRIMARY KEY,
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  expires_at timestamptz NOT NULL,
  revoked bool NOT NULL DEFAULT false
);
CREATE INDEX sessions_expires_at ON sessions (expires_at);
```

`tenants` 持 `identifier`、K8s Namespace 映射、展示元数据与停用状态，经 cluster-manager REST 物化为 Tenant CR；`suspended_at` 非空即停用态，新建工作负载闸门由 Platform 在创建入口强制；租户采用受保护的硬删除。`user_roles.tenant_name` 引用 `tenants.identifier`（唯一且不可变）；租户硬删前必须清空成员，随后删除 Tenant 行与 Tenant CR。`sessions` 为会话白名单（`jti` 存在且未 `revoked` / 未过期才有效），认证路径对其有效性校验与身份 / RBAC 解析由可选 Redis 缓存前置加速（PostgreSQL 始终为权威，[auth.md §2.1](auth.md#21-会话与身份缓存)）；过期行由 `serve` 后台 sweep 周期清理（`sessions_expires_at` 索引支撑）。**bootstrap**：首次 `axisml-platform bootstrap` 插入 `admin` 用户（默认密码 `admin`，`must_change_password=true`，可由 `AXISML_BOOTSTRAP_PASSWORD` 覆盖），并登记内置租户 `default`，映射到 K8s Namespace `axisml-tenant`，经 cluster-manager 创建 Tenant CR（承载 `public` 制品）。

## 3. 定义（jobs / experiments / models / images）

四张 name 级**定义 / 模板**表，运行 / 版本实例由下游持有、实时关联，Platform **不建 run/version 索引表**（语义见 [backend.md §3.2](backend.md#32-定义jobs--experiments--models--images)）。四表同构，下给出 `jobs`：

```sql
CREATE TABLE jobs (
  id           uuid PRIMARY KEY,
  tenant_name  text NOT NULL,                 -- 分区键（= tenants.identifier）
  name         text NOT NULL,
  display_name text,  description text,
  owner_user   text,
  labels       jsonb NOT NULL DEFAULT '{}',
  annotations  jsonb NOT NULL DEFAULT '{}',
  spec         jsonb NOT NULL,                -- 可复用模板（无 run 列）
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now(),
  deleted_at   timestamptz
);
CREATE UNIQUE INDEX jobs_tenant_name_active_uniq ON jobs (tenant_name, name) WHERE deleted_at IS NULL;
CREATE INDEX jobs_created_at ON jobs (created_at DESC);
CREATE INDEX jobs_labels_gin ON jobs USING GIN (labels jsonb_path_ops);
-- experiments / models / images：列与索引同 jobs（仅表名 / 索引名前缀替换）。
```

- `jobs.spec`：`backend{name,engine,config}` / `roles[]`（含镜像引用）/ `scheduling{poolName,unitName,quota}`（仅名字，compute 内部展开）/ `runPolicy` / 制品引用。`experiments.spec` 与 `jobs.spec` 同构（训练超参即 `roles[*].template.{args,env}`）。`models.spec` / `images.spec`：name 级业务元数据（`framework` / `purpose`），版本级硬校验在 artifacts。**均无 run / version 列**。
- **关联（实时，无索引表）**：Run 经 `MLRun` 的 `axisml.io/{job,experiment}=<定义>` label 反查（命名 `<定义>-<n>`）；制品版本经 artifacts `(namespace, kind, name)` 列举。软删后同名可重建，定义可在零 Run / 零版本下存在。
- **训练指标 / checkpoint 不入 PG、不经 Platform**：实验 Run 的 TensorBoard event log（`experiments/<exp>/runs/<run>/tb/`）/ checkpoint（`.../output/`）由 compute 注入路径与凭证写入对象存储，TensorBoard 实例读取、Run 删除时由 compute 一并 GC。PG 仅存定义。
