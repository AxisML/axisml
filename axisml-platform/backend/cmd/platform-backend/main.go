// Command axisml-platform-backend is the user-facing entry point for the AxisML
// Platform: it fronts the System-layer services (cluster-manager / compute /
// artifacts) over HTTP and adds identity, RBAC, and orchestration.
//
// Subcommands: serve (HTTP API + probes), migrate (apply DB migrations),
// bootstrap (migrate + seed the initial system-admin and the default tenant).
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/axisml/axisml/components/platform/internal/app"
	"github.com/axisml/axisml/components/platform/internal/config"
)

func main() {
	root := &cobra.Command{
		Use:           "platform",
		Short:         "AxisML Platform Backend",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(serveCmd(), migrateCmd(), bootstrapCmd())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "platform:", err)
		os.Exit(1)
	}
}

func serveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Run the HTTP API + probes",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			return app.Serve(cmd.Context(), cfg)
		},
	}
}

func migrateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "Apply pending DB migrations and exit",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			return app.Migrate(cmd.Context(), cfg)
		},
	}
}

func bootstrapCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "bootstrap",
		Short: "Migrate, then seed the initial system-admin and default tenant",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			return app.Bootstrap(cmd.Context(), cfg)
		},
	}
}
