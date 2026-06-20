// Package workspace implements the Workspaces tag: long-running interactive dev
// containers backed by compute MLService(kind=workspace) (backend.md §4.4).
// Workspaces are always single-replica; stop/start = scale 0/1.
package workspace

import (
	"context"
	"encoding/json"

	"github.com/axisml/axisml/components/platform/internal/clients/computeservice"
	"github.com/axisml/axisml/components/platform/internal/guard"
	"github.com/axisml/axisml/components/platform/internal/server"
	"github.com/axisml/axisml/components/platform/internal/store"
	"github.com/axisml/axisml/components/platform/internal/svcutil"
	apperrors "github.com/axisml/axisml/components/platform/pkg/errors"
)

// Service holds workspace orchestration.
type Service struct {
	compute *computeservice.Client
	tenants *store.TenantRepo
}

// NewService constructs a Workspace Service.
func NewService(compute *computeservice.Client, tenants *store.TenantRepo) *Service {
	return &Service{compute: compute, tenants: tenants}
}

// Create provisions a workspace (with its PVC, compute-side).
func (s *Service) Create(ctx context.Context, tenant, owner string, req server.WorkspaceCreateRequest) (*server.Workspace, error) {
	if err := guard.TenantActive(ctx, s.tenants, tenant); err != nil {
		return nil, err
	}
	input, err := svcutil.BuildWorkspaceInput(req)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ClassInternal, "build workspace", err)
	}
	svc, err := s.compute.CreateMLService(ctx, tenant, input)
	if err != nil {
		return nil, err
	}
	v := svcutil.WorkspaceToView(svc, tenant)
	return &v, nil
}

// Get returns a workspace (must be kind=workspace).
func (s *Service) Get(ctx context.Context, tenant, name string) (*server.Workspace, error) {
	svc, err := s.getWorkspace(ctx, tenant, name)
	if err != nil {
		return nil, err
	}
	v := svcutil.WorkspaceToView(svc, tenant)
	return &v, nil
}

// List lists workspaces for a tenant.
func (s *Service) List(ctx context.Context, tenant string) ([]server.Workspace, error) {
	svcs, err := s.compute.ListMLServices(ctx, tenant, "")
	if err != nil {
		return nil, err
	}
	out := make([]server.Workspace, 0, len(svcs))
	for i := range svcs {
		if svcs[i].Kind != "workspace" {
			continue
		}
		out = append(out, svcutil.WorkspaceToView(&svcs[i], tenant))
	}
	return out, nil
}

// UpdateMeta patches display metadata.
func (s *Service) UpdateMeta(ctx context.Context, tenant, name, displayName, description string) (*server.Workspace, error) {
	if _, err := s.getWorkspace(ctx, tenant, name); err != nil {
		return nil, err
	}
	patch := patchMap(map[string]any{"displayName": displayName, "description": description})
	svc, err := s.compute.PatchMLService(ctx, tenant, name, patch)
	if err != nil {
		return nil, err
	}
	v := svcutil.WorkspaceToView(svc, tenant)
	return &v, nil
}

// Start scales the workspace to 1.
func (s *Service) Start(ctx context.Context, tenant, name string) (*server.Workspace, error) {
	return s.setReplicas(ctx, tenant, name, 1)
}

// Stop scales the workspace to 0.
func (s *Service) Stop(ctx context.Context, tenant, name string) (*server.Workspace, error) {
	return s.setReplicas(ctx, tenant, name, 0)
}

func (s *Service) setReplicas(ctx context.Context, tenant, name string, replicas int) (*server.Workspace, error) {
	if _, err := s.getWorkspace(ctx, tenant, name); err != nil {
		return nil, err
	}
	svc, err := s.compute.ScaleMLService(ctx, tenant, name, replicas)
	if err != nil {
		return nil, err
	}
	v := svcutil.WorkspaceToView(svc, tenant)
	return &v, nil
}

// Delete removes a workspace (PVC cascade default true).
func (s *Service) Delete(ctx context.Context, tenant, name string, deletePVC *bool) error {
	if _, err := s.getWorkspace(ctx, tenant, name); err != nil {
		return err
	}
	if deletePVC == nil {
		t := true
		deletePVC = &t
	}
	return s.compute.DeleteMLService(ctx, tenant, name, deletePVC)
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

func patchMap(m map[string]any) computeservice.MLServicePatch {
	var out computeservice.MLServicePatch
	b, _ := json.Marshal(m)
	_ = json.Unmarshal(b, &out)
	return out
}

func (s *Service) getWorkspace(ctx context.Context, tenant, name string) (*computeservice.MLService, error) {
	svc, err := s.compute.GetMLService(ctx, tenant, name)
	if err != nil {
		return nil, err
	}
	if svc.Kind != "workspace" {
		return nil, server.NotFound("workspace not found")
	}
	return svc, nil
}
