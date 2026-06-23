// Package computeruntime publishes the Compute Runtime contract: the
// deployment-form-neutral seam between Compute Service business logic and the
// engine that actually executes AxisML workloads.
//
// The MLRun / MLService / MLTrafficPolicy API types are Compute's unified
// desired-state contract. Compute Service builds the desired object from its
// PostgreSQL business records and hands it to a ComputeRuntime:
//
//   - The Kubernetes runtime writes the object to the apiserver, where
//     compute-operator maps it onto Job / Deployment / StatefulSet / Service /
//     HTTPRoute. See pkg/computeruntime/kubernetes.
//   - A Standalone (Docker) runtime — maintained out-of-tree in the axisml-lite
//     repo — receives the same object and maps it onto containers, volumes,
//     networks and Traefik dynamic config.
//
// Both runtimes share the published CR API contract, defaulting, immutable
// field checks, Spec validation and CR-Status semantics. The runtime port only
// speaks the AxisML MLRun / MLService / MLTrafficPolicy API types plus standard
// Kubernetes types; Docker SDK requests, Traefik config, Deployments and
// HTTPRoutes are handler/adapter internals that never cross this boundary.
package computeruntime

import (
	"context"
	"io"

	corev1 "k8s.io/api/core/v1"
	eventsv1 "k8s.io/api/events/v1"
	"k8s.io/apimachinery/pkg/types"

	mlrunv1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mlrun/v1alpha1"
	mlservicev1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mlservice/v1alpha1"
	mltrafficpolicyv1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mltrafficpolicy/v1alpha1"
)

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
