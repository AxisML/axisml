# AxisML Compute 详细设计

## 1. 概述

AxisML Compute 是平台的计算服务层，基于 Go 开发，承载 **计算任务管理** 与 **系统管理（租户 / 资源池 / 资源单元 / 队列）** 两大职责。Compute 通过 REST API 暴露能力，仅接受来自 AxisML Platform 的内部调用，不直接对外部用户流量开放。

**关键边界原则**：Compute 不直接创建 Namespace、Pod 等底层 K8s 资源，这些由对应 Operator 或集群管理员负责；Compute 仅维护业务元数据，并通过 CRD 向 Operator 声明意图。

## 2. 职责与边界

Compute 内部按 6 个模块划分，分两类组织：

| 类别 | 模块 | 职责 | 边界外 |
| --- | --- | --- | --- |
| 配置对象（长生命周期） | 租户 Tenant | 元数据 CRUD，下发 `Tenant` CRD | Namespace 创建、ResourceQuota 落地（由 tenant-operator） |
| | 资源池 ResourcePool | `labelSelector` + `tolerations` 描述的池元数据 | Node 自身打标 / 污染（由集群管理员） |
| | 资源单元 ResourceUnit | 池内资源规格模板（`requests` / `limits` + 节点标签匹配） | 实际调度（Volcano） |
| | 队列 Queue | 扁平队列 `capacity` 元数据 + Volcano Queue CR 同步 + 用量回流 | 实际公平调度 / 抢占 / 用量核算（Volcano） |
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

### 5.1 权威划分

> **PG 为配额定义（`capacity`）与业务元数据的权威；K8s / Volcano 为运行状态与配额用量（`used`）的权威。**

| 类别 | 权威方 | 流向 |
| --- | --- | --- |
| 队列配额定义 `capacity` | PG | API → PG → Volcano Queue `spec.capability` |
| 队列实际用量 `used` | Volcano | Volcano Queue `status.allocated` → Informer → PG `used` 缓存 |
| 业务元数据（名称、引用、spec 快照） | PG | API → PG |
| 运行状态（phase、endpoint、副本就绪） | K8s | CR `status` → Informer → PG |

PG 的 `used` 只用于 UI 列表展示和 best-effort 预检，**不参与写入事务记账**。

### 5.2 写路径（Outbox + Reconciler）

采用 **Outbox 模式**：

1. **API 同步路径只写 PG**：业务校验 → PG 事务插入 / 更新业务记录（新建时 `status='Creating'`）→ commit → 返回业务 ID。API 不直接调用 K8s
2. **Compute 内 reconciler worker 异步下发 CR**：每个模块（`internal/{job,service,tenant,queue}`）起 goroutine：
   - 周期性 `SELECT ... WHERE status='Creating' AND deleted_at IS NULL`
   - 按 PG 快照 Create/Apply 对应 CR，附 label `axisml.io/<resource>-id=<uuid>` 作稳定锚点
   - 收到 409 `AlreadyExists` 视为成功（幂等：`metadata.name` + label 双重去重）
   - 创建成功后**不直接更新 status**——交给 Informer 收到 ADD 事件时把 `Creating` 推到 `Pending` / `Active`
   - 失败按指数退避重试，错误写入业务记录的 `message` 字段供 UI 展示

**状态机**（jobs / services 共享；tenants / queues 简化为 `Creating → Active`）：

```
Creating ─ Informer ADD ──────────────▶ Pending
Pending  ─ CR phase=Running ──────────▶ Running
Running  ─ CR phase=Succeeded ────────▶ Succeeded (终态)
Running  ─ CR phase=Failed   ────────▶ Failed    (终态)
*        ─ User cancel / CR delete ──▶ Cancelled (终态)
```

### 5.3 状态回流（Informer）

四条独立 Informer，分别由对应模块持有，通过 `k8sclient` 的 SharedInformerFactory 共享底层 cache：

| Informer | 监听对象 | 维护方 | 主要用途 | CR DELETE 策略 |
| --- | --- | --- | --- | --- |
| MLJob Informer | `MLJob` CR | `internal/job/` | 推进 `Creating → Pending`；回写 `jobs.status` / `started_at` / `finished_at` | 非终态 PG 行 → `Cancelled`，**不重建** |
| MLService Informer | `MLService` CR | `internal/service/` | 推进 `Creating → Pending`；回写 `services.status` / `ready_replicas` / `endpoint` | 非终态 PG 行 → `Cancelled`，**不重建** |
| Tenant Informer | `Tenant` CR | `internal/tenant/` | 推进 `Creating → Active` | 按 PG 快照**补偿重建**（长生命周期配置） |
| Queue Informer | Volcano `Queue` CR | `internal/queue/` | 回写 `queues.used`（来自 `status.allocated`） | 按 PG 快照**补偿重建**；`spec.capability` 漂移回覆 |

