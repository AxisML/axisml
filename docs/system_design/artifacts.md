# AxisML Artifacts 详细设计

## 1. 概述

AxisML Artifacts 是平台的制品管理服务，基于 Go 开发，承载模型、数据集、镜像、评估报告等所有"非运行态"资产的**元数据管理**与**引用寻址**。Artifacts 通过 REST API 暴露能力，调用方为 AxisML Platform / Gateway 与各 Operator，不直接对外部用户流量开放；`axisml-cli` 通过 Platform / Gateway 中转调用 Artifacts，由 Platform 注入用户身份与角色。

**核心范式**：元数据服务 / 存储后端分离——Artifacts 只持有元数据（PG），制品 bytes 由 `axisml-cli` / 集群内消费方凭 Artifacts 返回的引用或临时凭证**直连**对应后端（OCI 走内置 zot Registry，S3 走 RustFS），服务自身不在数据通道上，不做大文件代理。

**关键边界原则**：
- Artifacts 只写自己的 PG，不持有制品 bytes，不做用户身份提供（认证由 Platform 完成）
- 制品类型（Kind）通过编译期 handler registry 扩展：内置 Model / Dataset / Image / EvalReport 四个 Kind；新增 Kind 不需要 DB 表结构迁移，但需要新增 handler、更新 OpenAPI 枚举 / 允许列表与客户端契约
- 制品分两级：**ArtifactRepo**（仓库 / 组织外壳）+ **Artifact**（具体版本化制品）；后者承载几乎全部业务信息

## 2. 职责与边界

Artifacts 内部按 4 个模块划分：

| 类别 | 模块 | 职责 | 边界外 |
| --- | --- | --- | --- |
| 元数据 | ArtifactRepo | 仓库 CRUD：命名、Kind 绑定、归属（租户私有 / 平台公共） | 任何与具体内容相关的字段 |
| | Artifact | 版本化制品 CRUD、状态机、引用解析（resolve） | 制品 bytes 的存取 |
| 存储委派 | ArtifactHandler | 按 Kind 路由到后端：URI 拼装 / 上传凭证签发 / 完整性校验 / GC | 实际 bytes 存储（由 zot / RustFS 承担） |
| 后台 | GC worker | 清理过期 Uploading、级联删除、孤儿告警 | 反向孤儿主动清理（误删风险，仅告警） |

**关键不变式**：

> **PG 是元数据权威；OCI Registry / 对象存储是数据权威。**
> Artifact 一旦进入 `Ready` 即不可变（digest 锁定）；更新内容 = 在同 repo 下新建一个新 version。

## 3. 整体架构

```
   ┌─────────────────┐   ┌─────────────────┐   ┌─────────────────┐
   │   axisml-cli    │   │  AxisML Platform│   │    Operators    │
   │   (用户侧 CLI)  │──▶│  (UI / Gateway) │   │  (内部消费方)    │
   └─────────────────┘   └────────┬────────┘   └────────┬────────┘
                                  │                     │
                                  │ REST / JSON         │ REST / JSON
                                  │ + identity headers  │ + tenant context
                                  └──────────┬──────────┘
                                             ▼
                   ┌──────────────────────────────┐
                   │   AxisML Artifacts (Go)       │
                   │   ┌──────────────────────┐    │
                   │   │ Repo / Artifact / GC │    │
                   │   │ ArtifactHandler      │    │
                   │   └──────────────────────┘    │
                   └──┬───────────────────────┬───┘
       ┌──────────────┘                       └────────────────┐
       ▼                                                        ▼
 ┌──────────────────────┐                    ┌─────────────────────────────┐
 │   PostgreSQL          │                    │   Storage Backends           │
 │   artifact_repos /    │                    │   ┌──────────┐ ┌──────────┐  │
 │   artifacts           │                    │   │   zot    │ │  RustFS  │  │
 │   (元数据权威)         │                    │   │  (OCI)   │ │   (S3)   │  │
 └──────────────────────┘                    │   └──────────┘ └──────────┘  │
                                              └──────────────▲──────────────┘
                                                             │
                                                  直传（push / PUT）
                                                  不经 Artifacts 服务
                                                             │
                                                  ←── 来自 cli / workload
```

**读写模型**：

- **API 写**：仅落 PG 元数据；上传凭证由 Artifacts 即时签发，`axisml-cli` 经 Platform 取得凭证后直连后端 push
- **API 读**：从 PG 解析元数据，按需即时拼装 `uri` 返回；不读后端 bytes
- **后台 GC**：扫 PG 谓词，调对应 Handler 操作后端

## 4. 代码布局

```
components/artifacts/
├── cmd/artifacts/             # 服务入口 main.go
├── api/
│   ├── openapi.yaml           # OpenAPI 3.0 契约源
│   └── types/                 # oapi-codegen 生成的 request/response types + server stub
├── internal/
│   ├── server/                # HTTP router、middleware（身份解析、错误处理、访问日志、metrics）
│   ├── repo/                  # ArtifactRepo CRUD
│   ├── artifact/              # Artifact CRUD、状态机、resolve、initiate/complete
│   │   └── handler/           # Kind handler 注册表
│   │       ├── model/
│   │       ├── dataset/
│   │       ├── image/
│   │       └── evalreport/
│   ├── storage/               # 后端 client 抽象
│   │   ├── oci/               # zot 客户端 + scope token 签发
│   │   └── s3/                # RustFS 客户端 + prefix-scoped STS 签发
│   ├── gc/                    # GC worker（leader-only）
│   ├── db/                    # GORM 客户端 + golang-migrate 迁移
│   ├── tenantresolver/        # 只读解析 Compute tenants(name/status/namespace)，不建跨服务 FK
│   └── auth/                  # 从 X-Axisml-User header 解析调用方身份
└── pkg/                       # Artifacts 内部可复用工具（错误、分页）
```

跨组件复用的公共库（日志、配置、错误）放在仓库根 `pkg/`。

## 5. 运行机制

### 5.1 权威划分

> **PG 是元数据权威；后端存储是 bytes 权威。Artifacts 只写自己的 PG。**

| 类别 | 权威方 | 流向 |
| --- | --- | --- |
| 元数据（repo / artifact 行） | PG | API → PG |
| 制品 bytes | zot / RustFS | cli → 后端（直传） |
| digest（manifest 哈希 / S3 内容哈希） | 后端 | 后端 → complete API → PG（落入 `artifacts.digest`） |
| 上传凭证（push / PUT） | Artifacts 服务签发 | Artifacts → Platform → cli（短期 OCI token / prefix-scoped 临时 STS） |
| 下载凭证（pull / GET） | Artifacts 服务签发 | Artifacts → Platform → cli（短期 pull token / 临时 STS）；集群内 operator 走 `auth_hint` 路径，不签发明文凭证 |

### 5.2 写路径（Two-Phase Register-Upload）

