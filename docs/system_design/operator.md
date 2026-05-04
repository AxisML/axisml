# axisml-operator 详细设计

axisml-operator 是 AxisML 控制平面里**唯一**的 Kubernetes operator 二进制，由一个 Manager 同时承载三个 controller：

| Controller | CRD（`axisml.io/v1alpha1`） | Scope | 详细设计章节 |
| --- | --- | --- | --- |
| Tenant | `Tenant` | Cluster-scoped | [§4 Tenant Controller](#4-tenant-controller) |
| MLJob | `MLJob` | Namespaced | [§5 MLJob Controller](#5-mljob-controller) |
| MLService | `MLService` | Namespaced | [§6 MLService Controller](#6-mlservice-controller) |

§1–§3 覆盖**跨 controller 的合并设计**与**作为单一 Deployment 的运维契约**；§4–§6 是各 controller 自身的 CRD 契约、字段不可变性、状态机、子资源管理等深度内容。

## 1. 合并动机

历史上 tenant / mljob / mlservice 是三个独立的 Go 模块、独立的 Deployment、独立的 ServiceAccount/ClusterRole。三者长期 lock-step：共用 image tag（`components/operators/Dockerfile` + `entrypoint.sh` 已经把它们打成一个镜像，靠 argv[0] 分派）、共用 controller-runtime / k8s.io 版本、共用 RBAC 习惯。把它们折叠进同一个 Manager 后：

- 三个 Deployment / SA / ClusterRole / Lease 缩减为一份；
- 三个 Go module（`go.mod`）+ 三个 envtest module 合并成一个 production module + 一个 envtest module；
- CI lint matrix 6 → 2、envtest matrix 3 → 1；
- 升级路径与运维诊断只看一个 Pod 的日志即可。

代价：**单个 Deployment 失去了按 CRD 独立 rollout 的能力**。这在过去三个 operator 始终同 image tag 同时升级的现实下是可接受的；如果未来某个 controller 需要独立 rollout，可以利用下文 §3.3 的 `--enable-*` 开关，把同一镜像以不同 flag 启动多个 Deployment。

## 2. 架构总览

```
┌──────────────────── axisml-operator (one Pod, leader-elected) ────────────────────┐
│                                                                                    │
│  ctrl.Manager (scheme: clientgoscheme + axisml + scheduling.sigs.k8s.io +          │
│                gateway.networking.k8s.io)                                          │
│  Lease: axisml-operator.axisml.io                                                  │
│                                                                                    │
│  ┌──────────────────┐  ┌────────────────────────┐  ┌───────────────────────────┐   │
│  │ TenantReconciler │  │ MLJob: dispatcher +    │  │ MLService: dispatcher +   │   │
│  │ (single, no      │  │ Registry → handlers/   │  │ handler.Build() →         │   │
│  │  dispatcher)     │  │ {nativejob,podgroup}   │  │ handler/{nativedeploy.}   │   │
│  └──────────────────┘  └────────────────────────┘  └───────────────────────────┘   │
│        │                       │                            │                      │
│        ▼                       ▼                            ▼                      │
│   Namespace,                Job, Pod, PodGroup        Deployment, Service,         │
│   ElasticQuota,             (koord-scheduler          HTTPRoute (Gateway API)      │
│   Secret/CM/SA/             gang scheduling)                                       │
│   Role/RoleBinding                                                                 │
└────────────────────────────────────────────────────────────────────────────────────┘
```

Tenant 走**单 reconciler**直接调度（无 dispatcher）；MLJob 与 MLService 共用 **dispatcher + handler** 模式：CR 的 `spec.backend.{name, engine}` 元组路由到注册过的 Handler，handler 渲染目标 GVK 并把状态回流到 CR.status。这两个 controller 的具体 dispatch 表与默认后端见 [§5.7 MLJob 内置 Handler](#57-内置-handler) 与 [§6.7 MLService 内置 Handler](#67-内置-handler)。

## 3. 合并后的运行时契约

### 3.1 Scheme 注册

```go
clientgoscheme.AddToScheme(scheme)           // core, apps, rbac, batch, coordination
schedulingv1alpha1.AddToScheme(scheme)       // ElasticQuota + PodGroup（Koordinator vendored）
gwapiv1.Install(scheme)                      // HTTPRoute
tenant_v1alpha1.AddToScheme(scheme)          // Tenant
mljob_v1alpha1.AddToScheme(scheme)           // MLJob
mlservice_v1alpha1.AddToScheme(scheme)       // MLService
mlservicehandler.RegisterStubs()             // MLService 占位 handler 注册
```

三个 CRD 共享 group `axisml.io/v1alpha1`，但 Go 类型分别定义在 `components/operator/api/{tenant,mljob,mlservice}/v1alpha1/` 三个子包里——避免 `Phase`、`RoleSpec`、`LabelQuota` 等同名常量在同一包内冲突，同时仍然让一个 Manager 通过分别 `AddToScheme` 把三种 Kind 全注册进去。

### 3.2 Cache 选择性过滤

Tenant 的子资源（Secret / ConfigMap / ServiceAccount / Role / RoleBinding / ElasticQuota）在生产中受 `managed-by=tenant-operator` label 过滤，避免缓存全集群 Secret。**关键约束：这条过滤必须按对象类型挂在 `cache.Options.ByObject` 上**，不能升格成 `cache.Options.DefaultLabelSelector`——否则 MLJob 的 `Job/Pod/PodGroup` informer 与 MLService 的 `Deployment/HTTPRoute` informer 会被同样的 label 过滤掉，导致丢事件。

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

### 3.3 Flag 集合

| Flag | 默认 | 说明 |
| --- | --- | --- |
| `--metrics-bind-address` | `:8080` | Prometheus metrics |
| `--health-probe-bind-address` | `:8081` | `/healthz`, `/readyz` |
| `--leader-elect` | `true` | leader election |
| `--leader-election-id` | `axisml-operator.axisml.io` | Lease 名 |
| `--enable-tenant` | `true` | 启用 Tenant controller |
| `--enable-mljob` | `true` | 启用 MLJob controller |
| `--enable-mlservice` | `true` | 启用 MLService controller |
| `--enable-native-job` | `true` | MLJob: 注册 (native, job) handler |
| `--enable-native-podgroup` | `true` | MLJob: 注册 (native, podgroup) handler |

Pod 上还会注入两个环境变量供 Tenant 子模块消费：`RESYNC_PERIOD`（默认 `10m`）、`NAMESPACE_DENYLIST`（逗号分隔列表，默认值见 Helm `values.yaml`）。

### 3.4 RBAC

合并后只保留**一个** ClusterRole（`<release>-operator`），rules 是三个 controller 所需权限的并集，按 controller 分段；段头按 `--enable-*` Helm value 条件渲染（见 `deploy/helm/axisml-system/templates/operator/clusterrole.yaml`）。leader election Lease 在部署 namespace 通过 Role + RoleBinding 授权（不放进 cluster-scoped 角色）。

### 3.5 Helm values 接口

```yaml
operator:
  enabled: true
  image: { registry, repository, tag, pullPolicy }
  leaderElection: { enabled, id }
  resources: { requests, limits }
  controllers:
    tenant:    { enabled, resyncPeriod, namespaceDenylist }
    mljob:     { enabled, handlers: { nativeJob, nativePodGroup } }
    mlservice: { enabled }
```

旧的 `operators.{tenantOperator, mljobOperator, mlserviceOperator}` 三段配置在升级到合并版本时**必须**重写为 `operator.controllers.*`——这是一次破坏性 values 变更，Deployment / SA / ClusterRole 也都重命名了，因此 `helm upgrade` 时旧资源会被删除、新资源会被创建。`helm upgrade` 不会清理三个旧 Lease（`tenant-operator.axisml.io` 等），需手工 `kubectl delete lease`。

## 4. Tenant Controller


### 1. 概述

tenant-operator 把 AxisML Compute 下发的 `Tenant` CR 翻译为 Kubernetes 侧的命名空间、租户配额与租户级初始化资源，并把执行状态回流到 `Tenant.status`。它承载三类职责：

1. **Namespace 落地**：按 `spec.namespace.name` 创建并维护租户使用的 Namespace；同一 Namespace 允许被多个 Tenant CR 共享
2. **ElasticQuota 派生**：把 `spec.quotas[]` 渲染为 Koordinator `ElasticQuota` CR（每 `(tenant, pool, quota)` 一条，落在租户 Namespace 下），并把 `status.used` 回流到 `Tenant.status.quotas[].used`——这是 Tenant CR 与 Compute / koord-scheduler 之间双向数据链路的承载
3. **初始化资源下发**：按 `spec.initResources` 创建租户私有的 ImagePullSecrets / 通用 Secret / ConfigMap / ServiceAccount + RBAC

operator 与 Compute 的分工以 [compute.md §5 / §6.2.1](compute.md) 为准；本文档只展开 operator 这一侧的实现契约。

Tenant 是 [compute.md §5.4](compute.md) 中的 **配置对象**——CR 缺失/漂移会被 Compute 按 PG 快照补偿重建，因此 operator 的 `Reconcile` 必须可重复执行；operator 不主动反向写 Compute PG。

与 mljob-operator / mlservice-operator 的关键差异：tenant-operator **不存在多后端实现**，无 dispatcher/handler 分层；所有 Tenant CR 由单一 controller 处理。

### 2. 与 Compute 的写路径契约

Compute 采用 Operation Outbox + reconciler 异步下发 CR（详见 [compute.md §5.2](compute.md)）。Operator 暴露给 Compute 的核心契约只有四条；其余约束（不引入 finalizer、`spec.namespace.name` 不可变、namespace 永不删除等）分散在 §3.3 字段不可变性、§5 Reconcile 生命周期、§9 不变量与约束。

- **`Create` 幂等**：相同 `metadata.name` 的二次 `Create()` 返回 409 `AlreadyExists`，且不引发副作用（不重复创建 Namespace、不重置 status）。Compute 收到 409 后会 `Get()` 现有 CR 并校验 label `axisml.io/tenant-id=<uuid>`；只有 label 一致才视为成功
- **status 单向权威**：operator 只写 `Tenant.status`，Compute 只写 `Tenant.metadata` / `Tenant.spec`；状态推进与配额用量回流均由 Compute 侧 Informer 按 CR `status` 消费，operator 不感知 Compute 的 `tenants` / `quotas` 表，也不向 Compute PG 写入任何数据
- **配置补偿友好**：Tenant CR 被误删后 Compute 会按 PG 快照重建（[compute.md §5.4](compute.md) 配置对象路径），operator 的 `Reconcile` 必须可在已存在的底层资源上幂等收敛——已存在的 Namespace / ElasticQuota / Secret 等不重建，只对齐 spec 漂移
- **Namespace 永不级联删除**：Tenant 删除时 operator 仅依赖 ownerReference 让 K8s GC 清理 per-tenant 资源；Namespace 自身不被删除（详见 §6.1 与 §9）

### 3. CRD 契约

Tenant 为 cluster-scoped CR（CRD 定义见 `deploy/helm/axisml-system/crds/tenant-crd.yaml`）：

| 字段 | 取值 |
| --- | --- |
| `apiVersion` | `axisml.io/v1alpha1` |
| `kind` | `Tenant` |
| `scope` | `Cluster` |
| `shortNames` | `tnt` |

Compute 负责设置以下 metadata（与 [compute.md §6.2.1](compute.md) 对齐）：

- `metadata.name` ← `tenants.name`
- `metadata.labels["axisml.io/tenant-id"]` ← `tenants.id`（UUID，孤儿检测稳定锚点）

#### 3.1 spec 设计取舍

把 Tenant 的三类职责建模为同级字段——`namespace`（命名空间引用）、`quotas`（租户配额数组，1:1 渲染为 ElasticQuota CR）、`initResources`（初始化清单），避免任何一类职责喧宾夺主。

**为何把 quotas 内联到 Tenant.spec 而非独立 CR**：Quota 在概念上是 Tenant 的子资源（每条配额都依附于一个 `(tenant, pool)` 组合），生命周期与 Tenant 强绑定。把 quotas 内联到 Tenant CR 让 tenant-operator 成为 ElasticQuota 的 single writer，给 Compute 提供统一的双向数据链路：`spec.quotas[]` 下行表达 desired `min` / `max`，`status.quotas[].used` 上行回流实际用量。Compute 侧仍保留独立的 `quotas` PG 表（[compute.md §6.2.4](compute.md)）以承载 API 行级 CRUD 与跨租户查询；CR 端只是该表的渲染。

**为何 quotas 用数组而非 map**：每条 quota 的标识由 `(pool, name)` 元组确定，map 在 spec 里只能用字符串单 key，会丢失结构。数组配合 §3.3 的不可变约束（`{pool, name}` 一旦写入即作为该项稳定锚点）即可表达。

**为何不在 Tenant 上保留 K8s `ResourceQuota` 兜底字段**：K8s `ResourceQuota` 按 Namespace 聚合计量，不会按 Tenant CR、ServiceAccount 或 `axisml.io/tenant-id` label 自动拆分用量。共享 Namespace 下不能表达 per-tenant 额度；独占 Namespace 下又与 ElasticQuota `max` 形成两套上限，徒增复杂度。租户级容量边界统一收敛到 ElasticQuota（`min` / `max` + Pod label `quota.scheduling.koordinator.sh/name`），tenant-operator 不再创建 `ResourceQuota`。

**为何把 namespace 名放在 `spec.namespace.name` 而非 `metadata.namespace`**：Tenant 是 cluster-scoped CR，不属于任何 Namespace；同时多个 Tenant 可共享同一个 Namespace，把 namespace 作为引用而非容器是更自然的建模——它是 Tenant 的"目标命名空间"而非"宿主命名空间"。这也让"切换 Namespace"作为不可变约束（§3.3）更自然——切 namespace 即切 spec 一个字段，不是 CR 重建。

**为何 per-tenant 资源命名统一加 `axisml-tenant-<tenant-name>-` 前缀**：共享 Namespace 场景下，多个 Tenant 在同一 Namespace 内创建同名 ImagePullSecret / ServiceAccount 会 collide。命名前缀一致化避免 collide，也让 selector 检索（"找出该 Namespace 下属于 tenant X 的所有资源"）有稳定锚点。

**长度上限**：`metadata.name` 已被 Compute API 限制为 ≤40 字符（[compute.md §6](compute.md)）；`axisml-tenant-` 前缀 14 字符 + tenant-name 40 + 分隔符 1 = 55 字符。`spec.initResources.*[].name` 与 `serviceAccounts[].name` 的理论上限因此为 `253 - 55 = 198` 字符（DNS-1123 subdomain 总长 253）。实际命名场景远低于此，operator 不引入额外校验。

**为何初始化资源都从 `sourceXxxRef` 复制而非内联数据**：避免敏感数据（dockerconfigjson、对象存储凭证）以明文形式写入 Tenant CR。源 Secret / ConfigMap 由集群管理员预先放在受控 Namespace（如 `axisml-system`），operator 用 reader 权限读出再写入租户 Namespace。详见 §6.3–§6.5。

**为何 `runPolicy` 字段缺席**：Tenant 不是 workload——没有 `activeDeadlineSeconds` / `progressDeadlineSeconds` / `backoffLimit` 等概念。生命周期控制只有 `suspended` 一个开关（compute.md §6.2.1 状态机里 `Active ⇄ Suspended` 的载体）。

#### 3.2 spec 结构

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
  #      （Compute API 仍可在 best-effort 预检层拦截，详见 compute.md §6.2.4）
  # 共享 Namespace 下允许；ElasticQuota 通过 Pod label 按 quota name 跨 namespace 绑定，
  # quota name 在集群内唯一即可（tenant-operator 用 axisml-<tenant>-<pool>-<name> 命名前缀保证）
  quotas:
    - pool: string               # 必填: ResourcePool 名（与 Compute resource_pools.name 对齐）
      name: string               # 必填: 配额名（"default" / "training" / ...）；(pool, name) 创建后不可变
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
        type: Opaque             # 默认 Opaque；允许 dockerconfigjson / tls 等任意 K8s Secret type
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
          rules: []              # K8s PolicyRule 数组；声明则创建 Role 并绑定到本 SA
          roleRef:               # 可选: 改为绑定到已存在的 ClusterRole 而非新建 Role
            kind: Role | ClusterRole
            name: string

  # ── 控制 ───────────────────────────────────────────────────────
  suspended: false               # cancel 信号；Operator 标记并写 status.phase=Suspended
```

#### 3.3 字段归属与不可变性

| 字段路径 | 写入方 | 创建后可变？ |
| --- | --- | --- |
| `metadata.name` / `labels[axisml.io/tenant-id]` | Compute | 否 |
| `spec.displayName` / `annotations` | Compute（透传用户输入） | 是 |
| `spec.namespace.name` | Compute | **否**；Compute API 在写路径必须拒绝该字段变更（CRD 用 `x-kubernetes-preserve-unknown-fields`，OpenAPI schema 不强制不可变；controller 仅做行为兜底——拒绝动作并写 `status.message`，但 spec 已被持久化），admission webhook 为最终兜底 |
| `spec.namespace.labels` / `annotations` | Compute | 是；只在 Namespace **首次创建** 时落地，已存在的 Namespace 不被覆盖（避免污染共享 Namespace） |
| `spec.quotas[].{pool, name}` | Compute | **否**（每项标识锚点）；删除某项 → reconcile `Delete()` 对应 ElasticQuota CR；新增某项 → reconcile 创建。修改 `(pool, name)` 不被识别为同项漂移，等价于"删旧增新" |
| `spec.quotas[].min` / `max` | Compute | 是；reconcile 同步覆盖到对应 ElasticQuota `spec.min` / `spec.max` |
| `spec.initResources.*` | Compute | 是；增删 → reconcile 创建 / 删除对应资源（per-tenant 命名前缀保证不会误删其他租户资源） |
| `spec.suspended` | API（`/suspend` / `/unsuspend` 触发） | 是 |

**默认值注入**：`spec.suspended` 默认 `false`；`spec.initResources` 各列表默认 `[]`；`spec.namespace.labels` / `annotations` 默认 `{}`；`spec.quotas` 默认 `[]`（视为租户在 K8s 调度层不限额）；`spec.quotas[].min` 默认 `{}`；`spec.initResources.secrets[].type` 默认 `Opaque`。

**CRD schema 现状**：当前 CRD 的 `spec` / `status` 用 `x-kubernetes-preserve-unknown-fields: true`，重新设计字段无需 CRD bump；待行为稳定后再启用 OpenAPI schema 严格校验（§10）。

**status subresource 要求**：`Tenant` CRD 必须启用 `subresources.status`，保证 controller 只写 `status`、Compute 只写 `metadata` / `spec` 的边界能由 Kubernetes API Server 隔离。当前 `deploy/helm/axisml-system/crds/tenant-crd.yaml` 尚未声明该 subresource，属于实现对齐项；本文档先锁定契约，不在本次修订中修改 CRD 文件。

### 4. Status 契约

```yaml
status:
  observedGeneration: int64      # controller reconcile 自洽用；Compute 不强消费
  phase: Active | Suspended | Failed   # ← Compute 消费字段（驱动 tenants.status）
  message: string                # 错误或状态附加信息（Compute 透传到 tenants.message）
  namespaceReady: bool           # Namespace 已就绪（Compute 可观测）
  quotas:                        # 每条 quota 的就绪与用量回流（Compute 消费 used → quotas.used 缓存）
    - pool: string               # 与 spec.quotas[i].pool 对齐
      name: string               # 与 spec.quotas[i].name 对齐
      ready: bool                # ElasticQuota CR 已就绪（spec 已 apply、status.used 已被 koord-scheduler 填充）
      used: {}                   # 资源 map，来自 ElasticQuota.status.used
      message: string            # 错误或状态附加信息
  initResources:                 # 各类初始化资源逐项 ready 状态（UI 可观测，Compute 不消费）
    imagePullSecrets:
      - name: string
        ready: bool
        message: string
    secrets:
      - name: string
        ready: bool
        message: string
    configMaps:
      - name: string
        ready: bool
        message: string
    serviceAccounts:
      - name: string
        ready: bool
        message: string
  conditions:                    # K8s 标准 conditions（UI 可观测，Compute 不消费）
    - type: NamespaceReady | QuotasReady | InitResourcesReady | Suspended | Failed
      status: True | False | Unknown
      lastTransitionTime: timestamp
      reason: string
      message: string
```

**phase 枚举冻结为三态**——`Active | Suspended | Failed`。新增 phase 必须 CRD schema 与 Compute 双侧同步演进。Compute 的状态映射规则（与 [compute.md §6.2.1](compute.md) 对齐）：

| Tenant status.phase | tenants.status |
| --- | --- |
| `Active` | `Active` |
| `Suspended` | `Suspended` |
| `Failed` | `Suspended`，并写入 `message`（保持 Tenant 不可提交新 Job/Service，由人工排查） |

Tenant CR ADD 事件会把 Compute 侧 `Creating` 推进为 `Active`；若 operator 后续回流 `Suspended`，Compute 保持租户不可提交新 Job / Service。`Failed` 收敛为 `Suspended` 是为了让"配置出错"与"管理员暂停"在 Compute 侧表现一致——租户提交链路同样受阻，靠 `message` 区分原因。

**phase 推导规则**（由 controller 在 reconcile 末尾计算）：

| 条件 | phase |
| --- | --- |
| `spec.suspended == true` | `Suspended` |
| `namespaceReady && 所有 quotas[*].ready == true && 所有 initResources[*].ready == true` | `Active` |
| 任一关键资源（Namespace / ElasticQuota）创建失败且非短暂瞬态 | `Failed` |
| 否则（瞬态创建过程中） | 维持上一态，`message` 写当前进展 |

**`quotas` 缺省语义**：当 `spec.quotas` 为空数组时，`status.quotas` 同为空数组，`Active` 推导只看 `namespaceReady` 与 initResources。

**`status.quotas[].used` 回流路径**：tenant-operator 通过 SharedInformerFactory watch 本集群所有 namespace 的 ElasticQuota CR，按 ownerReference 反查所属 Tenant，把 `ElasticQuota.status.used` 聚合到对应 `Tenant.status.quotas[i].used`。Compute Tenant Informer 消费该字段更新 PG `quotas.used` 缓存（详见 [compute.md §5.3 / §6.2.4](compute.md)）。

`conditions` 与 `initResources[]` 是给 UI 与运维用的 observability 字段，Compute 不消费。

### 5. Reconcile 生命周期

按事件源切分 controller 职责：

| 事件 | Controller 行为 |
| --- | --- |
| Tenant ADD（首次创建） | `Validate(spec)` 校验失败写 `status.phase=Failed`；通过则按 §6 顺序确保底层资源就位（Namespace → ElasticQuota → initResources），最后写 `status.phase=Active`。ElasticQuota CR 派生不依赖 Namespace 共享性判断（命名前缀 `axisml-<tenant>-<pool>-<name>` 已天然 per-tenant 隔离） |
| Tenant UPDATE（spec 变更） | 校验 `spec.namespace.name` 不变（违反则写 `status.message` 拒绝并维持原 phase）；其余 spec 变化按 §6 各小节"spec 漂移处理"覆盖底层资源。`spec.quotas[]` 增删项分别触发 ElasticQuota Create / Delete；`min` / `max` 漂移触发 Patch |
| Tenant UPDATE（`spec.suspended` 切换） | true → 写 `status.phase=Suspended`；false → 重新走 phase 推导（§4）。controller 不停机底层资源，只标记 phase；阻断新 Job/Service 提交由 Compute API 在 `tenant.status='Suspended'` 时拦截（compute.md §6.2.1） |
| Tenant DELETE | 不阻断；不引入 finalizer；K8s GC 通过 ownerReference 级联删除 per-tenant 资源（ElasticQuota / Secret / ConfigMap / SA / Role / RoleBinding）；**Namespace 不删除**（§6.1） |
| 底层资源事件（ElasticQuota / Secret / ConfigMap / SA 等被外部修改或删除） | 按 ownerReference 反查到 Tenant，重新触发 Reconcile；漂移按各小节策略覆盖回 spec 快照。ElasticQuota 的 `status.used` 变更通过 watch 触发轻量 reconcile，仅刷新 `Tenant.status.quotas[i].used`，不重写 ElasticQuota spec |
| 周期 resync（默认 10 min，Helm values 可配） | 触发对所有 Tenant CR 的 reconcile，重读 `sourceSecretRef` / `sourceConfigMapRef` 源数据，按 §6.3–§6.5 漂移策略覆盖 per-tenant 副本——operator 不为源资源建立 watch（避免对 `axisml-system` 等受控 Namespace 引入额外 informer + RBAC），源更新最大延迟 = resync 间隔 |

**关键约束**：

- Controller **不引入 finalizer**；ownerReference 级联清理是默认路径；Namespace 不参与级联（无 ownerReference）
- `Validate(spec)` 必须是纯函数（不发起 K8s 调用），便于未来在 admission webhook 中复用
- `Reconcile` 多次调用相同 spec 不重建底层资源；只有语义字段变化才触发更新
- Status 写入只在 reconcile 末尾通过一次 patch 完成，避免半成品 status
- 源 Secret / ConfigMap 不建立 watch；其内容变更经"周期 resync"路径感知，最大延迟 = resync 间隔（默认 10 min）。需要更短延迟可由集群管理员 bump Tenant CR 的某个 annotation 显式触发 reconcile

### 6. 底层资源管理

每条 per-tenant 资源都叠加以下 label，便于在共享 Namespace 内 selector 过滤：

- `axisml.io/tenant-id=<uuid>`
- `axisml.io/managed-by=tenant-operator`

每条 per-tenant 资源都通过 `ownerReferences` 指向 Tenant CR（cluster-scoped owner → namespaced dependent，K8s GC 原生支持）；Tenant 删除后由 K8s GC 异步清理。

#### 6.1 Namespace

| 维度 | 行为 |
| --- | --- |
| 命名 | `<spec.namespace.name>`（直接采用 spec 字段） |
| 创建 | Namespace 不存在 → 创建，附加 `spec.namespace.labels` / `annotations` 并叠加 `axisml.io/managed-by=tenant-operator` label |
| 已存在 | 仅补 `axisml.io/managed-by=tenant-operator` label（如缺失）；不覆盖任何其他既有 label / annotation，避免污染共享 Namespace 中由其他 Tenant 或管理员设置的字段。`spec.namespace.labels` / `annotations` 也不会回填到已存在的 Namespace（与 §3.3 行为一致）。**风险**：`Namespace` 是 cluster-scoped 资源，K8s RBAC 不能按前缀或业务范围限制 `create`；controller 配置中必须维护目标 Namespace denylist / allowlist（默认拒绝 `kube-*`、`default`、`axisml-system` 等系统 Namespace），admission webhook 作为后续兜底（§10） |
| ownerReference | **不设置**——Namespace 不属于任何单一 Tenant |
| spec 漂移 | 不主动对账（Namespace 自身没有"由 Tenant 决定"的 spec 字段） |
| 删除 | **永不删除**——即使最后一个引用本 Namespace 的 Tenant 被删除，Namespace 也保留。空 Namespace 由集群管理员手工清理 |

**为何不删 Namespace**：Namespace 中可能存在 Tenant 不可见的 PV、外部 controller 创建的资源（如 PodGroup 历史记录、ElasticQuota 与 KServe 派生对象、用户手工创建的 ConfigMap）。误删 Namespace 会引发不可逆的状态丢失。把"清理空 Namespace"作为运维操作而非 operator 的自动行为是更安全的取舍——代价仅是空 Namespace 残留。

`status.namespaceReady` 在 Namespace `phase=Active` 时为 `true`。

#### 6.2 ElasticQuota

`spec.quotas[]` 每项 1:1 渲染为一条 Koordinator `ElasticQuota` CR（`scheduling.sigs.k8s.io/v1alpha1`，namespace-scoped）。CR 落在 `spec.namespace.name` 下；Pod 通过 label `quota.scheduling.koordinator.sh/name=<eq-name>` 按集群唯一名跨 namespace 绑定 quota（详见 [compute.md §6.2.4 多 namespace 契约](compute.md)），所以共享 Namespace 与独占 Namespace 在 quota 隔离上没有差别——命名前缀 `axisml-<tenant>-<pool>-<name>` 已天然 per-tenant 隔离。

| 维度 | 行为 |
| --- | --- |
| 命名 | `axisml-<tenant-name>-<pool>-<quota-name>`（与 [compute.md §6.2.4](compute.md) 命名约定对齐；集群内唯一）|
| 创建 | 在 `<spec.namespace.name>` 内创建 ElasticQuota，`spec.min` / `spec.max` 直接来自 `spec.quotas[i].min` / `max` |
| **缺省** | `spec.quotas` 为空数组 → operator 不创建任何 ElasticQuota；后续 spec 增项时按项创建；spec 删项时 reconcile 显式 Delete 对应 ElasticQuota |
| 共享 Namespace | 不影响 ElasticQuota 创建——ElasticQuota name 集群唯一，多个 Tenant 在同一 Namespace 内的 quotas 通过命名前缀互不干扰；Pod 按 label 名字绑定 quota，与 Pod 所在 Namespace 解耦 |
| ownerReference | Tenant CR（cluster-scoped owner → namespaced dependent，K8s GC 原生支持） |
| spec 漂移 | reconcile 检测到 `ElasticQuota.spec.{min, max}` 与 `spec.quotas[i].{min, max}` 不一致时按 spec 覆盖（Tenant CR 为本端权威；Tenant CR 自身由 Compute 按 PG 快照重算） |
| status.used 回流 | watch ElasticQuota，把 `status.used` 写入 `Tenant.status.quotas[i].used`；不写回 ElasticQuota |
| 删除 | 随 Tenant 删除 K8s GC；spec 中删除某项 → reconcile 显式 Delete 对应 ElasticQuota CR |

**配额校验**：`Validate(spec)` 对每条 quota 校验 `min[k] ≤ max[k]` 且均 ≥ 0；`(pool, name)` 在 `spec.quotas[]` 内唯一；`pool` / `name` 命名遵循 §3.1 长度上限。校验失败 → `status.phase=Failed`、`status.quotas[i].ready=false`、`message` 指明违规项。

**与 Compute 的写路径分工**：Compute 维护 PG `quotas` 表（行级 CRUD、跨租户查询、used 缓存），并在 reconciler 渲染 Tenant CR 时把 `quotas WHERE tenant_id=$1 AND deleted_at IS NULL` 写入 `Tenant.spec.quotas[]`；tenant-operator 只读 `Tenant.spec` 派生 ElasticQuota CR，不感知 Compute PG。

`status.quotas[i].ready` 在对应 ElasticQuota 创建成功且 `status.used` 已被 koord-scheduler 填充时为 `true`。

#### 6.3 ImagePullSecrets

| 维度 | 行为 |
| --- | --- |
| 命名 | `axisml-tenant-<tenant-name>-<spec-name>`（`spec-name` 取 `spec.initResources.imagePullSecrets[].name`） |
| 类型 | `kubernetes.io/dockerconfigjson` |
| 数据来源 | `sourceSecretRef.{namespace, name}` 指向受控 Namespace（如 `axisml-system`）中预先创建好的 Secret；operator 用 reader 权限 `Get()` 后把 `data` 写入新 Secret |
| ownerReference | Tenant CR |
| spec 漂移 | reconcile 检测到本端 Secret 数据与源 Secret 不一致时覆盖；源 Secret 不存在时把对应 `status.initResources.imagePullSecrets[i].ready=false` 并写 message |
| 删除 | 随 Tenant 删除 GC；spec 中删除某项 → reconcile 显式 Delete 对应 Secret |

**为何用 `sourceSecretRef` 而非内联 dockerconfigjson**：避免明文凭证写入 Tenant CR（CR 通过 etcd 持久化、可能被备份导出）。源 Secret 集中放在受控 Namespace 由集群管理员维护，权限收敛。

#### 6.4 通用 Secret

| 维度 | 行为 |
| --- | --- |
| 命名 | `axisml-tenant-<tenant-name>-<spec-name>` |
| 类型 | 取 `spec.initResources.secrets[].type`，默认 `Opaque`；允许 `dockerconfigjson` / `kubernetes.io/tls` 等任意 K8s Secret type |
| 类型不一致 | `spec.type` 与源 Secret 的 `type` 不一致时以 spec 为准，operator 在 `status.initResources.secrets[i].message` 写警告并按 `spec.type` 创建本端 Secret；若结构性约束失败（如 `dockerconfigjson` 要求特定 key、`tls` 要求 `tls.crt`/`tls.key`） → 该项 `ready=false`、message 指明缺失字段。Secret type 在 K8s 中不可变，运行时 `spec.type` 改动 → reconcile 删除现有 Secret 重建 |
| 数据来源 | 同 §6.3，`sourceSecretRef` 复制 |
| ownerReference / 漂移 / 删除 | 同 §6.3 |

典型用途：对象存储访问凭证、TLS 证书、OAuth client secret 等租户私有的非镜像凭证。

#### 6.5 ConfigMap

| 维度 | 行为 |
| --- | --- |
| 命名 | `axisml-tenant-<tenant-name>-<spec-name>` |
| 数据来源 | `sourceConfigMapRef.{namespace, name}`，operator `Get()` 源 ConfigMap 后把 `data` / `binaryData` 写入新 ConfigMap |
| ownerReference | Tenant CR |
| spec 漂移 | reconcile 检测到本端数据与源不一致时覆盖；源不存在时 `ready=false` |
| 删除 | 同 §6.3 |

典型用途：租户级默认环境变量、CA 证书包、统一日志采集配置等。

#### 6.6 ServiceAccount + RBAC

| 维度 | 行为 |
| --- | --- |
| ServiceAccount 命名 | `axisml-tenant-<tenant-name>-<spec-name>` |
| `imagePullSecrets` | 把 `spec.initResources.serviceAccounts[].imagePullSecrets`（按用户可见 name）解析为最终 Secret 名（`axisml-tenant-<tenant>-<name>`）后填到 SA `imagePullSecrets[]` |
| **引用校验** | `Validate(spec)` 检查 `serviceAccounts[].imagePullSecrets[]` 中每个 name 必须能在 `spec.initResources.imagePullSecrets[].name` 中找到；找不到 → 校验失败 → `status.phase=Failed`，message 指明悬空引用名 |
| Role 命名 | `axisml-tenant-<tenant-name>-<sa-name>`（仅当声明 `rbac.rules` 且未指定 `rbac.roleRef.kind=ClusterRole`） |
| RoleBinding 命名 | `axisml-tenant-<tenant-name>-<sa-name>`（声明 `rbac` 即创建） |
| RoleBinding 绑定关系 | `subjects` 指向本 SA；`roleRef` 指向同名 Role（默认）或 `spec.rbac.roleRef`（指定时） |
| ownerReference | Tenant CR（SA / Role / RoleBinding 均挂） |
| spec 漂移 | reconcile 检测 SA `imagePullSecrets` / Role `rules` / RoleBinding `subjects+roleRef` 漂移时覆盖 |
| 删除 | 随 Tenant 删除 GC；spec 中删除某条 SA → reconcile 显式 Delete 对应 SA + 关联 Role + RoleBinding |

**RBAC 的两种使用形态**：

1. `rbac.rules` 非空、未指定 `roleRef.kind=ClusterRole` → operator 创建一个独立 Role 持有 `rules`，并创建 RoleBinding 绑定本 SA
2. `rbac.roleRef.kind=ClusterRole` → operator 不创建 Role，只创建 RoleBinding 把本 SA 绑定到指定 ClusterRole（适用于"复用平台预置 ClusterRole"场景）

### 7. 多 Tenant 共享 Namespace 语义

`spec.namespace.name` 允许多个 Tenant CR 指向同一 Namespace；典型场景是多个轻量级团队共享一个开发 / 沙箱环境。共享时的关键不变量：

- **Namespace 自身不绑定 ownerReference**——Namespace 是共享资源，不属于任一 Tenant
- **per-tenant 资源命名前缀**：tenant-operator 派生的 per-tenant 资源（Secret / ConfigMap / SA / Role / RoleBinding）用 `axisml-tenant-<tenant-name>-` 前缀；ElasticQuota 用 `axisml-<tenant-name>-<pool>-<quota>` 前缀。两套前缀都集群唯一，共享 Namespace 内不会 collide
- **per-tenant 资源 label `axisml.io/tenant-id=<uuid>`** 提供 selector 检索能力（"该 Namespace 内属于 tenant X 的所有资源"）
- **per-tenant ElasticQuota**：每个 Tenant 在共享 Namespace 内仍各自持有独立 ElasticQuota CR，Pod 通过 label `quota.scheduling.koordinator.sh/name` 按集群唯一 quota name 绑定；koord-scheduler 按名字跨 namespace 维护用量，per-tenant 隔离照常生效
- **Pod 通过 ServiceAccount 关联 tenant**：Pod 选择 `axisml-tenant-<tenant>-<sa>` SA → 自动获得本 tenant 的 imagePullSecrets / RBAC；ServiceAccount 是 tenant 身份在 K8s API 调用面的载体
- **Tenant A 删除不影响 Tenant B**：Tenant A 删除 → K8s GC 清理 A 的 per-tenant 资源（ElasticQuota / Secret / ConfigMap / SA / Role / RoleBinding 等）→ B 的 per-tenant 资源不受影响 → Namespace 保留

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

#### 与 Compute schema 的关系

[compute.md §6.2.1](compute.md) 已把 `tenants.namespace` 建模为非唯一索引，允许多个 Tenant 指向同一 Namespace。tenant-operator 端不再需要通过 Tenant CR 列表识别共享关系——ElasticQuota 的 per-tenant 隔离由命名前缀 + Pod label 自然达成，与 Namespace 是否被共享解耦。

### 8. RBAC

operator binary 启动时聚合以下权限到 ServiceAccount。Helm chart 通过 values 控制是否启用 ServiceAccount + RBAC 子能力（关闭时可裁剪 §6.6 相关权限）。

| 资源 | 动作 | 用途 |
| --- | --- | --- |
| `tenants.axisml.io` | `get / list / watch / patch` | watch Tenant CR、写 status |
| `namespaces` | `create / get / list / watch / update / patch` | 创建并对齐 Namespace metadata；**不含 `delete`** |
| `elasticquotas.scheduling.sigs.k8s.io` | `create / get / list / watch / update / patch / delete` | 派生并维护 per-tenant ElasticQuota；watch `status.used` 回流到 Tenant.status |
| `secrets` | `create / get / list / watch / update / patch / delete`（目标 Namespace）；`get`（源 Namespace） | 维护 per-tenant Secret（ImagePullSecrets / 通用 Secret）；源 Secret 只按 `sourceSecretRef` 点查复制，不建立 watch |
| `configmaps` | `create / get / list / watch / update / patch / delete`（目标 Namespace）；`get`（源 Namespace） | 维护 per-tenant ConfigMap；源 ConfigMap 只按 `sourceConfigMapRef` 点查复制，不建立 watch |
| `serviceaccounts` | `create / get / list / watch / update / patch / delete` | 维护 per-tenant ServiceAccount |
| `roles.rbac.authorization.k8s.io` | `create / get / list / watch / update / patch / delete` | 维护 per-tenant Role |
| `rolebindings.rbac.authorization.k8s.io` | `create / get / list / watch / update / patch / delete` | 维护 per-tenant RoleBinding |
| `events` | `create / patch` | 写 K8s Event 用于运维观测 |

**`namespaces` 不含 `delete`** 是显式的最小权限策略，对应 §6.1 的"永不删除 Namespace"行为约束——即便代码出错也无法越权删除 Namespace。

**跨 Namespace 复制源资源的最小权限**：operator 默认只需要对源 Secret / ConfigMap 做 `get`；如需收敛权限，可在 RBAC 上把源资源读取限定到指定 Namespace（例如 `axisml-system`），通过 Role + RoleBinding 而非 ClusterRole 表达。源资源不建立 watch，更新感知仍走 §5 的周期 resync。

### 9. 不变量与约束

- `spec.namespace.name` 创建后不可变；controller 拒绝并写 `status.message`，admission webhook 后续接管
- `spec.quotas[].{pool, name}` 创建后不可变（修改等价于"删旧增新"）；ElasticQuota CR `metadata.name` 不会被 in-place 改名
- **operator 永不删除 Namespace**；即使最后一个引用本 Namespace 的 Tenant 被删除，Namespace 也保留
- Controller **不引入 finalizer**；级联清理依赖 ownerReference（cluster-scoped Tenant → namespaced 子资源）
- per-tenant 初始化资源命名带 `axisml-tenant-<tenant-name>-` 前缀；ElasticQuota 命名带 `axisml-<tenant-name>-<pool>-<quota>` 前缀；两套前缀都集群唯一，避免共享 Namespace 内 collide
- per-tenant 资源必须打 `axisml.io/tenant-id` 与 `axisml.io/managed-by=tenant-operator` label；缺失即视为 operator 不应识别（用于人工排障误打 label 的场景）
- ElasticQuota CR 由 tenant-operator 独占 owner（spec 写、status.used 读）；operator 不读写 MLJob / MLService CR——这些由对应 operator 维护
- operator 不向 Compute PG 写入任何数据；状态与 quota 用量全部经由 Tenant `status` + Compute Tenant Informer 回流
- `status.phase` 取值集合冻结为三态（`Active | Suspended | Failed`）；新增 phase 必须经 CRD schema 与 Compute 双侧同步演进
- 初始化资源数据来源限定为同集群内的 `sourceSecretRef` / `sourceConfigMapRef`；不接受内联敏感数据（避免 etcd 明文持久化）

### 10. 后续设计文档（不在本文档范围）

- [compute.md §6.2.1](compute.md) 已放宽 `tenants.namespace` 唯一约束（本文档随 namespace 共享语义同步落地）；尚需补齐"反查同 Namespace 下所有 Tenant"的查询路径以及配套的 UI 展示
- Admission webhook：`spec.namespace.name` / `spec.quotas[].{pool, name}` 不可变约束、`spec.initResources.*.sourceXxxRef` 跨 Namespace 读权限白名单、`spec.quotas[].{min, max}` 结构性校验
- **目标 Namespace 白名单**：当前通过 controller Helm values 配置 denylist / allowlist，拒绝 Tenant 指向系统 Namespace（如 `kube-system` / `kube-public` / `axisml-system`）；后续由 admission webhook 前移到准入阶段（与 §6.1 风险脚注呼应）
- **源资源结构性校验前移**：admission webhook 在 Tenant 创建/更新时校验源 Secret 的 type 与 spec 一致、`dockerconfigjson` / `tls` 等结构性 key 完整，避免运行时才暴露错误
- **resync 间隔的 Helm values 暴露**：默认 10 min，运维可下调到分钟级换取更短的源资源更新延迟
- 加密源支持：从 KMS / Vault / Sealed Secrets 拉取凭证作为 `sourceSecretRef` 替代方案
- `spec.initResources` templating：按 tenant 上下文（id / name / namespace）渲染 ConfigMap 数据（如把 `<tenant-name>` 注入到统一日志采集配置）
- 跨 Namespace 复制源的 RBAC 收敛：把源 Namespace 限定为单一受控 Namespace（如 `axisml-system`），通过 Role + RoleBinding 而非 ClusterRole 表达
- ServiceAccount + RBAC 子能力的 Helm values 开关与对应 RBAC 收敛
- 分层配额：在 `spec.quotas[]` 引入 `parent` 字段，落到 ElasticQuota 的 `quota.scheduling.koordinator.sh/parent` annotation；在出现真实多团队/多业务线共享租户的诉求时按需启用
- CRD 严格 schema（启用 OpenAPI 校验，替换当前的 `x-kubernetes-preserve-unknown-fields`）

## 5. MLJob Controller


### 1. 概述

mljob-operator 把 AxisML Compute 下发的 `MLJob` CR 翻译为底层执行资源（Pod / PodGroup / 第三方 CR），并把执行状态回流到 `MLJob.status`。它内部按 `spec.backend.{name, engine}` 二级元组路由到不同 Handler，由 Handler 真正生成底层资源、把后端原生状态映射回统一的 phase 集合（详见 [overview.md §5.3](overview.md)）。

operator 与 Compute 的分工以 [compute.md §5](compute.md) 为准；本文档只展开 operator 这一侧的实现契约。

### 2. 与 Compute 的写路径契约

Compute 采用 Operation Outbox + reconciler 异步下发 CR（详见 [compute.md §5.2](compute.md)）。Operator 暴露给 Compute 的核心契约只有两条；其余约束（`backend.{name, engine}` 不可变、不引入 finalizer、Suspend 声明义务等）分散在 §3.3 字段不可变性、§6 Reconcile 生命周期、§7 Handler 接口契约、§10 不变量与约束。

- **`Create` 幂等**：相同 `metadata.name` 的二次 `Create()` 返回 409 `AlreadyExists`，且不引发副作用（不重启 Pod、不重置 status）。Compute 收到 409 后会 `Get()` 现有 CR 并校验 label `axisml.io/job-id=<uuid>`；只有 label 一致才视为成功
- **status 单向权威**：operator 只写 `MLJob.status`，Compute 只写 `MLJob.metadata` / `MLJob.spec`；状态推进由 Compute 侧 Informer 按 CR `status` 回流，operator 不感知 Compute 的 `jobs` 表，也不向 Compute PG 写入任何数据

### 3. CRD 契约

MLJob 为 namespaced CR（CRD 定义见 `deploy/helm/axisml-system/crds/mljob-crd.yaml`）：

| 字段 | 取值 |
| --- | --- |
| `apiVersion` | `axisml.io/v1alpha1` |
| `kind` | `MLJob` |
| `scope` | `Namespaced`（创建在租户 namespace 下） |
| `shortNames` | `mlj` |

Compute 负责设置以下 metadata（与 [compute.md §6.3.1](compute.md) 对齐）：

- `metadata.name` ← `jobs.name`
- `metadata.namespace` ← `tenants.namespace`
- `metadata.labels["axisml.io/job-id"]` ← `jobs.id`（UUID，孤儿检测稳定锚点）
- `metadata.labels["axisml.io/tenant"]` ← 租户名
- `metadata.labels["axisml.io/quota"]` ← Compute Quota bare name（如 `training`，**不是** ElasticQuota 全名）

#### 3.1 spec 设计取舍

把"角色拓扑"提升为一等公民。Job 的执行域用 `roles[]` 数组承载：

- 单角色 backend（如 `(native, job)`、`(native, podgroup)`）声明一个 role
- 多角色 backend（如 PyTorchJob 的 master/worker、TFJob 的 chief/worker/ps/evaluator、MPIJob 的 launcher/worker）声明多个 role
- role 名集合由各 Handler 在 §8 中约定，由 Handler 的 `Validate` 强制

替代方案是把 `image / command / replicas / resources` 全部摆在 spec 顶层（早期方案），对单角色自然，但多角色 backend 不得不把角色切分挤进 `backend.config`，让 generic 字段失去意义——`spec.replicas` 在多角色场景下到底指哪个？`spec.resources` 又对哪个角色生效？引入 `roles[]` 后，单角色场景退化为"只有一个 role 的特例"，避免这种"通用字段对一类后端无意义"的尴尬。

调度域的 `nodeSelector` / `tolerations` 沿用 K8s PodSpec 扁平惯例直接放在 `spec.scheduling` 下，不再额外包一层 `placement`。

#### 3.2 spec 结构

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

#### 3.3 字段归属与不可变性

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

### 4. Status 契约

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

**phase 枚举冻结为四态**——`Pending | Running | Succeeded | Failed`。新增 phase 必须 CRD schema 与 Compute 双侧同步演进。Compute 的状态映射规则（与 [compute.md §6.3.1](compute.md) 对齐）：

| MLJob status.phase | jobs.status | 终态 |
| --- | --- | --- |
| `Pending` | `Pending` | 否 |
| `Running` | `Running` | 否 |
| `Succeeded` | `Succeeded` | 是 |
| `Failed` | `Failed` | 是 |

**Cancel 推进信号**——`Cancelled` 与 `Deleted` 仍不由 operator 直接产出，但 cancel 路径有明确的链上信号：Handler 在收到 `spec.runPolicy.suspend=true` 并完成"暂停或清理底层资源"后，**必须向 dispatcher 返回 `suspendCompleted=true` 与 `reason=CancelRequested`**；dispatcher 统一合并写入 `status.conditions[type=Suspended,status=True,reason=CancelRequested]`，且在非终态时让 `status.phase` 维持在 `Pending`。Compute Informer 在 PG `status='Canceling'` 时把这个 condition 当作推进信号 → 写 `Cancelled` → 入队 `Delete()` 做 CR 资源回收（DELETE 事件幂等到达，不再变更 PG 状态；详见 [compute.md §5.2 / §5.3 / §6.3.1](compute.md)）。`Deleted` 仍由 Compute Informer 在观察到 CR DELETE 事件后基于 PG 当前 `status` 推导。

**终态优先**：cancel 只面向仍处于 `Pending` / `Running` 的 Job。若底层资源已经进入 `Succeeded` / `Failed`，或同一轮 `MapStatus` 已经推导出终态，dispatcher 必须保留终态 phase 与 `finishedAt`，不能为了 cancel 信号把 `status.phase` 回退为 `Pending`；此时不写 `Suspended=True` 作为成功取消信号。

`Suspended` 之外的 `conditions` 与 `roles[]` 是给 UI 与运维用的 observability 字段，Compute 不消费。这样既保留了 K8s 标准实践（`metav1.Condition`、per-role 副本聚合）又不污染 Compute 状态机的简洁性。

跨 Handler 的 phase 映射规则原则：所有 Handler 在 `MapStatus` 中负责把后端原生状态映射到这四态，映射表写入对应 Handler 章节（§8）。

### 5. 总体架构：Dispatcher + Handler

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

### 6. Reconcile 生命周期

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

**Pod 模板注入约定**（跨 Handler 通用，体现 [infra.md §8.3](infra.md) 的 Quota 全覆盖不变式）：

| Pod 字段 / Label | 必填 | 取值 | 用途 |
| --- | --- | --- | --- |
| `spec.schedulerName` | 是 | `koord-scheduler` | 所有 AxisML workload Pod 一律走 koord-scheduler；不允许任何 backend 让 Pod 落到默认 kube-scheduler 上 |
| label `quota.scheduling.koordinator.sh/name` | 是 | `<spec.scheduling.quota>` | Koordinator 原生 quota 关联 label；ElasticQuota plugin 据此把该 Pod 计入 `status.used` |
| label `axisml.io/job-id` | 是 | `jobs.id`（UUID） | 反查 MLJob，与 CR 上同名 label 一致 |
| label `axisml.io/role` | 是 | role 名（如 `worker` / `master` / `launcher`） | 区分多角色拓扑下的 Pod |
| label `axisml.io/quota` | 是 | Compute Quota bare name（取自 MLJob CR `metadata.labels["axisml.io/quota"]` 透传，**与 `quota.scheduling.koordinator.sh/name` 取值不同**：前者是裸名如 `training`，后者是 ElasticQuota 全名如 `axisml-<tenant>-<pool>-training`） | AxisML 自有审计 / 查询；不参与调度 |
| label `axisml.io/replica-index` | 否 | role 内 0-based 序号 | 副本身份天然稳定时建议透传：StatefulSet 的 `apps.kubernetes.io/pod-index`、Indexed Job 的 `batch.kubernetes.io/job-completion-index`；NonIndexed Job、裸 Pod 拓扑下省略 |

前 5 项必填，所有 Handler 一律遵守；`axisml.io/replica-index` 只是可观测增强，缺失时 Compute §7.4 日志 API 退化为按 pod 名定位（详见 [compute.md §7.4](compute.md)）。

### 7. Handler 接口契约

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

### 8. 内置 Handler

每个 Handler 章节统一按以下小节组织：**底层资源 / `backend.config` / 通用字段映射 / Status 映射 / Suspend / RBAC**。

#### 8.1 `(native, job)`

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

#### 8.2 `(native, podgroup)`

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

裸 Pod 拓扑没有稳定 index，省略 `axisml.io/replica-index`；日志 API 通过 pod 名直接定位（详见 [compute.md §7.4](compute.md)）。

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

#### 8.3 `(kubeflow-trainer, pytorchjob)`

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

#### 8.4 `(kubeflow-trainer, tfjob)`

同 §8.3 思路，将 MLJob 翻译为 Kubeflow Trainer 的 `TFJob`。Role 集合约定为 `chief` / `worker` / `ps` / `evaluator`（任一可省略，replicas=0 表示禁用）。各 replica 模板同样必须注入 `schedulerName: koord-scheduler` + §6 必填 label；多角色 gang 通过同一 PodGroup（`minMember=sum(replicas)`）表达。Status 映射沿用 TFJob 的 condition 集，与 §8.3 PyTorchJob 同构。Suspend 优先走原生 `runPolicy.suspend`，目标版本不支持时 fallback 为 `Cleanup()`；Handler 完成底层动作后返回结构化 suspend 结果，由 dispatcher 统一写 `Suspended` condition。

#### 8.5 `(kubeflow-trainer, mpijob)`

将 MLJob 翻译为 Kubeflow [`MPIJob`](https://www.kubeflow.org/docs/components/training/mpi/) CR。Role 集合约定为 `launcher`（replicas=1）+ `worker`（replicas≥1）。`backend.config` 携带 MPI 实现选择（OpenMPI / Intel MPI）与 launcher / worker 通讯参数。各 replica 模板同样必须注入 `schedulerName: koord-scheduler` + §6 必填 label；MPIJob 的 PodGroup 由本 Handler 创建（`minMember=launcher.replicas + worker.replicas`）。Status 映射对齐 MPIJob `status.conditions`。Suspend 优先走原生 `runPolicy.suspend`，目标版本不支持时 fallback 为 `Cleanup()`；Handler 完成底层动作后返回结构化 suspend 结果，由 dispatcher 统一写 `Suspended` condition。

#### 8.6 `(custom, *)`

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

由 custom Handler 通过 unstructured client 创建并跟踪。**仍受 §6 Pod 注入约定与 [infra.md §8.3](infra.md) Quota 全覆盖不变式约束**：custom Handler 必须保证最终落地的 Pod 模板上有 `schedulerName: koord-scheduler` 与 `quota.scheduling.koordinator.sh/name` label，否则 Validate 直接拒绝。完整 schema 与 unstructured 操作约定由独立设计文档落地（见 §11）。

### 9. RBAC 聚合

operator binary 启动时遍历 registry，把每个启用 Handler 的 `RequiredRBAC()` 合并去重，生成本 binary 实际需要的 ClusterRole rules。Helm chart 通过 values 控制启用集合，渲染最小化 RBAC 而非全集——例如仅启用 `(native, job)` 时不引入 PodGroup 相关 RBAC；启用 `(native, podgroup)` 才注入 `podgroups.scheduling.sigs.k8s.io` CRUD；启用 `(kubeflow-trainer, *)` 才注入对应 CR 的 RBAC。所有路径都不需要 ElasticQuota 的 RBAC（ElasticQuota 由 Compute 独占维护）。

### 10. 不变量与约束

- `spec.backend.{name, engine}` 创建后不可变；dispatcher 拒绝并写 `status.message`，admission webhook 后续可前置拦截
- `(backend, engine)` 元组未在 registry 注册 → MLJob 直接进入 `Failed`，message 写明缺失原因
- 任一 Handler 的 `Reconcile` / `MapStatus` 必须保持 §2 列出的写路径契约——这是把"插件"安全嵌入 Compute Outbox 模型的根基
- Handler 不直接修改 ElasticQuota CR；ElasticQuota CR 由 Compute 独占维护
- Handler 不向 Compute PG 写入任何数据；状态全部经由 MLJob `status` + Informer 回流
- **Handler 不引入 finalizer**；级联清理依赖 ownerReference + `Cleanup()`
- **`status.phase` 取值集合冻结为四态**（`Pending | Running | Succeeded | Failed`）；新增 phase 必须经 CRD schema 与 Compute 双侧同步演进
- 所有 Handler 派生的 Pod 必须满足 §6 Pod 注入约定的前 5 项必填字段；缺失任一项视为契约违反，Validate 必须在创建前拦截
- 所有 Handler 在 cancel 路径完成 suspend / Cleanup 后必须返回 `suspendCompleted=true, reason=CancelRequested`；dispatcher 据此写 `status.conditions[type=Suspended,status=True,reason=CancelRequested]`，非终态 `phase` 维持 `Pending`；这是 Compute 推进 `Canceling → Cancelled` 的唯一信号

### 11. 后续设计文档（不在本文档范围）

- `(native, job)` Handler 的 Indexed Job 模式与 `podFailurePolicy` 直通策略细节
- `(native, podgroup)` Handler 的 PodGroup `minResources` 与 elastic gang 演进
- `(kubeflow-trainer, pytorchjob / tfjob / mpijob / paddlejob / xgboostjob)` 各自的字段映射与状态映射细节，含每路径下的 PodGroup 创建策略
- `(custom, *)` Handler 的 `config` 完整 schema 与 unstructured 操作约定（包括 schedulerName / quota label 强制注入校验）
- Admission webhook：`spec.backend.{name, engine}` 不可变约束、`backend.config` 按 Handler 自带 schema 的统一校验
- Handler chart 的 values 控制（启用 / 禁用 backend，对应 RBAC 与 watch 的开关）
- CRD 严格 schema（启用 OpenAPI 校验，替换当前的 `x-kubernetes-preserve-unknown-fields`，含 `spec.backend.name` enum 收为 `{native, kubeflow-trainer, custom}` 与 `spec.scheduling.quota` 必填）

## 6. MLService Controller


### 1. 概述

mlservice-operator 把 AxisML Compute 下发的 `MLService` CR 翻译为底层在线推理资源（Deployment + Service / KServe `InferenceService` / 自定义 GVK），并把执行状态回流到 `MLService.status`。它内部按 `spec.backend.{name, engine}` 二级元组路由到不同 Handler，由 Handler 真正生成底层资源、把后端原生状态映射回统一的 phase 集合（详见 [overview.md §5.3](overview.md)）。

operator 与 Compute 的分工以 [compute.md §5 / §6.3.2](compute.md) 为准；本文档只展开 operator 这一侧的实现契约。

### 2. 与 Compute 的写路径契约

Compute 采用 Operation Outbox + reconciler 异步下发 CR（详见 [compute.md §5.2](compute.md)）。Operator 暴露给 Compute 的核心契约只有两条；其余约束（`backend.{name, engine}` 不可变、不引入 finalizer、`/scale` 唯一可变、无 suspend 语义等）分散在 §3.3 字段不可变性、§6 Reconcile 生命周期、§10 不变量与约束。

- **`Create` 幂等**：相同 `metadata.name` 的二次 `Create()` 返回 409 `AlreadyExists`，且不引发副作用（不重启 Pod、不重置 status）。Compute 收到 409 后会 `Get()` 现有 CR 并校验 label `axisml.io/service-id=<uuid>`；只有 label 一致才视为成功
- **status 单向权威**：operator 只写 `MLService.status`，Compute 只写 `MLService.metadata` / `MLService.spec`；状态推进由 Compute 侧 Informer 按 CR `status` 回流，operator 不感知 Compute 的 `services` 表，也不向 Compute PG 写入任何数据

### 3. CRD 契约

MLService 为 namespaced CR（CRD 定义见 `deploy/helm/axisml-system/crds/mlservice-crd.yaml`）：

| 字段 | 取值 |
| --- | --- |
| `apiVersion` | `axisml.io/v1alpha1` |
| `kind` | `MLService` |
| `scope` | `Namespaced`（创建在租户 namespace 下） |
| `shortNames` | `mls` |

Compute 负责设置以下 metadata（与 [compute.md §6.3.2](compute.md) 对齐）：

- `metadata.name` ← `services.name`
- `metadata.namespace` ← `tenants.namespace`
- `metadata.labels["axisml.io/service-id"]` ← `services.id`（UUID，孤儿检测稳定锚点）
- `metadata.labels["axisml.io/tenant"]` ← 租户名
- `metadata.labels["axisml.io/quota"]` ← Compute Quota bare name（如 `training`，**不是** ElasticQuota 全名）

#### 3.1 spec 设计取舍

把 "角色拓扑" 提升为一等公民，与 mljob-operator §3.1 同源。Service 的执行域用 `roles[]` 数组承载：

- 单角色 backend（如 `(native, deployment)` / `(native, statefulset)`）声明一个 role（约定 `name=predictor`）
- 多角色 backend（如 KServe `InferenceService` 的 `predictor` / `transformer` / `explainer`）声明多个 role
- role 名集合由各 Handler 在 §8 中约定，由 Handler 的 `Validate` 强制
- 单角色 Handler（`(native, deployment)` / `(native, statefulset)` 等）`Validate` 拒绝多 role 提交；多角色 Handler 在自身章节中明确开放节奏

调度域沿用 K8s PodSpec 扁平惯例直接放在 `spec.scheduling` 下，不再额外包一层 `placement`，与 mljob-operator 同构。

**Service 不引入独立 koordinator backend**：与 MLJob 不同，service 是常驻 + 弹性扩缩 workload，不应默认获得"所有副本同时调度"的 gang 语义，故 `native` 直接走 K8s 原生 Deployment / StatefulSet，不引入额外的 backend 维度。**但所有 native Service 与 KServe 派生的 Pod 仍强制走 koord-scheduler 并消耗 ElasticQuota**：Handler 在 podSpec 模板上设置 `schedulerName: koord-scheduler` 并打 label `quota.scheduling.koordinator.sh/name=axisml-<tenant>-<pool>-<quota>`，让 ElasticQuota plugin 自动按运行 Pod 把用量计入 `status.used`——**无需为 service 创建任何 PodGroup**（Koordinator ElasticQuota 通过 Pod label 直接关联，不依赖中间 PodGroup）。`spec.scheduling.quota` 因此既参与硬约束（koord-scheduler 调度时校验 ElasticQuota.max）也参与回流（Compute 通过 ElasticQuota.status.used 算出 `quotas.used`），无需 Compute 自行合成。若某类在线服务确实需要 gang / role-level gang（如 PD 分离要求 prefill 与 decode 成组启动），应作为后续显式设计，而不是复用 native 单角色默认行为。`(kserve, *)` 的 Pod 由 KServe 自身派生，本 Handler 通过写入 `InferenceService.spec.predictor.schedulerName` 与 `spec.predictor.labels` 透传 schedulerName + quota label，依赖 KServe 把它们传递到派生 Pod（KServe `PredictorSpec` 内联 `corev1.PodSpec` 与 `ComponentExtensionSpec`，因此这两个字段都直接位于 `spec.predictor` 下；详见 §8.3 / [infra.md §8.3](infra.md)）。

**与 MLJob 的差异点**：

- 顶层 `modelRef`：service 一等字段，指向 Artifacts model version；Handler 据此把模型工件解析为容器侧的位置（环境变量 / volume mount / KServe `storageUri` 等）
- `roles[*].template.ports[]`：与 K8s `PodSpec.containers[].ports` 同源约定。每个 role 是一个独立的 Deployment / StatefulSet（或 InferenceService 内的 component），各自的容器端口属于该 role 自身——这与多 role 拓扑（KServe transformer/explainer、PD 分离的 prefill/decode/router）天然一致。Handler 据此为每个 role 派生一个 K8s Service（targetPort=containerPort）。早期方案曾把 `ports[]` 放在 spec 顶层，是单 role 退化形态下的便捷写法，但在多 role 模型下"顶层 ports 到底属于哪个 role"无法回答，故下沉
- 顶层 `route`：可选；与 Gateway API `HTTPRoute` 同源命名。当 `enabled=true` 时由 Handler 创建 namespaced `HTTPRoute`（搭配 Envoy Gateway 的 `SecurityPolicy` / `BackendTrafficPolicy`）实现自助外部入口，`backendRefs` 指向 `route.targetRole` 对应的 K8s Service，详见 §8.1。`route.enabled` 还会切换 `status.endpoint` 的语义：`false` 时为集群内 Service DNS、`true` 时为外部 URL（详见 §4）。`(kserve, *)` Handler 自带 Route 机制，不接受 `route.enabled=true`
- `runPolicy` 字段集合不同：service 是常驻 workload，**没有** `suspend` / `activeDeadlineSeconds` / `ttlSecondsAfterFinished` / `backoffLimit`；改为 `progressDeadlineSeconds`（rollout 进度超时，与 K8s Deployment 同名字段语义一致）

#### 3.2 spec 结构

```yaml
apiVersion: axisml.io/v1alpha1
kind: MLService
spec:
  # ── 后端选择（创建后不可变；dispatcher 路由依据）─────────────────────
  backend:
    name: native              # 必填: native | kserve | custom
                              #      （kubeflow-trainer 仅用于 MLJob）
    engine: deployment        # 必填: 语义随 backend 而定（见 §8）；engine 与目标 CR 1:1 映射
                              #   native: deployment | statefulset
                              #   kserve: inference | llminference
                              #          （inference → InferenceService CR；llminference → LLMInferenceService CR；
                              #           runtime 由 backend.config.runtime 在 inference 下选择）
                              #   custom: 任意名（由 backend.config 描述目标 GVK）
    config: {}                # 可选: 该 (backend, engine) 元组特有的 schemaless 配置

  # ── 调度域（由 Compute 从 Quota / ResourcePool / ResourceUnit 合成注入）──
  scheduling:
    quota: string             # 必填: ElasticQuota CR 名（axisml-<tenant>-<pool>-<quota>，与 Compute Quota 1:1 映射）
    priorityClass: string     # 可选: K8s PriorityClass 名
    nodeSelector: {}          # Compute 按 compute.md §6.2.3 合并 pool + unit 后注入
    tolerations: []           # 来自 ResourcePool

  # ── 模型引用（service 特有，指向 Artifacts）─────────────────────────
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

#### 3.3 字段归属与不可变性

| 字段路径 | 写入方 | 创建后可变？ |
| --- | --- | --- |
| `metadata.name` / `namespace` / `labels[axisml.io/*]` | Compute | 否 |
| `spec.backend.name` / `spec.backend.engine` | Compute（默认 `{native, deployment}`） | **否**；dispatcher 拒绝并写 `status.message` |
| `spec.backend.config` | Compute（默认 `{}`） | 否（仅允许 `roles[*].replicas` 通过 `/scale` 变更；config 热更新见 §11） |
| `spec.scheduling.quota` / `priorityClass` / `nodeSelector` / `tolerations` | Compute（合并 Quota + Pool + Unit） | 否 |
| `spec.modelRef` | 用户提交 | 否（更换模型版本走重建） |
| `spec.roles[*].name` / `template.*`（含 `ports[]`，除 resources） | 用户提交 | 否 |
| `spec.roles[*].template.resources` | Compute（注入 ResourceUnit） | 否 |
| `spec.roles[*].replicas` | API（`/scale` 触发） | **是**（扩缩容路径专用） |
| `spec.runPolicy.progressDeadlineSeconds` | 用户提交 | 否 |
| `spec.route`（整块，含 `enabled` / `targetRole` / `portName` / `hostname` / `path` / `auth` / `rateLimit` / `timeout`） | 用户提交 | 否（不可变；mutable 演进见 §11） |

**默认值注入**：用户未指定 `spec.backend` 时，Compute 写 CR 时显式补 `{name: native, engine: deployment}`；`backend.config` 默认空对象 `{}`。dispatcher 不接受 `backend.name` 或 `backend.engine` 为空。

**`spec.route` 与 backend 的兼容性**：`(kserve, *)` Handler 在 `Validate` 中拒绝 `spec.route.enabled=true` 的提交（KServe `InferenceService` 自带对外 Route，避免双管）；`(native, *)` 与 `(custom, *)` 接受。详见各 Handler 章节。

**与 compute.md `services.replicas` 的兼容**：[compute.md §6.3.2](compute.md) 中的 `services.replicas` 字段在单 role 约定下定义为 `spec.roles[0].replicas`；`/scale` API 在 CR 侧 patch path 写 `spec/roles/0/replicas`。多 role 独立扩缩的契约扩展见 §11。

**CRD schema 现状**：当前 CRD 的 `spec` / `status` 用 `x-kubernetes-preserve-unknown-fields: true`，重新设计字段无需 CRD bump；待行为稳定后再启用 OpenAPI schema 严格校验（§11）。

**status subresource 要求**：`MLService` CRD 必须启用 `subresources.status`，保证 dispatcher 只写 `status`、Compute 只写 `metadata` / `spec` 的边界能由 Kubernetes API Server 隔离。当前 `deploy/helm/axisml-system/crds/mlservice-crd.yaml` 尚未声明该 subresource，属于实现对齐项；本文档先锁定契约，不在本次修订中修改 CRD 文件。

### 4. Status 契约

```yaml
status:
  observedGeneration: int64     # Handler reconcile 自洽用；Compute 不强消费
  phase: Pending | Ready | Degraded | Failed   # ← Compute 消费的主状态字段
  message: string               # 错误或状态附加信息（Compute 透传到 services.message）
  endpoint: string              # 单一服务地址（operator 写入 status；Compute 回流到 services.endpoint）：
                                #   - native/custom 且 route.enabled=false（默认）→ K8s Service DNS（<svc>.<ns>.svc.cluster.local:<port>）
                                #     ClusterIP Service / headless Service 共用此格式
                                #   - native/custom 且 route.enabled=true            → AxisML Gateway 外部 URL（形如 https://<hostname><path>）
                                #   - kserve backend → KServe 自带 route/status.url 暴露的 URL（不接受 spec.route.enabled=true）
                                # role 选择：native/custom 单 role 取唯一 role 的 Service；多 role 取 spec.route.targetRole；
                                #          未设置 spec.route 的多 role 场景由各 Handler 在 §8 中约定主 role；
                                #          kserve 取后端 CR 自带的主入口
                                # 端口选择：native/custom route.enabled=true 时按 route.portName；否则取主 role.template.ports[]
                                #          中 name="http" 的端口；不存在时取 ports[0] 并加 warning condition
  readyReplicas: int            # 主 role（单 role 约定下即 roles[0]）就绪副本聚合（Compute 回流到 services.ready_replicas）
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

**phase 枚举冻结为四态**——`Pending | Ready | Degraded | Failed`。新增 phase 必须 CRD schema 与 Compute 双侧同步演进。Compute 的状态映射规则（与 [compute.md §6.3.2](compute.md) 对齐）：

| MLService status.phase | services.status | 终态 |
| --- | --- | --- |
| `Pending` | `Pending` | 否 |
| `Ready` | `Ready` | 否 |
| `Degraded` | `Degraded` | 否，可恢复 |
| `Failed` | `Failed` | 否，可恢复（自愈） |

**`Pending / Ready / Degraded / Failed` 均为非终态**——operator 自愈（重建失败 Pod、健康检查恢复）后 Informer 回流可让 `Failed → Degraded → Ready` 自然恢复；只有 `Deleted` 是 Service 的最终终态，由 Compute Informer 在观察到 CR DELETE 事件后基于 PG 当前 `status` 推导（详见 [compute.md §5.3 / §5.4](compute.md)），不由 operator 产出。

`conditions` 与 `roles[]` 是给 UI 与运维用的 observability 字段，Compute 不消费。这样既保留了 K8s 标准实践（`metav1.Condition`、per-role 副本聚合）又不污染 Compute 状态机的简洁性。

跨 Handler 的 phase 映射规则原则：所有 Handler 在 `MapStatus` 中负责把后端原生状态映射到这四态，映射表写入对应 Handler 章节（§8）。

### 5. 总体架构：Dispatcher + Handler

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

**Watch 拓扑**：Dispatcher 始终 watch MLService 主队列；每个 Handler 启动时通过 `WatchTargets()` 声明自己关心的底层资源类型（Deployment、StatefulSet、Service、HTTPRoute、InferenceService …），由 dispatcher 统一建立 watch（controller-runtime `Watches()`），事件通过 `ownerReference` 反查回 MLService 后再交给对应 Handler 的 `MapStatus`。

**注册方式**：编译期注册（每个 handler 包 `init()` → 全局 registry）；不引入运行时插件加载（plugin / wasm / 外部 grpc）——若未来需要 "运行时安装新后端"，再演进为独立 operator binary 路由模式。

**未注册组合的兜底**：dispatcher 收到一条 `(backend, engine)` 无 handler 的 MLService → 写 `status.phase=Failed`、`status.message="no handler for backend=X engine=Y"`，不创建任何底层资源。

### 6. Reconcile 生命周期

按事件源切分 dispatcher 与 Handler 的职责：

| 事件 | Dispatcher 行为 | Handler 行为 |
| --- | --- | --- |
| MLService ADD（首次创建） | 路由到 Handler；调用 `Validate(spec)`，校验失败写 `status.phase=Failed` | `Reconcile(ctx, mlService)` 创建底层资源，设置 `ownerReference: MLService` |
| MLService UPDATE（仅 `roles[*].replicas` 变更，来自 `/scale`） | 路由 | `Reconcile` 透传为后端资源副本调整；不重建 Pod |
| MLService UPDATE（其他 spec 字段变更） | 校验 `backend.{name, engine}` 不变；其他字段变更属于约束违反，写 `status.message` 拒绝 | 不动 |
| MLService DELETE | 不阻断 | 一般依赖 ownerReference 级联清理；Handler 仅清理跨 namespace / 外部副作用（外部存储句柄、跨集群资源等） |
| 底层资源事件（Deployment / Service / PodGroup / HTTPRoute / 第三方 CR） | 通过 ownerReference 反查到 MLService 后路由，并从 informer cache 组装同一 MLService 下的相关子资源快照 | `MapStatus` 基于快照纯函数计算新 phase；dispatcher 把结果合并写入 `status` |

**关键约束**：

- Handler **不引入 finalizer**；ownerReference 级联清理是默认路径
- `MapStatus` 必须是纯函数（不发起 K8s 调用）；它可以读取 dispatcher 从 informer cache 传入的 MLService spec 与子资源快照，便于测试并保持状态回流路径可复现
- Handler 不能在 `Reconcile` 中直接写 `status`；所有 status 变更必须经过 `MapStatus`，由 dispatcher 统一合并写入，保证回流路径单一
- dispatcher 通过 `status` subresource 做 JSON merge patch（或 server-side apply）并用 `resourceVersion` 冲突重试；当前 CRD 仍是 schemaless，不能依赖 CRD strategic merge patch 或 list-map schema，`conditions[]` 必须由 dispatcher 在代码里按 `type` 去重后整体写回

**Pod 模板注入约定**（跨 Handler 通用，体现 [infra.md §8.3](infra.md) 的 Quota 全覆盖不变式）：

| Pod 字段 / Label | 必填 | 取值 | 用途 |
| --- | --- | --- | --- |
| `spec.schedulerName` | 是 | `koord-scheduler` | 所有 AxisML workload Pod 一律走 koord-scheduler；不允许任何 backend 让 Pod 落到默认 kube-scheduler 上 |
| label `quota.scheduling.koordinator.sh/name` | 是 | `<spec.scheduling.quota>` | Koordinator 原生 quota 关联 label；ElasticQuota plugin 据此把该 Pod 计入 `status.used` |
| label `axisml.io/service-id` | 是 | `services.id`（UUID） | 反查 MLService，与 CR 上同名 label 一致 |
| label `axisml.io/role` | 是 | role 名（如 `predictor` / `transformer` / `explainer`） | 区分多角色拓扑下的 Pod |
| label `axisml.io/quota` | 是 | Compute Quota bare name（取自 MLService CR `metadata.labels["axisml.io/quota"]` 透传，**与 `quota.scheduling.koordinator.sh/name` 取值不同**：前者是裸名如 `training`，后者是 ElasticQuota 全名如 `axisml-<tenant>-<pool>-training`） | AxisML 自有审计 / 查询；不参与调度 |
| label `axisml.io/replica-index` | 否 | role 内 0-based 序号 | 副本身份天然稳定时建议透传：`(native, statefulset)` 取 `apps.kubernetes.io/pod-index`；`(native, deployment)` / KServe autoscaling pod 集合等无稳定身份场景一律省略 |

前 5 项必填；`replica-index` 是可观测增强，缺失时按 pod 名定位。MLService 当前无 logs API，本约定主要服务于运维排障与 metrics 聚合。

**KServe 派生 Pod 的注入路径**：`(kserve, *)` Handler 不直接控制 podSpec；通过写入 `InferenceService.spec.predictor.schedulerName` + `spec.predictor.labels` 让 KServe 透传到派生 Pod 的 `spec.schedulerName` 与 `metadata.labels`（`PredictorSpec` 内联 `corev1.PodSpec` 与 `ComponentExtensionSpec`，所以两者都是 `spec.predictor` 的直接字段）。这要求 KServe 版本支持透传这两个字段（前置依赖见 §8.3）。

**`spec.route` 派生资源**：当 `enabled=true` 时，Handler 在租户 namespace 内创建 / 更新以下资源（统一打 `axisml.io/service-id` label，并设置 `ownerReference: MLService`，靠级联清理删除，不引入 finalizer）：

- `HTTPRoute`（`gateway.networking.k8s.io/v1`）：`parentRefs` 指向 `axisml-gateway`（跨 namespace 引用通过 `ReferenceGrant` 授权，由 infra chart 准备），`backendRefs` 指向 `route.targetRole` 对应的 K8s Service
- `SecurityPolicy`（`gateway.envoyproxy.io/v1alpha1`）：仅当 `auth.type != none` 时创建，`targetRefs` 指向上面的 HTTPRoute
- `BackendTrafficPolicy`（`gateway.envoyproxy.io/v1alpha1`）：仅当 `rateLimit` 或 `timeout` 非空时创建，`targetRefs` 指向上面的 HTTPRoute

### 7. Handler 接口契约

所有 Handler 必须实现以下行为（语言无关，描述责任）：

| 方法 | 责任 |
| --- | --- |
| `Key()` | 返回 `(backend, engine)` 元组，与 `spec.backend.{name, engine}` 对齐；用作注册表的主键 |
| `Validate(spec)` | 校验通用字段 + `backend.config` + role 集合（数量、命名）；纯函数，未来可被 admission webhook 复用 |
| `Reconcile(ctx, mlService)` | 创建 / 更新底层资源；幂等；通用字段由 Handler 自己注入到对应位置；处理 `spec.roles[*].replicas` 的扩缩容 |
| `MapStatus(snapshot)` | 把 MLService spec 与同 owner 子资源快照映射回 §4 的四态 phase + readyReplicas + 单一 endpoint + message + conditions + roles 副本聚合；不得发起 K8s API 调用 |
| `Cleanup(ctx, mlService)` | 删除底层资源。一般依赖 ownerReference 自动级联，Handler 仅负责需要主动清理的副本（跨 namespace 资源、外部存储句柄） |
| `WatchTargets()` | 声明本 Handler 需要 watch 的底层资源 GVK 列表，dispatcher 启动时统一建立 watch |
| `RequiredRBAC()` | 声明本 Handler 需要的 ClusterRole 规则；启动时聚合到 operator ServiceAccount |

**幂等性细则**：

- `Reconcile` 多次调用相同 spec 不重建底层资源；只有语义字段变化才触发更新
- 重复 patch 相同 `roles[*].replicas` 不得重建底层资源；只调整副本数
- `Cleanup` 对已删除资源返回 nil
- `MapStatus` 是纯函数（不发起 K8s 调用），状态推进只依赖 dispatcher 传入的快照

**Scale 透传义务**：每个 Handler 必须把 `roles[*].replicas` 透传为后端原生扩缩——

- `(native, deployment)` → patch `Deployment.spec.replicas`
- `(native, statefulset)` → patch `StatefulSet.spec.replicas`
- `(kserve, *)` → patch `InferenceService.spec.predictor.{minReplicas, maxReplicas}`（具体策略见 §8.3）
- 不支持原生扩缩的 backend → 兜底为重建底层资源（应避免，作为最后手段）

**`spec.route` 增量职责**：

- `Reconcile`：根据创建时确定的 `spec.route.enabled` 与各子字段创建 / 保持上面三类派生资源；`spec.route` 不可变，删除主要依赖 MLService ownerReference 级联清理；`Validate` 拒绝 `(kserve, *)` 下的 `enabled=true`、拒绝多 role 但未指定 `targetRole` 的提交、拒绝多端口但未指定 `portName` 的提交
- `MapStatus`：把 HTTPRoute `Accepted` / `ResolvedRefs` condition 翻译为 `status.endpoint`（按 §4 端口选择规则填写外部 URL）与 `status.conditions` 的 `Available` 条件；HTTPRoute `Accepted=False` 视同后端未就绪，应让 `phase=Degraded` 并把失败原因写入 `message`

**Status 写入约束**：Handler 只能通过 `MapStatus` 的返回值影响 `status`；不能在 `Reconcile` 中直接 `status` 写盘。dispatcher 统一合并 `phase` / `message` / `endpoint` / `readyReplicas` / `conditions` / `roles[]` 写入 CR，保证 [§2 写路径契约](#2-与-compute-的写路径契约) 中的 "status 单向权威"。校验失败、不可变字段被修改、未注册 Handler 等 dispatcher 级错误也由 dispatcher 写 `status.phase=Failed` 与 `status.message`，不交给 Handler 直接写。status patch 采用 JSON merge patch（或 server-side apply）+ `resourceVersion` 冲突重试；`conditions[]` 由 dispatcher 按 `type` 去重后整体写回。

### 8. 内置 Handler

每个 Handler 章节统一按以下小节组织：**底层资源 / `backend.config` / 通用字段映射 / Status 映射 / Scale / RBAC**。

#### 8.1 `(native, deployment)`

底层用 K8s 原生 Deployment + Service。所有 Pod 走 koord-scheduler 并通过 Pod label 计入 ElasticQuota，**不**创建 PodGroup（Koordinator ElasticQuota 通过 Pod label 直接关联 quota，不需要中间 PodGroup）。

**底层资源**：

- 必填且仅一个 role（`name=predictor`）；`Validate` 拒绝多 role 提交或其他 role 名
- 每个 MLService 创建一个 K8s `Deployment` 与一个 K8s `Service`：
  - `Service` 端口由 `roles[predictor].template.ports[]` 派生（`targetPort=containerPort`）
  - Deployment Pod 模板上设置 `schedulerName: koord-scheduler`，并打 §6 列出的 5 项必填 label
- 当 `spec.route.enabled=true` 时追加 `HTTPRoute` + 可选的 `SecurityPolicy` / `BackendTrafficPolicy`（与 §6 派生资源说明一致）
- Deployment / Service / 派生路由资源设置 `ownerReference` 指向 MLService，保证 MLService 删除后底层资源级联清理；Pod 删除后 ElasticQuota `status.used` 自然释放该 Service 的用量
- operator 不读写 ElasticQuota CR（ElasticQuota 由 Compute 独占维护）

**Pod label**：见 §6 表（`schedulerName: koord-scheduler` + 5 项必填 label）。Deployment Pod 没有稳定 index（ReplicaSet 用 hash 后缀，扩缩容/滚动更新都换 Pod 名），按 §6 约定省略 `axisml.io/replica-index`。

**`backend.config`**：本 Handler 不消费；非空时 `Validate` 返回 warning，不报错（为 future 字段预留），由 dispatcher 记录 event 或 warning condition。

**通用字段映射**：

| MLService 字段 | Deployment / Service / 派生路由资源落点 |
| --- | --- |
| `roles[predictor].template.image` / `imagePullPolicy` / `command` / `args` / `env` / `envFrom` / `workingDir` | Deployment Pod 主容器同名字段 |
| `roles[predictor].template.ports[]` | Deployment Pod 主容器 `ports` + K8s Service `spec.ports`（`targetPort` 取 `containerPort`） |
| `roles[predictor].template.resources.requests` / `limits` | Deployment Pod 主容器同名字段 |
| `roles[predictor].replicas` | `Deployment.spec.replicas` |
| `spec.scheduling.quota` | Pod `spec.template.metadata.labels[quota.scheduling.koordinator.sh/name]`（ElasticQuota 全名 `axisml-<tenant>-<pool>-<quota>`）；不创建 PodGroup |
| MLService `metadata.labels[axisml.io/quota]` | Pod `spec.template.metadata.labels[axisml.io/quota]`（bare quota name，由 Compute 在 MLService CR 上设置后由 Handler 透传） |
| 调度器选择 | Pod `spec.template.spec.schedulerName=koord-scheduler`（恒定） |
| `spec.scheduling.priorityClass` | Pod `spec.priorityClassName` |
| `spec.scheduling.nodeSelector` / `tolerations` | Pod 同名字段 |
| `spec.modelRef` | Artifacts client 解析为模型工件 URI，注入为环境变量 `AXISML_MODEL_URI`（containerPath / volume mount 形态留待后续策略） |
| `spec.route.targetRole` | 选取 HTTPRoute `backendRefs.name` 指向的 K8s Service（单 role 时省略，自动取 `predictor`） |
| `spec.route.portName` | HTTPRoute `backendRefs.port`（解析为 `targetRole` Service 中对应的端口） |
| `spec.route.hostname` / `path` | `HTTPRoute.spec.hostnames` / `rules[].matches[].path.value`（path 默认 `/`） |
| `spec.route.auth` | `SecurityPolicy.spec.{jwt | apiKeyAuth}`，`targetRefs` 指向上面 HTTPRoute |
| `spec.route.rateLimit` / `timeout` | `BackendTrafficPolicy.spec.rateLimit` / `timeout`，`targetRefs` 指向上面 HTTPRoute |
| `spec.runPolicy.progressDeadlineSeconds` | `Deployment.spec.progressDeadlineSeconds` |

**Status 映射**（沿用 [compute.md §6.3.2](compute.md) 规则，从 Deployment `status` 推导）：

| 条件 | MLService phase |
| --- | --- |
| `desired_replicas == 0` | `Pending`（扩缩至 0，视为待调度 / 停用） |
| `ready_replicas == 0 && desired_replicas > 0` 且 rollout 仍在推进中（`Progressing=True`，未超过 `progressDeadlineSeconds`，无 `ReplicaFailure`） | `Pending` |
| `ready_replicas == desired_replicas && desired_replicas > 0` | `Ready` |
| `0 < ready_replicas < desired_replicas` | `Degraded` |
| `ready_replicas == 0 && desired_replicas > 0` 且 Deployment 超过 `progressDeadlineSeconds` 或出现 `ReplicaFailure` / `ProgressDeadlineExceeded` | `Failed` |

`endpoint` 按 §4 二分规则填写：

- `spec.route.enabled=false`（默认）→ `<svc>.<namespace>.svc.cluster.local:<port>`，端口按 §4 选择规则（`roles[predictor].template.ports[]` 中 `name=http` 优先，否则 `ports[0]` 并加 warning condition）
- `spec.route.enabled=true` → `https://<hostname><path>`，从 HTTPRoute 派生；若 `hostname` 为空且 Gateway 只提供 wildcard/default host，Handler 必须返回可被客户端访问的具体 URL，无法确定时保持内部 Service DNS 并写 warning condition

`readyReplicas` 取 Deployment `status.readyReplicas`；`status.roles[predictor]` 聚合 desired / ready 副本数。

**`spec.route` 的 phase 影响**：`enabled=true` 且 HTTPRoute `Accepted=False`（或 `ResolvedRefs=False`）时——即 Deployment 已就绪但外部入口未生效——映射为 `phase=Degraded`，`message` 写明 HTTPRoute 拒绝原因；同时 `endpoint` 暂时回退为内部 Service DNS，避免暴露未就绪的外部 URL。HTTPRoute 就绪 + Deployment 就绪 → `phase=Ready`，`endpoint` 切换为外部 URL。

**Scale**：patch `Deployment.spec.replicas`；不重建 Pod。

**RBAC**：

- 基础：`deployments.apps` / `services` / `pods` / `events` 的 `create / get / list / watch / update / patch / delete`
- `spec.route` 派生资源：`httproutes.gateway.networking.k8s.io` / `securitypolicies.gateway.envoyproxy.io` / `backendtrafficpolicies.gateway.envoyproxy.io` 的 `create / get / list / watch / update / patch / delete`
- `secrets` 的 `get / list / watch`（仅当 `spec.route.auth.type=apiKey` 引用 Secret 时）
- 不需要 ElasticQuota / PodGroup 的 RBAC（ElasticQuota 由 Compute 独占；PodGroup 在 service 路径下不创建）

#### 8.2 `(native, statefulset)`

为有状态推理（在线 KV cache、模型分片、节点身份固定的副本）预留。底层用 K8s `StatefulSet` + headless Service，副本身份稳定；其余约束沿用 §8.1。

**底层资源**：

- 必填且仅一个 role（`name=predictor`）
- 每个 MLService 创建一个 `StatefulSet` 与一个 headless `Service`（`spec.clusterIP=None`）；StatefulSet Pod 模板上设置 `schedulerName: koord-scheduler` + §6 列出的 5 项必填 label。**不**创建 PodGroup（同 §8.1）
- Pod 通过 `<pod>.<svc>.<namespace>.svc.cluster.local` 直连
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

**通用字段映射**：与 §8.1 相同，`roles[predictor].replicas` 落到 `StatefulSet.spec.replicas`；`roles[predictor].template.ports[]` 落到 StatefulSet 主容器 `ports` + headless Service `spec.ports`；补充 `volumeClaimTemplates` 与 `serviceName` 字段；其余字段沿用 §8.1 表格（含 `schedulerName: koord-scheduler` 与 quota label 注入）。

**`spec.route` 行为**：与 §8.1 一致；HTTPRoute `backendRefs` 指向同一份 headless Service（headless Service 也可作 Gateway API backendRef 目标，由 EndpointSlice 解析具体 Pod）。

**Status 映射**：从 `StatefulSet.status` 推导，规则与 §8.1 同构（用 `readyReplicas` / `replicas` 替换 Deployment 同名字段）；`endpoint` 同样按 §4 二分规则填写（headless Service 的 `<svc>.<ns>.svc.cluster.local:<port>` 解析为 EndpointSlice 中所有就绪 Pod 的 IP），`spec.route` 对 phase 的影响与 §8.1 一致。

**Scale**：patch `StatefulSet.spec.replicas`；副本身份保留，扩容时新 index 追加，缩容时按高 index 优先终止。

**RBAC**：

- 基础：`statefulsets.apps` / `services` / `pods` / `events` 的 `create / get / list / watch / update / patch / delete`
- `spec.route` 派生资源：与 §8.1 同
- `secrets` 的 `get / list / watch`（仅当 `spec.route.auth.type=apiKey` 引用 Secret 时）
- 不需要 ElasticQuota / PodGroup 的 RBAC（同 §8.1）

#### 8.3 `(kserve, inference)`

将 MLService 翻译为 KServe [`InferenceService`](https://kserve.github.io/website/) CR（`serving.kserve.io/v1beta1`）。这是 KServe 通用 ML 服务路径——predictor 内的具体 runtime（NVIDIA Triton / [vLLM](https://docs.vllm.ai/) / TF Serving / TorchServe / sklearn / huggingface 等）由 `backend.config.runtime` 选择，转化为 KServe `(Cluster)ServingRuntime` 引用或 `predictor.model.modelFormat` 声明。

**前置依赖**：集群已安装 KServe，且版本支持 `InferenceService.spec.predictor.schedulerName` 与 `spec.predictor.labels` 透传到派生 Pod（`PredictorSpec` 内联 `corev1.PodSpec` 与 `ComponentExtensionSpec`，是 KServe v1beta1 的标准契约；这是 §6 quota 全覆盖契约的硬要求，落地 KServe 版本以 axisml-infra 安装时 pin 的 stable 版本为准）。其 RBAC 与 CRD 由 KServe chart 单独管理，本 Handler 仅需要 `inferenceservices.serving.kserve.io` 的 `create / get / list / watch / update / patch / delete`，外加各 runtime 对应 `(Cluster)ServingRuntime` 的 `get / list / watch`。

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
  storageUri: string              # 模型工件位置；可由 Artifacts 通过 modelRef 自动解析
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
- `spec.modelRef` → 通过 Artifacts 解析为 `predictor.storageUri`（runtime=triton 时也可解析为 `triton.modelRepository`；runtime=vllm 时优先解析为 `vllm.model`，缺失时回退到 `storageUri`）
- `spec.scheduling.quota` → 写入 `InferenceService.spec.predictor.schedulerName=koord-scheduler` 与 `InferenceService.spec.predictor.labels` 中的 `quota.scheduling.koordinator.sh/name=axisml-<tenant>-<pool>-<quota>` + `axisml.io/quota=<quota-name>`；KServe `PredictorSpec` 内联 `corev1.PodSpec` 与 `ComponentExtensionSpec`（参见 kserve `pkg/apis/serving/v1beta1/predictor.go`），因此 `schedulerName` 直接位于 `spec.predictor` 下，组件级 `labels` 也直接位于 `spec.predictor.labels`（KServe 会把它们透传到派生 Pod 的 `spec.schedulerName` 与 `metadata.labels`），使 KServe Pod 与 native Pod 一样消费 ElasticQuota
- `spec.scheduling.priorityClass` / `nodeSelector` / `tolerations` → predictor 同名字段
- `spec.runPolicy.progressDeadlineSeconds` → KServe 暂无对等字段，Handler 在 `Validate` 中返回 warning，dispatcher 记录 event 或 warning condition，不阻塞创建
- `spec.route` → **不支持**；KServe `InferenceService` 自带对外 Route，Handler 在 `Validate` 中拒绝 `spec.route.enabled=true`，由 dispatcher 写 `status.phase=Failed` 与 `status.message="spec.route not supported on (kserve, *) backend; KServe manages its own route"`

**runtime 专属约束**（由 `Validate` 强制）：

- `runtime=vllm`：`roles[predictor].template.resources.requests["nvidia.com/gpu"]` 必须等于 `config.vllm.tensorParallelSize × pipelineParallelSize`
- `runtime=huggingface`：`config.huggingface.task` 必填
- 其他 runtime 的强制项由 ServingRuntime 自身校验，本 Handler 透传

**扩展 transformer / explainer 角色**：映射至 `roles[transformer]` / `roles[explainer]`，字段映射镜像 predictor；开放节奏见 §11。

**Status 映射**：从 `InferenceService.status.conditions` 推导——

| InferenceService condition | MLService phase |
| --- | --- |
| `desired==0` | `Pending` |
| `Ready=False` / `PredictorReady=False` 且仍在创建或滚动更新中（未出现后端失败 condition） | `Pending` |
| `Ready=True` | `Ready` |
| `PredictorReady=False` 且 `0 < ready < desired` | `Degraded` |
| `Ready=False` 且 `ready==0 && desired>0`，并且 KServe condition 明确失败或超过进度期限 | `Failed` |

`endpoint` 取 `InferenceService.status.url`（KServe 自带对外 Route，本 Handler 不接受 `spec.route.enabled=true`，因此 `endpoint` 单字段定义直接对齐 KServe 自带的外部 URL）。

**Scale**：patch `InferenceService.spec.predictor.{minReplicas, maxReplicas}`；具体取舍（min 跟随 / max 联动）由独立设计文档落地。

**Quota 与 autoscaling 的相互作用**：KServe scale-to-zero / 自动扩缩可能让实际副本数动态变化；Compute quota 按 `maxReplicas × requests` 上限计费（与 native 的 "replicas × requests" 线性记账一致），保证账面与运行时不打架；`runtime=vllm` 单副本 GPU 数还受 `tensorParallelSize × pipelineParallelSize` 影响。具体细节由独立设计文档落地（§11）。

#### 8.4 `(kserve, llminference)`

> **本节为占位设计**：KServe LLM API 的 GVK / CRD 字段路径仍在演进，落地以引入版本为准。本节当前只锁两件事——role 命名约定（`prefill / decode / router`）与 PD 分离骨架（`backend.config` 形状）；schema 详细字段、`Validate` 强制项、Status condition 名等待 KServe LLM API GA 后在 §11 单独成文。读者切勿把本节字段当作可直接实现的契约。

将 MLService 翻译为 KServe LLM 原生 CR `LLMInferenceService`（占位命名；KServe 社区围绕 LLM 原生服务的 GVK 仍在演进，候选名包括 `LLMInferenceService` / `InferencePool` / `LLMRoute` 等，**实际 GVK 以引入 KServe 版本时为准**）。该 engine 承载 LLM 在线服务相对 `InferenceService` 的额外能力——核心是 **PD 分离（disaggregated serving）**：prefill 与 decode 拆成独立角色独立扩缩，搭配 router 角色做请求分发与 KV cache 协调。

**前置依赖**：集群已安装 KServe LLM API（含 `LLMInferenceService` CRD 与对应 controller / runtime），并支持把 schedulerName / labels 透传到各 role 派生 Pod（与 §8.3 同；此约束在 KServe LLM API GA 后随版本 pin 落地）。本 Handler 需要 `llminferenceservices.serving.kserve.io`（占位）的 `create / get / list / watch / update / patch / delete`，外加 KV cache 协议相关 ConfigMap / Secret 的读取权限（细节随 KServe LLM API 落地补全）。

**Role 集合约定**（PD 分离骨架；具体 role 名以 KServe LLM API 落地为准）：

- `prefill`：长上下文处理（compute-bound）；replicas≥1；GPU 配置通常偏算力
- `decode`：token 生成（memory-bound）；replicas≥1；GPU 配置通常偏显存与互联带宽
- `router`：请求入口与 KV cache 协调；replicas≥1；承载 KServe LLM API 自带的对外入口（PD 拓扑里"对外端点"应指向 router 而非 prefill / decode 单体）

`Validate` 强制：role 名属于上述集合；至少存在 `prefill` 与 `decode`；`router` 是否强制必填取决于引入时选定的 KServe LLM API 与路由模型（具体节奏见 §11）。在当前占位契约下，`(kserve, llminference)` 与 §8.3 一样拒绝 `spec.route.enabled=true`，避免同时由 AxisML Gateway 与 KServe LLM router 管理入口。

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
- `spec.scheduling.quota` → 各 role 的 podSpec 注入 `schedulerName: koord-scheduler` 与 `quota.scheduling.koordinator.sh/name` + `axisml.io/quota` label（`LLMInferenceService` 中各 role 的 podSpec / metadata 字段路径以实际 GVK 为准）
- `spec.runPolicy.progressDeadlineSeconds` → KServe 暂无对等字段，Handler 在 Validate 中返回 warning，dispatcher 记录 event 或 warning condition，不阻塞创建
- `spec.route` → **不支持**（同 §8.3）；KServe LLM API 自带 router / Route 机制

**单副本 GPU 数约束**：`roles[prefill / decode].template.resources.requests["nvidia.com/gpu"]` 必须等于该 role 的 `tensorParallelSize × pipelineParallelSize`，由 `Validate` 强制。

**Status 映射**：参照 KServe LLM API 的 condition 集合落地，原则上沿用 §8.3 四态映射；具体 condition 名以 KServe LLM API 实现为准（§11 写明落地节奏）。`endpoint` 取 KServe LLM API 暴露的 router 入口（与 §8.3 取 `status.url` 同思路；具体字段路径以引入版本为准）。

**Scale**：分别 patch 各 role 在 `LLMInferenceService` 中的 `minReplicas` / `maxReplicas`；多 role 独立扩缩需要 §11 中的 `/scale` API 路径携带 role 名。

**Quota 与 autoscaling 的相互作用**：与 §8.3 一致，按 `Σ(role.maxReplicas × role.requests)` 计费；`prefillToDecodeRatio` 仅作为 autoscaler 参考，不参与配额校验。

#### 8.5 `(custom, *)`

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

由 custom Handler 通过 unstructured client 创建并跟踪。**仍受 §6 Pod 注入约定与 [infra.md §8.3](infra.md) Quota 全覆盖不变式约束**：custom Handler 必须保证最终落地的 Pod 模板上有 `schedulerName: koord-scheduler` 与 `quota.scheduling.koordinator.sh/name` label，否则 Validate 直接拒绝。完整 schema 与 unstructured 操作约定由独立设计文档落地（见 §11）。

**`spec.route` 在 custom Handler 下的语义**：由 `config.routeBackend`（在独立设计文档中定义）显式描述外部入口对接的目标 Service；未在 `config` 中 wire `spec.route` 时，Handler 应在 `Validate` 中拒绝 `spec.route.enabled=true`。

### 9. RBAC 聚合

operator binary 启动时遍历 registry，把每个启用 Handler 的 `RequiredRBAC()` 合并去重，生成本 binary 实际需要的 ClusterRole rules。Helm chart 通过 values 控制启用集合，渲染最小化 RBAC 而非全集——例如仅启用 `(native, *)` 时，集群无需安装 KServe；启用 `(kserve, *)` 才注入 KServe 与 ServingRuntime 的 RBAC。

### 10. 不变量与约束

- `spec.backend.{name, engine}` 创建后不可变；dispatcher 拒绝并写 `status.message`，admission webhook 后续接管
- `(backend, engine)` 元组未在 registry 注册 → MLService 直接进入 `Failed`，message 写明缺失原因
- 任一 Handler 的 `Reconcile` / `MapStatus` 必须保持 §2 列出的写路径契约——这是把 "插件" 安全嵌入 Compute Outbox 模型的根基
- Handler 不直接修改 ElasticQuota CR；ElasticQuota CR 由 Compute 独占维护
- Handler 不向 Compute PG 写入任何数据；状态全部经由 MLService `status` + Informer 回流
- **Handler 不引入 finalizer**；级联清理依赖 ownerReference + `Cleanup()`
- **`status.phase` 取值集合冻结为四态**（`Pending | Ready | Degraded | Failed`）；新增 phase 必须经 CRD schema 与 Compute 双侧同步演进
- **`spec.roles[*].replicas` 是允许变更的字段**（`/scale` 路径专用）；其余 spec 字段创建后不可变，dispatcher 检测到变更需写 `status.message` 拒绝
- 所有 Handler 派生的 Pod 必须满足 §6 Pod 注入约定的前 5 项必填字段（含 `schedulerName: koord-scheduler` 与 quota label）；缺失任一项视为契约违反，Validate 必须在创建前拦截
- **`spec.route` 创建后不可变**；mutable 演进作为后续设计文档预留（见 §11）
- **`(kserve, *)` Handler 不接受 `spec.route.enabled=true`**（KServe 自带 Route，避免双管）；`(native, *)` 接受；`(custom, *)` 由 `config.routeBackend` 自描述，未 wire 时拒绝
- **`spec.route` 派生的 `HTTPRoute` / `SecurityPolicy` / `BackendTrafficPolicy` 通过 `ownerReference` 级联清理**；Handler 不引入 finalizer
- **`status.endpoint` 是单一服务地址字段**（Compute 透传到 `services.endpoint`）：native/custom 且 `spec.route.enabled=false` 时为 K8s Service DNS（ClusterIP / headless 共用 `<svc>.<ns>.svc.cluster.local:<port>` 格式），native/custom 且 `enabled=true` 时为 AxisML Gateway 外部 URL（`https://<hostname><path>`），`(kserve, *)` 时为 KServe 自带 route/status.url 暴露的 URL；不再单独建 `status.externalUrl` 字段，避免 compute.md services 表 schema churn

### 11. 后续设计文档（不在本文档范围）

- `(native, statefulset)` Handler 的 `volumeClaimTemplates` / 灰度更新 / pod-index 寻址细节
- 多 role 接入的具体 Handler 落地：
  - `(kserve, inference)` 的 `transformer` / `explainer` 字段映射与状态映射
  - `(kserve, llminference)`（对应 `LLMInferenceService` 占位 GVK，最终以 KServe LLM API 落地版本为准）：vLLM disaggregated / llm-d / NVIDIA Dynamo 等场景下，`prefill` / `decode` / `router` 三类 role 的命名约定、KV cache 传输契约（NIXL / Mooncake / …）、`disaggregation.prefillToDecodeRatio` autoscaler 接入、`Validate` 中的多 role 必填规则、KServe 自带 router 与 AxisML `spec.route` 是否统一的演进方式
- KServe scale-to-zero 与 Compute quota 的精细交互模型（含 `maxReplicas × requests` 上限计费策略）
- `(custom, *)` Handler 的 `config` 完整 schema 与 unstructured 操作约定（含 `config.routeBackend` 与 `spec.route` 的对接细则）
- 多 role 独立扩缩容的 `/scale` API 扩展（路径中携带 role 名）
- `spec.route` 可变化路径（轮换 API key / 调整限流不需要重建 Service；Handler 侧需要识别哪些子字段可热更新、哪些必须重建派生资源）
- `spec.route` 与 KServe 自带 Route 的统一化（让 `(kserve, *)` 也支持 `spec.route` 而非依赖 KServe 内置 Route）
- Admission webhook：`spec.backend.{name, engine}` 不可变约束、`backend.config` 按 Handler 自带 schema 的统一校验
- Handler chart 的 values 控制（启用 / 禁用 backend，对应 RBAC 与 watch 的开关）
- CRD 严格 schema（启用 OpenAPI 校验，替换当前的 `x-kubernetes-preserve-unknown-fields`）

## 7. 测试

合并后的 L1 envtest 在 `components/operator/test/envtest/` 单一 Go module 中，单一 `TestMain` 注册三个 reconciler 到同一个 envtest manager，跑七个 test 文件（tenant 2 + mljob 3 + mlservice 2）。CRDPaths 是 `deploy/helm/axisml-system/crds` 与 `test/crds/external/`（vendored ElasticQuota / PodGroup / HTTPRoute）的并集。

L2 e2e 仍在 `test/e2e/`，通过部署后的 axisml-operator 与 MLPlatform/Compute API 一起跑端到端；e2e 不直接关心 operator 二进制名。

## 8. 升级路径

旧布局：

```
components/operators/{tenant,mljob,mlservice}-operator/
deploy/helm/axisml-system/templates/operators/{tenant,mljob,mlservice}-operator/
docs/system_design/operators/{tenant,mljob,mlservice}-operator.md
```

新布局：

```
components/operator/                                       # 单 module
deploy/helm/axisml-system/templates/operator/              # 单组模板
docs/system_design/operator.md                             # 总览 + 各 controller 详细设计
```

`helm upgrade`：旧 Deployment / SA / ClusterRole / ClusterRoleBinding（三份）由 Helm release 记录释放后会被删除并替换为新的合并版本，预期短暂 downtime（秒级）。镜像名 `ghcr.io/axisml/axisml-operator` 不变。

## 9. 相关引用

- [docs/system_design/overview.md §5.3](overview.md) 概述了 axisml-operator 在控制平面里的位置。
- [docs/system_design/compute.md](compute.md) 描述 ml-compute 与 operator 之间的 CR 写路径与状态回流。
- [docs/system_design/infra.md §8](infra.md) 给出 koord-scheduler / ElasticQuota / Gateway API 等基础设施依赖契约。
