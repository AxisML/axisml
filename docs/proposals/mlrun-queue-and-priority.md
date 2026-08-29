# MLRun 排队与优先级方案

- 状态：Implemented
- 日期：2026-08-29
- 影响范围：Platform Job / Experiment、Compute Service、Kubernetes Runtime、Standalone Docker Runtime

## 1. 背景

当前两种部署形态的资源等待发生在不同位置：

- Kubernetes：Compute Service 先创建 `MLRun` CR，compute-operator 再创建
  `Job` / Pod；资源不足时 Pod 留在 Kubernetes scheduler 的 Pending 队列。
- standalone：Compute Service 调用 Docker Runtime；受管 GPU 不足时 Runtime
  返回 `ResourceUnavailableError`，Run 留在 `Pending` 并重试。CPU / 内存没有统一的
  提交前准入。

这两条路径都不能满足“资源不足时不要向 Kubernetes 或 Docker 提交任务”。同时，
当前没有跨部署形态一致的任务优先级语义。

本方案把 MLRun 的资源准入前移到 Compute Service：只有整个 Run 的资源需求能够一次性
满足时，才调用 `ComputeRuntime.ApplyMLRun`。资源不足的 Run 只保留在 PostgreSQL，
不会创建 MLRun CR、Kubernetes Job / Pod 或 Docker container。

### 1.1 术语

当前仓库没有名为 `MLJob` 的公共资源。Platform 的 `Job` / `Experiment` 是可复用定义，
每次触发产生一个 System 层 `MLRun`。因此本方案不新增 `MLJob` CR：

- 优先级声明在 Platform `Job` / `Experiment` 的 annotation 中；
- 触发时把有效优先级快照到本次 `MLRun` 的 annotation；
- 真正进入排队与准入状态机的是 `MLRun`。

## 2. 目标与非目标

### 2.1 目标

1. Kubernetes 与 standalone 共用同一套队列顺序、状态机、配额检查和准入算法。
2. 资源不足时，MLRun 不进入目标 runtime。
3. 一个 MLRun 的全部 role / replica 一次性准入，禁止部分提交。
4. 优先级来自 annotation；同优先级按 FIFO 顺序检查。
5. Compute Service 重启或 leader 切换后不丢队列，也不重复占用资源。
6. 运行中的 MLService / Workspace / TensorBoard 计入可用资源计算，防止队列忽略常驻负载。

### 2.2 非目标

- 本阶段不对运行中的任务做抢占；高优先级只影响尚未提交的 Run。
- 本阶段不把 MLService / Workspace / TensorBoard 纳入优先级队列；它们沿用现有创建路径，
  但其 desired resources 是 MLRun 准入的既有占用。
- 不实现队列超时、预约开始时间、用户级并发上限或 DRF 等公平调度。
- 不替代 `axisml-scheduler`。Kubernetes scheduler 仍是 Pod 级最终安全边界，并继续处理
  Kubernetes 原生约束。
- 不根据 Prometheus 的瞬时 CPU / 内存利用率调度。准入使用资源 request / reservation，
  避免负载波动导致反复超卖。

## 3. 核心不变量

1. **PG 是队列权威态**：不增加 Redis、消息队列或独立 scheduler service。
2. **Queued 表示 runtime 中不存在任务**：`phase=Queued` 时不得存在对应 MLRun CR、
   Kubernetes Job / Pod 或 Docker container。
3. **先持久化预留，再调用 runtime**：Run 从 `Queued` 原子推进到 `Creating` 后即占用
   admission reservation，随后才允许调用 `ApplyMLRun`。
4. **全量准入**：只有所有 replica 都能通过节点匹配、资源向量和租户 quota 检查时才推进。
5. **不重复计数**：已物化的 Pod / container 按实际节点占用计算；尚未物化但已进入
   `Creating` 的 workload 按 PG desired spec 计算。
6. **库存不可用时 fail closed**：无法取得完整资源快照时保留队列，不猜测容量、不提交任务。
7. **runtime 仍做最终校验**：资源在快照与实际 Apply 之间变化时，runtime 可以拒绝；拒绝必须
   保证没有创建任何 instance，Compute 再把 Run 放回 `Queued`。

## 4. 公共契约

### 4.1 优先级 annotation

公共 key：

```yaml
metadata:
  annotations:
    scheduling.axisml.io/priority: "100"
```

