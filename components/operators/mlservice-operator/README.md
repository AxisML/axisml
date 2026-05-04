# mlservice-operator

Kubernetes operator that manages the lifecycle of `MLService` custom resources — the AxisML abstraction for long-running online inference services.

The operator watches `MLService` and dispatches reconciliation to a backend handler selected by `spec.backend.{name, engine}`:

| Backend | Engine | Underlying resource |
| --- | --- | --- |
| `native` | `deployment` (default) | Kubernetes `apps/v1` `Deployment` (+ `Service` / route as needed) |
| `native` | `statefulset` | `apps/v1` `StatefulSet` |
| `kserve` | `inference` | KServe `InferenceService` |
| `kserve` | `llminference` | KServe LLM-native serving (placeholder; GVK & fields tracked against the upstream LLM API) |
| `custom` | any | User-supplied target GVK via `backend.config` |

All Pods produced by any backend must run under `schedulerName: koord-scheduler` and carry the tenant's `quota.scheduling.koordinator.sh/name` label so they consume the corresponding Koordinator `ElasticQuota`. For KServe-backed services this is wired through `spec.predictor.schedulerName` and `spec.predictor.labels`.

Full design and the write-path contract with AxisML Compute live in [`docs/system_design/operators/mlservice-operator.md`](../../../docs/system_design/operators/mlservice-operator.md) (Chinese).

## Layout

```
api/v1alpha1/             MLService CRD Go types + deepcopy
cmd/manager/              Manager entrypoint (scheme registration, controller wiring)
internal/dispatcher/      MLService reconciler + immutability checks + backend list
internal/handler/         Handler interface, registry, and per-backend implementations
  └── nativedeployment/      (native, deployment)
```

The CRD generated from `api/v1alpha1` is checked in under `deploy/helm/axisml-system/crds/` and installed by the Helm chart.

## Local development

```sh
make help                 # list all targets
make / make build         # compile bin/mlservice-operator
make test                 # unit tests (no cluster required)
make image                # docker build -> ghcr.io/axisml/axisml/axisml-operator:0.1.0
make image-load-minikube  # build and `minikube image load` the image
make clean                # remove build artifacts

make fmt vet              # format + static checks
make tidy                 # go mod tidy
```

`IMAGE_TAG` defaults to `0.1.0` and must track the `appVersion` in [`deploy/helm/axisml-system/Chart.yaml`](../../../deploy/helm/axisml-system/Chart.yaml) — the chart derives the default image tag from `appVersion`, so locally built images need a matching tag for the rendered Deployment to pull successfully.

## Deployment

The operator ships as part of the `axisml-system` chart:

```sh
helm install axisml deploy/helm/axisml-system
```

Related templates:

- CRD: `deploy/helm/axisml-system/crds/mlservice-crd.yaml`
- Deployment / RBAC / ServiceAccount: `deploy/helm/axisml-system/templates/operators/mlservice-operator/`
- Values: the `mlserviceOperator` section in `deploy/helm/axisml-system/values.yaml`
