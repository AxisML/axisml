# AxisML Standalone

## 1. 范围

Standalone 是 AxisML 的单 Docker host 部署形态，不是独立产品。它复用
Platform、System HTTP API、数据库 schema、migration、领域状态机和 workload
contract；与 Kubernetes 形态的差异只位于资源目录、运行时 adapter 和部署资产。

## 2. 拓扑

`axisml-standalone/compose.yaml` 启动 PostgreSQL、`axisml-standalone`、Platform
和 zot。System 进程在同一 `:8080` HTTP server 上装配 Cluster Manager、Compute
Service 与 Artifact Hub；Platform 的三个 downstream endpoint 均指向该地址。
System 进程是唯一挂载 Docker socket 的容器。

可选 `storage` profile 启动 RustFS，`gateway` profile 以 standalone/file-provider
模式启动 Envoy Gateway。Envoy Gateway 版本从 Kubernetes Infra chart 的
`gateway-helm` 依赖读取，保证两种部署形态使用同一版本；其他镜像使用同一个
`AXISML_VERSION`，默认由根 Makefile 的 `IMAGE_TAG` 注入。

Envoy Gateway standalone 模式当前仍是上游 experimental 功能；该 profile 适合
开发、评估和受控的单机环境，生产采用前需完成实际流量与故障恢复验证。

## 3. 装配边界

Standalone 代码属于 `github.com/axisml/axisml/axisml-standalone`，并依赖
`github.com/axisml/axisml/axisml-system` 暴露的三个服务模块。三个服务通过各自
`pkg/module` 暴露构造、路由、migration 和后台任务；repository 与 handler 仍
留在 System 组件内部。根 `standalone` package 是 composition root：

- `New(ctx, Config, ...Option)` 装配数据库、资源目录、Docker runtime 和模块；
- `App.Migrate()` 执行 standalone、Compute Service 与 Artifact Hub migration，并导入 ResourcePool/Tenant 初始种子；
- `App.Handler()` 返回完整 `http.Handler`；`RegisterRoutes` 支持 Gin 宿主；
- `App.Serve(ctx)` 或 `App.Runnables()` 负责后台 reconcile / status / GC 生命周期；
- `App.Close()` 释放由 `New` 创建的数据库和 Docker client。

发布模块的 `go.mod` 不包含 `replace`。仓库内的 `go.work` 只负责把 APIs、
Cluster Manager、Compute Service 与 Artifact Hub 解析到相邻源码。公共 module
按依赖顺序发布：`axisml-system/apis/vX.Y.Z` → 三个服务各自的
`axisml-system/<service>/vX.Y.Z` → `axisml-standalone/vX.Y.Z`；版本号保持一致。

`Serve` 与 `Runnables` 只能认领后台任务一次。宿主必须在关闭数据库和 Docker
client 前取消任务上下文并等待任务退出。

## 4. 资源目录与公共合同

`/etc/axisml/pools/resourcepools/*.yaml` 和 `/etc/axisml/pools/tenants/*.yaml`
只作为首次启动种子导入 PostgreSQL。之后 ResourcePool、内嵌 ResourceUnit 和 Tenant
分别通过与 Kubernetes 形态相同的 `/api/v1/resourcepools`、`/api/v1/tenants` CRUD 和
`/api/v1/tenants/{tenant}/quotas` 配额接口维护，API 删除不会因种子文件仍在而在重启
后复活。Compute 创建负载时直接读取持久化 ResourcePool/unit，队列读取持久化
Tenant 的 pool quota，对 MLRun 执行跨运行时的算力准入。Tenant、quota、
ResourcePool、ResourceUnit、Artifact 和 MLRun 队列/优先级使用与 Kubernetes 形态
相同的公共 API，不再通过静态 capability 文档区分。

种子 YAML 使用严格字段解码；未知或拼错的键会让启动失败。字符串中的 `${VAR}`
在解码前从进程环境展开，变量未设置或值为空时错误会直接指出变量名；裸 `$VAR`
保持字面值。这样同一份 Tenant 配置可以按宿主设置 `hostPath`，同时避免未配置变量
静默落到文件系统根目录。嵌入宿主可以调用 `LoadStaticConfigWithOptions` 并通过
`LoadStaticConfigOptions.LookupEnv` 注入受控环境；原有 `LoadStaticConfig` 默认使用
`os.LookupEnv`。

