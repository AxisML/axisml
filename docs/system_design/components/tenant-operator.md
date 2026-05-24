# AxisML Tenant Operator 详细设计

tenant-operator 是 AxisML 控制平面里负责"管理员域"资源落地的 Kubernetes operator 二进制。它持有一个 Manager，承载单一 Tenant controller，把 [cluster-manager](cluster-manager.md) 渲染下发的 `Tenant` CR 翻译为 Kubernetes 侧的 Namespace、Koordinator `ElasticQuota`、租户级初始化资源（Secret / ConfigMap / ServiceAccount / Role / RoleBinding），并把执行状态回流到 `Tenant.status`。

> **关于 Tenant CR 的权威**：Tenant CR 是 cluster-manager PG `tenants` 表的**派生产物**——由 cluster-manager 的 reconciler 通过 outbox + 双 hash 同步渲染下发（详见 [cluster-manager.md §4](cluster-manager.md#4-写路径与同步)）。外部 `kubectl create / edit tenant` **不受支持**：MVP 阶段 cluster-manager reconciler 会把外部 spec 漂移擦回 PG 期望态；阶段二 admission webhook 上线后将硬阻断。tenant-operator 自身仍把 Tenant CR 视作输入 watch 对象，只是上游 producer 收敛到 cluster-manager 一家——这一变更对本文档其余章节（reconcile 行为、字段映射、状态推导）**无任何影响**。

| Controller | CRD（`axisml.io/v1alpha1`） | Scope | 架构 | 状态机 | 主要外部依赖 |
| --- | --- | --- | --- | --- | --- |
| Tenant ([§4](#4-tenant-controller)) | `Tenant` | Cluster | 单 reconciler（无 dispatcher） | `Active \| Suspended \| Failed` | Koordinator ElasticQuota |

**文档组织**：

- **Part I — 运行时框架**（§1 架构总览 + §2 运行时契约）：单一 Deployment / Manager 的运维契约（Scheme、Cache、Flag、RBAC、Helm values）。
- **Part II — 与 Cluster Manager 的协作契约**（§3）：CR 写路径、metadata 约定、Reconcile 行为约束。
- **Part III — Controller 详细设计**（§4 Tenant）：CRD 字段、状态推导、底层资源管理、共享 Namespace 语义、RBAC。
- **Part IV — 实施与验证**（§5 实现路径、§6 测试、§7 相关引用）。

---

## Part I — 运行时框架

> 本部分描述 tenant-operator binary 与 Kubernetes Manager 的运维契约：Scheme 注册、Cache 过滤、CLI flag、RBAC 与 Helm values。

## 1. 架构总览

```
┌──────────── tenant-operator (one Pod, leader-elected) ────────────┐
│                                                                    │
│  ctrl.Manager (scheme: clientgoscheme + axisml.tenant +            │
│                scheduling.sigs.k8s.io ElasticQuota)                │
│  Lease: axisml-tenant-operator.axisml.io                           │
│                                                                    │
│  ┌──────────────────────┐                                          │
│  │ Tenant Reconciler    │                                          │
│  │ (single, no          │                                          │
│  │  dispatcher)         │                                          │
│  └──────────────────────┘                                          │
│              │                                                     │
│              ▼                                                     │
│   Namespace, ElasticQuota,                                         │
│   Secret / ConfigMap / SA /                                        │
│   Role / RoleBinding                                               │
└────────────────────────────────────────────────────────────────────┘
```

Tenant controller 不存在多后端实现，没有 dispatcher/handler 分层；所有 Tenant CR 由单一 reconciler 处理。

## 2. 运行时契约

### 2.1 Scheme 注册

```go
clientgoscheme.AddToScheme(scheme)           // core, apps, rbac, coordination
schedulingv1alpha1.AddToScheme(scheme)       // ElasticQuota（Koordinator vendored）
tenant_v1alpha1.AddToScheme(scheme)          // Tenant
```

Tenant CRD 的 Go 类型定义在 `components/tenant-operator/api/v1alpha1/`，与 compute-operator 的 MLJob / MLService 子包互不依赖。

### 2.2 Cache 选择性过滤

tenant-operator 派生的所有子资源（Secret / ConfigMap / ServiceAccount / Role / RoleBinding / ElasticQuota）都打上 `axisml.io/managed-by=tenant-operator` label，避免缓存全集群同类对象。本 binary 不承载 MLJob / MLService controller，因此可以用 `cache.Options.DefaultLabelSelector` 全局过滤；保留 `ByObject` 形式只是为后续如果有"非 managed 类型也要 watch"的扩展留位：

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

`Tenant` CR 自身不打 `managed-by` label（cluster-scoped、无歧义），不进 `ByObject` 表。

### 2.3 Flag 集合

| Flag | 默认 | 说明 |
| --- | --- | --- |
| `--metrics-bind-address` | `:8080` | Prometheus metrics |
| `--health-probe-bind-address` | `:8081` | `/healthz`, `/readyz` |
| `--leader-elect` | `true` | leader election |
| `--leader-election-id` | `axisml-tenant-operator.axisml.io` | Lease 名 |
| `--resync-period` | `10m` | 周期 resync 节流；用于源 Secret / ConfigMap 漂移收敛 |
| `--namespace-denylist` | 见 Helm values | 禁止落到的 Namespace 列表（默认覆盖 `kube-*`、`default`、`axisml-system`、`axisml-infra`） |

本 binary 只承载一个 reconciler，不需要多 controller 开关。

### 2.4 RBAC

tenant-operator 只声明**一个** ClusterRole（`<release>-tenant-operator`）；leader election Lease 在部署 namespace 通过 Role + RoleBinding 授权（不放进 cluster-scoped 角色）。

| 资源 | 动作 | 用途 |
| --- | --- | --- |
| `tenants.axisml.io` | `get / list / watch / patch` | watch Tenant CR、写 status |
| `namespaces` | `create / get / list / watch / update / patch` | 创建并对齐 Namespace metadata；**不含 `delete`**（§4.6.1） |
| `elasticquotas.scheduling.sigs.k8s.io` | `create / get / list / watch / update / patch / delete` | 派生并维护 per-tenant ElasticQuota |
| `secrets` | `create / get / list / watch / update / patch / delete`（目标 Namespace）；`get`（源 Namespace） | 维护 per-tenant Secret |
| `configmaps` | 同 secrets | 维护 per-tenant ConfigMap |
| `serviceaccounts` | `create / get / list / watch / update / patch / delete` | 维护 per-tenant ServiceAccount |
| `roles.rbac.authorization.k8s.io` | `create / get / list / watch / update / patch / delete` | 维护 per-tenant Role |
| `rolebindings.rbac.authorization.k8s.io` | `create / get / list / watch / update / patch / delete` | 维护 per-tenant RoleBinding |
| `events` | `create / patch` | 写 K8s Event |
| `coordination.k8s.io/leases`（自身 ns） | `create / get / list / watch / update / patch / delete` | leader election Lease |

`namespaces` 不含 `delete` 是显式的最小权限策略，对应 §4.6.1 的"永不删除 Namespace"行为约束。

### 2.5 Helm values 接口

```yaml
tenantOperator:
  enabled: true
  image: { registry, repository, tag, pullPolicy }
  leaderElection: { enabled: true, id: axisml-tenant-operator.axisml.io }
  resources: { requests, limits }
  resyncPeriod: 10m
  namespaceDenylist:
    - kube-*
    - default
    - axisml-system
    - axisml-infra
```

**Helm 模板清单**详见 [deployment.md §6.2](../deployment.md#62-tenant-operator--compute-operator)。

---

## Part II — 与 Cluster Manager 的协作契约

> 本部分集中 Tenant CR 的写路径与 Reconcile 行为约束。tenant-operator 的唯一上游写者是 [cluster-manager](cluster-manager.md)。

## 3. 与 Cluster Manager 的协作契约

### 3.1 写路径

cluster-manager 是无状态薄壳，把外部 REST 请求直接翻译为对 Tenant CR 的 K8s API 调用（详见 [cluster-manager.md §3](cluster-manager.md)）。tenant-operator 只通过 K8s API 与 cluster-manager 协作：

- **Create 幂等**：相同 `metadata.name` 的二次 Create 返回 409 `AlreadyExists`，不引发副作用（不重建底层资源、不重置 status）。cluster-manager 收到 409 后会 GET 现有 CR 并返回给调用方。
- **status 单向权威**：tenant-operator 只写 `status`；cluster-manager 只写 `metadata` / `spec`。状态推进由 cluster-manager 在响应 GET 时按 CR `status` 透传给调用方。tenant-operator **不向任何外部 PG 写入数据**。
- **配置补偿**：CR 被误删时由 cluster-manager 上游调用方按需重建（无自动补偿；管理员域操作天然低频，依赖人工或 platform 重新触发）；tenant-operator 的 `Reconcile` 必须可在已存在的底层资源上幂等收敛——已存在的资源不重建，只对齐 spec 漂移。

### 3.2 CRD metadata 约定

- `metadata.name` ← cluster-manager 透传调用方提交的 tenant 名；DNS-1123 + ≤40 字符（cluster-manager 在 API 层硬校验）。
- `metadata.labels["axisml.io/tenant-id"]` ← cluster-manager 在创建时填入 UUID（用于 platform 侧的稳定锚点；删除并重建同名 Tenant 时 UUID 会变）。
- 其余 label / annotation 可由 cluster-manager 透传 platform 提供的内容。

**status subresource 必启用**：`tenant-crd.yaml` 必须声明 `subresources.status`，由 Kubernetes API Server 隔离 controller 写 `status` 与 cluster-manager 写 `metadata` / `spec` 的边界。

**当前 CRD schema 现状**：`spec` / `status` 暂用 `x-kubernetes-preserve-unknown-fields: true`，重新设计字段不需 CRD bump；待行为稳定后再启用 OpenAPI schema 严格校验（§4.9）。

### 3.3 Reconcile 通用约束

- **不引入 finalizer**：Tenant 不挂 finalizer；级联清理依赖 `ownerReference`（cluster-scoped Tenant → namespaced 子资源）。
- **`Validate(spec)` 必须是纯函数**：不发起 K8s 调用，便于未来在 admission webhook 中复用。校验失败 → `status.phase=Failed`、`status.message` 写明违规项。
- **`Reconcile` 幂等**：多次调用相同 spec 不重建底层资源；只有语义字段变化才触发底层资源更新。
- **Status 写盘单一路径**：reconcile 末尾通过一次 patch 完成；`conditions[]` 按 `type` 去重后整体写回。

---

## Part III — Controller 详细设计

## 4. Tenant Controller

### 4.1 概述

Tenant controller 把 cluster-manager 下发的 `Tenant` CR 翻译为 Kubernetes 侧的命名空间、租户配额与租户级初始化资源，并把执行状态回流到 `Tenant.status`。它承载三类职责：

1. **Namespace 落地**：按 `spec.namespace.name` 创建并维护租户使用的 Namespace；同一 Namespace 允许被多个 Tenant CR 共享（详见 §4.7）。
2. **ElasticQuota 派生**：把 `spec.quotas[]` 渲染为 Koordinator `ElasticQuota` CR（每 `(tenant, pool, quota)` 一条，落在租户 Namespace 下），并把 `status.used` 回流到 `Tenant.status.quotas[].used`——这是 Tenant CR 与 cluster-manager / koord-scheduler 之间双向数据链路的承载。
3. **初始化资源下发**：按 `spec.initResources` 创建租户私有的 ImagePullSecrets / 通用 Secret / ConfigMap / ServiceAccount + RBAC。

### 4.2 与 Cluster Manager 的额外契约

除了 §3 列出的通用契约，Tenant 还有两条特有约束：

- **Namespace 永不级联删除**：Tenant 删除时仅依赖 ownerReference 让 K8s GC 清理 per-tenant 资源（ElasticQuota / Secret / ConfigMap / SA / Role / RoleBinding）；Namespace 自身不被删除（详见 §4.6.1）。
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

**为何把 quotas 内联到 Tenant.spec 而非独立 CRD**：Quota 在概念上是 Tenant 的子资源（每条配额都依附于一个 `(tenant, pool)` 组合），生命周期与 Tenant 强绑定。把 quotas 内联到 Tenant CR 让 tenant-operator 成为 ElasticQuota 的 single writer，给 cluster-manager 提供统一的双向数据链路：`spec.quotas[]` 下行表达 desired `min` / `max`，`status.quotas[].used` 上行回流实际用量。cluster-manager 通过 JSON Patch 修改 `Tenant.spec.quotas[]` 即可完成 quota 增删，无需引入独立 Quota CRD。

**为何 quotas 用数组而非 map**：每条 quota 的标识由 `(pool, name)` 元组确定，map 在 spec 里只能用字符串单 key，会丢失结构。

**为何不在 Tenant 上保留 K8s `ResourceQuota` 兜底字段**：K8s `ResourceQuota` 按 Namespace 聚合计量，不会按 Tenant CR 自动拆分用量。共享 Namespace 下不能表达 per-tenant 额度；独占 Namespace 下又与 ElasticQuota `max` 形成两套上限。租户级容量边界统一收敛到 ElasticQuota（`min` / `max` + Pod label `quota.scheduling.koordinator.sh/name`）。

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
  quotas:
    - pool: string               # 必填: ResourcePool 名
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
  suspended: false               # cluster-manager 标记并写 status.phase=Suspended
```

#### 4.3.3 字段归属与不可变性

| 字段路径 | 写入方 | 创建后可变？ |
| --- | --- | --- |
| `metadata.name` / `labels[axisml.io/tenant-id]` | cluster-manager | 否 |
| `spec.displayName` / `annotations` | cluster-manager（透传调用方输入） | 是 |
| `spec.namespace.name` | cluster-manager | **否**；controller 行为兜底（拒绝并写 `status.message`），admission webhook 为最终兜底 |
| `spec.namespace.labels` / `annotations` | cluster-manager | 是；只在 Namespace **首次创建** 时落地，已存在的 Namespace 不被覆盖（避免污染共享 Namespace） |
| `spec.quotas[].{pool, name}` | cluster-manager | **否**（每项标识锚点）；删除某项 → reconcile `Delete()` 对应 ElasticQuota CR；新增某项 → reconcile 创建 |
| `spec.quotas[].min` / `max` | cluster-manager | 是；reconcile 同步覆盖到对应 ElasticQuota `spec.min` / `spec.max` |
| `spec.initResources.*` | cluster-manager | 是；增删 → reconcile 创建 / 删除对应资源 |
| `spec.suspended` | cluster-manager（`/suspend` / `/unsuspend` API 触发） | 是 |

**默认值注入**：`spec.suspended` 默认 `false`；`spec.initResources` 各列表默认 `[]`；`spec.quotas` 默认 `[]`（视为租户在 K8s 调度层不限额）；`spec.quotas[].min` 默认 `{}`；`spec.initResources.secrets[].type` 默认 `Opaque`。

### 4.4 Status

```yaml
status:
  observedGeneration: int64
  phase: Active | Suspended | Failed
  message: string
  namespaceReady: bool
  quotas:
    - pool: string
      name: string
      ready: bool
      used: {}                   # 资源 map，来自 ElasticQuota.status.used
      message: string
  initResources:
    imagePullSecrets: [{ name, ready, message }]
    secrets:          [{ name, ready, message }]
    configMaps:       [{ name, ready, message }]
    serviceAccounts:  [{ name, ready, message }]
  conditions:
    - type: NamespaceReady | QuotasReady | InitResourcesReady | Suspended | Failed
      status: True | False | Unknown
      lastTransitionTime: timestamp
      reason: string
      message: string
```

**phase 推导规则**（reconcile 末尾计算）：

| 条件 | phase |
| --- | --- |
| `spec.suspended == true` | `Suspended` |
| `namespaceReady && 所有 quotas[*].ready == true && 所有 initResources[*].ready == true` | `Active` |
| 任一关键资源（Namespace / ElasticQuota）创建失败且非短暂瞬态 | `Failed` |
| 否则（瞬态创建过程中） | 维持上一态，`message` 写当前进展 |

`spec.quotas` 为空数组时 `status.quotas` 同为空，`Active` 推导只看 `namespaceReady` 与 initResources。

**`status.quotas[].used` 回流路径**：controller 通过 SharedInformerFactory watch 本集群所有 namespace 的 ElasticQuota CR，按 ownerReference 反查所属 Tenant，把 `ElasticQuota.status.used` 聚合到对应 `Tenant.status.quotas[i].used`。cluster-manager 在响应 `GET /api/v1/tenants/{name}` 时直接读取该字段返回给调用方。

### 4.5 Reconcile 生命周期

| 事件 | Controller 行为 |
| --- | --- |
| Tenant ADD（首次创建） | `Validate(spec)` 校验失败写 `status.phase=Failed`；通过则按 §4.6 顺序确保底层资源就位（Namespace → ElasticQuota → initResources），最后写 `status.phase=Active` |
| Tenant UPDATE（spec 变更） | 校验 `spec.namespace.name` 不变（违反则写 `status.message` 拒绝并维持原 phase）；其余 spec 变化按 §4.6 各小节"spec 漂移处理"覆盖底层资源 |
| Tenant UPDATE（`spec.suspended` 切换） | true → 写 `status.phase=Suspended`；false → 重新走 phase 推导。controller 不停机底层资源，只标记 phase；阻断新业务提交由上层（platform / compute）在自身侧拦截 |
| Tenant DELETE | 不阻断；K8s GC 通过 ownerReference 级联删除 per-tenant 资源；**Namespace 不删除**（§4.6.1） |
| 底层资源事件（ElasticQuota / Secret / ConfigMap / SA 等被外部修改或删除） | 按 ownerReference 反查到 Tenant，重新触发 Reconcile；ElasticQuota 的 `status.used` 变更触发轻量 reconcile，仅刷新 `Tenant.status.quotas[i].used` |
| 周期 resync（默认 10 min） | 触发对所有 Tenant CR 的 reconcile，重读 `sourceSecretRef` / `sourceConfigMapRef` 源数据，按漂移策略覆盖 per-tenant 副本 |

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
| 已存在 | 仅补 `axisml.io/managed-by=tenant-operator` label（如缺失）；不覆盖任何其他既有 label / annotation。`Namespace` 是 cluster-scoped 资源，K8s RBAC 不能按前缀或业务范围限制 `create`；controller 配置中维护 denylist / allowlist（默认拒绝 `kube-*`、`default`、`axisml-system`、`axisml-infra`），admission webhook 作为后续兜底（§4.9） |
| ownerReference | **不设置**——Namespace 不属于任何单一 Tenant |
| spec 漂移 | 不主动对账 |
| 删除 | **永不删除**——即使最后一个引用本 Namespace 的 Tenant 被删除，Namespace 也保留。空 Namespace 由集群管理员手工清理 |

**为何不删 Namespace**：Namespace 中可能存在 Tenant 不可见的 PV、外部 controller 创建的资源、用户手工创建的 ConfigMap。误删 Namespace 会引发不可逆的状态丢失。

`status.namespaceReady` 在 Namespace `phase=Active` 时为 `true`。

#### 4.6.2 ElasticQuota

`spec.quotas[]` 每项 1:1 渲染为一条 Koordinator `ElasticQuota` CR（`scheduling.sigs.k8s.io/v1alpha1`，namespace-scoped）。CR 落在 `spec.namespace.name` 下；Pod 通过 label `quota.scheduling.koordinator.sh/name=<eq-name>` 按集群唯一名跨 namespace 绑定 quota，所以共享 Namespace 与独占 Namespace 在 quota 隔离上没有差别——命名前缀已天然 per-tenant 隔离。

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
| 类型不一致 | `spec.type` 与源 Secret 的 `type` 不一致时以 spec 为准并写警告；若结构性约束失败 → 该项 `ready=false`、message 指明缺失字段。Secret type 在 K8s 中不可变，运行时 `spec.type` 改动 → reconcile 删除现有 Secret 重建 |
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

### 4.8 共享场景示意

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

### 4.9 后续工作

- Admission webhook：`spec.namespace.name` / `spec.quotas[].{pool, name}` 不可变约束、`spec.initResources.*.sourceXxxRef` 跨 Namespace 读权限白名单、`spec.quotas[].{min, max}` 结构性校验。
- **目标 Namespace 白名单**：当前通过 controller Helm values 配置 denylist / allowlist；后续由 admission webhook 前移到准入阶段。
- **源资源结构性校验前移**：admission webhook 在 Tenant 创建 / 更新时校验源 Secret 的 type 与 spec 一致。
- 加密源支持：从 KMS / Vault / Sealed Secrets 拉取凭证作为 `sourceSecretRef` 替代方案。
- `spec.initResources` templating：按 tenant 上下文（id / name / namespace）渲染 ConfigMap 数据。
- 跨 Namespace 复制源的 RBAC 收敛：把源 Namespace 限定为单一受控 Namespace。
- ServiceAccount + RBAC 子能力的 Helm values 开关。
- 分层配额：在 `spec.quotas[]` 引入 `parent` 字段，落到 ElasticQuota 的 `quota.scheduling.koordinator.sh/parent` annotation。
- CRD 严格 schema（启用 OpenAPI 校验，替换当前的 `x-kubernetes-preserve-unknown-fields`）。

---

## Part IV — 实施与验证

## 5. 实现路径

### 5.1 阶段一：MVP

支撑端到端最小演示路径："cluster-manager 创建 Tenant CR → 看到 phase=Active、namespace 与 ElasticQuota 落地"。

| 模块 | 范围 | 完成信号 |
| --- | --- | --- |
| Operator binary | 单 Manager / 单 reconciler、leader election、Cache `ByObject` 选择性过滤 | 单 Pod 启动后 Tenant controller Ready |
| Tenant | Namespace 创建（永不删除）、ElasticQuota 1:1 派生 + `status.used` 回流、ImagePullSecrets / Secrets / ConfigMaps / SAs + RBAC initResources、suspend、phase 推导 | integration 覆盖：happy path、suspend / unsuspend、quota update、源 Secret 缺失 |
| CRD | `tenant-crd.yaml` 用 `x-kubernetes-preserve-unknown-fields: true` 宽松 schema + `subresources.status` 显式声明 | helm install / upgrade 通过；status 写入不影响 spec 写入 |
| 测试 | integration 覆盖 happy path + suspend + immutability | `make tenant-operator-integration` 通过 |

### 5.2 阶段二：功能完善

1. **Admission webhook 上线**：`spec.namespace.name` / `spec.quotas[].{pool, name}` 不可变、跨 namespace `sourceXxxRef` 白名单。
2. **严格 CRD OpenAPI schema**：替换 `x-kubernetes-preserve-unknown-fields`；phase enum 收紧。
3. **resync 间隔 Helm values 暴露**：默认 10 min；可调到分钟级。
4. **目标 Namespace allowlist / denylist 默认硬化**：把 §4.6.1 默认拒绝列表落到 Helm `values.yaml` + 启动期校验。

### 5.3 阶段三：未来规划

参见 §4.9 后续工作清单。

## 6. 测试

integration 在 `components/tenant-operator/test/integration/` 单一 Go module 中，单一 `TestMain` 把 Tenant reconciler 注册到 envtest manager，覆盖 happy path、suspend / unsuspend、quota update、源 Secret 缺失等场景。CRDPaths 是 `deploy/helm/axisml-system/crds/tenant-crd.yaml` 与 `test/crds/external/elasticquota.yaml` 的并集。

仓库当前不维护 minikube 驱动的 e2e 层；端到端验证靠 integration（envtest）覆盖。

## 7. 相关引用

- [docs/system_design/overview.md](../overview.md) 概述了 tenant-operator 在控制平面里的位置。
- [docs/system_design/cluster-manager.md](cluster-manager.md) 描述 cluster-manager 与 tenant-operator 之间的 CR 写路径与 status 读路径。
- [docs/system_design/compute-operator.md](compute-operator.md) 是 tenant-operator 的兄弟 operator，承载 MLJob / MLService 调度。
- [docs/system_design/infra.md](../infra.md) 给出 koord-scheduler / ElasticQuota 等基础设施依赖契约。
