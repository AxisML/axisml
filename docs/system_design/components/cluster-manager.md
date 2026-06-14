# AxisML Cluster Manager 概要设计

## 1. 定位与边界

集群级 admin REST 抽象层：把 admin 视角的 K8s 写 / 读操作收敛成简单 REST，让 [Platform](platform.md) 全程不直接调 Kubernetes API。承载 `ResourcePool` CRD 的 CRUD（含内嵌的 `units[]` 数组）；admin 域的 K8s 操作（节点 / StorageClass / 集群容量聚合 / 共享卷管理等）统一收纳在本服务内，不外溢到其它组件。

| 做 | 不做 |
| --- | --- |
| ResourcePool CRD CRUD（pool + 内嵌 units 数组） | 修改 Node label / taint（admin 手工维护） |
| 默认 `default` pool 初始化（Helm post-install） | 租户 / 配额管理 (→ [compute-service.md](compute-service.md)) |
| 用户认证与角色鉴权 (← Platform) | 计算负载 / 工作区 PVC / Pod 日志（这些属 workload 域，归 compute） |
| `?labelSelector=` 列表过滤 | 制品管理 (→ [artifact-hub.md](artifact-hub.md)) |
| 集群容量聚合 + 集群级指标代理（节点 allocatable/used、集群时序） | 工作负载 / 租户级指标 (→ [compute-service.md](compute-service.md)) |

形态：**REST 网关层**，无独立持久化（CRD 落 K8s etcd），无 reconciler，无 leader election，多副本对等运行。

## 2. 架构

### 2.1 上下文

```
        ┌──────────────┐  REST + X-Axisml-User   ┌──────────────────┐
        │  Platform    │ ───────────────────────▶│ Cluster Manager  │
        └──────────────┘                          └──────┬───────────┘
                                                         │ K8s API (CR CRUD)
                                                         ▼
                                                ┌─────────────────┐
                                                │ Kubernetes etcd │
                                                │ ResourcePool CR │
                                                └────────▲────────┘
                                                         │ Informer watch
                                                         │
                                                ┌────────┴────────┐
                                                │ compute-service │
                                                │ (展开消费方)    │
                                                └─────────────────┘
```

