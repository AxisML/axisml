# axisml-operator 详细设计

axisml-operator 是 AxisML 控制平面里**唯一**的 Kubernetes operator 二进制，由一个 Manager 同时承载三个 controller：

| Controller | CRD（`axisml.io/v1alpha1`） | Scope | 架构 | 状态机 | 主要外部依赖 |
| --- | --- | --- | --- | --- | --- |
| Tenant ([§4](#4-tenant-controller)) | `Tenant` | Cluster | 单 reconciler | `Active \| Suspended \| Failed` | Koordinator ElasticQuota |
| MLJob ([§5](#5-mljob-controller)) | `MLJob` | Namespaced | dispatcher + handler 注册表 | `Pending \| Running \| Succeeded \| Failed` | scheduler-plugins PodGroup（Koordinator vendored）、Kubeflow Trainer |
| MLService ([§6](#6-mlservice-controller)) | `MLService` | Namespaced | dispatcher + handler 注册表 | `Pending \| Ready \| Degraded \| Failed` | Gateway API HTTPRoute、KServe |

**文档组织**：

- **Part I — 运行时框架**（§1 架构总览 + §2 运行时契约）：单一 Deployment / Manager 的运维契约（Scheme、Cache、Flag、RBAC、Helm values）。
- **Part II — 通用契约**（§3 跨 controller 通用契约）：三个 controller 共享的 spec/status 边界、Reconcile 约束、Pod 注入、dispatcher + handler 架构；§4–§6 引用本节而不重复。
- **Part III — Controller 详细设计**（§4 Tenant、§5 MLJob、§6 MLService）：各 controller 的 CRD 字段、状态推导、Handler 落地。
- **Part IV — 实施与验证**（§7 实现路径、§8 测试、§9 相关引用）：功能落地路线（MVP / 功能完善 / 未来规划）、测试层次、跨文档引用。

三个 controller 通过 `--enable-{tenant,mljob,mlservice}` 单独启用 / 关闭，详见 §2.3。

---

## Part I — 运行时框架

> 本部分描述 operator binary 与 Kubernetes Manager 的运维契约：Scheme 注册、Cache 过滤、CLI flag、RBAC 与 Helm values。三个 controller 共享这些契约。

## 1. 架构总览

```
┌──────────────────── axisml-operator (one Pod, leader-elected) ────────────────────┐
│                                                                                    │
│  ctrl.Manager (scheme: clientgoscheme + axisml + scheduling.sigs.k8s.io +          │
│                gateway.networking.k8s.io)                                          │
│  Lease: axisml-operator.axisml.io                                                  │
│                                                                                    │
│  ┌──────────────────┐  ┌────────────────────────┐  ┌───────────────────────────┐   │
│  │ Tenant           │  │ MLJob                  │  │ MLService                 │   │
│  │ Reconciler       │  │ Dispatcher + Registry  │  │ Dispatcher + Registry     │   │
│  │ (single, no      │  │ → handlers/{nativejob, │  │ → handlers/{nativedeploy, │   │
│  │  dispatcher)     │  │   nativepodgroup,      │  │   nativestatefulset,      │   │
│  │                  │  │   kubeflow-*, custom}  │  │   kserve-*, custom}       │   │
│  └──────────────────┘  └────────────────────────┘  └───────────────────────────┘   │
│        │                       │                            │                      │
│        ▼                       ▼                            ▼                      │
│   Namespace,                Job, Pod, PodGroup        Deployment, Service,         │
│   ElasticQuota,             (koord-scheduler          HTTPRoute (Gateway API),     │
│   Secret/CM/SA/             gang scheduling)          KServe InferenceService      │
│   Role/RoleBinding                                                                 │
└────────────────────────────────────────────────────────────────────────────────────┘
```

Tenant 走单 reconciler 直接调度（无 dispatcher，没有多后端实现）。MLJob 与 MLService 同构使用 **dispatcher + handler** 模式：CR 的 `spec.backend.{name, engine}` 元组路由到注册过的 Handler，Handler 渲染目标 GVK 并把状态回流到 CR.status——见 §3.5。所有 backend 派生的 Pod 强制走 koord-scheduler 并消费对应 ElasticQuota（[infra.md §8](infra.md)）。

## 2. 运行时契约

### 2.1 Scheme 注册

```go
clientgoscheme.AddToScheme(scheme)           // core, apps, rbac, batch, coordination
schedulingv1alpha1.AddToScheme(scheme)       // ElasticQuota + PodGroup（Koordinator vendored）
gwapiv1.Install(scheme)                      // HTTPRoute
tenant_v1alpha1.AddToScheme(scheme)          // Tenant
mljob_v1alpha1.AddToScheme(scheme)           // MLJob
mlservice_v1alpha1.AddToScheme(scheme)       // MLService
mlservicehandler.RegisterStubs()             // MLService 占位 handler 注册
```

三个 CRD 共享 group `axisml.io/v1alpha1`，但 Go 类型分别定义在 `components/operator/api/{tenant,mljob,mlservice}/v1alpha1/` 三个子包里——避免 `Phase`、`RoleSpec`、`LabelQuota` 等同名常量在同一包内冲突，同时仍然让一个 Manager 通过分别 `AddToScheme` 把三种 Kind 全注册进去。实现细节：每个子包的 `groupversion_info.go` 用 `runtime.SchemeBuilder` 声明 `SchemeBuilder` + 私有 `addKnownTypes(...)` helper；各 `<kind>_types.go` 的 `init()` 调用 `addKnownTypes(&Foo{}, &FooList{})` 完成注册。

### 2.2 Cache 选择性过滤

Tenant 的子资源（Secret / ConfigMap / ServiceAccount / Role / RoleBinding / ElasticQuota）受 `managed-by=tenant-operator` label 过滤，避免缓存全集群 Secret。**关键约束：这条过滤必须按对象类型挂在 `cache.Options.ByObject` 上**，不能升格成 `cache.Options.DefaultLabelSelector`——否则 MLJob 的 `Job/Pod/PodGroup` informer 与 MLService 的 `Deployment/HTTPRoute` informer 会被同样的 label 过滤掉，导致丢事件。

```go
cache.Options{
    SyncPeriod: &resync,
    ByObject: map[client.Object]cache.ByObject{
        &corev1.Secret{}:                   {Label: managedByOnly},
        &corev1.ConfigMap{}:                {Label: managedByOnly},
        &corev1.ServiceAccount{}:           {Label: managedByOnly},
        &rbacv1.Role{}:                     {Label: managedByOnly},
        &rbacv1.RoleBinding{}:              {Label: managedByOnly},
        &schedulingv1alpha1.ElasticQuota{}: {Label: managedByOnly},
    },
}
```

注意：`PodGroup` 与 `ElasticQuota` 都属于 `scheduling.sigs.k8s.io/v1alpha1`，但只有 `ElasticQuota` 由 Tenant 写入并打 `managed-by` label。`PodGroup` 由 MLJob handler 写入、不带这条 label，因此在表里**只列 `ElasticQuota`**。

### 2.3 Flag 集合

| Flag | 默认 | 说明 |
| --- | --- | --- |
| `--metrics-bind-address` | `:8080` | Prometheus metrics |
| `--health-probe-bind-address` | `:8081` | `/healthz`, `/readyz` |
| `--leader-elect` | `true` | leader election |
| `--leader-election-id` | `axisml-operator.axisml.io` | Lease 名 |
| `--enable-tenant` | `true` | 启用 Tenant controller |
| `--enable-mljob` | `true` | 启用 MLJob controller |
| `--enable-mlservice` | `true` | 启用 MLService controller |

Pod 上还会注入两个环境变量供 Tenant 子模块消费：`RESYNC_PERIOD`（默认 `10m`）、`NAMESPACE_DENYLIST`（逗号分隔列表，默认值见 Helm `values.yaml`）。

### 2.4 RBAC

整个 axisml-operator 只声明**一个** ClusterRole（`<release>-operator`），rules 是三个 controller 所需权限的并集，按 controller 分段；段头按 `--enable-*` Helm value 条件渲染（见 `deploy/helm/axisml-system/templates/operator/clusterrole.yaml`）。leader election Lease 在部署 namespace 通过 Role + RoleBinding 授权（不放进 cluster-scoped 角色）。Pod 资源在三个 controller 间存在共用（MLJob 需要 get/list/watch/delete，MLService 需要 CRUD），渲染时按"是否需要写权限"的并集生成单条规则，不重复声明。

各 controller 自身的 RBAC 明细：见 §4.7（Tenant）、§5.5 RBAC 行（每个 MLJob handler）、§6.6 RBAC 行（每个 MLService handler）。

### 2.5 Helm values 接口

```yaml
operator:
  enabled: true
  image: { registry, repository, tag, pullPolicy }
  leaderElection: { enabled, id }
  resources: { requests, limits }
  controllers:
    tenant:    { enabled, resyncPeriod, namespaceDenylist }
    mljob:     { enabled }
    mlservice: { enabled }
```

`controllers.<name>.enabled=false` 时对应 controller 的 reconciler 不挂到 Manager，且 ClusterRole 中相关分段不渲染——做到"按需启用"的最小权限。

---

## Part II — 通用契约

> 本部分集中三个 controller **共享**的边界与协议：与 Compute 的写路径、CRD 共同字段约束、Reconcile 行为、Pod 注入约定、dispatcher + handler 架构、Handler 接口契约。§4–§6 各 controller 章节引用本节而不重复。

## 3. 跨 controller 通用契约

§3 列出三个 controller **共享**的 spec/status 边界、Reconcile 行为、底层资源约定。§4–§6 引用本节而不重复。

### 3.1 与 Compute 的写路径契约

Compute 采用 Operation Outbox + reconciler 异步下发 CR（详见 [compute.md §5.2](compute.md)）。Operator 暴露给 Compute 的核心契约对所有三个 CRD 都成立：

- **`Create` 幂等**：相同 `metadata.name` 的二次 `Create()` 返回 409 `AlreadyExists`，且不引发副作用（不重建底层资源、不重置 status）。Compute 收到 409 后会 `Get()` 现有 CR 并校验 label `axisml.io/{tenant-id|job-id|service-id}=<uuid>`；只有 label 一致才视为成功。
- **status 单向权威**：operator 只写 `<CR>.status`，Compute 只写 `<CR>.metadata` / `<CR>.spec`；状态推进由 Compute 侧 Informer 按 CR `status` 消费。**operator 不感知 Compute 的 PG 表，也不向 Compute PG 写入任何数据**——状态与配额用量全部经由 CR `status` 回流。
- **配置补偿**：CR 被误删后由 Compute 按 PG 快照判定是否重建（[compute.md §5.4](compute.md)）；operator 的 `Reconcile` 必须可在已存在的底层资源上幂等收敛——已存在的资源不重建，只对齐 spec 漂移。

各 CRD 还有自身的额外写路径约束（如 Tenant 的"Namespace 永不级联删除"），见 §4.2 / §5.2 / §6.2。

### 3.2 CRD 共同约束

**metadata 由 Compute 设置**（与 [compute.md §6](compute.md) 对齐）：

- `metadata.name` ← Compute 的对应业务对象 name；DNS-1123 + ≤40 字符（业务硬校验，[compute.md §6](compute.md)）。
- `metadata.namespace`（仅 namespaced CRD MLJob/MLService）← `tenants.namespace`。
- `metadata.labels["axisml.io/{tenant,job,service}-id"]` ← UUID，孤儿检测稳定锚点。
- `metadata.labels["axisml.io/tenant"]` / `["axisml.io/quota"]`（仅 MLJob/MLService）：透传给 Pod 用于审计 / 查询。

**status subresource 必启用**：三个 CRD（`tenant-crd.yaml` / `mljob-crd.yaml` / `mlservice-crd.yaml`）都必须声明 `subresources.status`，由 Kubernetes API Server 隔离 controller 写 `status` 与 Compute 写 `metadata` / `spec` 的边界。当前 CRD 文件尚未声明，属于实现对齐项；本文档先锁定契约。

**当前 CRD schema 现状**：三个 CRD 的 `spec` / `status` 暂用 `x-kubernetes-preserve-unknown-fields: true`，重新设计字段不需 CRD bump；待行为稳定后再启用 OpenAPI schema 严格校验（各 controller 自身 §x.7 后续工作中提到的 enum / required 强校验）。

**phase 集合冻结**：Tenant 三态 / MLJob 与 MLService 各自四态（见 §1 总览表）。新增 phase 必须经 CRD schema 与 Compute 双侧同步演进。

### 3.3 Reconcile 通用约束

下列约束对三个 controller 都生效：

- **不引入 finalizer**：Tenant / MLJob / MLService 三个 controller 一律不挂 finalizer；级联清理依赖 `ownerReference`（cluster-scoped Tenant → namespaced 子资源；CR → Handler 创建的底层资源）。
- **`Validate(spec)` 必须是纯函数**：不发起 K8s 调用，便于未来在 admission webhook 中复用。校验失败 → `status.phase=Failed`、`status.message` 写明违规项。
- **`Reconcile` 幂等**：多次调用相同 spec 不重建底层资源；只有语义字段变化才触发底层资源更新。
- **Status 写盘单一路径**：reconcile 末尾通过一次 patch 完成，避免半成品 status；dispatcher 通过 `status` subresource 做 JSON merge patch（或 server-side apply）+ `resourceVersion` 冲突重试；`conditions[]` 由 dispatcher 按 `type` 去重后整体写回（CRD 不依赖 strategic merge patch 的 merge key 语义）。
- **MapStatus 纯函数**（仅 MLJob/MLService dispatcher+handler 模式）：状态推进只依赖 dispatcher 传入的快照，不发起 K8s 调用，便于单元测试与状态回放。
- **Handler 不直接写 `status`**（仅 MLJob/MLService）：Handler 通过 `MapStatus` 的返回值与 `Reconcile` 的结构化结果影响 `status`；dispatcher 统一合并写盘。

### 3.4 Pod 注入约定（MLJob + MLService）

所有 MLJob 与 MLService Handler 派生的 Pod（含 KServe 派生的 inference Pod）必须满足以下注入约定，体现 [infra.md §8.3](infra.md) 的 Quota 全覆盖不变式：

| Pod 字段 / Label | 必填 | 取值 | 用途 |
| --- | --- | --- | --- |
| `spec.schedulerName` | 是 | `koord-scheduler` | 所有 AxisML workload Pod 一律走 koord-scheduler；不允许任何 backend 让 Pod 落到默认 kube-scheduler 上 |
| label `quota.scheduling.koordinator.sh/name` | 是 | `<spec.scheduling.quota>` | Koordinator 原生 quota 关联 label；ElasticQuota plugin 据此把该 Pod 计入 `status.used` |
| label `axisml.io/{job-id\|service-id}` | 是 | UUID | 反查 MLJob / MLService，与 CR 上同名 label 一致 |
| label `axisml.io/role` | 是 | role 名（`worker` / `master` / `predictor` / ...） | 区分多角色拓扑下的 Pod |
| label `axisml.io/quota` | 是 | Compute Quota bare name（如 `training`） | AxisML 自有审计 / 查询；**与 `quota.scheduling.koordinator.sh/name` 取值不同**：前者裸名，后者 ElasticQuota 全名 `axisml-<tenant>-<pool>-<quota>` |
| label `axisml.io/replica-index` | 否 | role 内 0-based 序号 | 副本身份天然稳定时建议透传：StatefulSet 的 `apps.kubernetes.io/pod-index`、Indexed Job 的 `batch.kubernetes.io/job-completion-index`；Deployment / 裸 Pod / KServe autoscaling 等无稳定身份场景一律省略 |

前 5 项必填，所有 Handler 一律遵守；缺失任一项视为契约违反，Handler 的 `Validate` 必须在创建前拦截。`replica-index` 是可观测增强，缺失时 Compute §7.4 日志 API 退化为按 pod 名定位。

**KServe 派生 Pod 的注入路径**：`(kserve, *)` Handler 不直接控制 podSpec，通过写入 `InferenceService.spec.predictor.schedulerName` + `spec.predictor.labels` 让 KServe 透传到派生 Pod 的 `spec.schedulerName` 与 `metadata.labels`（KServe `PredictorSpec` 内联 `corev1.PodSpec` 与 `ComponentExtensionSpec`，所以两者都是 `spec.predictor` 的直接字段）。这要求 KServe 版本支持透传——前置依赖见 §6.6.3。

### 3.5 Dispatcher + Handler 架构（MLJob + MLService）

MLJob 与 MLService 同构使用两层结构：

- **Dispatcher Reconciler**：watch 所有 MLJob / MLService CR，按 `spec.backend.{name, engine}` 元组路由到对应 Handler；本身不直接生成底层资源。
- **Handler 注册表**：每个 `(backend, engine)` 元组对应一个 Handler 实例；编译期通过 `init()` 注册到全局 registry。Handler 负责把通用字段 + `backend.config` 翻译成具体的底层资源（Pod / PodGroup / Deployment / 第三方 CR），并把后端原生状态映射回 CR 统一 phase。

```
                 ┌────────────────────────────────┐
   <CR>     ───▶ │  Dispatcher Reconciler         │
                 │  (按 (backend, engine) 路由)    │
                 └─────────────┬──────────────────┘
                               │
       ┌───────────────────────┼───────────────────────┐
       ▼                       ▼                       ▼
   Handler A             Handler B               Handler N
       │                       │                       │
       └───────────────────────┴───────────────────────┘
                               │
                       status backflow
                               │
                               ▼
                      <CR>.status (统一 phase)
```

**Watch 拓扑**：Dispatcher 始终 watch 主 CR 队列；每个 Handler 启动时通过 `WatchTargets()` 声明自己关心的底层资源 GVK（Pod、PodGroup、PyTorchJob、Deployment、HTTPRoute、InferenceService …），由 dispatcher 统一建立 watch（controller-runtime `Watches()`），事件通过 `ownerReference` 反查回 CR 后再交给对应 Handler 的 `MapStatus`。

**注册方式**：编译期注册（每个 handler 包 `init()` → 全局 registry）；不引入运行时插件加载（plugin / wasm / 外部 grpc）。若未来需要"运行时安装新后端"，再演进为独立 operator binary 路由模式。

**未注册组合的兜底**：dispatcher 收到一条 `(backend, engine)` 无 handler 的 CR → 写 `status.phase=Failed`、`status.message="no handler for backend=X engine=Y"`，不创建任何底层资源。

### 3.6 Handler 接口契约（MLJob + MLService）

所有 Handler 必须实现以下方法（语言无关，描述责任）：

| 方法 | 责任 |
| --- | --- |
| `Key()` | 返回 `(backend, engine)` 元组，与 `spec.backend.{name, engine}` 对齐；用作注册表的主键 |
| `Validate(spec)` | 校验通用字段 + `backend.config` + role 集合（数量、命名）；纯函数，只返回 errors / warnings，不写 `status` |
| `Reconcile(ctx, cr)` | 创建 / 更新底层资源；幂等；通用字段由 Handler 自己注入到对应位置；处理控制信号（见下方"控制信号义务"）；返回结构化结果，不直接写 `status` |
| `MapStatus(snapshot)` | 把 CR spec + 底层资源快照映射回统一 phase + 公共状态字段；纯函数，不发起 K8s API 调用 |
| `Cleanup(ctx, cr)` | 删除底层资源。一般依赖 ownerReference 自动级联，Handler 仅负责需要主动清理的副本（跨 namespace 资源、外部存储句柄） |
| `WatchTargets()` | 声明本 Handler 需要 watch 的底层资源 GVK 列表，dispatcher 启动时统一建立 watch |
| `RequiredRBAC()` | 声明本 Handler 需要的 ClusterRole 规则；启动时聚合到 operator ServiceAccount |

#### 控制信号义务

**MLJob Suspend**：每个 MLJob Handler 在自身章节显式声明"原生支持 / 兜底为 Cleanup"，dispatcher 不做静默选择——不支持原生 suspend 时必须显式调用 `Cleanup()`，避免半暂停半运行的中间态。所有路径完成底层动作后都必须返回 `suspendCompleted=true, reason=CancelRequested`——这是 dispatcher 写入 §5.3 cancel 闭环推进信号的唯一来源；缺失会导致 Compute PG 永远卡在 `Canceling`。`status.phase` 在非终态 suspend 期间维持 `Pending`。若底层资源已经终态，终态优先，Handler 返回终态状态映射而不是 suspend 完成结果。

**MLService Scale**：每个 MLService Handler 必须把 `roles[*].replicas` 透传为后端原生扩缩（`Deployment.spec.replicas` / `StatefulSet.spec.replicas` / `InferenceService.spec.predictor.{minReplicas, maxReplicas}`）；不支持原生扩缩的 backend 兜底为重建底层资源（应避免，作为最后手段）。

### 3.7 不变量

仅列出独立于前面小节的硬约束；与 §3.2 / §3.4 / §3.6 重复的项目（phase 集合冻结、Pod 注入 5 项必填字段、`spec.backend.{name, engine}` 不可变）已在原章节声明，此处不再罗列。

- `(backend, engine)` 元组未在 registry 注册 → CR 直接进入 `Failed`，message 写明缺失原因。
- Handler 不直接修改 ElasticQuota CR；ElasticQuota 由 Tenant controller 独占维护（spec 写 + status.used 读）。
- operator 不向 Compute PG 写入任何数据；状态全部经由 CR `status` + Compute Informer 回流。

---

## Part III — Controller 详细设计

> 本部分逐个展开三个 controller 的 CRD 字段、状态推导、Handler 落地等独有内容。

## 4. Tenant Controller

### 4.1 概述

Tenant controller 把 Compute 下发的 `Tenant` CR 翻译为 Kubernetes 侧的命名空间、租户配额与租户级初始化资源，并把执行状态回流到 `Tenant.status`。它承载三类职责：

1. **Namespace 落地**：按 `spec.namespace.name` 创建并维护租户使用的 Namespace；同一 Namespace 允许被多个 Tenant CR 共享（详见 §4.6）。
2. **ElasticQuota 派生**：把 `spec.quotas[]` 渲染为 Koordinator `ElasticQuota` CR（每 `(tenant, pool, quota)` 一条，落在租户 Namespace 下），并把 `status.used` 回流到 `Tenant.status.quotas[].used`——这是 Tenant CR 与 Compute / koord-scheduler 之间双向数据链路的承载。
3. **初始化资源下发**：按 `spec.initResources` 创建租户私有的 ImagePullSecrets / 通用 Secret / ConfigMap / ServiceAccount + RBAC。

Tenant 是 [compute.md §5.4](compute.md) 中的**配置对象**——CR 缺失/漂移会被 Compute 按 PG 快照补偿重建，因此 `Reconcile` 必须可重复执行。Tenant controller **不存在多后端实现**，无 dispatcher/handler 分层；所有 Tenant CR 由单一 controller 处理。

### 4.2 与 Compute 的额外契约

除了 §3.1 列出的通用契约（Create 幂等、status 单向权威、配置补偿），Tenant 还有两条特有约束：

- **Namespace 永不级联删除**：Tenant 删除时仅依赖 ownerReference 让 K8s GC 清理 per-tenant 资源（ElasticQuota / Secret / ConfigMap / SA / Role / RoleBinding）；Namespace 自身不被删除（详见 §4.6.1 与 §3.7）。
- **共享 Namespace 友好**：同一 Namespace 可被多个 Tenant CR 引用；per-tenant 资源通过命名前缀 + label 实现隔离，不依赖独占 Namespace（详见 §4.7）。

### 4.3 CRD 契约

Tenant 为 cluster-scoped CR：

| 字段 | 取值 |
| --- | --- |
| `apiVersion` | `axisml.io/v1alpha1` |
| `kind` | `Tenant` |
| `scope` | `Cluster` |
| `shortNames` | `tnt` |

#### 4.3.1 spec 设计取舍

把 Tenant 的三类职责建模为同级字段——`namespace`（命名空间引用）、`quotas`（租户配额数组，1:1 渲染为 ElasticQuota CR）、`initResources`（初始化清单），避免任何一类职责喧宾夺主。

**为何把 quotas 内联到 Tenant.spec 而非独立 CR**：Quota 在概念上是 Tenant 的子资源（每条配额都依附于一个 `(tenant, pool)` 组合），生命周期与 Tenant 强绑定。把 quotas 内联到 Tenant CR 让 Tenant controller 成为 ElasticQuota 的 single writer，给 Compute 提供统一的双向数据链路：`spec.quotas[]` 下行表达 desired `min` / `max`，`status.quotas[].used` 上行回流实际用量。Compute 侧仍保留独立的 `quotas` PG 表（[compute.md §6.2.4](compute.md)）以承载 API 行级 CRUD 与跨租户查询；CR 端只是该表的渲染。

**为何 quotas 用数组而非 map**：每条 quota 的标识由 `(pool, name)` 元组确定，map 在 spec 里只能用字符串单 key，会丢失结构。

**为何不在 Tenant 上保留 K8s `ResourceQuota` 兜底字段**：K8s `ResourceQuota` 按 Namespace 聚合计量，不会按 Tenant CR、ServiceAccount 或 `axisml.io/tenant-id` label 自动拆分用量。共享 Namespace 下不能表达 per-tenant 额度；独占 Namespace 下又与 ElasticQuota `max` 形成两套上限，徒增复杂度。租户级容量边界统一收敛到 ElasticQuota（`min` / `max` + Pod label `quota.scheduling.koordinator.sh/name`）。

**为何把 namespace 名放在 `spec.namespace.name` 而非 `metadata.namespace`**：Tenant 是 cluster-scoped CR，不属于任何 Namespace；同时多个 Tenant 可共享同一个 Namespace，把 namespace 作为引用而非容器是更自然的建模。

**为何 per-tenant 资源命名统一加 `axisml-tenant-<tenant-name>-` 前缀**：共享 Namespace 场景下，多个 Tenant 在同一 Namespace 内创建同名 ImagePullSecret / ServiceAccount 会 collide。命名前缀一致化避免 collide，也让 selector 检索（"找出该 Namespace 下属于 tenant X 的所有资源"）有稳定锚点。

**长度上限**：`metadata.name` 限制为 ≤40 字符；`axisml-tenant-` 前缀 14 字符 + tenant-name 40 + 分隔符 1 = 55 字符。`spec.initResources.*[].name` 与 `serviceAccounts[].name` 的理论上限因此为 `253 - 55 = 198` 字符（DNS-1123 subdomain 总长 253）。

**为何初始化资源都从 `sourceXxxRef` 复制而非内联数据**：避免敏感数据（dockerconfigjson、对象存储凭证）以明文形式写入 Tenant CR。源 Secret / ConfigMap 由集群管理员预先放在受控 Namespace（如 `axisml-system`），controller 用 reader 权限读出再写入租户 Namespace。详见 §4.6.3–§4.6.5。

**为何 `runPolicy` 字段缺席**：Tenant 不是 workload——没有 `activeDeadlineSeconds` / `progressDeadlineSeconds` / `backoffLimit` 等概念。生命周期控制只有 `suspended` 一个开关。

#### 4.3.2 spec 结构

```yaml
apiVersion: axisml.io/v1alpha1
kind: Tenant
metadata:
  name: <tenant-name>
  labels:
    axisml.io/tenant-id: <uuid>
spec:
  # ── 元数据 ──────────────────────────────────────────────────────
  displayName: string
  annotations: {}                # 透传到 Tenant 自身的 annotations 区

  # ── 命名空间引用（多 Tenant 可指向同一 namespace）────────────────
  namespace:
    name: string                 # 必填: DNS-1123；创建后不可变
    labels: {}                   # 可选: 首次创建 Namespace 时附加；已存在则忽略
    annotations: {}              # 可选: 同上

  # ── 配额（数组，每项 1:1 渲染为 ElasticQuota CR）─────────────────
  # 可选: 整段缺省或空数组 → 租户在 K8s 层不受 ElasticQuota 约束
  # 共享 Namespace 下允许；ElasticQuota 通过 Pod label 按 quota name 跨 namespace 绑定
  quotas:
    - pool: string               # 必填: ResourcePool 名（与 Compute resource_pools.name 对齐）
      name: string               # 必填: 配额名；(pool, name) 创建后不可变
      min: {}                    # 可选: 资源 map，缺省 {} 等同各 key 取 0；ElasticQuota.spec.min 直传
      max: {}                    # 必填: 资源 map；ElasticQuota.spec.max 直传，硬上限

  # ── 初始化资源 ─────────────────────────────────────────────────
  initResources:
    imagePullSecrets:
      - name: string             # 用户可见名；最终 Secret 名 = axisml-tenant-<tenant>-<name>
        sourceSecretRef:         # 从受控 namespace 已有 Secret 复制（避免明文进入 spec）
          namespace: string
          name: string

    secrets:
      - name: string
        type: Opaque             # 默认 Opaque；允许 dockerconfigjson / tls 等
        sourceSecretRef:
          namespace: string
          name: string

    configMaps:
      - name: string
        sourceConfigMapRef:
          namespace: string
          name: string

    serviceAccounts:
      - name: string
        imagePullSecrets:        # 可选: 引用 spec.initResources.imagePullSecrets[].name
          - string
        rbac:                    # 可选: 同步创建 Role + RoleBinding
          rules: []
          roleRef:               # 可选: 改为绑定到已存在的 ClusterRole 而非新建 Role
            kind: Role | ClusterRole
            name: string

  # ── 控制 ───────────────────────────────────────────────────────
  suspended: false               # cancel 信号；Operator 标记并写 status.phase=Suspended
```

#### 4.3.3 字段归属与不可变性

| 字段路径 | 写入方 | 创建后可变？ |
| --- | --- | --- |
| `metadata.name` / `labels[axisml.io/tenant-id]` | Compute | 否 |
| `spec.displayName` / `annotations` | Compute（透传用户输入） | 是 |
| `spec.namespace.name` | Compute | **否**；controller 行为兜底（拒绝并写 `status.message`），admission webhook 为最终兜底 |
| `spec.namespace.labels` / `annotations` | Compute | 是；只在 Namespace **首次创建** 时落地，已存在的 Namespace 不被覆盖（避免污染共享 Namespace） |
| `spec.quotas[].{pool, name}` | Compute | **否**（每项标识锚点）；删除某项 → reconcile `Delete()` 对应 ElasticQuota CR；新增某项 → reconcile 创建 |
| `spec.quotas[].min` / `max` | Compute | 是；reconcile 同步覆盖到对应 ElasticQuota `spec.min` / `spec.max` |
| `spec.initResources.*` | Compute | 是；增删 → reconcile 创建 / 删除对应资源 |
| `spec.suspended` | API（`/suspend` / `/unsuspend` 触发） | 是 |

**默认值注入**：`spec.suspended` 默认 `false`；`spec.initResources` 各列表默认 `[]`；`spec.quotas` 默认 `[]`（视为租户在 K8s 调度层不限额）；`spec.quotas[].min` 默认 `{}`；`spec.initResources.secrets[].type` 默认 `Opaque`。

### 4.4 Status

```yaml
status:
  observedGeneration: int64
  phase: Active | Suspended | Failed   # ← Compute 消费字段（驱动 tenants.status）
  message: string                # 错误或状态附加信息（Compute 透传到 tenants.message）
  namespaceReady: bool
  quotas:                        # 每条 quota 的就绪与用量回流（Compute 消费 used → quotas.used 缓存）
    - pool: string
      name: string
      ready: bool                # ElasticQuota CR 已就绪（spec 已 apply、status.used 已被 koord-scheduler 填充）
      used: {}                   # 资源 map，来自 ElasticQuota.status.used
      message: string
  initResources:                 # 各类初始化资源逐项 ready 状态（UI 可观测，Compute 不消费）
    imagePullSecrets: [{ name, ready, message }]
    secrets:          [{ name, ready, message }]
    configMaps:       [{ name, ready, message }]
    serviceAccounts:  [{ name, ready, message }]
  conditions:                    # K8s 标准 conditions（UI 可观测，Compute 不消费）
    - type: NamespaceReady | QuotasReady | InitResourcesReady | Suspended | Failed
      status: True | False | Unknown
      ...
```

**Compute phase 映射规则**（与 [compute.md §6.2.1](compute.md) 对齐）：

| Tenant status.phase | tenants.status |
| --- | --- |
| `Active` | `Active` |
| `Suspended` | `Suspended` |
| `Failed` | `Suspended`，并写入 `message`（保持 Tenant 不可提交新 Job/Service，由人工排查） |

`Failed` 收敛为 `Suspended` 让"配置出错"与"管理员暂停"在 Compute 侧表现一致——租户提交链路同样受阻，靠 `message` 区分原因。

**phase 推导规则**（reconcile 末尾计算）：

| 条件 | phase |
| --- | --- |
| `spec.suspended == true` | `Suspended` |
| `namespaceReady && 所有 quotas[*].ready == true && 所有 initResources[*].ready == true` | `Active` |
| 任一关键资源（Namespace / ElasticQuota）创建失败且非短暂瞬态 | `Failed` |
| 否则（瞬态创建过程中） | 维持上一态，`message` 写当前进展 |

`spec.quotas` 为空数组时 `status.quotas` 同为空，`Active` 推导只看 `namespaceReady` 与 initResources。

**`status.quotas[].used` 回流路径**：controller 通过 SharedInformerFactory watch 本集群所有 namespace 的 ElasticQuota CR，按 ownerReference 反查所属 Tenant，把 `ElasticQuota.status.used` 聚合到对应 `Tenant.status.quotas[i].used`。Compute Tenant Informer 消费该字段更新 PG `quotas.used` 缓存（详见 [compute.md §5.3 / §6.2.4](compute.md)）。

### 4.5 Reconcile 生命周期

按事件源切分 controller 职责（通用约束见 §3.3）：

| 事件 | Controller 行为 |
| --- | --- |
| Tenant ADD（首次创建） | `Validate(spec)` 校验失败写 `status.phase=Failed`；通过则按 §4.6 顺序确保底层资源就位（Namespace → ElasticQuota → initResources），最后写 `status.phase=Active` |
| Tenant UPDATE（spec 变更） | 校验 `spec.namespace.name` 不变（违反则写 `status.message` 拒绝并维持原 phase）；其余 spec 变化按 §4.6 各小节"spec 漂移处理"覆盖底层资源 |
| Tenant UPDATE（`spec.suspended` 切换） | true → 写 `status.phase=Suspended`；false → 重新走 phase 推导。controller 不停机底层资源，只标记 phase；阻断新 Job/Service 提交由 Compute API 在 `tenant.status='Suspended'` 时拦截 |
| Tenant DELETE | 不阻断；K8s GC 通过 ownerReference 级联删除 per-tenant 资源；**Namespace 不删除**（§4.6.1） |
| 底层资源事件（ElasticQuota / Secret / ConfigMap / SA 等被外部修改或删除） | 按 ownerReference 反查到 Tenant，重新触发 Reconcile；漂移按各小节策略覆盖回 spec 快照。ElasticQuota 的 `status.used` 变更触发轻量 reconcile，仅刷新 `Tenant.status.quotas[i].used` |
| 周期 resync（默认 10 min，Helm values 可配） | 触发对所有 Tenant CR 的 reconcile，重读 `sourceSecretRef` / `sourceConfigMapRef` 源数据，按 §4.6.3–§4.6.5 漂移策略覆盖 per-tenant 副本——controller 不为源资源建立 watch，源更新最大延迟 = resync 间隔 |

源 Secret / ConfigMap 不建立 watch；其内容变更经"周期 resync"路径感知，最大延迟 = resync 间隔。需要更短延迟可由集群管理员 bump Tenant CR 的某个 annotation 显式触发 reconcile。

### 4.6 底层资源管理

每条 per-tenant 资源都叠加以下 label（便于在共享 Namespace 内 selector 过滤）：

- `axisml.io/tenant-id=<uuid>`
- `axisml.io/managed-by=tenant-operator`

每条 per-tenant 资源都通过 `ownerReferences` 指向 Tenant CR（cluster-scoped owner → namespaced dependent，K8s GC 原生支持）；Tenant 删除后由 K8s GC 异步清理。

#### 4.6.1 Namespace

| 维度 | 行为 |
| --- | --- |
| 命名 | `<spec.namespace.name>` |
| 创建 | Namespace 不存在 → 创建，附加 `spec.namespace.labels` / `annotations` 并叠加 `axisml.io/managed-by=tenant-operator` label |
| 已存在 | 仅补 `axisml.io/managed-by=tenant-operator` label（如缺失）；不覆盖任何其他既有 label / annotation，避免污染共享 Namespace。**风险**：`Namespace` 是 cluster-scoped 资源，K8s RBAC 不能按前缀或业务范围限制 `create`；controller 配置中必须维护目标 Namespace denylist / allowlist（默认拒绝 `kube-*`、`default`、`axisml-system` 等系统 Namespace），admission webhook 作为后续兜底（§4.8） |
| ownerReference | **不设置**——Namespace 不属于任何单一 Tenant |
| spec 漂移 | 不主动对账（Namespace 自身没有"由 Tenant 决定"的 spec 字段） |
| 删除 | **永不删除**——即使最后一个引用本 Namespace 的 Tenant 被删除，Namespace 也保留。空 Namespace 由集群管理员手工清理 |

**为何不删 Namespace**：Namespace 中可能存在 Tenant 不可见的 PV、外部 controller 创建的资源、用户手工创建的 ConfigMap。误删 Namespace 会引发不可逆的状态丢失。把"清理空 Namespace"作为运维操作而非 operator 的自动行为是更安全的取舍。

`status.namespaceReady` 在 Namespace `phase=Active` 时为 `true`。

#### 4.6.2 ElasticQuota

`spec.quotas[]` 每项 1:1 渲染为一条 Koordinator `ElasticQuota` CR（`scheduling.sigs.k8s.io/v1alpha1`，namespace-scoped）。CR 落在 `spec.namespace.name` 下；Pod 通过 label `quota.scheduling.koordinator.sh/name=<eq-name>` 按集群唯一名跨 namespace 绑定 quota（详见 [compute.md §6.2.4](compute.md)），所以共享 Namespace 与独占 Namespace 在 quota 隔离上没有差别——命名前缀已天然 per-tenant 隔离。

| 维度 | 行为 |
| --- | --- |
| 命名 | `axisml-<tenant-name>-<pool>-<quota-name>`（集群内唯一）|
| 创建 | 在 `<spec.namespace.name>` 内创建 ElasticQuota，`spec.min` / `spec.max` 直接来自 `spec.quotas[i].min` / `max` |
| 缺省 | `spec.quotas` 为空数组 → controller 不创建任何 ElasticQuota；增删项分别触发 Create / Delete |
| ownerReference | Tenant CR |
| spec 漂移 | reconcile 检测到 `ElasticQuota.spec.{min, max}` 与 `spec.quotas[i].{min, max}` 不一致时按 spec 覆盖 |
| status.used 回流 | watch ElasticQuota，把 `status.used` 写入 `Tenant.status.quotas[i].used`；不写回 ElasticQuota |
| 删除 | 随 Tenant 删除 K8s GC；spec 中删除某项 → reconcile 显式 Delete 对应 ElasticQuota CR |

**配额校验**：`Validate(spec)` 对每条 quota 校验 `min[k] ≤ max[k]` 且均 ≥ 0；`(pool, name)` 在 `spec.quotas[]` 内唯一。

#### 4.6.3 ImagePullSecrets

| 维度 | 行为 |
| --- | --- |
| 命名 | `axisml-tenant-<tenant-name>-<spec-name>` |
| 类型 | `kubernetes.io/dockerconfigjson` |
| 数据来源 | `sourceSecretRef.{namespace, name}` 指向受控 Namespace（如 `axisml-system`）中预先创建好的 Secret；controller 用 reader 权限 `Get()` 后把 `data` 写入新 Secret |
| ownerReference | Tenant CR |
| spec 漂移 | reconcile 检测到本端 Secret 数据与源 Secret 不一致时覆盖；源 Secret 不存在时把对应 `status.initResources.imagePullSecrets[i].ready=false` 并写 message |
| 删除 | 随 Tenant 删除 GC；spec 中删除某项 → reconcile 显式 Delete |

#### 4.6.4 通用 Secret

| 维度 | 行为 |
| --- | --- |
| 命名 | `axisml-tenant-<tenant-name>-<spec-name>` |
| 类型 | 取 `spec.initResources.secrets[].type`，默认 `Opaque`；允许 `dockerconfigjson` / `kubernetes.io/tls` 等任意 K8s Secret type |
| 类型不一致 | `spec.type` 与源 Secret 的 `type` 不一致时以 spec 为准并写警告；若结构性约束失败（如 `dockerconfigjson` / `tls` 缺失关键 key） → 该项 `ready=false`、message 指明缺失字段。Secret type 在 K8s 中不可变，运行时 `spec.type` 改动 → reconcile 删除现有 Secret 重建 |
| 数据来源 / 漂移 / 删除 | 同 §4.6.3 |

#### 4.6.5 ConfigMap

| 维度 | 行为 |
| --- | --- |
| 命名 | `axisml-tenant-<tenant-name>-<spec-name>` |
| 数据来源 | `sourceConfigMapRef.{namespace, name}` |
| ownerReference / 漂移 / 删除 | 同 §4.6.3 |

#### 4.6.6 ServiceAccount + RBAC

| 维度 | 行为 |
| --- | --- |
| ServiceAccount 命名 | `axisml-tenant-<tenant-name>-<spec-name>` |
| `imagePullSecrets` | 把 `spec.initResources.serviceAccounts[].imagePullSecrets`（按用户可见 name）解析为最终 Secret 名后填到 SA `imagePullSecrets[]` |
| **引用校验** | `Validate(spec)` 检查 `serviceAccounts[].imagePullSecrets[]` 中每个 name 必须能在 `spec.initResources.imagePullSecrets[].name` 中找到；找不到 → 校验失败 → `status.phase=Failed` |
| Role 命名 | `axisml-tenant-<tenant-name>-<sa-name>`（仅当声明 `rbac.rules` 且未指定 `rbac.roleRef.kind=ClusterRole`） |
| RoleBinding 命名 | `axisml-tenant-<tenant-name>-<sa-name>`（声明 `rbac` 即创建） |
| RoleBinding 绑定关系 | `subjects` 指向本 SA；`roleRef` 指向同名 Role（默认）或 `spec.rbac.roleRef`（指定时） |
| 漂移 / 删除 | reconcile 检测漂移时覆盖；spec 中删除某条 SA → reconcile 显式 Delete 对应 SA + 关联 Role + RoleBinding |

**RBAC 的两种使用形态**：

1. `rbac.rules` 非空、未指定 `roleRef.kind=ClusterRole` → controller 创建一个独立 Role 持有 `rules`，并创建 RoleBinding 绑定本 SA。
2. `rbac.roleRef.kind=ClusterRole` → controller 不创建 Role，只创建 RoleBinding 把本 SA 绑定到指定 ClusterRole（适用于"复用平台预置 ClusterRole"场景）。

### 4.7 多 Tenant 共享 Namespace 语义

`spec.namespace.name` 允许多个 Tenant CR 指向同一 Namespace；典型场景是多个轻量级团队共享一个开发 / 沙箱环境。共享时的关键不变量：

- **Namespace 自身不绑定 ownerReference**——Namespace 是共享资源，不属于任一 Tenant。
- **per-tenant 资源命名前缀**：派生的 per-tenant 资源（Secret / ConfigMap / SA / Role / RoleBinding）用 `axisml-tenant-<tenant-name>-` 前缀；ElasticQuota 用 `axisml-<tenant-name>-<pool>-<quota>` 前缀。两套前缀都集群唯一，共享 Namespace 内不会 collide。
- **per-tenant 资源 label `axisml.io/tenant-id=<uuid>`** 提供 selector 检索能力。
- **per-tenant ElasticQuota**：每个 Tenant 在共享 Namespace 内仍各自持有独立 ElasticQuota CR；Pod 按 label `quota.scheduling.koordinator.sh/name` 跨 namespace 绑定。
- **Pod 通过 ServiceAccount 关联 tenant**：选择 `axisml-tenant-<tenant>-<sa>` SA → 自动获得本 tenant 的 imagePullSecrets / RBAC。
- **Tenant A 删除不影响 Tenant B**：A 的 per-tenant 资源被 K8s GC 清理 → B 不受影响 → Namespace 保留。

#### 共享场景示意

假设 Namespace `shared-dev` 同时托管 Tenant `team-a` 与 `team-b`：

```
Namespace shared-dev
├── Secret         axisml-tenant-team-a-registry        (owner: Tenant team-a)
├── Secret         axisml-tenant-team-b-registry        (owner: Tenant team-b)
├── ServiceAccount axisml-tenant-team-a-default          (owner: Tenant team-a)
├── ServiceAccount axisml-tenant-team-b-default          (owner: Tenant team-b)
├── ElasticQuota   axisml-team-a-default-default         (owner: Tenant team-a)
├── ElasticQuota   axisml-team-b-default-training        (owner: Tenant team-b)
├── ...
```

team-a 的 Pod 只能选择 `axisml-tenant-team-a-*` SA，从而只看到本 tenant 的 imagePullSecrets / RBAC；同时通过 label `quota.scheduling.koordinator.sh/name=axisml-team-a-default-default` 把用量计入 team-a 的 ElasticQuota，与 team-b 完全独立。

### 4.8 Tenant RBAC

| 资源 | 动作 | 用途 |
| --- | --- | --- |
| `tenants.axisml.io` | `get / list / watch / patch` | watch Tenant CR、写 status |
| `namespaces` | `create / get / list / watch / update / patch` | 创建并对齐 Namespace metadata；**不含 `delete`** |
| `elasticquotas.scheduling.sigs.k8s.io` | `create / get / list / watch / update / patch / delete` | 派生并维护 per-tenant ElasticQuota |
| `secrets` | `create / get / list / watch / update / patch / delete`（目标 Namespace）；`get`（源 Namespace） | 维护 per-tenant Secret |
| `configmaps` | 同 secrets | 维护 per-tenant ConfigMap |
| `serviceaccounts` | `create / get / list / watch / update / patch / delete` | 维护 per-tenant ServiceAccount |
| `roles.rbac.authorization.k8s.io` | `create / get / list / watch / update / patch / delete` | 维护 per-tenant Role |
| `rolebindings.rbac.authorization.k8s.io` | `create / get / list / watch / update / patch / delete` | 维护 per-tenant RoleBinding |
| `events` | `create / patch` | 写 K8s Event |

**`namespaces` 不含 `delete`** 是显式的最小权限策略，对应 §4.6.1 的"永不删除 Namespace"行为约束。

### 4.9 后续工作

- Admission webhook：`spec.namespace.name` / `spec.quotas[].{pool, name}` 不可变约束、`spec.initResources.*.sourceXxxRef` 跨 Namespace 读权限白名单、`spec.quotas[].{min, max}` 结构性校验。
- **目标 Namespace 白名单**：当前通过 controller Helm values 配置 denylist / allowlist；后续由 admission webhook 前移到准入阶段。
- **源资源结构性校验前移**：admission webhook 在 Tenant 创建/更新时校验源 Secret 的 type 与 spec 一致。
- **resync 间隔的 Helm values 暴露**：默认 10 min，运维可下调到分钟级换取更短的源资源更新延迟。
- 加密源支持：从 KMS / Vault / Sealed Secrets 拉取凭证作为 `sourceSecretRef` 替代方案。
- `spec.initResources` templating：按 tenant 上下文（id / name / namespace）渲染 ConfigMap 数据。
- 跨 Namespace 复制源的 RBAC 收敛：把源 Namespace 限定为单一受控 Namespace。
- ServiceAccount + RBAC 子能力的 Helm values 开关。
- 分层配额：在 `spec.quotas[]` 引入 `parent` 字段，落到 ElasticQuota 的 `quota.scheduling.koordinator.sh/parent` annotation。
- CRD 严格 schema（启用 OpenAPI 校验，替换当前的 `x-kubernetes-preserve-unknown-fields`）。

## 5. MLJob Controller

### 5.1 概述

MLJob controller 把 Compute 下发的 `MLJob` CR 翻译为底层执行资源（Pod / PodGroup / 第三方 CR），并把执行状态回流到 `MLJob.status`。它使用 §3.5 的 dispatcher + handler 模式：CR 的 `spec.backend.{name, engine}` 二级元组路由到不同 Handler，由 Handler 真正生成底层资源、把后端原生状态映射回统一的 phase 集合。

`MLJob` 为 namespaced CR：

| 字段 | 取值 |
| --- | --- |
| `apiVersion` | `axisml.io/v1alpha1` |
| `kind` | `MLJob` |
| `scope` | `Namespaced`（创建在租户 namespace 下） |
| `shortNames` | `mlj` |

### 5.2 CRD 契约

#### 5.2.1 spec 设计取舍

把"角色拓扑"提升为一等公民。Job 的执行域用 `roles[]` 数组承载：

- 单角色 backend（如 `(native, job)`、`(native, podgroup)`）声明一个 role。
- 多角色 backend（如 PyTorchJob 的 master/worker、TFJob 的 chief/worker/ps/evaluator、MPIJob 的 launcher/worker）声明多个 role。
- role 名集合由各 Handler 在 §5.5 中约定，由 Handler 的 `Validate` 强制。

替代方案是把 `image / command / replicas / resources` 全部摆在 spec 顶层，对单角色自然，但多角色 backend 不得不把角色切分挤进 `backend.config`，让 generic 字段失去意义。引入 `roles[]` 后，单角色场景退化为"只有一个 role 的特例"。

调度域的 `nodeSelector` / `tolerations` 沿用 K8s PodSpec 扁平惯例直接放在 `spec.scheduling` 下，不再额外包一层 `placement`。

#### 5.2.2 spec 结构

```yaml
apiVersion: axisml.io/v1alpha1
kind: MLJob
spec:
  # ── 后端选择（创建后不可变；dispatcher 路由依据）─────────────────────
  backend:
    name: native              # 必填: native | kubeflow-trainer | custom
    engine: job               # 必填: 语义随 backend 而定（见 §5.5）
                              #   native:           job | podgroup
                              #   kubeflow-trainer: pytorchjob | tfjob | mpijob | …
                              #   custom:           任意名（由 backend.config 描述目标 GVK）
    config: {}                # 可选: 该 (backend, engine) 元组特有的 schemaless 配置

  # ── 调度域（由 Compute 从 Quota / ResourcePool / ResourceUnit 合成注入）──
  scheduling:
    quota: string             # 必填: ElasticQuota CR 名（axisml-<tenant>-<pool>-<quota>）
    priorityClass: string     # 可选
    nodeSelector: {}          # Compute 按 compute.md §6.2.3 合并 pool + unit 后注入
    tolerations: []           # 来自 ResourcePool

  # ── 执行域：roles 数组承载角色拓扑 ─────────────────────────────────
  roles:
    - name: worker            # role 标识；同一 MLJob 内唯一
      replicas: 1             # >= 0；为 0 时该角色禁用（如 TFJob 的 evaluator）
      restartPolicy: OnFailure # OnFailure | Never
      template:               # Pod template 子集：暴露常用字段，隐藏完整 PodSpec
        image: string
        imagePullPolicy: IfNotPresent
        command: []
        args: []
        env: []
        envFrom: []
        workingDir: string
        resources:
          requests: {}         # Compute 从 ResourceUnit.requests 注入
          limits: {}

  # ── 生命周期 ────────────────────────────────────────────────────
  runPolicy:
    suspend: false                  # 可选: cancel 信号；Handler 暂停或清理底层资源
    activeDeadlineSeconds: int      # 可选: 硬超时；超时后 Handler 推 Failed
    ttlSecondsAfterFinished: int    # 可选: 终态后底层资源 GC
    backoffLimit: int               # 可选: 重试预算；具体语义由各 Handler 解释
```

#### 5.2.3 字段归属与不可变性

| 字段路径 | 写入方 | 创建后可变？ |
| --- | --- | --- |
| `metadata.name` / `namespace` / `labels[axisml.io/*]` | Compute | 否 |
| `spec.backend.name` / `spec.backend.engine` | Compute（默认 `{native, job}`） | **否**；dispatcher 拒绝并写 `status.message` |
| `spec.backend.config` | Compute（默认 `{}`） | 视 Handler 语义；Handler 在 `Validate` 中决定 |
| `spec.scheduling.quota` / `priorityClass` / `nodeSelector` / `tolerations` | Compute | 否 |
| `spec.roles[*].template.resources` | Compute（注入 ResourceUnit） | 否 |
| `spec.roles[*].replicas` | 用户提交时给定 | **否**（Job 是一次性 workload；扩缩容是 Service 专属） |
| `spec.runPolicy.suspend` | API（`/cancel` 触发） | **是**（cancel 路径专用） |
| 其他 `spec.runPolicy.*` 与 `spec.roles[*].template.*`（除 resources） | 用户提交 | 否 |

**默认值**：`spec.backend` 默认 `{name: native, engine: job}`（K8s 原生 Job + koord-scheduler，详见 §5.5.1）；`backend.config` 默认 `{}`。

### 5.3 Status

```yaml
status:
  observedGeneration: int64
  phase: Pending | Running | Succeeded | Failed   # ← Compute 唯一消费的字段
  message: string                                  # Compute 透传到 jobs.message
  startedAt: timestamp                             # Compute 写入 jobs.started_at
  finishedAt: timestamp                            # Compute 写入 jobs.finished_at
  conditions:                                      # Suspended 会被 Compute 消费为 cancel 推进信号；其余仅 UI 观测
    - type: Initialized | Scheduled | Suspended | Failed
      status: True | False | Unknown
      lastTransitionTime: timestamp
      reason: string                               # Suspended 时约定 reason=CancelRequested
      message: string
  roles:                                           # 各 role 副本聚合（UI 可观测，Compute 不消费）
    - name: string
      replicas: int          # spec 期望
      activeReplicas: int    # 运行中
      readyReplicas: int     # 通过 readiness probe
      succeededReplicas: int
      failedReplicas: int
```

**Compute phase 映射规则**（与 [compute.md §6.3.1](compute.md) 对齐）：

| MLJob status.phase | jobs.status | 终态 |
| --- | --- | --- |
| `Pending` | `Pending` | 否 |
| `Running` | `Running` | 否 |
| `Succeeded` | `Succeeded` | 是 |
| `Failed` | `Failed` | 是 |

**Cancel 推进信号**——`Cancelled` 与 `Deleted` 不由 operator 直接产出，但 cancel 路径有明确的链上信号：Handler 在收到 `spec.runPolicy.suspend=true` 并完成"暂停或清理底层资源"后，**必须向 dispatcher 返回 `suspendCompleted=true` 与 `reason=CancelRequested`**；dispatcher 统一合并写入 `status.conditions[type=Suspended,status=True,reason=CancelRequested]`，且在非终态时让 `status.phase` 维持在 `Pending`。Compute Informer 在 PG `status='Canceling'` 时把这个 condition 当作推进信号 → 写 `Cancelled` → 入队 `Delete()` 做 CR 资源回收（详见 [compute.md §5.2 / §5.3 / §6.3.1](compute.md)）。

**终态优先**：cancel 只面向仍处于 `Pending` / `Running` 的 Job。若底层资源已经进入 `Succeeded` / `Failed`，或同一轮 `MapStatus` 已经推导出终态，dispatcher 必须保留终态 phase 与 `finishedAt`，不能为了 cancel 信号把 `status.phase` 回退为 `Pending`；此时不写 `Suspended=True` 作为成功取消信号。

### 5.4 Reconcile 事件路径

Dispatcher 与 Handler 职责切分（通用约束见 §3.3 / §3.6）：

| 事件 | Dispatcher 行为 | Handler 行为 |
| --- | --- | --- |
| MLJob ADD | 路由到 Handler；调用 `Validate(spec)`，校验失败写 `status.phase=Failed` | `Reconcile` 创建底层资源，设置 `ownerReference: MLJob` |
| MLJob UPDATE（spec 变更） | 校验 `backend.{name, engine}` 不变（违反则写 `status.message` 拒绝）；其余 spec 变化路由给 Handler | `Reconcile` 幂等更新 |
| MLJob `spec.runPolicy.suspend=true` | 路由；若当前或新映射出的 phase 已是 `Succeeded` / `Failed`，终态优先；否则在 Handler 返回 suspend 完成结果后合并写入 `Suspended=True,reason=CancelRequested`，`phase` 维持 `Pending` | 执行原生 suspend（如 `(native, job)` patch `Job.spec.suspend=true`）或 `Cleanup()` 删除底层资源；返回结构化 suspend 结果 |
| MLJob DELETE | 不阻断 | 一般依赖 ownerReference 级联清理 |
| 底层资源事件（Pod / PodGroup / 第三方 CR） | 通过 ownerReference 反查到 MLJob 后路由 | `MapStatus` 纯函数计算新 phase |

### 5.5 内置 Handler

每个 Handler 章节统一按以下小节组织：**底层资源 / `backend.config` / 通用字段映射 / Status 映射 / Suspend / RBAC**。

#### 5.5.1 `(native, job)`

底层用 K8s 原生 [`Job`](https://kubernetes.io/docs/concepts/workloads/controllers/job/)；适合不需要 gang scheduling 的单角色批处理场景。Pod 强制走 koord-scheduler 并通过 `quota.scheduling.koordinator.sh/name` label 计入 ElasticQuota；本 Handler **不**创建 PodGroup（gang 不适用）。

**前置依赖**：集群已安装 Koordinator（提供 koord-scheduler 与 ElasticQuota plugin）。本 Handler 仅需要 `jobs.batch` 的 CRUD，不直接读写 ElasticQuota / PodGroup CR。

**底层资源**：

- 必填且仅一个 role（`name=worker`）；`Validate` 拒绝多 role 提交（多角色场景应选 `(kubeflow-trainer, *)`）。
- 每个 MLJob 创建一个 K8s `Job`，Pod 由 Job controller 派生，但 Pod 模板上设置 `schedulerName: koord-scheduler`。
- Job 设置 `ownerReference` 指向 MLJob，保证 MLJob 删除后底层资源级联清理。

**Pod label**：§3.4 列出的 5 项必填 label，外加 `axisml.io/replica-index=<0-based>`（**仅在 `backend.config.completionMode=Indexed` 时透传** K8s 注入的 `batch.kubernetes.io/job-completion-index`；默认 NonIndexed 模式下省略）。

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
| `roles[worker].template.resources` | Pod 主容器同名字段 |
| `roles[worker].replicas` | `Job.spec.parallelism` 与 `Job.spec.completions`（同值；Indexed 模式下 `completions` 表示总分片数） |
| `roles[worker].restartPolicy` | `Job.spec.template.spec.restartPolicy`（仅允许 `OnFailure` / `Never`） |
| `spec.scheduling.priorityClass` | Pod `spec.priorityClassName` |
| `spec.scheduling.nodeSelector` / `tolerations` | Pod 同名字段 |
| `spec.scheduling.quota` | Pod `spec.template.metadata.labels[quota.scheduling.koordinator.sh/name]`；不写入 Job 级别字段 |
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

`startedAt` 取 `Job.status.startTime`；`finishedAt` 取 `Job.status.completionTime`。`status.roles[worker]` 聚合 Job 上报的副本数。

**Suspend**：原生支持。`spec.runPolicy.suspend=true` → patch `Job.spec.suspend=true`（K8s 原生字段，自动驱逐运行中的 Pod 并停止派生新 Pod），随后返回 `suspendCompleted=true, reason=CancelRequested`。

**RBAC**：`jobs.batch` / `pods` / `events` 的 CRUD。

#### 5.5.2 `(native, podgroup)`

将 MLJob 翻译为 sigs.k8s.io scheduler-plugins `PodGroup` + 裸 Pod，借助 Koordinator gang plugin 实现"全员就位才启动"的单角色任务（如分布式训练的多 Worker 同步启动）。

**前置依赖**：集群已安装 Koordinator；本 Handler 需要 `podgroups.scheduling.sigs.k8s.io` 的 CRUD。

**底层资源**：

- 必填且仅一个 role（`name=worker`）。
- 每个 MLJob 创建一个 `PodGroup`（`scheduling.sigs.k8s.io/v1alpha1`），`spec.minMember ← roles[worker].replicas`。
- 按 `roles[worker].replicas` 创建对应 Worker 裸 Pod；通过 label `pod-group.scheduling.sigs.k8s.io=<podgroup-name>` 关联到 PodGroup。
- PodGroup / Pod 设置 `ownerReference` 指向 MLJob。

**Pod label**：§3.4 列出的 5 项必填 label，外加 `pod-group.scheduling.sigs.k8s.io=<podgroup-name>`。裸 Pod 拓扑没有稳定 index，省略 `axisml.io/replica-index`。

**`backend.config`**：本 Handler 不消费；非空时 `Validate` 返回 warning，不报错。

**通用字段映射**：

| MLJob 字段 | Pod / PodGroup 落点 |
| --- | --- |
| `roles[worker].template.*`（除 resources） | Pod 主容器同名字段 |
| `roles[worker].template.resources` | Pod 主容器同名字段 |
| `roles[worker].restartPolicy` | Pod `spec.restartPolicy` |
| `roles[worker].replicas` | `PodGroup.spec.minMember` 与裸 Pod 数 |
| `spec.scheduling.*` | Pod 同名字段 |
| `spec.scheduling.quota` | Pod label `quota.scheduling.koordinator.sh/name`；PodGroup 不持有 quota 字段 |
| 调度器选择 | Pod `spec.schedulerName=koord-scheduler`（恒定） |
| `spec.runPolicy.activeDeadlineSeconds` | Pod 同名字段 |
| `spec.runPolicy.ttlSecondsAfterFinished` | 终态后由 Handler 显式 GC（裸 Pod 无原生 TTL） |
| `spec.runPolicy.backoffLimit` | 暂不支持；如需重试请由用户脚本兜底。统一的 Handler 内部计数实现见 §5.6 后续工作 |

**Status 映射**：

| 原生状态 | MLJob phase |
| --- | --- |
| 所有 Pod `Pending` 或 PodGroup 排队中 | `Pending` |
| 至少一个 Pod 进入 `Running` | `Running` |
| 所有 Pod `Succeeded` | `Succeeded` |
| 任一 Pod `Failed`、PodGroup 调度不可达、超 `activeDeadlineSeconds` | `Failed` |

`startedAt` 取首个 Pod `Running` 时间；`finishedAt` 取所有 Pod 进入终态的最晚时间。

**Suspend**：原生支持。

1. patch `PodGroup.spec.minMember=0`。
2. 删除现存 Pod；后续 reconcile 看到 `spec.runPolicy.suspend=true` 后不再重建 Pod。
3. 返回 `suspendCompleted=true, reason=CancelRequested`。

**顺序约束**：必须先 patch minMember=0、再删 Pod，否则 koord-scheduler 的 gang plugin 可能立即把刚被删除的 Pod 重新调度。

**RBAC**：`pods` / `podgroups.scheduling.sigs.k8s.io` / `events` 的 CRUD。

#### 5.5.3 `(kubeflow-trainer, *)`

将 MLJob 翻译为 Kubeflow Trainer 的多角色训练 CR。本节锁三件事：路由元组、role 集合约定、Status / Suspend 协议骨架。`backend.config` 详细 schema、elastic / rendezvous 字段、与 PodGroup 的交互细节由独立设计文档落地（见 §5.6）。

**前置依赖**：集群已安装 kubeflow training-operator；本 Handler 需要对应 CR 的 CRUD（`pytorchjobs.kubeflow.org` / `tfjobs.kubeflow.org` / `mpijobs.kubeflow.org` …）。

**Role 集合约定**（按 engine 分）：

| engine | role 集合 | 备注 |
| --- | --- | --- |
| `pytorchjob` | `master`（replicas=1，可省略默认）+ `worker`（replicas≥1），可选 `elasticAgent` | 主线落地 engine |
| `tfjob` | `chief` / `worker` / `ps` / `evaluator`，任一可省略，replicas=0 表示禁用 | 同 §5.5.3 同构扩展 |
| `mpijob` | `launcher`（replicas=1）+ `worker`（replicas≥1） | 同 §5.5.3 同构扩展 |

**通用字段映射骨架**：

- `roles[*].template.*` → 对应 CR 的 `*ReplicaSpecs.<Role>.template`。
- 各 replica 模板的 `template.spec.schedulerName` 必须设为 `koord-scheduler`；`template.metadata.labels` 必须注入 §3.4 列出的 5 项必填 label。
- 多角色 gang 由本 Handler 一并创建 PodGroup CR（`spec.minMember ← sum(replicas)`）；具体 elastic 场景下的 `minMember` 计算策略见 §5.6 后续工作。
- `spec.scheduling.*` → 各 replica 模板内的同名字段。
- `spec.runPolicy.activeDeadlineSeconds` / `backoffLimit` → 后端 CR `spec.runPolicy` 同名字段。

**Status 映射**：从后端 CR 的 `status.conditions` 推导四态——

| 后端 condition | MLJob phase |
| --- | --- |
| `Created` / `Restarting` | `Pending` |
| `Running` | `Running` |
| `Succeeded` | `Succeeded` |
| `Failed` | `Failed` |

**Suspend**：优先使用后端原生 `spec.runPolicy.suspend=true`；目标版本不支持时 fallback 为 `Cleanup()`。无论哪条路径都必须按 §3.6 控制信号义务返回 `suspendCompleted=true, reason=CancelRequested`。

#### 5.5.4 `(custom, *)`

为外部接入预留的一等公民。用户在 `backend.config` 中以 schemaless 方式描述目标 GVK 与字段映射，由 custom Handler 通过 unstructured client 创建并跟踪。

**仍受 §3.4 Pod 注入约定约束**：custom Handler 必须保证最终落地的 Pod 模板上有 `schedulerName: koord-scheduler` 与 `quota.scheduling.koordinator.sh/name` label，否则 Validate 直接拒绝。

> 完整 schemaless `config` schema、JSONPath fieldMappings / statusMappings、unstructured 操作约定由独立设计文档落地（见 §5.6）。当前 dispatcher 是精确 key 查找、未提供 wildcard 基础设施；落地前先确认至少 1-2 个真实接入需求场景。

### 5.6 后续工作

- `(native, job)` Handler 的 Indexed Job 模式与 `podFailurePolicy` 直通策略细节。
- `(native, podgroup)` Handler 的 PodGroup `minResources` 与 elastic gang 演进；统一的 `runPolicy.backoffLimit` 由 Handler 内部计数器实现。
- `(kubeflow-trainer, *)` 各 engine 的完整字段映射 / 状态映射 / `backend.config` schema：
  - `pytorchjob`：`elastic.{enabled, minReplicas, maxReplicas}` + `rdzv.{backend, endpoint}` 详细 schema、与 PodGroup 的交互、与 PyTorch 弹性训练的协议对齐。
  - `tfjob`：`chief / worker / ps / evaluator` 各 role 字段映射细节。
  - `mpijob`：MPI 实现选择（OpenMPI / Intel MPI）与 launcher / worker 通讯参数 schema。
  - `paddlejob` / `xgboostjob` 等扩展 engine 的引入节奏。
- `(custom, *)` Handler 的 `config` 完整 schema 与 unstructured 操作约定（fieldMappings / statusMappings JSONPath 语义、schedulerName / quota label 强制注入校验、wildcard 路由基础设施）。
- Admission webhook：`spec.backend.{name, engine}` 不可变约束、`backend.config` 按 Handler 自带 schema 的统一校验。
- Handler chart 的 values 控制（启用 / 禁用 backend，对应 RBAC 与 watch 的开关）。
- CRD 严格 schema（启用 OpenAPI 校验，`spec.backend.name` enum 收为 `{native, kubeflow-trainer, custom}` 与 `spec.scheduling.quota` 必填）。

## 6. MLService Controller

### 6.1 概述

MLService controller 把 Compute 下发的 `MLService` CR 翻译为底层在线推理资源（Deployment + Service / KServe `InferenceService` / 自定义 GVK），并把执行状态回流到 `MLService.status`。它使用 §3.5 的 dispatcher + handler 模式，与 MLJob 同构：CR 的 `spec.backend.{name, engine}` 二级元组路由到不同 Handler。

`MLService` 为 namespaced CR：

| 字段 | 取值 |
| --- | --- |
| `apiVersion` | `axisml.io/v1alpha1` |
| `kind` | `MLService` |
| `scope` | `Namespaced`（创建在租户 namespace 下） |
| `shortNames` | `mls` |

### 6.2 CRD 契约

#### 6.2.1 spec 设计取舍

把 "角色拓扑" 提升为一等公民，与 MLJob §5.2.1 同源。Service 的执行域用 `roles[]` 数组承载：

- 单角色 backend（如 `(native, deployment)` / `(native, statefulset)`）声明一个 role（约定 `name=predictor`）。
- 多角色 backend（如 KServe `InferenceService` 的 `predictor` / `transformer` / `explainer`）声明多个 role。
- role 名集合由各 Handler 在 §6.6 中约定，由 Handler 的 `Validate` 强制。
- 单角色 Handler `Validate` 拒绝多 role 提交；多角色 Handler 在自身章节中明确开放节奏。

**Service 不引入独立 koordinator backend**：与 MLJob 不同，service 是常驻 + 弹性扩缩 workload，不应默认获得"所有副本同时调度"的 gang 语义，故 `native` 直接走 K8s 原生 Deployment / StatefulSet，不引入额外的 backend 维度。但所有 native Service 与 KServe 派生的 Pod 仍强制走 koord-scheduler 并消耗 ElasticQuota（详见 §3.4）——**无需为 service 创建任何 PodGroup**（Koordinator ElasticQuota 通过 Pod label 直接关联，不依赖中间 PodGroup）。若某类在线服务确实需要 gang / role-level gang（如 PD 分离要求 prefill 与 decode 成组启动），应作为后续显式设计，而不是复用 native 单角色默认行为。

**与 MLJob 的差异点**：

- 顶层 `modelRef`：service 一等字段，指向 Artifacts model version；Handler 据此把模型工件解析为容器侧的位置（环境变量 / volume mount / KServe `storageUri` 等）。
- `roles[*].template.ports[]`：与 K8s `PodSpec.containers[].ports` 同源约定。每个 role 是一个独立的 Deployment / StatefulSet（或 InferenceService 内的 component），各自的容器端口属于该 role 自身。Handler 据此为每个 role 派生一个 K8s Service（targetPort=containerPort）。
- 顶层 `route`：可选；与 Gateway API `HTTPRoute` 同源命名。当 `enabled=true` 时由 Handler 创建 namespaced `HTTPRoute`（搭配 Envoy Gateway 的 `SecurityPolicy` / `BackendTrafficPolicy`）实现自助外部入口；详见 §6.5。`(kserve, *)` Handler 自带 Route 机制，不接受 `route.enabled=true`。
- `runPolicy` 字段集合不同：service 是常驻 workload，**没有** `suspend` / `activeDeadlineSeconds` / `ttlSecondsAfterFinished` / `backoffLimit`；改为 `progressDeadlineSeconds`（rollout 进度超时，与 K8s Deployment 同名字段语义一致）。

#### 6.2.2 spec 结构

```yaml
apiVersion: axisml.io/v1alpha1
kind: MLService
spec:
  # ── 后端选择（创建后不可变；dispatcher 路由依据）─────────────────────
  backend:
    name: native              # 必填: native | kserve | custom
    engine: deployment        # 必填: 语义随 backend 而定（见 §6.6）；engine 与目标 CR 1:1 映射
                              #   native: deployment | statefulset
                              #   kserve: inference | llminference
                              #   custom: 任意名（由 backend.config 描述目标 GVK）
    config: {}                # 可选: 该 (backend, engine) 元组特有的 schemaless 配置

  # ── 调度域 ─────────────────────────────────────────────────────
  scheduling:
    quota: string             # 必填: ElasticQuota CR 名（axisml-<tenant>-<pool>-<quota>）
    priorityClass: string     # 可选
    nodeSelector: {}
    tolerations: []

  # ── 模型引用（service 特有，指向 Artifacts）─────────────────────────
  modelRef:
    name: string
    version: string

  # ── 执行域：roles 数组承载角色拓扑 ─────────────────────────────────
  roles:
    - name: predictor          # 单角色 Handler 仅允许 1 个 role
      replicas: 1              # >= 0；为 0 时视为待调度（status.phase=Pending）
      template:
        image: string
        imagePullPolicy: IfNotPresent
        command: []
        args: []
        env: []
        envFrom: []
        workingDir: string
        ports:                  # 与 K8s containers[].ports 同源；Handler 据此派生该 role 的 K8s Service
          - name: http
            containerPort: 8080
            protocol: TCP
        resources:
          requests: {}          # Compute 从 ResourceUnit.requests 注入
          limits: {}

  # ── 生命周期 ────────────────────────────────────────────────────
  runPolicy:
    progressDeadlineSeconds: int   # 可选: rollout 进度超时；超时后 status.phase=Failed

  # ── 对外路由（可选；默认仅 ClusterIP；与 Gateway API HTTPRoute 同源）────
  route:
    enabled: false             # 默认 false：仅 ClusterIP；true 时创建 HTTPRoute 等资源
    targetRole: string         # 单 role 可省（自动取唯一 role 名）；多 role 必填
    portName: string           # 可选: 选取 roles[targetRole].template.ports[] 中的端口名
                               # 默认取 ports[0].name；多端口时必须显式指定
    hostname: string           # 可选: 外部主机名；不填则继承 Gateway 监听器配置
    path: string               # 可选: HTTPRoute 路径前缀，默认 "/"
    auth:                      # 可选: 认证策略 → SecurityPolicy
      type: none | jwt | apiKey
      jwt: { issuer, jwksUri }
      apiKey: { secretRef: { name } }
    rateLimit:                 # 可选: 限流 → BackendTrafficPolicy
      requestsPerSecond: int
      burst: int
    timeout: string            # 可选: 请求超时（Go duration，如 "30s"）
```

#### 6.2.3 字段归属与不可变性

| 字段路径 | 写入方 | 创建后可变？ |
| --- | --- | --- |
| `metadata.name` / `namespace` / `labels[axisml.io/*]` | Compute | 否 |
| `spec.backend.name` / `spec.backend.engine` | Compute（默认 `{native, deployment}`） | **否** |
| `spec.backend.config` | Compute（默认 `{}`） | 否（仅允许 `roles[*].replicas` 通过 `/scale` 变更；config 热更新见 §6.7） |
| `spec.scheduling.*` | Compute | 否 |
| `spec.modelRef` | 用户提交 | 否（更换模型版本走重建） |
| `spec.roles[*].name` / `template.*`（含 `ports[]`，除 resources） | 用户提交 | 否 |
| `spec.roles[*].template.resources` | Compute（注入 ResourceUnit） | 否 |
| `spec.roles[*].replicas` | API（`/scale` 触发） | **是**（扩缩容路径专用） |
| `spec.runPolicy.progressDeadlineSeconds` | 用户提交 | 否 |
| `spec.route`（整块） | 用户提交 | 否（不可变；mutable 演进见 §6.7） |

**`spec.route` 与 backend 的兼容性**：`(kserve, *)` Handler 在 `Validate` 中拒绝 `spec.route.enabled=true` 的提交（KServe `InferenceService` 自带对外 Route，避免双管）；`(native, *)` 与 `(custom, *)` 接受。

**与 compute.md `services.replicas` 的兼容**：[compute.md §6.3.2](compute.md) 中的 `services.replicas` 字段在单 role 约定下定义为 `spec.roles[0].replicas`；`/scale` API 在 CR 侧 patch path 写 `spec/roles/0/replicas`。多 role 独立扩缩的契约扩展见 §6.7。

### 6.3 Status

```yaml
status:
  observedGeneration: int64
  phase: Pending | Ready | Degraded | Failed   # ← Compute 消费的主状态字段
  message: string                              # Compute 透传到 services.message
  endpoint: string                              # 单一服务地址（Compute 回流到 services.endpoint）
  readyReplicas: int                            # 主 role（单 role 约定下即 roles[0]）就绪副本聚合
  conditions:
    - type: Initialized | Available | Progressing | Failed
      status: True | False | Unknown
      lastTransitionTime: timestamp
      reason: string
      message: string
  roles:                                        # 各 role 副本聚合（UI 可观测，Compute 不消费）
    - name: string
      replicas: int          # spec 期望
      readyReplicas: int     # 通过 readiness probe
```

**`endpoint` 取值规则**（`status.endpoint` 是单一服务地址字段）：

- native/custom 且 `route.enabled=false`（默认） → K8s Service DNS（`<svc>.<ns>.svc.cluster.local:<port>`），ClusterIP / headless Service 共用此格式。
- native/custom 且 `route.enabled=true` → AxisML Gateway 外部 URL（形如 `https://<hostname><path>`）。
- kserve backend → KServe 自带 route/status.url 暴露的 URL（不接受 `spec.route.enabled=true`）。

**role 选择**：native/custom 单 role 取唯一 role 的 Service；多 role 取 `spec.route.targetRole`；未设置 `spec.route` 的多 role 场景由各 Handler 在 §6.6 中约定主 role；kserve 取后端 CR 自带的主入口。

**端口选择**：native/custom `route.enabled=true` 时按 `route.portName`；否则取主 role.template.ports[] 中 `name=http` 的端口；不存在时取 `ports[0]` 并加 warning condition。

**Compute phase 映射规则**（与 [compute.md §6.3.2](compute.md) 对齐）：

| MLService status.phase | services.status | 终态 |
| --- | --- | --- |
| `Pending` | `Pending` | 否 |
| `Ready` | `Ready` | 否 |
| `Degraded` | `Degraded` | 否，可恢复 |
| `Failed` | `Failed` | 否，可恢复（自愈） |

`Pending / Ready / Degraded / Failed` 均为非终态——operator 自愈（重建失败 Pod、健康检查恢复）后 Informer 回流可让 `Failed → Degraded → Ready` 自然恢复；只有 `Deleted` 是 Service 的最终终态，由 Compute Informer 在观察到 CR DELETE 事件后基于 PG 当前 `status` 推导（详见 [compute.md §5.3 / §5.4](compute.md)），不由 operator 产出。

### 6.4 Reconcile 事件路径

Dispatcher 与 Handler 职责切分（通用约束见 §3.3 / §3.6）：

| 事件 | Dispatcher 行为 | Handler 行为 |
| --- | --- | --- |
| MLService ADD | 路由到 Handler；调用 `Validate(spec)`，校验失败写 `status.phase=Failed` | `Reconcile` 创建底层资源，设置 `ownerReference: MLService` |
| MLService UPDATE（仅 `roles[*].replicas` 变更，来自 `/scale`） | 路由 | `Reconcile` 透传为后端资源副本调整；不重建 Pod |
| MLService UPDATE（其他 spec 字段变更） | 校验 `backend.{name, engine}` 不变；其他字段变更属于约束违反，写 `status.message` 拒绝 | 不动 |
| MLService DELETE | 不阻断 | 一般依赖 ownerReference 级联清理 |
| 底层资源事件（Deployment / Service / HTTPRoute / 第三方 CR） | 通过 ownerReference 反查到 MLService 后路由，并从 informer cache 组装相关子资源快照 | `MapStatus` 基于快照纯函数计算新 phase |

### 6.5 spec.route 派生资源

当 `route.enabled=true` 时，Handler 在租户 namespace 内创建 / 更新以下资源（统一打 `axisml.io/service-id` label，`ownerReference: MLService` 级联清理，不引入 finalizer）：

- `HTTPRoute`（`gateway.networking.k8s.io/v1`）：`parentRefs` 指向 `axisml-gateway`（跨 namespace 引用通过 `ReferenceGrant` 授权，由 infra chart 准备），`backendRefs` 指向 `route.targetRole` 对应的 K8s Service。
- `SecurityPolicy`（`gateway.envoyproxy.io/v1alpha1`）：仅当 `auth.type != none` 时创建，`targetRefs` 指向上面的 HTTPRoute。
- `BackendTrafficPolicy`（`gateway.envoyproxy.io/v1alpha1`）：仅当 `rateLimit` 或 `timeout` 非空时创建，`targetRefs` 指向上面的 HTTPRoute。

**spec.route 增量职责**（按 §3.6 Handler 接口）：

- `Reconcile`：根据创建时确定的 `spec.route.enabled` 与各子字段创建 / 保持上面三类派生资源；`Validate` 拒绝 `(kserve, *)` 下的 `enabled=true`、拒绝多 role 但未指定 `targetRole` 的提交、拒绝多端口但未指定 `portName` 的提交。
- `MapStatus`：把 HTTPRoute `Accepted` / `ResolvedRefs` condition 翻译为 `status.endpoint`（按 §6.3 端口选择规则填写外部 URL）与 `status.conditions` 的 `Available` 条件；HTTPRoute `Accepted=False` 视同后端未就绪，应让 `phase=Degraded` 并把失败原因写入 `message`。

### 6.6 内置 Handler

每个 Handler 章节统一按以下小节组织：**底层资源 / `backend.config` / 通用字段映射 / Status 映射 / Scale / RBAC**。

#### 6.6.1 `(native, deployment)`

底层用 K8s 原生 Deployment + Service。所有 Pod 走 koord-scheduler 并通过 Pod label 计入 ElasticQuota，**不**创建 PodGroup。

**底层资源**：

- 必填且仅一个 role（`name=predictor`）；`Validate` 拒绝多 role 提交或其他 role 名。
- 每个 MLService 创建一个 K8s `Deployment` 与一个 K8s `Service`：
  - `Service` 端口由 `roles[predictor].template.ports[]` 派生（`targetPort=containerPort`）。
  - Deployment Pod 模板上设置 `schedulerName: koord-scheduler`，并打 §3.4 列出的 5 项必填 label。
- 当 `spec.route.enabled=true` 时追加 HTTPRoute + 可选的 SecurityPolicy / BackendTrafficPolicy（与 §6.5 派生资源说明一致）。
- Deployment / Service / 派生路由资源设置 `ownerReference` 指向 MLService。

**Pod label**：见 §3.4（Deployment Pod 没有稳定 index，按约定省略 `axisml.io/replica-index`）。

**`backend.config`**：本 Handler 不消费；非空时 `Validate` 返回 warning，不报错。

**通用字段映射**：

| MLService 字段 | Deployment / Service / 派生路由资源落点 |
| --- | --- |
| `roles[predictor].template.image` / `imagePullPolicy` / `command` / `args` / `env` / `envFrom` / `workingDir` | Deployment Pod 主容器同名字段 |
| `roles[predictor].template.ports[]` | Deployment Pod 主容器 `ports` + K8s Service `spec.ports`（`targetPort=containerPort`） |
| `roles[predictor].template.resources` | Deployment Pod 主容器同名字段 |
| `roles[predictor].replicas` | `Deployment.spec.replicas` |
| `spec.scheduling.quota` | Pod label `quota.scheduling.koordinator.sh/name`（ElasticQuota 全名）；不创建 PodGroup |
| 调度器选择 | Pod `spec.template.spec.schedulerName=koord-scheduler`（恒定） |
| `spec.scheduling.priorityClass` / `nodeSelector` / `tolerations` | Pod 同名字段 |
| `spec.modelRef` | Artifacts client 解析为模型工件 URI，注入为环境变量 `AXISML_MODEL_URI` |
| `spec.route.targetRole` | 选取 HTTPRoute `backendRefs.name` 指向的 K8s Service（单 role 时省略，自动取 `predictor`） |
| `spec.route.portName` | HTTPRoute `backendRefs.port`（解析为 `targetRole` Service 中对应的端口） |
| `spec.route.hostname` / `path` | `HTTPRoute.spec.hostnames` / `rules[].matches[].path.value`（path 默认 `/`） |
| `spec.route.auth` | `SecurityPolicy.spec.{jwt | apiKeyAuth}`，`targetRefs` 指向上面 HTTPRoute |
| `spec.route.rateLimit` / `timeout` | `BackendTrafficPolicy.spec.rateLimit` / `timeout`，`targetRefs` 指向上面 HTTPRoute |
| `spec.runPolicy.progressDeadlineSeconds` | `Deployment.spec.progressDeadlineSeconds` |

**Status 映射**（沿用 [compute.md §6.3.2](compute.md) 规则）：

| 条件 | MLService phase |
| --- | --- |
| `desired_replicas == 0` | `Pending`（扩缩至 0，视为待调度 / 停用） |
| `ready_replicas == 0 && desired_replicas > 0` 且 rollout 仍在推进中（`Progressing=True`，未超过 `progressDeadlineSeconds`，无 `ReplicaFailure`） | `Pending` |
| `ready_replicas == desired_replicas && desired_replicas > 0` | `Ready` |
| `0 < ready_replicas < desired_replicas` | `Degraded` |
| `ready_replicas == 0 && desired_replicas > 0` 且 Deployment 超过 `progressDeadlineSeconds` 或出现 `ReplicaFailure` / `ProgressDeadlineExceeded` | `Failed` |

`endpoint` 按 §6.3 二分规则填写。`spec.route.enabled=true` 且 HTTPRoute `Accepted=False` 时——即 Deployment 已就绪但外部入口未生效——映射为 `phase=Degraded`，`message` 写明 HTTPRoute 拒绝原因；同时 `endpoint` 暂时回退为内部 Service DNS。

**Scale**：patch `Deployment.spec.replicas`；不重建 Pod。

**RBAC**：

- 基础：`deployments.apps` / `services` / `pods` / `events` 的 CRUD。
- `spec.route` 派生资源：`httproutes.gateway.networking.k8s.io` / `securitypolicies.gateway.envoyproxy.io` / `backendtrafficpolicies.gateway.envoyproxy.io` 的 CRUD。
- `secrets` 的 `get / list / watch`（仅当 `spec.route.auth.type=apiKey` 引用 Secret 时）。

#### 6.6.2 `(native, statefulset)`

为有状态推理（在线 KV cache、模型分片、节点身份固定的副本）预留。底层用 K8s `StatefulSet` + headless Service，副本身份稳定；其余约束沿用 §6.6.1。

**底层资源**：

- 必填且仅一个 role（`name=predictor`）。
- 每个 MLService 创建一个 `StatefulSet` 与一个 headless `Service`（`spec.clusterIP=None`）；StatefulSet Pod 模板上设置 `schedulerName: koord-scheduler` + §3.4 必填 label。
- Pod 通过 `<pod>.<svc>.<namespace>.svc.cluster.local` 直连。
- StatefulSet Pod 副本身份稳定，Handler 透传 K8s 注入的 `apps.kubernetes.io/pod-index` 为 `axisml.io/replica-index`。

**`backend.config` 关键字段**（基础部分）：

```yaml
config:
  podManagementPolicy: OrderedReady | Parallel   # 默认 OrderedReady
  serviceName: string                             # headless Service 名；不填默认 = MLService 名
```

> `volumeClaimTemplates` / `updateStrategy.{type, partition}` 等存储与灰度更新维度由独立设计文档落地，见 §6.7 后续工作。

**通用字段映射**：与 §6.6.1 相同，`roles[predictor].replicas` 落到 `StatefulSet.spec.replicas`；`roles[predictor].template.ports[]` 落到 StatefulSet 主容器 `ports` + headless Service `spec.ports`；补充 `serviceName` 字段。

**Status 映射**：从 `StatefulSet.status` 推导，规则与 §6.6.1 同构。

**Scale**：patch `StatefulSet.spec.replicas`；副本身份保留，扩容时新 index 追加，缩容时按高 index 优先终止。

**RBAC**：

- 基础：`statefulsets.apps` / `services` / `pods` / `events` 的 CRUD。
- `spec.route` 派生资源：与 §6.6.1 同。

#### 6.6.3 `(kserve, inference)`

将 MLService 翻译为 KServe [`InferenceService`](https://kserve.github.io/website/) CR（`serving.kserve.io/v1beta1`）。这是 KServe 通用 ML 服务路径——predictor 内的具体 runtime（NVIDIA Triton / [vLLM](https://docs.vllm.ai/) / TF Serving / TorchServe / sklearn / huggingface 等）由 `backend.config.runtime` 选择。

**前置依赖**：集群已安装 KServe，且版本支持 `InferenceService.spec.predictor.schedulerName` 与 `spec.predictor.labels` 透传到派生 Pod（`PredictorSpec` 内联 `corev1.PodSpec` 与 `ComponentExtensionSpec`，是 KServe v1beta1 的标准契约；这是 §3.4 quota 全覆盖契约的硬要求，落地 KServe 版本以 axisml-infra 安装时 pin 的 stable 版本为准）。本 Handler 仅需要 `inferenceservices.serving.kserve.io` 的 CRUD，外加各 runtime 对应 `(Cluster)ServingRuntime` 的 `get / list / watch`。

**Role 集合约定**：当前仅开放 `predictor`（replicas≥0）；扩展角色 `transformer` / `explainer` 的接入节奏见 §6.7。

**`backend.config` 关键字段**（schema 待细化设计文档）：

```yaml
config:
  runtime: triton | vllm | tfserving | torchserve | sklearn | huggingface | <自定义 ServingRuntime 名>
  predictor:
    minReplicas: int              # 默认 = roles[predictor].replicas
    maxReplicas: int              # 自动扩缩上限；不填则等于 minReplicas
    scaleToZero: bool
    protocolVersion: v1 | v2
  storageUri: string              # 模型工件位置；可由 Artifacts 通过 modelRef 自动解析
  containerOverrides: {}

  # ── runtime 专属子段 ──
  triton:    { modelControlMode, modelRepository }
  vllm:      { model, dtype, tensorParallelSize, pipelineParallelSize, maxModelLen, maxNumBatchedTokens, enablePrefixCaching, quantization, extraArgs }
  huggingface: { task, modelId }
  torchserve: { modelStore }
```

**通用字段映射**：

- `roles[predictor].template.image` → predictor 容器（不填时由 `config.runtime` 选定的 ServingRuntime 提供默认镜像）。
- `roles[predictor].template.{command, args, env, envFrom, workingDir, ports}` → predictor 容器同名字段。
- `roles[predictor].template.resources` → predictor `resources`。
- `roles[predictor].replicas` → 写入 `predictor.minReplicas`；若未设置 `config.predictor.maxReplicas`，则同时写入 `maxReplicas`。
- `spec.modelRef` → 通过 Artifacts 解析为 `predictor.storageUri`（runtime=triton 时也可解析为 `triton.modelRepository`；runtime=vllm 时优先解析为 `vllm.model`，缺失时回退到 `storageUri`）。
- `spec.scheduling.quota` → 写入 `InferenceService.spec.predictor.schedulerName=koord-scheduler` 与 `spec.predictor.labels` 中的 `quota.scheduling.koordinator.sh/name` + `axisml.io/quota`，让 KServe 透传到派生 Pod。
- `spec.scheduling.priorityClass` / `nodeSelector` / `tolerations` → predictor 同名字段。
- `spec.runPolicy.progressDeadlineSeconds` → KServe 暂无对等字段，Handler 在 `Validate` 中返回 warning，dispatcher 记录 event 或 warning condition，不阻塞创建。
- `spec.route` → **不支持**；Handler 在 `Validate` 中拒绝 `spec.route.enabled=true`。

**runtime 专属约束**（由 `Validate` 强制）：

- `runtime=vllm`：`roles[predictor].template.resources.requests["nvidia.com/gpu"]` 必须等于 `config.vllm.tensorParallelSize × pipelineParallelSize`。
- `runtime=huggingface`：`config.huggingface.task` 必填。
- 其他 runtime 的强制项由 ServingRuntime 自身校验，本 Handler 透传。

**Status 映射**：从 `InferenceService.status.conditions` 推导——

| InferenceService condition | MLService phase |
| --- | --- |
| `desired==0` | `Pending` |
| `Ready=False` / `PredictorReady=False` 且仍在创建或滚动更新中 | `Pending` |
| `Ready=True` | `Ready` |
| `PredictorReady=False` 且 `0 < ready < desired` | `Degraded` |
| `Ready=False` 且 `ready==0 && desired>0`，并且 KServe condition 明确失败或超过进度期限 | `Failed` |

`endpoint` 取 `InferenceService.status.url`。

**Scale**：patch `InferenceService.spec.predictor.{minReplicas, maxReplicas}`。

**Quota 与 autoscaling 的相互作用**：KServe scale-to-zero / 自动扩缩可能让实际副本数动态变化；Compute quota 按 `maxReplicas × requests` 上限计费。具体细节由独立设计文档落地（§6.7）。

#### 6.6.4 `(kserve, llminference)`

> KServe LLM API 的 GVK / CRD 字段路径仍在演进，最终以引入版本为准。本节先锁两件事——路由元组与 role 命名约定。详细 `backend.config` schema、字段映射、`Validate` 强制项、Status condition 名等待 KServe LLM API GA 后在 §6.7 单独成文。

将 MLService 翻译为 KServe LLM 原生 CR `LLMInferenceService`（占位命名）。该 engine 承载 LLM 在线服务的 **PD 分离（disaggregated serving）**：prefill 与 decode 拆成独立角色独立扩缩，搭配 router 角色做请求分发与 KV cache 协调。

**前置依赖**：集群已安装 KServe LLM API。本 Handler 需要 `llminferenceservices.serving.kserve.io`（占位）的 CRUD。

**Role 集合约定**：

- `prefill`：长上下文处理（compute-bound）；replicas≥1。
- `decode`：token 生成（memory-bound）；replicas≥1。
- `router`：请求入口与 KV cache 协调；replicas≥1；承载 KServe LLM API 自带的对外入口。

`Validate` 强制：role 名属于上述集合；至少存在 `prefill` 与 `decode`。与 §6.6.3 一样拒绝 `spec.route.enabled=true`，避免同时由 AxisML Gateway 与 KServe LLM router 管理入口。

> 完整设计（PD 分离 `backend.config` schema、KV cache 传输契约 nixl / mooncake、各 role parallelism schema、单副本 GPU 数与 `tensorParallelSize × pipelineParallelSize` 校验、autoscaling 与 quota 计费策略）见 §6.7 后续工作。

#### 6.6.5 `(custom, *)`

为外部接入预留的一等公民。用户在 `backend.config` 中以 schemaless 方式描述目标 GVK 与字段映射，由 custom Handler 通过 unstructured client 创建并跟踪。

**仍受 §3.4 Pod 注入约定约束**：custom Handler 必须保证最终落地的 Pod 模板上有 `schedulerName: koord-scheduler` 与 `quota.scheduling.koordinator.sh/name` label，否则 Validate 直接拒绝。

**`spec.route` 在 custom Handler 下的语义**：由 `config.routeBackend`（独立设计文档定义）显式描述外部入口对接的目标 Service；未在 `config` 中 wire `spec.route` 时，Handler 应在 `Validate` 中拒绝 `spec.route.enabled=true`。

> 完整 schemaless `config` schema、JSONPath fieldMappings / statusMappings / endpointPath、unstructured 操作约定由独立设计文档落地（见 §6.7）。当前 dispatcher 是精确 key 查找、未提供 wildcard 基础设施；落地前先确认至少 1-2 个真实接入需求场景。

### 6.7 后续工作

- `(native, statefulset)` Handler 的 `volumeClaimTemplates` 持久卷模板、`updateStrategy.{type, partition}` 灰度更新、pod-index 寻址细节。
- 多 role 接入的具体 Handler 落地：
  - `(kserve, inference)` 的 `transformer` / `explainer` 字段映射与状态映射。
  - `(kserve, llminference)`（对应 `LLMInferenceService` 占位 GVK，最终以 KServe LLM API 落地版本为准）：vLLM disaggregated / llm-d / NVIDIA Dynamo 等场景下的命名约定、KV cache 传输契约（nixl / mooncake）、各 role parallelism schema、单副本 GPU 数与 `tensorParallelSize × pipelineParallelSize` 校验、autoscaler 接入。
- KServe scale-to-zero 与 Compute quota 的精细交互模型（含 `maxReplicas × requests` 上限计费策略）。
- `(custom, *)` Handler 的 `config` 完整 schema 与 unstructured 操作约定：JSONPath fieldMappings / statusMappings / endpointPath 语义、`config.routeBackend` 与 `spec.route` 的对接细则、wildcard 路由基础设施。
- 多 role 独立扩缩容的 `/scale` API 扩展（路径中携带 role 名）。
- `spec.route` 可变化路径（轮换 API key / 调整限流不需要重建 Service；Handler 侧需要识别哪些子字段可热更新、哪些必须重建派生资源）。
- `spec.route` 与 KServe 自带 Route 的统一化（让 `(kserve, *)` 也支持 `spec.route` 而非依赖 KServe 内置 Route）。
- Admission webhook：`spec.backend.{name, engine}` 不可变约束、`backend.config` 按 Handler 自带 schema 的统一校验。
- Handler chart 的 values 控制（启用 / 禁用 backend，对应 RBAC 与 watch 的开关）。
- CRD 严格 schema（启用 OpenAPI 校验，替换当前的 `x-kubernetes-preserve-unknown-fields`）。

---

## Part IV — 实施与验证

> 本部分给出 axisml-operator 的功能落地路线、测试策略与跨文档引用。新贡献者读完前三部分后从这里看"先做什么、再做什么、怎么验证"。

## 7. 实现路径

按功能优先级把交付内容映射到三个阶段。MVP 划定"能跑通端到端最小可发布范围"，功能完善覆盖主流场景与生产硬化，未来规划承接需求未明朗或上游依赖未稳定的方向。

### 7.1 阶段总览

```
┌──────────────────────────────────────────────────────────────┐
│ MVP（最小可发布）                                             │
│   单一 Pod / 三 Reconciler / 两个 Job backend / 两个 Service  │
│   backend / 完整 Tenant / L1 envtest                         │
│   ↓                                                           │
│ 功能完善（生产硬化）                                          │
│   补齐主流 Handler、外部入口策略、admission webhook、严格      │
│   CRD schema                                                  │
│   ↓                                                           │
│ 未来规划（需求 / 上游驱动）                                    │
│   custom backend、KServe LLM API、分层配额、加密源、运行时    │
│   插件                                                        │
└──────────────────────────────────────────────────────────────┘
```

每条目标都附完成信号，便于阶段闭合验证。

### 7.2 阶段一：MVP（最小可发布）

支撑端到端最小演示路径："创建 Tenant → 提交单角色 MLJob 与单 Service → 看到 phase 收敛"。

| 模块 | 范围 | 完成信号 |
| --- | --- | --- |
| Operator binary | 单一 Manager 承载三 Reconciler、`--enable-*` flag、leader election、Cache `ByObject` 选择性过滤（§2） | 单 Pod 启动后三 controller 同时 Ready；`--enable-mljob=false` 能独立关闭 MLJob 的 watch 与 RBAC |
| Tenant | Namespace 创建（永不删除）、ElasticQuota 1:1 派生 + `status.used` 回流、ImagePullSecrets / Secrets / ConfigMaps / SAs + RBAC initResources、suspend、Compute phase 映射（§4） | envtest 覆盖：happy path、suspend / unsuspend、quota update、源 Secret 缺失 |
| MLJob dispatcher | `(backend, engine)` 路由、`Validate` 拒绝未注册元组、Suspend cancel 推进信号合并写入 `conditions[type=Suspended,reason=CancelRequested]`（§5.4） | envtest 覆盖：未知 backend 直接进入 `Failed`；suspend 后 condition 与 phase 变化符合 §5.3 |
| MLJob `(native, job)` | K8s Job + koord-scheduler、quota label 注入、原生 `Job.spec.suspend` cancel（§5.5.1） | envtest 覆盖 happy path + suspend cancel |
| MLJob `(native, podgroup)` | scheduler-plugins PodGroup + 裸 Pod、`minMember=0` → 删 Pod 的暂停顺序（§5.5.2） | envtest 覆盖 happy path + suspend shutdown |
| MLService dispatcher | 同上 + `/scale` 透传 | envtest 覆盖：未知 backend 直接 `Failed`；scale 修改 `roles[0].replicas` 触发后端副本调整 |
| MLService `(native, deployment)` | Deployment + ClusterIP Service + 基础 HTTPRoute（仅创建 `HTTPRoute`，不创建 `SecurityPolicy` / `BackendTrafficPolicy`）（§6.6.1） | envtest 覆盖 happy path、route 启用 / 禁用、scale、字段不可变性 |
| MLService `(native, statefulset)` | StatefulSet + headless Service、透传 `apps.kubernetes.io/pod-index` 为 `axisml.io/replica-index`、scale 路径 patch `spec.replicas`（§6.6.2） | envtest 覆盖 happy path、scale、字段不可变性 |
| CRD | 三个 CRD 用 `x-kubernetes-preserve-unknown-fields: true` 宽松 schema + `subresources.status` 显式声明 | helm install / upgrade 通过；status 写入不影响 spec 写入 |
| 测试 | L1 envtest 一个 module 七个文件覆盖三 controller 的 happy path + suspend + immutability | `make operator-envtest` 通过 |

### 7.3 阶段二：功能完善（生产硬化）

按"对生产可用性的影响"排序，每条标明完成信号。

1. **MLService `route.auth` + `route.rateLimit / timeout` 派生资源**
   - 目标：`(native, deployment)` Handler 在 `route.auth.type != none` 时创建 `SecurityPolicy`；在 `rateLimit` / `timeout` 非空时创建 `BackendTrafficPolicy`；统一 ownerReference 到 MLService。
   - 完成信号：envtest 覆盖 `jwt` / `apiKey` / 限流 三条派生资源路径；`Validate` 不再只返回 warning。
2. **MLJob `(kubeflow-trainer, pytorchjob)` 主线**
   - 目标：覆盖多角色分布式训练里使用最广的 backend；以此把 §5.5.3 骨架扩展为完整 handler。
   - 完成信号：handler 注册到 dispatcher、`master + worker` role 校验、Status condition 映射到四态、原生 suspend + `Cleanup()` fallback 二选一；envtest 覆盖 happy path 与 suspend。
3. **MLService `(kserve, inference)` 主线（vllm / triton 优先）**
   - 目标：推理主路径接入。前置依赖：axisml-infra 安装时 pin KServe 版本、确认 `InferenceService.spec.predictor.{schedulerName, labels}` 透传到派生 Pod。
   - 完成信号：handler 写 `InferenceService` 时强制注入 `schedulerName: koord-scheduler` + quota label、`status.url` 回流到 `endpoint`、scale 路径 patch `predictor.{minReplicas, maxReplicas}`；envtest（用 fake KServe CRD）覆盖 vllm + triton 两条 runtime 校验分支。
4. **Admission webhook 上线**
   - 目标：把 `spec.backend.{name, engine}` 不可变、`spec.namespace.name` 不可变、跨 namespace `sourceXxxRef` 白名单、`backend.config` 按 Handler schema 校验，从 controller 兜底前移到准入阶段。
   - 完成信号：webhook server 部署、cert-manager 颁证；上述 4 类规则各 1 条 envtest（带 `--admission-webhook` flag）通过；`Validate` 实现保持纯函数，可被 webhook 与 controller 同时复用。
5. **严格 CRD OpenAPI schema**
   - 目标：替换 `x-kubernetes-preserve-unknown-fields: true`；`spec.backend.name` enum、`spec.scheduling.quota` required、phase enum 收紧、各 role 字段类型显式化。
   - 完成信号：三个 CRD 移除 `preserve-unknown-fields`；envtest 覆盖"非法 enum 值被 apiserver 直接拒绝"。
6. **resync 间隔 Helm values 暴露**
   - 目标：默认 10 min；运维侧可调到分钟级以缩短源 Secret / ConfigMap 的更新延迟。
   - 完成信号：Helm values `operator.controllers.tenant.resyncPeriod` 透传到 `--resync-period` flag；启动期校验下限。
7. **目标 Namespace allowlist / denylist 默认硬化**
   - 目标：把 §4.6.1 列出的 `kube-*` / `default` / `axisml-system` 默认拒绝列表落到 Helm `values.yaml` + 启动期校验。
   - 完成信号：默认 denylist 在代码与 chart 中保持一致；envtest 覆盖"试图把 Tenant 落到 `kube-system` 被拒绝并写 `status.message`"。

### 7.4 阶段三：未来规划

- **MLJob `(kubeflow-trainer, tfjob)` / `(mpijob)` / `paddlejob` / `xgboostjob`**——用户驱动；与 pytorchjob 同构，落地时复用 §5.5.3 骨架。
- **MLService `(kserve, llminference)` PD 分离**——等 KServe LLM API GA；KV cache 传输契约（nixl / mooncake）由独立设计文档落地。
- **MLService `(kserve, inference)` 多 role**（`transformer` / `explainer`）——前提是 KServe 自身的多 component 编排稳定。
- **MLJob / MLService `(custom, *)` Handler**——schemaless 字段映射 + JSONPath 操作；引入前先确认 1-2 个真实需求场景，避免做 "pluggable but used by nobody" 的接口。
- **多 role 独立扩缩 `/scale` 路径**——`services.replicas` 字段从单 role 升级为 role 显式寻址。
- **`spec.route` 可热更新路径**——轮换 API key / 调限流不重建 Service / Deployment。
- **`spec.route` 与 KServe 自带 Route 的统一化**——让 `(kserve, *)` 也走 AxisML Gateway 而非 KServe 内置 Route。
- **分层配额**——`spec.quotas[].parent` 落到 ElasticQuota `quota.scheduling.koordinator.sh/parent` annotation。
- **加密源 / Sealed Secrets / KMS / Vault** 作为 `sourceSecretRef` 的替代方案。
- **`spec.initResources` templating**——按 tenant 上下文渲染 ConfigMap 数据。
- **跨 Namespace 复制源 RBAC 收敛**——把允许的源 Namespace 限定为单一受控 Namespace。
- **运行时插件加载机制**——仅当出现"运行时安装新后端"的硬需求时再考虑；默认结论是编译期 `init()` 注册 + 多 binary 路由就够了。
- **`(native, podgroup)` 的 elastic gang / `minResources`**——与上层 PodGroup 演进绑定。
- **跨 controller `runPolicy.backoffLimit` 实现**——由 Handler 内部计数器统一支持。

### 7.5 跨阶段验证策略

| 阶段 | 主测层 | 工具 |
| --- | --- | --- |
| MVP | L1 envtest | `make operator-envtest` |
| 功能完善 | L1 envtest 扩展 + 关键路径 L2 e2e | `make envtest-test` + `make e2e-test`（minikube） |
| 未来规划 | 单独写 RFC 设计文档 → L1 envtest 先行 → L2 验证多组件链路 | 同上 |

## 8. 测试

L1 envtest 在 `components/operator/test/envtest/` 单一 Go module 中，单一 `TestMain` 把三个 reconciler 注册到同一个 envtest manager，跑七个 test 文件（tenant 2 + mljob 3 + mlservice 2）。CRDPaths 是 `deploy/helm/axisml-system/crds` 与 `test/crds/external/`（vendored ElasticQuota / PodGroup / HTTPRoute）的并集。

L2 e2e 在 `test/e2e/`，通过部署后的 axisml-operator 与 MLPlatform / Compute API 一起跑端到端。

## 9. 相关引用

- [docs/system_design/overview.md §5.3](overview.md) 概述了 axisml-operator 在控制平面里的位置。
- [docs/system_design/compute.md](compute.md) 描述 ml-compute 与 operator 之间的 CR 写路径与状态回流。
- [docs/system_design/infra.md §8](infra.md) 给出 koord-scheduler / ElasticQuota / Gateway API 等基础设施依赖契约。
