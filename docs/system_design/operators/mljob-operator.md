# MLJob Operator 详细设计

## 1. 与 Compute 的写路径契约

Compute 采用 Operation Outbox + reconciler 异步下发 CR（详见 [compute.md §5.2](../compute.md)）。Operator 必须满足以下契约：

- **`Create` 幂等**：相同 `metadata.name` 的二次 `Create()` 返回 409 `AlreadyExists`，且不引发副作用（不重启 Pod、不重置 status）。Compute 收到 409 后会 `Get()` 现有 CR 并校验 label `axisml.io/job-id=<uuid>`；只有 label 一致才视为成功
- **不主动反向写 PG**：operator 不感知 Compute 的 `jobs` / `services` / `tenants` 表；状态推进由 Compute 侧 Informer 按 CR `status` 回流
- **status 单向权威**：operator 只写 MLJob `status`；Compute 只写 MLJob `metadata` / `spec`
- **spec 幂等 Patch**：相同 `spec` 重复 Apply / Patch 不得重建 Pod；只有语义字段变化才触发底层资源变更

## 2. CRD 契约

MLJob 为 namespaced CR，创建在租户 namespace 下。Compute 负责设置：

- `metadata.name`：来自 `jobs.name`
- `metadata.namespace`：来自 `tenants.namespace`
- `metadata.labels["axisml.io/job-id"]`：`jobs.id`
- `metadata.labels["axisml.io/tenant"]`：租户名
- `metadata.labels["axisml.io/queue"]`：Compute Queue 名

最小 `spec`：

```yaml
apiVersion: compute.axisml.io/v1alpha1
kind: MLJob
spec:
  framework: pytorch | tensorflow | mpi | custom
  image: string
  command: []        # optional
  args: []           # optional
  env: []            # optional, Kubernetes EnvVar 数组
  replicas: int      # >= 1
  queueName: string  # Volcano Queue CR name
  suspend: bool      # optional, 用户取消时优先 patch 为 true
  resources:
    requests: {}     # Kubernetes ResourceList
    limits: {}       # Kubernetes ResourceList
  placement:
    nodeSelector: {}
    tolerations: []  # Kubernetes Toleration 数组
```

Operator 负责把 `queueName` 写入 `PodGroup.spec.queue`，并给所有 Pod 设置 `schedulerName: volcano`。

## 3. Status 契约

最小 `status`：

```yaml
status:
  observedGeneration: int64
  phase: Pending | Running | Succeeded | Failed
  message: string
  startedAt: timestamp
  finishedAt: timestamp
```

Compute 映射规则：

| MLJob status.phase | jobs.status | 终态 |
| --- | --- | --- |
| `Pending` | `Pending` | 否 |
| `Running` | `Running` | 否 |
| `Succeeded` | `Succeeded` | 是 |
| `Failed` | `Failed` | 是 |

CR 删除事件由 Compute Informer 映射为 `Cancelled`（若 PG 中 Job 尚未进入终态）。

## 4. 底层资源

- 每个 MLJob 至少创建一个 Volcano `PodGroup`
- 分布式训练按 `replicas` 创建对应 Worker Pod / Job，`PodGroup.spec.minMember` 默认等于 `replicas`
- PodGroup / Pod 必须设置 ownerReference 指向 MLJob，保证 MLJob 删除后底层资源级联清理
- operator 不读写 Volcano Queue CR；Queue CR 由 Compute 独占维护
