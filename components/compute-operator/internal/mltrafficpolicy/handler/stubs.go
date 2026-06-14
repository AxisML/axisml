package handler

import (
	"context"
	"fmt"

	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	mltp "github.com/axisml/axisml/components/compute-operator/api/mltrafficpolicy/v1alpha1"
)

// notImplementedHandler reserves a (backend, engine) slot in the registry so
// the dispatcher's "no handler" branch never fires for documented backends.
type notImplementedHandler struct{ key Key }

func (h *notImplementedHandler) Key() Key { return h.key }

func (h *notImplementedHandler) Validate(_ *mltp.MLTrafficPolicySpec) Validation {
	return Validation{Errors: []string{
		fmt.Sprintf(
			"backend (%s, %s) not implemented; see docs/system_design/components/compute-operator.md §4.3 / §9",
			h.key.Backend, h.key.Engine,
		),
	}}
}

func (h *notImplementedHandler) Reconcile(_ context.Context, _ *mltp.MLTrafficPolicy) (Result, error) {
	return Result{}, nil
}

func (h *notImplementedHandler) MapStatus(_ Snapshot) StatusUpdate {
	return StatusUpdate{Phase: mltp.PhaseFailed, Message: "handler not implemented"}
}

func (h *notImplementedHandler) Cleanup(_ context.Context, _ *mltp.MLTrafficPolicy) error { return nil }

func (h *notImplementedHandler) WatchTargets() []client.Object { return nil }

func (h *notImplementedHandler) RequiredRBAC() []rbacv1.PolicyRule { return nil }

// RegisterStubs registers placeholder handlers for the backend tuples the
// design doc enumerates but defers (compute-operator.md §4.3.4 / §9). The
// (kserve, inference) InferenceService canary handler is reserved.
func RegisterStubs() {
	stubs := []Key{
		{Backend: mltp.BackendKindKServe, Engine: mltp.EngineInference},
	}
	for _, k := range stubs {
		k := k
		Register(k, func(_ manager.Manager) (Handler, error) {
			return &notImplementedHandler{key: k}, nil
		})
	}
}
