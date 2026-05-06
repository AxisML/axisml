# AxisML 概要设计

## 1. 概述

**AxisML** 是一个面向机器学习工作负载的一站式平台，覆盖模型开发、训练、制品管理、在线推理与运维管理。本文档是系统设计的高层导航，描述平台边界、核心概念、主要组件与关键设计取舍；字段级 schema、状态机和具体实现契约以各详细设计文档为准。

## 2. 核心概念

平台的业务模型分为两层：**管理员域**（租户、配额、Namespace 派生资源）由 Cluster Manager + tenant-operator 承载；**业务域**（计算负载、制品）由 Compute / Artifacts / compute-operator 按"裸 namespace 分区"的方式承载，不再感知"租户"概念。Platform 作为最上层把"租户视图 ↔ namespace 二元组"的映射收敛于自身。

### 2.1 租户（Tenant）

集群级管理对象。每个租户对应一条 cluster-scoped `Tenant` CR，由 tenant-operator 负责在 Kubernetes 上落地目标 Namespace、Koordinator ElasticQuota、Secret / ConfigMap / ServiceAccount / RBAC 等初始化资源。Tenant 的写入入口是 [Cluster Manager](cluster-manager.md)；它是无状态薄壳，把外部 REST 请求直接翻译为对 Tenant CR 的 K8s API 调用，**不持有任何 PG 业务表**。

Tenant 的目标 Namespace 通过 `spec.namespace.name` 显式声明，多个 Tenant 可以共享同一个 Namespace。平台的隔离边界主要由租户业务模型、per-tenant 资源命名、配额与鉴权策略共同表达，而不是简单等同于 Kubernetes Namespace 的一租户一命名空间隔离。

> **Compute / Artifacts 不感知 Tenant 概念**——它们以裸 namespace 字符串作为分区键。"用户视角的租户"由 [Platform](platform.md) 自身持有视图层映射。

### 2.2 资源池（ResourcePool）

集群资源的物理或逻辑划分，由平台管理员定义，是 AxisML Compute 维护的元数据对象。典型划分维度：

- **用途**：训练池 / 推理池
- **硬件代际**：A100 池 / H100 池
- **来源**：不同云厂商或地域

ResourcePool 通过节点选择条件和容忍配置描述资源边界。ResourceUnit 与 Quota 均挂靠在 ResourcePool 下，同一池内的资源单元只能被该池内的配额消费，跨池资源相互独立。

### 2.3 资源单元（ResourceUnit）

ResourcePool 内预先定义的资源规格模板，是 AxisML Compute 维护的元数据对象。例如 `a100-1x-large` 可表示 1xA100 + 8 vCPU + 32 GiB。用户创建任务或服务时选择一个 ResourceUnit，平台据此注入 `requests` / `limits` 和节点匹配条件。命名规范详见 [compute.md](compute.md)。

### 2.4 资源配额（Quota）

租户在某个 ResourcePool 下的配额承载体。Quota 不是独立 CRD，而是 `Tenant.spec.quotas[]` 的内联子字段；Cluster Manager 通过对 Tenant CR 做 JSON Patch 来增删改 Quota，由 tenant-operator 1:1 渲染为 namespace-scoped Koordinator `ElasticQuota` CR（落在租户 Namespace 下），并把 `status.used` 回流到 `Tenant.status.quotas[].used`。

- 每个 `(tenant, pool)` 默认存在配额 `default`
- 用户可创建其他配额（如 `training`、`inference`、`nlp`）用于业务线 / 团队维度的拆分
- 配额采用上游 sigs.k8s.io scheduler-plugins ElasticQuota 的原生二维模型：`min`（保留份额，不可被抢占下界）、`max`（硬上限）；不引入 Koordinator 私有的 `shared-weight` annotation
- ElasticQuota CR 命名约定：`axisml-<tenant>-<pool>-<quota>`
- 当前版本为扁平结构（无父子层级）；分层配额作为后续演进方向

同一租户在不同 ResourcePool 下的配额结构与额度可以完全不同，互不干扰。

### 2.5 计算负载（Compute Workload）

