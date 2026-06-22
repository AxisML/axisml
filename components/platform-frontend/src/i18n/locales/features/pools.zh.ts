export default {
  pools: {
    title: "资源池管理",
    subtitle:
      "统一调度和管理计算、存储等资源池，提升资源利用率与任务运行稳定性。支持资源分配、监控与隔离，保障多业务场景下的高效运行。",
    newPool: "新建资源池",
    searchPlaceholder: "关键字搜索",
    // list columns
    colName: "名称",
    colDesc: "描述",
    colSelector: "节点选择器",
    colUnits: "资源单元",
    colCreated: "创建时间",
    total: "共 {{count}} 个资源池",
    noSelector: "无",
    more: "+{{count}}",
    // row actions
    manage: "管理",
    // delete pool
    deleteTitle: "删除资源池 {{name}}？",
    deleteDesc: "删除后池内资源单元将随资源池一并级联删除，且不可恢复。",
    deleteInfo: "若有活跃任务 / 服务正在引用本池，删除会被阻断，请先清空活跃负载后重试。",
    deleted: "资源池已删除",
    // create pool drawer
    drawerNew: "新建资源池",
    createPool: "创建资源池",
    created2: "资源池已创建，请添加资源单元",
    // manage pool drawer
    saved: "资源池已保存",
    // sections
    fsBasic: "基本信息",
    fsSchedule: "节点调度",
    fsUnits: "资源单元",
    fName: "名称",
    fNamePlaceholder: "gpu-a100",
    fNameHelp: "小写字母、数字、连字符；创建后不可修改",
    fDesc: "描述",
    fDescPlaceholder: "用途说明（可选）",
    // node selector (chip editor)
    fSelector: "节点选择器（K=V）",
    selectorKey: "键",
    selectorValue: "值",
    selectorAdd: "添加",
    selectorEmpty: "未设置节点选择器",
    // tolerations
    fTolerations: "容忍配置（tolerations）",
    tolKey: "key",
    tolOp: "operator",
    tolVal: "value",
    tolEffect: "effect",
    tolKeyPlaceholder: "污点键，如 nvidia.com/gpu",
    tolValPlaceholder: "如 true",
    addToleration: "添加容忍",
    noTolerations: "暂无容忍配置",
    // units grid
    unitsEmpty: "暂无资源单元，点击下方按钮添加",
    newUnit: "新建资源单元",
    // unit form drawer
    unitDrawerNew: "新建资源单元",
    unitDrawerEdit: "编辑资源单元",
    createUnit: "创建资源单元",
    unitCreated: "资源单元已创建",
    unitSaved: "资源单元已保存",
    unitDeleted: "资源单元已删除",
    unitDeleteTitle: "删除资源单元 {{name}}？",
    unitDeleteDesc: "删除后该规格不可恢复，引用该规格的任务将无法再申请。",
    // unit basics
    uName: "名称",
    uNamePlaceholder: "a100-1x-large",
    uDesc: "描述",
    uDescPlaceholder: "规格用途说明（可选），如：单卡训练 / 4 卡分布式",
    // resource spec matrix
    fsSpec: "资源规格",
    lockLabel: "limits 与 requests 保持一致",
    uReq: "requests",
    uLim: "limits",
    uCpu: "CPU",
    uMem: "内存",
    uGpu: "GPU",
    uCpuUnit: "核",
    uMemUnit: "GiB",
    uGpuUnit: "卡",
    reqEqLim: "requests = limits",
    // unit scheduling
    uSelector: "额外节点选择器（K=V）",
  },
};
