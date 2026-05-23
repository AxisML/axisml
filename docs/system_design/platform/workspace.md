# AxisML Platform 工作区 详细设计

本文档是 AxisML Platform 子系统下 **「训练 & 推理 → 工作区」** 一级功能的全栈设计，承接 [PRD §6.2.1 工作区](../../product/prd.md#621-工作区) 与系统层 [compute](../core/compute.md) / [compute-operator](../core/compute-operator.md) 之间的 Platform 入口：工作区列表 / 详情、创建与启停、浏览器接入与持久化目录管理。

工作区（Workspace）即 PRD 所称的「工作区」——一台 **长驻的交互式开发容器**，跑用户选定的镜像（jupyter / VSCode Server / 自定义 Web 服务等），用于代码调试、Notebook 试跑与小规模实验。本文不引入新的 CRD，整套语义复用 [`MLService(native, deployment)`](../core/compute-operator.md#561-native-deployment) 后端：创建 = 调 Compute 创建一个单 role 单副本的 MLService；启停 = patch `roles[0].replicas` 在 `0/1` 之间切换；浏览器接入 = 同一只 MLService 的 `spec.route` 派生 HTTPRoute 经 Envoy Gateway 反代。

| 模块 | 职责 | 边界外 |
| --- | --- | --- |
| 工作区视图与生命周期（[§4](#4-菜单与列表页) / [§7.1](#71-工作区-cruddelete)） | 创建 / 列表 / 详情 / 删除；按用户身份过滤可见工作区 | MLService CR 字段语义、Pod 调度、`(native, deployment)` handler 派生（→ [compute.md §7](../core/compute.md#7-service) / [compute-operator.md §5](../core/compute-operator.md#5-mlservice-controller)）|
| 启停（[§4.2](#42-操作按钮) / [§7.2](#72-启停)） | 用户视角的 start / stop 翻译为 Compute `/scale` 写 `replicas=1/0` | Compute 的双 hash 同步语义（→ [compute.md §3.5](../core/compute.md#35-desiredapplied-spec-hash-双-hash-机制)）|
| 浏览器接入（[Tab 2](#tab-2-访问) / [§7.3](#73-访问入口)） | 派生 `spec.route` 让 native/deployment handler 创建 HTTPRoute；下发短 TTL JWT | HTTPRoute / SecurityPolicy 资源渲染（→ [compute-operator.md §5.5 / §5.6.1](../core/compute-operator.md#55-specroute-派生资源)）|
| 持久化目录（[§5](#5-创建表单) / [§8.5](#85-pvc-管理)） | 创建 / 销毁 workspace 专属 PVC；MVP 默认挂 `/workspace` | StorageClass 选择、CSI driver 行为、跨节点共享（→ 集群运维；DataVolume 远期方案见 [overview.md §11](overview.md#11-后续迭代与-tbd)）|

**关键不变式：**

> **Platform 不为工作区建任何 PG 表**。「这个 service 是不是工作区」由 Compute `services.kind ∈ {service, workspace}` 列直接表达（详见 [core/compute.md §7.2](../core/compute.md#72-数据模型)）；「它属于哪个租户」由 `services.namespace` 直接给出；「它对应哪一行 Compute service」就是 service 本身。其他所有运行时 / 配置 / 镜像 / 资源信息全部由 Compute service 行承载，Platform 端无副本。
>
> 寻址用 Compute `services.id`（uuid）；Platform 列表 = `compute.ListServices(namespace=<tenant_ns>, kind=workspace)`，单次 RPC 拉完一个租户的全部工作区，天然 N+1 安全。
>
> Workspace 不抽象「IDE 模板」概念。镜像本身决定了它跑什么、监听哪个端口；Platform 仅透传 `image / containerPort / command / env`，不在数据模型上为「这是 jupyter 还是 vscode」做二级建模。

**文档组织：**

- **Part I — 服务边界**（[§1](#1-概述与定位) – [§3](#3-数据模型platform-自有部分)）：定位、可见性矩阵、Platform 自有数据模型与对 core 层的硬依赖。
- **Part II — UI 设计**（[§4](#4-菜单与列表页) – [§6](#6-详情页-tab)）：菜单 / 列表页 / 创建表单 / 详情页 Tab。
- **Part III — 后端 API 契约**（[§7](#7-rest-路径与响应格式)）：REST 路径、字段、RBAC。
- **Part IV — 后端实现**（[§8](#8-模块结构)）：模块结构、RBAC 装配、一致性策略、PVC 管理、JWT 颁发、可观测性。
- **Part V — 实施与验证**（[§9](#9-实现路径) – [§13](#13-相关引用)）：阶段化实现、关键决策、后续迭代、测试、参考。

---

## Part I — 服务边界

## 1. 概述与定位

「工作区」是「训练 & 推理」菜单下面向全部 persona（系统管理员 / 租户管理员 / 普通用户）的入口，覆盖以下能力：

- 选镜像 + 资源单元 + 容器端口创建一个工作区；
- 启动 / 停止单副本（`replicas=1/0`）以释放算力；
- 通过 Envoy Gateway 经路径前缀直接在浏览器打开 IDE；
- `/workspace` 目录由 Platform 申请的 PVC 承载，停机 / 重启不丢失；
- 删除时按需保留或一并销毁 PVC。

不在范围内的能力：

- MLService CR 字段语义、可变性约束、`(native, deployment)` 派生 Deployment / Service / HTTPRoute / SecurityPolicy / BackendTrafficPolicy 的细节：见 [compute-operator.md §5](../core/compute-operator.md#5-mlservice-controller)。
- Compute 的 outbox 写路径、双 hash 同步、status Informer 回流：见 [compute.md §3.4 / §3.5 / §7.3](../core/compute.md#34-写路径outbox--reconciler)。
- 镜像在 OCI 仓库的存储与 imagePullSecret 注入：见 [artifacts.md §5.3](../core/artifacts.md#53-image)。
- 跨节点共享数据卷 / 数据集挂载（`DataVolume`）：[overview.md §11](overview.md#11-后续迭代与-tbd) 标记为 TBD。
- 用户登录 / JWT 颁发 / 内置角色矩阵：见 `auth.md`。

## 2. 角色与可见性矩阵

下表只列与本功能相关的能力；persona ↔ RBAC 角色映射沿用 [platform/overview.md §2.2](overview.md#22-用户角色persona)。`@self` 表示「在该 workspace 所属租户上具备相应角色」；`@owner` 表示「该 workspace 的 owner_user 等于当前用户」。

| 能力 | `system-admin` | `tenant-admin@self` | `user@self & @owner` | `user@self & 非 owner` |
| --- | :---: | :---: | :---: | :---: |
| 列出可见工作区 | 全集群 | 该租户全部 | 仅自己创建的 | 仅自己创建的 |
| 查看工作区详情 | ✅ | ✅ | ✅ | ❌ |
| 创建工作区 | ✅ | ✅ | ✅ | — |
| 启动 / 停止 | ✅ | ✅ | ✅ | ❌ |
| 删除 | ✅ | ✅ | ✅ | ❌ |
| 拿 access JWT 进入 IDE | ✅ | ✅ | ✅ | ❌ |
| 修改展示元数据（display_name / description） | ✅ | ✅ | ✅ | ❌ |

`system-admin` 在所有动作上短路放行，不要求其在 `user_tenant_roles` 中显式绑定。`tenant-admin` 在自己绑定的租户内拥有等同于 `system-admin` 的工作区管理能力，覆盖所有成员的工作区；普通 `user` 仅能操作自己创建的工作区。

## 3. 数据模型（Platform 自有部分）

按 [overview.md §9 关键设计决策](overview.md#9-关键设计决策) 「Platform PG 范围 — 仅存身份、授权、会话、审计；不建任何视图缓存表」原则，**Platform 不为工作区建任何 PG 表**。「这是工作区」由 Compute `services.kind='workspace'` 直接表达，其他所有字段全部由 Compute service 行承载。

### 3.1 工作区元数据的归属

| 字段 | 写入位置 | 备注 |
| --- | --- | --- |
| 工作区身份 | Compute `services.id`（uuid） | URL 路径中的 `{id}` 即此列；同时 deterministic 派生 PVC 名 `axisml-ws-<id8>-data`（取 id 前 8 字符） |
| 「这是工作区」 | Compute `services.kind = 'workspace'` | Platform 创建时显式传；Compute 列表 / GET 端点响应必带此字段 |
| 租户归属 | Compute `services.namespace` | 即 Tenant CR `spec.namespace.name`；RBAC 裁剪从 `user_tenant_roles.tenant_name → Tenant.spec.namespace.name` 解析一次 |
| MLService CR 名 | Compute `services.name` | Platform 创建时生成 `mlservice_name = "ws-" + crockford32(rand40bit)` 作为 `services.name`；同 namespace 内冲突重试 |
| 显示名 / 描述 / Owner | Compute `services.{display_name, description, owner_user}` | `owner_user` 来自 `X-Axisml-User` 头注入 |
| Pool / Unit | Compute `services.{pool_id, resource_unit_id}` | |
| 镜像 / 端口 / 启动命令 / env | Compute `services.spec.roles[0].template.*` | |
| 配额（ElasticQuota CR 名） | Compute `services.spec.scheduling.quota` | |
| HTTPRoute 路径 | Compute `services.spec.route.path` | Platform 创建时拼 `/workspaces/<tenant>/<service.name>/` |
| 副本 / 就绪副本 / 端点 / 状态 | Compute `services.{replicas, ready_replicas, endpoint, status, message}` | Informer 回流 |
| `desired_state` / 工作区展示态 `status` | 派生：见 [§8.3](#83-状态读取与派生) 状态映射 |
| PVC 名 | 派生：`"axisml-ws-" + service.id 前 8 字符 + "-data"` | 无需存储 |
| PVC 容量 / 已用 | 详情页实时 `kubectl get pvc` + Prometheus（可选） | |
| Access URL | 派生：`https://<gateway-host>/workspaces/<tenant>/<service.name>/` | |
| 创建 / 更新时间 | Compute `services.{created_at, updated_at}` | |

### 3.2 列表查询路径

针对租户管理员 / 系统管理员的「列出整个租户的工作区」与普通用户的「列出我的工作区」两条路径，都走同一段流程：

1. 按 RBAC 取可见 `tenant_name` 集合；
2. 并行 `clustermanager.GetTenant(name)` 解析每个 `compute_namespace = Tenant.spec.namespace.name`（request-scoped memoize）；
3. 并行 `compute.ListServices(namespace=<compute_namespace>, kind=workspace, ownerUser?)` 单次 RPC 拉完每个租户的全部工作区；
4. 对普通用户由 Compute 端的 `ownerUser=<current_user.username>` 过滤直接下推（无内存二次过滤）；
5. 返回带派生 `status` / `desired_state` / `access_url` 的 Workspace DTO 列表。

整条路径完全无本地表 / 无 join；`kind=workspace` 过滤一次性把工作区与普通在线服务分开，列表 N+1 安全。

### 3.3 对 core 层的硬依赖（必须同 PR 推进）

本设计有两块对 core 层的扩展属于「不交付就实现不了」的硬依赖：

#### A. MLService spec 加 `volumes` / `volumeMounts`（PVC 持久化）

| 文件 / 章节 | 改动 |
| --- | --- |
| [compute-operator.md §5.2.2](../core/compute-operator.md#522-spec-结构) | 在 MLService `spec.roles[*].template` 下追加 `volumes[]` / `volumeMounts[]`，与 K8s `PodSpec` / `Container` 同源 |
| [compute-operator.md §5.2.3](../core/compute-operator.md#523-字段归属与不可变性) | 新增字段标 `用户提交 / 否`（工作区重启不需要改 PVC 引用） |
| [compute-operator.md §5.6.1](../core/compute-operator.md#561-native-deployment) 通用字段映射表 | 追加 `roles[].template.volumes` → `Deployment.spec.template.spec.volumes`；`roles[].template.volumeMounts` → 主容器 `volumeMounts` |
| [compute.md §7.2](../core/compute.md#72-数据模型) 字段归属 | `spec` jsonb 接受新增 `volumes` / `volumeMounts` 字段透传 |
| [compute.md §7.4.1](../core/compute.md#741-提交校验) 提交校验 | 增补「`volumeMounts.name` 必须在 `volumes[]` 中存在」校验 |

#### B. Compute service 加 `kind` 列 + id-based 寻址 / kind 过滤

| 文件 / 章节 | 改动 |
| --- | --- |
| [compute.md §7.2](../core/compute.md#72-数据模型) | services 表追加 `kind text NOT NULL DEFAULT 'service'`（`'service' \| 'workspace'`）；创建后不可变；不改变 backend handler / 状态机 / scale / delete 任何行为 |
| [compute.md §7.5](../core/compute.md#75-id-based-寻址端点--kind-过滤) | `POST /api/v1/namespaces/{ns}/services` 请求体接受 `kind`；`GET /api/v1/services/{id}` 返回 `kind`；`GET /api/v1/services?namespace=&kind=workspace` 支持 namespace + kind 过滤；所有 namespace-scoped LIST 支持 `?kind=` 过滤 |

写操作（`POST /scale`、`DELETE`）保留既有 namespace-scoped 形态——Platform 在写之前先 id-based GET 拿到 `(namespace, name)` 再走原路径，避免 Compute 引入双轨 API、写路径复用既有 outbox。

#### C. 对镜像的运行约束（不是 spec 改动，是约定）

Workspace 经 `https://<gateway>/workspaces/<tenant>/<service.name>/...` 访问，HTTPRoute **不做路径重写**；容器内的 web server 必须能在该前缀下工作：

- jupyter 启动需带 `--ServerApp.base_url=/workspaces/<tenant>/<service.name>/`；
- code-server 启动需带 `--abs-proxy-base-path /workspaces/<tenant>/<service.name>`；
- 自定义服务自负。

为减少用户负担，Platform 在创建 MLService 时给容器注入：

```
AXISML_WORKSPACE_BASE_URL=/workspaces/<tenant>/<service.name>/
```

约定 workspace 镜像的 entrypoint 读取该环境变量并自动配置 base-url；非约定镜像由用户在创建表单的「启动命令覆盖」字段中显式指定。Workspace 镜像目录（带正确 entrypoint 的 jupyter / vscode 启动脚本）由系统管理员在 Artifacts 镜像中心维护，作为运维文档（不属于本设计）。

---

## Part II — UI 设计

## 4. 菜单与列表页

菜单位置：「训练 & 推理 → 工作区」。

### 4.1 列表页

| 列 | 来源 | 说明 |
| --- | --- | --- |
| 显示名 | Compute `services.display_name` | 主展示列；点击进详情 |
| 镜像 | Compute `services.spec.roles[0].template.image` | 列过滤 |
| 资源池 · 单元 | Compute `services.pool_id` + `services.resource_unit_id` | 联表 ResourcePool / ResourceUnit 取展示名 |
| 状态 | 派生自 `(desired_state, services.status)`（[§8.3](#83-状态读取与派生)） | `Stopped` / `Starting` / `Running` / `Degraded` / `Failed` / `Deleting` / `Deleted` |
| Owner | Compute `services.owner_user` | 仅 admin 可见列 |
| 创建时间 | Compute `services.created_at` | |
| 操作 | — | 启动 / 停止 / 打开 / 详情 / 删除 |

- 过滤：状态、关键字（显示名 / 镜像名）、Owner（admin only）。
- 列表渲染：按 [§3.2](#32-列表查询路径) 走「按 RBAC 解析 compute_namespace → compute.ListServices(namespace, kind=workspace) 一次 RPC」的零本地表路径。
- 列表可见性：
  - `system-admin`：跨所有 compute namespace 并行 LIST（`kind=workspace`）；
  - `tenant-admin@self`：RBAC 可见租户对应的 compute namespace 集合；
  - `user`：上一条基础上由 Compute 端 `ownerUser=<current_user.username>` 过滤下推。

### 4.2 操作按钮

- **启动**（`Stopped` 状态）：调 `POST /api/v1/workspaces/{id}/start`；按钮 loading 至 `status` 转 `Running` 或 `Failed`。
- **停止**（`Running` / `Degraded` / `Starting` 状态）：调 `POST /api/v1/workspaces/{id}/stop`；二次确认弹窗提示「容器内 `/workspace` 之外的目录会丢失，PVC 数据保留」。
- **打开**（`Running` 状态）：前端先调 `GET /api/v1/workspaces/{id}/access` 拿 `{url, jwt}`，再用 `Authorization: Bearer <jwt>` 头通过新 tab 打开 `url`（推荐方式：临时 service worker 注入 header；退化方式：query string 透传 + 客户端清理）。
- **详情** — 任何在该 workspace 上有访问权的角色可点击。
- **删除** — owner / `tenant-admin@self` / `system-admin`；二次确认弹窗，body `{deletePvc: bool}`，默认勾选「同时删除 PVC」。

## 5. 创建表单

字段：

| UI 字段 | 写入位置 | 说明 |
| --- | --- | --- |
| 显示名 | Compute `services.display_name` | 必填；同一 owner 在同一租户下不重名（Platform Service 层 list + 去重校验） |
| 描述 | Compute `services.description` | 可选 |
| 租户 | Platform RBAC + 调用上下文（解析为 `compute_namespace`） | 默认当前活跃租户；`system-admin` 可选任意租户 |
| 镜像 | MLService `spec.roles[0].template.image` | 从 Artifacts `kind=image` 中选 + 可手填 OCI URI；后端解析校验能 resolve |
| 容器端口 | MLService `spec.roles[0].template.ports[0].containerPort` | 必填整数 ∈ [1, 65535]；用户应当知道镜像内 web server 的监听端口（如 jupyter 的 8888、code-server 的 8080） |
| 启动命令 / 参数 | MLService `spec.roles[0].template.command` / `args` | 可选；不填则使用镜像 entrypoint（依靠 `AXISML_WORKSPACE_BASE_URL` 自配置 base-url） |
| 环境变量 | MLService `spec.roles[0].template.env` | 可选；Platform 在末尾追加 `AXISML_WORKSPACE_BASE_URL` |
| 资源池 / 资源单元 | MLService `spec.scheduling.*` + Compute services `pool_id` / `resource_unit_id` | 沿用 [job.md / service.md](../core/compute.md) 同源校验：unit 必须属于 pool |
| 配额 | 拼接 ElasticQuota 名 `axisml-<tenant>-<pool>-<quota>` 写入 `spec.scheduling.quota` | 与 [overview.md §6.1](overview.md#61-用户提交训练任务) 一致 |
| PVC 大小 | 创建 PVC 时 `spec.resources.requests.storage` | 默认 `20Gi`；上限由 Platform 配置项约束（避免普通用户填 1Ti） |

校验：

- DNS-1123 显示名、长度 ≤ 40 字符（与 [tenant.md §5](tenant.md#5-创建租户表单system-admin-only) 一致）；
- 镜像必须能 `artifacts.Resolve`（image artifact 存在且对当前用户可见）；
- ResourceUnit 必须属于所选 ResourcePool（与 job/service 校验同源）；
- `containerPort` 范围 `[1, 65535]`；
- 跨字段约束 / 配额穿透校验最终以 Compute 返回为准。

**不引入「IDE 模板」抽象**：镜像本身决定 workspace 跑什么、监听哪个端口。镜像中心可由系统管理员维护一组「常用基础镜像」（如 `system/image/jupyter-cpu`、`system/image/jupyter-gpu`、`system/image/code-server`、`system/image/pytorch-jupyter`），让用户在选镜像时下拉看到——但这只是 Artifacts 侧的镜像目录管理，不是 Workspace 模型的一部分。后续也可以做「创建表单预设」让用户一键填好「镜像 + 启动命令 + 资源单元」（[§11](#11-后续迭代)），仍是前端语法糖、不入数据模型。

## 6. 详情页 Tab

详情页以 `id` 为维度，分为五个 Tab：概览、访问、事件、日志、审计。

### Tab 1 概览

展示：

- 基本信息：显示名 / 描述 / Owner / 镜像 / 资源池 · 单元（来自 Compute `services`，可编辑：仅 `display_name` / `description`）；
- 当前状态卡片：派生 `status`（`Running` / `Stopped` / ...）+ `replicas` / `ready_replicas` + Compute `services.message`；
- 启停时间：`last_started_at` / `last_stopped_at`（阶段一取自 `services.updated_at`，阶段二独立追踪）；
- PVC 信息：`pvc_name` / `pvc_size` / `pvc_used`（已用容量来自 Prometheus，可选；不可用时仅显示前两项）；
- 创建时间 / 创建者 / Tenant。

操作：

- **编辑展示元数据**（owner / admin）：仅写 Compute `services.display_name` / `description`；表单字段限于这两项。
- **启动 / 停止 / 删除**：与列表页同步逻辑。

### Tab 2 访问

展示：

- Access URL：`https://<gateway>/workspaces/<tenant>/<service.name>/`，复制按钮；
- JWT 用法说明：默认通过 service worker 注入 `Authorization: Bearer <jwt>` 头；客户端工具可直接拼接 query string `?access_token=<jwt>` 一次性进入；
- port-forward 命令模板（方便习惯命令行的用户）：
  ```
  kubectl -n <compute_namespace> port-forward svc/<service.name> 8888:<containerPort>
  ```
  本指南由前端按 detail 响应的 `compute_namespace` / `service.name` / `containerPort` 三个字段渲染。

操作：

- **打开**：拿 access JWT 后新 tab 打开（与列表页同步）；
- **轮换 access JWT**：调一次 `GET .../access` 即拿到新的 1h 有效期 token；不维护「禁用上一个」的状态机（短 TTL 自然失效）。

### Tab 3 事件

透传 Compute service events（`GET /api/v1/namespaces/{ns}/services/{svc}/events`）。Compute 当前 [§6.4.5](../core/compute.md#645-副本与事件端点) 仅给 Job 描述了 `/events` 端点，本设计在 [§11 后续迭代](#11-后续迭代) 标注「Compute 把 `/events` / `/replicas` / `/logs` 端点扩展到 service」作为后续工作。MVP 阶段 Tab 3 退化为「占位 + 跳到 K8s Events」（系统管理员可看），普通用户暂留空。

### Tab 4 日志

MVP 不交付。登记到 [§11 后续迭代](#11-后续迭代)：依赖 Compute service 端 `/logs` 端点扩展。短期建议用户用 `kubectl logs` 命令（Tab 2 的 port-forward 命令旁可附一行 `kubectl -n <ns> logs deploy/<service.name>`）。

### Tab 5 审计

入口保留，详细字段见 [§11 后续迭代](#11-后续迭代)，与 [tenant.md](tenant.md) 一致。

---

## Part III — 后端 API 契约

## 7. REST 路径与响应格式

- 所有路径统一在 `/api/v1/workspaces/...`；权限差异由 RBAC 中间件按角色 + 资源所有权判定。
- 错误格式：RFC 7807 problem+json，复用 [overview.md §7.3](overview.md#73-错误处理) 的样例。
- 出站调用：`internal/client/compute` typed client，定义见 [overview.md §7.5](overview.md#75-下游-typed-client)；通过 `GetServiceByID` / `ListServices(namespace, kind=workspace, ...)` 方法对接 [§3.3](#33-对-core-层的硬依赖必须同-pr-推进) 的 id-based + kind 过滤端点。
- 每个 endpoint 在下文括号内标注允许的角色：`system-admin` 表示全局；`tenant-admin@self` / `user@self` 表示「在该 workspace 所属租户上具备该角色」；`@owner` 表示「`services.owner_user == current_user.username`」。

### 7.1 工作区 CRUD（创建 / 读取 / 删除）

#### `POST /api/v1/workspaces`（已登录 + `user@self` 及以上）

请求体：

```json
{
  "tenantId": "...",
  "displayName": "ml-experiment-1",
  "description": "调试推理脚本",
  "image": "axisml.io/registry/namespaces/team-a/images/jupyter-cpu:2026-04",
  "containerPort": 8888,
  "command": [],
  "args": [],
  "env": [{"name": "PYTHONUNBUFFERED", "value": "1"}],
  "resourcePoolId": "...",
  "resourceUnitId": "...",
  "quota": "default",
  "pvcSize": "20Gi"
}
```

写入顺序：

1. RBAC + ResourcePool/Unit/Image 解析校验；通过 `clustermanager.GetTenant(tenantId)` 解析 `compute_namespace = Tenant.spec.namespace.name`；
2. 生成 `mlservice_name = "ws-" + crockford32(rand40bit)` 作为 desired CR 名；
3. 调 `compute.CreateService(compute_namespace, body)`，body 形如：
   ```jsonc
   {
     "name": "<mlservice_name>",
     "kind": "workspace",                                  // ★ 关键：把这条 service 标记为工作区
     "displayName": "<请求体 displayName>",
     "description": "<请求体 description>",
     "poolId": "<请求体 resourcePoolId>",
     "resourceUnitId": "<请求体 resourceUnitId>",
     "spec": {
       "scheduling": { "quota": "axisml-<tenant>-<pool>-<quota>" },
       "roles": [{
         "name": "predictor",
         "replicas": 1,
         "template": {
           "image": "<请求体 image>",
           "command": [...],
           "args": [...],
           "env": [
             "...用户输入...",
             { "name": "AXISML_WORKSPACE_BASE_URL",
               "value": "/workspaces/<tenant>/<mlservice_name>/" }
           ],
           "ports": [{
             "name": "web",
             "containerPort": "<请求体 containerPort>",
             "protocol": "TCP"
           }],
           "volumes": [{
             "name": "work",
             "persistentVolumeClaim": { "claimName": "<pvc_name>" }
           }],
           "volumeMounts": [{
             "name": "work",
             "mountPath": "/workspace"
           }]
         }
       }],
       "route": {
         "enabled": true,
         "targetRole": "predictor",
         "portName": "web",
         "path": "/workspaces/<tenant>/<mlservice_name>/",
         "auth": {
           "type": "jwt",
           "jwt": {
             "issuer": "axisml-platform",
             "jwksUri": "http://platform-backend.axisml-system.svc.cluster.local:8080/.well-known/jwks.json"
           }
         }
       }
     }
   }
   ```
4. Compute 同步返回 service 对象（含 Compute 生成的 `id` uuid）；
5. 由 service.id 派生 `pvc_name = "axisml-ws-" + service.id 前 8 字符 + "-data"`；Platform 直连 K8s 创建 PVC（[§8.5](#85-pvc-管理)）：`name=pvc_name`, `namespace=compute_namespace`, `accessModes=[ReadWriteOnce]`, `resources.requests.storage=<pvcSize>`, 默认 StorageClass；
6. PVC 创建失败 → 调 `compute.DeleteService(compute_namespace, mlservice_name)` 回滚 MLService，返 5xx；详见 [§8.2](#82-一致性策略pvc--mlservice)。

> **顺序倒置注记**：先建 MLService 再建 PVC 与原设计相反。原因：PVC 名 deterministic 派生自 service.id，而 service.id 由 Compute 生成，Platform 在 MLService 创建前拿不到该 id。这导致一个时间窗口：MLService 已存在但 PVC 未就绪——deployment 第一次拉起会因 PVC 不存在而 Pending，PVC 就位后自然进入 Running。可接受。

响应：`Workspace` DTO（结构与 [§7.1.GET](#get-apiv1workspacesid已登录按角色裁剪) 一致）。

#### `GET /api/v1/workspaces`（已登录，按角色裁剪）

按 [§3.2](#32-列表查询路径) 走 N+1 优化路径。支持 query：`status` / `tenantName` / `q`（关键字）/ `ownerUser`（admin only）/ `limit` / `continue`。

#### `GET /api/v1/workspaces/{id}`（`system-admin` 或 `tenant-admin@self` 或 `user@self & @owner`）

返回 `Workspace` DTO：

```json
{
  "id": "...",
  "tenantId": "...",
  "tenantName": "team-a",
  "computeNamespace": "team-a",
  "serviceId": "...",
  "name": "ws-x9k2a7bc",
  "displayName": "ml-experiment-1",
  "description": "...",
  "ownerUser": "alice",
  "image": "...",
  "containerPort": 8888,
  "resourcePoolId": "...",
  "resourceUnitId": "...",
  "quota": "axisml-team-a-default-default",
  "replicas": 1,
  "readyReplicas": 1,
  "endpoint": "ws-x9k2a7bc.team-a.svc.cluster.local:8888",
  "desiredState": "Running",
  "status": "Running",
  "message": "",
  "pvcName": "axisml-ws-abc12345-data",
  "pvcSize": "20Gi",
  "accessUrl": "https://gateway.axisml.io/workspaces/team-a/ws-x9k2a7bc/",
  "lastStartedAt": "...",
  "lastStoppedAt": null,
  "createdAt": "...",
  "updatedAt": "..."
}
```

合并器：直接以 Compute `GetServiceByID(id)` 返回为主体 + 可选的实时 PVC GET 派生 `pvc_size` / `pvc_used`。Platform 端无 PG 行参与合并。

#### `PATCH /api/v1/workspaces/{id}`（owner / `tenant-admin@self` / `system-admin`）

仅可改 `displayName` / `description`，写入 Compute `services.display_name` / `description`。其他字段（`image` / `containerPort` / `resourceUnit` / `pool` / `quota` / `pvcSize`）一律不可变；变更需「先删后建」，由前端明确引导。

#### `DELETE /api/v1/workspaces/{id}`（owner / `tenant-admin@self` / `system-admin`）

请求体：`{ "deletePvc": true }`（可选，默认 `true`）。

顺序：

1. `compute.GetServiceByID(id)` 拿 `(namespace, name)`；若返 `404` 直接返 200（幂等）；若 `kind != 'workspace'` 返 `404`（避免误删普通在线服务）；
2. 调 `compute.DeleteService(namespace, name)` 删 MLService；失败 → 4xx 透传 / 5xx 标 problem。
3. 视 `deletePvc`：
   - `true` → 直接 `kubectl delete pvc axisml-ws-<id8>-data`；
   - `false` → 保留 PVC，记录到「孤儿 PVC」管理界面（[§11](#11-后续迭代)）。

DELETE 是幂等的（步骤 1 的 404 路径）。

### 7.2 启停

| Endpoint | 方法 | 权限 | 说明 |
| --- | --- | --- | --- |
| `/api/v1/workspaces/{id}/start` | `POST` | owner / `tenant-admin@self` / `system-admin` | 翻译为 `compute.ScaleService(ns, name, {replicas: 1})` |
| `/api/v1/workspaces/{id}/stop` | `POST` | 同上 | 翻译为 `compute.ScaleService(ns, name, {replicas: 0})` |

行为：

- 先 `compute.GetServiceByID(id)` 拿 `(ns, name)` + 当前 `replicas`；若 `kind != 'workspace'` 返 `404`；
- 幂等保护：start 时若 `replicas == 1` 直接返 200；stop 时若 `replicas == 0` 直接返 200；
- `Deleted` / `Deleting` 状态拒绝 start / stop，返 `409 workspace-deleted`；
- 调 Compute `/scale`，按 [compute.md §7.4.2](../core/compute.md#742-扩缩容) 写后异步语义返回——不等 ready，调用方通过 `GET /workspaces/{id}` 轮询观察 `status`。

为什么用显式 `/start` `/stop` 而非通用 `/scale`：与 PRD「不用时停止释放资源」语义贴近、写权限模型清晰、UI 按钮 1:1 映射；底层翻译到 Compute `/scale` 这一步对用户透明。

### 7.3 访问入口

#### `GET /api/v1/workspaces/{id}/access`（owner / admin）

返回：

```json
{
  "url": "https://gateway.axisml.io/workspaces/team-a/ws-x9k2a7bc/",
  "jwt": "<short-TTL JWT>",
  "headerName": "Authorization",
  "expiresAt": "..."
}
```

JWT claim（详见 [§8.4](#84-jwt-与-jwks)）：

| claim | 值 |
| --- | --- |
| `iss` | `axisml-platform` |
| `aud` | `axisml-workspace` |
| `sub` | `<current-user-id>` |
| `workspace` | `<id>` |
| `service_id` | `<service_id>` |
| `exp` | `now + 1h`（默认） |

跳进去后镜像内的 web server 接管路由；base-url 通过 `AXISML_WORKSPACE_BASE_URL` env 已注入容器，符合 [§3.3 C](#33-对-core-层的硬依赖必须同-pr-推进) 的镜像约定即可正常工作。

### 7.4 事件 / 日志（占位）

| Endpoint | 方法 | 权限 | 说明 |
| --- | --- | --- | --- |
| `/api/v1/workspaces/{id}/events` | `GET` | 同 GET 详情 | 透传 `compute.GetServiceEvents(ns, name)`；MVP 阶段依赖 Compute 把 `/events` 端点扩展到 service（[§11](#11-后续迭代)） |
| `/api/v1/workspaces/{id}/logs` | `GET` | 同 GET 详情 | 同上 `/logs` 端点扩展（[§11](#11-后续迭代)） |

MVP 期间这两个端点可返回 `501 Not Implemented` + problem `type=https://axisml.io/errors/upstream-not-ready`，前端 Tab 3 / Tab 4 渲染占位提示。

---

## Part IV — 后端实现

## 8. 模块结构

目录：`components/platform/backend/internal/workspace/`

| 文件 | 职责 |
| --- | --- |
| `handler.go` | Gin 路由注册（`/api/v1/workspaces` 前缀）、请求解析、RBAC gate 装配、problem 渲染 |
| `service.go` | 业务编排：创建（MLService → PVC，PVC 失败时回滚 MLService）、启停（service_id GET → /scale）、删除（service_id GET → DELETE → PVC GC）、access JWT 颁发；状态完全由 `compute.GetServiceByID` 派生 |
| `dto.go` | 请求 / 响应类型；与 Compute API DTO 之间的显式映射 |
| `view.go` | `Workspace` DTO：`compute.GetServiceByID` 结果 + 可选的实时 PVC 数据 → 单一 DTO；含派生 `status` / `desired_state` / `access_url` 的纯函数 |
| `name.go` | `mlservice_name` 生成（crockford-base32 编码 40 位随机数；同 namespace 内冲突重试） |
| `pvc.go` | 受限 K8s client 创建 / 读取 / 删除 PVC；与 `service.go` 解耦以便单测 |
| `jwt.go` | access JWT 颁发；签名密钥与 [auth.md](auth.md) 复用 |

无 `repository.go`：Platform 不持有任何工作区相关 PG 表（[§3](#3-数据模型platform-自有部分)）。风格沿用 [overview.md §7.1](overview.md#71-仓库与目录布局) 与 [components/compute/](../../../components/compute/) 的 handler/service 两层。

无 `template.go`：本设计不引入「IDE 模板」抽象。镜像、端口、启动命令、环境变量由用户在创建表单显式提交，原样落到 MLService spec。

### 8.1 RBAC 中间件接入

`internal/auth` 提供 `RequireSystemAdmin` / `RequireTenantRole(role, tenantParam)` / `RequireWorkspaceOwner` 中间件，定义见 [auth.md](auth.md)。本文档使用：

| 路由 | 中间件链 |
| --- | --- |
| `GET /api/v1/workspaces` | 已登录即可；handler 内部按角色裁剪可见集合 |
| `POST /api/v1/workspaces` | `RequireTenantRole("user", "<body.tenantId>")`，`system-admin` / `tenant-admin@self` 短路 |
| `GET /api/v1/workspaces/{id}` | `RequireWorkspaceOwner("id")`；其语义为「`@owner` 或在 workspace 所属租户上具备 `tenant-admin` 以上角色」；`system-admin` 短路 |
| `PATCH /api/v1/workspaces/{id}`、`DELETE`、`POST .../start`、`POST .../stop`、`GET .../access` | 同上 |
| `GET .../events` / `GET .../logs` | 同上 |

### 8.2 一致性策略（PVC + MLService）

Compute / K8s 是权威，且 Platform 端不持有任何本地表——「上游写成功 / 本地写失败」这条失败路径不存在，原设计中的 inline retry + 持久化补偿队列全部不再需要。一致性策略简化为：

**创建**（按 [§7.1 POST](#post-apiv1workspaces已登录--userself-及以上) 顺序）：

1. MLService 创建失败 → 直接 4xx / 5xx 透传，无 PVC、无副作用。
2. MLService 创建成功后 PVC 创建失败 → 立即 `compute.DeleteService` 回滚 MLService，返 5xx；如果 delete MLService 也失败，登记到 `audit_logs` 由 system-admin 介入。
3. 二者均成功 → 返回 `Workspace` DTO。

**删除**（按 [§7.1 DELETE](#delete-apiv1workspacesidowner--tenant-adminself--system-admin) 顺序）：

1. `compute.GetServiceByID(id)`：若 `404` 或 `kind != 'workspace'` 返 `404`（幂等保护 + 防误删）；
2. 调 `compute.DeleteService` 失败 → 4xx / 5xx 透传；成功后继续；
3. PVC delete 失败 → 留作孤儿，登记到 [§11](#11-后续迭代)。

**启停**：单点写 Compute `/scale`，无跨对象副作用；失败直接透传 4xx / 5xx，无补偿。

**孤儿处理**：所有「这是工作区吗」的判定都依据 Compute `services.kind`——Platform 端没有本地行可能孤立。若 Platform 创建过程中 step 2 回滚也失败，会留下 `kind='workspace'` 但 PVC 缺失的孤儿 service；列表 GET 时其 `status` 会停留在 `Pending`，UI 上可见，system-admin 通过删除按钮清理。

### 8.3 状态读取与派生

- 任何展示运行态字段的请求都调 `compute.GetServiceByID(id)`，不本地缓存；不引入 K8s informer。
- 列表走 `compute.ListServices(namespace, kind=workspace)` 端点（[§3.3 B](#33-对-core-层的硬依赖必须同-pr-推进)），单租户百级 workspace 一次 RPC 拉完。
- 派生 `status` 规则（输入只剩 `(services.status, services.replicas, services.ready_replicas)`，无本地软删态）：

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

`Failed` / `Degraded` 是 Compute 的非终态（详见 [compute.md §7.3](../core/compute.md#73-状态机)），Platform 完全继承——operator 自愈后下一次 GET 自然回到 `Running`。

### 8.4 JWT 与 JWKS

- 复用 [auth.md](auth.md) 中 Platform 内置 JWT 签名密钥（不为 access JWT 单独配密钥）；
- access JWT TTL 默认 1h，可由 Platform 配置项 `--workspace-access-jwt-ttl` 调整，上限 24h；
- `aud=axisml-workspace` 与平台主用户 JWT (`aud=axisml-platform`) 区分：网关 SecurityPolicy 校验 `aud` claim 防止主用户 JWT 被滥用为工作区访问凭证；
- Platform 在 `axisml-system` namespace 内暴露 `/.well-known/jwks.json`（ClusterIP，无需走 Envoy Gateway），SecurityPolicy 通过 `cluster-local URL` 拉公钥；
- JWKS 端点的具体格式与密钥轮换策略 → [auth.md](auth.md)，本文不重复。

### 8.5 PVC 管理

- Platform 后端持有受限 K8s ServiceAccount，命名 `axisml-platform-pvc`，权限矩阵（CRD：核心 `persistentvolumeclaims` 资源）：
  ```
  apiGroups: [""]
  resources: ["persistentvolumeclaims"]
  verbs: ["get", "list", "create", "delete"]
  ```
  作用域：所有 compute namespace（即 Tenant CR 派生的 namespace）；通过 ClusterRole + 多 RoleBinding 落地（具体由 axisml-system Helm chart 渲染）。
- PVC spec：
  ```yaml
  spec:
    accessModes: [ReadWriteOnce]
    resources:
      requests:
        storage: <pvcSize>
    # storageClassName: 留空 → 用 cluster 默认；可由 --workspace-default-storage-class 覆盖
  ```
- 命名：`axisml-ws-<id 前 8 字符>-data`（deterministic，不入 PG）；同租户内不会撞名（id 是 uuid）。
- 删除策略（呼应 [§7.1 DELETE](#delete-apiv1workspacesidowner--tenant-adminself--system-admin)）：
  - `deletePvc=true`：workspace DELETE 时立即调 `kubectl delete pvc`；
  - `deletePvc=false`：保留 PVC；该 PVC 在「孤儿 PVC」管理界面里展示（[§11](#11-后续迭代) 后续上线），允许 system-admin 手动批量清理。

### 8.6 度量与日志

Prometheus 指标（特有于本功能；通用上游调用指标见 [overview.md §7.5](overview.md#75-下游-typed-client)）：

- `platform_workspace_action_total{action, status}`：`action ∈ {create, update_meta, start, stop, delete, access_token_issue}`，`status ∈ {success, failure}`。
- `platform_workspace_state{tenant_name, state}`：gauge，按租户聚合各派生 `status` 的 workspace 数；定期采样（拉 `compute.ListServices(namespace, kind=workspace)`）写入。
- `platform_workspace_pvc_orphan_total`：counter，记录 [§8.2](#82-一致性策略pvc--mlservice) 中 MLService 已删但 PVC 未能清理留下的孤儿 PVC 数。
- `platform_workspace_create_rollback_total{phase}`：counter，记录创建过程中 PVC 失败导致回滚 MLService 的次数；`phase ∈ {pvc_failed_mlservice_rolled_back, pvc_failed_mlservice_orphaned}`。
- `platform_workspace_access_jwt_issued_total{result}`：counter，access JWT 颁发量 + 失败原因。

zap 字段约定：每条 workspace 操作日志必带 `service_id` / `tenant_name` / `actor_user` / `action` / `status`；启停 / 删除额外带 `target_replicas` / `delete_pvc`（如适用）。

---

## Part V — 实施与验证

## 9. 实现路径

### 9.1 阶段一（MVP）

| 模块 | 范围 | 完成信号 |
| --- | --- | --- |
| handler / service / dto / view | [§7.1](#71-工作区-cruddelete) / [§7.2](#72-启停) / [§7.3](#73-访问入口) endpoint | `make platform-build` 通过 |
| RBAC 装配 | [§8.1](#81-rbac-中间件接入) 路由表全部接通；`system-admin` 短路与 `@owner` 校验单测覆盖 | 单元测试覆盖中间件分支 |
| PG 迁移 | 本模块无新增表；`migrate` 命令 no-op 通过即可 | `make platform-migrate` 干净 |
| PVC 受限 SA | ClusterRole + RoleBinding Helm 模板 | `kubectl auth can-i` 校验通过 |
| Compute `kind` 列与 kind 过滤端点 | [§3.3 B](#33-对-core-层的硬依赖必须同-pr-推进)：Compute 实现 `services.kind` 列；`POST` 接受 `kind`；`GET /api/v1/services/{id}` 返回 `kind`；`GET /api/v1/services?namespace=&kind=workspace` 与 namespace-scoped LIST 支持 `?kind=` 过滤 | Compute integration 覆盖 |
| MLService spec 扩展 | [§3.3 A](#33-对-core-层的硬依赖必须同-pr-推进)：MLService spec 加 `volumes` / `volumeMounts`；`(native, deployment)` handler 透传 | compute-operator integration 覆盖 |
| Integration | httptest 驱动 in-process gin + envtest 创建 PVC + httptest fake Compute；happy path：创建（jupyter 镜像）→ 启动 → MLService Ready → 停止 → 重新启动数据保留 → 删除（含 PVC） | `make platform-integration` 通过 |

### 9.2 阶段二

1. Compute service 端 `/events` / `/logs` 端点扩展 → 解锁详情页 Tab 3 / Tab 4。
2. 闲时自动 stop：基于 `services.updated_at` + Prometheus「无活跃连接」指标，定时把超时 workspace 自动 `replicas=0`。
3. 孤儿 PVC 清理 UI：「系统管理 → 孤儿 PVC」入口，列出 [§8.5](#85-pvc-管理) 中保留的 PVC，支持批量删除。
4. 列表分页 / 排序优化（按租户大小自适应）。

### 9.3 阶段三 / TBD

详见 [§11 后续迭代](#11-后续迭代)。

## 10. 关键设计决策

| 决策项 | 决策 | 理由 |
| --- | --- | --- |
| 底层 backend | 复用 `MLService(native, deployment)`，单 role `predictor` | 避免新增 CRD；与在线服务共用 backend handler 与 HTTPRoute 派生 |
| 工作区分类 | Compute `services.kind ∈ {service, workspace}` 列 | 1 个 enum 列代替 1 张表 + 三对象一致性补偿；Compute 行为对两种 kind 完全一致，仅供 Platform 列表过滤 |
| Platform PG 字段集 | **不为工作区建任何表** | 与 [overview.md §9](overview.md#9-关键设计决策) 「Platform PG 仅存身份与视图映射」对齐；杜绝双写漂移与补偿队列 |
| 寻址 | Compute `services.id`（uuid）；Platform URL 中的 `{id}` 直接等于 `services.id` | id 由 Compute 生成，Platform 端无独立 uuid |
| 不抽象 IDE 模板 | 镜像、端口、启动命令直接由用户提交 | 镜像本身决定 IDE；二级建模会带来 fragile 的「模板 ↔ 镜像」绑定 |
| 启停语义 | 显式 `POST /start` / `POST /stop` 两个 endpoint | 与 PRD「不用时停止释放资源」贴近；权限模型清晰；底层翻译为 Compute `/scale` 对用户透明 |
| 命名 | Platform 生成 `mlservice_name = "ws-" + crockford32(rand40bit)`；用户输入「显示名」走 `services.display_name` | 跨用户在共享 ns 无碰撞；用户视角的命名与底层 CR 名解耦 |
| 持久化 | MVP 内置 PVC；Platform 直管而非塞进 MLService handler；PVC 名 deterministic 派生自 `services.id` 前 8 字符 | 隔离 workspace 存储语义；MLService 仅负责 `volumes/volumeMounts` 透传，不引入 workspace 专用字段；id 来自 Compute，因此创建顺序为 MLService → PVC |
| 浏览器入口 | 路径前缀 HTTPRoute `/workspaces/<tenant>/<service.name>/`；JWT 鉴权 | 单一 Envoy Gateway，零额外 DNS / 证书；JWT 公钥来自 Platform 自身 |
| 镜像内 base-url 处理 | 注入 `AXISML_WORKSPACE_BASE_URL` env 让镜像 entrypoint 自配置 | 不需要在 HTTPRoute 上做路径重写；非约定镜像可由用户在「启动命令覆盖」字段自处理 |
| 状态派生 | 基于 `(services.status, services.replicas)` 二元组实时计算 | 不缓存避免漂移；与 Compute 的非终态语义一致 |
| 防误删 | DELETE 时校验 `kind='workspace'`，否则返 404 | 防止 workspace endpoint 被用来删普通在线服务 |
| 删除时 PVC 处置 | body `{deletePvc: bool}`，默认 `true` | 用户显式选择保留以便后续恢复；保留的 PVC 进孤儿管理界面 |
| Tab 3 / Tab 4 占位 | MVP 不交付事件 / 日志 Tab，依赖 Compute service 端端点扩展 | 避免在 Platform 引入 K8s 直读路径；与 Job 端点扩展同节奏推进 |

## 11. 后续迭代

- **Compute service `/events` / `/logs` / `/replicas` 端点扩展**：呼应 Job 端 [§6.4.5](../core/compute.md#645-副本与事件端点)；解锁详情页 Tab 3 / Tab 4。
- **闲时自动 stop**：基于 `services.updated_at` + 「无活跃连接」指标，超时自动 `replicas=0`；释放 GPU / 高规格资源单元。
- **孤儿 PVC 清理 UI**：见 [§8.5](#85-pvc-管理)。
- **审计日志 Tab**：复用 [overview.md §7.4](overview.md#74-pg-schema) 的 `audit_logs` 表，按 `target=workspace:{service_id}` 索引展示。
- **创建表单预设**：系统管理员维护一组「镜像 + 启动命令 + 资源单元」预设供普通用户一键填好；只是前端语法糖，不入数据模型。
- **SSH 接入**：通过 Platform tcp-proxy 或独立 SSH gateway 暴露 `:22`；与 HTTPRoute 路径方案并存。
- **多容器 Workspace**：同一 Pod 内 jupyter + tensorboard sidecar；需 MLService spec 支持多 init/sidecar container（非 role 维度）。
- **GPU 预热 / 镜像预拉**：常用镜像在节点上保活，启动时间从「分钟」级降到「秒」级。
- **DataVolume 集成**：把 `/workspace` 切到共享数据卷（跨 workspace / 跨用户共享），呼应 [overview.md §11](overview.md#11-后续迭代与-tbd) 的 DataVolume TBD。
- **cluster-manager 批量 GetTenant RPC**：减少 Platform 创建 / 启停 workspace 时多次解析租户 namespace 的 RPC 数。

## 12. 测试策略

- **单元**（`internal/workspace/*_test.go`）：
  - `view.go` 状态派生函数（[§8.3](#83-状态读取与派生) 的 8 个分支全覆盖）；
  - `name.go` mlservice_name 生成与冲突重试；
  - `service.go` 创建 / 启停的幂等性（重复 start `replicas==1` → 200）；
  - DELETE 防误删：`kind != 'workspace'` 返 404；
  - RBAC 中间件分支（`system-admin` 短路、`@owner` 比对、跨租户拒绝）；
  - access JWT claim 字段完整性。
- **integration**（`components/platform/backend/test/integration/`）：
  - testcontainers PostgreSQL（仅 `user_tenant_roles` / `audit_logs`）+ envtest（用于创建 PVC）+ in-process gin engine（`httptest`）+ httptest fake Compute（按 [§3.3 B](#33-对-core-层的硬依赖必须同-pr-推进) 模拟 `kind` 列与 kind 过滤端点）；
  - happy path：创建（jupyter 镜像，`kind='workspace'` 由 Platform 显式传）→ MLService 出现并 Ready → 停止 → MLService `replicas=0` → 重新启动 → PVC 数据保留校验 → 删除（含 / 不含 PVC）；
  - 故障注入：MLService 创建失败 → 不留 PVC；MLService 成功后 PVC 失败 → MLService 被回滚；DELETE 一个 `kind='service'` 的对象 → 404；
  - RBAC：`system-admin` / `tenant-admin@self` / `user@self & @owner` / `user@self & 非 owner` 在每个 endpoint 上的允许 / 拒绝矩阵；
  - 列表 N+1：构造 30 个 workspace + 10 个普通 service，断言列表只调一次 `compute.ListServices(namespace, kind=workspace)` 且只返回 30 个工作区。
- 不引入额外 minikube e2e：Platform 自身不直读 K8s API（PVC 管理是仅有的例外，已在 envtest 覆盖）；端到端 HTTPRoute / JWT 行为由 [compute-operator.md §7](../core/compute-operator.md#7-测试) 与 [infra.md](../infra/infra.md) 的 Envoy Gateway 测试链覆盖。

## 13. 相关引用

- [PRD §6.2.1 工作区](../../product/prd.md#621-工作区)
- [docs/system_design/overview.md](../overview.md)
- [docs/system_design/platform/overview.md §2.5 工作区 / 工作区](overview.md#25-工作区--工作区workspace)
- [docs/system_design/platform/tenant.md](tenant.md)
- [docs/system_design/core/compute.md §7 Service](../core/compute.md#7-service)
- [docs/system_design/core/compute-operator.md §5 MLService](../core/compute-operator.md#5-mlservice-controller)
- [docs/system_design/core/artifacts.md §5.3 image](../core/artifacts.md#53-image)
- [docs/system_design/infra/infra.md](../infra/infra.md)
- `auth.md`
