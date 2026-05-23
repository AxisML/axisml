# AxisML Platform 在线服务 详细设计

本文档是 AxisML Platform 子系统下 **「训练 & 推理 → 在线服务」** 一级功能的全栈设计，承接 [PRD §6.2.3 在线服务](../../product/prd.md#623-在线服务)。

「在线服务」（Online Service，本文简称 Service，避免与代码层 service 包名混淆时使用「在线服务」一词）即把一个已注册的模型版本以长驻 workload 形态部署出去，对外暴露 HTTP / gRPC 接口，承载推理流量。Platform 全部能力都通过调用 [compute](../core/compute.md) 的 Service 端点实现，自身不持有 Service 相关数据。

| 模块 | Platform 职责 | 调用下游 |
| --- | --- | --- |
| 在线服务视图与生命周期（[§4](#4-菜单与列表页) / [§7.1](#71-在线服务-cruddelete)） | 创建 / 列表 / 详情 / 删除；按用户身份过滤可见服务 | `compute.CreateService` / `GetServiceByID` / `ListServices(kind=service)` / `DeleteService` |
| 扩缩容（[§4.2](#42-操作按钮) / [§7.2](#72-扩缩容)） | 用户视角的 scale / start / stop 入口 | `compute.ScaleService` |
| 路由与访问（[Tab 2](#tab-2-访问) / [§7.3](#73-访问入口)） | 端点展示、`spec.route` 配置（鉴权 / 限流 / 超时）翻译为底层派生 HTTPRoute；可选 JWT 颁发 | `compute.GetServiceByID` + Platform JWKS |
| 流量与延迟指标（[Tab 5](#tab-5-指标) / [§7.4](#74-指标查询)） | 调 Prometheus 聚合 `request_rate` / `latency` / `error_rate` | `prometheus.Query` |
| 副本 / 事件 / 日志（[Tab 3](#tab-3-事件) – [Tab 4](#tab-4-日志) / [§7.5](#75-副本--事件--日志占位)） | 鉴权 + 字段透传（同 workspace 阶段二解锁） | `compute.GetServiceReplicas` / `GetServiceEvents` / `GetServiceLogs` |

**关键不变式：**

> Platform 自有 PG **不为在线服务建任何表**，与 [overview.md §7.1](overview.md#71-仓库与目录布局) 标注的「`internal/service/` 无本地表，仅代理」一致。「这是普通在线服务」由 Compute `services.kind='service'` 列直接表达（详见 [core/compute.md §7.2](../core/compute.md#72-数据模型)），与同一张表中的 `kind='workspace'` 行天然隔离；Platform 列表 = `compute.ListServices(namespace=<tenant_ns>, kind=service)`，单租户一次 RPC 拉完。
>
> 寻址用 Compute `services.id`（uuid）；URL `/api/v1/services/{id}`。与 [workspace.md](workspace.md) 共享 [§7.5 id-based 寻址 + kind 过滤端点](../core/compute.md#75-id-based-寻址端点--kind-过滤)，不引入 Platform 端独立 id。
>
> MLService spec 在 [compute](../core/compute.md) 侧除 `roles[*].replicas` 外不可变，因此「切换模型版本」= 新建一个 service + 灰度切流量 + 下线旧 service；Platform UI 提供「克隆为新版本」的反填语法糖，但底层是两次独立的创建 / 删除调用，**不**引入 Compute 不支持的 PATCH 路径。

**文档组织：**

- **Part I — 服务边界**（[§1](#1-概述与定位) – [§3](#3-数据模型platform-自有部分)）：定位、可见性矩阵、Platform 自有数据模型与对 core 层的硬依赖。
- **Part II — UI 设计**（[§4](#4-菜单与列表页) – [§6](#6-详情页-tab)）：菜单 / 列表页 / 创建表单 / 详情页 Tab。
- **Part III — 后端 API 契约**（[§7](#7-rest-路径与响应格式)）：REST 路径、字段、RBAC。
- **Part IV — 后端实现**（[§8](#8-模块结构)）：模块结构、RBAC 装配、上下文解析、可观测性。
- **Part V — 实施与验证**（[§9](#9-实现路径) – [§13](#13-相关引用)）：阶段化实现、关键决策、后续迭代、测试、参考。

---

## Part I — 服务边界

## 1. 概述与定位

「在线服务」是「训练 & 推理」菜单下面向普通用户（租户管理员 / 系统管理员同样可使用）的长驻 workload 入口，覆盖：

- 选模型版本 + backend / engine（默认 `(native, deployment)`）+ 镜像 + 资源池 + 资源单元 + 副本数 + 路由配置，部署一只在线推理服务；
- 列表 / 详情查看服务运行状态、副本就绪度、外部访问地址、流量与延迟指标；
- 扩缩容（`replicas` 0 ↔ N），其中 `replicas=0` 等价于「停服保留配置」；
- 修改服务展示元数据（display_name / description）；
- 删除服务（含 HTTPRoute 等派生资源由 ownerReference 级联清理）；
- 一键「克隆为新版本」生成同形参的新 service，配合外部流量切换实现版本灰度（MVP 不自动接管流量切换，UI 引导用户使用 [§5.3](#53-版本切换灰度发布)）。

不在范围内的能力：

- MLService CR 字段语义、可变性约束、`(native, deployment)` 派生 Deployment / Service / HTTPRoute / SecurityPolicy / BackendTrafficPolicy 的细节：见 [compute-operator.md §5](../core/compute-operator.md#5-mlservice-controller)。
- Compute 的 outbox 写路径、双 hash 同步、status Informer 回流：见 [compute.md §3.4 – §3.7](../core/compute.md#34-写路径outbox--reconciler)。
- 模型工件解析与 imagePullSecret 注入：见 [artifacts.md §5.1 / §5.3](../core/artifacts.md#5-artifact-kinds)。
- 用户登录 / JWT 颁发 / 内置角色矩阵：见 `auth.md`。
- 「工作区」kind 的 service：见 [workspace.md](workspace.md)；二者共用同一 Compute service 表，仅 `kind` 列与 Platform 入口不同。

## 2. 角色与可见性矩阵

下表只列与本功能相关的能力；persona ↔ RBAC 角色映射沿用 [platform/overview.md §2.2](overview.md#22-用户角色persona)。`@self` 表示「在该 service 所属租户上具备相应角色」；`@owner` 表示「`services.owner_user == current_user.username`」。

| 能力 | `system-admin` | `tenant-admin@self` | `user@self & @owner` | `user@self & 非 owner` |
| --- | :---: | :---: | :---: | :---: |
| 列出可见服务 | 全集群 | 该租户全部 | 仅自己创建的 | 仅自己创建的 |
| 查看服务详情 | ✅ | ✅ | ✅ | ❌ |
| 创建服务 | ✅ | ✅ | ✅ | — |
| 扩缩容 / start / stop | ✅ | ✅ | ✅ | ❌ |
| 修改展示元数据 | ✅ | ✅ | ✅ | ❌ |
| 删除 | ✅ | ✅ | ✅ | ❌ |
| 克隆为新版本 | ✅ | ✅ | ✅ | ❌ |
| 查看副本 / 事件 / 日志 | ✅ | ✅ | ✅ | ❌ |
| 查看流量与延迟指标 | ✅ | ✅ | ✅ | ❌ |

`system-admin` 在所有动作上短路放行，不要求其在 `user_tenant_roles` 中显式绑定。`tenant-admin` 在自己绑定的租户内拥有等同于 `system-admin` 的服务管理能力，覆盖所有成员的在线服务；普通 `user` 仅能操作自己创建的服务。

## 3. 数据模型（Platform 自有部分）

按 [overview.md §9 关键设计决策](overview.md#9-关键设计决策) 「Platform PG 范围 — 仅存身份、授权、会话、审计；不建任何视图缓存表」原则，**Platform 不为在线服务建任何 PG 表**。所有运行时 / 配置 / 镜像 / 模型 / 路由 / 资源信息全部由 Compute `services` 行（`kind='service'`）承载，Platform 端无副本。

### 3.1 在线服务元数据的归属

| 字段 | 写入位置 | 备注 |
| --- | --- | --- |
| 服务身份 | Compute `services.id`（uuid） | URL 路径中的 `{id}` 即此列 |
| 「这是普通在线服务」 | Compute `services.kind = 'service'` | Platform 创建时显式传；Compute 列表 / GET 端点响应必带此字段；Compute 自身不区分行为，仅供 Platform 列表过滤 |
| 租户归属 | Compute `services.namespace` | 即 Tenant CR `spec.namespace.name` |
| MLService CR 名 | Compute `services.name` | 用户在创建表单填「服务名」直接落到此列（DNS-1123，同 namespace 内唯一） |
| 显示名 / 描述 / Owner | Compute `services.{display_name, description, owner_user}` | `owner_user` 来自 `X-Axisml-User` 头注入 |
| Backend / Engine | Compute `services.spec.backend.{name, engine, config}` | MVP 默认 `(native, deployment)` |
| Pool / Unit | Compute `services.{pool_id, resource_unit_id}` | |
| 模型版本引用 | Compute `services.spec.modelRef.{name, version}` | Platform 创建前用 `artifacts.Resolve` 校验 |
| 镜像 / 端口 / 启动命令 / env | Compute `services.spec.roles[*].template.*` | |
| 配额（ElasticQuota CR 名） | Compute `services.spec.scheduling.quota` | Platform 写前拼接 `axisml-<tenant>-<pool>-<quota>` |
| 副本 / 期望副本 | Compute `services.replicas` + `spec.roles[0].replicas` | 单 role 约定下二者同步（`/scale` 写两者并重算 `desired_spec_hash`） |
| 就绪副本 / endpoint / 状态 | Compute `services.{ready_replicas, endpoint, status, message}` | Informer 回流 |
| 路由配置 | Compute `services.spec.route.{enabled, path, hostname, auth, rateLimit, timeout}` | Platform 在创建表单产出 |
| Access URL | 派生：`https://<gateway-host><services.spec.route.path>`（`route.enabled=true`）或 `services.endpoint`（关闭路由时回退内部 DNS） | |
| 流量 / 延迟 / 错误率 | 实时 Prometheus 查询；不入 PG | 详见 [§7.4](#74-指标查询) |
| 创建 / 更新时间 | Compute `services.{created_at, updated_at}` | |

### 3.2 列表查询路径

普通用户的「我的在线服务」与 admin 的「整个租户 / 全集群在线服务」走同一段流程：

1. 按 RBAC 取可见 `tenant_name` 集合（来自 `user_tenant_roles` 或 cluster-manager LIST，详见 [tenant.md §7.1](tenant.md#71-租户-crud)）；
2. 并行 `clustermanager.GetTenant(name)` 解析每个 `compute_namespace = Tenant.spec.namespace.name`（request-scoped memoize）；
3. 并行 `compute.ListServices(namespace=<compute_namespace>, kind=service, ownerUser?, status?, q?, limit, continue?)`；
4. 对普通用户由 Compute 端的 `ownerUser=<current_user.username>` 过滤直接下推（无内存二次过滤）；
5. 返回 `Service` DTO 列表（注入展示字段：tenant 名 / pool / unit 展示名 / accessUrl）。

整条路径完全无本地表 / 无 join；`kind=service` 过滤一次性把在线服务与工作区分开，列表 N+1 安全。跨租户合并仅 `system-admin` 触发。

### 3.3 对下游的依赖

| 调用 | 用途 |
| --- | --- |
| `clustermanager.GetTenant(name)` | 解析 `Tenant.spec.namespace.name` + `Tenant.spec.quotas[]`；校验 `status.phase == Active` |
| `compute.{Create,Scale,Delete}Service` + `GetServiceByID` + `ListServices` | Service CRUD + scale（[compute.md §7](../core/compute.md#7-service) 现有端点，无新增需求） |
| `compute.GetServiceReplicas/Events/Logs` | 阶段二解锁；与 [workspace.md §11](workspace.md#11-后续迭代) 同源等待 Compute 端扩展 |
| `compute.GetResourcePool(id)` | 把表单提交的 `resourcePoolId` (uuid) 翻译为 pool name，用于拼接 ElasticQuota CR 名 `axisml-<tenant>-<pool>-<quota>` 与校验该 quota 在 `Tenant.spec.quotas[poolName]` 内存在 |
| `artifacts.Resolve(namespace, "model", name, version)` | 校验 `modelRef.{name, version}` 可见且 Ready；并把解析结果（URI / digest / auth hint）回显给前端 |
| `artifacts.Resolve(namespace, "image", ...)` | 同上，针对镜像 |
| `prometheus.Query` / `QueryRange` | 详情页指标 Tab（[§7.4](#74-指标查询)） |

### 3.4 对 core 层的约束

本设计对 core 层 **无新增硬依赖**——所有 MVP 能力（创建 / 扩缩容 / 删除 / 路由 / kind 过滤 / id-based 寻址）均落在 [compute.md §7](../core/compute.md#7-service) 与 [compute-operator.md §5](../core/compute-operator.md#5-mlservice-controller) 现有契约内。下列项不属于硬依赖、但与本设计相关，集中登记如下：

- **`spec.route.stripPathPrefix`（计划，[§11](#11-后续迭代)）**：当前 `(native, deployment)` handler 派生的 HTTPRoute 不做路径重写——容器内的服务需要自行识别 `/services/<tenant>/<name>/` 前缀。MVP 通过注入 `AXISML_SERVICE_BASE_URL` 环境变量约定让镜像 entrypoint 自处理；少数推理框架（vLLM / Triton / TF Serving / TorchServe）不支持运行时 base path，则需要用户显式在 `command` / `args` 中传递路径参数，或在 [§11](#11-后续迭代) 推动 compute-operator 在 `spec.route` 上支持 `stripPathPrefix`。
- **Compute service `/events` / `/logs` / `/replicas` 端点扩展**：呼应 [workspace.md §11](workspace.md#11-后续迭代)；解锁详情页 Tab 3 / Tab 4。
- **多 role 独立扩缩**：解锁 KServe `(kserve, llminference)` 等 PD 分离 backend 的精细扩缩；登记在 [§11](#11-后续迭代) 与 [compute.md §7.7](../core/compute.md#77-后续工作) 同源等待。

---

## Part II — UI 设计

## 4. 菜单与列表页

菜单位置：「训练 & 推理 → 在线服务」。

### 4.1 列表页

| 列 | 来源 | 说明 |
| --- | --- | --- |
| 服务名 | `services.display_name`（fallback `services.name`） | 主展示列；点击进详情 |
| 模型 | `services.spec.modelRef.{name, version}` | 渲染为 `name@version`；列过滤 |
| 镜像 | `services.spec.roles[0].template.image` | 多 role 时取首个 role 镜像 + 角标 `+N` |
| 后端 | `services.spec.backend.{name, engine}` | 列过滤；MVP 仅 `(native, deployment)`，仍展示便于阶段二扩展 |
| 资源池 · 单元 | `services.pool_id` + `services.resource_unit_id` | 联表 ResourcePool / ResourceUnit 取展示名 |
| 副本 | `services.ready_replicas / services.replicas` | 形如 `3/3`；`replicas=0` 显示「已停服」 |
| 状态 | `services.status` 直接展示 | 状态集合见 [compute.md §7.3](../core/compute.md#73-状态机) |
| Owner | `services.owner_user` | 仅 admin 可见列 |
| 入口 | 派生 access URL | 仅展示 host + path 截断 + 复制按钮 |
| 创建时间 | `services.created_at` | |
| 操作 | — | 扩缩容 / 停 · 启 / 详情 / 克隆为新版本 / 删除 |

- 过滤：状态、关键字（服务名 / 镜像 / 模型名）、Owner（admin only）、backend、租户（admin 跨租户视图）。
- 列表渲染走 [§3.2](#32-列表查询路径) 的 N+1 安全路径。
- 列表可见性：
  - `system-admin`：默认仅展示自己「最近活跃」的租户视图，提供「切换租户」下拉跨租户浏览；
  - `tenant-admin@self`：可见租户对应的 compute namespace 集合；
  - `user`：上一条基础上由 Compute 端 `ownerUser=<current_user.username>` 过滤下推。
- 排序：默认 `created_at desc`；支持按 `ready_replicas` / `replicas` 排序便于排查异常。

### 4.2 操作按钮

- **扩缩容**：弹小输入框填 `replicas`（≥0 整数）；调 `POST .../scale`。replicas 0 表示停服。
- **停服**（`replicas > 0` 时可点）：语法糖 = scale 0；二次确认提示「将立即驱逐所有副本，路由与配置保留，可随时扩回 ≥1」。
- **启动**（`replicas == 0` 时可点）：语法糖 = scale 1（或表单填写）。
- **详情**：进入详情页。
- **克隆为新版本**：纯前端语法糖——把当前 `services.spec` 反填到创建表单，默认把 `modelRef.version` 字段聚焦让用户填新版本，提交后即新建一只 service；不调端点。详见 [§5.3](#53-版本切换灰度发布)。
- **删除**：owner / `tenant-admin@self` / `system-admin`；二次确认提示「将立即销毁所有副本、Service、HTTPRoute、SecurityPolicy、BackendTrafficPolicy；已建立的连接会断开」。

## 5. 创建在线服务表单

字段：

| UI 字段 | 写入位置 | 说明 |
| --- | --- | --- |
| 服务名 | `services.name`（同时是 MLService `metadata.name`） | 必填；DNS-1123 ≤40 字符；同一 `compute_namespace` 下唯一（含工作区） |
| 显示名 | `services.display_name` | 可选 |
| 描述 | `services.description` | 可选 |
| 租户 | RBAC + 路径变量；解析为 `compute_namespace` | 默认当前活跃租户；`system-admin` 可选任意租户 |
| 模型 | `services.spec.modelRef.{name, version}` | 必填；从 Artifacts `kind=model` 中选 → 版本下拉选 Ready 版本；后端用 `artifacts.Resolve` 校验 |
| 后端 / 引擎 | `services.spec.backend.{name, engine}` | 下拉；可选项见 [§5.1](#51-按后端渲染-roles)；MVP 默认且仅 `(native, deployment)` |
| 镜像 | MLService `spec.roles[0].template.image` | 必填；从 Artifacts `kind=image` 中选 + 可手填 OCI URI；后端用 `artifacts.Resolve` 校验 |
| 容器端口 | `spec.roles[0].template.ports[0].containerPort` | 必填整数 ∈ [1, 65535]；MVP 单端口，多端口在 [§11](#11-后续迭代) |
| 启动命令 / 参数 | `spec.roles[0].template.{command, args}` | 可选；不填则使用镜像 entrypoint（依靠 `AXISML_SERVICE_BASE_URL` / `AXISML_MODEL_URI` 自配置）|
| 环境变量 | `spec.roles[0].template.env` | 可选；Platform 在末尾追加 `AXISML_SERVICE_BASE_URL`（仅 `route.enabled=true` 时） |
| 资源池 / 资源单元 | `spec.scheduling.*` + `services.{pool_id, resource_unit_id}` | unit 必须属于 pool |
| 配额 | 拼接 `axisml-<tenant>-<pool>-<quota>` 写入 `spec.scheduling.quota` | 与 [overview.md §6.1](overview.md#61-用户提交训练任务) 一致 |
| 副本数 | `spec.roles[0].replicas` + `services.replicas`（单 role 约定） | 必填整数 ≥0；0 = 仅创建不拉起，后续手动 scale 起 |
| `progressDeadlineSeconds` | `spec.runPolicy.progressDeadlineSeconds` | 可选；rollout 超时（与 K8s Deployment 同名字段语义一致） |
| 路由 · 启用 | `spec.route.enabled` | 默认 `true`；关闭时仅内部 ClusterIP 暴露 |
| 路由 · 路径 | `spec.route.path` | Platform 自动拼 `/services/<tenant>/<service.name>/`；用户可在「高级」中覆盖（需自负唯一性） |
| 路由 · 主机名 | `spec.route.hostname` | 可选；默认走 Envoy Gateway 主域名 |
| 路由 · 鉴权 | `spec.route.auth.{type, jwt, apiKey}` | 默认 `jwt`，Platform 自带；切到 `apiKey` 时引用同租户 namespace 内的 K8s Secret；切到 `none` 时弹危险提示 |
| 路由 · 限流 | `spec.route.rateLimit.{requestsPerSecond, burst}` | 可选 |
| 路由 · 超时 | `spec.route.timeout` | 可选；ISO 8601 duration（如 `30s`） |

校验：

- DNS-1123 + 长度（[compute.md §3.2](../core/compute.md#32-pg-编排约定)）；
- 服务名与同 namespace 内已有 service / workspace 不重名；
- 模型 / 镜像必须能被 `artifacts.Resolve` 且对当前用户可见 + 模型版本 `status=Ready`；
- ResourceUnit 必须属于所选 ResourcePool；
- `quota` 必须出现在 `Tenant.spec.quotas[resourcePoolId]`（Platform 用 `clustermanager.GetTenant` 校验）；
- `containerPort` 范围 `[1, 65535]`；
- 路径前缀必须以 `/` 开头、不以 `/` 结尾外加 `/<service.name>/` 收尾（自动补，避免与其他 service 冲突）；
- `route.auth.type=apiKey` 时 `secretRef.name` 引用的 Secret 必须存在于 `compute_namespace` 内（系统管理员预置）；
- 其他跨字段约束以 Compute / compute-operator 返回为准。

### 5.1 按后端渲染 Roles

每个 backend / engine 对应一组固定 role（由 [compute-operator.md §5.6](../core/compute-operator.md#56-内置-handler) 约定），前端据此渲染：

| backend / engine | role 集合 | 阶段 |
| --- | --- | --- |
| `(native, deployment)` | `predictor`（replicas≥0） | 阶段一（MVP） |
| `(native, statefulset)` | `predictor`（replicas≥0，副本身份稳定） | 阶段二 |
| `(kserve, inference)` | `predictor`（replicas≥0；`backend.config.runtime` 决定 runtime） | 阶段二 |
| `(kserve, llminference)` | `prefill` + `decode` + `router`（PD 分离） | 阶段三 / TBD |
| `(custom, *)` | 由 `backend.config` 决定 | 阶段三 / TBD |

每个 role 块包含：副本数 / 镜像 / 容器端口 / 启动命令 / 参数 / 环境变量 / restartPolicy。MVP 单 role 单端口，多 role / 多端口在 [§11](#11-后续迭代)。

### 5.2 不在创建表单中的字段

- `spec.backend.config`：MVP 不开放；阶段二 `(kserve, *)` 上线时按需开放（如 vLLM 的 `tensorParallelSize`）。
- `spec.scheduling.priorityClass`：MVP 统一走 ResourcePool 默认。
- `nodeSelector` / `tolerations`：由 ResourcePool + ResourceUnit 自动合并（见 [compute.md §5.4](../core/compute.md#54-注入规则)）。
- `volumes` / `volumeMounts`：MVP 不开放——在线服务的模型工件通过 `spec.modelRef` 由 handler 解析注入 `AXISML_MODEL_URI` 环境变量；不需要 PVC。若用户有缓存盘 / 共享配置需求，作为 [§11](#11-后续迭代) DataVolume 集成项推进。
- `spec.modelRef` 不可变：「切换模型版本」走 [§5.3](#53-版本切换灰度发布) 的 clone-with-new-version 路径。

### 5.3 版本切换 / 灰度发布

MLService spec.modelRef 创建后不可变（[compute-operator.md §5.2.3](../core/compute-operator.md#523-字段归属与不可变性)）。「切换模型版本」在 Platform 上是：

1. 在旧 service 列表 / 详情页点「克隆为新版本」→ 前端把 `services.spec` 反填到创建表单，把 `modelRef.version` 聚焦让用户填新版本；服务名按 `<旧 name>-<新 version 简写>` 模板默认填好（用户可改）；
2. 提交后即建立一只新的 service，独立 endpoint、独立 HTTPRoute（路径前缀里带新 service.name）；
3. 用户通过外部流量切换（DNS / 反向代理 / 网关上的 weighted route 或 host header switch）从旧 service 切到新 service；
4. 验证完成后到旧 service 详情页点「停服」（scale 0）观察一段时间，再点删除。

MVP **不**接管 weighted route / canary / 金丝雀的 UI；用户自助通过外部入口或在 cluster 内直接调路径切流。完整灰度策略（同一路径前缀下 weighted 切分、自动指标判定回滚）作为 [§11](#11-后续迭代) 「流量切换与灰度」项推进。

## 6. 详情页 Tab

详情页以 `id` 为维度，分为六个 Tab：概览、访问、事件、日志、指标、审计。

### Tab 1 概览

展示（全部来自 `compute.GetServiceByID` 响应，全只读除明确「可编辑」字段）：

- 基本信息：显示名 / 描述 / Owner / Tenant（可编辑：仅 `display_name` / `description`）；
- 模型 / 镜像 / 后端 / 资源池 · 单元（只读）；
- 当前状态卡片：`status` + `ready_replicas / replicas` + Compute `services.message`；
- 副本概览：`roles[predictor]` 的 `replicas` / `readyReplicas`；
- 路由概览：`route.path` / `route.hostname` / `route.auth.type` / 限流 / 超时；
- 时间线：`createdAt` / `updatedAt` / `lastScaledAt`（阶段一取自 `services.updated_at`，阶段二独立追踪）。

操作：

- **扩缩容 / 停服 / 启动**：与列表页同步逻辑；
- **编辑展示元数据**：仅写 Compute `services.display_name` / `description`；表单字段限于这两项；
- **克隆为新版本**：跳到 [§5.3](#53-版本切换灰度发布)；
- **删除**：与列表页同步；
- **复制 YAML**：把 `services.spec` 渲染成等价 MLService YAML 供运维参考。

### Tab 2 访问

展示：

- Access URL：`https://<gateway><services.spec.route.path>`，复制按钮；当 `route.enabled=false` 时退化为 `<services.endpoint>` 内部 DNS；
- 鉴权说明：
  - `route.auth.type=none`：标注「公开访问，建议仅用于内部调试」+ 红色警示；
  - `route.auth.type=jwt`：展示 Platform 颁发的 issuer / audience（`axisml-inference`）+ 调一次 `GET .../access` 拿一次性 1h JWT 的按钮；
  - `route.auth.type=apiKey`：展示 Secret 名 + 字段（系统管理员维护）+ 「轮换」入口（[§11](#11-后续迭代)）；
- 调用示例：按当前路由配置生成 `curl` / `python` / `grpcurl` 三段示例（pre-fill header / endpoint / model 名）；
- port-forward 命令模板（开发期备用）：
  ```
  kubectl -n <compute_namespace> port-forward svc/<service.name> 8080:<containerPort>
  ```
  由前端按 detail 响应的 `compute_namespace` / `service.name` / `containerPort` 渲染。

操作：

- **拿 access JWT**（`auth.type=jwt`）：调 `GET .../access` 拿 `{url, jwt, expiresAt}`，弹模态框展示并提供「复制」按钮；语义见 [§7.3](#73-访问入口)。

### Tab 3 事件

透传 Compute service events（`GET /api/v1/namespaces/{ns}/services/{svc}/events`）；MVP 阶段同 [workspace.md §6 Tab 3](workspace.md#tab-3-事件)：依赖 Compute 端 `/events` 端点扩展到 service（[§11](#11-后续迭代)）。MVP 期间该 Tab 退化为「占位 + 跳到 K8s Events」（系统管理员可看），普通用户暂留空提示「即将开放」。

### Tab 4 日志

同 workspace —— MVP 不交付，依赖 Compute service 端 `/logs` 端点扩展（[§11](#11-后续迭代)）。短期建议用户走 `kubectl logs deploy/<service.name>` 命令；Tab 2 中已在 port-forward 命令旁附 `kubectl logs` 命令。

### Tab 5 指标

展示（来自 Prometheus 实时查询，详见 [§7.4](#74-指标查询)）：

- **流量**：每秒请求数 `request_rate`（istio_requests_total / envoy_http_downstream_rq_total / 自定义 backend metrics，按 backend 选择，详见 [§8.5](#85-prometheus-查询模板)）；
- **延迟**：p50 / p95 / p99；
- **错误率**：5xx 比例 + 4xx 比例（分离展示，4xx 通常是调用方问题）；
- **饱和**：副本 CPU / 内存 / GPU 利用率（kube-prometheus 标准指标）；
- **LLM 专项**（`backend=kserve, engine=llminference` 时，阶段三）：tokens/sec、TTFT、TBT、KV cache 占用——MVP 不交付，占位入口。

时间窗口选择器：5m / 15m / 1h / 6h / 24h；自动刷新 15s。

指标 Tab 与其他 Tab 一样仅展示数据；调优 / 告警阈值在「系统管理 → 监控告警」（[§11](#11-后续迭代)）独立菜单维护。

### Tab 6 审计

入口保留，MVP 不交付；登记到 [§11](#11-后续迭代)，与 [job.md Tab 5](job.md#tab-5-审计) / [workspace.md Tab 5](workspace.md#tab-5-审计) 一致——按 `target=service:{service_id}` 索引展示。

---

## Part III — 后端 API 契约

## 7. REST 路径与响应格式

- 所有路径统一在 `/api/v1/services/...`；权限差异由 RBAC 中间件按角色 + 资源所有权判定。
- 错误格式：RFC 7807 problem+json，复用 [overview.md §7.3](overview.md#73-错误处理) 的样例；Compute / cluster-manager / Artifacts 返回的 problem 原样透传。
- 出站调用：`internal/client/compute` typed client，方法集为 `CreateService` / `GetServiceByID` / `ListServices` / `ScaleService` / `DeleteService` / `GetServiceReplicas` / `GetServiceEvents` / `GetServiceLogs`。
- 每个 endpoint 在下文括号内标注允许的角色：`system-admin` 表示全局；`tenant-admin@self` / `user@self` 表示「在该 service 所属租户上具备该角色」；`@owner` 表示「`services.owner_user == current_user.username`」。

### 7.1 在线服务 CRUD（创建 / 读取 / 删除）

#### `POST /api/v1/services`（已登录 + `user@self` 及以上）

请求体：

```json
{
  "tenantId": "...",
  "name": "bert-base-finetuned-v1",
  "displayName": "BERT Base 微调 v1 推理服务",
  "description": "针对 v3 数据集微调后的在线推理服务",
  "modelRef": { "name": "bert-base-finetuned", "version": "v1" },
  "backend": { "name": "native", "engine": "deployment", "config": {} },
  "image": "axisml.io/registry/namespaces/team-a/images/pytorch-serve:2026-04",
  "containerPort": 8080,
  "command": [],
  "args": ["serve", "--model-dir", "$(AXISML_MODEL_URI)"],
  "env": [{"name": "TORCH_HOME", "value": "/tmp/torch"}],
  "resourcePoolId": "...",
  "resourceUnitId": "...",
  "quota": "default",
  "replicas": 2,
  "progressDeadlineSeconds": 600,
  "route": {
    "enabled": true,
    "path": "",
    "hostname": "",
    "auth": { "type": "jwt" },
    "rateLimit": { "requestsPerSecond": 100, "burst": 200 },
    "timeout": "30s"
  }
}
```

处理顺序：

1. RBAC + 提交校验（[§5](#5-创建在线服务表单) 校验列表）；
2. `clustermanager.GetTenant(tenantId)` 解析 `compute_namespace = Tenant.spec.namespace.name` 与可用 quota；
3. 解析 `modelRef` / `image` 经 `artifacts.Resolve` 校验；
4. 拼接 `spec.scheduling.quota = axisml-<tenant>-<pool>-<quota>`；
5. 若 `route.enabled=true` 且 `route.path == ""`，Platform 自动拼 `route.path = /services/<tenant>/<name>/`，并在 env 末尾追加 `AXISML_SERVICE_BASE_URL=<route.path>`；
6. 调 `compute.CreateService(compute_namespace, body)`，body 形如：
   ```jsonc
   {
     "name": "<请求体 name>",
     "kind": "service",                                  // ★ 关键：把这条 service 标记为普通在线服务
     "displayName": "<请求体 displayName>",
     "description": "<请求体 description>",
     "poolId": "<请求体 resourcePoolId>",
     "resourceUnitId": "<请求体 resourceUnitId>",
     "spec": {
       "backend": { "name": "native", "engine": "deployment" },
       "scheduling": { "quota": "axisml-<tenant>-<pool>-<quota>" },
       "modelRef": { "name": "<...>", "version": "<...>" },
       "roles": [{
         "name": "predictor",
         "replicas": "<请求体 replicas>",
         "template": {
           "image": "<请求体 image>",
           "command": [...],
           "args": [...],
           "env": [
             "...用户输入...",
             { "name": "AXISML_SERVICE_BASE_URL", "value": "<route.path>" }
           ],
           "ports": [{ "name": "http", "containerPort": "<请求体 containerPort>", "protocol": "TCP" }]
         }
       }],
       "runPolicy": { "progressDeadlineSeconds": 600 },
       "route": {
         "enabled": true,
         "targetRole": "predictor",
         "portName": "http",
         "path": "<route.path>",
         "auth": { "type": "jwt", "jwt": {
           "issuer": "axisml-platform",
           "jwksUri": "http://platform-backend.axisml-system.svc.cluster.local:8080/.well-known/jwks.json"
         } },
         "rateLimit": "<request.route.rateLimit>",
         "timeout": "<request.route.timeout>"
       }
     }
   }
   ```
7. Compute 同步返回 service 对象（含 Compute 生成的 `id` uuid）；
8. 透传响应为 `Service` DTO。

失败语义：`400` 提交校验失败；`404` tenant 不存在 / `Tenant.status.phase != Active` / 模型或镜像 artifact 不存在；`409` 同 namespace 下已有未软删的同名 service 或 workspace；`5xx` 透传下游 problem。

#### `GET /api/v1/services`（已登录，按角色裁剪）

按 [§3.2](#32-列表查询路径) 走 N+1 优化路径。query 参数分两类：

- **下推 Compute**（分页与过滤都精确）：`tenantName` / `status` / `ownerUser`（admin only）/ `limit` / `continue`——原样透传到 `compute.ListServices(namespace, kind=service, ...)`。
- **Platform 内存过滤**（在透传结果上做二次筛选，对分页结果不精确）：`q`（关键字，匹配 name / displayName / image / modelRef.name）/ `backendName` / `backendEngine` / `modelName`。

`q` 与 backend / modelName 过滤的下推已登记在 [§11](#11-后续迭代) 「列表过滤下推」中——MVP 阶段使用这几个参数时应配合较大的 `limit`，避免分页截断遗漏。响应可能附 `partial: true` 标志，详见 [§8.3](#83-失败语义)。

注：本端点同时覆盖 tenant-scoped 与跨租户视图——`tenantName` 不填且当前用户为 `system-admin` 时走全集群并行 LIST + 合并；普通用户必须在自己绑定的 tenant 集合内（handler 自动按 RBAC 限定）。

#### `GET /api/v1/services/{id}`（`system-admin` 或 `tenant-admin@self` 或 `user@self & @owner`）

返回 `Service` DTO：

```json
{
  "id": "...",
  "tenantId": "...",
  "tenantName": "team-a",
  "tenantDisplayName": "Team A",
  "computeNamespace": "team-a",
  "name": "bert-base-finetuned-v1",
  "displayName": "...",
  "description": "...",
  "ownerUser": "alice",
  "modelRef": { "name": "bert-base-finetuned", "version": "v1" },
  "backend": { "name": "native", "engine": "deployment", "config": {} },
  "image": "...",
  "containerPort": 8080,
  "resourcePoolId": "...",
  "resourcePoolName": "gpu-a100",
  "resourceUnitId": "...",
  "resourceUnitName": "a100-1x-medium",
  "quota": "axisml-team-a-default-default",
  "replicas": 2,
  "readyReplicas": 2,
  "endpoint": "bert-base-finetuned-v1.team-a.svc.cluster.local:8080",
  "route": {
    "enabled": true,
    "path": "/services/team-a/bert-base-finetuned-v1/",
    "hostname": "",
    "auth": { "type": "jwt" },
    "rateLimit": { "requestsPerSecond": 100, "burst": 200 },
    "timeout": "30s"
  },
  "accessUrl": "https://gateway.axisml.io/services/team-a/bert-base-finetuned-v1/",
  "status": "Ready",
  "message": "",
  "createdAt": "...",
  "updatedAt": "..."
}
```

合并器：直接以 `compute.GetServiceByID(id)` 返回为主体 + Platform 注入的展示字段（tenant 名 / pool 名 / unit 名 / accessUrl）。

#### `PATCH /api/v1/services/{id}`（owner / `tenant-admin@self` / `system-admin`）

仅可改 `displayName` / `description`，写入 Compute `services.display_name` / `description`。其他字段（`modelRef` / `image` / `containerPort` / `resourceUnit` / `pool` / `quota` / `route`）一律不可变；变更需走 [§5.3](#53-版本切换灰度发布) clone-with-new-version 路径，由前端明确引导。

> 路由配置（`spec.route` 整块）当前不可变——`auth` / `rateLimit` / `timeout` 热更新依赖 compute-operator 的 [§5.7 后续工作](../core/compute-operator.md#57-后续工作) 「`spec.route` 可热更新路径」。在该能力上线前，更新路由也需要走 clone-with-new-version + 流量切换。

#### `DELETE /api/v1/services/{id}`（owner / `tenant-admin@self` / `system-admin`）

顺序：

1. `compute.GetServiceByID(id)`：若返 `404` 直接返 200（幂等）；若 `kind != 'service'` 返 `404`（避免误删工作区）；
2. 调 `compute.DeleteService(namespace, name)` 删 MLService；失败 → 4xx 透传 / 5xx 标 problem。

派生的 K8s Service / HTTPRoute / SecurityPolicy / BackendTrafficPolicy 由 ownerReference 级联清理，无需 Platform 介入。DELETE 是幂等的（步骤 1 的 404 路径）。

### 7.2 扩缩容

| Endpoint | 方法 | 权限 | 说明 |
| --- | --- | --- | --- |
| `/api/v1/services/{id}/scale` | `POST` | owner / `tenant-admin@self` / `system-admin` | 翻译为 `compute.ScaleService(ns, name, {replicas})` |
| `/api/v1/services/{id}/start` | `POST` | 同上 | 语法糖：scale 到「上一次 >0 的 replicas」（来自 Compute 的 audit_logs / Platform `audit_logs` 表反查）或默认 1 |
| `/api/v1/services/{id}/stop` | `POST` | 同上 | 语法糖：scale 0 |

行为：

- `/scale` 请求体：`{"replicas": <int>}`，必须 ≥0；先 `compute.GetServiceByID(id)` 拿 `(ns, name)`，若 `kind != 'service'` 返 `404`；
- 幂等保护：`replicas == 当前 replicas` 时直接返 200；
- `Deleted` / `Deleting` 状态拒绝 scale，返 `409 service-deleted`；
- 调 Compute `/scale`，按 [compute.md §7.4.2](../core/compute.md#742-扩缩容) 写后异步语义返回——不等 ready，调用方通过 `GET /services/{id}` 轮询观察 `status`；
- `/start` 端点需要查 Platform 自身 `audit_logs` 表（[overview.md §7.4](overview.md#74-pg-schema)）拿最近一次成功 scale 的非零 replicas 作为目标值；若 audit 缺失（如首次部署 replicas=0），fallback 到 `1`。

`/start` 与 `/stop` 是 [§4.2](#42-操作按钮) UI 按钮的便捷端点；底层翻译到 `/scale` 对用户透明。

### 7.3 访问入口

#### `GET /api/v1/services/{id}/access`（owner / `tenant-admin@self` / `system-admin`）

仅当 `route.enabled=true && route.auth.type=jwt` 时有意义；其他情况下返回 `409 route-auth-mismatch` + problem，提示用户当前服务的鉴权类型不需要从 Platform 拿 JWT。

返回：

```json
{
  "url": "https://gateway.axisml.io/services/team-a/bert-base-finetuned-v1/",
  "jwt": "<short-TTL JWT>",
  "headerName": "Authorization",
  "expiresAt": "..."
}
```

JWT claim（详见 [§8.4](#84-jwt-与-jwks)）：

| claim | 值 |
| --- | --- |
| `iss` | `axisml-platform` |
| `aud` | `axisml-inference`（与 workspace 的 `axisml-workspace` 区分） |
| `sub` | `<current-user-id>` |
| `service_id` | `<id>` |
| `tenant` | `<tenant_name>` |
| `exp` | `now + 1h`（默认；上限由 `--service-access-jwt-ttl` 配） |

通过返回的 JWT 在 `Authorization: Bearer <jwt>` 头上调用服务即可；网关侧 SecurityPolicy 校验 `aud=axisml-inference` 防止跨用途滥用（如把主用户 JWT 用作服务调用凭证）。

### 7.4 指标查询

#### `GET /api/v1/services/{id}/metrics`（同 GET 详情）

query 参数：

| 参数 | 说明 |
| --- | --- |
| `metric` | 必填；枚举：`request_rate` / `latency` / `error_rate` / `cpu_util` / `mem_util` / `gpu_util` |
| `range` | 必填；ISO 8601 duration（`5m` / `15m` / `1h` / `6h` / `24h`） |
| `step` | 可选；查询步长，默认按 range 自动选 |
| `percentile` | 仅 `metric=latency` 时；`p50` / `p95` / `p99`，默认 `p95` |

处理顺序：

1. RBAC 同 GET 详情；
2. `compute.GetServiceByID(id)` 拿 `(namespace, name)` + backend；
3. 按 backend 选择 Prometheus 查询模板（[§8.5](#85-prometheus-查询模板)）；
4. 透传 Prometheus `Query` / `QueryRange` 响应为 `MetricSeries` DTO：
   ```json
   {
     "metric": "latency",
     "range": "15m",
     "percentile": "p95",
     "unit": "ms",
     "series": [
       { "timestamp": "...", "value": 47.2 },
       { "timestamp": "...", "value": 52.1 }
     ]
   }
   ```

Prometheus 查询失败时返 `502 upstream-failure` problem，前端 Tab 5 显示「指标数据不可用」+ 重试按钮。

### 7.5 副本 / 事件 / 日志（占位）

| Endpoint | 方法 | 权限 | 说明 |
| --- | --- | --- | --- |
| `/api/v1/services/{id}/replicas` | `GET` | 同 GET 详情 | 透传 `compute.GetServiceReplicas(ns, name)`；MVP 阶段 [§11](#11-后续迭代) 占位 |
| `/api/v1/services/{id}/events` | `GET` | 同 GET 详情 | 同上 `/events` 端点扩展 |
| `/api/v1/services/{id}/logs` | `GET` | 同 GET 详情 | 同上 `/logs` 端点扩展；阶段二上线后语义 / 透传规则与 [job.md §7.3](job.md#73-副本--事件--日志) 一致（含 SSE `follow=true`） |

MVP 期间这三个端点返回 `501 Not Implemented` + problem `type=https://axisml.io/errors/upstream-not-ready`，前端 Tab 3 / Tab 4 渲染占位提示。

---

## Part IV — 后端实现

## 8. 模块结构

目录：`components/platform/backend/internal/service/`

| 文件 | 职责 |
| --- | --- |
| `handler.go` | Gin 路由注册（`/api/v1/services` 前缀）、RBAC gate 装配 |
| `service.go` | 业务编排：tenant 解析 → quota 名拼接 → artifacts.Resolve 校验 → 调 Compute；列表跨租户并行合并；scale / start / stop / metrics 编排 |
| `context.go` | request-scoped 解析器：tenant → `compute_namespace` / `quotas`；与 [job.md §8.2](job.md#82-上下文解析) 共享底层 helper |
| `dto.go` | 请求 / 响应类型；与 Compute API DTO 的显式映射 |
| `view.go` | `Service` DTO 合并器：`compute.GetServiceByID` 响应 + Platform 注入展示字段（tenant / pool / unit 展示名、accessUrl） |
| `validate.go` | 提交前校验（[§5](#5-创建在线服务表单)） |
| `route.go` | 路由路径拼接（`/services/<tenant>/<name>/`）+ env 注入（`AXISML_SERVICE_BASE_URL`）+ 路由配置校验 |
| `jwt.go` | access JWT 颁发（`aud=axisml-inference`）；签名密钥与 `internal/auth` 共享 |
| `metrics.go` | Prometheus PromQL 模板组装 + 查询透传 |
| `logstream.go` | 阶段二：日志透传 pipe（与 [job.md §8 logstream](job.md#8-模块结构) 同构） |

无 `repository.go`：无 Platform 自有表（[§3](#3-数据模型platform-自有部分)）。

### 8.1 RBAC 中间件接入

`internal/auth` 提供 `RequireSystemAdmin` / `RequireTenantRole(role, tenantParam)` / `RequireServiceOwner` 中间件，定义见 [auth.md](auth.md)。本文档使用：

| 路由 | 中间件链 |
| --- | --- |
| `GET /api/v1/services` | 已登录即可；handler 内部按角色裁剪可见集合 |
| `POST /api/v1/services` | `RequireTenantRole("user", "<body.tenantId>")`，`system-admin` / `tenant-admin@self` 短路 |
| `GET /api/v1/services/{id}` | `RequireServiceOwner("id")`；其语义为「`@owner` 或在 service 所属租户上具备 `tenant-admin` 以上角色」；`system-admin` 短路 |
| `PATCH /api/v1/services/{id}`、`DELETE`、`POST .../scale`、`POST .../start`、`POST .../stop`、`GET .../access`、`GET .../metrics` | 同上 |
| `GET .../replicas` / `GET .../events` / `GET .../logs` | 同上 |

`RequireServiceOwner` 需要先 `compute.GetServiceByID(id)` 拿 `owner_user` + `namespace`；该 GET 结果通过 `gin.Context.Set("serviceView", view)` 注入后续 handler，避免重复调用。

### 8.2 上下文解析

**Tenant 解析**——与 [job.md §8.2](job.md#82-上下文解析) 共享：每个写请求与详情类读请求的第一步：

```go
ctx := resolveTenantContext(c, tenantName) // 内部：
   //   1. clustermanager.GetTenant(tenantName) → spec.namespace.name + spec.quotas[]
   //   2. 校验 spec.status.phase == Active
   //   返回 { tenantId, tenantDisplayName, computeNamespace, quotas: map[poolName][]quotaName }
```

**ElasticQuota 名拼接**——仅在创建服务时需要，复用上面的 `ctx`：

```go
pool := compute.GetResourcePool(req.ResourcePoolId)        // 拿 pool.name
// 校验 req.Quota 在 ctx.quotas[pool.name] 中存在
elasticQuotaName := "axisml-" + ctx.TenantName + "-" + pool.Name + "-" + req.Quota
// 写入 spec.scheduling.quota
```

拼接逻辑与 [job.md §8.2](job.md#82-上下文解析) / [workspace.md §7.1](workspace.md#71-工作区-cruddelete) 同源；建议抽到 `internal/tenant/quotaname.go` 共享。

**路由路径拼接**——`route.go` 内：

```go
// 用户未填 path 时
if route.Enabled && route.Path == "" {
    route.Path = fmt.Sprintf("/services/%s/%s/", ctx.TenantName, req.Name)
}
// 校验：必须 `/` 开头 `/` 结尾，且包含 service.name 防止路径冲突
// env 注入
if route.Enabled {
    env = append(env, corev1.EnvVar{Name: "AXISML_SERVICE_BASE_URL", Value: route.Path})
}
```

### 8.3 失败语义

Platform 端不持久化任何 service 字段，自然无双写一致性问题：

- 创建 / 扩缩容 / 删除：单点透传下游错误，不引入 Platform 补偿队列；
- 列表：并行调用中部分租户失败时，该租户在结果集中标 `partial=true` + `error.detail`，其余照常返回；前端在列表头部黄条提示「部分租户数据不可用」。

防误删：所有 service 写操作（PATCH / scale / start / stop / DELETE / access / metrics）先 `compute.GetServiceByID(id)` 校验 `kind == 'service'`，否则返 `404`——避免 service endpoint 被用来操作工作区。

### 8.4 JWT 与 JWKS

- 复用 [auth.md](auth.md) 中 Platform 内置 JWT 签名密钥（不为 access JWT 单独配密钥）；
- access JWT TTL 默认 1h，可由 Platform 配置项 `--service-access-jwt-ttl` 调整，上限 24h；
- `aud=axisml-inference` 与平台主用户 JWT (`aud=axisml-platform`) 与工作区 (`aud=axisml-workspace`) 区分：网关 SecurityPolicy 校验 `aud` claim 防止 JWT 跨用途滥用；
- JWKS 端点（`/.well-known/jwks.json`）与 [workspace.md §8.4](workspace.md#84-jwt-与-jwks) 共享，无需新增。

### 8.5 Prometheus 查询模板

`metrics.go` 维护按 backend 选择的 PromQL 模板，详情页 Tab 5 仅展示成熟模板覆盖的指标；不在表内的指标走「即将开放」占位。

| backend | metric | PromQL 模板（伪代码） |
| --- | --- | --- |
| `(native, deployment)` | `request_rate` | `sum(rate(envoy_http_downstream_rq_total{...service=<name>}[1m]))` |
| `(native, deployment)` | `latency` | `histogram_quantile(<percentile>, sum(rate(envoy_http_downstream_rq_time_bucket{...service=<name>}[5m])) by (le))` |
| `(native, deployment)` | `error_rate` | `sum(rate(envoy_http_downstream_rq_xx{response_code=~"5..", ...service=<name>}[5m])) / sum(rate(envoy_http_downstream_rq_total{...service=<name>}[5m]))` |
| `*` | `cpu_util` | `sum(rate(container_cpu_usage_seconds_total{namespace=<ns>, pod=~"<name>-.*"}[1m])) by (pod)` |
| `*` | `mem_util` | `sum(container_memory_working_set_bytes{namespace=<ns>, pod=~"<name>-.*"}) by (pod)` |
| `*` | `gpu_util` | `avg(DCGM_FI_DEV_GPU_UTIL{namespace=<ns>, pod=~"<name>-.*"}) by (pod)` |
| `(kserve, *)` | `request_rate` / `latency` / `error_rate` | 优先选用 KServe 自带的 `revision_request_count` / `revision_request_latencies_bucket`；模板在阶段二落地 |
| `(kserve, llminference)` | `tokens_per_second` / `ttft` / `tbt` | 占位，阶段三 |

模板中的 label selector 由 `metrics.go` 按 service `(namespace, name, backend, engine)` 实时拼装；Prometheus URL 来自 [overview.md §7.6 启动配置](overview.md#76-启动配置) 的 `--prometheus-url`。

### 8.6 度量与日志

特有 Prometheus 指标（通用上游调用指标见 [overview.md §7.5](overview.md#75-下游-typed-client)）：

- `platform_service_action_total{action, status}`：`action ∈ {create, get, list, scale, start, stop, patch, delete, access_token_issue, metrics_query, clone_for_new_version}`；
- `platform_service_list_tenant_fanout`：histogram，单次列表请求的下游扇出 namespace 数；
- `platform_service_list_partial_total{reason}`：counter，[§8.3](#83-失败语义) 中部分租户失败次数；
- `platform_service_access_jwt_issued_total{result}`：counter，access JWT 颁发量 + 失败原因；
- `platform_service_state{tenant_name, state}`：gauge，按租户聚合各 `services.status` 的 service 数；定期采样（拉 `compute.ListServices(namespace, kind=service)`）写入；
- `platform_service_metrics_query_total{metric, status}`：counter，Prometheus 查询调用结果分布。

zap 字段约定：每条 service 操作日志必带 `tenant_name` / `service_id` / `actor_user` / `action` / `status`；创建额外带 `backend_name` / `backend_engine` / `model_name` / `model_version` / `pool_id` / `resource_unit_id`；scale 额外带 `from_replicas` / `to_replicas`。

**审计日志**：Tab 6 审计是阶段二能力，但底层写入责任在 Platform service handler——`create` / `scale` / `patch` / `delete` 成功后，由 handler 向 [overview.md §7.4](overview.md#74-pg-schema) 的 `audit_logs` 表写一行：`action=service.<动作>`、`target=service:{service_id}`、`metadata` jsonb 含 `backend_name` / `backend_engine` / `model_ref` / `replicas` 等关键字段。Tab 6 渲染时按 `target` 前缀检索。MVP 阶段不展示——但写入路径在阶段二上 Tab 6 时只需打开开关即可，不再回头改 handler 逻辑。

---

## Part V — 实施与验证

## 9. 实现路径

### 9.1 阶段一（MVP）

| 模块 | 范围 | 完成信号 |
| --- | --- | --- |
| handler / service / dto / view | [§7.1](#71-在线服务-cruddelete) / [§7.2](#72-扩缩容) / [§7.3](#73-访问入口) / [§7.4](#74-指标查询) endpoint；仅 `(native, deployment)` backend；route 配置完整（`auth.type` 三选一） | `make platform-build` 通过 |
| RBAC 装配 | [§8.1](#81-rbac-中间件接入) 路由表全部接通；`system-admin` 短路与 `@owner` 校验单测覆盖 | 单元测试覆盖中间件分支 |
| validate.go | [§5](#5-创建在线服务表单) 校验全分支 + 防误删 `kind == 'service'` 校验 | 单元测试覆盖 |
| route.go | 路由路径拼接 + env 注入 + 路由配置校验 | 单元测试覆盖 |
| metrics.go | Prometheus 模板组装 + 透传；MVP 模板限 `(native, deployment)` 全部指标 | Integration 覆盖 |
| jwt.go | `aud=axisml-inference` access JWT 颁发与验证 | 单元 + 网关 SecurityPolicy 集成 |
| 创建表单（前端） | 单 role 块；backend 下拉只列 `(native, deployment)`；模型 / 镜像选择器 | 用户可在 UI 提交并跑通 |
| Integration | testcontainers PG + in-process gin + httptest fake Compute + httptest fake cluster-manager + httptest fake Artifacts + httptest fake Prometheus；happy path：创建（含 route）→ Ready → scale 0 → scale 2 → 删除 | `make platform-integration` 通过 |

阶段一显式不覆盖：`(native, statefulset)` / `(kserve, *)` / `(custom, *)`、Tab 3/4 副本 / 事件 / 日志、Tab 6 审计、多 role、`spec.route` 热更新、流量切换 UX。

### 9.2 阶段二

1. `(native, statefulset)`：handler / validate 加 statefulset 校验；详情页副本身份稳定展示。
2. `(kserve, inference)`：开放 `backend.config.{runtime, vllm/huggingface/triton/...}` 表单；KServe 指标模板补全。
3. Compute service 端 `/events` / `/logs` / `/replicas` 端点扩展（与 [workspace.md §11](workspace.md#11-后续迭代) 同源）→ 解锁详情页 Tab 3 / Tab 4。
4. Tab 6 审计日志：按 `target=service:{service_id}` 索引展示。
5. `spec.route` 热更新（依赖 compute-operator 的 [§5.7](../core/compute-operator.md#57-后续工作)）：解锁 PATCH 路由配置。
6. `spec.route.stripPathPrefix`：解锁不支持运行时 base path 的推理框架。

### 9.3 阶段三 / TBD

详见 [§11](#11-后续迭代)。

## 10. 关键设计决策

| 决策项 | 决策 | 理由 |
| --- | --- | --- |
| Platform PG 表 | **不为在线服务建任何表** | 与 [overview.md §9](overview.md#9-关键设计决策) 「Platform PG 仅存身份与视图映射」原则一致；建表只会带来双写漂移 |
| 寻址方式 | Compute `services.id`（uuid）；URL `/api/v1/services/{id}` | id 由 Compute 生成；与 [workspace.md](workspace.md) 共享 id-based + kind 过滤端点，复用代码 |
| 区分工作区 vs 在线服务 | 借用 Compute `services.kind` 列（`'service' \| 'workspace'`）| 1 个 enum 列代替 1 张表 + 三对象一致性补偿；Compute 行为对两种 kind 完全一致，仅供 Platform 列表过滤与防误删 |
| MVP backend 范围 | 只支持 `(native, deployment)` | 覆盖最常见的在线推理场景；其余 backend（statefulset / kserve / custom）单独拆分阶段 |
| 模型引用必填 | 创建表单要求填 `modelRef.{name, version}`；后端 schema 沿用 compute MLService 的 optional 字段，仍允许为空兼容未来「无 modelRef 的纯应用服务」 | 对齐 PRD §6.2.3「将已注册的模型版本部署为长驻推理服务」语义；不在 Compute 层引入额外约束 |
| spec 不可变性 | UI 不提供「修改模型 / 资源 / 路由」入口；改参数 = clone-with-new-version 新建 service + 用户自主切流量 | 与下游约束对齐；避免在 Platform 引入对 Compute 不支持的 PATCH 路径；版本切换语义清晰 |
| 扩缩容语义 | 显式 `POST /scale` + `start` / `stop` 语法糖端点 | scale 是核心；start/stop 是产品语义糖；底层翻译到 Compute `/scale` 对用户透明 |
| 停服语义 | `replicas=0` 即停服，保留 spec / route / 配额绑定 | 与 K8s Deployment scale 0 等价；与 PRD「下线旧版本」贴近 |
| 路由默认配置 | 默认 `route.enabled=true` + `auth.type=jwt` + Platform 自动拼路径 | 推理服务的「对外暴露」是核心使用场景；默认走 Platform 鉴权降低用户出错面 |
| 路径前缀处理 | MVP 注入 `AXISML_SERVICE_BASE_URL` env 让镜像 entrypoint 自处理；不支持的框架走 args 自配；compute-operator 的 `stripPathPrefix` 作为 [§11](#11-后续迭代) | 不引入 core 层硬依赖；与 [workspace.md §3.3 C](workspace.md#33-对-core-层的硬依赖必须同-pr-推进) 同源思路 |
| 版本切换 UX | 「克隆为新版本」前端反填，新建新 service 并由用户自主切外部流量；MVP 不接管 weighted route | 避免在 Platform 引入 ahead-of-time 流量管理；与 compute-operator 一次性 spec 落地原则一致 |
| 列表跨租户策略 | 默认单租户视图；`system-admin` 跨租户走并行 list + 内存合并；部分失败标 `partial=true` 而非整体 5xx | 普通用户日常只看自己租户；部分容忍策略避免单点故障拖垮全列表；与 [job.md §10](job.md#10-关键设计决策) 同源 |
| 防误删 | 所有 service 端点先 `compute.GetServiceByID(id)` 校验 `kind == 'service'`，否则 404 | 防止 service endpoint 被用来操作工作区 |
| 指标来源 | 实时调 Prometheus，不引入 Platform 侧时序缓存 | Prometheus 已是基础设施层一等公民（详见 [infra.md](../infra/infra.md)）；缓存只会带来漂移 |
| 日志 / 副本 / 事件 | 阶段二随 Compute service 端端点扩展开放，MVP 占位 | 与 [workspace.md §10](workspace.md#10-关键设计决策) 同节奏推进 |
| ElasticQuota 名拼接 | Platform 写前用 `axisml-<tenant>-<pool>-<quota>` 实时拼接，并校验该 quota 在 Tenant CR 内存在 | 提前校验避免 Pod 调度时才发现 quota 不存在；与 [job.md §10](job.md#10-关键设计决策) 同源 |

## 11. 后续迭代

- **`(native, statefulset)` backend**：解锁有状态推理（KV cache、模型分片、副本身份固定）；阶段二能力。
- **`(kserve, inference)` backend**：开放 vLLM / Triton / TF Serving / TorchServe / HuggingFace runtime；详见 [compute-operator.md §5.6.3](../core/compute-operator.md#563-kserve-inference)；阶段二能力。
- **`(kserve, llminference)` backend**：PD 分离推理 + LLM 专项指标；阶段三 / TBD。
- **`(custom, *)` 接入**：UI 表达需要 JSON schema 编辑器；与 [compute-operator.md §5.7](../core/compute-operator.md#57-后续工作) 一同推进。
- **`spec.route` 热更新**：解锁 PATCH 路由配置（auth 切换 / rateLimit 调整 / timeout 调整）；依赖 compute-operator 的 [§5.7](../core/compute-operator.md#57-后续工作)。
- **`spec.route.stripPathPrefix`**：在 HTTPRoute 上做路径重写，解锁不支持运行时 base path 的推理框架。
- **流量切换与灰度**：同一路径前缀下 weighted 切分新旧 service、自动指标判定回滚；产品 UX 与 Envoy Gateway 能力对齐。
- **自动扩缩容（HPA / KEDA）**：基于 `request_rate` / `gpu_util` / 自定义指标的弹性扩缩；与 [compute.md §7.7](../core/compute.md#77-后续工作) 同源。
- **多 role 独立扩缩**：解锁 `(kserve, llminference)` 的 `prefill` / `decode` / `router` 各自扩缩。
- **多端口 / 多协议**：HTTP + gRPC 共存、健康检查独立端口。
- **API key 轮换 UI**：`auth.type=apiKey` 的 Secret 自助轮换。
- **审计日志 Tab**：复用 [overview.md §7.4](overview.md#74-pg-schema) 的 `audit_logs` 表；按 `target=service:{service_id}` 索引展示。
- **Compute service `/events` / `/logs` / `/replicas` 端点扩展**：呼应 Job 端 [§6.4.5](../core/compute.md#645-副本与事件端点) 与 [workspace.md §11](workspace.md#11-后续迭代)；解锁详情页 Tab 3 / Tab 4。
- **SSE / WebSocket 增量列表**：替代轮询。
- **列表过滤下推**：Compute 端 `?ownerUser=` / `?labelSelector=` / `?modelName=` 等参数，把内存过滤收敛到 PG 查询。
- **cluster-manager 批量 GetTenant RPC**：减少 list 跨租户多次 RPC（与 [job.md §11](job.md#11-后续迭代) / [workspace.md §11](workspace.md#11-后续迭代) 同源）。
- **创建表单预设**：系统管理员维护一组「模型 + 镜像 + 资源单元 + 路由配置」预设供普通用户一键填好；只是前端语法糖，不入数据模型。
- **LLM 专项指标**：tokens/sec、TTFT、TBT、KV cache 占用、batch utilization；阶段三随 `(kserve, llminference)` 落地。
- **告警与 SLO**：基于 Tab 5 指标的告警规则模板，与 kube-prometheus AlertManager 集成；与 [overview.md §11](overview.md#11-后续迭代与-tbd) 「审计与告警」同步推进。

## 12. 测试策略

- **单元**（`internal/service/*_test.go`）：
  - `validate.go` 全分支（含模型 / 镜像 / 端口 / 路径 / 鉴权类型校验）；
  - `route.go` 路径拼接 + env 注入 + 校验；
  - `view.go` DTO 合并器（modelRef / route / 时间字段映射）；
  - `service.go` 列表合并器：部分租户失败时 `partial=true` 标记正确；
  - `service.go` start/stop 反查 audit_logs 拿最近 replicas 的回退逻辑；
  - 防误删：DELETE 一个 `kind='workspace'` 的对象 → 404；scale 同理；
  - RBAC 中间件分支（`system-admin` 短路 / `@owner` 比对 / 跨租户拒绝）；
  - `context.go` 解析器：`Tenant.status.phase != Active` 时返 400；
  - `metrics.go` PromQL 模板组装：按 backend 选择正确模板，label selector 正确；
  - `jwt.go` access JWT claim 字段完整性（`aud=axisml-inference`）。
- **integration**（`components/platform/backend/test/integration/`）：
  - testcontainers PostgreSQL（仅 `users` / `user_tenant_roles` / `audit_logs`）+ in-process gin（`httptest`）+ httptest fake Compute（按 [compute.md §7.5](../core/compute.md#75-id-based-寻址端点--kind-过滤) 模拟 `kind` 列与 kind 过滤端点）+ httptest fake cluster-manager + httptest fake Artifacts + httptest fake Prometheus；
  - happy path：创建（`(native, deployment)`，`kind='service'` 由 Platform 显式传，含 route）→ MLService Ready → scale 3 → scale 0 → start（反查 audit 拿到 3）→ 删除；
  - cancel / 防误删 path：DELETE 一个 `kind='workspace'` 的对象 → 404；
  - clone-with-new-version：基于现有 service 反填创建表单，提交后新建独立 service，两只 service 同时存在；
  - 列表多租户合并：构造 3 个租户、每租户 5 个 service + 5 个 workspace，断言 `kind=service` 过滤后只返回 15 个 service，并行 RPC 数 == 3；
  - 列表部分失败：1 个租户的 Compute 返 5xx，断言响应带 `partial=true` 且其余租户数据完整；
  - RBAC 矩阵：4 种角色在每个 endpoint 上的允许 / 拒绝；
  - 路由配置：`auth.type ∈ {none, jwt, apiKey}` 创建均能下发到 Compute；`apiKey` 引用不存在的 Secret 时 400；
  - 指标查询：fake Prometheus 返回固定时间序列，Platform 按 metric / range / percentile 正确组装 PromQL；
  - access JWT：仅 `auth.type=jwt` 时可拿，其他返 409；JWT claim 含 `aud=axisml-inference`。
- 不引入额外 minikube e2e：端到端 HTTPRoute / SecurityPolicy / Pod 行为由 [compute-operator.md §7](../core/compute-operator.md#7-测试) 与 [infra.md](../infra/infra.md) 的 Envoy Gateway 测试链覆盖。

## 13. 相关引用

- [PRD §6.2.3 在线服务](../../product/prd.md#623-在线服务)
- [docs/system_design/platform/overview.md](overview.md)
- [docs/system_design/platform/tenant.md](tenant.md)
- [docs/system_design/platform/workspace.md](workspace.md)
- [docs/system_design/platform/job.md](job.md)
- [docs/system_design/platform/model.md](model.md)
- [docs/system_design/platform/resource-pool.md](resource-pool.md)
- [docs/system_design/core/compute.md §7 Service](../core/compute.md#7-service)
- [docs/system_design/core/compute-operator.md §5 MLService](../core/compute-operator.md#5-mlservice-controller)
- [docs/system_design/core/artifacts.md](../core/artifacts.md)
- [docs/system_design/core/cluster-manager.md](../core/cluster-manager.md)
- [docs/system_design/infra/infra.md](../infra/infra.md)
- `auth.md`
