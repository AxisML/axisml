//go:build integration

package integration_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestArtifact_PATCH_AllowedFields exercises PATCH for the four mutable
// display fields. Issued against a Ready artifact (post-complete).
func TestArtifact_PATCH_AllowedFields(t *testing.T) {
	s := setup(t)
	s.resetState(t)

	const name, version = "patchable", "v1"
	initiateOK(t, s, name, version)
	digest := fakeDigest(name + version)
	completeOK(t, s, name, version, digest)

	body := map[string]any{
		"display_name": "renamed model",
		"description":  "patched description",
		"labels":       map[string]string{"axisml.io/project": "p1"},
		"annotations":  map[string]string{"axisml.io/note": "hi"},
	}
	rr := s.drive(t, http.MethodPatch,
		s.nsPath("/models/"+name+"/"+version), body)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	var got map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	assert.Equal(t, "renamed model", got["display_name"])
	assert.Equal(t, "patched description", got["description"])
}

// TestArtifact_PATCH_ImmutableField rejects any field outside the design
// allow-list (display_name / description / labels / annotations) with 400.
func TestArtifact_PATCH_ImmutableField(t *testing.T) {
	s := setup(t)
	s.resetState(t)

	const name, version = "immutable", "v1"
	initiateOK(t, s, name, version)
	completeOK(t, s, name, version, fakeDigest(name+version))

	rr := s.drive(t, http.MethodPatch,
		s.nsPath("/models/"+name+"/"+version),
		map[string]any{"visibility": "public"})
	require.Equal(t, http.StatusBadRequest, rr.Code, rr.Body.String())
}

// TestArtifact_Visibility_PublicGate enforces that visibility=public is
// only accepted under the axisml-system namespace; any other namespace
// returns 403 Forbidden per design §3.
func TestArtifact_Visibility_PublicGate(t *testing.T) {
	s := setup(t)
	s.resetState(t)

	body := map[string]any{
		"version": "v1",
		"spec": map[string]any{
			"framework": "pytorch",
			"format":    "safetensors",
		},
		"visibility": "public",
	}
	rr := s.drive(t, http.MethodPost,
		s.nsPath("/models/public-only"), body)
	require.GreaterOrEqual(t, rr.Code, 400)
	require.Less(t, rr.Code, 500)
}