Artifacts 没有 K8s CR 下发动作（与 compute 不同），采用更轻的**两阶段提交**：

**阶段 1：initiate**

```
cli → POST /artifact-repos/{repo}/artifacts:initiate
      body: {version, spec, display_name?, description?, labels?, annotations?}
```

1. API 校验：repo 存在 + 同 repo 内 version 唯一 + `Handler.ValidateSpec(spec)` 通过
2. PG insert artifact 行，`status='Uploading'`
3. `Handler.InitiateUpload` 签发上传凭证：
   - OCI（model / image）：调 zot 签发 scope-limited bearer token，scope=`repository:<scope>/<kind-prefix>/<repo>:push`，TTL=1h
   - S3（dataset / eval_report）：统一签发 prefix-scoped 临时 STS 凭证，scope 限定到 `<scope>/<kind-prefix>/<repo>/<version>/`，TTL=1h
4. 返回 `{artifact_id, uri, upload_credentials, expires_at}`

> **TTL 取舍**：1h 覆盖大多数大模型 / 数据集上传场景（10 GB 量级 ≤ 1h 的网络）。超过 1h 的极端场景由 cli 在剩余 5min 时调用 `:initiate?refresh=true` 续签新凭证（不改 PG 行、仅重发 token / URL），避免引入更长 TTL 带来的撤销难题。

**阶段 2：cli 直传后端**（不经 Artifacts）
- OCI：`oras push <uri> ...` 或封装在 `axisml-cli model push`
- S3：使用临时 STS 执行 S3 multipart upload / PutObject 到限定 prefix；上传完成前必须写入 `<prefix>/artifact-manifest.json`

**阶段 3：complete**

```
cli → POST /artifact-repos/{repo}/artifacts/{version}:complete
      body: {digest}
```

1. Artifacts 服务调后端校验：
   - OCI：`HEAD /v2/<scope>/<kind-prefix>/<repo>/manifests/<version>` → 比对 cli 提交的 digest
   - S3：`HEAD` 确认 `<prefix>/artifact-manifest.json` 存在后 `GET` manifest 对象，按 canonical JSON 计算 SHA256 并比对 cli 提交的 digest
2. 校验通过：PG update `status='Ready'`、`digest=<…>`；同时按 §6.2 规则刷新 `artifact_repos.latest_artifact_id`
3. 确定性校验失败（digest mismatch / manifest 缺失 / spec 不匹配）：PG update `status='Failed'`、`message=<原因>`；保留行供调试
4. 后端临时错误（zot / RustFS timeout、5xx、网络抖动）：API 返回 5xx，PG 行保持 `Uploading`，允许 cli 稍后重试 complete

> **类比**：本流程与 S3 multipart upload 的 `CreateMultipartUpload → UploadPart → CompleteMultipartUpload` 两段提交同构。与 compute 的 Outbox 不同：compute Outbox 是"先写 PG 再异步 reconcile 到 K8s"以保证最终一致；这里 initiate / complete 都是同步事务，没有事件表也没有重放需求，只保留一个低频 GC worker 兜底未完成的 Uploading。

**幂等性**：
- 同 `(repo_id, version)` 重复 initiate：
  - 已有 `Uploading` 未过期 → 返回原 credentials（透明续期）
  - 已 `Ready` / `Deleting` / `Deleted` → 409 `AlreadyExists`（同 version 不可复活，避免 `<kind>/<repo>@<version>` 引用语义漂移）
  - 已 `Failed` → 在同一 artifact 行上重试：先调用 `Handler.GCBackend` 幂等清理同 storage path 的残留内容，再把 `status` 重置为 `Uploading`、清空 `message` / `digest` / `ready_at`，重新签发凭证；返回时携带 `previous_failure_reason` 便于诊断
- complete 重复调用：
  - 当前 `Uploading` → 正常推进
  - 已 `Ready`：若提交 digest 与已存值一致 → 200；若不一致 → 409 `DigestMismatch`（防止误覆盖）
  - 已 `Failed` / `Deleting` / `Deleted` → 409

**TTL 兜底**：未在 24h 内走到 complete 的 `Uploading` 行由 GC worker 转 `Failed` 并清理后端残留 blob（见 §5.4）。

### 5.3 读路径（Resolve）

```
GET /artifact-repos/{repo}/artifacts/{version}:resolve?usage={inspect|download}[&pin=digest]
```

`usage` 决定凭证形态——集群内消费方与集群外 cli 对凭证的可用性不同，二者必须分开。Operator 只允许 `usage=inspect`，并必须携带租户 / workload namespace 上下文；`axisml-cli pull` 只走 Platform / Gateway 中转的 `usage=download`。

| usage | 调用方 | 返回字段 | 凭证形态 |
| --- | --- | --- | --- |
| `inspect`（默认） | mlservice-operator / mljob-operator 等**集群内**消费方 | `storage_kind` / `uri` / `digest` / `auth_hint` | `auth_hint = {secret_ref: "<resolved-name>", namespace: "<workload-ns>", username_key, password_key}`：仅指针；Secret 必须存在于 workload namespace，消费方按 ref 从 Pod 挂载或 imagePullSecret 中读凭证，Artifacts 不返回明文 |
| `download` | `axisml-cli pull` 等**集群外**调用方 | `storage_kind` / `uri` / `digest` / `pull_credentials` / `expires_at` | OCI：zot 签发 `pull` scope bearer token，TTL=1h；S3：返回 prefix-scoped 临时 STS 凭证（`access_key` / `secret_key` / `session_token` / `prefix`），TTL=1h |

公共字段：

| 字段 | 含义 |
| --- | --- |
| `storage_kind` | `oci` / `s3` |
| `uri` | 由 `Handler.BuildStorageURI` 即时拼装；默认返回可变 tag 形态（OCI: `<repo>:<version>`、S3: `<prefix>/<version>/`），`?pin=digest` 时改为不可变形态——OCI `<repo>@<digest>`，S3 `<prefix>/<version>/`（digest 仅作完整性校验，URI 本身不变；详见 §6.3 决策表） |
| `digest` | 从 PG 读；artifact 未到 `Ready` 时为空 |

**`auth_hint.secret_ref` 的解析规则**：

不再写死字面量 `axisml-tenant-<tenant>-registry`。Artifacts 的 ConfigMap 维护两个租户本地 Secret 模板（默认值由 Helm 注入），resolve 时按 Kind 的存储后端、租户上下文与 workload namespace 渲染。公共制品被租户 workload 消费时也返回租户 namespace 下的复制后 Secret；`axisml-system` 中的公共 Secret 只作为 tenant-operator 的复制源，不直接作为 `auth_hint` 返回给业务 Pod。

