package handler

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "github.com/axisml/axisml/axisml-system/artifact-hub/pkg/errors"
)

func TestDatasetHandler_Basics(t *testing.T) {
	h := NewDatasetHandler("", nil) // empty bucket falls back to the default
	assert.Equal(t, "dataset", h.Kind())
	assert.Equal(t, StorageS3, h.StorageKind())

	uri := h.BuildStorageURI("ns", "n", "v")
	assert.Equal(t, "s3://axisml-artifact-hub/namespaces/ns/datasets/n/v/", uri)
	// S3 has no content-addressable path: pull URI equals storage URI regardless
	// of the supplied digest.
	assert.Equal(t, uri, h.BuildPullURI("ns", "n", "v", "sha256:whatever"))
}

func TestDatasetHandler_CustomBucket(t *testing.T) {
	h := NewDatasetHandler("mybucket", nil)
	assert.Equal(t, "s3://mybucket/namespaces/ns/datasets/n/v/", h.BuildStorageURI("ns", "n", "v"))
}

func TestDatasetHandler_ValidateSpec(t *testing.T) {
	h := NewDatasetHandler("b", nil)
	cases := []struct {
		name    string
		spec    Spec
		wantErr bool
	}{
		{name: "parquet ok", spec: Spec{"format": "parquet"}},
		{name: "custom ok", spec: Spec{"format": "custom"}},
		{name: "missing format", spec: Spec{}, wantErr: true},
		{name: "empty format", spec: Spec{"format": ""}, wantErr: true},
		{name: "non-string format", spec: Spec{"format": 42}, wantErr: true},
		{name: "unsupported format", spec: Spec{"format": "orc"}, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := h.ValidateSpec(context.Background(), tc.spec)
			if !tc.wantErr {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			e, ok := apperrors.As(err)
			require.True(t, ok)
			assert.Equal(t, apperrors.CodeValidation, e.Code)
		})
	}
}

func TestDatasetHandler_Credentials(t *testing.T) {
	h := NewDatasetHandler("b", nil)
	a := Artifact{Namespace: "ns", Name: "n", Version: "v"}

	up, err := h.InitiateUpload(context.Background(), a, time.Hour)
	require.NoError(t, err)
	assert.Equal(t, "axisml-dataset-upload", up.Username)
	assert.Equal(t, "prefix:namespaces/ns/datasets/n/v/", up.Password)
	assert.WithinDuration(t, time.Now().Add(time.Hour), up.ExpiresAt, 5*time.Second)

	pull, err := h.IssuePullCredentials(context.Background(), a, time.Hour)
	require.NoError(t, err)
	assert.Equal(t, "axisml-dataset-pull", pull.Username)
	assert.Equal(t, "prefix:namespaces/ns/datasets/n/v/", pull.Password)
	assert.WithinDuration(t, time.Now().Add(time.Hour), pull.ExpiresAt, 5*time.Second)
}

func TestDatasetHandler_VerifyComplete_NoBackend(t *testing.T) {
	h := NewDatasetHandler("b", nil) // nil s3 client → digests trusted, GC no-op
	a := Artifact{Namespace: "ns", Name: "n", Version: "v"}

	got, err := h.VerifyComplete(context.Background(), a, CompleteClaim{Digest: "sha256:abc"})
	require.NoError(t, err)
	assert.Equal(t, "sha256:abc", got, "without a backend the claimed digest is trusted")

	_, err = h.VerifyComplete(context.Background(), a, CompleteClaim{})
	require.Error(t, err)
	e, ok := apperrors.As(err)
	require.True(t, ok)
	assert.Equal(t, apperrors.CodeValidation, e.Code)
}

func TestDatasetHandler_GCBackend_NoBackend(t *testing.T) {
	h := NewDatasetHandler("b", nil)
	assert.NoError(t, h.GCBackend(context.Background(), Artifact{Namespace: "ns", Name: "n", Version: "v"}))
}
