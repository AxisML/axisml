// Command axisml-platform-backend is the user-facing entry point for the
// AxisML Platform: it fronts the internal services (cluster-manager / compute /
// artifacts) over HTTP and adds identity, RBAC, and orchestration.
//
// The server is not implemented yet — see internal/app for the current state
// and docs/openapi/platform.yaml for the API it will serve. This entry point is
// in place so the component builds and ships like its siblings.
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/axisml/axisml/components/platform/backend/internal/app"
)

func main() {
	cfg := app.DefaultConfig()
	flag.StringVar(&cfg.APIBindAddress, "api-bind-address", cfg.APIBindAddress, "REST API listen address.")
	flag.StringVar(&cfg.ProbesBindAddress, "probes-bind-address", cfg.ProbesBindAddress, "Health probe listen address (/healthz, /readyz).")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx, cfg); err != nil {
		slog.Error("platform-backend exited with error", "err", err)
		os.Exit(1)
	}
}
