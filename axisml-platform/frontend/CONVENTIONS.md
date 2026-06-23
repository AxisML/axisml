# platform-frontend — page authoring conventions

This SPA is built on **shadcn/ui (Radix primitives) + Tailwind CSS v3 + react-i18next**,
per the engineering design in `docs/system_design/platform/frontend.md`. The product
prototype in `docs/product_design/prototype/` is the authority for page **structure,
fields, and interactions**; the **visual skin** (color / type / radius / spacing / cards)
follows the root `DESIGN.md` (Geist: near-black ink on a near-white canvas, hairline
cards, blue link/focus). Each prototype `*.html` becomes one React page under
`src/pages/`. The shared shell (Sidebar + Topbar) is provided by `AppShell`; pages
render only their content.

## Reference implementations
- `src/pages/jobs.tsx` — canonical LIST page (PageContainer + toolbar Card + `DataTable`
  + `Sheet` create·run·edit form + `confirm()` delete).
- `src/pages/dashboard.tsx` — metric cards / Tabs / lists / Recharts charts.
- Detail pages (`JobDetail`, `ServiceDetail`, …) — `Descriptions`-style key/value grid +
  `Tabs` + timeline/steps, with a back `<Link>` and breadcrumb parent section.

## File & folder naming
- **All files are `kebab-case`** (`page-container.tsx`, `service-detail.tsx`) — matching
  shadcn's own `ui/` files. The React component / hook *exports* stay `PascalCase` /
  `camelCase`; only the filename is kebab.
- Layout is type-based: `src/pages/` (one file per page), `src/components/` (shared
  widgets), `src/components/layout/` (app-shell + sidebar + topbar), `src/components/ui/`
  (shadcn primitives — added via the CLI, never hand-renamed), `src/app/` (router /
  session / store / ui providers), `src/api/`, `src/i18n/`, `src/lib/`, `src/styles/`.

## Building blocks (`@/components`)
- Page chrome: `<PageContainer breadcrumb title subtitle extra>`.
- Numbered form sections: `<FieldSection n title>`. Radio-card pickers: `<CardRadio>`.
- Phase/status: `<PhaseTag phase={...} />` (maps enums → colored `Badge` + i18n).
- Tables: `<DataTable>` wrapper over shadcn `Table` (sorting/pagination/empty/loading).
- Toasts / confirm: `const { toast, confirm } = useUI()` (`@/app/ui`, backed by `sonner`
  + a global `AlertDialog`).

## shadcn usage rules
- **Use shadcn components before custom markup.** Primitives live in `src/components/ui/`
  (added via `pnpm dlx shadcn@latest add`). Compose `Card`/`Sheet`/`Dialog`/`Tabs`/
  `Table`/`Select`/`Badge`/`Alert`/`Empty`/`Skeleton` rather than styled `div`s.
- **`className` is for layout, not color/typography.** Use semantic tokens
  (`bg-background`, `text-muted-foreground`, `border-border`, `text-primary`) — never raw
  hex or `bg-blue-500`. Status hues come from `<PhaseTag>` / `Badge` variants.
- Spacing: `flex`/`grid` + `gap-*`, never `space-y-*`. Equal dims: `size-*`.
- Icons: **lucide-react**. Inside `<Button>` use `data-icon`; no manual size classes.
- Forms: React Hook Form + `zod` via shadcn `Form`/`Field`; validation through
  `aria-invalid` + `FormMessage`. The required marker is a trailing red `*`.
- Overlays (`Dialog`/`Sheet`/`Popover`/`AlertDialog`) manage their own z-index — don't
  add manual `z-*`.

## Data
- List pages call their hook from `@/api/hooks` (`useJobs`, `useServices`, …), scoped to
  the active tenant. Render genuine async states — `DataTable` `loading` + empty/error
  (loadFailed vs noData), `Skeleton`/`Empty` for card grids.
  **Never fabricate fallback / demo rows** — a failed call must surface as an error.
- Writes go through `useApiMutation` (`@/api/mutations`) wrapping the generated SDK
  (`import * as sdk from "@/api/generated"`), with `{ invalidate: [[queryKey]],
  success: t(...) }`. Keep the same query-key prefixes the hooks use.
- Detail pages fetch via the matching `sdk.get*` call in a `useQuery`.

## i18n
- Every user-facing string goes through `t(...)`. Shared keys live in
  `src/i18n/locales/{zh,en}.ts` (`common.*`, `phase.*`, `nav.*`, `role.*`); each feature
  adds `src/i18n/locales/features/<feature>.{zh,en}.ts` exporting
  `default { <feature>: {...} }` — glob-merged, so no shared-file edits. Keep zh and en
  key sets identical. Dates/relative-time via `dayjs`.
- Do not localize free user text (display names, descriptions, log bodies) or raw machine
  enums — only their display labels.

## Don't
- Don't hard-code colors or override component colors/typography via `className`.
- Don't fabricate data to fill a UI a backend endpoint doesn't serve yet — show an honest
  empty/placeholder state (see the metrics/logs panes).
- Don't import from `antd` / `@ant-design/*` — they are removed.
