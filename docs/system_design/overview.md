# AxisML 概要设计

## 1. 概述

**AxisML** 是一个一站式机器学习平台，原生支持分布式训练、智能资源调度与弹性伸缩能力，能够统一管理模型从开发、训练到推理与运维的全生命周期流程。

## 2. 核心概念

平台的业务模型围绕以下核心概念组织，后续章节提及的术语均以本节定义为准。

### 2.1 租户（Tenant）

用户归属的基本单位。每个用户必须属于一个租户；租户内部的任务、服务、资源配额与存储相互可见，租户之间完全隔离。租户对应 `Tenant` CRD，由 tenant-operator 负责集群侧资源（如 Namespace、配额）的落地。

### 2.2 资源池（ResourcePool）

集群资源的物理或逻辑划分，由平台管理员定义。典型划分维度：

- **用途**：训练池 / 推理池
- **硬件代际**：A100 池 / H100 池
- **来源**：不同云厂商或地域

ResourcePool 是资源单元与资源队列挂靠的顶层边界——同一池内的资源单元只能被该池内的队列消费，跨池资源相互独立。

### 2.3 资源单元（ResourceUnit）

ResourcePool 内预先定义的资源规格模板，例如 `a100-1x-large`（1×A100 + 8 vCPU + 32 GiB）。用户创建任务或服务时选择一个 ResourceUnit，平台据此申请对应的底层资源，避免在 API 层手工填写 CPU/GPU/内存明细。命名规范详见 [compute.md §6.2.3](compute.md)。

### 2.4 资源队列（Queue）

租户资源配额的承载体。**每个租户在每个 ResourcePool 下拥有一组独立的队列**：

- 每个 `(tenant, pool)` 默认存在队列 `default`
- 用户可创建其他队列（如 `training`、`inference`、`nlp`）用于业务线 / 团队维度的配额拆分
- 当前版本为扁平结构（无父子层级）；分层队列作为后续演进方向

同一租户在不同 ResourcePool 下的队列结构与配额可以完全不同，互不干扰。

### 2.5 计算负载（Compute Workload）

平台中所有运行时算力承载体的统称，按生命周期分为两类——**任务（Job）** 是一次性 workload，有明确终止态；**服务（Service）** 是长驻 workload，通过扩缩容调节容量。两者共享租户、队列、资源池、配额体系；底层执行由对应 operator 按 `spec.backend.{name, engine}` 元组路由到不同 backend 实现（`native`、`kubeflow-trainer`、`kserve`、`custom`），见 §5.3。

### 2.6 任务（Job）

训练任务、分布式训练任务与数据处理任务的统称，对应 `MLJob` CRD，由 mljob-operator 负责生命周期管理。

### 2.7 服务（Service）

模型部署后对外提供在线推理的实体，对应 `MLService` CRD，由 mlservice-operator 负责生命周期管理。

### 2.8 概念速查

| 术语 | 英文名 | 对应对象 |
| --- | --- | --- |
| 租户 | Tenant | `Tenant` CRD |
| 资源池 | ResourcePool | 平台内部对象 |
| 资源单元 | ResourceUnit | 平台内部对象 |
| 资源队列 | Queue | 平台内部对象 |
| 计算负载 | Compute Workload | Job / Service 的概念伞 |
| 任务 | Job | `MLJob` CRD |
| 服务 | Service | `MLService` CRD |

## 3. 功能矩阵

| 功能分类 | 功能项 | 状态 |
| --- | --- | :---: |
| 训练 & 推理 | 模型定制 | TBD |
| | 开发机 | ✅ |
| | 自定义任务 | ✅ |
| | 在线服务 | ✅ |
| 制品中心 | 模型 | ✅ |
| | 镜像 | ✅ |
| | 数据集 | ✅ |
| 系统管理 | 租户管理 | ✅ |
| | 资源池管理 | ✅ |
| | 资源单元管理 | ✅ |
| | 资源队列管理 | ✅ |
| | 数据卷管理 | TBD |

## 4. 整体架构

```
┌───────────────────────────────────────────────────────────────────────────────┐
│                              AxisML Platform                                 │
│  ┌───────────────────────────────┐  ┌──────────────────────────────────┐     │
│  │       Frontend (React)        │  │         Backend (Go)             │     │
│  └───────────────────────────────┘  └──────────────┬───────────────────┘     │
└────────────────────────────────────────────────────┼─────────────────────────┘
                                                     │
              ┌──────────────────────────────────────┼──────────────────────┐
              │                                      │                      │
              ▼                                      ▼                      ▼
┌──────────────────────────┐ ┌───────────────────────────────┐ ┌────────────────────┐
│  AxisML Compute (Go)     │ │     AxisML Catalog (Go)       │ │   AxisML Infra     │
│                          │ │                               │ │                    │
│  API / 业务逻辑 / 编排    │ │  模型管理 / 镜像管理 / 数据集   │ │  gpu-operator      │
│  租户 / 资源单元 / 队列   │ │                               │ │  对象存储           │
│                          │ │      │            │           │ │  监控 / ...        │
└────────────┬─────────────┘ │ ┌────▼─────┐ ┌────▼─────┐    │ │                    │
             │               │ │PostgreSQL│ │ 对象存储  │    │ │                    │
             │               │ │ (元数据)  │ │(制品存储) │    │ │                    │
             ▼               │ └──────────┘ └──────────┘    │ │                    │
┌──────────────────────────┐ └───────────────────────────────┘ └────────────────────┘
│  AxisML Operators (Go)   │
│                          │
│  mljob-operator          │
│  mlservice-operator      │
│  tenant-operator         │
└──────────────────────────┘
```