通用模式：启动时 `List` 做差异 upsert 与孤儿检测；`Watch` 事件入各自 work queue；单 worker 串行 reconcile；以 `resourceVersion` / `generation` 作乐观并发字段保证幂等。PG 更新短暂失败时事件在 work queue 中重试至成功。

### 5.4 孤儿与补偿

补偿策略按资源类型分流。启动 `List` 时按 `axisml.io/<resource>-id` label 索引 CR 与 PG 行做对账：

**配置对象（Tenant、Queue）**

- **CR 漂移**：Informer 事件触发 Compute 按 PG 快照对齐——以 PG 为权威的字段（Queue 的 `spec.capability`、Tenant 的声明字段）覆盖回 CR；CR 权威字段（`status.allocated` 等）不由 Compute 写回
- **正向孤儿**（PG 有行但无 CR）：reconciler 幂等 `Create()` 重建
- **反向孤儿**（CR 存在但 PG 无对应行或已软删）：默认删除 CR 并记录审计；高敏感场景可切为告警人工介入

**工作负载对象（Job、Service）**

- **CR 漂移**：Compute **不反向重建 CR**。Informer 观察到非终态 Job/Service 的 CR DELETE 事件 → 按删除语义推进 PG 到终态（Job: `Cancelled`；Service: `Cancelled`），写入 `message` 标注来源（用户取消 / 外部删除 / operator 级联），不再触发重建
- **正向孤儿**（PG 有 `status='Creating'` 行但无 CR，且 `deleted_at IS NULL`）：这是 Outbox 正常窗口，reconciler 下一轮幂等重试 `Create()`。若 PG 行已推进到非 `Creating` 状态（Pending/Running/终态）或已软删，reconciler **不回补创建**
- **反向孤儿**（CR 存在但 PG 无行或已软删）：默认删除 CR 并记录审计（用户发起的软删需要 reconciler 级联删除 CR，这里是兜底清理）

**关键判据**：`status='Creating' AND deleted_at IS NULL` 是唯一触发"缺失→重建"的条件。PG 行离开 `Creating` 或 `deleted_at` 非空后，reconciler 不再基于"CR 缺失"做创建动作。

## 6. 领域模型

每个模块按 **数据模型 → 模块特有语义** 组织。共享机制见 §5。

### 6.1 表关联总览

关键外键关系：

- `resource_units.pool_id` → `resource_pools.id`
- `queues.tenant_id` → `tenants.id`；`queues.pool_id` → `resource_pools.id`
- `jobs.{tenant_id, pool_id, queue_id, resource_unit_id}` → 对应表
- `services.{tenant_id, pool_id, queue_id, resource_unit_id}` → 对应表

所有表统一带 `id uuid`、`created_at`、`updated_at`、`deleted_at`（软删除）。迁移由 `golang-migrate` embedded 方式在服务启动时执行，生产可选通过 Helm `Job` 隔离。

### 6.2 配置对象

CR 缺失/漂移按 §5.4 配置对象策略补偿。

#### 6.2.1 租户（Tenant）

**数据模型**

```
tenants(
  id            uuid PK,                  -- 同时作为 Tenant CR label `axisml.io/tenant-id` 的稳定锚点
  name          text UNIQUE,              -- 作为 Tenant CR 的 metadata.name
  namespace     text UNIQUE,              -- 租户命名空间，约定 `axisml-tenant-<name>`
  display_name  text,
  status        text,                     -- Creating / Active / Suspended（Creating 由 API 写入，其余由 Informer 回流）
  annotations   jsonb,
  created_at    timestamptz,
  updated_at    timestamptz,
  deleted_at    timestamptz               -- 软删除
)
```

**生命周期**

| 操作 | PG | Kubernetes（reconciler 异步执行） |
| --- | --- | --- |
| 创建 | insert `tenants`，`status='Creating'` | 创建 cluster-scoped `Tenant` CR（label `axisml.io/tenant-id=<uuid>`），tenant-operator 建 Namespace `axisml-tenant-<name>` 及默认 ResourceQuota |
| 更新 | update `tenants` | patch Tenant CR |
| 删除 | 软删除（填 `deleted_at`） | 删除 Tenant CR，由 operator 级联清理 |

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

