# AxisML Infra 详细设计

## 1. 概述

AxisML Infra 是平台的基础设施层，由一系列开源组件组成，为上层应用组件（Platform、Compute、Artifacts、Operators）提供底层支撑能力。`axisml-infra` Helm chart 主要通过 dependencies 管理第三方组件，同时包含必要的 Gateway API 路由、跨命名空间授权、Secret / ConfigMap 等 glue 模板；元数据数据库随 `axisml-system` 控制平面 chart 管理。

### 1.1 组件清单

| # | 组件 | 技术选型 | 职责 |
| --- | --- | --- | --- |
| 1 | 服务网关 | Envoy Gateway | 请求路由、认证鉴权、流量控制 |
| 2 | 对象存储 | RustFS | 数据集、评估报告等基于 S3 协议的制品文件持久化 |
| 3 | OCI Registry | zot | 模型、容器镜像等基于 OCI Distribution 协议的制品存储 |
| 4 | 数据库 | PostgreSQL（bitnami chart） | 元数据持久化存储 |
| 5 | GPU 管理 | NVIDIA GPU Operator | GPU 驱动、设备插件与监控 |
| 6 | 调度与配额 | Koordinator | koord-scheduler 接管所有 AxisML workload；ElasticQuota 多租户配额；PodGroup gang scheduling |
| 7 | 监控 | kube-prometheus-stack | 集群与业务可观测性 |

## 2. 整体架构

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                              AxisML Infra                                    │
│                                                                              │
│  ┌──────────────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────┐  │
│  │      服务网关         │  │   对象存储    │  │ OCI Registry │  │  数据库   │  │
│  │   Envoy Gateway      │  │    RustFS    │  │     zot      │  │PostgreSQL│  │
│  │   (Gateway API)      │  │   (S3 API)   │  │ (Distribution│  │ (bitnami)│  │
│  │                      │  │              │  │     v2)      │  │          │  │
│  └──────────────────────┘  └──────────────┘  └──────────────┘  └──────────┘  │
│                                                                              │
│  ┌──────────────────────┐  ┌──────────────────────────────────────────────┐  │
│  │      GPU 管理         │  │              调度与配额                      │  │
│  │ NVIDIA GPU Operator  │  │              Koordinator                     │  │
│  │                      │  │   koord-scheduler + ElasticQuota + PodGroup  │  │
│  └──────────────────────┘  └──────────────────────────────────────────────┘  │
│                                                                              │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │                             监控                                      │   │
│  │                    kube-prometheus-stack                             │   │
│  │          (Prometheus + Grafana + AlertManager)                       │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────────────────────┘
```

调用关系要点：

- 外部流量 → **Envoy Gateway** → Platform / Artifacts 等对外 Service；Compute 仅由 Platform 通过集群内 Service 调用
- Compute / Artifacts → **PostgreSQL**（元数据读写）
- Artifacts → **RustFS**（dataset / eval_report 制品文件读写，S3 API）
- Artifacts / axisml-cli → **zot**（model / image 制品 push / pull，OCI Distribution v2）
- mlservice-operator / mljob-operator 派生的 Pod → **zot**（按 imagePullSecret 拉取镜像 / 模型）
- 任何 AxisML workload Pod（jobs + services，含 KServe 派生 Pod）→ **Koordinator**（`schedulerName: koord-scheduler` + Pod label `quota.scheduling.koordinator.sh/name=axisml-<tenant>-<pool>-<quota>` 消费 ElasticQuota）
- 所有 Pod（含 GPU Operator 的 DCGM Exporter、网关、业务组件）→ **kube-prometheus-stack**（`/metrics` 被 ServiceMonitor 自动发现）
- 业务 Pod 申请 `nvidia.com/gpu` → **GPU Operator** 完成设备分配

## 3. 服务网关（Envoy Gateway）

AxisML 采用 [Envoy Gateway](https://gateway.envoyproxy.io/) 作为服务网关，基于 Kubernetes [Gateway API](https://gateway-api.sigs.k8s.io/) 标准以声明式方式配置路由、认证与流量控制。

### 3.1 架构设计

Gateway API 资源模型：

```
GatewayClass (envoy-gateway)
  │
  └── Gateway (axisml-gateway)
        │  Listener: HTTP (80)
        │  Listener: HTTPS (443)
        │
        ├── HTTPRoute (platform)
        │     pathPrefix: /
        │     → axisml-platform Service
        │
        └── HTTPRoute (artifacts-api)
              pathPrefix: /api/artifacts
              → axisml-artifacts Service
