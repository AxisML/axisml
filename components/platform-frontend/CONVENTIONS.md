# platform-frontend — page authoring conventions

This SPA is built on **Ant Design v6 + Tailwind CSS v3 + react-i18next**, per the
engineering design in `docs/system_design/platform/frontend.md`. The product
prototype in `docs/product_design/prototype/` is the reference for page structure,
fields, and copy — but pages are **adapted to Ant Design**, not ported verbatim.
Each prototype `*.html` becomes one React page under `src/pages/`. The shared
shell (Sider + Header) is provided by `AppShell`; pages render only their content.

## Reference implementations
- `src/pages/Jobs.tsx` — canonical LIST page (PageContainer + Card toolbar + AntD
  `Table` + `Drawer`/`Form` create·run·edit + `Modal.confirm` delete).
- `src/pages/Dashboard.tsx` — cards / `Statistic` / `Tabs` / `List` / charts.
- Detail pages (`JobDetail`, `ServiceDetail`, …) — `Descriptions` + `Tabs` +
  `Timeline`/`Steps`, with a back `<Link>` and breadcrumb parent section.

## Building blocks
- Page chrome: `<PageContainer breadcrumb title subtitle extra>` (`@/components`).
- Numbered form sections: `<FieldSection n title>`. Radio-card pickers: `<CardRadio>`.
- Phase/status: `<PhaseTag phase={...} />` (maps enums → colored AntD `Tag` + i18n).
- Toasts / confirm modals: `const { toast, confirm } = useUI()` (`@/app/ui`, backed
  by AntD `App` message + Modal).

## Styling
- **UI = Ant Design components.** Reach for third-party libs from the AntD
  recommendation list (https://ant.design/docs/react/recommendation) only when AntD
  has no equivalent; hand-roll as a last resort. Charts use `@ant-design/charts`.
- **Layout / spacing = Tailwind utilities.** Use the token color classes
  (`text-fg`, `text-fg-2`, `text-muted`, `text-accent`, `bg-bg`, `bg-surface`,
  `bg-surface-warm`, `border-border-soft`, `font-mono`) which map to the design
  tokens in `src/styles/tokens.css` — never hard-code hex. The red brand accent and
  light/dark algorithms are injected by `ConfigProvider` in `src/app/theme.tsx`.
- AntD + Tailwind coexist via `@layer` ordering (`src/styles/tailwind.css`) +
  `<StyleProvider layer>`. Tailwind utilities win over AntD; for the rare forced
  override use the `!` prefix (e.g. `!bg-accent`).
- Icons: `@ant-design/icons`.

## Data
- List pages call their hook from `@/api/hooks` (`useJobs`, `useServices`, …),
  scoped to the active tenant. Render genuine async states — `Table loading` +
  `locale.emptyText` (loadFailed vs noData), `<Spin>` / `<Empty>` for card grids.
  **Never fabricate fallback / demo rows** — a failed call must surface as an error.
- Writes go through `useApiMutation` (`@/api/mutations`) wrapping the generated SDK
  (`import * as sdk from "@/api/generated"`), with `{ invalidate: [[queryKey]],
  success: t(...) }`. Keep the same query-key prefixes the hooks use.
- Detail pages fetch via the matching `sdk.get*` call in a `useQuery`.

## i18n
- Every user-facing string goes through `t(...)`. Shared keys live in
  `src/i18n/locales/{zh,en}.ts` (`common.*`, `phase.*`, `nav.*`, `role.*`); each
  feature adds `src/i18n/locales/features/<feature>.{zh,en}.ts` exporting
  `default { <feature>: {...} }` — these are glob-merged, so no shared-file edits.
  Keep zh and en key sets identical. Dates/relative-time via `dayjs`.
- Do not localize free user text (display names, descriptions, log bodies) or raw
  machine enums — only their display labels.

## Don't
- Don't hard-code colors or use removed legacy CSS classes (`panel`, `tbl`, `btn`,
  `page`, `kpi`, …) — `src/styles/app.css` and the old hand-rolled components are gone.
- Don't fabricate data to fill a UI a backend endpoint doesn't serve yet — show an
  honest empty/placeholder state (see the metrics/logs panes).
