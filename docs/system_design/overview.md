# AxisML 概要设计

## 1. 概述

**AxisML** 是一个面向机器学习工作负载的一站式平台，覆盖模型开发、训练、制品管理、在线推理与运维管理。本文档是系统设计的高层导航，描述平台边界、核心概念、主要组件与关键设计取舍；字段级 schema、状态机和具体实现契约以各详细设计文档为准。

## 2. 核心概念

平台的业务模型围绕以下核心概念组织，后续章节提及的术语均以本节定义为准。

### 2.1 租户（Tenant）

用户归属的基本单位。每个用户必须属于一个租户；任务、服务、制品可见性、资源配额与初始化资源均按租户组织。租户对应 cluster-scoped `Tenant` CRD，由 tenant-operator 负责目标 Namespace、ResourceQuota、Secret、ConfigMap、ServiceAccount 等集群侧资源落地。

Tenant 的目标 Namespace 通过 `spec.namespace.name` 显式声明，多个 Tenant 可以共享同一个 Namespace。平台的隔离边界主要由租户业务模型、per-tenant 资源命名、配额与鉴权策略共同表达，而不是简单等同于 Kubernetes Namespace 的一租户一命名空间隔离。

### 2.2 资源池（ResourcePool）

集群资源的物理或逻辑划分，由平台管理员定义，是 AxisML Compute 维护的元数据对象。典型划分维度：

- **用途**：训练池 / 推理池
- **硬件代际**：A100 池 / H100 池
- **来源**：不同云厂商或地域

ResourcePool 通过节点选择条件和容忍配置描述资源边界。ResourceUnit 与 Quota 均挂靠在 ResourcePool 下，同一池内的资源单元只能被该池内的配额消费，跨池资源相互独立。

### 2.3 资源单元（ResourceUnit）

ResourcePool 内预先定义的资源规格模板，是 AxisML Compute 维护的元数据对象。例如 `a100-1x-large` 可表示 1xA100 + 8 vCPU + 32 GiB。用户创建任务或服务时选择一个 ResourceUnit，平台据此注入 `requests` / `limits` 和节点匹配条件，避免在 API 层手工填写 CPU/GPU/内存明细。命名规范详见 [compute.md §6.2.3](compute.md)。

### 2.4 资源配额（Quota）

租户在某个 ResourcePool 下的配额承载体。Quota 在 Compute 中保存 PG 元数据，并与一条 namespace-scoped Koordinator `ElasticQuota` CR 1:1 映射（CR 落在租户 namespace 下），由 Compute 负责配额下行与用量回流。

- 每个 `(tenant, pool)` 默认存在配额 `default`
- 用户可创建其他配额（如 `training`、`inference`、`nlp`）用于业务线 / 团队维度的拆分
- 配额采用上游 sigs.k8s.io scheduler-plugins ElasticQuota 的原生二维模型：`min`（保留份额，不可被抢占下界）、`max`（硬上限）；不引入 Koordinator 私有的 `shared-weight` annotation，借用容量分配按 koord-scheduler 默认平权处理
- ElasticQuota CR 命名约定：`axisml-<tenant>-<pool>-<quota>`
- 当前版本为扁平结构（无父子层级）；分层配额（Koord-Queue tree）作为后续演进方向

同一租户在不同 ResourcePool 下的配额结构与额度可以完全不同，互不干扰。

### 2.5 计算负载（Compute Workload）

平台中所有运行时算力承载体的统称，按生命周期分为两类：**任务（Job）** 是一次性 workload，有明确终止态；**服务（Service）** 是长驻 workload，通过扩缩容调节容量。两者共享租户、队列、资源池、配额体系。

MLJob / MLService 的底层执行由对应 operator 按 `spec.backend.{name, engine}` 元组路由到不同 backend handler。默认 backend 为 MLJob `(native, job)` 与 MLService `(native, deployment)`；分布式训练的 gang scheduling 由 `(native, podgroup)` 或 `(kubeflow-trainer, *)` 表达。所有 backend 派生的 Pod 都强制走 koord-scheduler 并消耗对应 ElasticQuota；Kubeflow Training Operator、KServe 等作为可选执行后端接入。Compute 只维护业务元数据、下发 CR 并消费统一的 `status.phase`，不感知具体 backend 的底层状态细节。

