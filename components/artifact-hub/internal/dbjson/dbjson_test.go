package dbjson

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

func TestMapToJSON_EmptyAndNilRenderAsObject(t *testing.T) {
	for _, in := range []map[string]string{nil, {}} {
		got, err := MapToJSON(in)
		require.NoError(t, err)
		assert.Equal(t, "{}", string(got),
			"empty input must render as an empty JSON object so the column always has a valid jsonb value")
	}
}

func TestMapToJSON_RoundTrip(t *testing.T) {
	in := map[string]string{"a": "1", "b": "two"}
	got, err := MapToJSON(in)
	require.NoError(t, err)
	out := DecodeStringMap(got)
	assert.Equal(t, in, out)
}

func TestDecodeStringMap_Empty(t *testing.T) {
	assert.Nil(t, DecodeStringMap(datatypes.JSON{}))
	assert.Nil(t, DecodeStringMap(nil))
}

func TestDecodeStringMap_InvalidReturnsNil(t *testing.T) {
	assert.Nil(t, DecodeStringMap(datatypes.JSON([]byte(`not-json`))),
		"corrupted column should not crash; decode returns nil")
}

func TestIsUniqueViolation_True(t *testing.T) {
	err := &pgconn.PgError{Code: "23505"}
	assert.True(t, IsUniqueViolation(err))
	wrapped := fmt.Errorf("insert: %w", err)
	assert.True(t, IsUniqueViolation(wrapped),
		"unique-violation detection must walk the error chain")
}

func TestIsUniqueViolation_False(t *testing.T) {
	assert.False(t, IsUniqueViolation(nil))
	assert.False(t, IsUniqueViolation(errors.New("plain")))
	assert.False(t, IsUniqueViolation(&pgconn.PgError{Code: "23502"}))
}