```

- **GatewayClass**：由 Envoy Gateway 控制面注册，声明控制器实现。
- **Gateway**：集群内"监听点"的声明，同一份 Gateway 实例承载全部 AxisML 路由。
- **HTTPRoute**：对外业务组件的路由规则，与 Service 绑定；Compute 不配置外部 HTTPRoute。

**MLService 派生路由**：mlservice-operator 在租户 namespace 内为开启了 `spec.route.enabled=true` 的 MLService 创建 namespaced `HTTPRoute` / `SecurityPolicy` / `BackendTrafficPolicy`，`parentRefs` 指向 `axisml-infra` namespace 下的同一份 `axisml-gateway`；Gateway listener 通过 `allowedRoutes.namespaces` 放行租户 namespace 的 Route 挂载。`ReferenceGrant` 仅用于跨 namespace backendRef 等被引用对象授权场景，本路径下 HTTPRoute、SecurityPolicy、BackendTrafficPolicy 与目标 Service 保持同 namespace。与 platform / artifacts 的静态 HTTPRoute 共存。详见 [operator.md#6-mlservice-controller](operator.md#6-mlservice-controller) §3 / §6 / §8.1。

### 3.2 认证鉴权

通过 Envoy Gateway 的 `SecurityPolicy` CRD 实现，可附加到 Gateway 或 HTTPRoute 级别：

| 能力 | 说明 |
| --- | --- |
| JWT 验证 | 校验请求头 JWT，支持配置 issuer 与 JWKS 端点 |
| OIDC 集成 | 支持 OpenID Connect，可对接外部身份提供商 |
| ExtAuth | 外部授权服务，支持自定义鉴权逻辑 |
| per-Service 认证 | MLService 通过 `spec.route.auth` 声明本服务的 JWT / API key 策略，由 mlservice-operator 翻译为 namespaced `SecurityPolicy`，`targetRefs` 指向该 MLService 的 HTTPRoute；与 Gateway 级 SecurityPolicy 叠加生效（policy attachment 语义详见 Envoy Gateway 文档） |

具体认证方案（如对接的 IdP）留待 Platform 设计文档确定，Infra 层只保证能力就位。

### 3.3 流量控制

通过 Envoy Gateway 的 `BackendTrafficPolicy` CRD 实现：

| 能力 | 说明 |
| --- | --- |
| 限流 | 支持全局限流和按路由限流 |
| 熔断 | 后端异常时自动熔断，防止级联故障 |
| 超时 / 重试 | 配置请求超时与重试策略 |
| 负载均衡 | Round Robin / Least Request 等算法 |

### 3.4 部署形态

作为 `axisml-infra` Helm chart 的 dependency 引入：

```yaml
# deploy/helm/axisml-infra/Chart.yaml
dependencies:
  - name: gateway-helm
    alias: envoy-gateway
    version: v1.3.x
    repository: oci://docker.io/envoyproxy
    condition: envoy-gateway.enabled
```

values.yaml 对应段：

```yaml
envoy-gateway:
  enabled: true
  # Envoy Gateway 子 chart 的 values pass-through
```

AxisML 自身的 `Gateway` / 静态 `HTTPRoute` / Gateway 级策略与 listener `allowedRoutes` 由 `axisml-infra` chart 的 gateway 模板提供（本文档定义设计，具体模板由后续 PR 落地）。

## 4. 对象存储（RustFS）

AxisML 使用 [RustFS](https://rustfs.dev/) 作为对象存储，用于数据集、评估报告等 S3 类制品文件的持久化。RustFS 是 Apache 2.0 许可证、基于 Rust 实现、S3 API 兼容的高性能对象存储。

### 4.1 架构设计

RustFS 提供标准 S3 API（`PutObject` / `GetObject` / `DeleteObject` / `ListObjects` / Presigned URL 等），部署支持：

| 模式 | 说明 | 适用场景 |
| --- | --- | --- |
| Standalone | 单 Pod + 单 PVC | 开发、测试 |
| Distributed (4x4) | 4 Pod × 4 PVC | 中等规模生产 |
| Distributed (16x1) | 16 Pod × 1 PVC | 大规模生产 |

### 4.2 用途

- **Artifacts**：dataset / eval_report 类制品（按 prefix 组织、整目录上传）；详见 [artifacts.md §6.3.2 / §6.3.4](artifacts.md)
- **未来**：日志归档

业务组件通过 S3 SDK 访问，对 RustFS 与其他 S3 兼容实现无感知。容器镜像 / 模型权重的 OCI Distribution 协议存储**不**走 RustFS，而是走 §5 zot——OCI 对 ML artifact 的 manifest / artifactType 语义、按 layer 复用与 multi-arch 支持是 RustFS 的 S3 协议无法提供的。

### 4.3 部署形态

作为 `axisml-infra` Helm chart 的 dependency：

```yaml
# deploy/helm/axisml-infra/Chart.yaml
dependencies:
  - name: rustfs
    version: 0.0.9x
    repository: https://charts.rustfs.com
    condition: rustfs.enabled
