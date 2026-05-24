# AxisML Cluster Manager 概要设计

## 1. 定位与边界

平台的管理员域入口；以 PostgreSQL `tenants` 表作为租户与配额的权威，把 Tenant CR 当作 PG 行的派生产物下发到 K8s。

| 做 | 不做 |
| --- | --- |
| 租户 CRUD、suspend / unsuspend、软删与 restore | Namespace / ElasticQuota / initResources 落地 (→ [tenant-operator.md](tenant-operator.md)) |
| 配额（内联 `tenants.quotas` jsonb）CRUD | 计算负载与制品管理 (→ [compute.md](compute.md) / [artifacts.md](artifacts.md)) |
| Tenant CR 同步 + status 回流 | 用户认证与角色鉴权 (→ [auth.md](../auth.md)) |
| 历史租户保留 + retention GC | 跨集群 / 多 region 联邦 |

## 2. 架构

### 2.1 上下文

```
        ┌──────────────┐  REST + X-Axisml-User   ┌──────────────────┐
        │  Platform    │ ───────────────────────▶│ Cluster Manager  │
        └──────────────┘                          └──────┬───────────┘
                                                         │ PG 读写 / CR patch
                                          ┌──────────────┼──────────────┐
                                          ▼                              ▼
                                ┌─────────────────┐           ┌─────────────────┐
                                │  PostgreSQL     │           │   K8s API       │
                                │  tenants 表     │           │   Tenant CR     │
                                └─────────────────┘           └────────┬────────┘
                                                                       │ watch
                                                                       ▼
                                                            ┌──────────────────┐
                                                            │ tenant-operator  │
                                                            └──────────────────┘
```

### 2.2 内部结构

```
┌────────────────────────── Cluster Manager (Go) ──────────────────────────┐
│  HTTP API (Gin)  ──写──▶  PG tenants 表 (spec + generation)              │
│         ▲                              │                                  │
│         │ 读                            ▼                                  │
│         │                       Reconciler (leader-only)                  │
│         │                       generation > observed_generation          │
│         │                                    → patch CR                   │
│         │                              │                                  │
│         │                              ▼                                  │
│         │                       K8s Tenant CR                             │
│         │                              │ status                            │
│         └────── PG status jsonb ◀── Informer (leader-only)                │
└──────────────────────────────────────────────────────────────────────────┘
```

## 3. 核心模型

| 实体 | 含义 | 标识键 | 备注 |
| --- | --- | --- | --- |
| Tenant | 租户 | `(namespace, name)`；`namespace` 是组织分组（如 `ai-team`），`name` DNS-1123 ≤40 字符；均创建后不可变 | PG 行 + Tenant CR 1:1 |
| Quota | 配额，内联在 `tenants.quotas` jsonb | `(tenant.namespace, tenant.name, pool, name)` | 每条 1:1 渲染为 ElasticQuota CR |

