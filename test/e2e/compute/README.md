# compute e2e

Build tag: `//go:build e2e`. Run via `make e2e-test` (top-level) or
`cd test/e2e && go test -tags=e2e ./compute/...`.

## Prerequisites

The full AxisML stack must already be helm-installed by `make e2e-test`'s
upstream targets (`cluster-up && image-load && helm-install && e2e-wait`).
That brings up:

- the three operators (tenant / mljob / mlservice)
- axisml-system Postgres (sub-chart of axisml-system)
- the compute Service (`axisml-compute`)

## What to test here

When the compute service ships, this subpackage holds e2e tests for its
HTTP/gRPC API. Recommended scenarios:

1. **Tenant CRUD via compute API** → assert that compute persists business
   metadata in Postgres AND that tenant-operator reconciles the resulting
   `Tenant` CR (tests Compute → Operator → K8s flow end-to-end).
2. **Job submit via compute** → POST /jobs creates an MLJob CR; mljob-operator
   schedules it; status reflows back into Compute via informer.
3. **Quota usage reflow** → verify compute reads ElasticQuota.status.used
   and flows it back into the Quota model.

## How to call the service

Use `e2e.SetupOrSkip(t)` for the cluster client, then `e2e.PortForward()`
(future helper, add when first compute test lands) to reach the in-cluster
`<release>-compute` Service. Direct typed-client API calls go through
the package-public client to be added under `components/compute/internal/client/`.

## Why no testcontainers

Compute's core value is its interaction with real K8s + Koordinator's
scheduler-plugins ElasticQuota + helm-installed Postgres. testcontainers
would let us spin up Postgres locally but would miss the K8s-side flows,
so we keep everything on minikube to avoid drift between test-deps and
production-deps.
