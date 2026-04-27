# MLJob Operator 详细设计

## 1. 概述

mljob-operator 把 AxisML Compute 下发的 `MLJob` CR 翻译为底层执行资源（Pod / PodGroup / 第三方 CR），并把执行状态回流到 `MLJob.status`。它内部按 `spec.backend.{name, engine}` 二级元组路由到不同 Handler，由 Handler 真正生成底层资源、把后端原生状态映射回统一的 phase 集合（详见 [overview.md §5.3](../overview.md)）。

**面向两类读者**：

- **Compute 侧 informer / reconciler 作者**：理解 operator 暴露的 spec / status 契约，便于按 [compute.md §5](../compute.md) 的 Outbox + Informer 模型对接
- **新 backend Handler 作者**：理解 dispatcher 提供的注册、watch、RBAC 聚合机制，便于以最小代价接入新引擎

operator 与 Compute 的分工以 [compute.md §5](../compute.md) 为准；本文档只展开 operator 这一侧的实现契约。

## 2. 与 Compute 的写路径契约

Compute 采用 Operation Outbox + reconciler 异步下发 CR（详见 [compute.md §5.2](../compute.md)）。Operator 必须满足以下契约：

- **`Create` 幂等**：相同 `metadata.name` 的二次 `Create()` 返回 409 `AlreadyExists`，且不引发副作用（不重启 Pod、不重置 status）。Compute 收到 409 后会 `Get()` 现有 CR 并校验 label `axisml.io/job-id=<uuid>`；只有 label 一致才视为成功
- **不反向写 PG**：operator 不感知 Compute 的 `jobs` / `services` / `tenants` 表；状态推进由 Compute 侧 Informer 按 CR `status` 回流
- **status 单向权威**：operator 只写 `MLJob.status`；Compute 只写 `MLJob.metadata` / `MLJob.spec`
- **spec 幂等 Patch**：相同 `spec` 重复 Apply / Patch 不得重建 Pod；只有语义字段变化才触发底层资源变更
- **`spec.backend.{name, engine}` 创建后不可变**：dispatcher 检测到变更直接拒绝（写 `status.message`），admission webhook 后续补；Compute 也保证不会修改这两个字段
- **不引入 finalizer**：Compute 的 `Deleting → Deleted` 终态推进依赖 Informer 观察到 CR 真正消失（见 [compute.md §6.3.1](../compute.md)）。Handler 必须把所有底层资源经由 `ownerReference` 级联清理或在 `Cleanup()` 中同步处理，避免任何 finalizer 让 CR 卡在 `Terminating`
- **Suspend 优先于 Delete**：Compute 取消运行态 Job 时优先 `patch spec.runPolicy.suspend=true`，仅在 Handler 不支持原生 suspend 时退化为 `Delete()`。Handler 必须在自己的章节明确声明是否支持原生 suspend

## 3. CRD 契约

MLJob 为 namespaced CR（CRD 定义见 `deploy/helm/axisml-system/crds/mljob-crd.yaml`）：

| 字段 | 取值 |
| --- | --- |
| `apiVersion` | `axisml.io/v1alpha1` |
| `kind` | `MLJob` |
| `scope` | `Namespaced`（创建在租户 namespace 下） |
| `shortNames` | `mlj` |

Compute 负责设置以下 metadata（与 [compute.md §6.3.1](../compute.md) 对齐）：

- `metadata.name` ← `jobs.name`
- `metadata.namespace` ← `tenants.namespace`
- `metadata.labels["axisml.io/job-id"]` ← `jobs.id`（UUID，孤儿检测稳定锚点）
- `metadata.labels["axisml.io/tenant"]` ← 租户名
- `metadata.labels["axisml.io/queue"]` ← Compute Queue 名

### 3.1 spec 设计取舍

把"角色拓扑"提升为一等公民。Job 的执行域用 `roles[]` 数组承载：

- 单角色 backend（如 `(native, default)`）声明一个 role
- 多角色 backend（如 PyTorchJob 的 master/worker、TFJob 的 chief/worker/ps/evaluator、MPIJob 的 launcher/worker）声明多个 role
- role 名集合由各 Handler 在 §8 中约定，由 Handler 的 `Validate` 强制

