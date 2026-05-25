package app

import (
	"time"

	"github.com/go-logr/logr"
	"gorm.io/gorm"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	artmod "github.com/axisml/axisml/components/artifact-hub/internal/artifact"
	"github.com/axisml/axisml/components/artifact-hub/internal/artifact/handler"
	"github.com/axisml/axisml/components/artifact-hub/internal/config"
	"github.com/axisml/axisml/components/artifact-hub/internal/gc"
	"github.com/axisml/axisml/components/artifact-hub/internal/server"
	"github.com/axisml/axisml/components/artifact-hub/internal/storage/oci"
)

// BuildModules constructs the full domain wiring (HTTP routes + background
// runnables).
func BuildModules(
	cfg config.Config,
	gormDB *gorm.DB,
	_ manager.Manager,
	log logr.Logger,
) ([]server.Module, []manager.Runnable, error) {
	ociClient := oci.New(oci.Config{
		Endpoint:    cfg.OCIEndpoint,
		Scheme:      cfg.OCIScheme,
		Username:    cfg.OCIAdminUser,
		Password:    cfg.OCIAdminPassword,
		HTTPTimeout: 15 * time.Second,
	})
	registerHandlers(ociClient)

	artifacts := artmod.NewService(cfg, gormDB)

	modules := []server.Module{
		artmod.NewHandler(artifacts),
	}
	runnables := []manager.Runnable{
		gc.New(cfg, gormDB, log.WithName("gc-worker")),
	}
	return modules, runnables, nil
}

// registerHandlers wires Kind handlers into the process-global registry.
// Idempotent on a fresh process; integration tests that re-invoke
// BuildModules in the same process should call handler.Reset() first.
func registerHandlers(client *oci.Client) {
	mh := handler.NewModelHandler(client)
	if _, ok := handler.Get(mh.Kind()); ok {
		return
	}
	handler.Register(mh)
}
