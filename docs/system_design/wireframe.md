# AxisML Platform UI 设计

## 1. 概述

本文是 Platform 前端 UI 的设计文档,集中描述页面结构、菜单与导航、列表页字段、详情页 Tab、创建/编辑表单字段、状态展示规则与权限可见性,作为前端开发与产品视觉评审的唯一对齐入口。

- **后端业务编排** (跨服务调用顺序、写入路径、一致性策略、PG schema) 见 [components/platform.md](components/platform.md)。
- **用户认证与角色矩阵** (RBAC 完整定义、JWT 颁发、IdentityProvider 接口) 见 [auth.md](auth.md)。
- **REST API 字段契约** 见 [apis/platform.yaml](apis/platform.yaml)。
- **Dashboard / 服务指标数据来源** 见 [monitoring.md](monitoring.md)。
- **整体系统概念** (Tenant / ResourcePool / Job / Service / Artifact) 见 [overview.md](overview.md)。

> 本期原型(Geist sandbox)仅在「系统管理 → 租户管理 / 资源池管理」交付详细 UI;其余菜单沿用 §3 通用占位骨架,字段契约 / 状态机 / 权限矩阵已在各服务文档落档,UI 详设待 prototype 升级到该菜单时按 §4 / §5 模板补齐。

---

## 2. 整体外壳 (App Shell)

### 2.1 栅格结构

```
┌──────────────────────────────────────────────────────────────────────┐
│ Topbar (56px)                                                        │
│  [A] AxisML │ Breadcrumbs ……………………………… │ env-pill · 📘 · 🔔 · 头像 │
├────────────┬─────────────────────────────────────────────────────────┤
│ Sidebar    │ Main (padding 28×32, max-w 1240)                        │
│ (232px)    │                                                         │
│            │   ┌ Page Head ─────────────────────────────┐            │
│ • Dashboard│   │ // eyebrow                             │            │
│            │   │ Page Title.                 [Actions]  │            │
│ 训练 & 推理│   │ Page Desc                              │            │
│ • 工作区   │   └────────────────────────────────────────┘            │
│ • 计算任务 │                                                         │
│ • 在线服务 │   ┌ Filters (search · select · select · 重置) ────┐     │
│            │   └────────────────────────────────────────────────┘     │
│ 制品中心   │                                                         │
│ • 模型     │   ┌ Card / Section / Table ──────────────────────┐      │
│ • 镜像     │   │                                              │      │
│ • 数据集   │   └──────────────────────────────────────────────┘      │
│            │                                                         │
│ 系统管理   │                                                         │
│ • 租户管理 │                                                         │
│ • 资源池管理│                                                        │
└────────────┴─────────────────────────────────────────────────────────┘
```

| 项 | 规格 |
| --- | --- |
| 栅格 | `grid-template-columns: 232px 1fr` × `rows: 56px 1fr`,`min-width: 1280` |
| Topbar | 品牌区 · 面包屑 · 环境 pill (`prod-cn-1`) · 文档 · 通知 · 头像 |
| Sidebar | 四个分组:**Dashboard** / **训练 & 推理** / **制品中心** / **系统管理**,mono 小字 label 区隔 |
| Main | Page Head → Filters → Card / Table,复用整套 Geist 组件 |

### 2.2 信息架构

| 菜单组 | 菜单项 | 原型页 ID | 生产路径 | 设计状态 |
| --- | --- | --- | --- | :---: |
| — | Dashboard | `home` | `/dashboard` | 占位(规划中) |
| 训练 & 推理 | 工作区 | `dev` | `/workspaces` | 占位(超出本期原型范围) |
|  | 计算任务 | `job` | `/jobs` | 占位 |
|  | 在线服务 | `service` | `/services` | 占位 |
| 制品中心 | 数据集 | `dataset` | `/datasets` | 占位 |
|  | 模型 | `model` | `/models` | 占位 |
|  | 镜像 | `image` | `/images` | 占位 |
| 系统管理 | **租户管理** | `tenants` / `tenant-detail` | `/tenants` · `/tenants/{name}` | ✅ 实交 |
|  | **资源池管理** | `pools` / `pool-detail` | `/resource-pools` · `/resource-pools/{name}` | ✅ 实交 |

