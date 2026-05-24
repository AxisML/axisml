# AxisML Tenant Operator 概要设计

## 1. 定位与边界

把 [cluster-manager](cluster-manager.md) 下发的 `Tenant` CR 翻译为 K8s 侧的 Namespace、Koordinator `ElasticQuota`、租户私有的 Secret / ConfigMap / ServiceAccount + RBAC,并把执行状态回流到 `Tenant.status`。

| 做 | 不做 |
| --- | --- |
| Namespace 创建与 metadata 对齐 (永不删除) | Tenant CR / 配额的 CRUD API (→ [cluster-manager.md](cluster-manager.md)) |
| 每条 `spec.quotas[]` 渲染为 ElasticQuota CR,回流 `status.used` | MLJob / MLService 生命周期 (→ [compute-operator.md](compute-operator.md)) |
| `spec.initResources` 下发 ImagePullSecret / Secret / ConfigMap / SA + RBAC | 用户认证、平台 RBAC (→ [auth.md](../auth.md)) |
| 周期 resync 收敛源 Secret / ConfigMap 漂移 | 跨集群 / 多 region 联邦 |

## 2. 架构

### 2.1 上下文

```
   ┌──────────────────┐  patch Tenant CR    ┌──────────────────┐
   │ cluster-manager  │ ───────────────────▶│   K8s API        │
   └──────────────────┘                     │   Tenant CR      │
                                            └────────┬─────────┘
                                                     │ watch
                                                     ▼
                                          ┌──────────────────────┐
                                          │   tenant-operator    │
                                          └──────────┬───────────┘
                              create / patch / GC    │
            ┌────────────────────────────┬───────────┴──────────────┐
            ▼                            ▼                          ▼
     ┌──────────────┐           ┌────────────────┐         ┌──────────────────┐
     │  Namespace   │           │ ElasticQuota   │         │ Secret / CM /    │
     │              │           │ (Koordinator)  │         │ SA + Role/RB     │
     └──────────────┘           └────────┬───────┘         └──────────────────┘
                                         │ status.used 回流
                                         ▼
                                  Tenant.status.quotas[].used
```

### 2.2 内部结构

```
┌──────────────── tenant-operator (one Pod, leader-elected) ───────────────┐
│  ctrl.Manager                                                             │
│    scheme: clientgoscheme + axisml.tenant + Koordinator ElasticQuota     │
│    lease : axisml-tenant-operator.axisml.io                              │
│    cache : managed-by=tenant-operator selector (per-tenant 子资源过滤)   │
│                                                                           │
│  ┌─────────────────────── Tenant Reconciler (单一) ────────────────────┐ │
│  │  Validate(spec)  →  Namespace  →  ElasticQuota[]  →  initResources │ │
│  │                              ↓                                       │ │
│  │                     phase 推导 + 单次 status patch                   │ │
│  └─────────────────────────────────────────────────────────────────────┘ │
└───────────────────────────────────────────────────────────────────────────┘
```

无 dispatcher / handler 分层;所有 Tenant CR 走同一 reconciler。

## 3. 核心模型

| 实体 | 含义 | 标识键 | 备注 |
| --- | --- | --- | --- |
| Tenant | 租户 CR,cluster-scoped | `metadata.name` (DNS-1123, ≤40) | 上游唯一写者为 cluster-manager |
| Namespace | 运行租户 Pod 的 K8s namespace | `spec.namespace.name` | 可被多 Tenant 共享 (§5.1) |
| ElasticQuota | Koordinator 配额 CR | `axisml-<tenant>-<pool>-<quota>` | 每条 `spec.quotas[]` 1:1 渲染 |
| InitResource | per-tenant Secret / CM / SA + RBAC | `axisml-tenant-<tenant>-<name>` | 由 `sourceXxxRef` 复制 |

Tenant CR 字段定义见 [deploy/helm/axisml-system/crds/tenant-crd.yaml](../../../deploy/helm/axisml-system/crds/tenant-crd.yaml);ElasticQuota 调度行为见 [infra.md](../infra.md)。

## 4. 核心功能

### 4.1 Tenant Controller

承载三件事:Namespace 落地、ElasticQuota 派生、初始化资源下发。统一遵循以下不变量:

- **不引入 finalizer**:级联清理依赖 ownerReference (cluster-scoped Tenant → namespaced 子资源);
- **`Validate(spec)` 纯函数**:不发起 K8s 调用,失败 → `phase=Failed` + `message`;
- **status 单次 patch**:reconcile 末尾整体写回 `conditions[]` 与 phase;
- **per-tenant 子资源叠加 label** `axisml.io/tenant-id=<uuid>` + `axisml.io/managed-by=tenant-operator`,便于共享 Namespace 内 selector 检索。

reconcile 触发事件:

| 事件 | 行为 |
| --- | --- |
| Tenant ADD / UPDATE(spec) | Validate → 按 §4.1.1 → §4.1.2 → §4.1.3 顺序就位,末尾推 phase |
| Tenant UPDATE(`spec.suspended`) | true → `phase=Suspended`;false → 重新走 phase 推导;不停机底层资源 |
| Tenant DELETE | 不阻断;K8s GC 按 ownerReference 清理子资源;Namespace 不删除 |
| 子资源事件 | 按 ownerReference 反查触发 reconcile;ElasticQuota `status.used` 只刷新 `Tenant.status.quotas[i].used` |
| 周期 resync (默认 10 min) | 重读 `sourceSecretRef` / `sourceConfigMapRef` 收敛漂移 |

#### 4.1.1 Namespace 落地

| 维度 | 行为 |
| --- | --- |
| 命名 | `spec.namespace.name` (创建后不可变) |
| 创建 | 不存在 → 创建,附加 `spec.namespace.labels` / `annotations` + `axisml.io/managed-by` label;受 denylist 约束 (默认拒绝 `kube-*` / `default` / `axisml-system` / `axisml-infra`) |
| 已存在 | 仅补 `axisml.io/managed-by` label;**不覆盖**既有 label / annotation |
| ownerReference | **不设置** — Namespace 不属于任何单一 Tenant |
| 漂移 | 不主动对账 |
| 删除 | **永不删除** — 避免误删 PV / 外部资源 / 手工对象;空 Namespace 由管理员清理 |

**关键不变量**:`spec.namespace.name` 创建后不可变;controller 行为兜底拒绝,admission webhook 为最终兜底。`status.namespaceReady` 在 Namespace `phase=Active` 时为 `true`。

#### 4.1.2 ElasticQuota 落地

每条 `spec.quotas[]` 1:1 渲染为 Koordinator `ElasticQuota` CR,落在 `spec.namespace.name` 下。

| 维度 | 行为 |
| --- | --- |
| 命名 | `axisml-<tenant-name>-<pool>-<quota-name>` (集群内唯一) |
| 创建 / 删除 | `spec.min` / `max` 直传;空数组不创建;spec 增删项 → Create / Delete 对应 CR |
| ownerReference | Tenant CR |
| 漂移 | reconcile 按 `spec.quotas[i].{min, max}` 覆盖 ElasticQuota |
| status.used 回流 | watch ElasticQuota,把 `status.used` 写入 `Tenant.status.quotas[i].used`;不写回 ElasticQuota |

**关键不变量**:`(pool, name)` 在 `spec.quotas[]` 内唯一且创建后不可变;`min[k] ≤ max[k]` 且均 ≥ 0。Pod 通过 label `quota.scheduling.koordinator.sh/name=<eq-name>` 跨 namespace 绑定 quota。

#### 4.1.3 初始化资源落地

`spec.initResources` 下的 ImagePullSecret / 通用 Secret / ConfigMap / ServiceAccount + RBAC 走同一套"从受控 Namespace 复制"模式。

| 资源 | 命名 | 数据来源 / 行为 |
| --- | --- | --- |
| ImagePullSecret | `axisml-tenant-<tenant>-<name>` | `sourceSecretRef`;type 固定 `dockerconfigjson` |
| 通用 Secret | `axisml-tenant-<tenant>-<name>` | `sourceSecretRef`;type 默认 `Opaque`,允许 `tls` 等;K8s Secret type 不可变 → 改 type 触发删除重建 |
| ConfigMap | `axisml-tenant-<tenant>-<name>` | `sourceConfigMapRef` |
| ServiceAccount | `axisml-tenant-<tenant>-<name>` | 解析 `imagePullSecrets[]` (用户可见 name → 最终 Secret 名) 后写入 SA |
| Role / RoleBinding | `axisml-tenant-<tenant>-<sa>` | 声明 `rbac` 即建 RoleBinding;`rules` 非空且未指定 `roleRef.kind=ClusterRole` 时同步建 Role |

通用行为:

- **复制**:`Get()` 源对象 → 写入本端 `data`;源不存在 → 对应 `status.initResources.*[i].ready=false`;
- **漂移**:reconcile 检测到本端 ≠ 源时覆盖;源 watch 不建立,延迟 ≤ resync 间隔;
- **删除**:spec 删项 → reconcile 显式 Delete;Tenant 删除 → ownerReference GC。

**关键不变量**:`serviceAccounts[].imagePullSecrets[]` 中每个 name 必须能在 `imagePullSecrets[].name` 中找到,否则 Validate 失败。这些 Secret 也是 [artifacts.md](artifacts.md) `auth_hint` 链路的落地端。

## 5. 关键机制

### 5.1 多 Tenant 共享 Namespace 隔离

`spec.namespace.name` 允许多 Tenant 指向同一 Namespace。隔离依靠命名前缀 + label:

| 维度 | 隔离方式 |
| --- | --- |
| Namespace 自身 | 共享,无 ownerReference |
| per-tenant Secret / CM / SA | 前缀 `axisml-tenant-<tenant>-<name>`,集群内唯一不 collide |
| ElasticQuota | 前缀 `axisml-<tenant>-<pool>-<quota>`,集群内唯一 |
| selector 检索 | label `axisml.io/tenant-id=<uuid>` |
| Pod 绑定 | 选定 `axisml-tenant-<tenant>-<sa>` SA → 关联本 tenant 的 imagePullSecrets / RBAC;label `quota.scheduling.koordinator.sh/name` → 关联本 tenant 的 ElasticQuota |
| 删除 | Tenant A 删除 → 其 per-tenant 资源被 GC;Tenant B 与 Namespace 自身保留 |

### 5.2 Status 推导与 phase 计算

reconcile 末尾按下表推 phase 并一次性 patch status:

| 条件 | phase |
| --- | --- |
| `spec.suspended == true` | `Suspended` |
| `namespaceReady && all quotas[*].ready && all initResources[*].ready` | `Active` |
| 关键资源 (Namespace / ElasticQuota) 创建失败且非瞬态 | `Failed` |
| 否则 (瞬态创建中) | 维持上一态,`message` 写当前进展 |

`status.conditions[]` 按 `type` (`NamespaceReady` / `QuotasReady` / `InitResourcesReady` / `Suspended` / `Failed`) 去重后整体写回。`spec.quotas` 为空时 `status.quotas` 同空,`Active` 推导只看 namespace + initResources。

### 5.3 ElasticQuota status.used 回流路径

```
Pod 调度 ──▶ koord-scheduler ──▶ ElasticQuota.status.used 累加
                                          │
                                          │ watch (SharedInformerFactory,
                                          │        全集群 ElasticQuota)
                                          ▼
                              tenant-operator
                                          │ ownerReference 反查 Tenant
                                          ▼
                            Tenant.status.quotas[i].used (patch)
                                          │
                                          ▼
                              cluster-manager informer 写入 PG
```

`status.used` 只读,不写回 ElasticQuota;cluster-manager 在 `GET /api/v1/tenants/{name}` 时直接返回。

## 6. 接口契约

| 类别 | 内容 | 引用 |
| --- | --- | --- |
| 输入 CR | `Tenant` (`axisml.io/v1alpha1`, cluster-scoped, `shortName=tnt`) | [tenant-crd.yaml](../../../deploy/helm/axisml-system/crds/tenant-crd.yaml) |
| 上游写者 | cluster-manager 是唯一写者 (`metadata` / `spec`);admission webhook 后续硬阻断外部写 | [cluster-manager.md](cluster-manager.md) |
| status subresource | CRD 声明 `subresources.status`;tenant-operator 是唯一 `status` 写者 | — |
| 字段归属 | `spec` 三大块 `namespace` / `quotas[]` / `initResources` + `displayName` / `suspended` | 详见下表 |
| 级联清理 | per-tenant 资源 ownerReference → Tenant CR;Tenant DELETE 由 K8s GC 异步清理;Namespace 永不删除 | — |
| metadata 命名上限 | `metadata.name` ≤40;子资源前缀 14+40+1=55 → `initResources.*[].name` 与 `serviceAccounts[].name` 上限 198 | — |

**字段归属与可变性** (字段级 schema 见 CRD yaml,本表只列写入边界):

