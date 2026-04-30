# AxisML Compute 详细设计

## 1. 概述

AxisML Compute 是平台的计算服务层，基于 Go 开发，承载 **计算任务管理** 与 **系统管理（租户 / 资源池 / 资源单元 / 队列）** 两大职责。Compute 通过 REST API 暴露能力，仅接受来自 AxisML Platform 的内部调用，不直接对外部用户流量开放。

**关键边界原则**：Compute 不直接创建 Namespace、Pod 等底层 K8s 资源，这些由对应 Operator 或集群管理员负责；Compute 仅维护业务元数据，并通过 CRD 向 Operator 声明意图。

**与 backend 解耦**：mljob-operator / mlservice-operator 内部按 `spec.backend.{name, engine}` 元组路由到不同 Handler。MLJob 支持 `(native, job)`（默认）/ `(volcano, podgroup)` / `(volcano, volcanojob)` / `(kubeflow-trainer, pytorch / tensorflow / mpi …)` / `(custom, *)`；MLService 支持 `(native, deployment)`（默认）/ `(native, statefulset)` / `(kserve, inference)` / `(kserve, llminference)` / `(custom, *)`，其中 `(kserve, inference)` 对应 KServe `InferenceService` CR、内部 runtime（triton / vllm / tfserving / torchserve / huggingface …）由 `backend.config.runtime` 选择，`(kserve, llminference)` 对应 KServe `LLMInferenceService` CR、承载 LLM 原生服务（PD 分离）。**Compute 完全不感知后端选择**：API 透传 `backend.{name, engine, config}` 三元组到 CR，Informer 仍只消费统一的 `status.phase`；状态机、PG schema、reconciler 三谓词均不变。具体插件机制见 [operators/mljob-operator.md](operators/mljob-operator.md) §2、[operators/mlservice-operator.md](operators/mlservice-operator.md) §2。

## 2. 职责与边界

Compute 内部按 6 个模块划分，分两类组织：

| 类别 | 模块 | 职责 | 边界外 |
| --- | --- | --- | --- |
| 配置对象（长生命周期） | 租户 Tenant | 元数据 CRUD，下发 `Tenant` CRD | Namespace 创建、ResourceQuota 落地（由 tenant-operator） |
| | 资源池 ResourcePool | `labelSelector` + `tolerations` 描述的池元数据 | Node 自身打标 / 污染（由集群管理员） |
| | 资源单元 ResourceUnit | 池内资源规格模板（`requests` / `limits` + 节点标签匹配） | 实际调度（Volcano） |
| | 队列 Queue | 扁平队列三维配额 `quota`（capability / deserved / guarantee）元数据 + Volcano Queue CR 同步 + 用量回流 | 实际公平调度 / 抢占 / 用量核算（Volcano） |
| 工作负载对象（生命周期） | 任务 Job | `MLJob` CRUD + 状态回流 | Pod / PodGroup 编排（由 mljob-operator） |
| | 服务 Service | `MLService` CRUD + 状态回流 | Deployment / 自动扩缩（由 mlservice-operator） |

两类对象的关键差异在补偿策略：**配置对象**的 CR 缺失/漂移默认由 Compute 按 PG 快照补偿重建；**工作负载对象**的 CR 缺失是合法的生命周期终点（取消、清理、operator 级联），Compute 不重建。详见 §5.4。

## 3. 整体架构

```
                          ┌──────────────────────┐
                          │  AxisML Platform      │
                          │     (唯一调用方)        │
                          └──────────┬───────────┘
                                     │ REST / JSON
                                     │ 身份透传 header
                                     ▼
                          ┌──────────────────────┐
                          │   AxisML Compute      │
                          │       (Go)            │
                          └──┬──────────┬────────┘
           ┌─────────────────┘          └──────────────────┐
           │                                               │
           ▼                                               ▼
 ┌──────────────────┐                         ┌─────────────────────────────┐
 │   PostgreSQL      │                        │   Kubernetes API Server      │
 │ 元数据 / 用量缓存  │                        │   MLJob / MLService / Tenant │
 │                   │                        │   Volcano Queue              │
 └──────────────────┘                         └─────────────┬───────────────┘
                                                            │ list-watch
                                                            ▼
                                                   ┌──────────────────┐
                                                   │  Informer Loop   │
                                                   │  状态回流         │
                                                   └──────────────────┘
```

读写模型：**API 只写 PG → reconciler 异步下发 CR → Informer 回流状态**。共享机制见 §5，各模块特有语义见 §6。

## 4. 代码布局

```
components/compute/
├── cmd/compute/              # 服务入口 main.go
├── api/
│   ├── openapi.yaml          # OpenAPI 3.0 契约源
│   └── types/                # oapi-codegen 生成的 request/response types + server stub
├── internal/
│   ├── server/               # HTTP router、middleware（身份解析、错误处理、访问日志、metrics）
│   ├── tenant/               # 租户管理
│   ├── resourcepool/         # 资源池管理
│   ├── resourceunit/         # 资源单元管理
│   ├── queue/                # 队列管理（Volcano Queue CR 容量下行 + 用量 informer）
│   ├── job/                  # 任务管理（MLJob informer）
│   ├── service/              # 在线服务管理（MLService informer）
│   ├── k8sclient/            # controller-runtime client + informer factory（各模块通过它共享 cache）
│   ├── db/                   # GORM 客户端 + golang-migrate 迁移
│   └── auth/                 # 从 X-Axisml-User header 解析调用方身份
└── pkg/                      # Compute 内部可复用工具（日志、错误、分页）
```

跨组件复用的公共库（如日志、错误、配置）放在仓库根 `pkg/`。

## 5. 运行机制

所有模块共享同一套"PG 为元数据权威，K8s 为运行状态权威；API 只写 PG，reconciler 与 Informer 完成双向同步"的写读模型。

> **核心不变式**：Reconciler 的工作集由且仅由三个 PG 谓词决定——`status='Creating' AND deleted_at IS NULL`（创建下发）、`status='Canceling'`（取消，Job 专属）、`status='Deleting'`（删除）。PG 行状态离开这三个谓词覆盖的范围后，reconciler 不再基于"CR 缺失"或任何外部信号做下发动作。其余状态转换由 Informer 回流推进，不由 reconciler 驱动。

### 5.1 权威划分

> **PG 为配额定义（`quota`）与业务元数据的权威；K8s / Volcano 为运行状态与配额用量（`used`）的权威。**

| 类别 | 权威方 | 流向 |
| --- | --- | --- |
| 队列配额定义 `quota`（capability / deserved / guarantee） | PG | API → PG → Volcano Queue `spec.capability` / `spec.deserved` / `spec.guarantee` |
| 队列实际用量 `used` | Volcano | Volcano Queue `status.allocated` → Informer → PG `used` 缓存 |
| 业务元数据（名称、引用、spec 快照） | PG | API → PG |
| 运行状态（phase、endpoint、副本就绪） | K8s | CR `status` → Informer → PG |

PG 的 `used` 只用于 UI 列表展示和 best-effort 预检，**不参与写入事务记账**。

### 5.2 写路径（Outbox + Reconciler）

采用 **Outbox 模式**：

