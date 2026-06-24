//go:build e2e

package e2e

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mlrunv1 "github.com/axisml/axisml/components/compute-operator/api/mlrun/v1alpha1"
	mlservicev1 "github.com/axisml/axisml/components/compute-operator/api/mlservice/v1alpha1"
	mltpv1 "github.com/axisml/axisml/components/compute-operator/api/mltrafficpolicy/v1alpha1"

	"github.com/axisml/axisml/test/e2e/internal/clients/clustermanager"
	"github.com/axisml/axisml/test/e2e/internal/clients/computeservice"
)

// Reusable client actions shared across the layer test files. Each is a thin
// wrapper over the generated, typed client and returns its typed response so
// callers can assert on StatusCode() and the JSONxxx payload.

// ----- cluster-manager: tenants -----
//
// Tenant lifecycle is owned by cluster-manager (REST writer over the Tenant CR);
// compute-service no longer serves /namespaces tenant CRUD.

func (s *suite) createTenant(ctx context.Context, body clustermanager.CreateTenantRequest) (*clustermanager.CreateTenantResponse, error) {
	return s.clusterManager.CreateTenantWithResponse(ctx, body)
}

func (s *suite) getTenant(ctx context.Context, name string) (*clustermanager.GetTenantResponse, error) {
	return s.clusterManager.GetTenantWithResponse(ctx, name)
}

func (s *suite) deleteTenant(ctx context.Context, name string) (*clustermanager.DeleteTenantResponse, error) {
	return s.clusterManager.DeleteTenantWithResponse(ctx, name)
}

// setTenantQuota creates or replaces one pool's quota for a tenant.
func (s *suite) setTenantQuota(ctx context.Context, tenant string, body clustermanager.SetQuotaRequest) (*clustermanager.SetTenantQuotaResponse, error) {
	return s.clusterManager.SetTenantQuotaWithResponse(ctx, tenant, body)
}

// ----- compute-service: mlruns -----

func (s *suite) createMLRun(ctx context.Context, ns string, body computeservice.MLRunCreateRequest) (*computeservice.CreateMLRunResponse, error) {
	return s.computeService.CreateMLRunWithResponse(ctx, ns, body)
}

func (s *suite) getMLRun(ctx context.Context, ns, name string) (*computeservice.GetMLRunResponse, error) {
	return s.computeService.GetMLRunWithResponse(ctx, ns, name)
}

func (s *suite) cancelMLRun(ctx context.Context, ns, name string) (*computeservice.CancelMLRunResponse, error) {
	return s.computeService.CancelMLRunWithResponse(ctx, ns, name)
}

func (s *suite) deleteMLRun(ctx context.Context, ns, name string) (*computeservice.DeleteMLRunResponse, error) {
	return s.computeService.DeleteMLRunWithResponse(ctx, ns, name)
}

// cleanupMLRun registers best-effort teardown for an API-created MLRun: the
// user-facing DELETE first, then a direct K8s delete as a fallback. A non-2xx
// API response (or an unreachable compute-service) would otherwise leak the CR
// and its backing workload into later tests sharing the namespace.
func cleanupMLRun(t *testing.T, ns, name string) {
	t.Helper()
	t.Cleanup(func() {
		bg := context.Background()
		_, _ = h.deleteMLRun(bg, ns, name)
		_ = h.k8s.Delete(bg, &mlrunv1.MLRun{ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name}})
	})
}

// ----- compute-service: mlservices -----

func (s *suite) createMLService(ctx context.Context, ns string, body computeservice.MLServiceCreateRequest) (*computeservice.CreateMLServiceResponse, error) {
	return s.computeService.CreateMLServiceWithResponse(ctx, ns, body)
}

func (s *suite) getMLService(ctx context.Context, ns, name string) (*computeservice.GetMLServiceResponse, error) {
	return s.computeService.GetMLServiceWithResponse(ctx, ns, name)
}

func (s *suite) scaleMLService(ctx context.Context, ns, name string, replicas int32) (*computeservice.ScaleMLServiceResponse, error) {
	return s.computeService.ScaleMLServiceWithResponse(ctx, ns, name, computeservice.MLServiceScaleRequest{Replicas: replicas})
}

func (s *suite) deleteMLService(ctx context.Context, ns, name string) (*computeservice.DeleteMLServiceResponse, error) {
	return s.computeService.DeleteMLServiceWithResponse(ctx, ns, name)
}

