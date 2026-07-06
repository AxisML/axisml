//go:build integration

// Package integration_test drives the real Platform API engine in-process via
// httptest against a testcontainers Postgres and an in-memory cluster-manager
// stub. No Kubernetes / envtest is needed — Platform holds no K8s client.
package integration

import (
	"context"
	"fmt"
	"log/slog"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/axisml/axisml/axisml-platform/backend/internal/app"
	"github.com/axisml/axisml/axisml-platform/backend/internal/config"
	"github.com/axisml/axisml/axisml-platform/backend/internal/db"
	"github.com/axisml/axisml/axisml-platform/backend/internal/store"
	"github.com/axisml/axisml/pkg/axismlconfig"
)

var (
	testEngine  *gin.Engine
	cmStub      *clusterManagerStub
	computeStub *computeServiceStub
	artStub     *artifactHubStub
)

func TestMain(m *testing.M) {
	code, err := run(m)
	if err != nil {
		fmt.Fprintln(os.Stderr, "integration setup:", err)
		os.Exit(1)
	}
	os.Exit(code)
}

func run(m *testing.M) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	ctr, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("axisml"),
		tcpostgres.WithUsername("axisml"),
		tcpostgres.WithPassword("axisml"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		return 0, err
	}
	defer func() { _ = ctr.Terminate(context.Background()) }()

	host, err := ctr.Host(ctx)
	if err != nil {
		return 0, err
	}
	port, err := ctr.MappedPort(ctx, "5432/tcp")
	if err != nil {
		return 0, err
	}

	cmStub = newClusterManagerStub()
	// The System chart's seed.tenant owns the built-in `default` Tenant CR before
	// Platform installs; mirror that so bootstrap's importDefaultTenant discovers
	// and imports it.
	cmStub.seedTenant(config.DefaultTenant, config.DefaultTenant)
	stubSrv := httptest.NewServer(cmStub)
	defer stubSrv.Close()

	computeStub = newComputeServiceStub()
	computeSrv := httptest.NewServer(computeStub)
	defer computeSrv.Close()

	artStub = newArtifactHubStub()
	artSrv := httptest.NewServer(artStub)
	defer artSrv.Close()

	cfg := config.Config{
		Common: axismlconfig.Common{
			Database: axismlconfig.Database{
				Host: host, Port: port.Int(), Name: "axisml",
				User: "axisml", Password: "axisml", SSLMode: "disable",
			},
			Log: axismlconfig.Log{Level: "info", Format: "console"},
		},
		System:    config.System{ClusterManager: stubSrv.URL, ComputeService: computeSrv.URL, ArtifactHub: artSrv.URL},
		Auth:      config.Auth{LoginTokenTTL: time.Hour},
		Bootstrap: config.Bootstrap{Username: "admin", Password: "admin"},
	}

	gormDB, err := db.Open(cfg)
	if err != nil {
		return 0, err
	}
	if err := app.Bootstrap(ctx, cfg); err != nil {
		return 0, fmt.Errorf("bootstrap: %w", err)
	}
	// The bootstrap admin is seeded with must_change_password, and the server now
	// enforces that gate on every protected route. Clear it so the suite drives
	// the API as a fully-onboarded admin (mirrors a real admin completing the
	// forced password change before using the platform); the admin/admin
	// credentials used by loginAdmin stay valid.
	if err := gormDB.Model(&store.User{}).Where("username = ?", cfg.Bootstrap.Username).
		Update("must_change_password", false).Error; err != nil {
		return 0, fmt.Errorf("clear admin password gate: %w", err)
	}

	srv, err := app.NewAPIServer(cfg, gormDB, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	if err != nil {
		return 0, err
	}
	testEngine = srv.Engine()

	return m.Run(), nil
}