值为十进制有符号 32 位整数，数值越大优先级越高。未声明时为 `0`。例如：

| annotation 值 | 含义 |
| ---: | --- |
| `100` | 高于默认优先级 |
| `0` 或缺省 | 默认优先级 |
| `-100` | 低于默认优先级 |

该值只定义 AxisML 控制面队列顺序，与 `spec.scheduling.priorityClass` 正交：

- `scheduling.axisml.io/priority` 在 Kubernetes 和 standalone 中语义一致；
- `priorityClass` 仍只控制任务提交后 Kubernetes scheduler 的 Pod 优先级；
- 系统不自动把数字 priority 映射成 Kubernetes `PriorityClass`，也不因此触发抢占。

校验规则：

- 非十进制整数或超出 `int32` 范围时返回 `400 validation_failed`；
- 该 key 是系统保留 annotation；MLRun 创建后不可修改；
- MLRun 的 replacement-annotations PATCH 必须保留创建时的有效值，新增、删除或改变该 key
  返回 `409 immutable-field`；其他 annotation 仍可按现有合同修改；
- Job / Experiment 定义更新只影响之后触发的 Run，不重排已经存在的 Run。

Platform 在 Job / Experiment create、patch 和 Run trigger 时校验，Compute MLRun Create 再做
一次边界校验，避免绕过 Platform 的内部调用写入非法值。

### 4.2 定义到 Run 的快照规则

触发 Job / Experiment 时只对保留的 priority annotation 增加继承，不改变其他 annotation
现有的非继承语义。有效值按以下顺序解析：

1. `RunTriggerRequest.annotations["scheduling.axisml.io/priority"]`；
2. Job / Experiment 定义上的同名 annotation；
3. 默认值 `0`。

有效值写入 Compute MLRun 的 `annotations` 和内部派生列 `priority`。直接调用 Compute
MLRun Create API 时，从请求自身 annotation 解析。MLRun 被准入后，Compute 在派生的
MLRun CR metadata 上保留该 annotation，便于 Kubernetes 侧排障；其他用户 annotation
仍保持现有 PG-only 语义。

### 4.3 生命周期

新增公共 phase `Queued`：

```text
                       capacity / quota not available
                              ┌──────────────┐
                              │              │
                              ▼              │
POST ──▶ Queued ──admit──▶ Creating ──Apply success──▶ Pending ──▶ Running
           │                    │                                  │
           │ cancel/delete      ├─ ResourceUnavailable + no instance ─▶ Queued
           │                    └─ TerminalApplyError ───────────────▶ Failed
           ▼
     Cancelled / Deleted
```

状态语义：

| phase | runtime 对象 | 资源语义 |
| --- | --- | --- |
| `Queued` | 不存在 | 尚未预留，等待准入 |
| `Creating` | 可能尚不存在 | 已持久化预留，正在幂等提交 |
| `Pending` | 已存在 | runtime 已接受，等待启动 / 调度 |
| `Running` 及后续 | 已存在或正在回收 | 沿用现有语义 |

API 创建成功仍立即返回业务对象，不因排队返回错误。资源不足通过 `phase=Queued` 和
`status.queueReason` / `status.message` 表达。`queueReason` 是稳定枚举，首版包括
`InventoryUnavailable`、`QuotaUnavailable`、`QuotaExceeded`、`NoMatchingNode` 和
`InsufficientResources`；离开 `Queued` 时清空。`message` 只提供可读细节，不作为程序合同，
例如：

```text
waiting for resources: need cpu=4,memory=16Gi,nvidia.com/gpu=1;
available cpu=8,memory=32Gi,nvidia.com/gpu=0
```

`createdAt` 即入队时间。新增 `scheduledAt`，在 `ApplyMLRun` 成功、phase 进入
`Pending` 时写入；如果 Apply 前容量变化并回到 `Queued`，不设置 `scheduledAt`。

Platform 的 `Run` 返回有效 annotations，使调用方能够看到本次 Run 已快照的 priority。
本阶段不提供稳定的 `queuePosition`：节点约束、配额和 backfill 会让单一位置数字具有误导性。

## 5. 组件边界

### 5.1 Compute Service 负责排序与决策

在 `compute-service/internal/admission` 增加 leader-only `RunQueueController`。它负责：