二级菜单的能力矩阵 (含横切的认证 / RBAC) 见 [components/platform.md §4 核心功能](components/platform.md#4-核心功能)。

### 2.3 通用元素约定

- **面包屑** — `一级 / 二级 / [资源名]`,详情页第三段可点击回到列表。
- **空态** — 列表页提供「创建第一个 X」CTA + 引导链接;未实交菜单见 §3 占位页骨架。
- **错误条** — 跨租户并行 LIST 部分失败 → 列表顶部黄条「N 个租户暂时不可达,显示其余结果」(对应 `partial=true`)。
- **二次确认** — 删除 / 取消 / 强制操作弹窗显示「前置阻断信息」(如使用此资源单元的活跃 Job / Service 计数)。
- **状态徽章** — 见各章「状态展示规则」子节统一色板。`●` 实心 / `◐` 半实心 / `○` 描边表区分。
- **mono 字体** — DNS-1123 内部名 / digest / namespace / 配额数值统一 mono 渲染;display name / 描述用普通字体。
- **Tag chips** — 节点选择器 / labels / annotations / requests / limits 用 chip 渲染,溢出折叠 `+N`,hover 展开完整列表。
- **KV grid** — 详情卡内的 label / value 双列网格,默认 `160px label / 1fr value`。
- **back-nav** — 详情页头部恒为「← 返回 X 列表」单行,左对齐,点击回上一级 list。

---

## 3. 占位页骨架

未实交的菜单 (`home` · `dev` · `job` · `service` · `dataset` · `model` · `image`) 沿用同一壳:Page Head + 虚线 placeholder (badge · 标题 · 说明)。

```
┌ Page Head ─────────────────────────┐
│ // overview                        │
│ Page Title.                        │
│ Page Desc                          │
└────────────────────────────────────┘

┌ Placeholder (dashed) ──────────────┐
│  [规划中 / 超出本期原型范围]         │
│  H4 + 一段说明文案                  │
└────────────────────────────────────┘
```

每个占位页保留菜单入口和路由,main 区只渲染 Page Head + 单个 dashed placeholder 块:

| 菜单项 | Page Title | 一句说明 (link 到详细 doc) |
| --- | --- | --- |
| Dashboard | `Dashboard.` | 登录后默认落地页,后续展示可见租户数 / 活跃任务 / 在线服务 / GPU 利用率,见 [monitoring.md](monitoring.md)。 |
| 工作区 | `工作区。` | Jupyter / VSCode 等开发容器入口,见 [components/compute-service.md](components/compute-service.md)。 |
| 计算任务 | `计算任务。` | PyTorchJob / MPIJob / 自定义训练任务,见 [components/compute-service.md](components/compute-service.md)。 |
| 在线服务 | `在线服务。` | 模型在线推理与路由,见 [components/compute-service.md](components/compute-service.md)。 |
| 数据集 / 模型 / 镜像 | 同菜单项名 | 制品中心,见 [components/artifact-hub.md](components/artifact-hub.md)。 |

placeholder 块固定四段:
1. badge — 「规划中」或「超出本期原型范围」二选一(对齐 IA 表的设计状态)。
2. H4 标题 — 与菜单项名一致。
3. 简要说明 — 1–2 句话,直接引用上表的「一句说明」。
4. 「了解更多」链接 — 跳本仓库内的权威 design doc。

---

## 4. 资源池管理 (系统管理 → 资源池)

菜单只有「资源池管理」,**资源单元作为下属能力嵌在池详情 Tab**——这是 ResourceUnit 在 Platform UI 中的唯一入口。

### 4.1 页面入口

| 入口 | 原型页 ID | 生产路径 | 权限 |
| --- | --- | --- | --- |
| 资源池列表 | `pools` | `/resource-pools` | 已登录 (读)、`system-admin` (写) |
| 资源池详情 | `pool-detail` | `/resource-pools/{name}` | 已登录 (读)、`system-admin` (写) |

资源池 / 资源单元都是全集群对象,不按租户过滤。

### 4.2 列表页

```
Page Head:  // system / resource pools   资源池管理。      [+ 新建资源池]

Filters:    🔍 关键字搜索  |  节点选择器 ▾  |  排序 ▾  |  重置

Card › Table
┌──────────┬──────────────────┬────────────────────────────┬──────┬───────────┬───────┐
│ 名称      │ 描述              │ 节点选择器 (tag chips)      │ 单元 │ 创建时间   │ 操作   │
├──────────┼──────────────────┼────────────────────────────┼──────┼───────────┼───────┤
│ gpu-a100 │ A100 训练池       │ [gpu.product=A100…] +1     │  3   │ 2026-03…  │详情 删除│
│ gpu-h100 │ H100 训练/推理…  │ [product=H100][network=ib]+1│  4   │ 2026-04…  │详情 删除│
│ gpu-l40s │ L40S 推理池       │ [gpu.product=L40S]          │  2   │ 2026-04…  │详情 删除│
│ cpu-medium│通用 CPU 池…      │ [arch=amd64][pool=cpu-medium]│ 2   │ 2026-02…  │详情 删除│
│ cpu-large │大内存 CPU 池…    │ [arch=amd64][memory-tier]   │  1   │ 2026-01…  │详情 删除│
│ cpu-arm-edge│ARM 边缘推理池…│ [arch=arm64]                │  —   │ 2026-05…  │详情 删除│
└──────────┴──────────────────┴────────────────────────────┴──────┴───────────┴───────┘
Footer: 共 6 个资源池                       ‹ [1] ›       每页 20 条
```

**列定义**:

| 列 | 来源 | 备注 |
| --- | --- | --- |
| 名称 | cluster-manager `ResourcePool.metadata.name` | mono link,点击进详情 |
| 描述 | `description` | 截断 + hover |
| 节点选择器 | `node_selector` | 渲染为 tag chips,溢出折叠 `+N` |
| 单元 | `compute.ListResourceUnits(pool, limit=0)` 计数 | mono;单池失败 → `—` |
| 创建时间 | `created_at` | mono muted |
| 操作 | — | 详情 / 删除 (`system-admin`) |

**过滤**:关键字搜索 (name / description) · 节点选择器筛选 · 排序 (创建时间 ▾ / 名称 ▾) · 重置。
**可见性**:所有已登录用户可读;`system-admin` 写。

> 池数预期 < 50,聚合策略为列表后并发查询 `compute.ListResourceUnits`。`5–10 秒 LRU` 为后续优化项。

### 4.3 详情页 Tab

详情页头部固定四元素:back-nav · 标题 (mono) · 状态徽章 · 描述,以及右上角的当前 path 提示 (`resource-pools/<name>`)。

```
← 返回资源池列表
gpu-a100.   [● 运行中]
A100 训练池                                resource-pools/gpu-a100

Tabs:  [基本信息]  [资源单元 (3)]  [节点匹配预览]  [审计日志]
```

#### Tab 1 · 基本信息

```
┌ 资源池信息                              [编辑] [删除资源池] ┐
│  KV grid (160px label / 1fr value)                       │
│   名称 (mono) · 描述                                       │
│   节点选择器 (tag list)                                    │
│   容忍配置  (tag list: key · op · value · effect)          │
│   扩展元数据 · 创建时间 · 最近更新                          │
└──────────────────────────────────────────────────────────┘

┌ 管理提示 ────────────────────────────────────────────────┐
│  说明:Node label / taint 由管理员 kubectl 维护            │
│  将匹配的节点标签 (tags)                                  │
└──────────────────────────────────────────────────────────┘
```

字段映射:

| 字段 | 来源 | 可改 |
| --- | --- | --- |
| 名称 | `name` (mono · DNS-1123) | ✗ |
| 描述 | `description` | ✅ |
| 节点选择器 | `node_selector` (K=V tag chips) | ✅ |
| 容忍配置 | `tolerations[]` (key · op · value · effect chips) | ✅ |
| 扩展元数据 | `metadata` (labels / annotations) | ✅ |
| 创建时间 / 最近更新 | `created_at` / `updated_at` (mono muted) | ✗ |

「管理提示」是固定文案块,提醒 Node label / taint 由集群管理员通过 `kubectl` 维护,**UI 不下发**。

#### Tab 2 · 资源单元

```
Filters: 🔍 搜索资源单元                       [+ 新建资源单元]

┌──┬──────────────────┬──────────────────────┬──────────────────────┬──────────────┬──────┐
│▸ │ 名称              │ requests (tags)      │ limits (tags)        │额外节点选择器 │操作   │
├──┼──────────────────┼──────────────────────┼──────────────────────┼──────────────┼──────┤
│▸ │ a100-1x-large    │ cpu=8 mem=64Gi gpu=1 │ cpu=8 mem=64Gi gpu=1 │ —            │编辑 删│
│  └─ 展开行:命名约定 / 合并节点选择器 (pool ⊕ unit,Pool 优先)                              │
│▸ │ a100-4x-xlarge   │ cpu=32 mem=256Gi gpu=4│ …                   │ —            │编辑 删│
│▸ │ a100-8x-xlarge-ib│ cpu=64 mem=512Gi gpu=8│ + hostdev=1         │ [network=ib] │编辑 删│
└──┴──────────────────┴──────────────────────┴──────────────────────┴──────────────┴──────┘
Footer: 共 3 个资源单元 · 合并选择器在展开行查看
```

要点:
- 行可展开 (`▸`),展开后渲染 `pool.nodeSelector` ⊕ `unit.nodeSelector` 合并预览 (Pool 优先;详细规则见 [components/cluster-manager.md §3.2](components/cluster-manager.md#32-展开合并规则))。
- 命名约定 `<accelerator>[-<count>x]-<tier>[-<variant>]`,如 `a100-1x-large` / `cpu-medium` / `a100-8x-xlarge-ib`,由 cluster-manager 服务兜底校验。
- requests / limits 用 tag chips 渲染,limits 缺省时显示 `…` (沿用 requests)。
- 删除前置阻断信息 (使用此 unit 的活跃 Job / Service 计数) 在二次确认弹窗呈现 → `409 unit-in-use`。

#### Tab 3 · 节点匹配预览 / Tab 4 · 审计日志

Placeholder (规划中),保留入口与 Tab pill 计数。

- **节点匹配预览** (规划):K8s typed client 反查命中 Node,显示 allocatable / requested。
- **审计日志** (规划):接入 audit_logs。

### 4.4 资源池创建表单

字段 = compute 创建请求 1:1 透传:`name` / `description` / `node_selector` / `tolerations` / `metadata`。

UI 即时校验 + compute 服务兜底。详见 [apis/platform.yaml](apis/platform.yaml) `ResourcePools` / `ResourceUnits` tag。

> Node label / taint 由管理员通过 `kubectl` 维护,**UI 不下发**。

### 4.5 资源池删除前置阻断

删除按钮二次确认对话框逐级展示:

```
确定删除资源池 [a100-pool]?
─────────────────────────────────
× 阻断:存在 3 个资源单元
  - a100-1x-large
  - a100-2x-large
  - a100-4x-large
× 阻断:存在 5 个活跃任务、2 个活跃服务
  示例:tenant-a/train-llm-7b, tenant-b/svc-id-xxx ...

请先清空后重试。
[取消]
```

- `compute.ListResourceUnits(pool)` > 0 → `409 pool-in-use`,弹窗列示例 unit name。
- `compute.ListJobs(pool, active)` / `compute.ListServices(pool, active)` > 0 → `409 unit-in-use`,弹窗列示例 job / service name 与计数。

### 4.6 状态展示规则

详情头部状态徽章基于资源池是否有活跃 unit / job / service 派生:

| 状态 | 视觉 | 含义 |
| --- | --- | --- |
| `● 运行中` | 绿色实心 | 池下存在 ≥ 1 个 unit,且 unit 被活跃 Job / Service 引用 |
| `◐ 空载` | 灰色实心 | 池下存在 unit 但未被引用 |
| `○ 未配置` | 灰色描边 | 池下尚无 unit (用户应先去 Tab 2 创建) |

### 4.7 权限可见性

| 操作 | system-admin | 其他已登录 |
| --- | :---: | :---: |
| 列表 / 详情 (含 ResourceUnit) | ✅ | ✅ (只读) |
| 创建 / 编辑 / 删除 Pool 与 Unit | ✅ | ✗ |

完整矩阵见 [auth.md §3](auth.md#3-rbac-角色)。

---

## 5. 租户管理 (系统管理 → 租户)

### 5.1 页面入口

| 入口 | 原型页 ID | 生产路径 | 权限 |
| --- | --- | --- | --- |
| 租户列表 | `tenants` | `/tenants` | 已登录;按角色裁剪 |
| 租户创建 | — | `/tenants/new` | `RequireSystemAdmin` |
| 租户详情 | `tenant-detail` | `/tenants/{name}` | 该租户 `user+` 可读;不同 Tab 写权限不同 |

### 5.2 列表页

```
Page Head:  // system / tenants   租户管理。            [+ 新建租户]

Filters:  🔍 显示名 / 名称搜索  |  状态 ▾  |  业务线 ▾  |  重置

Card › Table
┌─────────────────┬──────────────┬───────────┬───────────┬─────────────┬─────┬──────────┬────────┐
│ 显示名           │ 名称 (mono)   │ 业务线     │ 状态       │ 命名空间     │成员  │ 创建时间  │ 操作    │
├─────────────────┼──────────────┼───────────┼───────────┼─────────────┼─────┼──────────┼────────┤
│ Team A · 推理   │ team-a       │ [infra]    │ ● Active  │ team-a      │ 12  │ 2026-02… │详情 删除│
│ 推荐算法组       │ recsys-core  │ [recsys]   │ ● Active  │ recsys-core │ 28  │ 2025-12… │详情 删除│
│ 搜索算法组       │ search-rank  │ [search]   │ ● Active  │ search-rank │ 19  │ 2026-01… │详情 删除│
│ 平台 SRE        │ platform-sre │ [platform] │ ● Active  │ platform-sre│  6  │ 2025-09… │详情 删除│
│ 语音算法组       │ speech-asr   │ [recsys]   │ ● Failed  │ speech-asr  │  8  │ 2026-03… │详情 删除│
│ 客户演示租户     │ demo-cus     │ [platform] │ ● Deleting│ demo-cus    │  3  │ 2026-05… │详情 恢复│
│ 视觉算法组       │ cv-percep    │ [recsys]   │ ● Active  │ cv-percep   │ 22  │ 2025-11… │详情 删除│
│ NLP 通用        │ nlp-general  │ [recsys]   │ ● Active  │ nlp-general │ 15  │ 2025-08… │详情 删除│
└─────────────────┴──────────────┴───────────┴───────────┴─────────────┴─────┴──────────┴────────┘
Footer: 共 14 个租户 · 当前显示 1–8         ‹ [1] [2] ›      每页 8 条
```

**列定义**:

| 列 | 来源 (cluster-manager LIST) | 备注 |
| --- | --- | --- |
| 显示名 | `displayName` | 行点击进详情 |
| 名称 | `name` | mono;DNS-1123,创建后不可变 |
| 业务线 | `namespace` (顶层组织维度,与 `spec.namespace.name` 这个 K8s namespace 区分) | pill 渲染 |
| 状态 | `status.phase` | 见 §5.6 |
| 命名空间 | `spec.namespace.name` | mono |
| 成员 | `user_tenant_roles WHERE tenant_name = X` 计数 | Platform 内补充字段,聚合查询 |
| 创建时间 | `createdAt` | mono muted |
| 操作 | — | 详情 / 删除 / 恢复 (按 `phase` 切换) |

**过滤**:显示名 / 名称模糊搜索 · 状态 ▾ · 业务线 ▾ · 重置。`status` / `namespace` (业务线) 下推 cluster-manager;`q` (关键字) 由 Platform 内存二次筛选。
**可见性**:`system-admin` 看全集群;其他角色按 `user_tenant_roles` 取 `tenant_name` 集合裁剪。

### 5.3 详情页 Tab

```
← 返回租户列表
Team A · 推理.    [● Active]  [infra]
推理团队接入租户                            tenants/team-a

Tabs:  [基本信息]  [配额 (5)]  [成员 (12)]  [审计日志]
```

详情页头部:back-nav · 显示名 · 状态徽章 · 业务线 pill · 描述 · 右上角当前 path 提示 (`tenants/<name>`)。

#### Tab 1 · 基本信息

```
┌ 展示元数据                          [编辑] [删除] ┐
│ KV grid: 显示名 · 名称 (mono, DNS-1123, 不可变) · 描述    │
│          业务线 (pill) · 命名空间 · 创建/更新时间          │
└──────────────────────────────────────────────────────────┘

┌ 租户状态     // source · cluster-manager / read-only      ┐
│ 4 个 Stat 卡:                                             │
│   ┌ phase ┐ ┌ NS 就绪 ┐ ┌ 配额条目 ┐ ┌ 成员 ┐              │
│   │Active │ │  是    │ │ 5 (3池)  │ │ 12  │              │
│   └───────┘ └────────┘ └──────────┘ └─────┘              │
│                                                          │
│ Conditions 列表 (cond-type · pill · 文案 · 时间):         │
│   • NamespaceReady     [True]  NS 已就绪 …                │
│   • QuotaReady         [True]  ElasticQuota 同步成功      │
│   • InitResourcesReady [True]  imagePullSecrets 已下发    │
└──────────────────────────────────────────────────────────┘
```

字段映射:

| 区块 | 字段 | 来源 |
| --- | --- | --- |
| 展示元数据 | 显示名 / 名称 (mono · 不可变) / 描述 / 业务线 pill / 命名空间 / 创建+更新时间 | cluster-manager `tenants.*` |
| Stat 卡 · phase | `Active` / `Failed` / `Deleting` … | `status.phase` (见 §5.6) |
| Stat 卡 · NS 就绪 | `是` / `否` | `status.conditions[type=NamespaceReady].status` |
| Stat 卡 · 配额条目 | `5 (3 池)` | `Σ quotas[].count` + 涉及 pool 数 |
| Stat 卡 · 成员 | `12` | `user_tenant_roles WHERE tenant_name` 计数 |
| Conditions 列表 | type / status pill / message / lastTransitionTime | `status.conditions[]` |

`status.conditions[]` 异常 → Stat 卡 phase 旁加红点提示,Conditions 列表里对应行用 `[False]` 红色 pill。
写权限:`system-admin` 可编辑展示元数据 + 删除 / 恢复;其他角色只读。

#### Tab 2 · 配额 (按资源池分组)

```
信息条: 按 Pool 分组 · 合计 max cpu=210, gpu=14, mem=896Gi    [+ 新增配额]

┌ 资源池 gpu-a100                          3 个配额 · max cpu=80, gpu=10 ┐
│ ┌─────────┬───────────────┬───────────────┬──────────────┬──────────┬─────┐
│ │ 配额名   │ min (tags)    │ max (tags)    │ 已用          │ 用量条    │操作 │
│ │ default │ cpu=20 gpu=2  │ cpu=40 gpu=4  │ cpu=22 gpu=2 │ ▰▰▱ 55% │编辑删│
│ │ training│ cpu=10 gpu=2  │ cpu=30 gpu=4  │ cpu=28 gpu=4 │ ▰▰▰ 93%⚠│编辑删│
│ │ eval    │ cpu=2  gpu=0  │ cpu=10 gpu=2  │ cpu=0  gpu=0 │ ▱   0%  │编辑删│
└────────────────────────────────────────────────────────────────────────┘

┌ 资源池 gpu-h100                          1 个配额 · max cpu=80, gpu=4  ┐
│ inference                cpu=40 gpu=2 → cpu=80 gpu=4    80%            │
└────────────────────────────────────────────────────────────────────────┘

┌ 资源池 cpu-medium                        1 个配额 · max cpu=50         ┐
│ default                  cpu=10 → cpu=50                16%            │
└────────────────────────────────────────────────────────────────────────┘
```

要点:
- **信息条** — 顶部 1 行 `Σ max` 跨池合计 (cpu / gpu / memory) **仅作肉眼参考,不做硬阻断**,决策见 [components/platform.md §4.1 租户编排](components/platform.md#41-租户编排)。
- **分组卡** — 按 `quota.pool` 切分,卡头展示该 pool 的配额计数与 max 小计。多池则多个并列卡;单条配额时简化为一行内联表达 (见 `gpu-h100` / `cpu-medium` 示例)。
- **行字段** — `name` (`(pool, name)` 创建后不可变,编辑态置灰) · `min` (tag chips) · `max` (tag chips) · `used` (实时 `compute.GetQuotaUsage`) · 用量条 (`used / max` 比例,≥ 90% 加 `⚠` 警示) · 操作 (编辑 / 删除)。

写权限:`system-admin` 或本租户 `tenant-admin`。

#### Tab 3 · 成员

```
Filters: 🔍 用户名 / 邮箱  |  角色 ▾                       [+ 添加成员]

┌────────────┬──────┬──────────────────────┬───────────────┬──────────┬─────────┐
│ 用户名(mono)│显示名│ 邮箱                  │ 角色           │ 加入时间  │ 操作     │
├────────────┼──────┼──────────────────────┼───────────────┼──────────┼─────────┤
│ zhang.wei  │ 张伟 │ zhang.wei@axisml.io  │[tenant-admin] │ 2026-02… │改角色 移除│
│ li.na      │ 李娜 │ li.na@axisml.io      │[tenant-admin] │ 2026-02… │改角色 移除│
│ wang.lei   │ 王磊 │ wang.lei@axisml.io   │[tenant-admin] │ 2026-03… │改角色 移除│
│ chen.xi    │ 陈曦 │ chen.xi@axisml.io    │[user]         │ 2026-02… │改角色 移除│
│ liu.yang   │ 刘洋 │ liu.yang@axisml.io   │[user]         │ 2026-02… │改角色 移除│
│ zhao.min   │ 赵敏 │ zhao.min@axisml.io   │[user]         │ 2026-02… │改角色 移除│
│ …          │ …    │ …                    │ …             │ …        │ …       │
└────────────┴──────┴──────────────────────┴───────────────┴──────────┴─────────┘
```

字段映射:

| 列 | 来源 |
| --- | --- |
| 用户名 | `users.username` (mono · 主键) |
| 显示名 | `users.display_name` |
| 邮箱 | `users.email` |
| 角色 | `user_tenant_roles.role_name` pill (`tenant-admin` / `user`,不允许 `system-admin`) |
| 加入时间 | `user_tenant_roles.created_at` |
| 操作 | 改角色 / 移除 |

操作约束:
- **添加** — 输入用户名 + 选角色;`role_name` 不允许 `system-admin` → `400 role-not-bindable`。
- **自我保护** — 不能移除 / 降级自己**最后一个** `tenant-admin` 角色 → `409 last-tenant-admin`,UI 提示「请先指定其他租户管理员」。

写权限:`system-admin` 或本租户 `tenant-admin`。

#### Tab 4 · 审计日志

Placeholder (规划中,详见 §6 后续设计)。保留入口与 Tab pill。

### 5.4 租户创建表单 (`system-admin` only)

字段 = cluster-manager 创建请求 1:1 透传:

| 字段 | 说明 |
| --- | --- |
| `name` | 内部名,DNS-1123,创建后不可变 |
| `displayName` / `description` / `namespace` (业务线) | 展示元数据,可改 |
| `namespace.name` | 渲染目标 K8s namespace,创建后不可变 |
| `quotas[]` | 初始配额数组,可后续从详情页 Tab 2 增删 |
| `initResources` | 初始 Secret / ConfigMap / SA / RBAC (Vault / Sealed Secrets 接入为 TBD) |

UI 即时校验 + cluster-manager 兜底。完整字段清单与校验规则见 [apis/platform.yaml](apis/platform.yaml) `Tenants` tag。

### 5.5 列表 / 详情通用操作约束

- **DELETE 租户** — 前置检查 `user_tenant_roles WHERE tenant_name = :name`;非空 → `409 tenant-has-members`,二次确认弹窗列出残留成员。
- **PATCH 租户** — 不可变字段 `name` / `namespace.name` / `quotas[].(pool, name)` 在表单中置灰。
- **RESTORE 租户** — 仅 `system-admin`;对软删后的租户从 retention 窗口内恢复(详见 [components/compute-service.md](components/compute-service.md#41-tenant))。

### 5.6 状态展示规则

| `phase` | 视觉 |
| --- | --- |
| `Creating` | 灰色 + spinner |
| `Active` | 绿色实心 (前端解锁该租户的提交按钮) |
| `Failed` | 红色 (初始 NS / Quota 同步失败) |
| `Deleting` | 灰色 + spinner |
| `Deleted` (软删) | 灰色 (列表默认隐藏,过滤器开启「显示已删除」时可见;`system-admin` 可恢复) |

`conditions[]` 异常时在 Stat 卡 phase 旁加红点提示,详细原因在 Tab 1 的 Conditions 列表展开查看。

### 5.7 权限可见性

| 操作 | system-admin | tenant-admin (@self) | user (@self) |
| --- | :---: | :---: | :---: |
| 列出 | 全集群 | 本租户 | 本租户 |
| 创建 / 删除 / 恢复 | ✅ | ✗ | ✗ |
| 编辑展示元数据 (Tab 1) | ✅ | ✗ | ✗ |
| 配额 CRUD (Tab 2) | ✅ | ✅ | ✗ |
| 成员 CRUD (Tab 3) | ✅ | ✅ | ✗ |
| 查看配额 / 成员 | ✅ | ✅ | ✅ (仅查看) |

完整矩阵见 [auth.md §3](auth.md#3-rbac-角色)。

---

## 6. 后续设计

### 6.1 待补 ASCII mockup

下列页面尚无 ASCII mockup,待原型升级到对应菜单时按 §4 / §5 模板补齐:

- Dashboard 卡片占位 (我可见的租户 / 活跃任务 / 在线服务 / 工作区 / 配额 Top N / GPU 利用率趋势 / 最近事件流)
- 工作区列表 / 创建表单 / 详情页 (概览 · 访问 Tab)
- 计算任务列表 / 创建表单 / 详情页 (概览 · 副本 · 事件 · 日志 Tab)
- 在线服务列表 / 创建表单 / 详情页 (概览 · 访问 · 指标 Tab)
- 制品中心:模型 / 镜像 / 数据集 列表骨架与详情页 (概览 · 版本 · 后端 Tab)
- 系统管理 · 用户与角色页面
- 应用中心 (智能体 / Skills / MCP) 页面 (整套)

补齐顺序与各功能后续工作节奏对齐 ([components/platform.md §9 后续工作](components/platform.md#9-后续工作))。

### 6.2 待补 UI 设计 (横切)

- **应用中心 (Agent / Skills / MCP)** — 页面结构、列表字段、创建表单。
- **审计日志 UI** — 按 `target` 前缀检索 (`tenant:` / `job:` / `service:` / `resource-pool:` / `workspace:`),含告警规则模板入口。
- **OIDC 登录页** — OIDC 接入后的登录跳转 UX。
- **多集群 / 多区域选择器** — 顶栏增加集群切换器。

### 6.3 待补 UI 设计 (功能模块)

- **租户**:
  - 配额硬校验 / 分层配额 UI (依赖上游 ElasticQuota `parent` 字段);
  - 「已归档租户」管理界面 (restore 入口);
  - `initResources` 表单深度;
  - 租户克隆向导。
- **资源池**:
  - 按租户的池可见性 (池 → 租户白名单);
  - 节点匹配预览 Tab;
  - 池容量聚合 (allocatable / requested);
  - 资源单元成本元数据 `cost_per_hour` 列。
- **工作区**:
  - 事件 / 日志 / 副本 Tab (待 compute 端点扩展);
  - 闲时自动 stop 配置入口;
  - 孤儿 PVC 清理 UI;
  - 创建表单预设 (镜像 + 启动命令 + 资源单元 一键填好);
  - SSH 接入面板;
  - 多容器 Workspace (jupyter + tensorboard sidecar) 表单。
- **计算任务**:
  - per-role ResourceUnit 表单 (解锁 master CPU + worker GPU);
  - 任务模板 / 重新提交 UX (spec 反填);
  - DAG 工作流编辑器;
  - SSE / WebSocket 增量列表;
  - `(custom, *)` JSON schema 编辑器。
- **在线服务**:
  - 事件 / 日志 / 副本 Tab (后续工作);
  - 流量切换与灰度 UI (weighted route / canary / 自动指标判定回滚);
  - 自动扩缩容配置 (HPA / KEDA);
  - 多 role 独立扩缩;
  - 多端口 / 多协议;
  - API key 轮换 UI;
  - LLM 专项指标看板 (tokens/sec / TTFT / TBT / KV cache / batch utilization);
  - 告警与 SLO 配置 (AlertManager 集成)。
- **制品中心**:
  - 跨制品引用 UI (待 artifact-hub 引用方案定稿);
  - 镜像 Layer 浏览 Tab (zot manifest 解析 + per-layer 大小展示);
  - 数据集样本预览 (按 `format` 取首 N 行);
  - 浏览器直传支持范围扩展 (现仅 S3 Kind 小文件;OCI Kind 需在浏览器实现 chunked push,工作量高);
  - 制品签名 / SBOM 展示 (cosign / notation / trivy 集成,等待 artifacts 服务支持);
  - 制品配额展示 (per namespace / Kind 总大小 / 总数,等待 artifacts 服务 `size_bytes` 入表)。

---

## 7. 相关引用

- [components/platform.md](components/platform.md) — 后端业务编排、跨服务调用、PG schema
- [auth.md](auth.md) — RBAC 角色矩阵、JWT 颁发、IdentityProvider
- [apis/platform.yaml](apis/platform.yaml) — REST API 字段契约
- [monitoring.md](monitoring.md) — Dashboard 与服务指标数据来源
- [overview.md](overview.md) — 系统概念与组件关系
- [components/compute-service.md](components/compute-service.md) — Job / Service / ResourcePool / ResourceUnit 字段权威
- [components/cluster-manager.md](components/cluster-manager.md) — Tenant 字段权威
- [components/artifact-hub.md](components/artifact-hub.md) — 制品中心字段权威
