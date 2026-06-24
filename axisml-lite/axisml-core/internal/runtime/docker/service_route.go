package docker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"sigs.k8s.io/yaml"

	mlservicev1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mlservice/v1alpha1"
)

// applyServiceRoute renders (or removes) the Traefik file-provider config for an
// MLService's own spec.route: an HTTP router for the endpoint pointing at a
// loadBalancer over the service's replica containers. This is how a single
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
		return capabilityError("MLService route auth %q is unsupported in Lite", route.Auth.Type)
	}
	if route.RateLimit != nil {
		return capabilityError("MLService route rateLimit is unsupported in Lite")
	}

	port := resolveServicePort(svc, route)
	servers := make([]map[string]any, 0, len(plans))
	for i := range plans {
		servers = append(servers, map[string]any{"url": fmt.Sprintf("http://%s:%d", plans[i].Name, port)})
	}

	name := r.serviceResourceName(svc.Namespace, svc.Name)
	cfg := map[string]any{
		"http": map[string]any{
			"routers": map[string]any{
				name: map[string]any{
					"service":     name,
					"entryPoints": []string{"web"},
					"rule":        serviceRouteRule(route),
				},
			},
			"services": map[string]any{
				name: map[string]any{
					"loadBalancer": map[string]any{"servers": servers},
				},
			},
		},
		"x-axisml-endpoint": serviceRouteEndpointValue(route),
	}
	b, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal service route config: %w", err)
	}
	return r.writeTraefikFile(r.serviceRouteFileName(svc.Namespace, svc.Name), b)
}

// deleteServiceRoute removes the service's Traefik route config. Idempotent.
func (r *Runtime) deleteServiceRoute(namespace, name string) error {
	if err := os.Remove(r.serviceRouteFileName(namespace, name)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// serviceRouteEndpoint returns the configured endpoint for a service's route, or
// "" when no route is configured. Read back from the route file so Observe can
// surface it without the spec.
func (r *Runtime) serviceRouteEndpoint(namespace, name string) string {
	b, err := os.ReadFile(r.serviceRouteFileName(namespace, name))
	if err != nil {
		return ""
	}
	var doc struct {
		Endpoint string `json:"x-axisml-endpoint"`
	}
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return ""
	}
	return doc.Endpoint
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

func serviceRouteRule(route *mlservicev1alpha1.Route) string {
	var parts []string
	if route.Hostname != "" {
		parts = append(parts, fmt.Sprintf("Host(`%s`)", route.Hostname))
	}
	path := route.Path
	if path == "" {
		path = "/"
	}
	parts = append(parts, fmt.Sprintf("PathPrefix(`%s`)", path))
	return strings.Join(parts, " && ")
}

func serviceRouteEndpointValue(route *mlservicev1alpha1.Route) string {
	path := route.Path
	if path == "" {
		path = "/"
	}
	if route.Hostname != "" {
		return "http://" + route.Hostname + path
	}
	return path
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
	return filepath.Join(r.cfg.TraefikDir, fmt.Sprintf("svc-%s-%s.yaml", namespace, name))
}