- 从 PG 读取 `Queued` Run 和所有仍占资源的 Run / Service desired spec；
- 读取租户对各 pool 的 quota max；
- 读取部署形态提供的资源库存快照；
- 执行节点匹配、资源预留、排序和准入；
- 原子推进 `Queued → Creating`；
- 维护队列状态消息和指标。

MLRun reconciler 不再扫描 `Queued`。它只处理已经准入的 `Creating`，并在
`ApplyMLRun` 成功后推进到 `Pending`。这样 runtime adapter 不感知队列，也不拥有优先级逻辑。
Controller 复用现有 `ReconcileInterval` 周期运行；创建、终态和删除事件只用于提前唤醒，
正确性不依赖进程内事件是否送达。

### 5.2 部署形态只提供资源库存

在 `compute-service/pkg/extensions` 增加稳定的只读边界：

```go
type ResourceInventory interface {
    Snapshot(ctx context.Context) (CapacitySnapshot, error)
}

type QuotaResolver interface {
    ResolveQuota(ctx context.Context, tenant, pool string) (ResourceList, error)
}
```

`CapacitySnapshot` 至少包含：

- 可调度节点的 identity、labels、taints 和 allocatable resources；
- 当前 allocation 的 node、resources、workload stable ID、role 和 replica；
- 快照时间与来源版本，供日志和指标记录。

接口只使用 AxisML API / Kubernetes core resource types，不暴露 Docker SDK、scheduler
framework 或数据库类型。节点放置算法留在 Compute Service，保证两种部署形态得到相同结果。

`QuotaResolver` 的实现：

- Kubernetes：从 manager live API reader 读取 `Tenant.spec.quotas[pool].max`，避免新建 Tenant
  尚未进入 informer cache 的短窗口；
- standalone：复用启动时加载的 `StaticTenantStore`；
- quota 暂时不存在或未就绪时保留 `Queued`，并写明等待原因。

## 6. 容量与占用模型

### 6.1 Run 资源需求

每个 role replica 是一个独立 placement：

```text
replica_request = role.template.resources.requests
run_quota_request = Σ(role.replicas × replica_request)
```

准入以 `requests` 为准，与 Kubernetes scheduler 一致；ResourceUnit 当前会把 requests / limits
一并快照到 role template。所有标量使用 Kubernetes `resource.Quantity` 运算，不能转为
`float64`。CPU、memory、`nvidia.com/gpu` 以及其他 extended resource 都参与向量比较。
Run 所属 pool / unit 继续读取 PG 中现有的 `resource.axisml.io/{pool,unit}` 系统 label，
不在冻结 spec 中重复增加名字字段。

节点过滤使用已快照的 `spec.scheduling.nodeSelector` 和 `tolerations`。每个 Run 的全部
replica 都成功放置才算可准入；任何一步失败都回滚本轮内存模拟，不产生部分 reservation。

### 6.2 既有占用与持久化 reservation

为避免“CR 已创建但 Pod 尚未出现”的观察窗口造成重复准入，容量计算合并两类证据：

1. Runtime snapshot 中已经物化的 Pod / container allocation；
2. PG 中已准入 workload 的 desired reservation。

allocation 通过稳定的 `run-id` / `service-id`、role、replica identity 与 PG desired workload
匹配。已经物化的 replica 使用 runtime 中的真实节点位置；desired 中缺失的 replica 再由
admission 模拟放置。无法归属到当前 PG workload 的 allocation 一律作为外部占用扣除。

占用阶段：

- MLRun：`Creating`、`Pending`、`Running`、`Canceling`、`Deleting`；
- MLService：所有未 `Deleted` 且 desired replicas 大于 0 的记录，包括正在创建、降级或自愈者；
- 已结束且不再持有执行资源的 Run 不计入 reservation。

`Queued` 不占资源。`Queued → Creating` 的 PG 更新本身就是 durable reservation，不增加独立
queue / lease 表。

### 6.3 租户 quota

每次准入同时满足：

```text
tenant_pool_reserved + run_quota_request <= Tenant.spec.quotas[pool].max
```

`max` 是硬上限；`min` 暂不参与队列公平性，与当前 axisml-scheduler 只强制 max 的实现保持
一致。Kubernetes scheduler 继续做最终 quota 校验。standalone 通过同一 admission 逻辑首次
真正执行静态 Tenant quota，因而两种形态在这一点上收敛。

