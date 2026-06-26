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

	"github.com/axisml/axisml/components/platform/internal/app"
	"github.com/axisml/axisml/components/platform/internal/config"
	"github.com/axisml/axisml/components/platform/internal/db"
	"github.com/axisml/axisml/pkg/axismlconfig"
)

var (
	testEngine *gin.Engine
	cmStub     *clusterManagerStub
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

	cfg := config.Config{
		Common: axismlconfig.Common{
			Database: axismlconfig.Database{
				Host: host, Port: port.Int(), Name: "axisml",
				User: "axisml", Password: "axisml", SSLMode: "disable",
			},
			Log: axismlconfig.Log{Level: "info", Format: "console"},
		},
		System:    config.System{ClusterManager: stubSrv.URL},
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

	srv, err := app.NewAPIServer(cfg, gormDB, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	if err != nil {
		return 0, err
	}
	testEngine = srv.Engine()

	return m.Run(), nil
}
