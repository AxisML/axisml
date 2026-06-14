//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"net/http"

	corev1 "k8s.io/api/core/v1"

	mljobv1 "github.com/axisml/axisml/components/compute-operator/api/mljob/v1alpha1"
	mlservicev1 "github.com/axisml/axisml/components/compute-operator/api/mlservice/v1alpha1"
	mltpv1 "github.com/axisml/axisml/components/compute-operator/api/mltrafficpolicy/v1alpha1"
)

// Reusable client actions shared across the layer test files. Each returns the
// raw resp so callers decide how strictly to assert.

// ----- compute-service: tenants -----

func (s *suite) createTenant(ctx context.Context, req csCreateTenantReq) (resp, error) {
	return s.computeService.do(ctx, http.MethodPost, "/api/v1/namespaces", req)
}

func (s *suite) getTenant(ctx context.Context, name string) (resp, error) {
	return s.computeService.do(ctx, http.MethodGet, "/api/v1/namespaces/"+name, nil)
}

func (s *suite) deleteTenant(ctx context.Context, name string) (resp, error) {
	return s.computeService.do(ctx, http.MethodDelete, "/api/v1/namespaces/"+name, nil)
}

// ----- compute-service: jobs -----

func jobsPath(ns string) string { return fmt.Sprintf("/api/v1/namespaces/%s/jobs", ns) }
func jobPath(ns, name string) string {
	return fmt.Sprintf("/api/v1/namespaces/%s/jobs/%s", ns, name)
}

func (s *suite) createJob(ctx context.Context, ns string, req csCreateJobReq) (resp, error) {
	return s.computeService.do(ctx, http.MethodPost, jobsPath(ns), req)
}

func (s *suite) getJob(ctx context.Context, ns, name string) (resp, error) {
	return s.computeService.do(ctx, http.MethodGet, jobPath(ns, name), nil)
}

func (s *suite) cancelJob(ctx context.Context, ns, name string) (resp, error) {
	return s.computeService.do(ctx, http.MethodPost, jobPath(ns, name)+"/cancel", nil)
}

func (s *suite) deleteJob(ctx context.Context, ns, name string) (resp, error) {
	return s.computeService.do(ctx, http.MethodDelete, jobPath(ns, name), nil)
}

// ----- compute-service: services -----

func servicesPath(ns string) string { return fmt.Sprintf("/api/v1/namespaces/%s/services", ns) }
func servicePath(ns, name string) string {
	return fmt.Sprintf("/api/v1/namespaces/%s/services/%s", ns, name)
}

func (s *suite) createService(ctx context.Context, ns string, req csCreateServiceReq) (resp, error) {
	return s.computeService.do(ctx, http.MethodPost, servicesPath(ns), req)
}

func (s *suite) getService(ctx context.Context, ns, name string) (resp, error) {
	return s.computeService.do(ctx, http.MethodGet, servicePath(ns, name), nil)
}

func (s *suite) scaleService(ctx context.Context, ns, name string, replicas int32) (resp, error) {
	return s.computeService.do(ctx, http.MethodPost, servicePath(ns, name)+"/scale", csScaleReq{Replicas: replicas})
}

func (s *suite) deleteService(ctx context.Context, ns, name string) (resp, error) {
	return s.computeService.do(ctx, http.MethodDelete, servicePath(ns, name), nil)
}

// ----- compute-service: traffic policies -----

func trafficPoliciesPath(ns string) string {
	return fmt.Sprintf("/api/v1/namespaces/%s/traffic-policies", ns)
}
func trafficPolicyPath(ns, name string) string {
	return fmt.Sprintf("/api/v1/namespaces/%s/traffic-policies/%s", ns, name)
}

func (s *suite) createTrafficPolicy(ctx context.Context, ns string, req csCreateTrafficPolicyReq) (resp, error) {
	return s.computeService.do(ctx, http.MethodPost, trafficPoliciesPath(ns), req)
}

func (s *suite) getTrafficPolicy(ctx context.Context, ns, name string) (resp, error) {
	return s.computeService.do(ctx, http.MethodGet, trafficPolicyPath(ns, name), nil)
}

func (s *suite) splitTrafficPolicy(ctx context.Context, ns, name string, req csTrafficSplitReq) (resp, error) {
	return s.computeService.do(ctx, http.MethodPost, trafficPolicyPath(ns, name)+"/split", req)
}

func (s *suite) promoteTrafficPolicy(ctx context.Context, ns, name string) (resp, error) {
	return s.computeService.do(ctx, http.MethodPost, trafficPolicyPath(ns, name)+"/promote", nil)
}

func (s *suite) rollbackTrafficPolicy(ctx context.Context, ns, name string) (resp, error) {
	return s.computeService.do(ctx, http.MethodPost, trafficPolicyPath(ns, name)+"/rollback", nil)
}

func (s *suite) deleteTrafficPolicy(ctx context.Context, ns, name string) (resp, error) {
	return s.computeService.do(ctx, http.MethodDelete, trafficPolicyPath(ns, name), nil)
}

// canaryTrafficReq builds a canary traffic policy fronting two member services
// (stable @90 / canary @10).
func canaryTrafficReq(name, stableSvc, canarySvc string) csCreateTrafficPolicyReq {
	return csCreateTrafficPolicyReq{
		Name:     name,
		Mode:     string(mltpv1.TrafficModeCanary),
		Endpoint: mltpv1.Endpoint{Hostname: name + ".e2e.local"},
		Backends: []mltpv1.BackendMember{
			{ServiceName: stableSvc, Role: mltpv1.RoleStable, Weight: 90},
			{ServiceName: canarySvc, Role: mltpv1.RoleCanary, Weight: 10},
		},
	}
}

// ----- request builders -----

// busyboxJobReq builds a minimal native/job MLJob create request that runs to
// completion quickly.
func busyboxJobReq(name, pool, unit, quota string) csCreateJobReq {
	return csCreateJobReq{
		Name:     name,
		PoolName: pool,
		UnitName: unit,
		Quota:    quota,
		Backend:  &mljobv1.BackendSpec{Name: "native", Engine: "job"},
		Roles: []mljobv1.RoleSpec{{
			Name:          mljobv1.DefaultRoleName,
			Replicas:      1,
			RestartPolicy: corev1.RestartPolicyNever,
			Template: mljobv1.PodTemplateSubset{
				Image:           h.cfg.JobImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command:         []string{"sh", "-c", "echo hello"},
			},
		}},
	}
}

// nginxServiceReq builds a minimal native/deployment MLService create request.
func nginxServiceReq(name, pool, unit, quota string, route *mlservicev1.Route) csCreateServiceReq {
	return csCreateServiceReq{
		Name:     name,
		Kind:     mlservicev1.ServiceKindService,
		PoolName: pool,
		UnitName: unit,
		Quota:    quota,
		Backend:  &mlservicev1.Backend{Name: "native", Engine: "deployment"},
		Roles: []mlservicev1.RoleSpec{{
			Name:     mlservicev1.DefaultRoleName,
			Replicas: 1,
			Template: mlservicev1.PodTemplate{
				Image:           h.cfg.ServiceImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Ports:           []mlservicev1.PodPort{{Name: "http", ContainerPort: 80, Protocol: corev1.ProtocolTCP}},
			},
		}},
		Route: route,
	}
}
