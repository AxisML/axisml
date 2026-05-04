# axisml-operator 详细设计

axisml-operator 是 AxisML 控制平面里**唯一**的 Kubernetes operator 二进制，由一个 Manager 同时承载三个 controller：

| Controller | CRD（`axisml.io/v1alpha1`） | Scope | 子设计文档 |
| --- | --- | --- | --- |
| Tenant | `Tenant` | Cluster-scoped | [tenant.md](./operator/tenant.md) |
| MLJob | `MLJob` | Namespaced | [mljob.md](./operator/mljob.md) |
| MLService | `MLService` | Namespaced | [mlservice.md](./operator/mlservice.md) |

每个 controller 的 CRD 契约、字段不可变性、状态机、子资源管理等细节见各子文档；本文档只覆盖**跨 controller 的合并设计**与**作为单一 Deployment 的运维契约**。

## 1. 合并动机

历史上 tenant / mljob / mlservice 是三个独立的 Go 模块、独立的 Deployment、独立的 ServiceAccount/ClusterRole。三者长期 lock-step：共用 image tag（`components/operators/Dockerfile` + `entrypoint.sh` 已经把它们打成一个镜像，靠 argv[0] 分派）、共用 controller-runtime / k8s.io 版本、共用 RBAC 习惯。把它们折叠进同一个 Manager 后：

- 三个 Deployment / SA / ClusterRole / Lease 缩减为一份；
- 三个 Go module（`go.mod`）+ 三个 envtest module 合并成一个 production module + 一个 envtest module；
- CI lint matrix 6 → 2、envtest matrix 3 → 1；
- 升级路径与运维诊断只看一个 Pod 的日志即可。

代价：**单个 Deployment 失去了按 CRD 独立 rollout 的能力**。这在过去三个 operator 始终同 image tag 同时升级的现实下是可接受的；如果未来某个 controller 需要独立 rollout，可以利用下文 §3.3 的 `--enable-*` 开关，把同一镜像以不同 flag 启动多个 Deployment。

## 2. 架构总览

```
┌──────────────────── axisml-operator (one Pod, leader-elected) ────────────────────┐
│                                                                                    │
│  ctrl.Manager (scheme: clientgoscheme + axisml + scheduling.sigs.k8s.io +          │
│                gateway.networking.k8s.io)                                          │
│  Lease: axisml-operator.axisml.io                                                  │
│                                                                                    │
│  ┌──────────────────┐  ┌────────────────────────┐  ┌───────────────────────────┐   │
│  │ TenantReconciler │  │ MLJob: dispatcher +    │  │ MLService: dispatcher +   │   │
│  │ (single, no      │  │ Registry → handlers/   │  │ handler.Build() →         │   │
│  │  dispatcher)     │  │ {nativejob,podgroup}   │  │ handler/{nativedeploy.}   │   │
│  └──────────────────┘  └────────────────────────┘  └───────────────────────────┘   │
│        │                       │                            │                      │
│        ▼                       ▼                            ▼                      │
│   Namespace,                Job, Pod, PodGroup        Deployment, Service,         │
│   ElasticQuota,             (koord-scheduler          HTTPRoute (Gateway API)      │
│   Secret/CM/SA/             gang scheduling)                                       │
│   Role/RoleBinding                                                                 │
└────────────────────────────────────────────────────────────────────────────────────┘
```

Tenant 走**单 reconciler**直接调度（无 dispatcher）；MLJob 与 MLService 共用 **dispatcher + handler** 模式：CR 的 `spec.backend.{name, engine}` 元组路由到注册过的 Handler，handler 渲染目标 GVK 并把状态回流到 CR.status。这两个 controller 的具体 dispatch 表与默认后端见各自的子文档（[mljob §7](./operator/mljob.md), [mlservice §11](./operator/mlservice.md)）。

## 3. 合并后的运行时契约

### 3.1 Scheme 注册

