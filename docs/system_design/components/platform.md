# AxisML Platform 详细设计

## 0. 文档说明

本文档是 AxisML Platform 子系统的单文档详设：覆盖整体架构、后端工程契约、认证模型，以及租户 / 资源池 / 工作区 / 计算任务 / 在线服务五个一级功能。每个功能以独立章节组织。

页面级 ASCII mockup 单独抽到 [wireframe.md](../wireframe.md)；本文保留各功能的列表列定义、表单字段表、Tab 结构等结构化描述。

系统级架构与术语以 [docs/system_design/overview.md](../overview.md) 为准；与系统层重叠的概念（Tenant / ResourcePool / ResourceUnit / Quota / Job / Service / Artifact）一律沿用上层定义，本文不重复。

---

## 1. 概述

**AxisML Platform** 是 AxisML 系统中唯一直接面向用户的层。它由 **前端**（Web UI，TypeScript + React）与 **后端**（Go REST API 服务）两部分组成，部署在 Envoy Gateway 之后，对下串联 [Cluster Manager](cluster-manager.md)、[Compute](compute.md) 与 [Artifacts](artifacts.md) 三个内部服务，对上为登录用户提供统一的业务视图与操作入口。

### 1.1 菜单与功能矩阵

| 一级菜单 | 二级 | 本文章节 | 设计状态 |
| --- | --- | --- | :---: |
| Dashboard | — | 纯前端聚合视图，无独立后端详设 | ✅ (UI) |
| 应用中心 | 智能体 / Skills / MCP | [§13 后续迭代](#13-后续迭代) 入口预留 | TBD |
| 训练 & 推理 | 工作区 | [§8 工作区](#8-工作区) | ✅ |
| 训练 & 推理 | 计算任务 | [§9 计算任务](#9-计算任务) | ✅ |
| 训练 & 推理 | 在线服务 | [§10 在线服务](#10-在线服务) | ✅ |
| 制品中心 | 模型 / 镜像 / 数据集 | [§13 后续迭代](#13-后续迭代) 入口预留 | TBD |
| 系统管理 | 租户管理（含配额 Tab） | [§6 租户管理](#6-租户管理) | ✅ |
| 系统管理 | 资源池管理（含资源单元） | [§7 资源池与资源单元管理](#7-资源池与资源单元管理) | ✅ |
| 系统管理 | 数据卷管理 | [§13 后续迭代](#13-后续迭代) 入口预留 | TBD |
| 横切 | 用户 / 角色 / 鉴权 | [§5 认证与租户权限模型](#5-认证与租户权限模型) | ✅ |

图例：`✅` 表示已有完整设计或纯前端聚合，可直接进入实现；`TBD` 表示概要中保留入口，详设按需求展开。

### 1.2 核心概念（Platform 视角）

Platform 在系统层概念之上额外引入了一组用户视角的对象，把「裸 namespace 分区」的下层服务包装成租户化、用户化的体验。

#### 1.2.1 用户（User）

Platform 内部维护的登录身份。第一版采用内置用户表（用户名 + bcrypt 密码 + JWT 颁发），所有外部 HTTP 流量必须先通过 Platform 鉴权后才能进入下游服务。OIDC / SAML 等外部 IdP 通过抽象出 `IdentityProvider` 接口预留接入点，本期不交付。详见 [§5](#5-认证与租户权限模型)。

#### 1.2.2 用户角色（Persona）

Platform 在产品语义上区分三类用户身份：

- **系统管理员（system admin）**：平台级超管。负责租户、资源池、资源单元、数据卷的全生命周期管理；可见「系统管理」菜单全部入口，可读所有租户数据。
- **租户管理员（tenant admin）**：单租户负责人。负责本租户在各资源池下的配额拆分、成员管理与本租户内全部业务对象（Job / Service / Artifact / Workspace）的读写；不能跨租户操作。
- **普通用户（user）**：算法工程师、数据科学家、推理服务运维等业务使用者的统称。在所属租户内使用工作区、提交任务、部署服务、注册与消费制品。

Persona 与 RBAC 角色的对应：

| Persona | RBAC role | 范围 |
| --- | --- | --- |
| 系统管理员 / system admin | `system-admin` | 全局 |
| 租户管理员 / tenant admin | `tenant-admin` | 单租户 |
| 普通用户 / user | `user` | 单租户 |

#### 1.2.3 角色（Role）与权限（Permission）

三类 persona 在 Platform RBAC 中实现为三档角色：

| 角色 | 范围 | 能力概览 |
| --- | --- | --- |
| `system-admin` | 全局 | 用户 / 角色 / 租户 CRUD、资源池 / 资源单元 CRUD、读所有租户数据 |
| `tenant-admin` | 单租户 | 租户内成员管理、配额申请、对该租户全部业务对象（Job / Service / Artifact / Workspace）的读写 |
| `user` | 单租户 | 提交 / 管理自己创建的业务对象、读取本租户内的共享资产 |

角色与权限的具体绑定矩阵见 [§5.3](#53-角色与权限);每个功能章节给出本功能下的可见性子矩阵。

#### 1.2.4 租户（Tenant）

**Platform 自身不为租户建任何 PG 表**——租户的权威完全在 [cluster-manager](cluster-manager.md) 持有的 PG `tenants` 表（详见 [cluster-manager.md §3](cluster-manager.md#3-pg-schema)）；Tenant CR 是 cluster-manager 通过 outbox + reconciler 渲染下发的派生产物。所有租户字段（`displayName` / `description` / `business_unit` / `quotas` / 状态等）都是 cluster-manager API 的一级字段，Platform 仅做透传与展示。

`user_tenant_roles` 表通过 `tenant_name`（text）引用租户——cluster-manager `tenants.name` 不可变（partial unique on `WHERE deleted_at IS NULL`），等价于一个稳定 FK。

详细模型与 UI 入口见 [§6 租户管理](#6-租户管理)。

#### 1.2.5 工作区（Workspace）

用户菜单「训练 & 推理 → 工作区」下的具体对象。语义为一台 **长驻的交互式开发容器**（Jupyter Notebook / VSCode Server / SSH 等）。底层复用 [`MLService(native, deployment)`](compute-operator.md) 后端：

- Platform 不为工作区建任何 PG 表：工作区 = Compute `services` 表中 `kind='workspace'` 的行；
- 创建 = 调 Compute 创建一个 `MLService(native, deployment)`，`kind=workspace`，单 role、单副本，长驻容器；
- 启停 = patch 该 MLService 的 `roles[0].replicas`；
- 「这是工作区还是普通在线服务」由 Compute service 的 `kind` 列表达，Platform 列表查询 = `compute.ListServices(kind=workspace)`。

详见 [§8 工作区](#8-工作区)。

#### 1.2.6 数据卷（DataVolume，TBD）

预留概念，对应菜单「系统管理 → 数据卷管理」。可能的实现取向包括：用户级 PVC 抽象、数据集挂载路由（基于 Artifacts dataset）、集群 StorageClass 视图。本期暂不冻结字段，仅在 [§13](#13-后续迭代) 保留入口。

#### 1.2.7 概念速查

| 术语 | 英文名 | 来源 / 对应对象 |
| --- | --- | --- |
| 系统管理员 | system admin | RBAC `system-admin` |
| 租户管理员 | tenant admin | RBAC `tenant-admin` |
| 普通用户 | user | RBAC `user` |
| 用户 | User | Platform 内部表 |
| 角色 | Role | Platform 内部表，含内置三档 |
| 权限 | Permission | Platform 内部表，绑定到 Role |
| 租户 | Tenant | cluster-manager `tenants` 表（权威）+ Tenant CR（派生） |
| 工作区 | Workspace | Compute `services` 表中 `kind='workspace'` 的行 + `MLService(native, deployment)` |
| 数据卷 | DataVolume | TBD |

系统层共享的概念（Tenant、ResourcePool、ResourceUnit、Quota、Job、Service、Artifact）参见 [上层 overview §2](../overview.md#2-核心概念)。

---

## 2. 整体架构

```
                   ┌────────────────────────────┐
                   │       External Users       │
                   └──────────────┬─────────────┘
                                  ▼
                   ┌────────────────────────────┐
                   │  AxisML Infra Envoy Gateway│
                   │  TLS / Auth / 路由 / 限流   │
                   └──────────────┬─────────────┘
                                  ▼
   ┌──────────────────────────────────────────────────────────┐
   │                    AxisML Platform                       │
   │  ┌──────────────────┐         ┌────────────────────────┐ │
   │  │  Frontend (SPA)  │ ──────► │  Backend (Go, REST)    │ │
   │  │  React + TS +    │         │  Gin + GORM + Cobra    │ │
   │  │  Vite            │         │                        │ │
   │  └──────────────────┘         │  ┌──────────────────┐  │ │
   │                               │  │  Auth / RBAC     │  │ │
   │                               │  ├──────────────────┤  │ │
   │                               │  │  Resource APIs   │  │ │
   │                               │  ├──────────────────┤  │ │
   │                               │  │  Orchestrator    │  │ │
   │                               │  ├──────────────────┤  │ │
   │                               │  │  Typed Clients   │  │ │
   │                               │  └────────┬─────────┘  │ │
   │                               └───────────┼────────────┘ │
   │                                           │              │
   │                                  ┌────────┴─────┐        │
   │                                  ▼              ▼        │
   │                          Platform PG   下游服务调用       │
   │                          users / roles                    │
   │                          user_tenant_roles                │
   │                          sessions / audit_logs            │
   └───────────────────────────────────────────┬──────────────┘
                                               │
              ┌────────────────────┬───────────┴────────────┬───────────────────┐
              ▼                    ▼                        ▼                   ▼
      ┌───────────────┐  ┌───────────────────┐  ┌─────────────────┐  ┌─────────────────┐
      │ ClusterMgr    │  │ AxisML Compute    │  │ AxisML Artifacts│  │ Prometheus       │
      │ Tenant CR     │  │ Job / Service /   │  │ Model / Image / │  │ (kube-prom)      │
      │ → tenant-op   │  │ ResourcePool /    │  │ Dataset 元数据  │  │ Dashboard 数据源 │
      │               │  │ ResourceUnit      │  │                 │  │                 │
      └───────────────┘  └───────────────────┘  └─────────────────┘  └─────────────────┘
```

### 2.1 核心调用关系

- 外部流量经 Envoy Gateway 进入 Platform；下游服务全部为内部调用，不直接暴露到集群外。
- Platform 调用 **Cluster Manager** 进行租户与配额管理（写 `Tenant` CR）。
- Platform 调用 **Compute** 在指定 namespace 下管理任务、服务、资源池、资源单元。
- Platform 调用 **Artifacts** 在指定 namespace 下管理模型；未来扩展到镜像、数据集、评估报告。
- Platform 自身仅持有「身份 + 视图映射」类轻量元数据；业务对象（Job / Service / Artifact）的权威分别在 Compute / Artifacts 的 PG 与 K8s CR 中。
- Dashboard 视图所需的指标数据来自基础设施层的 Prometheus（kube-prometheus-stack，详见 [infra.md](../infra.md)），由 Platform 后端聚合 + 前端拼装。

### 2.2 组件职责

#### 2.2.1 前端

- **语言**：TypeScript
- **框架**：React
- **构建**：Vite（参见 [components/platform/frontend/README.md](../../components/platform/frontend/README.md)）
- **目录约定**：`src/{pages,components,api,hooks,styles}` + `public/`，由前端组组织页面级设计；后端不约束 UI 结构。
- **职责**：提供登录、菜单导航、列表 / 详情 / 表单操作页，调用 Platform 后端 REST API；不直接调用任何下游服务。

页面级设计不在系统设计文档范围内，由前端开发按各功能章节给出的 API 字段独立设计；视觉骨架占位见 [wireframe.md](../wireframe.md)。

#### 2.2.2 后端

- **语言 / 框架**：Go + [Gin](https://github.com/gin-gonic/gin) + [GORM](https://gorm.io/) + [Cobra](https://github.com/spf13/cobra)
- **结构化日志**：[zap](https://github.com/uber-go/zap)
- **K8s 客户端**：仅在需要直接读 `Tenant.status` 或直管 PVC 时使用 `controller-runtime` client；其他操作一律走 Cluster Manager / Compute / Artifacts 的 REST API
- **职责**：
  - **外部 API 入口**：所有用户操作的 REST 入口；Platform 是 Cluster Manager / Compute / Artifacts 唯一的外部调用方；
  - **业务编排**：跨服务的串联逻辑（创建租户 → 写 Platform PG → 校验 → 调下游）；
  - **身份与租户权限**：用户登录、JWT 颁发、RBAC 校验、租户访问边界；
  - **视图层映射**：把用户视角的「租户」「Workspace」翻译为下游服务需要的裸 namespace 字符串与 ElasticQuota CR 名；其中 namespace 在调用时按上下文实时确定，不在 Platform PG 缓存。

后端代码骨架、PG schema 与下游客户端的具体形态见 [§4 后端架构与代码骨架](#4-后端架构与代码骨架)。

---

## 3. 核心调用链

下面给出三条覆盖关键能力的端到端时序，作为后续详设的连接锚点。所有时序中的下游 API 字段以各 `core/<service>.md` 为准。

### 3.1 用户提交训练任务

```
User ──► Platform.Login                                  // JWT 颁发
User ──► Platform.GET /api/v1/tenants                    // 列出可见租户
User ──► Platform.POST /api/v1/jobs
        body: { tenant_view_id, pool, resource_unit_id,
                model_ref, image_ref, roles, ... }
   │
   ├─► Platform 内部：
   │     1. RBAC 校验（user 是否对该租户有 user 以上角色）
   │     2. 加载租户展示元数据；并确定本次调用 Compute / Artifacts
   │        所需的 namespace（典型来源：Tenant CR 的 spec.namespace.name 或调用上下文）
   │     3. 读 Tenant CR → 取 spec.quotas[pool] 中的 quota 名
   │        （ElasticQuota CR 名形如 axisml-<tenant>-<pool>-<quota>）
   │     4. 解析 model_ref / image_ref：
   │        Artifacts.GET /api/v1/namespaces/<artifacts_ns>/artifacts/<kind>/<name>/<version>/resolve?usage=inspect
   │        → 拿 URI / digest / auth_hint
   │
   ├─► Compute.POST /api/v1/namespaces/<compute_ns>/jobs
   │     body: { name, resource_unit_id, quota: <ElasticQuota name>,
   │             backend, roles, ... }
   │   ◄── 200 OK + Job View（status=Creating）
   │
   ◄── Platform 200 OK，把 Compute 的 View 透传给前端

[异步]
compute-operator 创建 MLJob → Pod → koord-scheduler 按 ElasticQuota 调度
↓
Compute Informer 回流状态 → Platform 通过轮询 / 流式 GET 把状态返回前端
```

### 3.2 用户上传 / 引用模型

```
User ──► Platform.POST /api/v1/models
        body: { tenant_view_id, name, version, displayName, ... }
   │
   ├─► RBAC + 按上下文确定 artifacts namespace（典型来源：Tenant CR 的 spec.namespace.name）
   │
   ├─► Artifacts.POST /api/v1/namespaces/<artifacts_ns>/artifacts/model/<name>
   │     body: InitiateInput
   │   ◄── { artifact_id, storage_kind, uri, upload_credentials, expires_at }
   │
   ◄── Platform 透传 InitiateResult（含 zot 凭证）

User CLI ──► zot 直传 OCI manifest + blob

User ──► Platform.POST /api/v1/models/<id>/complete
        body: { digest }
   │
   ├─► Artifacts.POST /api/v1/namespaces/<artifacts_ns>/artifacts/model/<name>/<version>/complete
   │     body: { digest }
   │   ◄── 200 OK + View（status=Ready, digest 锁定）
```

后续 MLJob / MLService 通过 `model_ref` 引用此模型时走 [§3.1](#31-用户提交训练任务) 的解析路径。

### 3.3 管理员创建租户 + 配额

```
Admin ──► Platform.POST /api/v1/tenants
         body: { name, displayName, namespace, init_resources, quotas[] }
   │
   ├─► RBAC：必须 system-admin
   │
   ├─► ClusterManager.POST /api/v1/tenants
   │     body: CreateTenantRequest（透传）
   │   ◄── TenantResponse（含 spec.namespace 落地结果）
   │
   ├─► Platform 内部 PG：写 user_tenant_roles 中创建者的 admin 绑定（如适用）
   │
   ◄── 200 OK + Tenant

Admin ──► Platform.POST /api/v1/tenants/<name>/quotas
   │
   ├─► ClusterManager.POST /api/v1/tenants/<name>/quotas
   │   ◄── TenantResponse（spec.quotas[] 已更新）
   │
   ◄── Platform 200 OK

[异步]
tenant-operator 监听 Tenant CR：
   - 渲染 / 更新 Namespace
   - 渲染 ElasticQuota CR：axisml-<tenant>-<pool>-<quota>
   - 渲染 init_resources：Secret / ConfigMap / SA / RBAC

Tenant.status.phase 翻为 Active 后，Platform 才允许该租户开始提交工作负载（前端置灰逻辑）。
```

---

## 4. 后端架构与代码骨架

本节是 Platform 后端的工程契约，吸收所有横切设计（代码组织、PG schema、下游客户端、配置、部署）。

### 4.1 仓库与目录布局

```
components/platform/backend/
├── cmd/
│   └── platform/
│       └── main.go                  # Cobra 入口
├── internal/
│   ├── app/
│   │   ├── serve.go                 # 启动 HTTP server + 后台任务
│   │   ├── bootstrap.go             # 首次启动初始化（管理员账号）
│   │   ├── migrate.go               # GORM 迁移
│   │   └── modules.go               # 依赖注入装配
│   ├── config/                      # 配置加载与校验
│   ├── server/                      # Gin 引擎、中间件、RFC7807 problem
│   ├── auth/                        # 内置用户、JWT、RBAC、IdentityProvider 接口
│   ├── db/                          # GORM 初始化、迁移脚本
│   ├── client/
│   │   ├── clustermanager/          # cluster-manager typed client
│   │   ├── compute/                 # compute typed client
│   │   └── artifacts/               # artifacts typed client
│   ├── orchestrator/                # 跨服务编排
│   ├── user/                        # 用户 handler/service/repository
│   ├── tenant/                      # 租户 handler/service/repository
│   ├── workspace/                   # 工作区 handler/service/repository
│   ├── job/                         # 计算任务编排（无本地表，仅代理）
│   ├── service/                     # 在线服务编排
│   ├── model/                       # 模型 handler（代理 Artifacts）
│   ├── resourcepool/                # 资源池 + 资源单元 handler（代理 Compute）
│   ├── dashboard/                   # 聚合视图
│   └── metrics/                     # Prometheus 指标
├── pkg/
│   ├── errors/
│   └── logging/
├── api/                             # OpenAPI 契约（生成 / 手写）
├── deploy/Dockerfile
├── go.mod
└── Makefile
```

风格与 [components/compute/](../../components/compute/) 保持一致；下游 typed client 的目录命名与各 `core/<service>.md` 中暴露的 REST 路径一一对应。

### 4.2 启动子命令

仿 `components/compute/cmd/compute/main.go`：

| 子命令 | 作用 |
| --- | --- |
| `serve` | 启动 HTTP API（默认）+ 后台任务（如租户状态轮询） |
| `bootstrap` | 一次性：检查并创建初始 `system-admin` 账号、内置角色与权限 |
| `migrate` | 执行 GORM 迁移；CI / 部署 init container 调用 |

### 4.3 错误处理

复用 RFC 7807 problem 风格（参考 [components/compute/internal/server/problem.go](../../components/compute/internal/server/problem.go)）：

```json
{
  "type": "https://axisml.io/errors/quota-exceeded",
  "title": "Quota exceeded",
  "status": 409,
  "detail": "ElasticQuota axisml-team-a-gpu-default has 8 GPUs available; request needs 16",
  "instance": "/api/v1/jobs"
}
```

下游服务返回的 problem 由对应 typed client 解析，必要时映射为 Platform 自己的 `type`，但不丢弃原始 `detail`。

### 4.4 PG Schema

Platform 自有 PG 表仅覆盖 **身份、授权、会话与审计** 四类，不缓存任何下游业务元数据；Tenant 展示元数据写入 cluster-manager（一级字段 + 可选 annotations）；Workspace / Job / Service / Artifact 等业务对象一律向下游服务实时查询。完整 schema 与索引约定见 [database.md §5](../database.md#5-platform)；表清单：`users` / `roles` / `permissions` / `role_permissions` / `user_tenant_roles` / `sessions` / `audit_logs`。

**显式不建的表**（与上游权威重复）：

| 不建表 | 数据归属 | 关联章节 |
| --- | --- | --- |
| `tenants` | cluster-manager `tenants` 表（一级字段含 `name` / `display_name` / `description` / `business_unit` / `quotas` / 软删态） | [§6 租户管理](#6-租户管理) / [cluster-manager.md §3](cluster-manager.md#3-pg-schema) |
| `workspaces` | Compute `services` 表 `kind='workspace'` 的行 | [§8 工作区](#8-工作区) |
| `jobs` / `services` / `resource_pools` / `resource_units` | Compute 同名表 | [§7](#7-资源池与资源单元管理) / [§9](#9-计算任务) / [§10](#10-在线服务) |
| `models` 等制品元数据 | Artifacts `artifacts` 表 | [§13](#13-后续迭代) |

各功能模块不引入新表（全部仅消费上述四类基础表 + 实时透传下游）。

### 4.5 下游 typed client

每个下游服务一个独立子包，对外暴露强类型方法。所有 client 共享同一个工厂与中间件链：

```go
// internal/client/clustermanager/client.go
type Client interface {
    CreateTenant(ctx context.Context, in *CreateTenantInput) (*Tenant, error)
    AddQuota(ctx context.Context, tenant string, in *QuotaSpec) (*Tenant, error)
    // ...
}
```

横切约定：

- **身份透传**：所有出站请求自动携带 `X-Axisml-User: <username>` 头；下游服务（cluster-manager / compute / artifacts）信任此头并只做审计，不做鉴权。
- **超时与重试**：默认 30s 超时；幂等读操作允许有限次数指数退避重试，写操作不自动重试（Platform 把错误透传给前端）。
- **错误映射**：HTTP 4xx → 直接透传 problem；5xx → 包装成 Platform 的 `type=https://axisml.io/errors/upstream-failure`，附带下游服务名。
- **可观测性**：每次调用打 zap 日志 + Prometheus 指标 `platform_upstream_request_total{service,method,status}`。

### 4.6 启动配置

| 配置项 | 默认值 | 说明 |
| --- | --- | --- |
| `--api-bind-address` | `:8080` | HTTP API 监听地址 |
| `--metrics-bind-address` | `:8081` | Prometheus 指标 |
| `--probes-bind-address` | `:8082` | `/healthz` 与 `/readyz` |
| `--db-dsn` | — | PostgreSQL 连接串 |
| `--auth-mode` | `internal` | `internal`（内置用户）/ `oidc`（预留，本期不交付） |
| `--jwt-signing-key-file` | — | JWT 签名密钥文件路径 |
| `--cluster-manager-url` | `http://cluster-manager:8080` | 来自 ConfigMap |
| `--compute-url` | `http://compute:8080` | 来自 ConfigMap |
| `--artifacts-url` | `http://artifacts:8080` | 来自 ConfigMap |
| `--prometheus-url` | `http://kube-prometheus-stack-prometheus.axisml-infra:9090` | Dashboard 数据源 |
| `--workspace-access-jwt-ttl` | `1h` | 工作区访问 JWT TTL（上限 24h） |
| `--service-access-jwt-ttl` | `1h` | 在线服务访问 JWT TTL（上限 24h） |
| `--workspace-default-storage-class` | — | 工作区 PVC 默认 StorageClass；为空则用集群默认 |
| `--audit-log-retention-days` | `90` | 审计日志保留期 |

环境变量同名上标 `AXISML_` 前缀（与现有组件一致，由 `internal/config` 解析）。

### 4.7 镜像与 Makefile

参考 [components/compute/Dockerfile](../../components/compute/Dockerfile) 与 [components/compute/Makefile](../../components/compute/Makefile)：

- 镜像：`ghcr.io/axisml/axisml-platform-backend:<IMAGE_TAG>`，`IMAGE_TAG` 必须等于 [`deploy/helm/axisml-system/Chart.yaml`](../../deploy/helm/axisml-system/Chart.yaml) 的 `appVersion`，由顶层 Makefile 统一注入；
- 多阶段构建从仓库根目录开始；
- 最终镜像基于 `gcr.io/distroless/static:nonroot`，以 `65532:65532` 运行；
- 标准目标：`build` / `test` / `image` / `image-load-minikube` / `clean` / `fmt` / `vet` / `tidy` / `doc-gen` / `integration` / `coverage`。

前端镜像由 [components/platform/frontend/Makefile](../../components/platform/frontend/Makefile) 单独构建，最终通过 Helm 模板部署，与后端松耦合。

---

## 5. 认证与租户权限模型

> 详细字段、API 路径与权限矩阵将由专门的 `auth` 章节细化（本期合并到本节）。

- **第一版**：内置 `users` 表 + bcrypt 密码哈希 + JWT；`sessions` 表存登出黑名单。
- **RBAC**：内置三档角色（`system-admin` / `tenant-admin` / `user`），权限通过 `role_permissions` 多对多绑定。
- **租户绑定**：`user_tenant_roles(user_id, tenant_name, role_id)` 表达「某用户在某租户中的角色」；前端可见的租户列表 = 该用户在 `user_tenant_roles` 中绑定到的所有 `tenant_name`，再调 cluster-manager `LIST tenants` 取展示元数据。
- **OIDC 预留**：`internal/auth/IdentityProvider` 接口屏蔽用户来源；`internal` 实现读 `users` 表，`oidc` 实现按需接入；切换由 `--auth-mode` 控制。本期只交付 `internal`。
- **下游透传**：Platform 校验通过后向下游服务注入 `X-Axisml-User`；下游不做二次鉴权。

### 5.1 内置中间件

`internal/auth` 暴露以下中间件供各功能 handler 拼装：

| 中间件 | 用途 |
| --- | --- |
| `RequireAuthenticated` | 已登录即可 |
| `RequireSystemAdmin` | 仅 `system-admin` |
| `RequireTenantRole(role, tenantParam)` | 在指定路径变量 `tenantParam` 对应的租户上拥有 ≥ `role` 角色；`system-admin` 短路放行 |
| `RequireWorkspaceOwner(idParam)` | `@owner` 或在 workspace 所属租户上有 `tenant-admin` 以上角色；`system-admin` 短路 |
| `RequireServiceOwner(idParam)` | 同上语义，针对在线服务 |
| `RequireJobOwner(tenantParam, nameParam)` | 同上语义，针对计算任务 |

### 5.2 Access JWT 颁发

工作区 / 在线服务两个场景均允许用户向 Platform 索取一次性 JWT 进入受 SecurityPolicy 保护的 HTTPRoute：

| 用途 | `aud` claim | TTL 配置项 |
| --- | --- | --- |
| 主用户登录 | `axisml-platform` | 不可配，默认 12h |
| 工作区接入 | `axisml-workspace` | `--workspace-access-jwt-ttl`，默认 1h |
| 在线服务接入 | `axisml-inference` | `--service-access-jwt-ttl`，默认 1h |

Platform 在 `axisml-system` namespace 内暴露 `/.well-known/jwks.json`（ClusterIP，无需走 Envoy Gateway），SecurityPolicy 通过 cluster-local URL 拉公钥。三种 `aud` 共享签名密钥但严格区分用途，网关校验 `aud` claim 防止跨用途滥用。

### 5.3 角色与权限

| 权限 | `system-admin` | `tenant-admin` | `user` |
| --- | :---: | :---: | :---: |
| 用户 / 角色 / 权限 CRUD | ✅ | ❌ | ❌ |
| 租户 CRUD（创建 / 暂停 / 恢复 / 删除） | ✅ | ❌ | ❌ |
| 租户成员管理 | ✅ | ✅（@self） | ❌ |
| 租户配额 CRUD | ✅ | ✅（@self） | ❌ |
| 资源池 / 资源单元 CRUD | ✅ | ❌ | ❌ |
| 工作区 / Job / Service 创建 | ✅ | ✅（@self） | ✅（@self） |
| 工作区 / Job / Service 启停 / 删 | ✅ | ✅（@self） | ✅（@owner） |
| 制品 CRUD | ✅ | ✅（@self） | ✅（@self） |
| 跨租户读 | ✅ | ❌ | ❌ |

每个功能章节给出本功能的子矩阵；本表是全局视图。

---

## 6. 租户管理

承接 [PRD §6.4.1 租户管理](../../product/prd.md#641-租户管理) 与系统层 [cluster-manager](cluster-manager.md) / [tenant-operator](tenant-operator.md) 之间的 Platform 入口：租户列表 / 详情、租户内成员↔角色绑定、租户管理 UI 与 REST 入口。

| 模块 | 职责 | 边界外 |
| --- | --- | --- |
| Tenant 视图与生命周期 | 系统管理员 CRUD 租户、暂停 / 恢复 / 删除；按用户身份过滤可见租户 | Tenant CR 字段语义、底层 Namespace / ElasticQuota 落地（→ [cluster-manager.md](cluster-manager.md) / [tenant-operator.md](tenant-operator.md)）|
| 配额（详情 Tab 2） | 系统 / 租户管理员对 `Tenant.spec.quotas[]` 的可视化 CRUD | ElasticQuota CR 命名 / 调度行为 |
| 成员（详情 Tab 3） | 本租户内的用户↔角色绑定 UI 与 REST 入口 | 用户身份来源、内置角色定义、跨租户 RBAC 矩阵（→ [§5](#5-认证与租户权限模型)） |

**关键不变式：** Platform 不为租户建任何视图表，权威在 cluster-manager；Platform 仅持有 `user_tenant_roles` 关联，其 `tenant_name` 列直接引用 cluster-manager `tenants.name`（partial unique on `WHERE deleted_at IS NULL`，等价于稳定 FK）。

### 6.1 角色与可见性矩阵

| 能力 | `system-admin` | `tenant-admin@self` | `user@self` |
| --- | :---: | :---: | :---: |
| 列出可见租户 | 全集群 | 仅自己绑定的租户 | 仅自己绑定的租户 |
| 查看租户详情（基本信息） | ✅ | ✅ | ✅ |
| 创建 / 编辑展示元数据 / 暂停 / 恢复 / 删除租户 | ✅ | ❌ | ❌ |
| 查看本租户配额 | ✅ | ✅ | ✅ |
| 新增 / 修改 / 删除本租户配额 | ✅ | ✅ | ❌ |
| 查看本租户成员 | ✅ | ✅ | ❌ |
| 增 / 改 / 删本租户成员 | ✅ | ✅ | ❌ |

`@self` 表示「在该 `{name}` 租户上具备相应角色」。`system-admin` 在所有租户级动作上短路放行。

### 6.2 数据模型（Platform 自有部分）

Platform **不为租户实体建表**。所有租户字段都由 cluster-manager 实时返回；Platform 只持有 `user_tenant_roles` 关联。

#### 6.2.1 租户元数据的归属

cluster-manager API 一级字段与 Platform 视图字段直接 1:1 对应（无 annotation 折叠 / 展开）：

| Platform 字段 | cluster-manager API 字段 | 备注 |
| --- | --- | --- |
| `name`（URL 锚点） | `name` | DNS-1123、≤40 字符；cluster-manager 校验；创建后不可变 |
| `displayName` | `displayName` | 主展示名，允许 Unicode（含中文） |
| `description` | `description` | 自由文本，允许 Unicode；cluster-manager 一级 PG 列，支持 ≤1000 字符 |
| `businessUnit` | `businessUnit` | 业务线 / 部门归属；cluster-manager 一级 PG 列 + 索引，支持列表过滤 |
| `annotations` | `annotations` | 透传给运维使用的扩展位 |
| `namespace` / `quotas` / `initResources` / `suspended` | 同名字段 | 透传 |
| `phase` / `namespaceReady` / `conditions` / `quotas[].used` | 同名字段 | cluster-manager informer 回流到 PG 后由 GET 返回 |
| `createdAt` / `updatedAt` / `deletedAt` | 同名字段 | cluster-manager `tenants` 表自动维护 |

所有 GET 请求都调 `clustermanager.GetTenant(name)` 取最新视图；Platform 端无任何缓存。

> 历史 / 软删租户：通过 `?includeArchived=true` 列出，通过 `POST /api/v1/tenants/{name}/restore` 恢复——由 cluster-manager 原生提供（[cluster-manager.md §6.2](cluster-manager.md#62-端点)），Platform 仅做权限校验后透传。

#### 6.2.2 `user_tenant_roles` 关联

成员管理消费 `user_tenant_roles(user_id, tenant_name, role_id)` 三元组：

- `tenant_name` 直接引用 Tenant `metadata.name`，text 类型，无 RDBMS 外键。
- 因 cluster-manager `tenants.name` 在活跃维度 partial unique 且创建后不可变，等价于稳定 FK。
- 删除租户时由 [§6.5.2](#652-一致性策略与级联) 同步级联清理该租户在 `user_tenant_roles` 中的所有行。

### 6.3 列表 / 详情字段

#### 6.3.1 列表页

| 列 | 来源（cluster-manager API 字段） | 说明 |
| --- | --- | --- |
| `displayName` | `displayName` | 主展示列 |
| `name` | `name` | 字符串锚点；URL 中的 `{name}` 即此列 |
| `business_unit` | `businessUnit` | 列过滤，cluster-manager 端 PG 索引下推 |
| 状态 | `phase` | `Active` / `Suspended` / `Failed` / `Deleted`（`includeArchived=true` 时） |
| 命名空间 | `namespace.name` | 只读展示 |
| 创建时间 | `createdAt` | |
| 操作 | — | 详情 / 暂停 / 恢复 / 删除 / 恢复软删 |

- 过滤：状态、business_unit、关键字（displayName / name 模糊匹配），全部下推 cluster-manager。
- 列表渲染：Platform 后端直接透传 cluster-manager `LIST tenants` 结果，按 RBAC 裁剪。
- 可见性：
  - `system-admin`：全集群所有 Tenant。
  - 其他角色：先按 `user_tenant_roles.user_id = current_user` 取 `tenant_name` 集合，再按集合过滤 cluster-manager 的 LIST 结果。

#### 6.3.2 操作按钮

- **详情** — 任何在该租户上有绑定的角色可点击。
- **暂停 / 恢复** — 仅 `system-admin`；分别调 `POST /api/v1/tenants/{name}/suspend`、`/unsuspend`。
- **删除** — 仅 `system-admin`；调 `DELETE /api/v1/tenants/{name}`，需二次确认弹窗，并展示「同租户成员数 / 当前 phase / 是否 Suspended」。

#### 6.3.3 创建租户表单（system-admin only）

| UI 字段 | cluster-manager API 字段 | 说明 |
| --- | --- | --- |
| `name` | `name` | DNS-1123、≤40 字符；DNS 校验由 cluster-manager 兜底 |
| `displayName` | `displayName` | 主展示名，允许中文 |
| `description` | `description` | 自由文本说明，允许中文；一级 PG 列 |
| `businessUnit` | `businessUnit` | 业务线 / 部门归属，一级 PG 列，支持列表过滤 |
| `namespace.name` | `namespace.name` | 创建后不可变 |
| `namespace.labels` / `annotations` | `namespace.{labels, annotations}` | 仅在 Namespace 首次创建时落地 |
| `quotas[]` | `quotas[]` 透传 | 每条 `(pool, name, min, max)`；`(pool, name)` 创建后不可变 |
| `initResources` | `initResources` 透传 | 列出 `name` + `sourceXxxRef`；UI 不构造源 Secret 数据 |

校验：必填项 + 长度 UI 即时；跨字段（`min ≤ max`、`(pool, name)` 唯一）UI 即时 + cluster-manager 最终；DNS-1123 / namespace denylist 依赖 cluster-manager 4xx 反馈。

### 6.4 详情页 Tab

详情页以 `name` 为维度，分为四个 Tab：基本信息、配额、成员、审计日志。

#### Tab 1 基本信息

展示：`displayName` / `description` / `business_unit`（可编辑）；状态卡片 `phase` / `namespaceReady` / `conditions[]`（只读）；命名空间 `namespace.name`（只读）。

操作（`system-admin`）：编辑展示元数据（单次 PATCH）；暂停 / 恢复；删除。

#### Tab 2 配额

按 `pool` 分组的二级表格。线框图见 [wireframe.md §2.1](../wireframe.md#21-详情页--配额-tab)。

- 数据来源：`GET /api/v1/tenants/{name}/quotas` 实时穿透 cluster-manager；`status.quotas[].used` 直接展示。
- 写权限：`system-admin` / `tenant-admin@self`。
- 不做 sum(max) 上限校验；表头展示「合计 max」聚合视图作肉眼参考。
- 不可变约束：`(pool, name)` 创建后不可变，UI 编辑态置灰；改名 = 先删后建。

#### Tab 3 成员

| 列 | 来源 |
| --- | --- |
| 用户名 / display_name / email | `users` 表 |
| 角色 | `roles.name`，限 `tenant-admin` / `user` |
| 加入时间 | `user_tenant_roles.created_at` |
| 操作 | 改角色 / 移除 |

- **添加成员**：用户搜索 → 角色下拉 → 提交。
- **修改角色**：行内角色下拉。
- **移除成员**：行内移除。
- 写权限：`system-admin` / `tenant-admin@self`。
- 角色限制：仅可绑定 `tenant-admin` / `user`；`system-admin` 由全局用户管理菜单维护。
- 自我保护：当前操作者不能移除 / 降级自己最后一个 `tenant-admin` 角色，否则返回 `409 last-tenant-admin`。

#### Tab 4 审计日志

入口保留，详细字段见 [§13](#13-后续迭代)。

### 6.5 REST API 与模块结构

端点详见 [apis/platform.yaml](../apis/platform.yaml) `Tenants` / `Quotas` / `Members` tag。出站调用 `internal/client/clustermanager` typed client。

#### 6.5.1 业务规则

下列业务规则不在 OpenAPI 中表达，handler 主动校验：

- **DELETE 租户**：先校验 `user_tenant_roles WHERE tenant_name = :name` 行数；非空 → `409 tenant-has-members`，UI 引导先在「成员」Tab 清空。
- **PATCH 租户**：提前拦截不可变字段 `name` / `namespace.name` / `quotas[].(pool, name)`，违反返回 `400 immutable-field`，不触达 cluster-manager。
- **成员绑定**：`role_name` 不允许 `system-admin`（`400 role-not-bindable`）；自我保护见上。
- **配额**：Platform 不做 sum(max) 上限校验。
- **LIST 按角色裁剪**：非 `system-admin` 时先按 `user_tenant_roles.user_id = current_user` 取 `tenant_name` 集合，再按集合过滤；其余 query 参数全部下推到 cluster-manager。

#### 6.5.2 一致性策略与级联

Tenant 实体的权威完全在 cluster-manager；Platform 不缓存任何 Tenant 字段——没有「上游写成功 / 本地写失败」失败路径。

**创建 / 更新**：单点写——映射到 cluster-manager DTO 后单次调用，失败直接 4xx/5xx 透传。

**删除**：
1. 先校验 `user_tenant_roles WHERE tenant_name = :name`；非空 → `409 tenant-has-members`。
2. 调 `clustermanager.Client.DeleteTenant`：失败 → 透传。
3. 成功 → 同事务级联清理 `user_tenant_roles` 中 `tenant_name = :name` 的残留行（兜底）。
4. **孤儿处理**：若 cluster-manager 已返 `404` 但 `user_tenant_roles` 仍有该 tenant 的行，下次 LIST 时按 cluster-manager 结果反向归并，handler 同步清理。

#### 6.5.3 模块结构

目录：`components/platform/backend/internal/tenant/`

| 文件 | 职责 |
| --- | --- |
| `handler.go` | Gin 路由注册（`/api/v1/tenants` 前缀）、请求解析、RBAC gate 装配、problem 渲染 |
| `service.go` | 业务编排：调 cluster-manager、suspend / unsuspend 透传、不可变字段拦截、成员增删校验 |
| `repository.go` | GORM 操作 `user_tenant_roles`（仅 join 与级联清理用） |
| `dto.go` | Platform 请求 / 响应类型；与 cluster-manager API DTO 的显式映射 |
| `mapping.go` | DTO 映射器：Platform DTO ⇄ cluster-manager DTO，1:1 透传 + 命名风格转换 |

RBAC 中间件装配：

| 路由 | 中间件链 |
| --- | --- |
| `POST/PATCH/DELETE /api/v1/tenants[...]`、`POST .../suspend`、`/unsuspend`、`/restore` | `RequireSystemAdmin` |
| `POST/PATCH/DELETE /api/v1/tenants/{name}/quotas[...]` | `RequireTenantRole("tenant-admin", "name")`，`system-admin` 短路 |
| `POST/PATCH/DELETE /api/v1/tenants/{name}/members[...]` | `RequireTenantRole("tenant-admin", "name")`，`system-admin` 短路 |
| `GET /api/v1/tenants/{name}` / `GET .../quotas` | `RequireTenantRole("user", "name")`，`system-admin` 短路 |
| `GET /api/v1/tenants/{name}/members` | `RequireTenantRole("tenant-admin", "name")`，`system-admin` 短路 |
| `GET /api/v1/tenants` | 已登录即可；handler 内部按角色裁剪 |

度量与日志：Prometheus 指标详见 [monitoring.md §5.2](../monitoring.md#52-tenant-模块)。zap 必带 `tenant_name` / `actor_user` / `action` / `status`；删除 / suspend / 成员变更额外带 `target_user` / `role_name`。

### 6.6 测试策略

- **单元**（`internal/tenant/*_test.go`）：DTO 映射器双向往返；不可变字段拦截；最后一个 `tenant-admin` 校验；列表可见性裁剪。
- **integration**（`components/platform/backend/test/integration/`）：testcontainers PG + cluster-manager fake，覆盖 happy path（创建 → 加成员 → 加配额 → suspend → 删除）+ DTO 映射器双向往返；删除孤儿归并；RBAC 矩阵。
- 不引入 envtest：Platform 自身不直读 K8s API；端到端由 cluster-manager / tenant-operator 自身集成层覆盖。

---

## 7. 资源池与资源单元管理

承接 [PRD §6.4.2 资源池管理](../../product/prd.md#642-资源池管理) 与系统层 [compute](compute.md) 之间的 Platform 入口。

资源池（ResourcePool）与资源单元（ResourceUnit）合并在同一章节：菜单只有「资源池管理」，§6.4.2 把「维护池下的资源单元规格档位」作为该入口的下属能力；compute 也以子路径 `/api/v1/resource-pools/{pool}/resource-units` 表达父子关系。

| 模块 | 职责 | 边界外 |
| --- | --- | --- |
| 资源池视图与生命周期 | 系统管理员 CRUD 资源池；所有已登录用户可读 | ResourcePool PG 字段语义、注入规则（→ [compute.md §4](compute.md#4-resourcepool)）；Node label / taint 维护（→ kubectl 手工） |
| 资源单元 | 系统管理员在池上下文中 CRUD 资源单元；所有已登录用户可读 | ResourceUnit 字段语义、命名约定、注入合并规则（→ [compute.md §5](compute.md#5-resourceunit)）|

**关键不变式：** Platform 不引入资源池 / 资源单元相关表；所有操作实时穿透 compute。集群侧 Node label / taint 由管理员通过 kubectl 维护，UI 不下发。

### 7.1 角色与可见性矩阵

| 能力 | `system-admin` | `tenant-admin` | `user` |
| --- | :---: | :---: | :---: |
| 列出资源池 / 资源单元 | ✅ | ✅ | ✅ |
| 查看资源池 / 资源单元详情 | ✅ | ✅ | ✅ |
| 创建 / 编辑 / 删除资源池 | ✅ | ❌ | ❌ |
| 创建 / 编辑 / 删除资源单元 | ✅ | ❌ | ❌ |

资源池 / 资源单元都是全集群对象，Platform 不做「按租户过滤可见性」。读权限只要求已登录身份，是因为 Job / Service / Workspace 提交表单需要对所有用户暴露下拉。

### 7.2 数据模型（Platform 自有部分）

Platform 自身**不为资源池或资源单元引入任何 PG 表**。所有字段权威以 compute 为准：

- 资源池字段：见 [compute.md §4.2 `resource_pools`](compute.md#42-数据模型)。
- 资源单元字段：见 [compute.md §5.2 `resource_units`](compute.md#52-数据模型)。

操作审计写入 Platform 通用 `audit_logs` 表（[§4.4](#44-pg-schema)），按 `target` 字段区分 `resource-pool/<name>` 与 `resource-unit/<pool>/<name>`。

### 7.3 列表 / 详情字段

#### 7.3.1 资源池列表页

| 列 | 来源 | 说明 |
| --- | --- | --- |
| `name` | compute `resource_pools.name` | 主展示列；URL 锚点 |
| `description` | compute `resource_pools.description` | 自由文本 |
| 节点选择器摘要 | compute `resource_pools.node_selector` | 取前两个 key 拼成 `k1=v1, k2=v2, …` |
| 资源单元数 | 聚合 `GET /resource-units?limit=0` | 见 [§7.3.3](#733-资源单元数量聚合) |
| 创建时间 | compute `resource_pools.created_at` | |
| 操作 | — | 详情 / 删除 |

- 过滤：关键字（`name` / `description` 模糊匹配）、`node_selector` key 命中。
- 可见性：所有已登录用户。

#### 7.3.2 操作按钮

- **详情** — 任何已登录用户可点击。
- **删除** — 仅 `system-admin`；调 `DELETE /api/v1/resource-pools/{pool}`，需二次确认弹窗。前端展示「池下资源单元数 / 引用此池的活跃 Job / Service 数」摘要（来自 [§7.5.1](#751-业务规则)）。

#### 7.3.3 资源单元数量聚合

资源池数量预期 < 50（每个机型 / 用途各一池），首版策略：

1. Platform 后端 `pool_service.List` 收到 compute 返回的 pool 列表后，按池并发触发 `compute.ListResourceUnits(pool, limit=0)` 拿 count。
2. 拼装到响应 DTO 的 `resource_unit_count` 字段。
3. 单池失败不阻塞整体响应：失败池 `resource_unit_count = -1`，前端显示 `—`。

后续若池规模膨胀或调用频次增加，再引入 5–10 秒级 in-process LRU 缓存。

#### 7.3.4 创建资源池表单（system-admin only）

| UI 字段 | 写入位置 | 说明 |
| --- | --- | --- |
| `name` | `resource_pools.name` | DNS-1123、≤40 字符；校验由 compute 兜底 |
| `description` | 同名列 | 自由文本 |
| `node_selector` | 同名列 | key/value 列表 UI；空对象表示「整集群可用」 |
| `tolerations` | 同名列 | JSON 编辑器 + 常见模板（`nvidia.com/gpu`、`axisml.io/dedicated`） |
| `metadata` | 同名列 | 可选 jsonb，留作扩展（如 `cost_per_hour` 后续迭代） |

辅助 UI：节点匹配预览（仅字符串拼装，不查 K8s）；管理提示「集群侧需提前给目标节点打 label / taint，可在 kubectl 中执行」。

### 7.4 详情页 Tab

详情页以 `name` 为维度，分为四个 Tab：基本信息、资源单元、节点匹配预览（占位）、审计日志（占位）。

#### Tab 1 基本信息

展示：`name` / `description` / `node_selector` / `tolerations` / `metadata`、创建 / 更新时间。

操作（`system-admin`）：编辑（允许改 `description` / `node_selector` / `tolerations` / `metadata`；`name` 不可变）；删除。

#### Tab 2 资源单元（完整 CRUD）

按池上下文管理资源单元，是 ResourceUnit 在 Platform UI 中的唯一入口。版面占位见 [wireframe.md §3.1](../wireframe.md#31-详情页--资源单元-tab)。

**创建资源单元抽屉表单**：

| 字段 | 写入位置 | 说明 |
| --- | --- | --- |
| `name` | `resource_units.name` | 遵循 [compute.md §5.3](compute.md#53-命名约定) 命名约定 `<accelerator>[-<count>x]-<tier>[-<variant>]`；UI 旁附常见模板按钮 |
| `description` | 同名列 | 自由文本 |
| `requests` | 同名列 | 表格 UI：CPU / Memory / GPU 三个常用字段独立行 + 「自定义资源」追加按钮 |
| `limits` | 同名列 | 同 `requests`；默认与 `requests` 相同 |
| `node_selector` | 同名列 | key/value 列表 UI |

提交后调 `POST /api/v1/resource-pools/{pool}/resource-units`。

**行内编辑**：允许改 `description` / `requests` / `limits` / `node_selector`；`name` 不可变；`pool` 不可变。

**删除**：行内删除按钮，二次确认。Platform 后端 DELETE 前置校验返回「使用此 unit 的活跃 Job / Service 数量」摘要。

**合并节点选择器预览**：资源单元 row 展开后展示 `pool.node_selector` ⊕ `unit.node_selector`（Pool 优先，详见 [compute.md §5.4](compute.md#54-注入规则)）合并后的最终 selector，帮助管理员在创建任务前校验意图。

#### Tab 3 节点匹配预览

入口保留，详情见 [§13](#13-后续迭代)。需要 Platform 直读 K8s Node 才能落地。

#### Tab 4 审计日志

入口保留，详细字段见 [§13](#13-后续迭代)。

### 7.5 REST API 与模块结构

端点详见 [apis/platform.yaml](../apis/platform.yaml) `ResourcePools` / `ResourceUnits` tag。出站调用 `internal/client/compute` typed client。

#### 7.5.1 业务规则

- **LIST**：默认列出所有池（无租户过滤）；响应每条记录追加 `resource_unit_count`。
- **PATCH**：允许字段 `description` / `node_selector` / `tolerations` / `metadata`；`name` 拦截 `400 immutable-field`，不触达 compute。
- **DELETE 资源池前置校验**（Platform 自做，不依赖 compute 级联）：
  1. `compute.ListResourceUnits(pool)`：池下资源单元数 > 0 → `409 pool-in-use`，problem `type=https://axisml.io/errors/pool-in-use`，`blockers.resource_units` 给出计数；
  2. `compute.ListJobs(filter={pool, active})` + `compute.ListServices(filter={pool, active})`：活跃（非终态）数 > 0 → `409 pool-in-use`，`blockers.active_jobs` / `blockers.active_services` 各列若干示例 name；
  3. 通过 → 调 `compute.DeleteResourcePool`。
  > **依赖**：上述 LIST API 需支持按 `pool_id` / `active` 过滤；如 compute 当前未提供，作为 compute 侧 API 补丁项跟踪。
- **DELETE 资源单元**：前置校验使用此 unit 的活跃 Job / Service 数 > 0 → `409 unit-in-use`，结构与 pool DELETE 一致。
- **错误透传**：DNS-1123 / 命名约定 / `(pool, name)` 重复 / 不可变字段依赖 compute 4xx 透传；compute 5xx 包装为 `https://axisml.io/errors/upstream-failure`。

#### 7.5.2 模块结构

目录：`components/platform/backend/internal/resourcepool/`

| 文件 | 职责 |
| --- | --- |
| `pool_handler.go` | 资源池 Gin 路由（`/api/v1/resource-pools`）、请求解析、RBAC gate 装配、problem 渲染 |
| `pool_service.go` | 资源池业务编排：透传 compute、删除前置校验 |
| `unit_handler.go` | 资源单元 Gin 路由（`/api/v1/resource-pools/{pool}/resource-units`） |
| `unit_service.go` | 资源单元业务编排：透传 compute、删除前置校验 |
| `dto.go` | 请求 / 响应类型；DTO 增 `resource_unit_count` 字段 |

无 `repository.go`。资源池与资源单元同包，便于共享 typed client 与 problem 映射函数。

RBAC 中间件：

| 路由 | 中间件链 |
| --- | --- |
| `POST/PATCH/DELETE /api/v1/resource-pools[...]` | `RequireSystemAdmin` |
| `POST/PATCH/DELETE /api/v1/resource-pools/{pool}/resource-units[...]` | `RequireSystemAdmin` |
| `GET /api/v1/resource-pools[...]` | `RequireAuthenticated` |
| `GET /api/v1/resource-pools/{pool}/resource-units[...]` | `RequireAuthenticated` |

一致性与代理：写路径业务校验通过后调 compute；DELETE 先聚合查询计数。读路径直接透传 compute；LIST 资源池在 handler 内并发聚合 `resource_unit_count`。所有出站请求自动注入 `X-Axisml-User`。

度量与日志：Prometheus 指标详见 [monitoring.md §5.6](../monitoring.md#56-resourcepool-模块)。zap 必带 `pool_name` / `actor_user` / `action` / `status`；资源单元额外带 `unit_name`；删除阻断额外带 `block_reason ∈ {resource-units, active-jobs, active-services}` 与 `child_count`。

### 7.6 测试策略

- **单元**：service 层删除前置校验逻辑（含 compute 部分失败的降级行为）；不可变字段拦截；RBAC 中间件分支；列表页 ResourceUnit 数量并发聚合（含部分池失败 → `resource_unit_count = -1`）。
- **integration**：testcontainers PG（仅审计日志）+ in-process compute fake；happy path（建池 → 加 unit → 删 unit 受阻断 → 改 unit → 删除清空 → 删池）；故障注入（compute 4xx / 5xx 透传）；RBAC 矩阵。
- 不引入 envtest：Platform 自身不直读 K8s API。

---

## 8. 工作区

承接 [PRD §6.2.1 工作区](../../product/prd.md#621-工作区) 与系统层 [compute](compute.md) / [compute-operator](compute-operator.md) 之间的 Platform 入口：工作区列表 / 详情、创建与启停、浏览器接入与持久化目录管理。

工作区即 PRD 所称的「工作区」——一台 **长驻的交互式开发容器**（jupyter / VSCode Server / 自定义 Web 服务）。本设计不引入新 CRD，整套语义复用 [`MLService(native, deployment)`](compute-operator.md#561-native-deployment)：创建 = 调 Compute 创建一个单 role 单副本的 MLService；启停 = patch `roles[0].replicas` 在 `0/1` 切换；浏览器接入 = `spec.route` 派生 HTTPRoute 经 Envoy Gateway 反代。

| 模块 | 职责 | 边界外 |
| --- | --- | --- |
| 工作区视图与生命周期 | 创建 / 列表 / 详情 / 删除；按用户身份过滤可见工作区 | MLService CR 字段语义、Pod 调度（→ [compute.md §7](compute.md#7-service) / [compute-operator.md §5](compute-operator.md#5-mlservice-controller)）|
| 启停 | 用户视角 start / stop 翻译为 Compute `/scale` 写 `replicas=1/0` | Compute 双 hash 同步语义（→ [compute.md §3.5](compute.md#35-desiredapplied-spec-hash-双-hash-机制)）|
| 浏览器接入 | 派生 `spec.route` 让 handler 创建 HTTPRoute；下发短 TTL JWT | HTTPRoute / SecurityPolicy 资源渲染（→ [compute-operator.md §5.5 / §5.6.1](compute-operator.md#55-specroute-派生资源)）|
| 持久化目录 | 创建 / 销毁 workspace 专属 PVC；MVP 默认挂 `/workspace` | StorageClass 选择、CSI driver 行为 |

**关键不变式：** Platform 不为工作区建任何 PG 表。「这个 service 是不是工作区」由 Compute `services.kind ∈ {service, workspace}` 列直接表达；其他所有字段全部由 Compute service 行承载。寻址用 Compute `services.id`（uuid）；Platform 列表 = `compute.ListServices(namespace=<tenant_ns>, kind=workspace)`。Workspace 不抽象「IDE 模板」——镜像本身决定它跑什么、监听哪个端口。

### 8.1 角色与可见性矩阵

`@self` 表示「在该 workspace 所属租户上具备相应角色」；`@owner` 表示「该 workspace 的 owner_user 等于当前用户」。

| 能力 | `system-admin` | `tenant-admin@self` | `user@self & @owner` | `user@self & 非 owner` |
| --- | :---: | :---: | :---: | :---: |
| 列出可见工作区 | 全集群 | 该租户全部 | 仅自己创建的 | 仅自己创建的 |
| 查看工作区详情 | ✅ | ✅ | ✅ | ❌ |
| 创建工作区 | ✅ | ✅ | ✅ | — |
| 启动 / 停止 | ✅ | ✅ | ✅ | ❌ |
| 删除 | ✅ | ✅ | ✅ | ❌ |
| 拿 access JWT 进入 IDE | ✅ | ✅ | ✅ | ❌ |
| 修改展示元数据 | ✅ | ✅ | ✅ | ❌ |

### 8.2 数据模型（Platform 自有部分）

**Platform 不为工作区建任何 PG 表**。「这是工作区」由 Compute `services.kind='workspace'` 直接表达。

#### 8.2.1 工作区元数据的归属

| 字段 | 写入位置 | 备注 |
| --- | --- | --- |
| 工作区身份 | Compute `services.id`（uuid） | URL 路径中的 `{id}` 即此列；同时 deterministic 派生 PVC 名 `axisml-ws-<id8>-data`（取 id 前 8 字符） |
| 「这是工作区」 | Compute `services.kind = 'workspace'` | Platform 创建时显式传 |
| 租户归属 | Compute `services.namespace` | 即 Tenant CR `spec.namespace.name`；RBAC 裁剪从 `user_tenant_roles.tenant_name → Tenant.spec.namespace.name` 解析一次 |
| MLService CR 名 | Compute `services.name` | Platform 创建时生成 `mlservice_name = "ws-" + crockford32(rand40bit)`；同 namespace 内冲突重试 |
| 显示名 / 描述 / Owner | Compute `services.{display_name, description, owner_user}` | `owner_user` 来自 `X-Axisml-User` 头注入 |
| Pool / Unit | Compute `services.{pool_id, resource_unit_id}` | |
| 镜像 / 端口 / 启动命令 / env | Compute `services.spec.roles[0].template.*` | |
| 配额（ElasticQuota CR 名） | Compute `services.spec.scheduling.quota` | |
| HTTPRoute 路径 | Compute `services.spec.route.path` | Platform 创建时拼 `/workspaces/<tenant>/<service.name>/` |
| 副本 / 就绪副本 / 端点 / 状态 | Compute `services.{replicas, ready_replicas, endpoint, status, message}` | Informer 回流 |
| `desired_state` / 工作区展示态 `status` | 派生：见 [§8.5.3](#853-状态读取与派生) |
| PVC 名 | 派生：`"axisml-ws-" + service.id 前 8 字符 + "-data"` | 无需存储 |
| PVC 容量 / 已用 | 详情页实时 `kubectl get pvc` + Prometheus（可选） | |
| Access URL | 派生：`https://<gateway-host>/workspaces/<tenant>/<service.name>/` | |
| 创建 / 更新时间 | Compute `services.{created_at, updated_at}` | |

#### 8.2.2 列表查询路径

针对租户管理员 / 系统管理员的「列出整个租户的工作区」与普通用户的「列出我的工作区」都走同一段流程：

1. 按 RBAC 取可见 `tenant_name` 集合；
2. 并行 `clustermanager.GetTenant(name)` 解析每个 `compute_namespace = Tenant.spec.namespace.name`（request-scoped memoize）；
3. 并行 `compute.ListServices(namespace=<compute_namespace>, kind=workspace, ownerUser?)`；
4. 对普通用户由 Compute 端的 `ownerUser=<current_user.username>` 过滤直接下推；
5. 返回带派生 `status` / `desired_state` / `access_url` 的 Workspace DTO 列表。

整条路径完全无本地表 / 无 join；`kind=workspace` 过滤一次性把工作区与普通在线服务分开，列表 N+1 安全。

#### 8.2.3 对 core 层的硬依赖（必须同 PR 推进）

##### A. MLService spec 加 `volumes` / `volumeMounts`（PVC 持久化）

| 文件 / 章节 | 改动 |
| --- | --- |
| [compute-operator.md §5.2.2](compute-operator.md#522-spec-结构) | 在 MLService `spec.roles[*].template` 下追加 `volumes[]` / `volumeMounts[]`，与 K8s `PodSpec` / `Container` 同源 |
| [compute-operator.md §5.2.3](compute-operator.md#523-字段归属与不可变性) | 新增字段标 `用户提交 / 否` |
| [compute-operator.md §5.6.1](compute-operator.md#561-native-deployment) | 通用字段映射表追加 `roles[].template.volumes` → `Deployment.spec.template.spec.volumes`；`roles[].template.volumeMounts` → 主容器 `volumeMounts` |
| [compute.md §7.2](compute.md#72-数据模型) | `spec` jsonb 接受新增 `volumes` / `volumeMounts` 字段透传 |
| [compute.md §7.4.1](compute.md#741-提交校验) | 增补「`volumeMounts.name` 必须在 `volumes[]` 中存在」校验 |

##### B. Compute service 加 `kind` 列 + id-based 寻址 / kind 过滤

| 文件 / 章节 | 改动 |
| --- | --- |
| [compute.md §7.2](compute.md#72-数据模型) | services 表追加 `kind text NOT NULL DEFAULT 'service'`（`'service' \| 'workspace'`）；创建后不可变 |
| [compute.md §7.5](compute.md#75-id-based-寻址端点--kind-过滤) | `POST /api/v1/namespaces/{ns}/services` 请求体接受 `kind`；`GET /api/v1/services/{id}` 返回 `kind`；`GET /api/v1/services?namespace=&kind=workspace` 支持过滤；所有 namespace-scoped LIST 支持 `?kind=` 过滤 |

写操作（`POST /scale`、`DELETE`）保留既有 namespace-scoped 形态——Platform 在写之前先 id-based GET 拿到 `(namespace, name)` 再走原路径。

##### C. 镜像运行约束（约定）

Workspace 经 `https://<gateway>/workspaces/<tenant>/<service.name>/...` 访问，HTTPRoute **不做路径重写**；容器内的 web server 必须能在该前缀下工作：

- jupyter 启动需带 `--ServerApp.base_url=/workspaces/<tenant>/<service.name>/`；
- code-server 启动需带 `--abs-proxy-base-path /workspaces/<tenant>/<service.name>`；
- 自定义服务自负。

为减少用户负担，Platform 在创建 MLService 时给容器注入：

```
AXISML_WORKSPACE_BASE_URL=/workspaces/<tenant>/<service.name>/
```

约定 workspace 镜像的 entrypoint 读取该环境变量并自动配置 base-url；非约定镜像由用户在「启动命令覆盖」字段中显式指定。Workspace 镜像目录由系统管理员在 Artifacts 镜像中心维护。

### 8.3 列表 / 创建表单

#### 8.3.1 列表页

| 列 | 来源 | 说明 |
| --- | --- | --- |
| 显示名 | Compute `services.display_name` | 主展示列；点击进详情 |
| 镜像 | Compute `services.spec.roles[0].template.image` | 列过滤 |
| 资源池 · 单元 | Compute `services.pool_id` + `services.resource_unit_id` | 联表 ResourcePool / ResourceUnit 取展示名 |
| 状态 | 派生自 `(desired_state, services.status)`（[§8.5.3](#853-状态读取与派生)） | `Stopped` / `Starting` / `Running` / `Degraded` / `Failed` / `Deleting` / `Deleted` |
| Owner | Compute `services.owner_user` | 仅 admin 可见列 |
| 创建时间 | Compute `services.created_at` | |
| 操作 | — | 启动 / 停止 / 打开 / 详情 / 删除 |

可见性：
- `system-admin`：跨所有 compute namespace 并行 LIST（`kind=workspace`）；
- `tenant-admin@self`：RBAC 可见租户对应的 compute namespace 集合；
- `user`：上一条基础上由 Compute 端 `ownerUser=<current_user.username>` 过滤下推。

操作按钮：
- **启动**（`Stopped` 状态）：调 `POST /api/v1/workspaces/{id}/start`；按钮 loading 至 `status` 转 `Running` 或 `Failed`。
- **停止**（`Running` / `Degraded` / `Starting` 状态）：调 `POST /api/v1/workspaces/{id}/stop`；二次确认提示「容器内 `/workspace` 之外的目录会丢失，PVC 数据保留」。
- **打开**（`Running` 状态）：前端先调 `GET /api/v1/workspaces/{id}/access` 拿 `{url, jwt}`，再用 `Authorization: Bearer <jwt>` 头通过新 tab 打开 `url`。
- **删除** — owner / `tenant-admin@self` / `system-admin`；二次确认弹窗，body `{deletePvc: bool}`，默认勾选「同时删除 PVC」。

#### 8.3.2 创建表单

| UI 字段 | 写入位置 | 说明 |
| --- | --- | --- |
| 显示名 | Compute `services.display_name` | 必填；同一 owner 在同一租户下不重名 |
| 描述 | Compute `services.description` | 可选 |
| 租户 | Platform RBAC + 调用上下文（解析为 `compute_namespace`） | 默认当前活跃租户；`system-admin` 可选任意租户 |
| 镜像 | MLService `spec.roles[0].template.image` | 从 Artifacts `kind=image` 中选 + 可手填 OCI URI；后端解析校验能 resolve |
| 容器端口 | MLService `spec.roles[0].template.ports[0].containerPort` | 必填整数 ∈ [1, 65535]；用户应知道镜像内 web server 监听端口（jupyter=8888、code-server=8080） |
| 启动命令 / 参数 | MLService `spec.roles[0].template.command` / `args` | 可选；不填则使用镜像 entrypoint（依靠 `AXISML_WORKSPACE_BASE_URL` 自配置 base-url） |
| 环境变量 | MLService `spec.roles[0].template.env` | 可选；Platform 在末尾追加 `AXISML_WORKSPACE_BASE_URL` |
| 资源池 / 资源单元 | MLService `spec.scheduling.*` + Compute services `pool_id` / `resource_unit_id` | unit 必须属于 pool |
| 配额 | 拼接 ElasticQuota 名 `axisml-<tenant>-<pool>-<quota>` 写入 `spec.scheduling.quota` | 与 [§3.1](#31-用户提交训练任务) 一致 |
| PVC 大小 | 创建 PVC 时 `spec.resources.requests.storage` | 默认 `20Gi`；上限由 Platform 配置项约束 |

校验：DNS-1123 显示名、长度 ≤ 40 字符；镜像必须能 `artifacts.Resolve`；ResourceUnit 必须属于所选 ResourcePool；`containerPort` 范围 `[1, 65535]`；跨字段约束 / 配额穿透校验以 Compute 返回为准。

**不引入「IDE 模板」抽象**：镜像本身决定 workspace 跑什么。镜像中心可由系统管理员维护一组「常用基础镜像」（如 `system/image/jupyter-cpu`、`system/image/code-server`），让用户在选镜像时下拉看到——但这只是 Artifacts 侧的镜像目录管理，不是 Workspace 模型的一部分。

### 8.4 详情页 Tab

详情页以 `id` 为维度，分为五个 Tab：概览、访问、事件、日志、审计。

#### Tab 1 概览

展示：基本信息（显示名 / 描述 / Owner / 镜像 / 资源池 · 单元；可编辑 `display_name` / `description`）；当前状态卡片（派生 `status` + `replicas` / `ready_replicas` + `services.message`）；启停时间（阶段一取自 `services.updated_at`）；PVC 信息（`pvc_name` / `pvc_size` / `pvc_used`）；创建时间 / 创建者 / Tenant。

操作：编辑展示元数据（owner / admin）；启动 / 停止 / 删除。

#### Tab 2 访问

展示：Access URL（`https://<gateway>/workspaces/<tenant>/<service.name>/`）；JWT 用法说明；port-forward 命令模板（方便习惯命令行的用户）：

```
kubectl -n <compute_namespace> port-forward svc/<service.name> 8888:<containerPort>
```

操作：打开（拿 access JWT 后新 tab 打开）；轮换 access JWT（调一次 `GET .../access` 即拿到新 1h 有效 token，短 TTL 自然失效，不维护「禁用上一个」状态机）。

#### Tab 3 事件

透传 Compute service events（`GET /api/v1/namespaces/{ns}/services/{svc}/events`）。Compute 当前 [§6.4.5](compute.md#645-副本与事件端点) 仅给 Job 描述了 `/events` 端点，service 端的 `/events` 端点扩展登记到 [§13](#13-后续迭代)。MVP 阶段 Tab 3 退化为「占位 + 跳到 K8s Events」（系统管理员可看），普通用户暂留空。

#### Tab 4 日志

MVP 不交付。登记到 [§13](#13-后续迭代)：依赖 Compute service 端 `/logs` 端点扩展。短期建议用户用 `kubectl logs deploy/<service.name>` 命令。

#### Tab 5 审计

入口保留，详细字段见 [§13](#13-后续迭代)。

### 8.5 REST API 与模块结构

端点详见 [apis/platform.yaml](../apis/platform.yaml) `Workspaces` tag。出站调用 `internal/client/compute` typed client；通过 `GetServiceByID` / `ListServices(namespace, kind=workspace, ...)` 对接 [§8.2.3 B](#822-列表查询路径) 的 id-based + kind 过滤端点。

#### 8.5.1 创建工作区写入顺序

POST `/workspaces` 涉及 MLService + PVC 双写，handler 必须按下列顺序：

1. RBAC + ResourcePool / Unit / Image 解析校验；通过 `clustermanager.GetTenant(tenantId)` 解析 `compute_namespace`；
2. 生成 `mlservice_name = "ws-" + crockford32(rand40bit)` 作为 desired CR 名；
3. 调 `compute.CreateService(compute_namespace, body)`，body 中 `kind="workspace"`、`spec.roles[0]` 注入 `volumes/volumeMounts/ports` 与 `AXISML_WORKSPACE_BASE_URL` env、`spec.route` 启用 JWT 鉴权；
4. Compute 同步返回 service 对象（含 Compute 生成的 `id` uuid）；
5. 由 service.id 派生 `pvc_name = "axisml-ws-" + service.id 前 8 字符 + "-data"`；Platform 直连 K8s 创建 PVC；
6. PVC 创建失败 → 调 `compute.DeleteService(...)` 回滚 MLService，返 5xx。

> **顺序倒置注记**：先建 MLService 再建 PVC——PVC 名 deterministic 派生自 service.id，service.id 由 Compute 生成；MLService 已存在但 PVC 未就绪的时间窗口内 deployment 第一次拉起会 Pending，PVC 就位后自然 Running。可接受。

#### 8.5.2 业务规则

- **PATCH**：仅可改 `displayName` / `description`；其他字段不可变，变更需「先删后建」。
- **DELETE**：请求体可选 `{ "deletePvc": true }`（默认 `true`）。先 `compute.GetServiceByID(id)`；404 → 直接返 200（幂等）；`kind != 'workspace'` → 返 `404`（避免误删普通在线服务）；删除 MLService 后视 `deletePvc` 决定是否删 PVC。
- **start / stop**：与 PRD「不用时停止释放资源」语义贴近，故用显式 `/start` `/stop` 而非通用 `/scale`；底层翻译到 `compute.ScaleService(ns, name, {replicas: 1|0})`。幂等保护：start 时 `replicas == 1` 直接返 200；stop 时 `replicas == 0` 直接返 200；`Deleted` / `Deleting` 状态拒绝 → `409 workspace-deleted`。
- **access**：返回 `{ url, jwt, headerName, expiresAt }`。JWT claim：`iss=axisml-platform`，`aud=axisml-workspace`，`sub=<current-user-id>`，`workspace=<id>`，`service_id=<service_id>`，`exp=now+1h`。
- **events / logs**：MVP 期间返回 `501 Not Implemented` + problem `type=https://axisml.io/errors/upstream-not-ready`。

#### 8.5.3 状态读取与派生

- 任何展示运行态字段的请求都调 `compute.GetServiceByID(id)`，不本地缓存；不引入 K8s informer。
- 列表走 `compute.ListServices(namespace, kind=workspace)` 端点，单租户百级 workspace 一次 RPC 拉完。
- 派生 `status` 规则（输入只剩 `(services.status, services.replicas, services.ready_replicas)`）：

| 条件 | Workspace `status` |
| --- | --- |
| `services.status == Creating` | `Creating` |
| `services.status == Deleting` | `Deleting` |
| `services.status == Deleted` | `Deleted` |
| `services.replicas == 0` 且 status ∉ {Creating, Deleting, Deleted} | `Stopped` |
| `services.replicas > 0 && services.status == Pending` | `Starting` |
| `services.replicas > 0 && services.status == Ready` | `Running` |
| `services.replicas > 0 && services.status == Degraded` | `Degraded` |
| `services.replicas > 0 && services.status == Failed` | `Failed` |

`Failed` / `Degraded` 是 Compute 的非终态——operator 自愈后下一次 GET 自然回到 `Running`。

#### 8.5.4 一致性策略（PVC + MLService）

Compute / K8s 是权威，且 Platform 端不持有任何本地表——「上游写成功 / 本地写失败」失败路径不存在。

**创建**（按 [§8.5.1](#851-创建工作区写入顺序) 顺序）：
1. MLService 创建失败 → 直接 4xx / 5xx 透传，无 PVC、无副作用；
2. MLService 创建成功后 PVC 创建失败 → 立即 `compute.DeleteService` 回滚 MLService，返 5xx；如果 delete MLService 也失败，登记到 `audit_logs` 由 system-admin 介入；
3. 二者均成功 → 返回 `Workspace` DTO。

**删除**：
1. `compute.GetServiceByID(id)`：若 `404` 或 `kind != 'workspace'` 返 `404`（幂等保护 + 防误删）；
2. 调 `compute.DeleteService` 失败 → 4xx / 5xx 透传；成功后继续；
3. PVC delete 失败 → 留作孤儿，登记到 [§13](#13-后续迭代)。

**启停**：单点写 Compute `/scale`，无跨对象副作用；失败直接透传 4xx / 5xx，无补偿。

**孤儿处理**：所有「这是工作区吗」的判定都依据 Compute `services.kind`——Platform 端没有本地行可能孤立。

#### 8.5.5 PVC 管理

- Platform 后端持有受限 K8s ServiceAccount，命名 `axisml-platform-pvc`，权限矩阵：
  ```
  apiGroups: [""]
  resources: ["persistentvolumeclaims"]
  verbs: ["get", "list", "create", "delete"]
  ```
  作用域：所有 compute namespace（即 Tenant CR 派生的 namespace）；通过 ClusterRole + 多 RoleBinding 落地。
- PVC spec：
  ```yaml
  spec:
    accessModes: [ReadWriteOnce]
    resources:
      requests:
        storage: <pvcSize>
    # storageClassName: 留空 → 用 cluster 默认；可由 --workspace-default-storage-class 覆盖
  ```
- 命名：`axisml-ws-<id 前 8 字符>-data`（deterministic）；同租户内不会撞名。
- 删除策略：
  - `deletePvc=true`：workspace DELETE 时立即调 `kubectl delete pvc`；
  - `deletePvc=false`：保留 PVC；进「孤儿 PVC」管理界面（[§13](#13-后续迭代)）。

#### 8.5.6 模块结构

目录：`components/platform/backend/internal/workspace/`

| 文件 | 职责 |
| --- | --- |
| `handler.go` | Gin 路由注册（`/api/v1/workspaces` 前缀）、请求解析、RBAC gate 装配、problem 渲染 |
| `service.go` | 业务编排：创建（MLService → PVC，PVC 失败回滚 MLService）、启停、删除、access JWT 颁发 |
| `dto.go` | 请求 / 响应类型；与 Compute API DTO 的显式映射 |
| `view.go` | `Workspace` DTO：`compute.GetServiceByID` 结果 + 可选 PVC 数据 → 单一 DTO；含派生 `status` / `desired_state` / `access_url` 的纯函数 |
| `name.go` | `mlservice_name` 生成（crockford-base32 编码 40 位随机数；同 namespace 内冲突重试） |
| `pvc.go` | 受限 K8s client 创建 / 读取 / 删除 PVC；与 `service.go` 解耦以便单测 |
| `jwt.go` | access JWT 颁发；签名密钥与 [§5](#5-认证与租户权限模型) 复用 |

无 `repository.go`：Platform 不持有任何工作区相关 PG 表。无 `template.go`：本设计不引入「IDE 模板」抽象。

RBAC 中间件：

| 路由 | 中间件链 |
| --- | --- |
| `GET /api/v1/workspaces` | 已登录即可；handler 内部按角色裁剪可见集合 |
| `POST /api/v1/workspaces` | `RequireTenantRole("user", "<body.tenantId>")`，`system-admin` / `tenant-admin@self` 短路 |
| `GET /api/v1/workspaces/{id}` | `RequireWorkspaceOwner("id")`；语义：`@owner` 或在 workspace 所属租户上具备 `tenant-admin` 以上角色；`system-admin` 短路 |
| `PATCH /api/v1/workspaces/{id}`、`DELETE`、`POST .../start`、`POST .../stop`、`GET .../access` | 同上 |
| `GET .../events` / `GET .../logs` | 同上 |

度量与日志：Prometheus 指标详见 [monitoring.md §5.3](../monitoring.md#53-workspace-模块)。zap 必带 `service_id` / `tenant_name` / `actor_user` / `action` / `status`；启停 / 删除额外带 `target_replicas` / `delete_pvc`。

### 8.6 测试策略

- **单元**（`internal/workspace/*_test.go`）：
  - `view.go` 状态派生函数（[§8.5.3](#853-状态读取与派生) 的 8 个分支全覆盖）；
  - `name.go` mlservice_name 生成与冲突重试；
  - `service.go` 创建 / 启停的幂等性（重复 start `replicas==1` → 200）；
  - DELETE 防误删：`kind != 'workspace'` 返 404；
  - RBAC 中间件分支（`system-admin` 短路、`@owner` 比对、跨租户拒绝）；
  - access JWT claim 字段完整性。
- **integration**：testcontainers PostgreSQL（仅 `user_tenant_roles` / `audit_logs`）+ envtest（用于创建 PVC）+ in-process gin engine + httptest fake Compute（按 [§8.2.3 B](#823-对-core-层的硬依赖必须同-pr-推进) 模拟）；happy path（创建 jupyter 镜像 → MLService Ready → 停止 → MLService `replicas=0` → 重新启动 → PVC 数据保留 → 删除）；故障注入；RBAC 矩阵；列表 N+1 保证。
- 不引入额外 minikube e2e：Platform 自身不直读 K8s API（PVC 管理是仅有例外，已在 envtest 覆盖）。

---

## 9. 计算任务

承接 [PRD §6.2.2 自定义任务](../../product/prd.md#622-自定义任务)。

「计算任务」（Compute Job，PRD 内称「自定义任务」）即一次性的训练 / 微调 / 数据处理 workload，支持单机与分布式（多机多卡）。Platform 全部能力都通过调用 [compute](compute.md) 的 Job 端点实现，自身不持有 Job 相关数据。

| 模块 | Platform 职责 | 调用下游 |
| --- | --- | --- |
| 任务视图与生命周期 | 创建 / 列表 / 详情 / 删除；按用户身份过滤可见任务 | `compute.CreateJob` / `GetJob` / `ListJobs` / `DeleteJob` |
| 取消 | 用户视角的 cancel 入口 | `compute.CancelJob` |
| 副本 / 事件 / 日志 | 鉴权 + 字段透传 | `compute.GetJobReplicas` / `GetJobEvents` / `GetJobLogs` |
| 产出回写制品 | 串联 Artifacts | `artifacts.InitiateUpload` / `CompleteUpload`（阶段二） |

**关键不变式：** Platform 自有 PG **不为计算任务建任何表**。任务的标识在 Platform 视图层是 `(tenant_name, job_name)`，URL 取 `/api/v1/tenants/{tenant}/jobs/{name}`；写之前调一次 `clustermanager.GetTenant(tenant_name)` 拿 `compute_namespace = Tenant.spec.namespace.name` 与可用 quota 清单。Job spec 在 [compute](compute.md) 侧不可变——UI 不提供「编辑任务」入口；改参数 = 新建任务。

### 9.1 角色与可见性矩阵

`@self` 表示「在该 job 所属租户上具备相应角色」；`@owner` 表示「`compute.jobs.owner_user == current_user.username`」。

| 能力 | `system-admin` | `tenant-admin@self` | `user@self & @owner` | `user@self & 非 owner` |
| --- | :---: | :---: | :---: | :---: |
| 列出可见任务 | 全集群 | 该租户全部 | 仅自己提交的 | 仅自己提交的 |
| 查看任务详情 | ✅ | ✅ | ✅ | ❌ |
| 提交任务 | ✅ | ✅ | ✅ | — |
| 取消 / 删除 | ✅ | ✅ | ✅ | ❌ |
| 查看副本 / 事件 / 日志 | ✅ | ✅ | ✅ | ❌ |
| 把产出注册为模型版本 | ✅ | ✅ | ✅ | ❌ |

### 9.2 数据模型（Platform 自有部分）

**Platform 不为计算任务建立任何表**。

#### 9.2.1 标识与寻址

- `tenant_name` 直接来自 Tenant CR `metadata.name`；
- `job_name` 由用户在创建表单中显式指定（DNS-1123，同一 `compute_namespace` 下唯一）；
- 下游 join key 用 `(compute_namespace, job_name)`，其中 `compute_namespace = Tenant.spec.namespace.name`，每次写请求前由 Platform 通过 `clustermanager.GetTenant(tenant_name)` 解析；
- 不为 Job 派生 Platform 端 uuid——Job 还在不在、它在哪个 namespace、它属于谁，三件事全部由 Compute 实时回答。

#### 9.2.2 列表查询路径

1. 按 RBAC 取可见租户集合 `tenant_names`；
2. 并行 `clustermanager.GetTenant(name)` 解析每个租户的 `compute_namespace`（request-scoped memoize）；
3. 并行 `compute.ListJobs(compute_namespace, {ownerUser?, status?, q?, limit, continue?})`；
4. 内存合并；对普通用户再按 `jobs.owner_user == current_user.username` 二次过滤；
5. 返回 `Job` DTO 列表（注入展示字段：tenant 名 / pool / unit 展示名）。

绝大多数日常查看都聚焦在「某一个租户的我的任务」上，单租户一次 RPC 即可。跨租户合并仅 `system-admin` 触发。

#### 9.2.3 对下游的依赖

| 调用 | 用途 |
| --- | --- |
| `clustermanager.GetTenant(name)` | 解析 `Tenant.spec.namespace.name` + `Tenant.spec.quotas[]`；校验 `status.phase == Active` |
| `compute.{Create,Get,List,Cancel,Delete}Job` | Job CRUD + 取消（[compute.md §6](compute.md#6-job) 现有端点） |
| `compute.GetJob{Replicas,Events,Logs}` | 详情页 Tab 2–4 透传 |
| `compute.GetResourcePool(id)` | 把表单提交的 `resourcePoolId` (uuid) 翻译为 pool name，拼接 ElasticQuota CR 名 `axisml-<tenant>-<pool>-<quota>` 与校验该 quota 在 `Tenant.spec.quotas[poolName]` 内存在 |
| `artifacts.InitiateUpload` / `CompleteUpload` | [§9.5.4](#954-产出注册为模型阶段二) 桥端点（阶段二） |

### 9.3 列表 / 创建表单

#### 9.3.1 列表页

| 列 | 来源 | 说明 |
| --- | --- | --- |
| 任务名 | `jobs.display_name`（fallback `jobs.name`） | 主展示列；点击进详情 |
| 镜像 | `jobs.spec.roles[0].template.image` | 多 role 时取首个 role 镜像 + 角标 `+N` |
| 后端 | `jobs.spec.backend.{name, engine}` | 列过滤 |
| 资源池 · 单元 | `jobs.pool_id` + `jobs.resource_unit_id` | 联表 ResourcePool / ResourceUnit 取展示名 |
| 状态 | `jobs.status` 直接展示 | 状态集合见 [compute.md §6.2](compute.md#62-数据模型) |
| Owner | `jobs.owner_user` | 仅 admin 可见列 |
| 开始 / 结束时间 | `jobs.started_at` / `jobs.finished_at` | finished 列空表示未完成 |
| 操作 | — | 取消 / 详情 / 删除 / 注册为模型 |

过滤：状态、关键字（任务名 / 镜像）、Owner（admin only）、backend、租户（admin 跨租户视图）。可见性：
- `system-admin`：默认仅展示自己「最近活跃」的租户视图，提供「切换租户」下拉跨租户浏览；
- `tenant-admin@self`：可见租户对应 namespace；
- `user`：上一条基础上再按 owner 二次过滤。

排序：默认 `created_at desc`；支持按 `started_at` / `finished_at` 排序。

操作按钮：
- **取消**：调 `POST .../cancel`；二次确认提示「已运行的 Pod 会被驱逐，已写出的产出文件保留」。`Creating` 状态由下游拒绝，UI 上禁用按钮。
- **详情**：进入详情页。
- **删除**：owner / `tenant-admin@self` / `system-admin`；二次确认。
- **注册为模型**（`Succeeded` 状态可点）：弹小表单填模型 `name` / `version` / `displayName` / 产出路径 → 调 `POST .../register-model`（阶段二）。
- **重新提交**：纯前端语法糖，把当前任务的 `spec` 反填到创建表单；不调端点，新任务有新 `job_name`。

#### 9.3.2 创建任务表单

| UI 字段 | 写入位置 | 说明 |
| --- | --- | --- |
| 任务名 | `jobs.name`（同时是 MLJob `metadata.name`）| 必填；DNS-1123；同一 `compute_namespace` 下唯一 |
| 显示名 | `jobs.display_name` | 可选 |
| 描述 | `jobs.description` | 可选 |
| 租户 | RBAC + 路径变量 `{tenant}` | 默认当前活跃租户；`system-admin` 可选任意租户 |
| 后端 / 引擎 | `jobs.spec.backend.{name, engine}` | 下拉；可选项见 [§9.3.3](#933-按后端渲染-roles)；默认 `(native, job)` |
| 资源池 / 资源单元 | `jobs.spec.scheduling.*` + `jobs.pool_id` / `resource_unit_id` | unit 必须属于 pool |
| 配额 | 拼接 `axisml-<tenant>-<pool>-<quota>` 写入 `spec.scheduling.quota` | 与 [§3.1](#31-用户提交训练任务) 一致 |
| Roles | `jobs.spec.roles[]` | 按 backend 渲染不同 role 集合 |
| `runPolicy.activeDeadlineSeconds` | `spec.runPolicy.activeDeadlineSeconds` | 可选硬超时（秒）|
| `runPolicy.ttlSecondsAfterFinished` | `spec.runPolicy.ttlSecondsAfterFinished` | 可选；终态后底层资源 GC 延迟 |
| `runPolicy.backoffLimit` | `spec.runPolicy.backoffLimit` | 可选重试次数 |

校验：DNS-1123 + 长度；镜像必须能被 `artifacts.Resolve`；ResourceUnit 必须属于所选 ResourcePool；`quota` 必须出现在 `Tenant.spec.quotas[resourcePoolId]`。

#### 9.3.3 按后端渲染 Roles

每个 backend / engine 对应一组固定 role（由 [compute-operator.md §4.5](compute-operator.md#45-内置-handler) 约定）：

| backend / engine | role 集合 | 阶段 |
| --- | --- | --- |
| `(native, job)` | `worker`（replicas=1，可改） | 阶段一（MVP）|
| `(native, podgroup)` | `worker`（replicas≥1，gang） | 阶段一 |
| `(kubeflow-trainer, pytorchjob)` | `master` + `worker`（可选 `elasticAgent`） | 阶段二 |
| `(kubeflow-trainer, tfjob)` | `chief` / `worker` / `ps` / `evaluator`（任一可空） | 阶段二 |
| `(kubeflow-trainer, mpijob)` | `launcher` + `worker` | 阶段二 |
| `(custom, *)` | 由 `backend.config` 决定 | 阶段三 / TBD |

每个 role 块包含：副本数 / 镜像 / 启动命令 / 参数 / 环境变量 / 重启策略。所有 role 共享同一 ResourceUnit（per-role ResourceUnit 作为后续迭代）。

#### 9.3.4 不在创建表单中的字段

- `spec.backend.config`：阶段一不开放；阶段二 `(kubeflow-trainer, *)` 上线时按需开放。
- `spec.scheduling.priorityClass`：阶段一统一走 ResourcePool 默认。
- `nodeSelector` / `tolerations`：由 ResourcePool + ResourceUnit 自动合并。
- `volumes` / `volumeMounts`：MVP 不开放；若需持久化产出，由系统管理员维护「带预挂载产出 PVC 的镜像」，或在 [§13](#13-后续迭代) 推动 DataVolume 集成。

### 9.4 详情页 Tab

详情页以 `(tenant_name, job_name)` 为维度，分为五个 Tab：概览、副本、事件、日志、审计。

#### Tab 1 概览

展示（全部来自 `compute.GetJob` 响应，全只读）：基本信息 + 当前状态卡片 + 时间线 + 调度参数（折叠）+ runPolicy + Roles 概览（`activeReplicas` / `readyReplicas` / `succeededReplicas` / `failedReplicas`）。

操作：取消 / 删除 / 注册为模型 / 重新提交 / 复制 YAML。

#### Tab 2 副本

调 `compute.GetJobReplicas` 透传：每个 Pod 一行（副本编号 / Pod 名 / phase / startedAt / 节点）。渲染等价 `kubectl exec` / `kubectl logs` 命令模板供用户复制；MVP 不提供 in-browser shell。

#### Tab 3 事件

调 `compute.GetJobEvents` 透传：聚合的 K8s Event，按 `lastTimestamp` 倒序。

#### Tab 4 日志

调 `compute.GetJobLogs` 透传：
- 默认按 `replica=0` 拉最近 1000 行；
- 支持切换 `replica` / `pod` / `container` / `tailLines` / `follow`；
- `follow=true` 走 SSE 流式渲染；
- Pod 已 GC 时下游返 410，UI 展示「日志已过期」+「重新提交」按钮。

#### Tab 5 审计

入口保留，MVP 不交付，登记到 [§13](#13-后续迭代)。

### 9.5 REST API 与模块结构

端点详见 [apis/platform.yaml](../apis/platform.yaml) `Jobs` tag。出站调用 `internal/client/compute` typed client。

#### 9.5.1 业务规则

- **POST 提交**：先 `clustermanager.GetTenant(tenant)` 解析 `compute_namespace` 与可用 quota；拼接 `spec.scheduling.quota = axisml-<tenant>-<pool>-<quota>`；再调 `compute.CreateJob(compute_namespace, body)`。失败语义：`400` 提交校验失败；`404` tenant 不存在 / `Tenant.status.phase != Active`；`409` 同 namespace 已有未软删的同名 job；`5xx` 透传下游 problem。
- **LIST 列表过滤**：`status` / `limit` / `continue` 下推到 Compute；`q`（关键字 name / displayName / image）/ `ownerUser`（admin only）/ `backendName` / `backendEngine` 在 Platform 内存做二次筛选（对分页结果不精确，MVP 阶段建议配合较大 `limit`）。响应可能附 `partial: true` 标志。
- **`GET /api/v1/jobs`（system-admin 全集群浏览）**：不填 `tenantName` = Platform 并行调所有 tenant 的 `compute.ListJobs` 后合并；填了 = 等价于 tenant-scoped 端点。
- **CANCEL**：无请求体。`compute.CancelJob` 透传——下游 message 由 Compute 写死 `'user cancelled'`，Platform 不参与拼接；下游对状态合法性的拒绝以 4xx problem 形式透传。自由文本 cancel reason 需要 Compute `/cancel` 接受请求体，目前不支持。
- **LOGS 流式**：`follow=false`（默认）透传 `text/plain` chunked 流；`follow=true` 透传 `text/event-stream` SSE 流；Platform 仅做 RBAC + header 转写后桥接，不缓冲不解析正文。
- **register-model**（阶段二）：本端点是 Job 与制品中心的桥端点；流程：RBAC + `compute.GetJob` 校验 `status == Succeeded` → `artifacts.InitiateUpload` 拿凭证 → 返回客户端 / 工作区脚本直推 zot → 客户端调 `POST .../register-model/complete` 转发到 `artifacts.CompleteUpload`。**前提依赖**：MVP 创建表单不开放 `volumes` / `volumeMounts`，故实际可用要等带预挂载产出 PVC 的镜像方案、`volumes` UI 或 DataVolume 集成上线（[§13](#13-后续迭代)）；MVP 阶段按钮灰显。

#### 9.5.2 上下文解析

**Tenant 解析**——每个写请求与详情类读请求的第一步：

```go
ctx := resolveTenantContext(c, tenantName) // 内部：
   //   1. clustermanager.GetTenant(tenantName) → spec.namespace.name + spec.quotas[]
   //   2. 校验 spec.status.phase == Active
   //   返回 { tenantId, tenantDisplayName, computeNamespace, quotas: map[poolName][]quotaName }
```

**ElasticQuota 名拼接**——仅在创建任务时需要：

```go
pool := compute.GetResourcePool(req.ResourcePoolId)        // 拿 pool.name
// 校验 req.Quota 在 ctx.quotas[pool.name] 中存在
elasticQuotaName := "axisml-" + ctx.TenantName + "-" + pool.Name + "-" + req.Quota
// 写入 spec.scheduling.quota
```

拼接逻辑唯一来源；后续可抽到 `internal/tenant/quotaname.go` 与 workspace / service 共享。

#### 9.5.3 失败语义

Platform 端不持久化任何 Job 字段，自然无双写一致性问题：

- 创建 / 取消 / 删除 / register-model：单点透传下游错误，不引入 Platform 补偿队列；
- 列表：并行调用中部分租户失败时，该租户在结果集中标 `partial=true` + `error.detail`，其余照常返回；前端在列表头部黄条提示「部分租户数据不可用」。

#### 9.5.4 产出注册为模型（阶段二）

POST `/api/v1/tenants/{tenant}/jobs/{name}/register-model` 是 Job 与制品中心之间的桥接：

1. RBAC 校验 + `compute.GetJob` 校验 `status == Succeeded`；
2. `artifacts.InitiateUpload` 拿 zot 凭证 + URI；
3. 返回客户端 / 工作区脚本，由客户端直推 zot；
4. 客户端调 `POST .../register-model/complete` 转 `artifacts.CompleteUpload`。

详见 [§9.5.1](#951-业务规则) 「register-model」条目的前提依赖。

#### 9.5.5 模块结构

目录：`components/platform/backend/internal/job/`

| 文件 | 职责 |
| --- | --- |
| `handler.go` | Gin 路由注册（`/api/v1/tenants/:tenant/jobs` 与 `/api/v1/jobs` 前缀）、RBAC gate 装配 |
| `service.go` | 业务编排：tenant 解析 → quota 名拼接 → 调 Compute；列表跨租户并行合并 |
| `context.go` | request-scoped 解析器：tenant → `compute_namespace` / `quotas` |
| `dto.go` | 请求 / 响应类型；与 Compute API DTO 的显式映射 |
| `view.go` | `Job` DTO 合并器：`compute.GetJob` 响应 + Platform 注入展示字段 |
| `validate.go` | 提交前校验 |
| `logstream.go` | 日志透传 pipe：`follow=false` 透传 chunked、`follow=true` 透传 SSE |
| `register.go` | 阶段二：注册为模型的桥接 |

无 `repository.go`。

RBAC 中间件：

| 路由 | 中间件链 |
| --- | --- |
| `GET /api/v1/jobs` | `RequireSystemAdmin` |
| `GET /api/v1/tenants/:tenant/jobs` | `RequireTenantRole("user", ":tenant")`；handler 内部按 `@owner` 二次裁剪 |
| `POST /api/v1/tenants/:tenant/jobs` | `RequireTenantRole("user", ":tenant")` |
| `GET /api/v1/tenants/:tenant/jobs/:name` | `RequireJobOwner(":tenant", ":name")`；语义：`@owner` 或在 tenant 上具备 `tenant-admin` 以上角色；`system-admin` 短路 |
| `POST .../cancel`、`DELETE`、`POST .../register-model` | 同上 |
| `GET .../replicas`、`GET .../events`、`GET .../logs` | 同上 |

`RequireJobOwner` 需要先 `compute.GetJob(ns, name)` 拿 `owner_user`；该 GET 结果通过 `gin.Context.Set("jobView", view)` 注入后续 handler，避免重复调用。

度量与日志：Prometheus 指标详见 [monitoring.md §5.4](../monitoring.md#54-job-模块)。zap 必带 `tenant_name` / `job_name` / `actor_user` / `action` / `status`；创建额外带 `backend_name` / `backend_engine` / `pool_id` / `resource_unit_id`；取消 / 删除额外带下游返回的 `compute_message`。

**审计日志**：Tab 5 审计是阶段二能力，但底层写入责任在 Platform job handler——`create` / `cancel` / `delete` / `register-model` 成功后，由 handler 向 `audit_logs` 表写一行：`action=job.<动作>`、`target=job:{tenant}/{name}`、`metadata` jsonb 含关键字段。Tab 5 渲染时按 `target` 前缀检索。

### 9.6 测试策略

- **单元**（`internal/job/*_test.go`）：`validate.go` 全分支；`view.go` DTO 合并器；列表合并器（部分租户失败时 `partial=true` 标记正确）；RBAC 中间件分支；`context.go` 解析器（`Tenant.status.phase != Active` 时返 400）。
- **integration**：testcontainers PostgreSQL + in-process gin + httptest fake Compute + httptest fake cluster-manager；happy path（创建 `(native, job)` 单 worker → status 推进 → 详情 → 删除）；cancel path；列表多租户合并（构造 3 个租户、每租户 5 任务，断言并行 RPC 数 == 3）；列表部分失败（1 个租户的 Compute 返 5xx，断言响应带 `partial=true`）；RBAC 矩阵；日志 SSE pipe（fake Compute 返回 chunked 流，Platform 透传后客户端按行收到）。
- 不引入额外 minikube e2e。

---

## 10. 在线服务

承接 [PRD §6.2.3 在线服务](../../product/prd.md#623-在线服务)。

「在线服务」（Online Service，本文简称 Service）即把一个已注册的模型版本以长驻 workload 形态部署出去，对外暴露 HTTP / gRPC 接口，承载推理流量。Platform 全部能力都通过调用 [compute](compute.md) 的 Service 端点实现，自身不持有 Service 相关数据。

| 模块 | Platform 职责 | 调用下游 |
| --- | --- | --- |
| 在线服务视图与生命周期 | 创建 / 列表 / 详情 / 删除；按用户身份过滤可见服务 | `compute.CreateService` / `GetServiceByID` / `ListServices(kind=service)` / `DeleteService` |
| 扩缩容 | 用户视角的 scale / start / stop 入口 | `compute.ScaleService` |
| 路由与访问 | 端点展示、`spec.route` 配置翻译为底层派生 HTTPRoute；可选 JWT 颁发 | `compute.GetServiceByID` + Platform JWKS |
| 流量与延迟指标 | 调 Prometheus 聚合 `request_rate` / `latency` / `error_rate` | `prometheus.Query` |
| 副本 / 事件 / 日志 | 鉴权 + 字段透传（同 workspace 阶段二解锁） | `compute.GetServiceReplicas` / `GetServiceEvents` / `GetServiceLogs` |

**关键不变式：** Platform 自有 PG **不为在线服务建任何表**。「这是普通在线服务」由 Compute `services.kind='service'` 列直接表达，与同一张表中的 `kind='workspace'` 行天然隔离；Platform 列表 = `compute.ListServices(namespace=<tenant_ns>, kind=service)`。寻址用 Compute `services.id`（uuid）；URL `/api/v1/services/{id}`。MLService spec 在 [compute](compute.md) 侧除 `roles[*].replicas` 外不可变——「切换模型版本」= 新建一个 service + 灰度切流量 + 下线旧 service。

### 10.1 角色与可见性矩阵

`@self` 表示「在该 service 所属租户上具备相应角色」；`@owner` 表示「`services.owner_user == current_user.username`」。

| 能力 | `system-admin` | `tenant-admin@self` | `user@self & @owner` | `user@self & 非 owner` |
| --- | :---: | :---: | :---: | :---: |
| 列出可见服务 | 全集群 | 该租户全部 | 仅自己创建的 | 仅自己创建的 |
| 查看服务详情 | ✅ | ✅ | ✅ | ❌ |
| 创建服务 | ✅ | ✅ | ✅ | — |
| 扩缩容 / start / stop | ✅ | ✅ | ✅ | ❌ |
| 修改展示元数据 | ✅ | ✅ | ✅ | ❌ |
| 删除 | ✅ | ✅ | ✅ | ❌ |
| 克隆为新版本 | ✅ | ✅ | ✅ | ❌ |
| 查看副本 / 事件 / 日志 | ✅ | ✅ | ✅ | ❌ |
| 查看流量与延迟指标 | ✅ | ✅ | ✅ | ❌ |

### 10.2 数据模型（Platform 自有部分）

**Platform 不为在线服务建任何 PG 表**。

#### 10.2.1 在线服务元数据的归属

| 字段 | 写入位置 | 备注 |
| --- | --- | --- |
| 服务身份 | Compute `services.id`（uuid） | URL 路径中的 `{id}` |
| 「这是普通在线服务」 | Compute `services.kind = 'service'` | Platform 创建时显式传 |
| 租户归属 | Compute `services.namespace` | 即 Tenant CR `spec.namespace.name` |
| MLService CR 名 | Compute `services.name` | 用户在创建表单填「服务名」直接落到此列（DNS-1123，同 namespace 内唯一） |
| 显示名 / 描述 / Owner | Compute `services.{display_name, description, owner_user}` | `owner_user` 来自 `X-Axisml-User` 头注入 |
| Backend / Engine | Compute `services.spec.backend.{name, engine, config}` | MVP 默认 `(native, deployment)` |
| Pool / Unit | Compute `services.{pool_id, resource_unit_id}` | |
| 模型版本引用 | Compute `services.spec.modelRef.{name, version}` | Platform 创建前用 `artifacts.Resolve` 校验 |
| 镜像 / 端口 / 启动命令 / env | Compute `services.spec.roles[*].template.*` | |
| 配额（ElasticQuota CR 名） | Compute `services.spec.scheduling.quota` | Platform 写前拼接 `axisml-<tenant>-<pool>-<quota>` |
| 副本 / 期望副本 | Compute `services.replicas` + `spec.roles[0].replicas` | 单 role 约定下二者同步（`/scale` 写两者并重算 `desired_spec_hash`） |
| 就绪副本 / endpoint / 状态 | Compute `services.{ready_replicas, endpoint, status, message}` | Informer 回流 |
| 路由配置 | Compute `services.spec.route.{enabled, path, hostname, auth, rateLimit, timeout}` | Platform 在创建表单产出 |
| Access URL | 派生：`https://<gateway-host><services.spec.route.path>`（`route.enabled=true`）或 `services.endpoint`（关闭路由时回退内部 DNS） | |
| 流量 / 延迟 / 错误率 | 实时 Prometheus 查询；不入 PG | 详见 [§10.5.4](#1054-prometheus-查询模板) |
| 创建 / 更新时间 | Compute `services.{created_at, updated_at}` | |

#### 10.2.2 列表查询路径

1. 按 RBAC 取可见 `tenant_name` 集合；
2. 并行 `clustermanager.GetTenant(name)` 解析每个 `compute_namespace`（request-scoped memoize）；
3. 并行 `compute.ListServices(namespace=<compute_namespace>, kind=service, ownerUser?, status?, q?, limit, continue?)`；
4. 对普通用户由 Compute 端的 `ownerUser=<current_user.username>` 过滤直接下推；
5. 返回 `Service` DTO 列表（注入展示字段：tenant 名 / pool / unit 展示名 / accessUrl）。

#### 10.2.3 对下游的依赖

| 调用 | 用途 |
| --- | --- |
| `clustermanager.GetTenant(name)` | 解析 `Tenant.spec.namespace.name` + `Tenant.spec.quotas[]`；校验 `status.phase == Active` |
| `compute.{Create,Scale,Delete}Service` + `GetServiceByID` + `ListServices` | Service CRUD + scale（[compute.md §7](compute.md#7-service)） |
| `compute.GetServiceReplicas/Events/Logs` | 阶段二解锁 |
| `compute.GetResourcePool(id)` | 拼接 ElasticQuota CR 名 + 校验 quota |
| `artifacts.Resolve(namespace, "model", name, version)` | 校验 `modelRef.{name, version}` 可见且 Ready |
| `artifacts.Resolve(namespace, "image", ...)` | 校验镜像 |
| `prometheus.Query` / `QueryRange` | 详情页指标 Tab |

#### 10.2.4 对 core 层的约束

本设计对 core 层 **无新增硬依赖**——所有 MVP 能力（创建 / 扩缩容 / 删除 / 路由 / kind 过滤 / id-based 寻址）均落在 [compute.md §7](compute.md#7-service) 与 [compute-operator.md §5](compute-operator.md#5-mlservice-controller) 现有契约内。下列项与本设计相关但不属于硬依赖：

- **`spec.route.stripPathPrefix`（计划，[§13](#13-后续迭代)）**：当前 `(native, deployment)` handler 派生的 HTTPRoute 不做路径重写——容器内的服务需要自行识别 `/services/<tenant>/<name>/` 前缀。MVP 通过注入 `AXISML_SERVICE_BASE_URL` 环境变量约定让镜像 entrypoint 自处理；少数推理框架（vLLM / Triton / TF Serving / TorchServe）不支持运行时 base path，则需要用户显式在 `command` / `args` 中传递路径参数。
- **Compute service `/events` / `/logs` / `/replicas` 端点扩展**：与 [§8.5.2](#852-业务规则) 同源；解锁详情页 Tab 3 / Tab 4。
- **多 role 独立扩缩**：解锁 KServe `(kserve, llminference)` 等 PD 分离 backend 的精细扩缩。

### 10.3 列表 / 创建表单

#### 10.3.1 列表页

| 列 | 来源 | 说明 |
| --- | --- | --- |
| 服务名 | `services.display_name`（fallback `services.name`） | 主展示列；点击进详情 |
| 模型 | `services.spec.modelRef.{name, version}` | 渲染为 `name@version`；列过滤 |
| 镜像 | `services.spec.roles[0].template.image` | 多 role 时取首个 role 镜像 + 角标 `+N` |
| 后端 | `services.spec.backend.{name, engine}` | 列过滤；MVP 仅 `(native, deployment)` |
| 资源池 · 单元 | `services.pool_id` + `services.resource_unit_id` | 联表 ResourcePool / ResourceUnit 取展示名 |
| 副本 | `services.ready_replicas / services.replicas` | 形如 `3/3`；`replicas=0` 显示「已停服」 |
| 状态 | `services.status` 直接展示 | 状态集合见 [compute.md §7.3](compute.md#73-状态机) |
| Owner | `services.owner_user` | 仅 admin 可见列 |
| 入口 | 派生 access URL | 仅展示 host + path 截断 + 复制按钮 |
| 创建时间 | `services.created_at` | |
| 操作 | — | 扩缩容 / 停 · 启 / 详情 / 克隆为新版本 / 删除 |

过滤：状态、关键字（服务名 / 镜像 / 模型名）、Owner（admin only）、backend、租户（admin 跨租户视图）。可见性：
- `system-admin`：默认仅展示自己「最近活跃」的租户视图，提供「切换租户」下拉跨租户浏览；
- `tenant-admin@self`：可见租户对应的 compute namespace 集合；
- `user`：上一条基础上由 Compute 端 `ownerUser=<current_user.username>` 过滤下推。

排序：默认 `created_at desc`；支持按 `ready_replicas` / `replicas` 排序。

操作按钮：
- **扩缩容**：弹小输入框填 `replicas`（≥0 整数）；调 `POST .../scale`。replicas 0 表示停服。
- **停服**（`replicas > 0` 时可点）：语法糖 = scale 0；二次确认提示「将立即驱逐所有副本，路由与配置保留，可随时扩回 ≥1」。
- **启动**（`replicas == 0` 时可点）：语法糖 = scale 1（或表单填写）。
- **详情**：进入详情页。
- **克隆为新版本**：纯前端语法糖——把当前 `services.spec` 反填到创建表单，默认把 `modelRef.version` 字段聚焦让用户填新版本，提交后即新建一只 service；不调端点。详见 [§10.3.4](#1034-版本切换灰度发布)。
- **删除**：owner / `tenant-admin@self` / `system-admin`；二次确认提示「将立即销毁所有副本、Service、HTTPRoute、SecurityPolicy、BackendTrafficPolicy；已建立的连接会断开」。

#### 10.3.2 创建在线服务表单

| UI 字段 | 写入位置 | 说明 |
| --- | --- | --- |
| 服务名 | `services.name`（同时是 MLService `metadata.name`） | 必填；DNS-1123 ≤40 字符；同一 `compute_namespace` 下唯一（含工作区） |
| 显示名 | `services.display_name` | 可选 |
| 描述 | `services.description` | 可选 |
| 租户 | RBAC + 路径变量；解析为 `compute_namespace` | 默认当前活跃租户；`system-admin` 可选任意租户 |
| 模型 | `services.spec.modelRef.{name, version}` | 必填；从 Artifacts `kind=model` 中选 → 版本下拉选 Ready 版本；后端用 `artifacts.Resolve` 校验 |
| 后端 / 引擎 | `services.spec.backend.{name, engine}` | 下拉；MVP 默认且仅 `(native, deployment)` |
| 镜像 | MLService `spec.roles[0].template.image` | 必填；从 Artifacts `kind=image` 中选 + 可手填 OCI URI；后端用 `artifacts.Resolve` 校验 |
| 容器端口 | `spec.roles[0].template.ports[0].containerPort` | 必填整数 ∈ [1, 65535] |
| 启动命令 / 参数 | `spec.roles[0].template.{command, args}` | 可选；不填则使用镜像 entrypoint |
| 环境变量 | `spec.roles[0].template.env` | 可选；Platform 在末尾追加 `AXISML_SERVICE_BASE_URL`（仅 `route.enabled=true` 时） |
| 资源池 / 资源单元 | `spec.scheduling.*` + `services.{pool_id, resource_unit_id}` | unit 必须属于 pool |
| 配额 | 拼接 `axisml-<tenant>-<pool>-<quota>` 写入 `spec.scheduling.quota` | |
| 副本数 | `spec.roles[0].replicas` + `services.replicas`（单 role 约定） | 必填整数 ≥0；0 = 仅创建不拉起 |
| `progressDeadlineSeconds` | `spec.runPolicy.progressDeadlineSeconds` | 可选；rollout 超时 |
| 路由 · 启用 | `spec.route.enabled` | 默认 `true`；关闭时仅内部 ClusterIP 暴露 |
| 路由 · 路径 | `spec.route.path` | Platform 自动拼 `/services/<tenant>/<service.name>/`；用户可在「高级」中覆盖 |
| 路由 · 主机名 | `spec.route.hostname` | 可选；默认走 Envoy Gateway 主域名 |
| 路由 · 鉴权 | `spec.route.auth.{type, jwt, apiKey}` | 默认 `jwt`，Platform 自带；切到 `apiKey` 时引用同租户 namespace 内的 K8s Secret；切到 `none` 时弹危险提示 |
| 路由 · 限流 | `spec.route.rateLimit.{requestsPerSecond, burst}` | 可选 |
| 路由 · 超时 | `spec.route.timeout` | 可选；ISO 8601 duration（如 `30s`） |

校验：DNS-1123 + 长度；服务名与同 namespace 内已有 service / workspace 不重名；模型 / 镜像必须能被 `artifacts.Resolve` 且对当前用户可见 + 模型版本 `status=Ready`；ResourceUnit 必须属于所选 ResourcePool；`quota` 必须出现在 `Tenant.spec.quotas[resourcePoolId]`；`containerPort` 范围；路径前缀必须以 `/` 开头、不以 `/` 结尾外加 `/<service.name>/` 收尾；`route.auth.type=apiKey` 时 `secretRef.name` 引用的 Secret 必须存在于 `compute_namespace` 内。

#### 10.3.3 按后端渲染 Roles

每个 backend / engine 对应一组固定 role：

| backend / engine | role 集合 | 阶段 |
| --- | --- | --- |
| `(native, deployment)` | `predictor`（replicas≥0） | 阶段一（MVP） |
| `(native, statefulset)` | `predictor`（replicas≥0，副本身份稳定） | 阶段二 |
| `(kserve, inference)` | `predictor`（replicas≥0；`backend.config.runtime` 决定 runtime） | 阶段二 |
| `(kserve, llminference)` | `prefill` + `decode` + `router`（PD 分离） | 阶段三 / TBD |
| `(custom, *)` | 由 `backend.config` 决定 | 阶段三 / TBD |

每个 role 块包含：副本数 / 镜像 / 容器端口 / 启动命令 / 参数 / 环境变量 / restartPolicy。MVP 单 role 单端口，多 role / 多端口在 [§13](#13-后续迭代)。

#### 10.3.4 版本切换 / 灰度发布

MLService spec.modelRef 创建后不可变（[compute-operator.md §5.2.3](compute-operator.md#523-字段归属与不可变性)）。「切换模型版本」在 Platform 上是：

1. 在旧 service 列表 / 详情页点「克隆为新版本」→ 前端把 `services.spec` 反填到创建表单，把 `modelRef.version` 聚焦让用户填新版本；服务名按 `<旧 name>-<新 version 简写>` 模板默认填好（用户可改）；
2. 提交后即建立一只新的 service，独立 endpoint、独立 HTTPRoute（路径前缀里带新 service.name）；
3. 用户通过外部流量切换（DNS / 反向代理 / 网关上的 weighted route 或 host header switch）从旧 service 切到新 service；
4. 验证完成后到旧 service 详情页点「停服」（scale 0）观察一段时间，再点删除。

MVP **不**接管 weighted route / canary / 金丝雀的 UI。完整灰度策略作为 [§13](#13-后续迭代) 「流量切换与灰度」项推进。

#### 10.3.5 不在创建表单中的字段

- `spec.backend.config`：MVP 不开放；阶段二 `(kserve, *)` 上线时按需开放。
- `spec.scheduling.priorityClass`：MVP 统一走 ResourcePool 默认。
- `nodeSelector` / `tolerations`：由 ResourcePool + ResourceUnit 自动合并。
- `volumes` / `volumeMounts`：MVP 不开放——在线服务的模型工件通过 `spec.modelRef` 由 handler 解析注入 `AXISML_MODEL_URI` 环境变量。
- `spec.modelRef` 不可变：「切换模型版本」走 [§10.3.4](#1034-版本切换灰度发布) clone-with-new-version 路径。

### 10.4 详情页 Tab

详情页以 `id` 为维度，分为六个 Tab：概览、访问、事件、日志、指标、审计。

#### Tab 1 概览

展示：基本信息（显示名 / 描述 / Owner / Tenant；可编辑：仅 `display_name` / `description`）；模型 / 镜像 / 后端 / 资源池 · 单元（只读）；当前状态卡片（`status` + `ready_replicas / replicas` + Compute `services.message`）；副本概览（`roles[predictor]` 的 `replicas` / `readyReplicas`）；路由概览（`route.path` / `route.hostname` / `route.auth.type` / 限流 / 超时）；时间线（`createdAt` / `updatedAt` / `lastScaledAt`）。

操作：扩缩容 / 停服 / 启动；编辑展示元数据；克隆为新版本；删除；复制 YAML。

#### Tab 2 访问

展示：Access URL（`https://<gateway><services.spec.route.path>`；`route.enabled=false` 时退化为 `<services.endpoint>` 内部 DNS）；鉴权说明（按 `auth.type` 分支：`none` 红色警示 / `jwt` 展示 issuer + audience + 「拿一次性 JWT」按钮 / `apiKey` 展示 Secret 名 + 轮换入口）；调用示例（`curl` / `python` / `grpcurl` 三段）；port-forward 命令模板。

操作：拿 access JWT（`auth.type=jwt`）：调 `GET .../access` 拿 `{url, jwt, expiresAt}`。

#### Tab 3 事件

透传 Compute service events，MVP 阶段同 [§8.4 Tab 3](#tab-3-事件) 留空，等待 Compute 端 `/events` 端点扩展。

#### Tab 4 日志

同 [§8.4 Tab 4](#tab-4-日志) —— MVP 不交付。

#### Tab 5 指标

展示（来自 Prometheus 实时查询，详见 [§10.5.4](#1054-prometheus-查询模板)）：
- **流量**：每秒请求数 `request_rate`；
- **延迟**：p50 / p95 / p99；
- **错误率**：5xx 比例 + 4xx 比例（分离展示，4xx 通常是调用方问题）；
- **饱和**：副本 CPU / 内存 / GPU 利用率（kube-prometheus 标准指标）；
- **LLM 专项**（`backend=kserve, engine=llminference` 时，阶段三）：tokens/sec、TTFT、TBT、KV cache 占用——MVP 不交付，占位入口。

时间窗口选择器：5m / 15m / 1h / 6h / 24h；自动刷新 15s。调优 / 告警阈值在「系统管理 → 监控告警」（[§13](#13-后续迭代)）独立菜单维护。

#### Tab 6 审计

入口保留，MVP 不交付；登记到 [§13](#13-后续迭代)，按 `target=service:{service_id}` 索引展示。

### 10.5 REST API 与模块结构

端点详见 [apis/platform.yaml](../apis/platform.yaml) `Services` tag。出站调用 `internal/client/compute` typed client。

#### 10.5.1 业务规则

- **POST 提交**：`clustermanager.GetTenant(tenantId)` 解析 `compute_namespace` 与可用 quota → `artifacts.Resolve` 校验 `modelRef` / `image` → 拼接 `spec.scheduling.quota` → 若 `route.enabled=true && route.path == ""`，Platform 自动拼 `route.path = /services/<tenant>/<name>/` 并在 env 末尾追加 `AXISML_SERVICE_BASE_URL=<route.path>` → 调 `compute.CreateService(ns, body)`，body 中 `kind="service"`。失败语义：`400` 提交校验；`404` tenant 不存在 / phase 非 Active / artifact 不存在；`409` 同 namespace 已有未软删的同名 service 或 workspace；`5xx` 透传。
- **LIST 列表过滤**：`tenantName` / `status` / `ownerUser`（admin only） / `limit` / `continue` 下推到 `compute.ListServices(namespace, kind=service, ...)`；`q` / `backendName` / `backendEngine` / `modelName` 在 Platform 内存做二次筛选（对分页结果不精确）。响应可能附 `partial: true`。`tenantName` 不填且当前用户为 `system-admin` 时走全集群并行 LIST + 合并。
- **PATCH**：仅可改 `displayName` / `description`；其他字段一律不可变，变更走 [§10.3.4](#1034-版本切换灰度发布) clone-with-new-version 路径。`spec.route` 热更新依赖 compute-operator 的 [§5.7 后续工作](compute-operator.md#57-后续工作)。
- **DELETE**：先 `compute.GetServiceByID(id)`；404 → 直接返 200（幂等）；`kind != 'service'` → 返 `404`（避免误删工作区）。派生的 K8s Service / HTTPRoute / SecurityPolicy / BackendTrafficPolicy 由 ownerReference 级联清理。
- **scale / start / stop**：`/scale` body `{"replicas": <int≥0>}`；幂等保护：`replicas == 当前 replicas` 直接返 200；`Deleted` / `Deleting` 拒绝 → `409 service-deleted`。`/start` 是 scale 到「上一次 >0 的 replicas」（查 Platform 自身 `audit_logs` 表），audit 缺失则 fallback 到 `1`；`/stop` 是 scale 0。
- **access**：仅当 `route.enabled=true && route.auth.type=jwt` 时有意义；其他情况返 `409 route-auth-mismatch` problem。JWT claim：`iss=axisml-platform`，`aud=axisml-inference`，`sub=<current-user-id>`，`service_id=<id>`，`tenant=<tenant_name>`，`exp=now+1h`。网关侧 SecurityPolicy 校验 `aud=axisml-inference` 防止跨用途滥用。
- **metrics**：`metric` 枚举 `request_rate` / `latency` / `error_rate` / `cpu_util` / `mem_util` / `gpu_util`；`range` ISO 8601 duration；`percentile` 仅 `metric=latency` 时（默认 `p95`）。流程：`compute.GetServiceByID(id)` 拿 backend → 按 backend 选择 Prometheus 查询模板（[§10.5.4](#1054-prometheus-查询模板)） → 透传 Prometheus `Query` / `QueryRange` 响应。Prometheus 查询失败返 `502 upstream-failure`。
- **events / logs / replicas**（占位）：MVP 期间返回 `501 Not Implemented` + problem `type=https://axisml.io/errors/upstream-not-ready`。阶段二上线后语义与 [§9.5.1](#951-业务规则) 一致（含 SSE `follow=true`）。

#### 10.5.2 上下文解析

**Tenant 解析**——与 [§9.5.2](#952-上下文解析) 共享：

```go
ctx := resolveTenantContext(c, tenantName) // 内部：
   //   1. clustermanager.GetTenant(tenantName) → spec.namespace.name + spec.quotas[]
   //   2. 校验 spec.status.phase == Active
   //   返回 { tenantId, tenantDisplayName, computeNamespace, quotas: map[poolName][]quotaName }
```

**ElasticQuota 名拼接**——与 [§9.5.2](#952-上下文解析) 同源。

**路由路径拼接**——`route.go` 内：

```go
// 用户未填 path 时
if route.Enabled && route.Path == "" {
    route.Path = fmt.Sprintf("/services/%s/%s/", ctx.TenantName, req.Name)
}
// 校验：必须 `/` 开头 `/` 结尾，且包含 service.name 防止路径冲突
// env 注入
if route.Enabled {
    env = append(env, corev1.EnvVar{Name: "AXISML_SERVICE_BASE_URL", Value: route.Path})
}
```

#### 10.5.3 失败语义

Platform 端不持久化任何 service 字段，自然无双写一致性问题：
- 创建 / 扩缩容 / 删除：单点透传下游错误，不引入 Platform 补偿队列；
- 列表：并行调用中部分租户失败时，该租户在结果集中标 `partial=true` + `error.detail`。

防误删：所有 service 写操作（PATCH / scale / start / stop / DELETE / access / metrics）先 `compute.GetServiceByID(id)` 校验 `kind == 'service'`，否则返 `404`。

#### 10.5.4 Prometheus 查询模板

`metrics.go` 维护按 backend 选择的 PromQL 模板，详情页 Tab 5 仅展示成熟模板覆盖的指标。

| backend | metric | PromQL 模板（伪代码） |
| --- | --- | --- |
| `(native, deployment)` | `request_rate` | `sum(rate(envoy_http_downstream_rq_total{...service=<name>}[1m]))` |
| `(native, deployment)` | `latency` | `histogram_quantile(<percentile>, sum(rate(envoy_http_downstream_rq_time_bucket{...service=<name>}[5m])) by (le))` |
| `(native, deployment)` | `error_rate` | `sum(rate(envoy_http_downstream_rq_xx{response_code=~"5..", ...service=<name>}[5m])) / sum(rate(envoy_http_downstream_rq_total{...service=<name>}[5m]))` |
| `*` | `cpu_util` | `sum(rate(container_cpu_usage_seconds_total{namespace=<ns>, pod=~"<name>-.*"}[1m])) by (pod)` |
| `*` | `mem_util` | `sum(container_memory_working_set_bytes{namespace=<ns>, pod=~"<name>-.*"}) by (pod)` |
| `*` | `gpu_util` | `avg(DCGM_FI_DEV_GPU_UTIL{namespace=<ns>, pod=~"<name>-.*"}) by (pod)` |
| `(kserve, *)` | `request_rate` / `latency` / `error_rate` | 优先选用 KServe 自带的 `revision_request_count` / `revision_request_latencies_bucket`；模板在阶段二落地 |
| `(kserve, llminference)` | `tokens_per_second` / `ttft` / `tbt` | 占位，阶段三 |

模板中的 label selector 由 `metrics.go` 按 service `(namespace, name, backend, engine)` 实时拼装；Prometheus URL 来自 [§4.6 启动配置](#46-启动配置) 的 `--prometheus-url`。

#### 10.5.5 模块结构

目录：`components/platform/backend/internal/service/`

| 文件 | 职责 |
| --- | --- |
| `handler.go` | Gin 路由注册（`/api/v1/services` 前缀）、RBAC gate 装配 |
| `service.go` | 业务编排：tenant 解析 → quota 名拼接 → artifacts.Resolve 校验 → 调 Compute；列表跨租户并行合并；scale / start / stop / metrics 编排 |
| `context.go` | request-scoped 解析器：tenant → `compute_namespace` / `quotas`；与 [§9.5.2](#952-上下文解析) 共享底层 helper |
| `dto.go` | 请求 / 响应类型；与 Compute API DTO 的显式映射 |
| `view.go` | `Service` DTO 合并器：`compute.GetServiceByID` 响应 + Platform 注入展示字段 |
| `validate.go` | 提交前校验 |
| `route.go` | 路由路径拼接 + env 注入（`AXISML_SERVICE_BASE_URL`）+ 路由配置校验 |
| `jwt.go` | access JWT 颁发（`aud=axisml-inference`）；签名密钥与 `internal/auth` 共享 |
| `metrics.go` | Prometheus PromQL 模板组装 + 查询透传 |
| `logstream.go` | 阶段二：日志透传 pipe |

无 `repository.go`。

RBAC 中间件：

| 路由 | 中间件链 |
| --- | --- |
| `GET /api/v1/services` | 已登录即可；handler 内部按角色裁剪可见集合 |
| `POST /api/v1/services` | `RequireTenantRole("user", "<body.tenantId>")`，`system-admin` / `tenant-admin@self` 短路 |
| `GET /api/v1/services/{id}` | `RequireServiceOwner("id")`；语义：`@owner` 或在 service 所属租户上具备 `tenant-admin` 以上角色；`system-admin` 短路 |
| `PATCH /api/v1/services/{id}`、`DELETE`、`POST .../scale`、`POST .../start`、`POST .../stop`、`GET .../access`、`GET .../metrics` | 同上 |
| `GET .../replicas` / `GET .../events` / `GET .../logs` | 同上 |

`RequireServiceOwner` 需要先 `compute.GetServiceByID(id)` 拿 `owner_user` + `namespace`；该 GET 结果通过 `gin.Context.Set("serviceView", view)` 注入后续 handler，避免重复调用。

度量与日志：Prometheus 指标详见 [monitoring.md §5.5](../monitoring.md#55-service-模块)。zap 必带 `tenant_name` / `service_id` / `actor_user` / `action` / `status`；创建额外带 `backend_name` / `backend_engine` / `model_name` / `model_version` / `pool_id` / `resource_unit_id`；scale 额外带 `from_replicas` / `to_replicas`。

**审计日志**：Tab 6 审计是阶段二能力，底层写入责任在 Platform service handler——`create` / `scale` / `patch` / `delete` 成功后，由 handler 向 `audit_logs` 表写一行：`action=service.<动作>`、`target=service:{service_id}`、`metadata` jsonb 含关键字段。

### 10.6 测试策略

- **单元**（`internal/service/*_test.go`）：`validate.go` 全分支；`route.go` 路径拼接 + env 注入 + 校验；`view.go` DTO 合并器；列表合并器（部分租户失败时 `partial=true` 标记）；start/stop 反查 audit_logs 拿最近 replicas 的回退逻辑；防误删（DELETE 一个 `kind='workspace'` 的对象 → 404；scale 同理）；RBAC 中间件分支；`context.go` 解析器；`metrics.go` PromQL 模板组装；`jwt.go` access JWT claim 字段完整性（`aud=axisml-inference`）。
- **integration**：testcontainers PostgreSQL（仅 `users` / `user_tenant_roles` / `audit_logs`）+ in-process gin + httptest fake Compute + httptest fake cluster-manager + httptest fake Artifacts + httptest fake Prometheus；happy path（创建 `(native, deployment)` 含 route → MLService Ready → scale 3 → scale 0 → start 反查 audit 拿到 3 → 删除）；防误删（DELETE 一个 `kind='workspace'` 的对象 → 404）；clone-with-new-version；列表多租户合并（构造 3 个租户、每租户 5 个 service + 5 个 workspace，断言 `kind=service` 过滤后只返回 15 个 service）；列表部分失败；RBAC 矩阵；路由配置（`auth.type ∈ {none, jwt, apiKey}` 创建均能下发到 Compute）；指标查询；access JWT。
- 不引入额外 minikube e2e。

---

## 11. 部署架构

详见 [deployment.md](../deployment.md)。要点：

- Platform 随 `axisml-system` chart 一起发布，模板位于 `templates/platform/`；前端镜像通过 Helm 模板的 `platform.frontend.image` 字段独立部署，本期与后端共用一个 Service；
- ConfigMap 注入下游 URL：`ML_CLUSTER_MANAGER_URL` / `ML_COMPUTE_URL` / `ML_ARTIFACTS_URL` / `ML_PROMETHEUS_URL`；
- 外部流量必须经 [Envoy Gateway](../infra.md) 进入 Platform；Cluster Manager / Compute / Artifacts 始终保持 ClusterIP，不接受外部直连；
- PostgreSQL 沿用 `axisml-system` chart 的内置实例或 `externalDatabase`，Platform 与 Compute / Artifacts 共享同一个 PG 实例（按表名前缀隔离，详见 [database.md §1](../database.md#1-概述)）。

---

## 12. 关键设计决策

| 决策项 | 决策 | 理由 |
| --- | --- | --- |
| 文档组织 | 单文档承载所有功能，每个功能独立章节 + 共享一份 wireframe 文件 | 减少跨文件查找；横切设计（PG schema / typed client / 部署）只在一处维护 |
| 后端语言与框架 | Go + Gin + GORM + Cobra，与 Compute 一致 | 复用现有组件骨架与 CI 流水线，降低维护成本 |
| 认证方式 | 内置用户 + RBAC，`IdentityProvider` 接口预留 OIDC | 第一版自建集群可独立运行；外部 IdP 按需接入而非默认依赖 |
| Platform PG 范围 | 仅存身份（users / roles / permissions）、授权（user_tenant_roles）、会话（sessions）、审计（audit_logs）四类；不建任何视图缓存表 | 业务对象权威在 Compute / Artifacts / cluster-manager；杜绝双写漂移与补偿队列 |
| 租户实体 | Platform 不建 `tenants` 表；权威完全在 cluster-manager PG | 消除 Platform ↔ cluster-manager 双写漂移；获得 cluster-manager 一级 `description` / `business_unit` 字段、软删保留、`restore` 等能力 |
| 配额 UI 归属 | 租户管理详情页内 Tab，不独立菜单 | 配额始终在 (tenant, pool) 上下文中操作 |
| 资源池/单元同文档 | 同入口、同权限、强父子关系 | 独立文档反而割裂同一用户故事 |
| 工作区实现 | 复用 `MLService(native, deployment)` 承载工作区；Compute service 加 `kind` 列区分普通服务与工作区 | 避免新增 CRD；与在线服务共用 backend handler；不在 Platform 维护「工作区标记」视图表 |
| 工作区不抽象 IDE 模板 | 镜像、端口、启动命令直接由用户提交 | 镜像本身决定 IDE；二级建模会带来 fragile 的「模板 ↔ 镜像」绑定 |
| 工作区启停语义 | 显式 `POST /start` / `POST /stop` 两个 endpoint | 与 PRD「不用时停止释放资源」贴近；权限模型清晰；底层翻译为 Compute `/scale` 对用户透明 |
| 工作区持久化 | MVP 内置 PVC；Platform 直管而非塞进 MLService handler；PVC 名 deterministic 派生自 `services.id` 前 8 字符 | 隔离 workspace 存储语义；MLService 仅负责 `volumes/volumeMounts` 透传 |
| Job 寻址方式 | `(tenant_name, job_name)` 二元组，URL `/api/v1/tenants/{tenant}/jobs/{name}` | 没有 Platform 自有 id 可用；tenant-scoped 路径自带 RBAC 边界 |
| Job spec 不可变 | UI 不提供「编辑任务」入口；改参数 = 新建 | 与下游约束对齐 |
| Service 寻址 | Compute `services.id`（uuid）；URL `/api/v1/services/{id}` | id 由 Compute 生成；与工作区共享 id-based + kind 过滤端点 |
| Service 模型引用必填 | 创建表单要求填 `modelRef.{name, version}`；后端 schema 沿用 compute MLService optional 字段 | 对齐 PRD 「部署模型版本」语义；不在 Compute 引入额外约束 |
| Service spec 不可变性 | UI 不提供「修改模型 / 资源 / 路由」入口；改参数 = clone-with-new-version 新建 + 用户自主切流量 | 与下游约束对齐；版本切换语义清晰 |
| Service 路径前缀处理 | MVP 注入 `AXISML_SERVICE_BASE_URL` env 让镜像 entrypoint 自处理；compute-operator 的 `stripPathPrefix` 作为后续迭代 | 不引入 core 层硬依赖 |
| 防误删 | 所有 service / workspace 端点先校验 `kind`，不一致返 404 | 避免端点交叉污染 |
| Service 指标来源 | 实时调 Prometheus，不引入 Platform 侧时序缓存 | Prometheus 已是基础设施层一等公民；缓存只会带来漂移 |
| 列表跨租户策略 | 默认单租户视图；`system-admin` 跨租户走并行 list + 内存合并；部分失败标 `partial=true` 而非整体 5xx | 普通用户日常只看自己租户；部分容忍策略避免单点故障拖垮全列表 |
| ElasticQuota 名拼接 | Platform 写前用 `axisml-<tenant>-<pool>-<quota>` 实时拼接，并校验该 quota 在 Tenant CR 内存在 | 提前校验避免 Pod 调度时才发现 quota 不存在 |
| 资源池/单元删除 | Platform 前置阻断（资源池查 unit + 活跃 Job/Service；资源单元查活跃 Job/Service） | compute 当前未约束级联；前置阻断给 UI 更清晰反馈 |
| 资源池/单元可见性 | 全局可见，不按租户过滤 | 与 [compute.md §4.1](compute.md#41-概述) 对齐；按租户白名单作为后续迭代 |
| 节点 label/taint 维护 | 管理员手工 kubectl 运维，UI 不下发 | Platform 不修改 Node 对象 |
| 制品 UI 范围 | 制品中心首版只覆盖模型 | 与菜单一致；dataset / image / eval_report 复用 Artifacts API 但 UI 后续迭代 |
| Dashboard | 纯前端聚合视图，无独立后端详设 | 数据全部来自 Job / Service / Artifact / Quota 列表 API + Prometheus 查询 |
| 用户身份透传 | Platform → 下游统一注入 `X-Axisml-User` 头 | 下游服务保持只接受内部调用、信任 Platform 鉴权 |

---

## 13. 后续迭代

按当前菜单与设计文档树，以下能力作为后续迭代项保留入口：

### 13.1 横切

- **应用中心**：智能体（Agent）/ Skills / MCP 三个子菜单仅在前端预留路由；后端契约、数据模型与 IdP 集成在该方向需求稳定后再展开。
- **数据卷管理**：可能的实现方向包括 PVC 抽象、数据集挂载路由、集群 StorageClass 视图，本期不冻结字段。
- **OIDC 接入**：`auth` 模块给出接口签名后即可切换，但本期只交付 `internal`。
- **审计日志 UI**：`audit_logs` 表已规划，下一步是 UI 视图（按 `target` 前缀检索）与告警规则模板。
- **多集群 / 多区域**：当前所有概念按单集群假设；多集群方向作为远期演进。

### 13.2 制品中心

- **制品扩展**：dataset / image / eval_report 三类 Artifact 已被 Artifacts 服务支持，仅缺 UI；UI 完工后从 [§1.1](#11-菜单与功能矩阵) 矩阵升级为 ✅。

### 13.3 租户

- **配额硬校验 / 分层配额**：等上游 ElasticQuota 提供 `parent` 字段后，Platform 可在配额 Tab 引入「上限 cap」视图与硬阻断。
- **「已归档租户」管理界面**：UI 入口（系统管理菜单下二级），调 `GET /api/v1/tenants?includeArchived=true&deletedAt_not_null=true` 列出，每行操作 `POST .../restore`。底层能力已由 cluster-manager 提供。
- **init_resources 表单深度**：当前仅暴露 `sourceXxxRef`；后续可接入 Vault / Sealed Secrets 等加密源直接创建。
- **租户克隆**：基于已有租户的 `quotas` / `initResources` 模板快速创建新租户。

### 13.4 资源池 / 资源单元

- **按租户的池可见性**：Platform 引入「池 → 租户白名单」表；提交 Job / Service 时按当前租户裁剪可见池。
- **节点匹配预览 Tab**：Platform 引入 K8s typed client，按 `node_selector` + `tolerations` 反查命中 Node，配合「容量 / 已分配 / 剩余」三列。
- **池容量聚合**：聚合命中 Node 的 `allocatable` / `requested`，作为配额规划的辅助视图。
- **池间调度借用策略**：当配额不足时是否允许跨池借用；当前默认禁止。
- **资源单元成本元数据**：UI 暴露 `unit.metadata.cost_per_hour`，用于成本核算。
- **资源单元 in-process 缓存**：列表页 `resource_unit_count` 聚合改为 5–10 秒级 LRU。

### 13.5 工作区

- **Compute service `/events` / `/logs` / `/replicas` 端点扩展**：呼应 Job 端 [§6.4.5](compute.md#645-副本与事件端点)；解锁详情页 Tab 3 / Tab 4。
- **闲时自动 stop**：基于 `services.updated_at` + 「无活跃连接」指标，超时自动 `replicas=0`；释放 GPU / 高规格资源单元。
- **孤儿 PVC 清理 UI**：见 [§8.5.5](#855-pvc-管理)。
- **创建表单预设**：系统管理员维护一组「镜像 + 启动命令 + 资源单元」预设供普通用户一键填好；只是前端语法糖。
- **SSH 接入**：通过 Platform tcp-proxy 或独立 SSH gateway 暴露 `:22`。
- **多容器 Workspace**：同一 Pod 内 jupyter + tensorboard sidecar；需 MLService spec 支持多 init/sidecar container（非 role 维度）。
- **GPU 预热 / 镜像预拉**：常用镜像在节点上保活，启动时间从「分钟」级降到「秒」级。
- **DataVolume 集成**：把 `/workspace` 切到共享数据卷（跨 workspace / 跨用户共享）。

### 13.6 计算任务

- **`(kubeflow-trainer, *)` 多 role backend**：阶段二能力。
- **`(custom, *)` 接入**：UI 表达需要 JSON schema 编辑器；与 [compute-operator.md §4.6](compute-operator.md#46-后续工作) 一同推进。
- **per-role ResourceUnit**：解锁 PyTorchJob master CPU、worker GPU 场景；需下游 `jobs` 表 schema 演进。
- **register-model 完整链路**：详见 [§9.5.4](#954-产出注册为模型阶段二)。
- **任务模板 / 预设**：常用 `(backend, image, command, args, env, runPolicy)` 组合一键创建；纯前端语法糖。
- **重新提交 UX**：spec 反填创建表单。
- **DAG 工作流**：组合多个 Job 形成依赖链。
- **SSE / WebSocket 增量列表**：替代轮询。
- **列表过滤下推**：Compute 端 `?ownerUser=` / `?labelSelector=` 等参数，把内存过滤收敛到 PG 查询。

### 13.7 在线服务

- **`(native, statefulset)` backend**：解锁有状态推理（KV cache、模型分片、副本身份固定）；阶段二能力。
- **`(kserve, inference)` backend**：开放 vLLM / Triton / TF Serving / TorchServe / HuggingFace runtime；阶段二能力。
- **`(kserve, llminference)` backend**：PD 分离推理 + LLM 专项指标；阶段三 / TBD。
- **`(custom, *)` 接入**：UI 表达需要 JSON schema 编辑器。
- **`spec.route` 热更新**：解锁 PATCH 路由配置（auth 切换 / rateLimit 调整 / timeout 调整）；依赖 compute-operator 的 [§5.7](compute-operator.md#57-后续工作)。
- **`spec.route.stripPathPrefix`**：在 HTTPRoute 上做路径重写，解锁不支持运行时 base path 的推理框架。
- **流量切换与灰度**：同一路径前缀下 weighted 切分新旧 service、自动指标判定回滚。
- **自动扩缩容（HPA / KEDA）**：基于 `request_rate` / `gpu_util` / 自定义指标的弹性扩缩。
- **多 role 独立扩缩**：解锁 `(kserve, llminference)` 的 `prefill` / `decode` / `router` 各自扩缩。
- **多端口 / 多协议**：HTTP + gRPC 共存、健康检查独立端口。
- **API key 轮换 UI**：`auth.type=apiKey` 的 Secret 自助轮换。
- **LLM 专项指标**：tokens/sec、TTFT、TBT、KV cache 占用、batch utilization；阶段三随 `(kserve, llminference)` 落地。
- **告警与 SLO**：基于 Tab 5 指标的告警规则模板，与 kube-prometheus AlertManager 集成。

### 13.8 跨模块共享

- **cluster-manager 批量 GetTenant RPC**：减少 Platform 多次跨租户解析 namespace 的 RPC 数（与 [§8.5](#85-rest-api-与模块结构) / [§9.5](#95-rest-api-与模块结构) / [§10.5](#105-rest-api-与模块结构) 同源）。

---

## 14. 相关引用

- [PRD](../../product/prd.md)
- [docs/system_design/overview.md](../overview.md)
- [docs/system_design/database.md](../database.md)
- [docs/system_design/monitoring.md](../monitoring.md)
- [docs/system_design/deployment.md](../deployment.md)
- [docs/system_design/components/cluster-manager.md](cluster-manager.md)
- [docs/system_design/components/tenant-operator.md](tenant-operator.md)
- [docs/system_design/components/compute.md](compute.md)
- [docs/system_design/components/compute-operator.md](compute-operator.md)
- [docs/system_design/components/artifacts.md](artifacts.md)
- [docs/system_design/infra.md](../infra.md)
- [docs/system_design/apis/platform.yaml](../apis/platform.yaml)
- [wireframe.md](../wireframe.md)
