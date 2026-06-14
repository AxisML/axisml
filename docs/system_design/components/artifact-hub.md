# AxisML Artifact Hub 概要设计

## 1. 定位与边界

平台的制品元数据服务；以 PostgreSQL `artifacts` 表为元数据权威，按 `(namespace, kind, name, version)` 四元组寻址，bytes 由调用方凭短期凭证直连 zot / RustFS。

| 做 | 不做 |
| --- | --- |
| Artifact CRUD、两阶段写（initiate / complete）、resolve | 制品 bytes 的存取（→ zot / RustFS 直连） |
| 上传 / 下载凭证签发（OCI scope token / S3 prefix-scoped STS） | 用户认证与角色鉴权（→ [auth.md](../auth.md)） |
| Kind 按 Handler 注册表分发（model / dataset / image） | tenant 存在性与权限校验（namespace 字段由 [compute-service.md](compute-service.md) 兜底 tenant 语义） |
| GC：Uploading TTL、Failed 留存、Deleting 推进 | 反向孤儿主动清理（仅告警）；跨 namespace 级联删除 |
| `visibility=public` 全局可见制品（落 `axisml-system` 内置租户） | tenant Secret 落地（→ [tenant-operator.md](tenant-operator.md)） |

## 2. 架构

### 2.1 上下文

```
  ┌──────────────┐  REST   ┌─────────────────┐
  │  Platform    │ ────────│   Operators     │
  └──────┬───────┘         │ (compute-op)    │
         │                 └────────┬────────┘
         ▼                          │ resolve?usage=inspect
  ┌────────────────────────────┐    │
  │    AxisML Artifact Hub        │◀───┘
  └──────┬───────────────┬─────┘
         │ PG 读写        │ 签 token / HEAD / GC
         ▼               ▼
┌─────────────────┐  ┌─────────────────────┐
│  PostgreSQL     │  │  zot (OCI)          │
│  artifacts 表   │  │  RustFS (S3)        │
└─────────────────┘  └────────▲────────────┘
                              │ 直传 / 直拉（短期凭证）
                              │
                  ┌───────────┴─────────┐
                  │ 客户端 / workload    │
                  └─────────────────────┘
```

### 2.2 内部结构

```
┌──────────────────────── Artifacts (Go) ─────────────────────────┐
│  HTTP API (Gin)                                                 │
│   ├── initiate / complete / resolve / GET / DELETE              │
│   └── middleware: 身份解析、错误、metrics                       │
│                                                                 │
│  ArtifactHandler Registry (compile-time init())                 │
│   └── handlers/{model, dataset, image}                          │
│         BuildStorageURI / ValidateSpec / InitiateUpload         │
│         VerifyComplete / GCBackend                              │
│                                                                 │
│  GC Worker (leader-only goroutine, 5 min tick)                  │
└─────────────────────────────────────────────────────────────────┘
```

## 3. 核心模型

| 实体 | 含义 | 寻址键 | 备注 |
| --- | --- | --- | --- |
| Artifact | 版本化制品 | `(namespace, kind, name, version)` | 四元组创建后不复用；spec / digest 进入 `Ready` 后冻结 |

