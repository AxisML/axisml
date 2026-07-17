<!--
  This file is the hand-maintained preamble of the configuration reference manual.
  The per-service tables in docs/configuration.md (below the marker) are GENERATED
  from each service's Config struct — run `make config-docs-gen` after changing a
  Config struct or this preamble. Do not hand-edit docs/configuration.md.
-->
# Configuration

AxisML's deployable services are configured by a YAML file at `/etc/axisml/config.yaml` plus
environment overrides under the mandatory `AXISML_` prefix; secrets are always supplied out of band.
The per-service reference tables at the bottom of this document are generated from the Go `Config`
structs, so they always match the running code.

## Scope

| Binary | Module |
|---|---|
| `compute-service` | `axisml-system/compute-service` |
| `artifact-hub` | `axisml-system/artifact-hub` |
| `platform-backend` | `axisml-platform/backend` |

Out of scope: the controller-runtime components (`tenant-operator`, `compute-operator`,
`cluster-manager`), which keep their CLI flags.

## Sources and precedence

Each key resolves from these layers, lowest priority first:

1. **Built-in defaults** — the `default` tag on each Config field.
2. **Config file** — `/etc/axisml/config.yaml`.
3. **Environment override** — any `AXISML_`-prefixed variable.
4. **Secret file** — for secret keys, `AXISML_<KEY>_FILE`. Highest priority.

A later layer overrides an earlier one per key. A missing config file is not an error; production fails
fast on validation instead. The only CLI flag is `--config`.

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

These do not differ across deployments, so they are code constants or derived values, not config:
listen ports (`:8080` API, `:8081` health, `:9090` metrics), HTTP/DB-pool tuning, reconcile cadence,
GC and session TTLs, leader election (unconditionally on — a no-op at one replica), the OCI scheme
(derived from the endpoint URL), and the JWT `kid` (RFC 7638 thumbprint of the signing key). The
default/public tenant is owned by the System Helm chart's `seed.tenant` and discovered by
`platform-backend` via cluster-manager.

---

<!-- BEGIN GENERATED — do not edit below; run `make config-docs-gen` -->

# Per-service reference

## compute-service

Config file: `/etc/axisml/config.yaml` (override with `--config` or `AXISML_CONFIG`). Every key is also overridable by its `AXISML_` variable.

