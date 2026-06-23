# AxisML Cluster Manager 设计

## 1. 定位与边界

集群级 admin REST 抽象层：把 admin 视角的 K8s 写 / 读收敛成简单 REST，让上游全程不直接调 Kubernetes API。承载 `ResourcePool` CRD 的 CRUD（含内嵌 `units[]`）与集群级 `Tenant` CRD 的 CRUD（含配额「资源单元 × 数量」折算为 ElasticQuota `min`/`max` 写入 CR）。仅接受 Platform 内部调用并信任 `X-Axisml-User` 透传。

| 做 | 不做 |
| --- | --- |
| ResourcePool CRD CRUD（pool + 内嵌 units） | 修改 Node label / taint（admin 手工维护） |
| Tenant CRD CRUD + 配额折算（「资源单元 × 数量」→ ElasticQuota `min`/`max`） | 租户持久记录 / 展示元数据 / 停用状态（→ 上游业务编排层） |
| 默认 `default` pool 初始化（Helm post-install） | Namespace / ElasticQuota / initResources 落地 (→ [tenant-operator.md](tenant-operator.md)) |
| `?labelSelector=` 列表过滤 | 计算负载 / 工作区 PVC / Pod 日志（属 workload 域，归 compute） |

形态：**REST 网关层**，无独立持久化（CRD 落 K8s etcd），无 reconciler，无 leader election，多副本对等运行。

## 2. 架构

```
   上游调用方 ──REST + X-Axisml-User──▶ Cluster Manager ──K8s API──▶ etcd
                                                                  (ResourcePool / Tenant)
                                                       ▲ watch              ▲ watch
                                               compute-service       tenant-operator
                                               (展开消费方)          (落地消费方)
```