- `kind` 枚举：`model` / `dataset` / `image`，由 Handler registry 校验。
- `namespace` 是 tenant 标识符（= compute `tenants.name`），由 Platform 透传；Artifacts 不解析、不做存在性校验，仅作为不透明分区键使用。`axisml-system` 是平台内置 tenant，承载 `visibility=public` 全局可见制品。
- `visibility` 枚举：`tenant`（默认，仅本 namespace 内可见）/ `public`（全局可见；仅允许在 `axisml-system` 内置 namespace 下创建，由调用方 Platform 做 RBAC 兜底）。
- 状态机集合：`Uploading` / `Ready` / `Failed` / `Deleting` / `Deleted`（详见 [§6](#6-接口契约)）。
- 扩展元数据 `labels` / `annotations` 双字段语义对齐 [database.md §1.6](../database.md#16-扩展元数据-labels--annotations)；artifacts 无 CR，扩展位天然只落 PG。

字段级 schema 见 [database.md §3.1](../database.md#31-artifacts-表)；spec 子字段见 [apis/artifact-hub.yaml](../apis/artifact-hub.yaml)。

## 4. 核心功能

写路径统一见 [§5.1](#51-写路径两阶段提交)；读路径统一见 [§5.2](#52-读路径resolve)。完整 spec 字段以 [apis/artifact-hub.yaml](../apis/artifact-hub.yaml) 为准。

### 4.1 Model

OCI artifact，承载训练好的模型权重。

| 项 | 值 |
| --- | --- |
| StorageKind | `oci`（zot） |
| 必填 spec | `framework`（`pytorch` / `tensorflow` / `onnx` / `safetensors` / `gguf` / `custom`）、`format`（OCI artifact media type） |
| URI 模板 | `<oci-host>/namespaces/<ns>/models/<name>:<version>`；ManifestType `application/vnd.oci.image.manifest.v1+json`，`artifactType` 承载 `spec.format` |
| 主要消费方 | mlservice handler → KServe `predictor.storageUri`（补 `oci://`）或 native env `AXISML_MODEL_URI` |

### 4.2 Dataset

S3 目录级制品，承载训练 / 评测数据集。

| 项 | 值 |
| --- | --- |
| StorageKind | `s3`（RustFS） |
| 必填 spec | `format`（`parquet` / `jsonl` / `csv` / `webdataset` / `tfrecord` / `custom`） |
| URI 模板 | `s3://axisml-artifact-hub/namespaces/<ns>/datasets/<name>/<version>/`；digest = canonical JSON SHA256(`artifact-manifest.json`) |
| 主要消费方 | mlrun handler 注入 env `AXISML_DATASET_URI` / `AXISML_DATASET_DIGEST`，并通过 init container 或 csi-s3 volume 挂载到 `/data` |

### 4.3 Image

OCI 容器镜像，承载训练 / 推理 / 开发运行时；阶段 2 由调用方本机 docker / nerdctl 直接 push 到 zot endpoint。

| 项 | 值 |
| --- | --- |
| StorageKind | `oci`（zot） |
| 必填 spec | `purpose`（`training` / `inference` / `dev`） |
| URI 模板 | `<oci-host>/namespaces/<ns>/images/<name>:<version>` |
| 主要消费方 | mlrun / mlservice handler 用 URI 作为 Pod `spec.containers[].image`；imagePullSecret 由 tenant-operator 落地的 per-tenant ServiceAccount 默认携带，operator 不显式拼 secret 名 |

## 5. 关键机制

### 5.1 写路径：两阶段提交

```
客户端            Artifacts               zot / RustFS         PG
 │── initiate ──▶│                            │                 │
 │                │── ValidateSpec ──────────▶│                 │
 │                │── insert(Uploading) ────────────────────────▶│
 │                │── sign scope token / STS  │                 │
 │◀── creds,uri ──│                            │                 │
 │── push / PutObject ─────────────────────▶│                 │
 │◀── digest ──────────────────────────────│                 │
 │── complete(digest) ─▶│                    │                 │
 │                │── HEAD manifest / GET artifact-manifest.json│
 │                │◀── 200 + digest ─────────│                 │
 │                │── update(Ready, digest) ────────────────────▶│
 │◀── 200 ────────│                            │                 │
```

**幂等性**：

- 重复 initiate 同 `(namespace, kind, name, version)`：未过期 `Uploading` 返原凭证；其他终态均 409（同 version 不可复活，复用先 DELETE 旧行）。
- 重复 complete：`Uploading` 正常推进；`Ready` 且 digest 一致 → 200，不一致 → 409 `DigestMismatch`；其他状态 → 409。
- 未在 24h 内 complete 的 `Uploading` 由 GC 转 `Failed` 并清理后端残留 blob。

### 5.2 读路径：resolve

`GET /api/v1/namespaces/{ns}/{kindPlural}/{name}/{version}/resolve?usage={inspect|download}`

| usage | 调用方 | 额外字段 | 凭证形态 |
| --- | --- | --- | --- |
| `inspect` | 集群内 operator（mlservice / mlrun handler） | — | 不签发任何凭证；operator 派生的 Pod 通过 per-tenant ServiceAccount（由 tenant-operator 在 workload namespace 内落地，已默认携带 zot / RustFS 的 imagePullSecrets / 通用 Secret）拉取/读写后端；Artifacts 只回 `uri` + `digest`。 |
| `download` | 终端用户 / 训练 / 推理脚本（经 Platform / Gateway） | `pull_credentials` / `expires_at` | OCI pull scope token / S3 prefix-scoped STS，TTL=1h |

公共字段：`storage_kind`（`oci` / `s3`）、`uri`（由 `Handler.BuildStorageURI` 拼装）、`digest`（PG 读，未 Ready 为空）、`visibility`。

### 5.3 GC 与生命周期清理

GC worker（leader-only，每 5 分钟一轮）扫描 PG 三类谓词：

| 谓词 | 动作 |
| --- | --- |
| `status='Uploading' AND created_at < now() - 24h` | 标 `Failed`，**同步**调 `Handler.GCBackend` 立刻清理后端残留 blob |
| `status='Failed' AND updated_at < now() - 30d AND deleted_at IS NULL` | 转 `Deleting`；后端 blob 已在转 Failed 时清空，PG 行保留 30 天供诊断 |
| `status='Deleting'` | 调 `Handler.GCBackend`；成功后 PG `status='Deleted'`、写 `deleted_at` |

`Handler.GCBackend` 实现需保证幂等，并把"对象不存在"视作成功（吞掉 OCI `MANIFEST_UNKNOWN` / S3 `NoSuchKey` / 404）。反向孤儿（后端有 blob 但 PG 无 Ready 行）仅记日志告警，不主动清理。整 namespace 批量删除由调用方按 list 逐条 DELETE 触发；Artifacts 不提供 namespace 级联删除端点。

### 5.4 元数据 / 字节分离权威

| 类别 | 权威方 | 流向 |
| --- | --- | --- |
| 元数据（artifact 行） | PG | API → PG |
| 制品 bytes | zot / RustFS | 客户端 / workload → 后端（直传 / 直拉，不经服务） |
| digest | 后端 | 后端 → complete API → PG |
| 上传 / 下载凭证 | Artifacts 签发 | Artifacts → Platform → 客户端（短期 token，TTL=1h）；集群内 operator 走 inspect，不签明文 |

## 6. 接口契约

| 类别 | 内容 | 引用 |
| --- | --- | --- |
| 对外 REST | `/api/v1/namespaces/{ns}/{kindPlural}/{name}[/{version}[/{complete,resolve}]]`；版本级 `GET` / `PATCH` / `DELETE` 同前缀。`kindPlural` 为 `ArtifactKind` 的 URL 复数形式（`model`↔`models`、`dataset`↔`datasets`、`image`↔`images`） | [apis/artifact-hub.yaml](../apis/artifact-hub.yaml) `Artifacts` tag |
| Handler 接口 | 见下表 | — |
| 身份头 | 调用方注入 `X-Axisml-User`，本服务仅做审计 | [auth.md §7](../auth.md#7-下游身份透传) |
| 列表查询 | list 端点支持 `?labelSelector=` K8s grammar，可按 Platform 注入的 `axisml.io/project` 等 label 过滤 | [database.md §1.6](../database.md#16-扩展元数据-labels--annotations) |
| 错误格式 | HTTP 标准状态码 + RFC 7807 problem+json | — |
| 写后语义 | initiate 在 PG 提交后返回上传凭证；Ready 由 complete 推进，调用方通过 GET 观察 status；PATCH 是纯 PG mutation，立即可读 | — |

**PATCH 可变字段**（任何非终态状态生效）：`displayName` / `description` / `labels` / `annotations` 四项。其它字段一律不可变（含 `visibility`：创建后不可变）；submitting any other field returns `400 ImmutableField`。`Deleting` / `Deleted` 行 PATCH 返 `409 ArtifactTerminal`。`labels` / `annotations` 按整体 map 替换语义（无 per-entry 合并）；缺省 key 保持原值。详见 [apis/artifact-hub.yaml `updateArtifact`](../apis/artifact-hub.yaml)。

**ArtifactHandler 接口**（编译期注册，key=`Kind()`）：

| 方法 | 责任 |
| --- | --- |
| `Kind()` | 返回 `model` / `dataset` / `image`；registry 主键 |
| `StorageKind()` | 返回 `oci` / `s3` |
| `BuildStorageURI(ns, name, version)` | 即时拼装存储 URI；不读 PG / 不调后端 |
| `ValidateSpec(ctx, spec)` | 校验 Kind 特化 spec 字段；纯函数 |
| `InitiateUpload(ctx, a)` | 签发上传凭证（OCI scope token / S3 prefix-scoped STS）；幂等 |
| `VerifyComplete(ctx, a, claim)` | 调后端 HEAD / GET manifest 校验 digest |
| `GCBackend(ctx, a)` | 删除后端残留 blob；幂等；NotFound 视作成功 |

**状态机**：

```
Uploading ──(complete + 后端 HEAD 通过)──▶ Ready
          ──(complete 失败 / GC TTL 超时)──▶ Failed
Ready / Failed ──(DELETE)──▶ Deleting ──(GCBackend 成功)──▶ Deleted
```

| 状态 | 含义 |
| --- | --- |
| `Uploading` | initiate 后中间态；24h 内未 complete 由 GC 转 `Failed` |
| `Ready` | 已上传 + digest 锁定；spec / digest 不可变；可 resolve / DELETE |
| `Failed` | 上传或校验失败；保留供调试；可 DELETE |
| `Deleting` → `Deleted` | 删除中间态；进入 `Deleting` 时写 `deleted_at` |

未注册 Kind 兜底：API 收到 `kind` 不在 registry 的请求 → 400，不创建 PG 行。

## 7. 依赖

| 依赖 | 用途 | 引用 |
| --- | --- | --- |
| PostgreSQL | 元数据权威；与 compute 共享 database，表前缀 `artifact_*` | [database.md](../database.md) / [infra.md](../infra.md) |
| zot | OCI 后端；Artifacts 持 admin 凭证签 scope token / HEAD 校验 / GC 删 blob，客户端持短期 token 直连 | [infra.md](../infra.md) |
| RustFS | S3 后端；Artifacts 签 prefix-scoped 临时 STS / HEAD `artifact-manifest.json` 校验 / GC 删 prefix，bucket `axisml-artifact-hub` | [infra.md](../infra.md) |
| tenant-operator | Operator-side `resolve?usage=inspect` 路径依赖 tenant-operator 在 workload namespace 落地的 per-tenant ServiceAccount + Secrets（默认 imagePullSecret 拉 zot、env / volume 形态读 RustFS）；Artifacts 本身不参与 Secret 落地，也不在 resolve 返回任何 secret 名引用 | [tenant-operator.md](tenant-operator.md) |

## 8. 运行时形态

| 维度 | 值 |
| --- | --- |
| 进程 | 单二进制 `axisml-artifact-hub`；启动时执行 golang-migrate embedded migration |
| 副本 | API 默认 `replicas=1`（无状态，可水平扩）；GC worker 单 leader（K8s Lease） |
| 暴露端口 | API `:8080`；Metrics `:8081`；Probes `:8082`（`/healthz` / `/readyz`，仅校验 PG 连通），均不对外 |
| RBAC scope | 自身 ns 的 `leases`（GC leader election）；不需要其他 K8s RBAC（只读 PG + 调后端 HTTP） |
| Helm values / 镜像 | 详见 [deployment.md](../deployment.md) |

## 9. 相关引用

- [overview.md](../overview.md) — Artifacts 在控制平面里的位置
- [auth.md](../auth.md) — `X-Axisml-User` 注入与传播
- [database.md](../database.md) — `artifacts` 表 schema
- [deployment.md](../deployment.md) — Helm chart 与部署形态
- [monitoring.md](../monitoring.md) — Metrics 与告警
- [infra.md](../infra.md) — zot / RustFS / PostgreSQL 基础设施
- [apis/artifact-hub.yaml](../apis/artifact-hub.yaml) — REST API 字段契约
- [tenant-operator.md](tenant-operator.md) — per-tenant SA + 默认 ImagePullSecret / Secret 落地契约（`resolve?usage=inspect` 的隐式凭证来源）
- [compute-operator.md](compute-operator.md) — mlrun / mlservice handler 作为 resolve 消费方
- [platform.md](platform.md) — 工作区到 Artifacts namespace 的映射
