# Tenant Operator 详细设计

## 1. 概述

tenant-operator 把 AxisML Compute 下发的 `Tenant` CR 翻译为 Kubernetes 侧的命名空间、租户配额与租户级初始化资源，并把执行状态回流到 `Tenant.status`。它承载三类职责：

1. **Namespace 落地**：按 `spec.namespace.name` 创建并维护租户使用的 Namespace；同一 Namespace 允许被多个 Tenant CR 共享
2. **ElasticQuota 派生**：把 `spec.quotas[]` 渲染为 Koordinator `ElasticQuota` CR（每 `(tenant, pool, quota)` 一条，落在租户 Namespace 下），并把 `status.used` 回流到 `Tenant.status.quotas[].used`——这是 Tenant CR 与 Compute / koord-scheduler 之间双向数据链路的承载
3. **初始化资源下发**：按 `spec.initResources` 创建租户私有的 ImagePullSecrets / 通用 Secret / ConfigMap / ServiceAccount + RBAC

operator 与 Compute 的分工以 [compute.md §5 / §6.2.1](../compute.md) 为准；本文档只展开 operator 这一侧的实现契约。

Tenant 是 [compute.md §5.4](../compute.md) 中的 **配置对象**——CR 缺失/漂移会被 Compute 按 PG 快照补偿重建，因此 operator 的 `Reconcile` 必须可重复执行；operator 不主动反向写 Compute PG。

与 mljob-operator / mlservice-operator 的关键差异：tenant-operator **不存在多后端实现**，无 dispatcher/handler 分层；所有 Tenant CR 由单一 controller 处理。

## 2. 与 Compute 的写路径契约

Compute 采用 Operation Outbox + reconciler 异步下发 CR（详见 [compute.md §5.2](../compute.md)）。Operator 暴露给 Compute 的核心契约只有四条；其余约束（不引入 finalizer、`spec.namespace.name` 不可变、namespace 永不删除等）分散在 §3.3 字段不可变性、§5 Reconcile 生命周期、§9 不变量与约束。

- **`Create` 幂等**：相同 `metadata.name` 的二次 `Create()` 返回 409 `AlreadyExists`，且不引发副作用（不重复创建 Namespace、不重置 status）。Compute 收到 409 后会 `Get()` 现有 CR 并校验 label `axisml.io/tenant-id=<uuid>`；只有 label 一致才视为成功
- **status 单向权威**：operator 只写 `Tenant.status`，Compute 只写 `Tenant.metadata` / `Tenant.spec`；状态推进与配额用量回流均由 Compute 侧 Informer 按 CR `status` 消费，operator 不感知 Compute 的 `tenants` / `quotas` 表，也不向 Compute PG 写入任何数据
- **配置补偿友好**：Tenant CR 被误删后 Compute 会按 PG 快照重建（[compute.md §5.4](../compute.md) 配置对象路径），operator 的 `Reconcile` 必须可在已存在的底层资源上幂等收敛——已存在的 Namespace / ElasticQuota / Secret 等不重建，只对齐 spec 漂移
- **Namespace 永不级联删除**：Tenant 删除时 operator 仅依赖 ownerReference 让 K8s GC 清理 per-tenant 资源；Namespace 自身不被删除（详见 §6.1 与 §9）

## 3. CRD 契约

Tenant 为 cluster-scoped CR（CRD 定义见 `deploy/helm/axisml-system/crds/tenant-crd.yaml`）：

| 字段 | 取值 |
| --- | --- |
| `apiVersion` | `axisml.io/v1alpha1` |
| `kind` | `Tenant` |
| `scope` | `Cluster` |
| `shortNames` | `tnt` |

Compute 负责设置以下 metadata（与 [compute.md §6.2.1](../compute.md) 对齐）：

- `metadata.name` ← `tenants.name`
- `metadata.labels["axisml.io/tenant-id"]` ← `tenants.id`（UUID，孤儿检测稳定锚点）

### 3.1 spec 设计取舍

