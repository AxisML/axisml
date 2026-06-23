# AxisML Compute Operator 设计

## 1. 定位与边界

承载 `MLRun` / `MLService` / `MLTrafficPolicy` 三个 namespaced CR 的 Kubernetes operator；以 dispatcher + handler 模式把 [compute-service](compute-service.md) 下发的期望状态翻译为底层 K8s 与第三方资源，并把执行状态回流到 CR `status`。

| 做 | 不做 |
| --- | --- |
| 三类 CR reconcile，dispatcher 按 `spec.backend.{name, engine}` 路由 | Tenant / Namespace / ElasticQuota 落地 (→ [tenant-operator.md](tenant-operator.md)) |
| 派生 Job / Pod / Deployment / StatefulSet / HTTPRoute（kubeflow-trainer / kserve / custom 保留扩展点） | 业务持久化、用量计费、Outbox 推进 (→ [compute-service.md](compute-service.md)) |
| MLTrafficPolicy 派生加权 `HTTPRoute` / kserve canary | 流量策略的成员校验 / 权重权威 (→ [compute-service.md](compute-service.md)) |
| `spec.route` 派生 Gateway API + Envoy Gateway 扩展资源 | 模型工件存储 (→ [artifact-hub.md](artifact-hub.md)) |
| Cancel 推进信号（`Suspended` condition）单向回流 | 用户认证 / 鉴权 (→ [auth.md](../../axisml-platform/docs/auth.md)) |
| Pod 注入 `schedulerName=koord-scheduler` + Quota label | 写 compute-service PG / 跨集群联邦 |

## 2. 架构

```
 compute-service ──Create/Patch CR(spec)──▶ K8s: MLRun/MLService/MLTrafficPolicy
        ▲ status(watch)                              │ watch
        └──── CR.status 回流 ──── compute-operator ──┘
                                       │ 派生
              ┌────────────────────────┼────────────────────────┐
              ▼                        ▼                        ▼
   K8s 原生(Job/Pod/Deploy/STS/Svc)                  第三方(kubeflow/kserve, §9)
                                       ▼
                  Gateway API + Envoy Gateway 扩展(HTTPRoute/SecurityPolicy/BackendTrafficPolicy)
```

```
┌─── compute-operator (单 Pod, leader-elected) ───┐
│ ctrl.Manager  Lease: axisml-compute-operator    │
│ ┌ MLRun Dispatcher ┐┌ MLService ┐┌ MLTrafficPolicy ┐ │
│ │ (native,job)     ││(native,deploy)││(native,httproute)│ │
│ │                  ││(native,sts) ││(kserve,inference)* │ │
│ └──────────────────┘└─────────────┘└──────────────────┘ │
│   watch + status patch（三 dispatcher 共享 ctrl.Manager）│
│   各主 CR + ownerReference 派生底层 / 网关资源           │
└──────────────────────────────────────────────────────────┘
```

`*` `(kserve,inference)` 为保留扩展位（§9），当前仅交付 `(native,httproute)`。`--enable-ml{run,service,trafficpolicy}` 单独启停对应 dispatcher，未启用时其 ClusterRole 分段不渲染（最小权限）。当前仅交付 `native` 后端；`kubeflow-trainer` / `kserve` / `custom` 在接口上保留扩展点。

## 3. 核心模型

| 实体 | 含义 | 范围 | 状态机 | `backend` 路由元组 |
| --- | --- | --- | --- | --- |
| MLRun | 一次性批训练 / 离线任务 | Namespaced (`mlj`) | `Pending / Running / Succeeded / Failed` | `(native,job)`（其它见 §9） |
| MLService | 在线推理服务 | Namespaced (`mls`) | `Pending / Ready / Degraded / Failed` | `(native,deployment) ｜ (native,statefulset)` |
| MLTrafficPolicy | 在线服务流量编排（加权 / 灰度 / 蓝绿） | Namespaced (`mltp`) | `Pending / Ready / Degraded / Failed` | `(native,httproute)`（`(kserve,inference)` 见 §9） |

三个 CR 均 `axisml.io/v1alpha1`，`status` subresource 必启。`spec.scheduling.quota` 是 compute-service 透传的 ElasticQuota CR 名字符串，对 operator 不透明——ElasticQuota 资源由 [tenant-operator](tenant-operator.md) 独占维护。

