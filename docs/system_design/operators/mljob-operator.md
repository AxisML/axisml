# MLJob Operator 详细设计

## 1. 概述

mljob-operator 把 AxisML Compute 下发的 `MLJob` CR 翻译为底层执行资源（Pod / PodGroup / 第三方 CR），并把执行状态回流到 `MLJob.status`。它内部按 `spec.backend.{name, engine}` 二级元组路由到不同 Handler，由 Handler 真正生成底层资源、把后端原生状态映射回统一的 phase 集合（详见 [overview.md §5.3](../overview.md)）。

operator 与 Compute 的分工以 [compute.md §5](../compute.md) 为准；本文档只展开 operator 这一侧的实现契约。

## 2. 与 Compute 的写路径契约

Compute 采用 Operation Outbox + reconciler 异步下发 CR（详见 [compute.md §5.2](../compute.md)）。Operator 暴露给 Compute 的核心契约只有两条；其余约束（`backend.{name, engine}` 不可变、不引入 finalizer、Suspend 声明义务等）分散在 §3.3 字段不可变性、§6 Reconcile 生命周期、§7 Handler 接口契约、§10 不变量与约束。

- **`Create` 幂等**：相同 `metadata.name` 的二次 `Create()` 返回 409 `AlreadyExists`，且不引发副作用（不重启 Pod、不重置 status）。Compute 收到 409 后会 `Get()` 现有 CR 并校验 label `axisml.io/job-id=<uuid>`；只有 label 一致才视为成功
- **status 单向权威**：operator 只写 `MLJob.status`，Compute 只写 `MLJob.metadata` / `MLJob.spec`；状态推进由 Compute 侧 Informer 按 CR `status` 回流，operator 不感知 Compute 的 `jobs` 表，也不向 Compute PG 写入任何数据

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
- `metadata.labels["axisml.io/quota"]` ← Compute Quota bare name（如 `training`，**不是** ElasticQuota 全名）

### 3.1 spec 设计取舍

把"角色拓扑"提升为一等公民。Job 的执行域用 `roles[]` 数组承载：

- 单角色 backend（如 `(native, job)`、`(native, podgroup)`）声明一个 role
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
    engine: job               # 必填: 语义随 backend 而定（见 §8）
                              #   native:           job | podgroup
                              #   kubeflow-trainer: pytorchjob | tfjob | mpijob | …
                              #   custom:           任意名（由 backend.config 描述目标 GVK）
    config: {}                # 可选: 该 (backend, engine) 元组特有的 schemaless 配置

  # ── 调度域（由 Compute 从 Quota / ResourcePool / ResourceUnit 合成注入）──
  scheduling:
    quota: string             # 必填: ElasticQuota CR 名（axisml-<tenant>-<pool>-<quota>，与 Compute Quota 1:1 映射）
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
| `spec.backend.name` / `spec.backend.engine` | Compute（默认 `{native, job}`） | **否**；dispatcher 拒绝并写 `status.message` |
| `spec.backend.config` | Compute（默认 `{}`） | 视 Handler 语义；Handler 在 §7 `Validate` 中决定 |
| `spec.scheduling.quota` / `priorityClass` / `nodeSelector` / `tolerations` | Compute（合并 Quota + Pool + Unit） | 否 |
| `spec.roles[*].template.resources` | Compute（注入 ResourceUnit） | 否 |
| `spec.roles[*].replicas` | 用户提交时给定 | **否**（Job 是一次性 workload；扩缩容是 Service 专属） |
| `spec.runPolicy.suspend` | API（`/cancel` 触发） | **是**（cancel 路径专用） |
| 其他 `spec.runPolicy.*` 与 `spec.roles[*].template.*`（除 resources） | 用户提交 | 否 |

**默认值注入**：用户未指定 `spec.backend` 时，Compute 写 CR 时显式补 `{name: native, engine: job}`（K8s 原生 Job + koord-scheduler，详见 §8.1）；`backend.config` 默认空对象 `{}`。dispatcher 不接受 `backend.name` 或 `backend.engine` 为空。

**CRD schema 现状**：当前 CRD 的 `spec` / `status` 用 `x-kubernetes-preserve-unknown-fields: true`，重新设计字段无需 CRD bump；待行为稳定后再启用 OpenAPI schema 严格校验。