```yaml
# Artifacts ConfigMap
authHint:
  oci:
    tenantSecretTemplate: "axisml-tenant-{tenant}-zot-pull"   # zot 拉取凭证
    publicSourceSecretRef:
      namespace: axisml-system
      name: zot-pull
  s3:
    tenantSecretTemplate: "axisml-tenant-{tenant}-rustfs"
    publicSourceSecretRef:
      namespace: axisml-system
      name: rustfs-pull
```

**Secret 由谁创建**：Artifacts 不创建任何 Secret。`axisml-tenant-<tenant>-zot-pull` 等租户级 Secret 由 tenant-operator 按租户 spec 落地——平台默认让 `Tenant.spec.initResources.imagePullSecrets[].name=zot-pull`、`Tenant.spec.initResources.secrets[].name=rustfs`（`tenant-operator §6.3 / §6.4` 的命名规则会拼成 `axisml-tenant-<tenant>-<name>`），来源 Secret 由集群管理员预放在 `axisml-system`。公共 Secret 由 axisml-infra Helm chart 创建并作为复制源。Artifacts 仅按模板返回 workload namespace 内的 `secret_ref`，不校验 Secret 是否存在（消费方拉取失败时上报）。

### 5.4 GC 与孤儿处理

GC worker（leader-only goroutine，每 5 分钟一轮）扫描 PG 四类谓词：

| 谓词 | 动作 |
| --- | --- |
| `artifacts WHERE status='Uploading' AND created_at < now() - 24h` | 标 `Failed`，调 `Handler.GCBackend` 清理后端可能残留的 blob |
| `artifacts WHERE status='Failed' AND updated_at < now() - 30d AND deleted_at IS NULL` | 转 `Deleting`；保留 30 天供诊断、之后清理后端残留并写 `deleted_at` 隐藏展示，但 `(repo_id, version)` 行仍保留且不可复用 |
| `artifacts WHERE status='Deleting'` | 调 `Handler.GCBackend`；成功后 PG `status='Deleted'`、写 `deleted_at` |
| `artifact_repos WHERE status='Deleting'` | 把其下所有非 `Deleted` artifact 推 `Deleting`；全部完成后把 repo 标 `Deleted` 并写 `deleted_at` |

**`GCBackend` 幂等契约**：同一 artifact 在生命周期内可能被调用多次（例如 Uploading TTL 过期已清理后端、用户随后再 DELETE Failed 行）。所有 handler 实现必须满足：
- 后端对象不存在视为成功（吞掉 OCI `MANIFEST_UNKNOWN` / S3 `NoSuchKey` / `404`）
- 重复调用不产生副作用、不返回非 NotFound 错误

**反向孤儿**（后端有 blob 但 PG 无 Ready 行）：仅记日志告警，**不主动清理**。理由：管理员可能手动 push 调试 blob，误删风险高于孤儿空间成本；后续引入对账模式（带审计 + 灰名单）再启用。

### 5.5 副本与 Leader Election

与 compute §5.5 同构：

- **API 层无状态**：所有副本都服务 HTTP，仅写 PG，水平扩容无需协调
- **GC worker 选主**：通过 controller-runtime 的 leader election（`coordination.k8s.io/Lease`）确保单点扫描；replicas=1 时退化为单成员 lease
- **单副本默认**：Standard 与 Lite 默认 `replicas=1`；多副本仅出于可用性
- **readiness 不绑定 leader**：`/readyz` 仅校验 PG 连通，避免 leader 切换期间非 leader 副本被摘流量

## 6. 领域模型

每个模块按 **数据模型 → 模块特有语义** 组织。

### 6.1 编排约定

继承 compute §6.1 全部约定：

- 通用字段 `id uuid` PK / `created_at` / `updated_at` / `deleted_at`
- UNIQUE 默认用 PG partial unique index，`WHERE deleted_at IS NULL`——软删行不占唯一键；但承载持久引用语义的 repo name 与 artifact version 明确使用全生命周期唯一索引，删除后也不释放唯一键
- `name` 字段 **DNS-1123 硬校验**：`[a-z0-9-]`，首尾字母数字，长度 3–40，禁止 `--`。Repo 名进一步约束为 OCI repository 合法字符集（一致即可，不放宽）
- `version` 字段采用 **OCI tag-safe 子集**：允许 `A-Za-z0-9_.-`，长度 1–128，首字符必须为字母数字或 `_`，禁止 `/` 与空值。这样既支持 `1.0.0` / `v1` / `2024-09`，又能安全映射到 OCI tag 与 S3 prefix
- `kind` 枚举值：`model` / `dataset` / `image` / `eval_report`，由 `ArtifactHandler` registry 校验合法性
- 迁移由 `golang-migrate` embedded 启动时执行；多副本并发依赖 `schema_migrations` 表的 PG advisory lock

**表结构遵循 K8s 资源的三段式**——`metadata` / `spec` / `observed (status 簇)`。虽然 Artifacts 没有真正的 CRD，但与 compute 的 `jobs` / `services` 表保持同一种心智模型，便于跨服务读 schema。

### 6.2 ArtifactRepo（仓库 / 组织外壳）

仓库是同一 Kind、同一逻辑名下版本族的容器，与 OCI 的 "repository" 同义。设计上**只承载组织信息**，不持有业务声明字段——所有内容信息都下沉到 artifact 行。

**数据模型**

```
artifact_repos(
  -- metadata
  id                  uuid PK,
  tenant_name         text NULL,                 -- NULL = 平台公共空间；非空时由 TenantResolver 校验存在且 Active，不建跨服务 FK
  kind                text,                       -- 'model' / 'dataset' / 'image' / 'eval_report' / 未来扩展
  name                text,                       -- DNS-1123；与 OCI repo 名一致
  display_name        text,
  description         text,
  labels              jsonb,                      -- 同 K8s metadata.labels
  annotations         jsonb,                      -- 同 K8s metadata.annotations
  owner_user          text,                       -- 来自 X-Axisml-User；公共仓库由 admin 创建

  -- spec：仓库当前无业务声明字段；预留扩展（visibility / mirror policy / 配额 ...）以后再增列，不引入空 jsonb 占位

  -- observed
  status              text,                       -- Active / Deleting / Deleted
  latest_artifact_id  uuid NULL,                  -- 指向 ready_at 最大的 Ready artifact（详见下方语义说明）

  created_at, updated_at, deleted_at
)
-- 命名唯一性：tenant_name IS NULL 与 NOT NULL 各一条 unique index，避免 COALESCE sentinel。
-- 不使用 deleted_at partial unique：repo 名进入 Deleted 后仍不能复用，防止旧引用漂移。
CREATE UNIQUE INDEX artifact_repos_tenant_uniq ON artifact_repos(tenant_name, kind, name)
  WHERE tenant_name IS NOT NULL;
CREATE UNIQUE INDEX artifact_repos_public_uniq ON artifact_repos(kind, name)
  WHERE tenant_name IS NULL;
```

