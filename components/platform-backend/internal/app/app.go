// Package app wires the platform-backend process: the HTTP API server (auth +
// RBAC + business modules), the health-probe server, and graceful shutdown.
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
	"github.com/axisml/axisml/components/platform/internal/clients/artifacthub"
	"github.com/axisml/axisml/components/platform/internal/clients/clustermanager"
	"github.com/axisml/axisml/components/platform/internal/clients/computeservice"
	"github.com/axisml/axisml/components/platform/internal/config"
	"github.com/axisml/axisml/components/platform/internal/experiment"
	"github.com/axisml/axisml/components/platform/internal/identity"
	"github.com/axisml/axisml/components/platform/internal/job"
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
}

// BuildDeps constructs the auth stack, typed clients, services and modules from
// a live DB handle and config. Shared by Serve and the integration tests.
func BuildDeps(cfg config.Config, db *gorm.DB) (*Deps, error) {
	signer, err := auth.NewSigner(cfg.JWTPrivateKeyPEM, cfg.JWTKeyID, cfg.LoginTokenTTL)
	if err != nil {
		return nil, err
	}

	users := store.NewUserRepo(db)
	roles := store.NewRoleRepo(db)
	sessions := store.NewSessionRepo(db)
	tenants := store.NewTenantRepo(db)
	idp := store.NewIdentityProvider(db)

	authn := auth.NewAuthenticator(signer, idp, sessions)

	cm, err := clustermanager.New(cfg.ClusterManagerURL, cfg.UpstreamTimeout)
	if err != nil {
		return nil, err
	}
	compute, err := computeservice.New(cfg.ComputeURL, cfg.UpstreamTimeout)
	if err != nil {
		return nil, err
	}
	artifacts, err := artifacthub.New(cfg.ArtifactsURL, cfg.UpstreamTimeout)
	if err != nil {
		return nil, err
	}

	identitySvc := identity.NewService(users, roles, sessions, idp, signer)
	tenantSvc := tenant.NewService(tenants, roles, users, cm)
	resourcePoolSvc := resourcepool.NewService(cm)
	jobSvc := job.NewService(store.NewDefinitionRepo(db, store.TableJobs), tenants, compute)
	experimentSvc := experiment.NewService(store.NewDefinitionRepo(db, store.TableExperiments), tenants, compute)
	mlserviceSvc := mlservice.NewService(compute, tenants)
	workspaceSvc := workspace.NewService(compute, tenants)
	trafficSvc := traffic.NewService(compute, tenants)
	modelSvc := artifactdef.NewService(store.NewDefinitionRepo(db, store.TableModels), artifacts, "model", cfg.PublicTenantScope)
	imageSvc := artifactdef.NewService(store.NewDefinitionRepo(db, store.TableImages), artifacts, "image", cfg.PublicTenantScope)

	modules := []server.Module{
		identity.NewHandler(identitySvc, authn),
		tenant.NewHandler(tenantSvc, authn),
		resourcepool.NewHandler(resourcePoolSvc, authn),
		job.NewHandler(jobSvc, authn),
		experiment.NewHandler(experimentSvc, authn),
		mlservice.NewHandler(mlserviceSvc, authn),
		workspace.NewHandler(workspaceSvc, authn),
		traffic.NewHandler(trafficSvc, authn),
		artifactdef.NewHandler(modelSvc, authn, "models"),
		artifactdef.NewHandler(imageSvc, authn, "images"),
	}
	return &Deps{Authn: authn, Signer: signer, Modules: modules}, nil
}

// NewAPIServer builds the API server (modules + JWKS + probes).
func NewAPIServer(cfg config.Config, db *gorm.DB, log *slog.Logger) (*server.Server, error) {
	if cfg.JWTPrivateKeyPEM == "" {
		log.Warn("JWT_PRIVATE_KEY_PEM is unset: signing with an ephemeral key. " +
			"All sessions are invalidated on restart and multi-replica deployments will fail to verify each other's tokens. " +
			"Inject a stable RSA key in production.")
	}
	deps, err := BuildDeps(cfg, db)
	if err != nil {
		return nil, err
	}
	jwks := deps.Signer.JWKS()
	return server.New(server.Options{
		Addr:        cfg.APIBindAddress,
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
