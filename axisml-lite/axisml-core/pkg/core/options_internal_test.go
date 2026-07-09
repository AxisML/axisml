package core

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// The Option functions mutate the unexported options struct, so this is an
// internal (package core) test.
func TestOptions_Apply(t *testing.T) {
	o := &options{}

	WithSettings(Settings{APIBindAddress: ":9999", WorkloadsNetwork: "custom-net"})(o)
	assert.Equal(t, ":9999", o.settings.APIBindAddress)
	assert.Equal(t, "custom-net", o.settings.WorkloadsNetwork)

	db := &gorm.DB{}
	WithDB(db)(o)
	assert.Same(t, db, o.db)

	sc := &StaticConfig{}
	WithStaticConfig(sc)(o)
	assert.Same(t, sc, o.static)

	lg := logr.Discard()
	WithLogger(lg)(o)
	require.NotNil(t, o.logger)
	assert.Equal(t, lg, *o.logger)
}