实验的 Run 复用 `(native,job)`（训练亦可走 `(kubeflow-trainer,*)`）、TensorBoard 复用 `(native,deployment)`——均为上游业务编排在既有 backend 上的组合，operator **不新增 handler、不感知实验 / TensorBoard 概念**；分组靠 PG label（`axisml.io/experiment`）+ 上游。

## 4. 核心功能

### 4.1 MLRun Controller

```yaml
spec:
  backend:    { name, engine, config }     # 路由元组 + 后端专属 schemaless 配置
  scheduling: { quota, priorityClass, nodeSelector, tolerations }
  roles:      [{ name, replicas, restartPolicy, template }]
  runPolicy:  { suspend, activeDeadlineSeconds, ttlSecondsAfterFinished, backoffLimit }
```

`backend.{name,engine}` 创建后不可变，`runPolicy.suspend` 是唯一允许由 API（`/cancel`）翻转的字段。

```
ADD ─▶ Pending ─(any pod Running)─▶ Running ─┬─▶ Succeeded
                                              └─▶ Failed
suspend=true: Pending/Running ─(Handler 完成 suspend)─▶ conditions[Suspended,True,CancelRequested]（phase 维持；终态优先）
```

| 事件 | Dispatcher | Handler |
| --- | --- | --- |
| ADD | 路由；`Validate(spec)` 失败 → `phase=Failed` | `Reconcile` 创建底层资源 + 设 ownerReference |
| UPDATE(spec) | 校验 `backend.{name,engine}` 不变 | `Reconcile` 幂等更新 |
| `suspend=true` | 终态优先；非终态合并 Handler suspend 结果，写 `Suspended` condition | 执行原生 suspend 或 `Cleanup()`，返回 `suspendCompleted=true` |
| DELETE / 底层资源事件 | 不阻断（无 finalizer），ownerReference 级联 / 反查路由 | `MapStatus` 纯函数计算 phase |

**`(native, job)` Handler**：底层 K8s `Job`（单 role `worker`，不创建 PodGroup）；必填 `roles[worker]` + `template.image` + `scheduling.quota`；`replicas → parallelism = completions`，`runPolicy.{activeDeadlineSeconds,ttlSecondsAfterFinished,backoffLimit} → Job.spec`；Suspend 原生支持（patch `Job.spec.suspend=true`）；RBAC `jobs.batch` / `pods` / `events`。

### 4.2 MLService Controller

```yaml
spec:
  backend:    { name, engine, config }
  scheduling: { quota, priorityClass, nodeSelector, tolerations }
  roles:      [{ name, replicas, template{ports,volumes,volumeMounts,…} }]
  runPolicy:  { progressDeadlineSeconds }
  route:      { enabled, targetRole, portName, hostname, path, auth, rateLimit, timeout }  # 可选
```

`roles[*].replicas` 是唯一允许由 API（`/scale`）变更的字段；`spec.route` 整块不可变。

```
ADD ─▶ Pending ─┬─▶ Ready ◀──▶ Degraded
                └─▶ Failed（operator 自愈 → Informer 自然恢复）
```

四态全非终态——`Deleted` 由 compute-service Informer 观察 CR DELETE 后基于 PG 推导。UPDATE 仅 `roles[*].replicas`（来自 `/scale`）透传为后端原生扩缩、不重建 Pod；其他 spec 字段校验失败拒绝。

**Handler**：

| | `(native, deployment)` | `(native, statefulset)` |
| --- | --- | --- |
| 底层资源 | `Deployment` + `Service`；`route.enabled` 时追加 HTTPRoute；SecurityPolicy / BackendTrafficPolicy 尚未交付 | `StatefulSet` + headless Service；透传 `apps.kubernetes.io/pod-index → axisml.io/replica-index` |
| 必填 | 单 role `predictor`，`template.image` + `ports[]` | 同左 |
| 映射 | `replicas → Deployment.replicas`；`volumes/volumeMounts → PodSpec`（`Validate` 强制 volumeMounts 在同 role volumes、PVC 同 namespace） | `replicas → StatefulSet.replicas`；`config.podManagementPolicy`（默认 `OrderedReady`）、`config.serviceName`（默认 = MLService 名） |
| RBAC | `deployments.apps` / `services` / `pods` / `events` + Gateway / Envoy CRD（按 `route`）；`secrets` RO（仅 `apiKey`） | `statefulsets.apps` / `services` / `pods` / `events` |