## 5. 组件职责

### 5.1 AxisML Platform

平台层，提供用户交互入口。

- **前端**：基于 TypeScript + React，提供 Web UI，涵盖任务管理、服务管理、制品管理、系统管理等功能页面。
- **后端**：基于 Go，提供 RESTful API，负责业务逻辑编排，协调 AxisML Compute 和 AxisML Catalog 完成具体操作。

> 详细设计见 [AxisML Platform 设计文档](platform.md)

### 5.2 AxisML Compute

计算服务层，基于 Go 开发，提供 API 接口，承载以下核心职责：

- **计算任务管理**：训练任务与模型服务的创建、编排与生命周期管理（实际调度由 Volcano 承担）。
- **租户管理**：租户的创建、资源隔离，以及该租户在各 ResourcePool 下队列元数据的承载；通过 Tenant CRD 与 tenant-operator 协作。
- **资源池管理**：ResourcePool 的定义、纳管节点分组，以及与底层集群的映射关系。
- **资源单元管理**：ResourcePool 内的资源规格模板（如 GPU 类型、CPU/内存配置）定义与管理。
- **资源队列管理**：扁平队列（v1 无父子层级）的 CRUD、配额分配、配额使用量统计，以及任务/服务提交时的配额校验。

> 详细设计见 [AxisML Compute 设计文档](compute.md)

### 5.3 AxisML Operators

Kubernetes Operator 组件，基于 Go 开发，通过 CRD 对核心概念进行抽象，负责在 Kubernetes 上执行具体的资源编排：

| 自定义资源 | Operator | 职责 |
| --- | --- | --- |
| **MLJob** | mljob-operator | 计算任务的生命周期管理（创建、调度、监控、清理） |
| **MLService** | mlservice-operator | 在线推理服务的生命周期管理（部署、扩缩容、流量管理） |
| **Tenant** | tenant-operator | 租户的资源配额、隔离与管理 |

AxisML Compute 通过创建/更新 CRD 资源与 AxisML Operators 协作，Operators 负责将声明式定义转化为实际的 Kubernetes 资源。

mljob-operator 与 mlservice-operator 内部按 `spec.backend.{name, engine}` 二级元组路由到不同 Handler，把工作下放到对应的执行后端：

