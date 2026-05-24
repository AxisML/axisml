# AxisML Platform 概要设计

## 1. 定位与边界

AxisML 系统中唯一直接面向用户的层；承担身份接入、业务编排与视图层映射，将用户视角操作翻译为对 cluster-manager / compute / artifacts 的下游调用。

| 做 | 不做 |
| --- | --- |
| 唯一外部 HTTP 入口（前端 SPA + 后端 REST） | 直接管理 K8s 资源（除受限 PVC，见 §5.4） |
| 身份认证 / JWT 颁发 / RBAC 校验 | 持有 Tenant / Quota / Job / Service 权威 (→ [compute.md](compute.md)) |
| 跨服务业务编排（创建/列表合并/克隆等） | 持有 ResourcePool / ResourceUnit 词汇 (→ [cluster-manager.md](cluster-manager.md)) |
| 视图层映射（Tenant view ↔ namespace；workspace ↔ MLService(kind)） | 持有 Artifact 权威 (→ [artifacts.md](artifacts.md)) |
| 工作区 PVC 直管（Platform 唯一直接操作 K8s 的范围） | 二次缓存下游业务字段（无视图缓存表） |

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
       └─┬──────┬──────┬──┬───────┐
         │      │      │  │       │
         ▼      ▼      ▼  ▼       ▼
   cluster-   compute  artifacts  Prometheus
   manager                        (kube-prom)