**`latest_artifact_id` 语义**（避免歧义）：
- 定义：当前 repo 下 `status='Ready' AND deleted_at IS NULL` 行中 `ready_at`（complete API 进入 Ready 的时刻，存于 `artifacts.observed`，见 §6.3）最大者；这是"最近 Ready 时间"，**不是 semver 最大版本**
- 维护：complete 推进到 Ready 时，仅当新 row 的 `ready_at` 大于当前 `latest_artifact_id` 指向行的 `ready_at`（或当前为 NULL）才更新；DELETE 当前 latest 时，同事务回填到次新 Ready row（无 Ready row 则置 NULL）
- 不允许靠"无条件覆盖"维护——v2、v1、v3 乱序到达时 latest 必须始终指向时间最新的成功上传，而非最后一次 complete

**多租户语义**：

- `tenant_name` 非空 = 租户私有：仅该租户用户可见、可读、可写
- `tenant_name` 为 `NULL` = **平台公共空间**：所有租户用户只读，写入需 admin
- `(tenant_name, kind, name)` 唯一：同一租户内同 Kind 不能重名；公共空间内独立命名空间
- 租户有效性由 `TenantResolver` 在 API 路径上只读校验：`tenant.name` 存在、`status='Active'`，并返回 workload namespace；Artifacts 不建立跨服务数据库外键，也不写 Compute 表
- 跨租户共享（除"公共"外的细粒度授权）暂不支持，作为后续演进

**生命周期**：

| 操作 | PG 侧 | 触发动作 |
| --- | --- | --- |
| 创建 | insert，`status='Active'` | — |
| 更新 | update（display_name / description / labels / annotations 可改；name / kind 不可改） | — |
| 删除 | `status='Deleting'`（暂不写 `deleted_at`，继续占住名称） | GC worker 级联推动其下所有 artifact `status='Deleting'`；全部完成后 repo 进入 `Deleted` 并写 `deleted_at` |

**引用稳定性约束**：`<kind>/<repo>@<version>` 是跨组件持久引用。Repo 一旦创建，其 `(tenant_name, kind, name)` 不能因删除而复用；Artifact version 一旦进入 `Ready` / `Deleted`，同 repo 下也不能复用。这样旧引用最多返回 410 Gone，不会指向新内容。

**默认仓库**：无。Helm 不预置任何仓库；由用户/admin 按需创建。

### 6.3 Artifact（具体版本化制品）

Artifact 是平台用户实际"引用 / 拉取 / 部署"的单位。承载几乎全部业务信息：版本、Kind 特化的 spec、可观察的状态簇。

**数据模型**

```
artifacts(
  -- metadata
  id            uuid PK,
  repo_id       uuid FK artifact_repos(id),
  version       text,                       -- "v1" / "1.0.0"；repo 内唯一；OCI tag-safe；扮演 metadata.name 角色
  display_name  text,
  description   text,                       -- 此版本说明 / changelog
  labels        jsonb,                      -- 用户自定义标签 {"task":"classification","framework":"pytorch"}
  annotations   jsonb,                      -- 系统/工具注解
  owner_user    text,

  -- spec（用户声明，进入 Ready 后不可变；CRD spec 语义）
  spec          jsonb,                      -- Kind 特化业务字段，详见 §6.3.1 - §6.3.4

  -- observed（status 簇：服务/后端回填，平铺成列以便索引/筛选）
  status        text,                       -- Uploading / Ready / Failed / Deleting / Deleted
  message       text,                       -- 失败原因或 GC 进度
  digest        text,                       -- complete API 校验通过后写入；OCI Kind 用作不可变引用，S3 Kind 仅作完整性校验
  ready_at      timestamptz,                -- 进入 Ready 的时刻；用于 latest_artifact_id 维护（§6.2）

  created_at, updated_at, deleted_at
)
CREATE UNIQUE INDEX artifacts_repo_version_uniq ON artifacts(repo_id, version);
```

**字段分组语义**：

- **metadata**：身份与组织信息，与 spec 无关；`labels` / `annotations` / `display_name` / `description` 在任何状态阶段（包括 Ready 后）均可改
- **spec**：用户声明，**进入 Ready 后冻结**——和 compute `jobs.spec` 不可变同义；想"改" → 同 repo 下新建版本
- **observed (status 簇)**：仅由 Artifacts 服务 / GC worker / 后端校验逻辑回写，API 层不允许直接写入

**存储地址不入表的设计取舍**：`storage_kind` / `uri` / `size_bytes` 都不作为列存储——

| 字段 | 不落库的理由 |
| --- | --- |
| `storage_kind` | 是 `repo.kind` 的纯函数（`ArtifactHandler.StorageKind()`），重复存储会造成漂移风险 |
| `uri` | 由 `ArtifactHandler.BuildStorageURI(scope, repo.name, version)` 即时构造，避免 PG 与命名约定漂移 |
| `size_bytes` | 列表/详情页需要时即时 HEAD 后端；后续若引入配额可再添列 |
| `digest`（OCI Kind） | **必须落库的不可变引用键**——OCI 的不可变引用形式是 `<repo>@<digest>`，没有 digest 就只能回 mutable 的 `:<tag>` 引用，引用方稳定性无从谈起 |
| `digest`（S3 Kind） | **仅作完整性校验**——S3 不支持按内容寻址访问 prefix，URI 始终是 `<prefix>/<version>/`；落库的目的是 cli 下载后比对、以及审计追溯。`?pin=digest` 在 S3 Kind 下不改变 URI，只在响应里把 digest 标为"pinned"提示消费方校验 |

`<scope>` 规则：公共制品 = `system`；租户私有 = `tenants/<tenant_name>`。`<kind-prefix>` ∈ `models / datasets / images / eval-reports`，由 `ArtifactHandler.BuildStorageURI` 拼装，避免不同 Kind 撞名。

**状态机**

```
Uploading ──(complete API + 后端 HEAD 通过)──▶ Ready
          ──(complete 失败 / GC TTL 超时)──▶ Failed

Ready / Failed ──(DELETE)──▶ Deleting ──(Handler.GCBackend 成功)──▶ Deleted
```

- `Uploading`：initiate 后的中间态；24h 内未 complete 由 GC 转 `Failed`
- `Ready`：上传完成、digest 锁定，**spec 不可变，digest 不可变**；可被 resolve、可被 DELETE
- `Failed`：上传失败 / 校验失败；行保留供调试；可被 DELETE 进入清理流程；也可由同 version initiate 在同一行上重试并回到 `Uploading`
- `Deleting` → `Deleted`：与 compute 的删除中间态语义一致；`deleted_at` 在进入 `Deleting` 时写入

> **`Uploading` 替代 `Creating` 的命名取舍**：compute 中 `Creating` 表示"PG 已写、CR 待下发"。Artifacts 没有 CR 下发动作，对应阶段是"PG 已写、bytes 待上传"，命名 `Uploading` 更贴合实际语义；其余状态术语（`Ready` 借自 K8s 通用、`Failed` / `Deleting` / `Deleted` 与 compute 完全一致）。

