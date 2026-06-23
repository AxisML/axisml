export default {
  traffic: {
    title: "流量配置",
    subtitle:
      "为在线服务编排多版本流量：加权切分、灰度放量与蓝绿式全量切换。一个对外入口按权重分发到多个后端服务。",
    newPolicy: "新建策略",
    searchPlaceholder: "名称搜索",
    modeAll: "模式：全部",
    statusAll: "状态：全部",
    total: "共 {{count}} 条策略",

    // 列
    colName: "名称",
    colMode: "模式",
    colStatus: "状态",
    colBackends: "后端（流量分布）",
    colEndpoint: "访问地址",

    // 模式
    modeWeighted: "加权",
    modeCanary: "灰度",
    modeWeightedDesc: "N 个后端按权重切分，Σ=100",
    modeCanaryDesc: "1 稳定 + 1 灰度，按百分比逐步放量",

    // 行内操作
    actSplitCanary: "调整比例",
    actSplitWeighted: "调整权重",
    actPromote: "提升为全量",
    actRollback: "回滚",

    // 删除 / 提升 / 回滚 确认
    deleteTitle: "确定删除流量策略 {{name}}？",
    deleteDesc: "删除后对外入口将停止分发流量，且不可恢复。",
    deleted: "流量策略已删除",
    promoteTitle: "提升 {{name}} 的灰度后端为全量？",
    promoteDesc: "灰度后端将接管 100% 流量，稳定后端退出。",
    promoteOk: "确认提升",
    promoted: "已提升灰度后端为全量",
    rollbackTitle: "回滚 {{name}} 至稳定后端？",
    rollbackDesc: "灰度后端将停止接收流量，稳定后端恢复全量。",
    rollbackOk: "确认回滚",
    rolledBack: "已回滚至稳定后端",

    // 创建抽屉
    drawerNew: "新建流量策略",
    created: "流量策略已创建",
    createPolicy: "创建策略",
    fsBasic: "基本信息与模式",
    fsEndpoint: "对外入口",
    fsBackendCanary: "后端服务（灰度）",
    fsBackendWeighted: "后端服务（加权）",
    fName: "名称",
    fNamePlaceholder: "rt-chat",
    fDesc: "描述",
    fDescPlaceholder: "用途说明（可选）",
    fMode: "模式",
    fPath: "Path",
    fPathPlaceholder: "留空自动生成 /services/<tenant>/rt-chat/",
    fStable: "稳定后端",
    fCanary: "灰度后端",
    fCanaryPercent: "初始灰度百分比",
    fCanaryHelp: "1 个稳定后端 + 1 个灰度后端，按百分比逐步放量。后端下拉只列当前租户 Ready 的服务。",
    fBackendWeights: "后端与权重",
    fWeightHelp: "N 个后端按权重切分，权重之和需为 100。后端下拉只列当前租户 Ready 的服务。",
    addBackend: "添加后端",
    pickService: "选择后端服务…",
    serviceReady: "{{name}}（Ready）",
    weightPlaceholder: "权重 0–100",

    // 切流抽屉
    drawerSplitCanary: "调整灰度后端的放量百分比",
    drawerSplitWeighted: "调整各后端服务的流量权重（Σ=100）",
    splitApply: "应用",
    splitApplied: "流量分布已更新",
    fsCanaryPercent: "灰度百分比",
    fCanaryPercentLabel: "灰度后端放量百分比",
    canaryPercentHelp: "灰度后端接收的流量百分比，剩余流量由稳定后端承接。",
    canarySplitHint: "灰度 {{canary}}% · 稳定 {{stable}}%",
    fsBackendWeight: "后端权重",

    // 详情
    backToList: "返回流量策略列表",
    loadFailedTitle: "无法加载流量策略",
    delete: "删除",
    detailDeleteDesc: "将移除加权路由，流量回落到默认网关。该操作不可恢复。",

    // 详情标签页
    tabOverview: "概览",
    tabDistribution: "流量配置",
    tabMonitor: "监控",
    tabEvents: "事件",
    policyInfo: "策略信息",

    // 概览字段
    fieldName: "名称",
    fieldDesc: "描述",
    fieldMode: "模式",
    fieldEndpoint: "对外入口",
    fieldBackendCount: "后端数",
    fieldOwner: "创建人",
    fieldCreatedAt: "创建时间",
    copyEndpoint: "复制入口地址",
    endpointCopied: "入口地址已复制",

    // 流量配置（灰度）
    canaryPercentTitle: "灰度百分比",
    canaryCurrent: "当前",
    canaryPending: "待应用",
    canaryPresets: "放量预设",
    stableShare: "稳定",
    canaryShare: "灰度",
    promoteToStable: "提升为稳定",
    applyCanary: "应用",
    rollback: "回滚",

    // 流量分布表
    backendDist: "流量分布",
    colService: "在线服务",
    colRole: "角色",
    colTargetWeight: "目标权重",
    colActualPct: "实际流量占比",
    colBackendStatus: "后端状态",
    roleStable: "稳定",
    roleCanary: "灰度",
    roleMember: "成员",
    backendReady: "就绪",
    backendNotReady: "未就绪",

    // 流量配置（加权）
    weightedHint: "直接编辑目标权重，实时 Σ=100 校验",
    sumOk: "Σ = {{sum}}% ✓",
    sumBad: "Σ = {{sum}}% ✕",
    applyWeights: "应用权重",
    weightsApplied: "权重已应用",

    // 监控
    monGrouped: "来自 compute 指标代理 · 按后端分组",
    monQps: "QPS",
    monLatency: "延迟 p95",
    monErrorRate: "错误率 (5xx)",
    monitorEmpty: "暂无监控数据",
    monitorNoBackends: "策略尚无后端，暂无监控数据。",

    // 事件
    noEvents: "暂无事件",
  },
};
