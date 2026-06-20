# AxisML System 层概要

System 层是 AxisML 的**控制面**：100% 自研领域能力，承接 Platform 的内部调用并把用户意图落地为 Kubernetes 上的真实资源。**不对外暴露**——全部 ClusterIP，仅接受 Platform 调用并信任 `X-Axisml-User` 身份透传。CRDs 随本层一同发布。

## 组成

| 文档 | 一句话职责 | 关键模型 |
| --- | --- | --- |
| [cluster-manager.md](cluster-manager.md) | admin 域 K8s REST 抽象（ResourcePool + Tenant CRUD、配额折算、集群容量 / 指标），让上游不直接调 K8s API | ResourcePool CR（含 `units[]`）+ Tenant CR |
| [compute-service.md](compute-service.md) | 业务域计算服务，以 PG 为权威管理 Run / Service / Workspace / TrafficPolicy，派生三类 CR | MLRun + MLService + MLTrafficPolicy（PG，namespace 分区） |
| [artifact-hub.md](artifact-hub.md) | 业务域制品服务，元数据（PG）/ 存储（zot·RustFS）分离 | Artifact 四元组 `(namespace, kind, name, version)` |
| [tenant-operator.md](tenant-operator.md) | 把 Tenant CR 翻译为 Namespace / ElasticQuota / 初始化资源，回流 `status.used` | Tenant CR（cluster-scoped） |
| [compute-operator.md](compute-operator.md) | dispatcher + handler 把三类 CR 按 `spec.backend.{name,engine}` 路由为 K8s 与网关资源 | 三类 CR + backend handler registry |

前三者是 REST 服务（业务域 / admin 域），后两者是 operator（消费 CR）。

## 层内关键约定

- **PG 为权威、CR 为派生**：compute-service / artifact-hub 以 PostgreSQL 为业务权威，把 CR 当作 PG 行的派生产物下发（compute 用内嵌 Outbox + reconciler 保证强一致）；cluster-manager 例外——它无 PG，直接以 REST 读写 etcd 上的 `Tenant` / `ResourcePool` CR。
- **operator 单向消费 CR**：operator 只读 `spec`、只写 `status`，从不回写上游 PG；operator 之间互不感知。
- **写 / 读路径经 etcd 收敛**：cluster-manager 写 ResourcePool / Tenant CR，compute-service（Informer 展开 pool/unit）与 tenant-operator（落地）直读 CR，组件间无直接调用。
- **配额与调度收编**：所有派生 Pod 强制 `schedulerName: koord-scheduler` + ElasticQuota label，不存在绕过配额的路径。

完整系统级不变量见 [high_level_design.md §2.2](../high_level_design.md#22-关键不变量)。schema 见 [database.md](../database.md)，部署见 [deployment.md](../deployment.md)，基础设施依赖见 [infra/overview.md](../infra/overview.md)。