**冲突校验**：创建 / 更新 ResourceUnit 时，若 Pool 的 `node_selector` 与 unit 的 `node_selector` 在同一 key 上值冲突，Compute 拒绝；key 不相交则合并。

**注入规则**（提交 MLJob / MLService 时）：

- `requests` / `limits` 由 Compute 传给 Operator（具体字段契约见 [operators/](operators/)）
- `placement.nodeSelector = pool.node_selector ∪ resource_unit.node_selector`
- `pool.tolerations` 作为 `placement.tolerations`

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
  capacity            jsonb,                     -- {"cpu":"100","memory":"400Gi","nvidia.com/gpu":"8"}
  used                jsonb,                     -- Informer 从 Volcano Queue status 缓存，只读不记账
  created_at          timestamptz,
  updated_at          timestamptz,
  deleted_at          timestamptz,
  UNIQUE(tenant_id, pool_id, name)
)
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
- 容量下行：`capacity` → `queue.spec.capability`
- 用量上行：Informer 观察 `queue.status.allocated` → 缓存到 PG `used`

**配额预检（best-effort）**：提交任务 / 服务时：

1. 读 `capacity` 与缓存 `used`，若 `used + request > capacity` 则早期拒绝并返回错误
2. 预检通过 → 创建 MLJob / MLService CR

硬约束由 Volcano 在调度准入层强制：PG 缓存陈旧时即便 Compute 放行越额任务，Volcano 也会让对应 PodGroup 停在 `Pending`。PG 不对 `queues.used` 加锁或事务记账。

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
  name                 text UNIQUE,              -- MLJob CR metadata.name
  display_name         text,                     -- 用户可见名
  description          text,                     -- 用户备注
  owner_user           text,                     -- 来自 X-Axisml-User
  spec                 jsonb,                    -- 提交时的 MLJob.spec 完整快照（不可变）
  requested_resources  jsonb,                    -- 配额记账快照
  status               text,                     -- Creating / Pending / Running / Succeeded / Failed / Cancelled（Creating 由 API 写入；其余由 Informer 回流）
  message              text,                     -- CR 下发错误或状态附加信息（reconciler / Informer 回流）
  started_at           timestamptz,              -- Pod 首次运行（Informer 回流）
  finished_at          timestamptz,              -- 终态时间（Informer 回流）
  created_at           timestamptz,
  updated_at           timestamptz,
  deleted_at           timestamptz
)
```

- `spec` 是提交时 MLJob.spec 的完整快照，包含 framework / image / command / replicas 等业务字段，不可变；Informer 回流只写 `status` 相关列
- `requested_resources` 冗余存提交时的资源申请，解耦后续 ResourceUnit 修改对已提交任务记账的影响
- `name` 是 Compute 与 K8s 的命名锚点（对应 MLJob CR `metadata.name`），UUID `id` 通过 label `axisml.io/job-id=<id>` 同步打到 CR 上，作为孤儿检测与跨重命名追踪的稳定锚点

**提交校验**（API 同步路径，在 §5.2 通用 Outbox 流程之外附加）：

1. 从 `X-Axisml-User` 读取调用方身份（审计与 ownership 归属）
2. 业务校验：路径中的租户存在且激活；队列归属该租户；ResourceUnit 所属 pool 与队列所属 pool 一致
3. 配额预检（best-effort，见 §6.2.4）

**MLJob 业务语义**

Compute 只负责把以下业务语义装进 MLJob CR；具体 `spec` 字段结构、默认值、校验规则由 [operators/mljob-operator.md](operators/mljob-operator.md) 定义。

**停止 / 删除语义**

Job 是生命周期对象，删除只沿下列路径发生（通用补偿策略见 §5.4）：

| 场景 | PG 侧 | CR 侧 |
| --- | --- | --- |
| 用户取消（运行中） | `status` → `Cancelled`（终态），写 `message='user cancelled'` | reconciler 感知到 PG 进入 `Cancelled` → 优先 patch `MLJob.spec.suspend=true`（若 operator 支持）否则 `Delete()` CR；mljob-operator 级联清理 Pod |
| 用户取消（尚未下发，PG 仍 `Creating`） | `status` → `Cancelled`，写 `deleted_at` | reconciler 跳过该行（不再试图创建） |
| 用户删除（已终态或取消） | 软删 `deleted_at` | reconciler `Delete()` CR（若仍存在），幂等 |
| 外部误删 CR | Informer DELETE → `Cancelled`（若未终态），写 `message` 标注来源 | 不补偿重建 |
| Job 自然结束 | Informer 回流 `Succeeded` / `Failed` | operator/GC 按策略清理 Pod；Compute 不主动删除 CR |

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
  name                 text UNIQUE,              -- MLService CR metadata.name
  display_name         text,
  description          text,                     -- 用户备注
  owner_user           text,
  spec                 jsonb,                    -- 当前 MLService.spec 快照（扩缩容时同步更新 spec.replicas）
  requested_resources  jsonb,                    -- 单副本资源申请快照（配额记账用）
  replicas             int,                      -- 冗余出来供配额记账（= spec.replicas）
  ready_replicas       int,                      -- Informer 回流
  endpoint             text,                     -- 对外服务地址（Informer 回流）
  status               text,                     -- Creating / Pending / Ready / Degraded / Failed（Creating 由 API 写入；其余由 Informer 回流）
  message              text,
  created_at           timestamptz,
  updated_at           timestamptz,
  deleted_at           timestamptz
)
```