```

外部流量必须经 Envoy Gateway 进入 Platform；下游全部 ClusterIP。

### 2.2 内部结构

```
┌──────────────────────── AxisML Platform ────────────────────────┐
│  Frontend (TS + React + Vite)                                   │
│        │                                                        │
│        ▼ REST                                                   │
│  Backend (Go + Gin + Cobra)                                     │
│   ├─ auth (JWT + RBAC + IdentityProvider 接口)                  │
│   ├─ orchestrator (跨服务编排)                                  │
│   ├─ business modules (tenant / job / service / workspace /     │
│   │                    model / resourcepool / dashboard)        │
│   └─ typed clients (clustermanager / compute / artifacts /      │
│                     prometheus)                                 │
│        │                              │                         │
│        ▼                              ▼                         │
│  Platform PG                    下游 ClusterIP 调用             │
│  (users / roles / sessions /                                    │
│   user_tenant_roles / audit_logs)                               │
└─────────────────────────────────────────────────────────────────┘
```

## 3. 核心模型

Platform 自有实体仅覆盖**身份 / 授权 / 会话 / 审计**四类，完整字段与索引见 [database.md §5](../database.md#5-platform)。

| 实体 | 含义 | 标识键 | 备注 |
| --- | --- | --- | --- |
| User | 内置用户 | `id` / `username` unique | bcrypt 密码；OIDC 预留 |
| Role | RBAC 角色 | `name` (`system-admin` / `tenant-admin` / `user`) | 内置三档；不开放自建 |
| Permission | 权限位 | `name` | 角色 × 权限矩阵详见 [auth.md §3](../auth.md#3-rbac-角色) |
| UserTenantRole | 用户↔租户成员关系 | `(user_id, tenant_name, role_id)` | `tenant_name` 引用 compute `tenants.name` |
| Session | JWT 会话 / 刷新 token | `id` | TTL 与 JWKS 由 auth 模块管理 |
| AuditLog | 操作流水 | `id` | `action` / `target` / `actor` / `payload` |

**视图层映射实体**：Tenant view ↔ `(tenant_name, k8s_namespace)` 二元组——Platform 调下游前调一次 `compute.GetTenant(name)` 解析 `spec.namespace.name`（K8s namespace，PVC 操作需要）与 quotas，不落 PG。注意 compute / artifacts 的 REST URL 中 `{namespace}` 段 = `tenant_name`，**不是** K8s namespace；K8s namespace 仅 Platform 直接操作 K8s 资源（如 PVC）时用到。

## 4. 核心功能

每节定义编排动作；UI 字段与布局见 [wireframe.md](../wireframe.md)。

### 4.1 租户编排

下游：compute。

| 用户操作 | Platform 内部步骤 | 下游调用 | 一致性策略 |
| --- | --- | --- | --- |
| 创建租户 | RBAC `system-admin` → 字段校验 | `CreateTenant` | 单点透传；4xx/5xx 直返 |
| 暂停 / 恢复 | RBAC → 透传 | `SuspendTenant` / `UnsuspendTenant` | 单点透传 |
| 软删 / 恢复 | RBAC + 成员校验（非空 → `409 tenant-has-members`） | `DeleteTenant` / `RestoreTenant` | 成功后级联清理 `user_tenant_roles` |
| 配额 CRUD | RBAC `tenant-admin@self` → 拦截 `(pool, name)` 不可变字段 | `Add/Update/DeleteQuota` | `400 immutable-field` Platform 兜底 |
| 成员管理 | RBAC + 自我保护（不能移除最后一个 `tenant-admin` → `409 last-tenant-admin`） | — (仅 Platform PG) | `user_tenant_roles` 内事务 |
| 列表 | 按角色裁剪：非 `system-admin` 取绑定 tenant 集合 | `ListTenants`（query 下推） | 单租户失败 → `partial=true` |

关键不变量：Platform 不为租户建任何视图表；`name` / `namespace.name` / 配额 `(pool, name)` 创建后不可变。UI 设计见 [wireframe.md](../wireframe.md)。

### 4.2 计算任务编排

下游：compute。

| 用户操作 | Platform 内部步骤 | 下游调用 | 一致性策略 |
| --- | --- | --- | --- |
| 提交任务 | 解析 namespace + quotas → `clustermanager.GetResourcePool(pool)` / `GetResourceUnit(pool, unit)` 取词汇 → 展开为 `nodeSelector` / `tolerations` / `requests` / `limits` 写进请求体 → 拼 `axisml-<tenant>-<pool>-<quota>` → `artifacts.Resolve` 校验镜像/模型 | `clustermanager.Get{ResourcePool,ResourceUnit}` + `compute.CreateJob(ns, body{expanded})` | 单点透传 |
| 取消 | RBAC `@owner` 或更高 → 透传 | `CancelJob` | 状态合法性由 compute 4xx 反馈 |
| 删除 | RBAC + 透传 | `DeleteJob` | 同上 |
| 列表 | §5.2 解析 active tenant：header 在 → 单租户透传；`system-admin` 无 header → §5.3 跨租户合并 | `ListJobs(ns, ...)` | 跨租户路径部分失败 → `partial=true` + `error.detail` |
| 副本/事件/日志 | `RequireJobOwner` 校验 → 透传（含 SSE `follow=true`） | `GetJob{Replicas,Events,Logs}` | 流式 chunked 透传 |
| register-model | RBAC `@owner`+ → 校验 `phase=Succeeded` → 解析 `spec.outputs[]` 选中项（或显式 `outputPath` override）→ 反查 PVC + sourcePath → 调 `artifacts.InitiateUpload(kind=model)` 并打 provenance labels → 返 `{artifact, upload, provenance}`，字节由客户端工具异步推送 | `compute.GetJob` + `artifacts.InitiateUpload` | 详见 [§4.5.3](#453-register-from-job计算任务--模型) |

关键不变量：任务标识为 `(tenant_name, job_name)`，`tenant_name` 由 `X-Axisml-Tenant` 头携带（详见 §5.2），URL 形态 `/api/v1/jobs/{name}`；Job spec 不可变——「编辑」= 新建。UI 设计见 [wireframe.md](../wireframe.md)。

### 4.3 在线服务编排

下游：compute (services) + artifacts (resolve) + Prometheus。

| 用户操作 | Platform 内部步骤 | 下游调用 | 一致性策略 |
| --- | --- | --- | --- |
| 创建 | 解析 namespace + quotas → `clustermanager.GetResourcePool` / `GetResourceUnit` 取词汇并展开为 nodeSelector / tolerations / requests / limits → `artifacts.Resolve` 校验 modelRef/image → 拼 quota 名 → `route.path==""` 时自动拼 `/services/<tenant>/<name>/` 并注入 `AXISML_SERVICE_BASE_URL` env | `clustermanager.Get{ResourcePool,ResourceUnit}` + `compute.CreateService(ns, body{kind=service, expanded})` | 单点透传 |
| 扩缩容 / start / stop | `RequireServiceOwner` → 翻译 `/start` = 上次>0 replicas（查 audit_logs，缺失 fallback 1），`/stop` = 0 | `ScaleService` | 幂等；`Deleted` → `409 service-deleted` |
| 删除 | 先 `GetServiceByID` 校验 `kind==service` 防误删工作区 | `DeleteService` | 派生 K8s 资源由 ownerReference 级联 |
| 路由 / 访问 | `auth.type=jwt` 时颁发 `aud=axisml-inference` 短 TTL JWT | — | `route-auth-mismatch` 时 `409` |
| 指标查询 | 按 backend 选 PromQL 模板（PromQL 见 [monitoring.md](../monitoring.md#6-业务指标查询service-metrics-端点)） | `prometheus.Query` / `QueryRange` | 查询失败 `502 upstream-failure` |
| clone-with-new-version | 前端语法糖：反填创建表单 → 用户改 `modelRef.version` → 提新 service → 外部切流量 → 旧 service 停服 | `CreateService` + `DeleteService` | MLService `spec.modelRef` 不可变；当前不接管 weighted route |
| 列表 | 同 §4.2；防误删校验 `kind=service` | `ListServices(ns, kind=service)` | 跨租户路径部分失败 → `partial=true` |

关键不变量：寻址 `services.id` (uuid)；spec 除 `roles[*].replicas` 外不可变。UI 设计见 [wireframe.md](../wireframe.md)。

### 4.4 工作区编排

下游：compute (services with `kind=workspace`) + K8s API (PVC)。工作区 = 长驻交互式开发容器，**复用** `MLService(native, deployment)`，不引入新 CRD。

| 用户操作 | Platform 内部步骤 | 下游调用 | 一致性策略 |
| --- | --- | --- | --- |
| 创建 | RBAC + `clustermanager.Get{ResourcePool,ResourceUnit}` 展开为 nodeSelector / requests / limits + Image 解析 → 生成 `mlservice_name="ws-"+crockford32(rand40bit)` → 注入 volumes / `AXISML_WORKSPACE_BASE_URL` env / `spec.route` → 创建 MLService → 由 `service.id` 派生 PVC 名 `axisml-ws-<id 前 8>-data` → 创建 PVC | `clustermanager.Get{ResourcePool,ResourceUnit}` + `compute.CreateService(kind=workspace, expanded)` + K8s PVC `create` | PVC 失败 → 调 `DeleteService` 回滚；回滚失败 → 写 audit_logs |
| start / stop | `RequireWorkspaceOwner` → 翻译为 `replicas=1/0` | `ScaleService` | 幂等；`Deleted` → `409 workspace-deleted` |
| 删除 | 校验 `kind==workspace` 防误删 service → 删 MLService → 按 `deletePvc`（默认 true）删 PVC | `DeleteService` + K8s PVC `delete` | 404 幂等 |
| 浏览器接入 | 颁发 `aud=axisml-workspace` 短 TTL JWT（`--workspace-access-jwt-ttl`，上限 24h） | — | — |
| 列表 | 同 §4.2；`kind=workspace` 过滤下推 | `ListServices(ns, kind=workspace, owner?)` | 同 §4.2 |

关键不变量：Platform 不为工作区建任何 PG 表；「这是工作区」由 compute `services.kind='workspace'` 直接表达；PVC 名 deterministic 派生自 `services.id` 无需存储。UI 设计见 [wireframe.md](../wireframe.md)。

### 4.5 制品编排

下游：artifacts。三个 Kind（`model` / `image` / `dataset`）共享同一组 Platform 端点形态（仿 Models tag），按 Kind 分子表暴露给前端；跨 Kind 共用的编排骨架先列在 §4.5.1，Kind 专属差异落 §4.5.2。

#### 4.5.1 跨 Kind 共骨架

| 用户操作 | Platform 内部步骤 | 下游调用 | 一致性策略 |
| --- | --- | --- | --- |
| 上传（initiate） | RBAC `user@self`+ → 解析 tenant → namespace → 透传 spec（artifacts handler 做 Kind 级校验，失败 4xx 反馈） | `artifacts.InitiateUpload(ns, kind, name, body)` | 返 `{artifact, upload}`：`uri` / `uploadCredentials` 直接透传，前端按 §7.2.2 渲染上传指引 |
| 完成（complete） | RBAC `@owner`+ → 校验 `digest` 非空 → 透传 | `artifacts.CompleteUpload(ns, kind, name, version, {digest, claim?})` | 单点透传；`DigestMismatch` / `Failed` 由下游 4xx 反馈 |
| 解析（resolve `usage=inspect`） | 内部编排子步骤（Job / Service / Workspace 创建前调用）；**不对外暴露独立 REST 端点** | `artifacts.Resolve(ns, kind, name, version, usage=inspect)` | 校验失败 `400` 阻断上游创建；返回的 `auth_hint` 用于 K8s spec 注入 |
| 获取下载凭证（resolve `usage=download`） | RBAC `user@self`+ → 透传 | `artifacts.Resolve(usage=download)` | 1h TTL pull token / S3 STS；前端复制即用 |
| 列表 | §5.2 解析 active tenant：header 在 → 单租户透传；`system-admin` 无 header → §5.3 跨租户合并 | `ListArtifactsByKind(ns, kind, ...)` | 跨租户路径部分失败 → `partial=true` + `error.detail` |
| 单条详情 | URL `{tenant}/{name}/{version}` 直接拼下游 tuple → 透传 | `GetArtifact(ns, kind, name, version)` | 410 透传（软删后 tuple 永不复用） |
| 编辑展示元数据 | RBAC `@owner` / `tenant-admin@self` → 透传（`displayName` / `description` / `labels` / `annotations` 可改，其它字段 `400 immutable-field`） | `artifacts.UpdateArtifact(ns, kind, name, version, body)` | 单点透传；`Deleting`/`Deleted` → `409 ArtifactTerminal`；`labels` / `annotations` 整体替换语义（见 [artifacts.md §6](artifacts.md#6-接口契约)） |
| 删除 | RBAC `@owner` / `tenant-admin@self`+ → 透传 | `artifacts.DeleteArtifact(ns, kind, name, version)` | 软删；GC 异步清后端；`(ns, kind, name, version)` 永不复用 |

**关键不变量**：
- 前端寻址 `(tenant, name, version)` 三元组；URL `/api/v1/{kind-plural}/{tenant}/{name}/{version}`；`{kind-plural} ∈ {models, images, datasets}`。Platform 不维护 id 反查表——url path 直接拼成 artifacts 寻址 tuple；DTO 仍带 `id` 字段供 label / 反向引用使用，但不作为 url 一级 key。
- artifact `namespace` 字段 = `tenant_name`（与 compute 对齐），无需 GetTenant 解析；只在 PVC / quota 名拼接等场景才需要 `compute.GetTenant(tenant).spec.namespace.name`。
- `(name, version)` 创建后不可变；spec / digest 进入 `Ready` 冻结；改 spec = 上传新版本。
- Platform 不为 artifact 建任何视图表；状态 / digest / labels 始终回源 artifacts。

#### 4.5.2 Kind 专属编排差异

| Kind | StorageKind | artifacts handler 必填字段（参考） | 跨制品引用懒校验 | 自动注册路径（非 UI） |
| --- | --- | --- | --- | --- |
| `model` | `oci` (zot) | `spec.framework` / `spec.format` | `baseModelRef` / `trainingDatasetRef` 由 artifacts handler 触发 `Resolve` 探活；失败 4xx 反馈 | 计算任务详情「注册为模型」按钮（详见 [§4.5.3](#453-register-from-job计算任务--模型)） |
| `image` | `oci` (zot) | `spec.purpose` | — | — |
| `dataset` | `s3` (RustFS) | `spec.format` | — | — |

> Platform 不验 spec；上表「必填字段」列出的是 artifacts handler 的硬校验（spec / 引用懒校验），Platform 仅做透传，校验失败时由下游 4xx 直传到客户端。

**共享镜像约定**：`system-admin` 在专用 namespace（约定名 `system-images`）维护「平台基础镜像」；Workspace / Job / Service 创建表单的镜像下拉合并展示当前租户 + system 命名空间。Platform 仅在 LIST 阶段做合并，不为「共享」语义建任何 PG 表。

UI 设计:本期原型未覆盖制品中心,占位见 [wireframe.md §3 占位页骨架](../wireframe.md#3-占位页骨架);详细 mockup 待补,见 [§6.1 待补 ASCII mockup](../wireframe.md#61-待补-ascii-mockup)。

#### 4.5.3 register-from-job（计算任务 → 模型）

把训练任务的产物注册成 `model` 制品的桥接路径。前置：MLJob 提交时已在 `spec.outputs[]` 声明产物（[compute-operator.md §4.1.1](compute-operator.md#411-mljob-spec-高层结构)），或调用方在请求体中显式提供 `outputPath` + `volumeName` 覆盖。

**端点**：`POST /api/v1/jobs/{name}/register-model`（[apis/platform.yaml](../apis/platform.yaml) `Jobs` tag `registerJobModel`）；当前 tenant 由 `X-Axisml-Tenant` 头携带，详见 §5.2。

**请求体**（详见 [apis/platform.yaml `JobRegisterModelRequest`](../apis/platform.yaml)）：
- 两选一寻址（互斥）：
  - **声明式**：`outputName` 指向 `MLJob.spec.outputs[]` 某项（须 `kind=model`）；
  - **ad-hoc**：同时给 `outputPath` + `volumeName`；二者缺一报 `400 InvalidOutputSelector`。
- `modelName` + `modelVersion`：制品标识，必填；
- `spec`：`ModelSpec`，必填（`spec.framework` / `spec.format` 由 artifacts handler 强校验，UI 可按 `image` 推断默认）；
- `spec.trainingDatasetRef?`：若 Job role 容器 env 含 `AXISML_DATASET_URI` 且能 resolve 到一个 `Ready` 的 dataset，UI 在打开 modal 时自动反填，用户可改可空。
- `displayName?` / `description?`：可空。

**Platform 内部步骤**（顺序）：

1. RBAC：`@owner` 或本租户 `tenant-admin+`；
2. `compute.GetJob(ns, name)` 拉 Job 全字段；
3. 校验 `job.phase=Succeeded`（其他 → `409 job-not-succeeded`）；
4. 解析产物位置：
   - `outputName` 命中 → 读 `spec.outputs[outputName]` 的 `volumeName` + `sourcePath`；
   - 否则用请求体 `outputPath` + `volumeName`；
   - 反查 `spec.roles[*].template.volumes[name=volumeName]` 取 `persistentVolumeClaim.claimName`（非 PVC 卷 → `400 output-volume-must-be-pvc`）；
5. `artifacts.InitiateUpload(ns, kind=model, name=modelName, body)` → 拿 `{artifact, upload}`；
6. 在 artifact 上打 provenance 标记：
   - `labels` 写入 `platform.axisml.io/source-job-tenant: <tenant>`、`platform.axisml.io/source-job-name: <jobName>`、`platform.axisml.io/source-job-id: <job.id>`（compute `jobs.id` uuid）；声明式分支再加 `platform.axisml.io/source-output: <outputName>`，ad-hoc 分支省略该 label；
   - `annotations` 写入 `platform.axisml.io/registered-by-user: <username>`、`platform.axisml.io/registered-at: <rfc3339>`；
7. 返回 `JobRegisterModelResponse`：`{artifact, upload, provenance: {jobId, jobName, outputName?, pvc, sourcePath}}`，前端在通用上传指引对话框顶部额外渲染「来源：任务 `<tenant>/<jobName>` 输出 `<outputName 或 ad-hoc>`（PVC `<pvc>`，路径 `<sourcePath>`）」（对话框 UI 待补，见 [wireframe.md §6.1](../wireframe.md#61-待补-ascii-mockup)）。

**幂等性**：
- 同一 `(modelName, modelVersion)` 已存在 → `artifacts.InitiateUpload` 透传 `409 ArtifactAlreadyExists`，UI 弹窗引导改版本号；
- `Failed` 行不可复活：用户需 DELETE 旧行后另起版本。

**字节如何上传**：Platform 只返回 artifact 行 + upload credentials + provenance hint，**不**渲染或下发 cli 命令。客户端（typically `axisml-cli`）拿到 credentials 后自行完成两阶段写的第二阶段（push + `complete`）；具体 cli 子命令形态另行设计，不在 Platform 契约内。

**不做**：
- Platform 不主动 push bytes（避免占自身 Pod 资源 / 不持长期文件读权限）；
- Platform 不为「已注册过的 Job」建索引——`artifact.labels[platform.axisml.io/source-job-id]` 即反向索引基础，由独立引用方反查端点（§9.2 后续工作）消费；
- compute-operator 不参与该运行时流程（保持 operator 不知 artifacts 存在的边界；operator 仅在 `Validate(spec)` 阶段对 `spec.outputs[]` 做静态约束）。

### 4.6 资源池 / 单元编排

下游：compute。

| 用户操作 | Platform 内部步骤 | 下游调用 | 一致性策略 |
| --- | --- | --- | --- |
| 资源池 CRUD | RBAC `system-admin`；`PATCH` 拦截 `name` 不可变 | `clustermanager.{Create,Update,Delete}ResourcePool` | 单点透传 |
| 资源池删除前置 | `ListResourceUnits(pool)` > 0 → `409 pool-in-use`；`compute.ListJobs/Services` 用量为 0 校验（按 `labels.axisml.io/pool` 过滤）| 多次 list | Platform 自做阻断，不依赖 cluster-manager 级联 |
| 资源单元 CRUD | RBAC `system-admin`；命名约定由 cluster-manager 兜底 | `clustermanager.{Create,Update,Delete}ResourceUnit` | 单点透传 |
| 资源单元删除前置 | 活跃 Job / Service 用量 > 0 → `409 unit-in-use` | list | Platform 自做阻断 |
| 列表（全局可见） | 已登录即可读；写仅 `system-admin` | `clustermanager.ListResourcePools` / `clustermanager.ListResourceUnits` | 资源单元数聚合并发取 count；单池失败该项标 `-1` |

关键不变量：Platform 不为池/单元建表；Node label / taint 由管理员通过 kubectl 维护，UI 不下发。UI 设计见 [wireframe.md](../wireframe.md)。

## 5. 关键机制

### 5.1 跨服务调用模型

每下游一个 typed client 子包，强类型方法。共享约定：

- **身份注入**：所有出站请求自动携带 `X-Axisml-User: <username>` 头；下游信任此头，只做审计不做鉴权。
- **超时**：默认 30s；幂等读可有限指数退避；写不自动重试。
- **错误透传**：4xx 透传 problem；5xx 包装为 `type=https://axisml.io/errors/upstream-failure` 并附下游服务名。
- **可观测性**：每次调用打 zap 日志 + `platform_upstream_request_total{service,method,status}`。