**status subresource 要求**：`MLJob` CRD 必须启用 `subresources.status`，保证 dispatcher 只写 `status`、Compute 只写 `metadata` / `spec` 的边界能由 Kubernetes API Server 隔离。当前 `deploy/helm/axisml-system/crds/mljob-crd.yaml` 尚未声明该 subresource，属于实现对齐项；本文档先锁定契约，不在本次修订中修改 CRD 文件。

## 4. Status 契约

```yaml
status:
  observedGeneration: int64     # Handler reconcile 自洽用；Compute 不强消费
  phase: Pending | Running | Succeeded | Failed   # ← Compute 唯一消费的字段
  message: string               # 错误或状态附加信息（Compute 透传到 jobs.message）
  startedAt: timestamp          # 首次进入 Running 的时间（Compute 写入 jobs.started_at）
  finishedAt: timestamp         # 进入终态的时间（Compute 写入 jobs.finished_at）
  conditions:                   # K8s 标准 conditions（Suspended 会被 Compute 消费为 cancel 推进信号；其余仅 UI 观测）
    - type: Initialized | Scheduled | Suspended | Failed
      status: True | False | Unknown
      lastTransitionTime: timestamp
      reason: string             # Suspended 时约定 reason=CancelRequested
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

**Cancel 推进信号**——`Cancelled` 与 `Deleted` 仍不由 operator 直接产出，但 cancel 路径有明确的链上信号：Handler 在收到 `spec.runPolicy.suspend=true` 并完成"暂停或清理底层资源"后，**必须向 dispatcher 返回 `suspendCompleted=true` 与 `reason=CancelRequested`**；dispatcher 统一合并写入 `status.conditions[type=Suspended,status=True,reason=CancelRequested]`，且在非终态时让 `status.phase` 维持在 `Pending`。Compute Informer 在 PG `status='Canceling'` 时把这个 condition 当作推进信号 → 写 `Cancelled` → 入队 `Delete()` 做 CR 资源回收（DELETE 事件幂等到达，不再变更 PG 状态；详见 [compute.md §5.2 / §5.3 / §6.3.1](../compute.md)）。`Deleted` 仍由 Compute Informer 在观察到 CR DELETE 事件后基于 PG 当前 `status` 推导。

**终态优先**：cancel 只面向仍处于 `Pending` / `Running` 的 Job。若底层资源已经进入 `Succeeded` / `Failed`，或同一轮 `MapStatus` 已经推导出终态，dispatcher 必须保留终态 phase 与 `finishedAt`，不能为了 cancel 信号把 `status.phase` 回退为 `Pending`；此时不写 `Suspended=True` 作为成功取消信号。

`Suspended` 之外的 `conditions` 与 `roles[]` 是给 UI 与运维用的 observability 字段，Compute 不消费。这样既保留了 K8s 标准实践（`metav1.Condition`、per-role 副本聚合）又不污染 Compute 状态机的简洁性。

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
   (native,job)       (kubeflow-trainer,pytorchjob) (custom, *)
     Handler                Handler                  Handler
   (K8s Job)            (PyTorchJob CR)        (用户声明的 GVK)
       │                       │                       │
       └───────────────────────┴───────────────────────┘
                               │
                       status backflow
                               │
                               ▼
                    MLJob.status (统一 phase)
```

**Watch 拓扑**：Dispatcher 始终 watch MLJob 主队列；每个 Handler 启动时通过 `WatchTargets()` 声明自己关心的底层资源类型（Pod、PodGroup、PyTorchJob、TFJob、MPIJob …），由 dispatcher 统一建立 watch（controller-runtime `Watches()`），事件通过 `ownerReference` 反查回 MLJob 后再交给对应 Handler 的 `MapStatus`。

**注册方式**：编译期注册（每个 handler 包 `init()` → 全局 registry）；不引入运行时插件加载（plugin / wasm / 外部 grpc）——若未来需要"运行时安装新后端"，再演进为独立 operator binary 路由模式。

**未注册组合的兜底**：dispatcher 收到一条 `(backend, engine)` 无 handler 的 MLJob → 写 `status.phase=Failed`、`status.message="no handler for backend=X engine=Y"`，不创建任何底层资源。

## 6. Reconcile 生命周期

