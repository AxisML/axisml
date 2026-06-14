# AxisML 认证与权限模型

本文档定义 AxisML 控制平面的认证（authn）、授权（authz）与下游身份透传契约。身份 / 会话的持久化 schema 见 [database.md §4](database.md#4-platform)，HTTP 端点见 [apis/platform.yaml](apis/platform.yaml)，系统级定位见 [overview.md](overview.md)。

---

## 1. 概述

控制面（Platform API + 下游服务）与数据面（工作区 / 在线服务）走两条独立的鉴权路径：

| 角色 | 职责 |
| --- | --- |
| [Platform](components/platform.md) | 控制面唯一外部入口：签发 / 校验登录 JWT，维护用户 / 角色 / 租户绑定，向下游注入身份头 |
| [cluster-manager](components/cluster-manager.md) / [compute-service](components/compute-service.md) / [artifact-hub](components/artifact-hub.md) | 仅接受集群内 ClusterIP 调用，信任 Platform 注入的 `X-Axisml-User`，自身不做角色鉴权（见 §6） |
| Envoy Gateway | 数据面入口：校验工作区的 Cookie JWT（见 §5）；在线服务当前无鉴权 |

约束：
- 控制面所有外部流量必须先经 Platform 鉴权再到下游；下游 Service 为 ClusterIP，由 NetworkPolicy 限定只允许 Platform namespace（默认 `axisml-platform`）及必要的 System / Infra 管理路径入站。
- 数据面（工作区 / 在线服务）经 Envoy Gateway 直达业务 Pod，不经过 Platform。
- 当前仅内置用户体系（用户名 + bcrypt）；OIDC / SAML 留待后续抽象，不在本文范围。

---

## 2. 身份与登录

### 2.1 身份源

- 用户名 + bcrypt 密码哈希存于 `users` 表；
- 登录签发 JWT（`aud=axisml-platform`，TTL 12h）；
- 登出 / 强制注销通过 `sessions` 表按 `jti` 黑名单实现；
- schema 见 [database.md §4.1](database.md#51-schema)。

### 2.2 bootstrap

`axisml-platform bootstrap` 首次安装时执行：

- 创建内置角色（§3）；
- 创建初始 `system-admin` 账号 `admin` / `admin`（**首次登录强制改密**；可由环境变量 `AXISML_BOOTSTRAP_PASSWORD` 覆盖）；
- 创建内置租户 `axisml-system`（承载 `visibility=public` 制品）。

### 2.3 登录 / 登出 / 续期

端点见 [apis/platform.yaml](apis/platform.yaml) `Auth` tag：`POST /api/v1/auth/login` `/logout` `/refresh`、`GET /api/v1/auth/me`。

---

## 3. RBAC 角色

三档**硬编码**角色，运行时不可增删角色或调整权限：

| 角色 | 范围 | 能力 |
| --- | --- | --- |
| `system-admin` | 全局 | 用户 / 租户 CRUD；资源池 / 资源单元 CRUD；读所有租户；维护内置租户 `axisml-system`（含 `visibility=public` 制品） |
| `tenant-admin` | 单租户 | 本租户成员管理、配额申请；对本租户全部业务对象（Job / Service / Artifact / Workspace）读写，**含跨 owner 启停 / 删** |
| `user` | 单租户 | 提交 / 管理自己创建的业务对象；读取本租户共享资产 |

### 3.1 权限矩阵

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

- **关联表** `user_tenant_roles(user_id, tenant_name, role)` 表达「某用户在某租户内的角色」（schema 见 [database.md §4.1](database.md#51-schema)）。
- **`tenant_name` 作稳定外键**：text 列直接引用 compute `tenants.name`，不在 PG 层做跨服务 FK；其稳定性依据是 `tenants.name` 在 `WHERE deleted_at IS NULL` 上 partial unique 且创建后不可变（见 [compute-service.md](components/compute-service.md)）。租户软删 / 硬删时由应用层级联清理本租户在 `user_tenant_roles` 的所有行（见 [platform.md §4.1](components/platform.md#41-租户编排)）。
- **绑定规则**：
  - 一个 `(user_id, tenant_name)` 组合最多一条记录；
  - 仅可绑定 `tenant-admin` / `user`；`system-admin` 由全局用户管理维护；
  - **自我保护**：操作者不能移除 / 降级自己在本租户的最后一个 `tenant-admin`，否则返回 `409 last-tenant-admin`；
  - 前端可见的租户列表 = 该用户绑定的所有 `tenant_name`，再调 compute-service `LIST tenants` 取展示元数据（租户归 compute 持有，见 [compute-service.md](components/compute-service.md)）。

端点见 [apis/platform.yaml](apis/platform.yaml) `Members` tag。

---

## 5. 数据面接入

工作区与在线服务的入口由 Envoy Gateway 守卫，与控制面登录 token 分离。

### 5.1 audience 与 TTL

| 用途 | `aud` | 传输 | TTL 参数 | 默认 | 上限 |
| --- | --- | --- | --- | --- | --- |
| 控制面登录 | `axisml-platform` | 前端持有 | （不可配） | 12h | — |
| 工作区接入 | `axisml-workspace` | Cookie | `--workspace-access-jwt-ttl` | 1h | 24h |

两种 audience **共享签名密钥**，但 Envoy SecurityPolicy 严格校验 `aud`，防止跨用途滥用。

### 5.2 工作区接入（Cookie + JWT）

工作区是 Web 服务，统一走 Cookie：

1. 用户先以登录 token 通过 Platform 鉴权；
2. 调 `GET /api/v1/workspaces/{name}/access`（见 [apis/platform.yaml](apis/platform.yaml) `Workspaces` tag），取得 workspace access JWT（`aud=axisml-workspace`）与目标 `url`；
3. JWT 写入工作区域名下的 Cookie；
4. 浏览器访问工作区 HTTPRoute 时自动携带 Cookie，Envoy SecurityPolicy 从 Cookie 提取 JWT，基于 JWKS（§5.4）验签 + 校验 `aud` 后放行。

### 5.3 在线服务接入（API KEY，规划中）

在线服务（推理端点）的数据面鉴权设计为 **API KEY**，由后续独立的「API KEY 管理」功能提供。**本版本不实现**，故当前在线服务数据面**无鉴权保护**——既不签发 access JWT，也不校验 API KEY，对应的颁发端点一并延后。

### 5.4 JWKS

- Platform 在 `axisml-platform` namespace 暴露 `/.well-known/jwks.json`，**走 ClusterIP，不暴露到 Envoy Gateway**；
- Envoy `SecurityPolicy` 经 cluster-local URL（默认 `http://axisml-platform-platform.axisml-platform:8080/.well-known/jwks.json`，即 `<platform-service>.<platform-namespace>`）拉取公钥；
- 公钥旋转 = 新旧 `kid` 并挂，网关按 JWKS 自动发现；
- compute-operator 渲染数据面 HTTPRoute 时引用同一 `jwksUri`（见 [compute-operator.md](components/compute-operator.md)）。

---

## 6. 下游身份透传

Platform 鉴权通过后，对 cluster-manager / compute / artifacts 的出站请求自动注入：

```
X-Axisml-User: <username>
```

| 服务 | 角色级鉴权 | `X-Axisml-User` 用途 |
| --- | --- | --- |
| cluster-manager | NO | 写 `tenants.last_modified_by` + K8s Event |
| compute | NO | 写 `mlservices.owner` / `mlruns.owner`；列表按 `@owner` 过滤 |
| artifacts | NO | 审计 + ownership 归属 |

- 下游网络面**只接受 ClusterIP**；操作员（compute-operator / tenant-operator）直连时仅持 controller service identity，权限受限（如 artifacts 仅允许 `resolve?usage=inspect`）。
- 下游完全信任 `X-Axisml-User`，靠 NetworkPolicy 隔离；集群内 mTLS 不在本文范围。

---

## 7. 中间件契约

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

- `RequireWorkspaceOwner` / `RequireServiceOwner` / `RequireJobOwner` 需先调下游 GET 拿 `owner`，结果经 `gin.Context.Set(...)` 注入后续 handler，避免重复调用；
- 角色升降序：`system-admin` > `tenant-admin` > `user`，所有 `≥` 比较按此序列；
- 失败统一返回 RFC 7807 problem：`401 unauthenticated` / `403 forbidden` / `409 last-tenant-admin`。
