# OpenAPI specs

Each `*.yaml` here is **generated** from the request/response Go structs of
the corresponding service by `cmd/openapi-gen` in that service's component
directory. Do not hand-edit — your changes will be overwritten by the next
`make doc-gen` run.

| Service | Source | Generator |
|---|---|---|
| `compute.yaml` | `components/compute/internal/{job,service,resourcepool,resourceunit}/service.go` | `components/compute/cmd/openapi-gen` |
| `artifacts.yaml` | `components/artifacts/internal/artifact/{service,render}.go` | `components/artifacts/cmd/openapi-gen` |
| `cluster-manager.yaml` | `components/cluster-manager/internal/server/types.go` | `components/cluster-manager/cmd/openapi-gen` |

The shared reflection engine lives in `pkg/openapigen/`. Each per-service
generator hardcodes its own route table (single source of truth, reviewable
in PRs) and registers component schemas via `pkg/openapigen.Generator`.

## Regenerating

```sh
make doc-gen        # regenerate all three specs
make doc-test       # regenerate + diff (CI guard; non-zero on drift)
make compute-doc-gen
make artifacts-doc-gen
make cluster-manager-doc-gen
```

`make doc-test` runs in CI and fails the build if the committed yaml is out
of sync with the Go types.
