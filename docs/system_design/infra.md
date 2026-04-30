# AxisML Infra 详细设计

## 1. 概述

AxisML Infra 是平台的基础设施层，由一系列开源组件组成，为上层应用组件（Platform、Compute、Artifacts、Operators）提供底层支撑能力。Infra 层不涉及自研代码，所有组件均通过 Helm 引入并在 AxisML Helm chart 中作为 dependencies 统一管理。

### 1.1 组件清单

| # | 组件 | 技术选型 | 职责 |
| --- | --- | --- | --- |
| 1 | 服务网关 | Envoy Gateway | 请求路由、认证鉴权、流量控制 |
| 2 | 对象存储 | RustFS | 制品文件（模型、镜像 layer、数据集）持久化存储 |
| 3 | 数据库 | PostgreSQL（bitnami chart） | 元数据持久化存储 |
| 4 | GPU 管理 | NVIDIA GPU Operator | GPU 驱动、设备插件与监控 |
| 5 | 批任务调度 | Volcano | Gang Scheduling、队列管理、公平调度 |
| 6 | 监控 | kube-prometheus-stack | 集群与业务可观测性 |

> overview.md 第 4.5 节未显式列出批任务调度器，本文档将其纳入 Infra 层，因为 Gang Scheduling 是分布式训练的硬需求（详见 §7 及 §10 关键设计决策）。

## 2. 整体架构

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                              AxisML Infra                                    │
│                                                                              │
│  ┌──────────────────────┐  ┌──────────────┐  ┌───────────────────────────┐  │
│  │      服务网关         │  │   对象存储    │  │         数据库            │  │
│  │   Envoy Gateway      │  │    RustFS    │  │       PostgreSQL          │  │
│  │   (Gateway API)      │  │   (S3 API)   │  │    (bitnami sub-chart)    │  │
│  └──────────────────────┘  └──────────────┘  └───────────────────────────┘  │
│                                                                              │
│  ┌──────────────────────┐  ┌──────────────────────────────────────────────┐  │
│  │      GPU 管理         │  │              批任务调度                      │  │
│  │ NVIDIA GPU Operator  │  │                Volcano                       │  │
│  └──────────────────────┘  └──────────────────────────────────────────────┘  │
│                                                                              │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │                             监控                                      │   │
│  │                    kube-prometheus-stack                             │   │
│  │          (Prometheus + Grafana + AlertManager)                       │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────────────────────┘
```

调用关系要点：

- 外部流量 → **Envoy Gateway** → Platform / Artifacts 等对外 Service；Compute 仅由 Platform 通过集群内 Service 调用
- Compute / Artifacts → **PostgreSQL**（元数据读写）
- Artifacts → **RustFS**（制品文件读写，S3 API）
- mljob-operator / mlservice-operator 创建的 Pod → **Volcano**（`schedulerName: volcano`）
- 所有 Pod（含 GPU Operator 的 DCGM Exporter、网关、业务组件）→ **kube-prometheus-stack**（`/metrics` 被 ServiceMonitor 自动发现）
- 业务 Pod 申请 `nvidia.com/gpu` → **GPU Operator** 完成设备分配

## 3. 服务网关（Envoy Gateway）

AxisML 采用 [Envoy Gateway](https://gateway.envoyproxy.io/) 作为服务网关，基于 Kubernetes [Gateway API](https://gateway-api.sigs.k8s.io/) 标准以声明式方式配置路由、认证与流量控制。

### 3.1 架构设计

Gateway API 资源模型：

```
GatewayClass (envoy-gateway)
  │
  └── Gateway (axisml-gateway)
        │  Listener: HTTP (80)
        │  Listener: HTTPS (443)
        │
        ├── HTTPRoute (platform)
        │     pathPrefix: /
        │     → axisml-platform Service
        │
        └── HTTPRoute (artifacts-api)
              pathPrefix: /api/artifacts
              → axisml-artifacts Service
