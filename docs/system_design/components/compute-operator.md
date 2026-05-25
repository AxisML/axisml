# AxisML Compute Operator 概要设计

## 1. 定位与边界

承载 `MLJob` / `MLService` 两个 namespaced CR 的 Kubernetes operator；以 dispatcher + handler 模式把 [compute-service](compute-service.md) 下发的期望状态翻译为底层 K8s 与第三方资源，并把执行状态回流到 CR `status`。

| 做 | 不做 |
| --- | --- |
| MLJob / MLService CR reconcile，dispatcher 按 `spec.backend.{name, engine}` 路由 | Tenant / Namespace / ElasticQuota 落地 (→ [tenant-operator.md](tenant-operator.md)) |
| 派生 Job / Pod / PodGroup / Deployment / StatefulSet / HTTPRoute / KServe `InferenceService` 等 | 业务持久化、用量计费、Outbox 推进 (→ [compute-service.md](compute-service.md)) |
| `spec.route` 派生 Gateway API + Envoy Gateway 扩展资源 | 模型工件存储 (→ [artifact-hub.md](artifact-hub.md)) |
| Cancel 推进信号（`Suspended` condition）单向回流 | 用户认证 / 鉴权 (→ [auth.md](../auth.md)) |
| Pod 注入 `schedulerName=koord-scheduler` + Quota label | 写 compute PG / 跨集群联邦 |

## 2. 架构

### 2.1 上下文

```
   ┌──────────────┐   Create / Patch CR (spec)   ┌──────────────────────┐
   │   compute    │ ────────────────────────────▶│   K8s API: MLJob /   │
   └──────────────┘                              │   MLService          │
          ▲                                       └─────────┬────────────┘
          │  status (watch)                                 │ watch
          │                                                 ▼
   ┌──────────────┐                            ┌────────────────────────┐
   │   compute    │◀─── CR.status 回流 ────────│   compute-operator     │
   └──────────────┘                            └─────────┬──────────────┘
                                                         │ 派生
              ┌──────────────────────────────────────────┼──────────────────────────────────────────┐
              ▼                                          ▼                                          ▼
   K8s 原生 (Job/Pod/Deploy/StatefulSet/Svc)  scheduler-plugins PodGroup       第三方 (Kubeflow Training / KServe)
                                                         │
                                                         ▼
                                            Gateway API + Envoy Gateway 扩展
                                            (HTTPRoute / SecurityPolicy / BackendTrafficPolicy)
```

### 2.2 内部结构

```
┌──────────────── compute-operator (单 Pod, leader-elected) ────────────────┐
│  ctrl.Manager   Lease: axisml-compute-operator.axisml.io                  │
│                                                                            │
│  ┌──────────────────────────┐    ┌──────────────────────────┐             │
│  │ MLJob Dispatcher         │    │ MLService Dispatcher     │             │
│  │ ── handler registry ──   │    │ ── handler registry ──   │             │
│  │  (native,job)            │    │  (native,deployment)     │             │
│  │  (native,podgroup)       │    │  (native,statefulset)    │             │
│  │  (kubeflow-trainer,*)    │    │  (kserve,inference)      │             │
│  │  (custom,*)              │    │  (kserve,llminference)   │             │
│  │                          │    │  (custom,*)              │             │
│  └────────────┬─────────────┘    └────────────┬─────────────┘             │
│               │ watch + status patch          │ watch + status patch       │
│               ▼                                ▼                            │
│  MLJob CR + ownerReference 派生资源   MLService CR + ownerReference 派生资源│
└────────────────────────────────────────────────────────────────────────────┘
```

`--enable-mljob` / `--enable-mlservice` 单独启停对应 dispatcher，未启用时其 ClusterRole 分段也不渲染（最小权限）。

## 3. 核心模型