### 6.4 Kubernetes inventory

Kubernetes 实现通过 informer cache 读取 Node 和 Pod：

- 排除 NotReady 或 `spec.unschedulable=true` 的 Node；
- Node 提供 labels、taints、`status.allocatable`；
- 非终态 Pod 按 scheduler 的 request 规则扣减，包括 init container 最大 request 与 Pod overhead；
- 已绑定 Pod 保留实际 `spec.nodeName`；尚未绑定但属于已准入 AxisML workload 的 Pod 作为
  desired reservation 再模拟放置；
- Compute Service RBAC 增加 Node 的 `get/list/watch`，现有 Pod 只读权限复用。

准入只覆盖当前 MLRun contract 已表达的 resources、nodeSelector 和 tolerations。PVC topology、
第三方 scheduler plugin、节点在快照后的变化仍由 Kubernetes scheduler 最终判定。因此本机制
对 RunQueueController 管理的并发 MLRun 准入提供确定性防超卖，但不声称能锁住集群中所有
外部写入或本轮快照之后发生的 MLService 变更。

Platform / Compute 是 MLRun 的正式提交路径。System RBAC 应只给 Compute Service 写 MLRun CR
的权限；集群管理员直接 `kubectl create MLRun` 属于 break-glass 操作，会绕过本队列且必须按
外部 allocation 计入后续快照。

### 6.5 Standalone inventory

Standalone 把 Docker host 表示成一个虚拟节点：

- CPU / memory 总量来自 Docker Engine `Info`；
- GPU 总量来自受管 `gpu.devices`；未配置受管设备时 GPU 容量视为未知，GPU Run 保持排队；
- 增加可选 `workload.system_reserved_cpu` 与 `workload.system_reserved_memory`，从总量中扣除
  OS / 控制面保留；默认 `0`，生产部署应显式配置；
- 活跃 Docker container 按 cgroup CPU / memory limit 和 GPU device request 形成 allocation；
- AxisML container 通过现有 `io.axisml.*` labels 关联到 PG workload；其他 container 作为
  外部占用，存在硬 limit 时扣减该 reservation。

为避免同名 Run 删除后重建时误认旧 container，Docker Runtime 还需要把 Compute 已提供的
`compute.axisml.io/run-id` / `compute.axisml.io/service-id` 透传为 container label；
`(kind, namespace, name)` 只保留为运行时寻址和兼容元数据。

CPU / memory 无限制的外部 container 无法形成可靠 reservation；Standalone 的容量保证以
“AxisML 独占或显式切分的 Docker host”为部署前提。系统仍不使用瞬时 `docker stats` 作为
可承诺容量。

现有 GPU allocator 保留为 Apply 阶段的原子最终检查，但正常情况下 Run 已在 Compute
admission 阶段通过 GPU 容量判断，不再依赖反复调用 Docker 来排队。

## 7. 排序与准入算法

队列每次 reconcile 使用以下稳定顺序：

```sql
ORDER BY priority DESC, created_at ASC, id ASC
```

- priority 高者先检查；
- 同 priority 时先检查更早创建的 Run；
- `id` 仅用于 created_at 相同情况下的确定性排序。

一次 reconcile 的步骤：

1. 在数据库事务外读取 runtime capacity snapshot 和 Tenant quota snapshot；外部调用期间不持有
   PG lock。
2. 开启短数据库事务并取得 transaction-scoped PG advisory lock，防止 leader 切换窗口出现
   并行 admission；随后在事务内重读 active desired workloads 和 `Queued` Run。
3. 用外部 inventory snapshot 与事务内最新 PG reservation 重建 node residual 和
   tenant/pool reservation ledger。
4. 按上述顺序扫描 `Queued` Run。
5. 对每个 Run 复制当前 residual，按 GPU、memory、CPU 的 dominant demand 从大到小放置 replica；
   匹配多个 Node 时选择放置后剩余资源最少的 Node，贴近 `MostAllocated` binpack。
6. 全部 replica 可放置且 quota 未超限时，在同一数据库事务内把 Run 更新为 `Creating`，
   清除等待消息，并把本次 reservation 加入内存 ledger，继续判断下一项。
7. 不可放置时保留 `Queued`，更新稳定的 `status.queueReason`、可读 `status.message` 和指标后
   继续扫描其他 Run；reason 未变化时节流 detail 更新，避免每个 tick 改写同一行。