```

- **GatewayClass**：由 Envoy Gateway 控制面注册，声明控制器实现。
- **Gateway**：集群内"监听点"的声明，同一份 Gateway 实例承载全部 AxisML 路由。
- **HTTPRoute**：对外业务组件的路由规则，与 Service 绑定；Compute 不配置外部 HTTPRoute。

**MLService 派生路由**：mlservice-operator 在租户 namespace 内为开启了 `spec.route.enabled=true` 的 MLService 创建 namespaced `HTTPRoute` / `SecurityPolicy` / `BackendTrafficPolicy`，`parentRefs` 指向同一份 `axisml-gateway`（跨 namespace 引用通过 `ReferenceGrant` 授权，由本 chart 的 `templates/infra/gateway/` 准备）；与 platform / artifacts 的静态 HTTPRoute 共存。详见 [operators/mlservice-operator.md](operators/mlservice-operator.md) §3 / §6 / §8.1。

### 3.2 认证鉴权

通过 Envoy Gateway 的 `SecurityPolicy` CRD 实现，可附加到 Gateway 或 HTTPRoute 级别：

| 能力 | 说明 |
| --- | --- |
| JWT 验证 | 校验请求头 JWT，支持配置 issuer 与 JWKS 端点 |
| OIDC 集成 | 支持 OpenID Connect，可对接外部身份提供商 |
| ExtAuth | 外部授权服务，支持自定义鉴权逻辑 |
| per-Service 认证 | MLService 通过 `spec.route.auth` 声明本服务的 JWT / API key 策略，由 mlservice-operator 翻译为 namespaced `SecurityPolicy`，`targetRefs` 指向该 MLService 的 HTTPRoute；与 Gateway 级 SecurityPolicy 叠加生效（policy attachment 语义详见 Envoy Gateway 文档） |

具体认证方案（如对接的 IdP）留待 Platform 设计文档确定，Infra 层只保证能力就位。

### 3.3 流量控制

通过 Envoy Gateway 的 `BackendTrafficPolicy` CRD 实现：

| 能力 | 说明 |
| --- | --- |
| 限流 | 支持全局限流和按路由限流 |
| 熔断 | 后端异常时自动熔断，防止级联故障 |
| 超时 / 重试 | 配置请求超时与重试策略 |
| 负载均衡 | Round Robin / Least Request 等算法 |

### 3.4 部署形态

作为 AxisML Helm chart 的 dependency 引入：

```yaml
# deploy/helm/axisml/Chart.yaml
dependencies:
  - name: gateway-helm
    alias: envoy-gateway
    version: v1.3.x
    repository: oci://docker.io/envoyproxy
    condition: envoy-gateway.enabled
```

values.yaml 对应段：

```yaml
envoy-gateway:
  enabled: true
  # Envoy Gateway 子 chart 的 values pass-through
```

AxisML 自身的 `Gateway` / `HTTPRoute` / `SecurityPolicy` 资源由 chart 的 `templates/infra/gateway/` 下模板提供（本文档定义设计，具体模板由后续 PR 落地）。

## 4. 对象存储（RustFS）

AxisML 使用 [RustFS](https://rustfs.dev/) 作为对象存储，用于模型权重、容器镜像 layer、数据集等制品文件的持久化。RustFS 是 Apache 2.0 许可证、基于 Rust 实现、S3 API 兼容的高性能对象存储。

### 4.1 架构设计

RustFS 提供标准 S3 API（`PutObject` / `GetObject` / `DeleteObject` / `ListObjects` / Presigned URL 等），部署支持：

| 模式 | 说明 | 适用场景 |
| --- | --- | --- |
| Standalone | 单 Pod + 单 PVC | 开发、测试 |
| Distributed (4x4) | 4 Pod × 4 PVC | 中等规模生产 |
| Distributed (16x1) | 16 Pod × 1 PVC | 大规模生产 |

### 4.2 用途

- **Artifacts**：模型文件、数据集文件
- **未来**：容器镜像 registry 后端、日志归档

业务组件通过 S3 SDK 访问，对 RustFS 与其他 S3 兼容实现无感知。

### 4.3 部署形态

作为 AxisML Helm chart 的 dependency：

```yaml
# deploy/helm/axisml/Chart.yaml
dependencies:
  - name: rustfs
    version: 0.0.9x
    repository: https://charts.rustfs.com
    condition: rustfs.enabled
