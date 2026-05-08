# AxisML Cluster Manager 详细设计

AxisML Cluster Manager 是平台的"管理员域"入口服务。它**不持有任何业务元数据**，是一个无状态薄壳：把外部 REST 请求直接翻译为对 `Tenant` CR 的 K8s API 调用，由 [tenant-operator](tenant-operator.md) 真正落地 Namespace / ElasticQuota / 初始化资源。

cluster-manager 与 [compute](compute.md) / [artifacts](artifacts.md) 的根本差异：

- **后者**是业务服务，持有 PG 元数据，向 compute-operator 异步下发 CR；
- **本服务**是 K8s API 的轻量代理 + 业务字段校验层，权威完全在 etcd（Tenant CR）。

| 模块 | 职责 | 边界外 |
| --- | --- | --- |
| Tenant ([§3](#3-tenant-api)) | Tenant CRUD / suspend / unsuspend：直接 CRUD `Tenant` CR | 业务元数据持久化、用量缓存 |
| Quota ([§4](#4-quota-api)) | 通过 JSON Patch 修改 `Tenant.spec.quotas[]` | Quota CR / ElasticQuota 细节（由 tenant-operator 处理）|

**关键不变式**：

> **cluster-manager 不持有 PG；权威完全是 K8s etcd 中的 Tenant CR。**
> 任何 GET 都走 K8s API（API Server cache / 本服务可选的 informer cache）；任何 mutation 都落到 Tenant CR。

**文档组织**：

- **Part I — 服务运行时**（§1 架构总览 + §2 运行时契约）：进程形态、端口、K8s 客户端、副本、RBAC、Helm values。
- **Part II — API**（§3 Tenant、§4 Quota）：REST 契约、与 Tenant CR 的字段映射。
- **Part III — 实施与验证**（§5 实现路径、§6 测试、§7 相关引用）。

---

## Part I — 服务运行时

## 1. 架构总览

```
                          ┌──────────────────────┐
                          │  AxisML Platform      │
                          │  （唯一调用方，集群内） │
                          └──────────┬───────────┘
                                     │ REST / JSON
                                     │ X-Axisml-User
                                     ▼
                          ┌──────────────────────┐
                          │  AxisML Cluster       │
                          │  Manager (Go binary)  │
                          │  无 PG / 无后台协程    │
                          └──────────┬───────────┘
                                     │ controller-runtime client
                                     ▼
                          ┌──────────────────────┐
                          │   Kubernetes API      │
                          │   Tenant CR           │
                          └──────────┬───────────┘
                                     │ watch
                                     ▼
                          ┌──────────────────────┐
                          │   tenant-operator     │
                          └──────────────────────┘
```

cluster-manager 不消费 Tenant CR 的事件流——所有读路径都直接 GET / LIST K8s API（可借助 controller-runtime 的本地 cache 加速）；写路径直接 Create / Patch / Delete CR。

```
components/cluster-manager/
├── cmd/cluster-manager/      # main.go
├── api/
│   ├── openapi.yaml          # OpenAPI 3.0 契约源
│   └── types/                # oapi-codegen 生成
├── internal/
│   ├── server/               # HTTP router、middleware
│   ├── tenant/               # Tenant API handler
│   ├── quota/                # Quota API handler（patch tenant.spec.quotas[]）
│   ├── k8sclient/            # controller-runtime client
│   ├── auth/                 # 从 X-Axisml-User header 解析调用方身份
│   ├── metrics/              # Prometheus metrics
│   └── app/                  # 启动装配（serve）
└── pkg/
```

## 2. 运行时契约

### 2.1 进程与端口

- 镜像：`ghcr.io/axisml/axisml-cluster-manager:<appVersion>`
- 启动命令：`/cluster-manager serve`
- 端口：`8082/tcp`（REST，仅集群内 ClusterIP，不配置 `HTTPRoute` 对外）
- 探针：`GET /healthz`（进程存活）、`GET /readyz`（K8s API 连通性）
- Metrics：`GET /metrics`（Prometheus 格式）

### 2.2 K8s 客户端

- controller-runtime `client.Client`；可选 `cache` 加速 List 路径，但写路径一律 bypass cache 直读 API Server
- 监听对象：`Tenant`（cluster-scoped）；可选派生 `ElasticQuota` 用于 list 时聚合 `status.used`
- 不引入 `Manager` / `Reconciler` 抽象——cluster-manager 不是 K8s controller

### 2.3 副本与 Leader Election

- **默认 `replicas=2`**（无后台协程，多副本天然安全）
- **API 层无状态**：所有副本都服务 HTTP，水平扩容无需协调
- **不需要 leader election**：服务的所有 mutation 都通过 K8s API 进行，K8s API Server 自带乐观并发控制（`resourceVersion`）；无后台 reconciler / informer 写盘

### 2.4 RBAC（cluster-manager → Kubernetes）

| 资源 | 动作 | 用途 |
| --- | --- | --- |
| `tenants.axisml.io` | `create / get / list / watch / update / patch / delete` | Tenant CR 全权 CRUD |
| `elasticquotas.scheduling.sigs.k8s.io` | `get / list / watch` | GET tenant 时聚合 quota 用量（非必需，提供更友好的响应） |

**不含**：`namespaces`、`secrets`、`configmaps`、`serviceaccounts`、`mljobs.axisml.io`、`mlservices.axisml.io`——这些不在 cluster-manager 的职责范围内。

### 2.5 Helm values 接口

```yaml
clusterManager:
  enabled: true
  image: { registry, repository, tag, pullPolicy }
  replicas: 2
  resources: { requests, limits }
```

**Helm 模板清单**（`deploy/helm/axisml-system/templates/cluster-manager/`）：

| 文件 | 用途 |
| --- | --- |
| `deployment.yaml` | cluster-manager 镜像，加探针 |
| `service.yaml` | ClusterIP 8082 |
| `serviceaccount.yaml` | 服务账号 |
| `clusterrole.yaml` / `clusterrolebinding.yaml` | §2.4 RBAC |
| `servicemonitor.yaml` | `/metrics` 暴露 |

### 2.6 与 Platform 的请求契约

cluster-manager 仅接受 Platform 通过集群内 Service DNS 发起的 REST 调用，不直接对外部用户流量开放。

- **路径前缀**：`/api/v1/...`；不配置外部 `HTTPRoute`
- **身份头**：Platform 注入 `X-Axisml-User`（调用方用户唯一 ID）；cluster-manager 仅做审计（写入 K8s Event 或 Tenant CR annotation `axisml.io/last-modified-by`），不做角色级鉴权（鉴权由 Platform 统一完成）
- **OpenAPI 契约**：`components/cluster-manager/api/openapi.yaml` 是唯一契约源，使用 `oapi-codegen` 生成 cluster-manager 侧 Go types + server stub 与 Platform 侧 Go client SDK
- **错误格式**：HTTP 标准状态码 + RFC 7807 problem+json
- **写后语义**：所有 mutation API 在 K8s API Server 返回 200 后即视为提交成功——但 tenant-operator 的 reconcile 是异步的，调用方需通过后续 GET 观察 `status.phase` / `status.quotas[].ready` 确认落地

---

## Part II — API

## 3. Tenant API

### 3.1 业务字段校验

cluster-manager 在请求层做 **DNS-1123 + 长度** 校验（与 [tenant-operator §4.3.1](tenant-operator.md) 一致），把违反规则的请求直接 4xx 拒绝，避免无效 CR 进入 etcd：

| 字段 | 校验 |
| --- | --- |
| `name` | DNS-1123；长度 3–40；`[a-z0-9-]`；首尾字母数字；无连续 `--` |
| `namespace.name` | DNS-1123；非系统 namespace（denylist 与 tenant-operator Helm values 同源） |
| `quotas[].pool` | DNS-1123 |
| `quotas[].name` | DNS-1123；同一 tenant 内 `(pool, name)` 唯一 |
| `quotas[].max[k]` | 必填；非负 |
| `quotas[].min[k]` | 可选；非负；`min[k] ≤ max[k]` |

更深的语义校验（源 Secret / ConfigMap 是否存在、`(pool, name)` 是否引用有效 ResourcePool）由 tenant-operator 在 reconcile 阶段写 `status.message`，cluster-manager 不前置查询。

### 3.2 端点

#### `POST /api/v1/tenants`

创建 Tenant CR。请求体直接映射到 `Tenant.spec`：

```json
{
  "name": "team-a",
  "displayName": "Team A",
  "annotations": {},
  "namespace": { "name": "team-a", "labels": {}, "annotations": {} },
  "quotas": [
    { "pool": "default", "name": "default", "min": {}, "max": { "cpu": "100", "memory": "200Gi" } }
  ],
  "initResources": {
    "imagePullSecrets": [...],
    "secrets": [...],
    "configMaps": [...],
    "serviceAccounts": [...]
  }
}
```

cluster-manager 在 `metadata.labels["axisml.io/tenant-id"]` 中填入服务侧生成的 UUID（仅作为 platform 使用的稳定锚点；删除并重建同名 Tenant 时 UUID 会变）。

**响应**：201 + 创建的 Tenant CR JSON 表示。
**幂等性**：第二次相同 name 的提交返回 409 `AlreadyExists`，调用方可继续 GET 拿当前 CR。

#### `GET /api/v1/tenants/{name}`

GET Tenant CR。响应包含 `spec`、`status`，以及（可选）按 ownerReference 反查并聚合的 `ElasticQuota.status.used` 用量信息（与 `status.quotas[].used` 等价；本接口提供后者作为权威来源）。

#### `GET /api/v1/tenants`

LIST Tenant CR；支持分页（query 参数 `limit` / `continue`，对应 K8s API 的同名参数）。

#### `PATCH /api/v1/tenants/{name}`

更新 Tenant CR 的可变字段（按 [tenant-operator §4.3.3](tenant-operator.md) 列表；不可变字段如 `spec.namespace.name` / `spec.quotas[].{pool, name}` 在请求层直接 4xx 拒绝）。请求体使用 RFC 7396 JSON Merge Patch。

#### `POST /api/v1/tenants/{name}/suspend` / `unsuspend`

通过 JSON Patch 修改 `Tenant.spec.suspended`：

```json
{ "op": "replace", "path": "/spec/suspended", "value": true }
```

#### `DELETE /api/v1/tenants/{name}`

删除 Tenant CR；K8s GC 通过 ownerReference 级联清理 per-tenant 资源（Namespace 不删除，详见 [tenant-operator §4.6.1](tenant-operator.md)）。

## 4. Quota API

Quota 不是独立 CRD——是 `Tenant.spec.quotas[]` 的一项。所有 Quota mutation 都翻译为对 Tenant CR 的 JSON Patch。

#### `POST /api/v1/tenants/{name}/quotas`

向 `Tenant.spec.quotas[]` 追加一项。等价于：

```json
{ "op": "add", "path": "/spec/quotas/-", "value": { "pool": "...", "name": "...", "min": {}, "max": {} } }
```

cluster-manager 在请求层校验 `(pool, name)` 在当前 spec 中不存在；若已存在 → 409。

#### `PATCH /api/v1/tenants/{name}/quotas/{pool}/{quotaName}`

更新某条 quota 的 `min` / `max`。先 GET 当前 spec，定位 `(pool, name)` 对应的下标 `i`，再 patch：

```json
[
  { "op": "replace", "path": "/spec/quotas/<i>/min", "value": {...} },
  { "op": "replace", "path": "/spec/quotas/<i>/max", "value": {...} }
]
```

并发更新由 K8s API Server 的 `resourceVersion` 乐观锁保护——cluster-manager 在 patch 失败时按 conflict 错误重试 1 次（重新 GET 重算下标）。

#### `DELETE /api/v1/tenants/{name}/quotas/{pool}/{quotaName}`

从 `Tenant.spec.quotas[]` 中删除某条。同上方式 patch `remove`。tenant-operator 监听到 spec 变更后会显式 Delete 对应 ElasticQuota CR（[tenant-operator §4.6.2](tenant-operator.md)）。

#### `GET /api/v1/tenants/{name}/quotas`

返回 Tenant CR 当前的 `spec.quotas[]` + `status.quotas[]`（每条 quota 的 `ready` / `used` / `message`）。

---

## Part III — 实施与验证

## 5. 实现路径

### 5.1 阶段一：MVP

| 模块 | 范围 | 完成信号 |
| --- | --- | --- |
| HTTP server | Gin / chi 任选；OpenAPI 契约源 + oapi-codegen 生成 server stub | `make cluster-manager-build` 通过；`/healthz` / `/readyz` 工作 |
| Tenant API | `POST` / `GET` / `LIST` / `PATCH` / `DELETE` / `suspend` / `unsuspend` | 单元测试覆盖请求层校验；integration（envtest）覆盖端到端 Tenant 创建 → tenant-operator 落地 |
| Quota API | `POST` / `PATCH` / `DELETE` / `GET` | integration 覆盖 Tenant 上多条 quota 的增删改 |
| RBAC | §2.4 ClusterRole + ClusterRoleBinding | helm install 后服务可正常 list / patch tenants |
| Helm | `deploy/helm/axisml-system/templates/cluster-manager/` | helm install 后 `kubectl get deploy/axisml-cluster-manager` Ready |

### 5.2 阶段二：功能完善

1. **审计增强**：把 `X-Axisml-User` 写入 K8s Event 与 Tenant CR annotation `axisml.io/last-modified-by`，便于追溯。
2. **OpenAPI 严格校验**：补全枚举、必填、长度上限的 schema。
3. **批量操作**：`POST /api/v1/tenants:batchCreate` 等，便于平台批量初始化。

### 5.3 阶段三：未来规划

- Admission webhook 模式：把请求层校验前移到 K8s admission，让 `kubectl apply` 直连场景也享受同一套校验。
- 操作审计持久化：把 mutation 流水写入独立审计存储（不进入 cluster-manager 自身的 PG，因为本服务保持无状态）。

## 6. 测试

cluster-manager 没有 reconciler，但因为它把请求落到真实的 K8s API，envtest 仍然适合验证 HTTP → Tenant CR 这条路径。测试层级：

- **单元**：请求层校验、字段映射逻辑、JSON Patch 构造逻辑。
- **integration**（`components/cluster-manager/test/integration/` 独立 Go module）：用 envtest 起一个本地 apiserver，加载 Tenant CRD，把 cluster-manager 的 Gin engine 注册到测试路由后通过 in-process `httptest` 驱动；覆盖 Tenant CRUD、suspend / unsuspend、quota 增改删、列表分页、denylist + DNS-1123 校验。

仓库当前不维护 minikube 驱动的 e2e 层。

## 7. 相关引用

- [docs/system_design/overview.md](../overview.md) 概述了 cluster-manager 在控制平面里的位置。
- [docs/system_design/tenant-operator.md](tenant-operator.md) 描述 cluster-manager 写入的 Tenant CR 的具体落地行为。
- [docs/system_design/platform/overview.md](../platform/overview.md) 描述 platform 如何把"租户视图"映射到 cluster-manager API。
