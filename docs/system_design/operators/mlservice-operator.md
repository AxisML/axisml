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

- 单角色 backend（如 `(native, deployment)` / `(native, statefulset)`）声明一个 role（约定 `name=predictor`）
- 多角色 backend（如 KServe `InferenceService` 的 `predictor` / `transformer` / `explainer`）声明多个 role
- role 名集合由各 Handler 在 §8 中约定，由 Handler 的 `Validate` 强制
- 单角色 Handler（`(native, deployment)` / `(native, statefulset)` 等）`Validate` 拒绝多 role 提交；多角色 Handler 在自身章节中明确开放节奏

调度域沿用 K8s PodSpec 扁平惯例直接放在 `spec.scheduling` 下，不再额外包一层 `placement`，与 mljob-operator 同构。

**Service 不引入 `volcano` backend**：与 MLJob 不同，service 不需要 gang scheduling（常驻 + 弹性扩缩），故 `native` 直接走 K8s 原生 Deployment / StatefulSet，不引入"`(volcano, *)` MLService backend"这种独立 backend 维度。**但 native Service Pod 仍走 Volcano 调度器并附带轻量 PodGroup**：每个 MLService 由 Handler 创建一个 minMember=`roles[0].replicas` 的 PodGroup（不要求 gang，仅作为 Volcano Queue 资源记账的锚点），Pod 设置 `schedulerName: volcano` 并通过 annotation 关联到该 PodGroup；Volcano Queue `status.allocated` 因此自然包含 MLService 用量，[compute.md §6.2.4](../compute.md) 的 `queues.used` 反映 Job + Service 的合计用量，无需 Compute 自行合成。`(kserve, *)` 的 Pod 由 KServe 自身派生，是否接入 Volcano 调度由独立设计文档决定（见 §11）。

**与 MLJob 的差异点**：

- 顶层 `modelRef`：service 一等字段，指向 Catalog model version；Handler 据此把模型工件解析为容器侧的位置（环境变量 / volume mount / KServe `storageUri` 等）
- `roles[*].template.ports[]`：与 K8s `PodSpec.containers[].ports` 同源约定。每个 role 是一个独立的 Deployment / StatefulSet（或 InferenceService 内的 component），各自的容器端口属于该 role 自身——这与多 role 拓扑（KServe transformer/explainer、PD 分离的 prefill/decode/router）天然一致。Handler 据此为每个 role 派生一个 K8s Service（targetPort=containerPort）。早期方案曾把 `ports[]` 放在 spec 顶层，是单 role 退化形态下的便捷写法，但在多 role 模型下"顶层 ports 到底属于哪个 role"无法回答，故下沉
- 顶层 `route`：可选；与 Gateway API `HTTPRoute` 同源命名。当 `enabled=true` 时由 Handler 创建 namespaced `HTTPRoute`（搭配 Envoy Gateway 的 `SecurityPolicy` / `BackendTrafficPolicy`）实现自助外部入口，`backendRefs` 指向 `route.targetRole` 对应的 K8s Service，详见 §8.1。`route.enabled` 还会切换 `status.endpoint` 的语义：`false` 时为集群内 Service DNS、`true` 时为外部 URL（详见 §4）。`(kserve, *)` Handler 自带 Route 机制，不接受 `route.enabled=true`
- `runPolicy` 字段集合不同：service 是常驻 workload，**没有** `suspend` / `activeDeadlineSeconds` / `ttlSecondsAfterFinished` / `backoffLimit`；改为 `progressDeadlineSeconds`（rollout 进度超时，与 K8s Deployment 同名字段语义一致）

### 3.2 spec 结构

```yaml
apiVersion: axisml.io/v1alpha1
kind: MLService
spec:
  # ── 后端选择（创建后不可变；dispatcher 路由依据）─────────────────────
  backend:
    name: native              # 必填: native | kserve | custom
                              #      （volcano / kubeflow-trainer 仅用于 MLJob）
    engine: deployment        # 必填: 语义随 backend 而定（见 §8）；engine 与目标 CR 1:1 映射
                              #   native: deployment | statefulset
                              #   kserve: inference | llminference
                              #          （inference → InferenceService CR；llminference → LLMInferenceService CR；
                              #           runtime 由 backend.config.runtime 在 inference 下选择）
                              #   custom: 任意名（由 backend.config 描述目标 GVK）
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

  # ── 执行域：roles 数组承载角色拓扑 ─────────────────────────────────
  roles:
    - name: predictor          # 单角色 Handler 仅允许 1 个 role；role 标识在同一 MLService 内唯一
      replicas: 1              # >= 0；为 0 时视为待调度（status.phase=Pending）
      template:                # Pod template 子集：暴露常用字段，隐藏完整 PodSpec
        image: string
        imagePullPolicy: IfNotPresent  # 可选: IfNotPresent | Always | Never
        command: []            # 可选
        args: []                # 可选
        env: []                 # 可选: K8s EnvVar 数组
        envFrom: []             # 可选: K8s EnvFromSource 数组（ConfigMap / Secret 引用）
        workingDir: string      # 可选
        ports:                  # 与 K8s containers[].ports 同源；Handler 据此派生该 role 的 K8s Service
          - name: http
            containerPort: 8080
            protocol: TCP        # 可选: TCP | UDP，默认 TCP
        resources:
          requests: {}          # Compute 从 ResourceUnit.requests 注入
          limits: {}            # Compute 从 ResourceUnit.limits 注入

  # ── 生命周期 ────────────────────────────────────────────────────
  runPolicy:
    progressDeadlineSeconds: int   # 可选: rollout 进度超时；超时后 status.phase=Failed

  # ── 对外路由（可选；默认仅 ClusterIP；与 Gateway API HTTPRoute 同源）────
  route:
    enabled: false             # 默认 false：仅 ClusterIP；true 时创建 HTTPRoute 等资源
    targetRole: string         # 单 role 可省（自动取唯一 role 名）；多 role 必填
                               # 指明哪个 role 的 K8s Service 作为 HTTPRoute backendRef
    portName: string           # 可选: 选取 roles[targetRole].template.ports[] 中的端口名
                               # 默认取 ports[0].name；多端口时必须显式指定
    hostname: string           # 可选: 外部主机名；不填则继承 Gateway 监听器配置
    path: string               # 可选: HTTPRoute 路径前缀，默认 "/"
    auth:                      # 可选: 认证策略 → SecurityPolicy
      type: none | jwt | apiKey  # 默认 none
      jwt:                       # type=jwt 时必填
        issuer: string
        jwksUri: string
      apiKey:                    # type=apiKey 时必填
        secretRef:               # 同 namespace Secret，包含 API key 列表
          name: string
    rateLimit:                 # 可选: 限流 → BackendTrafficPolicy
      requestsPerSecond: int
      burst: int
    timeout: string            # 可选: 请求超时（Go duration，如 "30s"）
```

