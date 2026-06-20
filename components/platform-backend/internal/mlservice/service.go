// Package mlservice implements the MLServices tag: online inference services
// backed by compute MLService(kind=service) (backend.md §4.3). Stop/start map to
// scale-0/scale-back via the last-replicas annotation (§5.5).
package mlservice

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/axisml/axisml/components/platform/internal/clients/computeservice"
	"github.com/axisml/axisml/components/platform/internal/guard"
	"github.com/axisml/axisml/components/platform/internal/server"
	"github.com/axisml/axisml/components/platform/internal/store"
	"github.com/axisml/axisml/components/platform/internal/svcutil"
	apperrors "github.com/axisml/axisml/components/platform/pkg/errors"
)

// Service holds online-service orchestration.
type Service struct {
	compute *computeservice.Client
	tenants *store.TenantRepo
}

// NewService constructs an MLService Service.
func NewService(compute *computeservice.Client, tenants *store.TenantRepo) *Service {
	return &Service{compute: compute, tenants: tenants}
}

// Create deploys an online service. Verifies the tenant is active, auto-fills the
// gateway path + base-URL env when route.path is empty.
//
// TODO(artifacts): preflight (modelName, modelVersion) Ready via artifact-hub and
// inject the resolved URI/digest snapshot once the artifacts client lands (§4.0).
func (s *Service) Create(ctx context.Context, tenant, owner string, req server.MLServiceCreateRequest) (*server.MLService, error) {
	if err := guard.TenantActive(ctx, s.tenants, tenant); err != nil {
		return nil, err
	}
	if req.Route.Enabled && req.Route.Path == "" {
		req.Route.Path = fmt.Sprintf("/services/%s/%s/", tenant, req.Name)
		req.Env = append(req.Env, server.EnvVar{Name: "AXISML_SERVICE_BASE_URL", Value: req.Route.Path})
	}
	input, err := svcutil.BuildServiceInput(req)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ClassInternal, "build service", err)
	}
	svc, err := s.compute.CreateMLService(ctx, tenant, input)
	if err != nil {
		return nil, err
	}
	v := svcutil.ServiceToView(svc, tenant)
	return &v, nil
}

// Get returns a service (must be kind=service).
func (s *Service) Get(ctx context.Context, tenant, name string) (*server.MLService, error) {
	svc, err := s.getService(ctx, tenant, name)
	if err != nil {
		return nil, err
	}
	v := svcutil.ServiceToView(svc, tenant)
	return &v, nil
}

// List lists online services for a tenant.
func (s *Service) List(ctx context.Context, tenant string) ([]server.MLService, error) {
	svcs, err := s.compute.ListMLServices(ctx, tenant, "")
	if err != nil {
		return nil, err
	}
	out := make([]server.MLService, 0, len(svcs))
	for i := range svcs {
		if svcs[i].Kind != "service" {
			continue
		}
		out = append(out, svcutil.ServiceToView(&svcs[i], tenant))
	}
	return out, nil
}

// UpdateMeta patches display metadata.
func (s *Service) UpdateMeta(ctx context.Context, tenant, name, displayName, description string) (*server.MLService, error) {
	if _, err := s.getService(ctx, tenant, name); err != nil {
		return nil, err
	}
	patch := patchMap(map[string]any{"displayName": displayName, "description": description})
	svc, err := s.compute.PatchMLService(ctx, tenant, name, patch)
	if err != nil {
		return nil, err
	}
	v := svcutil.ServiceToView(svc, tenant)
	return &v, nil
}

// Scale sets the replica count.
func (s *Service) Scale(ctx context.Context, tenant, name string, replicas int) (*server.MLService, error) {
	if _, err := s.getService(ctx, tenant, name); err != nil {
		return nil, err
	}
	svc, err := s.compute.ScaleMLService(ctx, tenant, name, replicas)
	if err != nil {
		return nil, err
	}
	v := svcutil.ServiceToView(svc, tenant)
	return &v, nil
}

// Stop scales to 0, recording the prior replica count for a later start.
func (s *Service) Stop(ctx context.Context, tenant, name string) (*server.MLService, error) {
	svc, err := s.getService(ctx, tenant, name)
	if err != nil {
		return nil, err
	}
	prev := svcutil.ServiceToView(svc, tenant).Replicas
	if prev > 0 {
		patch := patchMap(map[string]any{"annotations": map[string]string{svcutil.LastReplicasAnnotation: strconv.Itoa(prev)}})
		if _, err := s.compute.PatchMLService(ctx, tenant, name, patch); err != nil {
			return nil, err
		}
	}
	scaled, err := s.compute.ScaleMLService(ctx, tenant, name, 0)
	if err != nil {
		return nil, err
	}
	v := svcutil.ServiceToView(scaled, tenant)
	return &v, nil
}

// Start scales back to the recorded replica count (fallback 1).
func (s *Service) Start(ctx context.Context, tenant, name string) (*server.MLService, error) {
	svc, err := s.getService(ctx, tenant, name)
	if err != nil {
		return nil, err
	}
	replicas := lastReplicas(svc, 1)
	scaled, err := s.compute.ScaleMLService(ctx, tenant, name, replicas)
	if err != nil {
		return nil, err
	}
	v := svcutil.ServiceToView(scaled, tenant)
	return &v, nil
}

// Delete deletes a service (must be kind=service to avoid deleting a workspace).
func (s *Service) Delete(ctx context.Context, tenant, name string) error {
	if _, err := s.getService(ctx, tenant, name); err != nil {
		return err
	}
	return s.compute.DeleteMLService(ctx, tenant, name, nil)
}

// Pods / Events / PodEvents / PodLogs proxy compute.
func (s *Service) Pods(ctx context.Context, tenant, name string) ([]computeservice.Pod, error) {
	return s.compute.ListMLServicePods(ctx, tenant, name)
}
func (s *Service) Events(ctx context.Context, tenant, name string) ([]computeservice.Event, error) {
	return s.compute.ListMLServiceEvents(ctx, tenant, name)
}
func (s *Service) PodEvents(ctx context.Context, tenant, name, pod string) ([]computeservice.Event, error) {
	return s.compute.ListMLServicePodEvents(ctx, tenant, name, pod)
}
func (s *Service) PodLogs(ctx context.Context, tenant, name, pod string) ([]byte, error) {
	return s.compute.GetMLServicePodLogs(ctx, tenant, name, pod)
}

// getService loads a service and rejects non-service kinds.
func (s *Service) getService(ctx context.Context, tenant, name string) (*computeservice.MLServiceView, error) {
	svc, err := s.compute.GetMLService(ctx, tenant, name)
	if err != nil {
		return nil, err
	}
	if svc.Kind != "service" {
		return nil, server.NotFound("service not found")
	}
	return svc, nil
}

func lastReplicas(svc *computeservice.MLServiceView, fallback int) int {
	if svc.Annotations == nil {
		return fallback
	}
	if v, ok := (*svc.Annotations)[svcutil.LastReplicasAnnotation]; ok {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return fallback
}

func patchMap(m map[string]any) computeservice.MLServicePatch {
	var out computeservice.MLServicePatch
	b, _ := json.Marshal(m)
	_ = json.Unmarshal(b, &out)
	return out
}
