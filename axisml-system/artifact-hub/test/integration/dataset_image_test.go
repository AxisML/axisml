//go:build integration

package integration_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	artmod "github.com/axisml/axisml/axisml-system/artifact-hub/internal/artifact"
)

// TestDataset_HappyPath drives Initiate → Get for the dataset Kind.
// Dataset uses an S3 URI; the MVP handler signs a placeholder credential
// (no live STS), so we exercise the registry + ValidateSpec + persistence
// path without driving fakeZot.
func TestDataset_HappyPath(t *testing.T) {
	s := setup(t)
	s.resetState(t)

	body := map[string]any{
		"kind":    "dataset",
		"version": "v1",
		"spec": map[string]any{
			"format": "parquet",
		},
		"displayName": "demo dataset",
	}
	rr := s.drive(t, http.MethodPost, s.nsPath("/artifacts/sample-set"), body)
	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())

	var init map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &init))
	upload := init["upload"].(map[string]any)
	assert.Equal(t, "s3", upload["storageKind"])
	assert.Contains(t, upload["uri"], "s3://axisml-artifact-hub/namespaces/"+s.namespace+"/datasets/sample-set/v1/")

	rr = s.drive(t, http.MethodGet, s.nsPath("/artifacts/sample-set/v1"), nil)
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
		"kind":    "image",
		"version": "v1",
		"spec": map[string]any{
			"purpose": "training",
		},
	}
	rr := s.drive(t, http.MethodPost, s.nsPath("/artifacts/builder"), body)
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
	rr := s.drive(t, http.MethodPost, s.nsPath("/artifacts/bad"), map[string]any{
		"kind":    "image",
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
	rr := s.drive(t, http.MethodPost, s.nsPath("/artifacts/bad"), map[string]any{
		"kind":    "dataset",
		"version": "v1",
		"spec":    map[string]any{"format": "wat"},
	})
	require.GreaterOrEqual(t, rr.Code, 400)
}

// datasetInitiate initiates a dataset version and returns nothing (asserts 201).
func datasetInitiate(t *testing.T, s *suite, name, version string) {
	t.Helper()
	rr := s.drive(t, http.MethodPost, s.nsPath("/artifacts/"+name), map[string]any{
		"kind":    "dataset",
		"version": version,
		"spec":    map[string]any{"format": "parquet"},
	})
	require.Equalf(t, http.StatusCreated, rr.Code, "initiate dataset: %s", rr.Body.String())
}

// TestDataset_CompleteVerifiesManifest drives the dataset two-phase write
// against the real MinIO backend: initiate → PUT artifact-manifest.json →
// complete with its sha256 → Ready, and resolve echoes the digest.
func TestDataset_CompleteVerifiesManifest(t *testing.T) {
	s := setup(t)
	s.resetState(t)

	const name, version = "verified-set", "v1"
	datasetInitiate(t, s, name, version)

	digest := s.putDatasetManifest(t, name, version, []byte(`{"files":[{"path":"part-0.parquet","size":42}]}`))

	rr := s.drive(t, http.MethodPost,
		s.nsPath("/artifacts/"+name+"/"+version+"/complete"),
		map[string]any{"digest": digest})
	require.Equalf(t, http.StatusOK, rr.Code, "complete: %s", rr.Body.String())

	rr = s.drive(t, http.MethodGet, s.nsPath("/artifacts/"+name+"/"+version), nil)
	require.Equal(t, http.StatusOK, rr.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	assert.Equal(t, artmod.StatusReady, got["status"])
	assert.Equal(t, digest, got["digest"])

	// resolve echoes the verified digest (S3 prefix uri, not digest-pinned).
	rr = s.drive(t, http.MethodGet, s.nsPath("/artifacts/"+name+"/"+version+"/resolve?usage=inspect"), nil)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	var res map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &res))
	assert.Equal(t, "s3", res["storageKind"])
	assert.Equal(t, digest, res["digest"])
}

// TestDataset_CompleteDigestMismatch makes the claim disagree with the stored
// manifest; the service must mark the row Failed and return 409.
func TestDataset_CompleteDigestMismatch(t *testing.T) {
	s := setup(t)
	s.resetState(t)

	const name, version = "skewed-set", "v1"
	datasetInitiate(t, s, name, version)
	s.putDatasetManifest(t, name, version, []byte(`{"files":[]}`))

	rr := s.drive(t, http.MethodPost,
		s.nsPath("/artifacts/"+name+"/"+version+"/complete"),
		map[string]any{"digest": fakeDigest("wrong-claim")})
	require.Equal(t, http.StatusConflict, rr.Code, rr.Body.String())

	rr = s.drive(t, http.MethodGet, s.nsPath("/artifacts/"+name+"/"+version), nil)
	require.Equal(t, http.StatusOK, rr.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	assert.Equal(t, artmod.StatusFailed, got["status"])
}

// TestDataset_CompleteWithoutManifest covers the client claiming success while
// no manifest was uploaded: the S3 GET 404s → 412 Precondition Failed.
func TestDataset_CompleteWithoutManifest(t *testing.T) {
	s := setup(t)
	s.resetState(t)

	const name, version = "no-manifest", "v1"
	datasetInitiate(t, s, name, version)

	rr := s.drive(t, http.MethodPost,
		s.nsPath("/artifacts/"+name+"/"+version+"/complete"),
		map[string]any{"digest": fakeDigest("anything")})
	assert.Equal(t, http.StatusPreconditionFailed, rr.Code, rr.Body.String())
}
