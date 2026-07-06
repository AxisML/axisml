//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	artmod "github.com/axisml/axisml/axisml-system/artifact-hub/internal/artifact"
)

// modelInitiateBody returns a well-formed Initiate body for the model Kind.
func modelInitiateBody(version string, overrides map[string]any) map[string]any {
	body := map[string]any{
		"kind":    "model",
		"version": version,
		"spec": map[string]any{
			"framework": "pytorch",
			"format":    "safetensors",
		},
		"displayName": "demo model",
	}
	for k, v := range overrides {
		body[k] = v
	}
	return body
}

// initiateOK posts /artifacts/{name} and asserts 201; returns parsed body.
func initiateOK(t *testing.T, s *suite, name, version string) map[string]any {
	t.Helper()
	rr := s.drive(t, http.MethodPost,
		s.nsPath("/artifacts/"+name), modelInitiateBody(version, nil))
	require.Equalf(t, http.StatusCreated, rr.Code,
		"initiate should succeed; body=%s", rr.Body.String())
	var out map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &out))
	return out
}

// completeOK pushes a manifest into fakeZot and posts /complete.
func completeOK(t *testing.T, s *suite, name, version, digest string) map[string]any {
	t.Helper()
	repoPath := "namespaces/" + s.namespace + "/models/" + name
	s.zot.pushManifest(repoPath, version, digest)
	rr := s.drive(t, http.MethodPost,
		s.nsPath("/artifacts/"+name+"/"+version+"/complete"),
		map[string]any{"digest": digest})
	require.Equalf(t, http.StatusOK, rr.Code,
		"complete should succeed; body=%s", rr.Body.String())
	var out map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &out))
	return out
}