替代方案是把 `image / command / replicas / resources` 全部摆在 spec 顶层（早期方案），对单角色自然，但多角色 backend 不得不把角色切分挤进 `backend.config`，让 generic 字段失去意义——`spec.replicas` 在多角色场景下到底指哪个？`spec.resources` 又对哪个角色生效？引入 `roles[]` 后，单角色场景退化为"只有一个 role 的特例"，避免这种"通用字段对一类后端无意义"的尴尬。

调度域的 `nodeSelector` / `tolerations` 沿用 K8s PodSpec 扁平惯例直接放在 `spec.scheduling` 下，不再额外包一层 `placement`。

### 3.2 spec 结构

```yaml
apiVersion: axisml.io/v1alpha1
kind: MLJob
spec:
  # ── 后端选择（创建后不可变；dispatcher 路由依据）─────────────────────
  backend:
    name: native              # 必填: native | kubeflow-trainer | custom
                              #      （kserve 仅用于 MLService）
    engine: default           # 必填: 语义随 backend 而定（见 §8）
    config: {}                # 可选: 该 (backend, engine) 元组特有的 schemaless 配置

  # ── 调度域（由 Compute 从 Queue / ResourcePool / ResourceUnit 合成注入）──
  scheduling:
    queue: string             # 必填: Volcano Queue CR 名（与 Compute Queue 1:1 映射）
    priorityClass: string     # 可选: K8s PriorityClass 名
    nodeSelector: {}          # Compute 按 compute.md §6.2.3 合并 pool + unit 后注入
    tolerations: []           # 来自 ResourcePool

  # ── 执行域：roles 数组承载角色拓扑 ─────────────────────────────────
  roles:
    - name: worker            # role 标识；同一 MLJob 内唯一
      replicas: 1             # >= 0；为 0 时该角色禁用（如 TFJob 的 evaluator）
      restartPolicy: OnFailure # OnFailure | Never
      template:               # Pod template 子集：暴露常用字段，隐藏完整 PodSpec
        image: string
        imagePullPolicy: IfNotPresent  # 可选: IfNotPresent | Always | Never
        command: []           # 可选
        args: []               # 可选
        env: []                # 可选: K8s EnvVar 数组
        envFrom: []            # 可选: K8s EnvFromSource 数组（ConfigMap / Secret 引用）
        workingDir: string     # 可选
        resources:
          requests: {}         # Compute 从 ResourceUnit.requests 注入
          limits: {}           # Compute 从 ResourceUnit.limits 注入

  # ── 生命周期 ────────────────────────────────────────────────────
  runPolicy:
    suspend: false                  # 可选: cancel 信号；Handler 暂停或清理底层资源
    activeDeadlineSeconds: int      # 可选: 硬超时；超时后 Handler 推 Failed
    ttlSecondsAfterFinished: int    # 可选: 终态后底层资源 GC；不影响 PG 软删
    backoffLimit: int               # 可选: 重试预算；具体语义由各 Handler 解释
```

### 3.3 字段归属与不可变性

| 字段路径 | 写入方 | 创建后可变？ |
| --- | --- | --- |
| `metadata.name` / `namespace` / `labels[axisml.io/*]` | Compute | 否 |
| `spec.backend.name` / `spec.backend.engine` | Compute（默认 `{native, default}`） | **否**；dispatcher 拒绝并写 `status.message` |
| `spec.backend.config` | Compute（默认 `{}`） | 视 Handler 语义；Handler 在 §7 `Validate` 中决定 |
| `spec.scheduling.queue` / `priorityClass` / `nodeSelector` / `tolerations` | Compute（合并 Queue + Pool + Unit） | 否 |
| `spec.roles[*].template.resources` | Compute（注入 ResourceUnit） | 否 |
| `spec.roles[*].replicas` | 用户提交时给定 | **否**（Job 是一次性 workload；扩缩容是 Service 专属） |
| `spec.runPolicy.suspend` | API（`:cancel` 触发） | **是**（cancel 路径专用） |
| 其他 `spec.runPolicy.*` 与 `spec.roles[*].template.*`（除 resources） | 用户提交 | 否 |

**默认值注入**：用户未指定 `spec.backend` 时，Compute 写 CR 时显式补 `{name: native, engine: default}`；`backend.config` 默认空对象 `{}`。dispatcher 不接受 `backend.name` 或 `backend.engine` 为空。

**CRD schema 现状**：当前 CRD 的 `spec` / `status` 用 `x-kubernetes-preserve-unknown-fields: true`，重新设计字段无需 CRD bump；待行为稳定后再启用 OpenAPI schema 严格校验。

