# AxisML Platform 概要设计

## 1. 定位与边界

AxisML 系统中唯一直接面向用户的层；承担身份接入、业务编排与视图层映射，把用户视角的操作翻译为对 cluster-manager / compute / artifacts 三个 System 层服务的下游调用。

| 做 | 不做 |
| --- | --- |
| 唯一外部 HTTP 入口（前端 SPA + 后端 REST） | 直接管理 K8s 资源（全部下沉下游服务，自身不持 K8s client） |
| 身份认证 / JWT 颁发 / RBAC 校验 / access JWT 颁发 | 持有 Tenant / Quota / Job / Service 权威（→ [compute-service.md](compute-service.md)） |
| 跨服务业务编排（创建 / 跨租户列表合并 / Dashboard 聚合） | 持有 ResourcePool / ResourceUnit 词汇（→ [cluster-manager.md](cluster-manager.md)） |
| 视图层映射（用户 ↔ 租户绑定 ↔ `tenant_name`；workspace ↔ `MLService(kind=workspace)`） | 持有 Artifact 权威（→ [artifact-hub.md](artifact-hub.md)） |
| 全局可见制品的列表合并（`axisml-system` 内置租户 + `visibility=public`） | 拼 ElasticQuota 名 / 解析 K8s namespace（均下沉 compute） |
| 二次缓存下游业务字段（无任何视图缓存表） | — |

**统一分区键**：compute / artifacts 的 REST URL `{namespace}` 段 = `tenant_name`，Platform 直接透传，**不解析 K8s namespace**——PVC 生命周期、ElasticQuota 名组装均由 compute 内部完成，`spec.namespace.name` 不进入编排路径。`compute.GetTenant` 仅服务于租户详情页 / 配额 Tab 的展示。

## 2. 架构

### 2.1 上下文

```
       ┌──────────────────┐
       │  External Users  │
       └─────────┬────────┘
                 ▼
       ┌──────────────────┐
       │  Envoy Gateway   │
       └─────────┬────────┘
                 ▼
       ┌──────────────────┐
       │     Platform     │
       └─┬───────┬───────┬┘
         │       │       │
         ▼       ▼       ▼
   cluster-   compute  artifacts
   manager
```

外部流量必须经 Envoy Gateway 进入 Platform；下游全部 ClusterIP。Platform 只与三个 System 层自研服务通信，**不直接访问任何 Infra 层组件**——不调 K8s API、不碰 zot / RustFS、**也不直连 Prometheus**。所有指标查询封装在拥有该域的 System 服务里：在线服务 / 工作负载 / 租户运行指标由 compute 代理（backend→PromQL 模板选择在 compute 侧），集群容量与集群时序由 cluster-manager 代理；Platform 仅透传与聚合它们回传的 `MetricSeries`，不感知 PromQL，也不感知 backend。

### 2.2 内部结构

```
┌──────────────────────── AxisML Platform ────────────────────────┐
│  Frontend (TS + React + Vite)                                   │
│        │                                                        │
│        ▼ REST                                                   │
│  Backend (Go + Gin + Cobra)                                     │
│   ├─ auth (JWT + RBAC + access JWT + IdentityProvider 接口)     │
│   ├─ orchestrator (跨服务编排 / 跨租户 fanout 合并)             │
│   ├─ business modules (tenant / job / service / workspace /     │
│   │                    artifact / resourcepool / dashboard)     │
│   └─ typed clients (clustermanager / compute / artifacts)       │
│        │                              │                         │
│        ▼                              ▼                         │
│  Platform PG                    下游 ClusterIP 调用             │
│  (users / sessions /                                            │
│   user_tenant_roles / audit_logs)                               │
└─────────────────────────────────────────────────────────────────┘
```

## 3. 核心模型