1. **API 同步路径只写 PG**：业务校验 → PG 事务插入 / 更新业务记录（新建时 `status='Creating'`，取消时 `status='Canceling'`，删除时 `status='Deleting'` + `deleted_at=now()`）→ commit → 返回业务 ID。API 不直接调用 K8s
2. **Compute 内 reconciler worker 异步下发 CR**：每个模块（`internal/{job,service,tenant,queue}`）在 leader 副本起 goroutine，周期性扫描 PG 按三类谓词分派动作：
   - `status='Creating' AND deleted_at IS NULL` → 按 PG 快照 `Create()` CR（附 label `axisml.io/<resource>-id=<uuid>` 作稳定锚点；409 `AlreadyExists` 视为成功，靠 `metadata.name` + label 双重去重幂等）；Informer ADD 事件推进到就绪态（`Pending` / `Active`）
   - `status='Canceling'`（Job 专属） → reconciler `patch MLJob.spec.runPolicy.suspend=true`；mljob-operator Handler 完成 suspend / Cleanup 后写 `status.conditions[type=Suspended,status=True,reason=CancelRequested]`，Informer 观察到该 condition 推进 PG 到 `Cancelled` 并入队 `Delete()` CR 做资源回收（DELETE 事件幂等到达，不再变更 PG 状态。详见 [operators/mljob-operator.md §4](operators/mljob-operator.md)）
   - `status='Deleting'` → `Delete()` CR；Informer DELETE 事件推进到 `Deleted`（配合设置 `deleted_at`）
   - 失败按指数退避重试，错误写入业务记录的 `message` 字段供 UI 展示
   - PG 行状态离开这三个谓词覆盖的范围后，reconciler 不再做下发动作

**共享状态机骨架**（各资源完整状态集与转换见 §6.2 / §6.3）：

```
Creating ──(Informer ADD)──▶ 就绪态 ──(业务事件)──▶ 运行终态
                              │
      ┌───────────────────────┤
      │ cancel API (Job)      │ DELETE API (任一非 Canceling/Deleting/Deleted)
      ▼                       ▼
  Canceling                Deleting
      │                       │
      │ (CR 确认清理)          │ (CR 确认清理 + deleted_at)
      ▼                       ▼
  Cancelled               Deleted
```

- 就绪态：Tenant/Queue 为 `Active`；Job 为 `Pending`（继而 `Running`）；Service 为 `Pending`（继而 `Ready`/`Degraded`/`Failed`）
- 运行终态：Job 有 `Succeeded`/`Failed`；Service 无运行终态（`Failed` 可自愈）；Tenant/Queue 无运行终态
- `Cancelled` 为 Job 独有终态（资源释放但 PG 行保留供查阅，用户可后续 DELETE 进入 `Deleting → Deleted`）
- `Deleted` 为所有资源最终软删终态（`deleted_at` 非空，UI 默认不展示）

### 5.3 状态回流（Informer）

四条独立 Informer，分别由对应模块持有，通过 `k8sclient` 的 SharedInformerFactory 共享底层 cache：

| Informer | 监听对象 | 维护方 | 主要用途 | DELETE 事件语义 | spec 漂移处理 |
| --- | --- | --- | --- | --- | --- |
| MLJob Informer | `MLJob` CR | `internal/job/` | 推进 `Creating→Pending`、`Pending→Running`、运行终态（`Succeeded`/`Failed`）；`Canceling→Cancelled`（信号：`status.conditions[type=Suspended,status=True]`）；`Deleting→Deleted`（配合 `deleted_at`）；回写 `jobs.status` / `started_at` / `finished_at` | PG `status='Canceling'` → Suspended condition 推 `Cancelled` 后入队 `Delete()` CR；DELETE 事件幂等忽略。PG `status='Deleting'` → DELETE 推 `Deleted`。PG `status ∈ (Pending, Running)` 收到 DELETE → 外部误删，推 `Cancelled` + `message='external delete'`，**不补偿重建**；已终态忽略 | 不适用（`spec` 不可变） |
| MLService Informer | `MLService` CR | `internal/service/` | 推进 `Creating→Pending`、`Pending→Ready`、`Ready⇄Degraded⇄Failed`（由 `ready_replicas`/`desired_replicas` 映射，见 §6.3.2）；`Deleting→Deleted`；回写 `services.status` / `ready_replicas` / `endpoint` | PG `status='Deleting'` → 推 `Deleted`；其他运行态 → 外部误删，推 `Deleting` + `message='external delete'`，交 reconciler 完成清理；已 `Deleted` 忽略 | 扩缩容由 API 层同步 `patch` CR，Informer 只消费回流，Compute 不做 spec 漂移对账 |
| Tenant Informer | `Tenant` CR | `internal/tenant/` | 推进 `Creating→Active`、`Active⇄Suspended`、`Deleting→Deleted` | PG `status='Deleting'` → 推 `Deleted`；其他（外部误删）→ 按 §5.4 配置对象策略由 reconciler 补偿重建 | Tenant CR 声明字段漂移 → reconciler 按 PG 快照覆盖回 CR |
| Queue Informer | Volcano `Queue` CR | `internal/queue/` | 回写 `queues.used`（来自 `status.allocated`）；推进 `Creating→Active`、`Deleting→Deleted` | PG `status='Deleting'` → 推 `Deleted`；其他（外部误删）→ reconciler 按 PG 快照补偿重建 | `spec.capability` / `spec.deserved` / `spec.guarantee` 漂移 → reconciler 按 PG 快照覆盖回 CR |

通用模式：启动时 `List` 做差异 upsert 与孤儿检测；`Watch` 事件入各自 work queue；单 worker 串行 reconcile；以 `resourceVersion` / `generation` 作乐观并发字段保证幂等。PG 更新短暂失败时事件在 work queue 中重试至成功。

### 5.4 孤儿与补偿

补偿策略按资源类型分流。启动 `List` 时按 `axisml.io/<resource>-id` label 索引 CR 与 PG 行做对账：

**配置对象（Tenant、Queue）**

- **CR 漂移**：Informer 事件触发 Compute 按 PG 快照对齐——以 PG 为权威的字段（Queue 的 `spec.capability` / `spec.deserved` / `spec.guarantee`、Tenant 的声明字段）覆盖回 CR；CR 权威字段（`status.allocated` 等）不由 Compute 写回
- **正向孤儿**（PG 有行但无 CR）：reconciler 幂等 `Create()` 重建
- **反向孤儿**（CR 存在但 PG 无对应行或已软删）：默认删除 CR 并记录审计；高敏感场景可切为告警人工介入

**工作负载对象（Job、Service）**

- **CR 漂移**：Compute **不反向重建 CR**。Informer 观察到 CR DELETE 事件后按 PG 当前 `status` 分流：
  - `status ∈ (Canceling, Deleting)`：正常级联清理，分别推进 `Cancelled` / `Deleted`
  - Job 且 `status ∈ (Pending, Running)`：外部误删，推 `Cancelled` + `message='external delete'`，不补偿重建
  - Service 且 `status ∈ (Pending, Ready, Degraded, Failed)`：外部误删，推 `Deleting` + `message='external delete'`，交 reconciler 完成 `Deleting → Deleted` 流程，不补偿重建
  - 已终态（`Succeeded`/`Failed`/`Cancelled`/`Deleted`）：忽略