| Backend | 适用 CRD | 说明 |
| --- | --- | --- |
| `native` | MLJob, MLService | 内置 PodGroup+Pod / Deployment+Service 实现，无外部依赖 |
| `kubeflow-trainer` | MLJob | 委托给 [Kubeflow Training Operator](https://www.kubeflow.org/docs/components/training/) 的 PyTorchJob / TFJob / MPIJob 等 |
| `kserve` | MLService | 委托给 [KServe](https://kserve.github.io/website/) 的 InferenceService（Triton / TFServing / TorchServe / HuggingFace 等 predictor） |
| `custom` | MLJob, MLService | 用户自定义后端，通过 `backend.config` 描述目标 GVK 与字段映射 |

backend 选择是 operator 的内部实现细节——MLJob / MLService 仍是用户和 Compute 看到的稳定抽象。

> 详细设计见 [AxisML Operators 设计文档](operators/)

### 5.4 AxisML Catalog

制品管理服务，基于 Go 开发，提供 API 接口，负责平台中各类制品的统一管理：

- **模型管理**：模型版本、元数据、存储路径。
- **镜像管理**：训练/推理镜像的注册与版本管理。
- **数据集管理**：数据集元数据、存储位置、版本。

元数据存储于 PostgreSQL，制品文件存储于对象存储。

> 详细设计见 [AxisML Catalog 设计文档](catalog.md)

### 5.5 AxisML Infra

基础设施层，由开源组件组成，为平台提供底层支撑能力：

- **服务网关**：API 网关，负责请求路由、认证鉴权、流量控制等。
- **GPU 管理**：gpu-operator，负责 GPU 设备的发现、驱动管理与分配。
- **对象存储**：例如 RustFS，用于制品文件的持久化存储。
- **监控**：kube-prometheus-stack，提供集群与业务的可观测性。

> 详细设计见 [AxisML Infra 设计文档](infra.md)

## 6. 部署架构

基于 Kubernetes 部署，通过 Helm Chart 进行安装与管理，各组件以容器化方式运行：

```
Kubernetes Cluster
├── AxisML Platform (Deployment + Service)
├── AxisML Compute (Deployment + Service)
├── AxisML Operators
│   ├── mljob-operator (Deployment)
│   ├── mlservice-operator (Deployment)
│   └── tenant-operator (Deployment)
├── AxisML Catalog (Deployment + Service)
├── AxisML Infra
│   ├── 服务网关
│   ├── gpu-operator
│   ├── 对象存储
│   └── kube-prometheus-stack
└── 数据层
    └── PostgreSQL (StatefulSet)
```

## 7. 项目结构

采用 Monorepo 管理所有组件，目录结构如下：

```
axisml/
├── components/                   # 各组件代码
│   ├── platform/                 # AxisML Platform
│   │   ├── backend/              # 后端（Go）
│   │   │   ├── cmd/              # 服务入口
│   │   │   ├── internal/         # 业务逻辑
│   │   │   └── api/              # API 定义
│   │   └── frontend/             # 前端（React）
│   │       ├── src/
│   │       ├── package.json
│   │       └── tsconfig.json
│   ├── compute/                  # AxisML Compute（Go）
│   │   ├── cmd/                  # 服务入口
│   │   ├── internal/             # 业务逻辑
│   │   └── api/                  # API 定义
│   ├── operators/                # AxisML Operators（Go）
│   │   ├── cmd/                  # 各 Operator 入口
│   │   │   ├── mljob-operator/
│   │   │   ├── mlservice-operator/
│   │   │   └── tenant-operator/
│   │   ├── internal/             # Operator 实现
│   │   │   ├── mljob/
│   │   │   ├── mlservice/
│   │   │   └── tenant/
│   │   └── api/                  # CRD 类型定义
│   └── catalog/                  # AxisML Catalog（Go）
│       ├── cmd/                  # 服务入口
│       ├── internal/             # 业务逻辑
│       └── api/                  # API 定义
├── pkg/                          # 跨组件可复用的公共库
├── deploy/                       # 部署配置
│   └── helm/                     # Helm Charts
│       ├── axisml-system/        # 系统组件 Chart
│       │   ├── Chart.yaml
│       │   ├── values.yaml
│       │   ├── crds/             # CRD 定义（MLJob/MLService/Tenant）
│       │   └── templates/
│       │       ├── _helpers.tpl  # 共享模板函数
│       │       ├── platform/     # AxisML Platform 模板
│       │       ├── compute/      # AxisML Compute 模板
│       │       ├── catalog/      # AxisML Catalog 模板
│       │       └── operators/    # AxisML Operators 模板
│       │           ├── mljob-operator/
│       │           ├── mlservice-operator/
│       │           └── tenant-operator/
│       └── axisml-infra/         # 基础设施 Chart（gateway / gpu-operator / 对象存储 / 监控）
├── build/                        # 构建相关
│   └── docker/                   # 各组件 Dockerfile
│       ├── platform.Dockerfile
│       ├── compute.Dockerfile
│       ├── catalog.Dockerfile
│       ├── mljob-operator.Dockerfile
│       ├── mlservice-operator.Dockerfile
│       └── tenant-operator.Dockerfile
├── docs/                         # 文档
│   └── system_design/
├── Makefile
├── go.mod
├── go.sum
└── README.md
```

### 7.1 目录说明

| 目录 | 说明 |
| --- | --- |
| `components/` | 各组件代码，每个组件独立子目录，内部遵循 Go 标准项目布局 |
| `components/platform/` | 平台层，包含 backend（Go）和 frontend（React）两个子项目 |
| `components/compute/` | 计算服务，Go 服务 |
| `components/operators/` | Kubernetes Operators，包含 3 个 Operator |
| `components/catalog/` | 制品管理服务，Go 服务 |
| `pkg/` | 跨组件复用的公共库（如日志、配置、错误处理等） |
| `deploy/helm/` | Helm Chart |
| `build/docker/` | 各组件的 Dockerfile |

## 8. 关键设计决策

| 决策项 | 决策 | 理由 |
| --- | --- | --- |
| 计算任务抽象 | 通过 CRD（MLJob/MLService/Tenant）抽象 | 与 Kubernetes 原生集成，声明式管理，框架无关 |
| 制品元数据存储 | PostgreSQL | 关系型数据，支持事务，生态成熟 |
| 制品文件存储 | 对象存储 | 适合大文件存储，支持版本管理 |
| 后端语言 | Go | 云原生生态契合，Operator 开发原生支持 |
| 前端框架 | TypeScript + React | 社区生态成熟，组件丰富 |
| 系统管理归属 | 租户、资源池、资源单元、资源队列管理放在 AxisML Compute 中 | 与计算任务强耦合（配额校验、资源分配），避免跨服务调用开销；内部按 package 隔离，保留后续拆分空间 |
| 认证鉴权 | TBD（考虑开源方案） | 待评估 |
