# MLService Operator 详细设计

## 1. 概述

mlservice-operator 把 AxisML Compute 下发的 `MLService` CR 翻译为底层在线推理资源（Deployment + Service / KServe `InferenceService` / 自定义 GVK），并把执行状态回流到 `MLService.status`。它内部按 `spec.backend.{name, engine}` 二级元组路由到不同 Handler，由 Handler 真正生成底层资源、把后端原生状态映射回统一的 phase 集合（详见 [overview.md §5.3](../overview.md)）。

operator 与 Compute 的分工以 [compute.md §5 / §6.3.2](../compute.md) 为准；本文档只展开 operator 这一侧的实现契约。

## 2. 与 Compute 的写路径契约

Compute 采用 Operation Outbox + reconciler 异步下发 CR（详见 [compute.md §5.2](../compute.md)）。Operator 暴露给 Compute 的核心契约只有两条；其余约束（`backend.{name, engine}` 不可变、不引入 finalizer、`:scale` 唯一可变、无 suspend 语义等）分散在 §3.3 字段不可变性、§6 Reconcile 生命周期、§10 不变量与约束。

- **`Create` 幂等**：相同 `metadata.name` 的二次 `Create()` 返回 409 `AlreadyExists`，且不引发副作用（不重启 Pod、不重置 status）。Compute 收到 409 后会 `Get()` 现有 CR 并校验 label `axisml.io/service-id=<uuid>`；只有 label 一致才视为成功
- **status 单向权威**：operator 只写 `MLService.status`，Compute 只写 `MLService.metadata` / `MLService.spec`；状态推进由 Compute 侧 Informer 按 CR `status` 回流，operator 不感知 Compute 的 `services` 表，也不向 Compute PG 写入任何数据

## 3. CRD 契约

MLService 为 namespaced CR（CRD 定义见 `deploy/helm/axisml-system/crds/mlservice-crd.yaml`）：

| 字段 | 取值 |
| --- | --- |
| `apiVersion` | `axisml.io/v1alpha1` |
| `kind` | `MLService` |
| `scope` | `Namespaced`（创建在租户 namespace 下） |
| `shortNames` | `mls` |

Compute 负责设置以下 metadata（与 [compute.md §6.3.2](../compute.md) 对齐）：

- `metadata.name` ← `services.name`
- `metadata.namespace` ← `tenants.namespace`
- `metadata.labels["axisml.io/service-id"]` ← `services.id`（UUID，孤儿检测稳定锚点）
- `metadata.labels["axisml.io/tenant"]` ← 租户名
- `metadata.labels["axisml.io/queue"]` ← Compute Queue 名

### 3.1 spec 设计取舍

把 "角色拓扑" 提升为一等公民，与 mljob-operator §3.1 同源。Service 的执行域用 `roles[]` 数组承载：

- 单角色 backend（如 `(native, default)`）声明一个 role（约定 `name=predictor`）
- 多角色 backend（如 KServe `InferenceService` 的 `predictor` / `transformer` / `explainer`）声明多个 role
- role 名集合由各 Handler 在 §8 中约定，由 Handler 的 `Validate` 强制
- **MVP 仅允许 1 个 role**（`(native, default)` 强制 `predictor`）；多 role 接入推迟到对应 KServe Handler 的设计文档（§11）

调度域沿用 K8s PodSpec 扁平惯例直接放在 `spec.scheduling` 下，不再额外包一层 `placement`，与 mljob-operator 同构。

**与 MLJob 的差异点**：

- 顶层 `modelRef`：service 一等字段，指向 Catalog model version；Handler 据此把模型工件解析为容器侧的位置（环境变量 / volume mount / KServe `storageUri` 等）
- 顶层 `ports[]`：service 对外暴露的端口集合；Handler 据此生成 K8s Service 或 InferenceService predictor 的端口声明
- `runPolicy` 字段集合不同：service 是常驻 workload，**没有** `suspend` / `activeDeadlineSeconds` / `ttlSecondsAfterFinished` / `backoffLimit`；改为 `progressDeadlineSeconds`（rollout 进度超时，与 K8s Deployment 同名字段语义一致）