| 实体 | 含义 | 范围 | 状态机集合 | `backend` 路由元组 |
| --- | --- | --- | --- | --- |
| MLJob | 一次性批训练 / 离线任务 | Namespaced (`mlj`) | `Pending / Running / Succeeded / Failed` | `(native,job) \| (native,podgroup) \| (kubeflow-trainer,{pytorchjob,tfjob,mpijob,…}) \| (custom,*)` |
| MLService | 在线推理服务 | Namespaced (`mls`) | `Pending / Ready / Degraded / Failed` | `(native,deployment) \| (native,statefulset) \| (kserve,{inference,llminference}) \| (custom,*)` |

两个 CR 均使用 `axisml.io/v1alpha1`，`status` subresource 必启用。字段级 schema 见 [§6](#6-接口契约) 引用的 CRD yaml；spec 顶层结构与字段归属约定见 [§4.1.1](#411-mljob-spec-高层结构) / [§4.2.1](#421-mlservice-spec-高层结构)。

`spec.scheduling.quota` 是 compute 透传的 ElasticQuota CR 名字符串，对 operator 不透明——ElasticQuota 资源本身由 [tenant-operator](tenant-operator.md) 独占维护。

## 4. 核心功能

### 4.1 MLJob Controller

#### 4.1.1 MLJob spec 高层结构

```yaml
spec:
  backend:      { name, engine, config }     # 路由元组 + 后端专属 schemaless 配置
  scheduling:   { quota, priorityClass, nodeSelector, tolerations }
  roles:        [{ name, replicas, restartPolicy, template }]   # 多角色拓扑
  runPolicy:    { suspend, activeDeadlineSeconds, ttlSecondsAfterFinished, backoffLimit }
  outputs:      [{ name, kind, volumeName, sourcePath }]        # 可选；声明产物位置，供 Platform register-model 桥接
```

字段完整 schema 与不可变性约定以 CRD yaml + 后续 admission webhook 为准；`spec.backend.{name,engine}` 创建后不可变，`spec.runPolicy.suspend` 是唯一允许由 API（`/cancel`）翻转的字段。`spec.outputs[]` 创建后不可变。

**`spec.outputs[]` 语义**：纯元数据，operator 运行时不消费——仅在 `Validate(spec)` 阶段做静态约束（见下表）。Job 跑完后产物 bytes 留在用户挂载的 PVC 上；Platform 通过 [§4.5.3](platform.md#453-register-from-job计算任务--模型) 的 register-model 桥接流读取这份声明，反查 PVC + sourcePath，调 artifacts initiate 并向用户返回上传凭证 + provenance（字节由客户端工具异步推送，不由 operator 或 Platform 搬运）。

| 字段 | 约束（`Validate`） |
| --- | --- |
| `name` | 同 Job 内唯一；DNS-1123 |
| `kind` | 当前仅支持 `model`；其他值 `400` |
| `volumeName` | 必须命中 `roles[*].template.volumes[].name` 中某个 PVC 类型卷（`persistentVolumeClaim` 或 `ephemeral` 派生持久卷），拒绝 `emptyDir`——否则 Pod 终止后产物消失 |
| `sourcePath` | 相对路径，挂在 `volumeName` 之内；不做穿越校验 |

#### 4.1.2 状态机与事件路径

```
   ADD ──▶ Pending ──(any pod Running)──▶ Running ──┬──▶ Succeeded
                                                     │
                                                     └──▶ Failed
   spec.runPolicy.suspend=true:
       Pending/Running ──(Handler 完成 suspend)──▶ status.conditions[Suspended,True,CancelRequested]
                                                  (phase 维持 Pending；终态优先)
```

| 事件 | Dispatcher 行为 | Handler 行为 |
| --- | --- | --- |
| MLJob ADD | 路由到 Handler；`Validate(spec)` 失败 → `status.phase=Failed` | `Reconcile` 创建底层资源 + 设置 `ownerReference` |
| MLJob UPDATE（spec） | 校验 `backend.{name,engine}` 不变；其余 spec 路由 | `Reconcile` 幂等更新 |
| `runPolicy.suspend=true` | 终态优先；非终态时合并 Handler 返回的 suspend 结果，写 `Suspended=True,reason=CancelRequested` | 执行原生 suspend 或 `Cleanup()`，返回 `suspendCompleted=true` |
| MLJob DELETE | 不阻断（无 finalizer） | 依赖 `ownerReference` 级联清理 |
| 底层资源事件 | 通过 `ownerReference` 反查路由 | `MapStatus` 纯函数计算新 phase |

#### 4.1.3 `(native, job)` Handler

| 维度 | 取值 |
| --- | --- |
| 底层资源 | K8s `Job`（单 role `worker`），不创建 PodGroup |
| 必填字段 | `roles[worker]`，`template.image`，`scheduling.quota` |
| 关键字段映射 | `replicas → Job.spec.parallelism = Job.spec.completions`；`runPolicy.{activeDeadlineSeconds,ttlSecondsAfterFinished,backoffLimit} → Job.spec` 同名；其余 Pod 模板透传 |
| Suspend(MLJob) | 原生支持：patch `Job.spec.suspend=true`，返回 `suspendCompleted=true` |
| RBAC | `jobs.batch` / `pods` / `events` 的 CRUD |

#### 4.1.4 `(native, podgroup)` Handler

| 维度 | 取值 |
| --- | --- |
| 底层资源 | sigs.k8s.io scheduler-plugins `PodGroup` + 裸 Pod（单 role `worker`） |
| 必填字段 | `roles[worker]`，`template.image`，`scheduling.quota` |
| 关键字段映射 | `replicas → PodGroup.spec.minMember = 裸 Pod 数`；Pod 通过 label `pod-group.scheduling.sigs.k8s.io` 关联 PodGroup |
| Suspend(MLJob) | 先 patch `minMember=0`，再删 Pod（顺序约束：反过来 gang plugin 会立即重调度），最后返回 `suspendCompleted=true` |
| RBAC | `pods` / `podgroups.scheduling.sigs.k8s.io` / `events` 的 CRUD |

#### 4.1.5 `(kubeflow-trainer, *)` Handler

| 维度 | 取值 |
| --- | --- |
| 底层资源 | engine 对应的 Kubeflow CR：`PyTorchJob` / `TFJob` / `MPIJob` 等；多角色 gang 由 Handler 一并创建 PodGroup（`minMember = sum(replicas)`） |
| 必填字段 | engine 专属 role 集合：pytorchjob 至少 `worker`；tfjob 至少其一 `chief/worker/ps/evaluator`；mpijob 必含 `launcher`+`worker` |
| 关键字段映射 | `roles[*].template → CR 的 *ReplicaSpecs.<Role>.template`；各 replica 模板必须注入 `schedulerName=koord-scheduler` 与 [§5.2](#52-pod-注入约定) 全部 label；`runPolicy.{activeDeadlineSeconds,backoffLimit} → 后端 spec.runPolicy` 同名 |
| Suspend(MLJob) | 优先后端原生 `spec.runPolicy.suspend=true`；不支持的版本 fallback 为 `Cleanup()` |
| RBAC | Kubeflow CRD + `podgroups.scheduling.sigs.k8s.io` + `pods` / `events` 的 CRUD |

> engine 级完整 `backend.config` schema 与 status 映射细节见 §9。

#### 4.1.6 `(custom, *)` Handler

| 维度 | 取值 |
| --- | --- |
| 底层资源 | 用户在 `backend.config` 中以 schemaless 方式声明的目标 GVK；Handler 通过 unstructured client 创建并跟踪 |
| 必填字段 | `backend.config.targetGVK` + fieldMappings / statusMappings（待 RFC 定稿） |
| 关键字段映射 | JSONPath 驱动；最终落地的 Pod 模板必须含 `schedulerName=koord-scheduler` 与 Quota label，否则 `Validate` 拒绝 |
| Suspend(MLJob) | 兜底为 `Cleanup()`；自定义后端可在 `config` 中声明 suspend JSONPath |
| RBAC | 启动时由 Handler `RequiredRBAC()` 聚合到 operator ServiceAccount |

### 4.2 MLService Controller

#### 4.2.1 MLService spec 高层结构

```yaml
spec:
  backend:      { name, engine, config }
  scheduling:   { quota, priorityClass, nodeSelector, tolerations }
  modelRef:     { name, version }                              # 指向 Artifacts
  roles:        [{ name, replicas, template{ports,volumes,volumeMounts,…} }]
  runPolicy:    { progressDeadlineSeconds }
  route:        { enabled, targetRole, portName, hostname, path, auth, rateLimit, timeout }  # 可选
```

`roles[*].replicas` 是唯一允许由 API（`/scale`）变更的字段；`spec.modelRef` 切版本走重建；`spec.route` 整块不可变。

#### 4.2.2 状态机与事件路径

```
   ADD ──▶ Pending ──┬──▶ Ready ◀──▶ Degraded
                     │       ▲          │
                     └──▶ Failed ───────┘   (operator 自愈 → Informer 自然恢复)
```

四态全部非终态——`Deleted` 由 compute Informer 观察 CR DELETE 后基于 PG 推导。

| 事件 | Dispatcher 行为 | Handler 行为 |
| --- | --- | --- |
| MLService ADD | 路由；`Validate` 失败 → `status.phase=Failed` | `Reconcile` 创建底层资源 |
| UPDATE 仅 `roles[*].replicas`（来自 `/scale`） | 路由 | 透传为后端原生扩缩，不重建 Pod |
| UPDATE 其他 spec 字段 | 校验失败，`status.message` 拒绝 | 不动 |
| DELETE | 不阻断 | `ownerReference` 级联 |
| 底层资源事件 | 反查路由 | `MapStatus` 纯函数 |

#### 4.2.3 `(native, deployment)` Handler

| 维度 | 取值 |
| --- | --- |
| 底层资源 | K8s `Deployment` + `Service`（`targetPort=containerPort`）；不创建 PodGroup；`route.enabled=true` 时追加 HTTPRoute (+ 可选 SecurityPolicy / BackendTrafficPolicy) |
| 必填字段 | 单 role `predictor`，`template.image` + `template.ports[]` |
| 关键字段映射 | `replicas → Deployment.spec.replicas`；`template.volumes/volumeMounts → PodSpec` 同名（`Validate` 强制 volumeMounts 在同 role volumes 中、PVC 同 namespace）；`modelRef` → Artifacts 解析为 env `AXISML_MODEL_URI` |
| Scale(MLService) | patch `Deployment.spec.replicas`，不重建 Pod |
| RBAC | `deployments.apps` / `services` / `pods` / `events` + Gateway / Envoy CRD（按 `route` 开启）；`secrets` get/list/watch（仅 `apiKey` auth） |

#### 4.2.4 `(native, statefulset)` Handler

| 维度 | 取值 |
| --- | --- |
| 底层资源 | K8s `StatefulSet` + headless Service；副本身份稳定，透传 `apps.kubernetes.io/pod-index → axisml.io/replica-index` |
| 必填字段 | 单 role `predictor`，`template.image` + `template.ports[]` |
| 关键字段映射 | `replicas → StatefulSet.spec.replicas`；`config.podManagementPolicy`（默认 `OrderedReady`）、`config.serviceName`（默认 = MLService 名） |
| Scale(MLService) | patch `StatefulSet.spec.replicas` |
| RBAC | `statefulsets.apps` / `services` / `pods` / `events` 的 CRUD |

> `volumeClaimTemplates` / `updateStrategy` 等存储与灰度更新维度见 §9。

#### 4.2.5 `(kserve, inference)` Handler

| 维度 | 取值 |
| --- | --- |
| 底层资源 | KServe `InferenceService`（`serving.kserve.io/v1beta1`）；role 仅 `predictor` |
| 必填字段 | `backend.config.runtime`（`triton/vllm/tfserving/torchserve/sklearn/huggingface/<ServingRuntime>`）；runtime 专属约束由 `Validate` 强制（如 `vllm` 的 GPU = `tensorParallelSize × pipelineParallelSize`） |
| 关键字段映射 | `replicas → predictor.minReplicas`（必要时同时回填 `maxReplicas`）；`modelRef` → `predictor.storageUri`（或 runtime 专属字段）；Quota 注入 `predictor.schedulerName=koord-scheduler` + `predictor.labels`；**拒绝 `spec.route.enabled=true`** |
| Scale(MLService) | patch `predictor.{minReplicas, maxReplicas}` |
| RBAC | `inferenceservices.serving.kserve.io` / `pods` / `events` 的 CRUD |

#### 4.2.6 `(kserve, llminference)` Handler

| 维度 | 取值 |
| --- | --- |
| 底层资源 | KServe LLM API CR（GVK 演进中），承载 PD 分离（prefill / decode / router 三角色独立扩缩） |
| 必填字段 | role 集合 ⊆ `{prefill, decode, router}` 且至少含 `prefill`+`decode`；`backend.config.runtime` |
| 关键字段映射 | 各 role 独立 `replicas` → 后端对应字段；KV cache 传输（nixl / mooncake）走 `backend.config`；拒绝 `spec.route.enabled=true` |
| Scale(MLService) | 各 role 独立 patch；多 role 扩缩 API 扩展见 §9 |
| RBAC | KServe LLM CRD + `pods` / `events` 的 CRUD |

> 完整 schema、parallelism / autoscaler / KV cache 契约待 RFC，见 §9。

#### 4.2.7 `(custom, *)` Handler

| 维度 | 取值 |
| --- | --- |
| 底层资源 | schemaless 目标 GVK（unstructured client） |
| 必填字段 | `backend.config.targetGVK` + fieldMappings / statusMappings / endpointPath（待 RFC） |
| 关键字段映射 | JSONPath 驱动；Pod 模板必须含 `schedulerName=koord-scheduler` + Quota label |
| Scale(MLService) | 通过 `config` 中声明的 scale JSONPath patch；缺失时兜底为重建（应避免） |
| RBAC | Handler `RequiredRBAC()` 聚合 |

## 5. 关键机制

### 5.1 Dispatcher + Handler 路由

```
   <CR> ──▶ Dispatcher Reconciler ──路由(backend,engine)──▶ Handler
                                                              │
              ┌───────────────────────────────────────────────┤
              ▼                          ▼                    ▼
        Validate(spec)         Reconcile(ctx, cr)        WatchTargets()
        （纯函数）             （创建/对齐底层资源）      （声明 GVK）
                                        │
                                        ▼
                              MapStatus(snapshot) ──▶ dispatcher 合并写 status
```

**Handler 接口表**（语言无关，描述责任）：

| 方法 | 责任 |
| --- | --- |
| `Key()` | 返回 `(backend, engine)`；与 `spec.backend.{name,engine}` 对齐 |
| `Validate(spec)` | 通用字段 + `backend.config` + role 集合校验；纯函数，便于 admission webhook 复用 |
| `Reconcile(ctx, cr)` | 创建 / 对齐底层资源；幂等；返回结构化结果（含 suspend / scale 结果） |
| `MapStatus(snapshot)` | 后端原生状态 → 统一 phase + 公共字段；纯函数，便于状态回放 |
| `Cleanup(ctx, cr)` | 删除底层资源；一般依赖 ownerReference 级联 |
| `WatchTargets()` | 声明 watch 的底层资源 GVK；由 dispatcher 统一建立 controller-runtime `Watches()` |
| `RequiredRBAC()` | 声明 ClusterRole 规则；启动时聚合到 operator ServiceAccount |

**注册方式**：编译期 `init()` 注册到全局 registry，无运行时插件加载。**未注册组合的兜底**：`status.phase=Failed`、`status.message="no handler for backend=X engine=Y"`，不创建任何底层资源。**Handler 不直接写 `status`**——通过 `MapStatus` 与 `Reconcile` 返回值影响 status，dispatcher 统一通过 status subresource patch + resourceVersion 冲突重试合并写盘。

### 5.2 Pod 注入约定

所有 MLJob / MLService Handler（含 KServe 透传路径）派生的 Pod 必须满足以下注入，体现 [infra.md](../infra.md) 的 Quota 全覆盖不变式：

| Pod 字段 / Label | 必填 | 取值 | 用途 |
| --- | --- | --- | --- |
| `spec.schedulerName` | 是 | `koord-scheduler` | AxisML 所有 workload Pod 强制走 koord-scheduler |
| label `quota.scheduling.koordinator.sh/name` | 是 | `<spec.scheduling.quota>` | Koordinator ElasticQuota plugin 计入 `status.used` 的原生 label |
| label `axisml.io/{job-id\|service-id}` | 是 | UUID | 反查 MLJob / MLService，与 CR 同名 label 一致 |
| label `axisml.io/role` | 是 | role 名（`worker`/`master`/`predictor`/…） | 多角色拓扑区分 |
| label `axisml.io/quota` | 是 | `<spec.scheduling.quota>` | AxisML 自有审计 / 查询 |
| label `axisml.io/replica-index` | 否 | role 内 0-based 序号 | 仅在天然稳定时透传（StatefulSet `apps.kubernetes.io/pod-index`、Indexed Job `batch.kubernetes.io/job-completion-index`） |

缺失任一前 5 项视为契约违反，Handler `Validate` 必须创建前拦截。**KServe 派生 Pod 注入路径**：`(kserve, *)` 不直接控 podSpec，通过写 `InferenceService.spec.predictor.{schedulerName, labels}` 让 KServe 透传。

### 5.3 spec.route 派生资源

当 `route.enabled=true` 时，Handler 在 CR 所在 namespace 内创建以下资源，统一打 `axisml.io/service-id` label，`ownerReference: MLService` 级联清理：

| 资源 | GVK | 触发条件 | 关键字段 |
| --- | --- | --- | --- |
| `HTTPRoute` | `gateway.networking.k8s.io/v1` | 总开关 | `parentRefs → axisml-gateway`（跨 ns 通过 `ReferenceGrant`，由 infra chart 准备）；`backendRefs → route.targetRole` 对应的 K8s Service；`hostnames` / `rules[].matches[].path` 来自 `route.{hostname,path}` |
| `SecurityPolicy` | `gateway.envoyproxy.io/v1alpha1` | `auth.type != none` | `spec.jwt`（`issuer`/`jwksUri`）或 `spec.apiKeyAuth`（`secretRef`） |
| `BackendTrafficPolicy` | `gateway.envoyproxy.io/v1alpha1` | `rateLimit` 或 `timeout` 非空 | `spec.rateLimit` / `spec.timeout` |

`(kserve, *)` 拒绝 `route.enabled=true`——KServe 自带 route，`status.endpoint` 取 `InferenceService.status.url`。`(native,*)` / `(custom,*)` 在关闭时 `endpoint` 为 K8s Service DNS。

### 5.4 Status 单向回流与 cancel 推进信号

| 信号 | 写入方 | 读取方 | 备注 |
| --- | --- | --- | --- |
| `CR.status.phase` / `message` | operator dispatcher | compute Informer | 唯一被 compute 持久化的 phase 字段 |
| `CR.status.roles[*]` | operator dispatcher | UI / compute 观测 | role 级 replica 计数 |
| `MLService.status.endpoint` | operator dispatcher | compute Informer | 来源见 §5.3 |
| `MLJob.status.conditions[type=Suspended,status=True,reason=CancelRequested]` | dispatcher 合并 Handler 返回的 `suspendCompleted=true` | compute Informer | **cancel 闭环推进的唯一来源**；PG `Canceling → Cancelled → Delete()` 全靠它；缺失会卡住 |
| `CR.metadata` / `spec` | compute | operator 只读 | 单向；operator 永不向 compute PG 写入 |

**终态优先**：若底层资源已 `Succeeded` / `Failed`，cancel 信号被吞——`status.phase` 保留终态，`finishedAt` 不回退。

## 6. 接口契约

| 维度 | 内容 | 引用 |
| --- | --- | --- |
| CRD: MLJob | `axisml.io/v1alpha1`, Namespaced, shortName `mlj`；`status` subresource 必启 | [deploy/helm/axisml-system/crds/mljob-crd.yaml](../../../deploy/helm/axisml-system/crds/mljob-crd.yaml) |
| CRD: MLService | `axisml.io/v1alpha1`, Namespaced, shortName `mls`；`status` subresource 必启 | [deploy/helm/axisml-system/crds/mlservice-crd.yaml](../../../deploy/helm/axisml-system/crds/mlservice-crd.yaml) |
| 上游 compute 写契约 | `Create()` 幂等（重复返回 409 `AlreadyExists`，label `axisml.io/{job-id\|service-id}` 一致即视为成功）；`metadata`/`spec` 单向；`spec.runPolicy.suspend` 与 `roles[*].replicas` 是仅有的运行时可变路径；MLService 额外携带 `axisml.io/service-kind=<service\|workspace>` 稳定 label（operator 不消费，仅供 `kubectl` 与 selector 区分） | [compute-service.md](compute-service.md) |
| 路由元组 | MLJob: `(native,{job,podgroup}) \| (kubeflow-trainer,{pytorchjob,tfjob,mpijob,…}) \| (custom,*)`；MLService: `(native,{deployment,statefulset}) \| (kserve,{inference,llminference}) \| (custom,*)` | §4 |
| Pod 注入必填 | `spec.schedulerName=koord-scheduler` + 4 项 label（quota / job-id 或 service-id / role / quota 审计） | §5.2 |
| Status 回流字段 | `phase` / `message` / `roles[*]` / `conditions[type=Suspended]`（MLJob）/ `endpoint`（MLService） | §5.4 |
| 现状 schema | 两 CRD 的 `spec` / `status` 暂用 `x-kubernetes-preserve-unknown-fields: true`；严格 OpenAPI + admission webhook 见 §9 | — |

CRD 字段级 schema 不在本文展开，以上述 yaml + 后续 admission webhook 为准。

## 7. 依赖

| 依赖 | 用途 | 引用 |
| --- | --- | --- |
| Kubernetes API | 主 CR + 派生资源 CRUD；leader Lease | — |
| Koordinator (`koord-scheduler` + `koord-scheduler` ElasticQuota plugin) | 所有派生 Pod 强制 schedulerName + Quota label 计入 ElasticQuota；ElasticQuota CR 资源由 tenant-operator 维护，operator 只透传名字 | [infra.md](../infra.md) / [tenant-operator.md](tenant-operator.md) |
| scheduler-plugins `PodGroup`（Koordinator vendored） | `(native,podgroup)` 与 `(kubeflow-trainer,*)` 的 gang scheduling | [infra.md](../infra.md) |
| Kubeflow Training Operator | `(kubeflow-trainer,*)` 后端 CR 控制器 | [infra.md](../infra.md) |
| KServe | `(kserve,inference)` / `(kserve,llminference)` 后端 CR 控制器 | [infra.md](../infra.md) |
| Gateway API | `spec.route.enabled=true` 派生 `HTTPRoute`，挂到 `axisml-gateway` | [infra.md](../infra.md) |
| Envoy Gateway 扩展 (`SecurityPolicy` / `BackendTrafficPolicy`) | `route.auth` / `rateLimit` / `timeout` 派生 | [infra.md](../infra.md) |
| compute（上游 CR 写者） | 通过 `Create + Patch` 下发期望，status 单向回流；operator 不感知其 PG 表与 Outbox 推进机制 | [compute-service.md](compute-service.md) |
| artifacts | `spec.modelRef` 解析为 storageUri / env | [artifact-hub.md](artifact-hub.md) |

## 8. 运行时形态

| 维度 | 值 |
| --- | --- |
| 进程 | 单二进制 `axisml-compute-operator`，承载两个 dispatcher |
| 副本 | 单副本，K8s Lease leader election（`axisml-compute-operator.axisml.io`） |
| 探针 | `/healthz` / `/readyz` on `:8081`；Prometheus metrics on `:8080` |
| RBAC scope | 两 dispatcher 权限并集，按 `--enable-*` 分段渲染；**不含** `tenants.axisml.io` / `elasticquotas.scheduling.sigs.k8s.io` / `namespaces` / `secrets` / `configmaps` / `serviceaccounts`（属 tenant-operator） |
| 镜像 / Helm values / 资源 limit | 详见 [deployment.md](../deployment.md) |

**Flag 表**：

| Flag | 默认 | 说明 |
| --- | --- | --- |
| `--enable-mljob` | `true` | 启用 MLJob dispatcher；`false` 时对应 reconciler 不挂、ClusterRole 分段不渲染 |
| `--enable-mlservice` | `true` | 同上，MLService |
| `--leader-elect` | `true` | leader election 总开关 |
| `--leader-election-id` | `axisml-compute-operator.axisml.io` | Lease 名 |
| `--metrics-bind-address` | `:8080` | Prometheus 端口 |
| `--health-probe-bind-address` | `:8081` | 探针端口 |

## 9. 后续工作

- `(native, job)` Indexed Job 与 `podFailurePolicy` 直通策略细化。
- `(native, podgroup)` PodGroup `minResources` 与 elastic gang 演进。
- `(kubeflow-trainer, *)` 各 engine 完整字段映射 / 状态映射 / `backend.config` schema。
- `(native, statefulset)` `volumeClaimTemplates` / `updateStrategy` 灰度更新与 pod-index 寻址。
- `(kserve, inference)` 扩展 role（`transformer` / `explainer`）。
- `(kserve, llminference)` 完整 schema：vLLM disaggregated / llm-d / NVIDIA Dynamo 下的 KV cache 传输契约（nixl / mooncake）、parallelism schema、autoscaler 接入。
- KServe scale-to-zero 与 compute quota 的精细交互模型。
- `(custom, *)` Handler 的 `config` schema（JSONPath fieldMappings / statusMappings / endpointPath）与 unstructured 操作约定。
- 多 role 独立扩缩容的 `/scale` API 扩展。
- `spec.route` 可热更新路径；`spec.route` 与 KServe 自带 Route 的统一化。
- Admission webhook：`spec.backend.{name,engine}` 不可变约束、`backend.config` 按 Handler 自带 schema 统一校验，复用 `Validate` 纯函数。
- Handler chart values 控制（按 backend 启停 RBAC 与 watch）。
- CRD 严格 schema（移除 `x-kubernetes-preserve-unknown-fields`，启用 OpenAPI 校验）。
- 外部漂移检测（非 compute 主体 patch `spec` 时告警 / 拒绝）。

## 10. 相关引用

- [overview.md](../overview.md) — compute-operator 在控制平面拓扑中的位置
- [auth.md](../auth.md) — 身份与鉴权契约（operator 不直接认证终端用户）
- [database.md](../database.md) — compute / cluster-manager PG 表（operator 只读 CR，不触 PG）
- [deployment.md](../deployment.md) — Helm chart / 镜像 / 部署清单
- [monitoring.md](../monitoring.md) — Metrics 与告警
- [infra.md](../infra.md) — Koordinator / scheduler-plugins / Kubeflow / KServe / Gateway API / Envoy Gateway 依赖
- [compute-service.md](compute-service.md) — 上游 CR 写者；Outbox + 双 hash 推进机制
- [tenant-operator.md](tenant-operator.md) — 兄弟 operator；Tenant / ElasticQuota / Namespace 落地
- [artifact-hub.md](artifact-hub.md) — `spec.modelRef` 解析依赖的工件 registry
- CRD yaml：[deploy/helm/axisml-system/crds/mljob-crd.yaml](../../../deploy/helm/axisml-system/crds/mljob-crd.yaml) / [deploy/helm/axisml-system/crds/mlservice-crd.yaml](../../../deploy/helm/axisml-system/crds/mlservice-crd.yaml)
