# AxisML Lite 系统设计

## 1. 目标与边界

AxisML Lite 是 AxisML 的**无 Kubernetes 单机部署形态**。它在一台 Linux 主机上通过 Docker Compose 启动控制面与基础设施，并通过 Docker Engine API 动态运行训练任务、工作区和在线服务。

AxisML Lite 与 **Standard 形态**(AxisML 的完整 Kubernetes 部署形态,与 Lite 相对)共享 Platform API、`MLRun` / `MLService` / `MLTrafficPolicy` workload contract、PostgreSQL schema 和主要用户流程。System 层根据部署形态将 workload contract 映射为 Kubernetes 或 Docker 资源。

设计目标：

- 使用 `docker compose` 完成安装、升级、停止和基础备份。
- 保留 Job / Experiment / Run、Service / Workspace / TensorBoard、Model / Image / Dataset 等主要能力。
- 以独立 `axisml-lite` 项目交付产品封装、Standalone Runtime 和 Compose 资产。
- AxisML 主仓库维护共享 API contract、领域实现和 Standard 形态。
- Lite 固定单租户、单主机和单控制面副本。

非目标：

- 模拟 Kubernetes API、CRD 或 Namespace。
- 支持多节点调度、节点故障迁移、弹性配额借用和跨主机分布式训练。
- 支持 KServe、Kubeflow Trainer 等 Kubernetes 原生 backend。
- 在 Kubernetes 与 Lite 之间迁移 workload、运行态或本地持久卷。
- 修改 `compute-operator` 的 reconciler、handler 或底层 Kubernetes 资源映射逻辑。

## 2. 产品形态与功能兼容

Lite 固定使用 `default` tenant scope。现有 API 和 PostgreSQL schema 中的 `namespace` 字段在 Lite 中表示 tenant scope。

| 能力 | Lite | 说明 |
| --- | :---: | --- |
| 用户、登录、RBAC | ✅ | 保留 Platform 现有能力；租户角色都落在 `default` |
| Job / Experiment / Run | ✅ | 支持 `(native, job)`，由 Docker container 执行 |
| Run cancel / logs / events | ✅ | 映射 Docker stop / logs / runtime event |
| 在线 Service | ✅ | 支持 `(native, deployment)` 的单机实现 |
| Workspace / TensorBoard | ✅ | 复用常驻 container + volume + Traefik route |
| Service scale | ⚠️ | 支持本机多副本；不具备跨节点分布和故障迁移 |
| TrafficPolicy | ✅ | weighted / canary / bluegreen 映射为 Traefik 动态配置 |
| Model / Image / Dataset | ✅ | PostgreSQL + zot + S3 兼容对象存储 |
| ResourcePool / ResourceUnit | ⚠️ | 仅一个 `default` pool；unit 是本机资源规格模板 |
| 多租户 / Namespace 隔离 | ❌ | 仅 `default` tenant；隔离边界为容器、网络和目录 |
| ElasticQuota / 借用回收 | ❌ | 不做本机 admission、抢占或借用；ResourceUnit 只生成 Docker limits |
| Gang Scheduling | ❌ | 不做资源预留或原子启动；多容器 Run 仅按 best-effort 创建，失败时回滚本次创建结果 |
| GPU | ⚠️ | 依赖宿主机 NVIDIA Driver + Container Toolkit；无 GPU Operator |
| KServe / Kubeflow backend | ❌ | Standard 形态保留；Lite 创建时明确返回 unsupported backend |
| 高可用 / 自愈迁移 | ❌ | `restart` 可原地重启容器；主机故障由外部运维恢复 |

Lite 支持的操作保持现有 REST 资源、请求结构、状态枚举和错误格式。不支持的能力返回 `409 CapabilityUnavailable`，Platform 根据 capability API 控制对应入口。

现有 `PodList`、`/pods/{pod}/logs` 和 Pod events API 在 Lite 中返回受管 Docker container 的 Kubernetes Pod 投影。`pod` 参数使用稳定的运行单元名称；phase、容器状态和日志来自 Docker inspect 与 logs；事件来自 Runtime 进程维护的有界内存记录。Compute Service 将 Runtime 返回的 `corev1.Pod` 和 `eventsv1.Event` 映射为现有 REST DTO。

## 3. 整体架构

```text
 Browser / CLI
       │
       ▼
 Traefik (:80 / :443)
       ├── Platform routes ──► axisml-platform
       │                              │
       │                              │ 调用 System API
       │                              ▼
       │                       axisml-core
       │                 ┌─────────────────────────┐
       │                 │ Cluster Manager module  │
       │                 │ Compute Service module  │
       │                 │ Artifact Hub module     │
       │                 │ ─── ComputeRuntime ───  │
       │                 │ Standalone Runtime      │
       │                 └────────────┬────────────┘
       │                              ├──► Docker Engine ──► dynamic workloads
       │                              ├──► Traefik dynamic config
       │                              └──► logs / events
       │
       └── Workload routes ───────────────────────► dynamic workloads

 dynamic workloads
   ├── run containers（短生命周期）
   ├── service/workspace/tensorboard containers（长生命周期）
   └── managed volumes + axisml-workloads network

 PostgreSQL · Redis(optional) · zot · RustFS/S3 · Prometheus(optional)
```

Docker Compose 管理两个 AxisML binary、数据库、缓存、制品存储、网关和可观测组件。`axisml-core` 内置的 Standalone Compute Runtime 根据 API desired state 动态管理用户 workload。

API 调用边界如下：