按事件源切分 dispatcher 与 Handler 的职责：

| 事件 | Dispatcher 行为 | Handler 行为 |
| --- | --- | --- |
| MLJob ADD（首次创建） | 路由到 Handler；调用 `Validate(spec)`，校验失败写 `status.phase=Failed` | `Reconcile(ctx, mlJob)` 创建底层资源，设置 `ownerReference: MLJob` |
| MLJob UPDATE（spec 变更） | 校验 `backend.{name, engine}` 不变（违反则写 `status.message` 拒绝）；其余 spec 变化路由给 Handler | `Reconcile` 幂等更新；只有语义字段变化才触发底层资源变更 |
| MLJob `spec.runPolicy.suspend=true` | 路由；若当前或新映射出的 phase 已是 `Succeeded` / `Failed`，终态优先且不写 cancel 成功信号；否则在 Handler 返回 suspend 完成结果后合并写入 `Suspended=True,reason=CancelRequested`，`phase` 维持 `Pending` | 执行原生 suspend（如 `(native, job)` patch `Job.spec.suspend=true`、`(native, podgroup)` patch `PodGroup.spec.minMember=0` 后驱逐 Pod）或 `Cleanup()` 删除底层资源；完成后返回结构化 suspend 结果，不直接写 `status` |
| MLJob DELETE | 不阻断 | 一般依赖 ownerReference 级联清理；Handler 仅清理跨 namespace / 外部副作用（外部存储句柄、跨集群资源等） |
| 底层资源事件（Pod / PodGroup / 第三方 CR） | 通过 ownerReference 反查到 MLJob 后路由 | `MapStatus` 纯函数计算新 phase；dispatcher 把结果合并写入 `status` |

**关键约束**：

- Handler **不引入 finalizer**；ownerReference 级联清理是默认路径
- `MapStatus` 必须是纯函数（不发起 K8s 调用），便于单元测试、状态回放与底层事件重算
- Handler 不能在 `Reconcile` 中直接写 `status`；`status` 的输入只能来自 `MapStatus` 返回值与 `Reconcile` 结构化结果，由 dispatcher 统一合并写入，保证写盘路径单一
- dispatcher 读取现有 `status` 后在代码中按 `condition.type` 合并 `conditions[]`，再通过 `status` subresource 使用 JSON merge patch 或 update 重试写回；CRD 不依赖 strategic merge patch 的 merge key 语义

**Pod 模板注入约定**（跨 Handler 通用，体现 [infra.md §8.3](../infra.md) 的 Quota 全覆盖不变式）：

| Pod 字段 / Label | 必填 | 取值 | 用途 |
| --- | --- | --- | --- |
| `spec.schedulerName` | 是 | `koord-scheduler` | 所有 AxisML workload Pod 一律走 koord-scheduler；不允许任何 backend 让 Pod 落到默认 kube-scheduler 上 |
| label `quota.scheduling.koordinator.sh/name` | 是 | `<spec.scheduling.quota>` | Koordinator 原生 quota 关联 label；ElasticQuota plugin 据此把该 Pod 计入 `status.used` |
| label `axisml.io/job-id` | 是 | `jobs.id`（UUID） | 反查 MLJob，与 CR 上同名 label 一致 |
| label `axisml.io/role` | 是 | role 名（如 `worker` / `master` / `launcher`） | 区分多角色拓扑下的 Pod |
| label `axisml.io/quota` | 是 | Compute Quota bare name（取自 MLJob CR `metadata.labels["axisml.io/quota"]` 透传，**与 `quota.scheduling.koordinator.sh/name` 取值不同**：前者是裸名如 `training`，后者是 ElasticQuota 全名如 `axisml-<tenant>-<pool>-training`） | AxisML 自有审计 / 查询；不参与调度 |
| label `axisml.io/replica-index` | 否 | role 内 0-based 序号 | 副本身份天然稳定时建议透传：StatefulSet 的 `apps.kubernetes.io/pod-index`、Indexed Job 的 `batch.kubernetes.io/job-completion-index`；NonIndexed Job、裸 Pod 拓扑下省略 |

前 5 项必填，所有 Handler 一律遵守；`axisml.io/replica-index` 只是可观测增强，缺失时 Compute §7.4 日志 API 退化为按 pod 名定位（详见 [compute.md §7.4](../compute.md)）。