Standalone 的部署形态差异限定在运行时语义：

- MLRun 只支持 `(native, job)`；MLService 只支持
  `(native, deployment|statefulset)`；
- MLService route 的 auth 与 rate limit 不受支持；
- volume 支持创建、查询与删除，但不支持扩容；
- ResourceUnit requests/limits 与 Tenant quota min/max 仅支持 `cpu`、`memory`、
  `nvidia.com/gpu`；ResourcePool capacity 使用同一资源集合，非空时覆盖 Docker
  宿主 inventory 的准入容量；GPU 数量必须是整数。启动配置与 REST create/patch 使用同一校验；
- 不提供 Prometheus workload / ResourcePool 指标查询；
- 不执行 Kubernetes scheduler 的 ElasticQuota 准入。Tenant pool quota 由 Compute
  的统一 admission 执行：先增量准入 MLService，再准入 MLRun。

请求触发不受支持的运行时语义时，API 返回稳定的 `CapabilityUnavailable`（HTTP
409），由具体操作给出原因；客户端无需预先读取全局能力矩阵。

预定义 volume 在启动时创建。普通 volume 映射为受管 Docker volume；带
`hostPath` 的 volume 映射为宿主绝对路径，只允许在 standalone 使用。

## 5. Docker runtime

Compute Service 仍以 PostgreSQL 为 workload desired-state 权威。reconciler 把
`MLRun`、`MLService` 和 `MLTrafficPolicy` 交给进程内 Docker adapter：

- MLRun 先停留在 `Queued`；Compute 读取 Docker Engine CPU/内存、受管 GPU、
  活跃 container 预留和静态 Tenant pool quota，admission 成功后才调用 adapter，
  因此排队期不创建 container；
- MLService 同样先以 `Queued` 持久化；至少一个服务副本可放置后才创建 container，
  后续只把 admitted 副本增量提交到 adapter；
- `(native, job)` 映射为一次性 container；
- `(native, deployment|statefulset)` 映射为受管 container 集合；
- 服务路由与流量策略写入 Envoy Gateway `Backend` / `HTTPRoute` 资源；
- container、volume 和 network 使用 `io.axisml.managed=true` 等 label 标记归属；
- observe、日志和事件投影回与 Kubernetes runtime 相同的公共 DTO 与状态枚举。

adapter 只依赖公共 AxisML API 类型和 Kubernetes 标准投影类型。Docker SDK、
container plan 与 Envoy Gateway 文件资源不进入服务接口或数据库。

## 6. 配置与持久化

Standalone 只读取 `AXISML_` 环境变量；秘密字段同时支持 `_FILE`。端口、目录、
Docker network 与后台周期是 `DefaultSettings`，嵌入宿主可通过 `WithSettings`
覆盖。Tenant desired state 与 workload/artifact 元数据一并持久化到 PostgreSQL；
zot 数据、Envoy Gateway 资源与 workload ConfigMap 投影分别使用 Compose volume
持久化。动态 workload endpoint 使用 `<container>.axisml.local` Docker network
alias，供 Envoy Gateway `Backend` 的 FQDN endpoint 解析。

队列容量从 Docker host 总量扣除 `workload.system_reserved_cpu`（默认 `0`）和
`workload.system_reserved_memory`（默认 `0`），用于保留 OS 与控制面开销。生产部署应按宿主机
实际开销显式设置这两个值。GPU 只有在
`gpu.devices` 显式列出受管设备时才进入可调度容量；未配置时 GPU Run 保持排队。

## 7. 合同与验收

聚合 OpenAPI 由 `axisml-standalone/cmd/openapi-gen` 从三个组件生成规格折叠而成，并
同时写入 `axisml-standalone/docs/apis/standalone.yaml` 与运行时 embed 文件。
`make docs-test` 检查两份产物无漂移。黑盒套件通过
`pytest --mode kubernetes|standalone` 运行同一批公共合同测试；只有上述真实运行时
边界使用显式 `<mode>_only` marker 表达。