## 4. Status 契约

```yaml
status:
  observedGeneration: int64     # Handler reconcile 自洽用；Compute 不强消费
  phase: Pending | Running | Succeeded | Failed   # ← Compute 唯一消费的字段
  message: string               # 错误或状态附加信息（Compute 透传到 jobs.message）
  startedAt: timestamp          # 首次进入 Running 的时间（Compute 写入 jobs.started_at）
  finishedAt: timestamp         # 进入终态的时间（Compute 写入 jobs.finished_at）
  conditions:                   # K8s 标准 conditions（UI 可观测，Compute 不消费）
    - type: Initialized | Scheduled | Suspended | Failed
      status: True | False | Unknown
      lastTransitionTime: timestamp
      reason: string
      message: string
  roles:                        # 各 role 副本聚合（UI 可观测，Compute 不消费）
    - name: string
      replicas: int             # spec 期望
      activeReplicas: int       # 运行中
      readyReplicas: int        # 通过 readiness probe
      succeededReplicas: int
      failedReplicas: int
```

**phase 枚举冻结为四态**——`Pending | Running | Succeeded | Failed`。新增 phase 必须 CRD schema 与 Compute 双侧同步演进。Compute 的状态映射规则（与 [compute.md §6.3.1](../compute.md) 对齐）：

| MLJob status.phase | jobs.status | 终态 |
| --- | --- | --- |
| `Pending` | `Pending` | 否 |
| `Running` | `Running` | 否 |
| `Succeeded` | `Succeeded` | 是 |
| `Failed` | `Failed` | 是 |

**`Cancelled` 与 `Deleted` 不由 operator 产出**——这两者是 Compute 侧 Informer 观察到 CR DELETE 事件后基于 PG 当前 `status` 推导（详见 [compute.md §5.3 / §5.4](../compute.md)）。operator 收到 cancel 信号（`spec.runPolicy.suspend=true`）后只负责"暂停或清理底层资源"，无需自行写 `Cancelled`。

`conditions` 与 `roles[]` 是给 UI 与运维用的 observability 字段，Compute 不消费。这样既保留了 K8s 标准实践（`metav1.Condition`、per-role 副本聚合）又不污染 Compute 状态机的简洁性。

跨 Handler 的 phase 映射规则原则：所有 Handler 在 `MapStatus` 中负责把后端原生状态映射到这四态，映射表写入对应 Handler 章节（§8）。

## 5. 总体架构：Dispatcher + Handler

mljob-operator 由两层组成：

- **Dispatcher Reconciler**：watch 所有 MLJob CR，按 `spec.backend.{name, engine}` 元组路由到对应 Handler；本身不直接生成底层资源
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

**Watch 拓扑**：Dispatcher 始终 watch MLJob 主队列；每个 Handler 启动时通过 `WatchTargets()` 声明自己关心的底层资源类型（Pod、PodGroup、PyTorchJob、TFJob、MPIJob …），由 dispatcher 统一建立 watch（controller-runtime `Watches()`），事件通过 `ownerReference` 反查回 MLJob 后再交给对应 Handler 的 `MapStatus`。

**注册方式**：编译期注册（每个 handler 包 `init()` → 全局 registry）。MVP 不引入运行时插件加载（plugin / wasm / 外部 grpc）。后续若需要"运行时安装新后端"，再演进为独立 operator binary 路由模式。

**未注册组合的兜底**：dispatcher 收到一条 `(backend, engine)` 无 handler 的 MLJob → 写 `status.phase=Failed`、`status.message="no handler for backend=X engine=Y"`，不创建任何底层资源。

## 6. Reconcile 生命周期

按事件源切分 dispatcher 与 Handler 的职责：

| 事件 | Dispatcher 行为 | Handler 行为 |
| --- | --- | --- |
| MLJob ADD（首次创建） | 路由到 Handler；调用 `Validate(spec)`，校验失败写 `status.phase=Failed` | `Reconcile(ctx, mlJob)` 创建底层资源，设置 `ownerReference: MLJob` |
| MLJob UPDATE（spec 变更） | 校验 `backend.{name, engine}` 不变（违反则写 `status.message` 拒绝）；其余 spec 变化路由给 Handler | `Reconcile` 幂等更新；只有语义字段变化才触发底层资源变更 |
| MLJob `spec.runPolicy.suspend=true` | 路由 | 支持原生 suspend → patch 底层资源；不支持 → 调用 `Cleanup()` |
| MLJob DELETE | 不阻断 | 一般依赖 ownerReference 级联清理；Handler 仅清理跨 namespace / 外部副作用（外部存储句柄、跨集群资源等） |
| 底层资源事件（Pod / PodGroup / 第三方 CR） | 通过 ownerReference 反查到 MLJob 后路由 | `MapStatus` 纯函数计算新 phase；dispatcher 把结果合并写入 `status` |

