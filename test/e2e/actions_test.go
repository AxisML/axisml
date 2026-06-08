//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"net/http"

	corev1 "k8s.io/api/core/v1"

	mljobv1 "github.com/axisml/axisml/components/compute-operator/api/mljob/v1alpha1"
	mlservicev1 "github.com/axisml/axisml/components/compute-operator/api/mlservice/v1alpha1"
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