- **正向孤儿**（PG 有 `status='Creating'` 行但无 CR，且 `deleted_at IS NULL`）：这是 Outbox 正常窗口，reconciler 下一轮幂等重试 `Create()`。若 PG 行已推进到非 `Creating` 状态或已软删，reconciler **不回补创建**
- **反向孤儿**（CR 存在但 PG 无行或已 `Deleted`）：默认删除 CR 并记录审计

**关键判据**：Reconciler 工作集由三个谓词组成——`status='Creating' AND deleted_at IS NULL`（创建）、`status='Canceling'`（取消，Job 专属）、`status='Deleting'`（删除）。PG 行状态离开这三个谓词覆盖的范围后，reconciler 不再做下发动作。

### 5.5 副本与 Leader Election

Compute 默认 `replicas=1`（Standard 与 Lite 均同），但架构按"多副本可横向扩展"设计：

- **API 层无状态**：所有副本都服务 HTTP，仅写 PG，水平扩容无需额外协调
- **后台协程选主**：Reconciler goroutine 与各模块 Informer 只在 **leader 副本** 运行，通过 controller-runtime 的 leader election（K8s `coordination.k8s.io/Lease`）选主
- **Leader 切换行为**：lease duration 默认 15s；新 leader 启动 Informer 时先 `List` 做一次全量对账（复用 §5.4 的补偿路径）；原 leader 未完成的 `Creating` 行由新 leader 接续下发，依赖 `metadata.name` + label 双重去重保证幂等
- **单副本等价性**：`replicas=1` 时 leader election 退化为单成员 lease，瞬时获取，无额外延迟

## 6. 领域模型

每个模块按 **数据模型 → 模块特有语义** 组织。共享机制见 §5。

### 6.1 编排约定

本节列出所有模块共用的 schema 约定；具体表的外键关系在各表 schema 的字段注释（`FK tenants(id)` 等）中就近说明，不再集中列一份。

**通用字段**：所有表统一带 `id uuid`、`created_at`、`updated_at`、`deleted_at`（软删除）。

**UNIQUE 约束统一语义**：本节所有 schema 里标注的 `UNIQUE` 均实现为 PG partial unique index，`WHERE deleted_at IS NULL`——即软删行不占用唯一键，同名资源在原行被软删后可被再次创建。对应迁移示例：`CREATE UNIQUE INDEX ... ON tbl(col) WHERE deleted_at IS NULL`。

**`name` 字段 DNS-1123 硬校验**：所有承载业务标识并会映射到 K8s 对象名的 `name` 字段（Tenant、ResourcePool、Queue、Job、Service；ResourceUnit 在此之上还叠加 §6.2.3 的语义命名约定），API 层统一校验：字符集 `[a-z0-9-]`，首尾为字母或数字，长度 3–40，不允许连续 `--`。长度上限锁 40 是为了在最坏拼接场景下仍满足 K8s 对象名限制——例如 tenant-operator 的 per-tenant 资源前缀 `axisml-tenant-<tenant>-<sub>`（≤55 字符基础前缀 + 子名，DNS-1123 subdomain 上限 253）、Volcano Queue 名 `axisml-<tenant>-<pool>-<queue>`（≤129 字符，DNS-1123 subdomain 上限 253）。Tenant 自身的 namespace 名由 `Tenant.spec.namespace.name` 显式声明，不再约定固定格式（详见 [tenant-operator §3.1 / §6.1](operators/tenant-operator.md)）。需要更长或含大小写/空格的可读名请填 `display_name`。

**迁移**：由 `golang-migrate` embedded 方式在服务启动时执行，依赖 `schema_migrations` 表的 PG advisory lock 避免多副本并发迁移；生产可选通过 Helm `Job` 隔离。

### 6.2 配置对象

CR 缺失/漂移按 §5.4 配置对象策略补偿。

#### 6.2.1 租户（Tenant）

**数据模型**

```
tenants(
  id            uuid PK,                  -- 同时作为 Tenant CR label `axisml.io/tenant-id` 的稳定锚点
  name          text UNIQUE,              -- 作为 Tenant CR 的 metadata.name
  namespace     text,                     -- Tenant CR `spec.namespace.name` 的镜像；非唯一索引（多 Tenant 可共享同一 Namespace，详见 tenant-operator §7）
  display_name  text,
  status        text,                     -- Creating / Active / Suspended / Deleting / Deleted（Creating/Suspended/Deleting 由 API 写入；Active/Deleted 由 Informer 推进）
  message       text,                     -- 承载 Tenant CR `status.message` 的回流；可空。tenant-operator 写 phase=Failed 时由 Informer 写入，区分"配置出错"与"管理员暂停"
  annotations   jsonb,
  created_at    timestamptz,
  updated_at    timestamptz,
  deleted_at    timestamptz               -- 软删除，进入 Deleting 时写入
)
```

**状态机**

```
Creating ──(Informer ADD)──▶ Active ⇄ Suspended ──(DELETE req)──▶ Deleting ──(CR 确认清理)──▶ Deleted
                               │                                     ▲
                               └──(DELETE req)───────────────────────┘
```

- `Suspended` 语义：阻塞该租户下新 Job/Service 提交（API 在 Create 时校验 `tenant.status='Active'`）；已有任务保持运行；`Active ⇄ Suspended` 通过 `:suspend` / `:unsuspend` 动作端点
- **operator 侧 `Tenant.status.phase=Failed` 在 Compute 上等价于 `Suspended` + `message`**：tenant-operator 在校验失败 / 关键资源创建失败时写 `status.phase=Failed`（[tenant-operator §4](operators/tenant-operator.md)），Informer 把 `Failed` 收敛为 `tenants.status='Suspended'` 并把 `tenant.status.message` 写入 `tenants.message`——租户提交链路同样受阻，靠 `message` 区分"配置出错"与"管理员暂停"。Compute 不引入独立 `Failed` 终态

**生命周期**

| 操作 | PG | Kubernetes（reconciler 异步执行） |
| --- | --- | --- |
| 创建 | insert `tenants`，`status='Creating'` | 创建 cluster-scoped `Tenant` CR（label `axisml.io/tenant-id=<uuid>`），tenant-operator 按 `spec.namespace.name` 落地 Namespace（已存在则共享）、按 `spec.resourceQuota` 创建 per-tenant ResourceQuota（缺省=不限额，详见 [tenant-operator §6.1 / §6.2](operators/tenant-operator.md)）；Informer ADD 推 `Active` |
| 更新 | update `tenants` | patch Tenant CR |
| 挂起 | `status='Suspended'` | patch Tenant CR `spec.suspended=true`，tenant-operator 仅写 `status.phase=Suspended`、不停机底层资源；阻断新 Job/Service 提交由 Compute API 在 `tenant.status='Suspended'` 时拦截 |
| 恢复 | `status='Active'` | patch Tenant CR `spec.suspended=false` |
| 删除 | `status='Deleting'`，写 `deleted_at` | reconciler `Delete()` Tenant CR，K8s GC 通过 ownerReference 级联清理 per-tenant 资源（ResourceQuota / Secret / ConfigMap / SA / Role / RoleBinding）；**Namespace 不删除**（详见 [tenant-operator §6.1](operators/tenant-operator.md)）；Informer 观察 CR 消失 → `Deleted` |