// TestArtifact_HappyPath drives the full two-phase write + read flow for a
// model: Initiate → Complete → Get → List → Resolve(inspect) → Resolve(download).
func TestArtifact_HappyPath(t *testing.T) {
	s := setup(t)
	s.resetState(t)

	const name, version = "llama-7b", "v1"

	init := initiateOK(t, s, name, version)
	upload := init["upload"].(map[string]any)
	assert.Equal(t, "oci", upload["storageKind"])
	assert.Contains(t, upload["uri"], "namespaces/"+s.namespace+"/models/"+name+":"+version)
	artifactView := init["artifact"].(map[string]any)
	assert.NotEmpty(t, artifactView["id"])

	digest := fakeDigest(name + version)
	completed := completeOK(t, s, name, version, digest)
	assert.Equal(t, artmod.StatusReady, completed["status"])
	assert.Equal(t, digest, completed["digest"])

	// Idempotent re-Complete with same digest.
	rr := s.drive(t, http.MethodPost,
		s.nsPath("/artifacts/"+name+"/"+version+"/complete"),
		map[string]any{"digest": digest})
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	// Get.
	rr = s.drive(t, http.MethodGet, s.nsPath("/artifacts/"+name+"/"+version), nil)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	var got map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	assert.Equal(t, name, got["name"])
	assert.Equal(t, version, got["version"])
	assert.Equal(t, artmod.StatusReady, got["status"])

	// List under (kind, name).
	rr = s.drive(t, http.MethodGet, s.nsPath("/artifacts/"+name), nil)
	require.Equal(t, http.StatusOK, rr.Code)
	var listOut struct {
		Items []map[string]any `json:"items"`
		Total int64            `json:"total"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &listOut))
	assert.Equal(t, int64(1), listOut.Total)

	// Resolve(inspect) — no pull credentials.
	rr = s.drive(t, http.MethodGet,
		s.nsPath("/artifacts/"+name+"/"+version+"/resolve?usage=inspect"), nil)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	var inspect map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &inspect))
	assert.Equal(t, digest, inspect["digest"])
	assert.Nil(t, inspect["pullCredentials"])

	// Resolve(download) — pull credentials present.
	rr = s.drive(t, http.MethodGet,
		s.nsPath("/artifacts/"+name+"/"+version+"/resolve?usage=download"), nil)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	var download map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &download))
	assert.NotNil(t, download["pullCredentials"], "download must include pull credentials")
}

// TestArtifact_DuplicateInitiateConflict ensures the (namespace, kind, name,
// version) idempotency guard returns 409 not 5xx.
func TestArtifact_DuplicateInitiateConflict(t *testing.T) {
	s := setup(t)
	s.resetState(t)

	const name, version = "dup", "v1"
	initiateOK(t, s, name, version)

	rr := s.drive(t, http.MethodPost,
		s.nsPath("/artifacts/"+name), modelInitiateBody(version, nil))
	assert.Equal(t, http.StatusConflict, rr.Code, rr.Body.String())
}

// TestArtifact_ValidationRejections covers both bind-time tag failures
// (caught by the gin validator and routed through isBindingError → 400)
// and the service-layer handler.ValidateSpec branches. All paths should
// surface as 400.
func TestArtifact_ValidationRejections(t *testing.T) {
	s := setup(t)
	s.resetState(t)

	cases := []struct {
		name string
		body map[string]any
	}{
		{
			name: "invalid version charset (bind validator)",
			body: modelInitiateBody("not a tag!", nil),
		},
		{
			name: "missing framework",
			body: map[string]any{
				"kind":    "model",
				"version": "v1",
				"spec":    map[string]any{"format": "safetensors"},
			},
		},
		{
			name: "unknown framework",
			body: map[string]any{
				"kind":    "model",
				"version": "v1",
				"spec":    map[string]any{"framework": "wat", "format": "safetensors"},
			},
		},
		{
			name: "missing format",
			body: map[string]any{
				"kind":    "model",
				"version": "v1",
				"spec":    map[string]any{"framework": "pytorch"},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := s.drive(t, http.MethodPost,
				s.nsPath("/artifacts/some-model"), tc.body)
			assert.Equal(t, http.StatusBadRequest, rr.Code,
				"validation must surface as 400; body=%s", rr.Body.String())
		})
	}
}

// TestArtifact_UnknownKind covers the kind-not-registered path: a body kind
// with no handler is surfaced as CodeValidation → 400. We just assert it's 4xx.
func TestArtifact_UnknownKind(t *testing.T) {
	s := setup(t)
	s.resetState(t)

	rr := s.drive(t, http.MethodPost,
		s.nsPath("/artifacts/foo"),
		modelInitiateBody("v1", map[string]any{"kind": "totally-unknown"}))
	assert.GreaterOrEqual(t, rr.Code, 400)
	assert.Less(t, rr.Code, 500, "must be a client error, not a panic")
}

// TestArtifact_CompleteDigestMismatch makes the cli's claim disagree with
// what fakeZot reports; the service must mark the row Failed and return 409.
func TestArtifact_CompleteDigestMismatch(t *testing.T) {
	s := setup(t)
	s.resetState(t)

	const name, version = "skewed", "v1"
	initiateOK(t, s, name, version)

	registryDigest := fakeDigest(name + "-real")
	clientDigest := fakeDigest(name + "-claim")
	repoPath := "namespaces/" + s.namespace + "/models/" + name
	s.zot.pushManifest(repoPath, version, registryDigest)

	rr := s.drive(t, http.MethodPost,
		s.nsPath("/artifacts/"+name+"/"+version+"/complete"),
		map[string]any{"digest": clientDigest})
	require.Equal(t, http.StatusConflict, rr.Code, rr.Body.String())

	// The service must mark the row Failed (precondition for Phase-2 GC of
	// orphaned blobs).
	rr = s.drive(t, http.MethodGet,
		s.nsPath("/artifacts/"+name+"/"+version), nil)
	require.Equal(t, http.StatusOK, rr.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	assert.Equal(t, artmod.StatusFailed, got["status"])
}

// TestArtifact_CompleteWithoutManifest412 covers the case where the cli
// claims success but the registry HEAD returns 404 (cli skipped or aborted
// the push).
func TestArtifact_CompleteWithoutManifest(t *testing.T) {
	s := setup(t)
	s.resetState(t)

	const name, version = "no-push", "v1"
	initiateOK(t, s, name, version)

	rr := s.drive(t, http.MethodPost,
		s.nsPath("/artifacts/"+name+"/"+version+"/complete"),
		map[string]any{"digest": fakeDigest("anything")})
	assert.Equal(t, http.StatusPreconditionFailed, rr.Code, rr.Body.String())
}

// TestArtifact_ResolveBeforeReady412 ensures Resolve refuses Uploading rows.
func TestArtifact_ResolveBeforeReady(t *testing.T) {
	s := setup(t)
	s.resetState(t)

	const name, version = "pending", "v1"
	initiateOK(t, s, name, version)

	rr := s.drive(t, http.MethodGet,
		s.nsPath("/artifacts/"+name+"/"+version+"/resolve?usage=inspect"), nil)
	assert.Equal(t, http.StatusPreconditionFailed, rr.Code, rr.Body.String())
}

// TestArtifact_DeleteFlowMarksDeleting covers DELETE → 202; the row stays
// in Deleting until the GC worker finalises it (covered by TestGC_Deleting).
func TestArtifact_DeleteMarksDeleting(t *testing.T) {
	s := setup(t)
	s.resetState(t)

	const name, version = "rm-me", "v1"
	initiateOK(t, s, name, version)
	completeOK(t, s, name, version, fakeDigest(name+version))

	rr := s.drive(t, http.MethodDelete,
		s.nsPath("/artifacts/"+name+"/"+version), nil)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	rr = s.drive(t, http.MethodGet,
		s.nsPath("/artifacts/"+name+"/"+version), nil)
	require.Equal(t, http.StatusOK, rr.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	assert.Equal(t, artmod.StatusDeleting, got["status"])

	// Idempotent: a second DELETE on a Deleting row is still 202.
	rr = s.drive(t, http.MethodDelete,
		s.nsPath("/artifacts/"+name+"/"+version), nil)
	assert.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
}

// TestGC_StaleUploadingFlippedToFailed verifies the Uploading-TTL predicate.
// We fast-forward the worker's clock past UploadingTTL and assert the row
// transitions to Failed.
func TestGC_StaleUploadingFlippedToFailed(t *testing.T) {
	s := setup(t)
	s.resetState(t)

	const name, version = "stale", "v1"
	initiateOK(t, s, name, version)

	// Jump the worker's clock 48h into the future (UploadingTTL is 24h).
	s.gcW.SetClock(fakeClock{now: time.Now().UTC().Add(48 * time.Hour)})
	t.Cleanup(func() { s.gcW.SetClock(realClockUTC{}) })

	s.gcW.Tick(context.Background())

	rr := s.drive(t, http.MethodGet,
		s.nsPath("/artifacts/"+name+"/"+version), nil)
	require.Equal(t, http.StatusOK, rr.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	assert.Equal(t, artmod.StatusFailed, got["status"])
}

// TestGC_DeletingMarkedDeleted verifies the Deleting predicate. The row is
// flipped to Deleted with deleted_at populated; read paths use
// GetByCoordIncludingDeleted so GET still returns the row with the terminal
// status (so a client can confirm DELETE has propagated).
func TestGC_DeletingMarkedDeleted(t *testing.T) {
	s := setup(t)
	s.resetState(t)

	const name, version = "doomed", "v1"
	initiateOK(t, s, name, version)
	completeOK(t, s, name, version, fakeDigest(name+version))

	rr := s.drive(t, http.MethodDelete,
		s.nsPath("/artifacts/"+name+"/"+version), nil)
	require.Equal(t, http.StatusOK, rr.Code)

	s.gcW.Tick(context.Background())

	rr = s.drive(t, http.MethodGet,
		s.nsPath("/artifacts/"+name+"/"+version), nil)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	var got map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	assert.Equal(t, artmod.StatusDeleted, got["status"])

	// Per database.md §1.2, the artifact 4-tuple does NOT recycle — even
	// after a full Deleted tombstone, re-Initiate on the same coord must
	// fail. Callers bump the version instead.
	rr = s.drive(t, http.MethodPost,
		s.nsPath("/artifacts/"+name), modelInitiateBody(version, nil))
	assert.GreaterOrEqual(t, rr.Code, 400,
		"re-initiate over a Deleted tombstone must be rejected; body=%s",
		rr.Body.String())
}

// --- helpers ----------------------------------------------------------------

type fakeClock struct{ now time.Time }

func (c fakeClock) Now() time.Time { return c.now }

type realClockUTC struct{}

func (realClockUTC) Now() time.Time { return time.Now().UTC() }
