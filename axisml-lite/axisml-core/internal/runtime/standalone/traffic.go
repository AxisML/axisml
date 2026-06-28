package standalone

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	eventsv1 "k8s.io/api/events/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"

	mltp "github.com/axisml/axisml/components/compute-operator/api/mltrafficpolicy/v1alpha1"
)

// ApplyMLTrafficPolicy renders a Traefik file-provider dynamic config for the
// policy: an HTTP router for the endpoint pointing at a weighted service whose
// members are the per-MLService load balancers (design §6.3). The file is
// written atomically (temp + rename). Only native/httproute is supported.
func (r *Runtime) ApplyMLTrafficPolicy(ctx context.Context, desired *mltp.MLTrafficPolicy) error {
	if desired.Spec.Backend.Name != "native" || desired.Spec.Backend.Engine != "httproute" {
		return capabilityError("MLTrafficPolicy backend %s/%s is unsupported in Lite (only native/httproute)",
			desired.Spec.Backend.Name, desired.Spec.Backend.Engine)
	}
	cfg, err := r.renderTraefik(ctx, desired)
	if err != nil {
		return err
	}
	b, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal traefik config: %w", err)
	}
	if err := r.writeTraefikFile(r.trafficFileName(desired.Namespace, desired.Name), b); err != nil {
		return err
	}
	r.events.record(KindTraffic, desired.Namespace, desired.Name, "", "Applied", "traffic policy config written")
	return nil
}

// ObserveMLTrafficPolicy reports the policy status from the Traefik file
// presence and the member services' readiness. A missing file is NotFound.
func (r *Runtime) ObserveMLTrafficPolicy(ctx context.Context, key types.NamespacedName) (mltp.MLTrafficPolicyStatus, error) {
	path := r.trafficFileName(key.Namespace, key.Name)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return mltp.MLTrafficPolicyStatus{}, notFound("mltrafficpolicies", key)
		}
		return mltp.MLTrafficPolicyStatus{}, err
	}
	members, err := r.readTrafficMembers(path)
	if err != nil {
		return mltp.MLTrafficPolicyStatus{}, err
	}

	status := mltp.MLTrafficPolicyStatus{}
	readyCount, active := 0, 0
	for _, m := range members {
		ready := r.serviceHasReady(ctx, key.Namespace, m.service)
		status.Backends = append(status.Backends, mltp.BackendStatus{
			ServiceName: m.service, Weight: m.weight, Ready: ready,
		})
		if m.weight > 0 {
			active++
			if ready {
				readyCount++
			}
		}
	}
	switch {
	case active == 0:
		status.Phase = mltp.PhasePending
	case readyCount == active:
		status.Phase = mltp.PhaseReady
	case readyCount > 0:
		status.Phase = mltp.PhaseDegraded
	default:
		status.Phase = mltp.PhasePending
	}
	status.Endpoint = r.trafficEndpoint(members)
	return status, nil
}

// DeleteMLTrafficPolicy removes the policy's Traefik config file. Idempotent.
func (r *Runtime) DeleteMLTrafficPolicy(_ context.Context, key types.NamespacedName) error {
	path := r.trafficFileName(key.Namespace, key.Name)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	r.events.record(KindTraffic, key.Namespace, key.Name, "", "Deleted", "traffic policy config removed")
	return nil
}

// GetMLTrafficPolicyEvents returns resource-level runtime events for the policy.
func (r *Runtime) GetMLTrafficPolicyEvents(_ context.Context, key types.NamespacedName) (*eventsv1.EventList, error) {
	return r.events.list(KindTraffic, key.Namespace, key.Name, ""), nil
}

// --- Traefik rendering helpers ---

type trafficMember struct {
	service string
	weight  int32
	path    string
	host    string
}

func (r *Runtime) renderTraefik(ctx context.Context, p *mltp.MLTrafficPolicy) (map[string]any, error) {
	ns := p.Namespace
	routerName := r.trafficResourceName(ns, p.Name)
	weightedName := routerName

	services := map[string]any{}
	var weightedServices []map[string]any
	for _, m := range p.Spec.Backends {
		svcName := r.trafficResourceName(ns, m.ServiceName)
		servers, err := r.memberServers(ctx, ns, m.ServiceName)
		if err != nil {
			return nil, err
		}
		services[svcName] = map[string]any{
			"loadBalancer": map[string]any{"servers": servers},
		}
		weightedServices = append(weightedServices, map[string]any{
			"name":   svcName,
			"weight": m.Weight,
		})
	}
	services[weightedName] = map[string]any{
		"weighted": map[string]any{"services": weightedServices},
	}

	router := map[string]any{
		"service":     weightedName,
		"entryPoints": []string{"web"},
		"rule":        trafficRule(p.Spec.Endpoint),
	}
	return map[string]any{
		"http": map[string]any{
			"routers":  map[string]any{routerName: router},
			"services": services,
		},
		// Annotation comment for member→weight readback on Observe.
		"x-axisml-members": trafficMembersAnnotation(p),
	}, nil
}