**默认租户**：Helm `post-install` Job 幂等初始化 `default` 租户。

#### 6.2.2 资源池（ResourcePool）

**数据模型**

```
resource_pools(
  id             uuid PK,
  name           text UNIQUE,
  description    text,
  node_selector  jsonb,                   -- {"axisml.io/pool": "gpu-a100"}
  tolerations    jsonb,                   -- K8s Toleration 数组
  metadata       jsonb,                   -- 自由扩展字段
  created_at     timestamptz,
  updated_at     timestamptz,
  deleted_at     timestamptz
)
```

**职责分工**

- 管理员在集群侧给目标节点打标签 / 污染，Compute 不修改 Node 对象
- Compute 仅保存池的 `node_selector` / `tolerations`，生成 MLJob / MLService 时注入 `spec.placement`

**默认池**：Helm 初始化 `default` 池，`node_selector` 留空即表示整集群可用。

ResourcePool 无对应 CR，不走 Outbox 下发路径——它是纯 PG 元数据，仅在生成 MLJob/MLService 时作为注入源使用。

#### 6.2.3 资源单元（ResourceUnit）

**数据模型**

```
resource_units(
  id             uuid PK,
  pool_id        uuid FK resource_pools(id),
  name           text,
  description    text,
  requests       jsonb,                   -- {"cpu":"8","memory":"64Gi","nvidia.com/gpu":"1"}
  limits         jsonb,
  node_selector  jsonb,                   -- 通用节点标签匹配（见下）
  created_at     timestamptz,
  updated_at     timestamptz,
  deleted_at     timestamptz,
  UNIQUE(pool_id, name)
)
```

**字段说明**

- `requests` / `limits`：标准 K8s 资源，包括 `cpu`、`memory`、`nvidia.com/gpu`、其他 extended resources；数量一律在此表达
- `node_selector`：通用节点标签匹配，覆盖 GPU 之外的任意硬件维度

常见 `node_selector` 用法：

| 场景 | 示例 |
| --- | --- |
| GPU 型号 | `{"nvidia.com/gpu.product": "A100-SXM4-80GB"}`（GPU Operator 的 Node Feature Discovery 自动打标） |
| TPU | `{"cloud.google.com/gke-accelerator": "tpu-v4"}` |
| 自研加速卡 | `{"axisml.io/accelerator": "npu-x"}` |
| CPU instance type | `{"node.kubernetes.io/instance-type": "c5.4xlarge"}` |

**注入规则**（提交 MLJob / MLService 时）：

- `requests` / `limits` 由 Compute 传给 Operator（具体字段契约见 [operators/](operators/)）
- `placement.nodeSelector`：**Pool 优先合并**——`pool.node_selector` 的 key 全部保留，`resource_unit.node_selector` 只贡献 Pool 未声明的 key。等价 Go 伪码：

  ```go
  ns := maps.Clone(pool.NodeSelector)
  for k, v := range unit.NodeSelector {
      if _, exists := ns[k]; !exists {
          ns[k] = v
      }
  }
  ```
- `pool.tolerations` 作为 `placement.tolerations`

Pool 的硬件维度（如 `axisml.io/pool=gpu-a100`）一经设置即为池的"身份"，ResourceUnit 在同 key 上重新声明会被静默忽略；unit 只能补充 Pool 未涉及的维度（如 GPU 型号 / 内存档次 / 实例类型）。这样 Pool 更新不会把下辖 unit 推入"僵尸"状态，也省掉 API 层的级联冲突校验。

**命名约定**

格式 `<accelerator>[-<count>x]-<tier>[-<variant>]`：

- `<accelerator>`：加速卡或 CPU 标识（小写 kebab，如 `a100` / `h100` / `tpu-v4` / `ascend-910b` / `cpu`）
- `<count>x`：加速卡数量（如 `1x` / `2x` / `4x` / `8x`），CPU-only 省略
- `<tier>`：规格档 `small` / `medium` / `large` / `xlarge`
- `<variant>`：可选后缀，如 `himem` / `ssd` / `spot` / `ib`

示例：`cpu-small`、`a100-1x-large`、`h100-8x-xlarge-ib`、`tpu-v4-4x-large`。

字符集硬校验（DNS-1123 风格）：`[a-z0-9-]`，首尾字母/数字，长度 3–40，不允许连续 `--`。

ResourceUnit 同样无对应 CR，不走 Outbox 下发路径。

#### 6.2.4 资源队列（Queue）

**数据模型**

```
queues(
  id                  uuid PK,
  tenant_id           uuid FK tenants(id),
  pool_id             uuid FK resource_pools(id),
  name                text,                      -- "default" / "training" / ...
  quota               jsonb,                     -- 三维配额，见下
  status              text,                      -- Creating / Active / Deleting / Deleted
  used                jsonb,                     -- Informer 从 Volcano Queue status.allocated 缓存，只读不记账；反映所有走 Volcano 调度（schedulerName=volcano + PodGroup 关联）的 workload 用量：MLJob 的 (volcano, *) / (kubeflow-trainer, *) 路径自然入账（(native, job) 默认无 Volcano 依赖，不入账）；MLService 的 native 路径由 mlservice-operator 创建轻量 PodGroup（minMember=replicas，不要求 gang）入账（详见 operators/mlservice-operator.md §3.1）；(kserve, *) MLService 当前为已知缺口（见 mlservice §11 follow-up）
  created_at          timestamptz,
  updated_at          timestamptz,
  deleted_at          timestamptz,
  UNIQUE(tenant_id, pool_id, name)
)
```

`quota` 字段结构：

```json
{
  "capability": {"cpu": "100", "memory": "400Gi", "nvidia.com/gpu": "8"},
  "deserved":   {"cpu": "40",  "memory": "160Gi", "nvidia.com/gpu": "4"},
  "guarantee":  {"cpu": "20",  "memory": "80Gi",  "nvidia.com/gpu": "2"}
}
```

**三维配额语义**（对齐 Volcano capacity plugin）：

| 维度 | 含义 | 超限后果 |
| --- | --- | --- |
| `capability` | **硬上限**：队列占用资源的最大值 | 超过 → Volcano 拒绝调度，PodGroup 停在 `Pending` |
| `deserved` | **公平基线**：队列理应分到的份额；超出 `deserved` 但 ≤ `capability` 的部分属于"借用" | 借用部分可被其他队列未满足 `deserved` 的任务**抢占回收（reclaim）** |
| `guarantee` | **保留份额**：必须满足的底线 | 即便其他队列抢占，也不会被回收到 `guarantee` 以下 |

**校验约束**（每个资源 key 分别校验）：`guarantee ≤ deserved ≤ capability`，且均 ≥ 0。

**默认值策略**：API 仅强制要求 `capability`；`deserved`/`guarantee` 未填时默认为 0（等同"无公平基线、无保留份额"，所有队列平等争抢直到 `capability` 封顶——退化为 v1 行为）。

**状态机**

```
Creating ──(Informer ADD)──▶ Active ──(DELETE req)──▶ Deleting ──(CR 确认清理)──▶ Deleted
```

