// Package module is the public assembly API for Cluster Manager. A composition
// root — the Kubernetes binary, or Lite's axisml-system — injects the
// deployment-form-neutral stores (ResourcePool + Tenant) and mounts the REST
// routes on a shared /api/v1 router group.
//
// Cluster Manager is stateless (no PG, no migration): the source of truth is
// either the cluster-scoped CRs (Kubernetes) or the static CR-YAML config
// (Lite). The handlers, conversions and validation stay in internal packages;
// only this constructor and route registration are exported.
package module

import (
	"github.com/gin-gonic/gin"

	"github.com/axisml/axisml/components/cluster-manager/internal/resourcepool"
	"github.com/axisml/axisml/components/cluster-manager/internal/server"
	"github.com/axisml/axisml/components/cluster-manager/internal/tenant"
	"github.com/axisml/axisml/components/cluster-manager/pkg/provider"
)

// Route wires its endpoints into an /api/v1 router group.
type Route interface {
	Register(rg *gin.RouterGroup)
}

// Deps are the form-neutral stores a composition root injects.
type Deps struct {
	Pools   provider.ResourcePoolStore
	Tenants provider.TenantStore
}

// Module is the assembled Cluster Manager REST surface.
type Module struct {
	routes       []Route
	capabilities server.Capabilities
}

// New assembles the Cluster Manager handlers over the injected stores.
func New(d Deps) *Module {
	return &Module{
		routes: []Route{
			resourcepool.NewHandler(d.Pools),
			tenant.NewHandler(d.Tenants, d.Pools),
		},
		capabilities: server.Capabilities{
			MultiTenant:           d.Tenants.Writable(),
			ResourcePoolsWritable: d.Pools.Writable(),
		},
	}
}

// Capabilities returns the deployment-form capability document derived from the
// injected stores. A composition root serves it at GET /api/v1/capabilities
// (Standard, per-service) or folds it into an aggregate (Lite).
func (m *Module) Capabilities() server.Capabilities { return m.capabilities }

// RegisterRoutes mounts every Cluster Manager route on the supplied group.
func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	for _, r := range m.routes {
		r.Register(rg)
	}
}

// Routes returns the route handlers for composition roots that batch
// registration across services.
func (m *Module) Routes() []Route { return m.routes }
