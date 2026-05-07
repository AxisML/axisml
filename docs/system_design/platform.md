# AxisML Platform 详细设计

AxisML Platform 是平台的用户入口与业务编排层，由 **前端（TypeScript + React）** 与 **后端（Go）** 两个子组件组成。它对外承载用户视图（登录、租户切换、工作区视图、任务 / 服务 / 制品的 UI），对内把用户操作拆解为对 [cluster-manager](cluster-manager.md)、[compute](compute.md)、[artifacts](artifacts.md) 三个内部服务的协作调用。

> 本文档目前给出的是骨架，重点把"租户视图 ↔ namespace 分区"的映射关系锁定下来。IdP、角色模型、UI 的详细页面规范、API 详细 schema 等留待后续版本继续填充。

## 1. 定位

| 能力 | 由谁承担 |
| --- | --- |
| 用户登录 / 身份认证 | Platform（IdP 接入细节 TBD） |
| 角色 / 权限 | Platform |
| 用户视图（租户、工作区、Job / Service、制品的列表 / 详情） | Platform |
| 业务编排（如"创建租户"会同时调 cluster-manager + 在 platform 自身记账） | Platform |
| 调度 / 配额 / Namespace 派生资源落地 | cluster-manager → tenant-operator |
| Job / Service 元数据与 CR 下发 | compute → compute-operator |
| Artifact 元数据与存储后端凭证签发 | artifacts |

Platform 是**唯一**面向外部用户的入口；Compute / Artifacts / Cluster Manager 都只接受集群内 Service DNS 调用，不暴露到集群外（外部流量经 Envoy Gateway → Platform，再由 Platform 内部转调）。

## 2. 核心概念

Platform 引入两个**上层概念**，它们是平台对用户的视图，不是任何下层服务的字段：

### 2.1 租户视图（Tenant View）

用户面向的"租户"概念。一个租户视图持有：

- 一个 cluster-manager 中的 Tenant 名（持有 K8s Namespace、ElasticQuota、初始化资源）；
- 一组成员（用户 + 角色）；
- 一组工作区（§2.2）。

**租户视图的存在不要求 cluster-manager / 下游服务感知用户**——Platform 用自己的 PG 表（如 `platform_tenants`、`platform_memberships`）持有视图层信息，对下层只透出无用户语义的 namespace 字符串与 Tenant 名。

### 2.2 工作区（Workspace）

**工作区是 Platform 把"租户视图"投影到下层 namespace 分区的二元组**：

```
Workspace = (compute_namespace, artifacts_namespace)
```

- `compute_namespace`：传给 compute API 的 namespace 字段，决定 MLJob / MLService CR 落地的 K8s namespace。
- `artifacts_namespace`：传给 artifacts API 的 namespace 字段，决定 Artifact 元数据归属的逻辑分区。

**两者完全独立**：可以同名（`team-a` / `team-a`），也可以不同（`team-a` / `team-a-models`），还可以一对多（一个租户视图下多个工作区共享同一 artifacts namespace 但 compute namespace 不同，便于"训练环境隔离 + 模型仓库共用"）。

工作区与 Tenant 的关系也不是 1:1：一个 Tenant 下可以有多个工作区（多套环境共享 K8s Namespace 与配额），多个 Tenant 也可以共享一个 artifacts namespace（platform 自由编排）。具体映射策略由 platform 的产品设计决定，不下沉到下层服务。

### 2.3 与下层概念的关系

```
┌──────────────── Platform ────────────────┐
│  User → Tenant View → Workspace           │
│                          │                │
│              ┌───────────┼───────────┐    │
│              ▼           ▼           ▼    │
│         tenant_name  compute_ns  artifacts_ns │
└──────────────┬───────────┬───────────┬────┘
               │           │           │
               ▼           ▼           ▼
       cluster-manager   compute     artifacts
       （Tenant CR）      （MLJob /  （Artifact
                          MLService）  metadata）
```

下层三个服务都不知道"租户视图"或"工作区"——它们只看到 Tenant 名（cluster-manager / tenant-operator）或裸 namespace 字符串（compute / artifacts）。

## 3. 组件协作

### 3.1 创建租户视图

```
1. 用户在 UI 提交"创建租户"
2. Platform 后端校验输入、生成租户视图 ID
3. Platform 后端调用 cluster-manager:
     POST /api/v1/tenants
     { name, namespace.name, quotas, initResources, ... }
4. cluster-manager 创建 Tenant CR；tenant-operator 异步落地
5. Platform 在 platform_tenants 表中保存
     { tenant_view_id, cluster_manager_tenant_name, owner_user_id, ... }
6. Platform 创建默认工作区（compute_namespace = artifacts_namespace = tenant.namespace）
   并保存到 platform_workspaces
7. Platform 返回租户视图 ID 给前端
```

