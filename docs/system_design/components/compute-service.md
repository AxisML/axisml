# AxisML Compute Service 概要设计

## 1. 定位与边界

ML 工作负载 + 多租户管控面：以 PostgreSQL 为权威，承载 Tenant / Quota / Job / Service / TrafficPolicy 的元数据，把 `Tenant` / `MLRun` / `MLService` / `MLTrafficPolicy` CR 当作 PG 行的派生产物下发到 K8s。

| 做 | 不做 |
| --- | --- |
| Tenant CRUD、软删与 restore | Namespace / ElasticQuota / initResources 落地 (→ [tenant-operator.md](tenant-operator.md)) |
| Quota（内联 `tenants.quotas` jsonb）CRUD | 修改 Node label / taint（admin 手工维护） |
| Job / Service CRUD、cancel / scale、软删；workspace 创建时附带 PVC 生命周期管理 | 直接创建 Pod / Deployment / PodGroup 等底层资源 (→ [compute-operator.md](compute-operator.md)) |
| 流量策略（MLTrafficPolicy）CRUD、split / promote / rollback、指标代理；成员校验权威 | 加权 HTTPRoute / 灰度的网关派生 (→ [compute-operator.md](compute-operator.md)) |
| `Tenant` / `MLRun` / `MLService` / `MLTrafficPolicy` spec 下发 + status 回流 | ResourcePool / ResourceUnit 词汇 (→ [cluster-manager.md](cluster-manager.md)) |
| Pod 列表 / Pod 日志 / 事件端点透传 kube-apiserver | 用户认证与角色鉴权 (→ [auth.md](../auth.md)) |
| 工作区列表（`kind='workspace'` 过滤） | 工作区业务语义本身 (→ [platform.md](platform.md)) |
| 在线服务 / 工作负载运行指标代理（按 `spec.backend` 选 PromQL 查 Prometheus） | 指标采集 / 存储（→ kube-prometheus-stack） |
| 跨 namespace 活跃 workload 按 label 计数（供 Platform pool/unit 删除前置阻断） | — |

**namespace 字段语义**：`mlruns` / `mlservices` 表的 `namespace text` 是 tenant 标识符（= `tenants.name`），逻辑分区键；Compute 内部 join `tenants` 表得到 `spec.namespace.name` 用于 CR 下发。

**Pool/Unit 展开归属**：Platform 仅向 compute-service 传 `(poolName, unitName)` 名字对；compute-service 通过 K8s Informer 直读 `ResourcePool` CR cache 完成展开（合并 `nodeSelector` / `tolerations` / `requests` / `limits` 写进请求体的 `spec.scheduling` / `spec.roles[*].template.resources`），然后 snapshot 到 `mlruns.spec` / `mlservices.spec` jsonb。snapshot 一经写入即与 pool/unit CR 解耦，pool/unit 后续修改 / 删除不影响已创建 workload。详见 §5.4。

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
                                │ tenants/mlruns/ │  │   Tenant / MLRun /   │  │ compute-operator│
                                │  mlservices     │  │   MLService          │  └────────────────┘
                                └─────────────────┘  └──────────────────────┘