```go
clientgoscheme.AddToScheme(scheme)           // core, apps, rbac, batch, coordination
schedulingv1alpha1.AddToScheme(scheme)       // ElasticQuota + PodGroup（Koordinator vendored）
gwapiv1.Install(scheme)                      // HTTPRoute
tenant_v1alpha1.AddToScheme(scheme)          // Tenant
mljob_v1alpha1.AddToScheme(scheme)           // MLJob
mlservice_v1alpha1.AddToScheme(scheme)       // MLService
mlservicehandler.RegisterStubs()             // MLService 占位 handler 注册
```

三个 CRD 共享 group `axisml.io/v1alpha1`，但 Go 类型分别定义在 `components/operator/api/{tenant,mljob,mlservice}/v1alpha1/` 三个子包里——避免 `Phase`、`RoleSpec`、`LabelQuota` 等同名常量在同一包内冲突，同时仍然让一个 Manager 通过分别 `AddToScheme` 把三种 Kind 全注册进去。

### 3.2 Cache 选择性过滤

Tenant 的子资源（Secret / ConfigMap / ServiceAccount / Role / RoleBinding / ElasticQuota）在生产中受 `managed-by=tenant-operator` label 过滤，避免缓存全集群 Secret。**关键约束：这条过滤必须按对象类型挂在 `cache.Options.ByObject` 上**，不能升格成 `cache.Options.DefaultLabelSelector`——否则 MLJob 的 `Job/Pod/PodGroup` informer 与 MLService 的 `Deployment/HTTPRoute` informer 会被同样的 label 过滤掉，导致丢事件。

```go
cache.Options{
    SyncPeriod: &resync,
    ByObject: map[client.Object]cache.ByObject{
        &corev1.Secret{}:                   {Label: managedByOnly},
        &corev1.ConfigMap{}:                {Label: managedByOnly},
        &corev1.ServiceAccount{}:           {Label: managedByOnly},
        &rbacv1.Role{}:                     {Label: managedByOnly},
        &rbacv1.RoleBinding{}:              {Label: managedByOnly},
        &schedulingv1alpha1.ElasticQuota{}: {Label: managedByOnly},
    },
}
```

注意：`PodGroup` 与 `ElasticQuota` 都属于 `scheduling.sigs.k8s.io/v1alpha1`，但只有 `ElasticQuota` 由 Tenant 写入并打 `managed-by` label。`PodGroup` 由 MLJob handler 写入、不带这条 label，因此在表里**只列 `ElasticQuota`**。

### 3.3 Flag 集合

| Flag | 默认 | 说明 |
| --- | --- | --- |
| `--metrics-bind-address` | `:8080` | Prometheus metrics |
| `--health-probe-bind-address` | `:8081` | `/healthz`, `/readyz` |
| `--leader-elect` | `true` | leader election |
| `--leader-election-id` | `axisml-operator.axisml.io` | Lease 名 |
| `--enable-tenant` | `true` | 启用 Tenant controller |
| `--enable-mljob` | `true` | 启用 MLJob controller |
| `--enable-mlservice` | `true` | 启用 MLService controller |
| `--enable-native-job` | `true` | MLJob: 注册 (native, job) handler |
| `--enable-native-podgroup` | `true` | MLJob: 注册 (native, podgroup) handler |

Pod 上还会注入两个环境变量供 Tenant 子模块消费：`RESYNC_PERIOD`（默认 `10m`）、`NAMESPACE_DENYLIST`（逗号分隔列表，默认值见 Helm `values.yaml`）。

### 3.4 RBAC

合并后只保留**一个** ClusterRole（`<release>-operator`），rules 是三个 controller 所需权限的并集，按 controller 分段；段头按 `--enable-*` Helm value 条件渲染（见 `deploy/helm/axisml-system/templates/operator/clusterrole.yaml`）。leader election Lease 在部署 namespace 通过 Role + RoleBinding 授权（不放进 cluster-scoped 角色）。

