# AxisML Platform 计算任务 详细设计

本文档是 AxisML Platform 子系统下 **「训练 & 推理 → 计算任务」** 一级功能的全栈设计，承接 [PRD §6.2.2 自定义任务](../../product/prd.md#622-自定义任务)。

「计算任务」（Compute Job，PRD 内称「自定义任务」）即一次性的训练 / 微调 / 数据处理 workload，支持单机与分布式（多机多卡）。Platform 全部能力都通过调用 [compute](../core/compute.md) 的 Job 端点实现，自身不持有 Job 相关数据。

| 模块 | Platform 职责 | 调用下游 |
| --- | --- | --- |
| 任务视图与生命周期（[§4](#4-菜单与列表页) / [§7.1](#71-任务-crud)） | 创建 / 列表 / 详情 / 删除；按用户身份过滤可见任务 | `compute.CreateJob` / `GetJob` / `ListJobs` / `DeleteJob` |
| 取消（[§4.2](#42-操作按钮) / [§7.2](#72-取消)） | 用户视角的 cancel 入口 | `compute.CancelJob` |
| 副本 / 事件 / 日志（[Tab 2–4](#tab-2-副本) / [§7.3](#73-副本--事件--日志)） | 鉴权 + 字段透传 | `compute.GetJobReplicas` / `GetJobEvents` / `GetJobLogs` |
| 产出回写制品（[§7.4](#74-产出注册为模型) / [§11](#11-后续迭代)） | 串联 Artifacts | `artifacts.InitiateUpload` / `CompleteUpload`（阶段二） |

**关键不变式：**

> Platform 自有 PG **不为计算任务建任何表**，与 [platform/overview.md §7.1](overview.md#71-仓库与目录布局) 标注的「`internal/job/` 无本地表，仅代理」一致。任务的标识在 Platform 视图层是 `(tenant_name, job_name)`，URL 取 `/api/v1/tenants/{tenant}/jobs/{name}`；写之前调一次 `clustermanager.GetTenant(tenant_name)` 拿 `compute_namespace = Tenant.spec.namespace.name` 与可用 quota 清单。
>
> Job spec 在 [compute](../core/compute.md) 侧不可变，因此 Platform UI 不提供任何「编辑任务」入口；改参数 = 新建任务，由前端文案明确引导。

**文档组织：**

- **Part I — 服务边界**（[§1](#1-概述与定位) – [§3](#3-数据模型platform-自有部分)）：定位、可见性矩阵、Platform 自有数据模型。
- **Part II — UI 设计**（[§4](#4-菜单与列表页) – [§6](#6-详情页-tab)）：菜单 / 列表页 / 创建表单 / 详情页 Tab。
- **Part III — 后端 API 契约**（[§7](#7-rest-路径与响应格式)）：REST 路径、字段、RBAC。
- **Part IV — 后端实现**（[§8](#8-模块结构)）：模块结构、RBAC 装配、上下文解析、可观测性。
- **Part V — 实施与验证**（[§9](#9-实现路径) – [§13](#13-相关引用)）：阶段化实现、关键决策、后续迭代、测试、参考。

---

## Part I — 服务边界

## 1. 概述与定位

「计算任务」是「训练 & 推理」菜单下面向普通用户（租户管理员 / 系统管理员同样可使用）的一次性 workload 入口，覆盖：

- 选 backend / engine（默认 `(native, job)`）、镜像、资源池 + 资源单元、副本数、启动命令、环境变量后提交训练 / 微调 / 数据处理任务；
- 列表 / 详情查看任务运行状态、Pod 副本分布、K8s Event 与容器日志；
- 对运行中任务发起 cancel；对任一状态发起 delete；
- 任务结束后一键把约定输出目录注册为模型版本（[§7.4](#74-产出注册为模型)，阶段二）。

下层语义、字段契约与状态推进一律到 [compute.md §6](../core/compute.md#6-job) / [compute-operator.md §4](../core/compute-operator.md#4-mljob-controller) 查；本文不重复。

## 2. 角色与可见性矩阵

下表只列与本功能相关的能力；persona ↔ RBAC 角色映射沿用 [platform/overview.md §2.2](overview.md#22-用户角色persona)。`@self` 表示「在该 job 所属租户上具备相应角色」；`@owner` 表示「`compute.jobs.owner_user == current_user.username`」。

| 能力 | `system-admin` | `tenant-admin@self` | `user@self & @owner` | `user@self & 非 owner` |
| --- | :---: | :---: | :---: | :---: |
| 列出可见任务 | 全集群 | 该租户全部 | 仅自己提交的 | 仅自己提交的 |
| 查看任务详情 | ✅ | ✅ | ✅ | ❌ |
| 提交任务 | ✅ | ✅ | ✅ | — |
| 取消 / 删除 | ✅ | ✅ | ✅ | ❌ |
| 查看副本 / 事件 / 日志 | ✅ | ✅ | ✅ | ❌ |
| 把产出注册为模型版本 | ✅ | ✅ | ✅ | ❌ |

`system-admin` 在所有动作上短路放行。`tenant-admin` 在自己绑定的租户内拥有等同于 `system-admin` 的任务管理能力，覆盖所有成员提交的任务；普通 `user` 仅能操作自己提交的任务。

## 3. 数据模型（Platform 自有部分）

按 [platform/overview.md §9 关键设计决策](overview.md#9-关键设计决策) 「Platform PG 仅存身份与视图映射」原则，**Platform 不为计算任务建立任何表**。

### 3.1 标识与寻址

- `tenant_name` 直接来自 Tenant CR `metadata.name`（详见 [tenant.md §3](tenant.md#3-数据模型platform-自有部分)）；
- `job_name` 由用户在创建表单中显式指定（DNS-1123，同一 `compute_namespace` 下唯一）；
- 下游 join key 用 `(compute_namespace, job_name)`，其中 `compute_namespace = Tenant.spec.namespace.name`，每次写请求前由 Platform 通过 `clustermanager.GetTenant(tenant_name)` 解析；
- 不为 Job 派生 Platform 端 uuid——Job 还在不在、它在哪个 namespace、它属于谁，三件事全部由 Compute 实时回答。

### 3.2 列表查询路径

1. 按 RBAC 取可见租户集合 `tenant_names`（来自 `user_tenant_roles` 或 cluster-manager LIST，详见 [tenant.md §7.1](tenant.md#71-租户-crud)）；
2. 并行 `clustermanager.GetTenant(name)` 解析每个租户的 `compute_namespace`（request-scoped memoize）；
3. 并行 `compute.ListJobs(compute_namespace, {ownerUser?, status?, q?, limit, continue?})`；
4. 内存合并；对普通用户再按 `jobs.owner_user == current_user.username` 二次过滤；
5. 返回 `Job` DTO 列表（注入展示字段：tenant 名 / pool / unit 展示名）。

绝大多数日常查看都聚焦在「某一个租户的我的任务」上，单租户一次 RPC 即可。跨租户合并仅 `system-admin` 触发。

### 3.3 对下游的依赖

| 调用 | 用途 |
| --- | --- |
| `clustermanager.GetTenant(name)` | 解析 `Tenant.spec.namespace.name` + `Tenant.spec.quotas[]`；校验 `status.phase == Active` |
| `compute.{Create,Get,List,Cancel,Delete}Job` | Job CRUD + 取消（[compute.md §6](../core/compute.md#6-job) 现有端点，无新增需求） |
| `compute.GetJob{Replicas,Events,Logs}` | 详情页 Tab 2–4 透传 |
| `compute.GetResourcePool(id)` | 把表单提交的 `resourcePoolId` (uuid) 翻译为 pool name，用于拼接 ElasticQuota CR 名 `axisml-<tenant>-<pool>-<quota>` 与校验该 quota 在 `Tenant.spec.quotas[poolName]` 内存在 |
| `artifacts.InitiateUpload` / `CompleteUpload` | [§7.4](#74-产出注册为模型) 模型注册桥端点（阶段二与 [model.md](model.md) 同步） |

---

## Part II — UI 设计

## 4. 菜单与列表页

菜单位置：「训练 & 推理 → 计算任务」。

### 4.1 列表页

| 列 | 来源 | 说明 |
| --- | --- | --- |
| 任务名 | `jobs.display_name`（fallback `jobs.name`） | 主展示列；点击进详情 |
| 镜像 | `jobs.spec.roles[0].template.image` | 多 role 时取首个 role 镜像 + 角标 `+N` |
| 后端 | `jobs.spec.backend.{name, engine}` | 列过滤 |
| 资源池 · 单元 | `jobs.pool_id` + `jobs.resource_unit_id` | 联表 ResourcePool / ResourceUnit 取展示名 |
| 状态 | `jobs.status` 直接展示 | 状态集合见 [compute.md §6.2](../core/compute.md#62-数据模型) |
| Owner | `jobs.owner_user` | 仅 admin 可见列 |
| 开始 / 结束时间 | `jobs.started_at` / `jobs.finished_at` | finished 列空表示未完成 |
| 操作 | — | 取消 / 详情 / 删除 / 注册为模型 |

- 过滤：状态、关键字（任务名 / 镜像）、Owner（admin only）、backend、租户（admin 跨租户视图）。
- 列表渲染走 [§3.2](#32-列表查询路径) 路径。
- 列表可见性：
  - `system-admin`：默认仅展示自己「最近活跃」的租户视图，提供「切换租户」下拉跨租户浏览；
  - `tenant-admin@self`：可见租户对应 namespace；
  - `user`：上一条基础上再按 owner 二次过滤。
- 排序：默认 `created_at desc`；支持按 `started_at` / `finished_at` 排序。

### 4.2 操作按钮

- **取消**：调 `POST .../cancel`；二次确认提示「已运行的 Pod 会被驱逐，已写出的产出文件保留」。`Creating` 状态由下游拒绝（见 [compute.md §6.4.2](../core/compute.md#642-取消语义)），UI 上禁用按钮。
- **详情**：进入详情页。
- **删除**：owner / `tenant-admin@self` / `system-admin`；二次确认。
- **注册为模型**（`Succeeded` 状态可点）：弹小表单填模型 `name` / `version` / `displayName` / 产出路径 → 调 `POST .../register-model`（阶段二）。
- **重新提交**：纯前端语法糖，把当前任务的 `spec` 反填到创建表单；不调端点，新任务有新 `job_name`。

## 5. 创建任务表单

字段：

| UI 字段 | 写入位置 | 说明 |
| --- | --- | --- |
| 任务名 | `jobs.name`（同时是 MLJob `metadata.name`）| 必填；DNS-1123；同一 `compute_namespace` 下唯一 |
| 显示名 | `jobs.display_name` | 可选 |
| 描述 | `jobs.description` | 可选 |
| 租户 | RBAC + 路径变量 `{tenant}` | 默认当前活跃租户；`system-admin` 可选任意租户 |
| 后端 / 引擎 | `jobs.spec.backend.{name, engine}` | 下拉；可选项见 §5.1；默认 `(native, job)` |
| 资源池 / 资源单元 | `jobs.spec.scheduling.*` + `jobs.pool_id` / `resource_unit_id` | unit 必须属于 pool |
| 配额 | 拼接 `axisml-<tenant>-<pool>-<quota>` 写入 `spec.scheduling.quota` | 与 [overview.md §6.1](overview.md#61-用户提交训练任务) 一致 |
| Roles | `jobs.spec.roles[]` | 按 backend 渲染不同 role 集合，见 §5.1 |
| `runPolicy.activeDeadlineSeconds` | `spec.runPolicy.activeDeadlineSeconds` | 可选硬超时（秒）|
| `runPolicy.ttlSecondsAfterFinished` | `spec.runPolicy.ttlSecondsAfterFinished` | 可选；终态后底层资源 GC 延迟 |
| `runPolicy.backoffLimit` | `spec.runPolicy.backoffLimit` | 可选重试次数 |

校验：

- DNS-1123 + 长度（[compute.md §3.2](../core/compute.md#32-pg-编排约定)）；
- 镜像必须能被 `artifacts.Resolve`；
- ResourceUnit 必须属于所选 ResourcePool；
- `quota` 必须出现在 `Tenant.spec.quotas[resourcePoolId]`（Platform 用 `clustermanager.GetTenant` 校验）；
- 其他跨字段约束以 Compute 返回为准。

### 5.1 按后端渲染 Roles

每个 backend / engine 对应一组固定 role（由 [compute-operator.md §4.5](../core/compute-operator.md#45-内置-handler) 约定），前端据此渲染：

| backend / engine | role 集合 | 阶段 |
| --- | --- | --- |
| `(native, job)` | `worker`（replicas=1，可改） | 阶段一（MVP）|
| `(native, podgroup)` | `worker`（replicas≥1，gang） | 阶段一 |
| `(kubeflow-trainer, pytorchjob)` | `master` + `worker`（可选 `elasticAgent`） | 阶段二 |
| `(kubeflow-trainer, tfjob)` | `chief` / `worker` / `ps` / `evaluator`（任一可空） | 阶段二 |
| `(kubeflow-trainer, mpijob)` | `launcher` + `worker` | 阶段二 |
| `(custom, *)` | 由 `backend.config` 决定 | 阶段三 / TBD |

每个 role 块包含：副本数 / 镜像 / 启动命令 / 参数 / 环境变量 / 重启策略。所有 role 共享同一 ResourceUnit（per-role ResourceUnit 作为 [§11](#11-后续迭代) 项）。

### 5.2 不在创建表单中的字段

- `spec.backend.config`：阶段一不开放；阶段二 `(kubeflow-trainer, *)` 上线时按需开放。
- `spec.scheduling.priorityClass`：阶段一统一走 ResourcePool 默认。
- `nodeSelector` / `tolerations`：由 ResourcePool + ResourceUnit 自动合并（见 [compute.md §5.4](../core/compute.md#54-注入规则)）。
- `volumes` / `volumeMounts`：MVP 不开放；若需持久化产出，由系统管理员维护「带预挂载产出 PVC 的镜像」，或在 [§11](#11-后续迭代) 推动 DataVolume 集成。

## 6. 详情页 Tab

详情页以 `(tenant_name, job_name)` 为维度，分为五个 Tab：概览、副本、事件、日志、审计。

### Tab 1 概览

展示（全部来自 `compute.GetJob` 响应，全只读）：基本信息 + 当前状态卡片 + 时间线 + 调度参数（折叠）+ runPolicy + Roles 概览（`activeReplicas` / `readyReplicas` / `succeededReplicas` / `failedReplicas`）。

操作：取消 / 删除 / 注册为模型 / 重新提交 / 复制 YAML（把 `jobs.spec` 渲染成等价 MLJob YAML）。

### Tab 2 副本

调 `compute.GetJobReplicas` 透传：每个 Pod 一行（副本编号 / Pod 名 / phase / startedAt / 节点）。
操作：渲染等价 `kubectl exec` / `kubectl logs` 命令模板供用户复制；MVP 不提供 in-browser shell。

### Tab 3 事件

调 `compute.GetJobEvents` 透传：聚合的 K8s Event，按 `lastTimestamp` 倒序。

### Tab 4 日志

调 `compute.GetJobLogs` 透传：

- 默认按 `replica=0` 拉最近 1000 行；
- 支持切换 `replica` / `pod` / `container` / `tailLines` / `follow`；
- `follow=true` 走 SSE 流式渲染；
- Pod 已 GC 时下游返 410，UI 展示「日志已过期」+「重新提交」按钮。

### Tab 5 审计

入口保留，MVP 不交付，登记到 [§11](#11-后续迭代)。

---

## Part III — 后端 API 契约

## 7. REST 路径与响应格式

- 所有路径统一在 `/api/v1/tenants/{tenant}/jobs/...` 前缀下；权限差异由 RBAC 中间件按角色 + 资源所有权判定。
- 错误格式：RFC 7807 problem+json，复用 [overview.md §7.3](overview.md#73-错误处理) 的样例；Compute / cluster-manager 返回的 problem 原样透传。
- 出站调用：`internal/client/compute` typed client，方法集为 `CreateJob` / `GetJob` / `ListJobs` / `CancelJob` / `DeleteJob` / `GetJobReplicas` / `GetJobEvents` / `GetJobLogs`。
- 每个 endpoint 在下文括号内标注允许的角色：`system-admin` / `tenant-admin@self` / `user@self & @owner`。

### 7.1 任务 CRUD

#### `POST /api/v1/tenants/{tenant}/jobs`（已登录 + `user@self` 及以上）

请求体：

```json
{
  "name": "train-bert-base-01",
  "displayName": "BERT Base 微调实验 01",
  "description": "针对 v3 数据集的 5 epoch 微调",
  "backend": { "name": "native", "engine": "job", "config": {} },
  "resourcePoolId": "...",
  "resourceUnitId": "...",
  "quota": "default",
  "roles": [
    {
      "name": "worker",
      "replicas": 1,
      "restartPolicy": "OnFailure",
      "template": {
        "image": "axisml.io/registry/namespaces/team-a/images/pytorch-train:2026-04",
        "command": ["python"],
        "args": ["train.py", "--epoch", "5"],
        "env": [{ "name": "PYTHONUNBUFFERED", "value": "1" }]
      }
    }
  ],
  "runPolicy": {
    "activeDeadlineSeconds": 86400,
    "ttlSecondsAfterFinished": 3600,
    "backoffLimit": 0
  }
}
```

处理顺序：

1. RBAC + 提交校验（[§5](#5-创建任务表单) 校验列表）；
2. `clustermanager.GetTenant(tenant)` 解析 `compute_namespace` 与可用 quota；
3. 拼接 `spec.scheduling.quota = axisml-<tenant>-<pool>-<quota>`；
4. 调 `compute.CreateJob(compute_namespace, body)`；
5. 透传响应为 `Job` DTO。

失败语义：`400` 提交校验失败；`404` tenant 不存在 / `Tenant.status.phase != Active`；`409` 同 namespace 下已有未软删的同名 job；`5xx` 透传下游 problem。

#### `GET /api/v1/tenants/{tenant}/jobs`（已登录，按角色裁剪）

按 [§3.2](#32-列表查询路径) 流程返回单租户视图。query 参数分两类：

- **下推 Compute**（分页与过滤都精确）：`status` / `limit` / `continue`——原样透传到 `compute.ListJobs`。
- **Platform 内存过滤**（在透传结果上做二次筛选，对分页结果不精确）：`q`（关键字，匹配 name / displayName / image）/ `ownerUser`（admin only）/ `backendName` / `backendEngine`。

`q` 与 backend / ownerUser 过滤的下推已登记在 [§11](#11-后续迭代) 「列表过滤下推」中——MVP 阶段使用这几个参数时应配合较大的 `limit`，避免分页截断遗漏。响应可能附 `partial: true` 标志，详见 [§8.3](#83-失败语义)。

#### `GET /api/v1/jobs`（`system-admin` only）

跨租户全集群浏览入口。query 同上 + `tenantName?`：不填 = 全集群（Platform 并行调所有 tenant 对应 `compute.ListJobs` 后合并）；填了 = 等价于上面的 tenant-scoped 端点。

#### `GET /api/v1/tenants/{tenant}/jobs/{name}`（`system-admin` 或 `tenant-admin@self` 或 `user@self & @owner`）

返回 `Job` DTO：

```json
{
  "tenantId": "...",
  "tenantName": "team-a",
  "tenantDisplayName": "Team A",
  "computeNamespace": "team-a",
  "name": "train-bert-base-01",
  "displayName": "BERT Base 微调实验 01",
  "description": "...",
  "ownerUser": "alice",
  "backend": { "name": "native", "engine": "job", "config": {} },
  "resourcePoolId": "...",
  "resourcePoolName": "gpu-a100",
  "resourceUnitId": "...",
  "resourceUnitName": "a100-1x-large",
  "quota": "axisml-team-a-default-default",
  "roles": [
    {
      "name": "worker",
      "replicas": 1,
      "restartPolicy": "OnFailure",
      "template": { "image": "...", "command": ["python"], "args": ["train.py", "--epoch", "5"], "env": [] },
      "status": { "activeReplicas": 1, "readyReplicas": 1, "succeededReplicas": 0, "failedReplicas": 0 }
    }
  ],
  "runPolicy": { "activeDeadlineSeconds": 86400, "ttlSecondsAfterFinished": 3600, "backoffLimit": 0 },
  "status": "Running",
  "message": "",
  "startedAt": "2026-05-01T03:21:00Z",
  "finishedAt": null,
  "createdAt": "2026-05-01T03:20:42Z",
  "updatedAt": "..."
}
```

合并器：`compute.GetJob(ns, name)` 响应 + Platform 注入的展示字段（`tenant` / `resourcePoolName` / `resourceUnitName`）。

#### `DELETE /api/v1/tenants/{tenant}/jobs/{name}`（owner / `tenant-admin@self` / `system-admin`）

调 `compute.DeleteJob(compute_namespace, name)` 透传。幂等。

### 7.2 取消

#### `POST /api/v1/tenants/{tenant}/jobs/{name}/cancel`（owner / `tenant-admin@self` / `system-admin`）

无请求体。调 `compute.CancelJob(compute_namespace, name)` 透传——下游 message 由 Compute 写死 `'user cancelled'`，Platform 不参与拼接。下游对状态合法性的拒绝（如 `Creating` 不可 cancel）以 4xx problem 形式透传，UI 直接展示。

> 自由文本 cancel reason 需要 Compute `/cancel` 接受请求体，目前并不支持；如果将来产品确认需要，应作为 Compute 端的新增能力同步推进，而非 Platform 单边塞字段。

### 7.3 副本 / 事件 / 日志

三个端点都是「鉴权 + 透传 Compute 同名端点」。

| Endpoint | 方法 | 权限 | 透传到 |
| --- | --- | --- | --- |
| `/api/v1/tenants/{tenant}/jobs/{name}/replicas` | `GET` | 同 GET 详情 | `compute.GetJobReplicas` |
| `/api/v1/tenants/{tenant}/jobs/{name}/events` | `GET` | 同 GET 详情 | `compute.GetJobEvents` |
| `/api/v1/tenants/{tenant}/jobs/{name}/logs` | `GET` | 同 GET 详情 | `compute.GetJobLogs` |

`/logs` 端点的 query 参数（`pod` / `replica` / `container` / `tailLines` / `follow` / `previous`）与 Compute 一致，typed client 透传。响应体直接 pipe，Platform 不缓冲不解析正文：

- `follow=false`（默认）：透传 `text/plain` chunked 流；
- `follow=true`：透传 `text/event-stream` SSE 流（语义见 [compute.md §6.4.4](../core/compute.md#644-任务日志透传)）。

两种模式 Platform 仅做 RBAC + header 转写后桥接，不参与边界识别 / 重组。

### 7.4 产出注册为模型

#### `POST /api/v1/tenants/{tenant}/jobs/{name}/register-model`（owner / `tenant-admin@self` / `system-admin`）

> 本端点是 Job 与制品中心的桥端点；Artifacts 上传 / `complete` 流程主导文档是 [model.md](model.md)，本节只声明 Platform 视图层契约。

**前提依赖**：本端点的 `outputPath` 默认假设 Job 在执行期间已把产出写到持久化路径。但 MVP 的创建表单 **不开放** `volumes` / `volumeMounts`（[§5.2](#52-不在创建表单中的字段)），故实际可用要等以下任一就绪：

- 系统管理员维护一组「带预挂载产出 PVC 的镜像」并约定写入路径，或
- 阶段二开放 `volumes` / `volumeMounts` UI，或
- DataVolume 集成上线（[§11](#11-后续迭代)）。

这是 register-model 整体作为阶段二而非阶段一的另一个原因——阶段一就算交付 API 端点，用户也没有稳定的产出路径可指向。

请求体：

```json
{
  "modelName": "bert-base-finetuned",
  "modelVersion": "v1",
  "displayName": "BERT Base 微调 v1",
  "description": "...",
  "outputPath": "/workspace/output/model"
}
```

处理顺序（阶段二）：

1. RBAC + `compute.GetJob(ns, name)` 校验 `status == Succeeded`；
2. 调 `artifacts.InitiateUpload(compute_namespace, "model", modelName, version)` 拿 `{uploadCredentials, ...}`；
3. 返回凭证给客户端，由客户端 / 工作区脚本把 `outputPath` 文件直推到 zot；
4. 客户端调 `POST .../register-model/complete`（body `{digest}`），Platform 转发到 `artifacts.CompleteUpload`。

MVP 不交付；按钮在 UI 上灰显并标注「即将开放」。

---

## Part IV — 后端实现

## 8. 模块结构

目录：`components/platform/backend/internal/job/`

| 文件 | 职责 |
| --- | --- |
| `handler.go` | Gin 路由注册（`/api/v1/tenants/:tenant/jobs` 与 `/api/v1/jobs` 前缀）、RBAC gate 装配 |
| `service.go` | 业务编排：tenant 解析 → quota 名拼接 → 调 Compute；列表跨租户并行合并 |
| `context.go` | request-scoped 解析器：tenant → `compute_namespace` / `quotas`；详见 [§8.2](#82-上下文解析) |
| `dto.go` | 请求 / 响应类型；与 Compute API DTO 的显式映射 |
| `view.go` | `Job` DTO 合并器：`compute.GetJob` 响应 + Platform 注入展示字段（tenant / pool / unit 展示名） |
| `validate.go` | 提交前校验（[§5](#5-创建任务表单)） |
| `logstream.go` | 日志透传 pipe：`follow=false` 透传 chunked、`follow=true` 透传 SSE |
| `register.go` | 阶段二：[§7.4](#74-产出注册为模型) 注册为模型的桥接 |

无 `repository.go`：无 Platform 自有表（[§3](#3-数据模型platform-自有部分)）。

### 8.1 RBAC 中间件接入

`internal/auth` 提供 `RequireSystemAdmin` / `RequireTenantRole(role, tenantParam)` / `RequireJobOwner` 中间件，定义见 [auth.md](auth.md)。

| 路由 | 中间件链 |
| --- | --- |
| `GET /api/v1/jobs` | `RequireSystemAdmin` |
| `GET /api/v1/tenants/:tenant/jobs` | `RequireTenantRole("user", ":tenant")`；handler 内部按 `@owner` 二次裁剪 |
| `POST /api/v1/tenants/:tenant/jobs` | `RequireTenantRole("user", ":tenant")` |
| `GET /api/v1/tenants/:tenant/jobs/:name` | `RequireJobOwner(":tenant", ":name")`；语义为「`@owner` 或在 tenant 上具备 `tenant-admin` 以上角色」；`system-admin` 短路 |
| `POST .../cancel`、`DELETE`、`POST .../register-model` | 同上 |
| `GET .../replicas`、`GET .../events`、`GET .../logs` | 同上 |

`RequireJobOwner` 需要先 `compute.GetJob(ns, name)` 拿 `owner_user`；该 GET 结果通过 `gin.Context.Set("jobView", view)` 注入后续 handler，避免重复调用。

### 8.2 上下文解析

**Tenant 解析**——每个写请求与详情类读请求的第一步：

```go
ctx := resolveTenantContext(c, tenantName) // 内部：
   //   1. PG: SELECT id, display_name FROM tenants WHERE name=$1
   //   2. clustermanager.GetTenant(tenantName) → spec.namespace.name + spec.quotas[]
   //   3. 校验 spec.status.phase == Active
   //   返回 { tenantId, tenantDisplayName, computeNamespace, quotas: map[poolName][]quotaName }
```

**ElasticQuota 名拼接**——仅在创建任务时需要，复用上面的 `ctx`：

```go
pool := compute.GetResourcePool(req.ResourcePoolId)        // 拿 pool.name
// 校验 req.Quota 在 ctx.quotas[pool.name] 中存在
elasticQuotaName := "axisml-" + ctx.TenantName + "-" + pool.Name + "-" + req.Quota
// 写入 spec.scheduling.quota
```

拼接逻辑唯一来源于本模块；后续可抽到 `internal/tenant/quotaname.go` 与 [workspace.md §7.1](workspace.md#71-工作区-cruddelete) 共享。

### 8.3 失败语义

Platform 端不持久化任何 Job 字段，自然无双写一致性问题：

- 创建 / 取消 / 删除 / register-model：单点透传下游错误，不引入 Platform 补偿队列；
- 列表：并行调用中部分租户失败时，该租户在结果集中标 `partial=true` + `error.detail`，其余照常返回；前端在列表头部黄条提示「部分租户数据不可用」。

### 8.4 度量与日志

特有指标（通用上游调用指标见 [overview.md §7.5](overview.md#75-下游-typed-client)）：

- `platform_job_action_total{action, status}`：`action ∈ {create, get, list, cancel, delete, register_model, logs_stream}`；
- `platform_job_list_tenant_fanout`：histogram，单次列表请求的下游扇出 namespace 数；
- `platform_job_list_partial_total{reason}`：counter，[§8.3](#83-失败语义) 中部分租户失败次数；
- `platform_job_logs_stream_active`：gauge，当前活跃 SSE log stream 连接数。

zap 字段约定：每条任务操作日志必带 `tenant_name` / `job_name` / `actor_user` / `action` / `status`；创建额外带 `backend_name` / `backend_engine` / `pool_id` / `resource_unit_id`；取消 / 删除额外带下游返回的 `compute_message`（如有）。

**审计日志**：Tab 5 审计是阶段二能力，但底层写入责任在 Platform job handler——`create` / `cancel` / `delete` / `register-model` 成功后，由 handler 向 [overview.md §7.4](overview.md#74-pg-schema) 的 `audit_logs` 表写一行：`action=job.<动作>`、`target=job:{tenant}/{name}`、`metadata` jsonb 含 `backend_name` / `backend_engine` / `pool_id` / `resource_unit_id` 等关键字段。Tab 5 渲染时按 `target` 前缀检索。MVP 阶段不写入也不展示——但写入路径在阶段二上 Tab 5 时只需打开开关即可，不再回头改 handler 逻辑。

---

## Part V — 实施与验证

## 9. 实现路径

### 9.1 阶段一（MVP）

| 模块 | 范围 | 完成信号 |
| --- | --- | --- |
| handler / service / dto / view | [§7.1](#71-任务-crud) / [§7.2](#72-取消) / [§7.3](#73-副本--事件--日志) endpoint；仅 `(native, job)` / `(native, podgroup)` backend | `make platform-build` 通过 |
| RBAC 装配 | [§8.1](#81-rbac-中间件接入) 路由表全部接通 | 单元测试覆盖中间件分支 |
| validate.go | [§5](#5-创建任务表单) 校验全分支 | 单元测试覆盖 |
| logstream.go | SSE pipe 透传 Compute `/logs?follow=true` | Integration 覆盖 |
| 创建表单（前端） | 单 role 块；backend 下拉只列 `(native, job)` / `(native, podgroup)` | 用户可在 UI 提交并跑通 |
| Integration | testcontainers PG + in-process gin + httptest fake Compute + httptest fake cluster-manager；happy path：创建 → status 推进 → 删除 | `make platform-integration` 通过 |

阶段一显式不覆盖：`(kubeflow-trainer, *)` 多 role backend、`(custom, *)`、register-model、审计 Tab、列表 SSE 推送。

### 9.2 阶段二

1. `(kubeflow-trainer, pytorchjob)` / `tfjob` / `mpijob`：handler / validate 加 role 集合校验；创建表单加多 role 块；详情页 Tab 2 按 role 分组。
2. register-model 完整化：与 [model.md](model.md) 同步推进 Artifacts 上传桥接。
3. Tab 5 审计日志：按 `target=job:{tenant}/{name}` 索引展示。
4. 重新提交 UX：把当前任务的 `spec` 反序列化到创建表单（[§4.2](#42-操作按钮)）。

### 9.3 阶段三 / TBD

详见 [§11](#11-后续迭代)。

## 10. 关键设计决策

| 决策项 | 决策 | 理由 |
| --- | --- | --- |
| Platform PG 表 | **不为计算任务建任何表** | 与 [overview.md §9](overview.md#9-关键设计决策) 「Platform PG 仅存身份与视图映射」原则一致；建表只会带来双写漂移 |
| 寻址方式 | `(tenant_name, job_name)` 二元组，URL `/api/v1/tenants/{tenant}/jobs/{name}` | 没有 Platform 自有 id 可用；tenant-scoped 路径自带 RBAC 边界 |
| 取消语义 | 显式 `POST /cancel`，不暴露通用 patch | 与 PRD「随时可终止训练以释放资源」贴近；UI 按钮 1:1 映射 |
| 任务 spec 不可变 | UI 不提供「编辑任务」入口；改参数 = 新建 | 与下游约束对齐；避免在 Platform 引入对 Compute 不支持的 PATCH 路径 |
| 列表跨租户策略 | 默认单租户视图；`system-admin` 跨租户走并行 list + 内存合并；部分失败标 `partial=true` 而非整体 5xx | 普通用户日常只看自己租户；部分容忍策略避免单点故障拖垮全列表 |
| backend 范围 | MVP 只支持 `(native, job)` / `(native, podgroup)`；kubeflow-trainer 阶段二；custom 阶段三 | 多 role UI 工作量大，单独拆分阶段 |
| ResourceUnit 维度 | 单 Job 单 ResourceUnit，所有 role 共享 | 现状下游已经是单 unit；多 role 异构作为 [§11](#11-后续迭代) |
| 日志 / 副本 / 事件 | 全部透传 Compute 同名端点，Platform 不二次解析 | 避免在 Platform 引入 K8s 直读路径 |
| register-model 桥端点 | Platform 持权限边界 + 转 Artifacts，blob 由客户端 / 工作区脚本自上传 | Platform 后端不搬运二进制 |
| ElasticQuota 名拼接 | Platform 写前用 `axisml-<tenant>-<pool>-<quota>` 实时拼接，并校验该 quota 在 Tenant CR 内存在 | 提前校验避免 Pod 调度时才发现 quota 不存在 |

## 11. 后续迭代

- **`(kubeflow-trainer, *)` 多 role backend**：阶段二能力。
- **`(custom, *)` 接入**：UI 表达需要 JSON schema 编辑器；与 [compute-operator.md §4.6](../core/compute-operator.md#46-后续工作) 一同推进。
- **per-role ResourceUnit**：解锁 PyTorchJob master CPU、worker GPU 场景；需下游 `jobs` 表 schema 演进。
- **register-model 完整链路**：详见 [§7.4](#74-产出注册为模型) 与 [model.md](model.md)。
- **审计日志 Tab**：复用 [overview.md §7.4](overview.md#74-pg-schema) 的 `audit_logs` 表。
- **任务模板 / 预设**：常用 `(backend, image, command, args, env, runPolicy)` 组合一键创建；纯前端语法糖。
- **重新提交 UX**：spec 反填创建表单（[§4.2](#42-操作按钮)）。
- **DAG 工作流**：组合多个 Job 形成依赖链；呼应 [compute.md §6.6](../core/compute.md#66-后续工作)。
- **SSE / WebSocket 增量列表**：替代轮询。
- **列表过滤下推**：Compute 端 `?ownerUser=` / `?labelSelector=` 等参数，把内存过滤收敛到 PG 查询。
- **cluster-manager 批量 GetTenant**：减少 list 跨租户多次 RPC（与 [workspace.md §11](workspace.md#11-后续迭代) 同源）。

## 12. 测试策略

- **单元**（`internal/job/*_test.go`）：
  - `validate.go` 全分支；
  - `view.go` DTO 合并器（roles / runPolicy / 时间字段映射）；
  - `service.go` 列表合并器：部分租户失败时 `partial=true` 标记正确；
  - RBAC 中间件分支（`system-admin` 短路 / `@owner` 比对 / 跨租户拒绝）；
  - `context.go` 解析器：`Tenant.status.phase != Active` 时返 400。
- **integration**（`components/platform/backend/test/integration/`）：
  - testcontainers PostgreSQL + in-process gin（`httptest`）+ httptest fake Compute + httptest fake cluster-manager；
  - happy path：创建（`(native, job)`，单 worker）→ status 推进 → 详情 → 删除；
  - cancel path：创建 → cancel → 状态推进；
  - 列表多租户合并：构造 3 个租户、每租户 5 任务，断言并行 RPC 数 == 3、合并结果正确；
  - 列表部分失败：1 个租户的 Compute 返 5xx，断言响应带 `partial=true` 且其余租户数据完整；
  - RBAC 矩阵：4 种角色在每个 endpoint 上的允许 / 拒绝；
  - 日志 SSE pipe：fake Compute 返回 chunked 流，Platform 透传后客户端按行收到。
- 不引入额外 minikube e2e：端到端 MLJob 行为由 [compute.md §9](../core/compute.md#9-测试) 与 [compute-operator.md §7](../core/compute-operator.md#7-测试) 覆盖。

## 13. 相关引用

- [PRD §6.2.2 自定义任务](../../product/prd.md#622-自定义任务)
- [docs/system_design/platform/overview.md](overview.md)
- [docs/system_design/platform/tenant.md](tenant.md)
- [docs/system_design/platform/workspace.md](workspace.md)
- [docs/system_design/platform/resource-pool.md](resource-pool.md)
- [docs/system_design/platform/model.md](model.md)
- [docs/system_design/core/compute.md §6 Job](../core/compute.md#6-job)
- [docs/system_design/core/compute-operator.md §4 MLJob](../core/compute-operator.md#4-mljob-controller)
- [docs/system_design/core/cluster-manager.md](../core/cluster-manager.md)
- `auth.md`
