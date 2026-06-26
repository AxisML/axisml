package app

import (
	"github.com/go-logr/logr"
	"gorm.io/gorm"

	"github.com/axisml/axisml/components/artifact-hub/internal/config"
	"github.com/axisml/axisml/components/artifact-hub/internal/server"
	artifactmodule "github.com/axisml/axisml/components/artifact-hub/pkg/module"
)

// BuildModules is the composition root: it maps the binary's config onto the
// shared pkg/module assembly and returns the HTTP API modules plus the
// background GC worker. The worker is returned to the caller, which gates it
// behind PostgreSQL advisory-lock leader election.
func BuildModules(
	cfg config.Config,
	gormDB *gorm.DB,
	log logr.Logger,
) ([]server.Module, artifactmodule.Runnable, server.Capabilities, error) {
	mod, err := artifactmodule.New(artifactmodule.Deps{
		DB:  gormDB,
		Log: log,
		Config: artifactmodule.Config{
			// OCI scheme is derived from the endpoint URL by the client.
			OCIEndpoint:      cfg.OCI.Endpoint,
			OCIAdminUser:     cfg.OCI.AdminUser,
			OCIAdminPassword: cfg.OCI.AdminPassword,
			GCInterval:       config.GCInterval,
			UploadingTTL:     config.UploadingTTL,
			UploadTokenTTL:   config.UploadTokenTTL,
		},
	})
	if err != nil {
		return nil, nil, server.Capabilities{}, err
	}

	modules := make([]server.Module, 0, len(mod.Routes()))
	for _, r := range mod.Routes() {
		modules = append(modules, r)
	}

	var worker artifactmodule.Runnable
	if rs := mod.Runnables(); len(rs) > 0 {
		worker = rs[0]
	}
	return modules, worker, mod.Capabilities(), nil
}