平台中所有运行时算力承载体的统称，按生命周期分为两类：**任务（Job）** 是一次性 workload，有明确终止态；**服务（Service）** 是长驻 workload，通过扩缩容调节容量。两者都按 Compute 服务请求体里的 `namespace` 字段分区——Compute 不再关心这是哪个租户的工作区。

MLJob / MLService 的底层执行由 compute-operator 按 `spec.backend.{name, engine}` 元组路由到不同 backend handler。默认 backend 为 MLJob `(native, job)` 与 MLService `(native, deployment)`；分布式训练的 gang scheduling 由 `(native, podgroup)` 或 `(kubeflow-trainer, *)` 表达。所有 backend 派生的 Pod 都强制走 koord-scheduler 并消费对应 ElasticQuota；Kubeflow Training Operator、KServe 等作为可选执行后端接入。

### 2.6 任务（Job）

训练任务、分布式训练任务与数据处理任务的统称，对应 `MLJob` CRD，由 compute-operator 的 MLJob controller 负责生命周期管理。

### 2.7 服务（Service）

模型部署后对外提供在线推理的实体，对应 `MLService` CRD，由 compute-operator 的 MLService controller 负责生命周期管理。

### 2.8 制品（Artifact）

模型、数据集、镜像、评估报告等"非运行态"资产的统一抽象。Artifact 在 AxisML Artifacts 中以 `(namespace, kind, name, version)` 四元组寻址，**不再有"仓库（repo）"或"租户私有 / 平台公共"两级空间的概念**——所有 Artifact 平铺在 namespace 分区下，Kind 由编译期 ArtifactHandler registry 扩展。元数据存 PostgreSQL；模型与镜像通过 OCI Distribution 协议存储在 zot；数据集通过 S3 协议存储在 RustFS。

### 2.9 概念速查

| 术语 | 英文名 | 对应对象 |
| --- | --- | --- |
| 租户 | Tenant | `Tenant` CRD（由 cluster-manager 写、tenant-operator 落地）|
| 资源池 | ResourcePool | Compute 元数据对象 |
| 资源单元 | ResourceUnit | Compute 元数据对象 |
| 资源配额 | Quota | `Tenant.spec.quotas[]` 内联项 + Koordinator `ElasticQuota` CR |
| 计算负载 | Compute Workload | Job / Service 的概念伞 |
| 任务 | Job | `MLJob` CRD |
| 服务 | Service | `MLService` CRD |
| 制品 | Artifact | Artifacts 元数据：`(namespace, kind, name, version)` |
| 工作区 | Workspace | Platform 内部概念：`(compute_namespace, artifacts_namespace)` 二元组 |

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
| 系统管理 | 租户管理（cluster-manager） | ✅ |
| | 资源池管理（compute） | ✅ |
| | 资源单元管理（compute） | ✅ |
| | 资源配额管理（cluster-manager） | ✅ |
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
│            持有"租户视图 ↔ (compute_ns, artifacts_ns)" 映射                  │
└──────┬─────────────────────┬────────────────────────┬────────────────────────┘
       │                     │                        │
       ▼                     ▼                        ▼
┌─────────────────┐  ┌────────────────────┐  ┌─────────────────────────────┐
│ Cluster Manager │  │  AxisML Compute    │  │     AxisML Artifacts         │
│ (Go, 无状态)    │  │  (Go, namespace 分区)│  │  (Go, namespace 分区)        │
│ Tenant / Quota  │  │  ResourcePool /     │  │  Artifact 元数据             │
│ → Tenant CR     │  │  ResourceUnit /     │  │  model,image -> zot          │
│                 │  │  Job / Service      │  │  dataset -> RustFS           │
└────────┬────────┘  └─────────┬──────────┘  └─────────────┬───────────────┘
         │                     │                           │
         ▼                     ▼                           ▼