- 浏览器和 CLI 通过 Traefik 访问 Platform API。
- `axisml-platform` 通过 HTTP 调用 `axisml-core`。
- `axisml-core` 的 Compute Reconciler 通过进程内 `ComputeRuntime` 接口驱动同进程的 Standalone Runtime。
- Traefik 代理 Platform API 和动态 workload 路由。

### 3.1 Standalone 进程模型

Standalone 形态固定编译和部署两个独立可执行文件：

| Binary | 组成 | 权限 |
| --- | --- | --- |
| `axisml-platform` | Platform、认证、RBAC 和 BFF | 对外 HTTP、PostgreSQL、Redis；通过 HTTP 调用 `axisml-core`，**无 Docker socket** |
| `axisml-core` | Cluster Manager、Compute 领域层与 API、Artifact Hub，以及进程内实现 `ComputeRuntime` 的 Standalone Runtime（容器/volume/network/route 管理、运行态采集） | 内部 HTTP（对 `axisml-platform` 服务端）、PostgreSQL、zot、S3、Traefik 动态目录、Docker Engine Adapter；唯一可访问 Docker socket |

Platform 保持独立进程和现有 HTTP client 边界。Cluster Manager、Compute、Artifact Hub 和 Standalone Runtime 组合进 `axisml-core`，并各自维护业务逻辑、数据库 schema 和 migration。Standard 形态继续构建和部署现有独立 binary。

`axisml-core` 负责 Compute 业务状态和 PostgreSQL 持久化，并通过进程内实现 Compute Service 发布的 `computeruntime.ComputeRuntime` 接口驱动 Standalone Runtime。该 Runtime 实现与 Compute 模块同进程，不引入额外网络跳；它把 AxisML workload contract 映射为 Docker、volume、network 和 Traefik 资源，并返回运行状态。Docker socket 仅在 `axisml-core` 内由 Runtime adapter 访问，Compute 领域层与 Artifact Hub 不直接触及 Docker。

### 3.2 代码组织

AxisML Lite 是 AxisML monorepo 的顶层 `axisml-lite/` 目录，与 `axisml-platform/`、`axisml-system/`、`axisml-infra/` 并列。Go module 落在子目录 `axisml-core/`，构建单一 `axisml-core` 镜像，承载 System 模块装配、Standalone Runtime；部署资产与文档在 `axisml-lite/` 层维护：

```text
axisml-lite/
├── axisml-core/                  # Go module（单一 axisml-core 镜像）
│   ├── cmd/
│   │   └── axisml-core/          # Standalone composition root
│   ├── internal/
│   │   ├── core/                 # 模块装配、配置型 provider、PG coordination
│   │   └── runtime/
│   │       └── docker/           # ComputeRuntime 的 Docker 实现：container/volume/network/Traefik adapter、运行态采集（podman/containerd backend 为同级目录）
│   ├── Dockerfile
│   └── go.mod
├── deploy/
│   ├── docker-compose.yaml
│   ├── docker-compose.observability.yaml
│   ├── config/
│   │   ├── resource-pool.yaml
│   │   └── tenant.yaml
│   └── scripts/
├── docs/
│   └── system_design.md          # 本设计文档
└── Makefile
```

`axisml-platform` 使用 monorepo 发布的 Platform 镜像。`axisml-core` 通过仓库内 `go.work` / `replace` 依赖 Cluster Manager、Compute Service 和 Artifact Hub 的 `pkg/module` 公共装配 API，将三者编译进同一个 binary 并注入 Standalone provider；同时依赖 `MLRun` / `MLService` / `MLTrafficPolicy` API packages，在 `internal/runtime/docker` 进程内实现 Compute Service 发布的 `computeruntime.ComputeRuntime` 接口，并注入到 Compute 模块。该实现驱动 Docker Engine Adapter、volume、network 和 Traefik file provider 并采集运行态。Docker Engine Adapter、配置型资源目录和 Compose 资产同由 `axisml-lite/` 维护，构建单一 Dockerfile 和镜像，由 Compose 拉起。

monorepo 为三个 System 服务提供公共装配 API：

```text
axisml-system/<service>/pkg/module
  New(...)
  RegisterRoutes(...)
  Migrate(...)
```

公共装配 API 包含构造器、路由注册、migration 和装配 DTO。handler、repository、Kubernetes adapter 和业务实现保留在各组件 `internal` 中。Standard binary 和 Lite 的 `axisml-core` 共用该 API。

`axisml-core` 在同一个 `:8080` router 注册三组互不冲突的现有业务路由，并保留原 API path；Platform 的现有三个 downstream client 均指向该地址。Standalone composition root 根据已装配模块和部署形态注册 `/api/v1/capabilities`。模块间通过公开 module contract 协作，各自维护私有 repository。

依赖与发布规则：

- `axisml-lite` 作为 monorepo 内的单个 Go module，通过仓库内 `go.work` / `replace` 依赖 System 组件模块（Cluster Manager、Compute Service、Artifact Hub 三个 `pkg/module`，以及 workload contract API packages），与其余组件在同一提交中同步演进。
- Lite release 固定一组兼容的 Platform 镜像、System module 和数据库 schema 版本，不使用浮动 `latest`。
- monorepo 内的公共装配 API 对各形态保持稳定契约。
- OpenAPI DTO、migration 和领域状态机由 System 组件生成和维护。

## 4. 统一 Workload Contract 与双运行时

### 4.1 总体原则

