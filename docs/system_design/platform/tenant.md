# AxisML Platform 租户管理 详细设计

本文档是 AxisML Platform 子系统下 **「系统管理 → 租户管理」** 一级功能的全栈设计，承接 [PRD §6.4.1 租户管理](../../product/prd.md#641-租户管理) 与系统层 [cluster-manager](../core/cluster-manager.md) / [tenant-operator](../core/tenant-operator.md) 之间的 Platform 入口：租户列表 / 详情、租户内成员↔角色绑定、租户管理 UI 与 REST 入口。

| 模块 | 职责 | 边界外 |
| --- | --- | --- |
| Tenant 视图与生命周期（[§4](#4-菜单与列表页) / [§7.1](#71-租户-crud)） | 系统管理员 CRUD 租户、暂停 / 恢复 / 删除；按用户身份过滤可见租户 | Tenant CR 字段语义、底层 Namespace / ElasticQuota 落地（→ [cluster-manager.md](../core/cluster-manager.md) / [tenant-operator.md](../core/tenant-operator.md)）|
| 配额（[Tab 2](#tab-2-配额) / [§7.2](#72-配额)） | 系统 / 租户管理员对 `Tenant.spec.quotas[]` 的可视化 CRUD | ElasticQuota CR 命名 / 调度行为；分层配额（→ [tenant-operator.md §4.6.2](../core/tenant-operator.md#462-elasticquota)）|
| 成员（[Tab 3](#tab-3-成员) / [§7.3](#73-成员)） | 本租户内的用户↔角色绑定 UI 与 REST 入口 | 用户身份来源、内置角色定义、跨租户 RBAC 矩阵（→ `auth.md`）|

**关键不变式：**

> **Platform 不为租户建任何视图表**。租户实体的权威完全在 [cluster-manager](../core/cluster-manager.md) 的 PG `tenants` 表——cluster-manager 自身是 PG-first 业务服务，Tenant CR 是它通过 outbox + reconciler 渲染的下游派生产物。所有租户字段（`displayName` / `description` / `business_unit` / `quotas` / 状态等）都是 cluster-manager API 的一级字段，Platform 仅做透传与展示。
>
> Platform 自身只持有 `user_tenant_roles` 关联表（成员绑定），其 `tenant_name` 列直接引用 cluster-manager `tenants.name`（partial unique on `WHERE deleted_at IS NULL`，等价于稳定 FK）。
>
> 所有租户字段都实时穿透到 cluster-manager；Platform 端不需要"权威先行 + 本地补偿"那套双写策略；展示元数据也不需要 annotation 折叠 / 展开的 DTO 映射。
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
- 资源池 / 资源单元的定义与维护：见 `resource-pool.md`。
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

Platform **不为租户实体建表**。所有租户字段都由 cluster-manager 实时返回（cluster-manager 自身的 PG schema 详见 [cluster-manager.md §3](../core/cluster-manager.md#3-pg-schema)）；Platform 只持有 `user_tenant_roles` 关联用于成员绑定。

### 3.1 租户元数据的归属

cluster-manager API 一级字段与 Platform 视图字段直接 1:1 对应——**没有任何 annotation 折叠 / 展开逻辑**：

| Platform 字段 | cluster-manager API 字段 | 备注 |
| --- | --- | --- |
| `name`（URL 锚点） | `name` | DNS-1123、≤40 字符；cluster-manager 校验；创建后不可变 |
| `displayName` | `displayName` | 主展示名，允许 Unicode（含中文） |
| `description` | `description` | 自由文本，允许 Unicode；cluster-manager 一级 PG 列，支持 ≤1000 字符 |
| `businessUnit` | `businessUnit` | 业务线 / 部门归属；cluster-manager 一级 PG 列 + 索引，支持列表过滤 |
| `annotations` | `annotations` | 透传给运维使用的扩展位 |
| `namespace` / `quotas` / `initResources` / `suspended` | 同名字段 | 透传 |
| `phase` / `namespaceReady` / `conditions` / `quotas[].used` | 同名字段 | cluster-manager informer 回流到 PG 后由 GET 返回 |
| `createdAt` / `updatedAt` | 同名字段 | cluster-manager `tenants` 表自动维护 |
| `deletedAt` | `deletedAt` | 软删时间戳；`null` = 活跃 |

所有 GET 请求都调 `clustermanager.GetTenant(name)` 取最新视图；Platform 端无任何缓存。cluster-manager 内部走 PG 查询 + informer 回流（详见 [cluster-manager.md §4.3](../core/cluster-manager.md#43-informer-回流)），单租户 GET 延迟充分可控。

> 历史 / 软删租户：Platform 通过 `?includeArchived=true` 列出，通过 `POST /api/v1/tenants/{name}/restore` 恢复。这两个能力由 cluster-manager 原生提供（[cluster-manager.md §6.2](../core/cluster-manager.md#62-端点)）；Platform 仅做权限校验后透传。

### 3.2 `user_tenant_roles` 关联

成员管理消费 `user_tenant_roles(user_id, tenant_name, role_id)` 三元组：

- `tenant_name` 直接引用 Tenant CR `metadata.name`，text 类型，无 RDBMS 外键（Tenant CR 不在 PG 中）。
- 因 Tenant CR `metadata.name` 创建后不可变，等价于一个稳定 FK；rename 抗性损失为零。
- 删除租户时由 [§8.2](#82-一致性策略与级联) 同步级联清理该租户在 `user_tenant_roles` 中的所有行。

字段、PK、索引细节由 `auth.md` 给出。

---

## Part II — UI 设计

## 4. 菜单与列表页

菜单位置：「系统管理 → 租户管理」。

### 4.1 列表页

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
- 列表渲染：Platform 后端直接透传 cluster-manager `LIST tenants` 结果，按 RBAC 裁剪即可；无本地 join，状态字段也无 PG 缓存。cluster-manager 内部走 PG 查询 + informer 回流。
- 列表可见性：
  - `system-admin`：全集群所有 Tenant CR。
  - 其他角色：先按 `user_tenant_roles.user_id = current_user` 取 `tenant_name` 集合，再按集合过滤 cluster-manager 的 LIST 结果。

### 4.2 操作按钮

- **详情** — 任何在该租户上有绑定的角色可点击。
- **暂停 / 恢复** — 仅 `system-admin`；分别调 `POST /api/v1/tenants/{name}/suspend`、`/unsuspend`。
- **删除** — 仅 `system-admin`；调 `DELETE /api/v1/tenants/{name}`，需二次确认弹窗，并展示「同租户成员数 / 当前 phase / 是否 Suspended」。

## 5. 创建租户表单（system-admin only）

| UI 字段 | cluster-manager API 字段 | 说明 |
| --- | --- | --- |
| `name` | `name` | DNS-1123、≤40 字符；DNS 校验由 cluster-manager 兜底；URL 中的 `{name}` 即此列 |
| `displayName` | `displayName` | 主展示名，允许中文 |
| `description` | `description` | 自由文本说明，允许中文；cluster-manager 一级 PG 列 |
| `businessUnit` | `businessUnit` | 业务线 / 部门归属，cluster-manager 一级 PG 列，支持列表过滤 |
| `namespace.name` | `namespace.name` | 创建后不可变（详见 [tenant-operator.md §4.3.3](../core/tenant-operator.md#433-字段归属与不可变性)） |
| `namespace.labels` / `annotations` | `namespace.{labels, annotations}` | 仅在 Namespace 首次创建时落地 |
| `quotas[]` | `quotas[]` 透传 | 每条 `(pool, name, min, max)`；`(pool, name)` 创建后不可变 |
| `initResources` | `initResources` 透传 | 列出 `name` + `sourceXxxRef`；UI 不构造源 Secret 数据，由集群管理员预置 |

校验：

- 必填项 + 长度提示：UI 即时校验。
- 跨字段约束（`min ≤ max`、`(pool, name)` 唯一）：UI 即时校验作为快速反馈；最终以 cluster-manager 返回为准。
- DNS-1123 / namespace denylist：UI 不前置查询，依赖 cluster-manager 4xx 反馈，把 problem `detail` 透传到表单错误位。

## 6. 详情页 Tab

详情页以 `name` 为维度，分为四个 Tab：基本信息、配额、成员、审计日志。

### Tab 1 基本信息

展示：

- `displayName` / `description` / `business_unit`（均为 cluster-manager API 一级字段，可编辑）。
- 状态卡片：`phase` / `namespaceReady` / `conditions[]`（cluster-manager 由 informer 回流，只读）。
- 命名空间：`namespace.name`（只读）。

操作：

- **编辑展示元数据**（`system-admin`）：单次 PATCH 调 cluster-manager 更新对应一级字段；Platform 无本地表无需双写，cluster-manager 内部由 PG → outbox → CR 异步同步。
- **暂停 / 恢复**（`system-admin`）：调对应 endpoint。
- **删除**（`system-admin`）：与列表页同步逻辑；软删后可通过「已归档租户」入口恢复（详见 [§11 后续迭代](#11-后续迭代)）。

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

处理顺序：

1. Platform DTO → cluster-manager DTO 映射：全部一级字段 1:1 透传，无 annotation 折叠逻辑。
2. 调 `clustermanager.Client.CreateTenant`：失败 → 4xx/5xx 直接返给前端。
3. 成功 → 直接把 cluster-manager 返回的 DTO 透传给前端（仅做命名风格转换，详见 [§8.3](#83-dto-映射器)）。

整个过程 Platform 端无任何 PG 写入，自然没有"上游成功 / 本地失败"的补偿问题。

#### `GET /api/v1/tenants`（已登录即可，按角色裁剪）

- `system-admin`：直接透传 cluster-manager `LIST tenants`，映射回 Platform DTO。
- 其余角色：先按 `user_tenant_roles.user_id = current_user` 取 `tenant_name` 集合，再按集合过滤 cluster-manager 的 LIST 结果。
- 支持 query：`status` / `business_unit` / `q`（关键字）/ `limit` / `continue` / `includeArchived` / `sortBy`，全部下推到 cluster-manager（cluster-manager 内部走 PG 索引，[cluster-manager.md §6.2](../core/cluster-manager.md#62-端点)），Platform 不做内存过滤。

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
  "updatedAt": "...",
  "deletedAt": null
}
```

整个 DTO 是 cluster-manager 返回 DTO 的命名风格转换（`namespace.name` 扁平化为 `namespace` 字段等小幅调整）；`displayName` / `description` / `businessUnit` 等都已经是 cluster-manager 的一级字段，不需要 annotation 解码。

`?includeArchived=true` 时同时返回软删租户，`deletedAt` 非 null。

#### `PATCH /api/v1/tenants/{name}`（`system-admin`）

单点写：

- 请求体里的 `displayName` / `description` / `businessUnit` / `annotations` / `namespace.labels` / `namespace.annotations` / `initResources` 直接 1:1 透传给 cluster-manager PATCH；
- 提前拦截不可变字段：`name` / `namespace.name` / `quotas[].(pool, name)`，违反返回 `400 immutable-field`，不触达 cluster-manager。

#### `POST /api/v1/tenants/{name}/suspend` / `unsuspend`（`system-admin`）

直接透传 cluster-manager 同名 endpoint。

#### `DELETE /api/v1/tenants/{name}`（`system-admin`）

顺序：

1. 校验 `user_tenant_roles WHERE tenant_name = :name` 行数；非空 → `409 tenant-has-members`，UI 引导先在「成员」Tab 清空。
2. 调 `clustermanager.Client.DeleteTenant`（**软删**：cluster-manager 写 `deleted_at = now()`，reconciler 异步删除 CR）：失败 → 4xx/5xx 透传。
3. 成功后由后台 / 同事务 worker 级联清理 `user_tenant_roles` 中 `tenant_name = :name` 的残留行（理论上经 §1 校验已为空，作为兜底）。详见 [§8.2](#82-一致性策略与级联)。

cluster-manager 保留 `deleted_at IS NOT NULL` 的行 retention 期内（默认 365 天），可通过下面的 `restore` 端点恢复。

#### `POST /api/v1/tenants/{name}/restore`（`system-admin`）

恢复 cluster-manager 中软删的租户：

1. 校验当前没有同名活跃租户（cluster-manager 自身的 partial unique 兜底，409 时透传）。
2. 调 `clustermanager.Client.RestoreTenant`，cluster-manager 清空 `deleted_at` 并重新触发 reconciler 创建 CR。
3. 由于 [§7.1 DELETE](#delete-apiv1tenantsnamesystem-admin) 已级联清掉 `user_tenant_roles`，恢复后该租户初始无成员，需 `system-admin` 重新分配。

UI 入口：[§11 后续迭代](#11-后续迭代) 中的「已归档租户」管理界面。

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
| `service.go` | 业务编排：调 cluster-manager、suspend / unsuspend 透传、不可变字段拦截、成员增删校验 |
| `repository.go` | GORM 操作 `user_tenant_roles`（仅 join 与级联清理用） |
| `dto.go` | Platform 请求 / 响应类型；与 cluster-manager API DTO 的显式映射 |
| `mapping.go` | DTO 映射器：Platform DTO ⇄ cluster-manager DTO，承担 `description` / `businessUnit` ⇄ `spec.annotations` 的折叠 / 展开（详见 [§8.3](#83-dto-映射器)） |

风格沿用 [platform/overview.md §7.1](overview.md#71-仓库与目录布局) 与 [components/compute/](../../../components/compute/) 的 handler/service/repository 三层。无 `view.go`：单一来源直接由 mapping 产出，无需融合。

### 8.1 RBAC 中间件接入

`internal/auth` 提供 `RequireSystemAdmin` / `RequireTenantRole(role, tenantParam)` 中间件，定义见 `auth.md`。本文档使用：

| 路由 | 中间件链 |
| --- | --- |
| `POST/PATCH/DELETE /api/v1/tenants[...]`、`POST /api/v1/tenants/{name}/suspend`、`/unsuspend`、`/restore` | `RequireSystemAdmin` |
| `POST/PATCH/DELETE /api/v1/tenants/{name}/quotas[...]` | `RequireTenantRole("tenant-admin", "name")`，`system-admin` 短路 |
| `POST/PATCH/DELETE /api/v1/tenants/{name}/members[...]` | `RequireTenantRole("tenant-admin", "name")`，`system-admin` 短路 |
| `GET /api/v1/tenants/{name}` / `GET .../quotas` | `RequireTenantRole("user", "name")`，`system-admin` 短路 |
| `GET /api/v1/tenants/{name}/members` | `RequireTenantRole("tenant-admin", "name")`，`system-admin` 短路 |
| `GET /api/v1/tenants` | 已登录即可；handler 内部按角色裁剪可见集合 |

### 8.2 一致性策略与级联

Tenant 实体的权威完全在 cluster-manager 持有的 Tenant CR；Platform 不缓存任何 Tenant 字段，因此**没有"上游写成功 / 本地写失败"这条失败路径**。原设计中的 `tenants` 本地表 + inline retry + outbox 补偿队列全部不再需要。

**创建 / 更新**：单点写——映射到 cluster-manager DTO 后单次调用，失败 → 4xx/5xx 透传，无补偿路径。

**删除**：

1. 先校验 `user_tenant_roles WHERE tenant_name = :name`；非空 → `409 tenant-has-members`，引导先清空成员。
2. 调 `clustermanager.Client.DeleteTenant`：失败 → 透传。
3. 成功 → 同事务级联清理 `user_tenant_roles` 中 `tenant_name = :name` 的残留行（经步骤 1 校验后理论为空，作为兜底）。
4. **孤儿处理**：若 cluster-manager 已返 `404` 但 `user_tenant_roles` 仍有该 tenant 的行，下次 LIST 时按 cluster-manager 结果反向归并（认为已删除），handler 同步清理。

**不可变字段**：`name` / `namespace.name` / `quotas[].(pool, name)`，Platform 在写之前强校验，违反返回 `400 immutable-field`，从不触达 cluster-manager。

### 8.3 DTO 映射器

`mapping.go` 是 Platform ⇄ cluster-manager DTO 转换的唯一来源。**所有字段都是 1:1 透传或命名风格转换**——cluster-manager 已经把 `description` / `business_unit` 升级为 PG 一级字段，Platform 不再需要做 annotation 折叠 / 展开。

主要的命名风格 / 结构调整（其余字段同名同义直传）：

| Platform DTO | cluster-manager DTO |
| --- | --- |
| `namespace` (string) | `namespace.name` |
| `business_unit` (snake_case，沿用既有 OpenAPI 风格) | `businessUnit` (camelCase) |

未识别的 `annotations` 键由 cluster-manager 原样持久化（`tenants.annotations` jsonb）+ 同步到 Tenant CR `spec.annotations`；前端透传 map 显示，不丢失。

### 8.4 状态读取

- 任何展示 Tenant CR 字段（`phase` / `namespaceReady` / `quotas[].used` / `conditions`）的请求都调 `clustermanager.Client.GetTenant(name)`，不本地缓存。
- 不引入 K8s informer：tenant 操作天然低频；cluster-manager 端走 controller-runtime cache，单租户 GET 延迟充分可控。

### 8.5 PG schema

- `user_tenant_roles`：定义见 `auth.md`；本模块只消费 `(user_id, tenant_name, role_id)` 三元组，并在 [§8.2](#82-一致性策略与级联) 中级联清理。
- **不引入** `tenants` 表。

表在 `migrate` 子命令中自动迁移（详见 [overview.md §7.2](overview.md#72-启动子命令)）。

### 8.6 度量与日志

Prometheus 指标（特有于本功能；通用上游调用指标见 [overview.md §7.5](overview.md#75-下游-typed-client)）：

- `platform_tenant_action_total{action, status}`：`action ∈ {create, update_meta, suspend, unsuspend, delete, quota_create, quota_update, quota_delete, member_add, member_update, member_remove}`，`status ∈ {success, failure}`。
- `platform_tenant_orphan_role_cleanup_total{reason}`：counter，记录 [§8.2](#82-一致性策略与级联) 中孤儿 `user_tenant_roles` 行的级联清理次数；`reason ∈ {delete_cascade, list_reconcile}`。

zap 字段约定：每条租户操作日志必带 `tenant_name` / `actor_user` / `action` / `status`；删除 / suspend / 成员变更额外带 `target_user` / `role_name`（如适用）。

---

## Part V — 实施与验证

## 9. 实现路径

### 9.1 阶段一（MVP）

| 模块 | 范围 | 完成信号 |
| --- | --- | --- |
| handler / service / dto / mapping | [§7.1](#71-租户-crud) / [§7.2](#72-配额) / [§7.3](#73-成员) 全部 endpoint | `make platform-build` 通过 |
| RBAC 装配 | [§8.1](#81-rbac-中间件接入) 路由表全部接通；`system-admin` 短路逻辑生效 | 单元测试覆盖中间件分支 |
| PG 迁移 | `user_tenant_roles` 由 `auth.md` 提供；本模块无新增表 | `make platform-migrate` 干净 |
| Integration | testcontainers PG + cluster-manager fake，覆盖 happy path（创建 → 加成员 → 加配额 → suspend → 删除）+ DTO 映射器双向往返 | `make platform-integration` 通过 |

### 9.2 阶段二

1. 配额 Tab 表头「合计 max」聚合视图。
2. 列表页「按 phase 分组」聚合视图。

### 9.3 阶段三 / TBD

详见 [§11 后续迭代](#11-后续迭代)。

## 10. 关键设计决策

| 决策项 | 决策 | 理由 |
| --- | --- | --- |
| Platform PG 表 | **不为租户实体建表**；权威完全在 cluster-manager PG（cluster-manager 是 PG-first 业务服务） | 消除 Platform ↔ cluster-manager 双写漂移；同时获得 cluster-manager 的软删、`restore`、富查询能力 |
| 成员关联 FK | `user_tenant_roles.tenant_name` 直接引用 cluster-manager `tenants.name` | `tenants.name` 在活跃维度 partial unique 且不可变，等价于一个稳定 FK；无需 Platform 自建 uuid |
| DTO 映射 | 一级字段 1:1 透传；无 annotation 折叠 / 展开逻辑 | cluster-manager 已把 description / business_unit 升级为一级字段，Platform mapping.go 退化为命名风格转换 |
| 软删恢复 | 由 cluster-manager 原生 `restore` 端点提供；Platform 仅做权限校验后透传 | 「保留 tenant 记录」由 cluster-manager 的 `deleted_at` + retention 落地，Platform 无需引入归档表 |
| 命名空间归属 | `compute_namespace` / `artifacts_namespace` 不属于租户实体 | 与下游 namespace 分区模型一致；Platform 不维护重复映射 |
| API 路径形态 | 统一在 `/api/v1/tenants/...`，权限走中间件 | 路径按资源组织；权限差异完全由 RBAC 中间件按角色 + 资源所有权判定 |
| 详情页 Tab 划分 | 基本信息 / 配额 / 成员 / 审计日志 | 减少首屏字段密度；每个 Tab 对应独立 REST 资源 |
| 配额拆分总额校验 | Platform 不做；仅作转发 | ElasticQuota 当前为扁平结构，不天然支持父子关系；维持单一职责 |
| 删除级联 | cluster-manager 删除成功后再清理 `user_tenant_roles` 中残留行 | cluster-manager 是权威；level-1 校验已要求成员清空，级联清理为兜底 |
| 不可变字段拦截 | `name` / `namespace.name` / `quotas[].(pool, name)` | 在 Platform 层拦截，减少 cluster-manager 4xx 与前端误用 |
| 成员保护 | 不能移除自己最后一个 `tenant-admin` 角色 | 防止租户失管 |
| 成员角色集合 | 添加 / 修改成员仅允许 `tenant-admin` / `user` | `system-admin` 是平台级角色，不在租户菜单内绑定 |

## 11. 后续迭代

- **审计日志 Tab**：复用 [overview.md §7.4](overview.md#74-pg-schema) 的 `audit_logs` 表；落地按 `target=tenant:{name}` 过滤的索引、UI 表格、保留期可配置项。
- **「已归档租户」管理界面**：UI 入口（系统管理菜单下二级），调 `GET /api/v1/tenants?includeArchived=true&deletedAt_not_null=true` 列出，每行操作 `POST .../restore`。底层能力已由 cluster-manager 提供，缺前端表格 + 权限收口。
- **配额硬校验 / 分层配额**：等上游 ElasticQuota 提供 `parent` 字段后，Platform 可在 Tab 2 引入「上限 cap」视图与硬阻断。
- **init_resources 表单深度**：当前仅暴露 `sourceXxxRef`；后续可接入 Vault / Sealed Secrets 等加密源直接创建。
- **租户克隆**：基于已有租户的 `quotas` / `initResources` 模板快速创建新租户。

## 12. 测试策略

- **单元**（`internal/tenant/*_test.go`）：
  - DTO 映射器 `mapping.go` 双向往返：`description` / `businessUnit` ↔ `spec.annotations` 折叠 / 展开；未识别 annotation 透传保留。
  - 不可变字段拦截逻辑。
  - 最后一个 `tenant-admin` 校验逻辑。
  - 列表可见性裁剪（不同 RBAC 角色）。
- **integration**（`components/platform/backend/test/integration/`）：
  - testcontainers PostgreSQL；in-process cluster-manager fake（httptest）模拟 Tenant CR 行为，包括成功 / 4xx / 5xx 响应。
  - happy path：创建租户 → 加成员 → 加配额 → suspend → unsuspend → 删除；断言全程仅写 `user_tenant_roles` 一张表。
  - 删除时 `user_tenant_roles` 非空 → `409`；非空时绕过校验直接 cluster-manager 删除 → handler 在下次 LIST 时反向归并清理孤儿。
  - RBAC：`system-admin` / `tenant-admin@self` / `user@self` 在每个 endpoint 上的允许 / 拒绝矩阵。
- 不引入 envtest：Platform 本身不直读 K8s API；端到端校验由 [cluster-manager.md §6](../core/cluster-manager.md#6-测试) / [tenant-operator.md §6](../core/tenant-operator.md#6-测试) 在自身集成层覆盖。

## 13. 相关引用

- [PRD §6.4.1 租户管理](../../product/prd.md#641-租户管理)
- [docs/system_design/overview.md](../overview.md)
- [docs/system_design/platform/overview.md](overview.md)
- [docs/system_design/core/cluster-manager.md](../core/cluster-manager.md)
- [docs/system_design/core/tenant-operator.md](../core/tenant-operator.md)
- `auth.md`
