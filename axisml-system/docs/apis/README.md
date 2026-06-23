# System-layer OpenAPI specs

Each `*.yaml` here is **generated** from the request/response Go structs of the
corresponding System service by `cmd/openapi-gen` in that component. Do not
hand-edit — your changes are overwritten by the next `make doc-gen` run.

| Spec | Source | Generator |
|---|---|---|
| `cluster-manager.yaml` | `axisml-system/cluster-manager/internal/server/types.go` | `axisml-system/cluster-manager/cmd/openapi-gen` |
| `compute-service.yaml` | `axisml-system/compute-service/internal/{job,service,resourcepool,resourceunit}/service.go` | `axisml-system/compute-service/cmd/openapi-gen` |
| `artifact-hub.yaml` | `axisml-system/artifact-hub/internal/artifact/{service,render}.go` | `axisml-system/artifact-hub/cmd/openapi-gen` |

The shared reflection engine lives in `pkg/openapigen/`. Each per-service
generator hardcodes its own route table (single source of truth, reviewable in
PRs) and registers component schemas via `pkg/openapigen.Generator`. The
Platform backend consumes these three specs to generate its typed downstream
clients (`make -C axisml-platform/backend client-gen`).

## Regenerating

```sh
make doc-gen                      # regenerate every layer's specs
make doc-test                     # regenerate + diff (CI guard; non-zero on drift)
make cluster-manager-doc-gen
make compute-service-doc-gen
make artifact-hub-doc-gen
```

`make doc-test` runs in CI and fails the build if the committed yaml is out of
sync with the Go types.
