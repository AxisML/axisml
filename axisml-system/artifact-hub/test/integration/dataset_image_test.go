//go:build integration

package integration_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	artmod "github.com/axisml/axisml/components/artifact-hub/internal/artifact"
)

// TestDataset_HappyPath drives Initiate → Get for the dataset Kind.
// Dataset uses an S3 URI; the MVP handler signs a placeholder credential
// (no live STS), so we exercise the registry + ValidateSpec + persistence
// path without driving fakeZot.
func TestDataset_HappyPath(t *testing.T) {
	s := setup(t)
	s.resetState(t)

	body := map[string]any{
		"version": "v1",
		"spec": map[string]any{
			"format": "parquet",
		},
		"displayName": "demo dataset",
	}
	rr := s.drive(t, http.MethodPost, s.nsPath("/datasets/sample-set"), body)
	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())

	var init map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &init))
	upload := init["upload"].(map[string]any)
	assert.Equal(t, "s3", upload["storageKind"])
	assert.Contains(t, upload["uri"], "s3://axisml-artifact-hub/namespaces/"+s.namespace+"/datasets/sample-set/v1/")

	rr = s.drive(t, http.MethodGet, s.nsPath("/datasets/sample-set/v1"), nil)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	var got map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	assert.Equal(t, "dataset", got["kind"])
	assert.Equal(t, artmod.StatusUploading, got["status"])
}

// TestImage_HappyPath drives Initiate for the image Kind. Like model, image
// uses the OCI client → exercise spec.purpose validation + URI shape.
func TestImage_HappyPath(t *testing.T) {
	s := setup(t)
	s.resetState(t)

	body := map[string]any{
		"version": "v1",
		"spec": map[string]any{
			"purpose": "training",
		},
	}
	rr := s.drive(t, http.MethodPost, s.nsPath("/images/builder"), body)
	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())

	var init map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &init))
	upload := init["upload"].(map[string]any)
	assert.Equal(t, "oci", upload["storageKind"])
	assert.Contains(t, upload["uri"], "/namespaces/"+s.namespace+"/images/builder:v1")
}

// TestImage_InvalidPurpose rejects an unknown purpose with 4xx.
func TestImage_InvalidPurpose(t *testing.T) {
	s := setup(t)
	s.resetState(t)
	rr := s.drive(t, http.MethodPost, s.nsPath("/images/bad"), map[string]any{
		"version": "v1",
		"spec":    map[string]any{"purpose": "wat"},
	})
	require.GreaterOrEqual(t, rr.Code, 400)
	require.Less(t, rr.Code, 500)
}

// TestDataset_InvalidFormat rejects an unknown format with 4xx.
func TestDataset_InvalidFormat(t *testing.T) {
	s := setup(t)
	s.resetState(t)
	rr := s.drive(t, http.MethodPost, s.nsPath("/datasets/bad"), map[string]any{
		"version": "v1",
		"spec":    map[string]any{"format": "wat"},
	})
	require.GreaterOrEqual(t, rr.Code, 400)
}
