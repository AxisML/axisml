// Package module is the public assembly API for Compute Service. A composition
// root — the Kubernetes binary, or Lite's axisml-system — injects the
// deployment-form-neutral dependencies (a ComputeRuntime, a resource catalog, a
// workspace volume provisioner) and receives the HTTP routes plus the
// background reconcilers, then mounts them on a shared router and runs them.
//
// The handlers, repositories, Kubernetes adapter and business logic stay in the
// component's internal packages; only this constructor, the route registration,
// the runnables and migration are exported. The status reflow is intentionally
// NOT assembled here: it is form-specific (the Kubernetes binary uses an
// apiserver informer; Lite polls the runtime), while both share the same CR
// Status → PG mapping (design §4.2).
package module

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-logr/logr"
	"gorm.io/gorm"

	"github.com/axisml/axisml/components/compute-service/internal/db"
	jobmod "github.com/axisml/axisml/components/compute-service/internal/mlrun"
	servicemod "github.com/axisml/axisml/components/compute-service/internal/mlservice"
	"github.com/axisml/axisml/components/compute-service/internal/server"
	trafficmod "github.com/axisml/axisml/components/compute-service/internal/trafficpolicy"
	"github.com/axisml/axisml/components/compute-service/pkg/computeruntime"
	"github.com/axisml/axisml/components/compute-service/pkg/provider"
)

// Route wires its endpoints into an /api/v1 router group.
type Route interface {
	Register(rg *gin.RouterGroup)
}

// Runnable is a long-running background loop started under the composition
// root's lifecycle (and leader election, where applicable).
type Runnable interface {
	Start(ctx context.Context) error
}

// Deps are the form-neutral dependencies a composition root injects.
type Deps struct {
	DB                *gorm.DB
	Runtime           computeruntime.ComputeRuntime
	Catalog           provider.ResourceCatalog
	Volumes           provider.WorkspaceVolumeProvisioner
	Log               logr.Logger
	ReconcileInterval time.Duration
	// RuntimeName labels the workload execution engine in the capability
	// document ("kubernetes" or "standalone"). Defaults to "kubernetes".
	RuntimeName string
	// QuotaEnforcement reports whether the scheduler admits pods against an
	// ElasticQuota (true on Kubernetes, false on the Lite Standalone runtime).
	QuotaEnforcement bool
}

// Module is the assembled Compute Service: its HTTP routes and reconcilers.
type Module struct {
	routes       []Route
	runnables    []Runnable
	reflow       []Runnable
	capabilities server.Capabilities
}

// New assembles the Compute Service business modules from injected providers.
func New(d Deps) (*Module, error) {
	jobs := jobmod.NewService(d.DB, d.Catalog)
	serviceRepo := servicemod.NewRepository(d.DB)
	trafficRepo := trafficmod.NewRepository(d.DB)
	services := servicemod.NewMLService(d.DB, d.Catalog, d.Volumes, trafficRepo)
	traffic := trafficmod.NewService(d.DB, serviceRepo, trafficRepo)

	jobRecon := jobmod.NewReconciler(d.DB, d.Runtime, d.Log.WithName("mlrun-reconciler"), d.ReconcileInterval)
	serviceRecon := servicemod.NewReconciler(d.DB, d.Runtime, d.Log.WithName("mlservice-reconciler"), d.ReconcileInterval)
	trafficRecon := trafficmod.NewReconciler(d.DB, d.Runtime, d.Log.WithName("traffic-policy-reconciler"), d.ReconcileInterval)

	runtimeName := d.RuntimeName
	if runtimeName == "" {
		runtimeName = "kubernetes"
	}

	return &Module{
		routes: []Route{
			jobmod.NewHandler(jobs, d.Runtime),
			servicemod.NewHandler(services, d.Runtime),
			trafficmod.NewHandler(traffic),
		},
		runnables: []Runnable{jobRecon, serviceRecon, trafficRecon},
		reflow: []Runnable{
			jobmod.NewStatusPoller(d.DB, d.Runtime, d.Log.WithName("mlrun-status-poller"), d.ReconcileInterval),
			servicemod.NewStatusPoller(d.DB, d.Runtime, d.Log.WithName("mlservice-status-poller"), d.ReconcileInterval),
			trafficmod.NewStatusPoller(d.DB, d.Runtime, d.Log.WithName("traffic-policy-status-poller"), d.ReconcileInterval),
		},
		capabilities: server.Capabilities{
			Runtime:          runtimeName,
			QuotaEnforcement: d.QuotaEnforcement,
		},
	}, nil
}

// Capabilities returns the deployment-form capability document. A composition
// root serves it at GET /api/v1/capabilities (Standard, per-service) or folds it
// into an aggregate (Lite).
func (m *Module) Capabilities() server.Capabilities { return m.capabilities }

// RegisterRoutes mounts every Compute route on the supplied /api/v1 group.
func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	for _, r := range m.routes {
		r.Register(rg)
	}
}

// Routes returns the route handlers (e.g. for composition roots that batch
// route registration across services).
func (m *Module) Routes() []Route { return m.routes }

// Runnables returns the background reconcilers to start under the composition
// root's lifecycle.
func (m *Module) Runnables() []Runnable { return m.runnables }

// StatusReflowRunnables returns the runtime-Observe status pollers — the
// form-specific status reflow for composition roots WITHOUT an apiserver
// informer (Lite, design §4.2). The Kubernetes binary does not use these; it
// reflows via internal informers instead. Both share the same CR Status → PG
// mapping, so the two forms converge identically.
func (m *Module) StatusReflowRunnables() []Runnable { return m.reflow }

// Migrate runs the Compute Service schema migrations. Safe to call repeatedly.
func Migrate(gormDB *gorm.DB) error { return db.Migrate(gormDB) }
