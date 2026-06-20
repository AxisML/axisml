# platform-frontend — page authoring conventions

This SPA faithfully reproduces the static prototype in `docs/product_design/prototype/`.
Each prototype `*.html` page becomes one React page component under `src/pages/`.
The shared shell (sidebar + topbar) is already injected by `AppShell`; pages render
ONLY the content that the prototype puts inside `<main class="page" id="page">`
(WITHOUT the `id="page"`), i.e. start from `<main className="page">`.

## Golden rule
Translate the prototype HTML **verbatim** into JSX — same DOM structure, same
`className` strings, same text, same inline `style`. The design system lives in
`src/styles/app.css` (copied from the prototype). Do not restyle. The reference
implementations are `src/pages/Dashboard.tsx` and `src/pages/Jobs.tsx` — match
their patterns.

## HTML → JSX mechanics
- `class=` → `className=`; `for=` → `htmlFor=`; `stroke-width` → `strokeWidth`,
  `stroke-linecap` → `strokeLinecap`, `stroke-dasharray` → `strokeDasharray`,
  `stop-color`/`stop-opacity` → `stopColor`/`stopOpacity`, etc.
- `style="a:b;c:d"` → `style={{ a: "b", c: "d" }}`. Keep `var(--...)` values as strings.
- Self-close void tags (`<input />`, `<hr />`, `<br />`).
- Inline event-free demo `<svg>...</svg>` icons: prefer `<Icon name="..." />` from
  `@/components/Icon` when the glyph exists in the icon map (dashboard, job, plus,
  search, refresh, play, stop, trash→"trash", edit→"edit", x, copy, download,
  chevron, chevronR, etc.). For one-off glyphs not in the map (e.g. the "eye"
  detail icon `M2 12s3.5-7 10-7...`), inline the `<svg>` converted to JSX.

## Navigation & interactions (replace the prototype's `data-*` + app.js)
- Links between pages → `<Link to="/...">` from `react-router-dom`. Route map:
  jobs `/jobs` · job-detail `/jobs/:name` · run-detail `/jobs/:name/runs/:run`
  (and `/experiments/:name/runs/:run`) · experiments `/experiments` ·
  experiment-detail `/experiments/:name` · workspace `/workspaces` ·
  workspace-detail `/workspaces/:name` · services `/services` · service-detail
  `/services/:name` · traffic `/traffic` · traffic-detail `/traffic/:name` ·
  models `/models` · images `/images` · tenants `/tenants` · resource-pools
  `/resource-pools` · dashboard `/`.
- `data-drawer-open="x"` → component state `useState` toggling a `<Drawer>`
  (`@/components/Drawer`). `data-drawer-close` → the drawer's `onClose`.
- `data-toast="msg"` → `const { toast } = useUI()` (`@/app/ui`); call `toast("msg")`
  in the button's `onClick` (and close the drawer if it had `data-drawer-submit`).
- `data-confirm=...` (delete confirm) → `const { confirm } = useUI()` then
  `confirm({ title, desc, info, block, blocked, okLabel, toast })`. Map the
  `data-confirm-*` attributes to those fields. `data-confirm-block` → `block`,
  presence of `data-confirm-blocked` → `blocked: true`.
- `[data-tabs]` / `.tabs` + `.tabpane` → `<Tabs tabs={[{key,label,count,content}]} />`
  from `@/components/Tabs`.
- `.segmented` (view/range toggles) → `<Segmented options={[...]} defaultValue />`
  from `@/components/Segmented`, OR local `useState` if it drives a card/list view.
- `[data-view-switch]` card/list dual view → local `useState<"cards"|"list">` and
  conditionally render the two panes.
- `.pick-grid [data-pick-group]` → `<PickGrid options={[{title,spec}]} />` from
  `@/components/forms`.
- `[data-vol-list]` data-volume rows → `<VolList />` from `@/components/forms`.
- `.fieldset-title` numbered headers → `<FieldsetTitle n={1}>基本信息</FieldsetTitle>`.
- `.toggle` → a `<button className="toggle">` with local `useState` toggling `on`.
- `<details><summary>` "高级设置" → keep as native `<details>`/`<summary>`.
- Generic toggles/inputs that were purely cosmetic in the prototype can stay
  uncontrolled (`defaultValue`), since this is a faithful UI port, not a real form.

## List pages — wire the generated client (REQUIRED)
Every LIST page must call its hook from `@/api/hooks` (e.g. `useJobs`,
`useServices`, `useModels`, `useTenants`, `useResourcePools`, `useWorkspaces`,
`useExperiments`, `useTrafficPolicies`, `useImages`). Define a `FALLBACK` const
holding the prototype's exact demo rows and render `data?.items?.map(...) ??
FALLBACK` (see `Jobs.tsx`). The backend is currently a 501 shell, so FALLBACK is
what renders — it MUST reproduce the prototype rows exactly. Detail pages may use
static content (no list hook) unless a matching `get*` hook is trivial.

## File shape
```tsx
export default function PageName() {
  // hooks (useUI, useApp, list hook), local drawer/tab state
  return <main className="page"> ...content... </main>;
}
```
Default-export the component. Keep large repeated blocks (drawer forms, table rows)
factored into local helper components within the same file, like `Jobs.tsx` does.
Detail pages: the prototype starts with a `.back-link` — render it as
`<Link className="back-link" to="/...">`.

## Don't
- Don't add Ant Design components or new CSS. Don't invent layout.
- Don't import `app.js` — all its behavior is reimplemented via the shared
  components/hooks above.
- Don't leave `data-*` attributes that drove app.js (replace with React handlers).
  Cosmetic data-* that nothing reads can simply be dropped.
