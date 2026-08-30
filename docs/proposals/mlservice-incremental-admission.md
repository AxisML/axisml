# MLService 增量资源准入方案

- 状态：Implemented
- 日期：2026-08-30
- 影响范围：Compute Service、Platform Service / Workspace、Kubernetes Runtime、Standalone Docker Runtime

## 1. 决策

MLService、Workspace 与 TensorBoard 必须与 MLRun 一样尊重 runtime 容量、节点约束和 Tenant pool quota，但常驻服务允许部分副本先上线，并在每轮 admission 中先于尚未准入的 MLRun 获取新增容量。

初次创建以 `Queued` 持久化；此时 runtime 对象不存在且不占 reservation。至少一个最小服务单元能够完整放置后，PG 原子写入 admitted replica vector 并推进到 `Creating`。runtime 永远只接收 admitted 数量，不能在检查一个副本后提交完整 desired 数量。

## 2. 副本三层状态

| 层 | 权威字段 | 含义 |
| --- | --- | --- |
| desired | `spec.roles[*].replicas` | 用户目标；`/scale` 修改并增加 generation |
| admitted | `mlservices.admitted_replicas` | 已通过容量和 quota 检查的 durable reservation |
| ready | `status.readyReplicas` | runtime 实际通过 readiness 的主 role 副本 |

`dispatched_replicas` 记录 runtime 最后成功接受的 admitted vector，为 Apply 竞态失败提供精确回退边界。Platform 对主 role 暴露 `ready / admitted / desired`。

## 3. 生命周期

```text
POST → Queued → Creating → Pending → Ready
          │                       ↘ Degraded（部分 ready 或扩容待准入）
          └─ 无 runtime / 无 reservation
```

- 初次 0→1（多 role backend 为每个非空 role 各 1）是原子最小服务单元。
- 最小单元之后按 immutable role order 逐副本准入；一次循环可连续准入所有当前可放置副本。
- 扩容等待保持现有 `Pending` / `Ready` / `Degraded` 服务身份，不把整个对象退回 `Queued`。
- 缩容不需要 admission，立即降低 admitted 并同步 runtime；尚未退出的 allocation 仍由 inventory 计入。
- runtime 在 snapshot 后拒绝且保证未创建新增 instance 时，初次提交回到 `Queued`；扩容只把 admitted 回退到 dispatched，已有副本继续服务。

## 4. 排序与资源记账

同一 transaction-scoped PostgreSQL advisory lock 内：

1. 读取 runtime allocations 与现有 admitted reservations；
2. 按 `created_at ASC, id ASC` 处理 Service 初次创建和扩容；
3. 使用剩余容量按 priority/FIFO 处理 Queued MLRun。

Service 因而有隐式优先权，但不抢占已准入 Run。当前不能完整放置的下一个 Service 副本不做部分 reservation，允许 Run backfill，避免永久不可满足的 Service 阻塞整个 pool。

容量与 quota 使用量都是“已物化 allocation + 尚未物化 admitted reservation”；同一个副本不得重复计数。等待原因使用 `InventoryUnavailable`、`QuotaUnavailable`、`QuotaExceeded`、`NoMatchingNode`、`InsufficientResources`。

## 5. 恢复不变量

- `Queued`：admitted/dispatched 都为 0，runtime 对象不存在。
- `Creating`：至少一个最小服务单元已持久化 reservation，runtime 可能尚不存在。
- `admitted != dispatched`：reconciler 只渲染 admitted vector；成功后推进 dispatched。
- desired 全部 admitted 且 dispatched 后，才令 `observedGeneration == generation`。
- Compute 重启或 leader 切换不会丢失 desired、reservation 或最后一次 runtime 提交边界。