Platform 自有实体仅覆盖**身份 / 授权 / 会话 / 审计**四类，完整字段与索引见 [database.md §4](../database.md#4-platform)。

| 实体 | 含义 | 标识键 | 备注 |
| --- | --- | --- | --- |
| User | 内置用户 | `id` / `username` unique | bcrypt 密码；OIDC 预留 |
| Role | RBAC 角色 | `name`（`system-admin` / `tenant-admin` / `user`） | 硬编码三档；不开放自建 |
| Permission | 权限位 | `name` | 角色 × 权限矩阵详见 [auth.md §3](../auth.md#3-rbac-角色) |
| UserTenantRole | 用户 ↔ 租户成员关系 | `(user_id, tenant_name, role)` | `tenant_name` 引用 compute `tenants.name`（稳定 FK，跨服务不约束） |
| Session | JWT 会话 / 刷新 token | `jti` | TTL 与 JWKS 由 auth 模块管理 |
| AuditLog | 操作流水 | `id` | `action` / `target` / `metadata` / `actor` |

**关键不变量**：Platform 不为任何下游业务对象（Tenant / Workspace / Job / Service / Artifact / Pool）建视图表；状态 / phase / conditions / digest / quota 用量始终实时回源下游。视图层只持有「用户 → 租户绑定（`user_tenant_roles`）」这一关系，租户展示元数据靠 `compute.GetTenant` 现取。

## 4. 核心功能

每节定义编排动作；UI 字段与布局见 [wireframe.md](../wireframe.md)，字段契约见 [apis/platform.yaml](../apis/platform.yaml)，权限矩阵见 [auth.md §3](../auth.md#3-rbac-角色)。

### 4.1 租户编排

下游：compute。

| 用户操作 | Platform 内部步骤 | 下游调用 | 一致性策略 |
| --- | --- | --- | --- |
| 创建租户 | RBAC `system-admin` → 字段校验 | `CreateTenant` | 单点透传；4xx/5xx 直返 |
| 编辑展示元数据 | RBAC `system-admin` → 拦截不可变字段 | `UpdateTenant`（`display_name` / `description` / `namespace` 业务线 / `labels` / `annotations`） | `400 immutable-field` Platform 兜底 |
| 软删 / 恢复 | RBAC + 成员校验（非空 → `409 tenant-has-members`） | `DeleteTenant` / `RestoreTenant` | 成功后级联清理 `user_tenant_roles` |
| **暂停 / 恢复** | RBAC `system-admin` → 透传 | `SuspendTenant` / `ResumeTenant`（`Active ⇄ Suspended`） | 单点透传；compute 在创建入口对 `Suspended` 租户返 `409 tenant-suspended`，已运行 workload 不受影响 |
| 配额 CRUD | RBAC `tenant-admin@self` → 拦截 `(pool, name)` 不可变 | `Add/Update/DeleteQuota` | `400 immutable-field` Platform 兜底 |
| 成员管理 | RBAC + 自我保护（不能移除最后一个 `tenant-admin` → `409 last-tenant-admin`） | —（仅 Platform PG） | `user_tenant_roles` 内事务 |
| 列表 | 按角色裁剪：非 `system-admin` 取绑定 tenant 集合 | `ListTenants`（query 下推） | 单租户失败 → `partial=true` |

**暂停语义**：`Suspended` 是**提交闸门**——锁定该租户下 Job / Service / Workspace 的**新建入口**，已派生工作负载继续运行、可继续 scale / stop / delete。闸门由 compute 在创建端点强制（返 `409 tenant-suspended`），Platform 按 phase 置灰前端新建 CTA；`tenant-operator` 不参与暂停。

关键不变量：Platform 不为租户建任何视图表；`name` / `namespace.name` / 配额 `(pool, name)` 创建后不可变。成员数为 Platform 侧 `user_tenant_roles` 聚合，与 compute 无关。

### 4.2 计算任务编排

下游：compute（+ artifacts 预检）。

| 用户操作 | Platform 内部步骤 | 下游调用 | 一致性策略 |
| --- | --- | --- | --- |
| 提交任务 | RBAC `user@self`+ → 对引用的镜像 / 模型 / 数据集逐个 `GetArtifact` 预检 `Ready` → 请求体携带 `scheduling{poolName, unitName, quota}` 名字三元组 | `compute.CreateJob(ns=tenant, body)` | 单点透传；pool/unit 展开与 quota 名组装由 compute 内部完成 |
| 取消 | `RequireJobOwner` → 透传 | `CancelJob` | 状态合法性由 compute 4xx 反馈 |
| 删除 | RBAC + 透传 | `DeleteJob` | spec 不可变，删除即终态（重提交即新建） |
| 列表 | §5.2 解析 active tenant：header 在 → 单租户透传；`system-admin` 无 header → §5.3 跨租户合并 | `ListJobs(ns, labelSelector?, phase?)` | 跨租户路径部分失败 → `partial=true` |
| 副本 / 事件 / 日志 | `RequireJobOwner` → 透传（含 SSE `follow=true`） | `GetJob{Pods,Events,Logs}` | 流式 chunked / SSE 透传 |

关键不变量：任务标识 `(tenant_name, job_name)`，`tenant_name` 由 `X-Axisml-Tenant` 头携带（§5.2），URL 形态 `/api/v1/jobs/{name}`；Job spec 不可变——「编辑」=「再次提交」预填重建。Platform **不拼 ElasticQuota 名、不展开 pool/unit、不解析 namespace**，只透传名字三元组。

### 4.3 在线服务编排

下游：compute（services + 运行指标代理）+ artifacts（预检）。

| 用户操作 | Platform 内部步骤 | 下游调用 | 一致性策略 |
| --- | --- | --- | --- |
| 创建 | RBAC + `GetArtifact` 预检 → `route.path==""` 时自动拼 `/services/<tenant>/<name>/` 并注入 `AXISML_SERVICE_BASE_URL` env → 请求体携带 `scheduling{poolName, unitName, quota}` + `route` | `compute.CreateService(ns, body{kind=service,...})` | 单点透传 |
| 扩缩容 | `RequireServiceOwner` → 透传副本数 | `ScaleService` | 幂等；`Deleted` → `409 service-deleted` |
| 停止 / 启动 | `RequireServiceOwner` → `stop` = scale 0 并把停前副本数写入 `annotations[platform.axisml.io/last-replicas]`；`start` = scale 回该 annotation（缺失 fallback 1） | `UpdateService`（元数据）+ `ScaleService` | annotation 为 PG-only 元数据，随时可写（§5.5） |
| 删除 | 先 `GetService` 校验 `kind==service` 防误删工作区 → 透传 | `DeleteService` | 派生 K8s 资源由 ownerReference 级联 |
| 路由 / 访问 | `route.auth.type=jwt` 时颁发 `aud=axisml-inference` 短 TTL access JWT | —（Platform 自签） | 数据面网关验签放行；`route-auth-mismatch` → `409` |
| 指标查询 | `RequireServiceOwner` → 透传（按 backend 选 PromQL 模板的逻辑在 compute 侧，Platform 不感知 backend、不直连 Prometheus） | `compute.GetServiceMetrics(name, metric, range, step)` | 查询失败 → `502 upstream-failure` |
| 列表 | 同 §4.2；`kind=service` 过滤下推 | `ListServices(ns, kind=service)` | 跨租户部分失败 → `partial=true` |

关键不变量：寻址 `(tenant_name, service_name)`，URL `/api/v1/services/{name}`（与 Jobs 对称）；spec 除 `roles[*].replicas` 外不可变；在线服务运行指标由 compute 内部按 backend 选 PromQL 模板查询（模板见 [monitoring.md §6](../monitoring.md#6-业务指标查询prometheus-代理)），Platform 与 UI 均不内嵌 PromQL、不直连 Prometheus。

### 4.4 工作区编排

下游：compute（services with `kind=workspace`；PVC 由 compute 同事务派生与回收，详见 [compute-service.md §4.4](compute-service.md#44-service)）。工作区 = 长驻交互式开发容器，**复用** `MLService(native, deployment)`，不引入新 CRD。

| 用户操作 | Platform 内部步骤 | 下游调用 | 一致性策略 |
| --- | --- | --- | --- |
| 创建 | RBAC + `GetArtifact` 预检 → 生成 `workspace_name="ws-"+crockford32(rand40bit)` → 注入 `AXISML_WORKSPACE_BASE_URL` env / `spec.route` + `scheduling{poolName, unitName, quota}` + PVC `size`/`storageClass` | `compute.CreateService(kind=workspace, body)` | 单点透传；compute 同事务保证 MLService + PVC 同失同成、内部展开 pool/unit |
| 停止 / 启动 | `RequireWorkspaceOwner` → 同 §4.3：`stop`=scale 0（记 `last-replicas`，工作区恒为 1）、`start`=scale 回 | `ScaleService` | 幂等；`Deleted` → `409 workspace-deleted` |
| 删除 | 校验 `kind==workspace` 防误删 service → 透传（带 `?deletePvc=`，默认 true） | `DeleteService` | 派生 K8s 资源 + PVC 由 compute 级联清理 |
| 浏览器接入 | 颁发 `aud=axisml-workspace` 短 TTL access JWT（`--workspace-access-jwt-ttl`，上限 24h） | —（Platform 自签） | 数据面网关验签放行 |
| 列表 | 同 §4.2；`kind=workspace` 过滤下推 | `ListServices(ns, kind=workspace, owner?)` | 同 §4.2 |

关键不变量：Platform 不为工作区建任何 PG 表；「这是工作区」由 compute `services.kind='workspace'` 直接表达；Platform 自身**不直接调用 K8s API**，PVC 生命周期由 compute 接管。寻址 `(tenant_name, workspace_name)`，URL `/api/v1/workspaces/{name}`。

### 4.5 制品编排

下游：artifacts。三个 Kind（`model` / `image` / `dataset`）共享同一组 Platform 端点形态，按 Kind 分子表暴露给前端；共用编排骨架见 §4.5.1，Kind 专属差异见 §4.5.2。

#### 4.5.1 跨 Kind 共骨架

| 用户操作 | Platform 内部步骤 | 下游调用 | 一致性策略 |
| --- | --- | --- | --- |
| 上传（initiate） | RBAC `user@self`+ → 透传 spec（artifacts handler 做 Kind 级硬校验，失败 4xx 反馈） | `artifacts.InitiateUpload(ns, kind, name, body)` | 返 `{artifact, upload}`：`uri` / `uploadCredentials` 直接透传，前端按 §9.4 渲染上传指引 |
| 完成（complete） | RBAC `@owner`+ → 校验 `digest` 非空 → 透传 | `artifacts.CompleteUpload(ns, kind, name, version, {digest, claim?})` | 单点透传；`DigestMismatch` / `Failed` 由下游 4xx 反馈 |
| 获取下载凭证（resolve `usage=download`） | RBAC `user@self`+ → 透传 | `artifacts.Resolve(usage=download)` | 1h TTL pull token / S3 STS；前端复制即用 |
| 列表 | §5.2 解析 active tenant → §5.3 单租户 / 跨租户合并 → **再并入 `axisml-system` 的 `visibility=public` 行** | `ListArtifactsByKind(ns, kind, ...)` + `ListArtifactsByKind(axisml-system, kind, visibility=public)` | 跨租户部分失败 → `partial=true` |
| 单条详情 | URL `{tenant}/{name}/{version}` 直接拼下游 tuple → 透传 | `GetArtifact(ns, kind, name, version)` | 410 透传（软删后 tuple 永不复用） |
| 编辑展示元数据 | RBAC `@owner` / `tenant-admin@self` → 透传（仅 `displayName` / `description` / `labels` / `annotations` 可改） | `artifacts.UpdateArtifact(...)` | `Deleting`/`Deleted` → `409 ArtifactTerminal`；其它字段 `400 ImmutableField`；`labels`/`annotations` 整体替换语义（见 [artifact-hub.md §6](artifact-hub.md#6-接口契约)） |
| 删除 | RBAC `@owner` / `tenant-admin@self`+ → 透传 | `artifacts.DeleteArtifact(...)` | 软删；GC 异步清后端；`(ns, kind, name, version)` 永不复用 |

**消费侧预检**：Job / Service / Workspace 创建时对引用的制品调 `GetArtifact` 校验存在且 `status=Ready`（校验失败 `400` 阻断上游创建）。制品 URI 注入与凭证由 **operator handler 侧** `resolve?usage=inspect` 完成（Pod 经 per-tenant ServiceAccount 默认携带的 imagePullSecret / Secret 拉取 bytes）——Platform 不调 `resolve?usage=inspect`，该路径专属集群内 operator（[artifact-hub.md §5.2](artifact-hub.md#52-读路径resolve)）。

**关键不变量**：
- 前端寻址 `(tenant, name, version)` 三元组；URL `/api/v1/{kind-plural}/{tenant}/{name}/{version}`；`{kind-plural} ∈ {models, images, datasets}`。url path 直接拼成 artifacts 寻址 tuple；DTO 仍带 `id` 字段供 label / 反向引用，但不作 url 一级 key。
- artifact `namespace` 字段 = `tenant_name`（与 compute 对齐），无需任何 namespace 解析。
- `(name, version)` 创建后不可变；spec / digest 进入 `Ready` 冻结；改 spec = 上传新版本。
- Platform 不为 artifact 建任何视图表；状态 / digest / labels 始终回源 artifacts。

#### 4.5.2 Kind 专属编排差异

| Kind | StorageKind | artifacts handler 必填字段（参考） |
| --- | --- | --- |
| `model` | `oci`（zot） | `spec.framework` / `spec.format` |
| `image` | `oci`（zot） | `spec.purpose` |
| `dataset` | `s3`（RustFS） | `spec.format` |

> Platform 不验 spec；上表是 artifacts handler 的硬校验，Platform 仅透传，失败时由下游 4xx 直传客户端。

**全局可见制品（visibility=public）**：`system-admin` 在内置 namespace `axisml-system`（保留 tenant）下创建制品并设 `visibility=public`；Workspace / Job / Service 创建表单的镜像 / 模型 / 数据集下拉**合并展示当前租户 + `axisml-system` 的 public 行**。Platform 仅在 LIST 阶段做合并，不为「共享」建任何 PG 表；写入路径同普通制品，权限由 artifacts RBAC 兜底（仅 `system-admin` 能写 `axisml-system` 下制品）。

UI 设计见 [wireframe.md §9](../wireframe.md#9-制品中心-数据集--模型--镜像)。

### 4.6 资源池编排

下游：cluster-manager（ResourcePool CRD CRUD；units 为 pool 内嵌数组）+ compute（删除前置阻断的用量反查）。

| 用户操作 | Platform 内部步骤 | 下游调用 | 一致性策略 |
| --- | --- | --- | --- |
| 资源池 CRUD | RBAC `system-admin`；`PATCH` 拦截 `name` 不可变 | `clustermanager.{Create,Update,Delete}ResourcePool` | 单点透传；内部映射 K8s CR 操作 |
| 资源池删除前置 | 按 `labels.axisml.io/resource-pool=<name>` 统计活跃 Job / Service 用量 = 0 校验 | `compute.CountActiveWorkloads(label)` | Platform 自做阻断（>0 → `409 pool-in-use`）；units 跟随 pool 一起删，无独立 unit-in-use 检查 |
| 资源单元 CRUD | RBAC `system-admin`；命名约定由 cluster-manager 兜底 | `clustermanager.{Create,Update,Delete}ResourceUnit`（内部 patch `spec.units[]`） | 单点透传 |
| 资源单元删除前置 | 按 `labels.axisml.io/resource-unit=<name>` 统计活跃用量 | `compute.CountActiveWorkloads(label)` | >0 → `409 unit-in-use` |
| 列表（全局可见） | 已登录可读；写仅 `system-admin` | `clustermanager.ListResourcePools` | 单次 list 返回 pool + 内嵌 units，无需二次聚合 |

关键不变量：Platform 不为池 / 单元建表；Node label / taint 由管理员 `kubectl` 维护，UI 不下发；pool/unit 持久化全在 K8s etcd（ResourcePool CR），Platform 透明。资源池 / 单元为全集群对象，不按租户过滤。

> **删除前置阻断的用量反查**：pool / unit 是集群级、Job / Service 按租户分区，活跃用量经 compute 的 `CountActiveWorkloads(labelSelector)`（忽略 namespace 分区的按 label 计数）单次得出，无需 Platform 跨租户扇出。

### 4.7 Dashboard 编排

下游：compute（KPI 计数 / 租户配额用量 / 工作负载时序）+ artifacts（模型计数）+ cluster-manager（集群容量 / 集群时序）。两个端点：`GET /api/v1/dashboard/overview`（KPI + 资源用量快照）与 `GET /api/v1/dashboard/metrics`（时序图）。作用域由入站 `X-Axisml-Tenant` 头决定（§5.2），指标口径见 [monitoring.md](../monitoring.md)。Platform 不直连 Prometheus——时序由 compute / cluster-manager 内部查 Prometheus 后以 `MetricSeries` 回传。

| 数据块 | 全局视图（`system-admin` · 无 header） | 租户视图（任意成员 · 带 header） |
| --- | --- | --- |
| KPI 计数 | 跨租户 fanout（§5.3）：`tenantCount`/`activeTenantCount`(compute) · `activeJob`/`runningService`/`runningWorkspace`(compute) · `modelCount`(artifacts，含 public) | 单租户 LIST 计数（同字段，无「租户」卡） |
| 资源用量 gauge | **集群容量**：`clustermanager.GetClusterCapacity` → GPU/CPU/内存 `used/total`（可分配口径） | **配额用量**：`compute.GetQuotaUsage` 聚合 → `used / 本租户跨池 Σ quota.max`，≥90% 加 `⚠` |
| 时序图 | 集群级序列 `clustermanager.GetClusterMetrics`；工作负载并发 `compute.GetWorkloadMetrics`（集群级） | `compute.GetWorkloadMetrics`（注入 tenant label 收敛本租户） |

**降级与失败**：

| 场景 | 触发 | 呈现 |
| --- | --- | --- |
| 集群容量为 `null` | cluster-manager 容量聚合未就绪 | gauge 显示 `—` + hover「指标同步中」 |
| 配额用量为 `null` | compute Tenant Informer cache 未同步（[compute-service.md §5.3](compute-service.md#53-状态回流informer)） | 同上 |
| metrics 查询失败 | compute / cluster-manager 指标端点返 `502` | 图区占位「指标服务暂不可用」，KPI / gauge 不受影响 |
| 跨租户聚合部分失败 | overview `partial=true` | 页顶黄条「N 个租户暂时不可达，显示其余结果」 |

KPI + gauge 默认 30s 轮询；时序图随 range 选择器（`1h/24h/7d`）重查。具体 `metric` key 与 PromQL 模板由 [monitoring.md](../monitoring.md) 统一定义、在 compute / cluster-manager 内执行，Platform 与 UI 均不内嵌 PromQL。

## 5. 关键机制

### 5.1 跨服务调用模型

每下游一个 typed client 子包，强类型方法。共享约定：

- **身份注入**：所有出站请求自动携带 `X-Axisml-User: <username>` 头；下游信任此头，只做审计与 ownership 归属，不做角色级鉴权。
- **超时**：默认 30s；幂等读可有限指数退避；写不自动重试。
- **错误透传**：4xx 透传 problem；5xx 包装为 `type=https://axisml.io/errors/upstream-failure` 并附下游服务名。
- **可观测性**：每次调用打 zap 日志 + `platform_upstream_request_total{service,method,status}`（`service ∈ {cluster-manager, compute, artifacts}`）。

### 5.2 上下文解析

**Active tenant 来源**：单租户操作（`/api/v1/jobs|workspaces|services|models|images|datasets` 及子路径）的当前 tenant 不在 URL 里，由请求头 `X-Axisml-Tenant: <name>` 携带。RBAC 中间件按下表处理：

| 端点形态 | header 缺省 | header 存在 |
| --- | --- | --- |
| list（如 `GET /api/v1/jobs`） | `system-admin` 走 §5.3 跨租户 fanout；非 admin → `400 active-tenant-required` | scoped 到该 tenant；调用方需在此 tenant 有 binding（或为 `system-admin`），否则 `404` |
| create（如 `POST /api/v1/jobs`） | `400 active-tenant-required` | 必须有 `user@self`+ binding |
| name 寻址 detail（`/api/v1/{jobs\|services\|workspaces}/{name}/...`） | `400 active-tenant-required` | 用 header 取 `tenant_name` 作分区键后再用 `name` 寻址 |
| tuple 寻址 detail（`/api/v1/{kind}/{tenant}/{name}/{version}`） | URL 内已带 tenant，header 忽略 | 同上 |
| 租户 / 资源池管理路径（`/api/v1/tenants/{name}/...`、`/api/v1/resource-pools/...`） | URL 内已带标识或为全集群对象，header 忽略 | 同上 |
| dashboard（`/api/v1/dashboard/*`） | `system-admin` → 全局视图；非 admin → `400 active-tenant-required` | 收敛到该 tenant 视图 |

**下游 namespace**：compute / artifacts 的 URL `{namespace}` 段直接用 `tenant_name`，**Platform 全程不解析 K8s namespace**。pool/unit 展开、ElasticQuota 名组装、PVC 生命周期均由 compute 内部完成，Platform 只透传名字三元组。`compute.GetTenant` 仅服务于租户详情 / 配额 Tab 展示，不进入创建编排。

### 5.3 列表跨租户合并

当 `system-admin` 不带 `X-Axisml-Tenant` 调 list / dashboard 端点时进入此路径：按 RBAC 取可见租户集合 → 并行 LIST → 内存合并。**partial 失败策略**：单租户失败不中断整体，响应附 `partial=true` + `error.detail`；前端列表头黄条提示。带 header 的请求只查单租户，走快速路径。制品 list 额外并入 `axisml-system` 的 `visibility=public` 行（§4.5.1）。

### 5.4 失败语义

Platform 端不持有任何业务数据，且**不直接调用 K8s API**——所有 mutation 是单点透传下游错误：

- 创建 / 更新 / 删除：失败直接 4xx / 5xx 透传；无 outbox 无补偿队列。
- 跨租户列表 / dashboard 聚合：`partial=true` 标记，不阻断主响应。

下游各自的强一致策略（compute Outbox + reconciler、compute 工作区 PVC 同事务派生、artifacts 两阶段写）对 Platform 透明。

### 5.5 扩展元数据写入约定

Platform 需要在下游对象上挂载自定义元数据（审计标记、UI 状态、`last-replicas` 副本基线、批次 ID、外部系统关联键等），统一通过下游业务服务的 `labels` / `annotations` 字段写入，**不直接 patch CR**：

| 维度 | 约定 |
| --- | --- |
| 写入路径 | `compute.{Create,Update}{Tenant,Job,Service}` / `artifacts.UpdateArtifact` 请求体携带 `labels` / `annotations` |
| 存储位置 | 下游 PG 表的 `labels jsonb` + `annotations jsonb` 列（[database.md §1.6](../database.md#16-扩展元数据-labels--annotations)） |
| Key 命名空间 | Platform 内部固定 `platform.axisml.io/<key>` 前缀（如 `platform.axisml.io/last-replicas`）；终端用户透传走 `user.axisml.io/<key>` 或无前缀 |
| 同步语义 | 修改不触发 CR patch（不 `+generation`），不引发 reconcile；纯 PG mutation，写后立即可读 |
| 删除 | 软删行保留扩展位以支持 retention 期内恢复；硬删时一并清理 |

**典型用途 · 服务 / 工作区副本基线**：`stop` 前把当前副本数写入 `annotations[platform.axisml.io/last-replicas]`，`start` 时读回恢复（缺失 fallback 1）。

**反模式**：不向 K8s CR 的 `metadata.{labels,annotations}` 写业务扩展位（Platform 本就不直接调 K8s API）；不在 Platform PG 镜像下游业务对象元数据（保持 Platform PG 仅覆盖身份 / 授权 / 会话 / 审计）。

RBAC 中间件装配细节归 [auth.md](../auth.md)，Platform 在路由层挂载 `RequireSystemAdmin` / `RequireTenantRole` / `RequireJobOwner` / `RequireServiceOwner` / `RequireWorkspaceOwner` 标准件（均按 `name` 寻址）。

## 6. 接口契约

| 类别 | 内容 | 引用 |
| --- | --- | --- |
| 对外 REST | 业务 tag：`Auth` / `Tenants` / `Quotas` / `Members` / `Jobs` / `Services` / `Workspaces` / `Models` / `Images` / `Datasets` / `ResourcePools` / `ResourceUnits` / `Dashboard`；jobs / services / workspaces 一律 `/api/v1/{kind}/{name}` name 寻址；制品走 `/api/v1/{kind-plural}/{tenant}/{name}/{version}` tuple 寻址；系统类 tag（`Users` / `Audit` / `Health`）见 yaml | [apis/platform.yaml](../apis/platform.yaml) |
| 状态 | 不暴露任何 K8s CR；下游运行态字段（phase / conditions / status / quota 用量）作为只读字段透传 | — |
| 错误格式 | HTTP 标准状态码 + RFC 7807 `application/problem+json`；下游 problem 由 typed client 解析后透传或包装 | — |
| 流式 | 日志 / 事件 `follow=true` 采用 `text/event-stream` SSE；非 follow 用 `text/plain` chunked | — |
| 身份头 | 入站校验主登录 JWT；单租户操作的 active tenant 由入站 `X-Axisml-Tenant` 头携带（§5.2）；出站注入 `X-Axisml-User` | [auth.md §7](../auth.md#7-下游身份透传) |
| access JWT | 工作区 / 在线服务数据面入口走独立 access JWT（`aud=axisml-workspace` / `axisml-inference`），由 Platform 颁发、Envoy SecurityPolicy 验签 | [auth.md §5](../auth.md#5-access-jwt) |
| Prometheus 指标 | `platform_*` 系列自身指标；模块级清单见 [monitoring.md §5](../monitoring.md#5-platform-层指标) | — |

## 7. 依赖

| 依赖 | 用途 | 备注 / 契约依赖 | 引用 |
| --- | --- | --- | --- |
| PostgreSQL | 身份 / 授权 / 会话 / 审计；与 compute / artifacts 共享同一 DB，按表名前缀隔离 | — | [database.md §4](../database.md#4-platform) |
| Envoy Gateway | 唯一外部入口；TLS 终止 / 路由匹配；数据面 access JWT SecurityPolicy | — | [infra.md](../infra.md) |
| cluster-manager | ResourcePool / Unit CRUD 的 REST 入口；Dashboard 集群容量分母与集群时序 | `GetClusterCapacity`（GPU/CPU/内存 used/total 可分配口径）；`GetClusterMetrics`（集群级时序，内部查 Prometheus） | [cluster-manager.md](cluster-manager.md) |
| compute | Tenant / Quota / Job / Service / Workspace 权威；写路径由 Outbox + reconciler 保证强一致；在线服务 / 工作负载运行指标代理 | 创建体接收 `scheduling{poolName,unitName,quota}` 名字对，compute 校验 `(pool,quota)∈tenant.quotas` 并组装 ElasticQuota 名；`Suspended` phase + 创建闸门（`409 tenant-suspended`）；`CountActiveWorkloads(labelSelector)` 活跃用量计数；`GetServiceMetrics` / `GetWorkloadMetrics`（backend-aware 时序，内部查 Prometheus） | [compute-service.md](compute-service.md) |
| artifacts | 模型 / 镜像 / 数据集元数据；两阶段写（initiate → 直推 → complete） | 消费侧创建走 `GetArtifact` 预检 Ready；`resolve?usage=inspect` 专属 operator | [artifact-hub.md](artifact-hub.md) |

## 8. 运行时形态

| 维度 | 值 |
| --- | --- |
| 进程 | 单二进制 `axisml-platform`；子命令 `serve` / `migrate` / `bootstrap` |
| 副本 | 当前 chart 默认 `replicas=1`；Platform 服务无状态，后续真实后端 / 前端镜像拆分后可水平扩 |
| 启动子命令 | `serve` 启动 HTTP API + 后台任务；`migrate` 执行 GORM 迁移（部署 init container）；`bootstrap` 一次性创建内置角色、初始 `system-admin` 账号（默认 `admin/admin`，首次登录强制改密；可由 `AXISML_BOOTSTRAP_PASSWORD` 覆盖）以及内置 `axisml-system` 租户（承载 `visibility=public` 制品） |
| 暴露端口 | 当前 chart 以 nginx placeholder 暴露 HTTP `:8080`；真实 Platform Backend 目标为 API `:8080`、Metrics `:8081`、Probes `:8082`（`/healthz` / `/readyz`），JWKS `/.well-known/jwks.json` 走 ClusterIP，不经 Gateway |
| RBAC scope | 无 K8s API 需求（PVC / CR / Pod / 节点容量 / 指标查询透传一律下沉下游服务） |
| Helm values / 镜像 | 详见 [deployment.md](../deployment.md) |

## 9. 相关引用

- [overview.md](../overview.md) — 控制平面拓扑
- [auth.md](../auth.md) — 身份与鉴权契约、access JWT、中间件
- [database.md](../database.md) — Platform PG schema（§4）
- [deployment.md](../deployment.md) — Helm / 部署
- [monitoring.md](../monitoring.md) — Metrics、告警与 service metrics PromQL 模板
- [infra.md](../infra.md) — Envoy Gateway / kube-prometheus-stack
- [wireframe.md](../wireframe.md) — 所有 UI 设计（列表 / 详情 / 表单）
- [apis/platform.yaml](../apis/platform.yaml) — REST 契约源
- [cluster-manager.md](cluster-manager.md) / [tenant-operator.md](tenant-operator.md) / [compute-service.md](compute-service.md) / [compute-operator.md](compute-operator.md) / [artifact-hub.md](artifact-hub.md)
