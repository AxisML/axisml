# 调度与配额

## 需求

平台需要一个统一调度器同时满足多租户配额与分布式训练，要求：

- **多租户弹性配额**：namespace 级 `min`/`max`，空闲容量可被借用、争用时按 `min` 回收，超 `max` 拒绝调度；
- **Gang Scheduling**：分布式训练的全部 Pod 要么同时调度、要么都不调度，避免资源死锁；
- **与默认调度器共存**：只接管 AxisML 工作负载，控制面 / 基础设施 Pod 仍走 kube-scheduler，零副作用；
- **不锁定私有扩展**：配额 CR 字段尽量与上游 scheduler-plugins 对齐，降低切换成本。

## 技术选型

选用 **[Koordinator](https://koordinator.sh/)**。理由：其 `ElasticQuota`（sigs.k8s.io scheduler-plugins）提供 namespace-scoped `min`/`max` 满足弹性配额，`PodGroup` 提供 Gang Scheduling，二者由统一 koord-scheduler 承载；按 `schedulerName` 与 kube-scheduler 共存；不引入 Koordinator 私有 annotation，CR 字段与上游 scheduler-plugins ElasticQuota 一一对应，避免锁定。

## 组件构成

| 组件 | 职责 | 启用 |
| --- | --- | --- |
| koord-scheduler | 自定义调度器，承载 Gang Scheduling 与 ElasticQuota plugin | 启用 |
| koord-manager | 控制器集合，管理 ElasticQuota / PodGroup 等 CR 状态聚合 | 启用 |
| koord-descheduler / koordlet | Pod 重平衡 / 节点侧 QoS agent | 暂不启用 |

## 核心能力

- **Gang Scheduling**：通过 `PodGroup`（`scheduling.sigs.k8s.io/v1alpha1`）表达，同一 PodGroup 全部 Pod 要么同时调度、要么都不调度。
- **ElasticQuota**：`ElasticQuota`（namespace-scoped）承载 `min`/`max`，Pod 经 label `quota.scheduling.koordinator.sh/name=<eq-name>` 关联；借用容量按 koord-scheduler 默认平权处理。
- **Preemption / Reclaim**：已分配但低于其他 ElasticQuota `min` 的资源可被回收；高于 `max` 的请求一律拒绝调度。**Backfill**：空闲资源回填。

## 协作契约（不点名调用方）

- **Quota 全覆盖（系统级硬不变式）**：任何接入工作负载 Pod 必须设 `schedulerName: koord-scheduler` + `quota.scheduling.koordinator.sh/name` label，不允许绕过 quota 的调度路径。第三方 controller（如 KServe）派生 Pod 时必须透传这两字段，不支持透传的 controller 不应接入。
- **Gang scheduling 仅按需启用**：分布式训练等全员就位的工作负载创建 PodGroup；常驻服务 / 单 Pod 任务不创建 PodGroup，但仍走 koord-scheduler 并经 quota label 计入 ElasticQuota。
- **ElasticQuota / PodGroup CR 由调用方独占 owner**：`min`/`max`、命名、补偿、RBAC 全归调用方；本功能不预置任何 ElasticQuota CR、不持有其 mutation 权限。
- **与 kube-scheduler 共存**：`koord-scheduler` 仅接管设了 `schedulerName: koord-scheduler` 的 Pod；Infra 自身 Pod 不设此字段，走默认 kube-scheduler、不消耗 ElasticQuota。
