# AxisML 数据库设计

本文档汇总 AxisML 控制平面所有持久化在 PostgreSQL 中的表 schema。所有控制面服务共用同一个 database `axisml`，按表名前缀逻辑隔离；schema 迁移由各服务二进制内嵌 `golang-migrate` 在启动时执行。Postgres 部署形态见 [infra.md §4.4](infra.md#44-数据库postgresql)；系统级位置见 [overview.md](overview.md)。

**连接契约**：PostgreSQL 引擎归 Infra 层（`axisml-infra` chart，Service `axisml-database`）。System 层服务（compute-service / artifact-hub）与 Platform 层 backend 都经跨 namespace FQDN `axisml-database.axisml-infra:5432` 连接；连接凭据由各消费层从与 Infra 层同值的 `database.auth.password` 在本 namespace 自渲染为 Secret——Secret 为 namespace-scoped 不跨 namespace 引用，故密码作为各 chart 的共享输入各出现一次（生产环境均指向同一外部托管实例 / 凭据）。cluster-manager 与两个 operator 不连 DB。

| 服务 | 表 | 用途 |
| --- | --- | --- |
| [Compute Service](components/compute-service.md) | `mlruns` | 一次性计算任务 |
| Compute Service | `mlservices` | 常驻在线服务 / 工作区 |
| Compute Service | `traffic_policies` | 流量策略（稳定入口按权重分发到多个在线服务） |
| [artifact-hub](components/artifact-hub.md) | `artifacts` | 制品（model / dataset / image） |
| [Platform](components/platform.md) | `tenants` | 租户持久记录 / 生命周期 / 暂停 / 软删（`identifier` 标识） |
| Platform | `users` / `user_roles` / `sessions` / `audit_logs` | 身份、授权、会话、审计（角色硬编码三档，不入表） |

> [Cluster Manager](components/cluster-manager.md) **不入 PG**——ResourcePool（含内嵌 `spec.units[]`）与 `Tenant` CR 持久化在 K8s etcd 上，cluster-manager 是 REST + K8s API 调用层；租户的持久业务记录（生命周期 / 暂停 / 软删）在 Platform `tenants` 表（§4）。

---

## 1. 通用约定

下列约定对**业务表**生效（`mlruns` / `mlservices` / `traffic_policies` / `artifacts`，以及 Platform `tenants`）；Platform 的身份 / 会话 / 审计表（§4）按需自定义，不强制遵循。新增业务表或扩展现有表时必须满足。

### 1.1 通用字段

每张业务表都带 `id uuid` 主键、`created_at` / `updated_at` / `deleted_at timestamptz`。时间戳一律 `timestamptz`。字段归属表只列业务特有字段，通用字段不重复列。

### 1.2 软删与 UNIQUE 约束

- 唯一键统一实现为 PG **partial unique index** `WHERE deleted_at IS NULL`——软删行不占用唯一键，同名资源在原行被软删后可被再次创建（与 K8s namespace 复用规则一致）；
- `id`（uuid）永远全局唯一、不可复用，作为跨服务持久引用键；
- **例外**：`artifacts` 表的 `(namespace, kind, name, version)` 一旦创建即不复用，删除后也不释放唯一键，以保证跨组件持久引用语义。

### 1.3 命名校验

承载业务标识并会映射到 K8s 对象名的标识字段（Tenant `identifier` / ResourcePool / Job / Service / Artifact）：

- 字符集 `[a-z0-9-]`；首尾为字母或数字；长度 3–40；不允许连续 `--`；
- **DNS-1123 兼容**；
- ResourcePool / ResourceUnit 名称叠加 [cluster-manager.md §3.1](components/cluster-manager.md#31-resourcepool-形状) 的语义命名约定（CRD 校验，本表无 PG 实体）；
- Artifact `version` 改用 OCI tag-safe 子集（`A-Za-z0-9_.-`，长度 1–128，禁止 `/`）。

### 1.4 generation / observed_generation

允许变更 spec 的 CR-backed 表（当前为 `mlservices` / `traffic_policies`）采用 K8s 风格的 generation 双字段做 outbox 信号：

| 字段 | 类型 | 写入方 | 含义 |
| --- | --- | --- | --- |
| `generation` | `bigint` | API 层 | spec mutation 时 +1；对齐 K8s `metadata.generation` |
| `observed_generation` | `bigint` | reconciler | 成功 patch CR 后写入；`generation == observed_generation` 表示已同步；对齐 K8s `status.observedGeneration` |

reconciler 通过 partial index `WHERE generation <> observed_generation AND deleted_at IS NULL` 高效定位待同步行；spec 内容未变但 mutation 重复触发时仍会 +generation，reconciler 走幂等 server-side apply 不会产生副作用。

`mlruns` 表 spec 完全不可变，不使用 generation（同步信号借用 `status` 谓词扫描，见 [compute.md §5.1](components/compute-service.md#51-写路径内嵌-outbox--谓词扫描)）；`artifacts` / Platform 表无对应 CR，更不使用 generation。

### 1.5 CR 稳定锚点

所有 CR-backed 对象在 PG `id` 之外，向对应 CR 打 label `axisml.io/<resource>-id=<uuid>`：

| PG 表 | 对应 CR | label key |
| --- | --- | --- |
| `tenants` | `Tenant` | `axisml.io/tenant-id` |
| `mlruns` | `MLRun` | `axisml.io/run-id` |
| `mlservices` | `MLService` | `axisml.io/service-id` |
| `traffic_policies` | `MLTrafficPolicy` | `axisml.io/traffic-policy-id` |

`mlservices` 在 CR `metadata.labels` 上额外冗余写 `axisml.io/service-kind=<kind>`（`service` / `workspace`），便于 `kubectl` selector 区分工作区与普通服务；compute / operator 不按该 label 改变行为。

### 1.6 扩展元数据 `labels` / `annotations`

所有业务表统一以 `labels jsonb` + `annotations jsonb` 承载扩展元数据，K8s 风格：`labels` 短键短值用于过滤索引（key/value ≤ 63 字符、总条目 ≤ 64）；`annotations` 自由文本用于展示与跟踪（单 value ≤ 4 KiB、总 ≤ 32 KiB）。Key 前缀约定：`axisml.io/*` 为系统级保留前缀（如 `axisml.io/job` / `axisml.io/experiment`），`platform.axisml.io/*` 由 Platform 内部使用，`user.axisml.io/*` 或无前缀由业务服务调用方透传。

**查询**：业务服务的 list 端点接受 `?labelSelector=` 查询参数，语法沿用 K8s（`=`/`==`/`!=`/`in (…)`/`notin (…)`/`key`/`!key`，多条件逗号分隔为 AND）。`mlruns` / `mlservices` / `traffic_policies` / `artifacts` 表均建 GIN 索引兜底；高频 label key（如 `axisml.io/job`）额外建复合表达式索引。

**只落 PG**：扩展位不下发到 CR、不触发 `+generation`、不参与 reconcile。写入路径由业务服务 REST 唯一承载，Platform 走业务服务，不直写 K8s。

---

## 2. Compute Service
`mlruns` / `mlservices` / `traffic_policies` 的 `namespace text` 字段是租户 `identifier`，同时就是 CR 下发的 K8s `metadata.namespace`（单一规范名，由 Platform 传入）；Compute 不 join 任何租户表、不做名字解析。

### 3.1 `mlruns` 表

```sql
CREATE TABLE mlruns (
  id                   uuid PRIMARY KEY,
  namespace            text NOT NULL,                 -- 租户 identifier（= Platform tenants.identifier）
  name                 text NOT NULL,
  display_name         text,
  description          text,
  owner                text,                          -- 创建者；不可变
  labels               jsonb NOT NULL DEFAULT '{}',
  annotations          jsonb NOT NULL DEFAULT '{}',
  spec                 jsonb NOT NULL,                -- MLRun spec 快照；不可变；含已展开的 nodeSelector / tolerations / resources
  phase                text NOT NULL DEFAULT 'Creating',
  status               jsonb NOT NULL DEFAULT '{}',
  created_at           timestamptz NOT NULL DEFAULT now(),
  updated_at           timestamptz NOT NULL DEFAULT now(),
  deleted_at           timestamptz
);

CREATE UNIQUE INDEX mlruns_namespace_name_active_uniq ON mlruns (namespace, name) WHERE deleted_at IS NULL;
CREATE INDEX mlruns_phase      ON mlruns (phase) WHERE deleted_at IS NULL;
CREATE INDEX mlruns_created_at ON mlruns (created_at DESC);
CREATE INDEX mlruns_labels_gin ON mlruns USING GIN (labels jsonb_path_ops);
CREATE INDEX mlruns_namespace_job_created
  ON mlruns (namespace, (labels->>'axisml.io/job'), created_at DESC)
  WHERE deleted_at IS NULL;
```

`phase` 是 MLRun CR `status.phase` 的顶层冗余；`status` jsonb 持剩余子字段 `{message, startedAt, finishedAt, conditions[]}`。两者由 informer 写。`spec` 含 compute 已展开的 `nodeSelector` / `tolerations` / `resources` snapshot, 同时保留 `scheduling.poolName` / `scheduling.unitName` 做溯源 (compute 在 Create 入口完成 ResourcePool CR Informer cache lookup 与合并, 详见 [compute-service.md §5.4](components/compute-service.md#54-resourcepool-展开))。`spec.backend` 缺省时 Compute 写 CR 时补 `{name: "native", engine: "job"}`，创建后不可变。

GIN + 复合表达式索引支持 `?labelSelector=axisml.io/job=...`（列某 Job 的 Run）的过滤路径（详见 [§1.6](#16-扩展元数据-labels--annotations) 与 [compute.md §6](components/compute-service.md#6-接口契约)）。

### 3.2 `mlservices` 表

`mlservices` 表同时承载普通在线服务（`kind='service'`）和 [Platform 工作区](components/platform.md#44-工作区编排)（`kind='workspace'`）。

```sql
CREATE TABLE mlservices (
  id                   uuid PRIMARY KEY,
  namespace            text NOT NULL,                       -- 租户 identifier（= Platform tenants.identifier）
  name                 text NOT NULL,
  kind                 text NOT NULL DEFAULT 'service',     -- 'service' | 'workspace' | 'tensorboard'；不可变；冗余到 CR label axisml.io/service-kind
  display_name         text,
  description          text,
  owner                text,                                -- 创建者；不可变
  labels               jsonb NOT NULL DEFAULT '{}',
  annotations          jsonb NOT NULL DEFAULT '{}',
  spec                 jsonb NOT NULL,                      -- MLService spec 快照；仅 spec.roles[0].replicas 可变；含已展开的 nodeSelector / tolerations / resources
  generation           bigint NOT NULL DEFAULT 1,
  observed_generation  bigint NOT NULL DEFAULT 0,
  phase                text NOT NULL DEFAULT 'Creating',
  status               jsonb NOT NULL DEFAULT '{}',
  created_at           timestamptz NOT NULL DEFAULT now(),
  updated_at           timestamptz NOT NULL DEFAULT now(),
  deleted_at           timestamptz
);

CREATE UNIQUE INDEX mlservices_namespace_name_active_uniq ON mlservices (namespace, name) WHERE deleted_at IS NULL;
CREATE INDEX mlservices_namespace_kind ON mlservices (namespace, kind) WHERE deleted_at IS NULL;
CREATE INDEX mlservices_phase          ON mlservices (phase) WHERE deleted_at IS NULL;
CREATE INDEX mlservices_created_at     ON mlservices (created_at DESC);
CREATE INDEX mlservices_sync_pending   ON mlservices (id) WHERE generation <> observed_generation AND deleted_at IS NULL;
CREATE INDEX mlservices_labels_gin     ON mlservices USING GIN (labels jsonb_path_ops);
CREATE INDEX mlservices_namespace_created
  ON mlservices (namespace, created_at DESC)
  WHERE deleted_at IS NULL;
```

`phase` 是 MLService CR `status.phase` 的顶层冗余；`status` jsonb 持剩余子字段 `{message, readyReplicas, endpoint, conditions[]}`。两者由 informer 写。`spec` 中仅 `spec.roles[0].replicas` 可变（`/scale` 写入并 `+generation`）。`spec.backend` 缺省时 Compute 补 `{name: "native", engine: "deployment"}`。label GIN 索引服务于 labelSelector 查询；`(namespace, created_at)` 复合索引服务于租户内按时间列表（services 无 Job 父级，不按 job 分组）。

### 3.3 `traffic_policies` 表

把一个稳定对外入口的入站流量按权重分发到同 `namespace` 下多个在线服务（`mlservices` 表 `kind='service'` 的行）；CR 派生与契约见 [compute-service.md §4.3](components/compute-service.md#43-流量策略mltrafficpolicy) / [compute-operator.md §4.3](components/compute-operator.md#43-mltrafficpolicy-controller)。

```sql
CREATE TABLE traffic_policies (
  id                   uuid PRIMARY KEY,
  namespace            text NOT NULL,                       -- 租户 identifier（= Platform tenants.identifier）
  name                 text NOT NULL,
  mode                 text NOT NULL,                       -- 'weighted' | 'canary' | 'bluegreen'；创建后不可变
  display_name         text,
  description          text,
  owner                text,                                -- 创建者；不可变
  labels               jsonb NOT NULL DEFAULT '{}',
  annotations          jsonb NOT NULL DEFAULT '{}',
  spec                 jsonb NOT NULL,                      -- MLTrafficPolicy spec 快照；endpoint / mode / backend 元组创建后不可变，仅 backends[*].{weight,role} 可变（role 仅 canary promote 互换）
  generation           bigint NOT NULL DEFAULT 1,
  observed_generation  bigint NOT NULL DEFAULT 0,
  phase                text NOT NULL DEFAULT 'Creating',
  status               jsonb NOT NULL DEFAULT '{}',
  created_at           timestamptz NOT NULL DEFAULT now(),
  updated_at           timestamptz NOT NULL DEFAULT now(),
  deleted_at           timestamptz
);

CREATE UNIQUE INDEX traffic_policies_namespace_name_active_uniq ON traffic_policies (namespace, name) WHERE deleted_at IS NULL;
CREATE INDEX traffic_policies_phase        ON traffic_policies (phase) WHERE deleted_at IS NULL;
CREATE INDEX traffic_policies_created_at   ON traffic_policies (created_at DESC);
CREATE INDEX traffic_policies_sync_pending ON traffic_policies (id) WHERE generation <> observed_generation AND deleted_at IS NULL;
CREATE INDEX traffic_policies_labels_gin   ON traffic_policies USING GIN (labels jsonb_path_ops);
```

`phase` 是 MLTrafficPolicy CR `status.phase` 的顶层冗余；`status` jsonb 持剩余子字段 `{message, endpoint, backends[].{serviceName, weight, ready}, conditions[]}`，由 informer 整块回流。`spec` 持 `{mode, endpoint{path,hostname,auth}, backends[].{serviceName,role,weight}, backend{name,engine}}`——其中 `backends[*].weight` 由 `/split` `/rollback` 改、canary `/promote` 额外互换两后端的 `role`，均 `+generation`。canary 当前基线即 `role=stable` 的后端，**不设独立 `baselineRef` 指针**。成员以 `serviceName` 引用同 `namespace` 的 `mlservices` 行，**不冗余成员 spec**。

**成员占用唯一性**（一个在线服务同时只能被一个活跃策略引用）是跨 `traffic_policies.spec.backends[]` jsonb 数组的约束，PG 难以用单一索引表达，由 compute-service 在创建 / 删除事务内于应用层维护（见 [compute-service.md §4.3](components/compute-service.md#43-流量策略mltrafficpolicy)）。

---

## 3. Artifact Hub
### 4.1 `artifacts` 表

```sql
CREATE TABLE artifacts (
  id            uuid PRIMARY KEY,
  namespace     text NOT NULL,
  kind          text NOT NULL,                  -- model / dataset / image
  name          text NOT NULL,
  version       text NOT NULL,                  -- OCI tag-safe
  visibility    text NOT NULL DEFAULT 'tenant', -- 'tenant'（默认，本 namespace 内可见）| 'public'（全局可见；仅 axisml-system namespace 允许）
  display_name  text,
  description   text,
  labels        jsonb NOT NULL DEFAULT '{}',
  annotations   jsonb NOT NULL DEFAULT '{}',
  owner         text,                           -- 创建者；不可变
  source        text,                           -- webUpload / oras / dockerPush / external；Ready 后冻结

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
CREATE INDEX artifacts_visibility_public   ON artifacts (kind, name, version) WHERE visibility = 'public' AND status = 'Ready';
CREATE INDEX artifacts_created_at          ON artifacts (created_at DESC);
CREATE INDEX artifacts_labels_gin          ON artifacts USING GIN (labels jsonb_path_ops);
```

`(namespace, kind, name, version)` 是 §1.2 软删唯一性的例外——一旦创建即不复用、软删后也不释放，因此 unique index **不带** `WHERE deleted_at IS NULL`。`spec` / `digest` Ready 后冻结，"改"= 同 `(namespace, kind, name)` 下新建 `version`。`display_name` / `description` / `labels` / `annotations` 任何阶段可改；`visibility` 创建后不可变。`namespace` 是租户 `identifier`（= Platform `tenants.identifier`），Artifacts 不解析；`visibility='public'` 仅允许在 `axisml-system` 内置 namespace 下创建（由调用方 Platform RBAC 兜底）。

存储地址不入表：`storage_kind` 是 `kind` 的纯函数，`uri` 由 `Handler.BuildStorageURI(namespace, name, version)` 即时构造。新增 Kind 无需 schema 迁移（`spec jsonb` + `kind text` 兼容），只需新增 handler 与 OpenAPI 枚举。

---

## 4. Platform

Platform PG 覆盖 **租户持久记录**（`tenants`，§5.1）、**身份、授权、会话、审计** 四类，外加 **Job / Experiment / Model / Image 四张定义**（§5.2）。`tenants` 持租户生命周期意图（`identifier` / 展示元数据 / 暂停 / 软删），经 cluster-manager REST 物化为 Tenant CR；**不缓存任何下游可变实例状态**——run / version / phase / conditions / digest / quota 用量（含租户 phase 与配额用量）一律向下游服务实时查询；Workspace / Service / TrafficPolicy 等无 Platform 视图表。流量策略持久化在 compute `traffic_policies`（§3.3），经 Platform 代理、不建 Platform 表。

### 5.1 schema

```sql
CREATE TABLE tenants (
  id                uuid PRIMARY KEY,
  identifier        text NOT NULL,                       -- 租户唯一标识 = Tenant CR 名 = K8s namespace = compute/artifacts 分区键；DNS-1123；创建后不可变
  display_name      text,
  description       text,
  owner             text,                                -- 创建者；来自 X-Axisml-User；不可变
  labels            jsonb NOT NULL DEFAULT '{}',
  annotations       jsonb NOT NULL DEFAULT '{}',
  suspended_at      timestamptz,                         -- 非空 = 暂停态；新建工作负载闸门由 Platform 在创建入口强制
  last_modified_by  text NOT NULL DEFAULT '',
  created_at        timestamptz NOT NULL DEFAULT now(),
  updated_at        timestamptz NOT NULL DEFAULT now(),
  deleted_at        timestamptz                          -- 软删；retention 365 天后物理清理
);
CREATE UNIQUE INDEX tenants_identifier_active_uniq ON tenants (identifier) WHERE deleted_at IS NULL;
CREATE INDEX tenants_deleted_at ON tenants (deleted_at);
CREATE INDEX tenants_suspended  ON tenants (suspended_at) WHERE deleted_at IS NULL;
CREATE INDEX tenants_created_at ON tenants (created_at DESC);

CREATE TABLE users (
  id                    uuid PRIMARY KEY,
  username              text NOT NULL,
  password_hash         text NOT NULL,
  must_change_password  bool NOT NULL DEFAULT false,        -- bootstrap 时 admin/admin 默认 true
  email                 text,
  display_name          text,
  disabled              bool NOT NULL DEFAULT false,
  created_at            timestamptz NOT NULL DEFAULT now(),
  updated_at            timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX users_username_uniq ON users (username);

-- 角色硬编码三档（`system-admin` / `tenant-admin` / `user`），不入表；权限矩阵见 auth.md §3。

CREATE TABLE user_roles (
  user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  tenant_name text NOT NULL,                       -- = tenants.identifier（同库，可建真实 FK；当前走应用层约束）
  role        text NOT NULL,                       -- 'tenant-admin' | 'user'；硬编码枚举
  created_at  timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, tenant_name, role)
);
CREATE INDEX user_roles_user_tenant ON user_roles (user_id, tenant_name);

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

`user_roles.tenant_name` 引用本服务 `tenants.identifier`（同库，可建真实 FK；当前与下游对象一致走应用层约束）——`tenants.identifier` 在 `WHERE deleted_at IS NULL` 上 partial unique 且创建后不可变；租户软删时由 [platform.md §4.1](components/platform.md#41-租户编排) 在应用层级联清理本租户的 `user_roles` 行。`user_roles.role` 是硬编码 text 枚举（`tenant-admin` / `user`），完整矩阵见 [auth.md §3](auth.md#3-rbac-角色)。`audit_logs` 保留期由 `--audit-log-retention-days` 配置（默认 90 天）。

**bootstrap 行为**：首次 `axisml-platform bootstrap` 会插入 `admin` 用户（密码 hash 默认 `admin`，`must_change_password=true`；可通过环境变量 `AXISML_BOOTSTRAP_PASSWORD` 覆盖），同时在 Platform `tenants` 表登记内置租户 `axisml-system`（`identifier=axisml-system`）并经 cluster-manager 创建其 Tenant CR，承载 `visibility=public` 制品。

### 5.2 定义（jobs / experiments / models / images）

这四张表是 Platform 自有的 name 级**定义 / 模板**实体；运行 / 版本**实例**由下游持有，二者实时关联，Platform **不建 run/version 索引表**（语义见 [platform.md §3.2](components/platform.md#32-定义jobs--experiments--models--images)）。

四张定义表同构；下给出 `jobs` 完整定义，`experiments` / `models` / `images` 列与索引一致（仅表名与索引名前缀替换，`spec` 语义不同）。

```sql
CREATE TABLE jobs (
  id            uuid PRIMARY KEY,
  tenant_name   text NOT NULL,                 -- 分区键（= tenants.identifier）
  name          text NOT NULL,
  display_name  text,
  description   text,
  owner_user    text,                          -- 创建者
  labels        jsonb NOT NULL DEFAULT '{}',
  annotations   jsonb NOT NULL DEFAULT '{}',
  spec          jsonb NOT NULL,                -- Job 可复用模板（见下）
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now(),
  deleted_at    timestamptz
);
CREATE UNIQUE INDEX jobs_tenant_name_active_uniq ON jobs (tenant_name, name) WHERE deleted_at IS NULL;
CREATE INDEX jobs_created_at ON jobs (created_at DESC);
CREATE INDEX jobs_labels_gin ON jobs USING GIN (labels jsonb_path_ops);

-- experiments：列与索引同 jobs（表名与索引名前缀替换）；spec 持训练特化模板。
-- models / images：列与索引同 jobs（表名与索引名前缀替换）；spec 改持 name 级业务元数据。
```

- `jobs.spec`：Job 可复用模板——`backend{name,engine,config}` / `roles[]`（含镜像引用）/ `scheduling{poolName,unitName,quota}`（仅名字，compute 内部展开）/ `runPolicy` / 制品引用 `(kind,name,version)`。**无 run 列**。
- `experiments.spec`：与 `jobs.spec` 同构（训练超参即 `roles[*].template.{args,env}`，Platform 不单独建模），语义见 [platform.md §4.9](components/platform.md#49-实验编排)。**无 run 列**。
- `models.spec` / `images.spec`：name 级业务元数据（如 `framework` / `purpose`）；版本级硬校验在 artifacts。**无 version 列**。
- **关联（实时，无索引表）**：Run 经 compute `MLRun` 的 `axisml.io/{job,experiment}=<定义>` label 反查（Run 命名 `<定义>-<n>`）；制品版本经 artifacts `(namespace, kind, name)` 列举。
- 软删后同名可重建（§1.2 partial unique）；定义可在零 Run / 零版本状态下存在。
- **训练指标 / checkpoint 不入 PG、不经 Platform**：实验 Run 的 TensorBoard event log（`experiments/<exp>/runs/<run>/tb/`）/ checkpoint（`.../output/`）由 compute 注入路径与凭证写入对象存储；TensorBoard 实例读 event log，Run 删除时由 compute 一并 GC（编排见 [platform.md §4.9–§4.10](components/platform.md#49-实验编排)）。PG 仅存定义。
