# AxisML Infra 层设计

Infra 层是平台的基础设施底座，为工作负载与控制面服务提供底层支撑。**设计单位是平台需要的“功能 / 能力”**——每个功能选用一个成熟的开源技术组件来实现，组件是可替换的实现细节而非设计本身。Infra 自身不承载业务逻辑，AxisML 只负责定义功能契约、选型组装并补必要的 glue 资源（Gateway、HTTPRoute、Secret、ConfigMap、ServiceAccount 等）。全部由 Infra 层 chart `axisml-infra` 统一管理（[deployment.md](../../../docs/deployment.md)）。

| 功能（能力） | 实现技术 | 章节 |
| --- | --- | --- |
| 服务网关 | Envoy Gateway | [§3](#3-服务网关) |
| 存储（对象存储 / OCI Registry / 数据库） | RustFS · zot · PostgreSQL | [§4](#4-存储) |
| 缓存 | Redis | [§4.4](#44-缓存) |
| 加速器管理 | NVIDIA GPU Operator | [§5](#5-加速器管理) |
| 调度与配额 | axisml-scheduler（自研，基于 scheduler-plugins） | [§6](#6-调度与配额) |
| 监控 | kube-prometheus-stack | [§7](#7-监控) |

## 1. 功能职责与边界

| 功能（实现技术） | 对外契约 | 不做什么 |
| --- | --- | --- |
| 服务网关（Envoy Gateway） | 提供 `Gateway` 与 listener；接受声明式 `HTTPRoute` / `SecurityPolicy` / `BackendTrafficPolicy` | 不感知业务语义，不内置用户态鉴权策略 |
| 对象存储（RustFS） | 提供 S3 API；admin 凭证由调用方持有并按需签发 presigned URL | 不内置租户模型 / ACL（隔离由调用方在 bucket / prefix 完成） |
| OCI Registry（zot） | 提供 OCI Distribution v2；admin 凭证由调用方持有并签 scope-limited bearer token | 不内置租户；repo 路径命名由调用方决定 |
| 数据库（PostgreSQL） | 提供单一 database，调用方按表前缀逻辑隔离 | 不做应用层 SQL 迁移（由调用方自管） |
| 缓存（Redis） | 提供单一 key/value 缓存实例，调用方按 key 前缀逻辑隔离 | 不承载业务真相（仅缓存可重建数据，宕机即回退源库） |
| 加速器管理（GPU Operator） | 提供 `nvidia.com/gpu` extended resource、节点标签、DCGM Exporter | 不做调度决策（调度由 axisml-scheduler 完成） |
| 调度与配额（axisml-scheduler） | 提供 `axisml-scheduler`、`ElasticQuota` / `PodGroup` CRD（上游 scheduler-plugins 词汇表） | 不持有 ElasticQuota / PodGroup CR spec 写权限（由各 CR owner 派生；本组件只写其 status） |
| 监控（kube-prometheus-stack） | 提供 Prometheus / Grafana / AlertManager；自动发现 ServiceMonitor / PodMonitor | 不主动埋点（各组件自行暴露 `/metrics`） |

**面向接入工作负载的硬不变式**：任何接入本基础设施的工作负载 Pod 必须设 `schedulerName: axisml-scheduler` 并携带 label `scheduling.axisml.io/quota=<elastic-quota-name>`，否则视为绕过配额的 bug（详见 [§6 调度与配额](#6-调度与配额)）。

## 2. 调用关系

描述各功能对外接口，不对接入方点名：

- 外部流量 → **服务网关**（按 HTTPRoute 转发到 ClusterIP Service）。
- 接入服务 → **数据库**（元数据读写）；接入服务 / 终端 cli → **对象存储**（S3）/ **OCI Registry**（OCI Distribution v2）。
- 接入服务 → **缓存**（可选加速热点读；缓存不可达时回退源库，故为可选依赖）。
- 任何接入工作负载 Pod → **调度与配额**（`schedulerName: axisml-scheduler` + label `scheduling.axisml.io/quota` 消费 ElasticQuota）；申请 `nvidia.com/gpu` → **加速器管理** 完成设备分配。
- 已配置采集对象（DCGM Exporter + AxisML ServiceMonitor / PodMonitor）→ **监控**（Prometheus Operator 自动发现）。

## 3. 服务网关

平台需要一个统一的南北向流量入口，集中承担：把外部请求路由到控制面（Platform）与数据面（工作区 / 在线服务）的目标 Service；在入口层完成数据面 JWT 鉴权；提供限流、超时、熔断、重试等横切流量治理；**声明式、CRD 化**配置，可被各组件 controller 在工作负载 namespace 内动态派生路由；原生支持 gRPC / HTTP2（推理服务常用）。

**技术选型**：**[Envoy Gateway](https://gateway.envoyproxy.io/)**（基于 Kubernetes [Gateway API](https://gateway-api.sigs.k8s.io/)）。Gateway API 是官方 Ingress 继任者，配置完全 CRD 化、支持跨 namespace 派生路由；Envoy 数据面原生 gRPC / HTTP2；`SecurityPolicy` / `BackendTrafficPolicy` 扩展分别覆盖鉴权与流量治理；零外部依赖。

**资源模型**：

```
GatewayClass (envoy-gateway)
  └── Gateway (axisml-gateway, in axisml-infra ns)  Listener: HTTP(80)
        │  allowedRoutes.namespaces: 放行接入工作负载所在 namespace
        └── HTTPRoute（静态 / 派生）→ 目标 ClusterIP Service
```

- **Gateway**：单一 `axisml-gateway` 承载全部路由，由 `axisml-infra` chart 提供。
- **静态 HTTPRoute**：由调用方 chart 一同发布，对接控制面服务对外接口。
- **派生 HTTPRoute**：调用方 controller 在工作负载 namespace 内创建 `HTTPRoute` / `SecurityPolicy` / `BackendTrafficPolicy`，`parentRefs` 指向 `axisml-gateway`；`ReferenceGrant` 仅在跨 namespace `backendRef` 授权场景使用。

> **Future work**：当前仅配置 HTTP(80) listener；TLS 终止（HTTPS/443 listener + 证书引用）待后续补充。

**对外契约**：

| 能力 | 资源 | 说明 |
| --- | --- | --- |
| 认证鉴权 | `SecurityPolicy`（附加到 Gateway / HTTPRoute） | JWT 验证（issuer + JWKS）· OIDC 集成 · ExtAuth · per-Service（`targetRefs`）。具体 IdP 由调用方决定，本功能只保证能力就位 |
| 流量控制 | `BackendTrafficPolicy` | 限流 · 熔断 · 超时 / 重试 · 负载均衡 |

本功能只提供 `Gateway` 与 listener 能力，不感知业务语义、不内置用户态鉴权策略。部署形态见 [deployment.md §8.1](../../../docs/deployment.md#81-控制面-deployment)。

## 4. 存储

平台有三类持久化需求，分别选用不同技术实现：目录型大文件（对象存储）、不可变内容寻址制品（OCI Registry）、控制面关系元数据（数据库）；外加一类**非持久**的状态缓存（缓存）。四者均归 Infra 层、对调用方暴露标准协议、不内置租户模型。

### 4.1 对象存储

**需求**：承载数据集等**目录型 / 多文件**制品的 bytes，要求：标准 S3 协议（生态通用、可换实现）；客户端 / workload 凭短期凭证直连读写、不经业务服务代理大文件；按 bucket / prefix 自管命名隔离。

**技术选型**：**[RustFS](https://rustfs.dev/)**（Apache 2.0、Rust 实现、S3 API 兼容）。S3 兼容满足协议要求且切换成本有限；Apache 2.0 许可规避 MinIO 自 2021 转 AGPLv3 的商用传染风险。

**对外契约**：

- 调用方通过 S3 SDK 访问，对具体实现无感知；admin 凭证由 `axisml-infra` 自动生成（或预置），presigned URL 与短期凭证由调用方按需签发。
- 命名隔离：bucket / prefix 由调用方组织，本功能不内置 ACL 或租户模型。
- 部署模式：Standalone（单 Pod + PVC）/ Distributed 4×4 / 16×1。

### 4.2 OCI Registry

**需求**：承载模型权重与容器镜像等**不可变内容寻址**制品，要求：内容寻址（`@digest`）保证可复现引用；支持非容器制品（模型权重）；scope 限定的拉取凭证；manifest 完整性校验；后端存储可插拔。

**技术选型**：**[zot](https://zotregistry.dev/)**（CNCF Sandbox、单二进制 Go 实现）。原生支持 OCI Distribution v2 + 1.1 artifact manifest，对非容器制品的 `artifactType` 语义完整；后端可插拔（filesystem / S3，可把 blob 切到 RustFS）。

**对外契约**：

| 能力 | 说明 |
| --- | --- |
| artifact manifest | 原生支持 `application/vnd.oci.image.manifest.v1+json` + `artifactType`，承载 ML 模型权重等非容器制品 |
| 内容寻址 | `<repo>@sha256:<digest>` 不可变引用 |
| Bearer token 鉴权 | scope-limited（`repository:<repo>:push`/`pull`） |
| Manifest 校验 | `HEAD /v2/<repo>/manifests/<ref>` 返回 digest，调用方据此做完整性校验 |

Infra 层提供 zot endpoint（ConfigMap）、admin 凭证（平台级 Secret）、公共拉取凭证（`axisml-tenant` Namespace Secret，由 `default` Tenant 管理）；repo 路径命名 / 租户隔离 / scope token 形态由调用方决定。部署：Standalone（filesystem）/ HA 3×（共享 S3 后端）。

### 4.3 数据库

**需求**：控制面元数据的统一关系存储，要求：事务与关系约束；多服务共用一库、按 schema / 表前缀逻辑隔离；支持生产外接托管实例（RDS）。

**技术选型**：**PostgreSQL**（bitnami/postgresql 子 chart）。成熟关系库满足事务与生态要求；复用现成 chart 免自写 StatefulSet 模板；`externalDatabase` 段保留用于生产外接 RDS。作为第三方依赖归 Infra 层（与对象存储 / OCI Registry 同性质）。

**对外契约**：

- 部署在 `axisml-infra`（Service `axisml-database`）。System 层经跨 namespace FQDN `axisml-database.axisml-infra:5432` 连接，凭据从共享 `database.auth.password` 自渲染为本 namespace Secret。
- 模式：内置（StatefulSet + PVC）/ 外部（`database.enabled=false` + `externalDatabase.*` 接自建 / RDS）。
- schema 迁移由各调用方二进制内嵌 `golang-migrate` 在启动时执行（依赖 PG advisory lock 避免并发迁移）。

库内表 schema 按消费层归各自 docs：System 层见 [system/database.md](../../../axisml-system/docs/system_design/database.md)，Platform 层见 [platform/database.md](../../../axisml-platform/docs/system_design/database.md)。部署模式见 [deployment.md §5.2](../../../docs/deployment.md#52-postgresql内置或外接)。

### 4.4 缓存

**需求**：控制面热点读的可选加速：高频、读多写少、值可由源库重建（典型为认证路径的会话有效性与身份 / RBAC 解析）。要求：低延迟 key/value；TTL 过期；多副本共享（替代进程内缓存）；**非权威**——缓存不可达时调用方必须回退源库，缓存绝不作为唯一真相。

**技术选型**：**Redis**（bitnami/redis 子 chart，`architecture: standalone`）。成熟 key/value 与 TTL 语义；复用现成 chart。缓存内容均为可重建数据，单实例即可，无需 sentinel / 副本 HA——宕机或重启只触发一次回源（及会话强制重登），不丢业务真相。

**对外契约**：

- 部署在 `axisml-infra`（Service `axisml-redis-master`）。调用方经跨 namespace FQDN `axisml-redis-master.axisml-infra:6379` 连接，凭据从共享 `cache.auth.password` 自渲染为本 namespace Secret。
- 可选依赖：调用方未配置地址即跳过缓存（直连源库）；运行中缓存出错按操作回退源库，不影响请求成功。
- key 隔离：调用方按 key 前缀自行命名（如 Platform 用 `platform:`）。

部署模式见 [deployment.md §5.3](../../../docs/deployment.md#53-redis-缓存可选)；Platform 的具体缓存对象与失效策略见 [platform/auth.md §2.1](../../../axisml-platform/docs/system_design/auth.md#21-会话与身份缓存)。

## 5. 加速器管理

**需求**：把节点上的物理 GPU 暴露为 Kubernetes 可调度资源并保持可观测，要求：把 GPU 暴露为 extended resource `nvidia.com/gpu` 供调度器分配；自动化驱动 / 设备插件 / 运行时集成的生命周期（免节点手工装驱动）；按 GPU 型号给节点打标以支持亲和；导出 GPU 利用率 / 显存 / 温度等指标。

**技术选型**：**[NVIDIA GPU Operator](https://github.com/NVIDIA/gpu-operator)**。Kubernetes 原生 GPU 管理的事实标准，驱动 / 工具链容器化满足免手工装驱动；DCGM Exporter 与监控栈天然集成，开箱导出 GPU 指标。

**组件构成**：

| 组件 | 职责 |
| --- | --- |
| GPU Driver Container | 容器化 NVIDIA 驱动，自动安装与升级 |
| NVIDIA Container Toolkit | 容器运行时集成，使容器可访问 GPU |
| Device Plugin | 向调度器报告 `nvidia.com/gpu` 资源 |
| DCGM Exporter | 导出 GPU 利用率 / 显存 / 温度 / 功耗等 Prometheus 指标 |
| GPU Feature Discovery | 自动为节点打标（GPU 型号、驱动版本等） |
| MIG Manager | A100 / H100 多实例分区管理（按需启用） |

**调度契约**：

- 业务 Pod 申请 GPU 使用资源名 `nvidia.com/gpu`；实际调度由 [§6 调度与配额](#6-调度与配额) 完成，本功能不做调度决策。
- 节点标签 `nvidia.com/gpu.product`（如 `A100-SXM4-80GB`）可做 nodeSelector / affinity。
- DCGM Exporter 的 `/metrics` 由 [§7 监控](#7-监控) 自动采集。

## 6. 调度与配额

平台需要一个统一调度器同时满足多租户配额与分布式训练，要求：**多租户弹性配额**（namespace 级 `min`/`max`，空闲容量可被借用、争用时按 `min` 回收，超 `max` 拒绝调度，且**一个 namespace 可挂多个配额**）；**Gang Scheduling**（分布式训练的全部 Pod 要么同时调度、要么都不调度，避免资源死锁）；**Binpack**（优先填满已用节点，减少加速器碎片）；**与默认调度器共存**（只接管 AxisML 工作负载，控制面 / 基础设施 Pod 仍走 kube-scheduler，零副作用）；**不锁定私有扩展**（配额 CR 字段与上游 scheduler-plugins 对齐，降低切换成本）。

**技术选型**：**自研 `axisml-scheduler`**，基于 Kubernetes scheduler-framework 编译，复用上游 [scheduler-plugins](https://github.com/kubernetes-sigs/scheduler-plugins)：Gang 由其 `Coscheduling` plugin 原样提供，Binpack 由 in-tree `NodeResourcesFit`（`MostAllocated` 评分）纯配置提供；仅 ElasticQuota 因平台需要「一个 namespace 多个配额」（上游按 namespace 关联，一 namespace 一配额，不满足）而自研一个薄 plugin——把「Pod → Quota」关联从 namespace 改为 label。CR 词汇表全用上游 `scheduling.x-k8s.io/v1alpha1` 的 `ElasticQuota` / `PodGroup`，不引私有 annotation，避免锁定。详见 [axisml-scheduler.md](axisml-scheduler.md)。

**组件构成**：

| 组件 | 职责 | 启用 |
| --- | --- | --- |
| axisml-scheduler | scheduler-framework 二进制，注册 Coscheduling + NodeResourcesFit + 自研 ElasticScheduling plugin | 启用 |
| axisml-scheduler-controller | 维护 `PodGroup.status` 与 `ElasticQuota.status.used`（按 quota label 聚合用量供调用方回读） | 启用 |

**核心能力**：

- **Gang Scheduling**：通过 `PodGroup`（`scheduling.x-k8s.io/v1alpha1`）表达，同一 PodGroup 全部 Pod 要么同时调度、要么都不调度；由上游 Coscheduling plugin 承载。
- **Binpack**：`NodeResourcesFit` 以 `MostAllocated` 评分优先填满已用节点，加速器资源加权。
- **ElasticQuota**：`ElasticQuota`（namespace-scoped）承载 `min`/`max`，Pod 经 label `scheduling.axisml.io/quota=<eq-name>` 关联到同 namespace 下的具体配额；自研 ElasticScheduling plugin 按该 label（而非 namespace）汇总用量。
- **`max` 强制（已实现）**：超 `max` 的请求一律拒绝调度;缺 quota label 或配额不存在的 Pod 不放行(fail-closed)。**基于 `min` 的借用 / 回收 / 抢占 / Backfill** 为规划能力,详见 [axisml-scheduler.md](axisml-scheduler.md)。

**协作契约（不点名调用方）**：

- **Quota 全覆盖（系统级硬不变式）**：任何接入工作负载 Pod 必须设 `schedulerName: axisml-scheduler` + `scheduling.axisml.io/quota` label，不允许绕过 quota 的调度路径。第三方 controller（如 KServe）派生 Pod 时必须透传这两字段，不支持透传的 controller 不应接入。
- **Gang scheduling 仅按需启用**：分布式训练等全员就位的工作负载创建 PodGroup；常驻服务 / 单 Pod 任务不创建 PodGroup，但仍走 axisml-scheduler 并经 quota label 计入 ElasticQuota。
- **ElasticQuota / PodGroup CR 由调用方独占 owner**：`min`/`max`、`minMember`、命名、补偿、RBAC 全归调用方；本功能不预置任何此类 CR、不持有其 spec mutation 权限（只写 `status`）。
- **与 kube-scheduler 共存**：`axisml-scheduler` 仅接管设了 `schedulerName: axisml-scheduler` 的 Pod；Infra 自身 Pod 不设此字段，走默认 kube-scheduler、不消耗 ElasticQuota。

## 7. 监控

**需求**：平台需要集群与业务的统一可观测性，要求：采集、存储、可视化指标并提供告警通道；覆盖集群 / GPU / 网关 / 调度 / 业务多层指标；采集目标**自动发现**、免手工维护抓取配置；各组件以标准 `/metrics` 接入。

**技术选型**：**[kube-prometheus-stack](https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-prometheus-stack)**（Prometheus + Grafana + AlertManager）。Kubernetes 生态事实标准；Prometheus Operator 经 `ServiceMonitor` / `PodMonitor` 自动发现采集，免维护 `prometheus.yml`；与 GPU Operator 的 DCGM Exporter 开箱即用。

**接入模型**：各组件 (1) 在容器内暴露 `/metrics`（Prometheus 格式，端口 `:8081`）；(2) 随 Helm chart 提供 `ServiceMonitor` / `PodMonitor`，声明待采集对象与端口。Prometheus Operator 自动发现并配置采集目标。当前 compute-service / artifact-hub 提供 `ServiceMonitor`（默认 `*.serviceMonitor.enabled=false`，opt-in），其余组件暴露 metrics 端口、ServiceMonitor 待补。

**指标体系层级**：

| 层级 | 来源 | 典型指标 |
| --- | --- | --- |
| 集群层 | kube-state-metrics、node-exporter | 节点 CPU/内存/磁盘、Pod 状态 |
| GPU 层 | DCGM Exporter（来自 GPU Operator） | GPU 利用率、显存、温度、功耗 |
| 网关层 | Envoy Gateway | 请求量、延迟分位、错误率 |
| 调度层 | axisml-scheduler / axisml-scheduler-controller | ElasticQuota 用量与借用、PodGroup 调度状态、调度延迟 |
| 业务层 | 接入服务 | 各服务自行暴露 `axisml_*` / `platform_*` 指标 |

**告警**：当前**不预置** AlertManager 告警规则——AlertManager 随栈部署但无业务规则，调用方按需自定义。参考方向：节点 NotReady、GPU 异常（DCGM 上报错误率高）、PVC 容量、配额耗尽（ElasticQuota `min` 持续不可满足）、调度滞后（PodGroup gang 长时间 Pending）、API 5xx 比例超阈值。

> 控制面业务指标查询由拥有该域的服务负责；Dashboard 聚合接口待后续专项设计。训练指标走对象存储 + TensorBoard，不进 Prometheus。

---

部署细节（chart 组织、命名空间、安装顺序、依赖清单、fullnameOverride）见 [deployment.md](../../../docs/deployment.md)；三层架构见 [high_level_design.md §4](../../../docs/high_level_design.md#4-整体架构)。
