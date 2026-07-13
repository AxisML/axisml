# 设计：Lite 单机 GPU 设备调度

范围：`axisml-lite`（Standalone Runtime）+ `axisml-system/compute-service` 共享状态机。

## 1. 目标

1. **可配置哪些卡用于调度**：管理员声明本机哪几张物理卡交给 AxisML 调度，其余保留。
2. **按空闲卡调度**：创建 workload 时把它绑定到未被占用的物理卡；无足够空闲卡则停在 `Pending`，
   卡空出后自动拉起，不超卖、不直接失败。

只做整卡独占调度。

## 2. 可调度 GPU 设备配置

GPU 设备是单台宿主机的物理属性，配置放在进程级 `AXISML_` env（随 docker-compose 注入），不进
ResourcePool CR。

```
AXISML_GPU_DEVICES=0,1,2     # 交给 AxisML 调度的物理卡 index 列表
AXISML_GPU_DEVICES=all       # 用 NVML 探测本机全部可用卡并托管
# 不设置                       # 不启用 GPU
```

| 取值 | 可调度卡集合 | 效果 |
|---|---|---|
| 未设置 / 空 | 空集 | 不启用 GPU。请求 GPU 的 workload → `Pending`，message="GPU 调度未启用（设置 AXISML_GPU_DEVICES）" |
| `all` | NVML 探测的全部可用卡 | 全部托管 |
| `0,1,2` | 指定索引 | 只在这些卡上调度，其余卡不碰 |

- `standalone.Config.GPUDevices` 启动时解析为 `[]int`：`all` 走 NVML（`github.com/NVIDIA/go-nvml`）枚举，
  索引列表直接解析（去重、非负校验，解析失败即启动报错）。
- NVML 探测失败或无卡时集合为空（GPU workload 停在 `Pending` 并给出 message），不 panic。
- 绑卡用卡的 index（Docker `DeviceRequest.DeviceIDs` 接受）；NVML 路径保留 index→UUID 映射，供需要稳定
  绑定时切换为 UUID（同一字段接受 `GPU-<uuid>`）。

## 3. GPU 分配器（`internal/runtime/standalone/gpu.go`）

分配以"当前存活的托管容器上的绑卡标签"为唯一事实来源，每次准入时从 Docker 现算，不维护跨重启账本。

创建容器时打上托管标签：

```
io.axisml.gpu-devices = "0,2"   # 该容器占用的物理卡 index
```

**占用判定（仅记账，不查 nvidia-smi）**：

```
busy = 所有"存活"托管容器的 io.axisml.gpu-devices 标签之并集
free = schedulable − busy
```

| 容器状态 | 是否占卡 |
|---|---|
| `running` / `restarting` / `created` / `paused` | ✅ 占用 |
| `exited` / `dead` / `removing` | ❌ 释放 |

跑完退出的容器立即让出卡，其标签仍保留供查历史分配。分配记账即独占，不看瞬时利用率；只感知 AxisML
自身占用，可调度集合内的卡不应被 AxisML 以外的进程占用（保留卡由 `AXISML_GPU_DEVICES` 排除）。

分配器接口：

```go
type gpuAllocator struct {
    schedulable []int          // 来自配置
    mu          sync.Mutex     // 串行化准入
}

// allocate 为一批尚无容器的 plan 挑选空闲卡（needPerPlan[i] = 第 i 个 plan 需要几张卡）。
// free = schedulable − 所有存活托管容器已占用的卡。
// sum(needPerPlan) > len(free)          → ErrResourceUnavailable，reason=transient，一张不占（原子）。
// sum(needPerPlan) > len(schedulable)   → ErrResourceUnavailable，reason=infeasible。
// 否则返回每个 plan 分到的 device index。
func (a *gpuAllocator) allocate(ctx, needPerPlan []int) ([][]int, error)
```

- **原子准入**：够才逐个创建容器，不够一张不占、一个容器不建；避免多副本半占用与卡死。
- **不重复计己方**：re-apply 时本 workload 已建副本按占用计入，只为尚未建的 plan 分配。
- **并发**：Run 与 Service 共用同一 `Runtime` 单例与同一把 `mu`，串行化"算空闲 → 绑定 → 创建"临界区；
  非 GPU workload 不走锁；镜像 pull 在临界区外。
