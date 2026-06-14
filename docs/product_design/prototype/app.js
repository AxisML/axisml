/* =====================================================================
   AxisML console — app controller
   Hash routing, data-driven list rendering (list/card), SVG charts,
   artifact detail, create/upload drawers, and shell interactions.
   ===================================================================== */
(function () {
  'use strict';

  /* ---------------- Route config ---------------- */
  const NAV = {
    home:       { page: 'home',       crumbs: ['首页'],                  nav: 'home' },
    workspaces: { page: 'workspaces', crumbs: ['训练中心', '工作区'], nav: 'workspaces' },
    experiments:{ page: 'experiments', crumbs: ['训练中心', '实验管理'], nav: 'experiments' },
    evaluations:{ page: 'evaluations', crumbs: ['训练中心', '评估任务'], nav: 'evaluations' },
    jobs:       { page: 'jobs',       crumbs: ['训练中心', '自定义任务'], nav: 'jobs' },
    services:   { page: 'services',   crumbs: ['服务中心', '在线服务'], nav: 'services' },
    traffic:    { page: 'traffic',    crumbs: ['服务中心', '流量策略'], nav: 'traffic' },
    datasets:   { page: 'datasets',   crumbs: ['资产中心', '数据集'],    nav: 'datasets' },
    models:     { page: 'models',     crumbs: ['资产中心', '模型'],      nav: 'models' },
    images:     { page: 'images',     crumbs: ['资产中心', '镜像'],      nav: 'images' },
    tenants:    { page: 'tenants',    crumbs: ['系统管理', '租户管理'],  nav: 'tenants' },
    pools:      { page: 'pools',      crumbs: ['系统管理', '资源池管理'], nav: 'pools' },
  };
  const DETAIL = {
    workspaces: { page: 'workspace-detail', group: '训练中心', parent: '工作区',     nav: 'workspaces' },
    jobs:       { page: 'job-detail',       group: '训练中心', parent: '自定义任务',   nav: 'jobs' },
    services:   { page: 'service-detail',   group: '服务中心', parent: '在线服务',   nav: 'services' },
    traffic:    { page: 'traffic-detail',    group: '服务中心', parent: '流量策略',   nav: 'traffic' },
    datasets:   { page: 'artifact-detail',  group: '资产中心',   parent: '数据集',     nav: 'datasets' },
    models:     { page: 'artifact-detail',  group: '资产中心',   parent: '模型',       nav: 'models' },
    images:     { page: 'artifact-detail',  group: '资产中心',   parent: '镜像',       nav: 'images' },
    tenants:    { page: 'tenant-detail',    group: '系统管理',   parent: '租户管理',   nav: 'tenants' },
    pools:      { page: 'pool-detail',      group: '系统管理',   parent: '资源池管理', nav: 'pools' },
  };

  /* ---------------- Mock data ---------------- */
  const T = {
    success: { tone: 'success', label: '' }, // label set per use
  };
  const st = (tone, label) => ({ tone, label });

  const DATA = {
    workspaces: [
      { name: 'ws-dev-zhang',     status: st('success', '运行中'), unit: 'cpu-medium/1x', image: 'jupyter-ds:2024.3',   rep: '1/1', creator: '张伟' },
      { name: 'ws-train-li',      status: st('warn',    '启动中'), unit: 'gpu-a100/1x',   image: 'pytorch:2.3-cu121',  rep: '0/1', creator: '李娜' },
      { name: 'ws-notebook-chen', status: st('success', '运行中'), unit: 'gpu-l40s/1x',   image: 'jupyter-tf:2.16',    rep: '1/1', creator: '陈曦' },
      { name: 'ws-eval-wang',     status: st('muted',   '已停止'), unit: 'cpu-medium/1x', image: 'vscode-server:1.89', rep: '0/0', creator: '王磊' },
    ],
    // 计算任务采用 Job(可复用模板)→ Run(每次运行)两级模型。
    // 每个 Job 是模板定义;runs[] 是从该模板触发的运行,Run 名 = <job>-<n>。
    // 列表的「最近运行状态 / 运行数」由 runs[] 实时派生(见 jobLatest / renderList)。
    jobs: [
      {
        name: 'train-llm-7b', display: 'LLaMA-7B 全参微调', desc: '对话场景 SFT，3 epoch，bf16 混合精度',
        creator: '张伟', creatorId: 'zhang.wei', created: '2026-06-01 09:30:11', updated: '2 天前',
        tpl: {
          pool: 'gpu-a100', unit: 'a100-4x-xlarge', spec: 'cpu=32 mem=256Gi gpu=4',
          image: 'pytorch-train@sha256:9f8e7d', replicas: 4,
          cmd: ['torchrun', '--nproc_per_node=4', 'train_sft.py', '--lr=2e-5', '--epochs=3'],
          env: [['HF_HOME', '/data/hf'], ['NCCL_DEBUG', 'INFO']],
          policy: { timeout: '86400s', backoff: '0', ttl: '172800s' },
          artifacts: [['镜像', 'pytorch-train', '2.3-cu121'], ['模型', 'llama-7b-base', 'v1'], ['数据集', 'dialog-zh-clean', 'v4']],
        },
        runs: [
          { n: 12, status: st('success', '运行中'), unit: 'gpu-a100/4x', spec: 'cpu=32 mem=256Gi gpu=4', rep: '4', trigger: '张伟', dur: '02:14:30', started: '2026-06-13 06:21:08', finished: '—', msg: '4/4 Pod 运行中，step 12400 / 36000', override: '资源单元 gpu-a100/4x · lr=2e-5' },
          { n: 11, status: st('success', '成功'), unit: 'gpu-a100/4x', spec: 'cpu=32 mem=256Gi gpu=4', rep: '4', trigger: '张伟', dur: '03:40:02', started: '2026-06-11 22:10:30', finished: '2026-06-12 01:50:32', msg: '全部成功，产出已注册 llama-7b-sft@v3' },
          { n: 10, status: st('danger', '失败'), unit: 'gpu-a100/8x', spec: 'cpu=64 mem=512Gi gpu=8', rep: '8', trigger: '李娜', dur: '00:08:22', started: '2026-06-10 14:02:08', finished: '2026-06-10 14:10:30', msg: 'rank3 CUDA OOM，超出重试上限 backoffLimit=0', override: '资源单元 gpu-a100/8x（本次覆盖）' },
          { n: 9, status: st('muted', '已取消'), unit: 'gpu-a100/4x', spec: 'cpu=32 mem=256Gi gpu=4', rep: '4', trigger: '张伟', dur: '00:31:14', started: '2026-06-08 10:00:00', finished: '2026-06-08 10:31:14', msg: '用户主动取消' },
        ],
      },
      {
        name: 'eval-recall', display: '召回评测', desc: '在 recall-eval-set 上跑离线召回指标',
        creator: '李娜', creatorId: 'li.na', created: '2026-06-05 16:20:00', updated: '5 天前',
        tpl: {
          pool: 'gpu-h100', unit: 'h100-1x-large', spec: 'cpu=16 mem=128Gi gpu=1',
          image: 'triton-infer@sha256:24a05b', replicas: 1,
          cmd: ['python', 'eval_recall.py', '--topk=50'],
          env: [['TZ', 'Asia/Shanghai']],
          policy: { timeout: '7200s', backoff: '1', ttl: '86400s' },
          artifacts: [['镜像', 'triton-infer', '24.05'], ['模型', 'bge-embed', '1.5.0'], ['数据集', 'recall-eval-set', 'v2']],
        },
        runs: [
          { n: 3, status: st('success', '成功'), unit: 'gpu-h100/1x', spec: 'cpu=16 mem=128Gi gpu=1', rep: '1', trigger: '李娜', dur: '00:18:40', started: '2026-06-09 11:00:00', finished: '2026-06-09 11:18:40', msg: 'recall@50 = 0.873' },
          { n: 2, status: st('success', '成功'), unit: 'gpu-h100/1x', spec: 'cpu=16 mem=128Gi gpu=1', rep: '1', trigger: '李娜', dur: '00:17:55', started: '2026-06-07 09:30:00', finished: '2026-06-07 09:47:55', msg: 'recall@50 = 0.861' },
          { n: 1, status: st('success', '成功'), unit: 'gpu-h100/1x', spec: 'cpu=16 mem=128Gi gpu=1', rep: '1', trigger: '王磊', dur: '00:19:02', started: '2026-06-05 16:25:00', finished: '2026-06-05 16:44:02', msg: 'recall@50 = 0.852' },
        ],
      },
      {
        name: 'sft-baseline', display: 'SFT 基线', desc: '小规模 SFT 基线，单卡调参用',
        creator: '王磊', creatorId: 'wang.lei', created: '2026-06-14 12:30:00', updated: '1 小时前',
        tpl: {
          pool: 'gpu-a100', unit: 'a100-1x-large', spec: 'cpu=8 mem=64Gi gpu=1',
          image: 'pytorch-train@sha256:9f8e7d', replicas: 1,
          cmd: ['python', 'train_sft.py', '--lr=1e-5'],
          env: [],
          policy: { timeout: '43200s', backoff: '0', ttl: '172800s' },
          artifacts: [['镜像', 'pytorch-train', '2.3-cu121'], ['数据集', 'dialog-zh-clean', 'v4']],
        },
        runs: [],
      },
    ],
    services: [
      { name: 'svc-chat-api', status: st('success', '就绪'),   unit: 'gpu-h100/1x',  rep: '2/2', url: '/services/team-a/chat-api/' },
      { name: 'svc-embed',    status: st('warn',    '降级'),   unit: 'gpu-l40s/1x',  rep: '1/2', url: '/services/team-a/embed/' },
      { name: 'svc-rerank',   status: st('muted',   '已停止'), unit: 'cpu-large/1x', rep: '0/0', url: '—' },
    ],
    traffic: [
      {
        name: 'rt-chat', mode: '灰度', status: st('warn', '灰度中'), url: '/services/team-a/chat/',
        display: '对话服务灰度发布', desc: '将稳定版 chat-v1 的流量逐步切到 chat-v2，按灰度百分比放量',
        auth: 'jwt', creator: '张伟', created: '2026-06-10 15:42:08', canary: 10,
        backends: [
          { svc: 'svc-chat-v1', role: '稳定', weight: 90, phase: st('success', '就绪') },
          { svc: 'svc-chat-v2', role: '灰度', weight: 10, phase: st('success', '就绪') },
        ],
      },
      {
        name: 'rt-embed', mode: '加权', status: st('success', '生效中'), url: '/services/team-a/embed/',
        display: '向量服务加权切分', desc: '两个等价后端平摊流量，用于负载分担与可用性冗余',
        auth: 'apiKey', creator: '李娜', created: '2026-05-30 09:18:55',
        backends: [
          { svc: 'svc-embed-a', role: '成员', weight: 50, phase: st('success', '就绪') },
          { svc: 'svc-embed-b', role: '成员', weight: 50, phase: st('success', '就绪') },
        ],
      },
      {
        name: 'rt-rerank', mode: '灰度', status: st('muted', '未就绪'), url: '—',
        display: '重排序服务灰度', desc: '灰度后端已选，稳定后端缺失，策略尚未就绪',
        auth: 'none', creator: '王磊', created: '2026-06-12 11:03:20', canary: 0,
        backends: [
          { svc: 'svc-rerank-v2', role: '灰度', weight: 0, phase: st('warn', '启动中') },
          { svc: '稳定后端缺失', role: '稳定', weight: 0, phase: st('danger', '缺失'), missing: true },
        ],
      },
    ],
    datasets: [
      { name: 'dialog-zh-clean', spec: 'jsonl',   latest: 'v4',  versions: 4, vis: 'tenant', updated: '1 天前' },
      { name: 'imagenet-mini',   spec: 'parquet', latest: '1.0', versions: 1, vis: 'public', updated: '2 周前' },
      { name: 'recall-eval-set', spec: 'parquet', latest: 'v2',  versions: 2, vis: 'tenant', updated: '4 天前' },
    ],
    models: [
      { name: 'llama-7b-sft', spec: 'pytorch',     latest: 'v3',      versions: 3, vis: 'tenant', updated: '2 天前' },
      { name: 'bge-embed',    spec: 'safetensors', latest: '1.5.0',   versions: 5, vis: 'public', updated: '1 周前' },
      { name: 'resnet-cls',   spec: 'onnx',        latest: '2024-06', versions: 2, vis: 'tenant', updated: '3 天前' },
      { name: 'qwen2-vl-ft',  spec: 'pytorch',     latest: 'v2',      versions: 2, vis: 'tenant', updated: '5 小时前' },
    ],
    images: [
      { name: 'pytorch-train', spec: 'training',  latest: '2.3-cu121', versions: 6, vis: 'tenant', updated: '1 天前' },
      { name: 'triton-infer',  spec: 'inference', latest: '24.05',     versions: 3, vis: 'public', updated: '1 周前' },
      { name: 'jupyter-ds',    spec: 'dev',       latest: '2024.3',    versions: 8, vis: 'tenant', updated: '3 天前' },
    ],
  };

  const ART_META = {
    datasets: { label: '格式', display: '数据集', plural: 'datasets', pull: (n, v) => `aws s3 cp s3://axisml-datasets/team-a/${n}/${v}/ ./${n} --recursive`, sizeBase: '4.2GB' },
    models:   { label: '框架', display: '模型',   plural: 'models',   pull: (n, v) => `docker pull registry.axisml.io/team-a/${n}:${v}`, sizeBase: '13.4GB' },
    images:   { label: '用途', display: '镜像',   plural: 'images',   pull: (n, v) => `docker pull registry.axisml.io/team-a/${n}:${v}`, sizeBase: '6.1GB' },
  };
  const ART_DISPLAY = {
    'llama-7b-sft': 'LLaMA-7B 监督微调权重', 'bge-embed': 'BGE 文本向量模型', 'resnet-cls': 'ResNet 图像分类', 'qwen2-vl-ft': 'Qwen2-VL 视觉指令微调',
    'dialog-zh-clean': '中文对话清洗集', 'imagenet-mini': 'ImageNet 子集', 'recall-eval-set': '召回评测集',
    'pytorch-train': 'PyTorch 训练镜像', 'triton-infer': 'Triton 推理镜像', 'jupyter-ds': 'Jupyter 数据科学镜像',
  };

  /* Artifact category → logo glyph (simple geometric) for ArtifactHub-style cards */
  const ART_CAT = {
    datasets: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linejoin="round"><rect x="3.5" y="4.5" width="17" height="15" rx="2.2"/><path d="M3.5 9.6h17M3.5 14.5h17M9.2 9.6v9.9"/></svg>',
    models:   '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"><circle cx="6" cy="6.6" r="2.3"/><circle cx="18" cy="6.6" r="2.3"/><circle cx="12" cy="17.4" r="2.3"/><path d="M8 7.4 10.8 15.4M16 7.4 13.2 15.4M8.3 6.6h7.4"/></svg>',
    images:   '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linejoin="round"><path d="M12 3.2 20 7.3v9.4L12 20.8 4 16.7V7.3z"/><path d="M4 7.3 12 11.5l8-4.2M12 11.5v9.3"/></svg>',
  };
  /* Inline footer / badge icons */
  const AH_ICO = {
    repo:   '<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linejoin="round"><path d="M2 4.4 8 2l6 2.4L8 6.8z"/><path d="M2 8l6 2.4L14 8M2 11.6 8 14l6-2.4"/></svg>',
    check:  '<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="8" cy="8" r="6.2"/><path d="M5.4 8.2 7.2 10l3.4-3.9"/></svg>',
    lock:   '<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linejoin="round"><rect x="3.4" y="7" width="9.2" height="6.2" rx="1.4"/><path d="M5.4 7V5.4a2.6 2.6 0 0 1 5.2 0V7"/></svg>',
    tag:    '<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linejoin="round"><path d="M2.6 2.6h5L13.4 8.4 8.4 13.4 2.6 7.6z"/><circle cx="5" cy="5" r="0.9" fill="currentColor" stroke="none"/></svg>',
    layers: '<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linejoin="round"><path d="M8 2 14 5 8 8 2 5z"/><path d="M2 8l6 3 6-3M2 11l6 3 6-3"/></svg>',
  };

  const UNIT_SPECS = {
    'cpu-medium-1x': 'cpu=4 mem=16Gi', 'cpu-large-1x': 'cpu=16 mem=128Gi',
    'a100-1x-large': 'cpu=8 mem=64Gi gpu=1', 'a100-4x-xlarge': 'cpu=32 mem=256Gi gpu=4',
    'a100-8x-xlarge-ib': 'cpu=64 mem=512Gi gpu=8', 'h100-1x-large': 'cpu=16 mem=128Gi gpu=1',
    'l40s-1x': 'cpu=8 mem=48Gi gpu=1',
  };

  /* ---------------- Small helpers ---------------- */
  const $ = (s, r = document) => r.querySelector(s);
  const $$ = (s, r = document) => Array.from(r.querySelectorAll(s));
  const pill = (s) => `<span class="pill pill-${s.tone}">${s.label}</span>`;
  const globeIco = `<svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><circle cx="8" cy="8" r="6"/><path d="M2 8h12M8 2c1.7 1.6 2.7 3.8 2.7 6S9.7 12.4 8 14C6.3 12.4 5.3 10.2 5.3 8S6.3 3.6 8 2z"/></svg>`;
  const pubMark = `<span class="vis-ico" title="公共资产（axisml-system）">${globeIco}</span>`;
  const visMark = (v) => v === 'public' ? pubMark : '';
  const wsLogo = (image) => {
    const base = String(image || '?').split(':')[0];
    const mono = (base.replace(/[^a-zA-Z]/g, '').slice(0, 2) || '?').toUpperCase();
    return `<span class="ws-logo">${mono}</span>`;
  };
  const visPill = (v) => visMark(v);
  const esc = (x) => String(x).replace(/&/g, '&amp;').replace(/</g, '&lt;');
  const actLink = (t, opts = '') => `<a class="row-action-link${opts}">${t}</a>`;
  // ---- workspace card footer icon actions ----
  const WS_ICO = {
    jupyter:  '<svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"><rect x="2.5" y="2.5" width="11" height="11" rx="1.5"/><path d="M5.6 2.5v11"/><path d="M7.7 5.6h3.3M7.7 8h3.3M7.7 10.4h2.1"/></svg>',
    vscode:   '<svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M10 4.5 13.5 8 10 11.5M6 4.5 2.5 8 6 11.5"/></svg>',
    terminal: '<svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="3" width="12" height="10" rx="1.5"/><path d="M4.8 6.4 6.7 8 4.8 9.6M8.6 10h2.6"/></svg>',
    stop:     '<svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M8 2v5.5"/><path d="M4.6 4.4a5 5 0 1 0 6.8 0"/></svg>',
    del:      '<svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"><path d="M2.5 4h11M6 4V2.6h4V4M4.8 4l.5 9h5.4l.5-9"/></svg>',
    start:    '<svg width="14" height="14" viewBox="0 0 16 16"><path d="M5 3.4 12.2 8 5 12.6z" fill="currentColor"/></svg>',
  };
  const wsAct = (cls, title, ico, label, attr = '') => `<a class="row-action-link ws-act${cls}" title="${title}"${attr}>${ico}<span class="vh">${label}</span></a>`;
  function wsFoot(i) {
    const stopped = i.status.label === '已停止';
    const left = stopped
      ? wsAct('', '启动', WS_ICO.start, '启动')
      : wsAct('', '打开 Jupyter', WS_ICO.jupyter, '打开', ' data-tool="Jupyter"')
        + wsAct('', '打开 VS Code', WS_ICO.vscode, '打开', ' data-tool="VS Code"')
        + wsAct('', '打开终端', WS_ICO.terminal, '打开', ' data-tool="终端"');
    const right = (stopped ? '' : wsAct(' ws-act-stop', '停止', WS_ICO.stop, '停止'))
      + wsAct(' ws-act-del', '删除', WS_ICO.del, '删除');
    return `<div class="ws-foot-l">${left}</div><div class="ws-foot-r">${right}</div>`;
  }
  const copyIco = (val) => `<span class="icon-copy" data-copy="${val}"><svg width="13" height="13" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><rect x="5" y="5" width="8" height="8" rx="1.5"/><path d="M3 11V3h8"/></svg></span>`;

  // mode pill (加权 / 灰度) + role tag
  const modePill = (m) => `<span class="pill pill-plain">${m}</span>`;
  const roleTag = (r) => {
    const cls = r === '稳定' ? 'role-stable' : r === '灰度' ? 'role-canary' : 'role-member';
    return `<span class="role-tag ${cls}">${r}</span>`;
  };
  // backend distribution mini-bars for the list cell
  function flowBars(item) {
    return `<div class="flow-list">` + item.backends.map((b) => {
      const fillCls = b.weight === 0 ? 'zero' : (b.role === '灰度' ? 'canary' : 'stable');
      const wt = b.missing ? `<span class="flow-wt muted">缺失</span>` : `<span class="flow-wt">${b.weight}</span>`;
      return `<div class="flow-row">
        <span class="flow-name${b.missing ? ' missing' : ''}">${b.svc}</span>
        <span class="flow-track"><span class="flow-fill ${fillCls}" style="width:${b.missing ? 0 : b.weight}%"></span></span>
        ${wt}</div>`;
    }).join('') + `</div>`;
  }

  // Job two-level helpers — latest-run status + run count derived from runs[]
  const NEVER_RUN = st('muted', '从未运行');
  function jobLatest(job) { return (job.runs && job.runs.length) ? job.runs[0].status : NEVER_RUN; }
  function findJob(name) { return (DATA.jobs || []).find((j) => j.name === name); }
  function findRun(job, runName) {
    if (!job) return null;
    const n = parseInt(String(runName).split('-').pop(), 10);
    return (job.runs || []).find((r) => r.n === n) || null;
  }
  const RUN_TERMINAL = ['成功', '失败', '已取消'];
  const cmdTags = (arr) => `<div class="tag-list">${(arr || []).map((c) => {
    const eq = c.indexOf('=');
    return eq > 0 ? `<span class="tag"><span class="k">${esc(c.slice(0, eq))}</span><span class="eq">=</span><span class="v">${esc(c.slice(eq + 1))}</span></span>`
      : `<span class="tag"><span class="v">${esc(c)}</span></span>`;
  }).join('')}</div>`;
  const envTags = (arr) => (arr && arr.length)
    ? `<div class="tag-list">${arr.map((kv) => `<span class="tag"><span class="k">${esc(kv[0])}</span><span class="eq">=</span><span class="v">${esc(kv[1])}</span></span>`).join('')}</div>`
    : '<span class="text-muted">—</span>';

  /* ---------------- List rendering ---------------- */
  const viewState = {
    workspaces: 'card',
    artifacts: localStorage.getItem('axisml.artifactView') || 'card',
  };
  try { viewState.workspaces = localStorage.getItem('axisml.workspaceView') || 'card'; } catch (e) {}

  function renderList(kind) {
    const host = $(`[data-list="${kind}"]`);
    if (!host) return;
    const items = DATA[kind] || [];
    const isArtifact = ['datasets', 'models', 'images'].includes(kind);
    const view = isArtifact ? viewState.artifacts : (kind === 'workspaces' ? viewState.workspaces : 'list');
    host.innerHTML = view === 'card' ? cards(kind, items, isArtifact) : table(kind, items, isArtifact);
  }

  function table(kind, items, isArtifact) {
    let head, rows;
    if (kind === 'workspaces') {
      head = '<th>名称</th><th style="width:100px">状态</th><th style="width:150px">资源单元</th><th>镜像</th><th style="width:90px">创建人</th><th style="width:170px" class="actions">操作</th>';
      rows = items.map((i) => `<tr>
        <td><a class="row-link" href="#/workspaces/${i.name}">${i.name}</a></td>
        <td>${pill(i.status)}</td><td class="text-mono">${i.unit}</td>
        <td class="description-cell text-mono" style="font-size:12px">${i.image}</td>
        <td>${i.creator}</td>
        <td class="actions">${i.status.label === '已停止' ? actLink('启动') : actLink('打开') + actLink('停止')}${actLink('删除', ' danger')}</td></tr>`).join('');
    } else if (kind === 'jobs') {
      head = '<th>名称</th><th style="width:120px">最近运行状态</th><th style="width:80px">运行数</th><th style="width:90px">创建人</th><th style="width:110px">更新时间</th><th style="width:200px" class="actions">操作</th>';
      rows = items.map((i) => {
        const latest = jobLatest(i);
        const cnt = (i.runs || []).length;
        const acts = `<a class="row-action-link">运行</a><a class="row-action-link" href="#/jobs/${i.name}">详情</a><a class="row-action-link">编辑</a><a class="row-action-link danger">删除</a>`;
        return `<tr><td><a class="row-link" href="#/jobs/${i.name}">${i.name}</a></td>
        <td>${pill(latest)}</td><td class="text-mono">${cnt}</td>
        <td>${i.creator}</td><td class="text-muted">${i.updated}</td>
        <td class="actions">${acts}</td></tr>`;
      }).join('');
    } else if (kind === 'services') {
      head = '<th>名称</th><th style="width:100px">状态</th><th style="width:140px">资源单元</th><th style="width:70px">副本</th><th>访问地址</th><th style="width:190px" class="actions">操作</th>';
      rows = items.map((i) => {
        const acts = (i.rep === '0/0' ? actLink('启动') : actLink('扩缩') + actLink('停止')) + actLink('删除', ' danger');
        const url = i.url === '—' ? '<span class="text-muted">—</span>' : `<span class="text-mono" style="font-size:12px">${i.url}</span>`;
        return `<tr><td><a class="row-link" href="#/services/${i.name}">${i.name}</a></td>
        <td>${pill(i.status)}</td><td class="text-mono">${i.unit}</td><td class="text-mono">${i.rep}</td>
        <td>${url}</td><td class="actions">${acts}</td></tr>`;
      }).join('');
    } else if (kind === 'traffic') {
      head = '<th>名称</th><th style="width:80px">模式</th><th style="width:100px">状态</th><th style="width:240px">后端（流量分布）</th><th>访问地址</th><th style="width:130px" class="actions">操作</th>';
      rows = items.map((i) => {
        const enabled = i.status.label === '生效中' || i.status.label === '灰度中';
        const acts = (enabled ? actLink('禁用') : actLink('启用')) + actLink('删除', ' danger');
        const url = i.url === '—' ? '<span class="text-muted">—</span>' : `<span class="text-mono" style="font-size:12px">${i.url}</span>`;
        return `<tr><td><a class="row-link" href="#/traffic/${i.name}">${i.name}</a></td>
        <td>${modePill(i.mode)}</td><td>${pill(i.status)}</td>
        <td>${flowBars(i)}</td>
        <td>${url}</td><td class="actions">${acts}</td></tr>`;
      }).join('');
    } else { // artifacts
      const m = ART_META[kind];
      head = `<th>名称</th><th style="width:120px">${m.label}</th><th style="width:110px">最新版本</th><th style="width:80px">版本数</th><th style="width:110px">更新时间</th><th style="width:160px" class="actions">操作</th>`;
      rows = items.map((i) => {
        const acts = (i.vis === 'public' ? '' : actLink('上传新版本'));
        return `<tr><td><a class="row-link name-pub" href="#/${kind}/${i.name}">${i.name}${visMark(i.vis)}</a></td>
        <td><span class="pill pill-plain">${i.spec}</span></td><td class="text-mono">${i.latest}</td>
        <td class="text-mono">${i.versions}</td><td class="text-muted">${i.updated}</td>
        <td class="actions">${acts}</td></tr>`;
      }).join('');
    }
    const total = items.length;
    const pubNote = isArtifact ? ` · 含 ${items.filter((i) => i.vis === 'public').length} 个公共` : '';
    return `<div class="card"><table class="table"><thead><tr>${head}</tr></thead><tbody>${rows}</tbody></table>
      <div class="table-foot"><span>共 ${total} 个${pubNote}</span>
      <div class="pager"><button disabled><svg width="10" height="10" viewBox="0 0 10 10" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M6.5 2 3.5 5l3 3"/></svg></button><button class="active">1</button><button disabled><svg width="10" height="10" viewBox="0 0 10 10" fill="none" stroke="currentColor" stroke-width="1.5"><path d="m3.5 2 3 3-3 3"/></svg></button>
      <span class="page-size">每页 <select><option>20</option><option>50</option></select> 条</span></div></div></div>`;
  }

  function cards(kind, items, isArtifact) {
    let body;
    if (kind === 'workspaces') {
      body = items.map((i) => `<div class="res-card ws-card" data-card-href="#/workspaces/${i.name}">
        <div class="res-card-head with-logo">${wsLogo(i.image)}<div class="res-card-id"><a class="res-card-name" href="#/workspaces/${i.name}">${i.name}</a></div>${pill(i.status)}</div>
        <div class="res-card-meta"><span class="mono">${i.unit}</span><span>${i.creator}</span></div>
        <div class="res-card-foot ws-foot">${wsFoot(i)}</div></div>`).join('');
    } else { // artifacts — ArtifactHub-style cards
      const m = ART_META[kind];
      const ico = ART_CAT[kind] || '';
      body = items.map((i) => {
        const disp = ART_DISPLAY[i.name] || `${m.display}资产`;
        const pub = i.vis === 'public';
        const badge = pub
          ? `<span class="ah-badge ah-verified">${AH_ICO.check}已认证</span>`
          : '';
        return `<div class="res-card ah-card" data-cat="${kind}" data-card-href="#/${kind}/${i.name}">
        <div class="ah-top">
          <span class="ah-logo">${ico}</span>
          <div class="ah-id">
            <a class="res-card-name ah-name" href="#/${kind}/${i.name}">${i.name}</a>
            <p class="ah-desc">${disp}</p>
          </div>
          ${badge}
        </div>
        <div class="ah-foot">
          <span class="ah-spec">${i.spec}</span>
          <span class="ah-foot-meta">
            <span class="ah-fm" title="最新版本">${AH_ICO.tag}<span class="mono">${i.latest}</span></span>
            <span class="ah-fm" title="版本数">${AH_ICO.layers}<span class="mono">${i.versions}</span></span>
            <span class="ah-fm ah-updated">更新 ${i.updated}</span>
          </span>
        </div></div>`;
      }).join('');
    }
    return `<div class="card-grid">${body}</div>`;
  }

  /* ---------------- Artifact detail ---------------- */
  function renderArtifactDetail(kind, name) {
    const item = (DATA[kind] || []).find((i) => i.name === name) || (DATA[kind] || [])[0];
    if (!item) return;
    const m = ART_META[kind];
    const disp = ART_DISPLAY[item.name] || item.name;
    const sec = $('[data-page="artifact-detail"]');
    // header
    $('[data-art-head]', sec).innerHTML = `
      <div class="detail-head">
        <a class="detail-back" href="#/${kind}"><svg width="11" height="11" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M10 3 5 8l5 5"/></svg>返回${m.display}列表</a>
        <div class="detail-title-row"><h1 class="detail-title">${item.name}</h1>${visMark(item.vis)}</div>
        <p class="detail-sub">${disp}<span class="mono">${m.plural}/${item.name}</span></p>
        ${item.vis === 'public' ? '' : `<div class="detail-actions"><button class="btn btn-primary btn-sm" data-upload-version><svg width="12" height="12" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round"><path d="M8 3.5v9M3.5 8h9"/></svg>上传新版本</button></div>`}
      </div>`;
    // meta + tags
    $('[data-art-meta]', sec).innerHTML = `
      <section class="section" style="margin:0">
        <h3>元数据 <span class="h3-actions"><button class="btn btn-sm">编辑</button></span></h3>
        <dl class="kv-grid">
          <dt>${m.label}</dt><dd><span class="pill pill-plain">${item.spec}</span></dd>
          <dt>可见性</dt><dd>${item.vis === 'public' ? pubMark + ' <span style="vertical-align:middle">公共</span>' : '本租户'} <span class="text-muted" style="font-size:12px;margin-left:4px">创建后不可变</span></dd>
          <dt>创建人</dt><dd>张伟 · <span class="text-mono text-muted">zhang.wei</span></dd>
          <dt>创建时间</dt><dd class="text-mono text-muted">2026-05-01 10:22</dd>
          <dt>最近更新</dt><dd class="text-mono text-muted">更新 ${item.updated}</dd>
        </dl>
      </section>
      <section class="section" style="margin:0">
        <h3>标签 / 注解</h3>
        <div class="tag-list">
          <span class="tag"><span class="k">task</span><span class="eq">=</span><span class="v">chat</span></span>
          <span class="tag"><span class="k">lang</span><span class="eq">=</span><span class="v">zh</span></span>
          <span class="tag text-muted">+2</span>
        </div>
      </section>`;
    // versions
    const rows = buildVersions(item, m).map((v) => `<tr>
      <td class="text-mono">${v.ver}</td>
      <td>${pill(v.status)}</td>
      <td><span class="digest">${v.digest}</span> ${copyIco(v.digest)}</td>
      <td class="text-mono">${v.size}</td>
      <td>${v.creator}</td>
      <td class="actions"><a class="row-action-link" data-pull="${kind}|${item.name}|${v.ver}">拉取命令</a><a class="row-action-link">下载</a><a class="row-action-link danger">删除</a></td></tr>`).join('');
    $('[data-art-versions]', sec).innerHTML = `
      <h3 style="margin:0 0 12px;font-size:13px;font-weight:600;letter-spacing:-0.28px">版本列表</h3>
      <div class="card"><table class="table"><thead><tr><th style="width:110px">版本</th><th style="width:90px">状态</th><th>digest</th><th style="width:90px">大小</th><th style="width:90px">创建人</th><th style="width:200px" class="actions">操作</th></tr></thead><tbody>${rows}</tbody></table></div>`;
  }

  function buildVersions(item, m) {
    const out = [];
    const n = item.versions;
    const hexes = ['a1b2c3d4', '9f8e7d6c', '77cd33ab', '5c4d3e2f', 'b0a19283', 'cafe4218', 'd34db33f', '0fee1234'];
    const creators = ['张伟', '李娜', '陈曦', '王磊'];
    for (let k = 0; k < n; k++) {
      const ver = k === 0 ? item.latest : (/^v\d+$/.test(item.latest) ? 'v' + (item.versions - k) : `rev-${item.versions - k}`);
      const uploading = false;
      out.push({
        ver,
        status: st('success', '就绪'),
        digest: 'sha256:' + hexes[k % hexes.length] + '…',
        size: m.sizeBase,
        creator: creators[k % creators.length],
      });
    }
    return out;
  }

  /* ---------------- Traffic policy detail ---------------- */
  function renderTrafficDetail(name) {
    const item = (DATA.traffic || []).find((i) => i.name === name) || (DATA.traffic || [])[0];
    if (!item) return;
    const sec = $('[data-page="traffic-detail"]');
    const canary = item.mode === '灰度';
    const ready = item.status.label !== '未就绪';

    // header
    const acts = `<button class="btn btn-sm btn-danger">删除</button>`;
    $('[data-traffic-head]', sec).innerHTML = `
      <div class="detail-head">
        <a class="detail-back" href="#/traffic"><svg width="11" height="11" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M10 3 5 8l5 5"/></svg>返回流量策略列表</a>
        <div class="detail-title-row"><h1 class="detail-title">${item.name}</h1>${pill(item.status)}${modePill(item.mode)}</div>
        <p class="detail-sub">${item.display}<span class="mono">traffic/${item.name}</span></p>
        <div class="detail-actions">${acts}</div>
      </div>`;

    // basic
    const authMap = { jwt: 'JWT', apiKey: 'API Key', none: '无（公开）' };
    const urlVal = item.url === '—'
      ? '<span class="text-muted">— 策略未就绪</span>'
      : `<span class="copyable"><code>${item.url}</code>${copyIco(item.url)}</span>`;
    $('[data-traffic-basic]', sec).innerHTML = `
      <section class="section">
        <h3>基本信息 <span class="h3-actions"><button class="btn btn-sm">编辑</button></span></h3>
        <dl class="kv-grid">
          <dt>名称</dt><dd><code>${item.name}</code> <span class="text-muted" style="font-size:12px;margin-left:6px;">创建后不可变</span></dd>
          <dt>描述</dt><dd>${item.desc}</dd>
          <dt>模式</dt><dd>${modePill(item.mode)} <span class="text-muted" style="font-size:12px;margin-left:6px;">创建后不可变</span></dd>
          <dt>对外入口</dt><dd>${urlVal}</dd>
          <dt>鉴权</dt><dd><span class="pill pill-plain">${authMap[item.auth] || item.auth}</span> <span class="text-muted" style="font-size:12px;margin-left:6px;">创建后不可变</span></dd>
          <dt>后端数</dt><dd><span class="text-mono">${item.backends.length}</span></dd>
          <dt>创建人</dt><dd>${item.creator}</dd>
          <dt>创建时间</dt><dd class="text-mono text-muted">${item.created}</dd>
        </dl>
      </section>`;

    // flow distribution
    let flowHTML = '';
    if (canary && ready) {
      const c = item.canary || 0;
      flowHTML += `
        <div class="canary-panel">
          <div class="canary-head"><span class="lbl">灰度百分比</span><span class="canary-pct">${c}<small>%</small></span></div>
          <div class="slider-wrap">
            <input type="range" class="slider" min="0" max="100" value="${c}" data-canary-slider style="--pct:${c}%">
          </div>
          <div class="canary-split">
            <span class="stable">稳定 <b data-canary-stable>${100 - c}%</b></span>
            <span class="canary">灰度 <b data-canary-canary>${c}%</b></span>
          </div>
          <div class="canary-actions">
            <button class="btn btn-primary btn-sm" data-traffic-promote="${item.name}">提升为稳定</button>
            <button class="btn btn-sm" data-traffic-rollback="${item.name}">回滚</button>
          </div>
        </div>`;
    } else if (!canary && ready) {
      flowHTML += `
        <div class="canary-panel">
          <div class="canary-head"><span class="lbl">加权流量（实时 Σ=100 校验）</span></div>
          ${item.backends.map((b) => `<div class="wt-row">
            <span class="flow-name">${b.svc}</span>
            <span class="wt-input"><input class="input mono" value="${b.weight}" data-wt-input> <span class="text-muted">%</span></span>
          </div>`).join('')}
          <div class="wt-sum ok" data-wt-sum>Σ = 100% ✓</div>
          <div class="canary-actions"><button class="btn btn-primary btn-sm" data-traffic-adjust="${item.name}">应用权重</button></div>
        </div>`;
    } else {
      flowHTML += `<div class="banner"><svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="8" cy="8" r="6.5"/><path d="M8 5v4M8 11h.01"/></svg>至少一个成员服务非「就绪」或缺失，策略尚未生效。请先补齐稳定后端。</div>`;
    }
    // backend table
    flowHTML += `<section class="section" style="margin-top:16px">
      <h3>后端分布</h3>
      <div class="card"><table class="table">
        <thead><tr><th>在线服务</th><th style="width:90px">角色</th><th style="width:100px">目标权重</th><th style="width:150px">实际流量占比</th><th style="width:110px">后端状态</th></tr></thead>
        <tbody>${item.backends.map((b) => {
          const fillCls = b.weight === 0 ? 'zero' : (b.role === '灰度' ? 'canary' : 'stable');
          const svcCell = b.missing ? `<span class="flow-name missing">${b.svc}</span>`
            : `<a class="row-link" href="#/services/${b.svc}">${b.svc}</a>`;
          return `<tr>
            <td>${svcCell}</td>
            <td>${roleTag(b.role)}</td>
            <td class="text-mono">${b.missing ? '—' : b.weight}</td>
            <td><span class="flow-track" style="display:inline-block;width:80px;vertical-align:middle;margin-right:8px"><span class="flow-fill ${fillCls}" style="width:${b.missing ? 0 : b.weight}%"></span></span><span class="text-mono">${b.missing ? '—' : b.weight + '%'}</span></td>
            <td>${pill(b.phase)}</td></tr>`;
        }).join('')}</tbody>
      </table></div>
    </section>`;
    $('[data-traffic-flow]', sec).innerHTML = flowHTML;

    // monitor
    const legend = `<div class="legend"><span><i style="background:#0070f3"></i>${item.backends[0].svc}</span><span><i style="background:#f5a623"></i>${item.backends[1] ? item.backends[1].svc : '—'}</span></div>`;
    const cmp = canary ? `<div class="chart-card" style="grid-column:1/-1"><div class="chart-head"><div class="chart-title">稳定 vs 灰度 · 健康对比（错误率）</div><div class="chart-meta">灰度 +0.01%</div></div><svg class="chart-svg" data-chart="tr-err"></svg></div>` : '';
    $('[data-traffic-monitor]', sec).innerHTML = `
      <div class="filters" style="padding-top:0;">
        <span class="text-muted" style="font-size:12.5px;">来自 compute 指标代理 · 按后端分组</span>
        <div class="spacer"></div>
        <div class="range-picker" data-range><button>5m</button><button class="active">1h</button><button>24h</button></div>
      </div>
      <div class="chart-grid cols-3">
        <div class="chart-card"><div class="chart-head"><div class="chart-title">QPS</div>${legend}</div><svg class="chart-svg" data-chart="tr-qps"></svg></div>
        <div class="chart-card"><div class="chart-head"><div class="chart-title">延迟 p95</div>${legend}</div><svg class="chart-svg" data-chart="tr-lat"></svg></div>
        <div class="chart-card"><div class="chart-head"><div class="chart-title">错误率 (5xx)</div>${legend}</div><svg class="chart-svg" data-chart="tr-err"></svg></div>
        ${cmp}
      </div>`;

    // events
    const ev = canary
      ? [
          ['ok', 'WeightUpdated', 'Normal', item.created, `灰度百分比调整为 ${item.canary || 0}%`],
          ['ok', 'BackendReady', 'Normal', '2026-06-10 15:42:20', `后端 ${item.backends[1] ? item.backends[1].svc : ''} 已就绪`],
          ['', 'RouteDerived', 'Normal', '2026-06-10 15:42:08', `已派生加权路由（HTTPRoute 加权 backendRefs）`],
          ['', 'PolicyCreated', 'Normal', '2026-06-10 15:42:02', `创建灰度策略，对外入口 ${item.url}`],
        ]
      : [
          ['ok', 'WeightApplied', 'Normal', item.created, `权重下发生效：${item.backends.map((b) => b.svc + '=' + b.weight).join(' · ')}`],
          ['ok', 'BackendReady', 'Normal', '2026-05-30 09:19:10', `全部后端就绪`],
          ['', 'PolicyCreated', 'Normal', '2026-05-30 09:18:55', `创建加权策略，对外入口 ${item.url}`],
        ];
    $('[data-traffic-events]', sec).innerHTML = `<div class="section"><div class="timeline">${ev.map(function (e) {
      return `<div class="tl-item"><span class="tl-dot ${e[0]}"></span><div class="tl-head"><span class="tl-reason">${e[1]}</span><span class="tl-type">${e[2]}</span><span class="tl-time">${e[3]}</span></div><div class="tl-msg">${e[4]}</div></div>`;
    }).join('')}</div></div>`;

    resetTabs(sec);
  }

  /* ---------------- Job (template) → Run (运行) two-level detail ---------------- */
  function runActions(job, r) {
    const term = RUN_TERMINAL.includes(r.status.label);
    const base = `#/jobs/${job.name}/runs/${job.name}-${r.n}`;
    const cancel = term ? '' : '<a class="row-action-link">取消</a>';
    const del = term ? '<a class="row-action-link danger">删除</a>' : '';
    return `${cancel}<a class="row-action-link" href="${base}">日志</a><a class="row-action-link" href="${base}">详情</a>${del}`;
  }

  function renderJobDetail(name) {
    const job = findJob(name);
    const sec = $('[data-page="job-detail"]');
    if (!job || !sec) return;
    // header
    $('[data-job-head]', sec).innerHTML = `
      <div class="detail-head">
        <a class="detail-back" href="#/jobs"><svg width="11" height="11" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M10 3 5 8l5 5"/></svg>返回 Job 列表</a>
        <div class="detail-title-row"><h1 class="detail-title">${job.name}</h1>${pill(jobLatest(job))}</div>
        <p class="detail-sub">${job.display}<span class="mono">jobs/${job.name}</span></p>
        <div class="detail-actions">
          <button class="btn btn-primary btn-sm" data-run-job="${job.name}"><svg width="11" height="11" viewBox="0 0 16 16" fill="currentColor"><path d="M4 2.5 13 8 4 13.5z"/></svg>运行</button>
          <button class="btn btn-sm">编辑</button>
          <button class="btn btn-sm btn-danger">删除</button>
        </div>
      </div>`;
    // runs tab
    const runsHost = $('[data-job-runs]', sec);
    if (!job.runs.length) {
      runsHost.innerHTML = `<div class="card" style="padding:48px 24px;text-align:center"><p class="text-muted" style="margin:0 0 16px;font-size:13.5px">该 Job 还没有运行记录。</p><button class="btn btn-primary btn-sm" data-run-job="${job.name}" style="display:inline-flex"><svg width="11" height="11" viewBox="0 0 16 16" fill="currentColor"><path d="M4 2.5 13 8 4 13.5z"/></svg>触发第一次运行</button></div>`;
    } else {
      const rows = job.runs.map((r) => `<tr>
        <td><a class="row-link" href="#/jobs/${job.name}/runs/${job.name}-${r.n}">${job.name}-${r.n}</a></td>
        <td>${pill(r.status)}</td>
        <td class="text-mono">${r.unit}</td>
        <td class="text-mono">${r.rep}</td>
        <td>${r.trigger}</td>
        <td class="text-mono ${r.status.label === '运行中' ? 'live-dur' : 'text-muted'}"${r.status.label === '运行中' ? ' data-live-clock="1"' : ''}>${r.dur}</td>
        <td class="actions">${runActions(job, r)}</td></tr>`).join('');
      runsHost.innerHTML = `<div class="card"><table class="table">
        <thead><tr><th>Run</th><th style="width:96px">状态</th><th style="width:140px">资源单元</th><th style="width:64px">副本</th><th style="width:84px">触发人</th><th style="width:108px">耗时</th><th style="width:170px" class="actions">操作</th></tr></thead>
        <tbody>${rows}</tbody></table>
        <div class="table-foot"><span>共 ${job.runs.length} 次运行 · 实时回源 compute</span></div></div>`;
    }
    // spec tab
    const t = job.tpl;
    $('[data-job-spec]', sec).innerHTML = `<section class="section">
      <h3>配置 <span class="h3-actions"><button class="btn btn-sm">编辑</button></span></h3>
      <p class="form-help" style="margin:-4px 0 16px">对 Job 配置的修改只影响<strong>之后</strong>触发的运行；历史运行保留各自启动时的快照。</p>
      <dl class="kv-grid">
        <dt>名称</dt><dd><code>${job.name}</code> <span class="text-muted" style="font-size:12px;margin-left:4px">创建后不可变</span></dd>
        <dt>显示名</dt><dd>${job.display}</dd>
        <dt>描述</dt><dd>${job.desc}</dd>
        <dt>资源池 / 单元</dt><dd><code>${t.pool}/${t.unit}</code> <span class="text-muted" style="font-size:12px;margin-left:4px">${t.spec}</span></dd>
        <dt>镜像</dt><dd><code>${t.image}</code></dd>
        <dt>副本数</dt><dd><span class="text-mono">${t.replicas}</span> <span class="text-muted" style="font-size:12px;margin-left:4px">${t.replicas > 1 ? '分布式 · torchrun' : '单机'}</span></dd>
        <dt>命令 / 参数</dt><dd>${cmdTags(t.cmd)}</dd>
        <dt>环境变量</dt><dd>${envTags(t.env)}</dd>
        <dt>运行策略</dt><dd><div class="tag-list"><span class="tag"><span class="k">timeout</span><span class="eq">=</span><span class="v">${t.policy.timeout}</span></span><span class="tag"><span class="k">backoffLimit</span><span class="eq">=</span><span class="v">${t.policy.backoff}</span></span><span class="tag"><span class="k">ttlAfterFinished</span><span class="eq">=</span><span class="v">${t.policy.ttl}</span></span></div></dd>
        <dt>制品引用</dt><dd><div class="tag-list">${t.artifacts.map((a) => `<span class="tag"><span class="k">${a[0]}</span><span class="eq">:</span><span class="v">${a[1]}@${a[2]}</span></span>`).join('')}</div></dd>
      </dl></section>`;
    resetTabs(sec);
  }

  function renderRunDetail(jobName, runName) {
    const job = findJob(jobName);
    const r = findRun(job, runName);
    const sec = $('[data-page="job-run-detail"]');
    if (!job || !r || !sec) return;
    const full = `${job.name}-${r.n}`;
    const term = RUN_TERMINAL.includes(r.status.label);
    const t = job.tpl;
    // header
    const acts = (term ? '' : '<button class="btn btn-sm btn-danger">取消运行</button>') + '<button class="btn btn-sm">删除</button>';
    $('[data-run-head]', sec).innerHTML = `
      <div class="detail-head">
        <a class="detail-back" href="#/jobs/${job.name}"><svg width="11" height="11" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M10 3 5 8l5 5"/></svg>返回 ${job.name}</a>
        <div class="detail-title-row"><h1 class="detail-title">${full}</h1>${pill(r.status)}</div>
        <p class="detail-sub">${job.display}<span class="mono">jobs/${job.name}/runs/${full}</span></p>
        <div class="detail-actions">${acts}</div>
      </div>`;
    // basic
    const msgCls = r.status.label === '失败' ? 'style="color:var(--app-danger)"' : 'class="text-muted"';
    $('[data-run-basic]', sec).innerHTML = `<section class="section">
      <h3>基本信息</h3>
      <p class="form-help" style="margin:-4px 0 16px">Run 是该次运行的不可变快照，创建后 spec 不可变。</p>
      <dl class="kv-grid">
        <dt>Run 名</dt><dd><code>${full}</code></dd>
        <dt>资源单元</dt><dd><code>${r.unit}</code> <span class="text-muted" style="font-size:12px;margin-left:4px">${r.spec}</span></dd>
        <dt>镜像</dt><dd><code>${t.image}</code></dd>
        <dt>副本数</dt><dd><span class="text-mono">${r.rep}</span></dd>
        <dt>命令 / 参数</dt><dd>${cmdTags(t.cmd)}</dd>
        <dt>环境变量</dt><dd>${envTags(t.env)}</dd>
        <dt>运行策略</dt><dd><div class="tag-list"><span class="tag"><span class="k">timeout</span><span class="eq">=</span><span class="v">${t.policy.timeout}</span></span><span class="tag"><span class="k">backoffLimit</span><span class="eq">=</span><span class="v">${t.policy.backoff}</span></span><span class="tag"><span class="k">ttlAfterFinished</span><span class="eq">=</span><span class="v">${t.policy.ttl}</span></span></div></dd>
        ${r.override ? `<dt>触发期 override</dt><dd><span class="pill pill-plain">本次覆盖</span> <span class="text-muted" style="font-size:12.5px;margin-left:4px">${r.override}</span></dd>` : ''}
        <dt>触发人</dt><dd>${r.trigger}</dd>
        <dt>开始时间</dt><dd class="text-mono text-muted">${r.started}</dd>
        <dt>结束时间</dt><dd class="text-mono text-muted">${r.finished}</dd>
        <dt>耗时</dt><dd class="text-mono ${r.status.label === '运行中' ? 'live-dur' : 'text-muted'}"${r.status.label === '运行中' ? ' data-live-clock="1"' : ''}>${r.dur}</dd>
        <dt>状态消息</dt><dd ${msgCls}>${r.msg}</dd>
      </dl></section>`;
    // pods
    const n = parseInt(String(r.rep).split('/')[0], 10) || 1;
    const running = r.status.label === '运行中';
    const podRows = [];
    for (let k = 0; k < n; k++) {
      const phase = running ? st('success', 'Running')
        : r.status.label === '成功' ? st('success', 'Completed')
        : r.status.label === '失败' ? (k === 3 ? st('danger', 'Error') : st('success', 'Completed'))
        : st('muted', 'Terminated');
      const node = 'node-' + (t.pool.includes('a100') ? 'a100' : t.pool.includes('h100') ? 'h100' : 'cpu') + '-0' + (2 + (k % 4));
      const exit = running ? '—' : (phase.label === 'Error' ? '<span style="color:var(--app-danger)">1</span>' : '0');
      const restart = (r.status.label === '失败' && k === 3) ? '2' : (running && k === 2 ? '1' : '0');
      podRows.push(`<tr><td class="text-mono">${full}-${k}${k === 0 && n > 1 ? ' <span class="text-muted" style="font-size:11px">(master)</span>' : ''}</td>
        <td>${pill(phase)}</td><td class="text-mono text-muted">${node}</td><td class="text-mono">${restart}</td>
        <td class="text-mono${exit.includes('danger') ? '' : ' text-muted'}">${exit}</td>
        <td class="text-mono text-muted">${r.started.split(' ')[1] || '—'}</td>
        <td class="actions"><a class="row-action-link" data-tab-go="logs">日志</a></td></tr>`);
    }
    $('[data-run-pods]', sec).innerHTML = `<div class="card pods-mini"><table class="table">
      <thead><tr><th>Pod 名称</th><th style="width:110px">阶段</th><th>节点</th><th style="width:70px">重启</th><th style="width:70px">退出码</th><th style="width:90px">启动时间</th><th style="width:70px" class="actions">操作</th></tr></thead>
      <tbody>${podRows.join('')}</tbody></table></div>`;
    // logs
    const startT = (r.started.split(' ')[1] || '06:21:08');
    const logLines = running
      ? [['info', '[rank0] initializing process group: nccl, world_size=' + n], ['ok', '[rank0] model loaded, 6.74B params, bf16'], ['info', '[rank0] step 100/36000 | loss 1.842 | lr 2.0e-5'], ['dim', '[rank0] saving checkpoint step-12400 to s3://artifacts/...'], ['ok', '[rank0] checkpoint saved (13.4GB)'], ['info', '[rank0] step 12400/36000 | loss 0.958 | lr 1.59e-5 | 3.19 it/s']]
      : r.status.label === '失败'
        ? [['info', '[rank0] initializing process group: nccl, world_size=' + n], ['ok', '[rank0] model loaded, 6.74B params'], ['info', '[rank3] step 1/36000 | loss 1.91'], ['err', '[rank3] CUDA out of memory: tried to allocate 2.10 GiB'], ['err', '[rank0] NCCL watchdog: rank 3 unresponsive, aborting'], ['warn', 'job failed: backoffLimit=0 exhausted']]
        : [['info', '[rank0] initializing process group: nccl, world_size=' + n], ['ok', '[rank0] model loaded'], ['info', '[rank0] step 36000/36000 | loss 0.512'], ['dim', '[rank0] saving final checkpoint ...'], ['ok', '[rank0] artifact registered: ' + (job.tpl.artifacts.find((a) => a[0] === '模型') ? 'llama-7b-sft@v3' : 'done')], ['ok', 'job completed successfully']];
    $('[data-run-logs]', sec).innerHTML = `
      <div class="term-bar">
        <div class="field field-select"><span class="text-muted">Pod</span><select>${Array.from({ length: n }, (_, k) => `<option>${full}-${k}${k === 0 && n > 1 ? ' (master)' : ''}</option>`).join('')}</select></div>
        <div class="field field-select"><span class="text-muted">行数</span><select><option>1000</option><option>5000</option></select></div>
        <div class="spacer"></div>
        <div class="follow">实时跟随<span class="switch${running ? ' on' : ''}" data-switch></span></div>
      </div>
      <div class="term">${logLines.map((l, idx) => `<div class="ln"><span class="t">${startT}</span> <span class="${l[0]}">${l[0] === 'dim' ? l[1] : '[' + (l[0] === 'err' ? 'E' : l[0] === 'warn' ? 'W' : l[0] === 'ok' ? 'I' : 'I') + ']'}</span> ${l[0] === 'dim' ? '' : l[1]}</div>`).join('')}</div>`;
    // events
    const ev = running
      ? [['ok', 'Running', 'Normal', r.started, `All ${n} replicas are running`], ['', 'Scheduled', 'Normal', r.started, `Gang-scheduled ${n} pods to ${t.pool}`], ['', 'Created', 'Normal', r.started, `Run ${full} created from job template`]]
      : r.status.label === '失败'
        ? [['err', 'Failed', 'Warning', r.finished, r.msg], ['warn', 'BackOff', 'Warning', r.started, 'rank3 restarted, CUDA OOM'], ['', 'Scheduled', 'Normal', r.started, `Gang-scheduled ${n} pods to ${t.pool}`]]
        : r.status.label === '已取消'
          ? [['', 'Cancelled', 'Normal', r.finished, '用户主动取消，Pod 已终止'], ['', 'Scheduled', 'Normal', r.started, `Gang-scheduled ${n} pods to ${t.pool}`]]
          : [['ok', 'Completed', 'Normal', r.finished, r.msg], ['ok', 'Running', 'Normal', r.started, `All ${n} replicas running`], ['', 'Scheduled', 'Normal', r.started, `Gang-scheduled ${n} pods to ${t.pool}`]];
    $('[data-run-events]', sec).innerHTML = `<div class="section"><div class="timeline">${ev.map((e) => `<div class="tl-item"><span class="tl-dot ${e[0]}"></span><div class="tl-head"><span class="tl-reason">${e[1]}</span><span class="tl-type">${e[2]}</span><span class="tl-time">${e[3]}</span></div><div class="tl-msg">${e[4]}</div></div>`).join('')}</div></div>`;
    resetTabs(sec);
  }

  /* ---------------- Compute detail header sync ---------------- */
  function syncComputeDetail(kind, name) {
    const pageId = DETAIL[kind].page;
    const sec = $(`[data-page="${pageId}"]`);
    if (!sec) return;
    const item = (DATA[kind] || []).find((i) => i.name === name);
    const title = $('.detail-title', sec);
    const subMono = $('.detail-sub .mono', sec);
    if (title) title.textContent = name;
    if (subMono) subMono.textContent = `${kind}/${name}`;
    if (item) {
      const p = $('.detail-title-row .pill', sec);
      if (p) { p.className = 'pill pill-' + item.status.tone; p.textContent = item.status.label; }
    }
    // reset to first tab
    resetTabs(sec);
  }

  function resetTabs(sec) {
    const tabs = $('.tabs', sec);
    if (!tabs) return;
    const first = $('.tab', tabs);
    $$('.tab', tabs).forEach((t) => t.classList.toggle('active', t === first));
    const key = first.getAttribute('data-tab');
    $$('.tab-panel', sec).forEach((p) => p.classList.toggle('hidden', p.getAttribute('data-tab-panel') !== key));
  }

  /* ---------------- Charts ---------------- */
  const SERIES = {
    'tenant-trend': [3, 3, 4, 4, 3, 5, 6, 6, 5, 6, 7, 6, 6, 7, 8, 7, 6, 6],
    'svc-qps':  [90, 110, 105, 120, 135, 128, 140, 150, 138, 142, 160, 155, 148, 152, 168, 160, 150, 158],
    'svc-err':  [0.02, 0.03, 0.02, 0.05, 0.04, 0.03, 0.06, 0.04, 0.03, 0.05, 0.07, 0.05, 0.04, 0.06, 0.08, 0.05, 0.04, 0.04],
    'svc-cpu':  [55, 58, 52, 60, 62, 59, 63, 66, 60, 58, 64, 63, 61, 65, 67, 62, 60, 62],
    'svc-mem':  [68, 70, 69, 72, 71, 73, 72, 74, 73, 71, 74, 73, 72, 75, 74, 73, 72, 71],
    'svc-gpu':  [78, 82, 80, 85, 84, 86, 84, 88, 85, 83, 87, 86, 84, 89, 90, 86, 84, 84],
  };
  const SERIES_MULTI = {
    'svc-latency': [
      { vals: [40, 42, 38, 45, 44, 41, 43, 46, 42, 40, 44, 43, 41, 45, 47, 44, 42, 43], color: '#a1a1a1' },
      { vals: [80, 86, 82, 95, 90, 88, 92, 98, 90, 86, 94, 93, 88, 96, 99, 94, 90, 93], color: '#0070f3' },
      { vals: [120, 140, 130, 160, 150, 138, 150, 170, 150, 140, 160, 158, 148, 166, 180, 160, 150, 158], color: '#ee0000' },
    ],
    // traffic — per-backend comparison (line 0 = first backend / 稳定, line 1 = second / 灰度)
    'tr-qps': [
      { vals: [142, 150, 138, 146, 152, 148, 144, 150, 146, 142, 148, 145, 140, 147, 150, 146, 143, 145], color: '#0070f3' },
      { vals: [10, 12, 14, 16, 15, 17, 16, 18, 17, 16, 19, 18, 16, 18, 20, 18, 17, 18], color: '#f5a623' },
    ],
    'tr-lat': [
      { vals: [86, 90, 84, 92, 88, 90, 87, 91, 88, 86, 90, 89, 85, 90, 92, 89, 87, 88], color: '#0070f3' },
      { vals: [78, 82, 80, 85, 83, 84, 82, 86, 84, 82, 85, 84, 81, 85, 87, 84, 82, 83], color: '#f5a623' },
    ],
    'tr-err': [
      { vals: [0.03, 0.04, 0.03, 0.05, 0.04, 0.03, 0.05, 0.04, 0.03, 0.04, 0.05, 0.04, 0.03, 0.05, 0.06, 0.04, 0.03, 0.04], color: '#0070f3' },
      { vals: [0.02, 0.03, 0.04, 0.05, 0.04, 0.06, 0.05, 0.04, 0.05, 0.06, 0.05, 0.04, 0.05, 0.06, 0.05, 0.04, 0.05, 0.05], color: '#f5a623' },
    ],
  };
  const W = 600;

  function path(vals, h, padT, padB, fullMinZero) {
    const n = vals.length;
    let min = Math.min.apply(null, vals), max = Math.max.apply(null, vals);
    if (fullMinZero) min = Math.min(min, 0);
    const range = (max - min) || 1;
    const x = (i) => (n === 1 ? W / 2 : (i / (n - 1)) * W);
    const y = (v) => h - padB - ((v - min) / range) * (h - padT - padB);
    let line = '';
    vals.forEach((v, i) => { line += (i ? 'L' : 'M') + x(i).toFixed(1) + ' ' + y(v).toFixed(1) + ' '; });
    return { line, area: line + `L ${W} ${h} L 0 ${h} Z` };
  }

  function gridLines(h) {
    let g = '';
    for (let i = 1; i < 4; i++) { const yy = (h / 4) * i; g += `<line x1="0" y1="${yy}" x2="${W}" y2="${yy}" stroke="#ebebeb" stroke-width="1" vector-effect="non-scaling-stroke"/>`; }
    return g;
  }

  function renderChartsIn(sec) {
    $$('svg[data-chart]', sec).forEach((svg) => {
      const name = svg.dataset.chart;
      const h = svg.classList.contains('tall') ? 150 : 110;
      svg.setAttribute('viewBox', `0 0 ${W} ${h}`);
      svg.setAttribute('preserveAspectRatio', 'none');
      let inner = gridLines(h);
      if (SERIES_MULTI[name]) {
        SERIES_MULTI[name].forEach((s) => {
          const p = path(s.vals, h, 10, 10);
          inner += `<path d="${p.line}" fill="none" stroke="${s.color}" stroke-width="2" vector-effect="non-scaling-stroke" stroke-linejoin="round"/>`;
        });
      } else {
        const vals = SERIES[name] || [1, 2, 1, 2];
        const danger = name === 'svc-err';
        const col = danger ? '#ee0000' : '#0070f3';
        const fill = danger ? 'rgba(238,0,0,0.08)' : 'rgba(0,112,243,0.10)';
        const p = path(vals, h, 12, 10, danger);
        inner += `<path d="${p.area}" fill="${fill}" stroke="none"/>`;
        inner += `<path d="${p.line}" fill="none" stroke="${col}" stroke-width="2" vector-effect="non-scaling-stroke" stroke-linejoin="round"/>`;
      }
      svg.innerHTML = inner;
    });
  }

  /* ---------------- Router ---------------- */
  function applyRoute() {
    let key = (location.hash || '#/home').replace(/^#\//, '') || 'home';
    const parts = key.split('/');
    let page, crumbs, navKey;

    if (parts[0] === 'jobs' && parts[2] === 'runs' && parts[3]) {
      // Run 详情 /jobs/{job}/runs/{run}
      const jobName = parts[1], runName = parts[3];
      page = 'job-run-detail'; navKey = 'jobs';
      crumbs = ['训练中心', { text: '自定义任务', href: '#/jobs' }, { text: jobName, href: '#/jobs/' + jobName }, runName];
      renderRunDetail(jobName, runName);
    } else if (parts.length >= 2 && DETAIL[parts[0]]) {
      const d = DETAIL[parts[0]];
      const name = parts.slice(1).join('/');
      page = d.page; navKey = d.nav;
      const disp = parts[0] === 'tenants' ? 'Team A · 推理' : name;
      crumbs = [d.group, { text: d.parent, href: '#/' + parts[0] }, disp];
      if (page === 'artifact-detail') renderArtifactDetail(parts[0], name);
      else if (parts[0] === 'traffic') renderTrafficDetail(name);
      else if (parts[0] === 'jobs') renderJobDetail(name);
      else if (['workspaces', 'services'].includes(parts[0])) syncComputeDetail(parts[0], name);
    } else if (NAV[key]) {
      page = NAV[key].page; crumbs = NAV[key].crumbs; navKey = NAV[key].nav;
      if (DATA[key]) renderList(key);
    } else {
      page = NAV.home.page; crumbs = NAV.home.crumbs; navKey = 'home';
    }

    $$('.page').forEach((el) => el.classList.toggle('active', el.getAttribute('data-page') === page));

    const crumbEl = $('#crumbs');
    crumbEl.innerHTML = '';
    crumbs.forEach((c, i) => {
      if (i > 0) { const sep = document.createElement('span'); sep.className = 'sep'; sep.textContent = '/'; crumbEl.appendChild(sep); }
      const obj = typeof c === 'object';
      const item = document.createElement(obj && c.href ? 'a' : 'span');
      item.textContent = obj ? c.text : c;
      if (i === crumbs.length - 1) item.className = 'last';
      if (obj && c.href) item.href = c.href;
      crumbEl.appendChild(item);
    });

    $$('.nav-item, .nav-child').forEach((el) => el.classList.remove('active'));
    const a = document.querySelector(`.nav-item[data-route="${navKey}"], .nav-child[data-route="${navKey}"]`);
    if (a) a.classList.add('active');

    const activeSec = $(`.page[data-page="${page}"]`);
    if (activeSec) renderChartsIn(activeSec);

    window.scrollTo(0, 0);
  }

  /* ---------------- Drawer (create / upload) ---------------- */
  const scrim = $('#scrim'), drawer = $('#drawer');
  const dBody = $('#drawerBody'), dFoot = $('#drawerFoot'), dTitle = $('#drawerTitle');
  let uploadStep = 0, uploadType = 'model';

  // Reusable resource-spec picker (pool → unit). Shared across workspace / job / service.
  function poolUnitHTML() {
    return `<div class="form-section-label">资源规格</div>
      <div class="form-row">
        <div class="res-pick">
          <div><label class="form-label">资源池<span class="req">*</span></label>
            <select class="selectbox"><option value="">选择资源池…</option><option>gpu-a100</option><option>gpu-h100</option><option>gpu-l40s</option><option>cpu-medium</option><option>cpu-large</option></select></div>
          <div><label class="form-label">资源单元<span class="req">*</span></label>
            <select class="selectbox" data-unit><option value="">选择资源单元…</option><option>a100-1x-large</option><option>a100-4x-xlarge</option><option>a100-8x-xlarge-ib</option><option>h100-1x-large</option><option>l40s-1x</option><option>cpu-medium-1x</option><option>cpu-large-1x</option></select></div>
        </div>
        <div class="spec-readout" data-spec-readout><span class="rl">规格</span><span class="text-muted">选择资源单元后只读展示 requests / limits</span></div>
      </div>`;
  }
  // Image picker — full width, decoupled from pool/unit (references are long).
  function imageHTML() {
    return `<div class="form-section-label">镜像</div>
      <div class="form-row"><label class="form-label">镜像<span class="req">*</span></label>
        <select class="selectbox mono"><option value="">选择镜像…</option><option>registry.axisml.io/team-a/pytorch-train:2.3-cu121</option><option>registry.axisml.io/team-a/jupyter-ds:2024.3</option><option>registry.axisml.io/axisml-system/triton-infer:24.05</option><option>registry.axisml.io/axisml-system/vscode-server:1.89</option></select>
        <div class="form-help">合并「当前租户」+「公共（axisml-system）」镜像；引用路径较长，可输入完整引用兜底。</div>
      </div>`;
  }
  const baseFields = (prefix) => `
    <div class="form-row"><label class="form-label">名称<span class="req">*</span><span class="opt">创建后不可变</span></label><input class="input mono" placeholder="${prefix}-name" maxlength="63">
      <div class="form-help">需符合 DNS-1123：小写字母、数字与连字符（-），以字母或数字开头与结尾，最长 63 个字符。</div></div>
    <div class="form-row"><label class="form-label">描述</label><textarea class="textarea" style="font-family:var(--fnt-sans)" placeholder="可选"></textarea></div>`;
  const envRow = `<div class="form-row"><label class="form-label">环境变量<span class="opt">KEY=VALUE，每行一项</span></label><textarea class="textarea" placeholder="HF_HOME=/data/hf&#10;NCCL_DEBUG=INFO"></textarea></div>`;

  const FORMS = {
    workspace: () => ({
      eye: 'workspaces / create', title: '新建工作区',
      body: baseFields('ws') + poolUnitHTML() + imageHTML() +
        `<div class="form-section-label">数据卷</div>
         <div class="form-row cols-2"><div><label class="form-label">数据卷</label><select class="selectbox"><option value="">选择数据卷…</option><option>ws-dev-zhang-data</option><option>team-a-shared</option><option>新建数据卷…</option></select></div><div><label class="form-label">挂载路径</label><input class="input mono" placeholder="/data"></div></div>` +
        envRow,
      submit: '创建工作区',
    }),
    job: () => ({
      eye: 'jobs / create', title: '新建 Job',
      body: `<p class="form-help" style="margin:0 0 16px">保存后<strong>不会立即运行</strong>；之后在列表或详情点「运行」即按当前配置创建一次 Run。</p>` +
        baseFields('train') + poolUnitHTML() + imageHTML() +
        `<div class="form-row"><label class="form-label">副本数<span class="opt">分布式</span></label><input class="input mono" value="1"></div>
         <div class="form-row"><label class="form-label">命令 / 参数</label><textarea class="textarea" placeholder="torchrun --nproc_per_node=4 train.py --lr 2e-5"></textarea></div>` +
        envRow +
        `<div class="adv-toggle" data-adv><span class="chev"><svg width="10" height="10" viewBox="0 0 10 10" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M3 2.5 6 5 3 7.5"/></svg></span>高级设置</div>
         <div class="adv-body" data-adv-body>
           <div class="form-row cols-3">
             <div><label class="form-label">超时(s)<span class="opt">最长运行</span></label><input class="input mono" value="86400"></div>
             <div><label class="form-label">重试上限</label><input class="input mono" value="0"></div>
             <div><label class="form-label">TTL(s)<span class="opt">完成后保留</span></label><input class="input mono" value="172800"></div>
           </div>
           <div class="form-help" style="margin-top:0">超时 = 任务最长运行时间，超过则终止；TTL = 任务终态后记录 / Pod 保留多久再回收。</div>
         </div>`,
      submit: '保存 Job',
    }),
    service: () => ({
      eye: 'services / create', title: '新建服务',
      body: baseFields('svc') + poolUnitHTML() + imageHTML() +
        `<div class="form-row"><label class="form-label">副本数</label><input class="input mono" value="2"></div>
         <label class="form-label" style="margin-top:18px">端口</label>
         <div data-ports>
           <div class="port-row"><input class="input mono" placeholder="名称，如 http" value="http"><input class="input mono" placeholder="端口，如 8000" value="8000"><button class="port-rm" data-port-rm title="移除"><svg width="13" height="13" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"><path d="M4 4l8 8M12 4l-8 8"/></svg></button></div>
         </div>
         <button class="btn btn-sm" data-port-add type="button" style="margin-bottom:4px"><svg width="12" height="12" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"><path d="M8 3.5v9M3.5 8h9"/></svg>添加端口</button>
         <div class="form-row" style="margin-top:18px"><label class="form-label">命令 / 参数<span class="opt">可选</span></label><textarea class="textarea" placeholder="可选"></textarea></div>

         <label class="form-label" style="display:flex;align-items:center;justify-content:space-between;gap:12px;margin-top:22px">默认流量策略<span class="switch" data-switch data-reveal="[data-policy-body]"></span></label>
         <div class="form-help" style="margin-top:0">默认关闭。开启后随服务一并创建一条对外流量策略；关闭时仅上线服务，可稍后在「流量策略」中单独配置。</div>
         <div data-policy-body class="hidden" style="margin-top:14px">
           <div class="form-row"><label class="form-label">策略模式</label><div class="seg" data-mode-seg>
             <button type="button" class="seg-btn active" data-mode="weighted">加权切分</button>
             <button type="button" class="seg-btn" data-mode="canary">灰度发布</button>
           </div>
           <div class="form-help" data-mode-help>加权切分：服务按权重平摊流量（Σ=100），可一键置 100 做蓝绿切换。</div></div>
           <div class="form-row"><label class="form-label">Path<span class="req">*</span><span class="opt">留空自动生成</span></label><input class="input mono" placeholder="/services/team-a/&lt;name&gt;/">
             <div class="form-help">对外入口路径，创建后不可变；须以 / 开头，留空则按 <code style="font-family:var(--fnt-mono);font-size:12px;background:var(--app-soft);padding:1px 6px;border-radius:4px">/services/team-a/&lt;name&gt;/</code> 自动生成。</div></div>
         </div>`,
      submit: '上线服务',
    }),
    traffic: () => ({
      eye: 'traffic / create', title: '新建流量策略',
      body: baseFields('rt') +
        `<div class="form-section-label">模式<span class="opt" style="font-weight:400;text-transform:none;letter-spacing:0">创建后不可变</span></div>
         <div class="form-row"><div class="seg" data-mode-seg>
           <button type="button" class="seg-btn active" data-mode="weighted">加权切分</button>
           <button type="button" class="seg-btn" data-mode="canary">灰度发布</button>
         </div>
         <div class="form-help" style="margin-top:8px" data-mode-help>加权：N 个后端按权重平摊流量（Σ=100），可一键置 100 做蓝绿切换。</div></div>

         <div class="form-section-label">对外入口<span class="opt" style="font-weight:400;text-transform:none;letter-spacing:0">创建后不可变</span></div>
         <div class="form-row"><label class="form-label">Path<span class="opt">留空自动生成</span></label><input class="input mono" placeholder="/services/team-a/&lt;name&gt;/"></div>
         <div class="form-row cols-2"><div><label class="form-label">Hostname<span class="opt">可选</span></label><input class="input mono" placeholder="chat.team-a.axisml.io"></div><div><label class="form-label">鉴权</label><select class="selectbox"><option>jwt</option><option>apiKey</option><option>none</option></select></div></div>

         <div class="form-section-label">后端</div>
         <div data-mode-body="weighted">
           <div data-bk-rows>
             <div class="bk-row"><select class="selectbox mono"><option value="">选择就绪的在线服务…</option><option>svc-embed-a</option><option>svc-embed-b</option><option>svc-chat-api</option></select><input class="input mono" value="50" style="width:84px;text-align:right"><span class="text-muted" style="font-size:12px">%</span><button class="port-rm" data-bk-rm type="button" title="移除"><svg width="13" height="13" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"><path d="M4 4l8 8M12 4l-8 8"/></svg></button></div>
             <div class="bk-row"><select class="selectbox mono"><option value="">选择就绪的在线服务…</option><option>svc-embed-a</option><option>svc-embed-b</option><option>svc-chat-api</option></select><input class="input mono" value="50" style="width:84px;text-align:right"><span class="text-muted" style="font-size:12px">%</span><button class="port-rm" data-bk-rm type="button" title="移除"><svg width="13" height="13" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"><path d="M4 4l8 8M12 4l-8 8"/></svg></button></div>
           </div>
           <button class="btn btn-sm" data-bk-add type="button" style="margin-top:4px"><svg width="12" height="12" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"><path d="M8 3.5v9M3.5 8h9"/></svg>添加后端</button>
           <div class="form-help" style="margin-top:8px">后端下拉只列当前租户「就绪」的在线服务；已被其它活跃策略占用的服务置灰。权重实时 Σ=100 校验。</div>
         </div>
         <div data-mode-body="canary" class="hidden">
           <div class="form-row cols-2"><div><label class="form-label">稳定后端<span class="req">*</span></label><select class="selectbox mono"><option value="">选择…</option><option>svc-chat-v1</option><option>svc-chat-api</option></select></div><div><label class="form-label">灰度后端<span class="req">*</span></label><select class="selectbox mono"><option value="">选择…</option><option>svc-chat-v2</option></select></div></div>
           <div class="form-row"><label class="form-label">初始灰度百分比</label><input class="input mono" value="5" style="width:84px;text-align:right"><span class="text-muted" style="font-size:12px;margin-left:6px">%（0–100）</span></div>
           <div class="form-help" style="margin-top:0">灰度发布：1 个稳定后端 + 1 个灰度后端，按灰度百分比放量，支持一键提升 / 回滚。</div>
         </div>`,
      submit: '上线策略',
    }),
  };

  const STEP_TITLES = ['基本信息', '获取凭证', '推送数据', '完成校验'];
  function uploadBody() {
    const m = ART_META[uploadType + 's'] || ART_META.models;
    const specField = uploadType === 'dataset'
      ? `<div><label class="form-label">格式<span class="req">*</span></label><select class="selectbox"><option>parquet</option><option>jsonl</option><option>csv</option></select></div>`
      : uploadType === 'image'
        ? `<div><label class="form-label">用途<span class="req">*</span></label><select class="selectbox"><option>training</option><option>inference</option><option>dev</option></select></div>`
        : `<div><label class="form-label">框架<span class="req">*</span></label><select class="selectbox"><option>pytorch</option><option>safetensors</option><option>onnx</option></select></div>`;
    const pushCmd = uploadType === 'dataset' ? 'aws s3 cp ./data s3://axisml-datasets/team-a/&lt;name&gt;/&lt;version&gt;/ --recursive'
      : 'docker push registry.axisml.io/team-a/&lt;name&gt;:&lt;version&gt;';
    const stepper = `<div class="stepper">${STEP_TITLES.map((t, i) => `${i ? '<span class="step-line"></span>' : ''}<span class="step ${i === uploadStep ? 'active' : (i < uploadStep ? 'done' : '')}"><span class="num">${i < uploadStep ? '✓' : (i + 1)}</span>${t}</span>`).join('')}</div>`;
    const steps = [
      `<div class="wizard-step active">
        <div class="form-row"><label class="form-label">名称<span class="req">*</span></label><input class="input mono" placeholder="选择已有或新建" maxlength="63"><div class="form-help">新建时需符合 DNS-1123：小写字母、数字与连字符（-），以字母或数字开头与结尾。</div></div>
        <div class="form-row cols-2"><div><label class="form-label">版本<span class="opt">OCI tag 规则</span></label><input class="input mono" placeholder="v1"></div>${specField}</div>
        <div class="form-row"><label class="form-label">标签</label><input class="input mono" placeholder="task=chat,lang=zh"></div>
        <div class="form-row"><label class="form-label">描述</label><textarea class="textarea" placeholder="可选"></textarea></div>
      </div>`,
      `<div class="wizard-step active">
        <p class="form-help" style="margin:0 0 14px">点击「初始化上传」获取上传地址与临时凭证（有效期 1h）。</p>
        <button class="btn btn-primary" style="margin-bottom:16px">初始化上传</button>
        <div class="codeblock"><span class="c"># upload endpoint</span>\nregistry.axisml.io/team-a\n<span class="c"># credential expires in</span>\n59m 58s</div>
      </div>`,
      `<div class="wizard-step active">
        <p class="form-help" style="margin:0 0 12px">在本地执行以下命令推送数据：</p>
        <div class="codeblock">${pushCmd}</div>
        <p class="form-help" style="margin-top:14px">推送完成后进入下一步填入 digest 校验。</p>
      </div>`,
      `<div class="wizard-step active">
        <div class="form-row"><label class="form-label">digest<span class="req">*</span></label><input class="input mono" placeholder="sha256:..."></div>
        <p class="form-help">服务端校验通过后版本状态转为 <span class="pill pill-success" style="height:20px">就绪</span>，即可在任务 / 服务中引用。</p>
      </div>`,
    ];
    return stepper + steps[uploadStep];
  }

  function openDrawer(type, artType) {
    if (type === 'upload') {
      uploadStep = 0; uploadType = artType || 'model';
      renderUpload();
    } else {
      const f = FORMS[type]();
      dTitle.textContent = f.title;
      dBody.innerHTML = f.body;
      dFoot.innerHTML = `<span></span><div class="right"><button class="btn" data-drawer-cancel>取消</button><button class="btn btn-primary" data-drawer-cancel>${f.submit}</button></div>`;
    }
    scrim.classList.add('open'); drawer.classList.add('open');
  }
  function renderUpload() {
    const m = ART_META[uploadType + 's'] || ART_META.models;
    dTitle.textContent = `上传${m.display} › ${uploadStep === 0 ? '新版本' : STEP_TITLES[uploadStep]}`;
    dBody.innerHTML = uploadBody();
    const prev = `<button class="btn" data-step-prev ${uploadStep === 0 ? 'disabled' : ''}>上一步</button>`;
    const next = uploadStep === STEP_TITLES.length - 1
      ? `<button class="btn btn-primary" data-drawer-cancel>完成上传</button>`
      : `<button class="btn btn-primary" data-step-next>下一步</button>`;
    dFoot.innerHTML = `${prev}<div class="right"><button class="btn" data-drawer-cancel>取消</button>${next}</div>`;
  }
  function closeDrawer() { scrim.classList.remove('open'); drawer.classList.remove('open'); }

  /* ---------------- Modal (workspace create) ---------------- */
  const mScrim = $('#modalScrim'), modal = $('#modal');
  const mBody = $('#modalBody'), mFoot = $('#modalFoot'), mTitle = $('#modalTitle');
  function openModal(type) {
    const f = FORMS[type]();
    mTitle.textContent = f.title;
    mBody.innerHTML = f.body;
    mFoot.innerHTML = `<button class="btn" data-modal-cancel>取消</button><button class="btn btn-primary" data-modal-cancel>${f.submit}</button>`;
    mScrim.classList.add('open'); modal.classList.add('open');
  }
  function closeModal() { mScrim.classList.remove('open'); modal.classList.remove('open'); }

  /* ---------------- Run trigger dialog (§7.5) ---------------- */
  // 默认按模板直接运行;展开「高级 · 本次覆盖」可临时覆盖版本 / 资源 / 超参。
  function openRunDialog(jobName) {
    const job = findJob(jobName);
    if (!job) return;
    const t = job.tpl;
    const nextN = (job.runs[0] ? job.runs[0].n : 0) + 1;
    mTitle.textContent = `触发运行 · ${job.name}`;
    mBody.innerHTML = `
      <p class="form-help" style="margin:0 0 14px">将创建一次新运行 <code>${job.name}-${nextN}</code>。确认后对引用制品版本预检 <span class="pill pill-success" style="height:19px">Ready</span>，成功即调度并跳转 Run 详情。</p>
      <dl class="kv-grid" style="margin-bottom:6px">
        <dt>资源单元</dt><dd><code>${t.pool}/${t.unit}</code> <span class="text-muted" style="font-size:12px;margin-left:4px">${t.spec}</span></dd>
        <dt>镜像</dt><dd><code>${t.image}</code></dd>
        <dt>副本数</dt><dd><span class="text-mono">${t.replicas}</span></dd>
        <dt>制品引用</dt><dd><div class="tag-list">${t.artifacts.map((a) => `<span class="tag"><span class="k">${a[0]}</span><span class="eq">:</span><span class="v">${a[1]}@${a[2]}</span></span>`).join('')}</div></dd>
      </dl>
      <div class="adv-toggle" data-adv><span class="chev"><svg width="10" height="10" viewBox="0 0 10 10" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M3 2.5 6 5 3 7.5"/></svg></span>高级 · 本次覆盖</div>
      <div class="adv-body" data-adv-body>
        <p class="form-help" style="margin:2px 0 12px">仅影响<strong>本次</strong>运行，不修改 Job 配置；不可改 backend 与 role 拓扑（需在 Job 配置中修改）。</p>
        <div class="form-row cols-2">
          <div><label class="form-label">资源单元</label><select class="selectbox mono"><option value="">沿用 Job 默认（${t.unit}）</option><option>a100-1x-large</option><option>a100-4x-xlarge</option><option>a100-8x-xlarge-ib</option><option>h100-1x-large</option></select></div>
          <div><label class="form-label">副本数</label><input class="input mono" value="${t.replicas}"></div>
        </div>
        <div class="form-row cols-2">
          <div><label class="form-label">模型版本</label><select class="selectbox mono"><option value="">沿用 Job 默认</option><option>llama-7b-base@v1</option><option>llama-7b-base@v2</option></select></div>
          <div><label class="form-label">数据集版本</label><select class="selectbox mono"><option value="">沿用 Job 默认</option><option>dialog-zh-clean@v4</option><option>dialog-zh-clean@v3</option></select></div>
        </div>
        <div class="form-row"><label class="form-label">超参覆盖<span class="opt">命令 / 参数</span></label><textarea class="textarea" placeholder="留空沿用 Job 默认：${t.cmd.join(' ')}"></textarea></div>
      </div>`;
    mFoot.innerHTML = `<button class="btn" data-modal-cancel>取消</button><button class="btn btn-primary" data-run-confirm="${job.name}">确认运行</button>`;
    mScrim.classList.add('open'); modal.classList.add('open');
  }

  // create a new Run from template + jump to its detail
  function triggerRun(jobName) {
    const job = findJob(jobName);
    if (!job) return null;
    const n = (job.runs[0] ? job.runs[0].n : 0) + 1;
    const t = job.tpl;
    const unitShort = t.pool + '/' + (t.unit.match(/-(\d+)x/) ? t.unit.match(/-(\d+)x/)[1] + 'x' : '1x');
    job.runs.unshift({
      n, status: st('success', '运行中'), unit: unitShort, spec: t.spec, rep: String(t.replicas),
      trigger: '张伟', dur: '00:00:03', started: nowStamp(), finished: '—',
      msg: `${t.replicas}/${t.replicas} Pod 拉起中…`,
    });
    job.updated = '刚刚';
    return `${jobName}-${n}`;
  }
  function nowStamp() {
    const d = new Date();
    const p = (x) => String(x).padStart(2, '0');
    return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`;
  }

  /* ---------------- Global event wiring ---------------- */
  // sidebar nav
  $$('.nav-item[data-route], .nav-child[data-route]').forEach((el) => {
    el.addEventListener('click', () => { location.hash = '#/' + el.getAttribute('data-route'); });
  });

  // tabs (delegated; also handles data-tab-go links)
  document.addEventListener('click', (e) => {
    const tab = e.target.closest('.tab');
    if (tab && tab.closest('.tabs')) {
      const tabs = tab.closest('.tabs');
      const key = tab.getAttribute('data-tab');
      $$('.tab', tabs).forEach((t) => t.classList.toggle('active', t === tab));
      const page = tab.closest('.page');
      $$('.tab-panel', page).forEach((p) => p.classList.toggle('hidden', p.getAttribute('data-tab-panel') !== key));
      return;
    }
    const tg = e.target.closest('[data-tab-go]');
    if (tg) {
      const key = tg.getAttribute('data-tab-go');
      const page = tg.closest('.page');
      const tabs = $('.tabs', page);
      const target = $(`.tab[data-tab="${key}"]`, tabs);
      if (target) target.click();
      return;
    }
    // twirl (resource unit expand)
    const tw = e.target.closest('.twirl[data-twirl]');
    if (tw) {
      const key = tw.getAttribute('data-twirl');
      const target = document.querySelector(`tr[data-twirl-target="${key}"]`);
      if (target) { const open = target.classList.toggle('hidden'); tw.classList.toggle('open', !open); }
      return;
    }
    // kpi / recent navigation
    const go = e.target.closest('[data-go]');
    if (go) { location.hash = '#/' + go.getAttribute('data-go'); return; }
    // workspace card → detail page (clicking anywhere except links / buttons)
    const wsCard = e.target.closest('.res-card[data-card-href]');
    if (wsCard && !e.target.closest('a, button')) { location.hash = wsCard.getAttribute('data-card-href'); return; }
    // drawer open
    const dOpen = e.target.closest('[data-drawer]');
    if (dOpen) { openDrawer(dOpen.getAttribute('data-drawer'), dOpen.getAttribute('data-art-type')); return; }
    // modal open (workspace create)
    const mOpen = e.target.closest('[data-modal]');
    if (mOpen) { openModal(mOpen.getAttribute('data-modal')); return; }
    // run trigger dialog (▶ 运行 on job detail / list)
    const runJob = e.target.closest('[data-run-job]');
    if (runJob) { openRunDialog(runJob.getAttribute('data-run-job')); return; }
    // confirm a run from the dialog → create run + jump to its detail
    const runCfm = e.target.closest('[data-run-confirm]');
    if (runCfm) {
      const jobName = runCfm.getAttribute('data-run-confirm');
      const runName = triggerRun(jobName);
      closeModal();
      if (runName) location.hash = `#/jobs/${jobName}/runs/${runName}`;
      return;
    }
    if (e.target.closest('[data-modal-cancel]') || e.target === mScrim) { closeModal(); return; }
    // upload-new-version button on artifact detail → open upload drawer for current kind
    if (e.target.closest('[data-upload-version]')) {
      const k = (location.hash.replace(/^#\//, '').split('/')[0]) || 'models';
      openDrawer('upload', { datasets: 'dataset', models: 'model', images: 'image' }[k] || 'model');
      return;
    }
    // add / remove service port row
    if (e.target.closest('[data-port-add]')) {
      const host = e.target.closest('.modal, .drawer').querySelector('[data-ports]');
      const row = document.createElement('div');
      row.className = 'port-row';
      row.innerHTML = `<input class="input mono" placeholder="名称，如 grpc"><input class="input mono" placeholder="端口，如 8001"><button class="port-rm" data-port-rm title="移除"><svg width="13" height="13" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"><path d="M4 4l8 8M12 4l-8 8"/></svg></button>`;
      host.appendChild(row);
      return;
    }
    const prm = e.target.closest('[data-port-rm]');
    if (prm) { const host = prm.closest('[data-ports]'); if (host && host.children.length > 1) prm.closest('.port-row').remove(); return; }
    // traffic — create form mode segmented control
    const seg = e.target.closest('[data-mode-seg] .seg-btn');
    if (seg) {
      const wrap = seg.closest('.modal, .drawer');
      $$('[data-mode-seg] .seg-btn', wrap).forEach((b) => b.classList.toggle('active', b === seg));
      const mode = seg.getAttribute('data-mode');
      $$('[data-mode-body]', wrap).forEach((b) => b.classList.toggle('hidden', b.getAttribute('data-mode-body') !== mode));
      const help = $('[data-mode-help]', wrap);
      if (help) help.textContent = mode === 'canary'
        ? '灰度：1 个稳定后端 + 1 个灰度后端，按灰度百分比逐步放量，支持一键提升 / 回滚。'
        : '加权：N 个后端按权重平摊流量（Σ=100），可一键置 100 做蓝绿切换。';
      return;
    }
    // traffic — add / remove backend row in create form
    if (e.target.closest('[data-bk-add]')) {
      const host = e.target.closest('.modal, .drawer').querySelector('[data-bk-rows]');
      const row = document.createElement('div');
      row.className = 'bk-row';
      row.innerHTML = `<select class="selectbox mono"><option value="">选择就绪的在线服务…</option><option>svc-embed-a</option><option>svc-embed-b</option><option>svc-chat-api</option></select><input class="input mono" value="0" style="width:84px;text-align:right"><span class="text-muted" style="font-size:12px">%</span><button class="port-rm" data-bk-rm type="button" title="移除"><svg width="13" height="13" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"><path d="M4 4l8 8M12 4l-8 8"/></svg></button>`;
      host.appendChild(row);
      return;
    }
    const bkrm = e.target.closest('[data-bk-rm]');
    if (bkrm) { const host = bkrm.closest('[data-bk-rows]'); if (host && host.children.length > 1) bkrm.closest('.bk-row').remove(); return; }
    // traffic — promote / rollback (canary)
    const promo = e.target.closest('[data-traffic-promote]');
    if (promo) {
      const sec = $('[data-page="traffic-detail"]');
      const sl = $('[data-canary-slider]', sec);
      if (sl) { sl.value = 100; setCanary(sl); }
      flash(promo);
      return;
    }
    const rbk = e.target.closest('[data-traffic-rollback]');
    if (rbk) {
      const sec = $('[data-page="traffic-detail"]');
      const sl = $('[data-canary-slider]', sec);
      if (sl) { sl.value = 0; setCanary(sl); }
      flash(rbk);
      return;
    }
    // traffic — 调整流量 (jump to flow tab on detail page; from list, go to detail)
    const adj = e.target.closest('[data-traffic-adjust]');
    if (adj) {
      const sec = $('[data-page="traffic-detail"]');
      if (sec && sec.classList.contains('active')) {
        const t = $('.tab[data-tab="flow"]', sec);
        if (t) t.click();
        const sl = $('[data-canary-slider]', sec);
        if (sl) sl.focus();
      } else {
        location.hash = '#/traffic/' + adj.getAttribute('data-traffic-adjust');
      }
      return;
    }
    // advanced settings collapsible
    const adv = e.target.closest('[data-adv]');
    if (adv) { adv.classList.toggle('open'); const b = adv.parentElement.querySelector('[data-adv-body]'); if (b) b.classList.toggle('open'); return; }
    // drawer cancel / submit
    if (e.target.closest('[data-drawer-cancel]')) { closeDrawer(); return; }
    if (e.target.closest('[data-step-next]')) { uploadStep = Math.min(STEP_TITLES.length - 1, uploadStep + 1); renderUpload(); return; }
    if (e.target.closest('[data-step-prev]')) { uploadStep = Math.max(0, uploadStep - 1); renderUpload(); return; }
    // switch toggle
    const sw = e.target.closest('[data-switch]');
    if (sw) {
      const on = sw.classList.toggle('on');
      const rev = sw.getAttribute('data-reveal');
      if (rev) { const host = sw.closest('.modal, .drawer') || document; const t = host.querySelector(rev); if (t) t.classList.toggle('hidden', !on); }
      return;
    }
    // range picker
    const rb = e.target.closest('.range-picker button');
    if (rb) { $$('.range-picker button', rb.closest('.range-picker')).forEach((b) => b.classList.toggle('active', b === rb)); return; }
    // view toggle
    const vt = e.target.closest('.view-toggle button');
    if (vt) {
      const grp = vt.closest('.view-toggle').getAttribute('data-view-toggle');
      const v = vt.getAttribute('data-view');
      if (grp === 'artifacts') { viewState.artifacts = v; try { localStorage.setItem('axisml.artifactView', v); } catch (er) {} ['datasets', 'models', 'images'].forEach(renderListIfActive); syncViewToggles('artifacts', v); }
      else { viewState.workspaces = v; try { localStorage.setItem('axisml.workspaceView', v); } catch (er) {} renderList('workspaces'); syncViewToggles('workspaces', v); }
      return;
    }
    // copy
    const cp = e.target.closest('[data-copy]');
    if (cp) { try { navigator.clipboard.writeText(cp.getAttribute('data-copy')); } catch (er) {} flash(cp); return; }
    const pl = e.target.closest('[data-pull]');
    if (pl) { showPull(pl.getAttribute('data-pull')); return; }
    // user menu (avatar dropdown)
    const umenu = $('#umMenu');
    const ab = e.target.closest('#avatarBtn');
    if (ab) { umenu.classList.toggle('open'); e.stopPropagation(); return; }
    if (umenu) {
      // language toggle
      const langB = e.target.closest('#langToggle button');
      if (langB) { setLang(langB.getAttribute('data-val')); e.stopPropagation(); return; }
      // theme segmented (3-way)
      const themeB = e.target.closest('#themeToggle button');
      if (themeB) { applyTheme(themeB.getAttribute('data-val')); e.stopPropagation(); return; }
      // tenant submenu (click toggles flyout on touch; hover handles it on desktop)
      const subHead = e.target.closest('.um-opt.has-sub');
      if (subHead) { $('#tenantSub').classList.toggle('open'); e.stopPropagation(); return; }
      // pick a tenant
      const tOpt = e.target.closest('.um-opt[data-tenant]');
      if (tOpt) {
        $$('.um-opt[data-tenant]', umenu).forEach((o) => o.classList.toggle('current', o === tOpt));
        $('#umCurTenant').textContent = tOpt.getAttribute('data-tenant');
        $('#tenantSub').classList.remove('open');
        umenu.classList.remove('open');
        e.stopPropagation();
        return;
      }
      if (e.target.closest('#signOut')) { umenu.classList.remove('open'); return; }
      if (umenu.classList.contains('open') && !e.target.closest('#umMenu')) { umenu.classList.remove('open'); $('#tenantSub').classList.remove('open'); }
    }
  });

  // ---- Language ----
  function setLang(v) {
    localStorage.setItem('axisml-lang', v);
    $$('#langToggle button').forEach((b) => b.classList.toggle('active', b.getAttribute('data-val') === v));
  }

  // ---- Theme ----
  function applyTheme(mode) {
    localStorage.setItem('axisml-theme', mode);
    let effective = mode;
    if (mode === 'system') {
      effective = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
    }
    document.documentElement.setAttribute('data-theme', effective === 'dark' ? 'dark' : 'light');
    $$('#themeToggle button').forEach((b) => b.classList.toggle('active', b.getAttribute('data-val') === mode));
  }
  (function initPrefs() {
    applyTheme(localStorage.getItem('axisml-theme') || 'light');
    setLang(localStorage.getItem('axisml-lang') || 'zh');
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
      if ((localStorage.getItem('axisml-theme') || 'light') === 'system') applyTheme('system');
    });
  })();

  function renderListIfActive(kind) {
    const sec = $(`.page[data-page="${kind}"]`);
    if (sec) renderList(kind);
  }
  function syncViewToggles(grp, v) {
    $$(`.view-toggle[data-view-toggle="${grp}"]`).forEach((tg) => {
      $$('button', tg).forEach((b) => b.classList.toggle('active', b.getAttribute('data-view') === v));
    });
  }
  function flash(el) {
    const old = el.style.color; el.style.color = 'var(--app-accent)';
    setTimeout(() => { el.style.color = old; }, 600);
  }
  // canary slider → update fill, split labels, headline pct, and backend table
  function setCanary(sl) {
    const c = Math.max(0, Math.min(100, parseInt(sl.value, 10) || 0));
    const sec = sl.closest('.page') || document;
    sl.style.setProperty('--pct', c + '%');
    const pct = $('.canary-pct', sec); if (pct) pct.innerHTML = c + '<small>%</small>';
    const s = $('[data-canary-stable]', sec); if (s) s.textContent = (100 - c) + '%';
    const cn = $('[data-canary-canary]', sec); if (cn) cn.textContent = c + '%';
    // sync backend table bars (first row 稳定, second 灰度)
    const fills = $$('.section .flow-fill', sec);
    if (fills[0]) fills[0].style.width = (100 - c) + '%';
    if (fills[1]) fills[1].style.width = c + '%';
  }
  // weighted editor → live Σ=100 validation
  function setWeightSum(sec) {
    const inputs = $$('[data-wt-input]', sec);
    if (!inputs.length) return;
    let sum = 0; inputs.forEach((i) => { sum += parseInt(i.value, 10) || 0; });
    const out = $('[data-wt-sum]', sec);
    if (out) {
      const ok = sum === 100;
      out.className = 'wt-sum ' + (ok ? 'ok' : 'bad');
      out.textContent = ok ? 'Σ = 100% ✓' : `Σ = ${sum}%（需等于 100%）`;
    }
  }
  function showPull(payload) {
    const [kind, name, ver] = payload.split('|');
    const m = ART_META[kind];
    uploadType = null;
    dTitle.textContent = `拉取命令 · ${ver}`;
    dBody.innerHTML = `<p class="form-help" style="margin:0 0 12px">在本地执行以下命令拉取该版本（临时凭证有效期 1h）：</p>
      <div class="codeblock">${m.pull(name, ver)}</div>
      <p class="form-help" style="margin-top:14px">digest <span class="digest">sha256:a1b2c3d4…</span> · 不可变引用。</p>`;
    dFoot.innerHTML = `<span></span><div class="right"><button class="btn btn-primary" data-drawer-cancel>关闭</button></div>`;
    scrim.classList.add('open'); drawer.classList.add('open');
  }

  // unit spec readout (delegated change)
  document.addEventListener('change', (e) => {
    const u = e.target.closest('select[data-unit]');
    if (u) {
      const ro = u.closest('.form-row').querySelector('[data-spec-readout]');
      const spec = UNIT_SPECS[u.value];
      if (ro) {
        ro.innerHTML = spec
          ? `<span class="rl">requests / limits</span>` + spec.split(' ').map((kv) => { const [k, v] = kv.split('='); return `<span class="tag"><span class="k">${k}</span><span class="eq">=</span><span class="v">${v}</span></span>`; }).join(' ')
          : `<span class="rl">规格</span><span class="text-muted">选择资源单元后只读展示 requests / limits</span>`;
      }
    }
  });

  // traffic — canary slider + weighted sum (delegated input)
  document.addEventListener('input', (e) => {
    const sl = e.target.closest('[data-canary-slider]');
    if (sl) { setCanary(sl); return; }
    const wt = e.target.closest('[data-wt-input]');
    if (wt) { setWeightSum(wt.closest('.page') || document); return; }
  });

  document.addEventListener('keydown', (e) => { if (e.key === 'Escape') { closeDrawer(); closeModal(); } });
  // initial view toggle sync + route
  syncViewToggles('artifacts', viewState.artifacts);
  syncViewToggles('workspaces', viewState.workspaces);
  window.addEventListener('hashchange', applyRoute);
  applyRoute();

  /* ---------------- Bridge for interactions.js ---------------- */
  // Expose the minimum surface the interaction layer needs to mutate
  // state and re-render, without leaking the whole module.
  window.AX = {
    DATA: DATA,
    ART_META: ART_META,
    ART_DISPLAY: ART_DISPLAY,
    renderList: renderList,
    renderListIfActive: renderListIfActive,
    openDrawer: openDrawer,
    openModal: openModal,
    openRunDialog: openRunDialog,
    findJob: findJob,
    findRun: findRun,
    renderJobDetail: renderJobDetail,
    renderRunDetail: renderRunDetail,
    st: st,
  };
  document.dispatchEvent(new Event('ax:ready'));
})();
