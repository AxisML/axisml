# AxisML Platform 概要设计

## 1. 概述

**AxisML Platform** 是 AxisML 系统中唯一直接面向用户的层。它由 **前端**（Web UI，TypeScript + React）与 **后端**（Go REST API 服务）两部分组成，部署在 Envoy Gateway 之后，对下串联 [Cluster Manager](../core/cluster-manager.md)、[Compute](../core/compute.md) 与 [Artifacts](../core/artifacts.md) 三个内部服务，对上为登录用户提供统一的业务视图与操作入口。

本文档是 Platform 子系统的概要设计：

- 描述 Platform 的边界、核心概念、菜单组织与整体架构；
- 定义 Platform 后端的代码骨架、PG schema、下游客户端结构、配置与部署形态；
- 给出关键设计决策与后续迭代方向。

字段级 schema、API 路径、状态机与具体实现契约以各功能详细设计文档为准（见 [§3 菜单与功能矩阵](#3-菜单与功能矩阵)）。系统级架构与术语以 [docs/system_design/overview.md](../overview.md) 为准；本文档中遇到的与系统层重叠的术语（Tenant / ResourcePool / ResourceUnit / Quota / Job / Service / Artifact）一律沿用上层定义，不在此处重复描述。

## 2. 核心概念

Platform 在系统层概念之上额外引入了一组用户视角的对象，把「裸 namespace 分区」的下层服务包装成租户化、用户化的体验。

### 2.1 用户（User）

Platform 内部维护的登录身份。第一版采用内置用户表（用户名 + bcrypt 密码 + JWT 颁发），所有外部 HTTP 流量必须先通过 Platform 鉴权后才能进入下游服务。OIDC / SAML 等外部 IdP 通过抽象出 `IdentityProvider` 接口预留接入点，本期不交付。详见 [auth.md](auth.md)。

### 2.2 角色（Role）与权限（Permission）

第一版内置四档角色：

| 角色 | 范围 | 能力概览 |
| --- | --- | --- |
| `system-admin` | 全局 | 用户 / 角色 / 租户 CRUD、资源池 / 资源单元 CRUD、读所有租户数据 |
| `tenant-admin` | 单租户 | 租户内成员管理、配额申请、对该租户全部业务对象（Job / Service / Artifact / Workspace）的读写 |
| `tenant-member` | 单租户 | 提交 / 管理自己创建的业务对象、读取本租户内的共享资产 |
| `viewer` | 单租户 | 仅只读视图，不允许修改任何对象 |

角色与权限的具体绑定矩阵在 [auth.md](auth.md) 中给出；本文 §7 仅声明数据模型形态。

### 2.3 租户视图（TenantView）

`TenantView` 是 Platform 暴露给用户的「租户」概念，与系统层的 `Tenant` CR 是 1:1 关系，但额外携带 Platform 自己维护的两条信息：

- 该租户对应的 **`(compute_namespace, artifacts_namespace)` 二元组**；
- 该租户在前端可见的展示信息（图标、描述、所属业务线等）。

Cluster Manager 不持有任何 PG，权威完全在 Tenant CR；TenantView 的 `tenant_name`、配额、状态等字段始终读 Tenant CR，Platform 自己只缓存「namespace 二元组」与「展示元数据」。下游 Compute / Artifacts 不感知 TenantView，它们仅按 `namespace` 分区。

> **与上层 overview §2.9 中「工作区 = (compute_namespace, artifacts_namespace) 二元组」的关系**：上层 overview 把这个二元组称为「工作区」（Workspace），那是 Platform 内部的命名空间映射对象；本文档为了避免与用户菜单里的「工作区 = 开发机」冲突，统一把上层的「工作区映射」称为 **TenantView**，把用户菜单里的「工作区」称为 **Workspace（开发机）**。

### 2.4 工作区 / 开发机（Workspace）

用户菜单「训练&推理 → 工作区」下的具体对象。语义为一台 **长驻的交互式开发容器**（Jupyter Notebook / VSCode Server / SSH 等）。底层复用 [`MLService(native, deployment)`](../core/compute-operator.md) 后端：

- Platform 自行维护 `workspaces` 表，记录开发机的归属用户、TenantView、镜像、ResourceUnit、底层 MLService 名称等；
- 创建 = 调 Compute 创建一个 `MLService(native, deployment)`，单 role、单副本，长驻容器；
- 启停 = patch 该 MLService 的 `roles[0].replicas`；
- 用户连接 = 通过 Envoy Gateway 反代或本地 port-forward 到容器端口。

详见 [workspace.md](workspace.md)。

### 2.5 数据卷（DataVolume，TBD）

预留概念，对应菜单「系统管理 → 数据卷管理」。可能的实现取向包括：用户级 PVC 抽象、数据集挂载路由（基于 Artifacts dataset）、集群 StorageClass 视图。本期暂不冻结字段，仅在概要中保留入口。

### 2.6 概念速查

| 术语 | 英文名 | 来源 / 对应对象 |
| --- | --- | --- |
| 用户 | User | Platform 内部表 |
| 角色 | Role | Platform 内部表，含内置四档 |
| 权限 | Permission | Platform 内部表，绑定到 Role |
| 租户视图 | TenantView | Platform 内部表 + 上游 `Tenant` CR |
| 工作区 / 开发机 | Workspace | Platform 内部表 + Compute `MLService(native, deployment)` |
| 数据卷 | DataVolume | TBD |

系统层共享的概念（Tenant、ResourcePool、ResourceUnit、Quota、Job、Service、Artifact）参见 [上层 overview §2](../overview.md#2-核心概念)。

## 3. 菜单与功能矩阵

Platform 的用户界面菜单与各功能详设的对应关系如下表。状态列表示设计文档的覆盖度（与代码实现进度无关）。

| 一级菜单 | 二级 | 详设位置 | 设计状态 |
| --- | --- | --- | :---: |
| Dashboard | — | 纯前端聚合视图，无独立后端详设 | ✅ (UI) |
| 应用中心 | 智能体 | 本文 §11 入口预留 | TBD |
| 应用中心 | Skills | 本文 §11 入口预留 | TBD |
| 应用中心 | MCP | 本文 §11 入口预留 | TBD |
| 训练&推理 | 工作区 | [workspace.md](workspace.md) | ✅ |
| 训练&推理 | 计算任务 | [job.md](job.md) | ✅ |
| 训练&推理 | 在线服务 | [service.md](service.md) | ✅ |
| 制品中心 | 模型 | [model.md](model.md) | ✅ |
| 系统管理 | 租户管理（含配额 Tab） | [tenant.md](tenant.md) | ✅ |
| 系统管理 | 资源池管理 | [resource-pool.md](resource-pool.md) | ✅ |
| 系统管理 | 资源单元管理 | [resource-unit.md](resource-unit.md) | ✅ |
| 系统管理 | 数据卷管理 | 本文 §11 入口预留 | TBD |
| 横切 | 用户 / 角色 / 鉴权 | [auth.md](auth.md) | ✅ |

图例：`✅` 表示已有详细设计或纯前端聚合，可直接进入实现；`TBD` 表示概要中保留入口，详设按需求展开。

## 4. 整体架构

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
   │                          tenant_views                     │
   │                          workspaces ...                   │
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

核心调用关系：

- 外部流量经 Envoy Gateway 进入 Platform；下游服务全部为内部调用，不直接暴露到集群外。
- Platform 调用 **Cluster Manager** 进行租户与配额管理（写 `Tenant` CR）。
- Platform 调用 **Compute** 在指定 namespace 下管理任务、服务、资源池、资源单元。
- Platform 调用 **Artifacts** 在指定 namespace 下管理模型（首版）；未来扩展到镜像、数据集、评估报告。
- Platform 自身仅持有「身份 + 视图映射」类轻量元数据；业务对象（Job / Service / Artifact）的权威分别在 Compute / Artifacts 的 PG 与 K8s CR 中。
- Dashboard 视图所需的指标数据来自基础设施层的 Prometheus（kube-prometheus-stack，详见 [infra.md](../infra/infra.md)），由 Platform 后端聚合 + 前端拼装。

## 5. 组件职责

### 5.1 前端

- **语言**：TypeScript
- **框架**：React
- **构建**：Vite（参见 [components/platform/frontend/README.md](../../../components/platform/frontend/README.md)）
- **目录约定**：`src/{pages,components,api,hooks,styles}` + `public/`，由前端组组织页面级设计；后端不约束 UI 结构。
- **职责**：提供登录、菜单导航、列表 / 详情 / 表单操作页，调用 Platform 后端 REST API；不直接调用任何下游服务。

页面级设计不在系统设计文档范围内，由前端开发按 [§3](#3-菜单与功能矩阵) 各详设给出的 API 字段独立设计。

### 5.2 后端

- **语言 / 框架**：Go + [Gin](https://github.com/gin-gonic/gin) + [GORM](https://gorm.io/) + [Cobra](https://github.com/spf13/cobra)
- **结构化日志**：[zap](https://github.com/uber-go/zap)
- **K8s 客户端**：仅在需要直接读 `Tenant.status` 时使用 `controller-runtime` client；其他操作一律走 Cluster Manager / Compute / Artifacts 的 REST API
- **职责**：
  - **外部 API 入口**：所有用户操作的 REST 入口；Platform 是 Cluster Manager / Compute / Artifacts 唯一的外部调用方；
  - **业务编排**：跨服务的串联逻辑（创建租户 → 写 TenantView → 校验 → 调下游）；
  - **身份与租户权限**：用户登录、JWT 颁发、RBAC 校验、租户访问边界；
  - **视图层映射**：把用户视角的「TenantView」「Workspace」翻译为下游需要的裸 namespace 字符串与 ElasticQuota CR 名。

后端代码骨架、PG schema 与下游客户端的具体形态见 [§7 后端架构与代码骨架](#7-后端架构与代码骨架)。

## 6. 核心调用链

下面给出三条覆盖关键能力的端到端时序，作为后续详设的连接锚点。所有时序中的下游 API 字段以各 `core/<service>.md` 为准。

### 6.1 用户提交训练任务

```
User ──► Platform.Login                                  // JWT 颁发
User ──► Platform.GET /api/v1/tenants                    // 列出可见 TenantView
User ──► Platform.POST /api/v1/jobs
        body: { tenant_view_id, pool, resource_unit_id,
                model_ref, image_ref, roles, ... }
   │
   ├─► Platform 内部：
   │     1. RBAC 校验（user 是否对该 TenantView 有 tenant-member 以上）
   │     2. 加载 TenantView → 取 (compute_namespace, artifacts_namespace)
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

### 6.2 用户上传 / 引用模型

```
User ──► Platform.POST /api/v1/models
        body: { tenant_view_id, name, version, displayName, ... }
   │
   ├─► RBAC + TenantView → artifacts_namespace
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

后续 MLJob / MLService 通过 `model_ref` 引用此模型时走 [§6.1](#61-用户提交训练任务) 的解析路径。

### 6.3 管理员创建租户 + 配额

```
Admin ──► Platform.POST /api/v1/admin/tenants
         body: { name, displayName, namespace, init_resources, quotas[] }
   │
   ├─► RBAC：必须 system-admin
   │
   ├─► ClusterManager.POST /api/v1/tenants
   │     body: CreateTenantRequest（透传）
   │   ◄── TenantResponse（含 spec.namespace 落地结果）
   │
   ├─► Platform 内部 PG：写 tenant_views(tenant_name, compute_namespace, artifacts_namespace, display_meta)
   │
   ◄── 200 OK + TenantView

Admin ──► Platform.POST /api/v1/admin/tenants/<name>/quotas
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

## 7. 后端架构与代码骨架

本节是 Platform 后端的工程契约，吸收了所有横切设计（代码组织、PG schema、下游客户端、配置、部署）。

### 7.1 仓库与目录布局

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
│   ├── tenant/                      # 租户视图 handler/service/repository
│   ├── workspace/                   # 开发机 handler/service/repository
│   ├── job/                         # 计算任务编排（无本地表，仅代理）
│   ├── service/                     # 在线服务编排
│   ├── model/                       # 模型 handler（代理 Artifacts）
│   ├── resourcepool/                # 资源池 handler（代理 Compute）
│   ├── resourceunit/                # 资源单元 handler（代理 Compute）
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

风格与 [components/compute/](../../../components/compute/) 保持一致；下游 typed client 的目录命名与各 `core/<service>.md` 中暴露的 REST 路径一一对应。

### 7.2 启动子命令

仿 `components/compute/cmd/compute/main.go`：

| 子命令 | 作用 |
| --- | --- |
| `serve` | 启动 HTTP API（默认）+ 后台任务（如租户状态轮询） |
| `bootstrap` | 一次性：检查并创建初始 `system-admin` 账号、内置角色与权限 |
| `migrate` | 执行 GORM 迁移；CI / 部署 init container 调用 |

### 7.3 错误处理

复用 RFC 7807 problem 风格（参考 [components/compute/internal/server/problem.go](../../../components/compute/internal/server/problem.go)）：

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

### 7.4 PG Schema

Platform 自有的所有 PG 表如下表。除身份与视图映射外不缓存任何下游业务元数据；Job / Service / Artifact 等业务对象一律向下游服务实时查询。

| 表 | 主键 | 关键字段 | 备注 |
| --- | --- | --- | --- |
| `users` | `id` (uuid) | `username` (uniq), `password_hash`, `email`, `display_name`, `disabled`, `created_at`, `updated_at` | 内置用户体系；外部 IdP 模式下退化为身份缓存 |
| `roles` | `id` (uuid) | `name` (uniq), `description`, `built_in` | `built_in=true` 的内置角色不可删除 |
| `permissions` | `id` (uuid) | `name` (uniq), `description` | 字典表 |
| `role_permissions` | `(role_id, permission_id)` | — | 多对多 |
| `user_tenant_roles` | `(user_id, tenant_name, role_id)` | `created_at` | 用户在某租户内的角色绑定；`tenant_name` 为外部值（指向 Tenant CR 名），不在本表加外键 |
| `tenant_views` | `tenant_name` | `compute_namespace`, `artifacts_namespace`, `display_name`, `description`, `icon`, `business_unit`, `created_at`, `updated_at` | 主键直接用 Tenant CR 名（cluster-scoped 唯一），便于与上游对齐 |
| `workspaces` | `id` (uuid) | `name`, `owner_user_id`, `tenant_name`, `compute_namespace`, `mlservice_name`, `image`, `resource_pool_id`, `resource_unit_id`, `quota`, `status`, `created_at`, `updated_at` | 详见 [workspace.md](workspace.md) |
| `sessions` | `jti` | `user_id`, `expires_at`, `revoked` | JWT 黑名单（登出 / 强制注销）；按 `expires_at` 定期清理 |
| `audit_logs` | `id` (bigserial) | `user_id`, `action`, `target`, `metadata` (jsonb), `created_at` | 关键管理员操作的审计；保留期由配置项指定 |

**索引约定**：`username`、`tenant_name`、`(owner_user_id, tenant_name)`、`(user_id, tenant_name)` 等高频查询字段建索引。所有时间戳字段采用 `timestamptz`。

各功能详设可声明自己的扩展表（例如 `model.md` 不需要本地表，`workspace.md` 复用上表中的 `workspaces`），但应避免在 Platform 内复制下游已经持有的对象元数据。

### 7.5 下游 typed client

每个下游服务一个独立子包，对外暴露强类型方法。所有 client 共享同一个工厂与中间件链：

```go
// internal/client/clustermanager/client.go
type Client interface {
    CreateTenant(ctx context.Context, in *CreateTenantInput) (*TenantView, error)
    AddQuota(ctx context.Context, tenant string, in *QuotaSpec) (*TenantView, error)
    // ...
}
```

横切约定：

- **身份透传**：所有出站请求自动携带 `X-Axisml-User: <username>` 头；下游服务（cluster-manager / compute / artifacts）信任此头并只做审计，不做鉴权。
- **超时与重试**：默认 30s 超时；幂等读操作允许有限次数指数退避重试，写操作不自动重试（Platform 把错误透传给前端）。
- **错误映射**：HTTP 4xx → 直接透传 problem；5xx → 包装成 Platform 的 `type=https://axisml.io/errors/upstream-failure`，附带下游服务名。
- **可观测性**：每次调用打 zap 日志 + Prometheus 指标 `platform_upstream_request_total{service,method,status}`。

### 7.6 启动配置

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
| `--audit-log-retention-days` | `90` | 审计日志保留期 |

环境变量同名上标 `AXISML_` 前缀（与现有组件一致，由 `internal/config` 解析）。

### 7.7 镜像与 Makefile

参考 [components/compute/Dockerfile](../../../components/compute/Dockerfile) 与 [components/compute/Makefile](../../../components/compute/Makefile)：

- 镜像：`ghcr.io/axisml/axisml-platform-backend:<IMAGE_TAG>`，`IMAGE_TAG` 必须等于 [`deploy/helm/axisml-system/Chart.yaml`](../../../deploy/helm/axisml-system/Chart.yaml) 的 `appVersion`，由顶层 Makefile 统一注入；
- 多阶段构建从仓库根目录开始，复制必要的兄弟模块（不依赖 operator 模块时仅需 `components/platform/backend/`）；
- 最终镜像基于 `gcr.io/distroless/static:nonroot`，以 `65532:65532` 运行；
- 标准目标：`build` / `test` / `image` / `image-load-minikube` / `clean` / `fmt` / `vet` / `tidy` / `doc-gen` / `integration` / `coverage`。

前端镜像由 [components/platform/frontend/Makefile](../../../components/platform/frontend/Makefile) 单独构建，最终通过 Helm 模板部署，与后端松耦合。

## 8. 认证与租户权限模型（概要）

详细字段、API 路径与权限矩阵见 [auth.md](auth.md)；本节仅给出概要：

- **第一版**：内置 `users` 表 + bcrypt 密码哈希 + JWT；`sessions` 表存登出黑名单。
- **RBAC**：内置四档角色（`system-admin` / `tenant-admin` / `tenant-member` / `viewer`），权限通过 `role_permissions` 多对多绑定。
- **租户绑定**：`user_tenant_roles(user_id, tenant_name, role_id)` 表达「某用户在某租户中的角色」；前端可见的 TenantView 列表 = 该用户在 `user_tenant_roles` 中的所有 `tenant_name`。
- **OIDC 预留**：`internal/auth/IdentityProvider` 接口屏蔽用户来源；`internal` 实现读 `users` 表，`oidc` 实现按需接入；切换由 `--auth-mode` 控制。本期只交付 `internal`。
- **下游透传**：Platform 校验通过后向下游服务注入 `X-Axisml-User`；下游不做二次鉴权。

## 9. 关键设计决策

| 决策项 | 决策 | 理由 |
| --- | --- | --- |
| 文档组织粒度 | 按二级菜单（子功能）拆文档 + 一份横切 `auth.md` | 单文件保持短小、便于评审与并行迭代；按一级菜单聚合后单文档过长不利阅读 |
| 后端语言与框架 | Go + Gin + GORM + Cobra，与 Compute 一致 | 复用现有组件骨架与 CI 流水线，降低维护成本 |
| 认证方式 | 内置用户 + RBAC，`IdentityProvider` 接口预留 OIDC | 第一版自建集群可独立运行；外部 IdP 按需接入而非默认依赖 |
| 工作区实现 | 复用 `MLService(native, deployment)` 承载开发机 | 避免新增 CRD；与在线服务共用 backend handler |
| 租户视图 | Platform 自身 PG 维护 `tenant_name → (compute_ns, artifacts_ns)` | 下游服务保持 namespace-only 单一职责，不感知租户 |
| 配额 UI 归属 | 租户管理详情页内 Tab，不独立菜单 | 配额始终在 (tenant, pool) 上下文中操作 |
| Platform PG 范围 | 仅存身份与视图映射，不缓存下游业务元数据 | 业务对象权威在 Compute / Artifacts，避免双写漂移 |
| 制品 UI 范围 | 制品中心首版只覆盖模型 | 与菜单一致；dataset / image / eval_report 在本文 §11 列入口，复用 Artifacts API 但 UI 后续迭代 |
| Dashboard | 纯前端聚合视图，无独立后端详设 | 数据全部来自 Job / Service / Artifact / Quota 列表 API + Prometheus 查询 |
| 用户身份透传 | Platform → 下游统一注入 `X-Axisml-User` 头 | 下游服务保持只接受内部调用、信任 Platform 鉴权 |

## 10. 部署架构

Platform 随 [`axisml-system`](../../../deploy/helm/axisml-system/) chart 一起发布，模板位于 [`templates/platform/`](../../../deploy/helm/axisml-system/templates/platform/)：

```
deploy/helm/axisml-system/templates/platform/
├── deployment.yaml      # Backend Deployment（启用 .Values.platform.enabled）
├── service.yaml         # ClusterIP，默认 8080
├── configmap.yaml       # 注入下游 URL 与运行时环境变量
└── ingress.yaml         # 可选；生产环境建议关闭，由 Envoy Gateway 暴露
```

ConfigMap 注入的关键环境变量：

| 环境变量 | 来源 | 说明 |
| --- | --- | --- |
| `ML_CLUSTER_MANAGER_URL` | `axisml-system` chart | 指向 cluster-manager Service（**待新增**） |
| `ML_COMPUTE_URL` | `axisml-system` chart | 已存在 |
| `ML_ARTIFACTS_URL` | `axisml-system` chart | 已存在 |
| `ML_PROMETHEUS_URL` | `axisml-system` chart | Dashboard 数据源；指向 `axisml-infra` namespace 下的 Prometheus |

外部流量必须经 [Envoy Gateway](../infra/infra.md) 进入 Platform；Cluster Manager / Compute / Artifacts 始终保持 ClusterIP，不接受外部直连。

PostgreSQL 沿用 `axisml-system` chart 的内置实例或 `externalDatabase` 配置；Platform 与 Compute / Artifacts 共享同一个 PG 实例（按 schema / table prefix 隔离），与上层 [overview §6](../overview.md#6-部署架构) 一致。

前端镜像通过 Helm 模板的 `platform.frontend.image` 字段独立部署，本期与后端共用一个 Service；后续如需独立的 SSR / Edge 渲染可再拆分。

## 11. 后续迭代与 TBD

按当前菜单与设计文档树，以下能力作为后续迭代项保留入口：

- **应用中心**：智能体（Agent）/ Skills / MCP 三个子菜单仅在前端预留路由；后端契约、数据模型与 IdP 集成在该方向需求稳定后再展开。
- **数据卷管理**：可能的实现方向包括 PVC 抽象、数据集挂载路由、集群 StorageClass 视图，本期不冻结字段。
- **OIDC 接入**：`auth.md` 给出接口签名后即可切换，但本期只交付 `internal`。
- **IDE 内嵌**：开发机当前通过反代 / port-forward 访问；后续可在前端集成 code-server / JupyterLab 嵌入式视图。
- **制品扩展**：dataset / image / eval_report 三类 Artifact 已被 Artifacts 服务支持，仅缺 UI；UI 完工后从 §3 矩阵中升级为 ✅。
- **审计与告警**：`audit_logs` 表已规划，下一步是 UI 视图与告警规则模板。
- **多集群 / 多区域**：当前所有概念按单集群假设；多集群方向作为远期演进。
