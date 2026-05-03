# tenant-operator

Translates the cluster-scoped `Tenant` CRs that AxisML Compute emits into Kubernetes-side state:

- **Namespace** — created and maintained from `spec.namespace.name`. Multiple Tenants may share one Namespace; deleting a Tenant does **not** cascade-delete its Namespace.
- **ElasticQuota** — `spec.quotas[]` is rendered 1:1 into Koordinator scheduler-plugins `ElasticQuota` CRs; observed `status.used` flows back into `Tenant.status.quotas[].used`.
- **Per-tenant init resources** — `ImagePullSecret` / `Secret` / `ConfigMap` / `ServiceAccount` (+ optional RBAC) under `spec.initResources`, copied from a controlled source namespace (default `axisml-system`).

Full design and the write-path contract with Compute live in [`docs/system_design/operators/tenant-operator.md`](../../../docs/system_design/operators/tenant-operator.md) (Chinese).

## Layout

```
api/v1alpha1/        Tenant CRD Go types + deepcopy
cmd/main.go          Manager entrypoint (scheme registration, cache filters, leader election)
internal/config/     Runtime config loaded from environment variables
internal/controller/ TenantReconciler — watches Tenant + self-managed resources only
internal/reconcile/  Idempotent reconcile for each child resource kind
internal/validate/   Spec validation and namespace denylist
examples/            tenant-sample.yaml for local e2e
test/                Unit tests + envtest integration tests (build tag `integration`)
```

The CRD generated from `api/v1alpha1` is checked in at [`deploy/helm/axisml-system/crds/tenant-crd.yaml`](../../../deploy/helm/axisml-system/crds/tenant-crd.yaml) and installed by the Helm chart.

## Runtime configuration

Environment variables (wired via Helm values under `tenantOperator.*`):

| Variable | Default | Purpose |
| --- | --- | --- |
| `RESYNC_PERIOD` | `10m` | controller-runtime cache full resync period |
| `NAMESPACE_DENYLIST` | built-in (`kube-system`, `axisml-system`, ...) | comma-separated; rejects matching `spec.namespace.name` |

Command-line flags (see `cmd/main.go`):

| Flag | Default |
| --- | --- |
| `--metrics-bind-address` | `:8080` |
| `--health-probe-bind-address` | `:8081` |
| `--leader-elect` | `false` |

To avoid pulling every `Secret` / `ConfigMap` in the cluster into memory, the cache filters self-managed resources by the `axisml.io/managed-by=tenant-operator` label. Source `Secret` / `ConfigMap` reads bypass the cache and go through the `APIReader` — no cluster-wide watch is established for those.

## Local development

```sh
make help                 # list all targets
make fmt vet              # format + static checks
make build                # compile bin/tenant-operator
make test                 # unit tests (no cluster required)
make test-integration     # envtest integration tests against the kubeconfig
                          # context (see prerequisites below)
make image                # docker build → axisml/tenant-operator:0.1.0
make image-load-minikube  # build and `minikube image load` the image
```

`test-integration` runs against an existing Kubernetes cluster — typically
the local minikube cluster from `make cluster-up`. Prerequisites:

- Cluster reachable via the current kubeconfig context (`kubectl config current-context`).
- Koordinator's `ElasticQuota` CRD installed (provided by `make helm-install-infra`). The chart's `Tenant` CRD is applied by the test from `deploy/helm/axisml-system/crds/`.
- The in-cluster `tenant-operator` Deployment must be scaled to 0 so it doesn't race with the in-test reconciler:

  ```sh
  kubectl scale -n axisml-system deploy/axisml-tenant-operator --replicas=0
  ```

`IMAGE_TAG` must track the `appVersion` in [`deploy/helm/axisml-system/Chart.yaml`](../../../deploy/helm/axisml-system/Chart.yaml) — the chart derives the default image tag from `appVersion`, so locally built images need a matching tag for the rendered Deployment to pull successfully.

## Deployment

The operator ships as part of the `axisml-system` chart:

```sh
helm install axisml deploy/helm/axisml-system
```

Related templates:

- CRD: `deploy/helm/axisml-system/crds/tenant-crd.yaml`
- Deployment / RBAC / ServiceAccount: `deploy/helm/axisml-system/templates/operators/tenant-operator/`
- Values: the `tenantOperator` section in `deploy/helm/axisml-system/values.yaml`

## Trying it out

```sh
kubectl apply -f examples/tenant-sample.yaml
kubectl get tenant sample -o yaml
kubectl get ns,elasticquota,secret,sa,role,rolebinding \
  -A -l axisml.io/tenant-id=11111111-2222-3333-4444-555555555555
```

The header of `examples/tenant-sample.yaml` lists the prerequisites (a `registry-source` Secret and an `envs-source` ConfigMap in the `axisml-system` namespace).
