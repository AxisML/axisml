# MLJob Operator 详细设计

## 1. 与 Compute 的写路径契约

Compute 采用 Operation Outbox + reconciler 异步下发 CR（详见 [compute.md §5.2](../compute.md)）。Operator 必须满足以下契约：

- **`Create` 幂等**：相同 `metadata.name` 的二次 `Create()` 返回 409 `AlreadyExists`，且不引发副作用（不重启 Pod、不重置 status）。Compute 收到 409 后会 `Get()` 现有 CR 并校验 label `axisml.io/job-id=<uuid>`；只有 label 一致才视为成功
- **不主动反向写 PG**：operator 不感知 Compute 的 `jobs` / `services` / `tenants` 表；状态推进由 Compute 侧 Informer 按 CR `status` 回流
- **status 单向权威**：operator 只写 MLJob `status`；Compute 只写 MLJob `metadata` / `spec`
- **spec 幂等 Patch**：相同 `spec` 重复 Apply / Patch 不得重建 Pod；只有语义字段变化才触发底层资源变更
- **`spec.backend.{name,engine}` 创建后不可变**：reconciler 检测到变更直接拒绝（写 `status.message`），admission webhook 后续补；Compute 也保证不会修改这两个字段

## 2. 总体架构：Dispatcher + Handler

mljob-operator 由两层组成：

- **Dispatcher Reconciler**：watch 所有 MLJob CR，按 `spec.backend.{name, engine}` 元组路由到对应 Handler。本身不直接生成底层资源
- **Handler 注册表**：每个 `(backend, engine)` 元组对应一个 Handler 实例；编译期通过 `init()` 注册到全局 registry。Handler 负责把通用字段 + `backend.config` 翻译成具体的底层资源（Pod / PodGroup / 第三方 CR），并把后端原生状态映射回 MLJob 统一 phase

```
                 ┌────────────────────────────────┐
   MLJob CR ───▶ │  Dispatcher Reconciler         │
                 │  (按 (backend, engine) 路由)    │
                 └─────────────┬──────────────────┘
                               │
       ┌───────────────────────┼───────────────────────┐
       ▼                       ▼                       ▼
 (native,default)   (kubeflow-trainer,pytorch)   (custom, *)
     Handler                Handler                  Handler
 (PodGroup+Pod)         (PyTorchJob CR)        (用户声明的 GVK)
       │                       │                       │
       └───────────────────────┴───────────────────────┘
                               │
                       status backflow
                               │
                               ▼
                    MLJob.status (统一 phase)
```

**Watch 拓扑**：Dispatcher 始终 watch MLJob 主队列；每个 Handler 启动时声明自己关心的底层资源类型（Pod、PodGroup、PyTorchJob、TFJob、MPIJob …），由 dispatcher 统一建立 watch（controller-runtime `Watches()`），事件通过 `ownerReference` 反查回 MLJob 后再交给对应 Handler 的 `MapStatus`。

**注册方式**：编译期注册（每个 handler 包 `init()` → 全局 registry）。MVP 不引入运行时插件加载（plugin / wasm / 外部 grpc）。后续若需要"运行时安装新后端"，再演进为独立 operator binary 路由模式。

**未注册组合的兜底**：dispatcher 收到一条 `(backend, engine)` 无 handler 的 MLJob → 写 `status.phase=Failed`、`status.message="no handler for backend=X engine=Y"`，不创建任何底层资源。

## 3. CRD 契约

MLJob 为 namespaced CR，创建在租户 namespace 下。Compute 负责设置：

- `metadata.name`：来自 `jobs.name`
- `metadata.namespace`：来自 `tenants.namespace`
- `metadata.labels["axisml.io/job-id"]`：`jobs.id`
- `metadata.labels["axisml.io/tenant"]`：租户名
- `metadata.labels["axisml.io/queue"]`：Compute Queue 名

最小 `spec`：

```yaml
apiVersion: compute.axisml.io/v1alpha1
kind: MLJob
spec:
  backend:
    name: native             # 必填；枚举：native | kubeflow-trainer | custom
                             # （kserve 仅用于 MLService，不在 MLJob 枚举内）
    engine: default          # 必填；语义随 backend 而定，见 §6
    config: {}               # 可选；该 (backend, engine) 元组特有的配置，schemaless
  image: string
  command: []                # optional
  args: []                   # optional
  env: []                    # optional, Kubernetes EnvVar 数组
  replicas: int              # >= 1
  queueName: string          # Volcano Queue CR name
  suspend: bool              # optional, 用户取消时优先 patch 为 true
  resources:
    requests: {}             # Kubernetes ResourceList
    limits: {}               # Kubernetes ResourceList
  placement:
    nodeSelector: {}
    tolerations: []          # Kubernetes Toleration 数组
```

**字段归属**：