上游调本服务做 pool/unit 与 tenant CRUD。两个下游都不调本服务、也不感知本服务——compute 经 Informer 直 watch ResourcePool CR cache，创建时按 `(poolName, unitName)` 展开（[compute-service.md §5.4](compute-service.md#54-resourcepool-展开)）；tenant-operator watch Tenant CR 落地 Namespace / ElasticQuota / 初始化资源（[tenant-operator.md](tenant-operator.md)）。所有路径经 etcd 收敛，无直接调用。

```
┌──────── Cluster Manager (Go) ────────┐
│ HTTP API (Gin)                        │
│   ├─ ResourcePool CRUD（GET/PATCH 整体 CR）       │
│   ├─ units 子路径 CRUD（patch spec.units[]）      │
│   ├─ Tenant CRUD（读 Pool 折算配额 → 写 CR）       │
│   └─ tenant quotas 子路径 CRUD（patch spec.quotas[]）│
│ K8s client + 可选 Pool Informer (list 缓存)        │
└────────────────────────────────────────┘
```

无 reconciler / worker / leader election；ResourcePool Informer 可选（仅作 list 加速）。配额折算时按名直读 ResourcePool CR 取 ResourceUnit 规格；Tenant 运行态（phase / conditions / `quotas[].used`）在 GET 时从 CR `status` 实时读取，不建 cache。

## 3. 核心模型

| 实体 | 含义 | 标识键 | 备注 |
| --- | --- | --- | --- |
| ResourcePool | 节点切分维度 + 内嵌 `units[]` | `metadata.name`（cluster-scoped，全局唯一） | admin 给目标节点打 label/taint；schema 见 [resource-pool-crd.yaml](../../deploy/helm/crds/resource-pool-crd.yaml) |
| Tenant | 租户的 K8s 物化 CR：namespace + 配额 + 初始化资源 | `metadata.name` = `identifier`（cluster-scoped，全局唯一） | 本服务写 `spec`，tenant-operator 写 `status`；schema 见 [tenant-crd.yaml](../../deploy/helm/crds/tenant-crd.yaml) |

> 无独立 ResourceUnit CR——`units` 是 `ResourcePool.spec.units[]` 数组项，与 pool 同生灭、原子编辑。无独立配额 CRD——内联 `Tenant.spec.quotas[]`，折算后写入（§3.3）。

### 3.1 ResourcePool 形状

```yaml
apiVersion: axisml.io/v1alpha1
kind: ResourcePool
metadata:
  name: gpu-a100                  # DNS-1123, 集群内全局唯一
spec:
  nodeSelector: { axisml.io/pool: gpu-a100 }   # 池级（Pool 优先）
  tolerations:                                  # 直接作为 spec.scheduling.tolerations
    - { key: nvidia.com/gpu, operator: Exists, effect: NoSchedule }
  units:                                        # 数组内 name 唯一；同 pool 一起删
    - name: a100-1x-large
      requests: { cpu: "16", memory: 128Gi, nvidia.com/gpu: "1" }
      limits:   { cpu: "16", memory: 128Gi, nvidia.com/gpu: "1" }
      nodeSelector: {}                          # 可选；仅贡献 pool 未声明的 key
```

**字段不变性**：`metadata.name` 与 `units[i].name` 不可变（标识锚点）；`spec.nodeSelector` / `tolerations` / `units[i].{requests,limits,nodeSelector,annotations}` 可变。
**unit 命名约定**：`<accelerator>[-<count>x]-<tier>[-<variant>]`（`<tier>∈small|medium|large|xlarge`，`cpu` 类省略 `<count>x`）。
**默认池**：Helm post-install Job 经本服务 REST 创建 `default` pool（带 `cpu-small` / `cpu-medium` 两 unit），`nodeSelector` 空表示整集群可用。

### 3.2 展开合并规则

由 [compute-service §5.4](compute-service.md#54-resourcepool-展开) 在创建时完成（不由本服务或上游完成）：`pool.nodeSelector` 全保留；`unit.nodeSelector` 仅补 pool 未声明的 key；`pool.tolerations` 直作 `spec.scheduling.tolerations`；`unit.requests/limits` 写入 `roles[*].template.resources`。展开结果 snapshot 进 PG，与 CR 解耦——pool/unit 后续修改 / 删除不影响已创建 workload。

### 3.3 Tenant 形状与配额折算

```yaml
apiVersion: axisml.io/v1alpha1
kind: Tenant
metadata:
  name: team-alpha                # = identifier；DNS-1123，全局唯一
  annotations:
    axisml.io/quotas: '[{"pool":"gpu-a100","units":[...]}]'  # 业务形态配额（GET 往返用）
    axisml.io/last-modified-by: <user>                        # 来自 X-Axisml-User
spec:
  namespace: { name: axisml-tenant } # 物理 K8s Namespace，可共享；创建后不可变
  quotas:                         # 每 pool 一项；min/max 由折算写入
    - { pool: gpu-a100, name: gpu-a100,
        min: {cpu:"16",memory:128Gi,nvidia.com/gpu:"1"},
        max: {cpu:"32",memory:256Gi,nvidia.com/gpu:"2"} }
  initResources: { ... }          # ImagePullSecret / Secret / ConfigMap / SA + RBAC
```

REST 入参以业务形态 `{pool, units:[{unitName, quantity}]}` 表达配额；cluster-manager 按名读 `ResourcePool` CR，取每个 `unitName` 的 `requests` / `limits`，折算 `min = Σ(unit.requests × quantity)` / `max = Σ(unit.limits × quantity)`，写入 `spec.quotas[]`（`name` = `pool`）。折算有损，故业务原始选择 JSON 编码回存到 `axisml.io/quotas` annotation，GET 时据此还原 `units` 形态返回（tenant-operator 不读该 annotation，只消费 `spec.quotas[].min/max`）。

**折算与校验**：`quotas[].pool` 必须存在对应 CR（否则 `422 pool-not-found`）；`units[].unitName` 必须存在于该 pool（否则 `422 unit-not-found`）；`quantity ≥ 0`（否则 `400 bad-quantity`）；空 `units` → 该 pool 配额为零。
**字段不变性**：`metadata.name`（= identifier）/ `spec.namespace.name` / `quotas[].pool` 不可变；`quotas[].units[]→min/max`（折算写入）与 `initResources.*` 可变；`status.*` 由 tenant-operator 写、GET 时实时读。

## 4. 核心功能

### 4.1 ResourcePool CRUD

| 操作 | 内部行为 |
| --- | --- |
| 创建 POST `/api/v1/resourcepools` | K8s create CR；admission 校验 name / units name 唯一 |
| GET / LIST | 读 K8s API（或 Informer cache）；list 支持 `?labelSelector=` |
| PATCH（pool 级字段） | strategic merge patch；`metadata.name` 不可变 |
| DELETE | delete CR；units 跟随一起删 |

### 4.2 Unit 子路径 CRUD（UI 友好便利端点）

`GET/POST .../resourcepools/{pool}/units` 与 `PATCH/DELETE .../units/{name}`：每个端点映射为"读 pool CR → 局部改 `spec.units[]` → JSON Patch（带 resourceVersion 乐观锁，重试一次）"的原子封装，客户端无需关心 CR 整体形状。unit 修改后已创建的 Job/Service **不感知**（§3.2 snapshot）。

### 4.3 Tenant CRUD

| 操作 | 内部行为 |
| --- | --- |
| 创建 POST `/api/v1/tenants` | 校验 `identifier`（DNS-1123）；按 `quotas[]` 读 Pool 折算 min/max（§3.3）；create CR，透传调用方 `labels`/`annotations`，回存 `axisml.io/quotas` 与 `last-modified-by` |
| GET / LIST | GET 合并 `spec` 与 CR `status`（phase / conditions / `quotas[].used`）实时返回；list 支持 `?labelSelector=` |
| PATCH | JSON Patch（乐观锁重试）；`metadata.name` / `spec.namespace.name` / `quotas[].pool` 不可变；display 元数据不在此（归上游表） |
| DELETE | delete CR（硬删，幂等 204）；子资源经 ownerReference GC，Namespace 永不删除 |

> cluster-manager 不持有停用语义或历史归档；这些是上游业务编排层的职责。本服务只物化 / 回收 Tenant CR。

### 4.4 Tenant 配额子路径

`GET/POST .../tenants/{tenant}/quotas` 与 `PATCH/DELETE .../quotas/{pool}`：每个端点映射为"读 Tenant CR → 据 ResourceUnit 折算 → 局部改 `spec.quotas[]` → 写回 CR"的原子封装。GET 时一并从 CR `status.quotas[].used` 实时读用量。

## 5. 关键机制

无异步、无 reconciler。所有 mutation 是单次 K8s API 调用（最多一次乐观锁重试），返回时已写入 etcd。

## 6. 接口契约

| 类别 | 内容 | 引用 |
| --- | --- | --- |
| 对外 REST | `/api/v1/resourcepools[/{pool}[/units[/{unit}]]]`（`ResourcePools` tag）；`/api/v1/tenants[/{tenant}[/quotas[/{pool}]]]`（`Tenants` tag） | [openapi/cluster-manager.yaml](../apis/cluster-manager.yaml) |
| 下发 CR | `ResourcePool` / `Tenant`（`axisml.io/v1alpha1`，cluster-scoped）；cluster-manager 是 `spec` 的 REST 写者，kubectl 路径也允许；Tenant `status` 由 tenant-operator 单写 | [resource-pool-crd.yaml](../../deploy/helm/crds/resource-pool-crd.yaml) / [tenant-crd.yaml](../../deploy/helm/crds/tenant-crd.yaml) |
| 身份头 | 调用方注入 `X-Axisml-User`，本服务仅做 ownership 归属；透传为 CR annotation `axisml.io/last-modified-by`（[auth.md §6](../../../axisml-platform/docs/system_design/auth.md#6-下游身份透传)） | — |
| 错误格式 | HTTP 标准码 + RFC 7807 problem+json；K8s API 错误经 typed 映射 | — |
| 写后语义 | mutation 经 K8s API 写入 etcd 后返回；强一致 | — |

**防御等级**：`metadata.name` / `units[].name` 不可变当前由 admission webhook 兜底；非 cluster-manager 路径（kubectl）写 CR 完全允许，故不预设独占写者。

## 7. 依赖

| 依赖 | 用途 |
| --- | --- |
| Kubernetes API | ResourcePool / Tenant CR CRUD |
| 上游调用方 | 唯一外部调用方；admin 域 UI 入口（资源池 + 租户） |
| tenant-operator | 下游 Tenant CR 消费者，经 etcd watch 落地；非直接调用（[tenant-operator.md](tenant-operator.md)） |

不依赖 PostgreSQL、compute、artifacts；与 tenant-operator 经 etcd（Tenant CR）解耦，无直接调用。

## 8. 运行时形态

| 维度 | 值 |
| --- | --- |
| 进程 | 单二进制 `axisml-cluster-manager`；子命令 `serve` / `bootstrap`（创建默认 pool） |
| 副本 | 任意（无状态对等运行；无 leader election） |
| 暴露端口 | API `:8080`；Metrics `:8081`；Probes `:8082`（`/readyz` 校验 K8s API 可达），均不对外 |
| RBAC scope | ClusterRole：`resourcepools` / `tenants.axisml.io`（`get/list/watch/create/update/patch/delete`）、`events` `create/patch` |
| Helm / 镜像 | 见 [deployment.md](../../../docs/system_design/deployment.md) |

## 9. 相关引用

- [high_level_design.md](../../../docs/system_design/high_level_design.md) — 控制平面拓扑与系统不变量
- [auth.md](../../../axisml-platform/docs/system_design/auth.md) — 身份与鉴权契约
- [deployment.md](../../../docs/system_design/deployment.md)
- [openapi/cluster-manager.yaml](../apis/cluster-manager.yaml) — REST 契约源
- [compute-service.md](compute-service.md) — pool/unit 的展开消费方（Informer 直读）
- [tenant-operator.md](tenant-operator.md) — Tenant CR 的落地消费方
