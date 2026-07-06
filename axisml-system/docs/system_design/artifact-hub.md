# AxisML Artifact Hub 设计

## 1. 定位与边界

平台的制品元数据服务：以 PostgreSQL `artifacts` 表为元数据权威，按 `(tenantScope, name, version)` 三元组寻址，`kind` 是制品的不可变属性（非寻址键）。REST API 只暴露单一 `artifacts` 资源，不再按类型分资源族；bytes 由调用方凭短期凭证直连 zot / RustFS。REST / PG 中兼容字段仍名为 `namespace`，但其语义是 tenant scope，不是 K8s Namespace。

| 做 | 不做 |
| --- | --- |
| Artifact CRUD、两阶段写（initiate / complete）、resolve | 制品 bytes 存取（→ zot / RustFS 直连） |
| 上传 / 下载凭证签发（OCI scope token / S3 prefix-scoped STS） | 用户认证与角色鉴权（→ [auth.md](../../../axisml-platform/docs/system_design/auth.md)） |
| `kind` 按 Handler 注册表分发（model / dataset / image）——API 层不分资源族，kind 进 initiate body | tenant 存在性与权限校验（namespace 字段由 compute 兜底 tenant 语义） |
| GC：Uploading TTL、Failed 留存、Deleting 推进 | 反向孤儿主动清理（仅告警）；跨 namespace 级联删除 |
| `visibility=public` 全局可见制品（落 `default` 内置 tenant scope） | tenant Secret 落地（→ [tenant-operator.md](tenant-operator.md)） |

## 2. 架构

```
  Platform ──REST──▶ Artifact Hub
                ├─ PG 读写 ──▶ PostgreSQL (artifacts 表)
                └─ 签 token / HEAD / GC ──▶ zot(OCI) / RustFS(S3)
                                              ▲ 直传 / 直拉（短期凭证）
                                       客户端 / workload
```

```
┌──────────── Artifacts (Go) ────────────┐
│ HTTP API (Gin): initiate / complete /   │
│   resolve / GET / DELETE + middleware   │
│ ArtifactHandler Registry (compile-time) │
│   handlers/{model, dataset, image}      │
│ GC Worker (leader-only, 5 min tick)     │
└──────────────────────────────────────────┘
```

## 3. 核心模型

| 实体 | 含义 | 寻址键 | 备注 |
| --- | --- | --- | --- |
| Artifact | 版本化制品 | `(tenantScope, name, version)` | 三元组创建后不复用；兼容字段名为 `namespace`；`name` 在 tenantScope 内跨 kind 唯一 |

