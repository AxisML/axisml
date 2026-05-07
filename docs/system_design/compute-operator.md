# AxisML Compute Operator 详细设计

compute-operator 是 AxisML 控制平面里负责"业务负载"调度的 Kubernetes operator 二进制。它持有一个 Manager，承载 MLJob 与 MLService 两个 controller，把 [compute](compute.md) 服务下发的 `MLJob` / `MLService` CR 翻译为底层 Kubernetes 与第三方资源（Job / Pod / PodGroup / Deployment / StatefulSet / HTTPRoute / KServe `InferenceService` ...），并把执行状态回流到 CR `status`。

| Controller | CRD（`axisml.io/v1alpha1`） | Scope | 架构 | 状态机 | 主要外部依赖 |
| --- | --- | --- | --- | --- | --- |
| MLJob ([§4](#4-mljob-controller)) | `MLJob` | Namespaced | dispatcher + handler 注册表 | `Pending \| Running \| Succeeded \| Failed` | scheduler-plugins PodGroup（Koordinator vendored）、Kubeflow Trainer |
| MLService ([§5](#5-mlservice-controller)) | `MLService` | Namespaced | dispatcher + handler 注册表 | `Pending \| Ready \| Degraded \| Failed` | Gateway API HTTPRoute、KServe |

compute-operator **不感知 Tenant CR**——租户级资源由 [tenant-operator](tenant-operator.md) 独立维护。compute-operator 只在 MLJob / MLService CR 自身的 namespace 与 spec 范围内工作；`spec.scheduling.quota` 是一个由 compute 透传的 ElasticQuota CR 名字符串，operator 视为不透明字段。

**文档组织**：

- **Part I — 运行时框架**（§1 架构总览 + §2 运行时契约）：单一 Deployment / Manager 的运维契约。
- **Part II — 通用契约**（§3 跨 controller 通用契约）：两个 controller 共享的 spec/status 边界、Reconcile 约束、Pod 注入、dispatcher + handler 架构、Handler 接口契约。§4–§5 引用本节而不重复。
- **Part III — Controller 详细设计**（§4 MLJob、§5 MLService）：CRD 字段、状态推导、Handler 落地。
- **Part IV — 实施与验证**（§6 实现路径、§7 测试、§8 相关引用）。

两个 controller 通过 `--enable-{mljob,mlservice}` 单独启用 / 关闭，详见 §2.3。

---

## Part I — 运行时框架

## 1. 架构总览

```
┌──────────── compute-operator (one Pod, leader-elected) ────────────┐
│                                                                     │
│  ctrl.Manager (scheme: clientgoscheme + axisml.{mljob,mlservice} +  │
│                scheduling.sigs.k8s.io PodGroup +                    │
│                gateway.networking.k8s.io HTTPRoute)                 │
│  Lease: axisml-compute-operator.axisml.io                           │
│                                                                     │
│  ┌────────────────────────┐  ┌───────────────────────────┐          │
│  │ MLJob                  │  │ MLService                 │          │
│  │ Dispatcher + Registry  │  │ Dispatcher + Registry     │          │
│  │ → handlers/{nativejob, │  │ → handlers/{nativedeploy, │          │
│  │   nativepodgroup,      │  │   nativestatefulset,      │          │
│  │   kubeflow-*, custom}  │  │   kserve-*, custom}       │          │
│  └────────────────────────┘  └───────────────────────────┘          │
│              │                            │                         │
│              ▼                            ▼                         │
│   Job, Pod, PodGroup              Deployment, Service,              │
│   (koord-scheduler                HTTPRoute (Gateway API),          │
│    gang scheduling)               KServe InferenceService           │
└─────────────────────────────────────────────────────────────────────┘
```

MLJob 与 MLService 同构使用 **dispatcher + handler** 模式：CR 的 `spec.backend.{name, engine}` 元组路由到注册过的 Handler，Handler 渲染目标 GVK 并把状态回流到 CR.status——见 §3.5。所有 backend 派生的 Pod 强制走 koord-scheduler 并消费对应 ElasticQuota（[infra.md](infra.md)）。

## 2. 运行时契约

### 2.1 Scheme 注册

```go
clientgoscheme.AddToScheme(scheme)           // core, apps, rbac, batch, coordination
schedulingv1alpha1.AddToScheme(scheme)       // PodGroup（Koordinator vendored）
gwapiv1.Install(scheme)                      // HTTPRoute
mljob_v1alpha1.AddToScheme(scheme)           // MLJob
mlservice_v1alpha1.AddToScheme(scheme)       // MLService
mlservicehandler.RegisterStubs()             // MLService 占位 handler 注册
```

MLJob / MLService 的 Go 类型分别定义在 `components/compute-operator/api/{mljob,mlservice}/v1alpha1/` 子包，避免 `Phase`、`RoleSpec` 等同名常量在同一包内冲突。每个子包的 `groupversion_info.go` 用 `runtime.SchemeBuilder` 声明 `SchemeBuilder` + 私有 `addKnownTypes(...)` helper；各 `<kind>_types.go` 的 `init()` 调用 `addKnownTypes(&Foo{}, &FooList{})` 完成注册。

compute-operator 不注册 `Tenant` CRD 类型，也不注册 `ElasticQuota` 类型——前者归 tenant-operator，后者操作完全在 tenant-operator 一侧。

### 2.2 Cache 选择性过滤

compute-operator 监听 `MLJob`、`MLService` 与若干派生底层资源（`Job` / `Pod` / `PodGroup` / `Deployment` / `StatefulSet` / `Service` / `HTTPRoute` / KServe `InferenceService`），不需要 label-selector 收敛 cache——派生资源都通过 `ownerReference` 反查回 CR，且 CR 自身不会在集群内大规模存在。**不要**给 cache 加全局 `DefaultLabelSelector`，否则 Pod / PodGroup informer 会丢事件。

### 2.3 Flag 集合

| Flag | 默认 | 说明 |
| --- | --- | --- |
| `--metrics-bind-address` | `:8080` | Prometheus metrics |
| `--health-probe-bind-address` | `:8081` | `/healthz`, `/readyz` |
| `--leader-elect` | `true` | leader election |
| `--leader-election-id` | `axisml-compute-operator.axisml.io` | Lease 名 |
| `--enable-mljob` | `true` | 启用 MLJob controller |
| `--enable-mlservice` | `true` | 启用 MLService controller |

`--enable-*` flag 的 default 值通过 Helm value 渲染；`--enable-mljob=false` 时对应 reconciler 不挂到 Manager，且 ClusterRole 中相关分段不渲染——做到"按需启用"的最小权限。

### 2.4 RBAC

compute-operator 只声明**一个** ClusterRole（`<release>-compute-operator`），rules 是两个 controller 所需权限的并集，按 controller 分段；段头按 `--enable-*` Helm value 条件渲染。leader election Lease 在部署 namespace 通过 Role + RoleBinding 授权。

| 资源 | 动作 | 用途 |
| --- | --- | --- |
| `mljobs.axisml.io` | `get / list / watch / patch` | watch MLJob CR、写 status |
| `mlservices.axisml.io` | `get / list / watch / patch` | watch MLService CR、写 status |
| `jobs.batch` | `create / get / list / watch / update / patch / delete` | `(native, job)` Handler |
| `pods` | `create / get / list / watch / update / patch / delete` | 多 Handler 共用（Pod 创建 / 删除 / 终态观测） |
| `pods/log` | `get` | （留给 compute 服务侧透传，本 operator 不直接读） |
| `events` | `create / get / list / watch / patch` | 写 K8s Event / 观测后端事件 |
| `podgroups.scheduling.sigs.k8s.io` | `create / get / list / watch / update / patch / delete` | `(native, podgroup)` 与 `(kubeflow-trainer, *)` Handler |
| `deployments.apps` | `create / get / list / watch / update / patch / delete` | `(native, deployment)` Handler |
| `statefulsets.apps` | `create / get / list / watch / update / patch / delete` | `(native, statefulset)` Handler |
| `services` | `create / get / list / watch / update / patch / delete` | MLService Handler 派生的 K8s Service |
| `httproutes.gateway.networking.k8s.io` | `create / get / list / watch / update / patch / delete` | `spec.route.enabled=true` 派生 |
| `securitypolicies.gateway.envoyproxy.io` | `create / get / list / watch / update / patch / delete` | `spec.route.auth` 派生 |
| `backendtrafficpolicies.gateway.envoyproxy.io` | `create / get / list / watch / update / patch / delete` | `spec.route.rateLimit` / `timeout` 派生 |
| `inferenceservices.serving.kserve.io` | `create / get / list / watch / update / patch / delete` | `(kserve, *)` Handler |
| Kubeflow CRDs（`pytorchjobs.kubeflow.org` / `tfjobs.kubeflow.org` / ...） | `create / get / list / watch / update / patch / delete` | `(kubeflow-trainer, *)` Handler |
| `coordination.k8s.io/leases`（自身 ns） | `create / get / list / watch / update / patch / delete` | leader election Lease |

**不含**：`tenants.axisml.io`、`elasticquotas.scheduling.sigs.k8s.io`、`namespaces` / `secrets` / `configmaps` / `serviceaccounts`——这些归 tenant-operator。

### 2.5 Helm values 接口

```yaml
computeOperator:
  enabled: true
  image: { registry, repository, tag, pullPolicy }
  leaderElection: { enabled: true, id: axisml-compute-operator.axisml.io }
  resources: { requests, limits }
  controllers:
    mljob:     { enabled: true }
    mlservice: { enabled: true }
```

`controllers.<name>.enabled=false` 时对应 controller 的 reconciler 不挂到 Manager，且 ClusterRole 中相关分段不渲染。

**Helm 模板清单**（`deploy/helm/axisml-system/templates/compute-operator/`）：

| 文件 | 用途 |
| --- | --- |
| `deployment.yaml` | compute-operator 镜像 |
| `serviceaccount.yaml` | 服务账号 |
| `clusterrole.yaml` / `clusterrolebinding.yaml` | §2.4 RBAC |
| `role.yaml` / `rolebinding.yaml` | leader election Lease |
| `servicemonitor.yaml` | `/metrics` |

---

## Part II — 通用契约

> 本部分集中两个 controller **共享**的边界与协议：与 Compute 的写路径、CRD 共同字段约束、Reconcile 行为、Pod 注入约定、dispatcher + handler 架构、Handler 接口契约。§4–§5 各 controller 章节引用本节而不重复。

## 3. 跨 controller 通用契约

### 3.1 与 Compute 的写路径契约

compute 服务采用 Operation Outbox + reconciler 异步下发 CR（详见 [compute.md](compute.md)）。compute-operator 暴露给 compute 的核心契约对两个 CRD 都成立：

- **`Create` 幂等**：相同 `metadata.name` 的二次 `Create()` 返回 409 `AlreadyExists`，且不引发副作用（不重建底层资源、不重置 status）。compute 收到 409 后会 `Get()` 现有 CR 并校验 label `axisml.io/{job-id|service-id}=<uuid>`；只有 label 一致才视为成功。
- **status 单向权威**：operator 只写 `<CR>.status`，compute 只写 `<CR>.metadata` / `<CR>.spec`；状态推进由 compute 侧 Informer 按 CR `status` 消费。**operator 不感知 compute 的 PG 表，也不向 compute PG 写入任何数据**——状态全部经由 CR `status` 回流。
- **配置补偿**：CR 被误删后由 compute 按 PG 快照判定是否重建（[compute.md](compute.md)）；operator 的 `Reconcile` 必须可在已存在的底层资源上幂等收敛——已存在的资源不重建，只对齐 spec 漂移。

### 3.2 CRD 共同约束

**metadata 由 compute 设置**：

- `metadata.name` ← compute 业务对象 name；DNS-1123 + ≤40 字符（compute API 层硬校验）。
- `metadata.namespace` ← compute 请求体里的 `namespace` 字段（裸字符串分区键，由调用方提供）。
- `metadata.labels["axisml.io/{job,service}-id"]` ← UUID，孤儿检测稳定锚点。
- `metadata.labels["axisml.io/quota"]` ← 透传给 Pod 用于审计 / 查询，取值与 `spec.scheduling.quota`（ElasticQuota CR 名）一致。

**status subresource 必启用**：`mljob-crd.yaml` / `mlservice-crd.yaml` 都必须声明 `subresources.status`。

**当前 CRD schema 现状**：两个 CRD 的 `spec` / `status` 暂用 `x-kubernetes-preserve-unknown-fields: true`；待行为稳定后再启用 OpenAPI schema 严格校验（各 controller 自身后续工作中提到的 enum / required 强校验）。

**phase 集合冻结**：MLJob 与 MLService 各自四态（见 §1 总览表）。新增 phase 必须经 CRD schema 与 Compute 双侧同步演进。

### 3.3 Reconcile 通用约束

下列约束对两个 controller 都生效：

- **不引入 finalizer**：MLJob / MLService 一律不挂 finalizer；级联清理依赖 `ownerReference`（CR → Handler 创建的底层资源）。
- **`Validate(spec)` 必须是纯函数**：不发起 K8s 调用，便于未来在 admission webhook 中复用。校验失败 → `status.phase=Failed`、`status.message` 写明违规项。
- **`Reconcile` 幂等**：多次调用相同 spec 不重建底层资源；只有语义字段变化才触发底层资源更新。
- **Status 写盘单一路径**：reconcile 末尾通过一次 patch 完成，避免半成品 status；dispatcher 通过 `status` subresource 做 JSON merge patch（或 server-side apply）+ `resourceVersion` 冲突重试；`conditions[]` 由 dispatcher 按 `type` 去重后整体写回。
- **MapStatus 纯函数**：状态推进只依赖 dispatcher 传入的快照，不发起 K8s 调用，便于单元测试与状态回放。
- **Handler 不直接写 `status`**：Handler 通过 `MapStatus` 的返回值与 `Reconcile` 的结构化结果影响 `status`；dispatcher 统一合并写盘。

### 3.4 Pod 注入约定

所有 MLJob 与 MLService Handler 派生的 Pod（含 KServe 派生的 inference Pod）必须满足以下注入约定，体现 [infra.md](infra.md) 的 Quota 全覆盖不变式：

| Pod 字段 / Label | 必填 | 取值 | 用途 |
| --- | --- | --- | --- |
| `spec.schedulerName` | 是 | `koord-scheduler` | 所有 AxisML workload Pod 一律走 koord-scheduler；不允许任何 backend 让 Pod 落到默认 kube-scheduler 上 |
| label `quota.scheduling.koordinator.sh/name` | 是 | `<spec.scheduling.quota>` | Koordinator 原生 quota 关联 label；ElasticQuota plugin 据此把该 Pod 计入 `status.used` |
| label `axisml.io/{job-id\|service-id}` | 是 | UUID | 反查 MLJob / MLService，与 CR 上同名 label 一致 |
| label `axisml.io/role` | 是 | role 名（`worker` / `master` / `predictor` / ...） | 区分多角色拓扑下的 Pod |
| label `axisml.io/quota` | 是 | `<spec.scheduling.quota>` | AxisML 自有审计 / 查询；与 `quota.scheduling.koordinator.sh/name` 取值相同（operator 不从 ElasticQuota 全名中拆解 bare quota name） |
| label `axisml.io/replica-index` | 否 | role 内 0-based 序号 | 副本身份天然稳定时建议透传：StatefulSet 的 `apps.kubernetes.io/pod-index`、Indexed Job 的 `batch.kubernetes.io/job-completion-index`；Deployment / 裸 Pod / KServe autoscaling 等无稳定身份场景一律省略 |

前 5 项必填，所有 Handler 一律遵守；缺失任一项视为契约违反，Handler 的 `Validate` 必须在创建前拦截。

**KServe 派生 Pod 的注入路径**：`(kserve, *)` Handler 不直接控制 podSpec，通过写入 `InferenceService.spec.predictor.schedulerName` + `spec.predictor.labels` 让 KServe 透传到派生 Pod 的 `spec.schedulerName` 与 `metadata.labels`（KServe `PredictorSpec` 内联 `corev1.PodSpec` 与 `ComponentExtensionSpec`，所以两者都是 `spec.predictor` 的直接字段）。

### 3.5 Dispatcher + Handler 架构

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

**Watch 拓扑**：Dispatcher 始终 watch 主 CR 队列；每个 Handler 启动时通过 `WatchTargets()` 声明自己关心的底层资源 GVK，由 dispatcher 统一建立 watch（controller-runtime `Watches()`），事件通过 `ownerReference` 反查回 CR 后再交给对应 Handler 的 `MapStatus`。

**注册方式**：编译期注册（每个 handler 包 `init()` → 全局 registry）；不引入运行时插件加载。

**未注册组合的兜底**：dispatcher 收到一条 `(backend, engine)` 无 handler 的 CR → 写 `status.phase=Failed`、`status.message="no handler for backend=X engine=Y"`，不创建任何底层资源。

### 3.6 Handler 接口契约

所有 Handler 必须实现以下方法（语言无关，描述责任）：

| 方法 | 责任 |
| --- | --- |
| `Key()` | 返回 `(backend, engine)` 元组，与 `spec.backend.{name, engine}` 对齐 |
| `Validate(spec)` | 校验通用字段 + `backend.config` + role 集合；纯函数 |
| `Reconcile(ctx, cr)` | 创建 / 更新底层资源；幂等；通用字段由 Handler 自己注入；处理控制信号；返回结构化结果 |
| `MapStatus(snapshot)` | 把 CR spec + 底层资源快照映射回统一 phase + 公共状态字段；纯函数 |
| `Cleanup(ctx, cr)` | 删除底层资源；一般依赖 ownerReference 自动级联 |
| `WatchTargets()` | 声明本 Handler 需要 watch 的底层资源 GVK 列表 |
| `RequiredRBAC()` | 声明本 Handler 需要的 ClusterRole 规则；启动时聚合到 operator ServiceAccount |

#### 控制信号义务

**MLJob Suspend**：每个 MLJob Handler 在自身章节显式声明"原生支持 / 兜底为 Cleanup"，dispatcher 不做静默选择——不支持原生 suspend 时必须显式调用 `Cleanup()`，避免半暂停半运行的中间态。所有路径完成底层动作后都必须返回 `suspendCompleted=true, reason=CancelRequested`——这是 dispatcher 写入 cancel 闭环推进信号的唯一来源；缺失会导致 compute PG 永远卡在 `Canceling`。`status.phase` 在非终态 suspend 期间维持 `Pending`。若底层资源已经终态，终态优先，Handler 返回终态状态映射而不是 suspend 完成结果。

**MLService Scale**：每个 MLService Handler 必须把 `roles[*].replicas` 透传为后端原生扩缩（`Deployment.spec.replicas` / `StatefulSet.spec.replicas` / `InferenceService.spec.predictor.{minReplicas, maxReplicas}`）；不支持原生扩缩的 backend 兜底为重建底层资源（应避免，作为最后手段）。

### 3.7 不变量

- `(backend, engine)` 元组未在 registry 注册 → CR 直接进入 `Failed`，message 写明缺失原因。
- Handler 不直接修改 ElasticQuota CR；ElasticQuota 由 tenant-operator 独占维护。
- operator 不向 compute PG 写入任何数据；状态全部经由 CR `status` + compute Informer 回流。

---

## Part III — Controller 详细设计

## 4. MLJob Controller

### 4.1 概述

MLJob controller 把 compute 下发的 `MLJob` CR 翻译为底层执行资源（Pod / PodGroup / 第三方 CR），并把执行状态回流到 `MLJob.status`。它使用 §3.5 的 dispatcher + handler 模式：CR 的 `spec.backend.{name, engine}` 二级元组路由到不同 Handler。

`MLJob` 为 namespaced CR：

| 字段 | 取值 |
| --- | --- |
| `apiVersion` | `axisml.io/v1alpha1` |
| `kind` | `MLJob` |
| `scope` | `Namespaced`（创建在 compute 请求指定的 namespace 下） |
| `shortNames` | `mlj` |

### 4.2 CRD 契约

#### 4.2.1 spec 设计取舍

把"角色拓扑"提升为一等公民。Job 的执行域用 `roles[]` 数组承载：

- 单角色 backend（如 `(native, job)`、`(native, podgroup)`）声明一个 role。
- 多角色 backend（如 PyTorchJob 的 master/worker、TFJob 的 chief/worker/ps/evaluator、MPIJob 的 launcher/worker）声明多个 role。
- role 名集合由各 Handler 在 §4.5 中约定，由 Handler 的 `Validate` 强制。

调度域的 `nodeSelector` / `tolerations` 沿用 K8s PodSpec 扁平惯例直接放在 `spec.scheduling` 下。

#### 4.2.2 spec 结构

```yaml
apiVersion: axisml.io/v1alpha1
kind: MLJob
spec:
  # ── 后端选择（创建后不可变；dispatcher 路由依据）─────────────────────
  backend:
    name: native              # 必填: native | kubeflow-trainer | custom
    engine: job               # 必填: 语义随 backend 而定（见 §4.5）
                              #   native:           job | podgroup
                              #   kubeflow-trainer: pytorchjob | tfjob | mpijob | …
                              #   custom:           任意名（由 backend.config 描述目标 GVK）
    config: {}                # 可选: 该 (backend, engine) 元组特有的 schemaless 配置

  # ── 调度域 ─────────────────────────────────────────────────────
  scheduling:
    quota: string             # 必填: ElasticQuota CR 名（对 operator 不透明，直接透传）
    priorityClass: string     # 可选
    nodeSelector: {}          # Compute 按 compute.md §6 合并 pool + unit 后注入
    tolerations: []

  # ── 执行域：roles 数组 ─────────────────────────────────────────
  roles:
    - name: worker            # role 标识；同一 MLJob 内唯一
      replicas: 1             # >= 0；为 0 时该角色禁用
      restartPolicy: OnFailure # OnFailure | Never
      template:
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
    suspend: false                  # 可选: cancel 信号
    activeDeadlineSeconds: int      # 可选: 硬超时
    ttlSecondsAfterFinished: int    # 可选: 终态后底层资源 GC
    backoffLimit: int               # 可选: 重试预算；具体语义由各 Handler 解释
```

#### 4.2.3 字段归属与不可变性

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

**默认值**：`spec.backend` 默认 `{name: native, engine: job}`；`backend.config` 默认 `{}`。

### 4.3 Status

```yaml
status:
  observedGeneration: int64
  phase: Pending | Running | Succeeded | Failed   # ← Compute 唯一消费的字段
  message: string                                  # Compute 透传到 jobs.message
  startedAt: timestamp
  finishedAt: timestamp
  conditions:                                      # Suspended 会被 Compute 消费为 cancel 推进信号；其余仅 UI 观测
    - type: Initialized | Scheduled | Suspended | Failed
      status: True | False | Unknown
      lastTransitionTime: timestamp
      reason: string                               # Suspended 时约定 reason=CancelRequested
      message: string
  roles:
    - name: string
      replicas: int
      activeReplicas: int
      readyReplicas: int
      succeededReplicas: int
      failedReplicas: int
```

**Compute phase 映射规则**（与 [compute.md](compute.md) 对齐）：

| MLJob status.phase | jobs.status | 终态 |
| --- | --- | --- |
| `Pending` | `Pending` | 否 |
| `Running` | `Running` | 否 |
| `Succeeded` | `Succeeded` | 是 |
| `Failed` | `Failed` | 是 |

**Cancel 推进信号**——`Cancelled` 与 `Deleted` 不由 operator 直接产出，但 cancel 路径有明确的链上信号：Handler 在收到 `spec.runPolicy.suspend=true` 并完成"暂停或清理底层资源"后，**必须向 dispatcher 返回 `suspendCompleted=true` 与 `reason=CancelRequested`**；dispatcher 统一合并写入 `status.conditions[type=Suspended,status=True,reason=CancelRequested]`，且在非终态时让 `status.phase` 维持在 `Pending`。Compute Informer 在 PG `status='Canceling'` 时把这个 condition 当作推进信号 → 写 `Cancelled` → 入队 `Delete()` 做 CR 资源回收。

**终态优先**：cancel 只面向仍处于 `Pending` / `Running` 的 Job。若底层资源已经进入 `Succeeded` / `Failed`，dispatcher 必须保留终态 phase 与 `finishedAt`，不能为了 cancel 信号把 `status.phase` 回退为 `Pending`。

### 4.4 Reconcile 事件路径

| 事件 | Dispatcher 行为 | Handler 行为 |
| --- | --- | --- |
| MLJob ADD | 路由到 Handler；调用 `Validate(spec)`，校验失败写 `status.phase=Failed` | `Reconcile` 创建底层资源，设置 `ownerReference: MLJob` |
| MLJob UPDATE（spec 变更） | 校验 `backend.{name, engine}` 不变；其余 spec 变化路由给 Handler | `Reconcile` 幂等更新 |
| MLJob `spec.runPolicy.suspend=true` | 路由；若当前或新映射出的 phase 已是 `Succeeded` / `Failed`，终态优先；否则在 Handler 返回 suspend 完成结果后合并写入 `Suspended=True,reason=CancelRequested`，`phase` 维持 `Pending` | 执行原生 suspend 或 `Cleanup()`；返回结构化 suspend 结果 |
| MLJob DELETE | 不阻断 | 一般依赖 ownerReference 级联清理 |
| 底层资源事件 | 通过 ownerReference 反查到 MLJob 后路由 | `MapStatus` 纯函数计算新 phase |

### 4.5 内置 Handler

每个 Handler 章节统一按以下小节组织：**底层资源 / `backend.config` / 通用字段映射 / Status 映射 / Suspend / RBAC**。

#### 4.5.1 `(native, job)`

底层用 K8s 原生 [`Job`](https://kubernetes.io/docs/concepts/workloads/controllers/job/)；适合不需要 gang scheduling 的单角色批处理场景。Pod 强制走 koord-scheduler 并通过 `quota.scheduling.koordinator.sh/name` label 计入 ElasticQuota；本 Handler **不**创建 PodGroup（gang 不适用）。

**前置依赖**：集群已安装 Koordinator（提供 koord-scheduler 与 ElasticQuota plugin）。本 Handler 仅需要 `jobs.batch` 的 CRUD。

**底层资源**：

- 必填且仅一个 role（`name=worker`）；`Validate` 拒绝多 role 提交。
- 每个 MLJob 创建一个 K8s `Job`，Pod 由 Job controller 派生，但 Pod 模板上设置 `schedulerName: koord-scheduler`。
- Job 设置 `ownerReference` 指向 MLJob。

**Pod label**：§3.4 列出的 5 项必填 label，外加 `axisml.io/replica-index=<0-based>`（仅在 `backend.config.completionMode=Indexed` 时透传 K8s 注入的 `batch.kubernetes.io/job-completion-index`；默认 NonIndexed 模式下省略）。

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

`startedAt` 取 `Job.status.startTime`；`finishedAt` 取 `Job.status.completionTime`。

**Suspend**：原生支持。`spec.runPolicy.suspend=true` → patch `Job.spec.suspend=true`（K8s 原生字段，自动驱逐运行中的 Pod 并停止派生新 Pod），随后返回 `suspendCompleted=true, reason=CancelRequested`。

**RBAC**：`jobs.batch` / `pods` / `events` 的 CRUD。

#### 4.5.2 `(native, podgroup)`

将 MLJob 翻译为 sigs.k8s.io scheduler-plugins `PodGroup` + 裸 Pod，借助 Koordinator gang plugin 实现"全员就位才启动"的单角色任务（如分布式训练的多 Worker 同步启动）。

**前置依赖**：集群已安装 Koordinator；本 Handler 需要 `podgroups.scheduling.sigs.k8s.io` 的 CRUD。

**底层资源**：

- 必填且仅一个 role（`name=worker`）。
- 每个 MLJob 创建一个 `PodGroup`（`scheduling.sigs.k8s.io/v1alpha1`），`spec.minMember ← roles[worker].replicas`。
- 按 `roles[worker].replicas` 创建对应 Worker 裸 Pod；通过 label `pod-group.scheduling.sigs.k8s.io=<podgroup-name>` 关联到 PodGroup。
- PodGroup / Pod 设置 `ownerReference` 指向 MLJob。

**Pod label**：§3.4 列出的 5 项必填 label，外加 `pod-group.scheduling.sigs.k8s.io=<podgroup-name>`。

**通用字段映射**：

| MLJob 字段 | Pod / PodGroup 落点 |
| --- | --- |
| `roles[worker].template.*` | Pod 主容器同名字段 |
| `roles[worker].replicas` | `PodGroup.spec.minMember` 与裸 Pod 数 |
| `spec.scheduling.*` | Pod 同名字段 |
| `spec.scheduling.quota` | Pod label `quota.scheduling.koordinator.sh/name`；PodGroup 不持有 quota 字段 |
| 调度器选择 | Pod `spec.schedulerName=koord-scheduler`（恒定） |

**Status 映射**：

| 原生状态 | MLJob phase |
| --- | --- |
| 所有 Pod `Pending` 或 PodGroup 排队中 | `Pending` |
| 至少一个 Pod 进入 `Running` | `Running` |
| 所有 Pod `Succeeded` | `Succeeded` |
| 任一 Pod `Failed`、PodGroup 调度不可达、超 `activeDeadlineSeconds` | `Failed` |

**Suspend**：原生支持。

1. patch `PodGroup.spec.minMember=0`。
2. 删除现存 Pod；后续 reconcile 看到 `spec.runPolicy.suspend=true` 后不再重建 Pod。
3. 返回 `suspendCompleted=true, reason=CancelRequested`。

**顺序约束**：必须先 patch minMember=0、再删 Pod，否则 koord-scheduler 的 gang plugin 可能立即把刚被删除的 Pod 重新调度。

**RBAC**：`pods` / `podgroups.scheduling.sigs.k8s.io` / `events` 的 CRUD。

#### 4.5.3 `(kubeflow-trainer, *)`

将 MLJob 翻译为 Kubeflow Trainer 的多角色训练 CR。本节锁三件事：路由元组、role 集合约定、Status / Suspend 协议骨架。`backend.config` 详细 schema 由独立设计文档落地（见 §4.6）。

**前置依赖**：集群已安装 kubeflow training-operator；本 Handler 需要对应 CR 的 CRUD（`pytorchjobs.kubeflow.org` / `tfjobs.kubeflow.org` / `mpijobs.kubeflow.org` …）。

**Role 集合约定**（按 engine 分）：

| engine | role 集合 | 备注 |
| --- | --- | --- |
| `pytorchjob` | `master`（replicas=1，可省略默认）+ `worker`（replicas≥1），可选 `elasticAgent` | 主线落地 engine |
| `tfjob` | `chief` / `worker` / `ps` / `evaluator`，任一可省略，replicas=0 表示禁用 | 同构扩展 |
| `mpijob` | `launcher`（replicas=1）+ `worker`（replicas≥1） | 同构扩展 |

**通用字段映射骨架**：

- `roles[*].template.*` → 对应 CR 的 `*ReplicaSpecs.<Role>.template`。
- 各 replica 模板的 `template.spec.schedulerName` 必须设为 `koord-scheduler`；`template.metadata.labels` 必须注入 §3.4 列出的 5 项必填 label。
- 多角色 gang 由本 Handler 一并创建 PodGroup CR（`spec.minMember ← sum(replicas)`）。
- `spec.scheduling.*` → 各 replica 模板内的同名字段。
- `spec.runPolicy.activeDeadlineSeconds` / `backoffLimit` → 后端 CR `spec.runPolicy` 同名字段。

**Status 映射**：

| 后端 condition | MLJob phase |
| --- | --- |
| `Created` / `Restarting` | `Pending` |
| `Running` | `Running` |
| `Succeeded` | `Succeeded` |
| `Failed` | `Failed` |

**Suspend**：优先使用后端原生 `spec.runPolicy.suspend=true`；目标版本不支持时 fallback 为 `Cleanup()`。

#### 4.5.4 `(custom, *)`

为外部接入预留的一等公民。用户在 `backend.config` 中以 schemaless 方式描述目标 GVK 与字段映射，由 custom Handler 通过 unstructured client 创建并跟踪。

**仍受 §3.4 Pod 注入约定约束**：custom Handler 必须保证最终落地的 Pod 模板上有 `schedulerName: koord-scheduler` 与 `quota.scheduling.koordinator.sh/name` label，否则 Validate 直接拒绝。

> 完整 schemaless `config` schema、JSONPath fieldMappings / statusMappings、unstructured 操作约定由独立设计文档落地（见 §4.6）。

### 4.6 后续工作

- `(native, job)` Handler 的 Indexed Job 模式与 `podFailurePolicy` 直通策略细节。
- `(native, podgroup)` Handler 的 PodGroup `minResources` 与 elastic gang 演进。
- `(kubeflow-trainer, *)` 各 engine 的完整字段映射 / 状态映射 / `backend.config` schema。
- `(custom, *)` Handler 的 `config` 完整 schema 与 unstructured 操作约定。
- Admission webhook：`spec.backend.{name, engine}` 不可变约束、`backend.config` 按 Handler 自带 schema 的统一校验。
- Handler chart 的 values 控制（启用 / 禁用 backend，对应 RBAC 与 watch 的开关）。
- CRD 严格 schema（启用 OpenAPI 校验）。

## 5. MLService Controller

### 5.1 概述

MLService controller 把 compute 下发的 `MLService` CR 翻译为底层在线推理资源（Deployment + Service / KServe `InferenceService` / 自定义 GVK），并把执行状态回流到 `MLService.status`。它使用 §3.5 的 dispatcher + handler 模式，与 MLJob 同构。

`MLService` 为 namespaced CR：

| 字段 | 取值 |
| --- | --- |
| `apiVersion` | `axisml.io/v1alpha1` |
| `kind` | `MLService` |
| `scope` | `Namespaced`（创建在 compute 请求指定的 namespace 下） |
| `shortNames` | `mls` |

### 5.2 CRD 契约

#### 5.2.1 spec 设计取舍

把"角色拓扑"提升为一等公民，与 MLJob §4.2.1 同源。Service 的执行域用 `roles[]` 数组承载：

- 单角色 backend（如 `(native, deployment)` / `(native, statefulset)`）声明一个 role（约定 `name=predictor`）。
- 多角色 backend（如 KServe `InferenceService` 的 `predictor` / `transformer` / `explainer`）声明多个 role。

**Service 不引入独立 koordinator backend**：service 是常驻 + 弹性扩缩 workload，不应默认获得"所有副本同时调度"的 gang 语义，故 `native` 直接走 K8s 原生 Deployment / StatefulSet，不引入额外的 backend 维度。但所有 native Service 与 KServe 派生的 Pod 仍强制走 koord-scheduler 并消耗 ElasticQuota（详见 §3.4）。

**与 MLJob 的差异点**：

- 顶层 `modelRef`：service 一等字段，指向 Artifacts model version；Handler 据此把模型工件解析为容器侧的位置（环境变量 / volume mount / KServe `storageUri` 等）。
- `roles[*].template.ports[]`：与 K8s `PodSpec.containers[].ports` 同源约定。每个 role 是一个独立的 Deployment / StatefulSet（或 InferenceService 内的 component），各自的容器端口属于该 role 自身。Handler 据此为每个 role 派生一个 K8s Service（targetPort=containerPort）。
- 顶层 `route`：可选；与 Gateway API `HTTPRoute` 同源命名。当 `enabled=true` 时由 Handler 创建 namespaced `HTTPRoute`（搭配 Envoy Gateway 的 `SecurityPolicy` / `BackendTrafficPolicy`）实现自助外部入口；详见 §5.5。`(kserve, *)` Handler 自带 Route 机制，不接受 `route.enabled=true`。
- `runPolicy` 字段集合不同：service 是常驻 workload，**没有** `suspend` / `activeDeadlineSeconds` / `ttlSecondsAfterFinished` / `backoffLimit`；改为 `progressDeadlineSeconds`（rollout 进度超时，与 K8s Deployment 同名字段语义一致）。

#### 5.2.2 spec 结构

```yaml
apiVersion: axisml.io/v1alpha1
kind: MLService
spec:
  # ── 后端选择（创建后不可变；dispatcher 路由依据）─────────────────────
  backend:
    name: native              # 必填: native | kserve | custom
    engine: deployment        # 必填: 语义随 backend 而定（见 §5.6）
                              #   native: deployment | statefulset
                              #   kserve: inference | llminference
                              #   custom: 任意名（由 backend.config 描述目标 GVK）
    config: {}                # 可选

  # ── 调度域 ─────────────────────────────────────────────────────
  scheduling:
    quota: string             # 必填: ElasticQuota CR 名（对 operator 不透明）
    priorityClass: string
    nodeSelector: {}
    tolerations: []

  # ── 模型引用（service 特有，指向 Artifacts）─────────────────────────
  modelRef:
    name: string
    version: string

  # ── 执行域：roles 数组 ─────────────────────────────────────────
  roles:
    - name: predictor
      replicas: 1
      template:
        image: string
        imagePullPolicy: IfNotPresent
        command: []
        args: []
        env: []
        envFrom: []
        workingDir: string
        ports:
          - name: http
            containerPort: 8080
            protocol: TCP
        resources:
          requests: {}
          limits: {}

  # ── 生命周期 ────────────────────────────────────────────────────
  runPolicy:
    progressDeadlineSeconds: int

  # ── 对外路由（可选；默认仅 ClusterIP；与 Gateway API HTTPRoute 同源）────
  route:
    enabled: false
    targetRole: string
    portName: string
    hostname: string
    path: string
    auth:
      type: none | jwt | apiKey
      jwt: { issuer, jwksUri }
      apiKey: { secretRef: { name } }
    rateLimit:
      requestsPerSecond: int
      burst: int
    timeout: string
```

#### 5.2.3 字段归属与不可变性

| 字段路径 | 写入方 | 创建后可变？ |
| --- | --- | --- |
| `metadata.name` / `namespace` / `labels[axisml.io/*]` | Compute | 否 |
| `spec.backend.name` / `spec.backend.engine` | Compute（默认 `{native, deployment}`） | **否** |
| `spec.backend.config` | Compute（默认 `{}`） | 否（仅 `roles[*].replicas` 通过 `/scale` 变更） |
| `spec.scheduling.*` | Compute | 否 |
| `spec.modelRef` | 用户提交 | 否（更换模型版本走重建） |
| `spec.roles[*].name` / `template.*`（含 `ports[]`，除 resources） | 用户提交 | 否 |
| `spec.roles[*].template.resources` | Compute（注入 ResourceUnit） | 否 |
| `spec.roles[*].replicas` | API（`/scale` 触发） | **是** |
| `spec.runPolicy.progressDeadlineSeconds` | 用户提交 | 否 |
| `spec.route`（整块） | 用户提交 | 否 |

**`spec.route` 与 backend 的兼容性**：`(kserve, *)` Handler 在 `Validate` 中拒绝 `spec.route.enabled=true`；`(native, *)` 与 `(custom, *)` 接受。

### 5.3 Status

```yaml
status:
  observedGeneration: int64
  phase: Pending | Ready | Degraded | Failed
  message: string
  endpoint: string
  readyReplicas: int
  conditions:
    - type: Initialized | Available | Progressing | Failed
      status: True | False | Unknown
      lastTransitionTime: timestamp
      reason: string
      message: string
  roles:
    - name: string
      replicas: int
      readyReplicas: int
```

**`endpoint` 取值规则**：

- native/custom 且 `route.enabled=false`（默认） → K8s Service DNS（`<svc>.<ns>.svc.cluster.local:<port>`）。
- native/custom 且 `route.enabled=true` → AxisML Gateway 外部 URL。
- kserve backend → KServe 自带 route/status.url 暴露的 URL（不接受 `spec.route.enabled=true`）。

**Compute phase 映射规则**：

| MLService status.phase | services.status | 终态 |
| --- | --- | --- |
| `Pending` | `Pending` | 否 |
| `Ready` | `Ready` | 否 |
| `Degraded` | `Degraded` | 否，可恢复 |
| `Failed` | `Failed` | 否，可恢复（自愈） |

`Pending / Ready / Degraded / Failed` 均为非终态——operator 自愈后 Informer 回流可让 `Failed → Degraded → Ready` 自然恢复；只有 `Deleted` 是 Service 的最终终态，由 Compute Informer 在观察到 CR DELETE 事件后基于 PG 当前 `status` 推导。

### 5.4 Reconcile 事件路径

| 事件 | Dispatcher 行为 | Handler 行为 |
| --- | --- | --- |
| MLService ADD | 路由到 Handler；调用 `Validate(spec)`，校验失败写 `status.phase=Failed` | `Reconcile` 创建底层资源，设置 `ownerReference: MLService` |
| MLService UPDATE（仅 `roles[*].replicas` 变更，来自 `/scale`） | 路由 | `Reconcile` 透传为后端资源副本调整；不重建 Pod |
| MLService UPDATE（其他 spec 字段变更） | 校验 `backend.{name, engine}` 不变；其他字段变更属于约束违反，写 `status.message` 拒绝 | 不动 |
| MLService DELETE | 不阻断 | 一般依赖 ownerReference 级联清理 |
| 底层资源事件 | 通过 ownerReference 反查到 MLService 后路由 | `MapStatus` 基于快照纯函数计算新 phase |

### 5.5 spec.route 派生资源

当 `route.enabled=true` 时，Handler 在 CR 所在 namespace 内创建 / 更新以下资源（统一打 `axisml.io/service-id` label，`ownerReference: MLService` 级联清理，不引入 finalizer）：

- `HTTPRoute`（`gateway.networking.k8s.io/v1`）：`parentRefs` 指向 `axisml-gateway`（跨 namespace 引用通过 `ReferenceGrant` 授权，由 infra chart 准备），`backendRefs` 指向 `route.targetRole` 对应的 K8s Service。
- `SecurityPolicy`（`gateway.envoyproxy.io/v1alpha1`）：仅当 `auth.type != none` 时创建。
- `BackendTrafficPolicy`（`gateway.envoyproxy.io/v1alpha1`）：仅当 `rateLimit` 或 `timeout` 非空时创建。

### 5.6 内置 Handler

每个 Handler 章节统一按以下小节组织：**底层资源 / `backend.config` / 通用字段映射 / Status 映射 / Scale / RBAC**。

#### 5.6.1 `(native, deployment)`

底层用 K8s 原生 Deployment + Service。所有 Pod 走 koord-scheduler 并通过 Pod label 计入 ElasticQuota，**不**创建 PodGroup。

**底层资源**：

- 必填且仅一个 role（`name=predictor`）；`Validate` 拒绝多 role 提交或其他 role 名。
- 每个 MLService 创建一个 K8s `Deployment` 与一个 K8s `Service`（`targetPort=containerPort`）。
- Deployment Pod 模板上设置 `schedulerName: koord-scheduler`，并打 §3.4 列出的 5 项必填 label。
- 当 `spec.route.enabled=true` 时追加 HTTPRoute + 可选的 SecurityPolicy / BackendTrafficPolicy。

**通用字段映射**：

| MLService 字段 | Deployment / Service / 派生路由资源落点 |
| --- | --- |
| `roles[predictor].template.image` 等 | Deployment Pod 主容器同名字段 |
| `roles[predictor].template.ports[]` | Deployment Pod 主容器 `ports` + K8s Service `spec.ports` |
| `roles[predictor].replicas` | `Deployment.spec.replicas` |
| `spec.scheduling.quota` | Pod label `quota.scheduling.koordinator.sh/name`；不创建 PodGroup |
| `spec.modelRef` | Artifacts client 解析为模型工件 URI，注入为环境变量 `AXISML_MODEL_URI` |
| `spec.route.targetRole` | 选取 HTTPRoute `backendRefs.name` |
| `spec.route.portName` | HTTPRoute `backendRefs.port` |
| `spec.route.hostname` / `path` | `HTTPRoute.spec.hostnames` / `rules[].matches[].path.value` |
| `spec.route.auth` | `SecurityPolicy.spec.{jwt | apiKeyAuth}` |
| `spec.route.rateLimit` / `timeout` | `BackendTrafficPolicy.spec.rateLimit` / `timeout` |
| `spec.runPolicy.progressDeadlineSeconds` | `Deployment.spec.progressDeadlineSeconds` |

**Status 映射**：

| 条件 | MLService phase |
| --- | --- |
| `desired_replicas == 0` | `Pending` |
| `ready_replicas == 0 && desired_replicas > 0` 且 rollout 仍在推进中 | `Pending` |
| `ready_replicas == desired_replicas && desired_replicas > 0` | `Ready` |
| `0 < ready_replicas < desired_replicas` | `Degraded` |
| 超过 `progressDeadlineSeconds` 或出现 `ReplicaFailure` | `Failed` |

**Scale**：patch `Deployment.spec.replicas`；不重建 Pod。

**RBAC**：`deployments.apps` / `services` / `pods` / `events` 的 CRUD；`spec.route` 派生资源对应的 Gateway / EnvoyProxy CRD 的 CRUD；`secrets` 的 `get / list / watch`（仅当 `route.auth.type=apiKey` 引用 Secret 时）。

#### 5.6.2 `(native, statefulset)`

为有状态推理（在线 KV cache、模型分片、节点身份固定的副本）预留。底层用 K8s `StatefulSet` + headless Service，副本身份稳定；其余约束沿用 §5.6.1。

**`backend.config` 关键字段**：

```yaml
config:
  podManagementPolicy: OrderedReady | Parallel   # 默认 OrderedReady
  serviceName: string                             # headless Service 名；不填默认 = MLService 名
```

> `volumeClaimTemplates` / `updateStrategy.{type, partition}` 等存储与灰度更新维度由独立设计文档落地，见 §5.7 后续工作。

**通用字段映射**：与 §5.6.1 相同，`roles[predictor].replicas` 落到 `StatefulSet.spec.replicas`；StatefulSet Pod 副本身份稳定，Handler 透传 K8s 注入的 `apps.kubernetes.io/pod-index` 为 `axisml.io/replica-index`。

**Status 映射**：从 `StatefulSet.status` 推导，规则与 §5.6.1 同构。

**Scale**：patch `StatefulSet.spec.replicas`。

**RBAC**：`statefulsets.apps` / `services` / `pods` / `events` 的 CRUD。

#### 5.6.3 `(kserve, inference)`

将 MLService 翻译为 KServe [`InferenceService`](https://kserve.github.io/website/) CR（`serving.kserve.io/v1beta1`）。这是 KServe 通用 ML 服务路径——predictor 内的具体 runtime（NVIDIA Triton / [vLLM](https://docs.vllm.ai/) / TF Serving / TorchServe / sklearn / huggingface 等）由 `backend.config.runtime` 选择。

**前置依赖**：集群已安装 KServe，且版本支持 `InferenceService.spec.predictor.schedulerName` 与 `spec.predictor.labels` 透传到派生 Pod。

**Role 集合约定**：当前仅开放 `predictor`（replicas≥0）；扩展角色 `transformer` / `explainer` 的接入节奏见 §5.7。

**`backend.config` 关键字段**：

```yaml
config:
  runtime: triton | vllm | tfserving | torchserve | sklearn | huggingface | <自定义 ServingRuntime 名>
  predictor:
    minReplicas: int              # 默认 = roles[predictor].replicas
    maxReplicas: int              # 自动扩缩上限
    scaleToZero: bool
    protocolVersion: v1 | v2
  storageUri: string              # 模型工件位置
  containerOverrides: {}

  triton:    { modelControlMode, modelRepository }
  vllm:      { model, dtype, tensorParallelSize, pipelineParallelSize, maxModelLen, maxNumBatchedTokens, enablePrefixCaching, quantization, extraArgs }
  huggingface: { task, modelId }
  torchserve: { modelStore }
```

**通用字段映射**：

- `roles[predictor].template.image` → predictor 容器（不填时由 `config.runtime` 选定的 ServingRuntime 提供默认镜像）。
- `roles[predictor].replicas` → 写入 `predictor.minReplicas`；若未设置 `config.predictor.maxReplicas`，则同时写入 `maxReplicas`。
- `spec.modelRef` → 通过 Artifacts 解析为 `predictor.storageUri`（runtime=triton 时也可解析为 `triton.modelRepository`；runtime=vllm 时优先解析为 `vllm.model`，缺失时回退到 `storageUri`）。
- `spec.scheduling.quota` → 写入 `InferenceService.spec.predictor.schedulerName=koord-scheduler` 与 `spec.predictor.labels` 中的 `quota.scheduling.koordinator.sh/name` + `axisml.io/quota`。
- `spec.route` → **不支持**；Handler 在 `Validate` 中拒绝 `spec.route.enabled=true`。

**runtime 专属约束**（由 `Validate` 强制）：

- `runtime=vllm`：`roles[predictor].template.resources.requests["nvidia.com/gpu"]` 必须等于 `config.vllm.tensorParallelSize × pipelineParallelSize`。
- `runtime=huggingface`：`config.huggingface.task` 必填。

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

#### 5.6.4 `(kserve, llminference)`

> KServe LLM API 的 GVK / CRD 字段路径仍在演进，最终以引入版本为准。本节先锁两件事——路由元组与 role 命名约定。详细 schema 待 KServe LLM API GA 后在 §5.7 单独成文。

将 MLService 翻译为 KServe LLM 原生 CR `LLMInferenceService`（占位命名）。该 engine 承载 LLM 在线服务的 **PD 分离（disaggregated serving）**：prefill 与 decode 拆成独立角色独立扩缩，搭配 router 角色做请求分发与 KV cache 协调。

**Role 集合约定**：

- `prefill`：长上下文处理（compute-bound）；replicas≥1。
- `decode`：token 生成（memory-bound）；replicas≥1。
- `router`：请求入口与 KV cache 协调；replicas≥1；承载 KServe LLM API 自带的对外入口。

`Validate` 强制：role 名属于上述集合；至少存在 `prefill` 与 `decode`。与 §5.6.3 一样拒绝 `spec.route.enabled=true`。

#### 5.6.5 `(custom, *)`

为外部接入预留的一等公民。用户在 `backend.config` 中以 schemaless 方式描述目标 GVK 与字段映射，由 custom Handler 通过 unstructured client 创建并跟踪。

**仍受 §3.4 Pod 注入约定约束**：custom Handler 必须保证最终落地的 Pod 模板上有 `schedulerName: koord-scheduler` 与 `quota.scheduling.koordinator.sh/name` label，否则 Validate 直接拒绝。

> 完整 schemaless `config` schema、JSONPath fieldMappings / statusMappings / endpointPath 由独立设计文档落地（见 §5.7）。

### 5.7 后续工作

- `(native, statefulset)` Handler 的 `volumeClaimTemplates` / `updateStrategy` 灰度更新、pod-index 寻址细节。
- 多 role 接入的具体 Handler 落地：
  - `(kserve, inference)` 的 `transformer` / `explainer`。
  - `(kserve, llminference)`：vLLM disaggregated / llm-d / NVIDIA Dynamo 等场景下的 KV cache 传输契约（nixl / mooncake）、parallelism schema、autoscaler 接入。
- KServe scale-to-zero 与 Compute quota 的精细交互模型。
- `(custom, *)` Handler 的 `config` 完整 schema 与 unstructured 操作约定。
- 多 role 独立扩缩容的 `/scale` API 扩展。
- `spec.route` 可热更新路径。
- `spec.route` 与 KServe 自带 Route 的统一化。
- Admission webhook：`spec.backend.{name, engine}` 不可变约束、`backend.config` 按 Handler 自带 schema 的统一校验。
- Handler chart 的 values 控制。
- CRD 严格 schema。

---

## Part IV — 实施与验证

## 6. 实现路径

### 6.1 阶段总览

```
┌──────────────────────────────────────────────────────────────┐
│ MVP                                                           │
│   单 Pod / 双 Reconciler / 两个 Job backend / 两个 Service   │
│   backend / integration                                       │
│   ↓                                                           │
│ 功能完善                                                      │
│   补齐主流 Handler、外部入口策略、admission webhook、严格      │
│   CRD schema                                                  │
│   ↓                                                           │
│ 未来规划                                                      │
│   custom backend、KServe LLM API、运行时插件                   │
└──────────────────────────────────────────────────────────────┘
```

### 6.2 阶段一：MVP

| 模块 | 范围 | 完成信号 |
| --- | --- | --- |
| Operator binary | 单 Manager 承载双 Reconciler、`--enable-*` flag、leader election | 单 Pod 启动后两个 controller 同时 Ready |
| MLJob dispatcher | `(backend, engine)` 路由、`Validate` 拒绝未注册元组、Suspend cancel 推进信号 | integration 覆盖：未知 backend 直接 `Failed`；suspend 后 condition 与 phase 变化符合 §4.3 |
| MLJob `(native, job)` | K8s Job + koord-scheduler、quota label 注入、原生 `Job.spec.suspend` cancel | integration 覆盖 happy path + suspend cancel |
| MLJob `(native, podgroup)` | scheduler-plugins PodGroup + 裸 Pod、`minMember=0` → 删 Pod 的暂停顺序 | integration 覆盖 happy path + suspend shutdown |
| MLService dispatcher | 同上 + `/scale` 透传 | integration 覆盖：未知 backend 直接 `Failed`；scale 修改 `roles[0].replicas` 触发后端副本调整 |
| MLService `(native, deployment)` | Deployment + ClusterIP Service + 基础 HTTPRoute | integration 覆盖 happy path、route 启用 / 禁用、scale、字段不可变性 |
| MLService `(native, statefulset)` | StatefulSet + headless Service、`apps.kubernetes.io/pod-index` 透传 | integration 覆盖 happy path、scale、字段不可变性 |
| CRD | 两个 CRD 用 `x-kubernetes-preserve-unknown-fields: true` + `subresources.status` | helm install / upgrade 通过 |
| 测试 | integration 覆盖两 controller 的 happy path + suspend + immutability | `make compute-operator-integration` 通过 |

### 6.3 阶段二：功能完善

1. **MLService `route.auth` + `route.rateLimit / timeout` 派生资源**
   - 完成信号：integration 覆盖 `jwt` / `apiKey` / 限流 三条派生资源路径。
2. **MLJob `(kubeflow-trainer, pytorchjob)` 主线**
   - 完成信号：handler 注册、`master + worker` role 校验、Status condition 映射到四态、原生 suspend + `Cleanup()` fallback；integration 覆盖 happy path 与 suspend。
3. **MLService `(kserve, inference)` 主线（vllm / triton 优先）**
   - 完成信号：handler 写 `InferenceService` 时强制注入 `schedulerName: koord-scheduler` + quota label、`status.url` 回流到 `endpoint`、scale 路径 patch `predictor.{minReplicas, maxReplicas}`；integration 覆盖 vllm + triton 两条 runtime 校验分支。
4. **Admission webhook 上线**
   - 完成信号：webhook server 部署、cert-manager 颁证；`spec.backend.{name, engine}` 不可变 / `backend.config` 按 Handler schema 校验各 1 条 integration test 通过。
5. **严格 CRD OpenAPI schema**
   - 完成信号：两个 CRD 移除 `preserve-unknown-fields`；integration 覆盖"非法 enum 值被 apiserver 直接拒绝"。

### 6.4 阶段三：未来规划

参见 §4.6 / §5.7 后续工作清单。

### 6.5 跨阶段验证策略

| 阶段 | 主测层 | 工具 |
| --- | --- | --- |
| MVP | integration | `make compute-operator-integration` |
| 功能完善 | integration 扩展（envtest + testcontainers） | `make integration-test` |
| 未来规划 | 单独写 RFC 设计文档 → integration 先行 | 同上 |

## 7. 测试

integration 在 `components/compute-operator/test/integration/` 单一 Go module 中，单一 `TestMain` 把两个 reconciler 注册到同一个 envtest manager。CRDPaths 是 `deploy/helm/axisml-system/crds/{mljob,mlservice}-crd.yaml` 与 `test/crds/external/`（vendored PodGroup / HTTPRoute）的并集。

仓库当前不维护 minikube 驱动的 e2e 层；端到端验证靠 integration（envtest）覆盖。

## 8. 相关引用

- [docs/system_design/overview.md](overview.md) 概述了 compute-operator 在控制平面里的位置。
- [docs/system_design/compute.md](compute.md) 描述 compute 服务与 operator 之间的 CR 写路径与状态回流。
- [docs/system_design/tenant-operator.md](tenant-operator.md) 是 compute-operator 的兄弟 operator，承载 Tenant / Quota / Namespace 资源派生。
- [docs/system_design/infra.md](infra.md) 给出 koord-scheduler / ElasticQuota / Gateway API 等基础设施依赖契约。
