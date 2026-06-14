//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"net/http"

	corev1 "k8s.io/api/core/v1"

	mlrunv1 "github.com/axisml/axisml/components/compute-operator/api/mlrun/v1alpha1"
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

// ----- compute-service: mlruns -----

func mlrunsPath(ns string) string { return fmt.Sprintf("/api/v1/namespaces/%s/mlruns", ns) }
func mlrunPath(ns, name string) string {
	return fmt.Sprintf("/api/v1/namespaces/%s/mlruns/%s", ns, name)
}

func (s *suite) createMLRun(ctx context.Context, ns string, req csCreateMLRunReq) (resp, error) {
	return s.computeService.do(ctx, http.MethodPost, mlrunsPath(ns), req)
}

func (s *suite) getMLRun(ctx context.Context, ns, name string) (resp, error) {
	return s.computeService.do(ctx, http.MethodGet, mlrunPath(ns, name), nil)
}

func (s *suite) cancelMLRun(ctx context.Context, ns, name string) (resp, error) {
	return s.computeService.do(ctx, http.MethodPost, mlrunPath(ns, name)+"/cancel", nil)
}

func (s *suite) deleteMLRun(ctx context.Context, ns, name string) (resp, error) {
	return s.computeService.do(ctx, http.MethodDelete, mlrunPath(ns, name), nil)
}

// ----- compute-service: mlservices -----

func mlservicesPath(ns string) string { return fmt.Sprintf("/api/v1/namespaces/%s/mlservices", ns) }
func mlservicePath(ns, name string) string {
	return fmt.Sprintf("/api/v1/namespaces/%s/mlservices/%s", ns, name)
}

func (s *suite) createMLService(ctx context.Context, ns string, req csCreateMLServiceReq) (resp, error) {
	return s.computeService.do(ctx, http.MethodPost, mlservicesPath(ns), req)
}

func (s *suite) getMLService(ctx context.Context, ns, name string) (resp, error) {
	return s.computeService.do(ctx, http.MethodGet, mlservicePath(ns, name), nil)
}

func (s *suite) scaleMLService(ctx context.Context, ns, name string, replicas int32) (resp, error) {
	return s.computeService.do(ctx, http.MethodPost, mlservicePath(ns, name)+"/scale", csScaleReq{Replicas: replicas})
}

func (s *suite) deleteMLService(ctx context.Context, ns, name string) (resp, error) {
	return s.computeService.do(ctx, http.MethodDelete, mlservicePath(ns, name), nil)
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

// busyboxMLRunReq builds a minimal native/job MLRun create request that runs to
// completion quickly.
func busyboxMLRunReq(name, pool, unit, quota string) csCreateMLRunReq {
	return csCreateMLRunReq{
		Name:     name,
		PoolName: pool,
		UnitName: unit,
		Quota:    quota,
		Backend:  &mlrunv1.BackendSpec{Name: "native", Engine: "job"},
		Roles: []mlrunv1.RoleSpec{{
			Name:          mlrunv1.DefaultRoleName,
			Replicas:      1,
			RestartPolicy: corev1.RestartPolicyNever,
			Template: mlrunv1.PodTemplateSubset{
				Image:           h.cfg.MLRunImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command:         []string{"sh", "-c", "echo hello"},
			},
		}},
	}
}

// nginxMLServiceReq builds a minimal native/deployment MLService create request.
func nginxMLServiceReq(name, pool, unit, quota string, route *mlservicev1.Route) csCreateMLServiceReq {
	return csCreateMLServiceReq{
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
