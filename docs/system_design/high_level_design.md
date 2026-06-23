# AxisML 高层设计（High-Level Design）

## 1. 概述

**AxisML** 是面向机器学习工作负载的一站式平台，覆盖模型开发、训练、制品管理、在线推理与运维。本文是系统设计的高层导航：平台边界、核心概念、组件职责与关键不变量；字段级 schema、状态机与实现契约以各组件详细设计为准。

## 2. 核心概念

平台采用 **两层业务模型 + 一层定义 / 视图**：

- **集群词汇层**：`ResourcePool` CRD（内嵌 `units[]`）与集群级 `Tenant` CR，由 [cluster-manager](../../axisml-system/docs/system_design/cluster-manager.md) 经 REST 维护（admin 视角的"K8s 写抽象"）。cluster-manager 据 ResourceUnit 规格把配额「资源单元 × 数量」折算进 `Tenant.spec.quotas[]`；compute 通过 Informer 直读 ResourcePool 展开，[tenant-operator](../../axisml-system/docs/system_design/tenant-operator.md) 直读 Tenant CR 落地 Namespace / ElasticQuota / 初始化资源。
- **工作负载层**：运行（Run = `MLRun`）/ Service / 制品版本，由 [compute-service](../../axisml-system/docs/system_design/compute-service.md) 与 [artifact-hub](../../axisml-system/docs/system_design/artifact-hub.md) 承载；二者以 **tenant scope**（租户逻辑作用域，等于 `identifier`）分区。现有 API / PG 字段仍名为 `namespace`，但它不是 Kubernetes Namespace。
- **定义 / 视图层**：[Platform](../../axisml-platform/docs/system_design/backend.md) 持有租户持久记录与生命周期权威（自有 `tenants` 表，`identifier` 唯一标识），以及 Job / Experiment / Model / Image 的 name 级**定义**和“用户 → tenant scope → Kubernetes Namespace”映射。租户的 K8s 物化经 cluster-manager REST 下发，Platform 不直接操作任何 CR；运行与制品版本在下游，经 label 与 `(kind, name)` 实时关联。

### 2.1 概念速查

