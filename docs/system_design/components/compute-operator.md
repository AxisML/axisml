# AxisML Compute Operator 概要设计

## 1. 定位与边界

承载 `MLRun` / `MLService` / `MLTrafficPolicy` 三个 namespaced CR 的 Kubernetes operator；以 dispatcher + handler 模式把 [compute-service](compute-service.md) 下发的期望状态翻译为底层 K8s 与第三方资源，并把执行状态回流到 CR `status`。

| 做 | 不做 |
| --- | --- |
| MLRun / MLService / MLTrafficPolicy CR reconcile，dispatcher 按 `spec.backend.{name, engine}` 路由 | Tenant / Namespace / ElasticQuota 落地 (→ [tenant-operator.md](tenant-operator.md)) |
| 派生 Job / Pod / Deployment / StatefulSet / HTTPRoute 等（kubeflow-trainer / kserve / 自定义后端在 dispatcher / handler 接口上保留扩展点） | 业务持久化、用量计费、Outbox 推进 (→ [compute-service.md](compute-service.md)) |
| MLTrafficPolicy 派生加权 `HTTPRoute` / kserve canary，编排一个稳定入口到多个在线服务的流量分发 | 流量策略的成员校验 / 权重权威 (→ [compute-service.md](compute-service.md)) |
| `spec.route` 派生 Gateway API + Envoy Gateway 扩展资源 | 模型工件存储 (→ [artifact-hub.md](artifact-hub.md)) |
| Cancel 推进信号（`Suspended` condition）单向回流 | 用户认证 / 鉴权 (→ [auth.md](../auth.md)) |
| Pod 注入 `schedulerName=koord-scheduler` + Quota label | 写 compute-service PG / 跨集群联邦 |

## 2. 架构

### 2.1 上下文

```
   ┌─────────────────┐  Create / Patch CR (spec)  ┌──────────────────────┐
   │ compute-service │ ──────────────────────────▶│ K8s: MLRun/MLService │
   └─────────────────┘                            │  + MLTrafficPolicy   │
          ▲                                       └─────────┬────────────┘
          │  status (watch)                                 │ watch
          │                                                 ▼
   ┌─────────────────┐                          ┌────────────────────────┐
   │ compute-service │◀─── CR.status 回流 ──────│   compute-operator     │
   └─────────────────┘                          └─────────┬──────────────┘
                                                         │ 派生
              ┌──────────────────────────────────────────┼──────────────────────────────────────────┐
              ▼                                          ▼                                          ▼
   K8s 原生 (Job/Pod/Deploy/StatefulSet/Svc)                                第三方 (kubeflow / kserve, 见 §9)
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
│  ┌──────────────┐  ┌──────────────────┐  ┌────────────────────────┐       │
│  │ MLRun        │  │ MLService        │  │ MLTrafficPolicy        │       │
│  │ Dispatcher   │  │ Dispatcher       │  │ Dispatcher             │       │
│  │ ─ registry ─ │  │ ─ registry ─     │  │ ─ registry ─           │       │
│  │ (native,job) │  │ (native,deploy)  │  │ (native,httproute)     │       │
│  │              │  │ (native,sts)     │  │ (kserve,inference)*    │       │
│  └──────┬───────┘  └────────┬─────────┘  └───────────┬────────────┘       │
│         │  watch + status patch（三 dispatcher 共享 ctrl.Manager）         │
│         ▼              ▼                        ▼                          │
│  各主 CR + ownerReference 派生底层 / 网关资源                               │
└────────────────────────────────────────────────────────────────────────────┘
```

`*` `(kserve,inference)` 为保留扩展位（见 §9），当前仅交付 `(native,httproute)`。`--enable-mlrun` / `--enable-mlservice` / `--enable-mltrafficpolicy` 单独启停对应 dispatcher，未启用时其 ClusterRole 分段也不渲染（最小权限）。

