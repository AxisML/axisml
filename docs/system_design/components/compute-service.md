# AxisML Compute Service 概要设计

## 1. 定位与边界

ML 工作负载 + 多租户管控面：以 PostgreSQL 为权威，承载 Tenant / Quota / Job / Service 的元数据，把 `Tenant` / `MLJob` / `MLService` CR 当作 PG 行的派生产物下发到 K8s。

| 做 | 不做 |
| --- | --- |
| Tenant CRUD、suspend / unsuspend、软删与 restore | Namespace / ElasticQuota / initResources 落地 (→ [tenant-operator.md](tenant-operator.md)) |
| Quota（内联 `tenants.quotas` jsonb）CRUD | 修改 Node label / taint（admin 手工维护） |
| Job / Service CRUD、cancel / scale、软删 | 直接创建 Pod / Deployment / PodGroup 等底层资源 (→ [compute-operator.md](compute-operator.md)) |
| `Tenant` / `MLJob` / `MLService` spec 下发 + status 回流 | ResourcePool / ResourceUnit 词汇 (→ [cluster-manager.md](cluster-manager.md)) |
| Pod 列表 / Pod 日志 / 事件端点透传 kube-apiserver | 用户认证与角色鉴权 (→ [auth.md](../auth.md)) |
| 工作区列表（`kind='workspace'` 过滤） | 工作区业务语义本身 (→ [platform.md](platform.md)) |

**namespace 字段语义**：`jobs` / `services` 表的 `namespace text` 是 tenant 标识符（= `tenants.name`），逻辑分区键；Compute 内部 join `tenants` 表得到 `spec.namespace.name` 用于 CR 下发。Platform 在调 compute 前已经把 ResourcePool / ResourceUnit 展开成 `nodeSelector` / `tolerations` / `requests` / `limits` 原语写进请求体；compute 不感知 pool/unit 概念。

## 2. 架构

### 2.1 上下文

```
        ┌──────────────┐  REST + X-Axisml-User   ┌──────────────────┐
        │  Platform    │ ───────────────────────▶│  Compute (Go)    │
        └──────────────┘                          └──────┬───────────┘
                                                         │ PG 读写 / CR patch + watch
                                          ┌──────────────┼──────────────────────────────┐
                                          ▼              ▼                              ▼
                                ┌─────────────────┐  ┌──────────────────────┐  ┌────────────────┐
                                │  PostgreSQL     │  │   K8s API            │  │ tenant-operator│
                                │  tenants/jobs/  │  │   Tenant / MLJob /   │  │ compute-operator│
                                │  services       │  │   MLService          │  └────────────────┘
                                └─────────────────┘  └──────────────────────┘
```

### 2.2 内部结构

```
┌──────────────────────────── Compute (Go) ─────────────────────────────┐
│  HTTP API (Gin) ──写──▶  PG (generation + phase='Creating')          │
│        ▲                                │                              │
│        │ 读                              ▼                              │
│        │                       Reconciler goroutines (leader-only)    │
│        │            ┌──── tenant ────┬──── job ────┬──── service ───┐ │
│        │            └────────────────┴─────────────┴────────────────┘ │
│        │                                │                              │
│        │                                ▼ Create / Patch / Delete     │
│        │                       Tenant / MLJob / MLService CR          │
│        │                                │ status                       │
│        └──── PG status 列 ◀──── Informer (leader-only, shared cache)   │
└────────────────────────────────────────────────────────────────────────┘
```

三条 Reconciler 共享同一 leader Lease、同一 PG 连接池；三条 Informer 共享 `SharedInformerFactory`。

## 3. 核心模型

| 实体 | 含义 | 标识键 | 状态机 | 对应 CR |
| --- | --- | --- | --- | --- |
| Tenant | 租户 | `id` (uuid) / `name`（集群内全局唯一） | `Creating \| Active \| Suspended \| Failed \| Deleting \| Deleted` | `Tenant`（cluster-scoped） |
| Quota | 配额，内联在 `tenants.spec.quotas[]` | `(tenant.name, pool, name)` | 无独立状态 | 每条 1:1 渲染为 ElasticQuota CR（由 tenant-operator 落地） |
| Job | 一次性训练 / 离线任务 | `id` (uuid) / `(namespace, name)` | `Creating \| Pending \| Running \| Succeeded \| Failed \| Canceling \| Cancelled \| Deleting \| Deleted` | `MLJob`（namespaced） |
| Service | 常驻在线服务 / 工作区 | `id` (uuid) / `(namespace, name)` | `Creating \| Pending \| Ready \| Degraded \| Failed \| Deleting \| Deleted` | `MLService`（namespaced） |