把 Tenant 的三类职责建模为同级字段——`namespace`（命名空间引用）、`quotas`（租户配额数组，1:1 渲染为 ElasticQuota CR）、`initResources`（初始化清单），避免任何一类职责喧宾夺主。

**为何把 quotas 内联到 Tenant.spec 而非独立 CR**：Quota 在概念上是 Tenant 的子资源（每条配额都依附于一个 `(tenant, pool)` 组合），生命周期与 Tenant 强绑定。把 quotas 内联到 Tenant CR 让 tenant-operator 成为 ElasticQuota 的 single writer，给 Compute 提供统一的双向数据链路：`spec.quotas[]` 下行表达 desired `min` / `max`，`status.quotas[].used` 上行回流实际用量。Compute 侧仍保留独立的 `quotas` PG 表（[compute.md §6.2.4](../compute.md)）以承载 API 行级 CRUD 与跨租户查询；CR 端只是该表的渲染。

**为何 quotas 用数组而非 map**：每条 quota 的标识由 `(pool, name)` 元组确定，map 在 spec 里只能用字符串单 key，会丢失结构。数组配合 §3.3 的不可变约束（`{pool, name}` 一旦写入即作为该项稳定锚点）即可表达。

**为何不在 Tenant 上保留 K8s `ResourceQuota` 兜底字段**：K8s `ResourceQuota` 按 Namespace 聚合计量，不会按 Tenant CR、ServiceAccount 或 `axisml.io/tenant-id` label 自动拆分用量。共享 Namespace 下不能表达 per-tenant 额度；独占 Namespace 下又与 ElasticQuota `max` 形成两套上限，徒增复杂度。租户级容量边界统一收敛到 ElasticQuota（`min` / `max` + Pod label `quota.scheduling.koordinator.sh/name`），tenant-operator 不再创建 `ResourceQuota`。

**为何把 namespace 名放在 `spec.namespace.name` 而非 `metadata.namespace`**：Tenant 是 cluster-scoped CR，不属于任何 Namespace；同时多个 Tenant 可共享同一个 Namespace，把 namespace 作为引用而非容器是更自然的建模——它是 Tenant 的"目标命名空间"而非"宿主命名空间"。这也让"切换 Namespace"作为不可变约束（§3.3）更自然——切 namespace 即切 spec 一个字段，不是 CR 重建。

**为何 per-tenant 资源命名统一加 `axisml-tenant-<tenant-name>-` 前缀**：共享 Namespace 场景下，多个 Tenant 在同一 Namespace 内创建同名 ImagePullSecret / ServiceAccount 会 collide。命名前缀一致化避免 collide，也让 selector 检索（"找出该 Namespace 下属于 tenant X 的所有资源"）有稳定锚点。

**长度上限**：`metadata.name` 已被 Compute API 限制为 ≤40 字符（[compute.md §6](../compute.md)）；`axisml-tenant-` 前缀 14 字符 + tenant-name 40 + 分隔符 1 = 55 字符。`spec.initResources.*[].name` 与 `serviceAccounts[].name` 的理论上限因此为 `253 - 55 = 198` 字符（DNS-1123 subdomain 总长 253）。实际命名场景远低于此，operator 不引入额外校验。

**为何初始化资源都从 `sourceXxxRef` 复制而非内联数据**：避免敏感数据（dockerconfigjson、对象存储凭证）以明文形式写入 Tenant CR。源 Secret / ConfigMap 由集群管理员预先放在受控 Namespace（如 `axisml-system`），operator 用 reader 权限读出再写入租户 Namespace。详见 §6.3–§6.5。

**为何 `runPolicy` 字段缺席**：Tenant 不是 workload——没有 `activeDeadlineSeconds` / `progressDeadlineSeconds` / `backoffLimit` 等概念。生命周期控制只有 `suspended` 一个开关（compute.md §6.2.1 状态机里 `Active ⇄ Suspended` 的载体）。

### 3.2 spec 结构

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

### 3.3 字段归属与不可变性

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