func trafficRule(ep mltp.Endpoint) string {
	var parts []string
	if ep.Hostname != "" {
		parts = append(parts, fmt.Sprintf("Host(`%s`)", ep.Hostname))
	}
	path := ep.Path
	if path == "" {
		path = "/"
	}
	parts = append(parts, fmt.Sprintf("PathPrefix(`%s`)", path))
	return strings.Join(parts, " && ")
}

func trafficMembersAnnotation(p *mltp.MLTrafficPolicy) []map[string]any {
	out := make([]map[string]any, 0, len(p.Spec.Backends))
	for _, m := range p.Spec.Backends {
		out = append(out, map[string]any{
			"service": m.ServiceName,
			"weight":  m.Weight,
			"path":    p.Spec.Endpoint.Path,
			"host":    p.Spec.Endpoint.Hostname,
		})
	}
	return out
}

// memberServers builds Traefik loadBalancer server URLs for a member service's
// running replicas (DNS = container name on the workloads network).
func (r *Runtime) memberServers(ctx context.Context, namespace, serviceName string) ([]map[string]any, error) {
	conts, err := r.listContainers(ctx, KindService, namespace, serviceName)
	if err != nil {
		return nil, err
	}
	var servers []map[string]any
	for _, c := range conts {
		// listContainers returns All:true, including stopped/exited replicas.
		// Only route to running ones — a backend URL pointing at a stopped
		// container would make Traefik forward requests to a dead backend during
		// a replica restart or rolling replace.
		if c.State != "running" {
			continue
		}
		name := summaryName(c)
		port := r.firstExposedPort(ctx, c.ID)
		servers = append(servers, map[string]any{
			"url": fmt.Sprintf("http://%s:%d", name, port),
		})
	}
	if servers == nil {
		servers = []map[string]any{}
	}
	return servers, nil
}

func (r *Runtime) firstExposedPort(ctx context.Context, id string) int {
	ins, err := r.cli.ContainerInspect(ctx, id)
	if err != nil || ins.Config == nil || len(ins.Config.ExposedPorts) == 0 {
		return 80
	}
	ports := make([]int, 0, len(ins.Config.ExposedPorts))
	for p := range ins.Config.ExposedPorts {
		ports = append(ports, p.Int())
	}
	sort.Ints(ports)
	if len(ports) == 0 {
		return 80
	}
	return ports[0]
}

func (r *Runtime) serviceHasReady(ctx context.Context, namespace, serviceName string) bool {
	conts, err := r.listContainers(ctx, KindService, namespace, serviceName)
	if err != nil {
		return false
	}
	states, err := r.inspectAll(ctx, conts)
	if err != nil {
		return false
	}
	for i := range states {
		if states[i].ready() {
			return true
		}
	}
	return false
}

func (r *Runtime) readTrafficMembers(path string) ([]trafficMember, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Members []struct {
			Service string `json:"service"`
			Weight  int32  `json:"weight"`
			Path    string `json:"path"`
			Host    string `json:"host"`
		} `json:"x-axisml-members"`
	}
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return nil, err
	}
	out := make([]trafficMember, 0, len(doc.Members))
	for _, m := range doc.Members {
		out = append(out, trafficMember{service: m.Service, weight: m.Weight, path: m.Path, host: m.Host})
	}
	return out, nil
}

func (r *Runtime) trafficEndpoint(members []trafficMember) string {
	if len(members) == 0 {
		return ""
	}
	if members[0].host != "" {
		return "http://" + members[0].host + members[0].path
	}
	return members[0].path
}

func (r *Runtime) trafficResourceName(namespace, name string) string {
	raw := fmt.Sprintf("axisml-%s-%s", namespace, name)
	clean := nameSanitizer.ReplaceAllString(raw, "-")
	if clean == raw && len(clean) <= 100 {
		return clean
	}
	return fmt.Sprintf("axisml-tp-%s", shortHash(raw))
}

func (r *Runtime) trafficFileName(namespace, name string) string {
	return filepath.Join(r.cfg.TraefikDir, fmt.Sprintf("tp-%s-%s.yaml", namespace, name))
}

func (r *Runtime) writeTraefikFile(path string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
