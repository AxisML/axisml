# External CRDs vendored for envtest

These YAMLs are loaded into the embedded API server by every operator's
L1 envtest. envtest only knows about CRDs that are explicitly fed to it,
so any CRD an operator imports from outside this repo (Koordinator,
scheduler-plugins, Kubeflow, KServe, …) needs to live here.

## Sources

| File | Group / Kind | Upstream | Pinned version |
|------|--------------|----------|----------------|
| `koordinator-elasticquota.yaml` | `scheduling.sigs.k8s.io` / `ElasticQuota` | Koordinator's vendored copy of scheduler-plugins under `apis/thirdparty/scheduler-plugins`. NOTE the group is `scheduling.sigs.k8s.io`, not the upstream `scheduling.x-k8s.io` — Koordinator forked it. | matches `github.com/koordinator-sh/koordinator v1.8.0` (pinned in `tenant-operator/go.mod`) |
| `scheduler-plugins-podgroup.yaml` | `scheduling.x-k8s.io` / `PodGroup` | `kubernetes-sigs/scheduler-plugins` `config/crd/bases/scheduling.x-k8s.io_podgroups.yaml` | matches `sigs.k8s.io/scheduler-plugins v0.34.7` (pinned in `mljob-operator/go.mod`) |
| `gateway-api-httproute.yaml` | `gateway.networking.k8s.io` / `HTTPRoute` | `kubernetes-sigs/gateway-api` `config/crd/standard/gateway.networking.k8s.io_httproutes.yaml`. Required even for tests that don't enable spec.route, because the mlservice-operator dispatcher watches HTTPRoute. | matches `sigs.k8s.io/gateway-api v1.5.1` (pinned in `mlservice-operator/go.mod`) |

Every `.yaml` in this directory is fed to envtest's `CRDDirectoryPaths` by all three operators' TestMains. Don't add empty/placeholder files here — envtest tolerates them today but the contract isn't load-bearing, and a malformed placeholder would break every L1 envtest.

## Planned

These will be vendored when the corresponding handler lands; until then keep the file out of this directory (envtest loads everything here).

- **Kubeflow Training Operator** (`kubeflow.org` family — likely `TrainJob` `v2alpha1`, or `PyTorchJob`/`TFJob`/`MPIJob` `v1`). Source: `github.com/kubeflow/trainer/manifests/...`. Add as `kubeflow-trainjob.yaml` when a kubeflow handler lands in `mljob-operator`.
- **KServe** (`serving.kserve.io` — likely `InferenceService` `v1beta1`, optionally `LLMInferenceService` `v1alpha1`). Source: `github.com/kserve/kserve/config/crd/...`. Add as `kserve-inferenceservice.yaml` when a kserve handler lands in `mlservice-operator`.

## Refresh procedure

When the Go module version of an upstream changes:

1. Bump the version in the relevant operator's `go.mod`.
2. Re-vendor the CRD YAML from the matching tag, e.g.
   ```sh
   cp ~/go/pkg/mod/sigs.k8s.io/scheduler-plugins@<version>/config/crd/bases/scheduling.x-k8s.io_podgroups.yaml \
      test/crds/external/scheduler-plugins-podgroup.yaml
   ```
3. Update the version cell in the table above.
4. Run `make envtest-test` to confirm the new schema still satisfies the operator.

A `make sync-external-crds` helper that automates this is a planned follow-up.
