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
	defer func() { _ = ctr.Container.Terminate(context.Background()) }()

	host, err := ctr.Host(ctx)
	if err != nil {
		return 0, err
	}
	port, err := ctr.MappedPort(ctx, "5432/tcp")
	if err != nil {
		return 0, err
	}

	cmStub = newClusterManagerStub()
	stubSrv := httptest.NewServer(cmStub)
	defer stubSrv.Close()

	cfg := config.Config{
		DatabaseHost:      host,
		DatabasePort:      port.Int(),
		DatabaseName:      "axisml",
		DatabaseUser:      "axisml",
		DatabasePassword:  "axisml",
		DatabaseSSLMode:   "disable",
		APIBindAddress:    ":0",
		ClusterManagerURL: stubSrv.URL,
		UpstreamTimeout:   10 * time.Second,
		JWTKeyID:          "test-key",
		LoginTokenTTL:     time.Hour,
		BootstrapUsername: "admin",
		BootstrapPassword: "admin",
		BootstrapTenant:   "default",
		BootstrapTenantNS: "axisml-tenant",
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
