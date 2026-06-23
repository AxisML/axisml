# AxisML Platform 层概要

Platform 层是 AxisML **唯一直接面向用户、唯一对外暴露**的层：承担身份接入、业务编排与视图层映射，把用户操作翻译为对 System 层（cluster-manager / compute / artifacts）的内部调用。外部流量经 Envoy Gateway 进入 Platform，下游全部 ClusterIP 且信任 Platform 注入的 `X-Axisml-User`。

## 组成

| 文档 | 职责 |
| --- | --- |
| [backend.md](backend.md) | Go BFF：身份认证 / JWT、跨服务业务编排、持有租户持久记录 + Job/Experiment/Model/Image 四张 name 级定义 + 视图层映射；不持有下游实例状态（一律实时回源） |
| [frontend.md](frontend.md) | React SPA：路由 / 数据获取 / i18n / 数据面接入；消费后端 REST，不直连 System / Infra |
| [auth.md](auth.md) | 认证 / RBAC 三档 / 数据面接入 / 下游身份透传契约 |

> 认证（auth）只存在于 Platform 层——System / Infra 层组件不做用户认证，只信任 `X-Axisml-User` 透传，故 `auth.md` 归本层。

## 边界

- **持有**：租户持久记录（`tenants` 表，tenant scope / K8s Namespace 映射 / 停用 / 硬删除）、四张定义、用户 / 角色 / 会话。
- **不持有**：任何 K8s 资源 / CR、运行实例（Run / Service / Workspace）、制品版本、配额折算与 namespace 解析——全部下沉 System 层。

产品需求见 [PRD](product_design/prd.md)，页面设计以 [交互原型 prototype/](product_design/prototype) 为准，系统全景见 [high_level_design.md](../../docs/system_design/high_level_design.md)。
