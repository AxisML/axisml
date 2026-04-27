# Tenant Operator 详细设计

## 1. 概述

tenant-operator 把 AxisML Compute 下发的 `Tenant` CR 翻译为 Kubernetes 侧的命名空间与租户级初始化资源，并把执行状态回流到 `Tenant.status`。它承载三类职责：

1. **Namespace 落地**：按 `spec.namespace.name` 创建并维护租户使用的 Namespace；同一 Namespace 允许被多个 Tenant CR 共享
2. **ResourceQuota 隔离**：在 Namespace 内为每个 Tenant 创建一个独立的 `ResourceQuota`，即便共享 Namespace 也保持租户级配额隔离
3. **初始化资源下发**：按 `spec.initResources` 创建租户私有的 ImagePullSecrets / 通用 Secret / ConfigMap / ServiceAccount + RBAC

operator 与 Compute 的分工以 [compute.md §5 / §6.2.1](../compute.md) 为准；本文档只展开 operator 这一侧的实现契约。

Tenant 是 [compute.md §5.4](../compute.md) 中的 **配置对象**——CR 缺失/漂移会被 Compute 按 PG 快照补偿重建，因此 operator 的 `Reconcile` 必须可重复执行；operator 不主动反向写 Compute PG。

与 mljob-operator / mlservice-operator 的关键差异：tenant-operator **不存在多后端实现**，无 dispatcher/handler 分层；所有 Tenant CR 由单一 controller 处理。

## 2. 与 Compute 的写路径契约

Compute 采用 Operation Outbox + reconciler 异步下发 CR（详见 [compute.md §5.2](../compute.md)）。Operator 暴露给 Compute 的核心契约只有四条；其余约束（不引入 finalizer、`spec.namespace.name` 不可变、namespace 永不删除等）分散在 §3.3 字段不可变性、§5 Reconcile 生命周期、§9 不变量与约束。

- **`Create` 幂等**：相同 `metadata.name` 的二次 `Create()` 返回 409 `AlreadyExists`，且不引发副作用（不重复创建 Namespace、不重置 status）。Compute 收到 409 后会 `Get()` 现有 CR 并校验 label `axisml.io/tenant-id=<uuid>`；只有 label 一致才视为成功
- **status 单向权威**：operator 只写 `Tenant.status`，Compute 只写 `Tenant.metadata` / `Tenant.spec`；状态推进由 Compute 侧 Informer 按 CR `status` 回流，operator 不感知 Compute 的 `tenants` 表，也不向 Compute PG 写入任何数据
- **配置补偿友好**：Tenant CR 被误删后 Compute 会按 PG 快照重建（[compute.md §5.4](../compute.md) 配置对象路径），operator 的 `Reconcile` 必须可在已存在的底层资源上幂等收敛——已存在的 Namespace / ResourceQuota / Secret 等不重建，只对齐 spec 漂移
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

把 Tenant 的三类职责建模为同级字段——`namespace`（命名空间引用）、`resourceQuota`（租户级配额）、`initResources`（初始化清单），避免任何一类职责喧宾夺主。

**为何把 namespace 名放在 `spec.namespace.name` 而非 `metadata.namespace`**：Tenant 是 cluster-scoped CR，不属于任何 Namespace；同时多个 Tenant 可共享同一个 Namespace，把 namespace 作为引用而非容器是更自然的建模——它是 Tenant 的"目标命名空间"而非"宿主命名空间"。这也让"切换 Namespace"作为不可变约束（§3.3）更自然——切 namespace 即切 spec 一个字段，不是 CR 重建。

**为何 per-tenant 资源命名统一加 `axisml-tenant-<tenant-name>-` 前缀**：共享 Namespace 场景下，多个 Tenant 在同一 Namespace 内创建同名 ImagePullSecret / ServiceAccount 会 collide。命名前缀一致化避免 collide，也让 selector 检索（"找出该 Namespace 下属于 tenant X 的所有资源"）有稳定锚点。

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

  # ── 资源配额（per-tenant，落到 ResourceQuota CR）─────────────────
  resourceQuota:
    hard: {}                     # K8s ResourceQuota.spec.hard 直传
    scopes: []                   # 可选: ResourceQuota.spec.scopes
    scopeSelector: {}            # 可选: ResourceQuota.spec.scopeSelector

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
| `spec.namespace.name` | Compute | **否**；controller 拒绝并写 `status.message`，admission webhook 后续接管 |
| `spec.namespace.labels` / `annotations` | Compute | 是；只在 Namespace **首次创建** 时落地，已存在的 Namespace 不被覆盖（避免污染共享 Namespace） |
| `spec.resourceQuota.*` | Compute | 是；reconcile 同步覆盖到 ResourceQuota |
| `spec.initResources.*` | Compute | 是；增删 → reconcile 创建 / 删除对应资源（per-tenant 命名前缀保证不会误删其他租户资源） |
| `spec.suspended` | API（`:suspend` / `:unsuspend` 触发） | 是 |

