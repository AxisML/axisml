# AxisML Platform UI 设计

## 1. 概述

本文是 Platform 前端 UI 的设计文档,集中描述页面结构、菜单与导航、列表页字段、详情页 Tab、创建/编辑表单字段、状态展示规则与权限可见性,作为前端开发与产品视觉评审的唯一对齐入口。

- **后端业务编排** (跨服务调用顺序、写入路径、一致性策略、PG schema) 见 [components/platform.md](components/platform.md)。
- **用户认证与角色矩阵** (RBAC 完整定义、JWT 颁发、IdentityProvider 接口) 见 [auth.md](auth.md)。
- **REST API 字段契约** 见 [apis/platform.yaml](apis/platform.yaml)。
- **Dashboard / 服务指标数据来源** 见 [monitoring.md](monitoring.md)。
- **整体系统概念** (Tenant / ResourcePool / Job / Service / Artifact) 见 [overview.md](overview.md)。

> 各菜单按 §2.2 信息架构组织,列表 / 详情 / 表单布局在下文对应章节落档:Dashboard §3、资源池管理 §4、租户管理 §5、工作区 §6、计算任务 §7、在线服务 §8、制品中心(数据集 / 模型 / 镜像)§9。本文只描述**布局与字段呈现**;字段契约见 [apis/platform.yaml](apis/platform.yaml),状态机与编排见各服务文档,权限矩阵见 [auth.md](auth.md)。

---

## 2. 整体外壳 (App Shell)

### 2.1 栅格结构

```
┌──────────────────────────────────────────────────────────────────────┐
│ Topbar (56px)                                                        │
│  [A] AxisML │ Breadcrumbs …………………………… │ 租户 ▾ · 📘 · 🔔 · 头像 │
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
| Topbar | 品牌区 · 面包屑 · **租户切换器** · 文档 · 通知 · 头像 |
| 租户切换器 | 下拉显示当前用户绑定的租户列表,选中项即所有租户内菜单(工作区 / 任务 / 服务 / 制品)的操作上下文。`system-admin` 额外提供「全部租户」选项(跨租户聚合列表);单租户用户隐藏控件直接锁定 |
| Sidebar | 四个分组:**Dashboard** / **训练 & 推理** / **制品中心** / **系统管理**,mono 小字 label 区隔 |
| Main | Page Head → Filters → Card / Table,复用整套 Geist 组件 |

### 2.2 信息架构

| 菜单组 | 菜单项 | 生产路径 | 租户作用域 | UI 详设 |
| --- | --- | --- | :---: | :---: |
| — | Dashboard (中文 首页) | `/dashboard` | 切换器联动 | §3 |
| 训练 & 推理 | 工作区 | `/workspaces` · `/workspaces/{name}` | 租户内 | §6 |
|  | 计算任务 | `/jobs` · `/jobs/{name}` | 租户内 | §7 |
|  | 在线服务 | `/services` · `/services/{name}` | 租户内 | §8 |
| 制品中心 | 数据集 | `/datasets` · `/datasets/{name}` | 租户内 | §9 |
|  | 模型 | `/models` · `/models/{name}` | 租户内 | §9 |
|  | 镜像 | `/images` · `/images/{name}` | 租户内 | §9 |
| 系统管理 | **租户管理** | `/tenants` · `/tenants/{name}` | 全集群 | §5 |
|  | **资源池管理** | `/resource-pools` · `/resource-pools/{name}` | 全集群 | §4 |

二级菜单的能力矩阵 (含横切的认证 / RBAC) 见 [components/platform.md §4 核心功能](components/platform.md#4-核心功能)。

### 2.3 通用元素约定

- **面包屑** — `一级 / 二级 / [资源名]`,详情页第三段可点击回到列表。
- **空态** — 列表页无数据时渲染居中插画 + 「创建第一个 X」CTA + 引导链接;加载中用骨架行占位。
- **视图切换** — 工作区(§6)与制品中心(§9,数据集 / 模型 / 镜像)列表页支持「列表 / 卡片」双视图,切换控件 `[☰ 列表 | ▦ 卡片]` 居 Filters 行右端,默认列表视图,偏好持久化到 localStorage。两视图共享同一套过滤 / 排序 / 分页结果,仅呈现形态不同;卡片视图为响应式网格(每行 2–4 张)。其余列表页(任务 / 服务 / 租户 / 资源池)仅列表视图。
- **租户作用域** — 工作区 / 任务 / 服务 / 制品菜单的列表与创建均隶属 topbar 当前租户;切换租户即整页刷新数据。系统管理(租户 / 资源池)为全集群,不受切换器影响。
- **错误条** — 跨租户并行 LIST 部分失败 → 列表顶部黄条「N 个租户暂时不可达,显示其余结果」(对应 `partial=true`)。
- **二次确认** — 删除 / 取消 / 强制操作弹窗显示「前置阻断信息」(如使用此资源单元的活跃 Job / Service 计数)。
- **状态徽章** — 见各章「状态展示规则」子节统一色板。`●` 实心 / `◐` 半实心 / `○` 描边表区分。
- **mono 字体** — DNS-1123 内部名 / digest / namespace / 配额数值统一 mono 渲染;display name / 描述用普通字体。
- **Tag chips** — 节点选择器 / labels / annotations / requests / limits 用 chip 渲染,溢出折叠 `+N`,hover 展开完整列表。
- **KV grid** — 详情卡内的 label / value 双列网格,默认 `160px label / 1fr value`。
- **back-nav** — 详情页头部恒为「← 返回 X 列表」单行,左对齐,点击回上一级 list。

---

## 3. Dashboard (首页 · 登录默认落地页)

登录后默认落地路由 `/dashboard`,**作用域随 Topbar 租户切换器联动**的概览页。数据来自两个端点:`GET /api/v1/dashboard/overview`(KPI + 资源用量快照)与 `GET /api/v1/dashboard/metrics`(时序图)。字段契约见 [apis/platform.yaml](apis/platform.yaml) `Dashboard` tag,编排见 [components/platform.md §4](components/platform.md#4-核心功能),指标口径见 [monitoring.md](monitoring.md)。

### 3.1 作用域与视图

Dashboard 不自带作用域控件,完全由 Topbar 租户切换器(§2.1)决定:

| 切换器选择 | overview / metrics 请求 | 视图 | 可见角色 |
| --- | --- | --- | --- |
| 「全部租户」 | **不带** `X-Axisml-Tenant` | **全局视图**(跨租户聚合) | 仅 `system-admin` |
| 某个具体租户 | 带 `X-Axisml-Tenant: <tenant>` | **租户视图**(单租户聚合) | 该租户全部成员 |

- 单租户用户:切换器锁定,只有租户视图。
- 非 `system-admin` 且无活跃租户 → 端点返 `400 active-tenant-required`;UI 正常流程下切换器恒有选中项,此处仅作边界兜底。
- eyebrow 显示当前作用域(`// overview · 全部租户` 或 `// overview · team-a`)。