**关键约束**：

- Handler **不引入 finalizer**；ownerReference 级联清理是默认路径
- `MapStatus` 必须是纯函数（不发起 K8s 调用），便于未来在 admission webhook 中复用
- Handler 不能在 `Reconcile` 中直接写 `status`；所有 status 变更必须经过 `MapStatus`，由 dispatcher 统一合并写入，保证回流路径单一

**Pod label 约定**（跨 Handler 通用）：

| Label | 取值 | 用途 |
| --- | --- | --- |
| `axisml.io/job-id` | `jobs.id`（UUID） | 反查 MLJob，与 CR 上同名 label 一致 |
| `axisml.io/role` | role 名（如 `worker` / `master` / `launcher`） | 区分多角色拓扑下的 Pod |
| `axisml.io/replica-index` | role 内 0-based 序号 | **Compute §7.4 任务日志查询的定位依据** |

凡是管理可寻址副本的 Handler 都必须打这三个 label；不可寻址（例如 KServe 的 autoscaling pod 集合，但那属于 mlservice-operator）才允许省略 `replica-index`。

## 7. Handler 接口契约

所有 Handler 必须实现以下行为（语言无关，描述责任）：

| 方法 | 责任 |
| --- | --- |
| `Key()` | 返回 `(backend, engine)` 元组，与 `spec.backend.{name, engine}` 对齐；用作注册表的主键 |
| `Validate(spec)` | 校验通用字段 + `backend.config` + role 集合（数量、命名）；纯函数，未来可被 admission webhook 复用 |
| `Reconcile(ctx, mlJob)` | 创建 / 更新底层资源；幂等；通用字段由 Handler 自己注入到对应位置；处理 `spec.runPolicy.suspend` 的暂停语义 |
| `MapStatus(underlying)` | 把后端原生状态映射回 §4 的四态 phase + message + startedAt / finishedAt + conditions + roles 副本聚合 |
| `Cleanup(ctx, mlJob)` | 删除底层资源。一般依赖 ownerReference 自动级联，Handler 仅负责需要主动清理的副本（跨 namespace 资源、外部存储句柄） |
| `WatchTargets()` | 声明本 Handler 需要 watch 的底层资源 GVK 列表，dispatcher 启动时统一建立 watch |
| `RequiredRBAC()` | 声明本 Handler 需要的 ClusterRole 规则；启动时聚合到 operator ServiceAccount |

**幂等性细则**：

- `Reconcile` 多次调用相同 spec 不重建底层资源；只有语义字段变化才触发更新
- `Cleanup` 对已删除资源返回 nil
- `MapStatus` 是纯函数（不发起 K8s 调用），状态推进只依赖输入参数

**Suspend 声明义务**：每个 Handler 必须在自身章节（§8）显式声明 "原生支持 / 兜底为 Cleanup"。dispatcher 不做静默选择——不支持原生 suspend 时必须显式调用 `Cleanup()`，避免半暂停半运行的中间态。

