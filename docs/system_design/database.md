# AxisML 数据库设计

汇总控制平面持久化在 PostgreSQL 中的表 schema。所有控制面服务共用同一个 database `axisml`，按表名前缀逻辑隔离；schema 迁移由各服务二进制内嵌 `golang-migrate` 在启动时执行。Postgres 部署形态见 [infra/storage.md §3](infra/storage.md#3-数据库)。

**连接契约**：PostgreSQL 引擎归 Infra 层（`axisml-infra` chart，Service `axisml-database`）。System 层服务（compute-service / artifact-hub）与 Platform backend 都经跨 namespace FQDN `axisml-database.axisml-infra:5432` 连接；凭据由各消费层从与 Infra 层同值的 `database.auth.password` 在本 namespace 自渲染为 Secret（Secret namespace-scoped 不跨 namespace 引用，故密码作为各 chart 的共享输入各出现一次）。cluster-manager 与两个 operator 不连 DB。

| 服务 | 表 | 用途 |
| --- | --- | --- |
| [compute-service](system/compute-service.md) | `mlruns` / `mlservices` / `traffic_policies` | 计算任务 / 在线服务·工作区·TensorBoard / 流量策略 |
| [artifact-hub](system/artifact-hub.md) | `artifacts` | 制品（model / dataset / image） |
| [Platform](platform/backend.md) | `tenants` | 租户持久记录 / K8s Namespace 映射 / 停用 / 硬删除（`identifier` 标识） |
| Platform | `users` / `user_roles` / `sessions` + `jobs` / `experiments` / `models` / `images` | 身份 / 授权 / 会话 + 四张 name 级定义 |

> [cluster-manager](system/cluster-manager.md) **不入 PG**——ResourcePool（含 `spec.units[]`）与 `Tenant` CR 持久化在 K8s etcd，cluster-manager 是 REST + K8s API 调用层。

## 1. 通用约定

对**业务表**（`mlruns` / `mlservices` / `traffic_policies` / `artifacts` / `tenants` / 四张定义）生效；身份 / 会话表按需自定义。

### 1.1 通用字段

除 `tenants`（硬删除）外，每张业务表带 `id uuid` 主键、`created_at` / `updated_at` / `deleted_at timestamptz`（时间戳一律 `timestamptz`）；字段归属表只列业务特有字段。

### 1.2 软删与 UNIQUE 约束

唯一键统一为 PG **partial unique index** `WHERE deleted_at IS NULL`（软删行不占唯一键，同名可再创建）；`id` 永久唯一不复用，作跨服务持久引用键。**例外**：`tenants.identifier` 使用普通 UNIQUE 并硬删除；`artifacts` 的 `(namespace, kind, name, version)` 一旦创建即不复用、删除也不释放，以保证跨组件持久引用。

### 1.3 命名校验

会映射到 K8s 对象名的标识字段（Tenant `identifier` / ResourcePool / Job / Service / Artifact）字符集 `[a-z0-9-]`、首尾字母或数字、长度 3–40、不允许连续 `--`、DNS-1123 兼容；Artifact `version` 改用 OCI tag-safe 子集（`A-Za-z0-9_.-`，1–128，禁 `/`）。

### 1.4 generation / observed_generation

允许变更 spec 的 CR-backed 表（`mlservices` / `traffic_policies`）采用 K8s 风格双字段做 outbox 信号——`generation`（`bigint`，API 层 spec mutation 时 +1）/ `observed_generation`（reconciler 成功 patch CR 后写入）；reconciler 经 partial index `WHERE generation <> observed_generation AND deleted_at IS NULL` 定位待同步行。`mlruns` spec 不可变不用 generation（借 `status` 谓词扫描）；`artifacts` / Platform 表无对应 CR，不用。

### 1.5 CR 稳定锚点

CR-backed 对象在 `id` 之外向对应 CR 打 label `axisml.io/<resource>-id=<uuid>`（`tenant-id` / `run-id` / `service-id` / `traffic-policy-id`）；`mlservices` 额外冗余 `axisml.io/service-kind=<kind>` 便于 selector 区分（compute / operator 不按该 label 改变行为）。

### 1.6 扩展元数据 labels / annotations

所有业务表以 `labels jsonb` + `annotations jsonb` 承载扩展元数据（K8s 风格：`labels` 短键短值用于过滤、`annotations` 自由文本用于展示）。Key 前缀：`axisml.io/*` 系统保留（如 `axisml.io/{job,experiment}`），`platform.axisml.io/*` Platform 内部，`user.axisml.io/*` 或无前缀由调用方透传。list 端点接受 `?labelSelector=`（K8s grammar），各表建 GIN 索引兜底、高频 key 额外建复合表达式索引。扩展位**只落 PG**——不下发 CR、不 `+generation`、不参与 reconcile。

## 2. Compute Service

`namespace text` 是历史兼容字段名，语义是 **tenant scope**（租户 `identifier`），不是 K8s Namespace。K8s 落地点来自租户映射中的 `kubernetes_namespace`；新接口与代码变量应优先使用 `tenantScope` / `kubernetesNamespace` 明确区分。

### 2.1 `mlruns`

```sql
CREATE TABLE mlruns (
  id           uuid PRIMARY KEY,
  namespace    text NOT NULL,                 -- tenant scope（兼容字段名），= 租户 identifier
  name         text NOT NULL,
  display_name text,  description text,
  owner        text,                          -- 创建者；不可变
  labels       jsonb NOT NULL DEFAULT '{}',
  annotations  jsonb NOT NULL DEFAULT '{}',
  spec         jsonb NOT NULL,                -- MLRun spec 快照；不可变；含已展开的 nodeSelector / tolerations / resources
  phase        text NOT NULL DEFAULT 'Creating',
  status       jsonb NOT NULL DEFAULT '{}',
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now(),
  deleted_at   timestamptz
);
CREATE UNIQUE INDEX mlruns_namespace_name_active_uniq ON mlruns (namespace, name) WHERE deleted_at IS NULL;
CREATE INDEX mlruns_phase      ON mlruns (phase) WHERE deleted_at IS NULL;
CREATE INDEX mlruns_created_at ON mlruns (created_at DESC);
CREATE INDEX mlruns_labels_gin ON mlruns USING GIN (labels jsonb_path_ops);
CREATE INDEX mlruns_namespace_job_created
  ON mlruns (namespace, (labels->>'axisml.io/job'), created_at DESC) WHERE deleted_at IS NULL;
```

`phase` 是 MLRun CR `status.phase` 顶层冗余，`status` jsonb 持 `{message, startedAt, finishedAt, conditions[]}`（informer 写）。`spec` 含 compute 已展开的 `nodeSelector` / `tolerations` / `resources` snapshot，并保留 `scheduling.poolName` / `unitName` 做溯源（展开见 [compute-service.md §5.4](system/compute-service.md#54-resourcepool-展开)）；`spec.backend` 缺省补 `{native, job}`，创建后不可变。GIN + 复合索引支持 `?labelSelector=axisml.io/job=...`。

### 2.2 `mlservices`

同时承载在线服务（`kind='service'`）、工作区（`kind='workspace'`）、TensorBoard（`kind='tensorboard'`）。

```sql
CREATE TABLE mlservices (
  id           uuid PRIMARY KEY,
  namespace    text NOT NULL,
  name         text NOT NULL,
  kind         text NOT NULL DEFAULT 'service',  -- 'service' | 'workspace' | 'tensorboard'；不可变；冗余到 CR label
  display_name text,  description text,
  owner        text,                             -- 创建者；不可变
  labels       jsonb NOT NULL DEFAULT '{}',
  annotations  jsonb NOT NULL DEFAULT '{}',
  spec         jsonb NOT NULL,                   -- MLService spec 快照；仅 spec.roles[0].replicas 可变
  generation          bigint NOT NULL DEFAULT 1,
  observed_generation bigint NOT NULL DEFAULT 0,
  phase        text NOT NULL DEFAULT 'Creating',
  status       jsonb NOT NULL DEFAULT '{}',
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now(),
  deleted_at   timestamptz
);
CREATE UNIQUE INDEX mlservices_namespace_name_active_uniq ON mlservices (namespace, name) WHERE deleted_at IS NULL;
CREATE INDEX mlservices_namespace_kind ON mlservices (namespace, kind) WHERE deleted_at IS NULL;
CREATE INDEX mlservices_phase          ON mlservices (phase) WHERE deleted_at IS NULL;
CREATE INDEX mlservices_created_at     ON mlservices (created_at DESC);
CREATE INDEX mlservices_sync_pending   ON mlservices (id) WHERE generation <> observed_generation AND deleted_at IS NULL;
CREATE INDEX mlservices_labels_gin     ON mlservices USING GIN (labels jsonb_path_ops);
CREATE INDEX mlservices_namespace_created ON mlservices (namespace, created_at DESC) WHERE deleted_at IS NULL;
```

`phase` 顶层冗余 CR `status.phase`，`status` jsonb 持 `{message, readyReplicas, endpoint, conditions[]}`。`spec` 仅 `roles[0].replicas` 可变（`/scale` 写入并 `+generation`）；`spec.backend` 缺省补 `{native, deployment}`。

### 2.3 `traffic_policies`

把稳定入口的流量按权重分发到同 `namespace` 下多个在线服务（`mlservices` `kind='service'`）；CR 派生见 [compute-service.md §4.3](system/compute-service.md#43-流量策略mltrafficpolicy)。

```sql
CREATE TABLE traffic_policies (
  id           uuid PRIMARY KEY,
  namespace    text NOT NULL,
  name         text NOT NULL,
  mode         text NOT NULL,                    -- 'weighted' | 'canary' | 'bluegreen'；创建后不可变
  display_name text,  description text,
  owner        text,
  labels       jsonb NOT NULL DEFAULT '{}',
  annotations  jsonb NOT NULL DEFAULT '{}',
  spec         jsonb NOT NULL,                   -- endpoint / mode / backend 元组不可变，仅 backends[*].{weight,role} 可变
  generation          bigint NOT NULL DEFAULT 1,
  observed_generation bigint NOT NULL DEFAULT 0,
  phase        text NOT NULL DEFAULT 'Creating',
  status       jsonb NOT NULL DEFAULT '{}',
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now(),
  deleted_at   timestamptz
);
CREATE UNIQUE INDEX traffic_policies_namespace_name_active_uniq ON traffic_policies (namespace, name) WHERE deleted_at IS NULL;
CREATE INDEX traffic_policies_phase        ON traffic_policies (phase) WHERE deleted_at IS NULL;
CREATE INDEX traffic_policies_created_at   ON traffic_policies (created_at DESC);
CREATE INDEX traffic_policies_sync_pending ON traffic_policies (id) WHERE generation <> observed_generation AND deleted_at IS NULL;
CREATE INDEX traffic_policies_labels_gin   ON traffic_policies USING GIN (labels jsonb_path_ops);
```

`status` jsonb 持 `{message, endpoint, backends[].{serviceName, weight, ready}, conditions[]}`。`spec` 持 `{mode, endpoint{path,hostname,auth}, backends[].{serviceName,role,weight}, backend{name,engine}}`——`backends[*].weight` 由 `/split`·`/rollback` 改、canary `/promote` 额外互换两后端 `role`，均 `+generation`；canary 当前基线即 `role=stable` 后端，**不设独立指针**；成员以 `serviceName` 引用同 `namespace` 的 `mlservices` 行、**不冗余成员 spec**。成员占用唯一性（一服务同时只被一活跃策略引用）是跨 jsonb 数组约束，由 compute-service 在事务内应用层维护。

## 3. Artifact Hub

```sql
CREATE TABLE artifacts (
  id           uuid PRIMARY KEY,
  namespace    text NOT NULL,
  kind         text NOT NULL,                  -- model / dataset / image
  name         text NOT NULL,
  version      text NOT NULL,                  -- OCI tag-safe
  visibility   text NOT NULL DEFAULT 'tenant', -- 'tenant' | 'public'（仅 default tenant scope 允许）
  display_name text,  description text,
  labels       jsonb NOT NULL DEFAULT '{}',
  annotations  jsonb NOT NULL DEFAULT '{}',
  owner        text,                           -- 创建者；不可变
  source       text,                           -- webUpload / oras / dockerPush / external；Ready 后冻结
  spec         jsonb NOT NULL,                 -- Kind 特化业务字段；Ready 后冻结
  status       text NOT NULL,                  -- Uploading / Ready / Failed / Deleting / Deleted
  message      text,
  digest       text,                           -- 后端校验通过后写入；OCI Kind 作不可变引用键 <name>@<digest>
  ready_at     timestamptz,
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now(),
  deleted_at   timestamptz
);
CREATE UNIQUE INDEX artifacts_nknv_uniq  ON artifacts (namespace, kind, name, version);
CREATE INDEX artifacts_namespace_kind    ON artifacts (namespace, kind);
CREATE INDEX artifacts_visibility_public ON artifacts (kind, name, version) WHERE visibility = 'public' AND status = 'Ready';
CREATE INDEX artifacts_created_at        ON artifacts (created_at DESC);
CREATE INDEX artifacts_labels_gin        ON artifacts USING GIN (labels jsonb_path_ops);
```

`(namespace, kind, name, version)` 是 §1 软删唯一性的例外（一旦创建即不复用、软删也不释放，故 unique index **不带** `WHERE deleted_at IS NULL`）。`spec` / `digest` Ready 后冻结，"改"= 同 `(namespace, kind, name)` 下新建 `version`；`display_name` / `description` / `labels` / `annotations` 任何阶段可改，`visibility` 创建后不可变。存储地址不入表：`storage_kind` 是 `kind` 的纯函数，`uri` 由 `Handler.BuildStorageURI` 即时构造，新增 Kind 无需 schema 迁移。

## 4. Platform

覆盖**租户持久记录**（`tenants`）、**身份 / 授权 / 会话**、**四张定义**。`tenants` 持 `identifier`、K8s Namespace 映射、展示元数据与停用状态，经 cluster-manager REST 物化为 Tenant CR；租户采用受保护的硬删除。Platform **不缓存任何下游可变实例状态**（run / version / phase / digest / quota 用量一律实时回源），Workspace / Service / TrafficPolicy 等无 Platform 视图表。

### 4.1 身份 / 租户

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

`user_roles.tenant_name` 引用 `tenants.identifier`（唯一且不可变）；租户硬删前必须清空成员，随后删除 Tenant 行与 Tenant CR。**bootstrap**：首次 `axisml-platform bootstrap` 插入 `admin` 用户（默认密码 `admin`，`must_change_password=true`，可由 `AXISML_BOOTSTRAP_PASSWORD` 覆盖），并登记内置租户 `default`，映射到 K8s Namespace `axisml-tenant`，经 cluster-manager 创建 Tenant CR（承载 `public` 制品）。

### 4.2 定义（jobs / experiments / models / images）

四张 name 级**定义 / 模板**表，运行 / 版本实例由下游持有、实时关联，Platform **不建 run/version 索引表**（语义见 [platform.md §3.2](platform/backend.md#32-定义jobs--experiments--models--images)）。四表同构，下给出 `jobs`：

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