```

values.yaml 对应段：

```yaml
rustfs:
  enabled: true
  # rustfs 子 chart 的 values pass-through
```

> **成熟度说明**：截至 2026-04，RustFS 的 app version 为 `1.0.0-alpha.x`，项目仍在活跃迭代。本次选型在"关键设计决策"中已记录风险与切换方案（S3 API 抽象使切换成本有限）。

## 5. 数据库（PostgreSQL）

AxisML 使用 PostgreSQL 作为元数据存储，供 Compute、Artifacts 等 Go 组件持久化结构化数据。

### 5.1 架构设计

支持两种部署模式，由 `postgresql.enabled` 开关切换：

| 模式 | 说明 | 适用场景 |
| --- | --- | --- |
| 内置模式 | 通过 bitnami/postgresql 子 chart 部署（StatefulSet + PVC） | 开发、测试、轻量生产 |
| 外部模式 | 对接外部 PostgreSQL 实例（自建或 RDS） | 中大型生产 |

内置模式采用 bitnami/postgresql 官方 chart，避免自写 StatefulSet 模板——这也是此前自写模板 `database-statefulset.yaml` / `database-service.yaml` 被删除、改由子 chart 提供的原因。

### 5.2 消费方

| 消费方 | 使用场景 |
| --- | --- |
| AxisML Compute | 租户、资源单元、任务元数据 |
| AxisML Artifacts | 模型、镜像、数据集元数据 |

各消费方通过独立 database 或独立 schema 逻辑隔离（具体隔离粒度由各组件设计文档定义）。

### 5.3 部署形态

```yaml
# deploy/helm/axisml/Chart.yaml
dependencies:
  - name: postgresql
    version: 16.x.x
    repository: oci://registry-1.docker.io/bitnamicharts
    condition: postgresql.enabled
```

values.yaml 对应段（bitnami pass-through 字段命名）：

```yaml
postgresql:
  enabled: true          # 内置模式开关；false 时使用 externalPostgresql
  auth:
    database: axisml
    username: axisml
    password: axisml     # 生产环境应使用 existingSecret
  primary:
    persistence:
      size: 10Gi

# 外部模式（postgresql.enabled=false 时生效）
externalPostgresql:
  host: ""
  port: 5432
  database: axisml
  username: axisml
  existingSecret: ""
```

## 6. GPU 管理（NVIDIA GPU Operator）

AxisML 使用 [NVIDIA GPU Operator](https://github.com/NVIDIA/gpu-operator) 管理集群 GPU 资源，自动化驱动、设备插件、监控等组件的生命周期。

### 6.1 组件架构

| 组件 | 职责 |
| --- | --- |
| GPU Driver Container | 容器化 NVIDIA 驱动，自动安装与升级 |
| NVIDIA Container Toolkit | 容器运行时集成，使容器可访问 GPU |
| Device Plugin | 向 kube-scheduler / Volcano 报告 `nvidia.com/gpu` 资源 |
| DCGM Exporter | 导出 GPU 利用率、显存、温度等 Prometheus 指标 |
| GPU Feature Discovery | 自动为节点打标签（GPU 型号、驱动版本等） |
| MIG Manager | A100/H100 的多实例分区管理 |

### 6.2 调度契约

Infra 层对上层的契约：

- 业务 Pod 申请 GPU 时使用资源名 `nvidia.com/gpu`
- 节点标签可基于 `nvidia.com/gpu.product`（如 `A100-SXM4-80GB`）做 nodeSelector / affinity
- DCGM Exporter 的 `/metrics` 端点由 kube-prometheus-stack 自动采集（详见 §8）

### 6.3 部署形态

```yaml
# deploy/helm/axisml/Chart.yaml
dependencies:
  - name: gpu-operator
    version: v24.x.x
    repository: https://helm.ngc.nvidia.com/nvidia
    condition: gpuOperator.enabled
