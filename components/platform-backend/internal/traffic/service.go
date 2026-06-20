// Package traffic implements the TrafficPolicy tag: weighted / canary traffic
// distribution over member online services, backed by compute MLTrafficPolicy
// (backend.md §4.8). Weighted routing is derived compute-side.
package traffic

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/axisml/axisml/components/platform/internal/clients/computeservice"
	"github.com/axisml/axisml/components/platform/internal/guard"
	"github.com/axisml/axisml/components/platform/internal/server"
	"github.com/axisml/axisml/components/platform/internal/store"
	apperrors "github.com/axisml/axisml/components/platform/pkg/errors"
)

// Service holds traffic-policy orchestration.
type Service struct {
	compute *computeservice.Client
	tenants *store.TenantRepo
}

// NewService constructs a traffic Service.
func NewService(compute *computeservice.Client, tenants *store.TenantRepo) *Service {
	return &Service{compute: compute, tenants: tenants}
}

// Create validates members and creates the policy.
func (s *Service) Create(ctx context.Context, tenant, owner string, req server.TrafficPolicyCreateRequest) (*server.TrafficPolicy, error) {
	if err := guard.TenantActive(ctx, s.tenants, tenant); err != nil {
		return nil, err
	}
	for _, b := range req.Backends {
		svc, err := s.compute.GetMLService(ctx, tenant, b.ServiceName)
		if err != nil {
			return nil, err
		}
		if svc.Kind != "service" {
			return nil, apperrors.Newf(apperrors.ClassValidation, "%q is not an online service", b.ServiceName).WithReason("invalid-backend")
		}
	}
	if req.Endpoint.Path == "" {
		req.Endpoint.Path = fmt.Sprintf("/services/%s/%s/", tenant, req.Name)
	}
	input, err := buildInput(req)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ClassInternal, "build traffic policy", err)
	}
	tp, err := s.compute.CreateTrafficPolicy(ctx, tenant, input)
	if err != nil {
		return nil, err
	}
	v := toView(tp, tenant)
	return &v, nil
}

// Get returns a policy.
func (s *Service) Get(ctx context.Context, tenant, name string) (*server.TrafficPolicy, error) {
	tp, err := s.compute.GetTrafficPolicy(ctx, tenant, name)
	if err != nil {
		return nil, err
	}
	v := toView(tp, tenant)
	return &v, nil
}

// List lists policies for a tenant.
func (s *Service) List(ctx context.Context, tenant string) ([]server.TrafficPolicy, error) {
	tps, err := s.compute.ListTrafficPolicies(ctx, tenant, "")
	if err != nil {
		return nil, err
	}
	out := make([]server.TrafficPolicy, 0, len(tps))
	for i := range tps {
		out = append(out, toView(&tps[i], tenant))
	}
	return out, nil
}

// Split adjusts weights (weighted) or canary percent (canary).
func (s *Service) Split(ctx context.Context, tenant, name string, req server.TrafficPolicySplitRequest) (*server.TrafficPolicy, error) {
	var updates []computeservice.WeightUpdate
	if req.CanaryPercent != nil {
		// Canary: derive stable/canary weights from the policy's roles.
		tp, err := s.compute.GetTrafficPolicy(ctx, tenant, name)
		if err != nil {
			return nil, err
		}
		sp, _ := decode(tp)
		p := *req.CanaryPercent
		for _, b := range sp.Backends {
			switch b.Role {
			case "canary":
				updates = append(updates, computeservice.WeightUpdate{ServiceName: b.ServiceName, Weight: int32(p)})
			case "stable":
				updates = append(updates, computeservice.WeightUpdate{ServiceName: b.ServiceName, Weight: int32(100 - p)})
			}
		}
	} else {
		for _, b := range req.Backends {
			updates = append(updates, computeservice.WeightUpdate{ServiceName: b.ServiceName, Weight: int32(b.Weight)})
		}
	}
	tp, err := s.compute.SplitTrafficPolicy(ctx, tenant, name, updates)
	if err != nil {
		return nil, err
	}
	v := toView(tp, tenant)
	return &v, nil
}

// Promote promotes the canary to stable.
func (s *Service) Promote(ctx context.Context, tenant, name string) (*server.TrafficPolicy, error) {
	tp, err := s.compute.PromoteTrafficPolicy(ctx, tenant, name)
	if err != nil {
		return nil, err
	}
	v := toView(tp, tenant)
	return &v, nil
}

// Rollback rolls the canary back to the stable backend.
func (s *Service) Rollback(ctx context.Context, tenant, name string) (*server.TrafficPolicy, error) {
	tp, err := s.compute.RollbackTrafficPolicy(ctx, tenant, name)
	if err != nil {
		return nil, err
	}
	v := toView(tp, tenant)
	return &v, nil
}

// Delete removes a policy (members retained).
func (s *Service) Delete(ctx context.Context, tenant, name string) error {
	return s.compute.DeleteTrafficPolicy(ctx, tenant, name)
}

// Events lists the policy's events.
func (s *Service) Events(ctx context.Context, tenant, name string) ([]computeservice.Event, error) {
	// compute exposes events on MLServices, not policies; return empty for now.
	return nil, nil
}

func buildInput(req server.TrafficPolicyCreateRequest) (computeservice.TrafficCreate, error) {
	backends := make([]map[string]any, 0, len(req.Backends))
	for _, b := range req.Backends {
		m := map[string]any{"serviceName": b.ServiceName, "weight": b.Weight}
		if b.Role != "" {
			m["role"] = b.Role
		}
		backends = append(backends, m)
	}
	input := map[string]any{
		"name":     req.Name,
		"mode":     req.Mode,
		"backends": backends,
		"endpoint": map[string]any{"path": req.Endpoint.Path, "hostname": req.Endpoint.Hostname},
	}
	if req.DisplayName != "" {
		input["displayName"] = req.DisplayName
	}
	if req.Description != "" {
		input["description"] = req.Description
	}
	var out computeservice.TrafficCreate
	b, err := json.Marshal(input)
	if err != nil {
		return out, err
	}
	return out, json.Unmarshal(b, &out)
}
