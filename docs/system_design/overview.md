# AxisML 概要设计

## 1. 概述

**AxisML** 是一个面向机器学习工作负载的一站式平台，覆盖模型开发、训练、制品管理、在线推理与运维管理。本文档是系统设计的高层导航，描述平台边界、核心概念、主要组件与关键设计取舍；字段级 schema、状态机和具体实现契约以各详细设计文档为准。

## 2. 核心概念

平台采用 **两层业务模型 + 一层视图**：

- **集群词汇层**：ResourcePool CRD（内嵌 `units[]` 数组）由 [Cluster Manager](components/cluster-manager.md) 通过 REST 维护——admin 视角的"K8s 写抽象"层，cluster-manager 调 K8s API 写 CR；compute 通过 Informer 直读 CR 完成展开。
- **租户与工作负载层**：租户（Tenant）、配额（Quota）、计算负载（Job / Service）与制品（Artifact）由 [compute-service](components/compute-service.md) 与 [artifact-hub](components/artifact-hub.md) 承载；compute 持有 Tenant 权威并下发 Tenant CR 给 [tenant-operator](components/tenant-operator.md) 在 Kubernetes 上落地。
- **视图层**：[Platform](components/platform.md) 自己持有"用户 → 租户视图 → (compute_ns, artifacts_ns)"映射，把上述两层拼成用户看得到的工作区视图；调 compute 创建任务时只透传 `(poolName, unitName)` 名字对，pool/unit 展开由 compute 自完成。

### 2.1 概念速查