**默认值注入**：`spec.suspended` 默认 `false`；`spec.initResources` 各列表默认 `[]`；`spec.namespace.labels` / `annotations` 默认 `{}`。

**CRD schema 现状**：当前 CRD 的 `spec` / `status` 用 `x-kubernetes-preserve-unknown-fields: true`，重新设计字段无需 CRD bump；待行为稳定后再启用 OpenAPI schema 严格校验（§10）。

## 4. Status 契约

```yaml
status:
  observedGeneration: int64      # controller reconcile 自洽用；Compute 不强消费
  phase: Active | Suspended | Failed   # ← Compute 唯一消费的字段
  message: string                # 错误或状态附加信息（Compute 透传到 tenants.message）
  namespaceReady: bool           # Namespace 已就绪（Compute 可观测）
  resourceQuotaReady: bool       # ResourceQuota 已就绪（Compute 可观测）
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
    - type: NamespaceReady | ResourceQuotaReady | InitResourcesReady | Suspended | Failed
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
| `namespaceReady && resourceQuotaReady && 所有 initResources[*].ready == true` | `Active` |
| 任一关键资源（Namespace / ResourceQuota）创建失败且非短暂瞬态 | `Failed` |
| 否则（瞬态创建过程中） | 维持上一态，`message` 写当前进展 |

`conditions` 与 `initResources[]` 是给 UI 与运维用的 observability 字段，Compute 不消费。

## 5. Reconcile 生命周期

按事件源切分 controller 职责：

| 事件 | Controller 行为 |
| --- | --- |
| Tenant ADD（首次创建） | `Validate(spec)` 校验失败写 `status.phase=Failed`；通过则按 §6 顺序确保底层资源就位（Namespace → ResourceQuota → initResources），最后写 `status.phase=Active` |
| Tenant UPDATE（spec 变更） | 校验 `spec.namespace.name` 不变（违反则写 `status.message` 拒绝并维持原 phase）；其余 spec 变化按 §6 各小节"spec 漂移处理"覆盖底层资源 |
| Tenant UPDATE（`spec.suspended` 切换） | true → 写 `status.phase=Suspended`；false → 重新走 phase 推导（§4）。controller 不停机底层资源，只标记 phase；阻断新 Job/Service 提交由 Compute API 在 `tenant.status='Suspended'` 时拦截（compute.md §6.2.1） |
| Tenant DELETE | 不阻断；不引入 finalizer；K8s GC 通过 ownerReference 级联删除 per-tenant 资源（ResourceQuota / Secret / ConfigMap / SA / Role / RoleBinding）；**Namespace 不删除**（§6.1） |
| 底层资源事件（ResourceQuota / Secret / ConfigMap / SA 等被外部修改或删除） | 按 ownerReference 反查到 Tenant，重新触发 Reconcile；漂移按各小节策略覆盖回 spec 快照 |

**关键约束**：

- Controller **不引入 finalizer**；ownerReference 级联清理是默认路径；Namespace 不参与级联（无 ownerReference）
- `Validate(spec)` 必须是纯函数（不发起 K8s 调用），便于未来在 admission webhook 中复用
- `Reconcile` 多次调用相同 spec 不重建底层资源；只有语义字段变化才触发更新
- Status 写入只在 reconcile 末尾通过一次 patch 完成，避免半成品 status

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
| 已存在 | 不修改 metadata（避免污染共享 Namespace 中由其他 Tenant 或管理员设置的 label/annotation）；只把 `axisml.io/managed-by=tenant-operator` label 加上（如缺失） |
| ownerReference | **不设置**——Namespace 不属于任何单一 Tenant |
| spec 漂移 | 不主动对账（Namespace 自身没有"由 Tenant 决定"的 spec 字段） |
| 删除 | **永不删除**——即使最后一个引用本 Namespace 的 Tenant 被删除，Namespace 也保留。空 Namespace 由集群管理员手工清理 |

**为何不删 Namespace**：Namespace 中可能存在 Tenant 不可见的 PV、外部 controller 创建的资源（如 Volcano PodGroup 历史记录、用户手工创建的 ConfigMap）。误删 Namespace 会引发不可逆的状态丢失。把"清理空 Namespace"作为运维操作而非 operator 的自动行为是更安全的取舍——代价仅是空 Namespace 残留。

`status.namespaceReady` 在 Namespace `phase=Active` 时为 `true`。

### 6.2 ResourceQuota

| 维度 | 行为 |
| --- | --- |
| 命名 | `axisml-tenant-<tenant-name>` |
| 创建 | 在 `<spec.namespace.name>` 内创建 ResourceQuota，`spec.hard` / `scopes` / `scopeSelector` 直接来自 `spec.resourceQuota.*` |
| ownerReference | Tenant CR |
| spec 漂移 | reconcile 检测到 `ResourceQuota.spec` 与 `spec.resourceQuota` 不一致时按 spec 覆盖（PG 快照为权威） |
| 删除 | 随 Tenant 删除 K8s GC |

**多 Tenant 共享 Namespace 时的语义**：K8s 在同一 Namespace 内对多个 ResourceQuota 取 AND——任一 Quota 超限即拒绝。因此各 Tenant 各持一份 ResourceQuota，会自然得到"每个租户分别受限"的效果，且 Namespace 总量不会超过最严格者之和。对应配额预检由 Compute 在 [compute.md §6.2.4](../compute.md) 队列层面承担；ResourceQuota 仅作为 K8s 准入兜底。

`status.resourceQuotaReady` 在 ResourceQuota 创建成功且 `status.used` 已被 K8s 填充时为 `true`。

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
- **per-tenant 资源命名前缀 `axisml-tenant-<tenant-name>-`** 保证不 collide
- **per-tenant 资源 label `axisml.io/tenant-id=<uuid>`** 提供 selector 检索能力（"该 Namespace 内属于 tenant X 的所有资源"）
- **多 ResourceQuota AND 语义**：K8s 在同一 Namespace 内对多 ResourceQuota 取并集约束（任一超限即拒绝），自然实现"各 Tenant 分别受限 + Namespace 总量受最严格者之和约束"
- **Pod 通过 ServiceAccount 关联 tenant**：Pod 选择 `axisml-tenant-<tenant>-<sa>` SA → 自动获得本 tenant 的 imagePullSecrets / RBAC；ServiceAccount 是 tenant 身份在 K8s API 调用面的载体
- **Tenant A 删除不影响 Tenant B**：Tenant A 删除 → K8s GC 清理 A 的 per-tenant 资源（ResourceQuota / Secret / SA / RoleBinding 等）→ B 的 per-tenant 资源不受影响 → Namespace 保留

### 共享场景示意

假设 Namespace `shared-dev` 同时托管 Tenant `team-a` 与 `team-b`：

```
Namespace shared-dev
├── ResourceQuota axisml-tenant-team-a       (owner: Tenant team-a)
├── ResourceQuota axisml-tenant-team-b       (owner: Tenant team-b)
├── Secret        axisml-tenant-team-a-registry  (owner: Tenant team-a)
├── Secret        axisml-tenant-team-b-registry  (owner: Tenant team-b)
├── ServiceAccount axisml-tenant-team-a-default   (owner: Tenant team-a)
├── ServiceAccount axisml-tenant-team-b-default   (owner: Tenant team-b)
├── ...
```

team-a 的 Pod 只能选择 `axisml-tenant-team-a-*` SA，从而只看到本 tenant 的 imagePullSecrets / RBAC；ResourceQuota 由 K8s 准入对所有 Pod 同时校验，team-a 用满 quota 不会越权占用 team-b 的份额。

### 与 Compute schema 的关系

[compute.md §6.2.1](../compute.md) 当前的 `tenants.namespace UNIQUE` 约束假定一个 Namespace 仅对应一个 Tenant；为支持本节定义的共享语义，该 UNIQUE 约束需要在 compute.md 后续版本中放宽（迁移为非唯一索引；§10）。本文档定义的 operator 行为已为放宽后的契约准备好——operator 端无任何"一个 Namespace 唯一对应一个 Tenant"的隐式假设。

## 8. RBAC

operator binary 启动时聚合以下权限到 ServiceAccount。Helm chart 通过 values 控制是否启用 ServiceAccount + RBAC 子能力（关闭时可裁剪 §6.6 相关权限）。

| 资源 | 动作 | 用途 |
| --- | --- | --- |
| `tenants.axisml.io` | `get / list / watch / patch` | watch Tenant CR、写 status |
| `namespaces` | `create / get / list / watch / update / patch` | 创建并对齐 Namespace metadata；**不含 `delete`** |
| `resourcequotas` | `create / get / list / watch / update / patch / delete` | 维护 per-tenant ResourceQuota |
| `secrets` | `create / get / list / watch / update / patch / delete` | 维护 per-tenant Secret（ImagePullSecrets / 通用 Secret）；同时需要在源 Namespace（如 `axisml-system`）`get / list / watch` 用于复制源数据 |
| `configmaps` | `create / get / list / watch / update / patch / delete` | 维护 per-tenant ConfigMap；同时需要在源 Namespace `get / list / watch` |
| `serviceaccounts` | `create / get / list / watch / update / patch / delete` | 维护 per-tenant ServiceAccount |
| `roles.rbac.authorization.k8s.io` | `create / get / list / watch / update / patch / delete` | 维护 per-tenant Role |
| `rolebindings.rbac.authorization.k8s.io` | `create / get / list / watch / update / patch / delete` | 维护 per-tenant RoleBinding |
| `events` | `create / patch` | 写 K8s Event 用于运维观测 |

**`namespaces` 不含 `delete`** 是显式的最小权限策略，对应 §6.1 的"永不删除 Namespace"行为约束——即便代码出错也无法越权删除 Namespace。

**跨 Namespace 复制源资源的最小权限**：operator 默认信任所有 Namespace 中的源资源；如需收敛权限，可在 RBAC 上把 `secrets` / `configmaps` 的 `get / list / watch` 限定到指定 Namespace（例如 `axisml-system`），通过 Role + RoleBinding 而非 ClusterRole 表达——该收敛由 Helm values 控制，作为后续设计文档（§10）。

## 9. 不变量与约束

- `spec.namespace.name` 创建后不可变；controller 拒绝并写 `status.message`，admission webhook 后续接管
- **operator 永不删除 Namespace**；即使最后一个引用本 Namespace 的 Tenant 被删除，Namespace 也保留
- Controller **不引入 finalizer**；级联清理依赖 ownerReference（cluster-scoped Tenant → namespaced 子资源）
- per-tenant 资源命名一律带 `axisml-tenant-<tenant-name>-` 前缀，避免共享 Namespace 内 collide
- per-tenant 资源必须打 `axisml.io/tenant-id` 与 `axisml.io/managed-by=tenant-operator` label；缺失即视为 operator 不应识别（用于人工排障误打 label 的场景）
- operator 不读写 Volcano Queue / MLJob / MLService CR；这些 CR 由 Compute 与对应 operator 独占维护
- operator 不向 Compute PG 写入任何数据；状态全部经由 Tenant `status` + Informer 回流
- `status.phase` 取值集合冻结为三态（`Active | Suspended | Failed`）；新增 phase 必须经 CRD schema 与 Compute 双侧同步演进
- 初始化资源数据来源限定为同集群内的 `sourceSecretRef` / `sourceConfigMapRef`；不接受内联敏感数据（避免 etcd 明文持久化）

## 10. 后续设计文档（不在本文档范围）

- [compute.md §6.2.1](../compute.md) `tenants.namespace UNIQUE` 约束放宽：调整为非唯一索引，并补齐"反查同 Namespace 下所有 Tenant"的查询路径
- Admission webhook：`spec.namespace.name` 不可变约束、`spec.initResources.*.sourceXxxRef` 跨 Namespace 读权限白名单、`spec.resourceQuota.hard` 校验
- 加密源支持：从 KMS / Vault / Sealed Secrets 拉取凭证作为 `sourceSecretRef` 替代方案
- `spec.initResources` templating：按 tenant 上下文（id / name / namespace）渲染 ConfigMap 数据（如把 `<tenant-name>` 注入到统一日志采集配置）
- 跨 Namespace 复制源的 RBAC 收敛：把源 Namespace 限定为单一受控 Namespace（如 `axisml-system`），通过 Role + RoleBinding 而非 ClusterRole 表达
- ServiceAccount + RBAC 子能力的 Helm values 开关与对应 RBAC 收敛
- CRD 严格 schema（启用 OpenAPI 校验，替换当前的 `x-kubernetes-preserve-unknown-fields`）
