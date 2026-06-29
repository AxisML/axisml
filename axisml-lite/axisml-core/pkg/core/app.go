package core

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/docker/docker/client"
	"github.com/gin-gonic/gin"
	"github.com/go-logr/logr"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	arthubmodule "github.com/axisml/axisml/axisml-system/artifact-hub/pkg/module"
	clustermodule "github.com/axisml/axisml/axisml-system/cluster-manager/pkg/module"
	"github.com/axisml/axisml/axisml-system/compute-service/pkg/logging"
	computemodule "github.com/axisml/axisml/axisml-system/compute-service/pkg/module"

	"github.com/axisml/axisml/axisml-lite/axisml-core/internal/runtime/standalone"
)

// Runnable is the common shape of every background loop the modules expose (the
// Compute reconcilers + status pollers and the Artifact Hub GC worker). App.Serve
// starts them; an embedding host that mounts App.Handler on its own server must
// start App.Runnables itself.
type Runnable interface {
	Start(ctx context.Context) error
}

// App is an assembled axisml-core: the shared database, the in-process Standalone
// Runtime, the three System modules mounted on one gin engine, and their
// background loops. It is built by New and not yet serving — the caller either
// runs App.Serve (the binary) or mounts App.Handler on its own HTTP server and
// starts App.Runnables (an embedding host).
type App struct {
	settings  Settings
	log       logr.Logger
	db        *gorm.DB
	ownDB     bool // close the DB on Close only when New opened it
	dcli      *client.Client
	engine    *gin.Engine
	runnables []Runnable

	// loopsClaimed guards the background loops against a double start: it is set
	// the first time they are claimed, by either Serve (which starts them) or
	// Runnables (which hands them to a host to start). A second claim by either
	// path is refused, so a host that uses both — e.g. Runnables() then Serve —
	// cannot accidentally run every reconciler twice against the same DB.
	loopsClaimed atomic.Bool
}

// ErrLoopsAlreadyStarted is returned by Serve, and reported by Runnables, when
// the background loops have already been claimed by the other path. The loops
// are owned by exactly one runner — call Serve, or start Runnables yourself,
// never both.
var ErrLoopsAlreadyStarted = errors.New("axisml-core: background loops already started")

// New is the axisml-core composition root. It resolves the inputs (Settings,
// database, static config, logger — each overridable via Option), builds the
// in-process Standalone Runtime, assembles the three System modules on one
// router and collects their background loops. It does NOT migrate the database
// or start serving: call App.Migrate then App.Serve (or mount App.Handler).
func New(ctx context.Context, cfg Config, opts ...Option) (app *App, err error) {
	o := options{settings: DefaultSettings()}
	for _, opt := range opts {
		opt(&o)
	}

	var log logr.Logger
	if o.logger != nil {
		log = *o.logger
	} else {
		log, err = logging.New(cfg.Log.Level, cfg.Log.Format)
		if err != nil {
			return nil, err
		}
	}

	db := o.db
	ownDB := false
	if db == nil {
		db, err = openDB(cfg, log)
		if err != nil {
			return nil, err
		}
		ownDB = true
	}
	// From here on a failure must release a DB we opened ourselves.
	defer func() {
		if err != nil && ownDB {
			_ = closeDB(db)
		}
	}()

	static := o.static
	if static == nil {
		static, err = LoadStaticConfig(o.settings.PoolConfigDir)
		if err != nil {
			return nil, fmt.Errorf("load static config: %w", err)
		}
	} else if err = static.validate(); err != nil {
		return nil, fmt.Errorf("validate static config: %w", err)
	}
	catalog := NewConfigResourceCatalog(static.Pool)
	tenants := NewStaticTenantStore(static.Tenant)

	// DOCKER_HOST is read by the Docker SDK (client.FromEnv); no config key.
	dcli, err := standalone.NewClient("")
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}
	defer func() {
		if err != nil {
			_ = dcli.Close()
		}
	}()
	rt := standalone.New(dcli, standalone.Config{
		WorkloadsNetwork: o.settings.WorkloadsNetwork,
		Tenant:           DefaultName,
		TraefikDir:       o.settings.GatewayConfigDir,
	}, log.WithName("runtime"))
	if nerr := rt.EnsureNetwork(ctx); nerr != nil {
		log.Error(nerr, "ensure workloads network (continuing)")
	}

	clusterMod := clustermodule.New(clustermodule.Deps{Pools: catalog, Tenants: tenants, Volumes: rt})
	computeMod, err := computemodule.New(computemodule.Deps{
		DB:                db,
		Runtime:           rt,
		Resolver:          catalog,
		Log:               log,
		ReconcileInterval: o.settings.ReconcileInterval,
		// Lite Standalone runtime: no scheduler, so no ElasticQuota admission.
		RuntimeName:      "standalone",
		QuotaEnforcement: false,
	})
	if err != nil {
		return nil, fmt.Errorf("assemble compute module: %w", err)
	}
	arthubMod, err := arthubmodule.New(arthubmodule.Deps{
		DB: db,
		Config: arthubmodule.Config{
			OCIEndpoint:      cfg.OCI.Endpoint,
			OCIAdminUser:     cfg.OCI.AdminUser,
			OCIAdminPassword: cfg.OCI.AdminPassword,
			DatasetBucket:    o.settings.DatasetBucket,
			GCInterval:       o.settings.GCInterval,
			UploadingTTL:     o.settings.UploadingTTL,
			UploadTokenTTL:   o.settings.UploadTokenTTL,
		},
		Log: log,
	})
	if err != nil {
		return nil, fmt.Errorf("assemble artifact-hub module: %w", err)
	}

	engine, err := buildEngine(db, log, clusterMod, computeMod, arthubMod)
	if err != nil {
		return nil, err
	}

	var runnables []Runnable
	for _, r := range computeMod.Runnables() {
		runnables = append(runnables, r)
	}
	for _, r := range computeMod.StatusReflowRunnables() {
		runnables = append(runnables, r)
	}
	for _, r := range arthubMod.Runnables() {
		runnables = append(runnables, r)
	}

	return &App{
		settings:  o.settings,
		log:       log,
		db:        db,
		ownDB:     ownDB,
		dcli:      dcli,
		engine:    engine,
		runnables: runnables,
	}, nil
}

