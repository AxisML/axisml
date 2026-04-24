# MLService Operator 详细设计

## 1. 与 Compute 的写路径契约

Compute 采用 Operation Outbox + reconciler 异步下发 CR（详见 [compute.md §5.2](../compute.md)）。Operator 必须满足以下契约：

- **`Create` 幂等**：相同 `metadata.name` 的二次 `Create()` 返回 409 `AlreadyExists`，且不引发副作用。Compute 收到 409 后会 `Get()` 现有 CR 并校验 label `axisml.io/service-id=<uuid>`；只有 label 一致才视为成功
- **不主动反向写 PG**：operator 不感知 Compute 的 `services` 表；状态推进由 Compute 侧 Informer 按 CR `status` 回流
- **status 单向权威**：operator 只写 MLService `status`；Compute 只写 MLService `metadata` / `spec`
- **扩缩容幂等**：重复 patch 相同 `spec.replicas` 不得重建 Deployment；只调整副本数

## 2. CRD 契约

MLService 为 namespaced CR，创建在租户 namespace 下。Compute 负责设置：

- `metadata.name`：来自 `services.name`
- `metadata.namespace`：来自 `tenants.namespace`
- `metadata.labels["axisml.io/service-id"]`：`services.id`
- `metadata.labels["axisml.io/tenant"]`：租户名
- `metadata.labels["axisml.io/queue"]`：Compute Queue 名

最小 `spec`：

```yaml
apiVersion: compute.axisml.io/v1alpha1
kind: MLService
spec:
  image: string
  modelRef:
    name: string
    version: string
  replicas: int      # >= 0
  queueName: string  # Volcano Queue CR name
  resources:
    requests: {}     # Kubernetes ResourceList，单副本资源
    limits: {}       # Kubernetes ResourceList，单副本资源
  placement:
    nodeSelector: {}
    tolerations: []  # Kubernetes Toleration 数组
  ports:
    - name: http
      containerPort: 8080
```

Operator 负责把 `queueName` 写入服务 Pod 对应的 `PodGroup.spec.queue`，并给所有 Pod 设置 `schedulerName: volcano`。

## 3. Status 契约

最小 `status`：

```yaml
status:
  observedGeneration: int64
  phase: Pending | Ready | Degraded | Failed
  message: string
  readyReplicas: int
  endpoint: string
```

Compute 映射规则：

| MLService status.phase | services.status | 终态 |
| --- | --- | --- |
| `Pending` | `Pending` | 否 |
| `Ready` | `Ready` | 否 |
| `Degraded` | `Degraded` | 否 |
| `Failed` | `Failed` | 否，可恢复 |

CR 删除事件由 Compute Informer 映射为 `Cancelled`（若 PG 中 Service 尚未进入 `Cancelled` 或软删）。

## 4. 底层资源

- 每个 MLService 创建 Deployment / Service，并按实现需要创建 Volcano `PodGroup`
- Deployment / Service / PodGroup 必须设置 ownerReference 指向 MLService，保证 MLService 删除后底层资源级联清理
- operator 不读写 Volcano Queue CR；Queue CR 由 Compute 独占维护
