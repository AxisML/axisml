/* =====================================================================
   AxisML console — interaction layer
   Adds the "live product" feel on top of app.js:
     • toast notifications      • confirmation dialogs (with blocking info)
     • generic form modal        • optimistic list/detail mutations
     • working filters & pager    • notification center + topbar polish
   Reads app.js state through the window.AX bridge; renders its own
   chrome (toasts / dialogs) so it never collides with the base app.
   ===================================================================== */
(function () {
  'use strict';

  const $  = (s, r = document) => r.querySelector(s);
  const $$ = (s, r = document) => Array.from(r.querySelectorAll(s));
  const AX = () => window.AX || {};

  /* ---------- icon set (inline, stroke=currentColor) ---------- */
  const ICO = {
    check:  '<svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><circle cx="8" cy="8" r="6.5"/><path d="M5.2 8.2 7 10l3.8-4"/></svg>',
    info:   '<svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="8" cy="8" r="6.5"/><path d="M8 7v4M8 5h.01"/></svg>',
    warn:   '<svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M8 2.2 14.5 13.5H1.5z"/><path d="M8 6.6v3M8 11.4h.01"/></svg>',
    danger: '<svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="8" cy="8" r="6.5"/><path d="M5.5 5.5l5 5M10.5 5.5l-5 5"/></svg>',
    x:      '<svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"><path d="M4 4l8 8M12 4l-8 8"/></svg>',
    block:  '<svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="8" cy="8" r="6.5"/><path d="M3.6 3.6l8.8 8.8"/></svg>',
    dot:    '<svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="8" cy="8" r="6.5"/><path d="M8 5.5v3.2M8 11h.01"/></svg>',
  };

  /* =========================================================
     Toasts
     ========================================================= */
  let toastWrap;
  function toast(title, opts = {}) {
    if (!toastWrap) { toastWrap = document.createElement('div'); toastWrap.className = 'toast-wrap'; document.body.appendChild(toastWrap); }
    const type = opts.type || 'success';
    const el = document.createElement('div');
    el.className = 'toast';
    el.innerHTML =
      `<span class="toast-ico ${type}">${ICO[type] || ICO.check}</span>` +
      `<div class="toast-body"><div class="toast-title">${title}</div>${opts.desc ? `<div class="toast-desc">${opts.desc}</div>` : ''}</div>` +
      `<button class="toast-x" aria-label="关闭">${ICO.x}</button>`;
    toastWrap.appendChild(el);
    requestAnimationFrame(() => el.classList.add('in'));
    const close = () => {
      if (el.dataset.closing) return; el.dataset.closing = '1';
      el.classList.remove('in'); el.classList.add('out');
      setTimeout(() => el.remove(), 260);
    };
    el.querySelector('.toast-x').addEventListener('click', close);
    if (opts.sticky !== true) setTimeout(close, opts.duration || 3600);
    return close;
  }

  /* =========================================================
     Self-contained modal (reuses .modal / .scrim styles)
     ========================================================= */
  let scrim, modal, mTitle, mBody, mFoot, escBound = false;
  function ensureModal() {
    if (modal) return;
    scrim = document.createElement('div'); scrim.className = 'scrim';
    modal = document.createElement('div'); modal.className = 'modal'; modal.setAttribute('role', 'dialog');
    modal.innerHTML =
      '<div class="modal-head"><h2 class="modal-title"></h2>' +
      '<button class="drawer-close" data-ix-close title="关闭">' + ICO.x + '</button></div>' +
      '<div class="modal-body"></div><div class="modal-foot"></div>';
    document.body.appendChild(scrim); document.body.appendChild(modal);
    mTitle = $('.modal-title', modal); mBody = $('.modal-body', modal); mFoot = $('.modal-foot', modal);
    scrim.addEventListener('click', closeModal);
    if (!escBound) { escBound = true; document.addEventListener('keydown', (e) => { if (e.key === 'Escape') closeModal(); }); }
  }
  function openModal(title, bodyHTML, footHTML) {
    ensureModal();
    mTitle.textContent = title;
    mBody.innerHTML = bodyHTML;
    mFoot.innerHTML = footHTML;
    scrim.classList.add('open'); modal.classList.add('open');
    const first = mBody.querySelector('input,select,textarea,button');
    if (first) setTimeout(() => first.focus(), 60);
  }
  function closeModal() { if (modal) { scrim.classList.remove('open'); modal.classList.remove('open'); } }

  /* =========================================================
     Confirm dialog (destructive / blocking actions)
     opts: { title, message, lines:[{kind:'info'|'block', html}],
             check:'label', danger, confirmText, cancelText,
             fatal (disables confirm), onConfirm(payload) }
     ========================================================= */
  function confirm(opts) {
    const lines = (opts.lines || []).map((l) =>
      `<div class="cfm-line ${l.kind}"><span class="ci">${l.kind === 'block' ? ICO.block : ICO.info}</span><span>${l.html}</span></div>`).join('');
    const block = lines ? `<div class="cfm-block">${lines}</div>` : '';
    const check = opts.check ? `<label class="cfm-check"><input type="checkbox" data-cfm-check ${opts.checkDefault ? 'checked' : ''}>${opts.check}</label>` : '';
    const body = `<p class="cfm-msg">${opts.message || ''}</p>${block}${check}`;
    const danger = opts.danger !== false;
    const confirmCls = danger ? 'btn btn-sm btn-danger' : 'btn btn-sm btn-primary';
    const disabled = opts.fatal ? 'disabled style="opacity:.45;cursor:not-allowed"' : '';
    const foot =
      `<button class="btn btn-sm" data-ix-close>${opts.cancelText || '取消'}</button>` +
      (opts.fatal ? '' : `<button class="${confirmCls}" data-cfm-ok ${disabled}>${opts.confirmText || '确认删除'}</button>`);
    openModal(opts.title || '确认操作', body, foot);
    if (danger) mTitle.style.color = 'var(--app-danger-deep)'; else mTitle.style.color = '';
    const ok = $('[data-cfm-ok]', modal);
    if (ok) ok.addEventListener('click', () => {
      const checked = !!$('[data-cfm-check]', modal) && $('[data-cfm-check]', modal).checked;
      closeModal();
      if (opts.onConfirm) opts.onConfirm({ checked });
    }, { once: true });
  }

  /* =========================================================
     Generic form modal
     fields: [{ label, key, type:'text|textarea|select|number|mono',
                value, placeholder, options:[], help, req, disabled,
                suffix, cols }]
     ========================================================= */
  function fieldHTML(f) {
    const lab = `<label class="form-label">${f.label || ''}${f.req ? '<span class="req">*</span>' : ''}${f.opt ? `<span class="opt">${f.opt}</span>` : ''}</label>`;
    const dis = f.disabled ? 'disabled' : '';
    const mono = (f.type === 'mono' || f.type === 'number') ? ' mono' : '';
    let ctrl;
    if (f.type === 'textarea') {
      ctrl = `<textarea class="textarea" data-fk="${f.key}" placeholder="${f.placeholder || ''}" ${dis} style="font-family:var(--fnt-sans)">${f.value || ''}</textarea>`;
    } else if (f.type === 'select') {
      ctrl = `<select class="selectbox${f.mono ? ' mono' : ''}" data-fk="${f.key}" ${dis}>` +
        (f.options || []).map((o) => { const v = typeof o === 'object' ? o.value : o; const t = typeof o === 'object' ? o.label : o; return `<option ${String(v) === String(f.value) ? 'selected' : ''} value="${v}">${t}</option>`; }).join('') + '</select>';
    } else if (f.type === 'number') {
      ctrl = `<input class="input mono ix-num" type="number" data-fk="${f.key}" value="${f.value != null ? f.value : ''}" ${f.min != null ? `min="${f.min}"` : ''} ${f.max != null ? `max="${f.max}"` : ''} ${dis}>${f.suffix ? `<span class="text-muted" style="font-size:12px;margin-left:8px">${f.suffix}</span>` : ''}`;
    } else {
      ctrl = `<input class="input${mono}" data-fk="${f.key}" value="${f.value != null ? f.value : ''}" placeholder="${f.placeholder || ''}" ${dis}>`;
    }
    const help = f.help ? `<div class="form-help">${f.help}</div>` : '';
    if (f.raw) return f.raw;
    return `<div class="form-row">${lab}${ctrl}${help}</div>`;
  }
  function formModal(opts) {
    const body = (opts.intro ? `<p class="ix-inline-note" style="margin:-2px 0 16px">${opts.intro}</p>` : '') +
      (opts.fields || []).map(fieldHTML).join('');
    const foot =
      `<button class="btn" data-ix-close>${opts.cancelText || '取消'}</button>` +
      `<button class="btn btn-primary" data-form-ok>${opts.submitText || '保存'}</button>`;
    openModal(opts.title, body, foot);
    mTitle.style.color = '';
    const ok = $('[data-form-ok]', modal);
    ok.addEventListener('click', () => {
      const vals = {};
      $$('[data-fk]', modal).forEach((el) => { vals[el.getAttribute('data-fk')] = el.value; });
      closeModal();
      if (opts.onSubmit) opts.onSubmit(vals);
    }, { once: true });
  }

  /* =========================================================
     State helpers (optimistic mutation through the bridge)
     ========================================================= */
  const ENTITY = {
    workspaces: '工作区', jobs: '任务', services: '服务', traffic: '策略',
    datasets: '数据集', models: '模型', images: '镜像', pools: '资源池', tenants: '租户',
    'workspace-detail': '工作区', 'job-detail': 'Job', 'service-detail': '服务',
    'traffic-detail': '策略', 'artifact-detail': '资产', 'pool-detail': '资源池', 'tenant-detail': '租户',
    'job-run-detail': 'Run',
  };
  const st = (tone, label) => ({ tone, label });

  function activePage() { return $('.page.active'); }
  function pageId() { const p = activePage(); return p ? p.getAttribute('data-page') : ''; }
  function listKindOf(page) { const k = page.getAttribute('data-page'); return AX().DATA && AX().DATA[k] ? k : null; }
  function detailName(page) { const t = $('.detail-title', page); return t ? t.textContent.trim() : ''; }
  function rowName(el) {
    const tr = el.closest('tr');
    if (tr) { const a = tr.querySelector('.row-link, .res-card-name'); if (a) return a.textContent.trim(); const m = tr.querySelector('.text-mono'); if (m) return m.textContent.trim(); }
    const card = el.closest('.res-card');
    if (card) { const a = card.querySelector('.res-card-name'); if (a) return a.textContent.trim(); }
    return '';
  }
  function mutate(kind, name, fn) {
    const arr = AX().DATA && AX().DATA[kind]; if (!arr) return false;
    const it = arr.find((x) => x.name === name); if (!it) return false;
    fn(it); AX().renderList(kind); return true;
  }
  function removeFromData(kind, name) {
    const D = AX().DATA; if (!D || !D[kind]) return;
    D[kind] = D[kind].filter((x) => x.name !== name); AX().renderList(kind);
  }
  function animateRowRemove(el, after) {
    const row = el.closest('tr') || el.closest('.res-card');
    if (!row) { if (after) after(); return; }
    row.classList.add('row-removing');
    setTimeout(() => { if (after) after(); else row.remove(); }, 270);
  }

  /* =========================================================
     Action behaviours
     ========================================================= */
  // ---- compute resources: stop / start / scale / delete / cancel / resubmit / open
  function doStop(kind, name) {
    const noun = ENTITY[kind] || '资源';
    confirm({
      title: `停止${noun} ${name}`, danger: false, confirmText: '停止',
      message: `将副本数缩到 0；${kind === 'services' ? '对外访问会中断，' : ''}已挂载的数据卷保留，可随时重新启动。`,
      onConfirm() {
        if (!mutate(kind, name, (it) => { it.status = st('muted', '已停止'); it.rep = '0/0'; })) updateDetailStatus('已停止', 'muted');
        toast(`已停止 ${noun}`, { type: 'info', desc: `<code>${name}</code> 副本已缩至 0` });
      },
    });
  }
  function doStart(kind, name) {
    const noun = ENTITY[kind] || '资源';
    const onList = mutate(kind, name, (it) => { it.status = st('warn', '启动中'); it.rep = '0/1'; });
    if (!onList) updateDetailStatus('启动中', 'warn');
    toast(`正在启动 ${noun}`, { type: 'info', desc: `<code>${name}</code> 正在拉起副本…` });
    setTimeout(() => {
      if (!mutate(kind, name, (it) => { it.status = st('success', kind === 'services' ? '就绪' : '运行中'); it.rep = kind === 'services' ? '2/2' : '1/1'; }))
        updateDetailStatus(kind === 'services' ? '就绪' : '运行中', 'success');
      toast(`${noun}已就绪`, { type: 'success', desc: `<code>${name}</code> 已恢复运行` });
    }, 1100);
  }
  function doScale(kind, name) {
    const arr = AX().DATA && AX().DATA[kind];
    const it = arr && arr.find((x) => x.name === name);
    const cur = it ? parseInt(String(it.rep).split('/')[1] || it.rep, 10) || 2 : 2;
    formModal({
      title: `扩缩容 · ${name}`, submitText: '应用',
      intro: '在线服务唯一可在线变更的字段是副本数，其余规格创建后不可变。',
      fields: [
        { label: '当前副本', type: 'mono', value: `${cur}`, disabled: true },
        { label: '目标副本数', key: 'replicas', type: 'number', value: cur, min: 0, max: 32, req: true, suffix: '个副本', help: '设为 0 等同于停止服务。' },
      ],
      onSubmit(v) {
        const n = Math.max(0, parseInt(v.replicas, 10) || 0);
        if (!mutate(kind, name, (item) => { item.rep = `${n}/${n}`; item.status = n === 0 ? st('muted', '已停止') : st('success', '就绪'); }))
          { const cell = $$('.kv-grid dd').find((d) => /就绪/.test(d.textContent)); if (cell) cell.innerHTML = `<span class="text-mono">${n} / ${n}</span> 就绪`; }
        toast('已下发扩缩容', { type: 'success', desc: `<code>${name}</code> 目标副本数 → ${n}` });
      },
    });
  }
  function doCancel(name) {
    confirm({
      title: `取消任务 ${name}`, danger: true, confirmText: '取消任务', cancelText: '返回',
      message: '正在运行的 Pod 会立即收到终止信号，已产生的中间结果不会自动保存。',
      onConfirm() {
        if (!mutate('jobs', name, (it) => { it.status = st('muted', '已取消'); it.dur = it.dur === '—' ? '—' : it.dur; }))
          updateDetailStatus('已取消', 'muted');
        toast('已取消任务', { type: 'info', desc: `<code>${name}</code> 进入终止流程` });
      },
    });
  }
  function doResubmit(name) {
    toast('已重新提交', { type: 'success', desc: `<code>${name}</code> 按原配置重建，新任务进入排队` });
    mutate('jobs', name, () => {});
  }

  /* ---- Job (template) → Run (运行) two-level actions ---- */
  // jobName + runName helpers: on job-detail the job is the title; on run-detail derive from hash.
  function curJobName() {
    const parts = (location.hash.replace(/^#\//, '')).split('/');
    return parts[0] === 'jobs' ? parts[1] : '';
  }
  function rerenderJobViews(jobName) {
    const pid = pageId();
    if (pid === 'job-detail') AX().renderJobDetail(jobName);
    else if (pid === 'job-run-detail') {
      const parts = (location.hash.replace(/^#\//, '')).split('/');
      AX().renderRunDetail(parts[1], parts[3]);
    }
  }
  function mutateRun(jobName, runName, fn) {
    const job = AX().findJob(jobName);
    const r = AX().findRun(job, runName);
    if (!job || !r) return false;
    fn(r, job); job.updated = '刚刚';
    return true;
  }
  function doEditTemplate(jobName) {
    toast('编辑 Job', { type: 'info', desc: `<code>${jobName}</code> 的修改只影响之后触发的运行` });
    AX().openDrawer('job');
    const t = document.querySelector('#drawerTitle'); if (t) t.textContent = `编辑 Job · ${jobName}`;
  }
  function doCancelRun(jobName, runName) {
    confirm({
      title: `取消运行 ${runName}`, danger: true, confirmText: '取消运行', cancelText: '返回',
      message: '正在运行的 Pod 会立即收到终止信号，已产生的中间结果不会自动保存。',
      onConfirm() {
        mutateRun(jobName, runName, (r) => { r.status = st('muted', '已取消'); r.finished = r.finished === '—' ? '处理中' : r.finished; r.msg = '用户主动取消'; });
        rerenderJobViews(jobName);
        toast('已取消运行', { type: 'info', desc: `<code>${runName}</code> 进入终止流程` });
      },
    });
  }
  function doDeleteRun(jobName, runName) {
    confirm({
      title: `删除运行 ${runName}`, danger: true, confirmText: '删除',
      message: '删除该次运行的记录与 Pod，操作不可恢复。仅终态 Run 可删除。',
      onConfirm() {
        const job = AX().findJob(jobName);
        if (job) { const n = parseInt(String(runName).split('-').pop(), 10); job.runs = job.runs.filter((r) => r.n !== n); job.updated = '刚刚'; }
        const pid = pageId();
        if (pid === 'job-run-detail') { location.hash = '#/jobs/' + jobName; }
        else rerenderJobViews(jobName);
        toast('已删除运行', { type: 'info', desc: `<code>${runName}</code> 已移除` });
      },
    });
  }

  // ---- traffic policy: enable / disable ----
  function doEnableTraffic(name) {
    const arr = AX().DATA && AX().DATA.traffic;
    const it = arr && arr.find((x) => x.name === name);
    if (!it) return;
    // a policy that was never ready (missing backend) can't be enabled from the list
    if (it.url === '—' && it.status.label === '未就绪') {
      toast('策略未就绪，无法启用', { type: 'warn', desc: `<code>${name}</code> 后端未配置完整，请在详情页补全后再启用` });
      return;
    }
    const canary = it.mode === '灰度';
    mutate('traffic', name, (x) => { x.status = canary ? st('warn', '灰度中') : st('success', '生效中'); });
    toast('已启用流量策略', { type: 'success', desc: `<code>${name}</code> 已开始按规则分发流量` });
  }
  function doDisableTraffic(name) {
    confirm({
      title: `禁用流量策略 ${name}`, danger: false, confirmText: '禁用',
      message: '对外入口将停止按本策略分发流量，配置保留，可随时重新启用。',
      onConfirm() {
        mutate('traffic', name, (x) => { x.status = st('muted', '已禁用'); });
        toast('已禁用流量策略', { type: 'info', desc: `<code>${name}</code> 已停止分发流量` });
      },
    });
  }
  function doOpen(name, tool) {
    toast(`正在打开 ${tool || '工作区'}`, { type: 'info', desc: `<code>${name}</code> 的 ${tool || '开发环境'} 将在新标签页打开` });
  }
  function doDelete(kind, name, fromDetail) {
    const noun = ENTITY[kind] || '资源';
    const cfg = { title: `删除${noun} ${name}`, confirmText: '删除', danger: true };
    if (kind === 'workspaces' || kind === 'workspace-detail') {
      cfg.message = '删除后不可恢复。';
      cfg.check = '一并删除数据卷 PVC（ws-data）'; cfg.checkDefault = true;
    } else if (kind === 'pools' || kind === 'pool-detail') {
      const busy = ['gpu-a100', 'gpu-h100'].includes(name);
      cfg.lines = [
        { kind: 'info', html: '池内资源单元将随资源池级联删除（不阻断）。' },
      ];
      if (busy) {
        cfg.lines.push({ kind: 'block', html: '<b>5</b> 个活跃任务、<b>2</b> 个活跃服务正在引用本池，例如 <code>team-a/train-llm-7b</code>' });
        cfg.fatal = true; cfg.message = '存在活跃负载引用本池，无法删除。请先清空后重试。';
      } else {
        cfg.message = '删除资源池将级联移除其下所有资源单元，操作不可恢复。';
      }
    } else if (kind === 'tenants' || kind === 'tenant-detail') {
      cfg.lines = [{ kind: 'block', html: '该租户仍有 <b>12</b> 名成员，删除前需先在「成员」中移除全部成员。' }];
      cfg.fatal = true; cfg.message = '租户存在残留成员，无法删除。';
    } else {
      cfg.message = '删除后不可恢复。';
    }
    cfg.onConfirm = ({ checked }) => {
      const D = AX().DATA;
      if (fromDetail) {
        toast(`已删除${noun}`, { type: 'info', desc: `<code>${name}</code> 已移除` });
        const back = { 'workspace-detail': 'workspaces', 'job-detail': 'jobs', 'service-detail': 'services', 'traffic-detail': 'traffic', 'artifact-detail': artifactKind(), 'pool-detail': 'pools', 'tenant-detail': 'tenants' }[kind];
        if (back && D && D[back]) removeFromData(back, name);
        if (back) location.hash = '#/' + back;
      } else if (D && D[kind]) {
        removeFromData(kind, name);
        toast(`已删除${noun}`, { type: 'info', desc: `<code>${name}</code> 已移除${checked ? ' · 数据卷已回收' : ''}` });
      } else {
        // static DOM list
        const link = curClickEl;
        animateRowRemove(link, null);
        toast(`已删除${noun}`, { type: 'info', desc: `<code>${name}</code> 已移除` });
      }
    };
    confirm(cfg);
  }

  function updateDetailStatus(label, tone) {
    const page = activePage(); if (!page) return;
    const p = $('.detail-title-row .pill', page);
    if (p) { p.className = 'pill pill-' + tone; p.textContent = label; }
  }
  function artifactKind() { return (location.hash.replace(/^#\//, '').split('/')[0]) || 'models'; }

  // ---- edit metadata (display name + description) ----
  function doEditMeta(kind, name) {
    const noun = ENTITY[kind] || '资源';
    formModal({
      title: `编辑${noun} · ${name}`, submitText: '保存',
      fields: [
        { label: '名称', type: 'mono', value: name, disabled: true, opt: '创建后不可变' },
        { label: '显示名', key: 'display', type: 'text', value: '', placeholder: '可选，展示用名称' },
        { label: '描述', key: 'desc', type: 'textarea', placeholder: '可选' },
      ],
      onSubmit() { toast('已保存', { type: 'success', desc: `<code>${name}</code> 元数据已更新` }); },
    });
  }

  // ---- tenant suspend / resume ----
  function doSuspend(name, el) {
    const isResume = el && el.textContent.trim() === '恢复';
    if (isResume) {
      el.textContent = '暂停';
      const tr = el.closest('tr'); if (tr) { const pill = tr.querySelector('.pill'); if (pill) { pill.className = 'pill pill-success'; pill.textContent = 'Active'; } }
      else updateDetailStatus('Active', 'success');
      toast('已恢复租户', { type: 'success', desc: `<code>${name}</code> 恢复运行，提交入口已解锁` });
      return;
    }
    confirm({
      title: `暂停租户 ${name}`, danger: false, confirmText: '暂停',
      message: '暂停后锁定该租户的新建提交入口（任务 / 服务 / 工作区）；已运行的工作负载继续运行，配额与成员保留。',
      onConfirm() {
        if (el) {
          el.textContent = '恢复';
          const tr = el.closest('tr'); if (tr) { const pill = tr.querySelector('.pill'); if (pill) { pill.className = 'pill pill-warn'; pill.textContent = 'Suspended'; } }
          else updateDetailStatus('Suspended', 'warn');
        } else updateDetailStatus('Suspended', 'warn');
        toast('已暂停租户', { type: 'warn', desc: `<code>${name}</code> 新建提交入口已锁定` });
      },
    });
  }

  // ---- members ----
  function doAddMember() {
    formModal({
      title: '添加成员', submitText: '添加',
      fields: [
        { label: '用户名', key: 'user', type: 'mono', placeholder: 'zhang.wei', req: true, help: '需为平台已存在的用户名（users.username）。' },
        { label: '角色', key: 'role', type: 'select', value: 'user', options: ['user', 'tenant-admin'], help: '不允许绑定 system-admin。' },
      ],
      onSubmit(v) {
        const user = (v.user || '').trim(); if (!user) { toast('请输入用户名', { type: 'warn' }); return; }
        const tbody = $('.tab-panel[data-tab-panel="members"] tbody'); if (tbody) {
          const roleCls = v.role === 'tenant-admin' ? 'role-admin' : 'role-user';
          const tr = document.createElement('tr');
          tr.innerHTML = `<td class="text-mono">${user}</td><td>—</td><td class="text-muted text-mono">${user}@axisml.io</td>` +
            `<td><span class="role-pill ${roleCls}">${v.role}</span></td><td class="text-muted text-mono">${today()}</td>` +
            `<td class="actions"><a class="row-action-link">改角色</a><a class="row-action-link danger">移除</a></td>`;
          tbody.insertBefore(tr, tbody.firstChild);
        }
        toast('已添加成员', { type: 'success', desc: `<code>${user}</code> · ${v.role}` });
      },
    });
  }
  function doChangeRole(name, el) {
    const tr = el.closest('tr'); const cur = tr ? (tr.querySelector('.role-pill') || {}).textContent : 'user';
    formModal({
      title: `修改角色 · ${name}`, submitText: '保存',
      fields: [{ label: '角色', key: 'role', type: 'select', value: (cur || 'user').trim(), options: ['user', 'tenant-admin'] }],
      onSubmit(v) {
        if (tr) { const pill = tr.querySelector('.role-pill'); if (pill) { pill.className = 'role-pill ' + (v.role === 'tenant-admin' ? 'role-admin' : 'role-user'); pill.textContent = v.role; } }
        toast('已更新角色', { type: 'success', desc: `<code>${name}</code> → ${v.role}` });
      },
    });
  }
  function doRemoveMember(name, el) {
    const tr = el.closest('tr');
    const isAdmin = tr && /role-admin/.test(tr.innerHTML);
    const adminCount = $$('.tab-panel[data-tab-panel="members"] .role-admin').length;
    if (isAdmin && adminCount <= 1) {
      confirm({ title: '无法移除', fatal: true, danger: true, message: '这是该租户最后一个 tenant-admin。请先指定其他租户管理员后再移除。', cancelText: '知道了' });
      return;
    }
    confirm({
      title: `移除成员 ${name}`, confirmText: '移除', danger: true,
      message: '移除后该用户将失去本租户的全部访问权限。',
      onConfirm() { animateRowRemove(el, () => tr && tr.remove()); toast('已移除成员', { type: 'info', desc: `<code>${name}</code>` }); },
    });
  }

  // ---- quotas ----
  function quotaFields(name) {
    return [
      { label: '配额名', key: 'qname', type: 'mono', value: name || '', disabled: !!name, opt: name ? '(pool, name) 创建后不可变' : '', placeholder: 'default', req: !name },
      { label: '资源池', key: 'pool', type: 'select', value: 'gpu-a100', disabled: !!name, options: ['gpu-a100', 'gpu-h100', 'cpu-medium', 'gpu-l40s', 'cpu-large'] },
      { raw: '<div class="form-row cols-2"><div><label class="form-label">min</label><input class="input mono" data-fk="min" placeholder="cpu=10,gpu=2"></div><div><label class="form-label">max</label><input class="input mono" data-fk="max" placeholder="cpu=40,gpu=4"></div></div>' },
    ];
  }
  function doQuota(name, el) {
    formModal({
      title: name ? `编辑配额 · ${name}` : '新增配额', submitText: name ? '保存' : '创建',
      fields: quotaFields(name),
      onSubmit(v) {
        if (!name && !v.qname) { toast('请输入配额名', { type: 'warn' }); return; }
        toast(name ? '已更新配额' : '已新增配额', { type: 'success', desc: `<code>${name || v.qname}</code>` });
      },
    });
  }
  function doDeleteQuota(name, el) {
    confirm({
      title: `删除配额 ${name}`, confirmText: '删除', danger: true,
      message: '删除后该配额下的可分配资源立即释放，正在使用的工作负载不受影响但无法再新增。',
      onConfirm() { animateRowRemove(el, () => { const tr = el.closest('tr'); if (tr) tr.remove(); }); toast('已删除配额', { type: 'info', desc: `<code>${name}</code>` }); },
    });
  }

  // ---- resource units ----
  function doUnit(name, el) {
    formModal({
      title: name ? `编辑资源单元 · ${name}` : '新建资源单元', submitText: name ? '保存' : '创建',
      intro: '命名约定 <code>&lt;accelerator&gt;[-&lt;count&gt;x]-&lt;tier&gt;[-&lt;variant&gt;]</code>，由 cluster-manager 兜底校验。',
      fields: [
        { label: '名称', key: 'uname', type: 'mono', value: name || '', disabled: !!name, placeholder: 'a100-1x-large', req: !name, opt: name ? '创建后不可变' : '' },
        { raw: '<div class="form-row cols-2"><div><label class="form-label">requests<span class="req">*</span></label><input class="input mono" data-fk="req" placeholder="cpu=8,mem=64Gi,gpu=1"></div><div><label class="form-label">limits</label><input class="input mono" data-fk="lim" placeholder="缺省沿用 requests"></div></div>' },
        { label: '额外节点选择器', key: 'sel', type: 'mono', placeholder: 'axisml.io/network=ib', help: '与资源池选择器合并（pool ⊕ unit，Pool 优先）。' },
      ],
      onSubmit(v) {
        if (!name && !v.uname) { toast('请输入资源单元名', { type: 'warn' }); return; }
        toast(name ? '已更新资源单元' : '已新建资源单元', { type: 'success', desc: `<code>${name || v.uname}</code>` });
      },
    });
  }
  function doDeleteUnit(name, el) {
    confirm({
      title: `删除资源单元 ${name}`, confirmText: '删除', danger: true,
      message: '删除前会校验是否仍被活跃负载引用。',
      lines: [{ kind: 'info', html: '当前没有活跃 Job / Service 引用 <code>' + name + '</code>，可安全删除。' }],
      onConfirm() { animateRowRemove(el, () => { const tr = el.closest('tr'); if (tr) tr.remove(); }); toast('已删除资源单元', { type: 'info', desc: `<code>${name}</code>` }); },
    });
  }

  // ---- create pool / tenant ----
  function doNewPool() {
    formModal({
      title: '新建资源池', submitText: '创建资源池',
      fields: [
        { label: '名称', key: 'name', type: 'mono', placeholder: 'gpu-a100', req: true, help: '符合 DNS-1123，创建后不可变。' },
        { label: '描述', key: 'desc', type: 'text', placeholder: '可选' },
        { label: '节点选择器', key: 'sel', type: 'mono', placeholder: 'nvidia.com/gpu.product=A100-SXM4-80GB', req: true, help: 'K=V，多项以逗号分隔。Node label / taint 由管理员经 kubectl 维护，UI 不下发。' },
        { label: '容忍配置', key: 'tol', type: 'mono', placeholder: 'nvidia.com/gpu:Exists:NoSchedule' },
      ],
      onSubmit(v) { if (!v.name) { toast('请输入名称', { type: 'warn' }); return; } toast('已创建资源池', { type: 'success', desc: `<code>${v.name}</code> · 去详情页添加资源单元` }); },
    });
  }
  function doNewTenant() {
    formModal({
      title: '新建租户', submitText: '创建租户',
      fields: [
        { label: '显示名', key: 'display', type: 'text', placeholder: '推理团队', req: true },
        { label: '名称', key: 'name', type: 'mono', placeholder: 'team-a', req: true, help: 'DNS-1123，创建后不可变。' },
        { label: '业务线', key: 'biz', type: 'select', value: 'infra', options: ['infra', 'recsys', 'search', 'platform'] },
        { label: '命名空间', key: 'ns', type: 'mono', placeholder: '默认同名称', help: '渲染目标 K8s namespace，创建后不可变。' },
        { label: '描述', key: 'desc', type: 'textarea', placeholder: '可选' },
      ],
      onSubmit(v) { if (!v.display || !v.name) { toast('请填写显示名与名称', { type: 'warn' }); return; } toast('已创建租户', { type: 'success', desc: `<code>${v.name}</code> · 初始化命名空间与配额中…` }); },
    });
  }

  // ---- artifact version actions ----
  function doDownloadVersion(ver) { toast('开始下载', { type: 'info', desc: `${artifactKind()} 版本 <code>${ver || ''}</code> · 13.4GB` }); }
  function doDeleteVersion(ver, el) {
    confirm({
      title: `删除版本 ${ver}`, confirmText: '删除', danger: true,
      message: '该版本将被标记删除并进入回收流程，引用此 digest 的任务 / 服务将无法再拉取。',
      onConfirm() { animateRowRemove(el, () => { const tr = el.closest('tr'); if (tr) tr.remove(); }); toast('已删除版本', { type: 'info', desc: `<code>${ver}</code>` }); },
    });
  }
  function doUploadVersion(kind) { if (AX().openDrawer) AX().openDrawer('upload', { datasets: 'dataset', models: 'model', images: 'image' }[kind] || 'model'); }

  function today() { const d = new Date(); const p = (n) => String(n).padStart(2, '0'); return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`; }

  /* =========================================================
     Click dispatcher
     ========================================================= */
  let curClickEl = null;
  // attrs that app.js already owns — never double-handle
  const OWNED = '[data-copy],[data-pull],[data-tab-go],[data-traffic-promote],[data-traffic-rollback],[data-traffic-adjust],[data-upload-version],[data-drawer],[data-modal],[data-modal-cancel],[data-drawer-cancel],[data-port-add],[data-port-rm],[data-bk-add],[data-bk-rm],[data-step-next],[data-step-prev],[data-adv],[data-go],[data-twirl],[data-switch],[data-mode-seg],[data-canary-slider]';

  document.addEventListener('click', (e) => {
    // ---- notification center ----
    if (e.target.closest('[data-notif-btn]')) { e.stopPropagation(); toggleNotif(); return; }
    if (e.target.closest('[data-notif-clear]')) { clearNotif(); e.stopPropagation(); return; }
    if (notifMenu && notifMenu.classList.contains('open') && !e.target.closest('.notif-wrap')) closeNotif();
    if (e.target.closest('[data-docs-btn]')) { toast('文档中心', { type: 'info', desc: '产品文档将在新标签页打开' }); return; }

    // ---- modal internal close ----
    if (e.target.closest('[data-ix-close]')) { closeModal(); return; }

    // ---- dashboard refresh ----
    const refresh = e.target.closest('#dashRefresh');
    if (refresh) {
      const svg = refresh.querySelector('svg'); if (svg) { svg.classList.add('ax-spin'); setTimeout(() => svg.classList.remove('ax-spin'), 800); }
      toast('已刷新', { type: 'success', desc: '概览与时序指标已更新' });
      return;
    }

    // ---- filters: reset ----
    const clr = e.target.closest('.field-clear');
    if (clr) { const page = clr.closest('.page'); resetFilters(page); return; }

    // ---- pager ----
    const pg = e.target.closest('.pager button:not(:disabled)');
    if (pg && !pg.classList.contains('active')) {
      const wrap = pg.closest('.pager'); if (wrap) { $$('button', wrap).forEach((b) => { if (!b.querySelector('svg')) b.classList.remove('active'); }); if (!pg.querySelector('svg')) pg.classList.add('active'); }
      return;
    }

    // skip elements app.js owns
    if (e.target.closest(OWNED)) return;

    curClickEl = e.target;
    const page = e.target.closest('.page');

    // ---- row action links (lists & detail tables) ----
    const link = e.target.closest('.row-action-link');
    if (link && !link.getAttribute('href') && page) {
      const action = link.textContent.trim();
      handleRowAction(action, link, page);
      return;
    }

    // ---- create buttons ----
    if (e.target.closest('#btn-new-pool')) { doNewPool(); return; }
    const pa = e.target.closest('.page-actions .btn-primary');
    if (pa && page && page.getAttribute('data-page') === 'tenants') { doNewTenant(); return; }

    // ---- in-page primary buttons (units / quota / member create) ----
    const ipb = e.target.closest('.btn-primary');
    if (ipb && page && !ipb.closest('.modal') && !ipb.hasAttribute('data-drawer')) {
      const txt = ipb.textContent.trim();
      if (txt === '新建资源单元') { doUnit(null); return; }
      if (txt === '新增配额') { doQuota(null); return; }
      if (txt === '添加成员') { doAddMember(); return; }
    }

    // ---- detail action buttons + h3 edit ----
    const btn = e.target.closest('.detail-actions .btn, .h3-actions .btn, [data-art-versions] .btn');
    if (btn && page) { handleDetailButton(btn.textContent.trim(), btn, page); return; }
  });

  function handleRowAction(action, link, page) {
    const pid = page.getAttribute('data-page');
    const name = rowName(link);
    // artifact version rows
    if (pid === 'artifact-detail') {
      if (action === '下载') { const tr = link.closest('tr'); doDownloadVersion(tr ? (tr.querySelector('.text-mono') || {}).textContent : ''); return; }
      if (action === '删除') { const tr = link.closest('tr'); doDeleteVersion(tr ? (tr.querySelector('.text-mono') || {}).textContent.trim() : '', link); return; }
      return;
    }
    // tenant detail tabs (quota / members)
    if (pid === 'tenant-detail') {
      const inQuota = !!link.closest('[data-tab-panel="quota"]');
      const inMembers = !!link.closest('[data-tab-panel="members"]');
      if (inQuota) { const q = (link.closest('tr').querySelector('.text-mono') || {}).textContent.trim(); if (action === '编辑') doQuota(q, link); else if (action === '删除') doDeleteQuota(q, link); return; }
      if (inMembers) { const u = (link.closest('tr').querySelector('.text-mono') || {}).textContent.trim(); if (action === '改角色') doChangeRole(u, link); else if (action === '移除') doRemoveMember(u, link); return; }
    }
    // pool detail units tab
    if (pid === 'pool-detail') {
      const u = (link.closest('tr').querySelector('.row-link, .text-mono') || {}).textContent.trim();
      if (action === '编辑') doUnit(u, link); else if (action === '删除') doDeleteUnit(u, link);
      return;
    }
    // jobs list — 运行 / 编辑(模板) / 删除 (详情 is an href link, handled by router)
    if (pid === 'jobs') {
      if (action === '运行') { AX().openRunDialog(name); return; }
      if (action === '编辑') { doEditTemplate(name); return; }
      // 删除 falls through to generic switch (doDelete)
    }
    // job detail — Run history rows: 取消 / 删除 (日志 / 详情 are href links)
    if (pid === 'job-detail') {
      const runName = (link.closest('tr').querySelector('.row-link, .text-mono') || {}).textContent.trim();
      const jobName = detailName(page) || curJobName();
      if (action === '取消') { doCancelRun(jobName, runName); return; }
      if (action === '删除') { doDeleteRun(jobName, runName); return; }
      return;
    }
    // tenant list suspend
    if (pid === 'tenants' && (action === '暂停' || action === '恢复')) { doSuspend(name, link); return; }
    // pool list delete
    if (pid === 'pools' && action === '删除') { doDelete('pools', name); return; }

    // generic compute / artifact list actions
    switch (action) {
      case '打开': doOpen(name, link.getAttribute('data-tool')); break;
      case '停止': doStop(pid, name); break;
      case '启动': doStart(pid, name); break;
      case '删除': doDelete(pid, name); break;
      case '取消': doCancel(name); break;
      case '再次提交': doResubmit(name); break;
      case '日志': location.hash = '#/' + pid + '/' + name; break;
      case '扩缩': doScale('services', name); break;
      case '调整流量': location.hash = '#/traffic/' + name; break;
      case '提升': location.hash = '#/traffic/' + name; break;
      case '回滚': location.hash = '#/traffic/' + name; break;
      case '启用': doEnableTraffic(name); break;
      case '禁用': doDisableTraffic(name); break;
      case '上传新版本': doUploadVersion(pid); break;
      default: break;
    }
  }

  function handleDetailButton(action, btn, page) {
    const pid = page.getAttribute('data-page');
    const name = detailName(page);
    if (action === '编辑') { if (pid === 'job-detail') { doEditTemplate(name); return; } doEditMeta(pid, name); return; }
    if (action === '编辑模板') { doEditTemplate(name); return; }
    if (action === '运行') { return; } // handled in app.js via [data-run-job]
    if (action === '打开') { doOpen(name); return; }
    if (action === '停止') { doStop(pid, name); return; }
    if (action === '扩缩容') { doScale('services', name); return; }
    if (action === '取消任务') { doCancel(name); return; }
    if (action === '取消运行') { doCancelRun(curJobName(), name); return; }
    if (action === '再次提交') { doResubmit(name); return; }
    if (action === '暂停' || action === '恢复') { doSuspend(name, null); return; }
    if (action === '删除' || action === '删除资源池') {
      if (pid === 'job-run-detail') { doDeleteRun(curJobName(), name); return; }
      doDelete(pid, name, true); return;
    }
  }

  /* =========================================================
     Filters (live search + selects + reset)
     ========================================================= */
  function getFilterState(filters) {
    const q = (() => { const i = filters.querySelector('.field-search input, input'); return i ? i.value.trim().toLowerCase() : ''; })();
    const sels = $$('.field-select select', filters).map((s) => s.value).filter((v) => v && v !== '全部');
    return { q, sels };
  }
  function applyFilter(page) {
    if (!page) return;
    const filters = $('.filters', page); if (!filters) return;
    const { q, sels } = getFilterState(filters);
    const match = (txt) => { txt = txt.toLowerCase(); if (q && !txt.includes(q)) return false; for (const v of sels) { if (!txt.includes(String(v).toLowerCase())) return false; } return true; };
    let shown = 0, total = 0;
    const cards = $$('.res-card', page);
    if (cards.length) {
      cards.forEach((c) => { total++; const ok = match(c.textContent); c.style.display = ok ? '' : 'none'; if (ok) shown++; });
    } else {
      const table = $('table.table', page);
      if (table) $$('tbody tr', table).forEach((tr) => {
        if (tr.classList.contains('unit-expand')) { tr.style.display = 'none'; return; }
        total++; const ok = match(tr.textContent); tr.style.display = ok ? '' : 'none'; if (ok) shown++;
      });
    }
    const foot = $('.table-foot span', page);
    if (foot) { if (q || sels.length) { if (!foot.dataset.orig) foot.dataset.orig = foot.textContent; foot.textContent = `筛选出 ${shown} 个 / 共 ${total} 个`; } else if (foot.dataset.orig) { foot.textContent = foot.dataset.orig; } }
  }
  function resetFilters(page) {
    if (!page) return;
    const filters = $('.filters', page); if (!filters) return;
    $$('input', filters).forEach((i) => { i.value = ''; });
    $$('select', filters).forEach((s) => { s.selectedIndex = 0; });
    applyFilter(page);
    toast('已重置筛选', { type: 'info', duration: 1800 });
  }
  document.addEventListener('input', (e) => { if (e.target.closest('.filters')) { const p = e.target.closest('.page'); if (p) applyFilter(p); } });
  document.addEventListener('change', (e) => { if (e.target.closest('.filters .field-select')) { const p = e.target.closest('.page'); if (p) applyFilter(p); } });

  /* =========================================================
     Notification center + topbar wiring
     ========================================================= */
  let notifMenu, notifBell;
  const NOTIFS = [
    { m: '任务 <code>train-llm-7b</code> 已运行 2h14m，进度 34%', tm: '3 分钟前', read: false },
    { m: '服务 <code>svc-embed</code> 进入降级：1/2 副本就绪', tm: '22 分钟前', read: false },
    { m: '配额告警：<code>gpu-a100/training</code> 用量达 93%', tm: '1 小时前', read: false },
    { m: '灰度策略 <code>rt-chat</code> 已放量至 10%', tm: '2 小时前', read: true },
  ];
  function buildNotif() {
    notifBell = $$('.topbar-right .icon-btn').find((b) => b.title === '通知');
    const docs = $$('.topbar-right .icon-btn').find((b) => b.title === '文档');
    if (docs) docs.setAttribute('data-docs-btn', '');
    if (!notifBell) return;
    notifBell.setAttribute('data-notif-btn', '');
    notifBell.classList.add('has-dot');
    const wrap = document.createElement('div'); wrap.className = 'notif-wrap';
    notifBell.parentNode.insertBefore(wrap, notifBell); wrap.appendChild(notifBell);
    notifMenu = document.createElement('div'); notifMenu.className = 'notif-menu';
    wrap.appendChild(notifMenu);
    renderNotif();
  }
  function renderNotif() {
    if (!notifMenu) return;
    const unread = NOTIFS.filter((n) => !n.read).length;
    notifMenu.innerHTML =
      `<div class="notif-head"><span class="t">通知${unread ? ` · ${unread} 条未读` : ''}</span>${unread ? '<span class="clr" data-notif-clear>全部已读</span>' : ''}</div>` +
      (NOTIFS.length ? NOTIFS.map((n) => `<div class="notif-item ${n.read ? 'read' : ''}"><span class="notif-dot"></span><div class="notif-tx"><div class="m">${n.m}</div><div class="tm">${n.tm}</div></div></div>`).join('')
        : '<div class="notif-empty">暂无通知</div>');
    if (notifBell) notifBell.classList.toggle('has-dot', unread > 0);
  }
  function toggleNotif() { if (!notifMenu) return; notifMenu.classList.contains('open') ? closeNotif() : notifMenu.classList.add('open'); }
  function closeNotif() { if (notifMenu) notifMenu.classList.remove('open'); }
  function clearNotif() { NOTIFS.forEach((n) => (n.read = true)); renderNotif(); toast('已全部标为已读', { type: 'success', duration: 1800 }); }

  /* ---------- boot ---------- */
  function boot() { buildNotif(); }
  if (window.AX) boot(); else document.addEventListener('ax:ready', boot, { once: true });
})();


/* ===================================================================
   ▒▒  ENHANCEMENT SUB-LAYER  ▒▒  (was enhance.js — merged per layout)
   =================================================================== */

/* =====================================================================
   AxisML console — enhancement sub-layer (merged in)
   Part of the interaction layer. Sits on top of app.js via window.AX and
   pure DOM post-processing:
     • responsive / collapsible navigation (rail + off-canvas)
     • sortable table headers
     • multi-select rows + floating bulk-action bar
     • active filter chips (mirrors the existing live filter)
     • live status (pulsing pills, ticking elapsed, running progress)
   Kept as its own IIFE below so scopes never collide with the code above.
   ===================================================================== */
(function () {
  'use strict';

  const $  = (s, r = document) => r.querySelector(s);
  const $$ = (s, r = document) => Array.from(r.querySelectorAll(s));
  const AX = () => window.AX || {};
  const LIVE_LABELS = ['运行中', '启动中', '灰度中', '扩缩中', '排队中'];
  const PULSE_LABELS = ['运行中', '启动中', '灰度中'];

  /* ---------- tiny icon set ---------- */
  const I = {
    panel: '<svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="4" width="18" height="16" rx="2"/><path d="M9 4v16"/></svg>',
    x:     '<svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"><path d="M4 4l8 8M12 4l-8 8"/></svg>',
    trash: '<svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M2.5 4h11M6 4V2.5h4V4M5 4l.5 9h5l.5-9"/></svg>',
    stop:  '<svg width="13" height="13" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="4" y="4" width="8" height="8" rx="1.5"/></svg>',
    ok:    '<svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><circle cx="8" cy="8" r="6.5"/><path d="M5.2 8.2 7 10l3.8-4"/></svg>',
    info:  '<svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="8" cy="8" r="6.5"/><path d="M8 7v4M8 5h.01"/></svg>',
    warn:  '<svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M8 2.2 14.5 13.5H1.5z"/><path d="M8 6.6v3M8 11.4h.01"/></svg>',
  };

  /* =========================================================
     Toast (reuses existing .toast styles; coexists w/ interactions.js)
     ========================================================= */
  let tWrap;
  function toast(title, opts = {}) {
    if (!tWrap) { tWrap = $('.toast-wrap') || Object.assign(document.createElement('div'), { className: 'toast-wrap' }); if (!tWrap.parentNode) document.body.appendChild(tWrap); }
    const type = opts.type || 'success';
    const el = document.createElement('div'); el.className = 'toast';
    el.innerHTML = `<span class="toast-ico ${type}">${I[type] || I.ok}</span>` +
      `<div class="toast-body"><div class="toast-title">${title}</div>${opts.desc ? `<div class="toast-desc">${opts.desc}</div>` : ''}</div>` +
      `<button class="toast-x">${I.x}</button>`;
    tWrap.appendChild(el);
    requestAnimationFrame(() => el.classList.add('in'));
    const close = () => { if (el.dataset.c) return; el.dataset.c = 1; el.classList.remove('in'); el.classList.add('out'); setTimeout(() => el.remove(), 260); };
    el.querySelector('.toast-x').addEventListener('click', close);
    setTimeout(close, opts.duration || 3200);
  }

  /* =========================================================
     Lightweight confirm (reuses .scrim/.modal styles)
     ========================================================= */
  function confirmBox(opts) {
    const scrim = Object.assign(document.createElement('div'), { className: 'scrim' });
    const modal = Object.assign(document.createElement('div'), { className: 'modal' });
    modal.style.width = '440px';
    modal.innerHTML =
      `<div class="modal-head"><h2 class="modal-title">${opts.title || '确认操作'}</h2></div>` +
      `<div class="modal-body"><p class="cfm-msg">${opts.message || ''}</p></div>` +
      `<div class="modal-foot"><button class="btn btn-sm" data-x>${opts.cancelText || '取消'}</button>` +
      `<button class="btn btn-sm ${opts.danger ? 'btn-danger' : 'btn-primary'}" data-ok>${opts.confirmText || '确认'}</button></div>`;
    document.body.append(scrim, modal);
    requestAnimationFrame(() => { scrim.classList.add('open'); modal.classList.add('open'); });
    if (opts.danger) modal.querySelector('.modal-title').style.color = 'var(--app-danger-deep)';
    const close = () => { scrim.classList.remove('open'); modal.classList.remove('open'); setTimeout(() => { scrim.remove(); modal.remove(); }, 200); };
    scrim.addEventListener('click', close);
    modal.querySelector('[data-x]').addEventListener('click', close);
    modal.querySelector('[data-ok]').addEventListener('click', () => { close(); opts.onConfirm && opts.onConfirm(); });
  }

  /* =========================================================
     Helpers
     ========================================================= */
  function rowName(tr) {
    const a = tr.querySelector('.row-link, .res-card-name'); if (a) return a.textContent.trim();
    const m = tr.querySelector('.text-mono'); return m ? m.textContent.trim() : '';
  }
  function isListKind(k) { const D = AX().DATA; return !!(D && D[k]); }

  /* =========================================================
     1 · Responsive / collapsible navigation
     ========================================================= */
  const isMobile = () => window.matchMedia('(max-width: 1024px)').matches;

  function setupNav() {
    if ($('.nav-toggle')) return;
    // wrap bare text labels so rail mode can hide them
    $$('.sidebar .nav-item').forEach((it) => {
      const label = it.textContent.trim();
      Array.from(it.childNodes).forEach((n) => {
        if (n.nodeType === 3) {
          if (n.textContent.trim()) { const s = document.createElement('span'); s.className = 'nav-label'; s.textContent = n.textContent.trim(); it.replaceChild(s, n); }
          else it.removeChild(n);
        }
      });
      it.setAttribute('data-tip', label);
    });
    // toggle button into topbar (before brand)
    const topbar = $('.topbar'); const brand = $('.brand');
    const btn = document.createElement('button');
    btn.className = 'nav-toggle'; btn.title = '折叠 / 展开导航'; btn.setAttribute('aria-label', '切换导航'); btn.innerHTML = I.panel;
    topbar.insertBefore(btn, brand);
    // off-canvas scrim
    const scrim = Object.assign(document.createElement('div'), { className: 'nav-scrim' });
    document.body.appendChild(scrim);

    btn.addEventListener('click', () => {
      if (isMobile()) document.body.classList.toggle('nav-open');
      else { document.body.classList.toggle('nav-rail'); try { localStorage.setItem('axisml.navRail', document.body.classList.contains('nav-rail') ? '1' : '0'); } catch (e) {} }
    });
    scrim.addEventListener('click', () => document.body.classList.remove('nav-open'));
    $('.sidebar').addEventListener('click', (e) => { if (e.target.closest('.nav-item') && isMobile()) document.body.classList.remove('nav-open'); });
    window.addEventListener('resize', () => { if (!isMobile()) document.body.classList.remove('nav-open'); });

    // restore desktop rail preference
    try { if (localStorage.getItem('axisml.navRail') === '1' && !isMobile()) document.body.classList.add('nav-rail'); } catch (e) {}
  }

  /* =========================================================
     2 · Sortable headers
     ========================================================= */
  function attachSort(table) {
    const heads = $$('thead th', table);
    heads.forEach((th) => {
      if (th.classList.contains('actions') || th.classList.contains('col-sel')) return;
      if (th.querySelector('.th-sort')) return;
      th.classList.add('sortable');
      const ind = document.createElement('span'); ind.className = 'th-sort'; ind.innerHTML = '<i class="up"></i><i class="dn"></i>';
      th.appendChild(ind);
      th.addEventListener('click', () => sortBy(table, th));
    });
  }
  function sortBy(table, th) {
    const tbody = $('tbody', table);
    const idx = th.cellIndex;
    const asc = !(th.classList.contains('sorted-asc'));
    $$('thead th', table).forEach((h) => h.classList.remove('sorted-asc', 'sorted-desc'));
    th.classList.add(asc ? 'sorted-asc' : 'sorted-desc');
    const rows = $$('tbody > tr', tbody).filter((r) => !r.classList.contains('unit-expand') && !r.classList.contains('table-empty'));
    const val = (r) => { const c = r.cells[idx]; return c ? c.textContent.trim() : ''; };
    const numeric = rows.every((r) => { const v = val(r).replace(/[,\s]/g, ''); return v === '' || v === '—' || /^[-+]?\d*\.?\d+%?$/.test(v) || /^\d+\/\d+$/.test(v); });
    rows.sort((a, b) => {
      let x = val(a), y = val(b);
      if (numeric) {
        const num = (s) => { s = s.replace(/[,%\s]/g, ''); if (s === '' || s === '—') return -Infinity; if (/^\d+\/\d+$/.test(s)) return parseFloat(s.split('/')[0]); return parseFloat(s) || 0; };
        return (num(x) - num(y)) * (asc ? 1 : -1);
      }
      return x.localeCompare(y, 'zh-Hans-CN') * (asc ? 1 : -1);
    });
    rows.forEach((r) => tbody.appendChild(r));
  }

  /* =========================================================
     3 · Multi-select + bulk-action bar
     ========================================================= */
  let bulkBar, selected = new Set(), selKind = null;
  const BULK = {
    workspaces: [{ t: '停止', a: 'stop', ico: I.stop }, { t: '删除', a: 'del', danger: 1, ico: I.trash }],
    jobs:       [{ t: '删除', a: 'del', danger: 1, ico: I.trash }],
    services:   [{ t: '停止', a: 'stop', ico: I.stop }, { t: '删除', a: 'del', danger: 1, ico: I.trash }],
    traffic:    [{ t: '删除', a: 'del', danger: 1, ico: I.trash }],
    datasets:   [{ t: '删除', a: 'del', danger: 1, ico: I.trash }],
    models:     [{ t: '删除', a: 'del', danger: 1, ico: I.trash }],
    images:     [{ t: '删除', a: 'del', danger: 1, ico: I.trash }],
  };

  function ensureBulkBar() {
    if (bulkBar) return;
    bulkBar = document.createElement('div'); bulkBar.className = 'bulkbar';
    document.body.appendChild(bulkBar);
  }
  function renderBulkBar() {
    ensureBulkBar();
    const n = selected.size;
    if (!n || !selKind) { bulkBar.classList.remove('show'); return; }
    const noun = { workspaces: '工作区', jobs: '任务', services: '服务', traffic: '策略', datasets: '数据集', models: '模型', images: '镜像' }[selKind] || '项';
    const btns = (BULK[selKind] || [{ t: '删除', a: 'del', danger: 1, ico: I.trash }])
      .map((b) => `<button class="bb-btn ${b.danger ? 'danger' : ''}" data-bulk="${b.a}">${b.ico}<span class="lbl">${b.t}</span></button>`).join('');
    bulkBar.innerHTML =
      `<span class="bb-count"><span class="bb-n">${n}</span> 已选 ${noun}</span>` +
      `<span class="bb-sep"></span>${btns}` +
      `<button class="bb-x" data-bulk="clear" title="取消选择">${I.x}</button>`;
    bulkBar.classList.add('show');
  }
  function clearSelection() {
    selected.clear();
    $$('.page.active tr.row-selected, .page.active .res-card.row-selected').forEach((r) => r.classList.remove('row-selected'));
    $$('.page.active .ax-check').forEach((c) => { c.checked = false; c.classList.remove('indeterminate'); });
    renderBulkBar();
  }
  function syncSelectAll(table) {
    const all = $('.ax-check-all', table); if (!all) return;
    const boxes = visibleRowBoxes(table);
    const checked = boxes.filter((b) => b.checked).length;
    all.classList.toggle('indeterminate', checked > 0 && checked < boxes.length);
    all.checked = boxes.length > 0 && checked === boxes.length;
  }
  function visibleRowBoxes(table) {
    return $$('tbody > tr', table).filter((r) => r.style.display !== 'none' && !r.classList.contains('unit-expand') && !r.classList.contains('table-empty'))
      .map((r) => r.querySelector('.ax-check')).filter(Boolean);
  }

  function attachSelection(table, kind) {
    // header checkbox
    const headRow = $('thead tr', table);
    if (headRow && !$('.col-sel', headRow)) {
      const th = document.createElement('th'); th.className = 'col-sel';
      th.innerHTML = `<input type="checkbox" class="ax-check ax-check-all" title="全选本页">`;
      headRow.insertBefore(th, headRow.firstChild);
    }
    // row checkboxes
    $$('tbody > tr', table).forEach((tr) => {
      if (tr.classList.contains('unit-expand') || tr.classList.contains('table-empty')) return;
      if ($('.col-sel', tr)) return;
      const name = rowName(tr);
      const td = document.createElement('td'); td.className = 'col-sel';
      td.innerHTML = `<input type="checkbox" class="ax-check" data-name="${name}">`;
      tr.insertBefore(td, tr.firstChild);
    });

    table.addEventListener('change', (e) => {
      const box = e.target.closest('.ax-check'); if (!box) return;
      if (box.classList.contains('ax-check-all')) {
        const on = box.checked; box.classList.remove('indeterminate');
        visibleRowBoxes(table).forEach((b) => { b.checked = on; toggleRow(b, on); });
      } else {
        toggleRow(box, box.checked);
        syncSelectAll(table);
      }
      renderBulkBar();
    });
  }
  function toggleRow(box, on) {
    const tr = box.closest('tr'); const name = box.getAttribute('data-name');
    if (tr) tr.classList.toggle('row-selected', on);
    if (!name) return;
    if (on) selected.add(name); else selected.delete(name);
  }

  function runBulk(action) {
    const kind = selKind; const names = new Set(selected);
    if (!kind || !names.size) return;
    const D = AX().DATA; if (!D || !D[kind]) return;
    const noun = { workspaces: '工作区', jobs: '任务', services: '服务', traffic: '策略', datasets: '数据集', models: '模型', images: '镜像' }[kind] || '项';
    if (action === 'del') {
      confirmBox({
        title: `删除 ${names.size} 个${noun}`, danger: true, confirmText: '删除',
        message: `将永久删除选中的 ${names.size} 个${noun}，操作不可恢复。`,
        onConfirm() {
          D[kind] = D[kind].filter((x) => !names.has(x.name));
          AX().renderList(kind); clearSelection();
          toast(`已删除 ${names.size} 个${noun}`, { type: 'info' });
        },
      });
    } else if (action === 'stop') {
      D[kind].forEach((it) => { if (names.has(it.name)) { it.status = { tone: 'muted', label: '已停止' }; it.rep = '0/0'; } });
      AX().renderList(kind); clearSelection();
      toast(`已停止 ${names.size} 个${noun}`, { type: 'info', desc: '副本已缩至 0' });
    } else if (action === 'cancel') {
      D[kind].forEach((it) => { if (names.has(it.name)) { it.status = { tone: 'muted', label: '已取消' }; } });
      AX().renderList(kind); clearSelection();
      toast(`已取消 ${names.size} 个${noun}`, { type: 'info' });
    }
  }
  document.addEventListener('click', (e) => {
    const b = e.target.closest('[data-bulk]'); if (!b) return;
    const a = b.getAttribute('data-bulk');
    if (a === 'clear') clearSelection(); else runBulk(a);
  });

  /* card-view selection */
  function attachCardSelection(host, kind) {
    $$('.res-card', host).forEach((card) => {
      if ($('.ax-card-check', card)) return;
      const name = (card.querySelector('.res-card-name') || {}).textContent ? card.querySelector('.res-card-name').textContent.trim() : '';
      const box = document.createElement('input'); box.type = 'checkbox'; box.className = 'ax-check ax-card-check'; box.setAttribute('data-name', name);
      card.appendChild(box);
      box.addEventListener('change', () => { toggleRow(box, box.checked); renderBulkBar(); });
    });
  }

  /* =========================================================
     4 · Active filter chips
     ========================================================= */
  function ensureChips(page) {
    const filters = $('.filters', page); if (!filters) return null;
    let chips = filters.nextElementSibling && filters.nextElementSibling.classList.contains('filter-chips')
      ? filters.nextElementSibling : null;
    if (!chips) { chips = document.createElement('div'); chips.className = 'filter-chips'; filters.parentNode.insertBefore(chips, filters.nextSibling); }
    return chips;
  }
  function renderChips(page) {
    const chips = ensureChips(page); if (!chips) return;
    const filters = $('.filters', page);
    const parts = [];
    const search = $('.field-search input, .field input', filters);
    if (search && search.value.trim()) parts.push({ k: '搜索', v: search.value.trim(), el: search, type: 'text' });
    $$('.field-select', filters).forEach((fs) => {
      const sel = fs.querySelector('select'); if (!sel) return;
      const v = sel.value; const label = (fs.querySelector('.text-muted') || {}).textContent || '';
      if (v && v !== '全部' && sel.selectedIndex !== 0) parts.push({ k: label.trim(), v, el: sel, type: 'select' });
    });
    if (!parts.length) { chips.innerHTML = ''; return; }
    chips.innerHTML = parts.map((p, i) =>
      `<span class="fchip" data-ci="${i}"><span class="fk">${p.k}</span><span class="fv">${p.v}</span><span class="fx" data-ci="${i}">${I.x}</span></span>`
    ).join('') + `<span class="fchip-clear" data-chip-clear>清除全部</span>`;
    chips.__parts = parts;
  }
  document.addEventListener('click', (e) => {
    const fx = e.target.closest('.fchip .fx');
    if (fx) {
      const page = e.target.closest('.page'); const chips = e.target.closest('.filter-chips');
      const p = (chips.__parts || [])[+fx.getAttribute('data-ci')]; if (!p) return;
      if (p.type === 'text') { p.el.value = ''; p.el.dispatchEvent(new Event('input', { bubbles: true })); }
      else { p.el.selectedIndex = 0; p.el.dispatchEvent(new Event('change', { bubbles: true })); }
      renderChips(page);
      return;
    }
    if (e.target.closest('[data-chip-clear]')) {
      const page = e.target.closest('.page'); const clr = $('.field-clear', page);
      if (clr) clr.click(); else resetPageFilters(page);
      setTimeout(() => renderChips(page), 0);
    }
  });
  function resetPageFilters(page) {
    const filters = $('.filters', page); if (!filters) return;
    $$('input', filters).forEach((i) => { i.value = ''; i.dispatchEvent(new Event('input', { bubbles: true })); });
    $$('select', filters).forEach((s) => { s.selectedIndex = 0; s.dispatchEvent(new Event('change', { bubbles: true })); });
  }
  document.addEventListener('input', (e) => { if (e.target.closest('.filters')) { const p = e.target.closest('.page'); if (p) { renderChips(p); updateEmptyState(p); } } });
  document.addEventListener('change', (e) => { if (e.target.closest('.filters')) { const p = e.target.closest('.page'); if (p) { renderChips(p); updateEmptyState(p); } } });

  /* filtered-to-zero empty state */
  function updateEmptyState(page) {
    const table = $('[data-list] table.table', page); if (!table) return;
    const tbody = $('tbody', table);
    const rows = $$('tbody > tr', tbody).filter((r) => !r.classList.contains('table-empty') && !r.classList.contains('unit-expand'));
    const visible = rows.filter((r) => r.style.display !== 'none');
    let empty = $('.table-empty', tbody);
    if (visible.length === 0 && rows.length > 0) {
      if (!empty) {
        empty = document.createElement('tr'); empty.className = 'table-empty';
        const cols = $$('thead th', table).length || 6;
        empty.innerHTML = `<td colspan="${cols}"><div class="te-title">没有匹配的结果</div><div>试着调整搜索关键词或清除筛选条件。</div></td>`;
        tbody.appendChild(empty);
      } else empty.style.display = '';
    } else if (empty) empty.style.display = 'none';
  }

  /* =========================================================
     5 · Live status — pulsing pills, ticking elapsed, progress
     ========================================================= */
  function applyLive(scope) {
    $$('.pill', scope).forEach((p) => {
      const label = p.textContent.trim();
      p.classList.toggle('pill-live', PULSE_LABELS.includes(label));
    });
    // running jobs: ticking elapsed + progress sliver on the name row
    const jobsPage = scope.matches && scope.matches('[data-page="jobs"]') ? scope : $('[data-page="jobs"]', scope) || (scope.querySelector ? null : null);
    const page = $('.page.active');
    if (page && page.getAttribute('data-page') === 'jobs') {
      $$('[data-list="jobs"] tbody > tr', page).forEach((tr) => {
        const label = (tr.querySelector('.pill') || {}).textContent ? tr.querySelector('.pill').textContent.trim() : '';
        if (label === '运行中') {
          // elapsed clock = last mono cell that looks like HH:MM:SS
          const durCell = $$('td.text-mono', tr).find((c) => /^\d{2}:\d{2}:\d{2}$/.test(c.textContent.trim()));
          if (durCell && !durCell.hasAttribute('data-live-clock')) { durCell.setAttribute('data-live-clock', '1'); durCell.classList.add('live-dur'); }
          // progress sliver (stable pseudo-progress per row, advances over time)
          const nameCell = tr.cells[ tr.querySelector('.col-sel') ? 1 : 0 ];
          if (nameCell && !nameCell.classList.contains('row-prog')) {
            nameCell.classList.add('row-prog');
            const seed = (rowName(tr).length % 5) / 10 + 0.25; // 0.25–0.65 start
            nameCell.style.setProperty('--p', Math.min(0.95, seed).toFixed(2));
          }
        }
      });
    }
  }
  // one global ticker for any live elapsed clock
  setInterval(() => {
    $$('[data-live-clock]').forEach((c) => {
      const m = c.textContent.trim().match(/^(\d{2}):(\d{2}):(\d{2})$/); if (!m) return;
      let s = (+m[1]) * 3600 + (+m[2]) * 60 + (+m[3]) + 1;
      const hh = String(Math.floor(s / 3600)).padStart(2, '0');
      const mm = String(Math.floor((s % 3600) / 60)).padStart(2, '0');
      const ss = String(s % 60).padStart(2, '0');
      c.textContent = `${hh}:${mm}:${ss}`;
    });
    // nudge running progress slivers upward, slowly
    $$('.page.active .row-prog').forEach((n) => {
      const p = parseFloat(n.style.getPropertyValue('--p') || '0');
      if (p < 0.95) n.style.setProperty('--p', Math.min(0.95, p + 0.004).toFixed(3));
    });
  }, 1000);

  /* =========================================================
     Orchestration — enhance after each render / route
     ========================================================= */
  function enhanceList(kind) {
    const host = $(`[data-list="${kind}"]`); if (!host) return;
    const page = host.closest('.page');
    if (selKind !== kind) { selected.clear(); selKind = kind; }
    else selected.clear();           // list re-rendered → drop stale selection
    renderBulkBar();
    const table = $('table.table', host);
    if (table) { attachSort(table); attachSelection(table, kind); syncSelectAll(table); }
    else { attachCardSelection(host, kind); }
    if (page) { renderChips(page); updateEmptyState(page); applyLive(page); }
  }

  function enhanceActive() {
    const page = $('.page.active'); if (!page) return;
    applyLive(page);
    const host = $('[data-list]', page);
    if (host) { const k = host.getAttribute('data-list'); if (isListKind(k)) enhanceList(k); }
    else { selKind = null; selected.clear(); renderBulkBar(); }
  }

  /* =========================================================
     Boot — wrap the bridge, hook routing
     ========================================================= */
  function boot() {
    setupNav();
    // wrap renderList so every (re)render re-enhances
    const orig = AX().renderList;
    if (orig && !orig.__wrapped) {
      const wrapped = function (kind) { const r = orig.apply(this, arguments); try { enhanceList(kind); } catch (e) { console.warn('enhanceList', e); } return r; };
      wrapped.__wrapped = true;
      window.AX.renderList = wrapped;
    }
    // route changes: app.js handles its hashchange first; run after
    window.addEventListener('hashchange', () => setTimeout(enhanceActive, 0));
    // card/list view toggle re-renders via app.js internals → re-enhance
    document.addEventListener('click', (e) => { if (e.target.closest('.view-toggle button')) setTimeout(enhanceActive, 0); });
    // first paint
    setTimeout(enhanceActive, 0);
  }
  if (window.AX) boot(); else document.addEventListener('ax:ready', boot, { once: true });
})();
