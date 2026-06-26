package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/axisml/axisml/components/artifact-hub/internal/app"
	"github.com/axisml/axisml/components/artifact-hub/internal/config"
	"github.com/axisml/axisml/pkg/axismlconfig"
)

func main() {
	var cfgFile string

	root := &cobra.Command{
		Use:           "artifacts",
		Short:         "AxisML Artifact Hub",
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
		Short: "Run the HTTP API + GC worker",
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

	root.AddCommand(serve, migrate)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "artifacts:", err)
		os.Exit(1)
	}
}
