# AxisML 部署设计

本文档汇总 AxisML 控制平面与基础设施的部署形态：Helm chart 组织、Namespace 约定、安装顺序、依赖清单、镜像版本与各组件 Helm 模板清单。

---

## 1. 部署形态总览

AxisML 基于 Kubernetes 部署，按 Platform / System / Infra 三层各部署为一个 Helm chart：

| Chart | 路径 | Release | Namespace | 层 | 内容 |
| --- | --- | --- | --- | --- | --- |
| `axisml-infra` | `deploy/helm/axisml-infra` | `axisml-infra` | `axisml-infra` | Infra | 第三方基础设施（Envoy Gateway / RustFS / zot / GPU Operator / Koordinator / kube-prometheus-stack）+ 元数据数据库 PostgreSQL |
| `axisml-system` | `deploy/helm/axisml-system` | `axisml` | `axisml-system` | System | AxisML 自研控制面（Cluster Manager / Compute / Artifact Hub / tenant-operator / compute-operator）+ CRDs |
| `axisml-platform` | `deploy/helm/axisml-platform` | `axisml-platform` | `axisml-platform` | Platform | 用户面（Frontend + Backend），唯一对外入口 |

三者通过 namespace + Service DNS 解耦，并通过 chart `condition` 字段支持按需关闭并对接外部实例。三层职责边界一句话：**Platform 层唯一对外**、**System 层 100% 自研但不对外**、**Infra 层 100% 第三方**。

```
Kubernetes Cluster
├── axisml-infra namespace
│   ├── Envoy Gateway
│   ├── RustFS
│   ├── zot
│   ├── NVIDIA GPU Operator
│   ├── Koordinator
│   ├── kube-prometheus-stack
│   └── PostgreSQL (axisml-database) / externalDatabase
├── axisml-system namespace
│   ├── AxisML Cluster Manager    (Deployment + Service)
│   ├── AxisML Compute Service    (Deployment + Service)
│   ├── AxisML Artifact Hub       (Deployment + Service)
│   ├── tenant-operator           (Deployment)
│   └── compute-operator          (Deployment)
├── axisml-platform namespace
│   └── AxisML Platform           (Deployment + Service, Frontend + Backend)
└── tenant namespaces
    └── Tenant resources / workloads / routes / secrets / ElasticQuota
```

跨 namespace 访问统一走 `<service>.<namespace>.svc.cluster.local`，例如 `rustfs-svc.axisml-infra:9000`、`zot.axisml-infra:5000`、`axisml-database.axisml-infra:5432`、`axisml-compute-service.axisml-system:8080`。Platform 调用下游 System 服务、System 服务连接 Infra 层数据库均走此跨 namespace FQDN。端口以 Helm values 为准：自研服务 HTTP API 统一 `:8080`；metrics 统一 `:8081`；probes 统一 `:8082`（operator 无 API 端口）。

---

## 2. 安装顺序

```sh
make cluster-up             # 拉起本地集群
make helm-install-infra     # 先装基础设施（含 PostgreSQL 与 Koordinator/Envoy CRDs）
make helm-install-system    # 再装控制面（含 CRDs）
make helm-install-platform  # 最后装用户面
```

`make helm-install` 一次性按 infra → system → platform 顺序串装。卸载顺序相反：`helm-uninstall-platform` → `helm-uninstall-system` → `helm-uninstall-infra`。

`make helm-install-system` 内部先 `kubectl apply` `deploy/helm/axisml-system/crds/`，再做 `helm upgrade --install`——Helm 只在初次安装时处理 `crds/` 目录，CRD schema 升级靠这一步保证。

> **顺序约束**：
> - `axisml-infra` 提供 Koordinator / Envoy Gateway 等 CRDs 与 PostgreSQL；先于 system 安装，否则 Tenant / MLRun CR 找不到 ElasticQuota / HTTPRoute kind、compute / artifact 连不上数据库。
> - `axisml-platform` 在最后安装：Platform 启动即依赖 System 层的 compute-service / artifact-hub 就绪，且其 bootstrap（初始化 `system-admin` 与内置租户）需调用 compute。

---

## 3. Chart 依赖清单

### 3.1 `axisml-infra` dependencies

