# AxisML Infra 概要设计

## 1. 概述

AxisML Infra 是平台的基础设施层，由一组开源组件组成，为承载在其上的工作负载与控制平面服务提供底层支撑能力。Infra 自身不承载业务逻辑，全部能力由第三方组件提供，AxisML 只负责选型、组装、暴露契约与必要的 glue 资源（Gateway、HTTPRoute、Secret、ConfigMap、ServiceAccount 等）。

| # | 组件 | 技术选型 | 职责 |
| --- | --- | --- | --- |
| 1 | 服务网关 | Envoy Gateway | 请求路由、认证鉴权、流量控制（基于 Gateway API） |
| 2 | 对象存储 | RustFS | S3 兼容对象存储 |
| 3 | OCI Registry | zot | OCI Distribution v2 制品存储 |
| 4 | 数据库 | PostgreSQL（bitnami chart） | 元数据持久化存储 |
| 5 | GPU 管理 | NVIDIA GPU Operator | GPU 驱动、设备插件与监控 |
| 6 | 调度与配额 | Koordinator | koord-scheduler 接管接入工作负载；ElasticQuota 多租户配额；PodGroup gang scheduling |
| 7 | 监控 | kube-prometheus-stack | 集群与业务可观测性 |

部署上全部 7 个第三方组件（含元数据数据库 PostgreSQL）由 Infra 层 chart `axisml-infra` 统一管理（详见 [deployment.md](deployment.md)）。

## 2. 职责与边界

| 组件 | 对外契约 | 不做什么 |
| --- | --- | --- |
| Envoy Gateway | 提供 `Gateway` 与 listener；接受声明式 `HTTPRoute` / `SecurityPolicy` / `BackendTrafficPolicy` | 不感知业务语义，不内置用户态鉴权策略（IdP 对接由调用方决定） |
| RustFS | 提供 S3 API 端点；admin 凭证由调用方持有并按需签发 presigned URL | 不内置租户模型；不做 ACL（隔离由调用方在 bucket / prefix 层完成） |
| zot | 提供 OCI Distribution v2 端点；admin 凭证由调用方持有并签发 scope-limited bearer token | 不内置租户；repo 路径命名由调用方决定 |
| PostgreSQL | 提供单一 database，调用方按 schema 或表前缀逻辑隔离 | 不做应用层 SQL 迁移（由各调用方自管） |
| GPU Operator | 提供 `nvidia.com/gpu` extended resource、节点标签、DCGM Exporter `/metrics` | 不做调度决策；调度由 koord-scheduler 完成 |
| Koordinator | 提供 `koord-scheduler` 调度器、`ElasticQuota` / `PodGroup` CRD | 不持有 ElasticQuota / PodGroup CR 的写权限（由各 CR 的实际 owner 派生） |
| kube-prometheus-stack | 提供 Prometheus / Grafana / AlertManager；自动发现 ServiceMonitor / PodMonitor | 不主动埋点；各组件自行暴露 `/metrics` 与 ServiceMonitor |

**面向接入工作负载的硬不变式**：任何接入本基础设施的工作负载 Pod 必须设置 `schedulerName: koord-scheduler` 并携带 label `quota.scheduling.koordinator.sh/name=<elastic-quota-name>`，否则视为绕过配额的 bug。详见 §4.6.3。