### 3.3 字段归属与不可变性

| 字段路径 | 写入方 | 创建后可变？ |
| --- | --- | --- |
| `metadata.name` / `namespace` / `labels[axisml.io/*]` | Compute | 否 |
| `spec.backend.name` / `spec.backend.engine` | Compute（默认 `{native, deployment}`） | **否**；dispatcher 拒绝并写 `status.message` |
| `spec.backend.config` | Compute（默认 `{}`） | 视 Handler 语义；Handler 在 §7 `Validate` 中决定 |
| `spec.scheduling.queue` / `priorityClass` / `nodeSelector` / `tolerations` | Compute（合并 Queue + Pool + Unit） | 否 |
| `spec.modelRef` | 用户提交 | 否（更换模型版本走重建） |
| `spec.roles[*].name` / `template.*`（含 `ports[]`，除 resources） | 用户提交 | 否 |
| `spec.roles[*].template.resources` | Compute（注入 ResourceUnit） | 否 |
| `spec.roles[*].replicas` | API（`:scale` 触发） | **是**（扩缩容路径专用） |
| `spec.runPolicy.progressDeadlineSeconds` | 用户提交 | 否 |
| `spec.route`（整块，含 `enabled` / `targetRole` / `portName` / `hostname` / `path` / `auth` / `rateLimit` / `timeout`） | 用户提交 | 否（v1 不可变；mutable 演进见 §11） |

**默认值注入**：用户未指定 `spec.backend` 时，Compute 写 CR 时显式补 `{name: native, engine: deployment}`；`backend.config` 默认空对象 `{}`。dispatcher 不接受 `backend.name` 或 `backend.engine` 为空。

**`spec.route` 与 backend 的兼容性**：`(kserve, *)` Handler 在 `Validate` 中拒绝 `spec.route.enabled=true` 的提交（KServe `InferenceService` 自带对外 Route，避免双管）；`(native, *)` 与 `(custom, *)` 接受。详见各 Handler 章节。

**与 compute.md `services.replicas` 的兼容**：[compute.md §6.3.2](../compute.md) 中的 `services.replicas` 字段在单 role 约定下定义为 `spec.roles[0].replicas`；`:scale` API 在 CR 侧 patch path 写 `spec/roles/0/replicas`。多 role 独立扩缩的契约扩展见 §11。

**CRD schema 现状**：当前 CRD 的 `spec` / `status` 用 `x-kubernetes-preserve-unknown-fields: true`，重新设计字段无需 CRD bump；待行为稳定后再启用 OpenAPI schema 严格校验（§11）。

## 4. Status 契约

```yaml
status:
  observedGeneration: int64     # Handler reconcile 自洽用；Compute 不强消费
  phase: Pending | Ready | Degraded | Failed   # ← Compute 唯一消费的字段
  message: string               # 错误或状态附加信息（Compute 透传到 services.message）
  endpoint: string              # 单一对外服务地址（Compute 写入 services.endpoint）；按 spec.route.enabled 二分：
                                #   - route.enabled=false（默认）→ K8s Service DNS（<svc>.<ns>.svc.cluster.local:<port>）
                                #     ClusterIP Service / headless Service 共用此格式
                                #   - route.enabled=true            → 外部 URL（形如 https://<hostname><path>）
                                # role 选择：单 role 取唯一 role 的 Service；多 role 取 spec.route.targetRole；
                                #          未设置 spec.route 的多 role 场景由各 Handler 在 §8 中约定主 role
                                # 端口选择：route.enabled=true 时按 route.portName；否则取主 role.template.ports[]
                                #          中 name="http" 的端口；不存在时取 ports[0] 并加 warning condition
  readyReplicas: int            # 主 role（单 role 约定下即 roles[0]）就绪副本聚合（Compute 写入 services.ready_replicas）
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
 (native,deployment)    (kserve,inference)         (custom, *)
     Handler                Handler                  Handler
 (Deployment+Service) (InferenceService CR)    (用户声明的 GVK)
       │                       │                       │
       └───────────────────────┴───────────────────────┘
                               │
                       status backflow
                               │
                               ▼
                  MLService.status (统一 phase)
```