### 5.2 上下文解析

**Active tenant 来源**：单租户操作（`/api/v1/jobs|workspaces|services|models|images|datasets` 及其子路径）的当前 tenant 不在 URL 里，而是由请求头 `X-Axisml-Tenant: <name>` 携带。RBAC 中间件按下表处理：

| 端点形态 | header 缺省 | header 存在 |
| --- | --- | --- |
| list（如 `GET /api/v1/jobs`） | `system-admin` 走 §5.3 跨租户 fanout；非 admin → `400 active-tenant-required` | scoped 到该 tenant；调用方需在此 tenant 有 binding（或为 `system-admin`），否则 `404` |
| create（如 `POST /api/v1/jobs`） | `400 active-tenant-required` | 必须有 `user@self`+ binding |
| name 寻址 detail（如 `/api/v1/jobs/{name}/...`） | `400 active-tenant-required` | 用 header 解析 namespace 后再用 `name` 寻址 |
| uuid 寻址 detail（如 `/api/v1/workspaces/{id}`） | 用 uuid 自寻址；header 仅做一致性对账 | 同上 |
| tuple 寻址 detail（`/api/v1/{kind}/{tenant}/{name}/{version}`） | URL 内已带 tenant，header 忽略 | 同上 |
| 租户管理路径（`/api/v1/tenants/{name}/...`） | URL 内已带 tenant，header 忽略 | 同上 |