> `volumeClaimTemplates` / `updateStrategy` 等见 §9。

### 4.3 MLTrafficPolicy Controller

把一个稳定入口的流量按权重分发到同 namespace 下多个在线服务（`MLService`，`kind=service`）后端。compute-service 已完成成员校验、占用判定、同构判定，并把派生的 `spec.backend` 元组写入 CR；operator 只消费 CR、解析成员 K8s Service、派生网关资源。

```yaml
spec:
  backend:  { name, engine, config }       # compute 派生：(native,httproute) ｜ (kserve,inference)
  mode:     weighted | canary | bluegreen  # 创建后不可变
  endpoint: { path, hostname, auth }       # 稳定对外入口；创建后不可变
  backends: [{ serviceName, role, weight }] # role: canary=stable|canary, bluegreen=blue|green；weight 0-100
```

`backend.{name,engine}` / `endpoint` / `mode` 创建后不可变；`backends[*].weight`（canary `promote` 时连同 `role` 互换）是唯一允许由上游 `/split`、`/promote`、`/rollback` 变更的字段。canary 当前基线即 `role=stable` 后端，**不设独立指针**。operator **不**反查成员后端族——以 compute 写入的 `spec.backend` 元组为路由依据。

```
ADD ─▶ Pending ─(route programmed + 成员 Ready)─▶ Ready ◀──▶ Degraded
                └─▶ Failed（operator 自愈 → Informer 自然恢复）
```

四态全非终态。UPDATE `backends[*].weight`（来自 split/promote/rollback）→ patch 派生路由的 `backendRefs` 权重 / kserve `canaryTrafficPercent`，不重建路由；DELETE → ownerReference 级联回收派生路由，**不触碰成员 MLService**。

**`(native, httproute)` Handler**：底层单条 `HTTPRoute`（`rules[].backendRefs[]` → 各成员 K8s Service，`weight = spec.backends[*].weight`）；`endpoint.auth.type=apiKey` 时追加 `SecurityPolicy`——本版本 apiKey 未交付，当前仅 `none`。`endpoint.{hostname,path} → HTTPRoute.{hostnames, rules[].matches[].path}`；`parentRefs → axisml-gateway`（跨 ns 复用 infra chart 的 `ReferenceGrant`）。split / promote / rollback 均 patch `backendRefs[*].weight`，原子生效、不重建路由。**不派生任何 Pod**——§5.2 Pod 注入不适用，计入配额的算力来自成员 MLService。RBAC `httproutes` / `securitypolicies` + `services` RO + `events`。

**`(kserve, inference)` Handler**（保留扩展位）：将 `mode=canary` 两成员映射为目标 `InferenceService` 的 `canaryTrafficPercent`；当前未交付（§9）。

## 5. 关键机制

### 5.1 Dispatcher + Handler 路由

```
<CR> ─▶ Dispatcher Reconciler ─路由(backend,engine)─▶ Handler
          ┌ Validate(spec) 纯函数 ┐┌ Reconcile(创建/对齐) ┐┌ WatchTargets(声明 GVK) ┐
                                    │ MapStatus(snapshot) ─▶ dispatcher 合并写 status
```

**Handler 接口**（语言无关）：`Key()`（返回 `(backend,engine)`）· `Validate(spec)`（通用 + `backend.config` + role 校验，纯函数，便于 admission 复用）· `Reconcile(ctx,cr)`（创建 / 对齐，幂等，返回含 suspend / scale 结果）· `MapStatus(snapshot)`（后端原生状态 → 统一 phase，纯函数）· `Cleanup(ctx,cr)`（一般依赖 ownerReference）· `WatchTargets()`（声明 watch GVK）· `RequiredRBAC()`（声明 ClusterRole 规则，启动聚合）。

编译期 `init()` 注册到全局 registry，无运行时插件。**未注册组合兜底**：`phase=Failed` + `message="no handler for backend=X engine=Y"`，不创建任何资源。**Handler 不直接写 `status`**——通过 `MapStatus` / `Reconcile` 返回值影响，dispatcher 统一经 status subresource patch + 冲突重试合并写盘。

### 5.2 Pod 注入约定

