# AxisML Compute 详细设计

AxisML Compute 是平台的计算服务层，基于 Go 开发，仅接受来自 AxisML Platform 的内部 REST 调用，承载 **计算任务管理** 与 **系统管理（租户 / 资源池 / 资源单元 / 配额）** 两大职责。Compute 不直接创建 Namespace、Pod 等底层 K8s 资源——这些由 axisml-operator 或集群管理员负责；Compute 仅维护业务元数据，并通过 CRD 向 operator 声明意图。

| 模块 | PG 表 | 状态机 | 对应 K8s 资源 |
| --- | --- | --- | --- |
| Tenant ([§4](#4-tenant)) | `tenants` | `Active \| Suspended \| Deleting \| Deleted` | `Tenant` CR（tenant-controller 落地） |
| ResourcePool ([§5](#5-resourcepool)) | `resource_pools` | 无（纯 PG 元数据） | 无 CR；Job/Service 提交时注入 |
| ResourceUnit ([§6](#6-resourceunit)) | `resource_units` | 无（纯 PG 元数据） | 无 CR；Job/Service 提交时注入 |
| Quota ([§7](#7-quota)) | `quotas` | `Creating \| Active \| Deleting \| Deleted` | 借道 `Tenant.spec.quotas[]` → ElasticQuota |
| Job ([§8](#8-job)) | `jobs` | `Creating \| Pending \| Running \| Succeeded \| Failed \| Canceling \| Cancelled \| Deleting \| Deleted` | `MLJob` CR（mljob-controller） |
| Service ([§9](#9-service)) | `services` | `Creating \| Pending \| Ready \| Degraded \| Failed \| Deleting \| Deleted` | `MLService` CR（mlservice-controller） |

模块按生命周期分两类：**配置对象**（Tenant / ResourcePool / ResourceUnit / Quota）长期存在，CR 缺失/漂移由 Compute 按 PG 快照补偿重建；**工作负载对象**（Job / Service）的 CR 缺失是合法的生命周期终点（取消、清理、operator 级联），Compute 不重建。详见 [§3.7](#37-孤儿与补偿)。

**文档组织**：

- **Part I — 服务运行时**（§1 架构总览 + §2 服务运行时契约）：HTTP 服务进程的运维契约（端口、PG 客户端、K8s 客户端、副本与 leader election、RBAC、Helm values）。
- **Part II — 通用契约**（§3 跨模块通用契约）：6 个模块共享的写读模型、PG 编排约定、Outbox + Reconciler、双 hash spec 同步、Informer 状态回流、孤儿补偿、不变量；§4–§9 引用本节而不重复。
- **Part III — 模块详细设计**（§4 Tenant、§5 ResourcePool、§6 ResourceUnit、§7 Quota、§8 Job、§9 Service）：各模块的数据模型、状态机、生命周期与对外契约。
- **Part IV — 实施与验证**（§10 实现路径、§11 测试、§12 相关引用）：分阶段功能落地路线（MVP / 功能完善 / 未来规划）、测试层次、跨文档引用。

---

## Part I — 服务运行时

> 本部分描述 Compute 服务进程的运维契约：HTTP 服务形态、PostgreSQL 与 Kubernetes 客户端、副本与 leader election、Helm values。各模块共享这些契约。

## 1. 架构总览

```
                          ┌──────────────────────┐
                          │  AxisML Platform      │
                          │     (唯一调用方)        │
                          └──────────┬───────────┘
                                     │ REST / JSON
                                     │ 身份透传 header
                                     ▼
                          ┌──────────────────────┐
                          │   AxisML Compute      │
                          │       (Go)            │
                          └──┬──────────┬────────┘
           ┌─────────────────┘          └──────────────────┐
           │                                               │
           ▼                                               ▼
 ┌──────────────────┐                         ┌─────────────────────────────┐
 │   PostgreSQL      │                        │   Kubernetes API Server      │
 │ 元数据 / 用量缓存  │                        │   MLJob / MLService / Tenant │
 │                   │                        │   ElasticQuota / PodGroup   │
 └──────────────────┘                         └─────────────┬───────────────┘
                                                            │ list-watch
                                                            ▼
                                                   ┌──────────────────┐
                                                   │  Informer Loop   │
                                                   │  状态回流         │
                                                   └──────────────────┘
```

读写模型一句话：**API 只写 PG desired state → reconciler 异步下发 / patch CR → Informer 回流状态与 applied state**（详见 §3.4 / §3.6）。

**代码布局**

```
components/compute/
├── cmd/compute/              # 服务入口 main.go
├── api/
│   ├── openapi.yaml          # OpenAPI 3.0 契约源
│   └── types/                # oapi-codegen 生成的 request/response types + server stub
├── internal/
│   ├── server/               # HTTP router、middleware（身份解析、错误处理、访问日志、metrics）
│   ├── tenant/               # 租户管理
│   ├── resourcepool/         # 资源池管理
│   ├── resourceunit/         # 资源单元管理
│   ├── quota/                # 配额管理（PG `quotas` 表 CRUD；spec 下行与 used 回流借道 Tenant CR）
│   ├── job/                  # 任务管理（MLJob informer）
│   ├── service/              # 在线服务管理（MLService informer）
│   ├── k8sclient/            # controller-runtime client + informer factory（各模块共享 cache）
│   ├── db/                   # GORM 客户端 + golang-migrate 迁移
│   ├── spechash/             # desired/applied spec hash 计算
│   ├── auth/                 # 从 X-Axisml-User header 解析调用方身份
│   ├── metrics/              # Prometheus metrics
│   └── app/                  # 启动装配（serve / migrate / bootstrap）
└── pkg/                      # Compute 内部可复用工具（日志、错误、分页、字符串工具）
```

跨组件复用的公共库放在仓库根 `pkg/`。

## 2. 服务运行时契约

### 2.1 进程与端口

- 镜像：`ghcr.io/axisml/axisml-compute:<appVersion>`（由 `build/docker/compute.Dockerfile` 构建）
- 启动命令：`/compute serve`
- 端口：`8081/tcp`（REST，仅集群内 ClusterIP，不配置 `HTTPRoute` 对外）
- 探针：`GET /healthz`（进程存活）、`GET /readyz`（额外校验 PG 连通性；Informer 就绪状态不计入 readiness，避免 leader 切换期间非 leader 副本被摘流量）
- Metrics：`GET /metrics`（Prometheus 格式，详见 [§10.5 metrics 全集](#105-跨阶段验证策略)）

### 2.2 PG 客户端与迁移

- ORM：GORM；与 Artifacts 等服务共用 database `axisml`，通过表名前缀逻辑隔离
- 迁移：`golang-migrate` embedded 方式在服务启动时执行，依赖 `schema_migrations` 表的 PG advisory lock 避免多副本并发迁移；生产可选通过 Helm `Job` 隔离运行
- 连接：从 ConfigMap / Secret 读取 DSN；连接池参数（`maxOpen` / `maxIdle` / `connMaxLifetime`）通过 Helm values 暴露

### 2.3 Kubernetes 客户端

- controller-runtime `client.Client` + `SharedInformerFactory`，所有模块通过 `internal/k8sclient` 共享底层 cache
- 监听对象：`Tenant`（cluster-scoped）、`MLJob`（namespaced，全集群）、`MLService`（namespaced，全集群）；不直接读写 `ElasticQuota` CR（由 tenant-controller 独占，详见 [operator.md §3.7](operator.md#3-跨-controller-通用契约)）
- 不引入 controller-runtime 的 `Manager` / `Reconciler` 抽象——Compute 不是 K8s controller，reconciler goroutine 自行管理（详见 §3.4）

### 2.4 副本与 Leader Election

- **默认 `replicas=1`**（Standard 与 Lite 均同）
- **API 层无状态**：所有副本都服务 HTTP，仅写 PG，水平扩容无需协调
- **后台协程选主**：reconciler goroutine 与各模块 Informer 只在 leader 副本运行，通过 controller-runtime 的 K8s `Lease` 选主；单副本时退化为单成员瞬时 lease，无额外延迟
- **Leader 身份暴露**：`/metrics` 暴露 `axisml_compute_is_leader` gauge（0/1），便于告警与排障
- 完整的多副本压测、leader 切换语义验证作为后续演进项（[§10.4](#104-阶段三未来规划)）

### 2.5 RBAC（Compute → Kubernetes）

| 资源 | 动作 | 用途 |
| --- | --- | --- |
| `tenants.axisml.io` | `create / get / list / watch / update / patch / delete` | 下发 Tenant CR、消费 status |
| `mljobs.axisml.io` | `create / get / list / watch / update / patch / delete` | 下发 MLJob CR、消费 status、cancel patch |
| `mlservices.axisml.io` | `create / get / list / watch / update / patch / delete` | 下发 MLService CR、消费 status、scale patch |
| `pods` (target tenant ns) | `get / list` | Job 副本列表与日志透传 |
| `pods/log` (target tenant ns) | `get` | Job 日志 API 透传 kube-apiserver |
| `events` (target tenant ns) | `get / list` | Job 事件聚合端点 |
| `coordination.k8s.io/leases` (Compute 自身 ns) | `create / get / list / watch / update / patch / delete` | leader election Lease |

**不含**：`elasticquotas.scheduling.sigs.k8s.io`（由 tenant-controller 独占维护，Compute 仅通过 `Tenant.status.quotas[]` 间接消费 `used`）。`namespaces` / `secrets` / `configmaps` / `serviceaccounts` 等也不在 Compute RBAC 中（由 tenant-controller 派生）。

### 2.6 Helm values 接口

```yaml
compute:
  enabled: true
  image: { registry, repository, tag, pullPolicy }
  replicas: 1
  resources: { requests, limits }
  postgres:
    dsnSecret: { name, key }
    pool: { maxOpen, maxIdle, connMaxLifetimeSeconds }
  leaderElection:
    enabled: true
    id: axisml-compute.axisml.io
  bootstrap:
    enabled: true                # post-install Job 是否初始化默认数据
    defaultTenant: default
    defaultPool:   default
    defaultUnits:  [cpu-small, cpu-medium]
```

**Helm 模板清单**（`deploy/helm/axisml-system/templates/compute/`）：

| 文件 | 用途 |
| --- | --- |
| `configmap.yaml` | DB 连接、日志级别 |
| `deployment.yaml` | 使用 `axisml-compute` 镜像，加探针 |
| `service.yaml` | ClusterIP 8081 |
| `serviceaccount.yaml` | Compute 服务账号 |
| `rbac.yaml` | ClusterRole + ClusterRoleBinding：见 §2.5 |
| `servicemonitor.yaml` | `/metrics` 暴露，kube-prometheus-stack 自动发现 |
| `bootstrap-job.yaml` | post-install Job 初始化默认数据（详见 §10.2） |

---

## Part II — 通用契约

> 本部分集中 6 个模块**共享**的边界与协议：与 Platform 的请求契约、PG 编排约定、写读路径、状态回流、孤儿补偿、不变量。§4–§9 各模块章节引用本节而不重复。

## 3. 跨模块通用契约

### 3.1 与 Platform 的请求契约

Compute 仅接受 Platform 通过集群内 Service DNS 发起的 REST 调用，不直接对外部用户流量开放。

- **路径前缀**：`/api/v1/...`；不配置外部 `HTTPRoute`
- **身份头**：Platform 注入 `X-Axisml-User`（调用方用户唯一 ID）；Compute 仅做审计 / ownership 归属，不做角色级鉴权（鉴权由 Platform 统一完成）
- **租户归属**：通过 URL 路径 `/tenants/{tenant}/...` 表达；Compute 校验路径中的租户存在且 `status='Active'`，以及相关资源归属于该租户
- **OpenAPI 契约**：`components/compute/api/openapi.yaml` 是唯一契约源，使用 `oapi-codegen` 生成 Compute 侧 Go types + server stub（`api/types/`）与 Platform 侧 Go client SDK（通过 Makefile target 再次生成）
- **错误格式**：HTTP 标准状态码 + RFC 7807 problem+json（`type` / `title` / `status` / `detail` / `instance`）；业务错误码在 `detail` 中展开，避免 HTTP 状态码膨胀
- **写后异步语义**：所有会变更 CR spec 的 API（Tenant create/update/suspend/unsuspend、Quota CRUD、Service create/scale、Job create/cancel/delete）都只提交 PG desired state，**不承诺 CR 已同步完成**；同步失败由 reconciler 写入业务记录的 `message` 字段，调用方通过 Get/List 观察 `status`、`message` 与回流字段

### 3.2 PG 编排约定

- **通用字段**：所有表统一带 `id uuid`、`created_at`、`updated_at`、`deleted_at`（软删除）
- **UNIQUE 约束**：所有 schema 中标注的 `UNIQUE` 均实现为 PG partial unique index `WHERE deleted_at IS NULL`——软删行不占用唯一键，同名资源在原行被软删后可被再次创建。迁移示例：`CREATE UNIQUE INDEX ... ON tbl(col) WHERE deleted_at IS NULL`
- **`name` 字段 DNS-1123 硬校验**：所有承载业务标识并会映射到 K8s 对象名的 `name` 字段（Tenant / ResourcePool / Quota / Job / Service；ResourceUnit 在此之上还叠加 §6.3 的语义命名约定），API 层统一校验：字符集 `[a-z0-9-]`，首尾为字母或数字，长度 3–40，不允许连续 `--`（长度上限 40 是为了在 tenant-controller per-tenant 资源前缀拼接等最坏场景下仍满足 K8s DNS-1123 subdomain 253 字符限制）。需要更长或含大小写/空格的可读名请填 `display_name`
- **CR 稳定锚点**：所有 CR-backed 对象在 PG `id`（uuid）的同时打 label `axisml.io/{tenant,job,service}-id=<uuid>` 到 CR 上——`metadata.name` 因软删可能重用（partial UNIQUE），UUID 永久唯一；孤儿检测按 label 索引而非 name

### 3.3 权威划分

> **PG 为业务元数据与期望 spec（含配额 spec）的权威；Kubernetes / Koordinator 为运行状态与配额用量的权威。**

| 类别 | 权威方 | 流向 |
| --- | --- | --- |
| 配额定义 `spec`（min / max） | PG | API → PG `quotas` → Tenant CR `spec.quotas[]` → ElasticQuota（详见 §7.3） |
| 配额实际用量 `used` | Koordinator | `ElasticQuota.status.used` → `Tenant.status.quotas[].used` → PG `quotas.used`（详见 §7.3） |
| 业务元数据与期望 spec（名称、引用、spec 快照、desired hash） | PG | API → PG → reconciler → CR `spec` |
| 运行状态（phase、endpoint、副本就绪） | K8s | CR `status` → Informer → PG |

PG 的 `quotas.used` 只用于 UI 列表展示和 best-effort 预检，**不参与写入事务记账**。

### 3.4 写路径（Outbox + Reconciler）

采用 **Outbox 模式**：

1. **API 同步路径只写 PG**：业务校验 → PG 事务插入 / 更新业务记录（新建时 `status='Creating'`，取消时 `status='Canceling'`，删除时 `status='Deleting'` + `deleted_at=now()`；允许变更的 spec 写入 PG 快照并更新 `desired_spec_hash`）→ commit → 返回业务 ID。**API 不直接写 K8s**。
2. **Compute 内 reconciler worker 异步下发 CR**：每个模块（`internal/{tenant,job,service}` 持有 reconciler；`internal/quota` 通过 Tenant 间接下发）在 leader 副本起 goroutine，周期性扫描 PG 按下列**四个谓词**分派动作：

| 谓词 | 动作 | 适用模块 |
| --- | --- | --- |
| `status='Creating' AND deleted_at IS NULL` | `Create()` CR（附 label `axisml.io/<resource>-id=<uuid>`；409 `AlreadyExists` 视为成功，靠 `metadata.name` + label 双重去重幂等） | Tenant / Job / Service |
| `status='Canceling'` | `patch MLJob.spec.runPolicy.suspend=true`；后续推进与竞速处理详见 §8.4.2 | Job |
| `status='Deleting'` | `Delete()` CR；Informer DELETE 事件推进到 `Deleted`（配合设置 `deleted_at`） | Tenant / Job / Service |
| `desired_spec_hash != applied_spec_hash AND deleted_at IS NULL` | 按双 hash 机制 `Patch()`（详见 §3.5） | Tenant / Service / Quota（借道 Tenant） |

失败按指数退避重试，错误写入业务记录的 `message` 字段供 UI 展示。**PG 行不再满足任何谓词后，reconciler 不再做下发动作**——Job 自然结束、Service `Failed` 自愈、外部误删等情况由 Informer 回流推进，不进入 reconciler 工作集。

**共享状态机骨架**（各资源完整状态集与转换见 §4–§9）：

```
Creating ──(Informer ADD)──▶ 就绪态 ──(业务事件)──▶ 运行终态
                              │
      ┌───────────────────────┤
      │ cancel API (Job)      │ DELETE API (任一非 Canceling/Deleting/Deleted)
      ▼                       ▼
  Canceling                Deleting
      │                       │
      │ (Suspended condition)  │ (CR 确认清理 + deleted_at)
      ▼                       ▼
  Cancelled               Deleted
```

- 就绪态：Tenant/Quota 为 `Active`；Job 为 `Pending`（继而 `Running`）；Service 为 `Pending`（继而 `Ready`/`Degraded`/`Failed`）
- 运行终态：Job 有 `Succeeded`/`Failed`；Service 无运行终态（`Failed` 可自愈）；Tenant/Quota 无运行终态
- `Cancelled` 为 Job 独有（资源释放但 PG 行保留供查阅，用户可后续 DELETE 进入 `Deleting → Deleted`）；`Deleted` 为所有资源最终软删终态

### 3.5 desired/applied spec hash 双 hash 机制

允许变更的 CR-backed 对象通过双 hash 保持 PG-only 写路径：API 写 PG 时同步重算 `desired_spec_hash`；reconciler 在 `desired_spec_hash != applied_spec_hash AND deleted_at IS NULL` 时执行幂等 `Patch()`，成功后写 `applied_spec_hash=desired_spec_hash`；后续运行状态仍由 Informer 回流。Hash 由 `internal/spechash` 计算，输入是字段子集的归一化 JSON。

| 资源 | 是否使用 | 允许变更字段 |
| --- | --- | --- |
| Tenant | ✅ | `spec.displayName` / `annotations` / `namespace.labels` / `namespace.annotations` / `quotas` / `initResources` / `suspended`（其中 `spec.quotas[]` 由 `internal/tenant` 在 patch 时按 §7.3 渲染） |
| Service | ✅ | `spec.roles[0].replicas`（其他字段不可变） |
| Quota | 借道 Tenant | 通过标记同租户 `tenants.desired_spec_hash` 触发；自身不持有 hash 字段（详见 §7.3） |
| Job | ❌ | 不可变；cancel 由 `Canceling` 谓词 patch `spec.runPolicy.suspend=true`，不走 spec 同步 |
| ResourcePool / ResourceUnit | ❌ | 无对应 CR，无下行同步 |

### 3.6 状态回流（Informer）

三条独立 Informer，通过 `k8sclient` 的 `SharedInformerFactory` 共享底层 cache。Quota 不单设 Informer——`quotas.used` 与 quota 状态机都借道 Tenant CR `status.quotas[]` 回流（详见 §7.3）。

| Informer | 监听对象 | 维护方 | 主要用途 |
| --- | --- | --- | --- |
| MLJob Informer | `MLJob` CR | `internal/job/` | 推进 `Creating→Pending→Running→Succeeded/Failed`；`Canceling→Cancelled`（含与自然终态竞速处理，详见 §8.4.2）；`Deleting→Deleted`；DELETE 事件按 §3.7 工作负载策略处理；回写 `jobs.status` / `started_at` / `finished_at` |
| MLService Informer | `MLService` CR | `internal/service/` | 推进 `Creating→Pending→Ready/Degraded/Failed`（映射规则详见 §9.3）；`Deleting→Deleted`；DELETE 事件按 §3.7 工作负载策略处理；回写 `services.status` / `ready_replicas` / `endpoint` |
| Tenant Informer | `Tenant` CR | `internal/tenant/` | 推进 `Creating→Active`、`Active⇄Suspended`、`Deleting→Deleted`；按 `Tenant.status.quotas[].{ready, used}` 推进同租户 `quotas.status` 与 `quotas.used`；DELETE 事件按 §3.7 配置对象策略处理；spec 漂移按 §3.5 双 hash 同步 |

通用模式：启动时 `List` 做差异 upsert 与孤儿对账；`Watch` 事件入各自 work queue；单 worker 串行 reconcile；以 `resourceVersion` / `generation` 作乐观并发字段保证幂等。PG 更新短暂失败时事件在 work queue 中重试至成功。

### 3.7 孤儿与补偿

补偿策略按资源类型分流。启动 `List` 时按 `axisml.io/<resource>-id` label 索引 CR 与 PG 行做对账。

**配置对象（Tenant、Quota）**

| 场景 | 处理 |
| --- | --- |
| CR 漂移（声明字段被外部修改） | Informer 事件触发 Compute 按 PG 快照对齐——以 PG 为权威的字段（Tenant 的所有声明字段，含内联 `spec.quotas[]`）覆盖回 Tenant CR；CR 权威字段（`Tenant.status.quotas[].used`）不由 Compute 写回 |
| 正向孤儿（PG 有行但无 CR） | reconciler 幂等 `Create()` 重建 Tenant CR（连带 quotas 一并渲染）；Quota 没有独立 CR，反向孤儿仅出现在 ElasticQuota 层并由 tenant-controller 在 Tenant CR 漂移修复后自然纠正 |
| 反向孤儿（Tenant CR 存在但 PG 无对应行或已软删） | 默认删除 CR 并记录审计；高敏感场景可切为告警人工介入 |

**工作负载对象（Job、Service）**

Compute **不反向重建 CR**。Informer 观察到 CR DELETE 事件后按 PG 当前 `status` 分流（外部误删处理的单点定义）：

| PG 当前状态 | 处理 |
| --- | --- |
| `status='Canceling'` | CR 删除事件幂等忽略；`Cancelled` 只由 Suspended condition 推进 |
| `status='Deleting'` | 正常级联清理，推进 `Deleted` |
| Job 在 `Pending`/`Running`（外部误删） | 推 `Cancelled` + `finished_at` + `message='external delete'`，**不补偿重建** |
| Service 在 `Pending`/`Ready`/`Degraded`/`Failed`（外部误删） | 写 `Deleting` + `deleted_at=now()` + `message='external delete'`，交 reconciler 对不存在 CR 做幂等确认后推 `Deleted`，**不补偿重建** |
| 已终态（`Succeeded`/`Failed`/`Cancelled`/`Deleted`） | 忽略 |

**正向孤儿**（PG 有 `status='Creating'` 行但无 CR，且 `deleted_at IS NULL`）属于 Outbox 正常窗口，reconciler 下一轮幂等重试 `Create()`。若 PG 行已推进到非 `Creating` 状态或已软删，reconciler **不回补创建**。

**反向孤儿**（CR 存在但 PG 无行或已 `Deleted`）：默认删除 CR 并记录审计。

### 3.8 不变量

下列不变量对所有模块都生效，新增模块或扩展现有模块时必须满足：

- **API 不直接写 K8s**：业务校验通过后只写 PG；CR 下发由 reconciler 异步完成。
- **CR `metadata` / `spec` 单写**：Compute 是 CR `metadata` 与 `spec` 的唯一写入方；operator 只写 `status`（与 [operator.md §3.1](operator.md#3-跨-controller-通用契约) 对齐）。
- **status 单向消费**：Compute 只**读** CR `status`，并把统一字段（`phase` / `message` / `startedAt` / `finishedAt` / `quotas[].used` 等）回写 PG；不向 CR `status` 写任何字段。
- **配额用量不在 PG 记账**：`quotas.used` 是缓存，权威值在 `ElasticQuota.status.used`；不对 `quotas.used` 加锁或事务记账。
- **生命周期谓词收敛**：reconciler 工作集只来自 §3.4 的四个谓词；PG 行不再满足谓词后 reconciler 不做下发动作（避免把外部正常事件误判为孤儿）。
- **CR 稳定锚点**：所有 CR-backed 对象的 `axisml.io/<resource>-id=<uuid>` label 必填；孤儿检测按 label 索引而非 `metadata.name`。

---

## Part III — 模块详细设计

> 本部分逐个展开 6 个模块的数据模型、状态机、生命周期与对外契约。配置对象（Tenant / ResourcePool / ResourceUnit / Quota）在前，工作负载对象（Job / Service）在后。

## 4. Tenant

### 4.1 概述

Tenant 模块维护租户元数据 PG `tenants` 表，并把 `Tenant` CR 下发给 tenant-controller 落地 Namespace、ElasticQuota、initResources 等集群侧资源。Tenant 是 §3.7 中的**配置对象**——CR 缺失/漂移由 Compute 按 PG 快照补偿重建。

`Tenant` CR 是 cluster-scoped；Compute 通过 `metadata.labels[axisml.io/tenant-id]=<uuid>` 与 PG 行建立稳定锚点。Tenant CR 字段契约由 [operator.md §4.3](operator.md#4-tenant-controller) 定义；Compute 仅负责把 PG `tenants.spec` jsonb 渲染到 CR，并消费 `Tenant.status.phase` / `status.message` / `status.quotas[].{ready, used}`。

### 4.2 数据模型

```
tenants(
  id                  uuid PK,            -- 同时作为 Tenant CR label `axisml.io/tenant-id`
  name                text UNIQUE,        -- 作为 Tenant CR 的 metadata.name
  namespace           text,               -- Tenant CR `spec.namespace.name` 的查询镜像；非唯一索引（多 Tenant 可共享同一 Namespace）
  display_name        text,               -- Tenant CR `spec.displayName` 的查询镜像
  spec                jsonb,              -- 期望 Tenant.spec 快照：namespace / displayName / annotations / quotas / initResources / suspended
  desired_spec_hash   text,               -- 双 hash 机制详见 §3.5
  applied_spec_hash   text,               -- 双 hash 机制详见 §3.5
  status              text,               -- Creating / Active / Suspended / Deleting / Deleted
  message             text,               -- Tenant CR `status.message` 的回流；可空
  annotations         jsonb,              -- `spec.annotations` 的查询镜像；权威值在 `spec`
  created_at          timestamptz,
  updated_at          timestamptz,
  deleted_at          timestamptz         -- 软删除，进入 Deleting 时写入
)
```

**字段归属**：`namespace` / `display_name` / `annotations` 是列表查询与筛选用的镜像列；权威声明在 `spec` jsonb，至少覆盖 [operator.md §4.3.2](operator.md#4-tenant-controller) 定义的 `namespace`、`displayName`、`annotations`、`quotas`、`initResources`、`suspended`，其中 `spec.namespace.name` 创建后不可变。`spec.quotas[]` 由 reconciler 在 patch Tenant CR 前从 `quotas` 表实时渲染（详见 §7.3），PG `tenants.spec` 中的 `quotas` 字段是上次 patch 时的快照，主要用于幂等性比较。

**默认租户**：Helm `post-install` Job 幂等初始化 `default` 租户（详见 §10.2）。

### 4.3 状态机

```
Creating ──(Informer ADD)──▶ Active ⇄ Suspended ──(DELETE req)──▶ Deleting ──(CR 确认清理)──▶ Deleted
                               │                                     ▲
                               └──(DELETE req)───────────────────────┘
```

**`Suspended` 语义**：阻塞该租户下新 Job/Service 提交（API 在 Create 时校验 `tenant.status='Active'`）；已有任务保持运行；`Active ⇄ Suspended` 通过 `/suspend` / `/unsuspend` 子路径端点。

**operator 侧 `Tenant.status.phase=Failed` 在 Compute 上等价于 `Suspended` + `message`**：tenant-controller 在校验失败 / 关键资源创建失败时写 `status.phase=Failed`（[operator.md §4.4](operator.md#4-tenant-controller)），Informer 把 `Failed` 收敛为 `tenants.status='Suspended'` 并把 `tenant.status.message` 写入 `tenants.message`——租户提交链路同样受阻，靠 `message` 区分"配置出错"与"管理员暂停"。Compute 不引入独立 `Failed` 终态。

### 4.4 生命周期

| 操作 | API 端点 | PG | Kubernetes（reconciler 异步执行） |
| --- | --- | --- | --- |
| 创建 | `POST /api/v1/tenants` | insert `tenants`，写入 `spec`（含 `quotas[]` 渲染快照）与 `desired_spec_hash`，`applied_spec_hash=NULL`，`status='Creating'` | 创建 cluster-scoped `Tenant` CR（label `axisml.io/tenant-id=<uuid>`），tenant-controller 按 `spec.namespace.name` 落地 Namespace、按 `spec.quotas[]` 派生 ElasticQuota、按 `spec.initResources` 派生 Secret / ConfigMap / SA / RBAC（详见 [operator.md §4.6](operator.md#4-tenant-controller)）；reconciler 创建成功后写 `applied_spec_hash=desired_spec_hash`，Informer ADD 推 `Active` |
| 查询 | `GET /api/v1/tenants/{tenant}` / `GET /api/v1/tenants` | 读 PG | — |
| 更新 | `PATCH /api/v1/tenants/{tenant}` | 校验 `spec.namespace.name` 不变；更新 `tenants.spec` 及镜像列，重算 `desired_spec_hash` | reconciler 按 §3.4 spec 同步谓词 patch Tenant CR |
| 挂起 | `POST /api/v1/tenants/{tenant}/suspend` | 更新 `tenants.spec.suspended=true`，重算 `desired_spec_hash`；可同步把 `status='Suspended'` 用于立即阻断新提交 | reconciler patch Tenant CR `spec.suspended=true`；tenant-controller 仅写 `status.phase=Suspended`、不停机底层资源；Informer 回流确认 `Suspended` |
| 恢复 | `POST /api/v1/tenants/{tenant}/unsuspend` | 更新 `tenants.spec.suspended=false`，重算 `desired_spec_hash`；`status` 等待 Informer 回流 `Active` | reconciler patch Tenant CR `spec.suspended=false`；operator 重新推导 `Active` 后由 Informer 回流 |
| 删除 | `DELETE /api/v1/tenants/{tenant}` | `status='Deleting'`，写 `deleted_at`；同租户 quotas 也同步标记 `status='Deleting'` + `deleted_at` | reconciler `Delete()` Tenant CR，K8s GC 通过 ownerReference 级联清理 per-tenant 资源（ElasticQuota / Secret / ConfigMap / SA / Role / RoleBinding）；**Namespace 不删除**（详见 [operator.md §4.6.1](operator.md#4-tenant-controller)）；Informer 观察 CR 消失 → 同时推 Tenant 与同租户 quotas 至 `Deleted` |

### 4.5 与 tenant-controller 的契约

- **status 字段消费**：Compute 只消费 `Tenant.status.phase` / `status.message` / `status.quotas[].{ready, used}`；`namespaceReady` / `quotas[].observedGeneration` / `initResources[].ready` 等细分字段仅供 UI 观测，Compute 不写回 PG
- **`Failed` 收敛**：operator 写 `status.phase=Failed` 时，Compute 把 PG `status='Suspended'` 并把 `status.message` 写入 `tenants.message`
- **Namespace 永不级联删除**：与 [operator.md §4.6.1](operator.md#4-tenant-controller) 对齐；Tenant DELETE 后空 Namespace 保留供运维手动清理

### 4.6 后续工作

- **批量导入 / 导出**：用于灾难恢复或集群迁移
- **Tenant 配置模板**：预置常用配额 / initResources 组合，避免重复填写

## 5. ResourcePool

### 5.1 概述

ResourcePool 是 AxisML Compute 维护的纯 PG 元数据对象，用于把集群节点按物理或逻辑维度切分（GPU 池 / CPU 池、A100 池 / H100 池、训练池 / 推理池）。**ResourcePool 无对应 CR**，不走 Outbox 下发路径——它仅在生成 MLJob/MLService 时作为 `spec.scheduling.nodeSelector` / `tolerations` 的注入源。

管理员负责在集群侧给目标节点打标签 / 污染，Compute 不修改 Node 对象。

### 5.2 数据模型

```
resource_pools(
  id             uuid PK,
  name           text UNIQUE,
  description    text,
  node_selector  jsonb,                   -- {"axisml.io/pool": "gpu-a100"}
  tolerations    jsonb,                   -- K8s Toleration 数组
  metadata       jsonb,                   -- 自由扩展字段
  created_at     timestamptz,
  updated_at     timestamptz,
  deleted_at     timestamptz
)
```

**默认池**：Helm `post-install` Job 初始化 `default` 池，`node_selector` 留空即表示整集群可用（详见 §10.2）。

### 5.3 注入规则

提交 Job/Service 时由 `internal/resourceunit/inject.go` 与 ResourceUnit 的 `node_selector` 合并后写入 CR：

- `pool.node_selector` 的 key 全部保留（**Pool 优先**）
- `pool.tolerations` 直接作为 `spec.scheduling.tolerations`

详细合并规则见 §6.4。

### 5.4 API 端点

| 端点 | 方法 | 用途 |
| --- | --- | --- |
| `/api/v1/resource-pools` | `POST` / `GET` | Create / List |
| `/api/v1/resource-pools/{pool}` | `GET` / `PATCH` / `DELETE` | Get / Update / Delete |

仅管理员可写；Platform 通过角色判断后转发。

### 5.5 后续工作

- **池间调度策略**：当配额不足时是否允许跨池借用（默认禁止）
- **池容量预估**：聚合所属节点的可分配资源，用于配额规划

## 6. ResourceUnit

### 6.1 概述

ResourceUnit 是 ResourcePool 内预先定义的资源规格模板，是 AxisML Compute 维护的纯 PG 元数据对象。例如 `a100-1x-large` 可表示 1×A100 + 8 vCPU + 32 GiB。用户创建任务或服务时选择一个 ResourceUnit，Compute 据此注入 `requests` / `limits` 与节点匹配条件，避免在 API 层手工填写 CPU/GPU/内存明细。

**ResourceUnit 无对应 CR**，不走 Outbox 下发路径。

### 6.2 数据模型

```
resource_units(
  id             uuid PK,
  pool_id        uuid FK resource_pools(id),
  name           text,
  description    text,
  requests       jsonb,                   -- {"cpu":"8","memory":"64Gi","nvidia.com/gpu":"1"}
  limits         jsonb,
  node_selector  jsonb,                   -- 通用节点标签匹配（见 §6.4）
  created_at     timestamptz,
  updated_at     timestamptz,
  deleted_at     timestamptz,
  UNIQUE(pool_id, name)
)
```

**字段说明**

- `requests` / `limits`：标准 K8s 资源，包括 `cpu`、`memory`、`nvidia.com/gpu`、其他 extended resources；数量一律在此表达
- `node_selector`：通用节点标签匹配，覆盖 GPU 之外的任意硬件维度

**常见 `node_selector` 用法**：

| 场景 | 示例 |
| --- | --- |
| GPU 型号 | `{"nvidia.com/gpu.product": "A100-SXM4-80GB"}`（GPU Operator 的 Node Feature Discovery 自动打标） |
| TPU | `{"cloud.google.com/gke-accelerator": "tpu-v4"}` |
| 自研加速卡 | `{"axisml.io/accelerator": "npu-x"}` |
| CPU instance type | `{"node.kubernetes.io/instance-type": "c5.4xlarge"}` |

### 6.3 命名约定

格式 `<accelerator>[-<count>x]-<tier>[-<variant>]`：

- `<accelerator>`：加速卡或 CPU 标识（小写 kebab，如 `a100` / `h100` / `tpu-v4` / `ascend-910b` / `cpu`）
- `<count>x`：加速卡数量（如 `1x` / `2x` / `4x` / `8x`），CPU-only 省略
- `<tier>`：规格档 `small` / `medium` / `large` / `xlarge`
- `<variant>`：可选后缀，如 `himem` / `ssd` / `spot` / `ib`

示例：`cpu-small`、`a100-1x-large`、`h100-8x-xlarge-ib`、`tpu-v4-4x-large`。字符集校验同 §3.2。

### 6.4 注入规则

提交 Job/Service 时由 `internal/resourceunit/inject.go` 完成：

- `requests` / `limits` 注入到 CR `spec.roles[*].template.resources`
- `spec.scheduling.nodeSelector`：**Pool 优先合并**——`pool.node_selector` 的 key 全部保留，`resource_unit.node_selector` 只贡献 Pool 未声明的 key。等价 Go 伪码：

  ```go
  ns := maps.Clone(pool.NodeSelector)
  for k, v := range unit.NodeSelector {
      if _, exists := ns[k]; !exists {
          ns[k] = v
      }
  }
  ```
- `pool.tolerations` 直接作为 `spec.scheduling.tolerations`

Pool 的硬件维度（如 `axisml.io/pool=gpu-a100`）一经设置即为池的"身份"，ResourceUnit 在同 key 上重新声明会被静默忽略；unit 只能补充 Pool 未涉及的维度（如 GPU 型号 / 内存档次 / 实例类型）。这样 Pool 更新不会把下辖 unit 推入"僵尸"状态，也省掉 API 层的级联冲突校验。

### 6.5 API 端点

| 端点 | 方法 | 用途 |
| --- | --- | --- |
| `/api/v1/resource-pools/{pool}/resource-units` | `POST` / `GET` | Create / List |
| `/api/v1/resource-pools/{pool}/resource-units/{unit}` | `GET` / `PATCH` / `DELETE` | Get / Update / Delete |

仅管理员可写。

### 6.6 后续工作

- **混合资源单元**：单 unit 内表达"CPU 1 + GPU 0.5（MIG）"等复杂规格
- **价格元数据**：`unit.metadata.cost_per_hour` 用于 §10.4 成本核算演进

## 7. Quota

### 7.1 概述

Quota 是租户在某个 ResourcePool 下的配额承载体，对齐上游 sigs.k8s.io scheduler-plugins 的 `ElasticQuota`（namespace-scoped）。Quota 在概念上是 Tenant 的子资源——CR 端不单独下发 `ElasticQuota`，而是把 PG `quotas` 表渲染进 `Tenant.spec.quotas[]`，由 tenant-controller 1:1 派生 ElasticQuota（命名 `axisml-<tenant>-<pool>-<quota>`，详见 [operator.md §4.6.2](operator.md#4-tenant-controller)）。

每个 `(tenant, pool)` 默认建议存在一条名为 `default` 的配额（Helm `post-install` 仅为 bootstrap `(default, default)` 创建一条；其他 `(tenant, pool)` 不自动派生）。用户可另行创建 `training` / `inference` / `nlp` 等配额用于业务线维度的拆分。当前为扁平结构（无父子层级），分层配额（Koord-Queue tree）作为 §10.4 演进项。

### 7.2 数据模型

```
quotas(
  id                  uuid PK,
  tenant_id           uuid FK tenants(id),
  pool_id             uuid FK resource_pools(id),
  name                text,                      -- "default" / "training" / ...
  spec                jsonb,                     -- 配额 spec（min / max），与上游 ElasticQuota 字段一一对应
  status              text,                      -- Creating / Active / Deleting / Deleted（由 Tenant Informer 推进）
  used                jsonb,                     -- 缓存自 Tenant.status.quotas[i].used；只读不记账
  created_at          timestamptz,
  updated_at          timestamptz,
  deleted_at          timestamptz,
  UNIQUE(tenant_id, pool_id, name)
)
```

`spec` 字段结构（与上游 `scheduling.sigs.k8s.io/v1alpha1` ElasticQuota 完全一致）：

```json
{
  "min": {"cpu": "20",  "memory": "80Gi",  "nvidia.com/gpu": "2"},
  "max": {"cpu": "100", "memory": "400Gi", "nvidia.com/gpu": "8"}
}
```

**配额语义**（对齐 sigs.k8s.io scheduler-plugins ElasticQuota）：

| 维度 | 含义 | 超限后果 |
| --- | --- | --- |
| `max` | **硬上限**：配额可占用资源的最大值 | 超过 → koord-scheduler 拒绝调度，Pod 停在 `Pending` |
| `min` | **保留份额**：必须满足的底线（不可被其他配额抢占的下界） | 已分配资源若超过 `min`，超出部分（属于"借用"）可被其他配额未满足 `min` 的任务**抢占回收（reclaim）** |

**校验约束**（每个资源 key 分别校验）：`min ≤ max`，且均 ≥ 0。

**默认值策略**：API 仅强制要求 `max`；`min` 未填时默认为 0（等同"无保留份额"，所有配额平等争抢直到 `max` 封顶）。借用容量在多配额间按 koord-scheduler 默认平权分配；不引入 Koordinator 私有的 `shared-weight` annotation，保持 PG schema、API 与上游 CR 字段一一对应。

### 7.3 借道 Tenant CR 的双向数据链路

> **核心机制**：Quota 不持有独立 CR，也不持有独立的 `desired_spec_hash` / `applied_spec_hash`；下行 / 上行 / 状态机都借道 Tenant CR。

**下行（spec 同步）**

```
API 写 PG quotas
   │ (同事务标记同租户 tenants.desired_spec_hash 需重算)
   ▼
internal/tenant reconciler
   │ (patch Tenant CR 时按 SELECT * FROM quotas WHERE tenant_id=$1 AND deleted_at IS NULL 渲染)
   ▼
Tenant.spec.quotas[]
   │
   ▼
tenant-controller 派生 ElasticQuota.spec.{min, max}
   │ (patch 成功后 Compute 写 tenants.applied_spec_hash)
   ▼
koord-scheduler ElasticQuota plugin
```

**上行（用量回流）**

```
ElasticQuota.status.used
   │ (tenant-controller watch 全集群 ElasticQuota，按 ownerReference 反查 Tenant)
   ▼
Tenant.status.quotas[].used
   │
   ▼
Compute Tenant Informer
   │
   ▼
PG quotas.used (缓存)
```

**状态机**

```
Creating ──(Tenant.status.quotas[i].ready=true)──▶ Active ──(DELETE req)──▶ Deleting ──(Tenant.status 不再含该 quota / Tenant CR 删除)──▶ Deleted
```

由 `internal/tenant` 的 Tenant Informer 驱动：每次 Tenant CR `status` 更新时按 `Tenant.status.quotas[].{ready, used}` 与 PG 行做 join，推进对应 quota 的 `status` 与 `used`。Quota 不单设 Informer。

### 7.4 配额预检（best-effort）

提交 Job/Service 时，Compute 在 API 同步路径上做轻量预检：

1. 读 `spec.max` 与缓存 `used`
2. 若 `used + request > max` 则早期拒绝并返回错误
3. 预检通过 → 创建 MLJob / MLService CR

Koordinator 层硬约束覆盖**所有** AxisML workload Pod（[infra.md §8.3](infra.md) Quota 全覆盖契约 + [operator.md §3.4](operator.md#3-跨-controller-通用契约) Pod 注入约定）：MLJob 与 MLService 各 backend 派生的 Pod 都设置 `schedulerName: koord-scheduler` 并通过 label `quota.scheduling.koordinator.sh/name` 关联到 ElasticQuota，因此都进入 `ElasticQuota.status.used`。`min` 影响调度序关系与抢占行为，不做 Compute 层预检。PG 不对 `quotas.used` 加锁或事务记账。

**多 namespace 部署形态**：ElasticQuota CR 与 workload Pod 都落在租户 namespace 是 koord-scheduler 官方支持的形态——Pod 通过 label 按名字跨 namespace 绑定 quota，`status.used` 与 `min/max` 硬约束照常生效。详见 [infra.md §8](infra.md)。

### 7.5 API 端点

| 端点 | 方法 | 用途 |
| --- | --- | --- |
| `/api/v1/tenants/{tenant}/quotas` | `POST` / `GET` | Create / List |
| `/api/v1/tenants/{tenant}/quotas/{quota}` | `GET` / `PATCH` / `DELETE` | Get / Update / Delete |

`POST` / `PATCH` / `DELETE` 都只写 PG `quotas` + 标记同租户 `tenants.desired_spec_hash`；CR 端的 ElasticQuota 由 `internal/tenant` reconciler 异步派生（详见 §7.3）。

### 7.6 后续工作

- **分层配额（Koord-Queue tree）**：`quotas` 表新增 `parent_id`，利用 Koordinator ElasticQuota 的 `quota.scheduling.koordinator.sh/parent` annotation 实现父子配额与抢占
- **细粒度配额**：GPU 时长、存储容量、网络带宽（详见 §10.4）

## 8. Job

### 8.1 概述

Job 模块维护 PG `jobs` 表 + 下发 `MLJob` CR + 通过 MLJob Informer 回流执行状态。Job 是一次性 workload，对应 §3.4 中的**工作负载对象**——CR 缺失视为合法生命周期终点，不补偿重建。

`MLJob` CR 字段契约由 [operator.md §5](operator.md#5-mljob-controller) 定义；Compute 仅负责把业务语义装进 CR 并消费统一的 `status.phase` / `message` / `startedAt` / `finishedAt` / `conditions[type=Suspended]`。

### 8.2 数据模型

```
jobs(
  id                   uuid PK,
  tenant_id            uuid FK tenants(id),
  pool_id              uuid FK resource_pools(id),
  quota_id             uuid FK quotas(id),
  resource_unit_id     uuid FK resource_units(id),
  name                 text,                     -- MLJob CR metadata.name（在租户 namespace 内唯一）
  display_name         text,                     -- 用户可见名
  description          text,                     -- 用户备注
  owner_user           text,                     -- 来自 X-Axisml-User
  spec                 jsonb,                    -- 提交时的 MLJob.spec 完整快照（不可变）
  requested_resources  jsonb,                    -- 配额记账快照
  status               text,                     -- Creating / Pending / Running / Succeeded / Failed / Canceling / Cancelled / Deleting / Deleted
  message              text,                     -- CR 下发错误或状态附加信息
  started_at           timestamptz,              -- Pod 首次运行（Informer 回流）
  finished_at          timestamptz,              -- 运行终态时间（Informer 回流）
  created_at           timestamptz,
  updated_at           timestamptz,
  deleted_at           timestamptz,              -- 进入 Deleting 时写入
  UNIQUE(tenant_id, name)                        -- 租户内 name 唯一；partial on deleted_at IS NULL
)
```

**字段归属**

- `spec` 是提交时 MLJob.spec 的完整快照，包含 `roles[]` / `backend.{name,engine,config}` / `scheduling` / `runPolicy` 等业务字段（结构详见 [operator.md §5.2](operator.md#5-mljob-controller)），**不可变**；Informer 回流只写 `status` 相关列
- `requested_resources` 冗余存提交时的资源申请，解耦后续 ResourceUnit 修改对已提交任务记账的影响
- `name` 是 Compute 与 K8s 的命名锚点（对应 MLJob CR `metadata.name`），UUID `id` 通过 label `axisml.io/job-id=<id>` 同步打到 CR 上，作为孤儿检测与跨重命名追踪的稳定锚点

**`spec.backend` 默认值注入**：用户未指定 `spec.backend` 时，Compute 写 CR 时显式补 `{name: "native", engine: "job"}`；`backend.config` 默认空对象 `{}`。`backend.{name, engine}` 在 PG `jobs.spec` jsonb 中持久化，**创建后不可变**（Update API 拒绝修改这两个字段）。

### 8.3 状态机

```
Creating ──(Informer ADD)──▶ Pending ──(CR phase=Running)──▶ Running ──(CR phase=Succeeded)──▶ Succeeded
                                │                              │    ──(CR phase=Failed)────▶ Failed
                                │                              │
                                │  cancel API                  │  cancel API
                                └──────────────────────────────┴─▶ Canceling ──(Suspended condition)──▶ Cancelled

任一非 Canceling/Deleting/Deleted 状态 ──(DELETE req)──▶ Deleting ──(CR 确认清理 + deleted_at)──▶ Deleted
```

- `Creating` 状态不接受 cancel（CR 未下发，API 拒绝并要求调用 DELETE 直接进入 `Deleting`）
- `Cancelled` PG 行保留（`deleted_at IS NULL`），用户可再次 DELETE → `Deleting` → `Deleted`
- `Succeeded` / `Failed` / `Cancelled` 三者为运行终态；`Deleted` 为软删终态

### 8.4 业务语义

#### 8.4.1 提交校验

`POST /api/v1/tenants/{tenant}/jobs` 在 §3.4 通用 Outbox 流程之外附加：

1. 从 `X-Axisml-User` 读取调用方身份（审计与 ownership 归属）
2. 业务校验：路径中的租户存在且激活；配额归属该租户；ResourceUnit 所属 pool 与配额所属 pool 一致
3. 配额预检（best-effort，详见 §7.4）

#### 8.4.2 取消语义

`POST /api/v1/tenants/{tenant}/jobs/{job}/cancel`

| 场景 | PG 侧 | CR 侧 |
| --- | --- | --- |
| 运行态（`Pending` / `Running`） cancel | `status='Canceling'`，写 `message='user cancelled'` | reconciler `patch MLJob.spec.runPolicy.suspend=true` → operator Handler 完成 suspend/Cleanup 后写 `status.conditions[type=Suspended,status=True,reason=CancelRequested]`（[operator.md §5.3](operator.md#5-mljob-controller)）→ Informer 推 `Cancelled`、写 `finished_at` 并入队 `Delete()` CR 做资源回收 |
| `Canceling` 期间 Job 自然结束（**竞速**） | 维持 `Canceling` 直到 Informer 观察到 CR 终态；按 operator "终态优先"推到 `Succeeded`/`Failed` 并写 `finished_at`（operator 不写 Suspended condition） | reconciler 已发出 cancel patch；不再补偿 |
| `Creating` 状态 cancel | API 拒绝（要求改用 DELETE） | — |
| 已终态或已在 `Canceling`/`Deleting` cancel | API 返回无效操作 | — |

#### 8.4.3 删除语义

`DELETE /api/v1/tenants/{tenant}/jobs/{job}`

| 场景 | PG 侧 | CR 侧 |
| --- | --- | --- |
| 任一非 `Deleting`/`Deleted` 状态 DELETE | `status='Deleting'`，写 `deleted_at` | reconciler `Delete()` CR；Informer DELETE → `Deleted` |
| `Creating` 状态 DELETE（CR 未下发） | 同上；reconciler 确认 CR 不存在 → 直接推 `Deleted` | — |
| `Cancelled` 状态 DELETE（CR 已被 cancel 路径回收） | 同上；reconciler `Delete()` CR 收到 404 → 幂等确认后直接推 `Deleted`，不依赖 Informer DELETE 事件 | — |
| `Deleting`/`Deleted` 再次 DELETE | 幂等，忽略 | — |

外部误删（PG 在 `Pending`/`Running` 收到 CR DELETE 事件）的处理见 §3.7。Job 自然结束由 Informer 回流 `Succeeded`/`Failed`，operator/GC 按策略清理 Pod，Compute 不主动删除 CR。

#### 8.4.4 任务日志透传

`GET /api/v1/tenants/{tenant}/jobs/{job}/logs` 只做路径级鉴权与 Pod 定位，实际日志 IO 透传到 kube-apiserver 的 Pod Log API。

- **寻址**：通过 `replica`（基于 `axisml.io/replica-index` label，仅副本身份天然稳定的 backend 支持，如 Indexed Job、StatefulSet）或 `pod`（直接 Pod 名，可通过 `/jobs/{job}/replicas` 端点列出）定位目标 Pod；多容器场景可指定 `container`。两者均缺失时返 400
- **流式**：`follow=false`（默认）直接透传 `text/plain`；`follow=true` 进入 SSE 流式模式（`Content-Type: text/event-stream`），客户端断开时主动关闭 upstream watch
- **鉴权**：只校验路径中 `tenant` 归属 `job`（§3.1 原则）
- **GC 后**：Pod 已被回收（Job 进入终态 + GC）时返 410 Gone，建议调用方改用外部日志系统

具体查询参数（`container` / `tail_lines` / `since_time` / `previous` 等）以 `components/compute/api/openapi.yaml` 为准。Compute 不做多副本日志聚合、不做日志持久化 / 搜索。

#### 8.4.5 副本与事件端点

| 端点 | 用途 |
| --- | --- |
| `/api/v1/tenants/{tenant}/jobs/{job}/replicas` | List Pod 副本（编号 / pod 名 / phase / started_at），按 `axisml.io/job-id` label 查询 |
| `/api/v1/tenants/{tenant}/jobs/{job}/events` | 聚合 MLJob / Pod / PodGroup 的 K8s Event |

### 8.5 与 mljob-controller 的契约

- **Compute phase 映射**（与 [operator.md §5.3](operator.md#5-mljob-controller) 对齐）：`Pending → Pending`、`Running → Running`、`Succeeded → Succeeded`（终态）、`Failed → Failed`（终态）
- **Cancel 推进信号**：Compute Informer 在 PG `status='Canceling'` 时把 `conditions[type=Suspended,status=True,reason=CancelRequested]` 当作推进信号 → 写 `Cancelled` → 入队 `Delete()` CR 做资源回收
- **终态优先**：cancel patch 与 Job 自然完成竞速时，operator 优先保留终态 `phase` 与 `finishedAt`；Compute 此时直接把 PG 推到对应运行终态，不再等待 Suspended condition

**级联效果**：CR 删除 → mljob-controller 清理 Pod → Pod 终止后 ElasticQuota plugin 释放对应资源 → tenant-controller 把 `ElasticQuota.status.used` 聚合到 `Tenant.status.quotas[].used` → Compute Tenant Informer 刷新 `quotas.used`。

### 8.6 后续工作

- **Job spec 部分可变**：当前 `spec` 完全不可变；考虑允许 `display_name` / `description` / `runPolicy.activeDeadlineSeconds` 等元数据字段更新
- **Job 模板 / 工作流编排**：组合多个 Job 形成 DAG 级运行计划

## 9. Service

### 9.1 概述

Service 模块维护 PG `services` 表 + 下发 `MLService` CR + 通过 MLService Informer 回流副本就绪与 endpoint。Service 是常驻 workload，配额占用不随运行状态释放、仅在 `Deleted` 时释放；通过 `/scale` 端点实现弹性扩缩。

`MLService` CR 字段契约由 [operator.md §6](operator.md#6-mlservice-controller) 定义。

### 9.2 数据模型

```
services(
  id                   uuid PK,
  tenant_id            uuid FK tenants(id),
  pool_id              uuid FK resource_pools(id),
  quota_id             uuid FK quotas(id),
  resource_unit_id     uuid FK resource_units(id),
  name                 text,                     -- MLService CR metadata.name（在租户 namespace 内唯一）
  display_name         text,
  description          text,
  owner_user           text,
  spec                 jsonb,                    -- 当前 MLService.spec 快照
  desired_spec_hash    text,                     -- 双 hash 机制详见 §3.5
  applied_spec_hash    text,                     -- 双 hash 机制详见 §3.5
  requested_resources  jsonb,                    -- 单副本资源申请快照（配额记账用）
  replicas             int,                      -- 冗余出来供配额记账（单 role 约定下 = spec.roles[0].replicas）
  ready_replicas       int,                      -- Informer 回流
  endpoint             text,                     -- 服务地址（Informer 回流）
  status               text,                     -- Creating / Pending / Ready / Degraded / Failed / Deleting / Deleted
  message              text,
  created_at           timestamptz,
  updated_at           timestamptz,
  deleted_at           timestamptz,
  UNIQUE(tenant_id, name)
)
```

**字段归属**

- `spec` 与 jobs 不同：扩缩容 API 更新 `replicas` 同时回写 `spec.roles[0].replicas`（单 role 约定）并重算 `desired_spec_hash`，其他字段依然不可变
- 总配额占用 = `replicas × requested_resources`。Koordinator ElasticQuota plugin 通过 Pod label 自动核算实际用量；Compute 在 API 层做 best-effort 预检
- `endpoint`：`(native, *)` / `(custom, *)` 可为内部 Service DNS 或 AxisML Gateway URL；`(kserve, *)` 为 KServe `status.url`

**`spec.backend` 默认值注入**：用户未指定 `spec.backend` 时，Compute 写 CR 时显式补 `{name: "native", engine: "deployment"}`；`backend.config` 默认空对象 `{}`。`backend.{name, engine}` 创建后不可变。

### 9.3 状态机

```
Creating ──(Informer ADD)──▶ Pending ──(ready=desired, desired>0)──▶ Ready ⇄ Degraded ──▶ Failed
                                                                      ▲                     │
                                                                      └─────── 自愈 ────────┘

任一非 Deleting/Deleted 状态 ──(DELETE req)──▶ Deleting ──(CR 确认清理 + deleted_at)──▶ Deleted
```

**status 主流映射**（由 Informer 从 MLService CR `status.phase` / `status.readyReplicas` / `spec.roles[0].replicas` 推导，单 role 约定下 `services.replicas` 即 `spec.roles[0].replicas`，多 role 独立扩缩详见 [operator.md §6.7](operator.md#6-mlservice-controller)）：

| 条件 | status |
| --- | --- |
| `ready_replicas == desired_replicas && desired_replicas > 0` | `Ready` |
| `0 < ready_replicas < desired_replicas` | `Degraded` |
| `ready_replicas == 0 && desired_replicas > 0` 且 MLService `status.phase=Pending` | `Pending`（创建 / 滚动更新仍在推进中） |
| `ready_replicas == 0 && desired_replicas > 0` 且 MLService `status.phase=Failed` | `Failed`（后端超过进度期限或明确失败；非终态，可自愈） |

> **附注 corner case**：`desired_replicas == 0` 时（扩缩至 0，视为待调度 / 停用）`status='Pending'`。

`Ready` / `Degraded` / `Failed` 均为**非终态**——operator 自愈（重建失败 Pod、健康检查恢复）后 Informer 回流可让 `Failed → Degraded → Ready` 自然恢复；只有 `Deleted` 是 Service 的最终终态。

### 9.4 业务语义

#### 9.4.1 提交校验

同 §8.4.1（租户激活、配额归属、unit/pool 一致、配额预检）。

#### 9.4.2 扩缩容

`POST /api/v1/tenants/{tenant}/services/{service}/scale`

走 PG-only Outbox：API 层只更新 `services.replicas` + `services.spec.roles[0].replicas` 并重算 `desired_spec_hash`，返回"desired replicas 已提交"；reconciler 后续按 §3.5 双 hash 机制 patch MLService CR path `spec/roles/0/replicas`。实际 `ready_replicas` / `status` 仍由 Informer 回流；配额按 `replicas × requested_resources` 线性预检。

#### 9.4.3 删除语义

`DELETE /api/v1/tenants/{tenant}/services/{service}`

| 场景 | PG 侧 | CR 侧 |
| --- | --- | --- |
| 任一非 `Deleting`/`Deleted` 状态 DELETE | `status='Deleting'`，写 `deleted_at` | reconciler `Delete()` CR；Informer DELETE → `Deleted`；配额自然释放 |
| `Creating` 状态 DELETE（CR 未下发） | 同上；reconciler 确认 CR 不存在 → 直接推 `Deleted` | — |

外部误删（运行态收到 CR DELETE 事件）的处理见 §3.7。Service 没有 cancel 语义——常驻服务"下线"即"删除"。

### 9.5 与 mlservice-controller 的契约

- **Compute phase 映射**：详见 §9.3 表格
- **`/scale` 透传**：mlservice-controller 必须把 `roles[*].replicas` 透传为后端原生扩缩（`Deployment.spec.replicas` 等），详见 [operator.md §3.6](operator.md#3-跨-controller-通用契约) 控制信号义务
- **路由派生**：`(native, *)` 的外部入口由 mlservice-controller 派生 `HTTPRoute`（详见 [operator.md §6.5](operator.md#6-mlservice-controller)），Compute 不参与路由配置；`(kserve, *)` 使用 KServe 自带 route

### 9.6 后续工作

- **多 role 独立扩缩**：当前 `services.replicas` 字段在单 role 约定下定义为 `spec.roles[0].replicas`；多 role 场景需要 role 显式寻址的 `/scale` 路径
- **autoscaling**：基于 `quotas.used` / `request_rate` 的弹性扩缩
- **`spec.route` 热更新**：轮换 API key / 调限流不重建 Service / Deployment

---

## Part IV — 实施与验证

> 本部分给出 Compute 的功能落地路线、测试策略与跨文档引用。新贡献者读完前三部分后从这里看"先做什么、再做什么、怎么验证"。

## 10. 实现路径

按功能优先级把交付内容映射到三个阶段。MVP 划定"能跑通端到端最小可发布范围"，功能完善覆盖主流场景与生产硬化，未来规划承接需求未明朗或上游依赖未稳定的方向。每条目标都附完成信号，便于阶段闭合验证。

### 10.1 阶段总览

```
┌──────────────────────────────────────────────────────────────┐
│ MVP（最小可发布）                                             │
│   单副本 / 6 模块 CRUD / 默认 backend / Outbox 仅 Creating+   │
│   Deleting / Informer 状态回流 / bootstrap default 数据       │
│   ↓                                                           │
│ 功能完善（生产硬化）                                          │
│   双 hash spec 同步 / Cancel / Suspend / 外部误删处理 /       │
│   配额预检 / 多 backend / 日志 + 副本 + 事件端点 / metrics 全集│
│   ↓                                                           │
│ 未来规划（需求 / 上游驱动）                                    │
│   多副本 HA / 分层配额 / 数据卷管理 / 多集群 / 审计 / 计费     │
└──────────────────────────────────────────────────────────────┘
```

### 10.2 阶段一：MVP（最小可发布）

支撑端到端最小演示路径："建租户 → 配资源 → 提任务/服务 → 看结果"。

| 模块 | 范围 | 完成信号 |
| --- | --- | --- |
| 服务运行时 | 单副本 / `replicas=1`、Helm 模板（configmap / deployment / service / sa / rbac / servicemonitor / bootstrap-job）、`/healthz` + `/readyz` + `/metrics`、PG 启动迁移 | `helm install` 后 Pod Ready；`/healthz` 200；PG `schema_migrations` 表存在 |
| Tenant | Create / Get / List / Delete API；Tenant CR 下发 + Informer 推 `Active` / `Deleting → Deleted` | L1 integration 覆盖：API Create → Tenant CR 出现 + label `axisml.io/tenant-id` 一致；Tenant CR 删除 → PG `status='Deleted'` |
| ResourcePool | CRUD API（无 CR） | API 测试覆盖；新建 Pool 后 ResourceUnit 可挂靠 |
| ResourceUnit | CRUD API（无 CR）；提交 Job/Service 时注入 `requests` / `limits` / `nodeSelector` / `tolerations`（Pool 优先合并） | API 测试覆盖；L1 integration 覆盖：提交 Job 后 MLJob CR `spec.scheduling` 与 `spec.roles[*].template.resources` 字段值正确 |
| Quota | Create / Get / List / Delete API；通过 `tenants.desired_spec_hash` 驱动 Tenant CR 渲染；Informer 按 `Tenant.status.quotas[]` 推进 `Active` / `Deleted` 与 `used` 缓存 | L1 integration 覆盖：API Create Quota → Tenant CR `spec.quotas[]` 出现 → ElasticQuota 派生（fake tenant-controller）；`Tenant.status.quotas[i].ready=true` → PG quota `status='Active'` |
| Job | Create / Get / List / Delete API（不含 cancel / logs / events / replicas / spec update）；MLJob CR 下发；Informer 推 `Pending → Running → Succeeded/Failed`；DELETE → `Deleting → Deleted` | L1 integration 覆盖：API Create → MLJob CR 出现 + 状态推进；DELETE → CR 删除 + PG `Deleted`；外部误删 Pending/Running CR 时 PG 推 `Cancelled`（提前对齐 §3.7） |
| Service | Create / Get / List / Delete + Scale API（不含其他 spec 更新）；MLService CR 下发；Informer 推 `Pending → Ready` | L1 integration 覆盖：Scale API 修改 PG `replicas` → reconciler patch CR `spec.roles[0].replicas` → ready_replicas 回流 |
| Outbox + Informer | 谓词只覆盖 `Creating` + `Deleting`；单副本无 leader election 代码路径；reconciler 失败指数退避 | grep `Canceling` / `desired_spec_hash` 谓词代码路径不被触发；reconciler 单元测试覆盖 4 类失败场景重试 |
| Backend 默认值 | MLJob `(native, job)` / MLService `(native, deployment)` 一律默认注入 | API 创建未指定 backend 时 PG `spec.backend = {name:native, engine:job/deployment}` |
| Bootstrap | post-install Job：default tenant + default pool + cpu-small/cpu-medium unit + `(default, default)` quota（`max` 按集群可用资源估算，`min=0`） | `helm install` 后查询 4 类对象都存在；重复 `helm upgrade` 不重复创建 |
| 测试 | API 单元测试 + Informer integration（fake tenant-controller / fake mljob-controller / fake mlservice-controller） | `make compute-test` + `make compute-integration` 通过 |

**显式延后到 §10.3**：`desired/applied spec hash`、Job `Canceling`/`Cancelled`、Tenant `Suspended`、CR 漂移补偿（仅外部误删先到位）、配额预检、`Degraded` 状态、metrics 全集、多 backend、Job 日志/副本/事件端点、Service spec 多字段更新。

### 10.3 阶段二：功能完善（生产硬化）

按"对生产可用性的影响"排序，每条标明完成信号。

1. **`desired_spec_hash` / `applied_spec_hash` 双 hash 机制**
   - 目标：覆盖 Tenant `displayName` / `annotations` / `quotas` / `initResources` / `suspended` 与 Service `spec.roles[0].replicas` 的 PG-only 异步 patch 路径。
   - 完成信号：API PATCH 写 PG `desired_spec_hash` → reconciler 检测差异 → patch CR → 写 `applied_spec_hash`；L1 integration 覆盖：连续 PATCH 多次只产生最后一次 patch；patch 失败 message 写入 PG。
2. **Tenant suspend / unsuspend + spec update**
   - 目标：管理员可暂停租户提交链路与调整 `displayName` / `initResources`。
   - 完成信号：suspend API → PG `status='Suspended'` 立即生效（阻塞新 Job/Service Create 校验）；unsuspend → reconciler patch CR → Informer 回 `Active`；L1 integration 覆盖。
3. **Quota spec update**
   - 目标：调整 `(tenant, pool, name)` 的 `min` / `max` 不需要重建 Quota。
   - 完成信号：PATCH Quota → 标记 `tenants.desired_spec_hash` → Tenant reconciler 重新渲染 `spec.quotas[]` → ElasticQuota.spec 跟随。
4. **Job cancel（`Canceling → Cancelled`）**
   - 目标：用户可以取消运行中 Job 并保留 PG 行供查阅。
   - 完成信号：cancel API → PG `Canceling` → reconciler patch `spec.runPolicy.suspend=true` → Informer 见 Suspended condition → `Cancelled` + `finished_at` + 入队 `Delete()`；L1 integration 覆盖竞速：cancel patch 与 Job 自然终态同时到达时 Compute 推到对应运行终态而非 `Cancelled`。
5. **配置对象 CR 漂移与反向孤儿补偿**
   - 目标：Tenant CR 字段被外部修改 / Tenant CR 被外部删除时按 PG desired spec 重建；Tenant CR 存在但 PG 行已软删时删除 CR + 审计。
   - 完成信号：L1 integration 覆盖：手动修改 Tenant CR `spec.displayName` → reconciler 下一轮覆盖回 PG 值；删除 Tenant CR → reconciler 重新创建。
6. **配额 best-effort 预检**
   - 目标：提交 Job/Service 时按 `quotas.spec.max - quotas.used` 早期拒绝，避免空跑 reconciler。
   - 完成信号：构造 used + request > max 场景，API 返回 4xx；metrics `axisml_compute_quota_precheck_rejected_total` 增长。
7. **Service `Degraded` / `Failed` 状态映射**
   - 目标：精确反映 ready_replicas vs desired_replicas 与 MLService `status.phase` 的组合。
   - 完成信号：L1 integration 覆盖 §9.3 主表四条 + `desired_replicas==0` 附注一条。
8. **多 backend 接入（与 operator §7.2 / §7.3 对齐）**
   - 目标：透传 MLJob `(native, podgroup)` / `(kubeflow-trainer, *)` 与 MLService `(native, statefulset)` / `(kserve, inference)` / `(kserve, llminference)` 的 `backend.config`，Compute 不解释 backend 运行时语义。
   - 完成信号：API 接受任意已注册 backend；CR `spec.backend` 字段透传准确；L1 integration 覆盖 `(native, podgroup)` 与 `(kserve, inference)` 两条最常用路径。
9. **Job 日志 + 副本 + 事件端点**
   - 目标：`/logs` 透传 + SSE follow、`/replicas` 与 `/events` 列表。
   - 完成信号：`/logs?follow=true` 流式可用且断开时关闭 upstream；GC 后 Pod 返 410；`/replicas` 按 `axisml.io/job-id` label 检索 Pod。
10. **完整 metrics 集**
    - 目标：暴露 reconciler 滞后 / spec 同步待办 / drift 修复 / API 延迟等业务指标。
    - 完成信号：以下指标在 `/metrics` 出现且有合理标签：

| 指标 | 类型 | 用途 |
| --- | --- | --- |
| `axisml_compute_is_leader` | gauge | 当前副本是否为 leader |
| `axisml_compute_reconciler_oldest_pending_seconds{resource,predicate}` | gauge | 工作集最老未处理行的 age（resource=tenant/job/service/quota，predicate=creating/canceling/deleting/spec_sync） |
| `axisml_compute_reconciler_actions_total{resource,predicate,result}` | counter | reconciler 动作计数（成功 / 重试 / 失败） |
| `axisml_compute_informer_workqueue_depth{resource}` | gauge | 各模块 Informer work queue 深度 |
| `axisml_compute_cr_drift_repair_total{resource,kind}` | counter | CR 漂移修复次数（kind=missing/spec_mismatch/orphan_delete） |
| `axisml_compute_spec_sync_pending_total{resource}` | gauge | `desired_spec_hash != applied_spec_hash` 的待同步行数 |
| `axisml_compute_quota_precheck_rejected_total{tenant,quota}` | counter | 配额预检拒绝次数 |
| `axisml_compute_api_request_duration_seconds{route,status}` | histogram | API 请求延迟分布 |

11. **`axisml.io/<resource>-id` label 锚点 + 软删命名复用全链路**
    - 目标：所有 CR 创建时打 label；孤儿检测一律按 label 索引而非 name；UNIQUE partial index 允许软删后再创建同名资源。
    - 完成信号：L1 integration 覆盖：`DELETE Job foo → Create Job foo` 同租户内成功；CR label 与 PG `id` 始终一致。

### 10.4 阶段三：未来规划

- **多副本 HA**：启用 controller-runtime leader election 完整代码路径，多副本压测与 leader 切换语义验证（lease duration / 重试节奏 / 两副本短暂并存窗口的 reconcile 幂等验证）。当前架构已为横向扩容预留接口（详见 §2.4）。
- **分层配额（Koord-Queue tree）**：`quotas` 表新增 `parent_id`，利用 Koordinator ElasticQuota 的 `quota.scheduling.koordinator.sh/parent` annotation 实现父子配额与抢占；在有真实多团队/多业务线共享租户的诉求时引入。
- **数据卷管理**：纳入 Compute（`components/compute/internal/volume/`），需补 schema（按租户隔离的 volume 定义与挂载声明）、底层存储映射（PVC / NFS / S3 / hostPath 等）、operator 注入契约（经 MLJob/MLService `spec.volumes` 下发）。
- **Custom backend 透传支持**：`spec.backend.{name: custom, engine: *}` 元组的 API 接受 + 透传；Compute 不解释 backend 内部，operator 侧详见 [operator.md §5.5.4 / §6.6.5](operator.md#5-mljob-controller)。
- **多集群联邦**：配额 / 任务按集群维度扩展，跨集群调度。
- **细粒度配额**：GPU 时长、存储容量、网络带宽等维度。
- **审计日志**：独立 `audit_events` 表，关键写操作（Create / Update / Delete / Cancel / Suspend）落库。
- **成本核算**：基于 `jobs` / `services` 实际使用时长 × 资源单元成本的计费导出。
- **mTLS / Compute 主动鉴权**：当前完全信任 Platform 注入的 `X-Axisml-User`；演进路径包括 mTLS 双向认证与 token 验证。
- **Service autoscaling**：基于配额用量与 QPS 的弹性扩缩；多 role 独立扩缩 `/scale` 路径（`services.replicas` 字段从单 role 升级为 role 显式寻址）。
- **OpenAPI 严格 schema 与 admission 类校验前移**：Compute 当前在 handler 层做业务校验；后续可演进为 OpenAPI 严格 schema + 运行时校验组合。

### 10.5 跨阶段验证策略

| 阶段 | 主测层 | 工具 |
| --- | --- | --- |
| MVP | API 单元测试 + L1 integration test（含 fake tenant/mljob/mlservice controller） | `make compute-test` + `make compute-integration` |
| 功能完善 | L1 integration 扩展 + 关键路径 L2 e2e（与 axisml-operator + helm 一起跑） | `make integration-test` + `make e2e-test`（minikube） |
| 未来规划 | 单独写 RFC 设计文档 → L1 integration 先行 → L2 验证多组件链路 | 同上 |

## 11. 测试

L1 integration 在 `components/compute/test/integration/` 单一 Go module 中：每个模块的 reconciler / Informer 路径覆盖 happy path + 关键 corner case（外部误删、漂移修复、cancel 竞速等）。fake operator 使用 controller-runtime 的 `envtest.Environment` + 简易 reconciler，模拟 tenant-controller / mljob-controller / mlservice-controller 的 status 写入。

API 层单元测试在各 `internal/<module>/handler_test.go`，覆盖请求参数校验、错误格式、租户校验等。

L2 e2e 在 `test/e2e/` 跑端到端：通过部署后的 axisml-compute + axisml-operator + Platform 模拟用户路径。L2 不在 CI 中执行（minikube 太慢/flaky），由本地或专门 runner 触发。

## 12. 相关引用

- [docs/system_design/overview.md §5.2](overview.md) 概述了 AxisML Compute 在控制平面里的位置。
- [docs/system_design/operator.md](operator.md) 描述 axisml-operator（tenant / mljob / mlservice controller）与 Compute 之间的 CR 写路径与状态回流契约；§3.1 Compute 写路径契约、§3.4 Pod 注入约定、§4.6 Tenant 底层资源管理是 Compute 频繁交叉引用的核心条目。
- [docs/system_design/infra.md §8](infra.md) 给出 koord-scheduler / ElasticQuota 等基础设施依赖契约，包括 §8.3 Quota 全覆盖契约（所有 AxisML workload Pod 强制走 koord-scheduler 并消费 ElasticQuota）。
- [docs/system_design/artifacts.md](artifacts.md) 描述 Artifacts 服务，Compute 在 Job / Service 提交时通过 HTTP 客户端做 image / model / dataset 引用懒查询。