`MLRun`、`MLService` 和 `MLTrafficPolicy` API 类型是 Compute 的统一 desired-state contract。Compute Service 从 PostgreSQL 业务记录构造对应的 AxisML 对象：

- Kubernetes Runtime 将对象写入 apiserver，由 compute-operator 映射为 Job、Deployment、StatefulSet、Service 和 HTTPRoute。
- Standalone Runtime 直接接收同一对象，由 Docker handler 映射为 container、volume、network 和 Traefik 动态配置。

Lite 不修改 `compute-operator`。Standard 形态的 `ComputeRuntime` adapter 位于 Compute Service 内部，封装现有 `client.Client`、informer cache 和 kubeproxy 调用；adapter 仍通过现有 CR 与 `compute-operator` 交互。Standalone 复用已发布的 CR API 类型、默认值和校验规则，不要求 operator 增加 Lite 分支。

**Compute Runtime** 负责执行和观察 AxisML workload contract。Runtime contract 包含 CR Spec、状态枚举、condition、backend key、label 及必要 metadata。

各 binary 在 composition root 装配 provider，组件内部依赖只读资源目录、租户视图和 Runtime 接口。

Runtime 端口只使用 AxisML 发布的 `MLRun`、`MLService`、`MLTrafficPolicy` API 类型和 Kubernetes 标准类型。Docker SDK 请求、Traefik 配置、Deployment、HTTPRoute 和 `client.Object` 属于 handler / adapter 内部类型，不进入 Runtime contract。

核心接口由各组件的 `pkg/extensions` 包发布（compute-service 的 `ComputeRuntime` / `ResourceResolver` / `VolumeManager`，cluster-manager 的 `ResourcePoolStore` / `TenantStore`），如下：

```go
type ResourceStoreProvider interface {
    GetPool(ctx context.Context, name string) (ResourcePool, error)
    ListPools(ctx context.Context, selector string) ([]ResourcePool, error)
}

type TenantStoreProvider interface {
    Get(ctx context.Context, tenantScope string) (Tenant, error)
    List(ctx context.Context) ([]Tenant, error)
}

type ComputeRuntime interface {
    ApplyMLRun(ctx context.Context, desired *mlrunv1alpha1.MLRun) error
    ObserveMLRun(ctx context.Context, key types.NamespacedName) (mlrunv1alpha1.MLRunStatus, error)
    CancelMLRun(ctx context.Context, key types.NamespacedName) error
    DeleteMLRun(ctx context.Context, key types.NamespacedName) error
    ListMLRunInstances(ctx context.Context, key types.NamespacedName) (*corev1.PodList, error)
    GetMLRunInstanceLogs(
        ctx context.Context,
        key types.NamespacedName,
        instance string,
        opts *corev1.PodLogOptions,
    ) (io.ReadCloser, error)
    GetMLRunInstanceEvents(
        ctx context.Context,
        key types.NamespacedName,
        instance string,
    ) (*eventsv1.EventList, error)
    GetMLRunEvents(ctx context.Context, key types.NamespacedName) (*eventsv1.EventList, error)

    ApplyMLService(ctx context.Context, desired *mlservicev1alpha1.MLService) error
    ObserveMLService(ctx context.Context, key types.NamespacedName) (mlservicev1alpha1.MLServiceStatus, error)
    DeleteMLService(ctx context.Context, key types.NamespacedName) error
    ListMLServiceInstances(ctx context.Context, key types.NamespacedName) (*corev1.PodList, error)
    GetMLServiceInstanceLogs(
        ctx context.Context,
        key types.NamespacedName,
        instance string,
        opts *corev1.PodLogOptions,
    ) (io.ReadCloser, error)
    GetMLServiceInstanceEvents(
        ctx context.Context,
        key types.NamespacedName,
        instance string,
    ) (*eventsv1.EventList, error)
    GetMLServiceEvents(ctx context.Context, key types.NamespacedName) (*eventsv1.EventList, error)

    ApplyMLTrafficPolicy(ctx context.Context, desired *mltrafficpolicyv1alpha1.MLTrafficPolicy) error
    ObserveMLTrafficPolicy(ctx context.Context, key types.NamespacedName) (mltrafficpolicyv1alpha1.MLTrafficPolicyStatus, error)
    DeleteMLTrafficPolicy(ctx context.Context, key types.NamespacedName) error
    GetMLTrafficPolicyEvents(ctx context.Context, key types.NamespacedName) (*eventsv1.EventList, error)
}
```

Apply 请求携带空 `.status` 的完整 desired 对象；Runtime 从 CR metadata 获取 namespace 和 name，并与 workload kind 组成资源键。其余操作使用 Kubernetes `types.NamespacedName` 定位 workload。Observe 返回对应 CR Status；底层资源不存在时返回可由 `apierrors.IsNotFound` 识别的 Kubernetes `NotFound` 错误，其他错误表示本次观察失败。

Instance 是 Runtime 对单个运行单元的统一称谓：Kubernetes 实现对应 Pod，Standalone 实现对应 Docker container。Standalone Runtime 将 container 投影为 `corev1.Pod`，并将运行事件合成为 `events.k8s.io/v1.Event`。Instance 名称必须稳定，日志和事件查询必须校验该 instance 属于指定 workload。MLRun / MLService 的资源级事件与 instance 级事件分别由独立方法返回；MLTrafficPolicy 没有 instance，仅提供资源级事件。

Standard 和 Lite 形态均在各自进程内提供完整的 `ComputeRuntime` 实现：

