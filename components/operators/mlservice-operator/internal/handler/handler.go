// Package handler defines the (backend, engine) plugin contract that the
// dispatcher routes MLService reconciles through. See mlservice-operator.md §7.
package handler

import (
	"context"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	axisml "github.com/axisml/axisml/components/operators/mlservice-operator/api/v1alpha1"
)

// Key identifies a registered (backend.name, backend.engine) tuple.
type Key struct {
	Backend string
	Engine  string
}

func (k Key) String() string { return k.Backend + "/" + k.Engine }

// Validation is the result of pure spec validation.
// Errors fail fast; Warnings record non-blocking observations.
type Validation struct {
	Errors   []string
	Warnings []string
}

func (v Validation) OK() bool { return len(v.Errors) == 0 }

// Snapshot is the read-only input MapStatus operates on. The dispatcher
// populates Children from the informer cache; handlers must not issue
// additional API calls inside MapStatus (§7).
type Snapshot struct {
	Service  *axisml.MLService
	Children []client.Object
}

// StatusUpdate is the merge-patch payload the dispatcher writes to status.
// Handlers communicate every status mutation through this struct.
type StatusUpdate struct {
	Phase         axisml.Phase
	Message       string
	Endpoint      string
	ReadyReplicas int32
	Selector      string
	Conditions    []metav1.Condition
	Roles         []axisml.RoleStatus
}

// Result reports back to the dispatcher whether the reconcile triggered any
// observable side-effect. Reserved for future use (currently unused).
type Result struct{}

// Handler is the plugin contract per mlservice-operator.md §7. Handlers must
// be safe to call concurrently and must keep MapStatus pure.
type Handler interface {
	Key() Key
	Validate(spec *axisml.MLServiceSpec) Validation
	Reconcile(ctx context.Context, mls *axisml.MLService) (Result, error)
	MapStatus(snap Snapshot) StatusUpdate
	Cleanup(ctx context.Context, mls *axisml.MLService) error
	WatchTargets() []client.Object
	RequiredRBAC() []rbacv1.PolicyRule
}

// Factory builds a Handler for the supplied manager. Used by registry.Build
// to defer client wiring until the manager is ready.
type Factory func(mgr manager.Manager) (Handler, error)
