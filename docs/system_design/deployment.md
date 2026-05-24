# AxisML 部署设计

本文档汇总 AxisML 控制平面与基础设施的部署形态：Helm chart 组织、Namespace 约定、安装顺序、依赖清单、镜像版本与各组件 Helm 模板清单。

---

## 1. 部署形态总览

AxisML 基于 Kubernetes 部署，通过两个 Helm chart 分层安装：

| Chart | 路径 | Release | Namespace | 内容 |
| --- | --- | --- | --- | --- |
| `axisml-infra` | `deploy/helm/axisml-infra` | `axisml-infra` | `axisml-infra` | 第三方基础设施组件（Envoy Gateway / RustFS / zot / GPU Operator / Koordinator / kube-prometheus-stack） |
| `axisml-system` | `deploy/helm/axisml-system` | `axisml` | `axisml-system` | AxisML 自研控制平面 + CRDs + 元数据数据库（PostgreSQL） |

拆分原因：基础设施和控制平面发版节奏、回滚粒度不同；拆分后 infra 可共享给多套 axisml-system 实例。两者通过 namespace + Service DNS 解耦，并通过 chart `condition` 字段支持按需关闭并对接外部实例。

```
Kubernetes Cluster
├── axisml-infra namespace
│   ├── Envoy Gateway
│   ├── RustFS
│   ├── zot
│   ├── NVIDIA GPU Operator
│   ├── Koordinator
│   └── kube-prometheus-stack
├── axisml-system namespace
│   ├── AxisML Platform           (Deployment + Service)
│   ├── AxisML Cluster Manager    (Deployment + Service)
│   ├── AxisML Compute            (Deployment + Service)
│   ├── AxisML Artifacts          (Deployment + Service)
│   ├── tenant-operator           (Deployment)
│   ├── compute-operator          (Deployment)
│   └── PostgreSQL / externalDatabase
└── tenant namespaces
    └── Tenant resources / workloads / routes / secrets / ElasticQuota
```

跨 namespace 访问统一走 `<service>.<namespace>.svc.cluster.local`，例如 `rustfs-svc.axisml-infra:9000`、`zot.axisml-infra:5000`、`axisml-cluster-manager.axisml-system:8080`。

---

## 2. 安装顺序

```sh
make cluster-up             # 拉起本地集群
make helm-install-infra     # 先装基础设施
make helm-install-system    # 再装控制平面（含数据库与 CRDs）
```

卸载顺序相反：`helm-uninstall-system` → `helm-uninstall-infra`。

`make helm-install-system` 内部先 `kubectl apply` `deploy/helm/axisml-system/crds/`，再做 `helm upgrade --install`——Helm 只在初次安装时处理 `crds/` 目录，CRD schema 升级靠这一步保证。

> **顺序约束**：`axisml-infra` 提供 CRDs 和组件（Koordinator、Envoy Gateway 等），`axisml-system` 依赖；安装颠倒会导致 Tenant / MLJob CR 创建时找不到 ElasticQuota / HTTPRoute kind 而失败。

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

### 3.2 `axisml-system` dependencies

| Dependency | 仓库 | condition | values 根键 |
| --- | --- | --- | --- |
| `postgresql`（aliased 为 `database`） | `oci://registry-1.docker.io/bitnamicharts` | `database.enabled` | `database` / `externalDatabase` |

> **fullnameOverride 约定**：`kube-prometheus-stack` 的 `fullnameOverride` 设为 `prometheus`，避开上游 26 字符截断；`postgresql` 别名为 `database`，使资源命名 `axisml-database-*`。

