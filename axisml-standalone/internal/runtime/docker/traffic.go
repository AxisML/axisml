package docker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	eventsv1 "k8s.io/api/events/v1"
	"k8s.io/apimachinery/pkg/types"

	mltp "github.com/axisml/axisml/axisml-system/apis/mltrafficpolicy/v1alpha1"
)

// ApplyMLTrafficPolicy renders Envoy Gateway file-provider resources for the
// policy: one Backend per member MLService and a weighted HTTPRoute over those
// Backends (design §6.3). Only native/httproute is supported.
func (r *Runtime) ApplyMLTrafficPolicy(ctx context.Context, desired *mltp.MLTrafficPolicy) error {
	if desired.Spec.Backend.Name != "native" || desired.Spec.Backend.Engine != "httproute" {
		return capabilityError("MLTrafficPolicy backend %s/%s is unsupported in standalone (only native/httproute)",
			desired.Spec.Backend.Name, desired.Spec.Backend.Engine)
	}
	resources, err := r.renderGatewayTraffic(ctx, desired)
	if err != nil {
		return err
	}
	b, err := marshalGatewayResources(resources...)
	if err != nil {
		return fmt.Errorf("marshal traffic gateway resources: %w", err)
	}
	if err := r.writeGatewayFile(r.trafficFileName(desired.Namespace, desired.Name), b); err != nil {
		return err
	}
	r.events.record(KindTraffic, desired.Namespace, desired.Name, "", "Applied", "traffic policy config written")
	return nil
}

// ObserveMLTrafficPolicy reports the policy status from the gateway resource file
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

// DeleteMLTrafficPolicy removes the policy's gateway resource file. Idempotent.
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

// --- Envoy Gateway rendering helpers ---

type trafficMember struct {
	service string
	weight  int32
	path    string
	host    string
}

type trafficBackend struct {
	serviceName  string
	resourceName string
	endpoints    []gatewayEndpoint
	weight       int32
}

func (r *Runtime) renderGatewayTraffic(ctx context.Context, p *mltp.MLTrafficPolicy) ([]gatewayResource, error) {
	ns := p.Namespace
	backends := make([]trafficBackend, 0, len(p.Spec.Backends))
	for _, m := range p.Spec.Backends {
		endpoints, err := r.memberEndpoints(ctx, ns, m.ServiceName)
		if err != nil {
			return nil, err
		}
		backends = append(backends, trafficBackend{
			serviceName:  m.ServiceName,
			resourceName: r.trafficMemberResourceName(ns, p.Name, m.ServiceName),
			endpoints:    endpoints,
			weight:       m.Weight,
		})
	}
	return r.gatewayTrafficResources(p, backends), nil
}

func (r *Runtime) gatewayTrafficResources(p *mltp.MLTrafficPolicy, backends []trafficBackend) []gatewayResource {
	resources := make([]gatewayResource, 0, len(backends)+1)
	refs := make([]gatewayBackendRef, 0, len(backends))
	for _, backend := range backends {
		resources = append(resources, gatewayBackend(
			backend.resourceName, backend.serviceName, backend.endpoints,
		))
		weight := backend.weight
		refs = append(refs, gatewayBackendRef{
			Group:  gatewayBackendGroup,
			Kind:   "Backend",
			Name:   backend.resourceName,
			Weight: &weight,
		})
	}
	resources = append(resources, gatewayHTTPRoute(
		r.trafficResourceName(p.Namespace, p.Name),
		p.Spec.Endpoint.Hostname,
		p.Spec.Endpoint.Path,
		refs,
	))
	return resources
}

// memberEndpoints builds Envoy Gateway Backend FQDN endpoints for a member
// service's running replicas. Each FQDN is the multi-label Docker network alias
// installed by ContainerPlan.toDocker.
func (r *Runtime) memberEndpoints(ctx context.Context, namespace, serviceName string) ([]gatewayEndpoint, error) {
	conts, err := r.listContainers(ctx, KindService, namespace, serviceName)
	if err != nil {
		return nil, err
	}
	var endpoints []gatewayEndpoint
	for _, c := range conts {
		// listContainers returns All:true, including stopped/exited replicas.
		// Only route to running ones — a backend URL pointing at a stopped
		// container would make Envoy forward requests to a dead backend during
		// a replica restart or rolling replace.
		if c.State != "running" {
			continue
		}
		name := summaryName(c)
		port := r.firstExposedPort(ctx, c.ID)
		endpoints = append(endpoints, gatewayContainerEndpoint(name, port))
	}
	if endpoints == nil {
		endpoints = []gatewayEndpoint{}
	}
	return endpoints, nil
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
	docs, err := readGatewayDocuments(path)
	if err != nil {
		return nil, err
	}
	services := map[string]string{}
	var route *gatewayFileDocument
	for i := range docs {
		doc := &docs[i]
		switch doc.Kind {
		case "Backend":
			services[doc.Metadata.Name] = doc.Metadata.Annotations[gatewayServiceAnnotation]
		case "HTTPRoute":
			route = doc
		}
	}
	if route == nil || len(route.Spec.Rules) == 0 {
		return nil, nil
	}
	routePath := "/"
	if len(route.Spec.Rules[0].Matches) > 0 && route.Spec.Rules[0].Matches[0].Path.Value != "" {
		routePath = route.Spec.Rules[0].Matches[0].Path.Value
	}
	host := ""
	if len(route.Spec.Hostnames) > 0 {
		host = route.Spec.Hostnames[0]
	}
	out := make([]trafficMember, 0, len(route.Spec.Rules[0].BackendRefs))
	for _, ref := range route.Spec.Rules[0].BackendRefs {
		weight := int32(1)
		if ref.Weight != nil {
			weight = *ref.Weight
		}
		out = append(out, trafficMember{
			service: services[ref.Name],
			weight:  weight,
			path:    routePath,
			host:    host,
		})
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

func (r *Runtime) trafficMemberResourceName(namespace, policyName, serviceName string) string {
	raw := fmt.Sprintf("axisml-tp-%s-%s-%s", namespace, policyName, serviceName)
	clean := nameSanitizer.ReplaceAllString(raw, "-")
	if clean == raw && len(clean) <= 100 {
		return clean
	}
	return fmt.Sprintf("axisml-tp-backend-%s", shortHash(raw))
}

func (r *Runtime) trafficFileName(namespace, name string) string {
	return filepath.Join(r.cfg.GatewayConfigDir, fmt.Sprintf("traffic-policy-%s-%s.yaml", namespace, name))
}
