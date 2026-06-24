//go:build e2e

package e2e

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/axisml/axisml/test/e2e/internal/clients/artifacthub"
)

// artifact-hub. The only e2e-worthy path is the two-phase blob upload driven
// against the REAL zot registry (the integration suite stubs OCI with httptest,
// so it cannot verify the actual registry round-trip). Metadata-only lifecycle —
// initiate/patch/list/soft-delete — is deterministic and fully covered by the
// hermetic integration suite (TestArtifact_HappyPath, _PATCH_AllowedFields,
// _LabelSelectorFilter, _DeleteMarksDeleting), so it is NOT duplicated here.
func TestArtifactHub(t *testing.T) {
	ns, _ := provisionTenant(t)

	// ModelTwoPhaseUploadResolve exercises initiate -> push to real zot ->
	// complete -> resolve, asserting the resolved digest matches what was pushed.
	t.Run("ModelTwoPhaseUploadResolve", func(t *testing.T) {
		ctx := context.Background()
		name := uniqueName("e2e-2phase")
		res := initiateModel(t, ctx, ns, name, "1.0.0")
		t.Cleanup(func() {
			_, _ = h.artifactHub.DeleteModelWithResponse(context.Background(), ns, name, "1.0.0")
		})

		// Push a minimal manifest to zot through a port-forward.
		pf, err := startPortForward(h.cfg.InfraNamespace, "zot", 5000)
		require.NoError(t, err)
		defer pf.Stop()
		oc := &ociClient{base: pf.localURL(), creds: ociCredsFrom(res.Upload.Credentials), http: &http.Client{}}
		repo, ref := parseRepoRef(res.Upload.Uri)
		digest, err := oc.pushConfigOnlyManifest(ctx, repo, ref)
		require.NoError(t, err, "push manifest to zot")

		// Complete with the digest.
		c, err := h.artifactHub.CompleteModelWithResponse(ctx, ns, name, "1.0.0", artifacthub.ArtifactCompleteRequest{Digest: digest})
		require.NoError(t, err)
		require.True(t, is2xx(c.StatusCode()), "complete: %d: %s", c.StatusCode(), string(c.Body))

		// Status becomes Ready.
		eventually(t, h.cfg.CRProvisionTimeout, func() error {
			g, err := h.artifactHub.GetModelWithResponse(ctx, ns, name, "1.0.0")
			if err != nil {
				return err
			}
			if g.JSON200 == nil {
				return assertErr("GET model: %d", g.StatusCode())
			}
			if !strings.EqualFold(g.JSON200.Status, "Ready") {
				return assertErr("status=%q want Ready", g.JSON200.Status)
			}
			return nil
		})

		// Resolve returns pull info with the digest.
		r, err := h.artifactHub.ResolveModelWithResponse(ctx, ns, name, "1.0.0", nil)
		require.NoError(t, err)
		require.True(t, is2xx(r.StatusCode()), "resolve: %d: %s", r.StatusCode(), string(r.Body))
		require.NotNil(t, r.JSON200)
		require.NotNil(t, r.JSON200.Digest)
		assert.Equal(t, digest, *r.JSON200.Digest, "resolve should echo the completed digest")
	})
}

// ---- artifact-hub helpers ----

func initiateModel(t *testing.T, ctx context.Context, ns, name, version string) *artifacthub.ArtifactInitiateResponse {
	return initiateModelWithLabels(t, ctx, ns, name, version, nil)
}

func initiateModelWithLabels(t *testing.T, ctx context.Context, ns, name, version string, labels map[string]string) *artifacthub.ArtifactInitiateResponse {
	t.Helper()
	body := artifacthub.ArtifactInitiateRequest{
		Version: version,
		// Model spec requires a valid framework + a format (design §5.1).
		Spec: map[string]interface{}{"framework": "onnx", "format": "onnx"},
	}
	if labels != nil {
		body.Labels = &labels
	}
	r, err := h.artifactHub.InitiateModelWithResponse(ctx, ns, name, body)
	require.NoError(t, err)
	require.True(t, is2xx(r.StatusCode()), "initiate model: %d: %s", r.StatusCode(), string(r.Body))
	require.NotNil(t, r.JSON201, "initiate should return the artifact + upload envelope")
	return r.JSON201
}
