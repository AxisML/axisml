export default {
  traffic: {
    title: "Traffic",
    subtitle:
      "Orchestrate multi-version traffic for online services: weighted splits, canary rollouts and blue-green cutovers. One stable endpoint distributes by weight across backend services.",
    newPolicy: "New policy",
    searchPlaceholder: "Search by name",
    modeAll: "Mode: All",
    statusAll: "Status: All",
    total: "{{count}} policies",

    // columns
    colName: "Name",
    colMode: "Mode",
    colStatus: "Status",
    colBackends: "Backends (split)",
    colEndpoint: "Endpoint",

    // modes
    modeWeighted: "Weighted",
    modeCanary: "Canary",
    modeWeightedDesc: "N backends split by weight, Σ=100",
    modeCanaryDesc: "1 stable + 1 canary, ramp by percentage",

    // row actions
    actSplitCanary: "Adjust ratio",
    actSplitWeighted: "Adjust weights",
    actPromote: "Promote to full",
    actRollback: "Roll back",

    // delete / promote / rollback confirms
    deleteTitle: "Delete traffic policy {{name}}?",
    deleteDesc: "The endpoint stops distributing traffic and this cannot be undone.",
    deleted: "Traffic policy deleted",
    promoteTitle: "Promote the canary backend of {{name}} to full?",
    promoteDesc: "The canary backend takes 100% of traffic; the stable backend retires.",
    promoteOk: "Confirm promote",
    promoted: "Canary backend promoted to full",
    rollbackTitle: "Roll back {{name}} to the stable backend?",
    rollbackDesc: "The canary backend stops receiving traffic; the stable backend resumes full.",
    rollbackOk: "Confirm rollback",
    rolledBack: "Rolled back to stable backend",

    // create drawer
    drawerNew: "New traffic policy",
    drawerNewSub: "Bind a stable endpoint and distribute traffic to this tenant's online services",
    created: "Traffic policy created",
    createPolicy: "Create policy",
    fsBasic: "Basics & mode",
    fsEndpoint: "Endpoint",
    fsBackendCanary: "Backends (canary)",
    fsBackendWeighted: "Backends (weighted)",
    fName: "Name",
    fNamePlaceholder: "rt-chat",
    fDesc: "Description",
    fDescPlaceholder: "Purpose (optional)",
    fMode: "Mode",
    fPath: "Path",
    fPathPlaceholder: "Leave blank to auto-generate /services/<tenant>/rt-chat/",
    fStable: "Stable backend",
    fCanary: "Canary backend",
    fCanaryPercent: "Initial canary percent",
    fCanaryHelp:
      "1 stable backend + 1 canary backend, ramped by percentage. The dropdown lists only Ready services in the current tenant.",
    fBackendWeights: "Backends & weights",
    fWeightHelp:
      "N backends split by weight; weights must sum to 100. The dropdown lists only Ready services in the current tenant.",
    addBackend: "Add backend",
    pickService: "Select a backend service…",
    serviceReady: "{{name}} (Ready)",
    weightPlaceholder: "Weight 0–100",

    // split drawer
    drawerSplitCanary: "Adjust the canary backend ramp percentage",
    drawerSplitWeighted: "Adjust per-backend traffic weights (Σ=100)",
    splitApply: "Apply",
    splitApplied: "Traffic split updated",
    fsCanaryPercent: "Canary percent",
    fCanaryPercentLabel: "Canary backend ramp percent",
    canaryPercentHelp: "Percentage of traffic the canary backend receives; the rest goes to the stable backend.",
    canarySplitHint: "Canary {{canary}}% · Stable {{stable}}%",
    fsBackendWeight: "Backend weights",

    // detail
    backToList: "Back to traffic policies",
    loadFailedTitle: "Failed to load traffic policy",
    delete: "Delete",
    detailDeleteDesc: "Removes the weighted route; traffic falls back to the default gateway. This cannot be undone.",

    // detail tabs
    tabOverview: "Overview",
    tabDistribution: "Traffic config",
    tabEvents: "Events",
    policyInfo: "Policy info",

    // overview fields
    fieldName: "Name",
    fieldDesc: "Description",
    fieldMode: "Mode",
    fieldEndpoint: "Endpoint",
    fieldBackendCount: "Backends",
    fieldOwner: "Owner",
    fieldCreatedAt: "Created at",
    copyEndpoint: "Copy endpoint",
    endpointCopied: "Endpoint copied",

    // distribution (canary)
    canaryPercentTitle: "Canary percent",
    stableShare: "Stable",
    canaryShare: "Canary",
    promoteToStable: "Promote to stable",
    applyCanary: "Apply",

    // backend distribution table
    backendDist: "Backend distribution",
    colService: "Online service",
    colRole: "Role",
    colTargetWeight: "Target weight",
    colActualPct: "Actual share",
    colBackendStatus: "Backend status",
    roleStable: "Stable",
    roleCanary: "Canary",
    roleMember: "Member",
    backendReady: "Ready",
    backendNotReady: "Not ready",

    // distribution (weighted)
    weightedHint: "Edit target weights directly with live Σ=100 validation",
    sumOk: "Σ = {{sum}}% ✓",
    sumBad: "Σ = {{sum}}% ✕",
    applyWeights: "Apply weights",
    weightsApplied: "Weights applied",

    // events
    noEvents: "No events",
  },
};