| Key | Environment variable | Default | Secret | Description |
|---|---|---|---|---|
| `database.host` | `AXISML_DATABASE_HOST` | `localhost` | — | PostgreSQL host |
| `database.port` | `AXISML_DATABASE_PORT` | `5432` | — | PostgreSQL port |
| `database.name` | `AXISML_DATABASE_NAME` | `axisml` | — | Database name |
| `database.user` | `AXISML_DATABASE_USER` | `axisml` | — | Database user |
| `database.password` | `AXISML_DATABASE_PASSWORD`<br>`AXISML_DATABASE_PASSWORD_FILE` | — | yes | Database password |
| `database.sslmode` | `AXISML_DATABASE_SSLMODE` | `disable` | — | libpq sslmode: disable \| require \| verify-full |
| `log.level` | `AXISML_LOG_LEVEL` | `info` | — | Log level: debug \| info \| warn \| error |
| `log.format` | `AXISML_LOG_FORMAT` | `json` | — | Log format: json \| console |
| `prometheus.url` | `AXISML_PROMETHEUS_URL` | — | — | Prometheus query API base URL (e.g. http://kube-prometheus-stack-prometheus.axisml-infra:9090). Empty disables the workload metrics endpoints. |
| `workload.tenant_prefix` | `AXISML_WORKLOAD_TENANT_PREFIX` | `false` | — | Prefix physical workload names with a readable, collision-resistant tenant token |

## artifact-hub

Config file: `/etc/axisml/config.yaml` (override with `--config` or `AXISML_CONFIG`). Every key is also overridable by its `AXISML_` variable.

| Key | Environment variable | Default | Secret | Description |
|---|---|---|---|---|
| `database.host` | `AXISML_DATABASE_HOST` | `localhost` | — | PostgreSQL host |
| `database.port` | `AXISML_DATABASE_PORT` | `5432` | — | PostgreSQL port |
| `database.name` | `AXISML_DATABASE_NAME` | `axisml` | — | Database name |
| `database.user` | `AXISML_DATABASE_USER` | `axisml` | — | Database user |
| `database.password` | `AXISML_DATABASE_PASSWORD`<br>`AXISML_DATABASE_PASSWORD_FILE` | — | yes | Database password |
| `database.sslmode` | `AXISML_DATABASE_SSLMODE` | `disable` | — | libpq sslmode: disable \| require \| verify-full |
| `log.level` | `AXISML_LOG_LEVEL` | `info` | — | Log level: debug \| info \| warn \| error |
| `log.format` | `AXISML_LOG_FORMAT` | `json` | — | Log format: json \| console |
| `oci.endpoint` | `AXISML_OCI_ENDPOINT` | `http://axisml-infra-zot.axisml-infra:5000` | — | OCI registry endpoint (full URL; scheme derived from it) |
| `oci.admin_user` | `AXISML_OCI_ADMIN_USER` | `admin` | — | OCI registry admin username |
| `oci.admin_password` | `AXISML_OCI_ADMIN_PASSWORD`<br>`AXISML_OCI_ADMIN_PASSWORD_FILE` | — | yes | OCI registry admin password |
| `s3.endpoint` | `AXISML_S3_ENDPOINT` | — | — | S3/RustFS endpoint (host:port or full URL; scheme derived from it). Empty disables dataset digest verification. |
| `s3.access_key` | `AXISML_S3_ACCESS_KEY` | — | — | S3/RustFS access key |
| `s3.secret_key` | `AXISML_S3_SECRET_KEY`<br>`AXISML_S3_SECRET_KEY_FILE` | — | yes | S3/RustFS secret key |
| `s3.bucket` | `AXISML_S3_BUCKET` | `axisml-artifact-hub` | — | S3 bucket datasets are stored in |

## platform-backend

Config file: `/etc/axisml/config.yaml` (override with `--config` or `AXISML_CONFIG`). Every key is also overridable by its `AXISML_` variable.

| Key | Environment variable | Default | Secret | Description |
|---|---|---|---|---|
| `database.host` | `AXISML_DATABASE_HOST` | `localhost` | — | PostgreSQL host |
| `database.port` | `AXISML_DATABASE_PORT` | `5432` | — | PostgreSQL port |
| `database.name` | `AXISML_DATABASE_NAME` | `axisml` | — | Database name |
| `database.user` | `AXISML_DATABASE_USER` | `axisml` | — | Database user |
| `database.password` | `AXISML_DATABASE_PASSWORD`<br>`AXISML_DATABASE_PASSWORD_FILE` | — | yes | Database password |
| `database.sslmode` | `AXISML_DATABASE_SSLMODE` | `disable` | — | libpq sslmode: disable \| require \| verify-full |
| `log.level` | `AXISML_LOG_LEVEL` | `info` | — | Log level: debug \| info \| warn \| error |
| `log.format` | `AXISML_LOG_FORMAT` | `json` | — | Log format: json \| console |
| `system.cluster_manager` | `AXISML_SYSTEM_CLUSTER_MANAGER` | `http://axisml-cluster-manager.axisml-system:8080` | — | cluster-manager endpoint |
| `system.compute_service` | `AXISML_SYSTEM_COMPUTE_SERVICE` | `http://axisml-compute-service.axisml-system:8080` | — | compute-service endpoint |
| `system.artifact_hub` | `AXISML_SYSTEM_ARTIFACT_HUB` | `http://axisml-artifact-hub.axisml-system:8080` | — | artifact-hub endpoint |
| `cache.addr` | `AXISML_CACHE_ADDR` | — | — | Redis address host:port (empty disables the cache) |
| `cache.password` | `AXISML_CACHE_PASSWORD`<br>`AXISML_CACHE_PASSWORD_FILE` | — | yes | Redis password |
| `cache.db` | `AXISML_CACHE_DB` | `0` | — | Redis logical database |
| `auth.login_token_ttl` | `AXISML_AUTH_LOGIN_TOKEN_TTL` | `12h` | — | Login session token lifetime |
| `auth.jwt_private_key_pem` | `AXISML_AUTH_JWT_PRIVATE_KEY_PEM`<br>`AXISML_AUTH_JWT_PRIVATE_KEY_PEM_FILE` | — | yes | RS256 signing key PEM (ephemeral if unset; JWKS kid derived from the key) |
| `bootstrap.username` | `AXISML_BOOTSTRAP_USERNAME` | `admin` | — | Initial system-admin username |
| `bootstrap.password` | `AXISML_BOOTSTRAP_PASSWORD`<br>`AXISML_BOOTSTRAP_PASSWORD_FILE` | — | yes | Initial system-admin password |
