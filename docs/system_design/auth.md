# AxisML 认证与权限模型

本文档汇总 AxisML 控制平面的认证（authn）、授权（authz）与下游身份透传契约。所有控制面服务共享同一套身份模型与权限矩阵；身份与会话的持久化 schema 见 [database.md §4](database.md#4-platform)，HTTP API 端点见 [apis/platform.yaml](apis/platform.yaml)。系统级位置见 [overview.md](overview.md)。

---

## 1. 概述

| 角色 | 职责 |
| --- | --- |
| [Platform](components/platform.md) | 唯一的外部认证入口；签发并校验 JWT；维护用户 / 角色 / 租户绑定；向下游注入身份头 |
| [Cluster Manager](components/cluster-manager.md) / [compute-service](components/compute-service.md) / [artifact-hub](components/artifact-hub.md) | 只接受集群内 ClusterIP 调用；信任 Platform 注入的 `X-Axisml-User`，不做角色级鉴权 |
| Envoy Gateway | 在 Platform 之外另负责 **工作区 / 在线服务** 数据面入口的 JWT 校验（参见 [§5](#5-access-jwt) 与 [§6](#6-jwks)） |

约束：
- 所有外部 HTTP 流量必须先经 Platform 鉴权后才能落到下游服务。
- 下游服务的 Service 类型为 ClusterIP，不挂网关；NetworkPolicy 限制只允许 Platform 所在 namespace（默认 `axisml-platform`）以及必要的 System / Infra 管理路径入站。
- 当前仅支持内置用户体系（用户名 + bcrypt 密码）；OIDC / SAML 接入届时再设计抽象层，不在本文范围。

---

## 2. 身份（Identity）

### 2.1 内置身份源

- 用户名 + bcrypt 密码哈希存于 `users` 表；
- 登录成功后签发 JWT（`aud=axisml-platform`，主登录 token TTL 12h）；
- 登出 / 强制注销通过 `sessions` 表（按 `jti` 黑名单）实现；
- 表 schema 见 [database.md §4.1](database.md#51-schema)。

### 2.2 初始账号 / bootstrap

`axisml-platform bootstrap` 子命令在首次安装时执行：

- 创建内置角色（见 §3）；
- 创建初始 `system-admin` 账号 `admin` / `admin`（**首次登录强制改密**）；可通过环境变量 `AXISML_BOOTSTRAP_PASSWORD` 覆盖；
- 创建内置租户 `axisml-system`（承载 `visibility=public` 制品）。

OIDC / SAML 不在当前抽象内。

### 2.3 登录 / 登出 / 续期

API 路径与请求体见 [apis/platform.yaml](apis/platform.yaml) `Auth` tag（`POST /api/v1/auth/login` / `/logout` / `/refresh` / `GET /api/v1/auth/me`）。

---

## 3. RBAC 角色

三档**硬编码**内置角色，无运行时增删角色 / 调整权限的能力：

| 角色 | 范围 | 能力概览 |
| --- | --- | --- |
| `system-admin` | 全局 | 用户 / 租户 CRUD；资源池 / 资源单元 CRUD；读所有租户数据；维护 `axisml-system` 内置租户（含 `visibility=public` 制品） |
| `tenant-admin` | 单租户 | 本租户内成员管理、配额申请、对所有业务对象（Job / Service / Artifact / Workspace）的读写**（含跨 owner 的启停 / 删）** |
| `user` | 单租户 | 提交 / 管理自己创建的业务对象；读取本租户内的共享资产 |

### 3.1 全局权限矩阵

| 权限 | `system-admin` | `tenant-admin` | `user` |
| --- | :---: | :---: | :---: |
| 用户 CRUD | OK | NO | NO |
| 租户 CRUD（创建 / 暂停 / 恢复 / 删除） | OK | NO | NO |
| 租户成员管理 | OK | OK (@self) | NO |
| 租户配额 CRUD | OK | OK (@self) | NO |
| 资源池 / 资源单元 CRUD | OK | NO | NO |
| 工作区 / Job / Service 创建 | OK | OK (@self) | OK (@self) |
| 工作区 / Job / Service 启停 / 删 | OK | OK (@self, 跨 owner) | OK (@owner) |
| 制品 CRUD | OK | OK (@self, 跨 owner) | OK (@self, @owner) |
| `axisml-system` 制品 `visibility=public` 写 | OK | NO | NO |
| 跨租户读 | OK | NO | NO |

记号：
- `@self` = 仅对当前用户绑定的租户生效；
- `@owner` = 仅对当前用户创建的对象（`owner == X-Axisml-User`）生效；
- `@self, 跨 owner` = `tenant-admin` 在本租户内**可操作任意 owner** 的对象（不限本人创建）；
- `system-admin` 在所有 tenant 级 / owner 级判断上 **短路放行**。

---

## 4. 租户内角色绑定

### 4.1 关联表

`user_tenant_roles(user_id, tenant_name, role_id)` 表达「某用户在某租户内的角色」（schema 见 [database.md §4.1](database.md#51-schema)）。

### 4.2 `tenant_name` 作为稳定外键

- `tenant_name` 是 text 列，直接引用 compute `tenants.name`，**不**在 PG 层做跨服务 FK；
- 等价稳定 FK 的依据：compute `tenants.name` 在 `WHERE deleted_at IS NULL` 上 partial unique 且 **创建后不可变**（参见 [compute-service.md](components/compute-service.md)）；
- 租户软删 / 硬删时由应用层级联清理本租户在 `user_tenant_roles` 中的所有行（详见 [platform.md §4.1 租户编排](components/platform.md#41-租户编排)）。

### 4.3 角色绑定规则

- 一个 `(user_id, tenant_name)` 组合最多一条记录；
- 仅可绑定 `tenant-admin` / `user`；`system-admin` 由全局用户管理菜单维护；
- **自我保护**：当前操作者不能移除 / 降级自己在本租户的最后一个 `tenant-admin` 角色，否则返回 `409 last-tenant-admin`；
- 前端可见的租户列表 = 该用户在 `user_tenant_roles` 中绑定到的所有 `tenant_name`，再调 cluster-manager `LIST tenants` 取展示元数据。

API 入口见 [apis/platform.yaml](apis/platform.yaml) `Members` tag。

---

## 5. Access JWT

工作区与在线服务的数据面入口走独立的 access JWT（与主登录 token 分离），调用方先向 Platform 索取一次性 JWT，再用 `Authorization: Bearer <jwt>` 进入受 Envoy SecurityPolicy 保护的 HTTPRoute。

### 5.1 audience 与 TTL

| 用途 | `aud` claim | TTL 启动参数 | 默认 | 上限 |
| --- | --- | --- | --- | --- |
| 主用户登录（Platform API） | `axisml-platform` | （不可配） | 12h | — |
| 工作区接入 | `axisml-workspace` | `--workspace-access-jwt-ttl` | 1h | 24h |
| 在线服务接入 | `axisml-inference` | `--service-access-jwt-ttl` | 1h | 24h |

三种 audience **共享签名密钥** 但严格区分用途；网关侧 SecurityPolicy 校验 `aud` claim，防止跨用途滥用。

### 5.2 颁发流程

1. 用户先以主登录 token 通过 Platform 鉴权；
2. 调 `GET /api/v1/workspaces/{name}/access` 或 `GET /api/v1/mlservices/{name}/access`（见 [apis/platform.yaml](apis/platform.yaml) `Workspaces` / `MLServices` tag）；
3. 返回 `{ url, jwt, expiresAt }`；前端引导用户拼出 `<url>?token=<jwt>` 或在请求头注入；
4. 数据面网关基于 JWKS 验签 + `aud` 校验后放行。

---

## 6. JWKS

- Platform 在 `axisml-platform` namespace 内暴露 `/.well-known/jwks.json`；
- **走 ClusterIP，不暴露到 Envoy Gateway**；
- Envoy `SecurityPolicy` 通过 cluster-local URL（默认 `http://axisml-platform-platform.axisml-platform:8080/.well-known/jwks.json`，即 `<platform-service>.<platform-namespace>`）拉取公钥；
- 公钥旋转 = Platform 同时挂出新旧 kid，网关按 JWKS 自动发现新键；
- compute-operator 渲染数据面 HTTPRoute 时引用同一 `jwksUri`（参见 [compute-operator.md](components/compute-operator.md)）。

---

## 7. 下游身份透传

### 7.1 注入契约

Platform 校验通过后，所有出站到 cluster-manager / compute / artifacts 的请求自动注入：

```
X-Axisml-User: <username>
```

下游服务的契约：

| 服务 | 是否做角色级鉴权 | `X-Axisml-User` 的用途 |
| --- | --- | --- |
| cluster-manager | NO | 写入 `tenants.last_modified_by` + K8s Event |
| compute | NO | 写入 `mlservices.owner` / `mlruns.owner`；列表过滤 `@owner` |
| artifacts | NO | 审计 + ownership 归属 |

下游服务的网络面 **只接受 ClusterIP**；操作员（compute-operator / tenant-operator）直连时只携带 controller service identity，权限受限（例如 artifacts 仅允许 `resolve?usage=inspect`）。

### 7.2 mTLS

当前下游完全信任 `X-Axisml-User`，依赖网络面 NetworkPolicy 隔离。集群内 mTLS 不在本文范围。

---

## 8. 中间件契约

Platform 后端 `internal/auth` 暴露下列中间件供各功能 handler 拼装：

| 中间件 | 校验内容 | 短路规则 |
| --- | --- | --- |
| `RequireAuthenticated` | 主 token 有效且未在 `sessions` 黑名单 | — |
| `RequireSystemAdmin` | 当前用户具备 `system-admin` 角色 | — |
| `RequireTenantRole(role, tenantParam)` | 在路径变量 `tenantParam` 对应的租户上拥有 ≥ `role` 角色 | `system-admin` 短路放行 |
| `RequireWorkspaceOwner(nameParam)` | `@owner` 或在 workspace 所属租户上有 ≥ `tenant-admin`；租户由 `X-Axisml-Tenant` 头解析 | `system-admin` 短路 |
| `RequireServiceOwner(nameParam)` | `@owner` 或在 service 所属租户上有 ≥ `tenant-admin`；租户由 `X-Axisml-Tenant` 头解析 | `system-admin` 短路 |
| `RequireJobOwner(nameParam)` | `@owner` 或在 job 所属租户上有 ≥ `tenant-admin`；租户由 `X-Axisml-Tenant` 头解析 | `system-admin` 短路 |

实现要点：
- `RequireWorkspaceOwner` / `RequireServiceOwner` / `RequireJobOwner` 需要先调下游 GET 拿 `owner`；结果通过 `gin.Context.Set(...)` 注入后续 handler，避免重复调用；
- 角色升降序：`system-admin` > `tenant-admin` > `user`，所有 `≥` 比较按此序列；
- 失败统一返回 RFC 7807 problem：`401 unauthenticated` / `403 forbidden` / `409 last-tenant-admin`。