- Standard 形态在 Compute Service 进程内由 `internal/kuberuntime` 直接实现，封装 `client.Client`、informer 和 kubeproxy。
- Lite 形态在 `axisml-core` 进程内由 `internal/runtime/docker` 直接实现，根据 `(backend.name, backend.engine)` 选择 Docker handler，并将 AxisML 对象渲染为内部 `ContainerPlan` / `RoutePlan`。

接口的 Go 类型契约（CR API 类型、`types.NamespacedName` 定位、`apierrors.IsNotFound` 语义、instance 归属校验）在两种形态下保持一致；两种实现均为进程内 Go 调用，不引入网络传输层。

### 4.2 Provider 映射

| 抽象 | Standard 实现 | Lite 实现 |
| --- | --- | --- |
| Resource catalog | `ResourcePool` CR + informer/client | `ConfigResourceCatalog`，从 CR YAML 读取唯一 `default` ResourcePool |
| Tenant view | `Tenant` CR provider | `StaticTenantStoreProvider`，从 CR YAML 读取唯一 `default` Tenant |
| Run contract | `MLRun` → apiserver → compute-operator → Job | `MLRun` → Docker run handler → containers |
| Service contract | `MLService` → apiserver → compute-operator → Deployment/StatefulSet/Service | `MLService` → Docker service handler → containers + runtime registry |
| Traffic contract | `MLTrafficPolicy` → apiserver → compute-operator → HTTPRoute | `MLTrafficPolicy` → Traefik handler → file provider |
| 状态观察 | operator 写 CR Status，Compute informer 映射回 PG | Runtime 返回对应 CR Status，Compute Reconciler 使用同一映射写回 PG |
| Instance view / logs / events | Kubernetes Pod / PodLogOptions / Event | Docker container 投影为 `corev1.Pod`；Docker logs；Runtime 事件投影为 `eventsv1.Event` |
| 长期任务启动 | manager runnable / PG advisory lock | 单副本直接启动 |

两种 Runtime 共享 CR API contract、校验规则、backend key、状态枚举和 condition 约定。

## 5. System 层设计

### 5.1 Cluster Manager

Cluster Manager 的 REST handler 通过 provider 访问资源与租户视图：

- Standard 形态注入 `KubernetesResourceCatalog` 和 `KubernetesTenantStore`。
- Lite 模式注入只读的 `ConfigResourceCatalog` 和 `StaticTenantStoreProvider`。
- `default` ResourcePool、ResourceUnit 和 Tenant 从 AxisML CR YAML 加载，配置变更重启后生效。
- ResourceUnit 的 `requests/limits` 在创建 workload 时写入 Spec 快照。
- ResourcePool / ResourceUnit 的 GET、LIST 可用；CREATE、PATCH、DELETE 返回 `409 CapabilityUnavailable`。
- Tenant GET、LIST 可用；CREATE 对 `default` tenant 幂等，其他名称返回 `409 MultiTenancyUnavailable`；PATCH、DELETE 返回 `409 MultiTenancyUnavailable`。

数据权威：

| 权威 | 数据 |
| --- | --- |
| ResourcePool CR YAML | `default` pool、ResourceUnit 及其资源规格 |
| Tenant CR YAML | `default` tenant identity；`spec.quotas[pool=default].max` 是展示和配置校验用的声明容量，不执行严格 admission |
| Platform `tenants` 表 | `default` tenant 的展示信息、停用状态和用户角色关联 |
| Compute 业务表 | workload `spec` 快照与 `phase` / `status`；Service / TrafficPolicy 另有 `generation` / `observed_generation` |

Platform bootstrap 在 `tenants` 表创建唯一的 `default` 业务记录。Platform 在编排层合并该记录与 `StaticTenantStoreProvider` 返回的静态配置视图。

#### 5.1.1 CR YAML 配置格式

Lite 使用 AxisML `axisml.io/v1alpha1` ResourcePool 和 Tenant CR YAML。部署目录包含：

```text
config/
├── resource-pool.yaml
└── tenant.yaml
```

`resource-pool.yaml`：

```yaml
apiVersion: axisml.io/v1alpha1
kind: ResourcePool
metadata:
  name: default
spec:
  units:
    - name: cpu-small
      requests:
        cpu: "1"
        memory: 2Gi
      limits:
        cpu: "1"
        memory: 2Gi
    - name: gpu-1x
      requests:
        cpu: "4"
        memory: 16Gi
        nvidia.com/gpu: "1"
      limits:
        cpu: "4"
        memory: 16Gi
        nvidia.com/gpu: "1"
```

`tenant.yaml`：

```yaml
apiVersion: axisml.io/v1alpha1
kind: Tenant
metadata:
  name: default
spec:
  namespace:
    name: default
  quotas:
    - pool: default
      name: default
      min:
        cpu: "0"
        memory: "0"
      max:
        cpu: "16"
        memory: 64Gi
        nvidia.com/gpu: "1"
```

加载规则：

- 使用 AxisML Go API types 和 Kubernetes YAML decoder。
- 必须且只能存在一个 `ResourcePool` 和一个 `Tenant`；两者的 `metadata.name` 均固定为 `default`。
- `Tenant.spec.quotas` 必须且只能引用 `default` pool；`max` 用于配置校验和容量展示。
- `ResourcePool.spec.units[*].requests/limits` 不得超过对应 quota `max`，名称必须唯一。
- `ResourcePool.spec.nodeSelector`、`spec.tolerations` 和 unit `nodeSelector` 必须为空。
- `Tenant.spec.namespace.name` 固定为 `default`。
- `Tenant.spec.initResources` 必须为空；Standalone workload 的 volume 和凭证由 Runtime Controller 按 workload 需要创建。
- 配置不得包含 `status`、`metadata.uid`、`resourceVersion`、`generation` 或 `managedFields` 等 apiserver 生成字段。
- 任一对象校验失败时 `axisml-core` 保持 not ready。

