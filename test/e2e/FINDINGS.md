# E2E bring-up findings

Issues surfaced while getting the system-layer e2e suite to pass against a real
minikube cluster. Two kinds: **product/infra bugs** (fixed in the repo) and
**test-design corrections** (the e2e tests encoded wrong assumptions).

## Product / infra bugs (fixed)

1. **Service Dockerfiles miss `pkg/openapigen`** — `cluster-manager`,
   `compute-service`, `artifact-hub` all `replace … => ../../pkg/openapigen`
   but their Dockerfiles never `COPY pkg/openapigen/`, so `go mod download`
   fails and the images can't build. Added the COPY to all three.

2. **compute-service Dockerfile misses `cluster-manager`** — compute-service
   `replace … => ../cluster-manager` (ResourcePool API) but the Dockerfile only
   copied tenant-operator + compute-operator + itself. Added the COPY.

3. **cluster-manager chart passes a dead flag** — the system chart's
   cluster-manager Deployment passes `--namespace-denylist`, but the binary
   defines no such flag → CrashLoopBackOff. The `namespaceDenylist` values block
   (and its comment claiming cluster-manager validates Tenant CR namespaces) is
   stale from an earlier design where cluster-manager owned tenant validation;
   it now only does ResourcePool CRUD. Removed the arg from the template.
   *Recommendation:* also drop the orphaned `namespaceDenylist` values + comment,
   or re-introduce the flag if denylist validation is still wanted.

4. **artifact-hub never migrates its DB** — compute-service and artifact-hub
   share the one infra Postgres database and both used golang-migrate's default
   `schema_migrations` table. compute-service migrates first (version 1), so
   artifact-hub's own version-1 migration sees `ErrNoChange` and never creates
   its `artifacts` table → every artifact-hub call 500s. Fixed by giving
   artifact-hub a service-scoped migrations table (`artifact_hub_schema_migrations`).
   *Recommendation:* either give each system service its own database, or make
   **every** service use a service-scoped migrations table (compute-service
   still relies on the default), so the pattern is symmetric and collision-proof.

5. **compute-service scheme misses ResourcePool** — `k8sclient.Scheme()`
   registers tenant/mlrun/mlservice but not the cluster-manager ResourcePool
   type it reads via poolcache, so Job/Service creation 503s with "no kind is
   registered for the type v1alpha1.ResourcePool". Added `resourcepoolv1alpha1.AddToScheme`.

6. **compute-service ClusterRole misses permissions** — the chart granted
   tenants/mlruns/mlservices + core pods/events, but not:
   - `resourcepools` (read) → poolcache informer `forbidden`, so Job/Service
     creation hangs until the request times out;
   - `persistentvolumeclaims` → workspace services can't create their PVC;
   - `events.k8s.io` events → the kubeproxy `/events` endpoint's informer is
     forbidden and the cached read blocks ~40s before a 503 (it only had core
     `""` events, but the endpoint reads `events.k8s.io/v1`).
   Added all three to `templates/compute-service/rbac.yaml`.

7. **artifact-hub zot password mismatch** — the Makefile dev-default set
   `artifactHub.storage.oci.adminSecretRef.password=axisml`, but the infra zot
   chart ships a static `admin:admin` htpasswd. artifact-hub handed out
   `admin:axisml` → zot 401 on push (anonymous is allowed, but *wrong* creds are
   rejected). Changed the dev-default to `admin` to match zot's htpasswd.
   *Recommendation:* wire a single shared value so the zot htpasswd and
   artifact-hub's issued password can't drift.

8. **artifact-hub points at the wrong zot Service name** — the system chart's
   `artifactHub.storage.oci.endpoint` defaulted to `http://axisml-infra-zot.axisml-infra:5000`,
   but the infra zot chart sets `fullnameOverride: zot`, so the Service is
   `zot`. artifact-hub's `complete` HEAD to the registry failed with EOF (host
   doesn't resolve to zot). Changed the endpoint default to `http://zot.axisml-infra:5000`.

(My e2e OCI push helper in `oci_test.go` turned out correct — the "Known
validation point" flagged in the README is resolved; the failures were the two
deployment bugs above.)

## Test-design corrections (the e2e tests were wrong)

These were assumptions in the suite (some inherited from the original design
doc) that don't match the system's actual, intended behavior:

- **Tenant child resources are InitResources-gated.** A bare Tenant gets only a
  namespace + ElasticQuota; RBAC/SA/Secret/ConfigMap are created only when
  `spec.initResources` requests them. The provisioning test now requests a
  ServiceAccount-with-RBAC and asserts the resulting SA + Role.
- **Tenant namespace is intentionally retained on deletion** (design §6.1:
  "never delete, no ownerReference"). Deleting a Tenant CR removes the CR but
  keeps the namespace. Test renamed to `TestTenant_DeletionRetainsNamespace`.
- **Resource units are mutable** (only the unit *name* is immutable). The
  cluster-manager test now PATCHes a unit and asserts the change, instead of
  expecting a 4xx.
- **MLRun/MLService require identity labels.** The operator validates that
  compute-service's labels are present — `axisml.io/run-id` (mandatory) /
  `axisml.io/service-id`, plus tenant/quota. Direct-CR tests now stamp them.
- **Unit/tenant names have length rules** (e.g. resource-unit name 3–40 chars);
  test fixtures adjusted.
- **artifact-hub does not enforce identity** — a missing `X-Axisml-User` falls
  back to `anonymous` (only cluster-manager 401s). The 401 test became an
  "anonymous allowed" test.
- **Model spec requires `framework` (valid set) + `format`** (design §5.1), not
  just any spec object.
- **DELETE is a soft-delete to status `Deleting`**, reclaimed later by the GC
  worker (≈5 min interval); the row keeps showing in listings until then. The
  test now asserts the `Deleting` status transition, not immediate disappearance.
- **ElasticQuota admission needs a guaranteed `min` to be deterministic under
  churn.** A quota with only `max` (min=0) relies on koord-scheduler borrowing
  shared capacity, which gets flaky once the suite has created/deleted many
  quotas — the first pod can stay Pending for minutes despite free node CPU.
  Setting `min` equal to one pod's request makes admission deterministic. (This
  is a real operational caveat worth surfacing in the tenant/quota docs, not
  just a test fix.)

## Harness robustness fixes

- **Shared tenant is per-run-unique + hard-cleaned.** compute-service
  soft-deletes tenants, so reusing a fixed `e2e` name across runs returns 409 on
  re-create, leaving no CR/quota. The shared tenant now gets a per-run-unique
  name, setup waits for the namespace **and** its ElasticQuota, and teardown
  hard-removes the CR + namespace via the admin client.
- **Per-test namespaces are deleted immediately** (the operator never deletes
  them), instead of polling 90s for a deletion that never happens — this also
  prevents leftover `sleep` pods from accumulating and saturating the node.

## Environment gotcha (not a code bug)

- Rebuilding a component image under the same `0.1.0` tag is not picked up by the
  kubelet (`imagePullPolicy: IfNotPresent` + unchanged tag). Use a unique tag and
  `kubectl set image`, or delete the pod after `minikube image load`.