## 7. Handler 接口契约

所有 Handler 必须实现以下行为（语言无关，描述责任）：

| 方法 | 责任 |
| --- | --- |
| `Key()` | 返回 `(backend, engine)` 元组，与 `spec.backend.{name, engine}` 对齐；用作注册表的主键 |
| `Validate(spec)` | 校验通用字段 + `backend.config` + role 集合（数量、命名）；纯函数，只返回 errors / warnings，不写 `status` |
| `Reconcile(ctx, mlJob)` | 创建 / 更新底层资源；幂等；通用字段由 Handler 自己注入到对应位置；处理 `spec.runPolicy.suspend` 的暂停语义；返回结构化结果（如 `suspendCompleted` / `reason` / warnings），不直接写 `status` |
| `MapStatus(underlying)` | 把后端原生状态映射回 §4 的四态 phase + message + startedAt / finishedAt + conditions + roles 副本聚合 |
| `Cleanup(ctx, mlJob)` | 删除底层资源。一般依赖 ownerReference 自动级联，Handler 仅负责需要主动清理的副本（跨 namespace 资源、外部存储句柄） |
| `WatchTargets()` | 声明本 Handler 需要 watch 的底层资源 GVK 列表，dispatcher 启动时统一建立 watch |
| `RequiredRBAC()` | 声明本 Handler 需要的 ClusterRole 规则；启动时聚合到 operator ServiceAccount |

**幂等性细则**：

- `Reconcile` 多次调用相同 spec 不重建底层资源；只有语义字段变化才触发更新
- `Cleanup` 对已删除资源返回 nil
- `MapStatus` 是纯函数（不发起 K8s 调用），状态推进只依赖输入参数

**Suspend 声明义务**：每个 Handler 必须在自身章节（§8）显式声明 "原生支持 / 兜底为 Cleanup"。dispatcher 不做静默选择——不支持原生 suspend 时必须显式调用 `Cleanup()`，避免半暂停半运行的中间态。**所有路径完成底层动作后都必须返回 `suspendCompleted=true, reason=CancelRequested`**（这是 dispatcher 写入 §4 cancel 闭环推进信号的唯一来源；缺失会导致 Compute PG 永远卡在 `Canceling`）；`status.phase` 在非终态 suspend 期间维持 `Pending`。若底层资源已经终态，终态优先，Handler 返回终态状态映射而不是 suspend 完成结果。

