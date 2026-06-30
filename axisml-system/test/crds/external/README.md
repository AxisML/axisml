# External CRDs vendored for L1 integration tests

These YAMLs are loaded into the embedded API server by the merged
compute-operator's L1 integration suite. controller-runtime's envtest
framework only knows about CRDs that are explicitly fed to it, so any CRD
the operator imports from outside this repo (scheduler-plugins, Kubeflow,
KServe, …) needs to live here.

## Sources

The `ElasticQuota` / `PodGroup` schemas are upstream `kubernetes-sigs/scheduler-plugins`
(group `scheduling.x-k8s.io`) — the same vocabulary the AxisML self-built scheduler
(`axisml-scheduler`) reads. The operators carry an in-repo, wire-compatible Go copy of
the ElasticQuota type under `axisml-system/tenant-operator/api/scheduling/v1alpha1`
rather than depending on a scheduler-plugins release (which lags the k8s line these
modules target), so these YAMLs are the source of truth for the CRD shape in tests.

| File | Group / Kind | Upstream | Pinned version |
|------|--------------|----------|----------------|
| `scheduler-plugins-elasticquota.yaml` | `scheduling.x-k8s.io` / `ElasticQuota` | `kubernetes-sigs/scheduler-plugins` `config/crd/bases/scheduling.x-k8s.io_elasticquotas.yaml`. Mirrors the in-repo Go type. | tracks `kubernetes-sigs/scheduler-plugins` (the release `axisml-scheduler` pins) |
| `scheduler-plugins-podgroup.yaml` | `scheduling.x-k8s.io` / `PodGroup` | `kubernetes-sigs/scheduler-plugins` `config/crd/bases/scheduling.x-k8s.io_podgroups.yaml`. Watched by the gang-scheduling path. | tracks `kubernetes-sigs/scheduler-plugins` (the release `axisml-scheduler` pins) |
| `gateway-api-httproute.yaml` | `gateway.networking.k8s.io` / `HTTPRoute` | `kubernetes-sigs/gateway-api` `config/crd/standard/gateway.networking.k8s.io_httproutes.yaml`. Required even for tests that don't enable spec.route, because the MLService dispatcher watches HTTPRoute. | matches `sigs.k8s.io/gateway-api v1.5.1` (pinned in `axisml-system/compute-operator/go.mod`) |

Every `.yaml` in this directory is fed to envtest's `CRDDirectoryPaths` by the merged operator's TestMain. Don't add empty/placeholder files here — envtest tolerates them today but the contract isn't load-bearing, and a malformed placeholder would break every L1 integration test.

## Planned

These will be vendored when the corresponding handler lands; until then keep the file out of this directory (envtest loads everything here).

- **Kubeflow Training Operator** (`kubeflow.org` family — likely `TrainJob` `v2alpha1`, or `PyTorchJob`/`TFJob`/`MPIJob` `v1`). Source: `github.com/kubeflow/trainer/manifests/...`. Add as `kubeflow-trainjob.yaml` when a kubeflow MLRun handler lands.
- **KServe** (`serving.kserve.io` — likely `InferenceService` `v1beta1`, optionally `LLMInferenceService` `v1alpha1`). Source: `github.com/kserve/kserve/config/crd/...`. Add as `kserve-inferenceservice.yaml` when a kserve MLService handler lands.

## Refresh procedure

When the scheduler-plugins release that `axisml-scheduler` pins changes:

1. Bump the pinned version in `axisml-infra/axisml-scheduler/go.mod`.
2. Re-vendor the CRD YAMLs from the matching tag (group `scheduling.x-k8s.io`):
   ```sh
   cp ~/go/pkg/mod/sigs.k8s.io/scheduler-plugins@<version>/config/crd/bases/scheduling.x-k8s.io_elasticquotas.yaml \
      axisml-system/test/crds/external/scheduler-plugins-elasticquota.yaml
   cp ~/go/pkg/mod/sigs.k8s.io/scheduler-plugins@<version>/config/crd/bases/scheduling.x-k8s.io_podgroups.yaml \
      axisml-system/test/crds/external/scheduler-plugins-podgroup.yaml
   ```
3. Keep the in-repo Go copy under `axisml-system/tenant-operator/api/scheduling/v1alpha1`
   in sync with the ElasticQuota schema.
4. Run `make integration-test` to confirm the new schema still satisfies the operator.

A `make sync-external-crds` helper that automates this is a planned follow-up.
