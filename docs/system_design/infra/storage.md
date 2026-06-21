# 存储

平台有三类持久化需求，分别选用不同技术实现：目录型大文件（对象存储）、不可变内容寻址制品（OCI Registry）、控制面关系元数据（数据库）；外加一类**非持久**的状态缓存（缓存）。四者均归 Infra 层、对调用方暴露标准协议、不内置租户模型。

## 1. 对象存储

### 需求

承载数据集等**目录型 / 多文件**制品的 bytes，要求：标准 S3 协议（生态通用、可换实现）；客户端 / workload 凭短期凭证直连读写、不经业务服务代理大文件；按 bucket / prefix 自管命名隔离。

### 技术选型

选用 **[RustFS](https://rustfs.dev/)**（Apache 2.0、Rust 实现、S3 API 兼容）。理由：S3 兼容满足协议要求且切换成本有限；Apache 2.0 许可规避 MinIO 自 2021 转 AGPLv3 的商用传染风险。

### 对外契约

- 调用方通过 S3 SDK 访问，对具体实现无感知；admin 凭证由 `axisml-infra` 自动生成（或预置），presigned URL 与短期凭证由调用方按需签发。
- 命名隔离：bucket / prefix 由调用方组织，本功能不内置 ACL 或租户模型。
- 部署模式：Standalone（单 Pod + PVC）/ Distributed 4×4 / 16×1。

## 2. OCI Registry

### 需求

承载模型权重与容器镜像等**不可变内容寻址**制品，要求：内容寻址（`@digest`）保证可复现引用；支持非容器制品（模型权重）；scope 限定的拉取凭证；manifest 完整性校验；后端存储可插拔。

### 技术选型

选用 **[zot](https://zotregistry.dev/)**（CNCF Sandbox、单二进制 Go 实现）。理由：原生支持 OCI Distribution v2 + 1.1 artifact manifest，对非容器制品的 `artifactType` 语义完整；后端可插拔（filesystem / S3，可把 blob 切到 RustFS）。

### 对外契约

| 能力 | 说明 |
| --- | --- |
| artifact manifest | 原生支持 `application/vnd.oci.image.manifest.v1+json` + `artifactType`，承载 ML 模型权重等非容器制品 |
| 内容寻址 | `<repo>@sha256:<digest>` 不可变引用 |
| Bearer token 鉴权 | scope-limited（`repository:<repo>:push`/`pull`） |
| Manifest 校验 | `HEAD /v2/<repo>/manifests/<ref>` 返回 digest，调用方据此做完整性校验 |

Infra 层提供 zot endpoint（ConfigMap）、admin 凭证（平台级 Secret）、公共拉取凭证（`axisml-tenant` Namespace Secret，由 `default` Tenant 管理）；repo 路径命名 / 租户隔离 / scope token 形态由调用方决定。部署：Standalone（filesystem）/ HA 3×（共享 S3 后端）。

## 3. 数据库

### 需求

控制面元数据的统一关系存储，要求：事务与关系约束；多服务共用一库、按 schema / 表前缀逻辑隔离；支持生产外接托管实例（RDS）。

### 技术选型

选用 **PostgreSQL**（bitnami/postgresql 子 chart）。理由：成熟关系库满足事务与生态要求；复用现成 chart 免自写 StatefulSet 模板；`externalDatabase` 段保留用于生产外接 RDS。作为第三方依赖归 Infra 层（与对象存储 / OCI Registry 同性质）。

### 对外契约

- 部署在 `axisml-infra`（Service `axisml-database`）。System 层经跨 namespace FQDN `axisml-database.axisml-infra:5432` 连接，凭据从共享 `database.auth.password` 自渲染为本 namespace Secret。
- 模式：内置（StatefulSet + PVC）/ 外部（`database.enabled=false` + `externalDatabase.*` 接自建 / RDS）。
- schema 迁移由各调用方二进制内嵌 `golang-migrate` 在启动时执行（依赖 PG advisory lock 避免并发迁移）。

schema 细节见 [database.md](../database.md)；部署模式见 [deployment.md §7](../deployment.md#7-postgresql-部署模式)。

## 4. 缓存

### 需求

控制面热点读的可选加速：高频、读多写少、值可由源库重建（典型为认证路径的会话有效性与身份 / RBAC 解析）。要求：低延迟 key/value；TTL 过期；多副本共享（替代进程内缓存）；**非权威**——缓存不可达时调用方必须回退源库，缓存绝不作为唯一真相。

### 技术选型

选用 **Redis**（bitnami/redis 子 chart，`architecture: standalone`）。理由：成熟 key/value 与 TTL 语义；复用现成 chart。缓存内容均为可重建数据，单实例即可，无需 sentinel / 副本 HA——宕机或重启只触发一次回源（及会话强制重登），不丢业务真相。

### 对外契约

- 部署在 `axisml-infra`（Service `axisml-redis-master`）。调用方经跨 namespace FQDN `axisml-redis-master.axisml-infra:6379` 连接，凭据从共享 `cache.auth.password` 自渲染为本 namespace Secret。
- 可选依赖：调用方未配置地址即跳过缓存（直连源库）；运行中缓存出错按操作回退源库，不影响请求成功。
- key 隔离：调用方按 key 前缀自行命名（如 Platform 用 `platform:`）。

部署模式见 [deployment.md §8](../deployment.md#8-redis-缓存部署模式)；Platform 的具体缓存对象与失效策略见 [platform/auth.md §2.1](../platform/auth.md#21-会话与身份缓存)。