**Watch 拓扑**：Dispatcher 始终 watch MLService 主队列；每个 Handler 启动时通过 `WatchTargets()` 声明自己关心的底层资源类型（Deployment、Service、PodGroup、InferenceService …），由 dispatcher 统一建立 watch（controller-runtime `Watches()`），事件通过 `ownerReference` 反查回 MLService 后再交给对应 Handler 的 `MapStatus`。

**注册方式**：编译期注册（每个 handler 包 `init()` → 全局 registry）；不引入运行时插件加载（plugin / wasm / 外部 grpc）——若未来需要 "运行时安装新后端"，再演进为独立 operator binary 路由模式。

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
- dispatcher 通过 `status` subresource 做 strategic merge patch；`conditions[]` 按 K8s 标准 merge-by-`type` 语义合并，不会全量覆盖

**Pod label 约定**（跨 Handler 通用）：

| Label | 必填 | 取值 | 用途 |
| --- | --- | --- | --- |
| `axisml.io/service-id` | 是 | `services.id`（UUID） | 反查 MLService，与 CR 上同名 label 一致 |
| `axisml.io/role` | 是 | role 名（如 `predictor` / `transformer` / `explainer`） | 区分多角色拓扑下的 Pod |
| `axisml.io/replica-index` | 否 | role 内 0-based 序号 | 副本身份天然稳定时建议透传：`(native, statefulset)` 取 `apps.kubernetes.io/pod-index`；`(native, deployment)` / KServe autoscaling pod 集合等无稳定身份场景一律省略 |

`service-id` + `role` 两件套必填；`replica-index` 是可观测增强，缺失时按 pod 名定位。MLService 当前无 logs API，本约定主要服务于运维排障与 metrics 聚合。

**`spec.route` 派生资源**：当 `enabled=true` 时，Handler 在租户 namespace 内创建 / 更新以下资源（统一打 `axisml.io/service-id` label，并设置 `ownerReference: MLService`，靠级联清理删除，不引入 finalizer）：

- `HTTPRoute`（`gateway.networking.k8s.io/v1`）：`parentRefs` 指向 `axisml-gateway`（跨 namespace 引用通过 `ReferenceGrant` 授权，由 infra chart 准备），`backendRefs` 指向 `route.targetRole` 对应的 K8s Service
- `SecurityPolicy`（`gateway.envoyproxy.io/v1alpha1`）：仅当 `auth.type != none` 时创建，`targetRefs` 指向上面的 HTTPRoute
- `BackendTrafficPolicy`（`gateway.envoyproxy.io/v1alpha1`）：仅当 `rateLimit` 或 `timeout` 非空时创建，`targetRefs` 指向上面的 HTTPRoute

## 7. Handler 接口契约

所有 Handler 必须实现以下行为（语言无关，描述责任）：

| 方法 | 责任 |
| --- | --- |
| `Key()` | 返回 `(backend, engine)` 元组，与 `spec.backend.{name, engine}` 对齐；用作注册表的主键 |
| `Validate(spec)` | 校验通用字段 + `backend.config` + role 集合（数量、命名）；纯函数，未来可被 admission webhook 复用 |
| `Reconcile(ctx, mlService)` | 创建 / 更新底层资源；幂等；通用字段由 Handler 自己注入到对应位置；处理 `spec.roles[*].replicas` 的扩缩容 |
| `MapStatus(underlying)` | 把后端原生状态映射回 §4 的四态 phase + readyReplicas + 单一 endpoint（按 `spec.route.enabled` 二分内/外） + message + conditions + roles 副本聚合 |
| `Cleanup(ctx, mlService)` | 删除底层资源。一般依赖 ownerReference 自动级联，Handler 仅负责需要主动清理的副本（跨 namespace 资源、外部存储句柄） |
| `WatchTargets()` | 声明本 Handler 需要 watch 的底层资源 GVK 列表，dispatcher 启动时统一建立 watch |
| `RequiredRBAC()` | 声明本 Handler 需要的 ClusterRole 规则；启动时聚合到 operator ServiceAccount |

**幂等性细则**：

- `Reconcile` 多次调用相同 spec 不重建底层资源；只有语义字段变化才触发更新
- 重复 patch 相同 `roles[*].replicas` 不得重建底层资源；只调整副本数
- `Cleanup` 对已删除资源返回 nil
- `MapStatus` 是纯函数（不发起 K8s 调用），状态推进只依赖输入参数

**Scale 透传义务**：每个 Handler 必须把 `roles[*].replicas` 透传为后端原生扩缩——

- `(native, deployment)` → patch `Deployment.spec.replicas`
- `(native, statefulset)` → patch `StatefulSet.spec.replicas`
- `(kserve, *)` → patch `InferenceService.spec.predictor.{minReplicas, maxReplicas}`（具体策略见 §8.3）
- 不支持原生扩缩的 backend → 兜底为重建底层资源（应避免，作为最后手段）

**`spec.route` 增量职责**：

- `Reconcile`：根据 `spec.route.enabled` 与各子字段创建 / 删除上面三类派生资源；`Validate` 拒绝 `(kserve, *)` 下的 `enabled=true`、拒绝多 role 但未指定 `targetRole` 的提交、拒绝多端口但未指定 `portName` 的提交
- `MapStatus`：把 HTTPRoute `Accepted` / `ResolvedRefs` condition 翻译为 `status.endpoint`（按 §4 端口选择规则填写外部 URL）与 `status.conditions` 的 `Available` 条件；HTTPRoute `Accepted=False` 视同后端未就绪，应让 `phase=Degraded` 并把失败原因写入 `message`

