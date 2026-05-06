package app

import (
	"time"

	"github.com/go-logr/logr"
	"gorm.io/gorm"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	artmod "github.com/axisml/axisml/components/artifacts/internal/artifact"
	"github.com/axisml/axisml/components/artifacts/internal/artifact/handler"
	"github.com/axisml/axisml/components/artifacts/internal/config"
	"github.com/axisml/axisml/components/artifacts/internal/gc"
	repomod "github.com/axisml/axisml/components/artifacts/internal/repo"
	"github.com/axisml/axisml/components/artifacts/internal/server"
	"github.com/axisml/axisml/components/artifacts/internal/storage/oci"
	"github.com/axisml/axisml/components/artifacts/internal/tenantresolver"
)

// BuildModules constructs the full domain wiring (HTTP routes + background
// runnables). Construction order matters because of cross-module deps.
// Exported so integration tests can reuse the same wiring.
func BuildModules(
	cfg config.Config,
	gormDB *gorm.DB,
	_ manager.Manager,
	log logr.Logger,
) ([]server.Module, []manager.Runnable, error) {
	// Storage backend client + Kind handler registration.
	ociClient := oci.New(oci.Config{
		Endpoint:    cfg.OCIEndpoint,
		Scheme:      cfg.OCIScheme,
		Username:    cfg.OCIAdminUser,
		Password:    cfg.OCIAdminPassword,
		HTTPTimeout: 15 * time.Second,
	})
	registerHandlers(ociClient)

	tenants := tenantresolver.New(gormDB)
	tenantMW := tenantresolver.Middleware(tenants)

	repos := repomod.NewService(gormDB)
	artifacts := artmod.NewService(cfg, gormDB, repos)

	modules := []server.Module{
		repomod.NewHandler(repos, tenantMW),
		artmod.NewHandler(artifacts, tenantMW),
	}
	runnables := []manager.Runnable{
		gc.New(cfg, gormDB, log.WithName("gc-worker")),
	}
	return modules, runnables, nil
}

// registerHandlers wires Kind handlers into the process-global registry.
// Phase 2 will add dataset / image / eval_report.
//
// Idempotent: re-registration of an already-registered Kind is a no-op so
// integration tests can call BuildModules multiple times in the same
// process.
func registerHandlers(client *oci.Client) {
	if _, ok := handler.Get(repomod.KindModel); ok {
		return
	}
	handler.Register(handler.NewModelHandler(client))
}
