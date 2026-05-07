# AxisML Compute 详细设计

AxisML Compute 是平台的计算服务层，基于 Go 开发，仅接受来自 AxisML Platform 的内部 REST 调用，承载 **计算任务管理** 与 **资源池 / 资源单元管理**。Compute 不直接创建 Pod 等底层 K8s 资源——这些由 [compute-operator](compute-operator.md) 负责；Compute 仅维护业务元数据，并通过 CRD 向 operator 声明意图。

> **职责边界**：Compute 不持有租户与配额的元数据。租户与配额由 [cluster-manager](cluster-manager.md) + [tenant-operator](tenant-operator.md) 负责；Compute 把请求体里的 `namespace` 字段当作裸字符串分区键，把 `spec.scheduling.quota` 当作不透明 ElasticQuota CR 名透传。"用户视角的租户"由 [Platform](platform.md) 自己持有视图层映射。

| 模块 | PG 表 | 状态机 | 对应 K8s 资源 |
| --- | --- | --- | --- |
| ResourcePool ([§4](#4-resourcepool)) | `resource_pools` | 无（纯 PG 元数据） | 无 CR；Job/Service 提交时注入 |
| ResourceUnit ([§5](#5-resourceunit)) | `resource_units` | 无（纯 PG 元数据） | 无 CR；Job/Service 提交时注入 |
| Job ([§6](#6-job)) | `jobs` | `Creating \| Pending \| Running \| Succeeded \| Failed \| Canceling \| Cancelled \| Deleting \| Deleted` | `MLJob` CR（compute-operator） |
| Service ([§7](#7-service)) | `services` | `Creating \| Pending \| Ready \| Degraded \| Failed \| Deleting \| Deleted` | `MLService` CR（compute-operator） |

模块按生命周期分两类：**配置对象**（ResourcePool / ResourceUnit）长期存在，纯 PG 不下发 CR；**工作负载对象**（Job / Service）的 CR 缺失是合法的生命周期终点（取消、清理、operator 级联），Compute 不重建。详见 [§3.7](#37-孤儿与补偿)。

**文档组织**：

- **Part I — 服务运行时**（§1 架构总览 + §2 服务运行时契约）：HTTP 服务进程的运维契约。
- **Part II — 通用契约**（§3 跨模块通用契约）：4 个模块共享的写读模型、PG 编排约定、Outbox + Reconciler、双 hash spec 同步、Informer 状态回流、孤儿补偿、不变量；§4–§7 引用本节而不重复。
- **Part III — 模块详细设计**（§4 ResourcePool、§5 ResourceUnit、§6 Job、§7 Service）。
- **Part IV — 实施与验证**（§8 实现路径、§9 测试、§10 相关引用）。

---

## Part I — 服务运行时

> 本部分描述 Compute 服务进程的运维契约：HTTP 服务形态、PostgreSQL 与 Kubernetes 客户端、副本与 leader election、Helm values。各模块共享这些契约。

## 1. 架构总览

```
                          ┌──────────────────────┐
                          │  AxisML Platform     │
                          │     (唯一调用方)      │
                          └──────────┬───────────┘
                                     │ REST / JSON
                                     │ 身份透传 header
                                     ▼
                          ┌──────────────────────┐
                          │   AxisML Compute     │
                          │       (Go)           │
                          └──┬──────────┬────────┘
           ┌─────────────────┘          └──────────────────┐
           │                                               │
           ▼                                               ▼
 ┌──────────────────┐                         ┌─────────────────────────────┐
 │   PostgreSQL     │                         │   Kubernetes API Server     │
 │ 元数据           │                         │   MLJob / MLService         │
 │                  │                         │                             │
 └──────────────────┘                         └─────────────┬───────────────┘
                                                            │ list-watch
                                                            ▼
                                                   ┌──────────────────┐
                                                   │  Informer Loop   │
                                                   │  状态回流        │
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
│   ├── resourcepool/         # 资源池管理
│   ├── resourceunit/         # 资源单元管理
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
- Metrics：`GET /metrics`（Prometheus 格式，详见 §8.5）

### 2.2 PG 客户端与迁移

- ORM：GORM；与 Artifacts 等服务共用 database `axisml`，通过表名前缀逻辑隔离
- 迁移：`golang-migrate` embedded 方式在服务启动时执行，依赖 `schema_migrations` 表的 PG advisory lock 避免多副本并发迁移
- 连接：从 ConfigMap / Secret 读取 DSN；连接池参数（`maxOpen` / `maxIdle` / `connMaxLifetime`）通过 Helm values 暴露

### 2.3 Kubernetes 客户端

- controller-runtime `client.Client` + `SharedInformerFactory`，所有模块通过 `internal/k8sclient` 共享底层 cache
- 监听对象：`MLJob`（namespaced，全集群）、`MLService`（namespaced，全集群）；不监听 `Tenant` / `ElasticQuota`（归 cluster-manager / tenant-operator）
- 不引入 controller-runtime 的 `Manager` / `Reconciler` 抽象——Compute 不是 K8s controller，reconciler goroutine 自行管理（详见 §3.4）

### 2.4 副本与 Leader Election

- **默认 `replicas=1`**（Standard 与 Lite 均同）
- **API 层无状态**：所有副本都服务 HTTP，仅写 PG，水平扩容无需协调
- **后台协程选主**：reconciler goroutine 与各模块 Informer 只在 leader 副本运行，通过 controller-runtime 的 K8s `Lease` 选主；单副本时退化为单成员瞬时 lease
- **Leader 身份暴露**：`/metrics` 暴露 `axisml_compute_is_leader` gauge

### 2.5 RBAC（Compute → Kubernetes）

| 资源 | 动作 | 用途 |
| --- | --- | --- |
| `mljobs.axisml.io` | `create / get / list / watch / update / patch / delete` | 下发 MLJob CR、消费 status、cancel patch |
| `mlservices.axisml.io` | `create / get / list / watch / update / patch / delete` | 下发 MLService CR、消费 status、scale patch |
| `pods` | `get / list` | Job 副本列表与日志透传（跨 namespace） |
| `pods/log` | `get` | Job 日志 API 透传 kube-apiserver |
| `events` | `get / list` | Job 事件聚合端点 |
| `coordination.k8s.io/leases` (Compute 自身 ns) | `create / get / list / watch / update / patch / delete` | leader election Lease |

**不含**：`tenants.axisml.io`、`elasticquotas.scheduling.sigs.k8s.io`、`namespaces`、`secrets` / `configmaps` / `serviceaccounts`——这些归 cluster-manager / tenant-operator。

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
    defaultPool: default
    defaultUnits: [cpu-small, cpu-medium]
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
| `bootstrap-job.yaml` | post-install Job 初始化默认数据（详见 §8.2） |

---

## Part II — 通用契约

> 本部分集中 4 个模块**共享**的边界与协议：与 Platform 的请求契约、PG 编排约定、写读路径、状态回流、孤儿补偿、不变量。§4–§7 各模块章节引用本节而不重复。

## 3. 跨模块通用契约

### 3.1 与 Platform 的请求契约

Compute 仅接受 Platform 通过集群内 Service DNS 发起的 REST 调用，不直接对外部用户流量开放。

- **路径前缀**：`/api/v1/...`；不配置外部 `HTTPRoute`
- **身份头**：Platform 注入 `X-Axisml-User`（调用方用户唯一 ID）；Compute 仅做审计 / ownership 归属，不做角色级鉴权（鉴权由 Platform 统一完成）
- **namespace 分区**：通过 URL 路径 `/namespaces/{namespace}/...` 表达；Compute **不校验 namespace 是否存在或与 K8s namespace 强一致**——namespace 是裸字符串分区键，由 Platform 保证语义。Compute 在创建 MLJob / MLService CR 时直接把请求体里的 `namespace` 写到 `metadata.namespace`
- **OpenAPI 契约**：`components/compute/api/openapi.yaml` 是唯一契约源，使用 `oapi-codegen` 生成 Compute 侧 Go types + server stub（`api/types/`）与 Platform 侧 Go client SDK
- **错误格式**：HTTP 标准状态码 + RFC 7807 problem+json
- **写后异步语义**：所有会变更 CR spec 的 API（Service create/scale、Job create/cancel/delete）都只提交 PG desired state，**不承诺 CR 已同步完成**；同步失败由 reconciler 写入业务记录的 `message` 字段，调用方通过 Get/List 观察 `status`、`message` 与回流字段

### 3.2 PG 编排约定

- **通用字段**：所有表统一带 `id uuid`、`created_at`、`updated_at`、`deleted_at`（软删除）
- **UNIQUE 约束**：所有 schema 中标注的 `UNIQUE` 均实现为 PG partial unique index `WHERE deleted_at IS NULL`——软删行不占用唯一键，同名资源在原行被软删后可被再次创建
- **`name` 字段 DNS-1123 硬校验**：所有承载业务标识并会映射到 K8s 对象名的 `name` 字段（ResourcePool / Job / Service；ResourceUnit 在此之上还叠加 §5.3 的语义命名约定），API 层统一校验：字符集 `[a-z0-9-]`，首尾为字母或数字，长度 3–40，不允许连续 `--`
- **CR 稳定锚点**：所有 CR-backed 对象在 PG `id`（uuid）的同时打 label `axisml.io/{job,service}-id=<uuid>` 到 CR 上——`metadata.name` 因软删可能重用（partial UNIQUE），UUID 永久唯一；孤儿检测按 label 索引而非 name
- **namespace 字段**：Job 与 Service 表持有 `namespace text` 字段，`(namespace, name)` partial UNIQUE on `deleted_at IS NULL`

### 3.3 权威划分

> **PG 为业务元数据与期望 spec 的权威；Kubernetes 为运行状态的权威。**

| 类别 | 权威方 | 流向 |
| --- | --- | --- |
| 业务元数据与期望 spec（名称、引用、spec 快照、desired hash） | PG | API → PG → reconciler → CR `spec` |
| 运行状态（phase、endpoint、副本就绪） | K8s | CR `status` → Informer → PG |

配额 spec 与用量都不由 Compute 持有——`spec.scheduling.quota` 字段值是 Platform 在请求时透传的 ElasticQuota CR 名（cluster-manager / tenant-operator 一侧的产物）。

### 3.4 写路径（Outbox + Reconciler）

采用 **Outbox 模式**：

1. **API 同步路径只写 PG**：业务校验 → PG 事务插入 / 更新业务记录（新建时 `status='Creating'`，取消时 `status='Canceling'`，删除时 `status='Deleting'` + `deleted_at=now()`；允许变更的 spec 写入 PG 快照并更新 `desired_spec_hash`）→ commit → 返回业务 ID。**API 不直接写 K8s**。
2. **Compute 内 reconciler worker 异步下发 CR**：每个模块（`internal/{job,service}` 持有 reconciler）在 leader 副本起 goroutine，周期性扫描 PG 按下列**四个谓词**分派动作：

| 谓词 | 动作 | 适用模块 |
| --- | --- | --- |
| `status='Creating' AND deleted_at IS NULL` | `Create()` CR（附 label `axisml.io/<resource>-id=<uuid>`；409 `AlreadyExists` 视为成功，靠 `metadata.name` + label 双重去重幂等） | Job / Service |
| `status='Canceling'` | `patch MLJob.spec.runPolicy.suspend=true`；后续推进与竞速处理详见 §6.4.2 | Job |
| `status='Deleting'` | `Delete()` CR；Informer DELETE 事件推进到 `Deleted`（配合设置 `deleted_at`） | Job / Service |
| `desired_spec_hash != applied_spec_hash AND deleted_at IS NULL` | 按双 hash 机制 `Patch()`（详见 §3.5） | Service |

失败按指数退避重试，错误写入业务记录的 `message` 字段供 UI 展示。**PG 行不再满足任何谓词后，reconciler 不再做下发动作**——Job 自然结束、Service `Failed` 自愈、外部误删等情况由 Informer 回流推进，不进入 reconciler 工作集。

**共享状态机骨架**（各资源完整状态集与转换见 §6–§7）：

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

- 就绪态：Job 为 `Pending`（继而 `Running`）；Service 为 `Pending`（继而 `Ready`/`Degraded`/`Failed`）
- 运行终态：Job 有 `Succeeded`/`Failed`；Service 无运行终态（`Failed` 可自愈）
- `Cancelled` 为 Job 独有；`Deleted` 为所有资源最终软删终态

### 3.5 desired/applied spec hash 双 hash 机制

允许变更的 CR-backed 对象通过双 hash 保持 PG-only 写路径：API 写 PG 时同步重算 `desired_spec_hash`；reconciler 在 `desired_spec_hash != applied_spec_hash AND deleted_at IS NULL` 时执行幂等 `Patch()`，成功后写 `applied_spec_hash=desired_spec_hash`；后续运行状态仍由 Informer 回流。Hash 由 `internal/spechash` 计算，输入是字段子集的归一化 JSON。

| 资源 | 是否使用 | 允许变更字段 |
| --- | --- | --- |
| Service | ✅ | `spec.roles[0].replicas`（其他字段不可变） |
| Job | ❌ | 不可变；cancel 由 `Canceling` 谓词 patch `spec.runPolicy.suspend=true`，不走 spec 同步 |
| ResourcePool / ResourceUnit | ❌ | 无对应 CR，无下行同步 |

### 3.6 状态回流（Informer）

两条独立 Informer，通过 `k8sclient` 的 `SharedInformerFactory` 共享底层 cache。

| Informer | 监听对象 | 维护方 | 主要用途 |
| --- | --- | --- | --- |
| MLJob Informer | `MLJob` CR | `internal/job/` | 推进 `Creating→Pending→Running→Succeeded/Failed`；`Canceling→Cancelled`（含与自然终态竞速处理，详见 §6.4.2）；`Deleting→Deleted`；DELETE 事件按 §3.7 工作负载策略处理；回写 `jobs.status` / `started_at` / `finished_at` |
| MLService Informer | `MLService` CR | `internal/service/` | 推进 `Creating→Pending→Ready/Degraded/Failed`（映射规则详见 §7.3）；`Deleting→Deleted`；DELETE 事件按 §3.7 工作负载策略处理；回写 `services.status` / `ready_replicas` / `endpoint` |

通用模式：启动时 `List` 做差异 upsert 与孤儿对账；`Watch` 事件入各自 work queue；单 worker 串行 reconcile；以 `resourceVersion` / `generation` 作乐观并发字段保证幂等。

### 3.7 孤儿与补偿

补偿策略只覆盖 **工作负载对象（Job、Service）**——配置对象 ResourcePool / ResourceUnit 没有对应 CR，无需 CR 漂移补偿。Compute **不反向重建 CR**。Informer 观察到 CR DELETE 事件后按 PG 当前 `status` 分流（外部误删处理的单点定义）：

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
- **CR `metadata` / `spec` 单写**：Compute 是 CR `metadata` 与 `spec` 的唯一写入方；operator 只写 `status`（与 [compute-operator.md §3.1](compute-operator.md) 对齐）。
- **status 单向消费**：Compute 只**读** CR `status`，并把统一字段（`phase` / `message` / `startedAt` / `finishedAt` 等）回写 PG；不向 CR `status` 写任何字段。
- **不感知 namespace 是否真实存在**：namespace 是裸字符串分区键；如果 K8s 中没有该 namespace，CR Create 会失败并被 reconciler 写入 `message`，调用方（Platform）需自行排查。
- **生命周期谓词收敛**：reconciler 工作集只来自 §3.4 的四个谓词；PG 行不再满足谓词后 reconciler 不做下发动作。
- **CR 稳定锚点**：所有 CR-backed 对象的 `axisml.io/<resource>-id=<uuid>` label 必填。

---

## Part III — 模块详细设计

> 本部分逐个展开 4 个模块的数据模型、状态机、生命周期与对外契约。配置对象（ResourcePool / ResourceUnit）在前，工作负载对象（Job / Service）在后。

## 4. ResourcePool

### 4.1 概述

ResourcePool 是 AxisML Compute 维护的纯 PG 元数据对象，用于把集群节点按物理或逻辑维度切分（GPU 池 / CPU 池、A100 池 / H100 池、训练池 / 推理池）。**ResourcePool 无对应 CR**，不走 Outbox 下发路径——它仅在生成 MLJob/MLService 时作为 `spec.scheduling.nodeSelector` / `tolerations` 的注入源。

管理员负责在集群侧给目标节点打标签 / 污染，Compute 不修改 Node 对象。ResourcePool 是全局可见的——Compute 不做"按租户隔离 ResourcePool 可见性"的校验，由 Platform 决定哪个工作区可以使用哪个 pool。

### 4.2 数据模型

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

**默认池**：Helm `post-install` Job 初始化 `default` 池，`node_selector` 留空即表示整集群可用（详见 §8.2）。

### 4.3 注入规则

提交 Job/Service 时由 `internal/resourceunit/inject.go` 与 ResourceUnit 的 `node_selector` 合并后写入 CR：

- `pool.node_selector` 的 key 全部保留（**Pool 优先**）
- `pool.tolerations` 直接作为 `spec.scheduling.tolerations`

详细合并规则见 §5.4。

### 4.4 API 端点

| 端点 | 方法 | 用途 |
| --- | --- | --- |
| `/api/v1/resource-pools` | `POST` / `GET` | Create / List |
| `/api/v1/resource-pools/{pool}` | `GET` / `PATCH` / `DELETE` | Get / Update / Delete |

仅管理员可写；Platform 通过角色判断后转发。

### 4.5 后续工作

- **池间调度策略**：当配额不足时是否允许跨池借用（默认禁止）
- **池容量预估**：聚合所属节点的可分配资源，用于配额规划

## 5. ResourceUnit

### 5.1 概述

ResourceUnit 是 ResourcePool 内预先定义的资源规格模板，是 AxisML Compute 维护的纯 PG 元数据对象。例如 `a100-1x-large` 可表示 1×A100 + 8 vCPU + 32 GiB。用户创建任务或服务时选择一个 ResourceUnit，Compute 据此注入 `requests` / `limits` 与节点匹配条件。

**ResourceUnit 无对应 CR**，不走 Outbox 下发路径。

### 5.2 数据模型

```
resource_units(
  id             uuid PK,
  pool_id        uuid FK resource_pools(id),
  name           text,
  description    text,
  requests       jsonb,                   -- {"cpu":"8","memory":"64Gi","nvidia.com/gpu":"1"}
  limits         jsonb,
  node_selector  jsonb,                   -- 通用节点标签匹配（见 §5.4）
  created_at     timestamptz,
  updated_at     timestamptz,
  deleted_at     timestamptz,
  UNIQUE(pool_id, name)
)
```

**字段说明**

- `requests` / `limits`：标准 K8s 资源
- `node_selector`：通用节点标签匹配，覆盖 GPU 之外的任意硬件维度

**常见 `node_selector` 用法**：

| 场景 | 示例 |
| --- | --- |
| GPU 型号 | `{"nvidia.com/gpu.product": "A100-SXM4-80GB"}` |
| TPU | `{"cloud.google.com/gke-accelerator": "tpu-v4"}` |
| 自研加速卡 | `{"axisml.io/accelerator": "npu-x"}` |
| CPU instance type | `{"node.kubernetes.io/instance-type": "c5.4xlarge"}` |

### 5.3 命名约定

格式 `<accelerator>[-<count>x]-<tier>[-<variant>]`：

- `<accelerator>`：加速卡或 CPU 标识（小写 kebab，如 `a100` / `h100` / `tpu-v4` / `ascend-910b` / `cpu`）
- `<count>x`：加速卡数量（如 `1x` / `2x` / `4x` / `8x`），CPU-only 省略
- `<tier>`：规格档 `small` / `medium` / `large` / `xlarge`
- `<variant>`：可选后缀，如 `himem` / `ssd` / `spot` / `ib`

示例：`cpu-small`、`a100-1x-large`、`h100-8x-xlarge-ib`、`tpu-v4-4x-large`。字符集校验同 §3.2。

### 5.4 注入规则

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

### 5.5 API 端点

| 端点 | 方法 | 用途 |
| --- | --- | --- |
| `/api/v1/resource-pools/{pool}/resource-units` | `POST` / `GET` | Create / List |
| `/api/v1/resource-pools/{pool}/resource-units/{unit}` | `GET` / `PATCH` / `DELETE` | Get / Update / Delete |

仅管理员可写。

### 5.6 后续工作

- **混合资源单元**：单 unit 内表达"CPU 1 + GPU 0.5（MIG）"等复杂规格
- **价格元数据**：`unit.metadata.cost_per_hour` 用于成本核算演进

## 6. Job

### 6.1 概述

Job 模块维护 PG `jobs` 表 + 下发 `MLJob` CR + 通过 MLJob Informer 回流执行状态。Job 是一次性 workload，对应 §3.4 中的**工作负载对象**——CR 缺失视为合法生命周期终点，不补偿重建。

`MLJob` CR 字段契约由 [compute-operator.md §4](compute-operator.md) 定义；Compute 仅负责把业务语义装进 CR 并消费统一的 `status.phase` / `message` / `startedAt` / `finishedAt` / `conditions[type=Suspended]`。

### 6.2 数据模型

```
jobs(
  id                   uuid PK,
  namespace            text,                     -- 裸字符串分区键，写入 MLJob CR metadata.namespace
  pool_id              uuid FK resource_pools(id),
  resource_unit_id     uuid FK resource_units(id),
  name                 text,                     -- MLJob CR metadata.name
  display_name         text,
  description          text,
  owner_user           text,                     -- 来自 X-Axisml-User
  spec                 jsonb,                    -- 提交时的 MLJob.spec 完整快照（不可变）
  requested_resources  jsonb,                    -- 资源申请快照
  status               text,                     -- Creating / Pending / Running / Succeeded / Failed / Canceling / Cancelled / Deleting / Deleted
  message              text,
  started_at           timestamptz,
  finished_at          timestamptz,
  created_at           timestamptz,
  updated_at           timestamptz,
  deleted_at           timestamptz,
  UNIQUE(namespace, name)                        -- partial on deleted_at IS NULL
)
```

**字段归属**

- `spec` 是提交时 MLJob.spec 的完整快照，包含 `roles[]` / `backend.{name,engine,config}` / `scheduling` / `runPolicy` 等业务字段（结构详见 [compute-operator.md §4.2](compute-operator.md)），**不可变**；Informer 回流只写 `status` 相关列
- `requested_resources` 冗余存提交时的资源申请，解耦后续 ResourceUnit 修改对已提交任务的影响
- `name` 是 Compute 与 K8s 的命名锚点（对应 MLJob CR `metadata.name`），UUID `id` 通过 label `axisml.io/job-id=<id>` 同步打到 CR 上
- `spec.scheduling.quota` 字段值由 Platform 在提交请求时透传——Compute 不解析，也不校验该 ElasticQuota 是否真实存在；如果 Pod 调度时发现 quota 缺失，由 koord-scheduler 拒绝调度并由 operator 回流到 `status.phase=Pending` + 失败 condition

**`spec.backend` 默认值注入**：用户未指定 `spec.backend` 时，Compute 写 CR 时显式补 `{name: "native", engine: "job"}`；`backend.config` 默认空对象 `{}`。`backend.{name, engine}` 创建后不可变。

### 6.3 状态机

```
Creating ──(Informer ADD)──▶ Pending ──(CR phase=Running)──▶ Running ──(CR phase=Succeeded)──▶ Succeeded
                                │                              │    ──(CR phase=Failed)────▶ Failed
                                │                              │
                                │  cancel API                  │  cancel API
                                └──────────────────────────────┴─▶ Canceling ──(Suspended condition)──▶ Cancelled

任一非 Canceling/Deleting/Deleted 状态 ──(DELETE req)──▶ Deleting ──(CR 确认清理 + deleted_at)──▶ Deleted
```

- `Creating` 状态不接受 cancel
- `Cancelled` PG 行保留（`deleted_at IS NULL`），用户可再次 DELETE → `Deleting` → `Deleted`
- `Succeeded` / `Failed` / `Cancelled` 三者为运行终态；`Deleted` 为软删终态

### 6.4 业务语义

#### 6.4.1 提交校验

`POST /api/v1/namespaces/{namespace}/jobs` 在 §3.4 通用 Outbox 流程之外附加：

1. 从 `X-Axisml-User` 读取调用方身份（审计与 ownership 归属）
2. 业务校验：ResourceUnit 所属 pool 与请求中的 pool 一致；`spec.scheduling.quota` 字段必填（值由 Platform 透传，Compute 不解析含义）

Compute 不做"租户激活"或"配额归属租户"校验——这些由 Platform 自身保证。

#### 6.4.2 取消语义

`POST /api/v1/namespaces/{namespace}/jobs/{job}/cancel`

| 场景 | PG 侧 | CR 侧 |
| --- | --- | --- |
| 运行态（`Pending` / `Running`） cancel | `status='Canceling'`，写 `message='user cancelled'` | reconciler `patch MLJob.spec.runPolicy.suspend=true` → operator Handler 完成 suspend/Cleanup 后写 `status.conditions[type=Suspended,status=True,reason=CancelRequested]` → Informer 推 `Cancelled`、写 `finished_at` 并入队 `Delete()` CR 做资源回收 |
| `Canceling` 期间 Job 自然结束（**竞速**） | 维持 `Canceling` 直到 Informer 观察到 CR 终态；按 operator "终态优先"推到 `Succeeded`/`Failed` 并写 `finished_at`（operator 不写 Suspended condition） | reconciler 已发出 cancel patch；不再补偿 |
| `Creating` 状态 cancel | API 拒绝（要求改用 DELETE） | — |
| 已终态或已在 `Canceling`/`Deleting` cancel | API 返回无效操作 | — |

#### 6.4.3 删除语义

`DELETE /api/v1/namespaces/{namespace}/jobs/{job}`

| 场景 | PG 侧 | CR 侧 |
| --- | --- | --- |
| 任一非 `Deleting`/`Deleted` 状态 DELETE | `status='Deleting'`，写 `deleted_at` | reconciler `Delete()` CR；Informer DELETE → `Deleted` |
| `Creating` 状态 DELETE（CR 未下发） | 同上；reconciler 确认 CR 不存在 → 直接推 `Deleted` | — |
| `Cancelled` 状态 DELETE（CR 已被 cancel 路径回收） | 同上；reconciler `Delete()` CR 收到 404 → 幂等确认后直接推 `Deleted` | — |
| `Deleting`/`Deleted` 再次 DELETE | 幂等，忽略 | — |

外部误删（PG 在 `Pending`/`Running` 收到 CR DELETE 事件）的处理见 §3.7。

#### 6.4.4 任务日志透传

`GET /api/v1/namespaces/{namespace}/jobs/{job}/logs` 只做路径级鉴权与 Pod 定位，实际日志 IO 透传到 kube-apiserver 的 Pod Log API。

- **寻址**：通过 `replica`（基于 `axisml.io/replica-index` label）或 `pod`（直接 Pod 名）定位；多容器场景可指定 `container`
- **流式**：`follow=false`（默认）直接透传 `text/plain`；`follow=true` 进入 SSE 流式模式
- **鉴权**：只校验路径中 `namespace` 归属 `job`
- **GC 后**：Pod 已被回收时返 410 Gone

具体查询参数以 `components/compute/api/openapi.yaml` 为准。

#### 6.4.5 副本与事件端点

| 端点 | 用途 |
| --- | --- |
| `/api/v1/namespaces/{namespace}/jobs/{job}/replicas` | List Pod 副本（编号 / pod 名 / phase / started_at），按 `axisml.io/job-id` label 查询 |
| `/api/v1/namespaces/{namespace}/jobs/{job}/events` | 聚合 MLJob / Pod / PodGroup 的 K8s Event |

### 6.5 与 compute-operator 的契约

- **Compute phase 映射**：`Pending → Pending`、`Running → Running`、`Succeeded → Succeeded`（终态）、`Failed → Failed`（终态）
- **Cancel 推进信号**：Compute Informer 在 PG `status='Canceling'` 时把 `conditions[type=Suspended,status=True,reason=CancelRequested]` 当作推进信号 → 写 `Cancelled` → 入队 `Delete()` CR 做资源回收
- **终态优先**：cancel patch 与 Job 自然完成竞速时，operator 优先保留终态 `phase` 与 `finishedAt`

### 6.6 后续工作

- **Job spec 部分可变**：当前 `spec` 完全不可变；考虑允许 `display_name` / `description` / `runPolicy.activeDeadlineSeconds` 等元数据字段更新
- **Job 模板 / 工作流编排**：组合多个 Job 形成 DAG 级运行计划

## 7. Service

### 7.1 概述

Service 模块维护 PG `services` 表 + 下发 `MLService` CR + 通过 MLService Informer 回流副本就绪与 endpoint。Service 是常驻 workload，通过 `/scale` 端点实现弹性扩缩。

`MLService` CR 字段契约由 [compute-operator.md §5](compute-operator.md) 定义。

### 7.2 数据模型

```
services(
  id                   uuid PK,
  namespace            text,                     -- 裸字符串分区键，写入 MLService CR metadata.namespace
  pool_id              uuid FK resource_pools(id),
  resource_unit_id     uuid FK resource_units(id),
  name                 text,                     -- MLService CR metadata.name
  display_name         text,
  description          text,
  owner_user           text,
  spec                 jsonb,                    -- 当前 MLService.spec 快照
  desired_spec_hash    text,
  applied_spec_hash    text,
  requested_resources  jsonb,                    -- 单副本资源申请快照
  replicas             int,                      -- 单 role 约定下 = spec.roles[0].replicas
  ready_replicas       int,                      -- Informer 回流
  endpoint             text,                     -- 服务地址（Informer 回流）
  status               text,                     -- Creating / Pending / Ready / Degraded / Failed / Deleting / Deleted
  message              text,
  created_at           timestamptz,
  updated_at           timestamptz,
  deleted_at           timestamptz,
  UNIQUE(namespace, name)
)
```

**字段归属**

- `spec` 与 jobs 不同：扩缩容 API 更新 `replicas` 同时回写 `spec.roles[0].replicas`（单 role 约定）并重算 `desired_spec_hash`，其他字段依然不可变
- `endpoint`：`(native, *)` / `(custom, *)` 可为内部 Service DNS 或 AxisML Gateway URL；`(kserve, *)` 为 KServe `status.url`

**`spec.backend` 默认值注入**：用户未指定 `spec.backend` 时，Compute 写 CR 时显式补 `{name: "native", engine: "deployment"}`。

### 7.3 状态机

```
Creating ──(Informer ADD)──▶ Pending ──(ready=desired, desired>0)──▶ Ready ⇄ Degraded ──▶ Failed
                                                                      ▲                     │
                                                                      └─────── 自愈 ────────┘

任一非 Deleting/Deleted 状态 ──(DELETE req)──▶ Deleting ──(CR 确认清理 + deleted_at)──▶ Deleted
```

**status 主流映射**：

| 条件 | status |
| --- | --- |
| `ready_replicas == desired_replicas && desired_replicas > 0` | `Ready` |
| `0 < ready_replicas < desired_replicas` | `Degraded` |
| `ready_replicas == 0 && desired_replicas > 0` 且 MLService `status.phase=Pending` | `Pending` |
| `ready_replicas == 0 && desired_replicas > 0` 且 MLService `status.phase=Failed` | `Failed`（非终态，可自愈） |

> **附注 corner case**：`desired_replicas == 0` 时 `status='Pending'`。

`Ready` / `Degraded` / `Failed` 均为**非终态**——operator 自愈后 Informer 回流可让 `Failed → Degraded → Ready` 自然恢复；只有 `Deleted` 是 Service 的最终终态。

### 7.4 业务语义

#### 7.4.1 提交校验

同 §6.4.1（ResourceUnit / pool 一致；`spec.scheduling.quota` 字段必填）。

#### 7.4.2 扩缩容

`POST /api/v1/namespaces/{namespace}/services/{service}/scale`

走 PG-only Outbox：API 层只更新 `services.replicas` + `services.spec.roles[0].replicas` 并重算 `desired_spec_hash`，返回"desired replicas 已提交"；reconciler 后续按 §3.5 双 hash 机制 patch MLService CR path `spec/roles/0/replicas`。

#### 7.4.3 删除语义

`DELETE /api/v1/namespaces/{namespace}/services/{service}`

| 场景 | PG 侧 | CR 侧 |
| --- | --- | --- |
| 任一非 `Deleting`/`Deleted` 状态 DELETE | `status='Deleting'`，写 `deleted_at` | reconciler `Delete()` CR；Informer DELETE → `Deleted` |
| `Creating` 状态 DELETE（CR 未下发） | 同上；reconciler 确认 CR 不存在 → 直接推 `Deleted` | — |

外部误删（运行态收到 CR DELETE 事件）的处理见 §3.7。Service 没有 cancel 语义。

### 7.5 与 compute-operator 的契约

- **Compute phase 映射**：详见 §7.3 表格
- **`/scale` 透传**：mlservice handler 必须把 `roles[*].replicas` 透传为后端原生扩缩，详见 [compute-operator.md §3.6](compute-operator.md) 控制信号义务
- **路由派生**：`(native, *)` 的外部入口由 mlservice handler 派生 `HTTPRoute`，Compute 不参与路由配置；`(kserve, *)` 使用 KServe 自带 route

### 7.6 后续工作

- **多 role 独立扩缩**：当前 `services.replicas` 字段在单 role 约定下定义为 `spec.roles[0].replicas`；多 role 场景需要 role 显式寻址
- **autoscaling**：基于 `request_rate` 的弹性扩缩
- **`spec.route` 热更新**：轮换 API key / 调限流不重建 Service / Deployment

---

## Part IV — 实施与验证

> 本部分给出 Compute 的功能落地路线、测试策略与跨文档引用。

## 8. 实现路径

### 8.1 阶段总览

```
┌──────────────────────────────────────────────────────────────┐
│ MVP（最小可发布）                                             │
│   单副本 / 4 模块 CRUD / 默认 backend / Outbox 仅 Creating+   │
│   Deleting / Informer 状态回流 / bootstrap default 数据       │
│   ↓                                                           │
│ 功能完善（生产硬化）                                          │
│   双 hash spec 同步 / Cancel / 外部误删处理 / 多 backend /    │
│   日志 + 副本 + 事件端点 / metrics 全集                       │
│   ↓                                                           │
│ 未来规划（需求 / 上游驱动）                                   │
│   多副本 HA / 数据卷管理 / 多集群 / 审计 / 计费                │
└──────────────────────────────────────────────────────────────┘
```

### 8.2 阶段一：MVP

支撑端到端最小演示路径："建池 → 配资源 → 提任务/服务 → 看结果"。

| 模块 | 范围 | 完成信号 |
| --- | --- | --- |
| 服务运行时 | 单副本 / `replicas=1`、Helm 模板（configmap / deployment / service / sa / rbac / servicemonitor / bootstrap-job）、`/healthz` + `/readyz` + `/metrics`、PG 启动迁移 | `helm install` 后 Pod Ready；`/healthz` 200；PG `schema_migrations` 表存在 |
| ResourcePool | CRUD API（无 CR） | API 测试覆盖；新建 Pool 后 ResourceUnit 可挂靠 |
| ResourceUnit | CRUD API（无 CR）；提交 Job/Service 时注入 `requests` / `limits` / `nodeSelector` / `tolerations`（Pool 优先合并） | API 测试覆盖；L1 integration 覆盖：提交 Job 后 MLJob CR `spec.scheduling` 与 `spec.roles[*].template.resources` 字段值正确 |
| Job | Create / Get / List / Delete API（不含 cancel / logs / events / replicas / spec update）；MLJob CR 下发；Informer 推 `Pending → Running → Succeeded/Failed`；DELETE → `Deleting → Deleted` | L1 integration 覆盖：API Create → MLJob CR 出现 + 状态推进；DELETE → CR 删除 + PG `Deleted`；外部误删 Pending/Running CR 时 PG 推 `Cancelled` |
| Service | Create / Get / List / Delete + Scale API（不含其他 spec 更新）；MLService CR 下发；Informer 推 `Pending → Ready` | L1 integration 覆盖：Scale API 修改 PG `replicas` → reconciler patch CR `spec.roles[0].replicas` → ready_replicas 回流 |
| Outbox + Informer | 谓词只覆盖 `Creating` + `Deleting`；单副本无 leader election 代码路径 | reconciler 单元测试覆盖 4 类失败场景重试 |
| Backend 默认值 | MLJob `(native, job)` / MLService `(native, deployment)` 一律默认注入 | API 创建未指定 backend 时 PG `spec.backend = {name:native, engine:job/deployment}` |
| Bootstrap | post-install Job：default pool + cpu-small/cpu-medium unit | `helm install` 后查询 2 类对象都存在 |
| 测试 | API 单元测试 + Informer integration（fake compute-operator） | `make compute-test` + `make compute-integration` 通过 |

### 8.3 阶段二：功能完善

按"对生产可用性的影响"排序，每条标明完成信号。

1. **`desired_spec_hash` / `applied_spec_hash` 双 hash 机制（Service `replicas`）**
   - 完成信号：API PATCH 写 PG `desired_spec_hash` → reconciler 检测差异 → patch CR → 写 `applied_spec_hash`；L1 integration 覆盖。
2. **Job cancel（`Canceling → Cancelled`）**
   - 完成信号：cancel API → PG `Canceling` → reconciler patch `spec.runPolicy.suspend=true` → Informer 见 Suspended condition → `Cancelled` + `finished_at` + 入队 `Delete()`；L1 integration 覆盖竞速：cancel patch 与 Job 自然终态同时到达时 Compute 推到对应运行终态而非 `Cancelled`。
3. **Service `Degraded` / `Failed` 状态映射**
   - 完成信号：L1 integration 覆盖 §7.3 主表四条 + `desired_replicas==0` 附注一条。
4. **多 backend 接入**
   - 完成信号：API 接受任意已注册 backend；CR `spec.backend` 字段透传准确；L1 integration 覆盖 `(native, podgroup)` 与 `(kserve, inference)` 两条最常用路径。
5. **Job 日志 + 副本 + 事件端点**
   - 完成信号：`/logs?follow=true` 流式可用且断开时关闭 upstream；GC 后 Pod 返 410；`/replicas` 按 `axisml.io/job-id` label 检索 Pod。
6. **完整 metrics 集**

| 指标 | 类型 | 用途 |
| --- | --- | --- |
| `axisml_compute_is_leader` | gauge | 当前副本是否为 leader |
| `axisml_compute_reconciler_oldest_pending_seconds{resource,predicate}` | gauge | 工作集最老未处理行的 age |
| `axisml_compute_reconciler_actions_total{resource,predicate,result}` | counter | reconciler 动作计数 |
| `axisml_compute_informer_workqueue_depth{resource}` | gauge | 各模块 Informer work queue 深度 |
| `axisml_compute_spec_sync_pending_total{resource}` | gauge | 待同步行数 |
| `axisml_compute_api_request_duration_seconds{route,status}` | histogram | API 请求延迟分布 |

7. **`axisml.io/<resource>-id` label 锚点 + 软删命名复用全链路**
   - 完成信号：L1 integration 覆盖：`DELETE Job foo → Create Job foo` 同 namespace 内成功；CR label 与 PG `id` 始终一致。

### 8.4 阶段三：未来规划

- **多副本 HA**：启用 controller-runtime leader election 完整代码路径，多副本压测与 leader 切换语义验证。
- **数据卷管理**：纳入 Compute（`components/compute/internal/volume/`），需补 schema、底层存储映射、operator 注入契约。
- **Custom backend 透传支持**：`spec.backend.{name: custom, engine: *}` 元组的 API 接受 + 透传。
- **多集群联邦**：任务按集群维度扩展。
- **审计日志**：独立 `audit_events` 表，关键写操作落库。
- **成本核算**：基于 `jobs` / `services` 实际使用时长 × 资源单元成本的计费导出。
- **mTLS / Compute 主动鉴权**：当前完全信任 Platform 注入的 `X-Axisml-User`。
- **Service autoscaling**：基于 QPS 的弹性扩缩；多 role 独立扩缩 `/scale` 路径。
- **OpenAPI 严格 schema 与 admission 类校验前移**。

### 8.5 跨阶段验证策略

| 阶段 | 主测层 | 工具 |
| --- | --- | --- |
| MVP | API 单元测试 + L1 integration test（含 fake compute-operator） | `make compute-test` + `make compute-integration` |
| 功能完善 | L1 integration 扩展（envtest + testcontainers Postgres + httptest 驱动 HTTP API） | `make integration-test` |
| 未来规划 | 单独写 RFC 设计文档 → L1 integration 先行 | 同上 |

## 9. 测试

L1 integration 在 `components/compute/test/integration/` 单一 Go module 中：每个模块的 reconciler / Informer 路径覆盖 happy path + 关键 corner case（外部误删、cancel 竞速等）。fake operator 使用 controller-runtime 的 `envtest.Environment` + 简易 reconciler，模拟 compute-operator 的 status 写入。

API 层单元测试在各 `internal/<module>/handler_test.go`，覆盖请求参数校验、错误格式、namespace 校验等。HTTP API 契约测试在 L1 integration 中以 in-process gin engine + `httptest` 驱动（参见 `test/integration/httptest_helpers_test.go`）。

仓库当前不维护 minikube 驱动的 L2 e2e 层；端到端验证靠 L1 integration（envtest + testcontainers）覆盖。

## 10. 相关引用

- [docs/system_design/overview.md](overview.md) 概述了 AxisML Compute 在控制平面里的位置。
- [docs/system_design/compute-operator.md](compute-operator.md) 描述 compute-operator（mljob / mlservice controller）与 Compute 之间的 CR 写路径与状态回流契约。
- [docs/system_design/cluster-manager.md](cluster-manager.md) 描述租户与配额的入口；Compute 不直接调用，由 Platform 维护两者的关联。
- [docs/system_design/infra.md](infra.md) 给出 koord-scheduler / ElasticQuota 等基础设施依赖契约。
- [docs/system_design/artifacts.md](artifacts.md) 描述 Artifacts 服务，Compute 在 Job / Service 提交时通过 HTTP 客户端做 image / model / dataset 引用懒查询。