Platform 调本服务做 pool/unit CRUD。compute 不调本服务——而是通过 K8s Informer 直接 watch ResourcePool CR cache，在创建 Job / Service 时按 `(poolName, unitName)` 展开为 `nodeSelector` / `tolerations` / `requests` / `limits` 并 snapshot 到 PG（详见 [compute-service.md §5.4](compute-service.md#54-resourcepool-展开)）。两条路径都通过 etcd 收敛，cluster-manager 与 compute 之间无直接调用。

### 2.2 内部结构

```
┌──────── Cluster Manager (Go) ────────┐
│  HTTP API (Gin)                       │
│    ├── ResourcePool CRUD              │
│    │    (内部 GET/PATCH 整体 CR)      │
│    └── units 子路径 CRUD              │
│         (内部 patch spec.units[])     │
│                                       │
│  K8s client (clientgoscheme + axisml) │
│  + 可选 Pool Informer (list 缓存)     │
└──────────────────────────────────────┘
```

无 reconciler / 无 worker goroutine / 无 leader election；ResourcePool Informer 可选（仅用作 list 端点加速 cache，不参与一致性）。

## 3. 核心模型

| 实体 | 含义 | 标识键 | 备注 |
| --- | --- | --- | --- |
| ResourcePool | 节点切分维度（GPU 池 / CPU 池 / 训练池）+ 内嵌 `units[]` | `metadata.name`（集群内全局唯一，cluster-scoped CR） | admin 给目标节点打 label/taint；CRD 字段级 schema 见 [resource-pool-crd.yaml](../../../deploy/helm/axisml-system/crds/resource-pool-crd.yaml) |

> 没有独立的 ResourceUnit CR——`units` 是 `ResourcePool.spec.units[]` 的数组项，与 pool 同生共死、原子编辑。

### 3.1 ResourcePool 形状

```yaml
apiVersion: axisml.io/v1alpha1
kind: ResourcePool
metadata:
  name: gpu-a100                          # DNS-1123, 集群内全局唯一
  labels:
    axisml.io/accelerator: a100
  annotations:
    axisml.io/description: "A100 80GB single-node pool"
spec:
  nodeSelector:                           # 池级 nodeSelector（Pool 优先）
    axisml.io/pool: gpu-a100
  tolerations:                            # 池级 tolerations，直接作为 spec.scheduling.tolerations
    - key: nvidia.com/gpu
      operator: Exists
      effect: NoSchedule
  units:                                  # 数组内 name 唯一；同 pool 一起删
    - name: a100-1x-large
      requests:
        cpu: "16"
        memory: 128Gi
        nvidia.com/gpu: "1"
      limits:
        cpu: "16"
        memory: 128Gi
        nvidia.com/gpu: "1"
      nodeSelector: {}                    # 可选；仅贡献 pool 未声明的 key
      annotations:
        axisml.io/description: "1×A100 single-node, 16 CPU"
    - name: a100-2x-large
      requests: { cpu: "32", memory: 256Gi, nvidia.com/gpu: "2" }
      limits:   { cpu: "32", memory: 256Gi, nvidia.com/gpu: "2" }
```

**字段不变性**：

| 字段 | 写入方 | 可变? |
| --- | --- | --- |
| `metadata.name` | Platform 调本服务 | 否 (CR 名即标识) |
| `spec.nodeSelector` / `spec.tolerations` | 同上 | 是 |
| `spec.units[i].name` | 同上 | 否 (标识锚点) |
| `spec.units[i].requests` / `limits` / `nodeSelector` / `annotations` | 同上 | 是 |

**unit 命名约定**：`<accelerator>[-<count>x]-<tier>[-<variant>]`，例如 `cpu-small` / `a100-1x-large` / `h100-8x-xlarge-ib` / `tpu-v4-4x-large`。`<tier>` ∈ `small | medium | large | xlarge`；`cpu` 类型 `<count>x` 段省略。

**默认池**：Helm post-install Job 通过本服务 REST 创建 `default` pool（带 `cpu-small` / `cpu-medium` 两个 unit），`spec.nodeSelector` 空表示整集群可用。

### 3.2 展开合并规则

由 [compute-service §5.4](compute-service.md#54-resourcepool-展开) 在创建 Job/Service 时完成（不由 cluster-manager 或 Platform 完成）：

| 来源 | 合并行为 |
| --- | --- |
| `pool.spec.nodeSelector` | key 全部保留 |
| `unit.nodeSelector` | 仅贡献 pool 未声明的 key |
| `pool.spec.tolerations` | 直接作为 `spec.scheduling.tolerations` |
| `unit.requests` / `limits` | 写入 `spec.roles[*].template.resources` |

snapshot 语义：compute 在 Create 入口完成展开后立刻把 nodeSelector / tolerations / requests / limits 写入 `mlruns.spec` / `mlservices.spec` jsonb。pool 或 unit 后续修改 / 删除**不影响**已创建的 workload——它们存的是展开后的原语，跟 CR 解耦。

## 4. 核心功能

### 4.1 ResourcePool CRUD

| 操作 | 内部行为 |
| --- | --- |
| 创建（POST `/api/v1/resource-pools`） | K8s create ResourcePool CR；admission webhook 校验 name 唯一性、units 数组内 name 唯一 |
| GET / LIST | 直接读 K8s API（或 Informer cache）；list 支持 `?labelSelector=` （K8s 原生 selector grammar） |
| PATCH（pool 级字段） | K8s strategic merge patch；`metadata.name` 不可变（admission 拒绝） |
| DELETE | K8s delete ResourcePool CR；units 跟随一起删（无独立 ResourceUnit 资源） |

### 4.2 Unit 子路径 CRUD（UI 友好的便利端点）

| 操作 | 内部行为 |
| --- | --- |
| GET `/api/v1/resource-pools/{pool}/units` | 返回 `pool.spec.units[]`；不另起 K8s 调用 |
| POST `/api/v1/resource-pools/{pool}/units` | 先 GET pool → append `units[]` → JSON Patch（带 resourceVersion 乐观锁）→ 重试一次防冲突 |
| PATCH `/api/v1/resource-pools/{pool}/units/{name}` | 同上，定位数组项后局部更新 |
| DELETE `/api/v1/resource-pools/{pool}/units/{name}` | 同上，移除数组项 |

每个 unit 端点都映射为"读 pool CR → 局部改 `spec.units[]` → 写回 CR"的原子封装。UI / API 客户端不需要关心 CR 整体形状。

> unit 修改后已创建的 Job/Service **不感知**——见 §3.2 snapshot 语义。

### 4.3 集群容量与指标

admin 域的集群事实由本服务即时聚合，供 [Platform](platform.md#47-dashboard-编排) Dashboard 全局视图使用：

| 端点 | 内部行为 |
| --- | --- |
| `GET /api/v1/cluster/capacity` | K8s typed client 聚合 Node `status.allocatable` 得 GPU / CPU / 内存总量，扣减已调度 Pod requests 得已用量，返回 `{gpu,cpu,memory}{used,total}` |
| `GET /api/v1/cluster/metrics?metric=&range=&step=` | 按 `metric` 选 PromQL 模板查 Prometheus，返回集群级 `MetricSeries`（如集群 GPU 利用率、活跃任务并发） |

容量与时序均为即时聚合，不引入持久化；PromQL 模板见 [monitoring.md §6](../monitoring.md#6-业务指标查询prometheus-代理)。

## 5. 关键机制

无异步、无 reconciler。所有 mutation 是单次 K8s API 调用（最多一次乐观锁重试），返回时已写入 etcd。

## 6. 接口契约

| 类别 | 内容 | 引用 |
| --- | --- | --- |
| 对外 REST | `/api/v1/resource-pools[/{pool}]`、`/api/v1/resource-pools/{pool}/units[/{unit}]` | [apis/cluster-manager.yaml](../apis/cluster-manager.yaml) `ResourcePools` tag |
| 对外 REST（集群事实） | `/api/v1/cluster/capacity`、`/api/v1/cluster/metrics` | [apis/cluster-manager.yaml](../apis/cluster-manager.yaml) `Cluster` tag |
| 下发 CR | `ResourcePool`（`axisml.io/v1alpha1`，cluster-scoped）；cluster-manager 是 REST 写者，kubectl 路径也允许 | [resource-pool-crd.yaml](../../../deploy/helm/axisml-system/crds/resource-pool-crd.yaml) |
| 身份头 | 调用方注入 `X-Axisml-User`，本服务仅做审计；同时透传为 CR annotation `axisml.io/last-modified-by` | [auth.md §7](../auth.md#7-下游身份透传) |
| 错误格式 | HTTP 标准状态码 + RFC 7807 problem+json；K8s API 错误经 typed 映射 | — |
| 写后语义 | mutation 经 K8s API 写入 etcd 后返回；强一致 | — |

**防御等级**：`metadata.name` / `spec.units[].name` 不可变约束当前由 admission webhook 兜底；非 cluster-manager 路径（kubectl）写 CR 完全允许，因此 cluster-manager 不预设独占写者。

## 7. 依赖

| 依赖 | 用途 | 引用 |
| --- | --- | --- |
| Kubernetes API | ResourcePool CR CRUD + Node `allocatable` / Pod requests 读（容量聚合） | — |
| Prometheus | 集群级时序查询（`/cluster/metrics`）；只读 | [infra.md](../infra.md) / [monitoring.md](../monitoring.md) |
| Platform | 唯一外部调用方；admin 域 UI 入口 + Dashboard 集群容量 / 时序 | [platform.md §4.6](platform.md#46-资源池编排) |

不依赖 PostgreSQL、tenant-operator、compute、artifacts。

## 8. 运行时形态

| 维度 | 值 |
| --- | --- |
| 进程 | 单二进制 `axisml-cluster-manager`；子命令 `serve` / `bootstrap`（创建默认 pool CR） |
| 副本 | 任意（无状态对等运行；无 leader election） |
| 暴露端口 | API `:8080`；Metrics `:8081`；Probes `:8082`（`/healthz` / `/readyz`，仅校验 K8s API 可达），均不对外 |
| RBAC scope | ClusterRole：`resourcepools.axisml.io` (`get/list/watch/create/update/patch/delete`)、`nodes` / `pods` `get/list`（容量聚合）、`events` `create/patch` |
| Helm values / 镜像 | 详见 [deployment.md](../deployment.md) |

## 9. 相关引用

- [overview.md](../overview.md) — 控制平面拓扑
- [auth.md](../auth.md) — 身份与鉴权契约
- [deployment.md](../deployment.md) — Helm / 部署
- [monitoring.md](../monitoring.md) — Metrics
- [apis/cluster-manager.yaml](../apis/cluster-manager.yaml) — REST 契约源
- [platform.md](platform.md) — Platform 调本服务做 pool/unit admin UI
- [compute-service.md](compute-service.md) — pool/unit 的展开消费方（通过 K8s Informer 直读 CR）
