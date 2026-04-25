# MLService Operator 详细设计

## 1. 与 Compute 的写路径契约

Compute 采用 Operation Outbox + reconciler 异步下发 CR（详见 [compute.md §5.2](../compute.md)）。Operator 必须满足以下契约：

- **`Create` 幂等**：相同 `metadata.name` 的二次 `Create()` 返回 409 `AlreadyExists`，且不引发副作用。Compute 收到 409 后会 `Get()` 现有 CR 并校验 label `axisml.io/service-id=<uuid>`；只有 label 一致才视为成功
- **不主动反向写 PG**：operator 不感知 Compute 的 `services` 表；状态推进由 Compute 侧 Informer 按 CR `status` 回流
- **status 单向权威**：operator 只写 MLService `status`；Compute 只写 MLService `metadata` / `spec`
- **扩缩容幂等**：重复 patch 相同 `spec.replicas` 不得重建底层资源；只调整副本数
- **`spec.backend.{name,engine}` 创建后不可变**：reconciler 检测到变更直接拒绝（写 `status.message`），admission webhook 后续补；Compute 也保证不会修改这两个字段

## 2. 总体架构：Dispatcher + Handler

mlservice-operator 与 mljob-operator 同构，由两层组成：

- **Dispatcher Reconciler**：watch 所有 MLService CR，按 `spec.backend.{name, engine}` 元组路由到对应 Handler。本身不直接生成底层资源
- **Handler 注册表**：每个 `(backend, engine)` 元组对应一个 Handler 实例；编译期通过 `init()` 注册到全局 registry。Handler 负责把通用字段 + `backend.config` 翻译成具体的底层资源（Deployment+Service / KServe InferenceService 等），并把后端原生状态映射回 MLService 统一 phase

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

**Watch 拓扑**：Dispatcher 始终 watch MLService 主队列；每个 Handler 启动时声明自己关心的底层资源类型（Deployment、Service、PodGroup、InferenceService …），由 dispatcher 统一建立 watch（controller-runtime `Watches()`），事件通过 `ownerReference` 反查回 MLService 后再交给对应 Handler 的 `MapStatus`。

**注册方式**：编译期注册（每个 handler 包 `init()` → 全局 registry）。

**未注册组合的兜底**：dispatcher 收到 `(backend, engine)` 无 handler 的 MLService → 写 `status.phase=Failed`、`status.message="no handler for backend=X engine=Y"`，不创建任何底层资源。

## 3. CRD 契约

MLService 为 namespaced CR，创建在租户 namespace 下。Compute 负责设置：

- `metadata.name`：来自 `services.name`
- `metadata.namespace`：来自 `tenants.namespace`
- `metadata.labels["axisml.io/service-id"]`：`services.id`
- `metadata.labels["axisml.io/tenant"]`：租户名
- `metadata.labels["axisml.io/queue"]`：Compute Queue 名

最小 `spec`：

```yaml
apiVersion: compute.axisml.io/v1alpha1
kind: MLService
spec:
  backend:
    name: native             # 必填；枚举：native | kserve | custom
                             # （kubeflow-trainer 仅用于 MLJob，不在 MLService 枚举内）
    engine: default          # 必填；语义随 backend 而定，见 §6
    config: {}               # 可选；该 (backend, engine) 元组特有的配置，schemaless
  image: string
  modelRef:
    name: string
    version: string
  replicas: int              # >= 0
  queueName: string          # Volcano Queue CR name
  resources:
    requests: {}             # Kubernetes ResourceList，单副本资源
    limits: {}               # Kubernetes ResourceList，单副本资源
  placement:
    nodeSelector: {}
    tolerations: []          # Kubernetes Toleration 数组
  ports:
    - name: http
      containerPort: 8080
```

**字段归属**：

- `backend.{name,engine,config}` 由 dispatcher 用于路由，由具体 Handler 解释
- 通用字段（`image` / `modelRef` / `replicas` / `queueName` / `resources` / `placement` / `ports`）由所有 Handler 共同遵守语义；Handler 负责把它们注入到底层资源的对应位置
- `backend.config` 仅用于通用字段表达不出来的引擎特有配置（如 KServe `predictor.protocolVersion`、autoscaling 阈值）

**默认值注入**：Compute 在创建 MLService CR 时，若用户未指定 `spec.backend`，显式补 `{name: native, engine: default}`。dispatcher 不接受 `backend.name` 或 `backend.engine` 为空。

## 4. Status 契约

最小 `status`：