后续"暂停 / 恢复 / 删除"都按此模式走 cluster-manager API；Platform 自身的 PG 行只在最外层做关联记账。

### 3.2 创建额外工作区

```
1. 用户在 UI 选择"为租户 X 新增工作区 dev"
2. Platform 后端校验：用户在租户 X 下有权限
3. Platform 后端选择 namespace 策略：
     - compute_namespace：复用 Tenant 的 namespace，或单独申请新 namespace（后者需先调 cluster-manager 给该 Tenant 加一个新 namespace 引用 / 或新建一个 Tenant CR 共享同一 namespace）
     - artifacts_namespace：自由命名（artifacts 不校验存在）
4. Platform 写 platform_workspaces 表
5. （可选）Platform 调 cluster-manager 给该 Tenant 追加一条 quota，与新工作区对应
```

### 3.3 提交训练任务

```
1. 用户在 UI 选择 workspace W、提交 Job 表单
2. Platform 后端解析 W → compute_namespace = ns
3. Platform 调用 compute:
     POST /api/v1/namespaces/{ns}/jobs
     { name, backend, scheduling.quota, modelRef, ... }
   - scheduling.quota 由 Platform 根据 W 关联的 ElasticQuota 名填入
   - modelRef 由 Platform 解析为 artifacts 的 (namespace, kind, name, version) 后传给 compute
4. compute 写 PG → reconciler 异步下发 MLJob CR 到 ns
5. compute-operator 监听 MLJob → 派生 Pod / PodGroup / ...
6. Platform 通过 compute 的 list / get API 给前端展示状态
```

### 3.4 注册 / 解析模型

```
1. 用户在 UI 选择 workspace W、上传模型
2. Platform 后端解析 W → artifacts_namespace = ans
3. Platform 调 artifacts initiate:
     POST /api/v1/namespaces/{ans}/artifacts/{kind}/{name}/initiate
4. artifacts 返回 storage URI + 短期凭证；Platform 把凭证透传给 cli / 前端直传
5. cli 直连 zot / RustFS 完成上传
6. Platform 调 artifacts complete 完成两阶段写
7. 用户后续在 W 下提交 Job/Service 时，Platform 把 (kind=model, name, version)
   解析为 artifacts 的引用，作为 modelRef 传给 compute
```

artifacts namespace 与 compute namespace 不需要相等——Platform 在解析时按 W 的二元组分别映射。

## 4. Namespace 映射策略

Platform 的产品逻辑决定如何分配 namespace。常见策略：

| 策略 | 适用场景 | 特征 |
| --- | --- | --- |
| 1:1 同名（compute_ns == artifacts_ns == tenant.namespace） | 简单租户、单环境 | 最简单；查询时 namespace 与租户视图直接对应 |
| 1:N（多 workspace 共享 K8s Namespace 与 quota，但 artifacts_ns 各自独立） | 团队下多产品线，模型仓库分离 | 训练环境共享，模型隔离 |
| N:1（多 tenant 共享同一个 artifacts_ns） | 多团队复用基础模型 | 模型仓库公共化，训练隔离 |
| 跨租户共享 K8s Namespace（多 Tenant CR 指向同一 namespace） | 沙箱环境、轻量团队共享 | 由 [tenant-operator §4.7](tenant-operator.md) 共享 Namespace 语义支撑 |

**关键原则**：下层三个服务对策略选择保持完全无知；Platform 是唯一负责"如何编排"的层。

## 5. 后续工作

- **认证 / 鉴权**：IdP 接入（OIDC / SAML / 内置）、角色模型（platform 内置 RBAC）、API 鉴权 middleware、对下层服务的身份透传协议（X-Axisml-User header 的具体语义）。
- **前端架构**：Vite + React + 组件库、状态管理、API client 自动生成、UI 路由。
- **后端架构**：HTTP server 框架、ORM、与三个内部服务的 client SDK（基于 OpenAPI 自动生成）。
- **审计与计费**：把租户视图、工作区、操作流水的审计日志持久化（与 cluster-manager 的 K8s Event 互补）。
- **多集群联邦**：当一个租户视图横跨多个 K8s 集群时的 namespace 映射策略（远期）。
- **完整 OpenAPI 契约**：Platform 后端面向前端 / 外部 SDK 的 API schema。

## 6. 相关引用

- [docs/system_design/overview.md](overview.md) 概述了 Platform 在系统中的位置。
- [docs/system_design/cluster-manager.md](cluster-manager.md) 描述 Platform 创建 / 暂停 / 删除租户时调用的 API。
- [docs/system_design/compute.md](compute.md) 描述 Platform 提交 Job / Service 时调用的 API。
- [docs/system_design/artifacts.md](artifacts.md) 描述 Platform 注册 / 解析制品时调用的 API。
