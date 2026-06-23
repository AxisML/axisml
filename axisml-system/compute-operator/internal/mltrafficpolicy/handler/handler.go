// Package handler defines the (backend, engine) plugin contract that the
// MLTrafficPolicy dispatcher routes reconciles through. See
// compute-operator.md §4.3 / §5.1.
package handler

import (
	"context"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	mltp "github.com/axisml/axisml/components/compute-operator/api/mltrafficpolicy/v1alpha1"
)

// Key identifies a registered (backend.name, backend.engine) tuple.
type Key struct {
	Backend string
	Engine  string
}

func (k Key) String() string { return k.Backend + "/" + k.Engine }

// Validation is the result of pure spec validation.
type Validation struct {
	Errors   []string
	Warnings []string
}

func (v Validation) OK() bool { return len(v.Errors) == 0 }

// Snapshot is the read-only input MapStatus operates on. The dispatcher
// populates Children from the informer cache; handlers must not issue
// additional API calls inside MapStatus.
type Snapshot struct {
	Policy   *mltp.MLTrafficPolicy
	Children []client.Object
}

// StatusUpdate is the merge-patch payload the dispatcher writes to status.
type StatusUpdate struct {
	Phase      mltp.Phase
	Message    string
	Endpoint   string
	Backends   []mltp.BackendStatus
	Conditions []metav1.Condition
}

// Result reports back to the dispatcher whether the reconcile triggered any
// observable side-effect. Reserved for future use.
type Result struct{}

// Handler is the plugin contract. Handlers must be safe to call concurrently
// and must keep MapStatus pure.
type Handler interface {
	Key() Key
	Validate(spec *mltp.MLTrafficPolicySpec) Validation
	Reconcile(ctx context.Context, p *mltp.MLTrafficPolicy) (Result, error)
	MapStatus(snap Snapshot) StatusUpdate
	Cleanup(ctx context.Context, p *mltp.MLTrafficPolicy) error
	WatchTargets() []client.Object
	RequiredRBAC() []rbacv1.PolicyRule
}

// Factory builds a Handler for the supplied manager.
type Factory func(mgr manager.Manager) (Handler, error)