┌─────────────────┐  ┌────────────────────┐  ┌─────────────────────────────┐
│ tenant-operator │  │  compute-operator  │  │       Metadata DB            │
│ Tenant CR       │  │  MLJob / MLService │  │       PostgreSQL             │
│ → Namespace /   │  │  → backend handler │  └─────────────────────────────┘
│   ElasticQuota /│  │  → K8s 资源        │
│   Secret / CM / │  └─────────┬──────────┘
│   SA / RBAC     │            │
└────────┬────────┘            │
         │                     │
         ▼                     ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│                              Kubernetes Cluster                              │
│  Workloads / Tenant Resources / ElasticQuota / PodGroup / HTTPRoute          │
└──────────────────────────────────────────────────────────────────────────────┘

AxisML Infra 还提供：RustFS、zot、Koordinator、NVIDIA GPU Operator、kube-prometheus-stack。
```

核心调用关系：

- 外部流量经 Envoy Gateway 进入 Platform；Cluster Manager / Compute / Artifacts 不直接暴露到集群外。
- Platform 调用 **Cluster Manager** 管理租户与配额（写 Tenant CR）。
- Platform 调用 **Compute** 在指定 namespace 下管理任务、服务、资源池、资源单元。
- Platform 调用 **Artifacts** 在指定 namespace 下管理模型、镜像、数据集。
- Compute 通过 PostgreSQL 保存业务元数据，通过 CRD 与 compute-operator 协作；Quota 不再由 Compute 维护。
- Cluster Manager 不持有 PG，所有权威都落到 Tenant CR；通过对 `Tenant.spec.quotas[]` 做 JSON Patch 完成 quota 增删改。
- Artifacts 元数据存 PostgreSQL；模型与镜像走 zot；数据集走 RustFS。
- tenant-operator 把 Tenant CR 翻译为 Namespace / ElasticQuota / 初始化资源；compute-operator 把 MLJob / MLService CR 翻译为 K8s 与第三方资源。

## 5. 组件职责

### 5.1 AxisML Platform

平台层，提供用户交互入口与统一业务入口。

- **前端**：基于 TypeScript + React，提供 Web UI。
- **后端**：基于 Go，提供 RESTful API，负责业务逻辑编排，协调 Cluster Manager / Compute / Artifacts 完成具体操作。
- **租户视图持有方**：Platform 自己的 PG 表持有"用户 → 租户视图 → 工作区"的映射；下层服务对此无感知。
- **认证鉴权入口**：Platform 是用户身份、角色与租户访问控制的统一入口；具体 IdP、角色模型和鉴权细节仍待 [platform.md](platform.md) 补充。

> 详细设计见 [AxisML Platform 设计文档](platform.md)

### 5.2 AxisML Cluster Manager

管理员域入口，无状态薄壳，承载以下核心职责：

- **租户管理**：通过 REST API 接收创建 / 更新 / 暂停 / 恢复 / 删除请求，翻译为对 `Tenant` CR 的 K8s API 调用。
- **配额管理**：通过 JSON Patch 修改 `Tenant.spec.quotas[]`，间接驱动 tenant-operator 创建 / 更新 / 删除 ElasticQuota CR。

Cluster Manager **不持有 PG 元数据**，权威完全在 etcd。所有 GET 请求都直接读 K8s API。

> 详细设计见 [AxisML Cluster Manager 设计文档](cluster-manager.md)

### 5.3 AxisML Compute

计算服务层，基于 Go 开发，仅接受 Platform 的内部调用，承载以下核心职责：

- **计算负载管理**：维护 Job / Service 业务元数据，创建或更新 `MLJob` / `MLService` CR，并通过 Informer 消费 operator 回流状态。
- **资源池管理**：维护 ResourcePool 的节点选择、容忍配置与底层集群映射关系。
- **资源单元管理**：维护 ResourcePool 内的资源规格模板，并在提交 Job / Service 时注入资源请求与节点匹配条件。

Compute 按请求体里的 `namespace` 字段分区——它不再持有"租户"概念，也不维护 quota PG 表。配额由 cluster-manager 写入、tenant-operator 落地；compute 只在 CR `spec.scheduling.quota` 字段透传 ElasticQuota CR 名。

Compute 不直接创建 Pod、Deployment、PodGroup 等运行时资源；这些资源由 compute-operator 或底层 controller 根据 CR 声明生成。

> 详细设计见 [AxisML Compute 设计文档](compute.md)

### 5.4 AxisML Operators

控制平面拆为两个独立的 Kubernetes operator 二进制：

| Operator | 承载 Controller | 职责 |
| --- | --- | --- |
| [tenant-operator](tenant-operator.md) | Tenant | 把 `Tenant` CR 翻译为 Namespace / Koordinator ElasticQuota / Secret / ConfigMap / ServiceAccount / RBAC 等初始化资源 |
| [compute-operator](compute-operator.md) | MLJob、MLService | 按 `spec.backend.{name, engine}` 元组路由到不同 Handler，渲染 K8s 与第三方 CR（Job / Pod / PodGroup / Deployment / HTTPRoute / KServe `InferenceService` 等） |

两个 operator 互不依赖：tenant-operator 不感知 MLJob / MLService；compute-operator 不感知 Tenant CR 与 ElasticQuota（仅在 `spec.scheduling.quota` 字段中透传 ElasticQuota CR 名）。

MLJob 与 MLService controller 内部按 `spec.backend.{name, engine}` 二级元组路由到不同 Handler：

| Backend | 适用 CRD | engine 示例 | 说明 |
| --- | --- | --- | --- |
| `native` | MLJob、MLService | MLJob: `job` / `podgroup`；MLService: `deployment` / `statefulset` | 直接使用 Kubernetes 原生工作负载与 sigs.k8s.io scheduler-plugins `PodGroup` |
| `kubeflow-trainer` | MLJob | `pytorchjob` / `tfjob` / `mpijob` / ... | 委托给 Kubeflow Training Operator |
| `kserve` | MLService | `inference` / `llminference` | 对应 KServe `InferenceService` 与 LLM 原生服务 |
| `custom` | MLJob、MLService | 任意 | 用户自定义后端，通过 `backend.config` 描述目标 GVK 与字段映射 |

默认值：MLJob `(native, job)`、MLService `(native, deployment)`。**所有 backend 派生的 Pod 都必须设置 `schedulerName: koord-scheduler` 并携带 ElasticQuota label**（详见 [infra.md](infra.md)）。

### 5.5 AxisML Artifacts

制品管理服务，基于 Go 开发，负责平台中主要制品类型的元数据管理、引用解析与存储凭证签发：

- **模型管理**：模型版本、元数据、不可变引用与存储 URI。
- **镜像管理**：训练 / 推理镜像的注册、版本管理与引用解析。
- **数据集管理**：数据集元数据、存储位置、版本与访问凭证。

Artifacts 按请求体里的 `namespace` 字段分区——同 Compute 一样，不感知"租户"或"仓库"两级空间的概念。Artifact 直接以 `(namespace, kind, name, version)` 四元组寻址。

Artifacts 采用元数据服务 / 存储后端分离模式：元数据存储于 PostgreSQL；模型与镜像通过 OCI Distribution 协议存储在 zot；数据集通过 S3 协议存储在 RustFS。上传下载由 CLI 或消费方直连存储后端，Artifacts 不代理大文件 bytes。

> 详细设计见 [AxisML Artifacts 设计文档](artifacts.md)

### 5.6 AxisML Infra

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
│   ├── AxisML Platform           (Deployment + Service)
│   ├── AxisML Cluster Manager    (Deployment + Service)
│   ├── AxisML Compute            (Deployment + Service)
│   ├── AxisML Artifacts          (Deployment + Service)
│   ├── tenant-operator           (Deployment)
│   ├── compute-operator          (Deployment)
│   └── PostgreSQL / externalDatabase
└── tenant namespaces
    └── Tenant resources / workloads / routes / secrets / ElasticQuota
```

