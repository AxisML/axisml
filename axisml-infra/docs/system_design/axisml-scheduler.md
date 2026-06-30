# axisml-scheduler 设计

## 1. 概述

`axisml-scheduler` 是 AxisML 自研的批调度器，承载平台全部工作负载 Pod 的调度，提供三项能力：**Binpack**（装箱，优先填满节点以提升加速器利用率）、**Gang Scheduling**（分布式训练全员就位或都不调度）、**ElasticQuota**（多租户弹性配额，namespace 级 `min`/`max`、空闲可借、争用按 `min` 回收）。

它基于 Kubernetes **scheduler-framework**（`k8s.io/kubernetes/cmd/kube-scheduler/app`）编译，复用上游 [scheduler-plugins](https://github.com/kubernetes-sigs/scheduler-plugins) 的 Coscheduling 与 in-tree NodeResourcesFit，仅在「label 绑定的多配额」这一上游不支持的点上自研一个薄 plugin。它通过 `schedulerName` 与 kube-scheduler 共存，只接管显式选用它的 Pod，对控制面 / 基础设施 Pod 零副作用。

## 2. 设计目标与约束

- **多租户弹性配额**：namespace 级 `min`/`max`；空闲容量可被其他配额借用，争用时按 `min` 回收，超 `max` 一律拒绝调度。
- **一个 namespace 多个配额**：一个 K8s Namespace（= tenant）下可挂多个 ElasticQuota（每 pool 一个），Pod 经 label 关联到具体配额——这是上游 scheduler-plugins（按 namespace 关联，一 namespace 一配额）不满足的点，也是自研的唯一动因。
- **Gang Scheduling**：分布式训练等「全员就位」工作负载的全部 Pod 要么同时调度、要么都不调度，避免部分占位导致的资源死锁。
- **Binpack**：在满足约束的前提下优先填满已用节点，减少加速器碎片。
- **与默认调度器共存**：只接管设了 `schedulerName: axisml-scheduler` 的 Pod；其余走 kube-scheduler、零副作用、不消耗配额。
- **对齐上游生态**：CR 词汇表与上游 scheduler-plugins 对齐（`scheduling.x-k8s.io/v1alpha1` 的 `ElasticQuota` / `PodGroup`），不引入私有 annotation，便于跟随上游演进。

非目标：节点侧 QoS / 资源超卖 / 干扰隔离（node agent 类能力）、Pod 重平衡（re-scheduling 类能力）、拓扑 / NUMA 感知调度——当前不纳入，列入未来工作。

## 3. 组件构成

`axisml-scheduler` 部署为两个二进制：

| 组件 | 职责 | 副本 |
| --- | --- | --- |
| `axisml-scheduler`（scheduler） | scheduler-framework 二进制，注册 Coscheduling + NodeResourcesFit + ElasticScheduling 三个 plugin，承载实际调度决策 | 启用（HA 多副本经 leader election，单机 / dev 降为 1） |
| `axisml-scheduler-controller`（controller） | 维护 `PodGroup.status` 与 `ElasticQuota.status.used`：按 label 聚合各配额的实时用量，供调用方回读 | 启用（同上） |

controller 不可省：它负责把「已调度 Pod 的用量」按 quota label 聚合写回 `ElasticQuota.status.used`，调用方据此回读配额用量。上游 scheduler-plugins 的 ElasticQuota controller 按 namespace 聚合，与本调度器的 label 绑定不一致，故 controller 一并自研（fork 上游 controller 改聚合维度）。

`status.used` 与调度器**最终一致**,而非强一致:scheduler 进程内的预占缓存(Reserve + Pod informer)是准入判定的权威来源,controller 独立地从已调度 Pod 聚合 `status.used` 供外部展示。二者在瞬态(Reserve 与 informer 同步之间、bind 失败回滚窗口)可短暂不同,随后收敛——`status.used` 是展示值,不被任何组件用于准入。

## 4. 调度能力

三项能力的实现成本与来源差异显著，分别说明。

### 4.1 Binpack（零代码，纯配置）

复用 in-tree `NodeResourcesFit` 评分插件，`scoringStrategy` 设为 `MostAllocated`（或 `RequestedToCapacityRatio`），并对加速器资源加权，使调度器优先把 Pod 放到已用率高的节点，填满后再开新节点。纯 `KubeSchedulerConfiguration` 配置，无自研代码。

### 4.2 Gang Scheduling（原样复用上游）

引入上游 scheduler-plugins 的 `Coscheduling` plugin（PreFilter / Filter / Permit / Reserve），读取 `PodGroup` CR 的 `spec.minMember`，在 Permit 阶段等待同一 PodGroup 的全部成员就位后统一放行，否则一并退回，实现 all-or-nothing。逻辑不改，仅引依赖。

**按需启用**：分布式训练等全员就位的工作负载创建 `PodGroup`；常驻服务 / 单 Pod 任务不创建 PodGroup，但仍走 `axisml-scheduler` 并经 quota label 计入 ElasticQuota。

### 4.3 ElasticQuota（自研薄 plugin）

唯一自研的调度逻辑，命名 `ElasticScheduling`，由上游 scheduler-plugins 的 `CapacityScheduling` plugin fork 而来，**唯一改动是把「Pod → Quota」的关联从 namespace 改为 label**：

- **绑定**：Pod 经 label `scheduling.axisml.io/quota=<eq-name>` 关联到同 namespace 下某个 `ElasticQuota`；plugin 按该 label（而非 namespace）汇总用量。
- **PreFilter（已实现）**：累加同一配额已用 + 本 Pod 请求,超出 `spec.max` 拒绝调度。**fail-closed**：缺 quota label 或对应 `ElasticQuota` 不存在时,Pod 置 `Unschedulable`(不放行未计量 Pod),待 label 补齐 / `ElasticQuota` 创建后由 `EventsToRegister` 重新入队。
- **Reserve / Unreserve（已实现）**：在 Reserve 阶段登记预占、调度失败时回滚,保证并发调度下用量账本一致。
- **借用与回收（未实现,见 [§10 未来工作](#10-未来工作)）**：`spec.min` 当前不参与调度决策;空闲容量按 `max` 平权使用,尚无基于 `min` 的借用回收与抢占。

当前实现聚焦 `spec.max` 强制(系统硬不变式),量级在数百行,不重写调度器。

## 5. CR 词汇表

调度器读两类 CR，均为上游 scheduler-plugins 原生类型，spec 字段不做私有扩展：

| 对象 | API Group / Version / Kind | 关键字段 | 写者 |
| --- | --- | --- | --- |
| 弹性配额 | `scheduling.x-k8s.io/v1alpha1` · `ElasticQuota`（namespace-scoped） | `spec.min` / `spec.max`（调用方写）；`status.used`（controller 写） | 调用方拥有 spec；controller 拥有 status |
| 调度组 | `scheduling.x-k8s.io/v1alpha1` · `PodGroup`（namespace-scoped） | `spec.minMember`（调用方写）；`status`（controller 写） | 同上 |

Pod 侧两个约定字段：

| 字段 | 取值 | 含义 |
| --- | --- | --- |
| `spec.schedulerName` | `axisml-scheduler` | 选用本调度器；不设则走 kube-scheduler |
| `metadata.labels["scheduling.axisml.io/quota"]` | `<eq-name>` | 关联到同 namespace 下的具体 ElasticQuota |
| `metadata.labels["scheduling.axisml.io/pod-group"]` | `<pg-name>` | （gang 工作负载）关联到 PodGroup；单 Pod 任务不设 |

## 6. 调度器配置（KubeSchedulerConfiguration）

```yaml
apiVersion: kubescheduler.config.k8s.io/v1
kind: KubeSchedulerConfiguration
leaderElection:
  leaderElect: true
profiles:
  - schedulerName: axisml-scheduler
    plugins:
      preFilter: { enabled: [{ name: Coscheduling }, { name: ElasticScheduling }] }
      filter:    { enabled: [{ name: Coscheduling }] }
      score:     { enabled: [{ name: NodeResourcesFit, weight: 5 }] }
      reserve:   { enabled: [{ name: Coscheduling }, { name: ElasticScheduling }] }
      permit:    { enabled: [{ name: Coscheduling }] }
    pluginConfig:
      - name: NodeResourcesFit
        args:
          scoringStrategy:
            type: MostAllocated          # binpack：优先填满已用节点
            resources:
              - { name: cpu, weight: 1 }
              - { name: memory, weight: 1 }
              - { name: nvidia.com/gpu, weight: 10 }
```

## 7. 协作契约（不点名调用方）

- **配额全覆盖（系统级硬不变式）**：任何接入工作负载 Pod 必须设 `schedulerName: axisml-scheduler` + `scheduling.axisml.io/quota` label，不存在绕过配额的调度路径。第三方 controller（如 KServe）派生 Pod 时必须透传这两字段，不支持透传的 controller 不应接入。
- **Gang 仅按需启用**：仅分布式训练等全员就位工作负载创建 `PodGroup` 并打 pod-group label；常驻服务 / 单 Pod 任务不创建 PodGroup，但仍走本调度器并计入配额。
- **CR 由调用方独占 owner**：`ElasticQuota` / `PodGroup` 的 spec（`min`/`max`、`minMember`、命名、补偿、RBAC）全归调用方；本组件不预置任何此类 CR、不持有其 spec 的 mutation 权限，只写它们的 `status`。
- **与 kube-scheduler 共存**：仅接管设了 `schedulerName: axisml-scheduler` 的 Pod；Infra 自身 Pod 不设此字段，走默认 kube-scheduler、不消耗配额。

## 8. 部署形态

`axisml-scheduler` 是 Infra 层组件，随 `axisml-infra` Helm chart 部署到 `axisml-infra` namespace，承载集群的调度与配额职责。

- **代码与镜像**：Go module 位于 `axisml-infra/axisml-scheduler/`（`cmd/scheduler`、`cmd/controller`、`internal/plugins/elasticscheduling`、`internal/controllers`）；构建与镜像 tag 经 `IMAGE_TAG` 与其余 AxisML 组件对齐。
- **CRD**：`ElasticQuota` / `PodGroup` 两份 CRD 随本 chart 发布；因 Helm 仅在首装时安装 `crds/`，schema 升级经 chart 安装流程中独立的 `kubectl apply crds/` 步骤补齐。
- **RBAC**：scheduler 需读 Pod / Node / 两类 CR、写其 status；controller 需读 Pod 与两类 CR、写其 status。

## 9. 监控

scheduler 与 controller 各在 `:8081` 暴露 `/metrics`（Prometheus 格式），随 chart 提供 `PodMonitor` / `ServiceMonitor`。关注指标：ElasticQuota 各配额的 `used` / 借用量、超 `max` 拒绝次数、PodGroup gang 等待时长与 Pending 时长、调度延迟分位、binpack 后的节点装箱率。

## 10. 未来工作

- **基于 `min` 的借用回收与抢占**：让空闲容量可被借用、争用时按各配额 `min` 抢占回收（当前仅强制 `max`，`min` 不参与调度）。
- 节点侧 QoS / 资源超卖 / 干扰隔离（node agent 类能力）。
- Pod 重平衡 / 抢占优化（re-scheduling 类能力）。
- 拓扑 / NUMA / NVLink 感知调度，加速器亲和编排。
- 配额借用的优先级与公平性策略（DRF 等）。
