# AxisML Compute 概要设计

## 1. 定位与边界

计算服务层；维护 Job / Service / ResourcePool / ResourceUnit 的业务元数据，把 `MLJob` / `MLService` CR 当作 PG 行的派生产物下发到 K8s。

| 做 | 不做 |
| --- | --- |
| Job / Service CRUD、cancel / scale、软删 | 直接创建 Pod / Deployment / PodGroup 等底层资源 (→ [compute-operator.md](compute-operator.md)) |
| ResourcePool / ResourceUnit CRUD + 注入合并 | 修改 Node label / taint（管理员手工维护） |
| `MLJob` / `MLService` spec 下发 + status 回流 | 租户、ElasticQuota、Namespace 管理 (→ [cluster-manager.md](cluster-manager.md) / [tenant-operator.md](tenant-operator.md)) |
| 日志 / 副本 / 事件端点透传 kube-apiserver | 用户认证与角色鉴权 (→ [auth.md](../auth.md)) |
| 工作区列表（`kind='workspace'` 过滤） | 工作区业务语义本身 (→ [platform.md](platform.md)) |

`namespace` 字段作为裸字符串分区键，`spec.scheduling.quota` 作为不透明 ElasticQuota 名透传；Compute 不解析也不校验存在性。

## 2. 架构

### 2.1 上下文

```
        ┌──────────────┐  REST + X-Axisml-User   ┌──────────────────┐
        │  Platform    │ ───────────────────────▶│  Compute (Go)    │
        └──────────────┘                          └──────┬───────────┘
                                                         │ PG 读写 / CR patch + watch
                                          ┌──────────────┼──────────────┐
                                          ▼                              ▼
                                ┌─────────────────┐           ┌──────────────────────┐
                                │  PostgreSQL     │           │   K8s API            │
                                │  jobs/services/ │           │   MLJob / MLService  │
                                │  resource_*     │           └──────────┬───────────┘
                                └─────────────────┘                      │ watch
                                                                         ▼
                                                              ┌────────────────────┐
                                                              │  compute-operator  │
                                                              └────────────────────┘
```

### 2.2 内部结构

```
┌──────────────────────────── Compute (Go) ─────────────────────────────┐
│  HTTP API (Gin) ──写──▶  PG (generation + status='Creating')          │
│        ▲                                │                              │
│        │ 读                              ▼                              │
│        │                       Reconciler goroutines (leader-only)    │
│        │            ┌────── job ────────┬────── service ───────┐       │
│        │            │ resource-pool/unit (纯 PG，无 reconciler) │       │
│        │            └───────────────────┴──────────────────────┘       │
│        │                                │                              │
│        │                                ▼ Patch / Delete                │
│        │                       MLJob / MLService CR                    │
│        │                                │ status                        │
│        └──── PG status 列 ◀──── Informer (leader-only, shared cache)   │
└────────────────────────────────────────────────────────────────────────┘
```

## 3. 核心模型

| 实体 | 含义 | 标识键 | 状态机 | 对应 CR |
| --- | --- | --- | --- | --- |
| Job | 一次性训练 / 离线任务 | `id` (uuid) / `(namespace, name)` | `Creating \| Pending \| Running \| Succeeded \| Failed \| Canceling \| Cancelled \| Deleting \| Deleted` | `MLJob` |
| Service | 常驻在线服务 / 工作区 | `id` (uuid) / `(namespace, name)` | `Creating \| Pending \| Ready \| Degraded \| Failed \| Deleting \| Deleted` | `MLService` |
| ResourcePool | 节点切分维度（GPU 池 / CPU 池 / 训练池） | `name` (全局唯一) | 无（纯配置） | 无 |
| ResourceUnit | 池内资源规格模板（含 requests/limits/nodeSelector） | `(pool, name)` | 无（纯配置） | 无 |

