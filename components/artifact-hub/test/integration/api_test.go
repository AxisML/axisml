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

	artmod "github.com/axisml/axisml/components/artifact-hub/internal/artifact"
)

// modelInitiateBody returns a well-formed Initiate body for the model Kind.
func modelInitiateBody(version string, overrides map[string]any) map[string]any {
	body := map[string]any{
		"version": version,
		"spec": map[string]any{
			"framework": "pytorch",
			"format":    "safetensors",
		},
		"display_name": "demo model",
	}
	for k, v := range overrides {
		body[k] = v
	}
	return body
}

// initiateOK posts /artifacts/model/{name} and asserts 201; returns parsed body.
func initiateOK(t *testing.T, s *suite, name, version string) map[string]any {
	t.Helper()
	rr := s.drive(t, http.MethodPost,
		s.nsPath("/artifacts/model/"+name), modelInitiateBody(version, nil))
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
		s.nsPath("/artifacts/model/"+name+"/"+version+"/complete"),
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
	assert.Equal(t, "oci", init["storage_kind"])
	assert.Contains(t, init["uri"], "namespaces/"+s.namespace+"/models/"+name+":"+version)
	assert.NotEmpty(t, init["artifact_id"])

	digest := fakeDigest(name + version)
	completed := completeOK(t, s, name, version, digest)
	assert.Equal(t, artmod.StatusReady, completed["status"])
	assert.Equal(t, digest, completed["digest"])

	// Idempotent re-Complete with same digest.
	rr := s.drive(t, http.MethodPost,
		s.nsPath("/artifacts/model/"+name+"/"+version+"/complete"),
		map[string]any{"digest": digest})
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	// Get.
	rr = s.drive(t, http.MethodGet, s.nsPath("/artifacts/model/"+name+"/"+version), nil)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	var got map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	assert.Equal(t, name, got["name"])
	assert.Equal(t, version, got["version"])
	assert.Equal(t, artmod.StatusReady, got["status"])

	// List under (kind, name).
	rr = s.drive(t, http.MethodGet, s.nsPath("/artifacts/model/"+name), nil)
	require.Equal(t, http.StatusOK, rr.Code)
	var listOut struct {
		Items []map[string]any `json:"items"`
		Total int64            `json:"total"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &listOut))
	assert.Equal(t, int64(1), listOut.Total)

	// Resolve(inspect) — no pull credentials.
	rr = s.drive(t, http.MethodGet,
		s.nsPath("/artifacts/model/"+name+"/"+version+"/resolve?usage=inspect"), nil)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	var inspect map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &inspect))
	assert.Equal(t, digest, inspect["digest"])
	assert.Nil(t, inspect["pull_credentials"])

	// Resolve(download) — pull credentials present.
	rr = s.drive(t, http.MethodGet,
		s.nsPath("/artifacts/model/"+name+"/"+version+"/resolve?usage=download"), nil)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	var download map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &download))
	assert.NotNil(t, download["pull_credentials"], "download must include pull credentials")
}

// TestArtifact_ListByKindAcrossNames verifies GET /artifacts/{kind} fans out
// across every (name, version) under the kind.
func TestArtifact_ListByKindAcrossNames(t *testing.T) {
	s := setup(t)
	s.resetState(t)

	for _, n := range []string{"alpha", "beta", "gamma"} {
		initiateOK(t, s, n, "v1")
	}

	rr := s.drive(t, http.MethodGet, s.nsPath("/artifacts/model"), nil)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	var out struct {
		Items []map[string]any `json:"items"`
		Total int64            `json:"total"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &out))
	assert.Equal(t, int64(3), out.Total)
}