`ConfigResourceCatalog` 和 `StaticTenantStoreProvider` 持有启动时解析出的不可变快照。配置变更需要重启 `axisml-core` 并重新执行跨对象校验；已持久化的 workload Spec 保留原 ResourceUnit 快照。

### 5.2 Compute Service

PostgreSQL 中的 `mlruns`、`mlservices` 和 `traffic_policies` 是 Compute 业务权威。`spec jsonb` 保存对应 CR `.spec` 的规范化 JSON。Compute 补齐 name、tenant namespace、现有 AxisML ID label 和其他 metadata，生成 desired `MLRun`、`MLService` 或 `MLTrafficPolicy`。这些 ID labels 为 Standard 形态和现有 CR contract 保留，Standalone Runtime 不使用它们标识底层资源。Run Spec 创建后不可变；Service 和 TrafficPolicy 使用 `generation` / `observed_generation`。

Standalone Compute 流程：

```text
API transaction
  └── axisml-core 校验业务规则与 ResourceUnit
        └── 写入 desired Spec；Service / TrafficPolicy 变更时推进 generation

Compute reconciliation
  └── axisml-core 单实例运行 Compute Reconciler
        ├── 从 PG 构造 MLRun / MLService / MLTrafficPolicy
        ├── 执行业务状态迁移、取消/删除/scale 规则
        ├── 调用 Runtime API apply / cancel / delete / observe
        └── 更新 status / phase；Service / TrafficPolicy 更新 observed_generation

Runtime convergence
  └── Standalone Runtime 校验 installation ID、CR metadata 与 backend capability
        ├── Docker handler 将 CR contract 渲染为 ContainerPlan / RoutePlan
        ├── Docker / volume / network / route adapter 幂等收敛
        └── 返回 MLRunStatus / MLServiceStatus / MLTrafficPolicyStatus，不读写 PostgreSQL
```

Compute Reconciler 是 Compute 业务表中 `phase`、`status` 和 `observed_generation` 的唯一写入者。Runtime Controller 收敛 Docker、volume、network 和 route，并生成对应 CR Status。

每个 Standalone handler 提供 capability validator。`axisml-core` 的 Compute 模块在持久化前校验请求，进程内 Standalone Runtime 在 apply 时再次校验。字段映射规则如下：

- `image`、`command/args`、普通 `env`、working directory、ports、resource limits 和 restart policy 映射为 Docker 等价配置。
- `requests` 保留在 contract 和状态展示中；`limits` 映射为 cgroup / DeviceRequest。
- `PriorityClass`、`NodeSelector`、`Tolerations` 在 Lite 配置校验阶段要求为空。
- `EnvFrom`、Secret/ConfigMap 引用、PVC/volume source 和 backend `config` 按 handler 支持矩阵校验；不支持的字段返回 `409 CapabilityUnavailable`。
- backend capability 以 `(backend.name, backend.engine)` 为粒度。

Standalone Compute 路径使用现有 Compute schema：

| 现有字段 | Standalone 语义 |
| --- | --- |
| `spec jsonb` | 对应 CR `.spec` 的规范化 JSON，包含 ResourceUnit 展开后的资源快照 |
| `generation` | 仅 Service / TrafficPolicy 使用的 desired state 版本；Run 不增加该字段 |
| `observed_generation` | 仅 Service / TrafficPolicy 使用；Compute Reconciler 确认 Runtime 已接受并成功 apply 的 desired spec 版本，不表示 workload 已 Ready |
| `phase` | 对外高频状态 |
| `status jsonb` | message、conditions、readyReplicas、endpoint、startedAt / finishedAt 等标准化状态 |
| `id uuid` | 现有 Compute 主键；继续写入既有 CR ID label，但不进入 Standalone Runtime 资源身份 |

Standalone Runtime 使用 `(kind, namespace, name)` 定位 Docker 和 Traefik 资源。Compute 写入的 `status.observedAt` 记录最近观察时间。Runtime 将操作结果和 inspect 状态变化写入有界内存事件环，供 events API 查询。

所有 apply/delete 操作必须幂等。Docker 资源使用以下稳定 label 定位：

- `io.axisml.managed=true`
- `io.axisml.resource-kind=run|service|traffic-policy`
- `io.axisml.resource-namespace=<workload namespace>`
- `io.axisml.resource-name=<workload name>`
- `io.axisml.replica-index=<n>`
- `io.axisml.tenant=default`
- `io.axisml.instance-id=<stable-installation-id>`

Compute Reconciler 逐条观察 PG 中的活动记录。Observe 返回 Kubernetes `NotFound` 时，活动记录重新 apply，删除中记录完成删除；其他观察错误保留最近一次 `phase/status` 并等待重试。Observe 成功时写回当前 `phase/status`。Service / TrafficPolicy 的 apply 成功后推进 `observed_generation`；实际运行状态继续由后续 Observe 结果表达。运维工具按受管 label 列出不存在对应活动 PG 记录的 `(kind, namespace, name)`，由管理员确认清理。Runtime 仅删除资源键、managed label 和 installation ID 均匹配的资源。installation ID 持久化在 Compose 管理的 root-only 配置目录中。

