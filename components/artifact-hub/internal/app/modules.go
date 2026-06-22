package app

import (
	"time"

	"github.com/go-logr/logr"
	"gorm.io/gorm"

	artmod "github.com/axisml/axisml/components/artifact-hub/internal/artifact"
	"github.com/axisml/axisml/components/artifact-hub/internal/artifact/handler"
	"github.com/axisml/axisml/components/artifact-hub/internal/config"
	"github.com/axisml/axisml/components/artifact-hub/internal/gc"
	"github.com/axisml/axisml/components/artifact-hub/internal/server"
	"github.com/axisml/axisml/components/artifact-hub/internal/storage/oci"
)

// BuildModules constructs the full domain wiring: the HTTP API modules and the
// background GC worker. The worker is returned to the caller, which gates it
// behind leader election.
func BuildModules(
	cfg config.Config,
	gormDB *gorm.DB,
	log logr.Logger,
) ([]server.Module, *gc.Worker, error) {
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
	worker := gc.New(cfg, gormDB, log.WithName("gc-worker"))
	return modules, worker, nil
}

// registerHandlers wires Kind handlers into the process-global registry.
// Idempotent on a fresh process; integration tests that re-invoke
// BuildModules in the same process check the model entry first.
func registerHandlers(client *oci.Client) {
	if _, ok := handler.Get("model"); ok {
		return // already registered (test re-runs)
	}
	handler.Register(handler.NewModelHandler(client))
	handler.Register(handler.NewImageHandler(client))
	// Dataset bucket is conventionally `axisml-artifact-hub` per infra.md;
	// the MVP handler issues prefix-scoped placeholder credentials without
	// a live STS integration.
	handler.Register(handler.NewDatasetHandler("axisml-artifact-hub",
		client.Endpoint()))
}
