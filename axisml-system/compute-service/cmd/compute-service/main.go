package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/axisml/axisml/axisml-system/compute-service/internal/app"
	"github.com/axisml/axisml/axisml-system/compute-service/internal/config"
)

func main() {
	var cfgFile string

	root := &cobra.Command{
		Use:           "compute",
		Short:         "AxisML Compute Service",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().StringVar(&cfgFile, "config", "",
		"path to the YAML config file (default: $AXISML_CONFIG or /etc/axisml/config.yaml)")

	load := func() (config.Config, error) {
		return config.Load(config.Options{File: cfgFile})
	}

	serve := &cobra.Command{
		Use:   "serve",
		Short: "Run the HTTP API + reconciler/informer loops",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := load()
			if err != nil {
				return err
			}
			return app.Serve(cmd.Context(), cfg)
		},
	}
	bootstrap := &cobra.Command{
		Use:   "bootstrap",
		Short: "Run database migrations (Helm post-install)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := load()
			if err != nil {
				return err
			}
			return app.Bootstrap(cmd.Context(), cfg)
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

	root.AddCommand(serve, bootstrap, migrate)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "compute:", err)
		os.Exit(1)
	}
}
