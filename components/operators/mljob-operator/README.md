# mljob-operator

Kubernetes operator that manages the lifecycle of `MLJob` custom resources — the AxisML abstraction for one-shot compute workloads (training, distributed training, batch / data jobs).

The operator watches `MLJob` and dispatches reconciliation to a backend handler selected by `spec.backend.{name, engine}`:

| Backend | Engine | Underlying resource |
| --- | --- | --- |
| `native` | `job` (default) | Kubernetes `batch/v1` `Job` |
| `native` | `podgroup` | `Job` + sigs.k8s.io scheduler-plugins `PodGroup` (gang scheduling for single-role distributed training) |
| `kubeflow-trainer` | `pytorchjob` / `tfjob` / `mpijob` / ... | Kubeflow Training Operator CRs (multi-role) |
| `custom` | any | User-supplied target GVK via `backend.config` |

All Pods produced by any backend must run under `schedulerName: koord-scheduler` and carry the tenant's `quota.scheduling.koordinator.sh/name` label so they consume the corresponding Koordinator `ElasticQuota`.

Full design and the write-path contract with AxisML Compute live in [`docs/system_design/operators/mljob-operator.md`](../../../docs/system_design/operators/mljob-operator.md) (Chinese).

## Layout

```
api/v1alpha1/        MLJob CRD Go types + deepcopy
cmd/manager/         Manager entrypoint (scheme registration, controller wiring)
internal/dispatcher/ Backend selection by spec.backend.{name, engine}
internal/handler/    Handler interface + shared reconcile primitives
internal/handlers/   Per-backend handlers
  ├── nativejob/        (native, job)
  └── nativepodgroup/   (native, podgroup)
internal/labels/     Common label / annotation helpers
hack/                controller-gen boilerplate
```

The CRD generated from `api/v1alpha1` is checked in under `deploy/helm/axisml-system/crds/` and installed by the Helm chart.

## Local development

```sh
make help                 # list all targets
make / make build         # compile bin/mljob-operator
make test                 # unit tests (no cluster required)
make image                # docker build → axisml/mljob-operator:0.1.0
make image-load-minikube  # build and `minikube image load` the image
make clean                # remove build artifacts

# Kubebuilder workflow
make generate             # regenerate deepcopy code via controller-gen
make tidy                 # go mod tidy
make fmt vet              # format + static checks
```

`IMAGE_TAG` defaults to `0.1.0` and must track the `appVersion` in [`deploy/helm/axisml-system/Chart.yaml`](../../../deploy/helm/axisml-system/Chart.yaml) — the chart derives the default image tag from `appVersion`, so locally built images need a matching tag for the rendered Deployment to pull successfully.

## Deployment

The operator ships as part of the `axisml-system` chart:

```sh
helm install axisml deploy/helm/axisml-system
```

Related templates:

- CRD: `deploy/helm/axisml-system/crds/mljob-crd.yaml`
- Deployment / RBAC / ServiceAccount: `deploy/helm/axisml-system/templates/operators/mljob-operator/`
- Values: the `mljobOperator` section in `deploy/helm/axisml-system/values.yaml`
