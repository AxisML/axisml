# AxisML 概要设计

## 1. 概述

**AxisML** 是一个面向机器学习工作负载的一站式平台，覆盖模型开发、训练、制品管理、在线推理与运维管理。本文档是系统设计的高层导航，描述平台边界、核心概念、主要组件与关键设计取舍；字段级 schema、状态机和具体实现契约以各详细设计文档为准。

## 2. 核心概念

平台采用 **两层业务模型 + 一层定义 / 视图**：

- **集群词汇层**：ResourcePool CRD（内嵌 `units[]` 数组）与集群级 `Tenant` CR 由 [Cluster Manager](components/cluster-manager.md) 通过 REST 维护——admin 视角的"K8s 写抽象"层，cluster-manager 调 K8s API 写 CR，并据 ResourceUnit 规格把配额「资源单元 × 数量」折算进 `Tenant.spec.quotas[]`；compute 通过 Informer 直读 ResourcePool CR 完成展开，[tenant-operator](components/tenant-operator.md) 直读 Tenant CR 落地 Namespace / ElasticQuota / 初始化资源。
- **工作负载层**：运行（Run = `MLRun`）/ Service 与制品版本（Artifact version）由 [compute-service](components/compute-service.md) 与 [artifact-hub](components/artifact-hub.md) 承载；二者以 namespace（= 租户 `identifier`）为分区写下游 CR，不持有 Tenant 权威。
- **定义 / 视图层**：[Platform](components/platform.md) 持有租户（Tenant）的持久记录与生命周期权威（自有 `tenants` 表，`identifier` 唯一标识），以及 Job / 实验（Experiment）/ Model / Image 的 name 级**定义**和"用户 → 租户视图 → namespace"映射；租户的 K8s 物化经 cluster-manager REST 下发 Tenant CR，Platform 不直接操作任何 CR。运行（Run）与制品版本仍在下游，经 `axisml.io/{job,experiment}` label 与 `(kind, name)` 实时关联。调 compute 触发运行时只透传 `(poolName, unitName)` 名字对，pool/unit 展开由 compute 自完成。

### 2.1 概念速查