```yaml
status:
  observedGeneration: int64
  phase: Pending | Ready | Degraded | Failed
  message: string
  readyReplicas: int
  endpoint: string
```

Compute 映射规则：

| MLService status.phase | services.status | 终态 |
| --- | --- | --- |
| `Pending` | `Pending` | 否 |
| `Ready` | `Ready` | 否 |
| `Degraded` | `Degraded` | 否 |
| `Failed` | `Failed` | 否，可恢复 |

CR 删除事件由 Compute Informer 映射为 `Deleting → Deleted`（详见 [compute.md §6.3.2](../compute.md)）。

**Phase 由 Handler 产出**：每个 Handler 在 `MapStatus` 中负责把后端原生状态映射到上述四态；映射规则写入对应 Handler 的章节（§6）作为契约。

## 5. Handler 接口契约

所有 Handler 必须实现以下行为（语言无关，描述责任）：

| 方法 | 责任 |
| --- | --- |
| `Key()` | 返回 `(backend, engine)` 元组，与 `spec.backend.{name,engine}` 对齐；用作注册表的主键 |
| `Validate(spec)` | 校验通用字段 + `backend.config`，纯函数，未来可被 admission webhook 复用 |
| `Reconcile(ctx, mlService)` | 创建 / 更新底层资源；保证幂等；通用字段由 Handler 自己注入到对应位置；处理 `spec.replicas` 的扩缩容 |
| `MapStatus(underlying)` | 把后端原生状态映射回 §4 的四态 phase + readyReplicas + endpoint + message |
| `Cleanup(ctx, mlService)` | 删除底层资源。一般依赖 ownerReference 自动级联，Handler 仅负责需要主动清理的副本 |
| `WatchTargets()` | 声明本 Handler 需要 watch 的底层资源 GVK 列表 |
| `RequiredRBAC()` | 声明本 Handler 需要的 ClusterRole 规则 |

**幂等性要求**：

- `Reconcile` 多次调用相同 spec 不重建底层资源；只有语义字段变化才触发更新
- `Cleanup` 对已删除资源返回 nil
- `MapStatus` 是纯函数（不发起 K8s 调用）

**扩缩容语义统一**：Compute 通过 `:scale` API 同步 patch MLService `spec.replicas`；dispatcher 把变更投递给当前 Handler，Handler 必须把 replicas 透传到后端 CR——

- native → patch `Deployment.spec.replicas`
- kserve → patch `InferenceService.spec.predictor.{minReplicas,maxReplicas}`（具体策略见 §6.2）
- 不支持原生扩缩的 backend → 兜底为重建底层资源（应避免，作为最后手段）

## 6. 内置 Handler

### 6.1 `(native, default)` —— MVP

MVP 唯一落地的 Handler，保留现有 Deployment + Service + PodGroup 行为：

**底层资源**：

- 每个 MLService 创建一个 Deployment、一个 K8s Service、按需创建 Volcano `PodGroup`
- Deployment Pod 设置 `schedulerName: volcano`，`PodGroup.spec.queue` 写 MLService `spec.queueName`
- Deployment / Service / PodGroup 设置 ownerReference 指向 MLService，保证 MLService 删除后底层资源级联清理
- operator 不读写 Volcano Queue CR；Queue CR 由 Compute 独占维护

**`backend.config`**：本 Handler 不消费 `backend.config`（schema 校验时若非空写 warning，不报错；为 future 字段预留）。

**Status 映射**（沿用现行规则，从 Deployment `status` 推导）：

| 条件 | MLService phase |
| --- | --- |
| `desired_replicas == 0` | `Pending`（扩缩至 0，视为待调度 / 停用） |
| `ready_replicas == desired_replicas && desired_replicas > 0` | `Ready` |
| `0 < ready_replicas < desired_replicas` | `Degraded` |
| `ready_replicas == 0 && desired_replicas > 0` | `Failed` |

`endpoint` 取 K8s Service ClusterIP DNS（`<svc>.<namespace>.svc.cluster.local:<port>`）。

**RBAC**：`deployments.apps`、`services`、`podgroups.scheduling.volcano.sh`、`pods`、`events` 的 `create/get/list/watch/update/patch/delete`。

### 6.2 `(kserve, triton)` —— 占位

将 MLService 翻译为 KServe `InferenceService` CR，predictor 使用 Triton。

**前置依赖**：集群已安装 KServe；其 RBAC 与 CRD 由 KServe chart 单独管理，本 Handler 仅需要 `inferenceservices.serving.kserve.io` 的 `create/get/list/watch/update/patch/delete`。