```

values.yaml 对应段：

```yaml
rustfs:
  enabled: true
  # rustfs 子 chart 的 values pass-through
```

> **成熟度说明**：截至 2026-04，RustFS 的 app version 为 `1.0.0-alpha.x`，项目仍在活跃迭代。本次选型在"关键设计决策"中已记录风险与切换方案（S3 API 抽象使切换成本有限）。

## 5. OCI Registry（zot）

AxisML 使用 [zot](https://zotregistry.dev/) 作为 OCI Distribution v2 兼容的制品仓库，承载模型权重（`Kind=model`）与容器镜像（`Kind=image`）的存储与分发。zot 是 CNCF Sandbox 项目，Apache 2.0 许可证、单二进制 Go 实现，对 OCI 1.1 artifact manifest（含 `artifactType`）支持完整。

### 5.1 架构设计

zot 提供完整的 OCI Distribution v2 协议（`/v2/<repo>/blobs/uploads/`、`/v2/<repo>/manifests/<ref>` 等），关键能力：

| 能力 | 说明 |
| --- | --- |
| OCI artifact manifest | 原生支持 `application/vnd.oci.image.manifest.v1+json` + `artifactType`，承载 ML 模型类非容器制品 |
| 内容寻址 | `<repo>@sha256:<digest>` 不可变引用，对应 [artifacts.md §5.3](artifacts.md) `?pin=digest` 形态 |
| 后端可插拔 | 本地 filesystem / S3 兼容存储；未来可把 blob 后端切到 RustFS 实现 OCI metadata + S3 blobs 双层架构 |
| Bearer token 鉴权 | 支持 scope-limited bearer token（`repository:<repo>:push` / `pull`），由 Artifacts 服务签发后下发给 axisml-cli |
| Manifest 校验 | `HEAD /v2/<repo>/manifests/<ref>` 返回 digest，Artifacts 在 complete API 阶段比对 cli 提交值（[artifacts.md §5.2](artifacts.md)） |

部署形态：

| 模式 | 说明 | 适用场景 |
| --- | --- | --- |
| Standalone | 单 Pod + 单 PVC（filesystem 后端） | 开发、测试、Lite |
| HA (3x) | 3 Pod + 共享后端（S3 / RustFS） | 中等规模生产 |

### 5.2 用途

- **Artifacts（model Kind）**：模型权重作为 OCI artifact 存储，`artifactType` 由 `spec.format` 携带（如 `application/vnd.axisml.model.pytorch.v1+tar`）
- **Artifacts（image Kind）**：训练 / 推理容器镜像，标准 docker push / nerdctl push 即可
- **Operator 拉取**：mlservice-operator / mljob-operator 派生的 Pod 通过 K8s `imagePullSecrets` 直接拉取（凭证由 tenant-operator 落地，详见 [artifacts.md §5.3 / §8](artifacts.md)）

业务组件通过 OCI Distribution v2 客户端（`oras` / `crane` / docker daemon）访问，对 zot 与其他 OCI 兼容实现无感知。

### 5.3 与 Artifacts / tenant-operator 的接入契约

zot 目标上由 `axisml-infra` chart 提供为纯协议端，本身不感知 AxisML 的租户模型；租户 / 命名隔离由 Artifacts 在 repo 路径上完成，鉴权由 Artifacts 签发的 scope token 表达：

| 资源 | 落点 | 由谁维护 |
| --- | --- | --- |
| zot endpoint | ConfigMap（Artifacts 注入） | axisml-infra Helm |
| zot admin 凭证（Artifacts 用于校验 / GC / 签 scope token） | 平台级 Secret，挂入 Artifacts Pod | axisml-infra Helm（自动生成 / 由管理员预置） |
| 租户拉取凭证（`axisml-tenant-<tenant>-zot-pull`） | 租户 Namespace Secret | tenant-operator 按 `Tenant.spec.initResources.imagePullSecrets[].name='zot-pull'` 落地（[tenant-operator §6.3](operator.md#4-tenant-controller)） |
| 公共拉取凭证（`zot-pull@axisml-system`） | `axisml-system` Namespace Secret | axisml-infra Helm |
| repo 路径命名（`<scope>/<kind>/<repo>`） | URI 由 Artifacts handler 即时构造 | Artifacts |

zot 不需要任何 AxisML 自定义扩展，所有定制都在 Artifacts 服务侧完成。

### 5.4 部署形态

作为 axisml-infra Helm chart 的 dependency：

```yaml
# deploy/helm/axisml-infra/Chart.yaml
dependencies:
  - name: zot
    version: 0.1.x                          # https://artifacthub.io/packages/helm/zot/zot
    repository: https://zotregistry.dev/helm-charts
    condition: zot.enabled
