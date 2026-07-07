// Package artifacthub is a thin, typed wrapper over the generated artifact-hub
// client. The {namespace} path segment is the tenant scope.
//
// artifact-hub exposes a single /artifacts resource for every kind: artifacts
// are addressed by (namespace, name, version) and the kind travels in the
// Initiate body. This wrapper keeps a kind parameter on its methods so the
// Models / Images modules share one code path; on Initiate it is written into
// the request body, and on the read/mutate calls it is informational only
// (a name is unique across kinds within a namespace).
package artifacthub

import (
	"context"
	"net/http"
	"time"

	gen "github.com/axisml/axisml/axisml-platform/backend/internal/clients/artifacthub/generated"
	"github.com/axisml/axisml/axisml-platform/backend/internal/clients/clienterr"
	"github.com/axisml/axisml/axisml-platform/backend/internal/clients/reqedit"
)

const service = "artifacts"

// Re-exported generated types.
type (
	Artifact         = gen.Artifact
	InitiateRequest  = gen.ArtifactInitiateRequest
	InitiateResponse = gen.ArtifactInitiateResponse
	ResolveResponse  = gen.ArtifactResolveResponse
	PatchRequest     = gen.ArtifactPatchRequest
)

// Client wraps the generated artifact-hub client.
type Client struct{ gen *gen.ClientWithResponses }

// New builds an artifact-hub client for baseURL.
func New(baseURL string, timeout time.Duration) (*Client, error) {
	c, err := gen.NewClientWithResponses(baseURL,
		gen.WithHTTPClient(&http.Client{Timeout: timeout}),
		gen.WithRequestEditorFn(reqedit.Identity),
	)
	if err != nil {
		return nil, err
	}
	return &Client{gen: c}, nil
}

// Initiate starts a new version upload. kind is written into the request body.
func (c *Client) Initiate(ctx context.Context, ns, kind, name string, in InitiateRequest) (*InitiateResponse, error) {
	in.Kind = kind
	res, err := c.gen.InitiateArtifactWithResponse(ctx, ns, name, in)
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	if res.JSON201 != nil {
		return res.JSON201, nil
	}
	return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
}

// Get returns one artifact version.
func (c *Client) Get(ctx context.Context, ns, kind, name, version string) (*Artifact, error) {
	res, err := c.gen.GetArtifactWithResponse(ctx, ns, name, version)
	return view(res.JSON200, res.HTTPResponse, res.Body, err)
}

// Update patches mutable version metadata.
func (c *Client) Update(ctx context.Context, ns, kind, name, version string, in PatchRequest) (*Artifact, error) {
	res, err := c.gen.UpdateArtifactWithResponse(ctx, ns, name, version, in)
	return view(res.JSON200, res.HTTPResponse, res.Body, err)
}

// Delete soft-deletes one artifact version.
func (c *Client) Delete(ctx context.Context, ns, kind, name, version string) error {
	res, err := c.gen.DeleteArtifactWithResponse(ctx, ns, name, version)
	return ok(res.HTTPResponse, res.Body, err)
}

// Complete finalizes a version upload with its digest.
func (c *Client) Complete(ctx context.Context, ns, kind, name, version, digest string) (*Artifact, error) {
	body := gen.ArtifactCompleteRequest{Digest: digest}
	res, err := c.gen.CompleteArtifactWithResponse(ctx, ns, name, version, body)
	return view(res.JSON200, res.HTTPResponse, res.Body, err)
}

// Resolve returns download credentials / inspect URI for a version.
func (c *Client) Resolve(ctx context.Context, ns, kind, name, version, usage string) (*ResolveResponse, error) {
	res, err := c.gen.ResolveArtifactWithResponse(ctx, ns, name, version, &gen.ResolveArtifactParams{Usage: &usage})
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	if res.JSON200 != nil {
		return res.JSON200, nil
	}
	return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
}

// ListVersions lists a definition's versions.
func (c *Client) ListVersions(ctx context.Context, ns, kind, name string) ([]Artifact, error) {
	res, err := c.gen.ListArtifactVersionsWithResponse(ctx, ns, name, &gen.ListArtifactVersionsParams{})
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	if res.JSON200 == nil {
		return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
	}
	return res.JSON200.Items, nil
}

func view(j *Artifact, resp *http.Response, body []byte, err error) (*Artifact, error) {
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	if j != nil {
		return j, nil
	}
	return nil, clienterr.FromResponse(service, resp, body)
}

func ok(resp *http.Response, body []byte, err error) error {
	if err != nil {
		return clienterr.Transport(service, err)
	}
	if resp != nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	return clienterr.FromResponse(service, resp, body)
}
