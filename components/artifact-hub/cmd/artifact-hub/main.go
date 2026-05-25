package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/axisml/axisml/components/artifact-hub/internal/app"
	"github.com/axisml/axisml/components/artifact-hub/internal/config"
)

func main() {
	root := &cobra.Command{
		Use:           "artifacts",
		Short:         "AxisML Artifact Hub",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(serveCmd(), migrateCmd())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "artifacts:", err)
		os.Exit(1)
	}
}

func serveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Run the HTTP API + GC worker",
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