```

### 2.2 内部结构

```
┌──────────────────────────── Compute (Go) ─────────────────────────────┐
│  HTTP API (Gin) ──写──▶  PG (generation + phase='Creating')          │
│        ▲                                │                              │
│        │ 读                              ▼                              │
│        │                       Reconciler goroutines (leader-only)    │
│        │            ┌── tenant ─┬─── job ───┬─ service ─┬─ traffic ─┐ │
│        │            └───────────┴───────────┴───────────┴───────────┘ │
│        │                                │                              │
│        │                                ▼ Create / Patch / Delete     │
│        │             Tenant / MLRun / MLService / MLTrafficPolicy CR   │
│        │                                │ status                       │
│        └──── PG status 列 ◀──── Informer (leader-only, shared cache)   │
└────────────────────────────────────────────────────────────────────────┘
```

四条 Reconciler 共享同一 leader Lease、同一 PG 连接池；四条 Informer 共享 `SharedInformerFactory`。

## 3. 核心模型

| 实体 | 含义 | 标识键 | 状态机 | 对应 CR |
| --- | --- | --- | --- | --- |
| Tenant | 租户 | `id` (uuid) / `name`（集群内全局唯一） | `Creating \| Active \| Suspended \| Failed \| Deleting \| Deleted` | `Tenant`（cluster-scoped） |
| Allocation | 资源池总额度（cap），内联在 `tenants.spec.allocations[]`，由 system-admin 设定 | `(tenant.name, pool)` | 无独立状态 | 每条渲染为父 ElasticQuota CR（由 tenant-operator 落地） |
| Quota | 子配额，内联在 `tenants.spec.quotas[]`，由 tenant-admin 在所属 pool 的 allocation 内拆分 | `(tenant.name, pool, name)` | 无独立状态 | 每条 1:1 渲染为子 ElasticQuota CR（父为对应 pool 的 allocation） |
| Job | 一次性训练 / 离线任务 | `id` (uuid) / `(namespace, name)` | `Creating \| Pending \| Running \| Succeeded \| Failed \| Canceling \| Cancelled \| Deleting \| Deleted` | `MLRun`（namespaced） |
| Service | 常驻在线服务 / 工作区 | `id` (uuid) / `(namespace, name)` | `Creating \| Pending \| Ready \| Degraded \| Failed \| Deleting \| Deleted` | `MLService`（namespaced） |
| TrafficPolicy | 流量策略：稳定入口按权重分发到多个在线服务 | `id` (uuid) / `(namespace, name)` | `Creating \| Pending \| Ready \| Degraded \| Failed \| Deleting \| Deleted` | `MLTrafficPolicy`（namespaced） |

字段级 schema 见 [database.md §2](../database.md#2-compute-service)；CR spec 字段见 [tenant-operator.md](tenant-operator.md) / [compute-operator.md](compute-operator.md)。

**通用 PG 约定**：所有表带 `id uuid` / `created_at` / `updated_at` / `deleted_at`；所有 UNIQUE 实现为 partial unique index `WHERE deleted_at IS NULL`（软删行不占用唯一键，同名可再次创建）；`name` 统一 DNS-1123 校验，长度 3–40。CR-backed 对象额外打 `axisml.io/{tenant,run,service,traffic-policy}-id=<uuid>` label 作为稳定锚点（`metadata.name` 因软删可重用，UUID 永久唯一）；`mlservices` 还会同步打 `axisml.io/service-kind=<service|workspace>` label，便于 `kubectl` selector 区分工作区与普通服务（compute-service / operator 不按 kind 改变行为）。

**扩展元数据 + 分组维度**：`tenants` / `mlruns` / `mlservices` 表均带 `labels jsonb` + `annotations jsonb` 双字段，对齐 K8s 风格语义，统一约定见 [database.md §1.6](../database.md#16-扩展元数据-labels--annotations)。上层可借 `labels` 实现任意分组维度，list 端点支持 `?labelSelector=` K8s 语法。两类扩展位均 **PG-only、不下发 CR、不 `+generation`**。

## 4. 核心功能

### 4.1 Tenant

**状态机**：

```
[POST]──▶ Creating ──(namespaceReady)──▶ Active ⇄[suspend/resume]⇄ Suspended
                                            │
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
| 更新 spec（`spec.allocations[].{min,max}` / `spec.quotas[].{min,max}` / `spec.initResources`） | update `spec` + `generation += 1` | reconciler patch CR | `spec.namespace.name` / `spec.allocations[].pool` / `spec.quotas[].{pool,name}` 不可变 |
| 更新顶层 PG 元数据（`display_name` / `description` / `labels` / `annotations`） | update 行 | **不影响 CR** | 不 `+generation`；扩展位见 [database.md §1.6](../database.md#16-扩展元数据-labels--annotations)；包含 `namespace.labels` / `namespace.annotations` 这类纯展示位 |
| 软删 | `phase='Deleting'` + `deleted_at = now()` + `generation += 1` | reconciler 删 CR | 行保留到 retention |
| 恢复 | `deleted_at = NULL` + `generation += 1` | reconciler 重建 CR | 仅适用 `phase='Deleted'` 行 |
| 暂停 / 恢复 | `phase='Suspended'` / `phase='Active'` | **不下发 / 不改 CR**；不 `+generation` | 仅 `Active`↔`Suspended` 切换；reconciler 无匹配谓词 |

`Failed` 是非终态，operator 自愈后自然回到 `Active`。

**暂停语义**：`Suspended` 与 `Creating` / `Canceling` / `Deleting` 一样，是 API 直接写入 `phase` 的意图态——区别是它没有匹配的 reconciler 谓词（§5.1），故不下发 / 不改 Tenant CR、不 `+generation`、tenant-operator 不参与。informer 回流时**跳过** `phase='Suspended'` 行的 phase 推进，仅刷新其 `status` 子树（conditions / quotas）；resume 把 `phase` 复位 `Active` 后恢复正常回流。创建 Job / Service / Workspace 端点在写 PG 前校验本租户 `phase != 'Suspended'`，否则 `409 tenant-suspended`；存量 workload 的 `cancel` / `scale` / `stop` / `delete` 不受闸门限制。

**与 Tenant CR 契约**：

| Tenant `status.phase` | PG `status` |
| --- | --- |
| `Pending` | `Creating` |
| `Active` | `Active`（含 `namespaceReady=true`） |
| `Failed` | `Failed`（可自愈） |

`Suspended` 不来自 Tenant CR——它是 API 直接置入 `phase` 的意图态，informer 不覆盖（见上文「暂停语义」）。

`status.quotas[].used` **不入 PG**——由 tenant-operator 写到 Tenant CR `status.quotas[].used`，compute-service Tenant Informer 内存 cache 现读，GET 时与 PG 中的 spec / phase 合并返回。详见 [§5.3](#53-状态回流informer)。

### 4.2 Allocation 与 Quota（两级配额）

配额是**两级模型**，均无独立 CRD、均内联在 Tenant spec：

- **总额度 Allocation**（`tenants.spec.allocations[]`，键 `(pool)`）：system-admin 为租户在某资源池下设定的**总量上限（cap）**，渲染为该 `(tenant, pool)` 的**父** ElasticQuota。
- **子配额 Quota**（`tenants.spec.quotas[]`，键 `(pool, name)`）：tenant-admin 在所属 pool 的 allocation 内**拆分**出的命名子额度（如按团队），渲染为父下的**子** ElasticQuota；workload 只绑定子配额。

| 操作 | 角色 | PG 写 | CR 影响 |
| --- | --- | --- | --- |
| 设 / 改总额度 `allocations[]` | `system-admin` | upsert jsonb 项 + `generation += 1` | reconciler patch 父 ElasticQuota |
| 删总额度 | `system-admin` | 该 pool 无子配额时移除项 + `generation += 1` | reconciler 删父 ElasticQuota |
| 增 / 改 / 删子配额 `quotas[]` | `tenant-admin@self` | 写 jsonb 项 + `generation += 1` | reconciler patch / 删子 ElasticQuota |
| 用量回流 | — | 不入 PG；从 Tenant Informer cache 现读 `status.quotas[i].used` | — |

**额度不超分**（compute-service 写前校验，超出返 `400 quota-exceeds-allocation`）：对每个 pool，`Σ quotas[].min ≤ allocations[pool].min` 且 每项 `quotas[].max ≤ allocations[pool].max`；子配额的 `pool` 必须存在对应 allocation。tenant-operator 落地时按同一约束兜底，Koordinator 按父 / 子 ElasticQuota 层级在调度期强制。

**不变约束**：`allocations[].pool` 与 `quotas[].(pool, name)` 一旦创建即不可变；子配额改名 = 先删后建；删总额度前该 pool 的子配额必须先清空。

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
| 提交 | insert `Creating` 行 + spec 快照（已含展开后的 `nodeSelector` / `tolerations` / `resources`） | reconciler `Create()` MLRun | 创建后 spec 不可变；无后续 mutation，仅靠 `phase='Creating'` 谓词推进 |
| cancel | `phase='Canceling'` + `message='user cancelled'` | reconciler `patch spec.runPolicy.suspend=true` | `Creating` 状态拒绝；要求改用 DELETE |
| 更新 PG 元数据（`display_name` / `description` / `labels` / `annotations`） | update 行 | **不影响 CR** | Job spec 不可变，但 PG 扩展位任意阶段可改 |
| 软删 | `phase='Deleting'` + `deleted_at=now()` | reconciler `Delete()` CR；Informer DELETE → `Deleted` | 任一非 `Deleting`/`Deleted` 状态适用 |
| Pod 列表 / Pod 日志 / Pod 事件 / Job 事件 | — | 按 `axisml.io/run-id` label list Pod；按 Pod 名透传 Pod Log；按 `involvedObject` 过滤 Event（Pod 端点只回 Pod 事件，Job 端点只回 MLRun/PodGroup 事件） | 详见 [apis/compute-service.yaml](../apis/compute-service.yaml) `MLRuns` tag |
| list 过滤 | — | `?labelSelector=` 支持 K8s 语法，常用 `axisml.io/project=<id>` | 见 [database.md §1.6](../database.md#16-扩展元数据-labels--annotations) |

`Succeeded` / `Failed` / `Cancelled` 为运行终态；`Deleted` 为软删终态。`Cancelled` PG 行保留（`deleted_at IS NULL`），用户可再次 DELETE。

**与 MLRun CR 契约**：

| MLRun `status.phase` | PG `status` |
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
| 创建 | insert `Creating` 行 + spec 快照（含展开后的 `nodeSelector` / `tolerations` / `resources`） | reconciler `Create()` MLService（含 `axisml.io/service-{id,kind}` label）；`kind='workspace'` 时同事务派生 PVC `axisml-ws-<service-name>-data`，挂到 `roles[0].template.volumes[]` |
| scale | 更新 `spec.roles[0].replicas` + `generation += 1` | reconciler `generation>observed_generation` 触发 patch `spec/roles/0/replicas` |
| 更新 PG 元数据（`display_name` / `description` / `labels` / `annotations`） | update 行 | **不影响 CR**；不 `+generation` |
| 软删 | `phase='Deleting'` + `deleted_at=now()` | reconciler `Delete()` CR；Informer DELETE → `Deleted`；`kind='workspace'` 时按 `?deletePvc=true`（默认 true）一并 `Delete()` PVC |
| Pod 列表 / Pod 日志 / Pod 事件 / Service 事件 | — | 按 `axisml.io/service-id` label list Pod；按 Pod 名透传 Pod Log；按 `involvedObject` 过滤 Event（Pod 端点只回 Pod 事件，Service 端点只回 MLService/底层 Workload/HTTPRoute 事件） |
| list 过滤 | — | `?labelSelector=` 支持 K8s 语法，常用 `axisml.io/project=<id>` |
| 指标查询 | — | 按 `spec.backend` 选 PromQL 模板查 Prometheus，返回 `MetricSeries`（QPS / 延迟 / 错误率 / CPU·内存·GPU 利用率） |

Service 无 cancel 语义；除 `roles[0].replicas` 外其他 spec 字段不可变。`kind` 创建后不可变。

**workspace PVC 生命周期**：`kind='workspace'` 时 compute-service 同事务创建 PVC（命名 deterministic：`axisml-ws-<service-name>-data`，size / storageClass 由请求体携带）；PVC 失败 → 同事务回滚 MLService 行，整个 POST 返 5xx。删除时按 `?deletePvc=true`（默认）一并 GC 后端 PVC，否则保留供后续 restore 接入。MLService Pod spec 的 `volumes[].persistentVolumeClaim.claimName` 直接指向该 PVC；compute-operator 不感知 workspace 的存储语义。

**与 MLService CR 契约**：

| 条件 | PG `status` |
| --- | --- |
| `ready_replicas == desired_replicas && desired_replicas > 0` | `Ready` |
| `0 < ready_replicas < desired_replicas` | `Degraded` |
| `ready_replicas == 0 && desired_replicas > 0` 且 CR `phase=Pending` | `Pending` |
| `ready_replicas == 0 && desired_replicas > 0` 且 CR `phase=Failed` | `Failed`（可自愈） |
| `desired_replicas == 0` | `Pending` |

**kind 过滤**：`GET /api/v1/namespaces/{ns}/mlservices?kind=workspace` 供 [Platform 工作区](platform.md) 在同一张表上区分 `kind='service'` 与 `kind='workspace'`；`kind` 创建后不可变，Compute 不按 `kind` 改变行为，仅作分类与过滤。

### 4.5 流量策略（MLTrafficPolicy）

流量策略把一个稳定对外入口的入站流量按权重分发到同租户下多个**在线服务**（`kind='service'` 的 Service）后端，支撑加权切分、多版本灰度与蓝绿切换。compute-service 是 `MLTrafficPolicy` CR 的唯一 spec 写者，持有权重 / 灰度态权威并代理指标查询；加权路由的网关派生（`(native,httproute)` → Envoy Gateway `HTTPRoute` 加权 `backendRefs`；`(kserve,inference)` → `InferenceService` canary）由 [compute-operator](compute-operator.md#43-mltrafficpolicy-controller) 完成，compute-service 既不直连网关也不内嵌 PromQL。Platform 侧编排见 [platform.md §4.8](platform.md#48-流量配置编排)。

**模式（`mode`，创建后不可变）**：

| mode | 语义 | backends | 可变维度 | 派生权重 |
| --- | --- | --- | --- | --- |
| `weighted` | 加权切分 | N≥2 成员，各带权重 | `backends[*].weight`（Σ=100） | 各成员显式权重 |
| `canary` | 多版本灰度 | 稳定基线 + 灰度后端 | 灰度百分比 `p∈[0,100]` | 灰度 = p、稳定 = 100−p 自动派生 |
| `bluegreen` | 蓝绿切换 | 恰好 2 成员 | 全量切换 | 一侧 100、另一侧 0（all-or-nothing） |

`bluegreen` 切换复用 `split` 写路径（权重约束为全量切换），不引入独立动作；`canary` 额外有 `promote` / `rollback` 两个专属动作。

**`role` 与基线约定**：canary 模式下 backends 恒为「一个 `role=stable`（当前基线）+ 一个 `role=canary`（灰度）」——**当前稳定基线即 `role=stable` 的后端，不另设指针字段**。`promote` 互换二者 `role`（灰度升为新 `stable`），`rollback` 仅回收 canary 权重。`weighted` 模式 `role` 留空、按显式权重分发；`bluegreen` 模式用 `blue` / `green` 标识两槽，active 一侧由权重派生。

**状态机**：

```
Creating ──(Informer ADD)──▶ Pending ──(route programmed + 成员 Ready)──▶ Ready ⇄ Degraded ──▶ Failed
                                                                          ▲                     │
                                                                          └─────── 自愈 ────────┘

任一非 Deleting/Deleted ──[DELETE]──▶ Deleting ──(CR 确认清理 + deleted_at)──▶ Deleted
```

同 Service，`Ready` / `Degraded` / `Failed` 均为非终态，只有 `Deleted` 为终态。weighted split、canary promote/rollback、bluegreen switch 都是 `backends[*].weight` 的 spec mutation（`+generation`），不改 phase。

**行为约束**：

| 操作 | PG 写 | CR 影响 |
| --- | --- | --- |
| 创建 | 成员预检通过后 insert `Creating` 行 + spec 快照（`mode` / `endpoint` / `backends` / 派生 `backend` 元组） | reconciler `Create()` MLTrafficPolicy（含 `axisml.io/traffic-policy-id` label） |
| 调整流量（split） | weighted 写回 `backends[*].weight`；canary 写灰度百分比并派生稳定权重；bluegreen 翻转 100/0 — 均 `generation += 1` | reconciler patch `spec.backends[*].weight` |
| 提升 / 回滚（canary 专属） | `promote` 互换 stable/canary 两后端的 `role` 并置新 `stable`=100、原 stable=0；`rollback` 置 canary 后端权重 0（stable 回到 100）— 均 `generation += 1` | reconciler patch |
| 更新 PG 元数据（`display_name` / `description` / `labels` / `annotations`） | update 行 | **不影响 CR**；不 `+generation` |
| 软删 | `phase='Deleting'` + `deleted_at=now()` | reconciler `Delete()` CR；派生 `HTTPRoute` / `SecurityPolicy` 经 ownerReference 级联回收；**成员 MLService 不随策略删除** |
| 指标查询 | — | 按成员后端分组查 PromQL（QPS / 延迟 / 错误率 + 灰度健康对比），返回 `MetricSeries`；不入 PG |
| list 过滤 | — | `?labelSelector=` 支持 K8s 语法 |

`endpoint`（path / hostname / auth）与 `mode` 创建后不可变——改模式 / 改入口 = 先删后建。

**成员校验（创建时，compute-service 为权威）**：

- 每个成员经内部 `GetService` 预检 `kind=='service'`（拒绝 workspace）且当前 `Ready`；成员必须同租户（同 `namespace` 分区）；
- 一个 MLService 同时只能被一个活跃策略引用（占用校验，避免多策略争抢同一后端的对外入口；非 DB 约束，由应用层在创建/删除事务内维护）；
- 成员后端类型同构——全 `native` 或全 `kserve`；`kserve` 成员要求 `mode='canary'` 且恰好 2 个 backends。校验后把派生的 `spec.backend` 路由元组（`(native,httproute)` / `(kserve,inference)`）写入 CR，供 operator dispatcher 路由；
- `endpoint.path==""` 时自动拼 `/services/<tenant>/<name>/`。

Platform 侧创建编排（[platform.md §4.8](platform.md#48-流量配置编排)）也做同一组成员预检，但仅为快速失败 UX；本服务的校验是事务内的权威闸门。

**与 MLTrafficPolicy CR 契约**：

| 条件 | PG `status` |
| --- | --- |
| route programmed 且所有成员 `ready` | `Ready` |
| 部分成员 ready、部分 degraded | `Degraded` |
| route 未 programmed 或全部成员未就绪，CR `phase=Pending` | `Pending` |
| CR `phase=Failed`（成员 Service 解析失败 / 派生冲突） | `Failed`（可自愈） |

`status` jsonb 持 `{phase, message, endpoint, backends[].{serviceName, weight, ready}, conditions[]}`，由 MLTrafficPolicy Informer 整块回流（[§5.3](#53-状态回流informer)）。权重 / 灰度态 / 成员 phase 始终以本服务为权威源回源给 Platform。

## 5. 关键机制

### 5.1 写路径：内嵌 Outbox + 谓词扫描

无独立 outbox 表——借用业务表自身的 `status` / `deleted_at` / `generation` 列。API 同步路径只写 PG（业务校验 → 事务内写状态 + spec 快照 + `generation += 1` → commit → 返回业务 ID）；reconciler goroutine 在 leader 副本按以下谓词扫描分派：

| 谓词 | 动作 | 适用模块 |
| --- | --- | --- |
| `phase='Creating' AND deleted_at IS NULL` | `Create()` CR（带 `axisml.io/<resource>-id` label；409 视为成功） | Tenant / Job / Service / TrafficPolicy |
| `phase='Canceling'` | `patch MLRun.spec.runPolicy.suspend=true` | Job |
| `phase='Deleting'` | `Delete()` CR；Informer DELETE 推进 `Deleted` | Tenant / Job / Service / TrafficPolicy |
| `generation <> observed_generation AND deleted_at IS NULL` | `Patch()` CR；成功后 `observed_generation = generation` | Tenant / Service / TrafficPolicy |

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
| Tenant | 是 | `spec.quotas[].{min,max}`、`spec.initResources` |
| Service | 是 | `spec.roles[0].replicas` |
| TrafficPolicy | 是 | `spec.backends[*].{weight,role}`（split / rollback 仅改 weight；canary `promote` 同时互换 stable/canary 两后端的 role） |
| Job | 否 | 不可变；cancel 走 `Canceling` 谓词单独 patch suspend |

### 5.3 状态回流（Informer）

四条独立 Informer，通过 `k8sclient` 的 `SharedInformerFactory` 共享 cache，仅 leader 副本运行。

| Informer | 监听对象 | 主要回写字段 |
| --- | --- | --- |
| Tenant Informer | `Tenant` CR | 写 `tenants.status` jsonb（`{phase, message, namespaceReady, conditions[], quotas[].{pool, name, ready}}`）；**主动 strip `quotas[].used`**——该字段保留在 Informer in-memory cache 中，GET 时实时聚合返回；按 §4.1 phase 映射表推进，但 `phase='Suspended'` 的行只刷新 status 子树、不推进 phase（API 意图态） |
| MLRun Informer | `MLRun` CR | 整块写 `mlruns.status` jsonb（`{phase, message, startedAt, finishedAt, conditions[]}`）；按 §4.3 phase 映射表推进 |
| MLService Informer | `MLService` CR | 整块写 `mlservices.status` jsonb（`{phase, message, readyReplicas, endpoint, conditions[]}`）；按 §4.4 条件表推进 |
| MLTrafficPolicy Informer | `MLTrafficPolicy` CR | 整块写 `traffic_policies.status` jsonb（`{phase, message, endpoint, backends[].{serviceName, weight, ready}, conditions[]}`）；按 §4.5 条件表推进 |

启动时 `List` 做差异 upsert 与孤儿对账；`Watch` 事件入各自 work queue；单 worker 串行 reconcile；以 `resourceVersion` / `generation` 作乐观并发字段。

**`quotas[].used` 不入 PG 的语义约束**：

- 该字段是调度即变的 ephemeral 态，落 PG 会制造"PG 值与现实漂移"的窗口；权威源是 koord-scheduler 写入的 `ElasticQuota.status.used`，由 [tenant-operator §5.3](tenant-operator.md#53-elasticquota-statusused-回流路径) 聚合到 Tenant CR `status.quotas[].used`，compute-service 仅以 Tenant CR 为消费入口。
- GET tenant 端点合并路径：`SELECT FROM tenants` 取 spec + PG `status`（不含 `used`） → Tenant Informer cache lookup 拿 `status.quotas[].used` → merge 后返回。
- 启动期 `WaitForCacheSync` 通过前 `/readyz` 不就绪，避免 cache 冷启窗口返回 `used=null`；运行期 informer 断线超过 stale TTL（默认 30s）时 GET 返回 `used=null` + warning header，不撒谎。
- 多副本 compute-service 各自维护 cache，自然收敛，无须协调。
- 历史趋势查询走 Prometheus（koord-scheduler 自带 ElasticQuota 指标），PG 不承担时序。

### 5.4 ResourcePool 展开

compute-service 接收 Job / Service / Workspace 创建请求时，**自己**完成 pool/unit 到 K8s Pod 原语的展开，Platform 不参与展开逻辑：

```
POST /api/v1/namespaces/{ns}/mlruns
   body: { name, scheduling: { poolName, unitName, quota, ... }, ... }
         │
         ▼ ResourcePool Informer cache lookup
         │
   合并 pool.spec.{nodeSelector,tolerations} + unit.{requests,limits,nodeSelector}
         │
         ▼
   注入 spec.scheduling.{nodeSelector, tolerations} + spec.roles[*].template.resources
         │
         ▼
   snapshot 到 mlruns.spec / mlservices.spec jsonb (展开后即冻结，与 CR 解耦)
```

**合并规则**（与 [cluster-manager.md §3.2](cluster-manager.md#32-展开合并规则) 一致）：

| 来源 | 合并行为 |
| --- | --- |
| `pool.spec.nodeSelector` | key 全部保留（Pool 优先） |
| `unit.nodeSelector` | 仅贡献 pool 未声明的 key |
| `pool.spec.tolerations` | 直接作为 `spec.scheduling.tolerations` |
| `unit.requests` / `limits` | 写入 `spec.roles[*].template.resources` |

**校验失败语义**：
- pool CR 不存在 → `400 pool-not-found`
- unit 名在 `pool.spec.units[]` 内找不到 → `400 unit-not-found`
- `(poolName, quota)` 不在 `tenant.spec.quotas[]` 中 → `400 quota-not-found`
- Informer cache 未 sync（compute-service 冷启） → `WaitForCacheSync` 通过前 `/readyz` 不就绪，Pod 不接流量，外部观察不到这种状态

**quota 名组装**：Platform 仅传短 quota 名；compute 校验 `(poolName, quota) ∈ tenant.spec.quotas` 后组装全名 `axisml-<tenant>-<pool>-<quota>` 写入 `spec.scheduling.quota`，随展开结果一并 snapshot 冻结。

**snapshot 语义**：pool/unit CR 仅在 Create 入口被读取一次，展开结果一次性写入 PG `spec` 快照后即固化。后续 reconciler 把 spec 透传到 MLRun / MLService CR；compute-operator 直接读 spec 渲染 Pod，全程不感知 pool/unit 概念。pool 删除或 unit 改值不会影响已创建 workload。

**与 [§5.3 Tenant Informer cache](#53-状态回流informer) 复用一套基础设施**：compute-service 已经持有 K8s SharedInformerFactory，加一条 ResourcePool Informer 零边际复杂度。

**溯源 label**：展开成功后, compute-service 在创建的 `MLRun` / `MLService` CR 上自动打 label `axisml.io/resource-pool=<poolName>` + `axisml.io/resource-unit=<unitName>`, 同时往 `mlruns.labels` / `mlservices.labels` 写同一对 key, 便于 Platform 在 pool/unit 删除前置阻断（按 labelSelector 反查活跃 workload, 详见 [platform.md §4.6](platform.md#46-资源池编排)）。

### 5.5 孤儿与补偿

Compute **不反向重建 CR**。Informer 观察到 CR DELETE 事件后按 PG 当前 `status` 分流：

| PG 当前状态 | 处理 |
| --- | --- |
| `Canceling` | 幂等忽略；`Cancelled` 只由 Suspended condition 推进 |
| `Deleting` | 正常级联清理，推进 `Deleted` |
| Job 在 `Pending`/`Running`（外部误删） | 推 `phase=Cancelled` + `status.finishedAt` + `status.message='external delete'`，不补偿重建 |
| Service 在 `Pending`/`Ready`/`Degraded`/`Failed`（外部误删） | 写 `phase=Deleting` + `deleted_at=now()` + `status.message='external delete'`，下一轮谓词幂等确认后推 `Deleted` |
| Tenant 在 `Active`/`Failed`（外部误删） | 下一轮 reconciler 检测到 `generation <> observed_generation` 时重建 CR 恢复期望状态（PG 为权威） |
| TrafficPolicy 在 `Pending`/`Ready`/`Degraded`/`Failed`（外部误删） | 同 Tenant：策略为纯声明态、PG 为权威，下一轮 reconciler 按 `generation` 幂等重建 CR 恢复对外入口（成员 MLService 不受影响） |
| 已终态 | 忽略 |

**正向孤儿**（PG `Creating` 但无 CR，且 `deleted_at IS NULL`）属 Outbox 正常窗口，reconciler 下一轮幂等重试 `Create()`。**反向孤儿**（CR 存在但 PG 无行或已 `Deleted`）：默认删除 CR 并记录审计。

## 6. 接口契约

| 类别 | 内容 | 引用 |
| --- | --- | --- |
| 对外 REST（K8s 风格） | `/api/v1/namespaces[/{namespace}]`（Tenant CRUD）、`/api/v1/namespaces/{namespace}/{restore,suspend,resume}`、`/api/v1/namespaces/{namespace}/quotas[/{pool}/{name}]`、`/api/v1/namespaces/{namespace}/mlruns[...]`、`/api/v1/namespaces/{namespace}/mlservices[...]`、`/api/v1/namespaces/{namespace}/mlservices/{name}/metrics`、`/api/v1/namespaces/{namespace}/traffic-policies[...]`（含 `/{name}/{split,promote,rollback,metrics}` 子路径） | [apis/compute-service.yaml](../apis/compute-service.yaml) `Tenants` / `Quotas` / `MLRuns` / `MLServices` / `TrafficPolicies` tag |
| 跨 namespace 聚合 | `GET /api/v1/workloads/count?labelSelector=&active=true`（忽略分区按 label 统计活跃 Job/Service）、`GET /api/v1/workloads/metrics?...`（租户 / 集群级工作负载时序） | [apis/compute-service.yaml](../apis/compute-service.yaml) `Workloads` tag |
| 下发 CR | `Tenant`（`axisml.io/v1alpha1`, cluster-scoped）、`MLRun` / `MLService` / `MLTrafficPolicy`（namespaced）；Compute 是唯一 `spec` 写者 | [tenant-operator.md](tenant-operator.md) / [compute-operator.md](compute-operator.md) |
| 回流字段 | `tenants.status` / `mlruns.status` / `mlservices.status` / `traffic_policies.status`（jsonb 整块） | [database.md §2](../database.md#2-compute-service) |
| 不变量 | CR `metadata` / `spec` 单写（Compute 写）；CR `status` 单读（operator 写）；API 不直接写 K8s | — |
| 列表查询 | 所有 list 端点支持 `?labelSelector=` （K8s grammar：`=`/`==`/`!=`/`in (...)`/`notin (...)`/`key`/`!key`，逗号分隔 AND） | [database.md §1.6](../database.md#16-扩展元数据-labels--annotations) |
| 身份头 | 调用方注入 `X-Axisml-User`，本服务仅做审计与 ownership 归属 | [auth.md §6](../auth.md#6-下游身份透传) |
| 错误格式 | HTTP 标准状态码 + RFC 7807 problem+json | — |
| 写后语义 | mutation 在 PG 提交后即返回；CR 同步异步进行，调用方通过 GET 观察 `status` | — |

## 7. 依赖

| 依赖 | 用途 | 引用 |
| --- | --- | --- |
| PostgreSQL | 业务元数据权威；与 artifacts 共享 database `axisml`，表前缀隔离 | [database.md §2](../database.md#2-compute-service) / [infra.md](../infra.md) |
| Kubernetes API | `Tenant` / `MLRun` / `MLService` / `MLTrafficPolicy` CR 下发 + status watch + Pod log / Event 透传 + leader Lease | — |
| tenant-operator | 下游 CR 消费者，把 Tenant CR 落地为 K8s Namespace / Secret / ElasticQuota 等 | [tenant-operator.md](tenant-operator.md) |
| compute-operator | 下游 CR 消费者，把 MLRun / MLService / MLTrafficPolicy CR 落地为底层资源（含加权路由 / 灰度派生） | [compute-operator.md](compute-operator.md) |
| Platform | 上游唯一调用方；注入 `X-Axisml-User`；请求体仅携带 `(poolName, unitName)` 名字对，由 compute-service 自己展开 | [auth.md](../auth.md) / [platform.md §4.2](platform.md#42-计算任务编排) |
| ResourcePool CR | compute-service 通过 K8s Informer cache 直读, 创建 Job/Service 时按 `(poolName, unitName)` 展开为 Pod 原语 snapshot | [cluster-manager.md §3](cluster-manager.md#3-核心模型) |
| artifacts | image / model / dataset 引用懒查询；**由 operator handler 侧调用 resolve API，Compute 自身不调用** | [artifact-hub.md](artifact-hub.md) |
| Prometheus | 在线服务 / 工作负载运行指标查询（`GetServiceMetrics` / `GetWorkloadMetrics`，`--prometheus-url`）；只读 | [infra.md](../infra.md) / [monitoring.md](../monitoring.md) |

Compute 不感知 ElasticQuota CR 内部结构——接收 Platform 传入的短 quota 名，校验 `(pool, quota) ∈ tenant.spec.quotas` 后组装全名 `axisml-<tenant>-<pool>-<quota>` 写入 `spec.scheduling.quota`，再字符串透传到 Tenant CR / MLRun / MLService。

## 8. 运行时形态

| 维度 | 值 |
| --- | --- |
| 进程 | 单二进制 `axisml-compute-service`；子命令 `serve` / `migrate` |
| 副本 | 默认 `replicas=1`（API 无状态可水平扩，reconciler / Informer 单 leader） |
| Leader Election | K8s `Lease`（`axisml-compute-service.axisml.io`）；单副本退化为单成员瞬时 lease；`/metrics` 暴露 `axisml_compute_is_leader` gauge |
| 暴露端口 | API `:8080`；Metrics `:8081`；Probes `:8082`（`/healthz` liveness / `/readyz` 校验 PG）；ClusterIP，无外部 `HTTPRoute` |
| RBAC scope | `tenants.axisml.io` / `mlruns.axisml.io` / `mlservices.axisml.io` / `mltrafficpolicies.axisml.io` 全权 + `resourcepools.axisml.io` `get/list/watch` (Pool 展开) + 跨 tenant ns 的 `persistentvolumeclaims` `get/list/watch/create/delete`（仅 workspace 派生）+ `pods` / `pods/log` / `events` RO + 自身 ns 的 `leases`；**不含** `elasticquotas` / `namespaces` / `secrets`（这些由 tenant-operator 落地） |
| Helm values / 镜像 | 详见 [deployment.md §6.1](../deployment.md#61-cluster-manager--compute--artifacts--platform-backend) |

## 9. 相关引用

- [overview.md](../overview.md) — Compute 在控制平面里的位置
- [auth.md](../auth.md) — `X-Axisml-User` 身份头协议与鉴权边界
- [database.md](../database.md) — Compute 表结构（§3）
- [deployment.md](../deployment.md) — Helm 模板与发布编排
- [monitoring.md](../monitoring.md) — Compute Prometheus 指标列表
- [infra.md](../infra.md) — Koordinator / koord-scheduler / ElasticQuota 依赖契约
- [apis/compute-service.yaml](../apis/compute-service.yaml) — REST API 契约源
- [tenant-operator.md](tenant-operator.md) — Tenant CR 字段契约与落地行为
- [compute-operator.md](compute-operator.md) — `MLRun` / `MLService` CR 字段契约与 handler 实现
- [artifact-hub.md](artifact-hub.md) — Job / Service 提交时引用懒查询（image / model / dataset）
- [cluster-manager.md](cluster-manager.md) — ResourcePool CRD 的写入方; compute-service 通过 Informer 直读做展开
- [platform.md](platform.md) — 上游调用方；工作区在 `mlservices` 表上的 `kind='workspace'` 复用