## 4. Status 契约

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

**phase 枚举冻结为三态**——`Active | Suspended | Failed`。新增 phase 必须 CRD schema 与 Compute 双侧同步演进。Compute 的状态映射规则（与 [compute.md §6.2.1](../compute.md) 对齐）：

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

**`status.quotas[].used` 回流路径**：tenant-operator 通过 SharedInformerFactory watch 本集群所有 namespace 的 ElasticQuota CR，按 ownerReference 反查所属 Tenant，把 `ElasticQuota.status.used` 聚合到对应 `Tenant.status.quotas[i].used`。Compute Tenant Informer 消费该字段更新 PG `quotas.used` 缓存（详见 [compute.md §5.3 / §6.2.4](../compute.md)）。

`conditions` 与 `initResources[]` 是给 UI 与运维用的 observability 字段，Compute 不消费。

## 5. Reconcile 生命周期

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

## 6. 底层资源管理

每条 per-tenant 资源都叠加以下 label，便于在共享 Namespace 内 selector 过滤：

- `axisml.io/tenant-id=<uuid>`
- `axisml.io/managed-by=tenant-operator`

每条 per-tenant 资源都通过 `ownerReferences` 指向 Tenant CR（cluster-scoped owner → namespaced dependent，K8s GC 原生支持）；Tenant 删除后由 K8s GC 异步清理。

### 6.1 Namespace

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

### 6.2 ElasticQuota

`spec.quotas[]` 每项 1:1 渲染为一条 Koordinator `ElasticQuota` CR（`scheduling.sigs.k8s.io/v1alpha1`，namespace-scoped）。CR 落在 `spec.namespace.name` 下；Pod 通过 label `quota.scheduling.koordinator.sh/name=<eq-name>` 按集群唯一名跨 namespace 绑定 quota（详见 [compute.md §6.2.4 多 namespace 契约](../compute.md)），所以共享 Namespace 与独占 Namespace 在 quota 隔离上没有差别——命名前缀 `axisml-<tenant>-<pool>-<name>` 已天然 per-tenant 隔离。

| 维度 | 行为 |
| --- | --- |
| 命名 | `axisml-<tenant-name>-<pool>-<quota-name>`（与 [compute.md §6.2.4](../compute.md) 命名约定对齐；集群内唯一）|
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

### 6.3 ImagePullSecrets

| 维度 | 行为 |
| --- | --- |
| 命名 | `axisml-tenant-<tenant-name>-<spec-name>`（`spec-name` 取 `spec.initResources.imagePullSecrets[].name`） |
| 类型 | `kubernetes.io/dockerconfigjson` |
| 数据来源 | `sourceSecretRef.{namespace, name}` 指向受控 Namespace（如 `axisml-system`）中预先创建好的 Secret；operator 用 reader 权限 `Get()` 后把 `data` 写入新 Secret |
| ownerReference | Tenant CR |
| spec 漂移 | reconcile 检测到本端 Secret 数据与源 Secret 不一致时覆盖；源 Secret 不存在时把对应 `status.initResources.imagePullSecrets[i].ready=false` 并写 message |
| 删除 | 随 Tenant 删除 GC；spec 中删除某项 → reconcile 显式 Delete 对应 Secret |

**为何用 `sourceSecretRef` 而非内联 dockerconfigjson**：避免明文凭证写入 Tenant CR（CR 通过 etcd 持久化、可能被备份导出）。源 Secret 集中放在受控 Namespace 由集群管理员维护，权限收敛。

### 6.4 通用 Secret

