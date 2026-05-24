# AxisML 概要设计

## 1. 概述

**AxisML** 是一个面向机器学习工作负载的一站式平台，覆盖模型开发、训练、制品管理、在线推理与运维管理。本文档是系统设计的高层导航，描述平台边界、核心概念、主要组件与关键设计取舍；字段级 schema、状态机和具体实现契约以各详细设计文档为准。

## 2. 核心概念

平台采用 **两层业务模型 + 一层视图**：

- **管理员域**：租户（Tenant）、配额（Quota）、Namespace 派生资源由 [Cluster Manager](components/cluster-manager.md) 写入、[tenant-operator](components/tenant-operator.md) 在 Kubernetes 上落地。
- **业务域**：计算负载（Job / Service）与制品（Artifact）由 [Compute](components/compute.md) 与 [Artifacts](components/artifacts.md) 按 **裸 namespace 字符串** 分区承载，**不感知"租户"概念**。
- **视图层**：[Platform](components/platform.md) 自己持有"用户 → 租户视图 → (compute_ns, artifacts_ns)"映射，把上述两层拼成用户看得到的工作区视图。

### 2.1 概念速查

| 术语 | 英文 | 对应对象 | 详细设计 |
| --- | --- | --- | --- |
| 租户 | Tenant | 集群级 `Tenant` CR + PG `tenants` 行 | [cluster-manager #3](components/cluster-manager.md#3-核心模型) / [tenant-operator #3](components/tenant-operator.md#3-核心模型) |
| 资源池 | ResourcePool | Compute 元数据对象（按用途 / 硬件 / 来源划分） | [compute #3](components/compute.md#3-核心模型) |
| 资源单元 | ResourceUnit | ResourcePool 内的资源规格模板 | [compute #3](components/compute.md#3-核心模型) |
| 资源配额 | Quota | `Tenant.spec.quotas[]` 内联项 → namespace-scoped `ElasticQuota` CR | [cluster-manager #3](components/cluster-manager.md#3-核心模型) / [tenant-operator #4](components/tenant-operator.md#4-核心功能) |
| 计算负载 | Compute Workload | Job / Service 的概念伞 | [compute #3](components/compute.md#3-核心模型) |
| 任务 | Job | `MLJob` CRD | [compute-operator #3](components/compute-operator.md#3-核心模型) |
| 服务 | Service | `MLService` CRD | [compute-operator #3](components/compute-operator.md#3-核心模型) |
| 工作区 | Workspace | Compute `services` 表中 `kind='workspace'`（底层复用 `MLService(native, deployment)`） | [compute #3](components/compute.md#3-核心模型) |
| 制品 | Artifact | `(namespace, kind, name, version)` 四元组寻址 | [artifacts #3](components/artifacts.md#3-核心模型) |

### 2.2 关键不变量

- **Compute / Artifacts 不感知 Tenant**：仅按裸 namespace 分区，租户语义收敛在 Platform 视图层。
- **PG 为权威，CR 为派生**：Cluster Manager 以 `tenants` 表为唯一权威，Tenant CR 由内部 reconciler 派生并持续对账。
- **所有 AxisML Pod 走 koord-scheduler**：任何 backend handler 渲染出的 Pod 必须设置 `schedulerName: koord-scheduler` 并携带 `quota.scheduling.koordinator.sh/name` label —— 不存在"绕过配额"的调度路径。
- **Operator 之间不互相感知**：tenant-operator 不看 MLJob / MLService；compute-operator 不看 Tenant / ElasticQuota（仅透传 quota 名）。
- **外部入口只在 Platform**：Cluster Manager / Compute / Artifacts 不暴露到集群外，仅接受 Platform 内部调用并信任 `X-Axisml-User` 身份透传。

## 3. 功能矩阵

本表表示系统设计覆盖状态，不等同于代码实现完成度。

| 功能分类 | 功能项 | 设计状态 |
| --- | --- | :---: |
| 训练 & 推理 | 模型定制 | TBD |
| | 工作区 | ✅ |
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

- 外部流量经 Envoy Gateway 进入 Platform；下层服务仅接受 Platform 内部调用。
- Platform → Cluster Manager（租户 / 配额）、Compute（任务 / 服务 / 资源池 / 资源单元）、Artifacts（模型 / 镜像 / 数据集），均以 namespace 为分区入参。
- 管理员域闭环：Cluster Manager 写 PG `tenants` → reconciler patch Tenant CR → tenant-operator 落地 Namespace / ElasticQuota / 初始化资源。
- 业务域闭环：Compute 写 PG 业务表 + MLJob / MLService CR → compute-operator 按 `spec.backend.{name, engine}` 路由到 backend handler → 渲染 K8s 与第三方 CR。
- 制品域：Artifacts 元数据走 PG；模型 / 镜像走 zot（OCI），数据集走 RustFS（S3），上传下载由消费方直连存储，Artifacts 不代理大文件 bytes。

## 5. 设计文档导航

| 文档 | 目的 |
| --- | --- |
| [overview.md](overview.md) | 本文，系统级导航与高层模型 |
| [components/platform.md](components/platform.md) | Platform（用户入口、业务编排、租户视图层） |
| [components/cluster-manager.md](components/cluster-manager.md) | Cluster Manager（管理员域 REST + Tenant CR 派生 reconciler） |
| [components/compute.md](components/compute.md) | Compute（Job / Service / ResourcePool / ResourceUnit 业务服务） |
| [components/artifacts.md](components/artifacts.md) | Artifacts（制品元数据 + 存储后端分离） |
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
| **Cluster Manager** | 管理员域 REST 入口，租户与配额的 PG 权威源 | PG `tenants` 表 + Tenant CR（派生） | [cluster-manager.md](components/cluster-manager.md) |
| **Compute** | 业务域计算服务，管理 Job / Service 与资源元数据 | ResourcePool / ResourceUnit / MLJob / MLService（namespace 分区） | [compute.md](components/compute.md) |
| **Artifacts** | 业务域制品服务，元数据 / 存储分离 | Artifact 四元组 `(namespace, kind, name, version)` | [artifacts.md](components/artifacts.md) |
| **tenant-operator** | 把 Tenant CR 翻译为 Namespace / ElasticQuota / 初始化资源 | Tenant CR（cluster-scoped） | [tenant-operator.md](components/tenant-operator.md) |
| **compute-operator** | 把 MLJob / MLService 路由到 backend handler 渲染 K8s 与第三方 CR | MLJob / MLService + backend handler registry | [compute-operator.md](components/compute-operator.md) |

各组件的定位、架构、模型与接口契约请进入对应文档 §1–§6 查阅，本文不再展开。

## 7. 部署架构

详见 [deployment.md](deployment.md)。AxisML 通过 `axisml-infra`（第三方基础设施）与 `axisml-system`（控制平面 + 元数据数据库）两个 Helm chart 分层部署，安装顺序 infra → system。

## 8. 项目结构

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

### 8.1 目录说明

| 目录 | 说明 |
| --- | --- |
| `components/` | 各组件代码目录（Platform / Cluster Manager / Compute / Tenant Operator / Compute Operator / Artifacts） |
| `deploy/helm/axisml-infra/` | 基础设施 Helm chart |
| `deploy/helm/axisml-system/` | 控制平面 Helm chart，承载 Platform、Cluster Manager、Compute、Artifacts、tenant-operator、compute-operator、CRDs 与数据库依赖 |
| `docs/system_design/` | 系统设计文档 |
| `docs/development/` | 本地开发与环境说明 |
| `scripts/` | 本地集群、安装、调试等脚本 |

## 9. 关键设计决策

| 决策项 | 决策 | 理由 |
| --- | --- | --- |
| 计算任务抽象 | 通过 CRD（MLJob / MLService / Tenant）抽象 | 与 Kubernetes 原生集成，声明式管理，框架无关 |
| 控制平面拆分 | tenant-operator + compute-operator 两个独立二进制 | 管理员域与业务域按变更频率与权限边界分离 |
| 管理员域入口 | 独立 Cluster Manager 服务，PG 为权威 + reconciler 派生 Tenant CR | 权威收敛到单一 PG 表，避免多组件重复维护元数据；详见 [cluster-manager #5](components/cluster-manager.md#5-关键机制) |
| Compute / Artifacts 分区模型 | 仅按裸 namespace 字符串分区，由 Platform 维护租户语义 | 让基础服务保持单一职责，Platform 灵活映射视图层 |
| backend 扩展机制 | compute-operator 内部按 `spec.backend.{name, engine}` 路由 | 用户面向稳定 CRD，底层后端可按需接入 native / Kubeflow / KServe / custom；详见 [compute-operator #4](components/compute-operator.md#4-核心功能) |
| 配额模型 | Quota 内联在 `Tenant.spec.quotas[]` → 1:1 映射上游 `ElasticQuota` CR（namespace-scoped），纯 `min` / `max` 二维 | 对齐 sigs.k8s.io scheduler-plugins 原生语义，避免独立 Quota CRD |
| 调度与配额收编 | 所有 Pod 强制 `schedulerName: koord-scheduler` + ElasticQuota label | 保证不存在绕过配额的调度路径 |
| 制品抽象 | Artifact 直接以 `(namespace, kind, name, version)` 四元组寻址，无"仓库"两级空间 | 与 Compute 分区模型对齐，统一为裸 namespace |
| 制品元数据存储 | PostgreSQL | 关系型 + 事务 + 生态成熟 |
| 制品文件存储 | model / image 走 zot，dataset 走 RustFS | OCI 适合不可变模型 / 镜像引用，S3 适合目录型多文件数据集 |
| 部署分层 | `axisml-infra` / `axisml-system` 两个 Helm chart | 基础设施与控制平面发版节奏、回滚粒度不同 |
| 后端语言 | Go | 云原生生态契合，Operator 开发原生支持 |
| 前端框架 | TypeScript + React | 社区生态成熟，组件丰富 |
| 系统管理归属 | 租户与配额归 cluster-manager；资源池、资源单元归 compute | 管理员域与业务域按职责分离，但资源池 / 资源单元与任务链路强耦合 |
| 认证鉴权 | Platform 统一入口；内置用户体系 + RBAC 三档；`IdentityProvider` 接口预留 OIDC | 外部入口收敛到 Platform，下层信任 `X-Axisml-User` 透传；详见 [auth.md](auth.md) |
