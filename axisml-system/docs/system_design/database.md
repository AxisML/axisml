# AxisML System 层数据库设计

汇总 System 层服务持久化在 PostgreSQL 中的表 schema：compute-service 的 `mlruns` / `mlservices` / `traffic_policies` 与 artifact-hub 的 `artifacts`。所有控制面服务共用同一个 database `axisml`，按表名前缀逻辑隔离；schema 迁移由各服务二进制内嵌 `golang-migrate` 在启动时执行。Postgres 部署形态见 [infra/overview.md §4.3](../../../axisml-infra/docs/system_design/overview.md#43-数据库)。

**连接契约**：PostgreSQL 引擎归 Infra 层（`axisml-infra` chart，Service `axisml-database`）。System 层服务（compute-service / artifact-hub）经跨 namespace FQDN `axisml-database.axisml-infra:5432` 连接；凭据由本层从与 Infra 层同值的 `database.auth.password` 在本 namespace 自渲染为 Secret（Secret namespace-scoped 不跨 namespace 引用，故密码作为各 chart 的共享输入各出现一次）。cluster-manager 与两个 operator 不连 DB。

| 服务 | 表 | 用途 |
| --- | --- | --- |
| [compute-service](compute-service.md) | `mlruns` / `mlservices` / `traffic_policies` | 计算任务 / 在线服务·工作区·TensorBoard / 流量策略 |
| [artifact-hub](artifact-hub.md) | `artifacts` | 制品（model / dataset / image） |

> [cluster-manager](cluster-manager.md) **不入 PG**——ResourcePool（含 `spec.units[]`）与 `Tenant` CR 持久化在 K8s etcd，cluster-manager 是 REST + K8s API 调用层。

## 1. 通用约定

对 System 层**业务表**（`mlruns` / `mlservices` / `traffic_policies` / `artifacts`）生效。

### 1.1 通用字段

每张业务表带 `id uuid` 主键、`created_at` / `updated_at` / `deleted_at timestamptz`（时间戳一律 `timestamptz`）；字段归属表只列业务特有字段。

### 1.2 软删与 UNIQUE 约束

唯一键统一为 PG **partial unique index** `WHERE deleted_at IS NULL`（软删行不占唯一键，同名可再创建）；`id` 永久唯一不复用，作跨服务持久引用键。**例外**：`artifacts` 的 `(namespace, kind, name, version)` 一旦创建即不复用、删除也不释放，以保证跨组件持久引用。

### 1.3 命名校验

会映射到 K8s 对象名的标识字段（ResourcePool / MLRun `name` / MLService `name` / Artifact `name`）字符集 `[a-z0-9-]`、首尾字母或数字、长度 3–40、不允许连续 `--`、DNS-1123 兼容；Artifact `version` 改用 OCI tag-safe 子集（`A-Za-z0-9_.-`，1–128，禁 `/`）。

### 1.4 generation / observed_generation

允许变更 spec 的 CR-backed 表（`mlservices` / `traffic_policies`）采用 K8s 风格双字段做 outbox 信号——`generation`（`bigint`，API 层 spec mutation 时 +1）/ `observed_generation`（reconciler 成功 patch CR 后写入）；reconciler 经 partial index `WHERE generation <> observed_generation AND deleted_at IS NULL` 定位待同步行。`mlruns` spec 不可变不用 generation（借 `status` 谓词扫描）；`artifacts` 无对应 CR，不用。

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

`phase` 是 MLRun CR `status.phase` 顶层冗余，`status` jsonb 持 `{message, startedAt, finishedAt, conditions[]}`（informer 写）。`spec` 含 compute 已展开的 `nodeSelector` / `tolerations` / `resources` snapshot，并保留 `scheduling.poolName` / `unitName` 做溯源（展开见 [compute-service.md §5.4](compute-service.md#54-resourcepool-展开)）；`spec.backend` 缺省补 `{native, job}`，创建后不可变。GIN + 复合索引支持 `?labelSelector=axisml.io/job=...`。

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

把稳定入口的流量按权重分发到同 `namespace` 下多个在线服务（`mlservices` `kind='service'`）；CR 派生见 [compute-service.md §4.3](compute-service.md#43-流量策略mltrafficpolicy)。

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
