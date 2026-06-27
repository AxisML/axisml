package core

import (
	"github.com/go-logr/logr"
	"gorm.io/gorm"
)

// options is the resolved set of New inputs beyond Config. The zero value is
// never used directly — New seeds it with DefaultSettings before applying the
// caller's Option list.
type options struct {
	settings Settings
	db       *gorm.DB
	static   *StaticConfig
	logger   *logr.Logger
}

// Option customizes how New assembles the App. The Lite binary passes none and
// gets the fixed-constant behaviour; an embedding host uses these to inject its
// own database, in-memory static config, logger or operational Settings.
type Option func(*options)

// WithSettings overrides the operational parameters (bind address, filesystem
// layout, Docker network, background cadences). Without it New uses
// DefaultSettings.
func WithSettings(s Settings) Option {
	return func(o *options) { o.settings = s }
}

// WithDB supplies an already-open *gorm.DB instead of letting New open one from
// Config. The caller retains ownership: App.Close will NOT close an injected DB.
func WithDB(db *gorm.DB) Option {
	return func(o *options) { o.db = db }
}

// WithStaticConfig supplies the single default ResourcePool + Tenant in memory
// instead of reading them from Settings.PoolConfigDir. The supplied config is
// validated by New the same way the on-disk form is.
func WithStaticConfig(sc *StaticConfig) Option {
	return func(o *options) { o.static = sc }
}

// WithLogger supplies the logger instead of building one from Config.Log.
func WithLogger(log logr.Logger) Option {
	return func(o *options) { o.logger = &log }
}