### 3.2 spec 结构

```yaml
apiVersion: axisml.io/v1alpha1
kind: MLService
spec:
  # ── 后端选择（创建后不可变；dispatcher 路由依据）─────────────────────
  backend:
    name: native              # 必填: native | kserve | custom
                              #      （kubeflow-trainer 仅用于 MLJob，不在 MLService 枚举内）
    engine: default           # 必填: 语义随 backend 而定（见 §8）
    config: {}                # 可选: 该 (backend, engine) 元组特有的 schemaless 配置

  # ── 调度域（由 Compute 从 Queue / ResourcePool / ResourceUnit 合成注入）──
  scheduling:
    queue: string             # 必填: Volcano Queue CR 名（与 Compute Queue 1:1 映射）
    priorityClass: string     # 可选: K8s PriorityClass 名
    nodeSelector: {}          # Compute 按 compute.md §6.2.3 合并 pool + unit 后注入
    tolerations: []           # 来自 ResourcePool

  # ── 模型引用（service 特有，指向 Catalog）──────────────────────────
  modelRef:
    name: string
    version: string

  # ── 对外暴露端口 ──────────────────────────────────────────────────
  ports:
    - name: http
      containerPort: 8080
      protocol: TCP            # 可选: TCP | UDP，默认 TCP

  # ── 执行域：roles 数组承载角色拓扑 ─────────────────────────────────
  roles:
    - name: predictor          # MVP 仅允许 1 个 role；role 标识在同一 MLService 内唯一
      replicas: 1              # >= 0；为 0 时视为待调度（status.phase=Pending）
      template:                # Pod template 子集：暴露常用字段，隐藏完整 PodSpec
        image: string
        imagePullPolicy: IfNotPresent  # 可选: IfNotPresent | Always | Never
        command: []            # 可选
        args: []                # 可选
        env: []                 # 可选: K8s EnvVar 数组
        envFrom: []             # 可选: K8s EnvFromSource 数组（ConfigMap / Secret 引用）
        workingDir: string      # 可选
        resources:
          requests: {}          # Compute 从 ResourceUnit.requests 注入
          limits: {}            # Compute 从 ResourceUnit.limits 注入

  # ── 生命周期 ────────────────────────────────────────────────────
  runPolicy:
    progressDeadlineSeconds: int   # 可选: rollout 进度超时；超时后 status.phase=Failed
```

### 3.3 字段归属与不可变性

| 字段路径 | 写入方 | 创建后可变？ |
| --- | --- | --- |
| `metadata.name` / `namespace` / `labels[axisml.io/*]` | Compute | 否 |
| `spec.backend.name` / `spec.backend.engine` | Compute（默认 `{native, default}`） | **否**；dispatcher 拒绝并写 `status.message` |
| `spec.backend.config` | Compute（默认 `{}`） | 视 Handler 语义；Handler 在 §7 `Validate` 中决定 |
| `spec.scheduling.queue` / `priorityClass` / `nodeSelector` / `tolerations` | Compute（合并 Queue + Pool + Unit） | 否 |
| `spec.modelRef` | 用户提交 | 否（更换模型版本走重建） |
| `spec.ports` | 用户提交 | 否 |
| `spec.roles[*].name` / `template.*`（除 resources） | 用户提交 | 否 |
| `spec.roles[*].template.resources` | Compute（注入 ResourceUnit） | 否 |
| `spec.roles[*].replicas` | API（`:scale` 触发） | **是**（扩缩容路径专用） |
| `spec.runPolicy.progressDeadlineSeconds` | 用户提交 | 否 |

**默认值注入**：用户未指定 `spec.backend` 时，Compute 写 CR 时显式补 `{name: native, engine: default}`；`backend.config` 默认空对象 `{}`。dispatcher 不接受 `backend.name` 或 `backend.engine` 为空。

**与 compute.md `services.replicas` 的兼容**：[compute.md §6.3.2](../compute.md) 中的 `services.replicas` 字段在 MVP 单 role 约定下定义为 `spec.roles[0].replicas`；`:scale` API 在 CR 侧 patch path 写 `spec/roles/0/replicas`。多 role 独立扩缩等真实需求出现后再扩展 :scale 契约（§11）。