**模型特征**

- 扁平结构，每条队列都是独立对象，无父子层级
- 每个 `(tenant, pool)` 默认存在队列 `default`
- 用户可另行创建队列（如 `training`、`inference`、`nlp`）用于业务线 / 团队维度的配额拆分
- 同一租户在不同池下队列结构 / 配额互不干扰
- 分层队列作为后续演进方向（见 §11）

**与 Volcano 的映射**

- 每条 Compute Queue 1:1 对应一条 Volcano `Queue` CR（cluster-scoped）
- 命名约定：`axisml-<tenant>-<pool>-<queue>`，例：`axisml-default-gpu-a100-training`
- 配额下行：`quota.capability` → `queue.spec.capability`；`quota.deserved` → `queue.spec.deserved`；`quota.guarantee` → `queue.spec.guarantee`
- 用量上行：Informer 观察 `queue.status.allocated` → 缓存到 PG `used`

**配额预检（best-effort）**：提交任务 / 服务时：

1. 读 `quota.capability` 与缓存 `used`，若 `used + request > capability` 则早期拒绝并返回错误
2. 预检通过 → 创建 MLJob / MLService CR

硬约束由 Volcano 在调度准入层强制：`capability` 保证不会越额占用；`deserved` 与 `guarantee` 影响调度序关系与抢占行为，不做 Compute 层预检（Volcano 调度器按全局视图做决策，Compute 无足够上下文判断"当前是否会被抢占")。PG 不对 `queues.used` 加锁或事务记账。

### 6.3 工作负载对象

提交流程遵循 §5.2 Outbox，状态回流遵循 §5.3 Informer，CR 删除 / 孤儿处理遵循 §5.4 工作负载对象策略。

#### 6.3.1 任务（Job）

**数据模型**

```
jobs(
  id                   uuid PK,
  tenant_id            uuid FK tenants(id),
  pool_id              uuid FK resource_pools(id),
  queue_id             uuid FK queues(id),
  resource_unit_id     uuid FK resource_units(id),
  name                 text,                     -- MLJob CR metadata.name（在租户 namespace 内唯一）
  display_name         text,                     -- 用户可见名
  description          text,                     -- 用户备注
  owner_user           text,                     -- 来自 X-Axisml-User
  spec                 jsonb,                    -- 提交时的 MLJob.spec 完整快照（不可变）
  requested_resources  jsonb,                    -- 配额记账快照
  status               text,                     -- Creating / Pending / Running / Succeeded / Failed / Canceling / Cancelled / Deleting / Deleted（Creating/Canceling/Deleting 由 API 写入；其余由 reconciler / Informer 推进）
  message              text,                     -- CR 下发错误或状态附加信息（reconciler / Informer 回流）
  started_at           timestamptz,              -- Pod 首次运行（Informer 回流）
  finished_at          timestamptz,              -- 运行终态时间（Informer 回流，Succeeded/Failed/Cancelled）
  created_at           timestamptz,
  updated_at           timestamptz,
  deleted_at           timestamptz,              -- 进入 Deleting 时写入
  UNIQUE(tenant_id, name)                        -- 租户内 name 唯一；partial on deleted_at IS NULL
)
```

- `spec` 是提交时 MLJob.spec 的完整快照，包含 `roles[]` / `backend.{name,engine,config}` / `scheduling` / `runPolicy` 等业务字段（结构详见 [operators/mljob-operator.md §3.2](operators/mljob-operator.md)），不可变；Informer 回流只写 `status` 相关列
- `requested_resources` 冗余存提交时的资源申请，解耦后续 ResourceUnit 修改对已提交任务记账的影响
- `name` 是 Compute 与 K8s 的命名锚点（对应 MLJob CR `metadata.name`），UUID `id` 通过 label `axisml.io/job-id=<id>` 同步打到 CR 上，作为孤儿检测与跨重命名追踪的稳定锚点

**状态机**

```
Creating ──(Informer ADD)──▶ Pending ──(CR phase=Running)──▶ Running ──(CR phase=Succeeded)──▶ Succeeded
                                │                              │    ──(CR phase=Failed)────▶ Failed
                                │                              │
                                │  cancel API                  │  cancel API
                                └──────────────────────────────┴─▶ Canceling ──(Informer DELETE)──▶ Cancelled

任一非 Canceling/Deleting/Deleted 状态 ──(DELETE req)──▶ Deleting ──(CR 确认清理 + deleted_at)──▶ Deleted
```

- `Creating` 状态不接受 cancel（CR 未下发，API 拒绝并要求调用 DELETE 直接进入 `Deleting`）
- `Cancelled` PG 行保留（`deleted_at IS NULL`），用户可再次 DELETE → `Deleting` → `Deleted`
- `Succeeded`/`Failed`/`Cancelled` 三者为运行终态；`Deleted` 为软删终态

**提交校验**（API 同步路径，在 §5.2 通用 Outbox 流程之外附加）：

1. 从 `X-Axisml-User` 读取调用方身份（审计与 ownership 归属）
2. 业务校验：路径中的租户存在且激活；队列归属该租户；ResourceUnit 所属 pool 与队列所属 pool 一致
3. 配额预检（best-effort，见 §6.2.4）

**MLJob 业务语义**

Compute 只负责把以下业务语义装进 MLJob CR；具体 `spec` 字段结构、默认值、校验规则由 [operators/mljob-operator.md](operators/mljob-operator.md) 定义。

**`spec.backend` 默认值注入**：用户未指定 `spec.backend` 时，Compute 写 CR 时显式补 `{name: "native", engine: "job"}`；`backend.config` 默认空对象 `{}`。`backend.{name, engine}` 在 PG `jobs.spec` jsonb 中持久化，创建后不可变（Update API 拒绝修改这两个字段）。

**取消语义**（`POST /tenants/{tenant}/jobs/{job}:cancel`）

| 场景 | PG 侧 | CR 侧 |
| --- | --- | --- |
| 运行态（`Pending` / `Running`） cancel | `status='Canceling'`，写 `message='user cancelled'` | reconciler `patch MLJob.spec.runPolicy.suspend=true` → operator Handler 完成 suspend/Cleanup 后写 `status.conditions[type=Suspended,status=True,reason=CancelRequested]` → Informer 推 `Cancelled` 并入队 `Delete()` CR 做资源回收（DELETE 事件幂等忽略）|
| `Creating` 状态 cancel | API 拒绝（要求改用 DELETE） | — |
| 已终态（`Succeeded`/`Failed`/`Cancelled`/`Deleted` 或已在 `Canceling`/`Deleting`）cancel | API 返回无效操作 | — |

**删除语义**（`DELETE /tenants/{tenant}/jobs/{job}`）

| 场景 | PG 侧 | CR 侧 |
| --- | --- | --- |
| 任一非 `Deleting`/`Deleted` 状态 DELETE | `status='Deleting'`，写 `deleted_at` | reconciler `Delete()` CR；Informer DELETE → `Deleted` |
| `Creating` 状态 DELETE（CR 未下发） | 同上；reconciler 确认 CR 不存在 → 直接推 `Deleted` | — |
| `Deleting`/`Deleted` 再次 DELETE | 幂等，忽略 | — |

