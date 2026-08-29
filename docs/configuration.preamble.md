<!--
  This file is the hand-maintained preamble of the configuration reference manual.
  The per-service tables in docs/configuration.md (below the marker) are GENERATED
  from each service's Config struct — run `make config-docs-gen` after changing a
  Config struct or this preamble. Do not hand-edit docs/configuration.md.
-->
# Configuration

Kubernetes services use `/etc/axisml/config.yaml` plus `AXISML_` environment
overrides. The standalone distribution is environment-only. Secrets are
always supplied out of band.
The per-service reference tables at the bottom of this document are generated from the Go `Config`
structs, so they always match the running code.

## Scope

| Binary | Module |
|---|---|
| `compute-service` | `axisml-system/compute-service` |
| `artifact-hub` | `axisml-system/artifact-hub` |
| `axisml-standalone` | `axisml-standalone` |
| `platform-backend` | `axisml-platform/backend` |

Out of scope: the controller-runtime components (`tenant-operator`, `compute-operator`,
`cluster-manager`), which keep their CLI flags.

## Sources and precedence

Each key resolves from these layers, lowest priority first:

1. **Built-in defaults** — the `default` tag on each Config field.
2. **Config file** — `/etc/axisml/config.yaml`.
3. **Environment override** — any `AXISML_`-prefixed variable.
4. **Secret file** — for secret keys, `AXISML_<KEY>_FILE`. Highest priority.

A later layer overrides an earlier one per key. Standalone omits layer 2 and
does not accept `--config`. A missing Kubernetes config file is not an error;
production fails fast on validation instead.

## File discovery

First match wins: `--config <path>` → `AXISML_CONFIG` → `/etc/axisml/config.yaml` → `./config.yaml`.

## The `AXISML_` rule

Every key is overridable by an environment variable, mechanically:

```
<section>.<key>   ⇄   AXISML_<SECTION>_<KEY>     (uppercased; dots → underscores)
```

Leaf keys are `snake_case`; the config tree is exactly two levels, so there is one env name per
setting. Every variable that configures an AxisML binary's own behaviour is named `AXISML_<...>`.

## Secrets

Secrets are never written into the config file. Each can be supplied either way — both first-class:

- **`AXISML_<KEY>`** — the value directly in the environment (e.g. a Kubernetes `secretKeyRef`).
- **`AXISML_<KEY>_FILE`** — a path to a mounted file whose trimmed contents become the value (e.g. a
  projected Secret volume); keeps plaintext out of the environment and supports rotation.

Set whichever fits; if both are set for a key, `_FILE` wins. Secret values are redacted from logs.

## Reserved / third-party variables

These are read by third-party SDKs or sibling images at their own names and are exempt from the
`AXISML_` rule: `KUBECONFIG`, the downward-API `POD_*`, the Postgres
image's `POSTGRES_*`, and Go/OS runtime knobs (`GOMAXPROCS`, `TZ`, …).

## Fixed by design (not configurable)

These are code constants or derived values rather than configuration:
listen ports (`:8080` API, `:8081` health, `:9090` metrics), HTTP/DB-pool tuning, reconcile cadence,
GC and session TTLs, leader election (unconditionally on — a no-op at one replica), the OCI scheme
(derived from the endpoint URL), and the JWT `kid` (RFC 7638 thumbprint of the signing key). The
default/public tenant is owned by the System Helm chart's `seed.tenant` and discovered by
`platform-backend` via cluster-manager.

---

<!-- BEGIN GENERATED — do not edit below; run `make config-docs-gen` -->
