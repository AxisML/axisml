// Package config loads Tenant-controller runtime knobs from environment
// variables. These knobs are surfaced through Helm values
// (deploy/helm/axisml-system/values.yaml -> operator.controllers.tenant).
package config

import (
	"os"
	"strings"
	"time"

	"github.com/axisml/axisml/components/tenant-operator/internal/validate"
)

// Defaults match the recommendations in tenant-operator design §5 and §6.1.
const (
	defaultResyncPeriod = 10 * time.Minute
)

// Config holds resolved runtime configuration.
type Config struct {
	ResyncPeriod      time.Duration
	NamespaceDenylist map[string]struct{}
}

// Load reads RESYNC_PERIOD and NAMESPACE_DENYLIST from the environment.
// Missing or unparsable values fall back to the documented defaults.
func Load() Config {
	cfg := Config{
		ResyncPeriod:      defaultResyncPeriod,
		NamespaceDenylist: validate.DefaultNamespaceDenylist(),
	}

	if v := os.Getenv("RESYNC_PERIOD"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			cfg.ResyncPeriod = d
		}
	}

	if v := os.Getenv("NAMESPACE_DENYLIST"); v != "" {
		cfg.NamespaceDenylist = parseDenylist(v)
	}

	return cfg
}

func parseDenylist(s string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, item := range strings.Split(s, ",") {
		name := strings.TrimSpace(item)
		if name != "" {
			out[name] = struct{}{}
		}
	}
	return out
}
