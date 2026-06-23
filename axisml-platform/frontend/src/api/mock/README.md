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
   └── mock/index.ts   mockFetch(Request) → Response   (+ simulated latency)
        └── mock/router.ts   method + path → handler   (":param" path matching)
             └── mock/data.ts   fixtures (typed as the generated API types)
```

- `data.ts` — the fixtures. Shapes are the generated types, so anything that
  compiles renders exactly as it would against a live backend. Values are grounded
  in `docs/product_design/prototype`.
- `router.ts` — one line per endpoint. Unknown GETs fall back to an empty list and
  unknown writes to `{}`, so a missing route degrades gracefully instead of hanging.
- `index.ts` — the `fetch` shim + the `VITE_USE_MOCK_API` flag.

## Adding / changing data

Edit `data.ts` (add an item to an array) or `router.ts` (add an `on(method, path,
handler)` line). No backend, codegen or restart-of-anything required beyond the
normal Vite HMR.
</content>
