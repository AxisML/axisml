# AxisML 认证与权限模型

定义控制平面的认证（authn）、授权（authz）与下游身份透传契约。身份 / 会话 schema 见 [database.md §4](../database.md#4-platform)，端点见 [openapi/platform.yaml](../../openapi/platform.yaml)。

## 1. 概述

控制面与数据面走两条独立鉴权路径：

| 角色 | 职责 |
| --- | --- |
| [Platform](backend.md) | 控制面唯一外部入口：签发 / 校验登录 JWT，维护用户 / 角色 / 租户绑定，向下游注入身份头 |
| cluster-manager / compute-service / artifact-hub | 仅接受集群内 ClusterIP 调用，信任 `X-Axisml-User`，自身不做角色鉴权（§6） |
| Envoy Gateway | 数据面入口；工作区 JWT SecurityPolicy 尚未交付，相关路由当前 fail-closed；在线服务当前无鉴权 |

约束：控制面所有外部流量必须先经 Platform 鉴权再到下游；下游 Service 为 ClusterIP，由 NetworkPolicy 限定只允许 Platform namespace 及必要管理路径入站。数据面（工作区 / 在线服务）经 Envoy Gateway 直达业务 Pod，不经 Platform。当前仅内置用户体系（用户名 + bcrypt）；OIDC / SAML 留待后续。

## 2. 身份与登录

- 用户名 + bcrypt 密码存于 `users` 表；登录签发 JWT（`aud=axisml-platform`，TTL 12h）；登出 / 强制注销通过 `sessions` 表按 `jti` 黑名单实现（schema 见 [database.md §4](../database.md#4-platform)）。
- 端点：`POST /api/v1/auth/{login,logout,refresh}`、`GET /api/v1/auth/me`（`Auth` tag）。
- **bootstrap**（`axisml-platform bootstrap`，首次安装）：创建内置角色；创建初始 `system-admin` 账号 `admin`/`admin`（**首登强制改密**，可由 `AXISML_BOOTSTRAP_PASSWORD` 覆盖）；创建内置租户 `default`，其 K8s Namespace 为 `axisml-tenant`（承载 `visibility=public` 制品）。

## 3. RBAC 角色

三档**硬编码**角色，运行时不可增删或调权：

| 角色 | 范围 | 能力 |
| --- | --- | --- |
| `system-admin` | 全局 | 用户 / 租户 / 配额 CRUD；资源池 / 资源单元 CRUD；读所有租户；维护内置租户 `default`（含 `public` 制品） |
| `tenant-admin` | 单租户 | 本租户成员管理；对本租户全部业务对象读写，**含跨 owner 启停 / 删** |
| `user` | 单租户 | 提交 / 管理自己创建的业务对象；读取本租户共享资产 |

### 3.1 权限矩阵

| 权限 | `system-admin` | `tenant-admin` | `user` |
| --- | :---: | :---: | :---: |
| 用户 CRUD | OK | NO | NO |
| 租户 CRUD（创建 / 停用 / 启用 / 删除） | OK | NO | NO |
| 租户成员管理 | OK | OK (@self) | NO |
| 配额 CRUD | OK | NO | NO |
| 资源池 / 资源单元 CRUD | OK | NO | NO |
| 工作区 / Job / 实验 / Service 创建 | OK | OK (@self) | OK (@self) |
| 工作区 / Job / 实验 / Service 启停 / 删；TensorBoard 启停 | OK | OK (@self, 跨 owner) | OK (@owner) |
| 制品 CRUD | OK | OK (@self, 跨 owner) | OK (@self, @owner) |
| `default` 租户制品 `visibility=public` 写 | OK | NO | NO |
| 跨租户读 | OK | NO | NO |

记号：`@self` = 仅对当前用户绑定的租户生效；`@owner` = 仅对 `owner == X-Axisml-User` 的对象生效；`@self, 跨 owner` = `tenant-admin` 在本租户内可操作任意 owner 的对象；`system-admin` 在所有 tenant / owner 级判断上**短路放行**。

## 4. 租户内角色绑定

- 关联表 `user_roles(user_id, tenant_name, role)` 表达"某用户在某租户内的角色"；`tenant_name` 引用 Platform 自有 `tenants.identifier`（同库，可建真实 FK；当前走应用层约束）。其稳定性依据是 `tenants.identifier` 在 `WHERE deleted_at IS NULL` 上 partial unique 且创建后不可变。
- **绑定规则**：一个 `(user_id, tenant_name)` 最多一条记录；仅可绑定 `tenant-admin` / `user`（`system-admin` 由全局用户管理维护）；**自我保护**——不能移除 / 降级自己在本租户的最后一个 `tenant-admin`（否则 `409 last-tenant-admin`）；前端可见租户列表 = 该用户绑定的所有 `tenant_name`，展示元数据取 Platform `tenants` 表、运行态按需实时回源 cluster-manager。
- 租户软删 / 硬删时由应用层级联清理本租户在 `user_roles` 的所有行。端点：`Members` tag。

## 5. 数据面接入

工作区与在线服务入口由 Envoy Gateway 守卫，与控制面登录 token 分离。

| 用途 | `aud` | 传输 | TTL | 默认 / 上限 |
| --- | --- | --- | --- | --- |
| 控制面登录 | `axisml-platform` | 前端持有 | 不可配 | 12h |
| 工作区接入（规划） | `axisml-workspace` | Cookie | `--workspace-access-jwt-ttl` | 1h / 24h |

两 audience **共享签名密钥**，但 Envoy SecurityPolicy 严格校验 `aud` 防滥用。

- **工作区接入（规划，当前 fail-closed）**：目标流程仍是 Cookie + `aud=axisml-workspace` JWT + Envoy SecurityPolicy；但 SecurityPolicy 派生尚未交付，因此 compute-operator 当前拒绝 `kind=workspace|tensorboard` 的任何外部 route，并拒绝其他 MLService 的 `route.auth.type != none`。Platform 不暴露 workspace access 端点，不会创建看似受保护但实际裸露的 HTTPRoute。
- **在线服务接入（API KEY，规划中）**：设计为 API KEY，由后续独立"API KEY 管理"功能提供。**本版本不实现**，故当前在线服务数据面**无鉴权保护**。
- **JWKS**：Platform 在 `axisml-platform` namespace 暴露 `/.well-known/jwks.json`，**走 ClusterIP、不暴露到 Gateway**；Envoy `SecurityPolicy` 经 cluster-local URL 拉公钥；公钥旋转 = 新旧 `kid` 并挂、网关自动发现；compute-operator 渲染数据面 HTTPRoute 时引用同一 `jwksUri`。

## 6. 下游身份透传

Platform 鉴权通过后，对下游出站请求自动注入 `X-Axisml-User: <username>`：

| 服务 | 角色级鉴权 | 用途 |
| --- | --- | --- |
| cluster-manager | NO | 写 CR annotation `axisml.io/last-modified-by` + K8s Event |
| compute | NO | 写 `mlservices.owner` / `mlruns.owner`；列表按 `@owner` 过滤 |
| artifacts | NO | ownership 归属 |

下游网络面只接受 ClusterIP；Cluster Manager / Compute Service / Artifact Hub 的 API `:8080` 由 NetworkPolicy 限定仅 `axisml-platform` namespace 可访问，metrics 单独允许监控 namespace，probes 单独放行 kubelet。下游完全信任 `X-Axisml-User`，靠该网络边界隔离；集群内 mTLS 不在本文范围。

## 7. 中间件契约

Platform 后端 `internal/auth` 暴露下列中间件供 handler 拼装（角色升降序 `system-admin` > `tenant-admin` > `user`，所有 `≥` 比较按此序；失败统一返回 RFC 7807：`401 unauthenticated` / `403 forbidden` / `409 last-tenant-admin`）：

| 中间件 | 校验内容 | 短路 |
| --- | --- | --- |
| `RequireAuthenticated` | 主 token 有效且未在 `sessions` 黑名单 | — |
| `RequireSystemAdmin` | 当前用户具备 `system-admin` | — |
| `RequireTenantRole(role, tenantParam)` | 在路径变量对应租户上拥有 ≥ `role` | `system-admin` 短路 |
| `Require{Workspace,Service,Job,Experiment}Owner(nameParam)` | `@owner` 或在对象所属租户上有 ≥ `tenant-admin`；租户由 `X-Axisml-Tenant` 头解析 | `system-admin` 短路 |

`Require*Owner` 需先调下游 GET 拿 `owner`（实验 owner 取自 Platform PG 定义行），经 `gin.Context.Set` 注入后续 handler 避免重复调用；TensorBoard 启停复用 `RequireExperimentOwner`（普通成员不可启动）。