| 维度 | 行为 |
| --- | --- |
| 命名 | `axisml-tenant-<tenant-name>-<spec-name>` |
| 类型 | 取 `spec.initResources.secrets[].type`，默认 `Opaque`；允许 `dockerconfigjson` / `kubernetes.io/tls` 等任意 K8s Secret type |
| 类型不一致 | `spec.type` 与源 Secret 的 `type` 不一致时以 spec 为准，operator 在 `status.initResources.secrets[i].message` 写警告并按 `spec.type` 创建本端 Secret；若结构性约束失败（如 `dockerconfigjson` 要求特定 key、`tls` 要求 `tls.crt`/`tls.key`） → 该项 `ready=false`、message 指明缺失字段。Secret type 在 K8s 中不可变，运行时 `spec.type` 改动 → reconcile 删除现有 Secret 重建 |
| 数据来源 | 同 §6.3，`sourceSecretRef` 复制 |
| ownerReference / 漂移 / 删除 | 同 §6.3 |

典型用途：对象存储访问凭证、TLS 证书、OAuth client secret 等租户私有的非镜像凭证。

### 6.5 ConfigMap

| 维度 | 行为 |
| --- | --- |
| 命名 | `axisml-tenant-<tenant-name>-<spec-name>` |
| 数据来源 | `sourceConfigMapRef.{namespace, name}`，operator `Get()` 源 ConfigMap 后把 `data` / `binaryData` 写入新 ConfigMap |
| ownerReference | Tenant CR |
| spec 漂移 | reconcile 检测到本端数据与源不一致时覆盖；源不存在时 `ready=false` |
| 删除 | 同 §6.3 |

典型用途：租户级默认环境变量、CA 证书包、统一日志采集配置等。

### 6.6 ServiceAccount + RBAC

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

## 7. 多 Tenant 共享 Namespace 语义

`spec.namespace.name` 允许多个 Tenant CR 指向同一 Namespace；典型场景是多个轻量级团队共享一个开发 / 沙箱环境。共享时的关键不变量：

- **Namespace 自身不绑定 ownerReference**——Namespace 是共享资源，不属于任一 Tenant
- **per-tenant 资源命名前缀**：tenant-operator 派生的 per-tenant 资源（Secret / ConfigMap / SA / Role / RoleBinding）用 `axisml-tenant-<tenant-name>-` 前缀；ElasticQuota 用 `axisml-<tenant-name>-<pool>-<quota>` 前缀。两套前缀都集群唯一，共享 Namespace 内不会 collide
- **per-tenant 资源 label `axisml.io/tenant-id=<uuid>`** 提供 selector 检索能力（"该 Namespace 内属于 tenant X 的所有资源"）
- **per-tenant ElasticQuota**：每个 Tenant 在共享 Namespace 内仍各自持有独立 ElasticQuota CR，Pod 通过 label `quota.scheduling.koordinator.sh/name` 按集群唯一 quota name 绑定；koord-scheduler 按名字跨 namespace 维护用量，per-tenant 隔离照常生效
- **Pod 通过 ServiceAccount 关联 tenant**：Pod 选择 `axisml-tenant-<tenant>-<sa>` SA → 自动获得本 tenant 的 imagePullSecrets / RBAC；ServiceAccount 是 tenant 身份在 K8s API 调用面的载体
- **Tenant A 删除不影响 Tenant B**：Tenant A 删除 → K8s GC 清理 A 的 per-tenant 资源（ElasticQuota / Secret / ConfigMap / SA / Role / RoleBinding 等）→ B 的 per-tenant 资源不受影响 → Namespace 保留

### 共享场景示意

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

### 与 Compute schema 的关系

[compute.md §6.2.1](../compute.md) 已把 `tenants.namespace` 建模为非唯一索引，允许多个 Tenant 指向同一 Namespace。tenant-operator 端不再需要通过 Tenant CR 列表识别共享关系——ElasticQuota 的 per-tenant 隔离由命名前缀 + Pod label 自然达成，与 Namespace 是否被共享解耦。

## 8. RBAC

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

## 9. 不变量与约束

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

## 10. 后续设计文档（不在本文档范围）

- [compute.md §6.2.1](../compute.md) 已放宽 `tenants.namespace` 唯一约束（本文档随 namespace 共享语义同步落地）；尚需补齐"反查同 Namespace 下所有 Tenant"的查询路径以及配套的 UI 展示
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