advisory lock 只覆盖步骤 2–7 的 PG 读写，不跨 `ResourceInventory`、Docker、Kubernetes、zot
或 S3 调用。库存快照可能在事务前后变化，Apply 阶段的最终校验和
`ResourceUnavailableError` 回队路径负责收敛该窗口。

步骤 7 是 work-conserving backfill：高优先级 Run 总是先获得检查机会，但一个当前无法放置的
大任务不会阻断可以利用剩余资源的小任务。因此 FIFO 是检查顺序，不是“被阻塞任务禁止后续
任务运行”的严格队头语义。它不承诺为高优先级任务提前空出资源；不希望被 backfill 延迟的
预约 / 严格队头策略留待后续单独设计。

## 8. 并发、失败与恢复

### 8.1 提交竞态

`Queued → Creating` 先于 runtime Apply 提交事务，因此同一轮后续 Run 以及 leader 重启后的
下一轮都会看到 reservation。Apply 必须幂等。

`ResourceUnavailableError` 的合同加强为：返回该错误时不能留下任何新 instance。如果
Apply 遇到快照后的容量变化：

- Compute 确认 runtime 中没有 instance；
- 把 Run 从 `Creating` 放回 `Queued`，清除 `scheduledAt`；
- 下一轮重新排序，不保留原 admission 特权。

其他瞬时 API / Docker 错误保留 `Creating` 并幂等重试；`TerminalApplyError` 进入 `Failed`
并释放 reservation。

### 8.2 卡在 Creating 的恢复

首版使用固定的 `dispatch_timeout=2m`。`Creating` 超时后先调用 `ObserveMLRun`：

- runtime 对象存在：修复为 `Pending` 并补 `scheduledAt`；
- runtime 对象不存在：放回 `Queued`；
- Observe 不可用：保持 `Creating`，继续保留 reservation，fail closed。

这样不会因控制面重启而重复放行同一份资源。

### 8.3 cancel / delete

- cancel `Queued` Run：不调用 runtime，直接进入 `Cancelled` 并写 `finishedAt`；
- delete `Queued` Run：不调用 runtime，直接进入 `Deleted`；
- cancel / delete `Creating` 及后续状态沿用幂等 runtime 路径；
- Platform 判断 Job / Experiment 是否存在活跃 Run 时把 `Queued` 加入 active phase。

### 8.4 升级恢复

上线迁移后，启动恢复任务处理旧状态：

- 旧 `Creating` 且 runtime 中不存在对象：改为 `Queued`；
- 旧 `Pending` 且 runtime 中不存在对象（包括原 standalone GPU 等待）：改为 `Queued`；
- runtime 对象已存在：保留 / 修复为 `Pending`；
- 所有历史 Run 从 priority annotation 回填派生列，缺省为 `0`。

恢复由 leader 的 MLRun reconciler 在正常 dispatch 之前持续执行。恢复期间旧 active row
仍计入 reservation，因此不会与新 admission 发生容量重叠；不额外阻塞只读 API readiness。

## 9. 数据模型与 API 变更

### 9.1 Compute 数据库

`mlruns` 增加：

```sql
priority     integer     NOT NULL DEFAULT 0,
scheduled_at timestamptz
```

并增加队列 partial index：

```sql
CREATE INDEX mlruns_admission_queue
ON mlruns (priority DESC, created_at ASC, id ASC)
WHERE phase = 'Queued' AND deleted_at IS NULL;
```

不增加独立 queue 表。`priority` 是 annotation 的校验后派生值，用于稳定排序；annotation
仍是公共声明源。

### 9.2 API

- Compute 与 Platform Run phase enum 增加 `Queued`；
- Compute MLRun / Platform Run 增加 `scheduledAt`；
- Compute MLRun status / Platform Run 增加 `queueReason`；
- Platform Run 投影 annotations。

队列、优先级和 Tenant pool quota 准入是两种部署形态的公共 MLRun 合同，不通过
部署 capability 字段区分。Kubernetes scheduler 的 ElasticQuota 是下游运行时语义，
不改变 MLRun API。

所有 OpenAPI、生成客户端、前端 phase 映射、i18n 和示例需要同步更新。

## 10. 可观测性

