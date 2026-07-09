package handler

import (
	"context"
	"fmt"

	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	axisml "github.com/axisml/axisml/axisml-system/apis/mlservice/v1alpha1"
)

// notImplementedHandler reserves a (backend, engine) slot in the registry so
// the dispatcher's "no handler" branch never fires for documented backends.
// The Validate method returns an explicit pointer to the design doc so that
// Compute and operators see the same wording for unimplemented backends.
type notImplementedHandler struct{ key Key }

func (h *notImplementedHandler) Key() Key { return h.key }

func (h *notImplementedHandler) Validate(_ *axisml.MLServiceSpec) Validation {
	return Validation{Errors: []string{
		fmt.Sprintf(
			"backend (%s, %s) not implemented; see axisml-system/docs/compute-operator.md §4 / §5.1",
			h.key.Backend, h.key.Engine,
		),
	}}
}

func (h *notImplementedHandler) Reconcile(_ context.Context, _ *axisml.MLService) (Result, error) {
	return Result{}, nil
}

func (h *notImplementedHandler) MapStatus(_ Snapshot) StatusUpdate {
	return StatusUpdate{Phase: axisml.PhaseFailed, Message: "handler not implemented"}
}

func (h *notImplementedHandler) Cleanup(_ context.Context, _ *axisml.MLService) error { return nil }

func (h *notImplementedHandler) WatchTargets() []client.Object { return nil }

func (h *notImplementedHandler) RequiredRBAC() []rbacv1.PolicyRule { return nil }

// RegisterStubs registers placeholder handlers for the backend tuples the
// design doc enumerates but defers to follow-up specs (§11). Called from
// cmd/manager/main.go so the set is explicit.
//
// We do NOT register a `custom/*` wildcard: the dispatcher does exact key
// lookup, so a literal "*" engine would never match a real CR. Custom backends
// fall through to the dispatcher's "no handler for backend=…" branch until a
// concrete (custom, <engine>) handler ships.
func RegisterStubs() {
	stubs := []Key{
		{Backend: "kserve", Engine: "inference"},
		{Backend: "kserve", Engine: "llminference"},
	}
	for _, k := range stubs {
		k := k
		Register(k, func(_ manager.Manager) (Handler, error) {
			return &notImplementedHandler{key: k}, nil
		})
	}
}