### 5.3 Runtime Handler

AxisML 主仓库维护 `MLRun`、`MLService`、`MLTrafficPolicy` API packages，以及共享的默认值、不可变字段检查和 Spec 校验。

| Handler | 输入 | 输出 |
| --- | --- | --- |
| Kubernetes | CR contract | Job / Deployment / StatefulSet / Service / HTTPRoute |
| Standalone | CR contract | `ContainerPlan` / `RoutePlan` |

Kubernetes handler 继续由现有 `compute-operator` 原样执行；Standalone handler 由 `axisml-core` 内置的 Standalone Runtime 执行。backend 通过 `(backend.name, backend.engine)` 声明 Runtime 支持范围。双 Runtime 改造仅重构 Compute Service 的调用边界和装配方式，不修改 operator 的 reconciler、handler 或底层资源映射。

### 5.4 Artifact Hub

Artifact Hub 使用 PostgreSQL、OCI 和 S3：

- GC worker 提供可取消启动入口 `Start(ctx) error`。
- Standard binary 使用 PG session 级 advisory lock（`pg_try_advisory_lock`）选主；Lite 在单副本 `axisml-core` 中直接运行。Artifact Hub 的两种构建均只依赖 PostgreSQL 完成单活控制。
- zot 和 S3 分别配置服务间地址与浏览器/CLI 地址。
- workload 拉取容器镜像由 Standalone Compute Runtime 通过 Docker Engine Adapter 执行；模型 / 数据集通过 init container 等价步骤或受管 volume 下载。
- Standalone Compute Runtime 从受控 secret 目录读取凭证，并只向目标 workload 注入最小范围的凭证。

### 5.5 Capability 与 Platform

System capability 由 composition root 根据已装配模块构造。每个部署入口注册一个 capability endpoint：

- Kubernetes：各独立 System binary 的 composition root 返回该服务的静态能力；Platform 调用已配置的 System capability endpoints 并合并结果。
- Standalone：`axisml-core` 返回 Cluster Manager、Compute 和 Artifact Hub 的完整静态能力。

Platform 对外提供 `GET /api/v1/capabilities`：

- Kubernetes：Platform 合并各 System 服务返回的组件能力。
- Standalone：Platform 调用单个 `axisml-core` capability endpoint 并转发其结果。
- 必要 System capability 无法读取时返回 `503 UpstreamUnavailable`。

响应按组件组织：

```json
{
  "components": {
    "cluster-manager": {
      "multiTenant": false,
      "resourcePools": "single-default",
      "distributedScheduling": false
    },
    "compute-service": {
      "runtime": "standalone",
      "backends": ["native/job", "native/deployment", "native/statefulset", "native/httproute"],
      "gpu": true
    },
    "artifact-hub": {
      "kinds": ["model", "image", "dataset"]
    }
  }
}
```

Capability 表达版本和部署形态决定的静态支持范围；readiness、status 和 condition 表达服务、operator、Docker daemon 与存储的瞬时健康状态。Platform 的功能开关以 System 返回的 capability 为准。

## 6. Standalone Compute Runtime 详细设计

### 6.1 运行单元

Standalone Runtime 接收带必要 AxisML metadata 的 `MLRun`、`MLService` 或 `MLTrafficPolicy`。Runtime 根据 `(kind, backend.name, backend.engine)` 选择 handler，依次执行：

1. 校验 CR contract、Lite capability 和字段支持范围。
2. 规范化默认值并校验 metadata 中的 namespace、name 和 tenant。
3. 将对象渲染为内部 `ContainerPlan` / `RoutePlan`。
4. 调用 Docker / volume / network / route adapter 幂等收敛。
5. 将 inspect 结果归一化为对应的 `MLRunStatus`、`MLServiceStatus` 或 `MLTrafficPolicyStatus`。

`ContainerPlan` 至少包含 image、command/args、env、mounts、ports、resources、healthcheck、restart policy、GPU request 和 AxisML labels。它是 Standalone Runtime 内部的 adapter boundary 和单元测试对象，由 Docker Engine Adapter 翻译为 Docker API 请求。

| AxisML 对象 | Docker 映射 | 重启策略 |
| --- | --- | --- |
| Run | 单个或同机多角色 containers | `no`；由 Runtime Controller 采集退出状态 |
| Service | 每副本一个 container | `unless-stopped` |
| Workspace | 一个 container + 独立 volume | `unless-stopped` |
| TensorBoard | 一个临时 container + 只读日志挂载 | `unless-stopped`，另有空闲 TTL |

`axisml-core` 校验 ResourceUnit，并将展开结果写入 `MLRunSpec`。Docker run handler 按 pull image → 创建全部容器 → 启动的顺序执行；失败时回滚本次创建的容器并返回失败 `MLRunStatus`。多角色任务采用本机 best-effort 创建和失败回滚。

### 6.2 网络与服务发现

- 固定服务加入 `axisml-control` network。
- 动态 workload 加入 `axisml-workloads` network；固定服务中只有 Traefik 同时加入该网络。
- `axisml-core` 加入 `axisml-control` network，通过 Docker Engine API 管理和观察 workload。
- 容器名称由 `(kind, namespace, name, replica)` 生成；超长或含非法字符时使用规范化名称加资源键短 hash。资源键 labels 是权威标识。
- Service 副本由 Traefik 直接负载均衡。
- workload 仅加入 `axisml-workloads` network。

### 6.3 流量路由

Traefik 使用 file provider：