```

values.yaml 对应段：

```yaml
zot:
  enabled: true
  # zot 子 chart 的 values pass-through；后端默认 filesystem，HA 场景切 s3
```

> **落地状态说明**：zot 是 Infra 目标架构中的 OCI Registry 后端；当前 `deploy/helm/axisml-infra/Chart.yaml` 仍需补齐该 dependency 与 values，本文保留目标设计以和 Artifacts 的 model / image 存储契约保持一致。

> **与 RustFS 的关系**：当前形态下 zot 使用本地 filesystem 作为 blob 后端，与 RustFS 数据通道无耦合；后续若引入"zot metadata + RustFS blobs"双层架构，可把 zot 的 storage backend 配置成 S3 协议指向 RustFS，从而把所有制品 bytes 物理上汇聚到 RustFS、由 zot 维护 OCI 协议层。

## 6. 数据库（PostgreSQL）

AxisML 使用 PostgreSQL 作为元数据存储，供 Compute、Artifacts 等 Go 组件持久化结构化数据。

### 6.1 架构设计

支持两种部署模式，由 `database.enabled` 开关切换：

| 模式 | 说明 | 适用场景 |
| --- | --- | --- |
| 内置模式 | 通过 bitnami/postgresql 子 chart 部署（StatefulSet + PVC） | 开发、测试、轻量生产 |
| 外部模式 | 对接外部 PostgreSQL 实例（自建或 RDS） | 中大型生产 |

内置模式采用 bitnami/postgresql 官方 chart，避免自写 StatefulSet 模板——这也是此前自写模板 `database-statefulset.yaml` / `database-service.yaml` 被删除、改由子 chart 提供的原因。

### 6.2 消费方

| 消费方 | 使用场景 |
| --- | --- |
| AxisML Compute | 租户、资源单元、任务元数据 |
| AxisML Artifacts | 模型、镜像、数据集元数据 |

各消费方通过独立 database 或独立 schema 逻辑隔离（具体隔离粒度由各组件设计文档定义）。

### 6.3 部署形态

```yaml
# deploy/helm/axisml-system/Chart.yaml
dependencies:
  - name: postgresql
    alias: database
    version: 16.x.x
    repository: oci://registry-1.docker.io/bitnamicharts
    condition: database.enabled
```

values.yaml 对应段（bitnami pass-through 字段命名）：

```yaml
database:
  enabled: true          # 内置模式开关；false 时使用 externalDatabase
  auth:
    database: axisml
    username: axisml
    password: axisml     # 生产环境应使用 existingSecret
  primary:
    persistence:
      size: 10Gi

# 外部模式（database.enabled=false 时生效）
externalDatabase:
  host: ""
  port: 5432
  database: axisml
  username: axisml
  existingSecret: ""
```

## 7. GPU 管理（NVIDIA GPU Operator）

AxisML 使用 [NVIDIA GPU Operator](https://github.com/NVIDIA/gpu-operator) 管理集群 GPU 资源，自动化驱动、设备插件、监控等组件的生命周期。

### 7.1 组件架构

| 组件 | 职责 |
| --- | --- |
| GPU Driver Container | 容器化 NVIDIA 驱动，自动安装与升级 |
| NVIDIA Container Toolkit | 容器运行时集成，使容器可访问 GPU |
| Device Plugin | 向 kube-scheduler / koord-scheduler 报告 `nvidia.com/gpu` 资源 |
| DCGM Exporter | 导出 GPU 利用率、显存、温度等 Prometheus 指标 |
| GPU Feature Discovery | 自动为节点打标签（GPU 型号、驱动版本等） |
| MIG Manager | A100/H100 的多实例分区管理 |

### 7.2 调度契约

Infra 层对上层的契约：

- 业务 Pod 申请 GPU 时使用资源名 `nvidia.com/gpu`
- 节点标签可基于 `nvidia.com/gpu.product`（如 `A100-SXM4-80GB`）做 nodeSelector / affinity
- DCGM Exporter 的 `/metrics` 端点由 kube-prometheus-stack 自动采集（详见 §9）

### 7.3 部署形态

```yaml
# deploy/helm/axisml-infra/Chart.yaml
dependencies:
  - name: gpu-operator
    version: v24.x.x
    repository: https://helm.ngc.nvidia.com/nvidia
    condition: gpu-operator.enabled