// TestArtifact_DuplicateInitiateConflict ensures the (namespace, kind, name,
// version) idempotency guard returns 409 not 5xx.
func TestArtifact_DuplicateInitiateConflict(t *testing.T) {
	s := setup(t)
	s.resetState(t)

	const name, version = "dup", "v1"
	initiateOK(t, s, name, version)

	rr := s.drive(t, http.MethodPost,
		s.nsPath("/artifacts/model/"+name), modelInitiateBody(version, nil))
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
				"version": "v1",
				"spec":    map[string]any{"format": "safetensors"},
			},
		},
		{
			name: "unknown framework",
			body: map[string]any{
				"version": "v1",
				"spec":    map[string]any{"framework": "wat", "format": "safetensors"},
			},
		},
		{
			name: "missing format",
			body: map[string]any{
				"version": "v1",
				"spec":    map[string]any{"framework": "pytorch"},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := s.drive(t, http.MethodPost,
				s.nsPath("/artifacts/model/some-model"), tc.body)
			assert.Equal(t, http.StatusBadRequest, rr.Code,
				"validation must surface as 400; body=%s", rr.Body.String())
		})
	}
}

// TestArtifact_UnknownKind404Style covers the kind-not-registered path. It's
// surfaced as CodeValidation → 400 today (registry lookup happens before the
// route knows the URL refers to "no such handler"); we just assert it's 4xx.
func TestArtifact_UnknownKind(t *testing.T) {
	s := setup(t)
	s.resetState(t)

	rr := s.drive(t, http.MethodPost,
		s.nsPath("/artifacts/dataset/foo"), modelInitiateBody("v1", nil))
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
		s.nsPath("/artifacts/model/"+name+"/"+version+"/complete"),
		map[string]any{"digest": clientDigest})
	require.Equal(t, http.StatusConflict, rr.Code, rr.Body.String())

	// The service must mark the row Failed (precondition for Phase-2 GC of
	// orphaned blobs).
	rr = s.drive(t, http.MethodGet,
		s.nsPath("/artifacts/model/"+name+"/"+version), nil)
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
		s.nsPath("/artifacts/model/"+name+"/"+version+"/complete"),
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
		s.nsPath("/artifacts/model/"+name+"/"+version+"/resolve?usage=inspect"), nil)
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
		s.nsPath("/artifacts/model/"+name+"/"+version), nil)
	require.Equal(t, http.StatusAccepted, rr.Code, rr.Body.String())

	rr = s.drive(t, http.MethodGet,
		s.nsPath("/artifacts/model/"+name+"/"+version), nil)
	require.Equal(t, http.StatusOK, rr.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	assert.Equal(t, artmod.StatusDeleting, got["status"])

	// Idempotent: a second DELETE on a Deleting row is still 202.
	rr = s.drive(t, http.MethodDelete,
		s.nsPath("/artifacts/model/"+name+"/"+version), nil)
	assert.Equal(t, http.StatusAccepted, rr.Code, rr.Body.String())
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
		s.nsPath("/artifacts/model/"+name+"/"+version), nil)
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
		s.nsPath("/artifacts/model/"+name+"/"+version), nil)
	require.Equal(t, http.StatusAccepted, rr.Code)

	s.gcW.Tick(context.Background())

	rr = s.drive(t, http.MethodGet,
		s.nsPath("/artifacts/model/"+name+"/"+version), nil)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	var got map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	assert.Equal(t, artmod.StatusDeleted, got["status"])

	// Re-Initiate over the tombstone is allowed (Initiate keeps using the
	// deleted_at-filtered GetByCoord).
	rr = s.drive(t, http.MethodPost,
		s.nsPath("/artifacts/model/"+name), modelInitiateBody(version, nil))
	assert.Equal(t, http.StatusCreated, rr.Code,
		"new version with same coord must be allowed once the prior is fully Deleted; body=%s",
		rr.Body.String())
}

// --- helpers ----------------------------------------------------------------

type fakeClock struct{ now time.Time }

func (c fakeClock) Now() time.Time { return c.now }

type realClockUTC struct{}

func (realClockUTC) Now() time.Time { return time.Now().UTC() }
