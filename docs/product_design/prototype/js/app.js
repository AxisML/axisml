/* ============================================================================
   AxisML 控制台 · 共享外壳与交互
   - 单一导航配置（NAV）
   - 自动注入侧边栏 + 顶栏（页面只写 <main class="page" id="page"> 内容）
   - 角色 / 租户切换（localStorage 持久化，驱动菜单可见性）
   - 抽屉 / 弹窗 / 标签页 / 下拉 / Toast 等通用交互
   - 图表小工具（折线 / sparkline）
   ========================================================================== */
(function () {
  "use strict";

  /* ── 图标库（monoline，stroke=currentColor）──────────────────────────── */
  const I = {
    dashboard: '<path d="M3 13h8V3H3zM13 21h8V11h-8zM13 3v6h8V3zM3 21h8v-6H3z"/>',
    workspace: '<rect x="3" y="4" width="18" height="14" rx="2"/><path d="M8 21h8M12 18v3M7 9l2 2-2 2M12 13h3"/>',
    job: '<path d="M5 4h14a1 1 0 0 1 1 1v4H4V5a1 1 0 0 1 1-1zM4 9v10a1 1 0 0 0 1 1h14a1 1 0 0 0 1-1V9"/><path d="M9 13h6"/>',
    experiment: '<path d="M9 3h6M10 3v6L5 19a2 2 0 0 0 1.8 3h10.4A2 2 0 0 0 19 19l-5-10V3"/><path d="M7.5 15h9"/>',
    eval: '<path d="M4 19V5M4 19h16M8 16v-5M12 16V8M16 16v-3M20 16V6"/>',
    service: '<rect x="3" y="4" width="18" height="6" rx="1.5"/><rect x="3" y="14" width="18" height="6" rx="1.5"/><path d="M7 7h.01M7 17h.01"/>',
    traffic: '<circle cx="6" cy="12" r="2.2"/><circle cx="18" cy="6" r="2.2"/><circle cx="18" cy="18" r="2.2"/><path d="M8 11l8-4M8 13l8 4"/>',
    dataset: '<ellipse cx="12" cy="6" rx="8" ry="3"/><path d="M4 6v6c0 1.7 3.6 3 8 3s8-1.3 8-3V6M4 12v6c0 1.7 3.6 3 8 3s8-1.3 8-3v-6"/>',
    model: '<path d="M12 2l8 4.5v9L12 20l-8-4.5v-9z"/><path d="M12 11l8-4.5M12 11v9M12 11L4 6.5"/>',
    image: '<rect x="4" y="4" width="16" height="16" rx="2.5"/><path d="M4 9.33h16M4 14.66h16"/>',
    tenant: '<path d="M3 21V8l6-4 6 4v13M15 21V11l6 4v6M3 21h18M7 12h.01M7 16h.01"/>',
    pool: '<rect x="3" y="3" width="7" height="7" rx="1.5"/><rect x="14" y="3" width="7" height="7" rx="1.5"/><rect x="3" y="14" width="7" height="7" rx="1.5"/><rect x="14" y="14" width="7" height="7" rx="1.5"/>',
    search: '<circle cx="11" cy="11" r="7"/><path d="m20 20-3.5-3.5"/>',
    bell: '<path d="M18 8a6 6 0 1 0-12 0c0 7-3 9-3 9h18s-3-2-3-9M13.7 21a2 2 0 0 1-3.4 0"/>',
    help: '<circle cx="12" cy="12" r="9"/><path d="M9.5 9a2.5 2.5 0 1 1 3.5 2.3c-.8.4-1 .9-1 1.7M12 17h.01"/>',
    chevron: '<path d="M6 9l6 6 6-6"/>',
    menu: '<path d="M3 6h18M3 12h18M3 18h18"/>',
    plus: '<path d="M12 5v14M5 12h14"/>',
    check: '<path d="M20 6 9 17l-5-5"/>',
    play: '<path d="M6 4l14 8-14 8z"/>',
    stop: '<rect x="5" y="5" width="14" height="14" rx="2"/>',
    refresh: '<path d="M21 12a9 9 0 1 1-3-6.7L21 7M21 3v4h-4"/>',
    filter: '<path d="M3 5h18l-7 8v5l-4 2v-7z"/>',
    more: '<circle cx="5" cy="12" r="1.6"/><circle cx="12" cy="12" r="1.6"/><circle cx="19" cy="12" r="1.6"/>',
    cpu: '<rect x="6" y="6" width="12" height="12" rx="1.5"/><path d="M9 1v3M15 1v3M9 20v3M15 20v3M1 9h3M1 15h3M20 9h3M20 15h3"/>',
    gpu: '<rect x="2" y="6" width="20" height="12" rx="2"/><circle cx="8" cy="12" r="2.5"/><path d="M14 10h5M14 14h5"/>',
    mem: '<rect x="3" y="7" width="18" height="10" rx="1.5"/><path d="M7 7V5M11 7V5M15 7V5M19 7V5M7 21v-2M11 21v-2M15 21v-2"/>',
    rocket: '<path d="M5 15c-1 1-1.5 4-1.5 4s3-.5 4-1.5M9 11a14 14 0 0 1 7-7c2 0 3 1 3 3a14 14 0 0 1-7 7l-3 .5z"/><circle cx="14.5" cy="9.5" r="1.3"/>',
    clock: '<circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/>',
    user: '<circle cx="12" cy="8" r="4"/><path d="M4 21a8 8 0 0 1 16 0"/>',
    arrowUp: '<path d="M12 19V5M5 12l7-7 7 7"/>',
    arrowDown: '<path d="M12 5v14M19 12l-7 7-7-7"/>',
    ext: '<path d="M14 4h6v6M20 4l-9 9M19 14v5a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1V6a1 1 0 0 1 1-1h5"/>',
    layers: '<path d="M12 3 2 8l10 5 10-5z"/><path d="m2 13 10 5 10-5M2 18l10 5 10-5"/>',
    globe: '<circle cx="12" cy="12" r="9"/><path d="M3 12h18"/><path d="M12 3c2.5 2.4 3.9 5.6 4 9-.1 3.4-1.5 6.6-4 9-2.5-2.4-3.9-5.6-4-9 .1-3.4 1.5-6.6 4-9z"/>',
    theme: '<circle cx="12" cy="12" r="9"/><path d="M12 3v18a9 9 0 0 0 0-18z" fill="currentColor" stroke="none"/>',
    power: '<path d="M12 4v8"/><path d="M7.6 7.6a7 7 0 1 0 8.8 0"/>',
    chevronR: '<path d="M9 6l6 6-6 6"/>',
    sun: '<circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4"/>',
    moon: '<path d="M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8z"/>',
    monitor: '<rect x="3" y="4" width="18" height="13" rx="2"/><path d="M8 21h8M12 17v4"/>',
    logout: '<path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/><path d="M16 17l5-5-5-5"/><path d="M21 12H9"/>',
  };
  function svg(name, cls) {
    return '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round"' + (cls ? ' class="' + cls + '"' : '') + '>' + (I[name] || '') + '</svg>';
  }

  /* ── 导航配置（单一来源）─────────────────────────────────────────────── */
  const NAV = [
    { key: "dashboard", label: "首页", icon: "dashboard", href: "index.html" },
    { group: "训练中心", items: [
      { key: "workspace", label: "工作区", icon: "workspace", href: "workspace.html" },
      { key: "experiments", label: "实验管理", icon: "experiment", href: "experiments.html" },
      { key: "jobs", label: "自定义任务", icon: "job", href: "jobs.html" },
    ]},
    { group: "服务中心", items: [
      { key: "services", label: "在线服务", icon: "service", href: "services.html" },
      { key: "traffic", label: "流量配置", icon: "traffic", href: "traffic.html" },
    ]},
    { group: "资产中心", items: [
      { key: "models", label: "模型仓", icon: "model", href: "models.html" },
      { key: "images", label: "镜像仓", icon: "image", href: "images.html" },
    ]},
    { group: "系统管理", items: [
      { key: "tenants", label: "租户管理", icon: "tenant", href: "tenants.html", roles: ["system-admin", "tenant-admin"] },
      { key: "pools", label: "资源池管理", icon: "pool", href: "resource-pools.html", roles: ["system-admin"] },
    ]},
  ];

  const ROLES = {
    "system-admin": { name: "系统管理员", short: "系统管理员", note: "平台级超管", person: "张伟", email: "zhangwei@axisml.io", initials: "ZW" },
    "tenant-admin": { name: "租户管理员", short: "租户管理员", note: "本租户负责人", person: "李娜", email: "lina@axisml.io", initials: "LN" },
    "user": { name: "普通用户", short: "普通用户", note: "算法 / 推理工程师", person: "王芳", email: "wangfang@axisml.io", initials: "WF" },
  };
  const TENANTS = [
    { id: "llm-lab", name: "大模型研究院", note: "12 成员 · A100/H100" },
    { id: "rec-algo", name: "推荐算法团队", note: "8 成员 · A100" },
    { id: "av-perception", name: "智能驾驶感知", note: "15 成员 · H100/L40S" },
    { id: "risk-ai", name: "风控 AI", note: "6 成员 · 通用 CPU" },
  ];

  const LS = { role: "axisml.role", tenant: "axisml.tenant", collapsed: "axisml.collapsed", theme: "axisml.theme", lang: "axisml.lang" };
  const state = {
    role: localStorage.getItem(LS.role) || "system-admin",
    tenant: localStorage.getItem(LS.tenant) || "all",
    collapsed: localStorage.getItem(LS.collapsed) === "1",
    theme: localStorage.getItem(LS.theme) || "light",
    lang: localStorage.getItem(LS.lang) || "zh",
  };
  if (!ROLES[state.role]) state.role = "system-admin";
  if (["light", "dark", "system"].indexOf(state.theme) === -1) state.theme = "light";
  if (["zh", "en"].indexOf(state.lang) === -1) state.lang = "zh";

  /* ── 主题（浅色 / 深色 / 跟随系统）—— token 级覆盖，即时生效 ────────────── */
  const THEME_NAMES = { light: "浅色", dark: "深色", system: "跟随系统" };
  function prefersDark() { return !!(window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches); }
  function resolveTheme(p) { return p === "system" ? (prefersDark() ? "dark" : "light") : (p === "dark" ? "dark" : "light"); }
  function applyTheme() { document.documentElement.setAttribute("data-theme", resolveTheme(state.theme)); }
  applyTheme();
  if (window.matchMedia) {
    window.matchMedia("(prefers-color-scheme: dark)").addEventListener("change", function () { if (state.theme === "system") applyTheme(); });
  }

  function canSee(item) { return !item.roles || item.roles.indexOf(state.role) !== -1; }
  function tenantLabel() {
    if (state.tenant === "all") return "全部租户";
    const t = TENANTS.find(function (x) { return x.id === state.tenant; });
    return t ? t.name : "全部租户";
  }

  /* ── 渲染侧边栏 ───────────────────────────────────────────────────────── */
  function renderSidebar(activeKey) {
    let html = '<div class="side-brand"><div class="mark">A</div><div class="name">Axis<b>ML</b></div></div><div class="side-scroll">';
    NAV.forEach(function (blk) {
      if (blk.items) {
        const visible = blk.items.filter(canSee);
        if (!visible.length) return;
        html += '<div class="nav-group"><div class="grp-label">' + blk.group + '</div>';
        visible.forEach(function (it) { html += navItem(it, activeKey); });
        html += '</div>';
      } else {
        html += '<div class="nav-group">' + navItem(blk, activeKey) + '</div>';
      }
    });
    html += '</div>';
    return html;
  }
  function navItem(it, activeKey) {
    const on = it.key === activeKey ? " active" : "";
    const count = it.count ? '<span class="badge-dot">' + it.count + '</span>' : '';
    return '<a class="nav-item' + on + '" href="' + it.href + '" title="' + it.label + '">' +
      svg(it.icon) + '<span class="lbl">' + it.label + '</span>' + count + '</a>';
  }

  /* ── 渲染顶栏 ─────────────────────────────────────────────────────────── */
  function renderTopbar() {
    const r = ROLES[state.role];
    const isAdmin = state.role === "system-admin";
    return '' +
      '<button class="icon-btn menu-trigger" id="navToggle" aria-label="菜单">' + svg("menu") + '</button>' +
      '<button class="icon-btn" id="sideCollapse" aria-label="折叠" title="折叠侧栏">' + svg("layers") + '</button>' +
      '<div class="search"><span>' + svg("search") + '</span><input placeholder="搜索任务 / 服务 / 模型 / 镜像…" /><kbd>⌘K</kbd></div>' +
      '<div class="spacer"></div>' +
      '<div style="position:relative">' +
        '<button class="switch" id="roleBtn"><span class="cap">角色</span><span>' + r.short + '</span><span class="chev">' + svg("chevron") + '</span></button>' +
        roleMenu() +
      '</div>' +
      '<button class="icon-btn" title="帮助">' + svg("help") + '</button>' +
      '<button class="icon-btn" title="通知">' + svg("bell") + '<span class="ping"></span></button>' +
      '<div class="user-wrap">' +
        '<button class="avatar" id="userBtn" aria-label="用户菜单" title="' + r.person + '">' + r.initials + '</button>' +
        userMenu(isAdmin, r) +
      '</div>';
  }
  function userMenu(isAdmin, r) {
    return '<div class="menu user-menu" id="userMenu">' +
      '<div class="user-card">' +
        '<div class="avatar">' + r.initials + '</div>' +
        '<div class="u-meta"><div class="u-name">' + r.person + '</div><div class="u-sub">' + r.email + '</div></div>' +
      '</div>' +
      '<hr/>' +
      // 切换租户：当前租户 + 左侧 flyout 选择
      '<div class="menu-sub">' +
        '<div class="menu-item tenant-trigger has-flyout">' +
          '<div class="ti-text"><span class="ti-label">所属租户</span><span class="ti-val">' + tenantLabel() + '</span></div>' +
          svg("chevronR", "caret") +
        '</div>' +
        '<div class="flyout">' + tenantFlyout(isAdmin) + '</div>' +
      '</div>' +
      '<hr/>' +
      // 切换语言（内联分段控件，即时生效）
      '<div class="opt-row"><span class="opt-label">语言</span>' + langSeg() + '</div>' +
      // 切换主题（浅色 / 深色 / 跟随系统，图标分段控件）
      '<div class="opt-row"><span class="opt-label">主题</span>' +
        '<div class="segmented theme-seg">' + themeBtn("light", "sun") + themeBtn("dark", "moon") + themeBtn("system", "monitor") + '</div>' +
      '</div>' +
      '<hr/>' +
      // 退出登录
      '<a class="menu-item danger logout-row" data-logout href="#"><span>退出登录</span>' + svg("logout", "mi") + '</a>' +
    '</div>';
  }
  function themeBtn(val, icon) {
    return '<button data-theme-set="' + val + '" title="' + THEME_NAMES[val] + '" aria-label="' + THEME_NAMES[val] + '"' + (state.theme === val ? ' class="on"' : '') + '>' + svg(icon, "seg-ic") + '</button>';
  }
  function langSeg() {
    const langs = [["zh", "中文"], ["en", "English"]];
    let h = '<div class="segmented lang-seg">';
    langs.forEach(function (l) {
      h += '<button data-lang="' + l[0] + '"' + (state.lang === l[0] ? ' class="on"' : '') + '>' + l[1] + '</button>';
    });
    return h + '</div>';
  }
  function tenantFlyout(isAdmin) {
    let h = '<div class="menu-label">切换租户作用域</div>';
    if (isAdmin) {
      h += menuItem("all", "全部租户", "平台全局视图", state.tenant === "all");
      h += '<hr/>';
    }
    TENANTS.forEach(function (t) { h += menuItem(t.id, t.name, t.note, state.tenant === t.id); });
    return h;
  }
  function roleMenu() {
    let h = '<div class="menu" id="roleMenu"><div class="menu-label">切换演示角色</div>';
    Object.keys(ROLES).forEach(function (k) {
      h += '<a class="menu-item' + (state.role === k ? ' sel' : '') + '" data-role="' + k + '">' +
        '<div>' + ROLES[k].name + '<small>' + ROLES[k].note + '</small></div>' +
        (state.role === k ? '<span class="ck">' + svg("check") + '</span>' : '') + '</a>';
    });
    h += '</div>';
    return h;
  }
  function menuItem(id, name, note, sel) {
    return '<a class="menu-item' + (sel ? ' sel' : '') + '" data-tenant="' + id + '">' +
      '<div>' + name + '<small>' + note + '</small></div>' + (sel ? '<span class="ck">' + svg("check") + '</span>' : '') + '</a>';
  }

  /* ── 组装外壳 ─────────────────────────────────────────────────────────── */
  function mount() {
    const body = document.body;
    const activeKey = body.getAttribute("data-nav") || "";
    const page = document.getElementById("page");
    const pageHTML = page ? page.outerHTML : '<main class="page" id="page"></main>';

    const shell = document.createElement("div");
    shell.className = "app-shell" + (state.collapsed ? " collapsed" : "");
    shell.innerHTML =
      '<aside class="sidebar" id="sidebar">' + renderSidebar(activeKey) + '</aside>' +
      '<div class="app-main"><header class="topbar" id="topbar">' + renderTopbar() + '</header>' + pageHTML + '</div>';

    if (page) page.remove();
    body.insertBefore(shell, body.firstChild);

    // toast container
    if (!document.querySelector(".toast-wrap")) {
      const tw = document.createElement("div"); tw.className = "toast-wrap"; document.body.appendChild(tw);
    }
    wire();
  }

  /* ── 交互绑定 ─────────────────────────────────────────────────────────── */
  function wire() {
    const sidebar = document.getElementById("sidebar");
    const shell = document.querySelector(".app-shell");

    // collapse
    const cb = document.getElementById("sideCollapse");
    if (cb) cb.addEventListener("click", function () {
      state.collapsed = !state.collapsed;
      shell.classList.toggle("collapsed", state.collapsed);
      localStorage.setItem(LS.collapsed, state.collapsed ? "1" : "0");
    });
    // mobile nav
    const nt = document.getElementById("navToggle");
    if (nt) nt.addEventListener("click", function () { sidebar.classList.toggle("mobile-open"); });

    // dropdown toggles
    setupMenu("roleBtn", "roleMenu");
    setupMenu("userBtn", "userMenu");

    // tenant select
    document.querySelectorAll("[data-tenant]").forEach(function (el) {
      el.addEventListener("click", function () {
        state.tenant = el.getAttribute("data-tenant");
        localStorage.setItem(LS.tenant, state.tenant);
        location.reload();
      });
    });
    // role select
    document.querySelectorAll("[data-role]").forEach(function (el) {
      el.addEventListener("click", function () {
        state.role = el.getAttribute("data-role");
        localStorage.setItem(LS.role, state.role);
        // 普通用户没有系统管理权限，若当前在管理页则回首页
        const k = document.body.getAttribute("data-nav");
        const restricted = { tenants: ["system-admin", "tenant-admin"], pools: ["system-admin"] };
        if (restricted[k] && restricted[k].indexOf(state.role) === -1) { location.href = "index.html"; return; }
        location.reload();
      });
    });

    // 主题切换（浅色 / 深色 / 跟随系统）—— 即时生效，无需刷新；菜单保持打开
    document.querySelectorAll("[data-theme-set]").forEach(function (el) {
      el.addEventListener("click", function () {
        state.theme = el.getAttribute("data-theme-set");
        localStorage.setItem(LS.theme, state.theme);
        applyTheme();
        toast("主题已切换为「" + THEME_NAMES[state.theme] + "」");
      });
    });

    // 语言切换（内联分段控件：持久化 + 标记当前项，界面文案不整体翻译）
    document.querySelectorAll(".lang-seg [data-lang]").forEach(function (el) {
      el.addEventListener("click", function (e) {
        e.stopPropagation();
        const code = el.getAttribute("data-lang");
        const seg = el.closest(".lang-seg");
        if (seg) seg.querySelectorAll("button").forEach(function (x) { x.classList.toggle("on", x === el); });
        if (state.lang === code) return;
        state.lang = code;
        localStorage.setItem(LS.lang, code);
        document.documentElement.setAttribute("lang", code === "en" ? "en" : "zh-CN");
        toast(code === "en" ? "界面语言已切换为 English（演示）" : "界面语言已切换为简体中文");
      });
    });

    // 退出登录（二次确认）
    document.querySelectorAll("[data-logout]").forEach(function (el) {
      el.addEventListener("click", function (e) {
        e.preventDefault();
        document.querySelectorAll(".menu.open").forEach(function (m) { m.classList.remove("open"); });
        openConfirm({
          title: "退出登录",
          desc: "确定要退出当前登录吗？退出后需要重新登录才能继续访问控制台。",
          okLabel: "退出登录",
          toast: "已退出登录（演示）"
        });
      });
    });

    // close menus on outside click
    document.addEventListener("click", function (e) {
      if (!e.target.closest("[id$=Btn]") && !e.target.closest(".menu")) {
        document.querySelectorAll(".menu.open").forEach(function (m) { m.classList.remove("open"); });
      }
    });

    // tabs
    document.querySelectorAll("[data-tabs]").forEach(function (group) {
      const btns = group.querySelectorAll(".tabs button");
      btns.forEach(function (b) {
        b.addEventListener("click", function () {
          btns.forEach(function (x) { x.classList.remove("on"); });
          b.classList.add("on");
          const target = b.getAttribute("data-tab");
          group.querySelectorAll(".tabpane").forEach(function (p) {
            p.classList.toggle("on", p.getAttribute("data-pane") === target);
          });
        });
      });
    });

    // segmented
    document.querySelectorAll(".segmented:not([data-view-switch])").forEach(function (seg) {
      seg.querySelectorAll("button").forEach(function (b) {
        b.addEventListener("click", function () {
          seg.querySelectorAll("button").forEach(function (x) { x.classList.remove("on"); });
          b.classList.add("on");
        });
      });
    });

    // card / list dual view (persisted per page)
    document.querySelectorAll("[data-view-switch]").forEach(function (seg) {
      const key = "axisml.view." + seg.getAttribute("data-view-switch");
      const panes = document.querySelectorAll("[data-view-pane]");
      const btns = seg.querySelectorAll("button");
      function apply(view) {
        btns.forEach(function (b) { b.classList.toggle("on", b.getAttribute("data-view") === view); });
        panes.forEach(function (p) { p.hidden = p.getAttribute("data-view-pane") !== view; });
      }
      apply(localStorage.getItem(key) || seg.getAttribute("data-view-default") || "cards");
      btns.forEach(function (b) {
        b.addEventListener("click", function () {
          const v = b.getAttribute("data-view");
          localStorage.setItem(key, v);
          apply(v);
        });
      });
    });

    // drawer open/close
    document.querySelectorAll("[data-drawer-open]").forEach(function (el) {
      el.addEventListener("click", function () { openDrawer(el.getAttribute("data-drawer-open")); });
    });
    document.querySelectorAll("[data-drawer-close]").forEach(function (el) {
      el.addEventListener("click", function () { closeDrawers(); });
    });
    const ov = document.getElementById("overlay");
    if (ov) ov.addEventListener("click", closeDrawers);
    document.addEventListener("keydown", function (e) { if (e.key === "Escape") closeDrawers(); });

    // generic toggles
    document.querySelectorAll(".toggle").forEach(function (t) {
      t.addEventListener("click", function () { t.classList.toggle("on"); });
    });

    // resource unit picker
    document.querySelectorAll("[data-pick-group]").forEach(function (g) {
      g.querySelectorAll(".pick").forEach(function (p) {
        p.addEventListener("click", function () {
          g.querySelectorAll(".pick").forEach(function (x) { x.classList.remove("on"); });
          p.classList.add("on");
        });
      });
    });

    // 卡片整体点击进入详情（忽略卡片内的按钮 / 链接 / 表单控件）
    document.querySelectorAll("[data-card-href]").forEach(function (card) {
      card.style.cursor = "pointer";
      card.addEventListener("click", function (e) {
        if (e.target.closest("a, button, input, select, textarea, label")) return;
        window.location.href = card.getAttribute("data-card-href");
      });
    });

    // 数据卷可重复挂载行
    document.querySelectorAll("[data-vol-list]").forEach(function (list) {
      var add = list.parentElement && list.parentElement.querySelector("[data-vol-add]");
      function sync() {
        var rows = list.querySelectorAll(".vol-row");
        rows.forEach(function (r) { if (rows.length <= 1) r.setAttribute("data-only", ""); else r.removeAttribute("data-only"); });
      }
      function addRow() {
        var first = list.querySelector(".vol-row");
        if (!first) return;
        var clone = first.cloneNode(true);
        clone.querySelectorAll("input").forEach(function (i) { if (i.type !== "checkbox" && i.type !== "radio") i.value = ""; });
        var sel = clone.querySelector("select"); if (sel) sel.selectedIndex = 0;
        list.appendChild(clone);
        var focusEl = clone.querySelector("input"); if (focusEl) focusEl.focus();
        sync();
      }
      if (add) {
        add.addEventListener("click", addRow);
        add.addEventListener("keydown", function (e) { if (e.key === "Enter" || e.key === " ") { e.preventDefault(); addRow(); } });
      }
      list.addEventListener("click", function (e) {
        var btn = e.target.closest("[data-vol-remove]");
        if (!btn) return;
        if (list.querySelectorAll(".vol-row").length <= 1) return;
        btn.closest(".vol-row").remove();
        sync();
      });
      sync();
    });

    // 容忍配置：贴合 Kubernetes Toleration 语义 —— operator=Exists 时 value 不适用
    function syncTolRow(row) {
      var op = row.querySelector("[data-tol-op]");
      var val = row.querySelector("[data-tol-val]");
      if (!op || !val) return;
      var exists = op.value === "Exists";
      val.disabled = exists;
      if (exists) { val.value = ""; val.setAttribute("placeholder", "Exists 无需取值"); }
      else val.setAttribute("placeholder", "如 true");
    }
    document.querySelectorAll("[data-tol-list]").forEach(function (list) {
      list.querySelectorAll(".vol-row").forEach(syncTolRow);
      list.addEventListener("change", function (e) {
        var op = e.target.closest("[data-tol-op]");
        if (op) syncTolRow(op.closest(".vol-row"));
      });
      var add = list.parentElement && list.parentElement.querySelector("[data-vol-add]");
      if (add) add.addEventListener("click", function () {
        setTimeout(function () { list.querySelectorAll(".vol-row").forEach(syncTolRow); }, 0);
      });
    });

    // 资源配额：多 Tab（每个 Tab 一个资源池），池内列出自有资源单元；数量直接输入，0 置灰并按数量排序
    document.querySelectorAll("[data-pool-tabs]").forEach(function (root) {
      var nav = root.querySelector("[data-ptab-nav]");
      var panes = root.querySelector("[data-ptab-panes]");
      if (!nav || !panes) return;

      function activate(id) {
        nav.querySelectorAll(".ptab").forEach(function (t) { t.classList.toggle("on", t.getAttribute("data-ptab") === id); });
        panes.querySelectorAll(".ptab-pane").forEach(function (p) { p.classList.toggle("on", p.getAttribute("data-ppane") === id); });
      }

      function qtyOf(row) {
        var input = row.querySelector(".step-val");
        var v = input ? parseInt(input.value, 10) : 0;
        return isNaN(v) || v < 0 ? 0 : v;
      }
      // 数量为 0 时卡片置灰
      function syncRow(row) { row.classList.toggle("is-zero", qtyOf(row) === 0); }
      // 按数量降序排序（0 沉底，等量保持原相对顺序）
      function sortList(list) {
        Array.prototype.slice.call(list.querySelectorAll(".q-row"))
          .map(function (r, i) { return { r: r, q: qtyOf(r), i: i }; })
          .sort(function (a, b) { return b.q - a.q || a.i - b.i; })
          .forEach(function (o) { list.appendChild(o.r); });
      }

      // 初始：置灰 + 排序
      panes.querySelectorAll(".qp-units").forEach(function (list) {
        list.querySelectorAll(".q-row").forEach(syncRow);
        sortList(list);
      });

      root.addEventListener("click", function (e) {
        // 切换资源池 Tab
        var tab = e.target.closest(".ptab");
        if (tab && tab.getAttribute("data-ptab")) { activate(tab.getAttribute("data-ptab")); }
      });

      // 手动输入数量：实时置灰，失焦/提交后重新排序
      root.addEventListener("input", function (e) {
        var input = e.target.closest(".step-val");
        if (input) { var r = input.closest(".q-row"); if (r) syncRow(r); }
      });
      root.addEventListener("change", function (e) {
        var input = e.target.closest(".step-val");
        if (input) {
          var v = parseInt(input.value, 10);
          input.value = isNaN(v) || v < 0 ? 0 : v;
          var r = input.closest(".q-row");
          if (r) syncRow(r);
          var l = input.closest(".qp-units");
          if (l) sortList(l);
        }
      });
    });

    // 生成一个可移除的已选标签 chip
    function makeRemovableChip(text) {
      var chip = document.createElement("span");
      chip.className = "tag-opt on removable";
      chip.appendChild(document.createTextNode(text));
      var b = document.createElement("button");
      b.type = "button"; b.setAttribute("aria-label", "移除"); b.setAttribute("data-tag-remove", ""); b.textContent = "✕";
      chip.appendChild(b);
      return chip;
    }

    // 分面标签：multi 多选切换 / single 单选；支持手动输入追加
    document.querySelectorAll("[data-tag-group]").forEach(function (g) {
      var single = g.getAttribute("data-tag-group") === "single";
      var list = g.querySelector("[data-tag-list]") || g;
      g.addEventListener("click", function (e) {
        var rm = e.target.closest("[data-tag-remove]");
        if (rm) { var c = rm.closest(".tag-opt"); if (c) c.remove(); return; }
        var opt = e.target.closest(".tag-opt");
        if (!opt || !g.contains(opt)) return;
        if (single) {
          var was = opt.classList.contains("on");
          g.querySelectorAll(".tag-opt").forEach(function (x) { x.classList.remove("on"); });
          if (!was) opt.classList.add("on");
        } else {
          opt.classList.toggle("on");
        }
      });
      // 行内手动输入：回车追加自定义项（chip 插入到输入框之前）
      var fInput = g.querySelector("[data-facet-input]");
      if (fInput) {
        fInput.addEventListener("keydown", function (e) {
          if (e.key !== "Enter") return;
          e.preventDefault();
          var v = (fInput.value || "").trim();
          if (!v) return;
          list.insertBefore(makeRemovableChip(v), fInput);
          fInput.value = "";
        });
      }
    });

    // 自定义标签：键 / 值分开输入，组合为 key:value，可移除
    document.querySelectorAll("[data-custom-tags]").forEach(function (box) {
      var keyEl = box.querySelector("[data-tag-key]");
      var valEl = box.querySelector("[data-tag-val]");
      var add = box.querySelector("[data-tag-add]");
      var list = box.querySelector("[data-tag-list]");
      if (!list) return;
      function addTag() {
        var k = (keyEl && keyEl.value || "").trim();
        var v = (valEl && valEl.value || "").trim();
        if (!k && !v) return;
        var text = (k && v) ? (k + ":" + v) : (k || v);
        list.appendChild(makeRemovableChip(text));
        if (keyEl) keyEl.value = "";
        if (valEl) valEl.value = "";
        if (keyEl) keyEl.focus();
      }
      if (add) add.addEventListener("click", addTag);
      [keyEl, valEl].forEach(function (el) {
        if (el) el.addEventListener("keydown", function (e) { if (e.key === "Enter") { e.preventDefault(); addTag(); } });
      });
      list.addEventListener("click", function (e) {
        var rm = e.target.closest("[data-tag-remove]");
        if (rm) rm.closest(".tag-opt").remove();
      });
    });

    // 参数量滑块：拖动选择常见规模，输入框既显示当前值也可手动输入精确参数量
    document.querySelectorAll("[data-param-slider]").forEach(function (ps) {
      var range = ps.querySelector("[data-param-range]");
      var out = ps.querySelector("[data-param-input]");
      var STOPS = ["<1B", "1B", "3B", "7B", "8B", "13B", "32B", "70B", ">100B"];
      if (!range || !out) return;
      function fromRange() { out.value = STOPS[parseInt(range.value, 10) || 0] || STOPS[0]; }
      range.addEventListener("input", fromRange);
      if (!out.value) fromRange();
    });

    // 文件拖放上传区（通过 Web 上传）
    document.querySelectorAll("[data-dropzone]").forEach(function (dz) {
      var input = dz.querySelector("[data-dropzone-input]");
      var files = dz.parentElement && dz.parentElement.querySelector("[data-dropzone-files]");
      function fmt(n) { return n > 1e9 ? (n / 1e9).toFixed(1) + " GB" : (n / 1e6).toFixed(1) + " MB"; }
      function render(list) {
        if (!files) return;
        files.innerHTML = "";
        if (!list || !list.length) { files.hidden = true; return; }
        files.hidden = false;
        Array.prototype.forEach.call(list, function (f) {
          var row = document.createElement("div");
          row.className = "dz-file";
          var name = document.createElement("span"); name.className = "mono"; name.textContent = f.name;
          var size = document.createElement("span"); size.className = "muted mono"; size.textContent = fmt(f.size || 0);
          row.appendChild(name); row.appendChild(size);
          files.appendChild(row);
        });
      }
      if (input) input.addEventListener("change", function () { render(input.files); });
      ["dragenter", "dragover"].forEach(function (ev) {
        dz.addEventListener(ev, function (e) { e.preventDefault(); dz.classList.add("drag"); });
      });
      ["dragleave", "drop"].forEach(function (ev) {
        dz.addEventListener(ev, function (e) { e.preventDefault(); dz.classList.remove("drag"); });
      });
      dz.addEventListener("drop", function (e) {
        var dt = e.dataTransfer;
        if (dt && dt.files && dt.files.length) {
          if (input) { try { input.files = dt.files; } catch (_) {} }
          render(dt.files);
        }
      });
    });

    // 模型仓列表检索（名称 + 描述，同时作用于卡片视图与列表视图）
    document.querySelectorAll("[data-model-search]").forEach(function (input) {
      var cards = document.querySelectorAll('[data-view-pane="cards"] .art-card');
      var rows = document.querySelectorAll('[data-view-pane="list"] tbody tr');
      function filter(el, nameSel, descSel, q) {
        var name = (el.querySelector(nameSel) || {}).textContent || "";
        var desc = (el.querySelector(descSel) || {}).textContent || "";
        el.hidden = !!q && (name + " " + desc).toLowerCase().indexOf(q) === -1;
      }
      input.addEventListener("input", function () {
        var q = input.value.trim().toLowerCase();
        cards.forEach(function (c) { filter(c, ".ac-name", ".ac-desc", q); });
        rows.forEach(function (r) { filter(r, ".t-name", ".t-sub", q); });
      });
    });

    // 模型版本列表检索（按版本名称 + 描述过滤）
    document.querySelectorAll("[data-ver-search]").forEach(function (input) {
      var scope = input.closest(".drawer-body") || document;
      var items = scope.querySelectorAll(".ver-item");
      input.addEventListener("input", function () {
        var q = input.value.trim().toLowerCase();
        items.forEach(function (it) {
          var name = (it.querySelector(".ver-name") || {}).textContent || "";
          var desc = (it.querySelector(".ver-desc") || {}).textContent || "";
          it.hidden = !!q && (name + " " + desc).toLowerCase().indexOf(q) === -1;
        });
      });
    });

    // 存储地址：切换存储类型时更新地址输入框的 placeholder
    document.querySelectorAll("[data-addr-type]").forEach(function (sel) {
      var grid = sel.closest(".form-grid") || sel.closest("[data-pane]") || document;
      var input = grid.querySelector("[data-addr-input]");
      var PH = {
        s3: "s3://bucket/prefix",
        oci: "oci://registry/namespace/repo:tag",
        http: "https://example.com/model.safetensors",
        hf: "hf://organization/model-name",
        custom: "输入存储地址"
      };
      function apply() { if (input) input.setAttribute("placeholder", PH[sel.value] || "输入存储地址"); }
      sel.addEventListener("change", apply);
      apply();
    });

    // 输入字符过滤：data-en-only（字母/数字/连字符）、data-tag-name（镜像 tag：字母/数字/._-）
    document.addEventListener("input", function (e) {
      var t = e.target;
      if (!t || !t.matches) return;
      if (t.matches("[data-en-only]")) {
        var clean = t.value.replace(/[^A-Za-z0-9-]/g, "");
        if (clean !== t.value) t.value = clean;
      } else if (t.matches("[data-tag-name]")) {
        var ct = t.value.replace(/[^A-Za-z0-9._-]/g, "").replace(/^[.-]+/, "").slice(0, 128);
        if (ct !== t.value) t.value = ct;
      }
    });

    // 删除前置阻断确认弹窗
    document.querySelectorAll("[data-confirm]").forEach(function (el) {
      el.addEventListener("click", function (e) {
        e.preventDefault();
        openConfirm({
          title: el.getAttribute("data-confirm"),
          desc: el.getAttribute("data-confirm-desc"),
          info: el.getAttribute("data-confirm-info"),
          block: el.getAttribute("data-confirm-block"),
          blocked: el.hasAttribute("data-confirm-blocked"),
          okLabel: el.getAttribute("data-confirm-ok"),
          toast: el.getAttribute("data-confirm-toast")
        });
      });
    });

    // confirm actions (toast)
    document.querySelectorAll("[data-toast]").forEach(function (el) {
      el.addEventListener("click", function (e) {
        if (el.tagName === "A" && el.getAttribute("href") === "#") e.preventDefault();
        toast(el.getAttribute("data-toast"));
        if (el.hasAttribute("data-drawer-submit")) closeDrawers();
      });
    });

    // ── 资源单元表单：新建 / 编辑 复用同一抽屉 ───────────────────────────────
    var unitForm = document.getElementById("unitFormDrawer");
    if (unitForm) {
      var ufq = function (s) { return unitForm.querySelector(s); };
      // requests/limits 联动：开关开启时 limits 锁定并跟随 requests
      var syncResLock = function () {
        var on = !!(ufq("[data-uf-lock]") || {}).checked;
        unitForm.querySelectorAll("[data-uf-lim]").forEach(function (lim) {
          var key = lim.getAttribute("data-uf-lim");
          var req = ufq('[data-uf-req="' + key + '"]');
          lim.disabled = on;
          if (on && req) lim.value = req.value;
        });
      };
      var fillUnitForm = function (mode, d) {
        d = d || {};
        var edit = mode === "edit";
        var title = ufq("[data-uf-title]"); if (title) title.textContent = edit ? "编辑资源单元" : "新建资源单元";
        var setVal = function (sel, v) { var el = ufq(sel); if (el) el.value = v || ""; };
        setVal("[data-uf-name]", d.name); setVal("[data-uf-desc]", d.desc);
        setVal('[data-uf-req="cpu"]', d.cpu); setVal('[data-uf-lim="cpu"]', d.cpuLim || d.cpu);
        setVal('[data-uf-req="mem"]', d.mem); setVal('[data-uf-lim="mem"]', d.memLim || d.mem);
        setVal("[data-uf-gpu]", d.gpu);
        var lock = ufq("[data-uf-lock]"); if (lock) lock.checked = !(d.cpuLim && d.cpuLim !== d.cpu) && !(d.memLim && d.memLim !== d.mem);
        syncResLock();
        var sel = ufq("[data-uf-sel]");
        if (sel) {
          var chips = (d.sel || "").split(",").map(function (s) { return s.trim(); }).filter(Boolean)
            .map(function (s) { return '<span class="tag mono">' + s + ' ✕</span>'; }).join("");
          sel.innerHTML = chips + '<span class="tag mono" style="border-style:dashed;color:var(--muted)">+ 添加</span>';
        }
        var btn = ufq("[data-uf-submit]");
        if (btn) { btn.textContent = edit ? "保存" : "创建资源单元"; btn.setAttribute("data-toast", edit ? "资源单元已保存" : "资源单元已创建"); }
      };
      var lockEl = ufq("[data-uf-lock]");
      if (lockEl) lockEl.addEventListener("change", syncResLock);
      unitForm.addEventListener("input", function (e) {
        var req = e.target.closest("[data-uf-req]");
        if (!req || !lockEl || !lockEl.checked) return;
        var lim = ufq('[data-uf-lim="' + req.getAttribute("data-uf-req") + '"]');
        if (lim) lim.value = req.value;
      });
      document.querySelectorAll("[data-unit-new]").forEach(function (b) {
        b.addEventListener("click", function () { fillUnitForm("create"); openDrawer("unitFormDrawer"); });
      });
      document.querySelectorAll("[data-unit-edit]").forEach(function (b) {
        b.addEventListener("click", function () {
          fillUnitForm("edit", {
            name: b.getAttribute("data-name"), desc: b.getAttribute("data-desc"),
            cpu: b.getAttribute("data-cpu"), mem: b.getAttribute("data-mem"), gpu: b.getAttribute("data-gpu"),
            sel: b.getAttribute("data-sel")
          });
          openDrawer("unitFormDrawer");
        });
      });
    }
  }

  function setupMenu(btnId, menuId) {
    const btn = document.getElementById(btnId), menu = document.getElementById(menuId);
    if (!btn || !menu) return;
    btn.addEventListener("click", function (e) {
      e.stopPropagation();
      const wasOpen = menu.classList.contains("open");
      document.querySelectorAll(".menu.open").forEach(function (m) { m.classList.remove("open"); });
      menu.classList.toggle("open", !wasOpen);
    });
  }

  function openDrawer(id) {
    const d = document.getElementById(id); if (!d) return;
    ensureOverlay();
    document.getElementById("overlay").classList.add("open");
    d.classList.add("open");
  }
  function closeDrawers() {
    document.querySelectorAll(".drawer.open").forEach(function (d) { d.classList.remove("open"); });
    document.querySelectorAll(".modal.open").forEach(function (m) { m.classList.remove("open"); });
    const ov = document.getElementById("overlay"); if (ov) ov.classList.remove("open");
  }

  // 确认 / 删除前置阻断弹窗
  function ensureConfirm() {
    if (document.getElementById("confirmModal")) return;
    ensureOverlay();
    const m = document.createElement("div");
    m.id = "confirmModal"; m.className = "modal";
    m.innerHTML =
      '<div class="modal-head"><span class="warn-ico">' + svg("help") + '</span><span id="cfTitle"></span></div>' +
      '<div class="modal-body" id="cfBody"></div>' +
      '<div class="modal-foot"><button class="btn" id="cfCancel">取消</button><button class="btn btn-danger" id="cfOk">确认删除</button></div>';
    document.body.appendChild(m);
    m.querySelector("#cfCancel").addEventListener("click", closeDrawers);
  }
  function openConfirm(opts) {
    ensureConfirm();
    const m = document.getElementById("confirmModal");
    m.querySelector("#cfTitle").textContent = opts.title || "确认操作";
    let body = opts.desc ? "<p>" + opts.desc + "</p>" : "";
    if (opts.info) body += '<div class="block-info warn-soft"><div class="bi-title">' + svg("help") + "提示</div>" + opts.info + "</div>";
    if (opts.block) body += '<div class="block-info danger-soft"><div class="bi-title">× 阻断</div>' + opts.block + "</div>";
    m.querySelector("#cfBody").innerHTML = body;
    const ok = m.querySelector("#cfOk");
    ok.textContent = opts.okLabel || "确认删除";
    ok.disabled = !!opts.blocked;
    ok.onclick = function () { closeDrawers(); if (opts.toast) toast(opts.toast); };
    document.getElementById("overlay").classList.add("open");
    m.classList.add("open");
  }
  function ensureOverlay() {
    if (!document.getElementById("overlay")) {
      const o = document.createElement("div"); o.id = "overlay"; o.className = "overlay";
      o.addEventListener("click", closeDrawers); document.body.appendChild(o);
    }
  }

  function toast(msg) {
    const wrap = document.querySelector(".toast-wrap"); if (!wrap) return;
    const t = document.createElement("div"); t.className = "toast";
    t.innerHTML = svg("check") + "<span>" + msg + "</span>";
    wrap.appendChild(t);
    setTimeout(function () { t.style.opacity = "0"; t.style.transition = "opacity .3s"; }, 2400);
    setTimeout(function () { t.remove(); }, 2800);
  }

  /* ── 图表小工具 ───────────────────────────────────────────────────────── */
  // 生成一条折线 path（在 0..w, 0..h 视口内，data 归一化）
  function linePath(data, w, h, pad) {
    pad = pad || 0;
    const max = Math.max.apply(null, data), min = Math.min.apply(null, data);
    const rng = (max - min) || 1;
    const step = (w - pad * 2) / (data.length - 1);
    return data.map(function (v, i) {
      const x = pad + i * step;
      const y = pad + (h - pad * 2) * (1 - (v - min) / rng);
      return (i ? "L" : "M") + x.toFixed(1) + " " + y.toFixed(1);
    }).join(" ");
  }
  function sparkline(el) {
    const data = el.getAttribute("data-spark").split(",").map(Number);
    const w = 120, h = 36;
    const p = linePath(data, w, h, 3);
    const color = el.getAttribute("data-color") || "var(--accent)";
    const area = p + " L" + (w - 3) + " " + (h - 3) + " L3 " + (h - 3) + " Z";
    el.innerHTML =
      '<svg class="spark" viewBox="0 0 ' + w + ' ' + h + '" preserveAspectRatio="none">' +
      '<defs><linearGradient id="sg' + Math.round(data[0] * 1000) + '" x1="0" x2="0" y1="0" y2="1">' +
      '<stop offset="0" stop-color="' + color + '" stop-opacity="0.18"/><stop offset="1" stop-color="' + color + '" stop-opacity="0"/></linearGradient></defs>' +
      '<path d="' + area + '" fill="url(#sg' + Math.round(data[0] * 1000) + ')"/>' +
      '<path d="' + p + '" fill="none" stroke="' + color + '" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"/></svg>';
  }

  // 暴露给页面使用
  window.AxisUI = { svg: svg, toast: toast, linePath: linePath, openDrawer: openDrawer, closeDrawers: closeDrawers, openConfirm: openConfirm, state: state };

  function boot() {
    mount();
    document.querySelectorAll("[data-spark]").forEach(sparkline);
  }
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", boot);
  else boot();
})();
