# AxisML Infra 概要设计

AxisML Infra 是平台的基础设施层，由一组开源组件组成，为工作负载与控制面服务提供底层支撑。Infra 自身不承载业务逻辑，能力全部由第三方组件提供；AxisML 只负责选型、组装、暴露契约与必要的 glue 资源（Gateway、HTTPRoute、Secret、ConfigMap、ServiceAccount 等）。全部 7 个组件由 Infra 层 chart `axisml-infra` 统一管理（[deployment.md](deployment.md)）。

| 组件 | 选型 | 职责 |
| --- | --- | --- |
| 服务网关 | Envoy Gateway | 请求路由、认证鉴权、流量控制（Gateway API） |
| 对象存储 | RustFS | S3 兼容对象存储 |
| OCI Registry | zot | OCI Distribution v2 制品存储 |
| 数据库 | PostgreSQL（bitnami chart） | 元数据持久化 |
| GPU 管理 | NVIDIA GPU Operator | GPU 驱动、设备插件与监控 |
| 调度与配额 | Koordinator | koord-scheduler 接管工作负载；ElasticQuota 多租户配额；PodGroup gang scheduling |
| 监控 | kube-prometheus-stack | 集群与业务可观测性 |

## 1. 职责与边界

| 组件 | 对外契约 | 不做什么 |
| --- | --- | --- |
| Envoy Gateway | 提供 `Gateway` 与 listener；接受声明式 `HTTPRoute` / `SecurityPolicy` / `BackendTrafficPolicy` | 不感知业务语义，不内置用户态鉴权策略 |
| RustFS | 提供 S3 API；admin 凭证由调用方持有并按需签发 presigned URL | 不内置租户模型 / ACL（隔离由调用方在 bucket / prefix 完成） |
| zot | 提供 OCI Distribution v2；admin 凭证由调用方持有并签 scope-limited bearer token | 不内置租户；repo 路径命名由调用方决定 |
| PostgreSQL | 提供单一 database，调用方按表前缀逻辑隔离 | 不做应用层 SQL 迁移（由调用方自管） |
| GPU Operator | 提供 `nvidia.com/gpu` extended resource、节点标签、DCGM Exporter | 不做调度决策（调度由 koord-scheduler 完成） |
| Koordinator | 提供 `koord-scheduler`、`ElasticQuota` / `PodGroup` CRD | 不持有 ElasticQuota / PodGroup CR 写权限（由各 CR owner 派生） |
| kube-prometheus-stack | 提供 Prometheus / Grafana / AlertManager；自动发现 ServiceMonitor / PodMonitor | 不主动埋点（各组件自行暴露 `/metrics`） |

**面向接入工作负载的硬不变式**：任何接入本基础设施的工作负载 Pod 必须设 `schedulerName: koord-scheduler` 并携带 label `quota.scheduling.koordinator.sh/name=<elastic-quota-name>`，否则视为绕过配额的 bug（§3.6）。

## 2. 调用关系

描述基础设施自身接口，不对接入方点名：

- 外部流量 → **Envoy Gateway**（按 HTTPRoute 转发到 ClusterIP Service）。
- 接入服务 → **PostgreSQL**（元数据读写）；接入服务 / 终端 cli → **RustFS**（S3）/ **zot**（OCI Distribution v2）。
- 任何接入工作负载 Pod → **Koordinator**（`schedulerName: koord-scheduler` + label `quota.scheduling.koordinator.sh/name` 消费 ElasticQuota）；申请 `nvidia.com/gpu` → **GPU Operator** 完成设备分配。
- 已配置采集对象（DCGM Exporter + AxisML ServiceMonitor / PodMonitor）→ **kube-prometheus-stack**（Prometheus Operator 自动发现）。

## 3. 组件设计

### 3.1 服务网关（Envoy Gateway）