- Standalone Compute Runtime 为 Service / Workspace / TensorBoard 生成 router 和 backend 配置，并根据 TrafficPolicy 使用 weighted service 表达 weighted / canary / bluegreen。
- 配置先完成语法和引用校验，再写临时文件并通过 `fsync + rename` 原子替换。
- Runtime 通过控制网络中的只读状态接口确认目标配置已加载；加载失败或超时写入对应 CR Status condition。
- 路由名和 service 名由 `(kind, namespace, name)` 生成；必要时附加资源键短 hash。

路由目录仅供 Traefik 读取。Lite 入口支持 HTTP/HTTPS；Gateway API `SecurityPolicy` 和 `BackendTrafficPolicy` 标记为 unsupported capability。

### 6.4 存储

- Platform / System 元数据：PostgreSQL volume。
- Workspace 数据：每个 workspace 独立 named volume，删除策略与现有 Workspace 语义一致。
- Run 输出 / TensorBoard 日志：S3 前缀，与 Kubernetes 版路径约定一致。
- Model / Image：zot；Dataset：S3。
- 临时文件：`/var/lib/axisml/runtime` 受管目录。

所有持久目录支持独立备份。业务对象进入 `Deleted` 且保留策略允许时，Runtime 删除对应 volume / prefix。

### 6.5 资源、配额与 GPU

Lite 的单机资源模型如下：

1. `Tenant.spec.quotas[pool=default].max` 表示管理员声明的单机容量，用于配置校验和 UI 展示。
2. 创建 workload 时按 ResourceUnit 展开并快照 `requests/limits`。
3. Runtime 将 CPU / memory limits 翻译为 Docker cgroup 限制；`requests` 用于规格展示。
4. Docker 创建失败、进程 OOM 或 GPU 不足映射为 CR Status 中的 `Failed` / `Degraded` condition。
5. 宿主机容量和 workload 用量由外部监控观察。

Lite 的资源能力范围不包含抢占、借用、优先级、严格公平性和 gang scheduling。

GPU 使用 Docker DeviceRequest，默认按整卡计数。宿主机预先安装 NVIDIA Driver 和 Container Toolkit。

### 6.6 状态映射

Standalone Runtime 返回 CR Status，Compute Reconciler 通过共享的 CR Status → PG 映射推进业务状态：

| Docker 状态 / 事件 | `MLRunStatus` | `MLServiceStatus` |
| --- | --- | --- |
| image pulling / created | `Pending` | `Pending` |
| running | `Running` | health 未通过为 `Pending` |
| running + health healthy | — | `Ready` |
| 部分副本健康 | — | `Degraded` |
| exit code 0 | `Succeeded` | `Failed` |
| exit code 非 0 / OOMKilled | `Failed` | `Failed` |
| 用户 cancel 后停止 | phase 保持非终态，增加 `Suspended=True, reason=CancelRequested`；Compute 映射为 PG `Cancelled` | — |
| runtime 不可达 | 保留最近 phase，增加 `RuntimeUnavailable` condition | 同左 |

`MLTrafficPolicyStatus` 由 Traefik 已加载配置和成员 Service readiness 生成，包含 `Pending` / `Ready` / `Degraded` / `Failed` phase、`backends[*]` 和 endpoint。

周期性 inspect 驱动状态收敛。Runtime 将 apply / cancel / delete 结果和 inspect 状态变化写入固定容量的内存事件环；events API 返回当前 Runtime 进程生命周期内的近期事件。

## 7. 可靠性与安全

### 7.1 一致性

- PostgreSQL 的现有业务表是权威，Docker / Traefik 是派生运行态。
- API 写请求只承诺 desired state 已持久化；Compute Reconciler 在 Runtime 成功 apply Service / TrafficPolicy 后推进 `observed_generation`，再通过 Observe 结果持续更新实际 `phase/status`。
- Docker Compose 固定 `axisml-core` 的 `replicas=1`。
- Runtime API 的所有变更操作必须幂等，以覆盖 Runtime 调用成功但 PG 状态尚未提交时的进程崩溃。
- 每次 apply 记录 spec hash；hash 未变化时不得无意义重建容器。

### 7.2 Docker socket

Docker socket 等价于宿主机 root 权限。只有 `axisml-core` 可通过受限的 Docker socket proxy 访问必需 API，且仅其内部 Standalone Runtime adapter 使用；`axisml-platform` 和用户 workload 均不可挂载。`axisml-core` 同时持有 PostgreSQL 凭证和 Docker socket，因此进程内必须将 Runtime adapter 限定为唯一访问 Docker 的边界，并通过 socket proxy 收敛可调用的 Docker API。AxisML Lite 使用独占主机部署。

其他要求：

- 禁止 privileged、host PID/IPC、任意 hostPath 和额外 Linux capabilities。
- 可挂载路径必须位于 AxisML allowlist。
- 镜像默认按 digest 运行；tag 在创建时解析并快照 digest。
- 动态容器加入专用 network，不发布随机宿主机端口，统一经 Traefik 暴露。
- 生产配置中的密码和私钥放在 root-only secret 文件，不写入 Compose YAML。

## 8. 数据兼容与迁移

Kubernetes 与 Lite 共用业务 schema 和 API contract。迁移范围仅包含以下可移植持久化数据：

- PostgreSQL 中的用户、租户展示信息、RBAC、Job / Experiment 定义和 Artifact 元数据。
- S3 中的 Dataset、Run 输出和日志等对象数据。
- zot 中的镜像与模型制品。