### 2.6 任务（Job）

训练任务、分布式训练任务与数据处理任务的统称，对应 `MLJob` CRD，由 mljob-operator 负责生命周期管理。

### 2.7 服务（Service）

模型部署后对外提供在线推理的实体，对应 `MLService` CRD，由 mlservice-operator 负责生命周期管理。

### 2.8 概念速查

| 术语 | 英文名 | 对应对象 |
| --- | --- | --- |
| 租户 | Tenant | `Tenant` CRD |
| 资源池 | ResourcePool | Compute 元数据对象 |
| 资源单元 | ResourceUnit | Compute 元数据对象 |
| 资源配额 | Quota | Compute 元数据对象 + Koordinator `ElasticQuota` CR |
| 计算负载 | Compute Workload | Job / Service 的概念伞 |
| 任务 | Job | `MLJob` CRD |
| 服务 | Service | `MLService` CRD |

## 3. 功能矩阵

本表表示系统设计覆盖状态，不等同于代码实现完成度。

| 功能分类 | 功能项 | 设计状态 |
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
| | 资源配额管理 | ✅ |
| | 数据卷管理 | TBD |

图例：`✅` 表示已有对应详细设计；`TBD` 表示概要中保留能力入口，详细设计待补充或待稳定。

## 4. 整体架构

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                                External Users                                │
└──────────────────────────────────────┬───────────────────────────────────────┘
                                       │
                                       ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│                         AxisML Infra: Envoy Gateway                          │
│                     路由 / 认证接入 / 流量控制 / MLService 路由                 │
└──────────────────────────────────────┬───────────────────────────────────────┘
                                       │
                                       ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│                              AxisML Platform                                 │
│            Frontend (React) + Backend (Go) / 用户入口 / 业务编排               │
└───────────────────────┬──────────────────────────────┬───────────────────────┘
                        │                              │
                        ▼                              ▼
┌──────────────────────────────────┐     ┌────────────────────────────────────┐
│        AxisML Compute (Go)        │     │       AxisML Artifacts (Go)        │
│  Job / Service / Tenant / Quota   │     │  模型 / 镜像 / 数据集元数据与引用     │
│  ResourcePool / ResourceUnit      │     │  model,image -> zot               │
│  CR 声明 / ElasticQuota 同步 / 回流 │     │  dataset -> RustFS                │
└───────────────┬──────────────────┘     └───────────────┬────────────────────┘
                │                                        │
                ▼                                        ▼
┌──────────────────────────────────┐     ┌────────────────────────────────────┐
│        AxisML Operators (Go)      │     │           Metadata DB              │
│  mljob / mlservice / tenant       │     │           PostgreSQL               │
│  backend handler -> K8s 资源       │     └────────────────────────────────────┘
└───────────────┬──────────────────┘
                │
                ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│                              Kubernetes Cluster                              │
│  Workloads / Tenant Resources / ElasticQuota / PodGroup / HTTPRoute          │
└──────────────────────────────────────────────────────────────────────────────┘

