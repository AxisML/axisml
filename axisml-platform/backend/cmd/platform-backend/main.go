// Command axisml-platform-backend is the user-facing entry point for the AxisML
// Platform: it fronts the System-layer services (cluster-manager / compute /
// artifacts) over HTTP and adds identity, RBAC, and orchestration.
//
// Subcommands: serve (HTTP API + probes), migrate (apply DB migrations),
// bootstrap (migrate + seed the initial system-admin and import the System-defined tenants).
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/axisml/axisml/axisml-platform/backend/internal/app"
	"github.com/axisml/axisml/axisml-platform/backend/internal/config"
	"github.com/axisml/axisml/pkg/axismlconfig"
)

func main() {
	var cfgFile string

	root := &cobra.Command{
		Use:           "platform",
		Short:         "AxisML Platform Backend",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().StringVar(&cfgFile, "config", "",
		"path to the YAML config file (default: $AXISML_CONFIG or /etc/axisml/config.yaml)")

	load := func() (config.Config, error) {
		return config.Load(axismlconfig.Options{File: cfgFile})
	}

	serve := &cobra.Command{
		Use:   "serve",
		Short: "Run the HTTP API + probes",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := load()
			if err != nil {
				return err
			}
			return app.Serve(cmd.Context(), cfg)
		},
	}
	migrate := &cobra.Command{
		Use:   "migrate",
		Short: "Apply pending DB migrations and exit",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := load()
			if err != nil {
				return err
			}
			return app.Migrate(cmd.Context(), cfg)
		},
	}
	bootstrap := &cobra.Command{
		Use:   "bootstrap",
		Short: "Migrate, then seed the initial system-admin and import the System-defined tenants",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := load()
			if err != nil {
				return err
			}
			return app.Bootstrap(cmd.Context(), cfg)
		},
	}

	root.AddCommand(serve, migrate, bootstrap)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "platform:", err)
		os.Exit(1)
	}
}
