// Package extensions publishes the deployment-form-neutral extension seams that
// Compute Service business logic depends on — the contracts an alternate
// deployment form (notably AxisML Lite's axisml-core) must implement. Each
// composition root (the Kubernetes binary, or Lite's axisml-core) injects
// concrete implementations:
//
//   - ComputeRuntime is the engine that actually executes AxisML workloads. The
//     Kubernetes runtime (internal/kuberuntime) writes the MLRun / MLService /
//     MLTrafficPolicy CRs to the apiserver, where compute-operator maps them onto
//     Job / Deployment / StatefulSet / Service / HTTPRoute. Lite's Standalone
//     (Docker) runtime receives the same objects and maps them onto containers,
//     volumes, networks and dynamic proxy config.
//   - ResourceResolver reads the ResourcePool CR and its embedded units by name.
//     Kubernetes reads the ResourcePool CR informer cache; Lite reads a static
//     config catalog. The business layer merges the looked-up (pool, unit) into
//     the workload spec snapshot (internal/resource, design §5.4).
//   - WorkspaceVolumeProvisioner manages the durable volume backing a
//     kind=workspace MLService. Kubernetes creates a PVC; Lite creates a managed
//     Docker volume via the runtime.
//
// The MLRun / MLService / MLTrafficPolicy API types are Compute's unified
// desired-state contract: Compute Service builds the desired object from its
// PostgreSQL business records and hands it to a ComputeRuntime. Both runtimes
// share the published CR API contract, defaulting, immutable field checks, Spec
// validation and CR-Status semantics. The runtime port only speaks the AxisML
// API types plus standard Kubernetes types; Docker SDK requests, proxy config,
// Deployments and HTTPRoutes are handler/adapter internals that never cross this
// boundary.
//
// Keeping this a leaf package (CR + corev1 types only, no internal imports) lets
// both pkg/module and the internal business packages depend on it without an
// import cycle.
package extensions

import (
	"context"
	"errors"
	"io"

	corev1 "k8s.io/api/core/v1"
	eventsv1 "k8s.io/api/events/v1"
	"k8s.io/apimachinery/pkg/types"

	mlrunv1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mlrun/v1alpha1"
	mlservicev1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mlservice/v1alpha1"
	mltrafficpolicyv1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mltrafficpolicy/v1alpha1"
)

// ErrInstanceNotOwned is returned by instance log / event lookups when the
// named instance exists but does not belong to the addressed workload. Callers
// map it to 403 Forbidden (distinct from a 404 for a missing instance), so the
// workload URL space cannot leak instances owned by other workloads.
var ErrInstanceNotOwned = errors.New("instance does not belong to this workload")

// ComputeRuntime executes and observes the AxisML workload contract.
//
// Apply requests carry the full desired object with an empty .status; the
// runtime derives namespace and name from the CR metadata and combines them
// with the workload kind to form the resource key. All other operations locate
// a workload by its Kubernetes types.NamespacedName.
//
// Observe returns the corresponding CR Status. When the underlying resource
// does not exist, Observe returns a Kubernetes NotFound error recognizable by
// apierrors.IsNotFound; any other error indicates the observation itself
// failed. All Apply and Delete operations must be idempotent so that a process
// crash between a successful runtime call and the PG status commit is safe to
// retry.
//
// An instance is the runtime's unified term for a single running unit: a Pod in
// the Kubernetes implementation, a Docker container in the Standalone one.
// Instance names must be stable, and log / event lookups must verify that the
// named instance belongs to the addressed workload. Resource-level events and
// instance-level events are returned by distinct methods; MLTrafficPolicy has
// no instances and therefore exposes only resource-level events.
type ComputeRuntime interface {
	ApplyMLRun(ctx context.Context, desired *mlrunv1alpha1.MLRun) error
	ObserveMLRun(ctx context.Context, key types.NamespacedName) (mlrunv1alpha1.MLRunStatus, error)
	CancelMLRun(ctx context.Context, key types.NamespacedName) error
	DeleteMLRun(ctx context.Context, key types.NamespacedName) error
	ListMLRunInstances(ctx context.Context, key types.NamespacedName) (*corev1.PodList, error)
	GetMLRunInstanceLogs(
		ctx context.Context,
		key types.NamespacedName,
		instance string,
		opts *corev1.PodLogOptions,
	) (io.ReadCloser, error)
	GetMLRunInstanceEvents(
		ctx context.Context,
		key types.NamespacedName,
		instance string,
	) (*eventsv1.EventList, error)
	GetMLRunEvents(ctx context.Context, key types.NamespacedName) (*eventsv1.EventList, error)

	ApplyMLService(ctx context.Context, desired *mlservicev1alpha1.MLService) error
	ObserveMLService(ctx context.Context, key types.NamespacedName) (mlservicev1alpha1.MLServiceStatus, error)
	DeleteMLService(ctx context.Context, key types.NamespacedName) error
	ListMLServiceInstances(ctx context.Context, key types.NamespacedName) (*corev1.PodList, error)
	GetMLServiceInstanceLogs(
		ctx context.Context,
		key types.NamespacedName,
		instance string,
		opts *corev1.PodLogOptions,
	) (io.ReadCloser, error)
	GetMLServiceInstanceEvents(
		ctx context.Context,
		key types.NamespacedName,
		instance string,
	) (*eventsv1.EventList, error)
	GetMLServiceEvents(ctx context.Context, key types.NamespacedName) (*eventsv1.EventList, error)

	ApplyMLTrafficPolicy(ctx context.Context, desired *mltrafficpolicyv1alpha1.MLTrafficPolicy) error
	ObserveMLTrafficPolicy(ctx context.Context, key types.NamespacedName) (mltrafficpolicyv1alpha1.MLTrafficPolicyStatus, error)
	DeleteMLTrafficPolicy(ctx context.Context, key types.NamespacedName) error
	GetMLTrafficPolicyEvents(ctx context.Context, key types.NamespacedName) (*eventsv1.EventList, error)
}