**自然事件**

| 场景 | PG 侧 | CR 侧 |
| --- | --- | --- |
| Job 自然结束 | Informer 回流 `Succeeded` / `Failed` | operator/GC 按策略清理 Pod；Compute 不主动删除 CR |
| 外部误删 CR（PG 在 `Pending`/`Running`） | Informer DELETE → `Cancelled` + `message='external delete'` | 不补偿重建 |

级联效果：CR 删除 → mljob-operator 清理 Pod → Volcano 释放 Queue 用量 → Queue Informer 回流刷新 `queues.used`。

#### 6.3.2 在线服务（Service）

**数据模型**

```
services(
  id                   uuid PK,
  tenant_id            uuid FK tenants(id),
  pool_id              uuid FK resource_pools(id),
  queue_id             uuid FK queues(id),
  resource_unit_id     uuid FK resource_units(id),
  name                 text,                     -- MLService CR metadata.name（在租户 namespace 内唯一）
  display_name         text,
  description          text,                     -- 用户备注
  owner_user           text,
  spec                 jsonb,                    -- 当前 MLService.spec 快照（扩缩容时同步更新 spec.roles[0].replicas，单 role 约定下即 services.replicas；多 role 独立扩缩见 operators/mlservice-operator.md §11）
  requested_resources  jsonb,                    -- 单副本资源申请快照（配额记账用）
  replicas             int,                      -- 冗余出来供配额记账（单 role 约定下 = spec.roles[0].replicas）
  ready_replicas       int,                      -- Informer 回流
  endpoint             text,                     -- 对外服务地址（Informer 回流）
  status               text,                     -- Creating / Pending / Ready / Degraded / Failed / Deleting / Deleted（Creating/Deleting 由 API 写入；其余由 Informer 推进）
  message              text,
  created_at           timestamptz,
  updated_at           timestamptz,
  deleted_at           timestamptz,              -- 进入 Deleting 时写入
  UNIQUE(tenant_id, name)                        -- 租户内 name 唯一；partial on deleted_at IS NULL
)
```

- `spec` 是当前 MLService.spec 的快照；与 jobs 不同，`spec` 非完全只读——扩缩容 API 更新 `replicas` 同时回写 `spec.roles[0].replicas`（单 role 约定），其他字段依然不可变
- 总配额占用 = `replicas × requested_resources`，由 Volcano 在实际调度时核算；Compute 在 API 层做 best-effort 预检

**状态机**

```
Creating ──(Informer ADD)──▶ Pending ──(ready=desired, desired>0)──▶ Ready ⇄ Degraded ──▶ Failed
                                                                      ▲                     │
                                                                      └─────── 自愈 ────────┘

任一非 Deleting/Deleted 状态 ──(DELETE req)──▶ Deleting ──(CR 确认清理 + deleted_at)──▶ Deleted
```

**status 映射规则**（由 Informer 从 MLService CR `status.readyReplicas` / `spec.roles[0].replicas` 推导，单 role 约定下 `services.replicas` 即 `spec.roles[0].replicas`，多 role 独立扩缩见 [operators/mlservice-operator.md §11](operators/mlservice-operator.md)）：

| 条件 | status |
| --- | --- |
| `desired_replicas == 0` | `Pending`（扩缩至 0，视为待调度 / 停用） |
| `ready_replicas == desired_replicas && desired_replicas > 0` | `Ready` |
| `0 < ready_replicas < desired_replicas` | `Degraded` |
| `ready_replicas == 0 && desired_replicas > 0` | `Failed` |

`Ready` / `Degraded` / `Failed` 均为**非终态**——operator 自愈（重建失败 Pod、健康检查恢复）后 Informer 回流可让 `Failed → Degraded → Ready` 自然恢复；只有 `Deleted` 是 Service 的最终终态。

**MLService 业务语义**

与 Job 类似（`roles[]` / `backend.{name,engine,config}` / `scheduling` / `runPolicy`），额外增加**模型引用**（`spec.modelRef`，指向 Artifacts 的 model version）与**对外路由**（`spec.route`，决定 `status.endpoint` 是内部 Service DNS 还是外部 URL）。具体契约由 [operators/mlservice-operator.md](operators/mlservice-operator.md) 定义。

**`spec.backend` 默认值注入**：用户未指定 `spec.backend` 时，Compute 写 CR 时显式补 `{name: "native", engine: "deployment"}`；`backend.config` 默认空对象 `{}`。`backend.{name, engine}` 在 PG `services.spec` jsonb 中持久化，创建后不可变。

**与 Job 的差异**

- **常驻**：配额占用不随运行状态释放，仅进入 `Deleted` 时释放
- **扩缩容同步 patch CR**：`POST .../services/{id}:scale` 在 API 层同步更新 `services.replicas` + `services.spec.roles[0].replicas`，并直接 `patch` MLService CR path `spec/roles/0/replicas`；失败由 API 返回错误。**不走 Reconciler 对账**（不引入 `desired_generation` 字段，保持 Outbox 模型精简）；配额按 `replicas × requested_resources` 线性记账
- **无 Cancel 语义**：常驻服务"下线"即"删除"，直接走 DELETE → `Deleting` → `Deleted`
- **`Failed` 非终态**：与 Job 不同，operator 可能自愈；只有 `Deleted` 为终态

**删除语义**（`DELETE /tenants/{tenant}/services/{service}`）

| 场景 | PG 侧 | CR 侧 |
| --- | --- | --- |
| 任一非 `Deleting`/`Deleted` 状态 DELETE | `status='Deleting'`，写 `deleted_at` | reconciler `Delete()` CR；Informer DELETE → `Deleted`；配额自然释放 |
| `Creating` 状态 DELETE（CR 未下发） | 同上；reconciler 确认 CR 不存在 → 直接推 `Deleted` | — |
| 外部误删 CR（运行态 `Pending`/`Ready`/`Degraded`/`Failed`） | Informer DELETE → `status='Deleting'` + `message='external delete'`，reconciler 走 `Deleting→Deleted` 流程 | 不补偿重建 |

## 7. API 设计

### 7.1 路径规划

Compute 所有 API 置于 `/api/v1` 前缀下，仅供 Platform 通过集群内 Service 调用，不配置外部 `HTTPRoute`。Platform 对外暴露的 API 路径由 Platform 自身定义，必要时再转发到 Compute 的内部 `/api/v1/...` 契约。

