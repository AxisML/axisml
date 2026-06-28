# AxisML Tenant Operator 设计

## 1. 定位与边界

把 [cluster-manager](cluster-manager.md) 下发的 `Tenant` CR 翻译为 K8s 侧的 Namespace、Koordinator `ElasticQuota`、租户私有的 Secret / ConfigMap / ServiceAccount + RBAC，并把执行状态回流到 `Tenant.status`。

| 做 | 不做 |
| --- | --- |
| Namespace 创建与 metadata 对齐（永不删除） | Tenant CR / 配额的 CRUD API (→ [cluster-manager.md](cluster-manager.md)) |
| 每条 `spec.quotas[]` 渲染为一个 ElasticQuota CR，回流 `status.used` | MLRun / MLService 生命周期 (→ [compute-operator.md](compute-operator.md)) |
| `spec.initResources` 下发 ImagePullSecret / Secret / ConfigMap / SA + RBAC | 用户认证、平台 RBAC (→ [auth.md](../../../axisml-platform/docs/system_design/auth.md)) |
| 周期 resync 收敛源 Secret / ConfigMap 漂移 | 跨集群 / 多 region 联邦 |

## 2. 架构

```
   cluster-manager ──write Tenant CR──▶ K8s API (Tenant CR)
                                              │ watch
                                       tenant-operator
                          create/patch/GC     │
            ┌──────────────────┬──────────────┴──────────────┐
            ▼                  ▼                            ▼
        Namespace      ElasticQuota(Koordinator)    Secret/CM/SA + Role/RB
                              │ status.used 回流
                              ▼
                   Tenant.status.quotas[].used
```

```
┌──── tenant-operator (one Pod, leader-elected) ────┐
│ ctrl.Manager  scheme: clientgo + axisml.tenant +  │
│   Koordinator ElasticQuota                        │
│ cache: managed-by=tenant-operator selector        │
│ ┌ Tenant Reconciler (单一) ┐                       │
│ │ Validate → Namespace → ElasticQuota[] →         │
│ │ initResources → phase 推导 + 单次 status patch  │
│ └──────────────────────────┘                       │
└────────────────────────────────────────────────────┘
```

无 dispatcher / handler 分层；所有 Tenant CR 走同一 reconciler。

## 3. 核心模型

| 实体 | 含义 | 标识键 | 备注 |
| --- | --- | --- | --- |
| Tenant | 租户 CR，cluster-scoped | `metadata.name`（DNS-1123, ≤40） | 上游唯一写者为 cluster-manager |
| Kubernetes Namespace | 运行租户 Pod 的物理 namespace | `spec.namespace.name` | 与 tenant scope 分离，可被多 Tenant 共享（§5.1） |
| ElasticQuota | Koordinator 配额 CR | `axisml-<tenant>-<pool>-<quota>` | 每条 `spec.quotas[]` 1:1 渲染（`min`/`max` 由 cluster-manager 折算后写入） |
| InitResource | per-tenant Secret / CM / SA + RBAC | `axisml-tenant-<tenant>-<name>` | 由 `sourceXxxRef` 复制 |

Tenant CR 字段见 [tenant-crd.yaml](../../deploy/helm/crds/tenant-crd.yaml)；ElasticQuota 调度行为见 [infra.md](../../../axisml-infra/docs/system_design/overview.md)。

## 4. 核心功能

### 4.1 Tenant Controller

承载三件事：Namespace 落地、ElasticQuota 派生、初始化资源下发。统一不变量：不引入 finalizer（级联清理依赖 ownerReference）；`Validate(spec)` 纯函数（失败 → `phase=Failed` + message）；status 单次 patch（reconcile 末尾整体写回 `conditions[]` 与 phase）；per-tenant 子资源叠加 `axisml.io/tenant-id=<uuid>` + `axisml.io/managed-by=tenant-operator` label。

| 事件 | 行为 |
| --- | --- |
| Tenant ADD / UPDATE(spec) | Validate → 按 §4.1.1 → §4.1.2 → §4.1.3 顺序就位，末尾推 phase |
| Tenant DELETE | 不阻断；K8s GC 按 ownerReference 清子资源；Namespace 不删除 |
| 子资源事件 | 按 ownerReference 反查触发 reconcile；ElasticQuota `status.used` 只刷新 `Tenant.status.quotas[i].used` |
| 周期 resync（默认 10 min） | 重读 `sourceSecretRef` / `sourceConfigMapRef` 收敛漂移 |

#### 4.1.1 Namespace 落地

命名 `spec.namespace.name`（创建后不可变），它是物理 K8s Namespace，不要求等于 Tenant `metadata.name`。不存在 → 创建，附 `spec.namespace.labels`/`annotations` + `axisml.io/managed-by`，受 denylist 约束（默认拒 `kube-*` / `default` / `axisml-system` / `axisml-infra`）；已存在 → 仅补 `managed-by` label，**不覆盖**既有 label / annotation；**不设 ownerReference**（不属任何单一 Tenant）；**永不删除**（避免误删 PV / 外部资源；空 Namespace 由管理员清理）。`status.namespaceReady` 在 Namespace `phase=Active` 时为 `true`。