```

values.yaml 对应段：

```yaml
gpu-operator:
  enabled: true
  # gpu-operator 子 chart 的 values pass-through
  # 如 driver.enabled / dcgmExporter.enabled 等
```

## 8. 调度与配额（Koordinator）

AxisML 使用 [Koordinator](https://koordinator.sh/) 作为统一调度器与多租户配额引擎，与默认 kube-scheduler 按 `schedulerName` 共存。**所有 AxisML workload Pod**（MLJob、MLService 各 backend 派生的 Pod，含 KServe 派生 Pod）都强制走 koord-scheduler 并消耗对应 ElasticQuota；只有控制平面 Pod 留在默认 kube-scheduler 上。

### 8.1 组件架构

| 组件 | 职责 |
| --- | --- |
| koord-scheduler | 自定义调度器，承载 Gang Scheduling 与 ElasticQuota plugin；scheduler 名 `koord-scheduler` |
| koord-manager | Koordinator 控制器集合，管理 ElasticQuota / PodGroup 等 CR 状态聚合 |
| koord-descheduler（可选） | 在线服务弹性场景下做 Pod 重平衡；当前不启用 |
| koordlet（可选） | 节点侧 agent，用于在/离线协同 / QoS / 弹性资源；当前不启用 |

### 8.2 核心能力

| 能力 | 说明 |
| --- | --- |
| **Gang Scheduling** | 通过 sigs.k8s.io scheduler-plugins `PodGroup` (`scheduling.sigs.k8s.io/v1alpha1`) 表达；同一 PodGroup 的全部 Pod 要么同时调度，要么都不调度——避免分布式训练中部分 Worker 启动造成的资源死锁 |
| **ElasticQuota** | `scheduling.sigs.k8s.io/v1alpha1` `ElasticQuota`（namespace-scoped）承载 `min` / `max`；Pod 通过 label `quota.scheduling.koordinator.sh/name=<eq-name>` 关联到所属 ElasticQuota。AxisML 不引入 Koordinator 私有的 `shared-weight` annotation，借用容量分配按 koord-scheduler 默认平权处理，让 CR 字段集与上游 scheduler-plugins ElasticQuota 一一对应 |
| **Preemption / Reclaim** | 已分配但低于其他 ElasticQuota `min` 的资源可被回收；高于 `max` 的请求一律拒绝调度 |
| **Backfill** | 空闲资源回填，提升集群利用率 |
| **CoLocation / QoS**（可选） | 在/离线混部、CPU 预算管理；当前不启用，作为未来演进 |

### 8.3 与 MLJob / MLService 的协作契约

本文档定义 Infra 侧契约，具体实现细节见 `operators/`：

- **Quota 全覆盖（系统级硬不变式）**：任何 AxisML workload Pod 都必须设置 `schedulerName: koord-scheduler` 并携带 label `quota.scheduling.koordinator.sh/name=axisml-<tenant>-<pool>-<quota>`，不允许"绕过 quota 的调度路径"。MLJob / MLService 的所有 backend handler 都必须在 podSpec 模板上注入这两个字段；KServe 路径通过 `InferenceService.spec.predictor.schedulerName` + `spec.predictor.labels` 注入（KServe `PredictorSpec` 内联 `corev1.PodSpec` 与 `ComponentExtensionSpec`，所以两者都是 `spec.predictor` 的直接字段），依赖 KServe 把它们透传到派生 Pod 的 `spec.schedulerName` 与 `metadata.labels`。
- **Gang scheduling 仅在需要的 backend 启用**：MLJob `(native, podgroup)` / `(kubeflow-trainer, *)` 创建 PodGroup CR；MLJob `(native, job)`、MLService `(native, deployment)` / `(native, statefulset)` / `(kserve, *)` 不创建 PodGroup（gang 不适合非分布式训练 / 常驻服务），但仍走 koord-scheduler，仅通过 quota label 计入 ElasticQuota.
- **ElasticQuota CR 由 tenant-operator 独占 owner**：Compute 把 PG `quotas` 表渲染进 `Tenant.spec.quotas[]`，由 tenant-operator 派生 ElasticQuota CR（`spec.min` / `spec.max`、命名、补偿、RBAC 均归 tenant-operator）；mljob-operator / mlservice-operator 仅通过 Pod label `quota.scheduling.koordinator.sh/name` 引用 ElasticQuota，不读写 ElasticQuota CR。配额与租户 / 资源池的归属关系由 Compute 在 PG 中维护，命名约定 `axisml-<tenant>-<pool>-<quota>`，CR 落在租户 namespace 下，详见 [compute.md §7](compute.md) 与 [tenant-operator §4.6.2](operator.md#4-tenant-controller)
- **tenant-operator** 派生 Namespace、ElasticQuota、initResources（Secret / ConfigMap / SA / RBAC），是 Tenant CR 与 K8s 之间双向数据链路的承载
- **PodGroup CR** 由对应 backend handler 在租户 namespace 内自管，与 ElasticQuota 解耦

**KServe 版本要求**：`(kserve, *)` MLService 路径要求 KServe 版本支持 `InferenceService.spec.predictor.schedulerName` 与 `spec.predictor.labels` 字段透传到派生 Pod（`PredictorSpec` 内联 `corev1.PodSpec` 与 `ComponentExtensionSpec` 是 KServe v1beta1 的标准契约）。安装时必须 pin 一个已知支持该字段的 KServe stable 版本；详细的最低版本约束在 [operator.md#6-mlservice-controller §8.3](operator.md#6-mlservice-controller) 维护。运行时若发现 KServe 不透传则视为阻塞 bug，由升级 KServe 解决，不引入兜底 webhook。

### 8.4 与 kube-scheduler 共存

koord-scheduler 仅接管设置了 `schedulerName: koord-scheduler` 的 Pod。Infra 自身（网关、对象存储、监控、Koordinator 自身、GPU Operator）以及 Platform / Compute / Artifacts / 各 Operator / 数据库等控制平面 Pod 的 podSpec 不设置 `schedulerName`，自然走默认 kube-scheduler，**不消耗** ElasticQuota，也与 koord-scheduler 互不干扰。

### 8.5 部署形态

```yaml
# deploy/helm/axisml-infra/Chart.yaml
dependencies:
  - name: koordinator
    version: 1.5.x                                # pin 到上游 stable
    repository: https://koordinator-sh.github.io/charts
    condition: koordinator.enabled
