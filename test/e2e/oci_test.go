//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Minimal OCI-distribution push helper used by the artifact-hub two-phase
// upload tests. It talks to zot through a port-forward and pushes a tiny
// config-only image manifest, returning the manifest digest that the
// `complete` call records.
//
// NOTE: the exact shape of artifact-hub's upload credentials/URI is the one
// piece this suite cannot verify offline. The parsing below is defensive
// (username/password or bearer token, scheme-stripped repo path); if a live run
// shows a different contract, adjust ociCreds / parseRepoRef here — the rest of
// the flow is contract-stable.

type ociCreds struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Token    string `json:"token"`
}

func parseOCICreds(raw json.RawMessage) ociCreds {
	var c ociCreds
	_ = json.Unmarshal(raw, &c)
	return c
}

// parseRepoRef extracts (repository, reference) from an upload URI, ignoring the
// host (we reach the registry via the port-forward at 127.0.0.1).
func parseRepoRef(uri string) (repo, ref string) {
	s := uri
	for _, p := range []string{"oci://", "https://", "http://", "docker://"} {
		s = strings.TrimPrefix(s, p)
	}
	// drop host
	if i := strings.IndexByte(s, '/'); i >= 0 {
		s = s[i+1:]
	}
	ref = "latest"
	if i := strings.LastIndexByte(s, ':'); i >= 0 && !strings.Contains(s[i:], "/") {
		ref = s[i+1:]
		s = s[:i]
	}
	return s, ref
}

func sha256Digest(b []byte) string {
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

type ociClient struct {
	base  string // http://127.0.0.1:<port>
	creds ociCreds
	http  *http.Client
}

func (o *ociClient) req(ctx context.Context, method, path string, body []byte, contentType string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, o.base+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if o.creds.Token != "" {
		req.Header.Set("Authorization", "Bearer "+o.creds.Token)
	} else if o.creds.Username != "" {
		req.SetBasicAuth(o.creds.Username, o.creds.Password)
	}
	return o.http.Do(req)
}

// pushConfigOnlyManifest pushes an empty config blob and a manifest referencing
// it, returning the manifest digest.
func (o *ociClient) pushConfigOnlyManifest(ctx context.Context, repo, ref string) (string, error) {
	config := []byte(`{}`)
	cfgDigest := sha256Digest(config)
	if err := o.pushBlob(ctx, repo, cfgDigest, config); err != nil {
		return "", fmt.Errorf("push config blob: %w", err)
	}

	manifest := map[string]any{
		"schemaVersion": 2,
		"mediaType":     "application/vnd.oci.image.manifest.v1+json",
		"config": map[string]any{
			"mediaType": "application/vnd.oci.image.config.v1+json",
			"digest":    cfgDigest,
			"size":      len(config),
		},
		"layers": []any{},
	}
	mb, _ := json.Marshal(manifest)
	mDigest := sha256Digest(mb)

	resp, err := o.req(ctx, http.MethodPut, fmt.Sprintf("/v2/%s/manifests/%s", repo, ref),
		mb, "application/vnd.oci.image.manifest.v1+json")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("put manifest: %d: %s", resp.StatusCode, string(b))
	}
	return mDigest, nil
}

func (o *ociClient) pushBlob(ctx context.Context, repo, digest string, blob []byte) error {
	// Start an upload session.
	resp, err := o.req(ctx, http.MethodPost, fmt.Sprintf("/v2/%s/blobs/uploads/", repo), nil, "")
	if err != nil {
		return err
	}
	loc := resp.Header.Get("Location")
	resp.Body.Close()
	if resp.StatusCode/100 != 2 || loc == "" {
		return fmt.Errorf("start upload: status %d loc=%q", resp.StatusCode, loc)
	}
	// Monolithic PUT with the digest query param.
	sep := "?"
	if strings.Contains(loc, "?") {
		sep = "&"
	}
	// Location may be absolute or relative; normalize to a path on our base.
	put := loc
	for _, p := range []string{"http://", "https://"} {
		if strings.HasPrefix(put, p) {
			if i := strings.IndexByte(strings.TrimPrefix(put, p), '/'); i >= 0 {
				put = strings.TrimPrefix(put, p)[i:]
			}
		}
	}
	resp2, err := o.req(ctx, http.MethodPut, put+sep+"digest="+digest, blob, "application/octet-stream")
	if err != nil {
		return err
	}
	defer resp2.Body.Close()
	if resp2.StatusCode/100 != 2 {
		b, _ := io.ReadAll(resp2.Body)
		return fmt.Errorf("put blob: %d: %s", resp2.StatusCode, string(b))
	}
	return nil
}