各子 chart 的具体 values 透传通过 values 根键直接写入，由各子 chart 自身文档定义；本文不重复。PostgreSQL 部署模式见 [database.md](database.md) / [infra.md §4.4](infra.md#44-数据库postgresql)。

---

## 4. 镜像版本与 IMAGE_TAG

所有控制面组件的镜像 tag 由 `deploy/helm/axisml-system/Chart.yaml` 的 `appVersion` 统一注入：

| 组件 | 镜像 |
| --- | --- |
| Cluster Manager | `ghcr.io/axisml/axisml-cluster-manager:<appVersion>` |
| Compute | `ghcr.io/axisml/axisml-compute:<appVersion>` |
| Artifacts | `ghcr.io/axisml/axisml-artifacts:<appVersion>` |
| Platform Backend | `ghcr.io/axisml/axisml-platform-backend:<appVersion>` |
| Platform Frontend | `ghcr.io/axisml/axisml-platform-frontend:<appVersion>` |
| tenant-operator | `ghcr.io/axisml/axisml-tenant-operator:<appVersion>` |
| compute-operator | `ghcr.io/axisml/axisml-compute-operator:<appVersion>` |

顶层 `Makefile` 的 `IMAGE_TAG` 变量从 `Chart.yaml` 读取并传给各组件 `make image` 目标，保证 chart 与镜像 tag 一致。

**Dev loop**：`make image-load IMAGE_TAG=dev` 后，在 values override 中覆盖 `image.tag=dev` 即可使用本地构建的镜像。详见 [CLAUDE.md "Image tag synchronization" 段](../../CLAUDE.md)。

---

## 5. Namespace 与控制面 Deployment

`axisml-system` namespace 内的控制面 Deployment：

| 组件 | 副本 | 端口 | leader election | 备注 |
| --- | --- | --- | --- | --- |
| Cluster Manager | `1` 默认 | `:8080` API、`:8080` metrics | controller-runtime Lease | 后台 reconciler / informer 只在 leader 副本运行 |
| Compute | `1` 默认 | `:8081` API、`:8080` metrics | controller-runtime Lease | 同上；API 层无状态可水平扩 |
| Artifacts | `1` 默认 | `:8082` API、`:8080` metrics | `coordination.k8s.io/Lease` | GC worker 选主；API 层无状态 |
| Platform Backend | `1` 默认 | `:8080` API、`:8081` metrics | 无 | 完全无状态 |
| Platform Frontend | `1` 默认 | `:80` 静态资源 | 无 | 后端镜像独立部署，通过 Helm `platform.frontend.image` 字段配置 |
| tenant-operator | `1`（leader）+ N 备 | `:8080` metrics | controller-runtime Lease | 单 leader |
| compute-operator | `1`（leader）+ N 备 | `:8080` metrics | controller-runtime Lease | 单 leader；dispatcher + handler 模型 |

`tenant namespaces` 由 tenant-operator 在 `Tenant` CR reconcile 时创建（详见 [tenant-operator.md §4.6.1](components/tenant-operator.md)），不在 Helm chart 内静态声明。

---

## 6. Helm 模板清单

每个控制面组件的 Helm 模板放在 `deploy/helm/axisml-system/templates/<component>/` 下，文件清单基本一致：

### 6.1 Cluster Manager / Compute / Artifacts / Platform Backend

| 文件 | 用途 |
| --- | --- |
| `configmap.yaml` | DB 连接、日志级别、下游 URL |
| `secret.yaml` | DB DSN、JWT 签名密钥（仅 Platform） |
| `deployment.yaml` | 主 Deployment，含探针 `/healthz` / `/readyz` |
| `service.yaml` | ClusterIP（cluster-manager / compute / artifacts）；HTTPRoute 仅 Platform 暴露外部入口 |
| `serviceaccount.yaml` | 服务账号 |
| `rbac.yaml` | ClusterRole + ClusterRoleBinding（详见各组件详设 §2.5） |
| `role.yaml` / `rolebinding.yaml` | leader election Lease 权限（在 `axisml-system` namespace 内） |
| `servicemonitor.yaml` | `/metrics` 暴露，kube-prometheus-stack 自动发现 |
| `bootstrap-job.yaml` | post-install Job 初始化默认数据（仅 Compute：default pool + cpu-small/cpu-medium unit） |

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
| `mljobs.axisml.io` | `crds/mljob-crd.yaml` | compute-operator |
| `mlservices.axisml.io` | `crds/mlservice-crd.yaml` | compute-operator |

`make helm-install-system` 先 `kubectl apply -f crds/`，再 `helm upgrade --install`——Helm 只在初次安装时处理 `crds/` 目录，schema 升级靠这一步。

---

## 7. PostgreSQL 部署模式

PostgreSQL 由 `axisml-system` chart 提供，支持两种模式：

| 模式 | 说明 | 适用场景 |
| --- | --- | --- |
| 内置 | bitnami/postgresql 子 chart（StatefulSet + PVC） | 开发、测试、轻量生产 |
| 外部 | 对接外部 PostgreSQL 实例（自建 / RDS） | 中大型生产 |

由 `database.enabled` 开关切换；外部模式通过 `externalDatabase.{host,port,database,username,existingSecret}` 配置。

所有控制面服务共用同一个 database `axisml`，按表名前缀逻辑隔离（详见 [database.md §1](database.md)）。

---

## 8. MVP 部署形态

目标：调用方能完整跑通"提交工作负载 → 调度 → 拉镜像 / 制品 → 写元数据 → 看监控"的最小闭环。

| 组件 | MVP 形态 |
| --- | --- |
| Envoy Gateway | 装入 `axisml-infra` chart；提供 `axisml-gateway`（HTTP listener）；listener `allowedRoutes.namespaces` 放行接入工作负载所在 namespace |
| RustFS | Standalone（单 Pod + PVC）；admin 凭证由 chart 自动生成 |
| zot | Standalone（filesystem 后端）；admin 凭证由 chart 自动生成；公共拉取 Secret 落地到 `axisml-system` namespace |
| PostgreSQL | 内置模式（bitnami chart，单节点 StatefulSet）；database `axisml`，由各调用方自管 schema 迁移 |
| GPU Operator | 默认配置（driver + container toolkit + device plugin + DCGM Exporter + GFD）；MIG 暂不启用 |
| Koordinator | `koord-scheduler` + `koord-manager` + ElasticQuota plugin + PodGroup CRD；其他可选组件全部关闭 |
| kube-prometheus-stack | Prometheus + Grafana + AlertManager 默认部署；ServiceMonitor 自动发现已就绪；不预置告警规则 |
| Cluster Manager / Compute / Artifacts / Platform / tenant-operator / compute-operator | `replicas=1`；上述 Helm 模板清单 §6 全部启用；bootstrap Job 初始化默认数据 |

MVP 不含的能力：HTTPS / TLS、SecurityPolicy 实施、BackendTrafficPolicy 限流、对象存储 / OCI registry 的 HA、外部 PostgreSQL、自定义 Grafana dashboard、告警规则、MIG。

---

## 9. 生产硬化（阶段二）

### 9.1 网关与安全

- `axisml-gateway` 增加 HTTPS listener；TLS 证书通过 `cert-manager` 或 Secret 注入；
- Gateway 级 `SecurityPolicy`（JWT / OIDC，对接调用方选定的 IdP）；
- 静态 HTTPRoute 增加 `BackendTrafficPolicy`（限流 + 熔断 + 重试）；
- 跨 namespace `ReferenceGrant` 模板（仅在跨 namespace `backendRef` 出现时启用）。

### 9.2 对象存储 / OCI Registry

- RustFS 切换到 Distributed（4×4）部署；
- zot 切换到 HA（3×）+ 共享后端（S3 兼容存储或 RustFS）；
- zot 增加 GC、垃圾清理 CronJob、scrub 配置。

### 9.3 数据库

- 支持外部 PostgreSQL（`database.enabled=false` + `externalDatabase.*`）；提供示例 values 与连接 Secret 模板；
- 备份策略（CronJob + 远端对象存储）。

### 9.4 监控告警

- 预置 AlertManager 告警规则：节点 NotReady、GPU 异常、PVC 容量、配额耗尽、调度滞后、API 错误率（详见 [monitoring.md §8](monitoring.md#8-告警)）；
- 持久化 Prometheus 数据卷（默认 14 天保留期，可调）；
- 自定义 Grafana dashboard（控制面 + 业务）。

### 9.5 Operator HA

- tenant-operator / compute-operator 多副本 leader election；
- Compute / Cluster Manager / Artifacts 多副本（API 层水平扩，后台协程仍 leader-only）。

---

## 10. 关键设计决策

| 决策项 | 决策 | 理由 |
| --- | --- | --- |
| Chart 拆分 | `axisml-infra` / `axisml-system` 两个 chart | 基础设施和控制平面发版节奏、回滚粒度不同；infra 可共享给多套 axisml-system 实例 |
| 数据库归属 | 纳入 `axisml-system` 控制平面 chart | 数据库的生命周期、迁移、备份和控制面服务紧密耦合；同 namespace 可共享 Secret、ServiceMonitor，减少跨 chart 引用 |
| CRD 安装 | `kubectl apply -f crds/` + `helm upgrade --install` 组合 | Helm 只在初次安装时处理 `crds/`；apply 一步保证 schema 升级 |
| 镜像 tag 来源 | 由 `axisml-system/Chart.yaml` `appVersion` 注入 | 保证 chart 与镜像 tag 一致；避免 Helm rendered Deployment 拉不到镜像 |
| Operator 形态 | tenant-operator + compute-operator 拆为两个独立二进制 | 管理员域与业务域按变更频率与权限边界分离；任一 operator 演进或重启不影响另一域 |
| Namespace 命名 | `axisml-system` / `axisml-infra` + 租户 namespace 由 tenant-operator 派生 | 系统组件与租户工作负载隔离；租户 namespace 名由 Tenant CR `spec.namespace.name` 决定 |
| 外部 PostgreSQL | 通过 `database.enabled=false` + `externalDatabase.*` 切换 | 生产推荐外接 RDS；MVP 用内置 bitnami chart 简化本地起步 |

---

## 11. 关联文档

- [overview.md §6 部署架构](overview.md#6-部署架构)：总体部署示意；
- [infra.md §5 部署形态](infra.md#5-部署形态)：infra chart 子组件部署细节；
- [database.md §1 部署形态](database.md)：PostgreSQL 部署形态；
- [monitoring.md §1 接入模型](monitoring.md#1-接入模型)：ServiceMonitor 接入；
- 各组件详设的 §2.6 Helm values 章节：[cluster-manager](components/cluster-manager.md) / [compute](components/compute.md) / [artifacts](components/artifacts.md) / [tenant-operator](components/tenant-operator.md) / [compute-operator](components/compute-operator.md) / [platform §11 部署架构](components/platform.md#11-部署架构)。