安装顺序为先 `axisml-infra`、后 `axisml-system`；卸载顺序相反。PostgreSQL 随 `axisml-system` 管理，也可通过 `externalDatabase` 对接外部实例。

## 7. 项目结构

采用 Monorepo 管理所有组件：

```
axisml/
├── components/
│   ├── platform/                 # AxisML Platform
│   │   ├── backend/              # 后端（Go）
│   │   └── frontend/             # 前端（React）
│   ├── cluster-manager/          # AxisML Cluster Manager（Go）
│   ├── compute/                  # AxisML Compute（Go）
│   ├── tenant-operator/          # AxisML Tenant Operator（Go）
│   ├── compute-operator/         # AxisML Compute Operator（Go）
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
| `components/` | 各组件代码目录（Platform / Cluster Manager / Compute / Tenant Operator / Compute Operator / Artifacts） |
| `deploy/helm/axisml-infra/` | 基础设施 Helm chart |
| `deploy/helm/axisml-system/` | 控制平面 Helm chart，承载 Platform、Cluster Manager、Compute、Artifacts、tenant-operator、compute-operator、CRDs 与数据库依赖 |
| `docs/system_design/` | 系统设计文档 |
| `docs/development/` | 本地开发与环境说明 |
| `scripts/` | 本地集群、安装、调试等脚本 |

## 8. 关键设计决策

| 决策项 | 决策 | 理由 |
| --- | --- | --- |
| 计算任务抽象 | 通过 CRD（MLJob / MLService / Tenant）抽象 | 与 Kubernetes 原生集成，声明式管理，框架无关 |
| 控制平面拆分 | tenant-operator + compute-operator 两个独立二进制 | 管理员域（租户、配额、Namespace 派生资源）与业务域（计算负载）按变更频率与权限边界分离；任一 operator 演进或重启不影响另一域 |
| 管理员域入口 | 独立的 Cluster Manager 服务，无状态薄壳 | 把"创建租户 / 配额"的入口与"计算服务"解耦；权威完全收敛到 Tenant CR，避免在多个组件之间维护重复元数据 |
| Compute / Artifacts 去租户化 | 仅按裸 namespace 字符串分区，由 Platform 维护租户语义 | 让基础服务保持单一职责；Platform 可灵活映射"租户视图 ↔ (compute_ns, artifacts_ns)" |
| backend 扩展机制 | compute-operator 内部按 `spec.backend.{name, engine}` 路由 | 用户和 Compute 面向稳定 CRD，底层执行后端可按需接入 native / Kubeflow Trainer / KServe / custom |
| 配额模型 | Quota 内联在 `Tenant.spec.quotas[]` → 1:1 映射上游 `scheduling.sigs.k8s.io/v1alpha1` `ElasticQuota` CR（namespace-scoped），采用纯 `min` / `max` 二维模型 | 直接对齐 sigs.k8s.io scheduler-plugins ElasticQuota 原生语义；避免引入独立 Quota CRD 让管理面更复杂 |
| 调度与配额收编 | 所有 AxisML workload Pod 强制走 `schedulerName: koord-scheduler` 并携带 `quota.scheduling.koordinator.sh/name` label | 任何 job / service 都消耗对应 quota，无"绕过 quota 的调度路径" |
| 制品抽象 | 移除"仓库（repo）"两级空间；Artifact 直接以 `(namespace, kind, name, version)` 四元组寻址 | 让 Artifacts 与 Compute 在分区模型上对齐，统一为裸 namespace |
| 制品元数据存储 | PostgreSQL | 关系型数据，支持事务，生态成熟 |
| 制品文件存储 | model / image 走 zot，dataset 走 RustFS | OCI 适合不可变模型 / 镜像引用；S3 适合目录型、多文件数据集 |
| 部署分层 | `axisml-infra` / `axisml-system` 两个 Helm chart | 基础设施与控制平面发版节奏、回滚粒度不同 |
| 后端语言 | Go | 云原生生态契合，Operator 开发原生支持 |
| 前端框架 | TypeScript + React | 社区生态成熟，组件丰富 |
| 系统管理归属 | 租户与配额归 cluster-manager；资源池、资源单元归 compute | 管理员域与业务域按职责分离，但资源池 / 资源单元与任务提交链路强耦合，仍随 compute 一同维护 |
| 认证鉴权 | Platform 统一入口；具体 IdP、角色模型和策略 TBD | 外部入口收敛到 Platform，Cluster Manager / Compute / Artifacts 只接受内部调用并信任身份透传 |
