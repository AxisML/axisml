package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/axisml/axisml/components/compute-service/internal/app"
	"github.com/axisml/axisml/components/compute-service/internal/config"
)

func main() {
	root := &cobra.Command{
		Use:           "compute",
		Short:         "AxisML Compute Service",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(serveCmd(), bootstrapCmd(), migrateCmd())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "compute:", err)
		os.Exit(1)
	}
}

func serveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Run the HTTP API + reconciler/informer loops",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			return app.Serve(cmd.Context(), cfg)
		},
	}
}

func bootstrapCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "bootstrap",
		Short: "Idempotently seed default tenant / pool / unit / quota (Helm post-install)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			return app.Bootstrap(cmd.Context(), cfg)
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