| Dependency | 仓库 | condition | values 根键 |
| --- | --- | --- | --- |
| `gateway-helm`（Envoy Gateway） | `oci://docker.io/envoyproxy` | `envoy-gateway.enabled` | `envoy-gateway` |
| `rustfs` | `https://charts.rustfs.com` | `rustfs.enabled` | `rustfs` |
| `zot` | `https://zotregistry.dev/helm-charts` | `zot.enabled` | `zot` |
| `gpu-operator` | `https://helm.ngc.nvidia.com/nvidia` | `gpu-operator.enabled` | `gpu-operator` |
| `koordinator` | `https://koordinator-sh.github.io/charts` | `koordinator.enabled` | `koordinator` |
| `kube-prometheus-stack` | `https://prometheus-community.github.io/helm-charts` | `kube-prometheus-stack.enabled` | `kube-prometheus-stack` |
| `postgresql`（aliased 为 `database`） | `oci://registry-1.docker.io/bitnamicharts` | `database.enabled` | `database` |

### 3.2 `axisml-system` dependencies

无第三方子 chart。System 层把 PostgreSQL 当作 Infra 层提供的外部依赖：`database.enabled=true` 时连接 `axisml-database.<infraNamespace>:5432`，并在本 namespace 内自渲染连接 Secret；`database.enabled=false` 时改用 `externalDatabase` 指向托管实例。详见 [database.md](database.md)。

### 3.3 `axisml-platform` dependencies

无第三方子 chart。Platform 层通过跨 namespace FQDN 调用 System 层服务（`system.namespace` / `system.computeService` / `system.artifactHub` values）。

> **fullnameOverride 约定**：`kube-prometheus-stack` 的 `fullnameOverride` 设为 `prometheus`，避开上游 26 字符截断；`postgresql` 别名为 `database`，使资源命名 `axisml-database-*`。