| 术语 | 对应对象 | 详细设计 |
| --- | --- | --- |
| 租户 Tenant | 集群级 `Tenant` CR + Platform `tenants` 行；`identifier` 是 tenant scope，`spec.namespace.name` 是 K8s Namespace | [cluster-manager #3](../../axisml-system/docs/system_design/cluster-manager.md#3-核心模型) / [platform #3](../../axisml-platform/docs/system_design/backend.md#3-核心模型) |
| 资源池 ResourcePool | `ResourcePool` CRD（cluster-scoped），`spec.units[]` 内嵌 | [cluster-manager #3](../../axisml-system/docs/system_design/cluster-manager.md#3-核心模型) |
| 资源单元 ResourceUnit | `ResourcePool.spec.units[]` 内嵌项，与 pool 同生灭 | [cluster-manager #3](../../axisml-system/docs/system_design/cluster-manager.md#3-核心模型) |
| 资源配额 Quota | `Tenant.spec.quotas[]`（「资源单元 × 数量」）→ 每 pool 一个 `ElasticQuota`（`min`/`max` 由 cluster-manager 折算） | [cluster-manager #4](../../axisml-system/docs/system_design/cluster-manager.md#4-核心功能) / [tenant-operator #4](../../axisml-system/docs/system_design/tenant-operator.md#4-核心功能) |
| 任务（定义）Job | Platform `jobs` 行（可复用模板） | [platform #3.2](../../axisml-platform/docs/system_design/backend.md#32-定义jobs--experiments--models--images) |
| 实验（定义）Experiment | Platform `experiments` 行（训练特化模板）；Run 经 `axisml.io/experiment` label 关联 | [platform #3.2](../../axisml-platform/docs/system_design/backend.md#32-定义jobs--experiments--models--images) |
| 运行 Run | `MLRun` CR（Job / 实验的一次运行，`<定义>-<n>`），经 `axisml.io/{job,experiment}` label 关联 | [compute-operator #3](../../axisml-system/docs/system_design/compute-operator.md#3-核心模型) |
| 服务 Service | `MLService` CR | [compute-operator #3](../../axisml-system/docs/system_design/compute-operator.md#3-核心模型) |
| 工作区 Workspace | `mlservices.kind='workspace'`（复用 `MLService(native, deployment)`） | [compute-service #3](../../axisml-system/docs/system_design/compute-service.md#3-核心模型) |
| TensorBoard | `mlservices.kind='tensorboard'`（按需临时实例，复用同上） | [compute-service #3](../../axisml-system/docs/system_design/compute-service.md#3-核心模型) |
| 流量策略 Traffic Policy | `MLTrafficPolicy` CR + PG 行（一个稳定入口加权分发到多服务） | [compute-service #4.3](../../axisml-system/docs/system_design/compute-service.md#43-流量策略mltrafficpolicy) |
| 制品（定义）Model / Image | Platform `models` / `images` 行（name 级定义） | [platform #3.2](../../axisml-platform/docs/system_design/backend.md#32-定义jobs--experiments--models--images) |
| 制品版本 Artifact version | `(tenantScope, kind, name, version)` 四元组；兼容字段名仍为 `namespace` | [artifact-hub #3](../../axisml-system/docs/system_design/artifact-hub.md#3-核心模型) |

### 2.2 关键不变量

系统级不变量在此处定义一次，各组件文档引用而不重述：

- **tenant scope = 租户 `identifier`**：`identifier` 同时是 Tenant CR 名和 compute / artifacts 的逻辑分区键。现有 API / PG 的 `namespace` 字段表示 tenant scope，不表示 K8s Namespace。
- **Kubernetes Namespace 独立建模**：`Tenant.spec.namespace.name` 是物理落地点，可被多个 Tenant 共享；per-tenant 资源靠 tenant ID、命名前缀和 label 隔离。
- **Tenant CR 即权威态**：cluster-manager 以 REST 直接读写 etcd 上的 `Tenant` / `ResourcePool` CR（无 PG 派生、无 reconciler）；Platform `tenants` 表持有展示元数据、停用状态与物理 namespace 映射，经 cluster-manager REST 物化 / 回收 CR。
- **Cluster Manager 是 K8s admin REST 抽象**：把 admin 视角的 K8s 写 / 读收敛为 REST，让 Platform 全程不直接调 K8s API；无独立持久化、无 reconciler、无 leader election。
- **所有 AxisML Pod 走 koord-scheduler**：任何 backend handler 渲染的 Pod 必须设 `schedulerName: koord-scheduler` 并带 `quota.scheduling.koordinator.sh/name` label——不存在绕过配额的调度路径。
- **Operator 互不感知**：tenant-operator 不看 MLRun / MLService；compute-operator 不看 Tenant / ElasticQuota（仅透传 quota 名）。
- **Platform 拥有定义、下游拥有实例**：Job / 实验 / Model / Image 的 name 级定义在 Platform；运行与制品版本在下游，经 label 与 `(kind, name)` 实时关联。Platform 不建 run/version 索引表，不缓存 phase / digest / 配额用量等可变状态（一律实时回源）。
- **分组维度走 labels**：Run 经 `axisml.io/{job,experiment}` 归属；其余自定义分组走 `labels.axisml.io/<dim>`，list 端点支持 `?labelSelector=`。
- **外部入口只在 Platform**：cluster-manager / compute-service / artifact-hub 不暴露到集群外，仅接受 Platform 内部调用并信任 `X-Axisml-User` 身份透传。

## 3. 功能矩阵

表示系统设计覆盖状态，非代码实现完成度。

| 分类 | 功能项 | 设计状态 |
| --- | --- | :---: |
| 训练中心 | 工作区 / 实验 / 自定义任务 | ✅ |
| | 模型定制 | TBD |
| 服务中心 | 在线服务 / 流量配置 | ✅ |
| 资产中心 | 模型 / 镜像 | ✅ |
| | 数据集 | 底层 kind 已保留，产品形态 TBD |
| 系统管理 | 租户管理 / 资源池 / 资源单元 / 资源配额 | ✅ |
| 训练中心 | 评估 | TBD（沿用 Job → Run 两级模型） |

> **删除语义**：`Tenant` 删除前必须清空成员和活跃资源，随后同步删除持久记录与 Tenant CR，不提供 restore；`Job` / `Experiment` 定义删除——有活跃 Run 则阻止，否则级联软删全部 Run；`Run` / `Service` / `Workspace` / `TrafficPolicy` / `TensorBoard` 删除即终态；`Model` / `Image` 定义删除级联软删全部版本，`Artifact` 版本软删后四元组永不复用。详见各组件 §4。

## 4. 整体架构

系统沿职责划分三层，各部署为一个 Helm chart（见 [deployment.md](deployment.md)）：**Platform 层**（用户面，唯一对外）、**System 层**（控制面，自研领域能力，不对外）、**Infra 层**（第三方基础设施）。

```
                              External Users
                                    │  外部流量 → Envoy Gateway(Infra)
                                    ▼
┌──────────────  Platform 层（用户面 · 唯一对外）  ──────────────┐
│  AxisML Platform — Frontend (React) + Backend (Go)            │
│  用户入口 / 认证 / 业务编排；持有租户视图 ↔ namespace 映射     │
│  创建 Job / Service 时仅向 compute 传 pool/unit 名字           │
└────┬──────────────────┬──────────────────────┬───────────────┘
     │  内部调用（信任 X-Axisml-User）           │
     ▼                  ▼                      ▼
┌──────────────  System 层（控制面 · 不对外）  ──────────────────┐
│  ┌─────────────┐ ┌──────────────┐ ┌──────────────────────┐   │
│  │ Cluster Mgr │ │ Compute Svc  │ │  Artifact Hub        │   │
│  │ ResourcePool│ │ Job/Service/ │ │  model,image → zot   │   │
│  │ + Tenant CR │ │ Workspace/   │ │  dataset → RustFS    │   │
│  └─────────────┘ │ Traffic →CR  │ └──────────────────────┘   │
│  ┌─────────────┐ └──────┬───────┘                            │
│  │tenant-oper. │ ┌──────┴───────────┐   CRDs 随本层发布        │
│  │Tenant CR →  │ │ compute-operator │                        │
│  │NS/Quota/... │ │ MLRun/MLService →│                        │
│  └──────┬──────┘ │ backend handler  │                        │
└─────────┼────────┴──────┬───────────┴────────────────────────┘
          │ 元数据 / 存储   │ schedulerName: koord-scheduler
          ▼               ▼
┌──────────────  Infra 层（第三方基础设施）  ────────────────────┐
│  Envoy Gateway · Koordinator(koord-scheduler + Quota)        │
│  PostgreSQL · zot(OCI) · RustFS(S3) · GPU Operator · kube-prometheus│
└─────────────────────────────┬──────────────────────────────┘
                              ▼  所有 workload Pod 经 koord-scheduler 落到
                       Kubernetes Cluster
```

核心调用关系：

- 外部流量经 Envoy Gateway 进入 Platform；System 层仅接受 Platform 内部调用。
- **租户闭环**：Platform 写 `tenants` 表 → cluster-manager REST 创建 / 删除 Tenant CR → tenant-operator 落地 Namespace / ElasticQuota / 初始化资源。
- **负载闭环**：Platform 调 compute（创建 workload 时仅传 pool/unit 名字）→ compute 写 PG 并 patch MLRun / MLService / MLTrafficPolicy CR → compute-operator 按 `spec.backend.{name, engine}` 路由 handler 渲染 K8s 与第三方 CR。
- **制品域**：artifacts 元数据走 PG；model / image 走 zot（OCI），dataset 走 RustFS（S3），上传下载由消费方直连存储，artifacts 不代理大文件。

## 5. 文档导航

设计文档按系统三层组织，每层一个 `overview.md`：

**Platform 层**（用户面）— [platform/overview.md](../../axisml-platform/docs/system_design/overview.md)
| 文档 | 一句话职责 |
| --- | --- |
| [backend.md](../../axisml-platform/docs/system_design/backend.md) | 用户入口与业务编排，持有租户持久记录 + 四张定义 + 视图层映射 |
| [frontend.md](../../axisml-platform/docs/system_design/frontend.md) | React 前端：路由 / 状态 / i18n / 数据面接入 |
| [auth.md](../../axisml-platform/docs/system_design/auth.md) | 认证 / RBAC / 数据面接入 / 下游身份透传 |

**System 层**（控制面）— [system/overview.md](../../axisml-system/docs/system_design/overview.md)
| 文档 | 一句话职责 | 关键模型 |
| --- | --- | --- |
| [cluster-manager.md](../../axisml-system/docs/system_design/cluster-manager.md) | admin 域 K8s REST 抽象（ResourcePool + Tenant CRUD、配额折算、集群容量 / 指标） | ResourcePool CR（含 `units[]`）+ Tenant CR |
| [compute-service.md](../../axisml-system/docs/system_design/compute-service.md) | 业务域计算服务，管理 Run / Service / Workspace / TrafficPolicy 与三类 CR | MLRun + MLService + MLTrafficPolicy（PG，namespace 分区） |
| [artifact-hub.md](../../axisml-system/docs/system_design/artifact-hub.md) | 业务域制品服务，元数据 / 存储分离 | Artifact 四元组 `(namespace, kind, name, version)` |
| [tenant-operator.md](../../axisml-system/docs/system_design/tenant-operator.md) | Tenant CR → Namespace / ElasticQuota / 初始化资源 | Tenant CR（cluster-scoped） |
| [compute-operator.md](../../axisml-system/docs/system_design/compute-operator.md) | 三类 CR → backend handler → K8s 与网关 CR | 三类 CR + backend handler registry |

**Infra 层**（第三方基础设施）— [infra/overview.md](../../axisml-infra/docs/system_design/overview.md)：[服务网关](../../axisml-infra/docs/system_design/overview.md#3-服务网关) · [存储](../../axisml-infra/docs/system_design/overview.md#4-存储) · [加速器管理](../../axisml-infra/docs/system_design/overview.md#5-加速器管理) · [调度与配额](../../axisml-infra/docs/system_design/overview.md#6-调度与配额) · [监控](../../axisml-infra/docs/system_design/overview.md#7-监控)。

**跨层文档**：[deployment.md](deployment.md)（Helm 分层与部署顺序）。PostgreSQL schema 按层归各自 docs：[system/database.md](../../axisml-system/docs/system_design/database.md) · [platform/database.md](../../axisml-platform/docs/system_design/database.md)。

## 6. 关键设计决策

| 决策项 | 决策 |
| --- | --- |
| 计算抽象 | 通过 CRD（MLRun / MLService / MLTrafficPolicy / Tenant）声明式管理，框架无关 |
| 定义 / 实例分层 | Platform 自有四张 name 级定义；运行与版本留下游，经 label 与 `(kind, name)` 实时关联，不缓存可变状态 |
| 控制平面拆分 | tenant-operator + compute-operator 两个独立二进制，按变更频率与权限边界分离 |
| 租户与配额归属 | cluster-manager 持 Tenant CR + 配额折算（与 ResourcePool 同为集群级 admin CR）；Platform 持租户持久记录、K8s Namespace 映射与停用状态；租户删除为硬删除 |
| Pool/Unit 解耦 | ResourcePool CRD 由 cluster-manager 管（内嵌 units），compute 经 Informer 直读展开；写 / 读路径都经 etcd 收敛 |
| 配额模型 | 配额内联 `Tenant.spec.quotas[]`（「资源单元 × 数量」）→ cluster-manager 折算为每 pool 一个 ElasticQuota，避免独立 Quota CRD |
| 调度收编 | 所有 Pod 强制 `schedulerName: koord-scheduler` + ElasticQuota label |
| 制品抽象 | `(namespace, kind, name, version)` 四元组寻址，无"仓库"两级空间；model/image → zot，dataset → RustFS |
| 部署分层 | `axisml-infra` / `axisml-system` / `axisml-platform` 三 chart，对齐职责分层；PostgreSQL 归 infra |
| 技术栈 | 后端 Go，前端 TypeScript + React |
| 认证鉴权 | Platform 统一入口；内置用户 + RBAC 三档（硬编码）；下层信任 `X-Axisml-User`；OIDC 后续接入 |

## 7. 项目结构

Monorepo 管理所有组件：

```
axisml/                        # 按部署层组织，每层一个自包含目录 + 一个 Makefile
├── axisml-platform/           # 用户面
│   ├── backend/ frontend/     # 代码
│   ├── deploy/helm/           # Platform Helm chart
│   ├── docs/                  # overview + backend/frontend/auth
│   ├── docs/apis/             # 生成的 platform.yaml
│   └── docs/product_design/   # PRD / prototype
├── axisml-system/             # 控制面
│   ├── {tenant-operator, compute-operator, cluster-manager, compute-service, artifact-hub}/
│   ├── deploy/helm/           # System Helm chart (+ crds/)
│   ├── docs/  docs/apis/      # 5 组件设计 + 生成的 OpenAPI specs
│   └── test/                  # testutil / crds/external / setup-envtest（System 集成测试基建）
├── axisml-infra/              # 第三方基础设施
│   ├── deploy/helm/  docs/  scripts/minikube.sh
├── axisml-lite/               # 无 K8s 的 Docker Compose 形态（当前仅 docs/）
├── docs/system_design/        # 跨层：high_level_design.md + deployment.md
├── docs/development_workflow.md # 开发指南（环境 / 构建测试 / 测试分层）
├── pkg/openapigen/  test/e2e/ # 共享：OpenAPI 引擎 / 集中式 e2e 套件
├── Makefile                   # 根编排器（委派给各层 Makefile）
└── README.md
```

部署按三层各为一个 Helm chart，安装顺序 infra → system → platform，卸载反向（详见 [deployment.md](deployment.md)）。
