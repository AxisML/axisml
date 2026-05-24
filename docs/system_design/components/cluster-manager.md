# AxisML Cluster Manager 概要设计

## 1. 定位与边界

集群级 admin 词汇服务：以 PostgreSQL 为权威，承载 ResourcePool（节点切分维度）与 ResourceUnit（池内资源规格模板）的元数据。两者纯 PG 配置，无 CRD、无 K8s 调用；admin 离线维护节点 label / taint 即可。

| 做 | 不做 |
| --- | --- |
| ResourcePool / ResourceUnit CRUD | 修改 Node label / taint（admin 手工维护） |
| 默认池 `default` 初始化（Helm post-install） | 租户 / 配额管理 (→ [compute.md](compute.md)) |
| Pool / Unit 列表与详情查询，给 Platform 编排前展开 nodeSelector / requests / limits 用 | 计算负载与制品管理 (→ [compute.md](compute.md) / [artifacts.md](artifacts.md)) |
| 用户认证与角色鉴权 (← Platform) | 调用任何 K8s API |

Cluster Manager 没有 CR、没有 reconciler、没有 informer、没有 leader election。是纯 REST + PG。

## 2. 架构

### 2.1 上下文

```
        ┌──────────────┐  REST + X-Axisml-User   ┌──────────────────┐
        │  Platform    │ ───────────────────────▶│ Cluster Manager  │
        └──────────────┘                          └──────┬───────────┘
                                                         │ PG 读写
                                                         ▼
                                                ┌─────────────────┐
                                                │  PostgreSQL     │
                                                │  resource_*     │
                                                └─────────────────┘
```

Platform 在编排 Job / Service 时调本服务拿 Pool / Unit 详情，展开成 nodeSelector + tolerations + requests / limits 原语，再透传给 compute。Cluster Manager 自己不感知 Job / Service。

### 2.2 内部结构

```
┌──────── Cluster Manager (Go) ────────┐
│  HTTP API (Gin) ──┬─▶ PG (resource_pools)
│                   └─▶ PG (resource_units)
└──────────────────────────────────────┘
```

无 leader election / 无 worker goroutine。多副本对等运行。

## 3. 核心模型

| 实体 | 含义 | 标识键 | 备注 |
| --- | --- | --- | --- |
| ResourcePool | 节点切分维度（GPU 池 / CPU 池 / 训练池） | `name`（全局唯一） | 无 CR；admin 给目标节点打 label/taint |
| ResourceUnit | 池内资源规格模板（含 requests/limits/nodeSelector） | `(pool, name)` | 无 CR；纯 PG 元数据 |

