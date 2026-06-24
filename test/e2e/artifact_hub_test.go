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

// artifact-hub. Real PostgreSQL + real zot (OCI) + RustFS (S3). The metadata
// lifecycle is deterministic; the two-phase blob upload drives the real registry
// through the OCI helper in oci_test.go. Scenarios share one tenant's namespace
// (artifact-hub partitions by namespace) and run as subtests.
func TestArtifactHub(t *testing.T) {
	ns, _ := provisionTenant(t)

	// artifact-hub does not enforce the identity header — a missing X-Axisml-User
	// falls back to "anonymous" (only cluster-manager rejects it). Verify the
	// request still succeeds rather than 401.
	t.Run("AnonymousAllowed", func(t *testing.T) {
		ctx := context.Background()
		r, err := h.artifactHub.ListModelsWithResponse(ctx, ns, nil, anon)
		require.NoError(t, err)
		assert.True(t, is2xx(r.StatusCode()), "anonymous list should succeed, got %d", r.StatusCode())
	})

	t.Run("ModelInitiateReturnsUpload", func(t *testing.T) {
		ctx := context.Background()
		name := uniqueName("e2e-model")
		res := initiateModel(t, ctx, ns, name, "1.0.0")
		assert.NotEmpty(t, res.Upload.StorageKind, "initiate should return a storage kind")
		assert.NotEmpty(t, res.Upload.Uri, "initiate should return an upload URI")
		t.Cleanup(func() {
			_, _ = h.artifactHub.DeleteModelWithResponse(context.Background(), ns, name, "1.0.0")
		})
	})

	// ModelTwoPhaseUploadResolve exercises initiate -> push to zot -> complete ->
	// resolve against the real registry.
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

	t.Run("PatchMetadata", func(t *testing.T) {
		ctx := context.Background()
		name := uniqueName("e2e-patch")
		initiateModel(t, ctx, ns, name, "1.0.0")
		t.Cleanup(func() {
			_, _ = h.artifactHub.DeleteModelWithResponse(context.Background(), ns, name, "1.0.0")
		})

		p, err := h.artifactHub.UpdateModelWithResponse(ctx, ns, name, "1.0.0", artifacthub.ArtifactPatchRequest{
			DisplayName: ptr("Friendly name"),
			Labels:      &map[string]string{"team": "e2e"},
		})
		require.NoError(t, err)
		require.True(t, is2xx(p.StatusCode()), "patch: %d: %s", p.StatusCode(), string(p.Body))

		g, err := h.artifactHub.GetModelWithResponse(ctx, ns, name, "1.0.0")
		require.NoError(t, err)
		require.NotNil(t, g.JSON200)
		require.NotNil(t, g.JSON200.Labels)
		assert.Equal(t, "e2e", (*g.JSON200.Labels)["team"])
	})

	t.Run("ListAndLabelSelector", func(t *testing.T) {
		ctx := context.Background()
		name := uniqueName("e2e-list")
		initiateModelWithLabels(t, ctx, ns, name, "1.0.0", map[string]string{"suite": "e2elist"})
		t.Cleanup(func() {
			_, _ = h.artifactHub.DeleteModelWithResponse(context.Background(), ns, name, "1.0.0")
		})

		// List filtered by our unique label returns our artifact.
		l, err := h.artifactHub.ListModelsWithResponse(ctx, ns, &artifacthub.ListModelsParams{LabelSelector: ptr("suite=e2elist")})
		require.NoError(t, err)
		require.True(t, is2xx(l.StatusCode()), "list: %d", l.StatusCode())
		require.NotNil(t, l.JSON200)
		found := false
		for _, a := range l.JSON200.Items {
			if a.Name == name {
				found = true
				break
			}
		}
		assert.True(t, found, "filtered list should contain our model %s", name)
	})

	// DELETE soft-deletes: it flips the artifact to status "Deleting"; the GC
	// worker reclaims the blob + row later on its interval (so the row lingers in
	// listings until then). We assert the immediate, deterministic effect — the
	// status transition — rather than waiting on GC.
	t.Run("SoftDelete", func(t *testing.T) {
		ctx := context.Background()
		name := uniqueName("e2e-del")
		initiateModel(t, ctx, ns, name, "1.0.0")

		d, err := h.artifactHub.DeleteModelWithResponse(ctx, ns, name, "1.0.0")
		require.NoError(t, err)
		require.True(t, is2xx(d.StatusCode()), "delete: %d", d.StatusCode())

		eventually(t, h.cfg.CRProvisionTimeout, func() error {
			g, err := h.artifactHub.GetModelWithResponse(ctx, ns, name, "1.0.0")
			if err != nil {
				return err
			}
			if g.StatusCode() == http.StatusNotFound {
				return nil // GC already reclaimed it
			}
			if g.JSON200 == nil {
				return assertErr("GET model: %d", g.StatusCode())
			}
			if g.JSON200.Status != "Deleting" {
				return assertErr("status=%q want Deleting", g.JSON200.Status)
			}
			return nil
		})
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