- `backend.{name,engine,config}` 由 dispatcher 用于路由，由具体 Handler 解释
- 通用字段（`image` / `command` / `args` / `env` / `replicas` / `queueName` / `resources` / `placement` / `suspend`）由所有 Handler 共同遵守语义；Handler 负责把它们注入到底层资源的对应位置
- `backend.config` 仅用于通用字段表达不出来的引擎特有配置（如 PyTorchJob 的 master/worker 角色拆分、MPIJob 的 launcher/worker 比例）

**默认值注入**：Compute 在创建 MLJob CR 时，若用户未指定 `spec.backend`，显式补 `{name: native, engine: default}`。dispatcher 不接受 `backend.name` 或 `backend.engine` 为空。

## 4. Status 契约

最小 `status`：

```yaml
status:
  observedGeneration: int64
  phase: Pending | Running | Succeeded | Failed
  message: string
  startedAt: timestamp
  finishedAt: timestamp
```

Compute 映射规则：

| MLJob status.phase | jobs.status | 终态 |
| --- | --- | --- |
| `Pending` | `Pending` | 否 |
| `Running` | `Running` | 否 |
| `Succeeded` | `Succeeded` | 是 |
| `Failed` | `Failed` | 是 |

CR 删除事件由 Compute Informer 映射为 `Cancelled`（若 PG 中 Job 尚未进入终态）。

**Phase 由 Handler 产出**：不同 backend 的原生状态各异（PyTorchJob 有 `Created/Running/Restarting/Succeeded/Failed`、MPIJob 有 launcher/worker 分阶段状态等），但对外只暴露上述四态。每个 Handler 在 `MapStatus` 中负责完成映射；映射规则写入对应 Handler 的章节（§6）作为契约。

## 5. Handler 接口契约

所有 Handler 必须实现以下行为（语言无关，描述责任）：

| 方法 | 责任 |
| --- | --- |
| `Key()` | 返回 `(backend, engine)` 元组，与 `spec.backend.{name,engine}` 对齐；用作注册表的主键 |
| `Validate(spec)` | 校验通用字段 + `backend.config`，纯函数，未来可被 admission webhook 复用 |
| `Reconcile(ctx, mlJob)` | 创建 / 更新底层资源；保证幂等；通用字段由 Handler 自己注入到对应位置；处理 `spec.suspend` 的暂停语义 |
| `MapStatus(underlying)` | 把后端原生状态映射回 §4 的四态 phase + message + startedAt / finishedAt |
| `Cleanup(ctx, mlJob)` | 删除底层资源。一般依赖 ownerReference 自动级联，Handler 仅负责需要主动清理的副本（如跨 namespace 资源、外部存储句柄） |
| `WatchTargets()` | 声明本 Handler 需要 watch 的底层资源 GVK 列表，dispatcher 启动时统一建立 watch |
| `RequiredRBAC()` | 声明本 Handler 需要的 ClusterRole 规则；启动时聚合到 operator ServiceAccount |

**幂等性要求**：

- `Reconcile` 多次调用相同 spec 不重建底层资源；只有语义字段变化才触发更新
- `Cleanup` 对已删除资源返回 nil
- `MapStatus` 是纯函数（不发起 K8s 调用），状态推进只依赖输入参数

**Suspend 语义统一**：当 `spec.suspend=true` 时——

- 支持原生 suspend 的 backend（如 PyTorchJob 的 `runPolicy.suspend`、Job 的 `spec.suspend`）→ patch 对应字段
- 不支持 suspend 的 backend → 兜底为 `Cleanup`（删除底层资源）；Compute 侧 Informer 观察到 CR DELETE 后推进 `Canceling → Cancelled`

## 6. 内置 Handler

### 6.1 `(native, default)` —— MVP

MVP 唯一落地的 Handler，保留现有 PodGroup + Pod 行为：

**底层资源**：

- 每个 MLJob 创建一个 Volcano `PodGroup`，`spec.queue` 设为 MLJob `spec.queueName`，`spec.minMember` 默认等于 `spec.replicas`
- 按 `replicas` 创建对应 Worker Pod；所有 Pod 设置 `schedulerName: volcano`
- PodGroup / Pod 设置 ownerReference 指向 MLJob，保证 MLJob 删除后底层资源级联清理
- operator 不读写 Volcano Queue CR；Queue CR 由 Compute 独占维护

**`backend.config`**：本 Handler 不消费 `backend.config`（schema 校验时若非空写 warning，不报错；为 future 字段预留）。

**Status 映射**：

| 原生状态 | MLJob phase |
| --- | --- |
| 所有 Pod `Pending` 或 PodGroup 排队中 | `Pending` |
| 至少一个 Pod 进入 `Running` | `Running` |
| 所有 Pod `Succeeded` | `Succeeded` |
| 任一 Pod `Failed` 或 PodGroup `PodGroupUnschedulable` | `Failed` |

`startedAt` 取首个 Pod `Running` 时间；`finishedAt` 取所有 Pod 进入终态的最晚时间。

**Suspend**：通过修改 PodGroup `spec.minMember=0` + 删除现存 Pod 实现。