**跨制品引用**：

格式 `<kind>/<repo>@<version>`，例：

- `model/llama-7b@v3`
- `dataset/mmlu@v1`
- `image/training-base@2024-09`

由 Artifacts 在 initiate API 调用 `Handler.ValidateSpec` 时做存在性检查（懒校验、不级联约束，需要 PG 访问，因而 `ValidateSpec` 必须接受可注入的查询接口，见下）：被引用方进入 `Deleted` 后引用方仍可保留 ref 字符串，resolve 时返回 410 Gone。由于 repo / version 不复用，410 不会变成"意外命中新内容"。

**ArtifactHandler 接口与注册**：

```go
// internal/artifact/handler 包定义

// SpecValidator 是 Handler 校验 spec 时可用的依赖注入；在 API 层构造，注入当前 tenant 上下文
type SpecValidator interface {
    // LookupArtifact 解析 "<kind>/<repo>@<version>" 引用；同租户私有 + 公共空间均可命中
    // 引用方处于公共空间时，被引用方必须也是公共（避免公共制品依赖私有制品）
    LookupArtifact(ctx context.Context, ref string) (*Artifact, error)
}

type ArtifactHandler interface {
    Kind() string                                                // "model" / "dataset" / "image" / "eval_report"
    StorageKind() string                                         // "oci" / "s3"
    BuildStorageURI(scope, repo, version string) string
    InitiateUpload(ctx context.Context, a *Artifact) (*UploadCreds, error)
    VerifyComplete(ctx context.Context, a *Artifact, claim CompleteClaim) error
    // ValidateSpec 必须包含 ctx + Validator，以便对 base_model_ref / dataset_ref 等做存在性 / 可见性校验
    ValidateSpec(ctx context.Context, v SpecValidator, spec json.RawMessage) error
    // GCBackend 必须幂等，且把"对象不存在"视为成功（见 §5.4）
    GCBackend(ctx context.Context, a *Artifact) error
}

// 各 Kind 一个具体 handler，每包 init() 注册到全局 registry
var _ ArtifactHandler = (*ModelHandler)(nil)
var _ ArtifactHandler = (*DatasetHandler)(nil)
var _ ArtifactHandler = (*ImageHandler)(nil)
var _ ArtifactHandler = (*EvalReportHandler)(nil)
```

**注册方式**：编译期注册（每个 handler 包 `init()` → 全局 registry，key=`Kind()`），与 mljob-operator §3 / mlservice-operator §3 的 handler registry 模式同构；不引入运行时插件加载。新增 Kind 不需要改 schema（`spec jsonb` + `kind text` 兼容），只需新增一个 handler 包并扩展 PG 允许列表。

#### 6.3.1 Kind = model

**spec 关键字段**

| 字段 | 必填 | 含义 |
| --- | --- | --- |
| `framework` | ✅ | `pytorch` / `tensorflow` / `onnx` / `safetensors` / `gguf` / `custom` |
| `format` | ✅ | OCI artifact media type / 框架原生格式标签（如 `application/vnd.axisml.model.pytorch.v1+tar`） |
| `task` | 推荐 | `classification` / `embedding` / `llm-completion` / `llm-chat` / `seg` / ... |
| `parameters` | 推荐 | 参数量（数值，便于排序与筛选；如 `7000000000` 表示 7B） |
| `base_model_ref` | 否 | 微调基座引用，如 `model/llama-7b@v1` |
| `training_dataset_ref` | 否 | 训练数据引用，如 `dataset/squad@v1` |
| `entrypoint` | 否 | 推理入口提示（KServe runtime 选型时使用） |

**存储**

- StorageKind: `oci`
- 标准 URI（无 scheme，Distribution 原生形态）: `<oci-host>/<scope>/models/<repo>:<version>`
  - `<oci-host>` 由 ConfigMap 注入，集群内默认 `zot.axisml-infra.svc.cluster.local:5000`
- 不可变引用: `<oci-host>/<scope>/models/<repo>@sha256:<digest>`
- 上传协议: OCI Distribution v2 push（`oras push` / `axisml-cli model push` 封装）
- ManifestType: `application/vnd.oci.image.manifest.v1+json`（OCI 1.1 推荐形态），`artifactType` 字段承载 `spec.format`；不再使用已废弃的 `application/vnd.oci.artifact.manifest.v1+json`
- 协议 prefix 适配：消费方按需补全
  - KServe `storageUri`：补 `oci://` 前缀（KServe v0.13+ 通过 `storage-initializer` 支持 OCI artifacts）
  - docker / nerdctl / oras：直接用无 scheme 形态

**消费**

- mlservice-operator 解析 `MLService.spec.modelRef` → 调 resolve API → 拿到无 scheme URI →
  - `(kserve, inference)` 路径：补 `oci://` 前缀注入 KServe `predictor.storageUri`
  - `(native, *)` 路径：注入容器 env `AXISML_MODEL_URI`（具体 mount 策略见 mlservice-operator §4）

#### 6.3.2 Kind = dataset

**spec 关键字段**

| 字段 | 必填 | 含义 |
| --- | --- | --- |
| `format` | ✅ | `parquet` / `jsonl` / `csv` / `webdataset` / `tfrecord` / `custom` |
| `schema` | 否 | 字段定义（JSON Schema 或 Arrow schema 序列化字符串） |
| `num_records` | 否 | 总样本数 |
| `total_size` | 否 | 总字节数（人类可读字符串如 `"12 GiB"` 或精确 bigint） |
| `license` | 推荐 | SPDX license id（如 `cc-by-4.0` / `mit` / `proprietary`） |
| `splits` | 否 | `{"train": "train/", "val": "val/", "test": "test/"}` 子目录映射 |

**存储**

- StorageKind: `s3`
- URI: `s3://axisml-artifacts/<scope>/datasets/<repo>/<version>/`（一个版本对应一个目录前缀）
- 上传协议: RustFS S3 multipart upload / PutObject，凭证为 prefix-scoped 临时 STS（initiate 阶段签发），不使用单对象 presigned PUT 表达多文件目录上传
- digest 计算: 客户端上传 `artifact-manifest.json`（列出所有对象 path / size / sha256 / contentType 等），对 canonical JSON 做 SHA256；complete 时提交该 digest，服务端读取 manifest 后复算校验

**消费**

- mljob-operator 在容器中注入 env `AXISML_DATASET_URI=s3://...`、`AXISML_DATASET_DIGEST=sha256:...`，并通过 init container 或 csi-s3 volume 把数据挂载到 `/data`（具体策略见 mljob-operator）

#### 6.3.3 Kind = image

**spec 关键字段**

