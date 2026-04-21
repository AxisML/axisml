# AxisML Infra 详细设计

## 1. 概述

AxisML Infra 是平台的基础设施层，由一系列开源组件组成，为上层应用组件（Platform、Compute、Catalog、Operators）提供底层支撑能力，包括流量网关、对象存储、数据库、GPU 管理、批任务调度、监控、日志等。

Infra 层不涉及自研代码，所有组件均为成熟开源项目的部署与配置。

### 1.1 组件清单

| 组件 | 技术选型 | 职责 |
| --- | --- | --- |
| 服务网关 | Envoy Gateway / Envoy | 请求路由、认证鉴权、流量控制 |
| 对象存储 | RustFS | 制品文件（模型、镜像、数据集）持久化存储 |
| 数据库 | PostgreSQL | 元数据持久化存储 |
| GPU 管理 | NVIDIA GPU Operator | GPU 设备发现、驱动管理、监控 |
| 批任务调度 | Volcano | Gang Scheduling、队列管理、公平调度 |
| 共享存储 | JuiceFS | 训练数据集共享挂载 |
| 监控 | kube-prometheus-stack / Netdata | 集群与业务可观测性 |
| 容器镜像仓库 | Harbor | 训练/推理容器镜像存储与管理 |
| 日志采集 | Fluent Bit + ClickHouse | 集中式日志采集与查询 |

## 2. 整体架构

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                              AxisML Infra                                    │
│                                                                              │
│  ┌──────────────────────┐  ┌──────────────┐  ┌───────────────────────────┐  │
│  │    服务网关            │  │   对象存储    │  │         数据库             │  │
│  │  Envoy Gateway (标准) │  │   RustFS     │  │       PostgreSQL          │  │
│  │  Envoy (Lite)        │  │   (S3 兼容)   │  │                           │  │
│  └──────────────────────┘  └──────────────┘  └───────────────────────────┘  │
│                                                                              │
│  ┌──────────────────────┐  ┌──────────────────────────────────────────────┐  │
│  │    GPU 管理           │  │              批任务调度                       │  │
│  │  NVIDIA GPU Operator  │  │              Volcano                        │  │
│  └──────────────────────┘  └──────────────────────────────────────────────┘  │
│                                                                              │
│  ┌──────────────────────┐  ┌──────────────────────────────────────────────┐  │
│  │    共享存储           │  │              监控                            │  │
│  │  JuiceFS             │  │         kube-prometheus-stack                │  │
│  └──────────────────────┘  └──────────────────────────────────────────────┘  │
│                                                                              │
│  ┌──────────────────────┐  ┌──────────────────────────────────────────────┐  │
│  │    容器镜像仓库       │  │              日志采集                        │  │
│  │  Harbor              │  │         Fluent Bit + ClickHouse             │  │
│  └──────────────────────┘  └──────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────────────────┘
```

### 2.1 AxisML vs AxisML Lite

| Infra 组件 | AxisML（标准版） | AxisML Lite |
| --- | --- | --- |
| 服务网关 | Envoy Gateway（Gateway API） | Envoy（静态配置） |
| 对象存储 | RustFS（Helm 子 chart） | RustFS（Docker Compose） |
| 数据库 | PostgreSQL（StatefulSet） | PostgreSQL（Docker Compose） |
| GPU 管理 | NVIDIA GPU Operator | NVIDIA Container Toolkit（`--gpus`） |
| 批任务调度 | Volcano | 不适用（无 Kubernetes） |
| 共享存储 | JuiceFS CSI | 不适用（本地挂载） |
| 监控 | kube-prometheus-stack | Netdata |
| 镜像仓库 | Harbor | 不适用 |
| 日志采集 | Fluent Bit + ClickHouse | 不适用 |

## 3. 服务网关

AxisML 采用 Envoy 作为服务网关的数据面。标准版基于 **Envoy Gateway** 通过 Kubernetes Gateway API 进行声明式配置；Lite 版使用独立 Envoy 容器配合静态配置文件。

### 3.1 架构设计

#### 标准版（Envoy Gateway）

基于 Kubernetes [Gateway API](https://gateway-api.sigs.k8s.io/) 标准，资源模型如下：

```
GatewayClass (envoy-gateway)
  │
  └── Gateway (axisml-gateway)
        │  Listener: HTTP (80)
        │  Listener: HTTPS (443)
        │
        ├── HTTPRoute (platform)
        │     pathPrefix: /
        │     → AxisML Platform Service
        │
        ├── HTTPRoute (compute)
        │     pathPrefix: /api/compute
        │     → AxisML Compute Service
        │
        └── HTTPRoute (catalog)
              pathPrefix: /api/catalog
              → AxisML Catalog Service
