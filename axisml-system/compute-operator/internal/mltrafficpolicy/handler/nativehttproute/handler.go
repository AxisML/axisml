// Package nativehttproute implements the (native, httproute) MLTrafficPolicy
// Handler: a single weighted Gateway API HTTPRoute spanning the member
// services. No Pods are derived. See compute-operator.md §4.3.3.
package nativehttproute

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	mltp "github.com/axisml/axisml/axisml-system/apis/mltrafficpolicy/v1alpha1"
	"github.com/axisml/axisml/axisml-system/apis/pkg/workloadname"
	"github.com/axisml/axisml/axisml-system/compute-operator/internal/mltrafficpolicy/handler"
)

const (
	// GatewayParentName / GatewayParentNamespace mirror the MLService route:
	// the shared AxisML Envoy Gateway listener that derived HTTPRoutes attach
	// to. Cross-namespace attachment is permitted by the Gateway's
	// allowedRoutes.namespaces.from=All (infra.md §3.1).
	GatewayParentName      = "axisml-gateway"
	GatewayParentNamespace = "axisml-infra"
)

// Handler is the (native, httproute) implementation.
type Handler struct {
	client client.Client
}

// New is the registry Factory entry point.
func New(mgr manager.Manager) (handler.Handler, error) {
	return &Handler{client: mgr.GetClient()}, nil
}

func init() {
	handler.Register(handler.Key{Backend: mltp.BackendKindNative, Engine: mltp.EngineHTTPRoute}, New)
}

func (h *Handler) Key() handler.Key {
	return handler.Key{Backend: mltp.BackendKindNative, Engine: mltp.EngineHTTPRoute}
}

// Validate enforces the per-mode member-count + role invariants we can check
// from spec alone (compute-service is the authoritative gate; this is the
// operator-side defensive check).
func (h *Handler) Validate(spec *mltp.MLTrafficPolicySpec) handler.Validation {
	v := handler.Validation{}
	if len(spec.Backends) == 0 {
		v.Errors = append(v.Errors, "backends must declare at least one member")
		return v
	}
	switch spec.Mode {
	case mltp.TrafficModeWeighted:
		if len(spec.Backends) < 2 {
			v.Errors = append(v.Errors, "weighted mode requires at least 2 backends")
		}
	case mltp.TrafficModeCanary:
		if len(spec.Backends) != 2 {
			v.Errors = append(v.Errors, "canary mode requires exactly 2 backends")
		} else {
			var nStable, nCanary int
			for _, m := range spec.Backends {
				switch m.Role {
				case mltp.RoleStable:
					nStable++
				case mltp.RoleCanary:
					nCanary++
				}
			}
			if nStable != 1 || nCanary != 1 {
				v.Errors = append(v.Errors,
					"canary mode requires exactly one role=stable and one role=canary backend")
			}
		}
	case mltp.TrafficModeBlueGreen:
		if len(spec.Backends) != 2 {
			v.Errors = append(v.Errors, "bluegreen mode requires exactly 2 backends")
		}
	default:
		v.Errors = append(v.Errors, fmt.Sprintf("unknown mode %q", spec.Mode))
	}

	seen := map[string]bool{}
	for i, m := range spec.Backends {
		if m.ServiceName == "" {
			v.Errors = append(v.Errors, fmt.Sprintf("backends[%d].serviceName is required", i))
		}
		if seen[m.ServiceName] {
			v.Errors = append(v.Errors, fmt.Sprintf("backends[%d]: duplicate serviceName %q", i, m.ServiceName))
		}
		seen[m.ServiceName] = true
		if m.Weight < 0 || m.Weight > 100 {
			v.Errors = append(v.Errors, fmt.Sprintf("backends[%d].weight must be 0..100; got %d", i, m.Weight))
		}
	}

	if spec.Endpoint.Auth != nil && spec.Endpoint.Auth.Type == mltp.EndpointAuthJWT {
		// Fail closed: SecurityPolicy derivation is not wired yet, so programming
		// the route would expose an UNAUTHENTICATED endpoint under the impression
		// it is JWT-protected. Reject until the SecurityPolicy follow-up lands.
		v.Errors = append(v.Errors,
			"endpoint.auth=jwt is not yet supported (SecurityPolicy derivation pending); use auth.type=none")
	}
	if spec.Backend.Config != nil && len(spec.Backend.Config.Raw) > 0 {
		v.Warnings = append(v.Warnings,
			"(native, httproute) ignores backend.config; reserved for future fields")
	}
	return v
}