| 字段路径 | 写入方 | 可变? |
| --- | --- | --- |
| `metadata.name` / `labels[axisml.io/tenant-id]` | cluster-manager | 否 |
| `spec.displayName` / `suspended` | cluster-manager | 是 |
| `spec.namespace.name` | cluster-manager | 否 (controller 拒绝,webhook 兜底) |
| `spec.namespace.labels` / `annotations` | cluster-manager | 是 (仅首次创建落地) |
| `spec.quotas[].{pool, name}` | cluster-manager | 否 (标识锚点) |
| `spec.quotas[].{min, max}` | cluster-manager | 是 |
| `spec.initResources.*` | cluster-manager | 是 (增删 → reconcile 创建 / 删除) |
| `status.*` | tenant-operator | — |

## 7. 依赖

| 依赖 | 用途 | 引用 |
| --- | --- | --- |
| Kubernetes API | Tenant CR watch、子资源 CRUD、leader Lease | — |
| Koordinator ElasticQuota CRD | 渲染目标;`status.used` 回流来源 | [infra.md](../infra.md) |
| cluster-manager | 上游唯一 Tenant CR 写者;status 回流消费方 | [cluster-manager.md](cluster-manager.md) |
| 受控 Namespace 中的源 Secret / ConfigMap | `sourceSecretRef` / `sourceConfigMapRef` 复制数据源 | — |

## 8. 运行时形态

| 维度 | 值 |
| --- | --- |
| 进程 | 单二进制 `axisml-tenant-operator`;承载 Manager + Tenant Reconciler |
| 副本 | `replicas=1`,leader election Lease `axisml-tenant-operator.axisml.io` |
| 暴露端口 | metrics `:8080`、health probe `:8081` (`/healthz`, `/readyz`);无对外服务 |
| RBAC scope | ClusterRole:`tenants.axisml.io` (`get/list/watch/patch`)、`namespaces` (`create/get/list/watch/update/patch`,**无 delete**)、`elasticquotas.scheduling.sigs.k8s.io` RW、目标 ns `secrets/configmaps/serviceaccounts/roles/rolebindings` RW、源 ns `secrets/configmaps` RO、`events` `create/patch`;Role:自身 ns `leases` RW |
| Cache 过滤 | 子资源用 `axisml.io/managed-by=tenant-operator` label selector,避免拉全集群;Tenant CR 不过滤 |
| Helm values / 镜像 | `tenantOperator.*`,详见 [deployment.md](../deployment.md) |

## 9. 后续工作

- **Admission webhook**:`spec.namespace.name` / `spec.quotas[].{pool,name}` 不可变约束、跨 ns `sourceXxxRef` 白名单、`min/max` 结构性校验、源 Secret type 一致性、目标 Namespace allowlist / denylist 前移;同时硬阻断非 cluster-manager 写者。
- **CRD 移除 `spec.annotations` 字段**:扩展元数据已统一收回 PG `tenants.{labels,annotations}`（见 [database.md §1.6](../database.md#16-扩展元数据-labels--annotations)）,CR 上不再承载业务扩展位。
- **CRD 严格 schema**:替换 `x-kubernetes-preserve-unknown-fields`,启用 OpenAPI 校验并收紧 `phase` enum。
- **加密源支持**:KMS / Vault / Sealed Secrets 作为 `sourceSecretRef` 替代。
- **`initResources` templating**:按 tenant 上下文 (id / name / namespace) 渲染 ConfigMap 数据。
- **复制源 RBAC 收敛**:把源 Namespace 限定为单一受控 Namespace。
- **分层配额**:`spec.quotas[]` 引入 `parent` 字段,落到 ElasticQuota `quota.scheduling.koordinator.sh/parent` annotation。
- **resync 间隔 Helm values 暴露**:默认 10 min,可调到分钟级。
- **ServiceAccount + RBAC 子能力的 Helm values 开关**。

## 10. 相关引用

- [overview.md](../overview.md) — 控制平面拓扑
- [auth.md](../auth.md) — 平台用户身份与 RBAC 域 (tenant-operator 不直接参与)
- [database.md](../database.md) — `tenants` 表 schema (PG 权威由 cluster-manager 持有)
- [deployment.md](../deployment.md) — Helm 模板与部署形态
- [monitoring.md](../monitoring.md) — Metrics 与告警
- [infra.md](../infra.md) — Koordinator / ElasticQuota / scheduler-plugins 依赖契约
- [cluster-manager.md](cluster-manager.md) — Tenant CR 上游 producer
- [compute-operator.md](compute-operator.md) — 兄弟 operator,承载 MLJob / MLService
- [artifacts.md](artifacts.md) — `auth_hint` 链路依赖 tenant-operator 下发的 SA + ImagePullSecret