```

Envoy Gateway 以独立 Helm release 部署，作为集群级基础设施与 AxisML 应用 Helm chart 解耦。

#### Lite 版（独立 Envoy）

Lite 版使用独立 Envoy 容器，通过静态 `envoy.yaml` 配置文件实现路由，数据面与标准版一致（均为 Envoy Proxy）：

```yaml
# docker-compose.yml
services:
  envoy:
    image: envoyproxy/envoy:v1.32-latest
    ports:
      - "80:80"
    volumes:
      - ./envoy.yaml:/etc/envoy/envoy.yaml
```

### 3.2 认证鉴权

标准版通过 Envoy Gateway 的 **SecurityPolicy** CRD 提供认证鉴权能力：

- **JWT 验证**：验证请求携带的 JWT Token，支持配置 issuer 和 JWKS 端点
- **OIDC 集成**：支持 OpenID Connect 协议，可对接外部身份提供商
- **ExtAuth**：支持外部授权服务，实现自定义鉴权逻辑

SecurityPolicy 可附加到 Gateway 或 HTTPRoute 级别，实现全局或按路由的认证策略。

### 3.3 流量控制

标准版通过 **BackendTrafficPolicy** CRD 提供流量控制能力：

- **限流（Rate Limiting）**：支持全局限流和按路由限流，基于请求速率控制
- **熔断（Circuit Breaking）**：当后端服务异常时自动熔断，防止级联故障
- **超时控制**：配置请求超时和重试策略
- **负载均衡**：支持多种负载均衡算法（Round Robin、Least Request 等）

### 3.4 部署配置

**标准版 Helm 安装：**

```bash
# 安装 Envoy Gateway
helm install envoy-gateway oci://docker.io/envoyproxy/gateway-helm \
  --version v1.3.0 \
  -n envoy-gateway-system --create-namespace
```

Gateway 和 HTTPRoute 资源由 AxisML Helm chart 管理，在 `deploy/helm/axisml/templates/` 中定义。

## 4. 对象存储（RustFS）

AxisML 使用 **RustFS** 作为对象存储，用于制品文件（模型、镜像、数据集）的持久化存储。RustFS 是基于 Rust 实现的高性能对象存储服务，兼容 S3 API，采用 Apache 2.0 许可证。

### 4.1 架构设计

RustFS 提供 S3 兼容的 API 接口，支持标准的 S3 操作（PutObject、GetObject、DeleteObject、ListObjects、Presigned URL 等）。

部署模式：

| 模式 | 说明 | 适用场景 |
| --- | --- | --- |
| 单节点模式 | 单实例部署，数据存储在本地磁盘 | 开发、测试 |
| 分布式模式 | 多节点部署，数据分布式存储 | 生产环境 |

### 4.2 部署配置

**标准版：** 作为 AxisML Helm chart 的子 chart 部署，默认启用。

```yaml
# values.yaml
objectStorage:
  enabled: true
  replicas: 1
  persistence:
    size: 50Gi
  auth:
    rootUser: axisml
    rootPassword: axisml-secret
```

**Lite 版：** 通过 Docker Compose 部署。

```yaml
# docker-compose.yml
services:
  rustfs:
    image: rustfs/rustfs:latest
    ports:
      - "9000:9000"     # S3 API
      - "9001:9001"     # Console
    volumes:
      - rustfs-data:/data
    environment:
      RUSTFS_ROOT_USER: axisml
      RUSTFS_ROOT_PASSWORD: axisml-secret