## 3. 整体架构

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                              AxisML Infra                                    │
│                                                                              │
│  ┌──────────────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────┐  │
│  │      服务网关         │  │   对象存储    │  │ OCI Registry │  │  数据库   │  │
│  │   Envoy Gateway      │  │    RustFS    │  │     zot      │  │PostgreSQL│  │
│  │   (Gateway API)      │  │   (S3 API)   │  │ (Distribution│  │ (bitnami)│  │
│  │                      │  │              │  │     v2)      │  │          │  │
│  └──────────────────────┘  └──────────────┘  └──────────────┘  └──────────┘  │
│                                                                              │
│  ┌──────────────────────┐  ┌──────────────────────────────────────────────┐  │
│  │      GPU 管理         │  │              调度与配额                      │  │
│  │ NVIDIA GPU Operator  │  │              Koordinator                     │  │
│  │                      │  │   koord-scheduler + ElasticQuota + PodGroup  │  │
│  └──────────────────────┘  └──────────────────────────────────────────────┘  │
│                                                                              │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │                             监控                                      │   │
│  │                    kube-prometheus-stack                             │   │
│  │          (Prometheus + Grafana + AlertManager)                       │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────────────────────┘
```

调用关系（描述基础设施自身的内向 / 外向接口，不对接入方做点名）：

- 外部流量 → **Envoy Gateway**（按 HTTPRoute 把流量转发到 ClusterIP Service）
- 接入服务 → **PostgreSQL**（元数据读写）
- 接入服务 / 终端 cli → **RustFS**（S3 协议）/ **zot**（OCI Distribution v2 协议）
- 任何接入工作负载 Pod → **Koordinator**（`schedulerName: koord-scheduler` + Pod label `quota.scheduling.koordinator.sh/name` 消费 ElasticQuota）
- 所有 Pod（含 GPU Operator 的 DCGM Exporter）→ **kube-prometheus-stack**（`/metrics` 被 ServiceMonitor 自动发现）
- 业务 Pod 申请 `nvidia.com/gpu` → **GPU Operator** 完成设备分配

## 4. 组件设计

### 4.1 服务网关（Envoy Gateway）

[Envoy Gateway](https://gateway.envoyproxy.io/) 基于 Kubernetes [Gateway API](https://gateway-api.sigs.k8s.io/) 标准以声明式方式配置路由、认证与流量控制。

#### 4.1.1 资源模型

```
GatewayClass (envoy-gateway)
  └── Gateway (axisml-gateway, in axisml-infra ns)
        │  Listener: HTTP (80) / HTTPS (443)
        │     allowedRoutes.namespaces: 放行接入工作负载所在 namespace
        └── HTTPRoute（静态 / 派生）→ 目标 ClusterIP Service