#### 4.1.2 ElasticQuota 落地

每个 `(tenant, pool)` 一个 Koordinator `ElasticQuota`，均落 `spec.namespace.name` 下；`min`/`max` 由 cluster-manager 据 ResourceUnit 规格折算后写入 CR，operator 直传落地。

| 维度 | 行为 |
| --- | --- |
| 命名 | `axisml-<tenant>-<pool>-<quota>`（集群内唯一） |
| 创建 / 删除 | `spec.min`/`max` 直传；空数组不创建；spec 增删项 → Create / Delete 对应 CR |
| ownerReference | Tenant CR |
| 漂移 | reconcile 按 `spec.quotas[i].{min,max}` 覆盖 |
| status.used 回流 | watch ElasticQuota，把 `status.used` 写入 `Tenant.status.quotas[i].used`；不写回 ElasticQuota |

**不变量**：`quotas[].pool` 唯一且创建后不可变；`min[k] ≤ max[k]` 且均 ≥ 0。Pod 经 label `quota.scheduling.koordinator.sh/name=axisml-<tenant>-<pool>` 跨 namespace 绑定其 pool 的 ElasticQuota。

#### 4.1.3 初始化资源落地

`spec.initResources` 下的资源走同一套"从受控 Namespace 复制"模式：

| 资源 | 命名 | 数据来源 / 行为 |
| --- | --- | --- |
| ImagePullSecret | `axisml-tenant-<tenant>-<name>` | `sourceSecretRef`；type 固定 `dockerconfigjson` |
| 通用 Secret | `axisml-tenant-<tenant>-<name>` | `sourceSecretRef`；type 默认 `Opaque`；K8s Secret type 不可变 → 改 type 触发删除重建 |
| ConfigMap | `axisml-tenant-<tenant>-<name>` | `sourceConfigMapRef` |
| ServiceAccount | `axisml-tenant-<tenant>-<name>` | 解析 `imagePullSecrets[]`（用户可见 name → 最终 Secret 名）后写入 SA |
| Role / RoleBinding | `axisml-tenant-<tenant>-<sa>` | 声明 `rbac` 即建 RoleBinding；`rules` 非空且未指定 `roleRef.kind=ClusterRole` 时同步建 Role |

通用行为：**复制**（`Get()` 源对象 → 写本端 `data`；源不存在 → 对应 `ready=false`）；**漂移**（reconcile 检测本端 ≠ 源时覆盖，源 watch 不建立，延迟 ≤ resync）；**删除**（spec 删项 → 显式 Delete；Tenant 删除 → ownerReference GC）。

**不变量**：`serviceAccounts[].imagePullSecrets[]` 中每个 name 必须能在 `imagePullSecrets[].name` 找到，否则 Validate 失败。per-tenant SA + 默认 imagePullSecrets / Secret 是 workload 拉取 zot / RustFS 的凭证来源；Artifact Hub 不签发或返回 K8s Secret 引用。

## 5. 关键机制

### 5.1 多 Tenant 共享 Namespace 隔离

`spec.namespace.name` 允许多 Tenant 指向同一 K8s Namespace；tenant scope 仍是各自的 Tenant `metadata.name`，物理资源隔离靠命名前缀 + tenant ID label：

| 维度 | 隔离方式 |
| --- | --- |
| Namespace 自身 | 共享，无 ownerReference |
| per-tenant Secret / CM / SA | 前缀 `axisml-tenant-<tenant>-<name>`，集群内唯一不冲突 |
| ElasticQuota | 前缀 `axisml-<tenant>-<pool>`，集群内唯一 |
| Pod 绑定 | 选定 `axisml-tenant-<tenant>-<sa>` SA → 关联本 tenant 的 imagePullSecrets / RBAC；label `quota.scheduling.koordinator.sh/name` → 关联本 tenant 的 ElasticQuota |
| 删除 | Tenant A 删除 → 其 per-tenant 资源被 GC；Tenant B 与 Namespace 自身保留 |

### 5.2 Status 推导与 phase 计算

reconcile 末尾按下表推 phase 并一次性 patch status：

| 条件 | phase |
| --- | --- |
| `namespaceReady && all quotas[*].ready && all initResources[*].ready` | `Active` |
| 关键资源（Namespace / ElasticQuota）创建失败且非瞬态 | `Failed` |
| 否则（瞬态创建中） | 维持上一态，`message` 写当前进展 |

`status.conditions[]` 按 `type`（`NamespaceReady` / `QuotasReady` / `InitResourcesReady` / `Failed`）去重后整体写回。`spec.quotas` 为空时 `status.quotas` 同空，`Active` 推导只看 namespace + initResources。

### 5.3 ElasticQuota status.used 回流路径

```
Pod 调度 ─▶ koord-scheduler ─▶ ElasticQuota.status.used 累加
                                     │ watch (SharedInformerFactory, 全集群 ElasticQuota)
                                     ▼
                            tenant-operator ─ownerReference 反查─▶ Tenant.status.quotas[i].used (patch)
                                                                          │ GET 时实时读 CR status
                                                                          ▼
                                                              cluster-manager (无 cache, 不入 PG)
```

