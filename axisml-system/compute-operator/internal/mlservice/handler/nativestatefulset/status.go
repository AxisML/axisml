package nativestatefulset

import (
	"fmt"
	"sort"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	axisml "github.com/axisml/axisml/axisml-system/apis/mlservice/v1alpha1"
	"github.com/axisml/axisml/axisml-system/apis/pkg/workloadname"
	"github.com/axisml/axisml/axisml-system/compute-operator/internal/mlservice/handler"
)

const (
	// internalEndpointFmt is the in-cluster DNS fallback used when the route
	// is disabled or not yet Accepted (§4 / §6.6.2).
	internalEndpointFmt = "%s.%s.svc.cluster.local:%d"

	condTypeAvailable = "Available"
)

// mapStatus implements §6.6.2 Status 映射. Pure: no API calls, no time.Now()
// (the dispatcher stamps timestamps when it observes a transition).
func mapStatus(snap handler.Snapshot) handler.StatusUpdate {
	mls := snap.Service
	role := mls.Spec.Roles[0]

	// parseBackendConfig is safe to call again: Validate already gated, so
	// any error here would indicate a regression and we fall back to the
	// default service name.
	cfg, _ := parseBackendConfig(mls.Spec.Backend.Config)
	headlessName := defaultedServiceName(mls, cfg)

	sts := findStatefulSet(snap.Children, workloadname.Role(mls, mls.Spec.Roles[0].Name), mls.Namespace)
	svc := findService(snap.Children, headlessName, mls.Namespace)
	route := findHTTPRoute(snap.Children, workloadname.Workload(mls), mls.Namespace)

	desired := role.Replicas
	var ready int32
	if sts != nil {
		ready = sts.Status.ReadyReplicas
	}

	upd := handler.StatusUpdate{
		ReadyReplicas: ready,
		Selector:      formatSelector(selectorLabels(mls, role.Name)),
		Roles: []axisml.RoleStatus{{
			Name:          role.Name,
			Replicas:      desired,
			ReadyReplicas: ready,
		}},
	}

	upd.Phase, upd.Message = derivePhase(sts, desired, ready)
	upd.Endpoint = deriveEndpoint(mls, role, headlessName, svc, route)

	if mls.Spec.Route != nil && mls.Spec.Route.Enabled {
		if !routeAccepted(route) && upd.Phase != axisml.PhaseFailed {
			upd.Phase = axisml.PhaseDegraded
			if upd.Message == "" {
				upd.Message = "HTTPRoute not yet Accepted; falling back to in-cluster DNS"
			}
		}
	}

	upd.Conditions = buildConditions(upd.Phase, upd.Message, mls.Generation)
	return upd
}

// derivePhase maps StatefulSet.status to the four AxisML phases.
//
// StatefulSet has no ProgressDeadlineExceeded analogue, so the
// "ready==0 && desired>0" branch always returns Pending. Terminal failures
// (image pull errors, scheduler refusals, quota rejections) surface through
// the dispatcher's Reconcile error path rather than a status condition;
// users investigating a stuck Pending should read Pod events.
func derivePhase(sts *appsv1.StatefulSet, desired, ready int32) (axisml.Phase, string) {
	if desired == 0 {
		return axisml.PhasePending, "spec.roles[0].replicas is 0"
	}
	if sts == nil {
		return axisml.PhasePending, "StatefulSet not yet created"
	}
	if ready == desired {
		return axisml.PhaseReady, ""
	}
	if ready > 0 && ready < desired {
		return axisml.PhaseDegraded,
			fmt.Sprintf("only %d/%d replicas ready", ready, desired)
	}
	return axisml.PhasePending, "StatefulSet is rolling out"
}

func deriveEndpoint(mls *axisml.MLService, role axisml.RoleSpec, headlessName string, svc *corev1.Service, route *gwapiv1.HTTPRoute) string {
	port := pickEndpointPort(role)
	internal := fmt.Sprintf(internalEndpointFmt, headlessName, mls.Namespace, port)
	if svc == nil {
		return ""
	}
	if mls.Spec.Route == nil || !mls.Spec.Route.Enabled {
		return internal
	}
	if !routeAccepted(route) || mls.Spec.Route.Hostname == "" {
		return internal
	}
	path := "/"
	if mls.Spec.Route.Path != "" {
		path = mls.Spec.Route.Path
	}
	return fmt.Sprintf("https://%s%s", mls.Spec.Route.Hostname, path)
}

func pickEndpointPort(role axisml.RoleSpec) int32 {
	for _, p := range role.Template.Ports {
		if p.Name == "http" {
			return p.ContainerPort
		}
	}
	if len(role.Template.Ports) > 0 {
		return role.Template.Ports[0].ContainerPort
	}
	return 0
}

func routeAccepted(route *gwapiv1.HTTPRoute) bool {
	if route == nil {
		return false
	}
	for _, parent := range route.Status.Parents {
		for _, c := range parent.Conditions {
			if c.Type == string(gwapiv1.RouteConditionAccepted) && c.Status == metav1.ConditionTrue {
				return true
			}
		}
	}
	return false
}

func buildConditions(phase axisml.Phase, message string, generation int64) []metav1.Condition {
	available := metav1.Condition{
		Type:               condTypeAvailable,
		Status:             metav1.ConditionFalse,
		Reason:             string(phase),
		Message:            message,
		ObservedGeneration: generation,
	}
	if phase == axisml.PhaseReady {
		available.Status = metav1.ConditionTrue
		available.Reason = "AllReplicasReady"
	}
	return []metav1.Condition{available}
}

// formatSelector renders the selector labels into the standard
// "key=value,key=value" form expected by the scale subresource. The output
// is sorted by key for deterministic comparisons in tests and status diffs.
func formatSelector(labels map[string]string) string {
	parts := make([]string, 0, len(labels))
	for k, v := range labels {
		parts = append(parts, k+"="+v)
	}
	sort.Strings(parts)
	return strings.Join(parts, ",")
}

func findStatefulSet(children []client.Object, name, namespace string) *appsv1.StatefulSet {
	for _, c := range children {
		if s, ok := c.(*appsv1.StatefulSet); ok && s.Name == name && s.Namespace == namespace {
			return s
		}
	}
	return nil
}

func findService(children []client.Object, name, namespace string) *corev1.Service {
	for _, c := range children {
		if s, ok := c.(*corev1.Service); ok && s.Name == name && s.Namespace == namespace {
			return s
		}
	}
	return nil
}

func findHTTPRoute(children []client.Object, name, namespace string) *gwapiv1.HTTPRoute {
	for _, c := range children {
		if r, ok := c.(*gwapiv1.HTTPRoute); ok && r.Name == name && r.Namespace == namespace {
			return r
		}
	}
	return nil
}
