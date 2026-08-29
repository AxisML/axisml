package configutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type metadataConfig struct {
	Database struct {
		Password string `mapstructure:"password" secret:"true"`
	} `mapstructure:"database"`
}

func TestWalkUsesSelectedPrefixInFieldMetadata(t *testing.T) {
	fields := Walk(&metadataConfig{}, "AIOSML")
	require.Len(t, fields, 1)
	assert.Equal(t, "database.password", fields[0].Path)
	assert.Equal(t, "AIOSML_DATABASE_PASSWORD", fields[0].EnvVar)
}