**Status 写入约束**：Handler 只能通过 `MapStatus` 的返回值与 `Reconcile` 的结构化结果影响 `status`；不能在 `Reconcile` 中直接 `status` 写盘。dispatcher 统一合并 `phase` / `message` / `startedAt` / `finishedAt` / `conditions` / `roles[]` 写入 CR，保证 [§2 写路径契约](#2-与-compute-的写路径契约) 中的 "status 单向权威"。

## 8. 内置 Handler

每个 Handler 章节统一按以下小节组织：**底层资源 / `backend.config` / 通用字段映射 / Status 映射 / Suspend / RBAC**。

### 8.1 `(native, job)`

底层用 K8s 原生 [`Job`](https://kubernetes.io/docs/concepts/workloads/controllers/job/)；适合不需要 gang scheduling 的单角色批处理场景。Pod 强制走 koord-scheduler 并通过 `quota.scheduling.koordinator.sh/name` label 计入 ElasticQuota（与所有 AxisML workload 一致，无"绕过 quota 的调度路径"）；本 Handler **不**创建 PodGroup（gang 不适用）。

**前置依赖**：集群已安装 Koordinator（提供 koord-scheduler 与 ElasticQuota plugin）。本 Handler 仅需要 `jobs.batch` 的 `create / get / list / watch / update / patch / delete`，不直接读写 ElasticQuota / PodGroup CR。

**底层资源**：

- 必填且仅一个 role（`name=worker`）；`Validate` 拒绝多 role 提交（多角色场景应选 `(kubeflow-trainer, *)`）
- 每个 MLJob 创建一个 K8s `Job`，Pod 由 Job controller 派生，但 Pod 模板上设置 `schedulerName: koord-scheduler` —— Pod 由 koord-scheduler 调度而非默认 kube-scheduler
- Job 设置 `ownerReference` 指向 MLJob，保证 MLJob 删除后底层资源级联清理（Pod 进一步由 Job 级联清理）

**Pod label**（在 `Job.spec.template.metadata.labels` 上注入；§6 Pod 注入约定的具体落地）：

- `quota.scheduling.koordinator.sh/name=<spec.scheduling.quota>` —— Koordinator quota 关联
- `axisml.io/job-id=<jobs.id>`
- `axisml.io/role=worker`
- `axisml.io/quota=<quota-name>` —— AxisML 自有追踪
- `axisml.io/replica-index=<0-based>`（**仅在 `backend.config.completionMode=Indexed` 时透传** K8s 注入的 `batch.kubernetes.io/job-completion-index`；默认 NonIndexed 模式下省略）

**`backend.config` 关键字段**：

```yaml
config:
  completionMode: NonIndexed | Indexed   # 默认 NonIndexed
  podFailurePolicy: {}                    # K8s Job 原生 podFailurePolicy 直通
```

**通用字段映射**：

| MLJob 字段 | Job 落点 |
| --- | --- |
| `roles[worker].template.image` / `imagePullPolicy` / `command` / `args` / `env` / `envFrom` / `workingDir` | Pod 主容器同名字段 |
| `roles[worker].template.resources.requests` / `limits` | Pod 主容器同名字段 |
| `roles[worker].replicas` | `Job.spec.parallelism` 与 `Job.spec.completions`（同值；Indexed 模式下 `completions` 表示总分片数） |
| `roles[worker].restartPolicy` | `Job.spec.template.spec.restartPolicy`（仅允许 `OnFailure` / `Never`） |
| `spec.scheduling.priorityClass` | Pod `spec.priorityClassName` |
| `spec.scheduling.nodeSelector` / `tolerations` | Pod 同名字段 |
| `spec.scheduling.quota` | Pod `spec.template.metadata.labels[quota.scheduling.koordinator.sh/name]`（ElasticQuota 全名 `axisml-<tenant>-<pool>-<quota>`）；不写入 Job 级别字段 |
| MLJob `metadata.labels[axisml.io/quota]` | Pod `spec.template.metadata.labels[axisml.io/quota]`（bare quota name，由 Compute 在 MLJob CR 上设置后由 Handler 透传） |
| 调度器选择 | Pod `spec.template.spec.schedulerName=koord-scheduler`（恒定） |
| `spec.runPolicy.activeDeadlineSeconds` | `Job.spec.activeDeadlineSeconds` |
| `spec.runPolicy.ttlSecondsAfterFinished` | `Job.spec.ttlSecondsAfterFinished` |
| `spec.runPolicy.backoffLimit` | `Job.spec.backoffLimit` |

**Status 映射**：

| K8s Job 条件 | MLJob phase |
| --- | --- |
| `status.active==0 && status.succeeded==0 && status.failed==0` | `Pending` |
| `status.active>0` | `Running` |
| `status.conditions[type=Complete,status=True]` | `Succeeded` |
| `status.conditions[type=Failed,status=True]` 或超 `activeDeadlineSeconds` | `Failed` |

`startedAt` 取 `Job.status.startTime`；`finishedAt` 取 `Job.status.completionTime`（终态时由 Job controller 写入）。`status.roles[worker]` 聚合 Job 上报的 active / succeeded / failed 副本数。

**Suspend**：原生支持。`spec.runPolicy.suspend=true` → patch `Job.spec.suspend=true`（K8s 原生字段，自动驱逐运行中的 Pod 并停止派生新 Pod），随后返回 `suspendCompleted=true, reason=CancelRequested`，由 dispatcher 写 `Suspended` condition 并让非终态 phase 维持 `Pending`；`suspend=false` → 反向 patch（Compute 模型下不会被触发，Cancelled 是终态、无 resume）。

**RBAC**：`jobs.batch` / `pods` / `events` 的 `create / get / list / watch / update / patch / delete`。

### 8.2 `(native, podgroup)`

将 MLJob 翻译为 sigs.k8s.io scheduler-plugins `PodGroup` + 裸 Pod，借助 Koordinator gang plugin 实现"全员就位才启动"的单角色任务（如分布式训练的多 Worker 同步启动）。Pod 同样走 koord-scheduler 并计入 ElasticQuota。

**前置依赖**：集群已安装 Koordinator；本 Handler 需要 `podgroups.scheduling.sigs.k8s.io` 的 `create / get / list / watch / update / patch / delete`。

**底层资源**：

- 必填且仅一个 role（`name=worker`）；`Validate` 拒绝多 role 提交（多角色场景走 `(kubeflow-trainer, *)`）
- 每个 MLJob 创建一个 `PodGroup`（`scheduling.sigs.k8s.io/v1alpha1`），`spec.minMember ← roles[worker].replicas`；可按需设置 `spec.minResources`
- 按 `roles[worker].replicas` 创建对应 Worker 裸 Pod；所有 Pod 设置 `schedulerName: koord-scheduler`，并通过 label `pod-group.scheduling.sigs.k8s.io=<podgroup-name>` 关联到 PodGroup
- PodGroup / Pod 设置 `ownerReference` 指向 MLJob，保证 MLJob 删除后底层资源级联清理
- operator **不**读写 ElasticQuota CR（ElasticQuota 由 Compute 独占维护，本 Handler 仅通过 Pod label `quota.scheduling.koordinator.sh/name` 引用）

**Pod label**（除 §6 通用 5 项之外）：

- `pod-group.scheduling.sigs.k8s.io=<podgroup-name>` —— gang 关联

裸 Pod 拓扑没有稳定 index，省略 `axisml.io/replica-index`；日志 API 通过 pod 名直接定位（详见 [compute.md §7.4](../compute.md)）。

**`backend.config`**：本 Handler 不消费；非空时 `Validate` 返回 warning，不报错（为 future 字段预留）。

**通用字段映射**：

| MLJob 字段 | Pod / PodGroup 落点 |
| --- | --- |
| `roles[worker].template.image` / `imagePullPolicy` / `command` / `args` / `env` / `envFrom` / `workingDir` | Pod 主容器同名字段 |
| `roles[worker].template.resources.requests` / `limits` | Pod 主容器同名字段 |
| `roles[worker].restartPolicy` | Pod `spec.restartPolicy` |
| `roles[worker].replicas` | `PodGroup.spec.minMember` 与裸 Pod 数 |
| `spec.scheduling.nodeSelector` / `tolerations` | Pod 同名字段 |
| `spec.scheduling.priorityClass` | Pod `spec.priorityClassName` |
| `spec.scheduling.quota` | Pod label `quota.scheduling.koordinator.sh/name`（ElasticQuota 全名）；PodGroup 不持有 quota 字段，由 koord-scheduler 通过 Pod label 关联 |
| MLJob `metadata.labels[axisml.io/quota]` | Pod label `axisml.io/quota`（bare quota name，透传自 MLJob CR 上同名 label） |
| 调度器选择 | Pod `spec.schedulerName=koord-scheduler`（恒定） |
| `spec.runPolicy.activeDeadlineSeconds` | Pod 同名字段 |
| `spec.runPolicy.ttlSecondsAfterFinished` | 终态后由 Handler 显式 GC（裸 Pod 无原生 TTL） |
| `spec.runPolicy.backoffLimit` | 通过 PodGroup 重试 + Handler 内部计数实现 |

**Status 映射**：

| 原生状态 | MLJob phase |
| --- | --- |
| 所有 Pod `Pending` 或 PodGroup 排队中 | `Pending` |
| 至少一个 Pod 进入 `Running` | `Running` |
| 所有 Pod `Succeeded` | `Succeeded` |
| 任一 Pod `Failed`、PodGroup 调度不可达、超 `activeDeadlineSeconds` | `Failed` |

`startedAt` 取首个 Pod `Running` 时间；`finishedAt` 取所有 Pod 进入终态的最晚时间。`status.roles[worker]` 聚合 Pod 数（active / ready / succeeded / failed）。

**Suspend**：原生支持。`spec.runPolicy.suspend=true` 时——

1. patch `PodGroup.spec.minMember=0`
2. 删除现存 Pod；后续 reconcile 看到 `spec.runPolicy.suspend=true` 后不再重建 Pod
3. 返回 `suspendCompleted=true, reason=CancelRequested`，由 dispatcher 写 `Suspended` condition 并让非终态 phase 维持 `Pending`

**顺序约束**：必须先 patch minMember=0、再删 Pod，否则 koord-scheduler 的 gang plugin 可能立即把刚被删除的 Pod 重新调度。`suspend=false` 时反向恢复 minMember 与 Pod（Compute 模型下不会被触发，Cancelled 是终态、无 resume）。

**RBAC**：`pods` / `podgroups.scheduling.sigs.k8s.io` / `events` 的 `create / get / list / watch / update / patch / delete`。

### 8.3 `(kubeflow-trainer, pytorchjob)`

将 MLJob 翻译为 Kubeflow Trainer 的 [`PyTorchJob`](https://www.kubeflow.org/docs/components/training/pytorch/) CR。PS / Worker、launcher / worker 等多角色 / 多 task 拓扑统一由本 Handler 通过 Kubeflow Trainer 承载。

**前置依赖**：集群已安装 kubeflow training-operator；其 RBAC 与 CRD 由 operator chart 单独管理。本 Handler 仅需要 `pytorchjobs.kubeflow.org` 的 `create / get / list / watch / update / patch / delete`。目标版本若不支持原生 `runPolicy.suspend`，本 Handler 必须在自身实现中显式 fallback 为 `Cleanup()`。

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
- 每个 replica 模板的 `template.spec.schedulerName` 必须设为 `koord-scheduler`；`template.metadata.labels` 必须注入 §6 列出的 5 项必填 label，并且对于多 worker 的 gang 语义可叠加 `pod-group.scheduling.sigs.k8s.io=<pg-name>` + 由 Handler 一并创建 PodGroup CR（`spec.minMember ← sum(replicas)` 或按 elastic 配置）
- `spec.scheduling.*` → 各 replica 模板内的 nodeSelector / tolerations
- `spec.scheduling.quota` → 各 replica 模板的 `template.metadata.labels[quota.scheduling.koordinator.sh/name]`
- `spec.runPolicy.suspend` → 支持原生 suspend 的版本 patch `PyTorchJob.spec.runPolicy.suspend=true`，否则走 `Cleanup()` fallback
- `spec.runPolicy.activeDeadlineSeconds` / `backoffLimit` → `PyTorchJob.spec.runPolicy` 同名字段

**Status 映射**：从 `PyTorchJob.status.conditions` 推导——

| PyTorchJob condition | MLJob phase |
| --- | --- |
| `Created` / `Restarting` | `Pending` |
| `Running` | `Running` |
| `Succeeded` | `Succeeded` |
| `Failed` | `Failed` |

**Suspend**：优先使用原生 `PyTorchJob.spec.runPolicy.suspend`；目标版本不支持时 fallback 为 `Cleanup()`。Handler 完成底层动作后返回 `suspendCompleted=true, reason=CancelRequested`，由 dispatcher 写 `Suspended` condition 并让非终态 phase 维持 `Pending`，作为 §4 cancel 闭环的推进信号。

> 完整字段映射、容错策略、与 elastic training 的交互细节，由独立设计文档落地（见 §11）。

### 8.4 `(kubeflow-trainer, tfjob)`

同 §8.3 思路，将 MLJob 翻译为 Kubeflow Trainer 的 `TFJob`。Role 集合约定为 `chief` / `worker` / `ps` / `evaluator`（任一可省略，replicas=0 表示禁用）。各 replica 模板同样必须注入 `schedulerName: koord-scheduler` + §6 必填 label；多角色 gang 通过同一 PodGroup（`minMember=sum(replicas)`）表达。Status 映射沿用 TFJob 的 condition 集，与 §8.3 PyTorchJob 同构。Suspend 优先走原生 `runPolicy.suspend`，目标版本不支持时 fallback 为 `Cleanup()`；Handler 完成底层动作后返回结构化 suspend 结果，由 dispatcher 统一写 `Suspended` condition。

### 8.5 `(kubeflow-trainer, mpijob)`

将 MLJob 翻译为 Kubeflow [`MPIJob`](https://www.kubeflow.org/docs/components/training/mpi/) CR。Role 集合约定为 `launcher`（replicas=1）+ `worker`（replicas≥1）。`backend.config` 携带 MPI 实现选择（OpenMPI / Intel MPI）与 launcher / worker 通讯参数。各 replica 模板同样必须注入 `schedulerName: koord-scheduler` + §6 必填 label；MPIJob 的 PodGroup 由本 Handler 创建（`minMember=launcher.replicas + worker.replicas`）。Status 映射对齐 MPIJob `status.conditions`。Suspend 优先走原生 `runPolicy.suspend`，目标版本不支持时 fallback 为 `Cleanup()`；Handler 完成底层动作后返回结构化 suspend 结果，由 dispatcher 统一写 `Suspended` condition。

### 8.6 `(custom, *)`

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

由 custom Handler 通过 unstructured client 创建并跟踪。**仍受 §6 Pod 注入约定与 [infra.md §8.3](../infra.md) Quota 全覆盖不变式约束**：custom Handler 必须保证最终落地的 Pod 模板上有 `schedulerName: koord-scheduler` 与 `quota.scheduling.koordinator.sh/name` label，否则 Validate 直接拒绝。完整 schema 与 unstructured 操作约定由独立设计文档落地（见 §11）。

## 9. RBAC 聚合

operator binary 启动时遍历 registry，把每个启用 Handler 的 `RequiredRBAC()` 合并去重，生成本 binary 实际需要的 ClusterRole rules。Helm chart 通过 values 控制启用集合，渲染最小化 RBAC 而非全集——例如仅启用 `(native, job)` 时不引入 PodGroup 相关 RBAC；启用 `(native, podgroup)` 才注入 `podgroups.scheduling.sigs.k8s.io` CRUD；启用 `(kubeflow-trainer, *)` 才注入对应 CR 的 RBAC。所有路径都不需要 ElasticQuota 的 RBAC（ElasticQuota 由 Compute 独占维护）。

## 10. 不变量与约束

- `spec.backend.{name, engine}` 创建后不可变；dispatcher 拒绝并写 `status.message`，admission webhook 后续可前置拦截
- `(backend, engine)` 元组未在 registry 注册 → MLJob 直接进入 `Failed`，message 写明缺失原因
- 任一 Handler 的 `Reconcile` / `MapStatus` 必须保持 §2 列出的写路径契约——这是把"插件"安全嵌入 Compute Outbox 模型的根基
- Handler 不直接修改 ElasticQuota CR；ElasticQuota CR 由 Compute 独占维护
- Handler 不向 Compute PG 写入任何数据；状态全部经由 MLJob `status` + Informer 回流
- **Handler 不引入 finalizer**；级联清理依赖 ownerReference + `Cleanup()`
- **`status.phase` 取值集合冻结为四态**（`Pending | Running | Succeeded | Failed`）；新增 phase 必须经 CRD schema 与 Compute 双侧同步演进
- 所有 Handler 派生的 Pod 必须满足 §6 Pod 注入约定的前 5 项必填字段；缺失任一项视为契约违反，Validate 必须在创建前拦截
- 所有 Handler 在 cancel 路径完成 suspend / Cleanup 后必须返回 `suspendCompleted=true, reason=CancelRequested`；dispatcher 据此写 `status.conditions[type=Suspended,status=True,reason=CancelRequested]`，非终态 `phase` 维持 `Pending`；这是 Compute 推进 `Canceling → Cancelled` 的唯一信号

## 11. 后续设计文档（不在本文档范围）

- `(native, job)` Handler 的 Indexed Job 模式与 `podFailurePolicy` 直通策略细节
- `(native, podgroup)` Handler 的 PodGroup `minResources` 与 elastic gang 演进
- `(kubeflow-trainer, pytorchjob / tfjob / mpijob / paddlejob / xgboostjob)` 各自的字段映射与状态映射细节，含每路径下的 PodGroup 创建策略
- `(custom, *)` Handler 的 `config` 完整 schema 与 unstructured 操作约定（包括 schedulerName / quota label 强制注入校验）
- Admission webhook：`spec.backend.{name, engine}` 不可变约束、`backend.config` 按 Handler 自带 schema 的统一校验
- Handler chart 的 values 控制（启用 / 禁用 backend，对应 RBAC 与 watch 的开关）
- CRD 严格 schema（启用 OpenAPI 校验，替换当前的 `x-kubernetes-preserve-unknown-fields`，含 `spec.backend.name` enum 收为 `{native, kubeflow-trainer, custom}` 与 `spec.scheduling.quota` 必填）
