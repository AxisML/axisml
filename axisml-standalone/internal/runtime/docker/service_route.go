package docker

import (
	"fmt"
	"os"
	"path/filepath"

	mlservicev1alpha1 "github.com/axisml/axisml/axisml-system/apis/mlservice/v1alpha1"
)

// applyServiceRoute renders (or removes) Envoy Gateway file-provider resources
// for an MLService's own spec.route: a Backend over the service's replica
// containers plus an HTTPRoute for the endpoint. This is how a single
// service — including kind=workspace / kind=tensorboard — is exposed through the
// gateway (design §6.3). MLTrafficPolicy handles the multi-service weighted case
// separately. Unsupported route features (auth, rate limit — Gateway API
// SecurityPolicy / BackendTrafficPolicy) surface as CapabilityError.
func (r *Runtime) applyServiceRoute(svc *mlservicev1alpha1.MLService, plans []ContainerPlan) error {
	route := svc.Spec.Route
	if route == nil || !route.Enabled {
		// No route desired: clear any stale config.
		return r.deleteServiceRoute(svc.Namespace, svc.Name)
	}
	if route.Auth != nil && route.Auth.Type != "" && route.Auth.Type != mlservicev1alpha1.RouteAuthNone {
		return capabilityError("MLService route auth %q is unsupported in standalone", route.Auth.Type)
	}
	if route.RateLimit != nil {
		return capabilityError("MLService route rateLimit is unsupported in standalone")
	}

	port := resolveServicePort(svc, route)
	endpoints := make([]gatewayEndpoint, 0, len(plans))
	for i := range plans {
		endpoints = append(endpoints, gatewayContainerEndpoint(plans[i].Name, port))
	}

	name := r.serviceResourceName(svc.Namespace, svc.Name)
	b, err := marshalGatewayResources(
		gatewayBackend(name, svc.Name, endpoints),
		gatewayHTTPRoute(name, route.Hostname, route.Path, []gatewayBackendRef{{
			Group: gatewayBackendGroup,
			Kind:  "Backend",
			Name:  name,
		}}),
	)
	if err != nil {
		return fmt.Errorf("marshal service gateway resources: %w", err)
	}
	return r.writeGatewayFile(r.serviceRouteFileName(svc.Namespace, svc.Name), b)
}

// deleteServiceRoute removes the service's gateway resources. Idempotent.
func (r *Runtime) deleteServiceRoute(namespace, name string) error {
	if err := os.Remove(r.serviceRouteFileName(namespace, name)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// serviceRouteEndpoint returns the configured endpoint for a service's HTTPRoute,
// or "" when no route is configured. Read back from the route file so Observe
// can surface it without the spec.
func (r *Runtime) serviceRouteEndpoint(namespace, name string) string {
	docs, err := readGatewayDocuments(r.serviceRouteFileName(namespace, name))
	if err != nil {
		return ""
	}
	for _, doc := range docs {
		if doc.Kind == "HTTPRoute" {
			return gatewayRouteEndpoint(doc)
		}
	}
	return ""
}

// resolveServicePort picks the container port the route targets: the named port
// on the target role, else the role's first port, else 80.
func resolveServicePort(svc *mlservicev1alpha1.MLService, route *mlservicev1alpha1.Route) int {
	if len(svc.Spec.Roles) == 0 {
		return 80
	}
	ports := svc.Spec.Roles[0].Template.Ports
	if route.PortName != "" {
		for _, p := range ports {
			if p.Name == route.PortName {
				return int(p.ContainerPort)
			}
		}
	}
	if len(ports) > 0 {
		return int(ports[0].ContainerPort)
	}
	return 80
}

func (r *Runtime) serviceResourceName(namespace, name string) string {
	raw := fmt.Sprintf("axisml-svc-%s-%s", namespace, name)
	clean := nameSanitizer.ReplaceAllString(raw, "-")
	if clean == raw && len(clean) <= 100 {
		return clean
	}
	return fmt.Sprintf("axisml-svc-%s", shortHash(raw))
}

func (r *Runtime) serviceRouteFileName(namespace, name string) string {
	return filepath.Join(r.cfg.GatewayConfigDir, fmt.Sprintf("service-%s-%s.yaml", namespace, name))
}