```

## 5. 数据库（PostgreSQL）

AxisML 使用 **PostgreSQL** 作为元数据存储，服务于多个组件的元数据持久化需求。

### 5.1 架构设计

PostgreSQL 支持两种部署模式：

| 模式 | 说明 | 适用场景 |
| --- | --- | --- |
| 内置模式 | AxisML Helm chart 内置部署 PostgreSQL StatefulSet | 开发、测试、单节点 |
| 外部模式 | 对接外部 PostgreSQL 实例（自建或云托管 RDS） | 生产环境 |

内置模式使用 PostgreSQL 16，通过 StatefulSet + PVC 保证数据持久化。外部模式通过配置外部连接信息接入，AxisML 不负责外部数据库的生命周期管理。

### 5.2 部署配置

**标准版内置模式：**

```yaml
# values.yaml
database:
  enabled: true        # 启用内置 PostgreSQL
  image: postgres:16
  persistence:
    size: 10Gi
  auth:
    database: axisml
    username: axisml
    password: axisml
```

**标准版外部模式：**

```yaml
# values.yaml
database:
  enabled: false       # 禁用内置 PostgreSQL
externalDatabase:
  host: your-postgres-host
  port: 5432
  database: axisml
  username: axisml
  password: axisml
  # existingSecret: postgres-credentials
```

**Lite 版：**

```yaml
# docker-compose.yml
services:
  postgresql:
    image: postgres:16
    ports:
      - "5432:5432"
    environment:
      POSTGRES_DB: axisml
      POSTGRES_USER: axisml
      POSTGRES_PASSWORD: axisml
    volumes:
      - postgres-data:/var/lib/postgresql/data
```

## 6. GPU 管理（NVIDIA GPU Operator）

AxisML 使用 **NVIDIA GPU Operator** 管理 Kubernetes 集群中的 GPU 资源。GPU Operator 自动化 GPU 驱动、设备插件、监控等组件的部署和生命周期管理。

### 6.1 组件架构

NVIDIA GPU Operator 包含以下核心组件：

| 组件 | 职责 |
| --- | --- |
| **GPU Driver Container** | 容器化 NVIDIA 驱动，自动安装和管理 GPU 驱动 |
| **NVIDIA Container Toolkit** | 提供容器运行时支持，使容器能访问 GPU |
| **Device Plugin** | Kubernetes 设备插件，向调度器报告 GPU 资源（`nvidia.com/gpu`） |
| **DCGM Exporter** | GPU 监控指标导出器，提供 GPU 利用率、温度、显存等 Prometheus 指标 |
| **GPU Feature Discovery** | 自动发现 GPU 硬件特性并为节点添加标签（GPU 型号、驱动版本等） |
| **MIG Manager** | Multi-Instance GPU 管理，支持 GPU 分区（A100/H100） |

GPU Operator 作为独立 Helm release 部署，与 AxisML 应用 Helm chart 解耦。

**Lite 版**不使用 GPU Operator，而是直接依赖宿主机安装的 NVIDIA Container Toolkit，通过 Docker Compose 的 `deploy.resources.reservations.devices` 配置 GPU 分配。

### 6.2 部署配置

```bash
# 添加 NVIDIA Helm 仓库
helm repo add nvidia https://helm.ngc.nvidia.com/nvidia
helm repo update

# 安装 GPU Operator
helm install gpu-operator nvidia/gpu-operator \
  --version v24.9.0 \
  -n gpu-operator --create-namespace \
  --set driver.enabled=true \
  --set dcgmExporter.enabled=true
