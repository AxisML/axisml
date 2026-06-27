package core

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-logr/logr"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	arthubmodule "github.com/axisml/axisml/components/artifact-hub/pkg/module"
	clustermodule "github.com/axisml/axisml/components/cluster-manager/pkg/module"
	"github.com/axisml/axisml/components/compute-service/pkg/logging"
	computemodule "github.com/axisml/axisml/components/compute-service/pkg/module"

	"github.com/axisml/axisml/axisml-lite/axisml-core/internal/runtime/docker"
)

// runnable is the common shape of every background loop the modules expose
// (the Compute reconcilers + status pollers and the Artifact Hub GC worker).
type runnable interface {
	Start(ctx context.Context) error
}

// Run is the axisml-core composition root. It opens the shared database, runs
// the Compute → Artifact Hub migrations, loads the static Cluster Manager
// config, builds the in-process Standalone Runtime, assembles the three System
// modules on one router and starts the HTTP server plus every background loop.
func Run(ctx context.Context, cfg Config) error {
	log, err := logging.New(cfg.Log.Level, cfg.Log.Format)
	if err != nil {
		return err
	}

	db, err := openDB(cfg, log)
	if err != nil {
		return err
	}
	if err := migrate(db); err != nil {
		return err
	}

	static, err := LoadStaticConfig(PoolConfigDir)
	if err != nil {
		return fmt.Errorf("load static config: %w", err)
	}
	catalog := NewConfigResourceCatalog(static.Pool)
	tenants := NewStaticTenantStore(static.Tenant)

	// DOCKER_HOST is read by the Docker SDK (client.FromEnv); no config key.
	dcli, err := docker.NewClient("")
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}
	rt := docker.New(dcli, docker.Config{
		WorkloadsNetwork: WorkloadsNetwork,
		Tenant:           DefaultName,
		TraefikDir:       GatewayConfigDir,
		RuntimeDir:       StateDir,
	}, log.WithName("runtime"))
	if err := rt.EnsureNetwork(ctx); err != nil {
		log.Error(err, "ensure workloads network (continuing)")
	}

	clusterMod := clustermodule.New(clustermodule.Deps{Pools: catalog, Tenants: tenants, Volumes: rt})
	computeMod, err := computemodule.New(computemodule.Deps{
		DB:                db,
		Runtime:           rt,
		Resolver:          catalog,
		Log:               log,
		ReconcileInterval: ReconcileInterval,
		// Lite Standalone runtime: no scheduler, so no ElasticQuota admission.
		RuntimeName:      "standalone",
		QuotaEnforcement: false,
	})
	if err != nil {
		return fmt.Errorf("assemble compute module: %w", err)
	}
	arthubMod, err := arthubmodule.New(arthubmodule.Deps{
		DB: db,
		Config: arthubmodule.Config{
			OCIEndpoint:      cfg.OCI.Endpoint,
			OCIAdminUser:     cfg.OCI.AdminUser,
			OCIAdminPassword: cfg.OCI.AdminPassword,
			DatasetBucket:    DatasetBucket,
			GCInterval:       GCInterval,
			UploadingTTL:     UploadingTTL,
			UploadTokenTTL:   UploadTokenTTL,
		},
		Log: log,
	})
	if err != nil {
		return fmt.Errorf("assemble artifact-hub module: %w", err)
	}

	engine, err := buildEngine(cfg, db, log, clusterMod, computeMod, arthubMod)
	if err != nil {
		return err
	}

	var runnables []runnable
	for _, r := range computeMod.Runnables() {
		runnables = append(runnables, r)
	}
	for _, r := range computeMod.StatusReflowRunnables() {
		runnables = append(runnables, r)
	}
	for _, r := range arthubMod.Runnables() {
		runnables = append(runnables, r)
	}

	return serve(ctx, cfg, engine, runnables, log)
}

// buildEngine assembles the shared gin engine: the form-neutral middleware
// chain (request id, access log, recovery, BOTH services' identity stamping,
// shared RFC 7807 error rendering), the probes + capabilities routes, then the
// three modules under the X-Axisml-User gate.
func buildEngine(
	cfg Config,
	db *gorm.DB,
	log logr.Logger,
	clusterMod *clustermodule.Module,
	computeMod *computemodule.Module,
	arthubMod *arthubmodule.Module,
) (*gin.Engine, error) {
	if err := computemodule.RegisterValidators(); err != nil {
		return nil, fmt.Errorf("register compute validators: %w", err)
	}
	if err := arthubmodule.RegisterValidators(); err != nil {
		return nil, fmt.Errorf("register artifact-hub validators: %w", err)
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(
		computemodule.RequestID(),
		computemodule.AccessLog(log),
		computemodule.Recovery(log),
		computemodule.IdentityMiddleware(),
		arthubmodule.IdentityMiddleware(),
		computemodule.ErrorHandler(),
	)

	r.GET("/healthz", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	r.GET("/readyz", func(c *gin.Context) {
		cctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := pingDB(cctx, db); err != nil {
			c.String(http.StatusServiceUnavailable, "not ready: %v", err)
			return
		}
		c.String(http.StatusOK, "ok")
	})
	// Capability document is unauthenticated so Platform can read it pre-login.
	r.GET("/api/v1/capabilities", capabilitiesHandler(aggregateCapabilities(clusterMod, computeMod, arthubMod)))

	api := r.Group("/api/v1", clustermodule.RequireUser())
	clusterMod.RegisterRoutes(api)
	computeMod.RegisterRoutes(api)
	arthubMod.RegisterRoutes(api)
	return r, nil
}

// serve runs the HTTP server and every background loop, all bound to ctx.
func serve(ctx context.Context, cfg Config, engine *gin.Engine, runnables []runnable, log logr.Logger) error {
	srv := &http.Server{Addr: APIBindAddress, Handler: engine, ReadHeaderTimeout: 10 * time.Second}

	errCh := make(chan error, 1)
	for _, r := range runnables {
		go func(run runnable) {
			if err := run.Start(ctx); err != nil {
				log.Error(err, "background runnable exited")
			}
		}(r)
	}
	go func() {
		log.Info("axisml-core listening", "addr", APIBindAddress)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return srv.Shutdown(shutCtx)
	case err := <-errCh:
		return err
	}
}

func openDB(cfg Config, log logr.Logger) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.PostgresDSN()), &gorm.Config{
		Logger:  gormlogger.Default.LogMode(gormlogger.Warn),
		NowFunc: func() time.Time { return time.Now().UTC() },
	})
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(time.Hour)
	return db, nil
}

// Migrate opens the database and runs the module migrations, then returns. It
// backs the `axisml-core migrate` subcommand for an explicit pre-flight step.
func Migrate(cfg Config) error {
	log, err := logging.New(cfg.Log.Level, cfg.Log.Format)
	if err != nil {
		return err
	}
	db, err := openDB(cfg, log)
	if err != nil {
		return err
	}
	return migrate(db)
}

// migrate runs the module migrations in dependency order: Compute Service then
// Artifact Hub (design §8). Cluster Manager is stateless. Each module owns its
// own migration table, so they coexist in the shared database.
func migrate(db *gorm.DB) error {
	if err := computemodule.Migrate(db); err != nil {
		return fmt.Errorf("compute migrate: %w", err)
	}
	if err := arthubmodule.Migrate(db); err != nil {
		return fmt.Errorf("artifact-hub migrate: %w", err)
	}
	return nil
}

func pingDB(ctx context.Context, db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}
