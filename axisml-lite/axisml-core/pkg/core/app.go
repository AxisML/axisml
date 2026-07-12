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
	settings Settings
	log      logr.Logger
	db       *gorm.DB
	ownDB    bool // close the DB on Close only when New opened it
	dcli     *client.Client

	// The three System modules, retained so RegisterRoutes can mount them onto
	// any gin router (a host's own engine, or the internal engine Handler builds).
	clusterMod *clustermodule.Module
	computeMod *computemodule.Module
	arthubMod  *arthubmodule.Module

	engine     *gin.Engine // built once by Handler via engineOnce
	engineOnce sync.Once
	runnables  []Runnable

	// loopsClaimed guards the background loops against a double start: it is set
	// the first time they are claimed, by either Serve (which starts them) or
	// Runnables (which hands them to a host to start). A second claim by either
	// path is refused, so a host that uses both — e.g. Runnables() then Serve —
	// cannot accidentally run every reconciler twice against the same DB.
	loopsClaimed atomic.Bool

	// specOnce builds the served OpenAPI document once; the surface is static, so
	// both formats are rendered on first request and cached.
	specOnce sync.Once
	specYAML []byte
	specJSON []byte
	specErr  error
}

// ErrLoopsAlreadyStarted is returned by Serve, and reported by Runnables, when
// the background loops have already been claimed by the other path. The loops
// are owned by exactly one runner — call Serve, or start Runnables yourself,
// never both.
var ErrLoopsAlreadyStarted = errors.New("axisml-core: background loops already started")

// New is the axisml-core composition root. It resolves the inputs (Settings,
// database, static config, logger — each overridable via Option), builds the
// in-process Standalone Runtime, assembles the three System modules, registers
// their gin binding validators and collects their background loops. It does NOT
// build the HTTP engine, migrate the database or start serving: call App.Migrate
// then App.Serve, mount App.Handler, or register onto your own gin engine with
// App.RegisterRoutes.
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
	catalog := NewConfigResourceCatalog(static.Pools...)
	tenants := NewStaticTenantStore(static.Tenants...)

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
		TraefikDir:       o.settings.GatewayConfigDir,
		HostPathVolumes:  tenantsHostPathVolumes(static.Tenants),
	}, log.WithName("runtime"))
	if nerr := rt.EnsureNetwork(ctx); nerr != nil {
		log.Error(nerr, "ensure workloads network (continuing)")
	}
	// Ensure every tenant's predefined data volumes exist before any workload
	// reconcile mounts them (idempotent; safe on every boot).
	for _, t := range static.Tenants {
		seedTenantVolumes(ctx, rt, t, log)
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

	// Register the modules' custom gin binding validators on the process-global
	// binding engine. This is the only fallible step of engine composition, so we
	// do it here (fail fast in New) rather than in RegisterRoutes; it is idempotent
	// (a tag→fn map overwrite) so both the internal engine and a host's own engine
	// see the validators.
	if err = computemodule.RegisterValidators(); err != nil {
		return nil, fmt.Errorf("register compute validators: %w", err)
	}
	if err = arthubmodule.RegisterValidators(); err != nil {
		return nil, fmt.Errorf("register artifact-hub validators: %w", err)
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
		settings:   o.settings,
		log:        log,
		db:         db,
		ownDB:      ownDB,
		dcli:       dcli,
		clusterMod: clusterMod,
		computeMod: computeMod,
		arthubMod:  arthubMod,
		runnables:  runnables,
	}, nil
}

// RegisterRoutes mounts the full axisml-core HTTP surface — the liveness /
// readiness probes, the unauthenticated capability document and OpenAPI spec, and
// the three System modules under the X-Axisml-User gate — onto the supplied gin
// router. The host owns the *gin.Engine and passes either it or a prefix group
// (r.Group("/axisml")); it may register its own routes on the same router
// alongside these.
//
// axisml-core's middleware chain (request id, access log, recovery, both
// services' identity stamping, shared RFC 7807 error rendering) is applied to a
// child group, so it wraps ONLY axisml-core's routes — never the host's own
// routes on the parent router. Call it once per router: registering the same
// routes twice panics on gin's duplicate-route check.
func (a *App) RegisterRoutes(r gin.IRouter) {
	// Scope the axisml middleware to a child group so sibling routes the host
	// registers directly on r never inherit it.
	grp := r.Group("")
	grp.Use(
		computemodule.RequestID(),
		computemodule.AccessLog(a.log),
		computemodule.Recovery(a.log),
		computemodule.IdentityMiddleware(),
		arthubmodule.IdentityMiddleware(),
		computemodule.ErrorHandler(),
	)

	grp.GET("/healthz", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	grp.GET("/readyz", func(c *gin.Context) {
		cctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := pingDB(cctx, a.db); err != nil {
			c.String(http.StatusServiceUnavailable, "not ready: %v", err)
			return
		}
		c.String(http.StatusOK, "ok")
	})
	// Capability document is unauthenticated so Platform can read it pre-login.
	grp.GET("/api/v1/capabilities", capabilitiesHandler(aggregateCapabilities(a.clusterMod, a.computeMod, a.arthubMod)))
	// The OpenAPI document (same builder as OpenAPISpec) is unauthenticated so
	// clients and gateways can fetch the contract without an identity.
	grp.GET("/openapi.yaml", a.openapiHandler(SpecYAML, "application/yaml"))
	grp.GET("/openapi.json", a.openapiHandler(SpecJSON, "application/json"))

	api := grp.Group("/api/v1", clustermodule.RequireUser())
	a.clusterMod.RegisterRoutes(api)
	a.computeMod.RegisterRoutes(api)
	a.arthubMod.RegisterRoutes(api)
}

// openapiHandler serves the composite OpenAPI document in the given format from
// the same builder as the package-level OpenAPISpec. The bytes are rendered once
// (the surface is static) and cached on the App.
func (a *App) openapiHandler(format SpecFormat, contentType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		data, err := a.openAPISpec(format)
		if err != nil {
			c.String(http.StatusInternalServerError, "openapi spec unavailable: %v", err)
			return
		}
		c.Data(http.StatusOK, contentType, data)
	}
}

func (a *App) openAPISpec(format SpecFormat) ([]byte, error) {
	a.specOnce.Do(func() {
		if a.specYAML, a.specErr = OpenAPISpec(SpecYAML); a.specErr != nil {
			return
		}
		a.specJSON, a.specErr = OpenAPISpec(SpecJSON)
	})
	if a.specErr != nil {
		return nil, a.specErr
	}
	if format == SpecJSON {
		return a.specJSON, nil
	}
	return a.specYAML, nil
}

// Handler returns axisml-core's own gin engine, built once: the middleware
// chain, the probes + capabilities routes and the three System modules under the
// X-Axisml-User gate (i.e. RegisterRoutes on a fresh engine). A host that runs a
// stdlib net/http server — or any non-gin host — mounts this as an opaque
// http.Handler; a host that runs its OWN gin engine uses RegisterRoutes instead
// so axisml and host routes share one engine.
func (a *App) Handler() http.Handler {
	a.engineOnce.Do(func() {
		// Force ReleaseMode only for axisml-core's own engine; a native-gin host
		// that calls RegisterRoutes keeps whatever mode it chose.
		gin.SetMode(gin.ReleaseMode)
		e := gin.New()
		a.RegisterRoutes(e)
		a.engine = e
	})
	return a.engine
}

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

	srv := &http.Server{Addr: a.settings.APIBindAddress, Handler: a.Handler(), ReadHeaderTimeout: 10 * time.Second}

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
