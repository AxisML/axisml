// Package app wires the platform-backend process: the HTTP API server (auth +
// RBAC + business modules), the metrics and health-probe servers, and graceful
// shutdown.
//
// Platform is the user-facing aggregator: it fronts the System-layer services
// (cluster-manager / compute / artifacts) over HTTP and adds identity, RBAC,
// and orchestration. It holds no Kubernetes client and runs no reconciler.
package app

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/axisml/axisml/components/platform/internal/artifactdef"
	"github.com/axisml/axisml/components/platform/internal/auth"
	"github.com/axisml/axisml/components/platform/internal/cache"
	"github.com/axisml/axisml/components/platform/internal/clients/artifacthub"
	"github.com/axisml/axisml/components/platform/internal/clients/clustermanager"
	"github.com/axisml/axisml/components/platform/internal/clients/computeservice"
	"github.com/axisml/axisml/components/platform/internal/config"
	"github.com/axisml/axisml/components/platform/internal/datavolume"
	"github.com/axisml/axisml/components/platform/internal/experiment"
	"github.com/axisml/axisml/components/platform/internal/identity"
	"github.com/axisml/axisml/components/platform/internal/job"
	"github.com/axisml/axisml/components/platform/internal/metrics"
	"github.com/axisml/axisml/components/platform/internal/mlservice"
	"github.com/axisml/axisml/components/platform/internal/resourcepool"
	"github.com/axisml/axisml/components/platform/internal/server"
	"github.com/axisml/axisml/components/platform/internal/store"
	"github.com/axisml/axisml/components/platform/internal/tenant"
	"github.com/axisml/axisml/components/platform/internal/traffic"
	"github.com/axisml/axisml/components/platform/internal/workspace"
)

// Deps groups the constructed collaborators shared across modules.
type Deps struct {
	Authn   *auth.Authenticator
	Signer  *auth.Signer
	Modules []server.Module
	Cache   cache.Cache
}

// BuildDeps constructs the auth stack, typed clients, services and modules from
// a live DB handle and config. Shared by Serve and the integration tests.
func BuildDeps(cfg config.Config, db *gorm.DB, log *slog.Logger) (*Deps, error) {
	signer, err := auth.NewSigner(cfg.Auth.JWTPrivateKeyPEM, cfg.Auth.LoginTokenTTL)
	if err != nil {
		return nil, err
	}

	users := store.NewUserRepo(db)
	roles := store.NewRoleRepo(db)
	sessions := store.NewSessionRepo(db)
	tenants := store.NewTenantRepo(db)
	idp := store.NewIdentityProvider(db)

	// Front the auth hot path (session validity + identity/RBAC) with Redis when
	// configured; both decorators fall back to PostgreSQL on a miss or error.
	c := cache.New(cfg, log)
	sessionStore := cache.NewSessionCache(sessions, c, config.SessionCacheTTL, log)
	identityStore := cache.NewIdentityCache(idp, c, config.IdentityCacheTTL, log)

	authn := auth.NewAuthenticator(signer, identityStore, sessionStore)

	cm, err := clustermanager.New(cfg.System.ClusterManager, config.UpstreamTimeout)
	if err != nil {
		return nil, err
	}
	compute, err := computeservice.New(cfg.System.ComputeService, config.UpstreamTimeout)
	if err != nil {
		return nil, err
	}
	artifacts, err := artifacthub.New(cfg.System.ArtifactHub, config.UpstreamTimeout)
	if err != nil {
		return nil, err
	}

	identitySvc := identity.NewService(users, roles, sessionStore, idp, signer).
		OnIdentityChange(identityStore.Invalidate)
	tenantSvc := tenant.NewService(tenants, roles, users, cm, compute).
		OnIdentityChange(identityStore.Invalidate)
	resourcePoolSvc := resourcepool.NewService(cm)
	dataVolumeSvc := datavolume.NewService(cm, tenants)
	jobSvc := job.NewService(store.NewDefinitionRepo(db, store.TableJobs), tenants, compute)
	experimentSvc := experiment.NewService(store.NewDefinitionRepo(db, store.TableExperiments), tenants, compute)
	mlserviceSvc := mlservice.NewService(compute, tenants)
	workspaceSvc := workspace.NewService(compute, tenants)
	trafficSvc := traffic.NewService(compute, tenants)
	modelSvc := artifactdef.NewService(store.NewDefinitionRepo(db, store.TableModels), artifacts, "model", config.DefaultTenant)
	imageSvc := artifactdef.NewService(store.NewDefinitionRepo(db, store.TableImages), artifacts, "image", config.DefaultTenant)

	modules := []server.Module{
		identity.NewHandler(identitySvc, authn),
		tenant.NewHandler(tenantSvc, authn),
		resourcepool.NewHandler(resourcePoolSvc, authn),
		datavolume.NewHandler(dataVolumeSvc, authn),
		job.NewHandler(jobSvc, authn),
		experiment.NewHandler(experimentSvc, authn),
		mlservice.NewHandler(mlserviceSvc, authn),
		workspace.NewHandler(workspaceSvc, authn),
		traffic.NewHandler(trafficSvc, authn),
		artifactdef.NewHandler(modelSvc, authn, "models"),
		artifactdef.NewHandler(imageSvc, authn, "images"),
	}
	return &Deps{Authn: authn, Signer: signer, Modules: modules, Cache: c}, nil
}

// NewAPIServer builds the API server (modules + JWKS + probes).
func NewAPIServer(cfg config.Config, db *gorm.DB, log *slog.Logger) (*server.Server, error) {
	if cfg.Auth.JWTPrivateKeyPEM == "" {
		log.Warn("auth.jwt_private_key_pem is unset: signing with an ephemeral key. " +
			"All sessions are invalidated on restart and multi-replica deployments will fail to verify each other's tokens. " +
			"Inject a stable RSA key in production.")
	}
	deps, err := BuildDeps(cfg, db, log)
	if err != nil {
		return nil, err
	}
	jwks := deps.Signer.JWKS()
	return server.New(server.Options{
		Addr:        config.APIBindAddress,
		Log:         log,
		Modules:     deps.Modules,
		JWKSHandler: func(c *gin.Context) { c.JSON(http.StatusOK, jwks) },
		Ready: func(ctx context.Context) error {
			sqlDB, err := db.DB()
			if err != nil {
				return err
			}
			return sqlDB.PingContext(ctx)
		},
	})
}

// NewProbeRouter serves liveness/readiness on the dedicated probes port.
func NewProbeRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	ok := func(c *gin.Context) { c.JSON(http.StatusOK, server.HealthStatus{Status: "ok"}) }
	r.GET("/healthz", ok)
	r.GET("/readyz", ok)
	return r
}

func probeServer(addr string) *http.Server {
	return &http.Server{Addr: addr, Handler: NewProbeRouter(), ReadHeaderTimeout: 10 * time.Second}
}

// metricsServer serves Prometheus metrics on the dedicated metrics port.
func metricsServer(addr string) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", metrics.Handler())
	return &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}
}