- `spec` 是当前 MLService.spec 的快照；与 jobs 不同，`spec` 非完全只读——扩缩容 API 更新 `replicas` 同时回写 `spec.replicas`，其他字段依然不可变
- 总配额占用 = `replicas × requested_resources`，由 Volcano 在实际调度时核算；Compute 在 API 层做 best-effort 预检

**MLService 业务语义**

与 Job 类似（framework / image / replicas / resourceUnit / placement / queue），额外增加**模型引用**（指向 Catalog 的 model version）。具体契约由 [operators/mlservice-operator.md](operators/mlservice-operator.md) 定义。

**与 Job 的差异**

- **常驻**：配额占用不随运行状态释放，仅删除时释放
- **扩缩容**：API 更新 `services.replicas` 与 `spec.replicas`（同步路径无新 CR）→ reconciler 检测到 spec 漂移后 patch 现有 MLService CR → mlservice-operator 调整底层 Deployment；配额按 `replicas × requested_resources` 线性记账
- **终态定义**：`Failed` 不是 Service 的终态（operator 可能自愈），只有 `Cancelled` 和显式软删是终态；Job 的 `Failed` 是终态

**停止 / 删除语义**

| 场景 | PG 侧 | CR 侧 |
| --- | --- | --- |
| 用户下线（运行中） | `status` → `Cancelled`（终态），写 `deleted_at` 与 `message='user cancelled'` | reconciler 按软删标记 `Delete()` CR；mlservice-operator 清理 Deployment |
| 用户下线（尚未下发，PG 仍 `Creating`） | `status` → `Cancelled`，写 `deleted_at` | reconciler 跳过该行（不再补发 Create） |
| 外部误删 CR | Informer DELETE → `Cancelled`（若未终态），写 `message` 标注来源 | 不补偿重建；配额由 Volcano 在 Deployment 清理后自然释放 |
| Degraded → Failed（健康检查持续失败） | Informer 回流 `Failed`（非终态，可恢复） | 保持 CR 由 operator 自愈，不由 Compute 删除 |

## 7. API 设计

### 7.1 路径规划

Compute 所有 API 置于 `/api/v1` 前缀下，仅供 Platform 通过集群内 Service 调用，不配置外部 `HTTPRoute`。Platform 对外暴露的 API 路径由 Platform 自身定义，必要时再转发到 Compute 的内部 `/api/v1/...` 契约。

| 资源组 | 路径 | 主要动作 |
| --- | --- | --- |
| 健康检查 | `/healthz`、`/readyz` | Liveness / Readiness |
| 租户 | `/api/v1/tenants` | CRUD |
| 资源池 | `/api/v1/resource-pools` | CRUD（管理员） |
| 资源单元 | `/api/v1/resource-pools/{pool}/resource-units` | CRUD（管理员） |
| 队列 | `/api/v1/tenants/{tenant}/queues` | CRUD |
| 任务 | `/api/v1/tenants/{tenant}/jobs` | Create / Get / List / Delete / Logs（透传 kube-apiserver）/ Events |
| 服务 | `/api/v1/tenants/{tenant}/services` | Create / Get / List / Update / Delete / Scale |

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

## 8. 外部系统协作