`status.used` 只读，不写回 ElasticQuota；不被任何服务持久化，只活在 Tenant CR `status.quotas[].used`。cluster-manager 在 `GET /tenants/{tenant}` 时直读 CR `status` 实时返回。调度即变的 ephemeral 态权威源单一收敛到 ElasticQuota CR（经 Tenant CR 暴露）。详见 [cluster-manager.md §4.3](cluster-manager.md#43-tenant-crud)。

## 6. 接口契约

| 类别 | 内容 |
| --- | --- |
| 输入 CR | `Tenant`（`axisml.io/v1alpha1`, cluster-scoped, `shortName=tnt`）；schema 见 [tenant-crd.yaml](../../deploy/helm/crds/tenant-crd.yaml) |
| 上游写者 | cluster-manager 是唯一写者（`metadata`/`spec`）；admission webhook 后续硬阻断外部写 |
| status subresource | CRD 声明 `subresources.status`；tenant-operator 是唯一 `status` 写者 |
| 级联清理 | per-tenant 资源 ownerReference → Tenant CR；Tenant DELETE 由 K8s GC 异步清理；Namespace 永不删除 |
| 命名上限 | `metadata.name` ≤40；子资源前缀 14+40+1=55 → `initResources.*[].name` 与 `serviceAccounts[].name` 上限 198 |

**字段归属与可变性**（字段级 schema 见 CRD yaml）：`metadata.name` / `labels[tenant-id]` / `spec.namespace.name` / `spec.quotas[].pool` 不可变（cluster-manager 写）；`spec.quotas[].{min,max}`（折算）/ `spec.initResources.*` 可变；`status.*` 由 tenant-operator 写。

**防御等级**：`metadata` / `spec` 单写约束当前由 controller `Validate(spec)` 软兜底，**不防外部 `kubectl patch`**——系统在控制面信任边界内部署，admission webhook 是后续硬化路径。

**展示性元数据归属**：`display_name` / `description` 的权威在上游 Platform 的 `tenants` 表，**不**下发到 Tenant CR；tenant-operator 与 cluster-manager 都不感知。租户的 `labels` / `annotations` 经 cluster-manager 透传到 CR `metadata`，但 tenant-operator 不据此 reconcile、不影响 phase。

## 7. 依赖

| 依赖 | 用途 |
| --- | --- |
| Kubernetes API | Tenant CR watch、子资源 CRUD、leader Lease |
| Koordinator ElasticQuota CRD | 渲染目标；`status.used` 回流来源（[infra.md](../../../axisml-infra/docs/system_design/overview.md)） |
| cluster-manager | 上游唯一 Tenant CR 写者；status 回流消费方（GET 时直读 CR）（[cluster-manager.md](cluster-manager.md)） |
| 受控 Namespace 中的源 Secret / ConfigMap | `sourceSecretRef` / `sourceConfigMapRef` 复制数据源 |

## 8. 运行时形态

| 维度 | 值 |
| --- | --- |
| 进程 | 单二进制 `axisml-tenant-operator`；承载 Manager + Tenant Reconciler |
| 副本 | `replicas=1`，leader election Lease `axisml-tenant-operator.axisml.io` |
| 暴露端口 | Metrics `:8081`、Probes `:8082`；无 API 端口，无对外服务 |
| RBAC scope | ClusterRole：`tenants.axisml.io`（`get/list/watch/patch`）、`namespaces`（`create/get/list/watch/patch`，**无 update/delete**）、`elasticquotas.scheduling.sigs.k8s.io` RW、目标 ns `secrets/configmaps/serviceaccounts/roles/rolebindings` RW、源 ns `secrets/configmaps` RO、`events` `create/patch`；`roles` 额外授 `bind/escalate`、`clusterroles` 授 `bind`（apiserver 的 RBAC escalation 检查拒绝创建 operator 自身未持有规则的 Role，亦拒绝绑定 operator 无权 bind 的 Role/ClusterRole）；Role：自身 ns `leases` RW |
| Cache 过滤 | 子资源用 `axisml.io/managed-by=tenant-operator` selector，避免拉全集群；Tenant CR 不过滤 |
| Helm / 镜像 | `tenantOperator.*`，见 [deployment.md](../../../docs/deployment.md) |

## 9. 相关引用

- [high_level_design.md](../../../docs/high_level_design.md) — 控制平面拓扑与系统不变量
- [auth.md](../../../axisml-platform/docs/system_design/auth.md) · [deployment.md](../../../docs/deployment.md) · [infra.md](../../../axisml-infra/docs/system_design/overview.md)
- [cluster-manager.md](cluster-manager.md) — Tenant CR 上游 producer（REST 写 spec）
- [compute-operator.md](compute-operator.md) — 兄弟 operator
- [artifact-hub.md](artifact-hub.md) — workload 消费制品依赖 per-tenant SA + 默认 Secret 落地