| 字段 | 必填 | 含义 |
| --- | --- | --- |
| `purpose` | ✅ | `training` / `inference` / `dev` |
| `base_image` | 否 | 基础镜像引用（如 `pytorch/pytorch:2.1-cuda12`） |
| `platforms` | 推荐 | `["linux/amd64", "linux/arm64"]` 等架构列表（多架构 manifest list） |
| `entrypoint` | 否 | 默认 entrypoint 提示，便于平台 UI 展示 |
| `notes` | 否 | 工具链 / CUDA / cuDNN 版本说明 |

**存储**

- StorageKind: `oci`
- 标准 URI（与 model Kind 同构，host 由 ConfigMap 注入）: `<oci-host>/<scope>/images/<repo>:<version>`
- 上传协议: docker push / nerdctl push / podman push 完成阶段 2 的实际推送，但**两阶段流程仍由 cli 显式调用**：
  1. `axisml-cli image initiate <repo> <version> --spec ...` → 拿 push scope token + URI
  2. `docker login <oci-host>` 用 token，再 `docker push <oci-host>/<scope>/images/<repo>:<version>`
  3. `axisml-cli image complete <repo> <version> --digest <push 输出 digest>`
  - cli 子命令 `axisml-cli image push` 把上述三步封装成单命令，直接复用本机 docker daemon

**消费**

- mljob-operator / mlservice-operator 用 `<oci-host>/<scope>/images/<repo>:<version>` 或 `...@<digest>` 作为 Pod `spec.containers[].image`
- 镜像拉取凭证按 §5.3 `auth_hint` 规则从 ConfigMap 模板渲染（默认 `axisml-tenant-<tenant>-zot-pull`）；Secret 由 tenant-operator 按租户 spec 落地，详见 §5.3 / §8

#### 6.3.4 Kind = eval_report

**spec 关键字段**

| 字段 | 必填 | 含义 |
| --- | --- | --- |
| `model_ref` | ✅ | 被评测模型引用，如 `model/llama-7b@v3` |
| `dataset_ref` | ✅ | 评测数据集引用，如 `dataset/mmlu@v1` |
| `metrics` | ✅ | `{"accuracy": 0.78, "f1": 0.81, ...}` 关键指标抽取到 spec，便于检索与排序 |
| `evaluator` | ✅ | 评测器标识，如 `lm-eval-harness@v0.4.0` / `internal-pipeline@v2` |
| `notes` | 否 | 配置说明、随机种子、超参等 |

**存储**

- StorageKind: `s3`
- URI: `s3://axisml-artifacts/<scope>/eval-reports/<repo>/<version>/`
- 内容: 完整报告文件（HTML / JSON / 图表）作为对象存到目录前缀；关键指标重复抽取到 spec.metrics 用于平台筛选
- digest 计算: 同 dataset，必须上传 `artifact-manifest.json` 并以 manifest canonical SHA256 作为 `artifacts.digest`

**消费**

- 平台 UI 列表页直接读 spec.metrics 排序展示，详情页通过 resolve 获取 URI 跳转下载报告原文

## 7. API 设计

### 7.1 路径规划

Artifacts 所有 API 置于 `/api/v1` 前缀下。外部用户流量只到 Platform / Gateway，`axisml-cli` 不直连集群内 Artifacts；集群内 Operators 可直连 Artifacts Service，但只用于 `usage=inspect` 解析引用。

| 资源组 | 路径 | 主要动作 |
| --- | --- | --- |
| 健康检查 | `/healthz`、`/readyz` | Liveness / Readiness |
| 租户私有仓库 | `/api/v1/tenants/{tenant}/artifact-repos` | Create / Get / List / Update / Delete |
| 公共仓库 | `/api/v1/public/artifact-repos` | Get / List（普通用户）；Create / Update / Delete（admin） |
| 仓库下版本 | `/api/v1/.../artifact-repos/{repo}/artifacts` | List（含按 status / labels 筛选）；GET 单个 |
| 注册版本 | `/api/v1/.../artifact-repos/{repo}/artifacts:initiate` | POST：注册元数据 + 签发上传凭证 |
| 完成上传 | `/api/v1/.../artifact-repos/{repo}/artifacts/{version}:complete` | POST：校验 digest + 转 Ready |
| 解析引用 | `/api/v1/.../artifact-repos/{repo}/artifacts/{version}:resolve` | GET：返回 uri / digest / auth_hint |
| 删除版本 | `/api/v1/.../artifact-repos/{repo}/artifacts/{version}` | DELETE |
| 删除仓库 | `/api/v1/.../artifact-repos/{repo}` | DELETE（级联） |

`...` 表示 `/tenants/{tenant}` 或 `/public` 二选一。

### 7.2 身份上下文

由 Platform / Gateway 注入的请求头：

| Header | 含义 |
| --- | --- |
| `X-Axisml-User` | 调用方用户唯一 ID，用于审计与 ownership 归属 |
| `X-Axisml-Roles` | 逗号分隔的角色列表（如 `user` / `admin`），由 Platform 鉴权后注入；信任边界与 `X-Axisml-User` 一致 |

租户归属通过 URL 路径 `/tenants/{tenant}/...` 表达。Artifacts 不重做角色鉴权——鉴权由 Platform 统一完成；Artifacts 仅校验：
- 路径中的租户存在且激活（通过 `TenantResolver` 只读解析，当前实现共库直读 tenants 表，详见 §8）
- 相关资源归属于该租户
- 公共空间的写动作要求 `X-Axisml-Roles` 包含 `admin`

> **不再维护静态 admin 名单**：之前版本曾把 admin 名单放在 ConfigMap，与 Platform 的用户 / 角色系统会漂移；统一改成由 Platform 在每次请求中注入角色 header，单一事实来源在 Platform。

Operator 直连 Artifacts 时不代表终端用户身份，只携带 controller service identity 与明确的 tenant / workload namespace 参数；Artifacts 只允许这类调用访问 `resolve?usage=inspect`，并按租户可见性与 workload namespace 返回 `auth_hint`。

### 7.3 契约管理

`components/artifacts/api/openapi.yaml` 是唯一契约源，使用 `oapi-codegen` 生成：

- Artifacts 侧：Go types + server stub（`api/types/`）
- Platform / Operators 侧：Go client SDK（`pkg/artifacts-client/`，通过 Makefile target 生成）
- `axisml-cli` 侧：通过 Platform / Gateway 暴露的 API 使用同一契约；不持有集群内 Artifacts Service 地址

### 7.4 cli 协作时序

`axisml-cli` 的 push 子命令以 `model push` 为例；图中 cli 到 Artifacts 的控制面调用均经 Platform / Gateway 中转，省略中转层以突出 Artifacts 与存储后端的两阶段关系：

