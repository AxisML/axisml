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

可选 `storage` profile 启动 RustFS，`gateway` profile 启动 Traefik。所有镜像
使用同一个 `AXISML_VERSION`，默认由根 Makefile 的 `IMAGE_TAG` 注入。

## 3. 装配边界

Standalone 代码属于 `github.com/axisml/axisml/axisml-standalone`，并依赖
`github.com/axisml/axisml/axisml-system` 暴露的三个服务模块。三个服务通过各自
`pkg/module` 暴露构造、路由、migration 和后台任务；repository 与 handler 仍
留在 System 组件内部。根 `standalone` package 是 composition root：

- `New(ctx, Config, ...Option)` 装配数据库、静态资源目录、Docker runtime 和模块；
- `App.Migrate()` 执行 Compute Service 与 Artifact Hub migration；
- `App.Handler()` 返回完整 `http.Handler`；`RegisterRoutes` 支持 Gin 宿主；
- `App.Serve(ctx)` 或 `App.Runnables()` 负责后台 reconcile / status / GC 生命周期；
- `App.Close()` 释放由 `New` 创建的数据库和 Docker client。

发布模块的 `go.mod` 不包含 `replace`。仓库内的 `go.work` 只负责把 APIs、
Cluster Manager、Compute Service 与 Artifact Hub 解析到相邻源码。公共 module
按依赖顺序发布：`axisml-system/apis/vX.Y.Z` → 三个服务各自的
`axisml-system/<service>/vX.Y.Z` → `axisml-standalone/vX.Y.Z`；版本号保持一致。

`Serve` 与 `Runnables` 只能认领后台任务一次。宿主必须在关闭数据库和 Docker
client 前取消任务上下文并等待任务退出。

## 4. 资源目录与能力

ResourcePool 和 Tenant 从 `/etc/axisml/pools/resourcepools/*.yaml` 与
`/etc/axisml/pools/tenants/*.yaml` 加载，启动时完整校验并形成只读 provider。
Standalone 不提供租户和资源池写入，不执行 ElasticQuota，也不提供 Prometheus
指标查询。`/api/v1/capabilities` 返回三个模块的聚合能力文档，Platform 和测试
据此隐藏或跳过不可用操作。

预定义 volume 在启动时创建。普通 volume 映射为受管 Docker volume；带
`hostPath` 的 volume 映射为宿主绝对路径，只允许在 standalone 使用。

## 5. Docker runtime

Compute Service 仍以 PostgreSQL 为 workload desired-state 权威。reconciler 把
`MLRun`、`MLService` 和 `MLTrafficPolicy` 交给进程内 Docker adapter：

- MLRun 先停留在 `Queued`；Compute 读取 Docker Engine CPU/内存、受管 GPU、
  活跃 container 预留和静态 Tenant pool quota，admission 成功后才调用 adapter，
  因此排队期不创建 container；
- `(native, job)` 映射为一次性 container；
- `(native, deployment|statefulset)` 映射为受管 container 集合；
- 流量策略写入 Traefik 动态配置；
- container、volume 和 network 使用 `io.axisml.managed=true` 等 label 标记归属；
- observe、日志和事件投影回与 Kubernetes runtime 相同的公共 DTO 与状态枚举。

adapter 只依赖公共 AxisML API 类型和 Kubernetes 标准投影类型。Docker SDK、
container plan 与 Traefik 配置不进入服务接口或数据库。

## 6. 配置与持久化

Standalone 只读取 `AXISML_` 环境变量；秘密字段同时支持 `_FILE`。端口、目录、
Docker network 与后台周期是 `DefaultSettings`，嵌入宿主可通过 `WithSettings`
覆盖。PostgreSQL、zot 数据、Traefik 配置与 workload ConfigMap 投影分别使用
Compose volume 持久化。

队列容量从 Docker host 总量扣除 `workload.system_reserved_cpu`（默认 `0`）和
`workload.system_reserved_memory`（默认 `0`），用于保留 OS 与控制面开销。生产部署应按宿主机
实际开销显式设置这两个值。GPU 只有在
`gpu.devices` 显式列出受管设备时才进入可调度容量；未配置时 GPU Run 保持排队。

## 7. 合同与验收

聚合 OpenAPI 由 `axisml-standalone/cmd/openapi-gen` 从三个组件生成规格折叠而成，并
同时写入 `axisml-standalone/docs/apis/standalone.yaml` 与运行时 embed 文件。
`make docs-test` 检查两份产物无漂移。黑盒套件通过
`pytest --mode kubernetes|standalone` 运行同一批测试，形态差异只由 capability
matrix 和显式 `<mode>_only` marker 表达。