```

values.yaml 对应段：

```yaml
koordinator:
  enabled: true
  # koordinator 子 chart 的 values pass-through
  # scheduler 名固定为 koord-scheduler；所有 AxisML workload Pod 必须设置
  # schedulerName: koord-scheduler 才能被接管并消费 ElasticQuota。
```

## 9. 监控（kube-prometheus-stack）

AxisML 使用 [kube-prometheus-stack](https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-prometheus-stack) 作为统一监控栈，包含 Prometheus、Grafana、AlertManager 三件套。

### 9.1 架构

```
┌──────────────────────────────────────────────────────────────┐
│                     kube-prometheus-stack                    │
│                                                              │
│  ┌─────────────┐  ┌─────────────┐  ┌──────────────────────┐  │
│  │ Prometheus  │  │   Grafana   │  │    AlertManager      │  │
│  │ 指标采集/存储│  │  可视化看板  │  │  告警通知            │  │
│  └─────────────┘  └─────────────┘  └──────────────────────┘  │
└──────────────────────────────────────────────────────────────┘
         ▲
         │ ServiceMonitor / PodMonitor（CRD 自动发现）
         │
    各组件 /metrics 端点
```

### 9.2 采集模型

各 AxisML 组件只需：

1. 在容器内暴露 `/metrics` 端点（Prometheus 格式）
2. 随 Helm chart 提供对应的 `ServiceMonitor` CRD，声明待采集的 Service 与端口

kube-prometheus-stack 的 Prometheus Operator 会自动发现并配置采集目标，无需手动维护 `prometheus.yml`。

### 9.3 指标体系

| 层级 | 来源 | 典型指标 |
| --- | --- | --- |
| 集群层 | kube-state-metrics、node-exporter | 节点 CPU/内存/磁盘、Pod 状态 |
| GPU 层 | DCGM Exporter（来自 GPU Operator） | GPU 利用率、显存占用、温度、功耗 |
| 网关层 | Envoy Gateway | 请求量、延迟分位、错误率 |
| 调度层 | koord-scheduler / koord-manager | ElasticQuota 用量与借用、PodGroup 调度状态、调度延迟 |
| 业务层 | Platform / Compute / Artifacts / Operators | 任务状态、推理延迟、制品数量等自定义指标 |

### 9.4 部署形态

```yaml
# deploy/helm/axisml-infra/Chart.yaml
dependencies:
  - name: kube-prometheus-stack
    alias: kube-prometheus-stack
    version: 6x.x.x
    repository: https://prometheus-community.github.io/helm-charts
    condition: kube-prometheus-stack.enabled