**Status 写入约束**：Handler 只能通过 `MapStatus` 的返回值影响 `status`；不能在 `Reconcile` 中直接 `status` 写盘。dispatcher 统一合并 `phase` / `message` / `startedAt` / `finishedAt` / `conditions` / `roles[]` 写入 CR，保证 [§2 写路径契约](#2-与-compute-的写路径契约) 中的 "status 单向权威"。

## 8. 内置 Handler

每个 Handler 章节统一按以下小节组织：**底层资源 / `backend.config` / 通用字段映射 / Status 映射 / Suspend / RBAC**。

### 8.1 `(native, default)` —— MVP

MVP 唯一落地的 Handler，保留 PodGroup + Pod 行为。

**底层资源**：

- 必填且仅一个 role（`name=worker`）；`Validate` 拒绝多 role 提交
- 每个 MLJob 创建一个 Volcano `PodGroup`，`spec.queue ← spec.scheduling.queue`，`spec.minMember ← roles[worker].replicas`
- 按 `roles[worker].replicas` 创建对应 Worker Pod；所有 Pod 设置 `schedulerName: volcano`
- PodGroup / Pod 设置 `ownerReference` 指向 MLJob，保证 MLJob 删除后底层资源级联清理
- operator 不读写 Volcano Queue CR（Queue 由 Compute 独占维护）

**Pod label**：

- `axisml.io/job-id=<jobs.id>`
- `axisml.io/role=worker`
- `axisml.io/replica-index=<0-based>`（Compute §7.4 日志定位依据）

**`backend.config`**：本 Handler 不消费；非空时 `Validate` 写 warning，不报错（为 future 字段预留）。

**通用字段映射**：

| MLJob 字段 | Pod / PodGroup 落点 |
| --- | --- |
| `roles[worker].template.image` / `imagePullPolicy` / `command` / `args` / `env` / `envFrom` / `workingDir` | Pod 主容器同名字段 |
| `roles[worker].template.resources.requests` / `limits` | Pod 主容器同名字段 |
| `roles[worker].restartPolicy` | Pod `spec.restartPolicy` |
| `spec.scheduling.nodeSelector` / `tolerations` | Pod 同名字段 |
| `spec.scheduling.priorityClass` | Pod `spec.priorityClassName` |
| `spec.runPolicy.activeDeadlineSeconds` | Pod 同名字段 |
| `spec.runPolicy.ttlSecondsAfterFinished` | 终态后由 Handler 显式 GC（裸 Pod 无原生 TTL） |
| `spec.runPolicy.backoffLimit` | 通过 PodGroup 重试 + Handler 内部计数实现 |

**Status 映射**：

| 原生状态 | MLJob phase |
| --- | --- |
| 所有 Pod `Pending` 或 PodGroup 排队中 | `Pending` |
| 至少一个 Pod 进入 `Running` | `Running` |
| 所有 Pod `Succeeded` | `Succeeded` |
| 任一 Pod `Failed`、PodGroup `PodGroupUnschedulable`、超 `activeDeadlineSeconds` | `Failed` |

`startedAt` 取首个 Pod `Running` 时间；`finishedAt` 取所有 Pod 进入终态的最晚时间。`status.roles[worker]` 聚合 Pod 数（active / ready / succeeded / failed）。

**Suspend**：原生支持。`spec.runPolicy.suspend=true` 时——

1. patch `PodGroup.spec.minMember=0`
2. 删除现存 Pod（依赖 `restartPolicy=OnFailure` 不会自动重建）
3. 写 `status.conditions` 添加 `type=Suspended, status=True`

`suspend=false` 时反向恢复 minMember 与 Pod。

**RBAC**：`pods` / `podgroups.scheduling.volcano.sh` / `events` 的 `create / get / list / watch / update / patch / delete`。

### 8.2 `(kubeflow-trainer, pytorch)` —— 占位

将 MLJob 翻译为 Kubeflow Trainer 的 [`PyTorchJob`](https://www.kubeflow.org/docs/components/training/pytorch/) CR。

**前置依赖**：集群已安装 kubeflow training-operator；其 RBAC 与 CRD 由 operator chart 单独管理。本 Handler 仅需要 `pytorchjobs.kubeflow.org` 的 `create / get / list / watch / update / patch / delete`。

**Role 集合约定**：必须有 `master`（replicas=1，可省略默认）+ `worker`（replicas≥1），可选 `elasticAgent`。

**`backend.config` 关键字段**（schema 待细化设计文档）：

```yaml
config:
  elastic:
    enabled: bool
    minReplicas: int
    maxReplicas: int
  rdzv:                          # 分布式 rendezvous 后端
    backend: c10d | etcd
    endpoint: string
```

通用字段映射：

- `roles[master].template.*` → `pytorchReplicaSpecs.Master.template`（必为 1 副本）
- `roles[worker].template.*` → `pytorchReplicaSpecs.Worker.template`
- `spec.scheduling.*` → 各 replica 模板内的 nodeSelector / tolerations / schedulerName
- `spec.runPolicy.suspend` → patch `PyTorchJob.spec.runPolicy.suspend=true`
- `spec.runPolicy.activeDeadlineSeconds` / `backoffLimit` → `PyTorchJob.spec.runPolicy` 同名字段

**Status 映射**：从 `PyTorchJob.status.conditions` 推导——

| PyTorchJob condition | MLJob phase |
| --- | --- |
| `Created` / `Restarting` | `Pending` |
| `Running` | `Running` |
| `Succeeded` | `Succeeded` |
| `Failed` | `Failed` |

**Suspend**：原生支持（PyTorchJob `runPolicy.suspend`）。

> 完整字段映射、容错策略、与 elastic training 的交互细节，由独立设计文档落地（见 §11）。

### 8.3 `(kubeflow-trainer, tensorflow)` —— 占位

同 §8.2 思路，将 MLJob 翻译为 Kubeflow Trainer 的 `TFJob`。Role 集合约定为 `chief` / `worker` / `ps` / `evaluator`（任一可省略，replicas=0 表示禁用）。Status 映射沿用 TFJob 的 condition 集，与 §8.2 PyTorchJob 同构。Suspend 原生支持。

### 8.4 `(kubeflow-trainer, mpi)` —— 占位

将 MLJob 翻译为 Kubeflow [`MPIJob`](https://www.kubeflow.org/docs/components/training/mpi/) CR。Role 集合约定为 `launcher`（replicas=1）+ `worker`（replicas≥1）。`backend.config` 携带 MPI 实现选择（OpenMPI / Intel MPI）与 launcher / worker 通讯参数。Status 映射对齐 MPIJob `status.conditions`。Suspend 原生支持。

### 8.5 `(custom, *)` —— 占位

为外部接入预留的一等公民。用户在 `backend.config` 中以 schemaless 方式描述目标 GVK 与字段映射：

```yaml
backend:
  name: custom
  engine: any-name
  config:
    target:
      apiVersion: example.com/v1
      kind: MyTrainingRun
    fieldMappings:
      "spec.image": "$.roles[?(@.name=='worker')].template.image"
      "spec.replicas": "$.roles[?(@.name=='worker')].replicas"
      # ...
    statusMappings:
      "$.status.phase":
        Created: Pending
        Active: Running
        Done: Succeeded
        Error: Failed
```

由 custom Handler 通过 unstructured client 创建并跟踪。**MVP 不实现**——当首个第三方后端无法纳入 `(kubeflow-trainer, *)` 时再设计完整 schema。

## 9. RBAC 聚合

operator binary 启动时遍历 registry，把每个启用 Handler 的 `RequiredRBAC()` 合并去重，生成本 binary 实际需要的 ClusterRole rules。Helm chart 通过 values 控制启用集合，渲染最小化 RBAC 而非全集。MVP 仅启用 `(native, default)`，对应 RBAC 限定为 §8.1 列出的资源。

## 10. 不变量与约束

- `spec.backend.{name, engine}` 创建后不可变；dispatcher 拒绝并写 `status.message`，admission webhook 后续接管
- `(backend, engine)` 元组未在 registry 注册 → MLJob 直接进入 `Failed`，message 写明缺失原因
- 任一 Handler 的 `Reconcile` / `MapStatus` 必须保持 §2 列出的写路径契约——这是把"插件"安全嵌入 Compute Outbox 模型的根基
- Handler 不直接修改 Volcano Queue CR；Queue CR 由 Compute 独占维护
- Handler 不向 Compute PG 写入任何数据；状态全部经由 MLJob `status` + Informer 回流
- **Handler 不引入 finalizer**；级联清理依赖 ownerReference + `Cleanup()`
- **`status.phase` 取值集合冻结为四态**（`Pending | Running | Succeeded | Failed`）；新增 phase 必须经 CRD schema 与 Compute 双侧同步演进
- 管理可寻址副本的 Handler 必须打 `axisml.io/job-id` / `axisml.io/role` / `axisml.io/replica-index` 三件套 label

## 11. 后续设计文档（不在本文档范围）

- `(kubeflow-trainer, pytorch / tensorflow / mpi / paddle / xgboost)` 各自的字段映射与状态映射细节
- `(custom, *)` Handler 的 `config` 完整 schema 与 unstructured 操作约定
- Admission webhook：`spec.backend.{name, engine}` 不可变约束、`backend.config` 按 Handler 自带 schema 的统一校验
- Handler chart 的 values 控制（启用 / 禁用 backend，对应 RBAC 与 watch 的开关）
- CRD 严格 schema（启用 OpenAPI 校验，替换当前的 `x-kubernetes-preserve-unknown-fields`）