AxisML Infra 还提供：RustFS、zot、Koordinator、NVIDIA GPU Operator、kube-prometheus-stack。
```

核心调用关系：

- 外部流量经 Envoy Gateway 进入 Platform；Compute 不直接暴露到集群外。
- Platform 调用 Compute 管理任务、服务、租户、资源池、资源单元与队列。
- Platform / CLI / Operators 调用 Artifacts 解析或管理模型、镜像、数据集。
- Compute 通过 CRD 与 Operators 协作，通过 PostgreSQL 保存业务元数据，通过 Koordinator `ElasticQuota` CR 同步配额。
- Artifacts 元数据存 PostgreSQL；模型与镜像走 zot；数据集走 RustFS。其他 Artifact 扩展类型由详细设计承载，概要不逐一展开。
- Operators 把 MLJob / MLService / Tenant 的声明式定义翻译为 Kubernetes 与第三方 CR 资源，并把统一状态回流给 Compute。

## 5. 组件职责

### 5.1 AxisML Platform

平台层，提供用户交互入口与统一业务入口。

- **前端**：基于 TypeScript + React，提供 Web UI，涵盖任务管理、服务管理、制品管理、系统管理等功能页面。
- **后端**：基于 Go，提供 RESTful API，负责业务逻辑编排，协调 AxisML Compute 和 AxisML Artifacts 完成具体操作。
- **认证鉴权入口**：Platform 是用户身份、角色与租户访问控制的统一入口；具体 IdP、角色模型和鉴权细节仍待 [platform.md](platform.md) 补充。

> 详细设计见 [AxisML Platform 设计文档](platform.md)

### 5.2 AxisML Compute

计算服务层，基于 Go 开发，仅接受 Platform 的内部调用，承载以下核心职责：

- **计算负载管理**：维护 Job / Service 业务元数据，创建或更新 `MLJob` / `MLService` CR，并通过 Informer 消费 operator 回流状态。
- **租户管理**：维护租户元数据，下发 `Tenant` CR，由 tenant-operator 负责 Namespace、ResourceQuota 与初始化资源落地。
- **资源池管理**：维护 ResourcePool 的节点选择、容忍配置与底层集群映射关系。
- **资源单元管理**：维护 ResourcePool 内的资源规格模板，并在提交 Job / Service 时注入资源请求与节点匹配条件。
- **资源配额管理**：维护扁平配额的 CRUD、`min` / `max` 配额、best-effort 预检、Koordinator `ElasticQuota` CR 同步与用量缓存。

Compute 不直接创建 Pod、Deployment、PodGroup 等运行时资源；这些资源由对应 operator 或底层 controller 根据 CR 声明生成。

> 详细设计见 [AxisML Compute 设计文档](compute.md)

### 5.3 AxisML Operators

Kubernetes Operator 组件，基于 Go 开发，通过 CRD 对核心概念进行抽象，负责在 Kubernetes 上执行具体资源编排：

| 自定义资源 | Operator | 职责 |
| --- | --- | --- |
| `MLJob` | mljob-operator | 任务生命周期管理，按 backend handler 创建底层训练 / 批处理资源并回流状态 |
| `MLService` | mlservice-operator | 在线推理服务生命周期管理，创建部署、服务路由或 KServe 等后端资源并回流状态 |
| `Tenant` | tenant-operator | 租户目标 Namespace、ResourceQuota、初始化 Secret / ConfigMap / ServiceAccount 等资源维护 |

mljob-operator 与 mlservice-operator 内部按 `spec.backend.{name, engine}` 二级元组路由到不同 Handler：

| Backend | 适用 CRD | engine 示例 | 说明 |
| --- | --- | --- | --- |
| `native` | MLJob、MLService | MLJob: `job` / `podgroup`；MLService: `deployment` / `statefulset` | 直接使用 Kubernetes 原生工作负载与 sigs.k8s.io scheduler-plugins `PodGroup`；`(native, podgroup)` 提供单角色 gang scheduling |
| `kubeflow-trainer` | MLJob | `pytorchjob` / `tfjob` / `mpijob` / ... | 委托给 Kubeflow Training Operator 的对应 CR；多角色任务统一走此路径 |
| `kserve` | MLService | `inference` / `llminference` | `inference` 对应 KServe `InferenceService`；`llminference` 是 LLM 原生服务占位路径，具体 GVK 与字段随 KServe LLM API 落地细化 |
| `custom` | MLJob、MLService | 任意 | 用户自定义后端，通过 `backend.config` 描述目标 GVK 与字段映射 |

默认值：MLJob 为 `(native, job)`、MLService 为 `(native, deployment)`。backend 选择是 operator 的扩展机制；MLJob / MLService 仍是用户和 Compute 看到的稳定抽象。**所有 backend 派生的 Pod 都必须设置 `schedulerName: koord-scheduler` 并携带 ElasticQuota label**（详见 [infra.md §8](infra.md)）。

> 详细设计见 [AxisML Operators 设计文档](operators/)

### 5.4 AxisML Artifacts

制品管理服务，基于 Go 开发，负责平台中主要制品类型的元数据管理、引用解析与存储凭证签发：

- **模型管理**：模型版本、元数据、不可变引用与存储 URI。
- **镜像管理**：训练 / 推理镜像的注册、版本管理与引用解析。
- **数据集管理**：数据集元数据、存储位置、版本与访问凭证。

Artifacts 采用元数据服务 / 存储后端分离模式：元数据存储于 PostgreSQL；模型与镜像通过 OCI Distribution 协议存储在 zot；数据集通过 S3 协议存储在 RustFS。上传下载由 CLI 或消费方直连存储后端，Artifacts 不代理大文件 bytes。其他 Artifact 扩展类型由 [artifacts.md](artifacts.md) 详细设计定义。

> 详细设计见 [AxisML Artifacts 设计文档](artifacts.md)

### 5.5 AxisML Infra

基础设施层，由开源组件组成，为平台提供底层支撑能力：

| 能力 | 组件 | 职责 |
| --- | --- | --- |
| 服务网关 | Envoy Gateway | 外部请求路由、认证接入、流量控制、MLService 路由承载 |
| 对象存储 | RustFS | 数据集等 S3 类制品文件持久化 |
| OCI Registry | zot | 模型、容器镜像等 OCI 类制品存储与分发 |
| 数据库 | PostgreSQL | Compute / Artifacts 等业务元数据持久化 |
| GPU 管理 | NVIDIA GPU Operator | GPU 驱动、设备插件、DCGM 监控集成 |
| 调度与配额 | Koordinator | koord-scheduler 接管所有 AxisML workload；ElasticQuota 多租户配额；PodGroup gang scheduling |
| 监控 | kube-prometheus-stack | 集群与业务指标采集、告警与可视化 |

> 详细设计见 [AxisML Infra 设计文档](infra.md)

## 6. 部署架构

AxisML 基于 Kubernetes 部署，通过两个 Helm chart 分层安装：

- `axisml-infra`：第三方基础设施组件，默认安装到 `axisml-infra` Namespace。
- `axisml-system`：AxisML 自研控制平面与元数据数据库，默认安装到 `axisml-system` Namespace。

```
Kubernetes Cluster
├── axisml-infra namespace
│   ├── Envoy Gateway
│   ├── RustFS
│   ├── zot
│   ├── NVIDIA GPU Operator
│   ├── Koordinator
│   └── kube-prometheus-stack
├── axisml-system namespace
│   ├── AxisML Platform (Deployment + Service)
│   ├── AxisML Compute (Deployment + Service)
│   ├── AxisML Artifacts (Deployment + Service)
│   ├── AxisML Operators
│   │   ├── mljob-operator (Deployment)
│   │   ├── mlservice-operator (Deployment)
│   │   └── tenant-operator (Deployment)
│   └── PostgreSQL / externalDatabase
└── tenant namespaces
    └── Tenant resources / workloads / routes / secrets / ElasticQuota