**Status 写入约束**：Handler 只能通过 `MapStatus` 的返回值影响 `status`；不能在 `Reconcile` 中直接 `status` 写盘。dispatcher 统一合并 `phase` / `message` / `endpoint` / `readyReplicas` / `conditions` / `roles[]` 写入 CR，保证 [§2 写路径契约](#2-与-compute-的写路径契约) 中的 "status 单向权威"。dispatcher 通过 `status` subresource 做 strategic merge patch；`conditions[]` 按 K8s 标准 merge-by-`type` 语义合并，不会全量覆盖。

## 8. 内置 Handler

每个 Handler 章节统一按以下小节组织：**底层资源 / `backend.config` / 通用字段映射 / Status 映射 / Scale / RBAC**。

### 8.1 `(native, deployment)`

底层用 K8s 原生 Deployment + Service，外加一个 minMember=`replicas` 的轻量 Volcano PodGroup（不要求 gang，仅用于 Queue 资源记账，详见 §3.1 中的 "Service 不引入 volcano backend" 说明）。

**底层资源**：

- 必填且仅一个 role（`name=predictor`）；`Validate` 拒绝多 role 提交或其他 role 名
- 每个 MLService 创建一个 K8s `Deployment`、一个 K8s `Service`、一个 Volcano `PodGroup`：
  - `Service` 端口由 `roles[predictor].template.ports[]` 派生（`targetPort=containerPort`）
  - `PodGroup.spec.queue ← spec.scheduling.queue`，`spec.minMember ← roles[predictor].replicas`，扩缩容时 Handler 同步 patch `minMember`（不阻塞调度，仅作为 Queue accounting 锚点）
- 当 `spec.route.enabled=true` 时追加 `HTTPRoute` + 可选的 `SecurityPolicy` / `BackendTrafficPolicy`（与 §6 派生资源说明一致）
- Pod 设置 `schedulerName: volcano`，并通过 annotation `scheduling.k8s.io/group-name=<podgroup-name>` 关联到上述 PodGroup
- Deployment / Service / PodGroup / 派生路由资源设置 `ownerReference` 指向 MLService，保证 MLService 删除后底层资源级联清理；PodGroup 删除后 Volcano Queue `status.allocated` 自然释放该 Service 的用量
- operator 不读写 Volcano Queue CR（Queue 由 Compute 独占维护）

**Pod label**：

- `axisml.io/service-id=<services.id>`
- `axisml.io/role=predictor`

Deployment Pod 没有稳定 index（ReplicaSet 用 hash 后缀，扩缩容/滚动更新都换 Pod 名），按 §6 约定省略 `axisml.io/replica-index`。

**`backend.config`**：本 Handler 不消费；非空时 `Validate` 写 warning，不报错（为 future 字段预留）。

**通用字段映射**：

| MLService 字段 | Deployment / Service / 派生路由资源落点 |
| --- | --- |
| `roles[predictor].template.image` / `imagePullPolicy` / `command` / `args` / `env` / `envFrom` / `workingDir` | Deployment Pod 主容器同名字段 |
| `roles[predictor].template.ports[]` | Deployment Pod 主容器 `ports` + K8s Service `spec.ports`（`targetPort` 取 `containerPort`） |
| `roles[predictor].template.resources.requests` / `limits` | Deployment Pod 主容器同名字段 |
| `roles[predictor].replicas` | `Deployment.spec.replicas` 与 `PodGroup.spec.minMember`（同值；扩缩容时同步 patch） |
| `spec.scheduling.queue` | `PodGroup.spec.queue` 与 Pod label `axisml.io/queue` |
| `spec.scheduling.priorityClass` | Pod `spec.priorityClassName` |
| `spec.scheduling.nodeSelector` / `tolerations` | Pod 同名字段 |
| `spec.modelRef` | Catalog client 解析为模型工件 URI，注入为环境变量 `AXISML_MODEL_URI`（containerPath / volume mount 形态留待后续策略） |
| `spec.route.targetRole` | 选取 HTTPRoute `backendRefs.name` 指向的 K8s Service（单 role 时省略，自动取 `predictor`） |
| `spec.route.portName` | HTTPRoute `backendRefs.port`（解析为 `targetRole` Service 中对应的端口） |
| `spec.route.hostname` / `path` | `HTTPRoute.spec.hostnames` / `rules[].matches[].path.value`（path 默认 `/`） |
| `spec.route.auth` | `SecurityPolicy.spec.{jwt | apiKeyAuth}`，`targetRefs` 指向上面 HTTPRoute |
| `spec.route.rateLimit` / `timeout` | `BackendTrafficPolicy.spec.rateLimit` / `timeout`，`targetRefs` 指向上面 HTTPRoute |
| `spec.runPolicy.progressDeadlineSeconds` | `Deployment.spec.progressDeadlineSeconds` |

**Status 映射**（沿用 [compute.md §6.3.2](../compute.md) 规则，从 Deployment `status` 推导）：

| 条件 | MLService phase |
| --- | --- |
| `desired_replicas == 0` | `Pending`（扩缩至 0，视为待调度 / 停用） |
| `ready_replicas == desired_replicas && desired_replicas > 0` | `Ready` |
| `0 < ready_replicas < desired_replicas` | `Degraded` |
| `ready_replicas == 0 && desired_replicas > 0` | `Failed` |

`endpoint` 按 §4 二分规则填写：

- `spec.route.enabled=false`（默认）→ `<svc>.<namespace>.svc.cluster.local:<port>`，端口按 §4 选择规则（`roles[predictor].template.ports[]` 中 `name=http` 优先，否则 `ports[0]` 并加 warning condition）
- `spec.route.enabled=true` → `https://<hostname><path>`，从 HTTPRoute 派生（hostname 缺省时取 Gateway 监听器的 wildcard）

`readyReplicas` 取 Deployment `status.readyReplicas`；`status.roles[predictor]` 聚合 desired / ready 副本数。

**`spec.route` 的 phase 影响**：`enabled=true` 且 HTTPRoute `Accepted=False`（或 `ResolvedRefs=False`）时——即 Deployment 已就绪但外部入口未生效——映射为 `phase=Degraded`，`message` 写明 HTTPRoute 拒绝原因；同时 `endpoint` 暂时回退为内部 Service DNS，避免暴露未就绪的外部 URL。HTTPRoute 就绪 + Deployment 就绪 → `phase=Ready`，`endpoint` 切换为外部 URL。

**Scale**：patch `Deployment.spec.replicas`；不重建 Pod。

**RBAC**：

- 基础：`deployments.apps` / `services` / `pods` / `events` 的 `create / get / list / watch / update / patch / delete`
- Volcano 调度集成：`podgroups.scheduling.volcano.sh` 的 `create / get / list / watch / update / patch / delete`
- `spec.route` 派生资源：`httproutes.gateway.networking.k8s.io` / `securitypolicies.gateway.envoyproxy.io` / `backendtrafficpolicies.gateway.envoyproxy.io` 的 `create / get / list / watch / update / patch / delete`
- `secrets` 的 `get / list / watch`（仅当 `spec.route.auth.type=apiKey` 引用 Secret 时）

### 8.2 `(native, statefulset)`

为有状态推理（在线 KV cache、模型分片、节点身份固定的副本）预留。底层用 K8s `StatefulSet` + headless Service，副本身份稳定；其余约束沿用 §8.1。

**底层资源**：

- 必填且仅一个 role（`name=predictor`）
- 每个 MLService 创建一个 `StatefulSet`、一个 headless `Service`（`spec.clusterIP=None`）、一个 Volcano `PodGroup`（同 §8.1，minMember=`replicas`，仅作 Queue accounting 锚点）
- Pod 通过 `<pod>.<svc>.<namespace>.svc.cluster.local` 直连，`schedulerName: volcano` + annotation 关联到上述 PodGroup
- StatefulSet Pod 副本身份稳定，Handler 透传 K8s 注入的 `apps.kubernetes.io/pod-index` 为 `axisml.io/replica-index`

**`backend.config` 关键字段**（schema 待细化设计文档）：

```yaml
config:
  podManagementPolicy: OrderedReady | Parallel   # 默认 OrderedReady
  serviceName: string                             # headless Service 名；不填默认 = MLService 名
  volumeClaimTemplates: []                        # 持久卷模板
  updateStrategy:
    type: RollingUpdate | OnDelete
    partition: int                                # RollingUpdate 模式的灰度分界
```

**通用字段映射**：与 §8.1 相同，`roles[predictor].replicas` 落到 `StatefulSet.spec.replicas` 与 `PodGroup.spec.minMember`；`roles[predictor].template.ports[]` 落到 StatefulSet 主容器 `ports` + headless Service `spec.ports`；补充 `volumeClaimTemplates` 与 `serviceName` 字段；其余字段沿用 §8.1 表格。

**`spec.route` 行为**：与 §8.1 一致；HTTPRoute `backendRefs` 指向同一份 headless Service（headless Service 也可作 Gateway API backendRef 目标，由 EndpointSlice 解析具体 Pod）。

**Status 映射**：从 `StatefulSet.status` 推导，规则与 §8.1 同构（用 `readyReplicas` / `replicas` 替换 Deployment 同名字段）；`endpoint` 同样按 §4 二分规则填写（headless Service 的 `<svc>.<ns>.svc.cluster.local:<port>` 解析为 EndpointSlice 中所有就绪 Pod 的 IP），`spec.route` 对 phase 的影响与 §8.1 一致。

**Scale**：patch `StatefulSet.spec.replicas`；副本身份保留，扩容时新 index 追加，缩容时按高 index 优先终止。

**RBAC**：

- 基础：`statefulsets.apps` / `services` / `pods` / `events` 的 `create / get / list / watch / update / patch / delete`
- Volcano 调度集成：`podgroups.scheduling.volcano.sh` 的 `create / get / list / watch / update / patch / delete`
- `spec.route` 派生资源：与 §8.1 同
- `secrets` 的 `get / list / watch`（仅当 `spec.route.auth.type=apiKey` 引用 Secret 时）

### 8.3 `(kserve, inference)`

将 MLService 翻译为 KServe [`InferenceService`](https://kserve.github.io/website/) CR（`serving.kserve.io/v1beta1`）。这是 KServe 通用 ML 服务路径——predictor 内的具体 runtime（NVIDIA Triton / [vLLM](https://docs.vllm.ai/) / TF Serving / TorchServe / sklearn / huggingface 等）由 `backend.config.runtime` 选择，转化为 KServe `(Cluster)ServingRuntime` 引用或 `predictor.model.modelFormat` 声明。

**前置依赖**：集群已安装 KServe；其 RBAC 与 CRD 由 KServe chart 单独管理，本 Handler 仅需要 `inferenceservices.serving.kserve.io` 的 `create / get / list / watch / update / patch / delete`，外加各 runtime 对应 `(Cluster)ServingRuntime` 的 `get / list / watch`。

**Role 集合约定**：当前仅开放 `predictor`（replicas≥0）；扩展角色 `transformer` / `explainer` 的接入节奏见 §11。

**`backend.config` 关键字段**（schema 待细化设计文档）：

```yaml
config:
  runtime: triton | vllm | tfserving | torchserve | sklearn | huggingface | <自定义 ServingRuntime 名>
                                  # 必填: 选择 predictor 内的运行时框架
  predictor:
    minReplicas: int              # 默认 = roles[predictor].replicas
    maxReplicas: int              # 自动扩缩上限；不填则等于 minReplicas
    scaleToZero: bool             # 是否允许 scale-to-zero
    protocolVersion: v1 | v2      # KServe 推理协议
  storageUri: string              # 模型工件位置；可由 Catalog 通过 modelRef 自动解析
  containerOverrides: {}          # 容器级别 override（command / args / env）

  # ── runtime 专属子段（仅在 runtime=对应值 时生效）──
  triton:
    modelControlMode: none | poll | explicit
    modelRepository: string                  # 显式覆盖 storageUri 为 model repo 根路径
  vllm:
    model: string                            # 模型名（默认 = modelRef.name）
    dtype: auto | float16 | bfloat16 | float32
    tensorParallelSize: int                  # TP 并行度，等于单副本 GPU 数
    pipelineParallelSize: int                # PP 并行度
    maxModelLen: int                         # 上下文长度上限
    maxNumBatchedTokens: int                 # 调度批 token 上限
    enablePrefixCaching: bool
    quantization: awq | gptq | fp8 | none
    extraArgs: []                            # 透传给 vllm serve 的额外参数
  huggingface:
    task: string                             # text-generation / text-classification / ...
    modelId: string                          # HF Hub 模型 ID
  torchserve:
    modelStore: string
  # 其他 runtime（tfserving / sklearn）的子段按需扩展
```

**通用字段映射**：

- `roles[predictor].template.image` → predictor 容器（不填时由 `config.runtime` 选定的 ServingRuntime 提供默认镜像）
- `roles[predictor].template.{command, args, env, envFrom, workingDir}` → predictor 容器同名字段
- `roles[predictor].template.ports[]` → predictor 容器 `ports`；KServe runtime 据此暴露 inference endpoint
- `roles[predictor].template.resources` → predictor `resources`
- `roles[predictor].replicas` → 写入 `predictor.minReplicas`；若未设置 `config.predictor.maxReplicas`，则同时写入 `maxReplicas`
- `spec.modelRef` → 通过 Catalog 解析为 `predictor.storageUri`（runtime=triton 时也可解析为 `triton.modelRepository`；runtime=vllm 时优先解析为 `vllm.model`，缺失时回退到 `storageUri`）
- `spec.scheduling.queue` → 仅落到 Pod label `axisml.io/queue`，不强制注入 `schedulerName: volcano`（KServe Pod 由 KServe 自身派生，本 Handler 不直接管理；与 §8.1 / §8.2 native Handler 走 Volcano 调度 + 轻量 PodGroup 的方式不同）。**已知缺口**：KServe 路径下 MLService 用量当前不计入 Volcano Queue `status.allocated`，因此也不进入 Compute `queues.used`；KServe + Volcano 调度集成（决定 `schedulerName` / PodGroup 注入策略，让 KServe Pod 也参与 Queue accounting）由独立设计文档落地（见 §11）
- `spec.scheduling.priorityClass` / `nodeSelector` / `tolerations` → predictor 同名字段
- `spec.runPolicy.progressDeadlineSeconds` → KServe 暂无对等字段，Handler 在 Validate 中写 warning
- `spec.route` → **不支持**；KServe `InferenceService` 自带对外 Route，Handler 在 `Validate` 中拒绝 `spec.route.enabled=true`，写 `status.message="spec.route not supported on (kserve, *) backend; KServe manages its own route"`

**runtime 专属约束**（由 `Validate` 强制）：

- `runtime=vllm`：`roles[predictor].template.resources.requests["nvidia.com/gpu"]` 必须等于 `config.vllm.tensorParallelSize × pipelineParallelSize`
- `runtime=huggingface`：`config.huggingface.task` 必填
- 其他 runtime 的强制项由 ServingRuntime 自身校验，本 Handler 透传

**扩展 transformer / explainer 角色**：映射至 `roles[transformer]` / `roles[explainer]`，字段映射镜像 predictor；开放节奏见 §11。

**Status 映射**：从 `InferenceService.status.conditions` 推导——

| InferenceService condition | MLService phase |
| --- | --- |
| `PredictorReady=False` 且 `desired==0` | `Pending` |
| `Ready=True` | `Ready` |
| `PredictorReady=False` 且 `0 < ready < desired` | `Degraded` |
| `Ready=False` 且 `ready==0 && desired>0` | `Failed` |

`endpoint` 取 `InferenceService.status.url`（KServe 自带对外 Route，本 Handler 不接受 `spec.route.enabled=true`，因此 `endpoint` 单字段定义直接对齐 KServe 自带的外部 URL）。

**Scale**：patch `InferenceService.spec.predictor.{minReplicas, maxReplicas}`；具体取舍（min 跟随 / max 联动）由独立设计文档落地。

**Quota 与 autoscaling 的相互作用**：KServe scale-to-zero / 自动扩缩可能让实际副本数动态变化；Compute quota 按 `maxReplicas × requests` 上限计费（与 native 的 "replicas × requests" 线性记账一致），保证账面与运行时不打架；`runtime=vllm` 单副本 GPU 数还受 `tensorParallelSize × pipelineParallelSize` 影响。具体细节由独立设计文档落地（§11）。

### 8.4 `(kserve, llminference)`

> **本节为占位设计**：KServe LLM API 的 GVK / CRD 字段路径仍在演进，落地以引入版本为准。本节当前只锁两件事——role 命名约定（`prefill / decode / router`）与 PD 分离骨架（`backend.config` 形状）；schema 详细字段、`Validate` 强制项、Status condition 名等待 KServe LLM API GA 后在 §11 单独成文。读者切勿把本节字段当作可直接实现的契约。

将 MLService 翻译为 KServe LLM 原生 CR `LLMInferenceService`（占位命名；KServe 社区围绕 LLM 原生服务的 GVK 仍在演进，候选名包括 `LLMInferenceService` / `InferencePool` / `LLMRoute` 等，**实际 GVK 以引入 KServe 版本时为准**）。该 engine 承载 LLM 在线服务相对 `InferenceService` 的额外能力——核心是 **PD 分离（disaggregated serving）**：prefill 与 decode 拆成独立角色独立扩缩，搭配 router 角色做请求分发与 KV cache 协调。

**前置依赖**：集群已安装 KServe LLM API（含 `LLMInferenceService` CRD 与对应 controller / runtime）。本 Handler 需要 `llminferenceservices.serving.kserve.io`（占位）的 `create / get / list / watch / update / patch / delete`，外加 KV cache 协议相关 ConfigMap / Secret 的读取权限（细节随 KServe LLM API 落地补全）。

**Role 集合约定**（PD 分离骨架；具体 role 名以 KServe LLM API 落地为准）：

- `prefill`：长上下文处理（compute-bound）；replicas≥1；GPU 配置通常偏算力
- `decode`：token 生成（memory-bound）；replicas≥1；GPU 配置通常偏显存与互联带宽
- `router`：请求入口与 KV cache 协调；replicas≥1；可承载 `spec.route.targetRole`（PD 拓扑里"对外端点"应指向 router 而非 prefill / decode 单体）

`Validate` 强制：role 名属于上述集合；至少存在 `prefill` 与 `decode`；`router` 在 `spec.route.enabled=true` 或多 LLMInferenceService 共享路由层时强制必填（具体节奏见 §11）。

**`backend.config` 关键字段**（schema 占位；待 KServe LLM API 落地后细化）：

```yaml
config:
  runtime: vllm | <其他 LLM 原生 runtime>     # 必填: 当前主流为 vllm disaggregated
  storageUri: string                          # 模型工件位置；可由 modelRef 自动解析
  llm:
    model: string                              # 模型名（默认 = modelRef.name）
    maxModelLen: int
    quantization: awq | gptq | fp8 | none
  disaggregation:
    kvTransport: nixl | mooncake | <其他>      # KV cache 传输协议
    prefillToDecodeRatio: float                # prefill : decode 副本比建议值（autoscaler 参考）
  parallelism:
    prefill:
      tensorParallelSize: int
      pipelineParallelSize: int
    decode:
      tensorParallelSize: int
      pipelineParallelSize: int
```

**通用字段映射**（占位，`LLMInferenceService` 字段路径以实际 GVK 为准）：

- `roles[prefill / decode / router].template.{image, command, args, env, envFrom, workingDir, ports, resources}` → 对应 role 在 `LLMInferenceService.spec` 下的同构字段
- `roles[*].replicas` → 各 role 的 `minReplicas` / 副本数
- `spec.modelRef` → `config.llm.model` 或 `config.storageUri`
- `spec.scheduling.*` → 各 role Pod 的同名字段
- `spec.runPolicy.progressDeadlineSeconds` → KServe 暂无对等字段，Handler 在 Validate 中写 warning
- `spec.route` → **不支持**（同 §8.3）；KServe LLM API 自带 router / Route 机制

**单副本 GPU 数约束**：`roles[prefill / decode].template.resources.requests["nvidia.com/gpu"]` 必须等于该 role 的 `tensorParallelSize × pipelineParallelSize`，由 `Validate` 强制。

**Status 映射**：参照 KServe LLM API 的 condition 集合落地，原则上沿用 §8.3 四态映射；具体 condition 名以 KServe LLM API 实现为准（§11 写明落地节奏）。`endpoint` 取 KServe LLM API 暴露的 router 入口（与 §8.3 取 `status.url` 同思路；具体字段路径以引入版本为准）。

**Scale**：分别 patch 各 role 在 `LLMInferenceService` 中的 `minReplicas` / `maxReplicas`；多 role 独立扩缩需要 §11 中的 `:scale` API 路径携带 role 名。

**Quota 与 autoscaling 的相互作用**：与 §8.3 一致，按 `Σ(role.maxReplicas × role.requests)` 计费；`prefillToDecodeRatio` 仅作为 autoscaler 参考，不参与配额校验。

### 8.5 `(custom, *)`

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

由 custom Handler 通过 unstructured client 创建并跟踪。完整 schema 与 unstructured 操作约定由独立设计文档落地（见 §11）。

**`spec.route` 在 custom Handler 下的语义**：由 `config.routeBackend`（在独立设计文档中定义）显式描述外部入口对接的目标 Service；未在 `config` 中 wire `spec.route` 时，Handler 应在 `Validate` 中拒绝 `spec.route.enabled=true`。

## 9. RBAC 聚合

operator binary 启动时遍历 registry，把每个启用 Handler 的 `RequiredRBAC()` 合并去重，生成本 binary 实际需要的 ClusterRole rules。Helm chart 通过 values 控制启用集合，渲染最小化 RBAC 而非全集——例如仅启用 `(native, *)` 时，集群无需安装 KServe；启用 `(kserve, *)` 才注入 KServe 与 ServingRuntime 的 RBAC。

## 10. 不变量与约束

- `spec.backend.{name, engine}` 创建后不可变；dispatcher 拒绝并写 `status.message`，admission webhook 后续接管
- `(backend, engine)` 元组未在 registry 注册 → MLService 直接进入 `Failed`，message 写明缺失原因
- 任一 Handler 的 `Reconcile` / `MapStatus` 必须保持 §2 列出的写路径契约——这是把 "插件" 安全嵌入 Compute Outbox 模型的根基
- Handler 不直接修改 Volcano Queue CR；Queue CR 由 Compute 独占维护
- Handler 不向 Compute PG 写入任何数据；状态全部经由 MLService `status` + Informer 回流
- **Handler 不引入 finalizer**；级联清理依赖 ownerReference + `Cleanup()`
- **`status.phase` 取值集合冻结为四态**（`Pending | Ready | Degraded | Failed`）；新增 phase 必须经 CRD schema 与 Compute 双侧同步演进
- **`spec.roles[*].replicas` 是允许变更的字段**（`:scale` 路径专用）；其余 spec 字段创建后不可变，dispatcher 检测到变更需写 `status.message` 拒绝
- 所有 Handler 必须打 `axisml.io/service-id` + `axisml.io/role` 两件套 label；副本身份天然稳定的场景（`(native, statefulset)`）建议叠加 `axisml.io/replica-index`，`(native, deployment)` / KServe autoscaling pod 集合等无稳定身份场景一律省略
- **`spec.route` 创建后不可变（v1）**；mutable 演进作为后续设计文档预留（见 §11）
- **`(kserve, *)` Handler 不接受 `spec.route.enabled=true`**（KServe 自带 Route，避免双管）；`(native, *)` 接受；`(custom, *)` 由 `config.routeBackend` 自描述，未 wire 时拒绝
- **`spec.route` 派生的 `HTTPRoute` / `SecurityPolicy` / `BackendTrafficPolicy` 通过 `ownerReference` 级联清理**；Handler 不引入 finalizer
- **`status.endpoint` 是单一对外服务地址字段**（Compute 透传到 `services.endpoint`）：`spec.route.enabled=false` 时为 K8s Service DNS（ClusterIP / headless 共用 `<svc>.<ns>.svc.cluster.local:<port>` 格式），`enabled=true` 时为外部 URL（`https://<hostname><path>`）；不再单独建 `status.externalUrl` 字段，避免 compute.md services 表 schema churn

## 11. 后续设计文档（不在本文档范围）

- `(native, statefulset)` Handler 的 `volumeClaimTemplates` / 灰度更新 / pod-index 寻址细节
- 多 role 接入的具体 Handler 落地：
  - `(kserve, inference)` 的 `transformer` / `explainer` 字段映射与状态映射
  - `(kserve, llminference)`（对应 `LLMInferenceService` 占位 GVK，最终以 KServe LLM API 落地版本为准）：vLLM disaggregated / llm-d / NVIDIA Dynamo 等场景下，`prefill` / `decode` / `router` 三类 role 的命名约定、KV cache 传输契约（NIXL / Mooncake / …）、`disaggregation.prefillToDecodeRatio` autoscaler 接入、`Validate` 中的多 role 必填规则、router role 与 `spec.route` 的协作方式
- KServe Pod 与 Volcano 调度的可选集成方案（如需让 KServe Pod 接受 Volcano 队列调度，决定 `schedulerName` / PodGroup 注入策略）
- KServe scale-to-zero 与 Compute quota 的精细交互模型（含 `maxReplicas × requests` 上限计费策略）
- `(custom, *)` Handler 的 `config` 完整 schema 与 unstructured 操作约定（含 `config.routeBackend` 与 `spec.route` 的对接细则）
- 多 role 独立扩缩容的 `:scale` API 扩展（路径中携带 role 名）
- `spec.route` 可变化路径（轮换 API key / 调整限流不需要重建 Service；Handler 侧需要识别哪些子字段可热更新、哪些必须重建派生资源）
- `spec.route` 与 KServe 自带 Route 的统一化（让 `(kserve, *)` 也支持 `spec.route` 而非依赖 KServe 内置 Route）
- Admission webhook：`spec.backend.{name, engine}` 不可变约束、`backend.config` 按 Handler 自带 schema 的统一校验
- Handler chart 的 values 控制（启用 / 禁用 backend，对应 RBAC 与 watch 的开关）
- CRD 严格 schema（启用 OpenAPI 校验，替换当前的 `x-kubernetes-preserve-unknown-fields`）