字段级 schema 见 [database.md §3](../database.md#3-compute)；CR spec 字段见 [compute-operator.md](compute-operator.md)。

**通用 PG 约定**：所有表带 `id uuid` / `created_at` / `updated_at` / `deleted_at`；所有 UNIQUE 实现为 partial unique index `WHERE deleted_at IS NULL`（软删行不占用唯一键，同名可再次创建）；`name` 统一 DNS-1123 校验，长度 3–40。CR-backed 对象额外打 `axisml.io/{job,service}-id=<uuid>` label 作为稳定锚点（`metadata.name` 因软删可重用，UUID 永久唯一）；`services` 还会同步打 `axisml.io/service-kind=<service|workspace>` label，便于 `kubectl` selector 区分工作区与普通服务（compute / operator 不按 kind 改变行为）。

**扩展元数据**：`jobs` / `services` 表均带 `labels jsonb` + `annotations jsonb` 双字段，对齐 K8s 风格语义，统一约定见 [database.md §1.6](../database.md#16-扩展元数据-labels--annotations)。两者均 **PG-only、不下发 CR、不 `+generation`**——Platform 走 compute REST 写入，不直接 patch CR。

## 4. 核心功能

### 4.1 Job

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
| 提交 | insert `Creating` 行 + spec 快照（`generation=1`） | reconciler `Create()` MLJob | 创建后 spec 不可变；无后续 mutation，仅靠 `status='Creating'` 谓词推进 |
| cancel | `status='Canceling'` + `message='user cancelled'` | reconciler `patch spec.runPolicy.suspend=true` | `Creating` 状态拒绝；要求改用 DELETE |
| 更新 PG 元数据（`display_name` / `description` / `labels` / `annotations`） | update 行 | **不影响 CR** | Job spec 不可变，但 PG 扩展位任意阶段可改 |
| 软删 | `status='Deleting'` + `deleted_at=now()` | reconciler `Delete()` CR；Informer DELETE → `Deleted` | 任一非 `Deleting`/`Deleted` 状态适用 |
| 日志 / 副本 / 事件 | — | 透传 kube-apiserver Pod Log / 按 label list Pod / 聚合 Event | 详见 [apis/compute.yaml](../apis/compute.yaml) `Jobs` tag |

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

### 4.2 Service

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
| 创建 | insert `Creating` 行 + spec 快照（`generation=1`） | reconciler `Create()` MLService（含 `axisml.io/service-{id,kind}` label） |
| scale | 更新 `services.replicas` + `spec.roles[0].replicas` + `generation += 1` | reconciler `generation>observed_generation` 触发 patch `spec/roles/0/replicas` |
| 更新 PG 元数据（`display_name` / `description` / `labels` / `annotations`） | update 行 | **不影响 CR**；不 `+generation` |
| 软删 | `status='Deleting'` + `deleted_at=now()` | reconciler `Delete()` CR；Informer DELETE → `Deleted` |

Service 无 cancel 语义；除 `roles[0].replicas` 外其他 spec 字段不可变。`kind` 创建后不可变。

**与 MLService CR 契约**：

| 条件 | PG `status` |
| --- | --- |
| `ready_replicas == desired_replicas && desired_replicas > 0` | `Ready` |
| `0 < ready_replicas < desired_replicas` | `Degraded` |
| `ready_replicas == 0 && desired_replicas > 0` 且 CR `phase=Pending` | `Pending` |
| `ready_replicas == 0 && desired_replicas > 0` 且 CR `phase=Failed` | `Failed`（可自愈） |
| `desired_replicas == 0` | `Pending` |

**id-based 寻址 + kind 过滤**：除 `(namespace, name)` 路径外提供 `GET /api/v1/services/{id}` 与 `GET /api/v1/services?ids=...` 或 `?namespace=&kind=workspace`，供 [Platform 工作区](platform.md) 在同一张表上区分 `kind='service'` 与 `kind='workspace'`；`kind` 创建后不可变，Compute 不按 `kind` 改变行为，仅作分类与过滤。写操作（`/scale`、`DELETE`）保留 namespace-scoped 形态。

### 4.3 ResourcePool

纯 PG 元数据，无 CR；管理员负责给目标节点打标签 / 污染，Compute 不修改 Node 对象。Pool 全局可见，Compute 不做按租户隔离的可见性校验。

| 字段 | 用途 |
| --- | --- |
| `name` | DNS-1123，全局唯一 |
| `node_selector` | 注入到 `spec.scheduling.nodeSelector`（Pool 优先） |
| `tolerations` | 直接作为 `spec.scheduling.tolerations` |

**默认池**：Helm post-install Job 初始化 `default` 池，`node_selector` 为空表示整集群可用。

### 4.4 ResourceUnit

池内资源规格模板，纯 PG 元数据，无 CR。

| 字段 | 用途 |
| --- | --- |
| `(pool, name)` | partial UNIQUE |
| `requests` / `limits` | 注入到 CR `spec.roles[*].template.resources` |
| `node_selector` | 与 Pool 合并后注入 `spec.scheduling.nodeSelector` |

**命名约定**：`<accelerator>[-<count>x]-<tier>[-<variant>]`，例如 `cpu-small` / `a100-1x-large` / `h100-8x-xlarge-ib` / `tpu-v4-4x-large`。`<tier>` ∈ `small | medium | large | xlarge`。

**注入合并规则**（Pool 优先）：

| 来源 | 合并行为 |
| --- | --- |
| `pool.node_selector` | key 全部保留 |
| `unit.node_selector` | 仅贡献 Pool 未声明的 key |
| `pool.tolerations` | 直接作为 `spec.scheduling.tolerations` |
| `unit.requests` / `limits` | 写入 `spec.roles[*].template.resources` |

合并由 `internal/resourceunit/inject.go` 完成。

## 5. 关键机制

### 5.1 写路径：内嵌 Outbox + 4 谓词扫描

无独立 outbox 表——借用业务表自身的 `status` / `deleted_at` / `generation` 列。API 同步路径只写 PG（业务校验 → 事务内写状态 + spec 快照 + `generation += 1` → commit → 返回业务 ID）；reconciler goroutine 在 leader 副本按以下谓词扫描分派：

| 谓词 | 动作 | 适用模块 |
| --- | --- | --- |
| `status='Creating' AND deleted_at IS NULL` | `Create()` CR（带 `axisml.io/<resource>-id` label；409 视为成功） | Job / Service |
| `status='Canceling'` | `patch MLJob.spec.runPolicy.suspend=true` | Job |
| `status='Deleting'` | `Delete()` CR；Informer DELETE 推进 `Deleted` | Job / Service |
| `generation <> observed_generation AND deleted_at IS NULL` | `Patch()` CR；成功后 `observed_generation = generation` | Service |

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
| Service | 是 | `spec.roles[0].replicas` |
| Job | 否 | 不可变；cancel 走 `Canceling` 谓词单独 patch suspend |
| ResourcePool / ResourceUnit | 否 | 无 CR，无下行同步 |

### 5.3 状态回流（Informer）

两条独立 Informer，通过 `k8sclient` 的 `SharedInformerFactory` 共享 cache，仅 leader 副本运行。

| Informer | 监听对象 | 主要回写字段 |
| --- | --- | --- |
| MLJob Informer | `MLJob` CR | `jobs.status` / `started_at` / `finished_at` / `message`；按 4.1 phase 映射表推进 |
| MLService Informer | `MLService` CR | `services.status` / `ready_replicas` / `endpoint` / `message`；按 4.2 条件表推进 |

启动时 `List` 做差异 upsert 与孤儿对账；`Watch` 事件入各自 work queue；单 worker 串行 reconcile；以 `resourceVersion` / `generation` 作乐观并发字段。

### 5.4 孤儿与补偿

Compute **不反向重建 CR**。Informer 观察到 CR DELETE 事件后按 PG 当前 `status` 分流：

| PG 当前状态 | 处理 |
| --- | --- |
| `Canceling` | 幂等忽略；`Cancelled` 只由 Suspended condition 推进 |
| `Deleting` | 正常级联清理，推进 `Deleted` |
| Job 在 `Pending`/`Running`（外部误删） | 推 `Cancelled` + `finished_at` + `message='external delete'`，不补偿重建 |
| Service 在 `Pending`/`Ready`/`Degraded`/`Failed`（外部误删） | 写 `Deleting` + `deleted_at=now()` + `message='external delete'`，下一轮谓词幂等确认后推 `Deleted` |
| 已终态 | 忽略 |

**正向孤儿**（PG `Creating` 但无 CR，且 `deleted_at IS NULL`）属 Outbox 正常窗口，reconciler 下一轮幂等重试 `Create()`。**反向孤儿**（CR 存在但 PG 无行或已 `Deleted`）：默认删除 CR 并记录审计。

## 6. 接口契约

| 类别 | 内容 | 引用 |
| --- | --- | --- |
| 对外 REST | `/api/v1/namespaces/{ns}/jobs[...]`、`/api/v1/namespaces/{ns}/services[...]`、`/api/v1/services[/{id}]`、`/api/v1/resource-pools[...]`、`/api/v1/resource-pools/{pool}/resource-units[...]` | [apis/compute.yaml](../apis/compute.yaml) `Jobs` / `Services` / `ResourcePools` / `ResourceUnits` tag |
| 下发 CR | `MLJob` / `MLService`（`axisml.io/v1alpha1`，namespaced），Compute 是唯一 `spec` 写者 | [compute-operator.md](compute-operator.md) |
| 回流字段 | `jobs.status` / `started_at` / `finished_at`；`services.status` / `ready_replicas` / `endpoint` | — |
| 不变量 | CR `metadata` / `spec` 单写（Compute 写）；CR `status` 单读（operator 写）；API 不直接写 K8s | — |
| 身份头 | 调用方注入 `X-Axisml-User`，本服务仅做审计与 ownership 归属 | [auth.md §7](../auth.md#7-下游身份透传) |
| 错误格式 | HTTP 标准状态码 + RFC 7807 problem+json | — |
| 写后语义 | mutation 在 PG 提交后即返回；CR 同步异步进行，调用方通过 GET 观察 `status` | — |

## 7. 依赖

| 依赖 | 用途 | 引用 |
| --- | --- | --- |
| PostgreSQL | 业务元数据权威；与 cluster-manager / artifacts 共享 database `axisml`，表前缀隔离 | [database.md §3](../database.md#3-compute) / [infra.md](../infra.md) |
| Kubernetes API | `MLJob` / `MLService` CR 下发 + status watch + Pod log / 副本 / Event 透传 + leader Lease | — |
| compute-operator | 下游 CR 消费者，落地为底层资源 | [compute-operator.md](compute-operator.md) |
| Platform | 上游唯一调用方；注入 `X-Axisml-User`，请求体透传 `namespace` 与 `spec.scheduling.quota` 字符串 | [auth.md](../auth.md) |
| artifacts | image / model / dataset 引用懒查询；**由 operator handler 侧调用 resolve API，Compute 自身不调用** | [artifacts.md](artifacts.md) |

Compute 不直接调用 cluster-manager，也不感知 Tenant / ElasticQuota。

## 8. 运行时形态

| 维度 | 值 |
| --- | --- |
| 进程 | 单二进制 `axisml-compute`；子命令 `serve` / `migrate` / `bootstrap` |
| 副本 | 默认 `replicas=1`（API 无状态可水平扩，reconciler / Informer 单 leader） |
| Leader Election | K8s `Lease`（`axisml-compute.axisml.io`）；单副本退化为单成员瞬时 lease；`/metrics` 暴露 `axisml_compute_is_leader` gauge |
| 暴露 | ClusterIP `:8081`，无外部 `HTTPRoute`；探针 `/healthz`（liveness）/ `/readyz`（校验 PG） |
| RBAC scope | `mljobs.axisml.io` / `mlservices.axisml.io` 全权 + `pods` / `pods/log` / `events` RO + 自身 ns 的 `leases`；**不含** `tenants` / `elasticquotas` / `namespaces` / `secrets` |
| Helm values / 镜像 | 详见 [deployment.md §6.1](../deployment.md#61-cluster-manager--compute--artifacts--platform-backend) |

## 9. 后续工作

- Admission webhook：硬阻断非 Compute 的 `MLJob` / `MLService` spec 写请求。
- 多副本 HA：完整 leader election 路径与多副本压测。
- Job spec 部分可变：`display_name` / `description` / `runPolicy.activeDeadlineSeconds` 等元数据更新。
- Job 模板与 DAG 工作流编排。
- Service 多 role 独立扩缩（当前 `services.replicas` 单 role 约定）。
- Service 基于 `request_rate` 的 autoscaling、`spec.route` 热更新（轮换 API key / 调限流不重建）。
- ResourcePool 池间调度策略（配额不足时是否允许跨池借用，默认禁止）+ 池容量预估。
- ResourceUnit 混合资源单元（CPU + MIG 分片）与价格元数据用于成本核算。
- 数据卷管理模块；Custom backend 透传；多集群联邦。
- 独立 `audit_events` 表；按使用时长 × 单元成本的计费导出。
- mTLS / Compute 主动鉴权（当前完全信任 Platform 注入的 `X-Axisml-User`）。

## 10. 相关引用

- [overview.md](../overview.md) — Compute 在控制平面里的位置
- [auth.md](../auth.md) — `X-Axisml-User` 身份头协议与鉴权边界
- [database.md](../database.md) — Compute 表结构（§3）
- [deployment.md](../deployment.md) — Helm 模板与发布编排
- [monitoring.md](../monitoring.md) — Compute Prometheus 指标列表（§4.1）
- [infra.md](../infra.md) — Koordinator / koord-scheduler / ElasticQuota 依赖契约
- [apis/compute.yaml](../apis/compute.yaml) — REST API 契约源
- [compute-operator.md](compute-operator.md) — `MLJob` / `MLService` CR 字段契约与 handler 实现
- [artifacts.md](artifacts.md) — Job / Service 提交时引用懒查询（image / model / dataset）
- [cluster-manager.md](cluster-manager.md) — 同形态 PG + 下游 CR 业务服务样板
- [platform.md](platform.md) — 上游调用方；工作区在 `services` 表上的 `kind='workspace'` 复用
