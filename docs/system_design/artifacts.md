# AxisML Artifacts 详细设计

AxisML Artifacts 是平台的制品管理服务，承载模型、数据集、镜像、评估报告等所有"非运行态"资产的**元数据管理**与**引用寻址**。Artifacts 通过 REST API 暴露能力，调用方为 AxisML Platform / Gateway 与各 Operator，不直接对外部用户流量开放；`axisml-cli` 经由 Platform / Gateway 中转调用 Artifacts。

| 模块 | 职责 | 边界外 |
| --- | --- | --- |
| ArtifactRepo ([§4.2](#42-artifactrepo)) | 仓库 CRUD：命名、Kind 绑定、归属（租户私有 / 平台公共） | 任何与具体内容相关的字段 |
| Artifact ([§4.3](#43-artifact)) | 版本化制品 CRUD、状态机、引用解析（resolve） | 制品 bytes 的存取 |
| ArtifactHandler ([§4.4](#44-artifacthandler-接口) / [§5](#5-内置-kind)) | 按 Kind 路由到后端：URI 拼装 / 上传凭证签发 / 完整性校验 / 后端清理 | 实际 bytes 存储（由 zot / RustFS 承担） |
| GC worker ([§3.4](#34-gc-与生命周期清理)) | 清理过期 Uploading、级联删除、孤儿告警 | 反向孤儿主动清理（误删风险，仅告警） |

**核心范式**：元数据服务 / 存储后端分离——Artifacts 只持有元数据（PG），制品 bytes 由 cli 与集群内消费方凭 Artifacts 返回的引用或临时凭证**直连**对应后端（OCI 走 zot Registry，S3 走 RustFS）。服务自身不在数据通道上，不做大文件代理。

**关键不变式**：

> **PG 是元数据权威；OCI Registry / 对象存储是 bytes 权威。**
> Artifact 一旦进入 `Ready` 即不可变（spec / digest 锁定）；更新内容 = 在同 repo 下新建一个新 version。

**文档组织**：

- **Part I — 服务框架**（§1 架构总览 + §2 运行时契约）：服务的整体形态、代码布局与部署 / 副本 / 可观测契约。
- **Part II — 核心机制**（§3 跨模块通用机制）：元数据 / 字节分离原则、两阶段写路径、resolve 读路径、GC 生命周期、副本与选主；后续模块章节引用本节而不重复。
- **Part III — 领域模型与 API**（§4 领域模型、§5 内置 Kind、§6 API 设计、§7 外部系统协作）：表 schema、Kind 特化字段、REST 契约、外部依赖。
- **Part IV — 实施与验证**（§8 实现路径、§9 测试、§10 相关引用）：功能落地路线（MVP / 功能完善 / 未来规划）、测试层次、跨文档引用。

---

## Part I — 服务框架

> 本部分描述 Artifacts 服务的整体形态、代码布局与运维契约（部署、副本、可观测）。后续 Part II / III / IV 的内容都默认在本部分约定的框架内运行。

## 1. 架构总览

### 1.1 服务形态

Artifacts 是单一 Go 二进制（`axisml-artifacts`），以 Deployment 形态部署在 `axisml-system` namespace；REST API 通过 ClusterIP Service 暴露。服务内部按"API + Handler 注册表 + GC worker"三层组织，所有模块共享一个 Manager（无独立 Manager / leader 拆分）。

```
┌─────────────────────────────────────────────────────────────────────┐
│  AxisML Artifacts (Go binary, optionally leader-elected)            │
│                                                                      │
│  HTTP Server (chi / std net/http)                                    │
│   ├── /api/v1/.../artifact-repos                Repo CRUD            │
│   ├── /api/v1/.../artifacts:initiate            Two-Phase write 1    │
│   ├── /api/v1/.../artifacts/{v}:complete        Two-Phase write 2    │
│   ├── /api/v1/.../artifacts/{v}:resolve         Read path            │
│   └── /api/v1/.../artifacts/{v}                 DELETE / GET         │
│                                                                      │
│  ArtifactHandler Registry (compile-time init())                      │
│   └── handlers/{model, dataset, image, evalreport}                   │
│                                                                      │
│  GC Worker (leader-only goroutine, scan PG every 5 min)              │
└──────────┬──────────────────────────────────────────┬───────────────┘
           │                                          │
           ▼                                          ▼
   ┌──────────────────────┐         ┌─────────────────────────────┐
   │   PostgreSQL         │         │   Storage Backends           │
   │   artifact_repos /   │         │   ┌──────────┐ ┌──────────┐  │
   │   artifacts          │         │   │   zot    │ │  RustFS  │  │
   │   (元数据权威)        │         │   │  (OCI)   │ │   (S3)   │  │
   └──────────────────────┘         │   └──────────┘ └──────────┘  │
                                     └──────────────▲──────────────┘
                                                    │
                                       直传（push / PUT），不经 Artifacts
                                                    │
                                         ←── 来自 cli / workload
```

### 1.2 调用形态

```
┌─────────────────┐   ┌─────────────────┐   ┌─────────────────┐
│   axisml-cli    │   │  AxisML Platform│   │    Operators    │
│   (用户侧 CLI)  │──▶│  (UI / Gateway) │   │  (内部消费方)    │
└─────────────────┘   └────────┬────────┘   └────────┬────────┘
                               │                     │
                               │ REST + identity     │ REST + tenant ctx
                               └──────────┬──────────┘
                                          ▼
                              AxisML Artifacts (REST API)
```

- `axisml-cli` 与 UI 经 Platform / Gateway 中转调用 Artifacts；Platform 负责认证并注入用户身份与角色 header（§6.2）
- 集群内 Operators（mljob / mlservice）直连 Artifacts ClusterIP，仅用于读路径（`usage=inspect`）
- cli 拿到 Artifacts 返回的 URI 与短期凭证后，直接与 zot / RustFS 交互完成上传 / 下载，bytes **不经过** Artifacts 服务

### 1.3 模块边界

制品分两级：**ArtifactRepo**（仓库 / 组织外壳）+ **Artifact**（具体版本化制品）；后者承载几乎全部业务信息。Kind 通过编译期 ArtifactHandler registry 扩展：内置 Model / Dataset / Image / EvalReport 四个 Kind；新增 Kind 不需要 DB 表结构迁移，但需要新增 handler、扩展 OpenAPI 枚举与允许列表。

详细模块清单见文档开头表格；模块之间的接口契约见 §4.4。

## 2. 运行时契约

### 2.1 代码布局

```
components/artifacts/
├── cmd/artifacts/             # 服务入口 main.go
├── api/
│   ├── openapi.yaml           # OpenAPI 3.0 契约源
│   └── types/                 # oapi-codegen 生成的 request/response types + server stub
├── internal/
│   ├── server/                # HTTP router、middleware（身份解析、错误处理、metrics）
│   ├── repo/                  # ArtifactRepo CRUD
│   ├── artifact/              # Artifact CRUD、状态机、resolve、initiate / complete
│   │   └── handler/           # Kind handler 注册表（model / dataset / image / evalreport）
│   ├── storage/               # 后端 client 抽象
│   │   ├── oci/               # zot 客户端 + scope token 签发
│   │   └── s3/                # RustFS 客户端 + prefix-scoped STS 签发
│   ├── gc/                    # GC worker（leader-only）
│   ├── db/                    # GORM 客户端 + golang-migrate 迁移
│   ├── tenantresolver/        # 只读解析 compute tenants（name / status / namespace）
│   └── auth/                  # 从 X-Axisml-User / X-Axisml-Roles 解析调用方身份
└── pkg/                       # Artifacts 内部可复用工具（错误、分页）
```

跨组件复用的公共库（日志、配置、错误）放在仓库根 `pkg/`。

### 2.2 Flag 与 Helm values 接口

| Flag | 默认 | 说明 |
| --- | --- | --- |
| `--listen-address` | `:8082` | REST API 端口 |
| `--metrics-bind-address` | `:8080` | Prometheus metrics |
| `--health-probe-bind-address` | `:8081` | `/healthz` / `/readyz` |
| `--leader-elect` | `true` | leader election（多副本时启用） |
| `--leader-election-id` | `axisml-artifacts.axisml.io` | Lease 名 |
| `--db-dsn` | （Secret 注入） | Postgres DSN |

```yaml
artifacts:
  enabled: true
  image: { registry, repository, tag, pullPolicy }
  replicas: 1
  leaderElection: { enabled, id }
  resources: { requests, limits }
  storage:
    oci:    { endpoint, adminSecretRef }     # zot
    s3:     { endpoint, region, bucket, adminSecretRef }  # RustFS
  database:
    secretRef: { name, key }                  # PG DSN
  metrics:
    serviceMonitor: { enabled }
```

### 2.3 部署形态

- **镜像**：`ghcr.io/axisml/axisml-artifacts:<appVersion>`，端口 `8082/tcp`，启动命令 `/artifacts serve`
- **探针**：`GET /healthz`（进程存活）、`GET /readyz`（PG 连通；GC 就绪不计入 readiness）
- **Helm chart**：`deploy/helm/axisml-system/templates/artifacts/` 下提供 ConfigMap / Deployment / Service / ServiceAccount / Secret / ServiceMonitor 标准模板
- **引导数据**：启动时执行 migration（golang-migrate embedded）；不预置任何 ArtifactRepo
- **对外暴露**：Artifacts 不直接对集群外暴露；终端用户经 Platform / Gateway 中转拿到 URI 与凭证后直连 zot / RustFS

### 2.4 副本与 Leader Election

- **API 层无状态**：所有副本服务 HTTP，仅写 PG，水平扩容无需协调
- **GC worker 选主**：通过 `coordination.k8s.io/Lease` 确保单点扫描；`replicas=1` 时退化为单成员 lease
- **默认形态**：`replicas=1`；多副本仅出于可用性
- **readiness 不绑定 leader**：`/readyz` 仅校验 PG 连通，避免 leader 切换期间非 leader 副本被摘流量

### 2.5 可观测

`/metrics` 至少暴露：

| 指标 | 类型 | 含义 |
| --- | --- | --- |
| `axisml_artifacts_is_leader` | gauge | 当前副本是否为 leader（0/1） |
| `axisml_artifacts_uploading_count{kind}` | gauge | 当前 `status='Uploading'` 行数 |
| `axisml_artifacts_gc_actions_total{predicate,result}` | counter | GC 动作计数 |
| `axisml_artifacts_resolve_requests_total{kind,result}` | counter | resolve 请求计数 |
| `axisml_artifacts_initiate_duration_seconds{kind}` | histogram | initiate 端到端耗时 |
| `axisml_artifacts_complete_duration_seconds{kind}` | histogram | complete 端到端耗时（含后端 HEAD 校验） |
| `axisml_artifacts_api_request_duration_seconds{route,status}` | histogram | API 请求延迟 |

具体阈值与告警规则不在本文档定义，由运维侧 dashboard / runbook 承载。

---

## Part II — 核心机制

> 本部分集中跨模块**共享**的运行机制：元数据 / 字节分离原则、两阶段写路径、resolve 读路径、生命周期清理与状态机。Part III 的领域模型与 API 章节引用本节而不重复。

## 3. 跨模块通用机制

### 3.1 元数据 / 字节分离与权威划分

| 类别 | 权威方 | 流向 |
| --- | --- | --- |
| 元数据（repo / artifact 行） | PG | API → PG |
| 制品 bytes | zot / RustFS | cli → 后端（直传，不经服务） |
| digest（manifest 哈希 / 内容哈希） | 后端 | 后端 → complete API → PG |
| 上传凭证（push / PUT） | Artifacts 签发 | Artifacts → Platform → cli（短期 OCI token / prefix-scoped 临时 STS） |
| 下载凭证（pull / GET） | Artifacts 签发 | Artifacts → Platform → cli（短期 pull token / 临时 STS）；集群内 operator 走 inspect 路径，不签明文凭证 |

### 3.2 写路径：两阶段提交（initiate → complete）

Artifacts 没有 K8s CR 下发动作（与 compute 不同），采用与 S3 multipart upload 同构的两阶段提交。

**阶段 1：initiate**

```
cli → POST /artifact-repos/{repo}/artifacts:initiate
      body: {version, spec, display_name?, description?, labels?, annotations?}
```

1. API 校验：repo 存在 + 同 repo 内 version 唯一 + `Handler.ValidateSpec(spec)` 通过
2. PG insert artifact 行，`status='Uploading'`
3. `Handler.InitiateUpload` 签发上传凭证：
   - OCI（model / image）：调 zot 签发 scope-limited bearer token，scope=`repository:<scope>/<kind-prefix>/<repo>:push`，TTL=1h
   - S3（dataset / eval_report）：签发 prefix-scoped 临时 STS 凭证，scope 限定到 `<scope>/<kind-prefix>/<repo>/<version>/`，TTL=1h
4. 返回 `{artifact_id, uri, upload_credentials, expires_at}`

**阶段 2：cli 直传后端**（不经 Artifacts）

- OCI：`oras push <uri>` 或封装在 `axisml-cli model push` / `image push`
- S3：使用临时 STS 执行 multipart upload / PutObject 到限定 prefix；上传完成前必须写入 `<prefix>/artifact-manifest.json`（列出所有对象 path / size / sha256）

**阶段 3：complete**

```
cli → POST /artifact-repos/{repo}/artifacts/{version}:complete
      body: {digest}
```

1. Artifacts 调后端校验：
   - OCI：`HEAD /v2/<scope>/<kind-prefix>/<repo>/manifests/<version>` → 比对 cli 提交的 digest
   - S3：`HEAD` + `GET` 读取 `<prefix>/artifact-manifest.json`，按 canonical JSON 计算 SHA256 比对 cli 提交的 digest
2. 校验通过：PG update `status='Ready'`、`digest=<…>`、`ready_at=now()`
3. 确定性失败（digest mismatch / manifest 缺失 / spec 不匹配）：PG update `status='Failed'`、`message=<原因>`，保留行供调试
4. 后端临时错误（5xx / 网络抖动）：API 返回 5xx，PG 行保持 `Uploading`，允许 cli 稍后重试 complete

**幂等性**：

- 同 `(repo_id, version)` 重复 initiate：
  - 已有 `Uploading` 未过期 → 返回原 credentials
  - 已 `Ready` / `Deleting` / `Deleted` → 409 `AlreadyExists`（同 version 不可复活，避免 `<kind>/<repo>@<version>` 引用语义漂移）
  - 已 `Failed` → 409；想复用 version 需先 DELETE 该行
- complete 重复调用：
  - 当前 `Uploading` → 正常推进
  - 已 `Ready`：digest 一致 → 200；不一致 → 409 `DigestMismatch`
  - 已 `Failed` / `Deleting` / `Deleted` → 409

**TTL 兜底**：未在 24h 内 complete 的 `Uploading` 行由 GC 转 `Failed` 并清理后端残留 blob（见 §3.4）。

### 3.3 读路径：resolve

```
GET /artifact-repos/{repo}/artifacts/{version}:resolve?usage={inspect|download}
```

`usage` 决定凭证形态——集群内消费方与集群外 cli 对凭证的可用性不同，二者必须分开。Operator 只允许 `usage=inspect`，并必须携带租户 / workload namespace 上下文；`axisml-cli pull` 经 Platform / Gateway 走 `usage=download`。

| usage | 调用方 | 返回字段 | 凭证形态 |
| --- | --- | --- | --- |
| `inspect`（默认） | mlservice-operator / mljob-operator | `storage_kind` / `uri` / `digest` / `auth_hint?` | `auth_hint = {secret_ref, namespace, username_key, password_key}`：仅指针；Secret 必须存在于 workload namespace，消费方按 ref 从 Pod 挂载或 imagePullSecret 中读凭证，Artifacts 不返回明文。仅 S3 Kind 必须返回；OCI Kind 由 Pod `imagePullSecrets` 通过约定命名直接使用，可不返回 |
| `download` | `axisml-cli pull` | `storage_kind` / `uri` / `digest` / `pull_credentials` / `expires_at` | OCI：zot 签发 `pull` scope bearer token，TTL=1h；S3：返回 prefix-scoped 临时 STS 凭证（`access_key` / `secret_key` / `session_token` / `prefix`），TTL=1h |

公共字段：

| 字段 | 含义 |
| --- | --- |
| `storage_kind` | `oci` / `s3` |
| `uri` | 由 `Handler.BuildStorageURI` 即时拼装；OCI: `<oci-host>/<scope>/<kind-prefix>/<repo>:<version>`、S3: `s3://<bucket>/<scope>/<kind-prefix>/<repo>/<version>/` |
| `digest` | 从 PG 读；artifact 未到 `Ready` 时为空 |

**`auth_hint.secret_ref` 的命名约定**：

- OCI：`axisml-tenant-<tenant>-zot-pull`（imagePullSecret type）
- S3：`axisml-tenant-<tenant>-rustfs`（opaque type，含 `username` / `password`）
- 公共制品被租户 workload 消费时，仍返回租户 namespace 内的复制后 Secret；公共 Secret 仅作为 tenant-operator 的复制源

Artifacts 不创建任何 Secret——租户级 Secret 由 tenant-operator 按 `Tenant.spec.initResources` 落地，来源 Secret 由集群管理员预放在 `axisml-system`（详见 [operator.md §4.7](operator.md#4-tenant-controller)）。Artifacts 仅按约定返回 workload namespace 内的 `secret_ref`，不校验 Secret 是否存在（消费方拉取失败时上报）。

### 3.4 GC 与生命周期清理

GC worker（leader-only goroutine，每 5 分钟一轮）扫描 PG 四类谓词：

| 谓词 | 动作 |
| --- | --- |
| `artifacts WHERE status='Uploading' AND created_at < now() - 24h` | 标 `Failed`，调 `Handler.GCBackend` 清理后端残留 blob |
| `artifacts WHERE status='Failed' AND updated_at < now() - 30d AND deleted_at IS NULL` | 转 `Deleting`；保留 30 天供诊断 |
| `artifacts WHERE status='Deleting'` | 调 `Handler.GCBackend`；成功后 PG `status='Deleted'`、写 `deleted_at` |
| `artifact_repos WHERE status='Deleting'` | 把其下所有非 `Deleted` artifact 推 `Deleting`；全部完成后把 repo 标 `Deleted` 并写 `deleted_at` |

`Handler.GCBackend` 实现需保证幂等，并把"对象不存在"视作成功（吞掉 OCI `MANIFEST_UNKNOWN` / S3 `NoSuchKey` / 404）。

**反向孤儿**（后端有 blob 但 PG 无 Ready 行）：仅记日志告警，**不主动清理**。理由：管理员可能手动 push 调试 blob，误删风险高于孤儿空间成本；对账模式作为后续演进（§8.4）。

### 3.5 状态机

Artifact 状态机（与 ArtifactRepo 的 `Active / Deleting / Deleted` 三态独立）：

```
Uploading ──(complete + 后端 HEAD 通过)──▶ Ready
          ──(complete 失败 / GC TTL 超时)──▶ Failed

Ready / Failed ──(DELETE)──▶ Deleting ──(GCBackend 成功)──▶ Deleted
```

| 状态 | 含义 |
| --- | --- |
| `Uploading` | initiate 后的中间态；24h 内未 complete 由 GC 转 `Failed` |
| `Ready` | 上传完成、digest 锁定，**spec 不可变，digest 不可变**；可被 resolve、可被 DELETE |
| `Failed` | 上传失败 / 校验失败；行保留供调试；可被 DELETE 进入清理流程 |
| `Deleting` → `Deleted` | 删除中间态；`deleted_at` 在进入 `Deleting` 时写入 |

> **`Uploading` 替代 `Creating` 的命名取舍**：compute 中 `Creating` 表示"PG 已写、CR 待下发"。Artifacts 没有 CR 下发动作，对应阶段是"PG 已写、bytes 待上传"，命名 `Uploading` 更贴合实际语义。

---

## Part III — 领域模型与 API

> 本部分给出 Artifacts 的数据 schema、内置 Kind 的特化字段、REST API 契约与外部依赖系统的协作边界。

## 4. 领域模型

### 4.1 通用约定

- 通用字段 `id uuid` PK / `created_at` / `updated_at` / `deleted_at`
- `name` 字段 **DNS-1123** 兼容；repo 名进一步约束为 OCI repository 合法字符集
- `version` 字段采用 **OCI tag-safe** 子集（`A-Za-z0-9_.-`，长度 1–128，禁止 `/`），同时安全映射到 OCI tag 与 S3 prefix
- `kind` 枚举：`model` / `dataset` / `image` / `eval_report`，由 ArtifactHandler registry 校验
- 表结构遵循 K8s CR 三段式：**metadata** / **spec**（用户声明，进入 Ready 后冻结）/ **observed**（status 簇，仅服务端回写）
- repo / version 一旦创建即不复用，删除后也不释放唯一键，以保证 `<kind>/<repo>@<version>` 的跨组件持久引用语义

### 4.2 ArtifactRepo

仓库是同一 Kind、同一逻辑名下版本族的容器，与 OCI 的 "repository" 同义。设计上**只承载组织信息**，所有内容相关字段下沉到 artifact 行。

**数据模型**

```
artifact_repos(
  -- metadata
  id                  uuid PK,
  tenant_name         text NULL,            -- NULL = 平台公共空间；非空时由 TenantResolver 校验存在且 Active
  kind                text,                  -- 'model' / 'dataset' / 'image' / 'eval_report'
  name                text,                  -- DNS-1123；与 OCI repo 名一致
  display_name        text,
  description         text,
  labels              jsonb,
  annotations         jsonb,
  owner_user          text,

  -- observed
  status              text,                  -- Active / Deleting / Deleted
  latest_artifact_id  uuid NULL,             -- 指向 ready_at 最大的 Ready artifact

  created_at, updated_at, deleted_at
)
-- (tenant_name, kind, name) 唯一；tenant_name IS NULL 与 NOT NULL 各一条 unique index，避免 COALESCE sentinel
```

**多租户语义**

- `tenant_name` 非空 = 租户私有：仅该租户用户可见、可读、可写
- `tenant_name` 为 `NULL` = 平台公共空间：所有租户用户只读，写入需 `X-Axisml-Roles` 包含 `admin`
- `(tenant_name, kind, name)` 唯一：同一租户内同 Kind 不能重名；公共空间内独立命名空间
- 跨租户细粒度共享（除"公共"外的中间态）暂不支持，作为后续演进（§8.4）

**生命周期**

| 操作 | PG 侧 | 触发动作 |
| --- | --- | --- |
| 创建 | insert，`status='Active'` | — |
| 更新 | update（display_name / description / labels / annotations 可改；name / kind 不可改） | — |
| 删除 | `status='Deleting'`（暂不写 `deleted_at`，继续占住名称） | GC 级联推动其下所有 artifact 进 `Deleting`；全部完成后 repo 标 `Deleted` 并写 `deleted_at` |

`latest_artifact_id` 维护：complete 推进到 Ready 时按 `ready_at` 最大者更新；DELETE 当前 latest 时回填到次新 Ready row。

**默认仓库**：无。Helm 不预置任何仓库；由用户 / admin 按需创建。

### 4.3 Artifact

Artifact 是平台用户实际"引用 / 拉取 / 部署"的单位，承载几乎全部业务信息：版本、Kind 特化的 spec、可观察的状态簇。

**数据模型**

```
artifacts(
  -- metadata
  id            uuid PK,
  repo_id       uuid FK artifact_repos(id),
  version       text,                  -- repo 内唯一；OCI tag-safe；扮演 metadata.name 角色
  display_name  text,
  description   text,                  -- 此版本说明 / changelog
  labels        jsonb,
  annotations   jsonb,
  owner_user    text,

  -- spec：Kind 特化业务字段，进入 Ready 后冻结，详见 §5
  spec          jsonb,

  -- observed
  status        text,                  -- Uploading / Ready / Failed / Deleting / Deleted
  message       text,                  -- 失败原因或 GC 进度
  digest        text,                  -- complete 校验通过后写入；OCI Kind 用作不可变引用，S3 Kind 仅作完整性校验
  ready_at      timestamptz,

  created_at, updated_at, deleted_at
)
-- (repo_id, version) 唯一
```

**字段分组语义**

- **metadata**：身份与组织信息；`labels` / `annotations` / `display_name` / `description` 在任何状态阶段（包括 Ready 后）均可改
- **spec**：用户声明，**进入 Ready 后冻结**；想"改" → 同 repo 下新建版本
- **observed**：仅由服务端 / GC / 后端校验逻辑回写

**存储地址不入表**：`storage_kind` / `uri` / `size_bytes` 都不作列存储——`storage_kind` 是 `repo.kind` 的纯函数，`uri` 由 `Handler.BuildStorageURI(scope, repo.name, version)` 即时构造，避免 PG 与命名约定漂移。`digest` 是唯一入表的"内容哈希"——OCI Kind 用作不可变引用键 `<repo>@<digest>`；S3 Kind 仅作 manifest 完整性校验，URI 形态始终是 `<prefix>/<version>/`。

`<scope>` 规则：公共制品 = `system`；租户私有 = `tenants/<tenant_name>`。`<kind-prefix>` ∈ `models / datasets / images / eval-reports`。

**跨制品引用**

格式 `<kind>/<repo>@<version>`，例：

- `model/llama-7b@v3`
- `dataset/mmlu@v1`
- `image/training-base@2024-09`

由 `Handler.ValidateSpec` 在 initiate 阶段做存在性 + 可见性懒校验（公共空间不可依赖私有制品）。被引用方进入 `Deleted` 后引用方仍可保留 ref 字符串，resolve 时返回 410 Gone。由于 repo / version 不复用，410 不会变成"意外命中新内容"。

### 4.4 ArtifactHandler 接口

所有 Handler 必须实现以下方法（语言无关，描述责任）：

| 方法 | 责任 |
| --- | --- |
| `Kind()` | 返回 `model` / `dataset` / `image` / `eval_report`；用作 registry 主键 |
| `StorageKind()` | 返回 `oci` / `s3` |
| `BuildStorageURI(scope, repo, version)` | 即时拼装存储 URI；不读 PG / 不调后端 |
| `ValidateSpec(ctx, deps, spec)` | 校验 Kind 特化 spec 字段 + 跨制品引用；纯函数 + 注入的 lookup 接口；只返回 errors |
| `InitiateUpload(ctx, a)` | 签发上传凭证（OCI scope token / S3 prefix-scoped STS）；幂等 |
| `VerifyComplete(ctx, a, claim)` | 调后端 HEAD / GET manifest，校验 digest 匹配 |
| `GCBackend(ctx, a)` | 删除后端残留 blob；幂等；NotFound 视作成功（§3.4） |

**注册方式**：编译期注册（每个 handler 包 `init()` → 全局 registry，key=`Kind()`）。新增 Kind 不需要改 schema（`spec jsonb` + `kind text` 兼容），只需新增一个 handler 包并扩展 OpenAPI 枚举与允许列表。

**未注册 Kind 的兜底**：API 收到 `repo.kind` 不在 registry 的请求 → 直接 400，不创建 PG 行。

## 5. 内置 Kind

每个 Kind 列出**必填 spec 字段** + **存储后端形态** + **主要消费方**；完整字段见 OpenAPI。

### 5.1 model

- **StorageKind**：`oci`
- **必填 spec**：`framework`（`pytorch` / `tensorflow` / `onnx` / `safetensors` / `gguf` / `custom`）、`format`（OCI artifact media type）
- **推荐 spec**：`task`、`parameters`、`base_model_ref`、`training_dataset_ref`
- **URI**：`<oci-host>/<scope>/models/<repo>:<version>` 或 `...@sha256:<digest>`
- **ManifestType**：`application/vnd.oci.image.manifest.v1+json`，`artifactType` 字段承载 `spec.format`
- **上传**：OCI Distribution v2 push（`oras push` 或 `axisml-cli model push`）
- **消费**：mlservice-operator 解析 `MLService.spec.modelRef` → resolve → 注入 KServe `predictor.storageUri`（补 `oci://` 前缀）或 native env `AXISML_MODEL_URI`

### 5.2 dataset

- **StorageKind**：`s3`
- **必填 spec**：`format`（`parquet` / `jsonl` / `csv` / `webdataset` / `tfrecord` / `custom`）
- **推荐 spec**：`schema`、`num_records`、`total_size`、`license`、`splits`
- **URI**：`s3://axisml-artifacts/<scope>/datasets/<repo>/<version>/`
- **上传**：RustFS S3 multipart / PutObject，凭证为 prefix-scoped 临时 STS；上传完成前写入 `artifact-manifest.json`
- **digest**：服务端读取 manifest 后按 canonical JSON 计算 SHA256
- **消费**：mljob-operator 注入容器 env `AXISML_DATASET_URI` / `AXISML_DATASET_DIGEST`，并通过 init container 或 csi-s3 volume 挂载到 `/data`

### 5.3 image

- **StorageKind**：`oci`
- **必填 spec**：`purpose`（`training` / `inference` / `dev`）
- **推荐 spec**：`base_image`、`platforms`、`entrypoint`、`notes`
- **URI**：`<oci-host>/<scope>/images/<repo>:<version>`
- **上传**：仍走两阶段 initiate / complete，但阶段 2 由本机 docker / nerdctl daemon 完成 push；cli 子命令 `axisml-cli image push` 把三步封装为单命令
- **消费**：mljob-operator / mlservice-operator 用 URI 作为 Pod `spec.containers[].image`；imagePullSecret 走约定命名（§3.3）

### 5.4 eval_report

- **StorageKind**：`s3`
- **必填 spec**：`model_ref`（被评测模型）、`dataset_ref`（评测数据集）、`metrics`（关键指标抽取，便于检索）、`evaluator`（评测器标识）
- **推荐 spec**：`notes`
- **URI**：`s3://axisml-artifacts/<scope>/eval-reports/<repo>/<version>/`
- **digest**：同 dataset，必须上传 `artifact-manifest.json`
- **消费**：平台 UI 列表页直接读 `spec.metrics` 排序展示，详情页通过 resolve 跳转下载报告原文

## 6. API 设计

### 6.1 路径规划

Artifacts 所有 API 置于 `/api/v1` 前缀下。

| 资源组 | 路径 | 主要动作 |
| --- | --- | --- |
| 健康检查 | `/healthz`、`/readyz` | Liveness / Readiness |
| 租户私有仓库 | `/api/v1/tenants/{tenant}/artifact-repos` | Create / Get / List / Update / Delete |
| 公共仓库 | `/api/v1/public/artifact-repos` | Get / List（普通用户）；Create / Update / Delete（admin） |
| 仓库下版本 | `/api/v1/.../artifact-repos/{repo}/artifacts` | List；GET 单个 |
| 注册版本 | `/api/v1/.../artifact-repos/{repo}/artifacts:initiate` | POST：注册元数据 + 签发上传凭证 |
| 完成上传 | `/api/v1/.../artifact-repos/{repo}/artifacts/{version}:complete` | POST：校验 digest + 转 Ready |
| 解析引用 | `/api/v1/.../artifact-repos/{repo}/artifacts/{version}:resolve` | GET：返回 uri / digest / auth_hint |
| 删除版本 | `/api/v1/.../artifact-repos/{repo}/artifacts/{version}` | DELETE |
| 删除仓库 | `/api/v1/.../artifact-repos/{repo}` | DELETE（级联） |

`...` 表示 `/tenants/{tenant}` 或 `/public` 二选一。

### 6.2 身份上下文

由 Platform / Gateway 注入的请求头：

| Header | 含义 |
| --- | --- |
| `X-Axisml-User` | 调用方用户唯一 ID，用于审计与 ownership |
| `X-Axisml-Roles` | 逗号分隔角色列表（`user` / `admin`），由 Platform 鉴权后注入 |

Artifacts 不重做角色鉴权；只校验：

- 路径中的租户存在且激活（通过 `TenantResolver` 只读解析）
- 相关资源归属于该租户
- 公共空间的写动作要求 `X-Axisml-Roles` 包含 `admin`

Operator 直连 Artifacts 时不代表终端用户身份，只携带 controller service identity 与明确的 tenant / workload namespace 参数；只允许访问 `resolve?usage=inspect`。

### 6.3 契约管理

`components/artifacts/api/openapi.yaml` 是唯一契约源，使用 `oapi-codegen` 生成 server stub 与各调用方的 Go SDK（Platform / Operators / cli 共用）。

### 6.4 cli 协作时序（model push 为例）

```
cli                  Artifacts                zot                       PG
 │                     │                       │                         │
 │ ───── initiate ──▶ │                       │                         │
 │                     │ ── ValidateSpec ────▶│                         │
 │                     │ ── insert(Uploading) ──────────────────────────▶│
 │                     │ ── 签 OCI scope token │                         │
 │ ◀── creds, uri ──── │                       │                         │
 │                                                                       │
 │ ───── oras push <uri> ────────────────────▶│                         │
 │ ◀── digest ──────────────────────────────── │                         │
 │                                                                       │
 │ ───── complete(digest) ─▶                    │                         │
 │                     │ ── HEAD manifest ───▶│                         │
 │                     │ ◀── 200 + digest ────│                         │
 │                     │ ── update(Ready, digest) ──────────────────────▶│
 │ ◀── 200 ──────────── │                       │                         │
```

S3 路径（dataset / eval_report）类似，把 `oras push` 替换为 S3 multipart upload，并在 complete 前上传 `artifact-manifest.json`。

## 7. 外部系统协作

| 对象 | 交互方式 | 关注点 |
| --- | --- | --- |
| **zot** | OCI Distribution v2；Artifacts 持 admin 凭证用于校验 / GC / 签发 scope token，cli 持短期 push / pull token 直连 | 由 axisml-infra chart 提供；admin 凭证存于平台级 Secret，endpoint 由 ConfigMap 注入 |
| **RustFS** | S3 协议；Artifacts 签发 prefix-scoped 临时 STS；bucket `axisml-artifacts` 由 axisml-infra chart 创建 | 区分 prefix `<scope>/datasets/...` / `<scope>/eval-reports/...`；目录级完整性由 `artifact-manifest.json` 表达 |
| **PostgreSQL** | GORM；与其他服务共用 database `axisml`；表前缀 `artifact_*` | 迁移随二进制打包，启动时执行（golang-migrate） |
| **ml-compute（tenants 表）** | Artifacts 通过 `TenantResolver` 把 URL 中的 tenant name 解析为租户 namespace 并校验 `status='Active'`；当前共库直读，后续拆库时改为 compute 暴露 `GET /api/v1/tenants/{name}` internal API | 仅依赖 `tenants.name` / `namespace` / `status`，不建立 FK，不写 Compute 表 |
| **tenant-operator** | Artifacts 不创建 Secret；约定平台默认让 `Tenant.spec.initResources.imagePullSecrets[].name='zot-pull'`、`secrets[].name='rustfs'`，由 tenant-operator 按 `axisml-tenant-<tenant>-<name>` 命名规则落地（[operator.md §4](operator.md#4-tenant-controller)） | resolve 的 `auth_hint.secret_ref` 始终指向 workload namespace 内的租户本地 Secret |
| **mlservice-operator** | 通过 Artifacts client SDK 解析 `spec.modelRef` → KServe `storageUri` / `AXISML_MODEL_URI` env | 详见 [operator.md §6](operator.md#6-mlservice-controller) |
| **mljob-operator** | 通过 Artifacts client SDK 解析 `spec.imageRef` → Pod `image`；`spec.datasetRef` → 容器 env / volume | 详见 [operator.md §5](operator.md#5-mljob-controller) |
| **Platform / Gateway** | REST + UI 展示；同时是 `X-Axisml-User` / `X-Axisml-Roles` 的注入方 | 终端用户与 `axisml-cli` 均经 Platform / Gateway 中转 |

---

## Part IV — 实施与验证

> 本部分给出 Artifacts 的功能落地路线、测试策略与跨文档引用。新贡献者读完前三部分后从这里看"先做什么、再做什么、怎么验证"。

## 8. 实现路径

按功能优先级把交付内容映射到三个阶段。MVP 划定"能跑通端到端最小可发布范围"，功能完善覆盖主流场景与生产硬化，未来规划承接需求未明朗或上游依赖未稳定的方向。

### 8.1 阶段总览

```
┌──────────────────────────────────────────────────────────────┐
│ MVP（最小可发布）                                             │
│   单一 Pod / 仅 model Kind / 两阶段写 + resolve / 仅租户私有 │
│   / GC TTL 兜底 / 单副本                                      │
│   ↓                                                           │
│ 功能完善（生产硬化）                                          │
│   补齐 dataset / image / eval_report Kind、公共空间、         │
│   auth_hint、Failed 重试、上传凭证续签、多副本选主             │
│   ↓                                                           │
│ 未来规划（需求 / 上游驱动）                                    │
│   细粒度 ACL、配额、cosign 签名、跨集群同步、lineage、         │
│   漏洞扫描                                                    │
└──────────────────────────────────────────────────────────────┘
```

每条目标都附完成信号，便于阶段闭合验证。

### 8.2 阶段一：MVP（最小可发布）

支撑端到端最小演示路径："创建 model 仓库 → initiate → cli `oras push` → complete → operator resolve → Pod 拉取"。

| 模块 | 范围 | 完成信号 |
| --- | --- | --- |
| 服务框架 | 单 Deployment + ConfigMap + Secret + Service；`/healthz` `/readyz`；启动 migration（§2.3） | 单 Pod 启动后 `/readyz` 返回 200，PG schema 自动迁移到当前版本 |
| 数据模型 | `artifact_repos` / `artifacts` 两表，三段式（metadata / spec / observed）；repo / version 全生命周期唯一（§4.1–4.3） | golang-migrate 启动时把表与 unique index 落地；重复 repo / version 创建返回 409 |
| Kind 支持 | 仅 `model`（OCI / zot），spec 必填 `framework` / `format`；其他 Kind 请求一律 400（§5.1） | `repo.kind='model'` 之外的 initiate 直接 400，不在 PG 留行 |
| 写路径 | initiate（PG + OCI scope-limited bearer token，TTL=1h）→ cli `oras push` → complete（HEAD `manifests/<version>` + digest 校验 → Ready）；同 version 重复以 409 / 200 处理（§3.2） | initiate 后立即 `oras push` + complete 链路通过；未 complete 24h 后 GC 转 `Failed`；digest mismatch → 409 `DigestMismatch` |
| 读路径 | `usage=inspect` 返回 `storage_kind` / `uri` / `digest`，**不返回 auth_hint**（operator 按约定命名 `axisml-tenant-<tenant>-zot-pull` 直接设 imagePullSecret）；`usage=download` 返回短期 OCI pull token（§3.3） | mlservice-operator 用 inspect 结果创建 Pod 能拉到 model；cli 用 download token 能 `oras pull` |
| 删除 | DELETE 单 artifact / DELETE repo（级联推 Deleting）（§4.2 / §4.3） | DELETE artifact → 状态 `Deleting` → GC 完成后端清理 → `Deleted`；DELETE repo 级联推所有 artifact |
| GC | 仅扫两类谓词：`Uploading > 24h → Failed`、`Deleting → 后端清理后 Deleted`（§3.4） | GC worker leader 单实例运行；扫描间隔 5min；GCBackend 对 NotFound 幂等 |
| 多租户 | 仅租户私有；`tenant_name` 必填；不开放 `tenant_name=NULL` 路径；不读 `X-Axisml-Roles`（§4.2） | `/api/v1/public/...` 路径返回 404；TenantResolver 校验 `tenants.status='Active'` |
| 部署 | 单副本 Deployment；ServiceMonitor 暴露 metrics（§2.3 / §2.5） | Helm install 后 `axisml_artifacts_uploading_count` 指标可被 Prometheus 抓取 |
| 测试 | API 层单元测试（chi router + handler 注册表）+ 集成测试（PG + 真实 zot in test container） | `make artifacts-test` 通过；happy path + 24h TTL fast-forward + DELETE 级联三类用例覆盖 |

### 8.3 阶段二：功能完善（生产硬化）

按"对生产可用性的影响"排序，每条标明完成信号。

1. **dataset Kind 上线**
   - 目标：S3 + RustFS 路径打通；`artifact-manifest.json` canonical SHA256 校验；prefix-scoped 临时 STS 凭证签发。
   - 完成信号：`(kind=dataset)` initiate 返回 STS 凭证；cli 用临时凭证完成 multipart upload + 写 manifest；complete 校验通过；mljob-operator 用 inspect 结果挂载 `/data`。
2. **image Kind 上线**
   - 目标：复用 OCI 路径，新增 `purpose` / `platforms` 字段；`docker push` / `nerdctl push` 与 cli `image push` 子命令两段封装。
   - 完成信号：`axisml-cli image push` 单命令完成三步；多架构 manifest list 写入 zot 后 inspect 返回正确 `platforms`。
3. **eval_report Kind 上线**
   - 目标：S3 路径，spec 必填 `model_ref` / `dataset_ref` / `metrics` / `evaluator`；引用方校验。
   - 完成信号：initiate 时被引用 model / dataset 不存在或不可见 → 直接 400；UI 列表页能按 `spec.metrics.accuracy` 排序展示。
4. **`auth_hint` 字段**
   - 目标：resolve `usage=inspect` 新增 `auth_hint = {secret_ref, namespace, username_key, password_key}`；S3 Kind 必须返回，OCI Kind 同步补齐。命名仍走约定 `axisml-tenant-<tenant>-{zot-pull,rustfs}`。
   - 完成信号：mljob-operator 解析 dataset inspect 后从 workload namespace 读 Secret 挂载到 csi-s3 volume；OCI Kind 的 `auth_hint` 与 imagePullSecret 约定一致。
5. **公共空间**
   - 目标：开放 `tenant_name=NULL` 路径；`X-Axisml-Roles` 注入路径完整接入；公共空间写动作要求 `admin`。
   - 完成信号：非 admin 用户对 `/api/v1/public/...` 写操作返回 403；公共制品被租户 workload 消费时 inspect 返回租户 namespace 内复制后 Secret。
6. **Failed 重试**
   - 目标：同 version 在 Failed 行上重新 initiate（先 GCBackend 清理后端残留，再重置 Uploading），返回 `previous_failure_reason`。
   - 完成信号：digest mismatch 后再次 initiate 同 version 返回新凭证 + `previous_failure_reason`，complete 成功后状态推进到 `Ready`。
7. **上传凭证续签**
   - 目标：`POST /...:initiate?refresh=true`，TTL 末尾 5min cli 主动续签，覆盖 1h 以上的大模型上传。
   - 完成信号：cli 检测到 token 剩余 < 5min 时调用 refresh 拿到新凭证；PG 行不变，仅重发 token / URL。
8. **`latest_artifact_id` 维护**
   - 目标：complete 与 DELETE 时维护"最近 Ready 时间"指针；UI 列表能按"最近成功上传"排序。
   - 完成信号：v2、v1、v3 乱序到达后 `latest_artifact_id` 始终指向 `ready_at` 最大者；DELETE 当前 latest 后回填到次新。
9. **跨制品引用懒校验**
   - 目标：`Handler.ValidateSpec` 注入 `LookupArtifact`，对 `base_model_ref` / `dataset_ref` / `model_ref` 做存在性 + 可见性校验；被引用方 Deleted 时 resolve 返回 410 Gone。
   - 完成信号：私有制品引用公共制品 OK；公共制品引用私有制品被 initiate 阶段拒绝；删除被引用 model 后引用方 resolve 返回 410。
10. **`pin=digest` 语义**
    - 目标：OCI Kind 返回 `<repo>@<digest>` 不可变形态；S3 Kind 仅在响应里把 digest 标 `pinned`。
    - 完成信号：mlservice-operator 用 `pin=digest` 拿到的 URI 部署能在 zot 重新打 tag 后仍稳定指向同一 digest。
11. **`auth_hint` ConfigMap 模板化**
    - 目标：从约定命名升级为 ConfigMap 模板渲染（兼容自定义 Secret 命名 / 与 tenant-operator initResources 解耦）。
    - 完成信号：管理员通过 Helm values 覆盖 `authHint.oci.tenantSecretTemplate` 后，inspect 返回的 `secret_ref` 跟随模板变化。
12. **多副本 + Leader Election**
    - 目标：API 层水平扩容；GC worker 走 `coordination.k8s.io/Lease` 选主；`/metrics` 暴露 `is_leader` gauge。
    - 完成信号：`replicas=2` 时 `sum(axisml_artifacts_is_leader) == 1`；leader 切换后 GC 不停摆且无重复扫描。
13. **GC 完整谓词**
    - 目标：补齐 `Failed > 30d → Deleting`、`Deleting → Deleted` 完整四谓词；30 天保留期可配置。
    - 完成信号：Failed 行 30 天后被 GC 自动推到 Deleting 并最终 Deleted；可观测 `gc_actions_total` 四个 predicate 都有计数。
14. **完整 metrics**
    - 目标：补 `initiate_duration_seconds` / `complete_duration_seconds` / `is_leader` / `resolve_requests_total{kind,result}`。
    - 完成信号：Prometheus 上述指标可见；P99 延迟曲线在 dashboard 可绘制。

### 8.4 阶段三：未来规划

- **细粒度 ACL**——跨租户共享、外部 SaaS 模型订阅；从"租户私有 / 公共"二元升级到三元。前置：明确至少 1 个真实业务场景。
- **配额管理**——按租户 / 按 Kind 限制总大小、总数量、单版本大小；`size_bytes` 入表，GC worker 维护用量。与 [operator.md §4](operator.md#4-tenant-controller) 的 ElasticQuota 体系对齐。
- **制品签名 / SBOM**——cosign / notation 集成，supply-chain 审计；签名信息作为 annotations 回写到 `artifacts` 行。
- **跨集群同步**——多 region / 多集群部署时镜像 / 模型自动同步（zot replication 原生支持）；S3 Kind 走 RustFS 多区域复制。
- **在线 lineage**——`base_model_ref` / `training_dataset_ref` / `eval_report.model_ref` 的依赖图可视化与影响面分析。
- **漏洞扫描集成**——image Kind 接入 trivy / grype，扫描结果作为 annotations 回写。
- **反向孤儿对账**——从"仅告警"升级为带审计 + 灰名单的清理模式。
- **TenantResolver 拆库形态**——当 Artifacts / Compute 拆库时改为 compute 暴露 `GET /api/v1/tenants/{name}` internal API。
- **预留 spec 列**——当 ArtifactRepo 引入 `visibility` / `mirror policy` / `quota` 等业务字段时再加 `spec jsonb` 列；当前不引入空列占位。

### 8.5 跨阶段验证策略

| 阶段 | 主测层 | 工具 |
| --- | --- | --- |
| MVP | API 层单元测试 + PG / zot 集成测试 | `make artifacts-test`（PG + zot testcontainer） |
| 功能完善 | 单元测试扩展 + RustFS / KServe / mljob 跨组件 e2e | `make e2e-test`（minikube，含 axisml-infra + axisml-system） |
| 未来规划 | 单独写 RFC 设计文档 → 单元 / 集成测试先行 → e2e 验证多组件链路 | 同上 |

## 9. 测试

Artifacts 的测试分两层：

- **单元 / 集成测试**（`components/artifacts/internal/...`）：HTTP handler、Handler registry、状态机、PG migration 用 `testify` + 真实 PG 容器（`testcontainers-go`）；OCI / S3 后端用 `zot` / `rustfs` testcontainer 起真实进程，避免 mock 与真实后端漂移。
- **L2 e2e**（`test/e2e/`）：与 [operator.md §8](operator.md#8-测试) 共享 minikube + axisml-infra + axisml-system 部署；典型用例覆盖"创建 Tenant → 推 model → mlservice 引用 → resolve 返回正确 URI"。

测试约束沿用 [CLAUDE.md "三层测试金字塔"](../../CLAUDE.md)：框架是 `testing` + `testify`，无 Ginkgo / Gomega；轮询用 `testutil.Eventually`。

## 10. 相关引用

- [docs/system_design/overview.md §3](overview.md) 概述了 Artifacts 在控制平面里的位置与功能矩阵。
- [docs/system_design/operator.md §4](operator.md#4-tenant-controller) 描述 tenant-operator 如何按 `Tenant.spec.initResources` 落地租户级 zot / RustFS 凭证 Secret——Artifacts `auth_hint` 的命名契约由其决定。
- [docs/system_design/operator.md §5–§6](operator.md) 描述 mljob / mlservice operator 如何通过 Artifacts client SDK 解析 `imageRef` / `modelRef` / `datasetRef`。
- [docs/system_design/compute.md §6](compute.md) 给出 `tenants` 表 schema——Artifacts `TenantResolver` 共库直读依赖于此。
- [docs/system_design/infra.md](infra.md) 给出 zot / RustFS / PostgreSQL 等基础设施的部署契约。
