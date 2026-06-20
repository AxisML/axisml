# AxisML Infra 层概要

Infra 层是平台的基础设施底座，为工作负载与控制面服务提供底层支撑。**设计单位是平台需要的“功能 / 能力”**——每个功能选用一个成熟的开源技术组件来实现，组件是可替换的实现细节而非设计本身。Infra 自身不承载业务逻辑，AxisML 只负责定义功能契约、选型组装并补必要的 glue 资源（Gateway、HTTPRoute、Secret、ConfigMap、ServiceAccount 等）。全部由 Infra 层 chart `axisml-infra` 统一管理（[deployment.md](../deployment.md)）。

| 功能（能力） | 实现技术 | 文档 |
| --- | --- | --- |
| 服务网关 | Envoy Gateway | [gateway.md](gateway.md) |
| 存储（对象存储 / OCI Registry / 数据库） | RustFS · zot · PostgreSQL | [storage.md](storage.md) |
| 加速器管理 | NVIDIA GPU Operator | [accelerator.md](accelerator.md) |
| 调度与配额 | Koordinator | [scheduler.md](scheduler.md) |
| 监控 | kube-prometheus-stack | [monitoring.md](monitoring.md) |

## 1. 功能职责与边界

| 功能（实现技术） | 对外契约 | 不做什么 |
| --- | --- | --- |
| 服务网关（Envoy Gateway） | 提供 `Gateway` 与 listener；接受声明式 `HTTPRoute` / `SecurityPolicy` / `BackendTrafficPolicy` | 不感知业务语义，不内置用户态鉴权策略 |
| 对象存储（RustFS） | 提供 S3 API；admin 凭证由调用方持有并按需签发 presigned URL | 不内置租户模型 / ACL（隔离由调用方在 bucket / prefix 完成） |
| OCI Registry（zot） | 提供 OCI Distribution v2；admin 凭证由调用方持有并签 scope-limited bearer token | 不内置租户；repo 路径命名由调用方决定 |
| 数据库（PostgreSQL） | 提供单一 database，调用方按表前缀逻辑隔离 | 不做应用层 SQL 迁移（由调用方自管） |
| 加速器管理（GPU Operator） | 提供 `nvidia.com/gpu` extended resource、节点标签、DCGM Exporter | 不做调度决策（调度由 koord-scheduler 完成） |
| 调度与配额（Koordinator） | 提供 `koord-scheduler`、`ElasticQuota` / `PodGroup` CRD | 不持有 ElasticQuota / PodGroup CR 写权限（由各 CR owner 派生） |
| 监控（kube-prometheus-stack） | 提供 Prometheus / Grafana / AlertManager；自动发现 ServiceMonitor / PodMonitor | 不主动埋点（各组件自行暴露 `/metrics`） |

**面向接入工作负载的硬不变式**：任何接入本基础设施的工作负载 Pod 必须设 `schedulerName: koord-scheduler` 并携带 label `quota.scheduling.koordinator.sh/name=<elastic-quota-name>`，否则视为绕过配额的 bug（详见 [scheduler.md](scheduler.md)）。

## 2. 调用关系

描述各功能对外接口，不对接入方点名：

- 外部流量 → **服务网关**（按 HTTPRoute 转发到 ClusterIP Service）。
- 接入服务 → **数据库**（元数据读写）；接入服务 / 终端 cli → **对象存储**（S3）/ **OCI Registry**（OCI Distribution v2）。
- 任何接入工作负载 Pod → **调度与配额**（`schedulerName: koord-scheduler` + label `quota.scheduling.koordinator.sh/name` 消费 ElasticQuota）；申请 `nvidia.com/gpu` → **加速器管理** 完成设备分配。
- 已配置采集对象（DCGM Exporter + AxisML ServiceMonitor / PodMonitor）→ **监控**（Prometheus Operator 自动发现）。

## 3. 选型理由汇总

| 功能 | 实现技术 / 选型理由 |
| --- | --- |
| 服务网关 | Envoy Gateway——Gateway API 是官方 Ingress 继任者，原生 gRPC/HTTP2，配置全 CRD 化 |
| 对象存储 | RustFS——Apache 2.0、S3 兼容，规避 MinIO 转 AGPLv3 的传染风险，S3 抽象使切换成本有限 |
| OCI Registry | zot——OCI Distribution v2 + 1.1 artifact manifest 原生支持，CNCF Sandbox、单二进制、可选 S3 后端 |
| 制品分流 | OCI（zot）走不可变内容寻址制品，S3（RustFS）走目录型 / 多文件制品 |
| 数据库 | PostgreSQL（bitnami chart），`externalDatabase` 段保留用于生产外接 RDS |
| 加速器管理 | NVIDIA GPU Operator——K8s 原生 GPU 管理事实标准，DCGM 与监控栈天然集成 |
| 调度与配额 | Koordinator——scheduler-plugins ElasticQuota 提供 namespace-scoped `min`/`max`，PodGroup 提供 Gang Scheduling，统一 koord-scheduler 承载，与 kube-scheduler 按 `schedulerName` 共存 |
| 监控 | kube-prometheus-stack——K8s 生态事实标准，ServiceMonitor 自动发现免维护 |

部署细节（chart 组织、命名空间、安装顺序、依赖清单、fullnameOverride）见 [deployment.md](../deployment.md)；三层架构见 [high_level_design.md §4](../high_level_design.md#4-整体架构)。