- `kind`：`model` / `dataset` / `image`，创建时经 initiate body 提供、由 Handler registry 校验，进 `Ready` 前后均不可变；是 Handler 派发键，但不参与寻址。
- `namespace` 是历史兼容字段名，表示租户 `identifier` 对应的 tenant scope；Artifacts 不解析、不做存在性校验，仅作不透明分区键。`default` 是内置 tenant scope，承载 `visibility=public` 制品，其 K8s Namespace 是 `axisml-tenant`。
- `visibility`：`tenant`（默认，仅本 tenant scope 可见）/ `public`（全局可见；仅允许在 `default` 下创建，由上游做 RBAC 兜底）。
- `source`：`webUpload` / `oras`（model CLI 推送）/ `dockerPush`（image 本机推送）/ `external`（登记远端、免上传，§5.1）——进 `Ready` 后冻结。
- 状态机：`Uploading` / `Ready` / `Failed` / `Deleting` / `Deleted`（§6）。
- 扩展元数据 `labels` / `annotations` 对齐 [database.md §1.6](database.md#16-扩展元数据-labels--annotations)；artifacts 无 CR，扩展位只落 PG。

字段级 schema 见 [database.md §3](database.md#3-artifact-hub)；spec 子字段见 [openapi/artifact-hub.yaml](../apis/artifact-hub.yaml)。

## 4. 核心功能

写路径见 §5.1，读路径见 §5.2。三类 Kind 差异：

| 项 | Model | Dataset | Image |
| --- | --- | --- | --- |
| StorageKind | `oci`（zot） | `s3`（RustFS） | `oci`（zot） |
| 必填 spec | `framework`（pytorch / tensorflow / onnx / safetensors / gguf / custom）+ `format`（OCI media type） | `format`（parquet / jsonl / csv / webdataset / tfrecord / custom） | `purpose`（training / inference / workspace / custom） |
| URI 模板 | `<oci-host>/namespaces/<ns>/models/<name>:<version>` | `s3://axisml-artifact-hub/namespaces/<ns>/datasets/<name>/<version>/` | `<oci-host>/namespaces/<ns>/images/<name>:<version>` |
| digest | manifest digest | SHA256 of 存储的 `artifact-manifest.json` 字节（complete 时后端 GET 重算比对；S3 未配置时降级为记录 claim 不校验） | manifest digest |
| 外部登记 | `remoteUri` + `remoteSourceKind`（s3 / oci / http / hf / custom） | — | `sourceImageRef`（远端镜像地址） |
| 主要消费方 | mlservice handler → KServe `storageUri`（补 `oci://`）或 native env `AXISML_MODEL_URI` | mlrun handler 注入 env `AXISML_DATASET_URI`/`DIGEST` + init container / csi-s3 挂到 `/data` | mlrun / mlservice handler 用作 Pod `image`；imagePullSecret 由 per-tenant SA 默认携带 |

## 5. 关键机制

### 5.1 写路径：两阶段提交

```
客户端          Artifacts            zot/RustFS    PG
 │─ initiate ─▶│─ ValidateSpec ─────▶│             │
 │             │─ insert(Uploading) ───────────────▶│
 │◀ creds,uri ─│─ sign scope token / STS            │
 │─ push / PutObject ────────────────▶│             │
 │─ complete(digest) ─▶│─ HEAD manifest / GET artifact-manifest.json
 │                     │─ update(Ready, digest) ────▶│
 │◀── 200 ─────────────│                             │
```

**幂等性**：重复 initiate 同三元组 `(namespace, name, version)`——未过期 `Uploading` 返原凭证，其他终态 409（同 version 不可复活，复用先 DELETE 旧行；同名不同 kind 同 version 亦冲突）；重复 complete——`Uploading` 正常推进，`Ready` 且 digest 一致 → 200、不一致 → 409 `DigestMismatch`，其他 → 409；24h 内未 complete 的 `Uploading` 由 GC 转 `Failed` 并清后端残留 blob。

**外部登记（`source=external`）**：initiate 给定远端来源即登记，不返上传凭证、无客户端 push 阶段；Artifacts 异步从远端拉取 / 同步到后端，凭 HEAD 通过转 `Ready`（失败转 `Failed`，GC 按同档清理）。

### 5.2 读路径：resolve

`GET .../artifacts/{name}/{version}/resolve?usage={inspect|download}`

| usage | 调用方 | 凭证形态 |
| --- | --- | --- |
| `inspect` | 集群内 operator（mlservice / mlrun handler） | 不签发凭证；operator 派生的 Pod 经 per-tenant ServiceAccount（由 tenant-operator 落地，已默认携带 zot / RustFS 的 imagePullSecrets / Secret）拉取；Artifacts 只回 `uri` + `digest` |
| `download` | 终端用户 / 脚本（经上游 / Gateway） | OCI pull scope token / S3 prefix-scoped STS，TTL=1h（额外回 `pull_credentials` / `expires_at`） |

公共字段：`storage_kind`（`oci` / `s3`）、`uri`、`digest`（PG 读，未 Ready 为空）、`visibility`。`uri` 由 `Handler.BuildPullURI` 拼装：OCI kind 锚定 digest（`<name>@<digest>`），消费方拉取不可变内容而非可复推的 tag；S3 kind 无内容寻址路径，返回按 version 定死的 prefix（digest 并列返回供客户端自校验）。

### 5.3 GC 与生命周期清理

GC worker（leader-only，每 5 分钟）扫描 PG 三类谓词：

| 谓词 | 动作 |
| --- | --- |
| `status='Uploading' AND created_at < now()-24h` | 标 `Failed`，**同步**调 `Handler.GCBackend` 清后端残留 blob |
| `status='Failed' AND updated_at < now()-30d AND deleted_at IS NULL` | 转 `Deleting`；blob 已在转 Failed 时清空，PG 行保留 30 天供诊断 |
| `status='Deleting'` | 调 `Handler.GCBackend`；成功后 `status='Deleted'` + 写 `deleted_at` |

`GCBackend` 须幂等，把"对象不存在"视作成功（吞 OCI `MANIFEST_UNKNOWN` / S3 `NoSuchKey` / 404）。反向孤儿（后端有 blob 但 PG 无 Ready 行）仅告警；整 namespace 批量删除由调用方逐条 DELETE，Artifacts 不提供级联删除端点。

### 5.4 元数据 / 字节分离权威

| 类别 | 权威方 | 流向 |
| --- | --- | --- |
| 元数据（artifact 行） | PG | API → PG |
| 制品 bytes | zot / RustFS | 客户端 / workload → 后端（直传 / 直拉，不经服务） |
| digest | 后端 | 后端 → complete API → PG |
| 上传 / 下载凭证 | Artifacts 签发 | Artifacts → 上游 → 客户端（短期 token，TTL=1h）；operator 走 inspect，不签明文 |

## 6. 接口契约

| 类别 | 内容 | 引用 |
| --- | --- | --- |
| 对外 REST | `/api/v1/namespaces/{ns}/artifacts/{name}[/{version}[/{complete,resolve}]]`；版本级 GET / PATCH / DELETE 同前缀。单一资源族服务所有 kind——`kind` 经 initiate body 传入（`required`），列表端点接受可选 `?kind=` 过滤 | [openapi/artifact-hub.yaml](../apis/artifact-hub.yaml) `Artifacts` tag |
| 身份头 | 调用方注入 `X-Axisml-User`，本服务仅做 ownership 归属（[auth.md §6](../../../axisml-platform/docs/system_design/auth.md#6-下游身份透传)） | — |
| 列表查询 | list 支持 `?labelSelector=`（K8s grammar） | [database.md §1.6](database.md#16-扩展元数据-labels--annotations) |
| 错误格式 | HTTP 标准码 + RFC 7807 problem+json | — |
| 写后语义 | initiate PG 提交后返上传凭证；Ready 由 complete 推进；PATCH 纯 PG mutation 立即可读 | — |

**PATCH 可变字段**（任何非终态生效）：`displayName` / `description` / `labels` / `annotations`。其它一律不可变（含 `visibility`），否则 `400 ImmutableField`；`Deleting` / `Deleted` 行 PATCH 返 `409 ArtifactTerminal`。`labels` / `annotations` 整体 map 替换。

**ArtifactHandler 接口**（编译期注册，key=`Kind()`）：`Kind()`（registry 主键）· `StorageKind()`（`oci`/`s3`）· `BuildStorageURI`（上传/tag 形式，即时拼装，不读 PG / 不调后端）· `BuildPullURI`（resolve 消费引用；OCI 锚 digest，S3 返回 prefix）· `ValidateSpec`（纯函数）· `InitiateUpload`（签上传凭证，幂等）· `VerifyComplete`（调后端 HEAD / GET manifest 校验 digest）· `GCBackend`（删后端残留 blob，幂等，NotFound 视作成功）。

**状态机**：

```
Uploading ─(complete + HEAD 通过)─▶ Ready
          ─(complete 失败 / GC TTL)─▶ Failed
Ready / Failed ─(DELETE)─▶ Deleting ─(GCBackend 成功)─▶ Deleted
```

未注册 Kind 的请求 → 400，不创建 PG 行。

## 7. 依赖

| 依赖 | 用途 |
| --- | --- |
| PostgreSQL | 元数据权威；与 compute 共享 database，单表 `artifacts`（[database.md](database.md)） |
| zot | OCI 后端；Artifacts 持 admin 凭证签 scope token / HEAD 校验 / GC 删 blob，客户端持短期 token 直连 |
| RustFS | S3 后端（dataset）；Artifacts 持 admin 凭证经 SigV4 直连：启动 EnsureBucket 建桶、complete 时 GET `artifact-manifest.json` 重算 SHA256 校验 digest、GC 删 version prefix。endpoint（`s3.endpoint`）+ access/secret key 由 Helm 注入（`artifactHub.storage.s3.*`，opt-in）；**未配置 endpoint 时降级为记录 claim 不校验**（单机 dev 形态无对象存储），bucket `axisml-artifact-hub` |
| tenant-operator | 在 workload namespace 落地 per-tenant ServiceAccount + Secrets（默认 imagePullSecret 拉 zot、env / volume 读 RustFS）；Artifacts 不参与 Secret 落地、不在 resolve 返回 secret 名（[tenant-operator.md](tenant-operator.md)） |

## 8. 运行时形态

| 维度 | 值 |
| --- | --- |
| 进程 | 单二进制 `axisml-artifact-hub`；启动执行 golang-migrate embedded migration |
| 副本 | API 默认 `replicas=1`（无状态可水平扩）；GC worker 单 leader，经 PG session 级 advisory lock（`pg_try_advisory_lock`）选主，leader 崩溃时连接断开即自动释放锁 |
| 暴露端口 | API `:8080`；Metrics `:8081`；Probes `:8082`（`/readyz` 校验 PG），均不对外 |
| K8s 依赖 | 无。Artifacts 不连 K8s API（不 watch CRD、不持 Lease、不需 client-go / controller-runtime），因此不需要任何 K8s RBAC，只读写 PG + 调后端 HTTP；选主权威与数据权威统一在 PG |
| Helm / 镜像 | 见 [deployment.md](../../../docs/deployment.md) |

## 9. 相关引用

- [high_level_design.md](../../../docs/high_level_design.md) — Artifacts 在控制平面的位置与系统不变量
- [auth.md](../../../axisml-platform/docs/system_design/auth.md) — `X-Axisml-User` 注入与传播
- [database.md](database.md) — `artifacts` 表 schema
- [deployment.md](../../../docs/deployment.md) · [infra.md](../../../axisml-infra/docs/system_design/overview.md)
- [openapi/artifact-hub.yaml](../apis/artifact-hub.yaml) — REST 契约源
- [tenant-operator.md](tenant-operator.md) — per-tenant SA + 默认 Secret 落地契约（inspect 的隐式凭证来源）
- [compute-operator.md](compute-operator.md) — 消费 Platform 已解析并快照到 workload spec 的制品引用
