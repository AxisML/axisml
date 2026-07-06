# Platform API 缺口与 System 层接口需求

本文列出 Platform API 中待实现 / 降级的接口,及补齐它们所需的 System 层新增接口,作为实施依据。

口径:`axisml-platform/docs/apis/platform.yaml` 契约与 backend 实际注册路由(`internal/*/handler.go` 的 `Register`)比对;未注册的 `/api/v1` 路由由 `internal/server/server.go` 的 `NoRoute` 返回 `501`。

---

## 一、Platform API 待实现接口

### A. 未实现 —— 路由未注册,返回 `501`

| 方法 | 路径 | 返回 schema | 数据来源 |
|---|---|---|---|
| GET | `/api/v1/dashboard/cluster-usage` | `ClusterUsage` | N2(逐池) |
| GET | `/api/v1/dashboard/cluster-metrics` | `MetricSeries` | N3(逐池) |
| GET | `/api/v1/dashboard/activity` | `ActivityList` | Platform 审计 |
| GET | `/api/v1/workspace-images` | `WorkspaceImageList` | Platform 策展目录 |

### B. 指标端点 —— 已注册但返回 `502`

均走 `server.MetricsUnavailable()`(`internal/server/respond.go:88`),等待 N1。

| 方法 | 路径 | handler |
|---|---|---|
| GET | `/api/v1/jobs/{name}/runs/{run}/metrics` | `internal/job/handler.go:183` |
| GET | `/api/v1/experiments/{name}/runs/{run}/metrics` | `internal/experiment/handler.go:186` |
| GET | `/api/v1/mlservices/{name}/metrics` | `internal/mlservice/handler.go:145` |
| GET | `/api/v1/trafficpolicies/{name}/metrics` | `internal/traffic/handler.go:129` |

### C. 占位实现 —— System 已支持,待 Platform 接线

| 方法 | 路径 | 待办 |
|---|---|---|
| PATCH | `/api/v1/trafficpolicies/{name}` | `internal/traffic/handler.go:83` 当前回显 `get`,不持久化 `displayName/description/labels/annotations`。新增 `computeservice.PatchTrafficPolicy`(已有 `TrafficPatch` 类型别名)+ `traffic.Service.Update`,由 handler 调用 compute-service PATCH。 |

### D. 缺前置校验 —— 行为待补

| 位置 | 待补校验 | 依赖 |
|---|---|---|
| `internal/tenant/service.go:250` | 租户删除前阻断存在活跃 Run/Service/Workspace | compute-service 按 namespace `list` |
| `internal/mlservice/service.go:35` | MLService 创建前对 `(modelName, modelVersion)` 做 artifact-hub Ready 预检并注入解析后 URI/digest | artifact-hub `GET`/`resolve` |
| `internal/resourcepool/service.go:70` | 资源池删除前按 label `resource.axisml.io/pool=<name>` 枚举工作负载做 in-use 校验 | N4(可选)或遍历租户 namespace |

---

## 二、System 层新增接口

| # | 组件 | 接口 | 响应 | 数据来源 | 解锁 |
|---|---|---|---|---|---|
| **N1** | compute-service | `GET /namespaces/{ns}/mlruns/{mlrun}/metrics`<br>`GET /namespaces/{ns}/mlservices/{mlservice}/metrics`<br>`GET /namespaces/{ns}/traffic-policies/{policy}/metrics`<br>query:`range`,`step` | `MetricSeries` | Prometheus / PromQL | 一·B |
| **N2** | cluster-manager | `GET /resourcepools/{pool}/usage?tenant=<tenant_identifier>` | 该租户在该池的 `used/total`(resource 维度) | ElasticQuota `ServerQuotaStatus.Used` + 该池 quota `unit×quantity` 按 ResourceUnit 展开 | `dashboard/cluster-usage` |
| **N3** | cluster-manager | `GET /resourcepools/{pool}/metrics?tenant=<tenant_identifier>`<br>query:`range`,`step` | `MetricSeries` | Prometheus / PromQL | `dashboard/cluster-metrics` |

约定:

- N2/N3 粒度固定为 **(租户, 资源池)**:资源池为路径主体,`tenant` 为查询过滤;不做跨资源池聚合。
- Dashboard 中仅 `cluster-usage` / `cluster-metrics` 与 pool 相关,走 N2/N3;其余端点(`activity`)与 pool 无关。
- Platform 从租户配额列表取得其涉及的池,逐池调用 N2/N3,渲染为 per-pool 列表(一池一条)。
- N1/N3 各自内置 PromQL 查询(分别在 compute-service、cluster-manager),不抽独立 metrics 网关;两者均需为所在组件增加 Prometheus 查询客户端。

可选:

| # | 组件 | 接口 | 用途 |
|---|---|---|---|
| **N4** | compute-service | 跨命名空间工作负载 `list`(按 `labelSelector`) | 支撑一·D 资源池 in-use 校验;可先用"遍历租户 namespace + 现有 per-ns `list`"替代。 |

修改:无。一·C 的缺口在 Platform 侧,compute-service 契约已就绪。

---

## 三、Platform 侧独立项(无 System 依赖)

| 项 | 实施 |
|---|---|
| `dashboard/activity` | Platform 审计域落审计存储(可聚合 compute events),按租户返回 `ActivityList`。 |
| `workspace-images` | Platform 策展目录 / 配置提供 `WorkspaceImage` 列表(`displayName/kind/defaultPort/public/ref`)。 |
| `ClusterUsage` schema | 为 per-pool 列表(无跨池聚合字段);改 Go DTO 后 `make -C axisml-platform doc-gen`。 |

---

## 四、缺口 → 依赖 映射

| Platform 接口 | 状态 | 依赖 |
|---|---|---|
| `dashboard/cluster-usage` | 501 | N2 |
| `dashboard/cluster-metrics` | 501 | N3 |
| `dashboard/activity` | 501 | 无(Platform 审计) |
| `workspace-images` | 501 | 无(Platform 目录) |
| `jobs/*/runs/*/metrics` | 502 | N1(mlrun) |
| `experiments/*/runs/*/metrics` | 502 | N1(mlrun) |
| `mlservices/*/metrics` | 502 | N1(mlservice) |
| `trafficpolicies/*/metrics` | 502 | N1(traffic) |
| `PATCH trafficpolicies/*` | 占位 | 无(Platform 接线) |
| 租户删除 in-use 校验 | 待补 | 无(compute list) |
| MLService 创建预检 | 待补 | 无(artifact-hub resolve) |
| 资源池删除 in-use 校验 | 待补 | N4(可选) |

---

## 五、实施顺序

**第一批 — 无 System 依赖**

- 一·C traffic PATCH 接线。
- 一·D 租户删除、MLService 创建两处前置校验。
- 三 `dashboard/activity`、`workspace-images`、`ClusterUsage` 改 per-pool。

**第二批 — 依赖 System 新接口**

- N1 → 一·B 4 个端点。
- N2 → `dashboard/cluster-usage`。
- N3 → `dashboard/cluster-metrics`。

**可选**

- N4 或遍历方案 → 资源池 in-use 校验。