所有 MLRun / MLService Handler 派生的 Pod 必须满足以下注入，体现 [high_level_design §2.2](../../docs/system_design/high_level_design.md#22-关键不变量) 的 Quota 全覆盖不变式（未来第三方 backend 需保证同样语义）：

| Pod 字段 / Label | 必填 | 取值 | 用途 |
| --- | --- | --- | --- |
| `spec.schedulerName` | 是 | `koord-scheduler` | 所有 workload Pod 强制走 koord-scheduler |
| label `quota.scheduling.koordinator.sh/name` | 是 | `<spec.scheduling.quota>` | Koordinator ElasticQuota plugin 计入 `status.used` |
| label `axisml.io/{run-id｜service-id}` | 是 | UUID | 反查 MLRun / MLService |
| label `axisml.io/role` | 是 | role 名 | 多角色拓扑区分 |
| label `axisml.io/quota` | 是 | `<spec.scheduling.quota>` | AxisML 自有查询 |
| label `axisml.io/replica-index` | 否 | role 内 0-based 序号 | 仅天然稳定时透传（STS / Indexed Job） |

缺失任一前 5 项视为契约违反，Handler `Validate` 必须创建前拦截。第三方 backend 接入时同样必须透传 `schedulerName` + Quota label（具体策略待接入时设计，§9）。

**对象存储产出注入**：实验 Run 与 TensorBoard Pod 还在此注入对象存储访问（event log / checkpoint 的读写路径 + 凭证，路径约定 `experiments/<def>/runs/<run>/...`）；operator 不感知实验 / TensorBoard 概念，只按 [compute-service](compute-service.md) 在 spec 给定的注入项透传。MLTrafficPolicy handler 不派生 Pod，本节不适用。

### 5.3 spec.route 派生资源

`route.enabled=true` 时 Handler 在 CR namespace 内创建以下资源，统一打 `axisml.io/service-id` label、`ownerReference: MLService` 级联清理：

| 资源 | GVK | 触发 | 关键字段 |
| --- | --- | --- | --- |
| `HTTPRoute` | `gateway.networking.k8s.io/v1` | 总开关 | `parentRefs → axisml-gateway`（跨 ns 经 `ReferenceGrant`）；`backendRefs → route.targetRole` 对应 Service；`hostnames` / `path` 来自 `route` |
| `SecurityPolicy` | `gateway.envoyproxy.io/v1alpha1` | 规划中 | 当前不派生；`auth.type != none` 校验失败，避免未鉴权路由被创建 |
| `BackendTrafficPolicy` | `gateway.envoyproxy.io/v1alpha1` | 规划中 | 当前不派生，仅返回未生效 warning |

**与 MLTrafficPolicy 的职责切分**：`MLService.spec.route` 派生**单后端**对外入口（一条 HTTPRoute 指向自己的 Service）；`MLTrafficPolicy`（§4.3）派生**多后端加权**入口。二者互斥：被流量策略接管对外入口的成员 MLService 不应再开 `spec.route`（保持 ClusterIP 作 `backendRefs` 目标），由上游编排层在创建时保证，避免同一 hostname/path 路由冲突。

### 5.4 Status 单向回流与 cancel 推进信号

| 信号 | 写入方 | 读取方 |
| --- | --- | --- |
| `CR.status.phase` / `message` / `roles[*]` | operator dispatcher | compute-service Informer |
| `MLService.status.endpoint` | operator dispatcher | compute-service Informer |
| `MLTrafficPolicy.status.{endpoint, backends[*].{serviceName,weight,ready}}` | operator dispatcher | compute-service Informer（回源灰度健康视图） |
| `MLRun.status.conditions[Suspended,True,CancelRequested]` | dispatcher 合并 Handler 的 `suspendCompleted=true` | compute-service Informer——**cancel 闭环推进的唯一来源**，缺失会卡住 |
| `CR.metadata` / `spec` | compute-service | operator 只读（单向，永不向 PG 写） |

**终态优先**：底层资源已 `Succeeded` / `Failed` 时 cancel 信号被吞，`phase` 保留终态、`finishedAt` 不回退。

## 6. 接口契约

| 维度 | 内容 | 引用 |
| --- | --- | --- |
| CRD | MLRun(`mlj`) / MLService(`mls`) / MLTrafficPolicy(`mltp`)，`axisml.io/v1alpha1`, Namespaced，`status` subresource 必启 | crds/{mlrun,mlservice,mltrafficpolicy}-crd.yaml |
| 上游写契约 | `Create()` 幂等（重复 409 `AlreadyExists`，id label 一致即视为成功）；`metadata`/`spec` 单向；运行时可变路径仅 `suspend`(MLRun)/`roles[*].replicas`(MLService)/`backends[*].weight`(MLTrafficPolicy)；MLService 额外携带 `axisml.io/service-kind` 稳定 label（operator 不消费） | [compute-service.md](compute-service.md) |
| 路由元组 | MLRun `(native,job)`；MLService `(native,{deployment,statefulset})`；MLTrafficPolicy `(native,httproute)`（其它保留接口位，§9） | §4 |
| Pod 注入必填 | `schedulerName=koord-scheduler` + 4 项 label；MLTrafficPolicy 不派生 Pod，不适用 | §5.2 |
| Status 回流 | `phase`/`message`/`roles[*]`/`conditions[Suspended]`(MLRun)/`endpoint`(MLService)/`endpoint`+`backends[*]`(MLTrafficPolicy) | §5.4 |
| 现状 schema | `spec`/`status` 暂用 `x-kubernetes-preserve-unknown-fields: true`；严格 OpenAPI + admission 见 §9 | — |

**防御等级**：`metadata` / `spec` 单写约束当前由 dispatcher `Validate(spec)` 软兜底，**不防外部 `kubectl patch`**——系统在控制面信任边界内部署，admission webhook 是后续硬化路径（§9）。

## 7. 依赖

| 依赖 | 用途 |
| --- | --- |
| Kubernetes API | 主 CR + 派生资源 CRUD；leader Lease |
| Koordinator | 所有派生 Pod 强制 schedulerName + Quota label 计入 ElasticQuota；ElasticQuota 资源由 tenant-operator 维护，operator 只透传名字（[infra.md](../../axisml-infra/docs/overview.md) / [tenant-operator.md](tenant-operator.md)） |
| Gateway API + Envoy 扩展 | 当前派生 `HTTPRoute`；`SecurityPolicy` / `BackendTrafficPolicy` 尚未交付，认证配置 fail-closed（[infra.md](../../axisml-infra/docs/overview.md)） |
| compute-service（上游 CR 写者） | 通过 `Create + Patch` 下发期望，status 单向回流；operator 不感知其 PG 与 Outbox（[compute-service.md](compute-service.md)） |
| 对象存储（RustFS） | 按 spec 注入项把 Run / TensorBoard Pod 的 logdir / 产出路径 + 凭证透传进派生 Pod（§5.2） |

## 8. 运行时形态

| 维度 | 值 |
| --- | --- |
| 进程 | 单二进制 `axisml-compute-operator`，承载三个 dispatcher |
| 副本 | 单副本，K8s Lease leader election |
| 暴露端口 | Metrics `:8081`、Probes `:8082`；无 API 端口，无对外服务 |
| RBAC scope | 三 dispatcher 权限并集（含 `mltrafficpolicies` + `httproutes` / `securitypolicies` / `services` RO），按 `--enable-*` 分段渲染；**不含** `tenants` / `elasticquotas` / `namespaces` / `secrets` / `configmaps` / `serviceaccounts`（属 tenant-operator） |
| Flag | `--enable-ml{run,service,trafficpolicy}`（默认 `true`，`false` 时不挂 reconciler、不渲染 ClusterRole 分段）；`--leader-elect` / `--leader-election-id` / `--metrics-bind-address :8081` / `--health-probe-bind-address :8082` |
| 镜像 / Helm | 见 [deployment.md](../../docs/system_design/deployment.md) |

## 9. 相关引用

- [high_level_design.md](../../docs/system_design/high_level_design.md) — 控制平面拓扑与系统不变量
- [auth.md](../../axisml-platform/docs/auth.md) — 身份与鉴权契约（operator 不直接认证终端用户）
- [database.md](../../docs/system_design/database.md) · [deployment.md](../../docs/system_design/deployment.md) · [infra.md](../../axisml-infra/docs/overview.md)
- [compute-service.md](compute-service.md) — 上游 CR 写者
- [tenant-operator.md](tenant-operator.md) — 兄弟 operator；Tenant / ElasticQuota / Namespace 落地
- CRD yaml：crds/{mlrun,mlservice,mltrafficpolicy}-crd.yaml