| 对象 | 交互方式 | 关注点 |
| --- | --- | --- |
| Kubernetes API | controller-runtime `client.Client` + informer | RBAC：MLJob / MLService / Tenant 全部 verbs；Volcano Queue `create/update/delete/get/list/watch` |
| Volcano | Queue CRD 同步（见 §6.2.4、§5） | 不接管 Pod 调度逻辑，仅通过 `schedulerName: volcano` + queue 让 Volcano 接管 |
| PostgreSQL | GORM；与其他服务共用 database `axisml`（按表名前缀或 schema 逻辑隔离） | 迁移随二进制打包，启动时执行（golang-migrate） |
| Platform | REST，请求头透传身份 | Compute 信任内部调用；mTLS / Compute 主动鉴权列为未来规划 |
| Catalog | HTTP 客户端 | 懒查询：提交任务时校验 image / model / dataset 引用存在 |

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
- `(default, default)` 下的默认队列 `default`，容量按集群可用资源估算

## 10. 关键设计决策

| 决策项 | 决策 | 理由 |
| --- | --- | --- |
| 系统管理归属 | 留在 Compute | 与计算任务提交路径强耦合（配额校验、资源引用），避免跨服务调用开销；内部按 package 隔离保留未来拆分空间 |
| 队列模型 | 扁平结构（v1 无父子层级）；每个 `(tenant, pool)` 默认 `default`；1:1 映射到 Volcano Queue | 租户已是强隔离边界，业务线拆分用同级多队列即可；分层带来的 schema / 链式配额 / 孤儿补偿复杂度放到后续按需引入 |
| 配额记账 | PG 不记账，Volcano 为实际用量权威；Compute 仅做 best-effort 预检 | 避免 PG / K8s 双源冲突；Volcano 调度准入天然具备强约束 |
| MLJob spec 粒度 | 声明式高阶抽象（framework / replicas / resourceUnit / queue 等） | 隔离 K8s 细节变更影响；operator 可独立演进 Pod 模板 |
| 状态同步 | Informer + PG 落库 | K8s 原生；支持大列表与按状态筛选；与 controller-runtime 工具链吻合 |
| 认证鉴权 | Platform 统一认证与鉴权；Compute 仅通过 `X-Axisml-User` 记录调用方做审计 / ownership | 避免职责重复；外部入口收敛到 Platform；租户归属通过 URL 路径自然表达 |
| ResourceUnit 节点匹配 | 通用 `node_selector` 而非 `gpu` 字段 | 覆盖 GPU / TPU / 自研加速卡 / CPU instance type 等任意硬件维度，对未来硬件可扩展 |
| ResourcePool 纳管 | `labelSelector` + `tolerations` | K8s 原生；运维友好；多云节点标签可复用 |
| API 协议 | REST / JSON + OpenAPI 3.0 | 可生成 Platform 端 SDK；内部 HTTP 调用简单直接；工具链成熟 |
| 写路径一致性 | Outbox 模式：API 单写 PG (`Creating`) + Compute 内 reconciler worker 幂等下发 CR + Informer 推进状态机 | 规避 PG commit 与 K8s `Create()` 之间的非原子窗口；任意单点崩溃都可恢复，无孤儿对象 |
| 业务状态机 | 显式 `Creating` 起点：API 写入；Informer ADD 推 `Pending`；终态由 CR `phase` 决定 | `Creating` 可区分"已登记未下发"与"已下发"，让正向孤儿可识别可重试 |
| Reconciler 实现 | 每模块在 Compute 进程内起 goroutine 轮询 PG，复用 k8sclient | 复用现有模块结构；避免新增独立 controller 二进制；与 Informer 共用 work queue 顺序自然 |
| CR 稳定锚点 | `metadata.name` + label `axisml.io/<resource>-id=<uuid>` | name 因软删可能重用，UUID 永久唯一；孤儿检测按 label 索引而非 name |
| 补偿策略分流 | 配置对象按 PG 快照补偿重建；工作负载对象的 CR 缺失视为生命周期终点，不重建 | 避免把用户取消、operator 级联清理误判为孤儿；`status='Creating' AND deleted_at IS NULL` 是唯一触发重建的判据 |

## 11. 未来规划

- **分层队列**：`queues` 表新增 `parent_id`，利用 Volcano Capacity plugin 的 `spec.parent` 能力实现父子配额与抢占；在有真实多团队/多业务线共享租户的诉求时引入
- **数据卷管理**：待设计字段 schema、底层存储映射（PVC / NFS / S3 / hostPath 等）、operator 注入契约
- **多集群**：队列 / 任务按集群维度扩展，联邦调度
- **细粒度配额**：GPU 时长、存储容量、网络带宽等
- **审计日志**：独立 `audit_events` 表，关键写操作落库
- **成本核算**：基于 `jobs` / `services` 的实际使用时长 × 资源单元成本的计费导出