```

values.yaml 对应段：

```yaml
kube-prometheus-stack:
  enabled: true
  # kube-prometheus-stack 子 chart 的 values pass-through
```

## 10. 部署总览

### 10.1 Helm chart 组织

AxisML 拆分为两个独立的 Helm chart，按"基础设施 / 控制平面"职责分层部署：

| Chart | 路径 | Release | Namespace | 资源命名前缀 |
| --- | --- | --- | --- | --- |
| `axisml-infra` | `deploy/helm/axisml-infra` | `axisml-infra` | `axisml-infra` | `axisml-infra-*` |
| `axisml-system` | `deploy/helm/axisml-system` | `axisml` | `axisml-system` | `axisml-*` |

**axisml-infra** 的依赖：

| Dependency | 仓库 | condition |
| --- | --- | --- |
| gateway-helm | oci://docker.io/envoyproxy | `envoy-gateway.enabled` |
| rustfs | https://charts.rustfs.com | `rustfs.enabled` |
| zot（目标项；当前 Chart 待补齐） | https://zotregistry.dev/helm-charts | `zot.enabled` |
| gpu-operator | https://helm.ngc.nvidia.com/nvidia | `gpu-operator.enabled` |
| koordinator | https://koordinator-sh.github.io/charts | `koordinator.enabled` |
| kube-prometheus-stack | https://prometheus-community.github.io/helm-charts | `kube-prometheus-stack.enabled` |

**axisml-system** 的依赖（数据库归控制平面管理）：

| Dependency | 仓库 | condition |
| --- | --- | --- |
| postgresql（aliased 为 `database`） | oci://registry-1.docker.io/bitnamicharts | `database.enabled` |

通过 `condition` 字段保证组件可按需启停——例如对接外部 PostgreSQL 时关闭 `database.enabled`，对接现有 Prometheus 时关闭 `kube-prometheus-stack.enabled`。

### 10.2 命名空间约定

- `axisml-infra` 命名空间承载第三方基础设施子 chart 的全部资源；`make helm-install-infra` 默认值。
- `axisml-system` 命名空间承载 AxisML 自研组件（Platform/Compute/Artifacts/Operators）以及元数据数据库 `axisml-database`；`make helm-install-system` 默认值。
- 跨命名空间访问走 `<service>.<namespace>.svc.cluster.local`，例如 Artifacts 调用 RustFS：`rustfs-svc.axisml-infra:9000`、Artifacts 调用 zot：`zot.axisml-infra:5000`。

### 10.3 安装顺序

```
make cluster-up             # 拉起本地集群
make helm-install-infra     # 先装基础设施（监控栈 + 网关 + GPU + Koordinator + 对象存储 + OCI Registry）
make helm-install-system    # 再装控制平面（平台组件 + Operators + CRDs + 数据库）
```

卸载顺序相反：`helm-uninstall-system` → `helm-uninstall-infra`。

### 10.4 与 values.yaml 的对应关系

axisml-infra/values.yaml：

| 组件 | values 根键 | 说明 |
| --- | --- | --- |
| 服务网关 | `envoy-gateway` | Envoy Gateway |
| 对象存储 | `rustfs` | RustFS |
| OCI Registry | `zot` | zot（默认 filesystem 后端，可切 S3 指向 RustFS） |
| GPU 管理 | `gpu-operator` | NVIDIA GPU Operator |
| 调度与配额 | `koordinator` | Koordinator（scheduler 名固定为 `koord-scheduler`） |
| 监控 | `kube-prometheus-stack` | kube-prometheus-stack（`fullnameOverride` 设为 `prometheus`，避开上游 26 字符截断） |

axisml-system/values.yaml：

| 组件 | values 根键 | 说明 |
| --- | --- | --- |
| 数据库 | `database` / `externalDatabase` | PostgreSQL（内置 `axisml-database` / 外接二选一） |

## 11. 关键设计决策

| 决策项 | 决策 | 理由 |
| --- | --- | --- |
| 服务网关 | Envoy Gateway | Gateway API 是 Kubernetes 官方的 Ingress 继任者，Envoy 原生支持 gRPC/HTTP2；配置完全 CRD 化，零外部依赖 |
| 对象存储 | RustFS | Apache 2.0 许可证，S3 兼容；规避 MinIO 自 2021 年转为 AGPLv3 的商用传染风险。风险：项目较年轻（alpha 阶段），社区规模小于 MinIO；S3 API 抽象使切换成本有限 |
| OCI Registry | zot | OCI Distribution v2 + 1.1 artifact manifest 原生支持，对 ML 模型类非容器制品的 `artifactType` 语义完整；CNCF Sandbox、单二进制 Go 实现、可选 S3 后端，与 Artifacts "OCI 协议层 + RustFS bytes 层"演进路径相容。规避自建 Harbor / 复用公共 registry 的多种问题（多租户隔离、内网部署、无 ML artifact 语义） |
| 制品分流 | OCI Distribution（zot）走 model / image，S3（RustFS）走 dataset / eval_report | OCI 的 artifact manifest + 内容寻址契合模型权重与镜像的不可变引用语义；S3 的整目录上传 / 流式访问契合数据集与报告的"前缀寻址 + 多文件"语义。两条数据通道由 Artifacts 的 `ArtifactHandler` 按 Kind 路由（[artifacts.md §6.3](artifacts.md)） |
| 数据库 | bitnami/postgresql 子 chart | 复用成熟 chart，避免自写 StatefulSet 模板带来的维护负担；`externalDatabase` 段保留用于生产外接 RDS |
| 数据库归属 | 纳入 axisml-system 控制平面 chart | 数据库的生命周期、迁移、备份都和业务组件紧密耦合；与 Platform/Compute/Artifacts 放同一命名空间可共享 Secret、ServiceMonitor，减少跨 chart 引用 |
| GPU 管理 | NVIDIA GPU Operator | Kubernetes 原生 GPU 管理事实标准；DCGM Exporter 与监控栈天然集成 |
| 调度与配额 | Koordinator | sigs.k8s.io scheduler-plugins ElasticQuota 提供 namespace-scoped `min` / `max` 多租户配额模型（不引入 Koordinator 私有 annotation，保持与上游 CR 字段一一对应），PodGroup 提供 Gang Scheduling，二者由统一 koord-scheduler 承载；与 kube-scheduler 按 `schedulerName` 共存，零副作用。ElasticQuota 直接以 namespace 表达租户边界，并通过 Pod label 关联，MLService 路径只需 Pod label 计入配额，无需引入额外的 PodGroup |
| 调度归属 | 纳入 Infra 层 | Koordinator 与 GPU Operator 一样属于训练 / 推理底座能力；作为第三方基础设施组件独立于 AxisML 自研控制平面部署 |
| Quota 全覆盖 | 所有 AxisML workload Pod 强制走 koord-scheduler | 任何 job / service 都要消耗 quota，避免"绕过 quota 的调度路径"；KServe 派生 Pod 通过 `spec.predictor.schedulerName` 与 `spec.predictor.labels` 透传（KServe v1beta1 把 `PodSpec` 与 `ComponentExtensionSpec` 内联到 `PredictorSpec`），依赖最新 KServe，不引入兜底 webhook 以保持系统简单 |
| 监控 | kube-prometheus-stack | Kubernetes 生态事实标准；ServiceMonitor 自动发现免维护；与 GPU Operator 的 DCGM Exporter 开箱即用 |
| 部署策略 | 拆成 `axisml-infra` / `axisml-system` 两个 chart | 基础设施和控制平面发版节奏、回滚粒度不同；拆分后 infra 可共享给多套 axisml-system 实例。两者通过命名空间 + Service DNS 解耦，仍保持 `condition` 字段支持按需关闭并对接外部实例 |

## 12. 未来规划

本次 Infra 设计范围聚焦核心能力，以下组件暂不引入，留作后续扩展：

- **共享文件存储**（如 JuiceFS）：训练大数据集的 POSIX 挂载，Phase 1 可先通过 PVC + RustFS 的 S3 协议访问解决
- **OCI Registry 双层后端**：把 zot 的 storage backend 配置成 S3 协议指向 RustFS（"zot metadata + RustFS blobs"），把所有制品 bytes 物理上汇聚到对象存储层
- **日志采集**（如 Fluent Bit + ClickHouse）：集中式日志查询，短期内通过 `kubectl logs` + Prometheus 事件满足
- **链路追踪**：基于 OpenTelemetry 的分布式调用链，与业务组件改造同步推进

上述组件引入时机由后续 roadmap 决定，届时以增量方式更新本文档。