基于 Kubernetes [Gateway API](https://gateway-api.sigs.k8s.io/) 以声明式配置路由、认证、流量控制。

```
GatewayClass (envoy-gateway)
  └── Gateway (axisml-gateway, in axisml-infra ns)  Listener: HTTP(80)/HTTPS(443)
        │  allowedRoutes.namespaces: 放行接入工作负载所在 namespace
        └── HTTPRoute（静态 / 派生）→ 目标 ClusterIP Service
```

- **Gateway**：单一 `axisml-gateway` 承载全部路由，由 `axisml-infra` chart 提供。
- **静态 HTTPRoute**：由调用方 chart 一同发布，对接控制面服务对外接口。
- **派生 HTTPRoute**：调用方 controller 在工作负载 namespace 内创建 `HTTPRoute` / `SecurityPolicy` / `BackendTrafficPolicy`，`parentRefs` 指向 `axisml-gateway`；`ReferenceGrant` 仅在跨 namespace `backendRef` 授权场景使用。
- **认证鉴权**（`SecurityPolicy`，可附加到 Gateway / HTTPRoute）：JWT 验证（issuer + JWKS）· OIDC 集成 · ExtAuth · per-Service（`targetRefs`）。具体 IdP 由调用方决定，Infra 只保证能力就位。
- **流量控制**（`BackendTrafficPolicy`）：限流 · 熔断 · 超时 / 重试 · 负载均衡。

### 3.2 对象存储（RustFS）

Apache 2.0、Rust 实现、S3 API 兼容。部署模式：Standalone（单 Pod + PVC）/ Distributed 4×4 / 16×1。调用方通过 S3 SDK 访问；admin 凭证由 `axisml-infra` 自动生成（或预置），presigned URL 与短期凭证由调用方按需签发；bucket / prefix 由调用方组织，本服务不内置 ACL / 租户模型。

### 3.3 OCI Registry（zot）

CNCF Sandbox 的 OCI Distribution v2 兼容仓库：原生支持 OCI artifact manifest（`artifactType` 承载非容器制品如模型权重）· 内容寻址 `<repo>@sha256:<digest>` 不可变引用 · 后端可插拔（filesystem / S3，可把 blob 切到 RustFS）· scope-limited bearer token · `HEAD .../manifests/<ref>` 返 digest 供完整性校验。部署：Standalone（filesystem）/ HA 3×（共享 S3 后端）。Infra 层提供 zot endpoint（ConfigMap）、admin 凭证（平台级 Secret）、公共拉取凭证（`axisml-system` Namespace Secret）；repo 路径命名 / 租户隔离 / scope token 形态由调用方决定。

### 3.4 数据库（PostgreSQL）

元数据持久化，作为第三方依赖归 Infra 层（与 RustFS/zot 同性质），部署在 `axisml-infra`（Service `axisml-database`）。System 层经 FQDN `axisml-database.axisml-infra:5432` 连接，凭据从共享 `database.auth.password` 自渲染为本 namespace Secret。模式：内置（bitnami StatefulSet + PVC）/ 外部（`database.enabled=false` + `externalDatabase.*` 接 RDS）。schema 迁移由各调用方二进制内嵌 `golang-migrate` 在启动时执行（依赖 PG advisory lock 避免并发迁移）。详见 [database.md](database.md)。

### 3.5 GPU 管理（NVIDIA GPU Operator）

自动化 GPU 驱动、设备插件、监控生命周期。组件：Driver Container · Container Toolkit · Device Plugin（向调度器报告 `nvidia.com/gpu`）· DCGM Exporter（GPU 利用率 / 显存 / 温度指标）· GPU Feature Discovery（节点打标）· MIG Manager（A100/H100 分区，按需启用）。调度契约：业务 Pod 用资源名 `nvidia.com/gpu`；节点标签 `nvidia.com/gpu.product` 可做 nodeSelector / affinity；DCGM `/metrics` 由 kube-prometheus-stack 自动采集。

### 3.6 调度与配额（Koordinator）

同时承担统一调度器与多租户配额引擎。**任何接入工作负载 Pod 都强制走 `koord-scheduler` 并消耗对应 ElasticQuota**；只有控制平面 Pod 留在默认 kube-scheduler。组件：koord-scheduler（Gang Scheduling + ElasticQuota plugin，启用）· koord-manager（CR 状态聚合，启用）· koord-descheduler / koordlet（暂不启用）。

核心能力：

- **Gang Scheduling**：通过 sigs.k8s.io scheduler-plugins `PodGroup`（`scheduling.sigs.k8s.io/v1alpha1`）表达，同一 PodGroup 全部 Pod 要么同时调度、要么都不调度。
- **ElasticQuota**：`scheduling.sigs.k8s.io/v1alpha1` `ElasticQuota`（namespace-scoped）承载 `min`/`max`，Pod 经 label `quota.scheduling.koordinator.sh/name=<eq-name>` 关联；不引入 Koordinator 私有 `shared-weight` annotation，借用容量按 koord-scheduler 默认平权处理（CR 字段集与上游 scheduler-plugins 一一对应）。
- **Preemption / Reclaim**：已分配但低于其他 ElasticQuota `min` 的资源可被回收；高于 `max` 的请求一律拒绝调度。**Backfill**：空闲资源回填。

**协作契约**（不点名调用方）：

- **Quota 全覆盖（系统级硬不变式）**：任何接入工作负载 Pod 必须设 `schedulerName: koord-scheduler` + `quota.scheduling.koordinator.sh/name` label，不允许绕过 quota 的调度路径。第三方 controller（如 KServe）派生 Pod 时必须透传这两字段，不支持透传的 controller 不应接入。
- **Gang scheduling 仅按需启用**：分布式训练等全员就位的工作负载创建 PodGroup；常驻服务 / 单 Pod 任务不创建 PodGroup，但仍走 koord-scheduler 并经 quota label 计入 ElasticQuota。
- **ElasticQuota / PodGroup CR 由调用方独占 owner**：`min`/`max`、命名、补偿、RBAC 全归调用方；本基础设施不预置任何 ElasticQuota CR、不持有其 mutation 权限。
- **与 kube-scheduler 共存**：`koord-scheduler` 仅接管设了 `schedulerName: koord-scheduler` 的 Pod；Infra 自身 Pod 不设此字段，走默认 kube-scheduler、不消耗 ElasticQuota。

### 3.7 监控（kube-prometheus-stack）

含 Prometheus、Grafana、AlertManager。各组件接入模型：暴露 `/metrics` + 随 chart 提供 `ServiceMonitor` / `PodMonitor`，Prometheus Operator 自动发现，无需手动维护 `prometheus.yml`。指标体系层级：

| 层级 | 来源 | 典型指标 |
| --- | --- | --- |
| 集群层 | kube-state-metrics、node-exporter | 节点 CPU/内存/磁盘、Pod 状态 |
| GPU 层 | DCGM Exporter（GPU Operator） | GPU 利用率、显存、温度、功耗 |
| 网关层 | Envoy Gateway | 请求量、延迟分位、错误率 |
| 调度层 | koord-scheduler / koord-manager | ElasticQuota 用量与借用、PodGroup 调度状态、调度延迟 |
| 业务层 | 接入服务 | 各服务自行暴露 |

## 4. 部署形态与设计决策

部署细节（chart 组织、命名空间、安装顺序、依赖清单、fullnameOverride）见 [deployment.md](deployment.md)。选型理由：

| 决策项 | 决策 / 理由 |
| --- | --- |
| 服务网关 | Envoy Gateway——Gateway API 是官方 Ingress 继任者，原生 gRPC/HTTP2，配置全 CRD 化 |
| 对象存储 | RustFS——Apache 2.0、S3 兼容，规避 MinIO 转 AGPLv3 的传染风险，S3 抽象使切换成本有限 |
| OCI Registry | zot——OCI Distribution v2 + 1.1 artifact manifest 原生支持，CNCF Sandbox、单二进制、可选 S3 后端 |
| 制品分流 | OCI（zot）走不可变内容寻址制品，S3（RustFS）走目录型 / 多文件制品 |
| 数据库 | bitnami/postgresql 子 chart，`externalDatabase` 段保留用于生产外接 RDS；纳入 Infra 层 |
| GPU 管理 | NVIDIA GPU Operator——K8s 原生 GPU 管理事实标准，DCGM 与监控栈天然集成 |
| 调度与配额 | Koordinator——scheduler-plugins ElasticQuota 提供 namespace-scoped `min`/`max`，PodGroup 提供 Gang Scheduling，统一 koord-scheduler 承载，与 kube-scheduler 按 `schedulerName` 共存 |
| 配额 CR 表达 | 不引入 Koordinator 私有 annotation，保持与上游 scheduler-plugins ElasticQuota 字段一一对应（降低锁定 / 切换成本） |

部署架构图与三层 chart 拆分见 [overview.md §4](overview.md#4-整体架构) / [deployment.md](deployment.md)。