**下游 namespace 解析**：compute / artifacts 的 URL `{namespace}` 段直接用 `tenant_name`，无需解析。仅当需要 K8s 直接操作（PVC 等）或拼 ElasticQuota 名（`axisml-<tenant>-<pool>-<quota>`，其中 `<tenant>` = tenant_name）时调用 `compute.GetTenant(name)` 取 `spec.namespace.name` / quotas 清单，request-scoped memoize，不本地缓存。

### 5.3 列表跨租户合并

当 `system-admin` 不带 `X-Axisml-Tenant` 调用 list 端点时进入此路径：按 RBAC 取可见租户集合 → 并行解析 namespace → 并行 LIST → 内存合并。**partial 失败策略**：单租户失败不中断整体，响应附 `partial=true` + `error.detail`；前端列表头部黄条提示。带 header 的请求只查单租户，走快速路径。

### 5.4 PVC 管理（Platform 仅有的直接 K8s 操作）

工作区 PVC 由 Platform 直管；通过受限 ClusterRole `axisml-platform-pvc`：`pvc` 资源 `get/list/create/delete`，作用域全部 compute namespace（非整集群通配）。命名 deterministic：`axisml-ws-<service.id 前 8>-data`，无需在 PG 存储。

### 5.5 失败语义

Platform 端不持有任何业务数据，**自然无双写一致性问题**——所有 mutation 是单点透传下游错误：