**CRD schema 现状**：当前 CRD 的 `spec` / `status` 用 `x-kubernetes-preserve-unknown-fields: true`，重新设计字段无需 CRD bump；待行为稳定后再启用 OpenAPI schema 严格校验（§11）。

## 4. Status 契约

```yaml
status:
  observedGeneration: int64     # Handler reconcile 自洽用；Compute 不强消费
  phase: Pending | Ready | Degraded | Failed   # ← Compute 唯一消费的字段
  message: string               # 错误或状态附加信息（Compute 透传到 services.message）
  endpoint: string              # 对外服务地址（Compute 写入 services.endpoint）
  readyReplicas: int            # 主 role（MVP 即 roles[0]）就绪副本聚合（Compute 写入 services.ready_replicas）
  conditions:                   # K8s 标准 conditions（UI 可观测，Compute 不消费）
    - type: Initialized | Available | Progressing | Failed
      status: True | False | Unknown
      lastTransitionTime: timestamp
      reason: string
      message: string
  roles:                        # 各 role 副本聚合（UI 可观测，Compute 不消费）
    - name: string
      replicas: int             # spec 期望
      readyReplicas: int        # 通过 readiness probe
```

**phase 枚举冻结为四态**——`Pending | Ready | Degraded | Failed`。新增 phase 必须 CRD schema 与 Compute 双侧同步演进。Compute 的状态映射规则（与 [compute.md §6.3.2](../compute.md) 对齐）：

| MLService status.phase | services.status | 终态 |
| --- | --- | --- |
| `Pending` | `Pending` | 否 |
| `Ready` | `Ready` | 否 |
| `Degraded` | `Degraded` | 否，可恢复 |
| `Failed` | `Failed` | 否，可恢复（自愈） |

**`Pending / Ready / Degraded / Failed` 均为非终态**——operator 自愈（重建失败 Pod、健康检查恢复）后 Informer 回流可让 `Failed → Degraded → Ready` 自然恢复；只有 `Deleted` 是 Service 的最终终态，由 Compute Informer 在观察到 CR DELETE 事件后基于 PG 当前 `status` 推导（详见 [compute.md §5.3 / §5.4](../compute.md)），不由 operator 产出。

`conditions` 与 `roles[]` 是给 UI 与运维用的 observability 字段，Compute 不消费。这样既保留了 K8s 标准实践（`metav1.Condition`、per-role 副本聚合）又不污染 Compute 状态机的简洁性。

跨 Handler 的 phase 映射规则原则：所有 Handler 在 `MapStatus` 中负责把后端原生状态映射到这四态，映射表写入对应 Handler 章节（§8）。

## 5. 总体架构：Dispatcher + Handler

mlservice-operator 与 mljob-operator 同构，由两层组成：

- **Dispatcher Reconciler**：watch 所有 MLService CR，按 `spec.backend.{name, engine}` 元组路由到对应 Handler；本身不直接生成底层资源
- **Handler 注册表**：每个 `(backend, engine)` 元组对应一个 Handler 实例；编译期通过 `init()` 注册到全局 registry。Handler 负责把通用字段 + `backend.config` 翻译成具体的底层资源（Deployment + Service / InferenceService 等），并把后端原生状态映射回 MLService 统一 phase

```
                 ┌────────────────────────────────────┐
   MLService CR ─▶│  Dispatcher Reconciler            │
                  │  (按 (backend, engine) 路由)       │
                 └─────────────┬──────────────────────┘
                               │
       ┌───────────────────────┼───────────────────────┐
       ▼                       ▼                       ▼
 (native,default)       (kserve,triton)          (custom, *)
     Handler                Handler                  Handler
 (Deployment+Service) (InferenceService CR)  (用户声明的 GVK)
       │                       │                       │
       └───────────────────────┴───────────────────────┘
                               │
                       status backflow
                               │
                               ▼
                  MLService.status (统一 phase)
```

