# AxisML 部署设计

汇总控制平面与基础设施的部署形态：Helm chart 组织、Namespace 约定、安装顺序、依赖清单、镜像版本与模板清单。

## 1. 部署形态总览

按 Platform / System / Infra 三层各部署为一个 Helm chart：

| Chart | 路径 | Release | Namespace | 内容 |
| --- | --- | --- | --- | --- |
| `axisml-infra` | `deploy/helm/axisml-infra` | `axisml-infra` | `axisml-infra` | 第三方基础设施（Envoy Gateway / RustFS / zot / GPU Operator / Koordinator / kube-prometheus-stack）+ PostgreSQL |
| `axisml-system` | `deploy/helm/axisml-system` | `axisml` | `axisml-system` | 自研控制面（Cluster Manager / Compute / Artifact Hub / tenant-operator / compute-operator）+ CRDs |
| `axisml-platform` | `deploy/helm/axisml-platform` | `axisml-platform` | `axisml-platform` | 用户面（Frontend + Backend），唯一对外入口 |

三层职责边界：**Platform 唯一对外**、**System 100% 自研但不对外**、**Infra 100% 第三方**。三者通过 namespace + Service DNS 解耦，并通过 chart `condition` 支持按需关闭并对接外部实例。

```
Kubernetes Cluster
├── axisml-infra：Envoy Gateway · RustFS · zot · GPU Operator · Koordinator · kube-prometheus-stack · PostgreSQL(axisml-database)
├── axisml-system：Cluster Manager · Compute Service · Artifact Hub（Deployment+Service）· tenant-operator · compute-operator（Deployment）
├── axisml-platform：Platform（Deployment+Service，Frontend+Backend）
└── tenant namespaces：Tenant resources / workloads / routes / secrets / ElasticQuota
```

跨 namespace 访问走 `<service>.<namespace>.svc.cluster.local`（如 `axisml-database.axisml-infra:5432`、`axisml-compute-service.axisml-system:8080`）。端口约定：自研服务 HTTP API `:8080`、metrics `:8081`、probes `:8082`（operator 无 API 端口）。

## 2. 安装顺序

```sh
make cluster-up             # 拉起本地集群
make helm-install           # 一次性按 infra → system → platform 串装
# 或分步：helm-install-infra → helm-install-system → helm-install-platform
```

卸载顺序相反。`make helm-install-system` 内部先 `kubectl apply deploy/helm/axisml-system/crds/` 再 `helm upgrade --install`——Helm 只在初次安装时处理 `crds/`，CRD schema 升级靠这一步保证。

**顺序约束**：`axisml-infra` 提供 Koordinator / Envoy CRDs 与 PostgreSQL，先于 system（否则 Tenant / MLRun CR 找不到 ElasticQuota / HTTPRoute kind、服务连不上 DB）；`axisml-platform` 最后装（Platform 启动即依赖 System 层就绪，bootstrap 需调 compute）。

## 3. Chart 依赖

**`axisml-infra` 子 chart**（各带 `<name>.enabled` condition + values 根键）：Envoy Gateway（`gateway-helm`）· RustFS · zot · GPU Operator · Koordinator · kube-prometheus-stack · PostgreSQL（aliased 为 `database`）。

**`axisml-system` / `axisml-platform`**：无第三方子 chart。System 层把 PostgreSQL 当 Infra 层外部依赖（`database.enabled=true` 连 `axisml-database.<infraNamespace>:5432` 并自渲染连接 Secret；`=false` 改用 `externalDatabase`）；Platform 层通过 `system.*` values 拼装跨 namespace FQDN 调 System 服务。

> **fullnameOverride**：`kube-prometheus-stack` → `prometheus`（避上游 26 字符截断）；`postgresql` → `database`（资源命名 `axisml-database-*`）。

## 4. 镜像版本

所有控制面组件镜像 tag 由 `deploy/helm/axisml-system/Chart.yaml` 的 `appVersion` 统一注入（`ghcr.io/axisml/axisml-<component>:<appVersion>`，含 cluster-manager / compute-service / artifact-hub / platform-{backend,frontend} / tenant-operator / compute-operator）。顶层 `Makefile` 的 `IMAGE_TAG` 从该 `appVersion` 读取，作三层 chart 统一版本源，并在 `helm-install` / `helm-template` 注入 system 与 platform 的 `--set <component>.image.tag`。

> Platform 当前仍是 nginx 占位镜像（不跟 appVersion），故 `HELM_PLATFORM_IMAGE_SET` 暂留空。**Dev loop**：`make image-load IMAGE_TAG=dev` 后在 values override 覆盖 `image.tag=dev`。

## 5. 控制面 Deployment

| 组件 | 副本 | 端口 | leader election | 备注 |
| --- | --- | --- | --- | --- |
| Cluster Manager | `1` 默认 | `:8080`/`:8081`/`:8082` | 无 | 多副本对等；无 reconciler；bootstrap Job 初始化默认 ResourcePool |
| Compute Service | `1` 默认 | 同上 | controller-runtime Lease | API 无状态可水平扩；reconciler / informer 单 leader；workspace 同事务派生 PVC |
| Artifact Hub | `1` 默认 | 同上 | `coordination.k8s.io/Lease` | GC worker 选主；API 无状态 |
| tenant-operator / compute-operator | `1`(leader)+N 备 | `:8081`/`:8082`（无 API） | controller-runtime Lease | 单 leader |
| Platform | `1` 默认 | 当前 nginx placeholder 仅 `:8080` | 无 | 真实 backend 目标 API `:8080` / metrics `:8081` / probes `:8082`；经 FQDN 调 System 服务 |

