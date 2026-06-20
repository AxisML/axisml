# 监控

## 需求

平台需要集群与业务的统一可观测性，要求：采集、存储、可视化指标并提供告警通道；覆盖集群 / GPU / 网关 / 调度 / 业务多层指标；采集目标**自动发现**、免手工维护抓取配置；各组件以标准 `/metrics` 接入。

## 技术选型

选用 **[kube-prometheus-stack](https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-prometheus-stack)**（Prometheus + Grafana + AlertManager）。理由：Kubernetes 生态事实标准；Prometheus Operator 经 `ServiceMonitor` / `PodMonitor` 自动发现采集，免维护 `prometheus.yml`；与 GPU Operator 的 DCGM Exporter 开箱即用。

## 接入模型

各组件 (1) 在容器内暴露 `/metrics`（Prometheus 格式，端口 `:8081`）；(2) 随 Helm chart 提供 `ServiceMonitor` / `PodMonitor`，声明待采集对象与端口。Prometheus Operator 自动发现并配置采集目标。当前 compute-service / artifact-hub 提供 `ServiceMonitor`（默认 `*.serviceMonitor.enabled=false`，opt-in），其余组件暴露 metrics 端口、ServiceMonitor 待补。

## 指标体系层级

| 层级 | 来源 | 典型指标 |
| --- | --- | --- |
| 集群层 | kube-state-metrics、node-exporter | 节点 CPU/内存/磁盘、Pod 状态 |
| GPU 层 | DCGM Exporter（来自 GPU Operator） | GPU 利用率、显存、温度、功耗 |
| 网关层 | Envoy Gateway | 请求量、延迟分位、错误率 |
| 调度层 | koord-scheduler / koord-manager | ElasticQuota 用量与借用、PodGroup 调度状态、调度延迟 |
| 业务层 | 接入服务 | 各服务自行暴露 `axisml_*` / `platform_*` 指标 |

## 告警

当前**不预置** AlertManager 告警规则——AlertManager 随栈部署但无业务规则，调用方按需自定义。参考方向：节点 NotReady、GPU 异常（DCGM 上报错误率高）、PVC 容量、配额耗尽（ElasticQuota `min` 持续不可满足）、调度滞后（PodGroup gang 长时间 Pending）、API 5xx 比例超阈值。

> 控制面业务指标的 Prometheus 查询（在线服务指标、Dashboard 时序）由拥有该域的 System 服务执行并以 `MetricSeries` 回传，PromQL 模板在各服务内部维护；训练指标走对象存储 + TensorBoard，不进 Prometheus。
