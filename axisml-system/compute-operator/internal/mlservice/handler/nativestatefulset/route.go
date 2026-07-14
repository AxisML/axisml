package nativestatefulset

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	axisml "github.com/axisml/axisml/axisml-system/apis/mlservice/v1alpha1"
	"github.com/axisml/axisml/axisml-system/apis/pkg/workloadname"
)

// buildHTTPRoute renders the Gateway API HTTPRoute that fronts the MLService
// when spec.route.enabled=true. backendName is the headless Service name
// (defaultedServiceName(mls, cfg)) so route targeting follows
// backend.config.serviceName when the user overrides it.
func buildHTTPRoute(mls *axisml.MLService, backendName string) *gwapiv1.HTTPRoute {
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
					Name: gwapiv1.ObjectName(backendName),
					Kind: &backendKind,
					Port: &port,
				},
			},
		}},
	}

	route := &gwapiv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workloadname.Workload(mls),
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