> 当前仅交付 `native` 后端；`kubeflow-trainer` / `kserve` / `custom` 在 dispatcher / handler 接口上保留扩展点，生产实现不在本文范围。

## 3. 核心模型

| 实体 | 含义 | 范围 | 状态机集合 | `backend` 路由元组 |
| --- | --- | --- | --- | --- |
| MLRun | 一次性批训练 / 离线任务 | Namespaced (`mlj`) | `Pending / Running / Succeeded / Failed` | `(native,job)`（其它扩展元组见 §9） |
| MLService | 在线推理服务 | Namespaced (`mls`) | `Pending / Ready / Degraded / Failed` | `(native,deployment) \| (native,statefulset)`（其它扩展元组见 §9） |
| MLTrafficPolicy | 在线服务流量编排（加权 / 灰度 / 蓝绿） | Namespaced (`mltp`) | `Pending / Ready / Degraded / Failed` | `(native,httproute)`（`(kserve,inference)` 见 §9） |

三个 CR 均使用 `axisml.io/v1alpha1`，`status` subresource 必启用。字段级 schema 见 [§6](#6-接口契约) 引用的 CRD yaml；spec 顶层结构与字段归属约定见 [§4.1.1](#411-mlrun-spec-高层结构) / [§4.2.1](#421-mlservice-spec-高层结构) / [§4.3.1](#431-mltrafficpolicy-spec-高层结构)。

`spec.scheduling.quota` 是 compute-service 透传的 ElasticQuota CR 名字符串，对 operator 不透明——ElasticQuota 资源本身由 [tenant-operator](tenant-operator.md) 独占维护。

实验的 Run 复用 `(native,job)`（训练亦可走 `(kubeflow-trainer,*)`）、TensorBoard 复用 `(native,deployment)`——均为 Platform 业务编排在既有 backend 上的组合，operator **不新增 handler、不感知实验 / TensorBoard 概念**；分组与编排靠 PG label（`axisml.io/experiment`）+ Platform，见 [platform.md §4.9–§4.10](platform.md#49-实验编排)。

## 4. 核心功能

### 4.1 MLRun Controller

#### 4.1.1 MLRun spec 高层结构

```yaml
spec:
  backend:      { name, engine, config }     # 路由元组 + 后端专属 schemaless 配置
  scheduling:   { quota, priorityClass, nodeSelector, tolerations }
  roles:        [{ name, replicas, restartPolicy, template }]   # 多角色拓扑
  runPolicy:    { suspend, activeDeadlineSeconds, ttlSecondsAfterFinished, backoffLimit }
```

字段完整 schema 与不可变性约定以 CRD yaml + 后续 admission webhook 为准；`spec.backend.{name,engine}` 创建后不可变，`spec.runPolicy.suspend` 是唯一允许由 API（`/cancel`）翻转的字段。

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
| MLRun ADD | 路由到 Handler；`Validate(spec)` 失败 → `status.phase=Failed` | `Reconcile` 创建底层资源 + 设置 `ownerReference` |
| MLRun UPDATE（spec） | 校验 `backend.{name,engine}` 不变；其余 spec 路由 | `Reconcile` 幂等更新 |
| `runPolicy.suspend=true` | 终态优先；非终态时合并 Handler 返回的 suspend 结果，写 `Suspended=True,reason=CancelRequested` | 执行原生 suspend 或 `Cleanup()`，返回 `suspendCompleted=true` |
| MLRun DELETE | 不阻断（无 finalizer） | 依赖 `ownerReference` 级联清理 |
| 底层资源事件 | 通过 `ownerReference` 反查路由 | `MapStatus` 纯函数计算新 phase |

#### 4.1.3 `(native, job)` Handler

| 维度 | 取值 |
| --- | --- |
| 底层资源 | K8s `Job`（单 role `worker`），不创建 PodGroup |
| 必填字段 | `roles[worker]`，`template.image`，`scheduling.quota` |
| 关键字段映射 | `replicas → Job.spec.parallelism = Job.spec.completions`；`runPolicy.{activeDeadlineSeconds,ttlSecondsAfterFinished,backoffLimit} → Job.spec` 同名；其余 Pod 模板透传 |
| Suspend(MLRun) | 原生支持：patch `Job.spec.suspend=true`，返回 `suspendCompleted=true` |
| RBAC | `jobs.batch` / `pods` / `events` 的 CRUD |

### 4.2 MLService Controller

#### 4.2.1 MLService spec 高层结构

```yaml
spec:
  backend:      { name, engine, config }
  scheduling:   { quota, priorityClass, nodeSelector, tolerations }
  roles:        [{ name, replicas, template{ports,volumes,volumeMounts,…} }]
  runPolicy:    { progressDeadlineSeconds }
  route:        { enabled, targetRole, portName, hostname, path, auth, rateLimit, timeout }  # 可选
```

`roles[*].replicas` 是唯一允许由 API（`/scale`）变更的字段；`spec.route` 整块不可变。

#### 4.2.2 状态机与事件路径

```
   ADD ──▶ Pending ──┬──▶ Ready ◀──▶ Degraded
                     │       ▲          │
                     └──▶ Failed ───────┘   (operator 自愈 → Informer 自然恢复)
```

四态全部非终态——`Deleted` 由 compute-service Informer 观察 CR DELETE 后基于 PG 推导。

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
| 关键字段映射 | `replicas → Deployment.spec.replicas`；`template.volumes/volumeMounts → PodSpec` 同名（`Validate` 强制 volumeMounts 在同 role volumes 中、PVC 同 namespace） |
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

### 4.3 MLTrafficPolicy Controller

把一个稳定对外入口的入站流量按权重分发到同 namespace 下多个在线服务（`MLService`，`kind=service`）后端。compute-service 已完成成员校验、占用判定与同构判定，并把派生的 `spec.backend` 元组写入 CR（见 [compute-service.md §4.5](compute-service.md#45-流量策略mltrafficpolicy)）；operator 只消费 CR、解析成员 K8s Service、派生网关资源。

#### 4.3.1 MLTrafficPolicy spec 高层结构

```yaml
spec:
  backend:     { name, engine, config }      # compute 按成员后端族派生：(native,httproute) | (kserve,inference)
  mode:        weighted | canary | bluegreen # 创建后不可变
  endpoint:    { path, hostname, auth }      # 稳定对外入口；创建后不可变
  backends:    [{ serviceName, role, weight }]  # 成员 MLService 引用 + role（canary: stable|canary；bluegreen: blue|green）+ 权重(0-100)
```

`spec.backend.{name,engine}` / `endpoint` / `mode` 创建后不可变；`backends[*].weight`（canary `promote` 时连同 `backends[*].role` 互换）是唯一允许由上游 `/split`、`/promote`、`/rollback` 变更的字段。canary 当前 stable 基线即 `role=stable` 的后端，**不设独立指针字段**。operator **不**反查成员后端族——以 compute 写入的 `spec.backend` 元组为路由依据。

#### 4.3.2 状态机与事件路径

```
   ADD ──▶ Pending ──(route programmed + 成员 Ready)──▶ Ready ◀──▶ Degraded
                     │                                    ▲          │
                     └──▶ Failed ─────────────────────────┴──────────┘   (operator 自愈 → Informer 自然恢复)
```

四态全部非终态——`Deleted` 由 compute-service Informer 观察 CR DELETE 后基于 PG 推导。

| 事件 | Dispatcher 行为 | Handler 行为 |
| --- | --- | --- |
| MLTrafficPolicy ADD | 路由到 Handler；`Validate` 失败 → `status.phase=Failed` | 解析成员 MLService 对应的 K8s Service → 创建加权路由（+ `SecurityPolicy`） |
| UPDATE `backends[*].weight`（来自 `/split` `/promote` `/rollback`） | 校验 `backend` / `endpoint` / `mode` 不变；路由 | patch 派生路由的 `backendRefs` 权重 / kserve `canaryTrafficPercent`，不重建路由 |
| UPDATE 其他字段 | 校验失败，`status.message` 拒绝 | 不动 |
| DELETE | 不阻断 | `ownerReference` 级联回收派生路由；**不触碰成员 MLService** |
| 成员 Service / 派生路由事件 | 反查路由 | `MapStatus` 纯函数计算 phase + 成员就绪 |

#### 4.3.3 `(native, httproute)` Handler

| 维度 | 取值 |
| --- | --- |
| 底层资源 | 单条 `HTTPRoute`（`rules[].backendRefs[]` → 各成员 MLService 对应的 K8s Service，`backendRefs[*].weight = spec.backends[*].weight`）；`endpoint.auth.type=apiKey` 时追加 `SecurityPolicy`（`apiKeyAuth`，`secretRef`）——本版本 apiKey 未交付，当前仅 `none` |
| 必填字段 | `backends[]` 非空且各成员 K8s Service 已存在；`endpoint.path` |
| 关键字段映射 | `endpoint.{hostname,path} → HTTPRoute.{hostnames, rules[].matches[].path}`；`parentRefs → axisml-gateway`（跨 ns 复用 infra chart 的 `ReferenceGrant`）；`backends[*].weight → rules[].backendRefs[*].weight` |
| Split / Promote / Rollback | patch `HTTPRoute.rules[].backendRefs[*].weight`，原子生效，不重建路由 |
| 派生 Pod | 无——流量策略不派生任何 Pod，[§5.2](#52-pod-注入约定) 的 Pod 注入不变式对本 handler 不适用 |
| RBAC | `httproutes` / `securitypolicies` 的 CRUD + `services` `get/list/watch`（解析成员 Service）+ `events` |

#### 4.3.4 `(kserve, inference)` Handler（保留扩展位）

将 `mode=canary` 的两成员映射为目标 `InferenceService` 的 `canaryTrafficPercent`（稳定 = default revision、灰度 = canary revision）；当前未交付，接口位同 §9 保留。

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

所有 MLRun / MLService Handler 派生的 Pod 必须满足以下注入，体现 [infra.md](../infra.md) 的 Quota 全覆盖不变式（未来接入的第三方 backend 需保证同样语义）：

| Pod 字段 / Label | 必填 | 取值 | 用途 |
| --- | --- | --- | --- |
| `spec.schedulerName` | 是 | `koord-scheduler` | AxisML 所有 workload Pod 强制走 koord-scheduler |
| label `quota.scheduling.koordinator.sh/name` | 是 | `<spec.scheduling.quota>` | Koordinator ElasticQuota plugin 计入 `status.used` 的原生 label |
| label `axisml.io/{run-id\|service-id}` | 是 | UUID | 反查 MLRun / MLService，与 CR 同名 label 一致 |
| label `axisml.io/role` | 是 | role 名（`worker`/`master`/`predictor`/…） | 多角色拓扑区分 |
| label `axisml.io/quota` | 是 | `<spec.scheduling.quota>` | AxisML 自有审计 / 查询 |
| label `axisml.io/replica-index` | 否 | role 内 0-based 序号 | 仅在天然稳定时透传（StatefulSet `apps.kubernetes.io/pod-index`、Indexed Job `batch.kubernetes.io/job-completion-index`） |

缺失任一前 5 项视为契约违反，Handler `Validate` 必须创建前拦截。第三方 backend（kserve / kubeflow-trainer / custom）接入时同样必须把 `schedulerName` + Quota label 透传到派生 Pod，否则视为不合规接入路径——具体透传策略待对应 backend 接入时设计（见 §9）。

**对象存储产出注入**：实验 Run 与 TensorBoard 服务的 Pod 还在此处注入对象存储访问（event log / checkpoint 的写或读路径 + 凭证，路径约定 `experiments/<def>/runs/<run>/...`），使产出落对象存储而非 PG；operator 不感知实验 / TensorBoard 概念，只按 [compute-service](compute-service.md) 在 spec 里给定的注入项透传。

MLTrafficPolicy handler 只派生网关路由资源（`HTTPRoute` / `SecurityPolicy`）、不派生任何 Pod，故本节 Pod 注入不变式对其不适用——计入 ElasticQuota 的算力始终来自被引用的成员 MLService 的 Pod，流量策略本身不占用配额。

### 5.3 spec.route 派生资源

当 `route.enabled=true` 时，Handler 在 CR 所在 namespace 内创建以下资源，统一打 `axisml.io/service-id` label，`ownerReference: MLService` 级联清理：

| 资源 | GVK | 触发条件 | 关键字段 |
| --- | --- | --- | --- |
| `HTTPRoute` | `gateway.networking.k8s.io/v1` | 总开关 | `parentRefs → axisml-gateway`（跨 ns 通过 `ReferenceGrant`，由 infra chart 准备）；`backendRefs → route.targetRole` 对应的 K8s Service；`hostnames` / `rules[].matches[].path` 来自 `route.{hostname,path}` |
| `SecurityPolicy` | `gateway.envoyproxy.io/v1alpha1` | `auth.type != none` | `spec.jwt`（`issuer`/`jwksUri`）或 `spec.apiKeyAuth`（`secretRef`） |
| `BackendTrafficPolicy` | `gateway.envoyproxy.io/v1alpha1` | `rateLimit` 或 `timeout` 非空 | `spec.rateLimit` / `spec.timeout` |

`(native,*)` 关闭 `route` 时 `endpoint` 为 K8s Service DNS；其它后端（kserve / custom）接入时的 route 与 endpoint 策略另行设计（见 §9）。

**与 MLTrafficPolicy 的职责切分**：本节 `MLService.spec.route` 派生的是**单后端**对外入口（一条 `HTTPRoute` 指向自己的 K8s Service）；[§4.3](#43-mltrafficpolicy-controller) 的 `MLTrafficPolicy` 派生的是**多后端加权**入口（一条 `HTTPRoute` 跨多个成员 Service 加权）。二者互斥：被流量策略接管对外入口的成员 MLService 不应再开 `spec.route`（保持 ClusterIP Service 作为 `backendRefs` 目标），由 Platform 编排层在创建时保证，避免同一 hostname/path 上的路由冲突。

### 5.4 Status 单向回流与 cancel 推进信号

| 信号 | 写入方 | 读取方 | 备注 |
| --- | --- | --- | --- |
| `CR.status.phase` / `message` | operator dispatcher | compute-service Informer | 唯一被 compute-service 持久化的 phase 字段 |
| `CR.status.roles[*]` | operator dispatcher | UI / compute-service 观测 | role 级 replica 计数 |
| `MLService.status.endpoint` | operator dispatcher | compute-service Informer | 来源见 §5.3 |
| `MLTrafficPolicy.status.endpoint` | operator dispatcher | compute-service Informer | 稳定对外入口 URL；来源见 §4.3.3 |
| `MLTrafficPolicy.status.backends[*].{serviceName,weight,ready}` | operator dispatcher | compute-service Informer | 成员就绪与生效权重，回源给 Platform 的灰度健康视图 |
| `MLRun.status.conditions[type=Suspended,status=True,reason=CancelRequested]` | dispatcher 合并 Handler 返回的 `suspendCompleted=true` | compute-service Informer | **cancel 闭环推进的唯一来源**；PG `Canceling → Cancelled → Delete()` 全靠它；缺失会卡住 |
| `CR.metadata` / `spec` | compute-service | operator 只读 | 单向；operator 永不向 compute-service PG 写入 |

**终态优先**：若底层资源已 `Succeeded` / `Failed`，cancel 信号被吞——`status.phase` 保留终态，`finishedAt` 不回退。

## 6. 接口契约

| 维度 | 内容 | 引用 |
| --- | --- | --- |
| CRD: MLRun | `axisml.io/v1alpha1`, Namespaced, shortName `mlj`；`status` subresource 必启 | [deploy/helm/axisml-system/crds/mlrun-crd.yaml](../../../deploy/helm/axisml-system/crds/mlrun-crd.yaml) |
| CRD: MLService | `axisml.io/v1alpha1`, Namespaced, shortName `mls`；`status` subresource 必启 | [deploy/helm/axisml-system/crds/mlservice-crd.yaml](../../../deploy/helm/axisml-system/crds/mlservice-crd.yaml) |
| CRD: MLTrafficPolicy | `axisml.io/v1alpha1`, Namespaced, shortName `mltp`；`status` subresource 必启 | [deploy/helm/axisml-system/crds/mltrafficpolicy-crd.yaml](../../../deploy/helm/axisml-system/crds/mltrafficpolicy-crd.yaml) |
| 上游 compute-service 写契约 | `Create()` 幂等（重复返回 409 `AlreadyExists`，label `axisml.io/{run-id\|service-id\|traffic-policy-id}` 一致即视为成功）；`metadata`/`spec` 单向；`spec.runPolicy.suspend`（MLRun）、`roles[*].replicas`（MLService）、`backends[*].weight`（MLTrafficPolicy）是仅有的运行时可变路径；MLService 额外携带 `axisml.io/service-kind=<service\|workspace>` 稳定 label（operator 不消费，仅供 `kubectl` 与 selector 区分） | [compute-service.md](compute-service.md) |
| 路由元组 | MLRun: `(native,job)`；MLService: `(native,{deployment,statefulset})`；MLTrafficPolicy: `(native,httproute)`（其它扩展元组保留接口位，未交付——见 §9） | §4 |
| Pod 注入必填 | `spec.schedulerName=koord-scheduler` + 4 项 label（quota / run-id 或 service-id / role / quota 审计）；MLTrafficPolicy 不派生 Pod，不适用 | §5.2 |
| Status 回流字段 | `phase` / `message` / `roles[*]` / `conditions[type=Suspended]`（MLRun）/ `endpoint`（MLService）/ `endpoint` + `backends[*].{serviceName,weight,ready}`（MLTrafficPolicy） | §5.4 |
| 现状 schema | 两 CRD 的 `spec` / `status` 暂用 `x-kubernetes-preserve-unknown-fields: true`；严格 OpenAPI + admission webhook 见 §9 | — |

CRD 字段级 schema 不在本文展开，以上述 yaml + 后续 admission webhook 为准。

**防御等级**：`metadata` / `spec` 单写约束（compute-service 为唯一写者）当前由 dispatcher `Validate(spec)` 软兜底，**不防止外部直接 `kubectl patch` 改 CR 的攻击面**——系统目前在控制面信任边界内部署，admission webhook 是后续硬化路径（见 §9）。

## 7. 依赖

| 依赖 | 用途 | 引用 |
| --- | --- | --- |
| Kubernetes API | 主 CR + 派生资源 CRUD；leader Lease | — |
| Koordinator (`koord-scheduler` + `koord-scheduler` ElasticQuota plugin) | 所有派生 Pod 强制 schedulerName + Quota label 计入 ElasticQuota；ElasticQuota CR 资源由 tenant-operator 维护，operator 只透传名字 | [infra.md](../infra.md) / [tenant-operator.md](tenant-operator.md) |
| Gateway API | `MLService.spec.route.enabled=true` 派生单后端 `HTTPRoute`；`MLTrafficPolicy` 派生多后端加权 `HTTPRoute`；均挂到 `axisml-gateway` | [infra.md](../infra.md) |
| Envoy Gateway 扩展 (`SecurityPolicy` / `BackendTrafficPolicy`) | `route.auth` / `rateLimit` / `timeout` 派生；`MLTrafficPolicy.endpoint.auth=jwt` 派生 `SecurityPolicy` | [infra.md](../infra.md) |
| compute-service（上游 CR 写者） | 通过 `Create + Patch` 下发期望，status 单向回流；operator 不感知其 PG 表与 Outbox 推进机制 | [compute-service.md](compute-service.md) |
| 对象存储（RustFS） | 按 spec 注入项把 Run / TensorBoard Pod 的对象存储 logdir / 产出路径 + 凭证透传进派生 Pod（§5.2）；operator 不解析产出内容 | [infra.md](../infra.md) |

## 8. 运行时形态

| 维度 | 值 |
| --- | --- |
| 进程 | 单二进制 `axisml-compute-operator`，承载三个 dispatcher |
| 副本 | 单副本，K8s Lease leader election（`axisml-compute-operator.axisml.io`） |
| 暴露端口 | Metrics `:8081`、Probes `:8082`（`/healthz` / `/readyz`）；无 API 端口，无对外服务 |
| RBAC scope | 三 dispatcher 权限并集（含 `mltrafficpolicies.axisml.io` + `httproutes` / `securitypolicies` / `services` RO），按 `--enable-*` 分段渲染；**不含** `tenants.axisml.io` / `elasticquotas.scheduling.sigs.k8s.io` / `namespaces` / `secrets` / `configmaps` / `serviceaccounts`（属 tenant-operator） |
| 镜像 / Helm values / 资源 limit | 详见 [deployment.md](../deployment.md) |

**Flag 表**：

| Flag | 默认 | 说明 |
| --- | --- | --- |
| `--enable-mlrun` | `true` | 启用 MLRun dispatcher；`false` 时对应 reconciler 不挂、ClusterRole 分段不渲染 |
| `--enable-mlservice` | `true` | 同上，MLService |
| `--enable-mltrafficpolicy` | `true` | 同上，MLTrafficPolicy dispatcher |
| `--leader-elect` | `true` | leader election 总开关 |
| `--leader-election-id` | `axisml-compute-operator.axisml.io` | Lease 名 |
| `--metrics-bind-address` | `:8081` | Prometheus 端口 |
| `--health-probe-bind-address` | `:8082` | 探针端口 |

## 9. 相关引用

- [overview.md](../overview.md) — compute-operator 在控制平面拓扑中的位置
- [auth.md](../auth.md) — 身份与鉴权契约（operator 不直接认证终端用户）
- [database.md](../database.md) — compute-service / cluster-manager PG 表（operator 只读 CR，不触 PG）
- [deployment.md](../deployment.md) — Helm chart / 镜像 / 部署清单
- [monitoring.md](../monitoring.md) — Metrics 与告警
- [infra.md](../infra.md) — Koordinator / scheduler-plugins / Kubeflow / KServe / Gateway API / Envoy Gateway 依赖
- [compute-service.md](compute-service.md) — 上游 CR 写者；Outbox + 双 hash 推进机制
- [tenant-operator.md](tenant-operator.md) — 兄弟 operator；Tenant / ElasticQuota / Namespace 落地
- CRD yaml：[deploy/helm/axisml-system/crds/mlrun-crd.yaml](../../../deploy/helm/axisml-system/crds/mlrun-crd.yaml) / [deploy/helm/axisml-system/crds/mlservice-crd.yaml](../../../deploy/helm/axisml-system/crds/mlservice-crd.yaml) / [deploy/helm/axisml-system/crds/mltrafficpolicy-crd.yaml](../../../deploy/helm/axisml-system/crds/mltrafficpolicy-crd.yaml)