```

安装顺序为先 `axisml-infra`、后 `axisml-system`；卸载顺序相反。PostgreSQL 随 `axisml-system` 管理，也可通过 `externalDatabase` 对接外部实例。

## 7. 项目结构

采用 Monorepo 管理所有组件。当前仓库已包含文档、Helm chart 与本地开发脚本；以下是目标代码与部署结构，用于说明组件边界和后续落地方向：

```
axisml/
├── components/                   # 目标：各组件代码
│   ├── platform/                 # AxisML Platform
│   │   ├── backend/              # 后端（Go）
│   │   └── frontend/             # 前端（React）
│   ├── compute/                  # AxisML Compute（Go）
│   ├── operators/                # AxisML Operators（Go）
│   │   ├── mljob/
│   │   ├── mlservice/
│   │   └── tenant/
│   └── artifacts/                # AxisML Artifacts（Go）
├── deploy/
│   └── helm/
│       ├── axisml-infra/         # 基础设施 Chart
│       └── axisml-system/        # 控制平面 Chart + CRDs + 数据库依赖
├── docs/
│   ├── development/
│   └── system_design/
├── scripts/                      # 本地开发与运维脚本
├── Makefile
└── README.md
```

### 7.1 目录说明

| 目录 | 说明 |
| --- | --- |
| `components/` | 目标组件代码目录，后续按 Platform / Compute / Operators / Artifacts 拆分 |
| `deploy/helm/axisml-infra/` | 基础设施 Helm chart，承载 gateway、RustFS、zot、GPU Operator、Koordinator、监控等依赖 |
| `deploy/helm/axisml-system/` | 控制平面 Helm chart，承载 Platform、Compute、Artifacts、Operators、CRDs 与数据库依赖 |
| `docs/system_design/` | 系统设计文档 |
| `docs/development/` | 本地开发与环境说明 |
| `scripts/` | 本地集群、安装、调试等脚本 |

## 8. 关键设计决策

| 决策项 | 决策 | 理由 |
| --- | --- | --- |
| 计算任务抽象 | 通过 CRD（MLJob / MLService / Tenant）抽象 | 与 Kubernetes 原生集成，声明式管理，框架无关 |
| backend 扩展机制 | operator 内部按 `spec.backend.{name, engine}` 路由 | 用户和 Compute 面向稳定 CRD，底层执行后端可按需接入 native / Kubeflow Trainer / KServe / custom |
| 配额模型 | Compute Quota 1:1 映射上游 `scheduling.sigs.k8s.io/v1alpha1` `ElasticQuota` CR（namespace-scoped，落在租户 ns），采用纯 `min` / `max` 二维模型 | 直接对齐 sigs.k8s.io scheduler-plugins ElasticQuota 原生语义（min 即不可被抢占下界，max 即硬上限），namespace-scoped 表达租户边界，避免自研配额仲裁；不引入 Koordinator 私有 annotation，让 PG schema、API 与 K8s 原生 CR 字段一一对应 |
| 调度与配额收编 | 所有 AxisML workload Pod 强制走 `schedulerName: koord-scheduler` 并携带 `quota.scheduling.koordinator.sh/name` label | 任何 job / service 都消耗对应 quota，无"绕过 quota 的调度路径"；KServe 派生 Pod 通过 `spec.predictor.schedulerName` + `spec.predictor.labels` 透传（KServe v1beta1 把 `corev1.PodSpec` 与 `ComponentExtensionSpec` 内联到 `PredictorSpec`，因此两者都是 `spec.predictor` 的直接字段），依赖最新 KServe |
| 制品元数据存储 | PostgreSQL | 关系型数据，支持事务，生态成熟 |
| 制品文件存储 | model / image 走 zot，dataset 走 RustFS | OCI 适合不可变模型 / 镜像引用；S3 适合目录型、多文件数据集 |
| 部署分层 | `axisml-infra` / `axisml-system` 两个 Helm chart | 基础设施与控制平面发版节奏、回滚粒度不同；infra 可被多套 system 复用 |
| 后端语言 | Go | 云原生生态契合，Operator 开发原生支持 |
| 前端框架 | TypeScript + React | 社区生态成熟，组件丰富 |
| 系统管理归属 | 租户、资源池、资源单元、资源配额管理放在 AxisML Compute 中 | 与任务 / 服务提交链路强耦合，便于配额校验、资源注入和状态回流 |
| 认证鉴权 | Platform 统一入口；具体 IdP、角色模型和策略 TBD | 外部入口收敛到 Platform，Compute / Artifacts 只接受内部调用并信任身份透传 |