// Handler returns the assembled gin engine: the form-neutral middleware chain,
// the probes + capabilities routes and the three System modules under the
// X-Axisml-User gate. An embedding host mounts this on its own HTTP server to
// expose the full axisml-core API.
func (a *App) Handler() http.Handler { return a.engine }

// Runnables returns the modules' background loops (Compute reconcilers + status
// pollers and the Artifact Hub GC worker) for a host that runs its own server
// instead of calling Serve. Start each one (go r.Start(ctx)) bound to a context
// it cancels on shutdown.
//
// It claims the loops: calling it after Serve has started them — or calling it
// twice — returns nil (and logs), so the reconcilers are never started twice.
func (a *App) Runnables() []Runnable {
	if !a.loopsClaimed.CompareAndSwap(false, true) {
		a.log.Error(ErrLoopsAlreadyStarted, "Runnables() called after the loops were already started; returning nil")
		return nil
	}
	return a.runnables
}

// Migrate runs the module migrations against the App's database.
func (a *App) Migrate() error { return migrate(a.db) }

// Close releases the resources New acquired: the Docker client always, and the
// database only when New opened it (an injected DB is the caller's to close).
func (a *App) Close() error {
	var errs []error
	if a.dcli != nil {
		if err := a.dcli.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close docker client: %w", err))
		}
	}
	if a.ownDB {
		if err := closeDB(a.db); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// Serve starts every background loop and the HTTP server bound to
// Settings.APIBindAddress, then blocks until ctx is cancelled (graceful
// shutdown) or the server fails. Embedding hosts that mount Handler on their own
// server use Runnables instead and do not call Serve.
//
// On the way out it stops the background loops and waits for any in-flight
// reconcile tick to finish before returning, so the caller's deferred Close can
// safely tear down the shared DB / Docker client without racing a live loop.
func (a *App) Serve(ctx context.Context) error {
	if !a.loopsClaimed.CompareAndSwap(false, true) {
		return ErrLoopsAlreadyStarted
	}

	srv := &http.Server{Addr: a.settings.APIBindAddress, Handler: a.engine, ReadHeaderTimeout: 10 * time.Second}

	// Run the background loops on a context we cancel on the way out, so they are
	// also torn down when Serve returns because the HTTP server failed (a path
	// where the parent ctx is still live).
	runCtx, cancelRun := context.WithCancel(ctx)
	defer cancelRun()
	var wg sync.WaitGroup
	for _, r := range a.runnables {
		wg.Add(1)
		go func(run Runnable) {
			defer wg.Done()
			if err := run.Start(runCtx); err != nil {
				a.log.Error(err, "background runnable exited")
			}
		}(r)
	}

	errCh := make(chan error, 1)
	go func() {
		a.log.Info("axisml-core listening", "addr", a.settings.APIBindAddress)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	var err error
	select {
	case <-ctx.Done():
		shutCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		err = srv.Shutdown(shutCtx)
	case err = <-errCh:
	}

	// Stop the loops and wait for their goroutines to return before we do, so the
	// DB / Docker client outlives every reconcile tick that might still touch it.
	cancelRun()
	wg.Wait()
	return err
}

// Run is the binary's convenience entry point: assemble with the fixed Lite
// constants, run the migrations and serve until ctx is cancelled. It owns the
// full process lifecycle; embedding hosts use New + Handler/Runnables instead.
func Run(ctx context.Context, cfg Config) error {
	app, err := New(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() { _ = app.Close() }()
	if err := app.Migrate(); err != nil {
		return err
	}
	return app.Serve(ctx)
}

// buildEngine assembles the shared gin engine: the form-neutral middleware
// chain (request id, access log, recovery, BOTH services' identity stamping,
// shared RFC 7807 error rendering), the probes + capabilities routes, then the
// three modules under the X-Axisml-User gate.
func buildEngine(
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

func closeDB(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
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
	defer func() { _ = closeDB(db) }()
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