字段级 schema 见 [database.md §2](../database.md#2-cluster-manager)。

### 3.1 ResourcePool

| 字段 | 用途 |
| --- | --- |
| `name` | DNS-1123，全局唯一 |
| `node_selector` | Platform 注入到 Job/Service `spec.scheduling.nodeSelector`（Pool 优先） |
| `tolerations` | Platform 直接作为 `spec.scheduling.tolerations` |

**默认池**：Helm post-install Job 初始化 `default` 池，`node_selector` 为空表示整集群可用。

### 3.2 ResourceUnit

池内资源规格模板。

| 字段 | 用途 |
| --- | --- |
| `(pool, name)` | partial UNIQUE |
| `requests` / `limits` | Platform 注入到 CR `spec.roles[*].template.resources` |
| `node_selector` | 与 Pool 合并后注入 `spec.scheduling.nodeSelector` |

**命名约定**：`<accelerator>[-<count>x]-<tier>[-<variant>]`，例如 `cpu-small` / `a100-1x-large` / `h100-8x-xlarge-ib` / `tpu-v4-4x-large`。`<tier>` ∈ `small | medium | large | xlarge`。

**展开合并规则**（由 Platform 在调 compute 前完成）：

| 来源 | 合并行为 |
| --- | --- |
| `pool.node_selector` | key 全部保留 |
| `unit.node_selector` | 仅贡献 Pool 未声明的 key |
| `pool.tolerations` | 直接作为 `spec.scheduling.tolerations` |
| `unit.requests` / `limits` | 写入 `spec.roles[*].template.resources` |

合并参考实现详见 [platform.md §4.2](platform.md#42-计算任务编排) 的注入步骤。

## 4. 核心功能

### 4.1 ResourcePool CRUD

| 操作 | PG 写 |
| --- | --- |
| 创建 | insert 行 |
| PATCH | update 行（`name` 不可变）|
| 软删 | `deleted_at = now()`；返回 `409 pool-in-use`，若被 ResourceUnit 引用 |
| 列表 | 分页 + `?labelSelector=` 过滤 |

### 4.2 ResourceUnit CRUD

| 操作 | PG 写 |
| --- | --- |
| 创建 | insert 行；`(pool, name)` 唯一 |
| PATCH | update 行（`pool` / `name` 不可变）|
| 软删 | `deleted_at = now()`；不影响已创建的 Job/Service 资源快照（Platform 在创建时已展开） |
| 列表 | `/api/v1/resource-pools/{pool}/resource-units`，可加 `?labelSelector=` |

注：Unit 修改不会影响已创建的 Job/Service——Platform 在创建时已经把 unit 展开成具体 requests/limits 写进 compute 请求体，compute 存的是展开后的原语，跟 Unit 行解耦。

## 5. 关键机制

无异步、无 reconciler。所有 mutation 在单事务内完成、立即可读。

## 6. 接口契约

| 类别 | 内容 | 引用 |
| --- | --- | --- |
| 对外 REST | `/api/v1/resource-pools[/{pool}]`、`/api/v1/resource-pools/{pool}/resource-units[/{unit}]` | [apis/cluster-manager.yaml](../apis/cluster-manager.yaml) `ResourcePools` / `ResourceUnits` tag |
| 身份头 | 调用方注入 `X-Axisml-User`，本服务仅做审计 | [auth.md §7](../auth.md#7-下游身份透传) |
| 错误格式 | HTTP 标准状态码 + RFC 7807 problem+json | — |
| 写后语义 | mutation 在 PG 提交后即返回；强一致 | — |

## 7. 依赖

| 依赖 | 用途 | 引用 |
| --- | --- | --- |
| PostgreSQL | 元数据权威；与 compute / artifacts 共享 database `axisml`，表前缀隔离 | [database.md §2](../database.md#2-cluster-manager) |
| Platform | 唯一调用方；编排前查询 Pool/Unit 展开成原语 | [platform.md §4.2](platform.md#42-计算任务编排) |

不依赖 Kubernetes API、tenant-operator、compute、artifacts。

## 8. 运行时形态

| 维度 | 值 |
| --- | --- |
| 进程 | 单二进制 `axisml-cluster-manager`；子命令 `serve` / `migrate` / `bootstrap`（创建默认池） |
| 副本 | 任意（无状态对等运行；无 leader election） |
| 暴露 | ClusterIP `:8082`，不对外；探针 `/healthz` / `/readyz`（仅校验 PG） |
| RBAC scope | 无 K8s API 需求 |
| Helm values / 镜像 | 详见 [deployment.md](../deployment.md) |

## 9. 后续工作

- Pool 池间调度策略（配额不足时是否允许跨池借用，默认禁止）+ 池容量预估。
- ResourceUnit 混合资源单元（CPU + MIG 分片）与价格元数据用于成本核算。
- Pool / Unit 用量回流：Platform / compute 周期上报，便于 admin 看到每个池的实际占用。

## 10. 相关引用

- [overview.md](../overview.md) — 控制平面拓扑
- [auth.md](../auth.md) — 身份与鉴权契约
- [database.md](../database.md) — `resource_pools` / `resource_units` 表 schema（§2）
- [deployment.md](../deployment.md) — Helm / 部署
- [monitoring.md](../monitoring.md) — Metrics
- [apis/cluster-manager.yaml](../apis/cluster-manager.yaml) — REST 契约源
- [platform.md](platform.md) — Platform 在编排前调 Pool/Unit 做展开
- [compute.md](compute.md) — Tenant / Job / Service 权威服务
