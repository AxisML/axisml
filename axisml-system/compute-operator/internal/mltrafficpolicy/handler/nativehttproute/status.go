package nativehttproute

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	mltp "github.com/axisml/axisml/axisml-system/apis/mltrafficpolicy/v1alpha1"
	"github.com/axisml/axisml/axisml-system/compute-operator/internal/mltrafficpolicy/handler"
)

const condTypeAvailable = "Available"

// mapStatus is pure: it derives the policy phase + endpoint + per-backend
// status from the cached HTTPRoute. Backend readiness is coarse — it reflects
// whether the route programmed (Accepted + ResolvedRefs), not live member
// pod readiness; live member readiness is a follow-up refinement.
func mapStatus(snap handler.Snapshot) handler.StatusUpdate {
	p := snap.Policy
	route := findHTTPRoute(snap.Children, p.Name, p.Namespace)

	accepted := routeAccepted(route)
	resolved := routeResolved(route)
	programmed := accepted && resolved

	backends := make([]mltp.BackendStatus, 0, len(p.Spec.Backends))
	for _, m := range p.Spec.Backends {
		backends = append(backends, mltp.BackendStatus{
			ServiceName: m.ServiceName,
			Weight:      m.Weight,
			Ready:       programmed,
		})
	}

	upd := handler.StatusUpdate{Backends: backends}
	switch {
	case route == nil:
		upd.Phase = mltp.PhasePending
		upd.Message = "HTTPRoute not yet created (waiting for member services)"
	case programmed:
		upd.Phase = mltp.PhaseReady
	case accepted && !resolved:
		upd.Phase = mltp.PhaseDegraded
		upd.Message = "some backendRefs unresolved"
	default:
		upd.Phase = mltp.PhasePending
		upd.Message = "HTTPRoute not yet Accepted"
	}
	upd.Endpoint = deriveEndpoint(p, accepted)
	upd.Conditions = buildConditions(upd.Phase, upd.Message, p.Generation)
	return upd
}

// deriveEndpoint returns the external entrypoint URL once the route is
// Accepted. With a hostname it is a full URL; otherwise the path under the
// shared gateway host (which the caller already knows).
func deriveEndpoint(p *mltp.MLTrafficPolicy, accepted bool) string {
	if !accepted {
		return ""
	}
	path := "/"
	if p.Spec.Endpoint.Path != "" {
		path = p.Spec.Endpoint.Path
	}
	if p.Spec.Endpoint.Hostname != "" {
		return fmt.Sprintf("https://%s%s", p.Spec.Endpoint.Hostname, path)
	}
	return path
}

func routeAccepted(route *gwapiv1.HTTPRoute) bool {
	return routeConditionTrue(route, string(gwapiv1.RouteConditionAccepted))
}

func routeResolved(route *gwapiv1.HTTPRoute) bool {
	return routeConditionTrue(route, string(gwapiv1.RouteConditionResolvedRefs))
}

func routeConditionTrue(route *gwapiv1.HTTPRoute, condType string) bool {
	if route == nil {
		return false
	}
	for _, parent := range route.Status.Parents {
		for _, c := range parent.Conditions {
			if c.Type == condType && c.Status == metav1.ConditionTrue {
				return true
			}
		}
	}
	return false
}

func buildConditions(phase mltp.Phase, message string, generation int64) []metav1.Condition {
	available := metav1.Condition{
		Type:               condTypeAvailable,
		Status:             metav1.ConditionFalse,
		Reason:             string(phase),
		Message:            message,
		ObservedGeneration: generation,
	}
	if phase == mltp.PhaseReady {
		available.Status = metav1.ConditionTrue
		available.Reason = "RouteProgrammed"
	}
	return []metav1.Condition{available}
}

func findHTTPRoute(children []client.Object, name, namespace string) *gwapiv1.HTTPRoute {
	for _, c := range children {
		if r, ok := c.(*gwapiv1.HTTPRoute); ok && r.Name == name && r.Namespace == namespace {
			return r
		}
	}
	return nil
}