### 3.2 全局视图(`system-admin` · 全部租户)

```
Page Head:  // overview · 全部租户      Dashboard。                       [刷新 ⟳]

KPI 卡行
┌ 租户 ───────┐ ┌ 活跃任务 ───┐ ┌ 在线服务 ───┐ ┌ 工作区 ─────┐ ┌ 模型 ───────┐
│  14         │ │  23         │ │  9          │ │  17         │ │  128        │
│ 12 活跃      │ │ 运行 + 排队  │ │ 就绪 / 降级  │ │ 运行中       │ │ 含公共       │
└─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘

集群资源用量
┌ GPU ─────────────────┐ ┌ CPU ─────────────────┐ ┌ 内存 ────────────────┐
│ 38 / 56 卡            │ │ 410 / 720 核          │ │ 2.1 / 4.0 TiB         │
│ ▰▰▰▰▰▰▱▱  68%        │ │ ▰▰▰▰▱▱▱▱  57%        │ │ ▰▰▰▰▱▱▱▱  53%         │
└───────────────────────┘ └───────────────────────┘ └───────────────────────┘

时序图                                                  range: 1h · [24h] · 7d ▾
┌ 集群 GPU 利用率 ───────────────────┐ ┌ 活跃任务并发 ──────────────────────┐
│      ╱╲    ╱╲╱╲                    │ │        ╱╲___╱╲                     │
│  ╱╲╱   ╲╱╲╱     ╲____              │ │   ____╱        ╲___                │
└─────────────────────────────────────┘ └─────────────────────────────────────┘
```

KPI 卡(`DashboardOverview`):

| 卡 | 字段 | 说明 |
| --- | --- | --- |
| 租户 | `tenantCount` · `activeTenantCount` | 总数 + `Active` 计数;点击跳 `/tenants` |
| 活跃任务 | `activeJobCount` | `Running` + `Pending`;跳 `/jobs` |
| 在线服务 | `runningServiceCount` | `Ready` + `Degraded`;跳 `/services` |
| 工作区 | `runningWorkspaceCount` | `Ready`;跳 `/workspaces` |
| 模型 | `modelCount` | 含 `axisml-system` 公共;跳 `/models` |

集群资源用量 gauge(`DashboardOverview`):

| gauge | 字段 | 含义 |
| --- | --- | --- |
| GPU | `gpuUsed` / `gpuTotal` | 集群可分配 GPU 卡 |
| CPU | `cpuUsedCores` / `cpuTotalCores` | 集群 CPU 核 |
| 内存 | `memoryUsedGiB` / `memoryTotalGiB` | 集群内存 |

### 3.3 租户视图(任意成员 · 单租户)

```
Page Head:  // overview · team-a       Dashboard。                        [刷新 ⟳]

KPI 卡行(无「租户」卡)
┌ 活跃任务 ───┐ ┌ 在线服务 ───┐ ┌ 工作区 ─────┐ ┌ 模型 ───────┐
│  6          │ │  3          │ │  4          │ │  21         │
│ 运行 + 排队  │ │ 就绪 / 降级  │ │ 运行中       │ │ 本租户 + 公共 │
└─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘

配额用量(used / 本租户 Σ max)
┌ GPU ─────────────────┐ ┌ CPU ─────────────────┐ ┌ 内存 ────────────────┐
│ 6 / 10 卡             │ │ 48 / 80 核            │ │ 320 / 512 GiB         │
│ ▰▰▰▰▰▰▱▱  60%        │ │ ▰▰▰▰▰▱▱▱  60%        │ │ ▰▰▰▰▰▱▱▱  63%         │
└───────────────────────┘ └───────────────────────┘ └───────────────────────┘

时序图  range: 1h · [24h] · 7d ▾         快捷入口
┌ 本租户资源用量趋势 ────────────────┐   [+ 新建任务] [+ 新建服务] [+ 新建工作区]
│  ▁▂▄▆▇▆▄▃▂▁▂▃▅▇                    │   最近任务  train-llm-7b   ● 运行中
│                                     │   最近服务  svc-chat-api   ● 就绪
└─────────────────────────────────────┘
```