**`backend.config` 关键字段**（schema 待细化设计文档）：

```yaml
config:
  predictor:
    minReplicas: int          # 默认 = spec.replicas
    maxReplicas: int          # 自动扩缩上限；不填则等于 minReplicas
    scaleToZero: bool         # 是否允许 scale-to-zero
    protocolVersion: v1 | v2  # KServe 推理协议
  storageUri: string          # 模型工件位置；可由 Catalog 通过 modelRef 自动解析
  containerOverrides: {}      # 容器级别 override（command / args / env）
```

通用字段映射：

- `image` → 写入 predictor 容器（Triton 默认镜像可由 KServe runtime 模板提供，`spec.image` 用作 override）
- `modelRef` → 通过 Catalog 解析为 `predictor.storageUri`
- `replicas` → 写入 `predictor.minReplicas`；若未设置 `config.predictor.maxReplicas`，则同时写入 maxReplicas
- `resources` → 写入 predictor `resources`
- `placement` → 写入 predictor `nodeSelector` / `tolerations`
- `queueName` → KServe pod 通过 podSpec 注入 `schedulerName: volcano` + 关联 PodGroup（具体集成方式以 KServe + Volcano 集成文档为准）

**Status 映射**：从 `InferenceService.status.conditions` 推导——

| InferenceService condition | MLService phase |
| --- | --- |
| `PredictorReady=False` 且 `desired==0` | `Pending` |
| `Ready=True` | `Ready` |
| `PredictorReady=False` 且 `0 < ready < desired` | `Degraded` |
| `Ready=False` 且 `ready==0 && desired>0` | `Failed` |

`endpoint` 取 `status.url`。

**Quota 与 autoscaling 的相互作用**：KServe scale-to-zero / 自动扩缩可能让实际副本数动态变化；Compute quota 按 `maxReplicas × requests` 上限计费（与 native 的"replicas × requests"线性记账一致），保证账面与运行时不打架。具体细节由独立设计文档落地。

> 完整字段映射、scale-to-zero 与 quota 的精细交互、Knative dependency 取舍等，由独立设计文档落地（见 §9）。

### 6.3 `(kserve, tfserving / torchserve / sklearn / huggingface)` —— 占位

同 §6.2，将 MLService 翻译为 KServe InferenceService 的对应 predictor 类型；`config` 携带 framework 特有字段（如 huggingface 的 `task` / `modelId`、torchserve 的 model store 路径）。状态映射沿用 §6.2 表格。

### 6.4 `(custom, *)` —— 占位

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
      "spec.image": "$.image"
      "spec.replicas": "$.replicas"
      # ...
    statusMappings:
      "$.status.phase": { Pending: Pending, Active: Ready, Degraded: Degraded, Error: Failed }
    endpointPath: "$.status.url"
```

由 custom Handler 通过 unstructured client 创建并跟踪。**MVP 不实现**；当首个第三方后端无法纳入 `(kserve, *)` 时再设计完整 schema。

## 7. RBAC 聚合

operator binary 启动时遍历 registry，把每个 Handler 的 `RequiredRBAC()` 合并去重，生成本 binary 实际需要的 ClusterRole rules。Helm chart 渲染时根据启用的 Handler 集合（通过 values 控制）生成最小化 RBAC 而非全集。

## 8. 不变量与约束

- `spec.backend.{name, engine}` 创建后不可变（dispatcher 拒绝并写 `status.message`；admission webhook 后续接管）
- `(backend, engine)` 元组未在 registry 注册 → MLService 直接进入 `Failed`，message 写明缺失原因
- 任一 Handler 的 `Reconcile` / `MapStatus` 必须保持 §1 列出的写路径契约
- Handler 不直接修改 Volcano Queue CR；Queue CR 由 Compute 独占维护
- Handler 不向 Compute PG 写入任何数据；状态全部经由 MLService `status` + Informer 回流

## 9. 后续设计文档（不在本文档范围）

- `(kserve, triton / tfserving / torchserve / sklearn / huggingface)` 各自的字段映射与状态映射细节
- `(custom, *)` Handler 的 `config` 完整 schema 与 unstructured 操作约定
- KServe scale-to-zero 与 Volcano quota 的精细交互模型
- Admission webhook：`spec.backend.{name,engine}` 不可变约束、`backend.config` 按 Handler 自带 schema 的统一校验
- Handler chart 的 values 控制（启用 / 禁用 backend，对应 RBAC 与 watch 的开关）