```

- **Gateway**：单一实例 `axisml-gateway` 承载全部路由，由 `axisml-infra` chart 提供。
- **静态 HTTPRoute**：由调用方 chart 一同发布，对接控制面服务对外接口。
- **派生 HTTPRoute**：调用方 controller 在工作负载所在 namespace 内创建 namespaced `HTTPRoute` / `SecurityPolicy` / `BackendTrafficPolicy`，`parentRefs` 指向 `axisml-gateway`；`ReferenceGrant` 仅在跨 namespace `backendRef` 等被引用对象授权场景使用。

#### 4.1.2 认证鉴权

通过 `SecurityPolicy` CRD 实现，可附加到 Gateway 或 HTTPRoute 级别：

| 能力 | 说明 |
| --- | --- |
| JWT 验证 | 校验请求头 JWT，支持配置 issuer 与 JWKS 端点 |
| OIDC 集成 | 支持 OpenID Connect，可对接外部身份提供商 |
| ExtAuth | 外部授权服务，支持自定义鉴权逻辑 |
| per-Service 认证 | 通过 `targetRefs` 指向具体 HTTPRoute；与 Gateway 级 SecurityPolicy 叠加生效 |

具体认证方案（如对接的 IdP）由调用方决定，Infra 层只保证能力就位。

#### 4.1.3 流量控制

通过 `BackendTrafficPolicy` CRD 实现：

| 能力 | 说明 |
| --- | --- |
| 限流 | 全局限流和按路由限流 |
| 熔断 | 后端异常时自动熔断，防止级联故障 |
| 超时 / 重试 | 配置请求超时与重试策略 |
| 负载均衡 | Round Robin / Least Request 等算法 |

### 4.2 对象存储（RustFS）

[RustFS](https://rustfs.dev/) 是 Apache 2.0 许可证、Rust 实现、S3 API 兼容的对象存储。

#### 4.2.1 部署模式

| 模式 | 说明 | 适用场景 |
| --- | --- | --- |
| Standalone | 单 Pod + 单 PVC | 开发、测试、Lite |
| Distributed (4×4) | 4 Pod × 4 PVC | 中等规模生产 |
| Distributed (16×1) | 16 Pod × 1 PVC | 大规模生产 |

具体形态选择详见 [deployment.md](deployment.md)。

#### 4.2.2 对外契约

- 调用方通过 S3 SDK 访问，对 RustFS 与其他 S3 兼容实现无感知。
- 凭证：admin 凭证由 `axisml-infra` 自动生成（或管理员预置），由调用方按 Secret 引用挂入 Pod；presigned URL 与短期凭证由调用方按需签发。
- 命名隔离：bucket / prefix 由调用方按业务模型组织，本服务不内置任何 ACL 或租户模型。

### 4.3 OCI Registry（zot）

[zot](https://zotregistry.dev/) 是 CNCF Sandbox 的 OCI Distribution v2 兼容制品仓库。

#### 4.3.1 协议能力

| 能力 | 说明 |
| --- | --- |
| OCI artifact manifest | 原生支持 `application/vnd.oci.image.manifest.v1+json` + `artifactType`，承载非容器制品（如 ML 模型权重） |
| 内容寻址 | `<repo>@sha256:<digest>` 不可变引用 |
| 后端可插拔 | 本地 filesystem / S3 兼容存储；可把 blob 后端切到 RustFS 实现 OCI metadata + S3 blobs 双层架构 |
| Bearer token 鉴权 | 支持 scope-limited bearer token（`repository:<repo>:push` / `pull`） |
| Manifest 校验 | `HEAD /v2/<repo>/manifests/<ref>` 返回 digest，调用方据此做完整性校验 |

#### 4.3.2 部署模式

| 模式 | 说明 | 适用场景 |
| --- | --- | --- |
| Standalone | 单 Pod + 单 PVC（filesystem 后端） | 开发、测试、Lite |
| HA (3×) | 3 Pod + 共享后端（S3 / RustFS） | 中等规模生产 |

#### 4.3.3 对外契约

zot 本身不感知任何业务模型。基础设施层提供：

| 资源 | 落点 | 由谁维护 |
| --- | --- | --- |
| zot endpoint | ConfigMap | axisml-infra Helm |
| zot admin 凭证 | 平台级 Secret，挂入需要校验 / GC / 签 scope token 的服务 Pod | axisml-infra Helm（自动生成 / 由管理员预置） |
| 公共拉取凭证 | `axisml-system` Namespace Secret | axisml-infra Helm |

repo 路径命名、租户隔离、scope token 的具体形态全部由调用方决定。

### 4.4 数据库（PostgreSQL）

PostgreSQL 是元数据持久化存储。作为第三方依赖归属 Infra 层 chart `axisml-infra`，与 RustFS/zot 同性质，部署在 `axisml-infra` namespace（Service `axisml-database`）。System 层服务把它当外部库消费：经跨 namespace FQDN `axisml-database.axisml-infra:5432` 连接，连接凭据由 System 层从共享 `database.auth.password` 在本 namespace 自渲染为 Secret（Secret 不跨 namespace 引用）。

#### 4.4.1 部署模式

| 模式 | 说明 | 适用场景 |
| --- | --- | --- |
| 内置模式 | bitnami/postgresql 子 chart（StatefulSet + PVC） | 开发、测试、轻量生产 |
| 外部模式 | 对接外部 PostgreSQL 实例（自建 / RDS） | 中大型生产 |

由 Infra 层 `database.enabled` 开关切换内置/关闭；外部模式下 System 层 `database.enabled=false` 并通过 `externalDatabase.{host,port,database,username,existingSecret}` 指向托管实例。各调用方通过独立 schema 或独立 database 逻辑隔离；schema 迁移由各调用方二进制内嵌 `golang-migrate` 在启动时执行，依赖 PG advisory lock 避免多副本并发迁移。Schema 细节详见 [database.md](database.md)。

### 4.5 GPU 管理（NVIDIA GPU Operator）

[NVIDIA GPU Operator](https://github.com/NVIDIA/gpu-operator) 自动化 GPU 驱动、设备插件、监控等组件的生命周期。

#### 4.5.1 组件构成

| 组件 | 职责 |
| --- | --- |
| GPU Driver Container | 容器化 NVIDIA 驱动，自动安装与升级 |
| NVIDIA Container Toolkit | 容器运行时集成，使容器可访问 GPU |
| Device Plugin | 向 koord-scheduler / kube-scheduler 报告 `nvidia.com/gpu` 资源 |
| DCGM Exporter | 导出 GPU 利用率、显存、温度等 Prometheus 指标 |
| GPU Feature Discovery | 自动为节点打标签（GPU 型号、驱动版本等） |
| MIG Manager | A100 / H100 多实例分区管理（按需启用） |

#### 4.5.2 调度契约

- 业务 Pod 申请 GPU 时使用资源名 `nvidia.com/gpu`。
- 节点标签可基于 `nvidia.com/gpu.product`（如 `A100-SXM4-80GB`）做 nodeSelector / affinity。
- DCGM Exporter 的 `/metrics` 端点由 kube-prometheus-stack 自动采集（详见 §4.7）。

### 4.6 调度与配额（Koordinator）

[Koordinator](https://koordinator.sh/) 同时承担统一调度器与多租户配额引擎。**任何接入本基础设施的工作负载 Pod**都强制走 `koord-scheduler` 并消耗对应 ElasticQuota；只有控制平面 Pod 留在默认 kube-scheduler 上。

#### 4.6.1 组件构成

| 组件 | 职责 | 启用 |
| --- | --- | --- |
| koord-scheduler | 自定义调度器，承载 Gang Scheduling 与 ElasticQuota plugin | 启用 |
| koord-manager | 控制器集合，管理 ElasticQuota / PodGroup 等 CR 状态聚合 | 启用 |
| koord-descheduler | 在线服务弹性场景下的 Pod 重平衡 | 暂不启用 |
| koordlet | 节点侧 agent，用于在/离线协同、QoS、弹性资源 | 暂不启用 |

#### 4.6.2 核心能力

| 能力 | 说明 |
| --- | --- |
| **Gang Scheduling** | 通过 sigs.k8s.io scheduler-plugins `PodGroup` (`scheduling.sigs.k8s.io/v1alpha1`) 表达；同一 PodGroup 的全部 Pod 要么同时调度，要么都不调度 |
| **ElasticQuota** | `scheduling.sigs.k8s.io/v1alpha1` `ElasticQuota`（namespace-scoped）承载 `min` / `max`；Pod 通过 label `quota.scheduling.koordinator.sh/name=<eq-name>` 关联到所属 ElasticQuota。本基础设施不引入 Koordinator 私有的 `shared-weight` annotation，借用容量按 koord-scheduler 默认平权处理，让 CR 字段集与上游 scheduler-plugins ElasticQuota 一一对应 |
| **Preemption / Reclaim** | 已分配但低于其他 ElasticQuota `min` 的资源可被回收；高于 `max` 的请求一律拒绝调度 |
| **Backfill** | 空闲资源回填，提升集群利用率 |
| CoLocation / QoS | 在/离线混部、CPU 预算管理；当前不启用，作为未来演进 |

#### 4.6.3 面向接入工作负载的协作契约

本节定义 Infra 侧契约，**不点名具体调用方组件**——任何接入本基础设施的工作负载都遵守同一套约束：

- **Quota 全覆盖（系统级硬不变式）**：任何接入工作负载 Pod 都必须设置 `schedulerName: koord-scheduler` 并携带 label `quota.scheduling.koordinator.sh/name=<elastic-quota-name>`，不允许"绕过 quota 的调度路径"。如果调用方使用第三方 controller（如 KServe `InferenceService`）派生 Pod，必须保证该 controller 把这两个字段透传到派生 Pod；不支持透传的 controller 不应接入本基础设施。
- **Gang scheduling 仅在需要的工作负载启用**：分布式训练等需要全员就位的工作负载创建 PodGroup CR；常驻服务、单 Pod 任务不创建 PodGroup，但仍走 koord-scheduler，仅通过 quota label 计入 ElasticQuota。
- **ElasticQuota CR 由调用方独占 owner**：ElasticQuota CR 的 `spec.min` / `spec.max`、命名、补偿、RBAC 全部归调用方负责；本基础设施不预置任何 ElasticQuota CR，也不为 ElasticQuota CR 持有 mutation 权限。
- **PodGroup CR** 由对应工作负载所属调用方在工作负载所在 namespace 内自管。

#### 4.6.4 与 kube-scheduler 共存

`koord-scheduler` 仅接管设置了 `schedulerName: koord-scheduler` 的 Pod。Infra 自身（网关、对象存储、监控、Koordinator 自身、GPU Operator）的 Pod 不设置 `schedulerName`，自然走默认 kube-scheduler，**不消耗** ElasticQuota，也与 koord-scheduler 互不干扰。

### 4.7 监控（kube-prometheus-stack）

[kube-prometheus-stack](https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-prometheus-stack) 包含 Prometheus、Grafana、AlertManager 三件套。接入细节详见 [monitoring.md](monitoring.md)。

#### 4.7.1 架构

```
┌──────────────────────────────────────────────────────────────┐
│                     kube-prometheus-stack                    │
│  ┌─────────────┐  ┌─────────────┐  ┌──────────────────────┐  │
│  │ Prometheus  │  │   Grafana   │  │    AlertManager      │  │
│  │ 指标采集/存储│  │  可视化看板  │  │  告警通知            │  │
│  └─────────────┘  └─────────────┘  └──────────────────────┘  │
└──────────────────────────────────────────────────────────────┘
         ▲
         │ ServiceMonitor / PodMonitor（CRD 自动发现）
         │
    各组件 /metrics 端点
