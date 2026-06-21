# AxisML Platform Backend 设计

本文档描述 Platform 层的**后端**（Go BFF）。前端架构见 [frontend.md](frontend.md)，认证 / RBAC / 数据面接入见 [auth.md](auth.md)，层概览见 [overview.md](overview.md)。

## 1. 定位与边界

AxisML 唯一直接面向用户的层：承担身份接入、业务编排与视图层映射，把用户操作翻译为对 cluster-manager / compute / artifacts 三个 System 层服务的下游调用。自有租户持久记录（`tenants` 表）与 Job / Experiment / Model / Image 的 name 级**定义**；运行实例（Run）与制品版本由下游持有。

| 做 | 不做 |
| --- | --- |
| 唯一外部 HTTP 入口（前端 SPA + 后端 REST） | 直接管理 K8s 资源（自身不持 K8s client、不碰 CR） |
| 身份认证 / JWT 颁发 / RBAC 校验 / access JWT 颁发 | 持有 Run / Service / Workspace 实例权威（→ compute） |
| 持有租户持久记录（`tenants`，`identifier` 标识）+ K8s Namespace 映射 / 停用 / 删除 | 配额折算 / Tenant CR 与物理资源落地（→ cluster-manager / tenant-operator） |
| 持有 Job / 实验 / Model / Image 定义（name 级 PG 表） | 持有制品**版本**权威（→ artifacts） |
| 跨服务业务编排（创建 / 租户内列表透传） | 持有 ResourcePool / ResourceUnit 词汇（→ cluster-manager） |
| 视图层映射（用户 ↔ 租户 ↔ `identifier`；workspace ↔ `MLService(kind=workspace)`） | 缓存下游可变实例状态（phase / status / digest / quota 用量 → 一律实时回源） |
| 前端 UI 多语言（§5.6） | 按 `Accept-Language` 本地化响应（后端与下游 locale-neutral，只返稳定机读 code） |