新增 `axisml_compute_mlrun_queue_depth` gauge；准入与 dispatch 恢复结果复用
`axisml_compute_reconciler_actions_total{resource,predicate,result}`。priority、tenant、run name
不进入 metrics label，避免高基数。结构化日志记录被准入 Run 的 namespace、name 和 priority；
等待原因保存在每个 Run 的 `status.queueReason` / `status.message` 中。

队列深度持续增长应作为默认告警候选。按 pool 的最老等待时间、等待时长 histogram 和 inventory
错误计数可在需要相应 SLO 时增加，本阶段不把这些尚未消费的序列加入公共观测面。

## 11. 验证策略

### 11.1 单元测试

- priority 缺省、合法边界、非法值和定义 / trigger 覆盖顺序；
- `priority DESC + FIFO + id` 稳定排序；
- 多资源向量、selector、taint / toleration、extended resource 和整组回滚；
- tenant quota max；
- 高优先级优先与不可放置任务的 backfill；
- active runtime allocation 与 PG desired reservation 去重。

### 11.2 Compute 集成测试

- 容量不足时 phase 为 `Queued` 且 `ApplyMLRun` 从未被调用；
- 资源释放后只准入能够完整放置的 Run；
- 高优先级后提交但先准入；同优先级且都可放置时按 FIFO 准入；
- cancel / delete Queued 不调用 runtime；
- leader 重启、Creating 超时和 `ResourceUnavailable` 回队不重复预留；
- MLService desired replicas 会减少 MLRun 可用容量。

### 11.3 Kubernetes 集成测试

- 构造 Node / Pod inventory，资源不足时 API Server 中不存在 MLRun CR；
- 释放 Pod 或增加 Node capacity 后才出现 MLRun CR；
- nodeSelector、taint / toleration、tenant quota 与多 replica all-or-nothing；
- CR 创建到 Pod 出现的窗口不会放行第二份相同资源。

### 11.4 Standalone 与黑盒测试

- Docker host capacity 不足时没有 image pull / container create；
- CPU、memory、受管 GPU 都参与准入；
- 容器退出 / 删除后下一任务自动提交；
- 同一组黑盒用例以 `--mode kubernetes|standalone` 验证 `Queued → Pending → Running`；
- 两个 capacity=1 的 Run 验证 priority 与 FIFO。

## 12. 实施拆分

1. **公共状态与持久化**：annotation 常量 / 校验、`Queued`、priority 派生列、scheduledAt、
   migration、OpenAPI。
2. **通用 admission engine**：inventory / quota seam、resource arithmetic、placement、队列 repository
   与 controller。
3. **Kubernetes inventory**：Node / Pod informer、Tenant quota cache、RBAC、集成测试。
4. **Standalone inventory**：Docker Info / container allocation、静态 Tenant quota、system reserve
   配置、替换 GPU-only 等待路径。
5. **Platform 与产品面**：Job / Experiment priority 快照、Run annotations / Queued 显示、客户端生成、
   双模式黑盒测试与正式设计文档更新。

每一步完成后都必须保持旧 Run 可恢复；只有第 3、4 步都完成后才把 capability 标记为启用。

## 13. 未采用的方案

### 13.1 继续依赖 Kubernetes scheduler / Docker Apply 重试

该方案最小，但任务已经提交到目标 runtime，直接违反本次需求，也无法给两种部署形态提供
统一优先级。

### 13.2 在 compute-operator 中排队

operator 只存在于 Kubernetes 形态，而且 MLRun CR 已经写入集群；既不能满足“不提交到集群”，
也无法复用到 standalone。

### 13.3 使用 Kubernetes PriorityClass 作为唯一优先级

PriorityClass 只在 Pod 已提交后生效，依赖集群预创建对象，standalone 没有等价语义；因此保留
为 downstream Kubernetes 能力，不作为 AxisML 队列合同。

### 13.4 新增独立 queue service 或消息队列

当前 `mlruns` 表已经是 durable desired-state / outbox。增加新服务会引入双写、恢复和部署复杂度，
但不会改善本阶段的排序与资源准入正确性。

### 13.5 按实时利用率判断空闲资源

CPU / GPU 利用率是瞬时观测，不是可承诺容量；按利用率准入会在任务负载上升后超卖。队列应按
allocatable、requests、quota 和 durable reservation 决策，利用率只用于观测与未来弹性策略。