运行态排除在迁移范围外：

- Run、Service、Workspace、TensorBoard 和 TrafficPolicy 的运行实例及 desired / observed state。
- Kubernetes CR、Pod、PVC、Service、HTTPRoute 等运行时资源。
- Docker container、network、named volume 和 Traefik 动态配置。
- Workspace 本地 volume 中未同步到对象存储的数据。

Job、Service、Workspace、TensorBoard 和 TrafficPolicy 在目标环境通过 API 重新创建。

- `axisml-platform` 只执行 Platform migration；`axisml-core` 按 Compute → Artifact Hub 的固定顺序执行各模块 migration。
- schema migration 使用 `golang-migrate` PostgreSQL driver 自带的 advisory lock 进行并发互斥。
- `axisml-core` 内置的 Standalone Runtime 只通过 Docker / Traefik adapter 操作运行态，不参与 schema migration。
- Cluster Manager 的 `default` ResourcePool、ResourceUnit 和静态 tenant view 由配置文件提供。
- Standalone 复用 `id`、`spec`、`phase` 和 `status` 字段，Service / TrafficPolicy 复用 `generation` / `observed_generation`。
- Platform `tenants.kubernetes_namespace` 允许 `NULL`；Standard 构建在应用层校验非空。
- 迁移使用按领域表和对象前缀定义的逻辑导出 / 导入。
- 已终态 Run 的历史记录可导出到归档或分析存储，但不导入目标环境的活动 Compute 表。

迁移流程为：停止业务写入，导出可移植 PostgreSQL 元数据，复制 S3 / zot 数据，在目标环境导入元数据并校验对象引用。

## 9. 实施阶段

实施按以下工程依赖顺序推进：

1. **Contract 准备**：在 Compute Service 公共包中发布 Runtime contract，包括 CR apply / observe、MLRun / MLService instance list / logs / events、资源级 events、共享默认值、不可变字段检查、Spec 校验和 Status 映射；不修改 `compute-operator`。
2. **公共装配面与 Kubernetes adapter**：为 Cluster Manager、Compute Service 和 Artifact Hub 提供 `pkg/module` API；在 Compute Service 内以 adapter 封装现有 `client.Client`、informer 和 kubeproxy，并接入现有 Standard binary。既有 `compute-operator` 保持不变。
3. **Lite 基础控制面**：创建 `axisml-lite` 目录，构建内置 Standalone Runtime 的 `axisml-core`，接入 `axisml-platform`、Compose、PostgreSQL、zot、S3 和 Traefik。
4. **Run 能力**：Docker job、Kubernetes Pod / Event 投影、日志、取消、状态收敛和 CPU/memory/GPU limits。
5. **Service 能力**：deployment/stateful workload、volume、health、scale、路由。
6. **流量与制品闭环**：TrafficPolicy、模型/数据集注入、Workspace/TensorBoard。
7. **运维完善**：备份恢复、升级、未关联资源检查与人工清理工具、Prometheus dashboard、安全加固。

每一阶段运行 Kubernetes provider 的既有单元 / 集成测试。两种 Runtime 使用同一组 CR contract fixtures 验证默认值、不可变字段、支持矩阵和 Status 映射；各 handler 单独验证底层资源。

## 10. 设计摘要

| 决策项 | 决策 |
| --- | --- |
| 代码组织 | AxisML monorepo 顶层 `axisml-lite/` 单个 Go module 与 Dockerfile，经仓库内 `go.work` / `replace` 依赖公共装配 API 与 workload contract，构建单一 `axisml-core` 镜像 |
| 部署入口 | Docker Compose 管固定服务，`axisml-core` 内置的 Standalone Compute Runtime 动态管用户 workload |
| Standalone binary | `axisml-platform` + `axisml-core`；`axisml-core` 装配三个 System 服务并进程内实现 `ComputeRuntime` 的 Standalone Runtime，独占 Docker socket |
| 运行时 | Docker Engine API |
| Workload contract | `MLRun` / `MLService` / `MLTrafficPolicy` API 类型 |
| 兼容策略 | 共享 CR Spec、backend key、状态和 condition；环境差异由 capability 和 validation 表达 |
| Capability | System composition root 声明部署形态能力；Standalone 由 `axisml-core` 直接返回完整能力；Platform 统一对外 |
| 多租户 | 配置定义唯一 `default` tenant；PG 保存其 Platform 业务记录 |
| 资源池 | 配置定义只读 `default` pool 和 ResourceUnit |
| 调度 | ResourceUnit 快照、Docker limits 和 best-effort 运行 |
| 网关 | Traefik file provider |
| 状态权威 | PG 为业务权威；Docker / Traefik 为可重建派生态 |
| K8s 兼容 | Standard 形态保留现有 CRD / operator 路径；`compute-operator` 不修改，Compute Service adapter 封装现有 Kubernetes 调用；两种 Runtime 共享 CR API contract 和纯校验 |
| 可恢复性 | 持久卷、幂等 reconcile 和备份 |

## 11. 关联文档

- [AxisML 高层设计](../../docs/high_level_design.md)
- [System 层概要](../../axisml-system/docs/system_design/overview.md)
- [Compute Service](../../axisml-system/docs/system_design/compute-service.md)
- [Compute Operator](../../axisml-system/docs/system_design/compute-operator.md)
- [Cluster Manager](../../axisml-system/docs/system_design/cluster-manager.md)
- [Artifact Hub](../../axisml-system/docs/system_design/artifact-hub.md)