| 术语 | 英文 | 对应对象 | 详细设计 |
| --- | --- | --- | --- |
| 租户 | Tenant | 集群级 `Tenant` CR（cluster-manager 写）+ Platform PG `tenants` 行（`identifier` 标识） | [cluster-manager #3](components/cluster-manager.md#3-核心模型) / [platform #3](components/platform.md#3-核心模型) / [tenant-operator #3](components/tenant-operator.md#3-核心模型) |
| 资源池 | ResourcePool | `ResourcePool` CRD（cluster-scoped）；`spec.units[]` 内嵌 unit 数组 | [cluster-manager #3](components/cluster-manager.md#3-核心模型) |
| 资源单元 | ResourceUnit (unit) | ResourcePool `spec.units[]` 内嵌项, 同 pool 一起生灭 | [cluster-manager #3](components/cluster-manager.md#3-核心模型) |
| 资源配额 | Quota | `Tenant.spec.quotas[]`（按「资源单元 × 数量」，system-admin 设）→ 每 pool 一个 namespace-scoped `ElasticQuota`（`min`/`max` = `Σ unit × quantity`，由 cluster-manager 据 ResourceUnit 规格折算后写入 CR） | [cluster-manager #4](components/cluster-manager.md#4-核心功能) / [tenant-operator #4](components/tenant-operator.md#4-核心功能) |
| 计算负载 | Compute Workload | Run / Service 的概念伞 | [compute-service #3](components/compute-service.md#3-核心模型) |
| 任务（定义） | Job | Platform PG `jobs` 行（可复用模板，name 级定义） | [platform #3.2](components/platform.md#32-定义jobs--experiments--models--images) |
| 运行 | Run | `MLRun` CRD（Job / 实验的一次运行，命名 `<定义>-<n>`）；Platform 经 `axisml.io/{job,experiment}` label 关联 | [compute-operator #3](components/compute-operator.md#3-核心模型) |
| 实验（定义） | Experiment | Platform PG `experiments` 行（训练特化模板，name 级定义）；Run = `MLRun`，经 `axisml.io/experiment` label 关联 | [platform #3.2](components/platform.md#32-定义jobs--experiments--models--images) |
| 服务 | Service | `MLService` CRD | [compute-operator #3](components/compute-operator.md#3-核心模型) |
| 工作区 | Workspace | Compute `mlservices` 表中 `kind='workspace'`（底层复用 `MLService(native, deployment)`） | [compute-service #3](components/compute-service.md#3-核心模型) |
| TensorBoard | TensorBoard | Compute `mlservices` 表中 `kind='tensorboard'`（实验指标查看的按需临时实例，底层复用 `MLService(native, deployment)`） | [compute-service #3](components/compute-service.md#3-核心模型) |
| 流量策略 | Traffic Policy | namespace-scoped `MLTrafficPolicy` CR + PG `traffic_policies` 行（一个稳定入口加权分发到多个在线服务） | [compute-service #4.3](components/compute-service.md#43-流量策略mltrafficpolicy) / [compute-operator #4.3](components/compute-operator.md#43-mltrafficpolicy-controller) |
| 制品（定义） | Model / Image | Platform PG `models` / `images` 行（name 级定义） | [platform #3.2](components/platform.md#32-定义jobs--experiments--models--images) |
| 制品版本 | Artifact version | `(namespace, kind, name, version)` 四元组寻址；`namespace` = 租户名 | [artifacts #3](components/artifact-hub.md#3-核心模型) |

### 2.2 关键不变量

- **namespace = 租户 `identifier`（单一规范名）**：`identifier`（Platform `tenants.identifier`）同时是 Tenant CR 名、K8s namespace 名、以及 compute / artifacts 的 `namespace` 分区字符串——全程一个字符串，无任何跨服务名字解析；compute / artifacts 直接以该字符串为分区键，不 join 任何租户表。
- **Tenant CR 即权威态**：cluster-manager 以 REST 直接读写 etcd 上的 `Tenant` / `ResourcePool` CR（无 PG 派生、无 reconciler 对账）；Platform 自有 `tenants` 表持有租户的生命周期意图（展示元数据 / 暂停 / 软删 / 保留期），经 cluster-manager REST 物化 / 回收 Tenant CR。
- **Cluster Manager 是 K8s admin REST 抽象**：把 admin 视角的 K8s 写 / 读（ResourcePool / Tenant CRD CRUD）收敛为 REST，让 Platform 全程不直接调 K8s API；无独立持久化、无 reconciler、无 leader election。
- **所有 AxisML Pod 走 koord-scheduler**：任何 backend handler 渲染出的 Pod 必须设置 `schedulerName: koord-scheduler` 并携带 `quota.scheduling.koordinator.sh/name` label —— 不存在"绕过配额"的调度路径。
- **Operator 之间不互相感知**：tenant-operator 不看 MLRun / MLService；compute-operator 不看 Tenant / ElasticQuota（仅透传 quota 名）。
- **Platform 拥有定义、下游拥有实例**：Job / 实验 / Model / Image 的 name 级定义在 Platform PG；运行（Run = `MLRun`）与制品版本在下游。二者经 `axisml.io/{job,experiment}` label 与 artifacts `(kind, name)` **实时关联**，Platform 不建 run/version 索引表，不缓存 phase / digest 等可变状态。
- **分组维度走 labels**：Job / 实验（Experiment）定义都是任务的正式分组（Run 分别经 `axisml.io/{job,experiment}` 归属）；其余自定义分组走 `labels.axisml.io/<dim>` 落下游 PG，list 端点支持 `?labelSelector=`；compute / artifacts 不感知 Platform 业务概念。
- **外部入口只在 Platform**：Cluster Manager / Compute Service / Artifact Hub 不暴露到集群外，仅接受 Platform 内部调用并信任 `X-Axisml-User` 身份透传。

## 3. 功能矩阵

本表表示系统设计覆盖状态，不等同于代码实现完成度。

| 功能分类 | 功能项 | 设计状态 |
| --- | --- | :---: |
| 训练中心 | 模型定制 | TBD |
| | 工作区 | ✅ |
| | 实验管理 | ✅ |
| | 自定义任务 | ✅ |
| 服务中心 | 在线服务 | ✅ |
| | 流量配置 | ✅ |
| 资产中心 | 模型 | ✅ |
| | 镜像 | ✅ |
| 系统管理 | 租户管理（platform + cluster-manager） | ✅ |
| | 资源池管理（cluster-manager） | ✅ |
| | 资源单元管理（cluster-manager） | ✅ |
| | 资源配额管理（cluster-manager） | ✅ |

图例：`✅` 表示已有对应详细设计；`TBD` 表示概要中保留能力入口，详细设计待补充或待稳定。

> **删除 / 恢复语义**：仅 `Tenant` 支持软删 + restore（365 天 retention 后物理清理）；`Job` / `Experiment`（实验）定义删除——有活跃 Run 则阻止，否则级联软删其全部 Run；`Run` / `Service` / `Workspace` / `TrafficPolicy` / `TensorBoard` 删除即终态，无 restore 路径（重新触发 / 提交 / 拉起即可）；`Model` / `Image` 定义删除直接级联软删其全部版本，`Artifact` 版本软删后 `(namespace, kind, name, version)` 永不复用。详见各组件 §4。

## 4. 整体架构

系统沿职责划分为三层：**Platform 层**（用户面，唯一对外）、**System 层**（控制面，自研领域能力，不对外）、**Infra 层**（第三方基础设施）。三层各自独立部署为一个 Helm chart（见 [deployment.md](deployment.md)）。

```
                                External Users
                                       │
                  外部流量 → Envoy Gateway(Infra 层) → Platform
                                       ▼
┌──────────────────────────  Platform 层（用户面 · 唯一对外）  ──────────────────────┐
│  AxisML Platform — Frontend (React) + Backend (Go)                              │
│  用户入口 / 认证 / 业务编排；持有"租户视图 ↔ (compute_ns, artifacts_ns)"映射        │
│  创建 Job / Service 时仅向 compute 传 pool/unit 名字                              │
└──────┬───────────────────────────┬────────────────────────────┬─────────────────┘
       │                           │                            │  内部调用（信任 X-Axisml-User）
       ▼                           ▼                            ▼
┌──────────────────────  System 层（控制面 · 自研领域能力 · 不对外）  ──────────────────┐
│  ┌─────────────────┐  ┌────────────────────┐  ┌─────────────────────────────┐    │
│  │ Cluster Manager │  │ Compute Service     │  │  Artifact Hub                │    │
│  │ (Go, K8s REST)  │  │ (Go, namespace 分区)│  │  (Go, namespace 分区)        │    │
│  │ ResourcePool +  │  │ Job / Service /     │  │  Artifact 元数据             │    │
│  │ Tenant CR       │  │ Workspace / Traffic │  │  model,image -> zot          │    │
│  │ admin 域 REST   │  │ → MLRun/MLService/..│  │  dataset -> RustFS           │    │
│  └─────────────────┘  └─────────┬──────────┘  └──────────────────────────────┘    │
│  ┌────────────────────┐  ┌──────┴─────────────┐                                   │
│  │ tenant-operator    │  │ compute-operator   │                                   │
│  │ Tenant CR →        │  │ MLRun/MLService →   │  CRDs 随本层一同发布               │
│  │ Namespace/Quota/   │  │ backend handler →   │                                   │
│  │ Secret/CM/SA/RBAC  │  │ K8s 资源            │                                   │
│  └─────────┬──────────┘  └─────────┬──────────┘                                   │
└────────────┼───────────────────────┼─────────────────────────────────────────────┘
             │ 元数据 / 存储          │ schedulerName: koord-scheduler
             ▼                       ▼
┌────────────────────────  Infra 层（第三方基础设施）  ──────────────────────────────┐
│  Envoy Gateway（对外入口路由 → Platform）   Koordinator（koord-scheduler + Quota）  │
│  PostgreSQL（元数据库）   zot（OCI: model/image）   RustFS（S3: dataset）            │
│  NVIDIA GPU Operator   kube-prometheus-stack                                      │
└──────────────────────────────────────┬───────────────────────────────────────────┘
                                       │ 所有 workload Pod 经 koord-scheduler 落到
                                       ▼
┌──────────────────────────────────────────────────────────────────────────────────┐
│                              Kubernetes Cluster                                   │
│  Workloads / Tenant Resources / ElasticQuota / PodGroup / HTTPRoute               │
└──────────────────────────────────────────────────────────────────────────────────┘
```

核心调用关系：

- 外部流量经 Envoy Gateway 进入 Platform；下层服务仅接受 Platform 内部调用。
- Platform → Cluster Manager（ResourcePool + Tenant admin REST，含配额折算）、Compute（任务 / 服务 / 工作区 / 流量；创建 workload 时仅传 pool/unit 名字, 由 compute 内部 Informer 直读 CR 展开）、Artifacts（模型 / 镜像）；compute / artifacts 以 namespace（= 租户 `identifier`）为分区入参。
- 租户闭环：Platform 写自有 `tenants` 表（生命周期意图）→ 经 cluster-manager REST 创建 / 删除集群级 Tenant CR → tenant-operator 落地 Namespace / ElasticQuota / 初始化资源。负载闭环：Compute 写 PG `mlruns` / `mlservices` / `traffic_policies` → reconciler patch MLRun / MLService / MLTrafficPolicy CR → compute-operator 按 `spec.backend.{name, engine}` 路由 backend handler 渲染 K8s 与第三方 CR（MLTrafficPolicy 派生加权 `HTTPRoute`）。
- 制品域：Artifacts 元数据走 PG；模型 / 镜像走 zot（OCI），数据集走 RustFS（S3），上传下载由消费方直连存储，Artifacts 不代理大文件 bytes。

## 5. 设计文档导航

| 文档 | 目的 |
| --- | --- |
| [overview.md](overview.md) | 本文，系统级导航与高层模型 |
| [components/platform.md](components/platform.md) | Platform（用户入口、业务编排、租户视图层） |
| [components/cluster-manager.md](components/cluster-manager.md) | Cluster Manager（ResourcePool CRD 的 admin REST 入口；含内嵌 `spec.units[]`） |
| [components/compute-service.md](components/compute-service.md) | Compute（Tenant / Quota / Job / Service 业务服务 + 三类 CR reconciler） |
| [components/artifact-hub.md](components/artifact-hub.md) | Artifacts（制品元数据 + 存储后端分离） |
| [components/tenant-operator.md](components/tenant-operator.md) | tenant-operator（Tenant CR → Namespace / ElasticQuota / 初始化资源） |
| [components/compute-operator.md](components/compute-operator.md) | compute-operator（MLRun / MLService → backend handler → K8s 资源） |
| [auth.md](auth.md) | 认证、用户体系、RBAC 与身份透传约定 |
| [database.md](database.md) | PostgreSQL schema 权威定义 |
| [deployment.md](deployment.md) | Helm chart 分层与部署顺序 |
| [monitoring.md](monitoring.md) | Prometheus 指标与监控约定 |
| [infra.md](infra.md) | 第三方基础设施（Envoy / zot / RustFS / Koordinator / GPU Operator / kube-prometheus-stack）契约 |

## 6. 组件职责一览

| 组件 | 一句话职责 | 关键模型 | 详细设计 |
| --- | --- | --- | --- |
| **Platform** | 用户入口与业务编排，持有租户持久记录 + Job / 实验 / Model / Image 定义 + 租户视图层映射 | User / `tenants`（identifier 标识）/ 定义 (jobs/experiments/models/images) / 视图层 (用户 ↔ 租户 ↔ namespace) 映射 | [platform.md](components/platform.md) |
| **Cluster Manager** | admin 域 K8s REST 抽象（ResourcePool + Tenant CRD CRUD、配额折算；扩展端点见组件文档 §9） | ResourcePool CR (含内嵌 `spec.units[]`) + Tenant CR | [cluster-manager.md](components/cluster-manager.md) |
| **Compute** | 业务域计算服务，管理 Run(MLRun) / Service / Workspace / TrafficPolicy 与三类 CR | MLRun + MLService + MLTrafficPolicy（PG mlruns/mlservices/traffic_policies，namespace 分区） | [compute-service.md](components/compute-service.md) |
| **Artifacts** | 业务域制品服务，元数据 / 存储分离 | Artifact 四元组 `(namespace, kind, name, version)` | [artifact-hub.md](components/artifact-hub.md) |
| **tenant-operator** | 把 Tenant CR 翻译为 Namespace / ElasticQuota / 初始化资源 | Tenant CR（cluster-scoped） | [tenant-operator.md](components/tenant-operator.md) |
| **compute-operator** | 把 MLRun / MLService / MLTrafficPolicy 路由到 backend handler 渲染 K8s 与第三方 / 网关 CR | MLRun / MLService / MLTrafficPolicy + backend handler registry | [compute-operator.md](components/compute-operator.md) |

各组件的定位、架构、模型与接口契约请进入对应文档 §1–§6 查阅，本文不展开。

## 7. 部署架构

详见 [deployment.md](deployment.md)。AxisML 按三层各部署为一个 Helm chart：`axisml-infra`（Infra 层：第三方基础设施 + PostgreSQL）、`axisml-system`（System 层：自研控制面 + CRDs）、`axisml-platform`（Platform 层：用户面）。安装顺序 infra → system → platform，卸载反向。

## 8. 项目结构

采用 Monorepo 管理所有组件：

```
axisml/
├── components/
│   ├── platform/                 # AxisML Platform
│   │   ├── backend/              # 后端（Go）
│   │   └── frontend/             # 前端（React）
│   ├── cluster-manager/          # AxisML Cluster Manager（Go）
│   ├── compute-service/          # AxisML Compute Service（Go）
│   ├── tenant-operator/          # AxisML Tenant Operator（Go）
│   ├── compute-operator/         # AxisML Compute Operator（Go）
│   └── artifact-hub/             # AxisML Artifact Hub（Go）
├── deploy/
│   └── helm/
│       ├── axisml-infra/         # Infra 层 Chart：第三方基础设施 + PostgreSQL
│       ├── axisml-system/        # System 层 Chart：自研控制面 + CRDs
│       └── axisml-platform/      # Platform 层 Chart：用户面（前端 + 后端）
├── docs/
│   ├── development/
│   └── system_design/
├── scripts/                      # 本地开发与运维脚本
├── Makefile
└── README.md
```

### 8.1 目录说明

| 目录 | 说明 |
| --- | --- |
| `components/` | 各组件代码目录（Platform / Cluster Manager / Compute / Tenant Operator / Compute Operator / Artifacts） |
| `deploy/helm/axisml-infra/` | Infra 层 Helm chart：第三方基础设施（Envoy / zot / RustFS / Koordinator / GPU Operator / kube-prometheus-stack）+ PostgreSQL |
| `deploy/helm/axisml-system/` | System 层 Helm chart：Cluster Manager、Compute、Artifacts、tenant-operator、compute-operator 与 CRDs |
| `deploy/helm/axisml-platform/` | Platform 层 Helm chart：用户面（Frontend + Backend） |
| `docs/system_design/` | 系统设计文档 |
| `docs/development/` | 本地开发与环境说明 |
| `scripts/` | 本地集群、安装、调试等脚本 |

## 9. 关键设计决策

| 决策项 | 决策 | 理由 |
| --- | --- | --- |
| 计算任务抽象 | 通过 CRD（MLRun / MLService / MLTrafficPolicy / Tenant）抽象 | 与 Kubernetes 原生集成，声明式管理，框架无关 |
| 任务 / 制品定义层 | Platform 自有 Job / 实验 / Model / Image 四张 name 级定义；运行（Run = MLRun）与版本留在下游，经 `axisml.io/{job,experiment}` label 与 `(kind, name)` 实时关联 | 给业务编排一个稳定的"定义 / 模板"抽象，便于后续扩展；下游只管实例，Platform 不缓存可变状态、不建索引表 |
| 控制平面拆分 | tenant-operator + compute-operator 两个独立二进制 | 管理员域与业务域按变更频率与权限边界分离 |
| 租户与配额归属 | cluster-manager 持有 Tenant CR + 配额折算（与 ResourcePool 同为集群级 admin CR）；Platform 持有租户持久记录（`tenants` 表）与生命周期 / 暂停 / 软删权威 | `identifier` 单一规范名贯穿 Tenant CR / namespace / 分区键，无跨服务名字解析；配额折算与 ResourceUnit 词汇共驻 cluster-manager；详见 [cluster-manager #4](components/cluster-manager.md#4-核心功能) |
| Pool/Unit 与租户分离 | ResourcePool CRD 由 cluster-manager 管 (内嵌 units), compute 通过 Informer 直读做展开 | pool/unit 是集群级 admin 词汇，跟租户生命周期解耦；写路径 (cluster-manager → K8s) 与读路径 (compute Informer) 都经 etcd 收敛, 无跨组件调用 |
| Compute / Artifacts 分区模型 | 按租户 `identifier` 作 namespace 分区字符串；`identifier` == K8s namespace == Tenant CR 名，compute / artifacts 均不解析、不 join | 单一规范名，分区清晰且无跨服务名字解析 |
| backend 扩展机制 | compute-operator 内部按 `spec.backend.{name, engine}` 路由 | 用户面向稳定 CRD，底层后端可按需接入 native / Kubeflow / KServe / custom；详见 [compute-operator #4](components/compute-operator.md#4-核心功能) |
| 配额模型 | 配额内联在 Tenant spec：`spec.quotas[]` 按「资源单元 × 数量」（system-admin 设）→ 每 pool 一个 namespace-scoped `ElasticQuota`（`min` / `max` = `Σ unit × quantity`，由 cluster-manager 据 ResourceUnit 规格折算后写入 CR） | 配额以业务可理解的「资源单元 × 数量」表达、由 cluster-manager 折算为 ElasticQuota，避免独立 Quota CRD |
| 调度与配额收编 | 所有 Pod 强制 `schedulerName: koord-scheduler` + ElasticQuota label | 保证不存在绕过配额的调度路径 |
| 制品抽象 | Artifact 直接以 `(namespace, kind, name, version)` 四元组寻址，无"仓库"两级空间 | 与 Compute 分区模型对齐，统一为裸 namespace |
| 制品元数据存储 | PostgreSQL | 关系型 + 事务 + 生态成熟 |
| 制品文件存储 | model / image 走 zot，dataset 走 RustFS | OCI 适合不可变模型 / 镜像引用，S3 适合目录型多文件数据集 |
| 部署分层 | `axisml-infra` / `axisml-system` / `axisml-platform` 三个 Helm chart，对齐 Platform / System / Infra 职责分层 | 用户面、自研控制面、第三方基础设施三者发版节奏、回滚粒度、对外暴露面各不相同；PostgreSQL 作为第三方依赖归 infra |
| 后端语言 | Go | 云原生生态契合，Operator 开发原生支持 |
| 前端框架 | TypeScript + React | 社区生态成熟，组件丰富 |
| 系统管理归属 | 租户的 K8s 物化（Tenant CR）+ 配额折算 + 资源池 + 资源单元归 cluster-manager；租户持久记录 + 生命周期 / 暂停 / 软删归 Platform | 集群级 admin 词汇集中在 cluster-manager；Platform 持有面向用户的租户生命周期，经 cluster-manager REST 物化 |
| 认证鉴权 | Platform 统一入口；内置用户体系 + RBAC 三档（硬编码）；OIDC 后续接入 | 外部入口收敛到 Platform，下层信任 `X-Axisml-User` 透传；详见 [auth.md](auth.md) |