```

#### 4.7.2 采集模型

各调用方组件只需：(1) 在容器内暴露 `/metrics` 端点（Prometheus 格式）；(2) 随 Helm chart 提供对应的 `ServiceMonitor` CRD，声明待采集的 Service 与端口。Prometheus Operator 自动发现并配置采集目标，无需手动维护 `prometheus.yml`。

#### 4.7.3 指标体系

| 层级 | 来源 | 典型指标 |
| --- | --- | --- |
| 集群层 | kube-state-metrics、node-exporter | 节点 CPU/内存/磁盘、Pod 状态 |
| GPU 层 | DCGM Exporter（来自 GPU Operator） | GPU 利用率、显存占用、温度、功耗 |
| 网关层 | Envoy Gateway | 请求量、延迟分位、错误率 |
| 调度层 | koord-scheduler / koord-manager | ElasticQuota 用量与借用、PodGroup 调度状态、调度延迟 |
| 业务层 | 接入服务 | 由各服务自行暴露 |

## 5. 部署形态

详见 [deployment.md](deployment.md)——含 chart 组织、命名空间约定、安装顺序、依赖清单与 fullnameOverride 约定。系统级位置见 [overview.md](overview.md)。

## 6. 关键设计决策

| 决策项 | 决策 | 理由 |
| --- | --- | --- |
| 服务网关 | Envoy Gateway | Gateway API 是 Kubernetes 官方的 Ingress 继任者，Envoy 原生支持 gRPC/HTTP2；配置完全 CRD 化，零外部依赖 |
| 对象存储 | RustFS | Apache 2.0 许可证，S3 兼容；规避 MinIO 自 2021 年转 AGPLv3 的商用传染风险。S3 API 抽象使切换成本有限 |
| OCI Registry | zot | OCI Distribution v2 + 1.1 artifact manifest 原生支持，对非容器制品的 `artifactType` 语义完整；CNCF Sandbox、单二进制 Go 实现、可选 S3 后端 |
| 制品分流 | OCI（zot）走不可变内容寻址类制品，S3（RustFS）走目录型 / 多文件类制品 | OCI 的 artifact manifest + 内容寻址契合不可变引用语义；S3 的整目录上传 / 流式访问契合"前缀寻址 + 多文件"语义 |
| 数据库 | bitnami/postgresql 子 chart | 复用成熟 chart，避免自写 StatefulSet 模板；`externalDatabase` 段保留用于生产外接 RDS |
| 数据库归属 | 纳入 `axisml-infra` Infra 层 chart | 沿"Infra 层 = 100% 第三方"边界，与 RustFS/zot 同性质；System 层把它当外部库消费，连接凭据用投影 Secret 解决跨 namespace 引用 |
| GPU 管理 | NVIDIA GPU Operator | Kubernetes 原生 GPU 管理事实标准；DCGM Exporter 与监控栈天然集成 |
| 调度与配额 | Koordinator | sigs.k8s.io scheduler-plugins ElasticQuota 提供 namespace-scoped `min` / `max` 多租户配额模型，PodGroup 提供 Gang Scheduling，二者由统一 koord-scheduler 承载；与 kube-scheduler 按 `schedulerName` 共存，零副作用 |
| 配额 CR 表达 | 不引入 Koordinator 私有 annotation，保持与上游 scheduler-plugins ElasticQuota 字段一一对应 | 避免锁定 Koordinator 私有扩展，将来切换或 mirror 上游 scheduler-plugins 的成本最小 |
| Quota 全覆盖 | 任何接入工作负载 Pod 强制走 koord-scheduler | 避免"绕过 quota 的调度路径"；不支持 `schedulerName` 透传的第三方 controller 不应接入 |
| 监控 | kube-prometheus-stack | Kubernetes 生态事实标准；ServiceMonitor 自动发现免维护；与 GPU Operator 的 DCGM Exporter 开箱即用 |
| Chart 拆分 | `axisml-infra` / `axisml-system` / `axisml-platform` 三个 chart，对齐 Platform / System / Infra 职责分层 | 用户面、自研控制面、第三方基础设施三者发版节奏、回滚粒度、对外暴露面各不相同；infra 可共享给多套实例 |
