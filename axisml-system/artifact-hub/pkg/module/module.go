// Package module is the public assembly API for Artifact Hub. A composition
// root — the Kubernetes binary, or Lite's axisml-system — injects the
// form-neutral config (OCI registry endpoint, dataset bucket, lifecycle TTLs)
// and receives the HTTP routes plus the background GC worker, then mounts the
// routes on a shared /api/v1 router group and runs the worker behind its own
// single-active control.
//
// Artifact Hub owns no Kubernetes client: PostgreSQL is both the source of
// truth and (via an advisory lock) the leader-election backend in both forms.
// The handlers, repositories, OCI adapter and GC logic stay in internal
// packages; only this constructor, route registration, runnables and migration
// are exported.
package module

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-logr/logr"
	"gorm.io/gorm"

	artmod "github.com/axisml/axisml/components/artifact-hub/internal/artifact"
	"github.com/axisml/axisml/components/artifact-hub/internal/artifact/handler"
	"github.com/axisml/axisml/components/artifact-hub/internal/config"
	"github.com/axisml/axisml/components/artifact-hub/internal/db"
	"github.com/axisml/axisml/components/artifact-hub/internal/gc"
	"github.com/axisml/axisml/components/artifact-hub/internal/storage/oci"
)

// DefaultDatasetBucket is the conventional object-storage bucket for datasets
// (infra.md) used when Config.DatasetBucket is empty.
const DefaultDatasetBucket = "axisml-artifact-hub"

// Route wires its endpoints into an /api/v1 router group.
type Route interface {
	Register(rg *gin.RouterGroup)
}

// Runnable is a long-running background loop started under the composition
// root's lifecycle (gated by single-active control where applicable).
type Runnable interface {
	Start(ctx context.Context) error
}

// Config is the form-neutral assembly DTO: the business knobs Artifact Hub
// needs, independent of how the composition root sources them.
type Config struct {
	// OCI registry (zot) — model and image artifacts.
	OCIEndpoint      string
	OCIScheme        string
	OCIAdminUser     string
	OCIAdminPassword string
	// DatasetBucket is the object-storage bucket for datasets; defaults to
	// DefaultDatasetBucket when empty.
	DatasetBucket string
	// Lifecycle.
	GCInterval     time.Duration
	UploadingTTL   time.Duration
	UploadTokenTTL time.Duration
}

// Deps are the dependencies a composition root injects.
type Deps struct {
	DB     *gorm.DB
	Config Config
	Log    logr.Logger
}

// Module is the assembled Artifact Hub: its HTTP routes and GC worker.
type Module struct {
	routes    []Route
	runnables []Runnable
}

// New assembles Artifact Hub from the injected config. It registers the model /
// image / dataset Kind handlers into the process-global registry (idempotent
// across re-invocation in the same process).
func New(d Deps) (*Module, error) {
	ociClient := oci.New(oci.Config{
		Endpoint:    d.Config.OCIEndpoint,
		Scheme:      d.Config.OCIScheme,
		Username:    d.Config.OCIAdminUser,
		Password:    d.Config.OCIAdminPassword,
		HTTPTimeout: 15 * time.Second,
	})
	bucket := d.Config.DatasetBucket
	if bucket == "" {
		bucket = DefaultDatasetBucket
	}
	registerHandlers(ociClient, bucket)

	cfg := config.Config{
		GCInterval:     d.Config.GCInterval,
		UploadingTTL:   d.Config.UploadingTTL,
		UploadTokenTTL: d.Config.UploadTokenTTL,
	}
	artifacts := artmod.NewService(cfg, d.DB)
	worker := gc.New(cfg, d.DB, d.Log.WithName("gc-worker"))

	return &Module{
		routes:    []Route{artmod.NewHandler(artifacts)},
		runnables: []Runnable{worker},
	}, nil
}

// RegisterRoutes mounts every Artifact Hub route on the supplied group.
func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	for _, r := range m.routes {
		r.Register(rg)
	}
}

// Routes returns the route handlers for composition roots that batch
// registration across services.
func (m *Module) Routes() []Route { return m.routes }

// Runnables returns the background workers to start under the composition
// root's lifecycle.
func (m *Module) Runnables() []Runnable { return m.runnables }

// Migrate runs the Artifact Hub schema migrations. Safe to call repeatedly.
func Migrate(gormDB *gorm.DB) error { return db.Migrate(gormDB) }

// registerHandlers wires the Kind handlers into the process-global registry.
// Idempotent on a fresh process; re-invocation (tests) checks the model entry
// first. The MVP dataset handler issues prefix-scoped placeholder credentials
// without a live STS integration.
func registerHandlers(client *oci.Client, datasetBucket string) {
	if _, ok := handler.Get("model"); ok {
		return // already registered (test re-runs)
	}
	handler.Register(handler.NewModelHandler(client))
	handler.Register(handler.NewImageHandler(client))
	handler.Register(handler.NewDatasetHandler(datasetBucket, client.Endpoint()))
}
