# AxisML 监控设计

本文档汇总 AxisML 控制平面所有组件的 Prometheus 监控指标。监控基础设施由 [kube-prometheus-stack](infra.md#47-监控kube-prometheus-stack) 提供（Prometheus + Grafana + AlertManager），各组件通过 `/metrics` 端点 + `ServiceMonitor` 接入。

---

## 1. 接入模型

各组件接入 Prometheus 的目标约定：

1. 在容器内暴露 `/metrics` 端点（Prometheus 格式）；
2. Helm chart 按组件提供 `ServiceMonitor` CRD，声明待采集的 Service 与端口；
3. Prometheus Operator 自动发现并配置采集目标，无需手动维护 `prometheus.yml`。

当前 Helm chart 已为 compute-service / artifact-hub 提供 `servicemonitor.yaml`，默认 `*.serviceMonitor.enabled=false`，部署 kube-prometheus-stack 后可按需打开。cluster-manager 暴露 metrics Service 端口但尚未提供 ServiceMonitor；tenant-operator / compute-operator 暴露 Pod metrics 端口但尚未提供 Service / ServiceMonitor；Platform chart 当前仍是 nginx placeholder，暂不暴露 backend metrics。

| 组件 | 端口 | 当前 Helm 接入状态 |
| --- | --- | --- |
| Cluster Manager | `/metrics` `:8081` | Service 暴露 `metrics` 端口；ServiceMonitor 待补 |
| Compute Service | `/metrics` `:8081` | `deploy/helm/axisml-system/templates/compute-service/servicemonitor.yaml` |
| Artifact Hub | `/metrics` `:8081` | `deploy/helm/axisml-system/templates/artifact-hub/servicemonitor.yaml` |
| Platform Backend | 目标 `/metrics` `:8081` | 当前 chart 为 nginx placeholder；ServiceMonitor 待真实 backend 落地 |
| tenant-operator | `/metrics` `:8081` | Pod 暴露 metrics；Service / ServiceMonitor 待补 |
| compute-operator | `/metrics` `:8081` | Pod 暴露 metrics；Service / ServiceMonitor 待补 |

`/metrics` 端口与 `--metrics-bind-address` 启动参数对应，可通过 Helm values 调整。

---

## 2. 指标体系层级

| 层级 | 来源 | 典型指标 |
| --- | --- | --- |
| 集群层 | kube-state-metrics、node-exporter（来自 kube-prometheus-stack） | 节点 CPU / 内存 / 磁盘、Pod 状态 |
| GPU 层 | DCGM Exporter（来自 GPU Operator） | GPU 利用率、显存占用、温度、功耗 |
| 网关层 | Envoy Gateway | 请求量、延迟分位、错误率 |
| 调度层 | koord-scheduler / koord-manager | ElasticQuota 用量与借用、PodGroup 调度状态、调度延迟 |
| 控制面 | AxisML 自研组件 | 见下文 §3–§5 |

集群 / GPU / 网关 / 调度层指标都由开源组件原生暴露，本文不重复列出；本文重点是控制面服务自定义指标。

---

## 3. 命名约定

控制面服务 metric 名遵循 Prometheus 社区约定：

| 段 | 含义 |
| --- | --- |
| `axisml_<component>_*` | core 层组件（compute / artifacts） |
| `platform_<module>_*` | Platform 后端各业务模块 |
| `_total` | counter |
| `_seconds` / `_duration_seconds` | histogram，单位秒 |
| `_count` / 无后缀 gauge 名 | gauge |

label 取值规则：

- `status` 一般使用 `{success, failure}`；
- `action` / `predicate` / `result` / `reason` 各模块自行定义枚举（详见各模块表格）；
- 不允许 high-cardinality label（如 `user_id`、`tenant_name` 仅在明确聚合用途下使用）。

---

## 4. Core 层指标

### 4.1 Compute Service
| 指标 | 类型 | 用途 |
| --- | --- | --- |
| `axisml_compute_is_leader` | gauge | 当前副本是否为 leader（0/1） |
| `axisml_compute_reconciler_oldest_pending_seconds{resource,predicate}` | gauge | 工作集最老未处理行的 age |
| `axisml_compute_reconciler_actions_total{resource,predicate,result}` | counter | reconciler 动作计数（含 tenant CR sync） |
| `axisml_compute_informer_workqueue_depth{resource}` | gauge | 各模块 Informer work queue 深度 |
| `axisml_compute_informer_cache_size{resource}` | gauge | Informer in-memory cache 条目数（`resource ∈ {tenant, job, service}`；Tenant cache 直接承担 `quotas[].used` 实时查询） |
| `axisml_compute_informer_cache_synced{resource}` | gauge | Informer cache 是否完成首次 list（0/1）；`tenant=0` 时 GET tenant 在 `quotas[].used` 上返 `null` + warning |
| `axisml_compute_informer_last_sync_age_seconds{resource}` | gauge | 距离上次成功 watch 事件的秒数；超过 stale TTL（默认 30s）时 used 字段视为不可信 |
| `axisml_compute_spec_sync_pending_total{resource}` | gauge | 待同步行数（`generation <> observed_generation`） |
| `axisml_compute_external_drift_total{resource,field}` | counter | 检测到非 compute 字段管理者写入 CR 的次数（Tenant / MLRun / MLService） |
| `axisml_compute_api_request_duration_seconds{route,status}` | histogram | API 请求延迟分布 |
| `axisml_compute_metrics_query_duration_seconds{scope,result}` | histogram | 经 compute 代理的 Prometheus 指标查询耗时（`scope ∈ {service, workload}`） |

label 取值：

- `resource ∈ {tenant, job, service}`；
- `predicate ∈ {creating, canceling, deleting, spec_sync}`；
- `result ∈ {success, conflict, not_found, error, skipped}`。

### 4.2 Artifact Hub
| 指标 | 类型 | 含义 |
| --- | --- | --- |
| `axisml_artifacts_is_leader` | gauge | 当前副本是否为 leader（0/1） |
| `axisml_artifacts_uploading_count{kind}` | gauge | 当前 `status='Uploading'` 行数 |
| `axisml_artifacts_gc_actions_total{predicate,result}` | counter | GC 动作计数 |
| `axisml_artifacts_resolve_requests_total{kind,result}` | counter | resolve 请求计数 |
| `axisml_artifacts_initiate_duration_seconds{kind}` | histogram | initiate 端到端耗时 |
| `axisml_artifacts_complete_duration_seconds{kind}` | histogram | complete 端到端耗时（含后端 HEAD 校验） |
| `axisml_artifacts_api_request_duration_seconds{route,status}` | histogram | API 请求延迟 |

label 取值：

- `kind ∈ {model, dataset, image}`；
- `predicate ∈ {expire_uploading, orphan_oci, orphan_s3}`；
- `result ∈ {success, conflict, not_found, error, skipped}`（与 §4.1 对齐）。

### 4.3 Cluster Manager

| 指标 | 类型 | 用途 |
| --- | --- | --- |
| `axisml_cluster_manager_api_request_duration_seconds{route,status}` | histogram | API 请求延迟 |
| `axisml_cluster_manager_api_requests_total{route,status}` | counter | API 请求计数 |
| `axisml_cluster_manager_resource_pools_total` | gauge | 当前活跃 ResourcePool CR 数 (从 K8s API 或 Informer cache 聚合) |
| `axisml_cluster_manager_resource_units_total` | gauge | 当前活跃 unit 数 (聚合自 `pool.spec.units[]`) |
| `axisml_cluster_manager_k8s_request_total{verb,resource,result}` | counter | 出站 K8s API 调用计数 |
| `axisml_cluster_manager_cluster_query_total{kind,result}` | counter | 集群事实查询计数（`kind ∈ {capacity, metrics}`；capacity 走 K8s 聚合、metrics 走 Prometheus） |

Cluster Manager 是 K8s admin REST 抽象（ResourcePool CR CRUD 入口）；无 reconciler / 无 leader election。Tenant 漂移检测由 compute 暴露：见 §4.1。

### 4.4 tenant-operator

| 指标 | 类型 | 用途 |
| --- | --- | --- |
| `axisml_tenant_operator_reconcile_total{result}` | counter | Tenant CR reconcile 总次数（`result ∈ {success, requeue, conflict, not_found, error, skipped}`） |
| `axisml_tenant_operator_reconcile_duration_seconds` | histogram | reconcile 耗时分布 |
| `axisml_tenant_operator_workqueue_depth` | gauge | controller-runtime work queue 深度 |
| `controller_runtime_*` | 多种 | controller-runtime 内置指标 |

### 4.5 compute-operator

| 指标 | 类型 | 用途 |
| --- | --- | --- |
| `axisml_compute_operator_reconcile_total{resource,backend,engine,result}` | counter | MLRun / MLService reconcile 总次数 |
| `axisml_compute_operator_reconcile_duration_seconds{resource,backend,engine}` | histogram | reconcile 耗时分布 |
| `axisml_compute_operator_handler_dispatch_total{backend,engine,result}` | counter | backend handler 派遣计数 |
| `controller_runtime_*` | 多种 | controller-runtime 内置指标 |

label 取值：

- `resource ∈ {mlrun, mlservice}`；
- `backend ∈ {native}`（其它 backend 元组未交付，label 预留）；
- `engine ∈ {job, deployment, statefulset}`；
- `result ∈ {success, requeue, conflict, not_found, error, skipped}`（与 §4.1 / §4.2 对齐）。

---

## 5. Platform 层指标

### 5.1 跨模块通用

由 Platform 后端的 typed client 中间件层统一打点（[platform.md §5.1](components/platform.md#51-跨服务调用模型)）：

| 指标 | 类型 | 用途 |
| --- | --- | --- |
| `platform_upstream_request_total{service,method,status}` | counter | 每次下游调用计数；`service ∈ {cluster-manager, compute, artifacts}` |
| `platform_upstream_request_duration_seconds{service,method,status}` | histogram | 下游调用延迟分布 |
| `platform_api_request_duration_seconds{route,status}` | histogram | Platform 自身 API 请求延迟 |
| `platform_auth_jwt_issued_total{kind,result}` | counter | JWT 颁发量（`kind ∈ {login, workspace}`） |

### 5.2 Tenant 模块

| 指标 | 类型 | 用途 |
| --- | --- | --- |
| `platform_tenant_action_total{action, status}` | counter | `action ∈ {create, update_meta, delete, restore, quota_create, quota_update, quota_delete, member_add, member_update, member_remove}` |
| `platform_tenant_orphan_role_cleanup_total{reason}` | counter | 孤儿 `user_tenant_roles` 行的级联清理次数；`reason ∈ {delete_cascade, list_reconcile}` |

业务编排见 [platform.md §4.1 租户编排](components/platform.md#41-租户编排)。

### 5.3 Workspace 模块

| 指标 | 类型 | 用途 |
| --- | --- | --- |
| `platform_workspace_action_total{action, status}` | counter | `action ∈ {create, update_meta, start, stop, delete, access_token_issue}` |
| `platform_workspace_state{tenant_name, state}` | gauge | 按租户聚合各派生 `status` 的 workspace 数；定期采样 |
| `platform_workspace_access_jwt_issued_total{result}` | counter | access JWT 颁发量 + 失败原因 |

> Workspace PVC 生命周期由 compute-service 同事务派生与回收（详见 [compute-service.md §4.4](components/compute-service.md#44-service)）；孤儿 PVC / 回滚相关指标归 compute 侧暴露，不在 Platform 模块。

业务编排见 [platform.md §4.4 工作区编排](components/platform.md#44-工作区编排)。

### 5.4 Job 模块

| 指标 | 类型 | 用途 |
| --- | --- | --- |
| `platform_job_action_total{action, status}` | counter | `action ∈ {create, get, list, cancel, delete, logs_stream}` |
| `platform_job_list_tenant_fanout` | histogram | 单次列表请求的下游扇出 namespace 数 |
| `platform_job_list_partial_total{reason}` | counter | 部分租户失败次数 |
| `platform_job_logs_stream_active` | gauge | 当前活跃 SSE log stream 连接数 |

业务编排见 [platform.md §4.2 计算任务编排](components/platform.md#42-计算任务编排)。

### 5.5 Service 模块

| 指标 | 类型 | 用途 |
| --- | --- | --- |
| `platform_service_action_total{action, status}` | counter | `action ∈ {create, get, list, scale, start, stop, patch, delete, access_token_issue, metrics_query}` |
| `platform_service_list_tenant_fanout` | histogram | 单次列表请求的下游扇出 namespace 数 |
| `platform_service_list_partial_total{reason}` | counter | 部分租户失败次数 |
| `platform_service_access_jwt_issued_total{result}` | counter | access JWT 颁发量 + 失败原因 |
| `platform_service_state{tenant_name, state}` | gauge | 按租户聚合各 `mlservices.status` 的 service 数；定期采样 |
| `platform_service_metrics_query_total{metric, status}` | counter | service 指标查询（经 compute-service）结果分布 |

业务编排见 [platform.md §4.3 在线服务编排](components/platform.md#43-在线服务编排)。

### 5.6 ResourcePool 模块

| 指标 | 类型 | 用途 |
| --- | --- | --- |
| `platform_resource_pool_action_total{action, status}` | counter | `action ∈ {create, update, delete, delete_blocked}` |
| `platform_resource_unit_action_total{action, status}` | counter | 同上 |
| `platform_resource_pool_unit_count_aggregation_failures_total` | counter | 列表页聚合资源单元数量时的失败计数 |

业务编排见 [platform.md §4.6 资源池编排](components/platform.md#46-资源池编排)。

---

## 6. 业务指标查询（Prometheus 代理）

Service 详情页指标 Tab 与 Dashboard 时序图的 Prometheus 查询由拥有该域的 System 服务执行，Platform 仅透传返回的 `MetricSeries`、不直连 Prometheus：

- **在线服务 / 工作负载指标** 由 compute-service 执行（`GetServiceMetrics` / `GetWorkloadMetrics`），按 `spec.backend` 选 PromQL 模板；
- **集群容量与集群时序** 由 cluster-manager 执行（`GetClusterCapacity` / `GetClusterMetrics`）。

下表为在线服务指标 Tab 的查询模板，字段契约见 [apis/platform.yaml](apis/platform.yaml) `Services` tag `metrics` 端点。

| 指标 | 含义 | PromQL 来源 |
| --- | --- | --- |
| `request_rate` | 每秒请求数 | Envoy `envoy_http_downstream_rq_total` / KServe / 自定义 backend metrics |
| `latency` | 请求延迟分位（`p50` / `p95` / `p99`） | `histogram_quantile(<p>, sum(rate(envoy_http_downstream_rq_time_bucket{...}[5m])) by (le))` |
| `error_rate` | 错误率 | `sum(rate(envoy_http_downstream_rq_total{response_code=~"5.."}[5m])) / sum(rate(envoy_http_downstream_rq_total[5m]))` |
| `cpu_util` | 副本 CPU 利用率 | `container_cpu_usage_seconds_total` |
| `mem_util` | 副本内存利用率 | `container_memory_working_set_bytes` |
| `gpu_util` | GPU 利用率 | DCGM `DCGM_FI_DEV_GPU_UTIL` |

compute-service 与 cluster-manager 各自以启动参数 `--prometheus-url`（指向 `axisml-infra` namespace 下的 Prometheus）执行查询；Platform 不持有 Prometheus 连接。

**训练指标 / 评估报告不走 Prometheus**：实验 Run 的训练指标以 TensorBoard event log 形式写入对象存储，由按需拉起的 TensorBoard 实例（compute `MLService(kind=tensorboard)`）读取展示；评估 Run 的结果以 `report.json` 写入对象存储，由 Platform 经 compute（`GetMLRunReport`）取回**临时只读地址**后直读渲染榜单（compute 只签发地址、不代理 bytes）。二者均为**文件态**指标，不被 Prometheus 抓取、不进 `MetricSeries` 代理通道（编排见 [platform.md §4.9–§4.11](components/platform.md#49-实验编排)）。

---

## 7. 日志约定

控制面服务统一使用 `zap` 结构化日志：

- 每条租户操作日志携带 `tenant_name` / `actor_user` / `action` / `status` 字段；
- 工作负载操作日志携带 `namespace` / `resource_id` / `resource_name` / `action`；
- 错误日志携带 `error` + `stack`（仅 fatal）；
- 日志级别由 `--log-level` 启动参数控制（默认 `info`）。

日志聚合当前不在交付范围内——Pod 日志由 K8s 默认 logging 驱动收集，运维通过 `kubectl logs` 或集群级聚合方案（如 Loki，未默认部署）查询。

**Pod 日志保留 SLA**：训练任务 / 在线服务 / 工作区的 Pod 日志保留期 **= Pod 自身 TTL**（受 `runPolicy.ttlSecondsAfterFinished` 与节点级 logrotate 影响，默认数小时到一天量级）。Pod 被 GC 后日志即丢失；长周期 retro debug 必须依赖集群级日志聚合方案（未默认部署）。

---

## 8. 告警

当前**不预置** AlertManager 告警规则——AlertManager 随 kube-prometheus-stack 部署但无业务告警规则，调用方按需自定义。

调用方自定义告警时可参考以下方向：

- 节点 NotReady；
- GPU 异常（DCGM 上报错误率高）；
- PVC 容量；
- 配额耗尽（ElasticQuota `min` 持续不可满足）；
- 调度滞后（PodGroup gang 调度长时间 Pending）；
- API 错误率（5xx 比例超阈值）。

告警规则当前不在产品菜单内；后续若纳入，将作为系统管理下的独立入口维护，与 service 详情页指标 Tab 解耦。

---

## 9. 关联文档

- [overview.md](overview.md)：系统级导航；
- [infra.md §4.7](infra.md#47-监控kube-prometheus-stack)：监控基础设施部署细节；
- [apis/platform.yaml](apis/platform.yaml) `Services` tag `metrics` 端点：Service 详情页指标 Tab 字段契约；
- [platform.md §4 核心功能](components/platform.md#4-核心功能)：各业务模块编排逻辑（度量 label 中的 `action` 与编排步骤一一对应）。