**Infra 组件部署形态**（默认 values）：Envoy Gateway（单 `axisml-gateway`，HTTP listener，`allowedRoutes` 放行工作负载 namespace）· RustFS（Standalone）· zot（Standalone filesystem，公共拉取 Secret 落 `axisml-system`）· GPU Operator（driver + toolkit + device plugin + DCGM + GFD，MIG 暂不启用）· Koordinator（koord-scheduler + koord-manager + ElasticQuota + PodGroup）· kube-prometheus-stack（不预置告警）· PostgreSQL（bitnami，Service `axisml-database`，`database.enabled=false` 时外接）。

`tenant namespaces` 由 tenant-operator 在 `Tenant` CR reconcile 时创建（[tenant-operator.md §4.1.1](system/tenant-operator.md#411-namespace-落地)），不在 Helm chart 内静态声明。

## 6. Helm 模板清单

System 层模板在 `deploy/helm/axisml-system/templates/<component>/` 下：

- **Cluster Manager / Compute Service / Artifact Hub**：`configmap.yaml`（DB 连接 / 日志 / 下游 URL）· `secret-db.yaml`（从共享 `database.auth.password` 投影到本 namespace）· `deployment.yaml`（含 `/healthz` `/readyz`）· `service.yaml`（ClusterIP）· `serviceaccount.yaml` · `rbac.yaml`（ClusterRole + Binding）· `role.yaml`/`rolebinding.yaml`（leader Lease）· `servicemonitor.yaml`（opt-in）· `post-install-job.yaml`。默认数据由 `templates/seed/` post-install hook 落地：`resource-pool-default.yaml`（default ResourcePool，内嵌 cpu-small/cpu-medium）、`tenant-system.yaml`（内置租户 `axisml-system`）。
- **tenant-operator / compute-operator**：`deployment.yaml` · `serviceaccount.yaml` · `clusterrole.yaml`/`clusterrolebinding.yaml` · `role.yaml`/`rolebinding.yaml`（leader Lease）· `servicemonitor.yaml`。
- **CRDs**（在 `crds/`，非 `templates/`，由 `make helm-install-system` 的 `kubectl apply` 保证 schema 升级）：`tenant-crd.yaml`（tenant-operator）· `mlrun/mlservice/mltrafficpolicy-crd.yaml`（compute-operator）· `resource-pool-crd.yaml`（cluster-manager 写 / compute-service Informer 读）。
- **Platform**（`deploy/helm/axisml-platform/templates/`）：`configmap.yaml`（下游 URL）· `deployment.yaml` · `service.yaml` · `ingress.yaml`（唯一对外入口，`platform.ingress.enabled` 开关）· `bootstrap-job.yaml`（规划中：创建初始 `system-admin`，依赖 System 层就绪，故最后安装）。

CRDs 随 System 层发布（operator 契约）；Platform 用户体系 bootstrap 随 Platform 层发布，两者不交叉。

## 7. PostgreSQL 部署模式

由 `axisml-infra` chart 提供（Infra 层第三方依赖），两种模式：**内置**（bitnami/postgresql 子 chart，StatefulSet + PVC，Service `axisml-database`；开发 / 测试 / 轻量生产）与**外部**（System 层 `database.enabled=false` + `externalDatabase.*` 对接自建 / RDS；中大型生产）。内置模式下 System 层经 FQDN `axisml-database.axisml-infra:5432` 连接，凭据由 System 层从共享 `database.auth.password` 在本 namespace 自渲染为 Secret。所有控制面服务共用 database `axisml`，按表名前缀逻辑隔离（[database.md](database.md)）。

## 8. 部署相关设计决策

| 决策项 | 决策 |
| --- | --- |
| Chart 拆分 | 三 chart 对齐 Platform / System / Infra 职责分层（发版节奏、回滚粒度、对外暴露面各不同；infra 可共享给多套实例） |
| 数据库归属 | 纳入 `axisml-infra`（沿"Infra = 100% 第三方"边界，与 RustFS/zot 同性质）；System 层当外部依赖消费，连接凭据用投影 Secret 解决跨 namespace 引用 |
| CRD 安装 | `kubectl apply -f crds/` + `helm upgrade --install` 组合（Helm 只在初次安装处理 `crds/`，apply 保证 schema 升级） |
| 镜像 tag 来源 | 由 `axisml-system/Chart.yaml` `appVersion` 统一注入三 chart（单一版本源，保证 chart 与镜像一致） |
| 外部 PostgreSQL | `database.enabled=false` + `externalDatabase.*` 切换（生产推荐外接 RDS） |

## 9. 关联文档

- [high_level_design.md](high_level_design.md) · [infra/overview.md](infra/overview.md)（infra 层组件）· [database.md](database.md)（PostgreSQL 形态）· 各组件详设 §8 运行时形态。
