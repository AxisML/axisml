# In-browser mock API

When the backend has no data (or isn't running), set `VITE_USE_MOCK_API=true` and
the frontend serves **every** API call from these fixtures — the network is never
touched.

```sh
VITE_USE_MOCK_API=true pnpm dev          # dev server
VITE_USE_MOCK_API=true pnpm build        # baked into a static bundle
```

Log in with **any** username / password — the mock accepts everything and returns a
system-admin session, so the full navigation (including 租户管理 / 资源池管理) is
visible.

## How it works

The mock is a custom `fetch` injected into the generated `@hey-api/client-fetch`
client — it is *not* a parallel client, so all 100+ SDK functions, the
request/response interceptors, URL building and the `{ data, error }` unwrap keep
working unchanged.

```
api/client.ts ── createClientConfig() sets { fetch: mockFetch } when USE_MOCK
   └── mock/index.ts        mockFetch(Request) → Response   (+ simulated latency)
        └── mock/router.ts        method + path → handler   (":param" path matching)
             └── mock/data.ts          fixtures (typed as the generated API types)
                  └── mock/examples.gen.ts   whole-object examples lifted from the spec
```

- `examples.gen.ts` — **generated, do not edit.** `pnpm run gen:mock` (also run by
  `gen:api`, and `make -C axisml-platform frontend-gen-mock`) reads the
  `components.schemas.*.example` blocks out of `axisml-platform/docs/apis/platform.yaml`
  and writes them here, keyed by schema name. Those examples are authored on the Go
  DTOs (`axisml-platform/backend/cmd/openapi-gen/examples_*.go`), so the fixtures
  can never drift from the API contract.
- `data.ts` — pulls each entity via the typed `ex<T>("SchemaName")` accessor and
  clones it into a few rows so list pages show variety. A few helpers (pod logs,
  metric series, the cluster-usage dashboard) have no endpoint in the contract and
  stay synthesized here, marked demo-only.
- `router.ts` — one line per endpoint. Unknown GETs fall back to an empty list and
  unknown writes to `{}`, so a missing route degrades gracefully instead of hanging.
- `index.ts` — the `fetch` shim + the `VITE_USE_MOCK_API` flag.

## Adding / changing data

To change an entity's **canonical** shape or values, edit its example on the Go DTO
(`examples_*.go`), run `make -C axisml-platform doc-gen` then `pnpm run gen:mock`.
For list variety or demo-only helpers, edit `data.ts` directly (no codegen needed —
Vite HMR picks it up). Add endpoints with an `on(method, path, handler)` line in
`router.ts`.
</content>