字段级 schema 见 [database.md §2.1](../database.md#21-tenants-表)；Tenant CR spec 字段见 [tenant-operator.md §6 接口契约](tenant-operator.md#6-接口契约)。

## 4. 核心功能

### 4.1 租户管理

**状态机**：

```
[POST]──▶ Creating ──(namespaceReady)──▶ Active ◀─[unsuspend]─ Suspended
                                            │ │
                                            │ └─[suspend]──▶ Suspended
                                            ▼
                                          Failed ──(自愈)──▶ Active

任一活跃态 ──[DELETE 软删]──▶ Deleting ──(CR DELETE)──▶ Deleted
                                                         │
                                                         ├─[restore]─▶ Creating
                                                         └─ retention 到期 ──▶ 物理清理
```

**行为约束**：

| 操作 | PG 写 | CR 影响 | 备注 |
| --- | --- | --- | --- |
| 创建 | insert `Creating` 行（`generation=1`） | reconciler 创建 CR | DNS-1123 校验由 API 层兜底 |
| 更新 spec（`spec.namespace.labels` / `spec.namespace.annotations` / `spec.quotas[].{min,max}` / `spec.initResources` / `spec.suspended`） | update `spec` + `generation += 1` | reconciler patch CR | `spec.namespace.name`、`spec.quotas[].{pool,name}` 不可变 |
| 更新顶层 PG 元数据（`display_name` / `description` / `labels` / `annotations`） | update 行 | **不影响 CR** | 不 `+generation`；扩展位见 [database.md §1.6](../database.md#16-扩展元数据-labels--annotations) |
| 软删 | `deleted_at = now()` + `generation += 1` | reconciler 删 CR | 行保留到 retention |
| 恢复 | `deleted_at = NULL` + `generation += 1` | reconciler 重建 CR | 仅适用 `phase='Deleted'` 行 |

`Failed` 是非终态，operator 自愈后自然回到 `Active`。

### 4.2 配额管理

Quota 没有独立 CRD——是 `tenants.spec.quotas[]` jsonb 中的一项。

| 操作 | PG 写 | CR 影响 |
| --- | --- | --- |
| 新增 | append jsonb 项 + `generation += 1` | reconciler patch `Tenant.spec.quotas[]` |
| 修改 `min` / `max` | update jsonb 项 + `generation += 1` | reconciler patch |
| 删除 | remove jsonb 项 + `generation += 1` | reconciler 删除对应 ElasticQuota CR |
| 用量回流 | informer 写入 `status.quotas[i].used` | — |

**不变约束**：`(pool, name)` 一旦创建即不可变；改名 = 先删后建。

## 5. 关键机制

### 5.1 写路径：内嵌 Outbox + generation

无独立 outbox 表——借用 `tenants` 表自身的 `generation` / `observed_generation` 两列（语义详见 [database.md §1.4](../database.md#14-generation--observed_generation)）。

```
API mutation
     │
     ▼
┌──────────────────────── 单事务 ─────────────────────────┐
│  1. UPDATE tenants SET spec = ..., generation = generation + 1  │
│  2. COMMIT                                                       │
└──────────────────────────────────────────────────────────────────┘
     │
     ▼ partial index `WHERE generation <> observed_generation`
     │
     ▼
Reconciler (10s tick, leader-only)
     │
     ▼
┌─────────────────────────────────────────────────────────┐
│  deleted_at ≠ NULL → K8s Delete()                       │
│  else              → render + Patch() (server-side apply)│
│  success           → observed_generation = generation    │
└─────────────────────────────────────────────────────────┘
```

进入 CR 的字段仅来自 `tenants.spec`（即 `spec.namespace` / `spec.quotas` / `spec.initResources` / `spec.suspended`）加上顶层 `name` / `display_name`；顶层的 `namespace`（组织分组，仅用于标识与 PG 索引）/ `description` / `labels` / `annotations` 等 PG-only 字段**不进 CR**，因此不影响 `generation`。

### 5.2 状态回流（Informer）

watch Tenant CR：

| 事件 | 写回 PG | 不动 |
| --- | --- | --- |
| ADD / UPDATE | `status` 整块（含 `phase` / `conditions` / `quotas[].used`） | `spec` 列、`generation` |
| DELETE | `phase = 'Deleted'` | 其他字段保留供历史查询 |

### 5.3 外部漂移修正

外部 `kubectl edit tenant` 修改 CR spec 时：

- Informer **不修正 PG**（PG 是权威）；
- 下一轮 reconciler tick 发现 CR spec ≠ PG 期望 → 重新 patch CR 擦回 PG 状态；
- 后续 admission webhook 上线后直接拒绝非 cluster-manager 的写请求 (见 [§9](#9-后续工作))。

## 6. 接口契约

| 类别 | 内容 | 引用 |
| --- | --- | --- |
| 对外 REST | `/api/v1/namespaces/{namespace}/tenants[...]`、`.../tenants/{name}/{suspend,unsuspend,restore}`、`.../tenants/{name}/quotas[...]` | [apis/cluster-manager.yaml](../apis/cluster-manager.yaml) `Tenants` / `Quotas` tag |
| 下发 CR | `Tenant`（`axisml.io/v1alpha1`, cluster-scoped），cluster-manager 是唯一写者 | [tenant-operator.md §6 接口契约](tenant-operator.md#6-接口契约) |
| 回流字段 | `phase` / `namespaceReady` / `conditions` / `quotas[].used` 通过 GET 返回 | — |
| 身份头 | 调用方注入 `X-Axisml-User`，本服务仅做审计 | [auth.md §7](../auth.md#7-下游身份透传) |
| 错误格式 | HTTP 标准状态码 + RFC 7807 problem+json | — |
| 写后语义 | mutation 在 PG 提交后即返回；CR 同步异步进行，调用方通过 GET 观察 phase | — |

## 7. 依赖

| 依赖 | 用途 | 引用 |
| --- | --- | --- |
| PostgreSQL | 租户元数据权威；与 compute / artifacts 共享 database，表前缀隔离 | [database.md](../database.md) / [infra.md](../infra.md) |
| Kubernetes API | Tenant CR 下发 + status watch + leader Lease | — |
| Platform | 唯一调用方；身份注入 | [auth.md](../auth.md) |
| tenant-operator | 下游消费者，把 Tenant CR 落地为 K8s 资源 | [tenant-operator.md](tenant-operator.md) |

## 8. 运行时形态

| 维度 | 值 |
| --- | --- |
| 进程 | 单二进制 `axisml-cluster-manager`；子命令 `serve` / `migrate` |
| 副本 | API 默认 `replicas=2`（无状态）；reconciler / informer 单 leader (K8s Lease) |
| 暴露 | ClusterIP `:8082`，不对外；探针 `/healthz` / `/readyz` |
| RBAC scope | `tenants.axisml.io` 全权 + `elasticquotas` RO + 自身 ns 的 `leases` |
| Helm values / 镜像 | 详见 [deployment.md](../deployment.md) |

## 9. 后续工作

- Admission webhook：硬阻断非 cluster-manager 的 Tenant CR 写请求。
- 批量端点：`POST /api/v1/namespaces/{namespace}/tenants:batchCreate` 等便于 Platform 批量初始化。
- 独立 `cluster_manager_audits` 表记录 PG mutation 流水（与 Platform `audit_logs` 分层）。
- 外部漂移自动告警（Prometheus + K8s Event）。
- Retention GC 守护：定期物理清理超期 `Deleted` 行并暴露指标。
- 配额用量周期采样，支持趋势分析与 Platform 报表回填。

## 10. 相关引用

- [overview.md](../overview.md) — 控制平面拓扑
- [auth.md](../auth.md) — 身份与鉴权契约
- [database.md](../database.md) — `tenants` 表 schema
- [deployment.md](../deployment.md) — Helm / 部署
- [monitoring.md](../monitoring.md) — Metrics 与告警
- [apis/cluster-manager.yaml](../apis/cluster-manager.yaml) — REST 契约源
- [tenant-operator.md](tenant-operator.md) — Tenant CR 字段与落地行为
- [compute.md](compute.md) — 同形态 PG + 下游 CR 业务服务样板