- **无标签的遗留 GPU 容器**：存活、托管、`limits` 含 `nvidia.com/gpu>0` 但无 `io.axisml.gpu-devices`
  标签者，按其请求数从可调度容量保守扣减。

## 4. 绑卡

`ResourcePlan` 增加 `GPUDeviceIDs []string`；`toDocker` 据此绑定具体卡：

```go
if len(p.Resources.GPUDeviceIDs) > 0 {
    host.DeviceRequests = []container.DeviceRequest{{
        Driver:       "nvidia",
        DeviceIDs:    p.Resources.GPUDeviceIDs,   // 如 ["0","2"]
        Capabilities: [][]string{{"gpu"}},
    }}
}
```

流程：渲染 plan（从 `limits` 得 GPU 需求数）→ 分配器挑出 index → 写入 `plan.Resources.GPUDeviceIDs` 与
`io.axisml.gpu-devices` 标签 → 创建容器。GPU 请求全程经分配器，拿到卡才创建。

`planIdentity`（spec-hash）**不含** `GPUDeviceIDs`——identity 只认 GPU 请求数（在 `GPUCount` 里），
不认分到哪张卡，使 re-apply 不因重新分配而 churn 容器。

## 5. 状态模型与准入

| Phase | 含义 |
|---|---|
| `Creating` | 已入库、等待 reconciler 取出放置（放置前的排队态，取出即离开） |
| `Pending` | reconciler 已取出、正在放置：拉镜像 / 建容器 / 等待资源（等 GPU 在这里） |
| `Running` | 已放置并运行 |

`pkg/extensions` 新增类型化 error（与 `CapabilityError→409` 平行）：

```go
// ResourceUnavailableError：workload 合法但当前无足够可用资源，应保持等待并稍后重试。
type ResourceUnavailableError struct{ Msg string }
func IsResourceUnavailable(err error) bool
```

`ApplyMLRun`/`ApplyMLService` 在 GPU 不足时返回它（未创建任何容器）。等待态用统一状态机呈现为
`Pending`，四处改动（不加 marker 列、不改 status DTO）：

1. **reconciler `handleCreate`（mlrun + mlservice）**：取出 `Creating` 行即置 `phase = Pending`（进入放置），
   再拉镜像 / 建容器 / 准入。GPU 不足（`ErrResourceUnavailable`）时保持 `Pending` 并写 message
   （transient："等待可用 GPU（需 N，空闲 M）"；infeasible："请求 N 卡超过可调度容量 C"），info 日志 +
   独立 metric label。K8s runtime 不返回此 error，"等 GPU"这层只在 Lite 出现。
2. **`FindWorkSet`（create outbox）**：`phase IN ('Creating','Pending') AND deleted_at IS NULL`，让等待中的
   `Pending` 行每 tick 继续被尝试放置，卡空出即建容器进入 Running。
3. **`reflectGone`（mlrun + mlservice）**：只对 `Running`/`Ready`/`Degraded`/`Failed`（及 `Deleting`/
   `Canceling`）驱动 gone 迁移；`Pending` 行 Observe 到 NotFound 视为"尚在放置"→ no-op。带外删除一个尚未
   running 的 `Pending` 负载因此被 reconciler 自愈重建（DB 行为权威）；已 running 的负载消失仍反映为 gone。
4. **`kuberuntime.ApplyMLService`**：spec 未变则跳过 `Update`，使每 tick re-apply 廉价幂等。`ApplyMLRun`
   已是"存在即 no-op"，无需改。

时序：`Creating`（入库、待取出）→ reconciler 取出即 `Pending`（拉镜像 / 建容器 / 无卡则等待，每 tick 重试）
→ 容器 running → `Running`。

## 6. 重启恢复

进程重启后分配器下次 `allocate` 从存活容器的 `io.axisml.gpu-devices` 标签重建占用视图；运行中容器持续持有
其绑定卡，`DeviceIDs` 已固化在容器上，重启不丢。

## 7. 构建与部署