- KPI / gauge 与全局视图同一套字段,但 overview 已按 `X-Axisml-Tenant` 收敛到本租户:GPU/CPU/内存的分母 = 本租户跨池 `Σ quota.max`(对齐 §5.3 Tab 2 配额合计口径),用量 ≥ 90% 加 `⚠`。
- 「租户」KPI 卡在租户视图隐藏。
- 右侧「快捷入口」为本租户成员的高频写操作 CTA + 最近任务 / 服务摘要(取各自列表首屏前若干条,纯前端裁剪,不新增端点)。

### 3.4 时序图与时间范围

- 时序图调 `GET /api/v1/dashboard/metrics?metric=&range=&step=`,返回 `MetricSeries`(`metric` / `unit` / `series[]`);range 选择器 `1h / 24h / 7d` 改写 `range` 与 `step` 后重查。
- 全局视图查询不带 `X-Axisml-Tenant`(集群级);租户视图自动注入 label selector 收敛到本租户。
- 具体 `metric` key 与 PromQL 模板由 [monitoring.md](monitoring.md)(§2 集群 / GPU 层原生指标 + §6 查询模板)统一定义,UI 不内嵌 PromQL。

### 3.5 数据来源与降级

| 场景 | 触发 | UI 呈现 |
| --- | --- | --- |
| overview GPU/CPU/内存为 `null` | compute-service Informer cache 未同步([monitoring.md §4.1](monitoring.md)) | gauge 显示 `—` + hover「指标同步中」 |
| metrics 查询失败 | `/dashboard/metrics` 返 `502 Bad Gateway` | 图区占位「指标服务暂不可用」,KPI / gauge 不受影响 |
| 跨租户聚合部分失败 | overview `partial=true` | 页顶黄条「N 个租户暂时不可达,显示其余结果」(§2.3 错误条) |

刷新:KPI + gauge 默认 30s 轮询,亦可点 `[刷新 ⟳]` 手动拉取;时序图随 range 选择器或刷新重查。

### 3.6 权限可见性

| 视图 | system-admin | tenant-admin | user |
| --- | :---: | :---: | :---: |
| 全局视图(全部租户) | ✅ | ✗(切换器无此项) | ✗ |
| 租户视图(本租户) | ✅ | ✅ | ✅ |
| 快捷入口写操作 CTA | ✅ | ✅ | ✅(@self) |