| 资源组 | 路径 | 主要动作 |
| --- | --- | --- |
| 健康检查 | `/healthz`、`/readyz` | Liveness / Readiness |
| 租户 | `/api/v1/tenants` | Create / Get / List / Update / Delete |
| 租户动作 | `/api/v1/tenants/{tenant}:suspend`、`:unsuspend` | `Active ⇄ Suspended` |
| 资源池 | `/api/v1/resource-pools` | CRUD（管理员） |
| 资源单元 | `/api/v1/resource-pools/{pool}/resource-units` | CRUD（管理员） |
| 队列 | `/api/v1/tenants/{tenant}/queues` | CRUD |
| 任务 | `/api/v1/tenants/{tenant}/jobs` | Create / Get / List / Delete |
| 任务副本 | `/api/v1/tenants/{tenant}/jobs/{job}/replicas` | List Pod 副本（编号 / pod 名 / phase / started_at） |
| 任务日志 | `/api/v1/tenants/{tenant}/jobs/{job}/logs` | 透传 kube-apiserver pod log；`follow=true` 时用 SSE，详见 §7.4 |
| 任务事件 | `/api/v1/tenants/{tenant}/jobs/{job}/events` | 聚合 MLJob / Pod / PodGroup 的 K8s Event |
| 任务动作 | `/api/v1/tenants/{tenant}/jobs/{job}:cancel` | 运行态 → `Canceling` |
| 服务 | `/api/v1/tenants/{tenant}/services` | Create / Get / List / Update / Delete |
| 服务动作 | `/api/v1/tenants/{tenant}/services/{service}:scale` | 调整 `replicas`（同步 patch CR） |

### 7.2 身份上下文

由 Platform 注入的请求头：

| Header | 含义 |
| --- | --- |
| `X-Axisml-User` | 调用方用户唯一 ID，用于审计和 ownership 归属 |

租户归属通过 URL 路径 `/tenants/{tenant}/...` 表达，不再经请求头传递。Compute 不做角色级鉴权——鉴权由 Platform 统一完成；Compute 只校验路径中的租户存在且激活，以及相关资源归属于该租户。

### 7.3 契约管理

`components/compute/api/openapi.yaml` 是唯一契约源，使用 `oapi-codegen` 生成：

- Compute 侧：Go types + server stub（`api/types/`）
- Platform 侧：Go client SDK（通过 Makefile target 再次生成）

### 7.4 任务日志透传

Compute 对 `GET /jobs/{job}/logs` 只做路径级鉴权与 Pod 定位，实际日志 IO 透传到 kube-apiserver 的 Pod Log API。

**查询参数**

| 参数 | 必填 | 语义 |
| --- | --- | --- |
| `replica` | 否 | 副本编号（0-based），通过 `axisml.io/replica-index` label 定位 Pod；仅副本身份天然稳定的 backend 支持（Indexed Job、StatefulSet 派生 Pod）。NonIndexed Job、裸 Pod 拓扑（如 `(volcano, podgroup)`）下该 label 不存在，应改用 `pod` 参数 |
| `pod` | 否 | 直接给出 Pod 名（绕过 label 索引）。可通过 `/jobs/{job}/replicas` 端点列出 Pod 名后选用。`replica` 与 `pod` 至少二选一；同时给出以 `pod` 为准 |
| `container` | 否 | 多容器 Pod 时指定容器名；默认取 mljob-operator 约定的主容器 |
| `follow` | 否 | `false`（默认）返回完整历史；`true` 进入流式模式 |
| `tail_lines` | 否 | 只返回末尾 N 行；与 `follow` 可叠加 |
| `since_time` | 否 | RFC3339 起始时间 |
| `previous` | 否 | `true` 时查上一次容器实例的日志（用于 crash 后排查退出原因） |

**寻址路径选择**：副本身份稳定的场景（Indexed Job、StatefulSet）建议直接用 `replica`；其他场景先调 `/jobs/{job}/replicas` 列出 pod 名再用 `pod` 参数定位。两者均缺失时返 400。

**流式（`follow=true`）**：Compute 响应 `Content-Type: text/event-stream`，把 kube-apiserver 返回的 chunked log 逐行封装为 SSE `data:` 事件向下游推送；客户端断开时主动关闭 upstream watch。非流式模式下直接透传 `text/plain`。

**鉴权与校验**：

- 校验路径中的 `tenant` 归属 `job`；不做角色校验（§7.2 原则）
- Pod 已被回收（Job 进入终态 + GC）时返 410 Gone，`message` 建议调用方改用外部日志系统（out-of-scope）

**out-of-scope**：Compute 不做多副本日志聚合、不做日志持久化 / 搜索；持久化与跨 Pod 检索由外部 logging pipeline 承担（未来通过另外的接口或跳转链接暴露）。

## 8. 外部系统协作

| 对象 | 交互方式 | 关注点 |
| --- | --- | --- |
| Kubernetes API | controller-runtime `client.Client` + informer | RBAC：MLJob / MLService / Tenant 全部 verbs；Volcano Queue `create/update/delete/get/list/watch` |
| Volcano | Queue CRD 同步（见 §6.2.4、§5） | 不接管 Pod 调度逻辑，仅通过 `schedulerName: volcano` + queue 让 Volcano 接管 |
| PostgreSQL | GORM；与其他服务共用 database `axisml`（按表名前缀或 schema 逻辑隔离） | 迁移随二进制打包，启动时执行（golang-migrate） |
| Platform | REST，请求头透传身份 | Compute 信任内部调用；mTLS / Compute 主动鉴权列为未来规划 |
| Artifacts | HTTP 客户端 | 懒查询：提交任务时校验 image / model / dataset 引用存在 |

## 9. 部署形态

### 9.1 镜像与容器

- 镜像：`axisml/compute:<appVersion>`（由 `build/docker/compute.Dockerfile` 构建）
- 端口：`8081/tcp`（REST）
- 启动命令：`/compute serve`
- 探针：`GET /healthz`、`GET /readyz`

### 9.2 Helm 模板

补充 `deploy/helm/axisml-system/templates/compute/` 下模板：

| 文件 | 用途 |
| --- | --- |
| `configmap.yaml`（已存在，补字段） | DB 连接、日志级别 |
| `deployment.yaml`（已存在，换镜像） | 将 nginx placeholder 替换为真实镜像，加探针 |
| `service.yaml`（已存在） | 保持 ClusterIP 8081 |
| `serviceaccount.yaml`（新增） | Compute 服务账号 |
| `rbac.yaml`（新增） | ClusterRole + ClusterRoleBinding：MLJob / MLService / Tenant / Volcano Queue |
| `servicemonitor.yaml`（新增） | `/metrics` 暴露，kube-prometheus-stack 自动发现 |

### 9.3 对外暴露

Compute 不直接暴露到集群外，也不在 `axisml-infra` chart 中配置 `compute-api` `HTTPRoute`。实际调用方只有 Platform，Platform 通过集群内 Service DNS 访问 Compute。

### 9.4 引导数据

Helm `post-install` Job 初始化（所有操作幂等，升级 Helm 版本不会重复创建）：

- 默认租户 `default`
- 默认资源池 `default`
- 默认资源单元（例如 `cpu-small` / `cpu-medium`；GPU 相关单元按实际节点打标后再创建）
- `(default, default)` 下的默认队列 `default`：`quota.capability` 按集群可用资源估算；`quota.deserved` / `quota.guarantee` 均取 0（单队列场景下抢占与保留不起作用）

### 9.5 副本与可用性

- **默认形态**：`replicas=1`（Standard 与 Lite 均同）；`strategy: RollingUpdate`，新老 pod 交替期间短暂并存由 leader election 自然处理
- **多副本**：修改 Helm values 即可横向扩容；API 层线性扩展，Reconciler/Informer 走 leader election（详见 §5.5）
- **探针**：`/healthz` 返回进程存活；`/readyz` 额外校验 PG 连通性（Informer 就绪状态不计入 readiness，避免 leader 切换期间非 leader 副本被摘流量）
- **Leader 身份暴露**：通过 `/metrics` 暴露 `axisml_compute_is_leader` gauge（0/1），便于告警与排障