**Watch 拓扑**：Dispatcher 始终 watch MLService 主队列；每个 Handler 启动时通过 `WatchTargets()` 声明自己关心的底层资源类型（Deployment、Service、PodGroup、InferenceService …），由 dispatcher 统一建立 watch（controller-runtime `Watches()`），事件通过 `ownerReference` 反查回 MLService 后再交给对应 Handler 的 `MapStatus`。

**注册方式**：编译期注册（每个 handler 包 `init()` → 全局 registry）。MVP 不引入运行时插件加载（plugin / wasm / 外部 grpc）。后续若需要 "运行时安装新后端"，再演进为独立 operator binary 路由模式。

**未注册组合的兜底**：dispatcher 收到一条 `(backend, engine)` 无 handler 的 MLService → 写 `status.phase=Failed`、`status.message="no handler for backend=X engine=Y"`，不创建任何底层资源。

## 6. Reconcile 生命周期

按事件源切分 dispatcher 与 Handler 的职责：

| 事件 | Dispatcher 行为 | Handler 行为 |
| --- | --- | --- |
| MLService ADD（首次创建） | 路由到 Handler；调用 `Validate(spec)`，校验失败写 `status.phase=Failed` | `Reconcile(ctx, mlService)` 创建底层资源，设置 `ownerReference: MLService` |
| MLService UPDATE（仅 `roles[*].replicas` 变更，来自 `:scale`） | 路由 | `Reconcile` 透传为后端资源副本调整；不重建 Pod |
| MLService UPDATE（其他 spec 字段变更） | 校验 `backend.{name, engine}` 不变；其他字段变更属于约束违反，写 `status.message` 拒绝 | 不动 |
| MLService DELETE | 不阻断 | 一般依赖 ownerReference 级联清理；Handler 仅清理跨 namespace / 外部副作用（外部存储句柄、跨集群资源等） |
| 底层资源事件（Deployment / Service / PodGroup / 第三方 CR） | 通过 ownerReference 反查到 MLService 后路由 | `MapStatus` 纯函数计算新 phase；dispatcher 把结果合并写入 `status` |

**关键约束**：

- Handler **不引入 finalizer**；ownerReference 级联清理是默认路径
- `MapStatus` 必须是纯函数（不发起 K8s 调用），便于未来在 admission webhook 中复用
- Handler 不能在 `Reconcile` 中直接写 `status`；所有 status 变更必须经过 `MapStatus`，由 dispatcher 统一合并写入，保证回流路径单一

**Pod label 约定**（跨 Handler 通用）：

| Label | 取值 | 用途 |
| --- | --- | --- |
| `axisml.io/service-id` | `services.id`（UUID） | 反查 MLService，与 CR 上同名 label 一致 |
| `axisml.io/role` | role 名（如 `predictor` / `transformer` / `explainer`） | 区分多角色拓扑下的 Pod |
| `axisml.io/replica-index` | role 内 0-based 序号 | 副本级观测与排障；不可寻址副本（KServe autoscaling pod）允许省略 |

凡是管理可寻址副本的 Handler 都必须打 `service-id` 与 `role` 两件套；`replica-index` 在副本身份稳定的场景（如 native Deployment）必须打，autoscaling 主导的场景可省略。

## 7. Handler 接口契约

所有 Handler 必须实现以下行为（语言无关，描述责任）：

| 方法 | 责任 |
| --- | --- |
| `Key()` | 返回 `(backend, engine)` 元组，与 `spec.backend.{name, engine}` 对齐；用作注册表的主键 |
| `Validate(spec)` | 校验通用字段 + `backend.config` + role 集合（数量、命名）；纯函数，未来可被 admission webhook 复用 |
| `Reconcile(ctx, mlService)` | 创建 / 更新底层资源；幂等；通用字段由 Handler 自己注入到对应位置；处理 `spec.roles[*].replicas` 的扩缩容 |
| `MapStatus(underlying)` | 把后端原生状态映射回 §4 的四态 phase + readyReplicas + endpoint + message + conditions + roles 副本聚合 |
| `Cleanup(ctx, mlService)` | 删除底层资源。一般依赖 ownerReference 自动级联，Handler 仅负责需要主动清理的副本（跨 namespace 资源、外部存储句柄） |
| `WatchTargets()` | 声明本 Handler 需要 watch 的底层资源 GVK 列表，dispatcher 启动时统一建立 watch |
| `RequiredRBAC()` | 声明本 Handler 需要的 ClusterRole 规则；启动时聚合到 operator ServiceAccount |

