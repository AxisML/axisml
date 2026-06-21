# AxisML Platform UI 设计

## 1. 概述

Platform 前端 UI 的对齐入口：页面结构、菜单导航、列表字段、详情 Tab、表单字段、状态展示与权限可见性。只描述**布局与字段呈现**；后端编排见 [platform.md](../system_design/platform/backend.md)，字段契约见 [openapi/platform.yaml](../openapi/platform.yaml)，RBAC 见 [auth.md](../system_design/platform/auth.md)，系统概念见 [high_level_design.md](../system_design/high_level_design.md)。交互原型见 [prototype/](prototype)。

---

## 2. 整体外壳（App Shell）

### 2.1 栅格结构

```
┌──────────────────────────────────────────────────────────┐
│ Topbar (56px)  [A] AxisML │ 面包屑 …… │ 📘 · 🔔 · 头像     │
├────────────┬─────────────────────────────────────────────┤
│ Sidebar    │ Main (Page Head → Filters → Card/Table)     │
│ (232px)    │                                             │
│ 首页        │   ┌ Page Head:  标题  [Actions] ┐           │
│ 训练中心    │   └ 描述 ──────────────────────┘           │
│ 服务中心    │   ┌ Filters: 搜索 · 筛选 · 重置 ┐           │
│ 资产中心    │   └─────────────────────────────┘           │
│ 系统管理    │   ┌ Card / Section / Table ─────┐           │
└────────────┴─────────────────────────────────────────────┘
```

| 项 | 规格 |
| --- | --- |
| 栅格 | `232px 1fr` × `56px 1fr`，`min-width 1280` |
| Topbar | 品牌 · 面包屑 · 文档 · 通知 · 头像 / 用户菜单 |
| 用户菜单 | 身份、**所属租户切换**、语言（中 / 英）、主题、退出。"所属租户"恒选中且仅选中一个租户，即所有租户内菜单的操作上下文；`system-admin` 可逐个切换到任意租户 |
| Sidebar | 五组：首页 / 训练中心 / 服务中心 / 资产中心 / 系统管理 |

### 2.2 信息架构

| 菜单组 | 菜单项 | 路径 | 作用域 | 详设 |
| --- | --- | --- | :---: | :---: |
| — | 首页 | `/dashboard` | 租户联动 | §3 |
| 训练中心 | 工作区 | `/workspaces` · `/{name}` | 租户内 | §6 |
|  | 实验 | `/experiments` · `/{name}` · `/{name}/runs/{run}` | 租户内 | §11 |
|  | 自定义任务 | `/jobs` · `/{name}` · `/{name}/runs/{run}` | 租户内 | §7 |
| 服务中心 | 在线服务 | `/mlservices` · `/{name}` | 租户内 | §8 |
|  | 流量配置 | `/traffic` · `/{name}` | 租户内 | §10 |
| 资产中心 | 模型 | `/models` · `/{name}` | 租户内 | §9 |
|  | 镜像 | `/images` · `/{name}` | 租户内 | §9 |
| 系统管理 | 租户管理 | `/tenants` · `/{name}` | 全集群 | §5 |
|  | 资源池管理 | `/resource-pools` · `/{name}` | 全集群 | §4 |

> 数据集与评估为规划中能力，本期无菜单入口，见 §12。

### 2.3 通用约定

