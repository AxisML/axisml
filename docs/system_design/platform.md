# AxisML Platform 详细设计

AxisML Platform 是平台的用户入口与业务编排层，由 **前端（TypeScript + React）** 与 **后端（Go）** 两个子组件组成。它对外承载用户视图（登录、租户切换、工作区视图、任务 / 服务 / 制品的 UI），对内把用户操作拆解为对 [cluster-manager](cluster-manager.md)、[compute](compute.md)、[artifacts](artifacts.md) 三个内部服务的协作调用。