```

values.yaml 对应段：

```yaml
gpuOperator:
  enabled: true
  # gpu-operator 子 chart 的 values pass-through
  # 如 driver.enabled / dcgmExporter.enabled 等
```

## 7. 批任务调度（Volcano）

AxisML 使用 [Volcano](https://volcano.sh/) 作为批任务调度器，与默认的 kube-scheduler 共存，专门接管训练/推理任务的调度。

### 7.1 组件架构

| 组件 | 职责 |
| --- | --- |
| Volcano Scheduler | 自定义调度器，按 `schedulerName: volcano` 接管 Pod 调度 |
| Volcano Controller | 管理 Volcano Job (`vcjob`) 与 `PodGroup` 的生命周期 |
| Volcano Admission Webhook | 准入控制，校验与默认值注入 |

### 7.2 核心能力

| 能力 | 说明 |
| --- | --- |
| **Gang Scheduling** | 同一 PodGroup 的全部 Pod 要么同时调度，要么都不调度——避免分布式训练中部分 Worker 启动造成的资源死锁 |
| **Queue** | 多队列并行，控制任务准入与优先级 |
| **Fair-share** | 基于权重的公平资源分配，契合多租户配额 |
| **Preemption** | 高优先级任务可抢占低优先级任务资源 |
| **Backfill** | 空闲资源回填，提升集群利用率 |

### 7.3 与 MLJob / MLService 的协作契约

本文档定义 Infra 侧契约，具体实现细节见 `operators/`：

- mljob-operator / mlservice-operator 为每个 MLJob / MLService 创建对应的 `PodGroup` 资源，并在 Pod spec 上设置 `schedulerName: volcano`
- Volcano Scheduler 基于 PodGroup 的 `minMember` / `minResources` 执行 Gang Scheduling
- **Volcano `Queue` CR（cluster-scoped）由 Compute 独占 owner**：容量（`spec.capability`）、命名、补偿与 RBAC 均由 Compute 维护，operator 仅通过名称引用（`PodGroup.spec.queue`），不读写 Queue CR。队列与租户 / 资源池的归属关系由 Compute 在 PG 中维护，命名约定 `axisml-<tenant>-<pool>-<queue>` 见 [compute.md §6.2.4](compute.md)
- tenant-operator 只负责租户的 Namespace 与 ResourceQuota 等集群侧资源，不涉及 Volcano Queue CR

### 7.4 与 kube-scheduler 共存

Volcano 仅接管带 `schedulerName: volcano` 的 Pod。Infra 自身（网关、数据库、对象存储、监控）以及 Platform / Compute / Artifacts 等业务组件的 Pod 仍走默认 kube-scheduler，互不干扰。

### 7.5 部署形态

```yaml
# deploy/helm/axisml/Chart.yaml
dependencies:
  - name: volcano
    version: 1.10.x
    repository: https://volcano-sh.github.io/helm-charts
    condition: volcano.enabled
```

values.yaml 对应段：

```yaml
volcano:
  enabled: true
  # volcano 子 chart 的 values pass-through
```

## 8. 监控（kube-prometheus-stack）

AxisML 使用 [kube-prometheus-stack](https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-prometheus-stack) 作为统一监控栈，包含 Prometheus、Grafana、AlertManager 三件套。

### 8.1 架构

```
┌──────────────────────────────────────────────────────────────┐
│                     kube-prometheus-stack                    │
│                                                              │
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

### 8.2 采集模型

各 AxisML 组件只需：