完整矩阵见 [auth.md §3](auth.md#3-rbac-角色)。

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
| 单元 | cluster-manager LIST 内嵌 `spec.units[]` 计数 | mono |
| 创建时间 | `created_at` | mono muted |
| 操作 | — | 详情 / 删除 (`system-admin`) |

**过滤**:关键字搜索 (name / description) · 节点选择器筛选 · 排序 (创建时间 ▾ / 名称 ▾) · 重置。
**可见性**:所有已登录用户可读;`system-admin` 写。

> 单元计数随 cluster-manager 的资源池 LIST 内嵌 `spec.units[]` 一并返回,无需按池二次查询。

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

字段 = cluster-manager 创建请求 1:1 透传:`name` / `description` / `node_selector` / `tolerations` / `metadata`。

UI 即时校验 + cluster-manager 兜底。详见 [apis/platform.yaml](apis/platform.yaml) `ResourcePools` / `ResourceUnits` tag。

> Node label / taint 由管理员通过 `kubectl` 维护,**UI 不下发**。

### 4.5 资源池删除前置阻断

删除按钮二次确认对话框逐级展示:

```
确定删除资源池 [a100-pool]?
─────────────────────────────────
ℹ 提示:池内 3 个资源单元将随资源池级联删除(不阻断)
  - a100-1x-large / a100-2x-large / a100-4x-large
× 阻断:5 个活跃任务、2 个活跃服务正在引用本池
  示例:tenant-a/train-llm-7b, tenant-b/svc-id-xxx ...

请先清空活跃负载后重试。
[取消]
```

- 池内 `spec.units[]` 随 pool 级联删除,不构成删除阻断(`cluster-manager` DELETE 一并移除)。
- `compute.ListJobs(pool, active)` / `compute.ListServices(pool, active)` > 0 → `409 pool-in-use`(按 `labels.axisml.io/resource-pool=<name>` 过滤),弹窗列示例 job / service name 与计数。

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

| 列 | 来源 (compute-service LIST) | 备注 |
| --- | --- | --- |
| 显示名 | `displayName` | 行点击进详情 |
| 名称 | `name` | mono;DNS-1123,创建后不可变 |
| 业务线 | `namespace` (顶层组织维度,与 `spec.namespace.name` 这个 K8s namespace 区分) | pill 渲染 |
| 状态 | `status.phase` | 见 §5.6 |
| 命名空间 | `spec.namespace.name` | mono |
| 成员 | `user_tenant_roles WHERE tenant_name = X` 计数 | Platform 内补充字段,聚合查询 |
| 创建时间 | `createdAt` | mono muted |
| 操作 | — | 详情 / 删除 / 暂停 / 恢复 (按 `phase` 切换) |

**过滤**:显示名 / 名称模糊搜索 · 状态 ▾ · 业务线 ▾ · 重置。`status` / `namespace` (业务线) 下推 compute-service;`q` (关键字) 由 Platform 内存二次筛选。
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

┌ 租户状态     // source · compute-service / read-only      ┐
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
| 展示元数据 | 显示名 / 名称 (mono · 不可变) / 描述 / 业务线 pill / 命名空间 / 创建+更新时间 | compute-service `tenants.*` |
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

Placeholder (规划中)。保留入口与 Tab pill。

### 5.4 租户创建表单 (`system-admin` only)

字段 = compute-service 创建请求 1:1 透传:

| 字段 | 说明 |
| --- | --- |
| `name` | 内部名,DNS-1123,创建后不可变 |
| `displayName` / `description` / `namespace` (业务线) | 展示元数据,可改 |
| `namespace.name` | 渲染目标 K8s namespace,创建后不可变 |
| `quotas[]` | 初始配额数组,可后续从详情页 Tab 2 增删 |
| `initResources` | 初始 Secret / ConfigMap / SA / RBAC (Vault / Sealed Secrets 接入为 TBD) |

UI 即时校验 + compute-service 兜底。完整字段清单与校验规则见 [apis/platform.yaml](apis/platform.yaml) `Tenants` tag。

### 5.5 列表 / 详情通用操作约束

- **DELETE 租户** — 前置检查 `user_tenant_roles WHERE tenant_name = :name`;非空 → `409 tenant-has-members`,二次确认弹窗列出残留成员。
- **PATCH 租户** — 不可变字段 `name` / `namespace.name` / `quotas[].(pool, name)` 在表单中置灰。
- **RESTORE 租户** — 仅 `system-admin`;对软删后的租户从 retention 窗口内恢复(详见 [components/compute-service.md](components/compute-service.md#41-tenant))。
- **SUSPEND / RESUME 租户** — 仅 `system-admin`;`Active ⇄ Suspended`;暂停后锁定该租户的提交入口、已派生工作负载暂停调度,配额与成员 / 元数据保留;恢复即回到 `Active`。

### 5.6 状态展示规则

| `phase` | 视觉 |
| --- | --- |
| `Creating` | 灰色 + spinner |
| `Active` | 绿色实心 (前端解锁该租户的提交按钮) |
| `Suspended` | 橙色实心 (管理员手动暂停;锁定提交入口,已派生工作负载暂停调度) |
| `Failed` | 红色 (初始 NS / Quota 同步失败) |
| `Deleting` | 灰色 + spinner |
| `Deleted` (软删) | 灰色 (列表默认隐藏,过滤器开启「显示已删除」时可见;`system-admin` 可恢复) |

`conditions[]` 异常时在 Stat 卡 phase 旁加红点提示,详细原因在 Tab 1 的 Conditions 列表展开查看。

### 5.7 权限可见性

| 操作 | system-admin | tenant-admin (@self) | user (@self) |
| --- | :---: | :---: | :---: |
| 列出 | 全集群 | 本租户 | 本租户 |
| 创建 / 删除 / 暂停 / 恢复 | ✅ | ✗ | ✗ |
| 编辑展示元数据 (Tab 1) | ✅ | ✗ | ✗ |
| 配额 CRUD (Tab 2) | ✅ | ✅ | ✗ |
| 成员 CRUD (Tab 3) | ✅ | ✅ | ✗ |
| 查看配额 / 成员 | ✅ | ✅ | ✅ (仅查看) |

完整矩阵见 [auth.md §3](auth.md#3-rbac-角色)。

---

## 6. 工作区 (训练 & 推理 → 工作区)

交互式开发容器(Jupyter / VSCode 等),隶属 topbar 当前租户。字段权威见 [components/compute-service.md](components/compute-service.md)。

### 6.0 资源选择链(工作区 / 任务 / 服务通用)

三类资源的创建表单共享同一条资源引用链,后一项依赖前一项:

```
资源池 ▾  →  资源单元 ▾  →  镜像 ▾
(ResourcePool)  (池内 unit,带 cpu/mem/gpu 规格)  (本租户 + 公共镜像)
```

- 选定资源单元后,表单只读展示其 `requests / limits`(tag chips),用户不手填资源量。
- 镜像下拉合并「当前租户」+「公共(`axisml-system`)」制品;可输入完整引用兜底。
- 提交后 spec 不可变,「编辑」= 重新创建(服务的副本数除外,见 §8)。

### 6.1 页面入口

| 入口 | 生产路径 | 权限 |
| --- | --- | --- |
| 工作区列表 | `/workspaces` | 本租户成员(读);`owner` / `tenant-admin`(写) |
| 工作区详情 | `/workspaces/{name}` | 同上 |

### 6.2 列表页

```
Page Head:  // train & serve / workspaces   工作区。        [+ 新建工作区]

Filters:  🔍 名称搜索  |  状态 ▾  |  资源池 ▾  |  创建人 ▾  |  重置          [☰ 列表 | ▦ 卡片]

列表视图 (Card › Table,默认)
┌──────────────────┬──────────┬───────────────┬───────────┬────────┬─────────┬──────────────┐
│ 名称              │ 状态      │ 资源单元       │ 镜像       │ 副本    │ 创建人  │ 操作          │
├──────────────────┼──────────┼───────────────┼───────────┼────────┼─────────┼──────────────┤
│ ws-dev-zhang     │ ● 运行中  │ cpu-medium/1x │ jupyter…  │ 1/1    │ 张伟    │打开 停止 删除 │
│ ws-train-li      │ ◐ 启动中  │ gpu-a100/1x   │ pytorch…  │ 0/1    │ 李娜    │打开 停止 删除 │
│ ws-eval-wang     │ ○ 已停止  │ cpu-medium/1x │ vscode…   │ 0/0    │ 王磊    │启动 删除      │
└──────────────────┴──────────┴───────────────┴───────────┴────────┴─────────┴──────────────┘

卡片视图 (响应式网格,每行 2–4 张)
┌ ws-dev-zhang        ● 运行中 ┐ ┌ ws-train-li        ◐ 启动中 ┐ ┌ ws-eval-wang       ○ 已停止 ┐
│ cpu-medium/1x · jupyter…     │ │ gpu-a100/1x · pytorch…      │ │ cpu-medium/1x · vscode…     │
│ 副本 1/1 · 张伟              │ │ 副本 0/1 · 李娜             │ │ 副本 0/0 · 王磊             │
│ [打开] [停止] [删除]         │ │ [打开] [停止] [删除]        │ │ [启动] [删除]               │
└──────────────────────────────┘ └─────────────────────────────┘ └─────────────────────────────┘

Footer: 共 3 个工作区                                  ‹ [1] ›        每页 20 条
```

**视图切换**:Filters 行右端 `[☰ 列表 | ▦ 卡片]` 段控件切换,默认列表视图(约定见 §2.3)。
**字段(两视图一致)**:名称(mono link)· 状态(phase 徽章,见 §6.6)· 资源单元(`pool/unit` mono)· 镜像(截断 + hover)· 副本(`ready/desired`)· 创建人 · 操作。卡片视图按「标题行(名称 + 状态)→ 资源单元 · 镜像 → 副本 · 创建人 → 操作按钮」纵向堆叠。
**操作**:打开(跳 endpoint,如 Jupyter)· 启动 / 停止(scale ↔ 0)· 删除(二次确认含「是否一并删除数据卷 PVC」)。

### 6.3 详情页 Tab

```
← 返回工作区列表
ws-dev-zhang.   [● 运行中]                          workspaces/ws-dev-zhang
开发调试环境
[🔗 打开]  [停止]  [删除]

Tabs:  [基本信息]  [实例 (Pods)]  [日志]  [事件]
```

- **基本信息** — KV grid:名称(mono)· 显示名 · 描述 · 资源单元(`pool/unit`)· 镜像 · 访问地址(endpoint,可复制)· 数据卷(PVC name / size / storageClass)· 环境变量(chips)· 创建人 / 时间。`display_name` / `description` / labels 可编辑;其余只读。
- **实例 (Pods)** — Pod 列表:名称 · 阶段 · 节点 · 重启数 · 启动时间;行内「日志」入口。
- **日志** — 按 Pod 选择 + 实时跟随(follow)开关 + 行数选择;mono 终端样式。
- **事件** — K8s Events 时间线(类型 / 原因 / 消息 / 时间)。

### 6.4 创建表单

`name`(DNS-1123,不可变)· `display_name` / `description` · **资源池 → 资源单元 → 镜像**(§6.0)· 数据卷(PVC `size` + `storageClass`)· 环境变量 · 访问路由(可选开关)。即时校验 + compute-service 兜底。

### 6.5 删除前置

二次确认弹窗:展示是否一并删除数据卷 PVC(默认勾选删除)。

### 6.6 状态展示规则

| `phase` | 视觉 | 含义 |
| --- | --- | --- |
| `Creating` | 灰 + spinner | 资源创建中 |
| `Pending` | 灰 + spinner | 已下发,副本未就绪 |
| `Ready` | ● 绿 | `ready == desired > 0` |
| `Degraded` | ◐ 橙 | `0 < ready < desired` |
| `Failed` | ● 红 | 副本全不可用(可自愈) |
| 已停止 | ○ 灰描边 | `desired == 0`(用户主动停止) |
| `Deleting` | 灰 + spinner | 删除中 |

### 6.7 权限可见性

| 操作 | system-admin | tenant-admin | user(owner) | user(他人) |
| --- | :---: | :---: | :---: | :---: |
| 列表 / 详情 / 日志 | ✅ | ✅(本租户) | ✅ | ✅(只读) |
| 创建 | ✅ | ✅ | ✅ | — |
| 启停 / 删除 | ✅ | ✅(本租户) | ✅(自己) | ✗ |

---

## 7. 计算任务 (训练 & 推理 → 计算任务)

批处理训练 / 评估任务,隶属 topbar 当前租户。一次性运行至终态。字段权威见 [components/compute-service.md](components/compute-service.md)。

### 7.1 页面入口

| 入口 | 生产路径 | 权限 |
| --- | --- | --- |
| 任务列表 | `/jobs` | 本租户成员(读);`owner` / `tenant-admin`(写) |
| 任务详情 | `/jobs/{name}` | 同上 |

### 7.2 列表页

```
Page Head:  // train & serve / jobs   计算任务。            [+ 新建任务]

Filters:  🔍 名称搜索  |  状态 ▾  |  资源池 ▾  |  创建人 ▾  |  重置

Card › Table
┌──────────────────┬──────────┬───────────────┬────────┬─────────┬───────────┬──────────────┐
│ 名称              │ 状态      │ 资源单元       │ 副本    │ 创建人  │ 耗时       │ 操作          │
├──────────────────┼──────────┼───────────────┼────────┼─────────┼───────────┼──────────────┤
│ train-llm-7b     │ ● 运行中  │ gpu-a100/4x   │ 4      │ 张伟    │ 02:14:30  │取消 日志 详情 │
│ eval-recall      │ ● 排队中  │ gpu-h100/1x   │ 1      │ 李娜    │ —         │取消 详情      │
│ sft-baseline     │ ● 成功    │ gpu-a100/1x   │ 1      │ 王磊    │ 01:02:11  │再次提交 删除  │
│ pretrain-debug   │ ● 失败    │ gpu-a100/8x   │ 8      │ 陈曦    │ 00:08:22  │再次提交 删除  │
└──────────────────┴──────────┴───────────────┴────────┴─────────┴───────────┴──────────────┘
Footer: 共 4 个任务                                    ‹ [1] ›        每页 20 条
```

**列定义**:名称(mono link)· 状态(phase 徽章,见 §7.6)· 资源单元(`pool/unit`)· 副本 · 创建人 · 耗时(`finishedAt − startedAt`,运行中实时累计)· 操作。
**操作**:取消(仅 `Pending` / `Running`)· 日志 · 详情 · 再次提交(预填表单重建,因 spec 不可变)· 删除(终态亦可)。

### 7.3 详情页 Tab

```
← 返回任务列表
train-llm-7b.   [● 运行中]                                  jobs/train-llm-7b
LLaMA-7B 全参微调
[取消任务]  [再次提交]  [删除]

Tabs:  [基本信息]  [实例 (Pods)]  [日志]  [事件]
```

- **基本信息** — KV grid:名称 · 显示名 · 描述 · 资源单元 · 镜像 · 副本数 · 命令 / 参数(chips)· 环境变量 · 运行策略(超时 / 重试 / TTL)· 开始 / 结束时间 · 状态消息(`message`,失败时高亮)。
- **实例 (Pods)** — Pod 列表(阶段 / 节点 / 重启 / 退出码),行内日志入口。
- **日志** — 按 Pod 选择 + follow;mono 终端。
- **事件** — K8s Events 时间线。

### 7.4 创建表单

`name`(不可变)· `display_name` / `description` · **资源池 → 资源单元 → 镜像**(§6.0)· `replicas` · 命令 / 参数 · 环境变量 · 运行策略(`activeDeadlineSeconds` / `backoffLimit` / `ttlSecondsAfterFinished`)· 可选挂载模型 / 数据集制品。

### 7.5 状态展示规则

| `phase` | 视觉 | 含义 |
| --- | --- | --- |
| `Creating` / `Pending` | 灰 + spinner | 创建 / 排队等待调度 |
| `Running` | ● 绿 | 至少一个 Pod 运行中 |
| `Succeeded` | ● 绿(终态) | 全部成功 |
| `Failed` | ● 红(终态) | 超出重试上限 |
| `Canceling` | 灰 + spinner | 取消处理中 |
| `Cancelled` | ○ 灰(终态) | 已取消 |
| `Deleting` | 灰 + spinner | 删除中 |

### 7.6 权限可见性

与 §6.7 工作区一致(列表 / 详情 / 日志本租户可读;创建任意成员;取消 / 删除限 `owner` 或 `tenant-admin`)。

---

## 8. 在线服务 (训练 & 推理 → 在线服务)

常驻在线推理服务,隶属 topbar 当前租户,可暴露路由对外访问。字段权威见 [components/compute-service.md](components/compute-service.md)。

### 8.1 页面入口

| 入口 | 生产路径 | 权限 |
| --- | --- | --- |
| 服务列表 | `/services` | 本租户成员(读);`owner` / `tenant-admin`(写) |
| 服务详情 | `/services/{name}` | 同上 |

### 8.2 列表页

```
Page Head:  // train & serve / services   在线服务。        [+ 新建服务]

Filters:  🔍 名称搜索  |  状态 ▾  |  资源池 ▾  |  创建人 ▾  |  重置

Card › Table
┌──────────────────┬──────────┬───────────────┬────────┬──────────────────────┬─────────────────┐
│ 名称              │ 状态      │ 资源单元       │ 副本    │ 访问地址              │ 操作             │
├──────────────────┼──────────┼───────────────┼────────┼──────────────────────┼─────────────────┤
│ svc-chat-api     │ ● 就绪    │ gpu-h100/1x   │ 2/2    │ /services/team-a/cha…│扩缩 停止 删除 详情│
│ svc-embed        │ ◐ 降级    │ gpu-l40s/1x   │ 1/2    │ /services/team-a/emb…│扩缩 停止 删除 详情│
│ svc-rerank       │ ○ 已停止  │ cpu-large/1x  │ 0/0    │ —                    │启动 删除 详情    │
└──────────────────┴──────────┴───────────────┴────────┴──────────────────────┴─────────────────┘
Footer: 共 3 个服务                                    ‹ [1] ›        每页 20 条
```

**列定义**:名称(mono link)· 状态(phase 徽章,见 §8.6)· 资源单元 · 副本(`ready/desired`)· 访问地址(路由 path / hostname,可复制;停止时 `—`)· 操作。
**操作**:扩缩容(改副本数,服务唯一可变 spec)· 启动 / 停止 · 删除 · 详情。

### 8.3 详情页 Tab

```
← 返回服务列表
svc-chat-api.   [● 就绪]                                services/svc-chat-api
对话推理服务
[扩缩容]  [停止]  [删除]

Tabs:  [基本信息]  [监控]  [实例 (Pods)]  [日志]  [事件]
```

- **基本信息** — KV grid:名称 · 显示名 · 描述 · 资源单元 · 镜像 · 副本(`ready/desired`)· 端口(`name:port` chips)· 访问地址 · 路由配置(path / hostname / 鉴权 type / 限流 / 超时,**创建后不可变**)· 创建人 / 时间。
- **监控** — 时序折线图(取自 Prometheus,见 [monitoring.md](monitoring.md)):QPS · 延迟 p50/p95/p99 · 错误率(5xx)· CPU / 内存 / GPU 利用率(按副本)。顶部带时间范围选择(5m / 1h / 24h)。
- **实例 (Pods)** · **日志** · **事件** — 同 §6.3。

### 8.4 创建表单

`name`(不可变)· `display_name` / `description` · **资源池 → 资源单元 → 镜像**(§6.0)· `replicas` · 端口 `ports[]`(`name` / `containerPort` / `protocol`)· 命令 / 参数 · 环境变量 · **路由**(可选,创建后不可变):开关 · `path`(留空自动生成 `/services/<租户>/<name>/`)· `hostname` · 鉴权(`jwt` / `apiKey` / `none`)· 限流 · 超时。

### 8.5 扩缩容 / 启停

- **扩缩容** — 弹窗仅改副本数(`spec.roles[*].replicas`),其余 spec 置灰。
- **启动 / 停止** — 停止 = 副本缩到 0(`○ 已停止`);启动 = 恢复上次副本数。

### 8.6 状态展示规则

与 §6.6 工作区一致:`Ready`(● 绿)/ `Degraded`(◐ 橙,部分副本就绪)/ `Failed`(● 红,可自愈)/ `已停止`(○ 灰描边)/ `Creating·Pending·Deleting`(灰 + spinner)。

### 8.7 权限可见性

与 §6.7 一致;扩缩容 / 启停 / 删除限 `owner` 或 `tenant-admin`。

---

## 9. 制品中心 (数据集 / 模型 / 镜像)

三个菜单(数据集 / 模型 / 镜像)共用同一列表 / 详情 / 上传模板,仅 **spec 字段** 与 **存储后端** 不同。制品身份为 `(租户, 类型, 名称, 版本)`;同名制品下挂多个版本。列表合并「当前租户」+「公共(`axisml-system`)」制品。字段权威见 [components/artifact-hub.md](components/artifact-hub.md)。

### 9.1 三类制品差异

| 维度 | 数据集 | 模型 | 镜像 |
| --- | --- | --- | --- |
| 路径 | `/datasets` | `/models` | `/images` |
| 存储 | S3 (RustFS) | OCI (zot) | OCI (zot) |
| 专属 spec 字段 | `format`(parquet / jsonl / …) | `framework`(pytorch / …)+ `format` | `purpose`(training / inference / dev) |
| 引用 | name + version(S3 路径) | `name@digest` | `name@digest` |
| 拉取方式 | `aws s3 cp s3://…` | `docker pull <uri>` | `docker pull <uri>` |

### 9.2 列表页(以模型为例)

```
Page Head:  // artifacts / models   模型。                  [+ 上传模型]

Filters:  🔍 名称搜索  |  框架 ▾  |  可见性 ▾  |  标签 ▾  |  重置          [☰ 列表 | ▦ 卡片]

列表视图 (Card › Table,默认)
┌──────────────────┬───────────┬───────────┬────────┬──────────┬─────────┬──────────────┐
│ 名称              │ 框架       │ 最新版本   │ 版本数  │ 可见性    │ 更新时间 │ 操作          │
├──────────────────┼───────────┼───────────┼────────┼──────────┼─────────┼──────────────┤
│ llama-7b-sft     │ [pytorch] │ v3        │ 3      │ [tenant] │ 2 天前  │详情 上传新版本│
│ bge-embed        │[safetens] │ 1.5.0     │ 5      │ [public] │ 1 周前  │详情          │
│ resnet-cls       │ [onnx]    │ 2024-06   │ 2      │ [tenant] │ 3 天前  │详情 上传新版本│
└──────────────────┴───────────┴───────────┴────────┴──────────┴─────────┴──────────────┘

卡片视图 (响应式网格,每行 2–4 张)
┌ llama-7b-sft        [tenant] ┐ ┌ bge-embed          [public] ┐ ┌ resnet-cls         [tenant] ┐
│ [pytorch]                    │ │ [safetens]                  │ │ [onnx]                      │
│ 最新 v3 · 3 版本             │ │ 最新 1.5.0 · 5 版本         │ │ 最新 2024-06 · 2 版本       │
│ 更新 2 天前                  │ │ 更新 1 周前                 │ │ 更新 3 天前                 │
│ [详情] [上传新版本]          │ │ [详情]                      │ │ [详情] [上传新版本]         │
└──────────────────────────────┘ └─────────────────────────────┘ └─────────────────────────────┘

Footer: 共 3 个模型(含 1 个公共)                      ‹ [1] ›       每页 20 条
```

**视图切换**:Filters 行右端 `[☰ 列表 | ▦ 卡片]` 段控件切换,默认列表视图(约定见 §2.3);数据集 / 模型 / 镜像三个菜单共用同一偏好键。
**字段(两视图一致)**:名称(mono link)· 专属 spec(数据集=`format` / 模型=`framework` / 镜像=`purpose`,pill)· 最新版本 · 版本数 · 可见性(`tenant` / `public` pill)· 更新时间 · 操作(详情 / 上传新版本 / 删除)。卡片视图按「标题行(名称 + 可见性)→ 专属 spec → 最新版本 · 版本数 → 更新时间 → 操作按钮」纵向堆叠。
**过滤**:名称搜索 · 专属 spec ▾ · 可见性 ▾ · 标签 ▾ · 重置。公共制品对当前租户只读,写入口在两视图中均隐藏。

### 9.3 详情页(制品 + 版本列表)

```
← 返回模型列表
llama-7b-sft.   [tenant]                              models/llama-7b-sft
LLaMA-7B 监督微调权重                                 [+ 上传新版本]

┌ 元数据 ──────────────────────[编辑] ┐  ┌ 标签 / 注解 ──────────────┐
│ 框架 pytorch · 创建人 张伟          │  │ [task=chat][lang=zh] +2   │
│ 创建 2026-05-01 · 更新 2026-06-11   │  └────────────────────────────┘
└─────────────────────────────────────┘

版本列表
┌──────────┬──────────┬───────────────────────┬────────┬─────────┬──────────────────────┐
│ 版本      │ 状态      │ digest                │ 大小    │ 创建人  │ 操作                  │
├──────────┼──────────┼───────────────────────┼────────┼─────────┼──────────────────────┤
│ v3       │ ● 就绪    │ sha256:a1b2…  📋      │ 13.4GB │ 张伟    │下载 拉取命令 删除     │
│ v2       │ ● 就绪    │ sha256:9f8e…  📋      │ 13.4GB │ 张伟    │下载 拉取命令 删除     │
│ v1       │ ● 就绪    │ sha256:77cd…  📋      │ 13.1GB │ 李娜    │下载 拉取命令 删除     │
└──────────┴──────────┴───────────────────────┴────────┴─────────┴──────────────────────┘
```

- **元数据卡** — `display_name` / `description` / labels / annotations 可编辑;`spec`(框架等)/ 名称 / 可见性 创建后不可变。
- **版本列表** — 版本(mono)· 状态(见 §9.5)· digest(mono 截断 + 📋 复制)· 大小 · 创建人 · 创建时间 · 操作(下载 / 拉取命令 / 复制 digest / 删除 / 编辑该版本元数据)。
- **拉取命令** — 弹窗按存储后端给出命令(数据集 `aws s3 cp` / 模型镜像 `docker pull`)+ 临时凭证有效期提示。

### 9.4 上传流程(引导对话框)

```
上传模型 › 新版本
─────────────────────────────────────
① 基本信息   名称(新建/选已有) · 版本(OCI tag 规则) · 框架 ▾ · 显示名 · 描述 · 标签
② 获取凭证   [初始化上传] → 返回上传地址 + 临时凭证(有效期 1h)
③ 推送数据   展示 docker push / aws s3 cp 命令,用户在本地执行
④ 完成校验   [完成上传] 填入 digest → 服务端校验 → 状态转 ● 就绪
─────────────────────────────────────
[上一步]                                   [下一步 / 完成]
```

- 数据集与模型 / 镜像三步一致,仅 ① 的专属 spec 字段(`format` / `framework`+`format` / `purpose`)与 ③ 的推送命令不同。
- 上传未完成的版本停留 `◐ 上传中`,24h 未完成自动转 `失败`。

### 9.5 状态展示规则

| `status` | 视觉 | 含义 |
| --- | --- | --- |
| `Uploading` | ◐ 灰 + spinner | 已初始化,等待推送 / 完成 |
| `Ready` | ● 绿 | digest 校验通过,可引用 |
| `Failed` | ● 红 | 上传超时 / 校验失败 |
| `Deleting` / `Deleted` | 灰 | 回收中 / 已删除 |

### 9.6 权限可见性

| 操作 | system-admin | tenant-admin | user(owner) | user(他人) |
| --- | :---: | :---: | :---: | :---: |
| 列表 / 详情(本租户 + 公共) | ✅ | ✅ | ✅ | ✅ |
| 上传新版本 / 创建 | ✅ | ✅ | ✅ | — |
| 编辑元数据 / 删除 | ✅ | ✅(本租户) | ✅(自己) | ✗ |
| 设为公共(`axisml-system`) | ✅ | ✗ | ✗ | ✗ |

---

## 10. 相关引用

- [components/platform.md](components/platform.md) — 后端业务编排、跨服务调用、PG schema
- [auth.md](auth.md) — RBAC 角色矩阵、JWT 颁发、IdentityProvider
- [apis/platform.yaml](apis/platform.yaml) — REST API 字段契约
- [monitoring.md](monitoring.md) — Dashboard 与服务指标数据来源
- [overview.md](overview.md) — 系统概念与组件关系
- [components/compute-service.md](components/compute-service.md) — Tenant / Quota / Job / Service / Workspace 字段权威
- [components/cluster-manager.md](components/cluster-manager.md) — ResourcePool / ResourceUnit 字段权威
- [components/artifact-hub.md](components/artifact-hub.md) — 制品中心字段权威
