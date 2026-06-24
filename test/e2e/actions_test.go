//go:build (e2e || standard) && !lite

package e2e

import (
	"context"

	"github.com/axisml/axisml/test/e2e/internal/clients/clustermanager"
	"github.com/axisml/axisml/test/e2e/internal/clients/computeservice"
)

// Reusable client actions shared across the remaining Standard test files. Each
// is a thin wrapper over the generated, typed client and returns its typed
// response so callers can assert on StatusCode() and the JSONxxx payload. The
// black-box CORE tests call the typed clients via the Harness accessors directly
// and do not use these.

// ----- cluster-manager: tenants -----
//
// Tenant lifecycle is owned by cluster-manager (REST writer over the Tenant CR).

func (s *suite) createTenant(ctx context.Context, body clustermanager.CreateTenantRequest) (*clustermanager.CreateTenantResponse, error) {
	return s.clusterManager.CreateTenantWithResponse(ctx, body)
}

func (s *suite) deleteTenant(ctx context.Context, name string) (*clustermanager.DeleteTenantResponse, error) {
	return s.clusterManager.DeleteTenantWithResponse(ctx, name)
}

// ----- compute-service: mlruns / mlservices -----

func (s *suite) createMLRun(ctx context.Context, ns string, body computeservice.MLRunCreateRequest) (*computeservice.CreateMLRunResponse, error) {
	return s.computeService.CreateMLRunWithResponse(ctx, ns, body)
}

func (s *suite) createMLService(ctx context.Context, ns string, body computeservice.MLServiceCreateRequest) (*computeservice.CreateMLServiceResponse, error) {
	return s.computeService.CreateMLServiceWithResponse(ctx, ns, body)
}

func (s *suite) getMLService(ctx context.Context, ns, name string) (*computeservice.GetMLServiceResponse, error) {
	return s.computeService.GetMLServiceWithResponse(ctx, ns, name)
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
