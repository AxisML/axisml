# AxisML Platform 租户管理 详细设计

本文档是 AxisML Platform 子系统下 **「系统管理 → 租户管理」** 一级功能的全栈设计，承接 [PRD §6.4.1 租户管理](../../product/prd.md#641-租户管理) 与系统层 [cluster-manager](../core/cluster-manager.md) / [tenant-operator](../core/tenant-operator.md) 之间的 Platform 入口：租户列表 / 详情、租户内成员↔角色绑定、租户管理 UI 与 REST 入口。

| 模块 | 职责 | 边界外 |
| --- | --- | --- |
| Tenant 视图与生命周期（[§4](#4-菜单与列表页) / [§7.1](#71-租户-crud)） | 系统管理员 CRUD 租户、暂停 / 恢复 / 删除；按用户身份过滤可见租户 | Tenant CR 字段语义、底层 Namespace / ElasticQuota 落地（→ [cluster-manager.md](../core/cluster-manager.md) / [tenant-operator.md](../core/tenant-operator.md)）|
| 配额（[Tab 2](#tab-2-配额) / [§7.2](#72-配额)） | 系统 / 租户管理员对 `Tenant.spec.quotas[]` 的可视化 CRUD | ElasticQuota CR 命名 / 调度行为；分层配额（→ [tenant-operator.md §4.6.2](../core/tenant-operator.md#462-elasticquota)）|
| 成员（[Tab 3](#tab-3-成员) / [§7.3](#73-成员)） | 本租户内的用户↔角色绑定 UI 与 REST 入口 | 用户身份来源、内置角色定义、跨租户 RBAC 矩阵（→ `auth.md`）|

**关键不变式：**

> Platform 自有 PG 仅持有 `tenants`（展示元数据 + Platform 级 `name` 标识）与 `user_tenant_roles`（成员绑定）；租户的 `spec` / `status` / `quotas` 不缓存，所有详情 GET 实时穿透到 cluster-manager。
>
> `compute_namespace` / `artifacts_namespace` 不是租户实体的属性；下游 Job / Service / Artifact 调用各自携带 `namespace` 字段（典型来源：`Tenant.spec.namespace.name`）。

**文档组织：**

- **Part I — 服务边界**（[§1](#1-概述与定位) – [§3](#3-数据模型platform-自有部分)）：定位、可见性矩阵、Platform 自有数据模型。
- **Part II — UI 设计**（[§4](#4-菜单与列表页) – [§6](#6-详情页-tab)）：菜单 / 列表页 / 创建表单 / 详情页 Tab。
- **Part III — 后端 API 契约**（[§7](#7-rest-路径与响应格式)）：REST 路径、字段、RBAC。
- **Part IV — 后端实现**（[§8](#8-模块结构)）：模块结构、RBAC 装配、一致性策略、PG schema、可观测性。
- **Part V — 实施与验证**（[§9](#9-实现路径) – [§13](#13-相关引用)）：阶段化实现、关键决策、后续迭代、测试、参考。

---

## Part I — 服务边界

## 1. 概述与定位

「租户管理」是系统管理菜单下面向系统管理员与租户管理员的入口，覆盖以下能力：

- 系统管理员：创建 / 暂停 / 恢复 / 删除租户；为租户在各资源池下设置初始配额；维护租户展示元数据；维护租户成员与角色。
- 租户管理员：在已开通的租户内，按业务线 / 团队拆分配额；维护本租户成员与角色。

不在范围内的能力：

- Tenant CR 字段语义、可变性约束、底层资源落地：见 [cluster-manager.md §3](../core/cluster-manager.md#3-tenant-api) 与 [tenant-operator.md §4](../core/tenant-operator.md#4-tenant-controller)。
- 用户登录、JWT、内置角色矩阵、`IdentityProvider` 抽象：见 `auth.md`。
- 资源池 / 资源单元的定义与维护：见 `resource-pool.md` / `resource-unit.md`。
- 数据卷管理：[platform/overview.md §11](overview.md#11-后续迭代与-tbd) 标记为 TBD。

## 2. 角色与可见性矩阵

下表只列与本功能相关的能力；persona ↔ RBAC 角色映射沿用 [platform/overview.md §2.2](overview.md#22-用户角色persona)。

| 能力 | `system-admin` | `tenant-admin@self` | `user@self` |
| --- | :---: | :---: | :---: |
| 列出可见租户 | 全集群 | 仅自己绑定的租户 | 仅自己绑定的租户 |
| 查看租户详情（基本信息） | ✅ | ✅ | ✅ |
| 创建 / 编辑展示元数据 / 暂停 / 恢复 / 删除租户 | ✅ | ❌ | ❌ |
| 查看本租户配额 | ✅ | ✅ | ✅ |
| 新增 / 修改 / 删除本租户配额 | ✅ | ✅ | ❌ |
| 查看本租户成员 | ✅ | ✅ | ❌ |
| 增 / 改 / 删本租户成员 | ✅ | ✅ | ❌ |

`@self` 表示「在该 `{name}` 租户上具备相应角色」。`system-admin` 在所有租户级动作上短路放行，不要求其在 `user_tenant_roles` 中显式绑定。

## 3. 数据模型（Platform 自有部分）

Platform 自身只为租户管理引入两张 PG 表，其余信息向 cluster-manager 实时穿透。

### 3.1 `tenants` 表

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | uuid PK | 内部稳定标识；`user_tenant_roles` / `workspaces` 等关联表通过该列引用 |
| `name` | text uniq | 租户名；REST 路径中的 `{name}` 即此列；同时透传作为 Tenant CR 的 `metadata.name`，受 cluster-manager DNS-1123 校验。创建后不可变 |
| `display_name` | text | UI 展示名，人类可读 |
| `description` | text | 自由文本说明 |
| `business_unit` | text | 业务线 / 部门归属，用于列表过滤 |
| `created_at` / `updated_at` | timestamptz | GORM 自动维护 |

约束：

- `name` 同时担任 Platform URL 锚点与 Tenant CR `metadata.name`，与 cluster-manager 同名锚点。
- 不缓存 Tenant CR 的 `spec` / `status` / `quotas`：所有 phase / namespaceReady / quotas 用量在 GET 时实时调 cluster-manager（按 `name` 寻址）。
- 不存 `compute_namespace` / `artifacts_namespace`：下游 Job / Artifact 调用在请求体上独立携带 `namespace`，由调用上下文（Workspace、表单输入或 Tenant CR `spec.namespace.name`）决定。

### 3.2 `user_tenant_roles` 关联

成员管理消费 `user_tenant_roles(user_id, tenant_id, role_id)` 三元组：

- `tenant_id` 引用 `tenants.id`（同库 FK，cascade 行为见 `auth.md`）。
- 删除租户前需校验该 tenant 下无 `user_tenant_roles` 行（[§7.1](#71-租户-crud) DELETE）。

字段、PK、索引细节由 `auth.md` 给出。

---

## Part II — UI 设计

## 4. 菜单与列表页

菜单位置：「系统管理 → 租户管理」。

### 4.1 列表页

| 列 | 来源 | 说明 |
| --- | --- | --- |
| `displayName` | `tenants.display_name` | 主展示列 |
| `name` | `tenants.name` | 字符串锚点；URL 中的 `{name}` 即此列 |
| `business_unit` | `tenants.business_unit` | 列过滤 |
| 状态 | Tenant CR `status.phase` | `Active` / `Suspended` / `Failed` |
| 命名空间 | Tenant CR `spec.namespace.name` | 只读展示 |
| 创建时间 | `tenants.created_at` | |
| 操作 | — | 详情 / 暂停 / 恢复 / 删除 |

- 过滤：状态、business_unit、关键字（display_name / name 模糊匹配）。
- 列表渲染：Platform 后端把 PG `tenants` 与 cluster-manager `LIST tenants` 结果按 `name` join 后返回；状态字段不在 PG 缓存，cluster-manager 自身走 controller-runtime cache，单次 RT 可控。
- 列表可见性：
  - `system-admin`：全集群所有 Tenant CR。
  - 其他角色：仅 `user_tenant_roles` 中显式绑定的 `tenant_id` 对应的租户。

### 4.2 操作按钮

- **详情** — 任何在该租户上有绑定的角色可点击。
- **暂停 / 恢复** — 仅 `system-admin`；分别调 `POST /api/v1/tenants/{name}/suspend`、`/unsuspend`。
- **删除** — 仅 `system-admin`；调 `DELETE /api/v1/tenants/{name}`，需二次确认弹窗，并展示「同租户成员数 / 当前 phase / 是否 Suspended」。

## 5. 创建租户表单（system-admin only）

| UI 字段 | 写入位置 | 说明 |
| --- | --- | --- |
| `name` | Tenant CR `metadata.name` + `tenants.name` | DNS-1123、≤40 字符；DNS 校验由 cluster-manager 兜底 |
| `displayName` | Tenant CR `spec.displayName` + `tenants.display_name` | 双写以便列表 / 详情两侧展示 |
| `description` / `businessUnit` | `tenants` 同名列 | 仅 Platform 本地展示，cluster-manager 不感知 |
| `namespace.name` | Tenant CR `spec.namespace.name` | 创建后不可变（详见 [tenant-operator.md §4.3.3](../core/tenant-operator.md#433-字段归属与不可变性)） |
| `namespace.labels` / `annotations` | Tenant CR `spec.namespace.{labels, annotations}` | 仅在 Namespace 首次创建时落地 |
| `quotas[]` | Tenant CR `spec.quotas[]` 透传 | 每条 `(pool, name, min, max)`；`(pool, name)` 创建后不可变 |
| `initResources` | Tenant CR `spec.initResources` 透传 | 列出 `name` + `sourceXxxRef`；UI 不构造源 Secret 数据，由集群管理员预置 |

校验：

- 必填项 + 长度提示：UI 即时校验。
- 跨字段约束（`min ≤ max`、`(pool, name)` 唯一）：UI 即时校验作为快速反馈；最终以 cluster-manager 返回为准。
- DNS-1123 / namespace denylist：UI 不前置查询，依赖 cluster-manager 4xx 反馈，把 problem `detail` 透传到表单错误位。

## 6. 详情页 Tab

详情页以 `name` 为维度，分为四个 Tab：基本信息、配额、成员、审计日志。

### Tab 1 基本信息

展示：

- `display_name` / `description` / `business_unit`（来自 PG `tenants`，可编辑）。
- 状态卡片：Tenant CR `status.phase` / `namespaceReady` / `conditions[]`（来自 cluster-manager，只读）。
- 命名空间：Tenant CR `spec.namespace.name`（只读）。

操作：

- **编辑展示元数据**（`system-admin`）：仅写本地 PG，不调 cluster-manager；表单字段限于 `display_name` / `description` / `business_unit`。
- **暂停 / 恢复**（`system-admin`）：调对应 endpoint。
- **删除**（`system-admin`）：与列表页同步逻辑。

### Tab 2 配额

按 `pool` 分组的二级表格：

```
[default 池]                                        [+ 新增配额]
┌──────────┬──────────────────┬──────────────────┬───────┬─────┐
│ 配额名    │ min              │ max              │ used  │ 操作 │
├──────────┼──────────────────┼──────────────────┼───────┼─────┤
│ default  │ cpu=20           │ cpu=100          │ cpu=8 │ ✏️ ❌│
│ training │ cpu=10, gpu=2    │ cpu=50,  gpu=8   │ ...   │ ✏️ ❌│
└──────────┴──────────────────┴──────────────────┴───────┴─────┘
合计 max（仅展示）：cpu=150, gpu=8
```

- 数据来源：`GET /api/v1/tenants/{name}/quotas` 实时穿透 cluster-manager；`status.quotas[].used` 等字段直接展示。
- 写权限：`system-admin` / `tenant-admin@self`。
- 拆分总额：Platform 不做 sum(max) 上限校验；表头展示「合计 max」聚合视图作肉眼参考。
- 不可变约束：`(pool, name)` 创建后不可变，UI 在编辑态置灰；改名等同于「先删后建」，由用户明确。

### Tab 3 成员

| 列 | 来源 |
| --- | --- |
| 用户名 / display_name / email | `users` 表 |
| 角色 | `roles.name`，限 `tenant-admin` / `user` |
| 加入时间 | `user_tenant_roles.created_at` |
| 操作 | 改角色 / 移除 |

操作：

- **添加成员**：用户搜索（[§7.4](#74-用户搜索)）→ 角色下拉 → 提交。
- **修改角色**：行内角色下拉。
- **移除成员**：行内移除。
- 写权限：`system-admin` / `tenant-admin@self`。
- 角色限制：本接口仅可绑定 `tenant-admin` / `user`；`system-admin` 由全局用户管理菜单维护。
- 自我保护：当前操作者不能移除 / 降级自己最后一个 `tenant-admin` 角色，否则该租户无人可管，返回 `409 last-tenant-admin`。

### Tab 4 审计日志

入口保留，详细字段见 [§11 后续迭代](#11-后续迭代)。

---

## Part III — 后端 API 契约

## 7. REST 路径与响应格式

- 所有路径统一在 `/api/v1/tenants/...`；权限差异由 RBAC 中间件按角色 + 资源所有权判定。
- 错误格式：RFC 7807 problem+json，复用 [platform/overview.md §7.3](overview.md#73-错误处理) 的样例。
- 出站调用：`internal/client/clustermanager` typed client，定义见 [overview.md §7.5](overview.md#75-下游-typed-client)。
- 每个 endpoint 在下文括号内标注允许的角色：`system-admin` 表示全局；`tenant-admin@self` / `user@self` 表示「在该 `{name}` 租户上具备该角色」。

### 7.1 租户 CRUD

#### `POST /api/v1/tenants`（`system-admin`）

请求体：

```json
{
  "name": "team-a",
  "displayName": "Team A",
  "description": "推理团队",
  "businessUnit": "infra",
  "namespace": { "name": "team-a", "labels": {}, "annotations": {} },
  "quotas": [
    { "pool": "default", "name": "default", "min": {}, "max": { "cpu": "100" } }
  ],
  "initResources": { "imagePullSecrets": [], "secrets": [], "configMaps": [], "serviceAccounts": [] }
}
```

写入顺序：先调 `clustermanager.Client.CreateTenant`，成功后写 `tenants` 行；详见 [§8.2](#82-一致性策略双写-cluster-manager--本地-pg)。响应为 `Tenant` DTO（结构与下方 GET 响应一致）。

#### `GET /api/v1/tenants`（已登录即可，按角色裁剪）

- `system-admin`：全集群所有 Tenant CR + 对应 `tenants` 行 join。
- 其余角色：先按 `user_tenant_roles.user_id = current_user` 取 `tenant_id` 集合，join `tenants` 拿 `name` 集合，按集合过滤 cluster-manager 的 LIST 结果。
- 支持 query：`status` / `business_unit` / `q`（关键字）/ `limit` / `continue`。

#### `GET /api/v1/tenants/{name}`（`system-admin` 或 `user@self` 及以上）

返回 `Tenant` DTO：

```json
{
  "name": "team-a",
  "displayName": "Team A",
  "description": "推理团队",
  "businessUnit": "infra",
  "namespace": "team-a",
  "phase": "Active",
  "namespaceReady": true,
  "conditions": [...],
  "createdAt": "...",
  "updatedAt": "..."
}
```

`phase` / `namespaceReady` / `conditions` / `namespace` 来自 `Tenant.{spec, status}`；其余来自 PG `tenants`。

#### `PATCH /api/v1/tenants/{name}`（`system-admin`）

双写：

- **可变 spec 字段**（`displayName` / `annotations` / `namespace.labels` / `namespace.annotations` / `initResources`）→ cluster-manager。
- **展示元数据**（`displayName` / `description` / `businessUnit`）→ 本地 PG。`displayName` 双写到两侧。
- 提前拦截不可变字段：`name` / `namespace.name` / `quotas[].(pool, name)`，违反返回 `400 immutable-field`。

#### `POST /api/v1/tenants/{name}/suspend` / `unsuspend`（`system-admin`）

直接透传 cluster-manager 同名 endpoint。

#### `DELETE /api/v1/tenants/{name}`（`system-admin`）

顺序：

1. 校验 `user_tenant_roles WHERE tenant_id = (SELECT id FROM tenants WHERE name = :name)` 行数；非空 → `409 tenant-has-members`。
2. 调 `clustermanager.Client.DeleteTenant`。
3. 成功后删 `tenants` 行。

### 7.2 配额

所有写接口透传 [cluster-manager.md §4](../core/cluster-manager.md#4-quota-api) 同名 endpoint。

| Endpoint | 权限 | 说明 |
| --- | --- | --- |
| `GET /api/v1/tenants/{name}/quotas` | `system-admin` 或 `user@self` 及以上 | 返回 `spec.quotas[]` + `status.quotas[]` |
| `POST /api/v1/tenants/{name}/quotas` | `system-admin` 或 `tenant-admin@self` | body `{pool, name, min, max}` |
| `PATCH /api/v1/tenants/{name}/quotas/{pool}/{quotaName}` | `system-admin` 或 `tenant-admin@self` | body `{min?, max?}` |
| `DELETE /api/v1/tenants/{name}/quotas/{pool}/{quotaName}` | `system-admin` 或 `tenant-admin@self` | — |

Platform 不做 sum(max) 上限校验；详见 [§10 关键设计决策](#10-关键设计决策)。

### 7.3 成员

| Endpoint | 权限 | 说明 |
| --- | --- | --- |
| `GET /api/v1/tenants/{name}/members` | `system-admin` 或 `tenant-admin@self` | 联表 `user_tenant_roles` + `users` + `roles` |
| `POST /api/v1/tenants/{name}/members` | 同上 | body `{user_id, role_name}`；`role_name` ∈ `{tenant-admin, user}` |
| `PATCH /api/v1/tenants/{name}/members/{user_id}` | 同上 | body `{role_name}` |
| `DELETE /api/v1/tenants/{name}/members/{user_id}` | 同上 | 移除绑定 |

约束：

- `role_name` 不允许 `system-admin`，否则返回 `400 role-not-bindable`。
- 当前操作者尝试移除 / 降级自己最后一个 `tenant-admin` 角色 → `409 last-tenant-admin`。
- 添加成员时被绑定的 `user_id` 必须存在；不存在 → `404 user-not-found`。

### 7.4 用户搜索

`GET /api/v1/users?q=`：在「添加成员」时供前端联想搜索。返回字段最小集 `{id, username, display_name, email}`。该接口由 `auth.md` 定义，本文档仅声明对其的依赖。

---

## Part IV — 后端实现

## 8. 模块结构

目录：`components/platform/backend/internal/tenant/`

| 文件 | 职责 |
| --- | --- |
| `handler.go` | Gin 路由注册（`/api/v1/tenants` 前缀）、请求解析、RBAC gate 装配、problem 渲染 |
| `service.go` | 业务编排：双写 cluster-manager + 本地 PG、suspend / unsuspend 透传、不可变字段拦截、成员增删校验 |
| `repository.go` | GORM 操作 `tenants` 与 `user_tenant_roles` |
| `dto.go` | 请求 / 响应类型；与 cluster-manager API DTO 之间的显式映射 |
| `view.go` | `Tenant` DTO 合并器：把 cluster-manager 返回的 Tenant CR 字段与 `tenants` 行融合 |

风格沿用 [platform/overview.md §7.1](overview.md#71-仓库与目录布局) 与 [components/compute/](../../../components/compute/) 的 handler/service/repository 三层。

### 8.1 RBAC 中间件接入

`internal/auth` 提供 `RequireSystemAdmin` / `RequireTenantRole(role, tenantParam)` 中间件，定义见 `auth.md`。本文档使用：

| 路由 | 中间件链 |
| --- | --- |
| `POST/PATCH/DELETE /api/v1/tenants[...]`、`POST /api/v1/tenants/{name}/suspend`、`/unsuspend` | `RequireSystemAdmin` |
| `POST/PATCH/DELETE /api/v1/tenants/{name}/quotas[...]` | `RequireTenantRole("tenant-admin", "name")`，`system-admin` 短路 |
| `POST/PATCH/DELETE /api/v1/tenants/{name}/members[...]` | `RequireTenantRole("tenant-admin", "name")`，`system-admin` 短路 |
| `GET /api/v1/tenants/{name}` / `GET .../quotas` | `RequireTenantRole("user", "name")`，`system-admin` 短路 |
| `GET /api/v1/tenants/{name}/members` | `RequireTenantRole("tenant-admin", "name")`，`system-admin` 短路 |
| `GET /api/v1/tenants` | 已登录即可；handler 内部按角色裁剪可见集合 |

### 8.2 一致性策略（双写 cluster-manager + 本地 PG）

cluster-manager 是 Tenant CR 的权威；本地 `tenants` 仅承载展示元数据。一致性按「权威先行」组织。

**创建**：

1. 调 `clustermanager.Client.CreateTenant`：失败 → 4xx/5xx 直接返给前端，本地 PG 不写。
2. 成功 → 写本地 `tenants` 行：写入失败 → 不回滚 cluster-manager；返回 5xx + problem detail，提示「Tenant CR 已创建但视图写入失败」；后台保留 inline retry，再失败由 [§11](#11-后续迭代) 持久化补偿队列接管。

**删除**：

1. 校验 `user_tenant_roles` 是否仍有该 tenant 行；非空 → `409 tenant-has-members`。
2. 调 `clustermanager.Client.DeleteTenant`：失败 → 4xx/5xx，本地不变。
3. 成功 → 删本地 `tenants` 行。
4. **孤儿处理**：若本地存在但 cluster-manager 返回 `404`，下次 GET 时归并清理。

**更新**：

- 可变 spec 字段 → cluster-manager；展示元数据 → 本地 PG，二者独立写入。
- 任一失败 → problem `type=https://axisml.io/errors/tenant-update-partial`，body 列出 `applied` / `failed` 字段集合。
- 不可变字段强校验：`name` / `namespace.name` / `quotas[].(pool, name)`，违反返回 `400 immutable-field`，从不触达 cluster-manager。

### 8.3 状态读取

- 任何展示 Tenant CR 字段（`phase` / `namespaceReady` / `quotas[].used` / `conditions`）的请求都调 `clustermanager.Client.GetTenant(name)`，不本地缓存。
- 不引入 K8s informer：tenant 操作天然低频；cluster-manager 端走 controller-runtime cache，单租户 GET 延迟充分可控。

### 8.4 PG schema

- `tenants`：见 [§3.1](#31-tenants-表)。
- `user_tenant_roles`：定义见 `auth.md`；本模块只消费 `(user_id, tenant_id, role_id)` 三元组。

两表均在 `migrate` 子命令中自动迁移（详见 [overview.md §7.2](overview.md#72-启动子命令)）。

### 8.5 度量与日志

Prometheus 指标（特有于本功能；通用上游调用指标见 [overview.md §7.5](overview.md#75-下游-typed-client)）：

- `platform_tenant_action_total{action, status}`：`action ∈ {create, update_meta, suspend, unsuspend, delete, quota_create, quota_update, quota_delete, member_add, member_update, member_remove}`，`status ∈ {success, failure}`。
- `platform_tenant_consistency_partial_total{phase}`：`phase ∈ {create_local_failed, update_partial}`，记录 [§8.2](#82-一致性策略双写-cluster-manager--本地-pg) 中部分成功的次数。

zap 字段约定：每条租户操作日志必带 `tenant_id` / `tenant_name` / `actor_user` / `action` / `status`；删除 / suspend / 成员变更额外带 `target_user` / `role_name`（如适用）。

---

## Part V — 实施与验证

## 9. 实现路径

### 9.1 阶段一（MVP）

| 模块 | 范围 | 完成信号 |
| --- | --- | --- |
| handler / service / repository / dto | [§7.1](#71-租户-crud) / [§7.2](#72-配额) / [§7.3](#73-成员) 全部 endpoint | `make platform-build` 通过 |
| RBAC 装配 | [§8.1](#81-rbac-中间件接入) 路由表全部接通；`system-admin` 短路逻辑生效 | 单元测试覆盖中间件分支 |
| PG 迁移 | `tenants` 表创建；`user_tenant_roles` 由 `auth.md` 提供 | `make platform-migrate` 干净 |
| Integration | testcontainers PG + cluster-manager fake，覆盖 happy path（创建 → 加成员 → 加配额 → suspend → 删除） | `make platform-integration` 通过 |

### 9.2 阶段二

1. cluster-manager 双写补偿持久化：把 inline retry 替换为 PG outbox 表 + 后台 worker。
2. 配额 Tab 表头「合计 max」聚合视图。
3. 列表页「按 phase 分组」聚合视图。

### 9.3 阶段三 / TBD

详见 [§11 后续迭代](#11-后续迭代)。

## 10. 关键设计决策

| 决策项 | 决策 | 理由 |
| --- | --- | --- |
| 视图与 Tenant CR 边界 | `tenants` 表只存展示元数据；不缓存 spec / status / quotas / namespace | 避免双写漂移；权威留在 cluster-manager；namespace 由下游 Job/Artifact 调用各自携带 |
| 命名空间归属 | `compute_namespace` / `artifacts_namespace` 不属于租户实体 | 与下游 namespace 分区模型一致；Platform 不维护重复映射 |
| API 路径形态 | 统一在 `/api/v1/tenants/...`，权限走中间件 | 路径按资源组织；权限差异完全由 RBAC 中间件按角色 + 资源所有权判定 |
| 详情页 Tab 划分 | 基本信息 / 配额 / 成员 / 审计日志 | 减少首屏字段密度；每个 Tab 对应独立 REST 资源 |
| 配额拆分总额校验 | Platform 不做；仅作转发 | ElasticQuota 当前为扁平结构，不天然支持父子关系；维持单一职责 |
| 创建 / 删除一致性 | cluster-manager 优先 → 本地 PG 后行 | cluster-manager 是权威；本地表丢失可重建，反之不可 |
| 不可变字段拦截 | `name` / `namespace.name` / `quotas[].(pool, name)` | 在 Platform 层拦截，减少 cluster-manager 4xx 与前端误用 |
| 成员保护 | 不能移除自己最后一个 `tenant-admin` 角色 | 防止租户失管 |
| 成员角色集合 | 添加 / 修改成员仅允许 `tenant-admin` / `user` | `system-admin` 是平台级角色，不在租户菜单内绑定 |

## 11. 后续迭代

- **审计日志 Tab**：复用 [overview.md §7.4](overview.md#74-pg-schema) 的 `audit_logs` 表；落地按 `tenant_id` 过滤的索引、UI 表格、保留期可配置项。
- **cluster-manager 双写补偿持久化**：把 [§8.2](#82-一致性策略双写-cluster-manager--本地-pg) 中的 inline retry 替换为持久化 outbox 表 + 后台 worker。
- **配额硬校验 / 分层配额**：等上游 ElasticQuota 提供 `parent` 字段后，Platform 可在 Tab 2 引入「上限 cap」视图与硬阻断。
- **init_resources 表单深度**：当前仅暴露 `sourceXxxRef`；后续可接入 Vault / Sealed Secrets 等加密源直接创建。
- **租户软删除**：当前 `DELETE` 直接清理；未来可加 `archived_at` 状态以便恢复。
- **租户克隆**：基于已有租户的 `quotas` / `initResources` 模板快速创建新租户。

## 12. 测试策略

- **单元**（`internal/tenant/*_test.go`）：
  - service 层一致性策略（cluster-manager 失败 / 本地失败 / 双失败）。
  - 不可变字段拦截逻辑。
  - 最后一个 `tenant-admin` 校验逻辑。
  - 列表可见性裁剪（不同 RBAC 角色）。
- **integration**（`components/platform/backend/test/integration/`）：
  - testcontainers PostgreSQL；in-process cluster-manager fake（httptest）模拟 Tenant CR 行为，包括成功 / 4xx / 5xx 响应。
  - happy path：创建租户 → 加成员 → 加配额 → suspend → unsuspend → 删除。
  - 故障注入：cluster-manager 创建后本地 PG 写入失败的补偿语义；删除时 `user_tenant_roles` 非空的 409。
  - RBAC：`system-admin` / `tenant-admin@self` / `user@self` 在每个 endpoint 上的允许 / 拒绝矩阵。
- 不引入 envtest：Platform 本身不直读 K8s API；端到端校验由 [cluster-manager.md §6](../core/cluster-manager.md#6-测试) / [tenant-operator.md §6](../core/tenant-operator.md#6-测试) 在自身集成层覆盖。

## 13. 相关引用

- [PRD §6.4.1 租户管理](../../product/prd.md#641-租户管理)
- [docs/system_design/overview.md](../overview.md)
- [docs/system_design/platform/overview.md](overview.md)
- [docs/system_design/core/cluster-manager.md](../core/cluster-manager.md)
- [docs/system_design/core/tenant-operator.md](../core/tenant-operator.md)
- `auth.md`
