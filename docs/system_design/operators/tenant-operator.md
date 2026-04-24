# Tenant Operator 详细设计

## 1. 与 Compute 的写路径契约

Compute 采用 Operation Outbox + reconciler 异步下发 CR（详见 [compute.md §5.2](../compute.md)）。Operator 必须满足以下契约：

- **`Create` 幂等**：相同 `metadata.name` 的二次 `Create()` 返回 409 `AlreadyExists`，且不引发副作用。Compute 收到 409 后会 `Get()` 现有 CR 并校验 label `axisml.io/tenant-id=<uuid>`；只有 label 一致才视为成功
- **不主动反向写 PG**：operator 不感知 Compute 的 `tenants` 表；状态推进由 Compute 侧 Informer 按 CR `status` 回流
- **status 单向权威**：operator 只写 Tenant `status`；Compute 只写 Tenant `metadata` / `spec`
- **配置补偿友好**：Tenant CR 被误删后，Compute 会按 PG 快照重建；operator 的 create / reconcile 必须可重复执行

## 2. CRD 契约

Tenant 为 cluster-scoped CR。Compute 负责设置：

- `metadata.name`：来自 `tenants.name`
- `metadata.labels["axisml.io/tenant-id"]`：`tenants.id`

最小 `spec`：

```yaml
apiVersion: compute.axisml.io/v1alpha1
kind: Tenant
spec:
  namespace: string
  displayName: string
  annotations: {}
  resourceQuota: {}  # optional，Kubernetes ResourceQuota spec 瘦封装
```

## 3. Status 契约

最小 `status`：

```yaml
status:
  observedGeneration: int64
  phase: Active | Suspended | Failed
  message: string
  namespaceReady: bool
```

Compute 映射规则：

| Tenant status.phase | tenants.status |
| --- | --- |
| `Active` | `Active` |
| `Suspended` | `Suspended` |
| `Failed` | `Suspended`，并写入 `message` |

Tenant CR ADD 事件会把 `Creating` 推进为 `Active`；若 operator 后续回流 `Suspended`，Compute 保持租户不可提交新 Job / Service。

## 4. 底层资源

- operator 负责创建 / 更新租户 namespace
- operator 负责按 `spec.resourceQuota` 创建 / 更新 namespace 内 ResourceQuota
- namespace / ResourceQuota 必须设置 ownerReference 或可追踪 label，便于 Tenant 删除时清理
- operator 不读写 Volcano Queue CR；Queue CR 由 Compute 独占维护