// Reconcile resolves each member's K8s Service to a port and renders a single
// weighted HTTPRoute.
//
// Weighted routing is all-or-nothing: Gateway API renormalises weights across
// only the backendRefs present, so programming a partial member set would
// silently shift an unresolved member's share onto the survivors — a 5%/95%
// canary collapses to 100% on the canary while the stable Service is briefly
// absent. So for a multi-member policy we refuse to (re)program until every
// member resolves, holding any existing route as last-good and returning an
// error to requeue. A single-member policy has no weights to skew, so a missing
// member just yields an empty route (Pending) as before.
func (h *Handler) Reconcile(ctx context.Context, p *mltp.MLTrafficPolicy) (handler.Result, error) {
	var refs []gwapiv1.HTTPBackendRef
	var missing []string
	for _, m := range p.Spec.Backends {
		serviceName := workloadname.Related(p, p.Labels[mltp.LabelTenant], m.ServiceName)
		port, ok, err := h.resolveServicePort(ctx, p.Namespace, serviceName)
		if err != nil {
			return handler.Result{}, fmt.Errorf("resolve service %s: %w", m.ServiceName, err)
		}
		if !ok {
			missing = append(missing, m.ServiceName)
			continue
		}
		refs = append(refs, backendRef(serviceName, port, m.Weight))
	}
	if len(missing) > 0 && len(p.Spec.Backends) > 1 {
		// Hold the last-good route rather than program a skewed weighted set.
		return handler.Result{}, fmt.Errorf(
			"not all member services resolve yet (missing: %s); holding route to avoid shifting weights onto the resolvable subset",
			strings.Join(missing, ", "))
	}
	if len(refs) == 0 {
		// Nothing routable yet — don't create an HTTPRoute with empty
		// backendRefs. MapStatus reports Pending until a member appears.
		return handler.Result{}, nil
	}
	route := buildHTTPRoute(p, refs)
	if err := h.upsertHTTPRoute(ctx, p, route); err != nil {
		return handler.Result{}, fmt.Errorf("upsert httproute: %w", err)
	}
	return handler.Result{}, nil
}

func (h *Handler) MapStatus(snap handler.Snapshot) handler.StatusUpdate {
	return mapStatus(snap)
}

// Cleanup is a no-op: ownerReference cascade deletes the HTTPRoute. Member
// MLServices are never touched by the policy (compute-operator.md §4.3.2).
func (h *Handler) Cleanup(_ context.Context, _ *mltp.MLTrafficPolicy) error {
	return nil
}

func (h *Handler) WatchTargets() []client.Object {
	return []client.Object{
		&gwapiv1.HTTPRoute{},
	}
}

func (h *Handler) RequiredRBAC() []rbacv1.PolicyRule {
	return []rbacv1.PolicyRule{
		{APIGroups: []string{"gateway.networking.k8s.io"}, Resources: []string{"httproutes"},
			Verbs: []string{"get", "list", "watch", "create", "update", "patch", "delete"}},
		{APIGroups: []string{""}, Resources: []string{"services"},
			Verbs: []string{"get", "list", "watch"}},
		{APIGroups: []string{""}, Resources: []string{"events"},
			Verbs: []string{"create", "patch"}},
	}
}

// resolveServicePort finds the member Service's routable port, preferring a
// port named "http". Returns ok=false when the Service has no ports or does
// not exist yet.
func (h *Handler) resolveServicePort(ctx context.Context, namespace, name string) (int32, bool, error) {
	svc := &corev1.Service{}
	err := h.client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, svc)
	switch {
	case apierrors.IsNotFound(err):
		return 0, false, nil
	case err != nil:
		return 0, false, err
	}
	for _, p := range svc.Spec.Ports {
		if p.Name == "http" {
			return p.Port, true, nil
		}
	}
	if len(svc.Spec.Ports) > 0 {
		return svc.Spec.Ports[0].Port, true, nil
	}
	return 0, false, nil
}

func (h *Handler) upsertHTTPRoute(ctx context.Context, p *mltp.MLTrafficPolicy, desired *gwapiv1.HTTPRoute) error {
	if err := controllerutil.SetControllerReference(p, desired, h.client.Scheme()); err != nil {
		return err
	}
	current := &gwapiv1.HTTPRoute{}
	err := h.client.Get(ctx, types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}, current)
	switch {
	case apierrors.IsNotFound(err):
		return h.client.Create(ctx, desired)
	case err != nil:
		return err
	}
	patched := current.DeepCopy()
	patched.Spec = desired.Spec
	patched.Labels = mergeLabels(current.Labels, desired.Labels)
	if equality.Semantic.DeepEqual(current.Spec, patched.Spec) &&
		equality.Semantic.DeepEqual(current.Labels, patched.Labels) {
		return nil
	}
	return h.client.Patch(ctx, patched, client.MergeFrom(current))
}

func mergeLabels(existing, desired map[string]string) map[string]string {
	if len(existing) == 0 {
		return desired
	}
	out := make(map[string]string, len(existing)+len(desired))
	for k, v := range existing {
		out[k] = v
	}
	for k, v := range desired {
		out[k] = v
	}
	return out
}
