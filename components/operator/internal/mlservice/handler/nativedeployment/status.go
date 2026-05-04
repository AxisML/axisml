package nativedeployment

import (
	"fmt"
	"sort"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	axisml "github.com/axisml/axisml/components/operator/api/mlservice/v1alpha1"
	"github.com/axisml/axisml/components/operator/internal/mlservice/handler"
)

const (
	// internalEndpointFmt is the fallback DNS form when route.enabled=false
	// or when the HTTPRoute has not yet been Accepted (§4 / §8.1).
	internalEndpointFmt = "%s.%s.svc.cluster.local:%d"

	condTypeAvailable = "Available"
	condTypeProgress  = "Progressing"
	condTypeDegraded  = "Degraded"
)

// mapStatus implements §8.1 Status 映射 + §4 endpoint 二分规则. Pure function:
// no API calls, no time.Now() (timestamps are stamped by the dispatcher when
// it observes a transition).
func mapStatus(snap handler.Snapshot) handler.StatusUpdate {
	mls := snap.Service
	role := mls.Spec.Roles[0]

	dep := findDeployment(snap.Children, mls.Name, mls.Namespace)
	svc := findService(snap.Children, mls.Name, mls.Namespace)
	route := findHTTPRoute(snap.Children, mls.Name, mls.Namespace)

	desired := role.Replicas
	var ready int32
	if dep != nil {
		ready = dep.Status.ReadyReplicas
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

	upd.Phase, upd.Message = derivePhase(dep, desired, ready)
	upd.Endpoint = deriveEndpoint(mls, role, svc, route)

	if mls.Spec.Route != nil && mls.Spec.Route.Enabled {
		if !routeAccepted(route) && upd.Phase != axisml.PhaseFailed {
			// External entrypoint failed to come up: degrade and explain.
			// A terminal Deployment failure outranks a routing problem — do
			// not downgrade Failed → Degraded, or callers will miss the
			// fact that the rollout itself is broken.
			upd.Phase = axisml.PhaseDegraded
			if upd.Message == "" {
				upd.Message = "HTTPRoute not yet Accepted; falling back to in-cluster DNS"
			}
		}
	}

	upd.Conditions = buildConditions(upd.Phase, upd.Message, mls.Generation)
	return upd
}

func derivePhase(dep *appsv1.Deployment, desired, ready int32) (axisml.Phase, string) {
	if desired == 0 {
		return axisml.PhasePending, "spec.roles[0].replicas is 0"
	}
	if dep == nil {
		return axisml.PhasePending, "Deployment not yet created"
	}
	if ready == desired {
		return axisml.PhaseReady, ""
	}
	if ready > 0 && ready < desired {
		return axisml.PhaseDegraded,
			fmt.Sprintf("only %d/%d replicas ready", ready, desired)
	}
	// ready == 0, desired > 0 — distinguish "still progressing" from "failed"
	// by looking at the Deployment's standard conditions.
	for _, c := range dep.Status.Conditions {
		if c.Type == appsv1.DeploymentReplicaFailure && c.Status == corev1.ConditionTrue {
			return axisml.PhaseFailed, fmt.Sprintf("ReplicaFailure: %s", c.Message)
		}
		if c.Type == appsv1.DeploymentProgressing && c.Status == corev1.ConditionFalse &&
			c.Reason == "ProgressDeadlineExceeded" {
			return axisml.PhaseFailed, "Deployment progress deadline exceeded"
		}
	}
	return axisml.PhasePending, "Deployment is rolling out"
}

func deriveEndpoint(mls *axisml.MLService, role axisml.RoleSpec, svc *corev1.Service, route *gwapiv1.HTTPRoute) string {
	port := pickEndpointPort(role)
	internal := fmt.Sprintf(internalEndpointFmt, mls.Name, mls.Namespace, port)
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

// pickEndpointPort applies §4 endpoint port selection: prefer name=http,
// otherwise the first port. (When the fallback fires the dispatcher would
// also stamp a warning condition; deferred to keep MVP terse.)
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

func findDeployment(children []client.Object, name, namespace string) *appsv1.Deployment {
	for _, c := range children {
		if d, ok := c.(*appsv1.Deployment); ok && d.Name == name && d.Namespace == namespace {
			return d
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
