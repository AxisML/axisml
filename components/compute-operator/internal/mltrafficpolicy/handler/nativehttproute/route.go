package nativehttproute

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	mltp "github.com/axisml/axisml/components/compute-operator/api/mltrafficpolicy/v1alpha1"
)

// buildHTTPRoute renders the single weighted HTTPRoute fronting all member
// services. backendRefs (with per-member weight + resolved port) are supplied
// by Reconcile.
func buildHTTPRoute(p *mltp.MLTrafficPolicy, backendRefs []gwapiv1.HTTPBackendRef) *gwapiv1.HTTPRoute {
	parentNS := gwapiv1.Namespace(GatewayParentNamespace)
	parentName := gwapiv1.ObjectName(GatewayParentName)

	pathPrefix := "/"
	if p.Spec.Endpoint.Path != "" {
		pathPrefix = p.Spec.Endpoint.Path
	}
	pathType := gwapiv1.PathMatchPathPrefix

	rule := gwapiv1.HTTPRouteRule{
		Matches: []gwapiv1.HTTPRouteMatch{{
			Path: &gwapiv1.HTTPPathMatch{Type: &pathType, Value: &pathPrefix},
		}},
		BackendRefs: backendRefs,
	}

	route := &gwapiv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.Name,
			Namespace: p.Namespace,
			Labels:    resourceLabels(p),
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
	if p.Spec.Endpoint.Hostname != "" {
		route.Spec.Hostnames = []gwapiv1.Hostname{gwapiv1.Hostname(p.Spec.Endpoint.Hostname)}
	}
	return route
}

// backendRef builds one weighted HTTPBackendRef pointing at a member's
// ClusterIP Service.
func backendRef(serviceName string, port, weight int32) gwapiv1.HTTPBackendRef {
	kind := gwapiv1.Kind("Service")
	pn := gwapiv1.PortNumber(port)
	w := weight
	return gwapiv1.HTTPBackendRef{
		BackendRef: gwapiv1.BackendRef{
			BackendObjectReference: gwapiv1.BackendObjectReference{
				Name: gwapiv1.ObjectName(serviceName),
				Kind: &kind,
				Port: &pn,
			},
			Weight: &w,
		},
	}
}

// resourceLabels stamps the derived HTTPRoute with the traffic-policy-id (so
// the dispatcher can list it as an owned child) plus the tenant label.
func resourceLabels(p *mltp.MLTrafficPolicy) map[string]string {
	labels := map[string]string{
		mltp.LabelTrafficPolicyID: p.Labels[mltp.LabelTrafficPolicyID],
	}
	if t := p.Labels[mltp.LabelTenant]; t != "" {
		labels[mltp.LabelTenant] = t
	}
	return labels
}
