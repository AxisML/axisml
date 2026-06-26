// Command axisml-core is the AxisML Lite Standalone composition root. It hosts
// the Cluster Manager, Compute Service and Artifact Hub modules on one HTTP
// server and the in-process Standalone (Docker) Compute Runtime, the only
// process with access to the Docker socket (design §3.1).
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/axisml/axisml/axisml-lite/axisml-core/internal/core"
)

func main() {
	root := &cobra.Command{
		Use:   "axisml-core",
		Short: "AxisML Lite Standalone control plane (Cluster Manager + Compute + Artifact Hub + Standalone Runtime)",
	}
	root.AddCommand(serveCmd(), migrateCmd())
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func serveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Run the axisml-core HTTP server, reconcilers and status pollers",
		RunE: func(_ *cobra.Command, _ []string) error {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			cfg, err := core.Load()
			if err != nil {
				return err
			}
			return core.Run(ctx, cfg)
		},
	}
}

func migrateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "Run database migrations and exit",
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := core.Load()
			if err != nil {
				return err
			}
			return core.Migrate(cfg)
		},
	}
}