**RBAC**：`pods`、`podgroups.scheduling.volcano.sh`、`events` 的 `create/get/list/watch/update/patch/delete`。

### 6.2 `(kubeflow-trainer, pytorch)` —— 占位

将 MLJob 翻译为 Kubeflow Trainer 的 [`PyTorchJob`](https://www.kubeflow.org/docs/components/training/pytorch/) CR。

**前置依赖**：集群已安装 kubeflow training-operator；其 RBAC 与 CRD 由 operator chart 单独管理，本 Handler 仅需要 `pytorchjobs.kubeflow.org` 的 `create/get/list/watch/update/patch/delete`。

**`backend.config` 关键字段**（schema 待细化设计文档）：

```yaml
config:
  master:
    replicas: 1                    # 默认 1
    restartPolicy: OnFailure
  worker:
    replicas: <override or use spec.replicas>
    restartPolicy: OnFailure
  elastic:
    enabled: bool
    minReplicas: int
    maxReplicas: int
```

通用字段映射：

- `image` / `command` / `args` / `env` / `resources` → 复制到 PyTorchJob 的 `master.template` 与 `worker.template` 的 container spec
- `placement` → 写入 `master.template.spec` / `worker.template.spec` 的 `nodeSelector` + `tolerations`
- `queueName` → 写入 PyTorchJob 模板 Pod 的 `schedulerName: volcano` + 关联 PodGroup（kubeflow training-operator 与 Volcano 集成方式以实际接入文档为准）
- `suspend` → patch `PyTorchJob.spec.runPolicy.suspend=true`

**Status 映射**：PyTorchJob `status.conditions` 映射到 MLJob phase——

| PyTorchJob condition | MLJob phase |
| --- | --- |
| `Created` / `Restarting` | `Pending` |
| `Running` | `Running` |
| `Succeeded` | `Succeeded` |
| `Failed` | `Failed` |

> 完整字段映射、容错策略、与 elastic training 的交互细节，由独立设计文档落地（见 §9）。

### 6.3 `(kubeflow-trainer, tensorflow)` —— 占位

同 §6.2，将 MLJob 翻译为 Kubeflow Trainer 的 `TFJob`。`config` 描述 chief / worker / parameter server / evaluator 角色拆分。Status 映射沿用 TFJob 的 condition 集，与 §6.2 PyTorchJob 同构。

### 6.4 `(kubeflow-trainer, mpi)` —— 占位

将 MLJob 翻译为 Kubeflow [`MPIJob`](https://www.kubeflow.org/docs/components/training/mpi/) CR。`config` 描述 launcher / worker 拆分与 MPI 实现（OpenMPI / Intel MPI）。状态映射对齐 MPIJob `status.conditions`。

### 6.5 `(custom, *)` —— 占位

为外部接入预留的一等公民。用户在 `backend.config` 中以 schemaless 方式描述目标 GVK 与字段映射，例如：

```yaml
backend:
  name: custom
  engine: any-name
  config:
    target:
      apiVersion: example.com/v1
      kind: MyTrainingRun
    fieldMappings:
      "spec.image": "$.image"
      "spec.replicas": "$.replicas"
      # ...
    statusMappings:
      "$.status.phase": { Created: Pending, Active: Running, Done: Succeeded, Error: Failed }
```

由 custom Handler 通过 unstructured client 创建并跟踪。**MVP 不实现**；当首个第三方后端无法纳入 `(kubeflow-trainer, *)` 时再设计完整 schema。

## 7. RBAC 聚合

operator binary 启动时遍历 registry，把每个 Handler 的 `RequiredRBAC()` 合并去重，生成本 binary 实际需要的 ClusterRole rules。Helm chart 渲染时根据启用的 Handler 集合（通过 values 控制）生成最小化 RBAC 而非全集。

## 8. 不变量与约束

- `spec.backend.{name, engine}` 创建后不可变（dispatcher 拒绝并写 `status.message`；admission webhook 后续接管）
- `(backend, engine)` 元组未在 registry 注册 → MLJob 直接进入 `Failed`，message 写明缺失原因
- 任一 Handler 的 `Reconcile` / `MapStatus` 必须保持 §1 列出的写路径契约——这是把"插件"安全嵌入现有架构的根基
- Handler 不直接修改 Volcano Queue CR；Queue CR 由 Compute 独占维护
- Handler 不向 Compute PG 写入任何数据；状态全部经由 MLJob `status` + Informer 回流

## 9. 后续设计文档（不在本文档范围）

- `(kubeflow-trainer, pytorch / tensorflow / mpi / paddle / xgboost)` 各自的字段映射与状态映射细节
- `(custom, *)` Handler 的 `config` 完整 schema 与 unstructured 操作约定
- Admission webhook：`spec.backend.{name,engine}` 不可变约束、`backend.config` 按 Handler 自带 schema 的统一校验
- Handler chart 的 values 控制（启用 / 禁用 backend，对应 RBAC 与 watch 的开关）
