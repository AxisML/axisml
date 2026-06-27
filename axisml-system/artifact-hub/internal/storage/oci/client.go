// Package oci wraps the bits of the OCI Distribution v2 protocol that
// axisml-artifact-hub needs against zot: HEAD manifest (for digest verification
// at complete time), DELETE manifest (for GC), and credential issuance for
// cli push / pull.
//
// MVP simplification (per axisml-system/docs/artifact-hub.md §8.3 #4):
// credential issuance is admin-htpasswd passthrough. The artifact-hub pod
// holds zot admin credentials and returns them verbatim on initiate /
// resolve?usage=download.
// This is NOT scope-limited and NOT TTL-bounded — any holder of the returned
// creds can push to any zot repo. Acceptable only because zot is ClusterIP-
// scoped inside axisml-infra. Phase 2 replaces this with a JWT issuer for
// zot's bearer-token realm.
package oci

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client is a thin wrapper around net/http for the OCI v2 endpoints
// artifact-hub needs against zot.
type Client struct {
	baseURL  string
	scheme   string
	username string
	password string
	http     *http.Client
}

// Config bundles the constructor inputs. Endpoint is the externally
// reachable host:port (e.g., axisml-infra-zot.axisml-infra:5000); Scheme is
// http or https.
type Config struct {
	Endpoint    string
	Scheme      string
	Username    string
	Password    string
	HTTPTimeout time.Duration
}

// New returns a Client. If Endpoint already carries a scheme it overrides
// cfg.Scheme; otherwise cfg.Scheme is prepended.
func New(cfg Config) *Client {
	scheme := cfg.Scheme
	endpoint := cfg.Endpoint
	if strings.HasPrefix(endpoint, "http://") {
		scheme = "http"
		endpoint = strings.TrimPrefix(endpoint, "http://")
	} else if strings.HasPrefix(endpoint, "https://") {
		scheme = "https"
		endpoint = strings.TrimPrefix(endpoint, "https://")
	}
	if scheme == "" {
		scheme = "http"
	}
	timeout := cfg.HTTPTimeout
	if timeout == 0 {
		timeout = 15 * time.Second
	}
	return &Client{
		baseURL:  endpoint,
		scheme:   scheme,
		username: cfg.Username,
		password: cfg.Password,
		http:     &http.Client{Timeout: timeout},
	}
}

// Endpoint returns the host:port form (no scheme). Useful for constructing
// OCI image references which never carry a scheme.
func (c *Client) Endpoint() string { return c.baseURL }

// Credentials is the credential bundle returned from IssueUploadCredentials
// and IssuePullCredentials. ExpiresAt is informational only in MVP — see the
// package doc.
type Credentials struct {
	Username  string    `json:"username" desc:"Username for authenticating to the OCI storage backend."`
	Password  string    `json:"password" desc:"Password (or token) for authenticating to the OCI storage backend."`
	ExpiresAt time.Time `json:"expires_at" desc:"Expiry of the credentials (RFC3339)."`
}

// IssueUploadCredentials returns push-capable creds for the given scope.
// MVP: admin-htpasswd passthrough; Phase 2 replaces with scope-limited JWT.
func (c *Client) IssueUploadCredentials(_ context.Context, _ string, ttl time.Duration) (Credentials, error) {
	return Credentials{
		Username:  c.username,
		Password:  c.password,
		ExpiresAt: time.Now().UTC().Add(ttl),
	}, nil
}

// IssuePullCredentials returns pull-capable creds for the given scope.
// MVP: same admin-htpasswd passthrough as IssueUploadCredentials.
func (c *Client) IssuePullCredentials(_ context.Context, _ string, ttl time.Duration) (Credentials, error) {
	return Credentials{
		Username:  c.username,
		Password:  c.password,
		ExpiresAt: time.Now().UTC().Add(ttl),
	}, nil
}

// HeadManifest issues a HEAD against /v2/<repoPath>/manifests/<reference>
// and returns the digest from the Docker-Content-Digest response header.
// repoPath is the full repository path e.g. "tenants/default/models/llama-7b";
// reference is the tag or sha256:... digest.
func (c *Client) HeadManifest(ctx context.Context, repoPath, reference string) (string, error) {
	url := fmt.Sprintf("%s://%s/v2/%s/manifests/%s", c.scheme, c.baseURL, repoPath, reference)
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(c.username, c.password)
	// OCI manifest content negotiation — accept the common manifest types
	// so zot returns the same digest regardless of which manifest variant
	// the client pushed.
	req.Header.Set("Accept", strings.Join([]string{
		"application/vnd.oci.image.manifest.v1+json",
		"application/vnd.oci.image.index.v1+json",
		"application/vnd.docker.distribution.manifest.v2+json",
		"application/vnd.docker.distribution.manifest.list.v2+json",
	}, ", "))

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("HEAD %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)

	switch resp.StatusCode {
	case http.StatusOK:
		digest := resp.Header.Get("Docker-Content-Digest")
		if digest == "" {
			return "", fmt.Errorf("HEAD %s: empty Docker-Content-Digest", url)
		}
		return digest, nil
	case http.StatusNotFound:
		return "", ErrNotFound
	default:
		return "", fmt.Errorf("HEAD %s: unexpected status %d", url, resp.StatusCode)
	}
}

// DeleteManifest removes a manifest by digest. NotFound is treated as
// success per design §3.4 (GCBackend must be idempotent).
func (c *Client) DeleteManifest(ctx context.Context, repoPath, digest string) error {
	url := fmt.Sprintf("%s://%s/v2/%s/manifests/%s", c.scheme, c.baseURL, repoPath, digest)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.username, c.password)
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("DELETE %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)

	switch resp.StatusCode {
	case http.StatusOK, http.StatusAccepted, http.StatusNoContent:
		return nil
	case http.StatusNotFound:
		return nil
	default:
		return fmt.Errorf("DELETE %s: unexpected status %d", url, resp.StatusCode)
	}
}

// ErrNotFound is returned by HeadManifest when the manifest is absent.
var ErrNotFound = errors.New("oci: manifest not found")