```

## 7. 批任务调度（Volcano）

AxisML 使用 **Volcano** 作为批任务调度器，提供 Gang Scheduling、队列管理和公平调度等 ML 训练场景所需的高级调度能力。Volcano 是 CNCF 孵化项目，专为高性能计算和 AI/ML 工作负载设计。

### 7.1 架构设计

Volcano 包含以下核心组件：

| 组件 | 职责 |
| --- | --- |
| **Volcano Scheduler** | 自定义调度器，替代或配合 kube-scheduler，提供高级调度策略 |
| **Volcano Controller** | 管理 Volcano Job（vcjob）的生命周期 |
| **Volcano Admission** | Webhook 准入控制器，校验和设置默认值 |

核心调度能力：

| 能力 | 说明 |
| --- | --- |
| **Gang Scheduling** | 确保分布式训练的所有 Worker Pod 同时调度或全部不调度，避免资源死锁 |
| **Queue** | 任务队列管理，支持多队列并行，控制任务准入 |
| **Fair-share** | 基于权重的公平资源分配，保证多租户间的资源公平性 |
| **Preemption** | 任务抢占，高优先级任务可抢占低优先级任务的资源 |
| **Backfill** | 空闲资源回填，利用碎片资源调度小任务，提升集群利用率 |

Volcano 作为独立 Helm release 部署，与 AxisML 应用 Helm chart 解耦。

### 7.2 部署配置

```bash
# 安装 Volcano
helm install volcano volcano-sh/volcano \
  --version v1.10.0 \
  -n volcano-system --create-namespace
```

## 8. 共享存储（JuiceFS）

AxisML 使用 **JuiceFS** 作为共享文件存储，为训练任务提供 POSIX 兼容的数据集挂载能力。JuiceFS 是一个云原生分布式文件系统，采用数据与元数据分离架构。

### 8.1 架构设计

```
┌─────────────────────┐
│   JuiceFS Client    │  ← CSI Driver 以 Pod 形式运行
│  (FUSE / CSI)       │
└──────┬──────────────┘
       │
  ┌────┴────┐     ┌───────────┐
  │ 元数据   │     │  数据存储  │
  │PostgreSQL│     │  RustFS   │
  └─────────┘     └───────────┘
```

JuiceFS 的架构天然复用 AxisML 已有的基础设施：

- **元数据引擎**：复用 PostgreSQL，无需额外部署元数据存储
- **数据存储后端**：复用 RustFS（S3 兼容），无需额外部署数据存储

通过 JuiceFS CSI Driver 为 Kubernetes 提供动态 PV 供给（StorageClass），训练任务 Pod 可直接通过 PVC 挂载共享数据卷。JuiceFS 客户端支持本地缓存加速，适合 ML 训练中大数据集反复读取的场景。

## 9. 监控

标准版使用 **kube-prometheus-stack** 提供集群与业务的可观测性；Lite 版使用 **Netdata** 提供零配置的单机实时监控。

### 9.1 标准版（kube-prometheus-stack）

```
┌──────────────────────────────────────────────────────────────┐
│                     kube-prometheus-stack                     │
│                                                              │
│  ┌─────────────┐  ┌─────────────┐  ┌──────────────────────┐ │
│  │ Prometheus   │  │  Grafana    │  │    AlertManager      │ │
│  │             │  │             │  │                      │ │
│  │ 指标采集     │  │  可视化看板  │  │  告警通知            │ │
│  │ 存储        │  │             │  │  (Webhook/Email)     │ │
│  └─────────────┘  └─────────────┘  └──────────────────────┘ │
└──────────────────────────────────────────────────────────────┘
         ▲
         │ ServiceMonitor / PodMonitor
         │
    各组件 /metrics 端点
```

通过 ServiceMonitor CRD 自动发现各 AxisML 组件暴露的 `/metrics` 端点，无需手动配置采集目标。kube-prometheus-stack 作为独立 Helm release 部署。

### 9.2 Lite 版（Netdata）

**Netdata** 是一款零配置的实时监控工具，开箱即用，适合单机和轻量级场景。通过 Docker Compose 部署单个容器即可自动采集宿主机和容器的 CPU、内存、磁盘、网络、GPU 等指标，并提供内置的 Web Dashboard。

```yaml
# docker-compose.yml
services:
  netdata:
    image: netdata/netdata:latest
    ports:
      - "19999:19999"
    cap_add:
      - SYS_PTRACE
    volumes:
      - /proc:/host/proc:ro
      - /sys:/host/sys:ro
      - /var/run/docker.sock:/var/run/docker.sock:ro
