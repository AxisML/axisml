//go:build e2e

package e2e

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// artifact-hub. Real PostgreSQL + real zot (OCI) + RustFS (S3). The
// metadata lifecycle is deterministic; the two-phase blob upload drives the
// real registry through the OCI helper in oci_test.go.

func modelPath(ns, name string) string {
	return "/api/v1/namespaces/" + ns + "/models/" + name
}

// artifact-hub does not enforce the identity header — a missing X-Axisml-User
// falls back to "anonymous" (only cluster-manager rejects it). Verify the
// request still succeeds rather than 401.
func TestArtifactHub_AnonymousAllowed(t *testing.T) {
	ctx := context.Background()
	r, err := h.artifactHub.doNoAuth(ctx, http.MethodGet, "/api/v1/namespaces/"+sharedNS()+"/models", nil)
	require.NoError(t, err)
	assert.True(t, r.is2xx(), "anonymous list should succeed, got %d", r.status)
}

func TestArtifactHub_ModelInitiateReturnsUpload(t *testing.T) {
	ctx := context.Background()
	ns := sharedNS()
	name := uniqueName("e2e-model")
	res := initiateModel(t, ctx, ns, name, "1.0.0")
	assert.NotEmpty(t, res.Upload.StorageKind, "initiate should return a storage kind")
	assert.NotEmpty(t, res.Upload.URI, "initiate should return an upload URI")
	t.Cleanup(func() {
		_, _ = h.artifactHub.do(context.Background(), http.MethodDelete, modelPath(ns, name)+"/1.0.0", nil)
	})
}

// TestArtifactHub_ModelTwoPhaseUploadResolve exercises initiate -> push to zot ->
// complete -> resolve against the real registry.
func TestArtifactHub_ModelTwoPhaseUploadResolve(t *testing.T) {
	ctx := context.Background()
	ns := sharedNS()
	name := uniqueName("e2e-2phase")
	res := initiateModel(t, ctx, ns, name, "1.0.0")
	t.Cleanup(func() {
		_, _ = h.artifactHub.do(context.Background(), http.MethodDelete, modelPath(ns, name)+"/1.0.0", nil)
	})

	// Push a minimal manifest to zot through a port-forward.
	pf, err := startPortForward(h.cfg.InfraNamespace, "zot", 5000)
	require.NoError(t, err)
	defer pf.Stop()
	oc := &ociClient{base: pf.localURL(), creds: parseOCICreds(res.Upload.Credentials), http: &http.Client{}}
	repo, ref := parseRepoRef(res.Upload.URI)
	digest, err := oc.pushConfigOnlyManifest(ctx, repo, ref)
	require.NoError(t, err, "push manifest to zot")

	// Complete with the digest.
	c := h.artifactHub.mustDo(t, ctx, http.MethodPost, modelPath(ns, name)+"/1.0.0/complete", ahCompleteReq{Digest: digest})
	require.True(t, c.is2xx(), "complete: %d: %s", c.status, string(c.body))

	// Status becomes Ready.
	eventually(t, h.cfg.CRProvisionTimeout, func() error {
		g := h.artifactHub.mustDo(t, ctx, http.MethodGet, modelPath(ns, name)+"/1.0.0", nil)
		var v ahView
		if err := g.decode(&v); err != nil {
			return err
		}
		if !strings.EqualFold(v.Status, "Ready") {
			return assertErr("status=%q want Ready", v.Status)
		}
		return nil
	})

	// Resolve returns pull info with the digest.
	r := h.artifactHub.mustDo(t, ctx, http.MethodGet, modelPath(ns, name)+"/1.0.0/resolve", nil)
	require.True(t, r.is2xx(), "resolve: %d: %s", r.status, string(r.body))
	var rr ahResolveResult
	require.NoError(t, r.decode(&rr))
	assert.Equal(t, digest, rr.Digest, "resolve should echo the completed digest")
}

