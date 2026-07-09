package handler

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/axisml/axisml/axisml-system/artifact-hub/internal/storage/oci"
)

// The OCI credential issuers are admin-passthrough in the MVP (no network), so
// the shared ociBacked upload/pull paths — and repoPath they route through —
// are exercisable without a live registry.
func TestOCIBacked_Credentials(t *testing.T) {
	h := NewModelHandler(oci.New(oci.Config{
		Endpoint: "zot.local:5000",
		Username: "admin",
		Password: "s3cret",
	}))
	a := Artifact{Namespace: "ns", Name: "n", Version: "v"}

	up, err := h.InitiateUpload(context.Background(), a, time.Hour)
	require.NoError(t, err)
	assert.Equal(t, "admin", up.Username)
	assert.Equal(t, "s3cret", up.Password)
	assert.WithinDuration(t, time.Now().Add(time.Hour), up.ExpiresAt, 5*time.Second)

	pull, err := h.IssuePullCredentials(context.Background(), a, time.Minute)
	require.NoError(t, err)
	assert.Equal(t, "admin", pull.Username)
	assert.Equal(t, "s3cret", pull.Password)
	assert.WithinDuration(t, time.Now().Add(time.Minute), pull.ExpiresAt, 5*time.Second)
}

func TestOCIBacked_GCBackend_EmptyDigestIsNoop(t *testing.T) {
	h := NewModelHandler(oci.New(oci.Config{Endpoint: "zot.local:5000"}))
	// Empty digest means the artifact never reached Ready — GC returns nil
	// without touching the registry.
	assert.NoError(t, h.GCBackend(context.Background(), Artifact{Namespace: "ns", Name: "n"}))
}

func TestKinds_ReturnsRegistered(t *testing.T) {
	t.Cleanup(reset)
	reset()

	Register(&stubHandler{kind: "model"})
	Register(&stubHandler{kind: "dataset"})
	// Kinds returns all registered kinds (order is not guaranteed).
	assert.ElementsMatch(t, []string{"model", "dataset"}, Kinds())
}