### 9.6 关键 metrics

`servicemonitor.yaml` 暴露 `/metrics`（Prometheus 格式）。至少包含以下业务指标（除标准 Go runtime / HTTP / controller-runtime 指标外）：

| 指标 | 类型 | 含义 | 告警建议 |
| --- | --- | --- | --- |
| `axisml_compute_is_leader` | gauge | 当前副本是否为 leader（0/1） | 多副本时 `sum == 1` 否则异常 |
| `axisml_compute_reconciler_oldest_pending_seconds{resource,predicate}` | gauge | 三谓词工作集中最老未处理行的 age，按资源（job/service/tenant/queue）与谓词（creating/canceling/deleting）拆分 | 持续 > 60s → reconciler 滞后 |
| `axisml_compute_reconciler_actions_total{resource,predicate,result}` | counter | reconciler 动作计数（成功 / 重试 / 失败） | 失败率或重试率突增 |
| `axisml_compute_informer_workqueue_depth{resource}` | gauge | 各模块 Informer work queue 当前深度 | 持续 > 阈值 → 回流滞后 |
| `axisml_compute_cr_drift_repair_total{resource,kind}` | counter | CR 漂移修复次数（kind：`missing`/`spec_mismatch`/`orphan_delete`） | 非零即值得看 |
| `axisml_compute_quota_precheck_rejected_total{tenant,queue}` | counter | 配额预检拒绝次数 | 突增 → 容量规划问题 |
| `axisml_compute_api_request_duration_seconds{route,status}` | histogram | API 请求延迟分布 | 常规 SLO 告警 |

## 10. 关键设计决策

| 决策项 | 决策 | 理由 |
| --- | --- | --- |
| 系统管理归属 | 留在 Compute | 与计算任务提交路径强耦合（配额校验、资源引用），避免跨服务调用开销；内部按 package 隔离保留未来拆分空间 |
| 队列模型 | 扁平结构（v1 无父子层级）；每个 `(tenant, pool)` 默认 `default`；1:1 映射到 Volcano Queue | 租户已是强隔离边界，业务线拆分用同级多队列即可；分层带来的 schema / 链式配额 / 孤儿补偿复杂度放到后续按需引入 |
| 配额记账 | PG 不记账，Volcano 为实际用量权威；Compute 仅做 best-effort 预检 | 避免 PG / K8s 双源冲突；Volcano 调度准入天然具备强约束 |
| 队列配额建模 | `quota` 采用 Volcano capacity plugin 的三维模型：`capability`（硬上限）/ `deserved`（公平基线）/ `guarantee`（保留份额）；API 必填 `capability`，另两项默认 0 | 单填 `capability` 时退化为"平等抢占直到封顶"的 v1 行为；多队列场景下 `deserved` 提供公平基线与抢占回收语义、`guarantee` 兜底关键队列；与 Volcano 原生调度语义直接对齐，避免自研配额仲裁 |
| MLJob spec 粒度 | 声明式高阶抽象（`backend.{name,engine,config}` / `roles[]` / `scheduling` / `runPolicy` 等） | 隔离 K8s 细节变更影响；operator 可独立演进 Pod 模板与 backend Handler |
| 状态同步 | Informer + PG 落库 | K8s 原生；支持大列表与按状态筛选；与 controller-runtime 工具链吻合 |
| 认证鉴权 | Platform 统一认证与鉴权；Compute 仅通过 `X-Axisml-User` 记录调用方做审计 / ownership | 避免职责重复；外部入口收敛到 Platform；租户归属通过 URL 路径自然表达 |
| ResourceUnit 节点匹配 | 通用 `node_selector` 而非 `gpu` 字段 | 覆盖 GPU / TPU / 自研加速卡 / CPU instance type 等任意硬件维度，对未来硬件可扩展 |
| ResourcePool 纳管 | `labelSelector` + `tolerations` | K8s 原生；运维友好；多云节点标签可复用 |
| API 协议 | REST / JSON + OpenAPI 3.0 | 可生成 Platform 端 SDK；内部 HTTP 调用简单直接；工具链成熟 |
| 写路径一致性 | Outbox 模式：API 单写 PG (`Creating`) + Compute 内 reconciler worker 幂等下发 CR + Informer 推进状态机 | 规避 PG commit 与 K8s `Create()` 之间的非原子窗口；任意单点崩溃都可恢复，无孤儿对象 |
| 业务状态机 | 显式 `Creating` 起点：API 写入；Informer ADD 推 `Pending`；终态由 CR `phase` 决定 | `Creating` 可区分"已登记未下发"与"已下发"，让正向孤儿可识别可重试 |
| 状态建模 | 为每个承载 CR 的资源定义独立状态机；统一引入 `Deleting/Deleted` 作为删除中间/终态，Job 额外引入 `Canceling/Cancelled`；Reconciler 工作集由三谓词组成（`Creating` / `Canceling` / `Deleting`） | CR 清理非瞬时，动作中间态显式化后 Reconciler 谓词与外部误删判定都能一一对应；`Cancelled` 与 `Deleted` 分离保留用户"已取消但可查阅"的体验 |
| HA / 副本 | 默认 `replicas=1`；API 层无状态可横向扩展；Reconciler 与 Informer 通过 K8s Lease（controller-runtime leader election）选主 | Compute 是控制平面、非性能热点；leader election 零成本保留未来横向扩容能力，无需改代码 |
| Reconciler 实现 | 每模块在 **leader 副本** 内起 goroutine 轮询 PG，复用 k8sclient | 复用现有模块结构；避免新增独立 controller 二进制；与 Informer 共用 work queue 顺序自然 |
| CR 稳定锚点 | `metadata.name` + label `axisml.io/<resource>-id=<uuid>` | name 因软删可能重用（UNIQUE 为 partial index，`WHERE deleted_at IS NULL`），UUID 永久唯一；孤儿检测按 label 索引而非 name |
| 补偿策略分流 | 配置对象按 PG 快照补偿重建；工作负载对象的 CR 缺失视为生命周期终点，不重建 | 避免把用户取消、operator 级联清理误判为孤儿；`status='Creating' AND deleted_at IS NULL` 是唯一触发重建的判据 |

## 11. 未来规划

- **分层队列**：`queues` 表新增 `parent_id`，利用 Volcano Capacity plugin 的 `spec.parent` 能力实现父子配额与抢占；在有真实多团队/多业务线共享租户的诉求时引入
- **数据卷管理**：纳入 Compute（`components/compute/internal/volume/`），v1 未实现；待补 schema（按租户隔离的 volume 定义与挂载声明）、底层存储映射（PVC / NFS / S3 / hostPath 等）、operator 注入契约（经 MLJob/MLService `spec.volumes` 下发）
- **多集群**：队列 / 任务按集群维度扩展，联邦调度
- **细粒度配额**：GPU 时长、存储容量、网络带宽等
- **审计日志**：独立 `audit_events` 表，关键写操作落库
- **成本核算**：基于 `jobs` / `services` 的实际使用时长 × 资源单元成本的计费导出
