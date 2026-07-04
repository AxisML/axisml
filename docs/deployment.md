# AxisML 部署手册

本手册介绍如何把 AxisML 部署到 Kubernetes 集群：前置条件、安装步骤、配置项、验证、升级与卸载，以及组件部署形态参考与常见问题。系统的整体架构与设计取舍见 [high_level_design.md](high_level_design.md)；本地开发环境搭建见 [development_workflow.md](development_workflow.md)。

AxisML 按 Platform / System / Infra 三层各打包为一个 Helm chart，按 **infra → system → platform** 顺序安装、反向卸载。

## 1. 前置条件

| 工具 | 用途 | 备注 |
| --- | --- | --- |
| Kubernetes 集群 | 运行目标 | 本地用 minikube（profile `axisml`）；生产用任意发行版 |
| `kubectl` | 集群操作 + CRD apply | 与集群版本匹配 |
| Helm 3 | chart 安装 | 三层 chart 均经 Helm |
| Docker | 构建 / 本地镜像、集成测试 | 本地开发需要 |
| Go 1.26+ | 从源码构建镜像 | 仅自行构建镜像时需要 |

集群需满足：支持 `ReadWriteOnce` PVC（PostgreSQL / Redis / RustFS / zot 落盘）；如需 GPU，节点装好 NVIDIA 驱动栈所需内核头（GPU Operator 负责其余）。最小可跑（无 GPU）即可在 minikube 上完成全栈安装。

## 2. 部署形态总览

| Chart | 路径 | Release | Namespace | 内容 |
| --- | --- | --- | --- | --- |
| `axisml-infra` | `axisml-infra/deploy/helm` | `axisml-infra` | `axisml-infra` | 第三方基础设施（Envoy Gateway / RustFS / zot / GPU Operator / axisml-scheduler / kube-prometheus-stack）+ PostgreSQL + Redis |
| `axisml-system` | `axisml-system/deploy/helm` | `axisml` | `axisml-system` | 自研控制面（Cluster Manager / Compute / Artifact Hub / tenant-operator / compute-operator）+ CRDs |
| `axisml-platform` | `axisml-platform/deploy/helm` | `axisml-platform` | `axisml-platform` | 用户面（Frontend + Backend），唯一对外入口 |

三层职责边界：**Platform 唯一对外**、**System 100% 自研但不对外**、**Infra 100% 第三方**。三者通过 namespace + Service DNS 解耦，并通过 chart `condition` 支持按需关闭并对接外部实例。

```
Kubernetes Cluster
├── axisml-infra：Envoy Gateway · RustFS · zot · GPU Operator · axisml-scheduler · kube-prometheus-stack · PostgreSQL(axisml-database) · Redis(axisml-redis)
├── axisml-system：Cluster Manager · Compute Service · Artifact Hub（Deployment+Service）· tenant-operator · compute-operator（Deployment）
├── axisml-platform：Platform（Deployment+Service，Frontend+Backend）
└── tenant namespaces：默认 `axisml-tenant`；可被多个 Tenant 共享，承载 workloads / routes / secrets / ElasticQuota
```

跨 namespace 访问走 `<service>.<namespace>.svc.cluster.local`（如 `axisml-database.axisml-infra:5432`、`axisml-redis-master.axisml-infra:6379`、`axisml-compute-service.axisml-system:8080`）。端口约定：自研服务 HTTP API `:8080`、metrics `:8081`、probes `:8082`（operator 无 API 端口）。

## 3. 快速开始

本地一键全栈：

```sh
make cluster-up             # 拉起本地 minikube 集群（profile "axisml"）
make helm-install           # 按 infra → system → platform 串装（idempotent upgrade --install）
```

