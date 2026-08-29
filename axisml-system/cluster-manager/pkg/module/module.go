// Package module is the public assembly API for Cluster Manager. A composition
// root — the Kubernetes binary or an external composition root — injects the
// deployment-form-neutral stores (ResourcePool + Tenant) and mounts the REST
// routes on a shared /api/v1 router group.
//
// Cluster Manager is stateless (no PG, no migration): the source of truth is
// either the cluster-scoped CRs (Kubernetes) or deployment-form providers (for
// example, PostgreSQL-backed pools and tenants in standalone). The
// handlers, conversions and validation stay in internal packages; only this
// constructor and route registration are exported.
package module

import (
	"github.com/gin-gonic/gin"

	"github.com/axisml/axisml/axisml-system/cluster-manager/internal/resourcepool"
	"github.com/axisml/axisml/axisml-system/cluster-manager/internal/tenant"
	"github.com/axisml/axisml/axisml-system/cluster-manager/internal/volume"
	"github.com/axisml/axisml/axisml-system/cluster-manager/pkg/extensions"
)

// Route wires its endpoints into an /api/v1 router group.
type Route interface {
	Register(rg *gin.RouterGroup)
}

// Deps are the form-neutral stores a composition root injects.
type Deps struct {
	Pools   extensions.ResourcePoolProvider
	Tenants extensions.TenantProvider
	Volumes extensions.VolumeManager
	// Metrics backs the per-pool metrics route (optional; nil = unavailable).
	Metrics resourcepool.MetricsQuerier
}

// Module is the assembled Cluster Manager REST surface.
type Module struct {
	routes []Route
}

// New assembles the Cluster Manager handlers over the injected stores.
func New(d Deps) *Module {
	return &Module{
		routes: []Route{
			resourcepool.NewHandler(d.Pools, d.Tenants, d.Metrics),
			tenant.NewHandler(d.Tenants, d.Pools),
			volume.NewHandler(d.Volumes),
		},
	}
}

// RegisterRoutes mounts every Cluster Manager route on the supplied group.
func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	for _, r := range m.routes {
		r.Register(rg)
	}
}

// Routes returns the route handlers for composition roots that batch
// registration across services.
func (m *Module) Routes() []Route { return m.routes }
