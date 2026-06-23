# Platform-layer OpenAPI spec

`platform.yaml` is **generated** from the request/response DTOs in
`axisml-platform/backend/internal/server` by `axisml-platform/backend/cmd/openapi-gen`.
Do not hand-edit — your changes are overwritten by the next `make doc-gen` run.

| Spec | Source | Generator |
|---|---|---|
| `platform.yaml` | `axisml-platform/backend/internal/server/*.go` | `axisml-platform/backend/cmd/openapi-gen` |

`platform.yaml` is a special case: Platform's server is not fully implemented
yet, so most handlers don't exist — but the component declares its complete HTTP
surface as DTOs in `internal/server`. The eventual handlers reuse those types,
so the spec stays in lock-step once they land. The frontend's typed API client
(`@hey-api`) and the backend's own contract are both driven from this spec.

## Regenerating

```sh
make backend-doc-gen     # regenerate platform.yaml from internal/server DTOs
make doc-test            # regenerate + diff (CI guard; non-zero on drift)
```
