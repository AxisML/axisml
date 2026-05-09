# AxisML Platform 资源池管理 详细设计

本文档是 AxisML Platform 子系统下 **「系统管理 → 资源池管理」** 一级功能的全栈设计，承接 [PRD §6.4.2 资源池管理](../../product/prd.md#642-资源池管理) 与系统层 [compute](../core/compute.md) 之间的 Platform 入口：资源池列表 / 详情、资源池下属资源单元的 CRUD、相关 UI 与 REST 入口。

资源池（ResourcePool）与资源单元（ResourceUnit）合并在一份文档：PRD §5 一级菜单只有「资源池管理」，§6.4.2 把「维护池下的资源单元规格档位」作为该入口的下属能力；compute 也以子路径 `/api/v1/resource-pools/{pool}/resource-units` 表达父子关系——独立成文反而割裂同一个用户故事。

| 模块 | 职责 | 边界外 |
| --- | --- | --- |
| 资源池视图与生命周期（[§4](#4-菜单与列表页) / [§7.1](#71-资源池-crud)） | 系统管理员 CRUD 资源池；所有已登录用户可读 | ResourcePool PG 字段语义、注入规则（→ [compute.md §4](../core/compute.md#4-resourcepool)）；Node label / taint 维护（→ kubectl 手工） |
| 资源单元（详情页 [Tab 2](#tab-2-资源单元完整-crud) / [§7.2](#72-资源单元-crud)） | 系统管理员在池上下文中 CRUD 资源单元；所有已登录用户可读 | ResourceUnit 字段语义、命名约定、注入合并规则（→ [compute.md §5](../core/compute.md#5-resourceunit)）|

**关键不变式：**

> Platform 自有 PG **不引入** 资源池 / 资源单元相关表；所有列表 / 详情 / 写操作都实时穿透 compute（`internal/client/compute` typed client）。
>
> 资源池与资源单元都是全集群可见对象（[compute.md §4.1](../core/compute.md#41-概述) / [§5.1](../core/compute.md#51-概述)），Platform 不做按租户过滤；按租户的池可见性作为后续迭代项。
>
> 集群侧 Node label / taint 由管理员通过 kubectl 维护，Platform UI 不下发；资源池仅持有匹配条件。

**文档组织：**

- **Part I — 服务边界**（[§1](#1-概述与定位) – [§3](#3-数据模型platform-自有部分)）：定位、可见性矩阵、Platform 自有数据模型。
- **Part II — UI 设计**（[§4](#4-菜单与列表页) – [§6](#6-详情页-tab)）：菜单 / 列表页 / 创建表单 / 详情页 Tab。
- **Part III — 后端 API 契约**（[§7](#7-rest-路径与响应格式)）：REST 路径、字段、RBAC。
- **Part IV — 后端实现**（[§8](#8-模块结构)）：模块结构、RBAC 装配、代理策略、可观测性。
- **Part V — 实施与验证**（[§9](#9-实现路径) – [§13](#13-相关引用)）：阶段化实现、关键决策、后续迭代、测试、参考。

---

## Part I — 服务边界

## 1. 概述与定位

「资源池管理」是系统管理菜单下面向系统管理员的入口，覆盖以下能力：

- 系统管理员：创建 / 编辑 / 删除资源池；维护池下的资源单元规格档位；上线后即可暴露给租户使用。
- 已登录的租户管理员 / 普通用户：在 Job / Service / Workspace 提交表单中只读消费资源池与资源单元下拉。

不在范围内的能力：

- ResourcePool / ResourceUnit 字段语义、注入合并规则、PG schema：见 [compute.md §4](../core/compute.md#4-resourcepool) / [§5](../core/compute.md#5-resourceunit)。
- Node label / taint 的下发与维护：管理员通过 `kubectl label` / `kubectl taint` 手工执行，平台不感知。
- 池间调度借用、池容量聚合、按租户的池可见性：见 [§11 后续迭代](#11-后续迭代)。
- 用户登录、JWT、内置角色矩阵：见 `auth.md`。

## 2. 角色与可见性矩阵

下表只列与本功能相关的能力；persona ↔ RBAC 角色映射沿用 [platform/overview.md §2.2](overview.md#22-用户角色persona)。

| 能力 | `system-admin` | `tenant-admin` | `user` |
| --- | :---: | :---: | :---: |
| 列出资源池 / 资源单元 | ✅ | ✅ | ✅ |
| 查看资源池 / 资源单元详情 | ✅ | ✅ | ✅ |
| 创建 / 编辑 / 删除资源池 | ✅ | ❌ | ❌ |
| 创建 / 编辑 / 删除资源单元 | ✅ | ❌ | ❌ |

资源池 / 资源单元都是全集群对象，Platform 不做「按租户过滤可见性」（见 [§10 关键设计决策](#10-关键设计决策)）。读权限只要求已登录身份，是因为 Job / Service / Workspace 提交表单需要对所有用户暴露下拉。

## 3. 数据模型（Platform 自有部分）

Platform 自身**不为资源池或资源单元引入任何 PG 表**。所有字段权威以 compute 为准：

- 资源池字段：见 [compute.md §4.2 `resource_pools`](../core/compute.md#42-数据模型)。
- 资源单元字段：见 [compute.md §5.2 `resource_units`](../core/compute.md#52-数据模型)。

操作审计写入 Platform 通用 `audit_logs` 表（[overview.md §7.4](overview.md#74-pg-schema)），按 `target` 字段区分 `resource-pool/<name>` 与 `resource-unit/<pool>/<name>`。

---

## Part II — UI 设计

## 4. 菜单与列表页

菜单位置：「系统管理 → 资源池管理」。

### 4.1 列表页

| 列 | 来源 | 说明 |
| --- | --- | --- |
| `name` | compute `resource_pools.name` | 主展示列；URL 锚点 |
| `description` | compute `resource_pools.description` | 自由文本 |
| 节点选择器摘要 | compute `resource_pools.node_selector` | 取前两个 key 拼成 `k1=v1, k2=v2, …` |
| 资源单元数 | 聚合 `GET /resource-units?limit=0` | 见 [§4.3 数量聚合](#43-资源单元数量聚合) |
| 创建时间 | compute `resource_pools.created_at` | |
| 操作 | — | 详情 / 删除 |

- 过滤：关键字（`name` / `description` 模糊匹配）、`node_selector` key 命中。
- 列表渲染：Platform 后端把 compute `LIST resource-pools` 结果直接透传，并行聚合每个 pool 的资源单元数后写入 `resource_unit_count` 字段。
- 列表可见性：所有已登录用户。

### 4.2 操作按钮

- **详情** — 任何已登录用户可点击。
- **删除** — 仅 `system-admin`；调 `DELETE /api/v1/resource-pools/{pool}`，需二次确认弹窗。前端在确认弹窗中展示 Platform 后端返回的「池下资源单元数 / 引用此池的活跃 Job / Service 数」摘要（来自 [§7.1 删除前置校验](#71-资源池-crud)）。

### 4.3 资源单元数量聚合

资源池数量预期 < 50（每个机型 / 用途各一池），首版策略：

1. Platform 后端 `pool_service.List` 收到 compute 返回的 pool 列表后，按池并发触发 `compute.ListResourceUnits(pool, limit=0)` 拿 count；
2. 拼装到响应 DTO 的 `resource_unit_count` 字段；
3. 单池失败不阻塞整体响应：失败池 `resource_unit_count = -1`，前端显示 `—`。

后续若池规模膨胀或调用频次增加，再引入 5–10 秒级 in-process LRU 缓存（见 [§11](#11-后续迭代)）。

## 5. 创建资源池表单（system-admin only）

| UI 字段 | 写入位置 | 说明 |
| --- | --- | --- |
| `name` | `resource_pools.name` | DNS-1123、≤40 字符；校验由 compute 兜底（详见 [compute.md §3.2](../core/compute.md#32-pg-编排约定)） |
| `description` | 同名列 | 自由文本，用于列表与详情展示 |
| `node_selector` | 同名列 | key/value 列表 UI；可加多行；空对象表示「整集群可用」 |
| `tolerations` | 同名列 | JSON 编辑器 + 常见模板（`nvidia.com/gpu`、`axisml.io/dedicated` 等） |
| `metadata` | 同名列 | 可选 jsonb，留作扩展（如 `cost_per_hour` 后续迭代） |

辅助 UI：

- **节点匹配预览**：表单边写边给「将匹配以下节点标签」摘要预览，仅做字符串拼装，不查 K8s。
- **管理提示**：「集群侧需提前给目标节点打 label / taint，Platform 不下发；可在 kubectl 中执行 `kubectl label node <name> <key>=<value>`」。

校验：

- 必填项 + 长度提示：UI 即时校验。
- DNS-1123 / `name` 唯一：UI 不前置查询，依赖 compute 4xx 反馈，把 problem `detail` 透传到表单错误位。

## 6. 详情页 Tab

详情页以 `name` 为维度，分为四个 Tab：基本信息、资源单元、节点匹配预览（占位）、审计日志（占位）。

### Tab 1 基本信息

展示：

- `name` / `description` / `node_selector` / `tolerations` / `metadata`（来自 compute）。
- 创建时间 / 最近更新时间（`created_at` / `updated_at`）。

操作（`system-admin`）：

- **编辑**：允许改 `description` / `node_selector` / `tolerations` / `metadata`；`name` 创建后不可变（compute 侧约束，UI 置灰）。
- **删除**：与列表页同步逻辑；前置阻断详见 [§7.1](#71-资源池-crud)。

### Tab 2 资源单元（完整 CRUD）

按池上下文管理资源单元，是 ResourceUnit 在 Platform UI 中的唯一入口。

```
[+ 新建资源单元]
┌──────────────────┬──────────────────────────┬──────────────────────┬────────────────────────┬─────┐
│ 名称              │ requests                 │ limits               │ node_selector          │ 操作 │
├──────────────────┼──────────────────────────┼──────────────────────┼────────────────────────┼─────┤
│ a100-1x-large    │ cpu=8, mem=64Gi, gpu=1   │ cpu=8, mem=64Gi      │ nvidia.com/gpu.product │ ✏️ ❌│
│                  │                          │                      │ =A100-SXM4-80GB        │     │
│ cpu-medium       │ cpu=4, mem=16Gi          │ cpu=4, mem=16Gi      │ —                      │ ✏️ ❌│
└──────────────────┴──────────────────────────┴──────────────────────┴────────────────────────┴─────┘
```

数据来源：`GET /api/v1/resource-pools/{pool}/resource-units` 实时穿透 compute。

#### 创建资源单元抽屉表单

| 字段 | 写入位置 | 说明 |
| --- | --- | --- |
| `name` | `resource_units.name` | 遵循 [compute.md §5.3](../core/compute.md#53-命名约定) 命名约定 `<accelerator>[-<count>x]-<tier>[-<variant>]`；UI 旁附常见模板按钮（`a100-1x-large` / `h100-8x-xlarge-ib` / `cpu-medium` 等）一键填充 |
| `description` | 同名列 | 自由文本 |
| `requests` | 同名列 | 表格 UI：CPU / Memory / GPU 三个常用字段独立行 + 「自定义资源」追加按钮 |
| `limits` | 同名列 | 同 `requests`；默认与 `requests` 相同 |
| `node_selector` | 同名列 | key/value 列表 UI |

提交后调 `POST /api/v1/resource-pools/{pool}/resource-units`。

#### 行内编辑

允许改 `description` / `requests` / `limits` / `node_selector`；`name` 不可变（UI 置灰）；`pool` 不可变（无 UI 入口）。提交调 `PATCH /api/v1/resource-pools/{pool}/resource-units/{unit}`。

#### 删除

行内删除按钮，二次确认。Platform 后端在 DELETE 前置校验中返回「使用此 unit 的活跃 Job / Service 数量」，UI 在确认弹窗中展示该摘要（详见 [§7.2](#72-资源单元-crud)）。

#### 合并节点选择器预览

资源单元 row 展开后展示 **`pool.node_selector` ⊕ `unit.node_selector`**（Pool 优先，详见 [compute.md §5.4](../core/compute.md#54-注入规则)）合并后的最终 selector，帮助管理员在创建任务前校验意图。本预览只是字符串合并，不查 K8s Node。

### Tab 3 节点匹配预览

入口保留，详情见 [§11 后续迭代](#11-后续迭代)。需要 Platform 直读 K8s Node 才能落地，与系统层 ROI 共评后再做。

### Tab 4 审计日志

入口保留，与 [tenant.md Tab 4](tenant.md#tab-4-审计日志) 同步迭代。

---

## Part III — 后端 API 契约

## 7. REST 路径与响应格式

- 路径前缀 `/api/v1/resource-pools/...`，与 [compute.md §4.4](../core/compute.md#44-api-端点) / [§5.5](../core/compute.md#55-api-端点) 同形，便于 Platform typed client 与 compute API 1:1 映射。
- 错误格式：RFC 7807 problem+json，复用 [overview.md §7.3](overview.md#73-错误处理) 的样例。
- 出站调用：`internal/client/compute` typed client，定义见 [overview.md §7.5](overview.md#75-下游-typed-client)。
- 每个 endpoint 在下文括号内标注允许的角色：`system-admin` 表示全局；`authenticated` 表示任意已登录用户。

### 7.1 资源池 CRUD

#### `POST /api/v1/resource-pools`（`system-admin`）

请求体：

```json
{
  "name": "gpu-a100",
  "description": "A100 训练池",
  "node_selector": { "axisml.io/pool": "gpu-a100" },
  "tolerations": [
    { "key": "nvidia.com/gpu", "operator": "Exists", "effect": "NoSchedule" }
  ],
  "metadata": {}
}
```

直接透传 `compute.Client.CreateResourcePool`；compute 4xx（DNS-1123 / 重名）由 typed client 包装为 problem 透传。响应为 `ResourcePool` DTO（结构与下方 GET 响应一致）。

#### `GET /api/v1/resource-pools`（`authenticated`）

- 默认列出所有池（无租户过滤）。
- 响应每条记录追加 `resource_unit_count`（详见 [§4.3](#43-资源单元数量聚合)）。
- 支持 query：`q`（关键字）/ `limit` / `continue`。

#### `GET /api/v1/resource-pools/{pool}`（`authenticated`）

返回 `ResourcePool` DTO：

```json
{
  "name": "gpu-a100",
  "description": "A100 训练池",
  "node_selector": { "axisml.io/pool": "gpu-a100" },
  "tolerations": [...],
  "metadata": {},
  "resource_unit_count": 3,
  "created_at": "...",
  "updated_at": "..."
}
```

#### `PATCH /api/v1/resource-pools/{pool}`（`system-admin`）

允许字段：`description` / `node_selector` / `tolerations` / `metadata`。`name` 拦截：违反返回 `400 immutable-field`，从不触达 compute。其他字段交由 compute 兜底。

#### `DELETE /api/v1/resource-pools/{pool}`（`system-admin`）

**前置校验**（Platform 自做，不依赖 compute 级联）：

1. `compute.ListResourceUnits(pool, limit=0)`：池下资源单元数 > 0 → `409 pool-in-use`，problem detail 形如：

   ```json
   {
     "type": "https://axisml.io/errors/pool-in-use",
     "title": "Resource pool in use",
     "status": 409,
     "detail": "pool gpu-a100 still has 3 resource unit(s)",
     "instance": "/api/v1/resource-pools/gpu-a100",
     "blockers": { "resource_units": 3 }
   }
   ```

2. `compute.ListJobs(filter={pool, active})` + `compute.ListServices(filter={pool, active})`：活跃（非终态）数 > 0 → `409 pool-in-use`，`blockers.active_jobs` / `blockers.active_services` 各列若干示例 name。
3. 通过 → 调 `compute.Client.DeleteResourcePool`。

> **依赖**：上述 LIST API 需要支持按 `pool_id` / `active` 过滤；如 compute 当前未提供，作为本设计落地前的 compute 侧 API 补丁项跟踪。

### 7.2 资源单元 CRUD

| Endpoint | 权限 | 说明 |
| --- | --- | --- |
| `GET /api/v1/resource-pools/{pool}/resource-units` | `authenticated` | 列出指定池下资源单元 |
| `POST /api/v1/resource-pools/{pool}/resource-units` | `system-admin` | 创建资源单元，body `{name, description?, requests, limits, node_selector?}` |
| `GET /api/v1/resource-pools/{pool}/resource-units/{unit}` | `authenticated` | 详情 |
| `PATCH /api/v1/resource-pools/{pool}/resource-units/{unit}` | `system-admin` | 仅允许 `description` / `requests` / `limits` / `node_selector`；`name` / `pool` 不可变，违反返 `400 immutable-field` |
| `DELETE /api/v1/resource-pools/{pool}/resource-units/{unit}` | `system-admin` | 前置校验：使用此 unit 的活跃 Job / Service 数 > 0 → `409 unit-in-use`，结构与 [§7.1 DELETE 一致](#71-资源池-crud) |

命名 / DTO 字段与 [compute.md §5.2](../core/compute.md#52-数据模型) 严格对齐。响应 DTO 透传 compute 字段，不二次裁剪。

### 7.3 错误透传约定

- DNS-1123 / 命名约定 / `(pool, name)` 重复 / 不可变字段：依赖 compute 4xx 透传，Platform 不重复校验。
- 删除前置阻断（pool / unit）：Platform 自身判定，`409` + 自定义 problem `type`，详见 §7.1 / §7.2。
- compute 5xx：包装为 `https://axisml.io/errors/upstream-failure`（详见 [overview.md §7.5](overview.md#75-下游-typed-client)）。

---

## Part IV — 后端实现

## 8. 模块结构

目录：`components/platform/backend/internal/resourcepool/`

| 文件 | 职责 |
| --- | --- |
| `pool_handler.go` | 资源池 Gin 路由（`/api/v1/resource-pools`）、请求解析、RBAC gate 装配、problem 渲染 |
| `pool_service.go` | 资源池业务编排：透传 compute、删除前置校验（聚合资源单元数 / 活跃 Job / Service 数） |
| `unit_handler.go` | 资源单元 Gin 路由（`/api/v1/resource-pools/{pool}/resource-units`） |
| `unit_service.go` | 资源单元业务编排：透传 compute、删除前置校验（活跃 Job / Service 数） |
| `dto.go` | 请求 / 响应类型；与 compute API DTO 之间的显式映射；DTO 增 `resource_unit_count` 字段 |

资源池与资源单元同包，便于共享 typed client 与 problem 映射函数。无 `repository.go`：Platform 不持有任何相关 PG 表。风格沿用 [overview.md §7.1](overview.md#71-仓库与目录布局) 与现有 [tenant 模块](tenant.md#8-模块结构) 的 handler / service 两层。

### 8.1 RBAC 中间件接入

`internal/auth` 提供 `RequireSystemAdmin` / `RequireAuthenticated` 中间件，定义见 `auth.md`。

| 路由 | 中间件链 |
| --- | --- |
| `POST/PATCH/DELETE /api/v1/resource-pools[...]` | `RequireSystemAdmin` |
| `POST/PATCH/DELETE /api/v1/resource-pools/{pool}/resource-units[...]` | `RequireSystemAdmin` |
| `GET /api/v1/resource-pools[...]` | `RequireAuthenticated` |
| `GET /api/v1/resource-pools/{pool}/resource-units[...]` | `RequireAuthenticated` |

### 8.2 一致性与代理策略

资源池与资源单元的权威完全在 compute；Platform 只做转发 + 前置校验。

**写路径**：

- POST / PATCH：业务校验通过后调 compute typed client；compute 4xx / 5xx 透传 problem。
- DELETE：先聚合查询资源单元 / Job / Service 计数；任意阻断条件命中 → 直接返 `409` 与 problem，不触达 compute；通过 → 调 compute DELETE。

**读路径**：

- LIST 资源池：调 compute LIST；Platform 在 handler 内并发聚合 `resource_unit_count`（详见 [§4.3](#43-资源单元数量聚合)）。
- 单池 / 单元 GET：直接透传 compute。

**身份头**：所有出站请求自动注入 `X-Axisml-User`（[overview.md §7.5](overview.md#75-下游-typed-client)）；compute 仅做审计 / ownership 归属，不二次鉴权（[compute.md §3.1](../core/compute.md#31-与-platform-的请求契约)）。

### 8.3 度量与日志

Prometheus 指标（特有于本功能；通用上游调用指标见 [overview.md §7.5](overview.md#75-下游-typed-client)）：

- `platform_resource_pool_action_total{action, status}`：`action ∈ {create, update, delete, delete_blocked}`，`status ∈ {success, failure}`。
- `platform_resource_unit_action_total{action, status}`：同上。
- `platform_resource_pool_unit_count_aggregation_failures_total`：列表页聚合资源单元数量时的失败计数（参见 [§4.3](#43-资源单元数量聚合)）。

zap 字段约定：每条日志必带 `pool_name` / `actor_user` / `action` / `status`；资源单元操作额外带 `unit_name`；删除阻断额外带 `block_reason ∈ {resource-units, active-jobs, active-services}` 与 `child_count`。

---

## Part V — 实施与验证

## 9. 实现路径

### 9.1 阶段一（MVP）

| 模块 | 范围 | 完成信号 |
| --- | --- | --- |
| handler / service / dto | [§7.1](#71-资源池-crud) / [§7.2](#72-资源单元-crud) 全部 endpoint | `make platform-build` 通过 |
| RBAC 装配 | [§8.1](#81-rbac-中间件接入) 路由表全部接通 | 单元测试覆盖中间件分支 |
| 删除前置校验 | 资源池 / 资源单元两个聚合查询 | 单元测试覆盖 happy / 阻断 / 部分失败 |
| Integration | testcontainers PG（仅审计日志）+ httptest compute fake；happy path：建池 → 加 unit → 删 unit 受阻断 → 改 unit → 删除清空 → 删池 | `make platform-integration` 通过 |

### 9.2 阶段二

1. 列表页 ResourceUnit 数量缓存（in-process LRU，5–10 秒级）。
2. 审计日志 Tab 接入（共用 [tenant.md §11](tenant.md#11-后续迭代) 的 outbox + UI 方案）。
3. 节点匹配预览 Tab：Platform 引入 K8s typed client（List Node + 字段过滤），需要与系统层 ROI 共评后落地。

### 9.3 阶段三 / TBD

详见 [§11 后续迭代](#11-后续迭代)。

## 10. 关键设计决策

| 决策项 | 决策 | 理由 |
| --- | --- | --- |
| 文档合并 | 资源池与资源单元同入口、同权限范围、强父子关系，独立成文反而冗余 | 与 PRD §5 / §6.4.2 口径一致；compute 也以子路径表达父子关系 |
| Platform 数据模型 | 不持有任何相关 PG 表，纯代理 compute | compute `resource_pools` / `resource_units` 字段已足够 UI 展示；避免双写漂移 |
| API 路径形态 | 与 compute 同形 `/api/v1/resource-pools[...]` | typed client 1:1 映射；运维心智一致 |
| 删除策略 | Platform 前置阻断（资源池查 unit + 活跃 Job/Service；资源单元查活跃 Job/Service） | compute.md 当前未约束级联；前置阻断给 UI 更清晰的反馈，且避免误删 |
| 池 / 单元可见性 | 全局可见，不按租户过滤 | 与 [compute.md §4.1](../core/compute.md#41-概述) 对齐；按租户白名单作为后续迭代 |
| 节点 label / taint 维护 | 管理员手工 kubectl 运维，UI 不下发 | Platform 不修改 Node 对象（与 [compute.md §4.1](../core/compute.md#41-概述) 决策一致） |
| 资源单元 CRUD 入口 | 收敛到资源池详情页 Tab，不做独立菜单 | 资源单元始终在 (pool) 上下文中操作；与 Tab 内的合并 selector 预览天然耦合 |
| 不可变字段拦截 | `pool.name` / `unit.name` / `unit.pool` | Platform 层拦截，减少 compute 4xx 与前端误用 |

## 11. 后续迭代

- **按租户的池可见性**：Platform 引入「池 → 租户白名单」表；提交 Job / Service 时按当前租户裁剪可见池。
- **节点匹配预览 Tab**：Platform 引入 K8s typed client，按 `node_selector` + `tolerations` 反查命中 Node，配合「容量 / 已分配 / 剩余」三列。
- **池容量聚合**：聚合命中 Node 的 `allocatable` / `requested`，作为配额规划的辅助视图（与 [compute.md §4.5](../core/compute.md#45-后续工作) 对齐）。
- **池间调度借用策略**：当配额不足时是否允许跨池借用；当前默认禁止。
- **资源单元成本元数据**：UI 暴露 `unit.metadata.cost_per_hour`，用于成本核算（与 [compute.md §5.6](../core/compute.md#56-后续工作) 对齐）。
- **资源单元 in-process 缓存**：列表页 `resource_unit_count` 聚合改为 5–10 秒级 LRU。
- **审计日志 Tab**：复用 [overview.md §7.4](overview.md#74-pg-schema) 的 `audit_logs` 表，按 `target` 前缀（`resource-pool/` / `resource-unit/`）过滤。

## 12. 测试策略

- **单元**（`internal/resourcepool/*_test.go`）：
  - service 层删除前置校验逻辑（资源池 / 资源单元），含 compute 部分失败的降级行为。
  - 不可变字段拦截逻辑（`pool.name` / `unit.name` / `unit.pool`）。
  - RBAC 中间件分支（system-admin / authenticated / 未登录）。
  - 列表页 ResourceUnit 数量并发聚合（含部分池失败 → `resource_unit_count = -1`）。
- **integration**（`components/platform/backend/test/integration/`）：
  - testcontainers PostgreSQL（仅审计日志）；in-process compute fake（httptest）模拟 ResourcePool / ResourceUnit / Job / Service LIST 响应。
  - happy path：建池 → 加资源单元 → 删资源单元受阻断（活跃 Job 计数 > 0）→ 移除 Job → 成功删资源单元 → 删池受阻断（仍有 unit）→ 删剩余 unit → 成功删池。
  - 故障注入：compute 4xx（DNS-1123 / 重名 / 不可变字段）透传；compute 5xx 包装为 upstream-failure。
  - RBAC：system-admin / authenticated 在每个 endpoint 上的允许 / 拒绝矩阵。
- 不引入 envtest：Platform 自身不直读 K8s API；端到端校验由 [compute.md §10](../core/compute.md) 在自身集成层覆盖。

## 13. 相关引用

- [PRD §6.4.2 资源池管理](../../product/prd.md#642-资源池管理)
- [docs/system_design/overview.md](../overview.md)
- [docs/system_design/platform/overview.md](overview.md)
- [docs/system_design/platform/tenant.md](tenant.md)
- [docs/system_design/core/compute.md](../core/compute.md)（§4 ResourcePool / §5 ResourceUnit）
- `auth.md`