- `axisml-core` 以 `CGO_ENABLED=1` 构建（go-nvml 需 cgo）；交叉编译需对应 cgo 工具链。
- 最终镜像用 `gcr.io/distroless/base-debian12`（含 glibc）。
- 运行期由 NVIDIA container runtime 给 core 容器注入驱动（`NVIDIA_DRIVER_CAPABILITIES` 含 `utility`）供
  NVML 在运行时 `dlopen libnvidia-ml.so.1`；构建期不需要 libnvidia-ml。
- `docker-compose.yaml` 暴露 GPU 给 core 容器并给出 `AXISML_GPU_DEVICES` 示例。

## 8. 涉及改动的文件

**Lite：**
- `pkg/core/config.go` — `GPUDevices` env 配置（`all` / 列表解析）。
- `pkg/core/app.go` — 透传到 `standalone.Config`。
- `internal/runtime/standalone/runtime.go` — `Config.GPUDevices`、持有 `gpuAllocator`、`LabelGPUDevices`。
- `internal/runtime/standalone/gpu.go`（新增）— 分配器 + NVML `all` 探测。
- `internal/runtime/standalone/plan.go` — `ResourcePlan.GPUDeviceIDs`、`toDocker` 绑卡。
- `internal/runtime/standalone/{run.go,service.go}` — Apply 接入准入；`planIdentity` 排除 `GPUDeviceIDs`。
- `go.mod` — `github.com/NVIDIA/go-nvml`。
- `Dockerfile` — `CGO_ENABLED=1`；最终镜像 `distroless/base-debian12`。

**compute-service（Lite/Standard 共享）：**
- `pkg/extensions/` — `ResourceUnavailableError` + `IsResourceUnavailable`。
- `internal/mlrun/{reconciler.go,repository.go,reflow.go}` 及 `internal/mlservice/` 对应文件 —
  reconciler 置 `Pending` + message；`FindWorkSet` 纳入 `Pending`；`reflectGone` 去掉 `Pending` 的 gone 分支。
- `internal/kuberuntime/runtime.go` — `ApplyMLService` 幂等守卫。

**部署：**
- `axisml-lite/deploy/docker-compose.yaml` — `AXISML_GPU_DEVICES` 示例 + GPU 暴露。

## 9. 非目标 / 边界

- 不做部分卡 / MIG / 时间片共享；仅整卡独占。
- 不做 CPU / 内存容量准入（仍 best-effort）；仅 GPU 走准入门。
- 不做抢占、借用、优先级、公平性、gang scheduling、跨节点。
- 租户 quota 的 `nvidia.com/gpu: max` 仍是声明 + UI 展示，真正的门是设备分配器。
- 请求卡数 > 可调度容量的 workload 保持 `Pending` 并给出 infeasible message。

## 10. 测试计划

- **单元**（`internal/runtime/standalone`）：空闲计算（exited 视为释放、re-apply 不重复计己方、原子性）、
  `toDocker` 绑卡、配置解析。fake/mock Docker 列表驱动。
- **集成**（`test/integration`，不依赖真实 GPU，以标签模拟占用）：卡满 → `Pending` + message、Observe
  NotFound 不被误删/误取消 → 释放后自动拉起 → `Running`；重启后从标签重建占用；re-apply 不 churn 容器；
  Service 等待不被静默删除。
- **回归**：`kuberuntime.ApplyMLService` re-apply 幂等；`reflectGone` 对 `Pending` no-op 后更新既有用例。
- **真实 GPU 冒烟**（非 CI）：容器内 `nvidia-smi` 只见分配到的卡；两个各占 1 卡的 workload 落在不同物理卡。

## 11. 文档更新

- `axisml-lite/docs/system_design.md`：§2 能力矩阵 GPU 行、§5.1.1 配置、§6.5 资源/配额/GPU、§6.6 状态映射
  （`Creating→Pending→Running`，`Pending` 含"等待可用 GPU"）。
- compute-service 状态机文档：`Pending` = 正在放置 / 等待资源；`reflectGone` 只对已放置负载做 gone 处理。
- `axisml-lite/deploy/config/` 与 `docker-compose.yaml` 示例。
