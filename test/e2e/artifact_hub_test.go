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

func TestArtifactHub_MissingIdentity401(t *testing.T) {
	ctx := context.Background()
	r, err := h.artifactHub.doNoAuth(ctx, http.MethodGet, "/api/v1/namespaces/"+sharedNS()+"/models", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, r.status)
}

func TestArtifactHub_ModelInitiateReturnsUpload(t *testing.T) {
	ctx := context.Background()
	ns := sharedNS()
	name := uniqueName("l5-model")
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
	name := uniqueName("l5-2phase")
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
	name := uniqueName("l5-patch")
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
	name := uniqueName("l5-list")
	res := initiateModelWithLabels(t, ctx, ns, name, "1.0.0", map[string]string{"suite": "l5list"})
	_ = res
	t.Cleanup(func() {
		_, _ = h.artifactHub.do(context.Background(), http.MethodDelete, modelPath(ns, name)+"/1.0.0", nil)
	})

	// List filtered by our unique label returns our artifact.
	l := h.artifactHub.mustDo(t, ctx, http.MethodGet,
		"/api/v1/namespaces/"+ns+"/models?labelSelector=suite%3Dl5list", nil)
	require.True(t, l.is2xx(), "list: %d", l.status)
	assert.Contains(t, string(l.body), name, "filtered list should contain our model")
}

func TestArtifactHub_SoftDeleteHidesFromList(t *testing.T) {
	ctx := context.Background()
	ns := sharedNS()
	name := uniqueName("l5-del")
	initiateModel(t, ctx, ns, name, "1.0.0")

	d := h.artifactHub.mustDo(t, ctx, http.MethodDelete, modelPath(ns, name)+"/1.0.0", nil)
	require.True(t, d.is2xx(), "delete: %d", d.status)

	// The version no longer appears in the default (non-deleted) listing.
	eventually(t, h.cfg.CRProvisionTimeout, func() error {
		l := h.artifactHub.mustDo(t, ctx, http.MethodGet, modelPath(ns, name), nil)
		if l.status == http.StatusNotFound {
			return nil
		}
		if strings.Contains(string(l.body), `"version":"1.0.0"`) {
			return assertErr("soft-deleted version still listed")
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
		Spec:    map[string]any{"format": "onnx"},
		Labels:  labels,
	}
	r := h.artifactHub.mustDo(t, ctx, http.MethodPost, modelPath(ns, name), req)
	require.True(t, r.is2xx(), "initiate model: %d: %s", r.status, string(r.body))
	var res ahInitiateResult
	require.NoError(t, r.decode(&res))
	return res
}