**统一 tenant scope**：compute / artifacts 的 URL `{namespace}` 兼容段表示租户 `identifier`，Platform 直接透传；它不是 K8s Namespace。物理落地点由 `tenants.kubernetes_namespace` / `Tenant.spec.namespace.name` 表达，可被多个 Tenant 共享（见 [high_level_design §2.2](../high_level_design.md#22-关键不变量)）。

## 2. 架构

```
External Users → Envoy Gateway → Platform ─┬─▶ cluster-manager
                                            ├─▶ compute
                                            └─▶ artifacts
```

外部流量必经 Envoy Gateway 进入 Platform；下游全 ClusterIP，API 端口通过 NetworkPolicy 仅允许 Platform namespace 访问。Platform **不直接访问任何 Infra 层组件**——不调 K8s API、不持 zot / RustFS client。Dashboard 与跨域指标聚合待后续专项设计。

```
┌──────── AxisML Platform ────────┐
│ Frontend (TS + React + Vite + Ant Design)        │
│      │ REST                                         │
│ Backend (Go + Gin + Cobra)                          │
│  ├─ auth (JWT + RBAC + access JWT + IdP 接口)       │
│  ├─ orchestrator (跨服务编排 / 租户内列表透传)      │
│  ├─ business modules (tenant/job/service/workspace/ │
│  │                    artifact/resourcepool)           │
│  └─ typed clients (clustermanager/compute/artifacts)│
│      │                                              │
│ Platform PG: tenants · users/sessions/user_roles    │
│   · 定义 jobs/experiments/models/images             │
│ Redis（可选）: 会话有效性 + 身份/RBAC 缓存          │
└─────────────────────────────────────────────────────┘
```

## 3. 核心模型

Platform 自有实体三类：**租户持久记录**、**身份 / 授权 / 会话**、**四张定义**。字段与索引见 [database.md §4](../database.md#4-platform)。

**租户（tenants）**：持有租户持久记录与生命周期权威——`identifier`（DNS-1123，唯一 tenant scope）、`kubernetes_namespace`（物理落地点，可共享）、展示元数据、`owner`、`suspended_at`（停用态）。经 cluster-manager REST 物化 / 回收 Tenant CR，不直接操作 CR；删除为硬删除，不提供 restore。

### 3.1 身份 / 授权 / 会话

| 实体 | 含义 | 标识键 | 备注 |
| --- | --- | --- | --- |
| User | 内置用户 | `id` / `username` unique | bcrypt 密码；OIDC 预留 |
| Role | RBAC 角色 | `name`（`system-admin` / `tenant-admin` / `user`） | 硬编码三档 |
| UserTenantRole | 用户 ↔ 租户成员关系 | `(user_id, tenant_name, role)` | `tenant_name` 引用 `tenants.identifier`（同库真实 FK） |
| Session | JWT 会话 / 刷新 token | `jti` | TTL 与 JWKS 由 auth 模块管理 |

角色 × 权限矩阵见 [auth.md §3](auth.md#3-rbac-角色)。

### 3.2 定义（jobs / experiments / models / images）

四张 name 级**定义 / 模板**实体；运行 / 版本**实例**由下游持有，经实时查询关联（Platform **不建** run/version 索引表）。`experiments` 是 `jobs` 的训练特化形态——`spec` 与 `jobs` 同构，Run 仍是 `MLRun`，仅多带 `axisml.io/experiment` label，**不引入新 compute CR / handler**。

| 定义 | 实例所有者 | `spec` 内容 | 关联（实时） |
| --- | --- | --- | --- |
| `jobs` | Run = `MLRun` | `backend` / `roles[]` / `scheduling{poolName,unitName,quota}` / `runPolicy` / 制品引用 | `compute.ListMLRuns(labelSelector=axisml.io/job=<job>)` |
| `experiments` | Run = `MLRun`（打 `axisml.io/experiment`） | 同 `jobs`；训练超参即 `roles[*].template.{args,env}`，指标经 TensorBoard（§4.9–§4.10） | `compute.ListMLRuns(labelSelector=axisml.io/experiment=<exp>)` |
| `models` | `model` 版本（artifacts） | name 级业务元数据（`framework` 等） | `artifacts.ListArtifactsByKind(ns, model, name)` |
| `images` | `image` 版本（artifacts） | name 级业务元数据（`purpose` 等） | `artifacts.ListArtifactsByKind(ns, image, name)` |

定义以 `(tenant_name, name)` 寻址，软删后同名可重建，可在零 Run / 零版本状态下存在。除四张定义、租户持久记录与身份外，Platform **不为任何下游对象建视图表**，也不持久化 run / version / phase / digest / quota 用量。

## 4. 核心功能

每节定义编排动作；字段契约见 [openapi/platform.yaml](../../openapi/platform.yaml)，权限矩阵见 [auth.md §3](auth.md#3-rbac-角色)。下游各自的强一致策略（compute Outbox + reconciler、PVC 同事务、artifacts 两阶段写）对 Platform 透明。

### 4.0 编排通则

各功能复用以下共用骨架，后续小节只列差异：

- **写定义** — RBAC 校验 → 字段校验 → 写 Platform PG（`(tenant_name, name)` 唯一）；不触下游。
- **触发实例**（Run / Service / Workspace / TensorBoard）— RBAC → 校验 `tenants.suspended_at` 为空（否则 `409 tenant-suspended`）→ 对引用制品版本逐个 `GetArtifact` 预检 `Ready`（失败 `400` 阻断）→ 快照 `定义.spec ⊕ overrides` → 透传名字三元组 `(poolName, unitName, quota)`，由 compute 内部展开 pool/unit 并组装 ElasticQuota 名。Platform **不拼 ElasticQuota 名、不展开 pool/unit、不解析 namespace、不建索引表**。
- **删除定义** — 实时列实例判活跃 → 有活跃则 `409 *-has-active-runs`，否则级联软删全部实例后软删定义（best-effort，部分失败上报）。
- **列表** — 租户分区端点**始终**要求活跃租户（`axisml.tenant` Cookie 优先，`X-Axisml-Tenant` 头兜底），scoped 到该单一租户（§5.3）；无对应绑定且非 admin → `404`；`system-admin` 可 scope 到任意租户。
- **身份** — 出站注入 `X-Axisml-User`；active tenant 解析见 §5.2。

### 4.1 租户编排

下游：cluster-manager（Tenant CR 物化 + 配额折算 + 运行态回源）。

| 用户操作 | 内部步骤 | 下游调用 |
| --- | --- | --- |
| 创建 | RBAC `system-admin` → 写 `tenants` 行 | `clustermanager.CreateTenant` |
| 编辑展示元数据 | 拦截不可变字段 → 更新 `tenants` 行 | `clustermanager.UpdateTenant`（同步 annotation，可选） |
| 删除 | 成员或活跃业务资源非空 → `409 tenant-in-use`；删除 Tenant 行 | `clustermanager.DeleteTenant` |
| 停用 / 恢复 | 置 / 清 `tenants.suspended_at` | —（仅 Platform PG；不下发 CR） |
| 配额 CRUD | 仅 `system-admin`；拦截 `pool` 不可变 | `clustermanager.Set/Update/DeleteQuota`（下发 `{unitName, quantity}[]`，cluster-manager 折算为 ElasticQuota） |
| 成员管理 | 自我保护（不能移除最后一个 `tenant-admin` → `409 last-tenant-admin`） | —（仅 Platform PG，事务内） |
| 列表 | 按角色裁剪绑定租户 + 运行态实时回源 | Platform PG + `clustermanager.GetTenant` |

**停用语义**：`suspended_at` 非空是**提交闸门**——Run 触发 / Service / Workspace 新建入口查此字段，非空即 `409 tenant-suspended`（Job 定义仍可编辑），已派生工作负载继续运行、可继续 scale / stop / delete。纯 Platform PG 态，不下发 CR；同时置灰前端新建 CTA。`identifier` / 配额 `pool` 创建后不可变。

### 4.2 计算任务编排

下游：compute（+ artifacts 预检）。Job 是可复用模板（`jobs` 表）；每次运行产生一个 Run（compute 的 `MLRun`），不落 Platform 表，实时回源。

| 用户操作 | 内部步骤 / 下游调用 |
| --- | --- |
| 创建 / 编辑 Job | 通则"写定义"；编辑只影响**之后**触发的 Run（已有 Run 已快照） |
| 删除 Job | 通则"删除定义"；有活跃 Run → `409 job-has-active-runs` |
| 触发运行 | 通则"触发实例" → 推导序号 `n` → 命名 `<job>-<n>` + 打 `axisml.io/job` label → `compute.CreateMLRun`；撞名（`409`）重列重算 `n` 重试（有界） |
| Run 列表 / 取消 / 删除 / 副本 / 事件 / 日志 | `RequireJobOwner` → 透传 `compute.{ListMLRuns(labelSelector),CancelMLRun,DeleteMLRun,GetMLRun{Pods,Events,Logs}}`（日志 / 事件 SSE follow） |

**触发期 override 白名单**：镜像 / 模型**版本**、`roles[*].template.resources`、`scheduling{poolName,unitName,quota}`、超参（`args` / `env`）。**禁止** override `backend.{name,engine}` 与 role 拓扑（增删 role / 改 replicas 结构）——只能改模板后重新触发。

寻址：Job `/api/v1/jobs/{name}`（活跃租户由 `axisml.tenant` Cookie/`X-Axisml-Tenant` 头携带）；Run 为子资源 `/jobs/{name}/runs/{run}`，`{run}` = `<job>-<n>`。Run spec 触发时由 `Job.spec ⊕ overrides` 快照冻结，创建后不可变。

### 4.3 在线服务编排

下游：compute（services + 运行指标代理）+ artifacts（预检）。

| 用户操作 | 内部步骤 / 下游调用 |
| --- | --- |
| 创建 | 通则"触发实例"（预检 `(modelName, modelVersion)`）→ `route.path==""` 时自动拼 `/services/<tenant>/<name>/` 并注入 `AXISML_SERVICE_BASE_URL` env → `compute.CreateMLService(kind=service)` |
| 扩缩容 | `RequireServiceOwner` → `ScaleMLService`（`Deleted` → `409 service-deleted`） |
| 停止 / 启动 | `stop` = scale 0 并把停前副本写 `annotations[platform.axisml.io/last-replicas]`；`start` = scale 回该 annotation（缺失 fallback 1）（§5.5） |
| 删除 | 先 `GetMLService` 校验 `kind==service` 防误删工作区 → `DeleteMLService` |
| 指标查询 | `RequireServiceOwner` → `compute.GetServiceMetrics`（按 backend 选 PromQL 的逻辑在 compute 侧，Platform 不感知 backend、不直连 Prometheus；失败 `502`） |

寻址 `/api/v1/mlservices/{name}`（与 jobs 对称）；spec 除 `roles[*].replicas` 外不可变。在线服务数据面鉴权设计为 API KEY（`route.auth.type=apiKey`，后续提供）；当前仅 `none`。

### 4.4 工作区编排

下游：compute（services with `kind=workspace`；PVC 由 compute 同事务派生与回收）。工作区 = 长驻交互式开发容器，复用 `MLService(native, deployment)`，不引入新 CRD。

| 用户操作 | 内部步骤 / 下游调用 |
| --- | --- |
| 创建 | 通则"触发实例" → 生成 `workspace_name="ws-"+crockford32(rand40bit)` → 注入 PVC `size`/`storageClass` → `compute.CreateMLService(kind=workspace, route.enabled=false)`；SecurityPolicy 交付后再启用外部 route |
| 停止 / 启动 | 同 §4.3（工作区恒为 1 副本） |
| 删除 | 校验 `kind==workspace` → `DeleteMLService(?deletePvc=`，默认 true`)` |
| 浏览器接入 | 当前不开放；SecurityPolicy 派生交付后再启用 `aud=axisml-workspace` access JWT，现阶段保持 fail-closed |

Platform 不为工作区建任何 PG 表；"这是工作区"由 compute `mlservices.kind='workspace'` 表达。寻址 `/api/v1/workspaces/{name}`。

### 4.5 制品编排

下游：artifacts。每个 Kind（`model` / `image`）是 name 级定义（`models` / `images` 表）；版本由 artifacts 持有，实时回源。

| 用户操作 | 内部步骤 / 下游调用 |
| --- | --- |
| 创建 / 编辑定义 | 通则"写定义"（name 级业务元数据）；定义可零版本存在 |
| 删除定义 | 通则"删除定义" → **直接级联软删**全部版本（不阻止），GC 异步清后端，四元组永不复用 |
| 版本列表 | `artifacts.ListArtifactsByKind(ns, kind, name)` |
| 新增版本（initiate） | 按 `source` 透传：`webUpload` / `oras`（模型）/ `dockerPush`（镜像）返推送凭证；`external` 登记远端免上传 → `artifacts.InitiateUpload` 返 `{artifact, upload}` |
| 完成版本（complete） | 校验 `digest` 非空 → `artifacts.CompleteUpload`（`DigestMismatch` / `Failed` 下游 4xx 反馈） |
| 版本详情 / 编辑 | tuple 直拼下游（仅 `displayName` / `description` / `labels` / `annotations` 可改）→ `artifacts.{Get,Update}Artifact` |
| 获取下载凭证 | `artifacts.Resolve(usage=download)`（1h TTL pull token / S3 STS） |
| 定义列表 | 通则"列表" → 再并入 `default` tenant scope 的 public 定义 |

**消费侧预检**见 §4.0；Platform 在创建 workload 时调用 `resolve?usage=inspect`，把不可变 URI / digest 快照进 compute 请求。operator 不调用 Artifact Hub，只注入快照与 tenant-operator 已落地的 ServiceAccount / Secret 引用。

寻址：定义 `/api/v1/{kind-plural}/{tenant}/{name}`（`{kind-plural} ∈ {models, images}`）；版本为子资源 `/versions/{version}`。`(name, version)` 创建后不可变，进入 `Ready` 冻结 spec / digest；改 spec = 新增版本。Kind 专属 spec（model `framework`、image `purpose`）由 artifacts handler 硬校验，Platform 仅透传。

### 4.6 资源池编排

下游：cluster-manager（ResourcePool CRUD；units 内嵌）+ compute（删除前置阻断的用量反查）。

| 用户操作 | 内部步骤 / 下游调用 |
| --- | --- |
| 资源池 / 单元 CRUD | RBAC `system-admin`；`PATCH` 拦截 `name` 不可变 → `clustermanager.{Create,Update,Delete}{ResourcePool,ResourceUnit}`（unit 内部 patch `spec.units[]`） |
| 删除前置 | 枚举可见 tenant scope，分别调用现有 MLRun / MLService list API，并用 `labelSelector=axisml.io/resource-{pool,unit}=<name>` 过滤活跃对象；命中则 `409 {pool,unit}-in-use`，否则允许删除 |
| 列表 | 已登录可读、`system-admin` 写 → `clustermanager.ListResourcePools`（单次返 pool + 内嵌 units） |

Platform 不为池 / 单元建表；Node label / taint 由管理员 `kubectl` 维护，UI 不下发；池 / 单元为全集群对象，不按租户过滤。

### 4.7 Dashboard 编排

Dashboard 的聚合模型与接口暂不在本版系统设计中定义，待后续专项设计。System 服务不为此预留 `/cluster/*` 或 `/workloads/*` 聚合 API。

### 4.8 流量配置编排

下游：compute（流量策略 + 加权路由派生 + 灰度指标代理）。流量策略把一个稳定入口的流量按权重分发到本租户多个在线服务后端，加权路由由 compute 内部派生，Platform 不直连网关、不内嵌 PromQL。字段契约见 [compute-service.md §4.3](../system/compute-service.md#43-流量策略mltrafficpolicy)。

| 用户操作 | 内部步骤 / 下游调用 |
| --- | --- |
| 创建 | 对每个成员 `GetMLService` 预检 `kind==service` 且 `Ready` → 校验同租户、未被其它活跃策略占用 → `endpoint.path==""` 自动拼 → `compute.CreateTrafficPolicy` |
| 调整流量 | `RequireTrafficPolicyOwner` → 校验 `Σweight==100`（加权）/ `0–100`（灰度）→ `compute.UpdateTrafficSplit` |
| 提升 / 回滚（灰度专属） | `promote` 置灰度后端 100 并互换 stable/canary `role`；`rollback` 置灰度 0 → `compute.{PromoteCanary,RollbackCanary}` |
| 指标查询 | `compute.GetTrafficMetrics`（按后端分组 + 灰度健康对比；失败 `502`） |
| 列表 / 详情 / 删除 | 通则"列表" / 透传；删除仅解除路由，成员 MLService 保留 |

寻址 `/api/v1/trafficpolicies/{name}`（动作子资源 `/split` `/promote` `/rollback`，只读 `/metrics` `/events`）；`endpoint` 与 `mode` 创建后不可变，可变仅权重与灰度百分比。鉴权同 §4.3（当前仅 `none`）。

### 4.9 实验编排

下游：compute（+ artifacts 预检）。实验是训练特化模板（`experiments` 表，`spec` 与 `jobs` 同构）；每次运行产生一个 Run（`MLRun`，打 `axisml.io/experiment` label），复用计算任务现有后端（`(native,job)` / `(kubeflow-trainer,*)`），**无需新 handler**。训练超参即 `args/env`，Platform **不单独建模 / 不展示超参**；训练指标不入 PG——Run 容器把 TensorBoard event log 写到 compute 注入的对象存储位置（路径与凭证由 compute 渲染 Pod 时注入，Platform 不碰对象存储），经 §4.10 TensorBoard 查看。

编排与 §4.2 同构（`RequireExperimentOwner`）：写 / 编辑 / 删除定义、触发运行（命名 `<exp>-<n>` + `axisml.io/experiment` label）、Run 列表 / 取消 / 删除 / 日志均按 §4.2；override 白名单同 §4.2。额外两项：

- **对比** — 选定 Run 集合 → 拉起 / 复用 TensorBoard（§4.10）；指标与超参对比全在 TensorBoard（HParams / Scalars），Platform 不自建对比视图。
- **登记模型** — `compute.GetMLRun` 取 checkpoint 产出位置 → 走 §4.5 制品登记流程。

寻址 `/api/v1/experiments/{name}`（Run 子资源 `/runs/{run}`，TensorBoard 子资源 `/tensorboard`）。训练指标 / checkpoint 不落 PG、不经 Platform：路径约定 `experiments/<exp>/runs/<run>/{tb,output}/` 与对象存储凭证由 compute 注入；Run 删除时由 compute 一并 GC 产出。

### 4.10 TensorBoard 编排

下游：compute（services with `kind=tensorboard`）。按需为某实验（或选定 Run 集合）拉起的**临时只读指标视图**，复用 `MLService(native, deployment)`、不引入新 CRD，logdir 指向 §4.9 约定的对象存储 event log 前缀（路径与读凭证由 compute 注入）。

| 用户操作 | 内部步骤 / 下游调用 |
| --- | --- |
| 启动 | `RequireExperimentOwner` → 解析目标 → 幂等（已存活则复用）→ `compute.CreateMLService(kind=tensorboard, route.enabled=false)`；SecurityPolicy 交付后再开放外部 route |
| 打开 / 停止 | 返回 route endpoint / 手动停或空闲 TTL 自动回收（`compute.{Get,Delete}MLService`） |

- 临时、只读、可空闲回收；不产出制品；Pod 仍走 koord-scheduler 与租户配额。
- 启动 / 打开 / 停止均限 `owner` 或 `tenant-admin`（会拉起占配额的 workload）。
- 数据面访问目标方案复用工作区 access JWT；当前 SecurityPolicy 未交付，外部访问不开放；`kind` 创建后不可变。

## 5. 关键机制

### 5.1 跨服务调用模型

每下游一个 typed client 子包，强类型方法。共享约定：

- **身份注入**：出站自动带 `X-Axisml-User: <username>`；下游信任此头，只做 ownership 归属，不做角色级鉴权。
- **超时**：默认 30s；幂等读可有限退避；写不自动重试。
- **错误透传**：4xx 透传 problem；5xx 包装为 `type=https://axisml.io/errors/upstream-failure` 并附下游服务名。
- **可观测性**：每次调用打日志 + `platform_upstream_request_total{service,method,status}`。

### 5.2 上下文解析

**Active tenant 来源**：名寻址路径（jobs / experiments / workspaces / mlservices / trafficpolicies）由活跃租户携带——`axisml.tenant` Cookie 优先，`X-Axisml-Tenant` 头兜底（供 CLI / e2e / 服务间调用）；tuple 寻址路径（`{kind}/{tenant}/{name}`）已在 URL 内带 tenant。RBAC 中间件：

| 端点形态 | header 缺省 | header 存在 |
| --- | --- | --- |
| list | `400 active-tenant-required`（租户分区端点恒需 header） | scoped 到该 tenant（无 binding 且非 admin → `404`；§5.3） |
| create / name 寻址 detail | `400 active-tenant-required` | 用 header 取 `tenant_name` 作分区键 |
| tuple / 租户 / 资源池路径 | URL 内已带标识或为全集群对象，header 忽略 | 同上 |

**下游 tenant scope**：URL `{namespace}` 兼容段直接用 `identifier`；物理 K8s Namespace 从 `kubernetes_namespace` 映射取得并单独传递。pool/unit 展开、ElasticQuota 名组装、PVC 生命周期均由 compute 内部完成。

### 5.3 列表租户作用域

租户分区端点的 list **始终**要求活跃租户（`axisml.tenant` Cookie 优先，`X-Axisml-Tenant` 头兜底），只查该单一租户，无跨租户 fanout / 合并。两者皆缺 → `400 active-tenant-required`；非 admin 且对该租户无绑定 → `404`；`system-admin` 可 scope 到任意租户（逐个，不聚合）。租户管理 / 资源池等全集群端点为集群对象，忽略租户作用域（§5.2）。

### 5.4 失败语义

Platform 持有定义与身份数据，不持有任何下游实例状态，且不直接调 K8s API：

- 定义 CRUD：Platform PG 单点写，失败本地 4xx。
- 实例操作（触发 / 上传 / 取消 / 删除 / 级联软删）：单点透传下游，4xx / 5xx 透传；无 outbox 无补偿队列；级联软删 best-effort。
- 列表：单租户透传下游 LIST，下游失败 4xx / 5xx 透传。

### 5.5 扩展元数据写入约定

Platform 在下游对象上挂自定义元数据（`last-replicas` 副本基线、批次 ID、外部系统关联键等），统一通过下游业务服务的 `labels` / `annotations` 字段写入，**不直接 patch CR**：

| 维度 | 约定 |
| --- | --- |
| 写入路径 | `clustermanager.{Create,Update}Tenant` / `compute.{Create,Update}{Job,Service}` / `artifacts.UpdateArtifact` 请求体携带 `labels` / `annotations` |
| 存储 | 下游 PG 表的 `labels`/`annotations` jsonb 列（[database.md §1.6](../database.md#16-扩展元数据-labels--annotations)） |
| Key 命名空间 | Platform 内部 `platform.axisml.io/<key>`；终端用户透传走 `user.axisml.io/<key>` 或无前缀 |
| 同步语义 | 不触发 CR patch（不 `+generation`）、不引发 reconcile；纯 PG mutation，写后即读 |

**典型用途**：`stop` 前把当前副本写入 `annotations[platform.axisml.io/last-replicas]`，`start` 时读回（缺失 fallback 1）。**反模式**：不向 K8s CR 写业务扩展位；不在 Platform PG 镜像下游实例状态。

### 5.6 多语言 / i18n

多语言是**前端**职责；Platform 后端与下游三服务一律 **locale-neutral**——不读 `Accept-Language`、不本地化任何响应文案，只产出稳定机读标识，由前端按当前 locale 映射。首批 `zh-CN` / `en-US`，catalog 可扩展，新增语言不触达后端。

**i18n 契约 = 稳定机读标识**：

| 机读标识 | 前端本地化方式 |
| --- | --- |
| RFC 7807 problem 的 `type` URI + 下游 error code | 作文案映射 key；`title` / `detail` 仅英文调试兜底、UI 不直接展示 |
| 机读枚举（`phase` / `status` / `source` / `kind` / `role` 等） | 展示层映射为本地化标签；枚举原值不翻译 |
| 时间戳（RFC3339 UTC）/ 数值 | 按 locale 格式化（`Intl` / dayjs） |

**前端实现**：文案经 react-i18next 维护；Ant Design 经 `ConfigProvider` 注入 locale。**语言选择**纯浏览器端持久化（localStorage，初值 `navigator.language`），不落 PG、不入会话、不随出站请求传播。**不本地化**：用户自由文本与日志正文原样回显。

**关键不变量**：后端零文案，新增语言 = 加 catalog + AntD locale 包，后端与下游零改动；error `type` / 下游 code 是稳定契约（改文案不改 code）。

RBAC 中间件装配见 [auth.md](auth.md)；Platform 路由层挂载 `RequireSystemAdmin` / `RequireTenantRole` / `RequireJobOwner` / `RequireExperimentOwner` / `RequireServiceOwner` / `RequireTrafficPolicyOwner` / `RequireWorkspaceOwner`（均按 `name` 寻址）。

## 6. 接口契约

| 类别 | 内容 |
| --- | --- |
| 对外 REST | 业务 tag：`Auth` / `Tenants` / `Quotas` / `Members` / `Jobs` / `Experiments` / `MLServices` / `TrafficPolicy` / `Workspaces` / `Models` / `Images` / `ResourcePools` / `ResourceUnits`；name 寻址 `/api/v1/{kind}/{name}`（Run / TensorBoard 为子资源），制品 tuple 寻址 `/{kind-plural}/{tenant}/{name}`；系统 tag（`Users` / `Health`）见 yaml。契约源 [openapi/platform.yaml](../../openapi/platform.yaml) |
| 状态 | 不暴露任何 K8s CR；下游运行态字段（phase / conditions / quota 用量）作只读透传 |
| 错误格式 | HTTP 标准码 + RFC 7807 problem+json；下游 problem 透传或包装；`type` URI / 下游 code 为稳定机读标识（§5.6） |
| 流式 | 日志 / 事件 `follow=true` 用 SSE；非 follow 用 `text/plain` chunked |
| 身份头 | 入站校验主登录 JWT + 活跃租户（`axisml.tenant` Cookie / `X-Axisml-Tenant` 头，§5.2）；出站注入 `X-Axisml-User`（[auth.md §6](auth.md#6-下游身份透传)） |
| 数据面接入 | 工作区 / TensorBoard JWT SecurityPolicy 与在线服务 API KEY 均未交付；受保护入口 fail-closed（[auth.md §5](auth.md#5-数据面接入)） |
| Prometheus | `platform_*` 自身指标 |

## 7. 依赖

| 依赖 | 用途 |
| --- | --- |
| PostgreSQL | 身份 / 授权 / 会话 + 四张定义；与 compute / artifacts 共享 DB，按表名前缀隔离（[database.md §4](../database.md#4-platform)） |
| Redis（可选） | 认证热点读缓存（会话有效性 + 身份 / RBAC），key 前缀 `platform:`；权威仍是 PostgreSQL，不可达即回退（[auth.md §2.1](auth.md#21-会话与身份缓存)） |
| Envoy Gateway | 唯一外部入口；TLS 终止 / HTTPRoute；数据面 SecurityPolicy 待交付（[infra.md](../infra/overview.md)） |
| cluster-manager | ResourcePool / Unit CRUD + 租户 CR 物化（含配额折算 + 运行态回源） |
| compute | Run / Service / Workspace / TrafficPolicy / TensorBoard 权威；创建体接 `scheduling{poolName,unitName,quota}` 名字对；资源池删除检查复用按 tenant scope 的 labelSelector 列表查询 |
| artifacts | 模型 / 镜像版本；两阶段写或 `external` 登记；Platform 负责 `GetArtifact` 预检与 `resolve?usage=inspect` 快照 |

## 8. 运行时形态

| 维度 | 值 |
| --- | --- |
| 进程 | 单二进制 `axisml-platform`；子命令 `serve` / `migrate` / `bootstrap` |
| 副本 | chart 默认 `replicas=1`；无状态，后续可水平扩 |
| 启动子命令 | `serve` 启 HTTP API + 后台任务（含过期会话清理 sweep，`SESSION_SWEEP_INTERVAL` 默认 1h）；`migrate` 执行 GORM 迁移；`bootstrap` 创建内置角色、初始 `system-admin`（默认 `admin/admin`，首登强制改密，可由 `AXISML_BOOTSTRAP_PASSWORD` 覆盖）及内置 `default` 租户（K8s Namespace `axisml-tenant`） |
| 缓存 | 可选 Redis 前置认证热点读；`REDIS_ADDR` 空则直连 PostgreSQL（[auth.md §2.1](auth.md#21-会话与身份缓存)） |
| 暴露端口 | 目标 API `:8080`、Metrics `:8081`、Probes `:8082`（`/healthz` / `/readyz`），JWKS `/.well-known/jwks.json` 走 ClusterIP（当前 workload 镜像仍为 nginx placeholder） |
| RBAC scope | 无 K8s API 需求（全部下沉下游服务） |
| Helm / 镜像 | 见 [deployment.md](../deployment.md) |

## 9. 相关引用

- [high_level_design.md](../high_level_design.md) — 控制平面拓扑与系统不变量
- [auth.md](auth.md) — 身份与鉴权契约、access JWT、中间件
- [database.md](../database.md) — Platform PG schema（§4）
- [deployment.md](../deployment.md) · [infra.md](../infra/overview.md)
- [openapi/platform.yaml](../../openapi/platform.yaml) — REST 契约源
- 下游：[cluster-manager.md](../system/cluster-manager.md) · [compute-service.md](../system/compute-service.md) · [artifact-hub.md](../system/artifact-hub.md) · [tenant-operator.md](../system/tenant-operator.md) · [compute-operator.md](../system/compute-operator.md)
