# AxisML Platform Frontend 设计

平台唯一的用户界面：单页应用（SPA），消费 Platform [后端](backend.md) 的 REST API，不直接访问任何 System / Infra 层服务。页面级布局、字段与交互的权威是 [交互原型 prototype/](../product_design/prototype)；视觉皮肤（配色 / 字体 / 圆角 / 间距 / 卡片）的权威是仓库根目录的 [DESIGN.md](../../../DESIGN.md)（Geist 极简体系：近黑墨水 on 近白画布、hairline 卡片、蓝色 link / focus）。本文档只描述前端**工程架构**。

## 1. 技术栈

| 维度 | 选型 |
| --- | --- |
| 语言 / 框架 | TypeScript + React + Vite |
| 组件库 | shadcn/ui（源码内置的 Radix primitives）+ Tailwind CSS；图标 lucide-react |
| 图表 / 反馈 | Recharts（图表）、sonner（toast）、Radix 弹层（Dialog / Sheet / Popover / AlertDialog） |
| 数据获取 | 对后端 REST 的 typed client（`@hey-api` 从 [openapi/platform.yaml](../apis/platform.yaml) 生成）；TanStack Query 缓存 + 轮询 |
| i18n | react-i18next + dayjs |

设计 token 以 CSS 变量承载（语义令牌：`background` / `foreground` / `primary` / `muted` / `border` / `card` / `destructive` / `ring` 等），同时驱动 Tailwind 与 shadcn 组件；深浅主题由 `<html data-theme>` 切换。组件库不携带任何业务语义，主题与 locale 由前端集中注入，不依赖第三方组件库的 locale 包。

## 2. 信息架构与路由

侧栏五组（首页 / 训练中心 / 服务中心 / 资产中心 / 系统管理），路由与页面结构 1:1 对应 [交互原型 prototype/](../product_design/prototype)。租户内菜单的作用域由顶部"所属租户"切换器决定，切换为软切换（仅更新本地状态与 `axisml.tenant` Cookie，由 TanStack Query 在新租户作用域下重新拉取数据，不整页刷新）；系统管理为全集群、不受租户选择影响。

## 3. 数据获取与状态

- **后端调用**：所有数据来自 Platform 后端 REST（契约见 [openapi/platform.yaml](../apis/platform.yaml)）；前端不感知下游服务、不内嵌 PromQL、不解析 K8s namespace。
- **Active tenant**：顶部"所属租户"恒选中且仅选中一个租户（默认为用户的首个租户），当前租户由 `axisml.tenant` Cookie 携带（后端 Cookie 优先、`X-Axisml-Tenant` 头兜底）；`system-admin` 可逐个切换到任意租户。
- **实时性**：日志 / 事件用 SSE（`follow=true`）；运行态（phase / 配额用量 / digest）始终实时回源，前端不缓存为权威。Dashboard 待后续专项设计。
- **偏好持久化**：列表 / 卡片视图切换、语言等偏好存 localStorage。

## 4. 鉴权与数据面接入

- **控制面**：登录获得主 JWT（`aud=axisml-platform`），随后端请求携带；失效走 `/auth/refresh`。RBAC 由后端裁剪，前端按返回的能力 / 角色置灰不可用入口。
- **数据面**（工作区 / TensorBoard）：Cookie JWT + Envoy SecurityPolicy 是目标方案；当前 SecurityPolicy 派生未交付，后端不提供 access 端点，相关外部入口保持 fail-closed。在线服务数据面鉴权（API KEY）同样规划中。详见 [auth.md §5](auth.md#5-数据面接入)。

## 5. 多语言 / i18n

多语言是前端职责，后端与下游一律 locale-neutral（只返稳定机读标识）。前端按当前 locale 把机读标识映射为本地化文案：

| 后端 / 下游产出 | 前端本地化方式 |
| --- | --- |
| RFC 7807 problem 的 `type` URI + 下游 error code | 作文案映射 key（缺映射回退 `title`）；`title` / `detail` 仅调试兜底 |
| 机读枚举（`phase` / `status` / `source` / `kind` / `role` …） | 展示层映射为本地化标签；枚举原值不翻译 |
| 时间戳（RFC3339 UTC）/ 数值 | 按 locale 格式化（`Intl` / dayjs：日期、相对时间、时区、千分位） |

- 首批 `zh-CN` / `en-US`；新增语言 = 加 message catalog（前端自带所有文案，无需组件库 locale 包），后端零改动。
- 语言选择**纯浏览器端持久化**（localStorage，初值 `navigator.language`），不回传后端、不入会话。
- **不本地化**：用户自由文本（显示名 / 描述 / 标签 / 制品元数据）与日志正文原样展示。

i18n 后端契约见 [backend.md §5.6](backend.md#56-多语言--i18n)。

## 6. 错误与降级展示

- RFC 7807 problem → 按 `type` 取本地化文案；表单字段错误就地高亮。
- 指标 / 容量为 `null` 或查询失败 → gauge 显示 `—` + hover「指标同步中」/ 图区占位，不影响其余区块。
