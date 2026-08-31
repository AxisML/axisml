export default {
  pools: {
    title: "Resource Pools",
    subtitle:
      "Schedule and manage compute and storage resource pools centrally to improve utilization and workload stability. Allocation, monitoring and isolation keep multi-workload scenarios running efficiently.",
    newPool: "New Pool",
    searchPlaceholder: "Search by keyword",
    // list columns
    colName: "Name",
    colDesc: "Description",
    colSelector: "Node Selector",
    colUnits: "Units",
    colCreated: "Created",
    total: "{{count}} pools",
    noSelector: "None",
    more: "+{{count}}",
    // row actions
    manage: "Manage",
    // delete pool
    deleteTitle: "Delete pool {{name}}?",
    deleteDesc: "Resource units inside the pool are cascade-deleted with it. This cannot be undone.",
    deleteInfo: "Deletion is blocked while active jobs / services reference this pool. Clear active workloads first.",
    deleted: "Pool deleted",
    // create pool drawer
    drawerNew: "New Resource Pool",
    createPool: "Create Pool",
    created2: "Pool created, add resource units next",
    // manage pool drawer
    saved: "Pool saved",
    // sections
    fsBasic: "Basic Info",
    fsSchedule: "Capacity & Node Scheduling",
    fsUnits: "Resource Units",
    fName: "Name",
    fNamePlaceholder: "gpu-a100",
    fNameHelp: "Lowercase letters, digits, hyphens; immutable after creation",
    fDesc: "Description",
    fDescPlaceholder: "Purpose (optional)",
    // node selector (chip editor)
    fSelector: "Node Selector (K=V)",
    selectorKey: "key",
    selectorValue: "value",
    selectorAdd: "Add",
    selectorEmpty: "No node selector set",
    // capacity override
    fCapacity: "Override calculated capacity",
    fCapacityHelp:
      "When disabled, Kubernetes derives capacity from matching nodes and Standalone uses host inventory.",
    capacityResource: "resource, e.g. cpu",
    capacityQuantity: "quantity, e.g. 64 or 256Gi",
    // units grid
    unitsEmpty: "No resource units yet — add one below",
    newUnit: "New Unit",
    // unit form drawer
    unitDrawerNew: "New Resource Unit",
    unitDrawerEdit: "Edit Resource Unit",
    createUnit: "Create Unit",
    unitCreated: "Resource unit created",
    unitSaved: "Resource unit saved",
    unitDeleted: "Resource unit deleted",
    unitDeleteTitle: "Delete unit {{name}}?",
    unitDeleteDesc: "The spec cannot be recovered; workloads referencing it can no longer request it.",
    // unit basics
    uName: "Name",
    uNamePlaceholder: "a100-1x-large",
    uDesc: "Description",
    uDescPlaceholder: "Spec purpose (optional), e.g. single-GPU training / 4-GPU distributed",
    // resource spec matrix
    fsSpec: "Resource Spec",
    lockLabel: "Keep limits equal to requests",
    uReq: "requests",
    uLim: "limits",
    uCpu: "CPU",
    uMem: "Memory",
    uGpu: "GPU",
    uCpuUnit: "cores",
    uMemUnit: "GiB",
    uGpuUnit: "cards",
    reqEqLim: "requests = limits",
    // unit scheduling
    uSelector: "Extra Node Selector (K=V)",
  },
};