// cleanupMLService is the MLService analogue of cleanupMLRun: API DELETE first,
// then a direct K8s delete as a fallback so a failed API teardown can't leak the
// CR + Deployment/PVC into later tests in the shared namespace.
func cleanupMLService(t *testing.T, ns, name string) {
	t.Helper()
	t.Cleanup(func() {
		bg := context.Background()
		_, _ = h.deleteMLService(bg, ns, name)
		_ = h.k8s.Delete(bg, &mlservicev1.MLService{ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name}})
	})
}

// ----- compute-service: traffic policies -----

func (s *suite) createTrafficPolicy(ctx context.Context, ns string, body computeservice.TrafficPolicyCreateRequest) (*computeservice.CreateTrafficPolicyResponse, error) {
	return s.computeService.CreateTrafficPolicyWithResponse(ctx, ns, body)
}

func (s *suite) splitTrafficPolicy(ctx context.Context, ns, name string, body computeservice.TrafficPolicySplitRequest) (*computeservice.SplitTrafficPolicyResponse, error) {
	return s.computeService.SplitTrafficPolicyWithResponse(ctx, ns, name, body)
}

func (s *suite) promoteTrafficPolicy(ctx context.Context, ns, name string) (*computeservice.PromoteTrafficPolicyResponse, error) {
	return s.computeService.PromoteTrafficPolicyWithResponse(ctx, ns, name)
}

func (s *suite) deleteTrafficPolicy(ctx context.Context, ns, name string) (*computeservice.DeleteTrafficPolicyResponse, error) {
	return s.computeService.DeleteTrafficPolicyWithResponse(ctx, ns, name)
}

// cleanupTrafficPolicy is the MLTrafficPolicy analogue of cleanupMLRun.
func cleanupTrafficPolicy(t *testing.T, ns, name string) {
	t.Helper()
	t.Cleanup(func() {
		bg := context.Background()
		_, _ = h.deleteTrafficPolicy(bg, ns, name)
		_ = h.k8s.Delete(bg, &mltpv1.MLTrafficPolicy{ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name}})
	})
}

// canaryTrafficReq builds a canary traffic policy fronting two member services
// (stable @90 / canary @10).
func canaryTrafficReq(name, stableSvc, canarySvc string) computeservice.TrafficPolicyCreateRequest {
	return computeservice.TrafficPolicyCreateRequest{
		Name:     name,
		Mode:     string(mltpv1.TrafficModeCanary),
		Endpoint: &computeservice.MLTrafficPolicyEndpoint{Hostname: ptr(name + ".e2e.local")},
		Backends: []computeservice.MLTrafficPolicyBackendMember{
			{ServiceName: stableSvc, Role: ptr(string(mltpv1.RoleStable)), Weight: 90},
			{ServiceName: canarySvc, Role: ptr(string(mltpv1.RoleCanary)), Weight: 10},
		},
	}
}

// ----- request builders -----

// busyboxMLRunReq builds a minimal native/job MLRun create request that runs to
// completion quickly.
func busyboxMLRunReq(name, pool, unit, quota string) computeservice.MLRunCreateRequest {
	return computeservice.MLRunCreateRequest{
		Name:     name,
		PoolName: pool,
		UnitName: unit,
		Quota:    quota,
		Backend:  &computeservice.MLRunBackendSpec{Name: "native", Engine: "job"},
		Roles: []computeservice.MLRunRoleSpec{{
			Name:          mlrunv1.DefaultRoleName,
			Replicas:      1,
			RestartPolicy: ptr(string(corev1.RestartPolicyNever)),
			Template: computeservice.MLRunPodTemplateSubset{
				Image:           h.cfg.MLRunImage,
				ImagePullPolicy: ptr(string(corev1.PullIfNotPresent)),
				Command:         &[]string{"sh", "-c", "echo hello"},
			},
		}},
	}
}

// nginxMLServiceReq builds a minimal native/deployment MLService create request.
func nginxMLServiceReq(name, pool, unit, quota string, route *computeservice.MLServiceRoute) computeservice.MLServiceCreateRequest {
	return computeservice.MLServiceCreateRequest{
		Name:     name,
		Kind:     ptr(mlservicev1.ServiceKindService),
		PoolName: pool,
		UnitName: unit,
		Quota:    quota,
		Backend:  &computeservice.MLServiceBackend{Name: "native", Engine: "deployment"},
		Roles: []computeservice.MLServiceRoleSpec{{
			Name:     mlservicev1.DefaultRoleName,
			Replicas: 1,
			Template: computeservice.MLServicePodTemplate{
				Image:           h.cfg.ServiceImage,
				ImagePullPolicy: ptr(string(corev1.PullIfNotPresent)),
				Ports:           &[]computeservice.MLServicePodPort{{Name: "http", ContainerPort: 80, Protocol: ptr(string(corev1.ProtocolTCP))}},
			},
		}},
		Route: route,
	}
}
