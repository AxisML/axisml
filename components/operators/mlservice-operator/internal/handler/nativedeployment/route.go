package nativedeployment

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	axisml "github.com/axisml/axisml/components/operators/mlservice-operator/api/v1alpha1"
)

// buildHTTPRoute renders the Gateway API HTTPRoute that fronts the MLService
// when spec.route.enabled=true. Auth / rate limit / timeout policies require
// Envoy Gateway-specific CRDs and are deferred (Validate emits a warning).
//
// Cross-namespace attachment to axisml-infra/axisml-gateway is permitted by
// the shared Gateway's allowedRoutes.namespaces.from=All (see
// axisml-infra/templates/gateway.yaml). No ReferenceGrant is required because
// BackendRef is in-namespace.
func buildHTTPRoute(mls *axisml.MLService) *gwapiv1.HTTPRoute {
	role := mls.Spec.Roles[0]
	r := mls.Spec.Route

	parentNS := gwapiv1.Namespace(GatewayParentNamespace)
	parentName := gwapiv1.ObjectName(GatewayParentName)

	pathPrefix := "/"
	if r.Path != "" {
		pathPrefix = r.Path
	}
	pathType := gwapiv1.PathMatchPathPrefix

	port := pickRoutePort(role, r.PortName)
	backendKind := gwapiv1.Kind("Service")

	rule := gwapiv1.HTTPRouteRule{
		Matches: []gwapiv1.HTTPRouteMatch{{
			Path: &gwapiv1.HTTPPathMatch{Type: &pathType, Value: &pathPrefix},
		}},
		BackendRefs: []gwapiv1.HTTPBackendRef{{
			BackendRef: gwapiv1.BackendRef{
				BackendObjectReference: gwapiv1.BackendObjectReference{
					Name: gwapiv1.ObjectName(mls.Name),
					Kind: &backendKind,
					Port: &port,
				},
			},
		}},
	}

	route := &gwapiv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mls.Name,
			Namespace: mls.Namespace,
			Labels:    resourceLabels(mls, role.Name),
		},
		Spec: gwapiv1.HTTPRouteSpec{
			CommonRouteSpec: gwapiv1.CommonRouteSpec{
				ParentRefs: []gwapiv1.ParentReference{{
					Name:      parentName,
					Namespace: &parentNS,
				}},
			},
			Rules: []gwapiv1.HTTPRouteRule{rule},
		},
	}
	if r.Hostname != "" {
		route.Spec.Hostnames = []gwapiv1.Hostname{gwapiv1.Hostname(r.Hostname)}
	}
	return route
}

// pickRoutePort resolves spec.route.portName → containerPort. Single-port
// roles fall back to the only port; multi-port roles must specify portName
// (Validate enforces this).
func pickRoutePort(role axisml.RoleSpec, portName string) gwapiv1.PortNumber {
	if portName != "" {
		for _, p := range role.Template.Ports {
			if p.Name == portName {
				return gwapiv1.PortNumber(p.ContainerPort)
			}
		}
	}
	if len(role.Template.Ports) > 0 {
		return gwapiv1.PortNumber(role.Template.Ports[0].ContainerPort)
	}
	return 0
}