```
cli                  Artifacts                zot                       PG
 │                     │                       │                         │
 │ push --kind model …│                       │                         │
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

S3 路径（dataset / eval_report）类似，把 `oras push` 替换为 S3 multipart upload / PutObject，并在 complete 前上传 `artifact-manifest.json`；Artifacts 读取 manifest 后复算 canonical SHA256，不用对象 ETag 表示目录级 digest。

## 8. 外部系统协作

| 对象 | 交互方式 | 关注点 |
| --- | --- | --- |
| **zot**（新增 infra 组件） | OCI Distribution v2 协议；Artifacts 持 admin 凭证用于校验 / GC / 签发 scope token，cli 持短期 push / pull token 直连 | 由 axisml-infra chart 提供（infra.md 待同步追加 zot 章节）；当前文档约定接入契约：admin 凭证存于平台级 Secret，endpoint 由 Artifacts ConfigMap 注入 |
| **RustFS**（已有） | S3 协议；Artifacts 服务签发 prefix-scoped 临时 STS 凭证；bucket `axisml-artifacts` 由 axisml-infra chart 创建 | 区分 prefix `<scope>/datasets/...` / `<scope>/eval-reports/...`；同 bucket 不混 OCI；目录级完整性由 `artifact-manifest.json` + canonical SHA256 表达 |
| **PostgreSQL** | GORM；与其他服务共用 database `axisml`；表前缀 `artifact_*`（`artifact_repos` / `artifacts`） | 迁移随二进制打包，启动时执行（golang-migrate） |
| **ml-compute（tenants 表）** | Artifacts 通过 `TenantResolver` 把 URL 中的 tenant name 解析为租户 namespace，并校验 `status='Active'`。**当前方案：跨服务直读 PG `tenants` 表**（与 compute 同库，只读 SELECT 权限）；若后续两服务拆库则改为 compute 暴露 `GET /api/v1/tenants/{name}` internal API | 仅依赖 `tenants` 这一张表的 `name / namespace / status`，不触碰 compute 私有列，不建立 FK，不写 Compute 表；compute schema 变更需在 PR 描述中提示 Artifacts 同步 |
| **tenant-operator** | 不创建 Artifacts 专用 Secret；约定平台默认让 `Tenant.spec.initResources.imagePullSecrets[].name='zot-pull'`、`secrets[].name='rustfs'`，由 tenant-operator 按 `axisml-tenant-<tenant>-<name>` 命名规则落地（[tenant-operator §6.3 / §6.4](operators/tenant-operator.md)）；来源 Secret 由集群管理员预放在 `axisml-system` | resolve API 的 `auth_hint.secret_ref` 始终指向 workload namespace 内的租户本地 Secret；公共 Secret 只作为复制源，Secret 的 schema（`username` / `password` / `server` 或 dockerconfigjson）由消费方按 K8s Secret type 判断 |
| **mlservice-operator** | 通过 Artifacts client SDK 解析 `spec.modelRef` → KServe `storageUri` / `AXISML_MODEL_URI` env | 见 [operators/mlservice-operator.md §4](operators/mlservice-operator.md) `spec.modelRef` 字段语义 |
| **mljob-operator** | 通过 Artifacts client SDK 解析 `spec.imageRef`（如适用）→ Pod `image`；解析 `spec.datasetRef` → 容器 env / volume | 详见 mljob-operator 对应字段 |
| **Platform / Gateway** | REST 调用 + UI 展示（仓库列表、版本详情、上传向导）；同时是 `X-Axisml-User` / `X-Axisml-Roles` 的注入方 | 唯一外部 API 调用方（终端用户与 `axisml-cli` 均经 Platform / Gateway 中转） |

## 9. 部署形态

### 9.1 镜像与容器

- 镜像：`ghcr.io/axisml/axisml-artifacts:<appVersion>`（由 `build/docker/artifacts.Dockerfile` 构建）
- 端口：`8082/tcp`（REST）
- 启动命令：`/artifacts serve`
- 探针：`GET /healthz`、`GET /readyz`

### 9.2 Helm 模板

补充 `deploy/helm/axisml-system/templates/artifacts/` 下模板：

| 文件 | 用途 |
| --- | --- |
| `configmap.yaml`（已存在，补字段） | DB 连接、zot endpoint、RustFS endpoint、`authHint` 模板（§5.3）、token TTL；不含 admin 名单（admin 走 `X-Axisml-Roles` header） |
| `deployment.yaml`（已存在） | 使用 `axisml-artifacts` 镜像，加探针 |
| `service.yaml`（已存在） | 保持 ClusterIP 8082 |
| `serviceaccount.yaml`（新增） | Artifacts 服务账号 |
| `secret.yaml`（新增） | 平台级 zot admin 凭证、RustFS 凭证（不进 ConfigMap） |
| `servicemonitor.yaml`（新增） | `/metrics` 暴露，kube-prometheus-stack 自动发现 |

### 9.3 对外暴露

Artifacts 不直接暴露到集群外：终端用户上传/下载经 `axisml-cli -> Platform/Gateway -> Artifacts` 取得 `uri` 与临时凭证后，cli 再直连 zot/RustFS；存储后端外部入口由 axisml-infra 的 Gateway 暴露。

### 9.4 引导数据

Helm `post-install` Job 仅做：

- 数据库 migration 启动检查（其余 schema 由服务进程 embedded 迁移完成）
- 不预置任何 ArtifactRepo（公共空间或租户私有都靠用户主动创建）

### 9.5 副本与可用性

- **默认形态**：`replicas=1`（Standard 与 Lite 均同）；`strategy: RollingUpdate`
- **多副本**：修改 Helm values 横扩；API 层无状态；GC worker 走 leader election
- **探针**：`/healthz` 进程存活；`/readyz` 校验 PG 连通（GC 就绪不计入 readiness）
- **Leader 身份暴露**：`/metrics` 暴露 `axisml_artifacts_is_leader` gauge（0/1）

### 9.6 关键 metrics

`servicemonitor.yaml` 暴露 `/metrics`（Prometheus 格式）。至少包含以下业务指标（除标准 Go runtime / HTTP / controller-runtime 指标外）：

| 指标 | 类型 | 含义 | 告警建议 |
| --- | --- | --- | --- |
| `axisml_artifacts_is_leader` | gauge | 当前副本是否为 leader（0/1） | 多副本时 `sum == 1` 否则异常 |
| `axisml_artifacts_uploading_count{kind}` | gauge | 当前 `status='Uploading'` 行数，按 Kind 拆分 | 持续高于阈值 → cli 上传失败率上升 |
| `axisml_artifacts_gc_actions_total{predicate,result}` | counter | GC 动作计数（`uploading_ttl` / `deleting_cascade` / `deleted_completion`，成功 / 失败） | 失败率突增 → 后端不可达 |
| `axisml_artifacts_orphan_blobs_total{backend}` | counter | 反向孤儿告警次数（PG 无对应行的后端 blob） | 非零即记录到运维台账 |
| `axisml_artifacts_resolve_requests_total{kind,result}` | counter | resolve API 请求计数（成功 / 410 Gone / 4xx / 5xx） | 410 突增 → 上游引用了已删除版本 |
| `axisml_artifacts_initiate_duration_seconds{kind}` | histogram | initiate 阶段端到端耗时（含 Handler 签发凭证） | P99 阈值告警 |
| `axisml_artifacts_complete_duration_seconds{kind}` | histogram | complete 阶段耗时（含后端 HEAD 校验） | P99 阈值告警 |
| `axisml_artifacts_api_request_duration_seconds{route,status}` | histogram | API 请求延迟分布 | 常规 SLO 告警 |

## 10. 关键设计决策

| 决策项 | 决策 | 理由 |
| --- | --- | --- |
| 服务定位 | 元数据服务，**bytes 不经过 Artifacts** | 模型 / 数据集动辄数十 GB，代理流量在单副本控制平面是反模式；OCI / S3 协议本身已成熟，cli 直传后端最简 |
| 制品分级 | `artifact_repos`（仓库外壳）+ `artifacts`（具体版本）；绝大部分信息在 artifact 行 | 与 OCI repository / tag 模型对齐；repo 仅做组织，避免双层存储字段冗余 |
| Kind 扩展 | 编译期 ArtifactHandler registry，Model / Dataset / Image / EvalReport 内置；新增 Kind 不改 DB 表结构，但要更新 handler / OpenAPI / 允许列表 | 与 mljob-operator / mlservice-operator 的 handler 注册模式同构；`spec jsonb` 提供天然扩展点，同时保持 API 契约显式 |
| 多租户 | `tenant_name` 可空（NULL = 平台公共空间）；公共空间只读 + admin 写；租户存在性由只读 `TenantResolver` 校验 | 兼顾"租户隔离"与"基础镜像 / 官方模型共享"两类常见场景；避免跨服务 FK，把 Compute 仅作为租户状态来源；细粒度 ACL 作为后续演进 |
| OCI Registry 选型 | 内置 zot | OCI 原生，对 ML 模型 artifact 类型支持完整；轻量、单二进制、可对接 S3；与 ML 模型托管场景天然契合 |
| 写路径 | Two-Phase Register-Upload：initiate（PG + 签凭证）→ cli 直传 → complete（校验 + Ready） | 与 S3 multipart upload 的 init / complete 同构；同步两段提交无需常驻 reconciler，仅留 GC TTL 兜底未完成的 Uploading；后端临时错误不把行推 Failed |
| 读路径凭证模型 | resolve 区分 `inspect`（返回 K8s `secret_ref` 给集群内 operator）与 `download`（返回短期 token / 临时 STS 给集群外 cli） | 集群内外消费方对凭证可用性不同：operator 只能读 K8s Secret，cli 只能拿可直连的临时凭证；混在一起会破坏其中一边 |
| 表 schema 模型 | CRD 风格三段式：metadata / spec（不可变）/ observed | 与 compute jobs / services 表保持同一种心智模型；spec jsonb 容纳 Kind 特化字段 |
| 存储字段策略 | `storage_kind` / `uri` / `size_bytes` 不入表（Handler 即时推导）；`digest` 入表，OCI Kind 用作不可变引用键、S3 Kind 仅作 manifest 完整性校验 | 减少 PG 与命名约定的漂移面；digest 是 OCI 不可变引用的必需字段，S3 仍需要 digest 但不构成 URI |
| 状态机命名 | `Uploading` 替代 compute 的 `Creating`；`Ready` / `Failed` / `Deleting` / `Deleted` 与 compute 一致 | 命名贴合实际语义；Artifacts 没有 CR 下发动作，"创建中" 不准确 |
| Ready 后不可变 | spec / digest 进入 Ready 后冻结；改内容 = 新版本；complete 重入须 digest 一致，否则 409 | 引用方（`modelRef` / `imageRef` / `datasetRef`）的稳定性是平台级合约 |
| Repo / version 不复用 | Repo 删除进入 `Deleting` 并占住名称；Ready / Deleted version 永不复用，旧引用最多 410 Gone | `<kind>/<repo>@<version>` 是跨组件持久引用，复用会让旧引用命中新内容 |
| Failed 重试 | 同 version 的 Failed 行允许 initiate 复用同一行：先清理同路径残留，再回到 Uploading，避免强制用户改 version | 多数 Failed 来自可修复问题；同一行重试不会释放唯一键，也不会和 GC 产生新旧行路径竞争 |
| 跨制品引用格式 | `<kind>/<repo>@<version>` 字符串；`Handler.ValidateSpec(ctx, deps, spec)` 通过注入的 `LookupArtifact` 做懒存在性校验 | 文本可读、跨进程可序列化；引用对象删除后引用方仍可保留字符串以利审计 |
| Secret 命名契约 | Artifacts 不创建 Secret；`auth_hint.secret_ref` 始终渲染为 workload namespace 内的租户本地 Secret，公共 Secret 只作为 tenant-operator 复制源 | Pod 不能直接依赖 `axisml-system` Secret；单一事实来源在 tenant-operator，避免 Artifacts 与 tenant-operator 分别维护命名 |
| Admin 鉴权 | 读 `X-Axisml-Roles` header（由 Platform 注入），不维护静态 admin 名单 | Platform 是用户 / 角色权威；静态名单会与 Platform 漂移 |
| 反向孤儿处理 | 仅告警，不主动清理 | 误删风险高于孤儿空间成本；后续引入对账模式 + 灰名单 |
| API 协议 | REST / JSON + OpenAPI 3.0 | 与 compute 一致；可生成 Platform / Operators / cli 共用的 Go SDK |
| HA / 副本 | 默认 `replicas=1`；API 层无状态可横扩；GC worker 通过 K8s Lease 选主 | 与 compute 同模式；零成本保留横扩能力 |

## 11. 未来规划

- **细粒度 ACL**：仓库级 / 版本级访问策略（除"租户私有 / 公共"二元外的中间态：跨租户共享、外部 SaaS 模型订阅）
- **制品签名与 SBOM**：cosign / notation 集成，supply-chain 审计
- **配额管理**：按租户 / 按 Kind 限制总大小、总数量、单版本大小；落 `size_bytes` 入表，GC worker 维护用量
- **跨集群同步**：多 region / 多集群部署时的镜像 / 模型自动同步（zot replication 原生支持）
- **在线 lineage**：`base_model_ref` / `training_dataset_ref` / `eval_report.model_ref` 的依赖图可视化与影响面分析
- **漏洞扫描集成**：image Kind 接入 trivy / grype，扫描结果作为 annotations 回写
- **预留 spec 列**：当 ArtifactRepo 引入 `visibility` / `mirror policy` / `quota` 等业务字段时，再加 `spec jsonb` 列；当前不引入空列