1. 在容器内暴露 `/metrics` 端点（Prometheus 格式）
2. 随 Helm chart 提供对应的 `ServiceMonitor` CRD，声明待采集的 Service 与端口

kube-prometheus-stack 的 Prometheus Operator 会自动发现并配置采集目标，无需手动维护 `prometheus.yml`。

### 8.3 指标体系

| 层级 | 来源 | 典型指标 |
| --- | --- | --- |
| 集群层 | kube-state-metrics、node-exporter | 节点 CPU/内存/磁盘、Pod 状态 |
| GPU 层 | DCGM Exporter（来自 GPU Operator） | GPU 利用率、显存占用、温度、功耗 |
| 网关层 | Envoy Gateway | 请求量、延迟分位、错误率 |
| 调度层 | Volcano | 队列堆积、PodGroup 调度状态 |
| 业务层 | Platform / Compute / Artifacts / Operators | 任务状态、推理延迟、制品数量等自定义指标 |

### 8.4 部署形态

```yaml
# deploy/helm/axisml/Chart.yaml
dependencies:
  - name: kube-prometheus-stack
    alias: kube-prometheus-stack
    version: 6x.x.x
    repository: https://prometheus-community.github.io/helm-charts
    condition: kube-prometheus-stack.enabled
```

values.yaml 对应段：

```yaml
kube-prometheus-stack:
  enabled: true
  # kube-prometheus-stack 子 chart 的 values pass-through
```

## 9. 部署总览

### 9.1 Helm chart 组织

AxisML 拆分为两个独立的 Helm chart，按"基础设施 / 控制平面"职责分层部署：

| Chart | 路径 | Release | Namespace | 资源命名前缀 |
| --- | --- | --- | --- | --- |
| `axisml-infra` | `deploy/helm/axisml-infra` | `axisml-infra` | `axisml-infra` | `axisml-infra-*` |
| `axisml-system` | `deploy/helm/axisml-system` | `axisml` | `axisml-system` | `axisml-*` |

**axisml-infra** 的依赖：

| Dependency | 仓库 | condition |
| --- | --- | --- |
| gateway-helm | oci://docker.io/envoyproxy | `envoy-gateway.enabled` |
| rustfs | https://charts.rustfs.com | `rustfs.enabled` |
| gpu-operator | https://helm.ngc.nvidia.com/nvidia | `gpu-operator.enabled` |
| volcano | https://volcano-sh.github.io/helm-charts | `volcano.enabled` |
| kube-prometheus-stack | https://prometheus-community.github.io/helm-charts | `kube-prometheus-stack.enabled` |

**axisml-system** 的依赖（数据库归控制平面管理）：

| Dependency | 仓库 | condition |
| --- | --- | --- |
| postgresql（aliased 为 `database`） | oci://registry-1.docker.io/bitnamicharts | `database.enabled` |

通过 `condition` 字段保证每个 Infra 组件都可按需启停——例如对接外部 PostgreSQL 时关闭 `database.enabled`，对接现有 Prometheus 时关闭 `kube-prometheus-stack.enabled`。

### 9.2 命名空间约定

- `axisml-infra` 命名空间承载第三方基础设施子 chart 的全部资源；`make helm-install-infra` 默认值。
- `axisml-system` 命名空间承载 AxisML 自研组件（Platform/Compute/Artifacts/Operators）以及元数据数据库 `axisml-database`；`make helm-install-system` 默认值。
- 跨命名空间访问走 `<service>.<namespace>.svc.cluster.local`，例如 Artifacts 调用 RustFS：`axisml-infra-rustfs-svc.axisml-infra:9000`。

### 9.3 安装顺序

```
make cluster-up             # 拉起本地集群
make helm-install-infra     # 先装基础设施（CRDs + 监控栈 + 网关 + GPU + 调度器 + 对象存储）
make helm-install-system    # 再装控制平面（平台组件 + 数据库）
```

卸载顺序相反：`helm-uninstall-system` → `helm-uninstall-infra`。

