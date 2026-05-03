# platform e2e

Build tag: `//go:build e2e`. Run via `make e2e-test` (top-level) or
`cd test/e2e && go test -tags=e2e ./platform/...`.

## Prerequisites

The full AxisML stack must already be helm-installed by `make e2e-test`.
Platform sits on top of compute + artifacts, so this subpackage's tests
also depend on those services being up.

## What to test here

When the platform-backend ships, this subpackage holds the user-facing
end-to-end happy-path coverage:

1. **Submit job via platform API** → assert it lands in Compute (Postgres
   row + MLJob CR), the operator reconciles, the Pod runs, status reflows
   all the way back to platform.
2. **Auth flow**: a request without a valid identity is rejected; with one,
   the propagated identity reaches Compute correctly.
3. **Cross-service orchestration**: deploying an MLService that references
   a model in Artifacts goes through Platform → Artifacts (model resolve)
   → Compute (CR creation) → Operator (Deployment + Service).

## How to call the service

`e2e.SetupOrSkip(t)` for the cluster client; `e2e.PortForward()` (future
helper) to reach `<release>-platform` directly, OR test the externally-facing
path through Envoy Gateway by resolving the gateway IP.

## Why no testcontainers

Platform is the integration point for everything; only the real cluster
exercises the full chain.
