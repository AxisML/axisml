# AxisML Compute Service 设计

## 1. 定位与边界

ML 工作负载服务：以 PostgreSQL 为权威，承载 Job / Service / Workspace / TrafficPolicy 的元数据，把 `MLRun` / `MLService` / `MLTrafficPolicy` CR 当作 PG 行的派生产物下发到 K8s。仅接受上游业务编排层的内部调用并信任 `X-Axisml-User` 透传。

| 做 | 不做 |
| --- | --- |
| Job / Service CRUD、cancel / scale、软删；数据卷由 Platform 经 cluster-manager 管理，compute 仅在 Pod 模板里以 PVC 引用挂载 | 租户 / 配额（Tenant CR + 折算）(→ [cluster-manager.md](cluster-manager.md))；持久卷 PVC 的创建 / 回收与对应 RBAC（→ Platform 经 [cluster-manager.md §4.5](cluster-manager.md#45-volume) Volume REST 管理数据卷；compute 不调 cluster-manager、无 PVC 写权限） |
| 流量策略 CRUD、split / promote / rollback、指标代理；成员校验权威 | Namespace / ElasticQuota / initResources 落地 (→ [tenant-operator.md](tenant-operator.md)) |
| 三类 CR spec 下发 + status 回流 | 直接创建 Pod / Deployment / PodGroup（→ [compute-operator.md](compute-operator.md)） |
| Pod 列表 / 日志 / 事件透传 kube-apiserver | 加权 HTTPRoute / 灰度的网关派生 (→ [compute-operator.md](compute-operator.md)) |
| 在线服务 / 工作负载运行指标代理（按 `spec.backend` 选 PromQL 查 Prometheus） | 用户认证与角色鉴权 (→ [auth.md](../../../axisml-platform/docs/system_design/auth.md)) |
| 按 tenant scope 列出 MLRun / MLService，支持 `labelSelector` 过滤（供上游 pool/unit 删除前置阻断） | ResourcePool / ResourceUnit 词汇 (→ [cluster-manager.md](cluster-manager.md)) |

**namespace 字段语义**：`mlruns` / `mlservices` 表的 `namespace text` 是兼容字段名，实际表示 tenant scope（租户 `identifier`），不是 K8s Namespace。代码与新接口描述优先使用 `tenantScope`；物理落地点使用 `kubernetesNamespace` 明确表达（见 [high_level_design §2.2](../../../docs/high_level_design.md#22-关键不变量)）。

**Pool/Unit 展开归属**：上游仅传 `(poolName, unitName)` 名字对；compute-service 通过 K8s Informer 直读 `ResourcePool` CR cache 完成展开（合并 `nodeSelector` / `tolerations` / `requests` / `limits`），snapshot 到 `spec` jsonb。snapshot 一经写入即与 pool/unit CR 解耦（§5.4）。

## 2. 架构

```
   上游调用方 ──REST + X-Axisml-User──▶ Compute (Go)
                                          │ PG 读写 / CR patch + watch
                      ┌───────────────────┼─────────────────────┐
                      ▼                   ▼                     ▼
              PostgreSQL          K8s API (MLRun /        compute-operator
        mlruns/mlservices/        MLService /
        traffic_policies          MLTrafficPolicy)
```

```
┌──────────── Compute (Go) ────────────┐
│ HTTP API (Gin) ──写──▶ PG (generation + phase='Creating') │
│   ▲ 读                      │                              │
│   │            Reconciler goroutines (leader-only)         │
│   │            ┌ job ┬ service ┬ traffic ┐                 │
│   │                         │ Create/Patch/Delete CR       │
│   └── PG status ◀── Informer (leader-only, shared cache)   │
└────────────────────────────────────────────────────────────┘
```

三条 Reconciler 共享 leader Lease 与 PG 连接池；三条 Informer 共享 `SharedInformerFactory`，外加一条只读 ResourcePool Informer（用于创建时的 pool/unit 展开，§5.4）。

## 3. 核心模型

| 实体 | 含义 | 标识键 | 状态机 | CR |
| --- | --- | --- | --- | --- |
| Job | 一次性训练 / 离线任务 | `id` / `(namespace, name)` | `Creating｜Pending｜Running｜Succeeded｜Failed｜Canceling｜Cancelled｜Deleting｜Deleted` | `MLRun` |
| Service | 常驻服务 / 工作区 / TensorBoard | `id` / `(namespace, name)` | `Creating｜Pending｜Ready｜Degraded｜Failed｜Deleting｜Deleted` | `MLService` |
| TrafficPolicy | 流量策略：稳定入口按权重分发到多服务 | `id` / `(namespace, name)` | 同 Service | `MLTrafficPolicy` |

字段级 schema 见 [database.md §2](database.md#2-compute-service)；CR spec 字段见 [compute-operator.md](compute-operator.md)。

**通用 PG 约定**：所有表带 `id uuid` / `created_at` / `updated_at` / `deleted_at`；UNIQUE 为 partial index `WHERE deleted_at IS NULL`（软删行不占唯一键）；`name` DNS-1123、长度 3–40。CR-backed 对象打 `compute.axisml.io/{run,service,traffic-policy}-id=<uuid>` label 作稳定锚点（`metadata.name` 可重用，UUID 永久唯一）；`mlservices` 还同步打 `compute.axisml.io/service-kind=<service|workspace|tensorboard>` label 便于 selector 区分（compute / operator 不按 kind 改变行为）。

**扩展元数据 + 分组**：三表均带 `labels` / `annotations` jsonb，对齐 K8s 语义（[database.md §1.6](database.md#16-扩展元数据-labels--annotations)）；list 端点支持 `?labelSelector=`。两类扩展位 **PG-only、不下发 CR、不 `+generation`**。

## 4. 核心功能

### 4.1 Job

```
Creating ─(Informer ADD)─▶ Pending ─▶ Running ─▶ Succeeded / Failed
                              │ cancel    │ cancel
                              └───────────┴─▶ Canceling ─(Suspended condition)─▶ Cancelled
任一非 Canceling/Deleting/Deleted ─[DELETE]─▶ Deleting ─(CR 清理 + deleted_at)─▶ Deleted
```

| 操作 | PG 写 | CR 影响 |
| --- | --- | --- |
| 提交 | insert `Creating` + spec 快照（已含展开后的 nodeSelector / tolerations / resources） | reconciler `Create()` MLRun；创建后 spec 不可变 |
| cancel | `phase='Canceling'` + message | patch `spec.runPolicy.suspend=true`；`Creating` 拒绝（改用 DELETE） |
| 更新 PG 元数据 | update 行 | 不影响 CR（spec 不可变，扩展位任意阶段可改） |
| 软删 | `phase='Deleting'` + `deleted_at` | `Delete()` CR；Informer DELETE → `Deleted` |
| Pod 列表 / 日志 / 事件 | — | 按 `compute.axisml.io/run-id` label list Pod；按 Pod 名透传 Log；按 `involvedObject` 过滤 Event |

`Succeeded` / `Failed` / `Cancelled` 为运行终态；`Deleted` 为软删终态（`Cancelled` 行保留，可再 DELETE）。

**Run 对象存储产出**：实验等 Run 把 TensorBoard event log / checkpoint 写到对象存储——路径（`experiments/<def>/runs/<run>/...`）与凭证由 operator 渲染 Pod 时注入（[compute-operator.md §5.2](compute-operator.md#52-pod-注入约定)）；Run 软删时按对象存储约定前缀一并 GC（与工作区卷 GC 同档）。

**MLRun CR 契约**：`status.phase` 直映 PG `status`（`Pending`/`Running`/`Succeeded`/`Failed`）；`Pending`/`Running` + `conditions[Suspended,True,CancelRequested]` → PG `Cancelled`（随后入队 `Delete()` 回收）。Cancel 与自然完成竞速时 operator 优先保留终态。

### 4.2 Service

```
Creating ─(Informer ADD)─▶ Pending ─(ready=desired>0)─▶ Ready ⇄ Degraded ─▶ Failed
                                                          ▲ ──── 自愈 ──── │
任一非 Deleting/Deleted ─[DELETE]─▶ Deleting ─(CR 清理 + deleted_at)─▶ Deleted
```

`Ready` / `Degraded` / `Failed` 均**非终态**（operator 自愈后 Informer 回流自然恢复）；只有 `Deleted` 为终态。

| 操作 | PG 写 | CR 影响 |
| --- | --- | --- |
| 创建 | insert `Creating` + spec 快照 | `Create()` MLService（含 `compute.axisml.io/service-{id,kind}` label）；需持久存储的工作负载由 Platform 以既有数据卷的 PVC 引用写在 `roles[0].template.volumes[]`，compute 原样下发 |
| scale | `spec.roles[0].replicas` + `generation += 1` | `generation>observed_generation` 触发 patch replicas |
| 更新 PG 元数据 | update 行 | 不影响 CR、不 `+generation` |
| 软删 | 在线服务先检查是否被活跃 TrafficPolicy 引用，命中则 `409 service-in-use`；通过后写 `phase='Deleting'` + `deleted_at` | `Delete()` CR；数据卷为独立受管对象，其生命周期由 Platform 经 cluster-manager 处理，compute 不删卷 |
| Pod 列表 / 日志 / 事件 / 指标 | — | 同 Job；指标按 `spec.backend` 选 PromQL 查 Prometheus，返 `MetricSeries`（QPS / 延迟 / 错误率 / CPU·内存·GPU） |

Service 无 cancel；除 `roles[0].replicas` 与 `kind` 外其他 spec 不可变。

**持久卷（数据卷）**：持久卷不是 compute 的职责，任何工作负载（工作区 / 任务 / 实验等）一视同仁。卷由 Platform 经 cluster-manager Volume REST 管理（[cluster-manager.md §4.5](cluster-manager.md#45-volume)），并由 Platform 以 PVC 引用（`persistentVolumeClaim.claimName`）写进 `roles[0].template.{volumes,volumeMounts}`。compute 只把该 Pod 模板原样下发到 CR，不创建、不删除、不感知卷，也不持有 PVC 写权限。

**MLService CR 契约**：

| 条件 | PG `status` |
| --- | --- |
| `ready == desired > 0` | `Ready` |
| `0 < ready < desired` | `Degraded` |
| `ready == 0 && desired > 0` 且 CR `Pending` | `Pending` |
| `ready == 0 && desired > 0` 且 CR `Failed` | `Failed`（可自愈） |
| `desired == 0` | `Pending` |

**kind 过滤**：`GET .../mlservices?kind=workspace`（或 `service` / `tensorboard`）供上游在同一张表上区分在线服务 / 交互式开发 / 实验指标查看实例。`kind='tensorboard'` 复用 `(native, deployment)`、**无持久卷**，logdir 与读凭证由 operator 渲染 Pod 时注入，可按空闲 TTL 回收（由上游编排）。

### 4.3 流量策略（MLTrafficPolicy）

把一个稳定对外入口的流量按权重分发到同租户下多个**在线服务**（`kind='service'`）后端，支撑加权 / 灰度 / 蓝绿。compute-service 是 CR 唯一 spec 写者，持权重 / 灰度态权威并代理指标；网关派生（`(native,httproute)` → Envoy `HTTPRoute` 加权 `backendRefs`；`(kserve,inference)` → `InferenceService` canary）由 [compute-operator](compute-operator.md#43-mltrafficpolicy-controller) 完成。

**模式（`mode`，创建后不可变）**：

| mode | 语义 | backends | 可变维度 | 派生权重 |
| --- | --- | --- | --- | --- |
| `weighted` | 加权切分 | N≥2，各带权重 | `backends[*].weight`（Σ=100） | 显式权重 |
| `canary` | 多版本灰度 | 稳定基线 + 灰度 | 灰度百分比 `p∈[0,100]` | 灰度=p、稳定=100−p |
| `bluegreen` | 蓝绿切换 | 恰好 2 成员 | 全量切换 | 一侧 100、一侧 0 |

`bluegreen` 复用 `split` 写路径（全量切换约束）；`canary` 额外有 `promote` / `rollback`。`role` 约定：canary 下 backends 恒为 `role=stable`（当前基线）+ `role=canary`（灰度），**不另设指针字段**；`promote` 互换二者 `role`，`rollback` 仅回收 canary 权重；`weighted` 留空，`bluegreen` 用 `blue`/`green`。

状态机同 Service（`Ready` / `Degraded` / `Failed` 非终态，仅 `Deleted` 终态）。split / promote / rollback / bluegreen 都是 `backends[*].weight` 的 spec mutation（`+generation`），不改 phase。

| 操作 | PG 写 | CR 影响 |
| --- | --- | --- |
| 创建 | 成员预检通过后 insert `Creating` + spec 快照（`mode` / `endpoint` / `backends` / 派生 `backend` 元组） | `Create()` MLTrafficPolicy（含 `compute.axisml.io/traffic-policy-id` label） |
| 调整流量（split） | weighted 写权重 / canary 写百分比并派生稳定权重 / bluegreen 翻转 100·0 — `+generation` | patch `spec.backends[*].weight` |
| 提升 / 回滚（canary） | `promote` 互换 stable/canary 的 `role` 并置 100·0；`rollback` 置 canary 0 — `+generation` | patch |
| 更新 PG 元数据 | update 行 | 不影响 CR |
| 软删 | `phase='Deleting'` + `deleted_at` | `Delete()` CR；派生 `HTTPRoute` / `SecurityPolicy` 经 ownerReference 级联；**成员 MLService 不随策略删除** |
| 指标查询 | — | 按成员后端分组查 PromQL（含灰度健康对比），返 `MetricSeries`；不入 PG |

`endpoint`（path / hostname / auth）与 `mode` 创建后不可变——改模式 / 入口 = 先删后建。

**成员引用完整性（本服务为权威闸门）**：创建策略时，每个成员经内部 `GetService` 预检 `kind=='service'`（拒绝 workspace）且 `Ready`；成员须同租户；一个 MLService 同时只能被一个活跃策略引用（应用层事务内维护）。删除在线服务时反查活跃 TrafficPolicy，仍被引用则拒绝删除。成员后端类型同构（全 `native` 或全 `kserve`，`kserve` 要求 `mode='canary'` 且恰好 2 成员）；校验后把派生的 `spec.backend` 元组写入 CR 供 operator 路由；`endpoint.path==""` 时自动拼 `/services/<tenant>/<name>/`。上游预检仅用于快速失败 UX。

**MLTrafficPolicy CR 契约**：route programmed 且全成员 ready → `Ready`；部分 ready / degraded → `Degraded`；未 programmed 或全未就绪且 CR `Pending` → `Pending`；CR `Failed` → `Failed`（可自愈）。`status` jsonb 持 `{phase, message, endpoint, backends[].{serviceName, weight, ready}}`，由 Informer 整块回流（§5.3）。权重 / 灰度态 / 成员 phase 始终以本服务为权威源回源。

## 5. 关键机制

### 5.1 写路径：内嵌 Outbox + 谓词扫描

无独立 outbox 表——借业务表自身的 `status` / `deleted_at` / `generation` 列。API 同步路径只写 PG（校验 → 事务内写状态 + spec 快照 + `generation += 1` → commit → 返业务 ID）；reconciler 在 leader 副本按谓词扫描分派：

| 谓词 | 动作 | 适用 |
| --- | --- | --- |
| `phase='Creating' AND deleted_at IS NULL` | `Create()` CR（带 id label；409 视为成功） | Job / Service / TrafficPolicy |
| `phase='Canceling'` | patch `MLRun.spec.runPolicy.suspend=true` | Job |
| `phase='Deleting'` | `Delete()` CR；Informer DELETE 推进 `Deleted` | 三者 |
| `generation <> observed_generation AND deleted_at IS NULL` | `Patch()` CR；成功后 `observed_generation = generation` | Service / TrafficPolicy |

失败指数退避重试，错误写入 `message`。PG 行不再满足谓词后 reconciler 不再下发——自然结束 / 自愈 / 外部误删由 Informer 回流推进。

### 5.2 generation spec 同步

API mutation 在事务内 spec 写入 + `generation += 1`；partial index `WHERE generation <> observed_generation AND deleted_at IS NULL` 触发 reconciler patch CR → `observed_generation = generation`。语义对齐 K8s（[database.md §1.4](database.md#14-generation--observed_generation)）。允许变更字段：Service `spec.roles[0].replicas`；TrafficPolicy `spec.backends[*].{weight,role}`；Job 不可变（cancel 走 `Canceling` 谓词单独 patch suspend）。

### 5.3 状态回流（Informer）

三条独立 Informer 共享 cache，仅 leader 副本运行。MLRun / MLService / MLTrafficPolicy Informer 各整块写对应表的 `status` jsonb，按 §4 条件表推进 phase。启动时 `List` 做差异 upsert 与孤儿对账；`Watch` 事件入 work queue，单 worker 串行 reconcile，以 `resourceVersion` / `generation` 作乐观并发字段。

### 5.4 ResourcePool 展开

创建请求时 compute-service **自己**完成 pool/unit 到 K8s Pod 原语的展开：

```
POST .../mlruns  body: { name, scheduling:{poolName, unitName, quota}, ... }
   ▼ ResourcePool Informer cache lookup
   合并 pool.spec.{nodeSelector,tolerations} + unit.{requests,limits,nodeSelector}
   ▼ 注入 spec.scheduling.{nodeSelector,tolerations} + spec.roles[*].template.resources
   ▼ snapshot 到 spec jsonb（展开后即冻结，与 CR 解耦）
```

**合并规则**（与 [cluster-manager.md §3.2](cluster-manager.md#32-展开合并规则) 一致）：`pool.nodeSelector` 全保留（Pool 优先）；`unit.nodeSelector` 仅补 pool 未声明的 key；`pool.tolerations` 直作 `spec.scheduling.tolerations`；`unit.requests/limits` 写入 `roles[*].template.resources`。

**校验失败**：pool 不存在 → `400 pool-not-found`；unit 名不在 `pool.spec.units[]` → `400 unit-not-found`；Informer 未 sync（冷启）→ `WaitForCacheSync` 通过前 `/readyz` 不就绪。

**quota 名透传**：ElasticQuota 全名由调用方在创建请求的 `scheduling.quota` 显式提供；compute 仅校验其非空（空则 `400 validation`），原样写入 `spec.scheduling.quota` 并随展开一并 snapshot，不做任何组装。不校验配额是否存在（由 cluster-manager / tenant-operator 维护，axisml-scheduler 调度期强制）。

**snapshot 语义**：pool/unit CR 仅在 Create 入口读一次，展开结果固化进 PG `spec`；后续 reconciler 透传到 CR，compute-operator 直接读 spec 渲染 Pod，全程不感知 pool/unit。pool 删除或 unit 改值不影响已创建 workload。

**溯源 label**：展开后在 CR 与 PG `labels` 写 `resource.axisml.io/pool=<pool>` + `resource.axisml.io/unit=<unit>`，便于上游 pool/unit 删除前置阻断（按 labelSelector 反查活跃 workload）。

### 5.5 孤儿与补偿

Compute **不反向重建 CR**。Informer 观察 CR DELETE 后按 PG 当前 `status` 分流：

| PG 状态 | 处理 |
| --- | --- |
| `Canceling` | 幂等忽略；`Cancelled` 只由 Suspended condition 推进 |
| `Deleting` | 正常级联清理，推进 `Deleted` |
| Job `Pending`/`Running`（外部误删） | 推 `Cancelled` + `finishedAt` + message，不补偿重建 |
| Service `Pending`/`Ready`/`Degraded`/`Failed`（外部误删） | 写 `Deleting` + `deleted_at` + message，下一轮幂等确认后推 `Deleted` |
| TrafficPolicy 同 Service（外部误删） | 策略为纯声明态、PG 权威，下一轮 reconciler 按 `generation` 幂等重建 CR 恢复入口（成员不受影响） |
| 已终态 | 忽略 |

**正向孤儿**（PG `Creating` 无 CR 且未软删）属 Outbox 正常窗口，下一轮幂等重试 `Create()`。**反向孤儿**（CR 存在但 PG 无行或已 `Deleted`）：默认删 CR 并记日志。

## 6. 接口契约

| 类别 | 内容 | 引用 |
| --- | --- | --- |
| 对外 REST（K8s 风格） | `/api/v1/namespaces/{ns}/{mlruns,mlservices,traffic-policies}[...]`，含 `/{name}/{metrics,split,promote,rollback}` 子路径 | [openapi/compute-service.yaml](../apis/compute-service.yaml) |
| 下发 CR | `MLRun` / `MLService` / `MLTrafficPolicy`（`axisml.io/v1alpha1`, namespaced）；Compute 是唯一 `spec` 写者 | [compute-operator.md](compute-operator.md) |
| 回流字段 | `mlruns.status` / `mlservices.status` / `traffic_policies.status`（jsonb 整块） | [database.md §2](database.md#2-compute-service) |
| 列表查询 | 所有 list 支持 `?labelSelector=`（K8s grammar） | [database.md §1.6](database.md#16-扩展元数据-labels--annotations) |
| 身份头 | 调用方注入 `X-Axisml-User`，本服务仅做 ownership 归属（[auth.md §6](../../../axisml-platform/docs/system_design/auth.md#6-下游身份透传)） | — |
| 错误格式 | HTTP 标准码 + RFC 7807 problem+json | — |
| 写后语义 | mutation PG 提交后即返回；CR 同步异步进行，调用方 GET 观察 `status` | — |

不变量：CR `metadata` / `spec` 单写（Compute 写）；CR `status` 单读（operator 写）；API 不直接写 K8s。

## 7. 依赖

| 依赖 | 用途 |
| --- | --- |
| PostgreSQL | 业务元数据权威；与 artifacts 共享 database，表前缀隔离（[database.md §2](database.md#2-compute-service)） |
| Kubernetes API | 三类 CR 下发 + status watch + Pod log / Event 透传 + leader Lease |
| compute-operator | 下游 CR 消费者，把三类 CR 落地为底层资源（含加权路由 / 灰度派生）（[compute-operator.md](compute-operator.md)） |
| 上游调用方 | 唯一调用方，注入 `X-Axisml-User`；请求体仅携带 `(poolName, unitName)` 名字对，由本服务展开 |
| ResourcePool CR | 经 K8s Informer cache 直读，创建时展开为 Pod 原语 snapshot（[cluster-manager.md §3](cluster-manager.md#3-核心模型)） |
| artifacts | Compute 与 operator 均不直接调用；Platform 在创建入口完成 resolve，并把 URI / digest 快照随 spec 下发（[artifact-hub.md](artifact-hub.md)） |
| Prometheus | 运行指标查询（`--prometheus-url`，只读） |
| 对象存储（RustFS） | 为 Run / TensorBoard Pod 注入读写凭证、Run 软删时 GC 产出（[infra.md](../../../axisml-infra/docs/system_design/overview.md)） |

Compute 不感知 ElasticQuota CR 内部结构——以 `axisml-<identifier>-<pool>` 字符串透传到 CR，不校验配额是否存在。

## 8. 运行时形态

| 维度 | 值 |
| --- | --- |
| 进程 | 单二进制 `axisml-compute-service`；子命令 `serve` / `migrate` |
| 副本 | 默认 `replicas=1`（API 无状态可水平扩，reconciler / Informer 单 leader） |
| Leader Election | K8s `Lease`（`axisml-compute-service.axisml.io`）；`/metrics` 暴露 `axisml_compute_is_leader` |
| 暴露端口 | API `:8080`；Metrics `:8081`；Probes `:8082`（`/readyz` 校验 PG）；ClusterIP，无外部 HTTPRoute |
| RBAC scope | `mlruns`/`mlservices`/`mltrafficpolicies.axisml.io` 全权 + `resourcepools` `get/list/watch`（展开）+ `pods`/`pods/log`/`events` RO + 自身 ns `leases`；**不含** `persistentvolumeclaims`（持久卷由 Platform 经 cluster-manager 提前创建）/ `tenants` / `elasticquotas` / `namespaces` / `secrets` |
| Helm / 镜像 | 见 [deployment.md §8.3](../../../docs/deployment.md#83-helm-模板清单) |

## 9. 相关引用

- [high_level_design.md](../../../docs/high_level_design.md) — Compute 在控制平面的位置与系统不变量
- [auth.md](../../../axisml-platform/docs/system_design/auth.md) — `X-Axisml-User` 身份头与鉴权边界
- [database.md](database.md) — Compute 表结构（§2）
- [deployment.md](../../../docs/deployment.md) · [infra.md](../../../axisml-infra/docs/system_design/overview.md)
- [openapi/compute-service.yaml](../apis/compute-service.yaml) — REST 契约源
- [compute-operator.md](compute-operator.md) — 下游 CR 消费者与 handler 实现
- [cluster-manager.md](cluster-manager.md) — ResourcePool CRD 写入方（本服务经 Informer 直读做展开）；Volume REST 由 Platform 调用管理数据卷，本服务仅在 Pod 模板里引用挂载
- [artifact-hub.md](artifact-hub.md) — Job / Service 引用懒查询