### 9.4 与 values.yaml 的对应关系

axisml-infra/values.yaml：

| 组件 | values 根键 | 说明 |
| --- | --- | --- |
| 服务网关 | `envoy-gateway` | Envoy Gateway |
| 对象存储 | `rustfs` | RustFS |
| GPU 管理 | `gpu-operator` | NVIDIA GPU Operator |
| 批任务调度 | `volcano` | Volcano（资源名形如 `axisml-infra-scheduler`） |
| 监控 | `kube-prometheus-stack` | kube-prometheus-stack（`fullnameOverride` 设为 `axisml-infra-prometheus`，避开上游 26 字符截断） |

axisml-system/values.yaml：

| 组件 | values 根键 | 说明 |
| --- | --- | --- |
| 数据库 | `database` / `externalDatabase` | PostgreSQL（内置 `axisml-database` / 外接二选一） |

## 10. 关键设计决策

| 决策项 | 决策 | 理由 |
| --- | --- | --- |
| 服务网关 | Envoy Gateway | Gateway API 是 Kubernetes 官方的 Ingress 继任者，Envoy 原生支持 gRPC/HTTP2；配置完全 CRD 化，零外部依赖 |
| 对象存储 | RustFS | Apache 2.0 许可证，S3 兼容；规避 MinIO 自 2021 年转为 AGPLv3 的商用传染风险。风险：项目较年轻（alpha 阶段），社区规模小于 MinIO；S3 API 抽象使切换成本有限 |
| 数据库 | bitnami/postgresql 子 chart | 复用成熟 chart，避免自写 StatefulSet 模板带来的维护负担；`externalDatabase` 段保留用于生产外接 RDS |
| 数据库归属 | 纳入 axisml-system 控制平面 chart | 数据库的生命周期、迁移、备份都和业务组件紧密耦合；与 Platform/Compute/Artifacts 放同一命名空间可共享 Secret、ServiceMonitor，减少跨 chart 引用 |
| GPU 管理 | NVIDIA GPU Operator | Kubernetes 原生 GPU 管理事实标准；DCGM Exporter 与监控栈天然集成 |
| 批任务调度 | Volcano | Gang Scheduling 是分布式训练的硬需求（避免部分 Worker 启动造成资源死锁）；CNCF 孵化项目，Queue + Fair-share 契合多租户模型；与 kube-scheduler 按 `schedulerName` 共存，零副作用 |
| 批任务调度归属 | 纳入 Infra 层（overview 未列出） | 所有 ML 训练任务都必须经由 Volcano 调度，其归属性与 GPU Operator 相同，均为训练能力的底座 |
| 监控 | kube-prometheus-stack | Kubernetes 生态事实标准；ServiceMonitor 自动发现免维护；与 GPU Operator 的 DCGM Exporter 开箱即用 |
| 部署策略 | 拆成 `axisml-infra` / `axisml-system` 两个 chart | 基础设施和控制平面发版节奏、回滚粒度不同；拆分后 infra 可共享给多套 axisml-system 实例。两者通过命名空间 + Service DNS 解耦，仍保持 `condition` 字段支持按需关闭并对接外部实例 |

## 11. 未来规划

本次 Infra 设计范围聚焦核心能力，以下组件暂不引入，留作后续扩展：

- **共享文件存储**（如 JuiceFS）：训练大数据集的 POSIX 挂载，Phase 1 可先通过 PVC + RustFS 的 S3 协议访问解决
- **容器镜像仓库**（如 Harbor）：训练/推理镜像的私有仓库管理，短期内可复用公共 registry
- **日志采集**（如 Fluent Bit + ClickHouse）：集中式日志查询，短期内通过 `kubectl logs` + Prometheus 事件满足
- **链路追踪**：基于 OpenTelemetry 的分布式调用链，与业务组件改造同步推进

上述组件引入时机由后续 roadmap 决定，届时以增量方式更新本文档。