各子 chart 的具体 values 透传由 values 根键直接写入。PostgreSQL 部署模式见 [database.md](database.md) / [infra.md §4.4](infra.md#44-数据库postgresql)。

---

## 4. 镜像版本与 IMAGE_TAG

所有控制面组件的镜像 tag 由 `deploy/helm/axisml-system/Chart.yaml` 的 `appVersion` 统一注入：

| 组件 | 镜像 |
| --- | --- |
| Cluster Manager | `ghcr.io/axisml/axisml-cluster-manager:<appVersion>` |
| Compute Service | `ghcr.io/axisml/axisml-compute-service:<appVersion>` |
| Artifact Hub | `ghcr.io/axisml/axisml-artifact-hub:<appVersion>` |
| Platform Backend | `ghcr.io/axisml/axisml-platform-backend:<appVersion>` |
| Platform Frontend | `ghcr.io/axisml/axisml-platform-frontend:<appVersion>` |
| tenant-operator | `ghcr.io/axisml/axisml-tenant-operator:<appVersion>` |
| compute-operator | `ghcr.io/axisml/axisml-compute-operator:<appVersion>` |

顶层 `Makefile` 的 `IMAGE_TAG` 变量从 `axisml-system/Chart.yaml` 的 `appVersion` 读取，作为三层 chart 的统一版本源，并传给各组件 `make image` 目标，保证 chart 与镜像 tag 一致。`helm-install` / `helm-template` 把它注入到 system 与 platform 两个 chart 的 `--set <component>.image.tag` 上。

> Platform 当前仍是 nginx 占位镜像（不跟随 appVersion），故 `HELM_PLATFORM_IMAGE_SET` 暂留空；待 Platform 发布真实镜像后再放开统一注入。

**Dev loop**：`make image-load IMAGE_TAG=dev` 后，在 values override 中覆盖 `image.tag=dev` 即可使用本地构建的镜像。详见 [CLAUDE.md "Image tag synchronization" 段](../../CLAUDE.md)。

---

## 5. Namespace 与控制面 Deployment

`axisml-system` namespace 内的控制面 Deployment：

| 组件 | 副本 | 端口 | leader election | 备注 |
| --- | --- | --- | --- | --- |
| Cluster Manager | `1` 默认 | `:8080` API / `:8081` metrics / `:8082` probes | 无 | K8s admin REST 抽象（ResourcePool CRD CRUD），多副本对等运行；无 reconciler / 无 informer (list 端点可选 Pool Informer cache)；bootstrap Job 初始化默认 ResourcePool CR (含 cpu-small/cpu-medium unit) |
| Compute Service | `1` 默认 | `:8080` API / `:8081` metrics / `:8082` probes | controller-runtime Lease | API 层无状态可水平扩；reconciler / informer 单 leader；workspace 创建时同事务派生 PVC |
| Artifact Hub | `1` 默认 | `:8080` API / `:8081` metrics / `:8082` probes | `coordination.k8s.io/Lease` | GC worker 选主；API 层无状态 |
| tenant-operator | `1`（leader）+ N 备 | `:8081` metrics / `:8082` probes（无 API） | controller-runtime Lease | 单 leader |
| compute-operator | `1`（leader）+ N 备 | 同上 | controller-runtime Lease | 单 leader；dispatcher + handler 模型 |

`axisml-platform` namespace 内的用户面 Deployment：

| 组件 | 副本 | 端口 | leader election | 备注 |
| --- | --- | --- | --- | --- |
| Platform chart | `1` 默认 | 当前 nginx placeholder 仅 `:8080` HTTP | 无 | 真实 Platform Backend 目标为 API `:8080` / metrics `:8081` / probes `:8082`；经跨 namespace FQDN 调 System 层服务 |

`axisml-infra` namespace 内的基础设施组件部署形态（默认 values）：

| 组件 | 形态 |
| --- | --- |
| Envoy Gateway | 单 GatewayClass + `axisml-gateway`（HTTP listener），`allowedRoutes.namespaces` 放行接入工作负载所在 namespace |
| RustFS | Standalone（单 Pod + PVC），admin 凭证由 chart 自动生成 |
| zot | Standalone（filesystem 后端），公共拉取 Secret 落地到 `axisml-system` namespace |
| GPU Operator | driver + container toolkit + device plugin + DCGM Exporter + GFD；MIG 暂不启用 |
| Koordinator | `koord-scheduler` + `koord-manager` + ElasticQuota plugin + PodGroup CRD |
| kube-prometheus-stack | Prometheus + Grafana + AlertManager；ServiceMonitor 自动发现；不预置告警规则 |
| PostgreSQL | bitnami 子 chart，Service `axisml-database`；System 层经 `axisml-database.axisml-infra:5432` 跨 namespace 连接；`database.enabled=false` 时改用外部托管实例 |

`tenant namespaces` 由 tenant-operator 在 `Tenant` CR reconcile 时创建（详见 [tenant-operator.md §4.1.1 Namespace 落地](components/tenant-operator.md#411-namespace-落地)），不在 Helm chart 内静态声明。

---

## 6. Helm 模板清单

System 层组件的 Helm 模板放在 `deploy/helm/axisml-system/templates/<component>/` 下，文件清单按组件略有差异：

### 6.1 Cluster Manager / Compute Service / Artifact Hub

| 文件 | 用途 |
| --- | --- |
| `configmap.yaml` | DB 连接、日志级别、下游 URL |
| `secret-db.yaml` | DB 连接凭据（从共享 `database.auth.password` 投影到本 namespace） |
| `deployment.yaml` | 主 Deployment，含探针 `/healthz` / `/readyz` |
| `service.yaml` | ClusterIP（cluster-manager / compute-service / artifact-hub） |
| `serviceaccount.yaml` | 服务账号 |
| `rbac.yaml` | ClusterRole + ClusterRoleBinding（详见各组件详设 §2.5） |
| `role.yaml` / `rolebinding.yaml` | leader election Lease 权限（在 `axisml-system` namespace 内） |
| `servicemonitor.yaml` | `/metrics` 暴露，当前由 compute-service / artifact-hub 提供并通过 `*.serviceMonitor.enabled` opt-in |
| `post-install-job.yaml` | post-install Job：compute-service 初始化默认 ResourcePool 引用 |

System 层的默认数据由 `templates/seed/` 下的 post-install hook 落地：`resource-pool-default.yaml`（default ResourcePool，内嵌 cpu-small/cpu-medium unit）、`tenant-system.yaml`（内置租户 `axisml-system`）。

### 6.2 tenant-operator / compute-operator

| 文件 | 用途 |
| --- | --- |
| `deployment.yaml` | operator Deployment |
| `serviceaccount.yaml` | service account |
| `clusterrole.yaml` / `clusterrolebinding.yaml` | 跨 namespace 资源读写权限 |
| `role.yaml` / `rolebinding.yaml` | leader election Lease（namespace-scoped） |
| `servicemonitor.yaml` | `/metrics` 暴露 |

### 6.3 CRDs

CRD 定义放在 `deploy/helm/axisml-system/crds/` 下（不在 `templates/`）：

| CRD | 文件 | 由谁消费 |
| --- | --- | --- |
| `tenants.axisml.io` | `crds/tenant-crd.yaml` | tenant-operator |
| `mlruns.axisml.io` | `crds/mlrun-crd.yaml` | compute-operator |
| `mlservices.axisml.io` | `crds/mlservice-crd.yaml` | compute-operator |
| `resourcepools.axisml.io` | `crds/resource-pool-crd.yaml` | cluster-manager (写) / compute-service (Informer 读做展开) |

CRD schema 升级由 `make helm-install-system` 的 `kubectl apply -f crds/` 一步保证（见 §2）。

### 6.4 Platform（`axisml-platform` chart）

Platform 层模板放在 `deploy/helm/axisml-platform/templates/` 下：

| 文件 | 用途 |
| --- | --- |
| `configmap.yaml` | 下游 System 层服务 URL（跨 namespace FQDN，由 `system.*` values 拼装） |
| `deployment.yaml` | Platform Deployment |
| `service.yaml` | ClusterIP |
| `ingress.yaml` | 对外入口（唯一暴露层；`platform.ingress.enabled` 开关） |
| `bootstrap-job.yaml`（规划中） | post-install Job：创建初始 `system-admin`（admin/admin，首次登录强制改密）；依赖 System 层 compute-service 就绪，故 Platform 在最后一层安装 |

### 6.5 CRDs 与 Platform bootstrap 的归属

CRDs 随 System 层发布（operator 的契约）；Platform 的用户体系 bootstrap 随 Platform 层发布。两者不交叉。

---

## 7. PostgreSQL 部署模式

PostgreSQL 由 `axisml-infra` chart 提供（Infra 层第三方依赖），支持两种模式：

| 模式 | 说明 | 适用场景 |
| --- | --- | --- |
| 内置 | bitnami/postgresql 子 chart（StatefulSet + PVC），Service `axisml-database` | 开发、测试、轻量生产 |
| 外部 | System 层 `database.enabled=false` + `externalDatabase.*` 对接外部实例（自建 / RDS） | 中大型生产 |

内置模式下，System 层服务经跨 namespace FQDN `axisml-database.axisml-infra:5432` 连接，连接凭据由 System 层从共享的 `database.auth.password`（与 infra chart 同值）在本 namespace 自渲染为 Secret——Secret 为 namespace-scoped，不跨 namespace 引用。详见 [database.md](database.md)。

所有控制面服务共用同一个 database `axisml`，按表名前缀逻辑隔离（详见 [database.md §1](database.md)）。

---

## 8. 关键设计决策

| 决策项 | 决策 | 理由 |
| --- | --- | --- |
| Chart 拆分 | `axisml-infra` / `axisml-system` / `axisml-platform` 三个 chart，对齐 Platform / System / Infra 职责分层 | 用户面、自研控制面、第三方基础设施三者发版节奏、回滚粒度、对外暴露面各不相同；infra 可共享给多套实例 |
| 数据库归属 | 纳入 `axisml-infra` chart（Infra 层） | 沿"Infra 层 = 100% 第三方"边界，PostgreSQL 与 RustFS/zot 同性质；System 层把它当外部依赖消费，连接凭据用投影 Secret 解决跨 namespace 引用 |
| CRD 安装 | `kubectl apply -f crds/` + `helm upgrade --install` 组合 | Helm 只在初次安装时处理 `crds/`；apply 一步保证 schema 升级 |
| 镜像 tag 来源 | 由 `axisml-system/Chart.yaml` `appVersion` 统一注入三 chart | 单一版本源，保证 chart 与镜像 tag 一致；避免 Helm rendered Deployment 拉不到镜像 |
| Operator 形态 | tenant-operator + compute-operator 拆为两个独立二进制 | 管理员域与业务域按变更频率与权限边界分离；任一 operator 演进或重启不影响另一域 |
| Namespace 命名 | `axisml-platform` / `axisml-system` / `axisml-infra` + 租户 namespace 由 tenant-operator 派生 | 三层 namespace 隔离，对外暴露面收敛到 Platform；租户 namespace 名由 Tenant CR `spec.namespace.name` 决定 |
| 外部 PostgreSQL | System 层 `database.enabled=false` + `externalDatabase.*` 切换 | 生产推荐外接 RDS；Infra 层内置 bitnami chart 简化本地起步 |

---

## 9. 关联文档

- [overview.md §6 部署架构](overview.md#6-部署架构)：总体部署示意；
- [infra.md §5 部署形态](infra.md#5-部署形态)：infra chart 子组件部署细节；
- [database.md §1 部署形态](database.md)：PostgreSQL 部署形态；
- [monitoring.md §1 接入模型](monitoring.md#1-接入模型)：ServiceMonitor 接入；
- 各组件详设的 §8 运行时形态 章节：[cluster-manager](components/cluster-manager.md#8-运行时形态) / [compute-service](components/compute-service.md#8-运行时形态) / [artifact-hub](components/artifact-hub.md#8-运行时形态) / [tenant-operator](components/tenant-operator.md#8-运行时形态) / [compute-operator](components/compute-operator.md#8-运行时形态) / [platform](components/platform.md#8-运行时形态)。