`make help` 列出全部目标；`make helm-template` 在不安装的前提下渲染三层 chart 供 review。卸载见 [§7](#7-卸载)。

## 4. 安装步骤

### 4.1 安装顺序与约束

务必按 **infra → system → platform** 安装。顺序由依赖决定：

- `axisml-infra` 先装——它提供 axisml-scheduler / Envoy 的 CRDs 与 PostgreSQL；否则 system 层的 Tenant / MLRun CR 找不到 ElasticQuota / HTTPRoute kind、服务也连不上 DB。
- `axisml-platform` 最后装——Platform 启动即依赖 System 层就绪（bootstrap 需调 compute）。

一键串装或分步：

```sh
make helm-install                    # infra → system → platform 一次装完
# 分步：
make helm-install-infra
make helm-install-system
make helm-install-platform
```

### 4.2 CRD 安装（重要）

`make helm-install-system` 内部先 `kubectl apply -f axisml-system/deploy/helm/crds/` 再 `helm upgrade --install`。**Helm 只在初次安装时处理 `crds/` 目录**，CRD schema 的后续升级靠这一步 `kubectl apply` 保证。若你手工用 `helm upgrade` 而非 make 目标，新增 / 变更的 CRD 不会被应用——始终经 `make helm-install-system`。

随 System 层发布的 CRD：`tenant-crd.yaml`（tenant-operator）· `mlrun/mlservice/mltrafficpolicy-crd.yaml`（compute-operator）· `resource-pool-crd.yaml`（cluster-manager 写 / compute-service Informer 读）。

## 5. 配置

### 5.1 镜像版本

所有控制面组件镜像 tag 由 `axisml-system/deploy/helm/Chart.yaml` 的 `appVersion` 统一注入（`ghcr.io/axisml/axisml-<component>:<appVersion>`，含 cluster-manager / compute-service / artifact-hub / platform / tenant-operator / compute-operator；platform 为单一镜像，后端同时托管前端 SPA）。顶层 `Makefile` 的 `IMAGE_TAG` 从该 `appVersion` 读取，作三层 chart 的统一版本源，并在 `helm-install` / `helm-template` 注入 system 与 platform 的 `--set <component>.image.tag`。

> Platform 为单一镜像（后端同时托管前端 SPA），tag 随 appVersion，由 platform 层 Makefile 经 `--set platform.image.tag` 注入。**Dev loop**：`make image-load IMAGE_TAG=dev` 后在 values override 覆盖 `image.tag=dev`。

### 5.2 PostgreSQL：内置或外接

PostgreSQL 由 `axisml-infra` chart 提供（Infra 层第三方依赖），两种模式：

- **内置**（默认）：bitnami/postgresql 子 chart，StatefulSet + PVC，Service `axisml-database`；适合开发 / 测试 / 轻量生产。System 层经 FQDN `axisml-database.axisml-infra:5432` 连接，凭据由 System 层从共享 `database.auth.password` 在本 namespace 自渲染为 Secret。
- **外接**：System 层设 `database.enabled=false` + `externalDatabase.*` 对接自建 / RDS；适合中大型生产。

所有控制面服务共用 database `axisml`，按表名前缀逻辑隔离（表 schema 见各层 database.md：[system](../axisml-system/docs/system_design/database.md) / [platform](../axisml-platform/docs/system_design/database.md)）。schema 迁移由各服务二进制内嵌 `golang-migrate` 在启动时执行（依赖 PG advisory lock 避免并发迁移）。

> **跨 namespace 凭据**：Secret 是 namespace-scoped，无法跨 namespace 引用。每个消费层从与 Infra 同值的 `database.auth.password` 在本 namespace 各自渲染一份 DB-credentials Secret，因此该密码要在 infra 与各消费层 values 中保持一致。

### 5.3 Redis 缓存（可选）

Redis 由 `axisml-infra` chart 提供（bitnami/redis 子 chart，`architecture: standalone`，StatefulSet + PVC，Service `axisml-redis-master`）。Platform 经跨 namespace FQDN `axisml-redis-master.axisml-infra:6379` 连接，凭据由 Platform 从共享 `cache.auth.password` 在本 namespace 自渲染为 Secret。

缓存仅承载可重建的会话有效性与身份 / RBAC 数据（[platform/auth.md §2.1](../axisml-platform/docs/system_design/auth.md#21-会话与身份缓存)），故为**可选加速器**：`axisml-platform` 设 `cache.enabled=false` 即不下发 `REDIS_ADDR`，Backend 全程直连 PostgreSQL；运行中 Redis 不可达按操作回退源库。单实例无需 HA——宕机 / 重启只触发一次回源（及会话强制重登），不丢业务真相。开发 / 测试 / CI 默认不依赖 Redis（`REDIS_ADDR` 空即 noop）。

### 5.4 Chart 依赖与子 chart

**`axisml-infra` 子 chart**（各带 `<name>.enabled` condition + values 根键）：Envoy Gateway（`gateway-helm`）· RustFS · zot · GPU Operator · axisml-scheduler · kube-prometheus-stack · PostgreSQL（aliased 为 `database`）· Redis（aliased 为 `cache`）。

**`axisml-system` / `axisml-platform`** 无第三方子 chart。System 层把 PostgreSQL 当 Infra 外部依赖（`database.enabled=true` 连 `axisml-database.<infraNamespace>:5432` 并自渲染连接 Secret；`=false` 改用 `externalDatabase`）；Platform 层通过 `system.*` values 拼装跨 namespace FQDN 调 System 服务，并把 PostgreSQL 与 Redis 当 Infra 外部依赖消费（各自从共享 `*.auth.password` 自渲染本 namespace Secret）。

> **fullnameOverride**：`kube-prometheus-stack` → `prometheus`（避上游 26 字符截断）；`postgresql` → `database`（资源命名 `axisml-database-*`）；`redis` → `axisml-redis`（资源命名 `axisml-redis-*`，主节点 Service `axisml-redis-master`）。
>
> **bitnamilegacy 镜像**：bitnami 2025 年移除 Docker Hub 免费 `bitnami/*` 镜像，PostgreSQL 与 Redis 子 chart 均改 pin `bitnamilegacy/*` 镜像；Redis 子 chart 因此需 `global.security.allowInsecureImages=true` 放行非官方镜像校验。

## 6. 验证

安装后逐层确认：

```sh
kubectl get pods -n axisml-infra        # 网关 / 存储 / DB / Redis / 调度 / 监控就绪
kubectl get pods -n axisml-system       # 5 个控制面组件 Running，post-install Job Completed
kubectl get pods -n axisml-platform     # Platform 就绪
kubectl get crds | grep axisml          # tenant / mlrun / mlservice / mltrafficpolicy / resourcepool
kubectl get tenant,resourcepool         # 内置租户 default 与默认 ResourcePool 已建
```

各服务暴露 `/healthz`（liveness）与 `/readyz`（readiness）；Pod 全部 Ready 即控制面就绪。黑盒 e2e 套件见 `tests/README.md`（Python + pytest，经 `uv run pytest` 运行）。

## 7. 卸载

```sh
make helm-uninstall          # 反向 platform → system → infra
```

CRD 与 `axisml-tenant` Namespace 标记了 `helm.sh/resource-policy=keep`，不随 chart 卸载删除（避免误删租户工作负载）；需要彻底清理时手工 `kubectl delete crd ...` 与对应 namespace。

## 8. 组件部署形态参考

### 8.1 控制面 Deployment

| 组件 | 副本 | 端口 | leader election | 备注 |
| --- | --- | --- | --- | --- |
| Cluster Manager | `1` 默认 | `:8080`/`:8081`/`:8082` | 无 | 多副本对等；无 reconciler；bootstrap Job 初始化默认 ResourcePool |
| Compute Service | `1` 默认 | 同上 | controller-runtime Lease | API 无状态可水平扩；reconciler / informer 单 leader（数据卷 PVC 由 Platform 经 cluster-manager 管理，compute 不派生） |
| Artifact Hub | `1` 默认 | 同上 | PG advisory lock | GC worker 经 `pg_try_advisory_lock` 选主；不连 K8s API；API 无状态 |
| tenant-operator / compute-operator | `1`(leader)+N 备 | `:8081`/`:8082`（无 API） | controller-runtime Lease | 单 leader |
| Platform | `1` 默认 | API+前端 `:8080` / probes `:8081` | 无 | 单一镜像：后端同源提供 API 与前端 SPA 静态资源；经 FQDN 调 System 服务 |

### 8.2 Infra 组件部署形态（默认 values）

Envoy Gateway（单 `axisml-gateway`，HTTP listener，`allowedRoutes` 放行工作负载 namespace）· RustFS（Standalone）· zot（Standalone filesystem，公共拉取 Secret 落 `axisml-tenant`）· GPU Operator（driver + toolkit + device plugin + DCGM + GFD，MIG 暂不启用）· axisml-scheduler（scheduler + controller + ElasticQuota + PodGroup）· kube-prometheus-stack（不预置告警）· PostgreSQL（bitnami，Service `axisml-database`，`database.enabled=false` 时外接）· Redis（bitnami，standalone，Service `axisml-redis-master`，`cache.enabled=false` 时关闭）。

默认共享 K8s Namespace `axisml-tenant` 由 System chart 声明并标记 `helm.sh/resource-policy=keep`；其他 tenant namespace 由 tenant-operator 在 `Tenant` CR reconcile 时按需创建（[tenant-operator.md §4.1.1](../axisml-system/docs/system_design/tenant-operator.md#411-namespace-落地)）。

### 8.3 Helm 模板清单

System 层模板在 `axisml-system/deploy/helm/templates/<component>/` 下：

- **Cluster Manager / Compute Service / Artifact Hub**：`configmap.yaml`（DB 连接 / 日志 / 下游 URL）· `secret-db.yaml`（从共享 `database.auth.password` 投影到本 namespace）· `deployment.yaml`（含 `/healthz` `/readyz`）· `service.yaml`（ClusterIP）· `networkpolicy.yaml`（API `:8080` 仅允许 `axisml-platform`，metrics 仅允许监控 namespace）· `serviceaccount.yaml` · `rbac.yaml`（ClusterRole + Binding）· `role.yaml`/`rolebinding.yaml`（leader Lease）· `servicemonitor.yaml`（opt-in）· `post-install-job.yaml`。Cluster Manager 的默认数据模板位于 `templates/cluster-manager/`：`resource-pool-default.yaml`、`namespace-tenant.yaml`（`axisml-tenant`）和 `tenant-default.yaml`（内置租户 `default`）。
- **tenant-operator / compute-operator**：`deployment.yaml` · `serviceaccount.yaml` · `clusterrole.yaml`/`clusterrolebinding.yaml` · `role.yaml`/`rolebinding.yaml`（leader Lease）· `servicemonitor.yaml`。
- **CRDs**（在 `crds/`，非 `templates/`，由 `make helm-install-system` 的 `kubectl apply` 保证 schema 升级）：`tenant-crd.yaml`（tenant-operator）· `mlrun/mlservice/mltrafficpolicy-crd.yaml`（compute-operator）· `resource-pool-crd.yaml`（cluster-manager 写 / compute-service Informer 读）。
- **Platform**（`axisml-platform/deploy/helm/templates/`）：`configmap.yaml`（下游 URL）· `deployment.yaml` · `service.yaml` · `httproute.yaml`（挂载 Infra 层 `axisml-gateway`，`platform.httpRoute.enabled` 开关）· `bootstrap-job.yaml`（规划中：创建初始 `system-admin`，依赖 System 层就绪，故最后安装）。

CRDs 随 System 层发布（operator 契约）；Platform 用户体系 bootstrap 随 Platform 层发布，两者不交叉。

## 9. 故障排查

| 现象 | 可能原因 / 处理 |
| --- | --- |
| Tenant / MLRun CR 创建报 `no matches for kind` | Infra 未先装或 CRD 未 apply——按 infra → system 顺序，且经 `make helm-install-system`（含 `kubectl apply crds/`） |
| 服务启动连不上 DB | `database.auth.password` 在 infra 与消费层 values 不一致；或外接模式下 `externalDatabase.*` 配错 |
| CRD schema 升级未生效 | 用了裸 `helm upgrade`；改用 `make helm-install-system` 触发 `kubectl apply crds/` |
| Redis 子 chart 镜像校验失败 | 需 `global.security.allowInsecureImages=true`（bitnamilegacy 非官方镜像） |
| Platform 启动失败 | System 层未就绪即装 platform；确认 system 全 Ready 后再装 platform |
| 卸载后租户工作负载仍在 | `axisml-tenant` 与 CRD 标了 `resource-policy=keep`，需手工清理 |

## 10. 部署相关设计决策

| 决策项 | 决策 |
| --- | --- |
| Chart 拆分 | 三 chart 对齐 Platform / System / Infra 职责分层（发版节奏、回滚粒度、对外暴露面各不同；infra 可共享给多套实例） |
| 数据库归属 | 纳入 `axisml-infra`（沿"Infra = 100% 第三方"边界，与 RustFS/zot 同性质）；System 层当外部依赖消费，连接凭据用投影 Secret 解决跨 namespace 引用 |
| 缓存归属 | Redis 同纳入 `axisml-infra`（与 PostgreSQL 同性质）；Platform 当外部依赖消费。仅缓存可重建数据，故 standalone 单实例 + 可选降级，权威始终在 PostgreSQL |
| CRD 安装 | `kubectl apply -f crds/` + `helm upgrade --install` 组合（Helm 只在初次安装处理 `crds/`，apply 保证 schema 升级） |
| 镜像 tag 来源 | 由 `axisml-system/Chart.yaml` `appVersion` 统一注入三 chart（单一版本源，保证 chart 与镜像一致） |
| 外部 PostgreSQL | `database.enabled=false` + `externalDatabase.*` 切换（生产推荐外接 RDS） |

## 11. 关联文档

- [high_level_design.md](high_level_design.md)（系统架构与三层职责）· [infra/overview.md](../axisml-infra/docs/system_design/overview.md)（infra 层组件详情）· [development_workflow.md](development_workflow.md)（本地开发）· 各组件详设 §8 运行时形态。
