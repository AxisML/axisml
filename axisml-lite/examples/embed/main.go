package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/axisml/axisml/axisml-lite/axisml-core/pkg/core"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := core.Config{}
	cfg.Database.Host = envOr("AXISML_DATABASE_HOST", "localhost")
	cfg.Database.Port = envInt("AXISML_DATABASE_PORT", 5432)
	cfg.Database.Name = envOr("AXISML_DATABASE_NAME", "axisml")
	cfg.Database.User = envOr("AXISML_DATABASE_USER", "axisml")
	cfg.Database.Password = envOr("AXISML_DATABASE_PASSWORD", "axisml")
	cfg.Database.SSLMode = envOr("AXISML_DATABASE_SSLMODE", "disable")
	cfg.Log.Level = envOr("AXISML_LOG_LEVEL", "info")
	cfg.Log.Format = envOr("AXISML_LOG_FORMAT", "console")
	cfg.OCI.Endpoint = envOr("AXISML_OCI_ENDPOINT", "http://localhost:5000")
	cfg.OCI.AdminUser = envOr("AXISML_OCI_ADMIN_USER", "admin")
	cfg.OCI.AdminPassword = envOr("AXISML_OCI_ADMIN_PASSWORD", "admin")

	settings := core.DefaultSettings()
	settings.PoolConfigDir = envOr("AXISML_POOL_CONFIG_DIR", "config")
	settings.GatewayConfigDir = envOr("AXISML_GATEWAY_CONFIG_DIR", filepath.Join(os.TempDir(), "axisml-core-embed", "traefik"))
	settings.WorkloadsNetwork = envOr("AXISML_WORKLOADS_NETWORK", "axisml-core-embed")

	app, err := core.New(ctx, cfg, core.WithSettings(settings))
	if err != nil {
		return err
	}
	defer func() { _ = app.Close() }()

	if err := app.Migrate(); err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.Handle("/axisml/", http.StripPrefix("/axisml", app.Handler()))
	mux.HandleFunc("/host/ping", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("pong from the host application\n"))
	})

	for _, r := range app.Runnables() {
		go func(run core.Runnable) {
			if err := run.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
				log.Printf("axisml-core background loop exited: %v", err)
			}
		}(r)
	}

	addr := envOr("ADDR", ":9090")
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("host app listening on %s; axisml-core is mounted under /axisml", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("host server failed: %v", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		log.Printf("invalid %s=%q, using %d", key, v, fallback)
		return fallback
	}
	return n
}