### 3.5 Helm values 接口

```yaml
operators:
  enabled: true
  image: { registry, repository, tag, pullPolicy }
  leaderElection: { enabled, id }
  resources: { requests, limits }
  controllers:
    tenant:    { enabled, resyncPeriod, namespaceDenylist }
    mljob:     { enabled, handlers: { nativeJob, nativePodGroup } }
    mlservice: { enabled }
```

旧的 `operators.{tenantOperator, mljobOperator, mlserviceOperator}` 三段配置在升级到合并版本时**必须**重写为 `operators.controllers.*`——这是一次破坏性 values 变更，Deployment / SA / ClusterRole 也都重命名了，因此 `helm upgrade` 时旧资源会被删除、新资源会被创建。`helm upgrade` 不会清理三个旧 Lease（`tenant-operator.axisml.io` 等），需手工 `kubectl delete lease`。

## 4. 三个 controller 的差异要点

| 维度 | Tenant | MLJob | MLService |
| --- | --- | --- | --- |
| 架构 | 单 reconciler | dispatcher + Registry | dispatcher + Registry（factory + stubs） |
| Scope | Cluster | Namespaced | Namespaced |
| 主要外部依赖 | Koordinator ElasticQuota | scheduler-plugins PodGroup（vendored 在 koordinator）| Gateway API HTTPRoute |
| 默认后端 | n/a | `(native, job)` | `(native, deployment)` |
| Pod 注入约束 | n/a | `schedulerName=koord-scheduler` + `quota.scheduling.koordinator.sh/name` label | 同左 |
| 状态机 | `Pending\|Ready\|Failed` | `Pending\|Running\|Succeeded\|Failed` | `Pending\|Ready\|Degraded\|Failed` |

详细 reconcile 时序、字段不可变性、子资源管理见各子文档：

- [tenant.md](./operator/tenant.md)
- [mljob.md](./operator/mljob.md)
- [mlservice.md](./operator/mlservice.md)

## 5. 测试

合并后的 L1 envtest 在 `components/operator/test/envtest/` 单一 Go module 中，单一 `TestMain` 注册三个 reconciler 到同一个 envtest manager，跑七个 test 文件（tenant 2 + mljob 3 + mlservice 2）。CRDPaths 是 `deploy/helm/axisml-system/crds` 与 `test/crds/external/`（vendored ElasticQuota / PodGroup / HTTPRoute）的并集。

L2 e2e 仍在 `test/e2e/`，通过部署后的 axisml-operator 与 MLPlatform/Compute API 一起跑端到端；e2e 不直接关心 operator 二进制名。

## 6. 升级路径

旧布局：

```
components/operators/{tenant,mljob,mlservice}-operator/
deploy/helm/axisml-system/templates/operators/{tenant,mljob,mlservice}-operator/
docs/system_design/operators/{tenant,mljob,mlservice}-operator.md
```

新布局：

```
components/operator/                                       # 单 module
deploy/helm/axisml-system/templates/operator/              # 单组模板
docs/system_design/operator.md                             # 总览
docs/system_design/operator/{tenant,mljob,mlservice}.md    # 各 controller 子文档
```

`helm upgrade`：旧 Deployment / SA / ClusterRole / ClusterRoleBinding（三份）由 Helm release 记录释放后会被删除并替换为新的合并版本，预期短暂 downtime（秒级）。镜像名 `ghcr.io/axisml/axisml-operator` 不变。

## 7. 相关引用

- 本目录下的子文档（tenant.md / mljob.md / mlservice.md）保留原样，是各 controller 的权威设计契约。
- [docs/system_design/overview.md §5.3](../overview.md) 概述了 axisml-operator 在控制平面里的位置。
- [docs/system_design/compute.md](../compute.md) 描述 ml-compute 与 operator 之间的 CR 写路径与状态回流。