字段级 schema 见 [database.md §3](../database.md#3-compute)；CR spec 字段见 [tenant-operator.md](tenant-operator.md) / [compute-operator.md](compute-operator.md)。

**通用 PG 约定**：所有表带 `id uuid` / `created_at` / `updated_at` / `deleted_at`；所有 UNIQUE 实现为 partial unique index `WHERE deleted_at IS NULL`（软删行不占用唯一键，同名可再次创建）；`name` 统一 DNS-1123 校验，长度 3–40。CR-backed 对象额外打 `axisml.io/{tenant,job,service}-id=<uuid>` label 作为稳定锚点（`metadata.name` 因软删可重用，UUID 永久唯一）；`services` 还会同步打 `axisml.io/service-kind=<service|workspace>` label，便于 `kubectl` selector 区分工作区与普通服务（compute / operator 不按 kind 改变行为）。

**扩展元数据 + 分组维度**：`tenants` / `jobs` / `services` 表均带 `labels jsonb` + `annotations jsonb` 双字段，对齐 K8s 风格语义，统一约定见 [database.md §1.6](../database.md#16-扩展元数据-labels--annotations)。Platform 用 `labels.axisml.io/project` 等 label 在 compute 之上实现 project / experiment / 自定义分组（见 [platform.md](platform.md)）。list 端点支持 `?labelSelector=` K8s 语法。两类扩展位均 **PG-only、不下发 CR、不 `+generation`**。

## 4. 核心功能

### 4.1 Tenant

**状态机**：

```
[POST]──▶ Creating ──(namespaceReady)──▶ Active ◀─[unsuspend]─ Suspended
                                            │ │
                                            │ └─[suspend]──▶ Suspended
                                            ▼
                                          Failed ──(自愈)──▶ Active

任一活跃态 ──[DELETE 软删]──▶ Deleting ──(CR DELETE)──▶ Deleted
                                                       │
                                                       ├─[restore]─▶ Creating
                                                       └─ retention 到期 ──▶ 物理清理
```

**行为约束**：

| 操作 | PG 写 | CR 影响 | 备注 |
| --- | --- | --- | --- |
| 创建 | insert `Creating` 行（`generation=1`） | reconciler 创建 Tenant CR | DNS-1123 校验由 API 层兜底；`name` 集群内全局唯一 |
| 更新 spec（`spec.namespace.labels` / `spec.namespace.annotations` / `spec.quotas[].{min,max}` / `spec.initResources` / `spec.suspended`） | update `spec` + `generation += 1` | reconciler patch CR | `spec.namespace.name` / `spec.quotas[].{pool,name}` 不可变 |
| 更新顶层 PG 元数据（`display_name` / `description` / `labels` / `annotations`） | update 行 | **不影响 CR** | 不 `+generation`；扩展位见 [database.md §1.6](../database.md#16-扩展元数据-labels--annotations) |
| 软删 | `phase='Deleting'` + `deleted_at = now()` + `generation += 1` | reconciler 删 CR | 行保留到 retention |
| 恢复 | `deleted_at = NULL` + `generation += 1` | reconciler 重建 CR | 仅适用 `phase='Deleted'` 行 |

`Failed` 是非终态，operator 自愈后自然回到 `Active`。

**与 Tenant CR 契约**：

| Tenant `status.phase` | PG `status` |
| --- | --- |
| `Pending` | `Creating` |
| `Active` | `Active`（含 `namespaceReady=true`） |
| `Suspended` | `Suspended` |
| `Failed` | `Failed`（可自愈） |

### 4.2 Quota

Quota 没有独立 CRD——是 `tenants.spec.quotas[]` jsonb 中的一项，由 tenant-operator 渲染成 ElasticQuota CR 落地。

| 操作 | PG 写 | CR 影响 |
| --- | --- | --- |
| 新增 | append jsonb 项 + `generation += 1` | reconciler patch `Tenant.spec.quotas[]` |
| 修改 `min` / `max` | update jsonb 项 + `generation += 1` | reconciler patch |
| 删除 | remove jsonb 项 + `generation += 1` | reconciler 删除对应 ElasticQuota CR |
| 用量回流 | informer 写入 `status.quotas[i].used` | — |

**不变约束**：`(pool, name)` 一旦创建即不可变；改名 = 先删后建。

### 4.3 Job

**状态机**：

```
Creating ──(Informer ADD)──▶ Pending ──▶ Running ──▶ Succeeded / Failed
                                │           │
                                │ cancel    │ cancel
                                └───────────┴──▶ Canceling ──(Suspended condition)──▶ Cancelled

任一非 Canceling/Deleting/Deleted ──[DELETE]──▶ Deleting ──(CR 确认清理 + deleted_at)──▶ Deleted
```

**行为约束**：

| 操作 | PG 写 | CR 影响 | 备注 |
| --- | --- | --- | --- |
| 提交 | insert `Creating` 行 + spec 快照（已含展开后的 `nodeSelector` / `tolerations` / `resources`） | reconciler `Create()` MLJob | 创建后 spec 不可变；无后续 mutation，仅靠 `phase='Creating'` 谓词推进 |
| cancel | `phase='Canceling'` + `message='user cancelled'` | reconciler `patch spec.runPolicy.suspend=true` | `Creating` 状态拒绝；要求改用 DELETE |
| 更新 PG 元数据（`display_name` / `description` / `labels` / `annotations`） | update 行 | **不影响 CR** | Job spec 不可变，但 PG 扩展位任意阶段可改 |
| 软删 | `phase='Deleting'` + `deleted_at=now()` | reconciler `Delete()` CR；Informer DELETE → `Deleted` | 任一非 `Deleting`/`Deleted` 状态适用 |
| Pod 列表 / Pod 日志 / Pod 事件 / Job 事件 | — | 按 `axisml.io/job-id` label list Pod；按 Pod 名透传 Pod Log；按 `involvedObject` 过滤 Event（Pod 端点只回 Pod 事件，Job 端点只回 MLJob/PodGroup 事件） | 详见 [apis/compute.yaml](../apis/compute.yaml) `Jobs` tag |
| list 过滤 | — | `?labelSelector=` 支持 K8s 语法，常用 `axisml.io/project=<id>` | 见 [database.md §1.6](../database.md#16-扩展元数据-labels--annotations) |

`Succeeded` / `Failed` / `Cancelled` 为运行终态；`Deleted` 为软删终态。`Cancelled` PG 行保留（`deleted_at IS NULL`），用户可再次 DELETE。

**与 MLJob CR 契约**：

| MLJob `status.phase` | PG `status` |
| --- | --- |
| `Pending` | `Pending` |
| `Running` | `Running` |
| `Succeeded` | `Succeeded`（终态） |
| `Failed` | `Failed`（终态） |
| `Pending`/`Running` + `conditions[type=Suspended,status=True,reason=CancelRequested]` | `Cancelled`（PG 端推进，随后入队 `Delete()` 回收资源） |

Cancel patch 与自然完成竞速时，operator 优先保留终态 phase 与 `finishedAt`。

### 4.4 Service

**状态机**：

```
Creating ──(Informer ADD)──▶ Pending ──(ready=desired, desired>0)──▶ Ready ⇄ Degraded ──▶ Failed
                                                                      ▲                     │
                                                                      └─────── 自愈 ────────┘

任一非 Deleting/Deleted ──[DELETE]──▶ Deleting ──(CR 确认清理 + deleted_at)──▶ Deleted
```

`Ready` / `Degraded` / `Failed` 均为**非终态**——operator 自愈后 Informer 回流可让 `Failed → Degraded → Ready` 自然恢复；只有 `Deleted` 为最终终态。

**行为约束**：

| 操作 | PG 写 | CR 影响 |
| --- | --- | --- |
| 创建 | insert `Creating` 行 + spec 快照（含展开后的 `nodeSelector` / `tolerations` / `resources`） | reconciler `Create()` MLService（含 `axisml.io/service-{id,kind}` label） |
| scale | 更新 `spec.roles[0].replicas` + `generation += 1` | reconciler `generation>observed_generation` 触发 patch `spec/roles/0/replicas` |
| 更新 PG 元数据（`display_name` / `description` / `labels` / `annotations`） | update 行 | **不影响 CR**；不 `+generation` |
| 软删 | `phase='Deleting'` + `deleted_at=now()` | reconciler `Delete()` CR；Informer DELETE → `Deleted` |
| Pod 列表 / Pod 日志 / Pod 事件 / Service 事件 | — | 按 `axisml.io/service-id` label list Pod；按 Pod 名透传 Pod Log；按 `involvedObject` 过滤 Event（Pod 端点只回 Pod 事件，Service 端点只回 MLService/底层 Workload/HTTPRoute 事件） |
| list 过滤 | — | `?labelSelector=` 支持 K8s 语法，常用 `axisml.io/project=<id>` |

Service 无 cancel 语义；除 `roles[0].replicas` 外其他 spec 字段不可变。`kind` 创建后不可变。

**与 MLService CR 契约**：

| 条件 | PG `status` |
| --- | --- |
| `ready_replicas == desired_replicas && desired_replicas > 0` | `Ready` |
| `0 < ready_replicas < desired_replicas` | `Degraded` |
| `ready_replicas == 0 && desired_replicas > 0` 且 CR `phase=Pending` | `Pending` |
| `ready_replicas == 0 && desired_replicas > 0` 且 CR `phase=Failed` | `Failed`（可自愈） |
| `desired_replicas == 0` | `Pending` |

**kind 过滤**：`GET /api/v1/namespaces/{ns}/services?kind=workspace` 供 [Platform 工作区](platform.md) 在同一张表上区分 `kind='service'` 与 `kind='workspace'`；`kind` 创建后不可变，Compute 不按 `kind` 改变行为，仅作分类与过滤。

## 5. 关键机制

### 5.1 写路径：内嵌 Outbox + 谓词扫描

无独立 outbox 表——借用业务表自身的 `status` / `deleted_at` / `generation` 列。API 同步路径只写 PG（业务校验 → 事务内写状态 + spec 快照 + `generation += 1` → commit → 返回业务 ID）；reconciler goroutine 在 leader 副本按以下谓词扫描分派：

| 谓词 | 动作 | 适用模块 |
| --- | --- | --- |
| `phase='Creating' AND deleted_at IS NULL` | `Create()` CR（带 `axisml.io/<resource>-id` label；409 视为成功） | Tenant / Job / Service |
| `phase='Canceling'` | `patch MLJob.spec.runPolicy.suspend=true` | Job |
| `phase='Deleting'` | `Delete()` CR；Informer DELETE 推进 `Deleted` | Tenant / Job / Service |
| `generation <> observed_generation AND deleted_at IS NULL` | `Patch()` CR；成功后 `observed_generation = generation` | Tenant / Service |

失败按指数退避重试，错误写入业务记录的 `message`。PG 行不再满足任何谓词后，reconciler 不再下发——自然结束 / 自愈 / 外部误删由 Informer 回流推进。

### 5.2 generation spec 同步

```
API mutation
     │
     ▼  (事务内) spec 写入 + generation = generation + 1
     │
     ▼  partial index `WHERE generation <> observed_generation AND deleted_at IS NULL`
     │
Reconciler (leader-only)
     │
     ▼  Patch CR → observed_generation = generation
```

语义对齐 K8s `metadata.generation` / `status.observedGeneration`，统一约定见 [database.md §1.4](../database.md#14-generation--observed_generation)。

| 资源 | 是否启用 | 允许变更字段 |
| --- | --- | --- |
| Tenant | 是 | `spec.namespace.labels`、`spec.namespace.annotations`、`spec.quotas[].{min,max}`、`spec.initResources`、`spec.suspended` |
| Service | 是 | `spec.roles[0].replicas` |
| Job | 否 | 不可变；cancel 走 `Canceling` 谓词单独 patch suspend |

### 5.3 状态回流（Informer）

三条独立 Informer，通过 `k8sclient` 的 `SharedInformerFactory` 共享 cache，仅 leader 副本运行。

| Informer | 监听对象 | 主要回写字段 |
| --- | --- | --- |
| Tenant Informer | `Tenant` CR | 整块写 `tenants.status` jsonb（`{phase, message, namespaceReady, conditions[], quotas[].used}`）；按 §4.1 phase 映射表推进 |
| MLJob Informer | `MLJob` CR | 整块写 `jobs.status` jsonb（`{phase, message, startedAt, finishedAt, conditions[]}`）；按 §4.3 phase 映射表推进 |
| MLService Informer | `MLService` CR | 整块写 `services.status` jsonb（`{phase, message, readyReplicas, endpoint, conditions[]}`）；按 §4.4 条件表推进 |

启动时 `List` 做差异 upsert 与孤儿对账；`Watch` 事件入各自 work queue；单 worker 串行 reconcile；以 `resourceVersion` / `generation` 作乐观并发字段。

### 5.4 孤儿与补偿

Compute **不反向重建 CR**。Informer 观察到 CR DELETE 事件后按 PG 当前 `status` 分流：

| PG 当前状态 | 处理 |
| --- | --- |
| `Canceling` | 幂等忽略；`Cancelled` 只由 Suspended condition 推进 |
| `Deleting` | 正常级联清理，推进 `Deleted` |
| Job 在 `Pending`/`Running`（外部误删） | 推 `phase=Cancelled` + `status.finishedAt` + `status.message='external delete'`，不补偿重建 |
| Service 在 `Pending`/`Ready`/`Degraded`/`Failed`（外部误删） | 写 `phase=Deleting` + `deleted_at=now()` + `status.message='external delete'`，下一轮谓词幂等确认后推 `Deleted` |
| Tenant 在 `Active`/`Suspended`/`Failed`（外部误删） | 下一轮 reconciler 检测到 `generation <> observed_generation` 时重建 CR 恢复期望状态（PG 为权威） |
| 已终态 | 忽略 |

**正向孤儿**（PG `Creating` 但无 CR，且 `deleted_at IS NULL`）属 Outbox 正常窗口，reconciler 下一轮幂等重试 `Create()`。**反向孤儿**（CR 存在但 PG 无行或已 `Deleted`）：默认删除 CR 并记录审计。

## 6. 接口契约

| 类别 | 内容 | 引用 |
| --- | --- | --- |
| 对外 REST（K8s 风格） | `/api/v1/namespaces[/{namespace}]`（Tenant CRUD）、`/api/v1/namespaces/{namespace}/{suspend,unsuspend,restore}`、`/api/v1/namespaces/{namespace}/quotas[/{pool}/{name}]`、`/api/v1/namespaces/{namespace}/jobs[...]`、`/api/v1/namespaces/{namespace}/services[...]` | [apis/compute.yaml](../apis/compute.yaml) `Tenants` / `Quotas` / `Jobs` / `Services` tag |
| 下发 CR | `Tenant`（`axisml.io/v1alpha1`, cluster-scoped）、`MLJob` / `MLService`（namespaced）；Compute 是唯一 `spec` 写者 | [tenant-operator.md](tenant-operator.md) / [compute-operator.md](compute-operator.md) |
| 回流字段 | `tenants.status` / `jobs.status` / `services.status`（jsonb 整块） | [database.md §3](../database.md#3-compute) |
| 不变量 | CR `metadata` / `spec` 单写（Compute 写）；CR `status` 单读（operator 写）；API 不直接写 K8s | — |
| 列表查询 | 所有 list 端点支持 `?labelSelector=` （K8s grammar：`=`/`==`/`!=`/`in (...)`/`notin (...)`/`key`/`!key`，逗号分隔 AND） | [database.md §1.6](../database.md#16-扩展元数据-labels--annotations) |
| 身份头 | 调用方注入 `X-Axisml-User`，本服务仅做审计与 ownership 归属 | [auth.md §7](../auth.md#7-下游身份透传) |
| 错误格式 | HTTP 标准状态码 + RFC 7807 problem+json | — |
| 写后语义 | mutation 在 PG 提交后即返回；CR 同步异步进行，调用方通过 GET 观察 `status` | — |

## 7. 依赖

| 依赖 | 用途 | 引用 |
| --- | --- | --- |
| PostgreSQL | 业务元数据权威；与 cluster-manager / artifacts 共享 database `axisml`，表前缀隔离 | [database.md §3](../database.md#3-compute) / [infra.md](../infra.md) |
| Kubernetes API | `Tenant` / `MLJob` / `MLService` CR 下发 + status watch + Pod log / Event 透传 + leader Lease | — |
| tenant-operator | 下游 CR 消费者，把 Tenant CR 落地为 K8s Namespace / Secret / ElasticQuota 等 | [tenant-operator.md](tenant-operator.md) |
| compute-operator | 下游 CR 消费者，把 MLJob / MLService CR 落地为底层资源 | [compute-operator.md](compute-operator.md) |
| Platform | 上游唯一调用方；注入 `X-Axisml-User`，请求体里已携带展开后的 nodeSelector / requests / limits（Pool/Unit 展开在 Platform 完成） | [auth.md](../auth.md) / [platform.md §4.2](platform.md#42-计算任务编排) |
| artifacts | image / model / dataset 引用懒查询；**由 operator handler 侧调用 resolve API，Compute 自身不调用** | [artifact-hub.md](artifact-hub.md) |

Compute 不感知 ResourcePool / ResourceUnit / ElasticQuota 内部结构——这些由 Platform 与 cluster-manager 配合在 compute 之外处理。

## 8. 运行时形态

| 维度 | 值 |
| --- | --- |
| 进程 | 单二进制 `axisml-compute-service`；子命令 `serve` / `migrate` / `bootstrap` |
| 副本 | 默认 `replicas=1`（API 无状态可水平扩，reconciler / Informer 单 leader） |
| Leader Election | K8s `Lease`（`axisml-compute-service.axisml.io`）；单副本退化为单成员瞬时 lease；`/metrics` 暴露 `axisml_compute_is_leader` gauge |
| 暴露 | ClusterIP `:8081`，无外部 `HTTPRoute`；探针 `/healthz`（liveness）/ `/readyz`（校验 PG） |
| RBAC scope | `tenants.axisml.io` / `mljobs.axisml.io` / `mlservices.axisml.io` 全权 + `pods` / `pods/log` / `events` RO + 自身 ns 的 `leases`；**不含** `elasticquotas` / `namespaces` / `secrets`（这些由 tenant-operator 落地） |
| Helm values / 镜像 | 详见 [deployment.md §6.1](../deployment.md#61-cluster-manager--compute--artifacts--platform-backend) |

## 9. 后续工作

- Admission webhook：硬阻断非 Compute 的 `Tenant` / `MLJob` / `MLService` spec 写请求。
- 多副本 HA：完整 leader election 路径与多副本压测。
- Job spec 部分可变：`display_name` / `description` / `runPolicy.activeDeadlineSeconds` 等元数据更新。
- Job 模板与 DAG 工作流编排。
- Service 多 role 独立扩缩。
- Service 基于 `request_rate` 的 autoscaling、`spec.route` 热更新（轮换 API key / 调限流不重建）。
- Tenant 批量端点；Tenant retention GC 守护进程。
- 数据卷管理模块；Custom backend 透传；多集群联邦。
- 独立 `audit_events` 表；按使用时长 × 单元成本的计费导出。
- mTLS / Compute 主动鉴权（当前完全信任 Platform 注入的 `X-Axisml-User`）。

## 10. 相关引用

- [overview.md](../overview.md) — Compute 在控制平面里的位置
- [auth.md](../auth.md) — `X-Axisml-User` 身份头协议与鉴权边界
- [database.md](../database.md) — Compute 表结构（§3）
- [deployment.md](../deployment.md) — Helm 模板与发布编排
- [monitoring.md](../monitoring.md) — Compute Prometheus 指标列表
- [infra.md](../infra.md) — Koordinator / koord-scheduler / ElasticQuota 依赖契约
- [apis/compute.yaml](../apis/compute.yaml) — REST API 契约源
- [tenant-operator.md](tenant-operator.md) — Tenant CR 字段契约与落地行为
- [compute-operator.md](compute-operator.md) — `MLJob` / `MLService` CR 字段契约与 handler 实现
- [artifact-hub.md](artifact-hub.md) — Job / Service 提交时引用懒查询（image / model / dataset）
- [cluster-manager.md](cluster-manager.md) — ResourcePool / ResourceUnit 元数据；Platform 在编排前展开为原语
- [platform.md](platform.md) — 上游调用方；工作区在 `services` 表上的 `kind='workspace'` 复用