- **面包屑 / back-nav** — 详情页头部恒为"← 返回 X 列表"。
- **空态 / 加载** — 列表无数据渲染"创建第一个 X"CTA；加载用骨架行。
- **视图切换** — 工作区（§6）与资产中心（§9）列表支持`[☰ 列表 | ▦ 卡片]`，默认卡片，偏好存 localStorage；其余列表仅列表视图。
- **租户作用域** — 租户内菜单恒隶属"所属租户"选中的单一租户，切换即整页刷新；系统管理为全集群，不受影响。
- **多语言** — 中 / 英切换在用户菜单，默认随浏览器、纯前端持久化；文案 / 状态徽章 / 错误按 locale 本地化，自由文本按原文。契约见 [platform.md §5.6](../system_design/platform/backend.md#56-多语言--i18n)。
- **二次确认** — 删除 / 停止等弹窗显示前置阻断信息（如引用此资源的活跃 Job / Service 计数）。
- **状态徽章** — 各章"状态展示规则"统一色板：`●` 实心 / `◐` 半实心 / `○` 描边。
- **mono 字体** — 内部名 / digest / namespace / 数值用 mono；显示名 / 描述用普通字体。
- **chips / KV grid** — 节点选择器 / labels / requests / limits 用 chip（溢出折叠 `+N`）；详情卡用 `160px label / 1fr value` 双列网格。

---

## 3. 首页（默认落地页）

`/dashboard`，作用域随用户菜单"所属租户"联动（恒为单一租户）。两个端点：`GET /dashboard/overview`（KPI + 容量快照）与 `GET /dashboard/metrics`（时序图）。编排见 [platform.md §4.7](../system_design/platform/backend.md#47-dashboard-编排)。

| 视图 | 内容 | 可见角色 |
| --- | --- | --- |
| **租户视图** | 当前"所属租户"的活跃任务 / 在线服务 / 工作区 / 模型 KPI + 配额水位 + 用量趋势 + 快捷入口 | 该租户全部成员 |
| **集群容量** | 集群容量水位（GPU / CPU / 内存），作为系统管理员看板，不依赖租户作用域 | 仅 `system-admin` |

```
Page Head:  首页。                                   [刷新 ⟳]
            <租户> 的运行概览，看一眼负载、配额和资源用量。

KPI 卡行   活跃任务 · 在线服务 · 工作区 · 模型
资源用量   GPU / CPU / 内存 gauge   (租户=配额 used/Σmax，≥90% 加 ⚠；system-admin 看板另含集群容量)
时序图     range 1h · [24h] · 7d     快捷入口  [+ 任务] [+ 服务] [+ 工作区] + 最近任务 / 服务
```

- KPI 卡可点击跳对应列表；快捷入口为本租户高频写操作 CTA。
- **降级**：容量 / 配额用量为 `null` → gauge 显示 `—` + hover「指标同步中」；metrics 失败 → 图区占位，KPI 不受影响。
- 刷新：KPI + gauge 默认 30s 轮询，亦可手动；时序图随 range 重查。具体 metric 与 PromQL 由后端定义，UI 不内嵌 PromQL。

---

## 4. 资源池管理（系统管理 → 资源池）

菜单只有"资源池管理"，**资源单元嵌在池详情**——这是 ResourceUnit 在 UI 中的唯一入口。资源池 / 单元都是全集群对象，不按租户过滤；已登录可读，`system-admin` 可写。

### 4.1 列表页

```
Page Head:  资源池管理。                              [+ 新建资源池]
            管理节点范围和调度配置，规格档位在池详情里维护。
Filters:    🔍 关键字  |  重置

│ 名称 │ 描述 │ 节点选择器(chips) │ 资源单元 │ 创建时间 │ 操作 │
│ gpu-a100 │ A100 训练池 │ [gpu.product=A100] +1 │ 3 │ 2026-03… │ 管理 删除 │
```

**列**：名称（mono link）· 描述 · 节点选择器（chips，溢出 `+N`）· 资源单元计数（随 LIST 内嵌 `spec.units[]` 返回）· 创建时间 · 操作（管理 / 删除）。

### 4.2 详情页 Tab

头部：back-nav · 名称（mono）· 状态徽章 · 描述。Tabs：`[基本信息] [资源单元 N] [节点匹配预览]`。

- **基本信息** — KV grid：名称（mono，不可改）· 描述 · 节点选择器（K=V chips）· 容忍配置（key·op·value·effect chips）· 扩展元数据 · 创建 / 更新时间。除名称外可编辑。固定提示块：Node label / taint 由管理员 `kubectl` 维护，**UI 不下发**。
- **资源单元** — 工具栏 `🔍 搜索 [+ 新建资源单元]`；表：名称 · requests（chips）· limits（chips）· 额外节点选择器 · 操作（编辑 / 删除）。行可展开看 `pool ⊕ unit` 合并节点选择器（Pool 优先，规则见 [cluster-manager.md §3.2](../system_design/system/cluster-manager.md#32-展开合并规则)）。命名约定 `<accelerator>[-<count>x]-<tier>[-<variant>]`。删除前置阻断（使用此 unit 的活跃 Job / Service 计数）→ `409 unit-in-use`。
- **节点匹配预览** — 规划中，仅保留 Tab 入口与计数。

### 4.3 创建 / 删除

- **资源池创建表单**：`name` / `description` / 节点选择器 / 容忍配置 / 扩展元数据（与 cluster-manager 请求 1:1）。资源单元在池详情里建。
- **资源单元创建表单**：`name` / 描述 / 资源规格矩阵（CPU / 内存 / GPU 的 requests / limits，可锁定 limits=requests）/ 额外节点选择器与容忍。
- **删除资源池二次确认**：池内 units 随池级联删除（不阻断）；活跃 Job / Service 引用本池（按 `labels.axisml.io/resource-pool=<name>` 反查）> 0 → `409 pool-in-use`，弹窗列示例与计数。

### 4.4 状态展示规则

| 状态 | 视觉 | 含义 |
| --- | --- | --- |
| `● 运行中` | 绿实心 | 池下有 unit 且被活跃 Job / Service 引用 |
| `◐ 空载` | 灰实心 | 有 unit 但未被引用 |
| `○ 未配置` | 灰描边 | 尚无 unit |

---

## 5. 租户管理（系统管理 → 租户）

### 5.1 列表页

```
Page Head:  租户管理。                                [+ 新建租户]
            开通、停用或删除租户，配额和成员在详情里管理。
Filters:  🔍 名称  |  状态 ▾  |  重置

│ 租户(显示名) │ 名称(mono) │ 状态 │ 资源配额 │ 成员 │ 创建时间 │ 操作 │
│ Team A·推理 │ team-a │ ● 启用 │ gpu-h100 14/16, +1 │ 12 │ 2026-02… │ 资源配额 成员管理 停用 │
│ 语音算法组 │ speech-asr │ ◐ 停用 │ … │ 8 │ 2026-03… │ 启用 删除 │
```

**列**：显示名（行点击进详情）· 名称（mono，DNS-1123，不可变）· 状态（§5.4）· 资源配额（各池配额 chips）· 成员计数 · 创建时间 · 操作。
**操作**：资源配额 / 成员管理 / 停用 ⇄ 启用 / 删除（仅停用态可删）。
**可见性**：`system-admin` 看全集群；其他角色按绑定租户裁剪。

### 5.2 详情页 Tab

头部：back-nav · 显示名 · 状态徽章 · 描述。Tabs：`[基本信息] [资源配额] [成员 N]`。

- **基本信息** — 展示元数据（显示名 / 名称(mono,不可变) / 描述 / 命名空间 / 创建 / 更新）+ 租户状态（phase · NS 就绪 · 配额条目 · 成员数 Stat 卡 + Conditions 列表：`NamespaceReady` / `QuotaReady` / `InitResourcesReady`，异常行 `[False]` 红 pill）。`system-admin` 可编辑展示元数据。
- **资源配额（按资源池分组）** — 每个资源池一张卡，列出该池下分配的资源单元与数量及用量：

```
┌ 资源池 gpu-a100                                          [+ 添加配额] ┐
│ 资源单元        │ 数量 │ 折算(max cpu/gpu) │ 已用 │ 用量条      │ 操作  │
│ a100-1x-large  │  2   │ cpu=16 gpu=2      │ gpu=2│ ▰▰▰ 93%⚠   │ 编辑 删│
│ a100-4x-xlarge │  1   │ cpu=32 gpu=4      │ gpu=0│ ▱   0%     │ 编辑 删│
└──────────────────────────────────────────────────────────────────────┘
```

  - 用户填**资源单元 × 数量**；min/max 由 cluster-manager 据规格折算为 ElasticQuota（见 [cluster-manager.md §3.3](../system_design/system/cluster-manager.md#33-tenant-形状与配额折算)），用量条按 `used / max`（≥90% 加 `⚠`）。`pool` 创建后不可变。
  - 写权限：`system-admin` 或本租户 `tenant-admin`。

- **成员** — 工具栏 `🔍 用户名 / 邮箱 [+ 添加成员]`；表：用户名（mono）· 显示名 · 邮箱 · 角色（`tenant-admin` / `user` pill）· 加入时间 · 操作（改角色 / 移除）。添加输入用户名 + 选角色（不允许 `system-admin`）；不能移除 / 降级自己最后一个 `tenant-admin`（→ `409 last-tenant-admin`）。写权限同配额。

### 5.3 创建表单 / 操作约束（`system-admin`）

- **创建表单**：租户名称（显示名）· 租户 ID（mono，DNS-1123，= 命名空间，创建后不可变）· 初始管理员邮箱 · 初始配额（按资源池 tab 切换，每池填各资源单元数量）。
- **删除** — 前置检查成员，非空 → `409 tenant-has-members`，弹窗列残留成员。
- **停用 / 启用** — `启用 ⇄ 停用`；停用后锁定新建提交入口（任务 / 服务 / 工作区），已运行工作负载继续运行、可继续 scale / stop / delete，配额与成员保留。

### 5.4 状态展示规则

| 状态 | 视觉 | 含义 |
| --- | --- | --- |
| 启用（Active） | ● 绿实心 | 就绪，解锁提交入口 |
| 停用（Suspended） | ◐ 橙实心 | 管理员停用，锁定新建入口 |
| 创建中 / 删除中 | 灰 + spinner | NS / Quota 同步中 |
| 失败（Failed） | ▲ 红 | 初始 NS / Quota 同步失败 |

---

## 6. 工作区（训练中心 → 工作区）

交互式开发容器（Jupyter / VSCode），隶属当前租户。本租户成员可读，`owner` / `tenant-admin` 可写。字段权威见 [compute-service.md](../system_design/system/compute-service.md)。

### 6.0 资源选择链（工作区 / 自定义任务 / 实验 / 在线服务通用）

```
资源池 ▾  →  资源单元 ▾（带 cpu/mem/gpu 规格）  →  镜像 ▾（本租户 + 公共）
```

- 选定资源单元后只读展示其 requests / limits（chips），用户不手填资源量。
- 提交后 spec 不可变，"编辑"= 重新创建（服务副本数与运行中工作区的显示名 / 描述除外）。

### 6.1 列表页

```
Page Head:  工作区。                                  [+ 新建工作区]
            交互式开发容器，支持 Jupyter / VSCode，不用时随时停掉。
Filters:  🔍 名称 | 状态 ▾ | 资源池 ▾ | 创建人 ▾ | 重置      [☰ 列表 | ▦ 卡片]

卡片(默认)：图标 · 名称 · 描述 · 状态 · 资源单元 · 创建人 · [Jupyter][VSCode][终端][停止][删除]
列表：名称 · 状态 · 资源单元(pool/unit) · 镜像 · 创建人 · 操作(打开/停止/删除)
```

运行中卡片提供 Jupyter / VSCode / 终端 / 停止 / 删除；已停止仅启动 / 删除。删除二次确认含"是否一并删除数据卷 PVC"（默认勾选）。

### 6.2 详情页 Tab

头部：back-nav · 名称（mono）· 状态徽章 · 描述 · `[🔗 打开] [停止] [删除]`。Tabs：`[基本信息] [实例(Pods) N] [日志] [事件]`。

- **基本信息** — KV grid：名称 · 显示名 · 描述 · 资源单元 · 镜像 · 访问地址（可复制）· 数据卷（PVC name / size / storageClass）· 环境变量（chips）· 创建人 / 时间。运行中仅显示名 / 描述可编辑，停止后可改全部。
- **实例 / 日志 / 事件** —（各页通用）实例列出 Pod（名称 / 阶段 / 节点 / 重启 / 启动时间，行内日志入口）；日志按 Pod 选择 + follow，mono 终端；事件为 K8s Events 时间线。

### 6.3 创建表单

`name`（不可变）· 显示名 / 描述 · **资源池 → 资源单元 → 镜像**（§6.0）· 数据卷（Volume + 挂载路径，可增删）· 环境变量。

### 6.4 状态展示规则（工作区 / 在线服务通用）

| `phase` | 视觉 | 含义 |
| --- | --- | --- |
| Creating / Pending | 灰 + spinner | 创建 / 副本未就绪 |
| `Ready` | ● 绿 | `ready == desired > 0` |
| `Degraded` | ◐ 橙 | `0 < ready < desired` |
| `Failed` | ● 红 | 副本全不可用（可自愈） |
| 已停止 | ○ 灰描边 | `desired == 0`（用户停止） |
| Deleting | 灰 + spinner | 删除中 |

---

## 7. 自定义任务（训练中心 → 自定义任务）

**Job（可复用模板）→ Run（每次运行）** 两级模型，隶属当前租户。Job 是 Platform 自有模板；每次运行是一个 Run（对应 compute 的 `MLRun`，命名 `<job>-<n>`）。本租户成员可读，`owner` / `tenant-admin` 可写。编排见 [platform.md §4.2](../system_design/platform/backend.md#42-计算任务编排)。

### 7.1 列表页（Job）

```
Page Head:  自定义任务。                              [+ 新建 Job]
            提交训练 / 微调 / 数据处理任务；每次运行生成一个 Run。
Filters:  🔍 名称 | 创建人 ▾ | 重置

│ 名称 │ 最近运行状态 │ 运行数 │ 创建人 │ 更新时间 │ 操作(运行/详情/编辑/删除) │
│ train-llm-7b │ ● 运行中 │ 4 │ 张伟 │ 2d ago │ … │
```

- 名称（mono link → Job 详情）· 最近运行状态（最新 Run 的 phase，§7.4）· 运行数 · 创建人 · 更新时间 · 操作。
- 删除：有活跃 Run 时禁用并提示 `409 job-has-active-runs`，否则级联软删全部 Run。
- "最近运行状态""运行数"由 Platform 实时回源 compute（`ListMLRuns(labelSelector=axisml.io/job=<job>)`），不落 Platform 表。

### 7.2 Job 详情页 Tab

头部：back-nav · 名称（mono）· 状态徽章 · 描述 · `[▶ 运行] [编辑] [删除]`。Tabs：`[配置] [运行历史(Runs)]`。

- **配置** — Job 模板 KV grid：名称 · 显示名 · 描述 · 资源池 / 单元（默认）· 镜像 · 副本数 · 命令 / 参数（chips）· 环境变量 · 数据卷 · 运行策略（超时 / 重试 / TTL）。编辑只影响**之后**触发的 Run。
- **运行历史** — Run 表（实时回源 compute）：Run（`<job>-<n>` mono link）· 状态 · 资源单元 · 副本 · 触发人 · 耗时 · 操作（取消仅 Pending / Running · 日志 · 详情 · 删除终态）。

### 7.3 Run 详情页（`/jobs/{name}/runs/{run}`）

头部：back-nav（→ Job）· Run 名（mono）· 状态徽章 · `[取消运行] [删除]`。Tabs：`[概览] [实例(Pods)] [日志] [事件]`。概览为该 Run 快照（资源单元 / 镜像 / 副本 / 命令 / 环境变量 / 运行策略 / 触发期 override / 起止时间 / 状态消息）。Run spec 创建后不可变。

### 7.4 表单与触发

- **Job 创建 / 编辑表单**：`name`（不可变）· 显示名 / 描述 · 镜像 · **资源池 → 资源单元 → 副本数** · 命令 / 参数 · 环境变量 · 数据卷 · 运行策略（超时 / 重试）。保存即写模板，**不触发运行**。
- **触发运行对话框**：默认按模板直接运行；展开"高级 · 本次覆盖"可改镜像 / 模型版本 · 资源单元 · 副本与资源 · 超参（命令 / 参数 / 环境变量）。**不可**改 backend 与 role 拓扑（需改模板）。确认后对引用制品版本预检 `Ready`，失败 `400`；成功则创建 `<job>-<n>` 并跳 Run 详情。

### 7.5 状态展示规则（Run phase）

| `phase` | 视觉 | | `phase` | 视觉 |
| --- | --- | --- | --- | --- |
| Creating / Pending | 灰 + spinner | | `Succeeded` | ● 绿（终态） |
| `Running` | ● 绿 | | `Failed` | ● 红（终态） |
| Canceling | 灰 + spinner | | `Cancelled` | ○ 灰（终态） |

> Job 列表"最近运行状态"取最新 Run 的 phase；无 Run 显示 `○ 从未运行`。

---

## 8. 在线服务（服务中心 → 在线服务）

常驻在线推理服务，隶属当前租户，可暴露路由对外访问；多版本灰度由 §10 承接。权限与可读写同工作区。字段权威见 [compute-service.md](../system_design/system/compute-service.md)。

### 8.1 列表页

```
Page Head:  在线服务。                                [+ 新建服务]
            常驻在线推理服务，可对外访问并按需扩缩容。
Filters:  🔍 名称 | 状态 ▾ | 资源池 ▾ | 重置

│ 名称 │ 状态 │ 副本(ready/desired) │ 资源单元 │ 访问地址 │ 操作 │
│ svc-chat-api │ ● 就绪 │ 2/2 │ gpu-h100/1x │ /services/team-a/cha… │ 编辑 扩缩 停止 删除 │
```

操作：编辑 · 扩缩容（改副本数）· 启动 / 停止（停 = 缩到 0）· 删除。状态规则同 §6.4。

### 8.2 详情页 Tab

头部：back-nav · 名称（mono）· 状态徽章 · 描述 · `[编辑] [扩缩容] [停止] [删除]`。Tabs：`[概览] [监控] [实例(Pods) N] [日志] [事件]`。

- **概览** — KV grid：名称 · 显示名 · 描述 · 模型版本 · 镜像 · 资源单元 · 副本（ready/desired）· 端口（`name:port` chips）· 访问地址 · 路由（path / hostname，**创建后不可变**）· 创建人 / 时间。
- **监控** — 时序折线（取自 Prometheus）：QPS · 延迟 p50/p95/p99 · 错误率（5xx）· CPU / 内存 / GPU 利用率；时间范围 5m / 1h / 24h。
- **实例 / 日志 / 事件** — 同 §6.2。

### 8.3 创建表单 / 扩缩

- **创建表单**：`name`（不可变）· 显示名 / 描述 · 模型版本 + 推理镜像 · **资源池 → 资源单元 → 副本数** · 端口 `ports[]`（name / port）· 路由（可选开关 · path 留空自动生成 `/services/<租户>/<name>/`）。
- **扩缩容** — 弹窗仅改副本数；启动 / 停止 = 副本恢复 / 缩到 0。

---

## 9. 资产中心（模型 / 镜像）

模型与镜像共用同一列表 / 详情 / 上传模板，仅 **spec 字段** 与 **存储后端** 不同。制品身份 `(租户, 类型, 名称, 版本)`；同名制品下挂多版本，列表合并"本租户 + 公共（`axisml-system`）"。字段权威见 [artifact-hub.md](../system_design/system/artifact-hub.md)。

| 维度 | 模型 | 镜像 |
| --- | --- | --- |
| 路径 / 存储 | `/models` · OCI（zot） | `/images` · OCI（zot） |
| 专属 spec | `framework`（+ 任务标签 / 参数量标签） | `purpose`（training / inference / dev / custom） |
| 引用 / 拉取 | `name@version` · `docker pull` | `name@version` · `docker pull` |

### 9.1 列表页（以模型为例）

```
Page Head:  模型。                                    [+ 新建模型]
            管理模型及版本，可在任务和在线服务中按版本引用。
Filters:  🔍 名称 | 框架 ▾ | 标签 ▾ | 重置          [☰ 列表 | ▦ 卡片]

卡片(默认)：图标 · 名称 · 描述 · 最新版本 + 版本数 · 更新时间
列表：名称 · 框架 · 最新版本 · 版本数 · 标签 · 更新时间 · 操作(上传新版本)
```

公共制品在名称旁显示公共图标、对当前租户只读（写入口隐藏）。

### 9.2 详情页（制品 + 版本列表）

头部：back-nav · 名称（mono）· 描述 · `[+ 上传新版本]`。元数据卡（框架 / 创建人 / 时间 + 标签 / 注解，显示名 / 描述 / 标签可编辑；spec / 名称不可变）+ 版本列表：

```
│ 版本 │ 状态 │ 来源 │ digest │ 大小 │ 创建人 │ 操作(拉取命令 / 删除) │
│ v3 │ ● 就绪 │ Oras 推送 │ sha256:a1b2… 📋 │ 13.4GB │ 张伟 │ … │
```

- 版本（mono）· 状态（§9.4）· 来源（Web 上传 / Oras 推送 / 远端登记）· digest（mono + 复制）· 大小 · 创建人 · 操作。
- **拉取命令** 弹窗按存储后端给出命令（`docker pull` / `aws s3 cp`）+ 临时凭证有效期（1h）提示。

### 9.3 新建 / 上传

- **新建制品表单**：名称（mono，英文）· 专属 spec（模型=框架 + 任务 / 参数量标签；镜像=用途）· 描述 · 自定义标签（K=V）。
- **上传新版本（引导对话框）**：① 版本号 + 描述 → ② 选方式（Web 上传 / 远端登记 / ORAS / docker push）→ ③ 按方式展示推送命令或上传区 → ④ 完成校验（填 digest，服务端校验后转 `● 就绪`）。未完成停留 `◐ 上传中`，24h 超时转失败。

### 9.4 状态展示规则

| `status` | 视觉 | 含义 |
| --- | --- | --- |
| `Uploading` | ◐ 灰 + spinner | 已初始化，待推送 / 完成 |
| `Ready` | ● 绿 | digest 校验通过，可引用 |
| `Failed` | ● 红 | 上传超时 / 校验失败 |
| Deleting / Deleted | 灰 | 回收中 / 已删除 |

---

## 10. 流量配置（服务中心 → 流量配置）

每条**流量策略**绑定一个稳定对外入口（path / hostname），把入站请求按权重分发到当前租户下多个**在线服务后端**，支撑灰度发布、加权切分与蓝绿式全量切换。底层加权路由由 compute 派生（`(native,*)` → Envoy `HTTPRoute` 加权 `backendRefs`；`kserve` → `InferenceService` canary）。建议成员服务建为内部服务（关闭自身 route）；一个在线服务至多被一条活跃策略引用。编排见 [platform.md §4.8](../system_design/platform/backend.md#48-流量配置编排)。

### 10.1 列表页

```
Page Head:  流量配置。                                [+ 新建策略]
            为在线服务编排多版本流量：加权切分、灰度放量与蓝绿切换。
Filters:  🔍 名称 | 模式 ▾ | 状态 ▾ | 重置

│ 名称 │ 模式 │ 状态 │ 后端(流量分布) │ 访问地址 │ 操作 │
│ rt-chat │ 灰度 │ ◐ 灰度中 │ v1 ▰▰▰▰▰▰▰▰▱ 90 / v2 ▰ 10 │ /services/team-a/cha… │ 详情 禁用 │
```

模式（加权 / 灰度 pill）· 状态（§10.4）· 后端（成员 + 权重 mini bar）· 访问地址 · 操作（详情 / 启用 / 禁用 / 删除）；调流量在详情页执行。

### 10.2 详情页 Tab

头部：back-nav · 名称（mono）· 状态徽章 · 模式徽章 · 描述 · `[删除]`。Tabs：`[概览] [流量配置] [监控] [事件]`。

- **概览** — KV grid：名称 · 显示名 · 描述 · 模式 · 对外入口（path / hostname，**创建后不可变**）· 后端数 · 创建人 / 时间。
- **流量配置** — 后端表：在线服务（mono link → §8）· 角色（稳定 / 灰度；加权模式为成员）· 目标权重 · 实际流量占比 · 后端状态（复用 §6.4 徽章）。灰度模式顶部带百分比 slider + `[提升] [回滚]`；加权模式各行可编辑权重（实时 `Σ=100` 校验 + `[应用权重]`）。
- **监控** — 按后端分组对比的时序（QPS / 延迟 p95 / 错误率）；灰度模式叠加稳定 vs 灰度健康对比辅助放量。
- **事件** — 策略与灰度操作流水（权重调整 / 提升 / 回滚 / 后端就绪 / 失联）。

### 10.3 创建 / 操作

- **创建表单**：`name`（不可变）· 显示名 / 描述 · **模式**（加权 / 灰度，创建后不可变）· **对外入口**（path 留空自动生成 · hostname，创建后不可变）· **后端**：加权 = N 个服务各设权重（Σ=100）；灰度 = 1 稳定 + 1 灰度 + 初始灰度百分比。后端下拉只列本租户 `Ready` 且未被占用的在线服务。
- **操作**：调流量（加权改权重 / 灰度拖 slider）· 提升（灰度置 100 并升为新稳定基线）· 回滚（灰度归 0）· 蓝绿切换（加权模式下某后端置 100、其余置 0）。
- **模式映射**：UI 的`加权` / `灰度`对应 compute 的 `weighted` / `canary`；`bluegreen` 不作独立创建模式，蓝绿在加权模式下全量切换实现。

### 10.4 状态展示规则

UI 标签是 compute `MLTrafficPolicy` phase 的展示映射（见 [compute-service.md §3](../system_design/system/compute-service.md#3-核心模型)）：

| compute phase | UI 标签 | 视觉 |
| --- | --- | --- |
| `Ready`（全量分发） | 生效中 | ● 绿 |
| `Ready`（灰度 ∈ (0,100)） | 灰度中 | ◐ 橙 |
| `Pending` / `Degraded` | 未就绪 | ○ 灰描边 |
| `Failed` | 失败 | ▲ 红 |
| Creating / Deleting | 同名 | 灰 + spinner |

---

## 11. 实验（训练中心 → 实验）

训练特化任务，沿用 §7 自定义任务的 **Job → Run** 两级模型与全部列表 / 详情 / 表单结构，差异如下。隶属当前租户。编排见 [platform.md §4.9](../system_design/platform/backend.md#49-实验编排)。

| 维度 | 相对自定义任务（§7）的差异 |
| --- | --- |
| 路径 | `/experiments` · `/{name}` · `/{name}/runs/{run}` |
| 列表 | 同 §7.1，按钮为 `[+ 新建实验]`，行操作含运行 / 详情 / 编辑 |
| 表单 | 命令支持**超参模板变量**（如 `{{lr}}`），触发运行时可覆盖；其余字段同 §7.4 |
| 详情头部 | 额外 `[TensorBoard]` 动作；Tabs `[实验信息] [运行历史(Runs)]` |
| 指标对比 | 经 **TensorBoard** 查看各 Run 训练指标与多 Run 对比（HParams / Scalars），UI 不自建对比视图 |
| 登记模型 | Run 详情可一键把 checkpoint 登记为模型版本（复用 §9 上传流程） |

Run 状态与日志 / 实例 / 事件均同 §7.3 / §7.5。

---

## 12. 规划中（数据集 / 评估）

数据集与评估为规划中能力，本期无菜单入口与页面，仅在此登记定位：

- **数据集** — 资产中心的第三类制品（S3 / RustFS 后端），供任务 / 实验按版本挂载；当前任务以内联数据卷（PVC）承载数据。系统设计已保留底层 `dataset` kind（[artifact-hub.md §4](../system_design/system/artifact-hub.md#4-核心功能)）。
- **评估** — 沿用 Job → Run 两级结构的训练特化任务：选模型版本 + 评测数据集 + 指标，运行后看分数与对比报告。

落地后复用 §6.0 资源选择链与 §7 的两级模型，UI 字段定稿后补充本节。

---

## 13. 相关引用

- [platform.md](../system_design/platform/backend.md) — 后端编排、跨服务调用、PG schema
- [auth.md](../system_design/platform/auth.md) — RBAC、JWT、数据面接入
- [openapi/platform.yaml](../openapi/platform.yaml) — REST 字段契约
- [high_level_design.md](../system_design/high_level_design.md) — 系统概念与组件关系
- [compute-service.md](../system_design/system/compute-service.md) · [cluster-manager.md](../system_design/system/cluster-manager.md) · [artifact-hub.md](../system_design/system/artifact-hub.md) — 字段权威
