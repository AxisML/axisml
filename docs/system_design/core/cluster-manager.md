# AxisML Cluster Manager 详细设计

AxisML Cluster Manager 是平台的"管理员域"入口服务。它持有 PG `tenants` 表作为租户的权威，把 [Tenant CR](tenant-operator.md) 当作 PG 行的下游派生产物——写路径走 PG → outbox(双 hash) → reconciler → Tenant CR 同步，读路径全部走 PG，CR 状态由 informer 回流到 PG 列。这套形态与 [compute](compute.md) 的 PG + 下游 CR 模型完全对位。

cluster-manager 与 [compute](compute.md) / [artifacts](artifacts.md) 的对齐与差异：

- **三者都是业务服务**，都以 PG 为权威。
- compute 的 PG 下游是 MLJob / MLService CR；artifacts 没有下游 CR（zot OCI 仓库作为 blob 后端）；cluster-manager 的 PG 下游是 Tenant CR。
- cluster-manager 仍只面向 Platform 一个调用方，集群内 ClusterIP，不暴露外部入口。

| 模块 | 职责 | 边界外 |
| --- | --- | --- |
| Tenant ([§6](#6-tenant-api)) | Tenant CRUD / suspend / unsuspend / 列表 / 软删 / 恢复：写 PG → outbox 派生 Tenant CR | Namespace / ElasticQuota / initResources 落地（→ [tenant-operator](tenant-operator.md)） |
| Quota ([§7](#7-quota-api)) | `tenants.quotas` jsonb 的 CRUD：写 PG 后由 reconciler patch 到 CR | ElasticQuota CR 渲染细节 |

**关键不变式**：

> cluster-manager 持有 PG `tenants` 表是租户的权威。Tenant CR 是 PG 行的下游派生产物，由 cluster-manager 自身的 outbox + reconciler 渲染下发；外部 `kubectl create tenant` 路径**不受支持**——admission webhook（阶段二）上线后将硬阻断，MVP 阶段由 reconciler 把外部 spec 漂移擦回 PG 期望态。
>
> 所有 mutation 都先落 PG（包括 `deleted_at` 软删），所有读路径都走 PG；K8s CR 状态（`phase` / `namespaceReady` / `conditions` / `quotas[].used`）由 informer 回流到 PG 同名列再返回给调用方。
>
> `deleted_at IS NOT NULL` 的行保留为历史记录，retention 期限可配；超出后由后台 GC 物理清理。租户历史查询、误删恢复都不需要额外数据结构。

**文档组织**：

- **Part I — 服务运行时**（§1 架构 / §2 运行时契约 / §3 PG schema / §4 写路径与同步 / §5 状态机）。
- **Part II — API**（§6 Tenant、§7 Quota）。
- **Part III — 实施与验证**（§8 实现路径、§9 测试、§10 相关引用）。

---

## Part I — 服务运行时

## 1. 架构总览

```
                          ┌──────────────────────┐
                          │   AxisML Platform    │
                          │  （唯一外部调用方）  │
                          └──────────┬───────────┘
                                     │ REST / JSON + X-Axisml-User
                                     ▼
            ┌───────────────────────────────────────────────────┐
            │           AxisML Cluster Manager (Go)             │
            │  ┌──────────────┐                                 │
            │  │  HTTP API    │ ─────► PG `tenants` 表          │
            │  │  (Gin)       │           │ desired_spec_hash   │
            │  └──────────────┘           ▼                     │
            │                       ┌──────────────┐            │
            │                       │  Reconciler  │ (单 leader)│
            │                       │  desired→CR  │            │
            │                       └──────┬───────┘            │
            │                              │ patch / delete CR  │
            │                              ▼                    │
            │                       ┌──────────────┐            │
            │                       │  K8s API     │            │
            │                       │  Tenant CR   │            │
            │                       └──────┬───────┘            │
            │                              │ status / events    │
            │                       ┌──────▼───────┐            │
            │                       │  Informer    │ → 回写 PG  │
            │                       │ (status回流) │   status 列│
            │                       └──────────────┘            │
            └───────────────────────────────────────────────────┘
                                     │ tenant-operator watches
                                     ▼
                          ┌──────────────────────┐
                          │   tenant-operator    │
                          │  Namespace / EQ /    │
                          │  initResources       │
                          └──────────────────────┘
```

**写路径**：HTTP API → PG（含 `desired_spec_hash` 重算）→ reconciler patch Tenant CR；
**读路径**：HTTP API → PG（status 字段由 informer 持续回流）；
**删除**：HTTP API → `deleted_at = now()` 软删 → reconciler 删除对应 CR → informer 看到 DELETE 后置 `phase=Deleted`；行保留到 retention 期满；
**恢复**：HTTP API → `deleted_at = NULL` → reconciler 重新创建 CR。

```
components/cluster-manager/
├── cmd/cluster-manager/      # main.go (serve / migrate)
├── api/
│   ├── openapi.yaml          # OpenAPI 3.0 契约源
│   └── types/                # oapi-codegen 生成
├── internal/
│   ├── server/               # Gin、middleware、RFC 7807 problem
│   ├── tenant/               # Tenant API handler/service/repository
│   ├── quota/                # Quota API handler/service（操作 tenants.quotas jsonb）
│   ├── reconciler/           # desired/applied 双 hash 扫描 + Tenant CR patch loop
│   ├── informer/             # Tenant CR watch → 回写 PG status
│   ├── db/                   # GORM 初始化、迁移
│   ├── k8sclient/            # controller-runtime client + cache
│   ├── leader/               # leader election (controller-runtime Lease)
│   ├── auth/                 # X-Axisml-User 解析
│   ├── metrics/              # Prometheus
│   └── app/                  # 启动装配
└── pkg/
```

## 2. 运行时契约

### 2.1 进程与端口

- 镜像：`ghcr.io/axisml/axisml-cluster-manager:<appVersion>`
- 启动子命令：
  | 子命令 | 作用 |
  | --- | --- |
  | `serve` | 启动 HTTP API（全副本）+ reconciler / informer（leader-only） |
  | `migrate` | 执行 GORM 迁移；CI / helm pre-install hook 调用 |
- 端口：`8082/tcp`（REST，仅集群内 ClusterIP，不配置外部 `HTTPRoute`）
- 探针：`GET /healthz`（进程存活）；`GET /readyz`（PG + K8s API 双连通）
- Metrics：`GET /metrics`

### 2.2 PG 客户端

- ORM：[GORM](https://gorm.io/)，与 compute / artifacts 一致
- 迁移：`golang-migrate` embedded；服务启动时按 PG advisory lock 串行执行
- 共享 PG 实例：与 compute / artifacts 共享 `axisml-system` namespace 内的 PG（按 schema / table prefix 隔离），与 [overview §6](../overview.md#6-部署架构) 一致
- 配置：`--db-dsn` 注入

### 2.3 K8s 客户端

- controller-runtime `client.Client` + `cache`：cache 用于 Tenant CR 的 watch（informer 回流）；reconciler 写路径直读 API Server，避免读到 stale cache 后误判 hash
- 监听对象：`Tenant`（cluster-scoped）；可选 watch `ElasticQuota` 用于 quota 用量回流
- 写路径形态对位 [compute §3.5](compute.md#35-desiredapplied-spec-hash-双-hash-机制)

### 2.4 副本与 Leader Election

- **HTTP API 层**：默认 `replicas=2`，无状态，多副本天然安全；所有副本都服务 HTTP 并向 PG 写入。
- **Reconciler / Informer**：单 leader——与 [compute](compute.md) 后台协程定位一致；使用 controller-runtime 的 Lease 锁，`axisml-system` namespace 内单一持有者。
- 后台协程仅在 leader 副本启动；非 leader 副本只服务 HTTP。leader 切换期间最坏延迟 = Lease renew 间隔（默认 15s），HTTP 写不受影响。

### 2.5 RBAC（cluster-manager → Kubernetes）

| 资源 | 动作 | 用途 |
| --- | --- | --- |
| `tenants.axisml.io` | `create / get / list / watch / update / patch / delete` | reconciler 渲染 CR；informer 回流 status |
| `elasticquotas.scheduling.sigs.k8s.io` | `get / list / watch` | quota 用量回流（可选） |
| `leases.coordination.k8s.io`（in `axisml-system`） | `get / list / watch / create / update / patch / delete` | leader election |

**不含**：`namespaces`、`secrets`、`configmaps`、`serviceaccounts`、`mljobs.axisml.io`、`mlservices.axisml.io`——这些不在 cluster-manager 的职责范围内。

### 2.6 Helm values 接口

```yaml
clusterManager:
  enabled: true
  image: { registry, repository, tag, pullPolicy }
  replicas: 2
  resources: { requests, limits }
  database:
    dsn: ""                        # 来自 axisml-system 共享 PG，由 chart 注入
  reconciler:
    interval: 10s                  # 扫描 desired ≠ applied 行的最大间隔
  retention:
    archivedTenantDays: 365        # deleted_at IS NOT NULL 的行保留期；GC 物理清理超期行
```

Helm 模板清单（`deploy/helm/axisml-system/templates/cluster-manager/`）：

| 文件 | 用途 |
| --- | --- |
| `deployment.yaml` | cluster-manager 镜像，探针，env 注入 DSN |
| `service.yaml` | ClusterIP 8082 |
| `serviceaccount.yaml` | 服务账号 |
| `clusterrole.yaml` / `clusterrolebinding.yaml` | §2.5 RBAC |
| `role.yaml` / `rolebinding.yaml`（in `axisml-system`） | leader election Lease 权限 |
| `servicemonitor.yaml` | `/metrics` 暴露 |
| `migrate-job.yaml` | helm pre-install / pre-upgrade hook，执行 `migrate` 子命令 |

### 2.7 与 Platform 的请求契约

cluster-manager 仅接受 Platform 通过集群内 Service DNS 发起的 REST 调用，不直接对外部用户流量开放。

- **路径前缀**：`/api/v1/...`
- **身份头**：Platform 注入 `X-Axisml-User`；cluster-manager 写入 PG `tenants.last_modified_by` 与 K8s Event；**不**做角色级鉴权（由 Platform 统一完成）
- **OpenAPI 契约**：`components/cluster-manager/api/openapi.yaml` 为唯一契约源，使用 `oapi-codegen` 生成 server stub 与 Platform 侧 client SDK
- **错误格式**：HTTP 标准状态码 + RFC 7807 problem+json
- **写后语义**：所有 mutation 在 PG 提交成功后即返回 200；CR 同步与 operator 处理是异步的，调用方通过 GET 观察 `phase` / `namespace_ready` / `quotas[].ready` 确认落地

## 3. PG schema

### 3.1 `tenants` 表

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

  desired_spec_hash        text NOT NULL,                    -- 见 §4.1
  applied_spec_hash        text NOT NULL DEFAULT '',         -- reconciler 写入

  phase                    text NOT NULL DEFAULT 'Creating', -- Informer 回流
  namespace_ready          bool NOT NULL DEFAULT false,      -- Informer 回流
  conditions               jsonb NOT NULL DEFAULT '[]',      -- Informer 回流
  quota_status             jsonb NOT NULL DEFAULT '[]',      -- Informer 回流：[{pool, name, ready, used, message}]
  message                  text NOT NULL DEFAULT '',
  last_modified_by         text NOT NULL DEFAULT '',         -- 来自 X-Axisml-User

  created_at               timestamptz NOT NULL DEFAULT now(),
  updated_at               timestamptz NOT NULL DEFAULT now(),
  deleted_at               timestamptz                       -- 软删除标记；retention 期内保留
);

CREATE UNIQUE INDEX tenants_name_active_uniq ON tenants (name) WHERE deleted_at IS NULL;
CREATE INDEX tenants_deleted_at        ON tenants (deleted_at);
CREATE INDEX tenants_business_unit     ON tenants (business_unit);
CREATE INDEX tenants_created_at        ON tenants (created_at DESC);
CREATE INDEX tenants_sync_pending      ON tenants (desired_spec_hash, applied_spec_hash) WHERE desired_spec_hash <> applied_spec_hash;
```

- `tenants_name_active_uniq` 是 partial unique index：存活租户 `name` 唯一；软删后同名可以重建（与 K8s namespace 复用规则匹配，见 [tenant-operator §4.6.1](tenant-operator.md)）。
- `tenants_sync_pending` 是 partial index，只对未同步行建索引——reconciler 扫描代价 O(待同步数) 而非 O(全表)。

### 3.2 字段归属

| 字段 | 写入方 | 备注 |
| --- | --- | --- |
| `id` | API 层（uuid_generate_v4） | 同时写入 Tenant CR `metadata.labels[axisml.io/tenant-id]`；删除并重建同名后 id 会变 |
| `name` / `display_name` / `description` / `business_unit` / `annotations` | API 层 | **`description` / `business_unit` 升级为 PG 一级字段**，富文本（含 Unicode / 中文）由 PG 原生承载；reconciler 渲染到 CR `spec.annotations[axisml.io/description, axisml.io/business-unit]` 作为 kubectl 调试位 |
| `namespace_name` | API 层 | 创建后不可变 |
| `namespace_labels` / `namespace_annotations` | API 层 | 只在 Namespace 首次创建时落地（[tenant-operator §4.6.1](tenant-operator.md)） |
| `quotas` / `init_resources` / `suspended` | API 层 | `quotas` 内每条 `(pool, name)` 创建后不可变（参见 §7） |
| `desired_spec_hash` | API 层 | 每次 mutation 重算 |
| `applied_spec_hash` | reconciler | 每次成功 patch CR 后写入；与 `desired_spec_hash` 相等表示已同步 |
| `phase` / `namespace_ready` / `conditions` / `quota_status` / `message` | informer | 由 Tenant CR `status.*` 回流；reconciler 删除完成（DELETE 事件）后置 `phase=Deleted` |
| `last_modified_by` | API 层 | 来自 `X-Axisml-User` |
| `deleted_at` | DELETE / restore 端点 | retention 期满后由后台 GC 物理清理 |

### 3.3 历史保留与清理

- `deleted_at IS NOT NULL` 的行保留 `retention.archivedTenantDays`（默认 365 天）；可由 Platform 端通过 `GET /api/v1/tenants?includeArchived=true` 列出，或 `POST /api/v1/tenants/{name}/restore` 恢复（§6.2）。
- 超过 retention 后由 reconciler 启动时的 GC 任务批量物理删除：
  ```sql
  DELETE FROM tenants WHERE deleted_at < now() - interval '<retention> days';
  ```
- 物理删除前 reconciler 确认对应 Tenant CR 不存在（防止 retention 期间被外部恢复重建后误清）。

## 4. 写路径与同步

参考 [compute.md §3.4 写路径](compute.md#34-写路径outbox--reconciler) 与 [§3.5 双 hash 同步](compute.md#35-desiredapplied-spec-hash-双-hash-机制)；cluster-manager 复用同一套机制，但 outbox 直接借用 `tenants` 表自身的 `desired_spec_hash` 列，不引入单独 outbox 表。

### 4.1 API 写

API 层的每一次 mutation：

1. 在一个事务内更新 `tenants` 行（含 `quotas` / `init_resources` 等 jsonb 字段）；
2. 用规范化 JSON 序列化 spec 视图（即 CR 渲染输入：含 `display_name` / `description` / `business_unit` / `annotations` / `namespace_*` / `quotas` / `init_resources` / `suspended` / `deleted_at`；不含 `applied_*` / `phase` / `conditions` / `updated_at` 等会持续变化的列）；
3. `desired_spec_hash = sha256(canonical_json)`；
4. COMMIT。

`tenants_sync_pending` partial index 立刻能被 reconciler 发现。

### 4.2 Reconciler

后台协程（leader-only）按 `--reconciler.interval`（默认 10s）周期扫描：

```sql
SELECT id, name, ... FROM tenants
WHERE desired_spec_hash <> applied_spec_hash
ORDER BY updated_at
LIMIT N FOR UPDATE SKIP LOCKED;
```

对每一行：

- `deleted_at IS NOT NULL`：调 K8s `Delete()` Tenant CR；CR 不存在则直接置 `applied_spec_hash = desired_spec_hash`（已收敛）。Informer 看到 DELETE 事件后置 `phase=Deleted`。
- 否则：调 CR 渲染函数生成 desired Tenant spec → K8s `Patch()`（server-side apply）→ 成功后 `applied_spec_hash = desired_spec_hash`、`updated_at = now()`。

CR 渲染逻辑映射（PG → Tenant CR）：

| PG 列 | Tenant CR 路径 |
| --- | --- |
| `name` | `metadata.name` |
| `id` | `metadata.labels[axisml.io/tenant-id]` |
| `display_name` | `spec.displayName` |
| `description` | `spec.annotations[axisml.io/description]` |
| `business_unit` | `spec.annotations[axisml.io/business-unit]` |
| `annotations` 中其它 key | `spec.annotations[...]` 合并 |
| `namespace_name` / `namespace_labels` / `namespace_annotations` | `spec.namespace.{name, labels, annotations}` |
| `quotas` | `spec.quotas[]` |
| `init_resources` | `spec.initResources` |
| `suspended` | `spec.suspended` |

冲突 / 重试：CR patch 失败时（K8s 409 或网络错误）保持 `applied_spec_hash` 不变，下一轮 tick 重试。连续 N 次失败的行会被打 `tenants.message` 提示运维。

### 4.3 Informer 回流

watch Tenant CR：

- **ADD / UPDATE**：把 `status.{phase, namespaceReady, conditions, quotas}` 写回 `tenants` 同名列；**不动 spec 列**（spec 权威在 PG，CR spec 漂移由 §4.4 处理）。
- **DELETE**：找到对应 PG 行，置 `phase=Deleted`；保留所有其他字段供历史查询。
- informer 回写只 patch 上述 status 列，绝不触发 `desired_spec_hash` / `applied_spec_hash` 重算。

### 4.4 外部漂移修正

如果有人绕过 cluster-manager 直接 `kubectl edit tenant` 修改 CR spec：

- informer 看到 spec 漂移时**不修正 PG**（PG 是权威，CR 不能改它）；
- 但下一轮 reconciler tick 会发现 CR spec ≠ PG 期望，重新 patch CR 把它擦回 PG 状态——server-side apply 的 conflict detection 会处理 field manager 冲突；
- 阶段二 admission webhook 上线后直接拒绝非 cluster-manager 的写请求。

## 5. 状态机

```
[POST] ──▶ Creating ──(informer: namespaceReady=true)──▶ Active ◀─[unsuspend]─ Suspended
                                                           │ │
                                                           │ └─[suspend]──────▶ Suspended
                                                           ▼
                                                        Failed (operator 写 status.phase)
                                                           │
                                                        (自愈)
                                                           │
                                                           ▼
任一活跃状态 ──[DELETE 软删 + reconciler 删 CR]──▶ Deleting ──(informer DELETE)──▶ Deleted
                                                                                  │
                                                                                  ▼
                                                                         retention 到期被 GC 物理清理

Deleted ──[POST /restore]──▶ Creating（清空 deleted_at，重算 desired_spec_hash）
```

`Failed` 是非终态——operator 自愈后下一次 informer 事件会自然回到 `Active`。`Deleted` 在 PG 中长期保留（直到 retention 到期），可通过 `GET /api/v1/tenants?includeArchived=true` / `restore` 端点恢复。

---

## Part II — API

## 6. Tenant API

### 6.1 字段校验

cluster-manager 在请求层做 **DNS-1123 + 长度** 校验，把违反规则的请求直接 4xx 拒绝，避免无效行进入 PG：

| 字段 | 校验 |
| --- | --- |
| `name` | DNS-1123；长度 3–40；`[a-z0-9-]`；首尾字母数字；无连续 `--` |
| `displayName` | 长度 1–100，允许 Unicode（含中文） |
| `description` | 长度 ≤ 1000，允许 Unicode |
| `businessUnit` | 长度 ≤ 100，允许 Unicode |
| `annotations` 自定义 key | qualified name（K8s annotation 规则） |
| `namespace.name` | DNS-1123；非系统 namespace（denylist 与 tenant-operator Helm values 同源） |
| `quotas[].pool` | DNS-1123 |
| `quotas[].name` | DNS-1123；同一 tenant 内 `(pool, name)` 唯一 |
| `quotas[].max[k]` | 必填；非负 |
| `quotas[].min[k]` | 可选；非负；`min[k] ≤ max[k]` |

更深的语义校验（源 Secret / ConfigMap 是否存在、`(pool, name)` 是否引用有效 ResourcePool）由 tenant-operator 在 reconcile 阶段写到 CR `status.message`，进而由 informer 回流到 PG `tenants.message`；cluster-manager 不前置查询 K8s 资源。

### 6.2 端点

#### `POST /api/v1/tenants`

请求体：

```json
{
  "name": "team-a",
  "displayName": "Team A",
  "description": "推理团队，负责 LLM 推理服务的研发",
  "businessUnit": "基础架构-推理平台",
  "annotations": {},
  "namespace": { "name": "team-a", "labels": {}, "annotations": {} },
  "quotas": [
    { "pool": "default", "name": "default", "min": {}, "max": { "cpu": "100", "memory": "200Gi" } }
  ],
  "initResources": {
    "imagePullSecrets": [],
    "secrets": [],
    "configMaps": [],
    "serviceAccounts": []
  }
}
```

处理：

1. 单事务 INSERT `tenants` 行，`id=uuid_generate_v4()`、`desired_spec_hash=sha256(...)`、`phase='Creating'`；
2. 若 `name` 已存在未软删行（partial unique 触发）→ 409 `AlreadyExists`；
3. 同名软删行存在但已软删 → 允许新建（新行 `id` 不同，CR 标签 `axisml.io/tenant-id` 也会更新）；
4. 返回 201 + 当前 view（`phase='Creating'`，`namespace_ready=false`）；
5. 下一轮 reconciler tick 创建 Tenant CR；后续 status 字段由 informer 回流。

#### `GET /api/v1/tenants/{name}`

按 `name` + `deleted_at IS NULL` 查 PG；返回完整字段（含 status 列）。query `?includeArchived=true` 时也匹配软删行；同名活跃 + 软删都存在时返回活跃行（partial unique 保证活跃唯一）。

#### `GET /api/v1/tenants`

纯 PG 查询，支持：

| 参数 | 含义 |
| --- | --- |
| `q` | 关键字模糊匹配 `name` / `display_name` |
| `business_unit` | 精确匹配 |
| `phase` | 精确匹配（`Creating` / `Active` / `Suspended` / `Failed` / `Deleting` / `Deleted`） |
| `limit` / `continue` | 分页（continue token 由 cluster-manager 颁发） |
| `includeArchived` | true 时包含 `deleted_at IS NOT NULL` 行 |
| `sortBy` | `created_at` / `updated_at`；缺省 `created_at desc` |

#### `PATCH /api/v1/tenants/{name}`

请求体使用 RFC 7396 JSON Merge Patch；只接受可变字段：`displayName` / `description` / `businessUnit` / `annotations` / `namespace.labels` / `namespace.annotations` / `initResources`。

不可变字段（`name` / `namespace.name`）在请求层 4xx 拒绝，从不写 PG。

处理：

1. 应用合并 patch；
2. 重算 `desired_spec_hash`；
3. 若 hash 未变（语义无变化）→ 直接返回 200 不写 PG；
4. 否则 UPDATE PG 行，返回 200 + 新 view（CR 同步异步进行）。

#### `POST /api/v1/tenants/{name}/suspend` / `unsuspend`

`UPDATE tenants SET suspended = $1, desired_spec_hash = $2 WHERE name = $3 AND deleted_at IS NULL`。reconciler 异步 patch CR `spec.suspended`。

#### `DELETE /api/v1/tenants/{name}`

软删：`UPDATE tenants SET deleted_at = now(), desired_spec_hash = $1 WHERE name = $2 AND deleted_at IS NULL`。reconciler 异步删除 Tenant CR。返回 200。

幂等：再次 DELETE 已软删的 tenant 直接返 200（`deleted_at IS NULL` 谓词不匹配，不改 PG）。

#### `POST /api/v1/tenants/{name}/restore`

恢复软删 tenant：`UPDATE tenants SET deleted_at = NULL, desired_spec_hash = $1, phase = 'Creating' WHERE name = $2 AND deleted_at IS NOT NULL`。

前置校验：当前没有同名活跃行（partial unique 兜底；若违反返 409）。reconciler 在下一轮 tick 重新创建 Tenant CR。

> 这是 PG-first 模型 free 的副作用——记录原本就在，恢复就是一条 UPDATE。Platform 「已归档租户」UI 入口由 [tenant.md §11](../platform/tenant.md) 暴露。

## 7. Quota API

Quota 不是独立 CRD——是 `tenants.quotas` jsonb 中的一项。所有 Quota mutation 都翻译为对该 jsonb 的事务性修改 + `desired_spec_hash` 重算，由 reconciler 异步 patch 到 Tenant CR `spec.quotas[]`。

| Endpoint | 方法 | 说明 |
| --- | --- | --- |
| `GET /api/v1/tenants/{name}/quotas` | `GET` | 返回 `tenants.quotas`（spec）+ `tenants.quota_status`（informer 回流的每条 `ready` / `used` / `message`） |
| `POST /api/v1/tenants/{name}/quotas` | `POST` | 在 jsonb 中追加 `{pool, name, min, max}`；`(pool, name)` 已存在 → 409 |
| `PATCH /api/v1/tenants/{name}/quotas/{pool}/{quotaName}` | `PATCH` | 更新某条的 `min` / `max`；定位失败 → 404 |
| `DELETE /api/v1/tenants/{name}/quotas/{pool}/{quotaName}` | `DELETE` | 从 jsonb 中移除一条；reconciler patch CR 后 tenant-operator 会显式 Delete 对应 ElasticQuota CR（[tenant-operator §4.6.2](tenant-operator.md)） |

并发更新由 PG 行级锁保护（`UPDATE ... WHERE name = $1 AND deleted_at IS NULL` 的事务序列化）；不需要 K8s API resourceVersion 重试逻辑——CR 写入完全由 reconciler 串行执行。

---

## Part III — 实施与验证

## 8. 实现路径

### 8.1 阶段一：MVP

| 模块 | 范围 | 完成信号 |
| --- | --- | --- |
| HTTP server | Gin + oapi-codegen 生成 server stub；problem+json 中间件 | `make cluster-manager-build` 通过；`/healthz` / `/readyz` 工作 |
| PG 迁移 | `tenants` 表创建（§3）；`migrate` 子命令；helm pre-install hook | `make cluster-manager-migrate` 干净 |
| Tenant / Quota API | §6 / §7 所有端点 | 单元测试覆盖请求层校验；integration 覆盖 PG → CR 端到端 |
| Reconciler | §4.2 + §4.3 双 hash 扫描 + CR patch + informer 回流 | leader election Lease 成立；双 hash 收敛 |
| RBAC | §2.5 ClusterRole + leader Lease Role | helm install 后服务可正常 list / patch tenants + 获取 Lease |
| Helm | `deploy/helm/axisml-system/templates/cluster-manager/`（含 migrate-job） | helm install 后 `kubectl get deploy/axisml-cluster-manager` Ready |
| Restore 端点 | `POST /tenants/{name}/restore` + `includeArchived` 查询 | 软删 + 恢复 happy path 通过 integration |

### 8.2 阶段二

1. **Admission webhook**：把请求层校验前移到 K8s admission，硬阻断非 cluster-manager（field manager 检查）的写请求，呼应 [tenant-operator §4.9](tenant-operator.md) 的兜底层。
2. **批量操作**：`POST /api/v1/tenants:batchCreate` 等，便于平台批量初始化。
3. **审计持久化**：独立 `cluster_manager_audits` 表记录 mutation 流水（与 Platform `audit_logs` 分层：本层记录"PG mutation 发生了什么"，Platform 层记录"是谁在 UI 上点了什么"）。
4. **外部漂移告警**：reconciler 检测到非 cluster-manager 字段管理者写入 CR 时打 K8s Event + Prometheus 指标 + 自动还原。

### 8.3 从旧版无 PG 形态迁移

> 仅对已有部署相关。如果是新部署可忽略本节。

旧版 cluster-manager 直接 CRUD Tenant CR、无 PG。升级到本版本时需要一次性"灌种"——`migrate` 子命令在创建 schema 后执行：

```
LIST tenants from K8s API
  → for each: 反向映射 spec/annotations 到 PG 列；
              phase / namespace_ready 等 status 字段也从 CR status 拷一份；
              desired_spec_hash = applied_spec_hash = sha256(reverse render)。
```

灌种完成后 reconciler 启动；后续所有 mutation 都走 PG-first 路径。整个过程对 Platform 端透明（API 兼容）。

## 9. 测试

- **单元**（`components/cluster-manager/internal/.../*_test.go`）：
  - 请求层校验、字段映射、CR 渲染函数；
  - `desired_spec_hash` 计算的稳定性（同输入同输出、不同输入不同输出，字段顺序无关）；
  - reconciler 单步：给定 PG 行与 CR 当前状态，断言生成的 Patch 正确；
  - informer 回流：模拟 ADD / UPDATE / DELETE event，断言写回的 PG 列正确；
  - leader election：mock Lease 客户端，断言非 leader 副本不启动后台协程。
- **integration**（`components/cluster-manager/test/integration/` 独立 Go module）：
  - testcontainers PostgreSQL + envtest（apiserver + Tenant CRD）；
  - **happy path**：POST tenant → reconciler 创建 CR → 模拟 operator 写 status → informer 回流 → GET 看到 `phase=Active`；
  - **软删与恢复**：DELETE → CR 被删除 → restore → CR 重新创建 → `phase` 回到 `Active`；
  - **Quota CRUD**：API mutation → PG jsonb → reconciler patch CR → informer 回流 `quota_status`；
  - **外部漂移修正**：在 PG 行就绪后直接 `kubectl edit tenant` 改 `spec.displayName`，断言 reconciler 在下一轮 tick 把它擦回 PG 期望态；
  - **retention GC**：人为前推 `deleted_at`，重启服务，断言超期行被物理清理；
  - **denylist + DNS-1123 校验**：跨字段拒绝矩阵。

仓库当前不维护 minikube 驱动的 e2e 层；端到端 Namespace / ElasticQuota / initResources 落地由 [tenant-operator §6](tenant-operator.md) 自身的 integration 覆盖。

## 10. 相关引用

- [docs/system_design/overview.md](../overview.md) 概述了 cluster-manager 在控制平面里的位置。
- [docs/system_design/core/compute.md](compute.md) §3.4 / §3.5 是本设计写路径与双 hash 同步的参考样板。
- [docs/system_design/core/tenant-operator.md](tenant-operator.md) 描述本服务下发的 Tenant CR 的具体落地行为；该文档需配套调整以反映 "CR 是 cluster-manager 派生产物" 的新关系（详见本文 §4.4 / §8.2 关于 admission webhook 的描述）。
- [docs/system_design/platform/tenant.md](../platform/tenant.md) 描述 Platform 如何消费本 API；本次重写新增的 `description` / `business_unit` 一级字段、`restore` 端点、`includeArchived` 查询参数都对应 Platform 端 [§11 后续迭代](../platform/tenant.md#11-后续迭代) 中的「租户软删除」「展示元数据 annotation 规范化」两条。