**幂等性细则**：

- `Reconcile` 多次调用相同 spec 不重建底层资源；只有语义字段变化才触发更新
- 重复 patch 相同 `roles[*].replicas` 不得重建底层资源；只调整副本数
- `Cleanup` 对已删除资源返回 nil
- `MapStatus` 是纯函数（不发起 K8s 调用），状态推进只依赖输入参数

**Scale 透传义务**：每个 Handler 必须把 `roles[*].replicas` 透传为后端原生扩缩——

- native → patch `Deployment.spec.replicas`，同步更新 `PodGroup.spec.minMember`
- kserve → patch `InferenceService.spec.predictor.{minReplicas, maxReplicas}`（具体策略见 §8.2）
- 不支持原生扩缩的 backend → 兜底为重建底层资源（应避免，作为最后手段）

**Status 写入约束**：Handler 只能通过 `MapStatus` 的返回值影响 `status`；不能在 `Reconcile` 中直接 `status` 写盘。dispatcher 统一合并 `phase` / `message` / `endpoint` / `readyReplicas` / `conditions` / `roles[]` 写入 CR，保证 [§2 写路径契约](#2-与-compute-的写路径契约) 中的 "status 单向权威"。

## 8. 内置 Handler

每个 Handler 章节统一按以下小节组织：**底层资源 / `backend.config` / 通用字段映射 / Status 映射 / Scale / RBAC**。

### 8.1 `(native, default)` —— MVP

MVP 唯一落地的 Handler，保留 Deployment + Service + PodGroup 行为。

**底层资源**：

- 必填且仅一个 role（`name=predictor`）；`Validate` 拒绝多 role 提交或其他 role 名
- 每个 MLService 创建一个 Deployment、一个 K8s Service、一个 Volcano `PodGroup`
- Deployment Pod 设置 `schedulerName: volcano`；`PodGroup.spec.queue ← spec.scheduling.queue`，`PodGroup.spec.minMember ← roles[predictor].replicas`
- Deployment / Service / PodGroup 设置 `ownerReference` 指向 MLService，保证 MLService 删除后底层资源级联清理
- operator 不读写 Volcano Queue CR（Queue 由 Compute 独占维护）

**Pod label**：

- `axisml.io/service-id=<services.id>`
- `axisml.io/role=predictor`
- `axisml.io/replica-index=<0-based>`

**`backend.config`**：本 Handler 不消费；非空时 `Validate` 写 warning，不报错（为 future 字段预留）。

**通用字段映射**：

| MLService 字段 | Deployment / Service / PodGroup 落点 |
| --- | --- |
| `roles[predictor].template.image` / `imagePullPolicy` / `command` / `args` / `env` / `envFrom` / `workingDir` | Deployment Pod 主容器同名字段 |
| `roles[predictor].template.resources.requests` / `limits` | Deployment Pod 主容器同名字段 |
| `roles[predictor].replicas` | `Deployment.spec.replicas`、`PodGroup.spec.minMember` |
| `spec.scheduling.queue` | `PodGroup.spec.queue` |
| `spec.scheduling.priorityClass` | Pod `spec.priorityClassName` |
| `spec.scheduling.nodeSelector` / `tolerations` | Pod 同名字段 |
| `spec.modelRef` | Catalog client 解析为模型工件 URI；MVP 注入为环境变量 `AXISML_MODEL_URI`（containerPath / volume mount 形态留待后续策略） |
| `spec.ports[]` | Deployment Pod 容器 `ports` + K8s Service `spec.ports`（`targetPort` 取 `containerPort`） |
| `spec.runPolicy.progressDeadlineSeconds` | `Deployment.spec.progressDeadlineSeconds` |

**Status 映射**（沿用 [compute.md §6.3.2](../compute.md) 规则，从 Deployment `status` 推导）：

| 条件 | MLService phase |
| --- | --- |
| `desired_replicas == 0` | `Pending`（扩缩至 0，视为待调度 / 停用） |
| `ready_replicas == desired_replicas && desired_replicas > 0` | `Ready` |
| `0 < ready_replicas < desired_replicas` | `Degraded` |
| `ready_replicas == 0 && desired_replicas > 0` | `Failed` |

`endpoint` 取 K8s Service ClusterIP DNS（`<svc>.<namespace>.svc.cluster.local:<port>`，`port` 取 `spec.ports[0]`）。`readyReplicas` 取 Deployment `status.readyReplicas`；`status.roles[predictor]` 聚合 desired / ready 副本数。

**Scale**：patch `Deployment.spec.replicas` + `PodGroup.spec.minMember`；不重建 Pod。

**RBAC**：`deployments.apps` / `services` / `podgroups.scheduling.volcano.sh` / `pods` / `events` 的 `create / get / list / watch / update / patch / delete`。

### 8.2 `(kserve, triton)` —— 占位

将 MLService 翻译为 KServe `InferenceService` CR，predictor 使用 Triton。

**前置依赖**：集群已安装 KServe；其 RBAC 与 CRD 由 KServe chart 单独管理，本 Handler 仅需要 `inferenceservices.serving.kserve.io` 的 `create / get / list / watch / update / patch / delete`。

**Role 集合约定**：MVP 仅允许 `predictor`（replicas≥0）；未来扩展可选 `transformer` / `explainer`，由 Handler 在自身设计文档中明确开放节奏。

**`backend.config` 关键字段**（schema 待细化设计文档）：

```yaml
config:
  predictor:
    minReplicas: int          # 默认 = roles[predictor].replicas
    maxReplicas: int          # 自动扩缩上限；不填则等于 minReplicas
    scaleToZero: bool         # 是否允许 scale-to-zero
    protocolVersion: v1 | v2  # KServe 推理协议
  storageUri: string          # 模型工件位置；可由 Catalog 通过 modelRef 自动解析
  containerOverrides: {}      # 容器级别 override（command / args / env）
```

**通用字段映射**：

- `roles[predictor].template.image` → 写入 predictor 容器（Triton 默认镜像可由 KServe runtime 模板提供，Handler 用作 override）
- `roles[predictor].template.{command, args, env, envFrom, workingDir}` → predictor 容器同名字段
- `roles[predictor].template.resources` → predictor `resources`
- `roles[predictor].replicas` → 写入 `predictor.minReplicas`；若未设置 `config.predictor.maxReplicas`，则同时写入 `maxReplicas`
- `spec.modelRef` → 通过 Catalog 解析为 `predictor.storageUri`
- `spec.scheduling.queue` → KServe pod 通过 podSpec 注入 `schedulerName: volcano` + 关联 PodGroup（具体集成方式以 KServe + Volcano 集成文档为准）
- `spec.scheduling.priorityClass` / `nodeSelector` / `tolerations` → predictor 同名字段
- `spec.ports[]` → predictor 容器 ports；KServe 自身负责生成对外 Route
- `spec.runPolicy.progressDeadlineSeconds` → KServe 暂无对等字段，Handler 在 Validate 中写 warning

**未来 transformer / explainer 角色映射**至 `roles[transformer]` / `roles[explainer]`，字段映射镜像 predictor，MVP 单 role 不开放。

**Status 映射**：从 `InferenceService.status.conditions` 推导——

| InferenceService condition | MLService phase |
| --- | --- |
| `PredictorReady=False` 且 `desired==0` | `Pending` |
| `Ready=True` | `Ready` |
| `PredictorReady=False` 且 `0 < ready < desired` | `Degraded` |
| `Ready=False` 且 `ready==0 && desired>0` | `Failed` |

`endpoint` 取 `status.url`。

**Scale**：patch `InferenceService.spec.predictor.{minReplicas, maxReplicas}`；具体取舍（min 跟随 / max 联动）由独立设计文档落地。

**Quota 与 autoscaling 的相互作用**：KServe scale-to-zero / 自动扩缩可能让实际副本数动态变化；Compute quota 按 `maxReplicas × requests` 上限计费（与 native 的 "replicas × requests" 线性记账一致），保证账面与运行时不打架。具体细节由独立设计文档落地（§11）。

### 8.3 `(kserve, tfserving / torchserve / sklearn / huggingface)` —— 占位

同 §8.2 思路，将 MLService 翻译为 KServe InferenceService 的对应 predictor 类型；`config` 携带 framework 特有字段（如 huggingface 的 `task` / `modelId`、torchserve 的 model store 路径）。Role 集合、字段映射、状态映射、Scale 策略均沿用 §8.2 表格。

### 8.4 `(custom, *)` —— 占位

为外部接入预留的一等公民。用户在 `backend.config` 中以 schemaless 方式描述目标 GVK 与字段映射：

```yaml
backend:
  name: custom
  engine: any-name
  config:
    target:
      apiVersion: example.com/v1
      kind: MyServingEndpoint
    fieldMappings:
      "spec.image":    "$.roles[?(@.name=='predictor')].template.image"
      "spec.replicas": "$.roles[?(@.name=='predictor')].replicas"
      # ...
    statusMappings:
      "$.status.phase":
        Pending: Pending
        Active:  Ready
        Degraded: Degraded
        Error:   Failed
    endpointPath: "$.status.url"
```

由 custom Handler 通过 unstructured client 创建并跟踪。**MVP 不实现**——当首个第三方后端无法纳入 `(kserve, *)` 时再设计完整 schema。

## 9. RBAC 聚合

operator binary 启动时遍历 registry，把每个启用 Handler 的 `RequiredRBAC()` 合并去重，生成本 binary 实际需要的 ClusterRole rules。Helm chart 通过 values 控制启用集合，渲染最小化 RBAC 而非全集。MVP 仅启用 `(native, default)`，对应 RBAC 限定为 §8.1 列出的资源。

## 10. 不变量与约束

- `spec.backend.{name, engine}` 创建后不可变；dispatcher 拒绝并写 `status.message`，admission webhook 后续接管
- `(backend, engine)` 元组未在 registry 注册 → MLService 直接进入 `Failed`，message 写明缺失原因
- 任一 Handler 的 `Reconcile` / `MapStatus` 必须保持 §2 列出的写路径契约——这是把 "插件" 安全嵌入 Compute Outbox 模型的根基
- Handler 不直接修改 Volcano Queue CR；Queue CR 由 Compute 独占维护
- Handler 不向 Compute PG 写入任何数据；状态全部经由 MLService `status` + Informer 回流
- **Handler 不引入 finalizer**；级联清理依赖 ownerReference + `Cleanup()`
- **`status.phase` 取值集合冻结为四态**（`Pending | Ready | Degraded | Failed`）；新增 phase 必须经 CRD schema 与 Compute 双侧同步演进
- **`spec.roles[*].replicas` 是允许变更的字段**（`:scale` 路径专用）；其余 spec 字段创建后不可变，dispatcher 检测到变更需写 `status.message` 拒绝
- 管理可寻址副本的 Handler 必须打 `axisml.io/service-id` / `axisml.io/role` label；副本身份稳定的场景必须叠加 `axisml.io/replica-index`

## 11. 后续设计文档（不在本文档范围）

- KServe Handler 多 role（`predictor` / `transformer` / `explainer`）字段映射与状态映射细节
- KServe scale-to-zero 与 Volcano quota 的精细交互模型（含 `maxReplicas × requests` 上限计费策略）
- `(custom, *)` Handler 的 `config` 完整 schema 与 unstructured 操作约定
- 多 role 独立扩缩容的 `:scale` API 扩展（路径中携带 role 名）
- Admission webhook：`spec.backend.{name, engine}` 不可变约束、`backend.config` 按 Handler 自带 schema 的统一校验
- Handler chart 的 values 控制（启用 / 禁用 backend，对应 RBAC 与 watch 的开关）
- CRD 严格 schema（启用 OpenAPI 校验，替换当前的 `x-kubernetes-preserve-unknown-fields`）
