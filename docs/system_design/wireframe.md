# AxisML Platform UI 设计

## 1. 概述

本文是 Platform 前端 UI 的设计文档,集中描述页面结构、菜单与导航、列表页字段、详情页 Tab、创建/编辑表单字段、状态展示规则与权限可见性,作为前端开发与产品视觉评审的唯一对齐入口。

- **后端业务编排** (跨服务调用顺序、写入路径、一致性策略、PG schema) 见 [components/platform.md](components/platform.md)。
- **用户认证与角色矩阵** (RBAC 完整定义、JWT 颁发、IdentityProvider 接口) 见 [auth.md](auth.md)。
- **REST API 字段契约** 见 [apis/platform.yaml](apis/platform.yaml)。
- **Dashboard / 服务指标数据来源** 见 [monitoring.md](monitoring.md)。
- **整体系统概念** (Tenant / ResourcePool / Job / Service / Artifact) 见 [overview.md](overview.md)。

---

## 2. 菜单与导航

### 2.1 全局菜单结构

| 一级菜单 | 二级菜单 | 本文章节 | 设计状态 |
| --- | --- | --- | :---: |
| Dashboard | — | [§3](#3-dashboard) | ✅ |
| 应用中心 | 智能体 / Skills / MCP | [§12.2](#122-待补-ui-设计-横切) (TBD) | TBD |
| 训练 & 推理 | 工作区 | [§4](#4-工作区训练--推理--工作区) | ✅ |
| 训练 & 推理 | 计算任务 | [§5](#5-计算任务训练--推理--计算任务) | ✅ |
| 训练 & 推理 | 在线服务 | [§6](#6-在线服务训练--推理--在线服务) | ✅ |
| 制品中心 | 模型 / 镜像 / 数据集 | [§7](#7-制品中心-模型--镜像--数据集) | ✅ |
| 系统管理 | 租户管理 (含配额 / 成员 Tab) | [§8](#8-租户管理系统管理--租户) | ✅ |
| 系统管理 | 资源池管理 (含资源单元) | [§9](#9-资源池与资源单元系统管理--资源池) | ✅ |
| 系统管理 | 数据卷管理 | [§10](#10-数据卷-tbd) (TBD) | TBD |
| 系统管理 | 用户与角色 | [§12](#12-后续设计) (TBD) | TBD |

二级菜单命名与上表保持一致;详细业务能力矩阵 (含横切的认证 / RBAC) 见 [components/platform.md §4 核心功能](components/platform.md#4-核心功能)。

### 2.2 全局布局

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ AxisML  │ 工作台 ▾  │ 训练&推理 ▾ │ 制品中心 ▾ │ 系统管理 ▾ │        zhang ▾ │
├─────────┴───────────┴──────────────┴────────────┴───────────┴───────────────┤
│  ┌────────────────────┐  ┌──────────────────────────────────────────────┐   │
│  │  侧边导航 (二级)    │  │                  主内容区                     │   │
│  │  • 工作区           │  │                                              │   │
│  │  • 计算任务         │  │   ─ 面包屑 ─                                  │   │
│  │  • 在线服务         │  │                                              │   │
│  │  ─────────────      │  │   ─ 列表 / 详情 / 表单 ─                       │   │
│  │  租户切换 ▾         │  │                                              │   │
│  └────────────────────┘  └──────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
```

- **顶栏**:一级菜单 + 用户菜单 (账号 / 角色 / 登出)。
- **左栏**:二级菜单 + 当前活跃租户切换器 (`system-admin` 跨租户切换时)。
- **主区**:列表 / 详情 / 表单三态切换;详情页统一使用 Tab 容器,Tab 名与下文章节一致。

### 2.3 通用元素约定

- **面包屑**:`一级 / 二级 / [资源名]`;详情页第三段可点击回到列表。
- **空态**:每个列表页提供「创建第一个 X」CTA + 引导链接。
- **错误条**:跨租户并行 LIST 部分失败 → 列表顶部黄条「N 个租户暂时不可达,显示其余结果」(对应 `partial=true`)。
- **二次确认**:删除 / 取消 / 强制操作弹窗显示「前置阻断信息」(如使用此资源单元的活跃 Job/Service 计数)。
- **状态徽章**:见各章 `状态展示规则` 子节统一色板与图标。

---

## 3. Dashboard

### 3.1 页面入口

- 路径:`/dashboard` (用户登录后默认页)。
- 权限:所有已登录用户可访问;内容按角色裁剪。

### 3.2 卡片与图表

| 区块 | 数据来源 | 角色裁剪 |
| --- | --- | --- |
| 我可见的租户数 | Platform `GET /api/v1/tenants` 计数 | `system-admin` 看全集群;其他角色看 `user_tenant_roles` |
| 活跃计算任务 (Running) | `GET /api/v1/jobs?status=Running` 并行聚合 | 按角色裁剪 |
| 在线服务 (Ready) | `GET /api/v1/services?status=Ready` 并行聚合 | 按角色裁剪 |
| 工作区运行中 (Running) | `GET /api/v1/workspaces?status=Running` | `@owner` / `tenant-admin@self` |
| 配额使用率 Top N | `GET /api/v1/tenants/{name}/quotas` + `compute.GetQuotaUsage` | 仅展示可见租户 |
| 最近事件流 | Platform `audit_logs` (按当前用户可见 target 过滤) | 按角色裁剪 |
| GPU 利用率趋势 | Prometheus `DCGM_FI_DEV_GPU_UTIL` (见 [monitoring.md](monitoring.md)) | `system-admin` only |

### 3.3 ASCII 占位

```
┌──────────────────────────────────────────────────────────────────────┐
│ Dashboard                                                            │
├──────────────────────────────────────────────────────────────────────┤
│ ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐                          │
│ │租户 12 │ │任务 47 │ │服务 8  │ │工作区 5│                          │
│ └────────┘ └────────┘ └────────┘ └────────┘                          │
│                                                                      │
│ ┌─────────────────────────┐ ┌──────────────────────────────────┐     │
│ │  配额使用率 Top 5       │ │  GPU 利用率 (24h)                │     │
│ │  ▇▇▇▇▇▇▇▇▇░ tenant-a    │ │  ╱╲   ╱╲╱╲                       │     │
│ │  ▇▇▇▇▇▇░░░░ tenant-b    │ │ ╱  ╲_╱    ╲_╱╲                   │     │
│ └─────────────────────────┘ └──────────────────────────────────┘     │
│                                                                      │
│ 最近事件                                                              │
│ • 10:32  zhang created job tenant-a/train-llm-7b                     │
│ • 10:28  admin scaled service svc-id-xxx to 3 replicas               │
└──────────────────────────────────────────────────────────────────────┘
```

---

## 4. 工作区 (训练 & 推理 → 工作区)

### 4.1 页面入口

| 入口 | 路径 | 权限 |
| --- | --- | --- |
| 列表 | `/workspaces` | 已登录 |
| 创建 | `/workspaces/new` | `RequireTenantRole("user", "<tenantId>")` |
| 详情 | `/workspaces/{id}` | `@owner` 或所属租户 `tenant-admin+`;`system-admin` 短路 |

### 4.2 列表页

| 列 | 来源字段 | 说明 |
| --- | --- | --- |
| 显示名 | compute `services.displayName` | 行点击进详情 |
| 镜像 | compute `services.spec.roles[0].template.image` | 截断显示;hover 显示完整 |
| 资源池 · 单元 | compute `services.spec.scheduling.resourcePool` + `resourceUnit` | 显示池名 / 单元名 |
| Owner | `services.owner` | 普通用户列表已按 `owner=current_user` 过滤 |
| 创建时间 | `services.createdAt` | 相对时间 + tooltip 绝对时间 |
| 状态 | 派生 (见 [§4.5](#45-状态展示规则)) | 徽章 |
| 操作 | — | 启动 / 停止 / 打开 / 详情 / 删除 |

**过滤**:租户 (admin 跨租户) / 状态 / 关键字 (显示名 / 镜像)。
**排序**:创建时间 (默认倒序) / 显示名。
**可见性**:`system-admin` 跨所有 compute namespace 并行;`tenant-admin@self` 限可见租户;普通用户 compute 端 `owner=` 过滤下推。

### 4.3 创建表单

| 字段 | 写入位置 | 校验 |
| --- | --- | --- |
| 显示名 | `services.displayName` | 非空 |
| 描述 | `services.description` | 可空 |
| 租户 | 决定 namespace | 用户在该租户至少 `user` 角色 |
| 镜像 | `spec.roles[0].template.image` | 必须能 `artifacts.Resolve` |
| 容器端口 | `spec.roles[0].template.ports[0].containerPort` | ∈ [1, 65535] |
| 启动命令 (覆盖) | `spec.roles[0].template.command` / `args` | 可空 |
| 环境变量 | `spec.roles[0].template.env` | key 唯一 |
| 资源池 | `spec.scheduling.resourcePool` (uuid) | 下拉,所有用户可读 |
| 资源单元 | `spec.scheduling.resourceUnit` | 必须 ∈ 所选 ResourcePool |
| 配额 | `spec.scheduling.quota` (name) | 必须为当前租户在所选池下可用 |
| PVC 大小 | Platform 直管 PVC,`spec.resources.requests.storage` | 默认 20 GiB |

要点:
- **不引入 IDE 模板抽象**——镜像本身决定 (jupyter / VSCode / 自定义)。镜像目录由系统管理员通过 Artifacts 镜像中心维护「常用基础镜像」。
- 容器内 web server 必须支持子路径 base-url;Platform 注入 `AXISML_WORKSPACE_BASE_URL=/workspaces/<tenant>/<service.name>/`,约定 workspace 镜像 entrypoint 读取并自配 base-url。
- 当前不暴露 `backend.config` / `priorityClass` / 自定义 `nodeSelector` / `volumes`。

对应 REST 端点见 [apis/platform.yaml](apis/platform.yaml) `Workspaces` tag。

### 4.4 详情页 Tab

| Tab | 内容 | 操作 |
| --- | --- | --- |
| **概览** | 基本信息 + 当前状态卡片 + PVC 信息 + 时间线 | 启动 / 停止 / 打开 / 删除 / 编辑 `displayName` `description` |
| **访问** | Access URL + JWT 用法 + `kubectl port-forward` 命令模板 | 「获取一次性 Access JWT」按钮 (调 `GET .../access`) |
| **事件** | (后续工作) 待 compute service `/events` 扩展 | — |
| **日志** | (后续工作) 待 compute service `/logs` 扩展 | — |
| **审计** | 入口保留 (TBD) | — |

要点:
- **PATCH** 仅可改 `displayName` / `description`,其他字段不可变 → 表单字段置灰。
- **start / stop** 等价于 `compute.ScaleService(replicas: 1|0)`;`Deleted` / `Deleting` 状态拒绝 → toast `workspace-deleted`。
- **删除**对话框可选 `保留 PVC` (默认不保留);删除前显示 PVC 名 `axisml-ws-<id 前 8 字符>-data`。
- **访问** Tab 的 JWT `aud=axisml-workspace`,TTL 由 `--workspace-access-jwt-ttl` 控制 (上限 24h)。

### 4.5 状态展示规则

派生自 compute `services.status` + `replicas`:

| 条件 | 显示状态 | 视觉 |
| --- | --- | --- |
| `services.status == Creating` | Creating | 蓝色徽章 + spinner |
| `services.status == Deleting` | Deleting | 灰色徽章 + spinner |
| `services.status == Deleted` | Deleted | 灰色徽章 |
| `replicas == 0` 且 status ∉ {Creating, Deleting, Deleted} | Stopped | 灰色实心 |
| `replicas > 0 && services.status == Pending` | Starting | 蓝色徽章 + spinner |
| `replicas > 0 && services.status == Ready` | Running | 绿色实心 |
| `replicas > 0 && services.status == Degraded` | Degraded | 黄色 |
| `replicas > 0 && services.status == Failed` | Failed | 红色 |

`Failed` / `Degraded` 为非终态,operator 自愈后下次 GET 自然回到 `Running`。

### 4.6 权限可见性

| 操作 | system-admin | tenant-admin | user |
| --- | :---: | :---: | :---: |
| 列出我可见的工作区 | 全集群 | 本租户全部 | `@owner` |
| 创建 | ✅ | ✅ (本租户) | ✅ (本租户) |
| 打开 / 启停 / 编辑元数据 / 删除 | ✅ | `@self` | `@owner` |

完整 RBAC 矩阵见 [auth.md §3](auth.md#3-rbac-角色)。

---

## 5. 计算任务 (训练 & 推理 → 计算任务)

### 5.1 页面入口

| 入口 | 路径 | 权限 |
| --- | --- | --- |
| 全集群列表 | `/jobs` | `system-admin` |
| 租户视图列表 | `/tenants/{tenant}/jobs` | 该租户 `user+` |
| 创建 | `/tenants/{tenant}/jobs/new` | 该租户 `user+` |
| 详情 | `/tenants/{tenant}/jobs/{name}` | `@owner` 或本租户 `tenant-admin+` |

寻址采用 `(tenant_name, job_name)` 二元组,URL 自带 RBAC 边界。

### 5.2 列表页

| 列 | 来源字段 | 说明 |
| --- | --- | --- |
| 任务名 | compute `jobs.name` | 行点击进详情 |
| 镜像 | `spec.roles[0].template.image` | 截断 + hover |
| 后端 | `spec.backend.name` / `engine` | 如 `native / job`、`kubeflow-trainer / pytorchjob` |
| 资源池 · 单元 | `spec.scheduling.resourcePool` + `resourceUnit` | — |
| 状态 | `jobs.status` | 见 [§5.5](#55-状态展示规则) |
| Owner | `jobs.owner` | 普通用户已下推过滤 |
| 开始 - 结束时间 | `jobs.startedAt` / `completedAt` | 相对时间 |
| 操作 | — | 取消 / 详情 / 删除 / 注册为模型 / 重新提交 |

**过滤**:状态 / Owner (admin) / backend.name / backend.engine / 关键字。`status`/`limit`/`continue` 下推 compute;其余 Platform 内存二次筛选。
**可见性**:`system-admin` 跨所有 compute namespace 并行;`tenant-admin@self` 限本租户;普通用户 `@owner`。

### 5.3 创建表单

| 字段 | 写入位置 | 校验 |
| --- | --- | --- |
| 任务名 | `jobs.name` | DNS-1123;同 namespace 唯一 |
| 显示名 / 描述 | `jobs.displayName` / `description` | 可空 |
| 租户 | 决定 namespace | 当前用户 `user+` |
| 后端 · 引擎 | `spec.backend.name` / `engine` | 见后端→Role 映射 |
| 资源池 / 资源单元 / 配额 | `spec.scheduling.*` | 单元 ∈ 池;配额属于该租户在池下的 quota |
| Roles | `spec.roles[]` | 按后端动态渲染 |
| runPolicy | `spec.runPolicy.{activeDeadlineSeconds, ttlSecondsAfterFinished, backoffLimit}` | 数字 |
| 输出声明 | `spec.outputs[]` | 可空;每条:`name` (DNS-1123) / `kind` (当前仅 `model`) / `volumeName` (须为 PVC 类型卷) / `sourcePath` |

**按后端动态渲染 Roles**:

| backend / engine | role 集合 |
| --- | --- |
| `(native, job)` | `worker` |
| `(native, podgroup)` | `worker` (gang) |
| `(kubeflow-trainer, pytorchjob)` | `master + worker` |
| `(kubeflow-trainer, tfjob)` | `chief / worker / ps / evaluator` |
| `(kubeflow-trainer, mpijob)` | `launcher + worker` |
| `(custom, *)` | 由 `backend.config` 决定 |

每个 role 块含:副本数 / 镜像 / 启动命令 / 参数 / 环境变量 / 重启策略。所有 role 共享同一 ResourceUnit。

> 当前表单不开放 `backend.config` / `priorityClass` / 自定义 `nodeSelector`。`volumes` 入口仅在「输出声明」启用时可见——表单提供一行 PVC 选择器(选已存在的 PVC,或新建 + 大小);该 PVC 同时挂到 `roles[*].template.volumes[]` 并被 `outputs[].volumeName` 引用。

**输出声明的取舍**:不强制(典型「调参跑分」任务无需声明);填写后 §5.4「注册为模型」按钮可直接选 dropdown,跳过手动 `outputPath` 输入,见 [components/platform.md §4.5.3](components/platform.md#453-register-from-job计算任务--模型)。

对应 REST 端点见 [apis/platform.yaml](apis/platform.yaml) `Jobs` tag。

### 5.4 详情页 Tab

| Tab | 内容 | 操作 |
| --- | --- | --- |
| **概览** | `compute.GetJob` 字段全只读 | 取消 / 删除 / 注册为模型 / 重新提交 / 复制 YAML |
| **副本** | `compute.GetJobReplicas` 透传:每副本 pod / 容器 / 状态 / nodeName | 渲染 `kubectl exec` / `kubectl logs` 命令模板 (当前不提供 in-browser shell) |
| **事件** | `compute.GetJobEvents` 透传:Kubernetes Events | — |
| **日志** | `compute.GetJobLogs` 透传 | 切换 replica / pod / container / tailLines / follow;`follow=true` 走 SSE;Pod 已 GC 时下游 410 → UI 展示「日志已过期」 |
| **审计** | 入口保留 (TBD) | — |

要点:
- Job spec **不可变**——UI 不提供「编辑任务」入口;改参数 = 「重新提交」 = 反填表单新建。
- **取消**:`compute.CancelJob` 透传,提示 `'user cancelled'`;状态合法性由下游 4xx 反馈。
- **注册为模型**:仅在 `phase == Succeeded` 启用。点击后弹出 modal:
  1. 若 Job 已声明 `spec.outputs[kind=model]`(读 `job.spec.outputs`),modal 顶部下拉列示;选中后 `outputName` 自动填入,源 PVC + sourcePath 只读展示。
  2. 否则要求手动填 `volumeName` + `outputPath`(PVC 名 dropdown 列出 `job.spec.roles[*].template.volumes[]` 中的 PVC 卷)。
  3. 填模型字段(`modelName` / `modelVersion` / `spec.framework` / `spec.format` / `displayName` / `description`);`spec.trainingDatasetRef` 由后端探测 `job.spec.roles[*].template.env` 是否含 `AXISML_DATASET_URI` 自动反填,可改可空。
  4. 提交 `POST /tenants/{t}/jobs/{name}/register-model` → 返回 `{artifact, upload, provenance}`。
  5. 复用 [§7.2.2 上传指引对话框](#722-上传表单通用字段--两阶段交互),额外渲染顶部「来源:任务 `<tenant>/<name>` 输出 `<outputName 或 ad-hoc>`(PVC `<provenance.pvc>`,路径 `<provenance.sourcePath>`)」;字节上传走 cli (不在本设计内,见 [§4.5.3](components/platform.md#453-register-from-job计算任务--模型) 末段)。
  6. 后端契约见 [components/platform.md §4.5.3](components/platform.md#453-register-from-job计算任务--模型)。

### 5.5 状态展示规则

直接展示 compute `jobs.status`:

| 状态 | 视觉 |
| --- | --- |
| `Pending` | 灰色徽章 + spinner |
| `Creating` | 蓝色徽章 + spinner |
| `Running` | 绿色实心 |
| `Succeeded` | 绿色边框 + ✓ |
| `Failed` | 红色 |
| `Cancelled` | 灰色 |
| `Cancelling` | 灰色 + spinner |
| `Deleting` / `Deleted` | 灰色 + spinner |

### 5.6 权限可见性

| 操作 | system-admin | tenant-admin | user |
| --- | :---: | :---: | :---: |
| 全集群列表 `/jobs` | ✅ | ✗ | ✗ |
| 租户列表 `/tenants/{t}/jobs` | ✅ | ✅ | `@owner` 过滤 |
| 创建 | ✅ | ✅ | ✅ |
| 取消 / 删除 / 注册为模型 / 看副本-事件-日志 | ✅ | `@self` | `@owner` |

完整矩阵见 [auth.md §3](auth.md#3-rbac-角色)。

---

## 6. 在线服务 (训练 & 推理 → 在线服务)

### 6.1 页面入口

| 入口 | 路径 | 权限 |
| --- | --- | --- |
| 列表 | `/services` | 已登录;按角色裁剪 |
| 创建 | `/services/new` | `RequireTenantRole("user", "<tenantId>")` |
| 详情 | `/services/{id}` | `@owner` 或所属租户 `tenant-admin+` |

寻址采用 `services.id` (uuid);URL 与工作区共享 id-based + kind 过滤端点,服务端通过 `kind='service'` 防误删工作区。

### 6.2 列表页

| 列 | 来源字段 | 说明 |
| --- | --- | --- |
| 服务名 | compute `services.name` | 行点击进详情 |
| 模型 (`name@version`) | `spec.modelRef.name` + `version` | — |
| 镜像 | `spec.roles[0].template.image` | — |
| 后端 | `spec.backend.name` / `engine` | — |
| 资源池 · 单元 | `spec.scheduling.*` | — |
| 副本 (`ready/total`) | `status.readyReplicas` / `spec.roles[0].replicas` | `2/3` 形式 |
| 状态 | `services.status` | 见 [§6.5](#65-状态展示规则) |
| Owner | `services.owner` | — |
| 入口 | `route.enabled` ? 渲染 `https://<gateway><route.path>` : `services.endpoint` | 一键复制 |
| 创建时间 | `services.createdAt` | — |
| 操作 | — | 扩缩容 / 停 · 启 / 详情 / 克隆为新版本 / 删除 |

**过滤**:租户 (admin) / 状态 / Owner (admin) / backend.name / engine / 模型名 / 关键字。`tenantName`/`status`/`owner`/`limit`/`continue` 下推 compute;`q`/`backendName`/`backendEngine`/`modelName` 内存二次筛选。

### 6.3 创建表单

| 字段 | 写入位置 | 校验 |
| --- | --- | --- |
| 服务名 | `services.name` | DNS-1123;同 namespace 内 service / workspace 不重名 |
| 显示名 / 描述 | `services.displayName` / `description` | — |
| 租户 | 决定 namespace | 当前用户 `user+` |
| 模型 (`name@version`) | `spec.modelRef.{name, version}` | 必须 `artifacts.Resolve` 且 `status=Ready` |
| 后端 · 引擎 | `spec.backend.name` / `engine` | 见后端→Role 映射 |
| 镜像 | `spec.roles[0].template.image` | 必须 `artifacts.Resolve` |
| 容器端口 / 启动命令 / env | `spec.roles[0].template.{ports, command, env}` | — |
| 资源池 / 单元 / 配额 | `spec.scheduling.*` | 同 §5.3 |
| 副本数 | `spec.roles[0].replicas` | ≥ 0 |
| `progressDeadlineSeconds` | `spec.progressDeadlineSeconds` | 数字 |
| 路由 - 启用 | `spec.route.enabled` | 布尔 |
| 路由 - 路径 | `spec.route.path` | 留空 → 自动 `/services/<tenant>/<name>/` |
| 路由 - hostname | `spec.route.hostnames[]` | 可空 |
| 路由 - 鉴权 | `spec.route.auth.{type, secretRef}` | `apiKey` 时 secret 必须存在于 namespace |
| 路由 - rateLimit / timeout | `spec.route.{rateLimit, timeout}` | — |

**按后端动态渲染 Roles**:

| backend / engine | role 集合 |
| --- | --- |
| `(native, deployment)` | `predictor` (replicas≥0) |
| `(native, statefulset)` | `predictor` (副本身份稳定) |
| `(kserve, inference)` | `predictor` (`backend.config.runtime` 决定 runtime) |
| `(kserve, llminference)` | `prefill + decode + router` (PD 分离) |
| `(custom, *)` | `backend.config` 决定 |

> 当前表单单 role 单端口,不开放 `backend.config` / `priorityClass` / `volumes`;模型工件通过 `spec.modelRef` 由 handler 解析注入 `AXISML_MODEL_URI` env。各 backend 的就位状态见 [components/compute-operator.md §9 后续工作](components/compute-operator.md#9-后续工作)。

#### 6.3.1 版本切换 / 灰度发布

MLService `spec.modelRef` 创建后不可变。「克隆为新版本」是前端语法糖:反填创建表单 → 用户改 `modelRef.version` → 提交新 service → 在外部流量层切流量 → 旧 service 停服 + 删除。当前**不**接管 weighted route / canary UI (TBD)。

对应 REST 端点见 [apis/platform.yaml](apis/platform.yaml) `Services` tag。

### 6.4 详情页 Tab

| Tab | 内容 | 操作 |
| --- | --- | --- |
| **概览** | 基本信息 + 状态卡片 + 副本概览 + 路由概览 + 时间线 | 扩缩容 / 停 / 启 / 编辑 `displayName` `description` / 克隆 / 删除 |
| **访问** | Access URL (`https://<gateway><route.path>` 或 fallback `services.endpoint`) + 鉴权说明 (按 `auth.type` 分支) + 调用示例 (curl / python / grpcurl) + `kubectl port-forward` 命令模板 | `auth.type=jwt` 时显示「获取一次性 JWT」按钮 → 调 `GET .../access` |
| **事件** | 入口保留 (TBD) | — |
| **日志** | 入口保留 (TBD) | — |
| **指标** | 来自 Prometheus 实时查询 (PromQL 模板按 backend 选择) | 见下表 |
| **审计** | 入口保留 (TBD) | — |

**指标 Tab 内容**:

| 指标 | 维度 | 说明 |
| --- | --- | --- |
| `request_rate` | rps | — |
| `latency` | p50 / p95 / p99 (默认 p95) | `percentile` 仅 latency 有效 |
| `error_rate` | 4xx / 5xx 分离 | — |
| `cpu_util` | — | — |
| `mem_util` | — | — |
| `gpu_util` | — | — |

- 时间窗口选择:5m / 15m / 1h / 6h / 24h。
- 自动刷新:15 秒。
- LLM 专项指标 (tokens/sec / TTFT / TBT / KV cache / batch utilization) 占位 (TBD)。
- 查询失败展示「监控数据暂不可用」(对应后端 `502 upstream-failure`)。

要点:
- **PATCH** 仅可改 `displayName` / `description`;其他字段一律不可变 → 走「克隆为新版本」路径。
- **scale / start / stop**:`/scale` body `{"replicas": <int≥0>}`;`/start` = scale 到「上一次 >0 的 replicas」 (查 `audit_logs`),缺失则 fallback 1;`/stop` = scale 0。
- **删除**派生 K8s Service / HTTPRoute / SecurityPolicy / BackendTrafficPolicy 由 ownerReference 级联清理。

### 6.5 状态展示规则

直接展示 compute `services.status`:

| 状态 | 视觉 |
| --- | --- |
| `Creating` | 蓝色徽章 + spinner |
| `Pending` | 灰色 + spinner |
| `Ready` | 绿色实心 (含 `readyReplicas == replicas`) |
| `Degraded` | 黄色 (部分副本 ready) |
| `Failed` | 红色 |
| `Deleting` / `Deleted` | 灰色 + spinner |

副本计数 `ready/total` 与状态徽章并列显示;`route.enabled=false` 时入口列显示「内网 (`services.endpoint`)」。

### 6.6 权限可见性

| 操作 | system-admin | tenant-admin | user |
| --- | :---: | :---: | :---: |
| 列出 | 全集群 | 本租户 | `@owner` |
| 创建 | ✅ | ✅ | ✅ |
| 扩缩容 / 停启 / 删除 / 克隆 / 看指标 | ✅ | `@self` | `@owner` |
| 获取 access JWT (`auth.type=jwt`) | ✅ | `@self` | `@owner` |

---

## 7. 制品中心 (模型 / 镜像 / 数据集)

二级菜单与 Artifacts 服务的三类 `kind` 一一对应:`model` / `image` / `dataset`。三类共用同一套上传 / 列表 / 详情 / 删除骨架,Kind 专属字段在 [§7.3](#73-模型-kindmodel) – [§7.5](#75-数据集-kinddataset) 分述。

底层服务字段与状态机权威定义见 [components/artifacts.md](components/artifacts.md);Platform 透传契约见 [components/platform.md §4.5 制品编排](components/platform.md#45-制品编排)。

### 7.1 页面入口

| 入口 | 路径 | 权限 |
| --- | --- | --- |
| 模型列表 | `/models` | 已登录;按角色裁剪 |
| 模型创建 | `/models/new` | `RequireTenantRole("user", "<tenantId>")` |
| 模型详情 | `/models/{tenant}/{name}/{version}` | `@owner` 或所属租户 `tenant-admin+` |
| 镜像列表 / 创建 / 详情 | `/images` `/images/new` `/images/{tenant}/{name}/{version}` | 同上 |
| 数据集列表 / 创建 / 详情 | `/datasets` `/datasets/new` `/datasets/{tenant}/{name}/{version}` | 同上 |

寻址采用 `(tenant, name, version)` 三元组——和下游 artifacts 同形——前端 url path 直接拼下游寻址 tuple,Platform 无 id ↔ tuple 反查。tuple 永不复用(软删后保留),稳定性等价于 uuid。租户视图列表走 `/api/v1/tenants/{tenant}/{kind-plural}`(operationId `listTenant{Models,Images,Datasets}`),角色裁剪同 §4 / §5 / §6。

### 7.2 通用模式

三个 Kind 共骨架,本节描述列表 / 上传 / 详情的统一交互;Kind 专属字段在后续小节细化。

#### 7.2.1 列表页(通用列)

| 列 | 来源字段 | 说明 |
| --- | --- | --- |
| Name @ Version | `artifact.name` + `artifact.version` | 行点击进详情;`name@version` 一键复制 |
| 显示名 | `artifact.displayName` | 可空;空时回退 `name` |
| 租户 | 由 `artifact.namespace` 反查 | `system-admin` 跨租户时展示 |
| Owner | `artifact.owner` | 普通用户已下推 `owner=` 过滤 |
| 大小 | `artifact.sizeBytes` | 人读单位 (MiB / GiB);未 `Ready` 显示 `—` |
| 状态 | `artifact.status` | 徽章,见 [§7.6](#76-状态展示规则) |
| 上传时间 | `artifact.readyAt` ?? `artifact.createdAt` | 相对时间 + tooltip 绝对时间 |
| Kind 专属列 | — | 见 §7.3 – §7.5 各小节 |
| 操作 | — | 详情 / 复制 ref / 上传新版本 / 删除 |

**过滤**:租户 (admin) / 状态 / Owner (admin) / 关键字 (`name` 前缀 / `displayName` 模糊) + Kind 专属过滤项。`tenantName` / `status` / `owner` / `limit` / `continue` 下推 artifacts;`q` 与 Kind 专属过滤在 Platform 内存二次筛选。
**排序**:上传时间 (默认倒序) / 名称 / 大小。
**可见性**:`system-admin` 跨所有 artifact namespace 并行 → 部分失败 `partial=true` 黄条;`tenant-admin@self` 限可见租户;普通用户 `@owner` 过滤下推。

下面是模型列表的典型布局,镜像 / 数据集结构相同,仅 Kind 专属列(虚线包围)替换:

```
┌ 模型 ───────────────────────────────────────────────────────────────────── [+ 上传模型] ┐
│ 租户▾ all   状态▾ Ready  Owner▾ all   q  llama        [↻]            12 项 (2 个租户) │
│ ⚠ 1 个租户暂时不可达,显示其余结果                                                     │
├─────────────────────────────────────────────────────────────────────────────────────────┤
│  Name @ Version     ┊Framework  Format  Params┊  租户   Owner  大小    状态   上传   ⋯ │
│  llama-7b @ v3      ┊pytorch    safe…   7B    ┊  team-a  zhang 13.4GiB ● Ready 2h     ⋯│
│  llama-7b @ v2      ┊pytorch    safe…   7B    ┊  team-a  zhang 13.4GiB ● Ready 5d     ⋯│
│  qwen-2-7b @ v0.4   ┊pytorch    safe…   7B    ┊  team-a  li    14.1GiB ◐ Upload 2m    ⋯│
│  bge-large @ v1     ┊onnx       onnx    335M  ┊  team-b  wang  650MiB  ● Ready 12d    ⋯│
│  bert-base @ v1     ┊tensorflow tf2     110M  ┊  team-b  wang  420MiB  ✗ Failed 1d    ⋯│
│  …                                                                                      │
│                                                          [前页]  cursor=abc…  [后页]    │
└─────────────────────────────────────────────────────────────────────────────────────────┘
```

要点:
- 顶部黄条是 §5.3 跨租户合并的 `partial=true` 信号;
- 状态徽章:●=Ready 绿、◐=Uploading 蓝(spinner)、✗=Failed 红、灰=Deleting / Deleted;
- 行 ⋯ 菜单:详情 / 复制 ref / 上传新版本 / 编辑展示元数据 / 删除;
- `system-admin` 的「租户」列才出现,其它角色省略。

#### 7.2.2 上传表单(通用字段 + 两阶段交互)

**通用字段**:

| 字段 | 写入位置 | 校验 |
| --- | --- | --- |
| 租户 | 决定 `namespace` | 当前用户 `user+` |
| 制品名 (`name`) | `artifact.name` | DNS-1123 + OCI repo 字符集;同 `(ns, kind, name)` 多版本可复用 |
| 版本 (`version`) | `artifact.version` | OCI tag 字符集;同 `(ns, kind, name)` 下唯一,**软删后不可复用** |
| 显示名 / 描述 | `displayName` / `description` | 可空 |
| labels / annotations | 见 [database.md §1.6](database.md#16-扩展元数据-labels--annotations) | 大小约束 |
| Kind 专属 spec | `spec.*` | 见 §7.3 – §7.5 |

**两阶段上传交互**:

1. 用户填表 → 提交 → Platform 调 `initiate{Model,Image,Dataset}` → 返回 `{tenant, name, version, uri, uploadCredentials, expiresAt}`;artifact 行立刻入 `Uploading` 状态。
2. UI 弹出**上传指引对话框**,根据 `storageKind` 渲染两种通路:
   - **客户端工具通路**(默认,对所有 Kind 通用):展示 `(tenant, name, version, uri, uploadCredentials, expiresAt)` 凭证与一个可拷贝的客户端调用片段,由本地客户端工具 (`axisml-cli`,具体命令形态另文设计) 推送字节到 `uri`、再调 `complete` 提交 digest。
   - **浏览器直传通路**(仅 S3 Kind:`dataset` 且 size < 阈值):拖入文件,JS 用 STS 直 PUT 到 `prefix`;上传完成后前端直接调 `complete` 提交 digest。
3. 「上传中」状态:UI 每 5s 轮询 `GET /{kind-plural}/{tenant}/{name}/{version}` 直到 `Ready` / `Failed`,或用户离开页面 (后台轮询持续 ≤ 24h)。
4. `complete` 阶段:工具通路下由客户端触发;浏览器通路下由前端触发。Platform 透传 `digest`,artifacts 后端 HEAD 校验后置 `Ready`。

```
┌─ 上传模型 team-a/llama-7b @ v3 ───────────────────────────┐
│ 状态: Uploading (剩余 23h 54m,token 在 59m 后过期 ↻ 续签)│
│                                                          │
│ ❶ 用客户端工具推送字节(凭证已嵌入下方片段):              │
│ ┌──────────────────────────────────────────────────────┐ │
│ │ <axisml-cli 调用,形态另文设计>                       │ │
│ │ 凭证: <uploadCredentials 摘要>  expiresAt: 14:59     │ │
│ │ 目标 uri: zot…/namespaces/team-a-…/models/llama-7b:v3│ │
│ │ 📋 复制凭证 JSON                                     │ │
│ └──────────────────────────────────────────────────────┘ │
│                                                          │
│ ❷ 完成后将自动刷新为 Ready;若失败可点击「重新生成凭证」 │
│                                                          │
│ [取消并删除草稿]              [关闭(后台继续轮询)]        │
└──────────────────────────────────────────────────────────┘
```

要点:
- **续签**:Token 剩 < 5min 时「重新生成凭证」按钮调同一 `(ns, kind, name, version)` initiate,artifacts 端幂等返回原行新 token;不需删行重建。
- **取消草稿**:`Uploading` 行可直接 DELETE,立刻进 `Deleting`;未上传任何 bytes 时 GC 一轮即清。
- **失败重试**:`Failed` 行不可复活,需 DELETE 后另起新版本 (`v3` → `v3.1`)。
- **冲突反馈**:同 `(ns, kind, name, version)` 已存在 → `409 ArtifactAlreadyExists`,弹窗引导改 version。

#### 7.2.3 详情页 Tab(通用骨架)

| Tab | 内容 | 操作 |
| --- | --- | --- |
| **概览** | 基本信息(`name` / `version` / 租户 / `digest` / `sizeBytes` / `createdAt` / `readyAt` / `owner`)+ Kind 专属信息(§7.3 – §7.5)+ `ref` 一键复制(`<ns>/<kind>/<name>@<version>`) | 编辑 `displayName` / `description` / `labels` / `annotations` / 删除 / 上传新版本 / Kind 专属跳转按钮 |
| **版本** | `listArtifactVersions(ns, kind, name)`:同 `name` 下所有版本(含已删除),按 `createdAt` 倒序;行点击切换详情 | 「上传新版本」反填 spec |
| **后端** | `storageKind` / `uri` / `digest` / `auth_hint` 路径(由 `resolve?usage=inspect` 拼接,不下发明文) | 「获取下载凭证」按钮 → 调 `resolve?usage=download` 拉短期 OCI/S3 token |
| **引用方** | 入口保留 (TBD,详见 §12.3) | — |
| **审计** | 入口保留 (TBD) | — |

要点:
- **spec 不可变**:`Ready` 后 spec / digest 冻结 → 「编辑」入口仅暴露 `displayName` / `description` / `labels` / `annotations` 四项,直接调 `PATCH /api/v1/{kind-plural}/{tenant}/{name}/{version}` (`[updateModel]` / `[updateImage]` / `[updateDataset]`,见 [components/platform.md §4.5.1](components/platform.md#451-跨-kind-共骨架));改 spec = 走「上传新版本」反填表单。
- **删除**:软删 → `Deleting` → GC 清后端 → `Deleted`;`(ns, kind, name, version)` 四元组**永不复用**,删除后同 version 提交 → `409`。
- **下载凭证**:OCI Kind 返回 1h TTL 的 bearer token;S3 Kind 返回 prefix-scoped STS;前端用 `clipboard` API 复制,不持久化。

下面是模型详情页概览 Tab 的典型布局,镜像 / 数据集结构相同,Kind 专属区(虚线包围)替换为各自字段:

```
模型 / team-a / llama-7b @ v3                                [⋯ 操作 ▾]
─────────────────────────────────────────────────────────────────────
[概览] [版本] [后端] [引用方] [审计]
─────────────────────────────────────────────────────────────────────

  显示名         Llama 7B fine-tuned (sft-v3)      ✎
  描述           SFT on internal qa-2024 dataset    ✎
  ref            team-a/model/llama-7b@v3          📋
  租户 · ns      team-a · team-a-models
  Owner          zhang                  上传时间   2026-05-22 14:08 (2h 前)
  Status         ● Ready                Ready 时间 2026-05-22 14:31
  大小           13.4 GiB               Digest    sha256:9b74…3a3 📋

  ┌─ Kind 专属 (model) ───────────────────────────────────────────┐
  │ Framework    pytorch          Format        safetensors      │
  │ Task         text-generation  Params        7B               │
  │ 基模型       team-a/model/llama-7b@v1  →                     │
  │ 训练数据集   team-a/dataset/qa-2024@v3 →                     │
  │                                                              │
  │ Provenance   来源任务 team-a/train-llama-sft-v3              │
  │              输出 weights (PVC axisml-jobs-…-output)     →   │
  └──────────────────────────────────────────────────────────────┘

  Labels        🔒 platform.axisml.io/source-job-tenant: team-a
                🔒 platform.axisml.io/source-job-name:   train-llama-sft-v3
                🔒 platform.axisml.io/source-job-id:     8e0b…f3c1
                🔒 platform.axisml.io/source-output:     weights
                   user.axisml.io/cost-center:           ml-platform   ✎
                   [+ 新增 label]
  Annotations   🔒 platform.axisml.io/registered-by-user: zhang
                🔒 platform.axisml.io/registered-at:      2026-05-22T14:08:01Z

  [创建在线服务]  [上传新版本]  [复制 ref]  [获取下载凭证]  [删除]
```

要点:
- 顶部「⋯ 操作 ▾」与底部按钮区互为镜像,两套触点等价;
- ✎ 入口仅在「显示名 / 描述 / 用户 namespace 的 labels & annotations」字段出现,提交即 `PATCH /api/v1/models/{tenant}/{name}/{version}`;
- **`platform.axisml.io/*` 前缀的 labels / annotations 渲染为只读 chip(图标 🔒 标识),禁止编辑——这是 [components/platform.md §5.6](components/platform.md#56-扩展元数据写入约定) 约定的 Platform 内部 namespace,改了会破坏 provenance / 反向索引;`user.axisml.io/*` 与无前缀的 key 允许 ✎ 编辑**;
- Provenance 卡片仅在 `labels[platform.axisml.io/source-job-tenant]` 存在时渲染(由 [§4.5.3 register-from-job](components/platform.md#453-register-from-job计算任务--模型) 写入,值取 `source-job-tenant` + `source-job-name` 拼接);
- 「→」表示可点击跳转;被引方已 `Deleting` / `Deleted` → 文本红色,hover 提示「来源已失效」。

### 7.3 模型 (Kind=`model`)

**列表 Kind 专属列**:

| 列 | 来源 |
| --- | --- |
| Framework | `spec.framework` (`pytorch` / `tensorflow` / `onnx` / `safetensors` / `gguf` / `custom`) |
| Format | `spec.format` (OCI artifactType,如 `application/vnd.axisml.model.safetensors.v1+json`) |
| Params | `spec.parameters`,人读单位(如 `7B`, `175B`) |
| Task | `spec.task` |

**Kind 专属过滤**:framework / task。

**上传表单 Kind 专属字段**:

| 字段 | 写入位置 | 校验 |
| --- | --- | --- |
| Framework | `spec.framework` | 单选 |
| Format | `spec.format` | 自由文本,默认按 framework 推荐 |
| Task | `spec.task` | 自由文本(`text-generation` / `image-classification` / ...) |
| Parameters | `spec.parameters` | 数字(可空) |
| 基模型 | `spec.baseModelRef` | ArtifactRef 选择器(只列 `kind=model`,跨可见租户搜索) |
| 训练数据集 | `spec.trainingDatasetRef` | ArtifactRef 选择器(只列 `kind=dataset`) |

**详情概览 Kind 专属**:Framework / Format / Params / Task;`baseModelRef` / `trainingDatasetRef` 渲染为可跳转链接(被引方 `Deleting` / `Deleted` 时显示红色「已失效」)。Provenance 卡片(若存在):读 `labels[platform.axisml.io/source-job-tenant]` + `labels[platform.axisml.io/source-job-name]` + `labels[platform.axisml.io/source-output]?`,渲染「来源任务 `<tenant>/<jobName>` 输出 `<outputName 或 ad-hoc>`」并提供反向跳转;详细 label 约定见 [components/platform.md §4.5.3](components/platform.md#453-register-from-job计算任务--模型)。

**Kind 专属跳转按钮**:
- **「创建在线服务」**:跳 `/services/new?modelRef=<name>@<version>&tenant=<t>`,见 §6.3。
- **「上传新版本」**:`/models/new?name=<name>&tenant=<t>` 并反填上一版本 spec。

**URI 模板**:`<oci-host>/namespaces/<ns>/models/<name>:<version>` (zot)。

**注**:从训练任务一键 register 的路径见 [§5.4 详情页](#54-详情页-tab)「注册为模型」按钮;Platform 编排见 [components/platform.md §4.5.3](components/platform.md#453-register-from-job计算任务--模型)。

### 7.4 镜像 (Kind=`image`)

**列表 Kind 专属列**:

| 列 | 来源 |
| --- | --- |
| Purpose | `spec.purpose`(`training` / `inference` / `dev` 徽章) |
| Platforms | `spec.platforms[]`,join (`linux/amd64,linux/arm64`) |
| Base | `spec.baseImage` (截断 + hover) |

**Kind 专属过滤**:purpose / platform。

**上传表单 Kind 专属字段**:

| 字段 | 写入位置 | 校验 |
| --- | --- | --- |
| Purpose | `spec.purpose` | 单选 |
| Base Image | `spec.baseImage` | 自由文本(仅信息) |
| Platforms | `spec.platforms[]` | 多选,默认 `linux/amd64` |
| Entrypoint | `spec.entrypoint[]` | string 数组,可空 |
| Notes | `spec.notes` | 长文本 |

**详情概览 Kind 专属**:Purpose / Base / Platforms / Entrypoint / Notes。Image Layer 浏览 Tab (TBD,见 §12.3) — 后端调 zot manifest API 解析 layers。

**Kind 专属跳转按钮**(按 `purpose` 渲染):
- `training` → **「用作训练镜像」** → `/tenants/{tenant}/jobs/new?image=<uri>`。
- `inference` → **「用作推理镜像」** → `/services/new?image=<uri>&tenant=<t>`。
- `dev` → **「用作工作区基础」** → `/workspaces/new?image=<uri>&tenant=<t>`。

**URI 模板**:`<oci-host>/namespaces/<ns>/images/<name>:<version>` (zot)。

**上传方式**:走 §7.2.2 通用工具通路;典型 image 客户端工具会进一步封装 `docker` / `nerdctl push`,具体形态另文设计。不暴露浏览器直传。

**共享镜像约定**:`system-admin` 可在专用 `system` 租户(命名约定 `system-images`)维护「平台常用基础镜像」;Workspace / Job / Service 创建表单的镜像下拉合并展示当前租户 + system 租户的 `image` 制品。

### 7.5 数据集 (Kind=`dataset`)

**列表 Kind 专属列**:

| 列 | 来源 |
| --- | --- |
| Format | `spec.format` (`parquet` / `jsonl` / `csv` / `webdataset` / `tfrecord` / `custom`) |
| Records | `spec.numRecords`,人读单位 |
| License | `spec.license` |

**Kind 专属过滤**:format。

**上传表单 Kind 专属字段**:

| 字段 | 写入位置 | 校验 |
| --- | --- | --- |
| Format | `spec.format` | 单选 |
| Schema | `spec.schema` | JSON 编辑器,可空 |
| numRecords | `spec.numRecords` | 数字(informational) |
| totalSize | `spec.totalSize` | 数字;浏览器直传时由 JS 自动估算填入 |
| License | `spec.license` | 自由文本 |
| Splits | `spec.splits[]` | 数组项 `(name, numRecords?, uri?)`;典型 `train` / `val` / `test`,行可增删 |

**详情概览 Kind 专属**:Format / Records / Size / License;Splits 列表(每个 split 含 `name` + `numRecords` + sub-prefix);Schema 折叠 JSON 展示。

**Kind 专属跳转按钮**:
- **「训练时使用」** → `/tenants/{tenant}/jobs/new?datasetRef=<name>@<version>` (注入 `AXISML_DATASET_URI` env)。

**URI 模板**:`s3://axisml-artifacts/namespaces/<ns>/datasets/<name>/<version>/` (RustFS)。

**上传方式**:走 §7.2.2 通用工具通路;< 100 MiB 单文件支持浏览器拖入直传(走 S3 STS PUT)。

### 7.6 状态展示规则

直接展示 `artifacts.status`,三个 Kind 共用:

| 状态 | 视觉 |
| --- | --- |
| `Uploading` | 蓝色徽章 + spinner;hover 显示 token `expiresAt` 倒计时 |
| `Ready` | 绿色实心 |
| `Failed` | 红色;hover 显示 `message` |
| `Deleting` | 灰色 + spinner |
| `Deleted` | 灰色 (列表默认隐藏;过滤器开启「显示已删除」时可见,仅可读,不可复活) |

补充:
- 上传超时(24h 内未 complete)由 GC 自动转 `Failed`,UI 下次刷新自然回落。
- `Ready` 后被引方 (`baseModelRef` / `modelRef` / `datasetRef`) 软删 → resolve 时下游返 410 → 本制品状态仍为 `Ready`,但「引用方」字段标红「已失效」。

### 7.7 权限可见性

三 Kind 共用矩阵:

| 操作 | system-admin | tenant-admin (@self) | user (@self) | user (@owner) |
| :--- | :---: | :---: | :---: | :---: |
| 列出 | 全集群 | 本租户 | 本租户 | 本租户 |
| 上传(initiate / complete) | ✅ | ✅ | ✅ | — |
| 编辑展示元数据 (`displayName` / `description` / `labels` / `annotations`) | ✅ | ✅ | — | ✅ |
| 删除 | ✅ | ✅ | — | ✅ |
| 获取下载凭证 (`resolve?usage=download`) | ✅ | ✅ | ✅ | — |

完整 RBAC 矩阵见 [auth.md §3](auth.md#3-rbac-角色)。

---

## 8. 租户管理 (系统管理 → 租户)

### 8.1 页面入口

| 入口 | 路径 | 权限 |
| --- | --- | --- |
| 列表 | `/tenants` | 已登录;按角色裁剪 |
| 创建 | `/tenants/new` | `RequireSystemAdmin` |
| 详情 | `/tenants/{name}` | 该租户 `user+` 可读;不同 Tab 写权限不同 |

### 8.2 列表页

| 列 | 来源 (cluster-manager LIST) |
| --- | --- |
| 显示名 | `displayName` |
| 内部名 | `name` |
| 组织分组 | `namespace`（顶层，组织维度；与 `spec.namespace.name` 这个 K8s namespace 区分） |
| 状态 | `status.phase` (见 [§8.6](#86-状态展示规则)) |
| K8s 命名空间 | `spec.namespace.name` |
| 创建时间 | `createdAt` |
| 操作 | 详情 / 暂停 / 恢复 / 删除 / 恢复软删 |

**过滤**:状态 / 组织分组 (`namespace`) / 关键字 (全部下推 cluster-manager)。
**可见性**:`system-admin` 看全集群;其他角色先按 `user_tenant_roles` 取 `tenant_name` 集合再裁剪。

### 8.3 详情页 Tab

#### Tab 1 基本信息

展示 `displayName` / `description` / `namespace`（组织分组）/ `status.phase` / `status.conditions` / `spec.namespace.name`（K8s namespace）。

- `system-admin`:可编辑展示元数据 + 暂停 / 恢复 / 删除。
- 其他角色:只读。

#### Tab 2 配额

按 `pool` 分组的二级表格,实时读 `GET /api/v1/tenants/{name}/quotas`。

| 字段 | 说明 |
| --- | --- |
| 池名 | 顶部 `[default 池]` 切换标签,多池时多组重复版面 |
| 配额名 (`name`) | `(pool, name)` 创建后不可变,编辑态置灰 |
| min | `cpu=` / `memory=` / `gpu=` 列出 |
| max | 同上 |
| used | 实时使用量 |
| 操作 | 编辑 / 删除 |

```
[default 池]                                        [+ 新增配额]
┌──────────┬──────────────────┬──────────────────┬───────┬─────┐
│ 配额名    │ min              │ max              │ used  │ 操作 │
├──────────┼──────────────────┼──────────────────┼───────┼─────┤
│ default  │ cpu=20           │ cpu=100          │ cpu=8 │ ✏️ ❌│
│ training │ cpu=10, gpu=2    │ cpu=50,  gpu=8   │ ...   │ ✏️ ❌│
└──────────┴──────────────────┴──────────────────┴───────┴─────┘
合计 max(仅展示):cpu=150, gpu=8
```

要点:
- 「合计 max」是肉眼参考,**不做硬阻断** (决策见 [components/platform.md §4.1 租户编排](components/platform.md#41-租户编排) 配额 CRUD 行)。
- 写权限:`system-admin` 或本租户 `tenant-admin`。

#### Tab 3 成员

| 列 | 说明 |
| --- | --- |
| 用户名 | — |
| display_name | — |
| 角色 | `tenant-admin` / `user` (不允许 `system-admin`) |
| 加入时间 | — |
| 操作 | 改角色 / 移除 |

操作:
- **添加**:输入用户名 + 选角色;`role_name` 不允许 `system-admin` (`400 role-not-bindable`)。
- **自我保护**:不能移除 / 降级自己**最后一个** `tenant-admin` 角色 → `409 last-tenant-admin`,UI 提示「请先指定其他租户管理员」。

#### Tab 4 审计

入口保留 (TBD,详见 [§12](#12-后续设计))。

### 8.4 创建表单 (`system-admin` only)

字段 = cluster-manager 创建请求 1:1 透传:

| 字段 | 说明 |
| --- | --- |
| `name` | 内部名,DNS-1123,创建后不可变 |
| `displayName` / `description` / `namespace`（组织分组）| 展示元数据,可改 |
| `namespace.name` | 渲染目标 namespace,创建后不可变 |
| `quotas[]` | 初始配额数组 (可后续从详情页 Tab 2 增删) |
| `initResources` | 初始 Secret / ConfigMap / SA / RBAC (Vault / Sealed Secrets 接入为 TBD) |

UI 即时校验 + cluster-manager 兜底。完整字段清单与校验规则见 [apis/platform.yaml](apis/platform.yaml) `Tenants` tag。

### 8.5 列表 / 详情通用操作约束

- **DELETE 租户**:前置检查 `user_tenant_roles WHERE tenant_name = :name`;非空 → `409 tenant-has-members`,二次确认弹窗列出残留成员。
- **PATCH 租户**:不可变字段 `name` / `namespace.name` / `quotas[].(pool, name)` 在表单中置灰。
- **暂停 / 恢复**:仅 `system-admin`;暂停后行为详见 [components/cluster-manager.md](components/cluster-manager.md)。

### 8.6 状态展示规则

| `phase` | 视觉 |
| --- | --- |
| `Pending` | 灰色 + spinner |
| `Active` | 绿色实心 (前端解锁该租户的提交按钮) |
| `Suspended` | 黄色 |
| `Deleting` | 灰色 + spinner |
| `Deleted` (软删) | 灰色 (列表默认隐藏,过滤器开启「显示已删除」时可见) |

`conditions[]` 在 Tab 1 以折叠面板列出,异常 condition 在状态徽章旁加红点提示。

### 8.7 权限可见性

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

## 9. 资源池与资源单元 (系统管理 → 资源池)

菜单只有「资源池管理」,**资源单元作为下属能力嵌在池详情 Tab**——这是 ResourceUnit 在 Platform UI 中的唯一入口。

### 9.1 页面入口

| 入口 | 路径 | 权限 |
| --- | --- | --- |
| 资源池列表 | `/resource-pools` | 已登录 (所有用户可读,Job / Service / Workspace 提交表单需要下拉) |
| 资源池创建 | `/resource-pools/new` | `RequireSystemAdmin` |
| 资源池详情 | `/resource-pools/{name}` | 已登录可读;写权限 `system-admin` |

资源池 / 资源单元都是全集群对象,不按租户过滤。

### 9.2 资源池列表页

| 列 | 来源 |
| --- | --- |
| 名称 | compute `resource_pools.name` |
| 描述 | `description` |
| 节点选择器摘要 | `node_selector` 简写 |
| 资源单元数 | 并发 `compute.ListResourceUnits(pool, limit=0)` 拿 count;单池失败 = `-1` → UI 展示 `—` |
| 创建时间 | — |
| 操作 | 详情 / 删除 (`system-admin`) |

> 池数预期 < 50;聚合策略为列表后并发查询。`5–10 秒 LRU` 为后续优化项。

### 9.3 资源池详情 Tab

#### Tab 1 基本信息

展示 `name` / `description` / `node_selector` / `tolerations` / `metadata`。
编辑除 `name` 外字段;`name` 在表单中置灰。

#### Tab 2 资源单元

ResourceUnit 完整 CRUD:

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

要点:
- 行可展开,展开后展示 `pool.node_selector` ⊕ `unit.node_selector` 合并预览 (Pool 优先;合并规则详见 [components/compute.md §4.4 ResourceUnit](components/compute.md#44-resourceunit))。
- 命名约定 `<accelerator>[-<count>x]-<tier>[-<variant>]`,如 `a100-1x-large` / `cpu-medium`;由 compute 兜底校验。
- 删除前置阻断信息 (使用此 unit 的活跃 Job / Service 数) 在二次确认弹窗呈现:
  - `compute.ListResourceUnits(pool)` > 0 → `409 pool-in-use`;
  - `compute.ListJobs(pool, active)` / `compute.ListServices(pool, active)` > 0 → `409 unit-in-use`,弹窗列示例 name 与计数。

#### Tab 3 节点匹配预览

入口保留 (TBD)。计划:K8s typed client 反查命中 Node,显示 allocatable / requested。

#### Tab 4 审计

入口保留 (TBD)。

### 9.4 资源池创建表单

字段 = compute 创建请求 1:1 透传:`name` / `description` / `node_selector` / `tolerations` / `metadata`。

详见 [apis/platform.yaml](apis/platform.yaml) `ResourcePools` / `ResourceUnits` tag。

> Node label / taint 由管理员通过 `kubectl` 维护,**UI 不下发**。

### 9.5 资源池删除前置阻断

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

### 9.6 权限可见性

| 操作 | system-admin | 其他已登录 |
| --- | :---: | :---: |
| 列表 / 详情 (含 ResourceUnit) | ✅ | ✅ (只读) |
| 创建 / 编辑 / 删除 Pool 与 Unit | ✅ | ✗ |

完整矩阵见 [auth.md §3](auth.md#3-rbac-角色)。

---

## 10. 数据卷 (TBD)

二级菜单占位,UI 详设待 DataVolume 抽象方案确定后补齐。计划包含:

- 数据卷列表 (name / 容量 / accessMode / StorageClass / 挂载方 / 创建时间)
- 创建表单 (容量 / StorageClass / 来源:空白 / 模型快照 / 数据集挂载)
- 详情页 Tab (基本信息 / 引用方 / 历史快照)
- 在 Job / Service / Workspace 创建表单中暴露「挂载已有数据卷」选项

---

## 11. ASCII Mockup 索引

集中索引现有 ASCII 占位,供视觉评审快速定位:

- [§2.2 全局布局](#22-全局布局)
- [§3.3 Dashboard 卡片占位](#33-ascii-占位)
- [§7.2.1 制品中心 · 列表页骨架](#721-列表页通用列)
- [§7.2.2 制品中心 · 上传指引对话框](#722-上传表单通用字段--两阶段交互)
- [§7.2.3 制品中心 · 详情页概览 Tab](#723-详情页-tab通用骨架)
- [§8.3 租户详情 · 配额 Tab](#tab-2-配额)
- [§9.3 资源池详情 · 资源单元 Tab](#tab-2-资源单元)
- [§9.5 资源池删除前置阻断对话框](#95-资源池删除前置阻断)

未来评审新增的 mockup 优先放回各功能章节,本节仅作目录。

---

## 12. 后续设计

### 12.1 待补 ASCII mockup

下列页面尚无 ASCII mockup,待视觉评审时补齐:

- 工作区列表 / 创建表单 / 详情页 概览 · 访问 Tab
- 计算任务列表 / 创建表单 / 详情页 概览 · 副本 · 事件 · 日志 Tab
- 在线服务列表 / 创建表单 / 详情页 概览 · 访问 · 指标 Tab
- 制品中心:列表骨架与详情概览 mockup 已在 §7.2.1 / §7.2.3 落位;待补 Kind 专属 (镜像 / 数据集) 详情差异 mockup
- 系统管理 · 用户与角色页面
- 数据卷管理页面 (整套)
- 应用中心 (智能体 / Skills / MCP) 页面 (整套)

补齐顺序与各功能后续工作节奏对齐 ([components/platform.md §9 后续工作](components/platform.md#9-后续工作))。

### 12.2 待补 UI 设计 (横切)

- **应用中心 (Agent / Skills / MCP)**:页面结构、列表字段、创建表单。
- **审计日志 UI**:按 `target` 前缀检索 (`tenant:` / `job:` / `service:` / `resource-pool:` / `workspace:`),含告警规则模板入口。
- **OIDC 登录页**:`--auth-mode=oidc` 切换后的登录跳转 UX。
- **多集群 / 多区域选择器**:顶栏增加集群切换器。

### 12.3 待补 UI 设计 (功能模块)

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
  - 引用方反查 Tab (`Service.spec.modelRef` / `Job.spec.datasetRef` 反向索引);
  - 镜像 Layer 浏览 Tab (zot manifest 解析 + per-layer 大小展示);
  - 数据集样本预览 (按 `format` 取首 N 行);
  - 浏览器直传支持范围扩展 (现仅 S3 Kind 小文件;OCI Kind 需在浏览器实现 chunked push,工作量高);
  - 制品签名 / SBOM 展示 (cosign / notation / trivy 集成,等待 artifacts 服务支持);
  - 跨制品引用懒校验失效提示从「红色徽章」升级为详情页顶部黄条;
  - 制品配额展示 (per namespace / Kind 总大小 / 总数,等待 artifacts 服务 `size_bytes` 入表)。

---

## 13. 相关引用

- [components/platform.md](components/platform.md) — 后端业务编排、跨服务调用、PG schema
- [auth.md](auth.md) — RBAC 角色矩阵、JWT 颁发、IdentityProvider
- [apis/platform.yaml](apis/platform.yaml) — REST API 字段契约
- [monitoring.md](monitoring.md) — Dashboard 与服务指标数据来源
- [overview.md](overview.md) — 系统概念与组件关系
- [components/compute.md](components/compute.md) — Job / Service / ResourcePool / ResourceUnit 字段权威
- [components/cluster-manager.md](components/cluster-manager.md) — Tenant 字段权威
- [components/artifacts.md](components/artifacts.md) — 制品中心字段权威