```

### 9.3 指标体系

| 层级 | 指标来源 | 典型指标 |
| --- | --- | --- |
| 集群层 | kube-state-metrics、node-exporter | 节点 CPU/内存/磁盘、Pod 状态 |
| GPU 层 | DCGM Exporter（GPU Operator） | GPU 利用率、显存、温度、功耗 |
| 网关层 | Envoy Gateway | 请求量、延迟、错误率 |
| 业务层 | AxisML 各组件 | 训练任务状态、推理延迟、制品数量 |

## 10. 容器镜像仓库（Harbor）

AxisML 使用 **Harbor** 作为容器镜像仓库，提供训练和推理镜像的存储与管理能力。Harbor 是 CNCF 毕业项目，提供镜像存储、漏洞扫描、访问控制、镜像签名等企业级功能。

### 10.1 架构设计

Harbor 提供以下核心能力：

| 能力 | 说明 |
| --- | --- |
| 镜像存储 | OCI 兼容的容器镜像存储 |
| 漏洞扫描 | 集成 Trivy 自动扫描镜像安全漏洞 |
| 访问控制 | 基于项目的 RBAC 权限管理 |
| 镜像复制 | 支持多仓库间镜像同步 |

Harbor 的后端存储可对接 RustFS（S3 兼容），复用已有的对象存储基础设施。

## 11. 日志采集（Fluent Bit + ClickHouse）

AxisML 使用 **Fluent Bit** 作为日志采集器，**ClickHouse** 作为日志存储与查询后端。

### 11.1 架构设计

```
┌──────────┐    ┌──────────┐    ┌──────────────┐
│ 各组件    │    │Fluent Bit│    │  ClickHouse  │
│ 容器日志  ├───►│(DaemonSet)├───►│  (日志存储)   │
│ stdout   ���    │ 采集/过滤 │    │  列式查询     │
└──────────┘    └──────────┘    └──────────────┘
```

- **Fluent Bit**：轻量级日志采集器，以 DaemonSet 方式部署在每个节点，采集容器标准输出日志，支持过滤、解析和路由
- **ClickHouse**：高性能列式数据库，适合日志场景的大规模写入和分析查询。支持 SQL 查询接口，可与 Grafana 集成实现日志可视化

## 12. 关键设计决策

| 决策项 | 决策 | 理由 |
| --- | --- | --- |
| 服务网关 | Envoy Gateway | Gateway API 是 Kubernetes 官方的 Ingress 继任者；Envoy 原生支持 gRPC/HTTP2；零外部依赖（纯 CRD 配置） |
| Lite 版网关 | 独立 Envoy 容器 | 与标准版共享相同的数据面（Envoy），保持行为一致性 |
| 对象存储 | RustFS | S3 ��容、高性能、Apache 2.0 许可证 |
| 数据库 | PostgreSQL | 关系型数据，支持事务，云原生生态成熟 |
| GPU 管理 | NVIDIA GPU Operator | Kubernetes 原生 GPU 管理标准方案，自动化驱动和设备插件生命周期 |
| 批任务调度 | Volcano | CNCF 孵化项目，Gang Scheduling 是分布式训练的核心需求 |
| 共享存储 | JuiceFS | POSIX 兼容、复用已有 RustFS + PostgreSQL、客户端缓存加速 |
| 监控（标准版） | kube-prometheus-stack | Kubernetes 生态事实标准，ServiceMonitor 自动发现 |
| 监控（Lite 版） | Netdata | 零配置实时监控，单机场景开箱即用 |
| 镜像仓库 | Harbor | CNCF 毕业项目，企业级镜像管理，后端可对接 RustFS |
| 日志采集 | Fluent Bit + ClickHouse | Fluent Bit 轻量高效；ClickHouse 列式存储适合日志分析场景 |
| 基础设施部署策略 | 独立 Helm release | 集群级基础设施与应用解耦，独立升级周期 |

## 13. 实现路���

| 阶段 | 组件 | 说明 |
| --- | --- | --- |
| **Phase 1** | 服务网关、对象存储、数据库、GPU 管理、批任务调度 | 平台核心能力所需的基础设施，优先实现 |
| **Phase 2** | 共享存储、监控、容器镜像仓库、日志采集 | 平台基本功能跑通后逐步引入，完善运维与可观测能力 |

## 14. 未来规划

- **链路追踪**：基于 OpenTelemetry 实现分布式链路追踪，打通请求在各组件间的调用链路