- 创建 / 更新 / 删除：失败直接 4xx / 5xx 透传；无 outbox 无补偿队列。
- 跨租户列表：`partial=true` 标记，不阻断主响应。
- 唯一例外：工作区创建 = MLService 创建 + PVC 创建，PVC 失败需回滚 MLService（见 §4.4）。

下游各自的强一致策略（compute Outbox + reconciler / artifacts 两阶段写）对 Platform 透明。

### 5.6 扩展元数据写入约定

Platform 自身需要在下游对象上挂载自定义元数据（审计标记、UI 状态、批次 ID、外部系统关联键等），统一通过下游业务服务的 `labels` / `annotations` 字段写入，**不直接 patch CR**：

| 维度 | 约定 |
| --- | --- |
| 写入路径 | `compute.{Create,Update}{Tenant,Job,Service}` / `artifacts.{Create,Update}Artifact` 请求体中携带 `labels` / `annotations` |
| 存储位置 | 下游 PG 表的 `labels jsonb` + `annotations jsonb` 列（详见 [database.md §1.6](../database.md#16-扩展元数据-labels--annotations)） |
| Key 命名空间 | Platform 内部固定使用 `platform.axisml.io/<key>` 前缀；终端用户透传字段走 `user.axisml.io/<key>` 或无前缀 |
| 同步语义 | 修改不触发 CR patch（不 `+generation`），不会引发 reconcile；纯 PG mutation，写后立即可读 |
| 大小约束 | 见 [database.md §1.6](../database.md#16-扩展元数据-labels--annotations)；Platform 写入前需校验 |
| 删除 | 软删行保留扩展位以支持 retention 期内恢复；硬删时一并清理 |

**反模式**：

- 不向 K8s CR 的 `metadata.annotations` / `metadata.labels` 写业务扩展位（即使有 PVC 直管的 RBAC，也仅用于工作区 PVC 自身的命名 / selector，不挂业务元数据）；
- 不在 Platform PG 镜像下游业务对象的元数据（保持 Platform 自身 PG 仅覆盖身份 / 授权 / 会话 / 审计四类的不变量）。

RBAC 中间件装配细节归 [auth.md](../auth.md)，Platform 仅在路由层挂载 `RequireSystemAdmin` / `RequireTenantRole` / `RequireJobOwner` / `RequireServiceOwner` / `RequireWorkspaceOwner` 标准件。

## 6. 接口契约

| 类别 | 内容 | 引用 |
| --- | --- | --- |
| 对外 REST | 主要业务 tag：`Auth` / `Tenants` / `Quotas` / `Members` / `Jobs` / `Services` / `Workspaces` / `Models` / `Images` / `Datasets` / `ResourcePools` / `ResourceUnits` / `Dashboard`；系统类 tag（`Users` / `Audit` / `Health`）省略，全量见 yaml | [apis/platform.yaml](../apis/platform.yaml) |
| 状态 | 不暴露任何 K8s CR；下游运行态字段（phase / conditions / status）作为只读字段透传 | — |
| 错误格式 | HTTP 标准状态码 + RFC 7807 `application/problem+json`；下游 problem 由 typed client 解析后透传或包装 | — |
| 流式 | 日志 / 事件 `follow=true` 采用 `text/event-stream` SSE；非 follow 用 `text/plain` chunked | — |
| 身份头 | 入站校验 JWT；单租户操作的 active tenant 由入站 `X-Axisml-Tenant` 头携带（详见 §5.2）；出站注入 `X-Axisml-User` | [auth.md §7](../auth.md#7-下游身份透传) |
| Prometheus 指标 | `platform_*` 系列；模块级清单见 [monitoring.md](../monitoring.md) | — |

## 7. 依赖

| 依赖 | 用途 | 引用 |
| --- | --- | --- |
| PostgreSQL | 身份 / 授权 / 会话 / 审计；与 cluster-manager / compute / artifacts 共享同一 DB，按表名前缀隔离 | [database.md](../database.md) / [infra.md](../infra.md) |
| Envoy Gateway | 唯一外部入口；TLS 终止 / 路由匹配 | [infra.md](../infra.md) |
| cluster-manager | ResourcePool / ResourceUnit 词汇；Platform 在编排前拉取并展开成 nodeSelector / requests / limits 原语 | [cluster-manager.md](cluster-manager.md) |
| compute | Tenant / Quota / Job / Service 权威；写路径由 compute Outbox + reconciler 保证强一致 | [compute.md](compute.md) |
| artifacts | 模型 / 镜像 / 数据集元数据；两阶段写（initiate → 直推 → complete） | [artifacts.md](artifacts.md) |
| Prometheus (kube-prometheus-stack) | 在线服务指标 Tab 数据源 | [infra.md](../infra.md) |
| Kubernetes API | 仅工作区 PVC `get/list/create/delete`，受限 RBAC | — |

## 8. 运行时形态

| 维度 | 值 |
| --- | --- |
| 进程 | 单二进制 `axisml-platform`；子命令 `serve` / `migrate` / `bootstrap` |
| 副本 | 默认 `replicas=2`（无状态）；后端 + 前端镜像分别构建，本期共用同一 Service |
| 启动子命令 | `serve` 启动 HTTP API + 后台任务；`migrate` 执行 GORM 迁移（部署 init container）；`bootstrap` 一次性创建初始 `system-admin` 账号、内置角色与权限 |
| 暴露端口 | API `:8080`；Metrics `:8081`；Probes `:8082`（`/healthz` / `/readyz`） |
| RBAC scope | 受限 ClusterRole `axisml-platform-pvc`（仅 PVC，全部 compute namespace） |
| Helm values / 镜像 | 详见 [deployment.md](../deployment.md) |

## 9. 后续工作

### 9.1 横切

- 应用中心（Agent / Skills / MCP）后端契约与数据模型；
- 数据卷管理（PVC 抽象 / 数据集挂载路由 / StorageClass 视图）；
- OIDC 接入（仅切换 `--auth-mode=oidc`）；
- 审计日志 UI（按 `target` 前缀检索）与告警规则模板；
- 多集群 / 多区域。

### 9.2 制品中心

- 引用方反查端点（`Service.spec.modelRef` / `Job.spec.datasetRef` 反向索引）；
- 浏览器直传支持范围扩展（现仅 S3 Kind 小文件；OCI Kind 需在浏览器实现 chunked push）；
- 跨制品引用懒校验失效后的状态广播（被引方 `Deleted` → 引用方 `Ready` 但 resolve 返 410；前端展示降级告知）；
- 制品配额 / 签名 / SBOM 接入（等待 artifacts 服务 `size_bytes` 入表 + cosign / notation / trivy 集成）；
- 镜像 Layer 浏览端点（zot manifest API 解析 + per-layer 大小）。

### 9.3 租户

- 配额硬校验 / 分层配额（待上游 ElasticQuota 提供 `parent` 字段）；
- 「已归档租户」管理界面（restore 入口）；
- `initResources` 表单深度（Vault / Sealed Secrets 接入）；
- 租户克隆模板。

### 9.4 资源池 / 资源单元

- 按租户的池可见性（池 → 租户白名单）；
- 节点匹配预览 Tab；
- 池容量聚合；
- 池间调度借用策略；
- 资源单元成本元数据；
- 列表页 `resource_unit_count` 改 LRU 缓存。

### 9.5 工作区

- compute service `/events` / `/pods` / `/pods/{pod}/logs` / `/pods/{pod}/events` 端点接入；
- 闲时自动 stop；
- 孤儿 PVC 清理 UI；
- SSH 接入；
- 多容器 Workspace；
- GPU 预热 / 镜像预拉；
- DataVolume 集成。

### 9.6 计算任务

- `(kubeflow-trainer, *)` 多 role backend；
- `(custom, *)` JSON schema 编辑器；
- per-role ResourceUnit；
- register-model 完整链路；
- 任务模板 / 重新提交 UX；
- DAG 工作流；
- SSE / WebSocket 增量列表。

### 9.7 在线服务

- `(native, statefulset)` / `(kserve, inference)` / `(kserve, llminference)` / `(custom, *)` backend；
- `spec.route` 热更新与 `stripPathPrefix`；
- 流量切换与灰度（weighted route / canary / 自动回滚）；
- 自动扩缩容（HPA / KEDA）；
- 多 role 独立扩缩；
- LLM 专项指标；
- 告警与 SLO。

### 9.8 跨模块共享

- compute 批量 GetTenant RPC（减少多租户 namespace 解析的 RPC 数）。

## 10. 相关引用

- [overview.md](../overview.md) — 控制平面拓扑
- [auth.md](../auth.md) — 身份与鉴权契约
- [database.md](../database.md) — Platform PG schema (`§5`)
- [deployment.md](../deployment.md) — Helm / 部署
- [monitoring.md](../monitoring.md) — Metrics、告警与 service metrics PromQL 模板
- [infra.md](../infra.md) — Envoy Gateway / kube-prometheus-stack
- [wireframe.md](../wireframe.md) — 所有 UI 设计（列表 / 详情 / 表单）
- [apis/platform.yaml](../apis/platform.yaml) — REST 契约源
- [cluster-manager.md](cluster-manager.md) / [tenant-operator.md](tenant-operator.md) / [compute.md](compute.md) / [compute-operator.md](compute-operator.md) / [artifacts.md](artifacts.md)
