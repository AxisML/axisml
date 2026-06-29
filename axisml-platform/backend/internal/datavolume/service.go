// Package datavolume serves the DataVolumes tag: a system-admin, tenant-scoped
// management surface over cluster-manager's Volume REST. Data volumes are durable
// PVCs the tenant's workloads mount; Platform persists nothing (cluster-manager's
// K8s etcd is the source of truth) and resolves the active tenant to its physical
// Kubernetes namespace before delegating downstream (backend.md §4.11).
package datavolume

import (
	"context"
	"errors"

	"github.com/axisml/axisml/axisml-platform/backend/internal/clients/clustermanager"
	"github.com/axisml/axisml/axisml-platform/backend/internal/server"
	"github.com/axisml/axisml/axisml-platform/backend/internal/store"
)

// Service orchestrates data-volume operations over cluster-manager.
type Service struct {
	cm      *clustermanager.Client
	tenants *store.TenantRepo
}

// NewService constructs the Service.
func NewService(cm *clustermanager.Client, tenants *store.TenantRepo) *Service {
	return &Service{cm: cm, tenants: tenants}
}

// namespaceFor maps a tenant identifier to its physical Kubernetes namespace.
func (s *Service) namespaceFor(ctx context.Context, tenant string) (string, error) {
	row, err := s.tenants.GetByIdentifier(ctx, tenant)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return "", server.NotFound("tenant not found")
		}
		return "", err
	}
	if row.KubernetesNamespace == "" {
		// Defensive: an empty namespace would make cluster-manager list/operate
		// across all namespaces. Create enforces a non-empty namespace, so this
		// only trips on a corrupt tenant row — fail rather than leak cross-tenant.
		return "", errors.New("tenant namespace is not configured")
	}
	return row.KubernetesNamespace, nil
}

// List returns the tenant's data volumes (with live phase/capacity).
func (s *Service) List(ctx context.Context, tenant string) ([]server.DataVolume, error) {
	ns, err := s.namespaceFor(ctx, tenant)
	if err != nil {
		return nil, err
	}
	vols, err := s.cm.ListVolumes(ctx, ns, "")
	if err != nil {
		return nil, err
	}
	out := make([]server.DataVolume, 0, len(vols))
	for i := range vols {
		out = append(out, toDataVolume(&vols[i]))
	}
	return out, nil
}

// Get returns one data volume with mount occupancy.
func (s *Service) Get(ctx context.Context, tenant, name string) (*server.DataVolume, error) {
	ns, err := s.namespaceFor(ctx, tenant)
	if err != nil {
		return nil, err
	}
	v, err := s.cm.GetVolume(ctx, ns, name)
	if err != nil {
		return nil, err
	}
	out := toDataVolume(v)
	return &out, nil
}

// Create materialises a data volume in the tenant's namespace.
func (s *Service) Create(ctx context.Context, tenant string, req server.DataVolumeCreateRequest) (*server.DataVolume, error) {
	ns, err := s.namespaceFor(ctx, tenant)
	if err != nil {
		return nil, err
	}
	v, err := s.cm.CreateVolume(ctx, createBody(ns, req))
	if err != nil {
		return nil, err
	}
	out := toDataVolume(v)
	return &out, nil
}

// Update expands and/or relabels a data volume.
func (s *Service) Update(ctx context.Context, tenant, name string, req server.DataVolumePatchRequest) (*server.DataVolume, error) {
	ns, err := s.namespaceFor(ctx, tenant)
	if err != nil {
		return nil, err
	}
	v, err := s.cm.UpdateVolume(ctx, ns, name, patchBody(req))
	if err != nil {
		return nil, err
	}
	out := toDataVolume(v)
	return &out, nil
}

// ListStorageClasses returns the cluster's storage classes for the create form.
func (s *Service) ListStorageClasses(ctx context.Context) ([]server.StorageClass, error) {
	classes, err := s.cm.ListStorageClasses(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]server.StorageClass, 0, len(classes))
	for i := range classes {
		out = append(out, toStorageClass(&classes[i]))
	}
	return out, nil
}

// Delete reclaims a data volume; occupancy-guarded downstream unless force.
func (s *Service) Delete(ctx context.Context, tenant, name string, force bool) error {
	ns, err := s.namespaceFor(ctx, tenant)
	if err != nil {
		return err
	}
	return s.cm.DeleteVolume(ctx, ns, name, force)
}
