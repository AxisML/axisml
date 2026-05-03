# artifacts e2e

Build tag: `//go:build e2e`. Run via `make e2e-test` (top-level) or
`cd test/e2e && go test -tags=e2e ./artifacts/...`.

## Prerequisites

The full AxisML stack must already be helm-installed by `make e2e-test`.
Required infra (axisml-infra chart):

- **zot** (OCI Distribution registry) — backs Models and Images.
- **RustFS** (S3) — backs Datasets.
- **axisml-system Postgres** — metadata for all three artifact types.

## What to test here

When the artifacts service ships, this subpackage holds e2e tests for its
HTTP/gRPC API. Recommended scenarios:

1. **Model lifecycle**: create a model record, push a model file via signed URL
   to zot, fetch by reference, assert metadata + immutable digest.
2. **Image lifecycle**: same, against zot.
3. **Dataset lifecycle**: create dataset, upload via signed URL to RustFS,
   verify metadata + content addressing.
4. **Reference resolution**: resolve an artifact reference and confirm the
   storage URI lines up with the live zot/RustFS instance.

## How to call the service

`e2e.SetupOrSkip(t)` for the cluster client; `e2e.PortForward()` (future
helper) to reach `<release>-artifacts` and the relevant infra Services
(zot, RustFS).

## Why no testcontainers

Artifacts uses zot and RustFS images that the helm chart already wires up.
Reproducing the same setup with testcontainers would duplicate config
without raising fidelity. Sticking to minikube keeps test deps in sync
with production deps.
