# AxisML Roadmap

本文档是 AxisML 的版本路线图。它回答两个问题：

1. **初始版本（v0.1.0）包含什么** —— 当前 `docs/system_design/` 描述的最终态系统能力。
2. **接下来打算做什么** —— 按主题归类的 backlog，**不绑定版本号**，具体实现路径与发布节奏由维护者另行排期。

各组件详设（`overview.md` / `components/*.md` / `auth.md` / `deployment.md` / `infra.md` / `monitoring.md` / `wireframe.md`）只描述**最终态**，不再保留"后续工作 / 未来"章节——所有未落地的能力都汇总到本文。

---

## 状态约定

| 状态 | 含义 |
| --- | --- |
| **Shipped** | 已在某个 tagged 版本里发布 |
| **In progress** | 当前版本目标，设计已定稿、正在实现 |
| **Backlog** | 已纳入路线图，待规划具体实现路径与版本 |
| **Considering** | 在评估，可能不做 |
| **Out of scope** | 已明确不做（理由见末节） |

`Backlog` 条目按主题归类，主题之间**不分先后**；条目内部也不暗示顺序。

---

## Released / In progress

### v0.1.0 — MVP（初始版本，In progress）

单集群、单副本、本地用户体系，覆盖最小可用 ML 平台闭环。功能矩阵参考 [system_design/overview.md §3](system_design/overview.md#3-功能矩阵)。

**控制平面**

- 单集群部署形态（minikube + `axisml-infra` + `axisml-system` 两 Helm chart）
- **tenant-operator**：单二进制单副本；`Tenant` CR → Namespace + ElasticQuota + 默认 Secret / ConfigMap / SA / RBAC
- **compute-operator**：单二进制单副本；dispatcher + handler 路由 `MLJob` / `MLService`
- **cluster-manager**：`ResourcePool` CRD 的 admin REST 入口（内嵌 `spec.units[]`）
- **compute-service**：Tenant / Quota / Job / Service 业务服务 + 三类 CR reconciler（PG 权威）
- **artifact-hub**：制品元数据 + 双后端（zot / RustFS）分离

**计算后端（compute-operator handler）**

- `(native, job)` — 单 role、batch/v1 Job
- `(native, deployment)` — 无状态在线服务
- `(native, statefulset)` — 有状态在线服务（基础字段）

**制品（artifact-hub）**

- Kind = `model`（zot OCI）端到端：initiate / complete / resolve / GC
- Kind = `image` / `dataset` 元数据模型已定义（端到端在 Backlog）

**平台层（Platform + frontend）**

- 本地用户名/密码登录 + JWT 颁发（HS256 / 单 issuer）
- RBAC：4 个角色（platform-admin / tenant-admin / tenant-member / viewer）+ 租户级绑定
- UI（[wireframe.md](system_design/wireframe.md) 已完整 mockup 的部分）：
  - App Shell / 顶栏 / 侧栏
  - 系统管理 → 资源池（列表 / 详情 / 创建 / 编辑）
  - 系统管理 → 租户（列表 / 详情 / 创建 / 编辑 / 配额）

**基础设施（axisml-infra chart）**

- Envoy Gateway（HTTP listener，`axisml-gateway`）
- RustFS Standalone 单副本
- zot 单副本（filesystem 后端）
- Koordinator（koord-scheduler + ElasticQuota；gang scheduling 通过 scheduler-plugins PodGroup）
- GPU Operator（NVIDIA 驱动 + device plugin，无 MIG）
- kube-prometheus-stack（Prometheus + Grafana，默认 emptyDir）

**元数据数据库**

- in-cluster bitnami PostgreSQL 单实例（compute / artifacts / platform 各自 schema）

**可观测性**

- 各服务暴露 `/metrics`，ServiceMonitor 自动接入
- 控制面 Core + Platform 指标按 [monitoring.md](system_design/monitoring.md) 命名约定上报
- 日志通过 `kubectl logs` 查询；无集中式日志栈

---

## Backlog（按主题归类，未排期）

以下主题之间彼此独立，条目内部也不暗示先后顺序。具体进入哪个版本由维护者基于需求与依赖决定。

### 1. 控制面硬化

- Admission webhook（每个 operator / service 一个）
  - **tenant-operator**：`spec.namespace.name` / `spec.quotas[].{pool,name}` 不可变；跨 ns `sourceXxxRef` 白名单；`min/max` 结构校验；目标 Namespace allowlist / denylist；硬阻断非 Compute 主体的 `Tenant` 写请求
  - **compute-operator**：`spec.backend.{name, engine}` 不可变；`backend.config` 按 Handler 自带 schema 校验；外部漂移检测（非 Compute 主体 patch `spec` 时拒绝）
  - **compute-service**：`Tenant` / `MLJob` / `MLService` 单写约束硬化（Compute 为唯一 `metadata` / `spec` 写者）
  - **cluster-manager**：`ResourcePool` 的 `metadata.name` / `spec.units[].name` 不可变；`requests <= limits` 结构校验；删除 pool 时阻断有活跃 Job/Service 引用
- CRD 严格 schema：移除所有 CRD 的 `x-kubernetes-preserve-unknown-fields`；启用 OpenAPI 校验；显式声明 spec 子结构与 `phase` enum
- 展示性元数据（display name / description）与扩展位（labels / annotations）一律落 PG，CR 不承载
- Operator 多副本 HA：`replicas≥2` + leader election；多副本压测 + leader 切换验证
- Service 层多副本：API 水平扩，后台协程（reconciler / GC / Informer）leader-only

### 2. 计算后端生态扩展

- **Kubeflow Trainer 系列（MLJob）**
  - `(kubeflow-trainer, pytorchjob)` — 字段映射 / 状态映射 / `backend.config` schema
  - `(kubeflow-trainer, tfjob)`
  - `(kubeflow-trainer, mpijob)`
  - 多 role（master / worker / ps / launcher）独立扩缩；per-role ResourceUnit
- **KServe 系列（MLService）**
  - `(kserve, inference)` — `InferenceService`，扩展 role（`transformer` / `explainer`）
  - `(kserve, llminference)` — vLLM disaggregated / llm-d / NVIDIA Dynamo 下的 KV cache 传输契约（nixl / mooncake）、parallelism schema、autoscaler 接入
  - KServe scale-to-zero 与 compute quota 的交互模型
- **Custom backend**
  - `(custom, *)` — `backend.config` 的 `targetGVK` + JSONPath `fieldMappings` / `statusMappings` / `endpointPath` schema 与 unstructured 操作约定
  - Platform 前端 `(custom, *)` JSON schema 编辑器
- **Native 系列演进**
  - `(native, podgroup)`：sigs.k8s.io scheduler-plugins PodGroup + 裸 Pod，作为 MLJob gang scheduling 单 backend 路径
  - `(native, job)` Indexed Job 与 `podFailurePolicy` 直通策略
  - `(native, statefulset)` `volumeClaimTemplates` / `updateStrategy` 灰度更新与 pod-index 寻址
- Handler chart values 控制（按 backend 启停 RBAC 与 watch）

### 3. Artifact Hub 能力扩展

- Kind = `dataset` / `image` 端到端：initiate / complete / resolve / GC 全谓词覆盖
- 上传凭证续签：客户端检测 token 剩余 < 5min 时刷新（长时上传必经路径）
- Failed 重试：digest mismatch 后允许同 version 再次 initiate
- `pin=digest` 语义：OCI Kind 返回 `<name>@<digest>` 不可变形态；S3 Kind 在响应中把 digest 标 `pinned`
- 浏览器直传扩展（OCI Kind 在浏览器实现 chunked push）
- 制品配额：按 namespace / Kind 限制总大小、总数量、单版本大小；`size_bytes` 入表
- 制品签名 / SBOM（cosign / notation）、image Kind 漏洞扫描（trivy / grype）
- 镜像 Layer 浏览端点（zot manifest API 解析 + per-layer 大小）
- 数据集样本预览（按 `format` 取首 N 行）
- 跨集群同步（zot replication / RustFS 多区域复制）
- 反向孤儿对账升级为审计 + 灰名单清理
- 批量 namespace 删除端点（默认禁用，需 admin token）

### 4. Platform UX 与前端补齐

- **待补 wireframe → 实现**
  - Dashboard 首页卡片（我可见的租户 / 活跃任务 / 在线服务 / 工作区 / 配额 Top N / GPU 利用率趋势 / 最近事件流）
  - 工作区：列表 / 创建表单 / 详情（概览 · 访问 Tab）
  - 计算任务：列表 / 创建表单 / 详情（概览 · 副本 · 事件 · 日志 Tab）
  - 在线服务：列表 / 创建表单 / 详情（概览 · 访问 · 指标 Tab）
  - 制品中心：模型 / 镜像 / 数据集 列表骨架与详情页（概览 · 版本 · 后端 Tab）
  - 系统管理 · 用户与角色页面
  - 应用中心整套（见 §10）
- **计算任务**
  - 任务模板 / 重新提交 UX（spec 反填）
  - SSE / WebSocket 增量列表
  - compute service `/events` / `/pods` / `/pods/{pod}/logs` / `/pods/{pod}/events` 端点 + UI 接入
  - DAG 工作流（编辑器 + 后端执行）
- **在线服务**
  - `spec.route` 热更新与 `stripPathPrefix`（轮换 API key / 调限流不重建）
  - 流量切换与灰度 UI（weighted route / canary / 自动指标判定回滚）
  - 自动扩缩容（HPA / KEDA），含 `request_rate` 触发
  - 多 role 独立扩缩、多端口 / 多协议、API key 轮换 UI
  - LLM 专项指标看板（tokens/sec / TTFT / TBT / KV cache / batch utilization）
- **工作区**
  - 闲时自动 stop 配置入口
  - 孤儿 PVC 清理 UI（compute 侧 GC + Platform 反查展示）
  - 创建表单预设（镜像 + 启动命令 + 资源单元 一键填好）
  - SSH 接入面板
  - 多容器 Workspace（jupyter + tensorboard sidecar）
- **资源池 / 资源单元**
  - 按租户的池可见性（池 → 租户白名单）
  - 节点匹配预览 Tab
  - 池容量聚合（allocatable / requested）
  - 池间调度借用策略
  - 资源单元成本元数据 `cost_per_hour` 列
  - 列表页 `resource_unit_count` 改 LRU 缓存
  - Pool / Unit 用量回流：从 ElasticQuota 聚合
- **租户**
  - 配额硬校验 / 分层配额 UI（依赖上游 ElasticQuota `parent` 字段）
  - 「已归档租户」管理界面（restore 入口）
  - `initResources` 表单深度（Vault / Sealed Secrets 接入）
  - 租户克隆向导
- **审计日志 UI**：按 `target` 前缀检索（`tenant:` / `job:` / `service:` / `resource-pool:` / `workspace:`）

### 5. 身份与安全

- OIDC 接入：引入 `IdentityProvider` 抽象 + OIDC 适配；登录页支持外部跳转；`users` 表退化为身份缓存
- OIDC 登录页（跳转 UX）
- 集群内 mTLS：Platform ↔ 下游 / 下游 ↔ 下游全部走 mTLS；下游基于 SPIFFE ID 校验调用方，而非裸 `X-Axisml-User`
- Compute 主动鉴权（当前完全信任 Platform 注入的 `X-Axisml-User`）
- 细粒度权限：把全局 RBAC 矩阵拆细到对象级，引入 `permissions` / `role_permissions` 字典化表
- 加密源支持：KMS / Vault / Sealed Secrets 作为 `sourceSecretRef` 替代
- `initResources` templating：按 tenant 上下文（id / name / namespace）渲染 ConfigMap 数据
- 复制源 RBAC 收敛：把源 Namespace 限定为单一受控 Namespace

### 6. 基础设施 HA 与生产化部署

- **网关与流量**
  - `axisml-gateway` 增加 HTTPS listener；TLS 证书通过 `cert-manager` 或 Secret 注入
  - Gateway 级 `SecurityPolicy`（JWT / OIDC，对接调用方选定的 IdP）
  - 静态 HTTPRoute 增加 `BackendTrafficPolicy`（限流 + 熔断 + 重试）
  - 跨 namespace `ReferenceGrant` 模板（仅在跨 namespace `backendRef` 出现时启用）
- **对象存储 / OCI Registry**
  - RustFS 切换到 Distributed (4×4) 或 (16×1) 生产形态
  - 双层后端：zot 的 storage backend 配置成 S3 协议指向 RustFS
  - zot HA (3×) + 共享后端 + GC、垃圾清理 CronJob、scrub 配置
- **数据库**
  - 支持外部 PostgreSQL（`database.enabled=false` + `externalDatabase.*`）
  - 内置模式主备 / 只读副本配置（bitnami chart 原生支持）
  - 备份策略（CronJob + 远端对象存储）
- **持久化与发版**
  - 持久化 Prometheus 数据卷（默认 14 天保留期，可调）
  - resync 间隔等 Helm values 暴露（默认 10min，可调到分钟级）
  - ServiceAccount + RBAC 子能力的 Helm values 开关

### 7. GPU 与调度

- GPU Operator 启用 MIG（A100 / H100 多实例分区），节点级开关
- Koordinator 调度策略调优（gang scheduling timeout、quota borrow 策略）
- Koordinator 在/离线协同：启用 `koordlet` + `koord-descheduler`，引入 CoLocation / QoS、Pod 重平衡
- 分层配额：`spec.quotas[]` 引入 `parent` 字段，落到 ElasticQuota `quota.scheduling.koordinator.sh/parent` annotation

### 8. 可观测性深化

- 预置 Grafana Dashboard：集群、GPU、网关、调度（ElasticQuota / PodGroup）、控制面、业务
- 预置 AlertManager 告警规则：节点 NotReady、GPU 异常、PVC 容量、配额耗尽、调度滞后、API 错误率
- 业务级 SLO 配置（AlertManager 集成）
- 集中式日志（Fluent Bit + Loki / ClickHouse）
- 分布式追踪（OpenTelemetry），与调用方组件改造同步推进
- 独立 `audit_events` 表（与现有 `audit_logs` 解耦）

### 9. 多集群 / 多区域

- 多集群下的 token 边界：Platform 跨集群签发 JWT 时，按集群隔离 `iss` / `kid` 与 JWKS endpoint
- 顶栏增加集群切换器
- Koordinator 多集群联邦：跨集群配额、调度策略下沉
- 跨集群同步（与 §3 制品同步对齐）

### 10. 应用中心与扩展模块

- 应用中心（Agent / Skills / MCP）后端契约与数据模型
- 应用中心整套页面（列表 / 详情 / 创建表单）
- 共享文件存储（如 JuiceFS）：训练大数据集的 POSIX 挂载（仅在出现强 POSIX 需求时引入）

### 11. 运营与成本

- 按使用时长 × 单元成本的计费导出
- 资源单元 `cost_per_hour` 全链路（资源池 UI → Job/Service 成本估算 → 导出报表）
- Tenant 批量端点 + retention GC 守护进程
- compute 批量 `GetTenant` RPC（减少多租户 namespace 解析的 RPC 数）
- Job 元数据部分可变：`display_name` / `description` / `runPolicy.activeDeadlineSeconds`

---

## Out of scope

下列能力**有意不做**，避免反复提案：

- **DataVolume / 数据卷管理模块**：训练数据通过 dataset artifact + 工作区 PVC 已覆盖，不再引入独立 DV 概念
- **GPU 预热 / 镜像预拉**：依赖节点级 daemonset，与 GPU Operator / containerd 重叠，价值低于复杂度
- **跨制品引用 / ModelRef 概念**：已在历史 commit f29fe20 移除，artifact 直接以 `(namespace, kind, name, version)` 四元组寻址

---

## 如何提议变更

- **修正本文**：直接提 PR；条目移动（Backlog → In progress / Shipped）需在 PR 描述里说明依据
- **新增条目**：建议先开 GitHub Issue 讨论需求与边界，达成共识后再写入 Backlog 主题
- **挑战 Out of scope**：开 Issue 论证为什么外部环境变化使原结论不再成立
- **版本化发布**：当 In progress 条目落地、形成 tag 时，把对应条目挪到 Released 区块并补对应版本号

各组件的字段契约、关键机制与运行时形态是路线图条目的设计基础——具体细节请进入 `docs/system_design/` 查阅。

---

## 关联文档

- [system_design/overview.md](system_design/overview.md) — 系统级导航与高层模型
- [system_design/components/](system_design/components/) — 各组件最终态详设
- [system_design/auth.md](system_design/auth.md) / [database.md](system_design/database.md) / [deployment.md](system_design/deployment.md) / [infra.md](system_design/infra.md) / [monitoring.md](system_design/monitoring.md) / [wireframe.md](system_design/wireframe.md) — 横切主题