| 术语 | 英文 | 对应对象 | 详细设计 |
| --- | --- | --- | --- |
| 租户 | Tenant | 集群级 `Tenant` CR + PG `tenants` 行 | [compute-service #3](components/compute-service.md#3-核心模型) / [tenant-operator #3](components/tenant-operator.md#3-核心模型) |
| 资源池 | ResourcePool | `ResourcePool` CRD（cluster-scoped）；`spec.units[]` 内嵌 unit 数组 | [cluster-manager #3](components/cluster-manager.md#3-核心模型) |
| 资源单元 | ResourceUnit (unit) | ResourcePool `spec.units[]` 内嵌项, 同 pool 一起生灭 | [cluster-manager #3](components/cluster-manager.md#3-核心模型) |
| 资源配额 | Quota | `Tenant.spec.quotas[]` 内联项 → namespace-scoped `ElasticQuota` CR | [compute-service #3](components/compute-service.md#3-核心模型) / [tenant-operator #4](components/tenant-operator.md#4-核心功能) |
| 计算负载 | Compute Workload | Job / Service 的概念伞 | [compute-service #3](components/compute-service.md#3-核心模型) |
| 任务 | Job | `MLJob` CRD | [compute-operator #3](components/compute-operator.md#3-核心模型) |
| 服务 | Service | `MLService` CRD | [compute-operator #3](components/compute-operator.md#3-核心模型) |
| 工作区 | Workspace | Compute `services` 表中 `kind='workspace'`（底层复用 `MLService(native, deployment)`） | [compute-service #3](components/compute-service.md#3-核心模型) |
| 流量策略 | Traffic Policy | namespace-scoped `MLTrafficPolicy` CR + PG `traffic_policies` 行（一个稳定入口加权分发到多个在线服务） | [compute-service #4.5](components/compute-service.md#45-流量策略mltrafficpolicy) / [compute-operator #4.3](components/compute-operator.md#43-mltrafficpolicy-controller) |
| 制品 | Artifact | `(namespace, kind, name, version)` 四元组寻址；`namespace` = 租户名 | [artifacts #3](components/artifact-hub.md#3-核心模型) |

### 2.2 关键不变量

- **namespace = 租户标识符**：compute / artifacts 的 `namespace` 字段是 tenant 名（= `tenants.name`）；compute 内部 join 自己的 `tenants` 表得到 K8s namespace 用于 CR 下发；artifacts 不解析。
- **PG 为权威，CR 为派生**：compute 以 `tenants` 表为唯一权威，Tenant CR 由内部 reconciler 派生并持续对账。
- **Cluster Manager 是 K8s admin REST 抽象**：把 admin 视角的 K8s 写 / 读（ResourcePool CRD CRUD）收敛为 REST，让 Platform 全程不直接调 K8s API；无独立持久化、无 reconciler、无 leader election。
- **所有 AxisML Pod 走 koord-scheduler**：任何 backend handler 渲染出的 Pod 必须设置 `schedulerName: koord-scheduler` 并携带 `quota.scheduling.koordinator.sh/name` label —— 不存在"绕过配额"的调度路径。
- **Operator 之间不互相感知**：tenant-operator 不看 MLJob / MLService；compute-operator 不看 Tenant / ElasticQuota（仅透传 quota 名）。
- **分组维度走 labels**：project / experiment 等用户分组通过 `labels.axisml.io/<dim>` 落 PG，list 端点支持 `?labelSelector=`；compute / artifacts 不感知 Platform 业务概念。
- **外部入口只在 Platform**：Cluster Manager / Compute Service / Artifact Hub 不暴露到集群外，仅接受 Platform 内部调用并信任 `X-Axisml-User` 身份透传。

## 3. 功能矩阵

本表表示系统设计覆盖状态，不等同于代码实现完成度。

| 功能分类 | 功能项 | 设计状态 |
| --- | --- | :---: |
| 训练中心 | 模型定制 | TBD |
| | 工作区 | ✅ |
| | 自定义任务 | ✅ |
| 服务中心 | 在线服务 | ✅ |
| | 流量配置 | ✅ |
| 资产中心 | 模型 | ✅ |
| | 镜像 | ✅ |
| | 数据集 | ✅ |
| 系统管理 | 租户管理（compute） | ✅ |
| | 资源池管理（cluster-manager） | ✅ |
| | 资源单元管理（cluster-manager） | ✅ |
| | 资源配额管理（compute） | ✅ |

图例：`✅` 表示已有对应详细设计；`TBD` 表示概要中保留能力入口，详细设计待补充或待稳定。

> **删除 / 恢复语义**：仅 `Tenant` 支持软删 + restore（365 天 retention 后物理清理）；`Job` / `Service` / `Workspace` 删除即终态，无 restore 路径（重新提交即可）；`Artifact` 软删后 `(namespace, kind, name, version)` 永不复用。详见各组件 §4。

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
│  │ ResourcePool CR │  │ Tenant / Quota /    │  │  Artifact 元数据             │    │
│  │ (含内嵌 units)  │  │ Job / Service       │  │  model,image -> zot          │    │
│  │ admin 域 REST   │  │ → Tenant/MLJob/...   │  │  dataset -> RustFS           │    │
│  └─────────────────┘  └─────────┬──────────┘  └──────────────────────────────┘    │
│  ┌────────────────────┐  ┌──────┴─────────────┐                                   │
│  │ tenant-operator    │  │ compute-operator   │                                   │
│  │ Tenant CR →        │  │ MLJob/MLService →   │  CRDs 随本层一同发布               │
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
- Platform → Cluster Manager（ResourcePool admin REST）、Compute（租户 / 配额 / 任务 / 服务；创建 workload 时仅传 pool/unit 名字, 由 compute 内部 Informer 直读 CR 展开）、Artifacts（模型 / 镜像 / 数据集）；compute / artifacts 以 namespace（tenant 名）为分区入参。
- 租户与负载闭环：Compute 写 PG `tenants` / `jobs` / `services` / `traffic_policies` → reconciler patch Tenant / MLJob / MLService / MLTrafficPolicy CR → tenant-operator 落地 Namespace / ElasticQuota / 初始化资源；compute-operator 按 `spec.backend.{name, engine}` 路由 backend handler 渲染 K8s 与第三方 CR（MLTrafficPolicy 派生加权 `HTTPRoute`）。
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
| [components/compute-operator.md](components/compute-operator.md) | compute-operator（MLJob / MLService → backend handler → K8s 资源） |
| [auth.md](auth.md) | 认证、用户体系、RBAC 与身份透传约定 |
| [database.md](database.md) | PostgreSQL schema 权威定义 |
| [deployment.md](deployment.md) | Helm chart 分层与部署顺序 |
| [monitoring.md](monitoring.md) | Prometheus 指标与监控约定 |
| [infra.md](infra.md) | 第三方基础设施（Envoy / zot / RustFS / Koordinator / GPU Operator / kube-prometheus-stack）契约 |
| [wireframe.md](wireframe.md) | 前端页面级线框图与交互设计 |

## 6. 组件职责一览

| 组件 | 一句话职责 | 关键模型 | 详细设计 |
| --- | --- | --- | --- |
| **Platform** | 用户入口与业务编排，持有租户视图层映射 | User / Org / 视图层 (compute_ns, artifacts_ns) 映射 | [platform.md](components/platform.md) |
| **Cluster Manager** | admin 域 K8s REST 抽象（ResourcePool CRD CRUD；扩展端点见组件文档 §9） | ResourcePool CR (含内嵌 `spec.units[]`) | [cluster-manager.md](components/cluster-manager.md) |
| **Compute** | 业务域计算服务，管理 Tenant / Quota / Job / Service / TrafficPolicy 与四类 CR | Tenant + MLJob + MLService + MLTrafficPolicy（PG tenants/jobs/services/traffic_policies，namespace 分区） | [compute-service.md](components/compute-service.md) |
| **Artifacts** | 业务域制品服务，元数据 / 存储分离 | Artifact 四元组 `(namespace, kind, name, version)` | [artifact-hub.md](components/artifact-hub.md) |
| **tenant-operator** | 把 Tenant CR 翻译为 Namespace / ElasticQuota / 初始化资源 | Tenant CR（cluster-scoped） | [tenant-operator.md](components/tenant-operator.md) |
| **compute-operator** | 把 MLJob / MLService / MLTrafficPolicy 路由到 backend handler 渲染 K8s 与第三方 / 网关 CR | MLJob / MLService / MLTrafficPolicy + backend handler registry | [compute-operator.md](components/compute-operator.md) |

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
| 计算任务抽象 | 通过 CRD（MLJob / MLService / MLTrafficPolicy / Tenant）抽象 | 与 Kubernetes 原生集成，声明式管理，框架无关 |
| 控制平面拆分 | tenant-operator + compute-operator 两个独立二进制 | 管理员域与业务域按变更频率与权限边界分离 |
| 租户与配额归属 | compute 持有 Tenant + Quota 权威，统一与 Job / Service 共驻一个 PG schema | 消除 cluster-manager 与 compute 间的 namespace 解析跨服务调用；compute 自己 join 出 K8s namespace；权威收敛到单一服务；详见 [compute-service #5](components/compute-service.md#5-关键机制) |
| Pool/Unit 与租户分离 | ResourcePool CRD 由 cluster-manager 管 (内嵌 units), compute 通过 Informer 直读做展开 | pool/unit 是集群级 admin 词汇，跟租户生命周期解耦；写路径 (cluster-manager → K8s) 与读路径 (compute Informer) 都经 etcd 收敛, 无跨组件调用 |
| Compute / Artifacts 分区模型 | 按 tenant 名作为 namespace 分区字符串，compute 内部 join 解析 K8s ns，artifacts 不解析 | 既保留分区清晰性，也避免在每次调用中传两套 namespace |
| backend 扩展机制 | compute-operator 内部按 `spec.backend.{name, engine}` 路由 | 用户面向稳定 CRD，底层后端可按需接入 native / Kubeflow / KServe / custom；详见 [compute-operator #4](components/compute-operator.md#4-核心功能) |
| 配额模型 | Quota 内联在 `Tenant.spec.quotas[]` → 1:1 映射上游 `ElasticQuota` CR（namespace-scoped），纯 `min` / `max` 二维 | 对齐 sigs.k8s.io scheduler-plugins 原生语义，避免独立 Quota CRD |
| 调度与配额收编 | 所有 Pod 强制 `schedulerName: koord-scheduler` + ElasticQuota label | 保证不存在绕过配额的调度路径 |
| 制品抽象 | Artifact 直接以 `(namespace, kind, name, version)` 四元组寻址，无"仓库"两级空间 | 与 Compute 分区模型对齐，统一为裸 namespace |
| 制品元数据存储 | PostgreSQL | 关系型 + 事务 + 生态成熟 |
| 制品文件存储 | model / image 走 zot，dataset 走 RustFS | OCI 适合不可变模型 / 镜像引用，S3 适合目录型多文件数据集 |
| 部署分层 | `axisml-infra` / `axisml-system` / `axisml-platform` 三个 Helm chart，对齐 Platform / System / Infra 职责分层 | 用户面、自研控制面、第三方基础设施三者发版节奏、回滚粒度、对外暴露面各不相同；PostgreSQL 作为第三方依赖归 infra |
| 后端语言 | Go | 云原生生态契合，Operator 开发原生支持 |
| 前端框架 | TypeScript + React | 社区生态成熟，组件丰富 |
| 系统管理归属 | 租户与配额归 compute；资源池、资源单元归 cluster-manager | compute 持有"租户 ↔ K8s namespace"映射后可独立解析 namespace；pool/unit 是集群级 admin 词汇，与租户生命周期解耦 |
| 认证鉴权 | Platform 统一入口；内置用户体系 + RBAC 三档（硬编码）；OIDC 后续接入 | 外部入口收敛到 Platform，下层信任 `X-Axisml-User` 透传；详见 [auth.md](auth.md) |
| 全局可见制品 | `axisml-system` 内置租户 + `artifacts.visibility=public` 字段 | 跨租户共享只在"平台基础镜像 / 公共数据集"这种 system-admin 维护的场景出现；不引入"租户 A 主动分享给租户 B" |