func TestArtifactHub_PatchMetadata(t *testing.T) {
	ctx := context.Background()
	ns := sharedNS()
	name := uniqueName("e2e-patch")
	initiateModel(t, ctx, ns, name, "1.0.0")
	t.Cleanup(func() {
		_, _ = h.artifactHub.do(context.Background(), http.MethodDelete, modelPath(ns, name)+"/1.0.0", nil)
	})

	disp := "Friendly name"
	p := h.artifactHub.mustDo(t, ctx, http.MethodPatch, modelPath(ns, name)+"/1.0.0",
		ahPatchReq{DisplayName: &disp, Labels: map[string]string{"team": "e2e"}})
	require.True(t, p.is2xx(), "patch: %d: %s", p.status, string(p.body))

	g := h.artifactHub.mustDo(t, ctx, http.MethodGet, modelPath(ns, name)+"/1.0.0", nil)
	var v ahView
	require.NoError(t, g.decode(&v))
	assert.Equal(t, "e2e", v.Labels["team"])
}

func TestArtifactHub_ListAndLabelSelector(t *testing.T) {
	ctx := context.Background()
	ns := sharedNS()
	name := uniqueName("e2e-list")
	res := initiateModelWithLabels(t, ctx, ns, name, "1.0.0", map[string]string{"suite": "e2elist"})
	_ = res
	t.Cleanup(func() {
		_, _ = h.artifactHub.do(context.Background(), http.MethodDelete, modelPath(ns, name)+"/1.0.0", nil)
	})

	// List filtered by our unique label returns our artifact.
	l := h.artifactHub.mustDo(t, ctx, http.MethodGet,
		"/api/v1/namespaces/"+ns+"/models?labelSelector=suite%3De2elist", nil)
	require.True(t, l.is2xx(), "list: %d", l.status)
	assert.Contains(t, string(l.body), name, "filtered list should contain our model")
}

// DELETE soft-deletes: it flips the artifact to status "Deleting"; the GC
// worker reclaims the blob + row later on its interval (so the row lingers in
// listings until then). We assert the immediate, deterministic effect — the
// status transition — rather than waiting on GC.
func TestArtifactHub_SoftDelete(t *testing.T) {
	ctx := context.Background()
	ns := sharedNS()
	name := uniqueName("e2e-del")
	initiateModel(t, ctx, ns, name, "1.0.0")

	d := h.artifactHub.mustDo(t, ctx, http.MethodDelete, modelPath(ns, name)+"/1.0.0", nil)
	require.True(t, d.is2xx(), "delete: %d", d.status)

	eventually(t, h.cfg.CRProvisionTimeout, func() error {
		g := h.artifactHub.mustDo(t, ctx, http.MethodGet, modelPath(ns, name)+"/1.0.0", nil)
		if g.status == http.StatusNotFound {
			return nil // GC already reclaimed it
		}
		var v ahView
		if err := g.decode(&v); err != nil {
			return err
		}
		if v.Status != "Deleting" {
			return assertErr("status=%q want Deleting", v.Status)
		}
		return nil
	})
}

// ---- artifact-hub helpers ----

func initiateModel(t *testing.T, ctx context.Context, ns, name, version string) ahInitiateResult {
	return initiateModelWithLabels(t, ctx, ns, name, version, nil)
}

func initiateModelWithLabels(t *testing.T, ctx context.Context, ns, name, version string, labels map[string]string) ahInitiateResult {
	t.Helper()
	req := ahInitiateReq{
		Version: version,
		// Model spec requires a valid framework + a format (design §5.1).
		Spec:   map[string]any{"framework": "onnx", "format": "onnx"},
		Labels: labels,
	}
	r := h.artifactHub.mustDo(t, ctx, http.MethodPost, modelPath(ns, name), req)
	require.True(t, r.is2xx(), "initiate model: %d: %s", r.status, string(r.body))
	var res ahInitiateResult
	require.NoError(t, r.decode(&res))
	return res
}
